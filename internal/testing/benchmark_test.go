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
	"sync"
	"testing"
	"time"
)

func BenchmarkSocketAgentThroughput(b *testing.B) {
	socketPath := BuildSocketPath("bench-agent")
	config := SocketAgentConfig{
		SocketPath: socketPath,
		Timeout:    30 * time.Second,
	}
	agent := NewSocketAgent(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processed int64
	var mu sync.Mutex

	handler := func(conn net.Conn) {
		defer conn.Close()
		mu.Lock()
		processed++
		mu.Unlock()

		// Simulate some work
		conn.Write([]byte("response"))
	}

	go func() {
		agent.Serve(ctx, handler)
	}()

	agent.WaitReady()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				b.Errorf("Failed to connect: %v", err)
				continue
			}

			conn.Write([]byte("request"))
			buffer := make([]byte, 1024)
			conn.Read(buffer)
			conn.Close()
		}
	})

	b.StopTimer()
	b.Logf("Processed %d connections", processed)
}

func BenchmarkAgentBehaviorFactory(b *testing.B) {
	behaviors := []*AgentBehavior{
		DefaultAgentBehavior(),
		SlowAgentBehavior(1 * time.Microsecond),
		ErrorAgentBehavior(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		behavior := behaviors[i%len(behaviors)]
		handler := CreateQmpHandler(b, behavior)

		// Simulate handler creation overhead
		_ = handler
	}
}

func BenchmarkConcurrentAgents(b *testing.B) {
	const numAgents = 10

	agents := make([]*SimpleTestAgent, numAgents)
	cleanups := make([]func(), numAgents)

	for i := 0; i < numAgents; i++ {
		agent, cleanup := SetupAgentWithBehavior(b, DefaultAgentBehavior())
		agents[i] = agent
		cleanups[i] = cleanup
	}

	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			agentIndex := 0 // Simple round-robin, could be randomized
			socketPath := GetSocketPath(agents[agentIndex])

			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				b.Errorf("Failed to connect to agent %d: %v", agentIndex, err)
				continue
			}

			conn.Write([]byte(`{"execute":"guest-get-host-name"}`))
			buffer := make([]byte, 1024)
			conn.Read(buffer)
			conn.Close()
		}
	})
}
