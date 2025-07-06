package detector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"gonum.org/v1/gonum/stat"
)

// ResourceConfig holds configuration for resource leak detection
type ResourceConfig struct {
	Interval            time.Duration
	FileHandleThreshold int     // File handle count threshold
	NetworkThreshold    int     // Network connection count threshold
	GrowthThreshold     float64 // Resource growth rate threshold
	Verbose             bool
	TrackStacks         bool // Track stack traces for resource allocation
}

// ResourceType represents different types of system resources
type ResourceType string

const (
	ResourceTypeFile    ResourceType = "file"
	ResourceTypeNetwork ResourceType = "network"
	ResourceTypeMemory  ResourceType = "memory"
	ResourceTypeThread  ResourceType = "thread"
	ResourceTypeHandle  ResourceType = "handle"
)

// ResourceInfo represents information about a tracked resource
type ResourceInfo struct {
	ID          string       `json:"id"`
	Type        ResourceType `json:"type"`
	Path        string       `json:"path,omitempty"`
	Address     string       `json:"address,omitempty"`
	State       string       `json:"state"`
	PID         int32        `json:"pid"`
	CreatedAt   time.Time    `json:"created_at"`
	LastSeen    time.Time    `json:"last_seen"`
	StackTrace  []string     `json:"stack_trace,omitempty"`
	Size        int64        `json:"size,omitempty"`
	Flags       []string     `json:"flags,omitempty"`
}

// ResourceMetrics represents resource usage metrics at a point in time
type ResourceMetrics struct {
	Timestamp         time.Time                    `json:"timestamp"`
	FileHandles       int                          `json:"file_handles"`
	NetworkConnections int                         `json:"network_connections"`
	MemoryMapped      int64                        `json:"memory_mapped"`
	ThreadCount       int                          `json:"thread_count"`
	TotalHandles      int                          `json:"total_handles"`
	ResourcesByType   map[ResourceType]int         `json:"resources_by_type"`
	ResourcesByState  map[string]int               `json:"resources_by_state"`
	TopResourcePaths  map[string]int               `json:"top_resource_paths"`
	LeakCandidates    []string                     `json:"leak_candidates"`
}

// ResourceLeak represents a detected resource leak
type ResourceLeak struct {
	ID              string       `json:"id"`
	Type            ResourceType `json:"type"`
	Path            string       `json:"path,omitempty"`
	Count           int          `json:"count"`
	GrowthRate      float64      `json:"growth_rate"`
	Severity        string       `json:"severity"`
	Confidence      float64      `json:"confidence"`
	FirstDetected   time.Time    `json:"first_detected"`
	LatestDetected  time.Time    `json:"latest_detected"`
	Description     string       `json:"description"`
	Suggestion      string       `json:"suggestion"`
	AffectedResources []string   `json:"affected_resources"`
	StackTrace      []string     `json:"stack_trace,omitempty"`
}

// ResourceAnalysis represents the result of resource leak analysis
type ResourceAnalysis struct {
	StartTime       time.Time        `json:"start_time"`
	EndTime         time.Time        `json:"end_time"`
	Duration        time.Duration    `json:"duration"`
	Metrics         []ResourceMetrics `json:"metrics"`
	DetectedLeaks   []ResourceLeak   `json:"detected_leaks"`
	Recommendations []string         `json:"recommendations"`
	Statistics      ResourceStats    `json:"statistics"`
	Hotspots        []ResourceHotspot `json:"hotspots"`
	Trends          ResourceTrends   `json:"trends"`
}

// ResourceStats contains statistical analysis of resource usage
type ResourceStats struct {
	MeanFileHandles      float64            `json:"mean_file_handles"`
	MeanNetworkConns     float64            `json:"mean_network_connections"`
	PeakFileHandles      int                `json:"peak_file_handles"`
	PeakNetworkConns     int                `json:"peak_network_connections"`
	ResourceGrowthRates  map[ResourceType]float64 `json:"resource_growth_rates"`
	LeaksByType          map[ResourceType]int `json:"leaks_by_type"`
	LeaksBySeverity      map[string]int     `json:"leaks_by_severity"`
	MostLeakyPaths       map[string]int     `json:"most_leaky_paths"`
	PeakUsagePeriod      string             `json:"peak_usage_period"`
}

// ResourceHotspot represents a resource allocation hotspot
type ResourceHotspot struct {
	Location     string       `json:"location"`
	Type         ResourceType `json:"type"`
	Count        int          `json:"count"`
	LeakRate     float64      `json:"leak_rate"`
	Severity     string       `json:"severity"`
	Description  string       `json:"description"`
	Suggestion   string       `json:"suggestion"`
}

// ResourceTrends represents resource usage trends
type ResourceTrends struct {
	FileHandleTrend    string  `json:"file_handle_trend"`
	NetworkConnTrend   string  `json:"network_connection_trend"`
	MemoryMappedTrend  string  `json:"memory_mapped_trend"`
	OverallTrend       string  `json:"overall_trend"`
	GrowthAcceleration float64 `json:"growth_acceleration"`
}

// ResourceDetector monitors and detects resource leaks
type ResourceDetector struct {
	config    *ResourceConfig
	resources map[string]*ResourceInfo
	metrics   []ResourceMetrics
	leaks     []ResourceLeak
	running   bool
	stopChan  chan struct{}
	mutex     sync.RWMutex
	pid       int32
	process   *process.Process
}

// NewResourceDetector creates a new resource detector
func NewResourceDetector(config *ResourceConfig) *ResourceDetector {
	pid := int32(os.Getpid())
	proc, _ := process.NewProcess(pid)

	return &ResourceDetector{
		config:    config,
		resources: make(map[string]*ResourceInfo),
		metrics:   make([]ResourceMetrics, 0),
		leaks:     make([]ResourceLeak, 0),
		stopChan:  make(chan struct{}),
		pid:       pid,
		process:   proc,
	}
}

// Start begins resource monitoring
func (rd *ResourceDetector) Start() error {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	if rd.running {
		return fmt.Errorf("resource detector already running")
	}

	rd.running = true
	go rd.monitorLoop()

	if rd.config.Verbose {
		fmt.Printf("Resource detector started with interval %v\n", rd.config.Interval)
	}

	return nil
}

// Stop stops resource monitoring
func (rd *ResourceDetector) Stop() {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	if !rd.running {
		return
	}

	rd.running = false
	close(rd.stopChan)

	if rd.config.Verbose {
		fmt.Println("Resource detector stopped")
	}
}

// GetMetrics returns current metrics
func (rd *ResourceDetector) GetMetrics() []ResourceMetrics {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()

	// Return a copy to avoid race conditions
	metrics := make([]ResourceMetrics, len(rd.metrics))
	copy(metrics, rd.metrics)
	return metrics
}

// GetLeaks returns detected leaks
func (rd *ResourceDetector) GetLeaks() []ResourceLeak {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()

	// Return a copy to avoid race conditions
	leaks := make([]ResourceLeak, len(rd.leaks))
	copy(leaks, rd.leaks)
	return leaks
}

// AnalyzeResources performs comprehensive resource leak analysis
func (rd *ResourceDetector) AnalyzeResources() (*ResourceAnalysis, error) {
	rd.mutex.RLock()
	metrics := make([]ResourceMetrics, len(rd.metrics))
	copy(metrics, rd.metrics)
	leaks := make([]ResourceLeak, len(rd.leaks))
	copy(leaks, rd.leaks)
	rd.mutex.RUnlock()

	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics available for analysis")
	}

	analysis := &ResourceAnalysis{
		StartTime:     metrics[0].Timestamp,
		EndTime:       metrics[len(metrics)-1].Timestamp,
		Metrics:       metrics,
		DetectedLeaks: leaks,
	}
	analysis.Duration = analysis.EndTime.Sub(analysis.StartTime)

	// Generate recommendations
	analysis.Recommendations = rd.generateRecommendations(analysis)

	// Calculate statistics
	analysis.Statistics = rd.calculateStatistics(metrics, leaks)

	// Identify hotspots
	analysis.Hotspots = rd.identifyHotspots(leaks)

	// Analyze trends
	analysis.Trends = rd.analyzeTrends(metrics)

	return analysis, nil
}

// SaveResults saves analysis results to files
func (rd *ResourceDetector) SaveResults(outputDir string) error {
	analysis, err := rd.AnalyzeResources()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Save detailed analysis
	analysisFile := filepath.Join(outputDir, "resource_analysis.json")
	file, err := os.Create(analysisFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(analysis); err != nil {
		return err
	}

	// Save detected leaks
	leaksFile := filepath.Join(outputDir, "resource_leaks.json")
	file, err = os.Create(leaksFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder = json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rd.GetLeaks())
}

// monitorLoop runs the monitoring loop
func (rd *ResourceDetector) monitorLoop() {
	ticker := time.NewTicker(rd.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rd.collectMetrics()
			rd.detectLeaks()
		case <-rd.stopChan:
			return
		}
	}
}

// collectMetrics collects current resource metrics
func (rd *ResourceDetector) collectMetrics() {
	metrics := ResourceMetrics{
		Timestamp:        time.Now(),
		ResourcesByType:  make(map[ResourceType]int),
		ResourcesByState: make(map[string]int),
		TopResourcePaths: make(map[string]int),
		LeakCandidates:   make([]string, 0),
	}

	// Collect file handle information
	rd.collectFileHandles(&metrics)

	// Collect network connection information
	rd.collectNetworkConnections(&metrics)

	// Collect memory mapping information
	rd.collectMemoryMappings(&metrics)

	// Collect thread information
	rd.collectThreadInfo(&metrics)

	// Update resource tracking
	rd.updateResourceTracking(&metrics)

	rd.mutex.Lock()
	rd.metrics = append(rd.metrics, metrics)

	// Keep only recent metrics to prevent memory growth
	maxMetrics := 10000
	if len(rd.metrics) > maxMetrics {
		rd.metrics = rd.metrics[len(rd.metrics)-maxMetrics:]
	}
	rd.mutex.Unlock()

	// Check for immediate alerts
	rd.checkForAlerts(metrics)
}

// collectFileHandles collects file handle information
func (rd *ResourceDetector) collectFileHandles(metrics *ResourceMetrics) {
	if runtime.GOOS == "linux" {
		rd.collectLinuxFileHandles(metrics)
	} else if runtime.GOOS == "windows" {
		rd.collectWindowsFileHandles(metrics)
	} else {
		// Fallback: use process info
		if rd.process != nil {
			if openFiles, err := rd.process.OpenFiles(); err == nil {
				metrics.FileHandles = len(openFiles)
				metrics.ResourcesByType[ResourceTypeFile] = len(openFiles)

				// Track individual files
				for _, file := range openFiles {
					metrics.TopResourcePaths[file.Path]++
					rd.trackResource(file.Path, ResourceTypeFile, file.Path, "open")
				}
			}
		}
	}
}

// collectLinuxFileHandles collects file handles on Linux
func (rd *ResourceDetector) collectLinuxFileHandles(metrics *ResourceMetrics) {
	fdDir := fmt.Sprintf("/proc/%d/fd", rd.pid)
	files, err := ioutil.ReadDir(fdDir)
	if err != nil {
		return
	}

	metrics.FileHandles = len(files)
	metrics.ResourcesByType[ResourceTypeFile] = len(files)

	// Analyze file descriptors
	for _, file := range files {
		fdPath := filepath.Join(fdDir, file.Name())
		if target, err := os.Readlink(fdPath); err == nil {
			metrics.TopResourcePaths[target]++
			rd.trackResource(file.Name(), ResourceTypeFile, target, "open")
		}
	}
}

// collectWindowsFileHandles collects file handles on Windows
func (rd *ResourceDetector) collectWindowsFileHandles(metrics *ResourceMetrics) {
	// Windows-specific implementation would go here
	// For now, use process info as fallback
	if rd.process != nil {
		if openFiles, err := rd.process.OpenFiles(); err == nil {
			metrics.FileHandles = len(openFiles)
			metrics.ResourcesByType[ResourceTypeFile] = len(openFiles)

			for _, file := range openFiles {
				metrics.TopResourcePaths[file.Path]++
				rd.trackResource(file.Path, ResourceTypeFile, file.Path, "open")
			}
		}
	}
}

// collectNetworkConnections collects network connection information
func (rd *ResourceDetector) collectNetworkConnections(metrics *ResourceMetrics) {
	if rd.process != nil {
		if connections, err := rd.process.Connections(); err == nil {
			metrics.NetworkConnections = len(connections)
			metrics.ResourcesByType[ResourceTypeNetwork] = len(connections)

			// Analyze connections by state
			for _, conn := range connections {
				metrics.ResourcesByState[conn.Status]++
				connID := fmt.Sprintf("%s:%d->%s:%d", conn.Laddr.IP, conn.Laddr.Port, conn.Raddr.IP, conn.Raddr.Port)
				rd.trackResource(connID, ResourceTypeNetwork, connID, conn.Status)
			}
		}
	}
}

// collectMemoryMappings collects memory mapping information
func (rd *ResourceDetector) collectMemoryMappings(metrics *ResourceMetrics) {
	if rd.process != nil {
		if memMaps, err := rd.process.MemoryMaps(false); err == nil {
			var totalMapped int64
			for i := range *memMaps {
				// Use a simple size estimation based on memory map count
				totalMapped += int64(4096) // Assume 4KB page size
				mapID := fmt.Sprintf("memmap_%d", i)
				rd.trackResource(mapID, ResourceTypeMemory, mapID, "mapped")
			}
			metrics.MemoryMapped = totalMapped
			metrics.ResourcesByType[ResourceTypeMemory] = len(*memMaps)
		}
	}
}

// collectThreadInfo collects thread information
func (rd *ResourceDetector) collectThreadInfo(metrics *ResourceMetrics) {
	if rd.process != nil {
		if numThreads, err := rd.process.NumThreads(); err == nil {
			metrics.ThreadCount = int(numThreads)
			metrics.ResourcesByType[ResourceTypeThread] = int(numThreads)
		}
	}
}

// trackResource tracks a resource for leak detection
func (rd *ResourceDetector) trackResource(id string, resourceType ResourceType, path, state string) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	now := time.Now()
	if resource, exists := rd.resources[id]; exists {
		// Update existing resource
		resource.LastSeen = now
		resource.State = state
	} else {
		// Create new resource
		resource := &ResourceInfo{
			ID:        id,
			Type:      resourceType,
			Path:      path,
			State:     state,
			PID:       rd.pid,
			CreatedAt: now,
			LastSeen:  now,
		}

		// Add stack trace if enabled
		if rd.config.TrackStacks {
			resource.StackTrace = rd.captureStackTrace()
		}

		rd.resources[id] = resource
	}
}

// captureStackTrace captures the current stack trace
func (rd *ResourceDetector) captureStackTrace() []string {
	var stackTrace []string
	for i := 2; i < 10; i++ { // Skip this function and trackResource
		if pc, file, line, ok := runtime.Caller(i); ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				frame := fmt.Sprintf("%s:%d %s", filepath.Base(file), line, fn.Name())
				stackTrace = append(stackTrace, frame)
			}
		} else {
			break
		}
	}
	return stackTrace
}

// updateResourceTracking updates resource tracking and identifies candidates
func (rd *ResourceDetector) updateResourceTracking(metrics *ResourceMetrics) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	// Clean up stale resources
	now := time.Now()
	staleThreshold := rd.config.Interval * 5 // Consider stale after 5 intervals

	for id, resource := range rd.resources {
		if now.Sub(resource.LastSeen) > staleThreshold {
			delete(rd.resources, id)
		}
	}

	// Identify leak candidates
	longLivedThreshold := time.Minute * 10 // Resources alive for more than 10 minutes
	for id, resource := range rd.resources {
		if now.Sub(resource.CreatedAt) > longLivedThreshold {
			metrics.LeakCandidates = append(metrics.LeakCandidates, id)
		}
	}
}

// detectLeaks detects resource leaks based on patterns
func (rd *ResourceDetector) detectLeaks() {
	rd.mutex.RLock()
	metricsLen := len(rd.metrics)
	rd.mutex.RUnlock()

	if metricsLen < 3 {
		return // Need at least 3 data points for trend analysis
	}

	rd.mutex.RLock()
	recentMetrics := rd.metrics[max(0, metricsLen-10):] // Analyze last 10 metrics
	rd.mutex.RUnlock()

	// Detect file handle leaks
	rd.detectFileHandleLeaks(recentMetrics)

	// Detect network connection leaks
	rd.detectNetworkLeaks(recentMetrics)

	// Detect memory mapping leaks
	rd.detectMemoryLeaks(recentMetrics)

	// Detect thread leaks
	rd.detectThreadLeaks(recentMetrics)
}

// detectFileHandleLeaks detects file handle leaks
func (rd *ResourceDetector) detectFileHandleLeaks(metrics []ResourceMetrics) {
	if len(metrics) < 3 {
		return
	}

	// Calculate growth rate
	growthRate := rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.FileHandles)
	})

	if growthRate > rd.config.GrowthThreshold {
		// Analyze which paths are growing
		pathGrowth := rd.analyzePathGrowth(metrics, "file")
		for path, growth := range pathGrowth {
			if growth > rd.config.GrowthThreshold/2 {
				rd.recordLeak(ResourceTypeFile, path, growth, "file_handle_leak")
			}
		}
	}
}

// detectNetworkLeaks detects network connection leaks
func (rd *ResourceDetector) detectNetworkLeaks(metrics []ResourceMetrics) {
	if len(metrics) < 3 {
		return
	}

	// Calculate growth rate
	growthRate := rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.NetworkConnections)
	})

	if growthRate > rd.config.GrowthThreshold {
		// Analyze connection states
		stateGrowth := rd.analyzeStateGrowth(metrics)
		for state, growth := range stateGrowth {
			if growth > rd.config.GrowthThreshold/2 {
				rd.recordLeak(ResourceTypeNetwork, state, growth, "network_connection_leak")
			}
		}
	}
}

// detectMemoryLeaks detects memory mapping leaks
func (rd *ResourceDetector) detectMemoryLeaks(metrics []ResourceMetrics) {
	if len(metrics) < 3 {
		return
	}

	// Calculate growth rate
	growthRate := rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.MemoryMapped)
	})

	if growthRate > rd.config.GrowthThreshold*1024*1024 { // MB threshold
		rd.recordLeak(ResourceTypeMemory, "memory_mapped", growthRate, "memory_mapping_leak")
	}
}

// detectThreadLeaks detects thread leaks
func (rd *ResourceDetector) detectThreadLeaks(metrics []ResourceMetrics) {
	if len(metrics) < 3 {
		return
	}

	// Calculate growth rate
	growthRate := rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.ThreadCount)
	})

	if growthRate > rd.config.GrowthThreshold/10 { // Threads grow slower
		rd.recordLeak(ResourceTypeThread, "threads", growthRate, "thread_leak")
	}
}

// calculateResourceGrowthRate calculates growth rate for a resource metric
func (rd *ResourceDetector) calculateResourceGrowthRate(metrics []ResourceMetrics, extractor func(ResourceMetrics) float64) float64 {
	if len(metrics) < 2 {
		return 0
	}

	// Use linear regression to calculate growth rate
	x := make([]float64, len(metrics))
	y := make([]float64, len(metrics))

	startTime := metrics[0].Timestamp
	for i, metric := range metrics {
		x[i] = metric.Timestamp.Sub(startTime).Minutes()
		y[i] = extractor(metric)
	}

	// Calculate linear regression slope
	alpha, beta := stat.LinearRegression(x, y, nil, false)
	_ = alpha // Intercept not needed for growth rate

	return beta // Slope is growth rate per minute
}

// analyzePathGrowth analyzes growth patterns for specific paths
func (rd *ResourceDetector) analyzePathGrowth(metrics []ResourceMetrics, resourceType string) map[string]float64 {
	pathGrowth := make(map[string]float64)

	if len(metrics) < 2 {
		return pathGrowth
	}

	// Track path counts over time
	pathTimeSeries := make(map[string][]float64)
	timePoints := make([]float64, len(metrics))

	startTime := metrics[0].Timestamp
	for i, metric := range metrics {
		timePoints[i] = metric.Timestamp.Sub(startTime).Minutes()
		for path, count := range metric.TopResourcePaths {
			if pathTimeSeries[path] == nil {
				pathTimeSeries[path] = make([]float64, len(metrics))
			}
			pathTimeSeries[path][i] = float64(count)
		}
	}

	// Calculate growth rate for each path
	for path, counts := range pathTimeSeries {
		if len(counts) >= 2 {
			_, beta := stat.LinearRegression(timePoints, counts, nil, false)
			pathGrowth[path] = beta
		}
	}

	return pathGrowth
}

// analyzeStateGrowth analyzes growth patterns for connection states
func (rd *ResourceDetector) analyzeStateGrowth(metrics []ResourceMetrics) map[string]float64 {
	stateGrowth := make(map[string]float64)

	if len(metrics) < 2 {
		return stateGrowth
	}

	// Track state counts over time
	stateTimeSeries := make(map[string][]float64)
	timePoints := make([]float64, len(metrics))

	startTime := metrics[0].Timestamp
	for i, metric := range metrics {
		timePoints[i] = metric.Timestamp.Sub(startTime).Minutes()
		for state, count := range metric.ResourcesByState {
			if stateTimeSeries[state] == nil {
				stateTimeSeries[state] = make([]float64, len(metrics))
			}
			stateTimeSeries[state][i] = float64(count)
		}
	}

	// Calculate growth rate for each state
	for state, counts := range stateTimeSeries {
		if len(counts) >= 2 {
			_, beta := stat.LinearRegression(timePoints, counts, nil, false)
			stateGrowth[state] = beta
		}
	}

	return stateGrowth
}

// recordLeak records a detected resource leak
func (rd *ResourceDetector) recordLeak(resourceType ResourceType, path string, growthRate float64, leakType string) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	// Check if this leak already exists
	leakID := fmt.Sprintf("%s_%s", resourceType, path)
	for i, existingLeak := range rd.leaks {
		if existingLeak.ID == leakID {
			// Update existing leak
			rd.leaks[i].LatestDetected = time.Now()
			rd.leaks[i].GrowthRate = growthRate
			rd.leaks[i].Count++
			return
		}
	}

	// Create new leak
	leak := ResourceLeak{
		ID:             leakID,
		Type:           resourceType,
		Path:           path,
		Count:          1,
		GrowthRate:     growthRate,
		FirstDetected:  time.Now(),
		LatestDetected: time.Now(),
	}

	// Calculate confidence and severity
	leak.Confidence = rd.calculateLeakConfidence(growthRate, resourceType)
	leak.Severity = rd.determineLeakSeverity(growthRate, resourceType)
	leak.Description = rd.generateLeakDescription(leak)
	leak.Suggestion = rd.generateLeakSuggestion(leak)

	// Find affected resources
	leak.AffectedResources = rd.findAffectedResources(resourceType, path)

	// Add stack trace if available
	if rd.config.TrackStacks {
		leak.StackTrace = rd.captureStackTrace()
	}

	rd.leaks = append(rd.leaks, leak)

	if rd.config.Verbose {
		fmt.Printf("Detected %s leak: %s (growth rate: %.2f/min)\n",
			resourceType, path, growthRate)
	}
}

// calculateLeakConfidence calculates confidence score for leak detection
func (rd *ResourceDetector) calculateLeakConfidence(growthRate float64, resourceType ResourceType) float64 {
	confidence := 0.0

	// Base confidence on growth rate relative to threshold
	threshold := rd.config.GrowthThreshold
	if resourceType == ResourceTypeMemory {
		threshold *= 1024 * 1024 // MB for memory
	} else if resourceType == ResourceTypeThread {
		threshold /= 10 // Threads grow slower
	}

	if threshold > 0 {
		confidence = math.Min(growthRate/threshold, 2.0) * 0.5
	}

	// Add confidence based on resource type
	switch resourceType {
	case ResourceTypeFile:
		confidence += 0.3 // File handles are easier to detect
	case ResourceTypeNetwork:
		confidence += 0.25 // Network connections are fairly reliable
	case ResourceTypeMemory:
		confidence += 0.2 // Memory can be more complex
	case ResourceTypeThread:
		confidence += 0.15 // Threads can vary more
	}

	return math.Min(confidence, 1.0)
}

// determineLeakSeverity determines the severity of a leak
func (rd *ResourceDetector) determineLeakSeverity(growthRate float64, resourceType ResourceType) string {
	threshold := rd.config.GrowthThreshold

	switch resourceType {
	case ResourceTypeFile:
		if growthRate > threshold*5 {
			return "critical"
		} else if growthRate > threshold*2 {
			return "high"
		} else if growthRate > threshold {
			return "medium"
		}
	case ResourceTypeNetwork:
		if growthRate > threshold*3 {
			return "critical"
		} else if growthRate > threshold*1.5 {
			return "high"
		} else if growthRate > threshold {
			return "medium"
		}
	case ResourceTypeMemory:
		mbThreshold := threshold * 1024 * 1024
		if growthRate > mbThreshold*10 {
			return "critical"
		} else if growthRate > mbThreshold*5 {
			return "high"
		} else if growthRate > mbThreshold {
			return "medium"
		}
	case ResourceTypeThread:
		threadThreshold := threshold / 10
		if growthRate > threadThreshold*5 {
			return "critical"
		} else if growthRate > threadThreshold*2 {
			return "high"
		} else if growthRate > threadThreshold {
			return "medium"
		}
	}
	return "low"
}

// generateLeakDescription generates a description for a leak
func (rd *ResourceDetector) generateLeakDescription(leak ResourceLeak) string {
	return fmt.Sprintf("%s %s leak detected with growth rate %.2f/min (confidence: %.1f%%)",
		strings.Title(leak.Severity), leak.Type, leak.GrowthRate, leak.Confidence*100)
}

// generateLeakSuggestion generates a suggestion for fixing a leak
func (rd *ResourceDetector) generateLeakSuggestion(leak ResourceLeak) string {
	switch leak.Type {
	case ResourceTypeFile:
		return "Ensure files are properly closed using defer statements or try-with-resources patterns"
	case ResourceTypeNetwork:
		return "Check for unclosed network connections and implement proper connection pooling"
	case ResourceTypeMemory:
		return "Review memory mapping usage and ensure proper cleanup of mapped regions"
	case ResourceTypeThread:
		return "Monitor goroutine lifecycle and implement proper goroutine termination"
	default:
		return "Review resource lifecycle and implement proper cleanup patterns"
	}
}

// findAffectedResources finds resources affected by a leak
func (rd *ResourceDetector) findAffectedResources(resourceType ResourceType, path string) []string {
	var affected []string

	for id, resource := range rd.resources {
		if resource.Type == resourceType && (path == "" || strings.Contains(resource.Path, path)) {
			affected = append(affected, id)
		}
	}

	return affected
}

// generateRecommendations generates recommendations based on analysis
func (rd *ResourceDetector) generateRecommendations(analysis *ResourceAnalysis) []string {
	var recommendations []string

	if len(analysis.DetectedLeaks) > 5 {
		recommendations = append(recommendations, "Multiple resource leaks detected. Implement comprehensive resource management patterns.")
	}

	criticalLeaks := 0
	for _, leak := range analysis.DetectedLeaks {
		if leak.Severity == "critical" {
			criticalLeaks++
		}
	}

	if criticalLeaks > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Found %d critical resource leaks. Immediate attention required.", criticalLeaks))
	}

	if analysis.Statistics.PeakFileHandles > rd.config.FileHandleThreshold {
		recommendations = append(recommendations, "File handle usage exceeds threshold. Review file management practices.")
	}

	if analysis.Statistics.PeakNetworkConns > rd.config.NetworkThreshold {
		recommendations = append(recommendations, "Network connection usage exceeds threshold. Implement connection pooling.")
	}

	if analysis.Trends.OverallTrend == "increasing" {
		recommendations = append(recommendations, "Overall resource usage is increasing. Monitor for potential leaks.")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Resource usage appears normal. Continue monitoring for any changes.")
	}

	return recommendations
}

// calculateStatistics calculates comprehensive resource statistics
func (rd *ResourceDetector) calculateStatistics(metrics []ResourceMetrics, leaks []ResourceLeak) ResourceStats {
	stats := ResourceStats{
		ResourceGrowthRates: make(map[ResourceType]float64),
		LeaksByType:         make(map[ResourceType]int),
		LeaksBySeverity:     make(map[string]int),
		MostLeakyPaths:      make(map[string]int),
	}

	if len(metrics) == 0 {
		return stats
	}

	// Calculate mean values and peaks
	fileHandles := make([]float64, len(metrics))
	networkConns := make([]float64, len(metrics))

	for i, metric := range metrics {
		fileHandles[i] = float64(metric.FileHandles)
		networkConns[i] = float64(metric.NetworkConnections)

		if metric.FileHandles > stats.PeakFileHandles {
			stats.PeakFileHandles = metric.FileHandles
		}
		if metric.NetworkConnections > stats.PeakNetworkConns {
			stats.PeakNetworkConns = metric.NetworkConnections
		}
	}

	stats.MeanFileHandles = stat.Mean(fileHandles, nil)
	stats.MeanNetworkConns = stat.Mean(networkConns, nil)

	// Calculate growth rates
	stats.ResourceGrowthRates[ResourceTypeFile] = rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.FileHandles)
	})
	stats.ResourceGrowthRates[ResourceTypeNetwork] = rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.NetworkConnections)
	})
	stats.ResourceGrowthRates[ResourceTypeMemory] = rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.MemoryMapped)
	})
	stats.ResourceGrowthRates[ResourceTypeThread] = rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.ThreadCount)
	})

	// Analyze leaks
	for _, leak := range leaks {
		stats.LeaksByType[leak.Type]++
		stats.LeaksBySeverity[leak.Severity]++
		stats.MostLeakyPaths[leak.Path]++
	}

	// Find peak usage period
	stats.PeakUsagePeriod = rd.findPeakUsagePeriod(metrics)

	return stats
}

// identifyHotspots identifies resource allocation hotspots
func (rd *ResourceDetector) identifyHotspots(leaks []ResourceLeak) []ResourceHotspot {
	locationStats := make(map[string]*ResourceHotspot)

	for _, leak := range leaks {
		// Use stack trace to identify location
		location := "unknown"
		if len(leak.StackTrace) > 0 {
			location = leak.StackTrace[0] // Use top of stack
		}

		if hotspot, exists := locationStats[location]; exists {
			hotspot.Count++
			hotspot.LeakRate = float64(hotspot.Count) / float64(len(leaks))
		} else {
			locationStats[location] = &ResourceHotspot{
				Location:    location,
				Type:        leak.Type,
				Count:       1,
				LeakRate:    1.0 / float64(len(leaks)),
				Description: fmt.Sprintf("Resource allocation hotspot for %s", leak.Type),
				Suggestion:  rd.generateHotspotSuggestion(leak.Type),
			}
		}
	}

	// Convert to slice and classify severity
	var hotspots []ResourceHotspot
	for _, hotspot := range locationStats {
		hotspot.Severity = rd.classifyHotspotSeverity(hotspot.Count, hotspot.LeakRate)
		hotspots = append(hotspots, *hotspot)
	}

	// Sort by count (highest first)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Count > hotspots[j].Count
	})

	return hotspots
}

// generateHotspotSuggestion generates suggestions for hotspots
func (rd *ResourceDetector) generateHotspotSuggestion(resourceType ResourceType) string {
	switch resourceType {
	case ResourceTypeFile:
		return "Review file opening patterns and implement resource pooling"
	case ResourceTypeNetwork:
		return "Implement connection pooling and proper connection lifecycle management"
	case ResourceTypeMemory:
		return "Review memory allocation patterns and implement proper cleanup"
	case ResourceTypeThread:
		return "Review goroutine creation patterns and implement worker pools"
	default:
		return "Review resource allocation patterns and implement proper management"
	}
}

// classifyHotspotSeverity classifies hotspot severity
func (rd *ResourceDetector) classifyHotspotSeverity(count int, leakRate float64) string {
	if count > 10 || leakRate > 0.5 {
		return "critical"
	} else if count > 5 || leakRate > 0.3 {
		return "high"
	} else if count > 2 || leakRate > 0.1 {
		return "medium"
	}
	return "low"
}

// analyzeTrends analyzes resource usage trends
func (rd *ResourceDetector) analyzeTrends(metrics []ResourceMetrics) ResourceTrends {
	trends := ResourceTrends{}

	if len(metrics) < 3 {
		return trends
	}

	// Analyze file handle trend
	fileGrowthRate := rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.FileHandles)
	})
	trends.FileHandleTrend = rd.classifyTrend(fileGrowthRate)

	// Analyze network connection trend
	networkGrowthRate := rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.NetworkConnections)
	})
	trends.NetworkConnTrend = rd.classifyTrend(networkGrowthRate)

	// Analyze memory mapped trend
	memoryGrowthRate := rd.calculateResourceGrowthRate(metrics, func(m ResourceMetrics) float64 {
		return float64(m.MemoryMapped)
	})
	trends.MemoryMappedTrend = rd.classifyTrend(memoryGrowthRate)

	// Determine overall trend
	overallGrowthRate := (fileGrowthRate + networkGrowthRate + memoryGrowthRate) / 3
	trends.OverallTrend = rd.classifyTrend(overallGrowthRate)

	// Calculate growth acceleration
	trends.GrowthAcceleration = rd.calculateGrowthAcceleration(metrics)

	return trends
}

// classifyTrend classifies a trend based on growth rate
func (rd *ResourceDetector) classifyTrend(growthRate float64) string {
	if growthRate > rd.config.GrowthThreshold {
		return "increasing"
	} else if growthRate < -rd.config.GrowthThreshold {
		return "decreasing"
	}
	return "stable"
}

// calculateGrowthAcceleration calculates growth acceleration
func (rd *ResourceDetector) calculateGrowthAcceleration(metrics []ResourceMetrics) float64 {
	if len(metrics) < 4 {
		return 0
	}

	// Calculate growth rates for two periods
	midPoint := len(metrics) / 2
	firstHalf := metrics[:midPoint]
	secondHalf := metrics[midPoint:]

	firstGrowthRate := rd.calculateResourceGrowthRate(firstHalf, func(m ResourceMetrics) float64 {
		return float64(m.FileHandles + m.NetworkConnections)
	})

	secondGrowthRate := rd.calculateResourceGrowthRate(secondHalf, func(m ResourceMetrics) float64 {
		return float64(m.FileHandles + m.NetworkConnections)
	})

	return secondGrowthRate - firstGrowthRate
}

// findPeakUsagePeriod finds the period with highest resource usage
func (rd *ResourceDetector) findPeakUsagePeriod(metrics []ResourceMetrics) string {
	if len(metrics) == 0 {
		return "no_data"
	}

	maxUsage := 0
	maxUsageTime := metrics[0].Timestamp

	for _, metric := range metrics {
		totalUsage := metric.FileHandles + metric.NetworkConnections + metric.ThreadCount
		if totalUsage > maxUsage {
			maxUsage = totalUsage
			maxUsageTime = metric.Timestamp
		}
	}

	return maxUsageTime.Format("2006-01-02 15:04:05")
}

// checkForAlerts checks for immediate alerts
func (rd *ResourceDetector) checkForAlerts(current ResourceMetrics) {
	if current.FileHandles > rd.config.FileHandleThreshold && rd.config.Verbose {
		fmt.Printf("ALERT: File handle count exceeds threshold: %d > %d\n",
			current.FileHandles, rd.config.FileHandleThreshold)
	}

	if current.NetworkConnections > rd.config.NetworkThreshold && rd.config.Verbose {
		fmt.Printf("ALERT: Network connection count exceeds threshold: %d > %d\n",
			current.NetworkConnections, rd.config.NetworkThreshold)
	}

	if len(current.LeakCandidates) > 10 && rd.config.Verbose {
		fmt.Printf("ALERT: High number of leak candidates detected: %d\n", len(current.LeakCandidates))
	}
}

// GetResourceCount returns the number of tracked resources
func (rd *ResourceDetector) GetResourceCount() int {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()
	return len(rd.resources)
}

// GetLeakCount returns the number of detected leaks
func (rd *ResourceDetector) GetLeakCount() int {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()
	return len(rd.leaks)
}

// CleanupStaleResources removes stale resource references
func (rd *ResourceDetector) CleanupStaleResources(maxAge time.Duration) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	now := time.Now()
	for id, resource := range rd.resources {
		if now.Sub(resource.LastSeen) > maxAge {
			delete(rd.resources, id)
		}
	}
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}