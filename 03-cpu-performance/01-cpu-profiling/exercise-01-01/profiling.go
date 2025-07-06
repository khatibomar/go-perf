package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Import for side effects - registers pprof handlers
	"os"
	"runtime"
	"runtime/pprof"
)

// ProfilingConfig holds configuration for profiling setup
type ProfilingConfig struct {
	CPUProfile  string // File path for CPU profile output
	HTTPProfile bool   // Enable HTTP profiling endpoint
	ProfilePort string // Port for HTTP profiling server
	Verbose     bool   // Enable verbose logging
}

// setupProfiling configures and starts profiling based on the provided config
// Returns a cleanup function that should be called when profiling is done
func setupProfiling(config ProfilingConfig) func() {
	var cleanupFuncs []func()

	// Setup CPU profiling to file if requested
	if config.CPUProfile != "" {
		cleanup := setupCPUProfiling(config.CPUProfile, config.Verbose)
		if cleanup != nil {
			cleanupFuncs = append(cleanupFuncs, cleanup)
		}
	}

	// Setup HTTP profiling if requested
	if config.HTTPProfile {
		cleanup := setupHTTPProfiling(config.ProfilePort, config.Verbose)
		if cleanup != nil {
			cleanupFuncs = append(cleanupFuncs, cleanup)
		}
	}

	// Return a function that calls all cleanup functions
	return func() {
		for _, cleanup := range cleanupFuncs {
			cleanup()
		}
	}
}

// setupCPUProfiling starts CPU profiling to a file
func setupCPUProfiling(filename string, verbose bool) func() {
	// Create the CPU profile file
	f, err := os.Create(filename)
	if err != nil {
		log.Printf("Error creating CPU profile file %s: %v", filename, err)
		return nil
	}

	// Start CPU profiling
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Printf("Error starting CPU profile: %v", err)
		f.Close()
		return nil
	}

	if verbose {
		fmt.Printf("CPU profiling started, writing to: %s\n", filename)
	}

	// Return cleanup function
	return func() {
		pprof.StopCPUProfile()
		f.Close()
		if verbose {
			fmt.Printf("CPU profiling stopped\n")
		}
	}
}

// setupHTTPProfiling starts the HTTP profiling server
func setupHTTPProfiling(port string, verbose bool) func() {
	// Start HTTP server for profiling in a goroutine
	go func() {
		addr := ":" + port
		if verbose {
			fmt.Printf("Starting HTTP profiling server on %s\n", addr)
			fmt.Printf("Available endpoints:\n")
			fmt.Printf("  http://localhost%s/debug/pprof/          - Index page\n", addr)
			fmt.Printf("  http://localhost%s/debug/pprof/profile   - CPU profile\n", addr)
			fmt.Printf("  http://localhost%s/debug/pprof/heap      - Heap profile\n", addr)
			fmt.Printf("  http://localhost%s/debug/pprof/goroutine - Goroutine profile\n", addr)
			fmt.Printf("  http://localhost%s/debug/pprof/trace     - Execution trace\n", addr)
		}

		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("HTTP profiling server error: %v", err)
		}
	}()

	// Return a no-op cleanup function since HTTP server runs in background
	// In a real application, you might want to implement graceful shutdown
	return func() {
		if verbose {
			fmt.Printf("HTTP profiling server cleanup (note: server continues running)\n")
		}
	}
}

// Additional profiling utilities

// writeHeapProfile writes a heap profile to the specified file
func writeHeapProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create heap profile file %s: %v", filename, err)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("could not write heap profile: %v", err)
	}

	return nil
}

// writeGoroutineProfile writes a goroutine profile to the specified file
func writeGoroutineProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create goroutine profile file %s: %v", filename, err)
	}
	defer f.Close()

	if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
		return fmt.Errorf("could not write goroutine profile: %v", err)
	}

	return nil
}

// writeAllProfiles writes all available profiles to files with the given prefix
func writeAllProfiles(prefix string) {
	profiles := []string{"heap", "goroutine", "threadcreate", "block", "mutex"}

	for _, profileName := range profiles {
		filename := fmt.Sprintf("%s_%s.prof", prefix, profileName)
		f, err := os.Create(filename)
		if err != nil {
			log.Printf("Error creating %s profile file: %v", profileName, err)
			continue
		}

		profile := pprof.Lookup(profileName)
		if profile == nil {
			log.Printf("Profile %s not found", profileName)
			f.Close()
			continue
		}

		if err := profile.WriteTo(f, 0); err != nil {
			log.Printf("Error writing %s profile: %v", profileName, err)
		}

		f.Close()
		fmt.Printf("Written %s profile to %s\n", profileName, filename)
	}
}

// ProfilingStats holds runtime profiling statistics
type ProfilingStats struct {
	Goroutines   int
	CGOCalls     int64
	MemAllocs    uint64
	MemFrees     uint64
	MemSys       uint64
	GCCycles     uint32
	GCPauseTotal uint64
}

// getProfilingStats returns current runtime statistics
func getProfilingStats() ProfilingStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return ProfilingStats{
		Goroutines:   runtime.NumGoroutine(),
		CGOCalls:     runtime.NumCgoCall(),
		MemAllocs:    m.Mallocs,
		MemFrees:     m.Frees,
		MemSys:       m.Sys,
		GCCycles:     m.NumGC,
		GCPauseTotal: m.PauseTotalNs,
	}
}

// printProfilingStats prints current runtime statistics
func printProfilingStats() {
	stats := getProfilingStats()
	fmt.Printf("\nRuntime Statistics:\n")
	fmt.Printf("  Goroutines: %d\n", stats.Goroutines)
	fmt.Printf("  CGO Calls: %d\n", stats.CGOCalls)
	fmt.Printf("  Memory Allocations: %d\n", stats.MemAllocs)
	fmt.Printf("  Memory Frees: %d\n", stats.MemFrees)
	fmt.Printf("  System Memory: %d bytes\n", stats.MemSys)
	fmt.Printf("  GC Cycles: %d\n", stats.GCCycles)
	fmt.Printf("  GC Pause Total: %d ns\n", stats.GCPauseTotal)
}