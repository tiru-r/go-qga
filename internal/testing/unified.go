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
	"path/filepath"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// UnifiedAgent provides a single interface for all agent types
type UnifiedAgent struct {
	agentType AgentType
	config    *UnifiedTestConfig
	agent     Agent
}

// AgentType represents different agent implementations
type AgentType string

const (
	SocketAgentType AgentType = "socket"
	PerfectAgentType         = "perfect"
	SimpleAgentType          = "simple"
)

// UnifiedConfig provides configuration for all agent types
type UnifiedTestConfig struct {
	SocketPath     string
	AgentType      AgentType
	BufferSize     int
	MaxConnections int
	Timeout        time.Duration
	Performance    PerformanceLevel
}

// PerformanceLevel represents different performance optimization levels
type PerformanceLevel string

const (
	TestPerformance       PerformanceLevel = "test"
	StandardPerformance                    = "standard"
	HighPerformance                        = "high"
	UltraHighPerformance                   = "ultra"
)

// NewUnifiedAgent creates a new unified agent
func NewUnifiedAgent(config *UnifiedTestConfig) (*UnifiedAgent, error) {
	if config == nil {
		config = DefaultUnifiedConfig()
	}
	
	// Apply performance-based defaults
	applyPerformanceDefaults(config)
	
	var agent Agent
	
	switch config.AgentType {
	case SocketAgentType:
		socketConfig := &SocketAgentConfig{
			SocketPath:     config.SocketPath,
			BufferSize:     config.BufferSize,
			MaxConnections: config.MaxConnections,
			ReadTimeout:    config.Timeout,
		}
		agent = NewSocketAgentWithConfig(socketConfig)
		
	case PerfectAgentType:
		// PerfectAgent doesn't implement Agent interface, use SocketAgent instead
		socketConfig := &SocketAgentConfig{
			SocketPath:     config.SocketPath,
			BufferSize:     config.BufferSize,
			MaxConnections: config.MaxConnections,
			ReadTimeout:    config.Timeout,
		}
		agent = NewSocketAgentWithConfig(socketConfig)
		
	case SimpleAgentType:
		// SimpleAgent doesn't implement Agent interface, use SocketAgent instead  
		socketConfig := &SocketAgentConfig{
			SocketPath:     config.SocketPath,
			BufferSize:     config.BufferSize,
			MaxConnections: config.MaxConnections,
			ReadTimeout:    config.Timeout,
		}
		agent = NewSocketAgentWithConfig(socketConfig)
		
	default:
		return nil, &ValidationError{Message: "unsupported agent type: " + string(config.AgentType)}
	}
	
	return &UnifiedAgent{
		agentType: config.AgentType,
		config:    config,
		agent:     agent,
	}, nil
}

// DefaultUnifiedConfig creates a default unified configuration
func DefaultUnifiedConfig() *UnifiedTestConfig {
	return &UnifiedTestConfig{
		AgentType:      SocketAgentType,
		BufferSize:     common.StandardBufferSize,
		MaxConnections: common.DefaultMaxConnections,
		Timeout:        common.DefaultTimeout,
		Performance:    StandardPerformance,
	}
}

// applyPerformanceDefaults applies performance-based configuration defaults
func applyPerformanceDefaults(config *UnifiedTestConfig) {
	switch config.Performance {
	case TestPerformance:
		if config.BufferSize == 0 {
			config.BufferSize = common.SmallBufferSize
		}
		if config.MaxConnections == 0 {
			config.MaxConnections = common.TestMaxConnections
		}
		if config.Timeout == 0 {
			config.Timeout = common.FastTimeout
		}
		
	case StandardPerformance:
		if config.BufferSize == 0 {
			config.BufferSize = common.StandardBufferSize
		}
		if config.MaxConnections == 0 {
			config.MaxConnections = common.DefaultMaxConnections
		}
		if config.Timeout == 0 {
			config.Timeout = common.DefaultTimeout
		}
		
	case HighPerformance:
		if config.BufferSize == 0 {
			config.BufferSize = common.LargeBufferSize
		}
		if config.MaxConnections == 0 {
			config.MaxConnections = common.HighPerfMaxConnections
		}
		if config.Timeout == 0 {
			config.Timeout = common.DefaultTimeout
		}
		
	case UltraHighPerformance:
		if config.BufferSize == 0 {
			config.BufferSize = common.XLargeBufferSize
		}
		if config.MaxConnections == 0 {
			config.MaxConnections = common.HighPerfMaxConnections * 2
		}
		if config.Timeout == 0 {
			config.Timeout = common.FastTimeout
		}
	}
}

// Serve implements the Agent interface
func (u *UnifiedAgent) Serve(ctx context.Context, handler func(net.Conn)) error {
	return u.agent.Serve(ctx, handler)
}

// WaitReady implements the Agent interface
func (u *UnifiedAgent) WaitReady() {
	u.agent.WaitReady()
}

// GetConfig returns the agent configuration
func (u *UnifiedAgent) GetConfig() *UnifiedTestConfig {
	return u.config
}

// GetType returns the agent type
func (u *UnifiedAgent) GetType() AgentType {
	return u.agentType
}

// UnifiedTestSuite provides a standardized testing framework
type UnifiedTestSuite struct {
	factory    *common.ComponentFactory
	agents     map[string]*UnifiedAgent
	cleanup    []func()
	tempDir    string
}

// NewUnifiedTestSuite creates a new unified test suite
func NewUnifiedTestSuite() *UnifiedTestSuite {
	return &UnifiedTestSuite{
		factory: common.NewComponentFactory(),
		agents:  make(map[string]*UnifiedAgent),
		cleanup: make([]func(), 0),
	}
}

// CreateAgent creates and registers a new agent
func (s *UnifiedTestSuite) CreateAgent(name string, config *UnifiedTestConfig) (*UnifiedAgent, error) {
	if config.SocketPath == "" {
		config.SocketPath = s.generateSocketPath(name)
	}
	
	agent, err := NewUnifiedAgent(config)
	if err != nil {
		return nil, err
	}
	
	s.agents[name] = agent
	s.cleanup = append(s.cleanup, func() {
		// All agent types use SocketAgent, no specific cleanup needed
		// The Agent interface doesn't provide a Stop method
	})
	
	return agent, nil
}

// GetAgent retrieves an agent by name
func (s *UnifiedTestSuite) GetAgent(name string) (*UnifiedAgent, bool) {
	agent, exists := s.agents[name]
	return agent, exists
}

// CreateTestAgent creates an agent optimized for testing
func (s *UnifiedTestSuite) CreateTestAgent(name string) (*UnifiedAgent, error) {
	config := DefaultUnifiedConfig()
	config.Performance = TestPerformance
	config.AgentType = SocketAgentType
	return s.CreateAgent(name, config)
}

// CreateHighPerfAgent creates an agent optimized for high performance
func (s *UnifiedTestSuite) CreateHighPerfAgent(name string) (*UnifiedAgent, error) {
	config := DefaultUnifiedConfig()
	config.Performance = HighPerformance
	config.AgentType = PerfectAgentType
	return s.CreateAgent(name, config)
}

// Cleanup cleans up all resources
func (s *UnifiedTestSuite) Cleanup() {
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
	s.cleanup = s.cleanup[:0]
	s.agents = make(map[string]*UnifiedAgent)
}

// generateSocketPath generates a unique socket path for testing
func (s *UnifiedTestSuite) generateSocketPath(name string) string {
	if s.tempDir == "" {
		s.tempDir = "/tmp"
	}
	return filepath.Join(s.tempDir, "qga-test-"+name+".sock")
}

// TestingHelper provides common testing utilities
type TestingHelper struct {
	suite *UnifiedTestSuite
}

// NewTestingHelper creates a new testing helper
func NewTestingHelper() *TestingHelper {
	return &TestingHelper{
		suite: NewUnifiedTestSuite(),
	}
}

// WithAgent configures a test with a specific agent
func (h *TestingHelper) WithAgent(name string, agentType AgentType, performance PerformanceLevel) *TestingHelper {
	config := DefaultUnifiedConfig()
	config.AgentType = agentType
	config.Performance = performance
	h.suite.CreateAgent(name, config)
	return h
}

// WithTestAgent configures a test with a test-optimized agent
func (h *TestingHelper) WithTestAgent(name string) *TestingHelper {
	h.suite.CreateTestAgent(name)
	return h
}

// WithHighPerfAgent configures a test with a high-performance agent
func (h *TestingHelper) WithHighPerfAgent(name string) *TestingHelper {
	h.suite.CreateHighPerfAgent(name)
	return h
}

// Run executes a test function with proper setup and cleanup
func (h *TestingHelper) Run(testFunc func(*UnifiedTestSuite)) {
	defer h.suite.Cleanup()
	testFunc(h.suite)
}

// Suite returns the underlying test suite
func (h *TestingHelper) Suite() *UnifiedTestSuite {
	return h.suite
}