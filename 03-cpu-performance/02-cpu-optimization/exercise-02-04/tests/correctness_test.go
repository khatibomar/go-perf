package tests

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"exercise-02-04/search"
)

// TestData holds test data for various scenarios
type TestData struct {
	Name        string
	Data        []int
	Sorted      bool
	Target      int
	ExpectedIdx int
	ShouldFind  bool
}

// generateTestCases creates comprehensive test cases
func generateTestCases() []TestData {
	testCases := []TestData{
		// Basic cases
		{"Empty array", []int{}, true, 5, -1, false},
		{"Single element - found", []int{42}, true, 42, 0, true},
		{"Single element - not found", []int{42}, true, 24, -1, false},
		
		// Small arrays
		{"Small sorted - first", []int{1, 3, 5, 7, 9}, true, 1, 0, true},
		{"Small sorted - middle", []int{1, 3, 5, 7, 9}, true, 5, 2, true},
		{"Small sorted - last", []int{1, 3, 5, 7, 9}, true, 9, 4, true},
		{"Small sorted - not found", []int{1, 3, 5, 7, 9}, true, 6, -1, false},
		
		{"Small unsorted - found", []int{9, 1, 5, 3, 7}, false, 5, 2, true},
		{"Small unsorted - not found", []int{9, 1, 5, 3, 7}, false, 6, -1, false},
		
		// Edge cases
		{"All same elements", []int{5, 5, 5, 5, 5}, true, 5, 0, true}, // Any index is valid
		{"Two elements - first", []int{1, 2}, true, 1, 0, true},
		{"Two elements - second", []int{1, 2}, true, 2, 1, true},
		{"Two elements - not found", []int{1, 2}, true, 3, -1, false},
	}
	
	// Generate larger test cases
	rand.Seed(42)
	
	// Large sorted array
	largeSorted := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		largeSorted[i] = i * 2 // Even numbers 0, 2, 4, ..., 1998
	}
	testCases = append(testCases, TestData{"Large sorted - found", largeSorted, true, 500, 250, true})
	testCases = append(testCases, TestData{"Large sorted - not found", largeSorted, true, 501, -1, false})
	
	// Large unsorted array
	largeUnsorted := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		largeUnsorted[i] = rand.Intn(2000)
	}
	// Ensure we have a known element
	largeUnsorted[500] = 12345
	testCases = append(testCases, TestData{"Large unsorted - found", largeUnsorted, false, 12345, 500, true})
	testCases = append(testCases, TestData{"Large unsorted - not found", largeUnsorted, false, 99999, -1, false})
	
	return testCases
}

// TestLinearSearchAlgorithms tests all linear search variants
func TestLinearSearchAlgorithms(t *testing.T) {
	testCases := generateTestCases()
	
	linearSearchFuncs := map[string]func([]int, int) int{
		"LinearSearch":         search.LinearSearch,
		"LinearSearchUnrolled": search.LinearSearchUnrolled,
		"LinearSearchBranchless": search.LinearSearchBranchless,
		"LinearSearchPrefetch": search.LinearSearchPrefetch,
		"LinearSearchSIMD": func(data []int, target int) int {
			// Convert []int to []int32 for SIMD function
			data32 := make([]int32, len(data))
			for i, v := range data {
				data32[i] = int32(v)
			}
			return search.LinearSearchSIMD(data32, int32(target))
		},
	}
	
	// LinearSearchEarlyTermination only works on sorted data
	linearSearchEarlyTerminationFunc := search.LinearSearchEarlyTermination
	
	for funcName, searchFunc := range linearSearchFuncs {
		t.Run(funcName, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.Name, func(t *testing.T) {
					idx := searchFunc(tc.Data, tc.Target)
					found := idx != -1
					
					if found != tc.ShouldFind {
						t.Errorf("%s: expected found=%v, got found=%v", funcName, tc.ShouldFind, found)
					}
					
					if tc.ShouldFind {
						if idx < 0 || idx >= len(tc.Data) {
							t.Errorf("%s: invalid index %d for array length %d", funcName, idx, len(tc.Data))
						} else if tc.Data[idx] != tc.Target {
							t.Errorf("%s: data[%d]=%d, expected %d", funcName, idx, tc.Data[idx], tc.Target)
						}
					} else {
						if idx != -1 {
							t.Errorf("%s: expected index -1 when not found, got %d", funcName, idx)
						}
					}
				})
			}
		})
	}
	
	// Test LinearSearchEarlyTermination only on sorted data
	t.Run("LinearSearchEarlyTermination", func(t *testing.T) {
		for _, tc := range testCases {
			if tc.Sorted { // Only test on sorted data
				t.Run(tc.Name, func(t *testing.T) {
					idx := linearSearchEarlyTerminationFunc(tc.Data, tc.Target)
					found := idx != -1
					
					if found != tc.ShouldFind {
						t.Errorf("LinearSearchEarlyTermination: expected found=%v, got found=%v", tc.ShouldFind, found)
					}
					
					if tc.ShouldFind {
						if idx < 0 || idx >= len(tc.Data) {
							t.Errorf("LinearSearchEarlyTermination: invalid index %d for array length %d", idx, len(tc.Data))
						} else if tc.Data[idx] != tc.Target {
							t.Errorf("LinearSearchEarlyTermination: data[%d]=%d, expected %d", idx, tc.Data[idx], tc.Target)
						}
					} else {
						if idx != -1 {
							t.Errorf("LinearSearchEarlyTermination: expected index -1 when not found, got %d", idx)
						}
					}
				})
			}
		}
	})
}

// TestBinarySearchAlgorithms tests all binary search variants
func TestBinarySearchAlgorithms(t *testing.T) {
	testCases := generateTestCases()
	
	// Filter to only sorted test cases for binary search
	sortedTestCases := make([]TestData, 0)
	for _, tc := range testCases {
		if tc.Sorted {
			sortedTestCases = append(sortedTestCases, tc)
		}
	}
	
	binarySearchFuncs := map[string]func([]int, int) int{
		"BinarySearch":              search.BinarySearch,
		"BinarySearchIterative":     search.BinarySearchIterative,
		"BinarySearchBranchless":    search.BinarySearchBranchless,
		"BinarySearchCacheFriendly": search.BinarySearchCacheFriendly,
		"InterpolationSearch":       search.InterpolationSearch,
		"ExponentialSearch":         search.ExponentialSearch,
		"FibonacciSearch":           search.FibonacciSearch,
		"TernarySearch":             search.TernarySearch,
	}
	
	for funcName, searchFunc := range binarySearchFuncs {
		t.Run(funcName, func(t *testing.T) {
			for _, tc := range sortedTestCases {
				t.Run(tc.Name, func(t *testing.T) {
					idx := searchFunc(tc.Data, tc.Target)
					found := idx != -1
					
					if found != tc.ShouldFind {
						t.Errorf("%s: expected found=%v, got found=%v", funcName, tc.ShouldFind, found)
					}
					
					if tc.ShouldFind {
						if idx < 0 || idx >= len(tc.Data) {
							t.Errorf("%s: invalid index %d for array length %d", funcName, idx, len(tc.Data))
						} else if tc.Data[idx] != tc.Target {
							t.Errorf("%s: data[%d]=%d, expected %d", funcName, idx, tc.Data[idx], tc.Target)
						}
					} else {
						if idx != -1 {
							t.Errorf("%s: expected index -1 when not found, got %d", funcName, idx)
						}
					}
				})
			}
		})
	}
}

// TestHashTableAlgorithms tests hash table implementations
func TestHashTableAlgorithms(t *testing.T) {
	testData := []int{1, 5, 10, 15, 20, 25, 30}
	testTargets := []struct {
		value      int
		shouldFind bool
	}{
		{1, true},
		{15, true},
		{30, true},
		{7, false},
		{100, false},
	}
	
	hashTables := map[string]func() interface{}{
		"HashTable": func() interface{} {
			ht := search.NewHashTable()
			for _, val := range testData {
				ht.Insert(strconv.Itoa(val), val)
			}
			return ht
		},
		"OptimizedHashTable": func() interface{} {
			ht := search.NewOptimizedHashTable()
			for _, val := range testData {
				ht.Insert(strconv.Itoa(val), val)
			}
			return ht
		},
		"RobinHoodHashTable": func() interface{} {
			ht := search.NewRobinHoodHashTable()
			for _, val := range testData {
				ht.Insert(strconv.Itoa(val), val)
			}
			return ht
		},
		"CuckooHashTable": func() interface{} {
			ht := search.NewCuckooHashTable()
			for _, val := range testData {
				ht.Insert(strconv.Itoa(val), val)
			}
			return ht
		},
	}
	
	for tableName, createTable := range hashTables {
		t.Run(tableName, func(t *testing.T) {
			table := createTable()
			
			for _, target := range testTargets {
				t.Run(strconv.Itoa(target.value), func(t *testing.T) {
					var found bool
					
					switch ht := table.(type) {
					case *search.HashTable:
						_, found = ht.Search(strconv.Itoa(target.value))
					case *search.OptimizedHashTable:
						_, found = ht.Search(strconv.Itoa(target.value))
					case *search.RobinHoodHashTable:
						_, found = ht.Search(strconv.Itoa(target.value))
					case *search.CuckooHashTable:
						_, found = ht.Search(strconv.Itoa(target.value))
					}
					
					if found != target.shouldFind {
						t.Errorf("%s: expected found=%v for value %d, got found=%v",
							tableName, target.shouldFind, target.value, found)
					}
				})
			}
		})
	}
}

// TestTreeAlgorithms tests tree-based search algorithms
func TestTreeAlgorithms(t *testing.T) {
	testData := []int{5, 3, 7, 1, 9, 2, 8, 4, 6}
	testTargets := []struct {
		value      int
		shouldFind bool
	}{
		{1, true},
		{5, true},
		{9, true},
		{10, false},
		{0, false},
	}
	
	t.Run("BST", func(t *testing.T) {
		bst := search.NewBST()
		for _, val := range testData {
			bst.Insert(val)
		}
		
		for _, target := range testTargets {
			t.Run(strconv.Itoa(target.value), func(t *testing.T) {
				found := bst.Search(target.value)
				if found != target.shouldFind {
					t.Errorf("BST: expected found=%v for value %d, got found=%v",
						target.shouldFind, target.value, found)
				}
			})
		}
	})
	
	t.Run("BTree", func(t *testing.T) {
		btree := search.NewBTree(3) // Minimum degree 3
		for _, val := range testData {
			btree.Insert(val)
		}
		
		for _, target := range testTargets {
			t.Run(strconv.Itoa(target.value), func(t *testing.T) {
				found := btree.Search(target.value)
				if found != target.shouldFind {
					t.Errorf("BTree: expected found=%v for value %d, got found=%v",
						target.shouldFind, target.value, found)
				}
			})
		}
	})
}

// TestStringSearchAlgorithms tests string-based search algorithms
func TestStringSearchAlgorithms(t *testing.T) {
	testWords := []string{"apple", "banana", "cherry", "date", "elderberry"}
	testTargets := []struct {
		word       string
		shouldFind bool
	}{
		{"apple", true},
		{"cherry", true},
		{"elderberry", true},
		{"grape", false},
		{"appl", false},
	}
	
	t.Run("LinearSearchString", func(t *testing.T) {
		for _, target := range testTargets {
				t.Run(target.word, func(t *testing.T) {
					idx := search.LinearSearchString(testWords, target.word)
					found := idx != -1
					if found != target.shouldFind {
						t.Errorf("LinearSearchString: expected found=%v for word %s, got found=%v",
							target.shouldFind, target.word, found)
					}
					if target.shouldFind && (idx < 0 || idx >= len(testWords) || testWords[idx] != target.word) {
						t.Errorf("LinearSearchString: invalid result for word %s", target.word)
					}
				})
			}
	})
	
	t.Run("Trie", func(t *testing.T) {
		trie := search.NewTrie()
		for i, word := range testWords {
			trie.Insert(word, i)
		}
		
		for _, target := range testTargets {
			t.Run(target.word, func(t *testing.T) {
				_, found := trie.Search(target.word)
				if found != target.shouldFind {
					t.Errorf("Trie: expected found=%v for word %s, got found=%v",
						target.shouldFind, target.word, found)
				}
			})
		}
		
		// Test prefix functionality
		if !trie.StartsWith("app") {
			t.Error("Trie: should find prefix 'app'")
		}
		if trie.StartsWith("xyz") {
			t.Error("Trie: should not find prefix 'xyz'")
		}
	})
	
	t.Run("CompressedTrie", func(t *testing.T) {
		compTrie := search.NewCompressedTrie()
		for i, word := range testWords {
			compTrie.Insert(word, i)
		}
		
		for _, target := range testTargets {
			t.Run(target.word, func(t *testing.T) {
				_, found := compTrie.Search(target.word)
				if found != target.shouldFind {
					t.Errorf("CompressedTrie: expected found=%v for word %s, got found=%v",
						target.shouldFind, target.word, found)
				}
			})
		}
	})
}

// TestSuffixTree tests suffix tree functionality
func TestSuffixTree(t *testing.T) {
	text := "banana"
	suffixTree := search.NewSuffixTree(text)
	
	testPatterns := []struct {
		pattern    string
		shouldFind bool
		expectedCount int
	}{
		{"a", true, 3},     // "a" appears 3 times
		{"an", true, 2},    // "an" appears 2 times
		{"ana", true, 2},   // "ana" appears 2 times
		{"banana", true, 1}, // "banana" appears 1 time
		{"xyz", false, 0},  // "xyz" doesn't appear
	}
	
	for _, test := range testPatterns {
		t.Run(test.pattern, func(t *testing.T) {
			positions := suffixTree.Search(test.pattern)
			found := len(positions) > 0
			
			if found != test.shouldFind {
				t.Errorf("SuffixTree: expected found=%v for pattern %s, got found=%v",
					test.shouldFind, test.pattern, found)
			}
			
			if test.shouldFind && len(positions) != test.expectedCount {
				t.Errorf("SuffixTree: expected %d occurrences of %s, got %d",
					test.expectedCount, test.pattern, len(positions))
			}
		})
	}
}

// TestGenericSearchAlgorithms tests generic search functions
func TestGenericSearchAlgorithms(t *testing.T) {
	t.Run("LinearSearchGeneric", func(t *testing.T) {
		// Test with integers
		intData := []int{1, 3, 5, 7, 9}
		idx := search.LinearSearchGeneric(intData, 5)
		found := idx != -1
		if !found || idx != 2 {
			t.Errorf("LinearSearchGeneric: expected found=true, idx=2, got found=%v, idx=%d", found, idx)
		}
		
		// Test with strings
		stringData := []string{"apple", "banana", "cherry"}
		idx = search.LinearSearchGeneric(stringData, "banana")
		found = idx != -1
		if !found || idx != 1 {
			t.Errorf("LinearSearchGeneric: expected found=true, idx=1, got found=%v, idx=%d", found, idx)
		}
	})
	
	t.Run("BinarySearchGeneric", func(t *testing.T) {
		// Test with integers
		intData := []int{1, 3, 5, 7, 9}
		idx := search.BinarySearchGeneric(intData, 5, func(a, b int) bool {
			return a < b
		})
		found := idx != -1
		if !found || idx != 2 {
			t.Errorf("BinarySearchGeneric: expected found=true, idx=2, got found=%v, idx=%d", found, idx)
		}
		
		// Test with strings
		stringData := []string{"apple", "banana", "cherry"}
		idx = search.BinarySearchGeneric(stringData, "banana", func(a, b string) bool {
			return a < b
		})
		found = idx != -1
		if !found || idx != 1 {
			t.Errorf("BinarySearchGeneric: expected found=true, idx=1, got found=%v, idx=%d", found, idx)
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("NilSlice", func(t *testing.T) {
		var nilSlice []int
		idx := search.LinearSearch(nilSlice, 5)
		found := idx != -1
		if found || idx != -1 {
			t.Errorf("LinearSearch on nil slice: expected found=false, idx=-1, got found=%v, idx=%d", found, idx)
		}
	})
	
	t.Run("LargeNumbers", func(t *testing.T) {
		data := []int{1000000, 2000000, 3000000}
		idx := search.LinearSearch(data, 2000000)
		found := idx != -1
		if !found || idx != 1 {
			t.Errorf("LinearSearch with large numbers: expected found=true, idx=1, got found=%v, idx=%d", found, idx)
		}
	})
	
	t.Run("NegativeNumbers", func(t *testing.T) {
		data := []int{-10, -5, 0, 5, 10}
		idx := search.BinarySearchIterative(data, -5)
		found := idx != -1
		if !found || idx != 1 {
			t.Errorf("BinarySearch with negative numbers: expected found=true, idx=1, got found=%v, idx=%d", found, idx)
		}
	})
	
	t.Run("DuplicateElements", func(t *testing.T) {
		data := []int{1, 2, 2, 2, 3}
		idx := search.BinarySearchIterative(data, 2)
		found := idx != -1
		if !found {
			t.Errorf("BinarySearch with duplicates: expected found=true, got found=%v", found)
		}
		// Any of the indices 1, 2, 3 would be valid
		if idx < 1 || idx > 3 || data[idx] != 2 {
			t.Errorf("BinarySearch with duplicates: invalid index %d", idx)
		}
	})
}

// BenchmarkCorrectness runs performance tests to ensure algorithms scale correctly
func BenchmarkCorrectness(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		data := make([]int, size)
		for i := 0; i < size; i++ {
			data[i] = i
		}
		
		b.Run(fmt.Sprintf("LinearSearch_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				target := i % size
				search.LinearSearch(data, target)
			}
		})
		
		b.Run(fmt.Sprintf("BinarySearch_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				target := i % size
				search.BinarySearchIterative(data, target)
			}
		})
	}
}

// TestConsistency ensures all algorithms return consistent results
func TestConsistency(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	
	// Generate random test data
	for trial := 0; trial < 10; trial++ {
		size := 100 + rand.Intn(900) // 100-1000 elements
		data := make([]int, size)
		for i := 0; i < size; i++ {
			data[i] = rand.Intn(size * 2)
		}
		
		// Test unsorted data with linear search algorithms
		target := data[rand.Intn(size)] // Ensure target exists
		
		linearResults := make(map[string]bool)
		linearFuncs := map[string]func([]int, int) int{
			"LinearSearch":          search.LinearSearch,
			"LinearSearchUnrolled":   search.LinearSearchUnrolled,
			"LinearSearchBranchless": search.LinearSearchBranchless,
		}
		
		for name, fn := range linearFuncs {
			idx := fn(data, target)
			found := idx != -1
			linearResults[name] = found
		}
		
		// All linear search variants should return the same result
		expectedFound := linearResults["LinearSearch"]
		for name, found := range linearResults {
			if found != expectedFound {
				t.Errorf("Trial %d: Inconsistent result from %s: expected %v, got %v",
					trial, name, expectedFound, found)
			}
		}
		
		// Test sorted data with binary search algorithms
		sort.Ints(data)
		target = data[rand.Intn(size)] // Ensure target exists in sorted data
		
		binaryResults := make(map[string]bool)
		binaryFuncs := map[string]func([]int, int) int{
			"BinarySearch":           search.BinarySearch,
			"BinarySearchIterative":  search.BinarySearchIterative,
			"BinarySearchBranchless": search.BinarySearchBranchless,
			"InterpolationSearch":    search.InterpolationSearch,
			"ExponentialSearch":      search.ExponentialSearch,
		}
		
		for name, fn := range binaryFuncs {
			idx := fn(data, target)
			found := idx != -1
			binaryResults[name] = found
		}
		
		// All binary search variants should return the same result
		expectedFound = binaryResults["BinarySearch"]
		for name, found := range binaryResults {
			if found != expectedFound {
				t.Errorf("Trial %d: Inconsistent result from %s: expected %v, got %v",
					trial, name, expectedFound, found)
			}
		}
	}
}