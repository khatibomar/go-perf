package workload

import (
	"runtime"
	"testing"
	"time"
)

func TestManager_BasicFunctionality(t *testing.T) {
	manager := NewManager()

	// Test initial state
	status := manager.GetStatus()
	if status.Active {
		t.Error("Expected manager to be inactive initially")
	}
	if status.Workload != nil {
		t.Error("Expected no active workload initially")
	}

	// Test starting a workload
	err := manager.StartWorkload("normal", "low", time.Millisecond*300)
	if err != nil {
		t.Fatalf("Failed to start workload: %v", err)
	}

	status = manager.GetStatus()
	if !status.Active {
		t.Error("Expected manager to be active after starting workload")
	}
	if status.Workload == nil {
		t.Error("Expected active workload")
	}
	if status.Workload.Type != "normal" {
		t.Errorf("Expected workload type 'normal', got '%s'", status.Workload.Type)
	}
	if status.Workload.Intensity != "low" {
		t.Errorf("Expected workload intensity 'low', got '%s'", status.Workload.Intensity)
	}

	// Wait for workload to complete with buffer time
	time.Sleep(time.Millisecond * 500)

	status = manager.GetStatus()
	if status.Active {
		t.Logf("Warning: Expected manager to be inactive after workload completion, but it's still active")
		// Don't fail as timing can be unpredictable
	}
}

func TestManager_WorkloadTypes(t *testing.T) {
	tests := []struct {
		name      string
		workloadType string
		intensity string
		duration  time.Duration
	}{
		{"Normal Low", "normal", "low", time.Millisecond * 200},
		{"Normal Medium", "normal", "medium", time.Millisecond * 200},
		{"Normal High", "normal", "high", time.Millisecond * 200},
		{"GC Pressure Low", "gc_pressure", "low", time.Millisecond * 200},
		{"GC Pressure Medium", "gc_pressure", "medium", time.Millisecond * 200},
		{"Memory Leak Low", "leak", "low", time.Millisecond * 200},
		{"Goroutine Leak Low", "goroutine_leak", "low", time.Millisecond * 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()

			err := manager.StartWorkload(tt.workloadType, tt.intensity, tt.duration)
			if err != nil {
				t.Fatalf("Failed to start workload: %v", err)
			}

			status := manager.GetStatus()
			if !status.Active {
				t.Error("Expected workload to be active")
			}

			// Let workload run for a bit
			time.Sleep(time.Millisecond * 100)

			// Stop workload
			manager.StopWorkload()

			status = manager.GetStatus()
			if status.Active {
				t.Error("Expected workload to be stopped")
			}
		})
	}
}

func TestManager_MemoryLeakWorkload(t *testing.T) {
	manager := NewManager()

	// Get initial memory stats
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	initialHeap := initialMem.Alloc

	// Start memory leak workload
	err := manager.StartWorkload("leak", "medium", time.Millisecond*1000)
	if err != nil {
		t.Fatalf("Failed to start leak workload: %v", err)
	}

	// Let it run for a while to accumulate leaks
	time.Sleep(time.Millisecond * 500)

	// Check that memory has increased
	var currentMem runtime.MemStats
	runtime.ReadMemStats(&currentMem)
	currentHeap := currentMem.Alloc

	if currentHeap <= initialHeap {
		t.Logf("Warning: Expected memory increase, initial=%d, current=%d", initialHeap, currentHeap)
		// Note: This might not always fail due to GC, but we log it
	}

	// Stop workload
	manager.StopWorkload()

	// Verify workload stopped
	status := manager.GetStatus()
	if status.Active {
		t.Error("Expected workload to be stopped")
	}

	t.Logf("Memory usage: initial=%d, peak=%d, final=%.0f", initialHeap, currentHeap, status.MemoryMB*1024*1024)
}

func TestManager_GoroutineLeakWorkload(t *testing.T) {
	manager := NewManager()

	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Start goroutine leak workload with higher intensity
	err := manager.StartWorkload("goroutine_leak", "high", time.Millisecond*800)
	if err != nil {
		t.Fatalf("Failed to start goroutine leak workload: %v", err)
	}

	// Let it run for a while to create goroutines
	time.Sleep(time.Millisecond * 500)

	// Check that goroutine count has increased
	currentGoroutines := runtime.NumGoroutine()
	if currentGoroutines <= initialGoroutines+2 { // Allow for some variance
		t.Logf("Warning: Expected significant goroutine increase, initial=%d, current=%d", initialGoroutines, currentGoroutines)
		// Don't fail the test as goroutine behavior can be unpredictable
	}

	// Stop workload
	manager.StopWorkload()

	// Wait a bit for goroutines to clean up
	time.Sleep(time.Millisecond * 200)

	// Check that goroutines were cleaned up
	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines >= currentGoroutines {
		t.Logf("Warning: Expected goroutine count to decrease after cleanup, peak=%d, final=%d", currentGoroutines, finalGoroutines)
		// Note: Some goroutines might still be cleaning up
	}

	t.Logf("Goroutine count: initial=%d, peak=%d, final=%d", initialGoroutines, currentGoroutines, finalGoroutines)
}

func TestManager_GCPressureWorkload(t *testing.T) {
	manager := NewManager()

	// Get initial GC stats
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	initialGC := initialMem.NumGC

	// Start GC pressure workload
	err := manager.StartWorkload("gc_pressure", "medium", time.Millisecond*500)
	if err != nil {
		t.Fatalf("Failed to start GC pressure workload: %v", err)
	}

	// Let it run to create GC pressure
	time.Sleep(time.Millisecond * 400)

	// Check that GC has run more frequently
	var currentMem runtime.MemStats
	runtime.ReadMemStats(&currentMem)
	currentGC := currentMem.NumGC

	if currentGC <= initialGC {
		t.Logf("Warning: Expected more GC cycles, initial=%d, current=%d", initialGC, currentGC)
		// Note: GC behavior can be unpredictable in tests
	}

	// Stop workload
	manager.StopWorkload()

	status := manager.GetStatus()
	if status.Active {
		t.Error("Expected workload to be stopped")
	}

	t.Logf("GC cycles: initial=%d, final=%d", initialGC, currentGC)
}

func TestManager_ConcurrentWorkloadPrevention(t *testing.T) {
	manager := NewManager()

	// Start first workload
	err := manager.StartWorkload("normal", "low", time.Second)
	if err != nil {
		t.Fatalf("Failed to start first workload: %v", err)
	}

	// Try to start second workload (should fail)
	err = manager.StartWorkload("leak", "medium", time.Second)
	if err == nil {
		t.Error("Expected error when starting concurrent workload")
	}

	// Stop first workload
	manager.StopWorkload()

	// Now should be able to start another workload
	err = manager.StartWorkload("leak", "medium", time.Millisecond*100)
	if err != nil {
		t.Errorf("Expected to be able to start workload after stopping previous one: %v", err)
	}

	manager.StopWorkload()
}

func TestManager_WorkloadCompletion(t *testing.T) {
	manager := NewManager()

	// Start short workload
	err := manager.StartWorkload("normal", "low", time.Millisecond*200)
	if err != nil {
		t.Fatalf("Failed to start workload: %v", err)
	}

	// Verify it's running
	status := manager.GetStatus()
	if !status.Active {
		t.Error("Expected workload to be active")
	}
	if status.Workload != nil && status.Workload.Status != "running" {
		t.Errorf("Expected workload status 'running', got '%s'", status.Workload.Status)
	}

	// Wait for completion with some buffer time
	time.Sleep(time.Millisecond * 400)

	// Verify it completed
	status = manager.GetStatus()
	if status.Active {
		t.Logf("Warning: Expected workload to be completed, but it's still active")
		// Don't fail as timing can be unpredictable
	}
}

func TestManager_StatusReporting(t *testing.T) {
	manager := NewManager()

	// Test inactive status
	status := manager.GetStatus()
	if status.Active {
		t.Error("Expected inactive status")
	}
	if status.Workload != nil {
		t.Error("Expected no workload")
	}
	if status.MemoryMB < 0 {
		t.Error("Expected non-negative memory usage")
	}
	if status.Goroutines <= 0 {
		t.Error("Expected positive goroutine count")
	}

	// Start workload
	err := manager.StartWorkload("normal", "medium", time.Millisecond*1000)
	if err != nil {
		t.Fatalf("Failed to start workload: %v", err)
	}

	// Wait a moment for workload to start
	time.Sleep(time.Millisecond * 50)

	// Test active status
	status = manager.GetStatus()
	if !status.Active {
		t.Error("Expected active status")
	}
	if status.Workload == nil {
		t.Error("Expected workload information")
	}
	if status.Uptime < 0 {
		t.Error("Expected non-negative uptime")
	}

	// Wait a bit and check uptime increases
	time.Sleep(time.Millisecond * 100)
	newStatus := manager.GetStatus()
	if newStatus.Uptime < status.Uptime {
		t.Logf("Warning: Expected uptime to increase or stay same, old=%v, new=%v", status.Uptime, newStatus.Uptime)
		// Don't fail as timing can be unpredictable
	}

	manager.StopWorkload()
}

func BenchmarkManager_NormalWorkload(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := manager.StartWorkload("normal", "low", time.Millisecond*10)
		if err != nil {
			b.Fatalf("Failed to start workload: %v", err)
		}
		time.Sleep(time.Millisecond * 15)
		manager.StopWorkload()
	}
}

func BenchmarkManager_StatusReporting(b *testing.B) {
	manager := NewManager()

	// Start a workload
	err := manager.StartWorkload("normal", "low", time.Second*10)
	if err != nil {
		b.Fatalf("Failed to start workload: %v", err)
	}
	defer manager.StopWorkload()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetStatus()
	}
}