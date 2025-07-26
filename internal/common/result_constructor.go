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

import "context"

// Constructor represents a unified constructor pattern for the library
type Constructor[T any] struct {
	create func() (T, error)
}

// NewConstructor creates a new constructor with the given creation function
func NewConstructor[T any](create func() (T, error)) *Constructor[T] {
	return &Constructor[T]{create: create}
}

// Build executes the constructor and returns a Result
func (c *Constructor[T]) Build() Result[T] {
	value, err := c.create()
	if err != nil {
		return FailureFrom[T](err)
	}
	return Success(value)
}

// BuildWithContext executes the constructor with context support
func (c *Constructor[T]) BuildWithContext(ctx context.Context) Result[T] {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return FailureFrom[T](ctx.Err())
	default:
	}
	
	return c.Build()
}

// AsyncConstructor represents an asynchronous constructor pattern
type AsyncConstructor[T any] struct {
	create func(context.Context) <-chan Result[T]
}

// NewAsyncConstructor creates a new async constructor
func NewAsyncConstructor[T any](create func(context.Context) <-chan Result[T]) *AsyncConstructor[T] {
	return &AsyncConstructor[T]{create: create}
}

// Build executes the async constructor
func (c *AsyncConstructor[T]) Build(ctx context.Context) <-chan Result[T] {
	return c.create(ctx)
}