package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"exercise-03-03/distributors"
	"exercise-03-03/monitoring"
	"exercise-03-03/types"
	"exercise-03-03/workers"
)

func main() {
	fmt.Println("Work Distribution Patterns Demo")
	fmt.Println("===============================")

	// Run demonstrations of different distribution patterns
	runRoundRobinDemo()
	runLoadBasedDemo()
	runWorkStealingDemo()
	runHierarchicalDemo()
	runAffinityDemo()

	// Run performance comparison
	runPerformanceComparison()
}

// runRoundRobinDemo demonstrates round-robin distribution
func runRoundRobinDemo() {
	fmt.Println("\n1. Round-Robin Distribution Demo")
	fmt.Println("---------------------------------")

	// Create worker pool
	pool := workers.NewWorkerPool(4, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		log.Printf("Failed to start worker pool: %v", err)
		return
	}
	defer pool.Stop()

	// Create round-robin distributor
	distributor := distributors.NewRoundRobinDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	// Create and distribute work
	works := createTestWorks(20, "round-robin")
	start := time.Now()

	for _, work := range works {
		if err := distributor.Distribute(work); err != nil {
			log.Printf("Failed to distribute work: %v", err)
		}
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	metrics := distributor.GetMetrics()
	fmt.Printf("Distributed %d works in %v\n", metrics.TotalDistributed, time.Since(start))
	fmt.Printf("Load Balance: %.2f\n", metrics.LoadBalance)
	fmt.Printf("Worker Count: %d\n", metrics.WorkerCount)
}

// runLoadBasedDemo demonstrates load-based distribution
func runLoadBasedDemo() {
	fmt.Println("\n2. Load-Based Distribution Demo")
	fmt.Println("--------------------------------")

	// Create worker pool
	pool := workers.NewWorkerPool(4, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		log.Printf("Failed to start worker pool: %v", err)
		return
	}
	defer pool.Stop()

	// Create load-based distributor
	distributor := distributors.NewLoadBasedDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	// Create works with varying processing times
	works := createVariableWorks(20, "load-based")
	start := time.Now()

	for _, work := range works {
		if err := distributor.Distribute(work); err != nil {
			log.Printf("Failed to distribute work: %v", err)
		}
	}

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	metrics := distributor.GetMetrics()
	fmt.Printf("Distributed %d works in %v\n", metrics.TotalDistributed, time.Since(start))
	fmt.Printf("Load Balance: %.2f\n", metrics.LoadBalance)

	// Show worker loads
	loads := distributor.GetWorkerLoads()
	fmt.Println("Worker Loads:")
	for workerID, load := range loads {
		fmt.Printf("  %s: %.2f\n", workerID, load)
	}
}

// runWorkStealingDemo demonstrates work-stealing coordination
func runWorkStealingDemo() {
	fmt.Println("\n3. Work-Stealing Distribution Demo")
	fmt.Println("-----------------------------------")

	// Create work-stealing coordinator
	coordinator := distributors.NewWorkStealingCoordinator(4, 10)
	if err := coordinator.Start(); err != nil {
		log.Printf("Failed to start coordinator: %v", err)
		return
	}
	defer coordinator.Stop()

	// Create and distribute work
	works := createTestWorks(30, "work-stealing")
	start := time.Now()

	for _, work := range works {
		if err := coordinator.Distribute(work); err != nil {
			log.Printf("Failed to distribute work: %v", err)
		}
	}

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	metrics := coordinator.GetMetrics()
	fmt.Printf("Distributed %d works in %v\n", metrics.TotalDistributed, time.Since(start))
	fmt.Printf("Load Balance: %.2f\n", metrics.LoadBalance)
	fmt.Printf("Worker Count: %d\n", metrics.WorkerCount)
}

// runHierarchicalDemo demonstrates hierarchical distribution
func runHierarchicalDemo() {
	fmt.Println("\n4. Hierarchical Distribution Demo")
	fmt.Println("----------------------------------")

	// Create hierarchical distributor
	hierarchical := distributors.NewHierarchicalDistributor(2)

	// Create worker pools for bottom level
	pool1 := workers.NewWorkerPool(2, 10)
	pool2 := workers.NewWorkerPool(2, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool1.Start(ctx); err != nil {
		log.Printf("Failed to start pool1: %v", err)
		return
	}
	defer pool1.Stop()

	if err := pool2.Start(ctx); err != nil {
		log.Printf("Failed to start pool2: %v", err)
		return
	}
	defer pool2.Stop()

	// Create distributors for bottom level
	dist1 := distributors.NewRoundRobinDistributor()
	dist2 := distributors.NewRoundRobinDistributor()

	for _, worker := range pool1.GetWorkers() {
		dist1.AddWorker(worker)
	}
	for _, worker := range pool2.GetWorkers() {
		dist2.AddWorker(worker)
	}

	// Add level to hierarchy
	hierarchical.AddLevel([]types.Distributor{dist1, dist2})

	// Create and distribute work
	works := createTestWorks(20, "hierarchical")
	start := time.Now()

	for _, work := range works {
		if err := hierarchical.Distribute(work); err != nil {
			log.Printf("Failed to distribute work: %v", err)
		}
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	metrics := hierarchical.GetMetrics()
	fmt.Printf("Distributed %d works in %v\n", metrics.TotalDistributed, time.Since(start))
	fmt.Printf("Load Balance: %.2f\n", metrics.LoadBalance)
	fmt.Printf("Levels: %d\n", hierarchical.GetLevelCount())
}

// runAffinityDemo demonstrates affinity-based distribution
func runAffinityDemo() {
	fmt.Println("\n5. Affinity-Based Distribution Demo")
	fmt.Println("------------------------------------")

	// Create worker pool
	pool := workers.NewWorkerPool(4, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		log.Printf("Failed to start worker pool: %v", err)
		return
	}
	defer pool.Stop()

	// Create affinity distributor
	distributor := distributors.NewAffinityDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	// Create works with affinity keys
	works := createAffinityWorks(20, "affinity")
	start := time.Now()

	for _, work := range works {
		if err := distributor.Distribute(work); err != nil {
			log.Printf("Failed to distribute work: %v", err)
		}
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	metrics := distributor.GetMetrics()
	fmt.Printf("Distributed %d works in %v\n", metrics.TotalDistributed, time.Since(start))
	fmt.Printf("Load Balance: %.2f\n", metrics.LoadBalance)

	// Show affinity mappings
	affinityMap := distributor.GetAffinityMap()
	fmt.Println("Affinity Mappings:")
	for key, workerIndex := range affinityMap {
		fmt.Printf("  %s -> worker-%d\n", key, workerIndex)
	}
}

// runPerformanceComparison compares different distribution patterns
func runPerformanceComparison() {
	fmt.Println("\n6. Performance Comparison")
	fmt.Println("--------------------------")

	patterns := []string{"round-robin", "load-based", "work-stealing", "affinity"}
	workCounts := []int{100, 500, 1000}

	for _, workCount := range workCounts {
		fmt.Printf("\nWork Count: %d\n", workCount)
		fmt.Println("Pattern\t\tThroughput (ops/sec)\tLatency (ms)")
		fmt.Println("-------\t\t--------------------\t------------")

		for _, pattern := range patterns {
			throughput, latency := benchmarkPattern(pattern, workCount)
			fmt.Printf("%s\t%.0f\t\t\t%.2f\n", pattern, throughput, latency)
		}
	}
}

// benchmarkPattern benchmarks a specific distribution pattern
func benchmarkPattern(pattern string, workCount int) (throughput float64, avgLatency float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create monitor
	monitor := monitoring.NewLoadMonitor(10 * time.Millisecond)
	monitor.Start(ctx)
	defer monitor.Stop()

	start := time.Now()

	switch pattern {
	case "round-robin":
		throughput, avgLatency = benchmarkRoundRobin(ctx, monitor, workCount)
	case "load-based":
		throughput, avgLatency = benchmarkLoadBased(ctx, monitor, workCount)
	case "work-stealing":
		throughput, avgLatency = benchmarkWorkStealing(ctx, monitor, workCount)
	case "affinity":
		throughput, avgLatency = benchmarkAffinity(ctx, monitor, workCount)
	}

	elapsed := time.Since(start).Seconds()
	if elapsed > 0 {
		throughput = float64(workCount) / elapsed
	}

	return throughput, avgLatency
}

// Helper functions for benchmarking
func benchmarkRoundRobin(ctx context.Context, monitor *monitoring.LoadMonitor, workCount int) (float64, float64) {
	pool := workers.NewWorkerPool(4, 50)
	pool.Start(ctx)
	defer pool.Stop()

	distributor := distributors.NewRoundRobinDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
		monitor.AddWorker(worker)
	}
	monitor.AddDistributor("round-robin", distributor)

	works := createTestWorks(workCount, "benchmark")
	var wg sync.WaitGroup

	for _, work := range works {
		wg.Add(1)
		go func(w types.Work) {
			defer wg.Done()
			distributor.Distribute(w)
		}(work)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond) // Allow processing to complete

	metrics := monitor.GetMetrics()
	return metrics.Overall.Throughput, float64(metrics.Overall.AverageLatency.Nanoseconds()) / 1e6
}

func benchmarkLoadBased(ctx context.Context, monitor *monitoring.LoadMonitor, workCount int) (float64, float64) {
	pool := workers.NewWorkerPool(4, 50)
	pool.Start(ctx)
	defer pool.Stop()

	distributor := distributors.NewLoadBasedDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
		monitor.AddWorker(worker)
	}
	monitor.AddDistributor("load-based", distributor)

	works := createVariableWorks(workCount, "benchmark")
	var wg sync.WaitGroup

	for _, work := range works {
		wg.Add(1)
		go func(w types.Work) {
			defer wg.Done()
			distributor.Distribute(w)
		}(work)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	metrics := monitor.GetMetrics()
	return metrics.Overall.Throughput, float64(metrics.Overall.AverageLatency.Nanoseconds()) / 1e6
}

func benchmarkWorkStealing(ctx context.Context, monitor *monitoring.LoadMonitor, workCount int) (float64, float64) {
	coordinator := distributors.NewWorkStealingCoordinator(4, 50)
	coordinator.Start()
	defer coordinator.Stop()

	monitor.AddDistributor("work-stealing", coordinator)

	works := createTestWorks(workCount, "benchmark")
	var wg sync.WaitGroup

	for _, work := range works {
		wg.Add(1)
		go func(w types.Work) {
			defer wg.Done()
			coordinator.Distribute(w)
		}(work)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	metrics := monitor.GetMetrics()
	return metrics.Overall.Throughput, float64(metrics.Overall.AverageLatency.Nanoseconds()) / 1e6
}

func benchmarkAffinity(ctx context.Context, monitor *monitoring.LoadMonitor, workCount int) (float64, float64) {
	pool := workers.NewWorkerPool(4, 50)
	pool.Start(ctx)
	defer pool.Stop()

	distributor := distributors.NewAffinityDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
		monitor.AddWorker(worker)
	}
	monitor.AddDistributor("affinity", distributor)

	works := createAffinityWorks(workCount, "benchmark")
	var wg sync.WaitGroup

	for _, work := range works {
		wg.Add(1)
		go func(w types.Work) {
			defer wg.Done()
			distributor.Distribute(w)
		}(work)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	metrics := monitor.GetMetrics()
	return metrics.Overall.Throughput, float64(metrics.Overall.AverageLatency.Nanoseconds()) / 1e6
}

// Helper functions to create test works
func createTestWorks(count int, prefix string) []types.Work {
	works := make([]types.Work, count)
	for i := 0; i < count; i++ {
		works[i] = &types.SimpleWork{
			ID:       fmt.Sprintf("%s-work-%d", prefix, i),
			Priority: rand.Intn(10),
			Size:     int64(rand.Intn(1000) + 100),
			Task: func() error {
				// Simulate work
				time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				return nil
			},
		}
	}
	return works
}

func createVariableWorks(count int, prefix string) []types.Work {
	works := make([]types.Work, count)
	for i := 0; i < count; i++ {
		// Create works with variable processing times
		processingTime := time.Duration(rand.Intn(20)+1) * time.Millisecond
		works[i] = &types.SimpleWork{
			ID:       fmt.Sprintf("%s-work-%d", prefix, i),
			Priority: rand.Intn(10),
			Size:     int64(processingTime.Nanoseconds() / 1e6), // Size correlates with processing time
			Task: func() error {
				time.Sleep(processingTime)
				return nil
			},
		}
	}
	return works
}

func createAffinityWorks(count int, prefix string) []types.Work {
	affinityKeys := []string{"cache-A", "cache-B", "cache-C", "cache-D"}
	works := make([]types.Work, count)

	for i := 0; i < count; i++ {
		affinityKey := affinityKeys[i%len(affinityKeys)]
		works[i] = &types.SimpleWork{
			ID:          fmt.Sprintf("%s-work-%d", prefix, i),
			AffinityKey: affinityKey,
			Priority:    rand.Intn(10),
			Size:        int64(rand.Intn(1000) + 100),
			Task: func() error {
				// Simulate cache-friendly work
				time.Sleep(time.Duration(rand.Intn(3)+1) * time.Millisecond)
				return nil
			},
		}
	}
	return works
}