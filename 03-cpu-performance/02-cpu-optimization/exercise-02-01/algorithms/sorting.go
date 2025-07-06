package algorithms

import (
	"math"
)

// BubbleSort implements the bubble sort algorithm (O(n²))
// This is intentionally inefficient for comparison purposes
func BubbleSort(arr []int) {
	n := len(arr)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if arr[j] > arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
			}
		}
	}
}

// QuickSort implements the quicksort algorithm (O(n log n) average, O(n²) worst)
func QuickSort(arr []int) {
	if len(arr) < 2 {
		return
	}
	quickSortHelper(arr, 0, len(arr)-1)
}

func quickSortHelper(arr []int, low, high int) {
	if low < high {
		pi := partition(arr, low, high)
		quickSortHelper(arr, low, pi-1)
		quickSortHelper(arr, pi+1, high)
	}
}

func partition(arr []int, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j] < pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// MergeSort implements the merge sort algorithm (O(n log n) guaranteed)
func MergeSort(arr []int) {
	if len(arr) < 2 {
		return
	}
	mergeSortHelper(arr, 0, len(arr)-1)
}

func mergeSortHelper(arr []int, left, right int) {
	if left < right {
		mid := left + (right-left)/2
		mergeSortHelper(arr, left, mid)
		mergeSortHelper(arr, mid+1, right)
		merge(arr, left, mid, right)
	}
}

func merge(arr []int, left, mid, right int) {
	n1 := mid - left + 1
	n2 := right - mid

	leftArr := make([]int, n1)
	rightArr := make([]int, n2)

	for i := 0; i < n1; i++ {
		leftArr[i] = arr[left+i]
	}
	for j := 0; j < n2; j++ {
		rightArr[j] = arr[mid+1+j]
	}

	i, j, k := 0, 0, left
	for i < n1 && j < n2 {
		if leftArr[i] <= rightArr[j] {
			arr[k] = leftArr[i]
			i++
		} else {
			arr[k] = rightArr[j]
			j++
		}
		k++
	}

	for i < n1 {
		arr[k] = leftArr[i]
		i++
		k++
	}

	for j < n2 {
		arr[k] = rightArr[j]
		j++
		k++
	}
}

// HeapSort implements the heap sort algorithm (O(n log n))
func HeapSort(arr []int) {
	n := len(arr)

	// Build max heap
	for i := n/2 - 1; i >= 0; i-- {
		heapify(arr, n, i)
	}

	// Extract elements from heap one by one
	for i := n - 1; i > 0; i-- {
		arr[0], arr[i] = arr[i], arr[0]
		heapify(arr, i, 0)
	}
}

func heapify(arr []int, n, i int) {
	largest := i
	left := 2*i + 1
	right := 2*i + 2

	if left < n && arr[left] > arr[largest] {
		largest = left
	}

	if right < n && arr[right] > arr[largest] {
		largest = right
	}

	if largest != i {
		arr[i], arr[largest] = arr[largest], arr[i]
		heapify(arr, n, largest)
	}
}

// RadixSort implements the radix sort algorithm (O(d*n) for integers)
// This is a non-comparison based sort that can be very efficient for integers
func RadixSort(arr []int) {
	if len(arr) == 0 {
		return
	}

	// Find the maximum number to know number of digits
	max := getMax(arr)

	// Do counting sort for every digit
	for exp := 1; max/exp > 0; exp *= 10 {
		countingSort(arr, exp)
	}
}

func getMax(arr []int) int {
	max := arr[0]
	for i := 1; i < len(arr); i++ {
		if arr[i] > max {
			max = arr[i]
		}
	}
	return max
}

func countingSort(arr []int, exp int) {
	n := len(arr)
	output := make([]int, n)
	count := make([]int, 10)

	// Store count of occurrences in count[]
	for i := 0; i < n; i++ {
		count[(arr[i]/exp)%10]++
	}

	// Change count[i] so that count[i] now contains actual
	// position of this digit in output[]
	for i := 1; i < 10; i++ {
		count[i] += count[i-1]
	}

	// Build the output array
	for i := n - 1; i >= 0; i-- {
		output[count[(arr[i]/exp)%10]-1] = arr[i]
		count[(arr[i]/exp)%10]--
	}

	// Copy the output array to arr[], so that arr[] now
	// contains sorted numbers according to current digit
	for i := 0; i < n; i++ {
		arr[i] = output[i]
	}
}

// InsertionSort implements insertion sort (O(n²) worst case, O(n) best case)
// Efficient for small datasets or nearly sorted data
func InsertionSort(arr []int) {
	for i := 1; i < len(arr); i++ {
		key := arr[i]
		j := i - 1

		// Move elements of arr[0..i-1], that are greater than key,
		// to one position ahead of their current position
		for j >= 0 && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
}

// SelectionSort implements selection sort (O(n²))
// Simple but inefficient algorithm for comparison
func SelectionSort(arr []int) {
	n := len(arr)
	for i := 0; i < n-1; i++ {
		minIdx := i
		for j := i + 1; j < n; j++ {
			if arr[j] < arr[minIdx] {
				minIdx = j
			}
		}
		arr[i], arr[minIdx] = arr[minIdx], arr[i]
	}
}

// TimSort is a hybrid stable sorting algorithm (used by Python and Java)
// Combines merge sort and insertion sort for optimal performance
func TimSort(arr []int) {
	n := len(arr)
	minRun := getMinRunLength(n)

	// Sort individual subarrays of size minRun using insertion sort
	for start := 0; start < n; start += minRun {
		end := Min(start+minRun-1, n-1)
		insertionSortRange(arr, start, end)
	}

	// Start merging from size minRun
	size := minRun
	for size < n {
		for start := 0; start < n; start += size * 2 {
			mid := start + size - 1
			end := Min(start+size*2-1, n-1)

			if mid < end {
				mergeRange(arr, start, mid, end)
			}
		}
		size *= 2
	}
}

func getMinRunLength(n int) int {
	r := 0
	for n >= 32 {
		r |= n & 1
		n >>= 1
	}
	return n + r
}

func insertionSortRange(arr []int, left, right int) {
	for i := left + 1; i <= right; i++ {
		key := arr[i]
		j := i - 1
		for j >= left && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
}

func mergeRange(arr []int, left, mid, right int) {
	n1 := mid - left + 1
	n2 := right - mid

	leftArr := make([]int, n1)
	rightArr := make([]int, n2)

	for i := 0; i < n1; i++ {
		leftArr[i] = arr[left+i]
	}
	for j := 0; j < n2; j++ {
		rightArr[j] = arr[mid+1+j]
	}

	i, j, k := 0, 0, left
	for i < n1 && j < n2 {
		if leftArr[i] <= rightArr[j] {
			arr[k] = leftArr[i]
			i++
		} else {
			arr[k] = rightArr[j]
			j++
		}
		k++
	}

	for i < n1 {
		arr[k] = leftArr[i]
		i++
		k++
	}

	for j < n2 {
		arr[k] = rightArr[j]
		j++
		k++
	}
}



// CountingSort implements counting sort (O(n+k) where k is the range)
// Efficient for sorting integers with a known, limited range
func CountingSort(arr []int, maxVal int) {
	n := len(arr)
	count := make([]int, maxVal+1)
	output := make([]int, n)

	// Store count of each element
	for i := 0; i < n; i++ {
		count[arr[i]]++
	}

	// Change count[i] so that count[i] now contains actual
	// position of this element in output array
	for i := 1; i <= maxVal; i++ {
		count[i] += count[i-1]
	}

	// Build the output array
	for i := n - 1; i >= 0; i-- {
		output[count[arr[i]]-1] = arr[i]
		count[arr[i]]--
	}

	// Copy the output array to original array
	for i := 0; i < n; i++ {
		arr[i] = output[i]
	}
}

// BucketSort implements bucket sort (O(n+k) average case)
// Efficient for uniformly distributed data
func BucketSort(arr []int, bucketCount int) {
	if len(arr) == 0 {
		return
	}

	// Find minimum and maximum values
	minVal, maxVal := arr[0], arr[0]
	for _, v := range arr {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Create buckets
	buckets := make([][]int, bucketCount)
	for i := range buckets {
		buckets[i] = make([]int, 0)
	}

	// Distribute elements into buckets
	bucketRange := float64(maxVal-minVal+1) / float64(bucketCount)
	for _, v := range arr {
		bucketIndex := int(math.Floor(float64(v-minVal) / bucketRange))
		if bucketIndex >= bucketCount {
			bucketIndex = bucketCount - 1
		}
		buckets[bucketIndex] = append(buckets[bucketIndex], v)
	}

	// Sort individual buckets and concatenate
	index := 0
	for i := 0; i < bucketCount; i++ {
		if len(buckets[i]) > 0 {
			InsertionSort(buckets[i])
			for _, v := range buckets[i] {
				arr[index] = v
				index++
			}
		}
	}
}