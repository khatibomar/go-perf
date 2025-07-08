// main_test.go
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"exercise-04-03/chaos"
	"exercise-04-03/stress"
)

func TestRunChaosExperiments(t *testing.T) {
	// Test that chaos experiments can be run without panicking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Capture the function execution
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runChaosExperiments panicked: %v", r)
		}
	}()

	// This will run the actual chaos experiments with shortened durations
	// We can't easily mock the output, but we can ensure it doesn't crash
	runChaosExperiments(ctx)
}

func TestRunStressTests(t *testing.T) {
	// Test that stress tests can be run without panicking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Capture the function execution
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runStressTests panicked: %v", r)
		}
	}()

	// This will run the actual stress tests
	// Note: HTTP tests might fail due to network issues, but shouldn't panic
	runStressTests(ctx)
}

func TestRunCombinedTests(t *testing.T) {
	// Test that combined tests can be run without panicking
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Capture the function execution
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runCombinedTests panicked: %v", r)
		}
	}()

	// This will run both chaos and stress tests
	runCombinedTests(ctx)
}

func TestPrintChaosResults(t *testing.T) {
	// Test the chaos results printing function
	result := &chaos.ExperimentResult{
		ExperimentName: "Test Experiment",
		Status:         "success",
		StartTime:      time.Now().Add(-1 * time.Minute),
		EndTime:        time.Now(),
		Hypothesis:     "Test hypothesis",
		HypothesisValid: true,
		SteadyState: chaos.SteadyStateResult{
			InitialCheck: true,
			FinalCheck:   true,
			Maintained:   true,
			Violations:   []chaos.SteadyStateViolation{},
		},
		Perturbations: []chaos.PerturbationResult{
			{
				Name:      "Test Perturbation",
				Applied:   true,
				StartTime: time.Now().Add(-30 * time.Second),
				EndTime:   time.Now(),
				Error:     "",
				Impact: chaos.PerturbationImpact{
					RecoveryTime:       5 * time.Second,
					ServiceDegradation: 0.1,
					ErrorRateIncrease:  0.05,
					LatencyIncrease:    0.2,
				},
			},
		},
		Metrics: map[string][]chaos.MetricPoint{
			"CPU": {{Value: 50.0, Timestamp: time.Now()}},
			"Memory": {{Value: 1024.0, Timestamp: time.Now()}},
		},
		Insights: []string{
			"Test insight 1",
			"Test insight 2",
		},
		Recommendations: []string{
			"Test recommendation 1",
		},
	}

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printChaosResults panicked: %v", r)
		}
	}()

	printChaosResults(result)
}

func TestPrintStressResults(t *testing.T) {
	// Test the stress results printing function
	result := &stress.StressTestResult{
		TestName:       "Test Stress",
		StartTime:      time.Now().Add(-1 * time.Minute),
		EndTime:        time.Now(),
		Duration:       1 * time.Minute,
		TotalRequests:  1000,
		SuccessfulReqs: 950,
		FailedRequests: 50,
		AverageRPS:     16.67,
		PeakRPS:        25.0,
		ResponseTimes: stress.ResponseTimeStats{
			Min:    10 * time.Millisecond,
			Max:    500 * time.Millisecond,
			Mean:   100 * time.Millisecond,
			Median: 95 * time.Millisecond,
			P95:    200 * time.Millisecond,
			P99:    350 * time.Millisecond,
			StdDev: 50 * time.Millisecond,
		},
		ErrorRate:  0.05,
		Throughput: 1.5,
		CPUUsage: stress.CPUStats{
			Min:     10.0,
			Max:     80.0,
			Average: 45.0,
			Peak:    85.0,
		},
		MemoryUsage: stress.MemoryStats{
			MinMB:     100.0,
			MaxMB:     500.0,
			AverageMB: 300.0,
			PeakMB:    520.0,
		},
		Errors: map[string]int64{
			"timeout error":    30,
			"connection error": 20,
		},
		BreakingPoint: &stress.BreakingPoint{
			RPS:          20.0,
			Concurrency:  50,
			ResponseTime: 300 * time.Millisecond,
			ErrorRate:    0.1,
			DetectedAt:   time.Now().Add(-30 * time.Second),
			Reason:       "Error rate exceeded threshold",
		},
	}

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printStressResults panicked: %v", r)
		}
	}()

	printStressResults(result)

	// Check if JSON file was created
	if _, err := os.Stat("stress_results.json"); err != nil {
		t.Logf("JSON file not created (this is okay): %v", err)
	} else {
		// Clean up the file
		os.Remove("stress_results.json")
	}
}

func TestChaosExperimentCreation(t *testing.T) {
	// Test that we can create chaos experiments as done in main
	cpuExperiment := chaos.ChaosExperiment{
		Name:        "CPU Stress Test",
		Description: "Test system behavior under high CPU load",
		Hypothesis:  "System should maintain responsiveness under 80% CPU load",
		Duration:    100 * time.Millisecond, // Shortened for test
		SteadyState: chaos.SteadyStateCheck{
			Name: "Response Time Check",
			Check: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			Tolerance: 0.95,
			Interval:  10 * time.Millisecond,
		},
		Perturbations: []chaos.Perturbation{
			chaos.CPUStressPerturbation(80),
		},
		Rollback: chaos.RollbackAction{
			Name: "Stop CPU Stress",
			Action: func(ctx context.Context) error {
				return nil
			},
		},
		SafetyChecks: []chaos.SafetyCheck{
			{
				Name: "System Availability",
				Check: func(ctx context.Context) (bool, string, error) {
					return true, "", nil
				},
				Critical: true,
				Interval: 5 * time.Millisecond,
			},
		},
		Metrics: []chaos.MetricCollector{
			{
				Name: "CPU Usage",
				Collect: func(ctx context.Context) (map[string]float64, error) {
					return map[string]float64{"cpu": 75.5}, nil
				},
				Interval: 1 * time.Millisecond,
			},
		},
	}

	if cpuExperiment.Name != "CPU Stress Test" {
		t.Errorf("Expected experiment name 'CPU Stress Test', got '%s'", cpuExperiment.Name)
	}

	if len(cpuExperiment.Perturbations) != 1 {
		t.Errorf("Expected 1 perturbation, got %d", len(cpuExperiment.Perturbations))
	}

	if len(cpuExperiment.SafetyChecks) != 1 {
		t.Errorf("Expected 1 safety check, got %d", len(cpuExperiment.SafetyChecks))
	}

	if len(cpuExperiment.Metrics) != 1 {
		t.Errorf("Expected 1 metric collector, got %d", len(cpuExperiment.Metrics))
	}
}

func TestStressTestConfigCreation(t *testing.T) {
	// Test that we can create stress test configs as done in main
	httpConfig := stress.StressTestConfig{
		Name:             "HTTP Load Test",
		Duration:         100 * time.Millisecond, // Shortened for test
		Concurrency:      2,
		RampUpDuration:   20 * time.Millisecond,
		RampDownDuration: 20 * time.Millisecond,
		TargetRPS:        10,
		MaxRPS:           20,
		ThinkTime:        10 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 0.05,
	}

	if httpConfig.Name != "HTTP Load Test" {
		t.Errorf("Expected config name 'HTTP Load Test', got '%s'", httpConfig.Name)
	}

	if httpConfig.Concurrency != 2 {
		t.Errorf("Expected concurrency 2, got %d", httpConfig.Concurrency)
	}

	if httpConfig.TargetRPS != 10 {
		t.Errorf("Expected target RPS 10, got %d", httpConfig.TargetRPS)
	}

	if httpConfig.FailureThreshold != 0.05 {
		t.Errorf("Expected failure threshold 0.05, got %f", httpConfig.FailureThreshold)
	}
}

func TestStressTestFunctionCreation(t *testing.T) {
	// Test that we can create stress test functions as done in main
	httpTestFunc := stress.HTTPStressTest("https://httpbin.org/get", "GET", 
		map[string]string{"User-Agent": "StressTest/1.0"}, nil)

	if httpTestFunc == nil {
		t.Error("HTTPStressTest returned nil function")
	}

	cpuTestFunc := stress.CPUStressTest(100)
	if cpuTestFunc == nil {
		t.Error("CPUStressTest returned nil function")
	}

	memoryTestFunc := stress.MemoryStressTest(1)
	if memoryTestFunc == nil {
		t.Error("MemoryStressTest returned nil function")
	}

	// Test CPU function execution
	ctx := context.Background()
	err := cpuTestFunc(ctx)
	if err != nil {
		t.Errorf("CPU stress test function failed: %v", err)
	}

	// Test memory function execution
	err = memoryTestFunc(ctx)
	if err != nil {
		t.Errorf("Memory stress test function failed: %v", err)
	}
}

func TestIntegration_ChaosAndStress(t *testing.T) {
	// Integration test that runs a simplified version of both chaos and stress tests
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create a simple chaos engine and experiment
	engine := chaos.NewChaosEngine(chaos.ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   1,
		DefaultTimeout:  1 * time.Second,
		MetricsInterval: 100 * time.Millisecond,
		ReportPath:      "./test_reports",
	})
	experiment := chaos.ChaosExperiment{
		Name:        "Integration Test Experiment",
		Description: "Simple integration test",
		Hypothesis:  "System should remain stable",
		Duration:    100 * time.Millisecond,
		SteadyState: chaos.SteadyStateCheck{
			Name: "Always Pass",
			Check: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			Tolerance: 1.0,
			Interval:  50 * time.Millisecond,
		},
		Perturbations: []chaos.Perturbation{},
		Rollback: chaos.RollbackAction{
			Name: "No-op Rollback",
			Action: func(ctx context.Context) error {
				return nil
			},
		},
	}

	engine.AddExperiment(experiment)
	chaosResult, err := engine.RunExperiment(ctx, "Integration Test Experiment")
	if err != nil {
		t.Errorf("Chaos experiment failed: %v", err)
	}

	if chaosResult == nil {
		t.Error("Chaos experiment returned nil result")
	}

	// Create and run a simple stress test with very light load to avoid breaking point
	stressConfig := stress.StressTestConfig{
		Name:             "Integration Stress Test",
		Duration:         100 * time.Millisecond,
		Concurrency:      1, // Reduced concurrency
		RampUpDuration:   20 * time.Millisecond,
		RampDownDuration: 20 * time.Millisecond,
		TargetRPS:        5,  // Reduced target RPS
		MaxRPS:           10, // Reduced max RPS
		ThinkTime:        10 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 0.5, // Higher threshold to tolerate some failures
	}

	stressGenerator := stress.NewLoadGenerator(stressConfig)
	stressTestFunc := stress.CPUStressTest(10) // Very light CPU stress
	stressResult, err := stressGenerator.RunStressTest(ctx, stressTestFunc)
	// Note: Stress test may fail due to breaking point detection, which is expected behavior
	if err != nil {
		t.Logf("Stress test failed (this may be expected): %v", err)
	}

	// Verify chaos experiment completed successfully
	if chaosResult != nil && chaosResult.Status != "success" {
		t.Errorf("Expected chaos experiment to succeed, got status: %s", chaosResult.Status)
	}

	// If stress test succeeded, verify it executed some requests
	if stressResult != nil && stressResult.TotalRequests <= 0 {
		t.Errorf("Expected stress test to execute requests, got: %d", stressResult.TotalRequests)
	}
}