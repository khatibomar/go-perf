// Package main demonstrates optimized memory allocation patterns.
// This file contains efficient versions of memory-intensive operations
// to showcase memory optimization techniques and best practices.
package main

import (
	"fmt"
	"strings"
	"sync"
)

// OptimizedMemoryManager demonstrates efficient memory management
type OptimizedMemoryManager struct {
	pool sync.Pool
	data [][]byte
}

// NewOptimizedMemoryManager creates a new optimized memory manager
func NewOptimizedMemoryManager() *OptimizedMemoryManager {
	return &OptimizedMemoryManager{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
		data: make([][]byte, 0),
	}
}

// EfficientMemoryUse demonstrates proper memory management with object pooling
func (omm *OptimizedMemoryManager) EfficientMemoryUse(size int) {
	for i := 0; i < size; i++ {
		// Get buffer from pool
		buf := omm.pool.Get().([]byte)
		
		// Use only what we need
		needed := buf[:100]
		for j := range needed {
			needed[j] = byte(j % 256)
		}
		
		// Make a copy of only what we need
		data := make([]byte, len(needed))
		copy(data, needed)
		omm.data = append(omm.data, data)
		
		// Return buffer to pool
		omm.pool.Put(buf)
	}
}

// stringConcatenationGood demonstrates efficient string concatenation
func stringConcatenationGood(n int) string {
	// Pre-allocate builder with estimated capacity
	var builder strings.Builder
	builder.Grow(n * 10) // Estimate final size
	
	for i := 0; i < n; i++ {
		// Use WriteString for efficiency
		builder.WriteString("item_")
		builder.WriteString(fmt.Sprintf("%d", i))
		builder.WriteString("_")
	}
	return builder.String()
}

// sliceGrowthGood demonstrates efficient slice growth
func sliceGrowthGood(n int) []int {
	// Pre-allocate with known capacity
	result := make([]int, 0, n)
	for i := 0; i < n; i++ {
		result = append(result, i*i)
	}
	return result
}

// mapAllocationsGood demonstrates efficient map usage
func mapAllocationsGood(n int) map[string]*DataEntry {
	// Pre-allocate map with estimated size
	result := make(map[string]*DataEntry, n)
	
	// Reuse data buffers
	dataPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}
	itemsPool := sync.Pool{
		New: func() interface{} {
			return make([]int, 100)
		},
	}
	
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key_%d", i)
		
		// Get buffers from pools
		data := dataPool.Get().([]byte)
		items := itemsPool.Get().([]int)
		
		// Create entry with pooled resources
		entry := &DataEntry{
			ID:    i,
			Name:  key, // Reuse key string
			Data:  data[:1024:1024], // Set capacity explicitly
			Items: items[:100:100],   // Set capacity explicitly
		}
		
		result[key] = entry
	}
	return result
}

// DataEntry represents optimized data structure
type DataEntry struct {
	ID    int
	Name  string
	Data  []byte
	Items []int
}

// interfaceAllocationsGood demonstrates avoiding interface allocations
func interfaceAllocationsGood(n int) *TypedContainer {
	container := &TypedContainer{
		Ints:    make([]int, 0, n/4),
		Floats:  make([]float64, 0, n/4),
		Strings: make([]string, 0, n/4),
		Slices:  make([][]int, 0, n/4),
	}
	
	for i := 0; i < n; i++ {
		// Store in typed containers instead of interface{}
		switch i % 4 {
		case 0:
			container.Ints = append(container.Ints, i)
		case 1:
			container.Floats = append(container.Floats, float64(i))
		case 2:
			container.Strings = append(container.Strings, fmt.Sprintf("str_%d", i))
		case 3:
			container.Slices = append(container.Slices, []int{i, i + 1, i + 2})
		}
	}
	return container
}

// TypedContainer avoids interface{} allocations
type TypedContainer struct {
	Ints    []int
	Floats  []float64
	Strings []string
	Slices  [][]int
}

// closureAllocationsGood demonstrates efficient closure usage
func closureAllocationsGood(n int) []func() int {
	funcs := make([]func() int, n)
	
	// Pre-allocate shared data to reduce per-closure allocations
	sharedData := make([][]int, n)
	for i := range sharedData {
		sharedData[i] = make([]int, 100)
		for j := range sharedData[i] {
			sharedData[i][j] = j + i
		}
	}
	
	for i := 0; i < n; i++ {
		value := i * i
		dataIndex := i // Capture index instead of slice
		
		funcs[i] = func() int {
			// Access shared data by index
			return value + len(sharedData[dataIndex])
		}
	}
	return funcs
}

// channelAllocationsGood demonstrates efficient channel usage
func channelAllocationsGood(n int) {
	// Use worker pool pattern instead of creating many channels
	const numWorkers = 4
	workCh := make(chan int, 100)
	resultCh := make(chan int, 100)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range workCh {
				// Process work
				for j := 0; j < 1000; j++ {
					resultCh <- id*1000 + j
				}
			}
		}()
	}
	
	// Send work
	go func() {
		defer close(workCh)
		for i := 0; i < n; i++ {
			workCh <- i
		}
	}()
	
	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()
	
	// Consume results
	for range resultCh {
		// Process results
	}
}

// OptimizedTreeNode demonstrates memory-efficient tree structure
type OptimizedTreeNode struct {
	Value    int
	Children []*OptimizedTreeNode
	// Remove large Data field, use external storage if needed
}

// TreeNodePool manages tree node allocation
var TreeNodePool = sync.Pool{
	New: func() interface{} {
		return &OptimizedTreeNode{}
	},
}

// createOptimizedTree creates memory-efficient tree structure
func createOptimizedTree(depth, branching int) *OptimizedTreeNode {
	if depth <= 0 {
		return nil
	}
	
	// Get node from pool
	node := TreeNodePool.Get().(*OptimizedTreeNode)
	node.Value = depth // Use depth as value to avoid random
	node.Children = make([]*OptimizedTreeNode, 0, branching)
	
	// Create children recursively
	for i := 0; i < branching; i++ {
		child := createOptimizedTree(depth-1, branching)
		if child != nil {
			node.Children = append(node.Children, child)
		}
	}
	
	return node
}

// releaseOptimizedTree returns nodes to pool
func releaseOptimizedTree(node *OptimizedTreeNode) {
	if node == nil {
		return
	}
	
	// Release children first
	for _, child := range node.Children {
		releaseOptimizedTree(child)
	}
	
	// Reset and return to pool
	node.Value = 0
	node.Children = node.Children[:0] // Keep capacity
	TreeNodePool.Put(node)
}

// temporaryAllocationsGood demonstrates efficient temporary allocation management
func temporaryAllocationsGood(n int) int {
	sum := 0
	
	// Reuse temporary allocations
	temp := make([]int, 1000)
	tempMap := make(map[int]string, 100)
	
	for i := 0; i < n; i++ {
		// Reuse slice, just reset length
		temp = temp[:1000]
		for j := range temp {
			temp[j] = i + j
		}
		
		// Clear map for reuse
		for k := range tempMap {
			delete(tempMap, k)
		}
		
		// Reuse map
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

// stringProcessingGood demonstrates efficient string processing
func stringProcessingGood(texts []string) []string {
	results := make([]string, 0, len(texts))
	
	// Pre-allocate builder
	var builder strings.Builder
	
	for _, text := range texts {
		// Reset builder for reuse
		builder.Reset()
		builder.Grow(len(text) + 100) // Estimate final size
		
		// Efficient string processing
		processed := strings.ToUpper(text)
		
		// Use replacer for multiple replacements
		replacer := strings.NewReplacer(" ", "_", "-", "_", ".", "_")
		processed = replacer.Replace(processed)
		
		// Use builder for concatenation
		builder.WriteString(processed)
		for i := 0; i < 10; i++ {
			builder.WriteString("_SUFFIX_")
			builder.WriteString(fmt.Sprintf("%d", i))
		}
		
		results = append(results, builder.String())
	}
	return results
}

// memoryOptimizedWorkload combines multiple memory-efficient operations
func memoryOptimizedWorkload(scale int) {
	// String concatenation
	_ = stringConcatenationGood(scale * 100)
	
	// Slice growth
	_ = sliceGrowthGood(scale * 1000)
	
	// Map allocations
	_ = mapAllocationsGood(scale * 50)
	
	// Interface allocations
	_ = interfaceAllocationsGood(scale * 200)
	
	// Closure allocations
	_ = closureAllocationsGood(scale * 10)
	
	// Channel allocations
	channelAllocationsGood(scale * 5)
	
	// Tree structures
	tree := createOptimizedTree(8, 3)
	releaseOptimizedTree(tree)
	
	// Temporary allocations
	_ = temporaryAllocationsGood(scale * 10)
	
	// String processing
	testData := generateTestData(scale * 20)
	_ = stringProcessingGood(testData)
	
	// Optimized memory manager
	manager := NewOptimizedMemoryManager()
	manager.EfficientMemoryUse(scale)
}

// BufferPool demonstrates object pooling for buffers
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	bp.pool.Put(buf)
}

// demonstrateBufferPooling shows efficient buffer reuse
func demonstrateBufferPooling(n int) {
	pool := NewBufferPool(1024)
	
	for i := 0; i < n; i++ {
		buf := pool.Get()
		
		// Use buffer
		for j := 0; j < len(buf); j++ {
			buf[j] = byte(i + j)
		}
		
		// Process buffer data
		_ = len(buf)
		
		// Return to pool
		pool.Put(buf)
	}
}

// Removed main function to avoid conflict with memory_intensive.go main function
// The main function is in memory_intensive.go which demonstrates both
// memory-intensive and memory-optimized operations