package cas

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// TestSimpleCASLoop tests the basic CAS loop functionality
func TestSimpleCASLoop(t *testing.T) {
	var counter int64
	
	// Test increment
	result := SimpleCASLoop(&counter, func(old int64) int64 {
		return old + 1
	})
	
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
	
	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected counter to be 1, got %d", atomic.LoadInt64(&counter))
	}
}

// TestOptimisticUpdate tests optimistic update pattern
func TestOptimisticUpdate(t *testing.T) {
	var counter int64
	
	// Test successful update
	success := OptimisticUpdate(&counter, func(old int64) (int64, bool) {
		return old + 5, true
	})
	
	if !success {
		t.Error("Expected update to succeed")
	}
	
	if atomic.LoadInt64(&counter) != 5 {
		t.Errorf("Expected counter to be 5, got %d", atomic.LoadInt64(&counter))
	}
	
	// Test rejected update
	success = OptimisticUpdate(&counter, func(old int64) (int64, bool) {
		return old + 10, false
	})
	
	if success {
		t.Error("Expected update to be rejected")
	}
	
	if atomic.LoadInt64(&counter) != 5 {
		t.Errorf("Expected counter to remain 5, got %d", atomic.LoadInt64(&counter))
	}
}

// TestCASWithRetryLimit tests CAS with retry limits
func TestCASWithRetryLimit(t *testing.T) {
	var counter int64
	
	// Test successful CAS within retry limit
	success := CASWithRetryLimit(&counter, 0, 1, 5)
	if !success {
		t.Error("Expected CAS to succeed")
	}
	
	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected counter to be 1, got %d", atomic.LoadInt64(&counter))
	}
	
	// Test failed CAS (wrong expected value)
	success = CASWithRetryLimit(&counter, 0, 2, 5)
	if success {
		t.Error("Expected CAS to fail")
	}
	
	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected counter to remain 1, got %d", atomic.LoadInt64(&counter))
	}
}

// TestAtomicIncrement tests atomic increment
func TestAtomicIncrement(t *testing.T) {
	var counter int64
	
	result := AtomicIncrement(&counter)
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
	
	result = AtomicIncrement(&counter)
	if result != 2 {
		t.Errorf("Expected 2, got %d", result)
	}
}

// TestAtomicMax tests atomic max operation
func TestAtomicMax(t *testing.T) {
	var value int64
	
	// Test setting initial value
	result := AtomicMax(&value, 10)
	if result != 10 {
		t.Errorf("Expected 10, got %d", result)
	}
	
	// Test setting larger value
	result = AtomicMax(&value, 15)
	if result != 15 {
		t.Errorf("Expected 15, got %d", result)
	}
	
	// Test setting smaller value (should not change)
	result = AtomicMax(&value, 5)
	if result != 15 {
		t.Errorf("Expected 15, got %d", result)
	}
}

// TestOptimisticCounter tests the optimistic counter
func TestOptimisticCounter(t *testing.T) {
	counter := NewOptimisticCounter()
	
	// Test increment
	result := counter.Increment()
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
	
	// Test add
	result = counter.Add(5)
	if result != 6 {
		t.Errorf("Expected 6, got %d", result)
	}
	
	// Test get
	value := counter.Get()
	if value != 6 {
		t.Errorf("Expected 6, got %d", value)
	}
}

// TestOptimisticMap tests the optimistic map
func TestOptimisticMap(t *testing.T) {
	m := NewOptimisticMap()
	
	// Test update
	success := m.Update("key1", func(old int64) int64 {
		return old + 1
	})
	
	if !success {
		t.Error("Expected update to succeed")
	}
	
	// Test get
	value := m.Get("key1")
	if value != 1 {
		t.Errorf("Expected 1, got %d", value)
	}
	
	// Test update existing key
	success = m.Update("key1", func(old int64) int64 {
		return old + 5
	})
	
	if !success {
		t.Error("Expected update to succeed")
	}
	
	value = m.Get("key1")
	if value != 6 {
		t.Errorf("Expected 6, got %d", value)
	}
}

// TestHelpingCounter tests the helping counter
func TestHelpingCounter(t *testing.T) {
	counter := NewHelpingCounter()
	
	// Test increment
	result := counter.Increment()
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
	
	// Test add
	result = counter.Add(5)
	if result != 6 {
		t.Errorf("Expected 6, got %d", result)
	}
	
	// Test get
	value := counter.Get()
	if value != 6 {
		t.Errorf("Expected 6, got %d", value)
	}
}

// TestVersionedCounter tests the versioned counter
func TestVersionedCounter(t *testing.T) {
	counter := NewVersionedCounter()
	
	// Test increment
	result := counter.Increment()
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
	
	// Test add
	result = counter.Add(5)
	if result != 6 {
		t.Errorf("Expected 6, got %d", result)
	}
	
	// Test get
	value, version := counter.Get()
	if value != 6 {
		t.Errorf("Expected 6, got %d", value)
	}
	
	// Test version increment
	initialVersion := version
	counter.Increment()
	_, newVersion := counter.Get()
	
	if newVersion <= initialVersion {
		t.Error("Expected version to increment")
	}
}

// TestVersionedCAS tests versioned CAS operations
func TestVersionedCAS(t *testing.T) {
	var value VersionedValue
	value.value = 10
	value.version = 1
	
	// Test successful CAS
	success := VersionedCAS(&value, 10, 1, 20, 2)
	if !success {
		t.Error("Expected CAS to succeed")
	}
	
	if value.value != 20 || value.version != 2 {
		t.Errorf("Expected value=20, version=2, got value=%d, version=%d", value.value, value.version)
	}
	
	// Test failed CAS (wrong version)
	success = VersionedCAS(&value, 20, 1, 30, 3)
	if success {
		t.Error("Expected CAS to fail due to version mismatch")
	}
	
	if value.value != 20 || value.version != 2 {
		t.Errorf("Expected value=20, version=2, got value=%d, version=%d", value.value, value.version)
	}
}

// TestConcurrentOperations tests concurrent operations
func TestConcurrentOperations(t *testing.T) {
	var counter int64
	var wg sync.WaitGroup
	goroutines := 10
	operationsPerGoroutine := 1000
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				AtomicIncrement(&counter)
			}
		}()
	}
	
	wg.Wait()
	
	expected := int64(goroutines * operationsPerGoroutine)
	if atomic.LoadInt64(&counter) != expected {
		t.Errorf("Expected %d, got %d", expected, atomic.LoadInt64(&counter))
	}
}

// TestConcurrentOptimisticCounter tests concurrent optimistic counter operations
func TestConcurrentOptimisticCounter(t *testing.T) {
	counter := NewOptimisticCounter()
	var wg sync.WaitGroup
	goroutines := 10
	operationsPerGoroutine := 100
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Increment()
			}
		}()
	}
	
	wg.Wait()
	
	expected := int64(goroutines * operationsPerGoroutine)
	value := counter.Get()
	if value != expected {
		t.Errorf("Expected %d, got %d", expected, value)
	}
}

// TestConcurrentVersionedCounter tests concurrent versioned counter operations
func TestConcurrentVersionedCounter(t *testing.T) {
	counter := NewVersionedCounter()
	var wg sync.WaitGroup
	goroutines := 10
	operationsPerGoroutine := 100
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Increment()
			}
		}()
	}
	
	wg.Wait()
	
	expected := int64(goroutines * operationsPerGoroutine)
	value := counter.GetValue()
	if value != expected {
		t.Errorf("Expected %d, got %d", expected, value)
	}
}

// TestDoubleWidthCAS tests double-width CAS operations
func TestDoubleWidthCAS(t *testing.T) {
	// Skip this test if DoubleWidth is not implemented
	t.Skip("DoubleWidth CAS not implemented")
}

// TestRetryWithBackoff tests retry mechanisms with backoff
func TestRetryWithBackoff(t *testing.T) {
	var counter int64
	var attempts int64
	
	// Simulate contention by having multiple goroutines
	var wg sync.WaitGroup
	goroutines := 20
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				for {
					old := atomic.LoadInt64(&counter)
					atomic.AddInt64(&attempts, 1)
					// Add some busy work to increase contention window
					for k := 0; k < 100; k++ {
						_ = k * k
					}
					if atomic.CompareAndSwapInt64(&counter, old, old+1) {
						break
					}
					// Simple backoff
					runtime.Gosched()
				}
			}
		}()
	}
	
	wg.Wait()
	
	expected := int64(goroutines * 50)
	if atomic.LoadInt64(&counter) != expected {
		t.Errorf("Expected %d, got %d", expected, atomic.LoadInt64(&counter))
	}
	
	// Verify that some retries occurred due to contention
	if atomic.LoadInt64(&attempts) <= expected {
		t.Error("Expected some retry attempts due to contention")
	}
}