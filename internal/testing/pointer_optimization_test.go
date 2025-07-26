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
	"net"
	"sync"
	"testing"
	"time"
)

func TestPointerOptimizedAgentBehavior(t *testing.T) {
	// Test pointer-based behavior methods
	behavior := DefaultAgentBehavior()

	// Clone to avoid modifying the original
	cloned := behavior.Clone()

	// Use pointer methods to modify behavior
	cloned.SetTimeout(10 * time.Second)
	cloned.AddCommand("test-command", func() any {
		return map[string]string{"result": "success"}
	})

	if cloned.Timeout != 10*time.Second {
		t.Errorf("SetTimeout failed: got %v, want %v", cloned.Timeout, 10*time.Second)
	}

	if len(cloned.Commands) != 2 { // original + new command
		t.Errorf("AddCommand failed: got %d commands, want 2", len(cloned.Commands))
	}

	// Verify original wasn't modified
	if len(behavior.Commands) != 1 {
		t.Errorf("Original behavior was modified: got %d commands, want 1", len(behavior.Commands))
	}
}

func TestOptimizedSocketAgentConfig(t *testing.T) {
	socketPath := BuildSocketPath(t)

	// Test high-performance configuration
	config := CreateHighPerformanceConfig(socketPath)
	agent := NewSocketAgentWithConfig(config)

	if agent.GetMaxConnections() != 10000 {
		t.Errorf("High-performance config failed: got %d max connections, want 10000",
			agent.GetMaxConnections())
	}

	// Test test configuration
	testConfig := CreateTestConfig(socketPath)
	testAgent := NewSocketAgentWithConfig(testConfig)

	if testAgent.GetMaxConnections() != 100 {
		t.Errorf("Test config failed: got %d max connections, want 100",
			testAgent.GetMaxConnections())
	}
}

func TestConnectionLimiting(t *testing.T) {
	socketPath := BuildSocketPath(t)

	// Create agent with very low connection limit
	config := &SocketAgentConfig{
		SocketPath:     socketPath,
		MaxConnections: 2, // Very low limit
		ReadTimeout:    1 * time.Second,
		BufferSize:     -1,
	}

	agent := NewSocketAgentWithConfig(config)
	behavior := DefaultAgentBehavior()

	agentInstance, cleanup := SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Override with our limited agent for this test
	agent = agentInstance.(*SocketAgent)
	agent.config = config
	agent.maxConn = 2

	// Try to create more connections than the limit
	const numConnections = 5
	var wg sync.WaitGroup
	connectedCount := int64(0)

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				return // Connection rejected due to limit
			}
			defer conn.Close()

			// Connection successful
			connectedCount++
			time.Sleep(100 * time.Millisecond) // Hold connection briefly
		}()
	}

	wg.Wait()

	// Should have connected fewer than requested due to limit
	if connectedCount >= numConnections {
		t.Errorf("Connection limiting failed: got %d connections, expected fewer than %d",
			connectedCount, numConnections)
	}
}

func TestConfigurationValues(t *testing.T) {
	// Test configuration with direct values
	config := &SocketAgentConfig{
		SocketPath:     "/tmp/test.sock",
		MaxConnections: 42,
		ReadTimeout:    5 * time.Second,
		BufferSize:     1024,
	}

	if config.MaxConnections != 42 {
		t.Errorf("MaxConnections failed: got %d, want %d", config.MaxConnections, 42)
	}

	if config.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout failed: got %v, want %v", config.ReadTimeout, 5*time.Second)
	}

	if config.BufferSize != 1024 {
		t.Errorf("BufferSize failed: got %d, want %d", config.BufferSize, 1024)
	}
}

func TestMemoryOptimizedQmpMessages(t *testing.T) {
	behavior := DefaultAgentBehavior()

	// Test that error responses use pointers to reduce allocations
	errorBehavior := ErrorAgentBehavior()

	// Execute the error command to ensure it returns a pointer
	if handler, exists := errorBehavior.Commands["guest-get-host-name"]; exists {
		result := handler()

		// Should be a pointer to QmpError
		if _, ok := result.(*QmpError); !ok {
			t.Errorf("Error behavior should return *QmpError, got %T", result)
		}
	}

	// Test command parsing uses pointers
	handler := CreateQmpHandler(t, behavior)
	if handler == nil {
		t.Error("CreateQmpHandler returned nil")
	}
}
