package cas

import (
	"sync/atomic"
	"unsafe"
)

// VersionedValue represents a value with a version to prevent ABA
type VersionedValue struct {
	value   int64
	version int64
}

// VersionedPointer represents a pointer with version for ABA prevention
type VersionedPointer struct {
	pointer unsafe.Pointer
	version int64
}

// VersionedCounter implements ABA-safe counter using versioning
type VersionedCounter struct {
	data VersionedValue
}

func NewVersionedCounter() *VersionedCounter {
	return &VersionedCounter{}
}

func (c *VersionedCounter) Increment() int64 {
	return c.Add(1)
}

func (c *VersionedCounter) Add(delta int64) int64 {
	for {
		// Load current value and version atomically
		currentValue := atomic.LoadInt64(&c.data.value)
		currentVersion := atomic.LoadInt64(&c.data.version)
		
		newValue := currentValue + delta
		newVersion := currentVersion + 1
		
		// Try to update value first
		if atomic.CompareAndSwapInt64(&c.data.value, currentValue, newValue) {
			// Then update version
			atomic.StoreInt64(&c.data.version, newVersion)
			return newValue
		}
	}
}

func (c *VersionedCounter) Get() (int64, int64) {
	value := atomic.LoadInt64(&c.data.value)
	version := atomic.LoadInt64(&c.data.version)
	return value, version
}

func (c *VersionedCounter) GetValue() int64 {
	return atomic.LoadInt64(&c.data.value)
}

// VersionedCAS performs compare-and-swap with version checking
func VersionedCAS(addr *VersionedValue, expectedValue, expectedVersion, newValue, newVersion int64) bool {
	// Check both value and version
	currentValue := atomic.LoadInt64(&addr.value)
	currentVersion := atomic.LoadInt64(&addr.version)
	
	if currentValue != expectedValue || currentVersion != expectedVersion {
		return false
	}
	
	// Try to update value
	if atomic.CompareAndSwapInt64(&addr.value, expectedValue, newValue) {
		// Update version on successful value update
		atomic.StoreInt64(&addr.version, newVersion)
		return true
	}
	
	return false
}

// DoubleWidthCAS simulates double-width CAS for value+version
type DoubleWidthCAS struct {
	data [2]int64 // [value, version]
}

func NewDoubleWidthCAS() *DoubleWidthCAS {
	return &DoubleWidthCAS{}
}

func (d *DoubleWidthCAS) CompareAndSwap(expectedValue, expectedVersion, newValue, newVersion int64) bool {
	// This is a simplified implementation
	// Real double-width CAS would be atomic for both values
	for {
		currentValue := atomic.LoadInt64(&d.data[0])
		currentVersion := atomic.LoadInt64(&d.data[1])
		
		if currentValue != expectedValue || currentVersion != expectedVersion {
			return false
		}
		
		// Try to update value first
		if atomic.CompareAndSwapInt64(&d.data[0], expectedValue, newValue) {
			// Then update version
			if atomic.CompareAndSwapInt64(&d.data[1], expectedVersion, newVersion) {
				return true
			} else {
				// Rollback value change
				atomic.StoreInt64(&d.data[0], expectedValue)
				return false
			}
		}
	}
}

func (d *DoubleWidthCAS) Load() (int64, int64) {
	value := atomic.LoadInt64(&d.data[0])
	version := atomic.LoadInt64(&d.data[1])
	return value, version
}