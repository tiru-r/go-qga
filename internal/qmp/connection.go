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

	"github.com/prevostcorentin/go-qga/internal/common"
	qgaerrors "github.com/prevostcorentin/go-qga/internal/errors"
	"github.com/prevostcorentin/go-qga/internal/qmp/transport"
)

type Connection interface {
	Connect(ctx context.Context, path string) *qgaerrors.ConnectionError
	Send(ctx context.Context, bytes []byte) ([]byte, *qgaerrors.ConnectionError)
	SendAsync(ctx context.Context, bytes []byte) <-chan AsyncResult
	Close() error
}

type AsyncResult struct {
	Data []byte
	Err  *qgaerrors.ConnectionError
}

type asyncRequest struct {
	id   uint64
	ch   chan AsyncResult
	data []byte
}

type qmpConnection struct {
	*common.BaseState
	transport      transport.Transport
	requestCounter uint64
	pendingReqs    map[uint64]chan AsyncResult
	pool           *sync.Pool
}

func Open(ctx context.Context, path string, transport transport.Transport) (Connection, *qgaerrors.ConnectionError) {
	if transport == nil {
		return nil, qgaerrors.ErrConnectionNil
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		errorReason := fmt.Errorf(`socket "%s" does not exist`, path)
		return nil, qgaerrors.NewConnectionError(errorReason, qgaerrors.ConnectErrorKind)
	}

	if err := transport.Connect(ctx); err != nil {
		if underlying := err.Unwrap(); underlying != nil {
			return nil, qgaerrors.NewConnectionError(underlying, qgaerrors.ConnectErrorKind)
		}
		return nil, qgaerrors.NewConnectionError(err, qgaerrors.ConnectErrorKind)
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

func (c *qmpConnection) Connect(ctx context.Context, path string) *qgaerrors.ConnectionError {
	return c.consumeBanner(ctx)
}

func (c *qmpConnection) consumeBanner(ctx context.Context) *qgaerrors.ConnectionError {
	// Read and consume the QMP banner (currently discarded)
	if _, err := c.transport.Read(ctx); err != nil {
		return qgaerrors.NewConnectionError(err, qgaerrors.ReadErrorKind)
	}
	return nil
}

func (c *qmpConnection) Send(ctx context.Context, bytes []byte) ([]byte, *qgaerrors.ConnectionError) {
	c.RLock()
	defer c.RUnlock()

	if c.IsClosedWhileLocked() {
		return nil, qgaerrors.ErrConnectionClosed
	}

	if err := c.transport.Write(ctx, bytes); err != nil {
		return nil, qgaerrors.NewConnectionError(err, qgaerrors.SendErrorKind)
	}

	responseBytes, err := c.transport.Read(ctx)
	if err != nil {
		return responseBytes, qgaerrors.NewConnectionError(err, qgaerrors.SendErrorKind)
	}
	return responseBytes, nil
}

func (c *qmpConnection) SendAsync(ctx context.Context, bytes []byte) <-chan AsyncResult {
	c.RLock()

	resultCh := c.pool.Get().(chan AsyncResult)
	
	// Drain any stale data from pooled channel
	select {
	case <-resultCh:
	default:
	}

	if c.IsClosedWhileLocked() {
		c.RUnlock() // Manual unlock to avoid defer overhead
		resultCh <- AsyncResult{
			Data: nil,
			Err:  qgaerrors.ErrConnectionClosed,
		}
		return resultCh
	}

	reqID := atomic.AddUint64(&c.requestCounter, 1)
	
	// Protect map write with full mutex
	c.RUnlock()
	c.Lock()
	c.pendingReqs[reqID] = resultCh
	c.Unlock()

	// Use optimized goroutine with closure optimization
	go func(localCtx context.Context, localBytes []byte, localResultCh chan AsyncResult, localReqID uint64) {
		defer func() {
			c.Lock()
			delete(c.pendingReqs, localReqID)
			c.Unlock()
			c.pool.Put(localResultCh)
		}()

		if err := c.transport.Write(localCtx, localBytes); err != nil {
			localResultCh <- AsyncResult{
				Data: nil,
				Err:  qgaerrors.NewConnectionError(err, qgaerrors.SendErrorKind),
			}
			return
		}

		responseBytes, err := c.transport.Read(localCtx)
		if err != nil {
			localResultCh <- AsyncResult{
				Data: responseBytes,
				Err:  qgaerrors.NewConnectionError(err, qgaerrors.SendErrorKind),
			}
			return
		}

		localResultCh <- AsyncResult{
			Data: responseBytes,
			Err:  nil,
		}
	}(ctx, bytes, resultCh, reqID)

	return resultCh
}

func (c *qmpConnection) Close() error {
	c.Lock()
	defer c.Unlock()

	if c.IsClosedWhileLocked() {
		return nil
	}

	c.CloseWithLock()

	// Cancel all pending requests
	for _, ch := range c.pendingReqs {
		ch <- AsyncResult{
			Data: nil,
			Err:  qgaerrors.NewConnectionError(fmt.Errorf("connection closed"), qgaerrors.CloseErrorKind),
		}
	}


	if err := c.transport.Close(); err != nil {
		return qgaerrors.NewConnectionError(err, qgaerrors.CloseErrorKind)
	}
	return nil
}
