// analysis/bottleneck_analyzer.go
package analysis

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"
)

type BottleneckAnalyzer struct {
	config      BottleneckConfig
	metrics     []PerformanceMetric
	mu          sync.RWMutex
	callbacks   []BottleneckCallback
	baselines   map[string]Baseline
	bottlenecks []Bottleneck
}

type BottleneckConfig struct {
	SampleInterval      time.Duration
	AnalysisWindow      time.Duration
	ThresholdMultiplier float64
	MinSamples          int
}

type PerformanceMetric struct {
	Timestamp       time.Time
	CPUUsage        float64
	MemoryUsage     uint64
	GoroutineCount  int
	GCPauseTime     time.Duration
	RequestLatency  time.Duration
	RequestRate     float64
	ErrorRate       float64
	DatabaseLatency time.Duration
	CacheHitRate    float64
	HeapObjects     uint64
	StackInuse      uint64
	NumGC           uint32
}

type Baseline struct {
	Metric string
	Value  float64
	StdDev float64
}

type Bottleneck struct {
	Type        string                 `json:"type"`
	Component   string                 `json:"component"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Metrics     map[string]interface{} `json:"metrics"`
	Timestamp   time.Time              `json:"timestamp"`
	Suggestions []string               `json:"suggestions"`
}

type BottleneckCallback func(bottleneck Bottleneck)

func NewBottleneckAnalyzer(config BottleneckConfig) *BottleneckAnalyzer {
	ba := &BottleneckAnalyzer{
		config:      config,
		metrics:     make([]PerformanceMetric, 0),
		baselines:   make(map[string]Baseline),
		bottlenecks: make([]Bottleneck, 0),
	}

	// Set default baselines
	ba.setDefaultBaselines()

	return ba
}

func (ba *BottleneckAnalyzer) setDefaultBaselines() {
	ba.baselines["cpu_usage"] = Baseline{"cpu_usage", 50.0, 15.0}
	ba.baselines["memory_usage"] = Baseline{"memory_usage", 100 * 1024 * 1024, 50 * 1024 * 1024} // 100MB ± 50MB
	ba.baselines["goroutine_count"] = Baseline{"goroutine_count", 50.0, 20.0}
	ba.baselines["gc_pause_time"] = Baseline{"gc_pause_time", 1000000.0, 500000.0} // 1ms ± 0.5ms in nanoseconds
	ba.baselines["request_latency"] = Baseline{"request_latency", 10000000.0, 5000000.0} // 10ms ± 5ms in nanoseconds
	ba.baselines["heap_objects"] = Baseline{"heap_objects", 10000.0, 5000.0}
}

func (ba *BottleneckAnalyzer) Start(ctx context.Context) {
	ticker := time.NewTicker(ba.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ba.collectMetrics()
			ba.analyzeBottlenecks()
		}
	}
}

func (ba *BottleneckAnalyzer) collectMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Simulate some additional metrics for demonstration
	metric := PerformanceMetric{
		Timestamp:       time.Now(),
		CPUUsage:        ba.simulateCPUUsage(),
		MemoryUsage:     m.Alloc,
		GoroutineCount:  runtime.NumGoroutine(),
		GCPauseTime:     time.Duration(m.PauseNs[(m.NumGC+255)%256]),
		RequestLatency:  ba.simulateRequestLatency(),
		RequestRate:     ba.simulateRequestRate(),
		ErrorRate:       ba.simulateErrorRate(),
		DatabaseLatency: ba.simulateDatabaseLatency(),
		CacheHitRate:    ba.simulateCacheHitRate(),
		HeapObjects:     m.HeapObjects,
		StackInuse:      m.StackInuse,
		NumGC:           m.NumGC,
	}

	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.metrics = append(ba.metrics, metric)

	// Keep only metrics within the analysis window
	cutoff := time.Now().Add(-ba.config.AnalysisWindow)
	for i, m := range ba.metrics {
		if m.Timestamp.After(cutoff) {
			ba.metrics = ba.metrics[i:]
			break
		}
	}
}

// Simulation functions for demonstration purposes
func (ba *BottleneckAnalyzer) simulateCPUUsage() float64 {
	// Simulate varying CPU usage
	base := 30.0
	variation := float64(time.Now().Unix()%20) * 2.0
	return base + variation
}

func (ba *BottleneckAnalyzer) simulateRequestLatency() time.Duration {
	// Simulate varying request latency
	base := 5 * time.Millisecond
	variation := time.Duration(time.Now().Unix()%10) * time.Millisecond
	return base + variation
}

func (ba *BottleneckAnalyzer) simulateRequestRate() float64 {
	// Simulate request rate
	return 100.0 + float64(time.Now().Unix()%50)
}

func (ba *BottleneckAnalyzer) simulateErrorRate() float64 {
	// Simulate error rate
	return float64(time.Now().Unix()%5) * 0.1
}

func (ba *BottleneckAnalyzer) simulateDatabaseLatency() time.Duration {
	// Simulate database latency
	base := 2 * time.Millisecond
	variation := time.Duration(time.Now().Unix()%5) * time.Millisecond
	return base + variation
}

func (ba *BottleneckAnalyzer) simulateCacheHitRate() float64 {
	// Simulate cache hit rate
	return 0.85 + float64(time.Now().Unix()%10)*0.01
}

func (ba *BottleneckAnalyzer) analyzeBottlenecks() {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	if len(ba.metrics) < ba.config.MinSamples {
		return
	}

	// Analyze different types of bottlenecks
	bottlenecks := []Bottleneck{}

	if bn := ba.analyzeCPUBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeMemoryBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeGoroutineBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeGCBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeLatencyBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeHeapObjectsBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	// Store bottlenecks and notify
	for _, bottleneck := range bottlenecks {
		ba.storeBottleneck(bottleneck)
		ba.notifyBottleneck(bottleneck)
	}
}

func (ba *BottleneckAnalyzer) analyzeCPUBottleneck() *Bottleneck {
	recent := ba.getRecentMetrics(time.Minute * 5)
	if len(recent) == 0 {
		return nil
	}

	avgCPU := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
		return m.CPUUsage
	})

	baseline := ba.getBaseline("cpu_usage")
	threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)

	if avgCPU > threshold {
		return &Bottleneck{
			Type:        "performance",
			Component:   "cpu",
			Severity:    ba.calculateSeverity(avgCPU, threshold),
			Description: fmt.Sprintf("High CPU usage detected: %.2f%% (threshold: %.2f%%)", avgCPU, threshold),
			Impact:      "High CPU usage may cause request latency and reduced throughput",
			Metrics: map[string]interface{}{
				"current_cpu": avgCPU,
				"threshold":   threshold,
				"baseline":    baseline.Value,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Profile CPU usage to identify hot paths",
				"Optimize algorithms and data structures",
				"Consider horizontal scaling",
				"Review goroutine usage patterns",
			},
		}
	}

	return nil
}

func (ba *BottleneckAnalyzer) analyzeMemoryBottleneck() *Bottleneck {
	recent := ba.getRecentMetrics(time.Minute * 5)
	if len(recent) == 0 {
		return nil
	}

	avgMemory := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
		return float64(m.MemoryUsage)
	})

	baseline := ba.getBaseline("memory_usage")
	threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)

	if avgMemory > threshold {
		return &Bottleneck{
			Type:        "resource",
			Component:   "memory",
			Severity:    ba.calculateSeverity(avgMemory, threshold),
			Description: fmt.Sprintf("High memory usage detected: %.2fMB (threshold: %.2fMB)", avgMemory/1024/1024, threshold/1024/1024),
			Impact:      "High memory usage may trigger frequent GC and cause performance degradation",
			Metrics: map[string]interface{}{
				"current_memory": avgMemory,
				"threshold":      threshold,
				"baseline":       baseline.Value,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Profile memory allocation patterns",
				"Identify and fix memory leaks",
				"Optimize data structures",
				"Implement object pooling",
				"Tune garbage collection settings",
			},
		}
	}

	return nil
}

func (ba *BottleneckAnalyzer) analyzeGoroutineBottleneck() *Bottleneck {
	recent := ba.getRecentMetrics(time.Minute * 5)
	if len(recent) == 0 {
		return nil
	}

	avgGoroutines := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
		return float64(m.GoroutineCount)
	})

	baseline := ba.getBaseline("goroutine_count")
	threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)

	if avgGoroutines > threshold {
		return &Bottleneck{
			Type:        "concurrency",
			Component:   "goroutines",
			Severity:    ba.calculateSeverity(avgGoroutines, threshold),
			Description: fmt.Sprintf("High goroutine count detected: %.0f (threshold: %.0f)", avgGoroutines, threshold),
			Impact:      "Excessive goroutines may cause scheduler overhead and memory pressure",
			Metrics: map[string]interface{}{
				"current_goroutines": avgGoroutines,
				"threshold":          threshold,
				"baseline":           baseline.Value,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Profile goroutine creation patterns",
				"Implement goroutine pooling",
				"Review blocking operations",
				"Optimize channel usage",
				"Check for goroutine leaks",
			},
		}
	}

	return nil
}

func (ba *BottleneckAnalyzer) analyzeGCBottleneck() *Bottleneck {
	recent := ba.getRecentMetrics(time.Minute * 5)
	if len(recent) == 0 {
		return nil
	}

	avgGCPause := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
		return float64(m.GCPauseTime.Nanoseconds())
	})

	baseline := ba.getBaseline("gc_pause_time")
	threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)

	if avgGCPause > threshold {
		return &Bottleneck{
			Type:        "gc",
			Component:   "garbage_collector",
			Severity:    ba.calculateSeverity(avgGCPause, threshold),
			Description: fmt.Sprintf("High GC pause time detected: %.2fms (threshold: %.2fms)", avgGCPause/1e6, threshold/1e6),
			Impact:      "Long GC pauses cause request latency spikes and reduced throughput",
			Metrics: map[string]interface{}{
				"current_gc_pause": avgGCPause,
				"threshold":        threshold,
				"baseline":         baseline.Value,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Tune GOGC environment variable",
				"Reduce allocation rate",
				"Implement object pooling",
				"Optimize data structures",
				"Consider using sync.Pool",
			},
		}
	}

	return nil
}

func (ba *BottleneckAnalyzer) analyzeLatencyBottleneck() *Bottleneck {
	recent := ba.getRecentMetrics(time.Minute * 5)
	if len(recent) == 0 {
		return nil
	}

	avgLatency := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
		return float64(m.RequestLatency.Nanoseconds())
	})

	baseline := ba.getBaseline("request_latency")
	threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)

	if avgLatency > threshold {
		return &Bottleneck{
			Type:        "latency",
			Component:   "request_processing",
			Severity:    ba.calculateSeverity(avgLatency, threshold),
			Description: fmt.Sprintf("High request latency detected: %.2fms (threshold: %.2fms)", avgLatency/1e6, threshold/1e6),
			Impact:      "High latency affects user experience and system throughput",
			Metrics: map[string]interface{}{
				"current_latency": avgLatency,
				"threshold":       threshold,
				"baseline":        baseline.Value,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Profile request processing paths",
				"Optimize database queries",
				"Implement caching strategies",
				"Review external service calls",
				"Optimize serialization/deserialization",
			},
		}
	}

	return nil
}

func (ba *BottleneckAnalyzer) analyzeHeapObjectsBottleneck() *Bottleneck {
	recent := ba.getRecentMetrics(time.Minute * 5)
	if len(recent) == 0 {
		return nil
	}

	avgHeapObjects := ba.calculateAverage(recent, func(m PerformanceMetric) float64 {
		return float64(m.HeapObjects)
	})

	baseline := ba.getBaseline("heap_objects")
	threshold := baseline.Value + (baseline.StdDev * ba.config.ThresholdMultiplier)

	if avgHeapObjects > threshold {
		return &Bottleneck{
			Type:        "memory",
			Component:   "heap_objects",
			Severity:    ba.calculateSeverity(avgHeapObjects, threshold),
			Description: fmt.Sprintf("High heap object count detected: %.0f (threshold: %.0f)", avgHeapObjects, threshold),
			Impact:      "High object count increases GC pressure and memory overhead",
			Metrics: map[string]interface{}{
				"current_heap_objects": avgHeapObjects,
				"threshold":            threshold,
				"baseline":             baseline.Value,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Reduce object allocations",
				"Implement object reuse patterns",
				"Use value types where possible",
				"Optimize data structures",
				"Consider object pooling",
			},
		}
	}

	return nil
}

func (ba *BottleneckAnalyzer) getRecentMetrics(duration time.Duration) []PerformanceMetric {
	cutoff := time.Now().Add(-duration)
	var recent []PerformanceMetric

	for _, metric := range ba.metrics {
		if metric.Timestamp.After(cutoff) {
			recent = append(recent, metric)
		}
	}

	return recent
}

func (ba *BottleneckAnalyzer) calculateAverage(metrics []PerformanceMetric, extractor func(PerformanceMetric) float64) float64 {
	if len(metrics) == 0 {
		return 0
	}

	sum := 0.0
	for _, metric := range metrics {
		sum += extractor(metric)
	}

	return sum / float64(len(metrics))
}

func (ba *BottleneckAnalyzer) getBaseline(metric string) Baseline {
	if baseline, exists := ba.baselines[metric]; exists {
		return baseline
	}

	// Return default baseline if not found
	return Baseline{
		Metric: metric,
		Value:  0,
		StdDev: 1,
	}
}

func (ba *BottleneckAnalyzer) calculateSeverity(current, threshold float64) string {
	ratio := current / threshold
	switch {
	case ratio >= 3:
		return "critical"
	case ratio >= 2:
		return "high"
	case ratio >= 1.5:
		return "medium"
	default:
		return "low"
	}
}

func (ba *BottleneckAnalyzer) storeBottleneck(bottleneck Bottleneck) {
	ba.bottlenecks = append(ba.bottlenecks, bottleneck)

	// Keep only recent bottlenecks (last 24 hours)
	cutoff := time.Now().Add(-24 * time.Hour)
	for i, b := range ba.bottlenecks {
		if b.Timestamp.After(cutoff) {
			ba.bottlenecks = ba.bottlenecks[i:]
			break
		}
	}
}

func (ba *BottleneckAnalyzer) notifyBottleneck(bottleneck Bottleneck) {
	for _, callback := range ba.callbacks {
		go callback(bottleneck)
	}
}

func (ba *BottleneckAnalyzer) RegisterBottleneckCallback(callback BottleneckCallback) {
	ba.callbacks = append(ba.callbacks, callback)
}

func (ba *BottleneckAnalyzer) UpdateBaseline(metric string, value, stdDev float64) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.baselines[metric] = Baseline{
		Metric: metric,
		Value:  value,
		StdDev: stdDev,
	}
}

func (ba *BottleneckAnalyzer) GetBaselines() map[string]Baseline {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	// Return a copy of baselines to avoid race conditions
	result := make(map[string]Baseline)
	for k, v := range ba.baselines {
		result[k] = v
	}
	return result
}

func (ba *BottleneckAnalyzer) GetCurrentMetrics() PerformanceMetric {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	if len(ba.metrics) == 0 {
		return PerformanceMetric{}
	}

	return ba.metrics[len(ba.metrics)-1]
}

func (ba *BottleneckAnalyzer) GetRecentBottlenecks(since time.Duration) []Bottleneck {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	cutoff := time.Now().Add(-since)
	var recent []Bottleneck

	for _, bottleneck := range ba.bottlenecks {
		if bottleneck.Timestamp.After(cutoff) {
			recent = append(recent, bottleneck)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Timestamp.After(recent[j].Timestamp)
	})

	return recent
}

func (ba *BottleneckAnalyzer) AnalyzeNow() []Bottleneck {
	start := time.Now()
	defer func() {
		// Ensure analysis completes within performance target (<5 minutes)
		if elapsed := time.Since(start); elapsed > 5*time.Minute {
			// Log warning about slow analysis
			fmt.Printf("Warning: Bottleneck analysis took %v (target: <5 minutes)\n", elapsed)
		}
	}()

	ba.mu.RLock()
	defer ba.mu.RUnlock()

	if len(ba.metrics) < ba.config.MinSamples {
		return []Bottleneck{}
	}

	// Analyze all bottleneck types
	bottlenecks := []Bottleneck{}

	if bn := ba.analyzeCPUBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeMemoryBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeGoroutineBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeGCBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeLatencyBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	if bn := ba.analyzeHeapObjectsBottleneck(); bn != nil {
		bottlenecks = append(bottlenecks, *bn)
	}

	return bottlenecks
}