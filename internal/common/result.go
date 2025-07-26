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

import "fmt"

// Result represents an optimistic result that can succeed or fail gracefully
type Result[T any] struct {
	value T
	err   error
}

// Success creates a successful result
func Success[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Failure creates a failed result with simple error message
func Failure[T any](message string, args ...any) Result[T] {
	return Result[T]{err: fmt.Errorf(message, args...)}
}

// FailureFrom creates a failed result from an existing error
func FailureFrom[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// IsOk returns true if the result is successful
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr returns true if the result failed
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Value returns the value if successful, otherwise the zero value
func (r Result[T]) Value() T {
	return r.value
}

// Error returns the error if failed, otherwise nil
func (r Result[T]) Error() error {
	return r.err
}

// Unwrap returns the value and error (compatibility with UnifiedResult)
func (r Result[T]) Unwrap() (T, error) {
	return r.value, r.err
}

// IsSuccess returns true if the result is successful (compatibility alias)
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// IsFailure returns true if the result failed (compatibility alias)
func (r Result[T]) IsFailure() bool {
	return r.err != nil
}

// ValueOr returns the value if successful, otherwise the provided default
func (r Result[T]) ValueOr(defaultValue T) T {
	if r.IsOk() {
		return r.value
	}
	return defaultValue
}

// Map transforms the value if successful, otherwise returns the same error
func (r Result[T]) Map(fn func(T) T) Result[T] {
	if r.IsErr() {
		return r
	}
	return Success(fn(r.value))
}

// Then chains operations, only proceeding if the current result is successful
func (r Result[T]) Then(fn func(T) Result[T]) Result[T] {
	if r.IsErr() {
		return r
	}
	return fn(r.value)
}

