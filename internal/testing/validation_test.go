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
	"testing"
	"time"
)

func TestConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name          string
		config        *SocketAgentConfig
		shouldPanic   bool
		expectedError string
	}{
		{
			name:        "nil_config",
			config:      nil,
			shouldPanic: true,
		},
		{
			name: "empty_socket_path",
			config: &SocketAgentConfig{
				SocketPath: "",
			},
			shouldPanic: true,
		},
		{
			name: "negative_max_connections",
			config: &SocketAgentConfig{
				SocketPath:     "/tmp/test.sock",
				MaxConnections: -1,
			},
			shouldPanic: true,
		},
		{
			name: "too_large_max_connections",
			config: &SocketAgentConfig{
				SocketPath:     "/tmp/test.sock",
				MaxConnections: 2000000,
			},
			shouldPanic: true,
		},
		{
			name: "negative_timeout",
			config: &SocketAgentConfig{
				SocketPath:  "/tmp/test.sock",
				ReadTimeout: -1 * time.Second,
			},
			shouldPanic: true,
		},
		{
			name: "negative_buffer_size",
			config: &SocketAgentConfig{
				SocketPath: "/tmp/test.sock",
				BufferSize: -1,
			},
			shouldPanic: true,
		},
		{
			name: "valid_config",
			config: &SocketAgentConfig{
				SocketPath:     "/tmp/test.sock",
				MaxConnections: 100,
				ReadTimeout:    5 * time.Second,
				BufferSize:     1024,
			},
			shouldPanic: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tc.shouldPanic && r == nil {
					t.Error("Expected panic but none occurred")
				} else if !tc.shouldPanic && r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			agent := NewSocketAgentWithConfig(tc.config)
			if !tc.shouldPanic && agent == nil {
				t.Error("Expected valid agent but got nil")
			}
		})
	}
}

func TestPathValidation(t *testing.T) {
	testCases := []struct {
		name        string
		socketPath  string
		shouldError bool
	}{
		{
			name:        "path_traversal_attack",
			socketPath:  "/tmp/../etc/passwd",
			shouldError: true,
		},
		{
			name:        "unsafe_system_path",
			socketPath:  "/etc/shadow",
			shouldError: true,
		},
		{
			name:        "safe_tmp_path",
			socketPath:  "/tmp/test.sock",
			shouldError: false,
		},
		{
			name:        "safe_var_tmp_path",
			socketPath:  "/var/tmp/test.sock",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &SocketAgentConfig{
				SocketPath:     tc.socketPath,
				BufferSize:     -1,
				MaxConnections: -1,
				ReadTimeout:    0,
			}
			agent := NewSocketAgentWithConfig(config)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := agent.Serve(ctx, func(conn net.Conn) {})

			if tc.shouldError && err == nil {
				t.Error("Expected error but none occurred")
			} else if !tc.shouldError && err != nil && err != context.DeadlineExceeded {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestConnectionLimitingRaceCondition(t *testing.T) {
	socketPath := BuildSocketPath(t)
	config := &SocketAgentConfig{
		SocketPath:     socketPath,
		MaxConnections: 2, // Very low limit to trigger race
		BufferSize:     -1,
		ReadTimeout:    0,
	}

	agent := NewSocketAgentWithConfig(config)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		agent.Serve(ctx, func(conn net.Conn) {
			// Hold the connection open longer to trigger limiting
			time.Sleep(500 * time.Millisecond)
		})
	}()

	agent.WaitReady()

	// Try to create many connections rapidly to test race condition
	const numGoroutines = 5
	results := make(chan bool, numGoroutines)

	// Start all connections nearly simultaneously
	for i := 0; i < numGoroutines; i++ {
		go func() {
			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				results <- false
				return
			}
			defer conn.Close()

			// Try to write something to see if connection is actually handled
			_, writeErr := conn.Write([]byte("test"))
			results <- writeErr == nil

			// Keep connection alive briefly
			time.Sleep(200 * time.Millisecond)
		}()
	}

	// Give time for connections to establish
	time.Sleep(100 * time.Millisecond)

	// Check current connection count during peak usage
	activeCount := agent.GetActiveConnections()
	if activeCount > 2 {
		t.Errorf("Active connection count exceeded limit: got %d, max should be 2", activeCount)
	}

	// Collect final results
	connected := 0
loop: // label the loop
	for i := 0; i < numGoroutines; i++ {
		select {
		case success := <-results:
			if success {
				connected++
			}
		case <-time.After(time.Second):
			break loop // break out of the labelled loop
		}
	}

	// Verify final internal counter is accurate
	finalActiveCount := agent.GetActiveConnections()
	if finalActiveCount < 0 || finalActiveCount > 2 {
		t.Errorf("Invalid final active connection count: %d", finalActiveCount)
	}
}
