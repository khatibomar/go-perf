package benchmark

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Runner manages benchmark execution and baseline comparison
type Runner struct {
	config      *BenchmarkConfig
	outputDir   string
	baselineDir string
	baseline    *Baseline
	verbose     bool
	client      *http.Client
	mu          sync.RWMutex
}

// BenchmarkResults contains all benchmark execution results
type BenchmarkResults struct {
	Name        string                    `json:"name"`
	StartTime   time.Time                 `json:"start_time"`
	EndTime     time.Time                 `json:"end_time"`
	Duration    time.Duration             `json:"duration"`
	Targets     map[string]*TargetResults `json:"targets"`
	Scenarios   map[string]*ScenarioResults `json:"scenarios"`
	Metrics     map[string]*MetricResults `json:"metrics"`
	Thresholds  *ThresholdResults         `json:"thresholds"`
	Baseline    *BaselineComparison       `json:"baseline,omitempty"`
	Summary     *ResultSummary            `json:"summary"`
}

// TargetResults contains results for a specific target
type TargetResults struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	Healthy      bool          `json:"healthy"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
}

// ScenarioResults contains results for a specific scenario
type ScenarioResults struct {
	Name            string                    `json:"name"`
	StartTime       time.Time                 `json:"start_time"`
	EndTime         time.Time                 `json:"end_time"`
	Duration        time.Duration             `json:"duration"`
	TotalRequests   int64                     `json:"total_requests"`
	SuccessRequests int64                     `json:"success_requests"`
	FailedRequests  int64                     `json:"failed_requests"`
	TotalBytes      int64                     `json:"total_bytes"`
	Latencies       *LatencyStats             `json:"latencies"`
	Throughput      float64                   `json:"throughput"`
	ErrorRate       float64                   `json:"error_rate"`
	Errors          map[string]int64          `json:"errors"`
	Requests        map[string]*RequestResults `json:"requests"`
}

// RequestResults contains results for specific request types
type RequestResults struct {
	Name            string        `json:"name"`
	Count           int64         `json:"count"`
	SuccessCount    int64         `json:"success_count"`
	FailedCount     int64         `json:"failed_count"`
	AvgLatency      time.Duration `json:"avg_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	Throughput      float64       `json:"throughput"`
	ErrorRate       float64       `json:"error_rate"`
}

// LatencyStats contains latency statistics
type LatencyStats struct {
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	Mean    time.Duration `json:"mean"`
	Median  time.Duration `json:"median"`
	P90     time.Duration `json:"p90"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
	P999    time.Duration `json:"p999"`
	StdDev  time.Duration `json:"std_dev"`
}

// MetricResults contains results for a specific metric
type MetricResults struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Unit        string      `json:"unit"`
	Value       float64     `json:"value"`
	Aggregation string      `json:"aggregation"`
	Critical    bool        `json:"critical"`
	Passed      bool        `json:"passed"`
	Threshold   float64     `json:"threshold,omitempty"`
	Baseline    *BaselineMetric `json:"baseline,omitempty"`
}

// ThresholdResults contains threshold validation results
type ThresholdResults struct {
	Passed     bool                      `json:"passed"`
	Latency    *LatencyThresholdResults  `json:"latency"`
	Throughput *ThroughputThresholdResults `json:"throughput"`
	ErrorRate  *ErrorRateThresholdResults `json:"error_rate"`
	Resources  *ResourceThresholdResults `json:"resources"`
}

// LatencyThresholdResults contains latency threshold results
type LatencyThresholdResults struct {
	P50Passed  bool `json:"p50_passed"`
	P90Passed  bool `json:"p90_passed"`
	P95Passed  bool `json:"p95_passed"`
	P99Passed  bool `json:"p99_passed"`
	MaxPassed  bool `json:"max_passed"`
}

// ThroughputThresholdResults contains throughput threshold results
type ThroughputThresholdResults struct {
	MinRPSPassed bool `json:"min_rps_passed"`
	MaxRPSPassed bool `json:"max_rps_passed"`
}

// ErrorRateThresholdResults contains error rate threshold results
type ErrorRateThresholdResults struct {
	MaxPercentPassed bool `json:"max_percent_passed"`
	MaxCountPassed   bool `json:"max_count_passed"`
}

// ResourceThresholdResults contains resource threshold results
type ResourceThresholdResults struct {
	CPUPassed     bool `json:"cpu_passed"`
	MemoryPassed  bool `json:"memory_passed"`
	DiskPassed    bool `json:"disk_passed"`
	NetworkPassed bool `json:"network_passed"`
}

// BaselineComparison contains baseline comparison results
type BaselineComparison struct {
	Exists      bool                       `json:"exists"`
	Version     string                     `json:"version"`
	Timestamp   time.Time                  `json:"timestamp"`
	Metrics     map[string]*MetricComparison `json:"metrics"`
	Regression  bool                       `json:"regression"`
	Improvement bool                       `json:"improvement"`
	Summary     string                     `json:"summary"`
}

// MetricComparison contains comparison between current and baseline metrics
type MetricComparison struct {
	Name            string  `json:"name"`
	Current         float64 `json:"current"`
	Baseline        float64 `json:"baseline"`
	Change          float64 `json:"change"`
	ChangePercent   float64 `json:"change_percent"`
	WithinTolerance bool    `json:"within_tolerance"`
	Tolerance       float64 `json:"tolerance"`
	Status          string  `json:"status"` // "improved", "degraded", "stable"
}

// ResultSummary contains high-level summary of results
type ResultSummary struct {
	OverallPassed   bool    `json:"overall_passed"`
	TotalRequests   int64   `json:"total_requests"`
	SuccessRate     float64 `json:"success_rate"`
	AvgThroughput   float64 `json:"avg_throughput"`
	AvgLatency      time.Duration `json:"avg_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	ErrorRate       float64 `json:"error_rate"`
	RegressionCount int     `json:"regression_count"`
	ImprovementCount int    `json:"improvement_count"`
	Recommendations []string `json:"recommendations"`
}

// NewRunner creates a new benchmark runner
func NewRunner(config *BenchmarkConfig, outputDir, baselineDir string) *Runner {
	return &Runner{
		config:      config,
		outputDir:   outputDir,
		baselineDir: baselineDir,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// SetVerbose enables or disables verbose output
func (r *Runner) SetVerbose(verbose bool) {
	r.verbose = verbose
}

// RunBenchmarks executes all configured benchmarks
func (r *Runner) RunBenchmarks(ctx context.Context) (*BenchmarkResults, error) {
	if r.verbose {
		fmt.Printf("Starting benchmark: %s\n", r.config.Name)
	}

	// Load baseline if enabled
	if r.config.Baseline.Enabled {
		if err := r.loadBaseline(); err != nil {
			if r.verbose {
				fmt.Printf("Warning: Failed to load baseline: %v\n", err)
			}
		}
	}

	// Initialize results
	results := &BenchmarkResults{
		Name:      r.config.Name,
		StartTime: time.Now(),
		Targets:   make(map[string]*TargetResults),
		Scenarios: make(map[string]*ScenarioResults),
		Metrics:   make(map[string]*MetricResults),
	}

	// Health check targets
	if err := r.healthCheckTargets(ctx, results); err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	// Warmup phase
	if r.config.WarmupTime > 0 {
		if r.verbose {
			fmt.Printf("Warming up for %v...\n", r.config.WarmupTime)
		}
		time.Sleep(r.config.WarmupTime)
	}

	// Run scenarios
	if err := r.runScenarios(ctx, results); err != nil {
		return nil, fmt.Errorf("scenario execution failed: %w", err)
	}

	// Cooldown phase
	if r.config.CooldownTime > 0 {
		if r.verbose {
			fmt.Printf("Cooling down for %v...\n", r.config.CooldownTime)
		}
		time.Sleep(r.config.CooldownTime)
	}

	results.EndTime = time.Now()
	results.Duration = results.EndTime.Sub(results.StartTime)

	// Calculate metrics
	r.calculateMetrics(results)

	// Validate thresholds
	r.validateThresholds(results)

	// Compare with baseline
	if r.baseline != nil {
		r.compareWithBaseline(results)
	}

	// Generate summary
	r.generateSummary(results)

	// Update baseline if configured
	if r.config.Baseline.Enabled && r.config.Baseline.AutoUpdate && results.Summary.OverallPassed {
		if err := r.updateBaseline(results); err != nil {
			if r.verbose {
				fmt.Printf("Warning: Failed to update baseline: %v\n", err)
			}
		}
	}

	if r.verbose {
		fmt.Printf("Benchmark completed in %v\n", results.Duration)
	}

	return results, nil
}

// healthCheckTargets performs health checks on all targets
func (r *Runner) healthCheckTargets(ctx context.Context, results *BenchmarkResults) error {
	if r.verbose {
		fmt.Println("Performing health checks...")
	}

	for _, target := range r.config.Targets {
		targetResult := &TargetResults{
			Name: target.Name,
			URL:  target.URL,
		}

		start := time.Now()
		healthURL := target.URL
		if target.HealthCheck != "" {
			healthURL = target.URL + target.HealthCheck
		}

		req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		if err != nil {
			targetResult.Error = err.Error()
			results.Targets[target.Name] = targetResult
			continue
		}

		// Add headers
		for k, v := range target.Headers {
			req.Header.Set(k, v)
		}

		resp, err := r.client.Do(req)
		if err != nil {
			targetResult.Error = err.Error()
		} else {
			resp.Body.Close()
			targetResult.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
			if !targetResult.Healthy {
				targetResult.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
		}

		targetResult.ResponseTime = time.Since(start)
		results.Targets[target.Name] = targetResult

		if r.verbose {
			status := "✅"
			if !targetResult.Healthy {
				status = "❌"
			}
			fmt.Printf("  %s %s (%v)\n", status, target.Name, targetResult.ResponseTime)
		}
	}

	return nil
}

// runScenarios executes all configured scenarios
func (r *Runner) runScenarios(ctx context.Context, results *BenchmarkResults) error {
	if r.verbose {
		fmt.Println("Running scenarios...")
	}

	var wg sync.WaitGroup
	for _, scenario := range r.config.Scenarios {
		wg.Add(1)
		go func(s Scenario) {
			defer wg.Done()
			r.runScenario(ctx, s, results)
		}(scenario)
	}

	wg.Wait()
	return nil
}

// runScenario executes a single scenario
func (r *Runner) runScenario(ctx context.Context, scenario Scenario, results *BenchmarkResults) {
	if r.verbose {
		fmt.Printf("  Running scenario: %s\n", scenario.Name)
	}

	scenarioResult := &ScenarioResults{
		Name:      scenario.Name,
		StartTime: time.Now(),
		Errors:    make(map[string]int64),
		Requests:  make(map[string]*RequestResults),
	}

	// Initialize request results
	for _, req := range scenario.Requests {
		scenarioResult.Requests[req.Name] = &RequestResults{
			Name: req.Name,
		}
	}

	// Calculate scenario duration
	duration := r.config.Duration
	if scenario.Duration > 0 {
		duration = scenario.Duration
	}

	// Create scenario context with timeout
	scenarioCtx, cancel := context.WithTimeout(ctx, duration+scenario.RampUpTime+scenario.RampDownTime)
	defer cancel()

	// Run scenario with specified concurrency
	var wg sync.WaitGroup
	var latencies []time.Duration
	var latenciesMu sync.Mutex

	for i := 0; i < scenario.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r.runWorker(scenarioCtx, scenario, scenarioResult, &latencies, &latenciesMu)
		}(i)

		// Ramp up delay
		if scenario.RampUpTime > 0 && i < scenario.Concurrency-1 {
			rampDelay := scenario.RampUpTime / time.Duration(scenario.Concurrency)
			select {
			case <-scenarioCtx.Done():
				return
			case <-time.After(rampDelay):
			}
		}
	}

	wg.Wait()

	scenarioResult.EndTime = time.Now()
	scenarioResult.Duration = scenarioResult.EndTime.Sub(scenarioResult.StartTime)

	// Calculate latency statistics
	if len(latencies) > 0 {
		scenarioResult.Latencies = r.calculateLatencyStats(latencies)
	}

	// Calculate throughput and error rate
	if scenarioResult.Duration > 0 {
		scenarioResult.Throughput = float64(scenarioResult.TotalRequests) / scenarioResult.Duration.Seconds()
	}
	if scenarioResult.TotalRequests > 0 {
		scenarioResult.ErrorRate = float64(scenarioResult.FailedRequests) / float64(scenarioResult.TotalRequests) * 100
	}

	// Calculate request-level statistics
	for _, reqResult := range scenarioResult.Requests {
		if reqResult.Count > 0 {
			reqResult.ErrorRate = float64(reqResult.FailedCount) / float64(reqResult.Count) * 100
			if scenarioResult.Duration > 0 {
				reqResult.Throughput = float64(reqResult.Count) / scenarioResult.Duration.Seconds()
			}
		}
	}

	r.mu.Lock()
	results.Scenarios[scenario.Name] = scenarioResult
	r.mu.Unlock()

	if r.verbose {
		fmt.Printf("    Completed: %d requests, %.2f RPS, %.2f%% errors\n",
			scenarioResult.TotalRequests, scenarioResult.Throughput, scenarioResult.ErrorRate)
	}
}

// runWorker executes requests for a single worker
func (r *Runner) runWorker(ctx context.Context, scenario Scenario, result *ScenarioResults, latencies *[]time.Duration, latenciesMu *sync.Mutex) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Select request based on weight
		req := r.selectRequest(scenario.Requests)
		if req == nil {
			continue
		}

		// Execute request
		latency, err := r.executeRequest(ctx, req, result)
		if err == nil {
			latenciesMu.Lock()
			*latencies = append(*latencies, latency)
			latenciesMu.Unlock()
		}

		// Think time
		if scenario.ThinkTime > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(scenario.ThinkTime):
			}
		}

		// Rate limiting
		if scenario.RequestRate > 0 {
			delay := time.Second / time.Duration(scenario.RequestRate)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}
}

// selectRequest selects a request based on weights
func (r *Runner) selectRequest(requests []Request) *Request {
	if len(requests) == 0 {
		return nil
	}
	if len(requests) == 1 {
		return &requests[0]
	}

	// Simple weighted selection (could be improved with proper algorithm)
	totalWeight := 0.0
	for _, req := range requests {
		totalWeight += req.Weight
	}

	if totalWeight == 0 {
		return &requests[0]
	}

	// For simplicity, just return the first request
	// In a real implementation, you'd use proper weighted random selection
	return &requests[0]
}

// executeRequest executes a single HTTP request
func (r *Runner) executeRequest(ctx context.Context, reqTemplate *Request, result *ScenarioResults) (time.Duration, error) {
	start := time.Now()

	// Find target URL
	targetURL := ""
	for _, target := range r.config.Targets {
		if target.Type == "http" {
			targetURL = target.URL
			break
		}
	}
	if targetURL == "" {
		return 0, fmt.Errorf("no HTTP target found")
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, reqTemplate.Method, targetURL+reqTemplate.Path, nil)
	if err != nil {
		r.recordError(result, reqTemplate.Name, err)
		return 0, err
	}

	// Set headers
	for k, v := range reqTemplate.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := r.client.Do(req)
	latency := time.Since(start)

	atomic.AddInt64(&result.TotalRequests, 1)
	r.updateRequestStats(result, reqTemplate.Name, latency, err == nil)

	if err != nil {
		atomic.AddInt64(&result.FailedRequests, 1)
		r.recordError(result, reqTemplate.Name, err)
		return latency, err
	}
	defer resp.Body.Close()

	// Validate response
	if reqTemplate.Validation.StatusCode > 0 && resp.StatusCode != reqTemplate.Validation.StatusCode {
		err := fmt.Errorf("expected status %d, got %d", reqTemplate.Validation.StatusCode, resp.StatusCode)
		atomic.AddInt64(&result.FailedRequests, 1)
		r.recordError(result, reqTemplate.Name, err)
		return latency, err
	}

	atomic.AddInt64(&result.SuccessRequests, 1)
	return latency, nil
}

// recordError records an error for a specific request
func (r *Runner) recordError(result *ScenarioResults, requestName string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	errorMsg := err.Error()
	result.Errors[errorMsg]++

	if reqResult, exists := result.Requests[requestName]; exists {
		reqResult.FailedCount++
	}
}

// updateRequestStats updates statistics for a specific request
func (r *Runner) updateRequestStats(result *ScenarioResults, requestName string, latency time.Duration, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	reqResult, exists := result.Requests[requestName]
	if !exists {
		return
	}

	reqResult.Count++
	if success {
		reqResult.SuccessCount++
	}

	// Update latency stats (simplified)
	if reqResult.MinLatency == 0 || latency < reqResult.MinLatency {
		reqResult.MinLatency = latency
	}
	if latency > reqResult.MaxLatency {
		reqResult.MaxLatency = latency
	}

	// Simple running average (could be improved)
	if reqResult.Count > 0 {
		totalLatency := time.Duration(float64(reqResult.AvgLatency) * float64(reqResult.Count-1))
		reqResult.AvgLatency = (totalLatency + latency) / time.Duration(reqResult.Count)
	}
}

// calculateLatencyStats calculates latency percentiles and statistics
func (r *Runner) calculateLatencyStats(latencies []time.Duration) *LatencyStats {
	if len(latencies) == 0 {
		return &LatencyStats{}
	}

	// Sort latencies
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	stats := &LatencyStats{
		Min: latencies[0],
		Max: latencies[len(latencies)-1],
	}

	// Calculate mean
	var sum time.Duration
	for _, lat := range latencies {
		sum += lat
	}
	stats.Mean = sum / time.Duration(len(latencies))

	// Calculate percentiles
	stats.Median = r.percentile(latencies, 0.5)
	stats.P90 = r.percentile(latencies, 0.9)
	stats.P95 = r.percentile(latencies, 0.95)
	stats.P99 = r.percentile(latencies, 0.99)
	stats.P999 = r.percentile(latencies, 0.999)

	// Calculate standard deviation
	var variance float64
	mean := float64(stats.Mean)
	for _, lat := range latencies {
		diff := float64(lat) - mean
		variance += diff * diff
	}
	variance /= float64(len(latencies))
	stats.StdDev = time.Duration(math.Sqrt(variance))

	return stats
}

// percentile calculates the specified percentile from sorted latencies
func (r *Runner) percentile(sortedLatencies []time.Duration, p float64) time.Duration {
	if len(sortedLatencies) == 0 {
		return 0
	}

	index := int(float64(len(sortedLatencies)) * p)
	if index >= len(sortedLatencies) {
		index = len(sortedLatencies) - 1
	}

	return sortedLatencies[index]
}