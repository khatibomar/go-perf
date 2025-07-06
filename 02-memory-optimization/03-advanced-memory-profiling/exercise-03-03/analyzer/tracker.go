package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Config holds configuration for the allocation tracker
type Config struct {
	SamplingRate     time.Duration
	AnomalyThreshold float64
	OutputDir        string
	Verbose          bool
	BufferSize       int
}

// AllocationMetrics represents allocation data at a point in time
type AllocationMetrics struct {
	Timestamp        time.Time `json:"timestamp"`
	BytesPerSecond   uint64    `json:"bytes_per_second"`
	ObjectsPerSecond uint64    `json:"objects_per_second"`
	HeapSize         uint64    `json:"heap_size"`
	HeapObjects      uint64    `json:"heap_objects"`
	StackSize        uint64    `json:"stack_size"`
	GCPauses         uint64    `json:"gc_pauses"`
	NumGC            uint32    `json:"num_gc"`
	Goroutines       int       `json:"goroutines"`
	CPUPercent       float64   `json:"cpu_percent"`
	MemoryPercent    float32   `json:"memory_percent"`
}

// AllocationTracker monitors allocation patterns in real-time
type AllocationTracker struct {
	config          *Config
	metrics         []AllocationMetrics
	mutex           sync.RWMutex
	lastMemStats    runtime.MemStats
	lastTime        time.Time
	process         *process.Process
	running         bool
	stopChan        chan struct{}
	metricsBuffer   chan AllocationMetrics
}

// NewAllocationTracker creates a new allocation tracker
func NewAllocationTracker(config *Config) *AllocationTracker {
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}

	proc, _ := process.NewProcess(int32(os.Getpid()))

	return &AllocationTracker{
		config:        config,
		metrics:       make([]AllocationMetrics, 0, config.BufferSize),
		process:       proc,
		stopChan:      make(chan struct{}),
		metricsBuffer: make(chan AllocationMetrics, config.BufferSize),
	}
}

// Start begins allocation tracking
func (at *AllocationTracker) Start(ctx context.Context) error {
	if at.running {
		return fmt.Errorf("tracker is already running")
	}

	at.running = true
	at.lastTime = time.Now()
	runtime.ReadMemStats(&at.lastMemStats)

	// Start metrics collection goroutine
	go at.collectMetrics(ctx)

	// Start metrics processing goroutine
	go at.processMetrics(ctx)

	return nil
}

// Stop stops allocation tracking
func (at *AllocationTracker) Stop() {
	if !at.running {
		return
	}

	at.running = false
	close(at.stopChan)
}

// GetCurrentMetrics returns the latest allocation metrics
func (at *AllocationTracker) GetCurrentMetrics() AllocationMetrics {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	if len(at.metrics) == 0 {
		return AllocationMetrics{Timestamp: time.Now()}
	}

	return at.metrics[len(at.metrics)-1]
}

// GetMetricsHistory returns all collected metrics
func (at *AllocationTracker) GetMetricsHistory() []AllocationMetrics {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]AllocationMetrics, len(at.metrics))
	copy(result, at.metrics)
	return result
}

// GetMetricsInRange returns metrics within a time range
func (at *AllocationTracker) GetMetricsInRange(start, end time.Time) []AllocationMetrics {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	var result []AllocationMetrics
	for _, metric := range at.metrics {
		if metric.Timestamp.After(start) && metric.Timestamp.Before(end) {
			result = append(result, metric)
		}
	}

	return result
}

// SaveMetrics saves collected metrics to a file
func (at *AllocationTracker) SaveMetrics(filename string) error {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(at.metrics)
}

// LoadMetrics loads metrics from a file
func (at *AllocationTracker) LoadMetrics(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var metrics []AllocationMetrics
	if err := json.NewDecoder(file).Decode(&metrics); err != nil {
		return err
	}

	at.mutex.Lock()
	at.metrics = metrics
	at.mutex.Unlock()

	return nil
}

// collectMetrics collects allocation metrics at regular intervals
func (at *AllocationTracker) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(at.config.SamplingRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-at.stopChan:
			return
		case <-ticker.C:
			metric := at.collectCurrentMetrics()
			select {
			case at.metricsBuffer <- metric:
			default:
				// Buffer is full, skip this metric
				if at.config.Verbose {
					fmt.Printf("Warning: metrics buffer full, skipping metric\n")
				}
			}
		}
	}
}

// processMetrics processes buffered metrics
func (at *AllocationTracker) processMetrics(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-at.stopChan:
			return
		case metric := <-at.metricsBuffer:
			at.mutex.Lock()
			at.metrics = append(at.metrics, metric)

			// Keep buffer size under control
			if len(at.metrics) > at.config.BufferSize {
				// Remove oldest 10% of metrics
				removeCount := at.config.BufferSize / 10
				at.metrics = at.metrics[removeCount:]
			}
			at.mutex.Unlock()
		}
	}
}

// collectCurrentMetrics collects current allocation metrics
func (at *AllocationTracker) collectCurrentMetrics() AllocationMetrics {
	now := time.Now()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate rates
	timeDelta := now.Sub(at.lastTime).Seconds()
	if timeDelta == 0 {
		timeDelta = 1 // Avoid division by zero
	}

	bytesDelta := memStats.TotalAlloc - at.lastMemStats.TotalAlloc
	objectsDelta := memStats.Mallocs - at.lastMemStats.Mallocs

	bytesPerSecond := uint64(float64(bytesDelta) / timeDelta)
	objectsPerSecond := uint64(float64(objectsDelta) / timeDelta)

	// Get process metrics
	var cpuPercent float64
	var memoryPercent float32
	if at.process != nil {
		if cpu, err := at.process.CPUPercent(); err == nil {
			cpuPercent = cpu
		}
		if mem, err := at.process.MemoryPercent(); err == nil {
			memoryPercent = mem
		}
	}

	metric := AllocationMetrics{
		Timestamp:        now,
		BytesPerSecond:   bytesPerSecond,
		ObjectsPerSecond: objectsPerSecond,
		HeapSize:         memStats.HeapAlloc,
		HeapObjects:      memStats.HeapObjects,
		StackSize:        memStats.StackInuse,
		GCPauses:         memStats.PauseTotalNs,
		NumGC:            memStats.NumGC,
		Goroutines:       runtime.NumGoroutine(),
		CPUPercent:       cpuPercent,
		MemoryPercent:    memoryPercent,
	}

	// Update last values
	at.lastMemStats = memStats
	at.lastTime = now

	return metric
}

// GetAllocationRate returns the current allocation rate in bytes per second
func (at *AllocationTracker) GetAllocationRate() float64 {
	metric := at.GetCurrentMetrics()
	return float64(metric.BytesPerSecond)
}

// GetObjectAllocationRate returns the current object allocation rate
func (at *AllocationTracker) GetObjectAllocationRate() float64 {
	metric := at.GetCurrentMetrics()
	return float64(metric.ObjectsPerSecond)
}

// GetMemoryUsage returns current memory usage statistics
func (at *AllocationTracker) GetMemoryUsage() (heapSize, stackSize uint64) {
	metric := at.GetCurrentMetrics()
	return metric.HeapSize, metric.StackSize
}

// GetGCStats returns garbage collection statistics
func (at *AllocationTracker) GetGCStats() (numGC uint32, totalPauses uint64) {
	metric := at.GetCurrentMetrics()
	return metric.NumGC, metric.GCPauses
}

// CalculateAverageRate calculates average allocation rate over a time window
func (at *AllocationTracker) CalculateAverageRate(window time.Duration) float64 {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	if len(at.metrics) == 0 {
		return 0
	}

	now := time.Now()
	cutoff := now.Add(-window)

	var totalBytes uint64
	var count int

	for i := len(at.metrics) - 1; i >= 0; i-- {
		if at.metrics[i].Timestamp.Before(cutoff) {
			break
		}
		totalBytes += at.metrics[i].BytesPerSecond
		count++
	}

	if count == 0 {
		return 0
	}

	return float64(totalBytes) / float64(count)
}

// DetectSpikes detects allocation spikes above threshold
func (at *AllocationTracker) DetectSpikes(threshold float64) []AllocationMetrics {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	if len(at.metrics) < 2 {
		return nil
	}

	// Calculate average rate
	var totalRate float64
	for _, metric := range at.metrics {
		totalRate += float64(metric.BytesPerSecond)
	}
	avgRate := totalRate / float64(len(at.metrics))

	// Find spikes
	var spikes []AllocationMetrics
	for _, metric := range at.metrics {
		if float64(metric.BytesPerSecond) > avgRate*threshold {
			spikes = append(spikes, metric)
		}
	}

	return spikes
}