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

	"github.com/prevostcorentin/go-qga/internal/common"
	qgaerrors "github.com/prevostcorentin/go-qga/internal/errors"
	"github.com/prevostcorentin/go-qga/internal/qmp/transport"
)

type Connection interface {
	Connect(ctx context.Context, path string) error
	Send(ctx context.Context, bytes []byte) ([]byte, error)
	SendAsync(ctx context.Context, bytes []byte) <-chan TransportResult
	Close() error
}

type TransportResult struct {
	Data []byte
	Err  error
}


type connection struct {
	transport   transport.Transport // 16 bytes (interface)
	*common.BaseState              // 8 bytes (pointer) - provides locking
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

	conn := &connection{
		BaseState: common.NewBaseState(),
		transport: transport,
	}

	if err := conn.Connect(ctx, path); err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *connection) Connect(ctx context.Context, path string) error {
	return c.consumeBanner(ctx)
}

func (c *connection) consumeBanner(ctx context.Context) error {
	// Read and consume the QMP banner (currently discarded)
	if _, err := c.transport.Read(ctx); err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	return nil
}

func (c *connection) Send(ctx context.Context, bytes []byte) ([]byte, error) {
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

func (c *connection) SendAsync(ctx context.Context, bytes []byte) <-chan TransportResult {
	resultCh := make(chan TransportResult, 1)
	
	// Check if closed
	c.RLock()
	closed := c.IsClosedWhileLocked()
	c.RUnlock()
	
	if closed {
		resultCh <- TransportResult{
			Data: nil,
			Err:  qgaerrors.ErrConnectionClosed,
		}
		close(resultCh)
		return resultCh
	}

	// Simplified goroutine with proper cleanup
	go func() {
		defer close(resultCh)
		
		// Check context before starting
		select {
		case <-ctx.Done():
			resultCh <- TransportResult{Data: nil, Err: ctx.Err()}
			return
		default:
		}
		
		// Perform write operation
		if err := c.transport.Write(ctx, bytes); err != nil {
			resultCh <- TransportResult{Data: nil, Err: fmt.Errorf("send error: %w", err)}
			return
		}
		
		// Check context before read
		select {
		case <-ctx.Done():
			resultCh <- TransportResult{Data: nil, Err: ctx.Err()}
			return
		default:
		}
		
		// Perform read operation
		responseBytes, err := c.transport.Read(ctx)
		if err != nil {
			resultCh <- TransportResult{Data: responseBytes, Err: fmt.Errorf("send error: %w", err)}
			return
		}
		
		// Send successful result
		resultCh <- TransportResult{Data: responseBytes, Err: nil}
	}()
	
	return resultCh
}

func (c *connection) Close() error {
	c.Lock()
	defer c.Unlock()

	if c.IsClosedWhileLocked() {
		return nil
	}

	c.CloseWithLock()

	if err := c.transport.Close(); err != nil {
		return fmt.Errorf("close error: %w", err)
	}
	return nil
}
