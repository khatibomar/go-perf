package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

// GCTuner provides comprehensive GC tuning and configuration management
type GCTuner struct {
	mu                sync.RWMutex
	currentConfig     *GCConfig
	baselineMetrics   *GCMetrics
	tuningHistory     []*TuningResult
	optimalConfig     *GCConfig
	monitoringActive  bool
	monitoringStop    chan struct{}
}

// GCConfig represents GC configuration parameters
type GCConfig struct {
	GOGC              int     // GC target percentage
	GOMAXPROCS        int     // Maximum number of OS threads
	GCPercent         int     // Heap growth percentage before GC
	MemoryLimit       int64   // Soft memory limit in bytes
	TargetHeapSize    int64   // Target heap size in bytes
	MaxHeapSize       int64   // Maximum allowed heap size
	GCCPUFraction     float64 // Maximum fraction of CPU time for GC
	EnableGCTrace     bool    // Enable GC tracing
	EnableMemProfile  bool    // Enable memory profiling
	CustomParameters  map[string]interface{} // Custom tuning parameters
}

// GCMetrics represents comprehensive GC performance metrics
type GCMetrics struct {
	Timestamp         time.Time
	NumGC             uint32
	PauseTotalNs      uint64
	PauseNs           []uint64
	PauseEnd          []uint64
	NumForcedGC       uint32
	GCCPUFraction     float64
	HeapAlloc         uint64
	HeapSys           uint64
	HeapIdle          uint64
	HeapInuse         uint64
	HeapReleased      uint64
	HeapObjects       uint64
	StackInuse        uint64
	StackSys          uint64
	MSpanInuse        uint64
	MSpanSys          uint64
	MCacheInuse       uint64
	MCacheSys         uint64
	BuckHashSys       uint64
	GCSys             uint64
	OtherSys          uint64
	NextGC            uint64
	LastGC            uint64
	TotalAlloc        uint64
	Lookups           uint64
	Mallocs           uint64
	Frees             uint64
	// ByGC and BySize fields removed - not available in runtime.MemStats
	
	// Derived metrics
	AllocationRate    float64 // Bytes per second
	GCFrequency       float64 // GCs per second
	AveragePause      float64 // Average pause time in ms
	MaxPause          float64 // Maximum pause time in ms
	P95Pause          float64 // 95th percentile pause time in ms
	P99Pause          float64 // 99th percentile pause time in ms
	HeapUtilization   float64 // Heap utilization percentage
	GCOverhead        float64 // GC overhead percentage
	Throughput        float64 // Operations per second
}

// TuningResult represents the result of a GC tuning experiment
type TuningResult struct {
	Config            *GCConfig
	Metrics           *GCMetrics
	Score             float64
	ImprovementPct    float64
	Duration          time.Duration
	WorkloadType      string
	Notes             string
	Timestamp         time.Time
}

// TuningStrategy defines different GC tuning strategies
type TuningStrategy int

const (
	StrategyLatency TuningStrategy = iota
	StrategyThroughput
	StrategyMemory
	StrategyBalanced
	StrategyCustom
)

// WorkloadProfile represents different application workload characteristics
type WorkloadProfile struct {
	Name                string
	AllocationPattern   string // "steady", "bursty", "mixed"
	ObjectLifetime      string // "short", "medium", "long", "mixed"
	ConcurrencyLevel    string // "low", "medium", "high"
	MemoryPressure      string // "low", "medium", "high"
	LatencySensitivity  string // "low", "medium", "high"
	ThroughputPriority  string // "low", "medium", "high"
}

// NewGCTuner creates a new GC tuner instance
func NewGCTuner() *GCTuner {
	tuner := &GCTuner{
		currentConfig:    getCurrentGCConfig(),
		tuningHistory:    make([]*TuningResult, 0),
		monitoringStop:   make(chan struct{}),
		monitoringActive: false,
	}
	
	// Collect baseline metrics
	tuner.baselineMetrics = tuner.CollectMetrics()
	return tuner
}

// getCurrentGCConfig retrieves the current GC configuration
func getCurrentGCConfig() *GCConfig {
	gogc := 100 // Default GOGC value
	if env := os.Getenv("GOGC"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			gogc = val
		}
	}
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &GCConfig{
		GOGC:             gogc,
		GOMAXPROCS:       runtime.GOMAXPROCS(0),
		GCPercent:        debug.SetGCPercent(-1), // Get current value
		MemoryLimit:      debug.SetMemoryLimit(-1), // Get current value
		TargetHeapSize:   int64(memStats.NextGC),
		MaxHeapSize:      int64(memStats.HeapSys),
		GCCPUFraction:    0.25, // Default GC CPU fraction
		EnableGCTrace:    os.Getenv("GODEBUG") != "",
		EnableMemProfile: false,
		CustomParameters: make(map[string]interface{}),
	}
}

// CollectMetrics gathers comprehensive GC metrics
func (tuner *GCTuner) CollectMetrics() *GCMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	metrics := &GCMetrics{
		Timestamp:       time.Now(),
		NumGC:           memStats.NumGC,
		PauseTotalNs:    memStats.PauseTotalNs,
		NumForcedGC:     memStats.NumForcedGC,
		GCCPUFraction:   memStats.GCCPUFraction,
		HeapAlloc:       memStats.HeapAlloc,
		HeapSys:         memStats.HeapSys,
		HeapIdle:        memStats.HeapIdle,
		HeapInuse:       memStats.HeapInuse,
		HeapReleased:    memStats.HeapReleased,
		HeapObjects:     memStats.HeapObjects,
		StackInuse:      memStats.StackInuse,
		StackSys:        memStats.StackSys,
		MSpanInuse:      memStats.MSpanInuse,
		MSpanSys:        memStats.MSpanSys,
		MCacheInuse:     memStats.MCacheInuse,
		MCacheSys:       memStats.MCacheSys,
		BuckHashSys:     memStats.BuckHashSys,
		GCSys:           memStats.GCSys,
		OtherSys:        memStats.OtherSys,
		NextGC:          memStats.NextGC,
		LastGC:          memStats.LastGC,
		TotalAlloc:      memStats.TotalAlloc,
		Lookups:         memStats.Lookups,
		Mallocs:         memStats.Mallocs,
		Frees:           memStats.Frees,
		// ByGC and BySize fields removed - not available in runtime.MemStats
	}
	
	// Copy pause times
	metrics.PauseNs = make([]uint64, len(memStats.PauseNs))
	copy(metrics.PauseNs, memStats.PauseNs[:])
	
	metrics.PauseEnd = make([]uint64, len(memStats.PauseEnd))
	copy(metrics.PauseEnd, memStats.PauseEnd[:])
	
	// Calculate derived metrics
	tuner.calculateDerivedMetrics(metrics)
	
	return metrics
}

// calculateDerivedMetrics computes additional performance metrics
func (tuner *GCTuner) calculateDerivedMetrics(metrics *GCMetrics) {
	// Calculate allocation rate
	if tuner.baselineMetrics != nil {
		timeDiff := metrics.Timestamp.Sub(tuner.baselineMetrics.Timestamp).Seconds()
		if timeDiff > 0 {
			allocDiff := metrics.TotalAlloc - tuner.baselineMetrics.TotalAlloc
			metrics.AllocationRate = float64(allocDiff) / timeDiff
			
			gcDiff := metrics.NumGC - tuner.baselineMetrics.NumGC
			metrics.GCFrequency = float64(gcDiff) / timeDiff
		}
	}
	
	// Calculate pause time statistics
	if len(metrics.PauseNs) > 0 {
		var totalPause, maxPause uint64
		validPauses := make([]uint64, 0)
		
		for _, pause := range metrics.PauseNs {
			if pause > 0 {
				validPauses = append(validPauses, pause)
				totalPause += pause
				if pause > maxPause {
					maxPause = pause
				}
			}
		}
		
		if len(validPauses) > 0 {
			metrics.AveragePause = float64(totalPause) / float64(len(validPauses)) / 1e6 // Convert to ms
			metrics.MaxPause = float64(maxPause) / 1e6
			
			// Calculate percentiles
			metrics.P95Pause = calculatePercentile(validPauses, 0.95) / 1e6
			metrics.P99Pause = calculatePercentile(validPauses, 0.99) / 1e6
		}
	}
	
	// Calculate heap utilization
	if metrics.HeapSys > 0 {
		metrics.HeapUtilization = float64(metrics.HeapInuse) / float64(metrics.HeapSys) * 100
	}
	
	// Calculate GC overhead
	metrics.GCOverhead = metrics.GCCPUFraction * 100
}

// calculatePercentile calculates the specified percentile of pause times
func calculatePercentile(pauses []uint64, percentile float64) float64 {
	if len(pauses) == 0 {
		return 0
	}
	
	// Simple percentile calculation (could be improved with proper sorting)
	index := int(float64(len(pauses)) * percentile)
	if index >= len(pauses) {
		index = len(pauses) - 1
	}
	
	// Find the value at the percentile index (simplified)
	max := uint64(0)
	for _, pause := range pauses {
		if pause > max {
			max = pause
		}
	}
	
	return float64(max)
}

// ApplyConfig applies a GC configuration
func (tuner *GCTuner) ApplyConfig(config *GCConfig) error {
	tuner.mu.Lock()
	defer tuner.mu.Unlock()
	
	// Apply GOGC setting
	if config.GOGC != tuner.currentConfig.GOGC {
		debug.SetGCPercent(config.GOGC)
	}
	
	// Apply GOMAXPROCS setting
	if config.GOMAXPROCS != tuner.currentConfig.GOMAXPROCS {
		runtime.GOMAXPROCS(config.GOMAXPROCS)
	}
	
	// Apply memory limit
	if config.MemoryLimit != tuner.currentConfig.MemoryLimit {
		debug.SetMemoryLimit(config.MemoryLimit)
	}
	
	// Apply GC percent
	if config.GCPercent != tuner.currentConfig.GCPercent {
		debug.SetGCPercent(config.GCPercent)
	}
	
	// Store the new configuration
	tuner.currentConfig = config
	
	return nil
}

// TuneForStrategy optimizes GC configuration for a specific strategy
func (tuner *GCTuner) TuneForStrategy(strategy TuningStrategy, workload *WorkloadProfile) (*GCConfig, error) {
	switch strategy {
	case StrategyLatency:
		return tuner.tuneForLatency(workload)
	case StrategyThroughput:
		return tuner.tuneForThroughput(workload)
	case StrategyMemory:
		return tuner.tuneForMemory(workload)
	case StrategyBalanced:
		return tuner.tuneForBalance(workload)
	default:
		return tuner.currentConfig, fmt.Errorf("unsupported tuning strategy: %v", strategy)
	}
}

// tuneForLatency optimizes for low latency (minimal pause times)
func (tuner *GCTuner) tuneForLatency(workload *WorkloadProfile) (*GCConfig, error) {
	config := *tuner.currentConfig // Copy current config
	
	// Lower GOGC for more frequent but shorter GC cycles
	config.GOGC = 50
	config.GCPercent = 50
	
	// Set conservative memory limit to prevent large heaps
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	config.MemoryLimit = int64(memStats.HeapSys * 2)
	
	// Reduce GC CPU fraction for lower pause times
	config.GCCPUFraction = 0.1
	
	// Enable GC tracing for monitoring
	config.EnableGCTrace = true
	
	return &config, nil
}

// tuneForThroughput optimizes for maximum throughput
func (tuner *GCTuner) tuneForThroughput(workload *WorkloadProfile) (*GCConfig, error) {
	config := *tuner.currentConfig // Copy current config
	
	// Higher GOGC for less frequent GC cycles
	config.GOGC = 200
	config.GCPercent = 200
	
	// Allow larger heap sizes
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	config.MemoryLimit = int64(memStats.HeapSys * 4)
	
	// Increase GC CPU fraction for faster collection
	config.GCCPUFraction = 0.5
	
	// Maximize GOMAXPROCS for parallel collection
	config.GOMAXPROCS = runtime.NumCPU()
	
	return &config, nil
}

// tuneForMemory optimizes for minimal memory usage
func (tuner *GCTuner) tuneForMemory(workload *WorkloadProfile) (*GCConfig, error) {
	config := *tuner.currentConfig // Copy current config
	
	// Very low GOGC for aggressive collection
	config.GOGC = 25
	config.GCPercent = 25
	
	// Tight memory limits
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	config.MemoryLimit = int64(memStats.HeapInuse * 3 / 2)
	
	// Moderate GC CPU fraction
	config.GCCPUFraction = 0.25
	
	return &config, nil
}

// tuneForBalance optimizes for balanced performance
func (tuner *GCTuner) tuneForBalance(workload *WorkloadProfile) (*GCConfig, error) {
	config := *tuner.currentConfig // Copy current config
	
	// Moderate GOGC setting
	config.GOGC = 100
	config.GCPercent = 100
	
	// Balanced memory limits
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	config.MemoryLimit = int64(memStats.HeapSys * 3)
	
	// Standard GC CPU fraction
	config.GCCPUFraction = 0.25
	
	// Optimize based on workload characteristics
	if workload != nil {
		switch workload.LatencySensitivity {
		case "high":
			config.GOGC = 75
		case "low":
			config.GOGC = 150
		}
		
		switch workload.MemoryPressure {
		case "high":
			config.GCPercent = 50
		case "low":
			config.GCPercent = 150
		}
	}
	
	return &config, nil
}

// RunTuningExperiment runs a tuning experiment with the given configuration
func (tuner *GCTuner) RunTuningExperiment(config *GCConfig, workloadFunc func(), duration time.Duration) (*TuningResult, error) {
	// Collect baseline metrics
	baselineMetrics := tuner.CollectMetrics()
	
	// Apply the configuration
	oldConfig := tuner.currentConfig
	err := tuner.ApplyConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to apply config: %w", err)
	}
	
	// Run the workload
	startTime := time.Now()
	workloadFunc()
	runtime.GC() // Force GC to get accurate measurements
	time.Sleep(duration)
	
	// Collect final metrics
	finalMetrics := tuner.CollectMetrics()
	
	// Restore old configuration
	tuner.ApplyConfig(oldConfig)
	
	// Calculate performance score
	score := tuner.calculatePerformanceScore(finalMetrics, baselineMetrics)
	improvementPct := ((score - tuner.calculatePerformanceScore(baselineMetrics, tuner.baselineMetrics)) / tuner.calculatePerformanceScore(baselineMetrics, tuner.baselineMetrics)) * 100
	
	result := &TuningResult{
		Config:         config,
		Metrics:        finalMetrics,
		Score:          score,
		ImprovementPct: improvementPct,
		Duration:       time.Since(startTime),
		Timestamp:      time.Now(),
	}
	
	// Store result in history
	tuner.mu.Lock()
	tuner.tuningHistory = append(tuner.tuningHistory, result)
	tuner.mu.Unlock()
	
	return result, nil
}

// calculatePerformanceScore calculates a composite performance score
func (tuner *GCTuner) calculatePerformanceScore(current, baseline *GCMetrics) float64 {
	if baseline == nil {
		return 50.0 // Default score
	}
	
	score := 100.0
	
	// Penalize for high pause times (weight: 30%)
	if current.AveragePause > 0 && baseline.AveragePause > 0 {
		pauseRatio := current.AveragePause / baseline.AveragePause
		score -= (pauseRatio - 1.0) * 30.0
	}
	
	// Penalize for high GC frequency (weight: 25%)
	if current.GCFrequency > 0 && baseline.GCFrequency > 0 {
		freqRatio := current.GCFrequency / baseline.GCFrequency
		score -= (freqRatio - 1.0) * 25.0
	}
	
	// Penalize for high memory usage (weight: 20%)
	if current.HeapInuse > 0 && baseline.HeapInuse > 0 {
		memRatio := float64(current.HeapInuse) / float64(baseline.HeapInuse)
		score -= (memRatio - 1.0) * 20.0
	}
	
	// Penalize for high GC overhead (weight: 15%)
	if current.GCOverhead > baseline.GCOverhead {
		overheadDiff := current.GCOverhead - baseline.GCOverhead
		score -= overheadDiff * 15.0
	}
	
	// Reward for high allocation rate (weight: 10%)
	if current.AllocationRate > baseline.AllocationRate {
		allocImprovement := (current.AllocationRate - baseline.AllocationRate) / baseline.AllocationRate
		score += allocImprovement * 10.0
	}
	
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// StartMonitoring begins continuous GC monitoring
func (tuner *GCTuner) StartMonitoring(interval time.Duration) {
	tuner.mu.Lock()
	if tuner.monitoringActive {
		tuner.mu.Unlock()
		return
	}
	tuner.monitoringActive = true
	tuner.baselineMetrics = tuner.CollectMetrics()
	tuner.mu.Unlock()
	
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				metrics := tuner.CollectMetrics()
				tuner.analyzeMetrics(metrics)
			case <-tuner.monitoringStop:
				return
			}
		}
	}()
}

// StopMonitoring stops continuous GC monitoring
func (tuner *GCTuner) StopMonitoring() {
	tuner.mu.Lock()
	defer tuner.mu.Unlock()
	
	if tuner.monitoringActive {
		close(tuner.monitoringStop)
		tuner.monitoringActive = false
		tuner.monitoringStop = make(chan struct{})
	}
}

// analyzeMetrics analyzes current metrics and suggests optimizations
func (tuner *GCTuner) analyzeMetrics(metrics *GCMetrics) {
	if tuner.baselineMetrics == nil {
		return
	}
	
	// Check for performance degradation
	if metrics.AveragePause > tuner.baselineMetrics.AveragePause*1.5 {
		fmt.Printf("WARNING: GC pause time increased by %.1f%%\n", 
			(metrics.AveragePause/tuner.baselineMetrics.AveragePause-1)*100)
	}
	
	if metrics.GCFrequency > tuner.baselineMetrics.GCFrequency*1.5 {
		fmt.Printf("WARNING: GC frequency increased by %.1f%%\n", 
			(metrics.GCFrequency/tuner.baselineMetrics.GCFrequency-1)*100)
	}
	
	if metrics.HeapUtilization < 50 {
		fmt.Printf("INFO: Low heap utilization (%.1f%%), consider increasing GOGC\n", 
			metrics.HeapUtilization)
	}
	
	if metrics.GCOverhead > 10 {
		fmt.Printf("WARNING: High GC overhead (%.1f%%), consider tuning\n", 
			metrics.GCOverhead)
	}
}

// GenerateReport generates a comprehensive tuning report
func (tuner *GCTuner) GenerateReport() string {
	tuner.mu.RLock()
	defer tuner.mu.RUnlock()
	
	report := "\n=== GC Tuning Report ===\n\n"
	
	// Current configuration
	report += "Current Configuration:\n"
	report += fmt.Sprintf("  GOGC: %d\n", tuner.currentConfig.GOGC)
	report += fmt.Sprintf("  GOMAXPROCS: %d\n", tuner.currentConfig.GOMAXPROCS)
	report += fmt.Sprintf("  GC Percent: %d\n", tuner.currentConfig.GCPercent)
	report += fmt.Sprintf("  Memory Limit: %d MB\n", tuner.currentConfig.MemoryLimit/(1024*1024))
	report += fmt.Sprintf("  GC CPU Fraction: %.2f\n", tuner.currentConfig.GCCPUFraction)
	report += "\n"
	
	// Current metrics
	if tuner.baselineMetrics != nil {
		report += "Current Metrics:\n"
		report += fmt.Sprintf("  Average Pause: %.2f ms\n", tuner.baselineMetrics.AveragePause)
		report += fmt.Sprintf("  Max Pause: %.2f ms\n", tuner.baselineMetrics.MaxPause)
		report += fmt.Sprintf("  GC Frequency: %.2f Hz\n", tuner.baselineMetrics.GCFrequency)
		report += fmt.Sprintf("  Heap Utilization: %.1f%%\n", tuner.baselineMetrics.HeapUtilization)
		report += fmt.Sprintf("  GC Overhead: %.1f%%\n", tuner.baselineMetrics.GCOverhead)
		report += fmt.Sprintf("  Allocation Rate: %.2f MB/s\n", tuner.baselineMetrics.AllocationRate/(1024*1024))
		report += "\n"
	}
	
	// Tuning history
	if len(tuner.tuningHistory) > 0 {
		report += "Tuning History:\n"
		for i, result := range tuner.tuningHistory {
			report += fmt.Sprintf("  Experiment %d:\n", i+1)
			report += fmt.Sprintf("    GOGC: %d\n", result.Config.GOGC)
			report += fmt.Sprintf("    Score: %.1f\n", result.Score)
			report += fmt.Sprintf("    Improvement: %.1f%%\n", result.ImprovementPct)
			report += fmt.Sprintf("    Duration: %v\n", result.Duration)
			report += "\n"
		}
	}
	
	// Recommendations
	report += "Recommendations:\n"
	if tuner.baselineMetrics != nil {
		if tuner.baselineMetrics.AveragePause > 10 {
			report += "  - Consider lowering GOGC to reduce pause times\n"
		}
		if tuner.baselineMetrics.GCFrequency > 10 {
			report += "  - Consider increasing GOGC to reduce GC frequency\n"
		}
		if tuner.baselineMetrics.HeapUtilization < 50 {
			report += "  - Consider increasing GOGC to improve memory utilization\n"
		}
		if tuner.baselineMetrics.GCOverhead > 10 {
			report += "  - Consider optimizing allocation patterns\n"
		}
	}
	
	return report
}

// DemonstrateGCTuning demonstrates GC tuning capabilities
func DemonstrateGCTuning() {
	fmt.Println("=== GC Tuning Demonstration ===")
	
	tuner := NewGCTuner()
	
	// Start monitoring
	tuner.StartMonitoring(time.Second)
	defer tuner.StopMonitoring()
	
	// Define a sample workload
	workload := func() {
		for i := 0; i < 1000; i++ {
			// Allocate various sized objects
			_ = make([]byte, 1024+i%1024)
			_ = make(map[int]string)
			_ = fmt.Sprintf("test_%d", i)
		}
	}
	
	// Test different tuning strategies
	strategies := []TuningStrategy{
		StrategyLatency,
		StrategyThroughput,
		StrategyMemory,
		StrategyBalanced,
	}
	
	workloadProfile := &WorkloadProfile{
		Name:               "Mixed Allocation",
		AllocationPattern:  "mixed",
		ObjectLifetime:     "short",
		ConcurrencyLevel:   "medium",
		MemoryPressure:     "medium",
		LatencySensitivity: "medium",
		ThroughputPriority: "high",
	}
	
	for _, strategy := range strategies {
		fmt.Printf("\nTesting strategy: %v\n", strategy)
		
		config, err := tuner.TuneForStrategy(strategy, workloadProfile)
		if err != nil {
			fmt.Printf("Error tuning for strategy: %v\n", err)
			continue
		}
		
		result, err := tuner.RunTuningExperiment(config, workload, time.Millisecond*100)
		if err != nil {
			fmt.Printf("Error running experiment: %v\n", err)
			continue
		}
		
		fmt.Printf("  GOGC: %d, Score: %.1f, Improvement: %.1f%%\n", 
			result.Config.GOGC, result.Score, result.ImprovementPct)
	}
	
	// Generate and display report
	fmt.Println(tuner.GenerateReport())
}

func main() {
	DemonstrateGCTuning()
}