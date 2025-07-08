package cas

import (
	"sync/atomic"
	"runtime"
)

// SimpleCASLoop demonstrates basic CAS retry pattern
func SimpleCASLoop(addr *int64, update func(int64) int64) int64 {
	for {
		old := atomic.LoadInt64(addr)
		new := update(old)
		
		if atomic.CompareAndSwapInt64(addr, old, new) {
			return new
		}
		
		// Simple yield on failure
		runtime.Gosched()
	}
}

// OptimisticUpdate implements optimistic concurrency pattern
func OptimisticUpdate(addr *int64, compute func(int64) (int64, bool)) bool {
	old := atomic.LoadInt64(addr)
	new, shouldUpdate := compute(old)
	
	if !shouldUpdate {
		return false
	}
	
	return atomic.CompareAndSwapInt64(addr, old, new)
}

// CASWithRetryLimit limits retry attempts
func CASWithRetryLimit(addr *int64, old, new int64, maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		if atomic.CompareAndSwapInt64(addr, old, new) {
			return true
		}
		
		// Check if value changed (someone else succeeded)
		current := atomic.LoadInt64(addr)
		if current != old {
			return false
		}
		
		runtime.Gosched()
	}
	return false
}

// AtomicIncrement demonstrates simple atomic increment
func AtomicIncrement(addr *int64) int64 {
	return atomic.AddInt64(addr, 1)
}

// AtomicMax atomically updates to maximum value
func AtomicMax(addr *int64, value int64) int64 {
	for {
		old := atomic.LoadInt64(addr)
		if value <= old {
			return old
		}
		if atomic.CompareAndSwapInt64(addr, old, value) {
			return value
		}
	}
}