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
	"testing"
	"time"

	qgatesting "github.com/prevostcorentin/go-qga/internal/testing"
)

func TestQmpClient(t *testing.T) {
	// Create a simple test - look how easy this is!
	socketPath := qgatesting.BuildSocketPath("client-test")
	
	// Connect with one simple call
	client, err := Connect(socketPath)
	if err != nil {
		t.Skipf("Failed to connect (no agent running): %v", err)
		return
	}
	defer client.Close()

	// Get hostname with one simple call  
	hostname, err := client.GetHostname()
	if err != nil {
		t.Skipf("Failed to get hostname: %v", err)
		return
	}

	// Just verify we got a hostname back
	if hostname == "" {
		t.Error("Expected hostname, got empty string")
	}
}

func TestQmpClientWithAgent(t *testing.T) {
	// Create simple test agent
	behavior := qgatesting.DefaultAgentBehavior()
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Simple connection and test
	client, err := Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	hostname, err := client.GetHostname()
	if err != nil {
		t.Fatalf("Failed to get hostname: %v", err)
	}

	if hostname != "fake-vm" {
		t.Errorf("Expected 'fake-vm', got '%s'", hostname)
	}
}

func TestQmpClientWithTimeout(t *testing.T) {
	// Create slow agent to test timeout handling
	behavior := qgatesting.SlowAgentBehavior(2 * time.Second)
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	client, err := Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test that we handle slow responses gracefully
	start := time.Now()
	hostname, err := client.GetHostname()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Failed to get hostname: %v", err)
		return
	}

	if duration < 2*time.Second {
		t.Errorf("Expected delay of at least 2s, got %v", duration)
	}

	if hostname != "slow-fake-vm" {
		t.Errorf("Expected 'slow-fake-vm', got '%s'", hostname)
	}
}

func TestQmpClientErrorHandling(t *testing.T) {
	// Test connection to non-existent socket
	client, err := Connect("/tmp/does-not-exist.sock")
	if err == nil {
		t.Error("Expected error when connecting to non-existent socket")
		if client != nil {
			client.Close()
		}
		return
	}

	// Error should be informative
	t.Logf("Expected error: %v", err)
}

func BenchmarkQmpClient(b *testing.B) {
	behavior := qgatesting.DefaultAgentBehavior()
	agent, cleanup := qgatesting.SetupAgentWithBehavior(b, behavior)
	defer cleanup()

	client, err := Connect(agent.Path())
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hostname, err := client.GetHostname()
		if err != nil {
			b.Errorf("Failed to get hostname: %v", err)
			break
		}
		_ = hostname
	}
}