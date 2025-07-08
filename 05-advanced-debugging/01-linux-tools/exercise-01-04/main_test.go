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

func TestAdvancedProfilerCreation(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	if profiler == nil {
		t.Fatal("Failed to create profiler")
	}
	
	if profiler.maxEvents != 1000 {
		t.Errorf("Expected maxEvents=1000, got %d", profiler.maxEvents)
	}
	
	if len(profiler.events) != 0 {
		t.Errorf("Expected empty events slice, got length %d", len(profiler.events))
	}
	
	if len(profiler.functionStats) != 0 {
		t.Errorf("Expected empty functionStats map, got length %d", len(profiler.functionStats))
	}
}

func TestProfilerStartStop(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	
	// Test initial state
	if profiler.isRunning {
		t.Error("Profiler should not be running initially")
	}
	
	// Test start
	profiler.Start()
	if !profiler.isRunning {
		t.Error("Profiler should be running after Start()")
	}
	
	// Test double start (should not panic)
	profiler.Start()
	if !profiler.isRunning {
		t.Error("Profiler should still be running after double Start()")
	}
	
	// Test stop
	profiler.Stop()
	if profiler.isRunning {
		t.Error("Profiler should not be running after Stop()")
	}
	
	// Test double stop (should not panic)
	profiler.Stop()
	if profiler.isRunning {
		t.Error("Profiler should still not be running after double Stop()")
	}
}

func TestEventRecording(t *testing.T) {
	profiler := NewAdvancedProfiler(100)
	profiler.Start()
	defer profiler.Stop()
	
	// Record a test event
	duration := time.Millisecond * 5
	profiler.RecordEvent("test.function", duration, "test")
	
	// Check event was recorded
	events := profiler.GetEvents(10)
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
	
	event := events[0]
	if event.Function != "test.function" {
		t.Errorf("Expected function='test.function', got '%s'", event.Function)
	}
	
	if event.Duration != duration.Nanoseconds() {
		t.Errorf("Expected duration=%d, got %d", duration.Nanoseconds(), event.Duration)
	}
	
	if event.EventType != "test" {
		t.Errorf("Expected eventType='test', got '%s'", event.EventType)
	}
	
	// Check function statistics
	stats := profiler.GetStats()
	funcStats, exists := stats.FunctionStats["test.function"]
	if !exists {
		t.Error("Function statistics not found")
	}
	
	if funcStats.Count != 1 {
		t.Errorf("Expected count=1, got %d", funcStats.Count)
	}
	
	if funcStats.TotalDuration != duration.Nanoseconds() {
		t.Errorf("Expected totalDuration=%d, got %d", duration.Nanoseconds(), funcStats.TotalDuration)
	}
}

func TestCircularBuffer(t *testing.T) {
	profiler := NewAdvancedProfiler(3) // Small buffer for testing
	profiler.Start()
	defer profiler.Stop()
	
	// Record more events than buffer size
	for i := 0; i < 5; i++ {
		profiler.RecordEvent("test.function", time.Microsecond*time.Duration(i+1), "test")
	}
	
	// Should only have 3 events (buffer size)
	events := profiler.GetEvents(10)
	if len(events) != 3 {
		t.Errorf("Expected 3 events in buffer, got %d", len(events))
	}
	
	// Should have the last 3 events (3, 4, 5 microseconds)
	expectedDurations := []int64{3000, 4000, 5000} // in nanoseconds
	for i, event := range events {
		if event.Duration != expectedDurations[i] {
			t.Errorf("Event %d: expected duration=%d, got %d", i, expectedDurations[i], event.Duration)
		}
	}
}

func TestFunctionStatistics(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	profiler.Start()
	defer profiler.Stop()
	
	// Record multiple events for the same function
	durations := []time.Duration{
		time.Microsecond * 100,
		time.Microsecond * 200,
		time.Microsecond * 300,
	}
	
	for _, duration := range durations {
		profiler.RecordEvent("test.function", duration, "test")
	}
	
	stats := profiler.GetStats()
	funcStats := stats.FunctionStats["test.function"]
	
	if funcStats.Count != 3 {
		t.Errorf("Expected count=3, got %d", funcStats.Count)
	}
	
	expectedTotal := int64(600000) // 100+200+300 microseconds in nanoseconds
	if funcStats.TotalDuration != expectedTotal {
		t.Errorf("Expected totalDuration=%d, got %d", expectedTotal, funcStats.TotalDuration)
	}
	
	expectedMin := int64(100000) // 100 microseconds in nanoseconds
	if funcStats.MinDuration != expectedMin {
		t.Errorf("Expected minDuration=%d, got %d", expectedMin, funcStats.MinDuration)
	}
	
	expectedMax := int64(300000) // 300 microseconds in nanoseconds
	if funcStats.MaxDuration != expectedMax {
		t.Errorf("Expected maxDuration=%d, got %d", expectedMax, funcStats.MaxDuration)
	}
	
	expectedAvg := float64(expectedTotal) / 3
	if funcStats.AvgDuration != expectedAvg {
		t.Errorf("Expected avgDuration=%.2f, got %.2f", expectedAvg, funcStats.AvgDuration)
	}
}

func TestSystemMetrics(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	profiler.Start()
	defer profiler.Stop()
	
	// Wait a bit for background monitoring
	time.Sleep(time.Millisecond * 200)
	
	stats := profiler.GetStats()
	if stats.SystemMetrics == nil {
		t.Fatal("SystemMetrics should not be nil")
	}
	
	// Check that metrics are reasonable
	if stats.SystemMetrics.CPUUsage < 0 || stats.SystemMetrics.CPUUsage > 100 {
		t.Errorf("CPU usage should be between 0-100%%, got %.2f", stats.SystemMetrics.CPUUsage)
	}
	
	if stats.SystemMetrics.MemoryUsage <= 0 {
		t.Errorf("Memory usage should be positive, got %d", stats.SystemMetrics.MemoryUsage)
	}
	
	if stats.SystemMetrics.Goroutines <= 0 {
		t.Errorf("Goroutine count should be positive, got %d", stats.SystemMetrics.Goroutines)
	}
}

func TestProfilingOverhead(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	profiler.Start()
	defer profiler.Stop()
	
	// Record many events to measure overhead
	for i := 0; i < 1000; i++ {
		profiler.RecordEvent("test.function", time.Microsecond, "test")
	}
	
	// Wait a bit for overhead calculation
	time.Sleep(time.Millisecond * 100)
	
	stats := profiler.GetStats()
	
	// Overhead should be reasonable (< 15%)
	if stats.ProfilingOverhead > 15.0 {
		t.Errorf("Profiling overhead too high: %.4f%% (should be < 15%%)", stats.ProfilingOverhead)
	}
	
	// Events per second should be reasonable
	if stats.EventsPerSecond < 100 {
		t.Errorf("Events per second too low: %.2f (should be > 100)", stats.EventsPerSecond)
	}
}

func TestConcurrentAccess(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	profiler.Start()
	defer profiler.Stop()
	
	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 100
	
	// Start multiple goroutines recording events concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				profiler.RecordEvent("concurrent.function", time.Microsecond*time.Duration(j+1), "test")
			}
		}(i)
	}
	
	wg.Wait()
	
	// Check that all events were recorded properly
	stats := profiler.GetStats()
	if stats.EventsCollected < int64(numGoroutines*eventsPerGoroutine) {
		t.Errorf("Expected at least %d events, got %d", numGoroutines*eventsPerGoroutine, stats.EventsCollected)
	}
	
	funcStats := stats.FunctionStats["concurrent.function"]
	if funcStats.Count < int64(numGoroutines*eventsPerGoroutine) {
		t.Errorf("Expected at least %d function calls, got %d", numGoroutines*eventsPerGoroutine, funcStats.Count)
	}
}

func TestHTTPHandlers(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	profiler.Start()
	defer profiler.Stop()
	
	// Record some test events
	profiler.RecordEvent("test.function", time.Millisecond, "test")
	time.Sleep(time.Millisecond * 100) // Let background monitoring run
	
	// Test stats handler
	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	profiler.statsHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
	
	// Parse response
	var stats ProfilerStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}
	
	if stats.EventsCollected <= 0 {
		t.Errorf("Expected events collected > 0, got %d", stats.EventsCollected)
	}
	
	// Test events handler
	req = httptest.NewRequest("GET", "/events", nil)
	w = httptest.NewRecorder()
	profiler.eventsHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var events []ProfileEvent
	err = json.Unmarshal(w.Body.Bytes(), &events)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}
	
	if len(events) == 0 {
		t.Error("Expected at least one event")
	}
	
	// Test dashboard handler
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	profiler.dashboardHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}
	
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Dashboard should return HTML content")
	}
}

func TestBackgroundMonitoring(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	profiler.Start()
	defer profiler.Stop()
	
	// Wait for background monitoring to collect events
	time.Sleep(time.Second)
	
	stats := profiler.GetStats()
	
	// Should have collected various types of events
	expectedEventTypes := []string{"system.monitor", "syscall.read", "syscall.write", "runtime.mallocgc", "goroutine.create"}
	found := make(map[string]bool)
	
	for funcName := range stats.FunctionStats {
		for _, eventType := range expectedEventTypes {
			if funcName == eventType {
				found[eventType] = true
			}
		}
	}
	
	// Should have found at least some background events
	if len(found) < 3 {
		t.Errorf("Expected at least 3 types of background events, found %d: %v", len(found), found)
	}
}

func TestPerformanceTargets(t *testing.T) {
	profiler := NewAdvancedProfiler(10000)
	profiler.Start()
	defer profiler.Stop()
	
	// Performance test: record many events quickly
	numEvents := 10000
	start := time.Now()
	
	for i := 0; i < numEvents; i++ {
		profiler.RecordEvent("perf.test", time.Microsecond, "performance")
	}
	
	elapsed := time.Since(start)
	eventsPerSecond := float64(numEvents) / elapsed.Seconds()
	
	// Should be able to handle at least 10,000 events per second
	if eventsPerSecond < 10000 {
		t.Errorf("Performance target not met: %.0f events/sec (target: 10,000+)", eventsPerSecond)
	}
	
	// Wait a bit for overhead calculation
	time.Sleep(time.Millisecond * 100)
	
	stats := profiler.GetStats()
	
	// Profiling overhead should be reasonable (< 10%)
	if stats.ProfilingOverhead > 10.0 {
		t.Errorf("Overhead target not met: %.4f%% (target: < 10%%)", stats.ProfilingOverhead)
	}
	
	// Memory usage should be reasonable
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := float64(m.Alloc) / 1024 / 1024
	
	// Should use less than 50MB for this test
	if memoryMB > 50 {
		t.Errorf("Memory usage too high: %.2f MB (target: < 50MB)", memoryMB)
	}
}

func TestContextCancellation(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	profiler.Start()
	
	// Wait for background goroutines to start
	time.Sleep(time.Millisecond * 100)
	
	initialGoroutines := runtime.NumGoroutine()
	
	// Stop profiler
	profiler.Stop()
	
	// Wait for goroutines to clean up
	time.Sleep(time.Millisecond * 200)
	
	finalGoroutines := runtime.NumGoroutine()
	
	// Should have fewer goroutines after stopping (background monitoring should stop)
	if finalGoroutines >= initialGoroutines {
		t.Errorf("Goroutines not cleaned up properly: initial=%d, final=%d", initialGoroutines, finalGoroutines)
	}
}

func TestZeroOverheadWhenStopped(t *testing.T) {
	profiler := NewAdvancedProfiler(1000)
	
	// Record events when profiler is stopped
	start := time.Now()
	for i := 0; i < 1000; i++ {
		profiler.RecordEvent("test.function", time.Microsecond, "test")
	}
	elapsed := time.Since(start)
	
	// Should be very fast when profiler is stopped
	if elapsed > time.Millisecond*10 {
		t.Errorf("Recording events when stopped took too long: %v (should be < 10ms)", elapsed)
	}
	
	// Should have recorded events even when stopped
	stats := profiler.GetStats()
	if stats.EventsCollected != 1000 {
		t.Errorf("Expected 1000 events recorded, got %d", stats.EventsCollected)
	}
}

func BenchmarkEventRecording(b *testing.B) {
	profiler := NewAdvancedProfiler(10000)
	profiler.Start()
	defer profiler.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.RecordEvent("bench.function", time.Microsecond, "benchmark")
	}
}

func BenchmarkConcurrentEventRecording(b *testing.B) {
	profiler := NewAdvancedProfiler(10000)
	profiler.Start()
	defer profiler.Stop()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			profiler.RecordEvent("bench.concurrent", time.Microsecond, "benchmark")
		}
	})
}