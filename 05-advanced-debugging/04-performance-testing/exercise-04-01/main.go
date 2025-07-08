package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"exercise-04-01/loadtest"
	"exercise-04-01/pipeline"
	"exercise-04-01/reporter"
)

func main() {
	fmt.Println("Load Testing Framework Demo")
	fmt.Println("============================")

	// Create a demo HTTP server to test against
	server := createDemoServer()
	defer server.Close()

	fmt.Printf("Demo server started at: %s\n\n", server.URL)

	// Demo 1: Basic Load Test
	fmt.Println("Demo 1: Basic Load Test")
	fmt.Println("-----------------------")
	runBasicLoadTest(server.URL)

	// Demo 2: Multi-scenario Load Test
	fmt.Println("\nDemo 2: Multi-scenario Load Test")
	fmt.Println("--------------------------------")
	runMultiScenarioTest(server.URL)

	// Demo 3: Report Generation
	fmt.Println("\nDemo 3: Report Generation")
	fmt.Println("-------------------------")
	runReportDemo(server.URL)

	// Demo 4: Performance Pipeline
	fmt.Println("\nDemo 4: Performance Pipeline")
	fmt.Println("----------------------------")
	runPipelineDemo(server.URL)

	fmt.Println("\nAll demos completed successfully!")
}

func createDemoServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate different response times based on endpoint
		switch r.URL.Path {
		case "/api/fast":
			time.Sleep(5 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "fast response", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
		case "/api/slow":
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "slow response", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
		case "/api/error":
			// Simulate 20% error rate
			if time.Now().UnixNano()%5 == 0 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "success"}`))
			}
		case "/api/data":
			time.Sleep(20 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": [1,2,3,4,5], "count": 5}`))
		default:
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "default response"}`))
		}
	}))
}

func runBasicLoadTest(baseURL string) {
	config := loadtest.LoadTestConfig{
		Duration: 3 * time.Second,
		Scenarios: []loadtest.TestScenario{
			{
				Name:  "basic_api_test",
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:    "/api/fast",
						Headers: map[string]string{
							"User-Agent":   "LoadTester/1.0",
							"Accept":       "application/json",
							"Content-Type": "application/json",
						},
					},
				},
			},
		},
		Assertions: []loadtest.Assertion{
			loadtest.AssertMaxLatency(50 * time.Millisecond),
			loadtest.AssertMinThroughput(50.0),
			loadtest.AssertMaxErrorRate(5.0),
		},
	}

	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()
	if err != nil {
		log.Printf("Load test failed: %v", err)
		return
	}

	printResults(result)
}

func runMultiScenarioTest(baseURL string) {
	config := loadtest.LoadTestConfig{
		Duration: 5 * time.Second,
		Scenarios: []loadtest.TestScenario{
			{
				Name:  "fast_endpoint",
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:    "/api/fast",
						Headers: map[string]string{
							"Accept": "application/json",
						},
					},
				},
			},
			{
				Name:  "slow_endpoint",
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:    "/api/slow",
						Headers: map[string]string{
							"Accept": "application/json",
						},
					},
				},
			},
			{
				Name:  "mixed_operations",
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:    "/api/data",
						Headers: map[string]string{
							"Accept": "application/json",
						},
					},
					{
						Method: "POST",
						Path:    "/api/data",
						Body:   []byte(`{"test": "data", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`),
						Headers: map[string]string{
							"Content-Type": "application/json",
							"Accept":       "application/json",
						},
					},
				},
			},
		},
		Assertions: []loadtest.Assertion{
			loadtest.AssertMaxLatency(200 * time.Millisecond),
			loadtest.AssertMinThroughput(30.0),
			loadtest.AssertMaxErrorRate(10.0),
		},
	}

	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()
	if err != nil {
		log.Printf("Multi-scenario test failed: %v", err)
		return
	}

	printResults(result)
	printScenarioResults(result)
}

func runReportDemo(baseURL string) {
	// Run a quick test to generate data for reports
	config := loadtest.LoadTestConfig{
		Duration: 3 * time.Second,
		Scenarios: []loadtest.TestScenario{
			{
				Name:  "report_demo",
				Requests: []loadtest.RequestTemplate{
					{
						Method: "GET",
						Path:    "/api/error", // This endpoint has some errors
					},
				},
			},
		},
	}

	tester := loadtest.NewLoadTester(config)
	result, err := tester.Run()
	if err != nil {
		log.Printf("Report demo test failed: %v", err)
		return
	}

	// Create reports directory
	reportsDir := "./reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		log.Printf("Failed to create reports directory: %v", err)
		return
	}

	reportConfig := reporter.ReportConfig{
		OutputPath:  reportsDir,
		Title:       "Load Testing Framework Demo Report",
		Description: "This report demonstrates the capabilities of our load testing framework.",
	}

	// Generate JSON report
	jsonReporter := reporter.NewReporter("json")
	if err := jsonReporter.GenerateReport(result, reportConfig); err != nil {
		log.Printf("Failed to generate JSON report: %v", err)
	} else {
		fmt.Println("✓ JSON report generated")
	}

	// Generate HTML report
	htmlReporter := reporter.NewReporter("html")
	if err := htmlReporter.GenerateReport(result, reportConfig); err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	} else {
		fmt.Println("✓ HTML report generated")
	}

	// Generate CSV report
	csvReporter := reporter.NewReporter("csv")
	if err := csvReporter.GenerateReport(result, reportConfig); err != nil {
		log.Printf("Failed to generate CSV report: %v", err)
	} else {
		fmt.Println("✓ CSV report generated")
	}

	fmt.Printf("Reports saved to: %s\n", reportsDir)
}

func runPipelineDemo(baseURL string) {
	// Create pipeline configuration
	pipelineConfig := pipeline.PipelineConfig{
		Name:         "Demo Performance Pipeline",
		Description:  "Demonstration of automated performance testing pipeline",
		BaselinePath: "./baseline/demo_baseline.json",
		ReportPath:   "./pipeline_reports",
		Notification: pipeline.NotificationConfig{
			Enabled: true, // Enable for demo (will just print)
		},
		RegressionRules: pipeline.RegressionRules{
			LatencyThreshold:    20.0, // 20% increase
			ThroughputThreshold: 15.0, // 15% decrease
			ErrorRateThreshold:  5.0,  // 5% increase
		},
	}

	// Create performance tests
	tests := []*pipeline.PerformanceTest{
		{
			Name:        "API Performance Test",
			Description: "Test API endpoints performance",
			Config: &loadtest.LoadTestConfig{
				Duration: 4 * time.Second,
				Scenarios: []loadtest.TestScenario{
						{
							Name:  "api_performance",
							Requests: []loadtest.RequestTemplate{
								{
									Method: "GET",
									Path:    "/api/fast",
								},
								{
									Method: "GET",
									Path:    "/api/data",
								},
							},
						},
				},
			},
			Expected: pipeline.ExpectedResults{
				MaxLatencyP95:  100 * time.Millisecond,
				MinThroughput:  20.0,
				MaxErrorRate:   5.0,
				MinSuccessRate: 95.0,
			},
		},
		{
			Name:        "Stress Test",
			Description: "High load stress test",
			Config: &loadtest.LoadTestConfig{
				Duration: 3 * time.Second,
				Scenarios: []loadtest.TestScenario{
						{
							Name:  "stress_test",
							Requests: []loadtest.RequestTemplate{
								{
									Method: "GET",
									Path:    "/api/slow",
								},
							},
						},
				},
			},
			Expected: pipeline.ExpectedResults{
				MaxLatencyP95:  200 * time.Millisecond,
				MinThroughput:  10.0,
				MaxErrorRate:   10.0,
				MinSuccessRate: 90.0,
			},
		},
	}

	// Create and run pipeline
	pipe := pipeline.NewPerformancePipeline(pipelineConfig)
	if err := pipe.RunPipeline(tests, "v1.0.0"); err != nil {
		log.Printf("Pipeline execution failed: %v", err)
		return
	}

	fmt.Println("✓ Performance pipeline completed successfully")
}

func printResults(result *loadtest.LoadTestResult) {
	duration := result.EndTime.Sub(result.StartTime)
	successRate := float64(result.SuccessRequests) / float64(result.TotalRequests) * 100
	errorRate := float64(result.FailedRequests) / float64(result.TotalRequests) * 100

	fmt.Printf("Test Duration: %v\n", duration)
	fmt.Printf("Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("Success Rate: %.2f%% (%d requests)\n", successRate, result.SuccessRequests)
	fmt.Printf("Error Rate: %.2f%% (%d requests)\n", errorRate, result.FailedRequests)
	fmt.Printf("Throughput: %.2f req/s\n", result.Throughput)
	fmt.Printf("Latency Stats:\n")
	fmt.Printf("  Min: %v\n", result.Latencies.Min)
	fmt.Printf("  Mean: %v\n", result.Latencies.Mean)
	fmt.Printf("  Max: %v\n", result.Latencies.Max)
	fmt.Printf("  P50: %v\n", result.Latencies.P50)
	fmt.Printf("  P90: %v\n", result.Latencies.P90)
	fmt.Printf("  P95: %v\n", result.Latencies.P95)
	fmt.Printf("  P99: %v\n", result.Latencies.P99)
	fmt.Printf("  P99.9: %v\n", result.Latencies.P999)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for errMsg, count := range result.Errors {
			fmt.Printf("  %s: %d\n", errMsg, count)
		}
	}
}

func printScenarioResults(result *loadtest.LoadTestResult) {
	if len(result.Scenarios) == 0 {
		return
	}

	fmt.Printf("\nScenario Results:\n")
	fmt.Printf("================\n")
	for name, scenario := range result.Scenarios {
			successRate := float64(scenario.SuccessRequests) / float64(scenario.TotalRequests) * 100
			fmt.Printf("\nScenario: %s\n", name)
			fmt.Printf("  Requests: %d (%.2f%% success)\n", scenario.TotalRequests, successRate)
			fmt.Printf("  Throughput: %.2f req/s\n", scenario.Throughput)
			fmt.Printf("  Avg Latency: %v\n", scenario.AvgLatency)
		}
}