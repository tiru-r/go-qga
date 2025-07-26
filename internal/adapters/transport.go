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

package adapters

import (
	"context"
	
	"github.com/prevostcorentin/go-qga/internal/core"
	"github.com/prevostcorentin/go-qga/internal/qmp/transport"
	"github.com/prevostcorentin/go-qga/internal/errors"
)

// TransportAdapter adapts the transport package to core interfaces
type TransportAdapter struct {
	transport transport.Transport
}

// NewTransportAdapter creates a new transport adapter
func NewTransportAdapter(t transport.Transport) *TransportAdapter {
	return &TransportAdapter{transport: t}
}

// Connect implements core.Transport interface
func (a *TransportAdapter) Connect(ctx context.Context) *errors.TransportError {
	return a.transport.Connect(ctx)
}

// Close implements core.Transport interface
func (a *TransportAdapter) Close() error {
	return a.transport.Close()
}

// Path implements core.Transport interface
func (a *TransportAdapter) Path() string {
	return a.transport.Path()
}

// Read implements core.Transport interface
func (a *TransportAdapter) Read(ctx context.Context) ([]byte, error) {
	return a.transport.Read(ctx)
}

// Write implements core.Transport interface
func (a *TransportAdapter) Write(ctx context.Context, bytes []byte) error {
	return a.transport.Write(ctx, bytes)
}

// TransportFactory creates transport adapters
type TransportFactory struct{}

// NewTransportFactory creates a new transport factory
func NewTransportFactory() *TransportFactory {
	return &TransportFactory{}
}

// CreateTransport implements core.Factory interface
func (f *TransportFactory) CreateTransport(transportType string, path string) (core.Transport, error) {
	var tt transport.TransportType
	switch transportType {
	case "unix":
		tt = transport.Unix
	default:
		return nil, errors.NewTransportError(nil, errors.Connect)
	}
	
	t, err := transport.NewTransport(tt, path)
	if err != nil {
		return nil, errors.NewTransportError(err, errors.Connect)
	}
	
	return NewTransportAdapter(t), nil
}

// CreateConnection creates a connection (placeholder)
func (f *TransportFactory) CreateConnection(transport core.Transport) (core.Connection, error) {
	// This would be implemented to create connection adapters
	return nil, errors.NewTransportError(nil, errors.Connect)
}

// CreateExecutor creates an executor (placeholder)
func (f *TransportFactory) CreateExecutor(connection core.Connection) (core.CommandExecutor, error) {
	// This would be implemented to create executor adapters
	return nil, errors.NewTransportError(nil, errors.Connect)
}

// CreateAgent creates an agent (placeholder)
func (f *TransportFactory) CreateAgent(config any) (core.Agent, error) {
	// This would be implemented to create agent adapters
	return nil, errors.NewTransportError(nil, errors.Connect)
}

// CreateClient creates a client (placeholder)
func (f *TransportFactory) CreateClient(socketPath string) (core.Client, error) {
	// This would be implemented to create client adapters
	return nil, errors.NewTransportError(nil, errors.Connect)
}