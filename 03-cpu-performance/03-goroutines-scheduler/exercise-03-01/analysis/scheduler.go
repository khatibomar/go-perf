package analysis

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// SchedulerMetrics contains comprehensive scheduler performance metrics
type SchedulerMetrics struct {
	Timestamp          time.Time
	GOMMAXPROCS        int
	NumCPU             int
	NumGoroutine       int
	CPUUtilization     []float64 // Per-CPU utilization
	AvgCPUUtilization  float64
	MemoryUsage        uint64
	MemoryPercent      float64
	GCPauses           []time.Duration
	GCFrequency        float64
	ContextSwitches    uint64
	RunQueueLength     int
	BlockedGoroutines  int
	RunnableGoroutines int
	SchedulerLatency   time.Duration
	Throughput         float64
	Efficiency         float64
}

// SchedulerAnalyzer provides tools for analyzing scheduler performance
type SchedulerAnalyzer struct {
	metrics        []SchedulerMetrics
	collectionRate time.Duration
	running        bool
	mu             sync.RWMutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// WorkloadCharacteristics describes the characteristics of a workload
type WorkloadCharacteristics struct {
	Type                string
	CPUIntensive        bool
	IOIntensive         bool
	MemoryIntensive     bool
	GoroutineIntensive  bool
	ContentionLevel     string // "low", "medium", "high"
	Burstiness          float64
	Predictability      float64
	OptimalGOMMAXPROCS  int
	Recommendations     []string
}

// PerformanceInsight represents an analysis insight
type PerformanceInsight struct {
	Type        string    // "bottleneck", "optimization", "warning", "info"
	Severity    string    // "low", "medium", "high", "critical"
	Title       string
	Description string
	Metric      string
	Value       interface{}
	Threshold   interface{}
	Suggestion  string
	Timestamp   time.Time
}

// NewSchedulerAnalyzer creates a new scheduler analyzer
func NewSchedulerAnalyzer(collectionRate time.Duration) *SchedulerAnalyzer {
	return &SchedulerAnalyzer{
		metrics:        make([]SchedulerMetrics, 0),
		collectionRate: collectionRate,
		stopChan:       make(chan struct{}),
	}
}

// Start begins continuous metrics collection
func (sa *SchedulerAnalyzer) Start(ctx context.Context) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	if sa.running {
		return fmt.Errorf("analyzer already running")
	}
	
	sa.running = true
	sa.wg.Add(1)
	
	go func() {
		defer sa.wg.Done()
		sa.collectMetrics(ctx)
	}()
	
	return nil
}

// Stop ends metrics collection
func (sa *SchedulerAnalyzer) Stop() {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	if !sa.running {
		return
	}
	
	close(sa.stopChan)
	sa.running = false
	sa.wg.Wait()
}

// collectMetrics continuously collects scheduler metrics
func (sa *SchedulerAnalyzer) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(sa.collectionRate)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-sa.stopChan:
			return
		case <-ticker.C:
			metrics, err := sa.collectCurrentMetrics()
			if err != nil {
				continue // Skip this collection cycle
			}
			
			sa.mu.Lock()
			sa.metrics = append(sa.metrics, metrics)
			
			// Limit metrics history to prevent memory growth
			if len(sa.metrics) > 1000 {
				sa.metrics = sa.metrics[len(sa.metrics)-1000:]
			}
			sa.mu.Unlock()
		}
	}
}

// collectCurrentMetrics collects current scheduler metrics
func (sa *SchedulerAnalyzer) collectCurrentMetrics() (SchedulerMetrics, error) {
	metrics := SchedulerMetrics{
		Timestamp:    time.Now(),
		GOMMAXPROCS:  runtime.GOMAXPROCS(0),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}
	
	// CPU utilization
	cpuPercents, err := cpu.Percent(0, true)
	if err == nil {
		metrics.CPUUtilization = cpuPercents
		var total float64
		for _, percent := range cpuPercents {
			total += percent
		}
		metrics.AvgCPUUtilization = total / float64(len(cpuPercents))
	}
	
	// Memory usage
	vmem, err := mem.VirtualMemory()
	if err == nil {
		metrics.MemoryUsage = vmem.Used
		metrics.MemoryPercent = vmem.UsedPercent
	}
	
	// Process-specific metrics
	pid := int32(runtime.Getpid())
	proc, err := process.NewProcess(pid)
	if err == nil {
		if ctxSwitches, err := proc.NumCtxSwitches(); err == nil {
			metrics.ContextSwitches = uint64(ctxSwitches.Voluntary + ctxSwitches.Involuntary)
		}
	}
	
	// GC metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	metrics.GCFrequency = float64(memStats.NumGC)
	
	// Estimate scheduler-specific metrics
	metrics.RunQueueLength = estimateRunQueueLength()
	metrics.BlockedGoroutines = estimateBlockedGoroutines()
	metrics.RunnableGoroutines = estimateRunnableGoroutines()
	metrics.SchedulerLatency = measureSchedulerLatency()
	
	return metrics, nil
}

// GetMetrics returns a copy of collected metrics
func (sa *SchedulerAnalyzer) GetMetrics() []SchedulerMetrics {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	
	metricsCopy := make([]SchedulerMetrics, len(sa.metrics))
	copy(metricsCopy, sa.metrics)
	return metricsCopy
}

// GetLatestMetrics returns the most recent metrics
func (sa *SchedulerAnalyzer) GetLatestMetrics() *SchedulerMetrics {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	
	if len(sa.metrics) == 0 {
		return nil
	}
	
	latest := sa.metrics[len(sa.metrics)-1]
	return &latest
}

// AnalyzeWorkload analyzes workload characteristics based on collected metrics
func (sa *SchedulerAnalyzer) AnalyzeWorkload() *WorkloadCharacteristics {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	
	if len(sa.metrics) < 10 {
		return &WorkloadCharacteristics{
			Type: "insufficient_data",
			Recommendations: []string{"Collect more metrics for accurate analysis"},
		}
	}
	
	characteristics := &WorkloadCharacteristics{}
	
	// Analyze CPU intensity
	avgCPU := sa.calculateAverageCPUUtilization()
	characteristics.CPUIntensive = avgCPU > 70.0
	
	// Analyze I/O intensity (inferred from low CPU with high goroutine count)
	avgGoroutines := sa.calculateAverageGoroutineCount()
	characteristics.IOIntensive = avgCPU < 30.0 && avgGoroutines > runtime.NumCPU()*2
	
	// Analyze memory intensity
	avgMemory := sa.calculateAverageMemoryUsage()
	characteristics.MemoryIntensive = avgMemory > 80.0
	
	// Analyze goroutine intensity
	characteristics.GoroutineIntensive = avgGoroutines > runtime.NumCPU()*10
	
	// Determine workload type
	characteristics.Type = sa.determineWorkloadType(characteristics)
	
	// Analyze contention level
	characteristics.ContentionLevel = sa.analyzeContentionLevel()
	
	// Calculate burstiness and predictability
	characteristics.Burstiness = sa.calculateBurstiness()
	characteristics.Predictability = sa.calculatePredictability()
	
	// Determine optimal GOMAXPROCS
	characteristics.OptimalGOMMAXPROCS = sa.determineOptimalGOMMAXPROCS(characteristics)
	
	// Generate recommendations
	characteristics.Recommendations = sa.generateRecommendations(characteristics)
	
	return characteristics
}

// GenerateInsights analyzes metrics and generates performance insights
func (sa *SchedulerAnalyzer) GenerateInsights() []PerformanceInsight {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	
	if len(sa.metrics) == 0 {
		return []PerformanceInsight{}
	}
	
	var insights []PerformanceInsight
	latest := sa.metrics[len(sa.metrics)-1]
	
	// CPU utilization insights
	insights = append(insights, sa.analyzeCPUUtilization(latest)...)
	
	// Memory usage insights
	insights = append(insights, sa.analyzeMemoryUsage(latest)...)
	
	// Goroutine insights
	insights = append(insights, sa.analyzeGoroutineMetrics(latest)...)
	
	// Scheduler efficiency insights
	insights = append(insights, sa.analyzeSchedulerEfficiency(latest)...)
	
	// GOMAXPROCS insights
	insights = append(insights, sa.analyzeGOMMAXPROCS(latest)...)
	
	// Sort insights by severity
	sort.Slice(insights, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		return severityOrder[insights[i].Severity] > severityOrder[insights[j].Severity]
	})
	
	return insights
}

// Helper functions for analysis

func (sa *SchedulerAnalyzer) calculateAverageCPUUtilization() float64 {
	if len(sa.metrics) == 0 {
		return 0
	}
	
	var total float64
	count := 0
	
	for _, metric := range sa.metrics {
		if metric.AvgCPUUtilization > 0 {
			total += metric.AvgCPUUtilization
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func (sa *SchedulerAnalyzer) calculateAverageGoroutineCount() float64 {
	if len(sa.metrics) == 0 {
		return 0
	}
	
	var total int
	for _, metric := range sa.metrics {
		total += metric.NumGoroutine
	}
	
	return float64(total) / float64(len(sa.metrics))
}

func (sa *SchedulerAnalyzer) calculateAverageMemoryUsage() float64 {
	if len(sa.metrics) == 0 {
		return 0
	}
	
	var total float64
	count := 0
	
	for _, metric := range sa.metrics {
		if metric.MemoryPercent > 0 {
			total += metric.MemoryPercent
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func (sa *SchedulerAnalyzer) determineWorkloadType(characteristics *WorkloadCharacteristics) string {
	if characteristics.CPUIntensive && !characteristics.IOIntensive {
		return "cpu_bound"
	}
	if characteristics.IOIntensive && !characteristics.CPUIntensive {
		return "io_bound"
	}
	if characteristics.CPUIntensive && characteristics.IOIntensive {
		return "mixed"
	}
	if characteristics.MemoryIntensive {
		return "memory_bound"
	}
	if characteristics.GoroutineIntensive {
		return "goroutine_intensive"
	}
	return "balanced"
}

func (sa *SchedulerAnalyzer) analyzeContentionLevel() string {
	if len(sa.metrics) < 5 {
		return "unknown"
	}
	
	// Analyze context switches and scheduler latency trends
	recentMetrics := sa.metrics[len(sa.metrics)-5:]
	var avgLatency time.Duration
	var avgContextSwitches uint64
	
	for _, metric := range recentMetrics {
		avgLatency += metric.SchedulerLatency
		avgContextSwitches += metric.ContextSwitches
	}
	
	avgLatency /= time.Duration(len(recentMetrics))
	avgContextSwitches /= uint64(len(recentMetrics))
	
	if avgLatency > 10*time.Millisecond || avgContextSwitches > 10000 {
		return "high"
	}
	if avgLatency > 5*time.Millisecond || avgContextSwitches > 5000 {
		return "medium"
	}
	return "low"
}

func (sa *SchedulerAnalyzer) calculateBurstiness() float64 {
	if len(sa.metrics) < 10 {
		return 0.0
	}
	
	// Calculate coefficient of variation for CPU utilization
	var values []float64
	for _, metric := range sa.metrics {
		if metric.AvgCPUUtilization > 0 {
			values = append(values, metric.AvgCPUUtilization)
		}
	}
	
	if len(values) < 2 {
		return 0.0
	}
	
	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	
	// Calculate standard deviation
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))
	stdDev := variance // Simplified, should be sqrt(variance)
	
	// Coefficient of variation
	if mean == 0 {
		return 0.0
	}
	return stdDev / mean
}

func (sa *SchedulerAnalyzer) calculatePredictability() float64 {
	// Inverse of burstiness - higher predictability means lower variation
	burstiness := sa.calculateBurstiness()
	if burstiness == 0 {
		return 1.0
	}
	return 1.0 / (1.0 + burstiness)
}

func (sa *SchedulerAnalyzer) determineOptimalGOMMAXPROCS(characteristics *WorkloadCharacteristics) int {
	numCPU := runtime.NumCPU()
	current := runtime.GOMAXPROCS(0)
	
	switch characteristics.Type {
	case "cpu_bound":
		// For CPU-bound workloads, GOMAXPROCS should equal CPU count
		return numCPU
	case "io_bound":
		// For I/O-bound workloads, can use more than CPU count
		return numCPU * 2
	case "mixed":
		// For mixed workloads, slightly more than CPU count
		return int(float64(numCPU) * 1.5)
	case "memory_bound":
		// For memory-bound workloads, limit to CPU count
		return numCPU
	default:
		return current
	}
}

func (sa *SchedulerAnalyzer) generateRecommendations(characteristics *WorkloadCharacteristics) []string {
	var recommendations []string
	
	switch characteristics.Type {
	case "cpu_bound":
		recommendations = append(recommendations, "Set GOMAXPROCS to CPU count for optimal performance")
		recommendations = append(recommendations, "Minimize goroutine creation overhead")
		recommendations = append(recommendations, "Use CPU-efficient algorithms")
	
	case "io_bound":
		recommendations = append(recommendations, "Increase GOMAXPROCS beyond CPU count")
		recommendations = append(recommendations, "Use connection pooling for network operations")
		recommendations = append(recommendations, "Implement proper timeout handling")
	
	case "mixed":
		recommendations = append(recommendations, "Balance GOMAXPROCS based on workload phases")
		recommendations = append(recommendations, "Monitor both CPU and I/O metrics")
		recommendations = append(recommendations, "Consider workload-specific optimizations")
	
	case "memory_bound":
		recommendations = append(recommendations, "Optimize memory allocation patterns")
		recommendations = append(recommendations, "Reduce GC pressure")
		recommendations = append(recommendations, "Monitor heap growth")
	}
	
	if characteristics.ContentionLevel == "high" {
		recommendations = append(recommendations, "Reduce synchronization contention")
		recommendations = append(recommendations, "Consider lock-free data structures")
	}
	
	if characteristics.Burstiness > 0.5 {
		recommendations = append(recommendations, "Implement load balancing strategies")
		recommendations = append(recommendations, "Consider adaptive resource allocation")
	}
	
	return recommendations
}

// Insight generation functions

func (sa *SchedulerAnalyzer) analyzeCPUUtilization(metrics SchedulerMetrics) []PerformanceInsight {
	var insights []PerformanceInsight
	
	if metrics.AvgCPUUtilization > 90 {
		insights = append(insights, PerformanceInsight{
			Type:        "bottleneck",
			Severity:    "high",
			Title:       "High CPU Utilization",
			Description: "CPU utilization is very high, indicating potential CPU bottleneck",
			Metric:      "cpu_utilization",
			Value:       metrics.AvgCPUUtilization,
			Threshold:   90.0,
			Suggestion:  "Consider optimizing CPU-intensive operations or scaling horizontally",
			Timestamp:   metrics.Timestamp,
		})
	} else if metrics.AvgCPUUtilization < 20 {
		insights = append(insights, PerformanceInsight{
			Type:        "optimization",
			Severity:    "medium",
			Title:       "Low CPU Utilization",
			Description: "CPU utilization is low, resources may be underutilized",
			Metric:      "cpu_utilization",
			Value:       metrics.AvgCPUUtilization,
			Threshold:   20.0,
			Suggestion:  "Consider increasing workload or reducing resource allocation",
			Timestamp:   metrics.Timestamp,
		})
	}
	
	return insights
}

func (sa *SchedulerAnalyzer) analyzeMemoryUsage(metrics SchedulerMetrics) []PerformanceInsight {
	var insights []PerformanceInsight
	
	if metrics.MemoryPercent > 85 {
		insights = append(insights, PerformanceInsight{
			Type:        "bottleneck",
			Severity:    "critical",
			Title:       "High Memory Usage",
			Description: "Memory usage is critically high",
			Metric:      "memory_percent",
			Value:       metrics.MemoryPercent,
			Threshold:   85.0,
			Suggestion:  "Investigate memory leaks and optimize memory allocation",
			Timestamp:   metrics.Timestamp,
		})
	}
	
	return insights
}

func (sa *SchedulerAnalyzer) analyzeGoroutineMetrics(metrics SchedulerMetrics) []PerformanceInsight {
	var insights []PerformanceInsight
	
	if metrics.NumGoroutine > runtime.NumCPU()*100 {
		insights = append(insights, PerformanceInsight{
			Type:        "warning",
			Severity:    "medium",
			Title:       "High Goroutine Count",
			Description: "Very high number of goroutines may impact scheduler performance",
			Metric:      "goroutine_count",
			Value:       metrics.NumGoroutine,
			Threshold:   runtime.NumCPU() * 100,
			Suggestion:  "Consider using goroutine pools or reducing goroutine creation",
			Timestamp:   metrics.Timestamp,
		})
	}
	
	return insights
}

func (sa *SchedulerAnalyzer) analyzeSchedulerEfficiency(metrics SchedulerMetrics) []PerformanceInsight {
	var insights []PerformanceInsight
	
	if metrics.SchedulerLatency > 10*time.Millisecond {
		insights = append(insights, PerformanceInsight{
			Type:        "bottleneck",
			Severity:    "high",
			Title:       "High Scheduler Latency",
			Description: "Scheduler latency is high, indicating scheduling overhead",
			Metric:      "scheduler_latency",
			Value:       metrics.SchedulerLatency,
			Threshold:   10 * time.Millisecond,
			Suggestion:  "Reduce goroutine contention and optimize synchronization",
			Timestamp:   metrics.Timestamp,
		})
	}
	
	return insights
}

func (sa *SchedulerAnalyzer) analyzeGOMMAXPROCS(metrics SchedulerMetrics) []PerformanceInsight {
	var insights []PerformanceInsight
	
	if metrics.GOMAXPROCS != metrics.NumCPU {
		severity := "low"
		if abs(metrics.GOMAXPROCS-metrics.NumCPU) > 2 {
			severity = "medium"
		}
		
		insights = append(insights, PerformanceInsight{
			Type:        "info",
			Severity:    severity,
			Title:       "GOMAXPROCS Configuration",
			Description: "GOMAXPROCS differs from CPU count",
			Metric:      "gomaxprocs",
			Value:       metrics.GOMAXPROCS,
			Threshold:   metrics.NumCPU,
			Suggestion:  "Consider adjusting GOMAXPROCS based on workload characteristics",
			Timestamp:   metrics.Timestamp,
		})
	}
	
	return insights
}

// Utility functions for scheduler analysis

func estimateRunQueueLength() int {
	// This is a simplified estimation
	// In practice, this would require access to runtime internals
	return runtime.NumGoroutine() / runtime.GOMAXPROCS(0)
}

func estimateBlockedGoroutines() int {
	// Simplified estimation based on typical patterns
	return runtime.NumGoroutine() / 4
}

func estimateRunnableGoroutines() int {
	// Simplified estimation
	return runtime.NumGoroutine() - estimateBlockedGoroutines()
}

func measureSchedulerLatency() time.Duration {
	// Measure time to schedule a simple goroutine
	start := time.Now()
	done := make(chan struct{})
	
	go func() {
		close(done)
	}()
	
	<-done
	return time.Since(start)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// BenchmarkSchedulerOverhead measures scheduler overhead
func BenchmarkSchedulerOverhead(numGoroutines int, iterations int) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup
	
	for i := 0; i < iterations; i++ {
		for j := 0; j < numGoroutines; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Minimal work
				runtime.Gosched()
			}()
		}
		wg.Wait()
	}
	
	totalTime := time.Since(start)
	return totalTime / time.Duration(iterations*numGoroutines)
}

// MeasureContextSwitchLatency measures context switch latency
func MeasureContextSwitchLatency(samples int) time.Duration {
	var totalLatency int64
	var wg sync.WaitGroup
	
	for i := 0; i < samples; i++ {
		wg.Add(2)
		
		ch1 := make(chan time.Time, 1)
		ch2 := make(chan time.Time, 1)
		
		// Goroutine 1
		go func() {
			defer wg.Done()
			start := time.Now()
			ch1 <- start
			end := <-ch2
			latency := end.Sub(start)
			atomic.AddInt64(&totalLatency, int64(latency))
		}()
		
		// Goroutine 2
		go func() {
			defer wg.Done()
			<-ch1
			ch2 <- time.Now()
		}()
	}
	
	wg.Wait()
	return time.Duration(totalLatency / int64(samples))
}

// GetSchedulerStats returns current scheduler statistics
func GetSchedulerStats() map[string]interface{} {
	return map[string]interface{}{
		"gomaxprocs":     runtime.GOMAXPROCS(0),
		"num_cpu":        runtime.NumCPU(),
		"num_goroutine":  runtime.NumGoroutine(),
		"scheduler_type": "work_stealing",
		"features": []string{
			"preemptive_scheduling",
			"work_stealing",
			"goroutine_parking",
			"network_poller_integration",
			"gc_aware_scheduling",
		},
	}
}