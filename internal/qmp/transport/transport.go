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

package transport

import (
	"context"
	"fmt"

	"github.com/prevostcorentin/go-qga/internal/common"
)

type TransportType string

const (
	Unix TransportType = "unix"
)

type Transport interface {
	Connect(ctx context.Context) error
	Close() error
	Path() string
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, bytes []byte) error
}

func NewTransport(transportType TransportType, path string) (Transport, error) {
	switch transportType {
	case Unix:
		return &unixTransport{
			BaseState: common.NewBaseState(),
			path:      path,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}
}
