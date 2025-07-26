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
	"testing"
	"time"

	"github.com/prevostcorentin/go-qga/internal/qmp"
)

// TestUltimateSimplicity shows how simple and perfect everything is now
func TestUltimateSimplicity(t *testing.T) {
	// Look how simple this is - just 3 lines for a complete QMP test!

	// 1. Start perfect agent
	agent := Perfect(BuildSocketPath(t)).Start().Value()
	defer agent.Stop()

	// 2. Connect and get hostname
	hostname := qmp.Connect(agent.Path()).Value().GetHostname().Value()

	// 3. Verify result
	if hostname != "perfect-vm" {
		t.Errorf("Expected 'perfect-vm', got '%s'", hostname)
	}

	// That's it! Perfect simplicity.
}

// TestOptimisticChaining demonstrates beautiful result chaining
func TestOptimisticChaining(t *testing.T) {
	// Simple chaining approach
	agent := Perfect(BuildSocketPath(t)).Start().Value()
	defer agent.Stop()

	client := qmp.Connect(agent.Path()).Value()
	defer client.Close()

	hostname := client.GetHostname().Map(func(h string) string {
		return "Hello, " + h + "!"
	})

	if hostname.IsErr() {
		t.Fatalf("Chain failed: %v", hostname.Error())
	}

	expected := "Hello, perfect-vm!"
	if hostname.Value() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, hostname.Value())
	}
}

// TestPerfectPerformance shows optimized performance with simple code
func TestPerfectPerformance(t *testing.T) {
	// High performance with simple configuration
	agent := Fast(BuildSocketPath(t)).Start().Value()
	defer agent.Stop()

	// Verify high-performance settings
	stats := agent.Stats()
	if stats["max_connections"] != 100000 {
		t.Errorf("Expected 100000 max connections, got %v", stats["max_connections"])
	}

	// Run performance test
	start := time.Now()
	client := qmp.Connect(agent.Path()).Value()
	defer client.Close()

	// Multiple rapid requests
	for i := 0; i < 10; i++ {
		hostname := client.GetHostname()
		if hostname.IsErr() {
			t.Errorf("Request %d failed: %v", i, hostname.Error())
			break
		}
	}

	duration := time.Since(start)
	requestsPerSec := float64(10) / duration.Seconds()

	t.Logf("Performance: %.0f requests/sec", requestsPerSec)

	// Should be reasonably fast
	if requestsPerSec < 50 {
		t.Errorf("Performance too slow: %.0f requests/sec", requestsPerSec)
	}
}

// TestRobustGracefulHandling demonstrates robust error handling
func TestRobustGracefulHandling(t *testing.T) {
	// Test connection to non-existent socket
	client := qmp.Connect("/tmp/does-not-exist.sock")

	// Should fail gracefully
	if client.IsOk() {
		t.Error("Expected connection to fail to non-existent socket")
	}

	// Error should be informative
	if client.Error() == nil {
		t.Error("Expected informative error message")
	}

	t.Logf("Graceful error: %v", client.Error())
}

// TestFluentConfigurationAPI shows beautiful fluent API
func TestFluentConfigurationAPI(t *testing.T) {
	// Fluent configuration is readable and powerful
	agent := Perfect(BuildSocketPath(t)).
		WithConnections(5000).         // Custom connection limit
		WithTimeout(10 * time.Second). // Custom timeout
		WithBuffer(16384).             // Custom buffer size
		Start().Value()                // Start and get value
	defer agent.Stop()

	// Verify all settings
	stats := agent.Stats()
	if stats["max_connections"] != 5000 {
		t.Errorf("Expected 5000 connections, got %v", stats["max_connections"])
	}

	if stats["buffer_size"] != 16384 {
		t.Errorf("Expected 16384 buffer, got %v", stats["buffer_size"])
	}

	// Should work perfectly
	hostname := qmp.Connect(agent.Path()).Value().GetHostname().Value()
	if hostname != "perfect-vm" {
		t.Errorf("Expected 'perfect-vm', got '%s'", hostname)
	}
}

// TestOptimisticDefaults shows smart defaults work perfectly
func TestOptimisticDefaults(t *testing.T) {
	// Just use Perfect() with no configuration - it should work great
	agent := Perfect(BuildSocketPath(t)).Start().Value()
	defer agent.Stop()

	stats := agent.Stats()

	// Verify sensible defaults
	if stats["max_connections"] != 10000 {
		t.Errorf("Expected default 10000 connections, got %v", stats["max_connections"])
	}

	if stats["buffer_size"] != 8192 {
		t.Errorf("Expected default 8192 buffer, got %v", stats["buffer_size"])
	}

	if stats["timeout"] != "30s" {
		t.Errorf("Expected default 30s timeout, got %v", stats["timeout"])
	}

	// Should work perfectly with defaults
	hostname := qmp.Connect(agent.Path()).Value().GetHostname().Value()
	if hostname != "perfect-vm" {
		t.Errorf("Expected 'perfect-vm', got '%s'", hostname)
	}
}

// BenchmarkUltimateSimplicity benchmarks the simple API
func BenchmarkUltimateSimplicity(b *testing.B) {
	agent := Perfect(BuildSocketPath(b)).Start().Value()
	defer agent.Stop()

	client := qmp.Connect(agent.Path()).Value()
	defer client.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hostname := client.GetHostname()
		if hostname.IsErr() {
			b.Errorf("Request failed: %v", hostname.Error())
			break
		}
	}
}

// BenchmarkPerfectAgentTypes benchmarks different agent types
func BenchmarkPerfectAgentTypes(b *testing.B) {
	b.Run("Perfect", func(b *testing.B) {
		agent := Perfect(BuildSocketPath(b)).Start().Value()
		defer agent.Stop()
		benchmarkAgent(b, agent)
	})

	b.Run("Fast", func(b *testing.B) {
		agent := Fast(BuildSocketPath(b)).Start().Value()
		defer agent.Stop()
		benchmarkAgent(b, agent)
	})

	b.Run("Simple", func(b *testing.B) {
		agent := Simple(BuildSocketPath(b)).Start().Value()
		defer agent.Stop()
		benchmarkAgent(b, agent)
	})
}

func benchmarkAgent(b *testing.B, agent *PerfectAgent) {
	client := qmp.Connect(agent.Path()).Value()
	defer client.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hostname := client.GetHostname()
		if hostname.IsErr() {
			b.Errorf("Request failed: %v", hostname.Error())
			break
		}
	}
}
