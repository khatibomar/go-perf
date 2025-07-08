package profiling

import (
	"runtime"
	"testing"
	"time"
)

func TestLiveProfiler_StartProfile(t *testing.T) {
	// Enable profiling
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	// Create temporary storage
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create profiler
	config := ProfilerConfig{
		EnableContinuous: false, // Disable for testing
		ProfileDuration:  2 * time.Second,
		MaxConcurrent:    2,
		StoragePath:      tempDir,
		RetentionDays:    1,
	}

	profiler := NewLiveProfiler(config, storage)
	defer profiler.Shutdown()

	// Test CPU profile
	t.Run("CPU Profile", func(t *testing.T) {
		session, err := profiler.StartProfile(CPUProfile, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to start CPU profile: %v", err)
		}

		if session.Type != CPUProfile {
			t.Errorf("Expected profile type %s, got %s", CPUProfile, session.Type)
		}

		if session.Status != "running" {
			t.Errorf("Expected status 'running', got %s", session.Status)
		}

		// Wait for profile to complete
		time.Sleep(2 * time.Second)

		// Check that profile is no longer active
		activeProfiles := profiler.GetActiveProfiles()
		if _, exists := activeProfiles[CPUProfile]; exists {
			t.Error("CPU profile should no longer be active")
		}
	})

	// Test Memory profile
	t.Run("Memory Profile", func(t *testing.T) {
		session, err := profiler.StartProfile(MemProfile, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to start memory profile: %v", err)
		}

		if session.Type != MemProfile {
			t.Errorf("Expected profile type %s, got %s", MemProfile, session.Type)
		}

		// Memory profiles complete quickly
		time.Sleep(500 * time.Millisecond)
	})

	// Test Goroutine profile
	t.Run("Goroutine Profile", func(t *testing.T) {
		session, err := profiler.StartProfile(GoroutineProfile, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to start goroutine profile: %v", err)
		}

		if session.Type != GoroutineProfile {
			t.Errorf("Expected profile type %s, got %s", GoroutineProfile, session.Type)
		}

		// Goroutine profiles complete quickly
		time.Sleep(500 * time.Millisecond)
	})
}

func TestLiveProfiler_ConcurrentLimit(t *testing.T) {
	// Create temporary storage
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create profiler with low concurrent limit
	config := ProfilerConfig{
		EnableContinuous: false,
		ProfileDuration:  5 * time.Second,
		MaxConcurrent:    1, // Only allow 1 concurrent profile
		StoragePath:      tempDir,
	}

	profiler := NewLiveProfiler(config, storage)
	defer profiler.Shutdown()

	// Start first profile
	_, err = profiler.StartProfile(CPUProfile, 3*time.Second)
	if err != nil {
		t.Fatalf("Failed to start first profile: %v", err)
	}

	// Try to start second profile - should fail
	_, err = profiler.StartProfile(MemProfile, 1*time.Second)
	if err == nil {
		t.Error("Expected error when exceeding concurrent limit")
	}

	// Wait for first profile to complete
	time.Sleep(4 * time.Second)

	// Now second profile should work
	_, err = profiler.StartProfile(MemProfile, 1*time.Second)
	if err != nil {
		t.Errorf("Expected to be able to start profile after first completed: %v", err)
	}
}

func TestLiveProfiler_DuplicateProfile(t *testing.T) {
	// Create temporary storage
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	config := ProfilerConfig{
		EnableContinuous: false,
		ProfileDuration:  3 * time.Second,
		MaxConcurrent:    5,
		StoragePath:      tempDir,
	}

	profiler := NewLiveProfiler(config, storage)
	defer profiler.Shutdown()

	// Start first CPU profile
	_, err = profiler.StartProfile(CPUProfile, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to start first CPU profile: %v", err)
	}

	// Try to start another CPU profile - should fail
	_, err = profiler.StartProfile(CPUProfile, 1*time.Second)
	if err == nil {
		t.Error("Expected error when starting duplicate profile type")
	}
}

func TestLiveProfiler_StopProfile(t *testing.T) {
	// Create temporary storage
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	config := ProfilerConfig{
		EnableContinuous: false,
		ProfileDuration:  10 * time.Second, // Long duration
		MaxConcurrent:    5,
		StoragePath:      tempDir,
	}

	profiler := NewLiveProfiler(config, storage)
	defer profiler.Shutdown()

	// Start profile
	_, err = profiler.StartProfile(CPUProfile, 10*time.Second)
	if err != nil {
		t.Fatalf("Failed to start profile: %v", err)
	}

	// Verify it's running
	activeProfiles := profiler.GetActiveProfiles()
	if _, exists := activeProfiles[CPUProfile]; !exists {
		t.Error("Profile should be active")
	}

	// Stop the profile
	err = profiler.StopProfile(CPUProfile)
	if err != nil {
		t.Errorf("Failed to stop profile: %v", err)
	}

	// Verify it's no longer running
	activeProfiles = profiler.GetActiveProfiles()
	if _, exists := activeProfiles[CPUProfile]; exists {
		t.Error("Profile should no longer be active")
	}

	// Try to stop non-existent profile
	err = profiler.StopProfile(MemProfile)
	if err == nil {
		t.Error("Expected error when stopping non-existent profile")
	}
}

func TestProfileScheduler(t *testing.T) {
	// Create temporary storage
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create profiler with continuous profiling enabled
	config := ProfilerConfig{
		EnableContinuous: true,
		ProfileInterval:  1 * time.Second, // Very short interval for testing
		ProfileDuration:  500 * time.Millisecond, // Short duration
		MaxConcurrent:    5,
		StoragePath:      tempDir,
		RetentionDays:    1,
	}

	profiler := NewLiveProfiler(config, storage)
	defer profiler.Shutdown()

	// Wait for some profiles to be generated
	time.Sleep(3 * time.Second)

	// Check that profiles were stored
	profiles, err := storage.List(CPUProfile, time.Now().Add(-10*time.Second))
	if err != nil {
		t.Errorf("Failed to list profiles: %v", err)
	}

	if len(profiles) == 0 {
		t.Error("Expected some CPU profiles to be generated")
	}

	// Check scheduler status
	if profiler.scheduler == nil {
		t.Error("Scheduler should be created when continuous profiling is enabled")
	}

	if !profiler.scheduler.IsRunning() {
		t.Error("Scheduler should be running")
	}

	status := profiler.scheduler.GetStatus()
	if !status["running"].(bool) {
		t.Error("Scheduler status should show running")
	}
}

func TestFileStorage(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test data
	testData := []byte("test profile data")
	metadata := map[string]string{
		"timestamp": time.Now().Format(time.RFC3339),
		"duration":  "10s",
		"type":      "cpu",
	}

	// Store profile
	err = storage.Store(CPUProfile, testData, metadata)
	if err != nil {
		t.Errorf("Failed to store profile: %v", err)
	}

	// List profiles
	profiles, err := storage.List(CPUProfile, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Errorf("Failed to list profiles: %v", err)
	}

	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(profiles))
	}

	profile := profiles[0]
	if profile.Type != CPUProfile {
		t.Errorf("Expected profile type %s, got %s", CPUProfile, profile.Type)
	}

	// Get profile data
	data, err := storage.Get(profile.ID)
	if err != nil {
		t.Errorf("Failed to get profile data: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Profile data mismatch")
	}

	// Delete profile
	err = storage.Delete(profile.ID)
	if err != nil {
		t.Errorf("Failed to delete profile: %v", err)
	}

	// Verify deletion
	profiles, err = storage.List(CPUProfile, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Errorf("Failed to list profiles after deletion: %v", err)
	}

	if len(profiles) != 0 {
		t.Errorf("Expected 0 profiles after deletion, got %d", len(profiles))
	}
}

func TestFileStorage_Cleanup(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Store old profile
	oldMetadata := map[string]string{
		"timestamp": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		"duration":  "10s",
		"type":      "cpu",
	}
	err = storage.Store(CPUProfile, []byte("old data"), oldMetadata)
	if err != nil {
		t.Errorf("Failed to store old profile: %v", err)
	}

	// Store new profile
	newMetadata := map[string]string{
		"timestamp": time.Now().Format(time.RFC3339),
		"duration":  "10s",
		"type":      "cpu",
	}
	err = storage.Store(CPUProfile, []byte("new data"), newMetadata)
	if err != nil {
		t.Errorf("Failed to store new profile: %v", err)
	}

	// Cleanup profiles older than 1 hour
	err = storage.Cleanup(time.Now().Add(-1 * time.Hour))
	if err != nil {
		t.Errorf("Failed to cleanup: %v", err)
	}

	// Check that only new profile remains
	profiles, err := storage.List(CPUProfile, time.Time{})
	if err != nil {
		t.Errorf("Failed to list profiles: %v", err)
	}

	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile after cleanup, got %d", len(profiles))
	}
}

// Benchmark tests

func BenchmarkLiveProfiler_StartStopProfile(b *testing.B) {
	tempDir := b.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	config := ProfilerConfig{
		EnableContinuous: false,
		ProfileDuration:  100 * time.Millisecond,
		MaxConcurrent:    10,
		StoragePath:      tempDir,
	}

	profiler := NewLiveProfiler(config, storage)
	defer profiler.Shutdown()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		session, err := profiler.StartProfile(MemProfile, 50*time.Millisecond)
		if err != nil {
			b.Errorf("Failed to start profile: %v", err)
			continue
		}

		// Wait for profile to complete
		time.Sleep(100 * time.Millisecond)

		// Verify completion
		if session.Status == "running" {
			profiler.StopProfile(MemProfile)
		}
	}
}

func BenchmarkFileStorage_Store(b *testing.B) {
	tempDir := b.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	testData := make([]byte, 1024) // 1KB test data
	metadata := map[string]string{
		"timestamp": time.Now().Format(time.RFC3339),
		"duration":  "10s",
		"type":      "cpu",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := storage.Store(CPUProfile, testData, metadata)
		if err != nil {
			b.Errorf("Failed to store profile: %v", err)
		}
	}
}

// Performance requirements tests

func TestPerformanceRequirements(t *testing.T) {
	t.Run("Profile Collection Latency", func(t *testing.T) {
		tempDir := t.TempDir()
		storage, err := NewFileStorage(tempDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		config := ProfilerConfig{
			EnableContinuous: false,
			ProfileDuration:  1 * time.Second,
			MaxConcurrent:    5,
			StoragePath:      tempDir,
		}

		profiler := NewLiveProfiler(config, storage)
		defer profiler.Shutdown()

		// Test memory profile latency (should be < 5 seconds)
		start := time.Now()
		_, err = profiler.StartProfile(MemProfile, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to start profile: %v", err)
		}

		// Wait for completion
		time.Sleep(2 * time.Second)
		latency := time.Since(start)

		if latency > 5*time.Second {
			t.Errorf("Profile collection latency too high: %v (requirement: <5s)", latency)
		}
	})

	t.Run("Storage Performance", func(t *testing.T) {
		tempDir := t.TempDir()
		storage, err := NewFileStorage(tempDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		// Test storage of 100 profiles
		testData := make([]byte, 10*1024) // 10KB profile
		metadata := map[string]string{
			"timestamp": time.Now().Format(time.RFC3339),
			"duration":  "10s",
			"type":      "cpu",
		}

		start := time.Now()
		for i := 0; i < 100; i++ {
			err := storage.Store(CPUProfile, testData, metadata)
			if err != nil {
				t.Errorf("Failed to store profile %d: %v", i, err)
			}
		}
		duration := time.Since(start)

		// Should be able to store 100 profiles quickly
		if duration > 10*time.Second {
			t.Errorf("Storage performance too slow: %v for 100 profiles", duration)
		}
	})
}