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

package qmp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/prevostcorentin/go-qga/internal/qmp"
	"github.com/prevostcorentin/go-qga/internal/qmp/transport"
	. "github.com/prevostcorentin/go-qga/internal/testing"
)

// Use reusable types from testing package
type QmpHostnameResponse struct {
	Return struct {
		Name string `json:"name"`
	} `json:"return"`
}

type fakeGuestAgent struct {
	listener net.Listener
	done     chan struct{}
	wg       sync.WaitGroup // Track active connections
	t        *testing.T
	path     string
}

func newFakeGuestAgent(t *testing.T) *fakeGuestAgent {
	return &fakeGuestAgent{t: t, done: make(chan struct{}), path: BuildSocketPath(t)}
}

func (agent *fakeGuestAgent) Start() {
	var listenerError error
	agent.listener, listenerError = net.Listen("unix", agent.Path())
	if listenerError != nil {
		agent.t.Fatalf("can't listen on %s: %v", agent.Path(), listenerError)
	}

	go func() {
		defer agent.listener.Close()

		for {
			connection, acceptError := agent.listener.Accept()
			if acceptError != nil {
				select {
				case <-agent.done:
					return
				default:
					// Log error instead of calling t.Fatalf from goroutine
					agent.t.Logf("accept error (may be normal during shutdown): %v", acceptError)
					return
				}
			}

			// Track each connection handler
			agent.wg.Add(1)
			go func(conn net.Conn) {
				defer agent.wg.Done()
				defer conn.Close()
				handleConnection(agent.t, conn)
			}(connection)
		}
	}()
}

func handleConnection(t *testing.T, connection net.Conn) {
	writer := bufio.NewWriter(connection)
	reader := bufio.NewReader(connection)

	banner := QmpBannerResponse{}
	bytes, _ := json.Marshal(banner)
	fmt.Fprintln(writer, string(bytes))
	writer.Flush()

	line, err := reader.ReadBytes(0x0A)
	t.Logf("%d bytes received", len(line))
	if err != nil {
		return
	}
	var command QmpCommand
	if err := json.Unmarshal(line, &command); err != nil {
		t.Fatalf("unmarshalling command: %v", err)
	}
	var response any
	if command.Execute == "guest-get-host-name" {
		qmpResponse := &QmpHostnameResponse{}
		qmpResponse.Return.Name = "fake-vm"
		response = qmpResponse
	} else {
		response = QmpError{}
	}
	bytes, err = json.Marshal(response)
	if err != nil {
		t.Fatalf("marshalling response: %v", err)
	}
	fmt.Fprintln(writer, string(bytes))
	t.Logf("%d bytes sent", len(bytes))
	writer.Flush()
}

func (agent *fakeGuestAgent) Path() string {
	return agent.path
}

func (agent *fakeGuestAgent) Stop() {
	// Signal shutdown
	close(agent.done)

	// Close listener to stop accepting new connections
	if err := agent.listener.Close(); err != nil {
		agent.t.Logf("error closing listener (may be normal): %v", err)
	}

	// Wait for all active connection handlers to complete
	agent.wg.Wait()
}

type hostNameCommand struct{}

func (command hostNameCommand) Execute() string {
	return "guest-get-host-name"
}

func (command hostNameCommand) Arguments() any {
	return nil
}

func (command hostNameCommand) Response() any {
	return &hostNameResponse{}
}

type hostNameResponse struct {
	Name string
}

func TestHostnameCommand(t *testing.T) {
	ctx := context.Background()
	agent := newFakeGuestAgent(t)
	socketPath := agent.Path()
	transport, err := transport.NewTransport(transport.Unix, socketPath)
	if err != nil {
		t.Fatalf("creating transport: %v", err)
	}
	agent.Start()
	qgaSocket, openErr := qmp.Open(ctx, socketPath, transport)
	if openErr != nil {
		t.Fatalf("while opening socket: %v", openErr)
	}
	defer qgaSocket.Close()

	command := hostNameCommand{}
	executor, err := qmp.NewExecutor(qgaSocket)
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	response, err := executor.Run(ctx, command)
	if err != nil {
		t.Fatalf("while running command: %v", err)
	}
	agent.Stop()
	typedResponse := response.(*hostNameResponse)
	if typedResponse.Name != "fake-vm" {
		t.Errorf(`vm name differs (got "%s", expecting "fake-vm")`, typedResponse.Name)
	}
}

func TestHostnameCommandWithStructuredAgent(t *testing.T) {
	socketPath := BuildSocketPath(t)
	agent := NewSocketAgent(socketPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	handler := func(conn net.Conn) {
		handleConnection(t, conn)
	}

	go func() {
		if err := agent.Serve(ctx, handler); err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("agent serve error: %v", err)
		}
	}()

	agent.WaitReady()

	transport, err := transport.NewTransport(transport.Unix, socketPath)
	if err != nil {
		t.Fatalf("creating transport: %v", err)
	}
	qgaSocket, openErr := qmp.Open(ctx, socketPath, transport)
	if openErr != nil {
		t.Fatalf("while opening socket: %v", openErr)
	}
	defer qgaSocket.Close()

	command := hostNameCommand{}
	executor, err := qmp.NewExecutor(qgaSocket)
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	response, err := executor.Run(ctx, command)
	if err != nil {
		t.Fatalf("while running command: %v", err)
	}

	typedResponse := response.(*hostNameResponse)
	if typedResponse.Name != "fake-vm" {
		t.Errorf(`vm name differs (got "%s", expecting "fake-vm")`, typedResponse.Name)
	}
}

// Robust test using the new testing infrastructure
func TestHostnameCommandRobust(t *testing.T) {
	testCases := []struct {
		name     string
		behavior *AgentBehavior
		expected string
	}{
		{
			name:     "default_behavior",
			behavior: DefaultAgentBehavior(),
			expected: "fake-vm",
		},
		{
			name:     "slow_response",
			behavior: SlowAgentBehavior(50 * time.Millisecond),
			expected: "slow-fake-vm",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			agent, cleanup := SetupAgentWithBehavior(t, tc.behavior)
			defer cleanup()

			socketPath := GetSocketPath(agent)
			transport, err := transport.NewTransport(transport.Unix, socketPath)
			if err != nil {
				t.Fatalf("creating transport: %v", err)
			}
			qgaSocket, openErr := qmp.Open(ctx, socketPath, transport)
			if openErr != nil {
				t.Fatalf("while opening socket: %v", openErr)
			}
			defer qgaSocket.Close()

			command := hostNameCommand{}
			executor, err := qmp.NewExecutor(qgaSocket)
			if err != nil {
				t.Fatalf("creating executor: %v", err)
			}
			response, err := executor.Run(ctx, command)
			if err != nil {
				t.Fatalf("while running command: %v", err)
			}

			typedResponse := response.(*hostNameResponse)
			if typedResponse.Name != tc.expected {
				t.Errorf("vm name differs (got %q, expecting %q)",
					typedResponse.Name, tc.expected)
			}
		})
	}
}

// Test for concurrent access scalability
func TestHostnameCommandConcurrent(t *testing.T) {
	ctx := context.Background()
	agent, cleanup := SetupAgentWithBehavior(t, DefaultAgentBehavior())
	defer cleanup()

	socketPath := GetSocketPath(agent)

	const numClients = 5
	results := make(chan string, numClients)
	errors := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			transport, err := transport.NewTransport(transport.Unix, socketPath)
			if err != nil {
				errors <- fmt.Errorf("client %d creating transport: %v", clientID, err)
				return
			}
			qgaSocket, openErr := qmp.Open(ctx, socketPath, transport)
			if openErr != nil {
				errors <- fmt.Errorf("client %d open error: %v", clientID, openErr)
				return
			}
			defer qgaSocket.Close()

			command := hostNameCommand{}
			executor, err := qmp.NewExecutor(qgaSocket)
			if err != nil {
				errors <- fmt.Errorf("client %d creating executor: %v", clientID, err)
				return
			}
			response, err := executor.Run(ctx, command)
			if err != nil {
				errors <- fmt.Errorf("client %d run error: %v", clientID, err)
				return
			}

			typedResponse := response.(*hostNameResponse)
			results <- typedResponse.Name
		}(i)
	}

	// Collect results
	for i := 0; i < numClients; i++ {
		select {
		case result := <-results:
			if result != "fake-vm" {
				t.Errorf("Unexpected result: %s", result)
			}
		case err := <-errors:
			t.Errorf("Concurrent execution error: %v", err)
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for concurrent results")
			return
		}
	}
}
