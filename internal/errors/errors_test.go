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

func TestFormatErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      QgaError
		expected string
	}{
		{
			name: "with_underlying_error",
			err: &ConnectionError{
				wrappedError: fmt.Errorf("connection failed"),
				kind:         ConnectErrorKind,
			},
			expected: "Error: Connection => connection failed",
		},
		{
			name: "with_nil_underlying",
			err: &ConnectionError{
				wrappedError: nil,
				kind:         ConnectErrorKind,
			},
			expected: "Error: Connection => <nil>",
		},
		{
			name: "transport_error",
			err: &TransportError{
				wrappedError: fmt.Errorf("write failed"),
				kind:         Write,
			},
			expected: "Error: Transport => write failed",
		},
		{
			name: "codec_error",
			err: &CodecError{
				wrappedError: fmt.Errorf("marshal failed"),
				kind:         Marshal,
			},
			expected: "Error: Codec => marshal failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorMessage(tt.err)
			if result != tt.expected {
				t.Errorf("formatErrorMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestQgaErrorInterface(t *testing.T) {
	// Test that all error types implement QgaError interface
	var _ QgaError = &ConnectionError{}
	var _ QgaError = &TransportError{}
	var _ QgaError = &CodecError{}
}

func TestDomainConstants(t *testing.T) {
	tests := []struct {
		domain   DomainType
		expected string
	}{
		{TransportDomain, "Transport"},
		{ConnectionDomain, "Connection"},
		{ProtocolDomain, "Protocol"},
		{CodecDomain, "Codec"},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			if string(tt.domain) != tt.expected {
				t.Errorf("Domain constant = %q, want %q", string(tt.domain), tt.expected)
			}
		})
	}
}

func TestErrorUnwrapping(t *testing.T) {
	originalErr := fmt.Errorf("original error")

	// Test ConnectionError
	connErr := NewConnectionError(originalErr, ConnectErrorKind)
	if !errors.Is(connErr, originalErr) {
		t.Error("ConnectionError should wrap original error")
	}

	// Test TransportError
	transportErr := NewTransportError(originalErr, Connect)
	if !errors.Is(transportErr, originalErr) {
		t.Error("TransportError should wrap original error")
	}

	// Test CodecError
	codecErr := NewCodecError(originalErr, Marshal)
	if !errors.Is(codecErr, originalErr) {
		t.Error("CodecError should wrap original error")
	}
}
