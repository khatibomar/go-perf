package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Goroutine scheduler analysis and performance patterns

// Pattern 1: CPU-bound vs I/O-bound goroutines
func cpuBoundTask(iterations int) {
	sum := 0
	for i := 0; i < iterations; i++ {
		sum += i * i
	}
	_ = sum // Prevent optimization
}

func ioBoundTask(duration time.Duration) {
	time.Sleep(duration) // Simulates I/O wait
}

// Pattern 2: Goroutine pool vs unlimited goroutines
type WorkerPool struct {
	workerCount int
	jobQueue    chan func()
	wg          sync.WaitGroup
}

func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	return &WorkerPool{
		workerCount: workerCount,
		jobQueue:    make(chan func(), queueSize),
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	for job := range wp.jobQueue {
		job()
	}
}

func (wp *WorkerPool) Submit(job func()) {
	wp.jobQueue <- job
}

func (wp *WorkerPool) Stop() {
	close(wp.jobQueue)
	wp.wg.Wait()
}

// Pattern 3: Context-aware goroutines
func contextAwareWorker(ctx context.Context, id int, work chan int, results chan int) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-work:
			if !ok {
				return
			}
			// Simulate work
			time.Sleep(time.Millisecond * 10)
			results <- job * job
		}
	}
}

// Pattern 4: Goroutine leaks demonstration
func leakyGoroutine() {
	ch := make(chan int)
	go func() {
		// This goroutine will block forever
		<-ch // No one sends to this channel
	}()
	// Channel goes out of scope, goroutine leaks
}

func properGoroutine() {
	ch := make(chan int, 1) // Buffered channel
	go func() {
		select {
		case <-ch:
			// Process message
		case <-time.After(time.Second):
			// Timeout to prevent leak
			return
		}
	}()
	ch <- 42 // Send value
}

// Pattern 5: GOMAXPROCS impact
func demonstrateGOMAXPROCS() {
	fmt.Printf("Current GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("Number of CPUs: %d\n", runtime.NumCPU())

	// Test with different GOMAXPROCS values
	original := runtime.GOMAXPROCS(0)

	for _, procs := range []int{1, 2, runtime.NumCPU(), runtime.NumCPU() * 2} {
		runtime.GOMAXPROCS(procs)
		start := time.Now()

		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				cpuBoundTask(1000000)
			}(i)
		}
		wg.Wait()

		elapsed := time.Since(start)
		fmt.Printf("GOMAXPROCS=%d: %v\n", procs, elapsed)
	}

	// Restore original value
	runtime.GOMAXPROCS(original)
}

// Pattern 6: Scheduler statistics
func printSchedulerStats() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
	fmt.Printf("OS Threads: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("CGO Calls: %d\n", runtime.NumCgoCall())
}

// Pattern 7: Channel vs Mutex performance
func channelBasedCounter(iterations int) int64 {
	ch := make(chan int64, 1)
	ch <- 0 // Initialize counter

	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	iterationsPerWorker := iterations / workers
	remainder := iterations % workers

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workLoad := iterationsPerWorker
			if workerID < remainder {
				workLoad++ // Distribute remainder among first workers
			}
			for j := 0; j < workLoad; j++ {
				counter := <-ch
				counter++
				ch <- counter
			}
		}(i)
	}

	wg.Wait()
	return <-ch
}

func mutexBasedCounter(iterations int) int64 {
	var counter int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	iterationsPerWorker := iterations / workers
	remainder := iterations % workers

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workLoad := iterationsPerWorker
			if workerID < remainder {
				workLoad++ // Distribute remainder among first workers
			}
			for j := 0; j < workLoad; j++ {
				mu.Lock()
				counter++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return counter
}

func atomicBasedCounter(iterations int) int64 {
	var counter int64
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	iterationsPerWorker := iterations / workers
	remainder := iterations % workers

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workLoad := iterationsPerWorker
			if workerID < remainder {
				workLoad++ // Distribute remainder among first workers
			}
			for j := 0; j < workLoad; j++ {
				atomic.AddInt64(&counter, 1)
			}
		}(i)
	}

	wg.Wait()
	return counter
}

// Pattern 8: Goroutine scheduling fairness
func demonstrateSchedulingFairness() {
	var counters [4]int64
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000000; j++ {
				atomic.AddInt64(&counters[id], 1)
				// Yield occasionally to demonstrate fairness
				if j%10000 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}

	wg.Wait()

	fmt.Println("Goroutine execution counts (should be roughly equal):")
	for i, count := range counters {
		fmt.Printf("Goroutine %d: %d\n", i, count)
	}
}

func timeOperation(name string, operation func()) {
	start := time.Now()
	operation()
	duration := time.Since(start)
	fmt.Printf("%s took: %v\n", name, duration)
}

func main() {
	fmt.Println("=== Go Scheduler Analysis ===")
	fmt.Println()

	printSchedulerStats()
	fmt.Println()

	// Pattern 1: CPU vs I/O bound tasks
	fmt.Println("=== CPU-bound vs I/O-bound Tasks ===")
	start := time.Now()

	// CPU-bound tasks
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cpuBoundTask(1000000)
		}(i)
	}
	wg.Wait()
	fmt.Printf("CPU-bound tasks completed in: %v\n", time.Since(start))

	// I/O-bound tasks
	start = time.Now()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ioBoundTask(time.Millisecond * 100)
		}(i)
	}
	wg.Wait()
	fmt.Printf("I/O-bound tasks completed in: %v\n", time.Since(start))
	fmt.Println()

	// Pattern 2: Worker pool vs unlimited goroutines
	fmt.Println("=== Worker Pool vs Unlimited Goroutines ===")

	// Unlimited goroutines
	timeOperation("Unlimited goroutines", func() {
		var wg sync.WaitGroup
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Millisecond)
			}(i)
		}
		wg.Wait()
	})

	// Worker pool
	timeOperation("Worker pool", func() {
		pool := NewWorkerPool(runtime.NumCPU(), 100)
		pool.Start()

		var wg sync.WaitGroup
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			pool.Submit(func() {
				defer wg.Done()
				time.Sleep(time.Millisecond)
			})
		}
		wg.Wait()
		pool.Stop()
	})
	fmt.Println()

	// Pattern 3: Context-aware goroutines
	fmt.Println("=== Context-aware Goroutines ===")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	work := make(chan int, 10)
	results := make(chan int, 10)

	// Start workers
	for i := 0; i < 3; i++ {
		go contextAwareWorker(ctx, i, work, results)
	}

	// Send work
	go func() {
		for i := 0; i < 20; i++ {
			select {
			case work <- i:
			case <-ctx.Done():
				return
			}
		}
		close(work)
	}()

	// Collect results
	go func() {
		for result := range results {
			fmt.Printf("Result: %d\n", result)
		}
	}()

	<-ctx.Done()
	close(results)
	fmt.Println()

	// Pattern 4: Goroutine leak demonstration
	fmt.Println("=== Goroutine Leak Demonstration ===")
	initialGoroutines := runtime.NumGoroutine()
	fmt.Printf("Initial goroutines: %d\n", initialGoroutines)

	// Create leaky goroutines
	for i := 0; i < 10; i++ {
		leakyGoroutine()
	}
	time.Sleep(time.Millisecond * 100) // Let goroutines start
	leakyGoroutines := runtime.NumGoroutine()
	fmt.Printf("After leaky goroutines: %d\n", leakyGoroutines)

	// Create proper goroutines
	for i := 0; i < 10; i++ {
		properGoroutine()
	}
	time.Sleep(time.Millisecond * 100) // Let goroutines complete
	runtime.GC()                       // Force garbage collection
	time.Sleep(time.Millisecond * 100)
	finalGoroutines := runtime.NumGoroutine()
	fmt.Printf("After proper goroutines: %d\n", finalGoroutines)
	fmt.Println()

	// Pattern 5: GOMAXPROCS impact
	fmt.Println("=== GOMAXPROCS Impact ===")
	demonstrateGOMAXPROCS()
	fmt.Println()

	// Pattern 6: Channel vs Mutex vs Atomic performance
	fmt.Println("=== Synchronization Performance ===")
	iterations := 100000

	timeOperation("Channel-based counter", func() {
		result := channelBasedCounter(iterations)
		fmt.Printf("Channel result: %d\n", result)
	})

	timeOperation("Mutex-based counter", func() {
		result := mutexBasedCounter(iterations)
		fmt.Printf("Mutex result: %d\n", result)
	})

	timeOperation("Atomic-based counter", func() {
		result := atomicBasedCounter(iterations)
		fmt.Printf("Atomic result: %d\n", result)
	})
	fmt.Println()

	// Pattern 7: Scheduling fairness
	fmt.Println("=== Scheduling Fairness ===")
	demonstrateSchedulingFairness()
	fmt.Println()

	printSchedulerStats()

	fmt.Println("=== Key Takeaways ===")
	fmt.Println("1. I/O-bound tasks benefit from more goroutines than CPU cores")
	fmt.Println("2. CPU-bound tasks should typically match the number of CPU cores")
	fmt.Println("3. Worker pools prevent goroutine explosion and improve resource management")
	fmt.Println("4. Always use context for cancellation and timeouts")
	fmt.Println("5. Avoid goroutine leaks by ensuring all goroutines can exit")
	fmt.Println("6. GOMAXPROCS should typically equal the number of CPU cores")
	fmt.Println("7. Atomic operations are fastest, followed by mutexes, then channels")
	fmt.Println("8. The Go scheduler provides fairness across goroutines")
}
