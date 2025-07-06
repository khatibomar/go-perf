package algorithms

import (
	"exercise-01-04/internal/datastructures"
)

// NaiveBinarySearch - Inefficient implementation without optimizations
func NaiveBinarySearch(arr []int, target int) bool {
	left, right := 0, len(arr)-1
	
	for left <= right {
		// Inefficient mid calculation (potential overflow)
		mid := (left + right) / 2
		
		// No branch prediction optimization
		if arr[mid] == target {
			return true
		} else if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
		
		// Unnecessary work in each iteration
		_ = performUnnecessaryWork()
	}
	
	return false
}

// OptimizedBinarySearch - Efficient implementation with caching and optimizations
func OptimizedBinarySearch(arr []int, target int, cache *datastructures.SearchCache) bool {
	// Check cache first
	if cache != nil {
		if result, found := cache.Get(target); found {
			return result
		}
	}
	
	left, right := 0, len(arr)-1
	
	for left <= right {
		// Overflow-safe mid calculation
		mid := left + (right-left)/2
		
		// Optimized comparison with branch prediction hints
		if arr[mid] == target {
			if cache != nil {
				cache.Set(target, true)
			}
			return true
		}
		
		if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	
	if cache != nil {
		cache.Set(target, false)
	}
	return false
}

// NaiveLinearSearch - Inefficient linear search
func NaiveLinearSearch(arr []int, target int) bool {
	for i := 0; i < len(arr); i++ {
		// Inefficient bounds checking
		if i >= 0 && i < len(arr) && arr[i] == target {
			return true
		}
		
		// Unnecessary work
		_ = performUnnecessaryWork()
	}
	return false
}

// OptimizedLinearSearch - Efficient linear search with loop unrolling
func OptimizedLinearSearch(arr []int, target int) bool {
	// Loop unrolling for better performance
	i := 0
	for i < len(arr)-3 {
		if arr[i] == target || arr[i+1] == target || arr[i+2] == target || arr[i+3] == target {
			return true
		}
		i += 4
	}
	
	// Handle remaining elements
	for i < len(arr) {
		if arr[i] == target {
			return true
		}
		i++
	}
	
	return false
}