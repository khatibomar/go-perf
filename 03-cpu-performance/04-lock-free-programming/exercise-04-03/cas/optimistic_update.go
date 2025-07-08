package cas

import (
	"sync/atomic"
	"runtime"
)

// OptimisticCounter implements an optimistic counter
type OptimisticCounter struct {
	value int64
}

func NewOptimisticCounter() *OptimisticCounter {
	return &OptimisticCounter{}
}

func (c *OptimisticCounter) Increment() int64 {
	for {
		old := atomic.LoadInt64(&c.value)
		new := old + 1
		if atomic.CompareAndSwapInt64(&c.value, old, new) {
			return new
		}
		runtime.Gosched()
	}
}

func (c *OptimisticCounter) Add(delta int64) int64 {
	for {
		old := atomic.LoadInt64(&c.value)
		new := old + delta
		if atomic.CompareAndSwapInt64(&c.value, old, new) {
			return new
		}
		runtime.Gosched()
	}
}

func (c *OptimisticCounter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// OptimisticMap implements a simple optimistic map operation
type OptimisticMap struct {
	data map[string]int64
	version int64
}

func NewOptimisticMap() *OptimisticMap {
	return &OptimisticMap{
		data: make(map[string]int64),
	}
}

func (m *OptimisticMap) Update(key string, updateFunc func(int64) int64) bool {
	for {
		version := atomic.LoadInt64(&m.version)
		oldValue := m.data[key]
		newValue := updateFunc(oldValue)
		
		// Check if version changed during computation
		if atomic.LoadInt64(&m.version) != version {
			continue
		}
		
		// Try to update with version increment
		if atomic.CompareAndSwapInt64(&m.version, version, version+1) {
			m.data[key] = newValue
			return true
		}
	}
}

func (m *OptimisticMap) Get(key string) int64 {
	return m.data[key]
}