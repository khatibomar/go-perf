package workloads

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MixedTask represents a task that combines CPU and I/O operations
type MixedTask struct {
	ID              int
	CPUComplexity   int
	IOSize          int
	CPUDuration     time.Duration
	IODuration      time.Duration
	TotalDuration   time.Duration
	StartTime       time.Time
	EndTime         time.Time
	CPUResult       int64
	IOSuccess       bool
	Error           error
}

// WorkloadPattern defines different mixed workload patterns
type WorkloadPattern struct {
	Name        string
	CPUWeight   float64 // 0.0 to 1.0
	IOWeight    float64 // 0.0 to 1.0
	Burstiness  float64 // 0.0 to 1.0 (how bursty the workload is)
	Description string
}

// GetWorkloadPatterns returns predefined workload patterns
func GetWorkloadPatterns() []WorkloadPattern {
	return []WorkloadPattern{
		{
			Name:        "Balanced",
			CPUWeight:   0.5,
			IOWeight:    0.5,
			Burstiness:  0.3,
			Description: "Equal mix of CPU and I/O operations",
		},
		{
			Name:        "CPU Heavy",
			CPUWeight:   0.8,
			IOWeight:    0.2,
			Burstiness:  0.4,
			Description: "Mostly CPU-intensive with occasional I/O",
		},
		{
			Name:        "IO Heavy",
			CPUWeight:   0.2,
			IOWeight:    0.8,
			Burstiness:  0.6,
			Description: "Mostly I/O-intensive with light CPU processing",
		},
		{
			Name:        "Bursty",
			CPUWeight:   0.6,
			IOWeight:    0.4,
			Burstiness:  0.9,
			Description: "High variability in workload intensity",
		},
		{
			Name:        "Web Server",
			CPUWeight:   0.3,
			IOWeight:    0.7,
			Burstiness:  0.7,
			Description: "Simulates web server workload pattern",
		},
		{
			Name:        "Data Processing",
			CPUWeight:   0.7,
			IOWeight:    0.3,
			Burstiness:  0.2,
			Description: "Simulates data processing pipeline",
		},
	}
}

// RunMixedWorkload executes mixed CPU/I/O workload across multiple goroutines
func RunMixedWorkload(ctx context.Context, numGoroutines int, pattern WorkloadPattern, verbose bool) error {
	if verbose {
		fmt.Printf("Starting mixed workload '%s' with %d goroutines\n", pattern.Name, numGoroutines)
		fmt.Printf("Pattern: CPU=%.1f%%, I/O=%.1f%%, Burstiness=%.1f%%\n", 
			pattern.CPUWeight*100, pattern.IOWeight*100, pattern.Burstiness*100)
	}
	
	var wg sync.WaitGroup
	var completedTasks int64
	var successfulTasks int64
	var totalCPUTime int64
	var totalIOTime int64
	
	// Create temporary directory for file operations
	tempDir, err := os.MkdirTemp("", "mixed_workload_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Channel for task distribution
	taskChan := make(chan MixedTask, numGoroutines*2)
	
	// Start worker goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			mixedWorker(ctx, workerID, taskChan, pattern, tempDir, 
				&completedTasks, &successfulTasks, &totalCPUTime, &totalIOTime, verbose)
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
			default:
				task := generateMixedTask(taskID, pattern)
				select {
				case taskChan <- task:
					taskID++
					if taskID%200 == 0 && verbose {
						fmt.Printf("Generated %d mixed tasks\n", taskID)
					}
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	// Wait for completion or context cancellation
	wg.Wait()
	
	if verbose {
		completed := atomic.LoadInt64(&completedTasks)
		successful := atomic.LoadInt64(&successfulTasks)
		avgCPUTime := time.Duration(0)
		avgIOTime := time.Duration(0)
		
		if completed > 0 {
			avgCPUTime = time.Duration(atomic.LoadInt64(&totalCPUTime) / completed)
			avgIOTime = time.Duration(atomic.LoadInt64(&totalIOTime) / completed)
		}
		
		fmt.Printf("Mixed workload completed. Tasks: %d, Successful: %d\n", completed, successful)
		fmt.Printf("Average CPU time: %v, Average I/O time: %v\n", avgCPUTime, avgIOTime)
	}
	
	return nil
}

// generateMixedTask creates a task based on the workload pattern
func generateMixedTask(taskID int, pattern WorkloadPattern) MixedTask {
	// Apply burstiness to vary task complexity
	burstFactor := 1.0
	if rand.Float64() < pattern.Burstiness {
		// Create burst: 2x to 5x normal complexity
		burstFactor = 2.0 + rand.Float64()*3.0
	}
	
	// Calculate CPU complexity based on pattern and burst
	baseCPUComplexity := 1000
	cpuComplexity := int(float64(baseCPUComplexity) * pattern.CPUWeight * burstFactor)
	
	// Calculate I/O size based on pattern and burst
	baseIOSize := 1024
	ioSize := int(float64(baseIOSize) * pattern.IOWeight * burstFactor)
	
	// Add some randomness
	cpuComplexity += rand.Intn(cpuComplexity/2)
	ioSize += rand.Intn(ioSize/2)
	
	// Ensure minimum values
	if cpuComplexity < 100 {
		cpuComplexity = 100
	}
	if ioSize < 256 {
		ioSize = 256
	}
	
	return MixedTask{
		ID:            taskID,
		CPUComplexity: cpuComplexity,
		IOSize:        ioSize,
		StartTime:     time.Now(),
	}
}

// mixedWorker performs mixed CPU/I/O operations
func mixedWorker(ctx context.Context, workerID int, taskChan <-chan MixedTask, 
	pattern WorkloadPattern, tempDir string, 
	completedTasks, successfulTasks, totalCPUTime, totalIOTime *int64, verbose bool) {
	
	var localCompleted int64
	var localSuccessful int64
	var localCPUTime int64
	var localIOTime int64
	
	// Create HTTP client for network operations
	client := &http.Client{Timeout: 3 * time.Second}
	
	for {
		select {
		case <-ctx.Done():
			atomic.AddInt64(completedTasks, localCompleted)
			atomic.AddInt64(successfulTasks, localSuccessful)
			atomic.AddInt64(totalCPUTime, localCPUTime)
			atomic.AddInt64(totalIOTime, localIOTime)
			return
			
		case task, ok := <-taskChan:
			if !ok {
				atomic.AddInt64(completedTasks, localCompleted)
				atomic.AddInt64(successfulTasks, localSuccessful)
				atomic.AddInt64(totalCPUTime, localCPUTime)
				atomic.AddInt64(totalIOTime, localIOTime)
				return
			}
			
			// Execute mixed task
			err := executeMixedTask(&task, pattern, tempDir, client, workerID)
			
			task.EndTime = time.Now()
			task.TotalDuration = task.EndTime.Sub(task.StartTime)
			task.Error = err
			task.IOSuccess = err == nil
			
			localCompleted++
			localCPUTime += int64(task.CPUDuration)
			localIOTime += int64(task.IODuration)
			if task.IOSuccess {
				localSuccessful++
			}
			
			// Periodic verbose output
			if verbose && localCompleted%25 == 0 {
				fmt.Printf("Worker %d completed %d mixed tasks (success rate: %.1f%%)\n", 
					workerID, localCompleted, float64(localSuccessful)/float64(localCompleted)*100)
			}
			
			// Occasional yield to allow scheduler decisions
			if localCompleted%10 == 0 {
				runtime.Gosched()
			}
		}
	}
}

// executeMixedTask performs both CPU and I/O operations in a mixed pattern
func executeMixedTask(task *MixedTask, pattern WorkloadPattern, tempDir string, 
	client *http.Client, workerID int) error {
	
	// Determine execution order based on pattern
	cpuFirst := rand.Float64() < 0.5
	
	if cpuFirst {
		// CPU first, then I/O
		if err := executeCPUPortion(task); err != nil {
			return err
		}
		if err := executeIOPortion(task, tempDir, client, workerID); err != nil {
			return err
		}
	} else {
		// I/O first, then CPU
		if err := executeIOPortion(task, tempDir, client, workerID); err != nil {
			return err
		}
		if err := executeCPUPortion(task); err != nil {
			return err
		}
	}
	
	// Sometimes interleave CPU and I/O operations
	if pattern.Burstiness > 0.5 && rand.Float64() < 0.3 {
		return executeInterleavedOperations(task, tempDir, client, workerID)
	}
	
	return nil
}

// executeCPUPortion performs the CPU-intensive part of the task
func executeCPUPortion(task *MixedTask) error {
	start := time.Now()
	
	// Perform CPU-intensive computation
	task.CPUResult = performCPUIntensiveTask(task.CPUComplexity)
	
	task.CPUDuration = time.Since(start)
	return nil
}

// executeIOPortion performs the I/O-intensive part of the task
func executeIOPortion(task *MixedTask, tempDir string, client *http.Client, workerID int) error {
	start := time.Now()
	defer func() {
		task.IODuration = time.Since(start)
	}()
	
	// Choose I/O operation type based on task characteristics
	ioType := rand.Intn(3)
	
	switch ioType {
	case 0: // File I/O
		ioTask := IOBoundTask{
			ID:   task.ID,
			Type: "file",
			Size: task.IOSize,
		}
		return performFileOperation(ioTask, tempDir, workerID)
		
	case 1: // Network I/O (simplified)
		// Simulate network delay
		delay := time.Duration(rand.Intn(50)) * time.Millisecond
		time.Sleep(delay)
		return nil
		
	case 2: // Database I/O
		ioTask := IOBoundTask{
			ID:   task.ID,
			Type: "database",
			Size: task.IOSize / 4, // Database operations typically smaller
		}
		return performDatabaseOperation(ioTask)
	}
	
	return nil
}

// executeInterleavedOperations performs CPU and I/O operations in an interleaved manner
func executeInterleavedOperations(task *MixedTask, tempDir string, client *http.Client, workerID int) error {
	// Split CPU work into smaller chunks
	chunkSize := task.CPUComplexity / 4
	if chunkSize < 100 {
		chunkSize = 100
	}
	
	cpuStart := time.Now()
	var totalCPUResult int64
	
	// Perform interleaved operations
	for i := 0; i < 4; i++ {
		// CPU chunk
		result := performCPUIntensiveTask(chunkSize)
		totalCPUResult += result
		
		// I/O operation (if not last iteration)
		if i < 3 {
			ioStart := time.Now()
			
			// Quick I/O operation
			if rand.Float64() < 0.5 {
				// File operation
				smallIOTask := IOBoundTask{
					ID:   task.ID*10 + i,
					Type: "file",
					Size: task.IOSize / 4,
				}
				performFileOperation(smallIOTask, tempDir, workerID)
			} else {
				// Database operation
				smallIOTask := IOBoundTask{
					ID:   task.ID*10 + i,
					Type: "database",
					Size: task.IOSize / 8,
				}
				performDatabaseOperation(smallIOTask)
			}
			
			task.IODuration += time.Since(ioStart)
		}
		
		// Yield to scheduler between chunks
		runtime.Gosched()
	}
	
	task.CPUResult = totalCPUResult
	task.CPUDuration = time.Since(cpuStart) - task.IODuration
	
	return nil
}

// MixedWorkloadBenchmark provides benchmarking for mixed workloads
type MixedWorkloadBenchmark struct {
	Pattern        WorkloadPattern
	Goroutines     int
	Duration       time.Duration
	TotalTasks     int64
	SuccessfulTasks int64
	SuccessRate    float64
	AvgCPUTime     time.Duration
	AvgIOTime      time.Duration
	AvgTotalTime   time.Duration
	Throughput     float64
	CPUUtilization float64
	IOWaitTime     float64
}

// BenchmarkMixedWorkloads benchmarks different mixed workload patterns
func BenchmarkMixedWorkloads(ctx context.Context, numGoroutines int, duration time.Duration) []MixedWorkloadBenchmark {
	patterns := GetWorkloadPatterns()
	benchmarks := make([]MixedWorkloadBenchmark, len(patterns))
	
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "mixed_benchmark_*")
	if err != nil {
		fmt.Printf("Warning: Could not create temp directory: %v\n", err)
		tempDir = os.TempDir()
	}
	defer os.RemoveAll(tempDir)
	
	for i, pattern := range patterns {
		benchmarks[i].Pattern = pattern
		benchmarks[i].Goroutines = numGoroutines
		benchmarks[i].Duration = duration
		
		fmt.Printf("Benchmarking pattern: %s\n", pattern.Name)
		
		// Run benchmark
		start := time.Now()
		var totalTasks int64
		var successfulTasks int64
		var totalCPUTime int64
		var totalIOTime int64
		var totalTaskTime int64
		
		benchCtx, cancel := context.WithTimeout(ctx, duration)
		var wg sync.WaitGroup
		
		// Task channel
		taskChan := make(chan MixedTask, numGoroutines*2)
		
		// Start workers
		for j := 0; j < numGoroutines; j++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				client := &http.Client{Timeout: 3 * time.Second}
				localTasks := int64(0)
				localSuccessful := int64(0)
				localCPUTime := int64(0)
				localIOTime := int64(0)
				localTotalTime := int64(0)
				
				for {
					select {
					case <-benchCtx.Done():
						atomic.AddInt64(&totalTasks, localTasks)
						atomic.AddInt64(&successfulTasks, localSuccessful)
						atomic.AddInt64(&totalCPUTime, localCPUTime)
						atomic.AddInt64(&totalIOTime, localIOTime)
						atomic.AddInt64(&totalTaskTime, localTotalTime)
						return
					case task, ok := <-taskChan:
						if !ok {
							atomic.AddInt64(&totalTasks, localTasks)
							atomic.AddInt64(&successfulTasks, localSuccessful)
							atomic.AddInt64(&totalCPUTime, localCPUTime)
							atomic.AddInt64(&totalIOTime, localIOTime)
							atomic.AddInt64(&totalTaskTime, localTotalTime)
							return
						}
						
						taskStart := time.Now()
						err := executeMixedTask(&task, pattern, tempDir, client, workerID)
						taskDuration := time.Since(taskStart)
						
						localTasks++
						localCPUTime += int64(task.CPUDuration)
						localIOTime += int64(task.IODuration)
						localTotalTime += int64(taskDuration)
						
						if err == nil {
							localSuccessful++
						}
					}
				}
			}(j)
		}
		
		// Generate tasks
		go func() {
			defer close(taskChan)
			taskID := 0
			
			for {
				select {
				case <-benchCtx.Done():
					return
				default:
					task := generateMixedTask(taskID, pattern)
					select {
					case taskChan <- task:
						taskID++
					case <-benchCtx.Done():
						return
					}
				}
			}
		}()
		
		wg.Wait()
		cancel()
		
		elapsed := time.Since(start)
		
		// Calculate metrics
		benchmarks[i].TotalTasks = totalTasks
		benchmarks[i].SuccessfulTasks = successfulTasks
		if totalTasks > 0 {
			benchmarks[i].SuccessRate = float64(successfulTasks) / float64(totalTasks) * 100
			benchmarks[i].AvgCPUTime = time.Duration(totalCPUTime / totalTasks)
			benchmarks[i].AvgIOTime = time.Duration(totalIOTime / totalTasks)
			benchmarks[i].AvgTotalTime = time.Duration(totalTaskTime / totalTasks)
			
			// Calculate utilization metrics
			totalTime := float64(totalTaskTime)
			cpuTime := float64(totalCPUTime)
			ioTime := float64(totalIOTime)
			
			if totalTime > 0 {
				benchmarks[i].CPUUtilization = cpuTime / totalTime * 100
				benchmarks[i].IOWaitTime = ioTime / totalTime * 100
			}
		}
		benchmarks[i].Throughput = float64(totalTasks) / elapsed.Seconds()
	}
	
	return benchmarks
}

// GetMixedWorkloadStats returns statistics about mixed workload performance
func GetMixedWorkloadStats() map[string]interface{} {
	return map[string]interface{}{
		"workload_type":    "mixed",
		"characteristics": "Variable CPU and I/O utilization, realistic application behavior",
		"scheduler_impact": "Complex scheduling decisions, frequent state transitions",
		"optimization_tips": []string{
			"Balance GOMAXPROCS based on workload characteristics",
			"Monitor both CPU utilization and I/O wait times",
			"Use profiling to identify bottlenecks",
			"Consider workload-specific goroutine pool sizes",
			"Implement adaptive scheduling strategies",
			"Use context for coordinated cancellation",
		},
		"patterns": GetWorkloadPatterns(),
	}
}