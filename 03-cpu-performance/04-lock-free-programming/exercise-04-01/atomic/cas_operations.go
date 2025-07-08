package atomic

import (
	"runtime"
	"sync/atomic"
	"time"
)

// CASCounter demonstrates various Compare-and-Swap patterns
type CASCounter struct {
	value int64
	_     [7]int64 // Cache line padding
}

func NewCASCounter() *CASCounter {
	return &CASCounter{}
}

// SimpleCAS performs a basic compare-and-swap
func (c *CASCounter) SimpleCAS(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&c.value, old, new)
}

// CASLoop performs CAS in a loop until success
func (c *CASCounter) CASLoop(delta int64) int64 {
	for {
		old := atomic.LoadInt64(&c.value)
		new := old + delta
		if atomic.CompareAndSwapInt64(&c.value, old, new) {
			return new
		}
		// Optional: yield to reduce CPU usage
		runtime.Gosched()
	}
}

// CASWithRetry performs CAS with limited retries
func (c *CASCounter) CASWithRetry(delta int64, maxRetries int) (int64, bool) {
	for i := 0; i < maxRetries; i++ {
		old := atomic.LoadInt64(&c.value)
		new := old + delta
		if atomic.CompareAndSwapInt64(&c.value, old, new) {
			return new, true
		}
		runtime.Gosched()
	}
	return atomic.LoadInt64(&c.value), false
}

// CASWithBackoff performs CAS with exponential backoff
func (c *CASCounter) CASWithBackoff(delta int64) int64 {
	backoff := time.Nanosecond
	maxBackoff := time.Microsecond * 100
	
	for {
		old := atomic.LoadInt64(&c.value)
		new := old + delta
		if atomic.CompareAndSwapInt64(&c.value, old, new) {
			return new
		}
		
		// Exponential backoff
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// Load returns the current value
func (c *CASCounter) Load() int64 {
	return atomic.LoadInt64(&c.value)
}

// Store sets the value
func (c *CASCounter) Store(val int64) {
	atomic.StoreInt64(&c.value, val)
}

// DoubleWidthCAS demonstrates double-width compare-and-swap simulation
type DoubleWidthCAS struct {
	low  int64
	high int64
	_    [6]int64 // Cache line padding
}

func NewDoubleWidthCAS() *DoubleWidthCAS {
	return &DoubleWidthCAS{}
}

// CAS128 simulates 128-bit compare-and-swap using two 64-bit values
func (d *DoubleWidthCAS) CAS128(oldLow, oldHigh, newLow, newHigh int64) bool {
	// This is a simplified simulation - real 128-bit CAS would be atomic
	if atomic.LoadInt64(&d.low) == oldLow && atomic.LoadInt64(&d.high) == oldHigh {
		if atomic.CompareAndSwapInt64(&d.low, oldLow, newLow) {
			if atomic.CompareAndSwapInt64(&d.high, oldHigh, newHigh) {
				return true
			}
			// Rollback low if high CAS failed
			atomic.CompareAndSwapInt64(&d.low, newLow, oldLow)
		}
	}
	return false
}

// Load128 loads both values
func (d *DoubleWidthCAS) Load128() (int64, int64) {
	return atomic.LoadInt64(&d.low), atomic.LoadInt64(&d.high)
}

// Store128 stores both values
func (d *DoubleWidthCAS) Store128(low, high int64) {
	atomic.StoreInt64(&d.low, low)
	atomic.StoreInt64(&d.high, high)
}