package approximate

import (
	"runtime"
	"sync/atomic"
)

// BoundedCounter maintains bounded error guarantees
type BoundedCounter struct {
	localCounters []localCounter
	threshold     int64
	globalCounter int64
	numCounters   int
	_             [40]byte // Padding
}

type localCounter struct {
	value int64
	_     [56]byte // Padding
}

func NewBoundedCounter(threshold int64, numCounters int) *BoundedCounter {
	if numCounters <= 0 {
		numCounters = runtime.NumCPU()
	}
	return &BoundedCounter{
		localCounters: make([]localCounter, numCounters),
		threshold:     threshold,
		numCounters:   numCounters,
	}
}

func (bc *BoundedCounter) Add(delta int64) {
	counterIdx := bc.getCounterIndex()
	newValue := atomic.AddInt64(&bc.localCounters[counterIdx].value, delta)
	
	// Flush to global counter if threshold exceeded
	if newValue >= bc.threshold {
		if atomic.CompareAndSwapInt64(&bc.localCounters[counterIdx].value, newValue, 0) {
			atomic.AddInt64(&bc.globalCounter, newValue)
		}
	} else if newValue <= -bc.threshold {
		if atomic.CompareAndSwapInt64(&bc.localCounters[counterIdx].value, newValue, 0) {
			atomic.AddInt64(&bc.globalCounter, newValue)
		}
	}
}

func (bc *BoundedCounter) Inc() {
	bc.Add(1)
}

func (bc *BoundedCounter) Dec() {
	bc.Add(-1)
}

func (bc *BoundedCounter) Get() int64 {
	global := atomic.LoadInt64(&bc.globalCounter)
	
	// Add local counter values
	var local int64
	for i := range bc.localCounters {
		local += atomic.LoadInt64(&bc.localCounters[i].value)
	}
	
	return global + local
}

func (bc *BoundedCounter) GetBounds() (int64, int64) {
	value := bc.Get()
	maxError := int64(bc.numCounters) * (bc.threshold - 1)
	return value - maxError, value + maxError
}

func (bc *BoundedCounter) Flush() {
	for i := range bc.localCounters {
		value := atomic.SwapInt64(&bc.localCounters[i].value, 0)
		if value != 0 {
			atomic.AddInt64(&bc.globalCounter, value)
		}
	}
}

func (bc *BoundedCounter) Reset() {
	atomic.StoreInt64(&bc.globalCounter, 0)
	for i := range bc.localCounters {
		atomic.StoreInt64(&bc.localCounters[i].value, 0)
	}
}

func (bc *BoundedCounter) getCounterIndex() int {
	gid := runtime.NumGoroutine()
	return gid % bc.numCounters
}

func (bc *BoundedCounter) GetThreshold() int64 {
	return bc.threshold
}

func (bc *BoundedCounter) GetNumCounters() int {
	return bc.numCounters
}

func (bc *BoundedCounter) GetLocalValues() []int64 {
	values := make([]int64, bc.numCounters)
	for i := range bc.localCounters {
		values[i] = atomic.LoadInt64(&bc.localCounters[i].value)
	}
	return values
}

func (bc *BoundedCounter) GetGlobalValue() int64 {
	return atomic.LoadInt64(&bc.globalCounter)
}