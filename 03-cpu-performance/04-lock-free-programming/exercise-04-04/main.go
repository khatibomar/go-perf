package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"exercise-04-04/adaptive"
	"exercise-04-04/approximate"
	"exercise-04-04/counters"
)

func main() {
	fmt.Println("High-Performance Counter Design Exercise")
	fmt.Println("========================================")
	
	// Test basic counters
	testBasicCounters()
	
	// Test NUMA-aware counter
	testNUMACounter()
	
	// Test approximate counters
	testApproximateCounters()
	
	// Test adaptive algorithms
	testAdaptiveAlgorithms()
	
	fmt.Println("\nAll tests completed successfully!")
}

func testBasicCounters() {
	fmt.Println("\n1. Testing Basic Counters")
	fmt.Println("--------------------------")
	
	// Test AtomicCounter
	fmt.Println("Testing AtomicCounter...")
	atomicCounter := counters.NewAtomicCounter()
	testCounter(atomicCounter, "Atomic")
	
	// Test StripedCounter
	fmt.Println("Testing StripedCounter...")
	stripedCounter := counters.NewStripedCounter(runtime.NumCPU())
	testCounter(stripedCounter, "Striped")
	
	// Test ShardedCounter
	fmt.Println("Testing ShardedCounter...")
	shardedCounter := counters.NewShardedCounter()
	testCounter(shardedCounter, "Sharded")
}

func testNUMACounter() {
	fmt.Println("\n2. Testing NUMA-Aware Counter")
	fmt.Println("------------------------------")
	
	numaCounter := counters.NewNUMACounter()
	testCounter(numaCounter, "NUMA")
	
	// Show NUMA-specific metrics
	nodeValues := numaCounter.GetByNode()
	fmt.Printf("NUMA node values: %v\n", nodeValues)
	fmt.Printf("Number of NUMA nodes: %d\n", numaCounter.GetNumNodes())
}

func testApproximateCounters() {
	fmt.Println("\n3. Testing Approximate Counters")
	fmt.Println("--------------------------------")
	
	// Test ProbabilisticCounter
	fmt.Println("Testing ProbabilisticCounter...")
	probCounter := approximate.NewProbabilisticCounter(0.1) // 10% sampling
	testApproximateCounter(probCounter, "Probabilistic")
	
	// Test BoundedCounter
	fmt.Println("Testing BoundedCounter...")
	boundedCounter := approximate.NewBoundedCounter(100, runtime.NumCPU())
	testBoundedCounter(boundedCounter)
	
	// Test HierarchicalCounter
	fmt.Println("Testing HierarchicalCounter...")
	hierCounter := approximate.NewHierarchicalCounter(50, 3, 4)
	testHierarchicalCounter(hierCounter)
}

func testAdaptiveAlgorithms() {
	fmt.Println("\n4. Testing Adaptive Algorithms")
	fmt.Println("-------------------------------")
	
	// Test AdaptiveCounter
	fmt.Println("Testing AdaptiveCounter...")
	adaptiveCounter := adaptive.NewAdaptiveCounter()
	testAdaptiveCounter(adaptiveCounter)
	
	// Test LoadBalancer
	fmt.Println("Testing LoadBalancer...")
	loadBalancer := adaptive.NewLoadBalancer(4, adaptive.RoundRobinMode)
	testLoadBalancer(loadBalancer)
}

func testCounter(counter interface{ Add(int64); Inc(); Dec(); Get() int64; Reset() }, name string) {
	counter.Reset()
	
	start := time.Now()
	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 10000
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	finalValue := counter.Get()
	expected := int64(numGoroutines * operationsPerGoroutine)
	opsPerSecond := float64(expected) / duration.Seconds()
	
	fmt.Printf("%s Counter: %d operations in %v (%.0f ops/sec)\n", 
		name, finalValue, duration, opsPerSecond)
	
	if finalValue != expected {
		fmt.Printf("WARNING: Expected %d, got %d\n", expected, finalValue)
	}
}

func testApproximateCounter(counter *approximate.ProbabilisticCounter, name string) {
	counter.Reset()
	
	start := time.Now()
	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 10000
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	estimate := counter.Get()
	exact := counter.GetExact()
	expected := int64(numGoroutines * operationsPerGoroutine)
	opsPerSecond := float64(expected) / duration.Seconds()
	errorPercent := float64(abs(estimate-expected)) / float64(expected) * 100
	
	fmt.Printf("%s Counter: estimate=%d, exact=%d, expected=%d (%.1f%% error, %.0f ops/sec)\n", 
		name, estimate, exact, expected, errorPercent, opsPerSecond)
	
	lower, upper := counter.GetErrorBounds(0.95)
	fmt.Printf("95%% confidence interval: [%.0f, %.0f]\n", lower, upper)
}

func testBoundedCounter(counter *approximate.BoundedCounter) {
	counter.Reset()
	
	start := time.Now()
	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 5000
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	value := counter.Get()
	lower, upper := counter.GetBounds()
	expected := int64(numGoroutines * operationsPerGoroutine)
	opsPerSecond := float64(expected) / duration.Seconds()
	
	fmt.Printf("Bounded Counter: %d, bounds=[%d, %d], expected=%d (%.0f ops/sec)\n", 
		value, lower, upper, expected, opsPerSecond)
	
	localValues := counter.GetLocalValues()
	fmt.Printf("Local values: %v\n", localValues)
}

func testHierarchicalCounter(counter *approximate.HierarchicalCounter) {
	counter.Reset()
	
	start := time.Now()
	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 5000
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	value := counter.Get()
	levelValues := counter.GetByLevel()
	expected := int64(numGoroutines * operationsPerGoroutine)
	opsPerSecond := float64(expected) / duration.Seconds()
	
	fmt.Printf("Hierarchical Counter: %d, expected=%d (%.0f ops/sec)\n", 
		value, expected, opsPerSecond)
	fmt.Printf("Level values: %v\n", levelValues)
}

func testAdaptiveCounter(counter *adaptive.AdaptiveCounter) {
	counter.Reset()
	
	start := time.Now()
	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU() * 2
	operationsPerGoroutine := 5000
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Inc()
				if j%1000 == 0 {
					time.Sleep(time.Microsecond) // Simulate varying load
				}
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	value := counter.Get()
	strategy := counter.GetStrategyName()
	contention := counter.GetContentionLevel()
	expected := int64(numGoroutines * operationsPerGoroutine)
	opsPerSecond := float64(expected) / duration.Seconds()
	
	fmt.Printf("Adaptive Counter: %d, strategy=%s, contention=%d (%.0f ops/sec)\n", 
		value, strategy, contention, opsPerSecond)
}

func testLoadBalancer(lb *adaptive.LoadBalancer) {
	lb.Reset()
	
	start := time.Now()
	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 5000
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				lb.Inc()
			}
		}()
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	value := lb.Get()
	counterValues := lb.GetCounterValues()
	loadMetrics := lb.GetLoadMetrics()
	expected := int64(numGoroutines * operationsPerGoroutine)
	opsPerSecond := float64(expected) / duration.Seconds()
	
	fmt.Printf("Load Balancer: %d, mode=%s (%.0f ops/sec)\n", 
		value, lb.GetBalanceMode().String(), opsPerSecond)
	fmt.Printf("Counter values: %v\n", counterValues)
	fmt.Printf("Load metrics: operations=%v\n", 
		func() []int64 {
			ops := make([]int64, len(loadMetrics))
			for i, m := range loadMetrics {
				ops[i] = m.Operations
			}
			return ops
		}())
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}