package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"exercise-04-02/correctness"
	"exercise-04-02/memory"
	"exercise-04-02/monitoring"
	"exercise-04-02/queue"
)

func main() {
	fmt.Println("=== Lock-Free Queue Implementation Demo ===")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("Number of CPUs: %d\n", runtime.NumCPU())
	fmt.Printf("GOMAXPROCS: %d\n\n", runtime.GOMAXPROCS(0))

	// Demo 1: Basic Queue Operations
	demoBasicOperations()

	// Demo 2: Concurrent Operations
	demoConcurrentOperations()

	// Demo 3: Memory Management
	demoMemoryManagement()

	// Demo 4: Performance Benchmarking
	demoBenchmarking()

	// Demo 5: Correctness Testing
	demoCorrectnessVerification()

	// Demo 6: Monitoring and Metrics
	demoMonitoring()

	// Demo 7: Queue Comparison
	demoQueueComparison()

	fmt.Println("\n=== Demo Complete ===")
}

// Demo 1: Basic Queue Operations
func demoBasicOperations() {
	fmt.Println("\n--- Demo 1: Basic Queue Operations ---")

	// Test Michael & Scott Queue
	fmt.Println("\nMichael & Scott Queue:")
	msQueue := queue.NewMichaelScottQueue()
	testBasicQueueOps(msQueue, "Michael & Scott")

	// Test Versioned Queue
	fmt.Println("\nVersioned Queue:")
	vQueue := queue.NewVersionedQueue()
	testBasicQueueOps(vQueue, "Versioned")

	// Test Hazard Pointer Queue
	fmt.Println("\nHazard Pointer Queue:")
	hpQueue := queue.NewHazardPointerQueue()
	testBasicQueueOps(hpQueue, "Hazard Pointer")

	// Test Bounded Queue
	fmt.Println("\nBounded Queue:")
	bQueue := queue.NewBoundedQueue(16)
	testBoundedQueueOps(bQueue)
}

func testBasicQueueOps(q interface{}, name string) {
	type BasicQueue interface {
		Enqueue(interface{}) bool
		Dequeue() (interface{}, bool)
		IsEmpty() bool
		Size() int
	}

	queue, ok := q.(BasicQueue)
	if !ok {
		fmt.Printf("  %s queue does not implement BasicQueue interface\n", name)
		return
	}

	// Test FIFO behavior
	values := []int{1, 2, 3, 4, 5}
	fmt.Printf("  Enqueuing values: %v\n", values)

	for _, v := range values {
		if !queue.Enqueue(v) {
			fmt.Printf("  Failed to enqueue %d\n", v)
		}
	}

	fmt.Printf("  Queue size: %d, Empty: %t\n", queue.Size(), queue.IsEmpty())

	fmt.Print("  Dequeuing values: ")
	for i := 0; i < len(values); i++ {
		if val, ok := queue.Dequeue(); ok {
			fmt.Printf("%v ", val)
		} else {
			fmt.Print("<empty> ")
		}
	}
	fmt.Printf("\n  Final size: %d, Empty: %t\n", queue.Size(), queue.IsEmpty())
}

func testBoundedQueueOps(bq *queue.BoundedQueue) {
	fmt.Printf("  Capacity: %d\n", bq.Capacity())

	// Fill the queue
	for i := 0; i < int(bq.Capacity()); i++ {
		if !bq.Enqueue(i) {
			fmt.Printf("  Failed to enqueue %d\n", i)
			break
		}
	}

	fmt.Printf("  After filling - Size: %d, Full: %t\n", bq.Size(), bq.IsFull())

	// Try to enqueue when full
	if bq.Enqueue(999) {
		fmt.Println("  ERROR: Should not be able to enqueue when full")
	} else {
		fmt.Println("  Correctly rejected enqueue when full")
	}

	// Dequeue half
	halfCap := int(bq.Capacity()) / 2
	for i := 0; i < halfCap; i++ {
		bq.Dequeue()
	}

	fmt.Printf("  After dequeuing %d items - Size: %d\n", halfCap, bq.Size())
}

// Demo 2: Concurrent Operations
func demoConcurrentOperations() {
	fmt.Println("\n--- Demo 2: Concurrent Operations ---")

	queues := map[string]func() interface{}{
		"Michael & Scott": func() interface{} { return queue.NewMichaelScottQueue() },
		"Versioned":       func() interface{} { return queue.NewVersionedQueue() },
		"Hazard Pointer":  func() interface{} { return queue.NewHazardPointerQueue() },
		"Bounded":         func() interface{} { return queue.NewBoundedQueue(1024) },
	}

	for name, createQueue := range queues {
		fmt.Printf("\nTesting %s Queue concurrency:\n", name)
		testConcurrency(createQueue(), name)
	}
}

func testConcurrency(q interface{}, name string) {
	type ConcurrentQueue interface {
		Enqueue(interface{}) bool
		Dequeue() (interface{}, bool)
	}

	queue, ok := q.(ConcurrentQueue)
	if !ok {
		fmt.Printf("  %s queue does not implement ConcurrentQueue interface\n", name)
		return
	}

	const (
		numProducers     = 4
		numConsumers     = 4
		itemsPerProducer = 1000
	)

	var wg sync.WaitGroup
	var totalProduced, totalConsumed int64
	start := time.Now()

	// Start producers
	for p := 0; p < numProducers; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			produced := 0
			for i := 0; i < itemsPerProducer; i++ {
				value := producerID*itemsPerProducer + i
				for !queue.Enqueue(value) {
					runtime.Gosched()
				}
				produced++
			}
			totalProduced += int64(produced)
		}(p)
	}

	// Start consumers
	for c := 0; c < numConsumers; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			consumed := 0
			for consumed < itemsPerProducer {
				if _, ok := queue.Dequeue(); ok {
					consumed++
				} else {
					runtime.Gosched()
				}
			}
			totalConsumed += int64(consumed)
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("  Produced: %d, Consumed: %d\n", totalProduced, totalConsumed)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Throughput: %.2f ops/sec\n", float64(totalProduced+totalConsumed)/duration.Seconds())
}

// Demo 3: Memory Management
func demoMemoryManagement() {
	fmt.Println("\n--- Demo 3: Memory Management ---")

	// Demo Hazard Pointers
	fmt.Println("\nHazard Pointer System:")
	hpm := memory.NewHazardPointerManager(8)

	// Acquire and use hazard pointers
	hp1 := hpm.AcquireHazardPointer(nil)
	hp2 := hpm.AcquireHazardPointer(nil)

	if hp1 != nil && hp2 != nil {
		fmt.Println("  Successfully acquired 2 hazard pointers")

		// Simulate pointer usage
		data1 := &struct{ value int }{42}
		data2 := &struct{ value int }{100}

		hp1.SetPointer(unsafe.Pointer(data1))
		hp2.SetPointer(unsafe.Pointer(data2))

		fmt.Printf("  Set hazard pointers to data structures\n")

		// Retire some pointers
		for i := 0; i < 10; i++ {
			retiredData := &struct{ value int }{i}
			hpm.RetirePointer(unsafe.Pointer(retiredData))
		}

		hpm.ReleaseHazardPointer(hp1)
		hpm.ReleaseHazardPointer(hp2)

		hpm.ForceCleanup()

		activeCount, retiredCount := hpm.Stats()
		fmt.Printf("  Final stats - Active: %d, Retired: %d\n", activeCount, retiredCount)
	}

	// Demo Epoch-based Reclamation
	fmt.Println("\nEpoch-based Reclamation:")
	em := memory.NewEpochManager()

	// Enter critical section and retire objects
	guard := em.EnterCriticalSection()
	for i := 0; i < 5; i++ {
		obj := &struct{ value int }{i}
		em.RetireObject(unsafe.Pointer(obj), func(p unsafe.Pointer) {
			// Cleanup function for retired objects
		})
	}
	em.ExitCriticalSection(guard)

	// Try reclamation
	em.TryAdvanceEpoch()

	epoch, activeThreads, totalRetired := em.Stats()
	fmt.Printf("  Epoch: %d, Active Threads: %d, Total Retired: %d\n", epoch, activeThreads, totalRetired)
}

// Demo 4: Performance Benchmarking
func demoBenchmarking() {
	fmt.Println("\n--- Demo 4: Performance Benchmarking ---")

	queues := map[string]func() interface{}{
		"Michael & Scott": func() interface{} { return queue.NewMichaelScottQueue() },
		"Versioned":       func() interface{} { return queue.NewVersionedQueue() },
		"Bounded":         func() interface{} { return queue.NewBoundedQueue(1024) },
	}

	config := struct {
		Workload       string
		Producers      int
		Consumers      int
		Duration       time.Duration
		WarmupDuration time.Duration
		ValueRange     int
		BatchSize      int
		MeasureLatency bool
	}{
		Workload:       "Balanced",
		Producers:      2,
		Consumers:      2,
		Duration:       2 * time.Second,
		WarmupDuration: 200 * time.Millisecond,
		ValueRange:     10000,
		BatchSize:      1,
		MeasureLatency: true,
	}

	fmt.Printf("\nBenchmark Configuration:\n")
	fmt.Printf("  Workload: %s\n", config.Workload)
	fmt.Printf("  Producers: %d, Consumers: %d\n", config.Producers, config.Consumers)
	fmt.Printf("  Duration: %v\n", config.Duration)

	for name, createQueue := range queues {
		fmt.Printf("\nBenchmarking %s Queue...\n", name)
		q := createQueue()

		// Simple benchmark simulation
		start := time.Now()
		operations := 0

		for time.Since(start) < config.Duration {
			if queue, ok := q.(interface{ Enqueue(interface{}) bool }); ok {
				queue.Enqueue(operations)
				operations++
			}
			if queue, ok := q.(interface{ Dequeue() (interface{}, bool) }); ok {
				queue.Dequeue()
				operations++
			}
		}

		duration := time.Since(start)
		throughput := float64(operations) / duration.Seconds()

		fmt.Printf("  Operations: %d\n", operations)
		fmt.Printf("  Throughput: %.2f ops/sec\n", throughput)
		fmt.Printf("  Duration: %v\n", duration)
	}
}

// Demo 5: Correctness Testing
func demoCorrectnessVerification() {
	fmt.Println("\n--- Demo 5: Correctness Verification ---")

	// Test linearizability checker
	fmt.Println("\nLinearizability Checker:")
	checker := correctness.NewLinearizabilityChecker()

	// Record some operations
	baseTime := time.Now()
	operations := []correctness.Operation{
		{Type: "enqueue", Value: 1, Success: true, StartTime: baseTime, EndTime: baseTime.Add(time.Microsecond), ThreadID: 0},
		{Type: "enqueue", Value: 2, Success: true, StartTime: baseTime.Add(time.Microsecond), EndTime: baseTime.Add(2 * time.Microsecond), ThreadID: 1},
		{Type: "dequeue", Value: 1, Success: true, StartTime: baseTime.Add(3 * time.Microsecond), EndTime: baseTime.Add(4 * time.Microsecond), ThreadID: 0},
		{Type: "dequeue", Value: 2, Success: true, StartTime: baseTime.Add(4 * time.Microsecond), EndTime: baseTime.Add(5 * time.Microsecond), ThreadID: 1},
	}

	for _, op := range operations {
		checker.RecordOperation(op)
	}

	analysis := checker.AnalyzeHistory()
	fmt.Printf("  Total operations: %d\n", analysis.TotalOperations)
	fmt.Printf("  Threads involved: %d\n", len(analysis.ThreadStats))
	fmt.Printf("  Duration: %v\n", analysis.TotalTestDuration)

	// Test stress testing framework
	fmt.Println("\nStress Testing:")
	fmt.Println("  Basic stress testing functionality available in correctness package")
	fmt.Println("  See correctness_test.go for examples of usage")
}

// Demo 6: Monitoring and Metrics
func demoMonitoring() {
	fmt.Println("\n--- Demo 6: Monitoring and Metrics ---")

	// Demo metrics collector
	fmt.Println("\nMetrics Collector:")
	collector := monitoring.NewMetricsCollector()

	// Simulate some metrics
	for i := 0; i < 100; i++ {
		collector.IncrementCounter("operations", nil)
		collector.SetGauge("queue_size", float64(i%20), nil)
		collector.RecordHistogram("latency", float64(i%10+1), nil)

		timer := collector.StartTimer("operation_time", nil)
		time.Sleep(time.Microsecond * time.Duration(i%5+1))
		timer.Stop()
	}

	metrics := collector.GetMetrics()
	fmt.Printf("  Total metrics collected: %d\n", len(metrics))

	latencyStats := collector.GetHistogramStats("latency", nil)
	if latencyStats != nil {
		fmt.Printf("  Latency histogram - Count: %d, Mean: %.2f\n", latencyStats.Count, latencyStats.Mean)
	}

	timerStats := collector.GetTimerStats("operation_time", nil)
	if timerStats != nil {
		fmt.Printf("  Operation timer - Count: %d, Mean: %v\n", timerStats.Count, timerStats.Mean)
	}

	// Demo queue monitor
	fmt.Println("\nQueue Monitor:")
	monitor := monitoring.NewQueueMonitor("demo_queue")

	// Simulate queue operations
	for i := 0; i < 50; i++ {
		duration := time.Duration(i%10+1) * time.Millisecond
		success := i%10 != 0 // 90% success rate

		monitor.RecordEnqueue(duration, success)
		monitor.RecordDequeue(duration/2, success)
		monitor.RecordQueueSize(int64(i % 25))

		if i%20 == 0 {
			monitor.RecordContention("enqueue", time.Duration(i%5+1)*time.Millisecond)
		}
	}

	report := monitor.GenerateStringReport()
	fmt.Printf("  Generated report:\n%s\n", report)
}

// Demo 7: Queue Comparison
func demoQueueComparison() {
	fmt.Println("\n--- Demo 7: Queue Implementation Comparison ---")

	queues := []string{"Michael & Scott", "Versioned", "Hazard Pointer", "Bounded"}

	fmt.Printf("Available Queue Implementations:\n")
	for i, name := range queues {
		fmt.Printf("%d. %s Queue\n", i+1, name)
	}

	fmt.Println("\nFor detailed performance comparison, see the correctness package tests.")
	fmt.Println("Run 'go test ./correctness -v' to see stress testing results.")
}
