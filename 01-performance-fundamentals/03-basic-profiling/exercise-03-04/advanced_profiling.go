// Package main demonstrates advanced profiling techniques in Go
// This exercise covers custom profilers, trace analysis, and performance regression detection
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"
)

// CustomProfiler demonstrates custom profiling implementation
type CustomProfiler struct {
	mu       sync.RWMutex
	samples  []ProfileSample
	active   bool
	interval time.Duration
	stopCh   chan struct{}
}

// ProfileSample represents a single profiling sample
type ProfileSample struct {
	Timestamp   time.Time
	Goroutines  int
	MemoryAlloc uint64
	MemorySys   uint64
	NumGC       uint32
	CPUUsage    float64
	CustomData  map[string]interface{}
}

// NewCustomProfiler creates a new custom profiler
func NewCustomProfiler(interval time.Duration) *CustomProfiler {
	return &CustomProfiler{
		samples:  make([]ProfileSample, 0),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins custom profiling
func (cp *CustomProfiler) Start() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	if cp.active {
		return
	}
	
	cp.active = true
	go cp.collectSamples()
}

// Stop ends custom profiling
func (cp *CustomProfiler) Stop() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	if !cp.active {
		return
	}
	
	cp.active = false
	close(cp.stopCh)
}

// collectSamples runs the sampling loop
func (cp *CustomProfiler) collectSamples() {
	ticker := time.NewTicker(cp.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cp.takeSample()
		case <-cp.stopCh:
			return
		}
	}
}

// takeSample collects a single profiling sample
func (cp *CustomProfiler) takeSample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	sample := ProfileSample{
		Timestamp:   time.Now(),
		Goroutines:  runtime.NumGoroutine(),
		MemoryAlloc: m.Alloc,
		MemorySys:   m.Sys,
		NumGC:       m.NumGC,
		CPUUsage:    getCurrentCPUUsage(),
		CustomData:  make(map[string]interface{}),
	}
	
	// Add custom metrics
	sample.CustomData["heap_objects"] = m.HeapObjects
	sample.CustomData["gc_pause_ns"] = m.PauseNs[(m.NumGC+255)%256]
	sample.CustomData["stack_inuse"] = m.StackInuse
	
	cp.mu.Lock()
	cp.samples = append(cp.samples, sample)
	cp.mu.Unlock()
}

// getCurrentCPUUsage returns current CPU usage (simplified)
func getCurrentCPUUsage() float64 {
	// This is a simplified CPU usage calculation
	// In production, you'd use more sophisticated methods
	return rand.Float64() * 100
}

// GetSamples returns all collected samples
func (cp *CustomProfiler) GetSamples() []ProfileSample {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	
	samples := make([]ProfileSample, len(cp.samples))
	copy(samples, cp.samples)
	return samples
}

// ExportToFile exports samples to a file
func (cp *CustomProfiler) ExportToFile(filename string) error {
	samples := cp.GetSamples()
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	fmt.Fprintf(file, "timestamp,goroutines,memory_alloc,memory_sys,num_gc,cpu_usage,heap_objects,gc_pause_ns,stack_inuse\n")
	
	// Write samples
	for _, sample := range samples {
		fmt.Fprintf(file, "%s,%d,%d,%d,%d,%.2f,%v,%v,%v\n",
			sample.Timestamp.Format(time.RFC3339),
			sample.Goroutines,
			sample.MemoryAlloc,
			sample.MemorySys,
			sample.NumGC,
			sample.CPUUsage,
			sample.CustomData["heap_objects"],
			sample.CustomData["gc_pause_ns"],
			sample.CustomData["stack_inuse"],
		)
	}
	
	return nil
}

// TraceAnalyzer provides trace analysis capabilities
type TraceAnalyzer struct {
	traceFile string
	analysis  TraceAnalysis
}

// TraceAnalysis contains trace analysis results
type TraceAnalysis struct {
	Duration        time.Duration
	GoroutineEvents map[string]int
	GCEvents        []GCEvent
	SchedulerEvents []SchedulerEvent
	NetworkEvents   []NetworkEvent
	CustomEvents    []CustomEvent
}

// GCEvent represents a garbage collection event
type GCEvent struct {
	Timestamp time.Time
	Duration  time.Duration
	Type      string
	HeapSize  uint64
}

// SchedulerEvent represents a scheduler event
type SchedulerEvent struct {
	Timestamp   time.Time
	GoroutineID uint64
	Event       string
	Processor   int
}

// NetworkEvent represents a network-related event
type NetworkEvent struct {
	Timestamp time.Time
	Operation string
	Duration  time.Duration
	Bytes     int64
}

// CustomEvent represents application-specific events
type CustomEvent struct {
	Timestamp time.Time
	Name      string
	Data      map[string]interface{}
}

// NewTraceAnalyzer creates a new trace analyzer
func NewTraceAnalyzer(traceFile string) *TraceAnalyzer {
	return &TraceAnalyzer{
		traceFile: traceFile,
		analysis: TraceAnalysis{
			GoroutineEvents: make(map[string]int),
			GCEvents:        make([]GCEvent, 0),
			SchedulerEvents: make([]SchedulerEvent, 0),
			NetworkEvents:   make([]NetworkEvent, 0),
			CustomEvents:    make([]CustomEvent, 0),
		},
	}
}

// AnalyzeTrace performs trace analysis (simplified implementation)
func (ta *TraceAnalyzer) AnalyzeTrace() error {
	// In a real implementation, this would parse the trace file
	// and extract detailed timing and event information
	// For this exercise, we'll simulate the analysis
	
	// Simulate trace analysis
	ta.analysis.Duration = time.Minute * 5
	
	// Simulate goroutine events
	ta.analysis.GoroutineEvents["created"] = 150
	ta.analysis.GoroutineEvents["blocked"] = 45
	ta.analysis.GoroutineEvents["unblocked"] = 43
	ta.analysis.GoroutineEvents["ended"] = 148
	
	// Simulate GC events
	for i := 0; i < 10; i++ {
		ta.analysis.GCEvents = append(ta.analysis.GCEvents, GCEvent{
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Duration:  time.Millisecond * time.Duration(50+rand.Intn(100)),
			Type:      "mark",
			HeapSize:  uint64(1024*1024*10 + rand.Intn(1024*1024*5)),
		})
	}
	
	// Simulate scheduler events
	for i := 0; i < 100; i++ {
		ta.analysis.SchedulerEvents = append(ta.analysis.SchedulerEvents, SchedulerEvent{
			Timestamp:   time.Now().Add(-time.Duration(i) * time.Second),
			GoroutineID: uint64(rand.Intn(1000)),
			Event:       []string{"running", "runnable", "blocked"}[rand.Intn(3)],
			Processor:   rand.Intn(runtime.NumCPU()),
		})
	}
	
	return nil
}

// GetAnalysis returns the trace analysis results
func (ta *TraceAnalyzer) GetAnalysis() TraceAnalysis {
	return ta.analysis
}

// PrintSummary prints a summary of the trace analysis
func (ta *TraceAnalyzer) PrintSummary() {
	fmt.Printf("\n=== Trace Analysis Summary ===\n")
	fmt.Printf("Duration: %v\n", ta.analysis.Duration)
	fmt.Printf("\nGoroutine Events:\n")
	for event, count := range ta.analysis.GoroutineEvents {
		fmt.Printf("  %s: %d\n", event, count)
	}
	
	fmt.Printf("\nGC Events: %d\n", len(ta.analysis.GCEvents))
	if len(ta.analysis.GCEvents) > 0 {
		totalGCTime := time.Duration(0)
		for _, gc := range ta.analysis.GCEvents {
			totalGCTime += gc.Duration
		}
		avgGCTime := totalGCTime / time.Duration(len(ta.analysis.GCEvents))
		fmt.Printf("  Average GC Duration: %v\n", avgGCTime)
		fmt.Printf("  Total GC Time: %v\n", totalGCTime)
	}
	
	fmt.Printf("\nScheduler Events: %d\n", len(ta.analysis.SchedulerEvents))
	fmt.Printf("Network Events: %d\n", len(ta.analysis.NetworkEvents))
	fmt.Printf("Custom Events: %d\n", len(ta.analysis.CustomEvents))
}

// PerformanceRegression represents a performance regression detection system
type PerformanceRegression struct {
	baseline    map[string]float64
	current     map[string]float64
	thresholds  map[string]float64
	regressions []Regression
}

// Regression represents a detected performance regression
type Regression struct {
	Metric      string
	Baseline    float64
	Current     float64
	Change      float64
	ChangePercent float64
	Severity    string
}

// NewPerformanceRegression creates a new regression detector
func NewPerformanceRegression() *PerformanceRegression {
	return &PerformanceRegression{
		baseline:    make(map[string]float64),
		current:     make(map[string]float64),
		thresholds:  make(map[string]float64),
		regressions: make([]Regression, 0),
	}
}

// SetBaseline sets baseline performance metrics
func (pr *PerformanceRegression) SetBaseline(metrics map[string]float64) {
	pr.baseline = make(map[string]float64)
	for k, v := range metrics {
		pr.baseline[k] = v
	}
}

// SetCurrent sets current performance metrics
func (pr *PerformanceRegression) SetCurrent(metrics map[string]float64) {
	pr.current = make(map[string]float64)
	for k, v := range metrics {
		pr.current[k] = v
	}
}

// SetThresholds sets regression detection thresholds (percentage)
func (pr *PerformanceRegression) SetThresholds(thresholds map[string]float64) {
	pr.thresholds = make(map[string]float64)
	for k, v := range thresholds {
		pr.thresholds[k] = v
	}
}

// DetectRegressions analyzes metrics and detects regressions
func (pr *PerformanceRegression) DetectRegressions() []Regression {
	pr.regressions = make([]Regression, 0)
	
	for metric, currentValue := range pr.current {
		baselineValue, hasBaseline := pr.baseline[metric]
		threshold, hasThreshold := pr.thresholds[metric]
		
		if !hasBaseline {
			continue
		}
		
		if !hasThreshold {
			threshold = 10.0 // Default 10% threshold
		}
		
		change := currentValue - baselineValue
		changePercent := (change / baselineValue) * 100
		
		// Check if this is a regression (performance got worse)
		if isRegression(metric, changePercent, threshold) {
			severity := calculateSeverity(changePercent, threshold)
			
			regression := Regression{
				Metric:        metric,
				Baseline:      baselineValue,
				Current:       currentValue,
				Change:        change,
				ChangePercent: changePercent,
				Severity:      severity,
			}
			
			pr.regressions = append(pr.regressions, regression)
		}
	}
	
	return pr.regressions
}

// isRegression determines if a change represents a performance regression
func isRegression(metric string, changePercent, threshold float64) bool {
	// For latency/duration metrics, increase is bad
	latencyMetrics := []string{"latency", "duration", "response_time", "gc_time"}
	for _, lm := range latencyMetrics {
		if contains(metric, lm) {
			return changePercent > threshold
		}
	}
	
	// For throughput/rate metrics, decrease is bad
	throughputMetrics := []string{"throughput", "rate", "qps", "rps"}
	for _, tm := range throughputMetrics {
		if contains(metric, tm) {
			return changePercent < -threshold
		}
	}
	
	// For memory metrics, increase is usually bad
	memoryMetrics := []string{"memory", "alloc", "heap"}
	for _, mm := range memoryMetrics {
		if contains(metric, mm) {
			return changePercent > threshold
		}
	}
	
	// Default: any significant change is a potential regression
	return abs(changePercent) > threshold
}

// calculateSeverity determines the severity of a regression
func calculateSeverity(changePercent, threshold float64) string {
	absChange := abs(changePercent)
	
	if absChange > threshold*3 {
		return "CRITICAL"
	} else if absChange > threshold*2 {
		return "HIGH"
	} else if absChange > threshold {
		return "MEDIUM"
	}
	return "LOW"
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetRegressions returns detected regressions
func (pr *PerformanceRegression) GetRegressions() []Regression {
	return pr.regressions
}

// PrintRegressions prints detected regressions
func (pr *PerformanceRegression) PrintRegressions() {
	if len(pr.regressions) == 0 {
		fmt.Println("\n✅ No performance regressions detected!")
		return
	}
	
	fmt.Printf("\n⚠️  Detected %d performance regression(s):\n\n", len(pr.regressions))
	
	for i, reg := range pr.regressions {
		fmt.Printf("%d. %s [%s]\n", i+1, reg.Metric, reg.Severity)
		fmt.Printf("   Baseline: %.2f\n", reg.Baseline)
		fmt.Printf("   Current:  %.2f\n", reg.Current)
		fmt.Printf("   Change:   %.2f (%.1f%%)\n\n", reg.Change, reg.ChangePercent)
	}
}

// WorkloadSimulator simulates various workloads for profiling
type WorkloadSimulator struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewWorkloadSimulator creates a new workload simulator
func NewWorkloadSimulator() *WorkloadSimulator {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkloadSimulator{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins workload simulation
func (ws *WorkloadSimulator) Start() {
	// Start various workload patterns
	ws.wg.Add(4)
	
	go ws.cpuIntensiveWorkload()
	go ws.memoryIntensiveWorkload()
	go ws.concurrentWorkload()
	go ws.ioIntensiveWorkload()
}

// Stop stops workload simulation
func (ws *WorkloadSimulator) Stop() {
	ws.cancel()
	ws.wg.Wait()
}

// cpuIntensiveWorkload simulates CPU-intensive operations
func (ws *WorkloadSimulator) cpuIntensiveWorkload() {
	defer ws.wg.Done()
	
	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
			// Simulate CPU work
			sum := 0
			for i := 0; i < 1000000; i++ {
				sum += i * i
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
}

// memoryIntensiveWorkload simulates memory-intensive operations
func (ws *WorkloadSimulator) memoryIntensiveWorkload() {
	defer ws.wg.Done()
	
	data := make([][]byte, 0)
	
	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
			// Allocate memory
			buf := make([]byte, 1024*1024) // 1MB
			data = append(data, buf)
			
			// Prevent unlimited growth
			if len(data) > 100 {
				data = data[1:]
			}
			
			time.Sleep(time.Millisecond * 50)
		}
	}
}

// concurrentWorkload simulates concurrent operations
func (ws *WorkloadSimulator) concurrentWorkload() {
	defer ws.wg.Done()
	
	var mu sync.Mutex
	counter := 0
	
	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
			// Spawn concurrent goroutines
			for i := 0; i < 10; i++ {
				go func() {
					mu.Lock()
					counter++
					mu.Unlock()
					time.Sleep(time.Millisecond)
				}()
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// ioIntensiveWorkload simulates I/O-intensive operations
func (ws *WorkloadSimulator) ioIntensiveWorkload() {
	defer ws.wg.Done()
	
	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
			// Simulate file I/O
			tmpFile := fmt.Sprintf("/tmp/test_%d.tmp", rand.Intn(1000))
			data := make([]byte, 1024)
			
			// Write file
			if err := os.WriteFile(tmpFile, data, 0644); err == nil {
				// Read file
				os.ReadFile(tmpFile)
				// Clean up
				os.Remove(tmpFile)
			}
			
			time.Sleep(time.Millisecond * 20)
		}
	}
}

// ProfileCollector manages multiple profiling techniques
type ProfileCollector struct {
	customProfiler *CustomProfiler
	traceAnalyzer  *TraceAnalyzer
	regressionDetector *PerformanceRegression
	workloadSim    *WorkloadSimulator
	outputDir      string
}

// NewProfileCollector creates a new profile collector
func NewProfileCollector(outputDir string) *ProfileCollector {
	return &ProfileCollector{
		customProfiler:     NewCustomProfiler(time.Second),
		traceAnalyzer:      NewTraceAnalyzer(outputDir + "/trace.out"),
		regressionDetector: NewPerformanceRegression(),
		workloadSim:        NewWorkloadSimulator(),
		outputDir:          outputDir,
	}
}

// RunAdvancedProfiling demonstrates advanced profiling techniques
func (pc *ProfileCollector) RunAdvancedProfiling(duration time.Duration) error {
	// Create output directory
	if err := os.MkdirAll(pc.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	
	fmt.Println("🚀 Starting advanced profiling session...")
	
	// Start trace collection
	traceFile, err := os.Create(pc.outputDir + "/trace.out")
	if err != nil {
		return fmt.Errorf("failed to create trace file: %v", err)
	}
	defer traceFile.Close()
	
	if err := trace.Start(traceFile); err != nil {
		return fmt.Errorf("failed to start trace: %v", err)
	}
	defer trace.Stop()
	
	// Start CPU profiling
	cpuFile, err := os.Create(pc.outputDir + "/cpu.prof")
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %v", err)
	}
	defer cpuFile.Close()
	
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return fmt.Errorf("failed to start CPU profile: %v", err)
	}
	defer pprof.StopCPUProfile()
	
	// Start custom profiling
	pc.customProfiler.Start()
	defer pc.customProfiler.Stop()
	
	// Start workload simulation
	pc.workloadSim.Start()
	defer pc.workloadSim.Stop()
	
	fmt.Printf("⏱️  Running profiling for %v...\n", duration)
	
	// Run for specified duration
	time.Sleep(duration)
	
	fmt.Println("📊 Collecting and analyzing results...")
	
	// Collect memory profile
	memFile, err := os.Create(pc.outputDir + "/mem.prof")
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %v", err)
	}
	defer memFile.Close()
	
	runtime.GC() // Force GC before memory profile
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		return fmt.Errorf("failed to write memory profile: %v", err)
	}
	
	// Export custom profiling data
	if err := pc.customProfiler.ExportToFile(pc.outputDir + "/custom_metrics.csv"); err != nil {
		return fmt.Errorf("failed to export custom metrics: %v", err)
	}
	
	// Analyze trace (simplified)
	if err := pc.traceAnalyzer.AnalyzeTrace(); err != nil {
		return fmt.Errorf("failed to analyze trace: %v", err)
	}
	
	fmt.Println("✅ Profiling session completed!")
	return nil
}

// DemonstrateRegressionDetection shows regression detection capabilities
func (pc *ProfileCollector) DemonstrateRegressionDetection() {
	fmt.Println("\n🔍 Demonstrating Performance Regression Detection...")
	
	// Set baseline metrics (simulated)
	baseline := map[string]float64{
		"response_time_ms":    50.0,
		"throughput_rps":      1000.0,
		"memory_alloc_mb":     100.0,
		"gc_time_ms":          5.0,
		"cpu_usage_percent":   25.0,
	}
	
	// Set current metrics (simulated with some regressions)
	current := map[string]float64{
		"response_time_ms":    75.0,  // 50% increase (regression)
		"throughput_rps":      800.0, // 20% decrease (regression)
		"memory_alloc_mb":     120.0, // 20% increase (regression)
		"gc_time_ms":          4.8,   // Slight improvement
		"cpu_usage_percent":   24.0,  // Slight improvement
	}
	
	// Set thresholds
	thresholds := map[string]float64{
		"response_time_ms":    15.0, // 15% threshold
		"throughput_rps":      10.0, // 10% threshold
		"memory_alloc_mb":     15.0, // 15% threshold
		"gc_time_ms":          20.0, // 20% threshold
		"cpu_usage_percent":   10.0, // 10% threshold
	}
	
	pc.regressionDetector.SetBaseline(baseline)
	pc.regressionDetector.SetCurrent(current)
	pc.regressionDetector.SetThresholds(thresholds)
	
	// Detect regressions
	regressions := pc.regressionDetector.DetectRegressions()
	
	// Print results
	pc.regressionDetector.PrintRegressions()
	
	// Export regression report
	if err := pc.exportRegressionReport(regressions); err != nil {
		log.Printf("Failed to export regression report: %v", err)
	}
}

// exportRegressionReport exports regression detection results
func (pc *ProfileCollector) exportRegressionReport(regressions []Regression) error {
	file, err := os.Create(pc.outputDir + "/regression_report.csv")
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	fmt.Fprintf(file, "metric,baseline,current,change,change_percent,severity\n")
	
	// Write regressions
	for _, reg := range regressions {
		fmt.Fprintf(file, "%s,%.2f,%.2f,%.2f,%.1f,%s\n",
			reg.Metric, reg.Baseline, reg.Current, reg.Change, reg.ChangePercent, reg.Severity)
	}
	
	return nil
}

// PrintSummary prints a summary of all profiling results
func (pc *ProfileCollector) PrintSummary() {
	fmt.Println("\n📈 Advanced Profiling Summary")
	fmt.Println("=============================")
	
	// Custom profiling summary
	samples := pc.customProfiler.GetSamples()
	if len(samples) > 0 {
		fmt.Printf("\n📊 Custom Profiling:\n")
		fmt.Printf("  Samples collected: %d\n", len(samples))
		fmt.Printf("  Duration: %v\n", samples[len(samples)-1].Timestamp.Sub(samples[0].Timestamp))
		
		// Calculate averages
		totalGoroutines := 0
		totalMemory := uint64(0)
		for _, sample := range samples {
			totalGoroutines += sample.Goroutines
			totalMemory += sample.MemoryAlloc
		}
		avgGoroutines := float64(totalGoroutines) / float64(len(samples))
		avgMemory := float64(totalMemory) / float64(len(samples)) / 1024 / 1024 // MB
		
		fmt.Printf("  Average goroutines: %.1f\n", avgGoroutines)
		fmt.Printf("  Average memory: %.1f MB\n", avgMemory)
	}
	
	// Trace analysis summary
	pc.traceAnalyzer.PrintSummary()
	
	fmt.Printf("\n📁 Output files created in: %s\n", pc.outputDir)
	fmt.Println("  - cpu.prof (CPU profile)")
	fmt.Println("  - mem.prof (Memory profile)")
	fmt.Println("  - trace.out (Execution trace)")
	fmt.Println("  - custom_metrics.csv (Custom metrics)")
	fmt.Println("  - regression_report.csv (Regression analysis)")
	
	fmt.Println("\n🔧 Analysis commands:")
	fmt.Printf("  go tool pprof %s/cpu.prof\n", pc.outputDir)
	fmt.Printf("  go tool pprof %s/mem.prof\n", pc.outputDir)
	fmt.Printf("  go tool trace %s/trace.out\n", pc.outputDir)
}

func main() {
	// Set up profiling
	outputDir := "advanced_profiles"
	profileCollector := NewProfileCollector(outputDir)
	
	fmt.Println("🎯 Advanced Go Profiling Techniques Demo")
	fmt.Println("========================================")
	
	// Run advanced profiling session
	if err := profileCollector.RunAdvancedProfiling(30 * time.Second); err != nil {
		log.Fatalf("Profiling failed: %v", err)
	}
	
	// Demonstrate regression detection
	profileCollector.DemonstrateRegressionDetection()
	
	// Print summary
	profileCollector.PrintSummary()
	
	fmt.Println("\n🎉 Advanced profiling demonstration completed!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Analyze the generated profiles using go tool pprof")
	fmt.Println("2. Examine the trace using go tool trace")
	fmt.Println("3. Review custom metrics in the CSV file")
	fmt.Println("4. Check regression report for performance issues")
	fmt.Println("5. Integrate these techniques into your CI/CD pipeline")
}