package counters

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// StripedCounter reduces contention using multiple stripes
type StripedCounter struct {
	stripes []paddedCounter
	mask    uint64
}

type paddedCounter struct {
	value int64
	_     [56]byte // Cache line padding
}

func NewStripedCounter(numStripes int) *StripedCounter {
	// Ensure power of 2 for efficient masking
	stripes := nextPowerOf2(numStripes)
	
	return &StripedCounter{
		stripes: make([]paddedCounter, stripes),
		mask:    uint64(stripes - 1),
	}
}

func (c *StripedCounter) Add(delta int64) {
	stripe := c.getStripeIndex()
	atomic.AddInt64(&c.stripes[stripe].value, delta)
}

func (c *StripedCounter) Inc() {
	c.Add(1)
}

func (c *StripedCounter) Dec() {
	c.Add(-1)
}

func (c *StripedCounter) Get() int64 {
	var total int64
	for i := range c.stripes {
		total += atomic.LoadInt64(&c.stripes[i].value)
	}
	return total
}

func (c *StripedCounter) GetStripeValues() []int64 {
	values := make([]int64, len(c.stripes))
	for i := range c.stripes {
		values[i] = atomic.LoadInt64(&c.stripes[i].value)
	}
	return values
}

func (c *StripedCounter) Reset() {
	for i := range c.stripes {
		atomic.StoreInt64(&c.stripes[i].value, 0)
	}
}

func (c *StripedCounter) getStripeIndex() int {
	// Use goroutine ID hash for stripe selection
	gid := getGoroutineID()
	return int(gid & c.mask)
}

// Helper functions
func nextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}

func getGoroutineID() uint64 {
	// Simple hash based on stack pointer and runtime info
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	return uint64(uintptr(unsafe.Pointer(&buf[0]))) + uint64(n)
}