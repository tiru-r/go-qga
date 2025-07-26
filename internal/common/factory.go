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

package common

import (
	"context"
	"time"
)

// ComponentFactory provides unified factory methods for all components
type ComponentFactory struct {
	contextManager    *ContextManager
	connectionHandler *ConnectionHandler
	channelFactory    *ChannelFactory
	bufferPool        *BufferPool
}

// NewComponentFactory creates a new component factory with all shared resources
func NewComponentFactory() *ComponentFactory {
	return &ComponentFactory{
		contextManager:    NewContextManager(),
		connectionHandler: NewConnectionHandler(),
		channelFactory:    NewChannelFactory(),
		bufferPool:        NewBufferPool(10), // Default pool size
	}
}

// ContextManager returns the shared context manager
func (f *ComponentFactory) ContextManager() *ContextManager {
	return f.contextManager
}

// ConnectionHandler returns the shared connection handler
func (f *ComponentFactory) ConnectionHandler() *ConnectionHandler {
	return f.connectionHandler
}

// ChannelFactory returns the shared channel factory
func (f *ComponentFactory) ChannelFactory() *ChannelFactory {
	return f.channelFactory
}

// BufferPool returns the shared buffer pool
func (f *ComponentFactory) BufferPool() *BufferPool {
	return f.bufferPool
}

// CreateStandardContext creates a context with standard timeout
func (f *ComponentFactory) CreateStandardContext() (context.Context, context.CancelFunc) {
	return f.contextManager.WithTimeout(DefaultTimeout)
}

// CreateFastContext creates a context with fast timeout
func (f *ComponentFactory) CreateFastContext() (context.Context, context.CancelFunc) {
	return f.contextManager.WithTimeout(FastTimeout)
}

// CreateSlowContext creates a context with slow timeout
func (f *ComponentFactory) CreateSlowContext() (context.Context, context.CancelFunc) {
	return f.contextManager.WithTimeout(SlowTimeout)
}

// CreateCustomContext creates a context with custom timeout
func (f *ComponentFactory) CreateCustomContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return f.contextManager.WithTimeout(timeout)
}

// ConfigBuilder provides a builder pattern for component configuration
type ConfigBuilder struct {
	socketPath     string
	bufferSize     int
	maxConnections int
	timeout        time.Duration
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		bufferSize:     StandardBufferSize,
		maxConnections: DefaultMaxConnections,
		timeout:        DefaultTimeout,
	}
}

// WithSocketPath sets the socket path
func (b *ConfigBuilder) WithSocketPath(path string) *ConfigBuilder {
	b.socketPath = path
	return b
}

// WithBufferSize sets the buffer size
func (b *ConfigBuilder) WithBufferSize(size int) *ConfigBuilder {
	b.bufferSize = size
	return b
}

// WithMaxConnections sets the maximum connections
func (b *ConfigBuilder) WithMaxConnections(max int) *ConfigBuilder {
	b.maxConnections = max
	return b
}

// WithTimeout sets the timeout
func (b *ConfigBuilder) WithTimeout(timeout time.Duration) *ConfigBuilder {
	b.timeout = timeout
	return b
}

// BuildForTesting creates a configuration optimized for testing
func (b *ConfigBuilder) BuildForTesting() *ConfigBuilder {
	return b.
		WithBufferSize(SmallBufferSize).
		WithMaxConnections(TestMaxConnections).
		WithTimeout(FastTimeout)
}

// BuildForProduction creates a configuration optimized for production
func (b *ConfigBuilder) BuildForProduction() *ConfigBuilder {
	return b.
		WithBufferSize(LargeBufferSize).
		WithMaxConnections(DefaultMaxConnections).
		WithTimeout(DefaultTimeout)
}

// BuildForHighPerformance creates a configuration optimized for high performance
func (b *ConfigBuilder) BuildForHighPerformance() *ConfigBuilder {
	return b.
		WithBufferSize(XLargeBufferSize).
		WithMaxConnections(HighPerfMaxConnections).
		WithTimeout(FastTimeout)
}

// GetSocketPath returns the socket path
func (b *ConfigBuilder) GetSocketPath() string {
	return b.socketPath
}

// GetBufferSize returns the buffer size
func (b *ConfigBuilder) GetBufferSize() int {
	return b.bufferSize
}

// GetMaxConnections returns the maximum connections
func (b *ConfigBuilder) GetMaxConnections() int {
	return b.maxConnections
}

// GetTimeout returns the timeout
func (b *ConfigBuilder) GetTimeout() time.Duration {
	return b.timeout
}