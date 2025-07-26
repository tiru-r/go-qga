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
	"testing"
)

func TestTransportError(t *testing.T) {
	originalErr := fmt.Errorf("network unreachable")

	tests := []struct {
		name         string
		wrappedError error
		kind         TransportErrorKind
		expectedKind string
	}{
		{
			name:         "connect_error",
			wrappedError: originalErr,
			kind:         Connect,
			expectedKind: "Connect",
		},
		{
			name:         "write_error",
			wrappedError: originalErr,
			kind:         Write,
			expectedKind: "Write",
		},
		{
			name:         "read_error",
			wrappedError: originalErr,
			kind:         Read,
			expectedKind: "Read",
		},
		{
			name:         "close_error",
			wrappedError: originalErr,
			kind:         Close,
			expectedKind: "Close",
		},
		{
			name:         "flush_error",
			wrappedError: originalErr,
			kind:         Flush,
			expectedKind: "Flush",
		},
		{
			name:         "timeout_error",
			wrappedError: originalErr,
			kind:         Timeout,
			expectedKind: "Timeout",
		},
		{
			name:         "not_connected_error",
			wrappedError: originalErr,
			kind:         NotConnected,
			expectedKind: "Not connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTransportError(tt.wrappedError, tt.kind)

			// Test Domain()
			if err.Domain() != TransportDomain {
				t.Errorf("Domain() = %v, want %v", err.Domain(), TransportDomain)
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
			expectedMsg := "Error: Transport => network unreachable"
			if err.Error() != expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
			}
		})
	}
}

func TestTransportErrorWithNilWrapped(t *testing.T) {
	err := NewTransportError(nil, Connect)

	// Test Error() with nil wrapped error
	expectedMsg := "Error: Transport => <nil>"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
	}

	// Test Unwrap() returns nil
	if err.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", err.Unwrap())
	}
}

func TestTransportErrorKinds(t *testing.T) {
	kinds := []struct {
		kind     TransportErrorKind
		expected string
	}{
		{Connect, "Connect"},
		{Write, "Write"},
		{Read, "Read"},
		{Close, "Close"},
		{Flush, "Flush"},
		{Timeout, "Timeout"},
		{NotConnected, "Not connected"},
	}

	for _, k := range kinds {
		t.Run(string(k.kind), func(t *testing.T) {
			if string(k.kind) != k.expected {
				t.Errorf("Kind constant = %v, want %v", string(k.kind), k.expected)
			}
		})
	}
}

func TestTransportErrorImplementsQgaError(t *testing.T) {
	originalErr := fmt.Errorf("test error")
	err := NewTransportError(originalErr, Write)

	// Verify it implements QgaError interface
	var qgaErr QgaError = err

	if qgaErr.Domain() != TransportDomain {
		t.Errorf("QgaError.Domain() = %v, want %v", qgaErr.Domain(), TransportDomain)
	}

	if qgaErr.Kind() != "Write" {
		t.Errorf("QgaError.Kind() = %v, want %v", qgaErr.Kind(), "Write")
	}

	if qgaErr.Unwrap() != originalErr {
		t.Errorf("QgaError.Unwrap() = %v, want %v", qgaErr.Unwrap(), originalErr)
	}
}
