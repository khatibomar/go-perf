package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"
)

// ProfilingConfig holds profiling configuration
type ProfilingConfig struct {
	CPUProfile     string
	MemProfile     string
	HTTPAddr       string
	Duration       time.Duration
	SampleRate     int
	EnableTrace    bool
	EnableBlock    bool
	EnableMutex    bool
	OutputDir      string
}

// ProfilingManager manages all profiling operations
type ProfilingManager struct {
	config     ProfilingConfig
	cpuFile    *os.File
	memFile    *os.File
	traceFile  *os.File
	blockFile  *os.File
	mutexFile  *os.File
	startTime  time.Time
	isRunning  bool
	stats      ProfilingStats
}

// ProfilingStats tracks profiling statistics
type ProfilingStats struct {
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	CPUProfileSize   int64
	MemProfileSize   int64
	TraceProfileSize int64
	BlockProfileSize int64
	MutexProfileSize int64
	Goroutines       int
	MemoryUsage      runtime.MemStats
	GCStats          runtime.MemStats
}

// NewProfilingManager creates a new profiling manager
func NewProfilingManager(config ProfilingConfig) *ProfilingManager {
	return &ProfilingManager{
		config: config,
		stats:  ProfilingStats{},
	}
}

// setupProfiling initializes profiling based on configuration


// setupCPUProfiling initializes CPU profiling
func (pm *ProfilingManager) setupCPUProfiling() error {
	filename := pm.config.CPUProfile
	if pm.config.OutputDir != "" {
		filename = pm.config.OutputDir + "/" + filename
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create CPU profile file: %v", err)
	}

	pm.cpuFile = file
	return nil
}

// setupMemoryProfiling initializes memory profiling
func (pm *ProfilingManager) setupMemoryProfiling() error {
	filename := pm.config.MemProfile
	if pm.config.OutputDir != "" {
		filename = pm.config.OutputDir + "/" + filename
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create memory profile file: %v", err)
	}

	pm.memFile = file
	return nil
}

// setupTraceProfiling initializes trace profiling
func (pm *ProfilingManager) setupTraceProfiling() error {
	filename := "trace.out"
	if pm.config.OutputDir != "" {
		filename = pm.config.OutputDir + "/" + filename
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create trace file: %v", err)
	}

	pm.traceFile = file
	return nil
}

// setupBlockProfiling initializes block profiling
func (pm *ProfilingManager) setupBlockProfiling() error {
	filename := "block.prof"
	if pm.config.OutputDir != "" {
		filename = pm.config.OutputDir + "/" + filename
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create block profile file: %v", err)
	}

	pm.blockFile = file
	return nil
}

// setupMutexProfiling initializes mutex profiling
func (pm *ProfilingManager) setupMutexProfiling() error {
	filename := "mutex.prof"
	if pm.config.OutputDir != "" {
		filename = pm.config.OutputDir + "/" + filename
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create mutex profile file: %v", err)
	}

	pm.mutexFile = file
	return nil
}

// setupHTTPProfiling starts the HTTP profiling server
func (pm *ProfilingManager) setupHTTPProfiling() {
	fmt.Printf("Starting HTTP profiling server on %s\n", pm.config.HTTPAddr)
	fmt.Printf("Available endpoints:\n")
	fmt.Printf("  http://%s/debug/pprof/          - Index page\n", pm.config.HTTPAddr)
	fmt.Printf("  http://%s/debug/pprof/profile   - CPU profile\n", pm.config.HTTPAddr)
	fmt.Printf("  http://%s/debug/pprof/heap      - Heap profile\n", pm.config.HTTPAddr)
	fmt.Printf("  http://%s/debug/pprof/goroutine - Goroutine profile\n", pm.config.HTTPAddr)
	fmt.Printf("  http://%s/debug/pprof/block     - Block profile\n", pm.config.HTTPAddr)
	fmt.Printf("  http://%s/debug/pprof/mutex     - Mutex profile\n", pm.config.HTTPAddr)
	fmt.Printf("  http://%s/debug/pprof/trace     - Trace profile\n", pm.config.HTTPAddr)

	log.Fatal(http.ListenAndServe(pm.config.HTTPAddr, nil))
}

// StartProfiling begins profiling session
func (pm *ProfilingManager) StartProfiling() error {
	if pm.isRunning {
		return fmt.Errorf("profiling is already running")
	}

	pm.startTime = time.Now()
	pm.stats.StartTime = pm.startTime
	pm.isRunning = true

	// Start CPU profiling
	if pm.cpuFile != nil {
		if err := pprof.StartCPUProfile(pm.cpuFile); err != nil {
			return fmt.Errorf("could not start CPU profile: %v", err)
		}
		fmt.Printf("CPU profiling started, writing to %s\n", pm.cpuFile.Name())
	}

	// Start trace profiling
	if pm.traceFile != nil {
		if err := trace.Start(pm.traceFile); err != nil {
			return fmt.Errorf("could not start trace: %v", err)
		}
		fmt.Printf("Trace profiling started, writing to %s\n", pm.traceFile.Name())
	}

	// Collect initial runtime stats
	pm.collectRuntimeStats()

	fmt.Printf("Profiling session started at %s\n", pm.startTime.Format(time.RFC3339))
	return nil
}

// StopProfiling ends profiling session
func (pm *ProfilingManager) StopProfiling() error {
	if !pm.isRunning {
		return fmt.Errorf("profiling is not running")
	}

	pm.stats.EndTime = time.Now()
	pm.stats.Duration = pm.stats.EndTime.Sub(pm.stats.StartTime)
	pm.isRunning = false

	// Stop CPU profiling
	if pm.cpuFile != nil {
		pprof.StopCPUProfile()
		pm.cpuFile.Close()
		if stat, err := os.Stat(pm.cpuFile.Name()); err == nil {
			pm.stats.CPUProfileSize = stat.Size()
		}
		fmt.Printf("CPU profiling stopped, profile saved to %s\n", pm.cpuFile.Name())
	}

	// Stop trace profiling
	if pm.traceFile != nil {
		trace.Stop()
		pm.traceFile.Close()
		if stat, err := os.Stat(pm.traceFile.Name()); err == nil {
			pm.stats.TraceProfileSize = stat.Size()
		}
		fmt.Printf("Trace profiling stopped, trace saved to %s\n", pm.traceFile.Name())
	}

	// Write memory profile
	if pm.memFile != nil {
		runtime.GC() // Force garbage collection before memory profile
		if err := pprof.WriteHeapProfile(pm.memFile); err != nil {
			return fmt.Errorf("could not write memory profile: %v", err)
		}
		pm.memFile.Close()
		if stat, err := os.Stat(pm.memFile.Name()); err == nil {
			pm.stats.MemProfileSize = stat.Size()
		}
		fmt.Printf("Memory profile saved to %s\n", pm.memFile.Name())
	}

	// Write block profile
	if pm.blockFile != nil {
		if err := pprof.Lookup("block").WriteTo(pm.blockFile, 0); err != nil {
			return fmt.Errorf("could not write block profile: %v", err)
		}
		pm.blockFile.Close()
		if stat, err := os.Stat(pm.blockFile.Name()); err == nil {
			pm.stats.BlockProfileSize = stat.Size()
		}
		fmt.Printf("Block profile saved to %s\n", pm.blockFile.Name())
	}

	// Write mutex profile
	if pm.mutexFile != nil {
		if err := pprof.Lookup("mutex").WriteTo(pm.mutexFile, 0); err != nil {
			return fmt.Errorf("could not write mutex profile: %v", err)
		}
		pm.mutexFile.Close()
		if stat, err := os.Stat(pm.mutexFile.Name()); err == nil {
			pm.stats.MutexProfileSize = stat.Size()
		}
		fmt.Printf("Mutex profile saved to %s\n", pm.mutexFile.Name())
	}

	// Collect final runtime stats
	pm.collectRuntimeStats()

	fmt.Printf("Profiling session completed in %v\n", pm.stats.Duration)
	return nil
}

// collectRuntimeStats collects runtime statistics
func (pm *ProfilingManager) collectRuntimeStats() {
	pm.stats.Goroutines = runtime.NumGoroutine()
	runtime.ReadMemStats(&pm.stats.MemoryUsage)
	// Copy MemStats to GCStats since they have the same type now
	pm.stats.GCStats = pm.stats.MemoryUsage
}

// PrintStats prints detailed profiling statistics
func (pm *ProfilingManager) PrintStats() {
	fmt.Printf("\n=== Profiling Session Statistics ===\n")
	fmt.Printf("Start Time: %s\n", pm.stats.StartTime.Format(time.RFC3339))
	fmt.Printf("End Time: %s\n", pm.stats.EndTime.Format(time.RFC3339))
	fmt.Printf("Duration: %v\n", pm.stats.Duration)
	fmt.Printf("Goroutines: %d\n", pm.stats.Goroutines)

	fmt.Printf("\n=== Profile File Sizes ===\n")
	if pm.stats.CPUProfileSize > 0 {
		fmt.Printf("CPU Profile: %s\n", formatBytes(pm.stats.CPUProfileSize))
	}
	if pm.stats.MemProfileSize > 0 {
		fmt.Printf("Memory Profile: %s\n", formatBytes(pm.stats.MemProfileSize))
	}
	if pm.stats.TraceProfileSize > 0 {
		fmt.Printf("Trace Profile: %s\n", formatBytes(pm.stats.TraceProfileSize))
	}
	if pm.stats.BlockProfileSize > 0 {
		fmt.Printf("Block Profile: %s\n", formatBytes(pm.stats.BlockProfileSize))
	}
	if pm.stats.MutexProfileSize > 0 {
		fmt.Printf("Mutex Profile: %s\n", formatBytes(pm.stats.MutexProfileSize))
	}

	fmt.Printf("\n=== Memory Statistics ===\n")
	fmt.Printf("Allocated Memory: %s\n", formatBytes(int64(pm.stats.MemoryUsage.Alloc)))
	fmt.Printf("Total Allocated: %s\n", formatBytes(int64(pm.stats.MemoryUsage.TotalAlloc)))
	fmt.Printf("System Memory: %s\n", formatBytes(int64(pm.stats.MemoryUsage.Sys)))
	fmt.Printf("Heap Objects: %d\n", pm.stats.MemoryUsage.HeapObjects)
	fmt.Printf("GC Cycles: %d\n", pm.stats.MemoryUsage.NumGC)
	fmt.Printf("GC Pause Total: %v\n", time.Duration(pm.stats.MemoryUsage.PauseTotalNs))

	if pm.stats.MemoryUsage.NumGC > 0 {
		avgPause := time.Duration(pm.stats.MemoryUsage.PauseTotalNs) / time.Duration(pm.stats.MemoryUsage.NumGC)
		fmt.Printf("Average GC Pause: %v\n", avgPause)
	}

	fmt.Printf("\n=== Analysis Commands ===\n")
	if pm.stats.CPUProfileSize > 0 {
		cpuFile := pm.cpuFile.Name()
		fmt.Printf("CPU Profile Analysis:\n")
		fmt.Printf("  go tool pprof %s\n", cpuFile)
		fmt.Printf("  go tool pprof -http=:8080 %s\n", cpuFile)
		fmt.Printf("  go tool pprof -web %s\n", cpuFile)
	}

	if pm.stats.MemProfileSize > 0 {
		memFile := pm.memFile.Name()
		fmt.Printf("Memory Profile Analysis:\n")
		fmt.Printf("  go tool pprof %s\n", memFile)
		fmt.Printf("  go tool pprof -http=:8081 %s\n", memFile)
	}

	if pm.stats.TraceProfileSize > 0 {
		traceFile := pm.traceFile.Name()
		fmt.Printf("Trace Analysis:\n")
		fmt.Printf("  go tool trace %s\n", traceFile)
	}

	fmt.Printf("\n=== Flame Graph Generation ===\n")
	if pm.stats.CPUProfileSize > 0 {
		cpuFile := pm.cpuFile.Name()
		fmt.Printf("Generate flame graph from CPU profile:\n")
		fmt.Printf("  go tool pprof -raw -output=cpu.raw %s\n", cpuFile)
		fmt.Printf("  stackcollapse-go.pl cpu.raw | flamegraph.pl > cpu_flamegraph.svg\n")
		fmt.Printf("\nOr use pprof's built-in flame graph:\n")
		fmt.Printf("  go tool pprof -http=:8080 %s\n", cpuFile)
		fmt.Printf("  # Then navigate to View -> Flame Graph\n")
	}
}

// formatBytes formats byte count as human readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// WriteAllProfiles writes all available profiles to files
func WriteAllProfiles(outputDir string) error {
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	// Write heap profile
	heapFile := "heap.prof"
	if outputDir != "" {
		heapFile = outputDir + "/" + heapFile
	}
	f, err := os.Create(heapFile)
	if err != nil {
		return fmt.Errorf("could not create heap profile: %v", err)
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("could not write heap profile: %v", err)
	}
	fmt.Printf("Heap profile written to %s\n", heapFile)

	// Write goroutine profile
	goroutineFile := "goroutine.prof"
	if outputDir != "" {
		goroutineFile = outputDir + "/" + goroutineFile
	}
	f, err = os.Create(goroutineFile)
	if err != nil {
		return fmt.Errorf("could not create goroutine profile: %v", err)
	}
	defer f.Close()

	if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
		return fmt.Errorf("could not write goroutine profile: %v", err)
	}
	fmt.Printf("Goroutine profile written to %s\n", goroutineFile)

	// Write threadcreate profile
	threadFile := "threadcreate.prof"
	if outputDir != "" {
		threadFile = outputDir + "/" + threadFile
	}
	f, err = os.Create(threadFile)
	if err != nil {
		return fmt.Errorf("could not create threadcreate profile: %v", err)
	}
	defer f.Close()

	if err := pprof.Lookup("threadcreate").WriteTo(f, 0); err != nil {
		return fmt.Errorf("could not write threadcreate profile: %v", err)
	}
	fmt.Printf("Threadcreate profile written to %s\n", threadFile)

	return nil
}

// PrintRuntimeStats prints current runtime statistics
func PrintRuntimeStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\n=== Runtime Statistics ===\n")
	fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
	fmt.Printf("CPU Count: %d\n", runtime.NumCPU())
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	fmt.Printf("\n=== Memory Statistics ===\n")
	fmt.Printf("Allocated: %s\n", formatBytes(int64(m.Alloc)))
	fmt.Printf("Total Allocated: %s\n", formatBytes(int64(m.TotalAlloc)))
	fmt.Printf("System: %s\n", formatBytes(int64(m.Sys)))
	fmt.Printf("Lookups: %d\n", m.Lookups)
	fmt.Printf("Mallocs: %d\n", m.Mallocs)
	fmt.Printf("Frees: %d\n", m.Frees)

	fmt.Printf("\n=== Heap Statistics ===\n")
	fmt.Printf("Heap Allocated: %s\n", formatBytes(int64(m.HeapAlloc)))
	fmt.Printf("Heap System: %s\n", formatBytes(int64(m.HeapSys)))
	fmt.Printf("Heap Idle: %s\n", formatBytes(int64(m.HeapIdle)))
	fmt.Printf("Heap In Use: %s\n", formatBytes(int64(m.HeapInuse)))
	fmt.Printf("Heap Released: %s\n", formatBytes(int64(m.HeapReleased)))
	fmt.Printf("Heap Objects: %d\n", m.HeapObjects)

	fmt.Printf("\n=== GC Statistics ===\n")
	fmt.Printf("GC Cycles: %d\n", m.NumGC)
	fmt.Printf("GC Forced: %d\n", m.NumForcedGC)
	fmt.Printf("GC Pause Total: %v\n", time.Duration(m.PauseTotalNs))
	fmt.Printf("Last GC: %v ago\n", time.Since(time.Unix(0, int64(m.LastGC))))

	if m.NumGC > 0 {
		avgPause := time.Duration(m.PauseTotalNs) / time.Duration(m.NumGC)
		fmt.Printf("Average GC Pause: %v\n", avgPause)
		recentPauses := m.PauseNs[(m.NumGC+255)%256]
		fmt.Printf("Recent GC Pause: %v\n", time.Duration(recentPauses))
	}

	fmt.Printf("\n=== Stack Statistics ===\n")
	fmt.Printf("Stack In Use: %s\n", formatBytes(int64(m.StackInuse)))
	fmt.Printf("Stack System: %s\n", formatBytes(int64(m.StackSys)))

	fmt.Printf("\n=== Other Statistics ===\n")
	fmt.Printf("MSpan In Use: %s\n", formatBytes(int64(m.MSpanInuse)))
	fmt.Printf("MSpan System: %s\n", formatBytes(int64(m.MSpanSys)))
	fmt.Printf("MCache In Use: %s\n", formatBytes(int64(m.MCacheInuse)))
	fmt.Printf("MCache System: %s\n", formatBytes(int64(m.MCacheSys)))
	fmt.Printf("Buck Hash System: %s\n", formatBytes(int64(m.BuckHashSys)))
	fmt.Printf("GC System: %s\n", formatBytes(int64(m.GCSys)))
	fmt.Printf("Other System: %s\n", formatBytes(int64(m.OtherSys)))
}