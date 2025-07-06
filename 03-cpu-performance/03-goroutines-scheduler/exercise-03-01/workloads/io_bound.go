package workloads

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// IOBoundTask represents an I/O intensive task
type IOBoundTask struct {
	ID          int
	Type        string // "file", "network", "database"
	Size        int
	Latency     time.Duration
	StartTime   time.Time
	EndTime     time.Time
	Success     bool
	Error       error
}

// RunIOBoundWorkload executes I/O-intensive tasks across multiple goroutines
func RunIOBoundWorkload(ctx context.Context, numGoroutines int, verbose bool) error {
	if verbose {
		fmt.Printf("Starting I/O-bound workload with %d goroutines\n", numGoroutines)
	}
	
	var wg sync.WaitGroup
	var completedTasks int64
	var successfulTasks int64
	var totalLatency int64
	
	// Create temporary directory for file operations
	tempDir, err := os.MkdirTemp("", "io_workload_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Channel for task distribution
	taskChan := make(chan IOBoundTask, numGoroutines*2)
	
	// Start worker goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ioWorker(ctx, workerID, taskChan, tempDir, &completedTasks, &successfulTasks, &totalLatency, verbose)
		}(i)
	}
	
	// Generate tasks
	go func() {
		defer close(taskChan)
		taskID := 0
		taskTypes := []string{"file", "network", "database"}
		
		for {
			select {
			case <-ctx.Done():
				return
			case taskChan <- IOBoundTask{
				ID:        taskID,
				Type:      taskTypes[taskID%len(taskTypes)],
				Size:      1024 + rand.Intn(8192), // 1KB to 9KB
				StartTime: time.Now(),
			}:
				taskID++
				if taskID%500 == 0 && verbose {
					fmt.Printf("Generated %d I/O tasks\n", taskID)
				}
			}
		}
	}()
	
	// Wait for completion or context cancellation
	wg.Wait()
	
	if verbose {
		completed := atomic.LoadInt64(&completedTasks)
		successful := atomic.LoadInt64(&successfulTasks)
		avgLatency := time.Duration(0)
		if completed > 0 {
			avgLatency = time.Duration(atomic.LoadInt64(&totalLatency) / completed)
		}
		
		fmt.Printf("I/O workload completed. Tasks: %d, Successful: %d, Avg Latency: %v\n", 
			completed, successful, avgLatency)
	}
	
	return nil
}

// ioWorker performs I/O-intensive operations
func ioWorker(ctx context.Context, workerID int, taskChan <-chan IOBoundTask, 
	tempDir string, completedTasks, successfulTasks, totalLatency *int64, verbose bool) {
	
	var localCompleted int64
	var localSuccessful int64
	var localLatency int64
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	for {
		select {
		case <-ctx.Done():
			atomic.AddInt64(completedTasks, localCompleted)
			atomic.AddInt64(successfulTasks, localSuccessful)
			atomic.AddInt64(totalLatency, localLatency)
			return
			
		case task, ok := <-taskChan:
			if !ok {
				atomic.AddInt64(completedTasks, localCompleted)
				atomic.AddInt64(successfulTasks, localSuccessful)
				atomic.AddInt64(totalLatency, localLatency)
				return
			}
			
			// Perform I/O operation based on task type
			var err error
			switch task.Type {
			case "file":
				err = performFileOperation(task, tempDir, workerID)
			case "network":
				err = performNetworkOperation(task, client)
			case "database":
				err = performDatabaseOperation(task)
			}
			
			task.EndTime = time.Now()
			task.Latency = task.EndTime.Sub(task.StartTime)
			task.Error = err
			task.Success = err == nil
			
			localCompleted++
			localLatency += int64(task.Latency)
			if task.Success {
				localSuccessful++
			}
			
			// Periodic verbose output
			if verbose && localCompleted%50 == 0 {
				fmt.Printf("Worker %d completed %d I/O tasks (success rate: %.1f%%)\n", 
					workerID, localCompleted, float64(localSuccessful)/float64(localCompleted)*100)
			}
		}
	}
}

// performFileOperation simulates file I/O operations
func performFileOperation(task IOBoundTask, tempDir string, workerID int) error {
	filename := filepath.Join(tempDir, fmt.Sprintf("worker_%d_task_%d.tmp", workerID, task.ID))
	
	// Generate random data
	data := make([]byte, task.Size)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	
	// Write file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write file error: %w", err)
	}
	
	// Read file back
	readData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file error: %w", err)
	}
	
	// Verify data integrity
	if len(readData) != len(data) {
		return fmt.Errorf("data size mismatch: expected %d, got %d", len(data), len(readData))
	}
	
	// Simulate additional file operations
	if rand.Float32() < 0.3 { // 30% chance of additional operations
		// Append operation
		additionalData := make([]byte, task.Size/4)
		for i := range additionalData {
			additionalData[i] = byte(rand.Intn(256))
		}
		
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open file for append error: %w", err)
		}
		
		_, err = file.Write(additionalData)
		file.Close()
		if err != nil {
			return fmt.Errorf("append file error: %w", err)
		}
	}
	
	// Clean up
	os.Remove(filename)
	
	return nil
}

// performNetworkOperation simulates network I/O operations
func performNetworkOperation(task IOBoundTask, client *http.Client) error {
	// List of test endpoints (using httpbin.org for testing)
	endpoints := []string{
		"https://httpbin.org/delay/1",
		"https://httpbin.org/bytes/1024",
		"https://httpbin.org/json",
		"https://httpbin.org/uuid",
		"https://httpbin.org/base64/SFRUUEJJTiBpcyBhd2Vzb21l",
	}
	
	// Select random endpoint
	endpoint := endpoints[rand.Intn(len(endpoints))]
	
	// Make HTTP request
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response error: %w", err)
	}
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	// Simulate processing response
	if len(body) == 0 {
		return fmt.Errorf("empty response body")
	}
	
	return nil
}

// performDatabaseOperation simulates database I/O operations
func performDatabaseOperation(task IOBoundTask) error {
	// Simulate database connection latency
	connectionDelay := time.Duration(rand.Intn(50)) * time.Millisecond
	time.Sleep(connectionDelay)
	
	// Simulate different types of database operations
	operationType := rand.Intn(4)
	
	switch operationType {
	case 0: // SELECT operation
		return simulateSelectOperation(task.Size)
	case 1: // INSERT operation
		return simulateInsertOperation(task.Size)
	case 2: // UPDATE operation
		return simulateUpdateOperation(task.Size)
	case 3: // DELETE operation
		return simulateDeleteOperation(task.Size)
	}
	
	return nil
}

// simulateSelectOperation simulates a database SELECT query
func simulateSelectOperation(recordCount int) error {
	// Simulate query planning time
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	
	// Simulate data retrieval based on record count
	retrievalTime := time.Duration(recordCount/100) * time.Millisecond
	if retrievalTime > 100*time.Millisecond {
		retrievalTime = 100 * time.Millisecond
	}
	time.Sleep(retrievalTime)
	
	// Simulate occasional query failures
	if rand.Float32() < 0.05 { // 5% failure rate
		return fmt.Errorf("database connection timeout")
	}
	
	return nil
}

// simulateInsertOperation simulates a database INSERT operation
func simulateInsertOperation(dataSize int) error {
	// Simulate data validation time
	time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
	
	// Simulate write operation based on data size
	writeTime := time.Duration(dataSize/200) * time.Millisecond
	if writeTime > 50*time.Millisecond {
		writeTime = 50 * time.Millisecond
	}
	time.Sleep(writeTime)
	
	// Simulate transaction commit
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	
	// Simulate occasional write failures
	if rand.Float32() < 0.03 { // 3% failure rate
		return fmt.Errorf("database write conflict")
	}
	
	return nil
}

// simulateUpdateOperation simulates a database UPDATE operation
func simulateUpdateOperation(recordCount int) error {
	// Simulate row locking time
	time.Sleep(time.Duration(rand.Intn(15)) * time.Millisecond)
	
	// Simulate update operation
	updateTime := time.Duration(recordCount/150) * time.Millisecond
	if updateTime > 75*time.Millisecond {
		updateTime = 75 * time.Millisecond
	}
	time.Sleep(updateTime)
	
	// Simulate occasional update failures
	if rand.Float32() < 0.04 { // 4% failure rate
		return fmt.Errorf("database deadlock detected")
	}
	
	return nil
}

// simulateDeleteOperation simulates a database DELETE operation
func simulateDeleteOperation(recordCount int) error {
	// Simulate cascade check time
	time.Sleep(time.Duration(rand.Intn(8)) * time.Millisecond)
	
	// Simulate delete operation
	deleteTime := time.Duration(recordCount/250) * time.Millisecond
	if deleteTime > 40*time.Millisecond {
		deleteTime = 40 * time.Millisecond
	}
	time.Sleep(deleteTime)
	
	// Simulate occasional delete failures
	if rand.Float32() < 0.02 { // 2% failure rate
		return fmt.Errorf("foreign key constraint violation")
	}
	
	return nil
}

// IOBoundBenchmark provides benchmarking for I/O-bound operations
type IOBoundBenchmark struct {
	Name           string
	OperationType  string
	Goroutines     int
	Duration       time.Duration
	Operations     int64
	SuccessRate    float64
	AvgLatency     time.Duration
	Throughput     float64
}

// BenchmarkIOOperations benchmarks different I/O operations
func BenchmarkIOOperations(ctx context.Context, numGoroutines int, duration time.Duration) []IOBoundBenchmark {
	benchmarks := []IOBoundBenchmark{
		{Name: "File Operations", OperationType: "file"},
		{Name: "Network Operations", OperationType: "network"},
		{Name: "Database Operations", OperationType: "database"},
	}
	
	// Create temporary directory for file operations
	tempDir, err := os.MkdirTemp("", "io_benchmark_*")
	if err != nil {
		fmt.Printf("Warning: Could not create temp directory: %v\n", err)
		tempDir = os.TempDir()
	}
	defer os.RemoveAll(tempDir)
	
	client := &http.Client{Timeout: 5 * time.Second}
	
	for i := range benchmarks {
		benchmarks[i].Goroutines = numGoroutines
		benchmarks[i].Duration = duration
		
		// Run benchmark
		start := time.Now()
		var operations int64
		var successful int64
		var totalLatency int64
		
		benchCtx, cancel := context.WithTimeout(ctx, duration)
		var wg sync.WaitGroup
		
		for j := 0; j < numGoroutines; j++ {
			wg.Add(1)
			go func(benchType string, workerID int) {
				defer wg.Done()
				localOps := int64(0)
				localSuccessful := int64(0)
				localLatency := int64(0)
				
				for {
					select {
					case <-benchCtx.Done():
						atomic.AddInt64(&operations, localOps)
						atomic.AddInt64(&successful, localSuccessful)
						atomic.AddInt64(&totalLatency, localLatency)
						return
					default:
						taskStart := time.Now()
						var err error
						
						switch benchType {
						case "file":
							task := IOBoundTask{ID: int(localOps), Size: 1024 + rand.Intn(2048)}
							err = performFileOperation(task, tempDir, workerID)
						case "network":
							task := IOBoundTask{ID: int(localOps), Size: 1024}
							err = performNetworkOperation(task, client)
						case "database":
							task := IOBoundTask{ID: int(localOps), Size: 100 + rand.Intn(500)}
							err = performDatabaseOperation(task)
						}
						
						latency := time.Since(taskStart)
						localOps++
						localLatency += int64(latency)
						if err == nil {
							localSuccessful++
						}
					}
				}
			}(benchmarks[i].OperationType, j)
		}
		
		wg.Wait()
		cancel()
		
		elapsed := time.Since(start)
		benchmarks[i].Operations = operations
		if operations > 0 {
			benchmarks[i].SuccessRate = float64(successful) / float64(operations) * 100
			benchmarks[i].AvgLatency = time.Duration(totalLatency / operations)
		}
		benchmarks[i].Throughput = float64(operations) / elapsed.Seconds()
	}
	
	return benchmarks
}

// GetIOWorkloadStats returns statistics about I/O workload performance
func GetIOWorkloadStats() map[string]interface{} {
	return map[string]interface{}{
		"workload_type":    "io_bound",
		"characteristics": "High I/O wait time, low CPU utilization, blocking operations",
		"scheduler_impact": "Frequent goroutine parking/unparking, context switching",
		"optimization_tips": []string{
			"Use more goroutines than CPU cores for I/O-bound tasks",
			"Implement connection pooling for network/database operations",
			"Use buffered I/O operations when possible",
			"Consider async I/O patterns",
			"Monitor goroutine blocking time",
			"Use context for timeout and cancellation",
		},
	}
}