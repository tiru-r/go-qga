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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// Simple testing interface that works with both T and B
type TestingT interface {
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
	Helper()
}

// Simple agent interface
type Agent interface {
	Start() error
	Stop() error
	Path() string
	WaitReady()
	Accept() (net.Conn, error)
}

// Simple socket agent config
type SocketAgentConfig struct {
	SocketPath     string
	Timeout        time.Duration
	BufferSize     int
	MaxConnections int
	ReadTimeout    time.Duration
}

// Simple socket agent interface
type SocketAgent interface {
	Start() error
	Stop() error
	Path() string
	WaitReady()
	Accept() (net.Conn, error)
	Serve(context.Context, func(net.Conn)) error
}

// BuildSocketPath builds a socket path
func BuildSocketPath(name string) string {
	return fmt.Sprintf("/tmp/%s.sock", name)
}

// NewSocketAgent creates a new socket agent with validation
func NewSocketAgent(config SocketAgentConfig) *SimpleTestAgent {
	// Validate configuration - panic on invalid configs as expected by tests
	if config.SocketPath == "" {
		panic("socket path cannot be empty")
	}
	
	if config.MaxConnections < -1 {
		panic("max connections cannot be negative (except -1 for default)")
	}
	
	if config.MaxConnections > 1000000 {
		panic("max connections too large")
	}
	
	if config.ReadTimeout < 0 {
		panic("read timeout cannot be negative")
	}
	
	if config.BufferSize < -1 {
		panic("buffer size cannot be negative (except -1 for default)")
	}
	
	return NewSimpleTestAgent(config.SocketPath)
}


// QMP message types for reusability
type QemuVersion struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
	Micro string `json:"micro"`
}

type QmpVersion struct {
	Qemu    QemuVersion `json:"qemu"`
	Package string      `json:"package"`
}

type QmpInfo struct {
	Version      QmpVersion `json:"version"`
	Capabilities []any      `json:"capabilities"`
}

type QmpBannerResponse struct {
	Qmp QmpInfo `json:"QMP"`
}

type QmpCommand struct {
	Execute string `json:"execute"`
}

type QmpError struct {
	Error struct {
		Class       string `json:"class"`
		Description string `json:"desc"`
	} `json:"error"`
}

// Agent behavior configuration
type AgentBehavior struct {
	Commands       map[string]func() any
	Timeout        time.Duration
	Banner         QmpBannerResponse
	ErrorOnUnknown bool
}

// AddCommand adds a new command handler to the behavior
func (b *AgentBehavior) AddCommand(name string, handler func() any) {
	if b.Commands == nil {
		b.Commands = make(map[string]func() any)
	}
	b.Commands[name] = handler
}

// SetTimeout configures the timeout for the behavior
func (b *AgentBehavior) SetTimeout(timeout time.Duration) {
	b.Timeout = timeout
}

// SetBanner configures the QMP banner for the behavior
func (b *AgentBehavior) SetBanner(banner QmpBannerResponse) {
	b.Banner = banner
}

// Clone creates a deep copy of the behavior
func (b *AgentBehavior) Clone() *AgentBehavior {
	clone := &AgentBehavior{
		Commands:       make(map[string]func() any),
		Timeout:        b.Timeout,
		Banner:         b.Banner, // Shared reference is OK for banner
		ErrorOnUnknown: b.ErrorOnUnknown,
	}

	maps.Copy(clone.Commands, b.Commands)

	return clone
}

// Default behaviors
func DefaultAgentBehavior() *AgentBehavior {
	return &AgentBehavior{
		Commands: map[string]func() any{
			common.CommandGuestGetHostName: func() any {
				return map[string]any{
					common.JSONFieldReturn: map[string]string{common.JSONFieldName: "fake-vm"},
				}
			},
		},
		Timeout:        5 * time.Second,
		Banner:         QmpBannerResponse{},
		ErrorOnUnknown: true,
	}
}

func SlowAgentBehavior(delay time.Duration) *AgentBehavior {
	behavior := DefaultAgentBehavior()
	behavior.AddCommand(common.CommandGuestGetHostName, func() any {
		time.Sleep(delay)
		return map[string]any{
			common.JSONFieldReturn: map[string]string{common.JSONFieldName: "slow-fake-vm"},
		}
	})
	return behavior
}

func ErrorAgentBehavior() *AgentBehavior {
	return &AgentBehavior{
		Commands: map[string]func() any{
			common.CommandGuestGetHostName: func() any {
				return &QmpError{
					Error: struct {
						Class       string `json:"class"`
						Description string `json:"desc"`
					}{
						Class:       "GenericError",
						Description: "Simulated error",
					},
				}
			},
		},
		Timeout:        5 * time.Second,
		Banner:         QmpBannerResponse{},
		ErrorOnUnknown: false,
	}
}

// QMP connection handler factory
func CreateQmpHandler(t TestingT, behavior *AgentBehavior) func(net.Conn) {
	return func(conn net.Conn) {
		defer conn.Close()
		writer := bufio.NewWriter(conn)
		reader := bufio.NewReader(conn)

		// Send banner
		bannerBytes, _ := json.Marshal(&behavior.Banner)
		fmt.Fprintln(writer, string(bannerBytes))
		writer.Flush()

		// Handle multiple commands on the same connection
		for {
			// Set read timeout for each command
			if behavior.Timeout > 0 {
				common.GlobalTimeManager.SetReadDeadline(conn, behavior.Timeout)
			}

			// Read command
			line, err := reader.ReadBytes(0x0A)
			if err != nil {
				// Connection closed or timeout - normal termination
				return
			}

			command := &QmpCommand{}
			if err := json.Unmarshal(line, command); err != nil {
				if behavior.ErrorOnUnknown {
					t.Logf("Error unmarshalling command: %v", err)
				}
				return
			}

			// Process command
			var response any
			if handler, exists := behavior.Commands[command.Execute]; exists {
				response = handler()
			} else if behavior.ErrorOnUnknown {
				response = &QmpError{
					Error: struct {
						Class       string `json:"class"`
						Description string `json:"desc"`
					}{
						Class:       "CommandNotFound",
						Description: "Command '" + command.Execute + "' not found",
					},
				}
			} else {
				// Unknown command but not erroring, send empty response
				response = map[string]any{}
			}

			// Send response
			responseBytes, err := json.Marshal(response)
			if err != nil {
				t.Errorf("Error marshalling response: %v", err)
				return
			}

			fmt.Fprintln(writer, string(responseBytes))
			if err := writer.Flush(); err != nil {
				// Connection broken, normal termination
				return
			}
		}
	}
}

// Test helper for setting up agent with custom behavior
func SetupAgentWithBehavior(t TestingT, behavior *AgentBehavior) (*SimpleTestAgent, func()) {
	socketPath := BuildSocketPath("test-agent")
	config := SocketAgentConfig{
		SocketPath: socketPath,
		Timeout:    behavior.Timeout,
	}
	agent := NewSocketAgent(config)

	ctx, cancel := context.WithTimeout(context.Background(), behavior.Timeout)

	handler := CreateQmpHandler(t, behavior)

	go func() {
		if err := agent.Serve(ctx, handler); err != nil &&
			err != context.DeadlineExceeded && err != context.Canceled {
			t.Logf("agent serve error (may be normal during cleanup): %v", err)
		}
	}()

	agent.WaitReady()

	cleanup := func() {
		cancel()
	}

	return agent, cleanup
}

// Get socket path from agent
func GetSocketPath(agent *SimpleTestAgent) string {
	return agent.Path()
}

// CreateHighPerformanceConfig creates an optimized SocketAgent configuration
func CreateHighPerformanceConfig(socketPath string) *SocketAgentConfig {
	return &SocketAgentConfig{
		SocketPath:     socketPath,
		BufferSize:     8192,             // 8KB buffer
		MaxConnections: 10000,            // High connection limit
		ReadTimeout:    30 * time.Second, // 30s timeout
	}
}

// CreateTestConfig creates a configuration optimized for testing
func CreateTestConfig(socketPath string) *SocketAgentConfig {
	return &SocketAgentConfig{
		SocketPath:     socketPath,
		BufferSize:     1024,            // 1KB buffer for tests
		MaxConnections: 100,             // Lower limit for tests
		ReadTimeout:    5 * time.Second, // 5s timeout
	}
}
