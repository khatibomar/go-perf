package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// CustomAllocatorSuite provides a comprehensive set of custom allocators
type CustomAllocatorSuite struct {
	stackAllocators  map[string]*StackAllocator
	poolAllocators   map[string]*PoolAllocator
	arenaAllocators  map[string]*ArenaAllocator
	linearAllocators map[string]*LinearAllocator
	freeListAllocators map[string]*FreeListAllocator
	slabAllocators   map[string]*SlabAllocator
	stats            *AllocatorSuiteStats
	mu               sync.RWMutex
}

// AllocatorSuiteStats tracks overall allocator performance
type AllocatorSuiteStats struct {
	TotalAllocations   uint64
	TotalDeallocations uint64
	TotalBytesAllocated uint64
	TotalBytesDeallocated uint64
	AverageFragmentation float64
	AverageEfficiency   float64
	GCPressureReduction float64
	PerformanceGain     float64
}

// LinearAllocator provides simple linear allocation
type LinearAllocator struct {
	buffer    []byte
	offset    uintptr
	size      uintptr
	alignment uintptr
	stats     *AllocatorStats
	mu        sync.Mutex
}

// FreeListAllocator manages free blocks for reuse
type FreeListAllocator struct {
	freeBlocks map[uintptr]*FreeBlock
	allBlocks  []*AllocatedBlock
	blockSize  uintptr
	stats      *AllocatorStats
	mu         sync.RWMutex
}

// FreeBlock represents a free memory block
type FreeBlock struct {
	size uintptr
	next *FreeBlock
	prev *FreeBlock
}

// AllocatedBlock represents an allocated memory block
type AllocatedBlock struct {
	ptr       unsafe.Pointer
	size      uintptr
	allocated bool
	timestamp time.Time
}

// SlabAllocator provides slab-based allocation for fixed-size objects
type SlabAllocator struct {
	slabs     []*Slab
	objectSize uintptr
	slabSize  uintptr
	stats     *AllocatorStats
	mu        sync.Mutex
}

// Slab represents a memory slab
type Slab struct {
	buffer    []byte
	freeList  []unsafe.Pointer
	freeCount int
	totalCount int
}

// MemoryPool provides pooled memory management
type MemoryPool struct {
	pools     map[int]*sync.Pool
	sizeClasses []int
	stats     *PoolStats
	mu        sync.RWMutex
}

// AdvancedBufferPool provides sophisticated buffer management
type AdvancedBufferPool struct {
	smallBuffers  *sync.Pool // < 1KB
	mediumBuffers *sync.Pool // 1KB - 64KB
	largeBuffers  *sync.Pool // > 64KB
	customSizes   map[int]*sync.Pool
	stats         *BufferPoolStats
	mu            sync.RWMutex
}

// BufferPoolStats tracks buffer pool performance
type BufferPoolStats struct {
	SmallHits    uint64
	MediumHits   uint64
	LargeHits    uint64
	CustomHits   uint64
	Misses       uint64
	TotalReused  uint64
	MemorySaved  uint64
	HitRate      float64
}

// ObjectFactory creates objects for pooling
type ObjectFactory interface {
	Create() interface{}
	Reset(obj interface{})
	Validate(obj interface{}) bool
}

// TypedObjectPool provides type-safe object pooling
type TypedObjectPool struct {
	pool    sync.Pool
	factory ObjectFactory
	stats   *TypedPoolStats
	mu      sync.RWMutex
}

// TypedPoolStats tracks typed pool performance
type TypedPoolStats struct {
	Gets        uint64
	Puts        uint64
	News        uint64
	Resets      uint64
	Validations uint64
	HitRate     float64
	TypeErrors  uint64
}

// GCFreeDataStructures provides GC-free data structure implementations
type GCFreeDataStructures struct {
	arrays    map[string]*GCFreeArray
	slices    map[string]*GCFreeSlice
	maps      map[string]*GCFreeMap
	strings   map[string]*GCFreeString
	mu        sync.RWMutex
}

// GCFreeArray provides a GC-free array implementation
type GCFreeArray struct {
	data     unsafe.Pointer
	length   int
	capacity int
	elemSize uintptr
	allocator Allocator
}

// GCFreeSlice provides a GC-free slice implementation
type GCFreeSlice struct {
	data     unsafe.Pointer
	length   int
	capacity int
	elemSize uintptr
	allocator Allocator
}

// GCFreeMap provides a GC-free map implementation
type GCFreeMap struct {
	buckets   unsafe.Pointer
	bucketCount int
	size      int
	keySize   uintptr
	valueSize uintptr
	allocator Allocator
}

// GCFreeString provides a GC-free string implementation
type GCFreeString struct {
	data      unsafe.Pointer
	length    int
	capacity  int
	allocator Allocator
}

// NewCustomAllocatorSuite creates a new allocator suite
func NewCustomAllocatorSuite() *CustomAllocatorSuite {
	suite := &CustomAllocatorSuite{
		stackAllocators:    make(map[string]*StackAllocator),
		poolAllocators:     make(map[string]*PoolAllocator),
		arenaAllocators:    make(map[string]*ArenaAllocator),
		linearAllocators:   make(map[string]*LinearAllocator),
		freeListAllocators: make(map[string]*FreeListAllocator),
		slabAllocators:     make(map[string]*SlabAllocator),
		stats:              &AllocatorSuiteStats{},
	}
	
	// Initialize baseline metrics
	suite.updateStats()
	return suite
}

// CreateStackAllocator creates a new stack allocator
func (s *CustomAllocatorSuite) CreateStackAllocator(name string, size int) *StackAllocator {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	allocator := NewStackAllocator(size)
	s.stackAllocators[name] = allocator
	return allocator
}

// CreateLinearAllocator creates a new linear allocator
func (s *CustomAllocatorSuite) CreateLinearAllocator(name string, size int, alignment uintptr) *LinearAllocator {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	allocator := NewLinearAllocator(size, alignment)
	s.linearAllocators[name] = allocator
	return allocator
}

// CreateArenaAllocator creates a new arena allocator
func (s *CustomAllocatorSuite) CreateArenaAllocator(name string, arenaSize int) *ArenaAllocator {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	allocator := NewArenaAllocator(arenaSize)
	s.arenaAllocators[name] = allocator
	return allocator
}

// CreateSlabAllocator creates a new slab allocator
func (s *CustomAllocatorSuite) CreateSlabAllocator(name string, objectSize, slabSize uintptr) *SlabAllocator {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	allocator := NewSlabAllocator(objectSize, slabSize)
	s.slabAllocators[name] = allocator
	return allocator
}

// GetAllocatorStats returns comprehensive allocator statistics
func (s *CustomAllocatorSuite) GetAllocatorStats() *AllocatorSuiteStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	totalAllocations := uint64(0)
	totalDeallocations := uint64(0)
	totalBytesAllocated := uint64(0)
	totalBytesDeallocated := uint64(0)
	totalFragmentation := 0.0
	totalEfficiency := 0.0
	allocatorCount := 0
	
	// Aggregate stats from all allocators
	for _, allocator := range s.stackAllocators {
		stats := allocator.Stats()
		totalAllocations += stats.Allocations
		totalDeallocations += stats.Deallocations
		totalBytesAllocated += stats.BytesAllocated
		totalBytesDeallocated += stats.BytesDeallocated
		totalFragmentation += stats.Fragmentation
		totalEfficiency += stats.Efficiency
		allocatorCount++
	}
	
	for _, allocator := range s.linearAllocators {
		stats := allocator.Stats()
		totalAllocations += stats.Allocations
		totalDeallocations += stats.Deallocations
		totalBytesAllocated += stats.BytesAllocated
		totalBytesDeallocated += stats.BytesDeallocated
		totalFragmentation += stats.Fragmentation
		totalEfficiency += stats.Efficiency
		allocatorCount++
	}
	
	for _, allocator := range s.arenaAllocators {
		stats := allocator.Stats()
		totalAllocations += stats.Allocations
		totalDeallocations += stats.Deallocations
		totalBytesAllocated += stats.BytesAllocated
		totalBytesDeallocated += stats.BytesDeallocated
		totalFragmentation += stats.Fragmentation
		totalEfficiency += stats.Efficiency
		allocatorCount++
	}
	
	for _, allocator := range s.slabAllocators {
		stats := allocator.Stats()
		totalAllocations += stats.Allocations
		totalDeallocations += stats.Deallocations
		totalBytesAllocated += stats.BytesAllocated
		totalBytesDeallocated += stats.BytesDeallocated
		totalFragmentation += stats.Fragmentation
		totalEfficiency += stats.Efficiency
		allocatorCount++
	}
	
	averageFragmentation := 0.0
	averageEfficiency := 0.0
	if allocatorCount > 0 {
		averageFragmentation = totalFragmentation / float64(allocatorCount)
		averageEfficiency = totalEfficiency / float64(allocatorCount)
	}
	
	// Calculate GC pressure reduction (estimated)
	gcPressureReduction := 0.0
	if totalBytesAllocated > 0 {
		gcPressureReduction = float64(totalBytesAllocated) / (1024 * 1024) * 0.1 // Rough estimate
	}
	
	// Calculate performance gain (estimated)
	performanceGain := averageEfficiency * 10.0 // Rough estimate
	
	s.stats.TotalAllocations = totalAllocations
	s.stats.TotalDeallocations = totalDeallocations
	s.stats.TotalBytesAllocated = totalBytesAllocated
	s.stats.TotalBytesDeallocated = totalBytesDeallocated
	s.stats.AverageFragmentation = averageFragmentation
	s.stats.AverageEfficiency = averageEfficiency
	s.stats.GCPressureReduction = gcPressureReduction
	s.stats.PerformanceGain = performanceGain
	
	return s.stats
}

// LinearAllocator implementation

// NewLinearAllocator creates a new linear allocator
func NewLinearAllocator(size int, alignment uintptr) *LinearAllocator {
	if alignment == 0 {
		alignment = 8 // Default alignment
	}
	
	return &LinearAllocator{
		buffer:    make([]byte, size),
		offset:    0,
		size:      uintptr(size),
		alignment: alignment,
		stats:     &AllocatorStats{},
	}
}

// Allocate allocates aligned memory from the linear allocator
func (l *LinearAllocator) Allocate(size int) unsafe.Pointer {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Align the offset
	alignedOffset := (l.offset + l.alignment - 1) & ^(l.alignment - 1)
	
	if alignedOffset+uintptr(size) > l.size {
		return nil // Out of space
	}
	
	ptr := unsafe.Pointer(&l.buffer[alignedOffset])
	l.offset = alignedOffset + uintptr(size)
	
	atomic.AddUint64(&l.stats.Allocations, 1)
	atomic.AddUint64(&l.stats.BytesAllocated, uint64(size))
	
	return ptr
}

// Deallocate is a no-op for linear allocator
func (l *LinearAllocator) Deallocate(ptr unsafe.Pointer) {
	atomic.AddUint64(&l.stats.Deallocations, 1)
}

// Reset resets the linear allocator
func (l *LinearAllocator) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.offset = 0
}

// Stats returns allocator statistics
func (l *LinearAllocator) Stats() *AllocatorStats {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	efficiency := 0.0
	if l.stats.BytesAllocated > 0 {
		efficiency = float64(l.offset) / float64(l.stats.BytesAllocated)
	}
	
	fragmentation := 0.0
	if l.size > 0 {
		fragmentation = float64(l.size-l.offset) / float64(l.size)
	}
	
	l.stats.Efficiency = efficiency
	l.stats.Fragmentation = fragmentation
	
	return l.stats
}

// ArenaAllocator implementation

// NewArenaAllocator creates a new arena allocator
func NewArenaAllocator(arenaSize int) *ArenaAllocator {
	return &ArenaAllocator{
		arenas:    make([]*Arena, 0),
		current:   nil,
		arenaSize: arenaSize,
		stats:     &AllocatorStats{},
	}
}

// Allocate allocates memory from the arena allocator
func (a *ArenaAllocator) Allocate(size int) unsafe.Pointer {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Check if current arena has enough space
	if a.current == nil || a.current.offset+size > a.current.size {
		// Create new arena
		arenaSize := a.arenaSize
		if size > arenaSize {
			arenaSize = size * 2 // Ensure we can fit the allocation
		}
		
		newArena := &Arena{
			buffer: make([]byte, arenaSize),
			offset: 0,
			size:   arenaSize,
		}
		
		a.arenas = append(a.arenas, newArena)
		a.current = newArena
	}
	
	ptr := unsafe.Pointer(&a.current.buffer[a.current.offset])
	a.current.offset += size
	
	atomic.AddUint64(&a.stats.Allocations, 1)
	atomic.AddUint64(&a.stats.BytesAllocated, uint64(size))
	
	return ptr
}

// Deallocate is a no-op for arena allocator
func (a *ArenaAllocator) Deallocate(ptr unsafe.Pointer) {
	atomic.AddUint64(&a.stats.Deallocations, 1)
}

// Reset resets all arenas
func (a *ArenaAllocator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	for _, arena := range a.arenas {
		arena.offset = 0
	}
	
	if len(a.arenas) > 0 {
		a.current = a.arenas[0]
	}
}

// Stats returns allocator statistics
func (a *ArenaAllocator) Stats() *AllocatorStats {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	totalUsed := 0
	totalSize := 0
	
	for _, arena := range a.arenas {
		totalUsed += arena.offset
		totalSize += arena.size
	}
	
	efficiency := 0.0
	if a.stats.BytesAllocated > 0 {
		efficiency = float64(totalUsed) / float64(a.stats.BytesAllocated)
	}
	
	fragmentation := 0.0
	if totalSize > 0 {
		fragmentation = float64(totalSize-totalUsed) / float64(totalSize)
	}
	
	a.stats.Efficiency = efficiency
	a.stats.Fragmentation = fragmentation
	
	return a.stats
}

// SlabAllocator implementation

// NewSlabAllocator creates a new slab allocator
func NewSlabAllocator(objectSize, slabSize uintptr) *SlabAllocator {
	return &SlabAllocator{
		slabs:      make([]*Slab, 0),
		objectSize: objectSize,
		slabSize:   slabSize,
		stats:      &AllocatorStats{},
	}
}

// Allocate allocates a fixed-size object from the slab allocator
func (s *SlabAllocator) Allocate(size int) unsafe.Pointer {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if uintptr(size) != s.objectSize {
		return nil // Size mismatch
	}
	
	// Find a slab with free objects
	for _, slab := range s.slabs {
		if slab.freeCount > 0 {
			ptr := slab.freeList[slab.freeCount-1]
			slab.freeCount--
			
			atomic.AddUint64(&s.stats.Allocations, 1)
			atomic.AddUint64(&s.stats.BytesAllocated, uint64(size))
			
			return ptr
		}
	}
	
	// Create new slab
	newSlab := s.createSlab()
	if newSlab == nil {
		return nil
	}
	
	s.slabs = append(s.slabs, newSlab)
	
	// Allocate from new slab
	ptr := newSlab.freeList[newSlab.freeCount-1]
	newSlab.freeCount--
	
	atomic.AddUint64(&s.stats.Allocations, 1)
	atomic.AddUint64(&s.stats.BytesAllocated, uint64(size))
	
	return ptr
}

// createSlab creates a new slab
func (s *SlabAllocator) createSlab() *Slab {
	objectsPerSlab := int(s.slabSize / s.objectSize)
	if objectsPerSlab == 0 {
		return nil
	}
	
	slab := &Slab{
		buffer:     make([]byte, s.slabSize),
		freeList:   make([]unsafe.Pointer, objectsPerSlab),
		freeCount:  objectsPerSlab,
		totalCount: objectsPerSlab,
	}
	
	// Initialize free list
	for i := 0; i < objectsPerSlab; i++ {
		offset := uintptr(i) * s.objectSize
		slab.freeList[i] = unsafe.Pointer(&slab.buffer[offset])
	}
	
	return slab
}

// Deallocate returns an object to the slab allocator
func (s *SlabAllocator) Deallocate(ptr unsafe.Pointer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find the slab containing this pointer
	for _, slab := range s.slabs {
		slabStart := uintptr(unsafe.Pointer(&slab.buffer[0]))
		slabEnd := slabStart + s.slabSize
		ptrAddr := uintptr(ptr)
		
		if ptrAddr >= slabStart && ptrAddr < slabEnd {
			// Return to free list
			if slab.freeCount < slab.totalCount {
				slab.freeList[slab.freeCount] = ptr
				slab.freeCount++
				
				atomic.AddUint64(&s.stats.Deallocations, 1)
				atomic.AddUint64(&s.stats.BytesDeallocated, uint64(s.objectSize))
			}
			return
		}
	}
}

// Reset resets all slabs
func (s *SlabAllocator) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, slab := range s.slabs {
		slab.freeCount = slab.totalCount
		// Reinitialize free list
		for i := 0; i < slab.totalCount; i++ {
			offset := uintptr(i) * s.objectSize
			slab.freeList[i] = unsafe.Pointer(&slab.buffer[offset])
		}
	}
}

// Stats returns allocator statistics
func (s *SlabAllocator) Stats() *AllocatorStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	totalObjects := 0
	freeObjects := 0
	
	for _, slab := range s.slabs {
		totalObjects += slab.totalCount
		freeObjects += slab.freeCount
	}
	
	efficiency := 0.0
	if totalObjects > 0 {
		efficiency = float64(totalObjects-freeObjects) / float64(totalObjects)
	}
	
	fragmentation := 0.0
	if totalObjects > 0 {
		fragmentation = float64(freeObjects) / float64(totalObjects)
	}
	
	s.stats.Efficiency = efficiency
	s.stats.Fragmentation = fragmentation
	
	return s.stats
}

// AdvancedBufferPool implementation

// NewAdvancedBufferPool creates a new advanced buffer pool
func NewAdvancedBufferPool() *AdvancedBufferPool {
	return &AdvancedBufferPool{
		smallBuffers: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024) // 1KB
			},
		},
		mediumBuffers: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 64*1024) // 64KB
			},
		},
		largeBuffers: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024*1024) // 1MB
			},
		},
		customSizes: make(map[int]*sync.Pool),
		stats:       &BufferPoolStats{},
	}
}

// GetBuffer gets a buffer of the specified size
func (p *AdvancedBufferPool) GetBuffer(size int) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	var buffer []byte
	
	switch {
	case size <= 1024:
		buffer = p.smallBuffers.Get().([]byte)
		atomic.AddUint64(&p.stats.SmallHits, 1)
	case size <= 64*1024:
		buffer = p.mediumBuffers.Get().([]byte)
		atomic.AddUint64(&p.stats.MediumHits, 1)
	case size <= 1024*1024:
		buffer = p.largeBuffers.Get().([]byte)
		atomic.AddUint64(&p.stats.LargeHits, 1)
	default:
		// Custom size
		if pool, exists := p.customSizes[size]; exists {
			buffer = pool.Get().([]byte)
			atomic.AddUint64(&p.stats.CustomHits, 1)
		} else {
			// Create new pool for this size
			p.customSizes[size] = &sync.Pool{
				New: func() interface{} {
					return make([]byte, 0, size)
				},
			}
			buffer = make([]byte, 0, size)
			atomic.AddUint64(&p.stats.Misses, 1)
		}
	}
	
	// Reset buffer length but keep capacity
	buffer = buffer[:0]
	atomic.AddUint64(&p.stats.TotalReused, 1)
	
	return buffer
}

// PutBuffer returns a buffer to the pool
func (p *AdvancedBufferPool) PutBuffer(buffer []byte) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	capacity := cap(buffer)
	
	switch {
	case capacity <= 1024:
		p.smallBuffers.Put(buffer)
	case capacity <= 64*1024:
		p.mediumBuffers.Put(buffer)
	case capacity <= 1024*1024:
		p.largeBuffers.Put(buffer)
	default:
		if pool, exists := p.customSizes[capacity]; exists {
			pool.Put(buffer)
		}
		// If no pool exists, just let it be garbage collected
	}
	
	atomic.AddUint64(&p.stats.MemorySaved, uint64(capacity))
}

// GetStats returns buffer pool statistics
func (p *AdvancedBufferPool) GetStats() *BufferPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	totalHits := p.stats.SmallHits + p.stats.MediumHits + p.stats.LargeHits + p.stats.CustomHits
	totalRequests := totalHits + p.stats.Misses
	
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(totalHits) / float64(totalRequests)
	}
	
	p.stats.HitRate = hitRate
	return p.stats
}

// TypedObjectPool implementation

// NewTypedObjectPool creates a new typed object pool
func NewTypedObjectPool(factory ObjectFactory) *TypedObjectPool {
	return &TypedObjectPool{
		pool: sync.Pool{
			New: func() interface{} {
				return factory.Create()
			},
		},
		factory: factory,
		stats:   &TypedPoolStats{},
	}
}

// Get gets an object from the pool
func (p *TypedObjectPool) Get() interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	obj := p.pool.Get()
	atomic.AddUint64(&p.stats.Gets, 1)
	
	// Validate object
	if !p.factory.Validate(obj) {
		atomic.AddUint64(&p.stats.TypeErrors, 1)
		// Create new object
		obj = p.factory.Create()
		atomic.AddUint64(&p.stats.News, 1)
	}
	
	atomic.AddUint64(&p.stats.Validations, 1)
	return obj
}

// Put returns an object to the pool
func (p *TypedObjectPool) Put(obj interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Reset object state
	p.factory.Reset(obj)
	atomic.AddUint64(&p.stats.Resets, 1)
	
	// Validate before putting back
	if p.factory.Validate(obj) {
		p.pool.Put(obj)
		atomic.AddUint64(&p.stats.Puts, 1)
	} else {
		atomic.AddUint64(&p.stats.TypeErrors, 1)
	}
	
	atomic.AddUint64(&p.stats.Validations, 1)
}

// GetStats returns typed pool statistics
func (p *TypedObjectPool) GetStats() *TypedPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	hitRate := 0.0
	if p.stats.Gets > 0 {
		hitRate = float64(p.stats.Puts) / float64(p.stats.Gets)
	}
	
	p.stats.HitRate = hitRate
	return p.stats
}

// updateStats updates the allocator suite statistics
func (cas *CustomAllocatorSuite) updateStats() {
	cas.mu.Lock()
	defer cas.mu.Unlock()
	
	var totalAllocs, totalDeallocs, totalBytes, totalDeallocBytes uint64
	var totalFragmentation, totalEfficiency float64
	var allocatorCount int
	
	// Aggregate stats from all allocators
	for _, alloc := range cas.stackAllocators {
		stats := alloc.Stats()
		totalAllocs += stats.Allocations
		totalDeallocs += stats.Deallocations
		totalBytes += stats.BytesAllocated
		totalDeallocBytes += stats.BytesDeallocated
		totalFragmentation += stats.Fragmentation
		totalEfficiency += stats.Efficiency
		allocatorCount++
	}
	
	for _, alloc := range cas.linearAllocators {
		stats := alloc.Stats()
		totalAllocs += stats.Allocations
		totalDeallocs += stats.Deallocations
		totalBytes += stats.BytesAllocated
		totalDeallocBytes += stats.BytesDeallocated
		totalFragmentation += stats.Fragmentation
		totalEfficiency += stats.Efficiency
		allocatorCount++
	}
	
	// Calculate averages
	if allocatorCount > 0 {
		cas.stats.AverageFragmentation = totalFragmentation / float64(allocatorCount)
		cas.stats.AverageEfficiency = totalEfficiency / float64(allocatorCount)
	}
	
	cas.stats.TotalAllocations = totalAllocs
	cas.stats.TotalDeallocations = totalDeallocs
	cas.stats.TotalBytesAllocated = totalBytes
	cas.stats.TotalBytesDeallocated = totalDeallocBytes
	
	// Estimate GC pressure reduction (simplified calculation)
	cas.stats.GCPressureReduction = float64(totalBytes) * 0.8 / (1024 * 1024) // 80% reduction estimate
	cas.stats.PerformanceGain = cas.stats.AverageEfficiency * 50 // Simplified performance gain
}



// DemonstrateCustomAllocators demonstrates various custom allocators
func DemonstrateCustomAllocators() {
	fmt.Println("=== Custom Allocators Demonstration ===")
	
	// Create allocator suite
	suite := NewCustomAllocatorSuite()
	
	// Demonstrate stack allocator
	fmt.Println("\n1. Stack Allocator:")
	stackAlloc := suite.CreateStackAllocator("demo_stack", 64*1024) // 64KB
	for i := 0; i < 100; i++ {
		ptr := stackAlloc.Allocate(512)
		if ptr == nil {
			fmt.Printf("Stack allocator full after %d allocations\n", i)
			break
		}
	}
	stackStats := stackAlloc.Stats()
	fmt.Printf("Stack allocator - Allocations: %d, Efficiency: %.2f%%, Fragmentation: %.2f%%\n",
		stackStats.Allocations, stackStats.Efficiency*100, stackStats.Fragmentation*100)
	
	// Demonstrate linear allocator
	fmt.Println("\n2. Linear Allocator:")
	linearAlloc := suite.CreateLinearAllocator("demo_linear", 64*1024, 8) // 64KB, 8-byte aligned
	for i := 0; i < 100; i++ {
		size := 256 + (i % 256) // Variable sizes
		ptr := linearAlloc.Allocate(size)
		if ptr == nil {
			fmt.Printf("Linear allocator full after %d allocations\n", i)
			break
		}
	}
	linearStats := linearAlloc.Stats()
	fmt.Printf("Linear allocator - Allocations: %d, Efficiency: %.2f%%, Fragmentation: %.2f%%\n",
		linearStats.Allocations, linearStats.Efficiency*100, linearStats.Fragmentation*100)
	
	// Demonstrate arena allocator
	fmt.Println("\n3. Arena Allocator:")
	arenaAlloc := suite.CreateArenaAllocator("demo_arena", 32*1024) // 32KB arenas
	for i := 0; i < 200; i++ {
		size := 128 + (i % 512) // Variable sizes
		ptr := arenaAlloc.Allocate(size)
		if ptr == nil {
			fmt.Printf("Arena allocator failed at allocation %d\n", i)
			break
		}
	}
	arenaStats := arenaAlloc.Stats()
	fmt.Printf("Arena allocator - Allocations: %d, Efficiency: %.2f%%, Fragmentation: %.2f%%\n",
		arenaStats.Allocations, arenaStats.Efficiency*100, arenaStats.Fragmentation*100)
	
	// Demonstrate slab allocator
	fmt.Println("\n4. Slab Allocator:")
	slabAlloc := suite.CreateSlabAllocator("demo_slab", 64, 4096) // 64-byte objects, 4KB slabs
	allocatedPtrs := make([]unsafe.Pointer, 0)
	for i := 0; i < 100; i++ {
		ptr := slabAlloc.Allocate(64)
		if ptr != nil {
			allocatedPtrs = append(allocatedPtrs, ptr)
		}
	}
	
	// Deallocate some objects
	for i := 0; i < len(allocatedPtrs)/2; i++ {
		slabAlloc.Deallocate(allocatedPtrs[i])
	}
	
	slabStats := slabAlloc.Stats()
	fmt.Printf("Slab allocator - Allocations: %d, Deallocations: %d, Efficiency: %.2f%%, Fragmentation: %.2f%%\n",
		slabStats.Allocations, slabStats.Deallocations, slabStats.Efficiency*100, slabStats.Fragmentation*100)
	
	// Demonstrate advanced buffer pool
	fmt.Println("\n5. Advanced Buffer Pool:")
	bufferPool := NewAdvancedBufferPool()
	buffers := make([][]byte, 0)
	
	// Get buffers of various sizes
	sizes := []int{512, 2048, 32768, 131072, 524288}
	for _, size := range sizes {
		for i := 0; i < 20; i++ {
			buf := bufferPool.GetBuffer(size)
			buffers = append(buffers, buf)
		}
	}
	
	// Return buffers to pool
	for _, buf := range buffers {
		bufferPool.PutBuffer(buf)
	}
	
	// Get buffers again to test reuse
	for _, size := range sizes {
		for i := 0; i < 10; i++ {
			buf := bufferPool.GetBuffer(size)
			bufferPool.PutBuffer(buf)
		}
	}
	
	bufferStats := bufferPool.GetStats()
	fmt.Printf("Buffer pool - Small hits: %d, Medium hits: %d, Large hits: %d, Hit rate: %.2f%%\n",
		bufferStats.SmallHits, bufferStats.MediumHits, bufferStats.LargeHits, bufferStats.HitRate*100)
	
	// Overall suite statistics
	fmt.Println("\n6. Overall Allocator Suite Statistics:")
	suiteStats := suite.GetAllocatorStats()
	fmt.Printf("Total allocations: %d\n", suiteStats.TotalAllocations)
	fmt.Printf("Total bytes allocated: %.2f MB\n", float64(suiteStats.TotalBytesAllocated)/(1024*1024))
	fmt.Printf("Average efficiency: %.2f%%\n", suiteStats.AverageEfficiency*100)
	fmt.Printf("Average fragmentation: %.2f%%\n", suiteStats.AverageFragmentation*100)
	fmt.Printf("Estimated GC pressure reduction: %.2f MB\n", suiteStats.GCPressureReduction)
	fmt.Printf("Estimated performance gain: %.2f%%\n", suiteStats.PerformanceGain)
	
	// Memory usage comparison
	fmt.Println("\n7. Memory Usage Comparison:")
	var beforeStats, afterStats runtime.MemStats
	runtime.ReadMemStats(&beforeStats)
	
	// Simulate workload with standard allocation
	standardData := make([][]byte, 1000)
	for i := range standardData {
		standardData[i] = make([]byte, 1024)
	}
	
	runtime.ReadMemStats(&afterStats)
	standardHeapGrowth := afterStats.HeapInuse - beforeStats.HeapInuse
	
	// Reset and test with custom allocators
	runtime.GC()
	runtime.ReadMemStats(&beforeStats)
	
	// Simulate workload with custom allocators
	customAlloc := NewLinearAllocator(1024*1024, 8) // 1MB
	customData := make([]unsafe.Pointer, 1000)
	for i := range customData {
		customData[i] = customAlloc.Allocate(1024)
	}
	
	runtime.ReadMemStats(&afterStats)
	customHeapGrowth := afterStats.HeapInuse - beforeStats.HeapInuse
	
	fmt.Printf("Standard allocation heap growth: %.2f KB\n", float64(standardHeapGrowth)/1024)
	fmt.Printf("Custom allocator heap growth: %.2f KB\n", float64(customHeapGrowth)/1024)
	fmt.Printf("Memory savings: %.2f KB (%.2f%%)\n",
		float64(standardHeapGrowth-customHeapGrowth)/1024,
		float64(standardHeapGrowth-customHeapGrowth)/float64(standardHeapGrowth)*100)
	
	fmt.Println("\n=== Custom Allocators Demonstration Complete ===")
}