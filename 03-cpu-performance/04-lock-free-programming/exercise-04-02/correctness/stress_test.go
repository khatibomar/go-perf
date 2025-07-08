package correctness

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Queue interface that all queue implementations must satisfy
type Queue interface {
	Enqueue(interface{}) bool
	Dequeue() (interface{}, bool)
	IsEmpty() bool
	Size() interface{} // Can return int, uint64, etc.
}

// StressTestConfig contains configuration for stress testing
type StressTestConfig struct {
	Duration        time.Duration
	Producers       int
	Consumers       int
	OperationsPerSec int
	ValueRange      int
	BatchSize       int
	RecordHistory   bool
}

// DefaultStressTestConfig returns a default stress test configuration
func DefaultStressTestConfig() StressTestConfig {
	return StressTestConfig{
		Duration:        10 * time.Second,
		Producers:       4,
		Consumers:       4,
		OperationsPerSec: 10000,
		ValueRange:      1000000,
		BatchSize:       1,
		RecordHistory:   false,
	}
}

// StressTestResult contains the results of a stress test
type StressTestResult struct {
	Duration           time.Duration
	TotalOperations    int64
	SuccessfulEnqueues int64
	SuccessfulDequeues int64
	FailedEnqueues     int64
	FailedDequeues     int64
	Throughput         float64
	Errors             []string
	Linearizable       bool
	History            *LinearizabilityChecker
}

// StressTester performs stress testing on queue implementations
type StressTester struct {
	config  StressTestConfig
	checker *LinearizabilityChecker
}

// NewStressTester creates a new stress tester
func NewStressTester(config StressTestConfig) *StressTester {
	var checker *LinearizabilityChecker
	if config.RecordHistory {
		checker = NewLinearizabilityChecker()
	}
	
	return &StressTester{
		config:  config,
		checker: checker,
	}
}

// RunStressTest runs a comprehensive stress test on the queue
func (st *StressTester) RunStressTest(queue Queue) StressTestResult {
	ctx, cancel := context.WithTimeout(context.Background(), st.config.Duration)
	defer cancel()
	
	var (
		totalOps           int64
		successfulEnqueues int64
		successfulDequeues int64
		failedEnqueues     int64
		failedDequeues     int64
		errors             []string
		errorsMu           sync.Mutex
	)
	
	var wg sync.WaitGroup
	startTime := time.Now()
	
	// Start producer goroutines
	for i := 0; i < st.config.Producers; i++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			st.runProducer(ctx, queue, producerID, &totalOps, &successfulEnqueues, &failedEnqueues, &errors, &errorsMu)
		}(i)
	}
	
	// Start consumer goroutines
	for i := 0; i < st.config.Consumers; i++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()
			st.runConsumer(ctx, queue, consumerID+st.config.Producers, &totalOps, &successfulDequeues, &failedDequeues, &errors, &errorsMu)
		}(i)
	}
	
	wg.Wait()
	actualDuration := time.Since(startTime)
	
	// Check linearizability if history was recorded
	linearizable := true
	if st.checker != nil {
		var err error
		linearizable, err = st.checker.CheckLinearizability()
		if err != nil {
			errorsMu.Lock()
			errors = append(errors, fmt.Sprintf("Linearizability check error: %v", err))
			errorsMu.Unlock()
		}
	}
	
	throughput := float64(atomic.LoadInt64(&totalOps)) / actualDuration.Seconds()
	
	return StressTestResult{
		Duration:           actualDuration,
		TotalOperations:    atomic.LoadInt64(&totalOps),
		SuccessfulEnqueues: atomic.LoadInt64(&successfulEnqueues),
		SuccessfulDequeues: atomic.LoadInt64(&successfulDequeues),
		FailedEnqueues:     atomic.LoadInt64(&failedEnqueues),
		FailedDequeues:     atomic.LoadInt64(&failedDequeues),
		Throughput:         throughput,
		Errors:             errors,
		Linearizable:       linearizable,
		History:            st.checker,
	}
}

// runProducer runs a producer goroutine
func (st *StressTester) runProducer(ctx context.Context, queue Queue, producerID int, totalOps, successfulEnqueues, failedEnqueues *int64, errors *[]string, errorsMu *sync.Mutex) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(producerID)))
	ticker := time.NewTicker(time.Second / time.Duration(st.config.OperationsPerSec/st.config.Producers))
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < st.config.BatchSize; i++ {
				value := rng.Intn(st.config.ValueRange)
				
				startTime := time.Now()
				success := queue.Enqueue(value)
				endTime := time.Now()
				
				atomic.AddInt64(totalOps, 1)
				
				if success {
					atomic.AddInt64(successfulEnqueues, 1)
				} else {
					atomic.AddInt64(failedEnqueues, 1)
				}
				
				// Record operation if history tracking is enabled
				if st.checker != nil {
					st.checker.RecordOperation(Operation{
						Type:      "enqueue",
						Value:     value,
						Success:   success,
						StartTime: startTime,
						EndTime:   endTime,
						ThreadID:  producerID,
					})
				}
			}
		}
	}
}

// runConsumer runs a consumer goroutine
func (st *StressTester) runConsumer(ctx context.Context, queue Queue, consumerID int, totalOps, successfulDequeues, failedDequeues *int64, errors *[]string, errorsMu *sync.Mutex) {
	ticker := time.NewTicker(time.Second / time.Duration(st.config.OperationsPerSec/st.config.Consumers))
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < st.config.BatchSize; i++ {
				startTime := time.Now()
				value, success := queue.Dequeue()
				endTime := time.Now()
				
				atomic.AddInt64(totalOps, 1)
				
				if success {
					atomic.AddInt64(successfulDequeues, 1)
				} else {
					atomic.AddInt64(failedDequeues, 1)
				}
				
				// Record operation if history tracking is enabled
				if st.checker != nil {
					st.checker.RecordOperation(Operation{
						Type:      "dequeue",
						Value:     value,
						Success:   success,
						StartTime: startTime,
						EndTime:   endTime,
						ThreadID:  consumerID,
					})
				}
			}
		}
	}
}

// RunCorrectnessTest runs a focused correctness test
func (st *StressTester) RunCorrectnessTest(queue Queue) StressTestResult {
	// Use a smaller, more controlled test for correctness
	originalConfig := st.config
	st.config.Duration = 5 * time.Second
	st.config.Producers = 2
	st.config.Consumers = 2
	st.config.OperationsPerSec = 1000
	st.config.RecordHistory = true
	
	if st.checker == nil {
		st.checker = NewLinearizabilityChecker()
	}
	
	result := st.RunStressTest(queue)
	
	// Restore original config
	st.config = originalConfig
	
	return result
}

// RunLoadTest runs a high-load performance test
func (st *StressTester) RunLoadTest(queue Queue) StressTestResult {
	originalConfig := st.config
	st.config.Duration = 30 * time.Second
	st.config.Producers = runtime.NumCPU()
	st.config.Consumers = runtime.NumCPU()
	st.config.OperationsPerSec = 100000
	st.config.RecordHistory = false // Disable for performance
	
	result := st.RunStressTest(queue)
	
	// Restore original config
	st.config = originalConfig
	
	return result
}

// String returns a string representation of the stress test result
func (str StressTestResult) String() string {
	successRate := float64(str.SuccessfulEnqueues+str.SuccessfulDequeues) / float64(str.TotalOperations) * 100
	
	result := fmt.Sprintf(
		"Stress Test Results:\n"+
		"  Duration: %v\n"+
		"  Total Operations: %d\n"+
		"  Successful Enqueues: %d\n"+
		"  Successful Dequeues: %d\n"+
		"  Failed Enqueues: %d\n"+
		"  Failed Dequeues: %d\n"+
		"  Success Rate: %.2f%%\n"+
		"  Throughput: %.2f ops/sec\n"+
		"  Linearizable: %t\n",
		str.Duration,
		str.TotalOperations,
		str.SuccessfulEnqueues,
		str.SuccessfulDequeues,
		str.FailedEnqueues,
		str.FailedDequeues,
		successRate,
		str.Throughput,
		str.Linearizable,
	)
	
	if len(str.Errors) > 0 {
		result += fmt.Sprintf("  Errors: %d\n", len(str.Errors))
		for i, err := range str.Errors {
			if i < 5 { // Show first 5 errors
				result += fmt.Sprintf("    %s\n", err)
			} else {
				result += fmt.Sprintf("    ... and %d more\n", len(str.Errors)-5)
				break
			}
		}
	}
	
	return result
}

// CompareQueues compares multiple queue implementations
func CompareQueues(queues map[string]Queue, config StressTestConfig) map[string]StressTestResult {
	results := make(map[string]StressTestResult)
	
	for name, queue := range queues {
		fmt.Printf("Testing %s...\n", name)
		tester := NewStressTester(config)
		results[name] = tester.RunStressTest(queue)
		fmt.Printf("Completed %s\n", name)
	}
	
	return results
}