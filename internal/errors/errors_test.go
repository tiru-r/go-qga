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
	"testing"
)

func TestSimplifiedErrorApproach(t *testing.T) {
	// Test that our simplified approach covers all the error scenarios
	// that the complex error system was trying to handle
	
	tests := []struct {
		name        string
		createError func() error
		checkError  func(error) bool
		description string
	}{
		{
			name: "connection_errors",
			createError: func() error {
				return fmt.Errorf("connection failed: %w", ErrConnectionClosed)
			},
			checkError: func(err error) bool {
				return errors.Is(err, ErrConnectionClosed)
			},
			description: "Connection errors should be wrappable and checkable",
		},
		{
			name: "transport_errors", 
			createError: func() error {
				return fmt.Errorf("transport write failed: %w", ErrTransportClosed)
			},
			checkError: func(err error) bool {
				return errors.Is(err, ErrTransportClosed)
			},
			description: "Transport errors should be wrappable and checkable",
		},
		{
			name: "codec_errors",
			createError: func() error {
				baseErr := errors.New("invalid JSON")
				return fmt.Errorf("marshal error: %w", baseErr)
			},
			checkError: func(err error) bool {
				return err.Error() == "marshal error: invalid JSON"
			}, 
			description: "Codec errors should provide clear context",
		},
		{
			name: "timeout_errors",
			createError: func() error {
				return fmt.Errorf("operation timed out: %w", ErrTimeout)
			},
			checkError: func(err error) bool {
				return errors.Is(err, ErrTimeout)
			},
			description: "Timeout errors should be identifiable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()
			if err == nil {
				t.Error("expected error, got nil")
			}
			
			if !tt.checkError(err) {
				t.Errorf("error check failed for %s: %v", tt.description, err)
			}
		})
	}
}

func TestErrorFormatting(t *testing.T) {
	// Test that our simplified error formatting is clear and useful
	baseErr := errors.New("network unreachable")
	wrappedErr := fmt.Errorf("failed to connect to /tmp/qga.sock: %w", baseErr)
	
	expected := "failed to connect to /tmp/qga.sock: network unreachable"
	if wrappedErr.Error() != expected {
		t.Errorf("got %q, want %q", wrappedErr.Error(), expected)
	}
	
	// Test unwrapping
	if !errors.Is(wrappedErr, baseErr) {
		t.Error("wrapped error should be detectable")
	}
}

func TestAllPredefinedErrors(t *testing.T) {
	// Ensure all our predefined errors work correctly
	predefinedErrors := []error{
		ErrTransportClosed,
		ErrConnectionClosed,
		ErrConnectionNil,
		ErrInvalidMessage,
		ErrTimeout,
		ErrCommandNil,
		ErrMissingReturn,
	}
	
	for _, err := range predefinedErrors {
		if err == nil {
			t.Error("predefined error should not be nil")
		}
		
		if err.Error() == "" {
			t.Error("predefined error should have non-empty message")
		}
		
		// Test that they can be wrapped
		wrapped := fmt.Errorf("context: %w", err)
		if !errors.Is(wrapped, err) {
			t.Errorf("wrapped error should be detectable: %v", err)
		}
	}
}