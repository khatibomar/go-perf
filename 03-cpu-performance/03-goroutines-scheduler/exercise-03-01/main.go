package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/trace"
	"time"

	"github.com/go-perf/cpu-performance/goroutines-scheduler/exercise-03-01/analysis"
	"github.com/go-perf/cpu-performance/goroutines-scheduler/exercise-03-01/workloads"
)

// Configuration holds the application configuration
type Configuration struct {
	WorkloadType   string
	Duration       time.Duration
	Goroutines     int
	GOMAXPROCS     int
	TraceFile      string
	AnalyzeOnly    bool
	Verbose        bool
	ReportFile     string
	Visualize      bool
}

// parseFlags parses command line flags and returns configuration
func parseFlags() *Configuration {
	config := &Configuration{}
	
	flag.StringVar(&config.WorkloadType, "workload", "cpu", "Workload type: cpu, io, mixed, stress")
	flag.DurationVar(&config.Duration, "duration", 10*time.Second, "Duration to run workload")
	flag.IntVar(&config.Goroutines, "goroutines", runtime.NumCPU()*2, "Number of goroutines to spawn")
	flag.IntVar(&config.GOMAXPROCS, "gomaxprocs", runtime.NumCPU(), "GOMAXPROCS value")
	flag.StringVar(&config.TraceFile, "trace", "", "Output trace file (empty = no tracing)")
	flag.BoolVar(&config.AnalyzeOnly, "analyze", false, "Only analyze existing trace file")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&config.ReportFile, "report", "scheduler_report.md", "Output report file")
	flag.BoolVar(&config.Visualize, "visualize", false, "Generate visualizations")
	
	flag.Parse()
	return config
}

// printSystemInfo displays system and runtime information
func printSystemInfo(config *Configuration) {
	fmt.Printf("=== Go Scheduler Analysis Tool ===\n")
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(config.GOMAXPROCS))
	fmt.Printf("Workload: %s\n", config.WorkloadType)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Goroutines: %d\n", config.Goroutines)
	if config.TraceFile != "" {
		fmt.Printf("Trace File: %s\n", config.TraceFile)
	}
	fmt.Printf("Report File: %s\n", config.ReportFile)
	fmt.Println("=================================")
}

// startTracing begins execution tracing if configured
func startTracing(traceFile string) (*os.File, error) {
	if traceFile == "" {
		return nil, nil
	}
	
	f, err := os.Create(traceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace file: %w", err)
	}
	
	if err := trace.Start(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to start tracing: %w", err)
	}
	
	return f, nil
}

// stopTracing stops execution tracing
func stopTracing(f *os.File) {
	if f == nil {
		return
	}
	
	trace.Stop()
	f.Close()
}

// runWorkload executes the specified workload type
func runWorkload(ctx context.Context, config *Configuration) error {
	switch config.WorkloadType {
	case "cpu":
		return workloads.RunCPUBoundWorkload(ctx, config.Goroutines, config.Verbose)
	case "io":
		return workloads.RunIOBoundWorkload(ctx, config.Goroutines, config.Verbose)
	case "mixed":
		return workloads.RunMixedWorkload(ctx, config.Goroutines, config.Verbose)
	case "stress":
		return workloads.RunSchedulerStress(ctx, config.Goroutines, config.Verbose)
	default:
		return fmt.Errorf("unknown workload type: %s", config.WorkloadType)
	}
}

// collectMetrics gathers runtime and scheduler metrics
func collectMetrics(config *Configuration) *analysis.SchedulerMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &analysis.SchedulerMetrics{
		Timestamp:     time.Now(),
		Goroutines:    runtime.NumGoroutine(),
		GOMAXPROCS:    runtime.GOMAXPROCS(0),
		NumCPU:        runtime.NumCPU(),
		MemStats:      memStats,
		WorkloadType:  config.WorkloadType,
		Duration:      config.Duration,
		TargetGoroutines: config.Goroutines,
	}
}

// runAnalysis performs scheduler analysis and generates reports
func runAnalysis(config *Configuration, metrics *analysis.SchedulerMetrics) error {
	analyzer := analysis.NewTraceAnalyzer(config.Verbose)
	
	// Analyze trace file if provided
	if config.TraceFile != "" {
		if err := analyzer.AnalyzeTraceFile(config.TraceFile); err != nil {
			return fmt.Errorf("failed to analyze trace: %w", err)
		}
	}
	
	// Generate performance report
	reporter := analysis.NewReporter(config.ReportFile)
	if err := reporter.GenerateReport(metrics, analyzer.GetResults()); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}
	
	// Generate visualizations if requested
	if config.Visualize {
		visualizer := analysis.NewVisualizer()
		if err := visualizer.GenerateSchedulerVisualizations(metrics, analyzer.GetResults()); err != nil {
			return fmt.Errorf("failed to generate visualizations: %w", err)
		}
	}
	
	return nil
}

// monitorScheduler continuously monitors scheduler metrics during workload execution
func monitorScheduler(ctx context.Context, config *Configuration, collector *analysis.MetricsCollector) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := collectMetrics(config)
			collector.Record(metrics)
			
			if config.Verbose {
				fmt.Printf("\rGoroutines: %d, CPU: %.1f%%, Mem: %s",
					metrics.Goroutines,
					float64(metrics.MemStats.Sys)/1024/1024,
					formatBytes(metrics.MemStats.Alloc))
			}
		}
	}
}

// formatBytes formats byte count as human readable string
func formatBytes(bytes uint64) string {
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

// showUsage displays usage information and examples
func showUsage() {
	fmt.Printf(`Go Scheduler Analysis Tool

Usage: %s [options]

Options:
`, os.Args[0])
	flag.PrintDefaults()
	fmt.Printf(`
Workload Types:
  cpu    - CPU-intensive computations (prime generation, matrix ops)
  io     - I/O-bound operations (file/network operations, sleeps)
  mixed  - Combination of CPU and I/O operations
  stress - Scheduler stress testing with rapid goroutine creation

Examples:
  # Run CPU workload with tracing
  %s -workload=cpu -duration=30s -trace=cpu.trace -goroutines=8
  
  # Run I/O workload with custom GOMAXPROCS
  %s -workload=io -gomaxprocs=4 -duration=20s -verbose
  
  # Analyze existing trace file
  %s -analyze -trace=existing.trace -report=analysis.md
  
  # Run mixed workload with visualizations
  %s -workload=mixed -duration=15s -visualize -report=mixed_report.md
  
  # Scheduler stress test
  %s -workload=stress -goroutines=1000 -duration=10s -trace=stress.trace

Output Files:
  - Trace files can be viewed with: go tool trace <file>
  - Reports are generated in Markdown format
  - Visualizations are saved as SVG files
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func main() {
	// Parse command line arguments
	config := parseFlags()
	
	// Show usage if help requested
	if len(os.Args) == 1 {
		showUsage()
		return
	}
	
	// Handle analyze-only mode
	if config.AnalyzeOnly {
		if config.TraceFile == "" {
			log.Fatal("Trace file required for analysis mode")
		}
		
		metrics := collectMetrics(config)
		if err := runAnalysis(config, metrics); err != nil {
			log.Fatalf("Analysis failed: %v", err)
		}
		
		fmt.Printf("Analysis complete. Report saved to: %s\n", config.ReportFile)
		return
	}
	
	// Display system information
	printSystemInfo(config)
	
	// Start tracing if configured
	traceFile, err := startTracing(config.TraceFile)
	if err != nil {
		log.Fatalf("Failed to start tracing: %v", err)
	}
	defer stopTracing(traceFile)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()
	
	// Initialize metrics collector
	collector := analysis.NewMetricsCollector()
	
	// Start monitoring goroutine
	go monitorScheduler(ctx, config, collector)
	
	// Record initial metrics
	initialMetrics := collectMetrics(config)
	collector.Record(initialMetrics)
	
	fmt.Printf("\nStarting %s workload with %d goroutines...\n", 
		config.WorkloadType, config.Goroutines)
	
	start := time.Now()
	
	// Run the workload
	if err := runWorkload(ctx, config); err != nil {
		log.Fatalf("Workload execution failed: %v", err)
	}
	
	elapsed := time.Since(start)
	
	// Record final metrics
	finalMetrics := collectMetrics(config)
	collector.Record(finalMetrics)
	
	if config.Verbose {
		fmt.Println() // New line after monitoring output
	}
	
	fmt.Printf("\nWorkload completed in %v\n", elapsed)
	fmt.Printf("Initial goroutines: %d\n", initialMetrics.Goroutines)
	fmt.Printf("Final goroutines: %d\n", finalMetrics.Goroutines)
	fmt.Printf("Peak goroutines: %d\n", collector.GetPeakGoroutines())
	
	// Stop tracing before analysis
	stopTracing(traceFile)
	traceFile = nil
	
	// Run analysis and generate reports
	if err := runAnalysis(config, finalMetrics); err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}
	
	fmt.Printf("\nAnalysis complete!\n")
	fmt.Printf("Report saved to: %s\n", config.ReportFile)
	
	if config.TraceFile != "" {
		fmt.Printf("Trace file: %s\n", config.TraceFile)
		fmt.Printf("View trace with: go tool trace %s\n", config.TraceFile)
	}
	
	if config.Visualize {
		fmt.Printf("Visualizations saved to: visualizations/\n")
	}
	
	// Display summary statistics
	stats := collector.GetSummaryStats()
	fmt.Printf("\n=== Summary Statistics ===\n")
	fmt.Printf("Average goroutines: %.1f\n", stats.AvgGoroutines)
	fmt.Printf("Peak goroutines: %d\n", stats.PeakGoroutines)
	fmt.Printf("Memory allocated: %s\n", formatBytes(stats.TotalAllocated))
	fmt.Printf("GC cycles: %d\n", stats.GCCycles)
	fmt.Printf("Samples collected: %d\n", stats.SampleCount)
}