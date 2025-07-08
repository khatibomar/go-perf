// loadtest/framework.go
package loadtest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestConfig struct {
	TargetURL      string
	MaxConcurrency int
	Duration       time.Duration
	RampUpTime     time.Duration
	RampDownTime   time.Duration
	RequestRate    int // requests per second
	Scenarios      []TestScenario
	Assertions     []Assertion
}

type TestScenario struct {
	Name       string
	Weight     float64 // percentage of total load
	Requests   []RequestTemplate
	ThinkTime  time.Duration
	Iterations int
}

type RequestTemplate struct {
	Method     string
	Path       string
	Headers    map[string]string
	Body       []byte
	Timeout    time.Duration
	Validation func(*http.Response) error
}

type Assertion struct {
	Name      string
	Condition func(LoadTestResult) bool
	Message   string
	Severity  string
}

type LoadTestResult struct {
	StartTime       time.Time
	EndTime         time.Time
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalBytes      int64
	Latencies       LatencyStats
	Errors          map[string]int64
	Throughput      float64
	Scenarios       map[string]ScenarioResult
}

type LatencyStats struct {
	Min  time.Duration
	Max  time.Duration
	Mean time.Duration
	P50  time.Duration
	P90  time.Duration
	P95  time.Duration
	P99  time.Duration
	P999 time.Duration
}

type ScenarioResult struct {
	Name            string
	ExecutedUsers   int
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	AvgLatency      time.Duration
	Throughput      float64
	Errors          map[string]int64
	Latencies       []time.Duration
}

type LoadTester struct {
	config    LoadTestConfig
	client    *http.Client
	results   LoadTestResult
	latencies []time.Duration
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewLoadTester(config LoadTestConfig) *LoadTester {
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)

	return &LoadTester{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		results: LoadTestResult{
			Errors:    make(map[string]int64),
			Scenarios: make(map[string]ScenarioResult),
		},
		latencies: make([]time.Duration, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (lt *LoadTester) Run() (*LoadTestResult, error) {
	lt.results.StartTime = time.Now()
	defer func() {
		lt.results.EndTime = time.Now()
		lt.calculateStats()
	}()

	// Start load generation
	var wg sync.WaitGroup

	// Calculate users per scenario
	totalUsers := lt.config.MaxConcurrency

	for _, scenario := range lt.config.Scenarios {
		scenarioUsers := int(float64(totalUsers) * scenario.Weight)
		if scenarioUsers == 0 {
			scenarioUsers = 1
		}

		wg.Add(1)
		go func(s TestScenario, users int) {
			defer wg.Done()
			lt.runScenario(s, users)
		}(scenario, scenarioUsers)
	}

	// Wait for all scenarios to complete
	wg.Wait()

	// Run assertions
	if err := lt.runAssertions(); err != nil {
		return &lt.results, fmt.Errorf("assertion failed: %w", err)
	}

	return &lt.results, nil
}

func (lt *LoadTester) runScenario(scenario TestScenario, users int) {
	var wg sync.WaitGroup

	// Ramp up users gradually
	rampUpInterval := lt.config.RampUpTime / time.Duration(users)

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			lt.runUser(scenario, userID)
		}(i)

		// Ramp up delay
		if i < users-1 {
			select {
			case <-lt.ctx.Done():
				return
			case <-time.After(rampUpInterval):
			}
		}
	}

	wg.Wait()
}

func (lt *LoadTester) runUser(scenario TestScenario, userID int) {
	scenarioResult := ScenarioResult{
		Name:   scenario.Name,
		Errors: make(map[string]int64),
	}

	// Ensure scenario results are saved even if context is cancelled
	defer func() {
		// Update scenario results
		lt.mu.Lock()
		if existing, exists := lt.results.Scenarios[scenario.Name]; exists {
			existing.ExecutedUsers++
			existing.TotalRequests += scenarioResult.TotalRequests
			existing.SuccessRequests += scenarioResult.SuccessRequests
			existing.FailedRequests += scenarioResult.FailedRequests
			existing.Latencies = append(existing.Latencies, scenarioResult.Latencies...)
			for k, v := range scenarioResult.Errors {
				existing.Errors[k] += v
			}
			lt.results.Scenarios[scenario.Name] = existing
		} else {
			scenarioResult.ExecutedUsers = 1
			lt.results.Scenarios[scenario.Name] = scenarioResult
		}
		lt.mu.Unlock()
	}()

	iterations := scenario.Iterations
	if iterations <= 0 {
		iterations = 1000000 // Run until context is cancelled
	}

	for i := 0; i < iterations; i++ {
		select {
		case <-lt.ctx.Done():
			return
		default:
		}

		// Execute requests in scenario
		for _, reqTemplate := range scenario.Requests {
			if err := lt.executeRequest(reqTemplate, &scenarioResult); err != nil {
				lt.recordError(err.Error())
				scenarioResult.Errors[err.Error()]++
			}

			// Think time between requests
			if scenario.ThinkTime > 0 {
				select {
				case <-lt.ctx.Done():
					return
				case <-time.After(scenario.ThinkTime):
				}
			}
		}
	}
}

func (lt *LoadTester) executeRequest(template RequestTemplate, scenarioResult *ScenarioResult) error {
	start := time.Now()

	// Create request
	req, err := http.NewRequestWithContext(lt.ctx, template.Method, lt.config.TargetURL+template.Path, bytes.NewReader(template.Body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range template.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := lt.client.Do(req)
	if err != nil {
		atomic.AddInt64(&lt.results.FailedRequests, 1)
		scenarioResult.FailedRequests++
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Record latency globally
	lt.mu.Lock()
	lt.latencies = append(lt.latencies, duration)
	lt.mu.Unlock()

	// Record latency for scenario
	scenarioResult.Latencies = append(scenarioResult.Latencies, duration)

	// Validate response
	if template.Validation != nil {
		if err := template.Validation(resp); err != nil {
			atomic.AddInt64(&lt.results.FailedRequests, 1)
			scenarioResult.FailedRequests++
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Record success
	atomic.AddInt64(&lt.results.TotalRequests, 1)
	atomic.AddInt64(&lt.results.SuccessRequests, 1)
	scenarioResult.TotalRequests++
	scenarioResult.SuccessRequests++

	// Read response body for byte count
	body, _ := io.ReadAll(resp.Body)
	atomic.AddInt64(&lt.results.TotalBytes, int64(len(body)))

	return nil
}

func (lt *LoadTester) recordError(errMsg string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.results.Errors[errMsg]++
}

func (lt *LoadTester) calculateStats() {
	if len(lt.latencies) == 0 {
		return
	}

	// Sort latencies for percentile calculation
	sort.Slice(lt.latencies, func(i, j int) bool {
		return lt.latencies[i] < lt.latencies[j]
	})

	lt.results.Latencies = LatencyStats{
		Min:  lt.latencies[0],
		Max:  lt.latencies[len(lt.latencies)-1],
		Mean: lt.calculateMean(),
		P50:  lt.calculatePercentile(0.50),
		P90:  lt.calculatePercentile(0.90),
		P95:  lt.calculatePercentile(0.95),
		P99:  lt.calculatePercentile(0.99),
		P999: lt.calculatePercentile(0.999),
	}

	// Calculate throughput
	duration := lt.results.EndTime.Sub(lt.results.StartTime)
	lt.results.Throughput = float64(lt.results.TotalRequests) / duration.Seconds()

	// Calculate scenario-specific statistics
	for name, scenario := range lt.results.Scenarios {
		if len(scenario.Latencies) > 0 {
			// Calculate average latency
			var sum time.Duration
			for _, latency := range scenario.Latencies {
				sum += latency
			}
			scenario.AvgLatency = sum / time.Duration(len(scenario.Latencies))
			
			// Calculate throughput
			scenario.Throughput = float64(scenario.TotalRequests) / duration.Seconds()
			
			lt.results.Scenarios[name] = scenario
		}
	}
}

func (lt *LoadTester) calculateMean() time.Duration {
	var sum time.Duration
	for _, latency := range lt.latencies {
		sum += latency
	}
	return sum / time.Duration(len(lt.latencies))
}

func (lt *LoadTester) calculatePercentile(p float64) time.Duration {
	index := int(float64(len(lt.latencies)) * p)
	if index >= len(lt.latencies) {
		index = len(lt.latencies) - 1
	}
	return lt.latencies[index]
}

func (lt *LoadTester) runAssertions() error {
	for _, assertion := range lt.config.Assertions {
		if !assertion.Condition(lt.results) {
			return fmt.Errorf("assertion '%s' failed: %s", assertion.Name, assertion.Message)
		}
	}
	return nil
}

func (lt *LoadTester) Stop() {
	lt.cancel()
}

// Helper functions for common assertions
func AssertMaxLatency(maxLatency time.Duration) Assertion {
	return Assertion{
		Name: "max_latency",
		Condition: func(result LoadTestResult) bool {
			return result.Latencies.P95 <= maxLatency
		},
		Message:  fmt.Sprintf("95th percentile latency should be <= %v", maxLatency),
		Severity: "critical",
	}
}

func AssertMinThroughput(minThroughput float64) Assertion {
	return Assertion{
		Name: "min_throughput",
		Condition: func(result LoadTestResult) bool {
			return result.Throughput >= minThroughput
		},
		Message:  fmt.Sprintf("throughput should be >= %.2f req/s", minThroughput),
		Severity: "critical",
	}
}

func AssertMaxErrorRate(maxErrorRate float64) Assertion {
	return Assertion{
		Name: "max_error_rate",
		Condition: func(result LoadTestResult) bool {
			if result.TotalRequests == 0 {
				return true
			}
			errorRate := float64(result.FailedRequests) / float64(result.TotalRequests)
			return errorRate <= maxErrorRate
		},
		Message:  fmt.Sprintf("error rate should be <= %.2f%%", maxErrorRate*100),
		Severity: "critical",
	}
}