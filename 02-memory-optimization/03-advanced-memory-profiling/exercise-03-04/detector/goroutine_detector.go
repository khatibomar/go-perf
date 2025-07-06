package detector

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat"
)

// GoroutineConfig holds configuration for goroutine leak detection
type GoroutineConfig struct {
	Interval         time.Duration
	GrowthThreshold  int     // Number of goroutines growth threshold
	GrowthRate       float64 // Goroutines per minute threshold
	Verbose          bool
	StackTraceDepth  int
	SampleStackTrace bool
}

// GoroutineMetrics represents goroutine metrics at a point in time
type GoroutineMetrics struct {
	Timestamp       time.Time            `json:"timestamp"`
	Count           int                  `json:"count"`
	Running         int                  `json:"running"`
	Runnable        int                  `json:"runnable"`
	Waiting         int                  `json:"waiting"`
	Syscall         int                  `json:"syscall"`
	Dead            int                  `json:"dead"`
	StackTraces     []GoroutineStack     `json:"stack_traces,omitempty"`
	FunctionCounts  map[string]int       `json:"function_counts"`
	PackageCounts   map[string]int       `json:"package_counts"`
	CreationSources map[string]int       `json:"creation_sources"`
	WaitReasons     map[string]int       `json:"wait_reasons"`
}

// GoroutineStack represents a goroutine stack trace
type GoroutineStack struct {
	ID          int      `json:"id"`
	State       string   `json:"state"`
	WaitReason  string   `json:"wait_reason,omitempty"`
	WaitSince   string   `json:"wait_since,omitempty"`
	CreatedBy   string   `json:"created_by,omitempty"`
	StackFrames []string `json:"stack_frames"`
	Function    string   `json:"function"`
	Package     string   `json:"package"`
}

// GoroutineAnalysis represents the result of goroutine analysis
type GoroutineAnalysis struct {
	StartTime           time.Time              `json:"start_time"`
	EndTime             time.Time              `json:"end_time"`
	Duration            time.Duration          `json:"duration"`
	Metrics             []GoroutineMetrics     `json:"metrics"`
	GrowthRate          float64                `json:"growth_rate_per_min"`
	TrendDirection      string                 `json:"trend_direction"`
	LeakProbability     float64                `json:"leak_probability"`
	Severity            string                 `json:"severity"`
	SuspiciousPatterns  []SuspiciousPattern    `json:"suspicious_patterns"`
	LeakCandidates      []LeakCandidate        `json:"leak_candidates"`
	Recommendations     []string               `json:"recommendations"`
	Statistics          GoroutineStats         `json:"statistics"`
	Hotspots            []GoroutineHotspot     `json:"hotspots"`
}

// SuspiciousPattern represents a suspicious goroutine pattern
type SuspiciousPattern struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Count       int       `json:"count"`
	GrowthRate  float64   `json:"growth_rate"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Function    string    `json:"function"`
	Package     string    `json:"package"`
}

// LeakCandidate represents a potential goroutine leak
type LeakCandidate struct {
	Function       string    `json:"function"`
	Package        string    `json:"package"`
	Count          int       `json:"count"`
	GrowthRate     float64   `json:"growth_rate"`
	Confidence     float64   `json:"confidence"`
	FirstDetected  time.Time `json:"first_detected"`
	LatestDetected time.Time `json:"latest_detected"`
	StackTrace     []string  `json:"stack_trace"`
	WaitReason     string    `json:"wait_reason"`
	Suggestion     string    `json:"suggestion"`
}

// GoroutineStats contains statistical analysis of goroutine metrics
type GoroutineStats struct {
	MeanCount           float64            `json:"mean_count"`
	StdDevCount         float64            `json:"stddev_count"`
	MinCount            int                `json:"min_count"`
	MaxCount            int                `json:"max_count"`
	GrowthVariance      float64            `json:"growth_variance"`
	MostCommonFunctions map[string]int     `json:"most_common_functions"`
	MostCommonPackages  map[string]int     `json:"most_common_packages"`
	StateDistribution   map[string]float64 `json:"state_distribution"`
	PeakGrowthPeriod    string             `json:"peak_growth_period"`
}

// GoroutineHotspot represents a goroutine creation hotspot
type GoroutineHotspot struct {
	Function    string  `json:"function"`
	Package     string  `json:"package"`
	Count       int     `json:"count"`
	GrowthRate  float64 `json:"growth_rate"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
}

// GoroutineDetector monitors and detects goroutine leaks
type GoroutineDetector struct {
	config   *GoroutineConfig
	metrics  []GoroutineMetrics
	running  bool
	stopChan chan struct{}
	mutex    sync.RWMutex
}

// NewGoroutineDetector creates a new goroutine detector
func NewGoroutineDetector(config *GoroutineConfig) *GoroutineDetector {
	return &GoroutineDetector{
		config:   config,
		metrics:  make([]GoroutineMetrics, 0),
		stopChan: make(chan struct{}),
	}
}

// Start begins goroutine monitoring
func (gd *GoroutineDetector) Start() error {
	gd.mutex.Lock()
	defer gd.mutex.Unlock()

	if gd.running {
		return fmt.Errorf("goroutine detector already running")
	}

	gd.running = true
	go gd.monitorLoop()

	if gd.config.Verbose {
		fmt.Printf("Goroutine detector started with interval %v\n", gd.config.Interval)
	}

	return nil
}

// Stop stops goroutine monitoring
func (gd *GoroutineDetector) Stop() {
	gd.mutex.Lock()
	defer gd.mutex.Unlock()

	if !gd.running {
		return
	}

	gd.running = false
	close(gd.stopChan)

	if gd.config.Verbose {
		fmt.Println("Goroutine detector stopped")
	}
}

// GetMetrics returns current metrics
func (gd *GoroutineDetector) GetMetrics() []GoroutineMetrics {
	gd.mutex.RLock()
	defer gd.mutex.RUnlock()

	// Return a copy to avoid race conditions
	metrics := make([]GoroutineMetrics, len(gd.metrics))
	copy(metrics, gd.metrics)
	return metrics
}

// AnalyzeGoroutines performs comprehensive goroutine analysis
func (gd *GoroutineDetector) AnalyzeGoroutines() (*GoroutineAnalysis, error) {
	gd.mutex.RLock()
	metrics := make([]GoroutineMetrics, len(gd.metrics))
	copy(metrics, gd.metrics)
	gd.mutex.RUnlock()

	if len(metrics) < 2 {
		return nil, fmt.Errorf("insufficient data for analysis (need at least 2 data points)")
	}

	analysis := &GoroutineAnalysis{
		StartTime: metrics[0].Timestamp,
		EndTime:   metrics[len(metrics)-1].Timestamp,
		Metrics:   metrics,
	}
	analysis.Duration = analysis.EndTime.Sub(analysis.StartTime)

	// Calculate growth rate
	analysis.GrowthRate = gd.calculateGrowthRate(metrics)
	analysis.TrendDirection = gd.determineTrendDirection(analysis.GrowthRate)

	// Calculate leak probability
	analysis.LeakProbability = gd.calculateLeakProbability(metrics, analysis.GrowthRate)
	analysis.Severity = gd.determineSeverity(analysis.LeakProbability, analysis.GrowthRate)

	// Detect suspicious patterns
	analysis.SuspiciousPatterns = gd.detectSuspiciousPatterns(metrics)

	// Identify leak candidates
	analysis.LeakCandidates = gd.identifyLeakCandidates(metrics)

	// Generate recommendations
	analysis.Recommendations = gd.generateRecommendations(analysis)

	// Calculate statistics
	analysis.Statistics = gd.calculateStatistics(metrics)

	// Identify hotspots
	analysis.Hotspots = gd.identifyHotspots(metrics)

	return analysis, nil
}

// SaveResults saves analysis results to files
func (gd *GoroutineDetector) SaveResults(outputDir string) error {
	analysis, err := gd.AnalyzeGoroutines()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Save detailed analysis
	analysisFile := filepath.Join(outputDir, "goroutine_analysis.json")
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

	// Save raw metrics
	metricsFile := filepath.Join(outputDir, "goroutine_metrics.json")
	file, err = os.Create(metricsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder = json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(gd.GetMetrics())
}

// monitorLoop runs the monitoring loop
func (gd *GoroutineDetector) monitorLoop() {
	ticker := time.NewTicker(gd.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gd.collectMetrics()
		case <-gd.stopChan:
			return
		}
	}
}

// collectMetrics collects current goroutine metrics
func (gd *GoroutineDetector) collectMetrics() {
	metrics := GoroutineMetrics{
		Timestamp:       time.Now(),
		Count:           runtime.NumGoroutine(),
		FunctionCounts:  make(map[string]int),
		PackageCounts:   make(map[string]int),
		CreationSources: make(map[string]int),
		WaitReasons:     make(map[string]int),
	}

	// Collect detailed stack traces if enabled
	if gd.config.SampleStackTrace {
		stackTraces, err := gd.collectStackTraces()
		if err == nil {
			metrics.StackTraces = stackTraces
			gd.analyzeStackTraces(stackTraces, &metrics)
		}
	}

	gd.mutex.Lock()
	gd.metrics = append(gd.metrics, metrics)

	// Keep only recent metrics to prevent memory growth
	maxMetrics := 10000
	if len(gd.metrics) > maxMetrics {
		gd.metrics = gd.metrics[len(gd.metrics)-maxMetrics:]
	}
	gd.mutex.Unlock()

	// Check for immediate alerts
	if len(gd.metrics) >= 2 {
		gd.checkForAlerts(metrics)
	}
}

// collectStackTraces collects goroutine stack traces
func (gd *GoroutineDetector) collectStackTraces() ([]GoroutineStack, error) {
	var buf bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&buf, 1); err != nil {
		return nil, err
	}

	return gd.parseStackTraces(buf.String())
}

// parseStackTraces parses goroutine stack traces from pprof output
func (gd *GoroutineDetector) parseStackTraces(pprofOutput string) ([]GoroutineStack, error) {
	var stacks []GoroutineStack
	scanner := bufio.NewScanner(strings.NewReader(pprofOutput))

	// Regex patterns for parsing
	goroutinePattern := regexp.MustCompile(`^goroutine (\d+) \[([^\]]+)(?:, (\d+) minutes)?(?:, locked to thread)?\]:$`)
	createdByPattern := regexp.MustCompile(`^created by (.+)$`)
	functionPattern := regexp.MustCompile(`^([^\s]+)\(.*\)$`)

	var currentStack *GoroutineStack
	var inStackTrace bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			if currentStack != nil {
				stacks = append(stacks, *currentStack)
				currentStack = nil
			}
			inStackTrace = false
			continue
		}

		// Parse goroutine header
		if matches := goroutinePattern.FindStringSubmatch(line); matches != nil {
			id, _ := strconv.Atoi(matches[1])
			state := matches[2]
			waitSince := ""
			if len(matches) > 3 && matches[3] != "" {
				waitSince = matches[3] + " minutes"
			}

			currentStack = &GoroutineStack{
				ID:         id,
				State:      state,
				WaitSince:  waitSince,
				WaitReason: gd.extractWaitReason(state),
			}
			inStackTrace = true
			continue
		}

		// Parse created by line
		if matches := createdByPattern.FindStringSubmatch(line); matches != nil && currentStack != nil {
			currentStack.CreatedBy = matches[1]
			continue
		}

		// Parse stack frames
		if inStackTrace && currentStack != nil {
			if strings.Contains(line, "(") && strings.Contains(line, ")") {
				// This is a function call line
				currentStack.StackFrames = append(currentStack.StackFrames, line)

				// Extract function and package for the first frame
				if currentStack.Function == "" {
					if matches := functionPattern.FindStringSubmatch(line); matches != nil {
						fullFunc := matches[1]
						parts := strings.Split(fullFunc, ".")
						if len(parts) >= 2 {
							currentStack.Package = strings.Join(parts[:len(parts)-1], ".")
							currentStack.Function = parts[len(parts)-1]
						} else {
							currentStack.Function = fullFunc
						}
					}
				}

				// Limit stack depth
				if len(currentStack.StackFrames) >= gd.config.StackTraceDepth {
					break
				}
			}
		}
	}

	// Add the last stack if exists
	if currentStack != nil {
		stacks = append(stacks, *currentStack)
	}

	return stacks, nil
}

// extractWaitReason extracts wait reason from goroutine state
func (gd *GoroutineDetector) extractWaitReason(state string) string {
	if strings.Contains(state, "chan") {
		return "channel_operation"
	} else if strings.Contains(state, "select") {
		return "select_operation"
	} else if strings.Contains(state, "sleep") {
		return "sleep"
	} else if strings.Contains(state, "IO") {
		return "io_operation"
	} else if strings.Contains(state, "sync") {
		return "synchronization"
	} else if strings.Contains(state, "syscall") {
		return "system_call"
	} else if strings.Contains(state, "GC") {
		return "garbage_collection"
	}
	return "unknown"
}

// analyzeStackTraces analyzes stack traces and updates metrics
func (gd *GoroutineDetector) analyzeStackTraces(stacks []GoroutineStack, metrics *GoroutineMetrics) {
	for _, stack := range stacks {
		// Count by state
		switch stack.State {
		case "running":
			metrics.Running++
		case "runnable":
			metrics.Runnable++
		case "syscall":
			metrics.Syscall++
		default:
			metrics.Waiting++
		}

		// Count by function and package
		if stack.Function != "" {
			metrics.FunctionCounts[stack.Function]++
		}
		if stack.Package != "" {
			metrics.PackageCounts[stack.Package]++
		}

		// Count by creation source
		if stack.CreatedBy != "" {
			metrics.CreationSources[stack.CreatedBy]++
		}

		// Count by wait reason
		if stack.WaitReason != "" {
			metrics.WaitReasons[stack.WaitReason]++
		}
	}
}

// calculateGrowthRate calculates goroutine growth rate per minute
func (gd *GoroutineDetector) calculateGrowthRate(metrics []GoroutineMetrics) float64 {
	if len(metrics) < 2 {
		return 0
	}

	// Use linear regression to calculate growth rate
	x := make([]float64, len(metrics))
	y := make([]float64, len(metrics))

	startTime := metrics[0].Timestamp
	for i, metric := range metrics {
		x[i] = metric.Timestamp.Sub(startTime).Minutes()
		y[i] = float64(metric.Count)
	}

	// Calculate linear regression slope
	alpha, beta := stat.LinearRegression(x, y, nil, false)
	_ = alpha // Intercept not needed for growth rate

	return beta // Slope is growth rate in goroutines/min
}

// determineTrendDirection determines the trend direction
func (gd *GoroutineDetector) determineTrendDirection(growthRate float64) string {
	if growthRate > 0.1 {
		return "increasing"
	} else if growthRate < -0.1 {
		return "decreasing"
	}
	return "stable"
}

// calculateLeakProbability calculates the probability of a goroutine leak
func (gd *GoroutineDetector) calculateLeakProbability(metrics []GoroutineMetrics, growthRate float64) float64 {
	probability := 0.0

	// Factor 1: Growth rate (higher rate = higher probability)
	if growthRate > gd.config.GrowthRate {
		probability += 0.4 * math.Min(growthRate/gd.config.GrowthRate, 2.0)
	}

	// Factor 2: Consistency of growth
	consistency := gd.calculateGrowthConsistency(metrics)
	probability += 0.3 * consistency

	// Factor 3: Suspicious patterns
	suspiciousScore := gd.calculateSuspiciousScore(metrics)
	probability += 0.2 * suspiciousScore

	// Factor 4: State distribution anomalies
	stateAnomalyScore := gd.calculateStateAnomalyScore(metrics)
	probability += 0.1 * stateAnomalyScore

	return math.Min(probability, 1.0)
}

// calculateGrowthConsistency calculates how consistent the growth pattern is
func (gd *GoroutineDetector) calculateGrowthConsistency(metrics []GoroutineMetrics) float64 {
	if len(metrics) < 3 {
		return 0
	}

	// Calculate growth deltas
	deltas := make([]float64, len(metrics)-1)
	for i := 1; i < len(metrics); i++ {
		timeDiff := metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Minutes()
		countDiff := float64(metrics[i].Count - metrics[i-1].Count)
		if timeDiff > 0 {
			deltas[i-1] = countDiff / timeDiff
		}
	}

	// Calculate coefficient of variation (lower = more consistent)
	mean := stat.Mean(deltas, nil)
	stddev := stat.StdDev(deltas, nil)

	if mean == 0 {
		return 0
	}

	cv := stddev / math.Abs(mean)
	// Convert to consistency score (higher = more consistent)
	return math.Max(0, 1.0-cv/2.0)
}

// calculateSuspiciousScore calculates a score based on suspicious patterns
func (gd *GoroutineDetector) calculateSuspiciousScore(metrics []GoroutineMetrics) float64 {
	if len(metrics) < 2 {
		return 0
	}

	score := 0.0

	// Check for functions that consistently grow
	functionGrowth := make(map[string][]int)
	for _, metric := range metrics {
		for function, count := range metric.FunctionCounts {
			functionGrowth[function] = append(functionGrowth[function], count)
		}
	}

	for _, counts := range functionGrowth {
		if len(counts) >= 3 {
			// Check if consistently growing
			growingCount := 0
			for i := 1; i < len(counts); i++ {
				if counts[i] > counts[i-1] {
					growingCount++
				}
			}
			if float64(growingCount)/float64(len(counts)-1) > 0.8 {
				score += 0.2
			}
		}
	}

	return math.Min(score, 1.0)
}

// calculateStateAnomalyScore calculates anomaly score based on goroutine states
func (gd *GoroutineDetector) calculateStateAnomalyScore(metrics []GoroutineMetrics) float64 {
	if len(metrics) < 2 {
		return 0
	}

	score := 0.0

	// Check for unusual state distributions
	for i := 1; i < len(metrics); i++ {
		current := metrics[i]
		previous := metrics[i-1]

		// Check for sudden increase in waiting goroutines
		if current.Count > 0 && previous.Count > 0 {
			currentWaitingRatio := float64(current.Waiting) / float64(current.Count)
			previousWaitingRatio := float64(previous.Waiting) / float64(previous.Count)

			if currentWaitingRatio > 0.8 && currentWaitingRatio > previousWaitingRatio*1.5 {
				score += 0.3
			}
		}
	}

	return math.Min(score, 1.0)
}

// determineSeverity determines the severity level
func (gd *GoroutineDetector) determineSeverity(leakProbability, growthRate float64) string {
	if leakProbability > 0.8 || growthRate > gd.config.GrowthRate*3 {
		return "critical"
	} else if leakProbability > 0.6 || growthRate > gd.config.GrowthRate*2 {
		return "high"
	} else if leakProbability > 0.4 || growthRate > gd.config.GrowthRate {
		return "medium"
	}
	return "low"
}

// detectSuspiciousPatterns detects suspicious goroutine patterns
func (gd *GoroutineDetector) detectSuspiciousPatterns(metrics []GoroutineMetrics) []SuspiciousPattern {
	var patterns []SuspiciousPattern

	if len(metrics) < 3 {
		return patterns
	}

	// Track function growth patterns
	functionGrowth := make(map[string][]int)
	functionTimestamps := make(map[string][]time.Time)

	for _, metric := range metrics {
		for function, count := range metric.FunctionCounts {
			functionGrowth[function] = append(functionGrowth[function], count)
			functionTimestamps[function] = append(functionTimestamps[function], metric.Timestamp)
		}
	}

	// Analyze each function's growth pattern
	for function, counts := range functionGrowth {
		if len(counts) < 3 {
			continue
		}

		// Calculate growth rate for this function
		timestamps := functionTimestamps[function]
		x := make([]float64, len(timestamps))
		y := make([]float64, len(counts))

		startTime := timestamps[0]
		for i := range timestamps {
			x[i] = timestamps[i].Sub(startTime).Minutes()
			y[i] = float64(counts[i])
		}

		_, growthRate := stat.LinearRegression(x, y, nil, false)

		// Detect suspicious patterns
		if growthRate > 1.0 { // More than 1 goroutine per minute growth
			pattern := SuspiciousPattern{
				Type:        "consistent_growth",
				Description: fmt.Sprintf("Function %s shows consistent goroutine growth", function),
				Severity:    gd.classifyPatternSeverity(growthRate),
				Count:       counts[len(counts)-1],
				GrowthRate:  growthRate,
				FirstSeen:   timestamps[0],
				LastSeen:    timestamps[len(timestamps)-1],
				Function:    function,
			}

			// Extract package name
			parts := strings.Split(function, ".")
			if len(parts) > 1 {
				pattern.Package = strings.Join(parts[:len(parts)-1], ".")
			}

			patterns = append(patterns, pattern)
		}
	}

	return patterns
}

// classifyPatternSeverity classifies pattern severity
func (gd *GoroutineDetector) classifyPatternSeverity(growthRate float64) string {
	if growthRate > 10 {
		return "critical"
	} else if growthRate > 5 {
		return "high"
	} else if growthRate > 2 {
		return "medium"
	}
	return "low"
}

// identifyLeakCandidates identifies potential goroutine leak candidates
func (gd *GoroutineDetector) identifyLeakCandidates(metrics []GoroutineMetrics) []LeakCandidate {
	var candidates []LeakCandidate

	if len(metrics) < 3 {
		return candidates
	}

	// Analyze function growth patterns
	functionGrowth := make(map[string][]int)
	functionTimestamps := make(map[string][]time.Time)
	functionStacks := make(map[string][]string)
	functionWaitReasons := make(map[string]string)

	for _, metric := range metrics {
		for function, count := range metric.FunctionCounts {
			functionGrowth[function] = append(functionGrowth[function], count)
			functionTimestamps[function] = append(functionTimestamps[function], metric.Timestamp)

			// Get stack trace and wait reason from latest metric
			for _, stack := range metric.StackTraces {
				if stack.Function == function {
					functionStacks[function] = stack.StackFrames
					functionWaitReasons[function] = stack.WaitReason
					break
				}
			}
		}
	}

	// Identify candidates based on growth patterns
	for function, counts := range functionGrowth {
		if len(counts) < 3 {
			continue
		}

		// Calculate growth rate and confidence
		timestamps := functionTimestamps[function]
		x := make([]float64, len(timestamps))
		y := make([]float64, len(counts))

		startTime := timestamps[0]
		for i := range timestamps {
			x[i] = timestamps[i].Sub(startTime).Minutes()
			y[i] = float64(counts[i])
		}

		_, growthRate := stat.LinearRegression(x, y, nil, false)
		confidence := gd.calculateLeakConfidence(counts, growthRate)

		// Only consider functions with significant growth and high confidence
		if growthRate > 0.5 && confidence > 0.6 {
			candidate := LeakCandidate{
				Function:       function,
				Count:          counts[len(counts)-1],
				GrowthRate:     growthRate,
				Confidence:     confidence,
				FirstDetected:  timestamps[0],
				LatestDetected: timestamps[len(timestamps)-1],
				StackTrace:     functionStacks[function],
				WaitReason:     functionWaitReasons[function],
				Suggestion:     gd.generateLeakSuggestion(function, functionWaitReasons[function]),
			}

			// Extract package name
			parts := strings.Split(function, ".")
			if len(parts) > 1 {
				candidate.Package = strings.Join(parts[:len(parts)-1], ".")
			}

			candidates = append(candidates, candidate)
		}
	}

	// Sort by confidence (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	return candidates
}

// calculateLeakConfidence calculates confidence score for leak detection
func (gd *GoroutineDetector) calculateLeakConfidence(counts []int, growthRate float64) float64 {
	confidence := 0.0

	// Factor 1: Consistency of growth (higher = more confident)
	growingCount := 0
	for i := 1; i < len(counts); i++ {
		if counts[i] >= counts[i-1] {
			growingCount++
		}
	}
	consistency := float64(growingCount) / float64(len(counts)-1)
	confidence += 0.5 * consistency

	// Factor 2: Growth rate magnitude
	confidence += 0.3 * math.Min(growthRate/5.0, 1.0)

	// Factor 3: Total count (higher counts = more confident)
	lastCount := counts[len(counts)-1]
	confidence += 0.2 * math.Min(float64(lastCount)/100.0, 1.0)

	return math.Min(confidence, 1.0)
}

// generateLeakSuggestion generates suggestions for fixing potential leaks
func (gd *GoroutineDetector) generateLeakSuggestion(function, waitReason string) string {
	switch waitReason {
	case "channel_operation":
		return "Check for unbuffered channels or missing channel receivers in " + function
	case "select_operation":
		return "Review select statements for missing default cases or deadlocks in " + function
	case "sleep":
		return "Verify sleep durations and consider using context cancellation in " + function
	case "io_operation":
		return "Check for hanging I/O operations and implement timeouts in " + function
	case "synchronization":
		return "Review mutex usage and check for deadlocks in " + function
	case "system_call":
		return "Monitor system call behavior and implement proper error handling in " + function
	default:
		return "Review goroutine lifecycle and ensure proper cleanup in " + function
	}
}

// generateRecommendations generates recommendations based on analysis
func (gd *GoroutineDetector) generateRecommendations(analysis *GoroutineAnalysis) []string {
	var recommendations []string

	if analysis.LeakProbability > 0.7 {
		recommendations = append(recommendations, "High probability of goroutine leak detected. Review goroutine lifecycle management.")
	}

	if analysis.GrowthRate > gd.config.GrowthRate*2 {
		recommendations = append(recommendations, "Rapid goroutine growth detected. Consider implementing goroutine pooling or rate limiting.")
	}

	if len(analysis.SuspiciousPatterns) > 3 {
		recommendations = append(recommendations, "Multiple suspicious patterns detected. Focus on the highest severity patterns first.")
	}

	if len(analysis.LeakCandidates) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Found %d potential leak candidates. Investigate functions with highest confidence scores.", len(analysis.LeakCandidates)))
	}

	if analysis.Statistics.StateDistribution["waiting"] > 0.8 {
		recommendations = append(recommendations, "High percentage of waiting goroutines. Check for blocking operations and deadlocks.")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Goroutine usage appears normal. Continue monitoring for any changes in patterns.")
	}

	return recommendations
}

// calculateStatistics calculates comprehensive goroutine statistics
func (gd *GoroutineDetector) calculateStatistics(metrics []GoroutineMetrics) GoroutineStats {
	if len(metrics) == 0 {
		return GoroutineStats{}
	}

	// Extract counts
	counts := make([]float64, len(metrics))
	for i, m := range metrics {
		counts[i] = float64(m.Count)
	}

	stats := GoroutineStats{
		MeanCount:           stat.Mean(counts, nil),
		StdDevCount:         stat.StdDev(counts, nil),
		MinCount:            metrics[0].Count,
		MaxCount:            metrics[0].Count,
		MostCommonFunctions: make(map[string]int),
		MostCommonPackages:  make(map[string]int),
		StateDistribution:   make(map[string]float64),
	}

	// Find min/max
	for _, m := range metrics {
		if m.Count < stats.MinCount {
			stats.MinCount = m.Count
		}
		if m.Count > stats.MaxCount {
			stats.MaxCount = m.Count
		}
	}

	// Calculate growth variance
	if len(metrics) > 1 {
		growthRates := make([]float64, len(metrics)-1)
		for i := 1; i < len(metrics); i++ {
			timeDiff := metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Minutes()
			countDiff := float64(metrics[i].Count - metrics[i-1].Count)
			if timeDiff > 0 {
				growthRates[i-1] = countDiff / timeDiff
			}
		}
		stats.GrowthVariance = stat.Variance(growthRates, nil)
	}

	// Aggregate function and package counts
	functionTotals := make(map[string]int)
	packageTotals := make(map[string]int)
	stateTotals := make(map[string]int)
	totalGoroutines := 0

	for _, metric := range metrics {
		for function, count := range metric.FunctionCounts {
			functionTotals[function] += count
		}
		for pkg, count := range metric.PackageCounts {
			packageTotals[pkg] += count
		}

		// Aggregate state counts
		stateTotals["running"] += metric.Running
		stateTotals["runnable"] += metric.Runnable
		stateTotals["waiting"] += metric.Waiting
		stateTotals["syscall"] += metric.Syscall
		totalGoroutines += metric.Count
	}

	// Get top functions and packages
	stats.MostCommonFunctions = gd.getTopN(functionTotals, 10)
	stats.MostCommonPackages = gd.getTopN(packageTotals, 10)

	// Calculate state distribution
	if totalGoroutines > 0 {
		for state, count := range stateTotals {
			stats.StateDistribution[state] = float64(count) / float64(totalGoroutines)
		}
	}

	// Find peak growth period
	stats.PeakGrowthPeriod = gd.findPeakGrowthPeriod(metrics)

	return stats
}

// getTopN returns top N items from a map
func (gd *GoroutineDetector) getTopN(items map[string]int, n int) map[string]int {
	type item struct {
		key   string
		value int
	}

	var sorted []item
	for k, v := range items {
		sorted = append(sorted, item{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].value > sorted[j].value
	})

	result := make(map[string]int)
	for i := 0; i < len(sorted) && i < n; i++ {
		result[sorted[i].key] = sorted[i].value
	}

	return result
}

// identifyHotspots identifies goroutine creation hotspots
func (gd *GoroutineDetector) identifyHotspots(metrics []GoroutineMetrics) []GoroutineHotspot {
	var hotspots []GoroutineHotspot

	if len(metrics) < 2 {
		return hotspots
	}

	// Calculate growth rates for each function
	functionGrowth := make(map[string][]int)
	functionTimestamps := make(map[string][]time.Time)

	for _, metric := range metrics {
		for function, count := range metric.FunctionCounts {
			functionGrowth[function] = append(functionGrowth[function], count)
			functionTimestamps[function] = append(functionTimestamps[function], metric.Timestamp)
		}
	}

	for function, counts := range functionGrowth {
		if len(counts) < 2 {
			continue
		}

		// Calculate growth rate
		timestamps := functionTimestamps[function]
		x := make([]float64, len(timestamps))
		y := make([]float64, len(counts))

		startTime := timestamps[0]
		for i := range timestamps {
			x[i] = timestamps[i].Sub(startTime).Minutes()
			y[i] = float64(counts[i])
		}

		_, growthRate := stat.LinearRegression(x, y, nil, false)

		// Consider as hotspot if growth rate is significant
		if growthRate > 0.5 {
			hotspot := GoroutineHotspot{
				Function:    function,
				Count:       counts[len(counts)-1],
				GrowthRate:  growthRate,
				Severity:    gd.classifyHotspotSeverity(growthRate, counts[len(counts)-1]),
				Description: fmt.Sprintf("Function creating goroutines at %.2f/min rate", growthRate),
			}

			// Extract package name
			parts := strings.Split(function, ".")
			if len(parts) > 1 {
				hotspot.Package = strings.Join(parts[:len(parts)-1], ".")
			}

			hotspots = append(hotspots, hotspot)
		}
	}

	// Sort by growth rate (highest first)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].GrowthRate > hotspots[j].GrowthRate
	})

	return hotspots
}

// classifyHotspotSeverity classifies hotspot severity
func (gd *GoroutineDetector) classifyHotspotSeverity(growthRate float64, count int) string {
	if growthRate > 10 || count > 1000 {
		return "critical"
	} else if growthRate > 5 || count > 500 {
		return "high"
	} else if growthRate > 2 || count > 100 {
		return "medium"
	}
	return "low"
}

// findPeakGrowthPeriod finds the period with highest growth rate
func (gd *GoroutineDetector) findPeakGrowthPeriod(metrics []GoroutineMetrics) string {
	if len(metrics) < 2 {
		return "insufficient_data"
	}

	maxGrowthRate := 0.0
	maxGrowthTime := metrics[0].Timestamp

	for i := 1; i < len(metrics); i++ {
		timeDiff := metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Minutes()
		countDiff := float64(metrics[i].Count - metrics[i-1].Count)
		if timeDiff > 0 {
			growthRate := countDiff / timeDiff
			if growthRate > maxGrowthRate {
				maxGrowthRate = growthRate
				maxGrowthTime = metrics[i].Timestamp
			}
		}
	}

	return maxGrowthTime.Format("2006-01-02 15:04:05")
}

// checkForAlerts checks for immediate alerts
func (gd *GoroutineDetector) checkForAlerts(current GoroutineMetrics) {
	gd.mutex.RLock()
	metricsLen := len(gd.metrics)
	gd.mutex.RUnlock()

	if metricsLen < 2 {
		return
	}

	gd.mutex.RLock()
	previous := gd.metrics[metricsLen-2]
	gd.mutex.RUnlock()

	// Check for rapid growth
	growth := current.Count - previous.Count
	if growth > gd.config.GrowthThreshold && gd.config.Verbose {
		fmt.Printf("ALERT: Rapid goroutine growth detected: +%d goroutines\n", growth)
	}

	// Check for excessive total count
	if current.Count > 10000 && gd.config.Verbose {
		fmt.Printf("ALERT: High goroutine count detected: %d goroutines\n", current.Count)
	}

	// Check for unusual state distribution
	if current.Count > 0 {
		waitingRatio := float64(current.Waiting) / float64(current.Count)
		if waitingRatio > 0.9 && gd.config.Verbose {
			fmt.Printf("ALERT: High percentage of waiting goroutines: %.1f%%\n", waitingRatio*100)
		}
	}
}