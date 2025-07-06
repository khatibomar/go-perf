package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/analyzer"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/patterns"
)

// AdvisorConfig holds configuration for optimization advisor
type AdvisorConfig struct {
	InputDir  string
	OutputDir string
	Verbose   bool
}

// OptimizationAdvisor provides optimization recommendations
type OptimizationAdvisor struct {
	config *AdvisorConfig
}

// NewOptimizationAdvisor creates a new optimization advisor
func NewOptimizationAdvisor(config *AdvisorConfig) *OptimizationAdvisor {
	return &OptimizationAdvisor{
		config: config,
	}
}

// AnalyzeOptimizations analyzes data and generates optimization recommendations
func (oa *OptimizationAdvisor) AnalyzeOptimizations() ([]Recommendation, error) {
	// Load analysis data
	patterns, err := oa.loadPatterns()
	if err != nil && oa.config.Verbose {
		fmt.Printf("Warning: failed to load patterns: %v\n", err)
	}

	hotspots, err := oa.loadHotspots()
	if err != nil && oa.config.Verbose {
		fmt.Printf("Warning: failed to load hotspots: %v\n", err)
	}

	metrics, err := oa.loadMetrics()
	if err != nil && oa.config.Verbose {
		fmt.Printf("Warning: failed to load metrics: %v\n", err)
	}

	// Generate recommendations
	var recommendations []Recommendation
	recommendations = append(recommendations, oa.analyzePatternOptimizations(patterns)...)
	recommendations = append(recommendations, oa.analyzeHotspotOptimizations(hotspots)...)
	recommendations = append(recommendations, oa.analyzeMetricOptimizations(metrics)...)
	recommendations = append(recommendations, oa.generateGeneralOptimizations(patterns, hotspots, metrics)...)

	// Sort by priority and severity
	sort.Slice(recommendations, func(i, j int) bool {
		if recommendations[i].Priority != recommendations[j].Priority {
			return recommendations[i].Priority < recommendations[j].Priority
		}
		return oa.getSeverityWeight(recommendations[i].Severity) > oa.getSeverityWeight(recommendations[j].Severity)
	})

	return recommendations, nil
}

// SaveRecommendations saves recommendations to file
func (oa *OptimizationAdvisor) SaveRecommendations(recommendations []Recommendation) error {
	if err := os.MkdirAll(oa.config.OutputDir, 0755); err != nil {
		return err
	}

	// Save as JSON
	jsonFile := filepath.Join(oa.config.OutputDir, "optimization_recommendations.json")
	file, err := os.Create(jsonFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(recommendations); err != nil {
		return err
	}

	// Save as text
	textFile := filepath.Join(oa.config.OutputDir, "optimization_recommendations.txt")
	return oa.saveRecommendationsAsText(recommendations, textFile)
}

// loadPatterns loads detected patterns
func (oa *OptimizationAdvisor) loadPatterns() ([]patterns.DetectedPattern, error) {
	patternsDir := filepath.Join(oa.config.InputDir, "patterns")
	files, err := filepath.Glob(filepath.Join(patternsDir, "pattern_*.json"))
	if err != nil {
		return nil, err
	}

	var allPatterns []patterns.DetectedPattern
	for _, file := range files {
		var pattern patterns.DetectedPattern
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(data, &pattern); err != nil {
			continue
		}

		allPatterns = append(allPatterns, pattern)
	}

	return allPatterns, nil
}

// loadHotspots loads hotspot analysis
func (oa *OptimizationAdvisor) loadHotspots() ([]*analyzer.HotspotInfo, error) {
	filename := filepath.Join(oa.config.InputDir, "hotspots.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var hotspots []*analyzer.HotspotInfo
	return hotspots, json.Unmarshal(data, &hotspots)
}

// loadMetrics loads allocation metrics
func (oa *OptimizationAdvisor) loadMetrics() ([]analyzer.AllocationMetrics, error) {
	filename := filepath.Join(oa.config.InputDir, "metrics.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var metrics []analyzer.AllocationMetrics
	return metrics, json.Unmarshal(data, &metrics)
}

// analyzePatternOptimizations generates recommendations based on patterns
func (oa *OptimizationAdvisor) analyzePatternOptimizations(detectedPatterns []patterns.DetectedPattern) []Recommendation {
	var recommendations []Recommendation
	now := time.Now()
	priority := 1

	// Group patterns by type
	patternGroups := make(map[patterns.PatternType][]patterns.DetectedPattern)
	for _, pattern := range detectedPatterns {
		patternGroups[pattern.Type] = append(patternGroups[pattern.Type], pattern)
	}

	// Generate recommendations for each pattern type
	for patternType, patternList := range patternGroups {
		switch patternType {
		case patterns.PatternBurst:
			recommendations = append(recommendations, oa.generateBurstOptimizations(patternList, &priority, now)...)
		case patterns.PatternGrowth:
			recommendations = append(recommendations, oa.generateGrowthOptimizations(patternList, &priority, now)...)
		case patterns.PatternCyclic:
			recommendations = append(recommendations, oa.generateCyclicOptimizations(patternList, &priority, now)...)
		case patterns.PatternSpike:
			recommendations = append(recommendations, oa.generateSpikeOptimizations(patternList, &priority, now)...)
		case patterns.PatternAnomaly:
			recommendations = append(recommendations, oa.generateAnomalyOptimizations(patternList, &priority, now)...)
		}
	}

	return recommendations
}

// generateBurstOptimizations generates recommendations for burst patterns
func (oa *OptimizationAdvisor) generateBurstOptimizations(detectedPatterns []patterns.DetectedPattern, priority *int, now time.Time) []Recommendation {
	var recommendations []Recommendation

	// Analyze burst characteristics
	totalBursts := len(detectedPatterns)
	highIntensityBursts := 0
	for _, pattern := range detectedPatterns {
		if pattern.Metrics.BurstIntensity > 3.0 {
			highIntensityBursts++
		}
	}

	if totalBursts > 5 {
		rec := Recommendation{
			ID:          fmt.Sprintf("burst_frequency_%d", now.Unix()),
			Type:        "burst_optimization",
			Severity:    "high",
			Title:       "Reduce Allocation Burst Frequency",
			Description: fmt.Sprintf("Detected %d allocation bursts. Consider implementing rate limiting or batching to smooth allocation patterns.", totalBursts),
			Impact:      "Reduce memory pressure spikes and improve GC performance",
			Effort:      "Medium",
			Priority:    *priority,
			Tags:        []string{"burst", "rate-limiting", "batching"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		*priority++
	}

	if highIntensityBursts > 0 {
		rec := Recommendation{
			ID:          fmt.Sprintf("burst_intensity_%d", now.Unix()),
			Type:        "burst_optimization",
			Severity:    "critical",
			Title:       "Implement Object Pooling for High-Intensity Bursts",
			Description: fmt.Sprintf("Detected %d high-intensity allocation bursts. Implement object pooling using sync.Pool to reuse objects.", highIntensityBursts),
			Impact:      "Significantly reduce allocation overhead and GC pressure",
			Effort:      "High",
			Priority:    *priority,
			Tags:        []string{"burst", "object-pooling", "sync.Pool"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		*priority++
	}

	return recommendations
}

// generateGrowthOptimizations generates recommendations for growth patterns
func (oa *OptimizationAdvisor) generateGrowthOptimizations(detectedPatterns []patterns.DetectedPattern, priority *int, now time.Time) []Recommendation {
	var recommendations []Recommendation

	// Find the highest growth rate
	maxGrowthRate := 0.0
	for _, pattern := range detectedPatterns {
		if pattern.Metrics.GrowthRate > maxGrowthRate {
			maxGrowthRate = pattern.Metrics.GrowthRate
		}
	}

	if maxGrowthRate > 50 {
		rec := Recommendation{
			ID:          fmt.Sprintf("memory_leak_%d", now.Unix()),
			Type:        "memory_leak",
			Severity:    "critical",
			Title:       "Investigate Potential Memory Leak",
			Description: fmt.Sprintf("Detected rapid memory growth (%.2f%% rate). This may indicate a memory leak. Review object lifecycle and ensure proper cleanup.", maxGrowthRate),
			Impact:      "Prevent out-of-memory errors and application crashes",
			Effort:      "High",
			Priority:    *priority,
			Tags:        []string{"memory-leak", "cleanup", "lifecycle"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		*priority++
	} else if maxGrowthRate > 20 {
		rec := Recommendation{
			ID:          fmt.Sprintf("growth_optimization_%d", now.Unix()),
			Type:        "growth_optimization",
			Severity:    "high",
			Title:       "Optimize Memory Growth Pattern",
			Description: fmt.Sprintf("Detected concerning memory growth (%.2f%% rate). Consider implementing cleanup routines and optimizing data structures.", maxGrowthRate),
			Impact:      "Improve memory efficiency and application stability",
			Effort:      "Medium",
			Priority:    *priority,
			Tags:        []string{"growth", "cleanup", "optimization"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		*priority++
	}

	return recommendations
}

// generateCyclicOptimizations generates recommendations for cyclic patterns
func (oa *OptimizationAdvisor) generateCyclicOptimizations(detectedPatterns []patterns.DetectedPattern, priority *int, now time.Time) []Recommendation {
	var recommendations []Recommendation

	// Analyze volatility
	highVolatility := 0
	for _, pattern := range detectedPatterns {
		if pattern.Metrics.Volatility > 0.5 {
			highVolatility++
		}
	}

	if highVolatility > 0 {
		rec := Recommendation{
			ID:          fmt.Sprintf("cyclic_smoothing_%d", now.Unix()),
			Type:        "cyclic_optimization",
			Severity:    "medium",
			Title:       "Smooth Cyclic Allocation Patterns",
			Description: fmt.Sprintf("Detected %d highly volatile cyclic patterns. Consider pre-allocation or buffering to smooth allocation cycles.", highVolatility),
			Impact:      "Reduce allocation variability and improve predictability",
			Effort:      "Medium",
			Priority:    *priority,
			Tags:        []string{"cyclic", "smoothing", "pre-allocation"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		*priority++
	}

	return recommendations
}

// generateSpikeOptimizations generates recommendations for spike patterns
func (oa *OptimizationAdvisor) generateSpikeOptimizations(detectedPatterns []patterns.DetectedPattern, priority *int, now time.Time) []Recommendation {
	var recommendations []Recommendation

	if len(detectedPatterns) > 3 {
		rec := Recommendation{
			ID:          fmt.Sprintf("spike_investigation_%d", now.Unix()),
			Type:        "spike_optimization",
			Severity:    "high",
			Title:       "Investigate Recurring Allocation Spikes",
			Description: fmt.Sprintf("Detected %d allocation spikes. Investigate the root cause and consider implementing spike prevention mechanisms.", len(detectedPatterns)),
			Impact:      "Prevent performance degradation during spike events",
			Effort:      "Medium",
			Priority:    *priority,
			Tags:        []string{"spike", "investigation", "prevention"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		*priority++
	}

	return recommendations
}

// generateAnomalyOptimizations generates recommendations for anomaly patterns
func (oa *OptimizationAdvisor) generateAnomalyOptimizations(detectedPatterns []patterns.DetectedPattern, priority *int, now time.Time) []Recommendation {
	var recommendations []Recommendation

	if len(detectedPatterns) > 0 {
		rec := Recommendation{
			ID:          fmt.Sprintf("anomaly_monitoring_%d", now.Unix()),
			Type:        "anomaly_optimization",
			Severity:    "medium",
			Title:       "Implement Anomaly Detection and Alerting",
			Description: fmt.Sprintf("Detected %d allocation anomalies. Implement real-time monitoring and alerting for early detection.", len(detectedPatterns)),
			Impact:      "Enable proactive response to allocation anomalies",
			Effort:      "Low",
			Priority:    *priority,
			Tags:        []string{"anomaly", "monitoring", "alerting"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		*priority++
	}

	return recommendations
}

// analyzeHotspotOptimizations generates recommendations based on hotspots
func (oa *OptimizationAdvisor) analyzeHotspotOptimizations(hotspots []*analyzer.HotspotInfo) []Recommendation {
	var recommendations []Recommendation
	now := time.Now()
	priority := 100 // Start after pattern recommendations

	// Group hotspots by function
	functionGroups := make(map[string][]*analyzer.HotspotInfo)
	for _, hotspot := range hotspots {
		functionGroups[hotspot.Function] = append(functionGroups[hotspot.Function], hotspot)
	}

	// Generate function-specific recommendations
	for function, hotspotList := range functionGroups {
		if len(hotspotList) > 1 {
			// Multiple hotspots in same function
			totalRate := 0.0
			for _, h := range hotspotList {
				totalRate += h.AllocationRate
			}

			rec := Recommendation{
				ID:          fmt.Sprintf("function_hotspot_%s_%d", strings.ReplaceAll(function, ".", "_"), now.Unix()),
				Type:        "function_optimization",
				Severity:    "high",
				Title:       fmt.Sprintf("Optimize Function: %s", function),
				Description: fmt.Sprintf("Function %s has %d allocation hotspots with total rate %.2f/s. Consider refactoring for better memory efficiency.", function, len(hotspotList), totalRate),
				Impact:      "Reduce allocation overhead in frequently called function",
				Effort:      "High",
				Priority:    priority,
				Tags:        []string{"hotspot", "function", "refactoring"},
				CreatedAt:   now,
			}
			recommendations = append(recommendations, rec)
			priority++
		}
	}

	// Generate specific recommendations for critical hotspots
	for _, hotspot := range hotspots {
		if hotspot.Severity == "critical" {
			rec := oa.generateHotspotSpecificRecommendation(hotspot, priority, now)
			recommendations = append(recommendations, rec)
			priority++
		}
	}

	return recommendations
}

// generateHotspotSpecificRecommendation generates specific recommendation for a hotspot
func (oa *OptimizationAdvisor) generateHotspotSpecificRecommendation(hotspot *analyzer.HotspotInfo, priority int, now time.Time) Recommendation {
	// Analyze function name for specific recommendations
	function := strings.ToLower(hotspot.Function)
	var specificAdvice string
	var tags []string

	if strings.Contains(function, "append") {
		specificAdvice = "Pre-allocate slice capacity using make([]T, 0, capacity) to avoid repeated reallocations."
		tags = []string{"slice", "pre-allocation", "append"}
	} else if strings.Contains(function, "make") {
		specificAdvice = "Consider using object pooling with sync.Pool to reuse allocated objects."
		tags = []string{"object-pooling", "sync.Pool", "reuse"}
	} else if strings.Contains(function, "string") {
		specificAdvice = "Use strings.Builder for efficient string concatenation instead of repeated string operations."
		tags = []string{"string", "strings.Builder", "concatenation"}
	} else if strings.Contains(function, "json") {
		specificAdvice = "Consider using streaming JSON processing or object pooling for JSON marshaling/unmarshaling."
		tags = []string{"json", "streaming", "object-pooling"}
	} else {
		specificAdvice = "Review allocation patterns and consider optimization techniques like object pooling or pre-allocation."
		tags = []string{"general", "optimization"}
	}

	return Recommendation{
		ID:          fmt.Sprintf("critical_hotspot_%s_%d", strings.ReplaceAll(hotspot.Location, ":", "_"), now.Unix()),
		Type:        "critical_hotspot",
		Severity:    "critical",
		Title:       fmt.Sprintf("Critical Hotspot: %s", hotspot.Function),
		Description: fmt.Sprintf("Critical allocation hotspot at %s with rate %.2f/s. %s", hotspot.Location, hotspot.AllocationRate, specificAdvice),
		Impact:      "Significant reduction in allocation overhead and improved performance",
		Effort:      "High",
		Priority:    priority,
		Tags:        tags,
		CreatedAt:   now,
	}
}

// analyzeMetricOptimizations generates recommendations based on metrics
func (oa *OptimizationAdvisor) analyzeMetricOptimizations(metrics []analyzer.AllocationMetrics) []Recommendation {
	var recommendations []Recommendation
	now := time.Now()
	priority := 200 // Start after hotspot recommendations

	if len(metrics) == 0 {
		return recommendations
	}

	// Analyze GC pressure
	gcPressure := oa.analyzeGCPressure(metrics)
	if gcPressure > 0.7 {
		rec := Recommendation{
			ID:          fmt.Sprintf("gc_pressure_%d", now.Unix()),
			Type:        "gc_optimization",
			Severity:    "high",
			Title:       "Reduce Garbage Collection Pressure",
			Description: fmt.Sprintf("High GC pressure detected (%.2f). Consider reducing allocation rate and implementing object reuse strategies.", gcPressure),
			Impact:      "Improve application responsiveness and reduce GC overhead",
			Effort:      "Medium",
			Priority:    priority,
			Tags:        []string{"gc", "pressure", "allocation-rate"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		priority++
	}

	// Analyze memory growth
	memoryGrowth := oa.analyzeMemoryGrowth(metrics)
	if memoryGrowth > 0.5 {
		rec := Recommendation{
			ID:          fmt.Sprintf("memory_growth_%d", now.Unix()),
			Type:        "memory_optimization",
			Severity:    "medium",
			Title:       "Monitor Memory Growth Trend",
			Description: fmt.Sprintf("Detected memory growth trend (%.2f). Implement memory monitoring and consider periodic cleanup.", memoryGrowth),
			Impact:      "Prevent potential memory issues and improve stability",
			Effort:      "Low",
			Priority:    priority,
			Tags:        []string{"memory", "growth", "monitoring"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		priority++
	}

	return recommendations
}

// generateGeneralOptimizations generates general optimization recommendations
func (oa *OptimizationAdvisor) generateGeneralOptimizations(patterns []patterns.DetectedPattern, hotspots []*analyzer.HotspotInfo, metrics []analyzer.AllocationMetrics) []Recommendation {
	var recommendations []Recommendation
	now := time.Now()
	priority := 300 // Start after specific recommendations

	// General recommendations based on overall analysis
	if len(patterns) > 10 || len(hotspots) > 5 {
		rec := Recommendation{
			ID:          fmt.Sprintf("general_monitoring_%d", now.Unix()),
			Type:        "monitoring",
			Severity:    "medium",
			Title:       "Implement Comprehensive Memory Monitoring",
			Description: "High number of patterns and hotspots detected. Implement continuous monitoring to track optimization progress.",
			Impact:      "Enable data-driven optimization decisions",
			Effort:      "Low",
			Priority:    priority,
			Tags:        []string{"monitoring", "metrics", "continuous"},
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		priority++
	}

	// Recommend profiling tools
	rec := Recommendation{
		ID:          fmt.Sprintf("profiling_tools_%d", now.Unix()),
		Type:        "tooling",
		Severity:    "low",
		Title:       "Utilize Go Profiling Tools",
		Description: "Use go tool pprof, go tool trace, and runtime/pprof for detailed memory analysis and optimization guidance.",
		Impact:      "Gain deeper insights into memory usage patterns",
		Effort:      "Low",
		Priority:    priority,
		Tags:        []string{"profiling", "pprof", "trace", "tools"},
		CreatedAt:   now,
	}
	recommendations = append(recommendations, rec)

	return recommendations
}

// Helper functions

// getSeverityWeight returns numeric weight for severity
func (oa *OptimizationAdvisor) getSeverityWeight(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// analyzeGCPressure analyzes GC pressure from metrics
func (oa *OptimizationAdvisor) analyzeGCPressure(metrics []analyzer.AllocationMetrics) float64 {
	if len(metrics) < 2 {
		return 0
	}

	// Simple GC pressure calculation based on GC frequency
	first := metrics[0]
	last := metrics[len(metrics)-1]

	gcDelta := last.NumGC - first.NumGC
	timeDelta := last.Timestamp.Sub(first.Timestamp).Seconds()

	if timeDelta == 0 {
		return 0
	}

	gcRate := float64(gcDelta) / timeDelta
	// Normalize to 0-1 scale (assuming 1 GC per second is high pressure)
	return gcRate
}

// analyzeMemoryGrowth analyzes memory growth trend
func (oa *OptimizationAdvisor) analyzeMemoryGrowth(metrics []analyzer.AllocationMetrics) float64 {
	if len(metrics) < 2 {
		return 0
	}

	first := metrics[0]
	last := metrics[len(metrics)-1]

	if first.HeapSize == 0 {
		return 0
	}

	growthRatio := float64(last.HeapSize) / float64(first.HeapSize)
	return growthRatio - 1.0 // Return growth as ratio above 1.0
}

// saveRecommendationsAsText saves recommendations in text format
func (oa *OptimizationAdvisor) saveRecommendationsAsText(recommendations []Recommendation, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Optimization Recommendations\n")
	fmt.Fprintf(file, "===========================\n\n")
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Total Recommendations: %d\n\n", len(recommendations))

	for i, rec := range recommendations {
		fmt.Fprintf(file, "%d. %s\n", i+1, rec.Title)
		fmt.Fprintf(file, "   Type: %s\n", rec.Type)
		fmt.Fprintf(file, "   Severity: %s\n", rec.Severity)
		fmt.Fprintf(file, "   Priority: %d\n", rec.Priority)
		fmt.Fprintf(file, "   Effort: %s\n", rec.Effort)
		fmt.Fprintf(file, "   Description: %s\n", rec.Description)
		fmt.Fprintf(file, "   Impact: %s\n", rec.Impact)
		fmt.Fprintf(file, "   Tags: %s\n", strings.Join(rec.Tags, ", "))
		fmt.Fprintf(file, "\n")
	}

	return nil
}