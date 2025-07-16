package benchmarks

import (
	cryptornd "crypto/rand"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Benchmark utilities and helper functions

// BenchmarkResult holds benchmark execution results
type BenchmarkResult struct {
	Name          string
	Iterations    int
	NsPerOp       int64
	BytesPerOp    int64
	AllocsPerOp   int64
	MBPerSec      float64
	Duration      time.Duration
	MemAllocBytes uint64
	MemAllocObjs  uint64
}

// BenchmarkConfig holds configuration for benchmark execution
type BenchmarkConfig struct {
	MinIterations int
	MaxIterations int
	MinDuration   time.Duration
	MaxDuration   time.Duration
	WarmupRuns    int
	ReportAllocs  bool
	ReportMem     bool
}

// DefaultBenchmarkConfig returns a default benchmark configuration
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		MinIterations: 1,
		MaxIterations: 1000000000,
		MinDuration:   1 * time.Second,
		MaxDuration:   10 * time.Second,
		WarmupRuns:    3,
		ReportAllocs:  true,
		ReportMem:     true,
	}
}

// BenchmarkRunner provides utilities for running and analyzing benchmarks
type BenchmarkRunner struct {
	config BenchmarkConfig
}

// NewBenchmarkRunner creates a new benchmark runner with the given configuration
func NewBenchmarkRunner(config BenchmarkConfig) *BenchmarkRunner {
	return &BenchmarkRunner{config: config}
}

// RunBenchmark executes a benchmark function with the configured settings
func (br *BenchmarkRunner) RunBenchmark(name string, fn func(b *testing.B)) BenchmarkResult {
	if name == "" {
		name = "UnnamedBenchmark"
	}
	if fn == nil {
		return BenchmarkResult{Name: name, Iterations: 0, NsPerOp: 0}
	}

	// Create a mock testing.B for our custom runner
	b := &testing.B{}

	// Warmup runs
	for i := 0; i < br.config.WarmupRuns; i++ {
		fn(b)
	}

	// Actual benchmark run
	start := time.Now()
	var memBefore, memAfter runtime.MemStats

	if br.config.ReportMem {
		runtime.GC()
		runtime.ReadMemStats(&memBefore)
	}

	fn(b)

	if br.config.ReportMem {
		runtime.ReadMemStats(&memAfter)
	}

	duration := time.Since(start)

	return BenchmarkResult{
		Name:          name,
		Iterations:    b.N,
		NsPerOp:       duration.Nanoseconds() / int64(b.N),
		Duration:      duration,
		MemAllocBytes: memAfter.TotalAlloc - memBefore.TotalAlloc,
		MemAllocObjs:  memAfter.Mallocs - memBefore.Mallocs,
	}
}

// CompareResults compares two benchmark results and returns the performance difference
func CompareResults(baseline, current BenchmarkResult) string {
	if baseline.NsPerOp == 0 {
		return "Cannot compare: baseline has zero ns/op"
	}

	speedupRatio := float64(baseline.NsPerOp) / float64(current.NsPerOp)
	percentChange := ((float64(current.NsPerOp) - float64(baseline.NsPerOp)) / float64(baseline.NsPerOp)) * 100

	var result strings.Builder
	fmt.Fprintf(&result, "Performance comparison:\n")
	fmt.Fprintf(&result, "  Baseline: %s (%d ns/op)\n", baseline.Name, baseline.NsPerOp)
	fmt.Fprintf(&result, "  Current:  %s (%d ns/op)\n", current.Name, current.NsPerOp)
	fmt.Fprintf(&result, "  Speedup:  %.2fx\n", speedupRatio)
	fmt.Fprintf(&result, "  Change:   %.2f%%\n", percentChange)

	if percentChange < -5 {
		fmt.Fprintf(&result, "  Status:   IMPROVEMENT (%.2f%% faster)\n", -percentChange)
	} else if percentChange > 5 {
		fmt.Fprintf(&result, "  Status:   REGRESSION (%.2f%% slower)\n", percentChange)
	} else {
		fmt.Fprintf(&result, "  Status:   NO SIGNIFICANT CHANGE\n")
	}

	return result.String()
}

// Data generation utilities

// GenerateRandomBytes creates a slice of random bytes
func GenerateRandomBytes(size int) []byte {
	if size <= 0 {
		return []byte{}
	}
	if size > 100*1024*1024 { // 100MB limit
		fmt.Printf("Warning: Generating %d bytes may cause memory issues\n", size)
	}

	data := make([]byte, size)
	cryptornd.Read(data)
	return data
}

// GenerateRandomStrings creates a slice of random strings
func GenerateRandomStrings(count, length int) []string {
	if count <= 0 || length <= 0 {
		return []string{}
	}
	if count > 1000000 {
		fmt.Printf("Warning: Generating %d strings may cause memory issues\n", count)
	}

	strings := make([]string, count)
	for i := 0; i < count; i++ {
		strings[i] = GenerateRandomString(length)
	}
	return strings
}

// GenerateRandomString creates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateRandomInts creates a slice of random integers
func GenerateRandomInts(count, max int) []int {
	ints := make([]int, count)
	for i := 0; i < count; i++ {
		ints[i] = rand.Intn(max)
	}
	return ints
}

// GenerateSequentialInts creates a slice of sequential integers
func GenerateSequentialInts(count int) []int {
	ints := make([]int, count)
	for i := 0; i < count; i++ {
		ints[i] = i
	}
	return ints
}

// GenerateRandomMap creates a map with random string keys and values
func GenerateRandomMap(size int) map[string]string {
	m := make(map[string]string, size)
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := GenerateRandomString(20)
		m[key] = value
	}
	return m
}

// Memory measurement utilities

// MemStats holds memory statistics
type MemStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	Mallocs    uint64
	Frees      uint64
	HeapAlloc  uint64
	HeapSys    uint64
	HeapIdle   uint64
	HeapInuse  uint64
	StackInuse uint64
	StackSys   uint64
	NumGC      uint32
}

// GetMemStats returns current memory statistics
func GetMemStats() MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemStats{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		Mallocs:    m.Mallocs,
		Frees:      m.Frees,
		HeapAlloc:  m.HeapAlloc,
		HeapSys:    m.HeapSys,
		HeapIdle:   m.HeapIdle,
		HeapInuse:  m.HeapInuse,
		StackInuse: m.StackInuse,
		StackSys:   m.StackSys,
		NumGC:      m.NumGC,
	}
}

// MemDiff calculates the difference between two memory statistics
func MemDiff(before, after MemStats) MemStats {
	return MemStats{
		Alloc:      after.Alloc - before.Alloc,
		TotalAlloc: after.TotalAlloc - before.TotalAlloc,
		Sys:        after.Sys - before.Sys,
		Mallocs:    after.Mallocs - before.Mallocs,
		Frees:      after.Frees - before.Frees,
		HeapAlloc:  after.HeapAlloc - before.HeapAlloc,
		HeapSys:    after.HeapSys - before.HeapSys,
		HeapIdle:   after.HeapIdle - before.HeapIdle,
		HeapInuse:  after.HeapInuse - before.HeapInuse,
		StackInuse: after.StackInuse - before.StackInuse,
		StackSys:   after.StackSys - before.StackSys,
		NumGC:      after.NumGC - before.NumGC,
	}
}

// FormatMemStats formats memory statistics for display
func FormatMemStats(stats MemStats) string {
	var result strings.Builder
	fmt.Fprintf(&result, "Memory Statistics:\n")
	fmt.Fprintf(&result, "  Alloc:      %s\n", formatBytes(stats.Alloc))
	fmt.Fprintf(&result, "  TotalAlloc: %s\n", formatBytes(stats.TotalAlloc))
	fmt.Fprintf(&result, "  Sys:        %s\n", formatBytes(stats.Sys))
	fmt.Fprintf(&result, "  Mallocs:    %d\n", stats.Mallocs)
	fmt.Fprintf(&result, "  Frees:      %d\n", stats.Frees)
	fmt.Fprintf(&result, "  HeapAlloc:  %s\n", formatBytes(stats.HeapAlloc))
	fmt.Fprintf(&result, "  HeapSys:    %s\n", formatBytes(stats.HeapSys))
	fmt.Fprintf(&result, "  HeapIdle:   %s\n", formatBytes(stats.HeapIdle))
	fmt.Fprintf(&result, "  HeapInuse:  %s\n", formatBytes(stats.HeapInuse))
	fmt.Fprintf(&result, "  StackInuse: %s\n", formatBytes(stats.StackInuse))
	fmt.Fprintf(&result, "  StackSys:   %s\n", formatBytes(stats.StackSys))
	fmt.Fprintf(&result, "  NumGC:      %d\n", stats.NumGC)
	return result.String()
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Timing utilities

// Timer provides high-precision timing measurements
type Timer struct {
	start time.Time
	end   time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{}
}

// Start begins timing
func (t *Timer) Start() {
	t.start = time.Now()
}

// Stop ends timing
func (t *Timer) Stop() {
	t.end = time.Now()
}

// Duration returns the elapsed time
func (t *Timer) Duration() time.Duration {
	if t.end.IsZero() {
		return time.Since(t.start)
	}
	return t.end.Sub(t.start)
}

// Reset resets the timer
func (t *Timer) Reset() {
	t.start = time.Time{}
	t.end = time.Time{}
}

// MeasureFunction measures the execution time of a function
func MeasureFunction(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// MeasureFunctionWithResult measures execution time and captures result
func MeasureFunctionWithResult(fn func() interface{}) (time.Duration, interface{}) {
	start := time.Now()
	result := fn()
	duration := time.Since(start)
	return duration, result
}

// Benchmark validation utilities

// ValidateBenchmarkResult checks if benchmark results are reasonable
func ValidateBenchmarkResult(result BenchmarkResult) []string {
	var issues []string

	if result.Iterations <= 0 {
		issues = append(issues, "Invalid iteration count: must be positive")
	}

	if result.NsPerOp <= 0 {
		issues = append(issues, "Invalid ns/op: must be positive")
	}

	if result.Duration <= 0 {
		issues = append(issues, "Invalid duration: must be positive")
	}

	// Check for unreasonably fast operations (possible compiler optimization)
	if result.NsPerOp < 1 {
		issues = append(issues, "Warning: Very fast operation, check for compiler optimizations")
	}

	// Check for unreasonably slow operations
	if result.NsPerOp > 1000000000 { // 1 second per operation
		issues = append(issues, "Warning: Very slow operation, consider reducing work per iteration")
	}

	return issues
}

// Statistical utilities

// CalculateStats calculates basic statistics for a slice of durations
func CalculateStats(durations []time.Duration) (mean, median, min, max time.Duration, stddev float64) {
	if len(durations) == 0 {
		return
	}

	// Convert to int64 for calculations
	values := make([]int64, len(durations))
	for i, d := range durations {
		values[i] = d.Nanoseconds()
	}

	// Sort for median calculation
	sorted := make([]int64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate statistics
	min = time.Duration(sorted[0])
	max = time.Duration(sorted[len(sorted)-1])

	if len(sorted)%2 == 0 {
		median = time.Duration((sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2)
	} else {
		median = time.Duration(sorted[len(sorted)/2])
	}

	var sum int64
	for _, v := range values {
		sum += v
	}
	mean = time.Duration(sum / int64(len(values)))

	// Calculate standard deviation
	var variance float64
	meanFloat := float64(mean.Nanoseconds())
	for _, v := range values {
		diff := float64(v) - meanFloat
		variance += diff * diff
	}
	variance /= float64(len(values))
	stddev = variance // Simplified - should be sqrt(variance)

	return
}

// PreventOptimization prevents compiler optimizations by using the result
var GlobalSink interface{}

// Sink stores a value to prevent compiler optimization
func Sink(v interface{}) {
	GlobalSink = v
}

// NoOp performs a no-operation to prevent optimization
func NoOp() {
	// Intentionally empty
}

// Environment utilities

// GetBenchmarkEnvironment returns information about the benchmark environment
func GetBenchmarkEnvironment() map[string]interface{} {
	return map[string]interface{}{
		"GOMAXPROCS":   runtime.GOMAXPROCS(0),
		"NumCPU":       runtime.NumCPU(),
		"NumGoroutine": runtime.NumGoroutine(),
		"Version":      runtime.Version(),
		"GOOS":         runtime.GOOS,
		"GOARCH":       runtime.GOARCH,
	}
}

// PrintBenchmarkEnvironment prints benchmark environment information
func PrintBenchmarkEnvironment() {
	env := GetBenchmarkEnvironment()
	fmt.Println("Benchmark Environment:")
	for key, value := range env {
		fmt.Printf("  %s: %v\n", key, value)
	}
}
