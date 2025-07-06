package detector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// ReferenceConfig holds configuration for reference cycle detection
type ReferenceConfig struct {
	Interval       time.Duration
	MaxDepth       int  // Maximum depth for cycle detection
	Verbose        bool
	TrackPointers  bool // Track pointer relationships
	SampleRate     float64 // Sampling rate for performance (0.0-1.0)
}

// ObjectInfo represents information about a tracked object
type ObjectInfo struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Size        uintptr     `json:"size"`
	Address     uintptr     `json:"address"`
	References  []string    `json:"references"`
	ReferencedBy []string   `json:"referenced_by"`
	CreatedAt   time.Time   `json:"created_at"`
	LastSeen    time.Time   `json:"last_seen"`
	Value       interface{} `json:"-"` // Don't serialize the actual value
}

// ReferenceCycle represents a detected reference cycle
type ReferenceCycle struct {
	ID          string      `json:"id"`
	Objects     []string    `json:"objects"`
	Path        []string    `json:"path"`
	Length      int         `json:"length"`
	TotalSize   uintptr     `json:"total_size"`
	Severity    string      `json:"severity"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	FirstSeen   time.Time   `json:"first_seen"`
	LastSeen    time.Time   `json:"last_seen"`
	Count       int         `json:"count"`
	Suggestion  string      `json:"suggestion"`
}

// ReferenceMetrics represents reference tracking metrics
type ReferenceMetrics struct {
	Timestamp       time.Time `json:"timestamp"`
	TrackedObjects  int       `json:"tracked_objects"`
	TotalReferences int       `json:"total_references"`
	CyclesDetected  int       `json:"cycles_detected"`
	CyclesByType    map[string]int `json:"cycles_by_type"`
	CyclesBySeverity map[string]int `json:"cycles_by_severity"`
	LargestCycle    int       `json:"largest_cycle"`
	TotalCycleSize  uintptr   `json:"total_cycle_size"`
}

// ReferenceAnalysis represents the result of reference cycle analysis
type ReferenceAnalysis struct {
	StartTime       time.Time          `json:"start_time"`
	EndTime         time.Time          `json:"end_time"`
	Duration        time.Duration      `json:"duration"`
	Metrics         []ReferenceMetrics `json:"metrics"`
	DetectedCycles  []ReferenceCycle   `json:"detected_cycles"`
	SuspiciousTypes []SuspiciousType   `json:"suspicious_types"`
	Recommendations []string           `json:"recommendations"`
	Statistics      ReferenceStats     `json:"statistics"`
	Hotspots        []ReferenceHotspot `json:"hotspots"`
}

// SuspiciousType represents a type that frequently appears in cycles
type SuspiciousType struct {
	TypeName     string    `json:"type_name"`
	CycleCount   int       `json:"cycle_count"`
	TotalSize    uintptr   `json:"total_size"`
	Severity     string    `json:"severity"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	Suggestion   string    `json:"suggestion"`
}

// ReferenceStats contains statistical analysis of reference cycles
type ReferenceStats struct {
	TotalCycles        int                `json:"total_cycles"`
	AverageCycleLength float64            `json:"average_cycle_length"`
	AverageCycleSize   float64            `json:"average_cycle_size"`
	CycleLengthDist    map[int]int        `json:"cycle_length_distribution"`
	TypeFrequency      map[string]int     `json:"type_frequency"`
	SeverityDist       map[string]int     `json:"severity_distribution"`
	GrowthRate         float64            `json:"growth_rate"`
	PeakDetectionTime  string             `json:"peak_detection_time"`
}

// ReferenceHotspot represents a reference creation hotspot
type ReferenceHotspot struct {
	Location    string  `json:"location"`
	TypeName    string  `json:"type_name"`
	Count       int     `json:"count"`
	CycleRate   float64 `json:"cycle_rate"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
}

// ReferenceDetector monitors and detects reference cycles
type ReferenceDetector struct {
	config      *ReferenceConfig
	objects     map[string]*ObjectInfo
	cycles      []ReferenceCycle
	metrics     []ReferenceMetrics
	running     bool
	stopChan    chan struct{}
	mutex       sync.RWMutex
	cycleIDCounter int
}

// NewReferenceDetector creates a new reference detector
func NewReferenceDetector(config *ReferenceConfig) *ReferenceDetector {
	return &ReferenceDetector{
		config:   config,
		objects:  make(map[string]*ObjectInfo),
		cycles:   make([]ReferenceCycle, 0),
		metrics:  make([]ReferenceMetrics, 0),
		stopChan: make(chan struct{}),
	}
}

// Start begins reference cycle monitoring
func (rd *ReferenceDetector) Start() error {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	if rd.running {
		return fmt.Errorf("reference detector already running")
	}

	rd.running = true
	go rd.monitorLoop()

	if rd.config.Verbose {
		fmt.Printf("Reference detector started with interval %v\n", rd.config.Interval)
	}

	return nil
}

// Stop stops reference cycle monitoring
func (rd *ReferenceDetector) Stop() {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	if !rd.running {
		return
	}

	rd.running = false
	close(rd.stopChan)

	if rd.config.Verbose {
		fmt.Println("Reference detector stopped")
	}
}

// TrackObject manually tracks an object for cycle detection
func (rd *ReferenceDetector) TrackObject(obj interface{}, id string) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	if obj == nil {
		return
	}

	v := reflect.ValueOf(obj)
	t := reflect.TypeOf(obj)

	if !v.IsValid() {
		return
	}

	// Create object info
	objInfo := &ObjectInfo{
		ID:          id,
		Type:        t.String(),
		Size:        t.Size(),
		Address:     v.Pointer(),
		References:  make([]string, 0),
		ReferencedBy: make([]string, 0),
		CreatedAt:   time.Now(),
		LastSeen:    time.Now(),
		Value:       obj,
	}

	// Extract references
	rd.extractReferences(v, objInfo)

	rd.objects[id] = objInfo
}

// GetMetrics returns current metrics
func (rd *ReferenceDetector) GetMetrics() []ReferenceMetrics {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()

	// Return a copy to avoid race conditions
	metrics := make([]ReferenceMetrics, len(rd.metrics))
	copy(metrics, rd.metrics)
	return metrics
}

// GetCycles returns detected cycles
func (rd *ReferenceDetector) GetCycles() []ReferenceCycle {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()

	// Return a copy to avoid race conditions
	cycles := make([]ReferenceCycle, len(rd.cycles))
	copy(cycles, rd.cycles)
	return cycles
}

// AnalyzeReferences performs comprehensive reference cycle analysis
func (rd *ReferenceDetector) AnalyzeReferences() (*ReferenceAnalysis, error) {
	rd.mutex.RLock()
	metrics := make([]ReferenceMetrics, len(rd.metrics))
	copy(metrics, rd.metrics)
	cycles := make([]ReferenceCycle, len(rd.cycles))
	copy(cycles, rd.cycles)
	rd.mutex.RUnlock()

	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics available for analysis")
	}

	analysis := &ReferenceAnalysis{
		StartTime:      metrics[0].Timestamp,
		EndTime:        metrics[len(metrics)-1].Timestamp,
		Metrics:        metrics,
		DetectedCycles: cycles,
	}
	analysis.Duration = analysis.EndTime.Sub(analysis.StartTime)

	// Identify suspicious types
	analysis.SuspiciousTypes = rd.identifySuspiciousTypes(cycles)

	// Generate recommendations
	analysis.Recommendations = rd.generateRecommendations(analysis)

	// Calculate statistics
	analysis.Statistics = rd.calculateStatistics(metrics, cycles)

	// Identify hotspots
	analysis.Hotspots = rd.identifyHotspots(cycles)

	return analysis, nil
}

// SaveResults saves analysis results to files
func (rd *ReferenceDetector) SaveResults(outputDir string) error {
	analysis, err := rd.AnalyzeReferences()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Save detailed analysis
	analysisFile := filepath.Join(outputDir, "reference_analysis.json")
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

	// Save detected cycles
	cyclesFile := filepath.Join(outputDir, "reference_cycles.json")
	file, err = os.Create(cyclesFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder = json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rd.GetCycles())
}

// monitorLoop runs the monitoring loop
func (rd *ReferenceDetector) monitorLoop() {
	ticker := time.NewTicker(rd.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rd.detectCycles()
			rd.collectMetrics()
		case <-rd.stopChan:
			return
		}
	}
}

// detectCycles detects reference cycles in tracked objects
func (rd *ReferenceDetector) detectCycles() {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	// Build reference graph
	graph := rd.buildReferenceGraph()

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	for objID := range graph {
		if !visited[objID] {
			rd.dfsDetectCycles(objID, graph, visited, recStack, path)
		}
	}
}

// buildReferenceGraph builds a reference graph from tracked objects
func (rd *ReferenceDetector) buildReferenceGraph() map[string][]string {
	graph := make(map[string][]string)

	for objID, objInfo := range rd.objects {
		graph[objID] = objInfo.References
	}

	return graph
}

// dfsDetectCycles performs DFS to detect cycles
func (rd *ReferenceDetector) dfsDetectCycles(objID string, graph map[string][]string, visited, recStack map[string]bool, path []string) {
	visited[objID] = true
	recStack[objID] = true
	path = append(path, objID)

	for _, refID := range graph[objID] {
		if !visited[refID] {
			rd.dfsDetectCycles(refID, graph, visited, recStack, path)
		} else if recStack[refID] {
			// Found a cycle
			cycleStart := -1
			for i, id := range path {
				if id == refID {
					cycleStart = i
					break
				}
			}

			if cycleStart >= 0 {
				cyclePath := append(path[cycleStart:], refID)
				rd.recordCycle(cyclePath)
			}
		}
	}

	recStack[objID] = false
	// Remove from path when backtracking
	if len(path) > 0 && path[len(path)-1] == objID {
		path = path[:len(path)-1]
	}
}

// recordCycle records a detected cycle
func (rd *ReferenceDetector) recordCycle(cyclePath []string) {
	if len(cyclePath) < 2 {
		return
	}

	// Check if this cycle already exists
	cycleID := rd.generateCycleID(cyclePath)
	for i, existingCycle := range rd.cycles {
		if existingCycle.ID == cycleID {
			// Update existing cycle
			rd.cycles[i].LastSeen = time.Now()
			rd.cycles[i].Count++
			return
		}
	}

	// Create new cycle
	cycle := ReferenceCycle{
		ID:        cycleID,
		Objects:   cyclePath[:len(cyclePath)-1], // Remove duplicate last element
		Path:      cyclePath,
		Length:    len(cyclePath) - 1,
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		Count:     1,
	}

	// Calculate total size and determine type
	cycle.TotalSize = rd.calculateCycleSize(cycle.Objects)
	cycle.Type = rd.determineCycleType(cycle.Objects)
	cycle.Severity = rd.determineCycleSeverity(cycle.Length, cycle.TotalSize)
	cycle.Description = rd.generateCycleDescription(cycle)
	cycle.Suggestion = rd.generateCycleSuggestion(cycle)

	rd.cycles = append(rd.cycles, cycle)

	if rd.config.Verbose {
		fmt.Printf("Detected reference cycle: %s (length: %d, size: %d bytes)\n",
			cycle.ID, cycle.Length, cycle.TotalSize)
	}
}

// generateCycleID generates a unique ID for a cycle
func (rd *ReferenceDetector) generateCycleID(cyclePath []string) string {
	// Sort the path to ensure consistent IDs for the same cycle
	sortedPath := make([]string, len(cyclePath)-1) // Remove duplicate last element
	copy(sortedPath, cyclePath[:len(cyclePath)-1])
	sort.Strings(sortedPath)
	return fmt.Sprintf("cycle_%s", strings.Join(sortedPath, "_"))
}

// calculateCycleSize calculates the total size of objects in a cycle
func (rd *ReferenceDetector) calculateCycleSize(objects []string) uintptr {
	var totalSize uintptr
	for _, objID := range objects {
		if objInfo, exists := rd.objects[objID]; exists {
			totalSize += objInfo.Size
		}
	}
	return totalSize
}

// determineCycleType determines the type of cycle based on object types
func (rd *ReferenceDetector) determineCycleType(objects []string) string {
	typeCount := make(map[string]int)
	for _, objID := range objects {
		if objInfo, exists := rd.objects[objID]; exists {
			typeCount[objInfo.Type]++
		}
	}

	// Find the most common type
	maxCount := 0
	mostCommonType := "mixed"
	for typeName, count := range typeCount {
		if count > maxCount {
			maxCount = count
			mostCommonType = typeName
		}
	}

	if len(typeCount) == 1 {
		return "homogeneous_" + mostCommonType
	} else if maxCount > len(objects)/2 {
		return "dominant_" + mostCommonType
	}
	return "mixed"
}

// determineCycleSeverity determines the severity of a cycle
func (rd *ReferenceDetector) determineCycleSeverity(length int, size uintptr) string {
	if length > 10 || size > 1024*1024 { // > 1MB
		return "critical"
	} else if length > 5 || size > 512*1024 { // > 512KB
		return "high"
	} else if length > 2 || size > 64*1024 { // > 64KB
		return "medium"
	}
	return "low"
}

// generateCycleDescription generates a description for a cycle
func (rd *ReferenceDetector) generateCycleDescription(cycle ReferenceCycle) string {
	return fmt.Sprintf("%s reference cycle with %d objects (total size: %d bytes)",
		strings.Title(cycle.Severity), cycle.Length, cycle.TotalSize)
}

// generateCycleSuggestion generates a suggestion for fixing a cycle
func (rd *ReferenceDetector) generateCycleSuggestion(cycle ReferenceCycle) string {
	switch {
	case strings.Contains(cycle.Type, "slice") || strings.Contains(cycle.Type, "array"):
		return "Consider using weak references or breaking the cycle by setting slice elements to nil"
	case strings.Contains(cycle.Type, "map"):
		return "Review map key-value relationships and consider using sync.Map or breaking references"
	case strings.Contains(cycle.Type, "chan"):
		return "Check channel usage patterns and ensure proper channel closure"
	case strings.Contains(cycle.Type, "struct"):
		return "Review struct field relationships and consider using pointers or interfaces to break cycles"
	case cycle.Length > 5:
		return "Complex cycle detected. Consider redesigning object relationships or using weak references"
	default:
		return "Review object lifecycle and consider breaking references when objects are no longer needed"
	}
}

// collectMetrics collects current reference metrics
func (rd *ReferenceDetector) collectMetrics() {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	metrics := ReferenceMetrics{
		Timestamp:        time.Now(),
		TrackedObjects:   len(rd.objects),
		CyclesDetected:   len(rd.cycles),
		CyclesByType:     make(map[string]int),
		CyclesBySeverity: make(map[string]int),
	}

	// Count total references
	for _, objInfo := range rd.objects {
		metrics.TotalReferences += len(objInfo.References)
	}

	// Analyze cycles
	var totalCycleSize uintptr
	for _, cycle := range rd.cycles {
		metrics.CyclesByType[cycle.Type]++
		metrics.CyclesBySeverity[cycle.Severity]++
		totalCycleSize += cycle.TotalSize

		if cycle.Length > metrics.LargestCycle {
			metrics.LargestCycle = cycle.Length
		}
	}
	metrics.TotalCycleSize = totalCycleSize

	rd.metrics = append(rd.metrics, metrics)

	// Keep only recent metrics to prevent memory growth
	maxMetrics := 10000
	if len(rd.metrics) > maxMetrics {
		rd.metrics = rd.metrics[len(rd.metrics)-maxMetrics:]
	}
}

// extractReferences extracts references from an object using reflection
func (rd *ReferenceDetector) extractReferences(v reflect.Value, objInfo *ObjectInfo) {
	if !v.IsValid() || rd.config.MaxDepth <= 0 {
		return
	}

	switch v.Kind() {
	case reflect.Ptr:
		if !v.IsNil() {
			rd.extractReferences(v.Elem(), objInfo)
		}
	case reflect.Interface:
		if !v.IsNil() {
			rd.extractReferences(v.Elem(), objInfo)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.CanInterface() {
				rd.extractFieldReferences(field, objInfo)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			rd.extractFieldReferences(elem, objInfo)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			rd.extractFieldReferences(key, objInfo)
			rd.extractFieldReferences(value, objInfo)
		}
	case reflect.Chan:
		// Channels are harder to introspect, but we can track the channel itself
		if rd.config.TrackPointers && v.Pointer() != 0 {
			refID := fmt.Sprintf("chan_%x", v.Pointer())
			objInfo.References = append(objInfo.References, refID)
		}
	}
}

// extractFieldReferences extracts references from a field
func (rd *ReferenceDetector) extractFieldReferences(field reflect.Value, objInfo *ObjectInfo) {
	if !field.IsValid() {
		return
	}

	switch field.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !field.IsNil() && rd.config.TrackPointers {
			refID := fmt.Sprintf("%s_%x", field.Type().String(), field.Pointer())
			objInfo.References = append(objInfo.References, refID)
		}
	case reflect.Slice, reflect.Map, reflect.Chan:
		if rd.config.TrackPointers && field.Pointer() != 0 {
			refID := fmt.Sprintf("%s_%x", field.Type().String(), field.Pointer())
			objInfo.References = append(objInfo.References, refID)
		}
	}
}

// identifySuspiciousTypes identifies types that frequently appear in cycles
func (rd *ReferenceDetector) identifySuspiciousTypes(cycles []ReferenceCycle) []SuspiciousType {
	typeStats := make(map[string]*SuspiciousType)

	for _, cycle := range cycles {
		for _, objID := range cycle.Objects {
			if objInfo, exists := rd.objects[objID]; exists {
				typeName := objInfo.Type
				if stats, exists := typeStats[typeName]; exists {
					stats.CycleCount++
					stats.TotalSize += objInfo.Size
					if cycle.LastSeen.After(stats.LastSeen) {
						stats.LastSeen = cycle.LastSeen
					}
				} else {
					typeStats[typeName] = &SuspiciousType{
						TypeName:   typeName,
						CycleCount: 1,
						TotalSize:  objInfo.Size,
						FirstSeen:  cycle.FirstSeen,
						LastSeen:   cycle.LastSeen,
					}
				}
			}
		}
	}

	// Convert to slice and classify severity
	var suspiciousTypes []SuspiciousType
	for _, stats := range typeStats {
		stats.Severity = rd.classifyTypeSeverity(stats.CycleCount, stats.TotalSize)
		stats.Suggestion = rd.generateTypeSuggestion(stats.TypeName, stats.CycleCount)
		suspiciousTypes = append(suspiciousTypes, *stats)
	}

	// Sort by cycle count (highest first)
	sort.Slice(suspiciousTypes, func(i, j int) bool {
		return suspiciousTypes[i].CycleCount > suspiciousTypes[j].CycleCount
	})

	return suspiciousTypes
}

// classifyTypeSeverity classifies type severity based on cycle frequency
func (rd *ReferenceDetector) classifyTypeSeverity(cycleCount int, totalSize uintptr) string {
	if cycleCount > 10 || totalSize > 1024*1024 {
		return "critical"
	} else if cycleCount > 5 || totalSize > 512*1024 {
		return "high"
	} else if cycleCount > 2 || totalSize > 64*1024 {
		return "medium"
	}
	return "low"
}

// generateTypeSuggestion generates suggestions for suspicious types
func (rd *ReferenceDetector) generateTypeSuggestion(typeName string, cycleCount int) string {
	switch {
	case strings.Contains(typeName, "slice"):
		return "Consider using slice capacity management and clearing references when done"
	case strings.Contains(typeName, "map"):
		return "Review map lifecycle and consider using sync.Map or manual cleanup"
	case strings.Contains(typeName, "chan"):
		return "Ensure proper channel closure and avoid circular channel dependencies"
	case strings.Contains(typeName, "struct"):
		return "Review struct design and consider using composition over inheritance"
	case cycleCount > 5:
		return "High cycle frequency detected. Consider redesigning object relationships"
	default:
		return "Review object lifecycle and implement proper cleanup patterns"
	}
}

// generateRecommendations generates recommendations based on analysis
func (rd *ReferenceDetector) generateRecommendations(analysis *ReferenceAnalysis) []string {
	var recommendations []string

	if len(analysis.DetectedCycles) > 10 {
		recommendations = append(recommendations, "High number of reference cycles detected. Consider implementing weak reference patterns.")
	}

	criticalCycles := 0
	for _, cycle := range analysis.DetectedCycles {
		if cycle.Severity == "critical" {
			criticalCycles++
		}
	}

	if criticalCycles > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Found %d critical reference cycles. Immediate attention required.", criticalCycles))
	}

	if len(analysis.SuspiciousTypes) > 5 {
		recommendations = append(recommendations, "Multiple types frequently involved in cycles. Review object design patterns.")
	}

	if analysis.Statistics.AverageCycleLength > 5 {
		recommendations = append(recommendations, "Complex reference cycles detected. Consider simplifying object relationships.")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Reference cycle analysis complete. No immediate issues detected.")
	}

	return recommendations
}

// calculateStatistics calculates comprehensive reference statistics
func (rd *ReferenceDetector) calculateStatistics(metrics []ReferenceMetrics, cycles []ReferenceCycle) ReferenceStats {
	stats := ReferenceStats{
		TotalCycles:     len(cycles),
		CycleLengthDist: make(map[int]int),
		TypeFrequency:   make(map[string]int),
		SeverityDist:    make(map[string]int),
	}

	if len(cycles) == 0 {
		return stats
	}

	// Calculate average cycle length and size
	totalLength := 0
	var totalSize uintptr
	for _, cycle := range cycles {
		totalLength += cycle.Length
		totalSize += cycle.TotalSize
		stats.CycleLengthDist[cycle.Length]++
		stats.TypeFrequency[cycle.Type]++
		stats.SeverityDist[cycle.Severity]++
	}

	stats.AverageCycleLength = float64(totalLength) / float64(len(cycles))
	stats.AverageCycleSize = float64(totalSize) / float64(len(cycles))

	// Calculate growth rate
	if len(metrics) > 1 {
		first := metrics[0]
		last := metrics[len(metrics)-1]
		timeDiff := last.Timestamp.Sub(first.Timestamp).Minutes()
		if timeDiff > 0 {
			cycleDiff := float64(last.CyclesDetected - first.CyclesDetected)
			stats.GrowthRate = cycleDiff / timeDiff
		}
	}

	// Find peak detection time
	stats.PeakDetectionTime = rd.findPeakDetectionTime(metrics)

	return stats
}

// findPeakDetectionTime finds the time with highest cycle detection rate
func (rd *ReferenceDetector) findPeakDetectionTime(metrics []ReferenceMetrics) string {
	if len(metrics) < 2 {
		return "insufficient_data"
	}

	maxDetectionRate := 0.0
	maxDetectionTime := metrics[0].Timestamp

	for i := 1; i < len(metrics); i++ {
		timeDiff := metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Minutes()
		cycleDiff := float64(metrics[i].CyclesDetected - metrics[i-1].CyclesDetected)
		if timeDiff > 0 {
			detectionRate := cycleDiff / timeDiff
			if detectionRate > maxDetectionRate {
				maxDetectionRate = detectionRate
				maxDetectionTime = metrics[i].Timestamp
			}
		}
	}

	return maxDetectionTime.Format("2006-01-02 15:04:05")
}

// identifyHotspots identifies reference creation hotspots
func (rd *ReferenceDetector) identifyHotspots(cycles []ReferenceCycle) []ReferenceHotspot {
	locationStats := make(map[string]*ReferenceHotspot)

	// Get stack trace for hotspot identification
	for _, cycle := range cycles {
		// Use the first object in the cycle to determine location
		if len(cycle.Objects) > 0 {
			objID := cycle.Objects[0]
			if objInfo, exists := rd.objects[objID]; exists {
				// Simplified location detection (in real implementation, you'd use runtime.Caller)
				location := "unknown"
				if pc, _, _, ok := runtime.Caller(2); ok {
					if fn := runtime.FuncForPC(pc); fn != nil {
						location = fn.Name()
					}
				}

				if hotspot, exists := locationStats[location]; exists {
					hotspot.Count++
					hotspot.CycleRate = float64(hotspot.Count) / float64(len(cycles))
				} else {
					locationStats[location] = &ReferenceHotspot{
						Location:    location,
						TypeName:    objInfo.Type,
						Count:       1,
						CycleRate:   1.0 / float64(len(cycles)),
						Description: fmt.Sprintf("Reference creation hotspot for %s", objInfo.Type),
					}
				}
			}
		}
	}

	// Convert to slice and classify severity
	var hotspots []ReferenceHotspot
	for _, hotspot := range locationStats {
		hotspot.Severity = rd.classifyHotspotSeverity(hotspot.Count, hotspot.CycleRate)
		hotspots = append(hotspots, *hotspot)
	}

	// Sort by count (highest first)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Count > hotspots[j].Count
	})

	return hotspots
}

// classifyHotspotSeverity classifies hotspot severity
func (rd *ReferenceDetector) classifyHotspotSeverity(count int, cycleRate float64) string {
	if count > 10 || cycleRate > 0.5 {
		return "critical"
	} else if count > 5 || cycleRate > 0.3 {
		return "high"
	} else if count > 2 || cycleRate > 0.1 {
		return "medium"
	}
	return "low"
}

// CleanupStaleObjects removes stale object references
func (rd *ReferenceDetector) CleanupStaleObjects(maxAge time.Duration) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	now := time.Now()
	for objID, objInfo := range rd.objects {
		if now.Sub(objInfo.LastSeen) > maxAge {
			delete(rd.objects, objID)
		}
	}
}

// GetObjectCount returns the number of tracked objects
func (rd *ReferenceDetector) GetObjectCount() int {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()
	return len(rd.objects)
}

// GetCycleCount returns the number of detected cycles
func (rd *ReferenceDetector) GetCycleCount() int {
	rd.mutex.RLock()
	defer rd.mutex.RUnlock()
	return len(rd.cycles)
}