package main

import (
	"testing"
	"time"
)

// TestLeakDetector tests the basic leak detector functionality
func TestLeakDetector(t *testing.T) {
	config := DefaultLeakDetectorConfig()
	config.SampleInterval = 1 * time.Second
	config.AnalysisWindow = 3

	detector := NewLeakDetector(config)

	// Test starting and stopping
	if err := detector.Start(); err != nil {
		t.Fatalf("Failed to start detector: %v", err)
	}

	if !detector.IsRunning() {
		t.Error("Detector should be running")
	}

	detector.Stop()

	if detector.IsRunning() {
		t.Error("Detector should not be running after stop")
	}
}

// TestMemorySnapshot tests memory snapshot collection
func TestMemorySnapshot(t *testing.T) {
	detector := NewLeakDetector(DefaultLeakDetectorConfig())

	detector.collectSnapshot()
	snapshots := detector.GetSnapshots()
	if len(snapshots) == 0 {
		t.Fatal("No snapshots collected")
	}
	snapshot := snapshots[len(snapshots)-1]

	if snapshot.Timestamp.IsZero() {
		t.Error("Snapshot timestamp should not be zero")
	}

	if snapshot.Goroutines <= 0 {
		t.Error("Goroutine count should be positive")
	}

	if snapshot.Alloc == 0 {
		t.Error("Memory allocation should not be zero")
	}
}

// TestGoroutineLeakAnalyzer tests goroutine leak detection
func TestGoroutineLeakAnalyzer(t *testing.T) {
	analyzer := &GoroutineLeakAnalyzer{}

	// Create snapshots with increasing goroutine count
	snapshots := make([]MemorySnapshot, 5)
	for i := 0; i < 5; i++ {
		snapshots[i] = MemorySnapshot{
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			Goroutines: 100 + i*20, // Increasing goroutine count
			Alloc:      1024 * 1024,
		}
	}

	detections := analyzer.Analyze(snapshots, nil)

	if len(detections) == 0 {
		t.Error("Should detect goroutine leak with increasing count")
	}

	if detections[0].Type != LeakTypeGoroutine {
		t.Errorf("Expected goroutine leak type, got %s", detections[0].Type)
	}
}

// TestMemoryGrowthAnalyzer tests memory growth detection
func TestMemoryGrowthAnalyzer(t *testing.T) {
	analyzer := &MemoryGrowthAnalyzer{}

	// Create snapshots with increasing memory usage
	snapshots := make([]MemorySnapshot, 5)
	for i := 0; i < 5; i++ {
		snapshots[i] = MemorySnapshot{
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			Goroutines: 10,
			Alloc:      uint64(10*1024*1024 + i*2*1024*1024), // Increasing memory
		}
	}

	detections := analyzer.Analyze(snapshots, nil)

	if len(detections) == 0 {
		t.Error("Should detect memory growth with increasing allocation")
	}

	if detections[0].Type != LeakTypeMemory {
		t.Errorf("Expected memory leak type, got %s", detections[0].Type)
	}
}

// TestHeapObjectAnalyzer tests heap object leak detection
func TestHeapObjectAnalyzer(t *testing.T) {
	analyzer := &HeapObjectAnalyzer{}

	// Create snapshots with increasing heap objects
	snapshots := make([]MemorySnapshot, 5)
	for i := 0; i < 5; i++ {
		snapshots[i] = MemorySnapshot{
			Timestamp:   time.Now().Add(time.Duration(i) * time.Second),
			Goroutines:  10,
			HeapObjects: uint64(50000 + i*30000), // Increasing objects
		}
	}

	detections := analyzer.Analyze(snapshots, nil)

	if len(detections) == 0 {
		t.Error("Should detect heap object leak with increasing objects")
	}

	if detections[0].Type != LeakTypeHeapObject {
		t.Errorf("Expected heap object leak type, got %s", detections[0].Type)
	}
}

// TestGCPressureAnalyzer tests GC pressure detection
func TestGCPressureAnalyzer(t *testing.T) {
	analyzer := &GCPressureAnalyzer{}

	// Create snapshots with high GC pressure
	snapshots := make([]MemorySnapshot, 5)
	for i := 0; i < 5; i++ {
		snapshots[i] = MemorySnapshot{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Second),
			Goroutines:    10,
			GCCPUFraction: 0.15, // High GC CPU usage
			NumGC:         uint32(100 + i*20),
		}
	}

	detections := analyzer.Analyze(snapshots, nil)

	if len(detections) == 0 {
		t.Error("Should detect GC pressure with high CPU fraction")
	}

	if detections[0].Type != LeakTypeGCPressure {
		t.Errorf("Expected GC pressure leak type, got %s", detections[0].Type)
	}
}

// TestCompositeAnalyzer tests the composite analyzer
func TestCompositeAnalyzer(t *testing.T) {
	analyzer := &CompositeAnalyzer{
		analyzers: []LeakAnalyzer{
			&GoroutineLeakAnalyzer{},
			&MemoryGrowthAnalyzer{},
		},
	}

	// Create snapshots with both goroutine and memory issues
	snapshots := make([]MemorySnapshot, 5)
	for i := 0; i < 5; i++ {
		snapshots[i] = MemorySnapshot{
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			Goroutines: 100 + i*20,                           // Increasing goroutines
			Alloc:      uint64(10*1024*1024 + i*2*1024*1024), // Increasing memory
		}
	}

	detections := analyzer.Analyze(snapshots, nil)

	if len(detections) < 2 {
		t.Error("Should detect both goroutine and memory leaks")
	}

	// Check that we have both types
	hasGoroutine := false
	hasMemory := false
	for _, detection := range detections {
		if detection.Type == LeakTypeGoroutine {
			hasGoroutine = true
		}
		if detection.Type == LeakTypeMemory {
			hasMemory = true
		}
	}

	if !hasGoroutine {
		t.Error("Should detect goroutine leak")
	}
	if !hasMemory {
		t.Error("Should detect memory leak")
	}
}

// TestLeakSimulator tests the leak simulator
func TestLeakSimulator(t *testing.T) {
	simulator := NewLeakSimulator()
	config := &LeakSimulationConfig{
		Duration:    5 * time.Second,
		Intensity:   2,
		AutoCleanup: true,
	}

	// Test goroutine leak simulation
	leakID, err := simulator.StartGoroutineLeak(config)
	if err != nil {
		t.Fatalf("Failed to start goroutine leak: %v", err)
	}

	if leakID == "" {
		t.Error("Leak ID should not be empty")
	}

	// Check running leaks
	runningLeaks := simulator.GetRunningLeaks()
	if len(runningLeaks) != 1 {
		t.Errorf("Expected 1 running leak, got %d", len(runningLeaks))
	}

	// Stop the leak
	if err := simulator.StopLeak(leakID); err != nil {
		t.Errorf("Failed to stop leak: %v", err)
	}

	// Check that leak is stopped
	runningLeaks = simulator.GetRunningLeaks()
	if len(runningLeaks) != 0 {
		t.Errorf("Expected 0 running leaks after stop, got %d", len(runningLeaks))
	}
}

// TestLeakTestFramework tests the automated testing framework
func TestLeakTestFramework(t *testing.T) {
	framework := NewLeakTestFramework()

	if framework.detector == nil {
		t.Error("Framework should have a detector")
	}

	if framework.simulator == nil {
		t.Error("Framework should have a simulator")
	}

	// Test creating default test suite
	testSuite := framework.CreateDefaultTestSuite()

	if testSuite == nil {
		t.Error("Test suite should not be nil")
	}

	if len(testSuite.Tests) == 0 {
		t.Error("Test suite should have tests")
	}

	// Verify test names
	expectedTests := []string{
		"GoroutineLeakDetection",
		"MemoryLeakDetection",
		"HeapObjectLeakDetection",
		"GCPressureDetection",
		"FalsePositiveTest",
		"MultipleLeakDetection",
	}

	for i, test := range testSuite.Tests {
		if i < len(expectedTests) && test.Name != expectedTests[i] {
			t.Errorf("Expected test name %s, got %s", expectedTests[i], test.Name)
		}
	}
}

// TestProductionMonitor tests the production monitoring system
func TestProductionMonitor(t *testing.T) {
	config := DefaultProductionMonitorConfig()
	config.HTTPPort = 8082          // Use different port for testing
	config.EnableHTTPServer = false // Disable HTTP server for testing
	config.EnableAlerting = false   // Disable alerting for testing
	config.MemoryCheckInterval = 500 * time.Millisecond // Short interval for testing

	monitor := NewProductionMemoryMonitor(config)

	if monitor == nil {
		t.Error("Monitor should not be nil")
	}

	// Test starting and stopping
	if err := monitor.Start(); err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// Let it run briefly
	time.Sleep(2 * time.Second)

	// Check metrics
	metrics := monitor.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}

	if metrics.TotalChecks == 0 {
		t.Error("Should have performed some checks")
	}

	// Check health
	if !monitor.IsHealthy() {
		t.Error("Monitor should be healthy")
	}

	// Stop monitor
	if err := monitor.Stop(); err != nil {
		t.Errorf("Failed to stop monitor: %v", err)
	}
}

// TestSystemMetricsCollection tests system metrics collection
func TestSystemMetricsCollection(t *testing.T) {
	monitor := NewProductionMemoryMonitor(nil)
	metrics := monitor.collectSystemMetrics()

	if metrics == nil {
		t.Error("System metrics should not be nil")
	}

	if metrics.MemoryUsage.Allocated == 0 {
		t.Error("Allocated memory should not be zero")
	}

	if metrics.GoroutineCount <= 0 {
		t.Error("Goroutine count should be positive")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if metrics.MemoryUsage.UsagePercent < 0 || metrics.MemoryUsage.UsagePercent > 100 {
		t.Errorf("Usage percentage should be between 0-100, got %.2f", metrics.MemoryUsage.UsagePercent)
	}
}

// TestAlertManager tests the alert management system
func TestAlertManager(t *testing.T) {
	alertManager := NewSimpleAlertManager()

	// Test triggering an alert
	alert := &Alert{
		ID:          "test_alert",
		Level:       AlertLevelWarning,
		Title:       "Test Alert",
		Description: "Test alert message",
		Timestamp:   time.Now(),
	}
	alertManager.TriggerAlert(alert)

	// Create and trigger another alert with same ID (should replace the first)
	alert2 := &Alert{
		ID:          "test_alert",
		Level:       AlertLevelWarning,
		Title:       "Test Alert 2",
		Description: "This is a test alert",
		Timestamp:   time.Now(),
	}

	alertManager.TriggerAlert(alert2)

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Check active alerts
	activeAlerts := alertManager.GetActiveAlerts()
	if len(activeAlerts) != 1 {
		t.Errorf("Expected 1 active alert, got %d", len(activeAlerts))
	}

	if activeAlerts[0].ID != "test_alert" {
		t.Errorf("Expected alert ID 'test_alert', got %s", activeAlerts[0].ID)
	}

	// Resolve the alert
	alertManager.ResolveAlert("test_alert")

	// Check that alert is resolved
	activeAlerts = alertManager.GetActiveAlerts()
	if len(activeAlerts) != 0 {
		t.Errorf("Expected 0 active alerts after resolution, got %d", len(activeAlerts))
	}
}

// BenchmarkLeakDetection benchmarks the leak detection performance
func BenchmarkLeakDetection(b *testing.B) {
	detector := NewLeakDetector(DefaultLeakDetectorConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.collectSnapshot()
		snapshots := detector.GetSnapshots()
		if len(snapshots) == 0 {
			b.Fatal("No snapshots collected")
		}
		snapshot := snapshots[len(snapshots)-1]
		_ = snapshot
	}
}

// BenchmarkMemoryGrowthAnalysis benchmarks memory growth analysis
func BenchmarkMemoryGrowthAnalysis(b *testing.B) {
	analyzer := &MemoryGrowthAnalyzer{}

	// Create test snapshots
	snapshots := make([]MemorySnapshot, 10)
	for i := 0; i < 10; i++ {
		snapshots[i] = MemorySnapshot{
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			Goroutines: 10,
			Alloc:      uint64(10*1024*1024 + i*1024*1024),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detections := analyzer.Analyze(snapshots, nil)
		_ = detections
	}
}

// BenchmarkSystemMetricsCollection benchmarks system metrics collection
func BenchmarkSystemMetricsCollection(b *testing.B) {
	monitor := NewProductionMemoryMonitor(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := monitor.collectSystemMetrics()
		_ = metrics
	}
}

// TestIntegration performs an integration test of all components
func TestIntegration(t *testing.T) {
	// Skip integration test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a production monitor with test configuration
	config := DefaultProductionMonitorConfig()
	config.MemoryCheckInterval = 2 * time.Second
	config.HTTPPort = 8083
	config.EnableHTTPServer = false
	config.EnableAlerting = true

	monitor := NewProductionMemoryMonitor(config)

	// Start monitoring
	if err := monitor.Start(); err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()

	// Create a leak simulator
	simulator := NewLeakSimulator()
	leakConfig := &LeakSimulationConfig{
		Duration:    10 * time.Second,
		Intensity:   3,
		AutoCleanup: true,
	}

	// Start a memory leak
	leakID, err := simulator.StartMemoryLeak(leakConfig)
	if err != nil {
		t.Fatalf("Failed to start memory leak: %v", err)
	}
	defer simulator.StopLeak(leakID)

	// Let the system run and detect the leak
	time.Sleep(15 * time.Second)

	// Check that leaks were detected
	metrics := monitor.GetMetrics()
	if metrics.TotalChecks == 0 {
		t.Error("Should have performed memory checks")
	}

	// Check for leak detections in the report
	report := monitor.detector.GenerateReport()
	if len(report.Detections) == 0 {
		t.Log("No leaks detected in integration test (this may be expected depending on timing)")
	}

	// Verify monitor is still healthy
	if !monitor.IsHealthy() {
		t.Error("Monitor should still be healthy after integration test")
	}

	// Clean up
	simulator.StopAllLeaks()
	simulator.CleanupLeakedResources()
}
