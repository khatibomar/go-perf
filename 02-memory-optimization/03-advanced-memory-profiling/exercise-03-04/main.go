package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"exercise-03-04/analyzer"
	"exercise-03-04/detector"
	"exercise-03-04/monitor"
	"exercise-03-04/reporting"
	"exercise-03-04/simulator"
)

// Configuration holds command-line options
type Configuration struct {
	// Common options
	Mode     string
	Verbose  bool
	Output   string
	Interval time.Duration
	Duration time.Duration

	// Detection options
	PID              int
	HeapThreshold    float64
	GoroutineThreshold int
	ResourceThreshold  int

	// Simulation options
	LeakType   string
	Rate       string
	Growth     string
	Objects    int
	Intensity  float64

	// Monitoring options
	Port        int
	MetricsPort int
	AlertEmail  string

	// Analysis options
	Input      string
	Format     string
	Baseline   string
	Threshold  float64
}

func main() {
	config := parseFlags()

	switch config.Mode {
	case "detect":
		runDetection(config)
	case "monitor":
		runMonitoring(config)
	case "simulate":
		runSimulation(config)
	case "analyze":
		runAnalysis(config)
	case "report":
		runReporting(config)
	case "dashboard":
		runDashboard(config)
	case "export":
		runExport(config)
	default:
		fmt.Printf("Unknown mode: %s\n", config.Mode)
		os.Exit(1)
	}
}

// parseFlags parses command-line flags
func parseFlags() *Configuration {
	config := &Configuration{}

	// Define flags
	flag.StringVar(&config.Mode, "mode", "detect", "Operation mode: detect, monitor, simulate, analyze, report, dashboard, export")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&config.Output, "output", "leak_data", "Output directory or file")
	flag.DurationVar(&config.Interval, "interval", 10*time.Second, "Monitoring interval")
	flag.DurationVar(&config.Duration, "duration", 5*time.Minute, "Monitoring duration")

	// Detection flags
	flag.IntVar(&config.PID, "pid", 0, "Process ID to monitor (0 for self)")
	flag.Float64Var(&config.HeapThreshold, "heap-threshold", 10.0, "Heap growth threshold (MB/min)")
	flag.IntVar(&config.GoroutineThreshold, "goroutine-threshold", 100, "Goroutine count threshold")
	flag.IntVar(&config.ResourceThreshold, "resource-threshold", 1000, "Resource count threshold")

	// Simulation flags
	flag.StringVar(&config.LeakType, "type", "goroutine", "Leak type: goroutine, memory, cycle, resource")
	flag.StringVar(&config.Rate, "rate", "10/s", "Leak rate (e.g., 10/s, 5/min)")
	flag.StringVar(&config.Growth, "growth", "1MB/s", "Memory growth rate")
	flag.IntVar(&config.Objects, "objects", 1000, "Number of objects for simulation")
	flag.Float64Var(&config.Intensity, "intensity", 1.0, "Leak intensity multiplier")

	// Monitoring flags
	flag.IntVar(&config.Port, "port", 8080, "Dashboard port")
	flag.IntVar(&config.MetricsPort, "metrics-port", 9090, "Metrics export port")
	flag.StringVar(&config.AlertEmail, "alert-email", "", "Email for alerts")

	// Analysis flags
	flag.StringVar(&config.Input, "input", "leak_data", "Input directory for analysis")
	flag.StringVar(&config.Format, "format", "json", "Output format: json, csv, html")
	flag.StringVar(&config.Baseline, "baseline", "", "Baseline data for comparison")
	flag.Float64Var(&config.Threshold, "threshold", 0.05, "Analysis threshold")

	flag.Parse()

	// Handle positional arguments
	args := flag.Args()
	if len(args) > 0 {
		config.Mode = args[0]
	}

	return config
}

// runDetection runs leak detection
func runDetection(config *Configuration) {
	if config.Verbose {
		log.Printf("Starting leak detection (PID: %d)\n", config.PID)
	}

	// Create detectors
	heapAnalyzer := detector.NewHeapAnalyzer(&detector.HeapConfig{
		Interval:  config.Interval,
		Threshold: config.HeapThreshold,
		Verbose:   config.Verbose,
	})

	goroutineDetector := detector.NewGoroutineDetector(&detector.GoroutineConfig{
		Interval:        config.Interval,
		GrowthThreshold: config.GoroutineThreshold,
		Verbose:         config.Verbose,
	})

	resourceDetector := detector.NewResourceDetector(&detector.ResourceConfig{
		Interval:            config.Interval,
		FileHandleThreshold: config.ResourceThreshold,
		NetworkThreshold:    config.ResourceThreshold,
		Verbose:             config.Verbose,
	})

	// Start detection
	if err := heapAnalyzer.Start(); err != nil {
		log.Fatalf("Failed to start heap analyzer: %v", err)
	}
	defer heapAnalyzer.Stop()

	if err := goroutineDetector.Start(); err != nil {
		log.Fatalf("Failed to start goroutine detector: %v", err)
	}
	defer goroutineDetector.Stop()

	if err := resourceDetector.Start(); err != nil {
		log.Fatalf("Failed to start resource detector: %v", err)
	}
	defer resourceDetector.Stop()

	// Run for specified duration
	if config.Verbose {
		log.Printf("Running detection for %v\n", config.Duration)
	}

	timer := time.NewTimer(config.Duration)
	defer timer.Stop()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-timer.C:
		if config.Verbose {
			log.Println("Detection completed")
		}
	case sig := <-sigChan:
		if config.Verbose {
			log.Printf("Received signal %v, stopping detection\n", sig)
		}
	}

	// Save results
	if err := saveDetectionResults(config, heapAnalyzer, goroutineDetector, resourceDetector); err != nil {
		log.Printf("Failed to save results: %v", err)
	}
}

// runMonitoring runs continuous monitoring
func runMonitoring(config *Configuration) {
	if config.Verbose {
		log.Printf("Starting continuous monitoring on port %d\n", config.Port)
	}

	// Create monitoring dashboard
	dashboard := monitor.NewDashboard(&monitor.DashboardConfig{
		Port:           config.Port,
		UpdateInterval: config.Interval,
		Verbose:        config.Verbose,
	})

	// Start dashboard
	if err := dashboard.Start(); err != nil {
		log.Fatalf("Failed to start dashboard: %v", err)
	}
	defer dashboard.Stop()

	if config.Verbose {
		log.Printf("Dashboard available at http://localhost:%d\n", config.Port)
		log.Printf("Metrics available at http://localhost:%d/metrics\n", config.MetricsPort)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	if config.Verbose {
		log.Println("Stopping monitoring")
	}
}

// runSimulation runs leak simulation
func runSimulation(config *Configuration) {
	if config.Verbose {
		log.Printf("Starting %s leak simulation\n", config.LeakType)
	}

	// Parse rate
	rate, err := parseRate(config.Rate)
	if err != nil {
		log.Fatalf("Invalid rate format: %v", err)
	}

	// Create simulator
	simulator := simulator.NewLeakSimulator(&simulator.LeakConfig{
		Type:      simulator.LeakType(config.LeakType),
		Duration:  config.Duration,
		Intensity: int(config.Intensity),
		GrowthRate: rate,
		BurstInterval: 30 * time.Second,
		BurstSize: 5,
		Randomness: 0.2,
		CleanupChance: 0.1,
		Verbose:   config.Verbose,
	})

	// Start simulation
	if err := simulator.Start(); err != nil {
		log.Fatalf("Failed to start simulation: %v", err)
	}
	defer simulator.Stop()

	if config.Verbose {
		log.Printf("Simulation running for %v\n", config.Duration)
	}

	// Run for specified duration
	timer := time.NewTimer(config.Duration)
	defer timer.Stop()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-timer.C:
		if config.Verbose {
			log.Println("Simulation completed")
		}
	case sig := <-sigChan:
		if config.Verbose {
			log.Printf("Received signal %v, stopping simulation\n", sig)
		}
	}

	// Print simulation stats
	stats := simulator.GetStats()
	fmt.Printf("Simulation Statistics:\n")
	fmt.Printf("  Duration: %v\n", stats.Duration)
	fmt.Printf("  Resources Created: %d\n", stats.ResourcesCreated)
	fmt.Printf("  Resources Leaked: %d\n", stats.ResourcesLeaked)
	fmt.Printf("  Peak Resources: %d\n", stats.PeakResources)
}

// runAnalysis runs leak analysis
func runAnalysis(config *Configuration) {
	if config.Verbose {
		log.Printf("Starting analysis of data in %s\n", config.Input)
	}

	// Create analyzer
	analyzerInst := analyzer.NewAnalyzer(&analyzer.AnalyzerConfig{
		OutputDir: config.Output,
		Verbose:   config.Verbose,
	})

	// Run analysis
	results, err := analyzerInst.RunAnalysis()
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Print results
	fmt.Printf("Analysis Results:\n")
	fmt.Printf("  Duration: %v\n", results.Duration)
	if results.HeapAnalysis != nil {
		fmt.Printf("  Heap Analysis: %d samples\n", len(results.HeapAnalysis.Metrics))
	}
	if results.GoroutineAnalysis != nil {
		fmt.Printf("  Goroutine Analysis: %d samples\n", len(results.GoroutineAnalysis.Metrics))
	}
	if results.ResourceAnalysis != nil {
		fmt.Printf("  Resource Analysis: %d samples\n", len(results.ResourceAnalysis.Metrics))
	}
	if results.ReferenceAnalysis != nil {
		fmt.Printf("  Reference Analysis: %d samples\n", len(results.ReferenceAnalysis.Metrics))
	}

	if len(results.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range results.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	// Save detailed results
	if err := analyzerInst.SaveResults(results, "analysis_results.json"); err != nil {
		log.Printf("Failed to save analysis results: %v", err)
	}
}

// runReporting generates leak reports
func runReporting(config *Configuration) {
	if config.Verbose {
		log.Printf("Generating leak report from %s\n", config.Input)
	}

	// Create reporter
	reporter := reporting.NewReporter(&reporting.ReportConfig{
		OutputDir: config.Output,
		Formats:   []string{config.Format},
		Verbose:   config.Verbose,
	})

	// Generate report
	report, err := reporter.GenerateReport(
		nil,
		nil,
		nil,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	// Save report
	if err := reporter.SaveReport(report); err != nil {
		log.Fatalf("Failed to save report: %v", err)
	}

	if config.Verbose {
		log.Printf("Report saved to %s\n", config.Output)
	}

	// Print summary
	fmt.Printf("Leak Report Summary:\n")
	fmt.Printf("  Top Leak Sources: %d\n", len(report.Summary.TopLeakSources))
	fmt.Printf("  Total Insights: %d\n", len(report.Insights))
	fmt.Printf("  Total Recommendations: %d\n", len(report.Recommendations))
	fmt.Printf("  Report Duration: %v\n", report.Duration)
}

// runDashboard starts the monitoring dashboard
func runDashboard(config *Configuration) {
	runMonitoring(config) // Dashboard is part of monitoring
}

// runExport exports metrics data
func runExport(config *Configuration) {
	if config.Verbose {
		log.Printf("Exporting metrics from %s to %s\n", config.Input, config.Output)
	}

	// Create reporter for export
	reporter := reporting.NewReporter(&reporting.ReportConfig{
		OutputDir: config.Output,
		Formats:   []string{config.Format},
		Verbose:   config.Verbose,
	})

	// Load and export data (placeholder)
	report, err := reporter.GenerateReport(
		nil,
		nil,
		nil,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to generate export: %v", err)
	}

	// Save export
	if err := reporter.SaveReport(report); err != nil {
		log.Fatalf("Failed to save export: %v", err)
	}

	if config.Verbose {
		log.Printf("Metrics exported to %s\n", config.Output)
	}
}

// Helper functions

// parseRate parses rate string (e.g., "10/s", "5/min")
func parseRate(rateStr string) (float64, error) {
	parts := strings.Split(rateStr, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid rate format: %s", rateStr)
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid rate value: %s", parts[0])
	}

	unit := parts[1]
	switch unit {
	case "s", "sec", "second":
		return value, nil
	case "m", "min", "minute":
		return value / 60, nil
	case "h", "hour":
		return value / 3600, nil
	default:
		return 0, fmt.Errorf("invalid rate unit: %s", unit)
	}
}

// saveDetectionResults saves detection results to files
func saveDetectionResults(config *Configuration, heapAnalyzer *detector.HeapAnalyzer, goroutineDetector *detector.GoroutineDetector, resourceDetector *detector.ResourceDetector) error {
	if err := os.MkdirAll(config.Output, 0755); err != nil {
		return err
	}

	// Save heap analysis results
	if err := heapAnalyzer.SaveResults(config.Output); err != nil {
		log.Printf("Failed to save heap analysis results: %v", err)
	}

	// Save goroutine detection results
	if err := goroutineDetector.SaveResults(config.Output); err != nil {
		log.Printf("Failed to save goroutine detection results: %v", err)
	}

	// Save resource detection results
	if err := resourceDetector.SaveResults(config.Output); err != nil {
		log.Printf("Failed to save resource detection results: %v", err)
	}

	return nil
}