package workloads

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// CPUBoundTask represents a CPU-intensive computation task
type CPUBoundTask struct {
	ID          int
	Complexity  int
	Result      int64
	Duration    time.Duration
	StartTime   time.Time
	EndTime     time.Time
}

// RunCPUBoundWorkload executes CPU-intensive tasks across multiple goroutines
func RunCPUBoundWorkload(ctx context.Context, numGoroutines int, verbose bool) error {
	if verbose {
		fmt.Printf("Starting CPU-bound workload with %d goroutines\n", numGoroutines)
	}
	
	var wg sync.WaitGroup
	var completedTasks int64
	var totalOperations int64
	
	// Channel for task distribution
	taskChan := make(chan CPUBoundTask, numGoroutines*2)
	
	// Start worker goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			cpuWorker(ctx, workerID, taskChan, &completedTasks, &totalOperations, verbose)
		}(i)
	}
	
	// Generate tasks
	go func() {
		defer close(taskChan)
		taskID := 0
		
		for {
			select {
			case <-ctx.Done():
				return
			case taskChan <- CPUBoundTask{
				ID:         taskID,
				Complexity: 1000 + (taskID%5000), // Variable complexity
				StartTime:  time.Now(),
			}:
				taskID++
				if taskID%1000 == 0 && verbose {
					fmt.Printf("Generated %d CPU tasks\n", taskID)
				}
			}
		}
	}()
	
	// Wait for completion or context cancellation
	wg.Wait()
	
	if verbose {
		fmt.Printf("CPU workload completed. Tasks: %d, Operations: %d\n", 
			atomic.LoadInt64(&completedTasks), atomic.LoadInt64(&totalOperations))
	}
	
	return nil
}

// cpuWorker performs CPU-intensive computations
func cpuWorker(ctx context.Context, workerID int, taskChan <-chan CPUBoundTask, 
	completedTasks, totalOperations *int64, verbose bool) {
	
	var localCompleted int64
	var localOperations int64
	
	for {
		select {
		case <-ctx.Done():
			atomic.AddInt64(completedTasks, localCompleted)
			atomic.AddInt64(totalOperations, localOperations)
			return
			
		case task, ok := <-taskChan:
			if !ok {
				atomic.AddInt64(completedTasks, localCompleted)
				atomic.AddInt64(totalOperations, localOperations)
				return
			}
			
			// Perform CPU-intensive computation
			result := performCPUIntensiveTask(task.Complexity)
			task.Result = result
			task.EndTime = time.Now()
			task.Duration = task.EndTime.Sub(task.StartTime)
			
			localCompleted++
			localOperations += int64(task.Complexity)
			
			// Periodic verbose output
			if verbose && localCompleted%100 == 0 {
				fmt.Printf("Worker %d completed %d tasks\n", workerID, localCompleted)
			}
			
			// Yield occasionally to allow scheduler to make decisions
			if localCompleted%50 == 0 {
				runtime.Gosched()
			}
		}
	}
}

// performCPUIntensiveTask executes various CPU-intensive computations
func performCPUIntensiveTask(complexity int) int64 {
	var result int64
	
	// Mix of different CPU-intensive operations
	switch complexity % 4 {
	case 0:
		// Prime number generation
		result = generatePrimes(complexity)
	case 1:
		// Mathematical computations
		result = mathematicalOperations(complexity)
	case 2:
		// Matrix operations
		result = matrixOperations(complexity)
	case 3:
		// Cryptographic-like operations
		result = cryptographicOperations(complexity)
	}
	
	return result
}

// generatePrimes finds prime numbers up to n using sieve of Eratosthenes
func generatePrimes(n int) int64 {
	if n < 2 {
		return 0
	}
	
	// Limit n to prevent excessive memory usage
	if n > 100000 {
		n = 100000
	}
	
	sieve := make([]bool, n+1)
	for i := 2; i <= n; i++ {
		sieve[i] = true
	}
	
	for i := 2; i*i <= n; i++ {
		if sieve[i] {
			for j := i * i; j <= n; j += i {
				sieve[j] = false
			}
		}
	}
	
	var count int64
	for i := 2; i <= n; i++ {
		if sieve[i] {
			count++
		}
	}
	
	return count
}

// mathematicalOperations performs various mathematical computations
func mathematicalOperations(iterations int) int64 {
	var result float64
	
	for i := 0; i < iterations; i++ {
		x := float64(i)
		
		// Trigonometric functions
		result += math.Sin(x) * math.Cos(x)
		result += math.Sqrt(x + 1)
		result += math.Log(x + 1)
		result += math.Pow(x, 0.5)
		
		// Prevent overflow
		if math.IsInf(result, 0) || math.IsNaN(result) {
			result = float64(i)
		}
	}
	
	return int64(result)
}

// matrixOperations performs matrix multiplication and operations
func matrixOperations(size int) int64 {
	// Limit matrix size to prevent excessive memory usage
	if size > 100 {
		size = 100
	}
	if size < 10 {
		size = 10
	}
	
	// Create matrices
	matrixA := make([][]int, size)
	matrixB := make([][]int, size)
	result := make([][]int, size)
	
	for i := 0; i < size; i++ {
		matrixA[i] = make([]int, size)
		matrixB[i] = make([]int, size)
		result[i] = make([]int, size)
		
		for j := 0; j < size; j++ {
			matrixA[i][j] = i + j
			matrixB[i][j] = i - j
		}
	}
	
	// Matrix multiplication
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			for k := 0; k < size; k++ {
				result[i][j] += matrixA[i][k] * matrixB[k][j]
			}
		}
	}
	
	// Calculate sum as result
	var sum int64
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			sum += int64(result[i][j])
		}
	}
	
	return sum
}

// cryptographicOperations simulates cryptographic computations
func cryptographicOperations(iterations int) int64 {
	var hash int64 = 5381 // DJB2 hash initial value
	
	for i := 0; i < iterations; i++ {
		// Simulate hash computation
		data := int64(i)
		hash = ((hash << 5) + hash) + data
		
		// Simulate encryption-like operations
		for j := 0; j < 10; j++ {
			hash ^= hash >> 13
			hash *= 0x5bd1e995
			hash ^= hash >> 15
		}
		
		// Prevent overflow issues
		if hash < 0 {
			hash = -hash
		}
	}
	
	return hash
}

// CPUBoundBenchmark provides benchmarking for CPU-bound operations
type CPUBoundBenchmark struct {
	Name        string
	Complexity  int
	Goroutines  int
	Duration    time.Duration
	Operations  int64
	Throughput  float64
}

// BenchmarkCPUOperations benchmarks different CPU operations
func BenchmarkCPUOperations(ctx context.Context, numGoroutines int, duration time.Duration) []CPUBoundBenchmark {
	benchmarks := []CPUBoundBenchmark{
		{Name: "Prime Generation", Complexity: 10000},
		{Name: "Mathematical Ops", Complexity: 50000},
		{Name: "Matrix Operations", Complexity: 50},
		{Name: "Crypto Operations", Complexity: 10000},
	}
	
	for i := range benchmarks {
		benchmarks[i].Goroutines = numGoroutines
		benchmarks[i].Duration = duration
		
		// Run benchmark
		start := time.Now()
		var operations int64
		
		benchCtx, cancel := context.WithTimeout(ctx, duration)
		var wg sync.WaitGroup
		
		for j := 0; j < numGoroutines; j++ {
			wg.Add(1)
			go func(benchType int) {
				defer wg.Done()
				localOps := int64(0)
				
				for {
					select {
					case <-benchCtx.Done():
						atomic.AddInt64(&operations, localOps)
						return
					default:
						switch benchType {
						case 0:
							generatePrimes(benchmarks[i].Complexity)
						case 1:
							mathematicalOperations(benchmarks[i].Complexity)
						case 2:
							matrixOperations(benchmarks[i].Complexity)
						case 3:
							cryptographicOperations(benchmarks[i].Complexity)
						}
						localOps++
					}
				}
			}(i)
		}
		
		wg.Wait()
		cancel()
		
		elapsed := time.Since(start)
		benchmarks[i].Operations = operations
		benchmarks[i].Throughput = float64(operations) / elapsed.Seconds()
	}
	
	return benchmarks
}

// GetCPUWorkloadStats returns statistics about CPU workload performance
func GetCPUWorkloadStats() map[string]interface{} {
	return map[string]interface{}{
		"workload_type":    "cpu_bound",
		"characteristics": "High CPU utilization, minimal I/O, compute-intensive",
		"scheduler_impact": "Even load distribution, preemption-based scheduling",
		"optimization_tips": []string{
			"Use GOMAXPROCS equal to CPU cores for CPU-bound tasks",
			"Minimize goroutine creation overhead",
			"Consider work-stealing efficiency",
			"Monitor CPU cache efficiency",
			"Use compiler optimizations",
		},
	}
}