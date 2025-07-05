package main

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// AllocationTracker tracks memory allocation patterns and hotspots
type AllocationTracker struct {
	allocationSites map[string]*AllocationSite
	totalAllocations uint64
	totalBytes       uint64
	startTime        time.Time
	mu               sync.RWMutex
	sampling         bool
	sampleRate       int // 1 in N allocations
	maxSites         int
}

// AllocationSite represents a specific allocation location
type AllocationSite struct {
	Location     string    // Function and line information
	Count        uint64    // Number of allocations
	TotalBytes   uint64    // Total bytes allocated
	AverageSize  float64   // Average allocation size
	LastSeen     time.Time // Last allocation time
	SizeHist     []uint64  // Size histogram
	Frequency    float64   // Allocations per second
	GCPressure   float64   // Estimated GC pressure contribution
}

// NewAllocationTracker creates a new allocation tracker
func NewAllocationTracker() *AllocationTracker {
	return &AllocationTracker{
		allocationSites: make(map[string]*AllocationSite),
		startTime:       time.Now(),
		sampleRate:      1,    // Track every allocation by default
		maxSites:        1000, // Limit memory usage
	}
}

// SetSampling configures allocation sampling
func (at *AllocationTracker) SetSampling(enabled bool, rate int) {
	at.mu.Lock()
	defer at.mu.Unlock()
	
	at.sampling = enabled
	if rate > 0 {
		at.sampleRate = rate
	}
}

// TrackAllocation records an allocation event
func (at *AllocationTracker) TrackAllocation(size uint64, location string) {
	if at.sampling && at.totalAllocations%uint64(at.sampleRate) != 0 {
		return
	}
	
	at.mu.Lock()
	defer at.mu.Unlock()
	
	at.totalAllocations++
	at.totalBytes += size
	
	site, exists := at.allocationSites[location]
	if !exists {
		// Limit number of tracked sites
		if len(at.allocationSites) >= at.maxSites {
			return
		}
		
		site = &AllocationSite{
			Location:  location,
			SizeHist:  make([]uint64, 0, 100),
			LastSeen:  time.Now(),
		}
		at.allocationSites[location] = site
	}
	
	site.Count++
	site.TotalBytes += size
	site.AverageSize = float64(site.TotalBytes) / float64(site.Count)
	site.LastSeen = time.Now()
	
	// Update size histogram (keep last 100 sizes)
	site.SizeHist = append(site.SizeHist, size)
	if len(site.SizeHist) > 100 {
		site.SizeHist = site.SizeHist[1:]
	}
	
	// Calculate frequency
	duration := time.Since(at.startTime).Seconds()
	if duration > 0 {
		site.Frequency = float64(site.Count) / duration
	}
	
	// Estimate GC pressure (simplified)
	site.GCPressure = site.Frequency * site.AverageSize
}

// GetTopAllocators returns the top allocation sites by various metrics
func (at *AllocationTracker) GetTopAllocators(limit int) TopAllocators {
	at.mu.RLock()
	defer at.mu.RUnlock()
	
	sites := make([]*AllocationSite, 0, len(at.allocationSites))
	for _, site := range at.allocationSites {
		sites = append(sites, site)
	}
	
	// Sort by different criteria
	byCount := make([]*AllocationSite, len(sites))
	copy(byCount, sites)
	sort.Slice(byCount, func(i, j int) bool {
		return byCount[i].Count > byCount[j].Count
	})
	
	byBytes := make([]*AllocationSite, len(sites))
	copy(byBytes, sites)
	sort.Slice(byBytes, func(i, j int) bool {
		return byBytes[i].TotalBytes > byBytes[j].TotalBytes
	})
	
	byFrequency := make([]*AllocationSite, len(sites))
	copy(byFrequency, sites)
	sort.Slice(byFrequency, func(i, j int) bool {
		return byFrequency[i].Frequency > byFrequency[j].Frequency
	})
	
	byGCPressure := make([]*AllocationSite, len(sites))
	copy(byGCPressure, sites)
	sort.Slice(byGCPressure, func(i, j int) bool {
		return byGCPressure[i].GCPressure > byGCPressure[j].GCPressure
	})
	
	// Limit results
	if limit > 0 {
		if len(byCount) > limit {
			byCount = byCount[:limit]
		}
		if len(byBytes) > limit {
			byBytes = byBytes[:limit]
		}
		if len(byFrequency) > limit {
			byFrequency = byFrequency[:limit]
		}
		if len(byGCPressure) > limit {
			byGCPressure = byGCPressure[:limit]
		}
	}
	
	return TopAllocators{
		ByCount:      byCount,
		ByBytes:      byBytes,
		ByFrequency:  byFrequency,
		ByGCPressure: byGCPressure,
	}
}

// TopAllocators contains allocation sites sorted by different criteria
type TopAllocators struct {
	ByCount      []*AllocationSite
	ByBytes      []*AllocationSite
	ByFrequency  []*AllocationSite
	ByGCPressure []*AllocationSite
}

// GetSummary returns allocation tracking summary
func (at *AllocationTracker) GetSummary() AllocationSummary {
	at.mu.RLock()
	defer at.mu.RUnlock()
	
	duration := time.Since(at.startTime)
	allocationRate := float64(at.totalAllocations) / duration.Seconds()
	byteRate := float64(at.totalBytes) / duration.Seconds() / (1024 * 1024) // MB/s
	
	return AllocationSummary{
		TotalAllocations: at.totalAllocations,
		TotalBytes:       at.totalBytes,
		UniqueSites:      len(at.allocationSites),
		Duration:         duration,
		AllocationRate:   allocationRate,
		ByteRate:         byteRate,
		AverageSize:      float64(at.totalBytes) / float64(at.totalAllocations),
	}
}

// AllocationSummary contains overall allocation statistics
type AllocationSummary struct {
	TotalAllocations uint64
	TotalBytes       uint64
	UniqueSites      int
	Duration         time.Duration
	AllocationRate   float64 // allocations per second
	ByteRate         float64 // MB per second
	AverageSize      float64 // average allocation size
}

// AllocationProfiler provides runtime allocation profiling
type AllocationProfiler struct {
	tracker    *AllocationTracker
	profiling  bool
	interval   time.Duration
	stopChan   chan struct{}
	lastStats  runtime.MemStats
	mu         sync.RWMutex
}

// NewAllocationProfiler creates a new allocation profiler
func NewAllocationProfiler(tracker *AllocationTracker) *AllocationProfiler {
	return &AllocationProfiler{
		tracker:  tracker,
		interval: 100 * time.Millisecond,
		stopChan: make(chan struct{}),
	}
}

// StartProfiling begins allocation profiling
func (ap *AllocationProfiler) StartProfiling() {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	
	if ap.profiling {
		return
	}
	
	ap.profiling = true
	runtime.ReadMemStats(&ap.lastStats)
	
	go ap.profileLoop()
}

// StopProfiling stops allocation profiling
func (ap *AllocationProfiler) StopProfiling() {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	
	if !ap.profiling {
		return
	}
	
	ap.profiling = false
	close(ap.stopChan)
	ap.stopChan = make(chan struct{})
}

// profileLoop continuously profiles allocations
func (ap *AllocationProfiler) profileLoop() {
	ticker := time.NewTicker(ap.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ap.stopChan:
			return
		case <-ticker.C:
			ap.sampleAllocations()
		}
	}
}

// sampleAllocations samples current allocation state
func (ap *AllocationProfiler) sampleAllocations() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	
	ap.mu.Lock()
	defer ap.mu.Unlock()
	
	// Calculate allocations since last sample
	allocDiff := stats.TotalAlloc - ap.lastStats.TotalAlloc
	mallocsDiff := stats.Mallocs - ap.lastStats.Mallocs
	
	if mallocsDiff > 0 {
		averageSize := float64(allocDiff) / float64(mallocsDiff)
		
		// Track allocation with estimated location
		location := ap.estimateAllocationLocation()
		ap.tracker.TrackAllocation(uint64(averageSize), location)
	}
	
	ap.lastStats = stats
}

// estimateAllocationLocation estimates allocation location from stack trace
func (ap *AllocationProfiler) estimateAllocationLocation() string {
	// Get stack trace
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	stack := string(buf[:n])
	
	// Parse stack to find likely allocation site
	// This is a simplified version - real implementation would be more sophisticated
	lines := strings.Split(stack, "\n")
	for i, line := range lines {
		if strings.Contains(line, "runtime.") {
			continue
		}
		if i+1 < len(lines) && strings.Contains(lines[i+1], ".go:") {
			return fmt.Sprintf("%s @ %s", strings.TrimSpace(line), strings.TrimSpace(lines[i+1]))
		}
	}
	
	return "unknown"
}

// AllocationPattern represents different allocation patterns for testing
type AllocationPattern struct {
	Name        string
	Description string
	Execute     func(iterations int)
}

// AllocationPatternSuite provides various allocation patterns for testing
type AllocationPatternSuite struct {
	patterns []AllocationPattern
	tracker  *AllocationTracker
	profiler *AllocationProfiler
}

// NewAllocationPatternSuite creates a new allocation pattern suite
func NewAllocationPatternSuite() *AllocationPatternSuite {
	tracker := NewAllocationTracker()
	profiler := NewAllocationProfiler(tracker)
	
	suite := &AllocationPatternSuite{
		tracker:  tracker,
		profiler: profiler,
	}
	
	suite.initializePatterns()
	return suite
}

// initializePatterns initializes various allocation patterns
func (aps *AllocationPatternSuite) initializePatterns() {
	aps.patterns = []AllocationPattern{
		{
			Name:        "SmallObjects",
			Description: "Frequent small object allocations",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					_ = make([]byte, 64)
					aps.tracker.TrackAllocation(64, "SmallObjects")
				}
			},
		},
		{
			Name:        "MediumObjects",
			Description: "Medium-sized object allocations",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					_ = make([]byte, 8192)
					aps.tracker.TrackAllocation(8192, "MediumObjects")
				}
			},
		},
		{
			Name:        "LargeObjects",
			Description: "Large object allocations",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					_ = make([]byte, 1024*1024)
					aps.tracker.TrackAllocation(1024*1024, "LargeObjects")
				}
			},
		},
		{
			Name:        "VariableSizeObjects",
			Description: "Variable-sized object allocations",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					size := uint64(64 + (i%1000)*32)
					_ = make([]byte, size)
					aps.tracker.TrackAllocation(size, "VariableSizeObjects")
				}
			},
		},
		{
			Name:        "StringConcatenation",
			Description: "String concatenation allocations",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					result := ""
					for j := 0; j < 10; j++ {
						result += fmt.Sprintf("item_%d_", j)
					}
					aps.tracker.TrackAllocation(uint64(len(result)), "StringConcatenation")
					_ = result
				}
			},
		},
		{
			Name:        "SliceGrowth",
			Description: "Slice growth allocations",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					slice := make([]int, 0)
					for j := 0; j < 1000; j++ {
						slice = append(slice, j)
						if cap(slice) > len(slice) {
							// Estimate allocation size
							size := uint64(cap(slice) * int(unsafe.Sizeof(int(0))))
							aps.tracker.TrackAllocation(size, "SliceGrowth")
						}
					}
					_ = slice
				}
			},
		},
		{
			Name:        "MapAllocations",
			Description: "Map allocation and growth",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					m := make(map[string]int)
					for j := 0; j < 100; j++ {
						key := fmt.Sprintf("key_%d", j)
						m[key] = j
						// Estimate allocation for map growth
						if j%10 == 0 {
							aps.tracker.TrackAllocation(uint64(len(key)+8), "MapAllocations")
						}
					}
					_ = m
				}
			},
		},
		{
			Name:        "InterfaceBoxing",
			Description: "Interface boxing allocations",
			Execute: func(iterations int) {
				for i := 0; i < iterations; i++ {
					var interfaces []interface{}
					for j := 0; j < 100; j++ {
						interfaces = append(interfaces, j) // Boxing allocation
						aps.tracker.TrackAllocation(16, "InterfaceBoxing") // Interface overhead
					}
					_ = interfaces
				}
			},
		},
	}
}

// RunPattern executes a specific allocation pattern
func (aps *AllocationPatternSuite) RunPattern(patternName string, iterations int) error {
	for _, pattern := range aps.patterns {
		if pattern.Name == patternName {
			fmt.Printf("Running pattern: %s (%s)\n", pattern.Name, pattern.Description)
			start := time.Now()
			pattern.Execute(iterations)
			duration := time.Since(start)
			fmt.Printf("Pattern completed in %v\n", duration)
			return nil
		}
	}
	return fmt.Errorf("pattern not found: %s", patternName)
}

// RunAllPatterns executes all allocation patterns
func (aps *AllocationPatternSuite) RunAllPatterns(iterations int) {
	fmt.Println("Running all allocation patterns...")
	
	for _, pattern := range aps.patterns {
		fmt.Printf("\nExecuting: %s\n", pattern.Name)
		start := time.Now()
		pattern.Execute(iterations)
		duration := time.Since(start)
		fmt.Printf("Completed in %v\n", duration)
		
		// Force GC between patterns
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
	}
}

// GetPatternNames returns names of all available patterns
func (aps *AllocationPatternSuite) GetPatternNames() []string {
	names := make([]string, len(aps.patterns))
	for i, pattern := range aps.patterns {
		names[i] = pattern.Name
	}
	return names
}

// GenerateAllocationReport generates a comprehensive allocation report
func (aps *AllocationPatternSuite) GenerateAllocationReport() string {
	var report strings.Builder
	
	report.WriteString("=== Allocation Tracking Report ===\n\n")
	
	// Summary
	summary := aps.tracker.GetSummary()
	report.WriteString("Summary:\n")
	report.WriteString(fmt.Sprintf("  Total allocations: %d\n", summary.TotalAllocations))
	report.WriteString(fmt.Sprintf("  Total bytes: %.2f MB\n", float64(summary.TotalBytes)/(1024*1024)))
	report.WriteString(fmt.Sprintf("  Unique sites: %d\n", summary.UniqueSites))
	report.WriteString(fmt.Sprintf("  Duration: %v\n", summary.Duration))
	report.WriteString(fmt.Sprintf("  Allocation rate: %.2f allocs/s\n", summary.AllocationRate))
	report.WriteString(fmt.Sprintf("  Byte rate: %.2f MB/s\n", summary.ByteRate))
	report.WriteString(fmt.Sprintf("  Average size: %.2f bytes\n", summary.AverageSize))
	report.WriteString("\n")
	
	// Top allocators
	topAllocators := aps.tracker.GetTopAllocators(5)
	
	report.WriteString("Top Allocators by Count:\n")
	for i, site := range topAllocators.ByCount {
		report.WriteString(fmt.Sprintf("  %d. %s: %d allocations, %.2f MB\n",
			i+1, site.Location, site.Count, float64(site.TotalBytes)/(1024*1024)))
	}
	report.WriteString("\n")
	
	report.WriteString("Top Allocators by Bytes:\n")
	for i, site := range topAllocators.ByBytes {
		report.WriteString(fmt.Sprintf("  %d. %s: %.2f MB, %d allocations\n",
			i+1, site.Location, float64(site.TotalBytes)/(1024*1024), site.Count))
	}
	report.WriteString("\n")
	
	report.WriteString("Top Allocators by GC Pressure:\n")
	for i, site := range topAllocators.ByGCPressure {
		report.WriteString(fmt.Sprintf("  %d. %s: %.2f pressure, %.2f freq\n",
			i+1, site.Location, site.GCPressure, site.Frequency))
	}
	report.WriteString("\n")
	
	return report.String()
}

// DemonstrateAllocationTracking demonstrates allocation tracking capabilities
func DemonstrateAllocationTracking() {
	fmt.Println("Allocation Tracking Demonstration")
	fmt.Println("================================")
	
	suite := NewAllocationPatternSuite()
	
	// Start profiling
	suite.profiler.StartProfiling()
	defer suite.profiler.StopProfiling()
	
	// Run some patterns
	suite.RunPattern("SmallObjects", 10000)
	suite.RunPattern("MediumObjects", 1000)
	suite.RunPattern("LargeObjects", 10)
	suite.RunPattern("StringConcatenation", 1000)
	
	// Generate and display report
	report := suite.GenerateAllocationReport()
	fmt.Println(report)
	
	fmt.Println("Allocation tracking demonstration completed!")
}