package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// PerformanceMetrics holds detailed performance statistics
type PerformanceMetrics struct {
	Algorithm       string
	DataSize        int
	Iterations      int
	TotalTime       time.Duration
	AverageTime     time.Duration
	MedianTime      time.Duration
	MinTime         time.Duration
	MaxTime         time.Duration
	StandardDev     time.Duration
	Percentile95    time.Duration
	Percentile99    time.Duration
	Throughput      float64 // Operations per second
	SuccessRate     float64
	MemoryUsage     int64
	Allocations     int64
	CacheHitRate    float64
	BranchMisses    int64
	Instructions    int64
	Cycles          int64
	IPC             float64 // Instructions per cycle
}

// ComparisonResult holds comparison between algorithms
type ComparisonResult struct {
	BaseAlgorithm string
	CompareAlgorithm string
	SpeedupFactor float64
	MemoryRatio float64
	EfficiencyScore float64
	Recommendation string
}

// AnalysisReport holds comprehensive analysis results
type AnalysisReport struct {
	Timestamp time.Time
	Metrics []PerformanceMetrics
	Comparisons []ComparisonResult
	Insights []string
	Recommendations []string
	SystemInfo SystemInfo
}

// SystemInfo holds system information
type SystemInfo struct {
	CPU string
	Cores int
	Memory int64
	CacheL1 int
	CacheL2 int
	CacheL3 int
	OS string
	GoVersion string
}

// Analyzer provides performance analysis capabilities
type Analyzer struct {
	metrics []PerformanceMetrics
	config AnalysisConfig
}

// AnalysisConfig holds configuration for analysis
type AnalysisConfig struct {
	EnableDetailedMetrics bool
	EnableCacheAnalysis bool
	EnableBranchAnalysis bool
	ConfidenceLevel float64
	SignificanceThreshold float64
}

// NewAnalyzer creates a new performance analyzer
func NewAnalyzer(config AnalysisConfig) *Analyzer {
	return &Analyzer{
		metrics: make([]PerformanceMetrics, 0),
		config: config,
	}
}

// AddMetrics adds performance metrics to the analyzer
func (a *Analyzer) AddMetrics(metrics PerformanceMetrics) {
	a.metrics = append(a.metrics, metrics)
}

// CalculateStatistics calculates detailed statistics from raw timing data
func (a *Analyzer) CalculateStatistics(algorithm string, dataSize int, timings []time.Duration) PerformanceMetrics {
	if len(timings) == 0 {
		return PerformanceMetrics{}
	}

	// Sort timings for percentile calculations
	sortedTimings := make([]time.Duration, len(timings))
	copy(sortedTimings, timings)
	sort.Slice(sortedTimings, func(i, j int) bool {
		return sortedTimings[i] < sortedTimings[j]
	})

	// Calculate basic statistics
	var total time.Duration
	min := sortedTimings[0]
	max := sortedTimings[len(sortedTimings)-1]

	for _, t := range timings {
		total += t
	}

	average := total / time.Duration(len(timings))
	median := sortedTimings[len(sortedTimings)/2]

	// Calculate standard deviation
	var variance float64
	for _, t := range timings {
		diff := float64(t - average)
		variance += diff * diff
	}
	variance /= float64(len(timings))
	stdDev := time.Duration(math.Sqrt(variance))

	// Calculate percentiles
	p95Index := int(float64(len(sortedTimings)) * 0.95)
	p99Index := int(float64(len(sortedTimings)) * 0.99)
	if p95Index >= len(sortedTimings) {
		p95Index = len(sortedTimings) - 1
	}
	if p99Index >= len(sortedTimings) {
		p99Index = len(sortedTimings) - 1
	}

	p95 := sortedTimings[p95Index]
	p99 := sortedTimings[p99Index]

	// Calculate throughput (operations per second)
	throughput := float64(time.Second) / float64(average)

	return PerformanceMetrics{
		Algorithm:    algorithm,
		DataSize:     dataSize,
		Iterations:   len(timings),
		TotalTime:    total,
		AverageTime:  average,
		MedianTime:   median,
		MinTime:      min,
		MaxTime:      max,
		StandardDev:  stdDev,
		Percentile95: p95,
		Percentile99: p99,
		Throughput:   throughput,
	}
}

// CompareAlgorithms compares two algorithms and provides insights
func (a *Analyzer) CompareAlgorithms(base, compare string) *ComparisonResult {
	var baseMetrics, compareMetrics *PerformanceMetrics

	// Find metrics for both algorithms
	for i := range a.metrics {
		if a.metrics[i].Algorithm == base {
			baseMetrics = &a.metrics[i]
		}
		if a.metrics[i].Algorithm == compare {
			compareMetrics = &a.metrics[i]
		}
	}

	if baseMetrics == nil || compareMetrics == nil {
		return nil
	}

	// Calculate comparison metrics
	speedupFactor := float64(baseMetrics.AverageTime) / float64(compareMetrics.AverageTime)
	memoryRatio := float64(compareMetrics.MemoryUsage) / float64(baseMetrics.MemoryUsage)

	// Calculate efficiency score (higher is better)
	efficiencyScore := speedupFactor / memoryRatio

	// Generate recommendation
	recommendation := a.generateRecommendation(speedupFactor, memoryRatio, efficiencyScore)

	return &ComparisonResult{
		BaseAlgorithm:    base,
		CompareAlgorithm: compare,
		SpeedupFactor:    speedupFactor,
		MemoryRatio:      memoryRatio,
		EfficiencyScore:  efficiencyScore,
		Recommendation:   recommendation,
	}
}

// generateRecommendation generates a recommendation based on comparison metrics
func (a *Analyzer) generateRecommendation(speedup, memoryRatio, efficiency float64) string {
	if speedup > 2.0 && memoryRatio < 1.5 {
		return "Strongly recommended: Significant performance improvement with reasonable memory usage"
	}
	if speedup > 1.5 && memoryRatio < 2.0 {
		return "Recommended: Good performance improvement with acceptable memory overhead"
	}
	if speedup > 1.2 && memoryRatio < 1.2 {
		return "Consider: Moderate improvement with low memory overhead"
	}
	if speedup < 0.8 {
		return "Not recommended: Performance regression detected"
	}
	if memoryRatio > 3.0 {
		return "Caution: High memory overhead may not justify performance gains"
	}
	return "Neutral: Trade-offs require careful consideration based on specific use case"
}

// AnalyzeComplexity estimates algorithmic complexity from performance data
func (a *Analyzer) AnalyzeComplexity(algorithm string) string {
	var algorithmMetrics []PerformanceMetrics

	// Collect all metrics for the algorithm
	for _, m := range a.metrics {
		if m.Algorithm == algorithm {
			algorithmMetrics = append(algorithmMetrics, m)
		}
	}

	if len(algorithmMetrics) < 2 {
		return "Insufficient data for complexity analysis"
	}

	// Sort by data size
	sort.Slice(algorithmMetrics, func(i, j int) bool {
		return algorithmMetrics[i].DataSize < algorithmMetrics[j].DataSize
	})

	// Analyze growth pattern
	growthRatios := make([]float64, 0)
	for i := 1; i < len(algorithmMetrics); i++ {
		sizeRatio := float64(algorithmMetrics[i].DataSize) / float64(algorithmMetrics[i-1].DataSize)
		timeRatio := float64(algorithmMetrics[i].AverageTime) / float64(algorithmMetrics[i-1].AverageTime)
		growthRatio := timeRatio / sizeRatio
		growthRatios = append(growthRatios, growthRatio)
	}

	// Calculate average growth ratio
	var avgGrowth float64
	for _, ratio := range growthRatios {
		avgGrowth += ratio
	}
	avgGrowth /= float64(len(growthRatios))

	// Estimate complexity
	if avgGrowth < 1.2 {
		return "O(1) - Constant time"
	} else if avgGrowth < 2.0 {
		return "O(log n) - Logarithmic time"
	} else if avgGrowth < 3.0 {
		return "O(n) - Linear time"
	} else if avgGrowth < 5.0 {
		return "O(n log n) - Linearithmic time"
	} else {
		return "O(n²) or worse - Quadratic or higher"
	}
}

// GenerateInsights generates performance insights from collected metrics
func (a *Analyzer) GenerateInsights() []string {
	insights := make([]string, 0)

	if len(a.metrics) == 0 {
		return []string{"No metrics available for analysis"}
	}

	// Find best and worst performing algorithms
	var fastest, slowest *PerformanceMetrics
	for i := range a.metrics {
		m := &a.metrics[i]
		if fastest == nil || m.AverageTime < fastest.AverageTime {
			fastest = m
		}
		if slowest == nil || m.AverageTime > slowest.AverageTime {
			slowest = m
		}
	}

	if fastest != nil && slowest != nil {
		speedup := float64(slowest.AverageTime) / float64(fastest.AverageTime)
		insights = append(insights, fmt.Sprintf(
			"Performance range: %s is %.2fx faster than %s",
			fastest.Algorithm, speedup, slowest.Algorithm))
	}

	// Analyze memory efficiency
	var mostMemoryEfficient, leastMemoryEfficient *PerformanceMetrics
	for i := range a.metrics {
		m := &a.metrics[i]
		if m.MemoryUsage > 0 {
			if mostMemoryEfficient == nil || m.MemoryUsage < mostMemoryEfficient.MemoryUsage {
				mostMemoryEfficient = m
			}
			if leastMemoryEfficient == nil || m.MemoryUsage > leastMemoryEfficient.MemoryUsage {
				leastMemoryEfficient = m
			}
		}
	}

	if mostMemoryEfficient != nil && leastMemoryEfficient != nil {
		memoryRatio := float64(leastMemoryEfficient.MemoryUsage) / float64(mostMemoryEfficient.MemoryUsage)
		insights = append(insights, fmt.Sprintf(
			"Memory efficiency: %s uses %.2fx less memory than %s",
			mostMemoryEfficient.Algorithm, memoryRatio, leastMemoryEfficient.Algorithm))
	}

	// Analyze consistency (standard deviation)
	var mostConsistent, leastConsistent *PerformanceMetrics
	for i := range a.metrics {
		m := &a.metrics[i]
		if mostConsistent == nil || m.StandardDev < mostConsistent.StandardDev {
			mostConsistent = m
		}
		if leastConsistent == nil || m.StandardDev > leastConsistent.StandardDev {
			leastConsistent = m
		}
	}

	if mostConsistent != nil {
		insights = append(insights, fmt.Sprintf(
			"Most consistent performance: %s (σ = %v)",
			mostConsistent.Algorithm, mostConsistent.StandardDev))
	}

	// Analyze throughput
	var highestThroughput *PerformanceMetrics
	for i := range a.metrics {
		m := &a.metrics[i]
		if highestThroughput == nil || m.Throughput > highestThroughput.Throughput {
			highestThroughput = m
		}
	}

	if highestThroughput != nil {
		insights = append(insights, fmt.Sprintf(
			"Highest throughput: %s (%.0f ops/sec)",
			highestThroughput.Algorithm, highestThroughput.Throughput))
	}

	return insights
}

// GenerateRecommendations generates actionable recommendations
func (a *Analyzer) GenerateRecommendations() []string {
	recommendations := make([]string, 0)

	if len(a.metrics) == 0 {
		return []string{"No metrics available for recommendations"}
	}

	// Group metrics by data size for size-specific recommendations
	sizeGroups := make(map[int][]PerformanceMetrics)
	for _, m := range a.metrics {
		sizeGroups[m.DataSize] = append(sizeGroups[m.DataSize], m)
	}

	// Generate size-specific recommendations
	for size, metrics := range sizeGroups {
		if len(metrics) > 1 {
			// Find best algorithm for this size
			var best *PerformanceMetrics
			for i := range metrics {
				m := &metrics[i]
				if best == nil || m.AverageTime < best.AverageTime {
					best = m
				}
			}

			if best != nil {
				recommendations = append(recommendations, fmt.Sprintf(
					"For data size %d: Use %s for optimal performance",
					size, best.Algorithm))
			}
		}
	}

	// General recommendations based on algorithm characteristics
	recommendations = append(recommendations, a.generateGeneralRecommendations()...)

	return recommendations
}

// generateGeneralRecommendations generates general performance recommendations
func (a *Analyzer) generateGeneralRecommendations() []string {
	recommendations := []string{
		"For small datasets (< 1000 elements): Consider linear search for simplicity",
		"For large sorted datasets: Binary search variants are typically optimal",
		"For frequent lookups: Hash tables provide O(1) average case performance",
		"For string searches: Tries are efficient for prefix-based operations",
		"For memory-constrained environments: Prefer algorithms with lower memory overhead",
		"For real-time systems: Choose algorithms with consistent performance (low standard deviation)",
		"For cache-sensitive applications: Consider cache-friendly algorithm variants",
	}

	return recommendations
}

// GenerateReport creates a comprehensive analysis report
func (a *Analyzer) GenerateReport() *AnalysisReport {
	report := &AnalysisReport{
		Timestamp:       time.Now(),
		Metrics:         a.metrics,
		Comparisons:     make([]ComparisonResult, 0),
		Insights:        a.GenerateInsights(),
		Recommendations: a.GenerateRecommendations(),
		SystemInfo:      a.getSystemInfo(),
	}

	// Generate all pairwise comparisons
	algorithms := make(map[string]bool)
	for _, m := range a.metrics {
		algorithms[m.Algorithm] = true
	}

	algorithmList := make([]string, 0, len(algorithms))
	for alg := range algorithms {
		algorithmList = append(algorithmList, alg)
	}

	for i := 0; i < len(algorithmList); i++ {
		for j := i + 1; j < len(algorithmList); j++ {
			comparison := a.CompareAlgorithms(algorithmList[i], algorithmList[j])
			if comparison != nil {
				report.Comparisons = append(report.Comparisons, *comparison)
			}
		}
	}

	return report
}

// getSystemInfo collects system information for the report
func (a *Analyzer) getSystemInfo() SystemInfo {
	// This would typically collect real system information
	// For now, return placeholder data
	return SystemInfo{
		CPU:       "Unknown CPU",
		Cores:     4,
		Memory:    8 * 1024 * 1024 * 1024, // 8GB
		CacheL1:   32 * 1024,              // 32KB
		CacheL2:   256 * 1024,             // 256KB
		CacheL3:   8 * 1024 * 1024,        // 8MB
		OS:        "Unknown OS",
		GoVersion: "go1.21",
	}
}

// StatisticalSignificance tests if performance differences are statistically significant
func (a *Analyzer) StatisticalSignificance(alg1, alg2 string, timings1, timings2 []time.Duration) (bool, float64) {
	if len(timings1) < 30 || len(timings2) < 30 {
		return false, 0.0 // Insufficient sample size
	}

	// Convert to float64 for calculations
	samples1 := make([]float64, len(timings1))
	samples2 := make([]float64, len(timings2))

	for i, t := range timings1 {
		samples1[i] = float64(t)
	}
	for i, t := range timings2 {
		samples2[i] = float64(t)
	}

	// Calculate means and standard deviations
	mean1 := calculateMean(samples1)
	mean2 := calculateMean(samples2)
	std1 := calculateStdDev(samples1, mean1)
	std2 := calculateStdDev(samples2, mean2)

	// Perform Welch's t-test
	n1, n2 := float64(len(samples1)), float64(len(samples2))
	pooledStdErr := math.Sqrt((std1*std1)/n1 + (std2*std2)/n2)
	tStatistic := (mean1 - mean2) / pooledStdErr

	// Degrees of freedom (Welch-Satterthwaite equation)
	df := math.Pow((std1*std1)/n1+(std2*std2)/n2, 2) /
		(math.Pow((std1*std1)/n1, 2)/(n1-1) + math.Pow((std2*std2)/n2, 2)/(n2-1))

	// For simplicity, use critical value for 95% confidence (approximately 1.96 for large samples)
	criticalValue := 1.96
	if df < 30 {
		criticalValue = 2.045 // Approximate t-value for small samples
	}

	isSignificant := math.Abs(tStatistic) > criticalValue
	pValue := 2 * (1 - normalCDF(math.Abs(tStatistic))) // Approximate p-value

	return isSignificant, pValue
}

// Helper functions for statistical calculations
func calculateMean(samples []float64) float64 {
	var sum float64
	for _, v := range samples {
		sum += v
	}
	return sum / float64(len(samples))
}

func calculateStdDev(samples []float64, mean float64) float64 {
	var variance float64
	for _, v := range samples {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(samples) - 1) // Sample standard deviation
	return math.Sqrt(variance)
}

// Approximate normal CDF for p-value calculation
func normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt(2)))
}