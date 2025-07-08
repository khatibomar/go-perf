// main_test.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"exercise-03-03/analysis"
	"exercise-03-03/simulation"
)

func TestHealthHandler(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if response["service"] != "bottleneck-analyzer" {
		t.Errorf("Expected service 'bottleneck-analyzer', got '%v'", response["service"])
	}

	// Check that timestamp is present and recent
	if _, ok := response["timestamp"]; !ok {
		t.Error("Response should include timestamp")
	}
}

func TestMetricsHandler(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Start analyzer to collect some metrics
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
	defer cancel()
	go analyzer.Start(ctx)
	time.Sleep(time.Millisecond * 200)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response analysis.PerformanceMetric
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify metrics structure
	if response.Timestamp.IsZero() {
		t.Error("Metrics should have a timestamp")
	}

	if response.GoroutineCount <= 0 {
		t.Error("Goroutine count should be positive")
	}

	if response.MemoryUsage <= 0 {
		t.Error("Memory usage should be positive")
	}

	if response.CPUUsage < 0 {
		t.Error("CPU usage should be non-negative")
	}
}

func TestBottlenecksHandler(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 50,
		AnalysisWindow:      time.Second * 2,
		ThresholdMultiplier: 1.5,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Set low baselines to trigger bottlenecks
	analyzer.UpdateBaseline("cpu_usage", 10.0, 5.0)
	analyzer.UpdateBaseline("memory_usage", 1024*1024, 512*1024)

	// Start analyzer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go analyzer.Start(ctx)
	time.Sleep(time.Millisecond * 500)

	req := httptest.NewRequest("GET", "/bottlenecks", nil)
	w := httptest.NewRecorder()

	server.handleBottlenecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response []analysis.Bottleneck
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Should return an array (may be empty)
	if response == nil {
		t.Error("Response should be an array")
	}
}

func TestAnalyzeHandler(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Collect some metrics first
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
	defer cancel()
	go analyzer.Start(ctx)
	time.Sleep(time.Millisecond * 200)

	req := httptest.NewRequest("POST", "/analyze", nil)
	w := httptest.NewRecorder()

	server.handleAnalyze(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check that bottlenecks field exists
	if _, ok := response["bottlenecks"]; !ok {
		t.Error("Response should include bottlenecks field")
	}

	// Check that timestamp exists
	if _, ok := response["timestamp"]; !ok {
		t.Error("Response should include timestamp")
	}
}

func TestBaselineHandler_GET(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	req := httptest.NewRequest("GET", "/baseline/", nil)
	w := httptest.NewRecorder()

	server.handleBaseline(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]analysis.Baseline
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Should have default baselines
	if _, ok := response["cpu_usage"]; !ok {
		t.Error("Response should include cpu_usage baseline")
	}

	if _, ok := response["memory_usage"]; !ok {
		t.Error("Response should include memory_usage baseline")
	}

	if _, ok := response["goroutine_count"]; !ok {
		t.Error("Response should include goroutine_count baseline")
	}
}

func TestBaselineHandler_POST(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Test updating baseline via URL parameters
	req := httptest.NewRequest("POST", "/baseline/test_metric?value=100.0&stddev=10.0", nil)
	w := httptest.NewRecorder()

	server.handleBaseline(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "updated" {
		t.Errorf("Expected status 'updated', got '%v'", response["status"])
	}

	if response["metric"] != "test_metric" {
		t.Errorf("Expected metric 'test_metric', got '%v'", response["metric"])
	}
}

func TestBaselineHandler_POST_InvalidJSON(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Test with invalid parameters
	req := httptest.NewRequest("POST", "/baseline/test_metric?value=invalid", nil)
	w := httptest.NewRecorder()

	server.handleBaseline(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestWorkloadHandler_GET(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	req := httptest.NewRequest("GET", "/simulate/status", nil)
	w := httptest.NewRecorder()

	server.handleSimulate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response simulation.WorkloadStatus
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Initially should be inactive
	if response.Active {
		t.Error("Workload should be inactive initially")
	}
}

func TestWorkloadHandler_POST(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Test starting workload
	requestBody := map[string]interface{}{
		"type":      "cpu",
		"intensity": "medium",
		"duration":  "500ms",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/simulate/cpu", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSimulate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "started" {
		t.Errorf("Expected status 'started', got '%v'", response["status"])
	}

	// Clean up
	simulator.Stop()
}

func TestWorkloadHandler_POST_InvalidDuration(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Test with invalid duration in query parameters
	req := httptest.NewRequest("POST", "/simulate/cpu?duration=invalid&intensity=medium", nil)
	w := httptest.NewRecorder()

	server.handleSimulate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestWorkloadHandler_DELETE(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Start a workload first
	err := simulator.StartWorkload("cpu", time.Second*10, "low")
	if err != nil {
		t.Fatalf("Failed to start workload: %v", err)
	}

	req := httptest.NewRequest("POST", "/simulate/stop", nil)
	w := httptest.NewRecorder()

	server.handleSimulate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "stopped" {
		t.Errorf("Expected status 'stopped', got '%v'", response["status"])
	}

	// Verify workload is actually stopped
	status := simulator.GetStatus()
	if status.Active {
		t.Error("Workload should be stopped")
	}
}

func TestWorkloadHandler_UnsupportedMethod(t *testing.T) {
	// Create server instance
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	req := httptest.NewRequest("PUT", "/simulate/cpu", nil)
	w := httptest.NewRecorder()

	server.handleSimulate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// Integration Tests

func TestIntegration_FullWorkflow(t *testing.T) {
	// Initialize components
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Start analyzer
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	go analyzer.Start(ctx)

	// 1. Check health
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.handleHealth(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Health check failed: %d", w.Code)
	}

	// 2. Get initial metrics
	time.Sleep(time.Millisecond * 200)
	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	server.handleMetrics(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Metrics request failed: %d", w.Code)
	}

	// 3. Start workload
	requestBody := map[string]interface{}{
		"type":      "cpu",
		"intensity": "medium",
		"duration":  "500ms",
	}
	body, _ := json.Marshal(requestBody)
	req = httptest.NewRequest("POST", "/simulate/cpu", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.handleSimulate(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Workload start failed: %d", w.Code)
	}

	// 4. Check workload status
	req = httptest.NewRequest("GET", "/simulate/status", nil)
	w = httptest.NewRecorder()
	server.handleSimulate(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Workload status failed: %d", w.Code)
	}

	// 5. Trigger analysis
	time.Sleep(time.Millisecond * 300)
	req = httptest.NewRequest("POST", "/analyze", nil)
	w = httptest.NewRecorder()
	server.handleAnalyze(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Analysis failed: %d", w.Code)
	}

	// 6. Get bottlenecks
	req = httptest.NewRequest("GET", "/bottlenecks", nil)
	w = httptest.NewRecorder()
	server.handleBottlenecks(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Bottlenecks request failed: %d", w.Code)
	}

	// 7. Update baseline
	req = httptest.NewRequest("POST", "/baseline/test_metric?value=100.0&stddev=10.0", nil)
	w = httptest.NewRecorder()
	server.handleBaseline(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Baseline update failed: %d", w.Code)
	}

	// 8. Stop workload
	req = httptest.NewRequest("POST", "/simulate/stop", nil)
	w = httptest.NewRecorder()
	server.handleSimulate(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Workload stop failed: %d", w.Code)
	}

	t.Log("Full workflow integration test completed successfully")
}

// Performance Tests

func TestPerformanceRequirements(t *testing.T) {
	// Initialize components
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	// Test API response times (should be <1 second)
	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		handler http.HandlerFunc
	}{
		{"Health", "GET", "/health", "", server.handleHealth},
		{"Metrics", "GET", "/metrics", "", server.handleMetrics},
		{"Bottlenecks", "GET", "/bottlenecks", "", server.handleBottlenecks},
		{"Analyze", "POST", "/analyze", "", server.handleAnalyze},
		{"Baseline POST", "POST", "/baseline/cpu_usage?value=100.0&stddev=10.0", "", server.handleBaseline},
		{"Workload GET", "GET", "/simulate/status", "", server.handleSimulate},
	}

	// Start analyzer for metrics collection
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	go analyzer.Start(ctx)
	time.Sleep(time.Millisecond * 200)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			start := time.Now()
			test.handler(w, req)
			responseTime := time.Since(start)

			if responseTime > time.Second {
				t.Errorf("%s response time: %v, requirement: <1 second", test.name, responseTime)
			}

			if w.Code >= 400 {
				t.Errorf("%s returned error status: %d", test.name, w.Code)
			}

			t.Logf("%s response time: %v", test.name, responseTime)
		})
	}
}

// Benchmark Tests

func BenchmarkHealthHandler(b *testing.B) {
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.handleHealth(w, req)
	}
}

func BenchmarkMetricsHandler(b *testing.B) {
	// Initialize analyzer
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		server.handleMetrics(w, req)
	}
}

func BenchmarkWorkloadHandler(b *testing.B) {
	// Initialize simulator
	config := analysis.BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}
	analyzer := analysis.NewBottleneckAnalyzer(config)
	simulator := simulation.NewWorkloadSimulator()
	server := &Server{
		analyzer:  analyzer,
		simulator: simulator,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/simulate/status", nil)
		w := httptest.NewRecorder()
		server.handleSimulate(w, req)
	}
}