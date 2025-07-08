// debugging/memory_analyzer.go
package debugging

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

type MemoryAnalyzer struct {
	config          AnalyzerConfig
	snapshots       []MemorySnapshot
	mu              sync.RWMutex
	alertThresholds AlertThresholds
	callbacks       []LeakCallback
}

type AnalyzerConfig struct {
	SnapshotInterval     time.Duration
	MaxSnapshots         int
	LeakThreshold        float64 // Percentage increase
	ConsecutiveChecks    int
	EnableGCOptimization bool
}

type MemorySnapshot struct {
	Timestamp     time.Time `json:"timestamp"`
	HeapAlloc     uint64    `json:"heap_alloc"`
	HeapSys       uint64    `json:"heap_sys"`
	HeapIdle      uint64    `json:"heap_idle"`
	HeapInuse     uint64    `json:"heap_inuse"`
	HeapReleased  uint64    `json:"heap_released"`
	HeapObjects   uint64    `json:"heap_objects"`
	StackInuse    uint64    `json:"stack_inuse"`
	StackSys      uint64    `json:"stack_sys"`
	MSpanInuse    uint64    `json:"mspan_inuse"`
	MSpanSys      uint64    `json:"mspan_sys"`
	MCacheInuse   uint64    `json:"mcache_inuse"`
	MCacheSys     uint64    `json:"mcache_sys"`
	GCSys         uint64    `json:"gc_sys"`
	OtherSys      uint64    `json:"other_sys"`
	NextGC        uint64    `json:"next_gc"`
	LastGC        time.Time `json:"last_gc"`
	NumGC         uint32    `json:"num_gc"`
	NumForcedGC   uint32    `json:"num_forced_gc"`
	GCCPUFraction float64   `json:"gc_cpu_fraction"`
	PauseTotalNs  uint64    `json:"pause_total_ns"`
	Goroutines    int       `json:"goroutines"`
}

type AlertThresholds struct {
	HeapGrowthRate      float64
	GoroutineGrowth     int
	GCFrequency         time.Duration
	MemoryUtilization   float64
}

type LeakCallback func(leak MemoryLeak)

type MemoryLeak struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
	Timestamp   time.Time              `json:"timestamp"`
	Snapshots   []MemorySnapshot       `json:"snapshots"`
}

func NewMemoryAnalyzer(config AnalyzerConfig) *MemoryAnalyzer {
	return &MemoryAnalyzer{
		config:    config,
		snapshots: make([]MemorySnapshot, 0, config.MaxSnapshots),
		alertThresholds: AlertThresholds{
			HeapGrowthRate:      0.1, // 10% growth
			GoroutineGrowth:     100,
			GCFrequency:         time.Minute,
			MemoryUtilization:   0.8, // 80% utilization
		},
	}
}

func (ma *MemoryAnalyzer) Start(ctx context.Context) {
	ticker := time.NewTicker(ma.config.SnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ma.takeSnapshot()
			ma.analyzeLeaks()
		}
	}
}

func (ma *MemoryAnalyzer) takeSnapshot() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := MemorySnapshot{
		Timestamp:     time.Now(),
		HeapAlloc:     m.Alloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		HeapReleased:  m.HeapReleased,
		HeapObjects:   m.HeapObjects,
		StackInuse:    m.StackInuse,
		StackSys:      m.StackSys,
		MSpanInuse:    m.MSpanInuse,
		MSpanSys:      m.MSpanSys,
		MCacheInuse:   m.MCacheInuse,
		MCacheSys:     m.MCacheSys,
		GCSys:         m.GCSys,
		OtherSys:      m.OtherSys,
		NextGC:        m.NextGC,
		LastGC:        time.Unix(0, int64(m.LastGC)),
		NumGC:         m.NumGC,
		NumForcedGC:   m.NumForcedGC,
		GCCPUFraction: m.GCCPUFraction,
		PauseTotalNs:  m.PauseTotalNs,
		Goroutines:    runtime.NumGoroutine(),
	}

	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.snapshots = append(ma.snapshots, snapshot)

	// Keep only the last N snapshots
	if len(ma.snapshots) > ma.config.MaxSnapshots {
		ma.snapshots = ma.snapshots[1:]
	}
}

func (ma *MemoryAnalyzer) analyzeLeaks() {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if len(ma.snapshots) < ma.config.ConsecutiveChecks {
		return
	}

	recent := ma.snapshots[len(ma.snapshots)-ma.config.ConsecutiveChecks:]

	// Check for heap growth
	if leak := ma.checkHeapGrowth(recent); leak != nil {
		ma.notifyLeak(*leak)
	}

	// Check for goroutine leaks
	if leak := ma.checkGoroutineLeak(recent); leak != nil {
		ma.notifyLeak(*leak)
	}

	// Check GC efficiency
	if leak := ma.checkGCEfficiency(recent); leak != nil {
		ma.notifyLeak(*leak)
	}
}

func (ma *MemoryAnalyzer) checkHeapGrowth(snapshots []MemorySnapshot) *MemoryLeak {
	if len(snapshots) < 2 {
		return nil
	}

	first := snapshots[0]
	last := snapshots[len(snapshots)-1]

	if first.HeapAlloc == 0 {
		return nil // Avoid division by zero
	}

	growthRate := float64(last.HeapAlloc-first.HeapAlloc) / float64(first.HeapAlloc)

	if growthRate > ma.alertThresholds.HeapGrowthRate {
		return &MemoryLeak{
			Type:        "heap_growth",
			Severity:    ma.getSeverity(growthRate, ma.alertThresholds.HeapGrowthRate),
			Description: fmt.Sprintf("Heap memory grew by %.2f%% over %d snapshots", growthRate*100, len(snapshots)),
			Metrics: map[string]interface{}{
				"growth_rate":  growthRate,
				"initial_heap": first.HeapAlloc,
				"current_heap": last.HeapAlloc,
				"time_span":    last.Timestamp.Sub(first.Timestamp),
			},
			Timestamp: time.Now(),
			Snapshots: snapshots,
		}
	}

	return nil
}

func (ma *MemoryAnalyzer) checkGoroutineLeak(snapshots []MemorySnapshot) *MemoryLeak {
	if len(snapshots) < 2 {
		return nil
	}

	first := snapshots[0]
	last := snapshots[len(snapshots)-1]

	growth := last.Goroutines - first.Goroutines

	if growth > ma.alertThresholds.GoroutineGrowth {
		return &MemoryLeak{
			Type:        "goroutine_leak",
			Severity:    ma.getSeverity(float64(growth), float64(ma.alertThresholds.GoroutineGrowth)),
			Description: fmt.Sprintf("Goroutine count increased by %d over %d snapshots", growth, len(snapshots)),
			Metrics: map[string]interface{}{
				"growth":        growth,
				"initial_count": first.Goroutines,
				"current_count": last.Goroutines,
				"time_span":     last.Timestamp.Sub(first.Timestamp),
			},
			Timestamp: time.Now(),
			Snapshots: snapshots,
		}
	}

	return nil
}

func (ma *MemoryAnalyzer) checkGCEfficiency(snapshots []MemorySnapshot) *MemoryLeak {
	if len(snapshots) < 2 {
		return nil
	}

	last := snapshots[len(snapshots)-1]

	// Check if GC is running too frequently
	if last.NumGC == 0 {
		return nil // Avoid division by zero
	}

	avgGCInterval := time.Since(last.LastGC) / time.Duration(last.NumGC)

	if avgGCInterval < ma.alertThresholds.GCFrequency {
		return &MemoryLeak{
			Type:        "gc_inefficiency",
			Severity:    "warning",
			Description: fmt.Sprintf("GC running too frequently: average interval %v", avgGCInterval),
			Metrics: map[string]interface{}{
				"avg_gc_interval": avgGCInterval,
				"gc_cpu_fraction": last.GCCPUFraction,
				"num_gc":          last.NumGC,
				"num_forced_gc":   last.NumForcedGC,
			},
			Timestamp: time.Now(),
			Snapshots: snapshots,
		}
	}

	return nil
}

func (ma *MemoryAnalyzer) getSeverity(actual, threshold float64) string {
	ratio := actual / threshold
	switch {
	case ratio > 3:
		return "critical"
	case ratio > 2:
		return "high"
	case ratio > 1.5:
		return "medium"
	default:
		return "low"
	}
}

func (ma *MemoryAnalyzer) notifyLeak(leak MemoryLeak) {
	for _, callback := range ma.callbacks {
		go callback(leak)
	}
}

func (ma *MemoryAnalyzer) RegisterLeakCallback(callback LeakCallback) {
	ma.callbacks = append(ma.callbacks, callback)
}

func (ma *MemoryAnalyzer) GetCurrentSnapshot() MemorySnapshot {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if len(ma.snapshots) == 0 {
		return MemorySnapshot{}
	}

	return ma.snapshots[len(ma.snapshots)-1]
}

func (ma *MemoryAnalyzer) GetSnapshots(since time.Time) []MemorySnapshot {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	var result []MemorySnapshot
	for _, snapshot := range ma.snapshots {
		if snapshot.Timestamp.After(since) {
			result = append(result, snapshot)
		}
	}

	return result
}

// Force garbage collection and return before/after stats
func (ma *MemoryAnalyzer) ForceGCAndAnalyze() (before, after MemorySnapshot, freed uint64) {
	// Take snapshot before GC
	ma.takeSnapshot()
	before = ma.GetCurrentSnapshot()

	// Force garbage collection
	runtime.GC()
	runtime.GC() // Run twice to ensure cleanup

	// Take snapshot after GC
	time.Sleep(100 * time.Millisecond) // Allow GC to complete
	ma.takeSnapshot()
	after = ma.GetCurrentSnapshot()

	freed = before.HeapAlloc - after.HeapAlloc

	return before, after, freed
}

// Optimize GC settings based on current memory patterns
func (ma *MemoryAnalyzer) OptimizeGC() {
	if !ma.config.EnableGCOptimization {
		return
	}

	current := ma.GetCurrentSnapshot()

	// Adjust GOGC based on heap size and growth patterns
	if current.HeapAlloc > 100*1024*1024 { // > 100MB
		debug.SetGCPercent(50) // More aggressive GC for large heaps
	} else {
		debug.SetGCPercent(100) // Default GC
	}

	// Set memory limit if heap is growing too fast
	ma.mu.RLock()
	if len(ma.snapshots) >= 2 {
		recent := ma.snapshots[len(ma.snapshots)-2:]
		if recent[0].HeapAlloc > 0 {
			growthRate := float64(recent[1].HeapAlloc-recent[0].HeapAlloc) / float64(recent[0].HeapAlloc)

			if growthRate > 0.2 { // 20% growth
				// Set soft memory limit to current heap * 1.5
				limit := int64(current.HeapAlloc * 3 / 2)
				debug.SetMemoryLimit(limit)
			}
		}
	}
	ma.mu.RUnlock()
}