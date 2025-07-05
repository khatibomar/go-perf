package benchmarks

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Validation tests to ensure benchmark implementations are correct

// Test string operations
func TestStringOperations(t *testing.T) {
	strs := []string{"hello", "world", "test"}
	expected := "helloworldtest"
	
	// Test string concatenation
	var result1 string
	for _, s := range strs {
		result1 += s
	}
	if result1 != expected {
		t.Errorf("String concatenation failed: got %s, want %s", result1, expected)
	}
	
	// Test string builder
	var builder strings.Builder
	for _, s := range strs {
		builder.WriteString(s)
	}
	result2 := builder.String()
	if result2 != expected {
		t.Errorf("String builder failed: got %s, want %s", result2, expected)
	}
	
	// Test string join
	result3 := strings.Join(strs, "")
	if result3 != expected {
		t.Errorf("String join failed: got %s, want %s", result3, expected)
	}
}

// Test sorting algorithms
func TestSortingAlgorithms(t *testing.T) {
	original := []int{64, 34, 25, 12, 22, 11, 90}
	expected := []int{11, 12, 22, 25, 34, 64, 90}
	
	// Test bubble sort
	data1 := make([]int, len(original))
	copy(data1, original)
	bubbleSort(data1)
	if !reflect.DeepEqual(data1, expected) {
		t.Errorf("Bubble sort failed: got %v, want %v", data1, expected)
	}
	
	// Test quick sort
	data2 := make([]int, len(original))
	copy(data2, original)
	quickSort(data2, 0, len(data2)-1)
	if !reflect.DeepEqual(data2, expected) {
		t.Errorf("Quick sort failed: got %v, want %v", data2, expected)
	}
	
	// Test standard sort
	data3 := make([]int, len(original))
	copy(data3, original)
	sort.Ints(data3)
	if !reflect.DeepEqual(data3, expected) {
		t.Errorf("Standard sort failed: got %v, want %v", data3, expected)
	}
}

// Test search algorithms
func TestSearchAlgorithms(t *testing.T) {
	data := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	target := 7
	expectedIndex := 3
	
	// Test linear search
	result1 := linearSearch(data, target)
	if result1 != expectedIndex {
		t.Errorf("Linear search failed: got %d, want %d", result1, expectedIndex)
	}
	
	// Test binary search
	result2 := binarySearch(data, target)
	if result2 != expectedIndex {
		t.Errorf("Binary search failed: got %d, want %d", result2, expectedIndex)
	}
	
	// Test standard binary search
	result3 := sort.SearchInts(data, target)
	if result3 != expectedIndex {
		t.Errorf("Standard binary search failed: got %d, want %d", result3, expectedIndex)
	}
	
	// Test search for non-existent element
	nonExistent := 8
	result4 := linearSearch(data, nonExistent)
	if result4 != -1 {
		t.Errorf("Linear search should return -1 for non-existent element, got %d", result4)
	}
	
	result5 := binarySearch(data, nonExistent)
	if result5 != -1 {
		t.Errorf("Binary search should return -1 for non-existent element, got %d", result5)
	}
}

// Test stack operations
func TestStackOperations(t *testing.T) {
	stack := &Stack{}
	
	// Test empty stack
	if stack.Size() != 0 {
		t.Errorf("Empty stack size should be 0, got %d", stack.Size())
	}
	
	if result := stack.Pop(); result != nil {
		t.Errorf("Pop from empty stack should return nil, got %v", result)
	}
	
	// Test push and pop operations
	values := []int{1, 2, 3, 4, 5}
	
	// Push values
	for _, v := range values {
		stack.Push(v)
	}
	
	if stack.Size() != len(values) {
		t.Errorf("Stack size should be %d, got %d", len(values), stack.Size())
	}
	
	// Pop values (should be in reverse order)
	for i := len(values) - 1; i >= 0; i-- {
		result := stack.Pop()
		if result != values[i] {
			t.Errorf("Pop should return %d, got %v", values[i], result)
		}
	}
	
	if stack.Size() != 0 {
		t.Errorf("Stack should be empty after popping all elements, size: %d", stack.Size())
	}
}

// Test queue operations
func TestQueueOperations(t *testing.T) {
	queue := &Queue{}
	
	// Test empty queue
	if result := queue.Dequeue(); result != nil {
		t.Errorf("Dequeue from empty queue should return nil, got %v", result)
	}
	
	// Test enqueue and dequeue operations
	values := []int{1, 2, 3, 4, 5}
	
	// Enqueue values
	for _, v := range values {
		queue.Enqueue(v)
	}
	
	// Dequeue values (should be in same order)
	for _, expected := range values {
		result := queue.Dequeue()
		if result != expected {
			t.Errorf("Dequeue should return %d, got %v", expected, result)
		}
	}
	
	// Test circular queue
	circularQueue := NewCircularQueue(3)
	
	// Fill queue to capacity
	for i := 0; i < 3; i++ {
		if !circularQueue.Enqueue(i) {
			t.Errorf("Enqueue should succeed for element %d", i)
		}
	}
	
	// Try to enqueue when full
	if circularQueue.Enqueue(99) {
		t.Error("Enqueue should fail when queue is full")
	}
	
	// Dequeue and verify order
	for i := 0; i < 3; i++ {
		result := circularQueue.Dequeue()
		if result != i {
			t.Errorf("Circular queue dequeue should return %d, got %v", i, result)
		}
	}
	
	// Test dequeue from empty circular queue
	if result := circularQueue.Dequeue(); result != nil {
		t.Errorf("Dequeue from empty circular queue should return nil, got %v", result)
	}
}

// Test LRU cache operations
func TestLRUCacheOperations(t *testing.T) {
	cache := NewLRUCache(3)
	
	// Test empty cache
	if _, exists := cache.Get("key1"); exists {
		t.Error("Get from empty cache should return false")
	}
	
	// Test put and get
	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3")
	
	if value, exists := cache.Get("key1"); !exists || value != "value1" {
		t.Errorf("Get should return value1, got %v, exists: %v", value, exists)
	}
	
	// Test LRU eviction
	cache.Put("key4", "value4") // Should evict key2 (least recently used)
	
	if _, exists := cache.Get("key2"); exists {
		t.Error("key2 should have been evicted")
	}
	
	if value, exists := cache.Get("key1"); !exists || value != "value1" {
		t.Error("key1 should still exist (was accessed recently)")
	}
	
	// Test update existing key
	cache.Put("key1", "updated_value1")
	if value, exists := cache.Get("key1"); !exists || value != "updated_value1" {
		t.Errorf("Updated value should be updated_value1, got %v", value)
	}
}

// Test binary tree operations
func TestBinaryTreeOperations(t *testing.T) {
	root := &TreeNode{Value: 50}
	values := []int{30, 70, 20, 40, 60, 80}
	
	// Insert values
	for _, v := range values {
		root.Insert(v)
	}
	
	// Test search for existing values
	for _, v := range append(values, 50) {
		if !root.Search(v) {
			t.Errorf("Search should find value %d", v)
		}
	}
	
	// Test search for non-existing values
	nonExisting := []int{10, 25, 35, 55, 75, 90}
	for _, v := range nonExisting {
		if root.Search(v) {
			t.Errorf("Search should not find value %d", v)
		}
	}
	
	// Test inorder traversal
	var result []int
	root.InorderTraversal(&result)
	expected := []int{20, 30, 40, 50, 60, 70, 80}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Inorder traversal failed: got %v, want %v", result, expected)
	}
}

// Test data processing pipeline
func TestDataProcessingPipeline(t *testing.T) {
	records := []DataRecord{
		{ID: 1, Name: "record1", Value: 10.5, Tags: []string{"tag1", "tag2"}},
		{ID: 2, Name: "record2", Value: 20.5, Tags: []string{"tag2", "tag3"}},
		{ID: 3, Name: "record3", Value: 30.5, Tags: []string{"tag1", "tag3"}},
	}
	
	result := processDataRecords(records)
	
	if result.TotalRecords != 3 {
		t.Errorf("Total records should be 3, got %d", result.TotalRecords)
	}
	
	expectedAverage := (10.5 + 20.5 + 30.5) / 3
	if result.AverageValue != expectedAverage {
		t.Errorf("Average value should be %f, got %f", expectedAverage, result.AverageValue)
	}
	
	if result.MaxValue != 30.5 {
		t.Errorf("Max value should be 30.5, got %f", result.MaxValue)
	}
	
	if result.MinValue != 10.5 {
		t.Errorf("Min value should be 10.5, got %f", result.MinValue)
	}
	
	// Check tag counts
	expectedTagCounts := map[string]int{
		"tag1": 2,
		"tag2": 2,
		"tag3": 2,
	}
	
	for tag, expectedCount := range expectedTagCounts {
		if count, exists := result.TagCounts[tag]; !exists || count != expectedCount {
			t.Errorf("Tag %s count should be %d, got %d", tag, expectedCount, count)
		}
	}
}

// Test in-memory database operations
func TestInMemoryDatabaseOperations(t *testing.T) {
	db := NewInMemoryDB()
	tableName := "test_table"
	
	// Test table creation
	db.CreateTable(tableName)
	
	// Test insert
	err := db.Insert(tableName, "key1", "value1")
	if err != nil {
		t.Errorf("Insert should succeed, got error: %v", err)
	}
	
	// Test select
	value, err := db.Select(tableName, "key1")
	if err != nil {
		t.Errorf("Select should succeed, got error: %v", err)
	}
	if value != "value1" {
		t.Errorf("Select should return value1, got %v", value)
	}
	
	// Test update
	err = db.Update(tableName, "key1", "updated_value1")
	if err != nil {
		t.Errorf("Update should succeed, got error: %v", err)
	}
	
	value, err = db.Select(tableName, "key1")
	if err != nil {
		t.Errorf("Select after update should succeed, got error: %v", err)
	}
	if value != "updated_value1" {
		t.Errorf("Select should return updated_value1, got %v", value)
	}
	
	// Test select all
	db.Insert(tableName, "key2", "value2")
	db.Insert(tableName, "key3", "value3")
	
	allData, err := db.SelectAll(tableName)
	if err != nil {
		t.Errorf("SelectAll should succeed, got error: %v", err)
	}
	if len(allData) != 3 {
		t.Errorf("SelectAll should return 3 items, got %d", len(allData))
	}
	
	// Test delete
	err = db.Delete(tableName, "key2")
	if err != nil {
		t.Errorf("Delete should succeed, got error: %v", err)
	}
	
	_, err = db.Select(tableName, "key2")
	if err == nil {
		t.Error("Select for deleted key should fail")
	}
	
	// Test operations on non-existent table
	err = db.Insert("non_existent", "key", "value")
	if err == nil {
		t.Error("Insert on non-existent table should fail")
	}
	
	_, err = db.Select("non_existent", "key")
	if err == nil {
		t.Error("Select on non-existent table should fail")
	}
}

// Test utility functions
func TestUtilityFunctions(t *testing.T) {
	// Test random data generation
	randomBytes := GenerateRandomBytes(100)
	if len(randomBytes) != 100 {
		t.Errorf("GenerateRandomBytes should return 100 bytes, got %d", len(randomBytes))
	}
	
	randomStrings := GenerateRandomStrings(5, 10)
	if len(randomStrings) != 5 {
		t.Errorf("GenerateRandomStrings should return 5 strings, got %d", len(randomStrings))
	}
	for _, s := range randomStrings {
		if len(s) != 10 {
			t.Errorf("Each random string should be 10 characters, got %d", len(s))
		}
	}
	
	randomInts := GenerateRandomInts(10, 100)
	if len(randomInts) != 10 {
		t.Errorf("GenerateRandomInts should return 10 integers, got %d", len(randomInts))
	}
	for _, i := range randomInts {
		if i < 0 || i >= 100 {
			t.Errorf("Random int should be between 0 and 99, got %d", i)
		}
	}
	
	sequentialInts := GenerateSequentialInts(5)
	expected := []int{0, 1, 2, 3, 4}
	if !reflect.DeepEqual(sequentialInts, expected) {
		t.Errorf("GenerateSequentialInts failed: got %v, want %v", sequentialInts, expected)
	}
	
	randomMap := GenerateRandomMap(3)
	if len(randomMap) != 3 {
		t.Errorf("GenerateRandomMap should return map with 3 entries, got %d", len(randomMap))
	}
}

// Test timer functionality
func TestTimerFunctionality(t *testing.T) {
	timer := NewTimer()
	
	timer.Start()
	time.Sleep(10 * time.Millisecond)
	timer.Stop()
	
	duration := timer.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("Timer duration should be at least 10ms, got %v", duration)
	}
	if duration > 50*time.Millisecond {
		t.Errorf("Timer duration should be less than 50ms, got %v", duration)
	}
	
	// Test MeasureFunction
	duration2 := MeasureFunction(func() {
		time.Sleep(5 * time.Millisecond)
	})
	if duration2 < 5*time.Millisecond {
		t.Errorf("MeasureFunction duration should be at least 5ms, got %v", duration2)
	}
	
	// Test MeasureFunctionWithResult
	duration3, result := MeasureFunctionWithResult(func() interface{} {
		time.Sleep(5 * time.Millisecond)
		return "test_result"
	})
	if duration3 < 5*time.Millisecond {
		t.Errorf("MeasureFunctionWithResult duration should be at least 5ms, got %v", duration3)
	}
	if result != "test_result" {
		t.Errorf("MeasureFunctionWithResult should return test_result, got %v", result)
	}
}

// Test benchmark result validation
func TestBenchmarkResultValidation(t *testing.T) {
	// Valid result
	validResult := BenchmarkResult{
		Name:        "TestBenchmark",
		Iterations:  1000,
		NsPerOp:     1000,
		Duration:    time.Second,
	}
	issues := ValidateBenchmarkResult(validResult)
	if len(issues) != 0 {
		t.Errorf("Valid result should have no issues, got: %v", issues)
	}
	
	// Invalid result - zero iterations
	invalidResult1 := BenchmarkResult{
		Name:        "TestBenchmark",
		Iterations:  0,
		NsPerOp:     1000,
		Duration:    time.Second,
	}
	issues1 := ValidateBenchmarkResult(invalidResult1)
	if len(issues1) == 0 {
		t.Error("Invalid result with zero iterations should have issues")
	}
	
	// Invalid result - zero ns/op
	invalidResult2 := BenchmarkResult{
		Name:        "TestBenchmark",
		Iterations:  1000,
		NsPerOp:     0,
		Duration:    time.Second,
	}
	issues2 := ValidateBenchmarkResult(invalidResult2)
	if len(issues2) == 0 {
		t.Error("Invalid result with zero ns/op should have issues")
	}
}

// Test statistical calculations
func TestStatisticalCalculations(t *testing.T) {
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}
	
	mean, median, min, max, _ := CalculateStats(durations)
	
	if mean != 30*time.Millisecond {
		t.Errorf("Mean should be 30ms, got %v", mean)
	}
	
	if median != 30*time.Millisecond {
		t.Errorf("Median should be 30ms, got %v", median)
	}
	
	if min != 10*time.Millisecond {
		t.Errorf("Min should be 10ms, got %v", min)
	}
	
	if max != 50*time.Millisecond {
		t.Errorf("Max should be 50ms, got %v", max)
	}
}

// Test memory statistics
func TestMemoryStatistics(t *testing.T) {
	// Get initial memory stats
	before := GetMemStats()
	
	// Allocate some memory
	data := make([][]byte, 1000)
	for i := range data {
		data[i] = make([]byte, 1024)
	}
	
	// Get memory stats after allocation
	after := GetMemStats()
	
	// Calculate difference
	diff := MemDiff(before, after)
	
	// Verify that memory allocation increased
	if diff.Alloc == 0 {
		t.Error("Memory allocation should have increased")
	}
	
	if diff.Mallocs == 0 {
		t.Error("Number of mallocs should have increased")
	}
	
	// Test memory formatting
	formatted := FormatMemStats(diff)
	if !strings.Contains(formatted, "Alloc:") {
		t.Error("Formatted memory stats should contain Alloc field")
	}
	
	// Keep reference to prevent optimization
	_ = data
}