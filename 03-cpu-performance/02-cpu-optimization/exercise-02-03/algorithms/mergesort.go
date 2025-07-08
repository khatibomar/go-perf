package algorithms

// MergeSort implements the standard recursive merge sort algorithm
func MergeSort(data []int) {
	if len(data) <= 1 {
		return
	}
	
	aux := make([]int, len(data))
	mergeSortRecursive(data, aux, 0, len(data)-1)
}

// mergeSortRecursive performs recursive merge sort
func mergeSortRecursive(data, aux []int, low, high int) {
	if low >= high {
		return
	}
	
	mid := low + (high-low)/2
	mergeSortRecursive(data, aux, low, mid)
	mergeSortRecursive(data, aux, mid+1, high)
	merge(data, aux, low, mid, high)
}

// merge merges two sorted subarrays
func merge(data, aux []int, low, mid, high int) {
	// Copy data to auxiliary array
	for i := low; i <= high; i++ {
		aux[i] = data[i]
	}
	
	i, j := low, mid+1
	
	// Merge back to data array
	for k := low; k <= high; k++ {
		if i > mid {
			data[k] = aux[j]
			j++
		} else if j > high {
			data[k] = aux[i]
			i++
		} else if aux[j] < aux[i] {
			data[k] = aux[j]
			j++
		} else {
			data[k] = aux[i]
			i++
		}
	}
}

// MergeSortIterative implements bottom-up iterative merge sort
func MergeSortIterative(data []int) {
	n := len(data)
	if n <= 1 {
		return
	}
	
	aux := make([]int, n)
	
	// Bottom-up merge sort
	for size := 1; size < n; size *= 2 {
		for low := 0; low < n-size; low += 2 * size {
			mid := low + size - 1
			high := min(low+2*size-1, n-1)
			merge(data, aux, low, mid, high)
		}
	}
}

// MergeSortInPlace implements in-place merge sort (optimized for space)
func MergeSortInPlace(data []int) {
	if len(data) <= 1 {
		return
	}
	mergeSortInPlaceRecursive(data, 0, len(data)-1)
}

// mergeSortInPlaceRecursive performs in-place recursive merge sort
func mergeSortInPlaceRecursive(data []int, low, high int) {
	if low >= high {
		return
	}
	
	mid := low + (high-low)/2
	mergeSortInPlaceRecursive(data, low, mid)
	mergeSortInPlaceRecursive(data, mid+1, high)
	mergeInPlace(data, low, mid, high)
}

// mergeInPlace merges two sorted subarrays in-place
func mergeInPlace(data []int, low, mid, high int) {
	// If already sorted, no need to merge
	if data[mid] <= data[mid+1] {
		return
	}
	
	i, j := low, mid+1
	
	for i <= mid && j <= high {
		if data[i] <= data[j] {
			i++
		} else {
			// data[j] is smaller, need to insert it at position i
			value := data[j]
			index := j
			
			// Shift elements to make space
			for index != i {
				data[index] = data[index-1]
				index--
			}
			
			data[i] = value
			i++
			mid++
			j++
		}
	}
}

// MergeSortOptimized implements optimized merge sort with insertion sort for small arrays
func MergeSortOptimized(data []int) {
	if len(data) <= 1 {
		return
	}
	
	aux := make([]int, len(data))
	mergeSortOptimizedRecursive(data, aux, 0, len(data)-1)
}

// mergeSortOptimizedRecursive performs optimized recursive merge sort
func mergeSortOptimizedRecursive(data, aux []int, low, high int) {
	// Use insertion sort for small arrays
	if high-low+1 <= 7 {
		insertionSortRange(data, low, high)
		return
	}
	
	if low >= high {
		return
	}
	
	mid := low + (high-low)/2
	mergeSortOptimizedRecursive(data, aux, low, mid)
	mergeSortOptimizedRecursive(data, aux, mid+1, high)
	
	// Skip merge if already sorted
	if data[mid] <= data[mid+1] {
		return
	}
	
	merge(data, aux, low, mid, high)
}