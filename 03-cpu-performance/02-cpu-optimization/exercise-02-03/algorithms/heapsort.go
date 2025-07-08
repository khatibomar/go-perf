package algorithms

// HeapSort implements the standard heap sort algorithm
func HeapSort(data []int) {
	n := len(data)
	if n <= 1 {
		return
	}
	
	// Build max heap
	for i := n/2 - 1; i >= 0; i-- {
		heapify(data, n, i)
	}
	
	// Extract elements from heap one by one
	for i := n - 1; i > 0; i-- {
		// Move current root to end
		data[0], data[i] = data[i], data[0]
		
		// Call heapify on the reduced heap
		heapify(data, i, 0)
	}
}

// heapify maintains the heap property for a subtree rooted at index i
func heapify(data []int, n, i int) {
	largest := i
	left := 2*i + 1
	right := 2*i + 2
	
	// If left child is larger than root
	if left < n && data[left] > data[largest] {
		largest = left
	}
	
	// If right child is larger than largest so far
	if right < n && data[right] > data[largest] {
		largest = right
	}
	
	// If largest is not root
	if largest != i {
		data[i], data[largest] = data[largest], data[i]
		
		// Recursively heapify the affected sub-tree
		heapify(data, n, largest)
	}
}

// HeapSortOptimized implements optimized heap sort with iterative heapify
func HeapSortOptimized(data []int) {
	n := len(data)
	if n <= 1 {
		return
	}
	
	// Build max heap using bottom-up approach
	for i := n/2 - 1; i >= 0; i-- {
		heapifyIterative(data, n, i)
	}
	
	// Extract elements from heap one by one
	for i := n - 1; i > 0; i-- {
		// Move current root to end
		data[0], data[i] = data[i], data[0]
		
		// Call heapify on the reduced heap
		heapifyIterative(data, i, 0)
	}
}

// heapifyIterative maintains heap property iteratively (avoids recursion overhead)
func heapifyIterative(data []int, n, start int) {
	parent := start
	
	for {
		largest := parent
		left := 2*parent + 1
		right := 2*parent + 2
		
		// Find largest among parent and children
		if left < n && data[left] > data[largest] {
			largest = left
		}
		
		if right < n && data[right] > data[largest] {
			largest = right
		}
		
		// If parent is largest, heap property is satisfied
		if largest == parent {
			break
		}
		
		// Swap and continue with the child
		data[parent], data[largest] = data[largest], data[parent]
		parent = largest
	}
}

// HeapSortWithInsertionSort implements hybrid heap sort with insertion sort for small arrays
func HeapSortWithInsertionSort(data []int) {
	n := len(data)
	if n <= 1 {
		return
	}
	
	// Use insertion sort for small arrays
	if n <= 16 {
		insertionSort(data)
		return
	}
	
	// Build max heap
	for i := n/2 - 1; i >= 0; i-- {
		heapifyIterative(data, n, i)
	}
	
	// Extract elements from heap until we reach threshold
	for i := n - 1; i > 16; i-- {
		// Move current root to end
		data[0], data[i] = data[i], data[0]
		
		// Call heapify on the reduced heap
		heapifyIterative(data, i, 0)
	}
	
	// Use insertion sort for the remaining small portion
	insertionSort(data[:17])
}



// BuildMaxHeap builds a max heap from an unsorted array
func BuildMaxHeap(data []int) {
	n := len(data)
	for i := n/2 - 1; i >= 0; i-- {
		heapifyIterative(data, n, i)
	}
}

// ExtractMax extracts the maximum element from a max heap
func ExtractMax(data []int, heapSize int) (int, int) {
	if heapSize < 1 {
		return 0, 0
	}
	
	max := data[0]
	data[0] = data[heapSize-1]
	heapifyIterative(data, heapSize-1, 0)
	
	return max, heapSize - 1
}

// HeapSortBottomUp implements bottom-up heap sort for better cache performance
func HeapSortBottomUp(data []int) {
	n := len(data)
	if n <= 1 {
		return
	}
	
	// Build heap bottom-up
	buildHeapBottomUp(data)
	
	// Extract elements
	for i := n - 1; i > 0; i-- {
		data[0], data[i] = data[i], data[0]
		siftDown(data, 0, i)
	}
}

// buildHeapBottomUp builds heap using bottom-up approach for better cache locality
func buildHeapBottomUp(data []int) {
	n := len(data)
	for i := n/2 - 1; i >= 0; i-- {
		siftDown(data, i, n)
	}
}

// siftDown maintains heap property by sifting element down
func siftDown(data []int, start, end int) {
	root := start
	
	for root*2+1 < end {
		child := root*2 + 1
		swap := root
		
		if data[swap] < data[child] {
			swap = child
		}
		
		if child+1 < end && data[swap] < data[child+1] {
			swap = child + 1
		}
		
		if swap == root {
			return
		}
		
		data[root], data[swap] = data[swap], data[root]
		root = swap
	}
}