package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ProfileEvent represents a profiling event
type ProfileEvent struct {
	PID       int       `json:"pid"`
	Timestamp time.Time `json:"timestamp"`
	Function  string    `json:"function"`
	Duration  int64     `json:"duration_ns"`
	EventType string    `json:"event_type"`
}

// FunctionStats holds statistics for a function
type FunctionStats struct {
	Count        int64   `json:"count"`
	TotalDuration int64   `json:"total_duration_ns"`
	MinDuration   int64   `json:"min_duration_ns"`
	MaxDuration   int64   `json:"max_duration_ns"`
	AvgDuration   float64 `json:"avg_duration_ns"`
}

// ProfilerStats holds overall profiler statistics
type ProfilerStats struct {
	EventsCollected   int64                      `json:"events_collected"`
	EventsPerSecond   float64                    `json:"events_per_second"`
	ProfilingOverhead float64                    `json:"profiling_overhead_percent"`
	FunctionStats     map[string]*FunctionStats  `json:"function_stats"`
	SystemMetrics     *SystemMetrics             `json:"system_metrics"`
}

// SystemMetrics holds system-level metrics
type SystemMetrics struct {
	CPUUsage     float64 `json:"cpu_usage_percent"`
	MemoryUsage  int64   `json:"memory_usage_bytes"`
	Goroutines   int     `json:"goroutines"`
	GCPauses     int64   `json:"gc_pauses_ns"`
	AllocRate    int64   `json:"alloc_rate_bytes_per_sec"`
}

// AdvancedProfiler simulates eBPF-like profiling capabilities
type AdvancedProfiler struct {
	mu              sync.RWMutex
	events          []ProfileEvent
	functionStats   map[string]*FunctionStats
	eventsCollected int64
	startTime       time.Time
	isRunning       bool
	ctx             context.Context
	cancel          context.CancelFunc
	maxEvents       int
	overheadStart   time.Time
	overheadTotal   time.Duration
}

// NewAdvancedProfiler creates a new advanced profiler
func NewAdvancedProfiler(maxEvents int) *AdvancedProfiler {
	ctx, cancel := context.WithCancel(context.Background())
	return &AdvancedProfiler{
		events:        make([]ProfileEvent, 0, maxEvents),
		functionStats: make(map[string]*FunctionStats),
		startTime:     time.Now(),
		ctx:           ctx,
		cancel:        cancel,
		maxEvents:     maxEvents,
	}
}

// Start begins profiling
func (p *AdvancedProfiler) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.isRunning {
		return
	}
	
	p.isRunning = true
	p.startTime = time.Now()
	
	// Start background monitoring
	go p.monitorSystemMetrics()
	go p.simulateKernelEvents()
	go p.monitorMemoryAllocations()
	go p.monitorGoroutines()
}

// Stop stops profiling
func (p *AdvancedProfiler) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.isRunning {
		return
	}
	
	p.isRunning = false
	p.cancel()
}

// RecordEvent records a profiling event
func (p *AdvancedProfiler) RecordEvent(function string, duration time.Duration, eventType string) {
	overheadStart := time.Now()
	
	event := ProfileEvent{
		PID:       os.Getpid(),
		Timestamp: time.Now(),
		Function:  function,
		Duration:  duration.Nanoseconds(),
		EventType: eventType,
	}
	
	p.mu.Lock()
	
	// Add event to buffer (circular buffer)
	if len(p.events) >= p.maxEvents {
		// Remove first element and append new one
		copy(p.events, p.events[1:])
		p.events[len(p.events)-1] = event
	} else {
		p.events = append(p.events, event)
	}
	
	// Update function statistics
	stats, exists := p.functionStats[function]
	if !exists {
		stats = &FunctionStats{
			MinDuration: duration.Nanoseconds(),
			MaxDuration: duration.Nanoseconds(),
		}
		p.functionStats[function] = stats
	}
	
	stats.Count++
	stats.TotalDuration += duration.Nanoseconds()
	if duration.Nanoseconds() < stats.MinDuration {
		stats.MinDuration = duration.Nanoseconds()
	}
	if duration.Nanoseconds() > stats.MaxDuration {
		stats.MaxDuration = duration.Nanoseconds()
	}
	stats.AvgDuration = float64(stats.TotalDuration) / float64(stats.Count)
	
	p.mu.Unlock()
	
	atomic.AddInt64(&p.eventsCollected, 1)
	
	// Track overhead with minimal impact
	atomic.AddInt64((*int64)(&p.overheadTotal), int64(time.Since(overheadStart)))
}

// GetStats returns profiler statistics
func (p *AdvancedProfiler) GetStats() ProfilerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	elapsed := time.Since(p.startTime).Seconds()
	eventsPerSecond := float64(atomic.LoadInt64(&p.eventsCollected)) / elapsed
	
	// Calculate profiling overhead using atomic load
	overheadNs := atomic.LoadInt64((*int64)(&p.overheadTotal))
	overheadPercent := (float64(overheadNs) / 1e9 / elapsed) * 100
	
	// Copy function stats
	funcStats := make(map[string]*FunctionStats)
	for k, v := range p.functionStats {
		funcStats[k] = &FunctionStats{
			Count:         v.Count,
			TotalDuration: v.TotalDuration,
			MinDuration:   v.MinDuration,
			MaxDuration:   v.MaxDuration,
			AvgDuration:   v.AvgDuration,
		}
	}
	
	return ProfilerStats{
		EventsCollected:   atomic.LoadInt64(&p.eventsCollected),
		EventsPerSecond:   eventsPerSecond,
		ProfilingOverhead: overheadPercent,
		FunctionStats:     funcStats,
		SystemMetrics:     p.getSystemMetrics(),
	}
}

// getSystemMetrics collects current system metrics
func (p *AdvancedProfiler) getSystemMetrics() *SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &SystemMetrics{
		CPUUsage:    p.getCPUUsage(),
		MemoryUsage: int64(m.Alloc),
		Goroutines:  runtime.NumGoroutine(),
		GCPauses:    int64(m.PauseTotalNs),
		AllocRate:   int64(m.TotalAlloc),
	}
}

// getCPUUsage simulates CPU usage calculation
func (p *AdvancedProfiler) getCPUUsage() float64 {
	// Simulate CPU usage based on goroutine count and activity
	goroutines := runtime.NumGoroutine()
	baseUsage := float64(goroutines) * 0.1
	if baseUsage > 100 {
		baseUsage = 100
	}
	return baseUsage
}

// monitorSystemMetrics monitors system metrics in background
func (p *AdvancedProfiler) monitorSystemMetrics() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.RecordEvent("system.monitor", time.Microsecond*50, "system")
		}
	}
}

// simulateKernelEvents simulates kernel-level events
func (p *AdvancedProfiler) simulateKernelEvents() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			// Simulate fewer kernel events to reduce overhead
			events := []string{"syscall.read", "syscall.write"}
			for _, event := range events {
				duration := time.Duration(100+len(event)*10) * time.Nanosecond
				p.RecordEvent(event, duration, "kernel")
			}
		}
	}
}

// monitorMemoryAllocations monitors memory allocation patterns
func (p *AdvancedProfiler) monitorMemoryAllocations() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.RecordEvent("runtime.mallocgc", time.Microsecond*2, "memory")
		}
	}
}

// monitorGoroutines monitors goroutine lifecycle
func (p *AdvancedProfiler) monitorGoroutines() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.RecordEvent("goroutine.create", time.Microsecond*10, "goroutine")
		}
	}
}

// GetEvents returns recent events
func (p *AdvancedProfiler) GetEvents(limit int) []ProfileEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if limit <= 0 || limit > len(p.events) {
		limit = len(p.events)
	}
	
	// Return most recent events
	start := len(p.events) - limit
	if start < 0 {
		start = 0
	}
	
	events := make([]ProfileEvent, limit)
	copy(events, p.events[start:])
	return events
}

// HTTP handlers for web interface
func (p *AdvancedProfiler) statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := p.GetStats()
	json.NewEncoder(w).Encode(stats)
}

func (p *AdvancedProfiler) eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	events := p.GetEvents(1000)
	json.NewEncoder(w).Encode(events)
}

func (p *AdvancedProfiler) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	stats := p.GetStats()
	
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Advanced eBPF-like Profiler</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 10px 0; padding: 10px; border: 1px solid #ddd; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .function-stats { max-height: 400px; overflow-y: auto; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>Advanced eBPF-like Profiler Dashboard</h1>
    
    <div class="stats-grid">
        <div class="metric">
            <h3>Profiler Statistics</h3>
            <p>Events Collected: ` + fmt.Sprintf("%d", stats.EventsCollected) + `</p>
            <p>Events/Second: ` + fmt.Sprintf("%.2f", stats.EventsPerSecond) + `</p>
            <p>Profiling Overhead: ` + fmt.Sprintf("%.4f%%", stats.ProfilingOverhead) + `</p>
        </div>
        
        <div class="metric">
            <h3>System Metrics</h3>
            <p>CPU Usage: ` + fmt.Sprintf("%.2f%%", stats.SystemMetrics.CPUUsage) + `</p>
            <p>Memory Usage: ` + fmt.Sprintf("%d bytes", stats.SystemMetrics.MemoryUsage) + `</p>
            <p>Goroutines: ` + fmt.Sprintf("%d", stats.SystemMetrics.Goroutines) + `</p>
            <p>GC Pauses: ` + fmt.Sprintf("%d ns", stats.SystemMetrics.GCPauses) + `</p>
        </div>
    </div>
    
    <div class="metric function-stats">
        <h3>Function Statistics</h3>
        <table>
            <tr>
                <th>Function</th>
                <th>Calls</th>
                <th>Total Duration (ns)</th>
                <th>Avg Duration (ns)</th>
                <th>Min Duration (ns)</th>
                <th>Max Duration (ns)</th>
            </tr>`
	
	for funcName, funcStats := range stats.FunctionStats {
		html += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%d</td>
                <td>%d</td>
                <td>%.2f</td>
                <td>%d</td>
                <td>%d</td>
            </tr>`, funcName, funcStats.Count, funcStats.TotalDuration, funcStats.AvgDuration, funcStats.MinDuration, funcStats.MaxDuration)
	}
	
	html += `
        </table>
    </div>
    
    <script>
        // Auto-refresh every 5 seconds
        setTimeout(function() {
            location.reload();
        }, 5000);
    </script>
</body>
</html>`
	
	fmt.Fprint(w, html)
}

// simulateWorkload simulates application workload for testing
func simulateWorkload(profiler *AdvancedProfiler) {
	for i := 0; i < 1000; i++ {
		start := time.Now()
		
		// Simulate some work
		time.Sleep(time.Microsecond * 100)
		
		duration := time.Since(start)
		profiler.RecordEvent(fmt.Sprintf("app.function_%d", i%10), duration, "application")
		
		if i%100 == 0 {
			time.Sleep(time.Millisecond * 10) // Simulate periodic heavy work
		}
	}
}

func main() {
	fmt.Println("Advanced eBPF-like Profiler Demo")
	fmt.Println("================================")
	
	// Create profiler with 10000 event buffer
	profiler := NewAdvancedProfiler(10000)
	
	// Start profiling
	profiler.Start()
	defer profiler.Stop()
	
	// Setup HTTP endpoints
	http.HandleFunc("/", profiler.dashboardHandler)
	http.HandleFunc("/stats", profiler.statsHandler)
	http.HandleFunc("/events", profiler.eventsHandler)
	
	// Start web server in background
	go func() {
		fmt.Println("Starting web server on :8084")
		fmt.Println("Dashboard: http://localhost:8084")
		fmt.Println("Stats API: http://localhost:8084/stats")
		fmt.Println("Events API: http://localhost:8084/events")
		log.Fatal(http.ListenAndServe(":8084", nil))
	}()
	
	// Wait for server to start
	time.Sleep(time.Second)
	
	// Simulate workload
	fmt.Println("\nSimulating application workload...")
	go simulateWorkload(profiler)
	
	// Run for demonstration
	time.Sleep(10 * time.Second)
	
	// Display final statistics
	fmt.Println("\n=== Final Profiling Statistics ===")
	stats := profiler.GetStats()
	fmt.Printf("Events Collected: %d\n", stats.EventsCollected)
	fmt.Printf("Events/Second: %.2f\n", stats.EventsPerSecond)
	fmt.Printf("Profiling Overhead: %.4f%%\n", stats.ProfilingOverhead)
	fmt.Printf("CPU Usage: %.2f%%\n", stats.SystemMetrics.CPUUsage)
	fmt.Printf("Memory Usage: %d bytes\n", stats.SystemMetrics.MemoryUsage)
	fmt.Printf("Goroutines: %d\n", stats.SystemMetrics.Goroutines)
	
	fmt.Println("\n=== Top Function Statistics ===")
	for funcName, funcStats := range stats.FunctionStats {
		if funcStats.Count > 10 { // Only show functions with significant activity
			fmt.Printf("Function: %s\n", funcName)
			fmt.Printf("  Calls: %d\n", funcStats.Count)
			fmt.Printf("  Avg Duration: %.2f ns\n", funcStats.AvgDuration)
			fmt.Printf("  Total Duration: %d ns\n", funcStats.TotalDuration)
			fmt.Println()
		}
	}
	
	fmt.Println("\nProfiler demo completed. Check the web dashboard for real-time monitoring.")
	fmt.Println("Press Ctrl+C to exit.")
	
	// Keep server running
	select {}
}