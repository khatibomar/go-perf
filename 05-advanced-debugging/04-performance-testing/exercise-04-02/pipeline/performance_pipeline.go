// pipeline/performance_pipeline.go
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PerformancePipeline struct {
	config     PipelineConfig
	testSuite  []PerformanceTest
	results    []TestResult
	baseline   PerformanceBaseline
	reporter   Reporter
	mu         sync.RWMutex
}

type PipelineConfig struct {
	Environment     string
	TestTimeout     time.Duration
	ParallelTests   int
	ResultsPath     string
	BaselinePath    string
	ReportFormats   []string
	Notifications   NotificationConfig
}

type PerformanceTest struct {
	Name            string
	Type            string // "load", "stress", "spike", "volume"
	Config          interface{}
	Prerequisites   []string
	Cleanup         []string
	Timeout         time.Duration
	RetryCount      int
	CriticalPath    bool
}

type TestResult struct {
	TestName        string
	Status          string // "passed", "failed", "skipped"
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	Metrics         map[string]interface{}
	Errors          []string
	Artifacts       []string
	Regression      *RegressionAnalysis
}

type PerformanceBaseline struct {
	Version         string
	Timestamp       time.Time
	Environment     string
	Metrics         map[string]BaselineMetric
}

type BaselineMetric struct {
	Name            string
	Value           float64
	Unit            string
	Tolerance       float64 // percentage
	Trend           string  // "improving", "stable", "degrading"
}

type RegressionAnalysis struct {
	HasRegression   bool
	Severity        string
	AffectedMetrics []string
	Comparison      map[string]MetricComparison
	Recommendation  string
}

type MetricComparison struct {
	Current         float64
	Baseline        float64
	Change          float64
	ChangePercent   float64
	WithinTolerance bool
}

type NotificationConfig struct {
	Slack           SlackConfig
	Email           EmailConfig
	Webhook         WebhookConfig
	OnFailure       bool
	OnRegression    bool
	OnSuccess       bool
}

type SlackConfig struct {
	WebhookURL string
	Channel    string
	Username   string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	From         string
	To           []string
}

type WebhookConfig struct {
	URL     string
	Headers map[string]string
}

type Reporter interface {
	GenerateReport(results []TestResult, baseline PerformanceBaseline) error
	SendNotification(result TestResult, regression *RegressionAnalysis) error
}

func NewPerformancePipeline(config PipelineConfig) *PerformancePipeline {
	return &PerformancePipeline{
		config:    config,
		testSuite: make([]PerformanceTest, 0),
		results:   make([]TestResult, 0),
		reporter:  NewHTMLReporter(config),
	}
}

func (pp *PerformancePipeline) LoadBaseline() error {
	baselinePath := filepath.Join(pp.config.BaselinePath, "baseline.json")
	
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		// No baseline exists, create empty one
		pp.baseline = PerformanceBaseline{
			Version:     "initial",
			Timestamp:   time.Now(),
			Environment: pp.config.Environment,
			Metrics:     make(map[string]BaselineMetric),
		}
		return nil
	}
	
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		return fmt.Errorf("failed to read baseline: %w", err)
	}
	
	if err := json.Unmarshal(data, &pp.baseline); err != nil {
		return fmt.Errorf("failed to parse baseline: %w", err)
	}
	
	return nil
}

func (pp *PerformancePipeline) AddTest(test PerformanceTest) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	pp.testSuite = append(pp.testSuite, test)
}

func (pp *PerformancePipeline) Run(ctx context.Context) error {
	// Load baseline
	if err := pp.LoadBaseline(); err != nil {
		return fmt.Errorf("failed to load baseline: %w", err)
	}
	
	// Create results directory
	if err := os.MkdirAll(pp.config.ResultsPath, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}
	
	// Run tests
	for _, test := range pp.testSuite {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		result := pp.runTest(ctx, test)
		pp.mu.Lock()
		pp.results = append(pp.results, result)
		pp.mu.Unlock()
		
		// Analyze regression
		if regression := pp.analyzeRegression(result); regression != nil {
			result.Regression = regression
			
			// Send notification if regression detected
			if pp.config.Notifications.OnRegression {
				pp.reporter.SendNotification(result, regression)
			}
		}
	}
	
	// Generate reports
	if err := pp.reporter.GenerateReport(pp.results, pp.baseline); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}
	
	// Update baseline if all tests passed
	if pp.allTestsPassed() {
		if err := pp.updateBaseline(); err != nil {
			return fmt.Errorf("failed to update baseline: %w", err)
		}
	}
	
	return nil
}

func (pp *PerformancePipeline) runTest(ctx context.Context, test PerformanceTest) TestResult {
	result := TestResult{
		TestName:  test.Name,
		StartTime: time.Now(),
		Metrics:   make(map[string]interface{}),
		Errors:    make([]string, 0),
		Artifacts: make([]string, 0),
	}
	
	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()
	
	// Create test context with timeout
	testCtx, cancel := context.WithTimeout(ctx, test.Timeout)
	defer cancel()
	
	// Run prerequisites
	for _, prereq := range test.Prerequisites {
		if err := pp.runCommand(testCtx, prereq); err != nil {
			result.Status = "failed"
			result.Errors = append(result.Errors, fmt.Sprintf("prerequisite failed: %v", err))
			return result
		}
	}
	
	// Run the actual test based on type
	var err error
	switch test.Type {
	case "load":
		err = pp.runLoadTest(testCtx, test, &result)
	case "stress":
		err = pp.runStressTest(testCtx, test, &result)
	case "spike":
		err = pp.runSpikeTest(testCtx, test, &result)
	case "volume":
		err = pp.runVolumeTest(testCtx, test, &result)
	default:
		err = fmt.Errorf("unknown test type: %s", test.Type)
	}
	
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.Status = "passed"
	}
	
	// Run cleanup
	for _, cleanup := range test.Cleanup {
		pp.runCommand(testCtx, cleanup)
	}
	
	return result
}

func (pp *PerformancePipeline) runLoadTest(ctx context.Context, test PerformanceTest, result *TestResult) error {
	// Simulate load test execution
	// In a real implementation, this would integrate with the load testing framework
	time.Sleep(2 * time.Second) // Simulate test execution
	
	// Mock metrics
	result.Metrics["throughput"] = 150.5
	result.Metrics["avg_latency"] = 0.025
	result.Metrics["p95_latency"] = 0.045
	result.Metrics["p99_latency"] = 0.080
	result.Metrics["error_rate"] = 0.01
	result.Metrics["total_requests"] = float64(1000)
	result.Metrics["failed_requests"] = float64(10)
	
	return nil
}

func (pp *PerformancePipeline) runStressTest(ctx context.Context, test PerformanceTest, result *TestResult) error {
	// Simulate stress test execution
	time.Sleep(3 * time.Second)
	
	// Mock metrics
	result.Metrics["max_throughput"] = 200.0
	result.Metrics["breaking_point"] = float64(500) // concurrent users
	result.Metrics["recovery_time"] = 5.0 // seconds
	
	return nil
}

func (pp *PerformancePipeline) runSpikeTest(ctx context.Context, test PerformanceTest, result *TestResult) error {
	// Simulate spike test execution
	time.Sleep(2 * time.Second)
	
	// Mock metrics
	result.Metrics["spike_throughput"] = 180.0
	result.Metrics["spike_latency"] = 0.055
	result.Metrics["recovery_time"] = 3.0
	
	return nil
}

func (pp *PerformancePipeline) runVolumeTest(ctx context.Context, test PerformanceTest, result *TestResult) error {
	// Simulate volume test execution
	time.Sleep(4 * time.Second)
	
	// Mock metrics
	result.Metrics["data_processed"] = 1000000.0 // bytes
	result.Metrics["memory_usage"] = 512.0 // MB
	result.Metrics["disk_usage"] = 1024.0 // MB
	
	return nil
}

func (pp *PerformancePipeline) analyzeRegression(result TestResult) *RegressionAnalysis {
	regression := &RegressionAnalysis{
		Comparison:      make(map[string]MetricComparison),
		AffectedMetrics: make([]string, 0),
	}
	
	for metricName, currentValue := range result.Metrics {
		if currentFloat, ok := currentValue.(float64); ok {
			if baselineMetric, exists := pp.baseline.Metrics[metricName]; exists {
				change := currentFloat - baselineMetric.Value
				changePercent := (change / baselineMetric.Value) * 100
				withinTolerance := abs(changePercent) <= baselineMetric.Tolerance
				
				comparison := MetricComparison{
					Current:         currentFloat,
					Baseline:        baselineMetric.Value,
					Change:          change,
					ChangePercent:   changePercent,
					WithinTolerance: withinTolerance,
				}
				
				regression.Comparison[metricName] = comparison
				
				if !withinTolerance {
					regression.HasRegression = true
					regression.AffectedMetrics = append(regression.AffectedMetrics, metricName)
				}
			}
		}
	}
	
	if regression.HasRegression {
		if len(regression.AffectedMetrics) > 2 {
			regression.Severity = "critical"
		} else {
			regression.Severity = "warning"
		}
		regression.Recommendation = "Review recent changes and consider rollback"
	}
	
	if !regression.HasRegression {
		return nil
	}
	
	return regression
}

func (pp *PerformancePipeline) allTestsPassed() bool {
	for _, result := range pp.results {
		if result.Status != "passed" {
			return false
		}
	}
	return true
}

func (pp *PerformancePipeline) updateBaseline() error {
	newBaseline := PerformanceBaseline{
		Version:     fmt.Sprintf("v%d", time.Now().Unix()),
		Timestamp:   time.Now(),
		Environment: pp.config.Environment,
		Metrics:     make(map[string]BaselineMetric),
	}
	
	// Aggregate metrics from all test results
	for _, result := range pp.results {
		for metricName, value := range result.Metrics {
			if floatValue, ok := value.(float64); ok {
				newBaseline.Metrics[metricName] = BaselineMetric{
					Name:      metricName,
					Value:     floatValue,
					Unit:      pp.getMetricUnit(metricName),
					Tolerance: pp.getMetricTolerance(metricName),
					Trend:     "stable",
				}
			}
		}
	}
	
	// Save baseline
	baselinePath := filepath.Join(pp.config.BaselinePath, "baseline.json")
	if err := os.MkdirAll(pp.config.BaselinePath, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(newBaseline, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(baselinePath, data, 0644)
}

func (pp *PerformancePipeline) getMetricUnit(metricName string) string {
	units := map[string]string{
		"throughput":   "req/s",
		"avg_latency":  "seconds",
		"p95_latency":  "seconds",
		"p99_latency":  "seconds",
		"memory_usage": "bytes",
		"cpu_usage":    "percent",
		"error_rate":   "percent",
	}
	
	if unit, exists := units[metricName]; exists {
		return unit
	}
	return "unknown"
}

func (pp *PerformancePipeline) getMetricTolerance(metricName string) float64 {
	tolerances := map[string]float64{
		"throughput":   5.0,  // 5% tolerance
		"avg_latency":  10.0, // 10% tolerance
		"p95_latency":  15.0, // 15% tolerance
		"p99_latency":  20.0, // 20% tolerance
		"memory_usage": 10.0, // 10% tolerance
		"cpu_usage":    15.0, // 15% tolerance
		"error_rate":   5.0,  // 5% tolerance
	}
	
	if tolerance, exists := tolerances[metricName]; exists {
		return tolerance
	}
	return 10.0 // Default 10% tolerance
}

func (pp *PerformancePipeline) runCommand(ctx context.Context, command string) error {
	// Implementation would run shell commands
	// This is a placeholder for the exercise
	return nil
}

func (pp *PerformancePipeline) GetResults() []TestResult {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	return append([]TestResult(nil), pp.results...)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}