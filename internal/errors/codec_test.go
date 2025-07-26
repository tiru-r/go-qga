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

func TestCodecError(t *testing.T) {
	originalErr := fmt.Errorf("invalid JSON")

	tests := []struct {
		name         string
		wrappedError error
		kind         CodecErrorKind
		expectedKind string
	}{
		{
			name:         "marshal_error",
			wrappedError: originalErr,
			kind:         Marshal,
			expectedKind: "Marshal",
		},
		{
			name:         "unmarshal_error",
			wrappedError: originalErr,
			kind:         Unmarshal,
			expectedKind: "Unmarshal",
		},
		{
			name:         "type_error",
			wrappedError: originalErr,
			kind:         Type,
			expectedKind: "Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewCodecError(tt.wrappedError, tt.kind)

			// Test Domain()
			if err.Domain() != CodecDomain {
				t.Errorf("Domain() = %v, want %v", err.Domain(), CodecDomain)
			}

			// Test Kind()
			if err.Kind() != tt.expectedKind {
				t.Errorf("Kind() = %v, want %v", err.Kind(), tt.expectedKind)
			}

			// Test Unwrap()
			if err.Unwrap() != tt.wrappedError {
				t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), tt.wrappedError)
			}

			// Test Error()
			expectedMsg := "Error: Codec => invalid JSON"
			if err.Error() != expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
			}
		})
	}
}

func TestCodecErrorWithNilWrapped(t *testing.T) {
	err := NewCodecError(nil, Marshal)

	// Test Error() with nil wrapped error
	expectedMsg := "Error: Codec => <nil>"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
	}

	// Test Unwrap() returns nil
	if err.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", err.Unwrap())
	}
}

func TestCodecErrorKinds(t *testing.T) {
	kinds := []struct {
		kind     CodecErrorKind
		expected string
	}{
		{Marshal, "Marshal"},
		{Unmarshal, "Unmarshal"},
		{Type, "Type"},
	}

	for _, k := range kinds {
		t.Run(string(k.kind), func(t *testing.T) {
			if string(k.kind) != k.expected {
				t.Errorf("Kind constant = %v, want %v", string(k.kind), k.expected)
			}
		})
	}
}

func TestCodecErrorImplementsQgaError(t *testing.T) {
	originalErr := fmt.Errorf("test error")
	err := NewCodecError(originalErr, Unmarshal)

	// Verify it implements QgaError interface
	var qgaErr QgaError = err

	if qgaErr.Domain() != CodecDomain {
		t.Errorf("QgaError.Domain() = %v, want %v", qgaErr.Domain(), CodecDomain)
	}

	if qgaErr.Kind() != "Unmarshal" {
		t.Errorf("QgaError.Kind() = %v, want %v", qgaErr.Kind(), "Unmarshal")
	}

	if qgaErr.Unwrap() != originalErr {
		t.Errorf("QgaError.Unwrap() = %v, want %v", qgaErr.Unwrap(), originalErr)
	}
}

func TestCodecErrorChaining(t *testing.T) {
	originalErr := fmt.Errorf("syntax error")
	codecErr := NewCodecError(originalErr, Unmarshal)

	// Test error chaining with errors.Is
	wrappedErr := fmt.Errorf("wrapped: %w", codecErr)
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Error chaining not working correctly with errors.Is")
	}

	// Test unwrapping works through the chain
	var targetCodecErr *CodecError
	if !errors.As(wrappedErr, &targetCodecErr) {
		t.Error("Error chaining not working correctly with errors.As")
	}

	if targetCodecErr.Unwrap() != originalErr {
		t.Error("CodecError unwrapping not working correctly")
	}
}
