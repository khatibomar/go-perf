// stress/testing_test.go
package stress

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewLoadGenerator(t *testing.T) {
	config := StressTestConfig{
		Name:             "Test Load Generator",
		Duration:         1 * time.Minute,
		Concurrency:      10,
		RampUpDuration:   10 * time.Second,
		RampDownDuration: 10 * time.Second,
		TargetRPS:        50,
		MaxRPS:           100,
		ThinkTime:        100 * time.Millisecond,
		Timeout:          5 * time.Second,
		FailureThreshold: 0.05,
	}

	lg := NewLoadGenerator(config)
	if lg == nil {
		t.Fatal("NewLoadGenerator returned nil")
	}

	if lg.config.Name != config.Name {
		t.Errorf("Expected config name '%s', got '%s'", config.Name, lg.config.Name)
	}

	if lg.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if lg.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}

	if lg.metrics.errorCounts == nil {
		t.Error("Expected error counts map to be initialized")
	}
}

func TestLoadGenerator_RunStressTest_Success(t *testing.T) {
	config := StressTestConfig{
		Name:             "Success Test",
		Duration:         500 * time.Millisecond,
		Concurrency:      2,
		RampUpDuration:   100 * time.Millisecond,
		RampDownDuration: 100 * time.Millisecond,
		TargetRPS:        10,
		MaxRPS:           20,
		ThinkTime:        5 * time.Millisecond,
		Timeout:          2 * time.Second,
		FailureThreshold: 0.1,
	}

	lg := NewLoadGenerator(config)
	ctx := context.Background()

	// Test function that always succeeds
	testFunc := func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond) // Simulate work
		return nil
	}

	result, err := lg.RunStressTest(ctx, testFunc)
	if err != nil {
		t.Fatalf("RunStressTest failed: %v", err)
	}

	if result == nil {
		t.Fatal("RunStressTest returned nil result")
	}

	if result.TestName != config.Name {
		t.Errorf("Expected test name '%s', got '%s'", config.Name, result.TestName)
	}

	if result.TotalRequests <= 0 {
		t.Errorf("Expected positive total requests, got %d", result.TotalRequests)
	}

	if result.SuccessfulReqs != result.TotalRequests {
		t.Errorf("Expected all requests to succeed, got %d successful out of %d total",
			result.SuccessfulReqs, result.TotalRequests)
	}

	if result.FailedRequests != 0 {
		t.Errorf("Expected no failed requests, got %d", result.FailedRequests)
	}

	if result.ErrorRate != 0 {
		t.Errorf("Expected zero error rate, got %f", result.ErrorRate)
	}

	if result.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", result.Duration)
	}
}

func TestLoadGenerator_RunStressTest_WithFailures(t *testing.T) {
	config := StressTestConfig{
		Name:             "Failure Test",
		Duration:         400 * time.Millisecond,
		Concurrency:      2,
		RampUpDuration:   75 * time.Millisecond,
		RampDownDuration: 75 * time.Millisecond,
		TargetRPS:        20,
		MaxRPS:           40,
		ThinkTime:        2 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 0.8, // High threshold to avoid breaking point
	}

	lg := NewLoadGenerator(config)
	ctx := context.Background()

	requestCount := 0
	// Test function that fails every other request
	testFunc := func(ctx context.Context) error {
		requestCount++
		if requestCount%2 == 0 {
			return errors.New("simulated failure")
		}
		return nil
	}

	result, err := lg.RunStressTest(ctx, testFunc)
	if err != nil {
		t.Fatalf("RunStressTest failed: %v", err)
	}

	if result.TotalRequests <= 0 {
		t.Errorf("Expected positive total requests, got %d", result.TotalRequests)
	}

	if result.FailedRequests == 0 {
		t.Error("Expected some failed requests")
	}

	if result.ErrorRate == 0 {
		t.Error("Expected non-zero error rate")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected error details to be recorded")
	}

	if _, exists := result.Errors["simulated failure"]; !exists {
		t.Error("Expected 'simulated failure' to be recorded in errors")
	}
}

func TestLoadGenerator_RecordMetrics(t *testing.T) {
	config := StressTestConfig{
		Name:        "Metrics Test",
		Duration:    1 * time.Second,
		Concurrency: 1,
		Timeout:     1 * time.Second,
	}

	lg := NewLoadGenerator(config)

	// Test successful request
	lg.recordMetrics(nil, 100*time.Millisecond)

	if lg.metrics.totalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", lg.metrics.totalRequests)
	}

	if lg.metrics.successfulReqs != 1 {
		t.Errorf("Expected 1 successful request, got %d", lg.metrics.successfulReqs)
	}

	if lg.metrics.failedRequests != 0 {
		t.Errorf("Expected 0 failed requests, got %d", lg.metrics.failedRequests)
	}

	if len(lg.metrics.responseTimes) != 1 {
		t.Errorf("Expected 1 response time, got %d", len(lg.metrics.responseTimes))
	}

	if lg.metrics.responseTimes[0] != 100*time.Millisecond {
		t.Errorf("Expected response time 100ms, got %v", lg.metrics.responseTimes[0])
	}

	// Test failed request
	lg.recordMetrics(errors.New("test error"), 200*time.Millisecond)

	if lg.metrics.totalRequests != 2 {
		t.Errorf("Expected 2 total requests, got %d", lg.metrics.totalRequests)
	}

	if lg.metrics.successfulReqs != 1 {
		t.Errorf("Expected 1 successful request, got %d", lg.metrics.successfulReqs)
	}

	if lg.metrics.failedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", lg.metrics.failedRequests)
	}

	if lg.metrics.errorCounts["test error"] != 1 {
		t.Errorf("Expected 1 'test error', got %d", lg.metrics.errorCounts["test error"])
	}
}

func TestLoadGenerator_CheckBreakingPoint(t *testing.T) {
	config := StressTestConfig{
		Name:             "Breaking Point Test",
		Duration:         1 * time.Second,
		Concurrency:      1,
		Timeout:          1 * time.Second,
		FailureThreshold: 0.5, // 50% threshold
	}

	lg := NewLoadGenerator(config)

	// Add some successful requests
	lg.recordMetrics(nil, 100*time.Millisecond)
	lg.recordMetrics(nil, 100*time.Millisecond)

	// Should not trigger breaking point
	if lg.checkBreakingPoint() {
		t.Error("Breaking point should not be triggered with low error rate")
	}

	// Add failed requests to exceed threshold
	lg.recordMetrics(errors.New("error1"), 100*time.Millisecond)
	lg.recordMetrics(errors.New("error2"), 100*time.Millisecond)
	lg.recordMetrics(errors.New("error3"), 100*time.Millisecond)

	// Should trigger breaking point (5 failed out of 7 total = 71% > 50%)
	if !lg.checkBreakingPoint() {
		t.Error("Breaking point should be triggered with high error rate")
	}

	if lg.breakingPoint == nil {
		t.Error("Breaking point should be recorded")
	}

	if lg.breakingPoint.ErrorRate <= 0.5 {
		t.Errorf("Expected error rate > 0.5, got %f", lg.breakingPoint.ErrorRate)
	}
}

func TestLoadGenerator_CalculateResponseTimeStats(t *testing.T) {
	config := StressTestConfig{
		Name:        "Stats Test",
		Duration:    1 * time.Second,
		Concurrency: 1,
		Timeout:     1 * time.Second,
	}

	lg := NewLoadGenerator(config)

	// Add response times: 100ms, 200ms, 300ms, 400ms, 500ms
	responseTimes := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
	}

	for _, rt := range responseTimes {
		lg.recordMetrics(nil, rt)
	}

	stats := lg.calculateResponseTimeStats()

	if stats.Min != 100*time.Millisecond {
		t.Errorf("Expected min 100ms, got %v", stats.Min)
	}

	if stats.Max != 500*time.Millisecond {
		t.Errorf("Expected max 500ms, got %v", stats.Max)
	}

	if stats.Median != 300*time.Millisecond {
		t.Errorf("Expected median 300ms, got %v", stats.Median)
	}

	expectedMean := 300 * time.Millisecond // (100+200+300+400+500)/5
	if stats.Mean != expectedMean {
		t.Errorf("Expected mean %v, got %v", expectedMean, stats.Mean)
	}

	// P95 should be 500ms (95% of 5 items = index 4)
	if stats.P95 != 500*time.Millisecond {
		t.Errorf("Expected P95 500ms, got %v", stats.P95)
	}

	// P99 should be 500ms (99% of 5 items = index 4)
	if stats.P99 != 500*time.Millisecond {
		t.Errorf("Expected P99 500ms, got %v", stats.P99)
	}
}

func TestLoadGenerator_CalculateResponseTimeStats_Empty(t *testing.T) {
	config := StressTestConfig{
		Name:        "Empty Stats Test",
		Duration:    1 * time.Second,
		Concurrency: 1,
		Timeout:     1 * time.Second,
	}

	lg := NewLoadGenerator(config)

	// No response times recorded
	stats := lg.calculateResponseTimeStats()

	if stats.Min != 0 {
		t.Errorf("Expected min 0, got %v", stats.Min)
	}

	if stats.Max != 0 {
		t.Errorf("Expected max 0, got %v", stats.Max)
	}

	if stats.Mean != 0 {
		t.Errorf("Expected mean 0, got %v", stats.Mean)
	}
}

func TestHTTPStressTest(t *testing.T) {
	// Test HTTP stress test function creation
	testFunc := HTTPStressTest("https://httpbin.org/get", "GET", 
		map[string]string{"User-Agent": "Test"}, nil)

	if testFunc == nil {
		t.Fatal("HTTPStressTest returned nil function")
	}

	// Note: We don't actually call the function in tests to avoid external dependencies
	// In a real scenario, you might use a mock HTTP server
}

func TestCPUStressTest(t *testing.T) {
	testFunc := CPUStressTest(100)

	if testFunc == nil {
		t.Fatal("CPUStressTest returned nil function")
	}

	ctx := context.Background()
	start := time.Now()
	err := testFunc(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("CPUStressTest failed: %v", err)
	}

	// Should take some time to execute
	if duration < 1*time.Millisecond {
		t.Errorf("Expected CPU stress test to take some time, took %v", duration)
	}
}

func TestMemoryStressTest(t *testing.T) {
	testFunc := MemoryStressTest(1) // 1MB

	if testFunc == nil {
		t.Fatal("MemoryStressTest returned nil function")
	}

	ctx := context.Background()
	err := testFunc(ctx)

	if err != nil {
		t.Errorf("MemoryStressTest failed: %v", err)
	}
}

func TestStressTestConfig_Validation(t *testing.T) {
	// Test valid configuration
	validConfig := StressTestConfig{
		Name:             "Valid Test",
		Duration:         1 * time.Minute,
		Concurrency:      10,
		RampUpDuration:   10 * time.Second,
		RampDownDuration: 10 * time.Second,
		TargetRPS:        50,
		MaxRPS:           100,
		ThinkTime:        100 * time.Millisecond,
		Timeout:          5 * time.Second,
		FailureThreshold: 0.05,
	}

	lg := NewLoadGenerator(validConfig)
	if lg == nil {
		t.Error("Valid configuration should create load generator")
	}
}

func TestLoadGenerator_GenerateResult(t *testing.T) {
	config := StressTestConfig{
		Name:        "Result Test",
		Duration:    1 * time.Second,
		Concurrency: 1,
		Timeout:     1 * time.Second,
	}

	lg := NewLoadGenerator(config)

	// Add some test data
	lg.recordMetrics(nil, 100*time.Millisecond)
	lg.recordMetrics(errors.New("test error"), 200*time.Millisecond)

	startTime := time.Now().Add(-1 * time.Second)
	endTime := time.Now()

	result := lg.generateResult(startTime, endTime)

	if result == nil {
		t.Fatal("generateResult returned nil")
	}

	if result.TestName != config.Name {
		t.Errorf("Expected test name '%s', got '%s'", config.Name, result.TestName)
	}

	if result.TotalRequests != 2 {
		t.Errorf("Expected 2 total requests, got %d", result.TotalRequests)
	}

	if result.SuccessfulReqs != 1 {
		t.Errorf("Expected 1 successful request, got %d", result.SuccessfulReqs)
	}

	if result.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", result.FailedRequests)
	}

	if result.ErrorRate != 0.5 {
		t.Errorf("Expected error rate 0.5, got %f", result.ErrorRate)
	}

	if result.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", result.Duration)
	}

	if result.AverageRPS <= 0 {
		t.Errorf("Expected positive average RPS, got %f", result.AverageRPS)
	}
}

func TestLoadGenerator_CopyErrorCounts(t *testing.T) {
	config := StressTestConfig{
		Name:        "Error Copy Test",
		Duration:    1 * time.Second,
		Concurrency: 1,
		Timeout:     1 * time.Second,
	}

	lg := NewLoadGenerator(config)

	// Add some errors
	lg.recordMetrics(errors.New("error1"), 100*time.Millisecond)
	lg.recordMetrics(errors.New("error2"), 100*time.Millisecond)
	lg.recordMetrics(errors.New("error1"), 100*time.Millisecond)

	errorCopy := lg.copyErrorCounts()

	if len(errorCopy) != 2 {
		t.Errorf("Expected 2 unique errors, got %d", len(errorCopy))
	}

	if errorCopy["error1"] != 2 {
		t.Errorf("Expected 2 occurrences of 'error1', got %d", errorCopy["error1"])
	}

	if errorCopy["error2"] != 1 {
		t.Errorf("Expected 1 occurrence of 'error2', got %d", errorCopy["error2"])
	}

	// Verify it's a copy (modifying original shouldn't affect copy)
	lg.metrics.errorCounts["error3"] = 5
	if _, exists := errorCopy["error3"]; exists {
		t.Error("Error copy should not be affected by changes to original")
	}
}