package workers

import (
	"context"
	"runtime"
	"time"
)

// CPUWorker is optimized for CPU-intensive tasks
type CPUWorker struct {
	*BasicWorker
	cpuAffinity   int
	batchSize     int
	processorInfo ProcessorInfo
}

// ProcessorInfo contains CPU-specific information
type ProcessorInfo struct {
	CoreID       int
	CacheSize    int64
	Frequency    float64
	Utilization  float64
}

// CPUWorkItem represents CPU-intensive work
type CPUWorkItem struct {
	ctx              context.Context
	priority         int
	estimatedDuration time.Duration
	computeFunc      func() error
	batchable        bool
	cacheAware       bool
}

// NewCPUWorker creates a new CPU-optimized worker
func NewCPUWorker(id string, queueSize int, cpuAffinity int) *CPUWorker {
	basicWorker := NewBasicWorker(id, queueSize)
	return &CPUWorker{
		BasicWorker: basicWorker,
		cpuAffinity: cpuAffinity,
		batchSize:   10, // Default batch size
		processorInfo: ProcessorInfo{
			CoreID: cpuAffinity,
		},
	}
}

// Start begins the CPU worker with CPU affinity settings
func (w *CPUWorker) Start(ctx context.Context) error {
	// Set CPU affinity if specified
	if w.cpuAffinity >= 0 {
		w.setCPUAffinity()
	}
	
	return w.BasicWorker.Start(ctx)
}

// setCPUAffinity attempts to set CPU affinity (platform-specific)
func (w *CPUWorker) setCPUAffinity() {
	// Note: This is a simplified implementation
	// In a real implementation, you would use platform-specific syscalls
	runtime.LockOSThread()
}

// SubmitCPUWork submits CPU-intensive work
func (w *CPUWorker) SubmitCPUWork(computeFunc func() error, priority int, estimatedDuration time.Duration) error {
	workItem := &CPUWorkItem{
		ctx:               context.Background(),
		priority:          priority,
		estimatedDuration: estimatedDuration,
		computeFunc:       computeFunc,
		batchable:         false,
		cacheAware:        true,
	}
	
	return w.Submit(workItem)
}

// SubmitBatchWork submits batchable CPU work
func (w *CPUWorker) SubmitBatchWork(computeFuncs []func() error, priority int) error {
	for _, computeFunc := range computeFuncs {
		workItem := &CPUWorkItem{
			ctx:               context.Background(),
			priority:          priority,
			estimatedDuration: time.Millisecond * 100, // Estimate
			computeFunc:       computeFunc,
			batchable:         true,
			cacheAware:        true,
		}
		
		if err := w.Submit(workItem); err != nil {
			return err
		}
	}
	return nil
}

// processWork overrides basic worker to optimize for CPU tasks
func (w *CPUWorker) processWork(work WorkItem) {
	start := time.Now()
	
	w.statsMu.Lock()
	w.stats.TasksInProgress++
	w.statsMu.Unlock()
	
	// CPU-specific optimizations
	w.optimizeForCPU()
	
	err := work.Execute()
	
	duration := time.Since(start)
	
	w.statsMu.Lock()
	w.stats.TasksInProgress--
	w.stats.TasksProcessed++
	w.stats.TotalExecutionTime += duration
	w.stats.AverageExecutionTime = w.stats.TotalExecutionTime / time.Duration(w.stats.TasksProcessed)
	w.stats.LastActivity = time.Now()
	
	if err != nil {
		w.stats.ErrorCount++
	}
	
	// Update CPU utilization
	w.updateCPUUtilization(duration)
	
	w.statsMu.Unlock()
}

// optimizeForCPU applies CPU-specific optimizations
func (w *CPUWorker) optimizeForCPU() {
	// Disable GC during CPU-intensive work (if appropriate)
	// This is a trade-off and should be used carefully
	// runtime.GC()
	
	// Ensure we're running on the correct CPU core
	if w.cpuAffinity >= 0 {
		runtime.LockOSThread()
	}
}

// updateCPUUtilization updates CPU utilization metrics
func (w *CPUWorker) updateCPUUtilization(executionTime time.Duration) {
	// Simplified CPU utilization calculation
	// In a real implementation, you would use more sophisticated metrics
	w.processorInfo.Utilization = float64(executionTime) / float64(time.Second)
	if w.processorInfo.Utilization > 1.0 {
		w.processorInfo.Utilization = 1.0
	}
}

// GetCPUStats returns CPU-specific statistics
func (w *CPUWorker) GetCPUStats() CPUWorkerStats {
	w.statsMu.RLock()
	defer w.statsMu.RUnlock()
	
	return CPUWorkerStats{
		WorkerStats:   w.stats,
		ProcessorInfo: w.processorInfo,
		CPUAffinity:   w.cpuAffinity,
		BatchSize:     w.batchSize,
	}
}

// SetBatchSize configures the batch processing size
func (w *CPUWorker) SetBatchSize(size int) {
	w.batchSize = size
}

// CPUWorkerStats extends WorkerStats with CPU-specific metrics
type CPUWorkerStats struct {
	WorkerStats
	ProcessorInfo ProcessorInfo
	CPUAffinity   int
	BatchSize     int
}

// Implementation of WorkItem interface for CPUWorkItem
func (c *CPUWorkItem) Execute() error {
	return c.computeFunc()
}

func (c *CPUWorkItem) Priority() int {
	return c.priority
}

func (c *CPUWorkItem) EstimatedDuration() time.Duration {
	return c.estimatedDuration
}

func (c *CPUWorkItem) Context() context.Context {
	return c.ctx
}

// IsBatchable returns whether this work item can be batched
func (c *CPUWorkItem) IsBatchable() bool {
	return c.batchable
}

// IsCacheAware returns whether this work item is cache-aware
func (c *CPUWorkItem) IsCacheAware() bool {
	return c.cacheAware
}

// CPUIntensiveTask creates a CPU-intensive work item for testing
func CPUIntensiveTask(iterations int, priority int) *CPUWorkItem {
	return &CPUWorkItem{
		ctx:      context.Background(),
		priority: priority,
		estimatedDuration: time.Duration(iterations) * time.Microsecond,
		computeFunc: func() error {
			// Simulate CPU-intensive work
			sum := 0
			for i := 0; i < iterations; i++ {
				sum += i * i
			}
			_ = sum // Prevent optimization
			return nil
		},
		batchable:  true,
		cacheAware: true,
	}
}

// MatrixMultiplicationTask creates a matrix multiplication work item
func MatrixMultiplicationTask(size int, priority int) *CPUWorkItem {
	return &CPUWorkItem{
		ctx:      context.Background(),
		priority: priority,
		estimatedDuration: time.Duration(size*size) * time.Microsecond,
		computeFunc: func() error {
			// Simple matrix multiplication
			a := make([][]int, size)
			b := make([][]int, size)
			c := make([][]int, size)
			
			for i := 0; i < size; i++ {
				a[i] = make([]int, size)
				b[i] = make([]int, size)
				c[i] = make([]int, size)
				for j := 0; j < size; j++ {
					a[i][j] = i + j
					b[i][j] = i - j
				}
			}
			
			// Multiply matrices
			for i := 0; i < size; i++ {
				for j := 0; j < size; j++ {
					for k := 0; k < size; k++ {
						c[i][j] += a[i][k] * b[k][j]
					}
				}
			}
			
			_ = c // Prevent optimization
			return nil
		},
		batchable:  false,
		cacheAware: true,
	}
}