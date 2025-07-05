package main

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// Optimized versions of the CPU-intensive functions
// These demonstrate proper performance optimization techniques

// Optimized string concatenation using strings.Builder
func buildLargeStringGood(size int) string {
	var builder strings.Builder
	// Pre-allocate capacity to avoid reallocations
	builder.Grow(size * 10) // Estimate: "item" + number + "," ≈ 10 chars average
	
	for i := 0; i < size; i++ {
		builder.WriteString("item")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString(",")
	}
	return builder.String()
}

// Optimized sorting using Go's built-in sort (Introsort - O(n log n))
func sortGood(arr []int) []int {
	data := make([]int, len(arr))
	copy(data, arr)
	sort.Ints(data)
	return data
}

// Optimized prime checking with square root optimization
func isPrimeGood(n int) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	
	// Only check odd numbers up to sqrt(n)
	sqrt := int(math.Sqrt(float64(n)))
	for i := 3; i <= sqrt; i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// Optimized prime finding using Sieve of Eratosthenes
func findPrimesGood(limit int) []int {
	if limit < 2 {
		return []int{}
	}
	
	// Sieve of Eratosthenes - O(n log log n)
	sieve := make([]bool, limit+1)
	for i := 2; i <= limit; i++ {
		sieve[i] = true
	}
	
	for i := 2; i*i <= limit; i++ {
		if sieve[i] {
			for j := i * i; j <= limit; j += i {
				sieve[j] = false
			}
		}
	}
	
	// Collect primes
	var primes []int
	for i := 2; i <= limit; i++ {
		if sieve[i] {
			primes = append(primes, i)
		}
	}
	return primes
}

// Optimized matrix multiplication with better cache locality
func matrixMultiplyGood(a, b [][]int) [][]int {
	n := len(a)
	m := len(b[0])
	p := len(b)
	
	result := make([][]int, n)
	for i := range result {
		result[i] = make([]int, m)
	}
	
	// Better cache locality: accessing rows first
	for i := 0; i < n; i++ {
		for k := 0; k < p; k++ {
			for j := 0; j < m; j++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return result
}

// Removed main function to avoid conflict with cpu_intensive.go main function
// The main function is in cpu_intensive.go which demonstrates both
// CPU-intensive and optimized operations

// Optimized Fibonacci using dynamic programming (linear time)
func fibonacciGood(n int) int {
	if n <= 1 {
		return n
	}
	
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// Alternative: Fibonacci with memoization
var fibMemo = make(map[int]int)

func fibonacciMemo(n int) int {
	if n <= 1 {
		return n
	}
	
	if val, exists := fibMemo[n]; exists {
		return val
	}
	
	fibMemo[n] = fibonacciMemo(n-1) + fibonacciMemo(n-2)
	return fibMemo[n]
}

// Optimized string processing with single pass and strings.Builder
func processTextGood(text string) string {
	words := strings.Split(text, " ")
	var builder strings.Builder
	builder.Grow(len(text) * 2) // Estimate output size
	
	// Create replacer for efficient multiple replacements
	replacer := strings.NewReplacer(
		"A", "@",
		"E", "3",
		"I", "1",
		"O", "0",
	)
	
	for i, word := range words {
		if i > 0 {
			builder.WriteString(" ")
		}
		
		// Process word efficiently
		processed := strings.ToUpper(strings.TrimSpace(word))
		processed = replacer.Replace(processed)
		builder.WriteString(processed)
	}
	return builder.String()
}

// Optimized duplicate finding using map (O(n) time complexity)
func findDuplicatesGood(items []string) []string {
	seen := make(map[string]bool)
	duplicateSet := make(map[string]bool)
	
	// Single pass to identify duplicates
	for _, item := range items {
		if seen[item] {
			duplicateSet[item] = true
		} else {
			seen[item] = true
		}
	}
	
	// Convert set to slice
	duplicates := make([]string, 0, len(duplicateSet))
	for item := range duplicateSet {
		duplicates = append(duplicates, item)
	}
	return duplicates
}

// Optimized mathematical computation with reduced function calls
func computeExpensiveMathGood(n int) float64 {
	var result float64
	for i := 1; i <= n; i++ {
		f := float64(i)
		
		// Pre-compute values to reduce function calls
		cube := f * f * f
		sinVal := math.Sin(f)
		cosVal := math.Cos(f)
		sqrtVal := math.Sqrt(f)
		logVal := math.Log(f)
		
		result += cube * sinVal * cosVal
		result += sqrtVal * logVal
	}
	return result
}

// Alternative: Optimized math with vectorization potential
func computeExpensiveMathVectorized(n int) float64 {
	// Pre-allocate slices for potential vectorization
	values := make([]float64, n)
	for i := 0; i < n; i++ {
		values[i] = float64(i + 1)
	}
	
	var result float64
	for _, f := range values {
		// Batch similar operations
		cube := f * f * f
		trigProduct := math.Sin(f) * math.Cos(f)
		rootLog := math.Sqrt(f) * math.Log(f)
		
		result += cube*trigProduct + rootLog
	}
	return result
}

// Benchmark comparison functions
func runOptimizedComparison() {
	// This function can be used to compare optimized vs unoptimized versions
	// in a controlled manner
}

// Additional optimization: Memory pool for matrix operations
type MatrixPool struct {
	pool chan [][]int
	size int
}

func NewMatrixPool(poolSize, matrixSize int) *MatrixPool {
	p := &MatrixPool{
		pool: make(chan [][]int, poolSize),
		size: matrixSize,
	}
	
	// Pre-allocate matrices
	for i := 0; i < poolSize; i++ {
		matrix := make([][]int, matrixSize)
		for j := range matrix {
			matrix[j] = make([]int, matrixSize)
		}
		p.pool <- matrix
	}
	return p
}

func (p *MatrixPool) Get() [][]int {
	select {
	case matrix := <-p.pool:
		return matrix
	default:
		// Pool empty, create new matrix
		matrix := make([][]int, p.size)
		for i := range matrix {
			matrix[i] = make([]int, p.size)
		}
		return matrix
	}
}

func (p *MatrixPool) Put(matrix [][]int) {
	// Clear matrix
	for i := range matrix {
		for j := range matrix[i] {
			matrix[i][j] = 0
		}
	}
	
	select {
	case p.pool <- matrix:
	default:
		// Pool full, let GC handle it
	}
}

// Optimized matrix multiplication using pool
func matrixMultiplyPooled(a, b [][]int, pool *MatrixPool) [][]int {
	n := len(a)
	m := len(b[0])
	p := len(b)
	
	result := pool.Get()
	
	// Ensure result matrix is the right size
	if len(result) != n || len(result[0]) != m {
		result = make([][]int, n)
		for i := range result {
			result[i] = make([]int, m)
		}
	}
	
	// Better cache locality: accessing rows first
	for i := 0; i < n; i++ {
		for k := 0; k < p; k++ {
			for j := 0; j < m; j++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return result
}