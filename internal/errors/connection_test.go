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

func TestConnectionError(t *testing.T) {
	originalErr := fmt.Errorf("connection timeout")

	tests := []struct {
		name         string
		wrappedError error
		kind         ConnectionErrorKind
		expectedKind string
	}{
		{
			name:         "connect_error",
			wrappedError: originalErr,
			kind:         ConnectErrorKind,
			expectedKind: "Connect",
		},
		{
			name:         "send_error",
			wrappedError: originalErr,
			kind:         SendErrorKind,
			expectedKind: "Send",
		},
		{
			name:         "read_error",
			wrappedError: originalErr,
			kind:         ReadErrorKind,
			expectedKind: "Read",
		},
		{
			name:         "close_error",
			wrappedError: originalErr,
			kind:         CloseErrorKind,
			expectedKind: "Close",
		},
		{
			name:         "unknown_error",
			wrappedError: originalErr,
			kind:         UnknownErrorKind,
			expectedKind: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewConnectionError(tt.wrappedError, tt.kind)

			// Test Domain()
			if err.Domain() != ConnectionDomain {
				t.Errorf("Domain() = %v, want %v", err.Domain(), ConnectionDomain)
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
			expectedMsg := "Error: Connection => connection timeout"
			if err.Error() != expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
			}
		})
	}
}

func TestConnectionErrorWithNilWrapped(t *testing.T) {
	err := NewConnectionError(nil, ConnectErrorKind)

	// Test Error() with nil wrapped error
	expectedMsg := "Error: Connection => <nil>"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %v, want %v", err.Error(), expectedMsg)
	}

	// Test Unwrap() returns nil
	if err.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", err.Unwrap())
	}
}

func TestConnectionErrorKinds(t *testing.T) {
	kinds := []struct {
		kind     ConnectionErrorKind
		expected string
	}{
		{UnknownErrorKind, "Unknown"},
		{ConnectErrorKind, "Connect"},
		{SendErrorKind, "Send"},
		{ReadErrorKind, "Read"},
		{CloseErrorKind, "Close"},
	}

	for _, k := range kinds {
		t.Run(string(k.kind), func(t *testing.T) {
			if string(k.kind) != k.expected {
				t.Errorf("Kind constant = %v, want %v", string(k.kind), k.expected)
			}
		})
	}
}
