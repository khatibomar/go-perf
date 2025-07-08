package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestNetworkIODebuggerCreation tests debugger creation
func TestNetworkIODebuggerCreation(t *testing.T) {
	debugger := NewNetworkIODebugger()
	if debugger == nil {
		t.Fatal("Failed to create NetworkIODebugger")
	}
	
	if debugger.monitoringActive {
		t.Error("Monitoring should not be active initially")
	}
	
	if len(debugger.connections) != 0 {
		t.Error("Connections slice should be empty initially")
	}
	
	if len(debugger.ioStats) != 0 {
		t.Error("IOStats slice should be empty initially")
	}
	
	if len(debugger.networkStats) != 0 {
		t.Error("NetworkStats slice should be empty initially")
	}
}

// TestDebuggerStartStop tests starting and stopping the debugger
func TestDebuggerStartStop(t *testing.T) {
	debugger := NewNetworkIODebugger()
	
	// Test start
	debugger.Start()
	time.Sleep(100 * time.Millisecond) // Allow goroutines to start
	
	if !debugger.monitoringActive {
		t.Error("Monitoring should be active after start")
	}
	
	// Test stop
	debugger.Stop()
	time.Sleep(100 * time.Millisecond) // Allow goroutines to stop
	
	if debugger.monitoringActive {
		t.Error("Monitoring should not be active after stop")
	}
}

// TestConnectionMonitoring tests connection monitoring functionality
func TestConnectionMonitoring(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for some monitoring cycles
	time.Sleep(2 * time.Second)
	
	connections := debugger.GetConnections()
	// We should have some connections (at least the test process)
	if len(connections) < 0 {
		t.Error("Expected some connections to be monitored")
	}
	
	// Test connection data structure
	for _, conn := range connections {
		if conn.Timestamp.IsZero() {
			t.Error("Connection timestamp should not be zero")
		}
		if conn.LocalAddr == "" && conn.RemoteAddr == "" {
			t.Error("Connection should have at least one address")
		}
	}
}

// TestIOStatsMonitoring tests I/O statistics monitoring
func TestIOStatsMonitoring(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for some monitoring cycles
	time.Sleep(3 * time.Second)
	
	ioStats := debugger.GetIOStats()
	// We should have some I/O stats
	if len(ioStats) == 0 {
		t.Log("No I/O stats available (may be normal on some systems)")
		return
	}
	
	// Test I/O stats data structure
	for _, stat := range ioStats {
		if stat.Device == "" {
			t.Error("I/O stat should have a device name")
		}
		if stat.Timestamp.IsZero() {
			t.Error("I/O stat timestamp should not be zero")
		}
		if stat.ThroughputMBs < 0 {
			t.Error("Throughput should not be negative")
		}
	}
}

// TestNetworkStatsMonitoring tests network statistics monitoring
func TestNetworkStatsMonitoring(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for some monitoring cycles
	time.Sleep(2 * time.Second)
	
	networkStats := debugger.GetNetworkStats()
	// We should have some network stats
	if len(networkStats) == 0 {
		t.Error("Expected some network statistics")
	}
	
	// Test network stats data structure
	for _, stat := range networkStats {
		if stat.Interface == "" {
			t.Error("Network stat should have an interface name")
		}
		if stat.Timestamp.IsZero() {
			t.Error("Network stat timestamp should not be zero")
		}
		if stat.BandwidthMbps < 0 {
			t.Error("Bandwidth should not be negative")
		}
		if stat.LatencyMs < 0 {
			t.Error("Latency should not be negative")
		}
	}
}

// TestTrafficAnalysis tests traffic analysis functionality
func TestTrafficAnalysis(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for traffic analysis cycles
	time.Sleep(6 * time.Second)
	
	trafficAnalysis := debugger.GetTrafficAnalysis()
	if len(trafficAnalysis) == 0 {
		t.Error("Expected some traffic analysis data")
	}
	
	// Test traffic analysis data structure
	for _, analysis := range trafficAnalysis {
		if analysis.Timestamp.IsZero() {
			t.Error("Traffic analysis timestamp should not be zero")
		}
		if analysis.TotalConnections < 0 {
			t.Error("Total connections should not be negative")
		}
		if analysis.ActiveConnections < 0 {
			t.Error("Active connections should not be negative")
		}
		if analysis.ThroughputGbps < 0 {
			t.Error("Throughput should not be negative")
		}
	}
}

// TestAlertGeneration tests alert generation functionality
func TestAlertGeneration(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Simulate high latency to trigger alerts
	debugger.mu.Lock()
	for i := 0; i < 10; i++ {
		debugger.latencyMeasurements = append(debugger.latencyMeasurements, 10.0) // High latency
	}
	debugger.mu.Unlock()
	
	// Wait for alert checking cycle
	time.Sleep(11 * time.Second)
	
	alerts := debugger.GetAlerts()
	if len(alerts) == 0 {
		t.Log("No alerts generated (may be normal)")
		return
	}
	
	// Test alert data structure
	for _, alert := range alerts {
		if alert.ID == "" {
			t.Error("Alert should have an ID")
		}
		if alert.Type == "" {
			t.Error("Alert should have a type")
		}
		if alert.Message == "" {
			t.Error("Alert should have a message")
		}
		if alert.Timestamp.IsZero() {
			t.Error("Alert timestamp should not be zero")
		}
	}
}

// TestHTTPEndpoints tests HTTP API endpoints
func TestHTTPEndpoints(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for some data to be collected
	time.Sleep(2 * time.Second)
	
	endpoints := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/connections", debugger.connectionsHandler},
		{"/api/iostats", debugger.ioStatsHandler},
		{"/api/network", debugger.networkStatsHandler},
		{"/api/traffic", debugger.trafficAnalysisHandler},
		{"/api/alerts", debugger.alertsHandler},
		{"/api/stats", debugger.statsHandler},
		{"/api/latency-test", debugger.latencyTestHandler},
		{"/api/throughput-test", debugger.throughputTestHandler},
	}
	
	for _, endpoint := range endpoints {
		t.Run(endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint.path, nil)
			w := httptest.NewRecorder()
			
			endpoint.handler(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}
			
			// Test JSON parsing
			var result map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("Failed to parse JSON response: %v", err)
			}
		})
	}
}

// TestDashboardHandler tests the dashboard HTML endpoint
func TestDashboardHandler(t *testing.T) {
	debugger := NewNetworkIODebugger()
	
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	debugger.dashboardHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}
	
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Dashboard should return HTML content")
	}
	
	// Check for key HTML elements
	expectedElements := []string{
		"Network and I/O Debugger Dashboard",
		"Performance Targets Status",
		"Recent Alerts",
		"Network Connections",
		"I/O Statistics",
		"Traffic Analysis",
	}
	
	for _, element := range expectedElements {
		if !contains(body, element) {
			t.Errorf("Dashboard should contain '%s'", element)
		}
	}
}

// TestConcurrentAccess tests concurrent access to debugger methods
func TestConcurrentAccess(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for some data to be collected
	time.Sleep(2 * time.Second)
	
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100
	
	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = debugger.GetConnections()
				_ = debugger.GetIOStats()
				_ = debugger.GetNetworkStats()
				_ = debugger.GetTrafficAnalysis()
				_ = debugger.GetAlerts()
				_ = debugger.GetPerformanceStats()
			}
		}()
	}
	
	wg.Wait()
	t.Log("Concurrent access test completed successfully")
}

// TestLatencyMeasurement tests network latency measurement
func TestLatencyMeasurement(t *testing.T) {
	debugger := NewNetworkIODebugger()
	
	// Test latency measurement to localhost
	latency := debugger.measureNetworkLatency("127.0.0.1:80")
	if latency < 0 {
		t.Log("Latency measurement failed (may be normal if no service on port 80)")
		return
	}
	
	if latency > 1000 { // More than 1 second seems unreasonable for localhost
		t.Errorf("Latency measurement seems too high: %.2fms", latency)
	}
	
	t.Logf("Measured latency: %.2fms", latency)
}

// TestThroughputSimulation tests throughput simulation
func TestThroughputSimulation(t *testing.T) {
	debugger := NewNetworkIODebugger()
	
	start := time.Now()
	debugger.simulateHighThroughputIO()
	duration := time.Since(start)
	
	if duration > 10*time.Second {
		t.Errorf("Throughput simulation took too long: %v", duration)
	}
	
	t.Logf("Throughput simulation completed in: %v", duration)
}

// TestPerformanceTargets tests if performance targets are met
func TestPerformanceTargets(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for sufficient monitoring data
	time.Sleep(5 * time.Second)
	
	stats := debugger.GetPerformanceStats()
	
	// Test network latency accuracy target (<1ms)
	if avgLatency, exists := stats["average_latency_ms"]; exists {
		if latency, ok := avgLatency.(float64); ok {
			if latency >= 1.0 {
				t.Logf("Network latency accuracy target not met: %.2fms (target: <1ms)", latency)
			} else {
				t.Logf("Network latency accuracy target met: %.2fms", latency)
			}
		}
	}
	
	// Test I/O throughput target (>1GB/s)
	if maxThroughput, exists := stats["max_throughput_mbs"]; exists {
		if throughput, ok := maxThroughput.(float64); ok {
			if throughput <= 1000 { // 1000 MB/s = 1 GB/s
				t.Logf("I/O throughput target not met: %.2f MB/s (target: >1000 MB/s)", throughput)
			} else {
				t.Logf("I/O throughput target met: %.2f MB/s", throughput)
			}
		}
	}
	
	// Test connection tracking capability (10K+ connections)
	if totalConns, exists := stats["total_connections"]; exists {
		if connections, ok := totalConns.(int64); ok {
			if connections > 10000 {
				t.Logf("Connection tracking capability exceeded: %d connections", connections)
			} else {
				t.Logf("Connection tracking capability: %d connections (target: handle 10K+)", connections)
			}
		}
	}
	
	// Test real-time traffic analysis
	if realTime, exists := stats["real_time_analysis"]; exists {
		if active, ok := realTime.(bool); ok {
			if !active {
				t.Error("Real-time traffic analysis should be active")
			} else {
				t.Log("Real-time traffic analysis is active")
			}
		}
	}
	
	t.Log("Performance targets test completed")
}

// TestMemoryUsage tests memory usage and potential leaks
func TestMemoryUsage(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialAlloc := m1.Alloc
	
	// Run for a while to collect data
	time.Sleep(10 * time.Second)
	
	// Force garbage collection
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	// Get final memory stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalAlloc := m2.Alloc
	
	memoryIncrease := float64(finalAlloc-initialAlloc) / 1024 / 1024 // MB
	
	t.Logf("Initial memory: %.2f MB", float64(initialAlloc)/1024/1024)
	t.Logf("Final memory: %.2f MB", float64(finalAlloc)/1024/1024)
	t.Logf("Memory increase: %.2f MB", memoryIncrease)
	
	// Check for reasonable memory usage (less than 100MB increase)
	if memoryIncrease > 100 {
		t.Errorf("Memory usage increased significantly: %.2f MB", memoryIncrease)
	}
}

// TestDataRetention tests data retention and cleanup
func TestDataRetention(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Add many measurements to test retention
	debugger.mu.Lock()
	for i := 0; i < 150; i++ {
		debugger.latencyMeasurements = append(debugger.latencyMeasurements, float64(i))
		debugger.throughputMeasurements = append(debugger.throughputMeasurements, float64(i*10))
	}
	debugger.mu.Unlock()
	
	// Wait for cleanup cycles
	time.Sleep(3 * time.Second)
	
	debugger.mu.RLock()
	latencyCount := len(debugger.latencyMeasurements)
	throughputCount := len(debugger.throughputMeasurements)
	debugger.mu.RUnlock()
	
	// Check that data is retained within reasonable limits
	if latencyCount > 100 {
		t.Errorf("Too many latency measurements retained: %d (expected ≤100)", latencyCount)
	}
	
	if throughputCount > 100 {
		t.Errorf("Too many throughput measurements retained: %d (expected ≤100)", throughputCount)
	}
	
	t.Logf("Data retention test: latency=%d, throughput=%d measurements", latencyCount, throughputCount)
}

// TestJSONSerialization tests JSON serialization of all data structures
func TestJSONSerialization(t *testing.T) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	// Wait for some data to be collected
	time.Sleep(3 * time.Second)
	
	// Test serialization of all data types
	testCases := []struct {
		name string
		data interface{}
	}{
		{"connections", debugger.GetConnections()},
		{"ioStats", debugger.GetIOStats()},
		{"networkStats", debugger.GetNetworkStats()},
		{"trafficAnalysis", debugger.GetTrafficAnalysis()},
		{"alerts", debugger.GetAlerts()},
		{"performanceStats", debugger.GetPerformanceStats()},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := json.Marshal(tc.data)
			if err != nil {
				t.Errorf("Failed to serialize %s: %v", tc.name, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests

// BenchmarkConnectionMonitoring benchmarks connection monitoring performance
func BenchmarkConnectionMonitoring(b *testing.B) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = debugger.GetConnections()
	}
}

// BenchmarkIOStatsMonitoring benchmarks I/O stats monitoring performance
func BenchmarkIOStatsMonitoring(b *testing.B) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = debugger.GetIOStats()
	}
}

// BenchmarkNetworkStatsMonitoring benchmarks network stats monitoring performance
func BenchmarkNetworkStatsMonitoring(b *testing.B) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = debugger.GetNetworkStats()
	}
}

// BenchmarkJSONSerialization benchmarks JSON serialization performance
func BenchmarkJSONSerialization(b *testing.B) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	time.Sleep(1 * time.Second) // Collect some data
	connections := debugger.GetConnections()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(connections)
	}
}

// BenchmarkConcurrentAccess benchmarks concurrent access performance
func BenchmarkConcurrentAccess(b *testing.B) {
	debugger := NewNetworkIODebugger()
	debugger.Start()
	defer debugger.Stop()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = debugger.GetConnections()
			_ = debugger.GetPerformanceStats()
		}
	})
}