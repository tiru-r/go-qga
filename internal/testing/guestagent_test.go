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

package testing

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// Helper function to check if error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "i/o timeout")
}

func TestSocketAgent_Serve(t *testing.T) {
	socketPath := BuildSocketPath("test-agent")
	config := SocketAgentConfig{
		SocketPath: socketPath,
		Timeout:    30 * time.Second,
	}
	agent := NewSocketAgent(config)

	ctx, cancel := context.WithCancel(context.Background())

	handler := func(conn net.Conn) {
		defer conn.Close()
		data, err := io.ReadAll(conn)
		if err == nil {
			conn.Write([]byte("echo: " + string(data)))
		}
	}

	done := common.ErrorChannel()
	go func() {
		done <- agent.Serve(ctx, handler)
	}()

	agent.WaitReady()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	testMessage := "test message"
	_, err = conn.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	conn.Close()
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled && !isTimeoutError(err) {
			t.Errorf("expected context.Canceled or timeout, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("agent did not stop after context cancellation")
	}
}

func TestSocketAgent_ServeContextCancellation(t *testing.T) {
	socketPath := BuildSocketPath("test-agent")
	config := SocketAgentConfig{
		SocketPath: socketPath,
		Timeout:    30 * time.Second,
	}
	agent := NewSocketAgent(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(conn net.Conn) {
		conn.Close()
	}

	done := common.ErrorChannel()
	go func() {
		done <- agent.Serve(ctx, handler)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled && !isTimeoutError(err) {
			t.Errorf("expected context.Canceled or timeout, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("agent did not stop after context cancellation")
	}
}

func TestSocketAgent_ServeConcurrentConnections(t *testing.T) {
	socketPath := BuildSocketPath("test-agent")
	config := SocketAgentConfig{
		SocketPath: socketPath,
		Timeout:    30 * time.Second,
	}
	agent := NewSocketAgent(config)

	ctx, cancel := context.WithCancel(context.Background())

	responses := make(chan string, 3)
	handler := func(conn net.Conn) {
		defer conn.Close()
		data, err := io.ReadAll(conn)
		if err == nil {
			response := "processed: " + string(data)
			conn.Write([]byte(response))
			responses <- response
		}
	}

	done := common.ErrorChannel()
	go func() {
		done <- agent.Serve(ctx, handler)
	}()

	agent.WaitReady()

	for i := 0; i < 3; i++ {
		go func(id int) {
			conn, err := net.Dial("unix", socketPath)
			if err != nil {
				t.Errorf("connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()

			message := fmt.Sprintf("message%d", id)
			conn.Write([]byte(message))
			conn.Close()
		}(i)
	}

	receivedCount := 0
	timeout := time.After(time.Second)
	for receivedCount < 3 {
		select {
		case response := <-responses:
			if !strings.HasPrefix(response, "processed: message") {
				t.Errorf("unexpected response: %s", response)
			}
			receivedCount++
		case <-timeout:
			t.Errorf("timeout waiting for responses, received %d/3", receivedCount)
			cancel()
			<-done
			return
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != context.Canceled && !isTimeoutError(err) {
			t.Errorf("expected context.Canceled or timeout, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("agent did not stop after context cancellation")
	}
}

func TestSocketAgent_ServeHandlerPanic(t *testing.T) {
	socketPath := BuildSocketPath("test-agent")
	config := SocketAgentConfig{
		SocketPath: socketPath,
		Timeout:    30 * time.Second,
	}
	agent := NewSocketAgent(config)

	ctx, cancel := context.WithCancel(context.Background())

	handler := func(conn net.Conn) {
		defer conn.Close()
		panic("handler panic")
	}

	done := common.ErrorChannel()
	go func() {
		done <- agent.Serve(ctx, handler)
	}()

	agent.WaitReady()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	conn.Close()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled && !isTimeoutError(err) {
			t.Errorf("expected context.Canceled or timeout, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("agent did not stop after context cancellation")
	}
}

func TestBuildSocketPath(t *testing.T) {
	path1 := BuildSocketPath("test1")
	path2 := BuildSocketPath("test2")

	if !strings.HasSuffix(path1, "test1.sock") {
		t.Errorf("socket path should end with 'test1.sock', got: %s", path1)
	}

	if path1 == path2 {
		t.Error("BuildSocketPath should return different paths for different calls")
	}
}
