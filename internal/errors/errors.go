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
	"errors"
	"fmt"
)

// Simple Go error patterns - no complex hierarchies needed

var (
	ErrTransportClosed    = errors.New("transport is closed")
	ErrConnectionClosed   = errors.New("connection is closed") 
	ErrConnectionNil      = errors.New("connection is nil")
	ErrInvalidMessage     = errors.New("invalid message format")
	ErrTimeout           = errors.New("operation timed out")
	ErrCommandNil         = errors.New("command cannot be nil")
	ErrMissingReturn      = errors.New("missing return field in QGA response")
)

// Wrap creates a simple wrapped error - idiomatic Go
func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf creates a simple wrapped error with formatting
func Wrapf(err error, format string, args ...any) error {
	return fmt.Errorf(format+": %w", append(args, err)...)
}
