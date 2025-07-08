package analysis

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Reporter handles formatting and output of analysis results
type Reporter struct {
	writer io.Writer
	config ReportConfig
}

// ReportConfig holds configuration for report generation
type ReportConfig struct {
	Format          string // "text", "markdown", "json", "csv"
	IncludeCharts   bool
	IncludeDetails  bool
	IncludeRawData  bool
	SortBy          string // "algorithm", "performance", "memory"
	ShowConfidence  bool
	DecimalPlaces   int
}

// NewReporter creates a new reporter
func NewReporter(writer io.Writer, config ReportConfig) *Reporter {
	return &Reporter{
		writer: writer,
		config: config,
	}
}

// GenerateReport generates a formatted report from analysis results
func (r *Reporter) GenerateReport(report *AnalysisReport) error {
	switch r.config.Format {
	case "markdown":
		return r.generateMarkdownReport(report)
	case "json":
		return r.generateJSONReport(report)
	case "csv":
		return r.generateCSVReport(report)
	default:
		return r.generateTextReport(report)
	}
}

// generateTextReport generates a plain text report
func (r *Reporter) generateTextReport(report *AnalysisReport) error {
	// Header
	fmt.Fprintf(r.writer, "\n" + strings.Repeat("=", 80) + "\n")
	fmt.Fprintf(r.writer, "SEARCH ALGORITHM PERFORMANCE ANALYSIS REPORT\n")
	fmt.Fprintf(r.writer, "Generated: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(r.writer, strings.Repeat("=", 80) + "\n\n")

	// System Information
	fmt.Fprintf(r.writer, "SYSTEM INFORMATION\n")
	fmt.Fprintf(r.writer, strings.Repeat("-", 40) + "\n")
	fmt.Fprintf(r.writer, "CPU: %s\n", report.SystemInfo.CPU)
	fmt.Fprintf(r.writer, "Cores: %d\n", report.SystemInfo.Cores)
	fmt.Fprintf(r.writer, "Memory: %s\n", formatBytes(report.SystemInfo.Memory))
	fmt.Fprintf(r.writer, "L1 Cache: %s\n", formatBytes(int64(report.SystemInfo.CacheL1)))
	fmt.Fprintf(r.writer, "L2 Cache: %s\n", formatBytes(int64(report.SystemInfo.CacheL2)))
	fmt.Fprintf(r.writer, "L3 Cache: %s\n", formatBytes(int64(report.SystemInfo.CacheL3)))
	fmt.Fprintf(r.writer, "OS: %s\n", report.SystemInfo.OS)
	fmt.Fprintf(r.writer, "Go Version: %s\n\n", report.SystemInfo.GoVersion)

	// Performance Metrics Summary
	fmt.Fprintf(r.writer, "PERFORMANCE METRICS SUMMARY\n")
	fmt.Fprintf(r.writer, strings.Repeat("-", 40) + "\n")

	// Sort metrics based on configuration
	sortedMetrics := r.sortMetrics(report.Metrics)

	// Table header
	fmt.Fprintf(r.writer, "%-20s %-10s %-12s %-12s %-12s %-10s\n",
		"Algorithm", "Data Size", "Avg Time", "Min Time", "Max Time", "Throughput")
	fmt.Fprintf(r.writer, strings.Repeat("-", 80) + "\n")

	// Table rows
	for _, m := range sortedMetrics {
		fmt.Fprintf(r.writer, "%-20s %-10d %-12s %-12s %-12s %-10.0f\n",
			m.Algorithm,
			m.DataSize,
			r.formatDuration(m.AverageTime),
			r.formatDuration(m.MinTime),
			r.formatDuration(m.MaxTime),
			m.Throughput)
	}
	fmt.Fprintf(r.writer, "\n")

	// Detailed Metrics (if enabled)
	if r.config.IncludeDetails {
		r.generateDetailedMetrics(sortedMetrics)
	}

	// Comparisons
	if len(report.Comparisons) > 0 {
		fmt.Fprintf(r.writer, "ALGORITHM COMPARISONS\n")
		fmt.Fprintf(r.writer, strings.Repeat("-", 40) + "\n")

		for _, comp := range report.Comparisons {
			fmt.Fprintf(r.writer, "%s vs %s:\n", comp.BaseAlgorithm, comp.CompareAlgorithm)
			fmt.Fprintf(r.writer, "  Speedup Factor: %.2fx\n", comp.SpeedupFactor)
			fmt.Fprintf(r.writer, "  Memory Ratio: %.2fx\n", comp.MemoryRatio)
			fmt.Fprintf(r.writer, "  Efficiency Score: %.2f\n", comp.EfficiencyScore)
			fmt.Fprintf(r.writer, "  Recommendation: %s\n\n", comp.Recommendation)
		}
	}

	// Insights
	if len(report.Insights) > 0 {
		fmt.Fprintf(r.writer, "KEY INSIGHTS\n")
		fmt.Fprintf(r.writer, strings.Repeat("-", 40) + "\n")
		for i, insight := range report.Insights {
			fmt.Fprintf(r.writer, "%d. %s\n", i+1, insight)
		}
		fmt.Fprintf(r.writer, "\n")
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintf(r.writer, "RECOMMENDATIONS\n")
		fmt.Fprintf(r.writer, strings.Repeat("-", 40) + "\n")
		for i, rec := range report.Recommendations {
			fmt.Fprintf(r.writer, "%d. %s\n", i+1, rec)
		}
		fmt.Fprintf(r.writer, "\n")
	}

	// Performance Charts (ASCII)
	if r.config.IncludeCharts {
		r.generateASCIICharts(sortedMetrics)
	}

	return nil
}

// generateMarkdownReport generates a Markdown formatted report
func (r *Reporter) generateMarkdownReport(report *AnalysisReport) error {
	fmt.Fprintf(r.writer, "# Search Algorithm Performance Analysis Report\n\n")
	fmt.Fprintf(r.writer, "**Generated:** %s\n\n", report.Timestamp.Format(time.RFC3339))

	// System Information
	fmt.Fprintf(r.writer, "## System Information\n\n")
	fmt.Fprintf(r.writer, "| Component | Value |\n")
	fmt.Fprintf(r.writer, "|-----------|-------|\n")
	fmt.Fprintf(r.writer, "| CPU | %s |\n", report.SystemInfo.CPU)
	fmt.Fprintf(r.writer, "| Cores | %d |\n", report.SystemInfo.Cores)
	fmt.Fprintf(r.writer, "| Memory | %s |\n", formatBytes(report.SystemInfo.Memory))
	fmt.Fprintf(r.writer, "| L1 Cache | %s |\n", formatBytes(int64(report.SystemInfo.CacheL1)))
	fmt.Fprintf(r.writer, "| L2 Cache | %s |\n", formatBytes(int64(report.SystemInfo.CacheL2)))
	fmt.Fprintf(r.writer, "| L3 Cache | %s |\n", formatBytes(int64(report.SystemInfo.CacheL3)))
	fmt.Fprintf(r.writer, "| OS | %s |\n", report.SystemInfo.OS)
	fmt.Fprintf(r.writer, "| Go Version | %s |\n\n", report.SystemInfo.GoVersion)

	// Performance Metrics
	fmt.Fprintf(r.writer, "## Performance Metrics\n\n")
	fmt.Fprintf(r.writer, "| Algorithm | Data Size | Avg Time | Min Time | Max Time | Throughput (ops/sec) |\n")
	fmt.Fprintf(r.writer, "|-----------|-----------|----------|----------|----------|---------------------|\n")

	sortedMetrics := r.sortMetrics(report.Metrics)
	for _, m := range sortedMetrics {
		fmt.Fprintf(r.writer, "| %s | %d | %s | %s | %s | %.0f |\n",
			m.Algorithm,
			m.DataSize,
			r.formatDuration(m.AverageTime),
			r.formatDuration(m.MinTime),
			r.formatDuration(m.MaxTime),
			m.Throughput)
	}
	fmt.Fprintf(r.writer, "\n")

	// Comparisons
	if len(report.Comparisons) > 0 {
		fmt.Fprintf(r.writer, "## Algorithm Comparisons\n\n")
		for _, comp := range report.Comparisons {
			fmt.Fprintf(r.writer, "### %s vs %s\n\n", comp.BaseAlgorithm, comp.CompareAlgorithm)
			fmt.Fprintf(r.writer, "- **Speedup Factor:** %.2fx\n", comp.SpeedupFactor)
			fmt.Fprintf(r.writer, "- **Memory Ratio:** %.2fx\n", comp.MemoryRatio)
			fmt.Fprintf(r.writer, "- **Efficiency Score:** %.2f\n", comp.EfficiencyScore)
			fmt.Fprintf(r.writer, "- **Recommendation:** %s\n\n", comp.Recommendation)
		}
	}

	// Insights
	if len(report.Insights) > 0 {
		fmt.Fprintf(r.writer, "## Key Insights\n\n")
		for _, insight := range report.Insights {
			fmt.Fprintf(r.writer, "- %s\n", insight)
		}
		fmt.Fprintf(r.writer, "\n")
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintf(r.writer, "## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			fmt.Fprintf(r.writer, "- %s\n", rec)
		}
		fmt.Fprintf(r.writer, "\n")
	}

	return nil
}

// generateJSONReport generates a JSON formatted report
func (r *Reporter) generateJSONReport(report *AnalysisReport) error {
	// This would typically use json.Marshal, but for simplicity, we'll format manually
	fmt.Fprintf(r.writer, "{\n")
	fmt.Fprintf(r.writer, "  \"timestamp\": \"%s\",\n", report.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(r.writer, "  \"metrics\": [\n")

	for i, m := range report.Metrics {
		fmt.Fprintf(r.writer, "    {\n")
		fmt.Fprintf(r.writer, "      \"algorithm\": \"%s\",\n", m.Algorithm)
		fmt.Fprintf(r.writer, "      \"dataSize\": %d,\n", m.DataSize)
		fmt.Fprintf(r.writer, "      \"averageTime\": %d,\n", m.AverageTime.Nanoseconds())
		fmt.Fprintf(r.writer, "      \"throughput\": %.2f\n", m.Throughput)
		if i < len(report.Metrics)-1 {
			fmt.Fprintf(r.writer, "    },\n")
		} else {
			fmt.Fprintf(r.writer, "    }\n")
		}
	}

	fmt.Fprintf(r.writer, "  ]\n")
	fmt.Fprintf(r.writer, "}\n")

	return nil
}

// generateCSVReport generates a CSV formatted report
func (r *Reporter) generateCSVReport(report *AnalysisReport) error {
	// CSV Header
	fmt.Fprintf(r.writer, "Algorithm,DataSize,AverageTime(ns),MinTime(ns),MaxTime(ns),Throughput(ops/sec),StandardDev(ns)\n")

	// CSV Data
	for _, m := range report.Metrics {
		fmt.Fprintf(r.writer, "%s,%d,%d,%d,%d,%.2f,%d\n",
			m.Algorithm,
			m.DataSize,
			m.AverageTime.Nanoseconds(),
			m.MinTime.Nanoseconds(),
			m.MaxTime.Nanoseconds(),
			m.Throughput,
			m.StandardDev.Nanoseconds())
	}

	return nil
}

// generateDetailedMetrics generates detailed performance metrics
func (r *Reporter) generateDetailedMetrics(metrics []PerformanceMetrics) {
	fmt.Fprintf(r.writer, "DETAILED PERFORMANCE METRICS\n")
	fmt.Fprintf(r.writer, strings.Repeat("-", 40) + "\n")

	for _, m := range metrics {
		fmt.Fprintf(r.writer, "\nAlgorithm: %s (Data Size: %d)\n", m.Algorithm, m.DataSize)
		fmt.Fprintf(r.writer, "  Iterations: %d\n", m.Iterations)
		fmt.Fprintf(r.writer, "  Total Time: %s\n", r.formatDuration(m.TotalTime))
		fmt.Fprintf(r.writer, "  Average Time: %s\n", r.formatDuration(m.AverageTime))
		fmt.Fprintf(r.writer, "  Median Time: %s\n", r.formatDuration(m.MedianTime))
		fmt.Fprintf(r.writer, "  Min Time: %s\n", r.formatDuration(m.MinTime))
		fmt.Fprintf(r.writer, "  Max Time: %s\n", r.formatDuration(m.MaxTime))
		fmt.Fprintf(r.writer, "  Standard Deviation: %s\n", r.formatDuration(m.StandardDev))
		fmt.Fprintf(r.writer, "  95th Percentile: %s\n", r.formatDuration(m.Percentile95))
		fmt.Fprintf(r.writer, "  99th Percentile: %s\n", r.formatDuration(m.Percentile99))
		fmt.Fprintf(r.writer, "  Throughput: %.2f ops/sec\n", m.Throughput)
		fmt.Fprintf(r.writer, "  Success Rate: %.2f%%\n", m.SuccessRate*100)

		if m.MemoryUsage > 0 {
			fmt.Fprintf(r.writer, "  Memory Usage: %s\n", formatBytes(m.MemoryUsage))
		}
		if m.Allocations > 0 {
			fmt.Fprintf(r.writer, "  Allocations: %d\n", m.Allocations)
		}
	}
	fmt.Fprintf(r.writer, "\n")
}

// generateASCIICharts generates simple ASCII charts
func (r *Reporter) generateASCIICharts(metrics []PerformanceMetrics) {
	fmt.Fprintf(r.writer, "PERFORMANCE VISUALIZATION\n")
	fmt.Fprintf(r.writer, strings.Repeat("-", 40) + "\n")

	// Group metrics by algorithm for comparison
	algorithmGroups := make(map[string][]PerformanceMetrics)
	for _, m := range metrics {
		algorithmGroups[m.Algorithm] = append(algorithmGroups[m.Algorithm], m)
	}

	// Generate throughput chart
	fmt.Fprintf(r.writer, "\nThroughput Comparison (ops/sec):\n")
	fmt.Fprintf(r.writer, strings.Repeat("-", 60) + "\n")

	// Find max throughput for scaling
	maxThroughput := 0.0
	for _, group := range algorithmGroups {
		for _, m := range group {
			if m.Throughput > maxThroughput {
				maxThroughput = m.Throughput
			}
		}
	}

	// Generate bars
	for alg, group := range algorithmGroups {
		if len(group) > 0 {
			avgThroughput := 0.0
			for _, m := range group {
				avgThroughput += m.Throughput
			}
			avgThroughput /= float64(len(group))

			barLength := int((avgThroughput / maxThroughput) * 40)
			bar := strings.Repeat("█", barLength)
			fmt.Fprintf(r.writer, "%-15s |%s %.0f\n", alg, bar, avgThroughput)
		}
	}
	fmt.Fprintf(r.writer, "\n")
}

// sortMetrics sorts metrics based on configuration
func (r *Reporter) sortMetrics(metrics []PerformanceMetrics) []PerformanceMetrics {
	sorted := make([]PerformanceMetrics, len(metrics))
	copy(sorted, metrics)

	switch r.config.SortBy {
	case "performance":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].AverageTime < sorted[j].AverageTime
		})
	case "memory":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].MemoryUsage < sorted[j].MemoryUsage
		})
	default: // "algorithm"
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].Algorithm == sorted[j].Algorithm {
				return sorted[i].DataSize < sorted[j].DataSize
			}
			return sorted[i].Algorithm < sorted[j].Algorithm
		})
	}

	return sorted
}

// formatDuration formats a duration for display
func (r *Reporter) formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000.0)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// formatBytes formats byte counts for display
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GenerateComparisonTable generates a comparison table for specific algorithms
func (r *Reporter) GenerateComparisonTable(metrics []PerformanceMetrics, algorithms []string) error {
	fmt.Fprintf(r.writer, "\nALGORITHM COMPARISON TABLE\n")
	fmt.Fprintf(r.writer, strings.Repeat("-", 80) + "\n")

	// Filter metrics for specified algorithms
	filteredMetrics := make([]PerformanceMetrics, 0)
	for _, m := range metrics {
		for _, alg := range algorithms {
			if m.Algorithm == alg {
				filteredMetrics = append(filteredMetrics, m)
				break
			}
		}
	}

	if len(filteredMetrics) == 0 {
		fmt.Fprintf(r.writer, "No metrics found for specified algorithms\n")
		return nil
	}

	// Group by data size
	sizeGroups := make(map[int][]PerformanceMetrics)
	for _, m := range filteredMetrics {
		sizeGroups[m.DataSize] = append(sizeGroups[m.DataSize], m)
	}

	// Generate table for each data size
	for size, group := range sizeGroups {
		fmt.Fprintf(r.writer, "\nData Size: %d\n", size)
		fmt.Fprintf(r.writer, "%-20s %-12s %-12s %-10s\n", "Algorithm", "Avg Time", "Throughput", "Relative")
		fmt.Fprintf(r.writer, strings.Repeat("-", 60) + "\n")

		// Find baseline (fastest) for relative comparison
		var baseline *PerformanceMetrics
		for i := range group {
			if baseline == nil || group[i].AverageTime < baseline.AverageTime {
				baseline = &group[i]
			}
		}

		// Sort by performance
		sort.Slice(group, func(i, j int) bool {
			return group[i].AverageTime < group[j].AverageTime
		})

		for _, m := range group {
			relative := float64(m.AverageTime) / float64(baseline.AverageTime)
			fmt.Fprintf(r.writer, "%-20s %-12s %-12.0f %-10.2fx\n",
				m.Algorithm,
				r.formatDuration(m.AverageTime),
				m.Throughput,
				relative)
		}
	}

	return nil
}

// GeneratePerformanceProfile generates a performance profile report
func (r *Reporter) GeneratePerformanceProfile(metrics []PerformanceMetrics) error {
	fmt.Fprintf(r.writer, "\nPERFORMANCE PROFILE ANALYSIS\n")
	fmt.Fprintf(r.writer, strings.Repeat("-", 80) + "\n")

	// Analyze scalability
	algorithmGroups := make(map[string][]PerformanceMetrics)
	for _, m := range metrics {
		algorithmGroups[m.Algorithm] = append(algorithmGroups[m.Algorithm], m)
	}

	for alg, group := range algorithmGroups {
		if len(group) < 2 {
			continue
		}

		// Sort by data size
		sort.Slice(group, func(i, j int) bool {
			return group[i].DataSize < group[j].DataSize
		})

		fmt.Fprintf(r.writer, "\n%s Scalability Analysis:\n", alg)
		fmt.Fprintf(r.writer, "%-10s %-12s %-12s %-10s\n", "Size", "Time", "Growth", "Efficiency")
		fmt.Fprintf(r.writer, strings.Repeat("-", 50) + "\n")

		for i, m := range group {
			growthFactor := "-"
			efficiency := "-"

			if i > 0 {
				prev := group[i-1]
				sizeRatio := float64(m.DataSize) / float64(prev.DataSize)
				timeRatio := float64(m.AverageTime) / float64(prev.AverageTime)
				growthFactor = fmt.Sprintf("%.2fx", timeRatio/sizeRatio)
				efficiency = fmt.Sprintf("%.2f", sizeRatio/timeRatio)
			}

			fmt.Fprintf(r.writer, "%-10d %-12s %-12s %-10s\n",
				m.DataSize,
				r.formatDuration(m.AverageTime),
				growthFactor,
				efficiency)
		}
	}

	return nil
}