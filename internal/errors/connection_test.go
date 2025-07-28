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

func TestConnectionErrors(t *testing.T) {
	baseErr := errors.New("network timeout")
	
	tests := []struct {
		name string
		err  error
		want string
		base error
	}{
		{
			name: "connection_error",
			err:  fmt.Errorf("connection error: %w", ErrConnectionClosed),
			want: "connection error: connection is closed", 
			base: ErrConnectionClosed,
		},
		{
			name: "connect_error",
			err:  fmt.Errorf("failed to connect to /tmp/test.sock: %w", baseErr),
			want: "failed to connect to /tmp/test.sock: network timeout",
			base: baseErr,
		},
		{
			name: "send_error",
			err:  fmt.Errorf("send error: %w", ErrTimeout),
			want: "send error: operation timed out",
			base: ErrTimeout,
		},
		{
			name: "read_error", 
			err:  fmt.Errorf("read error: %w", baseErr),
			want: "read error: network timeout",
			base: baseErr,
		},
		{
			name: "close_error",
			err:  fmt.Errorf("close error: %w", baseErr),
			want: "close error: network timeout", 
			base: baseErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.want)
			}
			
			if !errors.Is(tt.err, tt.base) {
				t.Errorf("error should wrap %v", tt.base)
			}
		})
	}
}

func TestConnectionErrorPatterns(t *testing.T) {
	// Test the error patterns used in the simplified codebase
	
	// Pattern 1: Simple predefined errors
	err1 := ErrConnectionNil
	if err1.Error() != "connection is nil" {
		t.Errorf("got %q, want %q", err1.Error(), "connection is nil")
	}
	
	// Pattern 2: Wrapped errors with context
	baseErr := errors.New("socket not found")
	err2 := fmt.Errorf("connection error: %w", baseErr)
	if !errors.Is(err2, baseErr) {
		t.Error("wrapped error should be detectable with errors.Is")
	}
	
	// Pattern 3: Formatted errors with details
	err3 := fmt.Errorf("failed to connect to %s: %w", "/tmp/test.sock", ErrTimeout)
	if !errors.Is(err3, ErrTimeout) {
		t.Error("formatted error should wrap original error")
	}
}