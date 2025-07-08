package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkClient tests connection pool performance
type BenchmarkClient struct {
	serverAddr     string
	numClients     int
	requestsPerClient int
	duration       time.Duration
	connectionReuse bool

	// Metrics
	totalRequests   uint64
	successfulReqs  uint64
	failedReqs      uint64
	totalLatency    uint64 // in nanoseconds
	startTime       time.Time
	endTime         time.Time
}

// NewBenchmarkClient creates a new benchmark client
func NewBenchmarkClient(serverAddr string, numClients, requestsPerClient int, duration time.Duration, connectionReuse bool) *BenchmarkClient {
	return &BenchmarkClient{
		serverAddr:        serverAddr,
		numClients:        numClients,
		requestsPerClient: requestsPerClient,
		duration:          duration,
		connectionReuse:   connectionReuse,
	}
}

// RunBenchmark executes the benchmark
func (bc *BenchmarkClient) RunBenchmark() error {
	log.Printf("Starting benchmark: %d clients, %d requests per client, %v duration, reuse=%v",
		bc.numClients, bc.requestsPerClient, bc.duration, bc.connectionReuse)

	bc.startTime = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), bc.duration)
	defer cancel()

	var wg sync.WaitGroup

	// Start metrics reporter
	wg.Add(1)
	go func() {
		defer wg.Done()
		bc.reportMetrics(ctx)
	}()

	// Start client goroutines
	for i := 0; i < bc.numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			bc.runClient(ctx, clientID)
		}(i)
	}

	wg.Wait()
	bc.endTime = time.Now()

	return nil
}

// runClient runs a single client
func (bc *BenchmarkClient) runClient(ctx context.Context, clientID int) {
	var conn net.Conn
	var err error

	// If connection reuse is enabled, create one connection per client
	if bc.connectionReuse {
		conn, err = net.Dial("tcp", bc.serverAddr)
		if err != nil {
			log.Printf("Client %d: Failed to connect: %v", clientID, err)
			return
		}
		defer conn.Close()
	}

	requestCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Limit requests per client if specified
		if bc.requestsPerClient > 0 && requestCount >= bc.requestsPerClient {
			return
		}

		start := time.Now()
		if bc.connectionReuse {
			bc.makeRequest(conn, clientID, requestCount)
		} else {
			bc.makeRequestWithNewConnection(clientID, requestCount)
		}
		latency := time.Since(start).Nanoseconds()
		atomic.AddUint64(&bc.totalLatency, uint64(latency))

		requestCount++
		atomic.AddUint64(&bc.totalRequests, 1)

		// Small delay to avoid overwhelming
		time.Sleep(1 * time.Millisecond)
	}
}

// makeRequest makes a request using an existing connection
func (bc *BenchmarkClient) makeRequest(conn net.Conn, clientID, requestID int) {
	message := fmt.Sprintf("client-%d-request-%d", clientID, requestID)

	// Set timeout
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send request
	if _, err := conn.Write([]byte(message)); err != nil {
		atomic.AddUint64(&bc.failedReqs, 1)
		return
	}

	// Read response
	buf := make([]byte, len(message))
	if _, err := conn.Read(buf); err != nil {
		atomic.AddUint64(&bc.failedReqs, 1)
		return
	}

	// Verify response
	if string(buf) != message {
		atomic.AddUint64(&bc.failedReqs, 1)
		return
	}

	atomic.AddUint64(&bc.successfulReqs, 1)
}

// makeRequestWithNewConnection makes a request with a new connection
func (bc *BenchmarkClient) makeRequestWithNewConnection(clientID, requestID int) {
	conn, err := net.Dial("tcp", bc.serverAddr)
	if err != nil {
		atomic.AddUint64(&bc.failedReqs, 1)
		return
	}
	defer conn.Close()

	bc.makeRequest(conn, clientID, requestID)
}

// reportMetrics periodically reports benchmark metrics
func (bc *BenchmarkClient) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastTotal uint64
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			total := atomic.LoadUint64(&bc.totalRequests)
			success := atomic.LoadUint64(&bc.successfulReqs)
			failed := atomic.LoadUint64(&bc.failedReqs)
			latency := atomic.LoadUint64(&bc.totalLatency)

			now := time.Now()
			duration := now.Sub(lastTime).Seconds()
			rate := float64(total-lastTotal) / duration

			avgLatency := float64(0)
			if total > 0 {
				avgLatency = float64(latency) / float64(total) / 1e6 // Convert to milliseconds
			}

			successRate := float64(0)
			if total > 0 {
				successRate = float64(success) / float64(total) * 100
			}

			log.Printf("Benchmark: Total=%d, Success=%d, Failed=%d, Rate=%.0f req/s, Avg Latency=%.2f ms, Success Rate=%.1f%%",
				total, success, failed, rate, avgLatency, successRate)

			lastTotal = total
			lastTime = now
		}
	}
}

// PrintResults prints the final benchmark results
func (bc *BenchmarkClient) PrintResults() {
	total := atomic.LoadUint64(&bc.totalRequests)
	success := atomic.LoadUint64(&bc.successfulReqs)
	failed := atomic.LoadUint64(&bc.failedReqs)
	latency := atomic.LoadUint64(&bc.totalLatency)

	duration := bc.endTime.Sub(bc.startTime)
	rate := float64(total) / duration.Seconds()

	avgLatency := float64(0)
	if total > 0 {
		avgLatency = float64(latency) / float64(total) / 1e6 // Convert to milliseconds
	}

	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Successful Requests: %d\n", success)
	fmt.Printf("Failed Requests: %d\n", failed)
	fmt.Printf("Request Rate: %.0f req/s\n", rate)
	fmt.Printf("Average Latency: %.2f ms\n", avgLatency)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	fmt.Printf("Connection Reuse: %v\n", bc.connectionReuse)

	// Performance evaluation
	bc.evaluatePerformance(rate, successRate, avgLatency)
}

// evaluatePerformance evaluates if performance targets are met
func (bc *BenchmarkClient) evaluatePerformance(rate, successRate, avgLatency float64) {
	fmt.Printf("\n=== Performance Evaluation ===\n")

	// Target: Connection reuse rate >95% (measured by success rate when reusing)
	if bc.connectionReuse {
		if successRate >= 95.0 {
			fmt.Printf("✅ Connection Reuse Success Rate: %.2f%% (target: >95%%)\n", successRate)
		} else {
			fmt.Printf("❌ Connection Reuse Success Rate: %.2f%% (target: >95%%)\n", successRate)
		}
	}

	// Target: Connection establishment time <10ms (measured by average latency)
	if avgLatency <= 10.0 {
		fmt.Printf("✅ Average Latency: %.2f ms (target: <10ms)\n", avgLatency)
	} else {
		fmt.Printf("❌ Average Latency: %.2f ms (target: <10ms)\n", avgLatency)
	}

	// Target: Request rate (depends on connection reuse)
	targetRate := 1000.0 // 1K req/s baseline
	if bc.connectionReuse {
		targetRate = 5000.0 // 5K req/s with connection reuse
	}

	if rate >= targetRate {
		fmt.Printf("✅ Request Rate: %.0f req/s (target: >%.0f req/s)\n", rate, targetRate)
	} else {
		fmt.Printf("❌ Request Rate: %.0f req/s (target: >%.0f req/s)\n", rate, targetRate)
	}

	// Memory usage check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Memory Usage: %d KB\n", m.Alloc/1024)
	fmt.Printf("GC Cycles: %d\n", m.NumGC)
	fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())

	// Overall assessment
	passed := 0
	total := 3

	if bc.connectionReuse && successRate >= 95.0 {
		passed++
	} else if !bc.connectionReuse {
		passed++ // Skip this check for non-reuse tests
	}

	if avgLatency <= 10.0 {
		passed++
	}

	if rate >= targetRate {
		passed++
	}

	if passed == total {
		fmt.Printf("\n🎉 BENCHMARK PASSED: All performance targets met!\n")
	} else {
		fmt.Printf("\n⚠️  BENCHMARK PARTIAL: %d/%d targets met. Consider optimization.\n", passed, total)
	}

	// Recommendations
	fmt.Printf("\n=== Recommendations ===\n")
	if rate < targetRate {
		fmt.Printf("- Increase connection pool size\n")
		fmt.Printf("- Optimize connection establishment\n")
		fmt.Printf("- Consider connection keep-alive settings\n")
	}
	if avgLatency > 10.0 {
		fmt.Printf("- Reduce network latency\n")
		fmt.Printf("- Optimize request processing\n")
		fmt.Printf("- Check for network congestion\n")
	}
	if bc.connectionReuse && successRate < 95.0 {
		fmt.Printf("- Check connection health monitoring\n")
		fmt.Printf("- Optimize connection lifecycle management\n")
		fmt.Printf("- Review error handling logic\n")
	}
}

// LoadTestClient performs load testing on the connection pool
type LoadTestClient struct {
	serverAddr string
	maxClients int
	rampUpTime time.Duration
	testDuration time.Duration
	rampDownTime time.Duration

	// Metrics
	currentClients uint64
	totalRequests  uint64
	successfulReqs uint64
	failedReqs     uint64
}

// NewLoadTestClient creates a new load test client
func NewLoadTestClient(serverAddr string, maxClients int, rampUpTime, testDuration, rampDownTime time.Duration) *LoadTestClient {
	return &LoadTestClient{
		serverAddr:   serverAddr,
		maxClients:   maxClients,
		rampUpTime:   rampUpTime,
		testDuration: testDuration,
		rampDownTime: rampDownTime,
	}
}

// RunLoadTest executes the load test
func (ltc *LoadTestClient) RunLoadTest() error {
	log.Printf("Starting load test: max %d clients, ramp-up %v, test %v, ramp-down %v",
		ltc.maxClients, ltc.rampUpTime, ltc.testDuration, ltc.rampDownTime)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start metrics reporter
	wg.Add(1)
	go func() {
		defer wg.Done()
		ltc.reportMetrics(ctx)
	}()

	// Ramp-up phase
	log.Println("Starting ramp-up phase...")
	rampUpInterval := ltc.rampUpTime / time.Duration(ltc.maxClients)

	for i := 0; i < ltc.maxClients; i++ {
		select {
		case <-ctx.Done():
			goto cleanup
		default:
		}

		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			atomic.AddUint64(&ltc.currentClients, 1)
			defer atomic.AddUint64(&ltc.currentClients, ^uint64(0))
			ltc.runLoadTestClient(ctx, clientID)
		}(i)

		time.Sleep(rampUpInterval)
	}

	// Steady state phase
	log.Println("Entering steady state phase...")
	time.Sleep(ltc.testDuration)

	// Ramp-down phase
	log.Println("Starting ramp-down phase...")
	cancel() // Signal all clients to stop

cleanup:
	wg.Wait()
	log.Println("Load test completed")

	return nil
}

// runLoadTestClient runs a single load test client
func (ltc *LoadTestClient) runLoadTestClient(ctx context.Context, clientID int) {
	conn, err := net.Dial("tcp", ltc.serverAddr)
	if err != nil {
		log.Printf("Load test client %d: Failed to connect: %v", clientID, err)
		return
	}
	defer conn.Close()

	requestID := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		message := fmt.Sprintf("load-client-%d-request-%d", clientID, requestID)

		// Set timeout
		conn.SetDeadline(time.Now().Add(5 * time.Second))

		// Send request
		if _, err := conn.Write([]byte(message)); err != nil {
			atomic.AddUint64(&ltc.failedReqs, 1)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Read response
		buf := make([]byte, len(message))
		if _, err := conn.Read(buf); err != nil {
			atomic.AddUint64(&ltc.failedReqs, 1)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Verify response
		if string(buf) == message {
			atomic.AddUint64(&ltc.successfulReqs, 1)
		} else {
			atomic.AddUint64(&ltc.failedReqs, 1)
		}

		atomic.AddUint64(&ltc.totalRequests, 1)
		requestID++

		// Small delay between requests
		time.Sleep(10 * time.Millisecond)
	}
}

// reportMetrics periodically reports load test metrics
func (ltc *LoadTestClient) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			clients := atomic.LoadUint64(&ltc.currentClients)
			total := atomic.LoadUint64(&ltc.totalRequests)
			success := atomic.LoadUint64(&ltc.successfulReqs)
			failed := atomic.LoadUint64(&ltc.failedReqs)

			successRate := float64(0)
			if total > 0 {
				successRate = float64(success) / float64(total) * 100
			}

			log.Printf("Load Test: Clients=%d, Total=%d, Success=%d, Failed=%d, Success Rate=%.1f%%",
				clients, total, success, failed, successRate)
		}
	}
}

// PrintLoadTestResults prints the final load test results
func (ltc *LoadTestClient) PrintLoadTestResults() {
	total := atomic.LoadUint64(&ltc.totalRequests)
	success := atomic.LoadUint64(&ltc.successfulReqs)
	failed := atomic.LoadUint64(&ltc.failedReqs)

	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Max Clients: %d\n", ltc.maxClients)
	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Successful Requests: %d\n", success)
	fmt.Printf("Failed Requests: %d\n", failed)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)

	// Evaluate load test performance
	fmt.Printf("\n=== Load Test Evaluation ===\n")
	if successRate >= 99.0 {
		fmt.Printf("✅ Success Rate: %.2f%% (target: >99%%)\n", successRate)
	} else {
		fmt.Printf("❌ Success Rate: %.2f%% (target: >99%%)\n", successRate)
	}

	if total > uint64(ltc.maxClients*10) { // At least 10 requests per client
		fmt.Printf("✅ Request Volume: %d requests (sufficient load)\n", total)
	} else {
		fmt.Printf("⚠️  Request Volume: %d requests (may need more load)\n", total)
	}
}