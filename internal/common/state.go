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

import "sync"

// BaseState provides common state management for components
// Fields ordered for optimal memory alignment
type BaseState struct {
	mu     sync.RWMutex // 24 bytes
	closed bool         // 1 byte (+ 7 bytes padding - unavoidable due to struct size)
}

// NewBaseState creates a new base state
func NewBaseState() *BaseState {
	return &BaseState{}
}

// IsClosed returns whether the component is closed (thread-safe)
func (b *BaseState) IsClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.closed
}

// MarkClosed marks the component as closed (thread-safe)
func (b *BaseState) MarkClosed() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
}

// CloseWithLock marks as closed while holding a lock (to avoid deadlock)
func (b *BaseState) CloseWithLock() bool {
	if b.closed {
		return false // already closed
	}
	b.closed = true
	return true // was closed by this call
}

// IsClosedWhileLocked checks if closed while already holding a lock (to avoid deadlock)
func (b *BaseState) IsClosedWhileLocked() bool {
	return b.closed
}

// Lock acquires write lock
func (b *BaseState) Lock() {
	b.mu.Lock()
}

// Unlock releases write lock
func (b *BaseState) Unlock() {
	b.mu.Unlock()
}

// RLock acquires read lock
func (b *BaseState) RLock() {
	b.mu.RLock()
}

// RUnlock releases read lock
func (b *BaseState) RUnlock() {
	b.mu.RUnlock()
}