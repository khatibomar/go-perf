package benchmarks

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"exercise-03-02/metrics"
	"exercise-03-02/pools"
)

// BenchmarkConfig defines benchmark parameters
type BenchmarkConfig struct {
	NumTasks         int
	TaskDuration     time.Duration
	TaskVariability  float64 // 0.0 = no variability, 1.0 = high variability
	Concurrency      int
	PoolSize         int
	BatchSize        int
	BenchmarkTime    time.Duration
	WarmupTime       time.Duration
	WorkloadType     WorkloadType
	PriorityLevels   int
	IOIntensity      float64 // 0.0 = CPU only, 1.0 = IO only
	MemoryPressure   bool
	Burstiness       float64 // 0.0 = steady, 1.0 = very bursty
}

// WorkloadType defines different types of workloads
type WorkloadType int

const (
	CPUIntensive WorkloadType = iota
	IOBound
	Mixed
	Bursty
	Steady
	HighThroughput
	LowLatency
)

// BenchmarkResult contains benchmark results
type BenchmarkResult struct {
	PoolType            string
	Config              BenchmarkConfig
	Duration            time.Duration
	TasksCompleted      int64
	TasksFailed         int64
	Throughput          float64 // tasks per second
	AverageLatency      time.Duration
	P50Latency          time.Duration
	P95Latency          time.Duration
	P99Latency          time.Duration
	MaxLatency          time.Duration
	MinLatency          time.Duration
	WorkerUtilization   float64
	MemoryUsage         int64
	GoroutineCount      int64
	CPUUsage            float64
	ErrorRate           float64
	ScalingEvents       int64
	WorkStealingSuccess float64
	QueueUtilization    float64
	CustomMetrics       map[string]interface{}
}

// BenchmarkSuite runs comprehensive benchmarks
type BenchmarkSuite struct {
	configs []BenchmarkConfig
	results []BenchmarkResult
	mu      sync.RWMutex
}

// TestWorkItem implements WorkItem for benchmarking
type TestWorkItem struct {
	id        string
	priority  int
	duration  time.Duration
	workType  WorkloadType
	data      []byte
	ctx       context.Context
	submitTime time.Time
	startTime  time.Time
	endTime    time.Time
	error     error
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	return &BenchmarkSuite{
		configs: make([]BenchmarkConfig, 0),
		results: make([]BenchmarkResult, 0),
	}
}

// AddConfig adds a benchmark configuration
func (bs *BenchmarkSuite) AddConfig(config BenchmarkConfig) {
	bs.mu.Lock()
	bs.configs = append(bs.configs, config)
	bs.mu.Unlock()
}

// RunAllBenchmarks runs all configured benchmarks
func (bs *BenchmarkSuite) RunAllBenchmarks() []BenchmarkResult {
	var allResults []BenchmarkResult
	
	for _, config := range bs.configs {
		// Test different pool types
		poolTypes := []string{"fixed", "dynamic", "workstealing", "priority"}
		
		for _, poolType := range poolTypes {
			result := bs.runSingleBenchmark(poolType, config)
			allResults = append(allResults, result)
			
			// Allow GC between benchmarks
			runtime.GC()
			time.Sleep(time.Millisecond * 100)
		}
	}
	
	bs.mu.Lock()
	bs.results = allResults
	bs.mu.Unlock()
	
	return allResults
}

// runSingleBenchmark runs a benchmark for a specific pool type
func (bs *BenchmarkSuite) runSingleBenchmark(poolType string, config BenchmarkConfig) BenchmarkResult {
	// Create metrics collector
	metricsCollector := metrics.NewMetricsCollector(time.Millisecond * 100)
	metricsCollector.Start()
	defer metricsCollector.Stop()
	
	// Create pool based on type
	pool := bs.createPool(poolType, config, metricsCollector)
	if pool == nil {
		return BenchmarkResult{
			PoolType: poolType,
			Config:   config,
			ErrorRate: 1.0,
		}
	}
	
	// Start pool
	var err error
	switch p := pool.(type) {
	case *pools.FixedPool:
		err = p.Start()
		defer p.Shutdown(5 * time.Second)
	case *pools.DynamicPool:
		err = p.Start()
		defer p.Shutdown(5 * time.Second)
	case *pools.WorkStealingPool:
		err = p.Start()
		defer p.Shutdown(5 * time.Second)
	case *pools.PriorityPool:
		err = p.Start()
		defer p.Shutdown(5 * time.Second)
	}
	
	if err != nil {
		return BenchmarkResult{
			PoolType: poolType,
			Config:   config,
			ErrorRate: 1.0,
		}
	}
	
	// Warmup phase
	if config.WarmupTime > 0 {
		bs.runWorkload(pool, config, config.WarmupTime, true)
	}
	
	// Actual benchmark
	startTime := time.Now()
	latencies := bs.runWorkload(pool, config, config.BenchmarkTime, false)
	benchmarkDuration := time.Since(startTime)
	
	// Collect final metrics
	poolMetrics := metricsCollector.GetPoolMetrics()
	
	// Calculate latency percentiles
	p50, p95, p99, min, max := calculateLatencyPercentiles(latencies)
	
	// Get memory and goroutine stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return BenchmarkResult{
		PoolType:            poolType,
		Config:              config,
		Duration:            benchmarkDuration,
		TasksCompleted:      poolMetrics.TasksCompleted,
		TasksFailed:         poolMetrics.TasksFailed,
		Throughput:          poolMetrics.Throughput,
		AverageLatency:      poolMetrics.AverageExecutionTime,
		P50Latency:          p50,
		P95Latency:          p95,
		P99Latency:          p99,
		MaxLatency:          max,
		MinLatency:          min,
		WorkerUtilization:   poolMetrics.WorkerUtilization,
		MemoryUsage:         int64(memStats.Alloc),
		GoroutineCount:      int64(runtime.NumGoroutine()),
		ErrorRate:           poolMetrics.ErrorRate,
		ScalingEvents:       poolMetrics.ScalingEvents,
		WorkStealingSuccess: float64(poolMetrics.WorkStealingSuccesses) / float64(poolMetrics.WorkStealingAttempts),
		CustomMetrics:       poolMetrics.CustomMetrics,
	}
}

// createPool creates a pool based on the specified type
func (bs *BenchmarkSuite) createPool(poolType string, config BenchmarkConfig, metricsCollector *metrics.MetricsCollector) interface{} {
	switch poolType {
	case "fixed":
		return pools.NewFixedPool(config.PoolSize, config.NumTasks)
	case "dynamic":
		return pools.NewDynamicPool(1, config.PoolSize, int(time.Second.Milliseconds()))
	case "workstealing":
		return pools.NewWorkStealingPool(config.PoolSize, config.NumTasks/config.PoolSize, config.NumTasks)
	case "priority":
		return pools.NewPriorityPool(config.PoolSize, []int{config.NumTasks, config.NumTasks, config.NumTasks, config.NumTasks})
	default:
		return nil
	}
}

// runWorkload runs the actual workload
func (bs *BenchmarkSuite) runWorkload(pool interface{}, config BenchmarkConfig, duration time.Duration, isWarmup bool) []time.Duration {
	var latencies []time.Duration
	var latenciesMu sync.Mutex
	
	startTime := time.Now()
	var tasksSubmitted int64
	
	// Create workload generator
	workloadGen := bs.createWorkloadGenerator(config)
	
	// Submit tasks
	for time.Since(startTime) < duration {
		task := workloadGen.NextTask()
		task.submitTime = time.Now()
		
		// Submit to pool
		go func(t *TestWorkItem) {
			submitStart := time.Now()
			
			// Submit task to pool based on type
			switch p := pool.(type) {
			case *pools.FixedPool:
				p.Submit(func() {
					t.Execute()
				})
			case *pools.DynamicPool:
				p.Submit(func() {
					t.Execute()
				})
			case *pools.WorkStealingPool:
				p.Submit(func() {
					t.Execute()
				})
			case *pools.PriorityPool:
				p.Submit(func() {
					t.Execute()
				}, 1) // Normal priority
			}
			
			if !isWarmup {
				latency := time.Since(submitStart)
				latenciesMu.Lock()
				latencies = append(latencies, latency)
				latenciesMu.Unlock()
			}
		}(task)
		
		atomic.AddInt64(&tasksSubmitted, 1)
		
		// Control submission rate based on burstiness
		if config.Burstiness > 0 {
			bs.applyBurstiness(config.Burstiness)
		} else {
			time.Sleep(time.Microsecond * 10) // Steady rate
		}
	}
	
	return latencies
}

// WorkloadGenerator generates test workloads
type WorkloadGenerator struct {
	config BenchmarkConfig
	rng    *rand.Rand
	taskID int64
}

// createWorkloadGenerator creates a workload generator
func (bs *BenchmarkSuite) createWorkloadGenerator(config BenchmarkConfig) *WorkloadGenerator {
	return &WorkloadGenerator{
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NextTask generates the next test task
func (wg *WorkloadGenerator) NextTask() *TestWorkItem {
	atomic.AddInt64(&wg.taskID, 1)
	
	// Calculate task duration with variability
	baseDuration := wg.config.TaskDuration
	variability := wg.config.TaskVariability
	actualDuration := baseDuration
	
	if variability > 0 {
		// Add random variability
		variation := time.Duration(float64(baseDuration) * variability * (wg.rng.Float64()*2 - 1))
		actualDuration = baseDuration + variation
		if actualDuration < 0 {
			actualDuration = baseDuration / 10
		}
	}
	
	// Determine priority
	priority := 1
	if wg.config.PriorityLevels > 1 {
		priority = wg.rng.Intn(wg.config.PriorityLevels) + 1
	}
	
	// Create data payload for memory pressure
	var data []byte
	if wg.config.MemoryPressure {
		size := 1024 + wg.rng.Intn(4096) // 1-5KB
		data = make([]byte, size)
		wg.rng.Read(data)
	}
	
	return &TestWorkItem{
		id:       fmt.Sprintf("task-%d", wg.taskID),
		priority: priority,
		duration: actualDuration,
		workType: wg.config.WorkloadType,
		data:     data,
		ctx:      context.Background(),
	}
}

// applyBurstiness applies bursty submission patterns
func (bs *BenchmarkSuite) applyBurstiness(burstiness float64) {
	if rand.Float64() < burstiness {
		// Burst: submit multiple tasks quickly
		burstSize := rand.Intn(10) + 1
		for i := 0; i < burstSize; i++ {
			time.Sleep(time.Microsecond)
		}
	} else {
		// Quiet period
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)+1))
	}
}

// calculateLatencyPercentiles calculates latency statistics
func calculateLatencyPercentiles(latencies []time.Duration) (p50, p95, p99, min, max time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0, 0, 0
	}
	
	// Sort latencies
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}
	
	min = latencies[0]
	max = latencies[len(latencies)-1]
	
	p50 = latencies[int(float64(len(latencies))*0.50)]
	p95 = latencies[int(float64(len(latencies))*0.95)]
	p99 = latencies[int(float64(len(latencies))*0.99)]
	
	return
}

// GetResults returns benchmark results
func (bs *BenchmarkSuite) GetResults() []BenchmarkResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.results
}

// PrintResults prints benchmark results in a formatted table
func (bs *BenchmarkSuite) PrintResults() {
	results := bs.GetResults()
	
	fmt.Printf("\n=== Goroutine Pool Benchmark Results ===\n\n")
	fmt.Printf("%-15s %-10s %-12s %-12s %-12s %-12s %-12s %-10s\n",
		"Pool Type", "Tasks", "Throughput", "Avg Latency", "P95 Latency", "P99 Latency", "Utilization", "Error Rate")
	fmt.Printf("%s\n", string(make([]byte, 120)))
	
	for _, result := range results {
		fmt.Printf("%-15s %-10d %-12.2f %-12s %-12s %-12s %-12.2f%% %-10.2f%%\n",
			result.PoolType,
			result.TasksCompleted,
			result.Throughput,
			result.AverageLatency.String(),
			result.P95Latency.String(),
			result.P99Latency.String(),
			result.WorkerUtilization*100,
			result.ErrorRate*100,
		)
	}
	
	fmt.Printf("\n")
}

// CompareResults compares results and identifies the best performing pool
func (bs *BenchmarkSuite) CompareResults() map[string]string {
	results := bs.GetResults()
	if len(results) == 0 {
		return nil
	}
	
	comparison := make(map[string]string)
	
	// Find best in each category
	bestThroughput := results[0]
	bestLatency := results[0]
	bestUtilization := results[0]
	lowestError := results[0]
	
	for _, result := range results {
		if result.Throughput > bestThroughput.Throughput {
			bestThroughput = result
		}
		if result.AverageLatency < bestLatency.AverageLatency {
			bestLatency = result
		}
		if result.WorkerUtilization > bestUtilization.WorkerUtilization {
			bestUtilization = result
		}
		if result.ErrorRate < lowestError.ErrorRate {
			lowestError = result
		}
	}
	
	comparison["best_throughput"] = bestThroughput.PoolType
	comparison["best_latency"] = bestLatency.PoolType
	comparison["best_utilization"] = bestUtilization.PoolType
	comparison["lowest_error"] = lowestError.PoolType
	
	return comparison
}

// Implementation of WorkItem interface for TestWorkItem
func (t *TestWorkItem) Execute() error {
	t.startTime = time.Now()
	defer func() {
		t.endTime = time.Now()
	}()
	
	// Simulate work based on workload type
	switch t.workType {
	case CPUIntensive:
		t.simulateCPUWork()
	case IOBound:
		t.simulateIOWork()
	case Mixed:
		if rand.Float64() < 0.5 {
			t.simulateCPUWork()
		} else {
			t.simulateIOWork()
		}
	default:
		time.Sleep(t.duration)
	}
	
	return t.error
}

func (t *TestWorkItem) Priority() int {
	return t.priority
}

func (t *TestWorkItem) EstimatedDuration() time.Duration {
	return t.duration
}

func (t *TestWorkItem) Context() context.Context {
	return t.ctx
}

// simulateCPUWork simulates CPU-intensive work
func (t *TestWorkItem) simulateCPUWork() {
	start := time.Now()
	for time.Since(start) < t.duration {
		// Perform some CPU-intensive calculation
		math.Sqrt(rand.Float64() * 1000000)
	}
}

// simulateIOWork simulates I/O-bound work
func (t *TestWorkItem) simulateIOWork() {
	// Simulate I/O wait time
	time.Sleep(t.duration)
}

// Predefined benchmark configurations
func GetStandardBenchmarkConfigs() []BenchmarkConfig {
	return []BenchmarkConfig{
		{
			NumTasks:        1000,
			TaskDuration:    time.Millisecond * 10,
			TaskVariability: 0.2,
			Concurrency:     10,
			PoolSize:        5,
			BenchmarkTime:   time.Second * 30,
			WarmupTime:      time.Second * 5,
			WorkloadType:    CPUIntensive,
		},
		{
			NumTasks:        2000,
			TaskDuration:    time.Millisecond * 50,
			TaskVariability: 0.5,
			Concurrency:     20,
			PoolSize:        10,
			BenchmarkTime:   time.Second * 30,
			WarmupTime:      time.Second * 5,
			WorkloadType:    IOBound,
		},
		{
			NumTasks:        5000,
			TaskDuration:    time.Millisecond * 5,
			TaskVariability: 0.8,
			Concurrency:     50,
			PoolSize:        20,
			BenchmarkTime:   time.Second * 60,
			WarmupTime:      time.Second * 10,
			WorkloadType:    Mixed,
			Burstiness:      0.3,
		},
	}
}

// RunStandardBenchmarks runs a set of standard benchmarks
func RunStandardBenchmarks() []BenchmarkResult {
	suite := NewBenchmarkSuite()
	
	configs := GetStandardBenchmarkConfigs()
	for _, config := range configs {
		suite.AddConfig(config)
	}
	
	return suite.RunAllBenchmarks()
}

// BenchmarkFixedPool benchmarks the fixed pool
func BenchmarkFixedPool(b *testing.B) {
	config := BenchmarkConfig{
		NumTasks:        b.N,
		TaskDuration:    time.Microsecond * 100,
		TaskVariability: 0.1,
		PoolSize:        runtime.NumCPU(),
		WorkloadType:    CPUIntensive,
	}
	
	suite := NewBenchmarkSuite()
	suite.AddConfig(config)
	
	b.ResetTimer()
	results := suite.RunAllBenchmarks()
	b.StopTimer()
	
	for _, result := range results {
		if result.PoolType == "fixed" {
			b.ReportMetric(result.Throughput, "tasks/sec")
			b.ReportMetric(float64(result.AverageLatency.Nanoseconds()), "ns/task")
			break
		}
	}
}

// BenchmarkWorkStealingPool benchmarks the work-stealing pool
func BenchmarkWorkStealingPool(b *testing.B) {
	config := BenchmarkConfig{
		NumTasks:        b.N,
		TaskDuration:    time.Microsecond * 100,
		TaskVariability: 0.3,
		PoolSize:        runtime.NumCPU(),
		WorkloadType:    Mixed,
	}
	
	suite := NewBenchmarkSuite()
	suite.AddConfig(config)
	
	b.ResetTimer()
	results := suite.RunAllBenchmarks()
	b.StopTimer()
	
	for _, result := range results {
		if result.PoolType == "workstealing" {
			b.ReportMetric(result.Throughput, "tasks/sec")
			b.ReportMetric(float64(result.AverageLatency.Nanoseconds()), "ns/task")
			b.ReportMetric(result.WorkStealingSuccess*100, "steal_success_%")
			break
		}
	}
}