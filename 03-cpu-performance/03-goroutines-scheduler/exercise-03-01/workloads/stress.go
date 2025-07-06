package workloads

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// StressTest represents different types of scheduler stress tests
type StressTest struct {
	Name        string
	Description string
	Goroutines  int
	Duration    time.Duration
	TestFunc    func(context.Context, int, time.Duration, bool) (*StressResult, error)
}

// StressResult contains the results of a stress test
type StressResult struct {
	TestName           string
	Goroutines         int
	Duration           time.Duration
	TasksCompleted     int64
	TasksPerSecond     float64
	AvgLatency         time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
	GoroutineOverhead  time.Duration
	ContextSwitches    int64
	MemoryAllocations  int64
	GCPauses           int
	CPUUtilization     float64
	SchedulerEfficiency float64
	Errors             int64
}

// GetStressTests returns predefined stress test scenarios
func GetStressTests() []StressTest {
	return []StressTest{
		{
			Name:        "High Goroutine Count",
			Description: "Tests scheduler with very high number of goroutines",
			TestFunc:    runHighGoroutineStress,
		},
		{
			Name:        "Rapid Creation/Destruction",
			Description: "Tests scheduler with rapid goroutine creation and destruction",
			TestFunc:    runRapidCreationStress,
		},
		{
			Name:        "Channel Contention",
			Description: "Tests scheduler with heavy channel contention",
			TestFunc:    runChannelContentionStress,
		},
		{
			Name:        "Mutex Contention",
			Description: "Tests scheduler with heavy mutex contention",
			TestFunc:    runMutexContentionStress,
		},
		{
			Name:        "Memory Pressure",
			Description: "Tests scheduler under memory pressure",
			TestFunc:    runMemoryPressureStress,
		},
		{
			Name:        "Mixed Blocking Operations",
			Description: "Tests scheduler with various blocking operations",
			TestFunc:    runMixedBlockingStress,
		},
		{
			Name:        "Work Stealing Efficiency",
			Description: "Tests work-stealing scheduler efficiency",
			TestFunc:    runWorkStealingStress,
		},
	}
}

// RunStressWorkload executes comprehensive scheduler stress tests
func RunStressWorkload(ctx context.Context, numGoroutines int, verbose bool) error {
	if verbose {
		fmt.Printf("Starting scheduler stress tests with base goroutines: %d\n", numGoroutines)
		fmt.Printf("GOMAXPROCS: %d, NumCPU: %d\n", runtime.GOMAXPROCS(0), runtime.NumCPU())
	}
	
	tests := GetStressTests()
	results := make([]*StressResult, len(tests))
	
	// Run each stress test
	for i, test := range tests {
		if verbose {
			fmt.Printf("\n=== Running Stress Test: %s ===\n", test.Name)
			fmt.Printf("Description: %s\n", test.Description)
		}
		
		// Adjust goroutine count for specific tests
		testGoroutines := numGoroutines
		switch test.Name {
		case "High Goroutine Count":
			testGoroutines = numGoroutines * 10 // 10x more goroutines
		case "Rapid Creation/Destruction":
			testGoroutines = numGoroutines * 5 // 5x more for rapid creation
		}
		
		testDuration := 10 * time.Second
		
		// Force GC before test
		runtime.GC()
		runtime.GC()
		
		start := time.Now()
		result, err := test.TestFunc(ctx, testGoroutines, testDuration, verbose)
		if err != nil {
			fmt.Printf("Error in stress test %s: %v\n", test.Name, err)
			continue
		}
		
		result.TestName = test.Name
		result.Duration = time.Since(start)
		results[i] = result
		
		if verbose {
			printStressResult(result)
		}
		
		// Brief pause between tests
		time.Sleep(2 * time.Second)
	}
	
	if verbose {
		printStressSummary(results)
	}
	
	return nil
}

// runHighGoroutineStress tests scheduler with very high goroutine count
func runHighGoroutineStress(ctx context.Context, numGoroutines int, duration time.Duration, verbose bool) (*StressResult, error) {
	result := &StressResult{Goroutines: numGoroutines}
	
	var tasksCompleted int64
	var totalLatency int64
	var maxLatency int64
	var minLatency int64 = int64(time.Hour) // Initialize to large value
	var errors int64
	
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	var wg sync.WaitGroup
	taskChan := make(chan int, numGoroutines*2)
	
	// Start many goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				case taskID, ok := <-taskChan:
					if !ok {
						return
					}
					
					start := time.Now()
					
					// Simple CPU work
					sum := 0
					for j := 0; j < 1000; j++ {
						sum += j * taskID
					}
					
					// Yield to scheduler
					runtime.Gosched()
					
					latency := time.Since(start)
					latencyNs := int64(latency)
					
					atomic.AddInt64(&tasksCompleted, 1)
					atomic.AddInt64(&totalLatency, latencyNs)
					
					// Update max latency
					for {
						current := atomic.LoadInt64(&maxLatency)
						if latencyNs <= current || atomic.CompareAndSwapInt64(&maxLatency, current, latencyNs) {
							break
						}
					}
					
					// Update min latency
					for {
						current := atomic.LoadInt64(&minLatency)
						if latencyNs >= current || atomic.CompareAndSwapInt64(&minLatency, current, latencyNs) {
							break
						}
					}
					
					if sum < 0 { // Prevent optimization
						atomic.AddInt64(&errors, 1)
					}
				}
			}
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
			case taskChan <- taskID:
				taskID++
			}
		}
	}()
	
	wg.Wait()
	
	// Calculate results
	completed := atomic.LoadInt64(&tasksCompleted)
	result.TasksCompleted = completed
	result.TasksPerSecond = float64(completed) / duration.Seconds()
	result.Errors = atomic.LoadInt64(&errors)
	
	if completed > 0 {
		result.AvgLatency = time.Duration(atomic.LoadInt64(&totalLatency) / completed)
		result.MaxLatency = time.Duration(atomic.LoadInt64(&maxLatency))
		result.MinLatency = time.Duration(atomic.LoadInt64(&minLatency))
	}
	
	return result, nil
}

// runRapidCreationStress tests rapid goroutine creation and destruction
func runRapidCreationStress(ctx context.Context, numGoroutines int, duration time.Duration, verbose bool) (*StressResult, error) {
	result := &StressResult{Goroutines: numGoroutines}
	
	var goroutinesCreated int64
	var goroutinesCompleted int64
	var totalCreationTime int64
	var errors int64
	
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// Rapid creation loop
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(creatorID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Create and immediately destroy goroutines
					start := time.Now()
					
					var innerWg sync.WaitGroup
					batchSize := 10 + rand.Intn(20) // 10-30 goroutines per batch
					
					for j := 0; j < batchSize; j++ {
						innerWg.Add(1)
						go func(taskID int) {
							defer innerWg.Done()
							
							// Short-lived work
							sum := 0
							for k := 0; k < 100; k++ {
								sum += k * taskID
							}
							
							atomic.AddInt64(&goroutinesCompleted, 1)
							
							if sum < 0 {
								atomic.AddInt64(&errors, 1)
							}
						}(j)
					}
					
					innerWg.Wait()
					
					creationTime := time.Since(start)
					atomic.AddInt64(&goroutinesCreated, int64(batchSize))
					atomic.AddInt64(&totalCreationTime, int64(creationTime))
					
					// Brief pause
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Calculate results
	created := atomic.LoadInt64(&goroutinesCreated)
	completed := atomic.LoadInt64(&goroutinesCompleted)
	
	result.TasksCompleted = completed
	result.TasksPerSecond = float64(completed) / duration.Seconds()
	result.Errors = atomic.LoadInt64(&errors)
	
	if created > 0 {
		totalTime := atomic.LoadInt64(&totalCreationTime)
		result.GoroutineOverhead = time.Duration(totalTime / created)
	}
	
	return result, nil
}

// runChannelContentionStress tests heavy channel contention
func runChannelContentionStress(ctx context.Context, numGoroutines int, duration time.Duration, verbose bool) (*StressResult, error) {
	result := &StressResult{Goroutines: numGoroutines}
	
	var messagesProcessed int64
	var totalLatency int64
	var errors int64
	
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	// Create multiple channels with different buffer sizes
	channels := make([]chan int, 5)
	channels[0] = make(chan int)      // Unbuffered
	channels[1] = make(chan int, 1)   // Small buffer
	channels[2] = make(chan int, 10)  // Medium buffer
	channels[3] = make(chan int, 100) // Large buffer
	channels[4] = make(chan int, 1000) // Very large buffer
	
	var wg sync.WaitGroup
	
	// Start producers
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			messageID := 0
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Select random channel
					chIdx := rand.Intn(len(channels))
					ch := channels[chIdx]
					
					select {
					case ch <- messageID:
						messageID++
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						// Timeout to prevent blocking
						atomic.AddInt64(&errors, 1)
					}
				}
			}
		}(i)
	}
	
	// Start consumers
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Select random channel
					chIdx := rand.Intn(len(channels))
					ch := channels[chIdx]
					
					start := time.Now()
					select {
					case msg := <-ch:
						latency := time.Since(start)
						
						// Process message
						result := msg * msg
						if result < 0 {
							atomic.AddInt64(&errors, 1)
						}
						
						atomic.AddInt64(&messagesProcessed, 1)
						atomic.AddInt64(&totalLatency, int64(latency))
						
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond):
						// Timeout to prevent blocking
						atomic.AddInt64(&errors, 1)
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Close channels
	for _, ch := range channels {
		close(ch)
	}
	
	// Calculate results
	processed := atomic.LoadInt64(&messagesProcessed)
	result.TasksCompleted = processed
	result.TasksPerSecond = float64(processed) / duration.Seconds()
	result.Errors = atomic.LoadInt64(&errors)
	
	if processed > 0 {
		result.AvgLatency = time.Duration(atomic.LoadInt64(&totalLatency) / processed)
	}
	
	return result, nil
}

// runMutexContentionStress tests heavy mutex contention
func runMutexContentionStress(ctx context.Context, numGoroutines int, duration time.Duration, verbose bool) (*StressResult, error) {
	result := &StressResult{Goroutines: numGoroutines}
	
	var operationsCompleted int64
	var totalWaitTime int64
	var errors int64
	
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	// Shared resources with different contention levels
	type SharedResource struct {
		mu    sync.Mutex
		value int64
		name  string
	}
	
	resources := []*SharedResource{
		{name: "high_contention"},
		{name: "medium_contention"},
		{name: "low_contention"},
	}
	
	var wg sync.WaitGroup
	
	// Start workers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Select resource based on contention pattern
					var resource *SharedResource
					switch {
					case rand.Float64() < 0.7: // 70% high contention
						resource = resources[0]
					case rand.Float64() < 0.9: // 20% medium contention
						resource = resources[1]
					default: // 10% low contention
						resource = resources[2]
					}
					
					start := time.Now()
					resource.mu.Lock()
					waitTime := time.Since(start)
					
					// Critical section work
					oldValue := resource.value
					for j := 0; j < 100; j++ {
						resource.value += int64(j)
					}
					resource.value -= oldValue + 4950 // Should result in original value
					
					if resource.value != oldValue {
						atomic.AddInt64(&errors, 1)
					}
					
					resource.mu.Unlock()
					
					atomic.AddInt64(&operationsCompleted, 1)
					atomic.AddInt64(&totalWaitTime, int64(waitTime))
					
					// Brief yield
					runtime.Gosched()
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Calculate results
	completed := atomic.LoadInt64(&operationsCompleted)
	result.TasksCompleted = completed
	result.TasksPerSecond = float64(completed) / duration.Seconds()
	result.Errors = atomic.LoadInt64(&errors)
	
	if completed > 0 {
		result.AvgLatency = time.Duration(atomic.LoadInt64(&totalWaitTime) / completed)
	}
	
	return result, nil
}

// runMemoryPressureStress tests scheduler under memory pressure
func runMemoryPressureStress(ctx context.Context, numGoroutines int, duration time.Duration, verbose bool) (*StressResult, error) {
	result := &StressResult{Goroutines: numGoroutines}
	
	var allocationsCompleted int64
	var totalAllocTime int64
	var errors int64
	var gcCount int64
	
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// Start memory-intensive workers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			var allocations [][]byte
			maxAllocations := 100
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()
					
					// Allocate memory of varying sizes
					size := 1024 + rand.Intn(8192) // 1KB to 9KB
					data := make([]byte, size)
					
					// Fill with data to prevent optimization
					for j := range data {
						data[j] = byte(j % 256)
					}
					
					allocations = append(allocations, data)
					
					// Limit memory usage per goroutine
					if len(allocations) > maxAllocations {
						// Free oldest allocations
						toFree := len(allocations) - maxAllocations/2
						allocations = allocations[toFree:]
						runtime.GC() // Force GC
						atomic.AddInt64(&gcCount, 1)
					}
					
					allocTime := time.Since(start)
					atomic.AddInt64(&allocationsCompleted, 1)
					atomic.AddInt64(&totalAllocTime, int64(allocTime))
					
					// Verify data integrity occasionally
					if rand.Float64() < 0.1 && len(allocations) > 0 {
						testData := allocations[rand.Intn(len(allocations))]
						for j, b := range testData {
							if b != byte(j%256) {
								atomic.AddInt64(&errors, 1)
								break
							}
						}
					}
					
					// Brief pause
					time.Sleep(time.Microsecond * 100)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Calculate results
	completed := atomic.LoadInt64(&allocationsCompleted)
	result.TasksCompleted = completed
	result.TasksPerSecond = float64(completed) / duration.Seconds()
	result.Errors = atomic.LoadInt64(&errors)
	result.GCPauses = int(atomic.LoadInt64(&gcCount))
	result.MemoryAllocations = completed
	
	if completed > 0 {
		result.AvgLatency = time.Duration(atomic.LoadInt64(&totalAllocTime) / completed)
	}
	
	return result, nil
}

// runMixedBlockingStress tests various blocking operations
func runMixedBlockingStress(ctx context.Context, numGoroutines int, duration time.Duration, verbose bool) (*StressResult, error) {
	result := &StressResult{Goroutines: numGoroutines}
	
	var operationsCompleted int64
	var totalBlockTime int64
	var errors int64
	
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// Start workers with mixed blocking operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()
					
					// Random blocking operation
					switch rand.Intn(4) {
					case 0: // Sleep
						time.Sleep(time.Microsecond * time.Duration(rand.Intn(1000)))
						
					case 1: // Channel operation
						ch := make(chan int, 1)
						go func() {
							time.Sleep(time.Microsecond * time.Duration(rand.Intn(100)))
							ch <- 1
						}()
						<-ch
						
					case 2: // Mutex lock
						var mu sync.Mutex
						mu.Lock()
						time.Sleep(time.Microsecond * time.Duration(rand.Intn(50)))
						mu.Unlock()
						
					case 3: // WaitGroup
						var innerWg sync.WaitGroup
						innerWg.Add(1)
						go func() {
							time.Sleep(time.Microsecond * time.Duration(rand.Intn(200)))
							innerWg.Done()
						}()
						innerWg.Wait()
					}
					
					blockTime := time.Since(start)
					atomic.AddInt64(&operationsCompleted, 1)
					atomic.AddInt64(&totalBlockTime, int64(blockTime))
					
					// Verify we're still responsive
					select {
					case <-ctx.Done():
						return
					default:
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Calculate results
	completed := atomic.LoadInt64(&operationsCompleted)
	result.TasksCompleted = completed
	result.TasksPerSecond = float64(completed) / duration.Seconds()
	result.Errors = atomic.LoadInt64(&errors)
	
	if completed > 0 {
		result.AvgLatency = time.Duration(atomic.LoadInt64(&totalBlockTime) / completed)
	}
	
	return result, nil
}

// runWorkStealingStress tests work-stealing efficiency
func runWorkStealingStress(ctx context.Context, numGoroutines int, duration time.Duration, verbose bool) (*StressResult, error) {
	result := &StressResult{Goroutines: numGoroutines}
	
	var tasksCompleted int64
	var totalLatency int64
	var workSteals int64
	var errors int64
	
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	// Create per-worker queues to simulate work-stealing
	workerQueues := make([]chan int, numGoroutines)
	for i := range workerQueues {
		workerQueues[i] = make(chan int, 100)
	}
	
	var wg sync.WaitGroup
	
	// Start workers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			myQueue := workerQueues[workerID]
			
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-myQueue:
					if !ok {
						return
					}
					
					start := time.Now()
					
					// Process task
					result := performCPUIntensiveTask(1000 + rand.Intn(2000))
					
					latency := time.Since(start)
					atomic.AddInt64(&tasksCompleted, 1)
					atomic.AddInt64(&totalLatency, int64(latency))
					
					if result < 0 {
						atomic.AddInt64(&errors, 1)
					}
					
				default:
					// Try to steal work from other workers
					if rand.Float64() < 0.3 { // 30% chance to steal
						victimID := rand.Intn(numGoroutines)
						if victimID != workerID {
							victimQueue := workerQueues[victimID]
							
							select {
							case stolenTask := <-victimQueue:
								// Successfully stole work
								atomic.AddInt64(&workSteals, 1)
								
								start := time.Now()
								result := performCPUIntensiveTask(1000 + rand.Intn(2000))
								latency := time.Since(start)
								
								atomic.AddInt64(&tasksCompleted, 1)
								atomic.AddInt64(&totalLatency, int64(latency))
								
								if result < 0 || stolenTask < 0 {
									atomic.AddInt64(&errors, 1)
								}
								
							default:
								// No work to steal, yield
								runtime.Gosched()
							}
						}
					}
				}
			}
		}(i)
	}
	
	// Generate work with uneven distribution
	go func() {
		taskID := 0
		
		for {
			select {
			case <-ctx.Done():
				// Close all queues
				for _, queue := range workerQueues {
					close(queue)
				}
				return
			default:
				// Create uneven work distribution
				if rand.Float64() < 0.7 {
					// 70% of work goes to first half of workers
					workerID := rand.Intn(numGoroutines / 2)
					select {
					case workerQueues[workerID] <- taskID:
						taskID++
					case <-time.After(time.Microsecond):
						// Queue full, skip
					}
				} else {
					// 30% of work goes to second half
					workerID := numGoroutines/2 + rand.Intn(numGoroutines/2)
					select {
					case workerQueues[workerID] <- taskID:
						taskID++
					case <-time.After(time.Microsecond):
						// Queue full, skip
					}
				}
			}
		}
	}()
	
	wg.Wait()
	
	// Calculate results
	completed := atomic.LoadInt64(&tasksCompleted)
	steals := atomic.LoadInt64(&workSteals)
	
	result.TasksCompleted = completed
	result.TasksPerSecond = float64(completed) / duration.Seconds()
	result.Errors = atomic.LoadInt64(&errors)
	result.ContextSwitches = steals
	
	if completed > 0 {
		result.AvgLatency = time.Duration(atomic.LoadInt64(&totalLatency) / completed)
		result.SchedulerEfficiency = float64(steals) / float64(completed) * 100
	}
	
	return result, nil
}

// printStressResult prints detailed results of a stress test
func printStressResult(result *StressResult) {
	fmt.Printf("\n--- %s Results ---\n", result.TestName)
	fmt.Printf("Goroutines: %d\n", result.Goroutines)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Tasks Completed: %d\n", result.TasksCompleted)
	fmt.Printf("Tasks/Second: %.2f\n", result.TasksPerSecond)
	fmt.Printf("Average Latency: %v\n", result.AvgLatency)
	
	if result.MaxLatency > 0 {
		fmt.Printf("Max Latency: %v\n", result.MaxLatency)
	}
	if result.MinLatency > 0 && result.MinLatency < time.Hour {
		fmt.Printf("Min Latency: %v\n", result.MinLatency)
	}
	if result.GoroutineOverhead > 0 {
		fmt.Printf("Goroutine Overhead: %v\n", result.GoroutineOverhead)
	}
	if result.ContextSwitches > 0 {
		fmt.Printf("Context Switches/Work Steals: %d\n", result.ContextSwitches)
	}
	if result.MemoryAllocations > 0 {
		fmt.Printf("Memory Allocations: %d\n", result.MemoryAllocations)
	}
	if result.GCPauses > 0 {
		fmt.Printf("GC Pauses: %d\n", result.GCPauses)
	}
	if result.SchedulerEfficiency > 0 {
		fmt.Printf("Scheduler Efficiency: %.2f%%\n", result.SchedulerEfficiency)
	}
	if result.Errors > 0 {
		fmt.Printf("Errors: %d\n", result.Errors)
	}
}

// printStressSummary prints a summary of all stress test results
func printStressSummary(results []*StressResult) {
	fmt.Printf("\n\n=== STRESS TEST SUMMARY ===\n")
	fmt.Printf("%-25s %10s %12s %12s %10s\n", "Test Name", "Tasks", "Tasks/Sec", "Avg Latency", "Errors")
	fmt.Printf("%s\n", string(make([]byte, 80)))
	
	for _, result := range results {
		if result != nil {
			fmt.Printf("%-25s %10d %12.2f %12v %10d\n", 
				result.TestName, result.TasksCompleted, result.TasksPerSecond, 
				result.AvgLatency, result.Errors)
		}
	}
	
	fmt.Printf("\nScheduler Configuration:\n")
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("NumGoroutine: %d\n", runtime.NumGoroutine())
}

// GetStressWorkloadStats returns statistics about stress workload performance
func GetStressWorkloadStats() map[string]interface{} {
	return map[string]interface{}{
		"workload_type":    "stress",
		"characteristics": "Extreme load conditions, scheduler limits testing",
		"scheduler_impact": "Maximum stress on all scheduler components",
		"optimization_tips": []string{
			"Monitor scheduler latency under load",
			"Identify bottlenecks in high-contention scenarios",
			"Tune GOMAXPROCS for specific workload patterns",
			"Implement backpressure mechanisms",
			"Use profiling to identify scheduler hotspots",
			"Consider workload-specific optimizations",
		},
		"test_scenarios": GetStressTests(),
	}
}