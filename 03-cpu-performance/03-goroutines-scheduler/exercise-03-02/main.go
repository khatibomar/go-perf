package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"exercise-03-02/benchmarks"
	"exercise-03-02/metrics"
	"exercise-03-02/pools"
	"exercise-03-02/workers"
)

// DemoTask implements WorkItem for demonstration
type DemoTask struct {
	id       string
	priority int
	duration time.Duration
	workFunc func() error
	ctx      context.Context
}

func (dt *DemoTask) Execute() error {
	if dt.workFunc != nil {
		return dt.workFunc()
	}
	// Default: simulate some work
	time.Sleep(dt.duration)
	return nil
}

func (dt *DemoTask) ID() string {
	return dt.id
}

func (dt *DemoTask) Priority() int {
	return dt.priority
}

func (dt *DemoTask) EstimatedDuration() time.Duration {
	return dt.duration
}

func (dt *DemoTask) Context() context.Context {
	return dt.ctx
}

func main() {
	fmt.Println("=== Goroutine Pool Performance Exercise ===")
	fmt.Printf("Runtime: Go %s, GOMAXPROCS: %d\n\n", runtime.Version(), runtime.GOMAXPROCS(0))

	// Run demonstrations
	runDemonstrations()

	// Run benchmarks
	runBenchmarks()

	// Run advanced scenarios
	runAdvancedScenarios()
}

func runDemonstrations() {
	fmt.Println("=== Pool Demonstrations ===")

	// Demo 1: Fixed Pool
	fmt.Println("\n1. Fixed Pool Demo")
	demoFixedPool()

	// Demo 2: Dynamic Pool
	fmt.Println("\n2. Dynamic Pool Demo")
	demoDynamicPool()

	// Demo 3: Work-Stealing Pool
	fmt.Println("\n3. Work-Stealing Pool Demo")
	demoWorkStealingPool()

	// Demo 4: Priority Pool
	fmt.Println("\n4. Priority Pool Demo")
	demoPriorityPool()

	// Demo 5: Batch Worker
	fmt.Println("\n5. Batch Worker Demo")
	demoBatchWorker()
}

func demoFixedPool() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create fixed pool with 4 workers
	pool := pools.NewFixedPool(4, 100)
	err := pool.Start()
	if err != nil {
		log.Printf("Failed to start fixed pool: %v", err)
		return
	}
	defer pool.Shutdown(5 * time.Second)

	// Submit tasks
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		taskID := fmt.Sprintf("fixed-task-%d", i)
		task := &DemoTask{
			id:       taskID,
			priority: 1,
			duration: time.Millisecond * time.Duration(50+rand.Intn(100)),
			workFunc: func() error {
				defer wg.Done()
				fmt.Printf("Executing task %s\n", taskID)
				return nil
			},
			ctx: ctx,
		}
		err := pool.Submit(func() {
			task.Execute()
		})
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			wg.Done()
		}
	}

	wg.Wait()
	fmt.Printf("Fixed Pool Demo completed successfully\n")
}

func demoDynamicPool() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create dynamic pool (1-8 workers, scale every 2 seconds)
	pool := pools.NewDynamicPool(1, 8, int(time.Second.Milliseconds()*2))
	err := pool.Start()
	if err != nil {
		log.Printf("Failed to start dynamic pool: %v", err)
		return
	}
	defer pool.Shutdown(5 * time.Second)

	// Submit burst of tasks
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		task := &DemoTask{
			id:       fmt.Sprintf("dynamic-task-%d", i),
			priority: 1,
			duration: time.Millisecond * time.Duration(100+rand.Intn(200)),
			workFunc: func() error {
				defer wg.Done()
				return nil
			},
			ctx: ctx,
		}
		err := pool.Submit(func() {
			task.Execute()
		})
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			wg.Done()
		}

		// Add delay to see scaling in action
		if i%10 == 0 {
			time.Sleep(time.Millisecond * 500)
			fmt.Printf("Dynamic Pool at task %d\n", i)
		}
	}

	wg.Wait()
	fmt.Printf("Dynamic Pool Demo completed successfully\n")
}

func demoWorkStealingPool() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create work-stealing pool with 4 workers, local queue size 10
	pool := pools.NewWorkStealingPool(4, 10, 100)
	err := pool.Start()
	if err != nil {
		log.Printf("Failed to start work-stealing pool: %v", err)
		return
	}
	defer pool.Shutdown(5 * time.Second)

	// Submit tasks with varying durations to trigger work stealing
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		// Create tasks with different durations to create imbalance
		duration := time.Millisecond * 50
		if i%5 == 0 {
			duration = time.Millisecond * 300 // Some longer tasks
		}

		task := &DemoTask{
			id:       fmt.Sprintf("ws-task-%d", i),
			priority: 1,
			duration: duration,
			workFunc: func() error {
				defer wg.Done()
				return nil
			},
			ctx: ctx,
		}
		err := pool.Submit(func() {
			task.Execute()
		})
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			wg.Done()
		}
	}

	wg.Wait()
	fmt.Printf("Work-Stealing Pool Demo completed successfully\n")
}

func demoPriorityPool() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create priority pool with 3 workers
	pool := pools.NewPriorityPool(3, []int{50, 50, 50, 50})
	err := pool.Start()
	if err != nil {
		log.Printf("Failed to start priority pool: %v", err)
		return
	}
	defer pool.Shutdown(5 * time.Second)

	// Submit tasks with different priorities
	var wg sync.WaitGroup
	var executionOrder []string
	var orderMu sync.Mutex

	// Submit low priority tasks first
	for i := 0; i < 10; i++ {
		wg.Add(1)
		taskID := fmt.Sprintf("low-priority-%d", i)
		task := &DemoTask{
			id:       taskID,
			priority: 1, // Low priority
			duration: time.Millisecond * 100,
			workFunc: func() error {
				defer wg.Done()
				orderMu.Lock()
				executionOrder = append(executionOrder, taskID)
				orderMu.Unlock()
				return nil
			},
			ctx: ctx,
		}
		err := pool.Submit(func() {
			task.Execute()
		}, task.Priority())
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			wg.Done()
		}
	}

	// Submit high priority tasks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		taskID := fmt.Sprintf("high-priority-%d", i)
		task := &DemoTask{
			id:       taskID,
			priority: 4, // High priority
			duration: time.Millisecond * 50,
			workFunc: func() error {
				defer wg.Done()
				orderMu.Lock()
				executionOrder = append(executionOrder, taskID)
				orderMu.Unlock()
				return nil
			},
			ctx: ctx,
		}
		err := pool.Submit(func() {
			task.Execute()
		}, task.Priority())
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			wg.Done()
		}
	}

	wg.Wait()
	fmt.Printf("Priority Pool Demo completed successfully\n")
	fmt.Printf("Execution order (first 10): %v\n", executionOrder[:min(10, len(executionOrder))])
}

func demoBatchWorker() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create batch worker (batch size 5, timeout 1 second)
	batchWorker := workers.NewBatchWorker("batch-demo", 100, 5, time.Second)
	err := batchWorker.Start(ctx)
	if err != nil {
		log.Printf("Failed to start batch worker: %v", err)
		return
	}
	defer batchWorker.Stop(5 * time.Second)

	// Submit batchable tasks
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		err := batchWorker.SubmitBatchWork(
			func() error {
				defer wg.Done()
				fmt.Printf("Processing batch item %d\n", i)
				time.Sleep(time.Millisecond * 50)
				return nil
			},
			"demo-batch",
			1,
			map[string]interface{}{"item": i},
		)
		if err != nil {
			log.Printf("Failed to submit batch work: %v", err)
			wg.Done()
		}

		// Add delay to see batching in action
		time.Sleep(time.Millisecond * 200)
	}

	wg.Wait()
	fmt.Printf("Batch Worker Demo completed successfully\n")
}

func runBenchmarks() {
	fmt.Println("\n=== Performance Benchmarks ===")

	// Run standard benchmarks
	results := benchmarks.RunStandardBenchmarks()

	// Print results
	suite := benchmarks.NewBenchmarkSuite()
	for _, result := range results {
		// Add results to suite for printing
		suite.AddConfig(result.Config)
	}

	fmt.Printf("\nBenchmark Results Summary:\n")
	fmt.Printf("%-15s %-12s %-12s %-12s %-12s\n", "Pool Type", "Throughput", "Avg Latency", "P95 Latency", "Utilization")
	fmt.Printf("%s\n", string(make([]rune, 75)))

	for _, result := range results {
		fmt.Printf("%-15s %-12.2f %-12s %-12s %-12.2f%%\n",
			result.PoolType,
			result.Throughput,
			result.AverageLatency.String(),
			result.P95Latency.String(),
			result.WorkerUtilization*100,
		)
	}

	// Find best performers
	bestThroughput := results[0]
	bestLatency := results[0]
	for _, result := range results {
		if result.Throughput > bestThroughput.Throughput {
			bestThroughput = result
		}
		if result.AverageLatency < bestLatency.AverageLatency {
			bestLatency = result
		}
	}

	fmt.Printf("\nBest Performers:\n")
	fmt.Printf("Highest Throughput: %s (%.2f tasks/sec)\n", bestThroughput.PoolType, bestThroughput.Throughput)
	fmt.Printf("Lowest Latency: %s (%s)\n", bestLatency.PoolType, bestLatency.AverageLatency)
}

func runAdvancedScenarios() {
	fmt.Println("\n=== Advanced Scenarios ===")

	// Scenario 1: Memory pressure test
	fmt.Println("\n1. Memory Pressure Test")
	testMemoryPressure()

	// Scenario 2: Burst workload test
	fmt.Println("\n2. Burst Workload Test")
	testBurstWorkload()

	// Scenario 3: Mixed workload test
	fmt.Println("\n3. Mixed Workload Test")
	testMixedWorkload()
}

func testMemoryPressure() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Create metrics collector
	metricsCollector := metrics.NewMetricsCollector(time.Millisecond * 500)
	metricsCollector.Start()
	defer metricsCollector.Stop()

	// Test with work-stealing pool
	pool := pools.NewWorkStealingPool(4, 20, 100)
	err := pool.Start()
	if err != nil {
		log.Printf("Failed to start pool: %v", err)
		return
	}
	defer pool.Shutdown(5 * time.Second)

	// Submit memory-intensive tasks
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		task := &DemoTask{
			id:       fmt.Sprintf("memory-task-%d", i),
			priority: 1,
			duration: time.Millisecond * 100,
			workFunc: func() error {
				defer wg.Done()
				// Allocate and use memory
				data := make([]byte, 1024*1024) // 1MB
				for j := range data {
					data[j] = byte(j % 256)
				}
				// Simulate processing
				time.Sleep(time.Millisecond * 50)
				return nil
			},
			ctx: ctx,
		}
		err := pool.Submit(func() {
			task.Execute()
		})
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			wg.Done()
		}
	}

	wg.Wait()

	// Check memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Printf("Memory Pressure Test: Alloc=%d KB, Sys=%d KB, NumGC=%d\n",
		memStats.Alloc/1024, memStats.Sys/1024, memStats.NumGC)
}

func testBurstWorkload() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Test with dynamic pool
	pool := pools.NewDynamicPool(2, 10, int(time.Second.Milliseconds()))
	err := pool.Start()
	if err != nil {
		log.Printf("Failed to start pool: %v", err)
		return
	}
	defer pool.Shutdown(5 * time.Second)

	// Submit burst workload
	var wg sync.WaitGroup
	for burst := 0; burst < 3; burst++ {
		fmt.Printf("Submitting burst %d...\n", burst+1)
		
		// Submit burst of tasks
		for i := 0; i < 20; i++ {
			wg.Add(1)
			task := &DemoTask{
				id:       fmt.Sprintf("burst-%d-task-%d", burst, i),
				priority: 1,
				duration: time.Millisecond * time.Duration(50+rand.Intn(100)),
				workFunc: func() error {
					defer wg.Done()
					return nil
				},
				ctx: ctx,
			}
			err := pool.Submit(func() {
				task.Execute()
			})
			if err != nil {
				log.Printf("Failed to submit task: %v", err)
				wg.Done()
			}
		}

		// Wait between bursts
		time.Sleep(time.Second * 2)
		fmt.Printf("After burst %d completed\n", burst+1)
	}

	wg.Wait()
	fmt.Printf("Burst Test completed successfully\n")
}

func testMixedWorkload() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Test with priority pool
	pool := pools.NewPriorityPool(6, []int{100, 100, 100, 100})
	err := pool.Start()
	if err != nil {
		log.Printf("Failed to start pool: %v", err)
		return
	}
	defer pool.Shutdown(5 * time.Second)

	// Submit mixed workload
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		
		// Vary task characteristics
		priority := 1 + rand.Intn(4) // Priority 1-4
		duration := time.Millisecond * time.Duration(25+rand.Intn(150))
		taskType := rand.Intn(3) // 0=CPU, 1=IO, 2=Memory

		task := &DemoTask{
			id:       fmt.Sprintf("mixed-task-%d", i),
			priority: priority,
			duration: duration,
			workFunc: func() error {
				defer wg.Done()
				switch taskType {
				case 0: // CPU intensive
					for j := 0; j < 100000; j++ {
						_ = j * j
					}
				case 1: // IO simulation
					time.Sleep(duration)
				case 2: // Memory allocation
					data := make([]int, 10000)
					for k := range data {
						data[k] = k
					}
				}
				return nil
			},
			ctx: ctx,
		}
		err := pool.Submit(func() {
			task.Execute()
		}, task.Priority())
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			wg.Done()
		}
	}

	wg.Wait()
	fmt.Printf("Mixed Workload Test completed successfully\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}