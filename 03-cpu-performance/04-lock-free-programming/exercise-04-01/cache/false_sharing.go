package cache

import (
	"sync/atomic"
	"unsafe"
)

const CacheLineSize = 64

// FalseSharingExample demonstrates false sharing problem
type FalseSharingExample struct {
	counter1 int64
	counter2 int64 // Same cache line as counter1
}

func NewFalseSharingExample() *FalseSharingExample {
	return &FalseSharingExample{}
}

func (f *FalseSharingExample) IncrementCounter1() {
	atomic.AddInt64(&f.counter1, 1)
}

func (f *FalseSharingExample) IncrementCounter2() {
	atomic.AddInt64(&f.counter2, 1)
}

func (f *FalseSharingExample) GetCounters() (int64, int64) {
	return atomic.LoadInt64(&f.counter1), atomic.LoadInt64(&f.counter2)
}

// CacheAlignedCounters prevents false sharing
type CacheAlignedCounters struct {
	counter1 int64
	_        [CacheLineSize/8 - 1]int64 // Padding
	counter2 int64
	_        [CacheLineSize/8 - 1]int64 // Padding
}

func NewCacheAlignedCounters() *CacheAlignedCounters {
	return &CacheAlignedCounters{}
}

func (c *CacheAlignedCounters) IncrementCounter1() {
	atomic.AddInt64(&c.counter1, 1)
}

func (c *CacheAlignedCounters) IncrementCounter2() {
	atomic.AddInt64(&c.counter2, 1)
}

func (c *CacheAlignedCounters) GetCounters() (int64, int64) {
	return atomic.LoadInt64(&c.counter1), atomic.LoadInt64(&c.counter2)
}

// DynamicAlignedCounter for runtime cache line alignment
type DynamicAlignedCounter struct {
	data []byte
}

func NewDynamicAlignedCounter() *DynamicAlignedCounter {
	// Allocate extra space to ensure alignment
	data := make([]byte, CacheLineSize*2)
	return &DynamicAlignedCounter{data: data}
}

func (d *DynamicAlignedCounter) getAlignedPointer() *int64 {
	// Find cache-line aligned address
	addr := uintptr(unsafe.Pointer(&d.data[0]))
	aligned := (addr + CacheLineSize - 1) &^ (CacheLineSize - 1)
	return (*int64)(unsafe.Pointer(aligned))
}

func (d *DynamicAlignedCounter) Increment() {
	ptr := d.getAlignedPointer()
	atomic.AddInt64(ptr, 1)
}

func (d *DynamicAlignedCounter) Load() int64 {
	ptr := d.getAlignedPointer()
	return atomic.LoadInt64(ptr)
}

func (d *DynamicAlignedCounter) Store(val int64) {
	ptr := d.getAlignedPointer()
	atomic.StoreInt64(ptr, val)
}

// MultiCounterArray demonstrates array-based false sharing
type MultiCounterArray struct {
	counters [8]int64 // Likely to cause false sharing
}

func NewMultiCounterArray() *MultiCounterArray {
	return &MultiCounterArray{}
}

func (m *MultiCounterArray) Increment(index int) {
	if index >= 0 && index < len(m.counters) {
		atomic.AddInt64(&m.counters[index], 1)
	}
}

func (m *MultiCounterArray) Load(index int) int64 {
	if index >= 0 && index < len(m.counters) {
		return atomic.LoadInt64(&m.counters[index])
	}
	return 0
}

// PaddedMultiCounterArray prevents false sharing in arrays
type PaddedCounter struct {
	value int64
	_     [CacheLineSize/8 - 1]int64 // Padding
}

type PaddedMultiCounterArray struct {
	counters [8]PaddedCounter
}

func NewPaddedMultiCounterArray() *PaddedMultiCounterArray {
	return &PaddedMultiCounterArray{}
}

func (p *PaddedMultiCounterArray) Increment(index int) {
	if index >= 0 && index < len(p.counters) {
		atomic.AddInt64(&p.counters[index].value, 1)
	}
}

func (p *PaddedMultiCounterArray) Load(index int) int64 {
	if index >= 0 && index < len(p.counters) {
		return atomic.LoadInt64(&p.counters[index].value)
	}
	return 0
}