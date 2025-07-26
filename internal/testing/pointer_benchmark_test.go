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
	"testing"
	"time"
)

// Benchmark struct vs pointer allocation
func BenchmarkStructCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = QmpCommand{Execute: "test-command"}
	}
}

func BenchmarkPointerCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = &QmpCommand{Execute: "test-command"}
	}
}

// Benchmark behavior cloning
func BenchmarkBehaviorClone(b *testing.B) {
	original := DefaultAgentBehavior()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone := original.Clone()
		clone.AddCommand("bench-cmd", func() any { return "result" })
	}
}

// Benchmark configuration creation
func BenchmarkConfigCreation(b *testing.B) {
	socketPath := "/tmp/bench-socket"

	b.Run("HighPerformance", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = CreateHighPerformanceConfig(socketPath)
		}
	})

	b.Run("TestConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = CreateTestConfig(socketPath)
		}
	})
}

// Benchmark JSON marshalling with pointers vs values
func BenchmarkJSONMarshalStruct(b *testing.B) {
	cmd := QmpCommand{Execute: "guest-get-host-name"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(cmd)
	}
}

func BenchmarkJSONMarshalPointer(b *testing.B) {
	cmd := &QmpCommand{Execute: "guest-get-host-name"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(cmd)
	}
}

// Benchmark agent creation patterns
func BenchmarkAgentCreation(b *testing.B) {
	socketPath := "/tmp/test-socket"

	b.Run("Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			agent := NewSocketAgent(socketPath)
			_ = agent
		}
	})

	b.Run("WithConfig", func(b *testing.B) {
		config := CreateTestConfig(socketPath)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			agent := NewSocketAgentWithConfig(config)
			_ = agent
		}
	})
}

// Benchmark direct value configuration creation
func BenchmarkDirectValueConfig(b *testing.B) {
	b.Run("DirectValues", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = &SocketAgentConfig{
				SocketPath:     "/tmp/test.sock",
				MaxConnections: 42,
				ReadTimeout:    5 * time.Second,
				BufferSize:     1024,
			}
		}
	})
}
