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
	"testing"

	"github.com/prevostcorentin/go-qga/internal/qmp"
	qgatesting "github.com/prevostcorentin/go-qga/internal/testing"
)

func TestExecutor(t *testing.T) {
	// Create test agent
	behavior := qgatesting.DefaultAgentBehavior()
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Test executor
	client, err := qmp.Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test command execution
	hostname, err := client.GetHostname()
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	if hostname != "fake-vm" {
		t.Errorf("Expected 'fake-vm', got '%s'", hostname)
	}
}

func TestExecutorWithSlowCommand(t *testing.T) {
	// Create slow agent
	behavior := qgatesting.SlowAgentBehavior(1000) // 1ms delay
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Test executor with slow commands
	client, err := qmp.Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test slow command execution
	hostname, err := client.GetHostname()
	if err != nil {
		t.Fatalf("Failed to execute slow command: %v", err)
	}

	if hostname != "slow-fake-vm" {
		t.Errorf("Expected 'slow-fake-vm', got '%s'", hostname)
	}
}

func TestExecutorWithErrorCommand(t *testing.T) {
	// Create error agent
	behavior := qgatesting.ErrorAgentBehavior()
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Test executor with error commands
	client, err := qmp.Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test error command execution
	_, err = client.GetHostname()
	if err == nil {
		t.Error("Expected error from error command")
		return
	}

	// Error should be informative
	t.Logf("Expected error from error command: %v", err)
}