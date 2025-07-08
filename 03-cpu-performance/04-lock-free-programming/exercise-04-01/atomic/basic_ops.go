package atomic

import (
	"sync/atomic"
	"unsafe"
)

// AtomicCounter demonstrates basic atomic operations
type AtomicCounter struct {
	value int64
	_     [7]int64 // Cache line padding
}

func NewAtomicCounter() *AtomicCounter {
	return &AtomicCounter{}
}

func (c *AtomicCounter) Load() int64 {
	return atomic.LoadInt64(&c.value)
}

func (c *AtomicCounter) Store(val int64) {
	atomic.StoreInt64(&c.value, val)
}

func (c *AtomicCounter) Add(delta int64) int64 {
	return atomic.AddInt64(&c.value, delta)
}

func (c *AtomicCounter) CompareAndSwap(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&c.value, old, new)
}

func (c *AtomicCounter) Swap(new int64) int64 {
	return atomic.SwapInt64(&c.value, new)
}

// AtomicPointer demonstrates pointer-based atomic operations
type AtomicPointer struct {
	ptr unsafe.Pointer
	_   [7]uintptr // Cache line padding
}

func NewAtomicPointer() *AtomicPointer {
	return &AtomicPointer{}
}

func (p *AtomicPointer) Load() unsafe.Pointer {
	return atomic.LoadPointer(&p.ptr)
}

func (p *AtomicPointer) Store(ptr unsafe.Pointer) {
	atomic.StorePointer(&p.ptr, ptr)
}

func (p *AtomicPointer) CompareAndSwap(old, new unsafe.Pointer) bool {
	return atomic.CompareAndSwapPointer(&p.ptr, old, new)
}

func (p *AtomicPointer) Swap(new unsafe.Pointer) unsafe.Pointer {
	return atomic.SwapPointer(&p.ptr, new)
}

// AtomicBool demonstrates boolean atomic operations
type AtomicBool struct {
	value int32
	_     [15]int32 // Cache line padding
}

func NewAtomicBool() *AtomicBool {
	return &AtomicBool{}
}

func (b *AtomicBool) Load() bool {
	return atomic.LoadInt32(&b.value) != 0
}

func (b *AtomicBool) Store(val bool) {
	var intVal int32
	if val {
		intVal = 1
	}
	atomic.StoreInt32(&b.value, intVal)
}

func (b *AtomicBool) CompareAndSwap(old, new bool) bool {
	var oldVal, newVal int32
	if old {
		oldVal = 1
	}
	if new {
		newVal = 1
	}
	return atomic.CompareAndSwapInt32(&b.value, oldVal, newVal)
}

func (b *AtomicBool) Swap(new bool) bool {
	var newVal int32
	if new {
		newVal = 1
	}
	oldVal := atomic.SwapInt32(&b.value, newVal)
	return oldVal != 0
}