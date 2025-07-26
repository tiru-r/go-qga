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

package transport_test

import (
	"bufio"
	"context"
	"net"
	"sync"
	"testing"

	. "github.com/prevostcorentin/go-qga/internal/errors"
	"github.com/prevostcorentin/go-qga/internal/qmp/transport"
	. "github.com/prevostcorentin/go-qga/internal/testing"
)

func TestUnexistingSocketFailure(t *testing.T) {
	unexistingSocketPath := "/this/socket/does/not/exist/for/sure"
	unixTransport, err := transport.NewTransport("unix", unexistingSocketPath)
	if err != nil {
		t.Fatalf("creating transport: %v", err)
	}
	if unixTransport.Path() != unexistingSocketPath {
		t.Fatalf(`wrong transport path "%v". expected "%s"`, unixTransport.Path(), unexistingSocketPath)
	}
	var transportError *TransportError
	ctx := context.Background()
	if transportError = unixTransport.Connect(ctx); transportError == nil {
		t.Fatal("there should have been an error here")
	}
	if transportError.Domain() != TransportDomain {
		t.Fatalf(`wrong error domain "%v". expected "%s"`, transportError.Domain(), TransportDomain)
	}
	if transportError.Kind() != string(Connect) {
		t.Fatalf(`wrong error kind "%v". expected "%s"`, transportError.Kind(), Connect)
	}
}

type echoAgent struct {
	listener net.Listener
	done     chan struct{}
	wg       sync.WaitGroup
	t        *testing.T
	path     string
}

func newEchoAgent(t *testing.T) *echoAgent {
	return &echoAgent{t: t, done: make(chan struct{}), path: BuildSocketPath(t)}
}

func (agent *echoAgent) Path() string {
	return agent.path
}

func (agent *echoAgent) Start() {
	var listenerError error
	agent.listener, listenerError = net.Listen("unix", agent.Path())
	if listenerError != nil {
		agent.t.Fatalf("can't listen on %s: %v", agent.Path(), listenerError)
	}
	go func() {
		for {
			connection, acceptError := agent.listener.Accept()
			if acceptError != nil {
				select {
				case <-agent.done:
					return
				default:
					agent.t.Fatalf("can't accept connection: %v", acceptError)
				}
			}
			agent.wg.Add(1)
			go func() {
				defer agent.wg.Done()
				defer connection.Close()
				connectionReader, connectionWriter := bufio.NewReader(connection), bufio.NewWriter(connection)
				bytes, connectionError := connectionReader.ReadBytes('\n')
				if connectionError != nil {
					agent.t.Logf("can't read (may be normal): %v", connectionError)
					return
				}
				if _, writeError := connectionWriter.Write(bytes); writeError != nil {
					agent.t.Logf("can't write (may be normal): %v", writeError)
					return
				}
				if flushError := connectionWriter.Flush(); flushError != nil {
					agent.t.Logf("can't flush (may be normal): %v", flushError)
					return
				}
			}()
		}
	}()
}

func (agent *echoAgent) Stop() {
	close(agent.done)
	if err := agent.listener.Close(); err != nil {
		agent.t.Fatalf("can't close listener: %v", err)
	}
	agent.wg.Wait()
}

func TestReadWrite(t *testing.T) {
	ctx := context.Background()
	agent := newEchoAgent(t)
	agent.Start()
	defer agent.Stop()
	unixTransport, err := transport.NewTransport("unix", agent.Path())
	if err != nil {
		t.Fatalf("creating transport: %v", err)
	}
	if err := unixTransport.Connect(ctx); err != nil {
		t.Fatalf("while connecting socket: %v", err)
	}
	expectedResponse := []byte("some string\n")
	if writeError := unixTransport.Write(ctx, expectedResponse); writeError != nil {
		t.Fatalf("while writing: %v", writeError)
	}
	response, readError := unixTransport.Read(ctx)
	if readError != nil {
		t.Fatalf("while reading: %v", readError)
	}
	if string(response) != string(expectedResponse) {
		t.Errorf(`wrong response "%v". expected "%s"`, string(response), string(expectedResponse))
	}
}

type closeConnectionAgent struct {
	listener net.Listener
	t        *testing.T
	done     chan struct{}
	wg       sync.WaitGroup
	path     string
}

func newCloseConnectionAgent(t *testing.T) *closeConnectionAgent {
	return &closeConnectionAgent{t: t, done: make(chan struct{}), path: BuildSocketPath(t)}
}

func (agent *closeConnectionAgent) Path() string {
	return agent.path
}

func (agent *closeConnectionAgent) Start() {
	var listenerError error
	agent.listener, listenerError = net.Listen("unix", agent.Path())
	if listenerError != nil {
		agent.t.Fatalf("can't listen on %s: %v", agent.Path(), listenerError)
	}
	go func() {
		for {
			connection, acceptError := agent.listener.Accept()
			if acceptError != nil {
				select {
				case <-agent.done:
					return
				default:
					agent.t.Fatalf("can't accept connection: %v", acceptError)
				}
			}
			agent.wg.Add(1)
			go func() {
				defer agent.wg.Done()
				defer connection.Close()
				// Immediately close the connection without sending data
				// This will cause the read to fail
			}()
		}
	}()
}

func (agent *closeConnectionAgent) Stop() {
	close(agent.done)
	if err := agent.listener.Close(); err != nil {
		agent.t.Fatalf("can't close listener: %v", err)
	}
	agent.wg.Wait()
}

func TestNoWrite(t *testing.T) {
	ctx := context.Background()
	agent := newCloseConnectionAgent(t)
	agent.Start()
	defer agent.Stop()
	transport, err := transport.NewTransport("unix", agent.Path())
	if err != nil {
		t.Fatalf("creating transport: %v", err)
	}
	if connectError := transport.Connect(ctx); connectError != nil {
		t.Fatalf("while connecting: %v", connectError)
	}
	largePayload := make([]byte, 1<<20) // 1 MiB of zeros
	var writeError error
	if writeError = transport.Write(ctx, largePayload); writeError == nil {
		t.Fatal("there should have been an error here")
	}
	transportError := writeError.(*TransportError)
	if transportError.Domain() != TransportDomain {
		t.Errorf(`wrong error domain "%v". expected "%s"`, transportError.Domain(), TransportDomain)
	}
	if transportError.Kind() != Write {
		t.Errorf(`wrong error kind "%v". expected "%s"`, transportError.Kind(), Write)
	}
}

func TestNoRead(t *testing.T) {
	ctx := context.Background()
	agent := newCloseConnectionAgent(t)
	agent.Start()
	defer agent.Stop()

	transport, err := transport.NewTransport("unix", agent.Path())
	if err != nil {
		t.Fatalf("creating transport: %v", err)
	}
	if connectError := transport.Connect(ctx); connectError != nil {
		t.Fatalf("while connecting: %v", connectError)
	}
	defer transport.Close()

	var readError error
	if _, readError = transport.Read(ctx); readError == nil {
		t.Fatal("there should have been an error here")
	}
	transportError := readError.(*TransportError)
	if transportError.Domain() != TransportDomain {
		t.Errorf(`wrong error domain "%v". expected "%s"`, transportError.Domain(), TransportDomain)
	}
	if transportError.Kind() != Read {
		t.Errorf(`wrong error kind "%v". expected "%s"`, transportError.Kind(), Read)
	}
}
