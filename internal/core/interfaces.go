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

package core

import (
	"context"
	"net"

	"github.com/prevostcorentin/go-qga/internal/errors"
)

// Transport defines the interface for transport layer implementations
type Transport interface {
	Connect(ctx context.Context) *errors.TransportError
	Close() error
	Path() string
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, bytes []byte) error
}

// Connection defines the interface for QMP connection management
type Connection interface {
	Connect(ctx context.Context, path string) *errors.ConnectionError
	Send(ctx context.Context, bytes []byte) ([]byte, *errors.ConnectionError)
	SendAsync(ctx context.Context, bytes []byte) <-chan AsyncResult
	Close() error
}

// AsyncResult represents the result of an asynchronous operation
type AsyncResult struct {
	Data []byte
	Err  *errors.ConnectionError
}

// Command defines the interface for QMP commands
type Command interface {
	Execute() string
	Arguments() any
	Response() any
}

// CommandExecutor defines the interface for command execution
type CommandExecutor interface {
	Run(ctx context.Context, command Command) (any, errors.QgaError)
	RunAsync(ctx context.Context, command Command) <-chan ExecutorResult
	Close() error
}

// ExecutorResult represents the result of command execution
type ExecutorResult struct {
	Data any
	Err  errors.QgaError
}

// Agent defines the interface for test agents
type Agent interface {
	Serve(ctx context.Context, handler func(net.Conn)) error
	WaitReady()
}

// Client defines the interface for QMP clients
type Client interface {
	Execute(command string, args ...any) any
	GetHostname() any
	Close()
}

// ConfigurableComponent defines the interface for configurable components
type ConfigurableComponent interface {
	Configure(config any) error
	GetConfig() any
}

// LifecycleComponent defines the interface for components with lifecycle management
type LifecycleComponent interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// MonitorableComponent defines the interface for components that can be monitored
type MonitorableComponent interface {
	GetStats() map[string]any
	GetHealth() HealthStatus
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status  string
	Message string
	Details map[string]any
}

// Factory defines the interface for component factories
type Factory interface {
	CreateTransport(transportType string, path string) (Transport, error)
	CreateConnection(transport Transport) (Connection, error)
	CreateExecutor(connection Connection) (CommandExecutor, error)
	CreateAgent(config any) (Agent, error)
	CreateClient(socketPath string) (Client, error)
}

// Registry defines the interface for component registry
type Registry interface {
	Register(name string, component any) error
	Get(name string) (any, bool)
	List() []string
	Remove(name string) bool
}

// Pool defines the interface for object pools
type Pool[T any] interface {
	Get() T
	Put(item T)
	Size() int
	Drain() []T
}

// Validator defines the interface for input validation
type Validator interface {
	Validate(input any) error
	ValidateContext(ctx context.Context, input any) error
}

// Serializer defines the interface for data serialization
type Serializer interface {
	Marshal(data any) ([]byte, error)
	Unmarshal(data []byte, target any) error
}

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Fatal(msg string, fields ...any)
}

// Metrics defines the interface for metrics collection
type Metrics interface {
	Counter(name string) Counter
	Gauge(name string) Gauge
	Histogram(name string) Histogram
	Timer(name string) Timer
}

// Counter defines the interface for counter metrics
type Counter interface {
	Inc()
	Add(delta int64)
	Get() int64
}

// Gauge defines the interface for gauge metrics
type Gauge interface {
	Set(value float64)
	Add(delta float64)
	Get() float64
}

// Histogram defines the interface for histogram metrics
type Histogram interface {
	Observe(value float64)
	Count() int64
	Sum() float64
}

// Timer defines the interface for timer metrics
type Timer interface {
	Start() TimerSample
	Observe(duration float64)
}

// TimerSample defines the interface for timer samples
type TimerSample interface {
	Stop()
}