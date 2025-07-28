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
	"context"
	"sync"
	"testing"
	"time"
)

func TestConcurrentCommandExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Create a mock connection for testing
	mockConn := &mockConnection{
		responses: make(map[string][]byte),
		delay:     10 * time.Millisecond,
	}
	
	executor, err := NewExecutor(mockConn)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Close()

	ctx := context.Background()
	numWorkers := 10
	numRequestsPerWorker := 5

	var wg sync.WaitGroup
	results := make(chan ExecutorResult, numWorkers*numRequestsPerWorker)

	// Launch concurrent workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numRequestsPerWorker; j++ {
				cmd := &mockCommand{
					execute: "test-command",
					args:    map[string]any{"worker": workerID, "request": j},
				}
				
				resultCh := executor.RunAsync(ctx, cmd)
				result := <-resultCh
				results <- result
			}
		}(i)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var successCount, errorCount int
	for result := range results {
		if result.Err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	expectedTotal := numWorkers * numRequestsPerWorker
	actualTotal := successCount + errorCount

	if actualTotal != expectedTotal {
		t.Errorf("Expected %d total results, got %d", expectedTotal, actualTotal)
	}

	t.Logf("Concurrent test completed: %d successes, %d errors", successCount, errorCount)
}

// Mock connection for testing
type mockConnection struct {
	mu        sync.RWMutex
	responses map[string][]byte
	delay     time.Duration
	closed    bool
}

func (m *mockConnection) Connect(ctx context.Context, path string) error {
	return nil
}

func (m *mockConnection) Send(ctx context.Context, bytes []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil, &mockError{msg: "connection closed"}
	}
	
	// Simulate network delay
	time.Sleep(m.delay)
	
	return []byte(`{"return": "success"}`), nil
}

func (m *mockConnection) SendAsync(ctx context.Context, bytes []byte) <-chan AsyncResult {
	ch := make(chan AsyncResult, 1)
	go func() {
		data, err := m.Send(ctx, bytes)
		ch <- AsyncResult{Data: data, Err: err}
	}()
	return ch
}

func (m *mockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Mock command for testing
type mockCommand struct {
	execute string
	args    map[string]any
}

func (m *mockCommand) Execute() string {
	return m.execute
}

func (m *mockCommand) Arguments() any {
	return m.args
}

func (m *mockCommand) Response() any {
	return &mockResponse{}
}

type mockResponse struct {
	Status string `json:"status"`
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}