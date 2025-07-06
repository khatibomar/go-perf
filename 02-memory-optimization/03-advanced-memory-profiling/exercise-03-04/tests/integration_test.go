package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"exercise-03-04/analyzer"
	"exercise-03-04/detector"
	"exercise-03-04/simulator"
)

// TestBasicLeakDetection tests basic leak detection functionality
func TestBasicLeakDetection(t *testing.T) {
	// Create temporary output directory
	tempDir := createTempDir(t, "basic_leak_test")
	defer os.RemoveAll(tempDir)

	// Create analyzer configuration
	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          30 * time.Second,
		SamplingInterval:            2 * time.Second,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false, // Skip for faster test
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              false,
	}

	analyzer := analyzer.NewAnalyzer(config)

	// Run analysis
	result, err := analyzer.RunAnalysis()
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	// Validate results
	if result == nil {
		t.Fatal("Analysis result is nil")
	}

	if result.Duration <= 0 {
		t.Error("Analysis duration should be positive")
	}

	if result.OverallSummary.ConfidenceLevel < 0 || result.OverallSummary.ConfidenceLevel > 1 {
		t.Errorf("Confidence level should be between 0 and 1, got %.2f", result.OverallSummary.ConfidenceLevel)
	}

	// Test saving results
	err = analyzer.SaveResults(result, "test_results.json")
	if err != nil {
		t.Errorf("Failed to save results: %v", err)
	}

	// Verify file was created
	resultFile := filepath.Join(tempDir, "test_results.json")
	if _, err := os.Stat(resultFile); os.IsNotExist(err) {
		t.Error("Results file was not created")
	}
}

// TestHeapLeakDetection tests heap leak detection with simulation
func TestHeapLeakDetection(t *testing.T) {
	tempDir := createTempDir(t, "heap_leak_test")
	defer os.RemoveAll(tempDir)

	// Create heap leak simulation
	simConfig := &simulator.LeakConfig{
		Type:          simulator.LeakTypeHeap,
		Intensity:     5,
		Duration:      45 * time.Second,
		GrowthRate:    2.0,
		BurstInterval: 10 * time.Second,
		BurstSize:     5,
		Verbose:       false,
	}

	sim := simulator.NewLeakSimulator(simConfig)
	err := sim.Start()
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	defer sim.Stop()

	// Create analyzer configuration focused on heap
	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          60 * time.Second,
		SamplingInterval:            3 * time.Second,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     false,
		EnableReferenceAnalysis:     false,
		EnableResourceAnalysis:      false,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              true,
		AlertThresholds: analyzer.AlertThresholds{
			MemoryGrowthRate: 0.5, // Very sensitive for testing
		},
	}

	analyzer := analyzer.NewAnalyzer(config)

	// Run analysis
	result, err := analyzer.RunAnalysis()
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	// Validate heap analysis results
	if result.HeapAnalysis == nil {
		t.Fatal("Heap analysis is nil")
	}

	// Should detect memory growth due to simulation
	if result.HeapAnalysis.GrowthRate <= 0 {
		t.Error("Expected positive memory growth rate due to simulation")
	}

	// Should have some level of leak probability
	if result.HeapAnalysis.LeakProbability <= 0.1 {
		t.Error("Expected higher leak probability due to simulation")
	}

	// Should detect some anomalies or have alerts
	if len(result.HeapAnalysis.Anomalies) == 0 && len(result.Alerts) == 0 {
		t.Error("Expected to detect anomalies or generate alerts due to simulation")
	}

	// Overall summary should reflect detected issues
	if result.OverallSummary.TotalLeaksDetected == 0 {
		t.Error("Expected to detect leaks due to simulation")
	}
}

// TestGoroutineLeakDetection tests goroutine leak detection
func TestGoroutineLeakDetection(t *testing.T) {
	tempDir := createTempDir(t, "goroutine_leak_test")
	defer os.RemoveAll(tempDir)

	// Create goroutine leak simulation
	simConfig := &simulator.LeakConfig{
		Type:          simulator.LeakTypeGoroutine,
		Intensity:     5,
		Duration:      45 * time.Second,
		GrowthRate:    2.0,
		BurstInterval: 10 * time.Second,
		BurstSize:     5,
		Verbose:       false,
	}

	sim := simulator.NewLeakSimulator(simConfig)
	err := sim.Start()
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	defer sim.Stop()

	// Create analyzer configuration focused on goroutines
	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          60 * time.Second,
		SamplingInterval:            3 * time.Second,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          false,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false,
		EnableResourceAnalysis:      false,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              true,
		AlertThresholds: analyzer.AlertThresholds{
			GoroutineGrowthRate: 0.5, // Very sensitive for testing
		},
	}

	analyzer := analyzer.NewAnalyzer(config)

	// Run analysis
	result, err := analyzer.RunAnalysis()
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	// Validate goroutine analysis results
	if result.GoroutineAnalysis == nil {
		t.Fatal("Goroutine analysis is nil")
	}

	// Should detect goroutine growth due to simulation
	if result.GoroutineAnalysis.GrowthRate <= 0 {
		t.Error("Expected positive goroutine growth rate due to simulation")
	}

	// Should have some level of leak probability
	if result.GoroutineAnalysis.LeakProbability <= 0.1 {
		t.Error("Expected higher leak probability due to simulation")
	}

	// Should detect leak candidates or have alerts
	if len(result.GoroutineAnalysis.LeakCandidates) == 0 && len(result.Alerts) == 0 {
		t.Error("Expected to detect leak candidates or generate alerts due to simulation")
	}
}

// TestResourceLeakDetection tests resource leak detection
func TestResourceLeakDetection(t *testing.T) {
	tempDir := createTempDir(t, "resource_leak_test")
	defer os.RemoveAll(tempDir)

	// Create resource leak simulation
	simConfig := &simulator.LeakConfig{
		Type:          simulator.LeakTypeFile,
		Intensity:     5,
		Duration:      45 * time.Second,
		GrowthRate:    2.0,
		BurstInterval: 10 * time.Second,
		BurstSize:     5,
		Verbose:       false,
	}

	sim := simulator.NewLeakSimulator(simConfig)
	err := sim.Start()
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	defer sim.Stop()

	// Create analyzer configuration focused on resources
	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          60 * time.Second,
		SamplingInterval:            3 * time.Second,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          false,
		EnableGoroutineAnalysis:     false,
		EnableReferenceAnalysis:     false,
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              true,
		AlertThresholds: analyzer.AlertThresholds{
			FileHandleGrowthRate: 1.0, // Very sensitive for testing
		},
	}

	analyzer := analyzer.NewAnalyzer(config)

	// Run analysis
	result, err := analyzer.RunAnalysis()
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	// Validate resource analysis results
	if result.ResourceAnalysis == nil {
		t.Fatal("Resource analysis is nil")
	}

	// Should detect resource leaks or have alerts
	if len(result.ResourceAnalysis.DetectedLeaks) == 0 && len(result.Alerts) == 0 {
		t.Error("Expected to detect resource leaks or generate alerts due to simulation")
	}
}

// TestMultiTypeLeakDetection tests detection of multiple leak types
func TestMultiTypeLeakDetection(t *testing.T) {
	tempDir := createTempDir(t, "multi_type_leak_test")
	defer os.RemoveAll(tempDir)

	// Create mixed leak simulation
	simConfig := &simulator.LeakConfig{
		Type:          simulator.LeakTypeMixed,
		Intensity:     5,
		Duration:      60 * time.Second,
		GrowthRate:    2.0,
		BurstInterval: 10 * time.Second,
		BurstSize:     5,
		Verbose:       false,
	}

	sim := simulator.NewLeakSimulator(simConfig)
	err := sim.Start()
	if err != nil {
		t.Fatalf("Failed to start simulator: %v", err)
	}
	defer sim.Stop()

	// Create comprehensive analyzer configuration
	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          90 * time.Second,
		SamplingInterval:            3 * time.Second,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false, // Skip for faster test
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json", "html"},
		Verbose:                     false,
		RealTimeAlerts:              true,
		AlertThresholds: analyzer.AlertThresholds{
			MemoryGrowthRate:     0.5,
			GoroutineGrowthRate:  0.3,
			FileHandleGrowthRate: 1.0,
			NetworkGrowthRate:    0.5,
			CriticalLeakCount:    1,
		},
	}

	analyzer := analyzer.NewAnalyzer(config)

	// Run analysis
	result, err := analyzer.RunAnalysis()
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	// Validate comprehensive results
	if result.OverallSummary.TotalLeaksDetected == 0 {
		t.Error("Expected to detect leaks due to mixed simulation")
	}

	// Should have multiple leak categories
	if len(result.OverallSummary.LeaksByCategory) == 0 {
		t.Error("Expected multiple leak categories")
	}

	// Should have generated some alerts
	if len(result.Alerts) == 0 {
		t.Error("Expected to generate alerts due to simulation")
	}

	// Test report generation
	err = analyzer.GenerateReport(result)
	if err != nil {
		t.Errorf("Failed to generate report: %v", err)
	}

	// Verify report files were created
	jsonReport := filepath.Join(tempDir, "leak_detection_report.json")
	htmlReport := filepath.Join(tempDir, "leak_detection_report.html")

	if _, err := os.Stat(jsonReport); os.IsNotExist(err) {
		t.Error("JSON report was not created")
	}

	if _, err := os.Stat(htmlReport); os.IsNotExist(err) {
		t.Error("HTML report was not created")
	}
}

// TestRealTimeAlerts tests real-time alert functionality
func TestRealTimeAlerts(t *testing.T) {
	tempDir := createTempDir(t, "realtime_alerts_test")
	defer os.RemoveAll(tempDir)

	// Create analyzer with very sensitive thresholds
	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          45 * time.Second,
		SamplingInterval:            2 * time.Second,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false,
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              true,
		AlertThresholds: analyzer.AlertThresholds{
			MemoryGrowthRate:     0.1, // Very sensitive
			GoroutineGrowthRate:  0.1,
			FileHandleGrowthRate: 0.5,
			NetworkGrowthRate:    0.1,
			CriticalLeakCount:    1,
		},
	}

	analyzer := analyzer.NewAnalyzer(config)

	// Start analyzer
	err := analyzer.Start()
	if err != nil {
		t.Fatalf("Failed to start analyzer: %v", err)
	}

	// Create some activity to trigger alerts
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go createTestActivity(ctx)

	// Wait for some time to allow alerts to be generated
	time.Sleep(25 * time.Second)

	// Stop analyzer
	err = analyzer.Stop()
	if err != nil {
		t.Errorf("Failed to stop analyzer: %v", err)
	}

	// Check for alerts
	alerts := analyzer.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Expected to generate alerts with sensitive thresholds")
	}

	// Validate alert structure
	for _, alert := range alerts {
		if alert.ID == "" {
			t.Error("Alert ID should not be empty")
		}
		if alert.Type == "" {
			t.Error("Alert type should not be empty")
		}
		if alert.Severity == "" {
			t.Error("Alert severity should not be empty")
		}
		if alert.Title == "" {
			t.Error("Alert title should not be empty")
		}
		if alert.Timestamp.IsZero() {
			t.Error("Alert timestamp should not be zero")
		}
	}
}

// TestDetectorConfiguration tests individual detector configuration
func TestDetectorConfiguration(t *testing.T) {
	// Test heap analyzer configuration
	heapConfig := &detector.HeapConfig{
		Interval:  5 * time.Second,
		Threshold: 1.0,
		Verbose:   false,
	}

	heapAnalyzer := detector.NewHeapAnalyzer(heapConfig)
	if heapAnalyzer == nil {
		t.Error("Heap analyzer should not be nil")
	}

	// Test goroutine detector configuration
	goroutineConfig := &detector.GoroutineConfig{
		Interval:         5 * time.Second,
		GrowthThreshold:  5,
		GrowthRate:       1.0,
		Verbose:          false,
		StackTraceDepth:  10,
		SampleStackTrace: true,
	}

	goroutineDetector := detector.NewGoroutineDetector(goroutineConfig)
	if goroutineDetector == nil {
		t.Error("Goroutine detector should not be nil")
	}

	// Test resource detector configuration
	resourceConfig := &detector.ResourceConfig{
		Interval:            5 * time.Second,
		FileHandleThreshold: 100,
		NetworkThreshold:    50,
		GrowthThreshold:     1.0,
		Verbose:             false,
		TrackStacks:         false,
	}

	resourceDetector := detector.NewResourceDetector(resourceConfig)
	if resourceDetector == nil {
		t.Error("Resource detector should not be nil")
	}

	// Test starting and stopping detectors
	err := heapAnalyzer.Start()
	if err != nil {
		t.Errorf("Failed to start heap analyzer: %v", err)
	}

	err = goroutineDetector.Start()
	if err != nil {
		t.Errorf("Failed to start goroutine detector: %v", err)
	}

	err = resourceDetector.Start()
	if err != nil {
		t.Errorf("Failed to start resource detector: %v", err)
	}

	// Let them run for a short time
	time.Sleep(10 * time.Second)

	// Stop detectors
	heapAnalyzer.Stop()
	goroutineDetector.Stop()
	resourceDetector.Stop()

	// Verify they collected some metrics
	heapMetrics := heapAnalyzer.GetMetrics()
	if len(heapMetrics) == 0 {
		t.Error("Heap analyzer should have collected metrics")
	}

	goroutineMetrics := goroutineDetector.GetMetrics()
	if len(goroutineMetrics) == 0 {
		t.Error("Goroutine detector should have collected metrics")
	}

	resourceMetrics := resourceDetector.GetMetrics()
	if len(resourceMetrics) == 0 {
		t.Error("Resource detector should have collected metrics")
	}
}

// TestAnalyzerLifecycle tests the complete analyzer lifecycle
func TestAnalyzerLifecycle(t *testing.T) {
	tempDir := createTempDir(t, "lifecycle_test")
	defer os.RemoveAll(tempDir)

	config := analyzer.GetDefaultConfig()
	config.MonitoringDuration = 20 * time.Second
	config.SamplingInterval = 2 * time.Second
	config.OutputDir = tempDir
	config.RealTimeAlerts = false

	analyzer := analyzer.NewAnalyzer(config)

	// Test initial state
	status := analyzer.GetCurrentStatus()
	if status["running"].(bool) {
		t.Error("Analyzer should not be running initially")
	}

	// Test start
	err := analyzer.Start()
	if err != nil {
		t.Fatalf("Failed to start analyzer: %v", err)
	}

	status = analyzer.GetCurrentStatus()
	if !status["running"].(bool) {
		t.Error("Analyzer should be running after start")
	}

	// Test double start (should fail)
	err = analyzer.Start()
	if err == nil {
		t.Error("Starting already running analyzer should fail")
	}

	// Let it run for a bit
	time.Sleep(5 * time.Second)

	// Test stop
	err = analyzer.Stop()
	if err != nil {
		t.Errorf("Failed to stop analyzer: %v", err)
	}

	status = analyzer.GetCurrentStatus()
	if status["running"].(bool) {
		t.Error("Analyzer should not be running after stop")
	}

	// Test double stop (should fail)
	err = analyzer.Stop()
	if err == nil {
		t.Error("Stopping already stopped analyzer should fail")
	}
}

// Helper functions

// createTempDir creates a temporary directory for testing
func createTempDir(t *testing.T, prefix string) string {
	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tempDir
}

// createTestActivity creates some test activity to trigger detection
func createTestActivity(ctx context.Context) {
	var data [][]byte
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Allocate some memory
			for i := 0; i < 5; i++ {
				chunk := make([]byte, 1024*1024) // 1MB
				data = append(data, chunk)
			}

			// Create some goroutines
			for i := 0; i < 3; i++ {
				go func(id int) {
					time.Sleep(time.Duration(id) * time.Second)
				}(i)
			}

			// Occasionally clean up some memory
			if len(data) > 20 {
				data = data[:len(data)/2]
			}
		}
	}
}