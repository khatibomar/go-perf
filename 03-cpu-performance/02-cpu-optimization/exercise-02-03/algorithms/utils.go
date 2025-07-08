package algorithms

// insertionSortRange performs insertion sort on a range of the slice
func insertionSortRange(data []int, left, right int) {
	for i := left + 1; i <= right; i++ {
		key := data[i]
		j := i - 1
		for j >= left && data[j] > key {
			data[j+1] = data[j]
			j--
		}
		data[j+1] = key
	}
}

// insertionSort performs insertion sort on the entire slice
func insertionSort(data []int) {
	for i := 1; i < len(data); i++ {
		key := data[i]
		j := i - 1
		for j >= 0 && data[j] > key {
			data[j+1] = data[j]
			j--
		}
		data[j+1] = key
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// swap swaps two elements in a slice
func swap(data []int, i, j int) {
	data[i], data[j] = data[j], data[i]
}