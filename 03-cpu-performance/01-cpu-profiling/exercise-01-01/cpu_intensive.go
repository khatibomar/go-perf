package main

import (
	"crypto/md5"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Mathematical operations

// calculatePrimes implements the Sieve of Eratosthenes algorithm
// This is intentionally CPU-intensive for profiling purposes
func calculatePrimes(limit int) []int {
	if limit < 2 {
		return []int{}
	}

	// Create a boolean array "prime[0..limit]" and initialize
	// all entries as true
	prime := make([]bool, limit+1)
	for i := range prime {
		prime[i] = true
	}

	// Sieve of Eratosthenes
	for p := 2; p*p <= limit; p++ {
		// If prime[p] is not changed, then it is a prime
		if prime[p] {
			// Update all multiples of p
			for i := p * p; i <= limit; i += p {
				prime[i] = false
			}
		}
	}

	// Collect all prime numbers
	var primes []int
	for i := 2; i <= limit; i++ {
		if prime[i] {
			primes = append(primes, i)
		}
	}

	return primes
}

// calculateFibonacci implements recursive Fibonacci (intentionally inefficient)
// This creates a deep call stack and is CPU-intensive
func calculateFibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return calculateFibonacci(n-1) + calculateFibonacci(n-2)
}

// calculateFibonacciOptimized implements iterative Fibonacci for comparison
func calculateFibonacciOptimized(n int) int {
	if n <= 1 {
		return n
	}

	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// String operations

// processStrings performs various string operations
// This tests string allocation and manipulation performance
func processStrings(data []string) []string {
	result := make([]string, 0, len(data)*2)

	for _, str := range data {
		// Convert to uppercase
		upper := strings.ToUpper(str)
		result = append(result, upper)

		// Reverse the string
		reversed := reverseString(str)
		result = append(result, reversed)

		// Perform string concatenation
		concatenated := str + "-processed-" + upper + "-" + reversed
		result = append(result, concatenated)

		// Split and rejoin
		parts := strings.Split(str, "-")
		rejoined := strings.Join(parts, "_")
		result = append(result, rejoined)

		// Perform substring operations
		if len(str) > 10 {
			substr := str[5 : len(str)-5]
			result = append(result, substr)
		}
	}

	return result
}

// reverseString reverses a string character by character
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// stringContainsPattern performs pattern matching
func stringContainsPattern(data []string, pattern string) []string {
	var matches []string
	for _, str := range data {
		if strings.Contains(str, pattern) {
			matches = append(matches, str)
		}
	}
	return matches
}

// Hash operations

// calculateHashes computes MD5 hashes for strings
// This is CPU-intensive due to cryptographic operations
func calculateHashes(data []string) []string {
	hashes := make([]string, len(data))

	for i, str := range data {
		// Calculate MD5 hash
		hash := md5.Sum([]byte(str))
		hashes[i] = fmt.Sprintf("%x", hash)

		// Perform multiple hash iterations to increase CPU usage
		for j := 0; j < 100; j++ {
			hash = md5.Sum([]byte(fmt.Sprintf("%x-%d", hash, j)))
		}
		hashes[i] = fmt.Sprintf("%x", hash)
	}

	return hashes
}

// Data structure operations

// sortAndSearch performs sorting and binary search operations
func sortAndSearch(data []int) int {
	// Make a copy to avoid modifying original
	dataCopy := make([]int, len(data))
	copy(dataCopy, data)

	// Perform multiple sorting operations
	for i := 0; i < 5; i++ {
		// Shuffle the data
		shuffleInts(dataCopy)

		// Sort using different methods
		switch i % 3 {
		case 0:
			sort.Ints(dataCopy)
		case 1:
			bubbleSort(dataCopy)
		case 2:
			quickSort(dataCopy, 0, len(dataCopy)-1)
		}
	}

	// Perform binary searches
	sort.Ints(dataCopy)
	found := 0
	for i := 0; i < len(dataCopy); i += 10 {
		target := dataCopy[i]
		if binarySearch(dataCopy, target) != -1 {
			found++
		}
	}

	return found
}

// shuffleInts shuffles an integer slice
func shuffleInts(data []int) {
	for i := len(data) - 1; i > 0; i-- {
		j := i % (i + 1) // Simple pseudo-random
		data[i], data[j] = data[j], data[i]
	}
}

// bubbleSort implements bubble sort (intentionally inefficient)
func bubbleSort(data []int) {
	n := len(data)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if data[j] > data[j+1] {
				data[j], data[j+1] = data[j+1], data[j]
			}
		}
	}
}

// quickSort implements quicksort algorithm
func quickSort(data []int, low, high int) {
	if low < high {
		pi := partition(data, low, high)
		quickSort(data, low, pi-1)
		quickSort(data, pi+1, high)
	}
}

// partition is a helper function for quicksort
func partition(data []int, low, high int) int {
	pivot := data[high]
	i := low - 1

	for j := low; j < high; j++ {
		if data[j] < pivot {
			i++
			data[i], data[j] = data[j], data[i]
		}
	}
	data[i+1], data[high] = data[high], data[i+1]
	return i + 1
}

// binarySearch implements binary search
func binarySearch(data []int, target int) int {
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

	return -1 // Not found
}

// Mathematical computation functions

// calculatePi estimates Pi using the Monte Carlo method
func calculatePi(iterations int) float64 {
	insideCircle := 0

	for i := 0; i < iterations; i++ {
		// Generate random point (using simple pseudo-random)
		x := float64(i%1000) / 1000.0
		y := float64((i*7)%1000) / 1000.0

		// Check if point is inside unit circle
		if x*x+y*y <= 1.0 {
			insideCircle++
		}
	}

	return 4.0 * float64(insideCircle) / float64(iterations)
}

// calculateSqrtNewton calculates square root using Newton's method
func calculateSqrtNewton(x float64, iterations int) float64 {
	if x < 0 {
		return math.NaN()
	}
	if x == 0 {
		return 0
	}

	// Initial guess
	guess := x / 2

	// Newton's method iterations
	for i := 0; i < iterations; i++ {
		guess = (guess + x/guess) / 2
	}

	return guess
}

// matrixMultiply performs matrix multiplication
func matrixMultiply(a, b [][]float64) [][]float64 {
	rows := len(a)
	cols := len(b[0])
	common := len(b)

	result := make([][]float64, rows)
	for i := range result {
		result[i] = make([]float64, cols)
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			for k := 0; k < common; k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}

	return result
}