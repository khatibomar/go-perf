package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"exercise-04-04/benchmark"
	"exercise-04-04/capacity"
	"exercise-04-04/sla"
)

func TestBenchmarkRunner(t *testing.T) {
	// Create test configuration
	config := &benchmark.BenchmarkConfig{
		Name:        "test-benchmark",
		Description: "Test benchmark configuration",
		Duration:    5 * time.Second,
		WarmupTime:  1 * time.Second,
		CooldownTime: 1 * time.Second,
		Targets: []benchmark.Target{
			{
				Name:        "test-server",
				Type:        "http",
				URL:         "http://httpbin.org",
				HealthCheck: "/status/200",
				Headers:     map[string]string{"User-Agent": "benchmark-test"},
			},
		},
		Scenarios: []benchmark.Scenario{
			{
				Name:        "basic-load",
				Description: "Basic load test",
				Concurrency: 2,
				Duration:    3 * time.Second,
				RampUpTime:  500 * time.Millisecond,
				RampDownTime: 500 * time.Millisecond,
				ThinkTime:   100 * time.Millisecond,
				RequestRate: 5,
				Requests: []benchmark.Request{
					{
						Name:   "get-request",
						Method: "GET",
						Path:   "/get",
						Weight: 1.0,
						Headers: map[string]string{"Accept": "application/json"},
						Validation: benchmark.Validation{
							StatusCode: 200,
						},
					},
				},
			},
		},
		Metrics: []benchmark.MetricConfig{
			{
				Name:        "avg-latency",
				Type:        "latency",
				Unit:        "ms",
				Aggregation: "mean",
				Threshold:   1000, // 1 second
				Critical:    true,
			},
			{
				Name:        "throughput",
				Type:        "throughput",
				Unit:        "rps",
				Aggregation: "sum",
				Threshold:   1, // At least 1 RPS
				Critical:    false,
			},
			{
				Name:        "error-rate",
				Type:        "error_rate",
				Unit:        "percent",
				Aggregation: "avg",
				Threshold:   5, // Max 5% error rate
				Critical:    true,
			},
		},
		Thresholds: benchmark.Thresholds{
			Latency: benchmark.LatencyThresholds{
				P50: 100, // 100ms
				P90: 200, // 200ms
				P95: 300, // 300ms
				P99: 500, // 500ms
				Max: 1000, // 1s
			},
			Throughput: benchmark.ThroughputThresholds{
				MinRPS: 1,
				MaxRPS: 100,
			},
			ErrorRate: benchmark.ErrorRateThresholds{
				MaxPercent: 5,
				MaxCount:   10,
			},
		},
		Baseline: benchmark.BaselineConfig{
			Enabled:     true,
			Path:        "test-baseline.json",
			Version:     "1.0.0",
			Environment: "test",
			AutoUpdate:  false,
		},
	}

	// Create temporary directories
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")
	baselineDir := filepath.Join(tempDir, "baseline")

	// Create runner
	runner := benchmark.NewRunner(config, outputDir, baselineDir)
	runner.SetVerbose(true)

	// Run benchmark
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := runner.RunBenchmarks(ctx)
	if err != nil {
		t.Fatalf("Benchmark failed: %v", err)
	}

	// Validate results
	if results == nil {
		t.Fatal("Results should not be nil")
	}

	if results.Name != config.Name {
		t.Errorf("Expected name %s, got %s", config.Name, results.Name)
	}

	if len(results.Targets) == 0 {
		t.Error("Expected at least one target result")
	}

	if len(results.Scenarios) == 0 {
		t.Error("Expected at least one scenario result")
	}

	if len(results.Metrics) == 0 {
		t.Error("Expected at least one metric result")
	}

	if results.Summary == nil {
		t.Error("Expected summary to be present")
	}

	// Check that some requests were made
	if results.Summary.TotalRequests == 0 {
		t.Error("Expected some requests to be made")
	}

	// Save results
	err = runner.SaveResults(results, "test-results.json")
	if err != nil {
		t.Errorf("Failed to save results: %v", err)
	}

	// Verify results file exists
	resultsPath := filepath.Join(outputDir, "test-results.json")
	if _, err := os.Stat(resultsPath); os.IsNotExist(err) {
		t.Error("Results file was not created")
	}
}

func TestSLAMonitor(t *testing.T) {
	// Create test SLA configuration
	config := &sla.SLAConfig{
		Name:        "test-sla-monitor",
		Description: "Test SLA monitoring",
		Enabled:     true,
		Interval:    1 * time.Second,
		SLAs: []sla.SLA{
			{
				Name:        "response-time-sla",
				Description: "Response time should be under 500ms",
				Enabled:     true,
				Metric:      "response_time",
				Objective: sla.SLAObjective{
					Type:      "latency",
					Target:    500, // 500ms
					Operator:  "<",
					Unit:      "ms",
					Threshold: 600, // Alert if over 600ms
				},
				Window:   5 * time.Minute,
				Severity: "warning",
				Tags:     []string{"performance", "latency"},
			},
			{
				Name:        "availability-sla",
				Description: "Service availability should be 99.9%",
				Enabled:     true,
				Metric:      "availability",
				Objective: sla.SLAObjective{
					Type:      "availability",
					Target:    99.9,
					Operator:  ">=",
					Unit:      "percent",
					Threshold: 99.0,
				},
				Window:   1 * time.Hour,
				Severity: "critical",
				Tags:     []string{"availability", "uptime"},
			},
		},
		Alerting: sla.AlertConfig{
			Enabled:  true,
			Cooldown: 5 * time.Minute,
			Channels: []sla.AlertChannel{
				{
					Name:    "console",
					Type:    "console",
					Enabled: true,
					Config:  map[string]interface{}{},
				},
				{
					Name:    "file",
					Type:    "file",
					Enabled: true,
					Config: map[string]interface{}{
						"filename": "test-alerts.json",
					},
				},
			},
			Rules: []sla.AlertRule{
				{
					Name:      "sla-violation-warning",
					Condition: "sla_violation",
					Severity:  "warning",
					Threshold: 1,
					Duration:  1 * time.Minute,
					Channels:  []string{"console", "file"},
					Cooldown:  5 * time.Minute,
					Enabled:   true,
				},
				{
					Name:      "sla-violation-critical",
					Condition: "sla_violation",
					Severity:  "critical",
					Threshold: 1,
					Duration:  30 * time.Second,
					Channels:  []string{"console", "file"},
					Cooldown:  2 * time.Minute,
					Enabled:   true,
				},
			},
		},
		Reporting: sla.ReportConfig{
			Enabled:   true,
			Interval:  1 * time.Hour,
			Format:    "json",
			Channels:  []string{"file"},
			Retention: 30 * 24 * time.Hour, // 30 days
		},
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Create monitor
	monitor := sla.NewSLAMonitor(config, tempDir)
	monitor.SetVerbose(true)

	// Update some metrics
	monitor.UpdateMetric("response_time", 300, "ms", map[string]string{"service": "test"})
	monitor.UpdateMetric("availability", 99.95, "percent", map[string]string{"service": "test"})

	// Test SLA violation
	monitor.UpdateMetric("response_time", 700, "ms", map[string]string{"service": "test"}) // Should trigger violation

	// Run a single check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start monitoring in background
	go func() {
		monitor.Start(ctx)
	}()

	// Wait a bit for monitoring to run
	time.Sleep(2 * time.Second)

	// Generate report
	report := monitor.GenerateReport()
	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if report.Name != config.Name {
		t.Errorf("Expected report name %s, got %s", config.Name, report.Name)
	}

	if len(report.SLAs) != len(config.SLAs) {
		t.Errorf("Expected %d SLAs in report, got %d", len(config.SLAs), len(report.SLAs))
	}

	if report.Summary == nil {
		t.Error("Expected summary to be present")
	}

	// Save report
	err := monitor.SaveReport(report, "test-sla-report.json")
	if err != nil {
		t.Errorf("Failed to save report: %v", err)
	}

	// Verify report file exists
	reportPath := filepath.Join(tempDir, "test-sla-report.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("Report file was not created")
	}
}

func TestCapacityPlanner(t *testing.T) {
	// Create test capacity planning configuration
	config := &capacity.PlanningConfig{
		Name:        "test-capacity-planner",
		Description: "Test capacity planning",
		Enabled:     true,
		Metrics: []capacity.MetricConfig{
			{
				Name:        "cpu_usage",
				Type:        "cpu",
				Unit:        "percent",
				Weight:      1.0,
				Critical:    true,
				Threshold:   80.0,
				Aggregation: "avg",
			},
			{
				Name:        "memory_usage",
				Type:        "memory",
				Unit:        "percent",
				Weight:      0.8,
				Critical:    true,
				Threshold:   85.0,
				Aggregation: "avg",
			},
			{
				Name:        "disk_usage",
				Type:        "disk",
				Unit:        "percent",
				Weight:      0.6,
				Critical:    false,
				Threshold:   90.0,
				Aggregation: "max",
			},
		},
		Models: []capacity.ModelConfig{
			{
				Name:       "linear-trend",
				Type:       "linear",
				Enabled:    true,
				Parameters: map[string]interface{}{},
				Horizon:    7 * 24 * time.Hour, // 7 days
				Confidence: 0.8,
			},
		},
		Scenarios: []capacity.ScenarioConfig{
			{
				Name:        "normal-growth",
				Description: "Normal growth scenario",
				Enabled:     true,
				GrowthRate:  0.05, // 5% growth
				Timeframe:   30 * 24 * time.Hour, // 30 days
				Factors:     map[string]interface{}{"seasonal": false},
			},
			{
				Name:        "high-growth",
				Description: "High growth scenario",
				Enabled:     true,
				GrowthRate:  0.15, // 15% growth
				Timeframe:   30 * 24 * time.Hour, // 30 days
				Factors:     map[string]interface{}{"seasonal": true},
			},
		},
		Thresholds: capacity.ThresholdConfig{
			CPU: capacity.ResourceThreshold{
				Warning:  70.0,
				Critical: 85.0,
				Max:      100.0,
			},
			Memory: capacity.ResourceThreshold{
				Warning:  75.0,
				Critical: 90.0,
				Max:      100.0,
			},
			Disk: capacity.ResourceThreshold{
				Warning:  80.0,
				Critical: 95.0,
				Max:      100.0,
			},
			Latency: capacity.LatencyThreshold{
				P50Warning:  100,
				P50Critical: 200,
				P95Warning:  500,
				P95Critical: 1000,
				P99Warning:  1000,
				P99Critical: 2000,
			},
		},
		Reporting: capacity.ReportingConfig{
			Enabled:   true,
			Interval:  24 * time.Hour, // Daily
			Format:    "json",
			Retention: 90 * 24 * time.Hour, // 90 days
		},
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Create planner
	planner := capacity.NewCapacityPlanner(config, tempDir)
	planner.SetVerbose(true)

	// Add some test data points
	now := time.Now()
	for i := 0; i < 10; i++ {
		timestamp := now.Add(time.Duration(-i) * time.Hour)
		metrics := map[string]float64{
			"cpu_usage":    50.0 + float64(i)*2.0, // Increasing trend
			"memory_usage": 60.0 + float64(i)*1.5, // Increasing trend
			"disk_usage":   70.0 + float64(i)*0.5, // Slow increase
		}
		metadata := map[string]interface{}{
			"host": "test-server",
			"env":  "test",
		}
		planner.AddDataPoint(timestamp, metrics, metadata)
	}

	// Run analysis
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := planner.RunAnalysis(ctx)
	if err != nil {
		t.Fatalf("Capacity analysis failed: %v", err)
	}

	// Validate report
	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if report.Name != config.Name {
		t.Errorf("Expected report name %s, got %s", config.Name, report.Name)
	}

	if len(report.Predictions) == 0 {
		t.Error("Expected at least one prediction")
	}

	if len(report.Scenarios) != len(config.Scenarios) {
		t.Errorf("Expected %d scenarios in report, got %d", len(config.Scenarios), len(report.Scenarios))
	}

	if report.Summary == nil {
		t.Error("Expected summary to be present")
	}

	// Check that predictions have values
	for name, prediction := range report.Predictions {
		if len(prediction.Values) == 0 {
			t.Errorf("Prediction %s should have values", name)
		}
		if prediction.Trend == "" {
			t.Errorf("Prediction %s should have a trend", name)
		}
	}

	// Save report
	err = planner.SaveReport(report, "test-capacity-report.json")
	if err != nil {
		t.Errorf("Failed to save report: %v", err)
	}

	// Verify report file exists
	reportPath := filepath.Join(tempDir, "test-capacity-report.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("Report file was not created")
	}
}

func TestBenchmarkConfigLoadSave(t *testing.T) {
	// Create test configuration
	originalConfig := &benchmark.BenchmarkConfig{
		Name:        "test-config",
		Description: "Test configuration for load/save",
		Duration:    10 * time.Second,
		Targets: []benchmark.Target{
			{
				Name: "test-target",
				Type: "http",
				URL:  "http://example.com",
			},
		},
		Scenarios: []benchmark.Scenario{
			{
				Name:        "test-scenario",
				Concurrency: 5,
				Requests: []benchmark.Request{
					{
						Name:   "test-request",
						Method: "GET",
						Path:   "/",
					},
				},
			},
		},
	}

	// Create temporary file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	// Save configuration
	err := benchmark.SaveBenchmarkConfig(originalConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load configuration
	loadedConfig, err := benchmark.LoadBenchmarkConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Compare configurations
	if loadedConfig.Name != originalConfig.Name {
		t.Errorf("Expected name %s, got %s", originalConfig.Name, loadedConfig.Name)
	}

	if loadedConfig.Description != originalConfig.Description {
		t.Errorf("Expected description %s, got %s", originalConfig.Description, loadedConfig.Description)
	}

	if loadedConfig.Duration != originalConfig.Duration {
		t.Errorf("Expected duration %v, got %v", originalConfig.Duration, loadedConfig.Duration)
	}

	if len(loadedConfig.Targets) != len(originalConfig.Targets) {
		t.Errorf("Expected %d targets, got %d", len(originalConfig.Targets), len(loadedConfig.Targets))
	}

	if len(loadedConfig.Scenarios) != len(originalConfig.Scenarios) {
		t.Errorf("Expected %d scenarios, got %d", len(originalConfig.Scenarios), len(loadedConfig.Scenarios))
	}
}

func TestBaselineManagement(t *testing.T) {
	// Create test results
	results := &benchmark.BenchmarkResults{
		Name:      "test-benchmark",
		StartTime: time.Now().Add(-5 * time.Minute),
		EndTime:   time.Now(),
		Duration:  5 * time.Minute,
		Metrics: map[string]*benchmark.MetricResults{
			"latency": {
				Name:     "latency",
				Type:     "latency",
				Unit:     "ms",
				Value:    150.0,
				Critical: true,
				Passed:   true,
			},
			"throughput": {
				Name:     "throughput",
				Type:     "throughput",
				Unit:     "rps",
				Value:    25.0,
				Critical: false,
				Passed:   true,
			},
		},
		Summary: &benchmark.ResultSummary{
			OverallPassed:   true,
			TotalRequests:   1000,
			SuccessRate:     99.5,
			AvgThroughput:   25.0,
			AvgLatency:      150 * time.Millisecond,
			P95Latency:      200 * time.Millisecond,
			ErrorRate:       0.5,
			Recommendations: []string{},
		},
	}

	// Create test config
	config := &benchmark.BenchmarkConfig{
		Name: "test-benchmark",
		Baseline: benchmark.BaselineConfig{
			Enabled:     true,
			Version:     "1.0.0",
			Environment: "test",
		},
	}

	// Create temporary directory
	tempDir := t.TempDir()
	baselinePath := filepath.Join(tempDir, "baseline.json")

	// Create baseline
	err := benchmark.CreateBaseline(results, config, baselinePath)
	if err != nil {
		t.Fatalf("Failed to create baseline: %v", err)
	}

	// Verify baseline file exists
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		t.Error("Baseline file was not created")
	}

	// Load baseline
	baseline, err := benchmark.LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("Failed to load baseline: %v", err)
	}

	// Validate baseline
	if baseline.Name != config.Name {
		t.Errorf("Expected baseline name %s, got %s", config.Name, baseline.Name)
	}

	if baseline.Version != config.Baseline.Version {
		t.Errorf("Expected baseline version %s, got %s", config.Baseline.Version, baseline.Version)
	}

	if len(baseline.Metrics) != len(results.Metrics) {
		t.Errorf("Expected %d baseline metrics, got %d", len(results.Metrics), len(baseline.Metrics))
	}

	// Check specific metrics
	if latencyMetric, exists := baseline.Metrics["latency"]; exists {
		if latencyMetric.Value != 150.0 {
			t.Errorf("Expected latency baseline value 150.0, got %f", latencyMetric.Value)
		}
		if latencyMetric.Type != "latency" {
			t.Errorf("Expected latency baseline type 'latency', got %s", latencyMetric.Type)
		}
	} else {
		t.Error("Expected latency metric in baseline")
	}
}

func TestIntegration_BenchmarkToSLA(t *testing.T) {
	// This test demonstrates how benchmark results can feed into SLA monitoring

	// Create benchmark config
	benchmarkConfig := &benchmark.BenchmarkConfig{
		Name:     "integration-test",
		Duration: 3 * time.Second,
		Targets: []benchmark.Target{
			{
				Name: "test-service",
				Type: "http",
				URL:  "http://httpbin.org",
			},
		},
		Scenarios: []benchmark.Scenario{
			{
				Name:        "load-test",
				Concurrency: 2,
				Duration:    2 * time.Second,
				Requests: []benchmark.Request{
					{
						Name:   "get-test",
						Method: "GET",
						Path:   "/get",
					},
				},
			},
		},
		Metrics: []benchmark.MetricConfig{
			{
				Name:        "response-time",
				Type:        "latency",
				Unit:        "ms",
				Aggregation: "p95",
			},
		},
	}

	// Create SLA config
	slaConfig := &sla.SLAConfig{
		Name:     "integration-sla",
		Enabled:  true,
		Interval: 1 * time.Second,
		SLAs: []sla.SLA{
			{
				Name:    "response-time-sla",
				Enabled: true,
				Metric:  "response_time",
				Objective: sla.SLAObjective{
					Type:     "latency",
					Target:   500,
					Operator: "<",
					Unit:     "ms",
				},
				Severity: "warning",
			},
		},
		Alerting: sla.AlertConfig{
			Enabled: true,
			Channels: []sla.AlertChannel{
				{
					Name:    "console",
					Type:    "console",
					Enabled: true,
				},
			},
			Rules: []sla.AlertRule{
				{
					Name:      "latency-violation",
					Condition: "sla_violation",
					Severity:  "warning",
					Channels:  []string{"console"},
					Enabled:   true,
				},
			},
		},
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Run benchmark
	runner := benchmark.NewRunner(benchmarkConfig, tempDir, tempDir)
	runner.SetVerbose(false) // Reduce noise in tests

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	results, err := runner.RunBenchmarks(ctx)
	if err != nil {
		t.Fatalf("Benchmark failed: %v", err)
	}

	// Extract metrics from benchmark results
	monitor := sla.NewSLAMonitor(slaConfig, tempDir)
	monitor.SetVerbose(false)

	// Feed benchmark results into SLA monitor
	for metricName, metric := range results.Metrics {
		// Map benchmark metrics to SLA metrics
		slaMetricName := "response_time" // Map to SLA metric name
		if metricName == "response-time" {
			monitor.UpdateMetric(slaMetricName, metric.Value, metric.Unit, map[string]string{
				"source": "benchmark",
				"test":   "integration",
			})
		}
	}

	// Generate SLA report
	slaReport := monitor.GenerateReport()
	if slaReport == nil {
		t.Fatal("SLA report should not be nil")
	}

	// Verify integration
	if len(slaReport.SLAs) == 0 {
		t.Error("Expected SLA status in report")
	}

	if slaReport.Summary == nil {
		t.Error("Expected SLA summary")
	}

	t.Logf("Integration test completed successfully")
	t.Logf("Benchmark: %d requests, %.2f%% success rate", results.Summary.TotalRequests, results.Summary.SuccessRate)
	t.Logf("SLA: %s overall health, %d SLAs monitored", slaReport.Summary.OverallHealth, slaReport.Summary.TotalSLAs)
}