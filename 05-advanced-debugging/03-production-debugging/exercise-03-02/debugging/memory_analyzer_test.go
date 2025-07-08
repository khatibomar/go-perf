package debugging

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestMemoryAnalyzer_BasicFunctionality(t *testing.T) {
	config := AnalyzerConfig{
		SnapshotInterval:     time.Millisecond * 100,
		MaxSnapshots:        10,
		LeakThreshold:       0.1,
		ConsecutiveChecks:   3,
		EnableGCOptimization: false,
	}

	analyzer := NewMemoryAnalyzer(config)

	// Test initial state
	snapshot := analyzer.GetCurrentSnapshot()
	if !snapshot.Timestamp.IsZero() {
		t.Error("Expected empty initial snapshot")
	}

	// Test snapshot collection
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for some snapshots to be collected
	time.Sleep(time.Millisecond * 350)

	snapshot = analyzer.GetCurrentSnapshot()
	if snapshot.Timestamp.IsZero() {
		t.Error("Expected snapshot to be collected")
	}

	if snapshot.Goroutines <= 0 {
		t.Error("Expected positive goroutine count")
	}

	if snapshot.HeapAlloc == 0 {
		t.Error("Expected non-zero heap allocation")
	}
}

func TestMemoryAnalyzer_LeakDetection(t *testing.T) {
	config := AnalyzerConfig{
		SnapshotInterval:     time.Millisecond * 50,
		MaxSnapshots:        20,
		LeakThreshold:       0.05, // 5% threshold for testing
		ConsecutiveChecks:   3,
		EnableGCOptimization: false,
	}

	analyzer := NewMemoryAnalyzer(config)

	// Register leak callback
	leakDetected := make(chan MemoryLeak, 1)
	analyzer.RegisterLeakCallback(func(leak MemoryLeak) {
		select {
		case leakDetected <- leak:
		default:
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	go analyzer.Start(ctx)

	// Simulate memory leak by allocating memory
	leakyData := make([][]byte, 0)
	for i := 0; i < 50; i++ {
		data := make([]byte, 1024*100) // 100KB each
		leakyData = append(leakyData, data)
		time.Sleep(time.Millisecond * 20)
	}

	// Wait for leak detection
	select {
	case leak := <-leakDetected:
		if leak.Type != "heap_growth" {
			t.Errorf("Expected heap_growth leak, got %s", leak.Type)
		}
		if leak.Severity == "" {
			t.Error("Expected non-empty severity")
		}
		t.Logf("Leak detected: %s - %s", leak.Type, leak.Description)
	case <-time.After(time.Second * 2):
		t.Error("Expected leak to be detected within 2 seconds")
	}

	// Keep reference to prevent GC
	_ = leakyData
}

func TestMemoryAnalyzer_GoroutineLeakDetection(t *testing.T) {
	config := AnalyzerConfig{
		SnapshotInterval:     time.Millisecond * 100,
		MaxSnapshots:        20,
		LeakThreshold:       0.3, // 30% increase
		ConsecutiveChecks:   2,
		EnableGCOptimization: false,
	}

	analyzer := NewMemoryAnalyzer(config)
	analyzer.alertThresholds.GoroutineGrowth = 10 // Lower threshold for testing

	// Register leak callback
	leakDetected := make(chan MemoryLeak, 1)
	analyzer.RegisterLeakCallback(func(leak MemoryLeak) {
		if leak.Type == "goroutine_leak" {
			select {
			case leakDetected <- leak:
			default:
			}
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for initial snapshots to establish baseline
	time.Sleep(time.Millisecond * 300)

	// Create significant goroutine leak
	stopChans := make([]chan struct{}, 0)
	for i := 0; i < 50; i++ {
		stopChan := make(chan struct{})
		stopChans = append(stopChans, stopChan)
		go func(stop chan struct{}) {
			ticker := time.NewTicker(time.Millisecond * 10)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					// Keep goroutine busy
				}
			}
		}(stopChan)
		time.Sleep(time.Millisecond * 5)
	}

	// Force runtime to update goroutine count
	runtime.Gosched()
	time.Sleep(time.Millisecond * 100)

	// Wait for leak detection with more time
	select {
	case leak := <-leakDetected:
		if leak.Type != "goroutine_leak" {
			t.Errorf("Expected goroutine_leak, got %s", leak.Type)
		}
		t.Logf("Goroutine leak detected: %s", leak.Description)
	case <-time.After(time.Second * 3):
		// Get current snapshots for debugging
		snapshots := analyzer.GetSnapshots(time.Time{})
		if len(snapshots) >= 2 {
			first := snapshots[0]
			last := snapshots[len(snapshots)-1]
			t.Logf("Goroutines: %d -> %d (increase: %.2f%%)", 
				first.Goroutines, last.Goroutines, 
				float64(last.Goroutines-first.Goroutines)/float64(first.Goroutines)*100)
		}
		t.Error("Expected goroutine leak to be detected within 3 seconds")
	}

	// Cleanup
	for _, stopChan := range stopChans {
		close(stopChan)
	}
}

func TestMemoryAnalyzer_ForceGC(t *testing.T) {
	config := AnalyzerConfig{
		SnapshotInterval:     time.Second,
		MaxSnapshots:        10,
		LeakThreshold:       0.1,
		ConsecutiveChecks:   3,
		EnableGCOptimization: false,
	}

	analyzer := NewMemoryAnalyzer(config)

	// Allocate some memory
	data := make([][]byte, 100)
	for i := range data {
		data[i] = make([]byte, 1024*10) // 10KB each
	}

	// Clear references to make memory eligible for GC
	data = nil

	before, after, freed := analyzer.ForceGCAndAnalyze()

	if before.Timestamp.IsZero() || after.Timestamp.IsZero() {
		t.Error("Expected valid timestamps in snapshots")
	}

	if after.Timestamp.Before(before.Timestamp) {
		t.Error("Expected after timestamp to be later than before")
	}

	// Note: freed might be 0 if GC didn't collect anything, which is normal
	t.Logf("GC Analysis: Before=%d, After=%d, Freed=%d bytes", before.HeapAlloc, after.HeapAlloc, freed)
}

func TestMemoryAnalyzer_SnapshotRetrieval(t *testing.T) {
	config := AnalyzerConfig{
		SnapshotInterval:     time.Millisecond * 50,
		MaxSnapshots:        5,
		LeakThreshold:       0.1,
		ConsecutiveChecks:   3,
		EnableGCOptimization: false,
	}

	analyzer := NewMemoryAnalyzer(config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
	defer cancel()

	go analyzer.Start(ctx)

	// Wait for snapshots to be collected
	time.Sleep(time.Millisecond * 250)

	// Test snapshot retrieval
	since := time.Now().Add(-time.Millisecond * 200)
	snapshots := analyzer.GetSnapshots(since)

	if len(snapshots) == 0 {
		t.Error("Expected at least one snapshot")
	}

	// Verify snapshots are in chronological order
	for i := 1; i < len(snapshots); i++ {
		if snapshots[i].Timestamp.Before(snapshots[i-1].Timestamp) {
			t.Error("Expected snapshots to be in chronological order")
		}
	}

	// Test max snapshots limit
	time.Sleep(time.Millisecond * 200) // Collect more snapshots
	allSnapshots := analyzer.GetSnapshots(time.Time{})
	if len(allSnapshots) > config.MaxSnapshots {
		t.Errorf("Expected at most %d snapshots, got %d", config.MaxSnapshots, len(allSnapshots))
	}
}

func TestMemoryAnalyzer_PerformanceRequirements(t *testing.T) {
	// Test that memory leak detection meets performance requirements:
	// - Memory leak detection within 10 minutes
	// - Heap analysis completion <30 seconds
	// - Zero production impact during analysis

	config := AnalyzerConfig{
		SnapshotInterval:     time.Millisecond * 100,
		MaxSnapshots:        50,
		LeakThreshold:       0.1,
		ConsecutiveChecks:   5,
		EnableGCOptimization: true,
	}

	analyzer := NewMemoryAnalyzer(config)

	// Test heap analysis performance
	start := time.Now()
	before, after, freed := analyzer.ForceGCAndAnalyze()
	analysisTime := time.Since(start)

	if analysisTime > time.Second*30 {
		t.Errorf("Heap analysis took %v, expected <30 seconds", analysisTime)
	}

	// Verify analysis produced valid results
	if before.Timestamp.IsZero() || after.Timestamp.IsZero() {
		t.Error("Expected valid analysis results")
	}

	t.Logf("Heap analysis completed in %v (freed %d bytes)", analysisTime, freed)

	// Test snapshot collection performance
	start = time.Now()
	analyzer.takeSnapshot()
	snapshotTime := time.Since(start)

	if snapshotTime > time.Millisecond*100 {
		t.Errorf("Snapshot collection took %v, expected <100ms", snapshotTime)
	}

	t.Logf("Snapshot collection completed in %v", snapshotTime)
}

func BenchmarkMemoryAnalyzer_TakeSnapshot(b *testing.B) {
	config := AnalyzerConfig{
		SnapshotInterval:     time.Second,
		MaxSnapshots:        100,
		LeakThreshold:       0.1,
		ConsecutiveChecks:   3,
		EnableGCOptimization: false,
	}

	analyzer := NewMemoryAnalyzer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.takeSnapshot()
	}
}

func BenchmarkMemoryAnalyzer_AnalyzeLeaks(b *testing.B) {
	config := AnalyzerConfig{
		SnapshotInterval:     time.Millisecond * 10,
		MaxSnapshots:        100,
		LeakThreshold:       0.1,
		ConsecutiveChecks:   10,
		EnableGCOptimization: false,
	}

	analyzer := NewMemoryAnalyzer(config)

	// Pre-populate with snapshots
	for i := 0; i < 50; i++ {
		analyzer.takeSnapshot()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.analyzeLeaks()
	}
}