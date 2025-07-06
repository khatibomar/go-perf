package algorithms

import (
	"math/rand"
	"time"
)

// GenerateRandomData creates a slice of random integers
func GenerateRandomData(size int) []int {
	rand.Seed(time.Now().UnixNano())
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	return data
}

// NaiveQuickSort - Inefficient implementation with poor pivot selection
func NaiveQuickSort(arr []int) {
	if len(arr) < 2 {
		return
	}
	
	// Always choose first element as pivot (poor choice)
	pivot := arr[0]
	var left, right []int
	
	// Inefficient partitioning with extra allocations
	for i := 1; i < len(arr); i++ {
		if arr[i] <= pivot {
			left = append(left, arr[i])
		} else {
			right = append(right, arr[i])
		}
		// Simulate unnecessary work
		performUnnecessaryWork()
	}
	
	NaiveQuickSort(left)
	NaiveQuickSort(right)
	
	// Copy back (inefficient)
	copy(arr, append(append(left, pivot), right...))
}

// OptimizedQuickSort - Efficient in-place implementation
func OptimizedQuickSort(arr []int) {
	quickSortOptimized(arr, 0, len(arr)-1)
}

func quickSortOptimized(arr []int, low, high int) {
	if low < high {
		// Use median-of-three pivot selection
		pi := partitionOptimized(arr, low, high)
		
		// Recursively sort elements before and after partition
		quickSortOptimized(arr, low, pi-1)
		quickSortOptimized(arr, pi+1, high)
	}
}

func partitionOptimized(arr []int, low, high int) int {
	// Median-of-three pivot selection
	mid := low + (high-low)/2
	if arr[mid] < arr[low] {
		arr[low], arr[mid] = arr[mid], arr[low]
	}
	if arr[high] < arr[low] {
		arr[low], arr[high] = arr[high], arr[low]
	}
	if arr[high] < arr[mid] {
		arr[mid], arr[high] = arr[high], arr[mid]
	}
	arr[mid], arr[high] = arr[high], arr[mid]
	
	pivot := arr[high]
	i := low - 1
	
	for j := low; j < high; j++ {
		if arr[j] <= pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// NaiveMergeSort - Inefficient implementation with excessive allocations
func NaiveMergeSort(arr []int) []int {
	if len(arr) <= 1 {
		return arr
	}
	
	mid := len(arr) / 2
	
	// Create new slices (inefficient memory usage)
	left := make([]int, mid)
	right := make([]int, len(arr)-mid)
	copy(left, arr[:mid])
	copy(right, arr[mid:])
	
	// Simulate unnecessary work
	for i := 0; i < 10; i++ {
		performUnnecessaryWork()
	}
	
	left = NaiveMergeSort(left)
	right = NaiveMergeSort(right)
	
	return naiveMerge(left, right)
}

func naiveMerge(left, right []int) []int {
	result := make([]int, 0, len(left)+len(right))
	i, j := 0, 0
	
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			result = append(result, left[i])
			i++
		} else {
			result = append(result, right[j])
			j++
		}
		// Unnecessary work in merge
		performUnnecessaryWork()
	}
	
	// Append remaining elements
	result = append(result, left[i:]...)
	result = append(result, right[j:]...)
	
	return result
}

// OptimizedMergeSort - Efficient in-place implementation
func OptimizedMergeSort(arr []int) {
	if len(arr) <= 1 {
		return
	}
	aux := make([]int, len(arr))
	mergeSortOptimized(arr, aux, 0, len(arr)-1)
}

func mergeSortOptimized(arr, aux []int, low, high int) {
	if low >= high {
		return
	}
	
	// Use insertion sort for small arrays
	if high-low < 10 {
		insertionSort(arr, low, high)
		return
	}
	
	mid := low + (high-low)/2
	mergeSortOptimized(arr, aux, low, mid)
	mergeSortOptimized(arr, aux, mid+1, high)
	
	// Skip merge if already sorted
	if arr[mid] <= arr[mid+1] {
		return
	}
	
	mergeOptimized(arr, aux, low, mid, high)
}

func mergeOptimized(arr, aux []int, low, mid, high int) {
	// Copy to auxiliary array
	for i := low; i <= high; i++ {
		aux[i] = arr[i]
	}
	
	i, j, k := low, mid+1, low
	
	for k <= high {
		if i > mid {
			arr[k] = aux[j]
			j++
		} else if j > high {
			arr[k] = aux[i]
			i++
		} else if aux[j] < aux[i] {
			arr[k] = aux[j]
			j++
		} else {
			arr[k] = aux[i]
			i++
		}
		k++
	}
}

func insertionSort(arr []int, low, high int) {
	for i := low + 1; i <= high; i++ {
		key := arr[i]
		j := i - 1
		for j >= low && arr[j] > key {
			arr[j+1] = arr[j]
			j--
		}
		arr[j+1] = key
	}
}

func performUnnecessaryWork() int {
	// Simulate unnecessary computation
	sum := 0
	for i := 0; i < 10; i++ {
		sum += i * i
	}
	return sum
}