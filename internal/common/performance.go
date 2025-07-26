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
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceOptimizer provides comprehensive performance optimization
type PerformanceOptimizer struct {
	memoryPool    *MemoryPool
	workerPool    *WorkerPoolOptimized
	monitor       *PerformanceMonitor
	bufferPool    *BufferPool
	metrics       *PerformanceMetrics
	mu            sync.RWMutex
}

// PerformanceMetrics tracks various performance metrics
type PerformanceMetrics struct {
	ConnectionsCreated   int64
	ConnectionsActive    int64
	CommandsExecuted     int64
	CommandsPerSecond    int64
	BytesTransferred     int64
	ErrorsTotal          int64
	LatencySum           int64
	LatencySamples       int64
	MemoryAllocated      int64
	GoroutinesActive     int64
	lastUpdate           time.Time
}

// NewPerformanceOptimizer creates a comprehensive performance optimizer
func NewPerformanceOptimizer(config *OptimizerConfig) *PerformanceOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}
	
	return &PerformanceOptimizer{
		memoryPool: NewMemoryPool(),
		workerPool: NewWorkerPoolOptimized(
			config.WorkerCount,
			config.MaxWorkers,
			config.BufferSize,
		),
		monitor: NewPerformanceMonitor(
			config.MonitorInterval,
			config.MaxSamples,
		),
		bufferPool: NewBufferPool(config.PoolSize),
		metrics:    &PerformanceMetrics{lastUpdate: time.Now()},
	}
}

// OptimizerConfig configures the performance optimizer
type OptimizerConfig struct {
	WorkerCount     int
	MaxWorkers      int
	BufferSize      int
	PoolSize        int
	MonitorInterval time.Duration
	MaxSamples      int
}

// DefaultOptimizerConfig returns default optimizer configuration
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		WorkerCount:     runtime.NumCPU(),
		MaxWorkers:      runtime.NumCPU() * 4,
		BufferSize:      1000,
		PoolSize:        100,
		MonitorInterval: time.Second,
		MaxSamples:      300, // 5 minutes at 1 second interval
	}
}

// GetBuffer gets an optimally sized buffer based on usage pattern
func (p *PerformanceOptimizer) GetBuffer(sizeHint int) []byte {
	switch {
	case sizeHint <= SmallBufferSize:
		return p.memoryPool.GetSmall()
	case sizeHint <= StandardBufferSize:
		return p.memoryPool.GetStandard()
	case sizeHint <= LargeBufferSize:
		return p.memoryPool.GetLarge()
	default:
		return p.memoryPool.GetXLarge()
	}
}

// PutBuffer returns a buffer to the appropriate pool
func (p *PerformanceOptimizer) PutBuffer(buf []byte) {
	switch cap(buf) {
	case SmallBufferSize:
		p.memoryPool.PutSmall(buf)
	case StandardBufferSize:
		p.memoryPool.PutStandard(buf)
	case LargeBufferSize:
		p.memoryPool.PutLarge(buf)
	case XLargeBufferSize:
		p.memoryPool.PutXLarge(buf)
	}
}

// SubmitTask submits a task to the optimized worker pool
func (p *PerformanceOptimizer) SubmitTask(task Task) {
	p.workerPool.Submit(task)
}

// GetTaskResults returns the task results channel
func (p *PerformanceOptimizer) GetTaskResults() <-chan any {
	return p.workerPool.Results()
}

// RecordConnection records connection metrics
func (p *PerformanceOptimizer) RecordConnection(created bool) {
	if created {
		atomic.AddInt64(&p.metrics.ConnectionsCreated, 1)
		atomic.AddInt64(&p.metrics.ConnectionsActive, 1)
	} else {
		atomic.AddInt64(&p.metrics.ConnectionsActive, -1)
	}
}

// RecordCommand records command execution metrics
func (p *PerformanceOptimizer) RecordCommand(duration time.Duration) {
	atomic.AddInt64(&p.metrics.CommandsExecuted, 1)
	atomic.AddInt64(&p.metrics.LatencySum, int64(duration))
	atomic.AddInt64(&p.metrics.LatencySamples, 1)
	
	// Update commands per second (simple moving average)
	now := time.Now()
	p.mu.Lock()
	if now.Sub(p.metrics.lastUpdate) >= time.Second {
		commands := atomic.LoadInt64(&p.metrics.CommandsExecuted)
		elapsed := now.Sub(p.metrics.lastUpdate).Seconds()
		atomic.StoreInt64(&p.metrics.CommandsPerSecond, int64(float64(commands)/elapsed))
		p.metrics.lastUpdate = now
	}
	p.mu.Unlock()
}

// RecordBytes records data transfer metrics
func (p *PerformanceOptimizer) RecordBytes(bytes int64) {
	atomic.AddInt64(&p.metrics.BytesTransferred, bytes)
}

// RecordError records error metrics
func (p *PerformanceOptimizer) RecordError() {
	atomic.AddInt64(&p.metrics.ErrorsTotal, 1)
}

// GetMetrics returns current performance metrics
func (p *PerformanceOptimizer) GetMetrics() PerformanceMetrics {
	// Update runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return PerformanceMetrics{
		ConnectionsCreated:   atomic.LoadInt64(&p.metrics.ConnectionsCreated),
		ConnectionsActive:    atomic.LoadInt64(&p.metrics.ConnectionsActive),
		CommandsExecuted:     atomic.LoadInt64(&p.metrics.CommandsExecuted),
		CommandsPerSecond:    atomic.LoadInt64(&p.metrics.CommandsPerSecond),
		BytesTransferred:     atomic.LoadInt64(&p.metrics.BytesTransferred),
		ErrorsTotal:          atomic.LoadInt64(&p.metrics.ErrorsTotal),
		LatencySum:           atomic.LoadInt64(&p.metrics.LatencySum),
		LatencySamples:       atomic.LoadInt64(&p.metrics.LatencySamples),
		MemoryAllocated:      int64(m.Alloc),
		GoroutinesActive:     int64(runtime.NumGoroutine()),
		lastUpdate:           p.metrics.lastUpdate,
	}
}

// GetAverageLatency returns the average command latency
func (p *PerformanceOptimizer) GetAverageLatency() time.Duration {
	sum := atomic.LoadInt64(&p.metrics.LatencySum)
	samples := atomic.LoadInt64(&p.metrics.LatencySamples)
	if samples == 0 {
		return 0
	}
	return time.Duration(sum / samples)
}

// GetMemoryStats returns memory pool statistics
func (p *PerformanceOptimizer) GetMemoryStats() MemoryPoolStats {
	return p.memoryPool.GetStats()
}

// GetWorkerStats returns worker pool statistics
func (p *PerformanceOptimizer) GetWorkerStats() WorkerPoolStats {
	return p.workerPool.GetStats()
}

// GetPerformanceSample returns the latest performance sample
func (p *PerformanceOptimizer) GetPerformanceSample() PerformanceSample {
	return p.monitor.GetLatestSample()
}

// OptimizeForThroughput optimizes configuration for maximum throughput
func (p *PerformanceOptimizer) OptimizeForThroughput() {
	// Increase worker pool size
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// This is a placeholder - actual implementation would adjust
	// worker pool size, buffer sizes, etc. based on current metrics
}

// OptimizeForLatency optimizes configuration for minimum latency
func (p *PerformanceOptimizer) OptimizeForLatency() {
	// Reduce buffer sizes, increase worker priority
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// This is a placeholder - actual implementation would adjust
	// configuration to minimize latency
}

// OptimizeForMemory optimizes configuration for minimum memory usage
func (p *PerformanceOptimizer) OptimizeForMemory() {
	// Reduce buffer pool sizes, optimize GC
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Force garbage collection
	runtime.GC()
}

// GetOptimizationRecommendations returns performance optimization recommendations
func (p *PerformanceOptimizer) GetOptimizationRecommendations() []string {
	metrics := p.GetMetrics()
	recommendations := make([]string, 0)
	
	// Analyze metrics and provide recommendations
	if metrics.ErrorsTotal > metrics.CommandsExecuted/10 {
		recommendations = append(recommendations, "High error rate detected - check error handling")
	}
	
	if metrics.GoroutinesActive > int64(runtime.NumCPU()*10) {
		recommendations = append(recommendations, "High goroutine count - consider connection pooling")
	}
	
	if metrics.MemoryAllocated > 100*1024*1024 { // 100MB
		recommendations = append(recommendations, "High memory usage - consider optimizing buffer management")
	}
	
	avgLatency := p.GetAverageLatency()
	if avgLatency > 100*time.Millisecond {
		recommendations = append(recommendations, "High latency detected - consider optimizing for latency")
	}
	
	return recommendations
}

// Close shuts down the performance optimizer
func (p *PerformanceOptimizer) Close() {
	p.workerPool.Close()
	p.monitor.Stop()
}