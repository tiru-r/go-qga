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
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// SimpleTestAgent provides a basic test agent - no over-engineering
type SimpleTestAgent struct {
	socketPath string
	listener   net.Listener
	ready      chan struct{}
}

// NewSimpleTestAgent creates a simple test agent
func NewSimpleTestAgent(socketPath string) *SimpleTestAgent {
	return &SimpleTestAgent{
		socketPath: socketPath,
		ready:      make(chan struct{}),
	}
}

// Start starts the test agent
func (a *SimpleTestAgent) Start() error {
	// Clean up any existing socket
	os.Remove(a.socketPath)
	
	listener, err := net.Listen("unix", a.socketPath)
	if err != nil {
		return err
	}
	
	a.listener = listener
	close(a.ready)
	
	return nil
}

// Stop stops the test agent
func (a *SimpleTestAgent) Stop() error {
	if a.listener != nil {
		err := a.listener.Close()
		os.Remove(a.socketPath)
		return err
	}
	return nil
}

// Path returns the socket path
func (a *SimpleTestAgent) Path() string {
	return a.socketPath
}

// WaitReady waits for the agent to be ready
func (a *SimpleTestAgent) WaitReady() {
	<-a.ready
}

// Accept accepts a connection (for testing)
func (a *SimpleTestAgent) Accept() (net.Conn, error) {
	if a.listener == nil {
		return nil, fmt.Errorf("agent not started")
	}
	return a.listener.Accept()
}

// Serve runs a simple echo server for testing with proper concurrent connection handling
func (a *SimpleTestAgent) Serve(ctx context.Context, handler func(net.Conn)) error {
	if err := a.Start(); err != nil {
		return err
	}
	defer a.Stop()
	
	// Track active connections to ensure proper cleanup
	var activeConnections sync.WaitGroup
	defer activeConnections.Wait() // Wait for all connections to finish
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Set accept timeout to allow checking context periodically
			if deadline, ok := ctx.Deadline(); ok {
				a.listener.(*net.UnixListener).SetDeadline(deadline)
			} else {
				a.listener.(*net.UnixListener).SetDeadline(time.Now().Add(100 * time.Millisecond))
			}
			
			conn, err := a.Accept()
			if err != nil {
				// Check if it's a timeout error and context is still active
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					if ctx.Err() == nil {
						continue // Context still active, continue accepting
					}
				}
				// Context cancelled or real error
				return err
			}
			
			// Clear deadline after successful accept
			a.listener.(*net.UnixListener).SetDeadline(time.Time{})
			
			activeConnections.Add(1)
			go func(connection net.Conn) {
				defer activeConnections.Done()
				defer connection.Close()
				
				// Create a context for this connection that can be cancelled
				connCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				
				// Monitor for context cancellation
				done := make(chan struct{})
				go func() {
					defer close(done)
					defer func() {
						if r := recover(); r != nil {
							// Log panic but don't crash the server
							// This allows tests to verify panic handling
						}
					}()
					if handler != nil {
						handler(connection)
					} else {
						// Simple echo by default with timeout
						connection.SetDeadline(time.Now().Add(5 * time.Second))
						buf := make([]byte, 1024)
						n, err := connection.Read(buf)
						if err == nil {
							connection.Write(buf[:n])
						}
					}
				}()
				
				// Wait for handler to finish or context cancellation
				select {
				case <-done:
					// Handler completed normally
				case <-connCtx.Done():
					// Context cancelled, close connection to interrupt handler
					connection.Close()
					<-done // Wait for handler to finish
				}
			}(conn)
		}
	}
}