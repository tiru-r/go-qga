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
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryPool provides optimized memory management with different pool sizes
type MemoryPool struct {
	small    sync.Pool
	standard sync.Pool
	large    sync.Pool
	xlarge   sync.Pool
	stats    MemoryPoolStats
}

// MemoryPoolStats tracks pool usage statistics
type MemoryPoolStats struct {
	SmallGets     int64
	SmallPuts     int64
	StandardGets  int64
	StandardPuts  int64
	LargeGets     int64
	LargePuts     int64
	XLargeGets    int64
	XLargePuts    int64
}

// NewMemoryPool creates an optimized memory pool
func NewMemoryPool() *MemoryPool {
	return &MemoryPool{
		small: sync.Pool{
			New: func() any { return make([]byte, SmallBufferSize) },
		},
		standard: sync.Pool{
			New: func() any { return make([]byte, StandardBufferSize) },
		},
		large: sync.Pool{
			New: func() any { return make([]byte, LargeBufferSize) },
		},
		xlarge: sync.Pool{
			New: func() any { return make([]byte, XLargeBufferSize) },
		},
	}
}

// GetSmall gets a small buffer with stats tracking
func (p *MemoryPool) GetSmall() []byte {
	atomic.AddInt64(&p.stats.SmallGets, 1)
	return p.small.Get().([]byte)
}

// PutSmall returns a small buffer with stats tracking
func (p *MemoryPool) PutSmall(buf []byte) {
	if cap(buf) >= SmallBufferSize {
		atomic.AddInt64(&p.stats.SmallPuts, 1)
		buf = buf[:SmallBufferSize]
		p.small.Put(buf)
	}
}

// GetStandard gets a standard buffer with stats tracking
func (p *MemoryPool) GetStandard() []byte {
	atomic.AddInt64(&p.stats.StandardGets, 1)
	return p.standard.Get().([]byte)
}

// PutStandard returns a standard buffer with stats tracking
func (p *MemoryPool) PutStandard(buf []byte) {
	if cap(buf) >= StandardBufferSize {
		atomic.AddInt64(&p.stats.StandardPuts, 1)
		buf = buf[:StandardBufferSize]
		p.standard.Put(buf)
	}
}

// GetLarge gets a large buffer with stats tracking
func (p *MemoryPool) GetLarge() []byte {
	atomic.AddInt64(&p.stats.LargeGets, 1)
	return p.large.Get().([]byte)
}

// PutLarge returns a large buffer with stats tracking
func (p *MemoryPool) PutLarge(buf []byte) {
	if cap(buf) >= LargeBufferSize {
		atomic.AddInt64(&p.stats.LargePuts, 1)
		buf = buf[:LargeBufferSize]
		p.large.Put(buf)
	}
}

// GetXLarge gets an extra large buffer with stats tracking
func (p *MemoryPool) GetXLarge() []byte {
	atomic.AddInt64(&p.stats.XLargeGets, 1)
	return p.xlarge.Get().([]byte)
}

// PutXLarge returns an extra large buffer with stats tracking
func (p *MemoryPool) PutXLarge(buf []byte) {
	if cap(buf) >= XLargeBufferSize {
		atomic.AddInt64(&p.stats.XLargePuts, 1)
		buf = buf[:XLargeBufferSize]
		p.xlarge.Put(buf)
	}
}

// GetStats returns current pool statistics
func (p *MemoryPool) GetStats() MemoryPoolStats {
	return MemoryPoolStats{
		SmallGets:     atomic.LoadInt64(&p.stats.SmallGets),
		SmallPuts:     atomic.LoadInt64(&p.stats.SmallPuts),
		StandardGets:  atomic.LoadInt64(&p.stats.StandardGets),
		StandardPuts:  atomic.LoadInt64(&p.stats.StandardPuts),
		LargeGets:     atomic.LoadInt64(&p.stats.LargeGets),
		LargePuts:     atomic.LoadInt64(&p.stats.LargePuts),
		XLargeGets:    atomic.LoadInt64(&p.stats.XLargeGets),
		XLargePuts:    atomic.LoadInt64(&p.stats.XLargePuts),
	}
}

// WorkerPoolOptimized provides an optimized worker pool with adaptive sizing
type WorkerPoolOptimized struct {
	workers     int
	maxWorkers  int
	taskQueue   chan Task
	results     chan any
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	once        sync.Once
	activeCount int64
	stats       WorkerPoolStats
}

// WorkerPoolStats tracks worker pool performance
type WorkerPoolStats struct {
	TasksSubmitted   int64
	TasksCompleted   int64
	TasksActive      int64
	WorkersActive    int64
	WorkersTotal     int64
	QueueLength      int
	ResultsLength    int
}

// NewWorkerPoolOptimized creates an optimized worker pool
func NewWorkerPoolOptimized(workers, maxWorkers, bufferSize int) *WorkerPoolOptimized {
	ctx, cancel := context.WithCancel(context.Background())
	
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU() * 2
	}
	if workers > maxWorkers {
		workers = maxWorkers
	}
	
	wp := &WorkerPoolOptimized{
		workers:    workers,
		maxWorkers: maxWorkers,
		taskQueue:  make(chan Task, bufferSize),
		results:    make(chan any, bufferSize),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	wp.start()
	return wp
}

// Submit submits a task with load balancing
func (wp *WorkerPoolOptimized) Submit(task Task) {
	atomic.AddInt64(&wp.stats.TasksSubmitted, 1)
	
	select {
	case wp.taskQueue <- task:
		// Task queued successfully
	case <-wp.ctx.Done():
		// Pool is closed
		return
	default:
		// Queue is full, try to scale up workers
		wp.scaleUpIfNeeded()
		
		// Try again with timeout
		ctx, cancel := context.WithTimeout(wp.ctx, 100*time.Millisecond)
		defer cancel()
		
		select {
		case wp.taskQueue <- task:
		case <-ctx.Done():
			// Drop task if can't queue within timeout
		}
	}
}

// scaleUpIfNeeded adds workers if queue is under pressure
func (wp *WorkerPoolOptimized) scaleUpIfNeeded() {
	currentWorkers := int(atomic.LoadInt64(&wp.stats.WorkersTotal))
	queueLen := len(wp.taskQueue)
	
	// Scale up if queue is > 75% full and we haven't reached max workers
	if queueLen > cap(wp.taskQueue)*3/4 && currentWorkers < wp.maxWorkers {
		wp.wg.Add(1)
		atomic.AddInt64(&wp.stats.WorkersTotal, 1)
		go wp.worker()
	}
}

// worker is the optimized worker goroutine function
func (wp *WorkerPoolOptimized) worker() {
	defer wp.wg.Done()
	defer atomic.AddInt64(&wp.stats.WorkersTotal, -1)
	
	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			
			atomic.AddInt64(&wp.stats.WorkersActive, 1)
			atomic.AddInt64(&wp.stats.TasksActive, 1)
			
			result := task(wp.ctx)
			
			atomic.AddInt64(&wp.stats.WorkersActive, -1)
			atomic.AddInt64(&wp.stats.TasksActive, -1)
			atomic.AddInt64(&wp.stats.TasksCompleted, 1)
			
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

// start initializes the worker goroutines
func (wp *WorkerPoolOptimized) start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		atomic.AddInt64(&wp.stats.WorkersTotal, 1)
		go wp.worker()
	}
}

// Results returns the results channel
func (wp *WorkerPoolOptimized) Results() <-chan any {
	return wp.results
}

// GetStats returns current worker pool statistics
func (wp *WorkerPoolOptimized) GetStats() WorkerPoolStats {
	return WorkerPoolStats{
		TasksSubmitted:   atomic.LoadInt64(&wp.stats.TasksSubmitted),
		TasksCompleted:   atomic.LoadInt64(&wp.stats.TasksCompleted),
		TasksActive:      atomic.LoadInt64(&wp.stats.TasksActive),
		WorkersActive:    atomic.LoadInt64(&wp.stats.WorkersActive),
		WorkersTotal:     atomic.LoadInt64(&wp.stats.WorkersTotal),
		QueueLength:      len(wp.taskQueue),
		ResultsLength:    len(wp.results),
	}
}

// Close shuts down the worker pool gracefully
func (wp *WorkerPoolOptimized) Close() {
	wp.once.Do(func() {
		close(wp.taskQueue)
		wp.wg.Wait()
		close(wp.results)
		wp.cancel()
	})
}

// PerformanceMonitor provides runtime performance monitoring
type PerformanceMonitor struct {
	startTime    time.Time
	samples      []PerformanceSample
	mu           sync.RWMutex
	maxSamples   int
	sampleTicker *time.Ticker
	stopCh       chan struct{}
}

// PerformanceSample represents a performance measurement sample
type PerformanceSample struct {
	Timestamp     time.Time
	Goroutines    int
	MemAlloc      uint64
	MemSys        uint64
	GCPauses      uint64
	HeapObjects   uint64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(sampleInterval time.Duration, maxSamples int) *PerformanceMonitor {
	pm := &PerformanceMonitor{
		startTime:  time.Now(),
		samples:    make([]PerformanceSample, 0, maxSamples),
		maxSamples: maxSamples,
		stopCh:     make(chan struct{}),
	}
	
	pm.sampleTicker = time.NewTicker(sampleInterval)
	go pm.monitor()
	
	return pm
}

// monitor continuously collects performance samples
func (pm *PerformanceMonitor) monitor() {
	defer pm.sampleTicker.Stop()
	
	for {
		select {
		case <-pm.sampleTicker.C:
			pm.collectSample()
		case <-pm.stopCh:
			return
		}
	}
}

// collectSample collects a performance sample
func (pm *PerformanceMonitor) collectSample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	sample := PerformanceSample{
		Timestamp:     time.Now(),
		Goroutines:    runtime.NumGoroutine(),
		MemAlloc:      m.Alloc,
		MemSys:        m.Sys,
		GCPauses:      m.PauseTotalNs,
		HeapObjects:   m.HeapObjects,
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if len(pm.samples) >= pm.maxSamples {
		// Remove oldest sample
		copy(pm.samples, pm.samples[1:])
		pm.samples = pm.samples[:len(pm.samples)-1]
	}
	
	pm.samples = append(pm.samples, sample)
}

// GetLatestSample returns the most recent performance sample
func (pm *PerformanceMonitor) GetLatestSample() PerformanceSample {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	if len(pm.samples) == 0 {
		return PerformanceSample{}
	}
	
	return pm.samples[len(pm.samples)-1]
}

// GetSamples returns all collected samples
func (pm *PerformanceMonitor) GetSamples() []PerformanceSample {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	result := make([]PerformanceSample, len(pm.samples))
	copy(result, pm.samples)
	return result
}

// GetUptime returns the monitor uptime
func (pm *PerformanceMonitor) GetUptime() time.Duration {
	return time.Since(pm.startTime)
}

// Stop stops the performance monitor
func (pm *PerformanceMonitor) Stop() {
	close(pm.stopCh)
}