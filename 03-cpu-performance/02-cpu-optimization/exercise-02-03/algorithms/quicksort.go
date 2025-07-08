package algorithms

import (
	"math/rand"
	"time"
)

// QuickSort implements the standard quicksort algorithm
func QuickSort(data []int) {
	if len(data) <= 1 {
		return
	}
	quickSortRecursive(data, 0, len(data)-1)
}

// quickSortRecursive performs recursive quicksort
func quickSortRecursive(data []int, low, high int) {
	if low < high {
		pivotIndex := partition(data, low, high)
		quickSortRecursive(data, low, pivotIndex-1)
		quickSortRecursive(data, pivotIndex+1, high)
	}
}

// partition partitions the array around a pivot
func partition(data []int, low, high int) int {
	pivot := data[high]
	i := low - 1
	
	for j := low; j < high; j++ {
		if data[j] <= pivot {
			i++
			data[i], data[j] = data[j], data[i]
		}
	}
	
	data[i+1], data[high] = data[high], data[i+1]
	return i + 1
}

// QuickSortOptimized implements optimized quicksort with better pivot selection
func QuickSortOptimized(data []int) {
	if len(data) <= 1 {
		return
	}
	quickSortOptimizedRecursive(data, 0, len(data)-1)
}

// quickSortOptimizedRecursive performs optimized recursive quicksort
func quickSortOptimizedRecursive(data []int, low, high int) {
	// Use insertion sort for small arrays
	if high-low+1 <= 10 {
		insertionSortRange(data, low, high)
		return
	}
	
	if low < high {
		pivotIndex := partitionOptimized(data, low, high)
		quickSortOptimizedRecursive(data, low, pivotIndex-1)
		quickSortOptimizedRecursive(data, pivotIndex+1, high)
	}
}

// partitionOptimized uses median-of-three pivot selection
func partitionOptimized(data []int, low, high int) int {
	// Median-of-three pivot selection
	mid := low + (high-low)/2
	if data[mid] < data[low] {
		data[low], data[mid] = data[mid], data[low]
	}
	if data[high] < data[low] {
		data[low], data[high] = data[high], data[low]
	}
	if data[high] < data[mid] {
		data[mid], data[high] = data[high], data[mid]
	}
	
	// Place median at end
	data[mid], data[high] = data[high], data[mid]
	
	return partition(data, low, high)
}

// QuickSortHybrid implements hybrid quicksort with insertion sort for small arrays
func QuickSortHybrid(data []int) {
	if len(data) <= 1 {
		return
	}
	quickSortHybridRecursive(data, 0, len(data)-1)
}

// quickSortHybridRecursive performs hybrid recursive quicksort
func quickSortHybridRecursive(data []int, low, high int) {
	// Use insertion sort for small arrays
	if high-low+1 <= 16 {
		insertionSortRange(data, low, high)
		return
	}
	
	if low < high {
		pivotIndex := partitionRandomized(data, low, high)
		quickSortHybridRecursive(data, low, pivotIndex-1)
		quickSortHybridRecursive(data, pivotIndex+1, high)
	}
}

// partitionRandomized uses randomized pivot selection
func partitionRandomized(data []int, low, high int) int {
	// Random pivot selection
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := low + rnd.Intn(high-low+1)
	data[randomIndex], data[high] = data[high], data[randomIndex]
	
	return partition(data, low, high)
}



// QuickSort3Way implements 3-way quicksort for handling duplicates efficiently
func QuickSort3Way(data []int) {
	if len(data) <= 1 {
		return
	}
	quickSort3WayRecursive(data, 0, len(data)-1)
}

// quickSort3WayRecursive performs 3-way recursive quicksort
func quickSort3WayRecursive(data []int, low, high int) {
	if low >= high {
		return
	}
	
	lt, gt := partition3Way(data, low, high)
	quickSort3WayRecursive(data, low, lt-1)
	quickSort3WayRecursive(data, gt+1, high)
}

// partition3Way partitions array into three parts: <pivot, =pivot, >pivot
func partition3Way(data []int, low, high int) (int, int) {
	pivot := data[low]
	i := low
	lt := low
	gt := high
	
	for i <= gt {
		if data[i] < pivot {
			data[lt], data[i] = data[i], data[lt]
			lt++
			i++
		} else if data[i] > pivot {
			data[i], data[gt] = data[gt], data[i]
			gt--
		} else {
			i++
		}
	}
	
	return lt, gt
}

// QuickSortIterative implements iterative quicksort to avoid stack overflow
func QuickSortIterative(data []int) {
	if len(data) <= 1 {
		return
	}
	
	// Create stack for storing subarray bounds
	stack := make([][2]int, 0, 64) // Pre-allocate reasonable capacity
	stack = append(stack, [2]int{0, len(data) - 1})
	
	for len(stack) > 0 {
		// Pop from stack
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		
		low, high := top[0], top[1]
		
		if low < high {
			pivotIndex := partitionOptimized(data, low, high)
			
			// Push subarrays to stack
			stack = append(stack, [2]int{low, pivotIndex - 1})
			stack = append(stack, [2]int{pivotIndex + 1, high})
		}
	}
}

// QuickSelect implements quickselect algorithm for finding kth smallest element
func QuickSelect(data []int, k int) int {
	if k < 0 || k >= len(data) {
		panic("k out of bounds")
	}
	
	return quickSelectRecursive(data, 0, len(data)-1, k)
}

// quickSelectRecursive performs recursive quickselect
func quickSelectRecursive(data []int, low, high, k int) int {
	if low == high {
		return data[low]
	}
	
	pivotIndex := partitionRandomized(data, low, high)
	
	if k == pivotIndex {
		return data[k]
	} else if k < pivotIndex {
		return quickSelectRecursive(data, low, pivotIndex-1, k)
	} else {
		return quickSelectRecursive(data, pivotIndex+1, high, k)
	}
}