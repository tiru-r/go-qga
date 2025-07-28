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
	"sync"
	"time"

	"github.com/prevostcorentin/go-qga/internal/common"
)

const (
	DefaultWorkerCount  = 5   // Default number of workers in pool
	DefaultBufferSize   = 100 // Default buffer size for worker pool
	ChannelBufferSize   = 1   // Buffer size for result channels
)

type CommandExecutor interface {
	Run(ctx context.Context, command Command) (any, error)
	RunAsync(ctx context.Context, command Command) <-chan ExecutorResult
	Close() error
}

type ExecutorResult struct {
	Data any
	Err  error
}

type commandExecutor struct {
	*common.BaseState                // 8 bytes (pointer)
	connection     Connection        // 16 bytes (interface)  
	workerPool     *common.WorkerPool // 8 bytes (pointer)
	channelPool    *sync.Pool        // 8 bytes (pointer)
}

// Request represents a reusable QMP request structure
type Request struct {
	Execute   string `json:"execute"`
	Arguments any    `json:"arguments,omitempty"`
}

func NewExecutor(connection Connection) (CommandExecutor, error) {
	if connection == nil {
		return nil, fmt.Errorf("connection is nil")
	}
	return &commandExecutor{
		BaseState:  common.NewBaseState(),
		connection: connection,
		workerPool: common.NewWorkerPool(DefaultWorkerCount, DefaultBufferSize),
		channelPool: &sync.Pool{
			New: func() any {
				return make(chan ExecutorResult, ChannelBufferSize)
			},
		},
	}, nil
}

func (e *commandExecutor) Run(ctx context.Context, command Command) (any, error) {
	e.RLock()
	defer e.RUnlock()

	if e.IsClosed() {
		return nil, fmt.Errorf("executor is closed")
	}

	if command == nil {
		return nil, fmt.Errorf("command is nil")
	}

	marshalled := Request{
		Execute:   command.Execute(),
		Arguments: command.Arguments(),
	}
	marshalledBytes, marshalErr := json.Marshal(marshalled)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal error: %w", marshalErr)
	}
	// Pre-allocate slice to avoid reallocation when appending line terminator
	finalBytes := make([]byte, len(marshalledBytes)+1)
	copy(finalBytes, marshalledBytes)
	finalBytes[len(marshalledBytes)] = common.LineTerminator
	marshalledBytes = finalBytes

	responseBytes, sendErr := e.connection.Send(ctx, marshalledBytes)
	if sendErr != nil {
		return nil, sendErr
	}
	return e.unmarshalCommandResponse(responseBytes, command)
}

func (e *commandExecutor) RunAsync(ctx context.Context, command Command) <-chan ExecutorResult {
	resultCh := e.channelPool.Get().(chan ExecutorResult)
	
	// Drain any stale data from pooled channel
	select {
	case <-resultCh:
	default:
	}

	e.RLock()
	defer e.RUnlock()

	if e.IsClosed() {
		resultCh <- ExecutorResult{
			Data: nil,
			Err:  fmt.Errorf("executor is closed"),
		}
		return resultCh
	}

	if command == nil {
		resultCh <- ExecutorResult{
			Data: nil,
			Err:  fmt.Errorf("command is nil"),
		}
		return resultCh
	}

	task := func(taskCtx context.Context) any {
		marshalled := Request{
			Execute:   command.Execute(),
			Arguments: command.Arguments(),
		}
		marshalledBytes, marshalErr := json.Marshal(marshalled)
		if marshalErr != nil {
			return ExecutorResult{
				Data: nil,
				Err:  fmt.Errorf("marshal error: %w", marshalErr),
			}
		}
		// Efficiently append line terminator
		marshalledBytes = append(marshalledBytes, common.LineTerminator)

		// Use async connection send
		asyncResult := e.connection.SendAsync(taskCtx, marshalledBytes)
		select {
		case result := <-asyncResult:
			if result.Err != nil {
				return ExecutorResult{
					Data: nil,
					Err:  result.Err,
				}
			}
			data, unmarshalErr := e.unmarshalCommandResponse(result.Data, command)
			return ExecutorResult{
				Data: data,
				Err:  unmarshalErr,
			}
		case <-taskCtx.Done():
			return ExecutorResult{
				Data: nil,
				Err:  fmt.Errorf("context error: %w", taskCtx.Err()),
			}
		}
	}

	e.workerPool.Submit(task)

	go func() {
		defer func() {
			// Return the pooled channel after use
			e.channelPool.Put(resultCh)
		}()
		
		select {
		case result := <-e.workerPool.Results():
			if execResult, ok := result.(ExecutorResult); ok {
				select {
				case resultCh <- execResult:
				case <-time.After(1 * time.Second):
					// Channel blocked, exit gracefully
				case <-ctx.Done():
					// Context cancelled
				}
			} else {
				select {
				case resultCh <- ExecutorResult{
					Data: nil,
					Err:  fmt.Errorf("unexpected result type"),
				}:
				case <-time.After(1 * time.Second):
					// Channel blocked, exit gracefully
				case <-ctx.Done():
					// Context cancelled
				}
			}
		case <-ctx.Done():
			select {
			case resultCh <- ExecutorResult{
				Data: nil,
				Err:  fmt.Errorf("context cancelled: %w", ctx.Err()),
			}:
			case <-time.After(1 * time.Second):
				// Channel blocked, exit gracefully
			}
		}
	}()

	return resultCh
}

func (e *commandExecutor) Close() error {
	e.Lock()
	defer e.Unlock()

	if !e.CloseWithLock() {
		return nil // already closed
	}

	e.workerPool.Close()
	return e.connection.Close()
}

func (e *commandExecutor) unmarshalCommandResponse(bytes []byte, command Command) (any, error) {
	typedResponse := command.Response()
	var root map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &root); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	if raw, ok := root[common.JSONFieldReturn]; ok {
		if err := json.Unmarshal(raw, &typedResponse); err != nil {
			return nil, fmt.Errorf("unmarshal error: %w", err)
		}
		return typedResponse, nil
	}
	return nil, fmt.Errorf("missing return value in response")
}
