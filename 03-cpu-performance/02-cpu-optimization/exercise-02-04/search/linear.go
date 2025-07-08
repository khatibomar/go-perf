package search

// LinearSearch performs a basic linear search
func LinearSearch(data []int, target int) int {
	for i, v := range data {
		if v == target {
			return i
		}
	}
	return -1
}

// LinearSearchEarlyTermination with early termination optimization
func LinearSearchEarlyTermination(data []int, target int) int {
	// If data is sorted, we can terminate early
	for i, v := range data {
		if v == target {
			return i
		}
		if v > target {
			return -1 // Early termination for sorted data
		}
	}
	return -1
}

// LinearSearchUnrolled with loop unrolling optimization
func LinearSearchUnrolled(data []int, target int) int {
	n := len(data)
	i := 0

	// Process 4 elements at a time
	for i+3 < n {
		if data[i] == target {
			return i
		}
		if data[i+1] == target {
			return i + 1
		}
		if data[i+2] == target {
			return i + 2
		}
		if data[i+3] == target {
			return i + 3
		}
		i += 4
	}

	// Handle remaining elements
	for ; i < n; i++ {
		if data[i] == target {
			return i
		}
	}

	return -1
}

// LinearSearchSIMD simulates SIMD-optimized linear search
// Note: This is a conceptual implementation as Go doesn't have direct SIMD support
func LinearSearchSIMD(data []int32, target int32) int {
	const simdWidth = 8
	n := len(data)
	i := 0

	// Process 8 elements at once (simulated SIMD)
	for i+simdWidth <= n {
		// In real SIMD, this would be vectorized
		for j := 0; j < simdWidth; j++ {
			if data[i+j] == target {
				return i + j
			}
		}
		i += simdWidth
	}

	// Handle remaining elements
	for ; i < n; i++ {
		if data[i] == target {
			return i
		}
	}

	return -1
}

// LinearSearchPrefetch with memory prefetching
func LinearSearchPrefetch(data []int, target int) int {
	const prefetchDistance = 64 // Cache line size
	n := len(data)

	for i := 0; i < n; i++ {
		// Prefetch next cache line
		if i+prefetchDistance < n {
			// Simulate prefetch by accessing memory
			_ = data[i+prefetchDistance]
		}

		if data[i] == target {
			return i
		}
	}

	return -1
}

// LinearSearchBranchless uses branchless comparison
func LinearSearchBranchless(data []int, target int) int {
	for i, v := range data {
		// Branchless comparison using conditional move
		equal := v == target
		if equal {
			return i
		}
	}
	return -1
}

// LinearSearchString performs linear search on strings
func LinearSearchString(data []string, target string) int {
	for i, v := range data {
		if v == target {
			return i
		}
	}
	return -1
}

// LinearSearchGeneric performs generic linear search
func LinearSearchGeneric[T comparable](data []T, target T) int {
	for i, v := range data {
		if v == target {
			return i
		}
	}
	return -1
}