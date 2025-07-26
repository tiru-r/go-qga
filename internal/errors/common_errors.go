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

package errors

import (
	"fmt"
	"strings"
	"sync"
)

// Common error messages to eliminate duplication
const (
	ConnectionClosedMessage = "connection is closed"
	TransportClosedMessage  = "transport is closed" 
	ExecutorClosedMessage   = "executor is closed"
	CommandNilMessage       = "command cannot be nil"
	TransportNilMessage     = "transport cannot be nil"
	ConnectionNilMessage    = "connection cannot be nil"
	ComponentNilMessage     = "component cannot be nil"
	ComponentNameEmptyMessage = "component name cannot be empty"
	ConfigNilMessage        = "config cannot be nil"
	SocketPathEmptyMessage  = "socket path cannot be empty"
)

// Predefined common errors to avoid repeated error creation
var (
	// Connection errors
	ErrConnectionClosed = NewConnectionError(fmt.Errorf(ConnectionClosedMessage), SendErrorKind)
	ErrConnectionNil    = NewConnectionError(fmt.Errorf(ConnectionNilMessage), ConnectErrorKind)
	
	// Transport errors  
	ErrTransportClosed = NewTransportError(fmt.Errorf(TransportClosedMessage), Write)
	ErrTransportNil    = NewTransportError(fmt.Errorf(TransportNilMessage), Connect)
	
	// Codec errors
	ErrCommandNil    = NewCodecError(fmt.Errorf(CommandNilMessage), Type)
	ErrExecutorClosed = NewCodecError(fmt.Errorf(ExecutorClosedMessage), Type)
	ErrMissingReturn = NewCodecError(fmt.Errorf(`missing "return" field in QGA response`), Key)
)

// ErrorFactory provides centralized error creation
type ErrorFactory struct{}

// NewErrorFactory creates a new error factory
func NewErrorFactory() *ErrorFactory {
	return &ErrorFactory{}
}

// ConnectionClosed creates a connection closed error
func (f *ErrorFactory) ConnectionClosed() *ConnectionError {
	return ErrConnectionClosed
}

// ConnectionError creates a connection error with custom message
func (f *ErrorFactory) ConnectionError(message string, kind ConnectionErrorKind) *ConnectionError {
	return NewConnectionError(fmt.Errorf("%s", message), kind)
}

// TransportClosed creates a transport closed error
func (f *ErrorFactory) TransportClosed() *TransportError {
	return ErrTransportClosed
}

// TransportError creates a transport error with custom message
func (f *ErrorFactory) TransportError(message string, kind TransportErrorKind) *TransportError {
	return NewTransportError(fmt.Errorf("%s", message), kind)
}

// CodecError creates a codec error with custom message
func (f *ErrorFactory) CodecError(message string, kind CodecErrorKind) *CodecError {
	return NewCodecError(fmt.Errorf("%s", message), kind)
}

// CommandNil creates a command nil error
func (f *ErrorFactory) CommandNil() *CodecError {
	return ErrCommandNil
}

// ExecutorClosed creates an executor closed error
func (f *ErrorFactory) ExecutorClosed() *CodecError {
	return ErrExecutorClosed
}

// MissingReturn creates a missing return field error
func (f *ErrorFactory) MissingReturn() *CodecError {
	return ErrMissingReturn
}

// WrapError wraps an existing error with QGA error type
func (f *ErrorFactory) WrapError(err error, domain DomainType) QgaError {
	switch domain {
	case ConnectionDomain:
		return NewConnectionError(err, UnknownErrorKind)
	case TransportDomain:
		return NewTransportError(err, NotConnected)
	case CodecDomain:
		return NewCodecError(err, Type)
	default:
		return NewCodecError(err, Type)
	}
}

// ErrorChain provides error chaining functionality
type ErrorChain struct {
	errors []QgaError
}

// NewErrorChain creates a new error chain
func NewErrorChain() *ErrorChain {
	return &ErrorChain{
		errors: make([]QgaError, 0),
	}
}

// Add adds an error to the chain
func (c *ErrorChain) Add(err QgaError) *ErrorChain {
	if err != nil {
		c.errors = append(c.errors, err)
	}
	return c
}

// HasErrors returns true if the chain contains errors
func (c *ErrorChain) HasErrors() bool {
	return len(c.errors) > 0
}

// First returns the first error in the chain
func (c *ErrorChain) First() QgaError {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[0]
}

// Last returns the last error in the chain
func (c *ErrorChain) Last() QgaError {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[len(c.errors)-1]
}

// All returns all errors in the chain
func (c *ErrorChain) All() []QgaError {
	result := make([]QgaError, len(c.errors))
	copy(result, c.errors)
	return result
}

// Error implements the error interface
func (c *ErrorChain) Error() string {
	if len(c.errors) == 0 {
		return "no errors"
	}
	
	if len(c.errors) == 1 {
		return c.errors[0].Error()
	}
	
	// Use string builder for efficient concatenation
	errCount := len(c.errors)
	lastIdx := errCount - 1
	
	var builder strings.Builder
	builder.WriteString("multiple errors: first=")
	builder.WriteString(c.errors[0].Error())
	builder.WriteString(", last=")
	builder.WriteString(c.errors[lastIdx].Error())
	builder.WriteString(", total=")
	builder.WriteString(fmt.Sprintf("%d", errCount))
	return builder.String()
}

// ErrorCollector provides thread-safe error collection
type ErrorCollector struct {
	chain *ErrorChain
	mu    sync.RWMutex
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		chain: NewErrorChain(),
	}
}

// Add adds an error thread-safely
func (c *ErrorCollector) Add(err QgaError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chain.Add(err)
}

// HasErrors returns true if there are collected errors
func (c *ErrorCollector) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.chain.HasErrors()
}

// GetChain returns a copy of the error chain
func (c *ErrorCollector) GetChain() *ErrorChain {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	newChain := NewErrorChain()
	for _, err := range c.chain.All() {
		newChain.Add(err)
	}
	return newChain
}

// Clear clears all collected errors
func (c *ErrorCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chain = NewErrorChain()
}