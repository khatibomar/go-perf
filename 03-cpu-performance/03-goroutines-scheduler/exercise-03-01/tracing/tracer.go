package tracing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"sync"
	"time"
)

// TraceConfig holds configuration for trace collection
type TraceConfig struct {
	OutputDir    string
	Filename     string
	Duration     time.Duration
	BufferSize   int
	AutoAnalysis bool
	Verbose      bool
}

// TraceSession represents an active tracing session
type TraceSession struct {
	config     TraceConfig
	file       *os.File
	filePath   string
	started    bool
	startTime  time.Time
	endTime    time.Time
	mu         sync.Mutex
}

// TraceMetrics contains metrics collected during tracing
type TraceMetrics struct {
	Duration          time.Duration
	GoroutineCount    int
	GoroutineCreated  int64
	GoroutineBlocked  int64
	GoroutineRunnable int64
	ContextSwitches   int64
	GCEvents          int
	HeapSize          uint64
	StackSize         uint64
	CPUUtilization    float64
	SchedulerLatency  time.Duration
}

// NewTraceConfig creates a default trace configuration
func NewTraceConfig() TraceConfig {
	return TraceConfig{
		OutputDir:    "traces",
		Filename:     fmt.Sprintf("scheduler_trace_%d.trace", time.Now().Unix()),
		Duration:     30 * time.Second,
		BufferSize:   1024 * 1024, // 1MB buffer
		AutoAnalysis: true,
		Verbose:      false,
	}
}

// NewTraceSession creates a new tracing session
func NewTraceSession(config TraceConfig) *TraceSession {
	return &TraceSession{
		config: config,
	}
}

// Start begins trace collection
func (ts *TraceSession) Start() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	if ts.started {
		return fmt.Errorf("trace session already started")
	}
	
	// Create output directory
	if err := os.MkdirAll(ts.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create trace file
	ts.filePath = filepath.Join(ts.config.OutputDir, ts.config.Filename)
	file, err := os.Create(ts.filePath)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %w", err)
	}
	ts.file = file
	
	// Start tracing
	if err := trace.Start(ts.file); err != nil {
		ts.file.Close()
		return fmt.Errorf("failed to start trace: %w", err)
	}
	
	ts.started = true
	ts.startTime = time.Now()
	
	if ts.config.Verbose {
		fmt.Printf("Trace started: %s\n", ts.filePath)
		fmt.Printf("Duration: %v\n", ts.config.Duration)
		fmt.Printf("Buffer size: %d bytes\n", ts.config.BufferSize)
	}
	
	return nil
}

// Stop ends trace collection
func (ts *TraceSession) Stop() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	if !ts.started {
		return fmt.Errorf("trace session not started")
	}
	
	// Stop tracing
	trace.Stop()
	ts.endTime = time.Now()
	
	// Close file
	if err := ts.file.Close(); err != nil {
		return fmt.Errorf("failed to close trace file: %w", err)
	}
	
	ts.started = false
	
	if ts.config.Verbose {
		fmt.Printf("Trace stopped: %s\n", ts.filePath)
		fmt.Printf("Actual duration: %v\n", ts.endTime.Sub(ts.startTime))
	}
	
	// Auto-analysis if enabled
	if ts.config.AutoAnalysis {
		if err := ts.analyzeTrace(); err != nil {
			fmt.Printf("Warning: Auto-analysis failed: %v\n", err)
		}
	}
	
	return nil
}

// GetFilePath returns the path to the trace file
func (ts *TraceSession) GetFilePath() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.filePath
}

// IsActive returns whether the trace session is currently active
func (ts *TraceSession) IsActive() bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.started
}

// GetDuration returns the actual duration of the trace
func (ts *TraceSession) GetDuration() time.Duration {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	if ts.started {
		return time.Since(ts.startTime)
	}
	return ts.endTime.Sub(ts.startTime)
}

// RunWithTrace executes a function while collecting trace data
func RunWithTrace(config TraceConfig, workloadFunc func() error) error {
	session := NewTraceSession(config)
	
	// Start tracing
	if err := session.Start(); err != nil {
		return fmt.Errorf("failed to start trace: %w", err)
	}
	
	// Ensure trace is stopped
	defer func() {
		if err := session.Stop(); err != nil {
			fmt.Printf("Warning: Failed to stop trace: %v\n", err)
		}
	}()
	
	// Run workload
	if err := workloadFunc(); err != nil {
		return fmt.Errorf("workload execution failed: %w", err)
	}
	
	return nil
}

// RunWithTimedTrace executes a function for a specific duration while tracing
func RunWithTimedTrace(config TraceConfig, workloadFunc func(context.Context) error) error {
	session := NewTraceSession(config)
	
	// Start tracing
	if err := session.Start(); err != nil {
		return fmt.Errorf("failed to start trace: %w", err)
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()
	
	// Run workload in goroutine
	errorChan := make(chan error, 1)
	go func() {
		errorChan <- workloadFunc(ctx)
	}()
	
	// Wait for completion or timeout
	select {
	case err := <-errorChan:
		if err != nil {
			fmt.Printf("Workload error: %v\n", err)
		}
	case <-ctx.Done():
		if config.Verbose {
			fmt.Printf("Trace duration completed: %v\n", config.Duration)
		}
	}
	
	// Stop tracing
	if err := session.Stop(); err != nil {
		return fmt.Errorf("failed to stop trace: %w", err)
	}
	
	return nil
}

// analyzeTrace performs basic analysis of the collected trace
func (ts *TraceSession) analyzeTrace() error {
	if ts.config.Verbose {
		fmt.Printf("\nAnalyzing trace: %s\n", ts.filePath)
	}
	
	// Get file info
	fileInfo, err := os.Stat(ts.filePath)
	if err != nil {
		return fmt.Errorf("failed to get trace file info: %w", err)
	}
	
	fmt.Printf("\n=== Trace Analysis ===\n")
	fmt.Printf("File: %s\n", ts.filePath)
	fmt.Printf("Size: %d bytes (%.2f MB)\n", fileInfo.Size(), float64(fileInfo.Size())/(1024*1024))
	fmt.Printf("Duration: %v\n", ts.GetDuration())
	fmt.Printf("Start time: %v\n", ts.startTime.Format(time.RFC3339))
	fmt.Printf("End time: %v\n", ts.endTime.Format(time.RFC3339))
	
	// Runtime statistics at trace end
	fmt.Printf("\n=== Runtime Statistics ===\n")
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("NumGoroutine: %d\n", runtime.NumGoroutine())
	
	// Memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Printf("\n=== Memory Statistics ===\n")
	fmt.Printf("Heap Alloc: %d bytes (%.2f MB)\n", memStats.HeapAlloc, float64(memStats.HeapAlloc)/(1024*1024))
	fmt.Printf("Heap Sys: %d bytes (%.2f MB)\n", memStats.HeapSys, float64(memStats.HeapSys)/(1024*1024))
	fmt.Printf("Stack Sys: %d bytes (%.2f MB)\n", memStats.StackSys, float64(memStats.StackSys)/(1024*1024))
	fmt.Printf("GC Runs: %d\n", memStats.NumGC)
	fmt.Printf("Total Alloc: %d bytes (%.2f MB)\n", memStats.TotalAlloc, float64(memStats.TotalAlloc)/(1024*1024))
	
	// Generate analysis commands
	fmt.Printf("\n=== Analysis Commands ===\n")
	fmt.Printf("View trace in browser:\n")
	fmt.Printf("  go tool trace %s\n", ts.filePath)
	fmt.Printf("\nGenerate goroutine analysis:\n")
	fmt.Printf("  go tool trace -http=:8080 %s\n", ts.filePath)
	fmt.Printf("\nExtract specific events:\n")
	fmt.Printf("  go tool trace -pprof=TYPE %s > analysis.pprof\n", ts.filePath)
	fmt.Printf("  (TYPE can be: net, sync, syscall, sched)\n")
	
	return nil
}

// CollectMetrics collects runtime metrics during trace collection
func (ts *TraceSession) CollectMetrics() *TraceMetrics {
	metrics := &TraceMetrics{
		Duration: ts.GetDuration(),
	}
	
	// Goroutine statistics
	metrics.GoroutineCount = runtime.NumGoroutine()
	
	// Memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	metrics.HeapSize = memStats.HeapAlloc
	metrics.StackSize = memStats.StackSys
	metrics.GCEvents = int(memStats.NumGC)
	
	return metrics
}

// TraceWorkload is a helper function to trace a specific workload
func TraceWorkload(name string, workloadFunc func(context.Context) error, duration time.Duration, verbose bool) (string, error) {
	config := NewTraceConfig()
	config.Filename = fmt.Sprintf("%s_trace_%d.trace", name, time.Now().Unix())
	config.Duration = duration
	config.Verbose = verbose
	
	if verbose {
		fmt.Printf("Starting trace for workload: %s\n", name)
	}
	
	err := RunWithTimedTrace(config, workloadFunc)
	if err != nil {
		return "", err
	}
	
	filePath := filepath.Join(config.OutputDir, config.Filename)
	return filePath, nil
}

// GenerateTraceReport creates a detailed report from trace analysis
func GenerateTraceReport(tracePath string, outputPath string) error {
	// Check if trace file exists
	if _, err := os.Stat(tracePath); os.IsNotExist(err) {
		return fmt.Errorf("trace file does not exist: %s", tracePath)
	}
	
	// Create report file
	reportFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer reportFile.Close()
	
	// Get trace file info
	fileInfo, err := os.Stat(tracePath)
	if err != nil {
		return fmt.Errorf("failed to get trace file info: %w", err)
	}
	
	// Write report header
	report := fmt.Sprintf(`# Go Scheduler Trace Analysis Report

Generated: %s
Trace File: %s
Trace Size: %d bytes (%.2f MB)

`, 
		time.Now().Format(time.RFC3339),
		tracePath,
		fileInfo.Size(),
		float64(fileInfo.Size())/(1024*1024))
	
	// Add runtime information
	report += fmt.Sprintf(`## Runtime Configuration

- GOMAXPROCS: %d
- NumCPU: %d
- NumGoroutine: %d

`,
		runtime.GOMAXPROCS(0),
		runtime.NumCPU(),
		runtime.NumGoroutine())
	
	// Add memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	report += fmt.Sprintf(`## Memory Statistics

- Heap Allocated: %d bytes (%.2f MB)
- Heap System: %d bytes (%.2f MB)
- Stack System: %d bytes (%.2f MB)
- GC Cycles: %d
- Total Allocated: %d bytes (%.2f MB)

`,
		memStats.HeapAlloc, float64(memStats.HeapAlloc)/(1024*1024),
		memStats.HeapSys, float64(memStats.HeapSys)/(1024*1024),
		memStats.StackSys, float64(memStats.StackSys)/(1024*1024),
		memStats.NumGC,
		memStats.TotalAlloc, float64(memStats.TotalAlloc)/(1024*1024))
	
	// Add analysis instructions
	report += `## Analysis Instructions

### View Interactive Trace
` + "`" + `bash
go tool trace ` + tracePath + `
` + "`" + `

`
	report += `### Start Trace Server
` + "`" + `bash
go tool trace -http=:8080 ` + tracePath + `
` + "`" + `

`
	report += `### Extract Profiling Data
` + "`" + `bash
# Network blocking profile
go tool trace -pprof=net ` + tracePath + ` > net.pprof

# Synchronization blocking profile
go tool trace -pprof=sync ` + tracePath + ` > sync.pprof

# Syscall blocking profile
go tool trace -pprof=syscall ` + tracePath + ` > syscall.pprof

# Scheduler latency profile
go tool trace -pprof=sched ` + tracePath + ` > sched.pprof
` + "`" + `

`
	
	// Add interpretation guide
	report += `## Trace Interpretation Guide

### Key Metrics to Analyze

1. **Goroutine States**
   - Running: Goroutine is executing on a CPU
   - Runnable: Goroutine is ready but waiting for CPU
   - Blocked: Goroutine is waiting for I/O, channels, or synchronization

2. **Scheduler Events**
   - Goroutine creation and destruction
   - Context switches between goroutines
   - Work stealing between processor queues
   - GC-related scheduling pauses

3. **Performance Indicators**
   - High runnable queue lengths indicate CPU saturation
   - Frequent context switches may indicate contention
   - Long blocking times suggest I/O or synchronization issues
   - GC pause frequency and duration impact overall performance

### Common Patterns

- **CPU-bound workloads**: High CPU utilization, minimal blocking
- **I/O-bound workloads**: Frequent blocking, lower CPU utilization
- **Mixed workloads**: Variable patterns with both CPU and I/O phases
- **Contention issues**: High synchronization blocking times

### Optimization Opportunities

- Adjust GOMAXPROCS based on workload characteristics
- Reduce goroutine creation overhead for short-lived tasks
- Optimize synchronization patterns to reduce contention
- Balance workload distribution across available processors
`
	
	// Write report to file
	if _, err := reportFile.WriteString(report); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}
	
	fmt.Printf("Trace report generated: %s\n", outputPath)
	return nil
}

// CleanupOldTraces removes trace files older than the specified duration
func CleanupOldTraces(traceDir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(traceDir)
	if err != nil {
		return fmt.Errorf("failed to read trace directory: %w", err)
	}
	
	cutoff := time.Now().Add(-maxAge)
	cleaned := 0
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Check if it's a trace file
		if filepath.Ext(entry.Name()) != ".trace" {
			continue
		}
		
		filePath := filepath.Join(traceDir, entry.Name())
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}
		
		if fileInfo.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err == nil {
				cleaned++
			}
		}
	}
	
	fmt.Printf("Cleaned up %d old trace files\n", cleaned)
	return nil
}

// GetTraceStats returns statistics about trace files in a directory
func GetTraceStats(traceDir string) (map[string]interface{}, error) {
	entries, err := os.ReadDir(traceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace directory: %w", err)
	}
	
	stats := map[string]interface{}{
		"directory":    traceDir,
		"total_files":  0,
		"total_size":   int64(0),
		"newest_file": "",
		"oldest_file": "",
		"files":        []map[string]interface{}{},
	}
	
	var newestTime, oldestTime time.Time
	var newestFile, oldestFile string
	files := []map[string]interface{}{}
	
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".trace" {
			continue
		}
		
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}
		
		filePath := filepath.Join(traceDir, entry.Name())
		modTime := fileInfo.ModTime()
		
		// Update newest/oldest tracking
		if newestTime.IsZero() || modTime.After(newestTime) {
			newestTime = modTime
			newestFile = filePath
		}
		if oldestTime.IsZero() || modTime.Before(oldestTime) {
			oldestTime = modTime
			oldestFile = filePath
		}
		
		// Add file info
		files = append(files, map[string]interface{}{
			"name":     entry.Name(),
			"path":     filePath,
			"size":     fileInfo.Size(),
			"modified": modTime.Format(time.RFC3339),
		})
		
		stats["total_files"] = stats["total_files"].(int) + 1
		stats["total_size"] = stats["total_size"].(int64) + fileInfo.Size()
	}
	
	stats["newest_file"] = newestFile
	stats["oldest_file"] = oldestFile
	stats["files"] = files
	
	return stats, nil
}