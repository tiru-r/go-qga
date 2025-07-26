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

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/prevostcorentin/go-qga/internal/qmp"
	"github.com/prevostcorentin/go-qga/internal/testing"
)

func main() {
	fmt.Println("🎯 Perfect & Simple Go QGA Example")
	fmt.Println("=====================================")

	// 1. Start a perfect test agent - one line!
	fmt.Println("\n📡 Starting perfect test agent...")
	agent := testing.Perfect("/tmp/perfect-qga.sock")

	result := agent.Start()
	if result.IsErr() {
		log.Fatalf("❌ Failed to start agent: %v", result.Error())
	}
	defer agent.Stop()

	fmt.Printf("✅ Agent started successfully on: %s\n", agent.Path())

	// 2. Show agent statistics
	fmt.Println("\n📊 Agent Statistics:")
	stats := agent.Stats()
	for key, value := range stats {
		fmt.Printf("   %s: %v\n", key, value)
	}

	// 3. Connect with simple client - one line!
	fmt.Println("\n🔌 Connecting simple client...")
	client := qmp.Connect(agent.Path())
	if client.IsErr() {
		log.Fatalf("❌ Failed to connect: %v", client.Error())
	}
	defer client.Value().Close()

	fmt.Println("✅ Client connected successfully!")

	// 4. Get hostname - one simple call!
	fmt.Println("\n🏠 Getting VM hostname...")
	hostname := client.Value().GetHostname()
	if hostname.IsErr() {
		log.Fatalf("❌ Failed to get hostname: %v", hostname.Error())
	}

	fmt.Printf("✅ VM hostname: '%s'\n", hostname.Value())

	// 5. Run a performance test
	fmt.Println("\n🚀 Running performance benchmark...")
	benchResults := agent.Benchmark(50, 500*time.Millisecond)

	fmt.Println("📈 Benchmark Results:")
	for key, value := range benchResults {
		fmt.Printf("   %s: %v\n", key, value)
	}

	// 6. Show different agent types
	fmt.Println("\n⚡ Different Agent Types:")

	// Fast agent
	fastAgent := testing.Fast("/tmp/fast-qga.sock")
	if fastResult := fastAgent.Start(); fastResult.IsOk() {
		defer fastAgent.Stop()
		fmt.Printf("   Fast Agent: %d max connections, %v buffer\n",
			fastAgent.Stats()["max_connections"],
			fastAgent.Stats()["buffer_size"])
	}

	// Simple agent
	simpleAgent := testing.Simple("/tmp/simple-qga.sock")
	if simpleResult := simpleAgent.Start(); simpleResult.IsOk() {
		defer simpleAgent.Stop()
		fmt.Printf("   Simple Agent: %d max connections, %s timeout\n",
			simpleAgent.Stats()["max_connections"],
			simpleAgent.Stats()["timeout"])
	}

	// Fluent configuration
	fluentAgent := testing.Perfect("/tmp/fluent-qga.sock").
		WithConnections(25000).
		WithTimeout(45 * time.Second).
		WithBuffer(32768)

	if fluentResult := fluentAgent.Start(); fluentResult.IsOk() {
		defer fluentAgent.Stop()
		fmt.Printf("   Fluent Agent: %d connections, %v buffer, %s timeout\n",
			fluentAgent.Stats()["max_connections"],
			fluentAgent.Stats()["buffer_size"],
			fluentAgent.Stats()["timeout"])
	}

	fmt.Println("\n🎉 Everything works perfectly and simply!")
	fmt.Println("💡 The Go QGA library is optimistic, robust, and elegant!")
}
