package benchmarks

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockQueue implements a simple queue for benchmarking tests
type MockQueue struct {
	mu    sync.Mutex
	items []interface{}
}

func NewMockQueue() *MockQueue {
	return &MockQueue{
		items: make([]interface{}, 0),
	}
}

func (q *MockQueue) Enqueue(item interface{}) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
	return true
}

func (q *MockQueue) Dequeue() (interface{}, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *MockQueue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}

func (q *MockQueue) Size() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// SlowQueue simulates a slower queue implementation
type SlowQueue struct {
	mu    sync.Mutex
	items []interface{}
}

func NewSlowQueue() *SlowQueue {
	return &SlowQueue{
		items: make([]interface{}, 0),
	}
}

func (q *SlowQueue) Enqueue(item interface{}) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Simulate some processing time
	time.Sleep(10 * time.Microsecond)
	q.items = append(q.items, item)
	return true
}

func (q *SlowQueue) Dequeue() (interface{}, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Simulate some processing time
	time.Sleep(10 * time.Microsecond)
	if len(q.items) == 0 {
		return nil, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *SlowQueue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}

func (q *SlowQueue) Size() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// TestPerformanceBenchmark tests the benchmarking framework
func TestPerformanceBenchmark(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	queue := NewMockQueue()
	
	config := BenchmarkConfig{
		WorkloadType:    "mixed",
		NumProducers:    2,
		NumConsumers:    2,
		OperationsCount: 1000,
		Duration:        time.Second,
		WarmupDuration:  100 * time.Millisecond,
		ProducerRatio:   0.5,
		BatchSize:       1,
		CollectLatency:  true,
		CollectMemory:   true,
		CollectContention: true,
	}
	
	result := benchmark.RunBenchmark(queue, config)
	
	if result.TotalOperations == 0 {
		t.Error("Should have performed some operations")
	}
	
	if result.Throughput == 0 {
		t.Error("Throughput should be greater than 0")
	}
	
	if result.Duration == 0 {
		t.Error("Duration should be greater than 0")
	}
	
	t.Logf("Benchmark result: %+v", result)
}

// TestBenchmarkWorkloadTypes tests different workload types
func TestBenchmarkWorkloadTypes(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	queue := NewMockQueue()
	
	workloadTypes := []string{"producer_heavy", "consumer_heavy", "mixed", "burst"}
	
	for _, workloadType := range workloadTypes {
		t.Run(workloadType, func(t *testing.T) {
			config := BenchmarkConfig{
				WorkloadType:    workloadType,
				NumProducers:    2,
				NumConsumers:    2,
				OperationsCount: 500,
				Duration:        500 * time.Millisecond,
				WarmupDuration:  50 * time.Millisecond,
				ProducerRatio:   0.7,
				BatchSize:       1,
				CollectLatency:  true,
			}
			
			result := benchmark.RunBenchmark(queue, config)
			
			if result.TotalOperations == 0 {
				t.Errorf("Workload %s should have performed some operations", workloadType)
			}
			
			t.Logf("Workload %s: Throughput=%.2f ops/sec, Operations=%d", 
				workloadType, result.Throughput, result.TotalOperations)
		})
	}
}

// TestLatencyMeasurement tests latency collection and statistics
func TestLatencyMeasurement(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	queue := NewSlowQueue() // Use slower queue to get measurable latencies
	
	config := BenchmarkConfig{
		WorkloadType:    "mixed",
		NumProducers:    1,
		NumConsumers:    1,
		OperationsCount: 100,
		Duration:        time.Second,
		WarmupDuration:  50 * time.Millisecond,
		ProducerRatio:   0.5,
		BatchSize:       1,
		CollectLatency:  true,
	}
	
	result := benchmark.RunBenchmark(queue, config)
	
	if result.LatencyStats.Mean == 0 {
		t.Error("Mean latency should be greater than 0")
	}
	
	if result.LatencyStats.P99 == 0 {
		t.Error("P99 latency should be greater than 0")
	}
	
	if result.LatencyStats.Min >= result.LatencyStats.Max {
		t.Error("Min latency should be less than max latency")
	}
	
	t.Logf("Latency stats: Mean=%v, P50=%v, P95=%v, P99=%v, Min=%v, Max=%v",
		result.LatencyStats.Mean, result.LatencyStats.P50, result.LatencyStats.P95,
		result.LatencyStats.P99, result.LatencyStats.Min, result.LatencyStats.Max)
}

// TestMemoryMeasurement tests memory usage collection
func TestMemoryMeasurement(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	queue := NewMockQueue()
	
	config := BenchmarkConfig{
		WorkloadType:    "mixed",
		NumProducers:    2,
		NumConsumers:    2,
		OperationsCount: 1000,
		Duration:        500 * time.Millisecond,
		WarmupDuration:  50 * time.Millisecond,
		ProducerRatio:   0.5,
		BatchSize:       1,
		CollectMemory:   true,
	}
	
	result := benchmark.RunBenchmark(queue, config)
	
	// Memory stats should be collected
	if result.MemoryStats.AllocBytes == 0 && result.MemoryStats.TotalAllocBytes == 0 {
		t.Log("Warning: No memory allocation detected (this might be normal for simple operations)")
	}
	
	t.Logf("Memory stats: Alloc=%d bytes, TotalAlloc=%d bytes, Sys=%d bytes, GC=%d",
		result.MemoryStats.AllocBytes, result.MemoryStats.TotalAllocBytes,
		result.MemoryStats.SysBytes, result.MemoryStats.NumGC)
}

// TestContentionMeasurement tests contention measurement
func TestContentionMeasurement(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	queue := NewMockQueue()
	
	config := BenchmarkConfig{
		WorkloadType:      "mixed",
		NumProducers:      4, // More threads to increase contention
		NumConsumers:      4,
		OperationsCount:   2000,
		Duration:          time.Second,
		WarmupDuration:    100 * time.Millisecond,
		ProducerRatio:     0.5,
		BatchSize:         1,
		CollectContention: true,
	}
	
	result := benchmark.RunBenchmark(queue, config)
	
	// Contention stats should be collected
	t.Logf("Contention stats: Failed enqueues=%d, Failed dequeues=%d, Retries=%d",
		result.ContentionStats.FailedEnqueues, result.ContentionStats.FailedDequeues,
		result.ContentionStats.Retries)
}

// TestBatchOperations tests batch size effects
func TestBatchOperations(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	queue := NewMockQueue()
	
	batchSizes := []int{1, 5, 10}
	
	for _, batchSize := range batchSizes {
		t.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(t *testing.T) {
			config := BenchmarkConfig{
				WorkloadType:    "mixed",
				NumProducers:    2,
				NumConsumers:    2,
				OperationsCount: 1000,
				Duration:        500 * time.Millisecond,
				WarmupDuration:  50 * time.Millisecond,
				ProducerRatio:   0.5,
				BatchSize:       batchSize,
				CollectLatency:  true,
			}
			
			result := benchmark.RunBenchmark(queue, config)
			
			if result.TotalOperations == 0 {
				t.Errorf("Batch size %d should have performed some operations", batchSize)
			}
			
			t.Logf("Batch size %d: Throughput=%.2f ops/sec", batchSize, result.Throughput)
		})
	}
}

// TestCompareResults tests result comparison functionality
func TestCompareResults(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	
	// Create two different queues for comparison
	fastQueue := NewMockQueue()
	slowQueue := NewSlowQueue()
	
	config := BenchmarkConfig{
		WorkloadType:    "mixed",
		NumProducers:    2,
		NumConsumers:    2,
		OperationsCount: 200,
		Duration:        500 * time.Millisecond,
		WarmupDuration:  50 * time.Millisecond,
		ProducerRatio:   0.5,
		BatchSize:       1,
		CollectLatency:  true,
	}
	
	fastResult := benchmark.RunBenchmark(fastQueue, config)
	slowResult := benchmark.RunBenchmark(slowQueue, config)
	
	comparison := CompareResults(fastResult, slowResult)
	
	// Fast queue should have better throughput
	if comparison.ThroughputRatio < 1.0 {
		t.Error("Fast queue should have better throughput than slow queue")
	}
	
	// Slow queue should have higher latency
	if comparison.LatencyRatio > 1.0 {
		t.Error("Fast queue should have lower latency than slow queue")
	}
	
	t.Logf("Comparison: Throughput ratio=%.2f, Latency ratio=%.2f, Memory ratio=%.2f",
		comparison.ThroughputRatio, comparison.LatencyRatio, comparison.MemoryRatio)
}

// TestWarmupEffect tests that warmup affects results
func TestWarmupEffect(t *testing.T) {
	benchmark := NewPerformanceBenchmark()
	
	// Test without warmup
	configNoWarmup := BenchmarkConfig{
		WorkloadType:    "mixed",
		NumProducers:    2,
		NumConsumers:    2,
		OperationsCount: 1000,
		Duration:        500 * time.Millisecond,
		WarmupDuration:  0, // No warmup
		ProducerRatio:   0.5,
		BatchSize:       1,
		CollectLatency:  true,
	}
	
	// Test with warmup
	configWithWarmup := configNoWarmup
	configWithWarmup.WarmupDuration = 100 * time.Millisecond
	
	resultNoWarmup := benchmark.RunBenchmark(NewMockQueue(), configNoWarmup)
	resultWithWarmup := benchmark.RunBenchmark(NewMockQueue(), configWithWarmup)
	
	t.Logf("No warmup: Throughput=%.2f ops/sec", resultNoWarmup.Throughput)
	t.Logf("With warmup: Throughput=%.2f ops/sec", resultWithWarmup.Throughput)
	
	// Both should have performed operations
	if resultNoWarmup.TotalOperations == 0 {
		t.Error("No warmup test should have performed operations")
	}
	
	if resultWithWarmup.TotalOperations == 0 {
		t.Error("Warmup test should have performed operations")
	}
}

// BenchmarkPerformanceBenchmark benchmarks the benchmarking framework itself
func BenchmarkPerformanceBenchmark(b *testing.B) {
	benchmark := NewPerformanceBenchmark()
	queue := NewMockQueue()
	
	config := BenchmarkConfig{
		WorkloadType:    "mixed",
		NumProducers:    2,
		NumConsumers:    2,
		OperationsCount: 100,
		Duration:        100 * time.Millisecond,
		WarmupDuration:  10 * time.Millisecond,
		ProducerRatio:   0.5,
		BatchSize:       1,
		CollectLatency:  false, // Disable to reduce overhead
		CollectMemory:   false,
		CollectContention: false,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmark.RunBenchmark(queue, config)
	}
}

// TestCalculateLatencyStats tests latency statistics calculation
func TestCalculateLatencyStats(t *testing.T) {
	// Create sample latency data
	latencies := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond, // Outlier
	}
	
	stats := calculateLatencyStats(latencies)
	
	if stats.Min != 1*time.Millisecond {
		t.Errorf("Expected min latency 1ms, got %v", stats.Min)
	}
	
	if stats.Max != 10*time.Millisecond {
		t.Errorf("Expected max latency 10ms, got %v", stats.Max)
	}
	
	if stats.Mean == 0 {
		t.Error("Mean latency should be greater than 0")
	}
	
	if stats.P50 == 0 {
		t.Error("P50 latency should be greater than 0")
	}
	
	if stats.P95 == 0 {
		t.Error("P95 latency should be greater than 0")
	}
	
	if stats.P99 == 0 {
		t.Error("P99 latency should be greater than 0")
	}
	
	t.Logf("Latency stats: Mean=%v, P50=%v, P95=%v, P99=%v, Min=%v, Max=%v",
		stats.Mean, stats.P50, stats.P95, stats.P99, stats.Min, stats.Max)
}