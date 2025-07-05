package main

import (
	"fmt"
	"unsafe"
)

// ZeroAllocProcessor demonstrates zero-allocation processing techniques
type ZeroAllocProcessor struct {
	buffer []byte
	intBuffer []int
	resultBuffer []byte
}

// NewZeroAllocProcessor creates a new processor with pre-allocated buffers
func NewZeroAllocProcessor(bufferSize int) *ZeroAllocProcessor {
	return &ZeroAllocProcessor{
		buffer: make([]byte, bufferSize),
		intBuffer: make([]int, bufferSize),
		resultBuffer: make([]byte, bufferSize),
	}
}

// ReverseInPlace reverses a slice in place without allocations
func (zap *ZeroAllocProcessor) ReverseInPlace(data []int) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

// SortInPlace implements a simple in-place sorting algorithm
func (zap *ZeroAllocProcessor) SortInPlace(data []int) {
	// Simple bubble sort for demonstration (in-place)
	n := len(data)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if data[j] > data[j+1] {
				data[j], data[j+1] = data[j+1], data[j]
			}
		}
	}
}

// QuickSortInPlace implements quicksort in place
func (zap *ZeroAllocProcessor) QuickSortInPlace(data []int) {
	zap.quickSortRecursive(data, 0, len(data)-1)
}

func (zap *ZeroAllocProcessor) quickSortRecursive(data []int, low, high int) {
	if low < high {
		pi := zap.partition(data, low, high)
		zap.quickSortRecursive(data, low, pi-1)
		zap.quickSortRecursive(data, pi+1, high)
	}
}

func (zap *ZeroAllocProcessor) partition(data []int, low, high int) int {
	pivot := data[high]
	i := low - 1
	
	for j := low; j < high; j++ {
		if data[j] < pivot {
			i++
			data[i], data[j] = data[j], data[i]
		}
	}
	data[i+1], data[high] = data[high], data[i+1]
	return i + 1
}

// FilterInPlace filters elements in place, returning new length
func (zap *ZeroAllocProcessor) FilterInPlace(data []int, predicate func(int) bool) int {
	writeIndex := 0
	for readIndex := 0; readIndex < len(data); readIndex++ {
		if predicate(data[readIndex]) {
			data[writeIndex] = data[readIndex]
			writeIndex++
		}
	}
	return writeIndex
}

// TransformInPlace applies a transformation function in place
func (zap *ZeroAllocProcessor) TransformInPlace(data []int, transform func(int) int) {
	for i := range data {
		data[i] = transform(data[i])
	}
}

// FindMaxInPlace finds maximum value without allocations
func (zap *ZeroAllocProcessor) FindMaxInPlace(data []int) (int, bool) {
	if len(data) == 0 {
		return 0, false
	}
	
	max := data[0]
	for i := 1; i < len(data); i++ {
		if data[i] > max {
			max = data[i]
		}
	}
	return max, true
}

// BinarySearchInPlace performs binary search without allocations
func (zap *ZeroAllocProcessor) BinarySearchInPlace(data []int, target int) int {
	left, right := 0, len(data)-1
	
	for left <= right {
		mid := left + (right-left)/2
		if data[mid] == target {
			return mid
		} else if data[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return -1
}

// CopyWithoutAlloc copies data using pre-allocated buffer
func (zap *ZeroAllocProcessor) CopyWithoutAlloc(src []byte) []byte {
	if len(src) > len(zap.buffer) {
		// Resize buffer if needed (this will allocate, but only once)
		zap.buffer = make([]byte, len(src)*2)
	}
	
	n := copy(zap.buffer, src)
	return zap.buffer[:n]
}

// AppendWithoutAlloc appends data using pre-allocated buffer
func (zap *ZeroAllocProcessor) AppendWithoutAlloc(dst, src []byte) []byte {
	requiredSize := len(dst) + len(src)
	if requiredSize > len(zap.buffer) {
		// Resize buffer if needed
		zap.buffer = make([]byte, requiredSize*2)
	}
	
	n := copy(zap.buffer, dst)
	n += copy(zap.buffer[n:], src)
	return zap.buffer[:n]
}

// BytesToStringZeroAlloc converts bytes to string without allocation
// WARNING: This is unsafe and should only be used when you're sure
// the byte slice won't be modified
func BytesToStringZeroAlloc(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytesZeroAlloc converts string to bytes without allocation
// WARNING: This is unsafe and the returned slice should not be modified
func StringToBytesZeroAlloc(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			int
		}{s, len(s)},
	))
}

// IntToStringZeroAlloc converts int to string using pre-allocated buffer
func (zap *ZeroAllocProcessor) IntToStringZeroAlloc(n int) string {
	// Use the buffer as a byte slice for string building
	buf := zap.buffer[:0]
	
	if n == 0 {
		buf = append(buf, '0')
		return BytesToStringZeroAlloc(buf)
	}
	
	negative := n < 0
	if negative {
		n = -n
	}
	
	// Convert digits in reverse order
	start := len(buf)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	
	if negative {
		buf = append(buf, '-')
	}
	
	// Reverse the digits
	for i, j := start, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	
	return BytesToStringZeroAlloc(buf)
}

// ParseIntZeroAlloc parses integer from string without allocations
func (zap *ZeroAllocProcessor) ParseIntZeroAlloc(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty string")
	}
	
	negative := false
	start := 0
	
	if s[0] == '-' {
		negative = true
		start = 1
	} else if s[0] == '+' {
		start = 1
	}
	
	if start >= len(s) {
		return 0, fmt.Errorf("invalid number")
	}
	
	result := 0
	for i := start; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("invalid character")
		}
		result = result*10 + int(s[i]-'0')
	}
	
	if negative {
		result = -result
	}
	
	return result, nil
}

// HashZeroAlloc computes a simple hash without allocations
func (zap *ZeroAllocProcessor) HashZeroAlloc(data []byte) uint32 {
	var hash uint32 = 2166136261 // FNV offset basis
	for _, b := range data {
		hash ^= uint32(b)
		hash *= 16777619 // FNV prime
	}
	return hash
}

// CompareZeroAlloc compares two byte slices without allocations
func (zap *ZeroAllocProcessor) CompareZeroAlloc(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	
	for i := 0; i < minLen; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	
	if len(a) < len(b) {
		return -1
	} else if len(a) > len(b) {
		return 1
	}
	return 0
}

// Example usage and demonstration
func main() {
	processor := NewZeroAllocProcessor(1024)
	
	// Demonstrate in-place operations
	data := []int{64, 34, 25, 12, 22, 11, 90}
	fmt.Printf("Original: %v\n", data)
	
	processor.QuickSortInPlace(data)
	fmt.Printf("Sorted: %v\n", data)
	
	processor.ReverseInPlace(data)
	fmt.Printf("Reversed: %v\n", data)
	
	// Demonstrate filtering
	filterData := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	newLen := processor.FilterInPlace(filterData, func(x int) bool { return x%2 == 0 })
	fmt.Printf("Even numbers: %v\n", filterData[:newLen])
	
	// Demonstrate string conversion
	number := 12345
	str := processor.IntToStringZeroAlloc(number)
	fmt.Printf("Number as string: %s\n", str)
	
	parsed, err := processor.ParseIntZeroAlloc(str)
	if err == nil {
		fmt.Printf("Parsed back: %d\n", parsed)
	}
	
	// Demonstrate binary search
	sortedData := []int{1, 3, 5, 7, 9, 11, 13, 15}
	index := processor.BinarySearchInPlace(sortedData, 7)
	fmt.Printf("Index of 7: %d\n", index)
	
	// Demonstrate hash calculation
	testData := []byte("Hello, World!")
	hash := processor.HashZeroAlloc(testData)
	fmt.Printf("Hash of '%s': %d\n", string(testData), hash)
}

// Global variables for demonstration (avoid in real code)
var (
	globalResult int
	globalBytes []byte
	globalString string
)

// Sink functions to prevent compiler optimizations in benchmarks
func SinkInt(x int) {
	globalResult = x
}

func SinkBytes(x []byte) {
	globalBytes = x
}

func SinkString(x string) {
	globalString = x
}