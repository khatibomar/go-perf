// simulation/workload_simulator.go
package simulation

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type WorkloadSimulator struct {
	mu       sync.RWMutex
	active   bool
	workload string
	startTime time.Time
	duration time.Duration
	intensity string
	cancel   context.CancelFunc
	stats    WorkloadStats
}

type WorkloadStats struct {
	OperationsPerformed uint64
	MemoryAllocated     uint64
	GoroutinesCreated   uint64
	CPUCycles          uint64
	ErrorsGenerated    uint64
}

type WorkloadStatus struct {
	Active      bool          `json:"active"`
	Workload    string        `json:"workload"`
	StartTime   time.Time     `json:"start_time"`
	Duration    time.Duration `json:"duration"`
	Intensity   string        `json:"intensity"`
	Elapsed     time.Duration `json:"elapsed"`
	Remaining   time.Duration `json:"remaining"`
	Stats       WorkloadStats `json:"stats"`
}

func NewWorkloadSimulator() *WorkloadSimulator {
	return &WorkloadSimulator{
		stats: WorkloadStats{},
	}
}

func (ws *WorkloadSimulator) StartWorkload(workloadType string, duration time.Duration, intensity string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.active {
		return fmt.Errorf("workload already active: %s", ws.workload)
	}

	// Validate workload type
	switch workloadType {
	case "cpu", "memory", "goroutine", "gc", "mixed":
		// Valid workload types
	default:
		return fmt.Errorf("invalid workload type: %s (valid: cpu, memory, goroutine, gc, mixed)", workloadType)
	}

	// Validate intensity
	switch intensity {
	case "low", "medium", "high":
		// Valid intensities
	default:
		return fmt.Errorf("invalid intensity: %s (valid: low, medium, high)", intensity)
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	ws.cancel = cancel
	ws.active = true
	ws.workload = workloadType
	ws.startTime = time.Now()
	ws.duration = duration
	ws.intensity = intensity
	ws.stats = WorkloadStats{} // Reset stats

	go ws.runWorkload(ctx, workloadType, intensity)

	return nil
}

func (ws *WorkloadSimulator) Stop() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.active && ws.cancel != nil {
		ws.cancel()
		ws.active = false
	}
}

func (ws *WorkloadSimulator) GetStatus() WorkloadStatus {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	status := WorkloadStatus{
		Active:    ws.active,
		Workload:  ws.workload,
		StartTime: ws.startTime,
		Duration:  ws.duration,
		Intensity: ws.intensity,
		Stats:     ws.stats,
	}

	if ws.active {
		status.Elapsed = time.Since(ws.startTime)
		status.Remaining = ws.duration - status.Elapsed
		if status.Remaining < 0 {
			status.Remaining = 0
		}
	}

	return status
}

func (ws *WorkloadSimulator) runWorkload(ctx context.Context, workloadType, intensity string) {
	defer func() {
		ws.mu.Lock()
		ws.active = false
		ws.mu.Unlock()
	}()

	switch workloadType {
	case "cpu":
		ws.runCPUWorkload(ctx, intensity)
	case "memory":
		ws.runMemoryWorkload(ctx, intensity)
	case "goroutine":
		ws.runGoroutineWorkload(ctx, intensity)
	case "gc":
		ws.runGCWorkload(ctx, intensity)
	case "mixed":
		ws.runMixedWorkload(ctx, intensity)
	}
}

func (ws *WorkloadSimulator) runCPUWorkload(ctx context.Context, intensity string) {
	var interval time.Duration
	var workDuration time.Duration

	switch intensity {
	case "low":
		interval = 100 * time.Millisecond
		workDuration = 10 * time.Millisecond
	case "medium":
		interval = 50 * time.Millisecond
		workDuration = 25 * time.Millisecond
	case "high":
		interval = 20 * time.Millisecond
		workDuration = 15 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ws.performCPUWork(workDuration)
			atomic.AddUint64(&ws.stats.OperationsPerformed, 1)
			atomic.AddUint64(&ws.stats.CPUCycles, 1)
		}
	}
}

func (ws *WorkloadSimulator) performCPUWork(duration time.Duration) {
	start := time.Now()
	for time.Since(start) < duration {
		// Perform CPU-intensive calculation
		for i := 0; i < 1000; i++ {
			math.Sqrt(float64(i))
		}
	}
}

func (ws *WorkloadSimulator) runMemoryWorkload(ctx context.Context, intensity string) {
	var allocSize int
	var interval time.Duration
	var maxAllocations int

	switch intensity {
	case "low":
		allocSize = 1024 * 1024 // 1MB
		interval = 500 * time.Millisecond
		maxAllocations = 10
	case "medium":
		allocSize = 5 * 1024 * 1024 // 5MB
		interval = 200 * time.Millisecond
		maxAllocations = 20
	case "high":
		allocSize = 10 * 1024 * 1024 // 10MB
		interval = 100 * time.Millisecond
		maxAllocations = 50
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	allocations := make([][]byte, 0, maxAllocations)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Allocate memory
			data := make([]byte, allocSize)
			// Write to memory to ensure allocation
			for i := 0; i < len(data); i += 4096 {
				data[i] = byte(i % 256)
			}

			allocations = append(allocations, data)
			atomic.AddUint64(&ws.stats.MemoryAllocated, uint64(allocSize))
			atomic.AddUint64(&ws.stats.OperationsPerformed, 1)

			// Keep only recent allocations to prevent unlimited growth
			if len(allocations) > maxAllocations {
				allocations = allocations[1:]
			}
		}
	}
}

func (ws *WorkloadSimulator) runGoroutineWorkload(ctx context.Context, intensity string) {
	var goroutineCount int
	var interval time.Duration
	var workDuration time.Duration

	switch intensity {
	case "low":
		goroutineCount = 10
		interval = 1 * time.Second
		workDuration = 100 * time.Millisecond
	case "medium":
		goroutineCount = 50
		interval = 500 * time.Millisecond
		workDuration = 200 * time.Millisecond
	case "high":
		goroutineCount = 200
		interval = 200 * time.Millisecond
		workDuration = 500 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create burst of goroutines
			for i := 0; i < goroutineCount; i++ {
				go func() {
					time.Sleep(workDuration)
					// Simulate some work
					for j := 0; j < 1000; j++ {
						math.Sin(float64(j))
					}
				}()
				atomic.AddUint64(&ws.stats.GoroutinesCreated, 1)
			}
			atomic.AddUint64(&ws.stats.OperationsPerformed, 1)
		}
	}
}

func (ws *WorkloadSimulator) runGCWorkload(ctx context.Context, intensity string) {
	var allocSize int
	var allocCount int
	var interval time.Duration

	switch intensity {
	case "low":
		allocSize = 1024
		allocCount = 1000
		interval = 200 * time.Millisecond
	case "medium":
		allocSize = 4096
		allocCount = 5000
		interval = 100 * time.Millisecond
	case "high":
		allocSize = 8192
		allocCount = 10000
		interval = 50 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create many small allocations to trigger GC
			for i := 0; i < allocCount; i++ {
				data := make([]byte, allocSize)
				// Use the data to prevent optimization
				data[0] = byte(i % 256)
				_ = data
			}

			// Force GC occasionally
			if atomic.LoadUint64(&ws.stats.OperationsPerformed)%10 == 0 {
				runtime.GC()
			}

			atomic.AddUint64(&ws.stats.MemoryAllocated, uint64(allocSize*allocCount))
			atomic.AddUint64(&ws.stats.OperationsPerformed, 1)
		}
	}
}

func (ws *WorkloadSimulator) runMixedWorkload(ctx context.Context, intensity string) {
	// Run multiple workload types concurrently
	var wg sync.WaitGroup

	// CPU workload
	wg.Add(1)
	go func() {
		defer wg.Done()
		ws.runCPUWorkload(ctx, intensity)
	}()

	// Memory workload
	wg.Add(1)
	go func() {
		defer wg.Done()
		ws.runMemoryWorkload(ctx, intensity)
	}()

	// Goroutine workload (with reduced intensity)
	reducedIntensity := "low"
	if intensity == "high" {
		reducedIntensity = "medium"
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ws.runGoroutineWorkload(ctx, reducedIntensity)
	}()

	// GC workload (with reduced intensity)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ws.runGCWorkload(ctx, reducedIntensity)
	}()

	wg.Wait()
}