// stress/testing.go
package stress

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// StressTestConfig defines configuration for stress testing
type StressTestConfig struct {
	Name            string        `json:"name"`
	Duration        time.Duration `json:"duration"`
	Concurrency     int           `json:"concurrency"`
	RampUpDuration  time.Duration `json:"ramp_up_duration"`
	RampDownDuration time.Duration `json:"ramp_down_duration"`
	TargetRPS       int           `json:"target_rps"`
	MaxRPS          int           `json:"max_rps"`
	ThinkTime       time.Duration `json:"think_time"`
	Timeout         time.Duration `json:"timeout"`
	FailureThreshold float64      `json:"failure_threshold"`
}

// StressTestResult contains results from stress testing
type StressTestResult struct {
	TestName        string            `json:"test_name"`
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	Duration        time.Duration     `json:"duration"`
	TotalRequests   int64             `json:"total_requests"`
	SuccessfulReqs  int64             `json:"successful_requests"`
	FailedRequests  int64             `json:"failed_requests"`
	AverageRPS      float64           `json:"average_rps"`
	PeakRPS         float64           `json:"peak_rps"`
	ResponseTimes   ResponseTimeStats `json:"response_times"`
	ErrorRate       float64           `json:"error_rate"`
	Throughput      float64           `json:"throughput_mbps"`
	CPUUsage        CPUStats          `json:"cpu_usage"`
	MemoryUsage     MemoryStats       `json:"memory_usage"`
	Errors          map[string]int64  `json:"errors"`
	BreakingPoint   *BreakingPoint    `json:"breaking_point,omitempty"`
}

// ResponseTimeStats contains response time statistics
type ResponseTimeStats struct {
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	Mean    time.Duration `json:"mean"`
	Median  time.Duration `json:"median"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
	StdDev  time.Duration `json:"std_dev"`
}

// CPUStats contains CPU usage statistics
type CPUStats struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	Peak    float64 `json:"peak"`
}

// MemoryStats contains memory usage statistics
type MemoryStats struct {
	MinMB     float64 `json:"min_mb"`
	MaxMB     float64 `json:"max_mb"`
	AverageMB float64 `json:"average_mb"`
	PeakMB    float64 `json:"peak_mb"`
}

// BreakingPoint represents the point where system starts failing
type BreakingPoint struct {
	RPS           float64       `json:"rps"`
	Concurrency   int           `json:"concurrency"`
	ResponseTime  time.Duration `json:"response_time"`
	ErrorRate     float64       `json:"error_rate"`
	DetectedAt    time.Time     `json:"detected_at"`
	Reason        string        `json:"reason"`
}

// LoadGenerator generates load for stress testing
type LoadGenerator struct {
	config       StressTestConfig
	client       *http.Client
	metrics      *StressMetrics
	breakingPoint *BreakingPoint
	mu           sync.RWMutex
}

// StressMetrics tracks real-time metrics during stress testing
type StressMetrics struct {
	totalRequests   int64
	successfulReqs  int64
	failedRequests  int64
	responseTimes   []time.Duration
	errorCounts     map[string]int64
	currentRPS      int64
	peakRPS         int64
	lastSecond      int64
	requestBytes    int64
	responseBytes   int64
	mu              sync.RWMutex
}

// NewLoadGenerator creates a new load generator
func NewLoadGenerator(config StressTestConfig) *LoadGenerator {
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &LoadGenerator{
		config: config,
		client: client,
		metrics: &StressMetrics{
			errorCounts: make(map[string]int64),
		},
	}
}

// RunStressTest executes a stress test with the given configuration
func (lg *LoadGenerator) RunStressTest(ctx context.Context, testFunc func(context.Context) error) (*StressTestResult, error) {
	startTime := time.Now()
	testCtx, cancel := context.WithTimeout(ctx, lg.config.Duration)
	defer cancel()

	// Start monitoring goroutines
	monitorCtx, monitorCancel := context.WithCancel(testCtx)
	defer monitorCancel()

	go lg.monitorRPS(monitorCtx)
	go lg.monitorSystemResources(monitorCtx)

	// Execute stress test with ramp-up
	if err := lg.executeWithRampUp(testCtx, testFunc); err != nil {
		return nil, fmt.Errorf("stress test execution failed: %w", err)
	}

	endTime := time.Now()
	return lg.generateResult(startTime, endTime), nil
}

// executeWithRampUp executes the stress test with gradual ramp-up
func (lg *LoadGenerator) executeWithRampUp(ctx context.Context, testFunc func(context.Context) error) error {
	totalDuration := lg.config.Duration
	rampUpDuration := lg.config.RampUpDuration
	rampDownDuration := lg.config.RampDownDuration
	steadyDuration := totalDuration - rampUpDuration - rampDownDuration

	if steadyDuration <= 0 {
		return fmt.Errorf("invalid duration configuration: steady state duration must be positive")
	}

	// Phase 1: Ramp-up
	if rampUpDuration > 0 {
		if err := lg.rampUpPhase(ctx, rampUpDuration, testFunc); err != nil {
			return fmt.Errorf("ramp-up phase failed: %w", err)
		}
	}

	// Phase 2: Steady state
	steadyCtx, steadyCancel := context.WithTimeout(ctx, steadyDuration)
	defer steadyCancel()

	if err := lg.steadyStatePhase(steadyCtx, testFunc); err != nil {
		return fmt.Errorf("steady state phase failed: %w", err)
	}

	// Phase 3: Ramp-down
	if rampDownDuration > 0 {
		if err := lg.rampDownPhase(ctx, rampDownDuration, testFunc); err != nil {
			return fmt.Errorf("ramp-down phase failed: %w", err)
		}
	}

	return nil
}

// rampUpPhase gradually increases load
func (lg *LoadGenerator) rampUpPhase(ctx context.Context, duration time.Duration, testFunc func(context.Context) error) error {
	startConcurrency := 1
	targetConcurrency := lg.config.Concurrency
	steps := 10
	stepDuration := duration / time.Duration(steps)

	for step := 0; step < steps; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		currentConcurrency := startConcurrency + (targetConcurrency-startConcurrency)*step/steps
		if err := lg.runLimitedRequests(ctx, currentConcurrency, stepDuration, testFunc); err != nil {
			return err
		}

		// Check for breaking point
		if lg.checkBreakingPoint() {
			return fmt.Errorf("breaking point detected during ramp-up")
		}
	}

	return nil
}

// steadyStatePhase runs at target concurrency
func (lg *LoadGenerator) steadyStatePhase(ctx context.Context, testFunc func(context.Context) error) error {
	return lg.runConcurrentRequests(ctx, lg.config.Concurrency, testFunc)
}

// rampDownPhase gradually decreases load
func (lg *LoadGenerator) rampDownPhase(ctx context.Context, duration time.Duration, testFunc func(context.Context) error) error {
	startConcurrency := lg.config.Concurrency
	targetConcurrency := 1
	steps := 5
	stepDuration := duration / time.Duration(steps)

	for step := 0; step < steps; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		currentConcurrency := startConcurrency - (startConcurrency-targetConcurrency)*step/steps
		if err := lg.runLimitedRequests(ctx, currentConcurrency, stepDuration, testFunc); err != nil {
			return err
		}
	}

	return nil
}

// runLimitedRequests executes requests with specified concurrency for a limited duration
func (lg *LoadGenerator) runLimitedRequests(ctx context.Context, concurrency int, duration time.Duration, testFunc func(context.Context) error) error {
	var wg sync.WaitGroup
	stepCtx, stepCancel := context.WithTimeout(ctx, duration)
	defer stepCancel()

	// Start worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stepCtx.Done():
					return
				default:
					startTime := time.Now()
					err := testFunc(stepCtx)
					responseTime := time.Since(startTime)

					lg.recordMetrics(err, responseTime)

					// Think time between requests
					if lg.config.ThinkTime > 0 {
						select {
						case <-stepCtx.Done():
							return
						case <-time.After(lg.config.ThinkTime):
						}
					}
				}
			}
		}()
	}

	// Wait for step duration to complete
	<-stepCtx.Done()
	wg.Wait()
	return nil
}

// runConcurrentRequests executes requests with specified concurrency
func (lg *LoadGenerator) runConcurrentRequests(ctx context.Context, concurrency int, testFunc func(context.Context) error) error {
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					startTime := time.Now()
					err := testFunc(ctx)
					responseTime := time.Since(startTime)

					lg.recordMetrics(err, responseTime)

					// Think time between requests
					if lg.config.ThinkTime > 0 {
						select {
						case <-ctx.Done():
							return
						case <-time.After(lg.config.ThinkTime):
						}
					}
				}
			}
		}()
	}

	// Wait for context cancellation
	<-ctx.Done()
	wg.Wait()
	// Don't return context deadline exceeded as an error for normal timeout
	return nil
}

// recordMetrics records request metrics
func (lg *LoadGenerator) recordMetrics(err error, responseTime time.Duration) {
	lg.metrics.mu.Lock()
	defer lg.metrics.mu.Unlock()

	atomic.AddInt64(&lg.metrics.totalRequests, 1)
	lg.metrics.responseTimes = append(lg.metrics.responseTimes, responseTime)

	if err != nil {
		atomic.AddInt64(&lg.metrics.failedRequests, 1)
		errorMsg := err.Error()
		lg.metrics.errorCounts[errorMsg]++
	} else {
		atomic.AddInt64(&lg.metrics.successfulReqs, 1)
	}
}

// monitorRPS monitors requests per second
func (lg *LoadGenerator) monitorRPS(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastTotal := int64(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentTotal := atomic.LoadInt64(&lg.metrics.totalRequests)
			currentRPS := currentTotal - lastTotal
			lastTotal = currentTotal

			atomic.StoreInt64(&lg.metrics.currentRPS, currentRPS)
			if currentRPS > atomic.LoadInt64(&lg.metrics.peakRPS) {
				atomic.StoreInt64(&lg.metrics.peakRPS, currentRPS)
			}
		}
	}
}

// monitorSystemResources monitors CPU and memory usage
func (lg *LoadGenerator) monitorSystemResources(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Monitor system resources
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			// Store memory stats (simplified)
			memoryMB := float64(m.Alloc) / 1024 / 1024
			_ = memoryMB // Use in actual implementation
		}
	}
}

// checkBreakingPoint checks if system has reached breaking point
func (lg *LoadGenerator) checkBreakingPoint() bool {
	lg.metrics.mu.RLock()
	defer lg.metrics.mu.RUnlock()

	totalReqs := atomic.LoadInt64(&lg.metrics.totalRequests)
	failedReqs := atomic.LoadInt64(&lg.metrics.failedRequests)

	if totalReqs == 0 {
		return false
	}

	errorRate := float64(failedReqs) / float64(totalReqs)
	if errorRate > lg.config.FailureThreshold {
		lg.mu.Lock()
		if lg.breakingPoint == nil {
			lg.breakingPoint = &BreakingPoint{
				RPS:         float64(atomic.LoadInt64(&lg.metrics.currentRPS)),
				ErrorRate:   errorRate,
				DetectedAt:  time.Now(),
				Reason:      fmt.Sprintf("Error rate %.2f%% exceeded threshold %.2f%%", errorRate*100, lg.config.FailureThreshold*100),
			}
		}
		lg.mu.Unlock()
		return true
	}

	return false
}

// generateResult generates the final test result
func (lg *LoadGenerator) generateResult(startTime, endTime time.Time) *StressTestResult {
	lg.metrics.mu.RLock()
	defer lg.metrics.mu.RUnlock()

	totalReqs := atomic.LoadInt64(&lg.metrics.totalRequests)
	successfulReqs := atomic.LoadInt64(&lg.metrics.successfulReqs)
	failedReqs := atomic.LoadInt64(&lg.metrics.failedRequests)
	peakRPS := atomic.LoadInt64(&lg.metrics.peakRPS)

	duration := endTime.Sub(startTime)
	averageRPS := float64(totalReqs) / duration.Seconds()
	errorRate := float64(failedReqs) / float64(totalReqs)

	// Calculate response time statistics
	responseTimeStats := lg.calculateResponseTimeStats()

	return &StressTestResult{
		TestName:       lg.config.Name,
		StartTime:      startTime,
		EndTime:        endTime,
		Duration:       duration,
		TotalRequests:  totalReqs,
		SuccessfulReqs: successfulReqs,
		FailedRequests: failedReqs,
		AverageRPS:     averageRPS,
		PeakRPS:        float64(peakRPS),
		ResponseTimes:  responseTimeStats,
		ErrorRate:      errorRate,
		Errors:         lg.copyErrorCounts(),
		BreakingPoint:  lg.breakingPoint,
	}
}

// calculateResponseTimeStats calculates response time statistics
func (lg *LoadGenerator) calculateResponseTimeStats() ResponseTimeStats {
	if len(lg.metrics.responseTimes) == 0 {
		return ResponseTimeStats{}
	}

	// Sort response times for percentile calculations
	times := make([]time.Duration, len(lg.metrics.responseTimes))
	copy(times, lg.metrics.responseTimes)

	// Simple bubble sort for small datasets
	for i := 0; i < len(times); i++ {
		for j := i + 1; j < len(times); j++ {
			if times[i] > times[j] {
				times[i], times[j] = times[j], times[i]
			}
		}
	}

	n := len(times)
	min := times[0]
	max := times[n-1]
	median := times[n/2]
	p95 := times[int(float64(n)*0.95)]
	p99 := times[int(float64(n)*0.99)]

	// Calculate mean
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	mean := sum / time.Duration(n)

	// Calculate standard deviation
	var variance float64
	for _, t := range times {
		diff := float64(t - mean)
		variance += diff * diff
	}
	stdDev := time.Duration(variance / float64(n))

	return ResponseTimeStats{
		Min:    min,
		Max:    max,
		Mean:   mean,
		Median: median,
		P95:    p95,
		P99:    p99,
		StdDev: stdDev,
	}
}

// copyErrorCounts creates a copy of error counts
func (lg *LoadGenerator) copyErrorCounts() map[string]int64 {
	errorCopy := make(map[string]int64)
	for k, v := range lg.metrics.errorCounts {
		errorCopy[k] = v
	}
	return errorCopy
}

// HTTPStressTest creates a stress test for HTTP endpoints
func HTTPStressTest(url string, method string, headers map[string]string, body []byte) func(context.Context) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Add headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		}

		return nil
	}
}

// CPUStressTest creates a CPU-intensive stress test
func CPUStressTest(intensity int) func(context.Context) error {
	return func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// CPU-intensive calculation - significantly increased workload
		for i := 0; i < intensity*100000; i++ {
			_ = i * i * i
			if i%10000 == 0 {
				// Check context periodically
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}
		}

		return nil
	}
}

// MemoryStressTest creates a memory-intensive stress test
func MemoryStressTest(sizeMB int) func(context.Context) error {
	return func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Allocate memory
		data := make([]byte, sizeMB*1024*1024)
		
		// Write to memory to ensure allocation
		for i := 0; i < len(data); i += 4096 {
			data[i] = byte(rand.Intn(256))
		}

		// Hold memory for a short time
		time.Sleep(100 * time.Millisecond)
		
		// Allow GC to clean up
		runtime.GC()

		return nil
	}
}