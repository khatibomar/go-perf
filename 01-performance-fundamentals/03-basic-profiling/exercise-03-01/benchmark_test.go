package main

import (
	"fmt"
	"runtime"
	"testing"
)

// Global variables to prevent compiler optimizations
var (
	globalString string
	globalIntSlice []int
	globalMatrix [][]int
	globalInt int
	globalFloat float64
)

// validateBenchmarkEnvironment ensures the benchmark environment is suitable
func validateBenchmarkEnvironment(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys < 100*1024*1024 { // Less than 100MB
		b.Skip("Insufficient memory for reliable benchmarking")
	}
}

// Benchmark for string building
func BenchmarkBuildLargeStringBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	var result string
	
	for i := 0; i < b.N; i++ {
		result = buildLargeStringBad(1000)
	}
	
	globalString = result
}

// Benchmark for bubble sort
func BenchmarkBubbleSortBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	data := generateTestData(1000)
	var result []int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = bubbleSortBad(data)
	}
	
	globalIntSlice = result
}

// Benchmark for prime finding
func BenchmarkFindPrimesBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	var result []int
	
	for i := 0; i < b.N; i++ {
		result = findPrimesBad(1000)
	}
	
	globalIntSlice = result
}

// Benchmark for matrix multiplication
func BenchmarkMatrixMultiplyBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	matrixA := generateMatrix(100)
	matrixB := generateMatrix(100)
	var result [][]int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = matrixMultiplyBad(matrixA, matrixB)
	}
	
	globalMatrix = result
}

// Benchmark for Fibonacci
func BenchmarkFibonacciBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	var result int
	
	for i := 0; i < b.N; i++ {
		result = fibonacciBad(35) // Reduced from 30 to make it more CPU intensive
	}
	
	globalInt = result
}

// Benchmark for text processing
func BenchmarkProcessTextBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	text := generateTestText(1000)
	var result string
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = processTextBad(text)
	}
	
	globalString = result
}

// Benchmark for duplicate finding
func BenchmarkFindDuplicatesBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	// Create a larger dataset with duplicates
	items := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = string(rune('a' + (i % 26))) // Creates duplicates
	}
	var result []string
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = findDuplicatesBad(items)
	}
	
	globalIntSlice = make([]int, len(result)) // Convert for global assignment
}

// Benchmark for expensive math computation
func BenchmarkComputeExpensiveMathBad(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	var result float64
	
	for i := 0; i < b.N; i++ {
		result = computeExpensiveMathBad(1000)
	}
	
	globalFloat = result
}

// Parametric benchmarks for different input sizes
func BenchmarkBubbleSortBad_100(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	data := generateTestData(100)
	var result []int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = bubbleSortBad(data)
	}
	
	globalIntSlice = result
}

func BenchmarkBubbleSortBad_500(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	data := generateTestData(500)
	var result []int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = bubbleSortBad(data)
	}
	
	globalIntSlice = result
}

func BenchmarkBubbleSortBad_1000(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	data := generateTestData(1000)
	var result []int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = bubbleSortBad(data)
	}
	
	globalIntSlice = result
}

// Sub-benchmarks for different Fibonacci numbers
func BenchmarkFibonacci(b *testing.B) {
	validateBenchmarkEnvironment(b)
	testCases := []struct {
		name string
		n    int
	}{
		{"Fib20", 20},
		{"Fib25", 25},
		{"Fib30", 30},
		{"Fib35", 35},
	}
	
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			var result int
			
			for i := 0; i < b.N; i++ {
				result = fibonacciBad(tc.n)
			}
			
			globalInt = result
		})
	}
}

// Sub-benchmarks for different matrix sizes
func BenchmarkMatrixMultiply(b *testing.B) {
	validateBenchmarkEnvironment(b)
	sizes := []int{10, 25, 50, 100}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			matrixA := generateMatrix(size)
			matrixB := generateMatrix(size)
			var result [][]int
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result = matrixMultiplyBad(matrixA, matrixB)
			}
			
			globalMatrix = result
		})
	}
}

// Benchmark to demonstrate CPU profiling setup
func BenchmarkCPUIntensiveWorkload(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ReportAllocs()
	
	// Setup test data
	data := generateTestData(500)
	text := generateTestText(100)
	matrixA := generateMatrix(50)
	matrixB := generateMatrix(50)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Mix of different CPU-intensive operations
		_ = bubbleSortBad(data)
		_ = processTextBad(text)
		_ = matrixMultiplyBad(matrixA, matrixB)
		_ = fibonacciBad(25)
		_ = computeExpensiveMathBad(100)
	}
}

// Example of how to run with profiling:
// go test -bench=BenchmarkCPUIntensiveWorkload -cpuprofile=cpu.prof
// go tool pprof cpu.prof