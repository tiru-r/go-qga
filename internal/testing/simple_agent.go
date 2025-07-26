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
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// SimpleAgent provides an optimistic test agent that just works
type SimpleAgent struct {
	listener   net.Listener
	socketPath string
	commands   map[string]func() any
	running    chan struct{}
	wg         sync.WaitGroup
}

// NewSimpleAgent creates a new optimistic test agent with smart defaults
func NewSimpleAgent(socketPath string) *SimpleAgent {
	return &SimpleAgent{
		socketPath: socketPath,
		commands: map[string]func() any{
			"guest-get-host-name": func() any {
				return map[string]any{
					"return": map[string]string{"name": "test-vm"},
				}
			},
		},
		running: make(chan struct{}, 1),
	}
}

// AddCommand adds a command handler optimistically
func (a *SimpleAgent) AddCommand(name string, handler func() any) *SimpleAgent {
	a.commands[name] = handler
	return a // Fluent interface for chaining
}

// WithHostname sets a custom hostname response optimistically
func (a *SimpleAgent) WithHostname(hostname string) *SimpleAgent {
	return a.AddCommand("guest-get-host-name", func() any {
		return map[string]any{
			"return": map[string]string{"name": hostname},
		}
	})
}

// WithDelay adds a delay to all responses (useful for testing timeouts)
func (a *SimpleAgent) WithDelay(delay time.Duration) *SimpleAgent {
	originalCommands := make(map[string]func() any)
	for name, handler := range a.commands {
		originalCommands[name] = handler
	}

	for name, handler := range originalCommands {
		a.commands[name] = func() any {
			time.Sleep(delay)
			return handler()
		}
	}

	return a
}

// Start starts the agent optimistically - it will just work
func (a *SimpleAgent) Start() common.Result[string] {
	// Clean up any existing socket file optimistically
	os.Remove(a.socketPath)

	// Create listener
	listener, err := net.Listen("unix", a.socketPath)
	if err != nil {
		return common.Failure[string]("failed to create listener: %v", err)
	}

	a.listener = listener

	// Start accepting connections optimistically
	go a.acceptConnections()

	// Wait a tiny bit to ensure we're ready
	time.Sleep(10 * time.Millisecond)

	return common.Success(a.socketPath)
}

// Stop stops the agent gracefully and optimistically
func (a *SimpleAgent) Stop() {
	if a.listener != nil {
		close(a.running) // Signal shutdown
		a.listener.Close()
		a.wg.Wait()             // Wait for all connections to finish
		os.Remove(a.socketPath) // Clean up
	}
}

// acceptConnections handles incoming connections optimistically
func (a *SimpleAgent) acceptConnections() {
	defer a.listener.Close()

	for {
		select {
		case <-a.running:
			return // Graceful shutdown
		default:
			// Set short timeout for accepts to allow graceful shutdown
			if tcpListener, ok := a.listener.(*net.UnixListener); ok {
				tcpListener.SetDeadline(common.GlobalTimeManager.NowPlus(100 * time.Millisecond))
			}

			conn, err := a.listener.Accept()
			if err != nil {
				select {
				case <-a.running:
					return // Expected during shutdown
				default:
					continue // Try again
				}
			}

			a.wg.Add(1)
			go a.handleConnection(conn)
		}
	}
}

// handleConnection handles a single connection optimistically
func (a *SimpleAgent) handleConnection(conn net.Conn) {
	defer a.wg.Done()
	defer conn.Close()

	// Send QMP banner optimistically
	banner := map[string]any{
		"QMP": map[string]any{
			"version": map[string]any{
				"qemu": map[string]any{
					"major": "8",
					"minor": "2",
					"micro": "0",
				},
				"package": "qemu-8.2.0",
			},
			"capabilities": []any{},
		},
	}

	bannerBytes := common.MarshalToBytesIgnoreError(banner)
	if len(bannerBytes) > 0 {
		conn.Write(append(bannerBytes, '\n'))
	}

	// Read and handle commands optimistically
	buffer := common.GlobalBufferPool.GetStandard()
	defer common.GlobalBufferPool.PutStandard(buffer)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			return // Connection closed
		}

		// Parse command optimistically
		var request map[string]any
		if err := json.Unmarshal(buffer[:n], &request); err != nil {
			continue // Skip malformed requests
		}

		// Handle command optimistically
		if cmdName, ok := request["execute"].(string); ok {
			if handler, exists := a.commands[cmdName]; exists {
				response := handler()
				responseBytes := common.MarshalToBytesIgnoreError(response)
				if len(responseBytes) > 0 {
					conn.Write(append(responseBytes, '\n'))
				}
			} else {
				// Unknown command - send simple error
				errorResponse := map[string]any{
					"error": map[string]any{
						"class": "CommandNotFound",
						"desc":  "Command not found: " + cmdName,
					},
				}
				if errorBytes, err := json.Marshal(errorResponse); err == nil {
					conn.Write(append(errorBytes, '\n'))
				}
			}
		}
	}
}

// QuickTest provides a simple way to test QMP operations
func QuickTest(socketPath string, testFunc func(string) bool) bool {
	agent := NewSimpleAgent(socketPath)

	result := agent.Start()
	if result.IsErr() {
		return false
	}
	defer agent.Stop()

	// Give the agent a moment to be ready
	time.Sleep(50 * time.Millisecond)

	return testFunc(result.Value())
}
