package main

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"time"
)

// MeasurementResult holds the results of a performance measurement
type MeasurementResult struct {
	Duration     time.Duration
	AllocBytes   uint64
	AllocObjects uint64
	GCCycles     uint32
}

// StatisticalResult holds statistical analysis of multiple measurements
type StatisticalResult struct {
	Mean     float64
	Median   float64
	StdDev   float64
	Min      float64
	Max      float64
	P95      float64
	P99      float64
	Samples  int
	Outliers int
}

// EnvironmentInfo holds information about the measurement environment
type EnvironmentInfo struct {
	CPUCount       int
	GOMAXPROCS     int
	Goroutines     int
	MemoryPressure bool
	SystemLoad     float64
	CPUScaling     bool
}

// MeasureExecutionTime measures the execution time of a function with high precision
func MeasureExecutionTime(fn func()) MeasurementResult {
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	fn()
	duration := time.Since(start)

	runtime.ReadMemStats(&memAfter)

	return MeasurementResult{
		Duration:     duration,
		AllocBytes:   memAfter.TotalAlloc - memBefore.TotalAlloc,
		AllocObjects: memAfter.Mallocs - memBefore.Mallocs,
		GCCycles:     memAfter.NumGC - memBefore.NumGC,
	}
}

// MeasureMemoryUsage measures memory allocation of a function
func MeasureMemoryUsage(fn func()) (allocBytes, allocObjects uint64) {
	var memBefore, memAfter runtime.MemStats

	// Force GC to get clean baseline
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	fn()

	runtime.ReadMemStats(&memAfter)

	return memAfter.TotalAlloc - memBefore.TotalAlloc,
		memAfter.Mallocs - memBefore.Mallocs
}

// MeasureWithWarmup performs measurements with proper warmup to ensure stable results
func MeasureWithWarmup(fn func(), warmupRuns, measureRuns int) []MeasurementResult {
	// Warmup phase - run function multiple times to stabilize performance
	for i := 0; i < warmupRuns; i++ {
		fn()
	}

	// Force GC before measurements
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	// Measurement phase
	results := make([]MeasurementResult, measureRuns)
	for i := 0; i < measureRuns; i++ {
		results[i] = MeasureExecutionTime(fn)
		// Small delay between measurements to avoid interference
		time.Sleep(time.Millisecond)
	}

	return results
}

// StatisticalAnalysis performs statistical analysis on measurement results
func StatisticalAnalysis(results []MeasurementResult) StatisticalResult {
	if len(results) == 0 {
		return StatisticalResult{}
	}

	// Extract durations in nanoseconds for analysis
	durations := make([]float64, len(results))
	for i, result := range results {
		durations[i] = float64(result.Duration.Nanoseconds())
	}

	// Sort for percentile calculations
	sort.Float64s(durations)

	// Calculate mean
	var sum float64
	for _, d := range durations {
		sum += d
	}
	mean := sum / float64(len(durations))

	// Calculate standard deviation
	var variance float64
	for _, d := range durations {
		variance += math.Pow(d-mean, 2)
	}
	stdDev := math.Sqrt(variance / float64(len(durations)))

	// Calculate percentiles
	median := percentile(durations, 50)
	p95 := percentile(durations, 95)
	p99 := percentile(durations, 99)

	// Detect outliers (values beyond 2 standard deviations)
	outliers := 0
	for _, d := range durations {
		if math.Abs(d-mean) > 2*stdDev {
			outliers++
		}
	}

	return StatisticalResult{
		Mean:     mean,
		Median:   median,
		StdDev:   stdDev,
		Min:      durations[0],
		Max:      durations[len(durations)-1],
		P95:      p95,
		P99:      p99,
		Samples:  len(durations),
		Outliers: outliers,
	}
}

// percentile calculates the given percentile of a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// ValidateEnvironment checks the measurement environment for optimal conditions
func ValidateEnvironment() EnvironmentInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Check memory pressure (if heap is close to system memory)
	memoryPressure := float64(memStats.Sys) > float64(memStats.HeapSys)*0.8

	// Simple system load estimation based on GC pressure
	systemLoad := memStats.GCCPUFraction * 100

	// CPU scaling detection (simplified - in real scenarios, would check /proc/cpuinfo on Linux)
	cpuScaling := detectCPUScaling()

	return EnvironmentInfo{
		CPUCount:       runtime.NumCPU(),
		GOMAXPROCS:     runtime.GOMAXPROCS(0),
		Goroutines:     runtime.NumGoroutine(),
		MemoryPressure: memoryPressure,
		SystemLoad:     systemLoad,
		CPUScaling:     cpuScaling,
	}
}

// detectCPUScaling attempts to detect if CPU frequency scaling is enabled
func detectCPUScaling() bool {
	// On Windows, this is a simplified check
	// In production, you'd check power management settings
	return false // Assume scaling is disabled for measurement accuracy
}

// PrintEnvironmentReport prints a detailed environment report
func PrintEnvironmentReport(env EnvironmentInfo) {
	fmt.Println("=== Environment Validation Report ===")
	fmt.Printf("CPU Count: %d\n", env.CPUCount)
	fmt.Printf("GOMAXPROCS: %d\n", env.GOMAXPROCS)
	fmt.Printf("Active Goroutines: %d\n", env.Goroutines)
	fmt.Printf("Memory Pressure: %t\n", env.MemoryPressure)
	fmt.Printf("System Load: %.2f%%\n", env.SystemLoad)
	fmt.Printf("CPU Scaling Detected: %t\n", env.CPUScaling)

	if env.MemoryPressure {
		fmt.Println("⚠️  WARNING: High memory pressure detected - measurements may be unreliable")
	}
	if env.SystemLoad > 10 {
		fmt.Println("⚠️  WARNING: High system load detected - consider reducing background processes")
	}
	if env.CPUScaling {
		fmt.Println("⚠️  WARNING: CPU frequency scaling detected - disable for consistent measurements")
	}
	if env.Goroutines > 10 {
		fmt.Println("⚠️  WARNING: Many goroutines active - may affect measurement consistency")
	}
	fmt.Println()
}

// PrintMeasurementReport prints a comprehensive measurement report
func PrintMeasurementReport(name string, stats StatisticalResult) {
	fmt.Printf("=== %s Performance Report ===\n", name)
	fmt.Printf("Samples: %d\n", stats.Samples)
	fmt.Printf("Mean: %.2f μs\n", stats.Mean/1000)
	fmt.Printf("Median: %.2f μs\n", stats.Median/1000)
	fmt.Printf("Std Dev: %.2f μs\n", stats.StdDev/1000)
	fmt.Printf("Min: %.2f μs\n", stats.Min/1000)
	fmt.Printf("Max: %.2f μs\n", stats.Max/1000)
	fmt.Printf("95th Percentile: %.2f μs\n", stats.P95/1000)
	fmt.Printf("99th Percentile: %.2f μs\n", stats.P99/1000)
	fmt.Printf("Outliers: %d (%.1f%%)\n", stats.Outliers, float64(stats.Outliers)/float64(stats.Samples)*100)

	// Calculate confidence interval (95%)
	marginOfError := 1.96 * (stats.StdDev / math.Sqrt(float64(stats.Samples)))
	fmt.Printf("95%% Confidence Interval: %.2f ± %.2f μs\n", stats.Mean/1000, marginOfError/1000)
	fmt.Println()
}

// CompareResults compares two sets of measurement results
func CompareResults(name1 string, stats1 StatisticalResult, name2 string, stats2 StatisticalResult) {
	fmt.Printf("=== Performance Comparison: %s vs %s ===\n", name1, name2)

	speedup := stats1.Mean / stats2.Mean
	if speedup > 1 {
		fmt.Printf("%s is %.2fx faster than %s\n", name2, speedup, name1)
	} else {
		fmt.Printf("%s is %.2fx faster than %s\n", name1, 1/speedup, name2)
	}

	// Statistical significance test (simplified)
	pooledStdDev := math.Sqrt((stats1.StdDev*stats1.StdDev + stats2.StdDev*stats2.StdDev) / 2)
	tStatistic := math.Abs(stats1.Mean-stats2.Mean) / (pooledStdDev * math.Sqrt(2.0/float64(stats1.Samples)))

	fmt.Printf("T-statistic: %.2f\n", tStatistic)
	if tStatistic > 2.0 { // Simplified threshold
		fmt.Println("✅ Difference is statistically significant")
	} else {
		fmt.Println("❌ Difference is NOT statistically significant")
	}
	fmt.Println()
}

// Sample benchmark functions for demonstration

// cpuIntensiveTask simulates CPU-intensive work
func cpuIntensiveTask() {
	sum := 0
	for i := 0; i < 1000000; i++ {
		sum += i * i
	}
	_ = sum
}

// memoryIntensiveTask simulates memory allocation
func memoryIntensiveTask() {
	data := make([][]int, 100)
	for i := range data {
		data[i] = make([]int, 100)
		for j := range data[i] {
			data[i][j] = i * j
		}
	}
	_ = data
}

// stringConcatenationTask simulates string operations
func stringConcatenationTask() {
	var result string
	for i := 0; i < 1000; i++ {
		result += fmt.Sprintf("String %d ", i)
	}
	_ = result
}

// optimizedStringTask uses strings.Builder for efficient concatenation
func optimizedStringTask() {
	var builder strings.Builder
	for i := 0; i < 1000; i++ {
		builder.WriteString(fmt.Sprintf("String %d ", i))
	}
	_ = builder.String()
}

func main() {
	fmt.Println("=== Performance Measurement Methodology Demo ===")
	fmt.Println("This demonstrates systematic approaches to performance measurement")
	fmt.Println("and establishes reliable benchmarking practices.")
	fmt.Println()

	// Task 1 & 3: Environment Setup and Validation
	fmt.Println("=== Task 1 & 3: Environment Validation ===")
	env := ValidateEnvironment()
	PrintEnvironmentReport(env)

	// Task 2: Implement Measurement Functions
	fmt.Println("=== Task 2: Measurement Functions Demo ===")

	// Demonstrate MeasureExecutionTime
	fmt.Println("1. MeasureExecutionTime Demo:")
	result := MeasureExecutionTime(cpuIntensiveTask)
	fmt.Printf("   Duration: %v\n", result.Duration)
	fmt.Printf("   Allocated: %d bytes\n", result.AllocBytes)
	fmt.Printf("   Objects: %d\n", result.AllocObjects)
	fmt.Printf("   GC Cycles: %d\n", result.GCCycles)
	fmt.Println()

	// Demonstrate MeasureMemoryUsage
	fmt.Println("2. MeasureMemoryUsage Demo:")
	allocBytes, allocObjects := MeasureMemoryUsage(memoryIntensiveTask)
	fmt.Printf("   Memory allocated: %d bytes\n", allocBytes)
	fmt.Printf("   Objects allocated: %d\n", allocObjects)
	fmt.Println()

	// Task 4: Measurement Best Practices with Warmup
	fmt.Println("=== Task 4: Measurement Best Practices ===")
	fmt.Println("3. MeasureWithWarmup Demo (CPU Intensive Task):")
	cpuResults := MeasureWithWarmup(cpuIntensiveTask, 5, 20)
	cpuStats := StatisticalAnalysis(cpuResults)
	PrintMeasurementReport("CPU Intensive Task", cpuStats)

	fmt.Println("4. MeasureWithWarmup Demo (Memory Intensive Task):")
	memResults := MeasureWithWarmup(memoryIntensiveTask, 3, 15)
	memStats := StatisticalAnalysis(memResults)
	PrintMeasurementReport("Memory Intensive Task", memStats)

	// Task 5: Create Measurement Reports and Comparisons
	fmt.Println("=== Task 5: Performance Comparison Framework ===")
	fmt.Println("Comparing String Concatenation Methods:")

	// Measure naive string concatenation
	naiveResults := MeasureWithWarmup(stringConcatenationTask, 3, 10)
	naiveStats := StatisticalAnalysis(naiveResults)

	// Measure optimized string concatenation
	optimizedResults := MeasureWithWarmup(optimizedStringTask, 3, 10)
	optimizedStats := StatisticalAnalysis(optimizedResults)

	PrintMeasurementReport("Naive String Concatenation", naiveStats)
	PrintMeasurementReport("Optimized String Concatenation", optimizedStats)

	// Compare the results
	CompareResults("Naive", naiveStats, "Optimized", optimizedStats)

	// Final environment check
	fmt.Println("=== Final Environment Check ===")
	finalEnv := ValidateEnvironment()
	if finalEnv.Goroutines != env.Goroutines {
		fmt.Printf("⚠️  Goroutine count changed: %d -> %d\n", env.Goroutines, finalEnv.Goroutines)
	}
	if math.Abs(finalEnv.SystemLoad-env.SystemLoad) > 5 {
		fmt.Printf("⚠️  System load changed significantly: %.2f%% -> %.2f%%\n", env.SystemLoad, finalEnv.SystemLoad)
	}

	fmt.Println("\n=== Key Learning Points Demonstrated ===")
	fmt.Println("✅ Environment validation and monitoring")
	fmt.Println("✅ High-precision timing measurements")
	fmt.Println("✅ Memory allocation tracking")
	fmt.Println("✅ Proper warmup procedures")
	fmt.Println("✅ Statistical analysis of results")
	fmt.Println("✅ Confidence intervals and significance testing")
	fmt.Println("✅ Outlier detection")
	fmt.Println("✅ Performance comparison framework")
	fmt.Println("✅ Measurement reproducibility")

	fmt.Println("\n=== Next Steps ===")
	fmt.Println("With solid measurement methodology established, you can now:")
	fmt.Println("- Apply these techniques to your own code")
	fmt.Println("- Create reliable performance benchmarks")
	fmt.Println("- Make data-driven optimization decisions")
	fmt.Println("- Proceed to Chapter 02: Go Runtime and Memory Management")
}
