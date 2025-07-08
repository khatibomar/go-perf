// pipeline/pipeline.go
package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"exercise-04-01/loadtest"
	"exercise-04-01/reporter"
)

type PerformancePipeline struct {
	config   PipelineConfig
	baseline *PerformanceBaseline
	reporter reporter.Reporter
}

type PipelineConfig struct {
	Name            string
	Description     string
	BaselinePath    string
	ReportPath      string
	Notification    NotificationConfig
	RegressionRules RegressionRules
}

type NotificationConfig struct {
	Enabled    bool
	WebhookURL string
	Email      string
	SlackToken string
	Channel    string
}

type RegressionRules struct {
	LatencyThreshold    float64 // Percentage increase threshold
	ThroughputThreshold float64 // Percentage decrease threshold
	ErrorRateThreshold  float64 // Percentage increase threshold
}

type PerformanceTest struct {
	Name        string
	Description string
	Config      *loadtest.LoadTestConfig
	Expected    ExpectedResults
}

type ExpectedResults struct {
	MaxLatencyP95    time.Duration
	MinThroughput    float64
	MaxErrorRate     float64
	MaxLatencyP99    time.Duration
	MinSuccessRate   float64
}

type TestResult struct {
	Test      *PerformanceTest
	Result    *loadtest.LoadTestResult
	Passed    bool
	Failures  []string
	Timestamp time.Time
	Duration  time.Duration
}

type PerformanceBaseline struct {
	Version   string
	Timestamp time.Time
	Metrics   map[string]BaselineMetric
}

type BaselineMetric struct {
	Name         string
	Value        float64
	Unit         string
	LatencyP50   time.Duration
	LatencyP95   time.Duration
	LatencyP99   time.Duration
	Throughput   float64
	ErrorRate    float64
	SuccessRate  float64
}

type RegressionAnalysis struct {
	HasRegression bool
	Comparisons   []MetricComparison
	Summary       string
}

type MetricComparison struct {
	Metric        string
	Baseline      float64
	Current       float64
	Change        float64
	ChangePercent float64
	IsRegression  bool
	Threshold     float64
}

func NewPerformancePipeline(config PipelineConfig) *PerformancePipeline {
	return &PerformancePipeline{
		config:   config,
		reporter: reporter.NewReporter("html"),
	}
}

func (p *PerformancePipeline) LoadBaseline() error {
	if p.config.BaselinePath == "" {
		return nil // No baseline configured
	}

	data, err := os.ReadFile(p.config.BaselinePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Baseline doesn't exist yet
		}
		return fmt.Errorf("failed to read baseline file: %w", err)
	}

	var baseline PerformanceBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return fmt.Errorf("failed to parse baseline file: %w", err)
	}

	p.baseline = &baseline
	return nil
}

func (p *PerformancePipeline) SaveBaseline(result *loadtest.LoadTestResult, version string) error {
	if p.config.BaselinePath == "" {
		return nil
	}

	baseline := &PerformanceBaseline{
		Version:   version,
		Timestamp: time.Now(),
		Metrics:   make(map[string]BaselineMetric),
	}

	// Create baseline metrics from test result
	baseline.Metrics["overall"] = BaselineMetric{
		Name:        "overall",
		LatencyP50:  result.Latencies.P50,
		LatencyP95:  result.Latencies.P95,
		LatencyP99:  result.Latencies.P99,
		Throughput:  result.Throughput,
		ErrorRate:   float64(result.FailedRequests) / float64(result.TotalRequests) * 100,
		SuccessRate: float64(result.SuccessRequests) / float64(result.TotalRequests) * 100,
	}

	// Add scenario-specific metrics
		for name, scenario := range result.Scenarios {
			baseline.Metrics[name] = BaselineMetric{
				Name:        name,
				LatencyP95:  scenario.AvgLatency, // Use average latency as approximation
				LatencyP99:  scenario.AvgLatency, // Use average latency as approximation
				Throughput:  scenario.Throughput,
				ErrorRate:   float64(scenario.FailedRequests) / float64(scenario.TotalRequests) * 100,
				SuccessRate: float64(scenario.SuccessRequests) / float64(scenario.TotalRequests) * 100,
			}
		}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(p.config.BaselinePath), 0755); err != nil {
		return fmt.Errorf("failed to create baseline directory: %w", err)
	}

	if err := os.WriteFile(p.config.BaselinePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write baseline file: %w", err)
	}

	p.baseline = baseline
	fmt.Printf("Baseline saved: %s\n", p.config.BaselinePath)
	return nil
}

func (p *PerformancePipeline) RunTest(test *PerformanceTest) (*TestResult, error) {
	start := time.Now()

	// Create and run load tester
	tester := loadtest.NewLoadTester(*test.Config)
	result, err := tester.Run()
	if err != nil {
		return nil, fmt.Errorf("load test failed: %w", err)
	}

	// Validate results against expectations
	failures := p.validateResults(result, test.Expected)

	testResult := &TestResult{
		Test:      test,
		Result:    result,
		Passed:    len(failures) == 0,
		Failures:  failures,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}

	return testResult, nil
}

func (p *PerformancePipeline) validateResults(result *loadtest.LoadTestResult, expected ExpectedResults) []string {
	var failures []string

	if expected.MaxLatencyP95 > 0 && result.Latencies.P95 > expected.MaxLatencyP95 {
		failures = append(failures, fmt.Sprintf("P95 latency %v exceeds maximum %v", result.Latencies.P95, expected.MaxLatencyP95))
	}

	if expected.MaxLatencyP99 > 0 && result.Latencies.P99 > expected.MaxLatencyP99 {
		failures = append(failures, fmt.Sprintf("P99 latency %v exceeds maximum %v", result.Latencies.P99, expected.MaxLatencyP99))
	}

	if expected.MinThroughput > 0 && result.Throughput < expected.MinThroughput {
		failures = append(failures, fmt.Sprintf("Throughput %.2f is below minimum %.2f", result.Throughput, expected.MinThroughput))
	}

	errorRate := float64(result.FailedRequests) / float64(result.TotalRequests) * 100
	if expected.MaxErrorRate >= 0 && errorRate > expected.MaxErrorRate {
		failures = append(failures, fmt.Sprintf("Error rate %.2f%% exceeds maximum %.2f%%", errorRate, expected.MaxErrorRate))
	}

	successRate := float64(result.SuccessRequests) / float64(result.TotalRequests) * 100
	if expected.MinSuccessRate > 0 && successRate < expected.MinSuccessRate {
		failures = append(failures, fmt.Sprintf("Success rate %.2f%% is below minimum %.2f%%", successRate, expected.MinSuccessRate))
	}

	return failures
}

func (p *PerformancePipeline) AnalyzeRegression(result *loadtest.LoadTestResult) *RegressionAnalysis {
	if p.baseline == nil {
		return &RegressionAnalysis{
			HasRegression: false,
			Summary:       "No baseline available for comparison",
		}
	}

	analysis := &RegressionAnalysis{
		Comparisons: make([]MetricComparison, 0),
	}

	// Compare overall metrics
	if baselineMetric, exists := p.baseline.Metrics["overall"]; exists {
		analysis.Comparisons = append(analysis.Comparisons, p.compareMetrics("P95 Latency", 
			float64(baselineMetric.LatencyP95.Nanoseconds()), 
			float64(result.Latencies.P95.Nanoseconds()), 
			p.config.RegressionRules.LatencyThreshold)...)

		analysis.Comparisons = append(analysis.Comparisons, p.compareMetrics("Throughput", 
			baselineMetric.Throughput, 
			result.Throughput, 
			-p.config.RegressionRules.ThroughputThreshold)...)

		currentErrorRate := float64(result.FailedRequests) / float64(result.TotalRequests) * 100
		analysis.Comparisons = append(analysis.Comparisons, p.compareMetrics("Error Rate", 
			baselineMetric.ErrorRate, 
			currentErrorRate, 
			p.config.RegressionRules.ErrorRateThreshold)...)
	}

	// Check if any comparison indicates regression
	for _, comp := range analysis.Comparisons {
		if comp.IsRegression {
			analysis.HasRegression = true
			break
		}
	}

	if analysis.HasRegression {
		analysis.Summary = "Performance regression detected"
	} else {
		analysis.Summary = "No performance regression detected"
	}

	return analysis
}

func (p *PerformancePipeline) compareMetrics(name string, baseline, current, threshold float64) []MetricComparison {
	change := current - baseline
	changePercent := (change / baseline) * 100

	// For throughput, negative threshold means we care about decreases
	// For latency/error rate, positive threshold means we care about increases
	isRegression := false
	if threshold > 0 {
		isRegression = changePercent > threshold
	} else {
		isRegression = changePercent < threshold
	}

	return []MetricComparison{{
		Metric:        name,
		Baseline:      baseline,
		Current:       current,
		Change:        change,
		ChangePercent: changePercent,
		IsRegression:  isRegression,
		Threshold:     threshold,
	}}
}

func (p *PerformancePipeline) GenerateReport(testResult *TestResult, analysis *RegressionAnalysis) error {
	if p.config.ReportPath == "" {
		return nil
	}

	reportConfig := reporter.ReportConfig{
		OutputPath:  p.config.ReportPath,
		Format:      "html",
		Title:       fmt.Sprintf("%s - Performance Test Report", p.config.Name),
		Description: fmt.Sprintf("%s\n\nTest: %s\nPassed: %v", p.config.Description, testResult.Test.Name, testResult.Passed),
	}

	return p.reporter.GenerateReport(testResult.Result, reportConfig)
}

func (p *PerformancePipeline) SendNotification(testResult *TestResult, analysis *RegressionAnalysis) error {
	if !p.config.Notification.Enabled {
		return nil
	}

	message := p.formatNotificationMessage(testResult, analysis)
	fmt.Printf("Notification: %s\n", message)

	// In a real implementation, you would send to Slack, email, webhook, etc.
	// For now, we just print the notification
	return nil
}

func (p *PerformancePipeline) formatNotificationMessage(testResult *TestResult, analysis *RegressionAnalysis) string {
	status := "✅ PASSED"
	if !testResult.Passed || analysis.HasRegression {
		status = "❌ FAILED"
	}

	message := fmt.Sprintf("%s Performance Test: %s\n", status, testResult.Test.Name)
	message += fmt.Sprintf("Duration: %v\n", testResult.Duration)
	message += fmt.Sprintf("Throughput: %.2f req/s\n", testResult.Result.Throughput)
	message += fmt.Sprintf("P95 Latency: %v\n", testResult.Result.Latencies.P95)
	message += fmt.Sprintf("Error Rate: %.2f%%\n", float64(testResult.Result.FailedRequests)/float64(testResult.Result.TotalRequests)*100)

	if len(testResult.Failures) > 0 {
		message += "\nFailures:\n"
		for _, failure := range testResult.Failures {
			message += fmt.Sprintf("- %s\n", failure)
		}
	}

	if analysis.HasRegression {
		message += "\n⚠️ Performance Regression Detected:\n"
		for _, comp := range analysis.Comparisons {
			if comp.IsRegression {
				message += fmt.Sprintf("- %s: %.2f%% change (threshold: %.2f%%)\n", 
					comp.Metric, comp.ChangePercent, comp.Threshold)
			}
		}
	}

	return message
}

func (p *PerformancePipeline) RunPipeline(tests []*PerformanceTest, version string) error {
	// Load existing baseline
	if err := p.LoadBaseline(); err != nil {
		return fmt.Errorf("failed to load baseline: %w", err)
	}

	fmt.Printf("Starting performance pipeline: %s\n", p.config.Name)

	for _, test := range tests {
		fmt.Printf("Running test: %s\n", test.Name)

		// Run the test
		testResult, err := p.RunTest(test)
		if err != nil {
			return fmt.Errorf("test %s failed: %w", test.Name, err)
		}

		// Analyze for regression
		analysis := p.AnalyzeRegression(testResult.Result)

		// Generate report
		if err := p.GenerateReport(testResult, analysis); err != nil {
			fmt.Printf("Warning: failed to generate report: %v\n", err)
		}

		// Send notification
		if err := p.SendNotification(testResult, analysis); err != nil {
			fmt.Printf("Warning: failed to send notification: %v\n", err)
		}

		// Update baseline if test passed and no regression
		if testResult.Passed && !analysis.HasRegression {
			if err := p.SaveBaseline(testResult.Result, version); err != nil {
				fmt.Printf("Warning: failed to save baseline: %v\n", err)
			}
		}

		fmt.Printf("Test %s completed: %s\n", test.Name, map[bool]string{true: "PASSED", false: "FAILED"}[testResult.Passed && !analysis.HasRegression])
	}

	fmt.Printf("Performance pipeline completed: %s\n", p.config.Name)
	return nil
}