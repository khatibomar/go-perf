// Package main demonstrates memory-intensive operations for profiling analysis.
// This file contains deliberately inefficient memory allocation patterns
// to showcase memory profiling techniques and optimization opportunities.
package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// MemoryLeaker demonstrates memory leaks through slice references
type MemoryLeaker struct {
	data [][]byte
}

// NewMemoryLeaker creates a new memory leaker instance
func NewMemoryLeaker() *MemoryLeaker {
	return &MemoryLeaker{
		data: make([][]byte, 0),
	}
}

// LeakMemory creates large slices and keeps references
func (ml *MemoryLeaker) LeakMemory(size int) {
	for i := 0; i < size; i++ {
		// Create large slice but only keep small portion
		bigSlice := make([]byte, 1024*1024) // 1MB
		for j := range bigSlice {
			bigSlice[j] = byte(j % 256)
		}
		// Keep reference to small portion, but entire slice stays in memory
		ml.data = append(ml.data, bigSlice[:100])
	}
}

// stringConcatenationBad demonstrates inefficient string concatenation
func stringConcatenationBad(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		// Each concatenation creates a new string, causing O(n²) allocations
		result += fmt.Sprintf("item_%d_", i)
	}
	return result
}

// sliceGrowthBad demonstrates inefficient slice growth
func sliceGrowthBad(n int) []int {
	// Start with zero capacity, causing multiple reallocations
	var result []int
	for i := 0; i < n; i++ {
		// Each append may trigger reallocation and copy
		result = append(result, i*i)
	}
	return result
}

// mapAllocationsBad demonstrates inefficient map usage
func mapAllocationsBad(n int) map[string]interface{} {
	// Create map without initial capacity
	result := make(map[string]interface{})
	for i := 0; i < n; i++ {
		// Create complex nested structures
		key := fmt.Sprintf("key_%d", i)
		value := map[string]interface{}{
			"id":    i,
			"name":  fmt.Sprintf("name_%d", i),
			"data":  make([]byte, 1024), // 1KB per entry
			"items": make([]int, 100),   // 100 integers per entry
		}
		result[key] = value
	}
	return result
}

// interfaceAllocationsBad demonstrates interface allocation overhead
func interfaceAllocationsBad(n int) []interface{} {
	result := make([]interface{}, n)
	for i := 0; i < n; i++ {
		// Boxing primitive values into interfaces causes heap allocations
		switch i % 4 {
		case 0:
			result[i] = i // int to interface{}
		case 1:
			result[i] = float64(i) // float64 to interface{}
		case 2:
			result[i] = fmt.Sprintf("str_%d", i) // string to interface{}
		case 3:
			result[i] = []int{i, i + 1, i + 2} // slice to interface{}
		}
	}
	return result
}

// closureAllocationsBad demonstrates closure allocation issues
func closureAllocationsBad(n int) []func() int {
	funcs := make([]func() int, n)
	for i := 0; i < n; i++ {
		// Each closure captures variables, causing heap allocations
		value := i * i
		data := make([]int, 100) // Large capture
		for j := range data {
			data[j] = j + i
		}
		funcs[i] = func() int {
			// Closure captures 'value' and 'data'
			return value + len(data)
		}
	}
	return funcs
}

// channelAllocationsBad demonstrates inefficient channel usage
func channelAllocationsBad(n int) {
	channels := make([]chan int, n)
	for i := 0; i < n; i++ {
		// Create many channels with large buffers
		channels[i] = make(chan int, 1000)
		go func(ch chan int, id int) {
			// Fill channel with data
			for j := 0; j < 1000; j++ {
				ch <- id*1000 + j
			}
			close(ch)
		}(channels[i], i)
	}
	
	// Wait and drain channels
	for _, ch := range channels {
		for range ch {
			// Consume all values
		}
	}
}

// recursiveStructuresBad demonstrates memory issues with recursive structures
type TreeNode struct {
	Value    int
	Data     []byte // Large data per node
	Children []*TreeNode
}

// createDeepTreeBad creates a deep tree structure with memory inefficiencies
func createDeepTreeBad(depth, branching int) *TreeNode {
	if depth <= 0 {
		return nil
	}
	
	node := &TreeNode{
		Value:    rand.Intn(1000),
		Data:     make([]byte, 1024), // 1KB per node
		Children: make([]*TreeNode, branching),
	}
	
	// Fill data with random values
	for i := range node.Data {
		node.Data[i] = byte(rand.Intn(256))
	}
	
	// Create children recursively
	for i := 0; i < branching; i++ {
		node.Children[i] = createDeepTreeBad(depth-1, branching)
	}
	
	return node
}

// temporaryAllocationsBad demonstrates excessive temporary allocations
func temporaryAllocationsBad(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		// Create temporary slices in loop
		temp := make([]int, 1000)
		for j := range temp {
			temp[j] = i + j
		}
		
		// Create temporary maps
		tempMap := make(map[int]string)
		for j := 0; j < 100; j++ {
			tempMap[j] = fmt.Sprintf("temp_%d_%d", i, j)
		}
		
		// Use temporary data
		for _, v := range temp {
			sum += v
		}
		sum += len(tempMap)
	}
	return sum
}

// stringProcessingBad demonstrates inefficient string processing
func stringProcessingBad(texts []string) []string {
	results := make([]string, 0)
	for _, text := range texts {
		// Multiple string operations creating temporary strings
		processed := strings.ToUpper(text)
		processed = strings.ReplaceAll(processed, " ", "_")
		processed = strings.ReplaceAll(processed, "-", "_")
		processed = strings.ReplaceAll(processed, ".", "_")
		
		// String concatenation in loop
		for i := 0; i < 10; i++ {
			processed += fmt.Sprintf("_SUFFIX_%d", i)
		}
		
		results = append(results, processed)
	}
	return results
}

// generateTestData creates test data for memory profiling
func generateTestData(size int) []string {
	data := make([]string, size)
	for i := 0; i < size; i++ {
		data[i] = fmt.Sprintf("test-data-item-%d.example", i)
	}
	return data
}

// generateLargeSlice creates a large slice for testing
func generateLargeSlice(size int) []int {
	data := make([]int, size)
	for i := range data {
		data[i] = rand.Intn(1000)
	}
	return data
}

// memoryIntensiveWorkload combines multiple memory-intensive operations
func memoryIntensiveWorkload(scale int) {
	// String concatenation
	_ = stringConcatenationBad(scale * 100)
	
	// Slice growth
	_ = sliceGrowthBad(scale * 1000)
	
	// Map allocations
	_ = mapAllocationsBad(scale * 50)
	
	// Interface allocations
	_ = interfaceAllocationsBad(scale * 200)
	
	// Closure allocations
	_ = closureAllocationsBad(scale * 10)
	
	// Channel allocations
	channelAllocationsBad(scale * 5)
	
	// Tree structures
	_ = createDeepTreeBad(8, 3)
	
	// Temporary allocations
	_ = temporaryAllocationsBad(scale * 10)
	
	// String processing
	testData := generateTestData(scale * 20)
	_ = stringProcessingBad(testData)
	
	// Memory leaker
	leaker := NewMemoryLeaker()
	leaker.LeakMemory(scale)
}

// init initializes random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}

// main function for standalone testing
func main() {
	fmt.Println("Memory-intensive operations for profiling...")
	memoryIntensiveWorkload(10)
	fmt.Println("Completed memory-intensive workload")
}