package atomic

import (
	"runtime"
	"sync/atomic"
)

// MemoryOrderingDemo demonstrates different memory ordering semantics
type MemoryOrderingDemo struct {
	flag int32
	data int32
	_    [14]int32 // Cache line padding
}

func NewMemoryOrderingDemo() *MemoryOrderingDemo {
	return &MemoryOrderingDemo{}
}

// RelaxedOrdering - No synchronization guarantees
func (m *MemoryOrderingDemo) RelaxedWrite(value int32) {
	atomic.StoreInt32(&m.data, value)
	atomic.StoreInt32(&m.flag, 1)
}

func (m *MemoryOrderingDemo) RelaxedRead() (int32, bool) {
	if atomic.LoadInt32(&m.flag) == 1 {
		return atomic.LoadInt32(&m.data), true
	}
	return 0, false
}

// AcquireReleaseOrdering - Synchronization semantics
func (m *MemoryOrderingDemo) AcquireReleaseWrite(value int32) {
	atomic.StoreInt32(&m.data, value)
	// Release: all previous operations complete before this store
	atomic.StoreInt32(&m.flag, 1)
}

func (m *MemoryOrderingDemo) AcquireReleaseRead() (int32, bool) {
	// Acquire: all subsequent operations happen after this load
	if atomic.LoadInt32(&m.flag) == 1 {
		return atomic.LoadInt32(&m.data), true
	}
	return 0, false
}

// SequentialConsistency - Strongest ordering guarantee
func (m *MemoryOrderingDemo) SequentialWrite(value int32) {
	atomic.StoreInt32(&m.data, value)
	// Note: Go's atomic operations provide sequential consistency by default
	atomic.StoreInt32(&m.flag, 1)
}

func (m *MemoryOrderingDemo) SequentialRead() (int32, bool) {
	if atomic.LoadInt32(&m.flag) == 1 {
		// Note: Go's atomic operations provide sequential consistency by default
		return atomic.LoadInt32(&m.data), true
	}
	return 0, false
}

// Reset clears the flag and data
func (m *MemoryOrderingDemo) Reset() {
	atomic.StoreInt32(&m.flag, 0)
	atomic.StoreInt32(&m.data, 0)
}

// RelaxedIncrement demonstrates relaxed ordering for performance-critical counters
func RelaxedIncrement(counter *int64) {
	// Use relaxed ordering when synchronization not needed
	for {
		old := atomic.LoadInt64(counter)
		if atomic.CompareAndSwapInt64(counter, old, old+1) {
			break
		}
		runtime.Gosched() // Yield to reduce contention
	}
}

// SynchronizedUpdate demonstrates acquire-release for synchronization
func SynchronizedUpdate(flag *int32, data *int64, value int64) {
	atomic.StoreInt64(data, value)
	atomic.StoreInt32(flag, 1) // Release semantics
}

func SynchronizedRead(flag *int32, data *int64) (int64, bool) {
	if atomic.LoadInt32(flag) == 1 { // Acquire semantics
		return atomic.LoadInt64(data), true
	}
	return 0, false
}