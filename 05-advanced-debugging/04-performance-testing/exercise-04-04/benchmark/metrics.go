package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// calculateMetrics calculates all configured metrics
func (r *Runner) calculateMetrics(results *BenchmarkResults) {
	if results.Metrics == nil {
		results.Metrics = make(map[string]*MetricResults)
	}
	
	for _, metricConfig := range r.config.Metrics {
		metricResult := &MetricResults{
			Name:        metricConfig.Name,
			Type:        metricConfig.Type,
			Unit:        metricConfig.Unit,
			Aggregation: metricConfig.Aggregation,
			Critical:    metricConfig.Critical,
		}

		// Calculate metric value based on type
		switch metricConfig.Type {
		case "latency":
			metricResult.Value = r.calculateLatencyMetric(results, metricConfig)
		case "throughput":
			metricResult.Value = r.calculateThroughputMetric(results, metricConfig)
		case "error_rate":
			metricResult.Value = r.calculateErrorRateMetric(results, metricConfig)
		case "success_rate":
			metricResult.Value = r.calculateSuccessRateMetric(results, metricConfig)
		case "response_time":
			metricResult.Value = r.calculateResponseTimeMetric(results, metricConfig)
		default:
			metricResult.Value = 0
		}

		// Check threshold if specified
		if metricConfig.Threshold > 0 {
			metricResult.Threshold = metricConfig.Threshold
			switch metricConfig.Type {
			case "latency", "response_time":
				// For latency metrics, lower is better
				metricResult.Passed = metricResult.Value <= metricConfig.Threshold
			case "throughput":
				// For throughput, higher is better
				metricResult.Passed = metricResult.Value >= metricConfig.Threshold
			case "error_rate":
				// For error rate, lower is better
				metricResult.Passed = metricResult.Value <= metricConfig.Threshold
			case "success_rate":
				// For success rate, higher is better
				metricResult.Passed = metricResult.Value >= metricConfig.Threshold
			default:
				metricResult.Passed = true
			}
		} else {
			metricResult.Passed = true
		}

		results.Metrics[metricConfig.Name] = metricResult
	}
}

// calculateLatencyMetric calculates latency-based metrics
func (r *Runner) calculateLatencyMetric(results *BenchmarkResults, config MetricConfig) float64 {
	var totalLatency time.Duration
	var count int

	for _, scenario := range results.Scenarios {
		if scenario.Latencies == nil {
			continue
		}

		switch config.Aggregation {
		case "mean", "avg":
			totalLatency += scenario.Latencies.Mean
			count++
		case "median", "p50":
			totalLatency += scenario.Latencies.Median
			count++
		case "p90":
			totalLatency += scenario.Latencies.P90
			count++
		case "p95":
			totalLatency += scenario.Latencies.P95
			count++
		case "p99":
			totalLatency += scenario.Latencies.P99
			count++
		case "max":
			totalLatency += scenario.Latencies.Max
			count++
		default:
			totalLatency += scenario.Latencies.Mean
			count++
		}
	}

	if count == 0 {
		return 0
	}

	avgLatency := totalLatency / time.Duration(count)

	// Convert to appropriate unit
	switch config.Unit {
	case "ms":
		return float64(avgLatency.Nanoseconds()) / 1e6
	case "us":
		return float64(avgLatency.Nanoseconds()) / 1e3
	case "s":
		return avgLatency.Seconds()
	default:
		return float64(avgLatency.Nanoseconds())
	}
}

// calculateThroughputMetric calculates throughput-based metrics
func (r *Runner) calculateThroughputMetric(results *BenchmarkResults, config MetricConfig) float64 {
	var totalThroughput float64
	var count int

	for _, scenario := range results.Scenarios {
		switch config.Aggregation {
		case "sum":
			totalThroughput += scenario.Throughput
		case "mean", "avg":
			totalThroughput += scenario.Throughput
			count++
		case "max":
			if scenario.Throughput > totalThroughput {
				totalThroughput = scenario.Throughput
			}
		default:
			totalThroughput += scenario.Throughput
			count++
		}
	}

	if config.Aggregation == "mean" || config.Aggregation == "avg" {
		if count > 0 {
			totalThroughput /= float64(count)
		}
	}

	return totalThroughput
}

// calculateErrorRateMetric calculates error rate metrics
func (r *Runner) calculateErrorRateMetric(results *BenchmarkResults, config MetricConfig) float64 {
	var totalErrorRate float64
	var count int

	for _, scenario := range results.Scenarios {
		switch config.Aggregation {
		case "mean", "avg":
			totalErrorRate += scenario.ErrorRate
			count++
		case "max":
			if scenario.ErrorRate > totalErrorRate {
				totalErrorRate = scenario.ErrorRate
			}
		case "sum":
			totalErrorRate += scenario.ErrorRate
		default:
			totalErrorRate += scenario.ErrorRate
			count++
		}
	}

	if config.Aggregation == "mean" || config.Aggregation == "avg" {
		if count > 0 {
			totalErrorRate /= float64(count)
		}
	}

	return totalErrorRate
}

// calculateSuccessRateMetric calculates success rate metrics
func (r *Runner) calculateSuccessRateMetric(results *BenchmarkResults, config MetricConfig) float64 {
	return 100.0 - r.calculateErrorRateMetric(results, config)
}

// calculateResponseTimeMetric calculates response time metrics for targets
func (r *Runner) calculateResponseTimeMetric(results *BenchmarkResults, config MetricConfig) float64 {
	var totalResponseTime time.Duration
	var count int

	for _, target := range results.Targets {
		if target.Healthy {
			totalResponseTime += target.ResponseTime
			count++
		}
	}

	if count == 0 {
		return 0
	}

	avgResponseTime := totalResponseTime / time.Duration(count)

	// Convert to appropriate unit
	switch config.Unit {
	case "ms":
		return float64(avgResponseTime.Nanoseconds()) / 1e6
	case "us":
		return float64(avgResponseTime.Nanoseconds()) / 1e3
	case "s":
		return avgResponseTime.Seconds()
	default:
		return float64(avgResponseTime.Nanoseconds())
	}
}

// validateThresholds validates all configured thresholds
func (r *Runner) validateThresholds(results *BenchmarkResults) {
	thresholds := &ThresholdResults{
		Passed: true,
	}

	// Validate latency thresholds
	thresholds.Latency = r.validateLatencyThresholds(results)
	if !thresholds.Latency.P50Passed || !thresholds.Latency.P90Passed ||
		!thresholds.Latency.P95Passed || !thresholds.Latency.P99Passed || !thresholds.Latency.MaxPassed {
		thresholds.Passed = false
	}

	// Validate throughput thresholds
	thresholds.Throughput = r.validateThroughputThresholds(results)
	if !thresholds.Throughput.MinRPSPassed || !thresholds.Throughput.MaxRPSPassed {
		thresholds.Passed = false
	}

	// Validate error rate thresholds
	thresholds.ErrorRate = r.validateErrorRateThresholds(results)
	if !thresholds.ErrorRate.MaxPercentPassed || !thresholds.ErrorRate.MaxCountPassed {
		thresholds.Passed = false
	}

	// Validate resource thresholds (placeholder)
	thresholds.Resources = &ResourceThresholdResults{
		CPUPassed:     true, // Placeholder
		MemoryPassed:  true, // Placeholder
		DiskPassed:    true, // Placeholder
		NetworkPassed: true, // Placeholder
	}

	results.Thresholds = thresholds
}

// validateLatencyThresholds validates latency thresholds
func (r *Runner) validateLatencyThresholds(results *BenchmarkResults) *LatencyThresholdResults {
	latencyThresholds := &LatencyThresholdResults{
		P50Passed: true,
		P90Passed: true,
		P95Passed: true,
		P99Passed: true,
		MaxPassed: true,
	}

	for _, scenario := range results.Scenarios {
		if scenario.Latencies == nil {
			continue
		}

		if r.config.Thresholds.Latency.P50 > 0 {
			if scenario.Latencies.Median > time.Duration(r.config.Thresholds.Latency.P50*1e6) {
				latencyThresholds.P50Passed = false
			}
		}

		if r.config.Thresholds.Latency.P90 > 0 {
			if scenario.Latencies.P90 > time.Duration(r.config.Thresholds.Latency.P90*1e6) {
				latencyThresholds.P90Passed = false
			}
		}

		if r.config.Thresholds.Latency.P95 > 0 {
			if scenario.Latencies.P95 > time.Duration(r.config.Thresholds.Latency.P95*1e6) {
				latencyThresholds.P95Passed = false
			}
		}

		if r.config.Thresholds.Latency.P99 > 0 {
			if scenario.Latencies.P99 > time.Duration(r.config.Thresholds.Latency.P99*1e6) {
				latencyThresholds.P99Passed = false
			}
		}

		if r.config.Thresholds.Latency.Max > 0 {
			if scenario.Latencies.Max > time.Duration(r.config.Thresholds.Latency.Max*1e6) {
				latencyThresholds.MaxPassed = false
			}
		}
	}

	return latencyThresholds
}

// validateThroughputThresholds validates throughput thresholds
func (r *Runner) validateThroughputThresholds(results *BenchmarkResults) *ThroughputThresholdResults {
	throughputThresholds := &ThroughputThresholdResults{
		MinRPSPassed: true,
		MaxRPSPassed: true,
	}

	var totalThroughput float64
	for _, scenario := range results.Scenarios {
		totalThroughput += scenario.Throughput
	}

	if r.config.Thresholds.Throughput.MinRPS > 0 {
		if totalThroughput < r.config.Thresholds.Throughput.MinRPS {
			throughputThresholds.MinRPSPassed = false
		}
	}

	if r.config.Thresholds.Throughput.MaxRPS > 0 {
		if totalThroughput > r.config.Thresholds.Throughput.MaxRPS {
			throughputThresholds.MaxRPSPassed = false
		}
	}

	return throughputThresholds
}

// validateErrorRateThresholds validates error rate thresholds
func (r *Runner) validateErrorRateThresholds(results *BenchmarkResults) *ErrorRateThresholdResults {
	errorRateThresholds := &ErrorRateThresholdResults{
		MaxPercentPassed: true,
		MaxCountPassed:   true,
	}

	var totalErrors int64
	var totalRequests int64
	for _, scenario := range results.Scenarios {
		totalErrors += scenario.FailedRequests
		totalRequests += scenario.TotalRequests
	}

	if totalRequests > 0 {
		errorRate := float64(totalErrors) / float64(totalRequests) * 100

		if r.config.Thresholds.ErrorRate.MaxPercent > 0 {
			if errorRate > r.config.Thresholds.ErrorRate.MaxPercent {
				errorRateThresholds.MaxPercentPassed = false
			}
		}
	}

	if r.config.Thresholds.ErrorRate.MaxCount > 0 {
		if totalErrors > r.config.Thresholds.ErrorRate.MaxCount {
			errorRateThresholds.MaxCountPassed = false
		}
	}

	return errorRateThresholds
}

// generateSummary generates a high-level summary of results
func (r *Runner) generateSummary(results *BenchmarkResults) {
	summary := &ResultSummary{
		OverallPassed: true,
		Recommendations: make([]string, 0),
	}

	// Calculate overall statistics
	var totalRequests, successRequests int64
	var totalLatency time.Duration
	var maxP95Latency time.Duration
	var totalThroughput float64
	scenarioCount := 0

	for _, scenario := range results.Scenarios {
		totalRequests += scenario.TotalRequests
		successRequests += scenario.SuccessRequests
		totalThroughput += scenario.Throughput
		scenarioCount++

		if scenario.Latencies != nil {
			totalLatency += scenario.Latencies.Mean
			if scenario.Latencies.P95 > maxP95Latency {
				maxP95Latency = scenario.Latencies.P95
			}
		}
	}

	summary.TotalRequests = totalRequests
	if totalRequests > 0 {
		summary.SuccessRate = float64(successRequests) / float64(totalRequests) * 100
		summary.ErrorRate = 100 - summary.SuccessRate
	}

	if scenarioCount > 0 {
		summary.AvgLatency = totalLatency / time.Duration(scenarioCount)
		summary.AvgThroughput = totalThroughput / float64(scenarioCount)
	}
	summary.P95Latency = maxP95Latency

	// Check overall pass/fail status
	if results.Thresholds != nil && !results.Thresholds.Passed {
		summary.OverallPassed = false
	}

	// Check critical metrics
	for _, metric := range results.Metrics {
		if metric.Critical && !metric.Passed {
			summary.OverallPassed = false
		}
	}

	// Count regressions and improvements from baseline
	if results.Baseline != nil {
		for _, comparison := range results.Baseline.Metrics {
			if comparison.Status == "degraded" {
				summary.RegressionCount++
			} else if comparison.Status == "improved" {
				summary.ImprovementCount++
			}
		}

		if results.Baseline.Regression {
			summary.OverallPassed = false
		}
	}

	// Generate recommendations
	if summary.ErrorRate > 5 {
		summary.Recommendations = append(summary.Recommendations,
			fmt.Sprintf("High error rate (%.2f%%) detected. Check application logs and error handling.", summary.ErrorRate))
	}

	if summary.AvgLatency > 1*time.Second {
		summary.Recommendations = append(summary.Recommendations,
			fmt.Sprintf("High average latency (%v) detected. Consider performance optimization.", summary.AvgLatency))
	}

	if summary.AvgThroughput < 10 {
		summary.Recommendations = append(summary.Recommendations,
			fmt.Sprintf("Low throughput (%.2f RPS) detected. Consider scaling or optimization.", summary.AvgThroughput))
	}

	if summary.RegressionCount > 0 {
		summary.Recommendations = append(summary.Recommendations,
			fmt.Sprintf("%d performance regressions detected compared to baseline.", summary.RegressionCount))
	}

	results.Summary = summary
}

// SaveResults saves benchmark results to a file
func (r *Runner) SaveResults(results *BenchmarkResults, filename string) error {
	if r.outputDir != "" {
		if err := os.MkdirAll(r.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		filename = filepath.Join(r.outputDir, filename)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	if r.verbose {
		fmt.Printf("Results saved to: %s\n", filename)
	}

	return nil
}