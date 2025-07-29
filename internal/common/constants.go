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

import (
	"encoding/json"
)

// Protocol constants
const (
	LineTerminator = '\n'
)

// Common JSON field names to eliminate duplication
const (
	JSONFieldExecute = "execute"
	JSONFieldReturn  = "return"
	JSONFieldName    = "name"
	JSONFieldQMP     = "QMP"
)


// Common QGA command names
const (
	CommandGuestGetHostName = "guest-get-host-name"
)

// QMP message constants (pre-allocated to avoid runtime concatenation)
const (
	QmpBannerMessage           = `{"QMP": {"version": {"qemu": {"major": "8", "minor": "2", "micro": "0"}, "package": "qemu-8.2.0"}, "capabilities": []}}` + "\n"
	QmpHostnameResponse        = `{"return": {"name": "perfect-vm"}}` + "\n"
	QmpCommandNotFoundResponse = `{"error": {"class": "CommandNotFound", "desc": "Command not found"}}` + "\n"
	QmpHostnameRequest         = `{"execute": "guest-get-host-name"}` + "\n"
)

// Common JSON utilities
func MarshalToBytes(v any) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalToBytesIgnoreError marshals to JSON bytes, ignoring errors (for optimistic code)
func MarshalToBytesIgnoreError(v any) []byte {
	bytes, _ := json.Marshal(v)
	return bytes
}

// Simplified constants - removed over-engineered buffer size options