package atomic

import (
	"sync/atomic"
	"unsafe"
)

// ArithmeticCounter demonstrates atomic arithmetic operations
type ArithmeticCounter struct {
	value int64
	_     [7]int64 // Cache line padding
}

func NewArithmeticCounter() *ArithmeticCounter {
	return &ArithmeticCounter{}
}

// Add performs atomic addition
func (a *ArithmeticCounter) Add(delta int64) int64 {
	return atomic.AddInt64(&a.value, delta)
}

// Increment atomically increments by 1
func (a *ArithmeticCounter) Increment() int64 {
	return atomic.AddInt64(&a.value, 1)
}

// Decrement atomically decrements by 1
func (a *ArithmeticCounter) Decrement() int64 {
	return atomic.AddInt64(&a.value, -1)
}

// Load returns the current value
func (a *ArithmeticCounter) Load() int64 {
	return atomic.LoadInt64(&a.value)
}

// Store sets the value
func (a *ArithmeticCounter) Store(val int64) {
	atomic.StoreInt64(&a.value, val)
}

// Reset sets the value to zero
func (a *ArithmeticCounter) Reset() {
	atomic.StoreInt64(&a.value, 0)
}

// BitwiseOperations demonstrates atomic bitwise operations
type BitwiseOperations struct {
	value uint64
	_     [7]uint64 // Cache line padding
}

func NewBitwiseOperations() *BitwiseOperations {
	return &BitwiseOperations{}
}

// SetBit sets a specific bit using CAS
func (b *BitwiseOperations) SetBit(bit uint) bool {
	mask := uint64(1) << bit
	for {
		old := atomic.LoadUint64(&b.value)
		if old&mask != 0 {
			return false // Bit already set
		}
		new := old | mask
		if atomic.CompareAndSwapUint64(&b.value, old, new) {
			return true
		}
	}
}

// ClearBit clears a specific bit using CAS
func (b *BitwiseOperations) ClearBit(bit uint) bool {
	mask := uint64(1) << bit
	for {
		old := atomic.LoadUint64(&b.value)
		if old&mask == 0 {
			return false // Bit already clear
		}
		new := old &^ mask
		if atomic.CompareAndSwapUint64(&b.value, old, new) {
			return true
		}
	}
}

// ToggleBit toggles a specific bit using CAS
func (b *BitwiseOperations) ToggleBit(bit uint) {
	mask := uint64(1) << bit
	for {
		old := atomic.LoadUint64(&b.value)
		new := old ^ mask
		if atomic.CompareAndSwapUint64(&b.value, old, new) {
			return
		}
	}
}

// TestBit checks if a specific bit is set
func (b *BitwiseOperations) TestBit(bit uint) bool {
	mask := uint64(1) << bit
	return atomic.LoadUint64(&b.value)&mask != 0
}

// Load returns the current value
func (b *BitwiseOperations) Load() uint64 {
	return atomic.LoadUint64(&b.value)
}

// Store sets the value
func (b *BitwiseOperations) Store(val uint64) {
	atomic.StoreUint64(&b.value, val)
}

// PopCount returns the number of set bits
func (b *BitwiseOperations) PopCount() int {
	val := atomic.LoadUint64(&b.value)
	count := 0
	for val != 0 {
		count++
		val &= val - 1 // Clear the lowest set bit
	}
	return count
}

// AtomicFloat64 demonstrates atomic operations on float64 using uint64
type AtomicFloat64 struct {
	bits uint64
	_    [7]uint64 // Cache line padding
}

func NewAtomicFloat64() *AtomicFloat64 {
	return &AtomicFloat64{}
}

// Load returns the current float64 value
func (f *AtomicFloat64) Load() float64 {
	bits := atomic.LoadUint64(&f.bits)
	return *(*float64)(unsafe.Pointer(&bits))
}

// Store sets the float64 value
func (f *AtomicFloat64) Store(val float64) {
	bits := *(*uint64)(unsafe.Pointer(&val))
	atomic.StoreUint64(&f.bits, bits)
}

// CompareAndSwap performs CAS on float64
func (f *AtomicFloat64) CompareAndSwap(old, new float64) bool {
	oldBits := *(*uint64)(unsafe.Pointer(&old))
	newBits := *(*uint64)(unsafe.Pointer(&new))
	return atomic.CompareAndSwapUint64(&f.bits, oldBits, newBits)
}