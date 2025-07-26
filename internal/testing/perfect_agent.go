// Copyright 2025 PREVOST Corentin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testing

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// PerfectAgent represents the ideal QMP guest agent - simple, fast, and just works
type PerfectAgent struct {
	path       string
	maxConn    int
	timeout    time.Duration
	bufferSize int

	listener net.Listener
	conns    chan struct{} // Simple semaphore for connection limiting
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// Perfect creates a perfect agent with optimal defaults that just work
func Perfect(socketPath string) *PerfectAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerfectAgent{
		path:       socketPath,
		maxConn:    common.HighPerfMaxConnections, // High performance default
		timeout:    common.DefaultTimeout,        // Generous timeout
		bufferSize: common.LargeBufferSize,       // Optimal buffer size
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Fast creates a perfect agent optimized for speed
func Fast(socketPath string) *PerfectAgent {
	return Perfect(socketPath).
		WithConnections(100000).           // Very high connection limit
		WithTimeout(common.FastTimeout).  // Shorter timeout for speed
		WithBuffer(common.XLargeBufferSize) // Larger buffer for throughput
}

// Simple creates a perfect agent optimized for simplicity
func Simple(socketPath string) *PerfectAgent {
	return Perfect(socketPath).
		WithConnections(common.TestMaxConnections). // Lower connection limit
		WithTimeout(common.SlowTimeout).           // Longer timeout for reliability
		WithBuffer(common.StandardBufferSize)      // Standard buffer size
}

// WithConnections sets the maximum connections optimistically
func (a *PerfectAgent) WithConnections(max int) *PerfectAgent {
	if max > 0 && max <= 1000000 { // Sensible bounds
		a.maxConn = max
	}
	return a
}

// WithTimeout sets the timeout optimistically
func (a *PerfectAgent) WithTimeout(timeout time.Duration) *PerfectAgent {
	if timeout > 0 && timeout <= 10*time.Minute { // Sensible bounds
		a.timeout = timeout
	}
	return a
}

// WithBuffer sets the buffer size optimistically
func (a *PerfectAgent) WithBuffer(size int) *PerfectAgent {
	if size >= common.SmallBufferSize && size <= common.XLargeBufferSize { // Sensible bounds
		a.bufferSize = size
	}
	return a
}

// Start starts the perfect agent and returns immediately
func (a *PerfectAgent) Start() common.Result[*PerfectAgent] {
	// Smart path handling - create directory if needed
	if err := a.ensureSocketPath(); err != nil {
		return common.FailureFrom[*PerfectAgent](err)
	}

	// Create listener with optimistic error handling
	listener, err := net.Listen("unix", a.path)
	if err != nil {
		return common.Failure[*PerfectAgent]("failed to listen on %s: %v", a.path, err)
	}

	a.listener = listener
	a.conns = make(chan struct{}, a.maxConn)

	// Start accepting connections in background
	go a.serve()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	return common.Success(a)
}

// Stop gracefully stops the agent
func (a *PerfectAgent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.listener != nil {
		a.listener.Close()
	}
	a.wg.Wait()
	os.Remove(a.path)
}

// Path returns the socket path
func (a *PerfectAgent) Path() string {
	return a.path
}

// Stats returns current agent statistics
func (a *PerfectAgent) Stats() map[string]any {
	activeConns := len(a.conns)
	return map[string]any{
		"path":               a.path,
		"max_connections":    a.maxConn,
		"active_connections": activeConns,
		"available_slots":    a.maxConn - activeConns,
		"timeout":            a.timeout.String(),
		"buffer_size":        a.bufferSize,
	}
}

// ensureSocketPath creates directory and cleans up old socket
func (a *PerfectAgent) ensureSocketPath() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(a.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Remove existing socket file
	os.Remove(a.path)

	return nil
}

// serve handles incoming connections optimistically
func (a *PerfectAgent) serve() {
	defer a.listener.Close()

	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			// Set accept timeout to allow graceful shutdown
			if unixListener, ok := a.listener.(*net.UnixListener); ok {
				unixListener.SetDeadline(common.GlobalTimeManager.NowPlus(100 * time.Millisecond))
			}

			conn, err := a.listener.Accept()
			if err != nil {
				select {
				case <-a.ctx.Done():
					return
				default:
					continue // Ignore timeout errors, keep trying
				}
			}

			// Try to acquire connection slot
			select {
			case a.conns <- struct{}{}:
				// Got slot, handle connection
				a.wg.Add(1)
				go a.handleConnection(conn)
			default:
				// No slots available, close immediately
				conn.Close()
			}
		}
	}
}

// handleConnection handles a single connection optimistically
func (a *PerfectAgent) handleConnection(conn net.Conn) {
	defer a.wg.Done()
	defer conn.Close()
	defer func() { <-a.conns }() // Release connection slot
	defer func() {
		if r := recover(); r != nil {
			// Optimistic recovery - log and continue
		}
	}()

	// Set connection timeout
	common.GlobalTimeManager.SetDeadline(conn, a.timeout)

	// Send QMP banner (simplified)
	conn.Write([]byte(common.QmpBannerMessage))

	// Handle commands optimistically
	buffer := common.GlobalBufferPool.GetStandard()
	defer common.GlobalBufferPool.PutStandard(buffer)
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			// Set read deadline to prevent blocking indefinitely
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := conn.Read(buffer)
			if err != nil {
				// Check if it's a timeout error and context is cancelled
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					select {
					case <-a.ctx.Done():
						return
					default:
						continue // Timeout but not cancelled, keep reading
					}
				}
				return
			}

			// Simple command parsing - look for hostname request (optimized: no string conversion)
			if bytes.Contains(buffer[:n], []byte("guest-get-host-name")) {
				conn.Write([]byte(common.QmpHostnameResponse))
			} else {
				// Unknown command - simple error
				conn.Write([]byte(common.QmpCommandNotFoundResponse))
			}
		}
	}
}

// Benchmark runs a simple performance test
func (a *PerfectAgent) Benchmark(connections int, duration time.Duration) map[string]any {
	start := time.Now()
	var successful, failed int64
	var wg sync.WaitGroup

	// Launch concurrent connections
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := net.Dial("unix", a.path)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			defer conn.Close()

			// Read banner
			buffer := common.GlobalBufferPool.GetStandard()
			defer common.GlobalBufferPool.PutStandard(buffer)
			conn.Read(buffer)

			// Send hostname request
			conn.Write([]byte(common.QmpHostnameRequest))

			// Read response
			n, err := conn.Read(buffer)
			if err != nil || n == 0 {
				atomic.AddInt64(&failed, 1)
				return
			}

			atomic.AddInt64(&successful, 1)
		}()

		// Small delay between connections
		time.Sleep(duration / time.Duration(connections))
	}

	wg.Wait()
	elapsed := time.Since(start)

	return map[string]any{
		"connections":      connections,
		"successful":       atomic.LoadInt64(&successful),
		"failed":           atomic.LoadInt64(&failed),
		"duration":         elapsed.String(),
		"requests_per_sec": float64(successful) / elapsed.Seconds(),
		"success_rate":     float64(successful) / float64(connections) * 100,
	}
}
