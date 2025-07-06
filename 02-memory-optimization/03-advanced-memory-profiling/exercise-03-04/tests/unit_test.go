package tests

import (
	"os"
	"runtime"
	"testing"
	"time"

	"exercise-03-04/detector"
	"exercise-03-04/simulator"
)

// TestHeapAnalyzer tests the heap analyzer functionality
func TestHeapAnalyzer(t *testing.T) {
	config := &detector.HeapConfig{
		Interval:  1 * time.Second,
		Threshold: 0.1,
		Verbose:   false,
	}

	analyzer := detector.NewHeapAnalyzer(config)
	if analyzer == nil {
		t.Fatal("Heap analyzer should not be nil")
	}

	// Test start
	err := analyzer.Start()
	if err != nil {
		t.Fatalf("Failed to start heap analyzer: %v", err)
	}

	// Let it collect some metrics
	time.Sleep(3 * time.Second)

	// Test metrics collection
	metrics := analyzer.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Should have collected some metrics")
	}

	// Validate metric structure
	for _, metric := range metrics {
		if metric.Timestamp.IsZero() {
			t.Error("Metric timestamp should not be zero")
		}
		if metric.HeapAlloc < 0 {
			t.Error("Heap alloc should not be negative")
		}
		if metric.HeapObjects < 0 {
			t.Error("Number of objects should not be negative")
		}
		if metric.NumGC < 0 {
			t.Error("GC count should not be negative")
		}
	}

	// Test analysis
	analysis, err := analyzer.AnalyzeHeap()
	if err != nil {
		t.Fatalf("Failed to analyze heap: %v", err)
	}
	if analysis == nil {
		t.Error("Analysis should not be nil")
	}

	// Validate analysis structure
	if analysis.GrowthRate < 0 {
		t.Error("Growth rate should not be negative")
	}
	if analysis.LeakProbability < 0 || analysis.LeakProbability > 1 {
		t.Errorf("Leak probability should be between 0 and 1, got %.2f", analysis.LeakProbability)
	}
	if len(analysis.Recommendations) == 0 {
		t.Error("Should have some recommendations")
	}

	// Test stop
	analyzer.Stop()

	// Test saving results
	tempDir := createTempDir(t, "heap_test")
	defer os.RemoveAll(tempDir)

	err = analyzer.SaveResults(tempDir)
	if err != nil {
		t.Errorf("Failed to save results: %v", err)
	}
}

// TestGoroutineDetector tests the goroutine detector functionality
func TestGoroutineDetector(t *testing.T) {
	config := &detector.GoroutineConfig{
		Interval:         1 * time.Second,
		GrowthThreshold:  2,
		GrowthRate:       0.1,
		Verbose:          false,
		StackTraceDepth:  5,
		SampleStackTrace: true,
	}

	detector := detector.NewGoroutineDetector(config)
	if detector == nil {
		t.Fatal("Goroutine detector should not be nil")
	}

	// Test start
	err := detector.Start()
	if err != nil {
		t.Fatalf("Failed to start goroutine detector: %v", err)
	}

	// Create some goroutines to detect
	for i := 0; i < 5; i++ {
		go func(id int) {
			time.Sleep(5 * time.Second)
		}(i)
	}

	// Let it collect some metrics
	time.Sleep(3 * time.Second)

	// Test metrics collection
	metrics := detector.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Should have collected some metrics")
	}

	// Validate metric structure
	for _, metric := range metrics {
		if metric.Timestamp.IsZero() {
			t.Error("Metric timestamp should not be zero")
		}
		if metric.Count < 0 {
			t.Error("Goroutine count should not be negative")
		}
		if metric.Count < runtime.NumGoroutine() {
			t.Error("Detected count should be at least current goroutine count")
		}
	}

	// Test analysis
	analysis, err := detector.AnalyzeGoroutines()
	if err != nil {
		t.Fatalf("Failed to analyze goroutines: %v", err)
	}
	if analysis == nil {
		t.Error("Analysis should not be nil")
	}

	// Validate analysis structure
	if analysis.GrowthRate < 0 {
		t.Error("Growth rate should not be negative")
	}
	if analysis.LeakProbability < 0 || analysis.LeakProbability > 1 {
		t.Errorf("Leak probability should be between 0 and 1, got %.2f", analysis.LeakProbability)
	}
	if len(analysis.Recommendations) == 0 {
		t.Error("Should have some recommendations")
	}

	// Test stop
	detector.Stop()

	// Test saving results
	tempDir := createTempDir(t, "goroutine_test")
	defer os.RemoveAll(tempDir)

	err = detector.SaveResults(tempDir)
	if err != nil {
		t.Errorf("Failed to save results: %v", err)
	}
}

// TestReferenceDetector tests the reference detector functionality
func TestReferenceDetector(t *testing.T) {
	config := &detector.ReferenceConfig{
		Interval:      2 * time.Second,
		MaxDepth:      10,
		Verbose:       false,
		TrackPointers: true,
		SampleRate:    0.5,
	}

	detector := detector.NewReferenceDetector(config)
	if detector == nil {
		t.Fatal("Reference detector should not be nil")
	}

	// Test start
	err := detector.Start()
	if err != nil {
		t.Fatalf("Failed to start reference detector: %v", err)
	}

	// Create some objects to track
	type TestStruct struct {
		ID   int
		Data []byte
		Ref  *TestStruct
	}

	// Create potential reference cycles
	obj1 := &TestStruct{ID: 1, Data: make([]byte, 1024)}
	obj2 := &TestStruct{ID: 2, Data: make([]byte, 1024)}
	obj1.Ref = obj2
	obj2.Ref = obj1 // Create cycle

	// Track objects
	detector.TrackObject(obj1, "obj1")
	detector.TrackObject(obj2, "obj2")

	// Let it collect some metrics
	time.Sleep(5 * time.Second)

	// Test metrics collection
	metrics := detector.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Should have collected some metrics")
	}

	// Test cycle detection
	cycles := detector.GetCycles()
	if len(cycles) == 0 {
		t.Error("Should have detected reference cycles")
	}

	// Validate cycle structure
	for _, cycle := range cycles {
		if cycle.ID == "" {
			t.Error("Cycle ID should not be empty")
		}
		if cycle.TotalSize <= 0 {
			t.Error("Cycle total size should be positive")
		}
		if len(cycle.Objects) == 0 {
			t.Error("Cycle should have objects")
		}
		if cycle.Severity == "" {
			t.Error("Cycle severity should not be empty")
		}
	}

	// Test analysis
	analysis, err := detector.AnalyzeReferences()
	if err != nil {
		t.Fatalf("Failed to analyze references: %v", err)
	}
	if analysis == nil {
		t.Error("Analysis should not be nil")
	}

	// Validate analysis structure
	if len(analysis.DetectedCycles) < 0 {
		t.Error("Detected cycles count should not be negative")
	}
	if analysis.Statistics.GrowthRate < 0 {
		t.Error("Growth rate should not be negative")
	}
	if len(analysis.Recommendations) == 0 {
		t.Error("Should have some recommendations")
	}

	// Test stop
	detector.Stop()

	// Test saving results
	tempDir := createTempDir(t, "reference_test")
	defer os.RemoveAll(tempDir)

	err = detector.SaveResults(tempDir)
	if err != nil {
		t.Errorf("Failed to save results: %v", err)
	}
}

// TestResourceDetector tests the resource detector functionality
func TestResourceDetector(t *testing.T) {
	config := &detector.ResourceConfig{
		Interval:            1 * time.Second,
		FileHandleThreshold: 100,
		NetworkThreshold:    50,
		GrowthThreshold:     0.1,
		Verbose:             false,
		TrackStacks:         false,
	}

	detector := detector.NewResourceDetector(config)
	if detector == nil {
		t.Fatal("Resource detector should not be nil")
	}

	// Test start
	err := detector.Start()
	if err != nil {
		t.Fatalf("Failed to start resource detector: %v", err)
	}

	// Create some resource usage
	files := make([]*os.File, 0)
	for i := 0; i < 3; i++ {
		tempFile, err := os.CreateTemp("", "test_resource_")
		if err == nil {
			files = append(files, tempFile)
		}
	}

	// Let it collect some metrics
	time.Sleep(3 * time.Second)

	// Test metrics collection
	metrics := detector.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Should have collected some metrics")
	}

	// Validate metric structure
	for _, metric := range metrics {
		if metric.Timestamp.IsZero() {
			t.Error("Metric timestamp should not be zero")
		}
		if metric.FileHandles < 0 {
			t.Error("File handles should not be negative")
		}
		if metric.NetworkConnections < 0 {
			t.Error("Network connections should not be negative")
		}
		if metric.MemoryMapped < 0 {
			t.Error("Memory mapped should not be negative")
		}
		if metric.ThreadCount < 0 {
			t.Error("Thread count should not be negative")
		}
	}

	// Test analysis
	analysis, err := detector.AnalyzeResources()
	if err != nil {
		t.Fatalf("Failed to analyze resources: %v", err)
	}
	if analysis == nil {
		t.Error("Analysis should not be nil")
	}

	// Validate analysis structure
	if len(analysis.Recommendations) == 0 {
		t.Error("Should have some recommendations")
	}

	// Clean up resources
	for _, file := range files {
		file.Close()
		os.Remove(file.Name())
	}

	// Test stop
	detector.Stop()

	// Test saving results
	tempDir := createTempDir(t, "resource_test")
	defer os.RemoveAll(tempDir)

	err = detector.SaveResults(tempDir)
	if err != nil {
		t.Errorf("Failed to save results: %v", err)
	}
}

// TestLeakSimulator tests the leak simulator functionality
func TestLeakSimulator(t *testing.T) {
	// Test heap leak simulation
	heapConfig := &simulator.LeakConfig{
		Type:          simulator.LeakTypeHeap,
		Intensity:     1,
		Duration:      10 * time.Second,
		GrowthRate:    1.5,
		BurstInterval: 5 * time.Second,
		BurstSize:     3,
		Verbose:       false,
	}

	heapSim := simulator.NewLeakSimulator(heapConfig)
	if heapSim == nil {
		t.Fatal("Heap simulator should not be nil")
	}

	// Test start
	err := heapSim.Start()
	if err != nil {
		t.Fatalf("Failed to start heap simulator: %v", err)
	}

	// Let it run for a bit
	time.Sleep(5 * time.Second)

	// Test stats
	stats := heapSim.GetStats()
	if stats.MemoryAllocated == 0 {
		t.Error("Should have allocated some memory")
	}
	if stats.ResourcesCreated == 0 {
		t.Error("Should have created some resources")
	}

	// Test stop
	heapSim.Stop()

	// Test goroutine leak simulation
	goroutineConfig := &simulator.LeakConfig{
		Type:          simulator.LeakTypeGoroutine,
		Intensity:     1,
		Duration:      10 * time.Second,
		GrowthRate:    1.5,
		BurstInterval: 5 * time.Second,
		BurstSize:     3,
		Verbose:       false,
	}

	goroutineSim := simulator.NewLeakSimulator(goroutineConfig)
	if goroutineSim == nil {
		t.Fatal("Goroutine simulator should not be nil")
	}

	initialGoroutines := runtime.NumGoroutine()

	// Test start
	err = goroutineSim.Start()
	if err != nil {
		t.Fatalf("Failed to start goroutine simulator: %v", err)
	}

	// Let it run for a bit
	time.Sleep(5 * time.Second)

	// Should have created more goroutines
	currentGoroutines := runtime.NumGoroutine()
	if currentGoroutines <= initialGoroutines {
		t.Error("Should have created more goroutines")
	}

	// Test stats
	stats = goroutineSim.GetStats()
	if stats.GoroutinesCreated == 0 {
		t.Error("Should have created some goroutines")
	}

	// Test stop
	goroutineSim.Stop()

	// Test file leak simulation
	fileConfig := &simulator.LeakConfig{
		Type:          simulator.LeakTypeFile,
		Intensity:     1,
		Duration:      10 * time.Second,
		GrowthRate:    1.5,
		BurstInterval: 5 * time.Second,
		BurstSize:     3,
		Verbose:       false,
	}

	fileSim := simulator.NewLeakSimulator(fileConfig)
	if fileSim == nil {
		t.Fatal("File simulator should not be nil")
	}

	// Test start
	err = fileSim.Start()
	if err != nil {
		t.Fatalf("Failed to start file simulator: %v", err)
	}

	// Let it run for a bit
	time.Sleep(5 * time.Second)

	// Test stats
	stats = fileSim.GetStats()
	if stats.FilesOpened == 0 {
		t.Error("Should have opened some files")
	}

	// Test stop
	fileSim.Stop()

	// Test report generation
	tempDir := createTempDir(t, "simulator_test")
	defer os.RemoveAll(tempDir)

	report := heapSim.GenerateReport()
	if report == nil {
		t.Error("Report should not be nil")
	}
}

// TestSimulatorPatterns tests different simulation patterns
func TestSimulatorPatterns(t *testing.T) {
	patterns := []string{
		"gradual",
		"burst",
		"periodic",
		"random",
	}

	for _, pattern := range patterns {
		config := &simulator.LeakConfig{
			Type:          simulator.LeakTypeHeap,
			Intensity:     1,
			Duration:      8 * time.Second,
			GrowthRate:    1.5,
			BurstInterval: 4 * time.Second,
			BurstSize:     2,
			Verbose:       false,
		}

		sim := simulator.NewLeakSimulator(config)
		if sim == nil {
			t.Fatalf("Simulator should not be nil for pattern %v", pattern)
		}

		err := sim.Start()
		if err != nil {
			t.Fatalf("Failed to start simulator for pattern %v: %v", pattern, err)
		}

		time.Sleep(3 * time.Second)

		stats := sim.GetStats()
		if stats.MemoryAllocated == 0 {
			t.Errorf("Should have allocated memory for pattern %v", pattern)
		}

		sim.Stop()
	}
}

// TestSimulatorIntensities tests different simulation intensities
func TestSimulatorIntensities(t *testing.T) {
	intensities := []int{
		1,
		3,
		5,
		7,
	}

	for _, intensity := range intensities {
		config := &simulator.LeakConfig{
			Type:          simulator.LeakTypeHeap,
			Intensity:     intensity,
			Duration:      6 * time.Second,
			GrowthRate:    1.5,
			BurstInterval: 3 * time.Second,
			BurstSize:     2,
			Verbose:       false,
		}

		sim := simulator.NewLeakSimulator(config)
		if sim == nil {
			t.Fatalf("Simulator should not be nil for intensity %v", intensity)
		}

		err := sim.Start()
		if err != nil {
			t.Fatalf("Failed to start simulator for intensity %v: %v", intensity, err)
		}

		time.Sleep(3 * time.Second)

		stats := sim.GetStats()
		if stats.MemoryAllocated == 0 {
			t.Errorf("Should have allocated memory for intensity %v", intensity)
		}

		sim.Stop()
	}
}

// TestDetectorMetricsValidation tests metric validation across detectors
func TestDetectorMetricsValidation(t *testing.T) {
	// Test heap metrics validation
	heapConfig := &detector.HeapConfig{
		Interval:  1 * time.Second,
		Threshold: 0.1,
		Verbose:   false,
	}

	heapAnalyzer := detector.NewHeapAnalyzer(heapConfig)
	heapAnalyzer.Start()
	time.Sleep(3 * time.Second)
	heapMetrics := heapAnalyzer.GetMetrics()
	heapAnalyzer.Stop()

	// Validate heap metrics consistency
	for i := 1; i < len(heapMetrics); i++ {
		prev := heapMetrics[i-1]
		curr := heapMetrics[i]

		if curr.Timestamp.Before(prev.Timestamp) {
			t.Error("Heap metrics timestamps should be in chronological order")
		}

		if curr.NumGC < prev.NumGC {
			t.Error("GC count should not decrease")
		}
	}

	// Test goroutine metrics validation
	goroutineConfig := &detector.GoroutineConfig{
		Interval:         1 * time.Second,
		GrowthThreshold:  2,
		GrowthRate:       0.1,
		Verbose:          false,
		StackTraceDepth:  5,
		SampleStackTrace: false, // Disable for faster test
	}

	goroutineDetector := detector.NewGoroutineDetector(goroutineConfig)
	goroutineDetector.Start()
	time.Sleep(3 * time.Second)
	goroutineMetrics := goroutineDetector.GetMetrics()
	goroutineDetector.Stop()

	// Validate goroutine metrics consistency
	for i := 1; i < len(goroutineMetrics); i++ {
		prev := goroutineMetrics[i-1]
		curr := goroutineMetrics[i]

		if curr.Timestamp.Before(prev.Timestamp) {
			t.Error("Goroutine metrics timestamps should be in chronological order")
		}

		if curr.Count < 1 {
			t.Error("Goroutine count should be at least 1")
		}
	}

	// Test resource metrics validation
	resourceConfig := &detector.ResourceConfig{
		Interval:            1 * time.Second,
		FileHandleThreshold: 100,
		NetworkThreshold:    50,
		GrowthThreshold:     0.1,
		Verbose:             false,
		TrackStacks:         false,
	}

	resourceDetector := detector.NewResourceDetector(resourceConfig)
	resourceDetector.Start()
	time.Sleep(3 * time.Second)
	resourceMetrics := resourceDetector.GetMetrics()
	resourceDetector.Stop()

	// Validate resource metrics consistency
	for i := 1; i < len(resourceMetrics); i++ {
		prev := resourceMetrics[i-1]
		curr := resourceMetrics[i]

		if curr.Timestamp.Before(prev.Timestamp) {
			t.Error("Resource metrics timestamps should be in chronological order")
		}

		if curr.FileHandles < 0 || curr.NetworkConnections < 0 || curr.MemoryMapped < 0 || curr.ThreadCount < 0 {
			t.Error("Resource counts should not be negative")
		}
	}
}

// TestDetectorErrorHandling tests error handling in detectors
func TestDetectorErrorHandling(t *testing.T) {
	// Test invalid configuration
	invalidHeapConfig := &detector.HeapConfig{
		Interval:  0, // Invalid
		Threshold: -1, // Invalid
		Verbose:   false,
	}

	heapAnalyzer := detector.NewHeapAnalyzer(invalidHeapConfig)
	if heapAnalyzer == nil {
		t.Error("Should handle invalid config gracefully")
	}

	// Test double start
	validConfig := &detector.HeapConfig{
		Interval:  1 * time.Second,
		Threshold: 0.1,
		Verbose:   false,
	}

	analyzer := detector.NewHeapAnalyzer(validConfig)
	err := analyzer.Start()
	if err != nil {
		t.Fatalf("First start should succeed: %v", err)
	}

	err = analyzer.Start()
	if err == nil {
		t.Error("Second start should fail")
	}

	analyzer.Stop()

	// Test double stop
	analyzer.Stop() // Should not panic or error

	// Test operations on stopped detector
	metrics := analyzer.GetMetrics()
	if metrics == nil {
		t.Error("Should return empty metrics, not nil")
	}

	analysis, err := analyzer.AnalyzeHeap()
	if err != nil {
		t.Errorf("Analysis should not error: %v", err)
	}
	if analysis == nil {
		t.Error("Should return empty analysis, not nil")
	}
}

// TestDetectorConcurrency tests concurrent access to detectors
func TestDetectorConcurrency(t *testing.T) {
	config := &detector.HeapConfig{
		Interval:  500 * time.Millisecond,
		Threshold: 0.1,
		Verbose:   false,
	}

	analyzer := detector.NewHeapAnalyzer(config)
	err := analyzer.Start()
	if err != nil {
		t.Fatalf("Failed to start analyzer: %v", err)
	}
	defer analyzer.Stop()

	// Concurrent access to metrics
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				metrics := analyzer.GetMetrics()
				if metrics == nil {
					t.Errorf("Goroutine %d: metrics should not be nil", id)
				}
				time.Sleep(100 * time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Concurrent analysis calls
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 3; j++ {
				analysis, err := analyzer.AnalyzeHeap()
				if err != nil {
					t.Errorf("Goroutine %d: analysis should not error: %v", id, err)
				}
				if analysis == nil {
					t.Errorf("Goroutine %d: analysis should not be nil", id)
				}
				time.Sleep(200 * time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 15; i++ {
		<-done
	}
}