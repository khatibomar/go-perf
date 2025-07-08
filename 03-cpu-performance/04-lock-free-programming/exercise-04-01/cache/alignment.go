package cache

import (
	"sync/atomic"
	"unsafe"
)

// AlignedData demonstrates cache line alignment
type AlignedData struct {
	value int64
	_     [7]int64 // Ensure cache line alignment
}

func NewAlignedData() *AlignedData {
	return &AlignedData{}
}

func (a *AlignedData) Load() int64 {
	return atomic.LoadInt64(&a.value)
}

func (a *AlignedData) Store(val int64) {
	atomic.StoreInt64(&a.value, val)
}

func (a *AlignedData) Add(delta int64) int64 {
	return atomic.AddInt64(&a.value, delta)
}

// CacheLineAlignedStruct ensures the entire struct is cache line aligned
type CacheLineAlignedStruct struct {
	data [CacheLineSize / 8]int64
}

func NewCacheLineAlignedStruct() *CacheLineAlignedStruct {
	return &CacheLineAlignedStruct{}
}

func (c *CacheLineAlignedStruct) LoadIndex(index int) int64 {
	if index >= 0 && index < len(c.data) {
		return atomic.LoadInt64(&c.data[index])
	}
	return 0
}

func (c *CacheLineAlignedStruct) StoreIndex(index int, val int64) {
	if index >= 0 && index < len(c.data) {
		atomic.StoreInt64(&c.data[index], val)
	}
}

func (c *CacheLineAlignedStruct) AddIndex(index int, delta int64) int64 {
	if index >= 0 && index < len(c.data) {
		return atomic.AddInt64(&c.data[index], delta)
	}
	return 0
}

// AlignmentChecker provides utilities to check memory alignment
type AlignmentChecker struct{}

func NewAlignmentChecker() *AlignmentChecker {
	return &AlignmentChecker{}
}

// IsAligned checks if a pointer is aligned to the specified boundary
func (ac *AlignmentChecker) IsAligned(ptr unsafe.Pointer, alignment uintptr) bool {
	return uintptr(ptr)%alignment == 0
}

// IsCacheLineAligned checks if a pointer is cache line aligned
func (ac *AlignmentChecker) IsCacheLineAligned(ptr unsafe.Pointer) bool {
	return ac.IsAligned(ptr, CacheLineSize)
}

// GetAlignment returns the alignment of a pointer
func (ac *AlignmentChecker) GetAlignment(ptr unsafe.Pointer) uintptr {
	addr := uintptr(ptr)
	if addr == 0 {
		return 0
	}
	
	// Find the largest power of 2 that divides the address
	alignment := uintptr(1)
	for (addr&alignment) == 0 && alignment <= CacheLineSize {
		alignment <<= 1
	}
	return alignment >> 1
}

// AlignedAllocator provides cache-line aligned memory allocation
type AlignedAllocator struct{}

func NewAlignedAllocator() *AlignedAllocator {
	return &AlignedAllocator{}
}

// AllocateAligned allocates cache-line aligned memory
func (aa *AlignedAllocator) AllocateAligned(size uintptr) unsafe.Pointer {
	// Allocate extra space for alignment
	buf := make([]byte, size+CacheLineSize-1)
	addr := uintptr(unsafe.Pointer(&buf[0]))
	aligned := (addr + CacheLineSize - 1) &^ (CacheLineSize - 1)
	return unsafe.Pointer(aligned)
}

// AlignedInt64Array provides a cache-line aligned array of int64
type AlignedInt64Array struct {
	data []int64
	buf  []byte // Keep reference to prevent GC
}

func NewAlignedInt64Array(size int) *AlignedInt64Array {
	// Calculate required buffer size
	bufSize := size*8 + CacheLineSize - 1
	buf := make([]byte, bufSize)
	
	// Find aligned address
	addr := uintptr(unsafe.Pointer(&buf[0]))
	aligned := (addr + CacheLineSize - 1) &^ (CacheLineSize - 1)
	
	// Create slice from aligned memory
	data := (*[1 << 30]int64)(unsafe.Pointer(aligned))[:size:size]
	
	return &AlignedInt64Array{
		data: data,
		buf:  buf,
	}
}

func (a *AlignedInt64Array) Load(index int) int64 {
	if index >= 0 && index < len(a.data) {
		return atomic.LoadInt64(&a.data[index])
	}
	return 0
}

func (a *AlignedInt64Array) Store(index int, val int64) {
	if index >= 0 && index < len(a.data) {
		atomic.StoreInt64(&a.data[index], val)
	}
}

func (a *AlignedInt64Array) Add(index int, delta int64) int64 {
	if index >= 0 && index < len(a.data) {
		return atomic.AddInt64(&a.data[index], delta)
	}
	return 0
}

func (a *AlignedInt64Array) Len() int {
	return len(a.data)
}

// GetBaseAddress returns the base address of the aligned array
func (a *AlignedInt64Array) GetBaseAddress() unsafe.Pointer {
	if len(a.data) > 0 {
		return unsafe.Pointer(&a.data[0])
	}
	return nil
}