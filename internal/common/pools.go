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

// BufferPool provides buffer management
type BufferPool struct {
	standard *sync.Pool
	large    *sync.Pool
}

// GetStandard gets a standard buffer
func (p *BufferPool) GetStandard() []byte {
	return p.standard.Get().([]byte)
}

// PutStandard returns a standard buffer
func (p *BufferPool) PutStandard(buf []byte) {
	p.standard.Put(buf)
}

// GetLarge gets a large buffer
func (p *BufferPool) GetLarge() []byte {
	return p.large.Get().([]byte)
}

// PutLarge returns a large buffer
func (p *BufferPool) PutLarge(buf []byte) {
	p.large.Put(buf)
}

// GlobalBufferPool provides global access to buffer pools
var GlobalBufferPool = &BufferPool{
	standard: &sync.Pool{New: func() any { return make([]byte, StandardBufferSize) }},
	large:    &sync.Pool{New: func() any { return make([]byte, LargeBufferSize) }},
}

// ErrorChannel creates a buffered error channel
func ErrorChannel() chan error {
	return make(chan error, 1)
}