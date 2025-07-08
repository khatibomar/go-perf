package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"exercise-03-03/distributors"
	"exercise-03-03/monitoring"
	"exercise-03-03/types"
	"exercise-03-03/workers"
)

func BenchmarkRoundRobinDistributor(b *testing.B) {
	pool := workers.NewWorkerPool(4, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	distributor := distributors.NewRoundRobinDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			work := &types.SimpleWork{
				ID:   fmt.Sprintf("bench-work-%d", i),
				Size: 100,
				Task: func() error {
					// Minimal work to focus on distribution overhead
					return nil
				},
			}
			distributor.Distribute(work)
			i++
		}
	})
}

func BenchmarkLoadBasedDistributor(b *testing.B) {
	pool := workers.NewWorkerPool(4, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	distributor := distributors.NewLoadBasedDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			work := &types.SimpleWork{
				ID:   fmt.Sprintf("bench-work-%d", i),
				Size: int64(rand.Intn(1000) + 100),
				Task: func() error {
					return nil
				},
			}
			distributor.Distribute(work)
			i++
		}
	})
}

func BenchmarkWorkStealingCoordinator(b *testing.B) {
	coordinator := distributors.NewWorkStealingCoordinator(4, 100)
	if err := coordinator.Start(); err != nil {
		b.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			work := &types.SimpleWork{
				ID:   fmt.Sprintf("bench-work-%d", i),
				Size: 100,
				Task: func() error {
					return nil
				},
			}
			coordinator.Distribute(work)
			i++
		}
	})
}

func BenchmarkAffinityDistributor(b *testing.B) {
	pool := workers.NewWorkerPool(4, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	distributor := distributors.NewAffinityDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	affinityKeys := []string{"cache-A", "cache-B", "cache-C", "cache-D"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			work := &types.SimpleWork{
				ID:          fmt.Sprintf("bench-work-%d", i),
				AffinityKey: affinityKeys[i%len(affinityKeys)],
				Size:        100,
				Task: func() error {
					return nil
				},
			}
			distributor.Distribute(work)
			i++
		}
	})
}

func BenchmarkBatchDistribution(b *testing.B) {
	pool := workers.NewWorkerPool(4, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	distributor := distributors.NewRoundRobinDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	batchSizes := []int{10, 50, 100}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize%d", batchSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				works := make([]types.Work, batchSize)
			for j := 0; j < batchSize; j++ {
				works[j] = &types.SimpleWork{
					ID:   fmt.Sprintf("batch-work-%d-%d", i, j),
					Size: 100,
					Task: func() error { return nil },
				}
			}
				distributor.DistributeBatch(works)
			}
		})
	}
}

func BenchmarkThroughputComparison(b *testing.B) {
	workerCounts := []int{2, 4, 8}
	workCounts := []int{1000, 5000, 10000}

	for _, workerCount := range workerCounts {
		for _, workCount := range workCounts {
			b.Run(fmt.Sprintf("Workers%d_Works%d", workerCount, workCount), func(b *testing.B) {
				benchmarkThroughput(b, workerCount, workCount)
			})
		}
	}
}

func benchmarkThroughput(b *testing.B, workerCount, workCount int) {
	pool := workers.NewWorkerPool(workerCount, workCount/workerCount+10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	distributor := distributors.NewRoundRobinDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
	}

	works := make([]types.Work, workCount)
	for i := 0; i < workCount; i++ {
		works[i] = &types.SimpleWork{
			ID:   fmt.Sprintf("throughput-work-%d", i),
			Size: 100,
			Task: func() error {
				// Simulate minimal processing
				time.Sleep(100 * time.Microsecond)
				return nil
			},
		}
	}

	b.ResetTimer()
	start := time.Now()

	var wg sync.WaitGroup
	for _, work := range works {
		wg.Add(1)
		go func(w types.Work) {
			defer wg.Done()
			distributor.Distribute(w)
		}(work)
	}

	wg.Wait()
	elapsed := time.Since(start)

	throughput := float64(workCount) / elapsed.Seconds()
	b.ReportMetric(throughput, "ops/sec")
	b.ReportMetric(elapsed.Seconds()*1000, "total_ms")
}

func BenchmarkLatencyMeasurement(b *testing.B) {
	pool := workers.NewWorkerPool(4, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		b.Fatalf("Failed to start worker pool: %v", err)
	}
	defer pool.Stop()

	monitor := monitoring.NewLoadMonitor(10 * time.Millisecond)
	monitor.Start(ctx)
	defer monitor.Stop()

	distributor := distributors.NewRoundRobinDistributor()
	for _, worker := range pool.GetWorkers() {
		distributor.AddWorker(worker)
		monitor.AddWorker(worker)
	}
	monitor.AddDistributor("round-robin", distributor)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			start := time.Now()
			work := &types.SimpleWork{
				ID:   fmt.Sprintf("latency-work-%d", i),
				Size: 100,
				Task: func() error {
					time.Sleep(1 * time.Millisecond)
					return nil
				},
			}
			distributor.Distribute(work)
			latency := time.Since(start)
			b.ReportMetric(float64(latency.Nanoseconds())/1e6, "latency_ms")
			i++
		}
	})

	// Wait for processing to complete
	time.Sleep(100 * time.Millisecond)

	metrics := monitor.GetMetrics()
	b.ReportMetric(float64(metrics.Overall.AverageLatency.Nanoseconds())/1e6, "avg_latency_ms")
	b.ReportMetric(float64(metrics.Overall.P99Latency.Nanoseconds())/1e6, "p99_latency_ms")
	b.ReportMetric(metrics.Overall.Throughput, "throughput_ops_sec")
}

func BenchmarkLoadBalanceEfficiency(b *testing.B) {
	patterns := []struct {
		name string
		createDistributor func() types.Distributor
	}{
		{
			name: "RoundRobin",
			createDistributor: func() types.Distributor {
				return distributors.NewRoundRobinDistributor()
			},
		},
		{
			name: "LoadBased",
			createDistributor: func() types.Distributor {
				return distributors.NewLoadBasedDistributor()
			},
		},
		{
			name: "Affinity",
			createDistributor: func() types.Distributor {
				return distributors.NewAffinityDistributor()
			},
		},
	}

	for _, pattern := range patterns {
		b.Run(pattern.name, func(b *testing.B) {
			pool := workers.NewWorkerPool(4, 100)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := pool.Start(ctx); err != nil {
				b.Fatalf("Failed to start worker pool: %v", err)
			}
			defer pool.Stop()

			distributor := pattern.createDistributor()
			for _, worker := range pool.GetWorkers() {
				distributor.AddWorker(worker)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create works with varying processing times
			works := make([]types.Work, 100)
			for j := 0; j < 100; j++ {
				processingTime := time.Duration(rand.Intn(10)+1) * time.Millisecond
				works[j] = &types.SimpleWork{
					ID:          fmt.Sprintf("balance-work-%d-%d", i, j),
					AffinityKey: fmt.Sprintf("key-%d", j%4),
					Size:        int64(processingTime.Nanoseconds() / 1e6),
					Task: func() error {
						time.Sleep(processingTime)
						return nil
					},
				}
			}

				// Distribute all works
				for _, work := range works {
					distributor.Distribute(work)
				}

				// Wait for processing
				time.Sleep(200 * time.Millisecond)

				// Measure load balance
				metrics := distributor.GetMetrics()
				b.ReportMetric(metrics.LoadBalance, "load_balance")
			}
		})
	}
}

func BenchmarkScalability(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("Workers%d", workerCount), func(b *testing.B) {
			pool := workers.NewWorkerPool(workerCount, 100)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := pool.Start(ctx); err != nil {
				b.Fatalf("Failed to start worker pool: %v", err)
			}
			defer pool.Stop()

			distributor := distributors.NewRoundRobinDistributor()
			for _, worker := range pool.GetWorkers() {
				distributor.AddWorker(worker)
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
				work := &types.SimpleWork{
					ID:   fmt.Sprintf("scale-work-%d", i),
					Size: 100,
					Task: func() error {
						time.Sleep(100 * time.Microsecond)
						return nil
					},
				}
					distributor.Distribute(work)
					i++
				}
			})
		})
	}
}