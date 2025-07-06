package tests

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"exercise-03-04/analyzer"
	"exercise-03-04/detector"
	"exercise-03-04/simulator"
)

// BenchmarkHeapAnalyzer benchmarks the heap analyzer performance
func BenchmarkHeapAnalyzer(b *testing.B) {
	config := &detector.HeapConfig{
		Interval:  100 * time.Millisecond,
		Threshold: 0.1,
		Verbose:   false,
	}

	analyzer := detector.NewHeapAnalyzer(config)
	analyzer.Start()
	defer analyzer.Stop()

	// Let it collect some initial metrics
	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysis, err := analyzer.AnalyzeHeap()
		if err != nil || analysis == nil {
			b.Fatal("Analysis should not be nil")
		}
	}
}

// BenchmarkGoroutineDetector benchmarks the goroutine detector performance
func BenchmarkGoroutineDetector(b *testing.B) {
	config := &detector.GoroutineConfig{
		Interval:         100 * time.Millisecond,
		GrowthThreshold:  5,
		GrowthRate:       0.1,
		Verbose:          false,
		StackTraceDepth:  10,
		SampleStackTrace: false, // Disable for performance
	}

	detector := detector.NewGoroutineDetector(config)
	detector.Start()
	defer detector.Stop()

	// Let it collect some initial metrics
	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysis, err := detector.AnalyzeGoroutines()
		if err != nil || analysis == nil {
			b.Fatal("Analysis should not be nil")
		}
	}
}

// BenchmarkReferenceDetector benchmarks the reference detector performance
func BenchmarkReferenceDetector(b *testing.B) {
	config := &detector.ReferenceConfig{
		Interval:      200 * time.Millisecond,
		MaxDepth:      10,
		Verbose:       false,
		TrackPointers: true,
		SampleRate:    0.5,
	}

	detector := detector.NewReferenceDetector(config)
	detector.Start()
	defer detector.Stop()

	// Create some test objects
	type TestObj struct {
		ID   int
		Data []byte
		Ref  *TestObj
	}

	for i := 0; i < 10; i++ {
		obj := &TestObj{ID: i, Data: make([]byte, 1024)}
		detector.TrackObject(obj, fmt.Sprintf("obj_%d", i))
	}

	// Let it collect some initial metrics
	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysis, err := detector.AnalyzeReferences()
		if err != nil || analysis == nil {
			b.Fatal("Analysis should not be nil")
		}
	}
}

// BenchmarkResourceDetector benchmarks the resource detector performance
func BenchmarkResourceDetector(b *testing.B) {
	config := &detector.ResourceConfig{
		Interval:            100 * time.Millisecond,
		FileHandleThreshold: 100,
		NetworkThreshold:    50,
		GrowthThreshold:     0.1,
		Verbose:             false,
		TrackStacks:         false,
	}

	detector := detector.NewResourceDetector(config)
	detector.Start()
	defer detector.Stop()

	// Let it collect some initial metrics
	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysis, err := detector.AnalyzeResources()
		if err != nil || analysis == nil {
			b.Fatal("Analysis should not be nil")
		}
	}
}

// BenchmarkAnalyzerOverhead benchmarks the overall analyzer overhead
func BenchmarkAnalyzerOverhead(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark_analyzer")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          10 * time.Second,
		SamplingInterval:            100 * time.Millisecond,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false, // Skip for performance
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              false,
	}

	analyzer := analyzer.NewAnalyzer(config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analyzer.Start()
		time.Sleep(1 * time.Second) // Short run
		analyzer.Stop()
	}
}

// BenchmarkMetricsCollection benchmarks metrics collection performance
func BenchmarkMetricsCollection(b *testing.B) {
	config := &detector.HeapConfig{
		Interval:  50 * time.Millisecond,
		Threshold: 0.1,
		Verbose:   false,
	}

	analyzer := detector.NewHeapAnalyzer(config)
	analyzer.Start()
	defer analyzer.Stop()

	// Let it collect many metrics
	time.Sleep(2 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics := analyzer.GetMetrics()
		if len(metrics) == 0 {
			b.Fatal("Should have collected metrics")
		}
	}
}

// BenchmarkLeakSimulation benchmarks leak simulation performance
func BenchmarkLeakSimulation(b *testing.B) {
	configs := []*simulator.LeakConfig{
		{
			Type:       simulator.LeakTypeHeap,
			Intensity:  3,
			Duration:   1 * time.Second,
			GrowthRate: 1.0,
			Verbose:    false,
		},
		{
			Type:       simulator.LeakTypeGoroutine,
			Intensity:  3,
			Duration:   1 * time.Second,
			GrowthRate: 1.0,
			Verbose:    false,
		},
		{
			Type:       simulator.LeakTypeFile,
			Intensity:  3,
			Duration:   1 * time.Second,
			GrowthRate: 1.0,
			Verbose:    false,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		config := configs[i%len(configs)]
		sim := simulator.NewLeakSimulator(config)
		sim.Start()
		time.Sleep(500 * time.Millisecond)
		sim.Stop()
	}
}

// BenchmarkConcurrentDetection benchmarks concurrent detection performance
func BenchmarkConcurrentDetection(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark_concurrent")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          5 * time.Second,
		SamplingInterval:            100 * time.Millisecond,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false,
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			analyzer := analyzer.NewAnalyzer(config)
			analyzer.Start()
			time.Sleep(500 * time.Millisecond)
			status := analyzer.GetCurrentStatus()
			if !status["running"].(bool) {
				b.Error("Analyzer should be running")
			}
			analyzer.Stop()
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage of the detection system
func BenchmarkMemoryUsage(b *testing.B) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	tempDir, err := os.MkdirTemp("", "benchmark_memory")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          2 * time.Second,
		SamplingInterval:            50 * time.Millisecond,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     true,
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analyzer := analyzer.NewAnalyzer(config)
		result, err := analyzer.RunAnalysis()
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
		if result == nil {
			b.Fatal("Result should not be nil")
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
}

// BenchmarkLargeDataset benchmarks performance with large datasets
func BenchmarkLargeDataset(b *testing.B) {
	config := &detector.HeapConfig{
		Interval:  10 * time.Millisecond, // High frequency
		Threshold: 0.01,                  // Sensitive threshold
		Verbose:   false,
	}

	analyzer := detector.NewHeapAnalyzer(config)
	analyzer.Start()
	defer analyzer.Stop()

	// Generate significant memory activity
	go func() {
		data := make([][]byte, 0)
		for i := 0; i < 1000; i++ {
			chunk := make([]byte, 1024)
			data = append(data, chunk)
			if i%100 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	// Let it collect many metrics
	time.Sleep(5 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysis, err := analyzer.AnalyzeHeap()
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
		if analysis == nil {
			b.Fatal("Analysis should not be nil")
		}
		if len(analysis.Recommendations) == 0 {
			b.Error("Should have recommendations")
		}
	}
}

// BenchmarkReportGeneration benchmarks report generation performance
func BenchmarkReportGeneration(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark_report")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          3 * time.Second,
		SamplingInterval:            100 * time.Millisecond,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false,
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json", "html", "text"},
		Verbose:                     false,
		RealTimeAlerts:              false,
	}

	analyzer := analyzer.NewAnalyzer(config)
	result, err := analyzer.RunAnalysis()
	if err != nil {
		b.Fatalf("Analysis failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := analyzer.GenerateReport(result)
		if err != nil {
			b.Fatalf("Report generation failed: %v", err)
		}
	}
}

// BenchmarkAlertProcessing benchmarks alert processing performance
func BenchmarkAlertProcessing(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark_alerts")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &analyzer.AnalyzerConfig{
		MonitoringDuration:          2 * time.Second,
		SamplingInterval:            50 * time.Millisecond,
		OutputDir:                   tempDir,
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     false,
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json"},
		Verbose:                     false,
		RealTimeAlerts:              true,
		AlertThresholds: analyzer.AlertThresholds{
			MemoryGrowthRate:     0.01, // Very sensitive
			GoroutineGrowthRate:  0.01,
			FileHandleGrowthRate: 0.01,
			NetworkGrowthRate:    0.01,
			CriticalLeakCount:    1,
		},
	}

	// Create simulation to trigger alerts
	simConfig := &simulator.LeakConfig{
			Type:       simulator.LeakTypeHeap,
			Intensity:  5,
			Duration:   5 * time.Second,
			GrowthRate: 5.0,
			Verbose:    false,
		}

	sim := simulator.NewLeakSimulator(simConfig)
	sim.Start()
	defer sim.Stop()

	analyzer := analyzer.NewAnalyzer(config)
	analyzer.Start()
	defer analyzer.Stop()

	// Let it generate some alerts
	time.Sleep(3 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		alerts := analyzer.GetAlerts()
		if len(alerts) == 0 {
			b.Error("Should have generated alerts")
		}
	}
}

// BenchmarkStackTraceCollection benchmarks stack trace collection performance
func BenchmarkStackTraceCollection(b *testing.B) {
	config := &detector.GoroutineConfig{
		Interval:         100 * time.Millisecond,
		GrowthThreshold:  5,
		GrowthRate:       0.1,
		Verbose:          false,
		StackTraceDepth:  20, // Deep stack traces
		SampleStackTrace: true,
	}

	detector := detector.NewGoroutineDetector(config)
	detector.Start()
	defer detector.Stop()

	// Create goroutines with deep call stacks
	for i := 0; i < 10; i++ {
		go createDeepCallStack(10)
	}

	// Let it collect metrics with stack traces
	time.Sleep(2 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics := detector.GetMetrics()
		if len(metrics) == 0 {
			b.Fatal("Should have collected metrics")
		}
		// Verify stack traces are collected
		for _, metric := range metrics {
			if len(metric.StackTraces) == 0 {
				b.Error("Should have collected stack traces")
			}
		}
	}
}

// BenchmarkReferenceTracking benchmarks reference tracking performance
func BenchmarkReferenceTracking(b *testing.B) {
	config := &detector.ReferenceConfig{
		Interval:      100 * time.Millisecond,
		MaxDepth:      10,
		Verbose:       false,
		TrackPointers: true,
		SampleRate:    0.5,
	}

	detector := detector.NewReferenceDetector(config)
	detector.Start()
	defer detector.Stop()

	type TestNode struct {
		ID       int
		Data     []byte
		Children []*TestNode
		Parent   *TestNode
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create complex object graph
		nodes := make([]*TestNode, 100)
		for j := 0; j < 100; j++ {
			node := &TestNode{
				ID:   j,
				Data: make([]byte, 512),
			}
			nodes[j] = node
			detector.TrackObject(node, fmt.Sprintf("node_%d", j))
		}

		// Create references
		for j := 0; j < 99; j++ {
			nodes[j].Children = append(nodes[j].Children, nodes[j+1])
			nodes[j+1].Parent = nodes[j]
		}

		// Create some cycles
		if len(nodes) > 10 {
			nodes[99].Children = append(nodes[99].Children, nodes[0])
		}
	}
}

// Helper function for deep call stack benchmark
func createDeepCallStack(depth int) {
	if depth <= 0 {
		time.Sleep(2 * time.Second)
		return
	}
	createDeepCallStack(depth - 1)
}

// BenchmarkSystemResourceUsage benchmarks system resource usage monitoring
func BenchmarkSystemResourceUsage(b *testing.B) {
	config := &detector.ResourceConfig{
		Interval:            50 * time.Millisecond, // High frequency
		FileHandleThreshold: 100,
		NetworkThreshold:    50,
		GrowthThreshold:     0.05,
		Verbose:             false,
		TrackStacks:         false,
	}

	detector := detector.NewResourceDetector(config)
	detector.Start()
	defer detector.Stop()

	// Create some resource usage
	files := make([]*os.File, 0)
	for i := 0; i < 5; i++ {
		if tempFile, err := os.CreateTemp("", "benchmark_resource_"); err == nil {
			files = append(files, tempFile)
		}
	}
	defer func() {
		for _, file := range files {
			file.Close()
			os.Remove(file.Name())
		}
	}()

	// Let it collect metrics
	time.Sleep(2 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysis, err := detector.AnalyzeResources()
		if err != nil {
			b.Fatalf("Analysis failed: %v", err)
		}
		if analysis == nil {
			b.Fatal("Analysis should not be nil")
		}
	}
}