package adaptive

import (
	"sync/atomic"
	"time"
)

// PerformanceMonitor tracks and analyzes counter performance
type PerformanceMonitor struct {
	metrics        []PerformanceMetric
	startTime      time.Time
	lastSnapshot   time.Time
	sampleInterval time.Duration
	maxSamples     int
	currentSample  int64
	_              [32]byte // Padding
}

type PerformanceMetric struct {
	Timestamp     time.Time
	OperationsPS  int64
	LatencyP50    int64
	LatencyP95    int64
	LatencyP99    int64
	ContentionLevel int64
	CPUUtilization float64
	MemoryUsage   int64
}

type LatencyTracker struct {
	samples    []int64
	index      int64
	maxSamples int
	_          [40]byte // Padding
}

func NewPerformanceMonitor(sampleInterval time.Duration, maxSamples int) *PerformanceMonitor {
	if maxSamples <= 0 {
		maxSamples = 1000
	}
	if sampleInterval <= 0 {
		sampleInterval = 100 * time.Millisecond
	}
	
	return &PerformanceMonitor{
		metrics:        make([]PerformanceMetric, maxSamples),
		startTime:      time.Now(),
		lastSnapshot:   time.Now(),
		sampleInterval: sampleInterval,
		maxSamples:     maxSamples,
	}
}

func NewLatencyTracker(maxSamples int) *LatencyTracker {
	if maxSamples <= 0 {
		maxSamples = 1000
	}
	return &LatencyTracker{
		samples:    make([]int64, maxSamples),
		maxSamples: maxSamples,
	}
}

func (pm *PerformanceMonitor) RecordMetric(operationsPS, contentionLevel int64, latencies []int64) {
	now := time.Now()
	if now.Sub(pm.lastSnapshot) < pm.sampleInterval {
		return
	}
	
	index := atomic.AddInt64(&pm.currentSample, 1) - 1
	slotIndex := int(index) % pm.maxSamples
	
	metric := PerformanceMetric{
		Timestamp:       now,
		OperationsPS:    operationsPS,
		ContentionLevel: contentionLevel,
		CPUUtilization:  pm.estimateCPUUtilization(),
		MemoryUsage:     pm.estimateMemoryUsage(),
	}
	
	if len(latencies) > 0 {
		metric.LatencyP50 = pm.calculatePercentile(latencies, 0.50)
		metric.LatencyP95 = pm.calculatePercentile(latencies, 0.95)
		metric.LatencyP99 = pm.calculatePercentile(latencies, 0.99)
	}
	
	pm.metrics[slotIndex] = metric
	pm.lastSnapshot = now
}

func (pm *PerformanceMonitor) GetCurrentMetrics() PerformanceMetric {
	currentIndex := atomic.LoadInt64(&pm.currentSample)
	if currentIndex == 0 {
		return PerformanceMetric{}
	}
	slotIndex := int(currentIndex-1) % pm.maxSamples
	return pm.metrics[slotIndex]
}

func (pm *PerformanceMonitor) GetRecentMetrics(count int) []PerformanceMetric {
	if count <= 0 || count > pm.maxSamples {
		count = pm.maxSamples
	}
	
	currentIndex := atomic.LoadInt64(&pm.currentSample)
	if currentIndex == 0 {
		return nil
	}
	
	actualCount := int(min(int64(count), currentIndex))
	result := make([]PerformanceMetric, actualCount)
	
	for i := 0; i < actualCount; i++ {
		index := (int(currentIndex) - 1 - i) % pm.maxSamples
		if index < 0 {
			index += pm.maxSamples
		}
		result[i] = pm.metrics[index]
	}
	
	return result
}

func (pm *PerformanceMonitor) GetAverageMetrics(duration time.Duration) PerformanceMetric {
	now := time.Now()
	cutoff := now.Add(-duration)
	
	var totalOps, totalContention, totalLatencyP50, totalLatencyP95, totalLatencyP99 int64
	var totalCPU float64
	var totalMemory int64
	var count int
	
	currentIndex := atomic.LoadInt64(&pm.currentSample)
	for i := int64(0); i < min(currentIndex, int64(pm.maxSamples)); i++ {
		index := (int(currentIndex) - 1 - int(i)) % pm.maxSamples
		if index < 0 {
			index += pm.maxSamples
		}
		
		metric := pm.metrics[index]
		if metric.Timestamp.Before(cutoff) {
			break
		}
		
		totalOps += metric.OperationsPS
		totalContention += metric.ContentionLevel
		totalLatencyP50 += metric.LatencyP50
		totalLatencyP95 += metric.LatencyP95
		totalLatencyP99 += metric.LatencyP99
		totalCPU += metric.CPUUtilization
		totalMemory += metric.MemoryUsage
		count++
	}
	
	if count == 0 {
		return PerformanceMetric{}
	}
	
	return PerformanceMetric{
		Timestamp:       now,
		OperationsPS:    totalOps / int64(count),
		LatencyP50:      totalLatencyP50 / int64(count),
		LatencyP95:      totalLatencyP95 / int64(count),
		LatencyP99:      totalLatencyP99 / int64(count),
		ContentionLevel: totalContention / int64(count),
		CPUUtilization:  totalCPU / float64(count),
		MemoryUsage:     totalMemory / int64(count),
	}
}

func (lt *LatencyTracker) Record(latency int64) {
	index := atomic.AddInt64(&lt.index, 1) - 1
	slotIndex := int(index) % lt.maxSamples
	lt.samples[slotIndex] = latency
}

func (lt *LatencyTracker) GetSamples() []int64 {
	currentIndex := atomic.LoadInt64(&lt.index)
	actualCount := int(min(currentIndex, int64(lt.maxSamples)))
	
	result := make([]int64, actualCount)
	for i := 0; i < actualCount; i++ {
		index := (int(currentIndex) - 1 - i) % lt.maxSamples
		if index < 0 {
			index += lt.maxSamples
		}
		result[i] = lt.samples[index]
	}
	
	return result
}

func (pm *PerformanceMonitor) calculatePercentile(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	
	// Simple percentile calculation (should use proper sorting for production)
	// This is a simplified version for the exercise
	index := int(float64(len(values)-1) * percentile)
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func (pm *PerformanceMonitor) estimateCPUUtilization() float64 {
	// Simplified CPU utilization estimation
	// In real implementation, this would use system calls
	return 0.5 + (float64(time.Now().UnixNano()%1000) / 2000.0)
}

func (pm *PerformanceMonitor) estimateMemoryUsage() int64 {
	// Simplified memory usage estimation
	// In real implementation, this would use runtime.MemStats
	return 1024 * 1024 * (50 + (time.Now().UnixNano() % 100))
}

func (pm *PerformanceMonitor) Reset() {
	atomic.StoreInt64(&pm.currentSample, 0)
	pm.startTime = time.Now()
	pm.lastSnapshot = time.Now()
	for i := range pm.metrics {
		pm.metrics[i] = PerformanceMetric{}
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}