// Package main provides load testing utilities for the web server.
// This file contains tools to generate various types of load patterns
// for profiling analysis of the live application.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// LoadTestConfig holds configuration for load testing
type LoadTestConfig struct {
	BaseURL     string
	Concurrency int
	Duration    time.Duration
	RPS         int // Requests per second (0 = unlimited)
	Endpoints   []EndpointConfig
}

// EndpointConfig defines load testing parameters for an endpoint
type EndpointConfig struct {
	Path   string
	Weight int // Relative weight for selection
	Params map[string][]string // Parameter variations
}

// LoadTestResults holds the results of a load test
type LoadTestResults struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	StartTime       time.Time
	EndTime         time.Time
	ErrorsByType    map[string]int64
	LatencyBuckets  map[string]int64
}

// LoadTester manages load testing operations
type LoadTester struct {
	config  LoadTestConfig
	client  *http.Client
	results LoadTestResults
	mu      sync.Mutex
}

// NewLoadTester creates a new load tester instance
func NewLoadTester(config LoadTestConfig) *LoadTester {
	return &LoadTester{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		results: LoadTestResults{
			MinLatency:     time.Hour, // Initialize to high value
			ErrorsByType:   make(map[string]int64),
			LatencyBuckets: make(map[string]int64),
		},
	}
}

// Run executes the load test
func (lt *LoadTester) Run(ctx context.Context) *LoadTestResults {
	lt.results.StartTime = time.Now()
	defer func() {
		lt.results.EndTime = time.Now()
	}()
	
	log.Printf("Starting load test: %d concurrent workers for %v", lt.config.Concurrency, lt.config.Duration)
	
	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, lt.config.Duration)
	defer cancel()
	
	// Rate limiting setup
	var rateLimiter <-chan time.Time
	if lt.config.RPS > 0 {
		ticker := time.NewTicker(time.Second / time.Duration(lt.config.RPS))
		defer ticker.Stop()
		rateLimiter = ticker.C
	}
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			lt.worker(testCtx, workerID, rateLimiter)
		}(i)
	}
	
	// Progress reporting
	go lt.reportProgress(testCtx)
	
	wg.Wait()
	log.Printf("Load test completed")
	
	return &lt.results
}

// worker performs load testing requests
func (lt *LoadTester) worker(ctx context.Context, workerID int, rateLimiter <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Rate limiting
			if rateLimiter != nil {
				select {
				case <-rateLimiter:
				case <-ctx.Done():
					return
				}
			}
			
			// Select endpoint and make request
			endpoint := lt.selectEndpoint()
			lt.makeRequest(ctx, endpoint)
		}
	}
}

// selectEndpoint randomly selects an endpoint based on weights
func (lt *LoadTester) selectEndpoint() EndpointConfig {
	if len(lt.config.Endpoints) == 0 {
		return EndpointConfig{Path: "/"}
	}
	
	// Calculate total weight
	totalWeight := 0
	for _, ep := range lt.config.Endpoints {
		totalWeight += ep.Weight
	}
	
	// Select based on weight
	random := rand.Intn(totalWeight)
	currentWeight := 0
	for _, ep := range lt.config.Endpoints {
		currentWeight += ep.Weight
		if random < currentWeight {
			return ep
		}
	}
	
	return lt.config.Endpoints[0] // Fallback
}

// makeRequest performs a single HTTP request
func (lt *LoadTester) makeRequest(ctx context.Context, endpoint EndpointConfig) {
	start := time.Now()
	
	// Build URL with random parameters
	url := lt.buildURL(endpoint)
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		lt.recordError("request_creation", time.Since(start))
		return
	}
	
	// Make request
	resp, err := lt.client.Do(req)
	if err != nil {
		lt.recordError("network", time.Since(start))
		return
	}
	defer resp.Body.Close()
	
	// Read response
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		lt.recordError("response_read", time.Since(start))
		return
	}
	
	latency := time.Since(start)
	
	// Record results
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		lt.recordSuccess(latency)
	} else {
		lt.recordError(fmt.Sprintf("http_%d", resp.StatusCode), latency)
	}
}

// buildURL constructs URL with random parameters
func (lt *LoadTester) buildURL(endpoint EndpointConfig) string {
	baseURL := lt.config.BaseURL + endpoint.Path
	
	if len(endpoint.Params) == 0 {
		return baseURL
	}
	
	// Add random parameters
	params := url.Values{}
	for key, values := range endpoint.Params {
		if len(values) > 0 {
			value := values[rand.Intn(len(values))]
			params.Add(key, value)
		}
	}
	
	if len(params) > 0 {
		return baseURL + "?" + params.Encode()
	}
	
	return baseURL
}

// recordSuccess records a successful request
func (lt *LoadTester) recordSuccess(latency time.Duration) {
	atomic.AddInt64(&lt.results.TotalRequests, 1)
	atomic.AddInt64(&lt.results.SuccessRequests, 1)
	
	lt.mu.Lock()
	lt.results.TotalLatency += latency
	if latency < lt.results.MinLatency {
		lt.results.MinLatency = latency
	}
	if latency > lt.results.MaxLatency {
		lt.results.MaxLatency = latency
	}
	
	// Record latency bucket
	bucket := lt.getLatencyBucket(latency)
	lt.results.LatencyBuckets[bucket]++
	lt.mu.Unlock()
}

// recordError records a failed request
func (lt *LoadTester) recordError(errorType string, latency time.Duration) {
	atomic.AddInt64(&lt.results.TotalRequests, 1)
	atomic.AddInt64(&lt.results.FailedRequests, 1)
	
	lt.mu.Lock()
	lt.results.ErrorsByType[errorType]++
	lt.mu.Unlock()
}

// getLatencyBucket categorizes latency into buckets
func (lt *LoadTester) getLatencyBucket(latency time.Duration) string {
	switch {
	case latency < 10*time.Millisecond:
		return "<10ms"
	case latency < 50*time.Millisecond:
		return "10-50ms"
	case latency < 100*time.Millisecond:
		return "50-100ms"
	case latency < 500*time.Millisecond:
		return "100-500ms"
	case latency < 1*time.Second:
		return "500ms-1s"
	default:
		return ">1s"
	}
}

// reportProgress periodically reports load test progress
func (lt *LoadTester) reportProgress(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			total := atomic.LoadInt64(&lt.results.TotalRequests)
			success := atomic.LoadInt64(&lt.results.SuccessRequests)
			
			if total > 0 {
				successRate := float64(success) / float64(total) * 100
				log.Printf("Progress: %d requests, %.1f%% success rate", total, successRate)
			}
		}
	}
}

// PrintResults prints detailed load test results
func (lt *LoadTester) PrintResults() {
	results := &lt.results
	duration := results.EndTime.Sub(results.StartTime)
	
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total Requests: %d\n", results.TotalRequests)
	fmt.Printf("Successful Requests: %d\n", results.SuccessRequests)
	fmt.Printf("Failed Requests: %d\n", results.FailedRequests)
	
	if results.TotalRequests > 0 {
		successRate := float64(results.SuccessRequests) / float64(results.TotalRequests) * 100
		fmt.Printf("Success Rate: %.2f%%\n", successRate)
		
		rps := float64(results.TotalRequests) / duration.Seconds()
		fmt.Printf("Requests/Second: %.2f\n", rps)
	}
	
	if results.SuccessRequests > 0 {
		avgLatency := results.TotalLatency / time.Duration(results.SuccessRequests)
		fmt.Printf("Average Latency: %v\n", avgLatency)
		fmt.Printf("Min Latency: %v\n", results.MinLatency)
		fmt.Printf("Max Latency: %v\n", results.MaxLatency)
	}
	
	if len(results.ErrorsByType) > 0 {
		fmt.Println("\nErrors by Type:")
		for errorType, count := range results.ErrorsByType {
			fmt.Printf("  %s: %d\n", errorType, count)
		}
	}
	
	if len(results.LatencyBuckets) > 0 {
		fmt.Println("\nLatency Distribution:")
		buckets := []string{"<10ms", "10-50ms", "50-100ms", "100-500ms", "500ms-1s", ">1s"}
		for _, bucket := range buckets {
			if count, exists := results.LatencyBuckets[bucket]; exists {
				percentage := float64(count) / float64(results.SuccessRequests) * 100
				fmt.Printf("  %s: %d (%.1f%%)\n", bucket, count, percentage)
			}
		}
	}
}

// Predefined load test scenarios

// GetCPUIntensiveScenario returns a CPU-intensive load test scenario
func GetCPUIntensiveScenario() LoadTestConfig {
	return LoadTestConfig{
		BaseURL:     "http://localhost:8080",
		Concurrency: 10,
		Duration:    2 * time.Minute,
		RPS:         0, // Unlimited
		Endpoints: []EndpointConfig{
			{
				Path:   "/cpu",
				Weight: 10,
				Params: map[string][]string{
					"iterations": {"100000", "500000", "1000000", "2000000"},
				},
			},
		},
	}
}

// GetMemoryIntensiveScenario returns a memory-intensive load test scenario
func GetMemoryIntensiveScenario() LoadTestConfig {
	return LoadTestConfig{
		BaseURL:     "http://localhost:8080",
		Concurrency: 5,
		Duration:    2 * time.Minute,
		RPS:         0,
		Endpoints: []EndpointConfig{
			{
				Path:   "/memory",
				Weight: 10,
				Params: map[string][]string{
					"size": {"1000", "5000", "10000", "20000"},
				},
			},
		},
	}
}

// GetMixedScenario returns a mixed workload scenario
func GetMixedScenario() LoadTestConfig {
	return LoadTestConfig{
		BaseURL:     "http://localhost:8080",
		Concurrency: 20,
		Duration:    3 * time.Minute,
		RPS:         50, // Rate limited
		Endpoints: []EndpointConfig{
			{
				Path:   "/cpu",
				Weight: 3,
				Params: map[string][]string{
					"iterations": {"100000", "500000", "1000000"},
				},
			},
			{
				Path:   "/memory",
				Weight: 2,
				Params: map[string][]string{
					"size": {"1000", "5000", "10000"},
				},
			},
			{
				Path:   "/database",
				Weight: 4,
				Params: map[string][]string{
					"type":    {"select", "insert", "update"},
					"records": {"10", "50", "100", "200"},
				},
			},
			{
				Path:   "/cache",
				Weight: 5,
				Params: map[string][]string{
					"key": {"user_1", "user_2", "user_3", "config", "settings"},
					"ttl": {"30s", "60s", "120s"},
				},
			},
			{
				Path:   "/concurrent",
				Weight: 2,
				Params: map[string][]string{
					"workers": {"5", "10", "20"},
					"tasks":   {"50", "100", "200"},
				},
			},
			{
				Path:   "/inefficient",
				Weight: 1,
			},
			{
				Path:   "/stats",
				Weight: 1,
			},
		},
	}
}

// GetLeakyScenario returns a scenario that triggers memory leaks
func GetLeakyScenario() LoadTestConfig {
	return LoadTestConfig{
		BaseURL:     "http://localhost:8080",
		Concurrency: 3,
		Duration:    5 * time.Minute,
		RPS:         10, // Slow rate to observe memory growth
		Endpoints: []EndpointConfig{
			{
				Path:   "/leaky",
				Weight: 10,
				Params: map[string][]string{
					"size": {"100", "200", "500"},
				},
			},
		},
	}
}

// runLoadTest function for standalone load testing
func runLoadTest() {
	var (
		scenario    = flag.String("scenario", "mixed", "Load test scenario: cpu, memory, mixed, leaky")
		concurrency = flag.Int("concurrency", 0, "Number of concurrent workers (0 = use scenario default)")
		duration    = flag.Duration("duration", 0, "Test duration (0 = use scenario default)")
		rps         = flag.Int("rps", -1, "Requests per second (-1 = use scenario default)")
		baseURL     = flag.String("url", "http://localhost:8080", "Base URL for testing")
	)
	flag.Parse()
	
	// Select scenario
	var config LoadTestConfig
	switch *scenario {
	case "cpu":
		config = GetCPUIntensiveScenario()
	case "memory":
		config = GetMemoryIntensiveScenario()
	case "mixed":
		config = GetMixedScenario()
	case "leaky":
		config = GetLeakyScenario()
	default:
		log.Fatalf("Unknown scenario: %s", *scenario)
	}
	
	// Override with command line parameters
	if *baseURL != "http://localhost:8080" {
		config.BaseURL = *baseURL
	}
	if *concurrency > 0 {
		config.Concurrency = *concurrency
	}
	if *duration > 0 {
		config.Duration = *duration
	}
	if *rps >= 0 {
		config.RPS = *rps
	}
	
	// Check if server is running
	resp, err := http.Get(config.BaseURL + "/health")
	if err != nil {
		log.Fatalf("Server not reachable at %s: %v", config.BaseURL, err)
	}
	resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.Fatalf("Server health check failed: %d", resp.StatusCode)
	}
	
	log.Printf("Server is healthy, starting load test...")
	
	// Run load test
	loadTester := NewLoadTester(config)
	ctx := context.Background()
	results := loadTester.Run(ctx)
	
	// Print results
	loadTester.PrintResults()
	
	// Print profiling instructions
	fmt.Println("\n=== Profiling Instructions ===")
	fmt.Printf("While load test is running, you can profile the server:\n")
	fmt.Printf("  CPU Profile: go tool pprof %s/debug/pprof/profile\n", config.BaseURL)
	fmt.Printf("  Memory Profile: go tool pprof %s/debug/pprof/heap\n", config.BaseURL)
	fmt.Printf("  Goroutine Profile: go tool pprof %s/debug/pprof/goroutine\n", config.BaseURL)
	fmt.Printf("  Web Interface: %s/debug/pprof/\n", config.BaseURL)
	
	if results.FailedRequests > 0 {
		log.Printf("Load test completed with %d failed requests", results.FailedRequests)
	} else {
		log.Printf("Load test completed successfully")
	}
}