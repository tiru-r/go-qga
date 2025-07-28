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

package qmp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
	qgaerrors "github.com/prevostcorentin/go-qga/internal/errors"
	"github.com/prevostcorentin/go-qga/internal/qmp/transport"
)

type Connection interface {
	Connect(ctx context.Context, path string) error
	Send(ctx context.Context, bytes []byte) ([]byte, error)
	SendAsync(ctx context.Context, bytes []byte) <-chan AsyncResult
	Close() error
}

type AsyncResult struct {
	Data []byte
	Err  error
}

type asyncRequest struct {
	id   uint64
	ch   chan AsyncResult
	data []byte
}

type qmpConnection struct {
	transport      transport.Transport         // 16 bytes (interface)
	*common.BaseState                          // 8 bytes (pointer) - provides locking
	pendingReqs    map[uint64]chan AsyncResult // 8 bytes (pointer)
	pool           *sync.Pool                  // 8 bytes (pointer)
	requestCounter uint64                      // 8 bytes - use atomic operations
}

func Open(ctx context.Context, path string, transport transport.Transport) (Connection, error) {
	if transport == nil {
		return nil, fmt.Errorf("connection is nil")
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := fmt.Errorf(`socket "%s" does not exist`, path)
		return nil, fmt.Errorf("connection error: %w", err)
	}

	if err := transport.Connect(ctx); err != nil {
		if underlying := errors.Unwrap(err); underlying != nil {
			return nil, fmt.Errorf("connection error: %w", underlying)
		}
		return nil, fmt.Errorf("connection error: %w", err)
	}

	conn := &qmpConnection{
		BaseState:   common.NewBaseState(),
		transport:   transport,
		pendingReqs: make(map[uint64]chan AsyncResult),
		pool: &sync.Pool{
			New: func() any {
				return make(chan AsyncResult, 1)
			},
		},
	}

	if err := conn.Connect(ctx, path); err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *qmpConnection) Connect(ctx context.Context, path string) error {
	return c.consumeBanner(ctx)
}

func (c *qmpConnection) consumeBanner(ctx context.Context) error {
	// Read and consume the QMP banner (currently discarded)
	if _, err := c.transport.Read(ctx); err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	return nil
}

func (c *qmpConnection) Send(ctx context.Context, bytes []byte) ([]byte, error) {
	c.RLock()
	defer c.RUnlock()

	if c.IsClosedWhileLocked() {
		return nil, qgaerrors.ErrConnectionClosed
	}

	if err := c.transport.Write(ctx, bytes); err != nil {
		return nil, fmt.Errorf("send error: %w", err)
	}

	responseBytes, err := c.transport.Read(ctx)
	if err != nil {
		return responseBytes, fmt.Errorf("send error: %w", err)
	}
	return responseBytes, nil
}

func (c *qmpConnection) SendAsync(ctx context.Context, bytes []byte) <-chan AsyncResult {
	// Get channel first to avoid doing it under lock
	resultCh := c.pool.Get().(chan AsyncResult)
	
	// Check if closed with atomic operation to avoid race condition
	c.RLock()
	closed := c.IsClosedWhileLocked()
	c.RUnlock()
	
	if closed {
		resultCh <- AsyncResult{
			Data: nil,
			Err:  qgaerrors.ErrConnectionClosed,
		}
		return resultCh
	}
	
	// Safely drain any stale data from pooled channel
	for {
		select {
		case <-resultCh:
			// Continue draining
		default:
			// Channel is empty, safe to use
			goto channelReady
		}
	}
channelReady:

	reqID := atomic.AddUint64(&c.requestCounter, 1)
	
	// Store pending request with separate mutex to prevent deadlock
	c.Lock()
	c.pendingReqs[reqID] = resultCh
	c.Unlock()

	// Use single goroutine with proper cleanup to prevent goroutine leaks
	go func(localCtx context.Context, localBytes []byte, localResultCh chan AsyncResult, localReqID uint64) {
		var sent bool
		
		defer func() {
			// Clean up pending request using unified locking
			c.Lock()
			delete(c.pendingReqs, localReqID)
			c.Unlock()
			
			// Always return channel to pool to prevent leaks
			// If no result was sent, send a cancellation error first
			if !sent {
				select {
				case localResultCh <- AsyncResult{
					Data: nil,
					Err:  context.Canceled,
				}:
				default:
					// Channel is full, don't block
				}
			}
			c.pool.Put(localResultCh)
		}()
		
		// Check context before starting
		select {
		case <-localCtx.Done():
			localResultCh <- AsyncResult{Data: nil, Err: localCtx.Err()}
			sent = true
			return
		default:
		}
		
		// Perform write operation
		if err := c.transport.Write(localCtx, localBytes); err != nil {
			localResultCh <- AsyncResult{Data: nil, Err: fmt.Errorf("send error: %w", err)}
			sent = true
			return
		}
		
		// Check context before read
		select {
		case <-localCtx.Done():
			localResultCh <- AsyncResult{Data: nil, Err: localCtx.Err()}
			sent = true
			return
		default:
		}
		
		// Perform read operation
		responseBytes, err := c.transport.Read(localCtx)
		if err != nil {
			localResultCh <- AsyncResult{Data: responseBytes, Err: fmt.Errorf("send error: %w", err)}
			sent = true
			return
		}
		
		// Send successful result
		localResultCh <- AsyncResult{Data: responseBytes, Err: nil}
		sent = true
	}(ctx, bytes, resultCh, reqID)
	
	return resultCh
}

func (c *qmpConnection) Close() error {
	c.Lock()

	if c.IsClosedWhileLocked() {
		c.Unlock()
		return nil
	}

	c.CloseWithLock()

	// Get copy of pending requests while holding lock
	pendingCopy := make(map[uint64]chan AsyncResult, len(c.pendingReqs))
	for id, ch := range c.pendingReqs {
		pendingCopy[id] = ch
	}

	// Clear all pending requests immediately
	for reqID := range c.pendingReqs {
		delete(c.pendingReqs, reqID)
	}

	c.Unlock()

	// Cancel all pending requests without holding lock
	for _, ch := range pendingCopy {
		select {
		case ch <- AsyncResult{
			Data: nil,
			Err:  fmt.Errorf("connection closed"),
		}:
			// Successfully sent close signal
		case <-time.After(50 * time.Millisecond):
			// Channel blocked, skip it
		}
	}

	if err := c.transport.Close(); err != nil {
		return fmt.Errorf("close error: %w", err)
	}
	return nil
}
