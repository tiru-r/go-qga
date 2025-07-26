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

package common

import (
	"context"
	"sync"
)

// Task represents a unit of work to be executed by the worker pool
type Task func(context.Context) any

// WorkerPool represents a pool of workers that execute tasks concurrently
// Fields ordered for optimal memory alignment (largest to smallest)
type WorkerPool struct {
	taskQueue chan Task        // 24 bytes
	results   chan any         // 24 bytes  
	ctx       context.Context  // 16 bytes
	cancel    context.CancelFunc // 8 bytes
	wg        sync.WaitGroup   // Variable size (24 bytes)
	once      sync.Once        // 8 bytes
	workers   int              // 8 bytes
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int, bufferSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	wp := &WorkerPool{
		workers:   workers,
		taskQueue: make(chan Task, bufferSize),
		results:   make(chan any, bufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	wp.start()
	return wp
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task Task) {
	select {
	case wp.taskQueue <- task:
	case <-wp.ctx.Done():
		// Pool is closed
	}
}

// Results returns a channel to receive task results
func (wp *WorkerPool) Results() <-chan any {
	return wp.results
}

// Close shuts down the worker pool gracefully
func (wp *WorkerPool) Close() {
	wp.once.Do(func() {
		close(wp.taskQueue)
		wp.wg.Wait()
		close(wp.results)
		wp.cancel()
	})
}

// start initializes the worker goroutines
func (wp *WorkerPool) start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker is the main worker goroutine function
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	
	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			result := task(wp.ctx)
			select {
			case wp.results <- result:
			case <-wp.ctx.Done():
				return
			}
		case <-wp.ctx.Done():
			return
		}
	}
}