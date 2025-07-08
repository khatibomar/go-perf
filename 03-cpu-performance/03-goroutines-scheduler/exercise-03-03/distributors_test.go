package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"exercise-03-03/distributors"
	"exercise-03-03/types"
	"exercise-03-03/workers"
)

func TestRoundRobinDistributor(t *testing.T) {
	// Create worker pool
	pool := workers.NewWorkerPool(3, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// Create distributor
	distributor := distributors.NewRoundRobinDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	// Test single work distribution
	work := &types.SimpleWork{
		ID:   "test-work-1",
		Size: 100,
		Task: func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	if err := distributor.Distribute(work); err != nil {
		t.Errorf("Failed to distribute work: %v", err)
	}

	// Test batch distribution
	works := make([]types.Work, 9)
	for i := 0; i < 9; i++ {
		works[i] = &types.SimpleWork{
			ID:   fmt.Sprintf("batch-work-%d", i),
			Size: int64(i * 10),
			Task: func() error {
				time.Sleep(5 * time.Millisecond)
				return nil
			},
		}
	}

	if err := distributor.DistributeBatch(works); err != nil {
		t.Errorf("Failed to distribute batch: %v", err)
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	// Check metrics
	metrics := distributor.GetMetrics()
	if metrics.TotalDistributed != 10 {
		t.Errorf("Expected 10 distributed works, got %d", metrics.TotalDistributed)
	}

	if metrics.WorkerCount != 3 {
		t.Errorf("Expected 3 workers, got %d", metrics.WorkerCount)
	}

	// Test worker removal
	workers := pool.GetWorkers()
	if len(workers) > 0 {
		distributor.RemoveWorker(workers[0].GetID())
		metrics = distributor.GetMetrics()
		if metrics.WorkerCount != 2 {
			t.Errorf("Expected 2 workers after removal, got %d", metrics.WorkerCount)
		}
	}
}

func TestLoadBasedDistributor(t *testing.T) {
	// Create worker pool
	pool := workers.NewWorkerPool(3, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// Create distributor
	distributor := distributors.NewLoadBasedDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	// Create works with different sizes
	works := make([]types.Work, 6)
	for i := 0; i < 6; i++ {
		processingTime := time.Duration((i+1)*5) * time.Millisecond
		works[i] = &types.SimpleWork{
			ID:   fmt.Sprintf("load-work-%d", i),
			Size: int64(processingTime.Nanoseconds() / 1e6),
			Task: func() error {
				time.Sleep(processingTime)
				return nil
			},
		}
	}

	// Distribute works
	for _, work := range works {
		if err := distributor.Distribute(work); err != nil {
			t.Errorf("Failed to distribute work: %v", err)
		}
	}

	// Wait for some processing
	time.Sleep(50 * time.Millisecond)

	// Check metrics
	metrics := distributor.GetMetrics()
	if metrics.TotalDistributed != 6 {
		t.Errorf("Expected 6 distributed works, got %d", metrics.TotalDistributed)
	}

	// Check worker loads
	loads := distributor.GetWorkerLoads()
	if len(loads) != 3 {
		t.Errorf("Expected 3 worker loads, got %d", len(loads))
	}

	// Load balance should be reasonable (not perfect due to timing)
	if metrics.LoadBalance < 0 || metrics.LoadBalance > 1 {
		t.Errorf("Load balance should be between 0 and 1, got %.2f", metrics.LoadBalance)
	}
}

func TestWorkStealingCoordinator(t *testing.T) {
	// Create coordinator
	coordinator := distributors.NewWorkStealingCoordinator(3, 10)
	if err := coordinator.Start(); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop()

	// Create works
	works := make([]types.Work, 15)
	for i := 0; i < 15; i++ {
		works[i] = &types.SimpleWork{
			ID:   fmt.Sprintf("steal-work-%d", i),
			Size: 100,
			Task: func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}
	}

	// Distribute works
	for _, work := range works {
		if err := coordinator.Distribute(work); err != nil {
			t.Errorf("Failed to distribute work: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	metrics := coordinator.GetMetrics()
	if metrics.TotalDistributed != 15 {
		t.Errorf("Expected 15 distributed works, got %d", metrics.TotalDistributed)
	}

	if metrics.WorkerCount != 3 {
		t.Errorf("Expected 3 workers, got %d", metrics.WorkerCount)
	}
}

func TestHierarchicalDistributor(t *testing.T) {
	// Create hierarchical distributor
	hierarchical := distributors.NewHierarchicalDistributor(2)

	// Create worker pools
	pool1 := workers.NewWorkerPool(2, 10)
	pool2 := workers.NewWorkerPool(2, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool1.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool1: %v", err)
	}
	defer pool1.Stop()

	if err := pool2.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool2: %v", err)
	}
	defer pool2.Stop()

	// Create distributors
	dist1 := distributors.NewRoundRobinDistributor()
	dist2 := distributors.NewRoundRobinDistributor()

	for _, worker := range pool1.GetWorkers() {
		dist1.AddWorker(worker)
	}
	for _, worker := range pool2.GetWorkers() {
		dist2.AddWorker(worker)
	}

	// Add level
	hierarchical.AddLevel([]types.Distributor{dist1, dist2})

	// Test distribution
	works := make([]types.Work, 8)
	for i := 0; i < 8; i++ {
		works[i] = &types.SimpleWork{
			ID:   fmt.Sprintf("hier-work-%d", i),
			Size: 100,
			Task: func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}
	}

	for _, work := range works {
		if err := hierarchical.Distribute(work); err != nil {
			t.Errorf("Failed to distribute work: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check metrics
	metrics := hierarchical.GetMetrics()
	if metrics.TotalDistributed != 8 {
		t.Errorf("Expected 8 distributed works, got %d", metrics.TotalDistributed)
	}

	if hierarchical.GetLevelCount() != 1 {
		t.Errorf("Expected 1 level, got %d", hierarchical.GetLevelCount())
	}
}

func TestAffinityDistributor(t *testing.T) {
	// Create worker pool
	pool := workers.NewWorkerPool(4, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// Create distributor
	distributor := distributors.NewAffinityDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	// Create works with affinity keys
	affinityKeys := []string{"cache-A", "cache-B", "cache-A", "cache-C", "cache-B"}
	works := make([]types.Work, len(affinityKeys))

	for i, key := range affinityKeys {
		works[i] = &types.SimpleWork{
			ID:          fmt.Sprintf("affinity-work-%d", i),
			AffinityKey: key,
			Size:        100,
			Task: func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}
	}

	// Distribute works
	for _, work := range works {
		if err := distributor.Distribute(work); err != nil {
			t.Errorf("Failed to distribute work: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check metrics
	metrics := distributor.GetMetrics()
	if metrics.TotalDistributed != 5 {
		t.Errorf("Expected 5 distributed works, got %d", metrics.TotalDistributed)
	}

	// Check affinity mappings
	affinityMap := distributor.GetAffinityMap()
	if len(affinityMap) == 0 {
		t.Error("Expected affinity mappings to be created")
	}

	// Verify that same affinity keys map to same workers
	if affinityMap["cache-A"] == affinityMap["cache-B"] {
		t.Error("Different affinity keys should not map to same worker (unless hash collision)")
	}
}

func TestConcurrentDistribution(t *testing.T) {
	// Test concurrent distribution across multiple distributors
	pool := workers.NewWorkerPool(4, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	// Create multiple distributors
	roundRobin := distributors.NewRoundRobinDistributor()
	loadBased := distributors.NewLoadBasedDistributor()

	for _, worker := range pool.GetWorkers() {
		roundRobin.AddWorker(worker)
		loadBased.AddWorker(worker)
	}

	const numGoroutines = 5
	const worksPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // 2 distributors

	// Test round-robin concurrency
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < worksPerGoroutine; j++ {
				work := &types.SimpleWork{
					ID:   fmt.Sprintf("rr-concurrent-%d-%d", id, j),
					Size: 50,
					Task: func() error {
						time.Sleep(1 * time.Millisecond)
						return nil
					},
				}
				if err := roundRobin.Distribute(work); err != nil {
					t.Errorf("Round-robin distribution failed: %v", err)
				}
			}
		}(i)
	}

	// Test load-based concurrency
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < worksPerGoroutine; j++ {
				work := &types.SimpleWork{
					ID:   fmt.Sprintf("lb-concurrent-%d-%d", id, j),
					Size: int64((j % 5) * 10),
					Task: func() error {
						time.Sleep(time.Duration((j%5)+1) * time.Millisecond)
						return nil
					},
				}
				if err := loadBased.Distribute(work); err != nil {
					t.Errorf("Load-based distribution failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Wait for processing to complete
	time.Sleep(500 * time.Millisecond)

	// Check final metrics
	rrMetrics := roundRobin.GetMetrics()
	lbMetrics := loadBased.GetMetrics()

	expectedWorks := numGoroutines * worksPerGoroutine
	if rrMetrics.TotalDistributed != int64(expectedWorks) {
		t.Errorf("Round-robin: expected %d works, got %d", expectedWorks, rrMetrics.TotalDistributed)
	}

	if lbMetrics.TotalDistributed != int64(expectedWorks) {
		t.Errorf("Load-based: expected %d works, got %d", expectedWorks, lbMetrics.TotalDistributed)
	}
}

func TestErrorHandling(t *testing.T) {
	// Test distribution without workers
	distributor := distributors.NewRoundRobinDistributor()

	work := &types.SimpleWork{
		ID:   "error-test",
		Size: 100,
		Task: func() error { return nil },
	}

	err := distributor.Distribute(work)
	if err == nil {
		t.Error("Expected error when distributing without workers")
	}

	// Test batch distribution without workers
	err = distributor.DistributeBatch([]types.Work{work})
	if err == nil {
		t.Error("Expected error when batch distributing without workers")
	}

	// Test removing non-existent worker
	err = distributor.RemoveWorker("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent worker")
	}
}