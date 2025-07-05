package main

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

// Memory layout visualization and analysis

// Different struct layouts to demonstrate padding
type BadLayout struct {
	a bool   // 1 byte
	b int64  // 8 bytes (7 bytes padding before)
	c bool   // 1 byte
	d int32  // 4 bytes (3 bytes padding before)
	e bool   // 1 byte
	f int16  // 2 bytes (1 byte padding before)
	g bool   // 1 byte
	// Total: 32 bytes with padding
}

type GoodLayout struct {
	b int64 // 8 bytes
	d int32 // 4 bytes
	f int16 // 2 bytes
	a bool  // 1 byte
	c bool  // 1 byte
	e bool  // 1 byte
	g bool  // 1 byte
	// Total: 20 bytes with minimal padding
}

type OptimalLayout struct {
	b int64 // 8 bytes
	d int32 // 4 bytes
	f int16 // 2 bytes
	// Pack all bools together
	flags struct {
		a, c, e, g bool // 4 bytes total
	}
	// Total: 18 bytes
}

// Demonstrate slice header layout
type SliceHeader struct {
	Data uintptr // 8 bytes on 64-bit
	Len  int     // 8 bytes on 64-bit
	Cap  int     // 8 bytes on 64-bit
	// Total: 24 bytes
}

// Demonstrate string header layout
type StringHeader struct {
	Data uintptr // 8 bytes on 64-bit
	Len  int     // 8 bytes on 64-bit
	// Total: 16 bytes
}

// Demonstrate interface layout
type InterfaceHeader struct {
	Type uintptr // 8 bytes - pointer to type information
	Data uintptr // 8 bytes - pointer to actual data
	// Total: 16 bytes
}

// Different array types for memory analysis
type SmallArray [10]int
type LargeArray [10000]int

// Pointer vs value demonstration
type Node struct {
	Value int
	Next  *Node
}

type ValueNode struct {
	Value int
	HasNext bool
	Next *ValueNode // Fixed: use pointer for recursive types
}

// Cache-friendly vs cache-unfriendly layouts
type CacheUnfriendly struct {
	HotData1  int64   // Frequently accessed
	ColdData1 [100]byte // Rarely accessed
	HotData2  int64   // Frequently accessed
	ColdData2 [100]byte // Rarely accessed
	HotData3  int64   // Frequently accessed
}

type CacheFriendly struct {
	// Group hot data together
	HotData1 int64
	HotData2 int64
	HotData3 int64
	// Group cold data together
	ColdData1 [100]byte
	ColdData2 [100]byte
}

// Memory alignment examples
type Aligned struct {
	a int64  // 8-byte aligned
	b int32  // 4-byte aligned
	c int16  // 2-byte aligned
	d int8   // 1-byte aligned
}

type Misaligned struct {
	a int8   // 1 byte + 7 bytes padding
	b int64  // 8 bytes
	c int8   // 1 byte + 3 bytes padding
	d int32  // 4 bytes
	e int8   // 1 byte + 1 byte padding
	f int16  // 2 bytes
}

// Function to analyze struct layout
func analyzeStruct(name string, s interface{}) {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	
	fmt.Printf("\n=== %s Analysis ===\n", name)
	fmt.Printf("Total size: %d bytes\n", t.Size())
	fmt.Printf("Alignment: %d bytes\n", t.Align())
	fmt.Printf("Number of fields: %d\n", t.NumField())
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		fmt.Printf("  Field %d (%s):\n", i, field.Name)
		fmt.Printf("    Type: %s\n", field.Type)
		fmt.Printf("    Size: %d bytes\n", field.Type.Size())
		fmt.Printf("    Offset: %d bytes\n", field.Offset)
		fmt.Printf("    Alignment: %d bytes\n", field.Type.Align())
		
		// Calculate padding
		if i > 0 {
			prevField := t.Field(i-1)
			prevEnd := prevField.Offset + prevField.Type.Size()
			padding := field.Offset - prevEnd
			if padding > 0 {
				fmt.Printf("    Padding before: %d bytes\n", padding)
			}
		}
		
		// Show memory address for demonstration
		if fieldValue.CanAddr() {
			fmt.Printf("    Address: %p\n", fieldValue.Addr().Interface())
		}
	}
	
	// Calculate total padding
	lastField := t.Field(t.NumField() - 1)
	lastFieldEnd := lastField.Offset + lastField.Type.Size()
	totalPadding := t.Size() - lastFieldEnd
	if totalPadding > 0 {
		fmt.Printf("  Trailing padding: %d bytes\n", totalPadding)
	}
	
	efficiency := float64(lastFieldEnd) / float64(t.Size()) * 100
	fmt.Printf("  Memory efficiency: %.1f%%\n", efficiency)
}

// Function to demonstrate memory addresses
func demonstrateMemoryAddresses() {
	fmt.Println("\n=== Memory Address Demonstration ===")
	
	// Stack vs heap allocation
	stackVar := 42
	heapVar := new(int)
	*heapVar = 42
	
	fmt.Printf("Stack variable address: %p\n", &stackVar)
	fmt.Printf("Heap variable address: %p\n", heapVar)
	
	// Array vs slice addresses
	arr := [5]int{1, 2, 3, 4, 5}
	slice := []int{1, 2, 3, 4, 5}
	
	fmt.Printf("\nArray address: %p\n", &arr)
	fmt.Printf("Array[0] address: %p\n", &arr[0])
	fmt.Printf("Slice header address: %p\n", &slice)
	fmt.Printf("Slice data address: %p\n", &slice[0])
	
	// String addresses
	str := "Hello, World!"
	fmt.Printf("\nString header address: %p\n", &str)
	fmt.Printf("String data address: %p\n", unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&str)).Data))
	
	// Interface addresses
	var iface interface{} = 42
	fmt.Printf("\nInterface header address: %p\n", &iface)
	ifaceHeader := (*InterfaceHeader)(unsafe.Pointer(&iface))
	fmt.Printf("Interface type pointer: %p\n", unsafe.Pointer(ifaceHeader.Type))
	fmt.Printf("Interface data pointer: %p\n", unsafe.Pointer(ifaceHeader.Data))
}

// Function to demonstrate cache effects
func demonstrateCacheEffects() {
	fmt.Println("\n=== Cache Effects Demonstration ===")
	
	// Create instances
	unfriendly := &CacheUnfriendly{}
	friendly := &CacheFriendly{}
	
	fmt.Printf("CacheUnfriendly size: %d bytes\n", unsafe.Sizeof(*unfriendly))
	fmt.Printf("CacheFriendly size: %d bytes\n", unsafe.Sizeof(*friendly))
	
	// Show hot data addresses in unfriendly layout
	fmt.Printf("\nCacheUnfriendly hot data addresses:\n")
	fmt.Printf("  HotData1: %p\n", &unfriendly.HotData1)
	fmt.Printf("  HotData2: %p (offset: %d)\n", &unfriendly.HotData2, 
		uintptr(unsafe.Pointer(&unfriendly.HotData2)) - uintptr(unsafe.Pointer(&unfriendly.HotData1)))
	fmt.Printf("  HotData3: %p (offset: %d)\n", &unfriendly.HotData3,
		uintptr(unsafe.Pointer(&unfriendly.HotData3)) - uintptr(unsafe.Pointer(&unfriendly.HotData1)))
	
	// Show hot data addresses in friendly layout
	fmt.Printf("\nCacheFriendly hot data addresses:\n")
	fmt.Printf("  HotData1: %p\n", &friendly.HotData1)
	fmt.Printf("  HotData2: %p (offset: %d)\n", &friendly.HotData2,
		uintptr(unsafe.Pointer(&friendly.HotData2)) - uintptr(unsafe.Pointer(&friendly.HotData1)))
	fmt.Printf("  HotData3: %p (offset: %d)\n", &friendly.HotData3,
		uintptr(unsafe.Pointer(&friendly.HotData3)) - uintptr(unsafe.Pointer(&friendly.HotData1)))
}

// Function to analyze slice and string internals
func analyzeSliceAndString() {
	fmt.Println("\n=== Slice and String Internals ===")
	
	// Slice analysis
	slice := []int{1, 2, 3, 4, 5}
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	
	fmt.Printf("Slice: %v\n", slice)
	fmt.Printf("Slice header size: %d bytes\n", unsafe.Sizeof(slice))
	fmt.Printf("Slice data pointer: %p\n", unsafe.Pointer(sliceHeader.Data))
	fmt.Printf("Slice length: %d\n", sliceHeader.Len)
	fmt.Printf("Slice capacity: %d\n", sliceHeader.Cap)
	fmt.Printf("Element size: %d bytes\n", unsafe.Sizeof(slice[0]))
	fmt.Printf("Total data size: %d bytes\n", sliceHeader.Cap * int(unsafe.Sizeof(slice[0])))
	
	// String analysis
	str := "Hello, Go!"
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&str))
	
	fmt.Printf("\nString: %q\n", str)
	fmt.Printf("String header size: %d bytes\n", unsafe.Sizeof(str))
	fmt.Printf("String data pointer: %p\n", unsafe.Pointer(stringHeader.Data))
	fmt.Printf("String length: %d\n", stringHeader.Len)
	fmt.Printf("String data size: %d bytes\n", len(str))
	
	// Demonstrate string immutability
	str2 := str
	stringHeader2 := (*reflect.StringHeader)(unsafe.Pointer(&str2))
	fmt.Printf("\nString copy data pointer: %p (same as original: %t)\n", 
		unsafe.Pointer(stringHeader2.Data), stringHeader.Data == stringHeader2.Data)
}

// Function to show memory allocation patterns
func showMemoryAllocationPatterns() {
	fmt.Println("\n=== Memory Allocation Patterns ===")
	
	// Small vs large allocations
	smallAlloc := make([]byte, 32)     // Likely on stack or small object pool
	largeAlloc := make([]byte, 32*1024) // Definitely on heap
	
	fmt.Printf("Small allocation address: %p\n", &smallAlloc[0])
	fmt.Printf("Large allocation address: %p\n", &largeAlloc[0])
	
	// Array vs slice allocation
	var stackArray [1000]int
	heapSlice := make([]int, 1000)
	
	fmt.Printf("\nStack array address: %p\n", &stackArray[0])
	fmt.Printf("Heap slice address: %p\n", &heapSlice[0])
	
	// Pointer chain analysis
	node1 := &Node{Value: 1}
	node2 := &Node{Value: 2, Next: node1}
	node3 := &Node{Value: 3, Next: node2}
	
	fmt.Printf("\nPointer chain addresses:\n")
	fmt.Printf("Node3: %p -> Node2: %p -> Node1: %p\n", node3, node2, node1)
	fmt.Printf("Address differences:\n")
	fmt.Printf("  Node3-Node2: %d bytes\n", 
		int64(uintptr(unsafe.Pointer(node3))) - int64(uintptr(unsafe.Pointer(node2))))
	fmt.Printf("  Node2-Node1: %d bytes\n", 
		int64(uintptr(unsafe.Pointer(node2))) - int64(uintptr(unsafe.Pointer(node1))))
}

// Function to demonstrate memory alignment
func demonstrateAlignment() {
	fmt.Println("\n=== Memory Alignment Demonstration ===")
	
	aligned := Aligned{}
	misaligned := Misaligned{}
	
	fmt.Printf("Aligned struct size: %d bytes\n", unsafe.Sizeof(aligned))
	fmt.Printf("Misaligned struct size: %d bytes\n", unsafe.Sizeof(misaligned))
	
	// Show field alignments
	fmt.Printf("\nField alignments:\n")
	fmt.Printf("int8 alignment: %d\n", unsafe.Alignof(int8(0)))
	fmt.Printf("int16 alignment: %d\n", unsafe.Alignof(int16(0)))
	fmt.Printf("int32 alignment: %d\n", unsafe.Alignof(int32(0)))
	fmt.Printf("int64 alignment: %d\n", unsafe.Alignof(int64(0)))
	fmt.Printf("pointer alignment: %d\n", unsafe.Alignof(uintptr(0)))
}

func main() {
	fmt.Println("=== Go Memory Layout Analysis ===")
	fmt.Printf("Architecture: %s\n", runtime.GOARCH)
	fmt.Printf("Pointer size: %d bytes\n", unsafe.Sizeof(uintptr(0)))
	fmt.Printf("Int size: %d bytes\n", unsafe.Sizeof(int(0)))
	
	// Analyze different struct layouts
	analyzeStruct("BadLayout", BadLayout{})
	analyzeStruct("GoodLayout", GoodLayout{})
	analyzeStruct("OptimalLayout", OptimalLayout{})
	
	// Show basic type sizes
	fmt.Println("\n=== Basic Type Sizes ===")
	fmt.Printf("bool: %d bytes\n", unsafe.Sizeof(bool(false)))
	fmt.Printf("int8: %d bytes\n", unsafe.Sizeof(int8(0)))
	fmt.Printf("int16: %d bytes\n", unsafe.Sizeof(int16(0)))
	fmt.Printf("int32: %d bytes\n", unsafe.Sizeof(int32(0)))
	fmt.Printf("int64: %d bytes\n", unsafe.Sizeof(int64(0)))
	fmt.Printf("int: %d bytes\n", unsafe.Sizeof(int(0)))
	fmt.Printf("uintptr: %d bytes\n", unsafe.Sizeof(uintptr(0)))
	fmt.Printf("float32: %d bytes\n", unsafe.Sizeof(float32(0)))
	fmt.Printf("float64: %d bytes\n", unsafe.Sizeof(float64(0)))
	fmt.Printf("complex64: %d bytes\n", unsafe.Sizeof(complex64(0)))
	fmt.Printf("complex128: %d bytes\n", unsafe.Sizeof(complex128(0)))
	fmt.Printf("string: %d bytes\n", unsafe.Sizeof(string("")))
	fmt.Printf("[]int: %d bytes\n", unsafe.Sizeof([]int{}))
	fmt.Printf("interface{}: %d bytes\n", unsafe.Sizeof(interface{}(nil)))
	fmt.Printf("chan int: %d bytes\n", unsafe.Sizeof(make(chan int)))
	fmt.Printf("map[int]int: %d bytes\n", unsafe.Sizeof(make(map[int]int)))
	fmt.Printf("func(): %d bytes\n", unsafe.Sizeof(func(){}))
	
	// Array sizes
	fmt.Println("\n=== Array Sizes ===")
	fmt.Printf("[10]int: %d bytes\n", unsafe.Sizeof(SmallArray{}))
	fmt.Printf("[10000]int: %d bytes\n", unsafe.Sizeof(LargeArray{}))
	
	// Demonstrate memory addresses
	demonstrateMemoryAddresses()
	
	// Analyze slice and string internals
	analyzeSliceAndString()
	
	// Show memory allocation patterns
	showMemoryAllocationPatterns()
	
	// Demonstrate cache effects
	demonstrateCacheEffects()
	
	// Demonstrate alignment
	demonstrateAlignment()
	
	fmt.Println("\n=== Key Takeaways ===")
	fmt.Println("1. Struct field ordering affects memory usage due to alignment")
	fmt.Println("2. Group frequently accessed fields together for better cache performance")
	fmt.Println("3. Slices and strings are headers pointing to underlying data")
	fmt.Println("4. Interfaces have overhead due to type information storage")
	fmt.Println("5. Memory alignment requirements can create padding")
	fmt.Println("6. Stack allocations are faster than heap allocations")
	fmt.Println("7. Understanding memory layout helps optimize data structures")
}