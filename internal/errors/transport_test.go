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

func TestTransportErrorPatterns(t *testing.T) {
	baseErr := errors.New("socket closed")
	
	tests := []struct {
		name        string
		createError func() error
		wantMessage string
		wantWrapped error
	}{
		{
			name: "connect_error",
			createError: func() error {
				return fmt.Errorf("failed to connect: %w", baseErr)
			},
			wantMessage: "failed to connect: socket closed",
			wantWrapped: baseErr,
		},
		{
			name: "write_error",
			createError: func() error {
				return fmt.Errorf("write error: %w", ErrTransportClosed)
			},
			wantMessage: "write error: transport is closed",
			wantWrapped: ErrTransportClosed,
		},
		{
			name: "read_error",
			createError: func() error {
				return fmt.Errorf("read error: %w", ErrTimeout)
			},
			wantMessage: "read error: operation timed out",
			wantWrapped: ErrTimeout,
		},
		{
			name: "close_error",
			createError: func() error {
				return fmt.Errorf("close error: %w", baseErr)
			},
			wantMessage: "close error: socket closed",
			wantWrapped: baseErr,
		},
		{
			name: "flush_error",
			createError: func() error {
				return fmt.Errorf("flush error: %w", baseErr)
			},
			wantMessage: "flush error: socket closed",
			wantWrapped: baseErr,
		},
		{
			name: "timeout_error",
			createError: func() error {
				return fmt.Errorf("operation timed out: %w", ErrTimeout)
			},
			wantMessage: "operation timed out: operation timed out",
			wantWrapped: ErrTimeout,
		},
		{
			name: "not_connected_error",
			createError: func() error {
				return fmt.Errorf("transport not connected: %w", ErrTransportClosed)
			},
			wantMessage: "transport not connected: transport is closed",
			wantWrapped: ErrTransportClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()
			
			if err.Error() != tt.wantMessage {
				t.Errorf("got message %q, want %q", err.Error(), tt.wantMessage)
			}
			
			if !errors.Is(err, tt.wantWrapped) {
				t.Errorf("error should wrap %v", tt.wantWrapped)
			}
		})
	}
}

func TestTransportErrorCreation(t *testing.T) {
	// Test the simple error creation patterns used in transport layer
	
	// Test predefined transport errors
	if ErrTransportClosed.Error() != "transport is closed" {
		t.Errorf("got %q, want %q", ErrTransportClosed.Error(), "transport is closed")
	}
	
	// Test error wrapping with context
	originalErr := errors.New("connection reset")
	wrappedErr := fmt.Errorf("transport write failed: %w", originalErr)
	
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("wrapped error should be detectable")
	}
	
	// Test error chaining
	err1 := errors.New("network down")
	err2 := fmt.Errorf("connection failed: %w", err1)
	err3 := fmt.Errorf("transport error: %w", err2)
	
	if !errors.Is(err3, err1) {
		t.Error("deeply wrapped error should be detectable")
	}
}