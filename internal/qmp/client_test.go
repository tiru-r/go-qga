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
	qgatesting.QuickTest(qgatesting.BuildSocketPath(t), func(socketPath string) bool {
		// Connect with one simple call
		client := Connect(socketPath)
		if client.IsErr() {
			t.Errorf("Failed to connect: %v", client.Error())
			return false
		}
		defer client.Value().Close()

		// Get hostname with one simple call
		hostname := client.Value().GetHostname()
		if hostname.IsErr() {
			t.Errorf("Failed to get hostname: %v", hostname.Error())
			return false
		}

		// Verify result optimistically
		if hostname.Value() != "test-vm" {
			t.Errorf("Expected 'test-vm', got '%s'", hostname.Value())
			return false
		}

		return true
	})
}

func TestQmpClientWithCustomHostname(t *testing.T) {
	socketPath := qgatesting.BuildSocketPath(t)

	// Create agent with custom hostname - fluent API!
	agent := qgatesting.NewSimpleAgent(socketPath).
		WithHostname("my-custom-vm")

	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start agent: %v", result.Error())
	}
	defer agent.Stop()

	// Simple connection and test
	client := Connect(socketPath)
	if client.IsErr() {
		t.Fatalf("Failed to connect: %v", client.Error())
	}
	defer client.Value().Close()

	hostname := client.Value().GetHostname()
	if hostname.IsErr() {
		t.Fatalf("Failed to get hostname: %v", hostname.Error())
	}

	if hostname.Value() != "my-custom-vm" {
		t.Errorf("Expected 'my-custom-vm', got '%s'", hostname.Value())
	}
}

func TestQmpClientWithTimeout(t *testing.T) {
	socketPath := qgatesting.BuildSocketPath(t)

	// Create slow agent to test timeout handling
	agent := qgatesting.NewSimpleAgent(socketPath).
		WithDelay(2 * time.Second) // 2 second delay

	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start agent: %v", result.Error())
	}
	defer agent.Stop()

	client := Connect(socketPath)
	if client.IsErr() {
		t.Fatalf("Failed to connect: %v", client.Error())
	}
	defer client.Value().Close()

	// Test that we handle slow responses gracefully
	start := time.Now()
	hostname := client.Value().GetHostname()
	duration := time.Since(start)

	if hostname.IsErr() {
		t.Errorf("Failed to get hostname: %v", hostname.Error())
		return
	}

	if duration < 2*time.Second {
		t.Errorf("Expected delay of at least 2s, got %v", duration)
	}

	if hostname.Value() != "test-vm" {
		t.Errorf("Expected 'test-vm', got '%s'", hostname.Value())
	}
}

func TestResultChaining(t *testing.T) {
	// Demonstrate optimistic result chaining
	qgatesting.QuickTest(qgatesting.BuildSocketPath(t), func(socketPath string) bool {
		client := Connect(socketPath)
		if client.IsErr() {
			t.Errorf("Failed to connect: %v", client.Error())
			return false
		}
		defer client.Value().Close()

		result := client.Value().GetHostname().Map(func(hostname string) string {
			return "Hello, " + hostname + "!"
		})

		if result.IsErr() {
			t.Errorf("Chain failed: %v", result.Error())
			return false
		}

		expected := "Hello, test-vm!"
		if result.Value() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result.Value())
			return false
		}

		return true
	})
}

func BenchmarkQmpClient(b *testing.B) {
	socketPath := qgatesting.BuildSocketPath(b)
	agent := qgatesting.NewSimpleAgent(socketPath)

	result := agent.Start()
	if result.IsErr() {
		b.Fatalf("Failed to start agent: %v", result.Error())
	}
	defer agent.Stop()

	client := Connect(socketPath)
	if client.IsErr() {
		b.Fatalf("Failed to connect: %v", client.Error())
	}
	defer client.Value().Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hostname := client.Value().GetHostname()
		if hostname.IsErr() {
			b.Errorf("Failed to get hostname: %v", hostname.Error())
			break
		}
	}
}
