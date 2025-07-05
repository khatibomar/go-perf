package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// LeakType represents different types of memory leaks
type LeakType int

const (
	LeakTypeGoroutine LeakType = iota
	LeakTypeMemory
	LeakTypeHeapObject
	LeakTypeGCPressure
	LeakTypeFileDescriptor
	LeakTypeNetworkConnection
	LeakTypeGeneral
)

func (lt LeakType) String() string {
	switch lt {
	case LeakTypeGoroutine:
		return "Goroutine"
	case LeakTypeMemory:
		return "Memory"
	case LeakTypeHeapObject:
		return "HeapObject"
	case LeakTypeGCPressure:
		return "GCPressure"
	case LeakTypeFileDescriptor:
		return "FileDescriptor"
	case LeakTypeNetworkConnection:
		return "NetworkConnection"
	case LeakTypeGeneral:
		return "General"
	default:
		return "Unknown"
	}
}

// LeakSeverity represents the severity of a detected leak
type LeakSeverity int

const (
	LeakSeverityLow LeakSeverity = iota
	LeakSeverityMedium
	LeakSeverityHigh
	LeakSeverityCritical
)

func (ls LeakSeverity) String() string {
	switch ls {
	case LeakSeverityLow:
		return "Low"
	case LeakSeverityMedium:
		return "Medium"
	case LeakSeverityHigh:
		return "High"
	case LeakSeverityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// MemorySnapshot represents a point-in-time memory state
type MemorySnapshot struct {
	Timestamp     time.Time
	Alloc         uint64
	TotalAlloc    uint64
	Sys           uint64
	NumGC         uint32
	Goroutines    int
	HeapObjects   uint64
	StackInUse    uint64
	MSpanInUse    uint64
	MCacheInUse   uint64
	GCCPUFraction float64
}

// LeakDetection represents a detected memory leak
type LeakDetection struct {
	ID          string
	Type        LeakType
	Severity    LeakSeverity
	Description string
	DetectedAt  time.Time
	Metrics     map[string]interface{}
	StackTrace  []string
	Suggestion  string
}

// LeakDetector is the main engine for detecting memory leaks
type LeakDetector struct {
	snapshots  []MemorySnapshot
	detections []LeakDetection
	mu         sync.RWMutex
	isRunning  bool
	stopChan   chan struct{}
	config     *LeakDetectorConfig
	baseline   *MemoryBaseline
	analyzers  []LeakAnalyzer
}

// LeakDetectorConfig contains configuration for the leak detector
type LeakDetectorConfig struct {
	SampleInterval        time.Duration
	MaxSnapshots          int
	MemoryThreshold       float64 // Percentage increase to trigger detection
	GoroutineThreshold    int     // Absolute increase in goroutines
	AnalysisWindow        time.Duration
	MinSamplesForAnalysis int
}

// MemoryBaseline represents baseline memory usage patterns
type MemoryBaseline struct {
	AvgAlloc       uint64
	AvgGoroutines  int
	AvgHeapObjects uint64
	StdDevAlloc    float64
	EstablishedAt  time.Time
	SampleCount    int
}

// LeakAnalyzer interface for different leak analysis strategies
type LeakAnalyzer interface {
	Analyze(snapshots []MemorySnapshot, baseline *MemoryBaseline) []LeakDetection
	Name() string
}

// NewLeakDetector creates a new leak detector instance
func NewLeakDetector(config *LeakDetectorConfig) *LeakDetector {
	if config == nil {
		config = DefaultLeakDetectorConfig()
	}

	ld := &LeakDetector{
		snapshots:  make([]MemorySnapshot, 0),
		detections: make([]LeakDetection, 0),
		stopChan:   make(chan struct{}),
		config:     config,
		analyzers:  make([]LeakAnalyzer, 0),
	}

	// Add default analyzers
	ld.AddAnalyzer(&GoroutineLeakAnalyzer{})
	ld.AddAnalyzer(&MemoryGrowthAnalyzer{})
	ld.AddAnalyzer(&HeapObjectAnalyzer{})
	ld.AddAnalyzer(&GCPressureAnalyzer{})

	return ld
}

// DefaultLeakDetectorConfig returns default configuration
func DefaultLeakDetectorConfig() *LeakDetectorConfig {
	return &LeakDetectorConfig{
		SampleInterval:        30 * time.Second,
		MaxSnapshots:          1000,
		MemoryThreshold:       20.0, // 20% increase
		GoroutineThreshold:    50,   // 50 goroutines increase
		AnalysisWindow:        10 * time.Minute,
		MinSamplesForAnalysis: 5,
	}
}

// AddAnalyzer adds a leak analyzer to the detector
func (ld *LeakDetector) AddAnalyzer(analyzer LeakAnalyzer) {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	ld.analyzers = append(ld.analyzers, analyzer)
}

// Start begins leak detection monitoring
func (ld *LeakDetector) Start() error {
	ld.mu.Lock()
	if ld.isRunning {
		ld.mu.Unlock()
		return fmt.Errorf("leak detector is already running")
	}
	ld.isRunning = true
	ld.mu.Unlock()

	go ld.monitoringLoop()
	fmt.Println("Leak detector started")
	return nil
}

// Stop stops leak detection monitoring
func (ld *LeakDetector) Stop() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	if !ld.isRunning {
		return
	}

	ld.isRunning = false
	close(ld.stopChan)
	fmt.Println("Leak detector stopped")
}

// GetSnapshots returns a copy of current snapshots
func (ld *LeakDetector) GetSnapshots() []MemorySnapshot {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	snapshots := make([]MemorySnapshot, len(ld.snapshots))
	copy(snapshots, ld.snapshots)
	return snapshots
}

// IsRunning returns whether the detector is currently running
func (ld *LeakDetector) IsRunning() bool {
	ld.mu.RLock()
	defer ld.mu.RUnlock()
	return ld.isRunning
}

// GetDetections returns all detected leaks
func (ld *LeakDetector) GetDetections() []LeakDetection {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	detections := make([]LeakDetection, len(ld.detections))
	copy(detections, ld.detections)
	return detections
}

// GetRecentDetections returns detections within the specified duration
func (ld *LeakDetector) GetRecentDetections(duration time.Duration) []LeakDetection {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var recent []LeakDetection

	for _, detection := range ld.detections {
		if detection.DetectedAt.After(cutoff) {
			recent = append(recent, detection)
		}
	}

	return recent
}

// GetBaseline returns the current memory baseline
func (ld *LeakDetector) GetBaseline() *MemoryBaseline {
	ld.mu.RLock()
	defer ld.mu.RUnlock()
	return ld.baseline
}

// monitoringLoop runs the main monitoring loop
func (ld *LeakDetector) monitoringLoop() {
	ticker := time.NewTicker(ld.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ld.collectSnapshot()
			ld.analyzeForLeaks()
		case <-ld.stopChan:
			return
		}
	}
}

// collectSnapshot takes a memory snapshot
func (ld *LeakDetector) collectSnapshot() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := MemorySnapshot{
		Timestamp:     time.Now(),
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		NumGC:         m.NumGC,
		Goroutines:    runtime.NumGoroutine(),
		HeapObjects:   m.HeapObjects,
		StackInUse:    m.StackInuse,
		MSpanInUse:    m.MSpanInuse,
		MCacheInUse:   m.MCacheInuse,
		GCCPUFraction: m.GCCPUFraction,
	}

	ld.mu.Lock()
	ld.snapshots = append(ld.snapshots, snapshot)

	// Limit snapshot history
	if len(ld.snapshots) > ld.config.MaxSnapshots {
		ld.snapshots = ld.snapshots[1:]
	}

	// Update baseline if we have enough samples
	if len(ld.snapshots) >= ld.config.MinSamplesForAnalysis {
		ld.updateBaseline()
	}
	ld.mu.Unlock()
}

// updateBaseline updates the memory usage baseline
func (ld *LeakDetector) updateBaseline() {
	if len(ld.snapshots) < ld.config.MinSamplesForAnalysis {
		return
	}

	// Use recent snapshots for baseline calculation
	recentCount := min(len(ld.snapshots), 20)
	recentSnapshots := ld.snapshots[len(ld.snapshots)-recentCount:]

	var totalAlloc, totalGoroutines, totalHeapObjects uint64
	for _, snapshot := range recentSnapshots {
		totalAlloc += snapshot.Alloc
		totalGoroutines += uint64(snapshot.Goroutines)
		totalHeapObjects += snapshot.HeapObjects
	}

	avgAlloc := totalAlloc / uint64(len(recentSnapshots))
	avgGoroutines := int(totalGoroutines / uint64(len(recentSnapshots)))
	avgHeapObjects := totalHeapObjects / uint64(len(recentSnapshots))

	// Calculate standard deviation for alloc
	var variance float64
	for _, snapshot := range recentSnapshots {
		diff := float64(snapshot.Alloc) - float64(avgAlloc)
		variance += diff * diff
	}
	stdDev := variance / float64(len(recentSnapshots))

	ld.baseline = &MemoryBaseline{
		AvgAlloc:       avgAlloc,
		AvgGoroutines:  avgGoroutines,
		AvgHeapObjects: avgHeapObjects,
		StdDevAlloc:    stdDev,
		EstablishedAt:  time.Now(),
		SampleCount:    len(recentSnapshots),
	}
}

// analyzeForLeaks runs leak analysis using all registered analyzers
func (ld *LeakDetector) analyzeForLeaks() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	if len(ld.snapshots) < ld.config.MinSamplesForAnalysis || ld.baseline == nil {
		return
	}

	// Get analysis window snapshots
	cutoff := time.Now().Add(-ld.config.AnalysisWindow)
	var windowSnapshots []MemorySnapshot
	for _, snapshot := range ld.snapshots {
		if snapshot.Timestamp.After(cutoff) {
			windowSnapshots = append(windowSnapshots, snapshot)
		}
	}

	if len(windowSnapshots) < ld.config.MinSamplesForAnalysis {
		return
	}

	// Run all analyzers
	for _, analyzer := range ld.analyzers {
		detections := analyzer.Analyze(windowSnapshots, ld.baseline)
		for _, detection := range detections {
			ld.detections = append(ld.detections, detection)
		}
	}
}

// GenerateReport generates a comprehensive leak detection report
func (ld *LeakDetector) GenerateReport() *LeakReport {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	report := &LeakReport{
		GeneratedAt:     time.Now(),
		TotalDetections: len(ld.detections),
		Detections:      make([]LeakDetection, len(ld.detections)),
		Summary:         make(map[LeakType]int),
		SeveritySummary: make(map[LeakSeverity]int),
	}

	copy(report.Detections, ld.detections)

	// Generate summaries
	for _, detection := range ld.detections {
		report.Summary[detection.Type]++
		report.SeveritySummary[detection.Severity]++
	}

	// Add current memory state
	if len(ld.snapshots) > 0 {
		report.CurrentSnapshot = &ld.snapshots[len(ld.snapshots)-1]
	}

	report.Baseline = ld.baseline

	return report
}

// LeakReport contains comprehensive leak detection results
type LeakReport struct {
	GeneratedAt     time.Time
	TotalDetections int
	Detections      []LeakDetection
	Summary         map[LeakType]int
	SeveritySummary map[LeakSeverity]int
	CurrentSnapshot *MemorySnapshot
	Baseline        *MemoryBaseline
	Recommendations []string
}

// AddRecommendation adds a recommendation to the report
func (lr *LeakReport) AddRecommendation(recommendation string) {
	lr.Recommendations = append(lr.Recommendations, recommendation)
}

// GetHighSeverityDetections returns detections with high or critical severity
func (lr *LeakReport) GetHighSeverityDetections() []LeakDetection {
	var highSeverity []LeakDetection
	for _, detection := range lr.Detections {
		if detection.Severity == LeakSeverityHigh || detection.Severity == LeakSeverityCritical {
			highSeverity = append(highSeverity, detection)
		}
	}
	return highSeverity
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateStackTrace captures current stack trace
func generateStackTrace() []string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return []string{string(buf[:n])}
}

// generateLeakID generates a unique ID for a leak detection
func generateLeakID(leakType LeakType, timestamp time.Time) string {
	return fmt.Sprintf("%s_%d", leakType.String(), timestamp.Unix())
}
