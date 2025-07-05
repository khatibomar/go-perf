package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkContinuousProfiler tests the performance of the continuous profiler
func BenchmarkContinuousProfiler(b *testing.B) {
	tempDir := b.TempDir()
	profiler := NewContinuousProfiler(30*time.Second, tempDir)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate profile collection
		if err := profiler.collectProfile(); err != nil {
			b.Fatalf("Failed to collect profile: %v", err)
		}
	}
}

// BenchmarkProfileStorage tests the performance of profile storage operations
func BenchmarkProfileStorage(b *testing.B) {
	tempDir := b.TempDir()
	storage := NewProfileStorage(tempDir)
	
	// Create test profile info
	profile := &ProfileInfo{
		Timestamp: time.Now(),
		Filename:  "test_profile.prof",
		Filepath:  filepath.Join(tempDir, "test_profile.prof"),
		Type:      "heap",
		Size:      1024,
	}
	
	// Create a dummy profile file
	file, err := os.Create(profile.Filepath)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	file.Write(make([]byte, 1024))
	file.Close()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Test storage operations
		profile.Filename = fmt.Sprintf("test_profile_%d.prof", i)
		profile.Filepath = filepath.Join(tempDir, profile.Filename)
		
		if err := storage.StoreProfile(profile); err != nil {
			b.Fatalf("Failed to store profile: %v", err)
		}
	}
}

// BenchmarkMemoryMetrics tests the performance of memory metrics collection
func BenchmarkMemoryMetrics(b *testing.B) {
	tempDir := b.TempDir()
	profiler := NewContinuousProfiler(30*time.Second, tempDir)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Test metrics collection
		metrics := profiler.GetCurrentMetrics()
		if metrics == nil {
			b.Fatal("Failed to get metrics")
		}
	}
}

// BenchmarkAlertManager tests the performance of alert rule evaluation
func BenchmarkAlertManager(b *testing.B) {
	tempDir := b.TempDir()
	profiler := NewContinuousProfiler(30*time.Second, tempDir)
	alertManager := NewAlertManager(profiler)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Test alert rule checking
		alertManager.checkRules()
	}
}

// BenchmarkProfileCleanup tests the performance of profile cleanup operations
func BenchmarkProfileCleanup(b *testing.B) {
	tempDir := b.TempDir()
	storage := NewProfileStorage(tempDir)
	
	// Create multiple test profiles
	for i := 0; i < 100; i++ {
		profile := &ProfileInfo{
			Timestamp: time.Now().AddDate(0, 0, -i), // Spread across days
			Filename:  fmt.Sprintf("test_profile_%d.prof", i),
			Filepath:  filepath.Join(tempDir, fmt.Sprintf("test_profile_%d.prof", i)),
			Type:      "heap",
			Size:      1024,
		}
		
		// Create dummy file
		file, err := os.Create(profile.Filepath)
		if err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
		file.Write(make([]byte, 1024))
		file.Close()
		
		storage.StoreProfile(profile)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Test cleanup operations
		if err := storage.CleanupOldProfiles(); err != nil {
			b.Fatalf("Failed to cleanup profiles: %v", err)
		}
	}
}

// TestContinuousProfilerIntegration tests the integration of all components
func TestContinuousProfilerIntegration(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create profiler and get its internal storage
	profiler := NewContinuousProfiler(100*time.Millisecond, tempDir)
	storage := profiler.GetStorage()
	alertManager := NewAlertManager(profiler)
	
	// Start profiler for a short duration
	if err := profiler.Start(); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	
	// Start alert manager
	alertManager.Start(50 * time.Millisecond)
	
	// Wait for some profiles to be collected
	time.Sleep(500 * time.Millisecond)
	
	// Stop profiler
	profiler.Stop()
	alertManager.Stop()
	
	// Verify profiles were collected
	profiles := storage.GetProfiles()
	if len(profiles) == 0 {
		t.Error("No profiles were collected")
	}
	
	// Verify stats
	stats := profiler.GetStats()
	if stats.ProfilesCount == 0 {
		t.Error("No profile collections recorded")
	}
	
	// Verify metrics
	metrics := profiler.GetCurrentMetrics()
	if metrics == nil {
		t.Error("Failed to get current metrics")
	}
	
	// Verify storage stats
	storageStats := storage.GetStorageStats()
	if storageStats.TotalProfiles == 0 {
		t.Error("No profiles in storage")
	}
	
	t.Logf("Integration test completed successfully:")
	t.Logf("- Profiles collected: %d", len(profiles))
	t.Logf("- Collection count: %d", stats.ProfilesCount)
	t.Logf("- Storage size: %s", storageStats.FormatSize())
}

// TestProfileStorageOperations tests profile storage functionality
func TestProfileStorageOperations(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewProfileStorage(tempDir)
	
	// Test storing a profile
	profile := &ProfileInfo{
		Timestamp: time.Now(),
		Filename:  "test_profile.prof",
		Filepath:  filepath.Join(tempDir, "test_profile.prof"),
		Type:      "heap",
		Size:      1024,
	}
	
	// Create dummy file
	file, err := os.Create(profile.Filepath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Write(make([]byte, 1024))
	file.Close()
	
	if err := storage.StoreProfile(profile); err != nil {
		t.Fatalf("Failed to store profile: %v", err)
	}
	
	// Test retrieving the profile
	retrieved, exists := storage.GetProfile("test_profile.prof")
	if !exists {
		t.Error("Profile not found after storage")
	}
	if retrieved.Filename != profile.Filename {
		t.Errorf("Retrieved profile filename mismatch: got %s, want %s", retrieved.Filename, profile.Filename)
	}
	
	// Test getting all profiles
	profiles := storage.GetProfiles()
	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(profiles))
	}
	
	// Test storage stats
	stats := storage.GetStorageStats()
	if stats.TotalProfiles != 1 {
		t.Errorf("Expected 1 profile in stats, got %d", stats.TotalProfiles)
	}
	if stats.TotalSize != 1024 {
		t.Errorf("Expected 1024 bytes in stats, got %d", stats.TotalSize)
	}
}

// TestAlertManagerRules tests alert rule evaluation
func TestAlertManagerRules(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewContinuousProfiler(30*time.Second, tempDir)
	alertManager := NewAlertManager(profiler)
	
	// Add a test rule that should trigger
	testRule := &AlertRule{
		Name:     "TestRule",
		Level:    AlertLevelWarning,
		Cooldown: 1 * time.Second,
		Condition: func(m *MemoryMetrics) bool {
			return true // Always trigger
		},
		Message: "Test alert message",
	}
	alertManager.AddRule(testRule)
	
	// Check rules manually
	alertManager.checkRules()
	
	// Verify alert was created
	alerts := alertManager.GetAlerts()
	if len(alerts) == 0 {
		t.Error("No alerts were generated")
	}
	
	activeAlerts := alertManager.GetActiveAlerts()
	if len(activeAlerts) == 0 {
		t.Error("No active alerts found")
	}
	
	// Test alert resolution
	if len(alerts) > 0 {
		alertManager.ResolveAlert(alerts[0].ID)
		activeAfterResolve := alertManager.GetActiveAlerts()
		if len(activeAfterResolve) >= len(activeAlerts) {
			t.Error("Alert was not resolved")
		}
	}
}

// Helper function to create test data
func createTestProfile(dir string, name string, size int64) (*ProfileInfo, error) {
	filepath := filepath.Join(dir, name)
	file, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	// Write test data
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	file.Write(data)
	
	return &ProfileInfo{
		Timestamp: time.Now(),
		Filename:  name,
		Filepath:  filepath,
		Type:      "heap",
		Size:      size,
	}, nil
}