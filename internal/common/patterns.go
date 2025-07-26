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
	"io"
	"net"
	"sync"
	"time"
)

// ConnectionHandler provides common connection management patterns
type ConnectionHandler struct {
	timeout time.Duration
}

// NewConnectionHandler creates a new connection handler with default timeout
func NewConnectionHandler() *ConnectionHandler {
	return &ConnectionHandler{timeout: DefaultTimeout}
}

// WithTimeout sets a custom timeout for operations
func (h *ConnectionHandler) WithTimeout(timeout time.Duration) *ConnectionHandler {
	h.timeout = timeout
	return h
}

// HandleConnection executes a handler function with proper connection cleanup
func (h *ConnectionHandler) HandleConnection(conn net.Conn, handler func(net.Conn) error) error {
	defer conn.Close()
	
	if err := GlobalTimeManager.SetDeadline(conn, h.timeout); err != nil {
		return err
	}
	
	return handler(conn)
}

// ChannelFactory provides common channel creation patterns with pooling
type ChannelFactory struct {
	errorPool     *sync.Pool
	donePool      *sync.Pool
	resultPool    *sync.Pool
	intPool       *sync.Pool
	byteSlicePool *sync.Pool
	stringPool    *sync.Pool
	boolPool      *sync.Pool
}

// NewChannelFactory creates a new channel factory with pools
func NewChannelFactory() *ChannelFactory {
	return &ChannelFactory{
		errorPool: &sync.Pool{
			New: func() any { return make(chan error, 1) },
		},
		donePool: &sync.Pool{
			New: func() any { return make(chan struct{}, 1) },
		},
		resultPool: &sync.Pool{
			New: func() any { return make(chan any, 1) },
		},
		intPool: &sync.Pool{
			New: func() any { return make(chan int, 1) },
		},
		byteSlicePool: &sync.Pool{
			New: func() any { return make(chan []byte, 1) },
		},
		stringPool: &sync.Pool{
			New: func() any { return make(chan string, 1) },
		},
		boolPool: &sync.Pool{
			New: func() any { return make(chan bool, 1) },
		},
	}
}

// ErrorChannel creates a buffered error channel from pool
func (f *ChannelFactory) ErrorChannel() chan error {
	ch := f.errorPool.Get().(chan error)
	// Drain any stale data
	select {
	case <-ch:
	default:
	}
	return ch
}

// PutErrorChannel returns an error channel to the pool
func (f *ChannelFactory) PutErrorChannel(ch chan error) {
	f.errorPool.Put(ch)
}

// DoneChannel creates a buffered done channel from pool
func (f *ChannelFactory) DoneChannel() chan struct{} {
	ch := f.donePool.Get().(chan struct{})
	// Drain any stale data
	select {
	case <-ch:
	default:
	}
	return ch
}

// PutDoneChannel returns a done channel to the pool
func (f *ChannelFactory) PutDoneChannel(ch chan struct{}) {
	f.donePool.Put(ch)
}

// ResultChannel creates a buffered result channel from pool
func (f *ChannelFactory) ResultChannel() chan any {
	ch := f.resultPool.Get().(chan any)
	// Drain any stale data
	select {
	case <-ch:
	default:
	}
	return ch
}

// PutResultChannel returns a result channel to the pool
func (f *ChannelFactory) PutResultChannel(ch chan any) {
	f.resultPool.Put(ch)
}

// IntChannel creates a buffered int channel
func (f *ChannelFactory) IntChannel() chan int {
	return make(chan int, 1)
}

// ByteSliceChannel creates a buffered byte slice channel
func (f *ChannelFactory) ByteSliceChannel() chan []byte {
	return make(chan []byte, 1)
}

// StringChannel creates a buffered string channel
func (f *ChannelFactory) StringChannel() chan string {
	return make(chan string, 1)
}

// BoolChannel creates a buffered bool channel
func (f *ChannelFactory) BoolChannel() chan bool {
	return make(chan bool, 1)
}

// ContextManager provides common context management patterns
type ContextManager struct{}

// NewContextManager creates a new context manager
func NewContextManager() *ContextManager {
	return &ContextManager{}
}

// WithCancel creates a context with cancellation
func (m *ContextManager) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// WithTimeout creates a context with timeout
func (m *ContextManager) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// WithDeadline creates a context with deadline
func (m *ContextManager) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), deadline)
}

// ResourceManager provides common resource management patterns
type ResourceManager struct {
	resources []io.Closer
}

// NewResourceManager creates a new resource manager
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		resources: make([]io.Closer, 0),
	}
}

// Add registers a resource for cleanup
func (rm *ResourceManager) Add(resource io.Closer) {
	rm.resources = append(rm.resources, resource)
}

// Cleanup closes all registered resources
func (rm *ResourceManager) Cleanup() error {
	var firstErr error
	
	for i := len(rm.resources) - 1; i >= 0; i-- {
		if err := rm.resources[i].Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	
	rm.resources = rm.resources[:0]
	return firstErr
}

// BufferPool provides common buffer management patterns
type BufferPool struct {
	small    chan []byte
	standard chan []byte
	large    chan []byte
	xlarge   chan []byte
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(poolSize int) *BufferPool {
	return &BufferPool{
		small:    make(chan []byte, poolSize),
		standard: make(chan []byte, poolSize),
		large:    make(chan []byte, poolSize),
		xlarge:   make(chan []byte, poolSize),
	}
}

// GetSmall gets a small buffer (1KB)
func (bp *BufferPool) GetSmall() []byte {
	select {
	case buf := <-bp.small:
		return buf
	default:
		return make([]byte, SmallBufferSize)
	}
}

// GetStandard gets a standard buffer (4KB)
func (bp *BufferPool) GetStandard() []byte {
	select {
	case buf := <-bp.standard:
		return buf
	default:
		return make([]byte, StandardBufferSize)
	}
}

// GetLarge gets a large buffer (8KB)
func (bp *BufferPool) GetLarge() []byte {
	select {
	case buf := <-bp.large:
		return buf
	default:
		return make([]byte, LargeBufferSize)
	}
}

// GetXLarge gets an extra large buffer (16KB)
func (bp *BufferPool) GetXLarge() []byte {
	select {
	case buf := <-bp.xlarge:
		return buf
	default:
		return make([]byte, XLargeBufferSize)
	}
}

// PutSmall returns a small buffer to the pool
func (bp *BufferPool) PutSmall(buf []byte) {
	if cap(buf) == SmallBufferSize {
		buf = buf[:SmallBufferSize]
		select {
		case bp.small <- buf:
		default:
		}
	}
}

// PutStandard returns a standard buffer to the pool
func (bp *BufferPool) PutStandard(buf []byte) {
	if cap(buf) == StandardBufferSize {
		buf = buf[:StandardBufferSize]
		select {
		case bp.standard <- buf:
		default:
		}
	}
}

// PutLarge returns a large buffer to the pool
func (bp *BufferPool) PutLarge(buf []byte) {
	if cap(buf) == LargeBufferSize {
		buf = buf[:LargeBufferSize]
		select {
		case bp.large <- buf:
		default:
		}
	}
}

// PutXLarge returns an extra large buffer to the pool
func (bp *BufferPool) PutXLarge(buf []byte) {
	if cap(buf) == XLargeBufferSize {
		buf = buf[:XLargeBufferSize]
		select {
		case bp.xlarge <- buf:
		default:
		}
	}
}

// Global instances for convenience
var (
	GlobalChannelFactory   = NewChannelFactory()
	GlobalContextManager   = NewContextManager()
	GlobalResourceManager  = NewResourceManager()
	GlobalBufferPool       = NewBufferPool(100)
)

// Convenience functions for channel creation
func ErrorChannel() chan error {
	return GlobalChannelFactory.ErrorChannel()
}

func DoneChannel() chan struct{} {
	return GlobalChannelFactory.DoneChannel()
}

func StringChannel() chan string {
	return GlobalChannelFactory.StringChannel()
}

func BoolChannel() chan bool {
	return GlobalChannelFactory.BoolChannel()
}

// Convenience functions for returning channels to pools
func PutErrorChannel(ch chan error) {
	GlobalChannelFactory.PutErrorChannel(ch)
}

func PutDoneChannel(ch chan struct{}) {
	GlobalChannelFactory.PutDoneChannel(ch)
}

func PutResultChannel(ch chan any) {
	GlobalChannelFactory.PutResultChannel(ch)
}