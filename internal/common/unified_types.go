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
	"sync"
	"time"
)

// BaseState provides common state management for components
// Fields ordered for optimal memory alignment
type BaseState struct {
	mu     sync.RWMutex // 24 bytes
	closed bool         // 1 byte (+ 7 bytes padding - unavoidable due to struct size)
}

// NewBaseState creates a new base state
func NewBaseState() *BaseState {
	return &BaseState{}
}

// IsClosed returns whether the component is closed (thread-safe)
func (b *BaseState) IsClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.closed
}

// MarkClosed marks the component as closed (thread-safe)
func (b *BaseState) MarkClosed() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
}

// CloseWithLock marks as closed while holding a lock (to avoid deadlock)
func (b *BaseState) CloseWithLock() bool {
	if b.closed {
		return false // already closed
	}
	b.closed = true
	return true // was closed by this call
}

// IsClosedWhileLocked checks if closed while already holding a lock (to avoid deadlock)
func (b *BaseState) IsClosedWhileLocked() bool {
	return b.closed
}

// Lock acquires write lock
func (b *BaseState) Lock() {
	b.mu.Lock()
}

// Unlock releases write lock
func (b *BaseState) Unlock() {
	b.mu.Unlock()
}

// RLock acquires read lock
func (b *BaseState) RLock() {
	b.mu.RLock()
}

// RUnlock releases read lock
func (b *BaseState) RUnlock() {
	b.mu.RUnlock()
}

// BaseComponent provides common functionality for all components
type BaseComponent struct {
	*BaseState
	path     string
	metadata map[string]any
}

// NewBaseComponent creates a new base component
func NewBaseComponent(path string) *BaseComponent {
	return &BaseComponent{
		BaseState: NewBaseState(),
		path:      path,
		metadata:  make(map[string]any),
	}
}

// Path returns the component path
func (b *BaseComponent) Path() string {
	b.RLock()
	defer b.RUnlock()
	return b.path
}

// SetPath sets the component path
func (b *BaseComponent) SetPath(path string) {
	b.Lock()
	defer b.Unlock()
	b.path = path
}

// SetMetadata sets metadata for the component
func (b *BaseComponent) SetMetadata(key string, value any) {
	b.Lock()
	defer b.Unlock()
	b.metadata[key] = value
}

// GetMetadata gets metadata from the component
func (b *BaseComponent) GetMetadata(key string) (any, bool) {
	b.RLock()
	defer b.RUnlock()
	value, exists := b.metadata[key]
	return value, exists
}

// UnifiedResult is now an alias to Result[T] for compatibility
type UnifiedResult[T any] = Result[T]

// NewSuccess creates a successful result (compatibility)
func NewSuccess[T any](value T) Result[T] {
	return Success(value)
}

// NewFailure creates a failed result (compatibility)
func NewFailure[T any](err error) Result[T] {
	return FailureFrom[T](err)
}

// UnifiedConfig consolidates all configuration types
type UnifiedConfig struct {
	Path           string
	BufferSize     int
	MaxConnections int
	Timeout        time.Duration
	Properties     map[string]any
}

// NewUnifiedConfig creates a new unified configuration
func NewUnifiedConfig(path string) *UnifiedConfig {
	return &UnifiedConfig{
		Path:           path,
		BufferSize:     StandardBufferSize,
		MaxConnections: DefaultMaxConnections,
		Timeout:        DefaultTimeout,
		Properties:     make(map[string]any),
	}
}

// WithBufferSize sets the buffer size
func (c *UnifiedConfig) WithBufferSize(size int) *UnifiedConfig {
	c.BufferSize = size
	return c
}

// WithMaxConnections sets the maximum connections
func (c *UnifiedConfig) WithMaxConnections(max int) *UnifiedConfig {
	c.MaxConnections = max
	return c
}

// WithTimeout sets the timeout
func (c *UnifiedConfig) WithTimeout(timeout time.Duration) *UnifiedConfig {
	c.Timeout = timeout
	return c
}

// SetProperty sets a custom property
func (c *UnifiedConfig) SetProperty(key string, value any) *UnifiedConfig {
	c.Properties[key] = value
	return c
}

// GetProperty gets a custom property
func (c *UnifiedConfig) GetProperty(key string) (any, bool) {
	value, exists := c.Properties[key]
	return value, exists
}

// UnifiedStats consolidates all statistics types
// Fields ordered for optimal memory alignment
type UnifiedStats struct {
	Timestamp           time.Time     // 24 bytes - largest first
	CustomMetrics       map[string]any // 8 bytes (pointer)
	ConnectionsActive   int64         // 8 bytes
	ConnectionsTotal    int64         // 8 bytes  
	CommandsExecuted    int64         // 8 bytes
	BytesTransferred    int64         // 8 bytes
	ErrorsTotal         int64         // 8 bytes
	MemoryAllocated     int64         // 8 bytes
	GoroutinesActive    int64         // 8 bytes
	AverageLatency      time.Duration // 8 bytes
}

// NewUnifiedStats creates new unified statistics
func NewUnifiedStats() *UnifiedStats {
	return &UnifiedStats{
		Timestamp:     time.Now(),
		CustomMetrics: make(map[string]any),
	}
}

// SetCustomMetric sets a custom metric
func (s *UnifiedStats) SetCustomMetric(key string, value any) {
	s.CustomMetrics[key] = value
}

// GetCustomMetric gets a custom metric
func (s *UnifiedStats) GetCustomMetric(key string) (any, bool) {
	value, exists := s.CustomMetrics[key]
	return value, exists
}

// UnifiedConstructor provides a single constructor pattern
type UnifiedConstructor[T any] struct {
	createFunc  func(*UnifiedConfig) (T, error)
	config      *UnifiedConfig
	middleware  []func(T) T
}

// NewUnifiedConstructor creates a new unified constructor
func NewUnifiedConstructor[T any](createFunc func(*UnifiedConfig) (T, error)) *UnifiedConstructor[T] {
	return &UnifiedConstructor[T]{
		createFunc: createFunc,
		config:     NewUnifiedConfig(""),
		middleware: make([]func(T) T, 0),
	}
}

// WithConfig sets the configuration
func (c *UnifiedConstructor[T]) WithConfig(config *UnifiedConfig) *UnifiedConstructor[T] {
	c.config = config
	return c
}

// WithMiddleware adds middleware
func (c *UnifiedConstructor[T]) WithMiddleware(middleware func(T) T) *UnifiedConstructor[T] {
	c.middleware = append(c.middleware, middleware)
	return c
}

// Build constructs the component
func (c *UnifiedConstructor[T]) Build() UnifiedResult[T] {
	component, err := c.createFunc(c.config)
	if err != nil {
		return NewFailure[T](err)
	}
	
	// Apply middleware
	for _, middleware := range c.middleware {
		component = middleware(component)
	}
	
	return NewSuccess(component)
}

// UnifiedContext consolidates context management
type UnifiedContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	values map[string]any
	mu     sync.RWMutex
}

// NewUnifiedContext creates a new unified context
func NewUnifiedContext() *UnifiedContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &UnifiedContext{
		ctx:    ctx,
		cancel: cancel,
		values: make(map[string]any),
	}
}

// WithTimeout creates a context with timeout
func (c *UnifiedContext) WithTimeout(timeout time.Duration) *UnifiedContext {
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	return &UnifiedContext{
		ctx:    ctx,
		cancel: cancel,
		values: c.copyValues(),
	}
}

// WithDeadline creates a context with deadline
func (c *UnifiedContext) WithDeadline(deadline time.Time) *UnifiedContext {
	ctx, cancel := context.WithDeadline(c.ctx, deadline)
	return &UnifiedContext{
		ctx:    ctx,
		cancel: cancel,
		values: c.copyValues(),
	}
}

// Context returns the underlying context
func (c *UnifiedContext) Context() context.Context {
	return c.ctx
}

// Cancel cancels the context
func (c *UnifiedContext) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// SetValue sets a context value
func (c *UnifiedContext) SetValue(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// GetValue gets a context value
func (c *UnifiedContext) GetValue(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.values[key]
	return value, exists
}

// copyValues creates a copy of the values map
func (c *UnifiedContext) copyValues() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	copy := make(map[string]any, len(c.values))
	for k, v := range c.values {
		copy[k] = v
	}
	return copy
}

// UnifiedResource provides resource management
type UnifiedResource struct {
	*BaseComponent
	resources []func() error
	mu        sync.Mutex
}

// NewUnifiedResource creates a new unified resource
func NewUnifiedResource(path string) *UnifiedResource {
	return &UnifiedResource{
		BaseComponent: NewBaseComponent(path),
		resources:     make([]func() error, 0),
	}
}

// AddResource adds a cleanup resource
func (r *UnifiedResource) AddResource(cleanup func() error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resources = append(r.resources, cleanup)
}

// Close closes all resources
func (r *UnifiedResource) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.IsClosed() {
		return nil
	}
	
	var firstError error
	
	// Close resources in reverse order
	for i := len(r.resources) - 1; i >= 0; i-- {
		if err := r.resources[i](); err != nil && firstError == nil {
			firstError = err
		}
	}
	
	r.MarkClosed()
	return firstError
}