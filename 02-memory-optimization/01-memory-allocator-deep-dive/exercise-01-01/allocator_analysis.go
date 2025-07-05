package main

import (
	"fmt"
	"runtime"
	"sort"
	"time"
	"unsafe"
)

// AllocationTracker tracks memory allocation patterns
type AllocationTracker struct {
	allocations map[uintptr]AllocationInfo
	sizeClasses map[int]int // size -> count
	totalAllocs int64
	totalBytes  int64
}

// AllocationInfo stores information about an allocation
type AllocationInfo struct {
	Size      uintptr
	SizeClass int
	Timestamp time.Time
	StackTrace []uintptr
}

// NewAllocationTracker creates a new allocation tracker
func NewAllocationTracker() *AllocationTracker {
	return &AllocationTracker{
		allocations: make(map[uintptr]AllocationInfo),
		sizeClasses: make(map[int]int),
	}
}

// Go's size classes (simplified version)
var sizeClasses = []int{
	0, 8, 16, 24, 32, 48, 64, 80, 96, 112, 128,
	144, 160, 176, 192, 208, 224, 240, 256, 288, 320,
	352, 384, 416, 448, 480, 512, 576, 640, 704, 768,
	896, 1024, 1152, 1280, 1408, 1536, 1792, 2048, 2304, 2688,
	3072, 3200, 3456, 4096, 4864, 5376, 6144, 6528, 6784, 6912,
	8192, 9472, 9728, 10240, 10880, 12288, 13568, 14336, 16384, 18432,
	19072, 20480, 21760, 24576, 27264, 28672, 32768,
}

// GetSizeClass returns the size class for a given allocation size
func GetSizeClass(size uintptr) int {
	if size == 0 {
		return 0
	}
	
	// Find the smallest size class that can accommodate the size
	for _, class := range sizeClasses {
		if uintptr(class) >= size {
			return class
		}
	}
	
	// Large object (>32KB)
	return -1
}

// TrackAllocation records an allocation
func (at *AllocationTracker) TrackAllocation(ptr uintptr, size uintptr) {
	sizeClass := GetSizeClass(size)
	
	at.allocations[ptr] = AllocationInfo{
		Size:      size,
		SizeClass: sizeClass,
		Timestamp: time.Now(),
	}
	
	at.sizeClasses[sizeClass]++
	at.totalAllocs++
	at.totalBytes += int64(size)
}

// TrackDeallocation records a deallocation
func (at *AllocationTracker) TrackDeallocation(ptr uintptr) {
	if info, exists := at.allocations[ptr]; exists {
		at.sizeClasses[info.SizeClass]--
		at.totalBytes -= int64(info.Size)
		delete(at.allocations, ptr)
	}
}

// GetStats returns allocation statistics
func (at *AllocationTracker) GetStats() AllocationStats {
	stats := AllocationStats{
		TotalAllocations: at.totalAllocs,
		TotalBytes:       at.totalBytes,
		ActiveAllocations: int64(len(at.allocations)),
		SizeClassStats:   make(map[int]SizeClassStat),
	}
	
	for sizeClass, count := range at.sizeClasses {
		if count > 0 {
			stats.SizeClassStats[sizeClass] = SizeClassStat{
				Count: count,
				Bytes: int64(sizeClass * count),
			}
		}
	}
	
	return stats
}

// AllocationStats contains allocation statistics
type AllocationStats struct {
	TotalAllocations  int64
	TotalBytes        int64
	ActiveAllocations int64
	SizeClassStats    map[int]SizeClassStat
}

// SizeClassStat contains statistics for a size class
type SizeClassStat struct {
	Count int
	Bytes int64
}

// PrintStats prints detailed allocation statistics
func (stats AllocationStats) PrintStats() {
	fmt.Println("=== Allocation Statistics ===")
	fmt.Printf("Total Allocations: %d\n", stats.TotalAllocations)
	fmt.Printf("Total Bytes: %d (%.2f MB)\n", stats.TotalBytes, float64(stats.TotalBytes)/1024/1024)
	fmt.Printf("Active Allocations: %d\n", stats.ActiveAllocations)
	fmt.Println()
	
	fmt.Println("Size Class Distribution:")
	fmt.Printf("%-12s %-10s %-15s %-10s\n", "Size Class", "Count", "Total Bytes", "Efficiency")
	fmt.Println("--------------------------------------------------------")
	
	// Sort size classes for better readability
	var sortedClasses []int
	for sizeClass := range stats.SizeClassStats {
		sortedClasses = append(sortedClasses, sizeClass)
	}
	sort.Ints(sortedClasses)
	
	for _, sizeClass := range sortedClasses {
		stat := stats.SizeClassStats[sizeClass]
		efficiency := 100.0 // Default for exact matches
		
		if sizeClass > 0 {
			// Calculate efficiency (how much of allocated space is actually used)
			// This is a simplified calculation
			actualUsage := float64(stat.Bytes) / float64(stat.Count)
			efficiency = (actualUsage / float64(sizeClass)) * 100
		}
		
		sizeClassStr := fmt.Sprintf("%d", sizeClass)
		if sizeClass == -1 {
			sizeClassStr = "Large (>32KB)"
		} else if sizeClass == 0 {
			sizeClassStr = "Zero"
		}
		
		fmt.Printf("%-12s %-10d %-15d %.1f%%\n", 
			sizeClassStr, stat.Count, stat.Bytes, efficiency)
	}
}

// MemoryAnalyzer provides memory allocation analysis
type MemoryAnalyzer struct {
	tracker *AllocationTracker
}

// NewMemoryAnalyzer creates a new memory analyzer
func NewMemoryAnalyzer() *MemoryAnalyzer {
	return &MemoryAnalyzer{
		tracker: NewAllocationTracker(),
	}
}

// AnalyzeAllocationPattern analyzes allocation patterns for different data types
func (ma *MemoryAnalyzer) AnalyzeAllocationPattern() {
	fmt.Println("=== Allocation Pattern Analysis ===")
	
	// Test different allocation patterns
	patterns := []struct {
		name string
		fn   func()
	}{
		{"Small Objects (8-64 bytes)", ma.allocateSmallObjects},
		{"Medium Objects (64-1024 bytes)", ma.allocateMediumObjects},
		{"Large Objects (>32KB)", ma.allocateLargeObjects},
		{"Mixed Allocation Pattern", ma.allocateMixedPattern},
	}
	
	for _, pattern := range patterns {
		fmt.Printf("\n--- %s ---\n", pattern.name)
		
		// Measure before
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		// Run allocation pattern
		pattern.fn()
		
		// Measure after
		runtime.ReadMemStats(&m2)
		
		// Calculate differences
		allocDiff := m2.TotalAlloc - m1.TotalAlloc
		heapDiff := m2.HeapAlloc - m1.HeapAlloc
		mallocsDiff := m2.Mallocs - m1.Mallocs
		
		fmt.Printf("Total Allocated: %d bytes\n", allocDiff)
		fmt.Printf("Heap Allocated: %d bytes\n", heapDiff)
		fmt.Printf("Number of Mallocs: %d\n", mallocsDiff)
		
		if mallocsDiff > 0 {
			avgSize := float64(allocDiff) / float64(mallocsDiff)
			fmt.Printf("Average Allocation Size: %.2f bytes\n", avgSize)
			fmt.Printf("Optimal Size Class: %d bytes\n", GetSizeClass(uintptr(avgSize)))
		}
	}
}

// allocateSmallObjects allocates many small objects
func (ma *MemoryAnalyzer) allocateSmallObjects() {
	var ptrs []unsafe.Pointer
	
	for i := 0; i < 1000; i++ {
		// Allocate objects of various small sizes
		sizes := []int{8, 16, 24, 32, 48, 64}
		for _, size := range sizes {
			data := make([]byte, size)
			ptrs = append(ptrs, unsafe.Pointer(&data[0]))
		}
	}
	
	// Keep references to prevent GC
	_ = ptrs
}

// allocateMediumObjects allocates medium-sized objects
func (ma *MemoryAnalyzer) allocateMediumObjects() {
	var ptrs []unsafe.Pointer
	
	for i := 0; i < 100; i++ {
		// Allocate objects of various medium sizes
		sizes := []int{128, 256, 512, 1024}
		for _, size := range sizes {
			data := make([]byte, size)
			ptrs = append(ptrs, unsafe.Pointer(&data[0]))
		}
	}
	
	// Keep references to prevent GC
	_ = ptrs
}

// allocateLargeObjects allocates large objects
func (ma *MemoryAnalyzer) allocateLargeObjects() {
	var ptrs []unsafe.Pointer
	
	for i := 0; i < 10; i++ {
		// Allocate large objects (>32KB)
		sizes := []int{40960, 65536, 131072} // 40KB, 64KB, 128KB
		for _, size := range sizes {
			data := make([]byte, size)
			ptrs = append(ptrs, unsafe.Pointer(&data[0]))
		}
	}
	
	// Keep references to prevent GC
	_ = ptrs
}

// allocateMixedPattern allocates a mix of different sized objects
func (ma *MemoryAnalyzer) allocateMixedPattern() {
	var ptrs []unsafe.Pointer
	
	for i := 0; i < 500; i++ {
		// Mix of small, medium, and occasional large allocations
		switch i % 10 {
		case 0, 1, 2, 3, 4: // 50% small
			data := make([]byte, 32)
			ptrs = append(ptrs, unsafe.Pointer(&data[0]))
		case 5, 6, 7: // 30% medium
			data := make([]byte, 512)
			ptrs = append(ptrs, unsafe.Pointer(&data[0]))
		case 8: // 10% large-ish
			data := make([]byte, 4096)
			ptrs = append(ptrs, unsafe.Pointer(&data[0]))
		case 9: // 10% very large
			data := make([]byte, 40960)
			ptrs = append(ptrs, unsafe.Pointer(&data[0]))
		}
	}
	
	// Keep references to prevent GC
	_ = ptrs
}

// AnalyzeSizeClassEfficiency analyzes how efficiently size classes are used
func (ma *MemoryAnalyzer) AnalyzeSizeClassEfficiency() {
	fmt.Println("\n=== Size Class Efficiency Analysis ===")
	
	// Test different struct layouts
	structs := []struct {
		name string
		size uintptr
		fn   func() interface{}
	}{
		{"Poorly Aligned Struct", unsafe.Sizeof(PoorlyAlignedStruct{}), func() interface{} { return &PoorlyAlignedStruct{} }},
		{"Well Aligned Struct", unsafe.Sizeof(WellAlignedStruct{}), func() interface{} { return &WellAlignedStruct{} }},
		{"Packed Struct", unsafe.Sizeof(PackedStruct{}), func() interface{} { return &PackedStruct{} }},
	}
	
	for _, s := range structs {
		sizeClass := GetSizeClass(s.size)
		efficiency := float64(s.size) / float64(sizeClass) * 100
		
		fmt.Printf("%-25s: %d bytes -> Size Class %d (%.1f%% efficient)\n", 
			s.name, s.size, sizeClass, efficiency)
		
		// Show memory waste
		if sizeClass > 0 {
			waste := uintptr(sizeClass) - s.size
			fmt.Printf("  Memory waste per allocation: %d bytes\n", waste)
		}
	}
}

// Example structs with different alignment characteristics
type PoorlyAlignedStruct struct {
	a bool   // 1 byte
	b int64  // 8 bytes (7 bytes padding before)
	c bool   // 1 byte
	d int32  // 4 bytes (3 bytes padding before)
	e bool   // 1 byte (3 bytes padding after)
}

type WellAlignedStruct struct {
	b int64  // 8 bytes
	d int32  // 4 bytes
	a bool   // 1 byte
	c bool   // 1 byte
	e bool   // 1 byte (1 byte padding after)
}

type PackedStruct struct {
	a, c, e bool // 3 bytes
	_       byte // 1 byte padding
	d       int32 // 4 bytes
	b       int64 // 8 bytes
}

func main() {
	fmt.Println("Go Memory Allocator Analysis")
	fmt.Println("============================")
	
	analyzer := NewMemoryAnalyzer()
	
	// Analyze allocation patterns
	analyzer.AnalyzeAllocationPattern()
	
	// Analyze size class efficiency
	analyzer.AnalyzeSizeClassEfficiency()
	
	// Run size class analysis from size_classes.go
	RunSizeClassAnalysis()
	
	// Show current memory stats
	fmt.Println("\n=== Current Memory Statistics ===")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	fmt.Printf("Heap Allocated: %d bytes (%.2f MB)\n", m.HeapAlloc, float64(m.HeapAlloc)/1024/1024)
	fmt.Printf("Total Allocations: %d\n", m.TotalAlloc)
	fmt.Printf("Number of Mallocs: %d\n", m.Mallocs)
	fmt.Printf("Number of Frees: %d\n", m.Frees)
	fmt.Printf("Heap Objects: %d\n", m.HeapObjects)
	fmt.Printf("GC Cycles: %d\n", m.NumGC)
	
	if m.Mallocs > 0 {
		avgAllocSize := float64(m.TotalAlloc) / float64(m.Mallocs)
		fmt.Printf("Average Allocation Size: %.2f bytes\n", avgAllocSize)
	}
	
	fmt.Println("\n=== Size Class Reference ===")
	fmt.Println("Go uses the following size classes for small object allocation:")
	for i, class := range sizeClasses {
		if i%8 == 0 {
			fmt.Println()
		}
		fmt.Printf("%6d ", class)
	}
	fmt.Println()
	fmt.Println("\nObjects larger than 32KB are allocated directly from the heap.")
}