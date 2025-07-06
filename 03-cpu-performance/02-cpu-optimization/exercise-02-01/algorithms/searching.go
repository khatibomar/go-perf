package algorithms

import (
	"math"
)

// LinearSearch implements linear search algorithm (O(n))
// Searches through the array sequentially
func LinearSearch(arr []int, target int) bool {
	for _, v := range arr {
		if v == target {
			return true
		}
	}
	return false
}

// LinearSearchIndex returns the index of the target element, -1 if not found
func LinearSearchIndex(arr []int, target int) int {
	for i, v := range arr {
		if v == target {
			return i
		}
	}
	return -1
}

// BinarySearch implements binary search algorithm (O(log n))
// Requires sorted array
func BinarySearch(arr []int, target int) bool {
	left, right := 0, len(arr)-1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			return true
		}

		if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return false
}

// BinarySearchIndex returns the index of the target element, -1 if not found
func BinarySearchIndex(arr []int, target int) int {
	left, right := 0, len(arr)-1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			return mid
		}

		if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return -1
}

// InterpolationSearch implements interpolation search (O(log log n) for uniform distribution)
// More efficient than binary search for uniformly distributed sorted arrays
func InterpolationSearch(arr []int, target int) bool {
	low, high := 0, len(arr)-1

	for low <= high && target >= arr[low] && target <= arr[high] {
		if low == high {
			return arr[low] == target
		}

		// Calculate position using interpolation formula
		pos := low + ((target-arr[low])*(high-low))/(arr[high]-arr[low])

		if arr[pos] == target {
			return true
		}

		if arr[pos] < target {
			low = pos + 1
		} else {
			high = pos - 1
		}
	}

	return false
}

// JumpSearch implements jump search algorithm (O(√n))
// Optimal jump size is √n
func JumpSearch(arr []int, target int) bool {
	n := len(arr)
	step := int(math.Sqrt(float64(n)))
	prev := 0

	// Finding the block where element is present
	for arr[Min(step, n)-1] < target {
		prev = step
		step += int(math.Sqrt(float64(n)))
		if prev >= n {
			return false
		}
	}

	// Linear search in the identified block
	for arr[prev] < target {
		prev++
		if prev == Min(step, n) {
			return false
		}
	}

	return arr[prev] == target
}

// ExponentialSearch implements exponential search (O(log n))
// Useful for unbounded or infinite arrays
func ExponentialSearch(arr []int, target int) bool {
	n := len(arr)
	if n == 0 {
		return false
	}

	if arr[0] == target {
		return true
	}

	// Find range for binary search
	i := 1
	for i < n && arr[i] <= target {
		i *= 2
	}

	// Binary search in the found range
	return binarySearchRange(arr, target, i/2, Min(i, n-1))
}

func binarySearchRange(arr []int, target, left, right int) bool {
	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			return true
		}

		if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return false
}

// TernarySearch implements ternary search (O(log₃ n))
// Divides array into three parts instead of two
func TernarySearch(arr []int, target int) bool {
	return ternarySearchHelper(arr, target, 0, len(arr)-1)
}

func ternarySearchHelper(arr []int, target, left, right int) bool {
	if right >= left {
		// Find mid points
		mid1 := left + (right-left)/3
		mid2 := right - (right-left)/3

		if arr[mid1] == target || arr[mid2] == target {
			return true
		}

		if target < arr[mid1] {
			return ternarySearchHelper(arr, target, left, mid1-1)
		} else if target > arr[mid2] {
			return ternarySearchHelper(arr, target, mid2+1, right)
		} else {
			return ternarySearchHelper(arr, target, mid1+1, mid2-1)
		}
	}
	return false
}

// FibonacciSearch implements Fibonacci search (O(log n))
// Uses Fibonacci numbers to divide the array
func FibonacciSearch(arr []int, target int) bool {
	n := len(arr)
	if n == 0 {
		return false
	}

	// Initialize Fibonacci numbers
	fibM2 := 0 // (m-2)'th Fibonacci number
	fibM1 := 1 // (m-1)'th Fibonacci number
	fibM := fibM2 + fibM1 // m'th Fibonacci number

	// Find the smallest Fibonacci number greater than or equal to n
	for fibM < n {
		fibM2 = fibM1
		fibM1 = fibM
		fibM = fibM2 + fibM1
	}

	// Marks the eliminated range from front
	offset := -1

	// While there are elements to be inspected
	for fibM > 1 {
		// Check if fibM2 is a valid location
		i := Min(offset+fibM2, n-1)

		if arr[i] < target {
			fibM = fibM1
			fibM1 = fibM2
			fibM2 = fibM - fibM1
			offset = i
		} else if arr[i] > target {
			fibM = fibM2
			fibM1 = fibM1 - fibM2
			fibM2 = fibM - fibM1
		} else {
			return true
		}
	}

	// Comparing the last element
	if fibM1 == 1 && offset+1 < n && arr[offset+1] == target {
		return true
	}

	return false
}

// SearchCache represents a simple cache for search results
type SearchCache struct {
	cache map[int]bool
	hits  int
	misses int
}

// NewSearchCache creates a new search cache
func NewSearchCache() *SearchCache {
	return &SearchCache{
		cache: make(map[int]bool),
	}
}

// Search performs a cached search
func (sc *SearchCache) Search(arr []int, target int) bool {
	// Check cache first
	if result, exists := sc.cache[target]; exists {
		sc.hits++
		return result
	}

	// Cache miss - perform actual search
	sc.misses++
	result := BinarySearch(arr, target)
	sc.cache[target] = result
	return result
}

// GetStats returns cache statistics
func (sc *SearchCache) GetStats() (hits, misses int) {
	return sc.hits, sc.misses
}

// ClearCache clears the cache
func (sc *SearchCache) ClearCache() {
	sc.cache = make(map[int]bool)
	sc.hits = 0
	sc.misses = 0
}

// HashSearch implements a hash-based search (O(1) average case)
type HashSearch struct {
	hashMap map[int]bool
}

// NewHashSearch creates a new hash search structure
func NewHashSearch(arr []int) *HashSearch {
	hashMap := make(map[int]bool)
	for _, v := range arr {
		hashMap[v] = true
	}
	return &HashSearch{hashMap: hashMap}
}

// Search performs hash-based search
func (hs *HashSearch) Search(target int) bool {
	return hs.hashMap[target]
}

// BloomFilter implements a simple Bloom filter for approximate membership testing
type BloomFilter struct {
	bitArray []bool
	size     int
	hashFuncs int
}

// NewBloomFilter creates a new Bloom filter
func NewBloomFilter(size, hashFuncs int) *BloomFilter {
	return &BloomFilter{
		bitArray: make([]bool, size),
		size:     size,
		hashFuncs: hashFuncs,
	}
}

// Add adds an element to the Bloom filter
func (bf *BloomFilter) Add(item int) {
	for i := 0; i < bf.hashFuncs; i++ {
		hash := bf.hash(item, i)
		bf.bitArray[hash] = true
	}
}

// Contains checks if an element might be in the set
func (bf *BloomFilter) Contains(item int) bool {
	for i := 0; i < bf.hashFuncs; i++ {
		hash := bf.hash(item, i)
		if !bf.bitArray[hash] {
			return false
		}
	}
	return true
}

// Simple hash function for demonstration
func (bf *BloomFilter) hash(item, seed int) int {
	hash := item + seed*31
	if hash < 0 {
		hash = -hash
	}
	return hash % bf.size
}

// BinarySearchFirst finds the first occurrence of target
func BinarySearchFirst(arr []int, target int) int {
	left, right := 0, len(arr)-1
	result := -1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			result = mid
			right = mid - 1 // Continue searching in left half
		} else if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return result
}

// BinarySearchLast finds the last occurrence of target
func BinarySearchLast(arr []int, target int) int {
	left, right := 0, len(arr)-1
	result := -1

	for left <= right {
		mid := left + (right-left)/2

		if arr[mid] == target {
			result = mid
			left = mid + 1 // Continue searching in right half
		} else if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return result
}

// SearchRange finds all occurrences of target in a sorted array
func SearchRange(arr []int, target int) (int, int) {
	first := BinarySearchFirst(arr, target)
	if first == -1 {
		return -1, -1
	}
	last := BinarySearchLast(arr, target)
	return first, last
}