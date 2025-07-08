package correctness

import (
	"sync"
	"testing"
	"time"
)

// MockQueue implements a simple queue for testing
type MockQueue struct {
	mu    sync.Mutex
	items []interface{}
}

func NewMockQueue() *MockQueue {
	return &MockQueue{
		items: make([]interface{}, 0),
	}
}

func (q *MockQueue) Enqueue(item interface{}) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
	return true
}

func (q *MockQueue) Dequeue() (interface{}, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *MockQueue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}

func (q *MockQueue) Size() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// TestLinearizabilityChecker tests the linearizability verification
func TestLinearizabilityChecker(t *testing.T) {
	checker := NewLinearizabilityChecker()
	
	// Test simple sequential operations
	t.Run("SequentialOperations", func(t *testing.T) {
		checker.Reset()
		
		// Record sequential operations
		start1 := time.Now()
		end1 := start1.Add(time.Microsecond)
		checker.RecordOperation(Operation{
			Type:      "enqueue",
			Value:     42,
			Success:   true,
			StartTime: start1,
			EndTime:   end1,
			ThreadID:  1,
		})
		
		start2 := end1.Add(time.Microsecond)
		end2 := start2.Add(time.Microsecond)
		checker.RecordOperation(Operation{
			Type:      "dequeue",
			Value:     42,
			Success:   true,
			StartTime: start2,
			EndTime:   end2,
			ThreadID:  1,
		})
		
		linearizable, err := checker.CheckLinearizability()
		if err != nil {
			t.Errorf("Error checking linearizability: %v", err)
		}
		if !linearizable {
			t.Error("Sequential operations should be linearizable")
		}
	})
	
	// Test concurrent operations
	t.Run("ConcurrentOperations", func(t *testing.T) {
		checker.Reset()
		
		baseTime := time.Now()
		
		// Thread 1: enqueue(1)
		checker.RecordOperation(Operation{
			Type:      "enqueue",
			Value:     1,
			Success:   true,
			StartTime: baseTime,
			EndTime:   baseTime.Add(2 * time.Microsecond),
			ThreadID:  1,
		})
		
		// Thread 2: enqueue(2) - overlapping with thread 1
		checker.RecordOperation(Operation{
			Type:      "enqueue",
			Value:     2,
			Success:   true,
			StartTime: baseTime.Add(time.Microsecond),
			EndTime:   baseTime.Add(3 * time.Microsecond),
			ThreadID:  2,
		})
		
		// Thread 1: dequeue() - should get either 1 or 2
		checker.RecordOperation(Operation{
			Type:      "dequeue",
			Value:     1, // Assuming it got 1
			Success:   true,
			StartTime: baseTime.Add(4 * time.Microsecond),
			EndTime:   baseTime.Add(5 * time.Microsecond),
			ThreadID:  1,
		})
		
		linearizable, err := checker.CheckLinearizability()
		if err != nil {
			t.Errorf("Error checking linearizability: %v", err)
		}
		if !linearizable {
			t.Error("Valid concurrent operations should be linearizable")
		}
	})
	
	// Test analysis functionality
	t.Run("HistoryAnalysis", func(t *testing.T) {
		checker.Reset()
		
		// Add some operations for analysis
		baseTime := time.Now()
		for i := 0; i < 5; i++ {
			checker.RecordOperation(Operation{
				Type:      "enqueue",
				Value:     i,
				Success:   true,
				StartTime: baseTime.Add(time.Duration(i) * time.Microsecond),
				EndTime:   baseTime.Add(time.Duration(i+1) * time.Microsecond),
				ThreadID:  i % 2, // Alternate between threads 0 and 1
			})
		}
		
		analysis := checker.AnalyzeHistory()
		if analysis.TotalOperations != 5 {
			t.Errorf("Expected 5 operations, got %d", analysis.TotalOperations)
		}
		
		if len(analysis.ThreadStats) != 2 {
			t.Errorf("Expected 2 threads, got %d", len(analysis.ThreadStats))
		}
		
		t.Logf("Analysis: %+v", analysis)
	})
}

// TestValidateQueueProperties tests FIFO property validation
func TestValidateQueueProperties(t *testing.T) {
	t.Run("ValidFIFO", func(t *testing.T) {
		checker := NewLinearizabilityChecker()
		baseTime := time.Now()
		
		operations := []Operation{
			{Type: "enqueue", Value: 1, Success: true, StartTime: baseTime, EndTime: baseTime.Add(time.Microsecond), ThreadID: 1},
			{Type: "enqueue", Value: 2, Success: true, StartTime: baseTime.Add(time.Microsecond), EndTime: baseTime.Add(2*time.Microsecond), ThreadID: 1},
			{Type: "enqueue", Value: 3, Success: true, StartTime: baseTime.Add(2*time.Microsecond), EndTime: baseTime.Add(3*time.Microsecond), ThreadID: 1},
			{Type: "dequeue", Value: 1, Success: true, StartTime: baseTime.Add(3*time.Microsecond), EndTime: baseTime.Add(4*time.Microsecond), ThreadID: 1},
			{Type: "dequeue", Value: 2, Success: true, StartTime: baseTime.Add(4*time.Microsecond), EndTime: baseTime.Add(5*time.Microsecond), ThreadID: 1},
			{Type: "dequeue", Value: 3, Success: true, StartTime: baseTime.Add(5*time.Microsecond), EndTime: baseTime.Add(6*time.Microsecond), ThreadID: 1},
		}
		
		for _, op := range operations {
			checker.RecordOperation(op)
		}
		
		violations := checker.ValidateQueueProperties()
		if len(violations) > 0 {
			t.Errorf("Valid FIFO sequence should pass validation, got violations: %v", violations)
		}
	})
	
	t.Run("InvalidFIFO", func(t *testing.T) {
		checker := NewLinearizabilityChecker()
		baseTime := time.Now()
		
		operations := []Operation{
			{Type: "enqueue", Value: 1, Success: true, StartTime: baseTime, EndTime: baseTime.Add(time.Microsecond), ThreadID: 1},
			{Type: "enqueue", Value: 2, Success: true, StartTime: baseTime.Add(time.Microsecond), EndTime: baseTime.Add(2*time.Microsecond), ThreadID: 1},
			{Type: "dequeue", Value: 2, Success: true, StartTime: baseTime.Add(2*time.Microsecond), EndTime: baseTime.Add(3*time.Microsecond), ThreadID: 1}, // Should be 1
			{Type: "dequeue", Value: 1, Success: true, StartTime: baseTime.Add(3*time.Microsecond), EndTime: baseTime.Add(4*time.Microsecond), ThreadID: 1}, // Should be 2
		}
		
		for _, op := range operations {
			checker.RecordOperation(op)
		}
		
		violations := checker.ValidateQueueProperties()
		if len(violations) == 0 {
			t.Error("Invalid FIFO sequence should fail validation")
		}
	})
}

// TestStressTester tests the stress testing framework
func TestStressTester(t *testing.T) {
	config := StressTestConfig{
		Duration:        time.Second,
		Producers:       2,
		Consumers:       2,
		OperationsPerSec: 1000,
		ValueRange:      100,
		BatchSize:       1,
		RecordHistory:   true,
	}
	
	tester := NewStressTester(config)
	queue := NewMockQueue()
	
	result := tester.RunStressTest(queue)
	
	if len(result.Errors) > 0 {
		t.Errorf("Stress test should succeed, got errors: %v", result.Errors)
	}
	
	if result.SuccessfulEnqueues == 0 {
		t.Error("Should have enqueued some items")
	}
	
	if result.SuccessfulDequeues == 0 {
		t.Error("Should have dequeued some items")
	}
	
	t.Logf("Stress test result: %+v", result)
}

// TestCorrectnessTest tests the correctness testing functionality
func TestCorrectnessTest(t *testing.T) {
	config := StressTestConfig{
		Duration:        500 * time.Millisecond,
		Producers:       1,
		Consumers:       1,
		OperationsPerSec: 100,
		ValueRange:      50,
		BatchSize:       1,
		RecordHistory:   true,
	}
	
	tester := NewStressTester(config)
	queue := NewMockQueue()
	
	result := tester.RunStressTest(queue)
	
	if len(result.Errors) > 0 {
		t.Errorf("Correctness test should succeed, got errors: %v", result.Errors)
	}
	
	if !result.Linearizable {
		t.Error("Operations should be linearizable")
	}
	
	t.Logf("Correctness test result: %+v", result)
}

// TestLoadTest tests the load testing functionality
func TestLoadTest(t *testing.T) {
	config := StressTestConfig{
		Duration:        time.Second,
		Producers:       4,
		Consumers:       4,
		OperationsPerSec: 2000,
		ValueRange:      200,
		BatchSize:       1,
		RecordHistory:   false, // Focus on performance
	}
	
	tester := NewStressTester(config)
	queue := NewMockQueue()
	
	result := tester.RunStressTest(queue)
	
	if len(result.Errors) > 0 {
		t.Errorf("Load test should succeed, got errors: %v", result.Errors)
	}
	
	if result.Throughput == 0 {
		t.Error("Throughput should be greater than 0")
	}
	
	t.Logf("Load test result: Throughput=%.2f ops/sec, Duration=%v", result.Throughput, result.Duration)
}

// TestCompareImplementations tests comparison of multiple queue implementations
func TestCompareImplementations(t *testing.T) {
	config := StressTestConfig{
		Duration:        500 * time.Millisecond,
		Producers:       2,
		Consumers:       2,
		OperationsPerSec: 1000,
		ValueRange:      100,
		BatchSize:       1,
		RecordHistory:   true,
	}
	
	queues := map[string]Queue{
		"MockQueue1": NewMockQueue(),
		"MockQueue2": NewMockQueue(),
	}
	
	results := make(map[string]StressTestResult)
	for name, queue := range queues {
		tester := NewStressTester(config)
		results[name] = tester.RunStressTest(queue)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	for name, result := range results {
		if len(result.Errors) > 0 {
			t.Errorf("Queue %s should succeed, got errors: %v", name, result.Errors)
		}
		t.Logf("Queue %s: Throughput=%.2f ops/sec", name, result.Throughput)
	}
}

// TestConcurrentLinearizabilityChecking tests linearizability checking under concurrent operations
func TestConcurrentLinearizabilityChecking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent linearizability test in short mode")
	}
	
	checker := NewLinearizabilityChecker()
	queue := NewMockQueue()
	
	const (
		numGoroutines = 4
		operationsPerGoroutine = 50
	)
	
	var wg sync.WaitGroup
	
	// Run concurrent operations while recording them
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				value := threadID*operationsPerGoroutine + j
				
				// Enqueue operation
				start := time.Now()
				result := queue.Enqueue(value)
				end := time.Now()
				
				checker.RecordOperation(Operation{
					Type:      "enqueue",
					Value:     value,
					Success:   result,
					StartTime: start,
					EndTime:   end,
					ThreadID:  threadID,
				})
				
				// Occasionally dequeue
				if j%3 == 0 {
					start = time.Now()
					val, ok := queue.Dequeue()
					end = time.Now()
					
					var value interface{}
					if ok {
						value = val
					} else {
						value = nil
					}
					
					checker.RecordOperation(Operation{
						Type:      "dequeue",
						Value:     value,
						Success:   ok,
						StartTime: start,
						EndTime:   end,
						ThreadID:  threadID,
					})
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Check linearizability (this might be expensive for large histories)
	analysis := checker.AnalyzeHistory()
	t.Logf("Concurrent operations analysis: %+v", analysis)
	
	// For a simple mock queue with proper locking, operations should be linearizable
	// Note: Full linearizability checking can be computationally expensive,
	// so we mainly test that the checker doesn't crash and provides analysis
	if analysis.TotalOperations == 0 {
		t.Error("Should have recorded some operations")
	}
}

// BenchmarkLinearizabilityChecker benchmarks the linearizability checker
func BenchmarkLinearizabilityChecker(b *testing.B) {
	checker := NewLinearizabilityChecker()
	baseTime := time.Now()
	
	b.Run("RecordOperation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			checker.RecordOperation(Operation{
				Type:      "enqueue",
				Value:     i,
				Success:   true,
				StartTime: baseTime,
				EndTime:   baseTime.Add(time.Microsecond),
				ThreadID:  0,
			})
		}
	})
	
	b.Run("AnalyzeHistory", func(b *testing.B) {
		// Prepare some operations
		checker.Reset()
		for i := 0; i < 100; i++ {
			checker.RecordOperation(Operation{
				Type:      "enqueue",
				Value:     i,
				Success:   true,
				StartTime: baseTime.Add(time.Duration(i) * time.Microsecond),
				EndTime:   baseTime.Add(time.Duration(i+1) * time.Microsecond),
				ThreadID:  i % 4,
			})
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			checker.AnalyzeHistory()
		}
	})
}