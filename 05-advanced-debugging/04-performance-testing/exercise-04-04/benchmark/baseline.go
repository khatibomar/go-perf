package benchmark

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Baseline represents a performance baseline
type Baseline struct {
	Name        string                     `json:"name"`
	Version     string                     `json:"version"`
	Timestamp   time.Time                  `json:"timestamp"`
	Environment string                     `json:"environment"`
	Commit      string                     `json:"commit,omitempty"`
	Branch      string                     `json:"branch,omitempty"`
	Metrics     map[string]*BaselineMetric `json:"metrics"`
	Config      *BenchmarkConfig           `json:"config,omitempty"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`
}

// BaselineMetric represents a single metric in the baseline
type BaselineMetric struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Unit        string    `json:"unit"`
	Value       float64   `json:"value"`
	Min         float64   `json:"min"`
	Max         float64   `json:"max"`
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"std_dev"`
	SampleCount int       `json:"sample_count"`
	Tolerance   float64   `json:"tolerance"`
	Critical    bool      `json:"critical"`
	Timestamp   time.Time `json:"timestamp"`
}

// loadBaseline loads the baseline from the configured path
func (r *Runner) loadBaseline() error {
	if r.baselineDir == "" {
		return fmt.Errorf("baseline directory not configured")
	}

	baselinePath := filepath.Join(r.baselineDir, r.config.Baseline.Path)
	if r.config.Baseline.Path == "" {
		baselinePath = filepath.Join(r.baselineDir, "baseline.json")
	}

	data, err := os.ReadFile(baselinePath)
	if err != nil {
		if os.IsNotExist(err) {
			if r.verbose {
				fmt.Printf("Baseline file not found: %s\n", baselinePath)
			}
			return nil // Not an error if baseline doesn't exist
		}
		return fmt.Errorf("failed to read baseline file: %w", err)
	}

	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return fmt.Errorf("failed to parse baseline file: %w", err)
	}

	r.baseline = &baseline

	if r.verbose {
		fmt.Printf("Loaded baseline: %s (version %s, %d metrics)\n",
			baseline.Name, baseline.Version, len(baseline.Metrics))
	}

	return nil
}

// compareWithBaseline compares current results with the baseline
func (r *Runner) compareWithBaseline(results *BenchmarkResults) {
	if r.baseline == nil {
		return
	}

	comparison := &BaselineComparison{
		Exists:    true,
		Version:   r.baseline.Version,
		Timestamp: r.baseline.Timestamp,
		Metrics:   make(map[string]*MetricComparison),
	}

	regressionCount := 0
	improvementCount := 0

	// Compare each metric with baseline
	for metricName, currentMetric := range results.Metrics {
		baselineMetric, exists := r.baseline.Metrics[metricName]
		if !exists {
			continue
		}

		metricComparison := &MetricComparison{
			Name:      metricName,
			Current:   currentMetric.Value,
			Baseline:  baselineMetric.Value,
			Tolerance: baselineMetric.Tolerance,
		}

		// Calculate change
		if baselineMetric.Value != 0 {
			metricComparison.Change = currentMetric.Value - baselineMetric.Value
			metricComparison.ChangePercent = (metricComparison.Change / baselineMetric.Value) * 100
		}

		// Determine if within tolerance
		if baselineMetric.Tolerance > 0 {
			toleranceValue := baselineMetric.Value * (baselineMetric.Tolerance / 100)
			metricComparison.WithinTolerance = math.Abs(metricComparison.Change) <= toleranceValue
		} else {
			// Default tolerance of 5%
			toleranceValue := baselineMetric.Value * 0.05
			metricComparison.WithinTolerance = math.Abs(metricComparison.Change) <= toleranceValue
			metricComparison.Tolerance = 5.0
		}

		// Determine status based on metric type
		metricComparison.Status = r.determineMetricStatus(currentMetric, baselineMetric, metricComparison)

		// Count regressions and improvements
		if metricComparison.Status == "degraded" {
			regressionCount++
			if baselineMetric.Critical {
				comparison.Regression = true
			}
		} else if metricComparison.Status == "improved" {
			improvementCount++
			comparison.Improvement = true
		}

		comparison.Metrics[metricName] = metricComparison
		currentMetric.Baseline = baselineMetric
	}

	// Generate summary
	if regressionCount > 0 && improvementCount > 0 {
		comparison.Summary = fmt.Sprintf("%d regressions, %d improvements detected", regressionCount, improvementCount)
	} else if regressionCount > 0 {
		comparison.Summary = fmt.Sprintf("%d regressions detected", regressionCount)
	} else if improvementCount > 0 {
		comparison.Summary = fmt.Sprintf("%d improvements detected", improvementCount)
	} else {
		comparison.Summary = "Performance stable compared to baseline"
	}

	results.Baseline = comparison

	if r.verbose {
		fmt.Printf("Baseline comparison: %s\n", comparison.Summary)
	}
}

// determineMetricStatus determines if a metric has improved, degraded, or stayed stable
func (r *Runner) determineMetricStatus(current *MetricResults, baseline *BaselineMetric, comparison *MetricComparison) string {
	if comparison.WithinTolerance {
		return "stable"
	}

	// For latency and error rate metrics, lower is better
	if current.Type == "latency" || current.Type == "error_rate" || current.Type == "response_time" {
		if comparison.Change < 0 {
			return "improved"
		}
		return "degraded"
	}

	// For throughput and success rate metrics, higher is better
	if current.Type == "throughput" || current.Type == "success_rate" {
		if comparison.Change > 0 {
			return "improved"
		}
		return "degraded"
	}

	// Default: assume higher is better
	if comparison.Change > 0 {
		return "improved"
	}
	return "degraded"
}

// updateBaseline updates the baseline with current results
func (r *Runner) updateBaseline(results *BenchmarkResults) error {
	if r.baselineDir == "" {
		return fmt.Errorf("baseline directory not configured")
	}

	// Create baseline directory if it doesn't exist
	if err := os.MkdirAll(r.baselineDir, 0755); err != nil {
		return fmt.Errorf("failed to create baseline directory: %w", err)
	}

	// Create new baseline from current results
	baseline := &Baseline{
		Name:        r.config.Name,
		Version:     r.config.Baseline.Version,
		Timestamp:   time.Now(),
		Environment: r.config.Baseline.Environment,
		Metrics:     make(map[string]*BaselineMetric),
		Config:      r.config,
		Metadata:    make(map[string]interface{}),
	}

	// Convert current metrics to baseline metrics
	for metricName, metric := range results.Metrics {
		baselineMetric := &BaselineMetric{
			Name:        metric.Name,
			Type:        metric.Type,
			Unit:        metric.Unit,
			Value:       metric.Value,
			Min:         metric.Value, // For now, use current value
			Max:         metric.Value, // For now, use current value
			Mean:        metric.Value, // For now, use current value
			StdDev:      0,            // Would need historical data
			SampleCount: 1,
			Tolerance:   r.getMetricTolerance(metric),
			Critical:    metric.Critical,
			Timestamp:   time.Now(),
		}

		// If we have an existing baseline, incorporate historical data
		if r.baseline != nil {
			if existingMetric, exists := r.baseline.Metrics[metricName]; exists {
				// Update with historical data (simple moving average approach)
				baselineMetric.SampleCount = existingMetric.SampleCount + 1
				baselineMetric.Mean = (existingMetric.Mean*float64(existingMetric.SampleCount-1) + metric.Value) / float64(baselineMetric.SampleCount)

				if metric.Value < existingMetric.Min {
					baselineMetric.Min = metric.Value
				} else {
					baselineMetric.Min = existingMetric.Min
				}

				if metric.Value > existingMetric.Max {
					baselineMetric.Max = metric.Value
				} else {
					baselineMetric.Max = existingMetric.Max
				}

				// Simple standard deviation calculation
				variance := math.Pow(metric.Value-baselineMetric.Mean, 2)
				baselineMetric.StdDev = math.Sqrt(variance)
			}
		}

		baseline.Metrics[metricName] = baselineMetric
	}

	// Add metadata
	baseline.Metadata["total_requests"] = results.Summary.TotalRequests
	baseline.Metadata["success_rate"] = results.Summary.SuccessRate
	baseline.Metadata["avg_throughput"] = results.Summary.AvgThroughput
	baseline.Metadata["avg_latency_ms"] = float64(results.Summary.AvgLatency.Nanoseconds()) / 1e6
	baseline.Metadata["p95_latency_ms"] = float64(results.Summary.P95Latency.Nanoseconds()) / 1e6

	// Save baseline
	baselinePath := filepath.Join(r.baselineDir, r.config.Baseline.Path)
	if r.config.Baseline.Path == "" {
		baselinePath = filepath.Join(r.baselineDir, "baseline.json")
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write baseline file: %w", err)
	}

	// Update current baseline reference
	r.baseline = baseline

	if r.verbose {
		fmt.Printf("Baseline updated: %s\n", baselinePath)
	}

	return nil
}

// getMetricTolerance returns the appropriate tolerance for a metric
func (r *Runner) getMetricTolerance(metric *MetricResults) float64 {
	// Default tolerances based on metric type
	switch metric.Type {
	case "latency", "response_time":
		return 10.0 // 10% tolerance for latency metrics
	case "throughput":
		return 5.0 // 5% tolerance for throughput metrics
	case "error_rate":
		return 2.0 // 2% tolerance for error rate metrics
	case "success_rate":
		return 2.0 // 2% tolerance for success rate metrics
	default:
		return 5.0 // 5% default tolerance
	}
}

// CreateBaseline creates a new baseline from benchmark results
func CreateBaseline(results *BenchmarkResults, config *BenchmarkConfig, baselinePath string) error {
	baseline := &Baseline{
		Name:        config.Name,
		Version:     config.Baseline.Version,
		Timestamp:   time.Now(),
		Environment: config.Baseline.Environment,
		Metrics:     make(map[string]*BaselineMetric),
		Config:      config,
		Metadata:    make(map[string]interface{}),
	}

	// Convert metrics to baseline metrics
	for metricName, metric := range results.Metrics {
		baselineMetric := &BaselineMetric{
			Name:        metric.Name,
			Type:        metric.Type,
			Unit:        metric.Unit,
			Value:       metric.Value,
			Min:         metric.Value,
			Max:         metric.Value,
			Mean:        metric.Value,
			StdDev:      0,
			SampleCount: 1,
			Critical:    metric.Critical,
			Timestamp:   time.Now(),
		}

		// Set tolerance based on metric type
		switch metric.Type {
		case "latency", "response_time":
			baselineMetric.Tolerance = 10.0
		case "throughput":
			baselineMetric.Tolerance = 5.0
		case "error_rate":
			baselineMetric.Tolerance = 2.0
		case "success_rate":
			baselineMetric.Tolerance = 2.0
		default:
			baselineMetric.Tolerance = 5.0
		}

		baseline.Metrics[metricName] = baselineMetric
	}

	// Add metadata
	baseline.Metadata["total_requests"] = results.Summary.TotalRequests
	baseline.Metadata["success_rate"] = results.Summary.SuccessRate
	baseline.Metadata["avg_throughput"] = results.Summary.AvgThroughput
	baseline.Metadata["avg_latency_ms"] = float64(results.Summary.AvgLatency.Nanoseconds()) / 1e6
	baseline.Metadata["p95_latency_ms"] = float64(results.Summary.P95Latency.Nanoseconds()) / 1e6

	// Create directory if it doesn't exist
	dir := filepath.Dir(baselinePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create baseline directory: %w", err)
	}

	// Save baseline
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write baseline file: %w", err)
	}

	return nil
}

// LoadBaseline loads a baseline from a file
func LoadBaseline(baselinePath string) (*Baseline, error) {
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read baseline file: %w", err)
	}

	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("failed to parse baseline file: %w", err)
	}

	return &baseline, nil
}