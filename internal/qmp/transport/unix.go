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
	"net"

	"github.com/prevostcorentin/go-qga/internal/common"
	"github.com/prevostcorentin/go-qga/internal/errors"
)

type unixTransport struct {
	*common.BaseState
	path       string
	connection net.Conn
	pipe       *bufio.ReadWriter
	readCh     chan readResult
	writeCh    chan writeRequest
	done       chan struct{}
}

type readResult struct {
	data []byte
	err  error
}

type writeRequest struct {
	data []byte
	resp chan error
}

func (t *unixTransport) Connect(ctx context.Context) *errors.TransportError {
	t.Lock()
	defer t.Unlock()

	if t.IsClosedWhileLocked() {
		return errors.ErrTransportClosed
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", t.path)
	if err != nil {
		return errors.NewTransportError(err, errors.Connect)
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

	// Start background goroutines
	go t.readLoop()
	go t.writeLoop()

	return nil
}

func (t *unixTransport) Write(ctx context.Context, bytes []byte) error {
	t.RLock()
	defer t.RUnlock()

	if t.IsClosedWhileLocked() {
		return errors.ErrTransportClosed
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
			return errors.NewTransportError(ctx.Err(), errors.Write)
		}
	case <-ctx.Done():
		return errors.NewTransportError(ctx.Err(), errors.Write)
	case <-t.done:
		return errors.ErrTransportClosed
	}
}

func (t *unixTransport) Read(ctx context.Context) ([]byte, error) {
	t.RLock()
	defer t.RUnlock()

	if t.IsClosed() {
		return nil, errors.ErrTransportClosed
	}

	select {
	case result := <-t.readCh:
		return result.data, result.err
	case <-ctx.Done():
		return nil, errors.NewTransportError(ctx.Err(), errors.Read)
	case <-t.done:
		return nil, errors.ErrTransportClosed
	}
}

func (transport *unixTransport) Path() string {
	return transport.path
}

func (transport *unixTransport) Close() error {
	transport.Lock()
	defer transport.Unlock()

	if !transport.CloseWithLock() {
		return nil // already closed
	}
	close(transport.done)

	var firstErr error

	if transport.pipe != nil {
		if err := transport.pipe.Writer.Flush(); err != nil {
			firstErr = errors.NewTransportError(err, errors.Flush)
		}
	}

	if transport.connection != nil {
		if err := transport.connection.Close(); err != nil && firstErr == nil {
			firstErr = errors.NewTransportError(err, errors.Close)
		}
	}

	return firstErr
}

func (t *unixTransport) readLoop() {
	for {
		select {
		case <-t.done:
			return
		default:
			// Set read timeout for blocking operation
			if err := common.GlobalTimeManager.SetReadDeadlineDefault(t.connection); err != nil {
				select {
				case t.readCh <- readResult{nil, errors.NewTransportError(err, errors.Read)}:
				case <-t.done:
					return
				}
				continue
			}

			// Blocking read with timeout - much more efficient than polling
			bytes, err := t.pipe.ReadBytes(common.LineTerminator)
			result := readResult{bytes, nil}
			if err != nil {
				// Check if this is a timeout error vs actual error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is expected, continue reading
					continue
				}
				result = readResult{nil, errors.NewTransportError(err, errors.Read)}
			}
			
			select {
			case t.readCh <- result:
			case <-t.done:
				return
			}
		}
	}
}

func (t *unixTransport) writeLoop() {
	for {
		select {
		case <-t.done:
			return
		case req := <-t.writeCh:
			// Set write timeout
			if err := common.GlobalTimeManager.SetWriteDeadlineDefault(t.connection); err != nil {
				req.resp <- errors.NewTransportError(err, errors.Write)
				continue
			}

			if _, err := t.pipe.Write(req.data); err != nil {
				req.resp <- errors.NewTransportError(err, errors.Write)
				continue
			}

			if err := t.pipe.Writer.Flush(); err != nil {
				req.resp <- errors.NewTransportError(err, errors.Flush)
			} else {
				req.resp <- nil
			}
		}
	}
}
