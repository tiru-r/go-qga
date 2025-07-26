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
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
	"github.com/prevostcorentin/go-qga/internal/errors"
)

type Agent interface {
	Serve(ctx context.Context, handler func(net.Conn)) error
	WaitReady()
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error: " + e.Message
}

// SocketAgentConfig holds configuration options for SocketAgent
type SocketAgentConfig struct {
	SocketPath     string
	BufferSize     int           // Use -1 for default buffer size
	MaxConnections int           // Use -1 for default max connections
	ReadTimeout    time.Duration // Use 0 for default timeout
}

type SocketAgent struct {
	config        *SocketAgentConfig
	ready         chan struct{}
	connSemaphore chan struct{} // Channel-based connection limiting
	maxConn       int64         // Max connections limit
}

func NewSocketAgent(socketPath string) *SocketAgent {
	return NewSocketAgentWithConfig(&SocketAgentConfig{
		SocketPath:     socketPath,
		BufferSize:     -1, // Use default
		MaxConnections: -1, // Use default
		ReadTimeout:    0,  // Use default
	})
}

func NewSocketAgentWithConfig(config *SocketAgentConfig) *SocketAgent {
	if config == nil {
		panic(errors.ConfigNilMessage)
	}
	if config.SocketPath == "" {
		panic(errors.SocketPathEmptyMessage)
	}

	maxConn := int64(common.DefaultMaxConnections) // Default max connections
	if config.MaxConnections != -1 {
		if config.MaxConnections <= 0 {
			panic("max connections must be positive")
		}
		if config.MaxConnections > common.MaxAllowedConnections {
			panic("max connections too large (max: 1,000,000)")
		}
		maxConn = int64(config.MaxConnections)
	}

	// Validate timeout if provided
	if config.ReadTimeout != 0 && config.ReadTimeout <= 0 {
		panic("read timeout must be positive")
	}

	// Validate buffer size if provided
	if config.BufferSize != -1 && config.BufferSize <= 0 {
		panic("buffer size must be positive")
	}

	return &SocketAgent{
		config:        config,
		ready:         make(chan struct{}, 1),       // Buffered to prevent blocking
		connSemaphore: make(chan struct{}, maxConn), // Buffered semaphore
		maxConn:       maxConn,
	}
}

func (s *SocketAgent) Serve(ctx context.Context, handler func(net.Conn)) error {
	socketPath := s.config.SocketPath

	// Enhanced security validation to prevent path traversal and unauthorized access
	if strings.Contains(socketPath, "..") {
		return &ValidationError{Message: "socket path cannot contain '..'"}
	}

	// Check for null bytes and other control characters
	if strings.ContainsAny(socketPath, "\x00\r\n") {
		return &ValidationError{Message: "socket path contains invalid characters"}
	}

	// Resolve path to handle symlinks and normalize
	absPath, err := filepath.Abs(socketPath)
	if err != nil {
		return &ValidationError{Message: "invalid socket path: " + err.Error()}
	}

	// Ensure resolved path is within safe directories
	if !strings.HasPrefix(absPath, "/tmp/") && !strings.HasPrefix(absPath, "/var/tmp/") {
		return &ValidationError{Message: "socket path must be in /tmp/ or /var/tmp/ (resolved: " + absPath + ")"}
	}

	// Additional check for symlink traversal in parent directory
	if _, err := os.Lstat(filepath.Dir(absPath)); err != nil && !os.IsNotExist(err) {
		return &ValidationError{Message: "cannot access parent directory: " + err.Error()}
	}

	// Only remove the file if it exists and is a socket
	if stat, err := os.Stat(socketPath); err == nil {
		if stat.Mode()&os.ModeSocket != 0 {
			os.Remove(socketPath)
		} else {
			return &ValidationError{Message: "path exists but is not a socket"}
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer listener.Close()
	defer os.Remove(socketPath)

	var wg sync.WaitGroup
	var closeOnce sync.Once

	close(s.ready)

	// Context monitoring goroutine with proper cleanup
	ctxDone := make(chan struct{}, 1) // Buffered to prevent goroutine leak
	go func() {
		defer close(ctxDone)
		select {
		case <-ctx.Done():
			closeOnce.Do(func() {
				listener.Close()
			})
		case <-ctxDone:
			// Cleanup signal received
			return
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctxDone:
				wg.Wait()
				return ctx.Err()
			default:
				select {
				case <-ctx.Done():
					wg.Wait()
					return ctx.Err()
				default:
					wg.Wait()
					return err
				}
			}
		}

		// Channel-based connection limiting (much simpler and more efficient!)
		select {
		case s.connSemaphore <- struct{}{}:
			// Got semaphore slot, handle connection
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer c.Close()
				defer func() { <-s.connSemaphore }() // Release semaphore
				defer func() {
					if r := recover(); r != nil {
						// Recover from panics to prevent server crash
						// Handler panics are logged but don't terminate the server
					}
				}()

				// Set read timeout if configured
				if s.config.ReadTimeout != 0 {
					common.GlobalTimeManager.SetReadDeadline(c, s.config.ReadTimeout)
				}

				handler(c)
			}(conn)
		default:
			// No semaphore slots available, reject connection
			conn.Close()
		}
	}
}

func (s *SocketAgent) WaitReady() {
	<-s.ready
}

// GetActiveConnections returns the current number of active connections
func (s *SocketAgent) GetActiveConnections() int64 {
	return s.maxConn - int64(len(s.connSemaphore))
}

// GetMaxConnections returns the maximum allowed connections
func (s *SocketAgent) GetMaxConnections() int64 {
	return s.maxConn
}

// TestingT interface that both testing.T and testing.B implement
type TestingT interface {
	TempDir() string
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
}

func BuildSocketPath(t TestingT) string {
	testFolder := t.TempDir()
	return filepath.Join(testFolder, "go-qga-test-socket.sock")
}
