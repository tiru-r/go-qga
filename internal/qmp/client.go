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

package qmp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// Client provides a simple, leak-free QMP client
type Client struct {
	mu     sync.Mutex
	conn   net.Conn
	buffer []byte
	closed bool
}

// Connect creates a new QMP connection (deprecated: use NewClient)
func Connect(socketPath string) (*Client, error) {
	return NewClient(socketPath)
}

// NewClient creates a new QMP client with proper resource management
func NewClient(socketPath string) (*Client, error) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", socketPath, err)
	}

	client := &Client{
		conn:   conn,
		buffer: make([]byte, 4096), // Direct allocation, no global pools
		closed: false,
	}

	// Consume the QMP banner
	if err := client.consumeBanner(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to consume banner: %w", err)
	}

	return client, nil
}

// Execute runs a QMP command synchronously
func (c *Client) Execute(command string, args any) (map[string]any, error) {
	return c.ExecuteWithContext(context.Background(), command, args)
}

// ExecuteWithContext runs a QMP command with context support
func (c *Client) ExecuteWithContext(ctx context.Context, command string, args any) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	// Build request
	request := map[string]any{
		common.JSONFieldExecute: command,
	}

	if args != nil {
		request["arguments"] = args
	}

	// Marshal request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Send request with newline
	data := append(jsonData, '\n')
	
	// Set write deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetWriteDeadline(deadline)
	} else {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	}
	
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	return c.readResponse(ctx)
}

// ExecuteAsync runs a QMP command asynchronously
func (c *Client) ExecuteAsync(ctx context.Context, command string, args any) <-chan AsyncResult {
	resultCh := make(chan AsyncResult, 1)
	
	go func() {
		defer close(resultCh)
		
		result, err := c.ExecuteWithContext(ctx, command, args)
		select {
		case resultCh <- AsyncResult{Data: result, Err: err}:
		case <-ctx.Done():
			// Context cancelled, don't send result
		}
	}()
	
	return resultCh
}

// GetHostname gets the VM hostname
func (c *Client) GetHostname() (string, error) {
	result, err := c.Execute(common.CommandGuestGetHostName, nil)
	if err != nil {
		return "", err
	}

	if returnData, ok := result[common.JSONFieldReturn].(map[string]any); ok {
		if hostname, ok := returnData[common.JSONFieldName].(string); ok {
			return hostname, nil
		}
	}

	return "", fmt.Errorf("hostname not found in response")
}

// Close closes the connection and cleans up resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	
	if c.conn != nil {
		return c.conn.Close()
	}
	
	return nil
}

// consumeBanner reads and discards the QMP banner
func (c *Client) consumeBanner() error {
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{})

	n, err := c.conn.Read(c.buffer)
	if err != nil {
		return fmt.Errorf("failed to read banner: %w", err)
	}
	
	// Banner should be valid JSON - basic validation
	var banner map[string]any
	if err := json.Unmarshal(c.buffer[:n], &banner); err != nil {
		return fmt.Errorf("invalid QMP banner: %w", err)
	}
	
	return nil
}

// readResponse reads and parses a QMP response
func (c *Client) readResponse(ctx context.Context) (map[string]any, error) {
	// Set read deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetReadDeadline(deadline)
	} else {
		c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	}
	defer c.conn.SetReadDeadline(time.Time{})

	n, err := c.conn.Read(c.buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var response map[string]any
	if err := json.Unmarshal(c.buffer[:n], &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for QMP errors
	if errorData, hasError := response["error"]; hasError {
		if errorInfo, ok := errorData.(map[string]any); ok {
			if desc, ok := errorInfo["desc"].(string); ok {
				return nil, fmt.Errorf("QMP error: %s", desc)
			}
		}
		return nil, fmt.Errorf("unknown QMP error occurred")
	}

	return response, nil
}

// AsyncResult represents the result of an async operation
type AsyncResult struct {
	Data map[string]any
	Err  error
}