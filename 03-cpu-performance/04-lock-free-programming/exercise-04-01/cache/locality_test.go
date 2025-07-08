package cache

import (
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// TestCacheLocality tests cache locality effects
func TestCacheLocality(t *testing.T) {
	// Test sequential vs random access patterns
	size := 1024 * 1024 // 1M elements
	data := make([]int64, size)
	
	// Initialize data
	for i := range data {
		data[i] = int64(i)
	}
	
	// Sequential access
	start := time.Now()
	for i := 0; i < size; i++ {
		atomic.AddInt64(&data[i], 1)
	}
	sequentialTime := time.Since(start)
	
	// Random access (stride pattern)
	stride := 64 // Cache line size in bytes / 8 bytes per int64
	start = time.Now()
	for i := 0; i < size; i += stride {
		atomic.AddInt64(&data[i], 1)
	}
	stridedTime := time.Since(start)
	
	t.Logf("Sequential access time: %v", sequentialTime)
	t.Logf("Strided access time: %v", stridedTime)
	
	// Sequential should be faster due to better cache locality
	if sequentialTime > stridedTime*2 {
		t.Logf("Warning: Sequential access unexpectedly slow")
	}
}

// TestFalseSharingImpact tests the impact of false sharing
func TestFalseSharingImpact(t *testing.T) {
	// Test false sharing vs cache-aligned counters
	iterations := 1000000
	
	// False sharing test
	falseSharingCounters := NewFalseSharingExample()
	start := time.Now()
	for i := 0; i < iterations; i++ {
		if i%2 == 0 {
			falseSharingCounters.IncrementCounter1()
		} else {
			falseSharingCounters.IncrementCounter2()
		}
	}
	falseSharingTime := time.Since(start)
	
	// Cache-aligned test
	alignedCounters := NewCacheAlignedCounters()
	start = time.Now()
	for i := 0; i < iterations; i++ {
		if i%2 == 0 {
			alignedCounters.IncrementCounter1()
		} else {
			alignedCounters.IncrementCounter2()
		}
	}
	alignedTime := time.Since(start)
	
	t.Logf("False sharing time: %v", falseSharingTime)
	t.Logf("Cache aligned time: %v", alignedTime)
	
	// Verify counters
	c1, c2 := falseSharingCounters.GetCounters()
	if c1+c2 != int64(iterations) {
		t.Errorf("False sharing counters: expected %d, got %d", iterations, c1+c2)
	}
	
	c1, c2 = alignedCounters.GetCounters()
	if c1+c2 != int64(iterations) {
		t.Errorf("Aligned counters: expected %d, got %d", iterations, c1+c2)
	}
}

// TestCacheLineAlignment tests cache line alignment detection
func TestCacheLineAlignment(t *testing.T) {
	checker := NewAlignmentChecker()
	
	// Test aligned data
	alignedData := NewAlignedData()
	if !checker.IsCacheLineAligned(unsafe.Pointer(alignedData)) {
		t.Logf("AlignedData is not cache line aligned (this may be expected)")
	}
	
	// Test dynamic alignment
	dynamicCounter := NewDynamicAlignedCounter()
	alignedPtr := dynamicCounter.getAlignedPointer()
	if !checker.IsCacheLineAligned(unsafe.Pointer(alignedPtr)) {
		t.Errorf("DynamicAlignedCounter pointer should be cache line aligned")
	}
	
	// Test aligned array
	alignedArray := NewAlignedInt64Array(100)
	baseAddr := alignedArray.GetBaseAddress()
	if baseAddr != nil && !checker.IsCacheLineAligned(baseAddr) {
		t.Errorf("AlignedInt64Array should be cache line aligned")
	}
}

// TestNUMALocality tests NUMA-aware operations
func TestNUMALocality(t *testing.T) {
	numaCounter := NewNUMACounter()
	iterations := 100000
	
	// Perform operations
	for i := 0; i < iterations; i++ {
		numaCounter.Increment()
	}
	
	// Verify total count
	total := numaCounter.Load()
	if total != int64(iterations) {
		t.Errorf("NUMA counter: expected %d, got %d", iterations, total)
	}
	
	// Check distribution across nodes
	numNodes := numaCounter.GetNumNodes()
	t.Logf("NUMA nodes: %d", numNodes)
	for i := 0; i < numNodes; i++ {
		nodeCount := numaCounter.LoadNode(i)
		t.Logf("Node %d count: %d", i, nodeCount)
	}
}

// BenchmarkCacheLocality benchmarks different access patterns
func BenchmarkSequentialAccess(b *testing.B) {
	size := 1024
	data := make([]int64, size)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < size; j++ {
			atomic.AddInt64(&data[j], 1)
		}
	}
}

func BenchmarkStridedAccess(b *testing.B) {
	size := 1024
	data := make([]int64, size)
	stride := 8 // Every 8th element
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < size; j += stride {
			atomic.AddInt64(&data[j], 1)
		}
	}
}

func BenchmarkRandomAccess(b *testing.B) {
	size := 1024
	data := make([]int64, size)
	
	// Pre-generate random indices
	indices := make([]int, size)
	for i := range indices {
		indices[i] = (i * 17) % size // Simple pseudo-random
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, idx := range indices {
			atomic.AddInt64(&data[idx], 1)
		}
	}
}