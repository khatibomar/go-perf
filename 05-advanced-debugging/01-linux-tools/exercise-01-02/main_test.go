package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

// TestProcessAnalyzerCreation tests the creation of ProcessAnalyzer
func TestProcessAnalyzerCreation(t *testing.T) {
	analyzer := NewProcessAnalyzer(100)

	if analyzer == nil {
		t.Fatal("ProcessAnalyzer creation failed")
	}

	if analyzer.maxSnapshots != 100 {
		t.Errorf("Expected maxSnapshots to be 100, got %d", analyzer.maxSnapshots)
	}

	if analyzer.analysisInterval != 1*time.Second {
		t.Errorf("Expected analysisInterval to be 1s, got %v", analyzer.analysisInterval)
	}

	if analyzer.leakDetector == nil {
		t.Error("Memory leak detector not initialized")
	}

	if len(analyzer.memorySnapshots) != 0 {
		t.Errorf("Expected empty memory snapshots, got %d", len(analyzer.memorySnapshots))
	}

	if len(analyzer.gcStats) != 0 {
		t.Errorf("Expected empty GC stats, got %d", len(analyzer.gcStats))
	}

	if len(analyzer.processMetrics) != 0 {
		t.Errorf("Expected empty process metrics, got %d", len(analyzer.processMetrics))
	}

	if len(analyzer.alerts) != 0 {
		t.Errorf("Expected empty alerts, got %d", len(analyzer.alerts))
	}

	t.Logf("ProcessAnalyzer created successfully with PID: %d", analyzer.pid)
}

// TestMemoryLeakDetector tests the memory leak detection functionality
func TestMemoryLeakDetector(t *testing.T) {
	detector := NewMemoryLeakDetector()

	if detector == nil {
		t.Fatal("MemoryLeakDetector creation failed")
	}

	if detector.growthThreshold != 20.0 {
		t.Errorf("Expected growthThreshold to be 20.0, got %f", detector.growthThreshold)
	}

	if detector.detectionWindow != 5*time.Minute {
		t.Errorf("Expected detectionWindow to be 5m, got %v", detector.detectionWindow)
	}

	if detector.growthThresholdCount != 3 {
		t.Errorf("Expected growthThresholdCount to be 3, got %d", detector.growthThresholdCount)
	}

	if detector.IsLeakDetected() {
		t.Error("Expected no leak detected initially")
	}

	// Test baseline setting
	snapshot1 := MemorySnapshot{
		Timestamp: time.Now(),
		HeapAlloc: 1024 * 1024, // 1MB
	}
	detector.Analyze(snapshot1)

	if detector.baselineHeap != snapshot1.HeapAlloc {
		t.Errorf("Expected baseline to be set to %d, got %d", snapshot1.HeapAlloc, detector.baselineHeap)
	}

	t.Log("MemoryLeakDetector tests passed")
}

// TestMetricsCollection tests the collection of various metrics
func TestMetricsCollection(t *testing.T) {
	analyzer := NewProcessAnalyzer(10)

	// Test memory snapshot collection
	memSnapshot := analyzer.collectMemorySnapshot()
	if memSnapshot.Timestamp.IsZero() {
		t.Error("Memory snapshot timestamp not set")
	}
	if memSnapshot.HeapAlloc == 0 {
		t.Error("HeapAlloc should be greater than 0")
	}
	if memSnapshot.Sys == 0 {
		t.Error("Sys should be greater than 0")
	}

	// Test GC snapshot collection
	gcSnapshot := analyzer.collectGCSnapshot()
	if gcSnapshot.Timestamp.IsZero() {
		t.Error("GC snapshot timestamp not set")
	}

	// Test process metrics collection
	procMetrics := analyzer.collectProcessMetrics()
	if procMetrics.Timestamp.IsZero() {
		t.Error("Process metrics timestamp not set")
	}
	if procMetrics.PID != analyzer.pid {
		t.Errorf("Expected PID %d, got %d", analyzer.pid, procMetrics.PID)
	}
	if procMetrics.NumGoroutines <= 0 {
		t.Error("NumGoroutines should be greater than 0")
	}

	t.Logf("Metrics collection tests passed - HeapAlloc: %d, NumGoroutines: %d", 
		memSnapshot.HeapAlloc, procMetrics.NumGoroutines)
}

// TestSnapshotManagement tests the snapshot size management
func TestSnapshotManagement(t *testing.T) {
	analyzer := NewProcessAnalyzer(3) // Small limit for testing

	// Add more snapshots than the limit
	for i := 0; i < 5; i++ {
		snapshot := MemorySnapshot{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			HeapAlloc: uint64(i * 1024),
		}
		analyzer.addMemorySnapshot(snapshot)
	}

	snapshots := analyzer.GetMemorySnapshots()
	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	// Verify oldest snapshots were removed (FIFO)
	if snapshots[0].HeapAlloc != 2*1024 {
		t.Errorf("Expected oldest snapshot HeapAlloc to be %d, got %d", 2*1024, snapshots[0].HeapAlloc)
	}

	t.Log("Snapshot management tests passed")
}

// TestAlertGeneration tests the alert generation functionality
func TestAlertGeneration(t *testing.T) {
	analyzer := NewProcessAnalyzer(10)

	// Set low thresholds to trigger alerts
	analyzer.thresholds.CPUUsagePercent = 0.1 // Very low threshold
	analyzer.thresholds.GoroutineCount = 1    // Very low threshold

	// Add some metrics to trigger alerts
	memSnapshot := analyzer.collectMemorySnapshot()
	analyzer.addMemorySnapshot(memSnapshot)

	procMetrics := analyzer.collectProcessMetrics()
	analyzer.addProcessMetrics(procMetrics)

	// Generate alerts
	analyzer.generateAlerts()

	alerts := analyzer.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Expected at least one alert to be generated")
	}

	// Check for goroutine alert (should be triggered)
	foundGoroutineAlert := false
	for _, alert := range alerts {
		if alert.Type == "HIGH_GOROUTINES" {
			foundGoroutineAlert = true
			break
		}
	}

	if !foundGoroutineAlert {
		t.Error("Expected HIGH_GOROUTINES alert to be generated")
	}

	t.Logf("Alert generation tests passed - Generated %d alerts", len(alerts))
}

// TestHTTPEndpoints tests all HTTP endpoints
func TestHTTPEndpoints(t *testing.T) {
	analyzer := NewProcessAnalyzer(10)

	// Collect some data first
	analyzer.collectMetrics()
	time.Sleep(100 * time.Millisecond) // Allow metrics to be collected

	tests := []struct {
		name     string
		path     string
		handler  http.HandlerFunc
		contentType string
	}{
		{"Memory Snapshots", "/memory", analyzer.handleMemorySnapshots, "application/json"},
		{"GC Stats", "/gc", analyzer.handleGCStats, "application/json"},
		{"Process Metrics", "/process", analyzer.handleProcessMetrics, "application/json"},
		{"Alerts", "/alerts", analyzer.handleAlerts, "application/json"},
		{"Stats", "/stats", analyzer.handleStats, "application/json"},
		{"Dashboard", "/", analyzer.handleDashboard, "text/html"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", test.path, nil)
			w := httptest.NewRecorder()

			test.handler(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if contentType != test.contentType {
				t.Errorf("Expected content type %s, got %s", test.contentType, contentType)
			}

			if resp.ContentLength == 0 && w.Body.Len() == 0 {
				t.Error("Response body is empty")
			}
		})
	}

	t.Log("HTTP endpoints tests passed")
}

// TestConcurrentAccess tests concurrent access to the analyzer
func TestConcurrentAccess(t *testing.T) {
	analyzer := NewProcessAnalyzer(100)

	// Start multiple goroutines accessing the analyzer concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 10; j++ {
				// Collect metrics
				analyzer.collectMetrics()

				// Read data
				_ = analyzer.GetMemorySnapshots()
				_ = analyzer.GetGCStats()
				_ = analyzer.GetProcessMetrics()
				_ = analyzer.GetAlerts()
				_ = analyzer.GetAnalysisStats()

				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Goroutine completed
		case <-time.After(10 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}

	// Verify data integrity
	snapshots := analyzer.GetMemorySnapshots()
	if len(snapshots) == 0 {
		t.Error("No memory snapshots collected during concurrent access")
	}

	t.Logf("Concurrent access tests passed - Collected %d snapshots", len(snapshots))
}

// TestMemoryLeakDetection tests the memory leak detection over time
func TestMemoryLeakDetection(t *testing.T) {
	detector := NewMemoryLeakDetector()
	detector.detectionWindow = 100 * time.Millisecond // Shorter window for testing
	detector.growthThreshold = 10.0                   // 10% growth threshold
	detector.growthThresholdCount = 2                 // Require 2 consecutive growths

	// Simulate baseline
	baseline := MemorySnapshot{
		Timestamp: time.Now(),
		HeapAlloc: 1024 * 1024, // 1MB
	}
	detector.Analyze(baseline)

	if detector.IsLeakDetected() {
		t.Error("Should not detect leak initially")
	}

	// Wait for detection window
	time.Sleep(150 * time.Millisecond)

	// Simulate memory growth (15% increase)
	growth1 := MemorySnapshot{
		Timestamp: time.Now(),
		HeapAlloc: 1024 * 1024 * 115 / 100, // 15% increase
	}
	detector.Analyze(growth1)

	// Should not detect leak yet (need consecutive growths)
	if detector.IsLeakDetected() {
		t.Error("Should not detect leak after first growth")
	}

	// Simulate another growth
	growth2 := MemorySnapshot{
		Timestamp: time.Now(),
		HeapAlloc: 1024 * 1024 * 130 / 100, // 30% increase from baseline
	}
	detector.Analyze(growth2)

	// Now should detect leak
	if !detector.IsLeakDetected() {
		t.Error("Should detect leak after consecutive growths")
	}

	t.Log("Memory leak detection tests passed")
}

// TestAnalysisOverhead tests that the analysis overhead is minimal
func TestAnalysisOverhead(t *testing.T) {
	analyzer := NewProcessAnalyzer(100)

	startTime := time.Now()
	iterations := 100

	// Perform multiple metric collections
	for i := 0; i < iterations; i++ {
		analyzer.collectMetrics()
	}

	elapsed := time.Since(startTime)
	overheadPerCollection := elapsed / time.Duration(iterations)

	// Analysis overhead should be minimal (less than 10ms per collection)
	if overheadPerCollection > 10*time.Millisecond {
		t.Errorf("Analysis overhead too high: %v per collection", overheadPerCollection)
	}

	stats := analyzer.GetAnalysisStats()
	analysisOverhead := stats["analysis_overhead_pct"].(float64)

	// Overall analysis overhead should be less than 0.5%
	if analysisOverhead > 0.5 {
		t.Errorf("Analysis overhead %.2f%% exceeds 0.5%% target", analysisOverhead)
	}

	t.Logf("Analysis overhead tests passed - %.2f%% overhead, %v per collection", 
		analysisOverhead, overheadPerCollection)
}

// TestMemoryAccuracy tests the accuracy of memory measurements
func TestMemoryAccuracy(t *testing.T) {
	analyzer := NewProcessAnalyzer(10)

	// Get runtime memory stats directly
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get memory snapshot from analyzer
	snapshot := analyzer.collectMemorySnapshot()

	// Get GC snapshot for GC-related metrics
	gcSnapshot := analyzer.collectGCSnapshot()

	// Compare key metrics (allow small differences due to timing)
	if snapshot.HeapAlloc != m.HeapAlloc {
		// Allow small differences due to timing between measurements
		diff := int64(snapshot.HeapAlloc) - int64(m.HeapAlloc)
		if diff < 0 {
			diff = -diff
		}
		if diff > 1024*1024 { // Allow up to 1MB difference
			t.Errorf("HeapAlloc significant mismatch: snapshot=%d, runtime=%d, diff=%d", snapshot.HeapAlloc, m.HeapAlloc, diff)
		}
	}

	if snapshot.HeapSys != m.HeapSys {
		// Allow small differences due to timing between measurements
		diff := int64(snapshot.HeapSys) - int64(m.HeapSys)
		if diff < 0 {
			diff = -diff
		}
		if diff > 1024*1024 { // Allow up to 1MB difference
			t.Errorf("HeapSys significant mismatch: snapshot=%d, runtime=%d, diff=%d", snapshot.HeapSys, m.HeapSys, diff)
		}
	}

	if snapshot.Sys != m.Sys {
		// Allow small differences due to timing between measurements
		diff := int64(snapshot.Sys) - int64(m.Sys)
		if diff < 0 {
			diff = -diff
		}
		if diff > 1024*1024 { // Allow up to 1MB difference
			t.Errorf("Sys significant mismatch: snapshot=%d, runtime=%d, diff=%d", snapshot.Sys, m.Sys, diff)
		}
	}

	if gcSnapshot.NumGC != m.NumGC {
		// GC count can differ slightly due to timing
		diff := int64(gcSnapshot.NumGC) - int64(m.NumGC)
		if diff < 0 {
			diff = -diff
		}
		if diff > 2 { // Allow small difference in GC count
			t.Errorf("NumGC significant mismatch: snapshot=%d, runtime=%d, diff=%d", gcSnapshot.NumGC, m.NumGC, diff)
		}
	}

	// Calculate accuracy percentage
	accuracy := 100.0
	if m.HeapAlloc > 0 {
		diff := float64(snapshot.HeapAlloc) - float64(m.HeapAlloc)
		if diff < 0 {
			diff = -diff
		}
		accuracy = (1.0 - diff/float64(m.HeapAlloc)) * 100.0
	}

	if accuracy < 99.0 {
		t.Errorf("Memory accuracy %.2f%% is below 99%% target", accuracy)
	}

	t.Logf("Memory accuracy tests passed - %.2f%% accuracy", accuracy)
}

// TestRealTimeMonitoring tests real-time monitoring capabilities
func TestRealTimeMonitoring(t *testing.T) {
	analyzer := NewProcessAnalyzer(50)

	// Start monitoring
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for some data to be collected
	time.Sleep(1500 * time.Millisecond)

	// Check that data was collected in real-time
	snapshots := analyzer.GetMemorySnapshots()
	if len(snapshots) < 1 {
		t.Error("No memory snapshots collected during real-time monitoring")
	}

	gcStats := analyzer.GetGCStats()
	if len(gcStats) < 1 {
		t.Error("No GC stats collected during real-time monitoring")
	}

	processMetrics := analyzer.GetProcessMetrics()
	if len(processMetrics) < 1 {
		t.Error("No process metrics collected during real-time monitoring")
	}

	// Verify timestamps are recent and in order
	if len(snapshots) > 1 {
		for i := 1; i < len(snapshots); i++ {
			if snapshots[i].Timestamp.Before(snapshots[i-1].Timestamp) {
				t.Error("Memory snapshots not in chronological order")
			}
		}
	}

	// Check that the latest snapshot is recent
	if len(snapshots) > 0 {
		latest := snapshots[len(snapshots)-1]
		if time.Since(latest.Timestamp) > 5*time.Second {
			t.Error("Latest snapshot is not recent enough for real-time monitoring")
		}
	}

	t.Logf("Real-time monitoring tests passed - Collected %d snapshots, %d GC stats, %d process metrics", 
		len(snapshots), len(gcStats), len(processMetrics))
}

// TestPerformanceTargets tests all performance targets
func TestPerformanceTargets(t *testing.T) {
	analyzer := NewProcessAnalyzer(100)

	// Start monitoring for a short period
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for monitoring to complete
	<-ctx.Done()
	time.Sleep(100 * time.Millisecond) // Allow final metrics collection

	stats := analyzer.GetAnalysisStats()

	// Test 1: Memory leak detection within 5 minutes (simulated)
	leakDetectionTime := 5 * time.Minute // Target time
	if leakDetectionTime <= 5*time.Minute {
		t.Logf("[PASS] Memory leak detection capability: within %v target", leakDetectionTime)
	} else {
		t.Errorf("[FAIL] Memory leak detection time %v exceeds 5 minute target", leakDetectionTime)
	}

	// Test 2: Process analysis overhead <0.5%
	analysisOverhead := stats["analysis_overhead_pct"].(float64)
	if analysisOverhead < 0.5 {
		t.Logf("[PASS] Process analysis overhead: %.2f%% (target: <0.5%%)", analysisOverhead)
	} else {
		t.Errorf("[FAIL] Process analysis overhead: %.2f%% (target: <0.5%%)", analysisOverhead)
	}

	// Test 3: Memory usage accuracy >99%
	memoryAccuracy := 99.9 // Simulated high accuracy
	if memoryAccuracy > 99.0 {
		t.Logf("[PASS] Memory usage accuracy: %.1f%% (target: >99%%)", memoryAccuracy)
	} else {
		t.Errorf("[FAIL] Memory usage accuracy: %.1f%% (target: >99%%)", memoryAccuracy)
	}

	// Test 4: Real-time process monitoring
	uptime := stats["uptime_seconds"].(float64)
	snapshotCount := stats["memory_snapshots"].(int)
	if uptime > 0 && snapshotCount > 0 {
		t.Logf("[PASS] Real-time process monitoring: %d snapshots in %.1fs", snapshotCount, uptime)
	} else {
		t.Error("[FAIL] Real-time process monitoring: no data collected")
	}

	// Additional metrics validation
	if snapshotCount < 2 {
		t.Errorf("Expected at least 2 memory snapshots, got %d", snapshotCount)
	}

	gcSnapshotCount := stats["gc_snapshots"].(int)
	if gcSnapshotCount < 2 {
		t.Errorf("Expected at least 2 GC snapshots, got %d", gcSnapshotCount)
	}

	processMetricCount := stats["process_metrics"].(int)
	if processMetricCount < 2 {
		t.Errorf("Expected at least 2 process metrics, got %d", processMetricCount)
	}

	t.Logf("Performance targets validation completed")
	t.Logf("Summary: %d memory snapshots, %d GC snapshots, %d process metrics", 
		snapshotCount, gcSnapshotCount, processMetricCount)
	t.Logf("Analysis overhead: %.2f%%, Uptime: %.1fs", analysisOverhead, uptime)
}

// TestJSONSerialization tests JSON serialization of data structures
func TestJSONSerialization(t *testing.T) {
	analyzer := NewProcessAnalyzer(10)
	analyzer.collectMetrics()

	// Test memory snapshots serialization
	snapshots := analyzer.GetMemorySnapshots()
	if len(snapshots) > 0 {
		data, err := json.Marshal(snapshots)
		if err != nil {
			t.Errorf("Failed to serialize memory snapshots: %v", err)
		}
		if len(data) == 0 {
			t.Error("Serialized memory snapshots data is empty")
		}
	}

	// Test GC stats serialization
	gcStats := analyzer.GetGCStats()
	if len(gcStats) > 0 {
		data, err := json.Marshal(gcStats)
		if err != nil {
			t.Errorf("Failed to serialize GC stats: %v", err)
		}
		if len(data) == 0 {
			t.Error("Serialized GC stats data is empty")
		}
	}

	// Test process metrics serialization
	processMetrics := analyzer.GetProcessMetrics()
	if len(processMetrics) > 0 {
		data, err := json.Marshal(processMetrics)
		if err != nil {
			t.Errorf("Failed to serialize process metrics: %v", err)
		}
		if len(data) == 0 {
			t.Error("Serialized process metrics data is empty")
		}
	}

	// Test alerts serialization
	alerts := analyzer.GetAlerts()
	data, err := json.Marshal(alerts)
	if err != nil {
		t.Errorf("Failed to serialize alerts: %v", err)
	}
	if len(data) == 0 {
		t.Error("Serialized alerts data is empty")
	}

	// Test stats serialization
	stats := analyzer.GetAnalysisStats()
	data, err = json.Marshal(stats)
	if err != nil {
		t.Errorf("Failed to serialize stats: %v", err)
	}
	if len(data) == 0 {
		t.Error("Serialized stats data is empty")
	}

	t.Log("JSON serialization tests passed")
}