package counters

import (
	"sync/atomic"
)

// AtomicCounter implements a simple atomic counter
type AtomicCounter struct {
	value int64
	_     [56]byte // Padding to prevent false sharing
}

func NewAtomicCounter() *AtomicCounter {
	return &AtomicCounter{}
}

func (c *AtomicCounter) Add(delta int64) {
	atomic.AddInt64(&c.value, delta)
}

func (c *AtomicCounter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

func (c *AtomicCounter) Dec() {
	atomic.AddInt64(&c.value, -1)
}

func (c *AtomicCounter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

func (c *AtomicCounter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}