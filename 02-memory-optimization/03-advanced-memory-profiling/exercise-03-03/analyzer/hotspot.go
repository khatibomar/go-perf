package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// HotspotInfo represents information about an allocation hotspot
type HotspotInfo struct {
	Location        string    `json:"location"`
	Function        string    `json:"function"`
	File            string    `json:"file"`
	Line            int       `json:"line"`
	AllocationRate  float64   `json:"allocation_rate"`
	ObjectCount     uint64    `json:"object_count"`
	TotalBytes      uint64    `json:"total_bytes"`
	AverageSize     float64   `json:"average_size"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	SampleCount     int       `json:"sample_count"`
	StackTrace      []string  `json:"stack_trace"`
	Severity        string    `json:"severity"`
	Recommendation  string    `json:"recommendation"`
}

// HotspotAnalyzer analyzes allocation hotspots
type HotspotAnalyzer struct {
	config          *Config
	hotspots        map[string]*HotspotInfo
	mutex           sync.RWMutex
	running         bool
	stopChan        chan struct{}
	samplingTicker  *time.Ticker
	thresholdBytes  uint64
	thresholdRate   float64
}

// NewHotspotAnalyzer creates a new hotspot analyzer
func NewHotspotAnalyzer(config *Config) *HotspotAnalyzer {
	return &HotspotAnalyzer{
		config:         config,
		hotspots:       make(map[string]*HotspotInfo),
		stopChan:       make(chan struct{}),
		thresholdBytes: 1024 * 1024, // 1MB threshold
		thresholdRate:  100,         // 100 allocs/sec threshold
	}
}

// Start begins hotspot analysis
func (ha *HotspotAnalyzer) Start(ctx context.Context) error {
	if ha.running {
		return fmt.Errorf("hotspot analyzer is already running")
	}

	ha.running = true
	ha.samplingTicker = time.NewTicker(ha.config.SamplingRate)

	// Start analysis goroutine
	go ha.analyzeHotspots(ctx)

	return nil
}

// Stop stops hotspot analysis
func (ha *HotspotAnalyzer) Stop() {
	if !ha.running {
		return
	}

	ha.running = false
	if ha.samplingTicker != nil {
		ha.samplingTicker.Stop()
	}
	close(ha.stopChan)
}

// GetHotspots returns current hotspots sorted by severity
func (ha *HotspotAnalyzer) GetHotspots() []*HotspotInfo {
	ha.mutex.RLock()
	defer ha.mutex.RUnlock()

	hotspots := make([]*HotspotInfo, 0, len(ha.hotspots))
	for _, hotspot := range ha.hotspots {
		hotspots = append(hotspots, hotspot)
	}

	// Sort by allocation rate (descending)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].AllocationRate > hotspots[j].AllocationRate
	})

	return hotspots
}

// GetTopHotspots returns the top N hotspots
func (ha *HotspotAnalyzer) GetTopHotspots(n int) []*HotspotInfo {
	allHotspots := ha.GetHotspots()
	if len(allHotspots) <= n {
		return allHotspots
	}
	return allHotspots[:n]
}

// GetHotspotsByFunction returns hotspots grouped by function
func (ha *HotspotAnalyzer) GetHotspotsByFunction() map[string][]*HotspotInfo {
	ha.mutex.RLock()
	defer ha.mutex.RUnlock()

	byFunction := make(map[string][]*HotspotInfo)
	for _, hotspot := range ha.hotspots {
		byFunction[hotspot.Function] = append(byFunction[hotspot.Function], hotspot)
	}

	return byFunction
}

// GetHotspotsByFile returns hotspots grouped by file
func (ha *HotspotAnalyzer) GetHotspotsByFile() map[string][]*HotspotInfo {
	ha.mutex.RLock()
	defer ha.mutex.RUnlock()

	byFile := make(map[string][]*HotspotInfo)
	for _, hotspot := range ha.hotspots {
		byFile[hotspot.File] = append(byFile[hotspot.File], hotspot)
	}

	return byFile
}

// SaveHotspots saves hotspot analysis to a file
func (ha *HotspotAnalyzer) SaveHotspots(filename string) error {
	ha.mutex.RLock()
	defer ha.mutex.RUnlock()

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	hotspots := make([]*HotspotInfo, 0, len(ha.hotspots))
	for _, hotspot := range ha.hotspots {
		hotspots = append(hotspots, hotspot)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(hotspots)
}

// LoadHotspots loads hotspot analysis from a file
func (ha *HotspotAnalyzer) LoadHotspots(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var hotspots []*HotspotInfo
	if err := json.NewDecoder(file).Decode(&hotspots); err != nil {
		return err
	}

	ha.mutex.Lock()
	ha.hotspots = make(map[string]*HotspotInfo)
	for _, hotspot := range hotspots {
		ha.hotspots[hotspot.Location] = hotspot
	}
	ha.mutex.Unlock()

	return nil
}

// analyzeHotspots performs continuous hotspot analysis
func (ha *HotspotAnalyzer) analyzeHotspots(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ha.stopChan:
			return
		case <-ha.samplingTicker.C:
			ha.sampleAllocations()
		}
	}
}

// sampleAllocations samples current allocations and updates hotspots
func (ha *HotspotAnalyzer) sampleAllocations() {
	// Get stack traces for current goroutines
	buf := make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	stackData := string(buf[:n])

	// Parse stack traces
	allocationSites := ha.parseStackTraces(stackData)

	// Update hotspots
	ha.updateHotspots(allocationSites)
}

// parseStackTraces parses stack trace data to find allocation sites
func (ha *HotspotAnalyzer) parseStackTraces(stackData string) map[string]*AllocationSite {
	allocationSites := make(map[string]*AllocationSite)
	lines := strings.Split(stackData, "\n")

	var currentGoroutine string
	var currentStack []string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect goroutine header
		if strings.HasPrefix(line, "goroutine ") {
			// Process previous goroutine if any
			if currentGoroutine != "" && len(currentStack) > 0 {
				ha.processGoroutineStack(currentGoroutine, currentStack, allocationSites)
			}
			currentGoroutine = line
			currentStack = nil
			continue
		}

		// Collect stack frames
		if strings.Contains(line, "(") && strings.Contains(line, ")") {
			currentStack = append(currentStack, line)
			// Also add the next line if it contains file:line info
			if i+1 < len(lines) && strings.Contains(lines[i+1], ":") {
				currentStack = append(currentStack, strings.TrimSpace(lines[i+1]))
			}
		}
	}

	// Process last goroutine
	if currentGoroutine != "" && len(currentStack) > 0 {
		ha.processGoroutineStack(currentGoroutine, currentStack, allocationSites)
	}

	return allocationSites
}

// AllocationSite represents an allocation site
type AllocationSite struct {
	Location   string
	Function   string
	File       string
	Line       int
	StackTrace []string
	Count      int
}

// processGoroutineStack processes a single goroutine's stack
func (ha *HotspotAnalyzer) processGoroutineStack(goroutine string, stack []string, sites map[string]*AllocationSite) {
	for i := 0; i < len(stack); i += 2 {
		if i+1 >= len(stack) {
			break
		}

		funcLine := stack[i]
		fileLine := stack[i+1]

		// Parse function name
		funcName := ha.extractFunctionName(funcLine)
		if funcName == "" {
			continue
		}

		// Parse file and line
		file, line := ha.extractFileAndLine(fileLine)
		if file == "" {
			continue
		}

		// Create location key
		location := fmt.Sprintf("%s:%d", file, line)

		// Update or create allocation site
		if site, exists := sites[location]; exists {
			site.Count++
		} else {
			sites[location] = &AllocationSite{
				Location:   location,
				Function:   funcName,
				File:       file,
				Line:       line,
				StackTrace: stack,
				Count:      1,
			}
		}
	}
}

// extractFunctionName extracts function name from stack line
func (ha *HotspotAnalyzer) extractFunctionName(line string) string {
	// Example: "main.allocateMemory(0x1, 0x2)"
	if idx := strings.Index(line, "("); idx > 0 {
		return strings.TrimSpace(line[:idx])
	}
	return ""
}

// extractFileAndLine extracts file path and line number
func (ha *HotspotAnalyzer) extractFileAndLine(line string) (string, int) {
	// Example: "/path/to/file.go:123 +0x45"
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", 0
	}

	fileLine := parts[0]
	if idx := strings.LastIndex(fileLine, ":"); idx > 0 {
		file := fileLine[:idx]
		lineStr := fileLine[idx+1:]
		if line := parseInt(lineStr); line > 0 {
			return file, line
		}
	}

	return "", 0
}

// parseInt safely parses an integer
func parseInt(s string) int {
	var result int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			break
		}
	}
	return result
}

// updateHotspots updates hotspot information based on allocation sites
func (ha *HotspotAnalyzer) updateHotspots(sites map[string]*AllocationSite) {
	ha.mutex.Lock()
	defer ha.mutex.Unlock()

	now := time.Now()

	for location, site := range sites {
		if hotspot, exists := ha.hotspots[location]; exists {
			// Update existing hotspot
			hotspot.LastSeen = now
			hotspot.SampleCount++
			hotspot.ObjectCount += uint64(site.Count)

			// Update allocation rate (exponential moving average)
			alpha := 0.1
			newRate := float64(site.Count) / ha.config.SamplingRate.Seconds()
			hotspot.AllocationRate = alpha*newRate + (1-alpha)*hotspot.AllocationRate

			// Update average size estimate
			if hotspot.ObjectCount > 0 {
				hotspot.AverageSize = float64(hotspot.TotalBytes) / float64(hotspot.ObjectCount)
			}
		} else {
			// Create new hotspot
			allocationRate := float64(site.Count) / ha.config.SamplingRate.Seconds()
			hotspot := &HotspotInfo{
				Location:       location,
				Function:       site.Function,
				File:           site.File,
				Line:           site.Line,
				AllocationRate: allocationRate,
				ObjectCount:    uint64(site.Count),
				TotalBytes:     uint64(site.Count) * 64, // Estimate 64 bytes per object
				AverageSize:    64,
				FirstSeen:      now,
				LastSeen:       now,
				SampleCount:    1,
				StackTrace:     site.StackTrace,
			}

			// Classify severity and generate recommendation
			ha.classifyHotspot(hotspot)
			ha.hotspots[location] = hotspot
		}
	}

	// Clean up old hotspots
	ha.cleanupOldHotspots(now)
}

// classifyHotspot classifies hotspot severity and generates recommendations
func (ha *HotspotAnalyzer) classifyHotspot(hotspot *HotspotInfo) {
	if hotspot.AllocationRate > ha.thresholdRate*10 {
		hotspot.Severity = "critical"
		hotspot.Recommendation = "Immediate optimization required - consider object pooling or reducing allocation frequency"
	} else if hotspot.AllocationRate > ha.thresholdRate*5 {
		hotspot.Severity = "high"
		hotspot.Recommendation = "High allocation rate detected - review algorithm efficiency and consider caching"
	} else if hotspot.AllocationRate > ha.thresholdRate {
		hotspot.Severity = "medium"
		hotspot.Recommendation = "Moderate allocation rate - monitor for growth and consider optimization"
	} else {
		hotspot.Severity = "low"
		hotspot.Recommendation = "Normal allocation rate - no immediate action required"
	}

	// Add specific recommendations based on function patterns
	if strings.Contains(hotspot.Function, "append") {
		hotspot.Recommendation += ". Consider pre-allocating slice capacity."
	} else if strings.Contains(hotspot.Function, "make") {
		hotspot.Recommendation += ". Consider reusing objects or using sync.Pool."
	} else if strings.Contains(hotspot.Function, "string") {
		hotspot.Recommendation += ". Consider using strings.Builder for string concatenation."
	}
}

// cleanupOldHotspots removes hotspots that haven't been seen recently
func (ha *HotspotAnalyzer) cleanupOldHotspots(now time.Time) {
	cleanupThreshold := 5 * time.Minute
	for location, hotspot := range ha.hotspots {
		if now.Sub(hotspot.LastSeen) > cleanupThreshold {
			delete(ha.hotspots, location)
		}
	}
}

// GetHotspotSummary returns a summary of hotspot analysis
func (ha *HotspotAnalyzer) GetHotspotSummary() map[string]interface{} {
	ha.mutex.RLock()
	defer ha.mutex.RUnlock()

	summary := make(map[string]interface{})
	summary["total_hotspots"] = len(ha.hotspots)

	severityCounts := make(map[string]int)
	var totalRate float64
	var maxRate float64

	for _, hotspot := range ha.hotspots {
		severityCounts[hotspot.Severity]++
		totalRate += hotspot.AllocationRate
		if hotspot.AllocationRate > maxRate {
			maxRate = hotspot.AllocationRate
		}
	}

	summary["severity_counts"] = severityCounts
	summary["total_allocation_rate"] = totalRate
	summary["max_allocation_rate"] = maxRate
	if len(ha.hotspots) > 0 {
		summary["average_allocation_rate"] = totalRate / float64(len(ha.hotspots))
	}

	return summary
}