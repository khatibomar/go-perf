package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GCStats represents garbage collection statistics
type GCStats struct {
	NumGC        uint32        // Number of GC cycles
	TotalPause   time.Duration // Total pause time
	LastGC       time.Time     // Time of last GC
	PauseHist    []time.Duration // Pause time history
	HeapSize     uint64        // Current heap size
	HeapObjects  uint64        // Number of objects in heap
	StackSize    uint64        // Stack size
	GCCPUFraction float64      // Fraction of CPU time spent in GC
}

// GCEvent represents a single garbage collection event
type GCEvent struct {
	Timestamp    time.Time     // When the GC occurred
	Sequence     int           // GC sequence number
	PauseTime    time.Duration // Stop-the-world pause time
	HeapBefore   uint64        // Heap size before GC
	HeapAfter    uint64        // Heap size after GC
	HeapGoal     uint64        // Target heap size
	CPUTime      time.Duration // CPU time spent in GC
	AssistTime   time.Duration // Time spent in GC assist
	BackgroundTime time.Duration // Time spent in background GC
	SweepTime    time.Duration // Time spent sweeping
	AllocRate    float64       // Allocation rate (MB/s)
	TriggerRatio float64       // Trigger ratio for this GC
}

// GCAnalyzer provides comprehensive GC analysis capabilities
type GCAnalyzer struct {
	events       []GCEvent
	startTime    time.Time
	lastStats    runtime.MemStats
	mu           sync.RWMutex
	monitoring   bool
	stopChan     chan struct{}
	metricsHist  []runtime.MemStats
	maxHistory   int
}

// NewGCAnalyzer creates a new GC analyzer
func NewGCAnalyzer() *GCAnalyzer {
	return &GCAnalyzer{
		events:      make([]GCEvent, 0),
		startTime:   time.Now(),
		stopChan:    make(chan struct{}),
		maxHistory:  1000, // Keep last 1000 measurements
		metricsHist: make([]runtime.MemStats, 0, 1000),
	}
}

// StartMonitoring begins continuous GC monitoring with default interval
func (gca *GCAnalyzer) StartMonitoring() error {
	return gca.StartMonitoringWithInterval(100 * time.Millisecond)
}

// StartMonitoringWithInterval begins continuous GC monitoring with specified interval
func (gca *GCAnalyzer) StartMonitoringWithInterval(interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("monitoring interval must be positive, got %v", interval)
	}
	
	gca.mu.Lock()
	defer gca.mu.Unlock()
	
	if gca.monitoring {
		return fmt.Errorf("monitoring is already active")
	}
	
	gca.monitoring = true
	gca.startTime = time.Now()
	
	go gca.monitorLoop(interval)
	return nil
}

// StopMonitoring stops GC monitoring
func (gca *GCAnalyzer) StopMonitoring() {
	gca.mu.Lock()
	defer gca.mu.Unlock()
	
	if !gca.monitoring {
		return
	}
	
	gca.monitoring = false
	close(gca.stopChan)
	gca.stopChan = make(chan struct{})
}

// monitorLoop continuously monitors GC statistics
func (gca *GCAnalyzer) monitorLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	var lastNumGC uint32
	
	for {
		select {
		case <-gca.stopChan:
			return
		case <-ticker.C:
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			
			// Store metrics history
			gca.mu.Lock()
			gca.metricsHist = append(gca.metricsHist, stats)
			if len(gca.metricsHist) > gca.maxHistory {
				gca.metricsHist = gca.metricsHist[1:]
			}
			
			// Check for new GC events
			if stats.NumGC > lastNumGC {
				for i := lastNumGC; i < stats.NumGC; i++ {
					event := gca.extractGCEvent(&stats, int(i))
					gca.events = append(gca.events, event)
				}
				lastNumGC = stats.NumGC
			}
			
			gca.lastStats = stats
			gca.mu.Unlock()
		}
	}
}

// extractGCEvent extracts GC event information from MemStats
func (gca *GCAnalyzer) extractGCEvent(stats *runtime.MemStats, sequence int) GCEvent {
	pauseIndex := (sequence + 255) % 256
	pauseTime := time.Duration(stats.PauseNs[pauseIndex])
	
	// Calculate allocation rate
	allocRate := 0.0
	if len(gca.metricsHist) > 0 {
		last := gca.metricsHist[len(gca.metricsHist)-1]
		timeDiff := time.Since(time.Unix(0, int64(last.LastGC)))
		if timeDiff > 0 {
			allocDiff := float64(stats.TotalAlloc - last.TotalAlloc)
			allocRate = allocDiff / timeDiff.Seconds() / (1024 * 1024) // MB/s
		}
	}
	
	return GCEvent{
		Timestamp:    time.Unix(0, int64(stats.LastGC)),
		Sequence:     sequence,
		PauseTime:    pauseTime,
		HeapBefore:   stats.HeapSys,
		HeapAfter:    stats.HeapInuse,
		HeapGoal:     stats.NextGC,
		CPUTime:      time.Duration(float64(pauseTime) * stats.GCCPUFraction),
		AllocRate:    allocRate,
		TriggerRatio: float64(stats.HeapInuse) / float64(stats.NextGC),
	}
}

// GetCurrentStats returns current GC statistics
func (gca *GCAnalyzer) GetCurrentStats() GCStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	
	gca.mu.RLock()
	defer gca.mu.RUnlock()
	
	// Calculate total pause time
	totalPause := time.Duration(0)
	for i := 0; i < len(stats.PauseNs) && i < int(stats.NumGC); i++ {
		totalPause += time.Duration(stats.PauseNs[i])
	}
	
	// Get pause history
	pauseHist := make([]time.Duration, 0, len(stats.PauseNs))
	for i := 0; i < len(stats.PauseNs) && i < int(stats.NumGC); i++ {
		if stats.PauseNs[i] > 0 {
			pauseHist = append(pauseHist, time.Duration(stats.PauseNs[i]))
		}
	}
	
	return GCStats{
		NumGC:         stats.NumGC,
		TotalPause:    totalPause,
		LastGC:        time.Unix(0, int64(stats.LastGC)),
		PauseHist:     pauseHist,
		HeapSize:      stats.HeapInuse,
		HeapObjects:   stats.HeapObjects,
		StackSize:     stats.StackInuse,
		GCCPUFraction: stats.GCCPUFraction,
	}
}

// GetStatistics returns enhanced GC statistics for testing
func (gca *GCAnalyzer) GetStatistics() TestGCStats {
	stats := gca.GetCurrentStats()
	gca.mu.RLock()
	defer gca.mu.RUnlock()
	
	return TestGCStats{
		TotalCycles:   int(stats.NumGC),
		TotalPause:    stats.TotalPause,
		LastGC:        stats.LastGC,
		HeapSize:      stats.HeapSize,
		HeapObjects:   stats.HeapObjects,
		GCCPUFraction: stats.GCCPUFraction,
		EventCount:    len(gca.events),
	}
}

// TestGCStats represents GC statistics for testing
type TestGCStats struct {
	TotalCycles   int
	TotalPause    time.Duration
	LastGC        time.Time
	HeapSize      uint64
	HeapObjects   uint64
	GCCPUFraction float64
	EventCount    int
}

// GenerateReport generates a comprehensive GC analysis report
func (gca *GCAnalyzer) GenerateReport() string {
	var report strings.Builder
	
	report.WriteString("=== GC Statistics Report ===\n\n")
	
	// Current statistics
	stats := gca.GetCurrentStats()
	report.WriteString("Current GC Statistics:\n")
	report.WriteString(fmt.Sprintf("  Number of GCs: %d\n", stats.NumGC))
	report.WriteString(fmt.Sprintf("  Total pause time: %v\n", stats.TotalPause))
	report.WriteString(fmt.Sprintf("  Last GC: %v\n", stats.LastGC.Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("  Heap size: %.2f MB\n", float64(stats.HeapSize)/(1024*1024)))
	report.WriteString(fmt.Sprintf("  Heap objects: %d\n", stats.HeapObjects))
	report.WriteString(fmt.Sprintf("  GC CPU fraction: %.4f\n", stats.GCCPUFraction))
	report.WriteString("\n")
	
	// Pause analysis
	gca.mu.RLock()
	eventCount := len(gca.events)
	gca.mu.RUnlock()
	
	if eventCount > 0 {
		pauseAnalysis := gca.AnalyzePausePatterns()
		report.WriteString("Pause Time Analysis:\n")
		report.WriteString(fmt.Sprintf("  Count: %d\n", pauseAnalysis.Count))
		report.WriteString(fmt.Sprintf("  Average: %v\n", pauseAnalysis.Average))
		report.WriteString(fmt.Sprintf("  Min: %v\n", pauseAnalysis.Min))
		report.WriteString(fmt.Sprintf("  Max: %v\n", pauseAnalysis.Max))
		report.WriteString("\n")
	}
	
	report.WriteString("=== End of Report ===\n")
	return report.String()
}

// GetEvents returns all recorded GC events
func (gca *GCAnalyzer) GetEvents() []GCEvent {
	gca.mu.RLock()
	defer gca.mu.RUnlock()
	
	events := make([]GCEvent, len(gca.events))
	copy(events, gca.events)
	return events
}

// AnalyzePausePatterns analyzes GC pause patterns
func (gca *GCAnalyzer) AnalyzePausePatterns() PauseAnalysis {
	gca.mu.RLock()
	defer gca.mu.RUnlock()
	
	if len(gca.events) == 0 {
		return PauseAnalysis{}
	}
	
	pauses := make([]time.Duration, len(gca.events))
	for i, event := range gca.events {
		pauses[i] = event.PauseTime
	}
	
	sort.Slice(pauses, func(i, j int) bool {
		return pauses[i] < pauses[j]
	})
	
	total := time.Duration(0)
	for _, pause := range pauses {
		total += pause
	}
	
	analysis := PauseAnalysis{
		Count:   len(pauses),
		Total:   total,
		Average: total / time.Duration(len(pauses)),
		Min:     pauses[0],
		Max:     pauses[len(pauses)-1],
	}
	
	// Calculate percentiles
	if len(pauses) > 0 {
		analysis.P50 = pauses[len(pauses)*50/100]
		analysis.P90 = pauses[len(pauses)*90/100]
		analysis.P95 = pauses[len(pauses)*95/100]
		analysis.P99 = pauses[len(pauses)*99/100]
	}
	
	return analysis
}

// PauseAnalysis contains pause time analysis results
type PauseAnalysis struct {
	Count   int
	Total   time.Duration
	Average time.Duration
	Min     time.Duration
	Max     time.Duration
	P50     time.Duration
	P90     time.Duration
	P95     time.Duration
	P99     time.Duration
}

// AnalyzeAllocationRate analyzes allocation rate patterns
func (gca *GCAnalyzer) AnalyzeAllocationRate() AllocationAnalysis {
	gca.mu.RLock()
	defer gca.mu.RUnlock()
	
	if len(gca.metricsHist) < 2 {
		return AllocationAnalysis{}
	}
	
	rates := make([]float64, 0, len(gca.metricsHist)-1)
	totalAlloc := float64(0)
	
	for i := 1; i < len(gca.metricsHist); i++ {
		curr := gca.metricsHist[i]
		prev := gca.metricsHist[i-1]
		
		timeDiff := time.Unix(0, int64(curr.LastGC)).Sub(time.Unix(0, int64(prev.LastGC)))
		if timeDiff > 0 {
			allocDiff := float64(curr.TotalAlloc - prev.TotalAlloc)
			rate := allocDiff / timeDiff.Seconds() / (1024 * 1024) // MB/s
			rates = append(rates, rate)
			totalAlloc += allocDiff
		}
	}
	
	if len(rates) == 0 {
		return AllocationAnalysis{}
	}
	
	sort.Float64s(rates)
	
	sum := 0.0
	for _, rate := range rates {
		sum += rate
	}
	
	return AllocationAnalysis{
		SampleCount:    len(rates),
		TotalAllocated: totalAlloc / (1024 * 1024), // MB
		AverageRate:    sum / float64(len(rates)),
		MinRate:        rates[0],
		MaxRate:        rates[len(rates)-1],
		P50Rate:        rates[len(rates)*50/100],
		P90Rate:        rates[len(rates)*90/100],
		P95Rate:        rates[len(rates)*95/100],
	}
}

// AllocationAnalysis contains allocation rate analysis results
type AllocationAnalysis struct {
	SampleCount    int
	TotalAllocated float64 // MB
	AverageRate    float64 // MB/s
	MinRate        float64 // MB/s
	MaxRate        float64 // MB/s
	P50Rate        float64 // MB/s
	P90Rate        float64 // MB/s
	P95Rate        float64 // MB/s
}

// AnalyzeHeapGrowth analyzes heap growth patterns
func (gca *GCAnalyzer) AnalyzeHeapGrowth() HeapAnalysis {
	gca.mu.RLock()
	defer gca.mu.RUnlock()
	
	if len(gca.metricsHist) < 2 {
		return HeapAnalysis{}
	}
	
	first := gca.metricsHist[0]
	last := gca.metricsHist[len(gca.metricsHist)-1]
	
	maxHeap := uint64(0)
	minHeap := ^uint64(0)
	totalGrowth := int64(0)
	growthEvents := 0
	
	for i := 1; i < len(gca.metricsHist); i++ {
		curr := gca.metricsHist[i]
		prev := gca.metricsHist[i-1]
		
		if curr.HeapInuse > maxHeap {
			maxHeap = curr.HeapInuse
		}
		if curr.HeapInuse < minHeap {
			minHeap = curr.HeapInuse
		}
		
		growth := int64(curr.HeapInuse) - int64(prev.HeapInuse)
		if growth > 0 {
			totalGrowth += growth
			growthEvents++
		}
	}
	
	analysis := HeapAnalysis{
		InitialSize:    first.HeapInuse,
		CurrentSize:    last.HeapInuse,
		MaxSize:        maxHeap,
		MinSize:        minHeap,
		TotalGrowth:    totalGrowth,
		GrowthEvents:   growthEvents,
		CurrentObjects: last.HeapObjects,
		Fragmentation:  float64(last.HeapSys-last.HeapInuse) / float64(last.HeapSys),
	}
	
	if growthEvents > 0 {
		analysis.AverageGrowth = float64(totalGrowth) / float64(growthEvents)
	}
	
	return analysis
}

// HeapAnalysis contains heap growth analysis results
type HeapAnalysis struct {
	InitialSize    uint64
	CurrentSize    uint64
	MaxSize        uint64
	MinSize        uint64
	TotalGrowth    int64
	AverageGrowth  float64
	GrowthEvents   int
	CurrentObjects uint64
	Fragmentation  float64
}

// GCTraceParser parses GC trace output
type GCTraceParser struct {
	events []GCTraceEvent
}

// GCTraceEvent represents a parsed GC trace event
type GCTraceEvent struct {
	GCNum       int
	StartTime   float64
	EndTime     float64
	STWTime     float64
	HeapBefore  uint64
	HeapAfter   uint64
	HeapGoal    uint64
	CPUTime     float64
	AssistTime  float64
	BackgroundTime float64
	SweepTime   float64
	AllocRate   float64
}

// NewGCTraceParser creates a new GC trace parser
func NewGCTraceParser() *GCTraceParser {
	return &GCTraceParser{
		events: make([]GCTraceEvent, 0),
	}
}

// ParseGCTrace parses GC trace from reader
func (gtp *GCTraceParser) ParseGCTrace(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	gcRegex := regexp.MustCompile(`gc (\d+) @([\d.]+)s (\d+)%: ([\d.]+)\+([\d.]+)\+([\d.]+) ms clock, ([\d.]+)\+([\d.]+)/([\d.]+)/([\d.]+)\+([\d.]+) ms cpu, (\d+)->(\d+)->(\d+) MB, (\d+) MB goal, (\d+) P`)
	
	for scanner.Scan() {
		line := scanner.Text()
		matches := gcRegex.FindStringSubmatch(line)
		
		if len(matches) >= 16 {
			event, err := gtp.parseGCTraceLine(matches)
			if err != nil {
				continue // Skip malformed lines
			}
			gtp.events = append(gtp.events, event)
		}
	}
	
	return scanner.Err()
}

// parseGCTraceLine parses a single GC trace line
func (gtp *GCTraceParser) parseGCTraceLine(matches []string) (GCTraceEvent, error) {
	gcNum, _ := strconv.Atoi(matches[1])
	startTime, _ := strconv.ParseFloat(matches[2], 64)
	
	// Parse timing information
	markTime, _ := strconv.ParseFloat(matches[4], 64)
	scanTime, _ := strconv.ParseFloat(matches[5], 64)
	sweepTime, _ := strconv.ParseFloat(matches[6], 64)
	
	// Parse CPU times
	cpuMark, _ := strconv.ParseFloat(matches[7], 64)
	cpuAssist, _ := strconv.ParseFloat(matches[8], 64)
	cpuBackground, _ := strconv.ParseFloat(matches[9], 64)
	cpuSweep, _ := strconv.ParseFloat(matches[11], 64)
	
	// Parse heap sizes (convert MB to bytes)
	heapBefore, _ := strconv.ParseUint(matches[12], 10, 64)
	heapAfter, _ := strconv.ParseUint(matches[13], 10, 64)
	heapGoal, _ := strconv.ParseUint(matches[14], 10, 64)
	
	heapBefore *= 1024 * 1024
	heapAfter *= 1024 * 1024
	heapGoal *= 1024 * 1024
	
	return GCTraceEvent{
		GCNum:          gcNum,
		StartTime:      startTime,
		EndTime:        startTime + (markTime+scanTime+sweepTime)/1000,
		STWTime:        markTime + scanTime,
		HeapBefore:     heapBefore,
		HeapAfter:      heapAfter,
		HeapGoal:       heapGoal,
		CPUTime:        cpuMark + cpuAssist + cpuBackground + cpuSweep,
		AssistTime:     cpuAssist,
		BackgroundTime: cpuBackground,
		SweepTime:      cpuSweep,
	}, nil
}

// GetTraceEvents returns all parsed trace events
func (gtp *GCTraceParser) GetTraceEvents() []GCTraceEvent {
	return gtp.events
}

// ParseTraceLine parses a single GC trace line and returns the event
func (gtp *GCTraceParser) ParseTraceLine(line string) (TestGCTraceEvent, error) {
	gcRegex := regexp.MustCompile(`gc (\d+) @([\d.]+)s (\d+)%: ([\d.]+)\+([\d.]+)\+([\d.]+) ms clock, ([\d.]+)\+([\d.]+)/([\d.]+)/([\d.]+)\+([\d.]+) ms cpu, (\d+)->(\d+)->(\d+) MB, (\d+) MB goal, (\d+) P`)
	
	matches := gcRegex.FindStringSubmatch(line)
	if len(matches) < 17 {
		return TestGCTraceEvent{}, fmt.Errorf("invalid GC trace line format")
	}
	
	// Parse the matches into a simplified test event
	cycleNumber, _ := strconv.Atoi(matches[1])
	timestamp, _ := strconv.ParseFloat(matches[2], 64)
	markTime, _ := strconv.ParseFloat(matches[4], 64)
	scanTime, _ := strconv.ParseFloat(matches[5], 64)
	stwTime := markTime + scanTime
	heapBefore, _ := strconv.Atoi(matches[12])
	heapAfter, _ := strconv.Atoi(matches[14])  // Skip matches[13] which is heap during GC
	heapGoal, _ := strconv.Atoi(matches[15])
	
	// Calculate total CPU time
	cpuMark, _ := strconv.ParseFloat(matches[7], 64)
	cpuAssist, _ := strconv.ParseFloat(matches[8], 64)
	cpuBackground, _ := strconv.ParseFloat(matches[9], 64)
	cpuSweep, _ := strconv.ParseFloat(matches[11], 64)
	cpuTime := cpuMark + cpuAssist + cpuBackground + cpuSweep
	
	return TestGCTraceEvent{
		CycleNumber:    cycleNumber,
		Timestamp:      timestamp,
		STWTime:        stwTime,
		HeapSizeBefore: heapBefore,
		HeapSizeAfter:  heapAfter,
		HeapGoal:       heapGoal,
		CPUTime:        cpuTime,
	}, nil
}

// TestGCTraceEvent represents a simplified GC trace event for testing
type TestGCTraceEvent struct {
	CycleNumber     int
	Timestamp       float64
	STWTime         float64
	HeapSizeBefore  int
	HeapSizeAfter   int
	HeapGoal        int
	CPUTime         float64
}

// AnalyzeTraceEvents analyzes parsed trace events
func (gtp *GCTraceParser) AnalyzeTraceEvents() TraceAnalysis {
	if len(gtp.events) == 0 {
		return TraceAnalysis{}
	}
	
	totalSTW := 0.0
	totalCPU := 0.0
	totalDuration := 0.0
	maxSTW := 0.0
	maxHeap := uint64(0)
	minHeap := ^uint64(0)
	
	for _, event := range gtp.events {
		totalSTW += event.STWTime
		totalCPU += event.CPUTime
		totalDuration += event.EndTime - event.StartTime
		
		if event.STWTime > maxSTW {
			maxSTW = event.STWTime
		}
		
		if event.HeapBefore > maxHeap {
			maxHeap = event.HeapBefore
		}
		if event.HeapAfter < minHeap {
			minHeap = event.HeapAfter
		}
	}
	
	count := float64(len(gtp.events))
	first := gtp.events[0]
	last := gtp.events[len(gtp.events)-1]
	totalTime := last.EndTime - first.StartTime
	
	return TraceAnalysis{
		EventCount:     len(gtp.events),
		TotalTime:      totalTime,
		AverageSTW:     totalSTW / count,
		MaxSTW:         maxSTW,
		TotalSTW:       totalSTW,
		AverageCPU:     totalCPU / count,
		TotalCPU:       totalCPU,
		GCFrequency:    count / totalTime, // GCs per second
		MaxHeapSize:    maxHeap,
		MinHeapSize:    minHeap,
		STWPercentage:  (totalSTW / 1000) / totalTime * 100, // Convert ms to s
		CPUPercentage:  (totalCPU / 1000) / totalTime * 100,
	}
}

// TraceAnalysis contains trace analysis results
type TraceAnalysis struct {
	EventCount    int
	TotalTime     float64 // seconds
	AverageSTW    float64 // milliseconds
	MaxSTW        float64 // milliseconds
	TotalSTW      float64 // milliseconds
	AverageCPU    float64 // milliseconds
	TotalCPU      float64 // milliseconds
	GCFrequency   float64 // GCs per second
	MaxHeapSize   uint64  // bytes
	MinHeapSize   uint64  // bytes
	STWPercentage float64 // percentage of total time
	CPUPercentage float64 // percentage of total time
}

// ReportGenerator generates comprehensive GC reports
type ReportGenerator struct {
	analyzer *GCAnalyzer
	parser   *GCTraceParser
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(analyzer *GCAnalyzer, parser *GCTraceParser) *ReportGenerator {
	return &ReportGenerator{
		analyzer: analyzer,
		parser:   parser,
	}
}

// GenerateReport generates a comprehensive GC analysis report
func (rg *ReportGenerator) GenerateReport() string {
	var report strings.Builder
	
	report.WriteString("=== Garbage Collection Analysis Report ===\n\n")
	
	// Current statistics
	stats := rg.analyzer.GetCurrentStats()
	report.WriteString("Current GC Statistics:\n")
	report.WriteString(fmt.Sprintf("  Number of GCs: %d\n", stats.NumGC))
	report.WriteString(fmt.Sprintf("  Total pause time: %v\n", stats.TotalPause))
	report.WriteString(fmt.Sprintf("  Last GC: %v\n", stats.LastGC.Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("  Heap size: %.2f MB\n", float64(stats.HeapSize)/(1024*1024)))
	report.WriteString(fmt.Sprintf("  Heap objects: %d\n", stats.HeapObjects))
	report.WriteString(fmt.Sprintf("  GC CPU fraction: %.4f\n", stats.GCCPUFraction))
	report.WriteString("\n")
	
	// Pause analysis
	pauseAnalysis := rg.analyzer.AnalyzePausePatterns()
	if pauseAnalysis.Count > 0 {
		report.WriteString("Pause Time Analysis:\n")
		report.WriteString(fmt.Sprintf("  Count: %d\n", pauseAnalysis.Count))
		report.WriteString(fmt.Sprintf("  Average: %v\n", pauseAnalysis.Average))
		report.WriteString(fmt.Sprintf("  Min: %v\n", pauseAnalysis.Min))
		report.WriteString(fmt.Sprintf("  Max: %v\n", pauseAnalysis.Max))
		report.WriteString(fmt.Sprintf("  P50: %v\n", pauseAnalysis.P50))
		report.WriteString(fmt.Sprintf("  P90: %v\n", pauseAnalysis.P90))
		report.WriteString(fmt.Sprintf("  P95: %v\n", pauseAnalysis.P95))
		report.WriteString(fmt.Sprintf("  P99: %v\n", pauseAnalysis.P99))
		report.WriteString("\n")
	}
	
	// Allocation analysis
	allocAnalysis := rg.analyzer.AnalyzeAllocationRate()
	if allocAnalysis.SampleCount > 0 {
		report.WriteString("Allocation Rate Analysis:\n")
		report.WriteString(fmt.Sprintf("  Sample count: %d\n", allocAnalysis.SampleCount))
		report.WriteString(fmt.Sprintf("  Total allocated: %.2f MB\n", allocAnalysis.TotalAllocated))
		report.WriteString(fmt.Sprintf("  Average rate: %.2f MB/s\n", allocAnalysis.AverageRate))
		report.WriteString(fmt.Sprintf("  Min rate: %.2f MB/s\n", allocAnalysis.MinRate))
		report.WriteString(fmt.Sprintf("  Max rate: %.2f MB/s\n", allocAnalysis.MaxRate))
		report.WriteString(fmt.Sprintf("  P90 rate: %.2f MB/s\n", allocAnalysis.P90Rate))
		report.WriteString("\n")
	}
	
	// Heap analysis
	heapAnalysis := rg.analyzer.AnalyzeHeapGrowth()
	if heapAnalysis.GrowthEvents > 0 {
		report.WriteString("Heap Growth Analysis:\n")
		report.WriteString(fmt.Sprintf("  Initial size: %.2f MB\n", float64(heapAnalysis.InitialSize)/(1024*1024)))
		report.WriteString(fmt.Sprintf("  Current size: %.2f MB\n", float64(heapAnalysis.CurrentSize)/(1024*1024)))
		report.WriteString(fmt.Sprintf("  Max size: %.2f MB\n", float64(heapAnalysis.MaxSize)/(1024*1024)))
		report.WriteString(fmt.Sprintf("  Growth events: %d\n", heapAnalysis.GrowthEvents))
		report.WriteString(fmt.Sprintf("  Average growth: %.2f MB\n", heapAnalysis.AverageGrowth/(1024*1024)))
		report.WriteString(fmt.Sprintf("  Fragmentation: %.2f%%\n", heapAnalysis.Fragmentation*100))
		report.WriteString("\n")
	}
	
	// Trace analysis (if available)
	if rg.parser != nil {
		traceAnalysis := rg.parser.AnalyzeTraceEvents()
		if traceAnalysis.EventCount > 0 {
			report.WriteString("GC Trace Analysis:\n")
			report.WriteString(fmt.Sprintf("  Event count: %d\n", traceAnalysis.EventCount))
			report.WriteString(fmt.Sprintf("  Total time: %.2f s\n", traceAnalysis.TotalTime))
			report.WriteString(fmt.Sprintf("  GC frequency: %.2f GCs/s\n", traceAnalysis.GCFrequency))
			report.WriteString(fmt.Sprintf("  Average STW: %.2f ms\n", traceAnalysis.AverageSTW))
			report.WriteString(fmt.Sprintf("  Max STW: %.2f ms\n", traceAnalysis.MaxSTW))
			report.WriteString(fmt.Sprintf("  STW percentage: %.2f%%\n", traceAnalysis.STWPercentage))
			report.WriteString(fmt.Sprintf("  CPU percentage: %.2f%%\n", traceAnalysis.CPUPercentage))
			report.WriteString("\n")
		}
	}
	
	// Recommendations
	report.WriteString(rg.generateRecommendations(stats, pauseAnalysis, allocAnalysis))
	
	return report.String()
}

// generateRecommendations generates optimization recommendations
func (rg *ReportGenerator) generateRecommendations(stats GCStats, pauseAnalysis PauseAnalysis, allocAnalysis AllocationAnalysis) string {
	var recommendations strings.Builder
	recommendations.WriteString("Optimization Recommendations:\n")
	
	// High GC frequency
	if stats.GCCPUFraction > 0.1 {
		recommendations.WriteString("  - High GC CPU usage (>10%). Consider reducing allocation rate or increasing GOGC.\n")
	}
	
	// High pause times
	if pauseAnalysis.P95 > 10*time.Millisecond {
		recommendations.WriteString("  - High P95 pause times (>10ms). Consider optimizing large object allocations.\n")
	}
	
	// High allocation rate
	if allocAnalysis.AverageRate > 100 {
		recommendations.WriteString("  - High allocation rate (>100 MB/s). Consider object pooling or allocation reduction.\n")
	}
	
	// Memory efficiency
	if stats.HeapObjects > 1000000 {
		recommendations.WriteString("  - High object count (>1M). Consider object consolidation or pooling.\n")
	}
	
	recommendations.WriteString("\n")
	return recommendations.String()
}

// DemonstrationSuite orchestrates GC analysis demonstrations
type DemonstrationSuite struct {
	analyzer *GCAnalyzer
	parser   *GCTraceParser
	reporter *ReportGenerator
}

// NewDemonstrationSuite creates a new demonstration suite
func NewDemonstrationSuite() *DemonstrationSuite {
	analyzer := NewGCAnalyzer()
	parser := NewGCTraceParser()
	reporter := NewReportGenerator(analyzer, parser)
	
	return &DemonstrationSuite{
		analyzer: analyzer,
		parser:   parser,
		reporter: reporter,
	}
}

// RunAnalysisDemo runs a comprehensive GC analysis demonstration
func (ds *DemonstrationSuite) RunAnalysisDemo() {
	fmt.Println("GC Behavior Analysis Demonstration")
	fmt.Println("==================================")
	
	// Start monitoring
	ds.analyzer.StartMonitoringWithInterval(100 * time.Millisecond)
	defer ds.analyzer.StopMonitoring()
	
	// Generate some GC activity
	ds.generateGCActivity()
	
	// Wait for monitoring to collect data
	time.Sleep(500 * time.Millisecond)
	
	// Generate and display report
	report := ds.reporter.GenerateReport()
	fmt.Println(report)
	
	// Demonstrate trace parsing if trace file exists
	ds.demonstrateTraceAnalysis()
}

// generateGCActivity generates various allocation patterns to trigger GC
func (ds *DemonstrationSuite) generateGCActivity() {
	fmt.Println("Generating GC activity...")
	
	// Small object allocations
	for i := 0; i < 100000; i++ {
		_ = make([]byte, 64)
	}
	
	// Medium object allocations
	for i := 0; i < 1000; i++ {
		_ = make([]byte, 8192)
	}
	
	// Large object allocations
	for i := 0; i < 10; i++ {
		_ = make([]byte, 1024*1024)
	}
	
	// Force GC
	runtime.GC()
	
	fmt.Println("GC activity generated.")
}

// demonstrateTraceAnalysis demonstrates GC trace analysis
func (ds *DemonstrationSuite) demonstrateTraceAnalysis() {
	// This would typically parse a real GC trace file
	// For demonstration, we'll show how it would work
	fmt.Println("\nGC Trace Analysis:")
	fmt.Println("To analyze GC traces, run your application with:")
	fmt.Println("  GODEBUG=gctrace=1 go run your_app.go 2> gc_trace.log")
	fmt.Println("Then parse the trace file with the GCTraceParser.")
}

// main demonstrates GC analysis capabilities
func main() {
	// Set up GC environment for demonstration
	debug.SetGCPercent(100) // Default GOGC
	
	suite := NewDemonstrationSuite()
	suite.RunAnalysisDemo()
	
	fmt.Println("\nGC analysis demonstration completed!")
}