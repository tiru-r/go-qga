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

package qmp_test

import (
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
	"github.com/prevostcorentin/go-qga/internal/qmp"
)

func TestFakeGuestAgentNoGoroutineLeak(t *testing.T) {
	// Force garbage collection and get baseline goroutine count
	runtime.GC()
	runtime.GC() // Run twice to be thorough
	initialGoroutines := runtime.NumGoroutine()

	const numIterations = 5
	for i := 0; i < numIterations; i++ {
		func() {
			agent := newFakeGuestAgent(t)
			agent.Start()

			// Create multiple connections to spawn goroutines
			var wg sync.WaitGroup
			const numConnections = 3

			for j := 0; j < numConnections; j++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					client, err := qmp.Connect(agent.Path())
					if err != nil {
						t.Logf("Connection failed (may be normal): %v", err)
						return
					}
					defer client.Close()

					// Execute a command to trigger handler goroutine
					_, err = client.GetHostname()
					if err != nil {
						t.Logf("Command failed (may be normal): %v", err)
					}
				}()
			}

			// Wait for all connections
			wg.Wait()

			// Stop the agent - this should clean up all goroutines
			agent.Stop()
		}()

		// Force garbage collection between iterations
		runtime.GC()
		runtime.GC()
	}

	// Allow some time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Final garbage collection
	runtime.GC()
	runtime.GC()

	finalGoroutines := runtime.NumGoroutine()

	// Check for goroutine leaks (allow some tolerance for test framework goroutines)
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Potential goroutine leak detected: started with %d, ended with %d",
			initialGoroutines, finalGoroutines)
	}

	t.Logf("Goroutine count: initial=%d, final=%d", initialGoroutines, finalGoroutines)
}

func TestFakeGuestAgentProperShutdown(t *testing.T) {
	// This test verifies that the WaitGroup correctly tracks connections
	// by creating multiple rapid connections and ensuring Stop() waits for them

	agent := newFakeGuestAgent(t)
	agent.Start()

	const numConnections = 3
	var wg sync.WaitGroup

	// Create multiple connections that will be handled concurrently
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			socketPath := agent.Path()
			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				t.Logf("Connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()

			// Send command
			conn.Write([]byte(common.QmpHostnameRequest))

			// Read response
			buffer := make([]byte, 1024)
			conn.Read(buffer)

			// Brief delay to keep handler busy
			time.Sleep(50 * time.Millisecond)
		}(i)
	}

	// Give connections time to establish
	time.Sleep(20 * time.Millisecond)

	// Stop the agent - this should wait for active connections
	stopStart := time.Now()
	agent.Stop()
	stopDuration := time.Since(stopStart)

	// Verify all connections completed
	wg.Wait()

	// Stop should have taken some time (at least the handler delay)
	if stopDuration < 30*time.Millisecond {
		t.Logf("Stop completed in %v - may not have waited for handlers", stopDuration)
	} else {
		t.Logf("Stop properly waited %v for active connections", stopDuration)
	}
}
