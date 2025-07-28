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

func TestConnection(t *testing.T) {
	// Create test agent
	behavior := qgatesting.DefaultAgentBehavior()
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Test connection
	client, err := qmp.Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test hostname retrieval
	hostname, err := client.GetHostname()
	if err != nil {
		t.Fatalf("Failed to get hostname: %v", err)
	}

	if hostname == "" {
		t.Error("Expected non-empty hostname")
	}
}

func TestConnectionFailure(t *testing.T) {
	// Test connection to non-existent socket
	client, err := qmp.Connect("/tmp/does-not-exist.sock")
	if err == nil {
		t.Error("Expected error when connecting to non-existent socket")
		if client != nil {
			client.Close()
		}
		return
	}

	// Error should be informative
	t.Logf("Expected error: %v", err)
}

func TestConnectionWithSlowAgent(t *testing.T) {
	// Create slow agent
	behavior := qgatesting.SlowAgentBehavior(1000) // 1ms delay
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Test connection still works
	client, err := qmp.Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect to slow agent: %v", err)
	}
	defer client.Close()

	// Test hostname retrieval works despite delay
	hostname, err := client.GetHostname()
	if err != nil {
		t.Fatalf("Failed to get hostname from slow agent: %v", err)
	}

	if hostname != "slow-fake-vm" {
		t.Errorf("Expected 'slow-fake-vm', got '%s'", hostname)
	}
}

func TestConnectionWithErrorAgent(t *testing.T) {
	// Create error agent
	behavior := qgatesting.ErrorAgentBehavior()
	agent, cleanup := qgatesting.SetupAgentWithBehavior(t, behavior)
	defer cleanup()

	// Test connection
	client, err := qmp.Connect(agent.Path())
	if err != nil {
		t.Fatalf("Failed to connect to error agent: %v", err)
	}
	defer client.Close()

	// Test hostname retrieval should fail gracefully
	hostname, err := client.GetHostname()
	if err == nil {
		t.Error("Expected error from error agent")
		return
	}

	// Error should be informative and hostname should be empty
	if hostname != "" {
		t.Errorf("Expected empty hostname on error, got '%s'", hostname)
	}

	t.Logf("Expected error from error agent: %v", err)
}