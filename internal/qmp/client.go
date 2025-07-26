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
	"net"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

// Connect creates a new optimistic QMP connection (deprecated: use NewClient)
func Connect(socketPath string) common.Result[*Client] {
	return NewClient(socketPath)
}

// Client provides an optimistic, easy-to-use QMP client
type Client struct {
	conn   net.Conn
	path   string
	buffer []byte
}

// NewClient creates a new optimistic QMP connection
func NewClient(socketPath string) common.Result[*Client] {
	// Optimistic: try connection with smart defaults
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return common.Failure[*Client]("failed to connect to %s: %v", socketPath, err)
	}

	client := &Client{
		conn:   conn,
		path:   socketPath,
		buffer: common.GlobalBufferPool.GetLarge(), // Use buffer pool
	}

	// Optimistic banner consumption - ignore errors, just consume
	client.consumeBanner()

	return common.Success(client)
}

// Execute runs a QMP command with optimistic error handling
func (c *Client) Execute(command string, args ...any) common.Result[map[string]any] {
	// Build request with smart defaults
	request := map[string]any{
		common.JsonFieldExecute: command,
	}

	// Add arguments only if provided (optimistic)
	if len(args) > 0 && args[0] != nil {
		request["arguments"] = args[0]
	}

	// Send request with optimistic JSON handling
	jsonData, err := json.Marshal(request)
	if err != nil {
		return common.Failure[map[string]any]("failed to encode request: %v", err)
	}

	// Pre-allocate slice to avoid reallocation when appending line terminator
	data := make([]byte, len(jsonData)+1)
	copy(data, jsonData)
	data[len(jsonData)] = '\n'
	if _, err := c.conn.Write(data); err != nil {
		return common.Failure[map[string]any]("failed to send command: %v", err)
	}

	// Read response with optimistic parsing
	return c.readResponse()
}

// GetHostname provides a simple, optimistic way to get VM hostname
func (c *Client) GetHostname() common.Result[string] {
	result := c.Execute(common.CommandGuestGetHostName)
	if result.IsErr() {
		return common.FailureFrom[string](result.Error())
	}

	response := result.Value()
	if returnData, ok := response[common.JsonFieldReturn].(map[string]any); ok {
		if hostname, ok := returnData[common.JsonFieldName].(string); ok {
			return common.Success(hostname)
		}
	}

	return common.Failure[string]("hostname not found in response")
}

// Close closes the connection gracefully
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	// Return buffer to pool
	if c.buffer != nil {
		common.GlobalBufferPool.PutLarge(c.buffer)
	}
}

// consumeBanner reads and discards the QMP banner (optimistic approach)
func (c *Client) consumeBanner() {
	// Set a short timeout for banner reading
	common.GlobalTimeManager.SetReadDeadline(c.conn, 1 * time.Second)
	defer common.GlobalTimeManager.ClearDeadlines(c.conn) // Clear timeout

	// Optimistically try to read banner, ignore errors
	c.conn.Read(c.buffer)
}

// readResponse reads and parses a QMP response optimistically
func (c *Client) readResponse() common.Result[map[string]any] {
	// Set reasonable timeout
	common.GlobalTimeManager.SetReadDeadline(c.conn, 10 * time.Second)
	defer common.GlobalTimeManager.ClearDeadlines(c.conn) // Clear timeout

	// Read response
	n, err := c.conn.Read(c.buffer)
	if err != nil {
		return common.Failure[map[string]any]("failed to read response: %v", err)
	}

	// Parse JSON optimistically
	var response map[string]any
	if err := json.Unmarshal(c.buffer[:n], &response); err != nil {
		return common.Failure[map[string]any]("failed to parse response: %v", err)
	}

	// Check for QMP errors optimistically
	if errorData, hasError := response["error"]; hasError {
		if errorInfo, ok := errorData.(map[string]any); ok {
			if desc, ok := errorInfo["desc"].(string); ok {
				return common.Failure[map[string]any]("QMP error: %s", desc)
			}
		}
		return common.Failure[map[string]any]("unknown QMP error occurred")
	}

	return common.Success(response)
}
