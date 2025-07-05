package main

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// Fixed-size object pool
type ObjectPool struct {
	pool     sync.Pool
	objectSize int
	allocated  int64
	reused    int64
}

func NewObjectPool(objectSize int, newFunc func() interface{}) *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: newFunc,
		},
		objectSize: objectSize,
	}
}

func (op *ObjectPool) Get() interface{} {
	obj := op.pool.Get()
	op.reused++
	return obj
}

func (op *ObjectPool) Put(obj interface{}) {
	op.pool.Put(obj)
}

func (op *ObjectPool) Stats() (allocated, reused int64) {
	return op.allocated, op.reused
}

// Slab allocator for fixed-size objects
type SlabAllocator struct {
	slabSize   int
	objectSize int
	slabs      []*Slab
	currentSlab *Slab
	mu         sync.Mutex
}

type Slab struct {
	data      []byte
	freeList  []int
	nextFree  int
	objectSize int
	capacity  int
	allocated int
}

func NewSlabAllocator(objectSize, slabSize int) *SlabAllocator {
	return &SlabAllocator{
		slabSize:   slabSize,
		objectSize: objectSize,
	}
}

func (sa *SlabAllocator) newSlab() *Slab {
	capacity := sa.slabSize / sa.objectSize
	slab := &Slab{
		data:       make([]byte, sa.slabSize),
		freeList:   make([]int, capacity),
		objectSize: sa.objectSize,
		capacity:   capacity,
	}
	
	// Initialize free list
	for i := 0; i < capacity; i++ {
		slab.freeList[i] = i
	}
	slab.nextFree = capacity - 1
	
	return slab
}

func (sa *SlabAllocator) Allocate() unsafe.Pointer {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	if sa.currentSlab == nil || sa.currentSlab.nextFree < 0 {
		// Need a new slab
		newSlab := sa.newSlab()
		sa.slabs = append(sa.slabs, newSlab)
		sa.currentSlab = newSlab
	}
	
	// Get object from current slab
	index := sa.currentSlab.freeList[sa.currentSlab.nextFree]
	sa.currentSlab.nextFree--
	sa.currentSlab.allocated++
	
	offset := index * sa.objectSize
	return unsafe.Pointer(&sa.currentSlab.data[offset])
}

func (sa *SlabAllocator) Free(ptr unsafe.Pointer) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	// Find which slab this pointer belongs to
	for _, slab := range sa.slabs {
		slabStart := uintptr(unsafe.Pointer(&slab.data[0]))
		slabEnd := slabStart + uintptr(sa.slabSize)
		ptrAddr := uintptr(ptr)
		
		if ptrAddr >= slabStart && ptrAddr < slabEnd {
			// Calculate object index
			offset := ptrAddr - slabStart
			index := int(offset) / sa.objectSize
			
			// Add to free list
			slab.nextFree++
			slab.freeList[slab.nextFree] = index
			slab.allocated--
			return
		}
	}
}

func (sa *SlabAllocator) Stats() (slabs, totalCapacity, totalAllocated int) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	
	slabs = len(sa.slabs)
	for _, slab := range sa.slabs {
		totalCapacity += slab.capacity
		totalAllocated += slab.allocated
	}
	return
}

// Ring buffer for streaming data
type RingBuffer struct {
	buffer   []byte
	size     int
	head     int
	tail     int
	full     bool
	mu       sync.RWMutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]byte, size),
		size:   size,
	}
}

func (rb *RingBuffer) Write(data []byte) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	if len(data) == 0 {
		return 0
	}
	
	written := 0
	for i, b := range data {
		if rb.full && rb.head == rb.tail {
			// Buffer is full, can't write more
			return i
		}
		
		rb.buffer[rb.head] = b
		rb.head = (rb.head + 1) % rb.size
		
		if rb.head == rb.tail {
			rb.full = true
		}
		
		written++
	}
	
	return written
}

func (rb *RingBuffer) Read(data []byte) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	if rb.isEmpty() {
		return 0
	}
	
	read := 0
	for i := 0; i < len(data) && !rb.isEmpty(); i++ {
		data[i] = rb.buffer[rb.tail]
		rb.tail = (rb.tail + 1) % rb.size
		rb.full = false
		read++
	}
	
	return read
}

func (rb *RingBuffer) isEmpty() bool {
	return !rb.full && rb.head == rb.tail
}

func (rb *RingBuffer) Available() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if rb.full {
		return rb.size
	}
	if rb.head >= rb.tail {
		return rb.head - rb.tail
	}
	return rb.size - rb.tail + rb.head
}

func (rb *RingBuffer) Capacity() int {
	return rb.size
}

// Memory arena for bulk allocations
type MemoryArena struct {
	blocks    [][]byte
	blockSize int
	current   int
	offset    int
	mu        sync.Mutex
}

func NewMemoryArena(blockSize int) *MemoryArena {
	return &MemoryArena{
		blockSize: blockSize,
	}
}

func (ma *MemoryArena) Allocate(size int) unsafe.Pointer {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	
	// Align size to 8 bytes
	alignedSize := (size + 7) & ^7
	
	if len(ma.blocks) == 0 || ma.offset+alignedSize > ma.blockSize {
		// Need a new block
		blockSize := ma.blockSize
		if alignedSize > blockSize {
			blockSize = alignedSize
		}
		
		newBlock := make([]byte, blockSize)
		ma.blocks = append(ma.blocks, newBlock)
		ma.current = len(ma.blocks) - 1
		ma.offset = 0
	}
	
	ptr := unsafe.Pointer(&ma.blocks[ma.current][ma.offset])
	ma.offset += alignedSize
	
	return ptr
}

func (ma *MemoryArena) Reset() {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	
	ma.current = 0
	ma.offset = 0
	// Keep blocks allocated for reuse
}

func (ma *MemoryArena) Stats() (blocks int, totalSize, usedSize int) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	
	blocks = len(ma.blocks)
	for i, block := range ma.blocks {
		totalSize += len(block)
		if i < ma.current {
			usedSize += len(block)
		} else if i == ma.current {
			usedSize += ma.offset
		}
	}
	return
}

// String interning pool
type StringInterner struct {
	strings    map[string]string
	totalCalls int
	mu         sync.RWMutex
}

func NewStringInterner() *StringInterner {
	return &StringInterner{
		strings: make(map[string]string),
	}
}

func (si *StringInterner) Intern(s string) string {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	si.totalCalls++
	
	if interned, exists := si.strings[s]; exists {
		return interned
	}
	
	// Store the string and return it
	si.strings[s] = s
	return s
}

func (si *StringInterner) Size() int {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return len(si.strings)
}

func (si *StringInterner) Stats() (unique, total int) {
	si.mu.RLock()
	defer si.mu.RUnlock()
	unique = len(si.strings)
	total = si.totalCalls
	return
}

func (si *StringInterner) MemorySavings() int {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	// Rough estimate of memory savings
	totalBytes := 0
	for s := range si.strings {
		totalBytes += len(s)
	}
	
	// Assume some savings from interning (simplified calculation)
	return totalBytes / 4 // Assume 25% savings
}

// Buffer pool for reusable byte slices
type BufferPool struct {
	pool     sync.Pool
	bufSize  int
	maxCount int
	pooled   int64
	created  int64
	mu       sync.Mutex
}

func NewBufferPool(bufSize, maxCount int) *BufferPool {
	bp := &BufferPool{
		bufSize:  bufSize,
		maxCount: maxCount,
	}
	bp.pool.New = func() interface{} {
		atomic.AddInt64(&bp.created, 1)
		return make([]byte, 0, bufSize)
	}
	return bp
}

func (bp *BufferPool) Get() []byte {
	buf := bp.pool.Get().([]byte)
	atomic.AddInt64(&bp.pooled, -1)
	return buf[:0] // Reset length to 0
}

func (bp *BufferPool) Put(buf []byte) {
	if cap(buf) >= bp.bufSize {
		atomic.AddInt64(&bp.pooled, 1)
		bp.pool.Put(buf[:0])
	}
}

func (bp *BufferPool) Stats() (pooled, created int) {
	return int(atomic.LoadInt64(&bp.pooled)), int(atomic.LoadInt64(&bp.created))
}