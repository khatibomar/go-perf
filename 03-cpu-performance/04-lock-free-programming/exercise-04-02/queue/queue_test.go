package queue

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMichaelScottQueue tests the Michael & Scott queue implementation
func TestMichaelScottQueue(t *testing.T) {
	q := NewMichaelScottQueue()
	testBasicQueueOperations(t, q, "MichaelScottQueue")
	testConcurrentOperations(t, q, "MichaelScottQueue")
}

// TestVersionedQueue tests the versioned queue implementation
func TestVersionedQueue(t *testing.T) {
	q := NewVersionedQueue()
	testBasicQueueOperations(t, q, "VersionedQueue")
	testConcurrentOperations(t, q, "VersionedQueue")
}

// TestHazardPointerQueue tests the hazard pointer queue implementation
func TestHazardPointerQueue(t *testing.T) {
	q := NewHazardPointerQueue()
	testBasicQueueOperations(t, q, "HazardPointerQueue")
	testConcurrentOperations(t, q, "HazardPointerQueue")
	
	// Test cleanup functionality
	q.Cleanup()
	activePointers, retiredPointers := q.Stats()
	t.Logf("HazardPointerQueue stats - Active: %d, Retired: %d", activePointers, retiredPointers)
}

// TestBoundedQueue tests the bounded queue implementation
func TestBoundedQueue(t *testing.T) {
	q := NewBoundedQueue(16) // Power of 2
	testBasicBoundedQueueOperations(t, q)
	testBoundedQueueConcurrency(t, q)
}

// testBasicQueueOperations tests basic queue operations
func testBasicQueueOperations(t *testing.T, q interface{}, name string) {
	t.Run(name+"_BasicOperations", func(t *testing.T) {
		// Test with interface that has Enqueue, Dequeue, IsEmpty, Size methods
		type BasicQueue interface {
			Enqueue(interface{}) bool
			Dequeue() (interface{}, bool)
			IsEmpty() bool
			Size() int
		}
		
		queue, ok := q.(BasicQueue)
		if !ok {
			t.Fatalf("Queue %s does not implement BasicQueue interface", name)
		}
		
		// Test empty queue
		if !queue.IsEmpty() {
			t.Error("New queue should be empty")
		}
		
		if size := queue.Size(); size != 0 {
			t.Errorf("Empty queue size should be 0, got %d", size)
		}
		
		// Test dequeue from empty queue
		if val, ok := queue.Dequeue(); ok {
			t.Errorf("Dequeue from empty queue should fail, got %v", val)
		}
		
		// Test single enqueue/dequeue
		if !queue.Enqueue(42) {
			t.Error("Enqueue should succeed")
		}
		
		if queue.IsEmpty() {
			t.Error("Queue should not be empty after enqueue")
		}
		
		val, ok := queue.Dequeue()
		if !ok {
			t.Error("Dequeue should succeed")
		}
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
		
		if !queue.IsEmpty() {
			t.Error("Queue should be empty after dequeue")
		}
		
		// Test FIFO order
		values := []int{1, 2, 3, 4, 5}
		for _, v := range values {
			if !queue.Enqueue(v) {
				t.Errorf("Enqueue of %d should succeed", v)
			}
		}
		
		for i, expected := range values {
			val, ok := queue.Dequeue()
			if !ok {
				t.Errorf("Dequeue %d should succeed", i)
			}
			if val != expected {
				t.Errorf("Expected %d, got %v at position %d", expected, val, i)
			}
		}
	})
}

// testConcurrentOperations tests concurrent queue operations
func testConcurrentOperations(t *testing.T, q interface{}, name string) {
	t.Run(name+"_ConcurrentOperations", func(t *testing.T) {
		type ConcurrentQueue interface {
			Enqueue(interface{}) bool
			Dequeue() (interface{}, bool)
		}
		
		queue, ok := q.(ConcurrentQueue)
		if !ok {
			t.Fatalf("Queue %s does not implement ConcurrentQueue interface", name)
		}
		
		const (
			numProducers = 2
			numConsumers = 2
			itemsPerProducer = 100
			testTimeout = 10 * time.Second
		)
		
		var wg sync.WaitGroup
		var totalProduced, totalConsumed int64
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		
		// Start producers
		for p := 0; p < numProducers; p++ {
			wg.Add(1)
			go func(producerID int) {
				defer wg.Done()
				produced := 0
				for produced < itemsPerProducer {
					select {
					case <-ctx.Done():
						return
					default:
						value := producerID*itemsPerProducer + produced
						if queue.Enqueue(value) {
							produced++
						} else {
							time.Sleep(time.Microsecond)
						}
					}
				}
				atomic.AddInt64(&totalProduced, int64(produced))
			}(p)
		}
		
		// Start consumers
		for c := 0; c < numConsumers; c++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				consumed := 0
				for {
					select {
					case <-ctx.Done():
						atomic.AddInt64(&totalConsumed, int64(consumed))
						return
					default:
						if _, ok := queue.Dequeue(); ok {
							consumed++
							// Check if we've consumed enough
							if consumed >= numProducers*itemsPerProducer {
								atomic.AddInt64(&totalConsumed, int64(consumed))
								return
							}
						} else {
							time.Sleep(time.Microsecond)
						}
					}
				}
			}()
		}
		
		wg.Wait()
		
		// Verify reasonable production and consumption
		producedCount := atomic.LoadInt64(&totalProduced)
		consumedCount := atomic.LoadInt64(&totalConsumed)
		
		t.Logf("%s concurrent test - Produced: %d, Consumed: %d", name, producedCount, consumedCount)
		
		// We should have produced some items
		if producedCount == 0 {
			t.Error("No items were produced")
		}
		
		// Consumed should not exceed produced
		if consumedCount > producedCount {
			t.Errorf("Consumed more items (%d) than produced (%d)", consumedCount, producedCount)
		}
		
		// We should have consumed at least some items if any were produced
		if producedCount > 0 && consumedCount == 0 {
			t.Error("Items were produced but none were consumed")
		}
	})
}

// testBasicBoundedQueueOperations tests basic bounded queue operations
func testBasicBoundedQueueOperations(t *testing.T, q *BoundedQueue) {
	t.Run("BoundedQueue_BasicOperations", func(t *testing.T) {
		// Test empty queue
		if !q.IsEmpty() {
			t.Error("New bounded queue should be empty")
		}
		
		if q.IsFull() {
			t.Error("New bounded queue should not be full")
		}
		
		if size := q.Size(); size != 0 {
			t.Errorf("Empty queue size should be 0, got %d", size)
		}
		
		// Fill the queue
		capacity := q.Capacity()
		for i := uint64(0); i < capacity; i++ {
			if !q.Enqueue(int(i)) {
				t.Errorf("Enqueue %d should succeed", i)
			}
		}
		
		if !q.IsFull() {
			t.Error("Queue should be full")
		}
		
		// Try to enqueue when full
		if q.Enqueue(999) {
			t.Error("Enqueue should fail when queue is full")
		}
		
		// Dequeue all items
		for i := uint64(0); i < capacity; i++ {
			val, ok := q.Dequeue()
			if !ok {
				t.Errorf("Dequeue %d should succeed", i)
			}
			if val != int(i) {
				t.Errorf("Expected %d, got %v", i, val)
			}
		}
		
		if !q.IsEmpty() {
			t.Error("Queue should be empty after dequeuing all items")
		}
		
		// Try to dequeue when empty
		if val, ok := q.Dequeue(); ok {
			t.Errorf("Dequeue should fail when queue is empty, got %v", val)
		}
	})
}

// testBoundedQueueConcurrency tests bounded queue under concurrent load
func testBoundedQueueConcurrency(t *testing.T, q *BoundedQueue) {
	t.Run("BoundedQueue_Concurrency", func(t *testing.T) {
		const (
			numProducers = 2
			numConsumers = 2
			itemsPerProducer = 500
			testDuration = 2 * time.Second
		)
		
		var wg sync.WaitGroup
		var totalProduced, totalConsumed int64
		
		// Start producers
		for p := 0; p < numProducers; p++ {
			wg.Add(1)
			go func(producerID int) {
				defer wg.Done()
				start := time.Now()
				produced := 0
				
				for time.Since(start) < testDuration && produced < itemsPerProducer {
					value := producerID*itemsPerProducer + produced
					if q.Enqueue(value) {
						produced++
					}
					runtime.Gosched()
				}
				
				totalProduced += int64(produced)
			}(p)
		}
		
		// Start consumers
		for c := 0; c < numConsumers; c++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				start := time.Now()
				consumed := 0
				
				for time.Since(start) < testDuration+time.Second { // Extra time to consume remaining
					if _, ok := q.Dequeue(); ok {
						consumed++
					}
					runtime.Gosched()
				}
				
				totalConsumed += int64(consumed)
			}()
		}
		
		wg.Wait()
		
		t.Logf("BoundedQueue concurrent test - Produced: %d, Consumed: %d", totalProduced, totalConsumed)
		
		// The number of consumed items should be close to produced items
		// (allowing for some items still in queue)
		if totalConsumed > totalProduced {
			t.Errorf("Consumed more items (%d) than produced (%d)", totalConsumed, totalProduced)
		}
	})
}

// BenchmarkQueues benchmarks different queue implementations
func BenchmarkQueues(b *testing.B) {
	queues := map[string]func() interface{}{
		"MichaelScott": func() interface{} { return NewMichaelScottQueue() },
		"Versioned":    func() interface{} { return NewVersionedQueue() },
		"HazardPointer": func() interface{} { return NewHazardPointerQueue() },
		"Bounded":      func() interface{} { return NewBoundedQueue(1024) },
	}
	
	for name, createQueue := range queues {
		b.Run(name, func(b *testing.B) {
			queue := createQueue()
			
			type BenchQueue interface {
				Enqueue(interface{}) bool
				Dequeue() (interface{}, bool)
			}
			
			q, ok := queue.(BenchQueue)
			if !ok {
				b.Fatalf("Queue %s does not implement BenchQueue interface", name)
			}
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				value := 42
				for pb.Next() {
					q.Enqueue(value)
					q.Dequeue()
				}
			})
		})
	}
}

// TestQueueStress performs stress testing on all queue implementations
func TestQueueStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	queues := map[string]func() interface{}{
		"MichaelScott": func() interface{} { return NewMichaelScottQueue() },
		"Versioned":    func() interface{} { return NewVersionedQueue() },
		"HazardPointer": func() interface{} { return NewHazardPointerQueue() },
		"Bounded":      func() interface{} { return NewBoundedQueue(1024) },
	}
	
	for name, createQueue := range queues {
		t.Run(name+"_Stress", func(t *testing.T) {
			queue := createQueue()
			
			type StressQueue interface {
				Enqueue(interface{}) bool
				Dequeue() (interface{}, bool)
			}
			
			q, ok := queue.(StressQueue)
			if !ok {
				t.Fatalf("Queue %s does not implement StressQueue interface", name)
			}
			
			const (
				numGoroutines = 8
				operationsPerGoroutine = 10000
			)
			
			var wg sync.WaitGroup
			
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					
					for j := 0; j < operationsPerGoroutine; j++ {
						value := id*operationsPerGoroutine + j
						
						// Enqueue
						for !q.Enqueue(value) {
							runtime.Gosched()
						}
						
						// Dequeue
						for {
							if _, ok := q.Dequeue(); ok {
								break
							}
							runtime.Gosched()
						}
					}
				}(i)
			}
			
			wg.Wait()
			t.Logf("%s stress test completed successfully", name)
		})
	}
}