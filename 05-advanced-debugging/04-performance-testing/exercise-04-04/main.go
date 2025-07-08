package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"exercise-04-04/benchmark"
	"exercise-04-04/capacity"
	"exercise-04-04/sla"
)

func main() {
	var (
		mode = flag.String("mode", "benchmark", "Mode: benchmark, sla, monitor, capacity")
		config = flag.String("config", "./configs/benchmark.json", "Configuration file path")
		output = flag.String("output", "./results", "Output directory for results")
		baseline = flag.String("baseline", "./baseline", "Baseline directory")
		duration = flag.Duration("duration", 5*time.Minute, "Test duration")
		verbose = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	ctx := context.Background()

	// Ensure output directories exist
	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	if err := os.MkdirAll(*baseline, 0755); err != nil {
		log.Fatalf("Failed to create baseline directory: %v", err)
	}

	switch *mode {
	case "benchmark":
		if err := runBenchmark(ctx, *config, *output, *baseline, *duration, *verbose); err != nil {
			log.Fatalf("Benchmark failed: %v", err)
		}
	case "sla":
		if err := runSLAMonitoring(ctx, *config, *output, *verbose); err != nil {
			log.Fatalf("SLA monitoring failed: %v", err)
		}
	case "monitor":
		if err := runContinuousMonitoring(ctx, *config, *output, *verbose); err != nil {
			log.Fatalf("Continuous monitoring failed: %v", err)
		}
	case "capacity":
		if err := runCapacityPlanning(ctx, *config, *output, *baseline, *verbose); err != nil {
			log.Fatalf("Capacity planning failed: %v", err)
		}
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runBenchmark(ctx context.Context, configPath, outputDir, baselineDir string, duration time.Duration, verbose bool) error {
	fmt.Println("🚀 Starting Performance Benchmarking...")

	// Load benchmark configuration
	benchmarkConfig, err := benchmark.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load benchmark config: %w", err)
	}

	// Override duration if specified
	if duration > 0 {
		benchmarkConfig.Duration = duration
	}

	// Create benchmark runner
	runner := benchmark.NewRunner(benchmarkConfig, outputDir, baselineDir)
	runner.SetVerbose(verbose)

	// Run benchmarks
	results, err := runner.RunBenchmarks(ctx)
	if err != nil {
		return fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Save results
	if err := runner.SaveResults(results, "benchmark_results.json"); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	fmt.Println("✅ Benchmarking completed successfully")
	fmt.Printf("📊 Results saved to: %s\n", outputDir)

	return nil
}

func runSLAMonitoring(ctx context.Context, configPath, outputDir string, verbose bool) error {
	fmt.Println("📈 Starting SLA Monitoring...")

	// Load SLA configuration
	slaConfig, err := sla.LoadSLAConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load SLA config: %w", err)
	}

	// Create SLA monitor
	monitor := sla.NewSLAMonitor(slaConfig, outputDir)
	monitor.SetVerbose(verbose)

	// Start monitoring
	if err := monitor.Start(ctx); err != nil {
		return fmt.Errorf("SLA monitoring failed: %w", err)
	}

	fmt.Println("✅ SLA monitoring completed")
	return nil
}

func runContinuousMonitoring(ctx context.Context, configPath, outputDir string, verbose bool) error {
	fmt.Println("🔄 Starting Continuous Performance Monitoring...")

	// Load monitoring configuration
	monitorConfig, err := sla.LoadSLAConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load monitoring config: %w", err)
	}

	// Create continuous monitor
	monitor := sla.NewSLAMonitor(monitorConfig, outputDir)
	monitor.SetVerbose(verbose)

	// Start continuous monitoring
	if err := monitor.Start(ctx); err != nil {
		return fmt.Errorf("continuous monitoring failed: %w", err)
	}

	fmt.Println("✅ Continuous monitoring started")
	return nil
}

func runCapacityPlanning(ctx context.Context, configPath, outputDir, baselineDir string, verbose bool) error {
	fmt.Println("📊 Starting Capacity Planning Analysis...")

	// Load capacity planning configuration
	capacityConfig, err := capacity.LoadPlanningConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load capacity config: %w", err)
	}

	// Create capacity planner
	planner := capacity.NewCapacityPlanner(capacityConfig, outputDir)
	planner.SetVerbose(verbose)

	// Run capacity analysis
	report, err := planner.RunAnalysis(ctx)
	if err != nil {
		return fmt.Errorf("capacity analysis failed: %w", err)
	}

	// Save capacity report
	if err := planner.SaveReport(report, "capacity_report.json"); err != nil {
		return fmt.Errorf("capacity report save failed: %w", err)
	}

	fmt.Println("✅ Capacity planning analysis completed")
	fmt.Printf("📊 Analysis saved to: %s\n", outputDir)

	return nil
}