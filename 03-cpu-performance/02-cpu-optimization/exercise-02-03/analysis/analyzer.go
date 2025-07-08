package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// AlgorithmResult represents the performance result of a sorting algorithm
type AlgorithmResult struct {
	Name           string
	DataPattern    string
	DataSize       int
	ExecutionTime  time.Duration
	Comparisons    int64
	Swaps          int64
	MemoryUsage    int64 // bytes
	CacheHitRate   float64
	BranchMisses   int64
	Throughput     float64 // elements per second
}

// PerformanceAnalyzer analyzes sorting algorithm performance
type PerformanceAnalyzer struct {
	Results []AlgorithmResult
}

// NewPerformanceAnalyzer creates a new performance analyzer
func NewPerformanceAnalyzer() *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		Results: make([]AlgorithmResult, 0),
	}
}

// AddResult adds a performance result to the analyzer
func (pa *PerformanceAnalyzer) AddResult(result AlgorithmResult) {
	pa.Results = append(pa.Results, result)
}

// AnalyzeByDataSize analyzes performance by data size
func (pa *PerformanceAnalyzer) AnalyzeByDataSize() map[string][]AlgorithmResult {
	analysis := make(map[string][]AlgorithmResult)
	
	for _, result := range pa.Results {
		key := fmt.Sprintf("%s_%s", result.Name, result.DataPattern)
		analysis[key] = append(analysis[key], result)
	}
	
	// Sort results by data size for each algorithm
	for key := range analysis {
		sort.Slice(analysis[key], func(i, j int) bool {
			return analysis[key][i].DataSize < analysis[key][j].DataSize
		})
	}
	
	return analysis
}

// AnalyzeByAlgorithm analyzes performance by algorithm
func (pa *PerformanceAnalyzer) AnalyzeByAlgorithm() map[string][]AlgorithmResult {
	analysis := make(map[string][]AlgorithmResult)
	
	for _, result := range pa.Results {
		analysis[result.Name] = append(analysis[result.Name], result)
	}
	
	return analysis
}

// AnalyzeByDataPattern analyzes performance by data pattern
func (pa *PerformanceAnalyzer) AnalyzeByDataPattern() map[string][]AlgorithmResult {
	analysis := make(map[string][]AlgorithmResult)
	
	for _, result := range pa.Results {
		analysis[result.DataPattern] = append(analysis[result.DataPattern], result)
	}
	
	return analysis
}

// FindBestAlgorithm finds the best algorithm for each data pattern and size
func (pa *PerformanceAnalyzer) FindBestAlgorithm() map[string]AlgorithmResult {
	best := make(map[string]AlgorithmResult)
	
	patternSizeGroups := make(map[string][]AlgorithmResult)
	
	// Group by pattern and size
	for _, result := range pa.Results {
		key := fmt.Sprintf("%s_%d", result.DataPattern, result.DataSize)
		patternSizeGroups[key] = append(patternSizeGroups[key], result)
	}
	
	// Find best for each group
	for key, results := range patternSizeGroups {
		if len(results) == 0 {
			continue
		}
		
		bestResult := results[0]
		for _, result := range results[1:] {
			if result.ExecutionTime < bestResult.ExecutionTime {
				bestResult = result
			}
		}
		best[key] = bestResult
	}
	
	return best
}

// CalculateComplexityTrend calculates the time complexity trend for an algorithm
func (pa *PerformanceAnalyzer) CalculateComplexityTrend(algorithmName, dataPattern string) ComplexityTrend {
	var results []AlgorithmResult
	
	// Filter results for specific algorithm and pattern
	for _, result := range pa.Results {
		if result.Name == algorithmName && result.DataPattern == dataPattern {
			results = append(results, result)
		}
	}
	
	if len(results) < 2 {
		return ComplexityTrend{}
	}
	
	// Sort by data size
	sort.Slice(results, func(i, j int) bool {
		return results[i].DataSize < results[j].DataSize
	})
	
	// Calculate growth rates
	growthRates := make([]float64, 0, len(results)-1)
	for i := 1; i < len(results); i++ {
		prev := results[i-1]
		curr := results[i]
		
		sizeRatio := float64(curr.DataSize) / float64(prev.DataSize)
		timeRatio := float64(curr.ExecutionTime) / float64(prev.ExecutionTime)
		
		if sizeRatio > 1 {
			growthRate := math.Log(timeRatio) / math.Log(sizeRatio)
			growthRates = append(growthRates, growthRate)
		}
	}
	
	// Calculate average growth rate
	avgGrowthRate := 0.0
	if len(growthRates) > 0 {
		sum := 0.0
		for _, rate := range growthRates {
			sum += rate
		}
		avgGrowthRate = sum / float64(len(growthRates))
	}
	
	// Determine complexity class
	complexityClass := DetermineComplexityClass(avgGrowthRate)
	
	return ComplexityTrend{
		Algorithm:       algorithmName,
		DataPattern:     dataPattern,
		GrowthRate:      avgGrowthRate,
		ComplexityClass: complexityClass,
		DataPoints:      len(results),
		Confidence:      CalculateConfidence(growthRates),
	}
}

// ComplexityTrend represents the complexity analysis of an algorithm
type ComplexityTrend struct {
	Algorithm       string
	DataPattern     string
	GrowthRate      float64
	ComplexityClass string
	DataPoints      int
	Confidence      float64
}

// DetermineComplexityClass determines the complexity class based on growth rate
func DetermineComplexityClass(growthRate float64) string {
	switch {
	case growthRate < 0.5:
		return "O(1) - Constant"
	case growthRate < 1.2:
		return "O(log n) - Logarithmic"
	case growthRate < 1.5:
		return "O(n log n) - Linearithmic"
	case growthRate < 1.8:
		return "O(n) - Linear"
	case growthRate < 2.2:
		return "O(n²) - Quadratic"
	case growthRate < 3.2:
		return "O(n³) - Cubic"
	default:
		return "O(n^k) - Polynomial or worse"
	}
}

// CalculateConfidence calculates confidence in the complexity analysis
func CalculateConfidence(growthRates []float64) float64 {
	if len(growthRates) < 2 {
		return 0.0
	}
	
	// Calculate standard deviation
	mean := 0.0
	for _, rate := range growthRates {
		mean += rate
	}
	mean /= float64(len(growthRates))
	
	variance := 0.0
	for _, rate := range growthRates {
		diff := rate - mean
		variance += diff * diff
	}
	variance /= float64(len(growthRates))
	
	stdDev := math.Sqrt(variance)
	
	// Confidence is inversely related to standard deviation
	// Lower std dev = higher confidence
	confidence := 1.0 / (1.0 + stdDev)
	return confidence
}

// GeneratePerformanceReport generates a comprehensive performance report
func (pa *PerformanceAnalyzer) GeneratePerformanceReport() PerformanceReport {
	report := PerformanceReport{
		TotalResults: len(pa.Results),
		Algorithms:   make(map[string]int),
		DataPatterns: make(map[string]int),
		DataSizes:    make(map[int]int),
	}
	
	// Count occurrences
	for _, result := range pa.Results {
		report.Algorithms[result.Name]++
		report.DataPatterns[result.DataPattern]++
		report.DataSizes[result.DataSize]++
	}
	
	// Find best and worst performers
	report.BestPerformers = pa.FindBestAlgorithm()
	report.WorstPerformers = pa.findWorstAlgorithm()
	
	// Calculate complexity trends
	report.ComplexityTrends = make([]ComplexityTrend, 0)
	for algorithm := range report.Algorithms {
		for pattern := range report.DataPatterns {
			trend := pa.CalculateComplexityTrend(algorithm, pattern)
			if trend.DataPoints > 1 {
				report.ComplexityTrends = append(report.ComplexityTrends, trend)
			}
		}
	}
	
	return report
}

// findWorstAlgorithm finds the worst performing algorithm for each scenario
func (pa *PerformanceAnalyzer) findWorstAlgorithm() map[string]AlgorithmResult {
	worst := make(map[string]AlgorithmResult)
	
	patternSizeGroups := make(map[string][]AlgorithmResult)
	
	// Group by pattern and size
	for _, result := range pa.Results {
		key := fmt.Sprintf("%s_%d", result.DataPattern, result.DataSize)
		patternSizeGroups[key] = append(patternSizeGroups[key], result)
	}
	
	// Find worst for each group
	for key, results := range patternSizeGroups {
		if len(results) == 0 {
			continue
		}
		
		worstResult := results[0]
		for _, result := range results[1:] {
			if result.ExecutionTime > worstResult.ExecutionTime {
				worstResult = result
			}
		}
		worst[key] = worstResult
	}
	
	return worst
}

// PerformanceReport represents a comprehensive performance analysis report
type PerformanceReport struct {
	TotalResults      int
	Algorithms        map[string]int
	DataPatterns      map[string]int
	DataSizes         map[int]int
	BestPerformers    map[string]AlgorithmResult
	WorstPerformers   map[string]AlgorithmResult
	ComplexityTrends  []ComplexityTrend
}

// CalculateSpeedup calculates speedup between two algorithms
func CalculateSpeedup(baseline, optimized AlgorithmResult) float64 {
	if optimized.ExecutionTime == 0 {
		return 0
	}
	return float64(baseline.ExecutionTime) / float64(optimized.ExecutionTime)
}

// CalculateEfficiency calculates efficiency (throughput per unit time)
func CalculateEfficiency(result AlgorithmResult) float64 {
	if result.ExecutionTime == 0 {
		return 0
	}
	return float64(result.DataSize) / result.ExecutionTime.Seconds()
}

// CompareAlgorithms compares multiple algorithms on the same dataset
func CompareAlgorithms(results []AlgorithmResult) AlgorithmComparison {
	if len(results) == 0 {
		return AlgorithmComparison{}
	}
	
	// Sort by execution time
	sort.Slice(results, func(i, j int) bool {
		return results[i].ExecutionTime < results[j].ExecutionTime
	})
	
	fastest := results[0]
	slowest := results[len(results)-1]
	
	// Calculate relative performance
	relativePerf := make(map[string]float64)
	for _, result := range results {
		relativePerf[result.Name] = float64(fastest.ExecutionTime) / float64(result.ExecutionTime)
	}
	
	return AlgorithmComparison{
		Fastest:             fastest,
		Slowest:             slowest,
		SpeedupRange:        float64(slowest.ExecutionTime) / float64(fastest.ExecutionTime),
		RelativePerformance: relativePerf,
		TotalAlgorithms:     len(results),
	}
}

// AlgorithmComparison represents a comparison between multiple algorithms
type AlgorithmComparison struct {
	Fastest             AlgorithmResult
	Slowest             AlgorithmResult
	SpeedupRange        float64
	RelativePerformance map[string]float64
	TotalAlgorithms     int
}