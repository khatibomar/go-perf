// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"exercise-04-03/chaos"
	"exercise-04-03/stress"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [chaos|stress|combined]")
		os.Exit(1)
	}

	mode := os.Args[1]
	ctx := context.Background()

	switch mode {
	case "chaos":
		runChaosExperiments(ctx)
	case "stress":
		runStressTests(ctx)
	case "combined":
		runCombinedTests(ctx)
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		os.Exit(1)
	}
}

// runChaosExperiments executes chaos engineering experiments
func runChaosExperiments(ctx context.Context) {
	fmt.Println("Starting Chaos Engineering Experiments...")

	// Create chaos engine
	config := chaos.ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := chaos.NewChaosEngine(config)

	// Add CPU stress experiment
	cpuExperiment := chaos.ChaosExperiment{
		Name:        "CPU Stress Test",
		Description: "Test system behavior under high CPU load",
		Hypothesis:  "System should maintain responsiveness under 80% CPU load",
		Duration:    2 * time.Minute,
		SteadyState: chaos.SteadyStateCheck{
			Name: "Response Time Check",
			Check: func(ctx context.Context) (bool, error) {
				// Simulate response time check
				time.Sleep(10 * time.Millisecond)
				return true, nil
			},
			Tolerance: 0.95,
			Interval:  10 * time.Second,
		},
		Perturbations: []chaos.Perturbation{
			chaos.CPUStressPerturbation(80),
		},
		Rollback: chaos.RollbackAction{
			Name: "Stop CPU Stress",
			Action: func(ctx context.Context) error {
				fmt.Println("Rolling back CPU stress...")
				return nil
			},
		},
		SafetyChecks: []chaos.SafetyCheck{
			{
				Name: "System Availability",
				Check: func(ctx context.Context) (bool, string, error) {
					// Simulate availability check
					return true, "System available", nil
				},
				Interval: 5 * time.Second,
				Critical: false,
			},
		},
		Metrics: []chaos.MetricCollector{
			{
				Name: "CPU Usage",
				Collect: func(ctx context.Context) (map[string]float64, error) {
					return map[string]float64{
						"cpu_usage": 75.5,
					}, nil
				},
				Interval: 1 * time.Second,
			},
			{
				Name: "Memory Usage",
				Collect: func(ctx context.Context) (map[string]float64, error) {
					return map[string]float64{
						"memory_usage": 512.0,
					}, nil
				},
				Interval: 1 * time.Second,
			},
		},
	}

	// Add memory stress experiment
	memoryExperiment := chaos.ChaosExperiment{
		Name:        "Memory Stress Test",
		Description: "Test system behavior under high memory pressure",
		Hypothesis:  "System should handle memory pressure gracefully",
		Duration:    90 * time.Second,
		SteadyState: chaos.SteadyStateCheck{
			Name: "Memory Availability Check",
			Check: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			Tolerance: 0.90,
			Interval:  15 * time.Second,
		},
		Perturbations: []chaos.Perturbation{
			chaos.MemoryStressPerturbation(256),
		},
		Rollback: chaos.RollbackAction{
			Name: "Free Memory",
			Action: func(ctx context.Context) error {
				fmt.Println("Rolling back memory stress...")
				return nil
			},
		},
		SafetyChecks: []chaos.SafetyCheck{
			{
				Name: "OOM Protection",
				Check: func(ctx context.Context) (bool, string, error) {
					return true, "Memory usage safe", nil
				},
				Interval: 10 * time.Second,
				Critical: true,
			},
		},
		Metrics: []chaos.MetricCollector{
			{
				Name: "Available Memory",
				Collect: func(ctx context.Context) (map[string]float64, error) {
					return map[string]float64{
						"available_memory": 1024.0,
					}, nil
				},
				Interval: 2 * time.Second,
			},
		},
	}

	// Add experiments to engine
	engine.AddExperiment(cpuExperiment)
	engine.AddExperiment(memoryExperiment)

	// Run experiments
	for _, experiment := range []chaos.ChaosExperiment{cpuExperiment, memoryExperiment} {
		fmt.Printf("\nRunning experiment: %s\n", experiment.Name)
		fmt.Printf("Hypothesis: %s\n", experiment.Hypothesis)
		
		result, err := engine.RunExperiment(ctx, experiment.Name)
		if err != nil {
			log.Printf("Experiment failed: %v", err)
			continue
		}

		// Print results
		printChaosResults(result)
	}
}

// runStressTests executes stress testing scenarios
func runStressTests(ctx context.Context) {
	fmt.Println("Starting Stress Testing...")

	// HTTP stress test configuration
	httpConfig := stress.StressTestConfig{
		Name:             "HTTP Load Test",
		Duration:         3 * time.Minute,
		Concurrency:      50,
		RampUpDuration:   30 * time.Second,
		RampDownDuration: 30 * time.Second,
		TargetRPS:        100,
		MaxRPS:           200,
		ThinkTime:        100 * time.Millisecond,
		Timeout:          10 * time.Second,
		FailureThreshold: 0.05, // 5% error rate threshold
	}

	// CPU stress test configuration
	cpuConfig := stress.StressTestConfig{
		Name:             "CPU Intensive Test",
		Duration:         2 * time.Minute,
		Concurrency:      10,
		RampUpDuration:   20 * time.Second,
		RampDownDuration: 20 * time.Second,
		TargetRPS:        50,
		MaxRPS:           100,
		ThinkTime:        50 * time.Millisecond,
		Timeout:          5 * time.Second,
		FailureThreshold: 0.1, // 10% error rate threshold
	}

	// Memory stress test configuration
	memoryConfig := stress.StressTestConfig{
		Name:             "Memory Intensive Test",
		Duration:         90 * time.Second,
		Concurrency:      20,
		RampUpDuration:   15 * time.Second,
		RampDownDuration: 15 * time.Second,
		TargetRPS:        30,
		MaxRPS:           60,
		ThinkTime:        200 * time.Millisecond,
		Timeout:          8 * time.Second,
		FailureThreshold: 0.15, // 15% error rate threshold
	}

	// Run HTTP stress test
	fmt.Println("\nRunning HTTP stress test...")
	httpGenerator := stress.NewLoadGenerator(httpConfig)
	httpTestFunc := stress.HTTPStressTest("https://httpbin.org/get", "GET", map[string]string{"User-Agent": "StressTest/1.0"}, nil)
	httpResult, err := httpGenerator.RunStressTest(ctx, httpTestFunc)
	if err != nil {
		log.Printf("HTTP stress test failed: %v", err)
	} else {
		printStressResults(httpResult)
	}

	// Run CPU stress test
	fmt.Println("\nRunning CPU stress test...")
	cpuGenerator := stress.NewLoadGenerator(cpuConfig)
	cpuTestFunc := stress.CPUStressTest(1000) // High intensity
	cpuResult, err := cpuGenerator.RunStressTest(ctx, cpuTestFunc)
	if err != nil {
		log.Printf("CPU stress test failed: %v", err)
	} else {
		printStressResults(cpuResult)
	}

	// Run Memory stress test
	fmt.Println("\nRunning Memory stress test...")
	memoryGenerator := stress.NewLoadGenerator(memoryConfig)
	memoryTestFunc := stress.MemoryStressTest(10) // 10MB per request
	memoryResult, err := memoryGenerator.RunStressTest(ctx, memoryTestFunc)
	if err != nil {
		log.Printf("Memory stress test failed: %v", err)
	} else {
		printStressResults(memoryResult)
	}
}

// runCombinedTests executes both chaos and stress tests
func runCombinedTests(ctx context.Context) {
	fmt.Println("Starting Combined Chaos and Stress Testing...")

	// Run chaos experiments first
	fmt.Println("\n=== Phase 1: Chaos Engineering ===")
	runChaosExperiments(ctx)

	// Wait between phases
	fmt.Println("\nWaiting 30 seconds between phases...")
	time.Sleep(30 * time.Second)

	// Run stress tests
	fmt.Println("\n=== Phase 2: Stress Testing ===")
	runStressTests(ctx)

	fmt.Println("\n=== Combined Testing Complete ===")
}

// printChaosResults prints chaos experiment results
func printChaosResults(result *chaos.ExperimentResult) {
	fmt.Printf("\n=== Chaos Experiment Results ===\n")
	fmt.Printf("Experiment: %s\n", result.ExperimentName)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Duration: %v\n", result.EndTime.Sub(result.StartTime))
	fmt.Printf("Hypothesis Valid: %t\n", result.HypothesisValid)

	fmt.Printf("Steady State:\n")
	fmt.Printf("  Initial: %t\n", result.SteadyState.InitialCheck)
	fmt.Printf("  Final: %t\n", result.SteadyState.FinalCheck)
	fmt.Printf("  Maintained: %t\n", result.SteadyState.Maintained)
	fmt.Printf("  Violations: %d\n", len(result.SteadyState.Violations))

	fmt.Printf("Perturbations Applied: %d\n", len(result.Perturbations))
	for _, pert := range result.Perturbations {
		fmt.Printf("  - %s: Applied=%t\n", pert.Name, pert.Applied)
	}

	if len(result.Insights) > 0 {
		fmt.Printf("\nInsights:\n")
		for _, insight := range result.Insights {
			fmt.Printf("  - %s\n", insight)
		}
	}

	if len(result.Recommendations) > 0 {
		fmt.Printf("\nRecommendations:\n")
		for _, rec := range result.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}
}

// printStressResults prints stress test results
func printStressResults(result *stress.StressTestResult) {
	fmt.Printf("\n--- Stress Test Results ---\n")
	fmt.Printf("Test: %s\n", result.TestName)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("Successful: %d\n", result.SuccessfulReqs)
	fmt.Printf("Failed: %d\n", result.FailedRequests)
	fmt.Printf("Error Rate: %.2f%%\n", result.ErrorRate*100)
	fmt.Printf("Average RPS: %.2f\n", result.AverageRPS)
	fmt.Printf("Peak RPS: %.2f\n", result.PeakRPS)

	fmt.Printf("\nResponse Times:\n")
	fmt.Printf("  Min: %v\n", result.ResponseTimes.Min)
	fmt.Printf("  Max: %v\n", result.ResponseTimes.Max)
	fmt.Printf("  Mean: %v\n", result.ResponseTimes.Mean)
	fmt.Printf("  Median: %v\n", result.ResponseTimes.Median)
	fmt.Printf("  P95: %v\n", result.ResponseTimes.P95)
	fmt.Printf("  P99: %v\n", result.ResponseTimes.P99)

	if result.BreakingPoint != nil {
		fmt.Printf("\nBreaking Point Detected:\n")
		fmt.Printf("  RPS: %.2f\n", result.BreakingPoint.RPS)
		fmt.Printf("  Error Rate: %.2f%%\n", result.BreakingPoint.ErrorRate*100)
		fmt.Printf("  Reason: %s\n", result.BreakingPoint.Reason)
		fmt.Printf("  Detected At: %s\n", result.BreakingPoint.DetectedAt.Format(time.RFC3339))
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nError Summary:\n")
		for errorMsg, count := range result.Errors {
			fmt.Printf("  %s: %d\n", errorMsg, count)
		}
	}

	fmt.Println("---------------------------")

	// Save detailed results to JSON file
	filename := fmt.Sprintf("stress_results_%s_%d.json", 
		result.TestName, result.StartTime.Unix())
	filename = "stress_results.json" // Simplified filename
	
	if jsonData, err := json.MarshalIndent(result, "", "  "); err == nil {
		if err := os.WriteFile(filename, jsonData, 0644); err == nil {
			fmt.Printf("Detailed results saved to: %s\n", filename)
		}
	}
}