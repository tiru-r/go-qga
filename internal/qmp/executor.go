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

	"github.com/prevostcorentin/go-qga/internal/common"
	"github.com/prevostcorentin/go-qga/internal/errors"
)

type CommandExecutor interface {
	Run(ctx context.Context, command Command) (any, errors.QgaError)
	RunAsync(ctx context.Context, command Command) <-chan ExecutorResult
	Close() error
}

type ExecutorResult struct {
	Data any
	Err  errors.QgaError
}

type commandExecutor struct {
	*common.BaseState
	connection     Connection
	workerPool     *common.WorkerPool
	channelPool    *sync.Pool
}

// Request represents a reusable QMP request structure
type Request struct {
	Execute   string `json:"execute"`
	Arguments any    `json:"arguments,omitempty"`
}

func NewExecutor(connection Connection) (CommandExecutor, error) {
	if connection == nil {
		return nil, fmt.Errorf(errors.ConnectionNilMessage)
	}
	return &commandExecutor{
		BaseState:  common.NewBaseState(),
		connection: connection,
		workerPool: common.NewWorkerPool(5, 100), // 5 workers, buffer size 100
		channelPool: &sync.Pool{
			New: func() any {
				return make(chan ExecutorResult, 1)
			},
		},
	}, nil
}

func (e *commandExecutor) Run(ctx context.Context, command Command) (any, errors.QgaError) {
	e.RLock()
	defer e.RUnlock()

	if e.IsClosed() {
		return nil, errors.ErrExecutorClosed
	}

	if command == nil {
		return nil, errors.ErrCommandNil
	}

	marshalled := Request{
		Execute:   command.Execute(),
		Arguments: command.Arguments(),
	}
	marshalledBytes, marshalErr := json.Marshal(marshalled)
	if marshalErr != nil {
		return nil, errors.NewCodecError(marshalErr, errors.Marshal)
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
			Err:  errors.ErrExecutorClosed,
		}
		return resultCh
	}

	if command == nil {
		resultCh <- ExecutorResult{
			Data: nil,
			Err:  errors.ErrCommandNil,
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
				Err:  errors.NewCodecError(marshalErr, errors.Marshal),
			}
		}
		// Pre-allocate slice to avoid reallocation when appending line terminator
		finalBytes := make([]byte, len(marshalledBytes)+1)
		copy(finalBytes, marshalledBytes)
		finalBytes[len(marshalledBytes)] = common.LineTerminator
		marshalledBytes = finalBytes

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
				Err:  errors.NewCodecError(taskCtx.Err(), errors.Type),
			}
		}
	}

	e.workerPool.Submit(task)

	go func() {
		select {
		case result := <-e.workerPool.Results():
			if execResult, ok := result.(ExecutorResult); ok {
				resultCh <- execResult
			} else {
				resultCh <- ExecutorResult{
					Data: nil,
					Err:  errors.NewCodecError(fmt.Errorf("unexpected result type"), errors.Type),
				}
			}
		case <-ctx.Done():
			resultCh <- ExecutorResult{
				Data: nil,
				Err:  errors.NewCodecError(ctx.Err(), errors.Type),
			}
		}
		// Don't close the channel since it's pooled - instead create a wrapper
	}()

	// Create a wrapper channel that handles pooling
	wrapperCh := make(chan ExecutorResult, 1)
	go func() {
		defer func() {
			// Return the pooled channel after use
			e.channelPool.Put(resultCh)
		}()
		
		// Forward the result from pooled channel to wrapper
		result := <-resultCh
		wrapperCh <- result
		close(wrapperCh)
	}()

	return wrapperCh
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

func (e *commandExecutor) unmarshalCommandResponse(bytes []byte, command Command) (any, errors.QgaError) {
	typedResponse := command.Response()
	var root map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &root); err != nil {
		return nil, errors.NewCodecError(err, errors.Unmarshal)
	}
	if raw, ok := root[common.JsonFieldReturn]; ok {
		if err := json.Unmarshal(raw, &typedResponse); err != nil {
			return nil, errors.NewCodecError(err, errors.Unmarshal)
		}
		return typedResponse, nil
	}
	return nil, errors.ErrMissingReturn
}
