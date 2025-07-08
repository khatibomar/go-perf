// main_test.go
package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"exercise-04-02/cicd"
	"exercise-04-02/pipeline"
)

func TestPerformancePipeline_BasicFunctionality(t *testing.T) {
	// Create temporary directory for test artifacts
	tempDir := t.TempDir()
	
	config := pipeline.PipelineConfig{
		Environment:   "test",
		TestTimeout:   30 * time.Second,
		ParallelTests: 1,
		ResultsPath:   filepath.Join(tempDir, "results"),
		BaselinePath:  filepath.Join(tempDir, "baseline"),
		ReportFormats: []string{"json"},
		Notifications: pipeline.NotificationConfig{
			OnFailure:    false,
			OnRegression: false,
			OnSuccess:    false,
		},
	}
	
	pp := pipeline.NewPerformancePipeline(config)
	
	// Add a test
	pp.AddTest(pipeline.PerformanceTest{
		Name:         "TestLoadTest",
		Type:         "load",
		Config:       map[string]interface{}{"duration": "10s", "users": 5},
		Timeout:      15 * time.Second,
		RetryCount:   1,
		CriticalPath: true,
	})
	
	// Run the pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	
	err := pp.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}
	
	// Verify results
	results := pp.GetResults()
	if len(results) != 1 {
		t.Fatalf("Expected 1 test result, got %d", len(results))
	}
	
	result := results[0]
	if result.TestName != "TestLoadTest" {
		t.Errorf("Expected test name 'TestLoadTest', got '%s'", result.TestName)
	}
	
	if result.Status != "passed" {
		t.Errorf("Expected test status 'passed', got '%s'", result.Status)
	}
	
	// Verify metrics are populated
	if len(result.Metrics) == 0 {
		t.Error("Expected test metrics to be populated")
	}
	
	// Check for required metrics
	requiredMetrics := []string{"throughput", "avg_latency", "p95_latency"}
	for _, metric := range requiredMetrics {
		if _, exists := result.Metrics[metric]; !exists {
			t.Errorf("Expected metric '%s' to be present", metric)
		}
	}
}

func TestPerformancePipeline_MultipleTestTypes(t *testing.T) {
	tempDir := t.TempDir()
	
	config := pipeline.PipelineConfig{
		Environment:   "test",
		TestTimeout:   30 * time.Second,
		ParallelTests: 1,
		ResultsPath:   filepath.Join(tempDir, "results"),
		BaselinePath:  filepath.Join(tempDir, "baseline"),
		ReportFormats: []string{"json"},
		Notifications: pipeline.NotificationConfig{},
	}
	
	pp := pipeline.NewPerformancePipeline(config)
	
	// Add different types of tests
	testTypes := []string{"load", "stress", "spike", "volume"}
	for _, testType := range testTypes {
		pp.AddTest(pipeline.PerformanceTest{
			Name:         "Test" + testType,
			Type:         testType,
			Config:       map[string]interface{}{"duration": "5s"},
			Timeout:      10 * time.Second,
			RetryCount:   1,
			CriticalPath: true,
		})
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	
	err := pp.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}
	
	results := pp.GetResults()
	if len(results) != len(testTypes) {
		t.Fatalf("Expected %d test results, got %d", len(testTypes), len(results))
	}
	
	// Verify all tests passed
	for _, result := range results {
		if result.Status != "passed" {
			t.Errorf("Test '%s' failed with status '%s'", result.TestName, result.Status)
		}
	}
}

func TestPerformancePipeline_BaselineManagement(t *testing.T) {
	tempDir := t.TempDir()
	
	config := pipeline.PipelineConfig{
		Environment:   "test",
		TestTimeout:   30 * time.Second,
		ParallelTests: 1,
		ResultsPath:   filepath.Join(tempDir, "results"),
		BaselinePath:  filepath.Join(tempDir, "baseline"),
		ReportFormats: []string{"json"},
		Notifications: pipeline.NotificationConfig{},
	}
	
	pp := pipeline.NewPerformancePipeline(config)
	
	pp.AddTest(pipeline.PerformanceTest{
		Name:         "BaselineTest",
		Type:         "load",
		Config:       map[string]interface{}{"duration": "5s"},
		Timeout:      10 * time.Second,
		RetryCount:   1,
		CriticalPath: true,
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	
	// First run - should create baseline
	err := pp.Run(ctx)
	if err != nil {
		t.Fatalf("First pipeline run failed: %v", err)
	}
	
	// Verify baseline file was created
	baselinePath := filepath.Join(tempDir, "baseline", "baseline.json")
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		t.Error("Baseline file was not created")
	}
	
	// Second run - should load existing baseline
	pp2 := pipeline.NewPerformancePipeline(config)
	pp2.AddTest(pipeline.PerformanceTest{
		Name:         "BaselineTest2",
		Type:         "load",
		Config:       map[string]interface{}{"duration": "5s"},
		Timeout:      10 * time.Second,
		RetryCount:   1,
		CriticalPath: true,
	})
	
	err = pp2.Run(ctx)
	if err != nil {
		t.Fatalf("Second pipeline run failed: %v", err)
	}
	
	results := pp2.GetResults()
	if len(results) != 1 {
		t.Fatalf("Expected 1 test result, got %d", len(results))
	}
	
	// The second run should have regression analysis
	// (even if no regression is detected)
	result := results[0]
	if result.Status != "passed" {
		t.Errorf("Expected test status 'passed', got '%s'", result.Status)
	}
}

func TestCICDIntegration_EnvironmentDetection(t *testing.T) {
	tempDir := t.TempDir()
	
	config := cicd.CICDConfig{
		Provider:         "github",
		Workspace:        tempDir,
		Branch:           "test-branch",
		CommitSHA:        "abc123",
		BuildNumber:      "42",
		ArtifactsPath:    filepath.Join(tempDir, "artifacts"),
		TestTimeout:      30 * time.Second,
		FailOnRegression: true,
		BaselineStrategy: "manual",
	}
	
	integration := cicd.NewCICDIntegration(config)
	
	// Test environment detection
	buildCtx := integration.DetectEnvironment()
	
	if buildCtx.Provider != "github" {
		t.Errorf("Expected provider 'github', got '%s'", buildCtx.Provider)
	}
	
	if buildCtx.Branch != "test-branch" {
		t.Errorf("Expected branch 'test-branch', got '%s'", buildCtx.Branch)
	}
	
	if buildCtx.CommitSHA != "abc123" {
		t.Errorf("Expected commit SHA 'abc123', got '%s'", buildCtx.CommitSHA)
	}
	
	if buildCtx.BuildNumber != "42" {
		t.Errorf("Expected build number '42', got '%s'", buildCtx.BuildNumber)
	}
}

func TestCICDIntegration_TestExecution(t *testing.T) {
	tempDir := t.TempDir()
	
	config := cicd.CICDConfig{
		Provider:         "github",
		Workspace:        tempDir,
		Branch:           "main",
		CommitSHA:        "def456",
		BuildNumber:      "43",
		ArtifactsPath:    filepath.Join(tempDir, "artifacts"),
		TestTimeout:      1 * time.Minute,
		FailOnRegression: false,
		BaselineStrategy: "manual",
	}
	
	integration := cicd.NewCICDIntegration(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	
	// Execute tests
	result, err := integration.ExecuteTests(ctx)
	if err != nil {
		t.Fatalf("Test execution failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected test execution result, got nil")
	}
	
	if result.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
	}
	
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	
	// Verify test results
	if len(result.TestResults) == 0 {
		t.Error("Expected test results to be populated")
	}
	
	// Verify artifacts directory was created
	artifactsDir := filepath.Join(tempDir, "artifacts")
	if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
		t.Error("Artifacts directory was not created")
	}
}

func TestCICDIntegration_PullRequestContext(t *testing.T) {
	tempDir := t.TempDir()
	
	// Set GitHub PR environment variables
	os.Setenv("GITHUB_EVENT_PULL_REQUEST_NUMBER", "123")
	os.Setenv("GITHUB_EVENT_PULL_REQUEST_TITLE", "Test PR")
	os.Setenv("GITHUB_HEAD_REF", "feature-branch")
	os.Setenv("GITHUB_BASE_REF", "main")
	os.Setenv("GITHUB_EVENT_PULL_REQUEST_USER_LOGIN", "testuser")
	defer func() {
		os.Unsetenv("GITHUB_EVENT_PULL_REQUEST_NUMBER")
		os.Unsetenv("GITHUB_EVENT_PULL_REQUEST_TITLE")
		os.Unsetenv("GITHUB_HEAD_REF")
		os.Unsetenv("GITHUB_BASE_REF")
		os.Unsetenv("GITHUB_EVENT_PULL_REQUEST_USER_LOGIN")
	}()
	
	config := cicd.CICDConfig{
		Provider:         "github",
		Workspace:        tempDir,
		ArtifactsPath:    filepath.Join(tempDir, "artifacts"),
		TestTimeout:      30 * time.Second,
		FailOnRegression: true,
		BaselineStrategy: "manual",
	}
	
	integration := cicd.NewCICDIntegration(config)
	buildCtx := integration.DetectEnvironment()
	
	// Verify PR context is detected
	if buildCtx.PullRequest == nil {
		t.Fatal("Expected pull request context to be detected")
	}
	
	if buildCtx.PullRequest.ID != "123" {
		t.Errorf("Expected PR ID '123', got '%s'", buildCtx.PullRequest.ID)
	}
	
	if buildCtx.PullRequest.Title != "Test PR" {
		t.Errorf("Expected PR title 'Test PR', got '%s'", buildCtx.PullRequest.Title)
	}
	
	if buildCtx.PullRequest.SourceBranch != "feature-branch" {
		t.Errorf("Expected source branch 'feature-branch', got '%s'", buildCtx.PullRequest.SourceBranch)
	}
	
	if buildCtx.PullRequest.TargetBranch != "main" {
		t.Errorf("Expected target branch 'main', got '%s'", buildCtx.PullRequest.TargetBranch)
	}
}

func TestHTMLReporter_ReportGeneration(t *testing.T) {
	tempDir := t.TempDir()
	
	config := pipeline.PipelineConfig{
		Environment:   "test",
		ResultsPath:   tempDir,
		ReportFormats: []string{"html", "json"},
	}
	
	reporter := pipeline.NewHTMLReporter(config)
	
	// Create test results
	results := []pipeline.TestResult{
		{
			TestName:  "TestReport",
			Status:    "passed",
			StartTime: time.Now().Add(-5 * time.Minute),
			EndTime:   time.Now(),
			Duration:  5 * time.Minute,
			Metrics: map[string]interface{}{
				"throughput":  150.5,
				"avg_latency": 0.025,
				"p95_latency": 0.045,
			},
			Errors:    []string{},
			Artifacts: []string{},
		},
	}
	
	baseline := pipeline.PerformanceBaseline{
		Version:     "test",
		Timestamp:   time.Now(),
		Environment: "test",
		Metrics: map[string]pipeline.BaselineMetric{
			"throughput": {
				Name:      "throughput",
				Value:     140.0,
				Unit:      "req/s",
				Tolerance: 5.0,
				Trend:     "stable",
			},
		},
	}
	
	// Generate reports
	err := reporter.GenerateReport(results, baseline)
	if err != nil {
		t.Fatalf("Report generation failed: %v", err)
	}
	
	// Verify HTML report was created
	htmlFiles, err := filepath.Glob(filepath.Join(tempDir, "*.html"))
	if err != nil {
		t.Fatalf("Failed to search for HTML files: %v", err)
	}
	
	if len(htmlFiles) == 0 {
		t.Error("Expected HTML report to be generated")
	}
	
	// Verify JSON report was created
	jsonFiles, err := filepath.Glob(filepath.Join(tempDir, "*.json"))
	if err != nil {
		t.Fatalf("Failed to search for JSON files: %v", err)
	}
	
	if len(jsonFiles) == 0 {
		t.Error("Expected JSON report to be generated")
	}
}

func TestIntegration_FullWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	
	config := cicd.CICDConfig{
		Provider:         "github",
		Workspace:        tempDir,
		Branch:           "main",
		CommitSHA:        "integration-test",
		BuildNumber:      "integration-1",
		ArtifactsPath:    filepath.Join(tempDir, "artifacts"),
		TestTimeout:      2 * time.Minute,
		FailOnRegression: false,
		BaselineStrategy: "manual",
	}
	
	integration := cicd.NewCICDIntegration(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	
	// Execute full workflow
	result, err := integration.ExecuteTests(ctx)
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}
	
	// Verify overall success
	if result.Status != "success" {
		t.Errorf("Expected overall status 'success', got '%s'", result.Status)
	}
	
	// Verify all components worked
	if len(result.TestResults) == 0 {
		t.Error("Expected test results to be populated")
	}
	
	if len(result.Reports) == 0 {
		t.Error("Expected reports to be generated")
	}
	
	// Verify artifacts were created
	artifactsDir := filepath.Join(tempDir, "artifacts")
	if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
		t.Error("Artifacts directory was not created")
	}
	
	// Verify baseline was created
	baselineFile := filepath.Join(tempDir, "artifacts", "baseline", "baseline.json")
	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		t.Error("Baseline file was not created")
	}
	
	// Test publishing (should not fail)
	err = integration.PublishResults(ctx, result)
	if err != nil {
		t.Errorf("Publishing results failed: %v", err)
	}
	
	t.Logf("Integration test completed successfully:")
	t.Logf("  Duration: %s", result.Duration)
	t.Logf("  Tests: %d", len(result.TestResults))
	t.Logf("  Reports: %d", len(result.Reports))
	t.Logf("  Artifacts: %d", len(result.Artifacts))
}