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

package qmp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/prevostcorentin/go-qga/internal/errors"
	"github.com/prevostcorentin/go-qga/internal/qmp"
)

const expectedCommandResponse = "777 holy trinity"

type fakeConnection struct{}

func (_ *fakeConnection) Connect(ctx context.Context, _ string) *ConnectionError {
	return nil
}

func (_ *fakeConnection) Send(ctx context.Context, _ []byte) ([]byte, *ConnectionError) {
	response := struct {
		Return *testCommandResponse `json:"return"`
	}{Return: &testCommandResponse{Value: expectedCommandResponse}}
	responseBytes, _ := json.Marshal(response)
	return responseBytes, nil
}

func (fc *fakeConnection) SendAsync(ctx context.Context, bytes []byte) <-chan qmp.AsyncResult {
	ch := make(chan qmp.AsyncResult, 1)
	go func() {
		data, err := fc.Send(ctx, bytes)
		ch <- qmp.AsyncResult{Data: data, Err: err}
	}()
	return ch
}

func (_ *fakeConnection) Close() error {
	return nil
}

type testCommand struct{}
type testCommandArguments struct {
	Argument int `json:"argument"`
}
type testCommandResponse struct {
	Value string `json:"value"`
}

func (_ *testCommand) Execute() string {
	return "test-command"
}

func (_ *testCommand) Arguments() any {
	return &testCommandArguments{}
}

func (_ *testCommand) Response() any {
	return &testCommandResponse{}
}

func TestRunCommand(t *testing.T) {
	executor, err := qmp.NewExecutor(&fakeConnection{})
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	ctx := context.Background()
	response, err := executor.Run(ctx, &testCommand{})
	if err != nil {
		t.Fatalf("while running command: %v", err)
	}
	if response.(*testCommandResponse).Value != expectedCommandResponse {
		t.Errorf(`wrong value "%v" for response. expected "777"`, response.(*testCommandResponse).Value)
	}
}

type testCommandRecursive struct{}
type testCommandArgumentsRecursive struct {
	Self *testCommandArgumentsRecursive `json:"self"`
}

func (_ *testCommandRecursive) Execute() string {
	return "test-command"
}

func (_ *testCommandRecursive) Arguments() any {
	arguments := &testCommandArgumentsRecursive{}
	arguments.Self = arguments
	return arguments
}

func (_ *testCommandRecursive) Response() any {
	return &testCommandResponse{}
}

func TestMarshalFailure(t *testing.T) {
	executor, err := qmp.NewExecutor(&fakeConnection{})
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	failingCommand := &testCommandRecursive{}
	ctx := context.Background()
	_, runErr := executor.Run(ctx, failingCommand)
	if runErr == nil {
		t.Fatal("should have raised an error")
	}
	if runErr.Domain() != CodecDomain {
		t.Errorf(`wrong error domain "%v". expected "%s"`, runErr.Domain(), CodecDomain)
	}
}

type unmarshableResponseConnection struct{}

func (_ *unmarshableResponseConnection) Connect(ctx context.Context, _ string) *ConnectionError {
	return nil
}

func (_ *unmarshableResponseConnection) Send(ctx context.Context, _ []byte) ([]byte, *ConnectionError) {
	return []byte("i am no object"), nil
}

func (_ *unmarshableResponseConnection) Close() error {
	return nil
}

func (uc *unmarshableResponseConnection) SendAsync(ctx context.Context, bytes []byte) <-chan qmp.AsyncResult {
	ch := make(chan qmp.AsyncResult, 1)
	go func() {
		data, err := uc.Send(ctx, bytes)
		ch <- qmp.AsyncResult{Data: data, Err: err}
	}()
	return ch
}

func TestUnmarshalResponseTopLevelFailure(t *testing.T) {
	executor, err := qmp.NewExecutor(&unmarshableResponseConnection{})
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	failingCommand := &testCommand{}
	var runErr QgaError
	ctx := context.Background()
	if _, runErr = executor.Run(ctx, failingCommand); runErr == nil {
		t.Errorf("there should have been an error here")
	}
	if runErr.Domain() != CodecDomain {
		t.Errorf(`wrong error domain "%v". expected "%s"`, runErr.Domain(), CodecDomain)
	}
	if runErr.Kind() != string(Unmarshal) {
		t.Errorf(`wrong error kind "%v". expected "%s"`, runErr.Kind(), Unmarshal)
	}
}

type childUnmarshalFailureCommand struct{}

func (_ *childUnmarshalFailureCommand) Execute() string {
	return "unmarshal-failure"
}

func (_ *childUnmarshalFailureCommand) Arguments() any {
	return &testCommandArguments{}
}

func (_ *childUnmarshalFailureCommand) Response() any {
	return &childUnmarshableResponse{}
}

type childUnmarshableResponse struct {
	Value int `json:"value"`
}

func TestUnmarshalResponseReturnsFailure(t *testing.T) {
	executor, err := qmp.NewExecutor(&fakeConnection{})
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	failingCommand := &childUnmarshalFailureCommand{}
	var runErr QgaError
	ctx := context.Background()
	if _, runErr = executor.Run(ctx, failingCommand); runErr == nil {
		t.Errorf("there should have been an error here")
	}
	if runErr.Domain() != CodecDomain {
		t.Errorf(`wrong error domain "%v". expected "%s"`, runErr.Domain(), CodecDomain)
	}
	if runErr.Kind() != string(Unmarshal) {
		t.Errorf(`wrong error kind "%v". expected "%s"`, runErr.Kind(), Unmarshal)
	}
}

type sendFailureConnection struct{}

func (_ *sendFailureConnection) Connect(ctx context.Context, _ string) *ConnectionError {
	return nil
}

func (_ *sendFailureConnection) Send(ctx context.Context, _ []byte) ([]byte, *ConnectionError) {
	return nil, NewConnectionError(fmt.Errorf("I am designed to fail"), SendErrorKind)
}

func (sfc *sendFailureConnection) SendAsync(ctx context.Context, bytes []byte) <-chan qmp.AsyncResult {
	ch := make(chan qmp.AsyncResult, 1)
	go func() {
		data, err := sfc.Send(ctx, bytes)
		ch <- qmp.AsyncResult{Data: data, Err: err}
	}()
	return ch
}

func (_ *sendFailureConnection) Close() error {
	return nil
}

func TestConnectionSendFailure(t *testing.T) {
	executor, err := qmp.NewExecutor(&sendFailureConnection{})
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	failingCommand := &testCommand{}
	var runErr QgaError
	ctx := context.Background()
	if _, runErr = executor.Run(ctx, failingCommand); runErr == nil {
		t.Errorf("there should have been an error here")
	}
	if runErr.Domain() != ConnectionDomain {
		t.Errorf(`wrong error domain "%v". expected "%s"`, runErr.Domain(), ConnectionDomain)
	}
	if runErr.Kind() != string(SendErrorKind) {
		t.Errorf(`wrong error kind "%v". expected "%s"`, runErr.Kind(), SendErrorKind)
	}
}

type missingReturnsConnection struct{}

func (_ *missingReturnsConnection) Connect(ctx context.Context, _ string) *ConnectionError {
	return nil
}

func (_ *missingReturnsConnection) Send(ctx context.Context, _ []byte) ([]byte, *ConnectionError) {
	response := struct {
		MissingReturn *testCommandResponse `json:"missing_return"`
	}{MissingReturn: &testCommandResponse{Value: expectedCommandResponse}}
	responseBytes, _ := json.Marshal(response)
	return responseBytes, nil
}

func (mrc *missingReturnsConnection) SendAsync(ctx context.Context, bytes []byte) <-chan qmp.AsyncResult {
	ch := make(chan qmp.AsyncResult, 1)
	go func() {
		data, err := mrc.Send(ctx, bytes)
		ch <- qmp.AsyncResult{Data: data, Err: err}
	}()
	return ch
}

func (_ *missingReturnsConnection) Close() error {
	return nil
}

func TestNoReturnsFailure(t *testing.T) {
	executor, err := qmp.NewExecutor(&missingReturnsConnection{})
	if err != nil {
		t.Fatalf("creating executor: %v", err)
	}
	failingCommand := &testCommand{}
	var runErr QgaError
	ctx := context.Background()
	if _, runErr = executor.Run(ctx, failingCommand); runErr == nil {
		t.Errorf("there should have been an error here")
	}
	if runErr.Domain() != CodecDomain {
		t.Errorf(`wrong error domain "%v". expected "%s"`, runErr.Domain(), CodecDomain)
	}
	if runErr.Kind() != string(Key) {
		t.Errorf(`wrong error kind "%v". expected "%s"`, runErr.Kind(), Key)
	}
}
