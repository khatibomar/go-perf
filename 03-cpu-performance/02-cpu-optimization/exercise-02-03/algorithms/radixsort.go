package algorithms

import (
	"math"
)

// RadixSortLSD implements Least Significant Digit radix sort
func RadixSortLSD(data []int) {
	if len(data) <= 1 {
		return
	}
	
	// Handle negative numbers by finding min value
	minVal := data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
	}
	
	// Shift all values to make them non-negative
	if minVal < 0 {
		for i := range data {
			data[i] -= minVal
		}
	}
	
	// Find maximum value to determine number of digits
	maxVal := data[0]
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	
	// Perform counting sort for each digit
	for exp := 1; maxVal/exp > 0; exp *= 10 {
		countingSortByDigit(data, exp)
	}
	
	// Shift values back if we had negative numbers
	if minVal < 0 {
		for i := range data {
			data[i] += minVal
		}
	}
}

// countingSortByDigit performs counting sort based on a specific digit
func countingSortByDigit(data []int, exp int) {
	n := len(data)
	output := make([]int, n)
	count := make([]int, 10)
	
	// Count occurrences of each digit
	for i := 0; i < n; i++ {
		digit := (data[i] / exp) % 10
		count[digit]++
	}
	
	// Change count[i] to actual position of this digit in output
	for i := 1; i < 10; i++ {
		count[i] += count[i-1]
	}
	
	// Build output array
	for i := n - 1; i >= 0; i-- {
		digit := (data[i] / exp) % 10
		output[count[digit]-1] = data[i]
		count[digit]--
	}
	
	// Copy output array back to data
	copy(data, output)
}

// RadixSortMSD implements Most Significant Digit radix sort
func RadixSortMSD(data []int) {
	if len(data) <= 1 {
		return
	}
	
	// Handle negative numbers
	minVal := data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
	}
	
	if minVal < 0 {
		for i := range data {
			data[i] -= minVal
		}
	}
	
	// Find maximum value to determine number of digits
	maxVal := data[0]
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	
	// Calculate the highest power of 10
	maxDigits := 0
	temp := maxVal
	for temp > 0 {
		maxDigits++
		temp /= 10
	}
	
	if maxDigits == 0 {
		maxDigits = 1
	}
	
	// Start MSD radix sort
	aux := make([]int, len(data))
	radixSortMSDRecursive(data, aux, 0, len(data)-1, maxDigits-1)
	
	// Shift values back
	if minVal < 0 {
		for i := range data {
			data[i] += minVal
		}
	}
}

// radixSortMSDRecursive performs recursive MSD radix sort
func radixSortMSDRecursive(data, aux []int, low, high, digit int) {
	if low >= high || digit < 0 {
		return
	}
	
	// Use insertion sort for small subarrays
	if high-low+1 <= 10 {
		insertionSortRange(data, low, high)
		return
	}
	
	count := make([]int, 11) // 0-9 digits + 1 for indexing
	exp := int(math.Pow10(digit))
	
	// Count occurrences
	for i := low; i <= high; i++ {
		digitVal := (data[i] / exp) % 10
		count[digitVal+1]++
	}
	
	// Transform counts to indices
	for i := 0; i < 10; i++ {
		count[i+1] += count[i]
	}
	
	// Distribute elements
	for i := low; i <= high; i++ {
		digitVal := (data[i] / exp) % 10
		aux[low+count[digitVal]] = data[i]
		count[digitVal]++
	}
	
	// Copy back
	for i := low; i <= high; i++ {
		data[i] = aux[i]
	}
	
	// Recursively sort each bucket
	for i := 0; i < 10; i++ {
		start := low + count[i]
		end := low + count[i+1] - 1
		if start <= end {
			radixSortMSDRecursive(data, aux, start, end, digit-1)
		}
	}
}

// RadixSortOptimized implements optimized radix sort with base 256
func RadixSortOptimized(data []int) {
	if len(data) <= 1 {
		return
	}
	
	// Handle negative numbers
	minVal := data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
	}
	
	if minVal < 0 {
		for i := range data {
			data[i] -= minVal
		}
	}
	
	// Find maximum value
	maxVal := data[0]
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	
	// Use base 256 for better performance
	base := 256
	for exp := 1; maxVal/exp > 0; exp *= base {
		countingSortBase256(data, exp, base)
	}
	
	// Shift values back
	if minVal < 0 {
		for i := range data {
			data[i] += minVal
		}
	}
}

// countingSortBase256 performs counting sort with base 256
func countingSortBase256(data []int, exp, base int) {
	n := len(data)
	output := make([]int, n)
	count := make([]int, base)
	
	// Count occurrences
	for i := 0; i < n; i++ {
		digit := (data[i] / exp) % base
		count[digit]++
	}
	
	// Transform counts to indices
	for i := 1; i < base; i++ {
		count[i] += count[i-1]
	}
	
	// Build output array
	for i := n - 1; i >= 0; i-- {
		digit := (data[i] / exp) % base
		output[count[digit]-1] = data[i]
		count[digit]--
	}
	
	// Copy back
	copy(data, output)
}

// RadixSortString implements radix sort for strings
func RadixSortString(data []string) {
	if len(data) <= 1 {
		return
	}
	
	// Find maximum length
	maxLen := 0
	for _, s := range data {
		if len(s) > maxLen {
			maxLen = len(s)
		}
	}
	
	// Sort by each character position from right to left
	for pos := maxLen - 1; pos >= 0; pos-- {
		countingSortStringByChar(data, pos)
	}
}

// countingSortStringByChar performs counting sort based on character at specific position
func countingSortStringByChar(data []string, pos int) {
	n := len(data)
	output := make([]string, n)
	count := make([]int, 257) // 256 ASCII chars + 1 for strings shorter than pos
	
	// Count occurrences
	for i := 0; i < n; i++ {
		var charCode int
		if pos >= len(data[i]) {
			charCode = 0 // Treat missing characters as 0
		} else {
			charCode = int(data[i][pos]) + 1
		}
		count[charCode]++
	}
	
	// Transform counts to indices
	for i := 1; i < 257; i++ {
		count[i] += count[i-1]
	}
	
	// Build output array
	for i := n - 1; i >= 0; i-- {
		var charCode int
		if pos >= len(data[i]) {
			charCode = 0
		} else {
			charCode = int(data[i][pos]) + 1
		}
		output[count[charCode]-1] = data[i]
		count[charCode]--
	}
	
	// Copy back
	copy(data, output)
}

// RadixSortInt32 implements optimized radix sort for int32
func RadixSortInt32(data []int32) {
	if len(data) <= 1 {
		return
	}
	
	// Convert to uint32 to handle negative numbers
	udata := make([]uint32, len(data))
	for i, v := range data {
		udata[i] = uint32(v) ^ 0x80000000 // Flip sign bit
	}
	
	// Sort using 4 passes of 8 bits each
	for shift := uint(0); shift < 32; shift += 8 {
		countingSortUint32(udata, shift)
	}
	
	// Convert back
	for i, v := range udata {
		data[i] = int32(v ^ 0x80000000)
	}
}

// countingSortUint32 performs counting sort on uint32 with specific bit shift
func countingSortUint32(data []uint32, shift uint) {
	n := len(data)
	output := make([]uint32, n)
	count := make([]int, 256)
	
	// Count occurrences
	for i := 0; i < n; i++ {
		byte := (data[i] >> shift) & 0xFF
		count[byte]++
	}
	
	// Transform counts to indices
	for i := 1; i < 256; i++ {
		count[i] += count[i-1]
	}
	
	// Build output array
	for i := n - 1; i >= 0; i-- {
		byte := (data[i] >> shift) & 0xFF
		output[count[byte]-1] = data[i]
		count[byte]--
	}
	
	// Copy back
	copy(data, output)
}