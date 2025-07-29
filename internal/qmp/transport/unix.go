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

package transport

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

type UnixTransport struct {
	*common.BaseState
	path       string
	connection net.Conn
	pipe       *bufio.ReadWriter
	readCh     chan readResult
	writeCh    chan writeRequest
	done       chan struct{}
	wg         sync.WaitGroup  // Track background goroutines
	closed     int32           // Atomic flag to prevent multiple closes
}

type readResult struct {
	data []byte
	err  error
}

type writeRequest struct {
	data []byte
	resp chan error
}

func (t *UnixTransport) Connect(ctx context.Context) error {
	t.Lock()
	defer t.Unlock()

	if t.IsClosedWhileLocked() {
		return fmt.Errorf("transport is closed")
	}

	d := &net.Dialer{}
	conn, err := d.DialContext(ctx, "unix", t.path)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", t.path, err)
	}

	t.connection = conn
	t.pipe = bufio.NewReadWriter(
		bufio.NewReader(t.connection),
		bufio.NewWriter(t.connection),
	)

	// Initialize channels
	t.readCh = make(chan readResult, 1)
	t.writeCh = make(chan writeRequest, 10)
	t.done = make(chan struct{})

	// Start background goroutines with proper tracking
	t.wg.Add(2) // Track both readLoop and writeLoop
	go func() {
		defer t.wg.Done()
		t.readLoop()
	}()
	go func() {
		defer t.wg.Done()
		t.writeLoop()
	}()

	return nil
}

func (t *UnixTransport) Write(ctx context.Context, bytes []byte) error {
	// Check atomic closed flag first to avoid lock contention
	if atomic.LoadInt32(&t.closed) != 0 {
		return fmt.Errorf("transport is closed")
	}

	t.RLock()
	defer t.RUnlock()

	// Double-check with BaseState
	if t.IsClosedWhileLocked() {
		return fmt.Errorf("transport is closed")
	}

	resp := common.ErrorChannel()
	req := writeRequest{
		data: bytes,
		resp: resp,
	}

	select {
	case t.writeCh <- req:
		select {
		case err := <-resp:
			return err
		case <-ctx.Done():
			return fmt.Errorf("write timeout: %w", ctx.Err())
		}
	case <-ctx.Done():
		return fmt.Errorf("write timeout: %w", ctx.Err())
	case <-t.done:
		return fmt.Errorf("transport is closed")
	}
}

func (t *UnixTransport) Read(ctx context.Context) ([]byte, error) {
	// Check atomic closed flag first to avoid lock contention
	if atomic.LoadInt32(&t.closed) != 0 {
		return nil, fmt.Errorf("transport is closed")
	}

	t.RLock()
	defer t.RUnlock()

	// Double-check with BaseState
	if t.IsClosed() {
		return nil, fmt.Errorf("transport is closed")
	}

	select {
	case result := <-t.readCh:
		return result.data, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("read timeout: %w", ctx.Err())
	case <-t.done:
		return nil, fmt.Errorf("transport is closed")
	}
}

func (u *UnixTransport) Path() string {
	return u.path
}

func (u *UnixTransport) Close() error {
	// Use atomic compare-and-swap to ensure only one close operation
	if !atomic.CompareAndSwapInt32(&u.closed, 0, 1) {
		return nil // Already closed by another goroutine
	}

	// Get local references without holding any locks to avoid deadlock
	doneChannel := u.done
	connection := u.connection
	pipe := u.pipe

	// Signal goroutines to stop first
	if doneChannel != nil {
		select {
		case <-doneChannel:
			// Already closed
		default:
			close(doneChannel)
		}
	}

	// Wait for background goroutines to terminate
	u.wg.Wait()

	// Now safely mark BaseState as closed
	u.Lock()
	u.CloseWithLock()
	u.Unlock()

	// Clean up resources without holding any locks
	var firstErr error

	if pipe != nil {
		if err := pipe.Writer.Flush(); err != nil {
			firstErr = fmt.Errorf("flush error: %w", err)
		}
	}

	if connection != nil {
		if err := connection.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close error: %w", err)
		}
	}

	return firstErr
}

func (t *UnixTransport) readLoop() {
	for {
		select {
		case <-t.done:
			return
		default:
			// Check atomic flag first for early exit
			if atomic.LoadInt32(&t.closed) != 0 {
				return
			}
			
			// Perform the entire read operation under lock to prevent race conditions
			t.RLock()
			if t.IsClosed() || t.connection == nil || t.pipe == nil {
				t.RUnlock()
				return // Transport was closed
			}
			
			// Set read timeout for blocking operation (connection access is safe under lock)
			if err := t.connection.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				t.RUnlock()
				select {
				case t.readCh <- readResult{nil, fmt.Errorf("read error: %w", err)}:
				case <-t.done:
					return
				}
				continue
			}

			// Blocking read with timeout - much more efficient than polling
			bytes, err := t.pipe.ReadBytes(common.LineTerminator)
			t.RUnlock() // Release lock after read operation
			result := readResult{bytes, nil}
			if err != nil {
				// Check if this is a timeout error vs actual error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is expected, continue reading
					continue
				}
				result = readResult{nil, fmt.Errorf("read error: %w", err)}
			}
			
			select {
			case t.readCh <- result:
			case <-t.done:
				return
			}
		}
	}
}

func (t *UnixTransport) writeLoop() {
	for {
		select {
		case <-t.done:
			return
		case req := <-t.writeCh:
			// Check atomic flag first for early exit
			if atomic.LoadInt32(&t.closed) != 0 {
				req.resp <- fmt.Errorf("write failed: transport closed")
				continue
			}
			
			// Perform the entire write operation under lock to prevent race conditions
			t.RLock()
			if t.IsClosed() || t.connection == nil || t.pipe == nil {
				t.RUnlock()
				req.resp <- fmt.Errorf("write failed: transport closed")
				continue
			}
			
			// Set write timeout (connection access is safe under lock)
			if err := t.connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				t.RUnlock()
				req.resp <- fmt.Errorf("write error: %w", err)
				continue
			}

			if _, err := t.pipe.Write(req.data); err != nil {
				t.RUnlock()
				req.resp <- fmt.Errorf("write error: %w", err)
				continue
			}

			if err := t.pipe.Writer.Flush(); err != nil {
				t.RUnlock()
				req.resp <- fmt.Errorf("flush error: %w", err)
			} else {
				t.RUnlock()
				req.resp <- nil
			}
		}
	}
}
