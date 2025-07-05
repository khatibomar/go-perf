package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Benchmark CPU-bound vs I/O-bound tasks
func BenchmarkCPUBoundTask(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cpuBoundTask(i, 10000)
	}
}

func BenchmarkIOBoundTask(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ioBoundTask(i, time.Microsecond*10)
	}
}

// Benchmark goroutine creation overhead
func BenchmarkGoroutineCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Minimal work
		}()
		wg.Wait()
	}
}

// Benchmark worker pool vs unlimited goroutines
func BenchmarkUnlimitedGoroutines(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Simulate small work
				sum := 0
				for k := 0; k < 1000; k++ {
					sum += k
				}
				_ = sum
			}()
		}
		wg.Wait()
	}
}

func BenchmarkWorkerPool(b *testing.B) {
	pool := NewWorkerPool(runtime.NumCPU(), 100)
	pool.Start()
	defer pool.Stop()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			pool.Submit(func() {
				defer wg.Done()
				// Simulate small work
				sum := 0
				for k := 0; k < 1000; k++ {
					sum += k
				}
				_ = sum
			})
		}
		wg.Wait()
	}
}

// Benchmark synchronization mechanisms
func BenchmarkChannelBasedCounter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = channelBasedCounter(1000)
	}
}

func BenchmarkMutexBasedCounter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mutexBasedCounter(1000)
	}
}

func BenchmarkAtomicBasedCounter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = atomicBasedCounter(1000)
	}
}

// Benchmark context overhead
func BenchmarkWithoutContext(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		work := make(chan int, 10)
		results := make(chan int, 10)
		
		// Start worker without context
		go func() {
			for job := range work {
				results <- job * job
			}
			close(results)
		}()
		
		// Send work
		for j := 0; j < 10; j++ {
			work <- j
		}
		close(work)
		
		// Collect results
		for range results {
			// Consume results
		}
	}
}

func BenchmarkWithContext(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		work := make(chan int, 10)
		results := make(chan int, 10)
		
		// Start context-aware worker
		go contextAwareWorker(ctx, 0, work, results)
		
		// Send work
		for j := 0; j < 10; j++ {
			work <- j
		}
		close(work)
		
		// Collect results
		for j := 0; j < 10; j++ {
			<-results
		}
		
		cancel()
	}
}

// Benchmark GOMAXPROCS impact
func BenchmarkGOMAXPROCS1(b *testing.B) {
	original := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(original)
	
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 4; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cpuBoundTask(0, 10000)
			}()
		}
		wg.Wait()
	}
}

func BenchmarkGOMAXPROCSDefault(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 4; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cpuBoundTask(0, 10000)
			}()
		}
		wg.Wait()
	}
}

// Benchmark goroutine scheduling overhead
func BenchmarkGoroutineScheduling(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 1000; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				runtime.Gosched() // Force scheduling
			}()
		}
		wg.Wait()
	}
}

// Benchmark channel operations
func BenchmarkChannelSendReceive(b *testing.B) {
	ch := make(chan int)
	
	go func() {
		for i := 0; i < b.N; i++ {
			ch <- i
		}
	}()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		<-ch
	}
}

func BenchmarkBufferedChannelSendReceive(b *testing.B) {
	ch := make(chan int, 100)
	
	go func() {
		for i := 0; i < b.N; i++ {
			ch <- i
		}
	}()
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		<-ch
	}
}

// Comparative benchmarks
func BenchmarkSynchronizationComparison(b *testing.B) {
	b.Run("Channel", BenchmarkChannelBasedCounter)
	b.Run("Mutex", BenchmarkMutexBasedCounter)
	b.Run("Atomic", BenchmarkAtomicBasedCounter)
}

func BenchmarkGoroutinePatterns(b *testing.B) {
	b.Run("Unlimited", BenchmarkUnlimitedGoroutines)
	b.Run("WorkerPool", BenchmarkWorkerPool)
}

func BenchmarkContextOverhead(b *testing.B) {
	b.Run("WithoutContext", BenchmarkWithoutContext)
	b.Run("WithContext", BenchmarkWithContext)
}

func BenchmarkChannelTypes(b *testing.B) {
	b.Run("Unbuffered", BenchmarkChannelSendReceive)
	b.Run("Buffered", BenchmarkBufferedChannelSendReceive)
}

// Benchmark different goroutine counts
func BenchmarkGoroutineCounts(b *testing.B) {
	counts := []int{1, 2, 4, 8, 16, 32}
	
	for _, count := range counts {
		b.Run(fmt.Sprintf("Goroutines_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for j := 0; j < count; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						cpuBoundTask(0, 1000)
					}()
				}
				wg.Wait()
			}
		})
	}
}

// Test correctness
func TestSchedulerAnalysis(t *testing.T) {
	// Test CPU-bound task
	cpuBoundTask(0, 1000) // Should not panic
	
	// Test I/O-bound task
	ioBoundTask(0, time.Microsecond) // Should not panic
	
	// Test worker pool
	pool := NewWorkerPool(2, 10)
	pool.Start()
	
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			time.Sleep(time.Millisecond)
		})
	}
	wg.Wait()
	pool.Stop()
	
	// Test context-aware worker
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	
	work := make(chan int, 5)
	results := make(chan int, 5)
	
	go contextAwareWorker(ctx, 0, work, results)
	
	for i := 0; i < 3; i++ {
		work <- i
	}
	close(work)
	
	// Collect some results
	for i := 0; i < 3; i++ {
		select {
		case result := <-results:
			if result < 0 {
				t.Errorf("Unexpected negative result: %d", result)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for result")
		}
	}
	
	// Test counters
	iterations := 1000
	
	channelResult := channelBasedCounter(iterations)
	if channelResult != int64(iterations) {
		t.Errorf("Channel counter result: %d, expected: %d", channelResult, iterations)
	}
	
	mutexResult := mutexBasedCounter(iterations)
	if mutexResult != int64(iterations) {
		t.Errorf("Mutex counter result: %d, expected: %d", mutexResult, iterations)
	}
	
	atomicResult := atomicBasedCounter(iterations)
	if atomicResult != int64(iterations) {
		t.Errorf("Atomic counter result: %d, expected: %d", atomicResult, iterations)
	}
	
	// Test goroutine leak prevention
	initialGoroutines := runtime.NumGoroutine()
	properGoroutine()
	time.Sleep(time.Millisecond * 50) // Allow goroutine to complete
	runtime.GC()
	time.Sleep(time.Millisecond * 50)
	finalGoroutines := runtime.NumGoroutine()
	
	if finalGoroutines > initialGoroutines+1 {
		t.Errorf("Potential goroutine leak: initial=%d, final=%d", initialGoroutines, finalGoroutines)
	}
}

// Benchmark memory allocation in goroutines
func BenchmarkGoroutineMemoryAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Allocate memory in goroutine
				data := make([]int, 1000)
				for k := range data {
					data[k] = k
				}
				_ = data
			}()
		}
		wg.Wait()
	}
}