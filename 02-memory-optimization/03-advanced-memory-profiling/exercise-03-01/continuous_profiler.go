package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

// ContinuousProfiler manages automated memory profile collection
type ContinuousProfiler struct {
	interval     time.Duration
	storageDir   string
	running      bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	alertManager *AlertManager
	storage      *ProfileStorage
}

// NewContinuousProfiler creates a new continuous profiler
func NewContinuousProfiler(interval time.Duration, storageDir string) *ContinuousProfiler {
	ctx, cancel := context.WithCancel(context.Background())
	cp := &ContinuousProfiler{
		interval:   interval,
		storageDir: storageDir,
		ctx:        ctx,
		cancel:     cancel,
		storage:    NewProfileStorage(storageDir),
	}
	cp.alertManager = NewAlertManager(cp)
	return cp
}

// Start begins continuous profiling
func (cp *ContinuousProfiler) Start() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	if cp.running {
		return fmt.Errorf("profiler already running")
	}
	
	// Ensure storage directory exists
	if err := os.MkdirAll(cp.storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	cp.running = true
	go cp.profileLoop()
	
	fmt.Printf("Continuous profiler started with %v interval\n", cp.interval)
	return nil
}

// Stop stops continuous profiling
func (cp *ContinuousProfiler) Stop() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	if !cp.running {
		return
	}
	
	cp.cancel()
	cp.running = false
	fmt.Println("Continuous profiler stopped")
}

// IsRunning returns whether the profiler is currently running
func (cp *ContinuousProfiler) IsRunning() bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.running
}

// profileLoop runs the continuous profiling loop
func (cp *ContinuousProfiler) profileLoop() {
	ticker := time.NewTicker(cp.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			if err := cp.collectProfile(); err != nil {
				fmt.Printf("Error collecting profile: %v\n", err)
			}
		}
	}
}

// collectProfile collects a single memory profile
func (cp *ContinuousProfiler) collectProfile() error {
	timestamp := time.Now()
	filename := fmt.Sprintf("heap_%s.prof", timestamp.Format("20060102_150405"))
	filepath := filepath.Join(cp.storageDir, filename)
	
	// Create profile file
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()
	
	// Force garbage collection for accurate heap profile
	runtime.GC()
	
	// Write heap profile
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}
	
	// Store profile metadata
	profileInfo := &ProfileInfo{
		Timestamp: timestamp,
		Filename:  filename,
		Filepath:  filepath,
		Type:      "heap",
	}
	
	if err := cp.storage.StoreProfile(profileInfo); err != nil {
		return fmt.Errorf("failed to store profile metadata: %w", err)
	}
	
	// Analyze profile for anomalies
	if err := cp.analyzeProfile(profileInfo); err != nil {
		fmt.Printf("Error analyzing profile: %v\n", err)
	}
	
	fmt.Printf("Profile collected: %s\n", filename)
	return nil
}

// analyzeProfile analyzes a profile for memory anomalies
func (cp *ContinuousProfiler) analyzeProfile(profile *ProfileInfo) error {
	// Get current memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Create memory metrics
	metrics := &MemoryMetrics{
		Timestamp:    profile.Timestamp,
		HeapAlloc:    m.Alloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		TotalAlloc:   m.TotalAlloc,
		NumGC:        m.NumGC,
		GCCPUFraction: m.GCCPUFraction,
	}
	
	// Check for anomalies
	return cp.alertManager.CheckMemoryAnomaly(metrics)
}

// GetStats returns profiler statistics
func (cp *ContinuousProfiler) GetStats() *ProfilerStats {
	return &ProfilerStats{
		Running:        cp.IsRunning(),
		Interval:       cp.interval,
		StorageDir:     cp.storageDir,
		ProfilesCount:  cp.storage.GetProfileCount(),
		LastCollection: cp.storage.GetLastCollectionTime(),
	}
}

// GetCurrentMetrics returns current memory metrics
func (cp *ContinuousProfiler) GetCurrentMetrics() *MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &MemoryMetrics{
		Timestamp:     time.Now(),
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		HeapObjects:   m.HeapObjects,
		NumGC:         m.NumGC,
		GCCPUFraction: m.GCCPUFraction,
	}
}

// GetStorage returns the internal storage instance
func (cp *ContinuousProfiler) GetStorage() *ProfileStorage {
	return cp.storage
}

// ProfilerStats contains profiler statistics
type ProfilerStats struct {
	Running        bool          `json:"running"`
	Interval       time.Duration `json:"interval"`
	StorageDir     string        `json:"storage_dir"`
	ProfilesCount  int           `json:"profiles_count"`
	LastCollection time.Time     `json:"last_collection"`
}

// MemoryMetrics contains memory usage metrics
type MemoryMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	Alloc         uint64    `json:"alloc"`
	TotalAlloc    uint64    `json:"total_alloc"`
	Sys           uint64    `json:"sys"`
	HeapAlloc     uint64    `json:"heap_alloc"`
	HeapSys       uint64    `json:"heap_sys"`
	HeapIdle      uint64    `json:"heap_idle"`
	HeapInuse     uint64    `json:"heap_inuse"`
	HeapObjects   uint64    `json:"heap_objects"`
	NumGC         uint32    `json:"num_gc"`
	GCCPUFraction float64   `json:"gc_cpu_fraction"`
}

// Example usage
func main() {
	// Create profiler with 30-second interval
	profiler := NewContinuousProfiler(30*time.Second, "./profiles")
	
	// Start profiling
	if err := profiler.Start(); err != nil {
		fmt.Printf("Failed to start profiler: %v\n", err)
		return
	}
	
	// Simulate application work
	go simulateWork()
	
	// Run for 5 minutes
	time.Sleep(5 * time.Minute)
	
	// Stop profiling
	profiler.Stop()
	
	// Print stats
	stats := profiler.GetStats()
	fmt.Printf("Profiler stats: %+v\n", stats)
}

// simulateWork simulates application workload
func simulateWork() {
	for {
		// Allocate some memory
		data := make([]byte, 1024*1024) // 1MB
		_ = data
		
		time.Sleep(1 * time.Second)
	}
}