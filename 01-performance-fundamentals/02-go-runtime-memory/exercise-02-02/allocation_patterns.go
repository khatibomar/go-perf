package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

// Memory allocation patterns demonstration

// Pattern 1: Pre-allocation vs Dynamic allocation
func dynamicAllocation(size int) []int {
	var slice []int
	for i := 0; i < size; i++ {
		slice = append(slice, i) // Multiple allocations as slice grows
	}
	return slice
}

func preAllocation(size int) []int {
	slice := make([]int, 0, size) // Pre-allocate capacity
	for i := 0; i < size; i++ {
		slice = append(slice, i) // No reallocation needed
	}
	return slice
}

func directAllocation(size int) []int {
	slice := make([]int, size) // Allocate exact size
	for i := 0; i < size; i++ {
		slice[i] = i // Direct assignment
	}
	return slice
}

// Pattern 2: Object pooling
type ObjectPool struct {
	pool chan []byte
}

func NewObjectPool(size int, bufferSize int) *ObjectPool {
	pool := make(chan []byte, size)
	for i := 0; i < size; i++ {
		pool <- make([]byte, bufferSize)
	}
	return &ObjectPool{pool: pool}
}

func (p *ObjectPool) Get() []byte {
	select {
	case buf := <-p.pool:
		return buf[:0] // Reset length but keep capacity
	default:
		return make([]byte, 0, 1024) // Fallback allocation
	}
}

func (p *ObjectPool) Put(buf []byte) {
	if cap(buf) < 1024 {
		return // Don't pool small buffers
	}
	select {
	case p.pool <- buf:
	default:
		// Pool is full, let GC handle it
	}
}

// Pattern 3: Memory-efficient string building
func inefficientStringBuilding(parts []string) string {
	result := ""
	for _, part := range parts {
		result += part // Creates new string each time
	}
	return result
}

func efficientStringBuilding(parts []string) string {
	// Calculate total length
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	
	// Pre-allocate builder with exact capacity
	var builder strings.Builder
	builder.Grow(totalLen)
	
	for _, part := range parts {
		builder.WriteString(part)
	}
	return builder.String()
}

// Pattern 4: Struct field ordering for memory efficiency
type BadStruct struct {
	a bool   // 1 byte + 7 bytes padding
	b int64  // 8 bytes
	c bool   // 1 byte + 7 bytes padding
	d int32  // 4 bytes + 4 bytes padding
	e bool   // 1 byte + 7 bytes padding
	// Total: 32 bytes due to padding
}

type GoodStruct struct {
	b int64 // 8 bytes
	d int32 // 4 bytes
	a bool  // 1 byte
	c bool  // 1 byte
	e bool  // 1 byte + 1 byte padding
	// Total: 16 bytes, much more efficient
}

// Pattern 5: Slice vs Array performance
func processArray(arr [1000]int) int {
	sum := 0
	for _, v := range arr {
		sum += v
	}
	return sum
}

func processSlice(slice []int) int {
	sum := 0
	for _, v := range slice {
		sum += v
	}
	return sum
}

// Pattern 6: Map pre-sizing
func dynamicMap(size int) map[int]string {
	m := make(map[int]string) // No initial size
	for i := 0; i < size; i++ {
		m[i] = fmt.Sprintf("value_%d", i)
	}
	return m
}

func preSizedMap(size int) map[int]string {
	m := make(map[int]string, size) // Pre-sized
	for i := 0; i < size; i++ {
		m[i] = fmt.Sprintf("value_%d", i)
	}
	return m
}

// Pattern 7: Interface{} boxing overhead
func processInterfaces(items []interface{}) int {
	sum := 0
	for _, item := range items {
		if v, ok := item.(int); ok {
			sum += v
		}
	}
	return sum
}

func processInts(items []int) int {
	sum := 0
	for _, item := range items {
		sum += item
	}
	return sum
}

// Memory measurement utilities
func measureMemory(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("%s:\n", label)
	fmt.Printf("  Heap Alloc: %d KB\n", m.HeapAlloc/1024)
	fmt.Printf("  Heap Objects: %d\n", m.HeapObjects)
	fmt.Printf("  GC Cycles: %d\n", m.NumGC)
	fmt.Printf("  Mallocs: %d\n", m.Mallocs)
	fmt.Printf("  Frees: %d\n", m.Frees)
	fmt.Println()
}

func timeOperation(name string, operation func()) {
	start := time.Now()
	operation()
	duration := time.Since(start)
	fmt.Printf("%s took: %v\n", name, duration)
}

func main() {
	fmt.Println("=== Memory Allocation Patterns ===")
	fmt.Println()
	
	// Force GC for clean baseline
	runtime.GC()
	measureMemory("Initial state")
	
	// Pattern 1: Slice allocation strategies
	fmt.Println("=== Slice Allocation Patterns ===")
	size := 100000
	
	timeOperation("Dynamic allocation", func() {
		_ = dynamicAllocation(size)
	})
	
	timeOperation("Pre-allocation", func() {
		_ = preAllocation(size)
	})
	
	timeOperation("Direct allocation", func() {
		_ = directAllocation(size)
	})
	
	measureMemory("After slice allocations")
	
	// Pattern 2: Object pooling demonstration
	fmt.Println("=== Object Pooling ===")
	pool := NewObjectPool(10, 1024)
	
	timeOperation("With object pool", func() {
		for i := 0; i < 10000; i++ {
			buf := pool.Get()
			// Simulate work
			buf = append(buf, byte(i%256))
			pool.Put(buf)
		}
	})
	
	timeOperation("Without object pool", func() {
		for i := 0; i < 10000; i++ {
			buf := make([]byte, 0, 1024)
			// Simulate work
			buf = append(buf, byte(i%256))
			// Let GC handle cleanup
			_ = buf
		}
	})
	
	measureMemory("After object pooling test")
	
	// Pattern 3: String building
	fmt.Println("=== String Building Patterns ===")
	parts := make([]string, 1000)
	for i := range parts {
		parts[i] = fmt.Sprintf("part_%d_", i)
	}
	
	timeOperation("Inefficient string building", func() {
		_ = inefficientStringBuilding(parts)
	})
	
	timeOperation("Efficient string building", func() {
		_ = efficientStringBuilding(parts)
	})
	
	measureMemory("After string building test")
	
	// Pattern 4: Struct memory layout
	fmt.Println("=== Struct Memory Layout ===")
	fmt.Printf("BadStruct size: %d bytes\n", unsafe.Sizeof(BadStruct{}))
	fmt.Printf("GoodStruct size: %d bytes\n", unsafe.Sizeof(GoodStruct{}))
	
	badStructs := make([]BadStruct, 100000)
	goodStructs := make([]GoodStruct, 100000)
	
	fmt.Printf("100k BadStructs memory: %d KB\n", len(badStructs)*int(unsafe.Sizeof(BadStruct{}))/1024)
	fmt.Printf("100k GoodStructs memory: %d KB\n", len(goodStructs)*int(unsafe.Sizeof(GoodStruct{}))/1024)
	
	// Pattern 5: Array vs Slice
	fmt.Println("\n=== Array vs Slice Performance ===")
	arr := [1000]int{}
	slice := make([]int, 1000)
	
	for i := range arr {
		arr[i] = i
		slice[i] = i
	}
	
	timeOperation("Process array", func() {
		for i := 0; i < 10000; i++ {
			_ = processArray(arr)
		}
	})
	
	timeOperation("Process slice", func() {
		for i := 0; i < 10000; i++ {
			_ = processSlice(slice)
		}
	})
	
	// Pattern 6: Map pre-sizing
	fmt.Println("\n=== Map Allocation Patterns ===")
	mapSize := 10000
	
	timeOperation("Dynamic map growth", func() {
		_ = dynamicMap(mapSize)
	})
	
	timeOperation("Pre-sized map", func() {
		_ = preSizedMap(mapSize)
	})
	
	measureMemory("After map allocation test")
	
	// Pattern 7: Interface boxing
	fmt.Println("=== Interface Boxing Overhead ===")
	ints := make([]int, 100000)
	interfaces := make([]interface{}, 100000)
	
	for i := range ints {
		ints[i] = i
		interfaces[i] = i // Boxing occurs here
	}
	
	timeOperation("Process typed slice", func() {
		_ = processInts(ints)
	})
	
	timeOperation("Process interface slice", func() {
		_ = processInterfaces(interfaces)
	})
	
	measureMemory("Final state")
	
	fmt.Println("=== Key Takeaways ===")
	fmt.Println("1. Pre-allocate slices and maps when size is known")
	fmt.Println("2. Use object pools for frequently allocated/deallocated objects")
	fmt.Println("3. Order struct fields by size (largest first) to minimize padding")
	fmt.Println("4. Avoid interface{} when possible to prevent boxing overhead")
	fmt.Println("5. Use strings.Builder with Grow() for efficient string concatenation")
	fmt.Println("6. Arrays can be faster than slices for fixed-size data")
	fmt.Println("7. Monitor memory allocations with runtime.MemStats")
}