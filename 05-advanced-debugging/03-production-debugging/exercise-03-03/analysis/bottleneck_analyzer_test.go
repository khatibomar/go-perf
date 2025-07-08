// analysis/bottleneck_analyzer_test.go
package analysis

import (
	"context"
	"testing"
	"time"
)

func TestBottleneckAnalyzer_BasicFunctionality(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Test initial state
	if analyzer == nil {
		t.Fatal("NewBottleneckAnalyzer returned nil")
	}

	// Test default baselines are set
	baseline := analyzer.getBaseline("cpu_usage")
	if baseline.Value == 0 {
		t.Error("Default CPU baseline not set")
	}

	// Test metrics collection
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for some metrics to be collected
	time.Sleep(time.Millisecond * 500)

	currentMetrics := analyzer.GetCurrentMetrics()
	if currentMetrics.Timestamp.IsZero() {
		t.Error("No metrics collected")
	}

	if currentMetrics.GoroutineCount <= 0 {
		t.Error("Invalid goroutine count in metrics")
	}
}

func TestBottleneckAnalyzer_CPUBottleneckDetection(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 50,
		AnalysisWindow:      time.Second * 2,
		ThresholdMultiplier: 1.5,
		MinSamples:          3,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Set a low CPU baseline to trigger bottleneck detection
	analyzer.UpdateBaseline("cpu_usage", 10.0, 5.0)

	bottleneckDetected := false
	analyzer.RegisterBottleneckCallback(func(bottleneck Bottleneck) {
		if bottleneck.Type == "performance" && bottleneck.Component == "cpu" {
			bottleneckDetected = true
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for analysis to run
	time.Sleep(time.Second * 2)

	if !bottleneckDetected {
		t.Error("CPU bottleneck not detected")
	}
}

func TestBottleneckAnalyzer_MemoryBottleneckDetection(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 50,
		AnalysisWindow:      time.Second * 2,
		ThresholdMultiplier: 1.5,
		MinSamples:          3,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Set a low memory baseline to trigger bottleneck detection
	analyzer.UpdateBaseline("memory_usage", 1024*1024, 512*1024) // 1MB ± 512KB

	bottleneckDetected := false
	analyzer.RegisterBottleneckCallback(func(bottleneck Bottleneck) {
		if bottleneck.Type == "resource" && bottleneck.Component == "memory" {
			bottleneckDetected = true
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for analysis to run
	time.Sleep(time.Second * 2)

	if !bottleneckDetected {
		t.Error("Memory bottleneck not detected")
	}
}

func TestBottleneckAnalyzer_GoroutineBottleneckDetection(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 50,
		AnalysisWindow:      time.Second * 2,
		ThresholdMultiplier: 1.5,
		MinSamples:          3,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Set a low goroutine baseline to trigger bottleneck detection
	analyzer.UpdateBaseline("goroutine_count", 5.0, 2.0)

	bottleneckDetected := false
	analyzer.RegisterBottleneckCallback(func(bottleneck Bottleneck) {
		if bottleneck.Type == "concurrency" && bottleneck.Component == "goroutines" {
			bottleneckDetected = true
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	go analyzer.Start(ctx)

	// Create additional goroutines to trigger detection
	for i := 0; i < 20; i++ {
		go func() {
			time.Sleep(time.Second * 2)
		}()
	}

	// Wait for analysis to run
	time.Sleep(time.Second * 2)

	if !bottleneckDetected {
		t.Error("Goroutine bottleneck not detected")
	}
}

func TestBottleneckAnalyzer_AnalyzeNow(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 5,
		ThresholdMultiplier: 1.5,
		MinSamples:          3,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Collect some metrics first
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	go analyzer.Start(ctx)
	time.Sleep(time.Millisecond * 400)

	// Test immediate analysis
	start := time.Now()
	bottlenecks := analyzer.AnalyzeNow()
	analysisTime := time.Since(start)

	// Verify analysis completes quickly (performance requirement)
	if analysisTime > time.Second*5 {
		t.Errorf("Analysis took too long: %v (should be <5 seconds)", analysisTime)
	}

	// Should return empty slice if no bottlenecks detected
	if bottlenecks == nil {
		t.Error("AnalyzeNow returned nil")
	}
}

func TestBottleneckAnalyzer_BaselineUpdate(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Test baseline update
	analyzer.UpdateBaseline("test_metric", 100.0, 10.0)

	baseline := analyzer.getBaseline("test_metric")
	if baseline.Value != 100.0 {
		t.Errorf("Baseline value not updated: got %f, want 100.0", baseline.Value)
	}

	if baseline.StdDev != 10.0 {
		t.Errorf("Baseline stddev not updated: got %f, want 10.0", baseline.StdDev)
	}
}

func TestBottleneckAnalyzer_RecentBottlenecks(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 50,
		AnalysisWindow:      time.Second * 2,
		ThresholdMultiplier: 1.5,
		MinSamples:          3,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Set low baselines to trigger bottlenecks
	analyzer.UpdateBaseline("cpu_usage", 10.0, 5.0)
	analyzer.UpdateBaseline("memory_usage", 1024*1024, 512*1024)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for bottlenecks to be detected
	time.Sleep(time.Second)

	// Get recent bottlenecks
	bottlenecks := analyzer.GetRecentBottlenecks(time.Hour)

	if len(bottlenecks) == 0 {
		t.Error("No recent bottlenecks found")
	}

	// Verify bottlenecks are sorted by timestamp (newest first)
	for i := 1; i < len(bottlenecks); i++ {
		if bottlenecks[i].Timestamp.After(bottlenecks[i-1].Timestamp) {
			t.Error("Bottlenecks not sorted by timestamp (newest first)")
		}
	}
}

func TestBottleneckAnalyzer_SeverityCalculation(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}

	analyzer := NewBottleneckAnalyzer(config)

	tests := []struct {
		current   float64
		threshold float64
		expected  string
	}{
		{100, 100, "low"},
		{150, 100, "medium"},
		{200, 100, "high"},
		{300, 100, "critical"},
	}

	for _, test := range tests {
		severity := analyzer.calculateSeverity(test.current, test.threshold)
		if severity != test.expected {
			t.Errorf("calculateSeverity(%f, %f) = %s, want %s",
				test.current, test.threshold, severity, test.expected)
		}
	}
}

func TestBottleneckAnalyzer_MetricsRetention(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 10,
		AnalysisWindow:      time.Millisecond * 100,
		ThresholdMultiplier: 2.0,
		MinSamples:          3,
	}

	analyzer := NewBottleneckAnalyzer(config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for metrics collection
	time.Sleep(time.Millisecond * 150)

	analyzer.mu.RLock()
	metricsCount := len(analyzer.metrics)
	analyzer.mu.RUnlock()

	// Should have limited metrics due to retention window
	if metricsCount > 20 { // Generous upper bound
		t.Errorf("Too many metrics retained: %d (analysis window should limit retention)", metricsCount)
	}

	if metricsCount == 0 {
		t.Error("No metrics collected")
	}
}

// Performance Tests

func TestBottleneckAnalyzer_PerformanceRequirements(t *testing.T) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 10,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Collect metrics for analysis
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go analyzer.Start(ctx)
	time.Sleep(time.Millisecond * 800)

	// Test bottleneck identification time (should be <5 minutes)
	start := time.Now()
	bottlenecks := analyzer.AnalyzeNow()
	identificationTime := time.Since(start)

	if identificationTime > 5*time.Minute {
		t.Errorf("Bottleneck identification took %v, requirement: <5 minutes", identificationTime)
	}

	// Test analysis overhead (should be <0.5%)
	// This is measured by the time spent in analysis vs total time
	analysisOverhead := float64(identificationTime.Nanoseconds()) / float64(time.Second.Nanoseconds()) * 100
	if analysisOverhead > 0.5 {
		t.Errorf("Analysis overhead: %.4f%%, requirement: <0.5%%", analysisOverhead)
	}

	// Verify analysis returns results
	if bottlenecks == nil {
		t.Error("AnalyzeNow returned nil")
	}

	t.Logf("Performance metrics:")
	t.Logf("  Identification time: %v (target: <5 minutes)", identificationTime)
	t.Logf("  Analysis overhead: %.4f%% (target: <0.5%%)", analysisOverhead)
	t.Logf("  Bottlenecks found: %d", len(bottlenecks))
}

// Benchmark Tests

func BenchmarkBottleneckAnalyzer_CollectMetrics(b *testing.B) {
	config := BottleneckConfig{
		SampleInterval:      time.Second,
		AnalysisWindow:      time.Minute,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}

	analyzer := NewBottleneckAnalyzer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.collectMetrics()
	}
}

func BenchmarkBottleneckAnalyzer_AnalyzeBottlenecks(b *testing.B) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 10,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Collect some metrics first
	for i := 0; i < 20; i++ {
		analyzer.collectMetrics()
		time.Sleep(time.Millisecond * 10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.analyzeBottlenecks()
	}
}

func BenchmarkBottleneckAnalyzer_AnalyzeNow(b *testing.B) {
	config := BottleneckConfig{
		SampleInterval:      time.Millisecond * 100,
		AnalysisWindow:      time.Second * 10,
		ThresholdMultiplier: 2.0,
		MinSamples:          5,
	}

	analyzer := NewBottleneckAnalyzer(config)

	// Collect some metrics first
	for i := 0; i < 20; i++ {
		analyzer.collectMetrics()
		time.Sleep(time.Millisecond * 10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bottlenecks := analyzer.AnalyzeNow()
		_ = bottlenecks
	}
}