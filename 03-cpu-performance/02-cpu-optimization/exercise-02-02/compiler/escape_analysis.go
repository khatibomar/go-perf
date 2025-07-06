package compiler

import (
	"fmt"
)

// ShowEscapeAnalysis displays information about escape analysis
func ShowEscapeAnalysis() {
	fmt.Println("Escape Analysis Information:")
	fmt.Println("- Variables that don't escape stay on the stack (faster)")
	fmt.Println("- Variables that escape are allocated on the heap (slower)")
	fmt.Println("- Use -gcflags='-m' to see escape analysis decisions")
	fmt.Println("- Use -gcflags='-m -m' for more detailed analysis")
	fmt.Println("- Stack allocation is much faster than heap allocation")
}

// StackAllocation - variables stay on stack
func StackAllocation(size int) int {
	// This array should stay on the stack if size is small
	var arr [1000]int
	sum := 0
	
	// Fill array with values
	for i := 0; i < size && i < len(arr); i++ {
		arr[i] = i * 2
	}
	
	// Calculate sum
	for i := 0; i < size && i < len(arr); i++ {
		sum += arr[i]
	}
	
	return sum
}

// HeapAllocation - slice will escape to heap
func HeapAllocation(size int) int {
	// This slice will be allocated on the heap
	arr := make([]int, size)
	sum := 0
	
	// Fill slice with values
	for i := 0; i < size; i++ {
		arr[i] = i * 2
	}
	
	// Calculate sum
	for i := 0; i < size; i++ {
		sum += arr[i]
	}
	
	return sum
}

// ReturnPointer - causes escape to heap
func ReturnPointer() *int {
	x := 42 // This will escape to heap because we return its address
	return &x
}

// NoEscape - stays on stack
func NoEscape() int {
	x := 42 // This stays on stack
	return x
}

// EscapeViaInterface - interface causes escape
func EscapeViaInterface() interface{} {
	x := 42 // This will escape because it's stored in interface{}
	return x
}

// EscapeViaSlice - appending to slice from outer scope
func EscapeViaSlice(slice []int) []int {
	x := 42 // This will escape because it's appended to slice
	return append(slice, x)
}

// EscapeViaClosure - captured by closure
func EscapeViaClosure() func() int {
	x := 42 // This will escape because it's captured by closure
	return func() int {
		return x
	}
}

// NoEscapeLocal - local variable processing
func NoEscapeLocal(data []int) int {
	sum := 0    // Stays on stack
	count := 0  // Stays on stack
	
	for _, v := range data {
		if v > 0 {
			sum += v
			count++
		}
	}
	
	if count > 0 {
		return sum / count
	}
	return 0
}

// EscapeViaChannel - sending to channel
func EscapeViaChannel(ch chan<- int) {
	x := 42 // This will escape because it's sent to channel
	ch <- x
}

// LargeStruct - might escape due to size
type LargeStruct struct {
	data [10000]int
	name string
	id   int
}

func CreateLargeStruct() LargeStruct {
	// This might escape to heap due to size
	var s LargeStruct
	s.id = 1
	s.name = "large"
	for i := range s.data {
		s.data[i] = i
	}
	return s
}

func CreateLargeStructPointer() *LargeStruct {
	// This will definitely escape to heap
	var s LargeStruct
	s.id = 1
	s.name = "large"
	for i := range s.data {
		s.data[i] = i
	}
	return &s
}

// SmallStruct - likely to stay on stack
type SmallStruct struct {
	x, y int
	name string
}

func CreateSmallStruct() SmallStruct {
	// This should stay on stack
	return SmallStruct{x: 1, y: 2, name: "small"}
}

func CreateSmallStructPointer() *SmallStruct {
	// This will escape to heap
	s := SmallStruct{x: 1, y: 2, name: "small"}
	return &s
}

// ProcessInPlace - no escape
func ProcessInPlace(data []int) {
	// Local variables stay on stack
	multiplier := 2
	offset := 10
	
	for i := range data {
		data[i] = data[i]*multiplier + offset
	}
}

// ProcessWithReturn - might cause escape
func ProcessWithReturn(data []int) []int {
	// Result slice will be on heap
	result := make([]int, len(data))
	multiplier := 2
	offset := 10
	
	for i, v := range data {
		result[i] = v*multiplier + offset
	}
	
	return result
}

// StackFrameExample - demonstrates stack frame usage
func StackFrameExample(depth int) int {
	if depth <= 0 {
		return 1
	}
	
	// Local variables on each stack frame
	localVar := depth * 2
	anotherVar := localVar + 1
	
	return localVar + anotherVar + StackFrameExample(depth-1)
}

// EscapeAnalysisComparison - compare stack vs heap allocation patterns
func EscapeAnalysisComparison(useHeap bool, size int) interface{} {
	if useHeap {
		// Force heap allocation
		data := make([]int, size)
		for i := range data {
			data[i] = i
		}
		return data // Escapes via interface{}
	} else {
		// Try to keep on stack
		var data [100]int
		for i := 0; i < size && i < len(data); i++ {
			data[i] = i
		}
		return data[0] // Just return first element
	}
}

// MemoryPoolExample - reusing heap allocations
type MemoryPool struct {
	pool chan []int
}

func NewMemoryPool(size int) *MemoryPool {
	return &MemoryPool{
		pool: make(chan []int, size),
	}
}

func (p *MemoryPool) Get(size int) []int {
	select {
	case buf := <-p.pool:
		if cap(buf) >= size {
			return buf[:size]
		}
	default:
	}
	return make([]int, size)
}

func (p *MemoryPool) Put(buf []int) {
	select {
	case p.pool <- buf:
	default:
		// Pool is full, let GC handle it
	}
}

// EscapeAnalysisBenchmarks - helper functions for benchmarking
func BenchmarkStackAllocation(iterations int) int64 {
	var total int64
	for i := 0; i < iterations; i++ {
		result := StackAllocation(100)
		total += int64(result)
	}
	return total
}

func BenchmarkHeapAllocation(iterations int) int64 {
	var total int64
	for i := 0; i < iterations; i++ {
		result := HeapAllocation(100)
		total += int64(result)
	}
	return total
}

// Advanced escape analysis examples

// EscapeViaMap - map values escape
func EscapeViaMap() map[string]int {
	m := make(map[string]int)
	x := 42 // This will escape when stored in map
	m["key"] = x
	return m
}

// EscapeViaGoroutine - variables captured by goroutines escape
func EscapeViaGoroutine() {
	x := 42 // This will escape because it's captured by goroutine
	go func() {
		fmt.Println(x)
	}()
}

// NoEscapeWithCopy - passing by value doesn't cause escape
func NoEscapeWithCopy(data SmallStruct) SmallStruct {
	// data is copied, original doesn't escape
	data.x *= 2
	data.y *= 2
	return data
}