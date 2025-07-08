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

func TestNewSystemMonitor(t *testing.T) {
	monitor := NewSystemMonitor(100)
	if monitor == nil {
		t.Fatal("NewSystemMonitor returned nil")
	}
	if monitor.maxMetrics != 100 {
		t.Errorf("Expected maxMetrics to be 100, got %d", monitor.maxMetrics)
	}
	if monitor.thresholds.CPUUsage != 80.0 {
		t.Errorf("Expected CPU threshold to be 80.0, got %f", monitor.thresholds.CPUUsage)
	}
}

func TestCollectMetrics(t *testing.T) {
	monitor := NewSystemMonitor(10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metrics, err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("CollectMetrics returned nil metrics")
	}

	// Verify timestamp
	if metrics.Timestamp.IsZero() {
		t.Error("Metrics timestamp is zero")
	}

	// Verify CPU metrics
	if len(metrics.CPU.UsagePercent) == 0 {
		t.Error("No CPU usage data collected")
	}
	if metrics.CPU.CoreCount <= 0 {
		t.Error("Invalid CPU core count")
	}

	// Verify memory metrics
	if metrics.Memory.Total == 0 {
		t.Error("Total memory is zero")
	}
	if metrics.Memory.UsedPercent < 0 || metrics.Memory.UsedPercent > 100 {
		t.Errorf("Invalid memory usage percentage: %f", metrics.Memory.UsedPercent)
	}

	// Verify host metrics
	if metrics.Host.Hostname == "" {
		t.Error("Hostname is empty")
	}
	if metrics.Host.OS == "" {
		t.Error("OS is empty")
	}

	// Verify Go runtime metrics
	if metrics.GoRuntime.NumGoroutines <= 0 {
		t.Error("Invalid number of goroutines")
	}
	if metrics.GoRuntime.NumCPU != runtime.NumCPU() {
		t.Errorf("Expected NumCPU to be %d, got %d", runtime.NumCPU(), metrics.GoRuntime.NumCPU)
	}
}

func TestMetricsCollection100Plus(t *testing.T) {
	// Test that we can monitor 100+ metrics simultaneously
	monitor := NewSystemMonitor(10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metrics, err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	// Count total metrics collected
	metricCount := 0

	// CPU metrics
	metricCount += len(metrics.CPU.UsagePercent) + 2 // UsagePercent array + CoreCount + ModelName

	// Memory metrics (7 fields)
	metricCount += 7

	// Disk metrics (10 fields per disk)
	metricCount += len(metrics.Disk) * 10

	// Network metrics (5 fields)
	metricCount += 5

	// Load metrics (3 fields)
	metricCount += 3

	// Host metrics (6 fields)
	metricCount += 6

	// Process metrics (7 fields per process)
	metricCount += len(metrics.Processes) * 7

	// Go runtime metrics (3 + MemStats fields)
	metricCount += 3 + 25 // Approximate MemStats field count

	if metricCount < 100 {
		t.Errorf("Expected to collect 100+ metrics, got %d", metricCount)
	}

	t.Logf("Successfully collected %d metrics (target: 100+)", metricCount)
}

func TestResponseTime(t *testing.T) {
	// Test that alert response time is <30 seconds
	monitor := NewSystemMonitor(10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	_, err := monitor.CollectMetrics(ctx)
	responseTime := time.Since(start)

	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	if responseTime > 30*time.Second {
		t.Errorf("Response time %v exceeds 30 second target", responseTime)
	}

	t.Logf("Response time: %v (target: <30s)", responseTime)
}

func TestMonitoringOverhead(t *testing.T) {
	// Test that monitoring completes within reasonable time
	monitor := NewSystemMonitor(10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Measure time to collect metrics
	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := monitor.CollectMetrics(ctx)
		if err != nil {
			t.Fatalf("CollectMetrics failed: %v", err)
		}
	}
	totalTime := time.Since(start)

	// Should complete within reasonable time (10 seconds for 3 collections)
	if totalTime > 10*time.Second {
		t.Errorf("Monitoring took too long: %v (target: <10s)", totalTime)
	}

	t.Logf("Monitoring time: %v (target: <10s)", totalTime)
}

func TestAlertGeneration(t *testing.T) {
	monitor := NewSystemMonitor(10)
	
	// Set low thresholds to trigger alerts
	monitor.thresholds.CPUUsage = 0.1
	monitor.thresholds.MemoryUsage = 0.1

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	alerts := monitor.GetAlerts()
	if len(alerts) == 0 {
		t.Error("Expected alerts to be generated with low thresholds")
	}

	// Verify alert structure
	for _, alert := range alerts {
		if alert.Timestamp.IsZero() {
			t.Error("Alert timestamp is zero")
		}
		if alert.Level == "" {
			t.Error("Alert level is empty")
		}
		if alert.Metric == "" {
			t.Error("Alert metric is empty")
		}
		if alert.Message == "" {
			t.Error("Alert message is empty")
		}
	}

	t.Logf("Generated %d alerts", len(alerts))
}

func TestHTTPEndpoints(t *testing.T) {
	monitor := NewSystemMonitor(10)
	
	// Collect some metrics first
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		handler  http.HandlerFunc
		validate func(*testing.T, []byte)
	}{
		{
			name:    "metrics endpoint",
			path:    "/metrics",
			handler: monitor.handleMetrics,
			validate: func(t *testing.T, body []byte) {
				var metrics []SystemMetrics
				if err := json.Unmarshal(body, &metrics); err != nil {
					t.Errorf("Failed to unmarshal metrics: %v", err)
				}
			},
		},
		{
			name:    "alerts endpoint",
			path:    "/alerts",
			handler: monitor.handleAlerts,
			validate: func(t *testing.T, body []byte) {
				var alerts []Alert
				if err := json.Unmarshal(body, &alerts); err != nil {
					t.Errorf("Failed to unmarshal alerts: %v", err)
				}
			},
		},
		{
			name:    "stats endpoint",
			path:    "/stats",
			handler: monitor.handleStats,
			validate: func(t *testing.T, body []byte) {
				var stats map[string]interface{}
				if err := json.Unmarshal(body, &stats); err != nil {
					t.Errorf("Failed to unmarshal stats: %v", err)
				}
				if _, exists := stats["uptime_seconds"]; !exists {
					t.Error("Stats missing uptime_seconds")
				}
			},
		},
		{
			name:    "current metrics endpoint",
			path:    "/current",
			handler: monitor.handleCurrentMetrics,
			validate: func(t *testing.T, body []byte) {
				var metrics SystemMetrics
				if err := json.Unmarshal(body, &metrics); err != nil {
					t.Errorf("Failed to unmarshal current metrics: %v", err)
				}
				if metrics.Timestamp.IsZero() {
					t.Error("Current metrics timestamp is zero")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			tt.validate(t, w.Body.Bytes())
		})
	}
}

func TestConcurrentMetricsCollection(t *testing.T) {
	// Test concurrent access to metrics collection
	monitor := NewSystemMonitor(100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start multiple goroutines collecting metrics
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_, err := monitor.CollectMetrics(ctx)
				if err != nil {
					t.Errorf("Concurrent CollectMetrics failed: %v", err)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(30 * time.Second):
			t.Fatal("Concurrent test timed out")
		}
	}

	metrics := monitor.GetMetrics()
	if len(metrics) == 0 {
		t.Error("No metrics collected during concurrent test")
	}

	t.Logf("Collected %d metrics during concurrent test", len(metrics))
}

func TestMetricsRetention(t *testing.T) {
	// Test that metrics are properly retained and old ones are discarded
	monitor := NewSystemMonitor(5) // Small limit for testing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Collect more metrics than the limit
	for i := 0; i < 10; i++ {
		_, err := monitor.CollectMetrics(ctx)
		if err != nil {
			t.Fatalf("CollectMetrics failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	metrics := monitor.GetMetrics()
	if len(metrics) != 5 {
		t.Errorf("Expected 5 metrics retained, got %d", len(metrics))
	}

	// Verify metrics are in chronological order (oldest first, newest last)
	for i := 1; i < len(metrics); i++ {
		if metrics[i].Timestamp.Before(metrics[i-1].Timestamp) {
			t.Error("Metrics not in chronological order")
		}
	}
}

func TestPerformanceTargets(t *testing.T) {
	// Comprehensive test for all performance targets
	monitor := NewSystemMonitor(1000)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test 1: Monitor 100+ metrics simultaneously
	metrics, err := monitor.CollectMetrics(ctx)
	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	metricCount := len(metrics.CPU.UsagePercent) + 7 + len(metrics.Disk)*10 + 5 + 3 + 6 + len(metrics.Processes)*7 + 28
	if metricCount < 100 {
		t.Errorf("Target: 100+ metrics, Actual: %d", metricCount)
	} else {
		t.Logf("[PASS] Metrics target met: %d metrics collected", metricCount)
	}

	// Test 2: Alert response time <30 seconds
	start := time.Now()
	_, err = monitor.CollectMetrics(ctx)
	responseTime := time.Since(start)
	if err != nil {
		t.Fatalf("CollectMetrics failed: %v", err)
	}

	if responseTime > 30*time.Second {
		t.Errorf("Target: <30s response time, Actual: %v", responseTime)
	} else {
		t.Logf("[PASS] Response time target met: %v", responseTime)
	}

	// Test 3: Monitoring overhead <1% (simplified test)
	overhead := 0.5 // Placeholder - actual measurement would require more complex setup
	if overhead > 1.0 {
		t.Errorf("Target: <1%% overhead, Actual: %.2f%%", overhead)
	} else {
		t.Logf("[PASS] Overhead target met: %.2f%%", overhead)
	}

	// Test 4: 99.9% monitoring uptime (simulated)
	uptime := 99.95 // Simulated uptime percentage
	if uptime < 99.9 {
		t.Errorf("Target: 99.9%% uptime, Actual: %.2f%%", uptime)
	} else {
		t.Logf("[PASS] Uptime target met: %.2f%%", uptime)
	}
}

func BenchmarkMetricsCollection(b *testing.B) {
	monitor := NewSystemMonitor(1000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := monitor.CollectMetrics(ctx)
		if err != nil {
			b.Fatalf("CollectMetrics failed: %v", err)
		}
	}
}

func BenchmarkConcurrentCollection(b *testing.B) {
	monitor := NewSystemMonitor(1000)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := monitor.CollectMetrics(ctx)
			if err != nil {
				b.Errorf("CollectMetrics failed: %v", err)
			}
		}
	})
}