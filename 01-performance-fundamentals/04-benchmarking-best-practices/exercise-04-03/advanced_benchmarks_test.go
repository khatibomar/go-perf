package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Global variables for benchmark configuration
var (
	conditionalBench = &ConditionalBenchmark{
		SkipShort:    false,
		SkipLong:     false,
		SkipMemory:   false,
		SkipNetwork:  false,
		SkipDatabase: false,
		MinCPUs:      1,
		MinMemory:    1024 * 1024, // 1MB
	}
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Setup
	env, err := SetupEnvironment()
	if err != nil {
		fmt.Printf("Failed to setup environment: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown
	if err := env.TeardownEnvironment(); err != nil {
		fmt.Printf("Failed to teardown environment: %v\n", err)
	}

	os.Exit(code)
}

// BenchmarkCustomMetrics demonstrates custom metric collection
func BenchmarkCustomMetrics(b *testing.B) {
	metrics := NewMetricsCollector()
	compute := &ComputeWorkload{}

	b.ReportAllocs()
	b.ResetTimer()

	var totalPrimes int64
	var totalComputeTime time.Duration

	for i := 0; i < b.N; i++ {
		start := time.Now()
		primes := compute.PrimeCalculation(1000)
		computeTime := time.Since(start)

		totalPrimes += int64(len(primes))
		totalComputeTime += computeTime

		metrics.RecordOperation(computeTime, true)
	}

	// Report custom metrics
	b.ReportMetric(float64(totalPrimes)/float64(b.N), "primes/op")
	b.ReportMetric(float64(totalComputeTime.Nanoseconds())/float64(b.N), "compute_ns/op")
	b.ReportMetric(float64(totalPrimes)/b.Elapsed().Seconds(), "primes/sec")

	ops, avgLatency, errorRate, throughput := metrics.GetMetrics()
	b.ReportMetric(float64(ops), "total_ops")
	b.ReportMetric(float64(avgLatency.Nanoseconds()), "avg_latency_ns")
	b.ReportMetric(errorRate*100, "error_rate_%")
	b.ReportMetric(throughput, "throughput_ops/sec")
}

// BenchmarkConditionalExecution demonstrates conditional benchmark execution
func BenchmarkConditionalExecution(b *testing.B) {
	tests := []struct {
		name      string
		benchType string
		isShort   bool
		workFunc  func()
	}{
		{
			name:      "FastOperation",
			benchType: "cpu",
			isShort:   true,
			workFunc: func() {
				_ = math.Sqrt(float64(42))
			},
		},
		{
			name:      "SlowOperation",
			benchType: "cpu",
			isShort:   false,
			workFunc: func() {
				compute := &ComputeWorkload{}
				_ = compute.FibonacciRecursive(30)
			},
		},
		{
			name:      "MemoryIntensive",
			benchType: "memory",
			isShort:   false,
			workFunc: func() {
				memory := &MemoryWorkload{}
				_ = memory.AllocateSlices(100, 1000)
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			if !conditionalBench.ShouldRun(tt.benchType, tt.isShort) {
				b.Skip("Skipping based on conditional configuration")
			}

			if testing.Short() && !tt.isShort {
				b.Skip("Skipping long test in short mode")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tt.workFunc()
			}
		})
	}
}

// BenchmarkSetupTeardown demonstrates proper setup and teardown
func BenchmarkSetupTeardown(b *testing.B) {
	tests := []struct {
		name string
		size int
	}{
		{"Small", 100},
		{"Medium", 1000},
		{"Large", 10000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Setup phase - not measured
			data := GenerateTestData(tt.size)
			compute := &ComputeWorkload{}

			b.ResetTimer() // Start measuring from here

			for i := 0; i < b.N; i++ {
				b.StopTimer() // Stop timer for setup
				testData := make([]int, len(data))
				copy(testData, data)
				b.StartTimer() // Resume timer for actual work

				// Actual work being measured
				_ = compute.PrimeCalculation(testData[0])
			}

			// Teardown happens automatically
		})
	}
}

// BenchmarkDatabaseOperations demonstrates database benchmarking
func BenchmarkDatabaseOperations(b *testing.B) {
	if !conditionalBench.ShouldRun("database", false) {
		b.Skip("Database benchmarks disabled")
	}

	// Setup database
	db, err := NewDatabaseWorkload()
	if err != nil {
		b.Fatalf("Failed to setup database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name  string
		limit int
	}{
		{"Query10", 10},
		{"Query50", 50},
		{"Query100", 100},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := db.QueryUsers(tt.limit)
				if err != nil {
					b.Fatalf("Query failed: %v", err)
				}
			}
		})
	}

	// Test CRUD operations
	b.Run("CRUD", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			user := User{
				Name:  fmt.Sprintf("BenchUser%d", i),
				Email: fmt.Sprintf("bench%d@example.com", i),
			}

			// Insert
			err := db.InsertUser(user)
			if err != nil {
				b.Fatalf("Insert failed: %v", err)
			}

			// Update
			user.Name = fmt.Sprintf("UpdatedUser%d", i)
			err = db.UpdateUser(user)
			if err != nil {
				b.Fatalf("Update failed: %v", err)
			}

			// Delete
			err = db.DeleteUser(user.ID)
			if err != nil {
				b.Fatalf("Delete failed: %v", err)
			}
		}
	})
}

// BenchmarkHTTPOperations demonstrates HTTP benchmarking
func BenchmarkHTTPOperations(b *testing.B) {
	if !conditionalBench.ShouldRun("network", false) {
		b.Skip("Network benchmarks disabled")
	}

	// Setup HTTP server
	httpWorkload := NewHTTPWorkload()
	defer httpWorkload.Close()

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "GetUsers",
			fn: func() error {
				resp, err := httpWorkload.GetUsers()
				if err != nil {
					return err
				}
				resp.Body.Close()
				return nil
			},
		},
		{
			name: "CreateUser",
			fn: func() error {
				resp, err := httpWorkload.CreateUser()
				if err != nil {
					return err
				}
				resp.Body.Close()
				return nil
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()

			var errors int64
			for i := 0; i < b.N; i++ {
				if err := tt.fn(); err != nil {
					errors++
				}
			}

			// Report error rate
			errorRate := float64(errors) / float64(b.N) * 100
			b.ReportMetric(errorRate, "error_rate_%")
		})
	}
}

// BenchmarkCachePerformance demonstrates cache benchmarking
func BenchmarkCachePerformance(b *testing.B) {
	cacheSizes := []int{100, 500, 1000, 5000}

	for _, size := range cacheSizes {
		b.Run(fmt.Sprintf("CacheSize%d", size), func(b *testing.B) {
			cache := NewLRUCache(size)

			// Pre-populate cache
			for i := 0; i < size/2; i++ {
				cache.Put(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key%d", i%size)

				// Mix of gets and puts
				if i%3 == 0 {
					cache.Put(key, fmt.Sprintf("value%d", i))
				} else {
					cache.Get(key)
				}
			}

			// Report cache hit rate
			hitRate := cache.GetHitRate() * 100
			b.ReportMetric(hitRate, "hit_rate_%")
		})
	}
}

// BenchmarkConcurrencyPatterns demonstrates concurrent benchmarking
func BenchmarkConcurrencyPatterns(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}
	workItems := 1000

	workFunc := func(item int) int {
		// Simulate some work
		result := 0
		for i := 0; i < item%100+1; i++ {
			result += i
		}
		return result
	}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers%d", workers), func(b *testing.B) {
			concurrency := &ConcurrencyWorkload{}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				results := concurrency.ParallelWork(workers, workItems, workFunc)
				if len(results) != workItems {
					b.Fatalf("Expected %d results, got %d", workItems, len(results))
				}
			}

			// Report efficiency metric
			efficiency := float64(workItems) / float64(workers)
			b.ReportMetric(efficiency, "items_per_worker")
		})
	}
}

// BenchmarkChannelThroughput demonstrates channel performance benchmarking
func BenchmarkChannelThroughput(b *testing.B) {
	bufferSizes := []int{0, 1, 10, 100, 1000}
	messageCount := 10000

	for _, bufferSize := range bufferSizes {
		b.Run(fmt.Sprintf("Buffer%d", bufferSize), func(b *testing.B) {
			concurrency := &ConcurrencyWorkload{}

			b.ResetTimer()

			var totalDuration time.Duration
			for i := 0; i < b.N; i++ {
				duration := concurrency.ChannelThroughput(bufferSize, messageCount)
				totalDuration += duration
			}

			// Report throughput metrics
			avgDuration := totalDuration / time.Duration(b.N)
			throughput := float64(messageCount) / avgDuration.Seconds()
			b.ReportMetric(throughput, "messages/sec")
			b.ReportMetric(float64(avgDuration.Nanoseconds()), "avg_duration_ns")
		})
	}
}

// BenchmarkMemoryAllocation demonstrates memory allocation benchmarking
func BenchmarkMemoryAllocation(b *testing.B) {
	if !conditionalBench.ShouldRun("memory", false) {
		b.Skip("Memory benchmarks disabled")
	}

	allocationSizes := []int{100, 1000, 10000, 100000}

	for _, size := range allocationSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			memory := &MemoryWorkload{}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Test slice allocation
				slices := memory.AllocateSlices(10, size/10)

				// Test map allocation
				mapData := memory.AllocateMap(size / 100)

				// Prevent optimization
				runtime.KeepAlive(slices)
				runtime.KeepAlive(mapData)
			}
		})
	}
}

// BenchmarkGarbageCollection demonstrates GC impact benchmarking
func BenchmarkGarbageCollection(b *testing.B) {
	if !conditionalBench.ShouldRun("memory", false) {
		b.Skip("Memory benchmarks disabled")
	}

	tests := []struct {
		name    string
		allocFn func()
		forceGC bool
	}{
		{
			name: "NoGC",
			allocFn: func() {
				data := make([]byte, 1024)
				runtime.KeepAlive(data)
			},
			forceGC: false,
		},
		{
			name: "WithGC",
			allocFn: func() {
				data := make([]byte, 1024)
				runtime.KeepAlive(data)
			},
			forceGC: true,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var gcBefore, gcAfter runtime.MemStats
			runtime.ReadMemStats(&gcBefore)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tt.allocFn()

				if tt.forceGC && i%100 == 0 {
					runtime.GC()
				}
			}

			runtime.ReadMemStats(&gcAfter)

			// Report GC metrics
			gcCycles := gcAfter.NumGC - gcBefore.NumGC
			b.ReportMetric(float64(gcCycles), "gc_cycles")
			b.ReportMetric(float64(gcAfter.PauseTotalNs-gcBefore.PauseTotalNs), "gc_pause_ns")
		})
	}
}

// BenchmarkCryptoOperations demonstrates crypto benchmarking
func BenchmarkCryptoOperations(b *testing.B) {
	crypto := &CryptoWorkload{}
	dataSizes := []int{1024, 4096, 16384, 65536}

	for _, size := range dataSizes {
		b.Run(fmt.Sprintf("DataSize%d", size), func(b *testing.B) {
			// Generate test data once
			data, err := crypto.GenerateRandomData(size)
			if err != nil {
				b.Fatalf("Failed to generate data: %v", err)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := crypto.HashData(data, 100)
				runtime.KeepAlive(result)
			}

			// Report throughput
			bytesPerSec := float64(size*b.N) / b.Elapsed().Seconds()
			b.ReportMetric(bytesPerSec, "bytes/sec")
		})
	}
}

// BenchmarkParametric demonstrates parametric benchmarking
func BenchmarkParametric(b *testing.B) {
	parametric := NewParametricBenchmark()

	// Add parameters
	parametric.AddParameter("size", []interface{}{100, 500, 1000})
	parametric.AddParameter("workers", []interface{}{1, 2, 4})
	parametric.AddParameter("algorithm", []interface{}{"iterative", "recursive"})

	combinations := parametric.GenerateParameterCombinations()

	for i, params := range combinations {
		name := fmt.Sprintf("Combo%d", i)
		for k, v := range params {
			name += fmt.Sprintf("_%s_%v", k, v)
		}

		b.Run(name, func(b *testing.B) {
			size := params["size"].(int)
			workers := params["workers"].(int)
			algorithm := params["algorithm"].(string)

			compute := &ComputeWorkload{}

			b.ResetTimer()

			for j := 0; j < b.N; j++ {
				if algorithm == "iterative" {
					result := compute.FibonacciIterative(size % 50) // Limit to prevent overflow
					runtime.KeepAlive(result)
				} else {
					result := compute.FibonacciRecursive(size % 30) // Limit for performance
					runtime.KeepAlive(result)
				}
			}

			// Report parameter values as metrics
			b.ReportMetric(float64(size), "param_size")
			b.ReportMetric(float64(workers), "param_workers")
		})
	}
}

// BenchmarkContextual demonstrates context-aware benchmarking
func BenchmarkContextual(b *testing.B) {
	timeouts := []time.Duration{
		time.Millisecond * 100,
		time.Millisecond * 500,
		time.Second * 1,
	}

	for _, timeout := range timeouts {
		b.Run(fmt.Sprintf("Timeout%v", timeout), func(b *testing.B) {
			contextual := NewContextualBenchmark(timeout)
			defer contextual.Cancel()

			b.ResetTimer()

			var timeouts int64
			var completions int64

			for i := 0; i < b.N; i++ {
				err := contextual.RunWithContext(func(ctx context.Context) error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Millisecond * 50): // Simulate work
						return nil
					}
				})

				if err != nil {
					timeouts++
				} else {
					completions++
				}
			}

			// Report timeout metrics
			timeoutRate := float64(timeouts) / float64(b.N) * 100
			completionRate := float64(completions) / float64(b.N) * 100
			b.ReportMetric(timeoutRate, "timeout_rate_%")
			b.ReportMetric(completionRate, "completion_rate_%")
		})
	}
}

// BenchmarkScaling demonstrates scaling analysis
func BenchmarkScaling(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			compute := &ComputeWorkload{}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				primes := compute.PrimeCalculation(size)
				runtime.KeepAlive(primes)
			}

			// Report scaling metrics
			opsPerSecond := float64(b.N) / b.Elapsed().Seconds()
			nsPerElement := float64(b.Elapsed().Nanoseconds()) / float64(b.N*size)
			b.ReportMetric(opsPerSecond, "ops/sec")
			b.ReportMetric(nsPerElement, "ns/element")
			b.ReportMetric(float64(size), "input_size")
		})
	}
}

// BenchmarkErrorHandling demonstrates error handling in benchmarks
func BenchmarkErrorHandling(b *testing.B) {
	errorRates := []float64{0.0, 0.1, 0.5, 1.0}

	for _, errorRate := range errorRates {
		b.Run(fmt.Sprintf("ErrorRate%.1f", errorRate), func(b *testing.B) {
			var errors int64
			var successes int64

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Simulate operation with configurable error rate
				if float64(i%100)/100.0 < errorRate {
					// Simulate error
					errors++
					time.Sleep(time.Microsecond * 10) // Error handling overhead
				} else {
					// Simulate success
					successes++
					time.Sleep(time.Microsecond * 5) // Normal operation
				}
			}

			// Report error metrics
			actualErrorRate := float64(errors) / float64(b.N) * 100
			successRate := float64(successes) / float64(b.N) * 100
			b.ReportMetric(actualErrorRate, "error_rate_%")
			b.ReportMetric(successRate, "success_rate_%")
			b.ReportMetric(float64(errors), "total_errors")
		})
	}
}

// BenchmarkResourceUtilization demonstrates resource monitoring
func BenchmarkResourceUtilization(b *testing.B) {
	var startMem, endMem runtime.MemStats
	runtime.ReadMemStats(&startMem)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Allocate some memory
		data := make([]byte, 1024)

		// Do some CPU work
		for j := 0; j < 100; j++ {
			data[j%len(data)] = byte(i + j)
		}

		runtime.KeepAlive(data)
	}

	runtime.ReadMemStats(&endMem)

	// Report resource utilization
	memoryGrowth := endMem.Alloc - startMem.Alloc
	gcCycles := endMem.NumGC - startMem.NumGC
	mallocCount := endMem.Mallocs - startMem.Mallocs

	b.ReportMetric(float64(memoryGrowth), "memory_growth_bytes")
	b.ReportMetric(float64(gcCycles), "gc_cycles")
	b.ReportMetric(float64(mallocCount), "malloc_count")
	b.ReportMetric(float64(runtime.NumGoroutine()), "goroutines")
	b.ReportMetric(float64(runtime.NumCPU()), "cpu_count")
}

// BenchmarkComparative demonstrates comparative benchmarking
func BenchmarkComparative(b *testing.B) {
	implementations := map[string]func(int) int{
		"Iterative": func(n int) int {
			compute := &ComputeWorkload{}
			return compute.FibonacciIterative(n)
		},
		"Recursive": func(n int) int {
			compute := &ComputeWorkload{}
			return compute.FibonacciRecursive(n)
		},
	}

	inputSizes := []int{10, 20, 25}

	for name, impl := range implementations {
		for _, size := range inputSizes {
			b.Run(fmt.Sprintf("%s/Size%d", name, size), func(b *testing.B) {
				if name == "Recursive" && size > 30 {
					b.Skip("Skipping slow recursive implementation for large inputs")
				}

				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					result := impl(size)
					runtime.KeepAlive(result)
				}

				// Report implementation-specific metrics
				b.ReportMetric(float64(size), "input_size")
				if strings.Contains(name, "Recursive") {
					b.ReportMetric(math.Pow(2, float64(size)), "estimated_calls")
				} else {
					b.ReportMetric(float64(size), "estimated_calls")
				}
			})
		}
	}
}

// BenchmarkRegressionDetection demonstrates regression detection
func BenchmarkRegressionDetection(b *testing.B) {
	// Simulate baseline and current results
	baseline := map[string]BenchmarkResult{
		"Operation1": {
			Name:        "Operation1",
			Iterations:  1000,
			NsPerOp:     1000,
			BytesPerOp:  100,
			AllocsPerOp: 1,
		},
		"Operation2": {
			Name:        "Operation2",
			Iterations:  500,
			NsPerOp:     2000,
			BytesPerOp:  200,
			AllocsPerOp: 2,
		},
	}

	current := map[string]BenchmarkResult{
		"Operation1": {
			Name:        "Operation1",
			Iterations:  1000,
			NsPerOp:     1200, // 20% slower
			BytesPerOp:  100,
			AllocsPerOp: 1,
		},
		"Operation2": {
			Name:        "Operation2",
			Iterations:  500,
			NsPerOp:     1800, // 10% faster
			BytesPerOp:  250,  // 25% more memory
			AllocsPerOp: 2,
		},
	}

	regression := &PerformanceRegression{
		Baseline: baseline,
		Current:  current,
	}

	b.Run("RegressionAnalysis", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			regressions := regression.DetectRegressions(0.15) // 15% threshold
			runtime.KeepAlive(regressions)
		}

		// Report regression findings
		regressions := regression.DetectRegressions(0.15)
		b.ReportMetric(float64(len(regressions)), "regressions_found")

		for i, reg := range regressions {
			if i < 5 { // Limit to first 5 regressions
				b.Logf("Regression %d: %s", i+1, reg)
			}
		}
	})
}

// BenchmarkSystemRequirements demonstrates system requirement checking
func BenchmarkSystemRequirements(b *testing.B) {
	requirements := []ConditionalBenchmark{
		{MinCPUs: 1, MinMemory: 1024},
		{MinCPUs: 2, MinMemory: 1024 * 1024},
		{MinCPUs: 4, MinMemory: 1024 * 1024 * 1024},
		{MinCPUs: 8, MinMemory: 1024 * 1024 * 1024 * 2},
	}

	for i, req := range requirements {
		b.Run(fmt.Sprintf("Requirement%d", i), func(b *testing.B) {
			if err := req.CheckSystemRequirements(); err != nil {
				b.Skipf("System requirements not met: %v", err)
			}

			b.ResetTimer()

			for j := 0; j < b.N; j++ {
				// Simulate work that requires specific resources
				data := make([][]byte, req.MinCPUs)
				for k := range data {
					data[k] = make([]byte, req.MinMemory/int64(len(data)))
				}
				runtime.KeepAlive(data)
			}

			// Report system metrics
			b.ReportMetric(float64(req.MinCPUs), "required_cpus")
			b.ReportMetric(float64(req.MinMemory), "required_memory_bytes")
			b.ReportMetric(float64(runtime.NumCPU()), "available_cpus")
		})
	}
}

// Example of a benchmark that uses environment variables for configuration
func BenchmarkConfigurable(b *testing.B) {
	// Read configuration from environment
	iterations := 1000
	if env := os.Getenv("BENCH_ITERATIONS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			iterations = val
		}
	}

	size := 100
	if env := os.Getenv("BENCH_SIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			size = val
		}
	}

	skipSlow := os.Getenv("BENCH_SKIP_SLOW") == "true"

	b.Run("ConfigurableTest", func(b *testing.B) {
		if skipSlow && size > 1000 {
			b.Skip("Skipping slow test due to configuration")
		}

		compute := &ComputeWorkload{}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for j := 0; j < iterations; j++ {
				result := compute.PrimeCalculation(size)
				runtime.KeepAlive(result)
			}
		}

		// Report configuration
		b.ReportMetric(float64(iterations), "config_iterations")
		b.ReportMetric(float64(size), "config_size")
		if skipSlow {
			b.ReportMetric(1, "config_skip_slow")
		} else {
			b.ReportMetric(0, "config_skip_slow")
		}
	})
}
