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

func TestSimpleErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "transport_closed",
			err:  ErrTransportClosed,
			want: "transport is closed",
		},
		{
			name: "connection_closed", 
			err:  ErrConnectionClosed,
			want: "connection is closed",
		},
		{
			name: "connection_nil",
			err:  ErrConnectionNil,
			want: "connection is nil",
		},
		{
			name: "invalid_message",
			err:  ErrInvalidMessage,
			want: "invalid message format",
		},
		{
			name: "timeout",
			err:  ErrTimeout,
			want: "operation timed out",
		},
		{
			name: "command_nil",
			err:  ErrCommandNil,
			want: "command cannot be nil",
		},
		{
			name: "missing_return",
			err:  ErrMissingReturn,
			want: "missing return field in QGA response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}

func TestWrapErrors(t *testing.T) {
	baseErr := errors.New("base error")
	
	wrappedErr := fmt.Errorf("marshal error: %w", baseErr)
	if !errors.Is(wrappedErr, baseErr) {
		t.Error("wrapped error should unwrap to base error")
	}
	
	if wrappedErr.Error() != "marshal error: base error" {
		t.Errorf("got %q, want %q", wrappedErr.Error(), "marshal error: base error")
	}
}

func TestErrorCreation(t *testing.T) {
	// Test simple error creation patterns used in the simplified codebase
	err1 := fmt.Errorf("connection error: %w", ErrConnectionClosed)
	err2 := fmt.Errorf("read error: %w", ErrTimeout)
	err3 := fmt.Errorf("write error: operation failed")
	
	if !errors.Is(err1, ErrConnectionClosed) {
		t.Error("err1 should wrap ErrConnectionClosed")
	}
	
	if !errors.Is(err2, ErrTimeout) {
		t.Error("err2 should wrap ErrTimeout")
	}
	
	if err3.Error() != "write error: operation failed" {
		t.Errorf("got %q, want %q", err3.Error(), "write error: operation failed")
	}
}