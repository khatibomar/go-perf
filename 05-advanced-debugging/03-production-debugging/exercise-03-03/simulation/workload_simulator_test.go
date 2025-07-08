// simulation/workload_simulator_test.go
package simulation

import (
	"runtime"
	"testing"
	"time"
)

func TestWorkloadSimulator_BasicFunctionality(t *testing.T) {
	simulator := NewWorkloadSimulator()

	if simulator == nil {
		t.Fatal("NewWorkloadSimulator returned nil")
	}

	// Check initial state
	status := simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive initially")
	}

	if status.Workload != "" {
		t.Error("Workload type should be empty initially")
	}

	if status.Intensity != "" {
		t.Error("Intensity should be empty initially")
	}

	// Start a CPU workload
	err := simulator.StartWorkload("cpu", time.Millisecond*500, "medium")
	if err != nil {
		t.Fatalf("Failed to start CPU workload: %v", err)
	}

	// Check active state
	status = simulator.GetStatus()
	if !status.Active {
		t.Error("Simulator should be active")
	}

	if status.Workload != "cpu" {
		t.Errorf("Expected workload type 'cpu', got '%s'", status.Workload)
	}

	if status.Intensity != "medium" {
		t.Errorf("Expected intensity 'medium', got '%s'", status.Intensity)
	}

	// Wait for completion
	time.Sleep(time.Millisecond * 600)
	status = simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive after completion")
	}
}

func TestWorkloadSimulator_CPUWorkload(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Start CPU workload
	err := simulator.StartWorkload("cpu", time.Millisecond*500, "medium")
	if err != nil {
		t.Fatalf("Failed to start CPU workload: %v", err)
	}

	// Check status
	status := simulator.GetStatus()
	if !status.Active {
		t.Error("Simulator should be active")
	}

	if status.Workload != "cpu" {
		t.Errorf("Expected workload type 'cpu', got '%s'", status.Workload)
	}

	if status.Intensity != "medium" {
		t.Errorf("Expected intensity 'medium', got '%s'", status.Intensity)
	}

	// Wait for workload to complete
	time.Sleep(time.Millisecond * 600)

	// Check that workload stopped
	status = simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive after workload completion")
	}
}

func TestWorkloadSimulator_MemoryWorkload(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialAlloc := m1.Alloc

	// Start memory workload
	err := simulator.StartWorkload("memory", time.Millisecond*300, "high")
	if err != nil {
		t.Fatalf("Failed to start memory workload: %v", err)
	}

	// Check that memory usage increased
	time.Sleep(time.Millisecond * 100)
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	currentAlloc := m2.Alloc

	if currentAlloc <= initialAlloc {
		t.Error("Memory workload should increase memory usage")
	}

	// Wait for workload to complete
	time.Sleep(time.Millisecond * 300)

	// Check status
	status := simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive after workload completion")
	}
}

func TestWorkloadSimulator_GoroutineWorkload(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Start goroutine workload
	err := simulator.StartWorkload("goroutine", time.Millisecond*400, "medium")
	if err != nil {
		t.Fatalf("Failed to start goroutine workload: %v", err)
	}

	// Check that goroutine count increased
	time.Sleep(time.Millisecond * 100)
	currentGoroutines := runtime.NumGoroutine()

	if currentGoroutines <= initialGoroutines {
		t.Error("Goroutine workload should increase goroutine count")
	}

	// Wait for workload to complete
	time.Sleep(time.Millisecond * 400)

	// Check status
	status := simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive after workload completion")
	}
}

func TestWorkloadSimulator_GCWorkload(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Get initial GC stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialGC := m1.NumGC

	// Start GC workload
	err := simulator.StartWorkload("gc", time.Millisecond*300, "high")
	if err != nil {
		t.Fatalf("Failed to start GC workload: %v", err)
	}

	// Wait for some GC activity
	time.Sleep(time.Millisecond * 200)

	// Check that GC count increased
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	currentGC := m2.NumGC

	if currentGC <= initialGC {
		t.Error("GC workload should trigger garbage collection")
	}

	// Wait for workload to complete
	time.Sleep(time.Millisecond * 200)

	// Check status
	status := simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive after workload completion")
	}
}

func TestWorkloadSimulator_MixedWorkload(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Get initial stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialAlloc := m1.Alloc
	initialGoroutines := runtime.NumGoroutine()

	// Start mixed workload
	err := simulator.StartWorkload("mixed", time.Millisecond*500, "medium")
	if err != nil {
		t.Fatalf("Failed to start mixed workload: %v", err)
	}

	// Check status
	status := simulator.GetStatus()
	if !status.Active {
		t.Error("Simulator should be active")
	}

	if status.Workload != "mixed" {
		t.Errorf("Expected workload type 'mixed', got '%s'", status.Workload)
	}

	// Wait for some activity
	time.Sleep(time.Millisecond * 200)

	// Check that resources are being used
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	currentAlloc := m2.Alloc
	currentGoroutines := runtime.NumGoroutine()

	if currentAlloc <= initialAlloc {
		t.Error("Mixed workload should increase memory usage")
	}

	if currentGoroutines <= initialGoroutines {
		t.Error("Mixed workload should increase goroutine count")
	}

	// Wait for workload to complete
	time.Sleep(time.Millisecond * 400)

	// Check status
	status = simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive after workload completion")
	}
}

func TestWorkloadSimulator_StopWorkload(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Start a long-running workload
	err := simulator.StartWorkload("cpu", time.Second*10, "low")
	if err != nil {
		t.Fatalf("Failed to start CPU workload: %v", err)
	}

	// Check that it's active
	status := simulator.GetStatus()
	if !status.Active {
		t.Error("Simulator should be active")
	}

	// Stop the workload
	time.Sleep(time.Millisecond * 100)
	simulator.Stop()

	// Check that it stopped
	time.Sleep(time.Millisecond * 100)
	status = simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should be inactive after stop")
	}
}

func TestWorkloadSimulator_ConcurrentWorkloadPrevention(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Start first workload
	err := simulator.StartWorkload("cpu", time.Second, "low")
	if err != nil {
		t.Fatalf("Failed to start first workload: %v", err)
	}

	// Try to start second workload (should fail)
	err = simulator.StartWorkload("memory", time.Second, "medium")
	if err == nil {
		t.Error("Should not be able to start concurrent workloads")
	}

	// Stop first workload
	simulator.Stop()
	time.Sleep(time.Millisecond * 100)

	// Now should be able to start new workload
	err = simulator.StartWorkload("memory", time.Millisecond*200, "medium")
	if err != nil {
		t.Errorf("Should be able to start workload after stopping previous: %v", err)
	}

	// Clean up
	simulator.Stop()
}

func TestWorkloadSimulator_InvalidWorkloadType(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Try to start invalid workload type
	err := simulator.StartWorkload("invalid", time.Second, "medium")
	if err == nil {
		t.Error("Should return error for invalid workload type")
	}

	// Check that simulator is still inactive
	status := simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should remain inactive after invalid workload")
	}
}

func TestWorkloadSimulator_InvalidIntensity(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Try to start workload with invalid intensity
	err := simulator.StartWorkload("cpu", time.Second, "invalid")
	if err == nil {
		t.Error("Should return error for invalid intensity")
	}

	// Check that simulator is still inactive
	status := simulator.GetStatus()
	if status.Active {
		t.Error("Simulator should remain inactive after invalid intensity")
	}
}

func TestWorkloadSimulator_IntensityLevels(t *testing.T) {
	simulator := NewWorkloadSimulator()

	intensities := []string{"low", "medium", "high"}

	for _, intensity := range intensities {
		// Start workload with this intensity
		err := simulator.StartWorkload("cpu", time.Millisecond*200, intensity)
		if err != nil {
			t.Errorf("Failed to start workload with intensity '%s': %v", intensity, err)
			continue
		}

		// Check status
		status := simulator.GetStatus()
		if status.Intensity != intensity {
			t.Errorf("Expected intensity '%s', got '%s'", intensity, status.Intensity)
		}

		// Wait for completion
		time.Sleep(time.Millisecond * 250)

		// Ensure it stopped
		status = simulator.GetStatus()
		if status.Active {
			t.Errorf("Workload with intensity '%s' should have stopped", intensity)
		}
	}
}

func TestWorkloadSimulator_StatusReporting(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Start workload
	err := simulator.StartWorkload("memory", time.Millisecond*300, "medium")
	if err != nil {
		t.Fatalf("Failed to start workload: %v", err)
	}

	// Check status fields
	status := simulator.GetStatus()

	if !status.Active {
		t.Error("Status should show active")
	}

	if status.Workload != "memory" {
		t.Errorf("Expected workload type 'memory', got '%s'", status.Workload)
	}

	if status.Intensity != "medium" {
		t.Errorf("Expected intensity 'medium', got '%s'", status.Intensity)
	}

	// Wait for completion
	time.Sleep(time.Millisecond * 350)

	// Check final status
	status2 := simulator.GetStatus()
	if status2.Active {
		t.Error("Status should show inactive after completion")
	}
}

// Performance Tests

func TestWorkloadSimulator_PerformanceRequirements(t *testing.T) {
	simulator := NewWorkloadSimulator()

	// Test workload startup time (should be <1 second)
	start := time.Now()
	err := simulator.StartWorkload("cpu", time.Millisecond*100, "low")
	startupTime := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to start workload: %v", err)
	}

	if startupTime > time.Second {
		t.Errorf("Workload startup took %v, requirement: <1 second", startupTime)
	}

	// Test status reporting time (should be <100ms)
	start = time.Now()
	status := simulator.GetStatus()
	statusTime := time.Since(start)

	if statusTime > time.Millisecond*100 {
		t.Errorf("Status reporting took %v, requirement: <100ms", statusTime)
	}

	// Test stop time (should be <500ms)
	start = time.Now()
	simulator.Stop()
	stopTime := time.Since(start)

	if stopTime > time.Millisecond*500 {
		t.Errorf("Workload stop took %v, requirement: <500ms", stopTime)
	}

	// Verify status is accurate
	if status.Active != true {
		t.Error("Status should reflect active state")
	}

	t.Logf("Performance metrics:")
	t.Logf("  Startup time: %v (target: <1 second)", startupTime)
	t.Logf("  Status time: %v (target: <100ms)", statusTime)
	t.Logf("  Stop time: %v (target: <500ms)", stopTime)
}

// Benchmark Tests

func BenchmarkWorkloadSimulator_StartStop(b *testing.B) {
	simulator := NewWorkloadSimulator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := simulator.StartWorkload("cpu", time.Millisecond*10, "low")
		if err != nil {
			b.Fatalf("Failed to start workload: %v", err)
		}
		time.Sleep(time.Millisecond * 15)
		simulator.Stop()
		time.Sleep(time.Millisecond * 5)
	}
}

func BenchmarkWorkloadSimulator_GetStatus(b *testing.B) {
	simulator := NewWorkloadSimulator()

	// Start a workload
	err := simulator.StartWorkload("cpu", time.Second*10, "low")
	if err != nil {
		b.Fatalf("Failed to start workload: %v", err)
	}
	defer simulator.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := simulator.GetStatus()
		_ = status
	}
}

func BenchmarkWorkloadSimulator_CPUWorkload(b *testing.B) {
	simulator := NewWorkloadSimulator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := simulator.StartWorkload("cpu", time.Millisecond*50, "low")
		if err != nil {
			b.Fatalf("Failed to start workload: %v", err)
		}
		time.Sleep(time.Millisecond * 60)
	}
}