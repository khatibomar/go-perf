package main

import (
	"fmt"
	"math"
)

// GoroutineLeakAnalyzer detects goroutine leaks
type GoroutineLeakAnalyzer struct{}

func (gla *GoroutineLeakAnalyzer) Name() string {
	return "GoroutineLeakAnalyzer"
}

func (gla *GoroutineLeakAnalyzer) Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection {
	var detections []LeakDetection

	if len(snapshots) < 3 {
		return detections
	}

	// If no baseline provided, create a simple one from early snapshots
	if baseline == nil {
		baseline = gla.createSimpleBaseline(snapshots)
	}

	// Check for sustained goroutine growth
	latest := snapshots[len(snapshots)-1]
	previous := snapshots[len(snapshots)-2]

	// Calculate goroutine growth rate
	growthRate := float64(latest.Goroutines-previous.Goroutines) / previous.Timestamp.Sub(latest.Timestamp).Seconds()

	// Check if current goroutine count significantly exceeds baseline
	if latest.Goroutines > baseline.AvgGoroutines {
		excess := latest.Goroutines - baseline.AvgGoroutines
		percentageIncrease := float64(excess) / float64(baseline.AvgGoroutines) * 100

		var severity LeakSeverity
		switch {
		case percentageIncrease > 200: // 200% increase
			severity = LeakSeverityCritical
		case percentageIncrease > 100: // 100% increase
			severity = LeakSeverityHigh
		case percentageIncrease > 50: // 50% increase
			severity = LeakSeverityMedium
		default:
			severity = LeakSeverityLow
		}

		if percentageIncrease > 25 { // Only report if > 25% increase
			detection := LeakDetection{
				ID:          generateLeakID(LeakTypeGoroutine, latest.Timestamp),
				Type:        LeakTypeGoroutine,
				Severity:    severity,
				Description: fmt.Sprintf("Goroutine count increased by %.1f%% (from %d to %d)", percentageIncrease, baseline.AvgGoroutines, latest.Goroutines),
				DetectedAt:  latest.Timestamp,
				Metrics: map[string]interface{}{
					"current_goroutines":  latest.Goroutines,
					"baseline_goroutines": baseline.AvgGoroutines,
					"percentage_increase": percentageIncrease,
					"growth_rate_per_sec": growthRate,
				},
				StackTrace: generateStackTrace(),
				Suggestion: "Check for goroutines that are not properly terminated. Look for blocked channels, infinite loops, or missing context cancellation.",
			}
			detections = append(detections, detection)
		}
	}

	// Check for rapid goroutine growth
	if len(snapshots) >= 5 {
		recentSnapshots := snapshots[len(snapshots)-5:]
		if gla.detectRapidGrowth(recentSnapshots) {
			detection := LeakDetection{
				ID:          generateLeakID(LeakTypeGoroutine, latest.Timestamp) + "_rapid",
				Type:        LeakTypeGoroutine,
				Severity:    LeakSeverityHigh,
				Description: "Rapid goroutine growth detected",
				DetectedAt:  latest.Timestamp,
				Metrics: map[string]interface{}{
					"growth_pattern": "rapid",
					"sample_count":   len(recentSnapshots),
				},
				StackTrace: generateStackTrace(),
				Suggestion: "Investigate recent code changes that might be creating goroutines without proper cleanup.",
			}
			detections = append(detections, detection)
		}
	}

	return detections
}

func (gla *GoroutineLeakAnalyzer) detectRapidGrowth(snapshots []MemorySnapshot) bool {
	if len(snapshots) < 3 {
		return false
	}

	// Check if goroutines are consistently increasing
	increasingCount := 0
	for i := 1; i < len(snapshots); i++ {
		if snapshots[i].Goroutines > snapshots[i-1].Goroutines {
			increasingCount++
		}
	}

	// If more than 80% of samples show increase, consider it rapid growth
	return float64(increasingCount)/float64(len(snapshots)-1) > 0.8
}

func (gla *GoroutineLeakAnalyzer) createSimpleBaseline(snapshots []MemorySnapshot) *MemoryBaseline {
	// Use first few snapshots as baseline
	baselineCount := min(len(snapshots)/2, 3)
	if baselineCount < 1 {
		baselineCount = 1
	}
	
	var totalGoroutines int
	for i := 0; i < baselineCount; i++ {
		totalGoroutines += snapshots[i].Goroutines
	}
	
	return &MemoryBaseline{
		AvgGoroutines: totalGoroutines / baselineCount,
		EstablishedAt: snapshots[0].Timestamp,
		SampleCount:   baselineCount,
	}
}

// MemoryGrowthAnalyzer detects memory growth patterns
type MemoryGrowthAnalyzer struct{}

func (mga *MemoryGrowthAnalyzer) Name() string {
	return "MemoryGrowthAnalyzer"
}

func (mga *MemoryGrowthAnalyzer) Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection {
	var detections []LeakDetection

	if len(snapshots) < 3 {
		return detections
	}

	// If no baseline provided, create a simple one from early snapshots
	if baseline == nil {
		baseline = mga.createSimpleBaseline(snapshots)
	}

	latest := snapshots[len(snapshots)-1]

	// Check for memory growth beyond baseline
	if latest.Alloc > baseline.AvgAlloc {
		excess := latest.Alloc - baseline.AvgAlloc
		percentageIncrease := float64(excess) / float64(baseline.AvgAlloc) * 100

		// Check if growth is beyond normal variance (using standard deviation)
		threshold := baseline.AvgAlloc + uint64(2*baseline.StdDevAlloc) // 2 standard deviations

		if latest.Alloc > threshold {
			var severity LeakSeverity
			switch {
			case percentageIncrease > 300: // 300% increase
				severity = LeakSeverityCritical
			case percentageIncrease > 150: // 150% increase
				severity = LeakSeverityHigh
			case percentageIncrease > 75: // 75% increase
				severity = LeakSeverityMedium
			default:
				severity = LeakSeverityLow
			}

			detection := LeakDetection{
				ID:          generateLeakID(LeakTypeMemory, latest.Timestamp),
				Type:        LeakTypeMemory,
				Severity:    severity,
				Description: fmt.Sprintf("Memory usage increased by %.1f%% beyond baseline", percentageIncrease),
				DetectedAt:  latest.Timestamp,
				Metrics: map[string]interface{}{
					"current_alloc":       latest.Alloc,
					"baseline_alloc":      baseline.AvgAlloc,
					"percentage_increase": percentageIncrease,
					"threshold":           threshold,
					"std_dev":             baseline.StdDevAlloc,
				},
				StackTrace: generateStackTrace(),
				Suggestion: "Check for memory leaks in data structures, unclosed resources, or accumulating caches.",
			}
			detections = append(detections, detection)
		}
	}

	// Check for sustained memory growth trend
	if len(snapshots) >= 10 {
		if mga.detectSustainedGrowth(snapshots[len(snapshots)-10:]) {
			detection := LeakDetection{
				ID:          generateLeakID(LeakTypeMemory, latest.Timestamp) + "_sustained",
				Type:        LeakTypeMemory,
				Severity:    LeakSeverityMedium,
				Description: "Sustained memory growth pattern detected",
				DetectedAt:  latest.Timestamp,
				Metrics: map[string]interface{}{
					"pattern": "sustained_growth",
					"samples": 10,
				},
				StackTrace: generateStackTrace(),
				Suggestion: "Monitor memory allocation patterns and check for gradual leaks in long-running operations.",
			}
			detections = append(detections, detection)
		}
	}

	return detections
}

func (mga *MemoryGrowthAnalyzer) detectSustainedGrowth(snapshots []MemorySnapshot) bool {
	if len(snapshots) < 5 {
		return false
	}

	// Calculate linear regression to detect trend
	n := float64(len(snapshots))
	var sumX, sumY, sumXY, sumX2 float64

	for i, snapshot := range snapshots {
		x := float64(i)
		y := float64(snapshot.Alloc)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope (trend)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// If slope is positive and significant, we have sustained growth
	return slope > 0 && slope > float64(snapshots[0].Alloc)*0.01 // 1% growth per sample
}

func (mga *MemoryGrowthAnalyzer) createSimpleBaseline(snapshots []MemorySnapshot) *MemoryBaseline {
	// Use first few snapshots as baseline
	baselineCount := min(len(snapshots)/2, 3)
	if baselineCount < 1 {
		baselineCount = 1
	}
	
	var totalAlloc uint64
	for i := 0; i < baselineCount; i++ {
		totalAlloc += snapshots[i].Alloc
	}
	
	return &MemoryBaseline{
		AvgAlloc:      totalAlloc / uint64(baselineCount),
		StdDevAlloc:   float64(totalAlloc) * 0.1, // Simple 10% variance
		EstablishedAt: snapshots[0].Timestamp,
		SampleCount:   baselineCount,
	}
}

// HeapObjectAnalyzer detects heap object accumulation
type HeapObjectAnalyzer struct{}

func (hoa *HeapObjectAnalyzer) Name() string {
	return "HeapObjectAnalyzer"
}

func (hoa *HeapObjectAnalyzer) Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection {
	var detections []LeakDetection

	if len(snapshots) < 3 {
		return detections
	}

	// If no baseline provided, create a simple one from early snapshots
	if baseline == nil {
		baseline = hoa.createSimpleBaseline(snapshots)
	}

	latest := snapshots[len(snapshots)-1]

	// Check for excessive heap objects
	if latest.HeapObjects > baseline.AvgHeapObjects {
		excess := latest.HeapObjects - baseline.AvgHeapObjects
		percentageIncrease := float64(excess) / float64(baseline.AvgHeapObjects) * 100

		if percentageIncrease > 50 { // 50% increase in objects
			var severity LeakSeverity
			switch {
			case percentageIncrease > 200:
				severity = LeakSeverityCritical
			case percentageIncrease > 100:
				severity = LeakSeverityHigh
			default:
				severity = LeakSeverityMedium
			}

			detection := LeakDetection{
				ID:          generateLeakID(LeakTypeHeapObject, latest.Timestamp) + "_objects",
				Type:        LeakTypeHeapObject,
				Severity:    severity,
				Description: fmt.Sprintf("Heap objects increased by %.1f%% (from %d to %d)", percentageIncrease, baseline.AvgHeapObjects, latest.HeapObjects),
				DetectedAt:  latest.Timestamp,
				Metrics: map[string]interface{}{
					"current_objects":     latest.HeapObjects,
					"baseline_objects":    baseline.AvgHeapObjects,
					"percentage_increase": percentageIncrease,
				},
				StackTrace: generateStackTrace(),
				Suggestion: "Check for object accumulation in maps, slices, or caches that are not being properly cleaned up.",
			}
			detections = append(detections, detection)
		}
	}

	return detections
}

func (hoa *HeapObjectAnalyzer) createSimpleBaseline(snapshots []MemorySnapshot) *MemoryBaseline {
	// Use first few snapshots as baseline
	baselineCount := min(len(snapshots)/2, 3)
	if baselineCount < 1 {
		baselineCount = 1
	}
	
	var totalHeapObjects uint64
	for i := 0; i < baselineCount; i++ {
		totalHeapObjects += snapshots[i].HeapObjects
	}
	
	return &MemoryBaseline{
		AvgHeapObjects: totalHeapObjects / uint64(baselineCount),
		EstablishedAt:  snapshots[0].Timestamp,
		SampleCount:    baselineCount,
	}
}

// GCPressureAnalyzer detects garbage collection pressure
type GCPressureAnalyzer struct{}

func (gpa *GCPressureAnalyzer) Name() string {
	return "GCPressureAnalyzer"
}

func (gpa *GCPressureAnalyzer) Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection {
	var detections []LeakDetection

	if len(snapshots) < 5 {
		return detections
	}

	latest := snapshots[len(snapshots)-1]
	previous := snapshots[len(snapshots)-2]

	// Check GC CPU fraction
	if latest.GCCPUFraction > 0.1 { // More than 10% CPU spent on GC
		var severity LeakSeverity
		switch {
		case latest.GCCPUFraction > 0.3: // 30%
			severity = LeakSeverityCritical
		case latest.GCCPUFraction > 0.2: // 20%
			severity = LeakSeverityHigh
		default:
			severity = LeakSeverityMedium
		}

		detection := LeakDetection{
			ID:          generateLeakID(LeakTypeGCPressure, latest.Timestamp) + "_gc_pressure",
			Type:        LeakTypeGCPressure,
			Severity:    severity,
			Description: fmt.Sprintf("High GC pressure detected: %.1f%% CPU time spent on GC", latest.GCCPUFraction*100),
			DetectedAt:  latest.Timestamp,
			Metrics: map[string]interface{}{
				"gc_cpu_fraction": latest.GCCPUFraction,
				"num_gc":          latest.NumGC,
				"gc_rate":         float64(latest.NumGC-previous.NumGC) / latest.Timestamp.Sub(previous.Timestamp).Seconds(),
			},
			StackTrace: generateStackTrace(),
			Suggestion: "High GC pressure may indicate memory allocation patterns that stress the garbage collector. Consider optimizing allocation patterns or increasing heap size.",
		}
		detections = append(detections, detection)
	}

	// Check for rapid GC cycles
	if len(snapshots) >= 5 {
		recentSnapshots := snapshots[len(snapshots)-5:]
		if gpa.detectRapidGCCycles(recentSnapshots) {
			detection := LeakDetection{
				ID:          generateLeakID(LeakTypeGCPressure, latest.Timestamp) + "_rapid_gc",
				Type:        LeakTypeGCPressure,
				Severity:    LeakSeverityMedium,
				Description: "Rapid GC cycles detected",
				DetectedAt:  latest.Timestamp,
				Metrics: map[string]interface{}{
					"pattern": "rapid_gc_cycles",
					"samples": len(recentSnapshots),
				},
				StackTrace: generateStackTrace(),
				Suggestion: "Frequent GC cycles may indicate short-lived object allocation patterns. Consider object pooling or reducing allocation frequency.",
			}
			detections = append(detections, detection)
		}
	}

	return detections
}

func (gpa *GCPressureAnalyzer) detectRapidGCCycles(snapshots []MemorySnapshot) bool {
	if len(snapshots) < 3 {
		return false
	}

	// Calculate average GC rate
	totalDuration := snapshots[len(snapshots)-1].Timestamp.Sub(snapshots[0].Timestamp)
	totalGCCycles := snapshots[len(snapshots)-1].NumGC - snapshots[0].NumGC

	gcRate := float64(totalGCCycles) / totalDuration.Seconds()

	// If more than 1 GC per second on average, consider it rapid
	return gcRate > 1.0
}

// StackAnalyzer detects stack-related issues
type StackAnalyzer struct{}

func (sa *StackAnalyzer) Name() string {
	return "StackAnalyzer"
}

func (sa *StackAnalyzer) Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection {
	var detections []LeakDetection

	if len(snapshots) < 3 {
		return detections
	}

	latest := snapshots[len(snapshots)-1]

	// Check for excessive stack usage
	if latest.StackInUse > 50*1024*1024 { // 50MB stack usage
		var severity LeakSeverity
		switch {
		case latest.StackInUse > 200*1024*1024: // 200MB
			severity = LeakSeverityCritical
		case latest.StackInUse > 100*1024*1024: // 100MB
			severity = LeakSeverityHigh
		default:
			severity = LeakSeverityMedium
		}

		detection := LeakDetection{
			ID:          generateLeakID(LeakTypeMemory, latest.Timestamp) + "_stack",
			Type:        LeakTypeMemory,
			Severity:    severity,
			Description: fmt.Sprintf("High stack usage detected: %d bytes", latest.StackInUse),
			DetectedAt:  latest.Timestamp,
			Metrics: map[string]interface{}{
				"stack_in_use": latest.StackInUse,
				"goroutines":   latest.Goroutines,
			},
			StackTrace: generateStackTrace(),
			Suggestion: "High stack usage may indicate deep recursion or many goroutines with large stacks. Check for recursive functions or goroutine leaks.",
		}
		detections = append(detections, detection)
	}

	return detections
}

// CompositeAnalyzer combines multiple analysis strategies
type CompositeAnalyzer struct {
	analyzers []LeakAnalyzer
}

func NewCompositeAnalyzer() *CompositeAnalyzer {
	return &CompositeAnalyzer{
		analyzers: []LeakAnalyzer{
			&GoroutineLeakAnalyzer{},
			&MemoryGrowthAnalyzer{},
			&HeapObjectAnalyzer{},
			&GCPressureAnalyzer{},
			&StackAnalyzer{},
		},
	}
}

func (ca *CompositeAnalyzer) Name() string {
	return "CompositeAnalyzer"
}

func (ca *CompositeAnalyzer) Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection {
	var allDetections []LeakDetection

	for _, analyzer := range ca.analyzers {
		detections := analyzer.Analyze(snapshots, baseline)
		allDetections = append(allDetections, detections...)
	}

	return allDetections
}

// StatisticalAnalyzer uses statistical methods for leak detection
type StatisticalAnalyzer struct {
	confidenceLevel float64
}

func NewStatisticalAnalyzer(confidenceLevel float64) *StatisticalAnalyzer {
	return &StatisticalAnalyzer{
		confidenceLevel: confidenceLevel,
	}
}

func (sa *StatisticalAnalyzer) Name() string {
	return "StatisticalAnalyzer"
}

func (sa *StatisticalAnalyzer) Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection {
	var detections []LeakDetection

	if len(snapshots) < 10 || baseline == nil {
		return detections
	}

	// Use statistical analysis to detect anomalies
	if sa.detectStatisticalAnomaly(snapshots, baseline) {
		latest := snapshots[len(snapshots)-1]
		detection := LeakDetection{
			ID:          generateLeakID(LeakTypeMemory, latest.Timestamp) + "_statistical",
			Type:        LeakTypeMemory,
			Severity:    LeakSeverityMedium,
			Description: "Statistical anomaly detected in memory usage pattern",
			DetectedAt:  latest.Timestamp,
			Metrics: map[string]interface{}{
				"confidence_level": sa.confidenceLevel,
				"analysis_type":    "statistical",
			},
			StackTrace: generateStackTrace(),
			Suggestion: "Memory usage pattern deviates significantly from established baseline. Investigate recent changes or unusual workload patterns.",
		}
		detections = append(detections, detection)
	}

	return detections
}

func (sa *StatisticalAnalyzer) detectStatisticalAnomaly(snapshots []MemorySnapshot, baseline *MemoryBaseline) bool {
	// Calculate z-score for latest memory usage
	latest := snapshots[len(snapshots)-1]
	zScore := math.Abs(float64(latest.Alloc)-float64(baseline.AvgAlloc)) / baseline.StdDevAlloc

	// Check if z-score exceeds confidence level threshold
	// For 95% confidence level, threshold is approximately 1.96
	// For 99% confidence level, threshold is approximately 2.58
	threshold := 1.96 // 95% confidence
	if sa.confidenceLevel > 0.98 {
		threshold = 2.58 // 99% confidence
	}

	return zScore > threshold
}
