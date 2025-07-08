package counters

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestAtomicCounter(t *testing.T) {
	counter := NewAtomicCounter()
	testBasicCounterOperations(t, counter, "AtomicCounter")
}

func TestStripedCounter(t *testing.T) {
	counter := NewStripedCounter(runtime.NumCPU())
	testBasicCounterOperations(t, counter, "StripedCounter")
	
	// Test stripe-specific functionality
	stripeValues := counter.GetStripeValues()
	if len(stripeValues) != runtime.NumCPU() {
		t.Errorf("Expected %d stripes, got %d", runtime.NumCPU(), len(stripeValues))
	}
}

func TestShardedCounter(t *testing.T) {
	counter := NewShardedCounter()
	testBasicCounterOperations(t, counter, "ShardedCounter")
	
	// Test shard-specific functionality
	shardValues := counter.GetShardValues()
	numShards := counter.GetNumShards()
	if len(shardValues) != numShards {
		t.Errorf("Expected %d shards, got %d", numShards, len(shardValues))
	}
}

func TestNUMACounter(t *testing.T) {
	counter := NewNUMACounter()
	testBasicCounterOperations(t, counter, "NUMACounter")
	
	// Test NUMA-specific functionality
	nodeValues := counter.GetByNode()
	numNodes := counter.GetNumNodes()
	if len(nodeValues) != numNodes {
		t.Errorf("Expected %d nodes, got %d", numNodes, len(nodeValues))
	}
}

func testBasicCounterOperations(t *testing.T, counter interface {
	Add(int64)
	Inc()
	Dec()
	Get() int64
	Reset()
}, name string) {
	// Test initial state
	counter.Reset()
	if value := counter.Get(); value != 0 {
		t.Errorf("%s: Expected initial value 0, got %d", name, value)
	}
	
	// Test Inc
	counter.Inc()
	if value := counter.Get(); value != 1 {
		t.Errorf("%s: Expected value 1 after Inc, got %d", name, value)
	}
	
	// Test Dec
	counter.Dec()
	if value := counter.Get(); value != 0 {
		t.Errorf("%s: Expected value 0 after Dec, got %d", name, value)
	}
	
	// Test Add
	counter.Add(10)
	if value := counter.Get(); value != 10 {
		t.Errorf("%s: Expected value 10 after Add(10), got %d", name, value)
	}
	
	// Test Add negative
	counter.Add(-5)
	if value := counter.Get(); value != 5 {
		t.Errorf("%s: Expected value 5 after Add(-5), got %d", name, value)
	}
	
	// Test Reset
	counter.Reset()
	if value := counter.Get(); value != 0 {
		t.Errorf("%s: Expected value 0 after Reset, got %d", name, value)
	}
}

func TestConcurrentOperations(t *testing.T) {
	counters := []struct {
		name    string
		counter interface {
			Add(int64)
			Inc()
			Dec()
			Get() int64
			Reset()
		}
	}{
		{"AtomicCounter", NewAtomicCounter()},
		{"StripedCounter", NewStripedCounter(runtime.NumCPU())},
		{"ShardedCounter", NewShardedCounter()},
		{"NUMACounter", NewNUMACounter()},
	}
	
	for _, tc := range counters {
		t.Run(tc.name, func(t *testing.T) {
			testConcurrentCounter(t, tc.counter, tc.name)
		})
	}
}

func testConcurrentCounter(t *testing.T, counter interface {
	Add(int64)
	Inc()
	Dec()
	Get() int64
	Reset()
}, name string) {
	counter.Reset()
	
	numGoroutines := runtime.NumCPU() * 2
	operationsPerGoroutine := 1000
	var wg sync.WaitGroup
	
	// Test concurrent increments
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
	
	expected := int64(numGoroutines * operationsPerGoroutine)
	if value := counter.Get(); value != expected {
		t.Errorf("%s: Expected %d after concurrent increments, got %d", name, expected, value)
	}
	
	// Test concurrent mixed operations
	counter.Reset()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if id%2 == 0 {
					counter.Inc()
				} else {
					counter.Add(2)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Half goroutines do Inc (1 each), half do Add(2)
	expectedMixed := int64((numGoroutines/2)*operationsPerGoroutine + (numGoroutines/2)*operationsPerGoroutine*2)
	if value := counter.Get(); value != expectedMixed {
		t.Errorf("%s: Expected %d after mixed operations, got %d", name, expectedMixed, value)
	}
}

func BenchmarkCounters(b *testing.B) {
	counters := []struct {
		name    string
		counter interface {
			Inc()
			Reset()
		}
	}{
		{"AtomicCounter", NewAtomicCounter()},
		{"StripedCounter", NewStripedCounter(runtime.NumCPU())},
		{"ShardedCounter", NewShardedCounter()},
		{"NUMACounter", NewNUMACounter()},
	}
	
	for _, tc := range counters {
		b.Run(tc.name, func(b *testing.B) {
			tc.counter.Reset()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					tc.counter.Inc()
				}
			})
		})
	}
}

func TestCounterPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	counters := []struct {
		name    string
		counter interface {
			Inc()
			Get() int64
			Reset()
		}
	}{
		{"AtomicCounter", NewAtomicCounter()},
		{"StripedCounter", NewStripedCounter(runtime.NumCPU())},
		{"ShardedCounter", NewShardedCounter()},
		{"NUMACounter", NewNUMACounter()},
	}
	
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 100000
	
	for _, tc := range counters {
		t.Run(tc.name, func(t *testing.T) {
			tc.counter.Reset()
			
			start := time.Now()
			var wg sync.WaitGroup
			
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < operationsPerGoroutine; j++ {
						tc.counter.Inc()
					}
				}()
			}
			
			wg.Wait()
			duration := time.Since(start)
			
			finalValue := tc.counter.Get()
			expected := int64(numGoroutines * operationsPerGoroutine)
			opsPerSecond := float64(expected) / duration.Seconds()
			
			if finalValue != expected {
				t.Errorf("%s: Expected %d, got %d", tc.name, expected, finalValue)
			}
			
			t.Logf("%s: %d operations in %v (%.0f ops/sec)", 
				tc.name, finalValue, duration, opsPerSecond)
		})
	}
}