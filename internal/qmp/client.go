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
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// Connect creates a new QMP connection (deprecated: use NewClient)
func Connect(socketPath string) (*Client, error) {
	return NewClient(socketPath)
}

// Client provides a QMP client
type Client struct {
	mu     sync.RWMutex // Protect concurrent access
	conn   net.Conn
	path   string
	buffer []byte
}

// NewClient creates a new QMP connection
func NewClient(socketPath string) (*Client, error) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", socketPath, err)
	}

	client := &Client{
		conn:   conn,
		path:   socketPath,
		buffer: common.GlobalBufferPool.GetLarge(),
	}

	// Consume the QMP banner
	client.consumeBanner()

	return client, nil
}

// Execute runs a QMP command
func (c *Client) Execute(command string, args ...any) (map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return nil, fmt.Errorf("client connection is closed")
	}

	// Build request
	request := map[string]any{
		common.JSONFieldExecute: command,
	}

	// Add arguments if provided
	if len(args) > 0 && args[0] != nil {
		request["arguments"] = args[0]
	}

	// Send request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %v", err)
	}

	// Pre-allocate slice to avoid reallocation when appending line terminator
	data := make([]byte, len(jsonData)+1)
	copy(data, jsonData)
	data[len(jsonData)] = '\n'
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	// Read response
	return c.readResponse()
}

// GetHostname gets the VM hostname
func (c *Client) GetHostname() (string, error) {
	result, err := c.Execute(common.CommandGuestGetHostName)
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

// Close closes the connection gracefully
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil // Prevent double close
	}
	// Return buffer to pool
	if c.buffer != nil {
		common.GlobalBufferPool.PutLarge(c.buffer)
		c.buffer = nil // Prevent double return
	}
}

// consumeBanner reads and discards the QMP banner
func (c *Client) consumeBanner() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil || c.buffer == nil {
		return // Connection already closed
	}

	// Set a short timeout for banner reading
	common.GlobalTimeManager.SetReadDeadline(c.conn, 1 * time.Second)
	defer common.GlobalTimeManager.ClearDeadlines(c.conn) // Clear timeout

	// Try to read banner, ignore errors
	c.conn.Read(c.buffer)
}

// readResponse reads and parses a QMP response
func (c *Client) readResponse() (map[string]any, error) {
	// Note: This method is called with read lock already held by Execute
	if c.conn == nil || c.buffer == nil {
		return nil, fmt.Errorf("client connection is closed")
	}

	// Set reasonable timeout
	common.GlobalTimeManager.SetReadDeadline(c.conn, 10 * time.Second)
	defer common.GlobalTimeManager.ClearDeadlines(c.conn) // Clear timeout

	// Read response
	n, err := c.conn.Read(c.buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse JSON
	var response map[string]any
	if err := json.Unmarshal(c.buffer[:n], &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
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
