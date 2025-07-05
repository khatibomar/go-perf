package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// CustomAllocator provides custom memory allocation strategies
type CustomAllocator interface {
	Allocate(size uintptr) unsafe.Pointer
	Deallocate(ptr unsafe.Pointer)
	Stats() AllocatorStats
	Reset()
}

// AllocatorStats contains allocator statistics
type AllocatorStats struct {
	TotalAllocations   int64
	TotalDeallocations int64
	BytesAllocated     int64
	BytesInUse         int64
	PoolHits           int64
	PoolMisses         int64
	Fragmentation      float64
}

// FixedSizePool implements a pool for fixed-size allocations
type FixedSizePool struct {
	mu        sync.Mutex
	blockSize uintptr
	freeList  []unsafe.Pointer
	chunks    [][]byte
	chunkSize int
	maxBlocks int
	stats     AllocatorStats
}

// NewFixedSizePool creates a new fixed-size memory pool
func NewFixedSizePool(blockSize uintptr, initialBlocks int) *FixedSizePool {
	pool := &FixedSizePool{
		blockSize: blockSize,
		chunkSize: initialBlocks,
		maxBlocks: initialBlocks,
		freeList:  make([]unsafe.Pointer, 0, initialBlocks),
	}
	
	// Pre-allocate initial chunk
	pool.allocateChunk()
	
	return pool
}

// allocateChunk allocates a new chunk of memory
func (p *FixedSizePool) allocateChunk() {
	chunkBytes := int(p.blockSize) * p.chunkSize
	chunk := make([]byte, chunkBytes)
	p.chunks = append(p.chunks, chunk)
	
	// Add blocks to free list
	for i := 0; i < p.chunkSize; i++ {
		offset := uintptr(i) * p.blockSize
		ptr := unsafe.Pointer(&chunk[offset])
		p.freeList = append(p.freeList, ptr)
	}
}

// Allocate allocates a block from the pool
func (p *FixedSizePool) Allocate(size uintptr) unsafe.Pointer {
	if size > p.blockSize {
		return nil // Size too large for this pool
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if len(p.freeList) == 0 {
		// Check if we can allocate more blocks
		totalBlocks := len(p.chunks) * p.chunkSize
		if totalBlocks >= p.maxBlocks {
			// Pool is full, cannot allocate more
			return nil
		}
		// No free blocks, allocate new chunk
		p.allocateChunk()
		atomic.AddInt64(&p.stats.PoolMisses, 1)
	} else {
		atomic.AddInt64(&p.stats.PoolHits, 1)
	}
	
	// Get block from free list
	ptr := p.freeList[len(p.freeList)-1]
	p.freeList = p.freeList[:len(p.freeList)-1]
	
	atomic.AddInt64(&p.stats.TotalAllocations, 1)
	atomic.AddInt64(&p.stats.BytesAllocated, int64(p.blockSize))
	atomic.AddInt64(&p.stats.BytesInUse, int64(p.blockSize))
	
	return ptr
}

// Deallocate returns a block to the pool
func (p *FixedSizePool) Deallocate(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Add back to free list
	p.freeList = append(p.freeList, ptr)
	
	atomic.AddInt64(&p.stats.TotalDeallocations, 1)
	atomic.AddInt64(&p.stats.BytesInUse, -int64(p.blockSize))
}

// Stats returns allocator statistics
func (p *FixedSizePool) Stats() AllocatorStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	stats := p.stats
	
	// Calculate fragmentation
	totalChunkBytes := int64(len(p.chunks)) * int64(p.chunkSize) * int64(p.blockSize)
	if totalChunkBytes > 0 {
		stats.Fragmentation = float64(totalChunkBytes-stats.BytesInUse) / float64(totalChunkBytes) * 100
	}
	
	return stats
}

// Reset resets the pool
func (p *FixedSizePool) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.freeList = p.freeList[:0]
	p.chunks = p.chunks[:0]
	p.stats = AllocatorStats{}
	
	// Re-allocate initial chunk
	p.allocateChunk()
}

// SizeClassAllocator implements a multi-size-class allocator
type SizeClassAllocator struct {
	pools map[uintptr]*FixedSizePool
	mu    sync.RWMutex
	stats AllocatorStats
}

// NewSizeClassAllocator creates a new size-class-based allocator
func NewSizeClassAllocator() *SizeClassAllocator {
	allocator := &SizeClassAllocator{
		pools: make(map[uintptr]*FixedSizePool),
	}
	
	// Create pools for common size classes
	commonSizes := []uintptr{8, 16, 24, 32, 48, 64, 80, 96, 128, 256, 512, 1024}
	for _, size := range commonSizes {
		allocator.pools[size] = NewFixedSizePool(size, 100)
	}
	
	return allocator
}

// getSizeClass returns the appropriate size class for a given size
func (a *SizeClassAllocator) getSizeClass(size uintptr) uintptr {
	// Find the smallest size class that can accommodate the size
	sizeClasses := []uintptr{8, 16, 24, 32, 48, 64, 80, 96, 128, 256, 512, 1024}
	
	for _, class := range sizeClasses {
		if size <= class {
			return class
		}
	}
	
	// For larger sizes, round up to next power of 2
	class := uintptr(1024)
	for class < size {
		class *= 2
	}
	return class
}

// Allocate allocates memory using size classes
func (a *SizeClassAllocator) Allocate(size uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}
	
	sizeClass := a.getSizeClass(size)
	
	a.mu.RLock()
	pool, exists := a.pools[sizeClass]
	a.mu.RUnlock()
	
	if !exists {
		// Create new pool for this size class
		a.mu.Lock()
		if pool, exists = a.pools[sizeClass]; !exists {
			pool = NewFixedSizePool(sizeClass, 50)
			a.pools[sizeClass] = pool
		}
		a.mu.Unlock()
	}
	
	ptr := pool.Allocate(sizeClass)
	if ptr != nil {
		atomic.AddInt64(&a.stats.TotalAllocations, 1)
		atomic.AddInt64(&a.stats.BytesAllocated, int64(sizeClass))
		atomic.AddInt64(&a.stats.BytesInUse, int64(sizeClass))
	}
	
	return ptr
}

// Deallocate deallocates memory
func (a *SizeClassAllocator) Deallocate(ptr unsafe.Pointer) {
	// In a real implementation, we'd need to track which pool each pointer belongs to
	// For this example, we'll implement a simplified version
	if ptr == nil {
		return
	}
	
	// This is a simplified implementation - in practice, you'd need to track
	// which size class each pointer belongs to
	atomic.AddInt64(&a.stats.TotalDeallocations, 1)
}

// Stats returns allocator statistics
func (a *SizeClassAllocator) Stats() AllocatorStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	stats := a.stats
	
	// Aggregate stats from all pools
	var totalPoolHits, totalPoolMisses int64
	for _, pool := range a.pools {
		poolStats := pool.Stats()
		totalPoolHits += poolStats.PoolHits
		totalPoolMisses += poolStats.PoolMisses
	}
	
	stats.PoolHits = totalPoolHits
	stats.PoolMisses = totalPoolMisses
	
	return stats
}

// Reset resets the allocator
func (a *SizeClassAllocator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	for _, pool := range a.pools {
		pool.Reset()
	}
	a.stats = AllocatorStats{}
}

// StackAllocator implements a stack-based allocator for temporary allocations
type StackAllocator struct {
	buffer   []byte
	offset   uintptr
	markers  []uintptr
	mu       sync.Mutex
	stats    AllocatorStats
}

// NewStackAllocator creates a new stack allocator
func NewStackAllocator(size uintptr) *StackAllocator {
	return &StackAllocator{
		buffer:  make([]byte, size),
		markers: make([]uintptr, 0, 32),
	}
}

// Allocate allocates memory from the stack
func (s *StackAllocator) Allocate(size uintptr) unsafe.Pointer {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Align to 8-byte boundary
	alignedSize := (size + 7) &^ 7
	
	if s.offset+alignedSize > uintptr(len(s.buffer)) {
		return nil // Out of space
	}
	
	ptr := unsafe.Pointer(&s.buffer[s.offset])
	s.offset += alignedSize
	
	atomic.AddInt64(&s.stats.TotalAllocations, 1)
	atomic.AddInt64(&s.stats.BytesAllocated, int64(alignedSize))
	atomic.AddInt64(&s.stats.BytesInUse, int64(alignedSize))
	
	return ptr
}

// Deallocate is a no-op for stack allocator (use markers instead)
func (s *StackAllocator) Deallocate(ptr unsafe.Pointer) {
	// Stack allocator doesn't support individual deallocation
	// Use PushMarker/PopMarker instead
}

// PushMarker pushes a marker onto the stack
func (s *StackAllocator) PushMarker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.markers = append(s.markers, s.offset)
}

// PopMarker pops a marker and resets allocation to that point
func (s *StackAllocator) PopMarker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.markers) == 0 {
		return
	}
	
	// Reset to previous marker
	prevOffset := s.markers[len(s.markers)-1]
	s.markers = s.markers[:len(s.markers)-1]
	
	freedBytes := int64(s.offset - prevOffset)
	s.offset = prevOffset
	
	atomic.AddInt64(&s.stats.BytesInUse, -freedBytes)
}

// Stats returns allocator statistics
func (s *StackAllocator) Stats() AllocatorStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	stats := s.stats
	stats.Fragmentation = 0 // Stack allocator has no fragmentation
	
	return stats
}

// Reset resets the stack allocator
func (s *StackAllocator) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.offset = 0
	s.markers = s.markers[:0]
	s.stats = AllocatorStats{}
}

// PoolAllocator wraps sync.Pool with statistics
type PoolAllocator struct {
	pools map[uintptr]*sync.Pool
	mu    sync.RWMutex
	stats AllocatorStats
}

// NewPoolAllocator creates a new pool-based allocator
func NewPoolAllocator() *PoolAllocator {
	return &PoolAllocator{
		pools: make(map[uintptr]*sync.Pool),
	}
}

// getPool returns a pool for the given size
func (p *PoolAllocator) getPool(size uintptr) *sync.Pool {
	p.mu.RLock()
	pool, exists := p.pools[size]
	p.mu.RUnlock()
	
	if !exists {
		p.mu.Lock()
		if pool, exists = p.pools[size]; !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					return make([]byte, size)
				},
			}
			p.pools[size] = pool
		}
		p.mu.Unlock()
	}
	
	return pool
}

// Allocate allocates memory using sync.Pool
func (p *PoolAllocator) Allocate(size uintptr) unsafe.Pointer {
	pool := p.getPool(size)
	buf := pool.Get().([]byte)
	
	atomic.AddInt64(&p.stats.TotalAllocations, 1)
	atomic.AddInt64(&p.stats.BytesAllocated, int64(size))
	atomic.AddInt64(&p.stats.BytesInUse, int64(size))
	atomic.AddInt64(&p.stats.PoolHits, 1)
	
	return unsafe.Pointer(&buf[0])
}

// Deallocate returns memory to sync.Pool
func (p *PoolAllocator) Deallocate(ptr unsafe.Pointer) {
	// In a real implementation, we'd need to track which pool each pointer belongs to
	// This is a simplified version
	atomic.AddInt64(&p.stats.TotalDeallocations, 1)
}

// DeallocateBuffer returns a buffer to the appropriate pool
func (p *PoolAllocator) DeallocateBuffer(buf []byte) {
	size := uintptr(len(buf))
	pool := p.getPool(size)
	pool.Put(buf)
	
	atomic.AddInt64(&p.stats.TotalDeallocations, 1)
	atomic.AddInt64(&p.stats.BytesInUse, -int64(size))
}

// Stats returns allocator statistics
func (p *PoolAllocator) Stats() AllocatorStats {
	return p.stats
}

// Reset resets the pool allocator
func (p *PoolAllocator) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Clear all pools
	for _, pool := range p.pools {
		// sync.Pool doesn't have a clear method, so we recreate
		*pool = sync.Pool{}
	}
	
	p.stats = AllocatorStats{}
}

// AllocatorBenchmark benchmarks different allocators
type AllocatorBenchmark struct {
	allocators map[string]CustomAllocator
}

// NewAllocatorBenchmark creates a new allocator benchmark
func NewAllocatorBenchmark() *AllocatorBenchmark {
	return &AllocatorBenchmark{
		allocators: map[string]CustomAllocator{
			"FixedSize32":    NewFixedSizePool(32, 1000),
			"FixedSize64":    NewFixedSizePool(64, 1000),
			"FixedSize128":   NewFixedSizePool(128, 1000),
			"SizeClass":      NewSizeClassAllocator(),
			"Stack":          NewStackAllocator(1024 * 1024), // 1MB stack
			"Pool":           NewPoolAllocator(),
		},
	}
}

// BenchmarkAllocators benchmarks all allocators
func (ab *AllocatorBenchmark) BenchmarkAllocators() {
	fmt.Println("=== Custom Allocator Benchmark ===")
	
	// Test different allocation patterns
	patterns := []struct {
		name      string
		size      uintptr
		count     int
		useMarkers bool
	}{
		{"Small (32B)", 32, 10000, false},
		{"Medium (128B)", 128, 5000, false},
		{"Large (512B)", 512, 1000, false},
		{"Mixed Sizes", 0, 1000, false}, // Special case for mixed
		{"Stack Pattern", 64, 1000, true}, // Use markers
	}
	
	for _, pattern := range patterns {
		fmt.Printf("\n--- %s Pattern ---\n", pattern.name)
		fmt.Printf("%-15s %-12s %-12s %-12s %-12s %-10s\n", 
			"Allocator", "Allocs/sec", "Memory MB", "Pool Hit%", "Frag%", "Time ms")
		fmt.Println("--------------------------------------------------------------------------------")
		
		for name, allocator := range ab.allocators {
			// Skip stack allocator for non-marker patterns
			if name == "Stack" && !pattern.useMarkers {
				continue
			}
			
			// Skip non-stack allocators for marker patterns
			if name != "Stack" && pattern.useMarkers {
				continue
			}
			
			allocator.Reset()
			
			startTime := time.Now()
			
			if pattern.name == "Mixed Sizes" {
				ab.runMixedPattern(allocator, pattern.count)
			} else if pattern.useMarkers {
				ab.runStackPattern(allocator, pattern.size, pattern.count)
			} else {
				ab.runFixedPattern(allocator, pattern.size, pattern.count)
			}
			
			duration := time.Since(startTime)
			stats := allocator.Stats()
			
			// Calculate metrics
			allocsPerSec := float64(stats.TotalAllocations) / duration.Seconds()
			memoryMB := float64(stats.BytesAllocated) / 1024 / 1024
			
			poolHitRate := float64(0)
			if stats.PoolHits+stats.PoolMisses > 0 {
				poolHitRate = float64(stats.PoolHits) / float64(stats.PoolHits+stats.PoolMisses) * 100
			}
			
			fmt.Printf("%-15s %-12.0f %-12.2f %-12.1f %-12.1f %-10.1f\n",
				name, allocsPerSec, memoryMB, poolHitRate, stats.Fragmentation, 
				float64(duration.Nanoseconds())/1e6)
		}
	}
}

// runFixedPattern runs a fixed-size allocation pattern
func (ab *AllocatorBenchmark) runFixedPattern(allocator CustomAllocator, size uintptr, count int) {
	var ptrs []unsafe.Pointer
	
	// Allocate
	for i := 0; i < count; i++ {
		ptr := allocator.Allocate(size)
		if ptr != nil {
			ptrs = append(ptrs, ptr)
		}
	}
	
	// Deallocate half
	for i := 0; i < len(ptrs)/2; i++ {
		allocator.Deallocate(ptrs[i])
	}
	
	// Allocate again
	for i := 0; i < count/2; i++ {
		ptr := allocator.Allocate(size)
		if ptr != nil {
			ptrs = append(ptrs, ptr)
		}
	}
}

// runMixedPattern runs a mixed-size allocation pattern
func (ab *AllocatorBenchmark) runMixedPattern(allocator CustomAllocator, count int) {
	sizes := []uintptr{16, 32, 64, 128, 256, 512}
	var ptrs []unsafe.Pointer
	
	for i := 0; i < count; i++ {
		size := sizes[i%len(sizes)]
		ptr := allocator.Allocate(size)
		if ptr != nil {
			ptrs = append(ptrs, ptr)
		}
		
		// Occasionally deallocate
		if i%10 == 0 && len(ptrs) > 0 {
			allocator.Deallocate(ptrs[0])
			ptrs = ptrs[1:]
		}
	}
}

// runStackPattern runs a stack allocation pattern with markers
func (ab *AllocatorBenchmark) runStackPattern(allocator CustomAllocator, size uintptr, count int) {
	if stackAlloc, ok := allocator.(*StackAllocator); ok {
		for i := 0; i < count/10; i++ {
			stackAlloc.PushMarker()
			
			// Allocate several objects
			for j := 0; j < 10; j++ {
				stackAlloc.Allocate(size)
			}
			
			// Pop marker to free all at once
			stackAlloc.PopMarker()
		}
	}
}

// CompareWithStandardAllocator compares custom allocators with standard Go allocator
func (ab *AllocatorBenchmark) CompareWithStandardAllocator() {
	fmt.Println("\n=== Comparison with Standard Go Allocator ===")
	
	patterns := []struct {
		name  string
		size  uintptr
		count int
	}{
		{"Small (32B)", 32, 10000},
		{"Medium (128B)", 128, 5000},
		{"Large (512B)", 512, 1000},
	}
	
	for _, pattern := range patterns {
		fmt.Printf("\n--- %s ---\n", pattern.name)
		
		// Benchmark standard allocator
		startTime := time.Now()
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		var ptrs [][]byte
		for i := 0; i < pattern.count; i++ {
			buf := make([]byte, pattern.size)
			ptrs = append(ptrs, buf)
		}
		
		runtime.ReadMemStats(&m2)
		standardDuration := time.Since(startTime)
		_ = m2.Mallocs - m1.Mallocs // standardAllocs not used
		standardBytes := m2.TotalAlloc - m1.TotalAlloc
		
		// Benchmark custom allocators
		fmt.Printf("%-15s %-12s %-12s %-12s %-12s\n", "Allocator", "Time (ms)", "Speedup", "Memory", "Allocs")
		fmt.Println("---------------------------------------------------------------")
		
		fmt.Printf("%-15s %-12.1f %-12s %-12d %-12s\n", "Standard Go", 
			float64(standardDuration.Nanoseconds())/1e6, "1.0x", standardBytes, "N/A")
		
		for name, allocator := range ab.allocators {
			if name == "Stack" {
				continue // Skip stack for this comparison
			}
			
			allocator.Reset()
			
			startTime := time.Now()
			ab.runFixedPattern(allocator, pattern.size, pattern.count)
			customDuration := time.Since(startTime)
			
			stats := allocator.Stats()
			speedup := float64(standardDuration) / float64(customDuration)
			
			fmt.Printf("%-15s %-12.1f %-12.1fx %-12d %-12d\n", name,
				float64(customDuration.Nanoseconds())/1e6, speedup, stats.BytesAllocated, stats.TotalAllocations)
		}
		
		// Keep references to prevent GC
		_ = ptrs
	}
}

func main() {
	fmt.Println("Custom Memory Pool Implementation")
	fmt.Println("=================================")
	
	// Create and run benchmark
	benchmark := NewAllocatorBenchmark()
	
	// Benchmark all allocators
	benchmark.BenchmarkAllocators()
	
	// Compare with standard allocator
	benchmark.CompareWithStandardAllocator()
	
	// Demonstrate specific allocator features
	fmt.Println("\n=== Allocator Feature Demonstrations ===")
	
	// Fixed-size pool demonstration
	fmt.Println("\n--- Fixed-Size Pool ---")
	pool := NewFixedSizePool(64, 10)
	for i := 0; i < 15; i++ {
		ptr := pool.Allocate(64)
		if ptr == nil {
			fmt.Printf("Allocation %d failed (expected after 10)\n", i+1)
			break
		}
		fmt.Printf("Allocation %d: %p\n", i+1, ptr)
	}
	stats := pool.Stats()
	fmt.Printf("Pool stats: %d hits, %d misses, %.1f%% fragmentation\n", 
		stats.PoolHits, stats.PoolMisses, stats.Fragmentation)
	
	// Stack allocator demonstration
	fmt.Println("\n--- Stack Allocator ---")
	stackAlloc := NewStackAllocator(1024)
	
	stackAlloc.PushMarker()
	ptr1 := stackAlloc.Allocate(100)
	ptr2 := stackAlloc.Allocate(200)
	fmt.Printf("Allocated: %p, %p\n", ptr1, ptr2)
	
	stackAlloc.PushMarker()
	ptr3 := stackAlloc.Allocate(50)
	fmt.Printf("Allocated: %p\n", ptr3)
	
	stackAlloc.PopMarker() // Frees ptr3
	fmt.Println("Popped marker (freed ptr3)")
	
	ptr4 := stackAlloc.Allocate(75)
	fmt.Printf("Allocated: %p (reused space)\n", ptr4)
	
	stackAlloc.PopMarker() // Frees ptr1, ptr2, ptr4
	fmt.Println("Popped marker (freed all)")
	
	stackStats := stackAlloc.Stats()
	fmt.Printf("Stack stats: %d allocations, %d bytes in use\n", 
		stackStats.TotalAllocations, stackStats.BytesInUse)
	
	fmt.Println("\n=== Summary ===")
	fmt.Println("Custom memory pools can provide significant performance benefits:")
	fmt.Println("1. Fixed-size pools eliminate allocation overhead for common sizes")
	fmt.Println("2. Size-class allocators reduce fragmentation")
	fmt.Println("3. Stack allocators are perfect for temporary allocations")
	fmt.Println("4. Pool allocators leverage Go's sync.Pool for automatic management")
	fmt.Println("5. Choose the right allocator based on your allocation patterns")
}