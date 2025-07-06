package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/analyzer"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/patterns"
)

// ComparatorConfig holds configuration for analysis comparator
type ComparatorConfig struct {
	BaselineDir string
	CurrentDir  string
	OutputDir   string
	Verbose     bool
}

// MetricsSummary holds summarized allocation metrics
type MetricsSummary struct {
	AvgAllocationRate float64 `json:"avg_allocation_rate"`
	AvgHeapSize       uint64  `json:"avg_heap_size"`
	GCFrequency       float64 `json:"gc_frequency"`
	AvgObjectCount    uint64  `json:"avg_object_count"`
	TotalAllocations  uint64  `json:"total_allocations"`
	PeakHeapSize      uint64  `json:"peak_heap_size"`
}

// ComparisonResult represents the result of comparing two analyses
type ComparisonResult struct {
	Timestamp       time.Time                `json:"timestamp"`
	BaselineDir     string                   `json:"baseline_dir"`
	CurrentDir      string                   `json:"current_dir"`
	MetricsChange   *MetricsComparison       `json:"metrics_change"`
	PatternsChange  *PatternsComparison      `json:"patterns_change"`
	HotspotsChange  *HotspotsComparison      `json:"hotspots_change"`
	OverallScore    float64                  `json:"overall_score"`
	Improvements    []string                 `json:"improvements"`
	Regressions     []string                 `json:"regressions"`
	Recommendations []ComparisonRecommendation `json:"recommendations"`
}

// MetricsComparison compares allocation metrics
type MetricsComparison struct {
	AllocationRateChange float64 `json:"allocation_rate_change"`
	HeapSizeChange       float64 `json:"heap_size_change"`
	GCFrequencyChange    float64 `json:"gc_frequency_change"`
	ObjectCountChange    float64 `json:"object_count_change"`
	BaselineMetrics      *MetricsSummary `json:"baseline_metrics"`
	CurrentMetrics       *MetricsSummary `json:"current_metrics"`
}

// PatternsComparison compares detected patterns
type PatternsComparison struct {
	NewPatterns     []patterns.PatternType `json:"new_patterns"`
	ResolvedPatterns []patterns.PatternType `json:"resolved_patterns"`
	PatternChanges  map[patterns.PatternType]*PatternChange `json:"pattern_changes"`
	BaselineCount   int `json:"baseline_count"`
	CurrentCount    int `json:"current_count"`
}

// PatternChange represents change in a specific pattern
type PatternChange struct {
	FrequencyChange  float64 `json:"frequency_change"`
	SeverityChange   string  `json:"severity_change"`
	IntensityChange  float64 `json:"intensity_change"`
	BaselineCount    int     `json:"baseline_count"`
	CurrentCount     int     `json:"current_count"`
}

// HotspotsComparison compares hotspot analysis
type HotspotsComparison struct {
	NewHotspots      []string `json:"new_hotspots"`
	ResolvedHotspots []string `json:"resolved_hotspots"`
	HotspotChanges   map[string]*HotspotChange `json:"hotspot_changes"`
	BaselineCount    int `json:"baseline_count"`
	CurrentCount     int `json:"current_count"`
}

// HotspotChange represents change in a specific hotspot
type HotspotChange struct {
	RateChange     float64 `json:"rate_change"`
	SeverityChange string  `json:"severity_change"`
	BaselineRate   float64 `json:"baseline_rate"`
	CurrentRate    float64 `json:"current_rate"`
}

// ComparisonRecommendation represents a recommendation based on comparison
type ComparisonRecommendation struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
}

// AnalysisComparator compares different analysis results
type AnalysisComparator struct {
	config *ComparatorConfig
}

// NewAnalysisComparator creates a new analysis comparator
func NewAnalysisComparator(config *ComparatorConfig) *AnalysisComparator {
	return &AnalysisComparator{
		config: config,
	}
}

// Compare compares baseline and current analysis results
func (ac *AnalysisComparator) Compare() (*ComparisonResult, error) {
	result := &ComparisonResult{
		Timestamp:   time.Now(),
		BaselineDir: ac.config.BaselineDir,
		CurrentDir:  ac.config.CurrentDir,
	}

	// Compare metrics
	metricsComparison, err := ac.compareMetrics()
	if err != nil && ac.config.Verbose {
		fmt.Printf("Warning: failed to compare metrics: %v\n", err)
	}
	result.MetricsChange = metricsComparison

	// Compare patterns
	patternsComparison, err := ac.comparePatterns()
	if err != nil && ac.config.Verbose {
		fmt.Printf("Warning: failed to compare patterns: %v\n", err)
	}
	result.PatternsChange = patternsComparison

	// Compare hotspots
	hotspotsComparison, err := ac.compareHotspots()
	if err != nil && ac.config.Verbose {
		fmt.Printf("Warning: failed to compare hotspots: %v\n", err)
	}
	result.HotspotsChange = hotspotsComparison

	// Calculate overall score and generate insights
	result.OverallScore = ac.calculateOverallScore(result)
	result.Improvements, result.Regressions = ac.identifyChanges(result)
	result.Recommendations = ac.generateComparisonRecommendations(result)

	return result, nil
}

// SaveComparison saves comparison results
func (ac *AnalysisComparator) SaveComparison(result *ComparisonResult) error {
	if err := os.MkdirAll(ac.config.OutputDir, 0755); err != nil {
		return err
	}

	// Save as JSON
	jsonFile := filepath.Join(ac.config.OutputDir, "comparison_result.json")
	file, err := os.Create(jsonFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return err
	}

	// Save as text report
	textFile := filepath.Join(ac.config.OutputDir, "comparison_report.txt")
	return ac.saveComparisonAsText(result, textFile)
}

// compareMetrics compares allocation metrics between baseline and current
func (ac *AnalysisComparator) compareMetrics() (*MetricsComparison, error) {
	// Load baseline metrics
	baselineMetrics, err := ac.loadMetrics(ac.config.BaselineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load baseline metrics: %w", err)
	}

	// Load current metrics
	currentMetrics, err := ac.loadMetrics(ac.config.CurrentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load current metrics: %w", err)
	}

	// Calculate summaries
	baselineSummary := ac.calculateMetricsSummary(baselineMetrics)
	currentSummary := ac.calculateMetricsSummary(currentMetrics)

	// Calculate changes
	comparison := &MetricsComparison{
		BaselineMetrics: baselineSummary,
		CurrentMetrics:  currentSummary,
	}

	if baselineSummary.AvgAllocationRate > 0 {
		comparison.AllocationRateChange = (currentSummary.AvgAllocationRate - baselineSummary.AvgAllocationRate) / baselineSummary.AvgAllocationRate * 100
	}

	if baselineSummary.AvgHeapSize > 0 {
		comparison.HeapSizeChange = (float64(currentSummary.AvgHeapSize) - float64(baselineSummary.AvgHeapSize)) / float64(baselineSummary.AvgHeapSize) * 100
	}

	if baselineSummary.GCFrequency > 0 {
		comparison.GCFrequencyChange = (currentSummary.GCFrequency - baselineSummary.GCFrequency) / baselineSummary.GCFrequency * 100
	}

	if baselineSummary.AvgObjectCount > 0 {
		comparison.ObjectCountChange = (float64(currentSummary.AvgObjectCount) - float64(baselineSummary.AvgObjectCount)) / float64(baselineSummary.AvgObjectCount) * 100
	}

	return comparison, nil
}

// comparePatterns compares detected patterns between baseline and current
func (ac *AnalysisComparator) comparePatterns() (*PatternsComparison, error) {
	// Load baseline patterns
	baselinePatterns, err := ac.loadPatterns(ac.config.BaselineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load baseline patterns: %w", err)
	}

	// Load current patterns
	currentPatterns, err := ac.loadPatterns(ac.config.CurrentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load current patterns: %w", err)
	}

	// Group patterns by type
	baselineGroups := ac.groupPatternsByType(baselinePatterns)
	currentGroups := ac.groupPatternsByType(currentPatterns)

	comparison := &PatternsComparison{
		PatternChanges: make(map[patterns.PatternType]*PatternChange),
		BaselineCount:  len(baselinePatterns),
		CurrentCount:   len(currentPatterns),
	}

	// Find new and resolved patterns
	for patternType := range currentGroups {
		if _, exists := baselineGroups[patternType]; !exists {
			comparison.NewPatterns = append(comparison.NewPatterns, patternType)
		}
	}

	for patternType := range baselineGroups {
		if _, exists := currentGroups[patternType]; !exists {
			comparison.ResolvedPatterns = append(comparison.ResolvedPatterns, patternType)
		}
	}

	// Compare pattern changes
	for patternType, currentList := range currentGroups {
		if baselineList, exists := baselineGroups[patternType]; exists {
			change := &PatternChange{
				BaselineCount: len(baselineList),
				CurrentCount:  len(currentList),
			}

			if len(baselineList) > 0 {
				change.FrequencyChange = (float64(len(currentList)) - float64(len(baselineList))) / float64(len(baselineList)) * 100
			}

			// Compare average intensity
			baselineIntensity := ac.calculateAverageIntensity(baselineList)
			currentIntensity := ac.calculateAverageIntensity(currentList)
			if baselineIntensity > 0 {
				change.IntensityChange = (currentIntensity - baselineIntensity) / baselineIntensity * 100
			}

			// Determine severity change
			baselineSeverity := ac.calculateAverageSeverity(baselineList)
			currentSeverity := ac.calculateAverageSeverity(currentList)
			change.SeverityChange = ac.compareSeverity(baselineSeverity, currentSeverity)

			comparison.PatternChanges[patternType] = change
		}
	}

	return comparison, nil
}

// compareHotspots compares hotspot analysis between baseline and current
func (ac *AnalysisComparator) compareHotspots() (*HotspotsComparison, error) {
	// Load baseline hotspots
	baselineHotspots, err := ac.loadHotspots(ac.config.BaselineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load baseline hotspots: %w", err)
	}

	// Load current hotspots
	currentHotspots, err := ac.loadHotspots(ac.config.CurrentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load current hotspots: %w", err)
	}

	// Group hotspots by function
	baselineGroups := ac.groupHotspotsByFunction(baselineHotspots)
	currentGroups := ac.groupHotspotsByFunction(currentHotspots)

	comparison := &HotspotsComparison{
		HotspotChanges: make(map[string]*HotspotChange),
		BaselineCount:  len(baselineHotspots),
		CurrentCount:   len(currentHotspots),
	}

	// Find new and resolved hotspots
	for function := range currentGroups {
		if _, exists := baselineGroups[function]; !exists {
			comparison.NewHotspots = append(comparison.NewHotspots, function)
		}
	}

	for function := range baselineGroups {
		if _, exists := currentGroups[function]; !exists {
			comparison.ResolvedHotspots = append(comparison.ResolvedHotspots, function)
		}
	}

	// Compare hotspot changes
	for function, currentList := range currentGroups {
		if baselineList, exists := baselineGroups[function]; exists {
			change := &HotspotChange{}

			// Calculate average rates
			baselineRate := ac.calculateAverageHotspotRate(baselineList)
			currentRate := ac.calculateAverageHotspotRate(currentList)

			change.BaselineRate = baselineRate
			change.CurrentRate = currentRate

			if baselineRate > 0 {
				change.RateChange = (currentRate - baselineRate) / baselineRate * 100
			}

			// Compare severity
			baselineSeverity := ac.calculateAverageHotspotSeverity(baselineList)
			currentSeverity := ac.calculateAverageHotspotSeverity(currentList)
			change.SeverityChange = ac.compareSeverity(baselineSeverity, currentSeverity)

			comparison.HotspotChanges[function] = change
		}
	}

	return comparison, nil
}

// calculateOverallScore calculates an overall performance score
func (ac *AnalysisComparator) calculateOverallScore(result *ComparisonResult) float64 {
	score := 50.0 // Start with neutral score

	// Metrics impact (40% weight)
	if result.MetricsChange != nil {
		// Allocation rate change (negative is better)
		score -= result.MetricsChange.AllocationRateChange * 0.1
		// Heap size change (negative is better)
		score -= result.MetricsChange.HeapSizeChange * 0.1
		// GC frequency change (negative is better)
		score -= result.MetricsChange.GCFrequencyChange * 0.1
		// Object count change (negative is better)
		score -= result.MetricsChange.ObjectCountChange * 0.1
	}

	// Patterns impact (30% weight)
	if result.PatternsChange != nil {
		// Resolved patterns are good
		score += float64(len(result.PatternsChange.ResolvedPatterns)) * 5
		// New patterns are bad
		score -= float64(len(result.PatternsChange.NewPatterns)) * 3
		// Overall pattern count change
		patternCountChange := result.PatternsChange.CurrentCount - result.PatternsChange.BaselineCount
		score -= float64(patternCountChange) * 2
	}

	// Hotspots impact (30% weight)
	if result.HotspotsChange != nil {
		// Resolved hotspots are good
		score += float64(len(result.HotspotsChange.ResolvedHotspots)) * 5
		// New hotspots are bad
		score -= float64(len(result.HotspotsChange.NewHotspots)) * 3
		// Overall hotspot count change
		hotspotCountChange := result.HotspotsChange.CurrentCount - result.HotspotsChange.BaselineCount
		score -= float64(hotspotCountChange) * 2
	}

	// Clamp score between 0 and 100
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	return score
}

// identifyChanges identifies improvements and regressions
func (ac *AnalysisComparator) identifyChanges(result *ComparisonResult) ([]string, []string) {
	var improvements, regressions []string

	// Analyze metrics changes
	if result.MetricsChange != nil {
		if result.MetricsChange.AllocationRateChange < -10 {
			improvements = append(improvements, fmt.Sprintf("Allocation rate reduced by %.1f%%", -result.MetricsChange.AllocationRateChange))
		} else if result.MetricsChange.AllocationRateChange > 10 {
			regressions = append(regressions, fmt.Sprintf("Allocation rate increased by %.1f%%", result.MetricsChange.AllocationRateChange))
		}

		if result.MetricsChange.HeapSizeChange < -10 {
			improvements = append(improvements, fmt.Sprintf("Heap size reduced by %.1f%%", -result.MetricsChange.HeapSizeChange))
		} else if result.MetricsChange.HeapSizeChange > 10 {
			regressions = append(regressions, fmt.Sprintf("Heap size increased by %.1f%%", result.MetricsChange.HeapSizeChange))
		}

		if result.MetricsChange.GCFrequencyChange < -10 {
			improvements = append(improvements, fmt.Sprintf("GC frequency reduced by %.1f%%", -result.MetricsChange.GCFrequencyChange))
		} else if result.MetricsChange.GCFrequencyChange > 10 {
			regressions = append(regressions, fmt.Sprintf("GC frequency increased by %.1f%%", result.MetricsChange.GCFrequencyChange))
		}
	}

	// Analyze pattern changes
	if result.PatternsChange != nil {
		if len(result.PatternsChange.ResolvedPatterns) > 0 {
			improvements = append(improvements, fmt.Sprintf("Resolved %d allocation patterns", len(result.PatternsChange.ResolvedPatterns)))
		}
		if len(result.PatternsChange.NewPatterns) > 0 {
			regressions = append(regressions, fmt.Sprintf("Detected %d new allocation patterns", len(result.PatternsChange.NewPatterns)))
		}
	}

	// Analyze hotspot changes
	if result.HotspotsChange != nil {
		if len(result.HotspotsChange.ResolvedHotspots) > 0 {
			improvements = append(improvements, fmt.Sprintf("Resolved %d allocation hotspots", len(result.HotspotsChange.ResolvedHotspots)))
		}
		if len(result.HotspotsChange.NewHotspots) > 0 {
			regressions = append(regressions, fmt.Sprintf("Detected %d new allocation hotspots", len(result.HotspotsChange.NewHotspots)))
		}
	}

	return improvements, regressions
}

// generateComparisonRecommendations generates recommendations based on comparison
func (ac *AnalysisComparator) generateComparisonRecommendations(result *ComparisonResult) []ComparisonRecommendation {
	var recommendations []ComparisonRecommendation
	now := time.Now()
	priority := 1

	// Recommendations based on overall score
	if result.OverallScore < 30 {
		rec := ComparisonRecommendation{
			Type:        "critical_regression",
			Severity:    "critical",
			Title:       "Critical Performance Regression Detected",
			Description: fmt.Sprintf("Overall performance score is %.1f, indicating significant regression. Immediate action required.", result.OverallScore),
			Action:      "Review recent changes and implement immediate optimizations",
			Priority:    priority,
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		priority++
	} else if result.OverallScore > 70 {
		rec := ComparisonRecommendation{
			Type:        "performance_improvement",
			Severity:    "low",
			Title:       "Performance Improvement Achieved",
			Description: fmt.Sprintf("Overall performance score is %.1f, indicating good optimization progress.", result.OverallScore),
			Action:      "Continue monitoring and maintain current optimization practices",
			Priority:    priority,
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		priority++
	}

	// Recommendations based on specific regressions
	if len(result.Regressions) > 0 {
		rec := ComparisonRecommendation{
			Type:        "regression_analysis",
			Severity:    "high",
			Title:       "Address Performance Regressions",
			Description: fmt.Sprintf("Detected %d performance regressions that need attention.", len(result.Regressions)),
			Action:      "Investigate and fix identified regressions",
			Priority:    priority,
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		priority++
	}

	// Recommendations based on new patterns/hotspots
	if result.PatternsChange != nil && len(result.PatternsChange.NewPatterns) > 0 {
		rec := ComparisonRecommendation{
			Type:        "new_patterns",
			Severity:    "medium",
			Title:       "Investigate New Allocation Patterns",
			Description: fmt.Sprintf("Detected %d new allocation patterns that require analysis.", len(result.PatternsChange.NewPatterns)),
			Action:      "Analyze new patterns and implement appropriate optimizations",
			Priority:    priority,
			CreatedAt:   now,
		}
		recommendations = append(recommendations, rec)
		priority++
	}

	return recommendations
}

// Helper functions

// loadMetrics loads metrics from directory
func (ac *AnalysisComparator) loadMetrics(dir string) ([]analyzer.AllocationMetrics, error) {
	filename := filepath.Join(dir, "metrics.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var metrics []analyzer.AllocationMetrics
	return metrics, json.Unmarshal(data, &metrics)
}

// loadPatterns loads patterns from directory
func (ac *AnalysisComparator) loadPatterns(dir string) ([]patterns.DetectedPattern, error) {
	patternsDir := filepath.Join(dir, "patterns")
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

// loadHotspots loads hotspots from directory
func (ac *AnalysisComparator) loadHotspots(dir string) ([]*analyzer.HotspotInfo, error) {
	filename := filepath.Join(dir, "hotspots.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var hotspots []*analyzer.HotspotInfo
	return hotspots, json.Unmarshal(data, &hotspots)
}

// calculateMetricsSummary calculates summary statistics for metrics
func (ac *AnalysisComparator) calculateMetricsSummary(metrics []analyzer.AllocationMetrics) *MetricsSummary {
	if len(metrics) == 0 {
		return &MetricsSummary{}
	}

	summary := &MetricsSummary{}
	totalAllocationRate := 0.0
	totalHeapSize := uint64(0)
	totalObjectCount := uint64(0)

	for _, m := range metrics {
		totalAllocationRate += float64(m.BytesPerSecond)
		totalHeapSize += m.HeapSize
		totalObjectCount += m.ObjectsPerSecond
	}

	summary.AvgAllocationRate = totalAllocationRate / float64(len(metrics))
	summary.AvgHeapSize = totalHeapSize / uint64(len(metrics))
	summary.AvgObjectCount = totalObjectCount / uint64(len(metrics))

	// Calculate GC frequency
	if len(metrics) > 1 {
		first := metrics[0]
		last := metrics[len(metrics)-1]
		gcDelta := last.NumGC - first.NumGC
		timeDelta := last.Timestamp.Sub(first.Timestamp).Seconds()
		if timeDelta > 0 {
			summary.GCFrequency = float64(gcDelta) / timeDelta
		}
	}

	return summary
}

// groupPatternsByType groups patterns by their type
func (ac *AnalysisComparator) groupPatternsByType(detectedPatterns []patterns.DetectedPattern) map[patterns.PatternType][]patterns.DetectedPattern {
	groups := make(map[patterns.PatternType][]patterns.DetectedPattern)
	for _, pattern := range detectedPatterns {
		groups[pattern.Type] = append(groups[pattern.Type], pattern)
	}
	return groups
}

// groupHotspotsByFunction groups hotspots by function
func (ac *AnalysisComparator) groupHotspotsByFunction(hotspots []*analyzer.HotspotInfo) map[string][]*analyzer.HotspotInfo {
	groups := make(map[string][]*analyzer.HotspotInfo)
	for _, hotspot := range hotspots {
		groups[hotspot.Function] = append(groups[hotspot.Function], hotspot)
	}
	return groups
}

// calculateAverageIntensity calculates average intensity for patterns
func (ac *AnalysisComparator) calculateAverageIntensity(detectedPatterns []patterns.DetectedPattern) float64 {
	if len(detectedPatterns) == 0 {
		return 0
	}

	total := 0.0
	for _, pattern := range detectedPatterns {
		switch pattern.Type {
		case patterns.PatternBurst:
			total += pattern.Metrics.BurstIntensity
		case patterns.PatternSpike:
			total += pattern.Metrics.BurstIntensity // Use BurstIntensity for spikes too
		default:
			total += pattern.Metrics.Volatility
		}
	}

	return total / float64(len(detectedPatterns))
}

// calculateAverageSeverity calculates average severity for patterns
func (ac *AnalysisComparator) calculateAverageSeverity(detectedPatterns []patterns.DetectedPattern) string {
	if len(detectedPatterns) == 0 {
		return "low"
	}

	severityScores := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	totalScore := 0
	for _, pattern := range detectedPatterns {
		totalScore += severityScores[pattern.Severity]
	}

	avgScore := float64(totalScore) / float64(len(detectedPatterns))
	if avgScore >= 3.5 {
		return "critical"
	} else if avgScore >= 2.5 {
		return "high"
	} else if avgScore >= 1.5 {
		return "medium"
	}
	return "low"
}

// calculateAverageHotspotRate calculates average allocation rate for hotspots
func (ac *AnalysisComparator) calculateAverageHotspotRate(hotspots []*analyzer.HotspotInfo) float64 {
	if len(hotspots) == 0 {
		return 0
	}

	total := 0.0
	for _, hotspot := range hotspots {
		total += hotspot.AllocationRate
	}

	return total / float64(len(hotspots))
}

// calculateAverageHotspotSeverity calculates average severity for hotspots
func (ac *AnalysisComparator) calculateAverageHotspotSeverity(hotspots []*analyzer.HotspotInfo) string {
	if len(hotspots) == 0 {
		return "low"
	}

	severityScores := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	totalScore := 0
	for _, hotspot := range hotspots {
		totalScore += severityScores[hotspot.Severity]
	}

	avgScore := float64(totalScore) / float64(len(hotspots))
	if avgScore >= 3.5 {
		return "critical"
	} else if avgScore >= 2.5 {
		return "high"
	} else if avgScore >= 1.5 {
		return "medium"
	}
	return "low"
}

// compareSeverity compares two severity levels
func (ac *AnalysisComparator) compareSeverity(baseline, current string) string {
	severityScores := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	baselineScore := severityScores[baseline]
	currentScore := severityScores[current]

	if currentScore > baselineScore {
		return "increased"
	} else if currentScore < baselineScore {
		return "decreased"
	}
	return "unchanged"
}

// saveComparisonAsText saves comparison results in text format
func (ac *AnalysisComparator) saveComparisonAsText(result *ComparisonResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Analysis Comparison Report\n")
	fmt.Fprintf(file, "=========================\n\n")
	fmt.Fprintf(file, "Generated: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(file, "Baseline: %s\n", result.BaselineDir)
	fmt.Fprintf(file, "Current: %s\n", result.CurrentDir)
	fmt.Fprintf(file, "Overall Score: %.1f/100\n\n", result.OverallScore)

	// Metrics comparison
	if result.MetricsChange != nil {
		fmt.Fprintf(file, "Metrics Comparison\n")
		fmt.Fprintf(file, "------------------\n")
		fmt.Fprintf(file, "Allocation Rate Change: %.1f%%\n", result.MetricsChange.AllocationRateChange)
		fmt.Fprintf(file, "Heap Size Change: %.1f%%\n", result.MetricsChange.HeapSizeChange)
		fmt.Fprintf(file, "GC Frequency Change: %.1f%%\n", result.MetricsChange.GCFrequencyChange)
		fmt.Fprintf(file, "Object Count Change: %.1f%%\n\n", result.MetricsChange.ObjectCountChange)
	}

	// Patterns comparison
	if result.PatternsChange != nil {
		fmt.Fprintf(file, "Patterns Comparison\n")
		fmt.Fprintf(file, "-------------------\n")
		fmt.Fprintf(file, "New Patterns: %d\n", len(result.PatternsChange.NewPatterns))
		fmt.Fprintf(file, "Resolved Patterns: %d\n", len(result.PatternsChange.ResolvedPatterns))
		fmt.Fprintf(file, "Pattern Count Change: %d -> %d\n\n", result.PatternsChange.BaselineCount, result.PatternsChange.CurrentCount)
	}

	// Hotspots comparison
	if result.HotspotsChange != nil {
		fmt.Fprintf(file, "Hotspots Comparison\n")
		fmt.Fprintf(file, "-------------------\n")
		fmt.Fprintf(file, "New Hotspots: %d\n", len(result.HotspotsChange.NewHotspots))
		fmt.Fprintf(file, "Resolved Hotspots: %d\n", len(result.HotspotsChange.ResolvedHotspots))
		fmt.Fprintf(file, "Hotspot Count Change: %d -> %d\n\n", result.HotspotsChange.BaselineCount, result.HotspotsChange.CurrentCount)
	}

	// Improvements and regressions
	if len(result.Improvements) > 0 {
		fmt.Fprintf(file, "Improvements\n")
		fmt.Fprintf(file, "------------\n")
		for i, improvement := range result.Improvements {
			fmt.Fprintf(file, "%d. %s\n", i+1, improvement)
		}
		fmt.Fprintf(file, "\n")
	}

	if len(result.Regressions) > 0 {
		fmt.Fprintf(file, "Regressions\n")
		fmt.Fprintf(file, "-----------\n")
		for i, regression := range result.Regressions {
			fmt.Fprintf(file, "%d. %s\n", i+1, regression)
		}
		fmt.Fprintf(file, "\n")
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Fprintf(file, "Recommendations\n")
		fmt.Fprintf(file, "---------------\n")
		for i, rec := range result.Recommendations {
			fmt.Fprintf(file, "%d. %s\n", i+1, rec.Title)
			fmt.Fprintf(file, "   Severity: %s\n", rec.Severity)
			fmt.Fprintf(file, "   Description: %s\n", rec.Description)
			fmt.Fprintf(file, "   Action: %s\n\n", rec.Action)
		}
	}

	return nil
}