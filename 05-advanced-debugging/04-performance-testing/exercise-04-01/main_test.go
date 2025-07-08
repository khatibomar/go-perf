package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"exercise-04-01/loadtest"
	"exercise-04-01/pipeline"
	"exercise-04-01/reporter"
)

func TestLoadTester_BasicFunctionality(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate some processing time
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Create load test configuration
	config := loadtest.LoadTestConfig{
		TargetURL:      server.URL,
		MaxConcurrency: 5,
		Duration:       2 * time.Second,
		RampUpTime:     500 * time.Millisecond,
		Scenarios: []loadtest.TestScenario{
			{
				Name:   "basic_test",
				Weight: 1.0,
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:   "/",
						Headers: map[string]string{
							"User-Agent": "LoadTester/1.0",
						},
					},
				},
			},
		},
		Assertions: []loadtest.Assertion{
			loadtest.AssertMaxLatency(100 * time.Millisecond),
			loadtest.AssertMaxErrorRate(0.05),
		},
	}

	// Run the load test
	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()

	// Verify results
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.TotalRequests == 0 {
		t.Error("Expected some requests to be made")
	}

	if result.SuccessRequests == 0 {
		t.Error("Expected some successful requests")
	}

	if result.Throughput <= 0 {
		t.Error("Expected positive throughput")
	}

	if result.Latencies.Mean <= 0 {
		t.Error("Expected positive mean latency")
	}

	if len(result.Scenarios) != 1 {
		t.Errorf("Expected 1 scenario result, got %d", len(result.Scenarios))
	}

	scenario, exists := result.Scenarios["basic_test"]
	if !exists {
		t.Error("Expected scenario 'basic_test' not found")
		return
	}

	if scenario.Name != "basic_test" {
		t.Errorf("Expected scenario name 'basic_test', got '%s'", scenario.Name)
	}

	if scenario.ExecutedUsers != 5 {
		t.Errorf("Expected 5 executed users, got %d", scenario.ExecutedUsers)
	}
}

func TestLoadTester_ErrorHandling(t *testing.T) {
	// Create a test server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := loadtest.LoadTestConfig{
		TargetURL:      server.URL,
		MaxConcurrency: 2,
		Duration:       1 * time.Second,
		RampUpTime:     200 * time.Millisecond,
		Scenarios: []loadtest.TestScenario{
			{
				Name:   "error_test",
				Weight: 1.0,
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:   "/",
					},
				},
			},
		},
	}

	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()

	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	if result.FailedRequests == 0 {
		t.Error("Expected some failed requests")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected some errors to be recorded")
	}
}

func TestLoadTester_Assertions(t *testing.T) {
	// Create a slow test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Slow response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := loadtest.LoadTestConfig{
		TargetURL:      server.URL,
		MaxConcurrency: 2,
		Duration:       1 * time.Second,
		RampUpTime:     200 * time.Millisecond,
		Scenarios: []loadtest.TestScenario{
			{
				Name:   "slow_test",
				Weight: 1.0,
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:   "/",
					},
				},
			},
		},
		Assertions: []loadtest.Assertion{
			loadtest.AssertMaxLatency(50 * time.Millisecond), // This should fail
		},
	}

	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()

	// The test should complete but assertions should fail
	if err == nil {
		t.Error("Expected assertion failures")
	}

	if result == nil {
		t.Fatal("Expected non-nil result even with assertion failures")
	}
}

func TestReporter_JSONReport(t *testing.T) {
	// Create test data
	result := &loadtest.LoadTestResult{
		StartTime:        time.Now().Add(-5 * time.Second),
		EndTime:          time.Now(),
		TotalRequests:    100,
		SuccessRequests:  95,
		FailedRequests:   5,
		Throughput:       20.0,
		Latencies: loadtest.LatencyStats{
			Min:  10 * time.Millisecond,
			Max:  100 * time.Millisecond,
			Mean: 50 * time.Millisecond,
			P50:  45 * time.Millisecond,
			P90:  80 * time.Millisecond,
			P95:  90 * time.Millisecond,
			P99:  95 * time.Millisecond,
			P999: 99 * time.Millisecond,
		},
		Errors: map[string]int64{
			"timeout": 3,
			"500":     2,
		},
		Scenarios: map[string]loadtest.ScenarioResult{
			"test_scenario": {
				Name:            "test_scenario",
				ExecutedUsers:   5,
				TotalRequests:   100,
				SuccessRequests: 95,
				FailedRequests:  5,
				Throughput:      20.0,
				AvgLatency:      50 * time.Millisecond,
				Latencies:       []time.Duration{30 * time.Millisecond, 50 * time.Millisecond, 90 * time.Millisecond, 95 * time.Millisecond},
				Errors:          map[string]int64{"timeout": 5},
			},
		},
	}

	// Create temporary directory for reports
	tempDir := t.TempDir()

	// Test JSON reporter
	jsonReporter := reporter.NewReporter("json")
	config := reporter.ReportConfig{
		OutputPath:  tempDir,
		Format:      "json",
		Title:       "Test Report",
		Description: "Test Description",
	}

	err := jsonReporter.GenerateReport(result, config)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify the report file exists
	reportFile := filepath.Join(tempDir, "load_test_report.json")
	if _, err := os.Stat(reportFile); os.IsNotExist(err) {
		t.Errorf("Report file was not created: %s", reportFile)
	}

	// Verify the report content
	data, err := os.ReadFile(reportFile)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	var reportData map[string]interface{}
	if err := json.Unmarshal(data, &reportData); err != nil {
		t.Fatalf("Failed to parse report JSON: %v", err)
	}

	if reportData["Title"] != "Test Report" {
		t.Errorf("Expected title 'Test Report', got %v", reportData["Title"])
	}
}

func TestReporter_HTMLReport(t *testing.T) {
	result := &loadtest.LoadTestResult{
		StartTime:        time.Now().Add(-5 * time.Second),
		EndTime:          time.Now(),
		TotalRequests:    50,
		SuccessRequests:  48,
		FailedRequests:   2,
		Throughput:       10.0,
		Latencies: loadtest.LatencyStats{
			Min:  5 * time.Millisecond,
			Max:  200 * time.Millisecond,
			Mean: 75 * time.Millisecond,
			P50:  70 * time.Millisecond,
			P90:  150 * time.Millisecond,
			P95:  180 * time.Millisecond,
			P99:  190 * time.Millisecond,
			P999: 195 * time.Millisecond,
		},
		Errors:    map[string]int64{},
		Scenarios: map[string]loadtest.ScenarioResult{},
	}

	tempDir := t.TempDir()

	htmlReporter := reporter.NewReporter("html")
	config := reporter.ReportConfig{
		OutputPath:  tempDir,
		Format:      "html",
		Title:       "HTML Test Report",
		Description: "HTML Test Description",
	}

	err := htmlReporter.GenerateReport(result, config)
	if err != nil {
		t.Fatalf("Failed to generate HTML report: %v", err)
	}

	reportFile := filepath.Join(tempDir, "load_test_report.html")
	if _, err := os.Stat(reportFile); os.IsNotExist(err) {
		t.Errorf("HTML report file was not created: %s", reportFile)
	}

	// Verify the HTML content contains expected elements
	data, err := os.ReadFile(reportFile)
	if err != nil {
		t.Fatalf("Failed to read HTML report file: %v", err)
	}

	content := string(data)
	if !contains(content, "HTML Test Report") {
		t.Error("HTML report should contain the title")
	}

	if !contains(content, "HTML Test Description") {
		t.Error("HTML report should contain the description")
	}

	if !contains(content, "Total Requests") {
		t.Error("HTML report should contain metrics")
	}
}

func TestPipeline_BasicFunctionality(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	baselinePath := filepath.Join(tempDir, "baseline.json")
	reportPath := tempDir

	// Create pipeline configuration
	pipelineConfig := pipeline.PipelineConfig{
		Name:         "Test Pipeline",
		Description:  "Test pipeline description",
		BaselinePath: baselinePath,
		ReportPath:   reportPath,
		Notification: pipeline.NotificationConfig{
			Enabled: false,
		},
		RegressionRules: pipeline.RegressionRules{
			LatencyThreshold:    20.0,  // 20% increase
			ThroughputThreshold: 10.0,  // 10% decrease
			ErrorRateThreshold:  5.0,   // 5% increase
		},
	}

	// Create performance test
	perfTest := &pipeline.PerformanceTest{
		Name:        "Basic Performance Test",
		Description: "Basic test description",
		Config: &loadtest.LoadTestConfig{
			TargetURL:      server.URL,
			MaxConcurrency: 5,
			Duration:       2 * time.Second,
			RampUpTime:     500 * time.Millisecond,
			Scenarios: []loadtest.TestScenario{
			{
				Name:   "basic_scenario",
				Weight: 1.0,
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:   "/",
					},
				},
			},
		},
		},
		Expected: pipeline.ExpectedResults{
			MaxLatencyP95:  100 * time.Millisecond,
			MinThroughput:  5.0,
			MaxErrorRate:   5.0,
			MinSuccessRate: 95.0,
		},
	}

	// Create and run pipeline
	pipe := pipeline.NewPerformancePipeline(pipelineConfig)
	err := pipe.RunPipeline([]*pipeline.PerformanceTest{perfTest}, "v1.0.0")

	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	// Verify baseline was created
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		t.Errorf("Baseline file was not created: %s", baselinePath)
	}

	// Verify report was generated
	reportFile := filepath.Join(reportPath, "load_test_report.html")
	if _, err := os.Stat(reportFile); os.IsNotExist(err) {
		t.Errorf("Report file was not created: %s", reportFile)
	}
}

func TestPipeline_RegressionDetection(t *testing.T) {
	tempDir := t.TempDir()
	baselinePath := filepath.Join(tempDir, "baseline.json")

	// Create a baseline manually
	baseline := pipeline.PerformanceBaseline{
		Version:   "v1.0.0",
		Timestamp: time.Now(),
		Metrics: map[string]pipeline.BaselineMetric{
			"overall": {
				Name:        "overall",
				LatencyP95:  50 * time.Millisecond,
				LatencyP99:  80 * time.Millisecond,
				Throughput:  100.0,
				ErrorRate:   1.0,
				SuccessRate: 99.0,
			},
		},
	}

	data, _ := json.MarshalIndent(baseline, "", "  ")
	os.WriteFile(baselinePath, data, 0644)

	// Create pipeline with regression rules
	pipelineConfig := pipeline.PipelineConfig{
		Name:         "Regression Test Pipeline",
		Description:  "Testing regression detection",
		BaselinePath: baselinePath,
		RegressionRules: pipeline.RegressionRules{
			LatencyThreshold:    10.0, // 10% increase threshold
			ThroughputThreshold: 10.0, // 10% decrease threshold
			ErrorRateThreshold:  2.0,  // 2% increase threshold
		},
	}

	pipe := pipeline.NewPerformancePipeline(pipelineConfig)
	err := pipe.LoadBaseline()
	if err != nil {
		t.Fatalf("Failed to load baseline: %v", err)
	}

	// Create a result that should trigger regression
	regressionResult := &loadtest.LoadTestResult{
		StartTime:        time.Now().Add(-5 * time.Second),
		EndTime:          time.Now(),
		TotalRequests:    100,
		SuccessRequests:  85, // Lower success rate
		FailedRequests:   15, // Higher failure rate
		Throughput:       80.0, // Lower throughput (20% decrease)
		Latencies: loadtest.LatencyStats{
			P95: 70 * time.Millisecond, // Higher latency (40% increase)
			P99: 100 * time.Millisecond,
		},
		Scenarios: map[string]loadtest.ScenarioResult{},
	}

	analysis := pipe.AnalyzeRegression(regressionResult)

	if !analysis.HasRegression {
		t.Error("Expected regression to be detected")
	}

	if len(analysis.Comparisons) == 0 {
		t.Error("Expected comparison results")
	}

	// Check that at least one comparison shows regression
	found := false
	for _, comp := range analysis.Comparisons {
		if comp.IsRegression {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected at least one metric to show regression")
	}
}

func TestLoadTester_MultipleScenarios(t *testing.T) {
	// Create test servers with different response times
	fastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Fast"))
	}))
	defer fastServer.Close()

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Slow"))
		} else {
			time.Sleep(5 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Fast"))
		}
	}))
	defer slowServer.Close()

	config := loadtest.LoadTestConfig{
		TargetURL:      slowServer.URL,
		MaxConcurrency: 5,
		Duration:       3 * time.Second,
		RampUpTime:     500 * time.Millisecond,
		Scenarios: []loadtest.TestScenario{
			{
				Name:   "fast_scenario",
				Weight: 0.6,
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:   "/",
					},
				},
			},
			{
				Name:   "slow_scenario",
				Weight: 0.4,
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:   "/slow",
					},
				},
			},
		},
	}

	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()

	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	if len(result.Scenarios) != 2 {
		t.Errorf("Expected 2 scenario results, got %d", len(result.Scenarios))
	}

	// Find scenarios by name
	var fastScenario, slowScenario *loadtest.ScenarioResult
	if fs, exists := result.Scenarios["fast_scenario"]; exists {
		fastScenario = &fs
	}
	if ss, exists := result.Scenarios["slow_scenario"]; exists {
		slowScenario = &ss
	}

	if fastScenario == nil {
		t.Error("Fast scenario result not found")
	}
	if slowScenario == nil {
		t.Error("Slow scenario result not found")
	}

	if fastScenario != nil && slowScenario != nil {
		// Fast scenario should have better latency
		if fastScenario.AvgLatency >= slowScenario.AvgLatency {
			t.Errorf("Expected fast scenario to have better latency: fast=%v, slow=%v", 
				fastScenario.AvgLatency, slowScenario.AvgLatency)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
				s[len(s)-len(substr):] == substr || 
				containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIntegration_FullWorkflow(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Response at %v", time.Now())))
	}))
	defer server.Close()

	tempDir := t.TempDir()

	// Step 1: Create and run a load test
	config := loadtest.LoadTestConfig{
		TargetURL:      server.URL,
		MaxConcurrency: 4,
		Duration:       2 * time.Second,
		RampUpTime:     500 * time.Millisecond,
		Scenarios: []loadtest.TestScenario{
			{
				Name:   "integration_test",
				Weight: 1.0,
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:   "/api/test",
						Headers: map[string]string{
							"Content-Type": "application/json",
							"User-Agent":   "LoadTester/1.0",
						},
					},
					{
						Method: "POST",
						Path:   "/api/data",
						Body:   []byte(`{"test": "data"}`),
						Headers: map[string]string{
							"Content-Type": "application/json",
						},
					},
				},
			},
		},
		Assertions: []loadtest.Assertion{
			loadtest.AssertMaxLatency(200 * time.Millisecond),
			loadtest.AssertMaxErrorRate(10.0),
		},
	}

	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	// Step 2: Generate reports in multiple formats
	reportConfig := reporter.ReportConfig{
		OutputPath:  tempDir,
		Title:       "Integration Test Report",
		Description: "Full workflow integration test",
	}

	// JSON Report
	jsonReporter := reporter.NewReporter("json")
	err = jsonReporter.GenerateReport(result, reportConfig)
	if err != nil {
		t.Errorf("Failed to generate JSON report: %v", err)
	}

	// HTML Report
	htmlReporter := reporter.NewReporter("html")
	err = htmlReporter.GenerateReport(result, reportConfig)
	if err != nil {
		t.Errorf("Failed to generate HTML report: %v", err)
	}

	// CSV Report
	csvReporter := reporter.NewReporter("csv")
	err = csvReporter.GenerateReport(result, reportConfig)
	if err != nil {
		t.Errorf("Failed to generate CSV report: %v", err)
	}

	// Step 3: Run through pipeline
	pipelineConfig := pipeline.PipelineConfig{
		Name:         "Integration Pipeline",
		Description:  "Full integration test pipeline",
		BaselinePath: filepath.Join(tempDir, "integration_baseline.json"),
		ReportPath:   tempDir,
		Notification: pipeline.NotificationConfig{Enabled: false},
		RegressionRules: pipeline.RegressionRules{
			LatencyThreshold:    25.0,
			ThroughputThreshold: 15.0,
			ErrorRateThreshold:  5.0,
		},
	}

	perfTest := &pipeline.PerformanceTest{
		Name:        "Integration Performance Test",
		Description: "Integration test through pipeline",
		Config:      &config,
		Expected: pipeline.ExpectedResults{
			MaxLatencyP95:  150 * time.Millisecond,
			MaxErrorRate:   5.0,
			MinSuccessRate: 95.0,
		},
	}

	pipe := pipeline.NewPerformancePipeline(pipelineConfig)
	err = pipe.RunPipeline([]*pipeline.PerformanceTest{perfTest}, "v1.0.0")
	if err != nil {
		t.Errorf("Pipeline execution failed: %v", err)
	}

	// Verify all expected files were created
	expectedFiles := []string{
		"load_test_report.json",
		"load_test_report.html",
		"load_test_report.csv",
		"integration_baseline.json",
	}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file was not created: %s", filename)
		}
	}

	// Verify test results meet basic expectations
	if result.TotalRequests == 0 {
		t.Error("Expected some requests to be executed")
	}

	if result.Throughput <= 0 {
		t.Error("Expected positive throughput")
	}

	if result.Latencies.Mean <= 0 {
		t.Error("Expected positive mean latency")
	}

	if len(result.Scenarios) == 0 {
		t.Error("Expected scenario results")
	}

	t.Logf("Integration test completed successfully:")
	t.Logf("  Total Requests: %d", result.TotalRequests)
	t.Logf("  Success Rate: %.2f%%", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	t.Logf("  Throughput: %.2f req/s", result.Throughput)
	t.Logf("  Mean Latency: %v", result.Latencies.Mean)
	t.Logf("  P95 Latency: %v", result.Latencies.P95)
	t.Logf("  P99 Latency: %v", result.Latencies.P99)
}