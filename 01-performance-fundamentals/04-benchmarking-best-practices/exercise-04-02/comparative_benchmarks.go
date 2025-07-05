// Package main demonstrates comparative benchmarking techniques in Go
// This exercise covers algorithm comparison, optimization validation, and performance analysis
package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
)

// StringConcatenation demonstrates different string concatenation approaches
type StringConcatenation struct{}

// NaiveConcat uses simple string concatenation
func (sc *StringConcatenation) NaiveConcat(parts []string) string {
	result := ""
	for _, part := range parts {
		result += part
	}
	return result
}

// BuilderConcat uses strings.Builder for efficient concatenation
func (sc *StringConcatenation) BuilderConcat(parts []string) string {
	var builder strings.Builder
	for _, part := range parts {
		builder.WriteString(part)
	}
	return builder.String()
}

// PreallocatedBuilderConcat uses strings.Builder with preallocation
func (sc *StringConcatenation) PreallocatedBuilderConcat(parts []string) string {
	var builder strings.Builder
	// Estimate total length
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	builder.Grow(totalLen)
	
	for _, part := range parts {
		builder.WriteString(part)
	}
	return builder.String()
}

// JoinConcat uses strings.Join
func (sc *StringConcatenation) JoinConcat(parts []string) string {
	return strings.Join(parts, "")
}

// SliceConcat uses slice operations and conversion
func (sc *StringConcatenation) SliceConcat(parts []string) string {
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	
	result := make([]byte, 0, totalLen)
	for _, part := range parts {
		result = append(result, part...)
	}
	return string(result)
}

// SortingAlgorithms demonstrates different sorting approaches
type SortingAlgorithms struct{}

// BubbleSort implements bubble sort algorithm
func (sa *SortingAlgorithms) BubbleSort(data []int) {
	n := len(data)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if data[j] > data[j+1] {
				data[j], data[j+1] = data[j+1], data[j]
			}
		}
	}
}

// QuickSort implements quicksort algorithm
func (sa *SortingAlgorithms) QuickSort(data []int) {
	sa.quickSortRecursive(data, 0, len(data)-1)
}

func (sa *SortingAlgorithms) quickSortRecursive(data []int, low, high int) {
	if low < high {
		pi := sa.partition(data, low, high)
		sa.quickSortRecursive(data, low, pi-1)
		sa.quickSortRecursive(data, pi+1, high)
	}
}

func (sa *SortingAlgorithms) partition(data []int, low, high int) int {
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

// MergeSort implements merge sort algorithm
func (sa *SortingAlgorithms) MergeSort(data []int) {
	if len(data) <= 1 {
		return
	}
	
	mid := len(data) / 2
	left := make([]int, mid)
	right := make([]int, len(data)-mid)
	
	copy(left, data[:mid])
	copy(right, data[mid:])
	
	sa.MergeSort(left)
	sa.MergeSort(right)
	
	sa.merge(data, left, right)
}

func (sa *SortingAlgorithms) merge(data, left, right []int) {
	i, j, k := 0, 0, 0
	
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			data[k] = left[i]
			i++
		} else {
			data[k] = right[j]
			j++
		}
		k++
	}
	
	for i < len(left) {
		data[k] = left[i]
		i++
		k++
	}
	
	for j < len(right) {
		data[k] = right[j]
		j++
		k++
	}
}

// HeapSort implements heap sort algorithm
func (sa *SortingAlgorithms) HeapSort(data []int) {
	n := len(data)
	
	// Build max heap
	for i := n/2 - 1; i >= 0; i-- {
		sa.heapify(data, n, i)
	}
	
	// Extract elements from heap
	for i := n - 1; i > 0; i-- {
		data[0], data[i] = data[i], data[0]
		sa.heapify(data, i, 0)
	}
}

func (sa *SortingAlgorithms) heapify(data []int, n, i int) {
	largest := i
	left := 2*i + 1
	right := 2*i + 2
	
	if left < n && data[left] > data[largest] {
		largest = left
	}
	
	if right < n && data[right] > data[largest] {
		largest = right
	}
	
	if largest != i {
		data[i], data[largest] = data[largest], data[i]
		sa.heapify(data, n, largest)
	}
}

// StandardSort uses Go's built-in sort
func (sa *SortingAlgorithms) StandardSort(data []int) {
	sort.Ints(data)
}

// MapImplementations demonstrates different map access patterns
type MapImplementations struct{}

// SyncMapAccess uses sync.Map for concurrent access
func (mi *MapImplementations) SyncMapAccess(operations int, concurrency int) {
	var sm sync.Map
	var wg sync.WaitGroup
	
	// Pre-populate
	for i := 0; i < operations/2; i++ {
		sm.Store(fmt.Sprintf("key_%d", i), i)
	}
	
	worker := func() {
		defer wg.Done()
		for i := 0; i < operations/concurrency; i++ {
			key := fmt.Sprintf("key_%d", rand.Intn(operations))
			
			// Mix of operations
			switch rand.Intn(3) {
			case 0:
				sm.Load(key)
			case 1:
				sm.Store(key, rand.Intn(1000))
			case 2:
				sm.Delete(key)
			}
		}
	}
	
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	wg.Wait()
}

// MutexMapAccess uses regular map with mutex protection
func (mi *MapImplementations) MutexMapAccess(operations int, concurrency int) {
	m := make(map[string]int)
	var mu sync.RWMutex
	var wg sync.WaitGroup
	
	// Pre-populate
	for i := 0; i < operations/2; i++ {
		m[fmt.Sprintf("key_%d", i)] = i
	}
	
	worker := func() {
		defer wg.Done()
		for i := 0; i < operations/concurrency; i++ {
			key := fmt.Sprintf("key_%d", rand.Intn(operations))
			
			// Mix of operations
			switch rand.Intn(3) {
			case 0:
				mu.RLock()
				_ = m[key]
				mu.RUnlock()
			case 1:
				mu.Lock()
				m[key] = rand.Intn(1000)
				mu.Unlock()
			case 2:
				mu.Lock()
				delete(m, key)
				mu.Unlock()
			}
		}
	}
	
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	wg.Wait()
}

// ChannelMapAccess uses channels for map access coordination
func (mi *MapImplementations) ChannelMapAccess(operations int, concurrency int) {
	m := make(map[string]int)
	requests := make(chan mapRequest, 100)
	var wg sync.WaitGroup
	
	// Pre-populate
	for i := 0; i < operations/2; i++ {
		m[fmt.Sprintf("key_%d", i)] = i
	}
	
	// Map manager goroutine
	go func() {
		for req := range requests {
			switch req.operation {
			case "load":
				value, exists := m[req.key]
				req.response <- mapResponse{value: value, exists: exists}
			case "store":
				m[req.key] = req.value
				req.response <- mapResponse{}
			case "delete":
				delete(m, req.key)
				req.response <- mapResponse{}
			}
		}
	}()
	
	worker := func() {
		defer wg.Done()
		for i := 0; i < operations/concurrency; i++ {
			key := fmt.Sprintf("key_%d", rand.Intn(operations))
			response := make(chan mapResponse, 1)
			
			// Mix of operations
			switch rand.Intn(3) {
			case 0:
				requests <- mapRequest{operation: "load", key: key, response: response}
			case 1:
				requests <- mapRequest{operation: "store", key: key, value: rand.Intn(1000), response: response}
			case 2:
				requests <- mapRequest{operation: "delete", key: key, response: response}
			}
			<-response
		}
	}
	
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	wg.Wait()
	close(requests)
}

type mapRequest struct {
	operation string
	key       string
	value     int
	response  chan mapResponse
}

type mapResponse struct {
	value  int
	exists bool
}

// SliceOperations demonstrates different slice manipulation approaches
type SliceOperations struct{}

// AppendGrowth demonstrates slice growth patterns
func (so *SliceOperations) AppendGrowth(size int) []int {
	var result []int
	for i := 0; i < size; i++ {
		result = append(result, i)
	}
	return result
}

// PreallocatedSlice uses preallocation
func (so *SliceOperations) PreallocatedSlice(size int) []int {
	result := make([]int, 0, size)
	for i := 0; i < size; i++ {
		result = append(result, i)
	}
	return result
}

// DirectIndexSlice uses direct indexing
func (so *SliceOperations) DirectIndexSlice(size int) []int {
	result := make([]int, size)
	for i := 0; i < size; i++ {
		result[i] = i
	}
	return result
}

// CopySlice demonstrates slice copying
func (so *SliceOperations) CopySlice(source []int) []int {
	result := make([]int, len(source))
	copy(result, source)
	return result
}

// AppendSlice demonstrates slice appending
func (so *SliceOperations) AppendSlice(source []int) []int {
	var result []int
	result = append(result, source...)
	return result
}

// MemoryPooling demonstrates object pooling patterns
type MemoryPooling struct {
	bufferPool sync.Pool
	slicePool  sync.Pool
}

// NewMemoryPooling creates a new memory pooling instance
func NewMemoryPooling() *MemoryPooling {
	return &MemoryPooling{
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		},
		slicePool: sync.Pool{
			New: func() interface{} {
				return make([]int, 0, 100)
			},
		},
	}
}

// WithoutPooling performs operations without pooling
func (mp *MemoryPooling) WithoutPooling(operations int) {
	for i := 0; i < operations; i++ {
		// Allocate buffer
		buffer := make([]byte, 1024)
		// Use buffer
		for j := 0; j < len(buffer); j++ {
			buffer[j] = byte(j % 256)
		}
		// Buffer goes out of scope and becomes eligible for GC
	}
}

// WithPooling performs operations with pooling
func (mp *MemoryPooling) WithPooling(operations int) {
	for i := 0; i < operations; i++ {
		// Get buffer from pool
		buffer := mp.bufferPool.Get().([]byte)
		buffer = buffer[:1024] // Reset length
		
		// Use buffer
		for j := 0; j < len(buffer); j++ {
			buffer[j] = byte(j % 256)
		}
		
		// Return buffer to pool
		buffer = buffer[:0] // Reset length
		mp.bufferPool.Put(buffer)
	}
}

// ConcurrencyPatterns demonstrates different concurrency approaches
type ConcurrencyPatterns struct{}

// SequentialProcessing processes items sequentially
func (cp *ConcurrencyPatterns) SequentialProcessing(items []int) []int {
	results := make([]int, len(items))
	for i, item := range items {
		results[i] = cp.expensiveOperation(item)
	}
	return results
}

// WorkerPoolProcessing uses a worker pool pattern
func (cp *ConcurrencyPatterns) WorkerPoolProcessing(items []int, numWorkers int) []int {
	jobs := make(chan job, len(items))
	results := make(chan result, len(items))
	
	// Start workers
	for w := 0; w < numWorkers; w++ {
		go func() {
			for j := range jobs {
				value := cp.expensiveOperation(j.item)
				results <- result{index: j.index, value: value}
			}
		}()
	}
	
	// Send jobs
	for i, item := range items {
		jobs <- job{index: i, item: item}
	}
	close(jobs)
	
	// Collect results
	output := make([]int, len(items))
	for i := 0; i < len(items); i++ {
		r := <-results
		output[r.index] = r.value
	}
	
	return output
}

// GoroutinePerItemProcessing spawns a goroutine per item
func (cp *ConcurrencyPatterns) GoroutinePerItemProcessing(items []int) []int {
	results := make([]int, len(items))
	var wg sync.WaitGroup
	
	for i, item := range items {
		wg.Add(1)
		go func(index, value int) {
			defer wg.Done()
			results[index] = cp.expensiveOperation(value)
		}(i, item)
	}
	
	wg.Wait()
	return results
}

// PipelineProcessing uses a pipeline pattern
func (cp *ConcurrencyPatterns) PipelineProcessing(items []int, stages int) []int {
	// Create pipeline stages
	channels := make([]chan int, stages+1)
	for i := range channels {
		channels[i] = make(chan int, 10)
	}
	
	// Start pipeline stages
	for stage := 0; stage < stages; stage++ {
		go func(s int) {
			for item := range channels[s] {
				processed := cp.pipelineStage(item, s)
				channels[s+1] <- processed
			}
			close(channels[s+1])
		}(stage)
	}
	
	// Send items to first stage
	go func() {
		for _, item := range items {
			channels[0] <- item
		}
		close(channels[0])
	}()
	
	// Collect results from last stage
	var results []int
	for result := range channels[stages] {
		results = append(results, result)
	}
	
	return results
}

type job struct {
	index int
	item  int
}

type result struct {
	index int
	value int
}

// expensiveOperation simulates a CPU-intensive operation
func (cp *ConcurrencyPatterns) expensiveOperation(n int) int {
	// Simulate work
	sum := 0
	for i := 0; i < n%1000+100; i++ {
		sum += i * i
	}
	return sum
}

// pipelineStage simulates a pipeline stage operation
func (cp *ConcurrencyPatterns) pipelineStage(item, stage int) int {
	// Simulate stage-specific processing
	time.Sleep(time.Microsecond * time.Duration(stage+1))
	return item * (stage + 1)
}

// DataStructures demonstrates different data structure performance
type DataStructures struct{}

// SliceSearch performs linear search on slice
func (ds *DataStructures) SliceSearch(data []int, target int) bool {
	for _, item := range data {
		if item == target {
			return true
		}
	}
	return false
}

// MapLookup performs map lookup
func (ds *DataStructures) MapLookup(data map[int]bool, target int) bool {
	return data[target]
}

// BinarySearch performs binary search on sorted slice
func (ds *DataStructures) BinarySearch(data []int, target int) bool {
	left, right := 0, len(data)-1
	
	for left <= right {
		mid := (left + right) / 2
		if data[mid] == target {
			return true
		} else if data[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return false
}

// Utility functions for generating test data

// GenerateRandomStrings creates random strings for testing
func GenerateRandomStrings(count, length int) []string {
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = generateRandomString(length)
	}
	return result
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateRandomInts creates random integers for testing
func GenerateRandomInts(count, max int) []int {
	result := make([]int, count)
	for i := 0; i < count; i++ {
		result[i] = rand.Intn(max)
	}
	return result
}

// GenerateSortedInts creates sorted integers for testing
func GenerateSortedInts(count int) []int {
	result := make([]int, count)
	for i := 0; i < count; i++ {
		result[i] = i
	}
	return result
}

// CopySlice creates a copy of an integer slice
func CopySlice(source []int) []int {
	result := make([]int, len(source))
	copy(result, source)
	return result
}

func main() {
	fmt.Println("🔬 Comparative Benchmarking Demo")
	fmt.Println("================================")
	
	// Demonstrate string concatenation approaches
	fmt.Println("\n📝 String Concatenation Comparison:")
	sc := &StringConcatenation{}
	parts := GenerateRandomStrings(100, 10)
	
	start := time.Now()
	result1 := sc.NaiveConcat(parts)
	naiveTime := time.Since(start)
	
	start = time.Now()
	result2 := sc.BuilderConcat(parts)
	builderTime := time.Since(start)
	
	start = time.Now()
	result3 := sc.JoinConcat(parts)
	joinTime := time.Since(start)
	
	fmt.Printf("  Naive concat:     %v (len: %d)\n", naiveTime, len(result1))
	fmt.Printf("  Builder concat:   %v (len: %d)\n", builderTime, len(result2))
	fmt.Printf("  Join concat:      %v (len: %d)\n", joinTime, len(result3))
	fmt.Printf("  Builder speedup:  %.2fx\n", float64(naiveTime)/float64(builderTime))
	
	// Demonstrate sorting algorithm comparison
	fmt.Println("\n🔢 Sorting Algorithm Comparison:")
	sa := &SortingAlgorithms{}
	data := GenerateRandomInts(1000, 10000)
	
	// Test different sorting algorithms
	algorithms := []struct {
		name string
		fn func([]int)
	}{
		{"Bubble Sort", sa.BubbleSort},
		{"Quick Sort", sa.QuickSort},
		{"Merge Sort", sa.MergeSort},
		{"Heap Sort", sa.HeapSort},
		{"Standard Sort", sa.StandardSort},
	}
	
	for _, alg := range algorithms {
		testData := CopySlice(data)
		start := time.Now()
		alg.fn(testData)
		duration := time.Since(start)
		fmt.Printf("  %-15s: %v\n", alg.name, duration)
	}
	
	// Demonstrate concurrency pattern comparison
	fmt.Println("\n⚡ Concurrency Pattern Comparison:")
	cp := &ConcurrencyPatterns{}
	items := GenerateRandomInts(100, 1000)
	
	start = time.Now()
	result4 := cp.SequentialProcessing(items)
	seqTime := time.Since(start)
	
	start = time.Now()
	result5 := cp.WorkerPoolProcessing(items, 4)
	poolTime := time.Since(start)
	
	start = time.Now()
	result6 := cp.GoroutinePerItemProcessing(items)
	goroutineTime := time.Since(start)
	
	fmt.Printf("  Sequential:       %v (len: %d)\n", seqTime, len(result4))
	fmt.Printf("  Worker Pool:      %v (len: %d)\n", poolTime, len(result5))
	fmt.Printf("  Goroutine/Item:   %v (len: %d)\n", goroutineTime, len(result6))
	fmt.Printf("  Pool speedup:     %.2fx\n", float64(seqTime)/float64(poolTime))
	
	// Demonstrate memory pooling comparison
	fmt.Println("\n🧠 Memory Pooling Comparison:")
	mp := NewMemoryPooling()
	
	start = time.Now()
	mp.WithoutPooling(1000)
	withoutPoolTime := time.Since(start)
	
	start = time.Now()
	mp.WithPooling(1000)
	withPoolTime := time.Since(start)
	
	fmt.Printf("  Without pooling:  %v\n", withoutPoolTime)
	fmt.Printf("  With pooling:     %v\n", withPoolTime)
	fmt.Printf("  Pooling speedup:  %.2fx\n", float64(withoutPoolTime)/float64(withPoolTime))
	
	fmt.Println("\n✅ Comparative benchmarking demo completed!")
	fmt.Println("\nRun 'go test -bench=.' to see detailed benchmark results.")
}