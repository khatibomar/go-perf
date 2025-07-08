package search

// BinarySearch performs standard recursive binary search
func BinarySearch(data []int, target int) int {
	return binarySearchRecursive(data, target, 0, len(data)-1)
}

func binarySearchRecursive(data []int, target, left, right int) int {
	if left > right {
		return -1
	}

	mid := left + (right-left)/2
	if data[mid] == target {
		return mid
	} else if data[mid] < target {
		return binarySearchRecursive(data, target, mid+1, right)
	} else {
		return binarySearchRecursive(data, target, left, mid-1)
	}
}

// BinarySearchIterative performs iterative binary search
func BinarySearchIterative(data []int, target int) int {
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

// BinarySearchBranchless performs branchless binary search
func BinarySearchBranchless(data []int, target int) int {
	left, right := 0, len(data)

	for left < right {
		mid := left + (right-left)/2
		// Branchless comparison
		cmp := boolToInt(data[mid] < target)
		left = cmp*(mid+1) + (1-cmp)*left
		right = (1-cmp)*mid + cmp*right
	}

	if left < len(data) && data[left] == target {
		return left
	}
	return -1
}

// BinarySearchCacheFriendly performs cache-friendly binary search
func BinarySearchCacheFriendly(data []int, target int) int {
	blockSize := 64 // Cache line size in elements
	slice := data
	offset := 0

	// Use smaller block size for better cache utilization
	for len(slice) > blockSize {
		mid := len(slice) / 2
		if slice[mid] <= target {
			slice = slice[mid:]
			offset += mid
		} else {
			slice = slice[:mid]
		}
	}

	// Linear search in final block
	for i, v := range slice {
		if v == target {
			return offset + i
		}
	}
	return -1
}

// InterpolationSearch performs interpolation search
func InterpolationSearch(data []int, target int) int {
	left, right := 0, len(data)-1

	for left <= right && target >= data[left] && target <= data[right] {
		if left == right {
			if data[left] == target {
				return left
			}
			return -1
		}

		// Interpolation formula
		pos := left + int(float64(target-data[left])/float64(data[right]-data[left])*float64(right-left))

		// Ensure pos is within bounds
		if pos < left {
			pos = left
		} else if pos > right {
			pos = right
		}

		if data[pos] == target {
			return pos
		} else if data[pos] < target {
			left = pos + 1
		} else {
			right = pos - 1
		}
	}

	return -1
}

// ExponentialSearch performs exponential search
func ExponentialSearch(data []int, target int) int {
	n := len(data)
	if n == 0 {
		return -1
	}

	if data[0] == target {
		return 0
	}

	// Find range for binary search
	i := 1
	for i < n && data[i] <= target {
		i *= 2
	}

	// Perform binary search in found range
	left := i / 2
	right := min(i, n-1)

	return binarySearchRange(data, target, left, right)
}

// FibonacciSearch performs Fibonacci search
func FibonacciSearch(data []int, target int) int {
	n := len(data)
	if n == 0 {
		return -1
	}

	// Generate Fibonacci numbers
	fibM2 := 0 // (m-2)'th Fibonacci number
	fibM1 := 1 // (m-1)'th Fibonacci number
	fibM := fibM2 + fibM1 // m'th Fibonacci number

	// Find smallest Fibonacci number >= n
	for fibM < n {
		fibM2 = fibM1
		fibM1 = fibM
		fibM = fibM2 + fibM1
	}

	offset := -1

	for fibM > 1 {
		i := min(offset+fibM2, n-1)

		if data[i] < target {
			fibM = fibM1
			fibM1 = fibM2
			fibM2 = fibM - fibM1
			offset = i
		} else if data[i] > target {
			fibM = fibM2
			fibM1 = fibM1 - fibM2
			fibM2 = fibM - fibM1
		} else {
			return i
		}
	}

	if fibM1 == 1 && offset+1 < n && data[offset+1] == target {
		return offset + 1
	}

	return -1
}

// TernarySearch performs ternary search
func TernarySearch(data []int, target int) int {
	return ternarySearchRecursive(data, target, 0, len(data)-1)
}

func ternarySearchRecursive(data []int, target, left, right int) int {
	if left > right {
		return -1
	}

	mid1 := left + (right-left)/3
	mid2 := right - (right-left)/3

	if data[mid1] == target {
		return mid1
	}
	if data[mid2] == target {
		return mid2
	}

	if target < data[mid1] {
		return ternarySearchRecursive(data, target, left, mid1-1)
	} else if target > data[mid2] {
		return ternarySearchRecursive(data, target, mid2+1, right)
	} else {
		return ternarySearchRecursive(data, target, mid1+1, mid2-1)
	}
}

// Helper functions
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func binarySearchRange(data []int, target, left, right int) int {
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

// BinarySearchGeneric performs generic binary search
func BinarySearchGeneric[T comparable](data []T, target T, less func(T, T) bool) int {
	left, right := 0, len(data)-1

	for left <= right {
		mid := left + (right-left)/2
		if data[mid] == target {
			return mid
		} else if less(data[mid], target) {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return -1
}