package benchmarks

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Queue interface for benchmarking
type Queue interface {
	Enqueue(interface{}) bool
	Dequeue() (interface{}, bool)
	IsEmpty() bool
	Size() interface{}
}

// BenchmarkConfig contains configuration for benchmarks
type BenchmarkConfig struct {
	Duration          time.Duration
	WarmupDuration    time.Duration
	Producers         int
	Consumers         int
	Workload          WorkloadType
	ValueRange        int
	BatchSize         int
	MeasureLatency    bool
	// Additional fields for test compatibility
	WorkloadType      string
	NumProducers      int
	NumConsumers      int
	OperationsCount   int
	ProducerRatio     float64
	CollectLatency    bool
	CollectMemory     bool
	CollectContention bool
}

// WorkloadType defines different workload patterns
type WorkloadType int

const (
	Balanced WorkloadType = iota // Equal producers and consumers
	ProducerHeavy                // More producers than consumers
	ConsumerHeavy                // More consumers than producers
	Bursty                       // Alternating bursts of production and consumption
	Steady                       // Constant rate operations
)

// DefaultBenchmarkConfig returns a default benchmark configuration
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		Duration:       30 * time.Second,
		WarmupDuration: 5 * time.Second,
		Producers:      runtime.NumCPU() / 2,
		Consumers:      runtime.NumCPU() / 2,
		Workload:       Balanced,
		ValueRange:     1000000,
		BatchSize:      1,
		MeasureLatency: true,
	}
}

// BenchmarkResult contains the results of a benchmark
type BenchmarkResult struct {
	Name               string
	Duration           time.Duration
	TotalOperations    int64
	EnqueueOperations  int64
	DequeueOperations  int64
	Throughput         float64
	EnqueueThroughput  float64
	DequeueThroughput  float64
	LatencyStats       LatencyStats
	MemoryStats        MemoryStats
	CPUUtilization     float64
	ContentionMetrics  ContentionMetrics
	ContentionStats    ContentionStats
}

// LatencyStats contains latency statistics
type LatencyStats struct {
	Mean   time.Duration
	Median time.Duration
	P50    time.Duration
	P95    time.Duration
	P99    time.Duration
	P999   time.Duration
	Min    time.Duration
	Max    time.Duration
	StdDev time.Duration
}

// MemoryStats contains memory usage statistics
type MemoryStats struct {
	AllocBytes      uint64
	TotalAllocBytes uint64
	SysBytes        uint64
	NumGC           uint32
	GCPauses        time.Duration
}

// ContentionMetrics contains contention-related metrics
type ContentionMetrics struct {
	Retries         int64
	BackoffTime     time.Duration
	FailedEnqueues  int64
	FailedDequeues  int64
	ContentionStats ContentionStats
}

// ContentionStats contains detailed contention statistics
type ContentionStats struct {
	FailedEnqueues int64
	FailedDequeues int64
	Retries        int64
}

// ComparisonResult represents the result of comparing two benchmark results
type ComparisonResult struct {
	ThroughputRatio float64
	LatencyRatio    float64
	MemoryRatio     float64
}

// PerformanceBenchmark runs performance benchmarks on queue implementations
type PerformanceBenchmark struct {
	config BenchmarkConfig
}

// NewPerformanceBenchmark creates a new performance benchmark
func NewPerformanceBenchmark(config ...BenchmarkConfig) *PerformanceBenchmark {
	if len(config) > 0 {
		return &PerformanceBenchmark{config: config[0]}
	}
	return &PerformanceBenchmark{config: DefaultBenchmarkConfig()}
}

// RunBenchmark runs a comprehensive benchmark on the queue
func (pb *PerformanceBenchmark) RunBenchmark(queue Queue, config ...BenchmarkConfig) BenchmarkResult {
	// Use provided config or existing config
	if len(config) > 0 {
		pb.config = config[0]
	}
	
	// Set name based on queue type
	name := fmt.Sprintf("%T", queue)
	// Warmup phase
	if pb.config.WarmupDuration > 0 {
		pb.runWarmup(queue)
	}
	
	// Collect initial memory stats
	initialMemStats := pb.collectMemoryStats()
	
	ctx, cancel := context.WithTimeout(context.Background(), pb.config.Duration)
	defer cancel()
	
	var (
		totalOps     int64
		enqueueOps   int64
		dequeueOps   int64
		retries      int64
		failedOps    int64
		latencies    []time.Duration
		latenciesMu  sync.Mutex
	)
	
	var wg sync.WaitGroup
	startTime := time.Now()
	
	// Start producer goroutines
	producers := pb.config.Producers
	if producers == 0 {
		producers = pb.config.NumProducers
	}
	for i := 0; i < producers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pb.runProducer(ctx, queue, id, &totalOps, &enqueueOps, &retries, &failedOps, &latencies, &latenciesMu)
		}(i)
	}
	
	// Start consumer goroutines
	consumers := pb.config.Consumers
	if consumers == 0 {
		consumers = pb.config.NumConsumers
	}
	for i := 0; i < consumers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pb.runConsumer(ctx, queue, id, &totalOps, &dequeueOps, &retries, &failedOps, &latencies, &latenciesMu)
		}(i)
	}
	
	wg.Wait()
	actualDuration := time.Since(startTime)
	
	// Collect final memory stats
	finalMemStats := pb.collectMemoryStats()
	
	// Calculate statistics
	totalOperations := atomic.LoadInt64(&totalOps)
	enqueueOperations := atomic.LoadInt64(&enqueueOps)
	dequeueOperations := atomic.LoadInt64(&dequeueOps)
	
	throughput := float64(totalOperations) / actualDuration.Seconds()
	enqueueThroughput := float64(enqueueOperations) / actualDuration.Seconds()
	dequeueThroughput := float64(dequeueOperations) / actualDuration.Seconds()
	
	// Calculate latency statistics
	latencyStats := pb.calculateLatencyStats(latencies)
	
	// Calculate memory statistics
	memoryStats := MemoryStats{
		AllocBytes:      finalMemStats.Alloc - initialMemStats.Alloc,
		TotalAllocBytes: finalMemStats.TotalAlloc - initialMemStats.TotalAlloc,
		SysBytes:        finalMemStats.Sys - initialMemStats.Sys,
		NumGC:           finalMemStats.NumGC - initialMemStats.NumGC,
		GCPauses:        time.Duration(finalMemStats.PauseTotalNs - initialMemStats.PauseTotalNs),
	}
	
	contentionMetrics := ContentionMetrics{
		Retries:        atomic.LoadInt64(&retries),
		FailedEnqueues: atomic.LoadInt64(&failedOps) / 2, // Approximate split
		FailedDequeues: atomic.LoadInt64(&failedOps) / 2, // Approximate split
		ContentionStats: ContentionStats{
			FailedEnqueues: atomic.LoadInt64(&failedOps) / 2,
			FailedDequeues: atomic.LoadInt64(&failedOps) / 2,
			Retries:        atomic.LoadInt64(&retries),
		},
	}
	
	return BenchmarkResult{
		Name:               name,
		Duration:           actualDuration,
		TotalOperations:    totalOperations,
		EnqueueOperations:  enqueueOperations,
		DequeueOperations:  dequeueOperations,
		Throughput:         throughput,
		EnqueueThroughput:  enqueueThroughput,
		DequeueThroughput:  dequeueThroughput,
		LatencyStats:       latencyStats,
		MemoryStats:        memoryStats,
		ContentionMetrics:  contentionMetrics,
		ContentionStats:    contentionMetrics.ContentionStats,
	}
}

// runWarmup runs a warmup phase to stabilize performance
func (pb *PerformanceBenchmark) runWarmup(queue Queue) {
	ctx, cancel := context.WithTimeout(context.Background(), pb.config.WarmupDuration)
	defer cancel()
	
	var wg sync.WaitGroup
	var dummy int64
	
	// Light warmup load
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					queue.Enqueue(42)
					queue.Dequeue()
					atomic.AddInt64(&dummy, 1)
				}
			}
		}()
	}
	
	wg.Wait()
	runtime.GC() // Force GC after warmup
}

// runProducer runs a producer goroutine for benchmarking
func (pb *PerformanceBenchmark) runProducer(ctx context.Context, queue Queue, id int, totalOps, enqueueOps, retries, failedOps *int64, latencies *[]time.Duration, latenciesMu *sync.Mutex) {
	value := id * 1000000 // Unique values per producer
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for i := 0; i < pb.config.BatchSize; i++ {
				startTime := time.Now()
				success := queue.Enqueue(value)
				latency := time.Since(startTime)
				
				atomic.AddInt64(totalOps, 1)
				
				if success {
					atomic.AddInt64(enqueueOps, 1)
				} else {
					atomic.AddInt64(failedOps, 1)
				}
				
				measureLatency := pb.config.MeasureLatency || pb.config.CollectLatency
				if measureLatency {
					latenciesMu.Lock()
					*latencies = append(*latencies, latency)
					latenciesMu.Unlock()
				}
				
				value++
			}
			
			// Apply workload-specific behavior
			pb.applyWorkloadPattern(id, true)
		}
	}
}

// runConsumer runs a consumer goroutine for benchmarking
func (pb *PerformanceBenchmark) runConsumer(ctx context.Context, queue Queue, id int, totalOps, dequeueOps, retries, failedOps *int64, latencies *[]time.Duration, latenciesMu *sync.Mutex) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for i := 0; i < pb.config.BatchSize; i++ {
				startTime := time.Now()
				_, success := queue.Dequeue()
				latency := time.Since(startTime)
				
				atomic.AddInt64(totalOps, 1)
				
				if success {
					atomic.AddInt64(dequeueOps, 1)
				} else {
					atomic.AddInt64(failedOps, 1)
				}
				
				measureLatency := pb.config.MeasureLatency || pb.config.CollectLatency
				if measureLatency {
					latenciesMu.Lock()
					*latencies = append(*latencies, latency)
					latenciesMu.Unlock()
				}
			}
			
			// Apply workload-specific behavior
			pb.applyWorkloadPattern(id, false)
		}
	}
}

// applyWorkloadPattern applies workload-specific timing patterns
func (pb *PerformanceBenchmark) applyWorkloadPattern(workerID int, isProducer bool) {
	switch pb.config.Workload {
	case Bursty:
		if workerID%10 == 0 {
			time.Sleep(time.Microsecond * 100) // Brief pause
		}
	case Steady:
		time.Sleep(time.Microsecond * 10) // Consistent small delay
	case ProducerHeavy:
		if !isProducer {
			time.Sleep(time.Microsecond * 50) // Slow down consumers
		}
	case ConsumerHeavy:
		if isProducer {
			time.Sleep(time.Microsecond * 50) // Slow down producers
		}
	// Balanced has no additional delays
	}
}

// calculateLatencyStats calculates latency statistics from recorded latencies
// calculateLatencyStats calculates latency statistics from a slice of durations
func calculateLatencyStats(latencies []time.Duration) LatencyStats {
	return (&PerformanceBenchmark{}).calculateLatencyStats(latencies)
}

func (pb *PerformanceBenchmark) calculateLatencyStats(latencies []time.Duration) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}
	
	// Sort latencies for percentile calculations
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	
	n := len(latencies)
	
	// Calculate percentiles
	median := latencies[n/2]
	p50 := latencies[n/2]
	p95 := latencies[int(float64(n)*0.95)]
	p99 := latencies[int(float64(n)*0.99)]
	p999 := latencies[int(float64(n)*0.999)]
	min := latencies[0]
	max := latencies[n-1]
	
	// Calculate mean
	var sum time.Duration
	for _, lat := range latencies {
		sum += lat
	}
	mean := sum / time.Duration(n)
	
	// Calculate standard deviation
	var variance float64
	for _, lat := range latencies {
		diff := float64(lat - mean)
		variance += diff * diff
	}
	variance /= float64(n)
	stdDev := time.Duration(math.Sqrt(variance))
	
	return LatencyStats{
		Mean:   mean,
		Median: median,
		P50:    p50,
		P95:    p95,
		P99:    p99,
		P999:   p999,
		Min:    min,
		Max:    max,
		StdDev: stdDev,
	}
}

// collectMemoryStats collects current memory statistics
func (pb *PerformanceBenchmark) collectMemoryStats() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}

// String returns a string representation of the benchmark result
func (br BenchmarkResult) String() string {
	return fmt.Sprintf(
		"Benchmark Results for %s:\n"+
		"  Duration: %v\n"+
		"  Total Operations: %d\n"+
		"  Enqueue Operations: %d\n"+
		"  Dequeue Operations: %d\n"+
		"  Total Throughput: %.2f ops/sec\n"+
		"  Enqueue Throughput: %.2f ops/sec\n"+
		"  Dequeue Throughput: %.2f ops/sec\n"+
		"  Latency (Mean/P95/P99): %v/%v/%v\n"+
		"  Memory Allocated: %d bytes\n"+
		"  GC Pauses: %v\n"+
		"  Failed Operations: %d\n",
		br.Name,
		br.Duration,
		br.TotalOperations,
		br.EnqueueOperations,
		br.DequeueOperations,
		br.Throughput,
		br.EnqueueThroughput,
		br.DequeueThroughput,
		br.LatencyStats.Mean,
		br.LatencyStats.P95,
		br.LatencyStats.P99,
		br.MemoryStats.AllocBytes,
		br.MemoryStats.GCPauses,
		br.ContentionMetrics.FailedEnqueues+br.ContentionMetrics.FailedDequeues,
	)
}

// CompareBenchmarks compares multiple benchmark results
// CompareResults compares two benchmark results and returns comparison ratios
func CompareResults(baseline, comparison BenchmarkResult) ComparisonResult {
	result := ComparisonResult{}
	
	// Calculate throughput ratio (baseline / comparison)
	if comparison.Throughput > 0 {
		result.ThroughputRatio = baseline.Throughput / comparison.Throughput
	}
	
	// Calculate latency ratio (baseline / comparison)
	if comparison.LatencyStats.Mean > 0 {
		result.LatencyRatio = float64(baseline.LatencyStats.Mean) / float64(comparison.LatencyStats.Mean)
	}
	
	// Calculate memory ratio (baseline / comparison)
	if comparison.MemoryStats.AllocBytes > 0 {
		result.MemoryRatio = float64(baseline.MemoryStats.AllocBytes) / float64(comparison.MemoryStats.AllocBytes)
	}
	
	return result
}

func CompareBenchmarks(results []BenchmarkResult) string {
	if len(results) == 0 {
		return "No benchmark results to compare"
	}
	
	output := "Benchmark Comparison:\n"
	output += fmt.Sprintf("%-20s %12s %12s %12s %12s\n", "Implementation", "Throughput", "P95 Latency", "Memory (MB)", "Failed Ops")
	output += fmt.Sprintf("%s\n", string(make([]byte, 80)))
	
	for _, result := range results {
		memoryMB := float64(result.MemoryStats.AllocBytes) / (1024 * 1024)
		output += fmt.Sprintf("%-20s %12.0f %12v %12.2f %12d\n",
			result.Name,
			result.Throughput,
			result.LatencyStats.P95,
			memoryMB,
			result.ContentionMetrics.FailedEnqueues+result.ContentionMetrics.FailedDequeues,
		)
	}
	
	return output
}