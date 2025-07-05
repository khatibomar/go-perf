package main

import (
	"fmt"
	"runtime"
)

// Example 1: Stack allocation - variable doesn't escape
func stackAllocation() int {
	x := 42 // This will be allocated on stack
	return x // Value is copied, x doesn't escape
}

// Example 2: Heap allocation - variable escapes via pointer
func heapAllocationPointer() *int {
	x := 42   // This will be allocated on heap
	return &x // Pointer returned, x escapes to heap
}

// Example 3: Heap allocation - variable escapes via slice
func heapAllocationSlice() []int {
	x := []int{1, 2, 3, 4, 5} // Slice allocated on heap
	return x                  // Slice escapes
}

// Example 4: Stack allocation - slice doesn't escape
func stackAllocationSlice() int {
	x := []int{1, 2, 3, 4, 5} // May be allocated on stack
	sum := 0
	for _, v := range x {
		sum += v
	}
	return sum // Only sum escapes (as value), slice doesn't
}

// Example 5: Heap allocation - large array
func largeArrayAllocation() [100000]int {
	var arr [100000]int // Too large for stack, goes to heap
	for i := range arr {
		arr[i] = i
	}
	return arr
}

// Example 6: Stack allocation - small array
func smallArrayAllocation() [10]int {
	var arr [10]int // Small enough for stack
	for i := range arr {
		arr[i] = i
	}
	return arr
}

// Example 7: Heap allocation - interface{} causes escape
func interfaceAllocation(x interface{}) {
	fmt.Println(x) // x escapes to heap due to interface{}
}

// Example 8: Heap allocation - closure captures variable
func closureAllocation() func() int {
	x := 42
	return func() int {
		return x // x escapes because it's captured by closure
	}
}

// Example 9: Stack allocation - no escape in closure
func noEscapeClosure() int {
	x := 42
	fn := func() int {
		return x // x doesn't escape if closure doesn't escape
	}
	return fn()
}

// Example 10: Heap allocation - slice append causes escape
func sliceAppendEscape(slice []int) []int {
	x := 100
	return append(slice, x) // slice may escape due to append
}

type Person struct {
	Name string
	Age  int
}

// Example 11: Stack allocation - struct doesn't escape
func structStackAllocation() Person {
	p := Person{Name: "John", Age: 30} // Allocated on stack
	return p                           // Value copied
}

// Example 12: Heap allocation - struct escapes via pointer
func structHeapAllocation() *Person {
	p := Person{Name: "John", Age: 30} // Allocated on heap
	return &p                          // Pointer returned
}

// Example 13: Conditional escape
func conditionalEscape(returnPointer bool) interface{} {
	x := 42
	if returnPointer {
		return &x // x escapes in this case
	}
	return x // x doesn't escape in this case, but compiler is conservative
}

func measureMemory(label string) {
	if label == "" {
		label = "Unknown"
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("%s - Heap: %d KB, Stack: %d KB, Allocs: %d\n",
		label, m.HeapAlloc/1024, m.StackInuse/1024, m.Mallocs)
}

func main() {
	// Validate runtime environment
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys < 50*1024*1024 { // Less than 50MB
		fmt.Println("Warning: Low system memory may affect escape analysis results")
	}
	
	fmt.Println("=== Escape Analysis Examples ===")
	fmt.Println("Run with: go build -gcflags=\"-m\" escape_analysis.go")
	fmt.Println("For detailed analysis: go build -gcflags=\"-m -m\" escape_analysis.go")
	fmt.Println()
	
	// Force GC to get clean baseline
	runtime.GC()
	measureMemory("Initial")
	
	// Test stack allocations
	fmt.Println("\n=== Stack Allocations ===")
	for i := 0; i < 1000000; i++ {
		_ = stackAllocation()
		_ = stackAllocationSlice()
		_ = smallArrayAllocation()
		_ = noEscapeClosure()
	}
	measureMemory("After stack allocations")
	
	// Test heap allocations
	fmt.Println("\n=== Heap Allocations ===")
	var pointers []*int
	var slices [][]int
	var structs []*Person
	
	for i := 0; i < 10000; i++ {
		pointers = append(pointers, heapAllocationPointer())
		slices = append(slices, heapAllocationSlice())
		structs = append(structs, structHeapAllocation())
	}
	measureMemory("After heap allocations")
	
	// Test large allocations
	fmt.Println("\n=== Large Allocations ===")
	var largeArrays [][100000]int
	for i := 0; i < 10; i++ {
		largeArrays = append(largeArrays, largeArrayAllocation())
	}
	measureMemory("After large allocations")
	
	// Test interface allocations
	fmt.Println("\n=== Interface Allocations ===")
	for i := 0; i < 100000; i++ {
		interfaceAllocation(i)
	}
	measureMemory("After interface allocations")
	
	// Keep references to prevent GC
	_ = pointers
	_ = slices
	_ = structs
	_ = largeArrays
	
	fmt.Println("\n=== Analysis Tips ===")
	fmt.Println("1. Variables that don't escape are allocated on stack (fast)")
	fmt.Println("2. Variables that escape are allocated on heap (slower, needs GC)")
	fmt.Println("3. Large objects are typically allocated on heap")
	fmt.Println("4. Interface{} parameters often cause escape")
	fmt.Println("5. Returning pointers causes escape")
	fmt.Println("6. Closures that capture variables cause escape")
}