package analysis

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// ReportGenerator generates various types of performance reports
type ReportGenerator struct {
	analyzer *PerformanceAnalyzer
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(analyzer *PerformanceAnalyzer) *ReportGenerator {
	return &ReportGenerator{
		analyzer: analyzer,
	}
}

// GenerateTextReport generates a comprehensive text report
func (rg *ReportGenerator) GenerateTextReport(writer io.Writer) error {
	report := rg.analyzer.GeneratePerformanceReport()
	
	fmt.Fprintf(writer, "\n=== SORTING ALGORITHM PERFORMANCE ANALYSIS ===\n\n")
	fmt.Fprintf(writer, "Total Test Results: %d\n", report.TotalResults)
	fmt.Fprintf(writer, "Algorithms Tested: %d\n", len(report.Algorithms))
	fmt.Fprintf(writer, "Data Patterns: %d\n", len(report.DataPatterns))
	fmt.Fprintf(writer, "Data Sizes: %d\n\n", len(report.DataSizes))
	
	// Algorithm summary
	fmt.Fprintf(writer, "=== ALGORITHM SUMMARY ===\n")
	for algorithm, count := range report.Algorithms {
		fmt.Fprintf(writer, "%-15s: %d tests\n", algorithm, count)
	}
	fmt.Fprintf(writer, "\n")
	
	// Data pattern summary
	fmt.Fprintf(writer, "=== DATA PATTERN SUMMARY ===\n")
	for pattern, count := range report.DataPatterns {
		fmt.Fprintf(writer, "%-20s: %d tests\n", pattern, count)
	}
	fmt.Fprintf(writer, "\n")
	
	// Best performers
	fmt.Fprintf(writer, "=== BEST PERFORMERS BY SCENARIO ===\n")
	for scenario, result := range report.BestPerformers {
		fmt.Fprintf(writer, "Scenario: %s\n", scenario)
		fmt.Fprintf(writer, "  Winner: %s\n", result.Name)
		fmt.Fprintf(writer, "  Time: %v\n", result.ExecutionTime)
		fmt.Fprintf(writer, "  Throughput: %.2f elements/sec\n\n", result.Throughput)
	}
	
	// Complexity analysis
	fmt.Fprintf(writer, "=== COMPLEXITY ANALYSIS ===\n")
	for _, trend := range report.ComplexityTrends {
		fmt.Fprintf(writer, "Algorithm: %s, Pattern: %s\n", trend.Algorithm, trend.DataPattern)
		fmt.Fprintf(writer, "  Growth Rate: %.2f\n", trend.GrowthRate)
		fmt.Fprintf(writer, "  Complexity: %s\n", trend.ComplexityClass)
		fmt.Fprintf(writer, "  Confidence: %.2f%%\n", trend.Confidence*100)
		fmt.Fprintf(writer, "  Data Points: %d\n\n", trend.DataPoints)
	}
	
	return nil
}

// GenerateCSVReport generates a CSV report of all results
func (rg *ReportGenerator) GenerateCSVReport(writer io.Writer) error {
	// CSV header
	fmt.Fprintf(writer, "Algorithm,DataPattern,DataSize,ExecutionTime(ns),Comparisons,Swaps,MemoryUsage(bytes),CacheHitRate,BranchMisses,Throughput\n")
	
	// Sort results for consistent output
	results := make([]AlgorithmResult, len(rg.analyzer.Results))
	copy(results, rg.analyzer.Results)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		if results[i].DataPattern != results[j].DataPattern {
			return results[i].DataPattern < results[j].DataPattern
		}
		return results[i].DataSize < results[j].DataSize
	})
	
	// Write data rows
	for _, result := range results {
		fmt.Fprintf(writer, "%s,%s,%d,%d,%d,%d,%d,%.4f,%d,%.2f\n",
			result.Name,
			result.DataPattern,
			result.DataSize,
			result.ExecutionTime.Nanoseconds(),
			result.Comparisons,
			result.Swaps,
			result.MemoryUsage,
			result.CacheHitRate,
			result.BranchMisses,
			result.Throughput,
		)
	}
	
	return nil
}

// GenerateMarkdownReport generates a markdown report
func (rg *ReportGenerator) GenerateMarkdownReport(writer io.Writer) error {
	report := rg.analyzer.GeneratePerformanceReport()
	
	fmt.Fprintf(writer, "# Sorting Algorithm Performance Analysis\n\n")
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "- **Total Test Results**: %d\n", report.TotalResults)
	fmt.Fprintf(writer, "- **Algorithms Tested**: %d\n", len(report.Algorithms))
	fmt.Fprintf(writer, "- **Data Patterns**: %d\n", len(report.DataPatterns))
	fmt.Fprintf(writer, "- **Data Sizes**: %d\n\n", len(report.DataSizes))
	
	// Algorithm performance table
	fmt.Fprintf(writer, "## Algorithm Performance Overview\n\n")
	fmt.Fprintf(writer, "| Algorithm | Tests Run | Avg Performance |\n")
	fmt.Fprintf(writer, "|-----------|-----------|----------------|\n")
	
	algorithmStats := rg.calculateAlgorithmStats()
	for algorithm, stats := range algorithmStats {
		fmt.Fprintf(writer, "| %s | %d | %.2f ms |\n", algorithm, stats.TestCount, stats.AvgTime.Seconds()*1000)
	}
	fmt.Fprintf(writer, "\n")
	
	// Best performers by data pattern
	fmt.Fprintf(writer, "## Best Performers by Data Pattern\n\n")
	patternWinners := rg.getBestByPattern()
	for pattern, winners := range patternWinners {
		fmt.Fprintf(writer, "### %s\n\n", strings.Title(pattern))
		fmt.Fprintf(writer, "| Data Size | Best Algorithm | Time | Speedup vs Worst |\n")
		fmt.Fprintf(writer, "|-----------|----------------|------|------------------|\n")
		
		for _, winner := range winners {
			speedup := rg.calculateSpeedupVsWorst(winner)
			fmt.Fprintf(writer, "| %d | %s | %v | %.2fx |\n", 
				winner.DataSize, winner.Name, winner.ExecutionTime, speedup)
		}
		fmt.Fprintf(writer, "\n")
	}
	
	// Complexity analysis
	fmt.Fprintf(writer, "## Complexity Analysis\n\n")
	fmt.Fprintf(writer, "| Algorithm | Data Pattern | Complexity Class | Growth Rate | Confidence |\n")
	fmt.Fprintf(writer, "|-----------|--------------|------------------|-------------|------------|\n")
	
	for _, trend := range report.ComplexityTrends {
		fmt.Fprintf(writer, "| %s | %s | %s | %.2f | %.1f%% |\n",
			trend.Algorithm, trend.DataPattern, trend.ComplexityClass, 
			trend.GrowthRate, trend.Confidence*100)
	}
	fmt.Fprintf(writer, "\n")
	
	// Recommendations
	fmt.Fprintf(writer, "## Recommendations\n\n")
	rg.generateRecommendations(writer, report)
	
	return nil
}

// AlgorithmStats represents statistics for an algorithm
type AlgorithmStats struct {
	TestCount int
	AvgTime   time.Duration
	MinTime   time.Duration
	MaxTime   time.Duration
}

// calculateAlgorithmStats calculates statistics for each algorithm
func (rg *ReportGenerator) calculateAlgorithmStats() map[string]AlgorithmStats {
	stats := make(map[string]AlgorithmStats)
	algorithmResults := make(map[string][]AlgorithmResult)
	
	// Group results by algorithm
	for _, result := range rg.analyzer.Results {
		algorithmResults[result.Name] = append(algorithmResults[result.Name], result)
	}
	
	// Calculate stats for each algorithm
	for algorithm, results := range algorithmResults {
		if len(results) == 0 {
			continue
		}
		
		totalTime := time.Duration(0)
		minTime := results[0].ExecutionTime
		maxTime := results[0].ExecutionTime
		
		for _, result := range results {
			totalTime += result.ExecutionTime
			if result.ExecutionTime < minTime {
				minTime = result.ExecutionTime
			}
			if result.ExecutionTime > maxTime {
				maxTime = result.ExecutionTime
			}
		}
		
		stats[algorithm] = AlgorithmStats{
			TestCount: len(results),
			AvgTime:   totalTime / time.Duration(len(results)),
			MinTime:   minTime,
			MaxTime:   maxTime,
		}
	}
	
	return stats
}

// getBestByPattern gets the best performing algorithm for each pattern and size
func (rg *ReportGenerator) getBestByPattern() map[string][]AlgorithmResult {
	patternResults := make(map[string][]AlgorithmResult)
	
	// Group by pattern
	for _, result := range rg.analyzer.Results {
		patternResults[result.DataPattern] = append(patternResults[result.DataPattern], result)
	}
	
	// Find best for each pattern and size combination
	patternWinners := make(map[string][]AlgorithmResult)
	for pattern, results := range patternResults {
		sizeGroups := make(map[int][]AlgorithmResult)
		
		// Group by size within pattern
		for _, result := range results {
			sizeGroups[result.DataSize] = append(sizeGroups[result.DataSize], result)
		}
		
		// Find best for each size
		for _, sizeResults := range sizeGroups {
			if len(sizeResults) == 0 {
				continue
			}
			
			best := sizeResults[0]
			for _, result := range sizeResults[1:] {
				if result.ExecutionTime < best.ExecutionTime {
					best = result
				}
			}
			patternWinners[pattern] = append(patternWinners[pattern], best)
		}
		
		// Sort by data size
		sort.Slice(patternWinners[pattern], func(i, j int) bool {
			return patternWinners[pattern][i].DataSize < patternWinners[pattern][j].DataSize
		})
	}
	
	return patternWinners
}

// calculateSpeedupVsWorst calculates speedup compared to worst performer in same scenario
func (rg *ReportGenerator) calculateSpeedupVsWorst(winner AlgorithmResult) float64 {
	var worstTime time.Duration
	
	// Find worst performer for same pattern and size
	for _, result := range rg.analyzer.Results {
		if result.DataPattern == winner.DataPattern && result.DataSize == winner.DataSize {
			if result.ExecutionTime > worstTime {
				worstTime = result.ExecutionTime
			}
		}
	}
	
	if winner.ExecutionTime == 0 {
		return 0
	}
	
	return float64(worstTime) / float64(winner.ExecutionTime)
}

// generateRecommendations generates performance recommendations
func (rg *ReportGenerator) generateRecommendations(writer io.Writer, report PerformanceReport) {
	// Analyze patterns and provide recommendations
	algorithmPerformance := make(map[string]map[string]float64)
	
	// Calculate average performance by algorithm and pattern
	for _, result := range rg.analyzer.Results {
		if algorithmPerformance[result.Name] == nil {
			algorithmPerformance[result.Name] = make(map[string]float64)
		}
		algorithmPerformance[result.Name][result.DataPattern] += result.ExecutionTime.Seconds()
	}
	
	// Generate specific recommendations
	fmt.Fprintf(writer, "### General Recommendations\n\n")
	
	// Find consistently best performers
	bestOverall := rg.findBestOverallAlgorithm()
	if bestOverall != "" {
		fmt.Fprintf(writer, "- **%s** shows consistently good performance across multiple scenarios\n", bestOverall)
	}
	
	// Pattern-specific recommendations
	fmt.Fprintf(writer, "\n### Pattern-Specific Recommendations\n\n")
	patternRecommendations := map[string]string{
		"random":          "Use QuickSort or TimSort for general-purpose sorting",
		"sorted":          "Use TimSort or InsertionSort for nearly sorted data",
		"reverse":         "Use HeapSort for guaranteed O(n log n) performance",
		"partially_sorted": "TimSort excels with partially sorted data",
		"many_duplicates": "Consider 3-way QuickSort or RadixSort for many duplicates",
	}
	
	for pattern, recommendation := range patternRecommendations {
		if _, exists := report.DataPatterns[pattern]; exists {
			fmt.Fprintf(writer, "- **%s data**: %s\n", strings.Title(strings.ReplaceAll(pattern, "_", " ")), recommendation)
		}
	}
	
	fmt.Fprintf(writer, "\n### Size-Based Recommendations\n\n")
	fmt.Fprintf(writer, "- **Small arrays (< 1000)**: Consider InsertionSort or TimSort\n")
	fmt.Fprintf(writer, "- **Medium arrays (1000-100000)**: QuickSort or MergeSort\n")
	fmt.Fprintf(writer, "- **Large arrays (> 100000)**: HeapSort for guaranteed performance, RadixSort for integers\n")
	
	fmt.Fprintf(writer, "\n### Optimization Opportunities\n\n")
	fmt.Fprintf(writer, "- Implement hybrid approaches (e.g., QuickSort + InsertionSort for small subarrays)\n")
	fmt.Fprintf(writer, "- Consider cache-aware implementations for large datasets\n")
	fmt.Fprintf(writer, "- Use SIMD instructions for primitive types where possible\n")
	fmt.Fprintf(writer, "- Implement branch prediction optimizations\n")
}

// findBestOverallAlgorithm finds the algorithm with best average performance
func (rg *ReportGenerator) findBestOverallAlgorithm() string {
	algorithmTotals := make(map[string]time.Duration)
	algorithmCounts := make(map[string]int)
	
	for _, result := range rg.analyzer.Results {
		algorithmTotals[result.Name] += result.ExecutionTime
		algorithmCounts[result.Name]++
	}
	
	bestAlgorithm := ""
	bestAverage := time.Duration(0)
	
	for algorithm, total := range algorithmTotals {
		count := algorithmCounts[algorithm]
		if count == 0 {
			continue
		}
		
		average := total / time.Duration(count)
		if bestAlgorithm == "" || average < bestAverage {
			bestAlgorithm = algorithm
			bestAverage = average
		}
	}
	
	return bestAlgorithm
}

// GenerateComparisonReport generates a detailed comparison between specific algorithms
func (rg *ReportGenerator) GenerateComparisonReport(writer io.Writer, algorithms []string) error {
	fmt.Fprintf(writer, "\n=== ALGORITHM COMPARISON REPORT ===\n\n")
	fmt.Fprintf(writer, "Comparing: %s\n\n", strings.Join(algorithms, ", "))
	
	// Filter results for specified algorithms
	filteredResults := make([]AlgorithmResult, 0)
	for _, result := range rg.analyzer.Results {
		for _, algorithm := range algorithms {
			if result.Name == algorithm {
				filteredResults = append(filteredResults, result)
				break
			}
		}
	}
	
	if len(filteredResults) == 0 {
		fmt.Fprintf(writer, "No results found for specified algorithms.\n")
		return nil
	}
	
	// Group by scenario (pattern + size)
	scenarios := make(map[string][]AlgorithmResult)
	for _, result := range filteredResults {
		key := fmt.Sprintf("%s_%d", result.DataPattern, result.DataSize)
		scenarios[key] = append(scenarios[key], result)
	}
	
	// Generate comparison for each scenario
	for scenario, results := range scenarios {
		if len(results) < 2 {
			continue // Need at least 2 algorithms to compare
		}
		
		comparison := CompareAlgorithms(results)
		fmt.Fprintf(writer, "Scenario: %s\n", scenario)
		fmt.Fprintf(writer, "  Fastest: %s (%v)\n", comparison.Fastest.Name, comparison.Fastest.ExecutionTime)
		fmt.Fprintf(writer, "  Slowest: %s (%v)\n", comparison.Slowest.Name, comparison.Slowest.ExecutionTime)
		fmt.Fprintf(writer, "  Speedup Range: %.2fx\n", comparison.SpeedupRange)
		fmt.Fprintf(writer, "  Relative Performance:\n")
		
		for algorithm, performance := range comparison.RelativePerformance {
			fmt.Fprintf(writer, "    %s: %.2fx\n", algorithm, performance)
		}
		fmt.Fprintf(writer, "\n")
	}
	
	return nil
}