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
	"fmt"
	"testing"
	"time"
)

func TestPerfectAgent(t *testing.T) {
	// Look how simple this is!
	agent := Perfect(BuildSocketPath(t))

	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start perfect agent: %v", result.Error())
	}
	defer agent.Stop()

	// Verify it's working
	stats := agent.Stats()
	if stats["max_connections"] != 10000 {
		t.Errorf("Expected 10000 max connections, got %v", stats["max_connections"])
	}

	if stats["active_connections"] != 0 {
		t.Errorf("Expected 0 active connections, got %v", stats["active_connections"])
	}
}

func TestFastAgent(t *testing.T) {
	// Fast configuration with one call
	agent := Fast(BuildSocketPath(t))

	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start fast agent: %v", result.Error())
	}
	defer agent.Stop()

	stats := agent.Stats()
	if stats["max_connections"] != 100000 {
		t.Errorf("Expected 100000 max connections, got %v", stats["max_connections"])
	}

	if stats["buffer_size"] != 16384 {
		t.Errorf("Expected 16384 buffer size, got %v", stats["buffer_size"])
	}
}

func TestSimpleAgent(t *testing.T) {
	// Simple configuration for reliability
	agent := Simple(BuildSocketPath(t))

	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start simple agent: %v", result.Error())
	}
	defer agent.Stop()

	stats := agent.Stats()
	if stats["max_connections"] != 100 {
		t.Errorf("Expected 100 max connections, got %v", stats["max_connections"])
	}
}

func TestFluentConfiguration(t *testing.T) {
	// Fluent API - chain configurations beautifully
	agent := Perfect(BuildSocketPath(t)).
		WithConnections(5000).
		WithTimeout(15 * time.Second).
		WithBuffer(12288)

	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start fluent agent: %v", result.Error())
	}
	defer agent.Stop()

	stats := agent.Stats()
	if stats["max_connections"] != 5000 {
		t.Errorf("Expected 5000 max connections, got %v", stats["max_connections"])
	}
}

func TestOptimisticBounds(t *testing.T) {
	// Test that bounds checking works optimistically
	agent := Perfect(BuildSocketPath(t)).
		WithConnections(-1).      // Invalid - should ignore
		WithConnections(2000000). // Too big - should ignore
		WithConnections(1000)     // Valid - should use

	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start bounded agent: %v", result.Error())
	}
	defer agent.Stop()

	stats := agent.Stats()
	if stats["max_connections"] != 1000 {
		t.Errorf("Expected 1000 max connections (optimistic bounds), got %v", stats["max_connections"])
	}
}

func TestAgentBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark test in short mode")
	}

	agent := Fast(BuildSocketPath(t))
	result := agent.Start()
	if result.IsErr() {
		t.Fatalf("Failed to start agent: %v", result.Error())
	}
	defer agent.Stop()

	// Run a quick benchmark
	results := agent.Benchmark(10, 100*time.Millisecond)

	t.Logf("Benchmark results:")
	for key, value := range results {
		t.Logf("  %s: %v", key, value)
	}

	// Verify we got reasonable results
	if successful, ok := results["successful"].(int64); ok {
		if successful < 8 { // Allow for some failures
			t.Errorf("Expected at least 8 successful connections, got %d", successful)
		}
	}

	if successRate, ok := results["success_rate"].(float64); ok {
		if successRate < 80.0 { // 80% success rate minimum
			t.Errorf("Expected at least 80%% success rate, got %.1f%%", successRate)
		}
	}
}

func BenchmarkPerfectAgent(b *testing.B) {
	agent := Perfect(BuildSocketPath(b))
	result := agent.Start()
	if result.IsErr() {
		b.Fatalf("Failed to start agent: %v", result.Error())
	}
	defer agent.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simple connection test
			stats := agent.Stats()
			if stats["path"] == "" {
				b.Error("Expected valid path in stats")
			}
		}
	})
}

func ExamplePerfect() {
	// This is how simple it should be!
	agent := Perfect("/tmp/my-qga.sock")

	result := agent.Start()
	if result.IsErr() {
		fmt.Printf("Failed to start: %v\n", result.Error())
		return
	}
	defer agent.Stop()

	fmt.Printf("Agent started on: %s\n", agent.Path())

	// Output:
	// Agent started on: /tmp/my-qga.sock
}

func ExampleFast() {
	// High-performance configuration
	agent := Fast("/tmp/fast-qga.sock")

	result := agent.Start()
	if result.IsErr() {
		return
	}
	defer agent.Stop()

	// Run a quick benchmark
	results := agent.Benchmark(100, 1*time.Second)
	fmt.Printf("Processed %.0f requests/sec\n", results["requests_per_sec"])
}

func ExampleSimple() {
	// Reliable, simple configuration
	agent := Simple("/tmp/simple-qga.sock").
		WithTimeout(2 * time.Minute) // Extra reliable

	agent.Start()
	defer agent.Stop()

	fmt.Println("Simple agent running reliably!")
	// Output: Simple agent running reliably!
}
