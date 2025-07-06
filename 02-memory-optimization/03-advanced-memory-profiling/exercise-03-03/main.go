package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/analyzer"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/patterns"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/reporting"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/workloads"
)

// Configuration holds all command-line options
type Configuration struct {
	Mode         string
	Pattern      string
	Duration     time.Duration
	Interval     time.Duration
	OutputDir    string
	InputDir     string
	Format       string
	Rate         string
	Period       time.Duration
	Amplitude    string
	Baseline     string
	Optimized    string
	Alerts       bool
	Recommend    bool
	Verbose      bool
	Threshold    float64
	SamplingRate time.Duration
}

func main() {
	config := parseFlags()

	if config.Verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, cleaning up...")
		cancel()
	}()

	switch config.Mode {
	case "analyze":
		runAnalysis(ctx, config)
	case "simulate":
		runSimulation(ctx, config)
	case "monitor":
		runMonitoring(ctx, config)
	case "report":
		generateReport(config)
	case "optimize":
		runOptimization(config)
	case "compare":
		runComparison(config)
	default:
		log.Fatalf("Unknown mode: %s", config.Mode)
	}
}

func parseFlags() *Configuration {
	config := &Configuration{}

	flag.StringVar(&config.Mode, "mode", "analyze", "Operation mode: analyze, simulate, monitor, report, optimize, compare")
	flag.StringVar(&config.Pattern, "pattern", "mixed", "Pattern type for simulation: burst, growth, cyclic, mixed")
	flag.DurationVar(&config.Duration, "duration", 5*time.Minute, "Analysis or simulation duration")
	flag.DurationVar(&config.Interval, "interval", time.Second, "Monitoring interval")
	flag.StringVar(&config.OutputDir, "output", "./results", "Output directory")
	flag.StringVar(&config.InputDir, "input", "", "Input directory for analysis")
	flag.StringVar(&config.Format, "format", "html", "Report format: text, html, json")
	flag.StringVar(&config.Rate, "rate", "1MB/s", "Allocation rate for growth simulation")
	flag.DurationVar(&config.Period, "period", 30*time.Second, "Period for cyclic patterns")
	flag.StringVar(&config.Amplitude, "amplitude", "10MB", "Amplitude for cyclic patterns")
	flag.StringVar(&config.Baseline, "baseline", "", "Baseline results directory for comparison")
	flag.StringVar(&config.Optimized, "optimized", "", "Optimized results directory for comparison")
	flag.BoolVar(&config.Alerts, "alerts", false, "Enable real-time alerts")
	flag.BoolVar(&config.Recommend, "recommendations", false, "Generate optimization recommendations")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.Float64Var(&config.Threshold, "threshold", 2.0, "Anomaly detection threshold (standard deviations)")
	flag.DurationVar(&config.SamplingRate, "sampling-rate", 100*time.Millisecond, "Metrics sampling rate")

	flag.Parse()

	return config
}

func runAnalysis(ctx context.Context, config *Configuration) {
	log.Printf("Starting allocation pattern analysis for %v", config.Duration)

	// Initialize analyzer
	analyzerConfig := &analyzer.Config{
		SamplingRate:     config.SamplingRate,
		AnomalyThreshold: config.Threshold,
		OutputDir:        config.OutputDir,
		Verbose:          config.Verbose,
	}

	allocationAnalyzer := analyzer.NewAllocationTracker(analyzerConfig)
	patternDetector := patterns.NewPatternDetector(&patterns.Config{
		WindowSize:       100,
		AnomalyThreshold: config.Threshold,
		MinPatternLength: 10,
	})
	hotspotAnalyzer := analyzer.NewHotspotAnalyzer(analyzerConfig)

	// Start analysis
	if err := allocationAnalyzer.Start(ctx); err != nil {
		log.Fatalf("Failed to start allocation tracker: %v", err)
	}

	if err := hotspotAnalyzer.Start(ctx); err != nil {
		log.Fatalf("Failed to start hotspot analyzer: %v", err)
	}

	// Run analysis for specified duration
	analysisCtx, analysisCancel := context.WithTimeout(ctx, config.Duration)
	defer analysisCancel()

	ticker := time.NewTicker(config.SamplingRate)
	defer ticker.Stop()

	for {
		select {
		case <-analysisCtx.Done():
			log.Println("Analysis completed")
			return
		case <-ticker.C:
			// Collect metrics
			metrics := allocationAnalyzer.GetCurrentMetrics()
			patternDetector.AddDataPoint(metrics)

			// Check for patterns
			if detectedPatterns := patternDetector.DetectPatterns(); len(detectedPatterns) > 0 {
				for _, pattern := range detectedPatterns {
					log.Printf("Detected pattern: %s (confidence: %.2f)", pattern.Type, pattern.Confidence)
					if err := savePattern(config.OutputDir, &pattern); err != nil {
						log.Printf("Failed to save pattern: %v", err)
					}
				}
			}

			// Check for hotspots
			if hotspots := hotspotAnalyzer.GetHotspots(); len(hotspots) > 0 {
				for _, hotspot := range hotspots {
					log.Printf("Hotspot detected: %s (rate: %.2f allocs/s)", hotspot.Location, hotspot.AllocationRate)
				}
			}
		}
	}
}

func runSimulation(ctx context.Context, config *Configuration) {
	log.Printf("Starting %s pattern simulation for %v", config.Pattern, config.Duration)

	var workload workloads.Workload
	var err error

	switch config.Pattern {
	case "burst":
		workload, err = workloads.NewBurstWorkload(&workloads.BurstConfig{
			BurstSize:     1000,
			BurstInterval: 5 * time.Second,
			ObjectSize:    1024,
		})
	case "growth":
		workload, err = workloads.NewSteadyWorkload(&workloads.SteadyConfig{
			InitialRate: 100,
			GrowthRate:  1.1,
			ObjectSize:  512,
		})
	case "cyclic":
		workload, err = workloads.NewCyclicWorkload(&workloads.CyclicConfig{
			Period:     config.Period,
			Amplitude:  1000,
			BaseRate:   100,
			ObjectSize: 256,
		})
	case "mixed":
		workload, err = workloads.NewMixedWorkload(&workloads.MixedConfig{
			Patterns: []string{"burst", "growth", "cyclic"},
			Weights:  []float64{0.3, 0.4, 0.3},
		})
	default:
		log.Fatalf("Unknown pattern: %s", config.Pattern)
	}

	if err != nil {
		log.Fatalf("Failed to create workload: %v", err)
	}

	// Run simulation
	simCtx, simCancel := context.WithTimeout(ctx, config.Duration)
	defer simCancel()

	if err := workload.Run(simCtx); err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	log.Println("Simulation completed")
}

func runMonitoring(ctx context.Context, config *Configuration) {
	log.Printf("Starting real-time monitoring (interval: %v)", config.Interval)

	analyzerConfig := &analyzer.Config{
		SamplingRate:     config.Interval,
		AnomalyThreshold: config.Threshold,
		OutputDir:        config.OutputDir,
		Verbose:          config.Verbose,
	}

	allocationAnalyzer := analyzer.NewAllocationTracker(analyzerConfig)
	patternDetector := patterns.NewPatternDetector(&patterns.Config{
		WindowSize:       50,
		AnomalyThreshold: config.Threshold,
		MinPatternLength: 5,
	})

	if err := allocationAnalyzer.Start(ctx); err != nil {
		log.Fatalf("Failed to start allocation tracker: %v", err)
	}

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Monitoring stopped")
			return
		case <-ticker.C:
			metrics := allocationAnalyzer.GetCurrentMetrics()
			patternDetector.AddDataPoint(metrics)

			if config.Verbose {
				log.Printf("Allocation rate: %.2f MB/s, Objects: %d/s",
					float64(metrics.BytesPerSecond)/1024/1024,
					metrics.ObjectsPerSecond)
			}

			if config.Alerts {
				if anomalies := patternDetector.DetectAnomalies(); len(anomalies) > 0 {
					for _, anomaly := range anomalies {
						log.Printf("ALERT: %s anomaly detected (severity: %.2f)",
							anomaly.Type, anomaly.Severity)
					}
				}
			}
		}
	}
}

func generateReport(config *Configuration) {
	log.Printf("Generating %s report from %s", config.Format, config.InputDir)

	if config.InputDir == "" {
		log.Fatal("Input directory is required for report generation")
	}

	reporter := reporting.NewPatternReporter(&reporting.Config{
		InputDir:  config.InputDir,
		OutputDir: config.OutputDir,
		Format:    config.Format,
		Verbose:   config.Verbose,
	})

	if err := reporter.GenerateReport(); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	log.Printf("Report generated in %s", config.OutputDir)
}

func runOptimization(config *Configuration) {
	log.Printf("Analyzing optimization opportunities from %s", config.InputDir)

	if config.InputDir == "" {
		log.Fatal("Input directory is required for optimization analysis")
	}

	advisor := reporting.NewOptimizationAdvisor(&reporting.AdvisorConfig{
		InputDir:  config.InputDir,
		OutputDir: config.OutputDir,
		Verbose:   config.Verbose,
	})

	recommendations, err := advisor.AnalyzeOptimizations()
	if err != nil {
		log.Fatalf("Failed to analyze optimizations: %v", err)
	}

	log.Printf("Generated %d optimization recommendations", len(recommendations))
	for i, rec := range recommendations {
		log.Printf("Recommendation %d: %s (impact: %s)", i+1, rec.Description, rec.Impact)
	}

	if config.Recommend {
		if err := advisor.SaveRecommendations(recommendations); err != nil {
			log.Printf("Failed to save recommendations: %v", err)
		}
	}
}

func runComparison(config *Configuration) {
	log.Printf("Comparing baseline (%s) vs optimized (%s)", config.Baseline, config.Optimized)

	if config.Baseline == "" || config.Optimized == "" {
		log.Fatal("Both baseline and optimized directories are required for comparison")
	}

	comparator := reporting.NewAnalysisComparator(&reporting.ComparatorConfig{
		BaselineDir: config.Baseline,
		CurrentDir:  config.Optimized,
		OutputDir:   config.OutputDir,
		Verbose:     config.Verbose,
	})

	comparison, err := comparator.Compare()
	if err != nil {
		log.Fatalf("Failed to perform comparison: %v", err)
	}

	log.Printf("Comparison completed:")
	if comparison.MetricsChange != nil {
		log.Printf("  Allocation rate change: %.2f%%", comparison.MetricsChange.AllocationRateChange)
		log.Printf("  Heap size change: %.2f%%", comparison.MetricsChange.HeapSizeChange)
		log.Printf("  GC frequency change: %.2f%%", comparison.MetricsChange.GCFrequencyChange)
	}
	log.Printf("  Overall score: %.1f/100", comparison.OverallScore)

	if err := comparator.SaveComparison(comparison); err != nil {
		log.Printf("Failed to save comparison: %v", err)
	}
}

func savePattern(outputDir string, pattern *patterns.DetectedPattern) error {
	patternsDir := filepath.Join(outputDir, "patterns")
	if err := os.MkdirAll(patternsDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(patternsDir, fmt.Sprintf("pattern_%s_%d.json", pattern.Type, time.Now().Unix()))
	return pattern.SaveToFile(filename)
}