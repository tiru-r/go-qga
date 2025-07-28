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
	"sync"
	"testing"

	qgatesting "github.com/prevostcorentin/go-qga/internal/testing"
)

func TestConcurrentClientConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Create test agent
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, qgatesting.DefaultAgentBehavior())
	defer cleanup()

	socketPath := qgatesting.GetSocketPath(agent)
	numClients := 10

	var wg sync.WaitGroup
	results := make(chan string, numClients)
	errors := make(chan error, numClients)

	// Launch concurrent clients
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client, err := Connect(socketPath)
			if err != nil {
				errors <- err
				return
			}
			defer client.Close()

			hostname, err := client.GetHostname()
			if err != nil {
				errors <- err
				return
			}

			results <- hostname
		}(i)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	var successCount, errorCount int
	var errorList []error
	
	// Collect all results first
	for results != nil || errors != nil {
		select {
		case result, ok := <-results:
			if !ok {
				results = nil
				continue
			}
			if result == "fake-vm" {
				successCount++
			} else {
				t.Errorf("Unexpected hostname: %s", result)
			}
		case err, ok := <-errors:
			if !ok {
				errors = nil
				continue
			}
			errorCount++
			errorList = append(errorList, err)
		}
	}

	if errorCount > 0 {
		t.Errorf("Got %d errors:", errorCount)
		for _, err := range errorList {
			t.Logf("  %v", err)
		}
	}

	if successCount+errorCount != numClients {
		t.Errorf("Expected %d total results, got %d", numClients, successCount+errorCount)
	}

	t.Logf("Concurrent client test completed: %d successes, %d errors", successCount, errorCount)
}