package main

import (
	"fmt"
	"strings"
	"testing"
)

// Benchmark functions for CPU profiling
// Run with: go test -bench=. -cpuprofile=bench.prof

func BenchmarkCalculatePrimes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculatePrimes(1000)
	}
}

func BenchmarkCalculatePrimesLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculatePrimes(10000)
	}
}

func BenchmarkCalculateFibonacci(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateFibonacci(30)
	}
}

func BenchmarkCalculateFibonacciOptimized(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateFibonacciOptimized(30)
	}
}

func BenchmarkProcessStrings(b *testing.B) {
	data := generateStringData(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processStrings(data)
	}
}

func BenchmarkCalculateHashes(b *testing.B) {
	data := generateStringData(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateHashes(data)
	}
}

func BenchmarkSortAndSearch(b *testing.B) {
	data := generateIntData(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sortAndSearch(data)
	}
}

func BenchmarkBubbleSort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := generateIntData(100) // Smaller dataset for bubble sort
		bubbleSort(data)
	}
}

func BenchmarkQuickSort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := generateIntData(1000)
		quickSort(data, 0, len(data)-1)
	}
}

func BenchmarkCalculatePi(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculatePi(10000)
	}
}

func BenchmarkCalculateSqrtNewton(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateSqrtNewton(2.0, 100)
	}
}

func BenchmarkMatrixMultiply(b *testing.B) {
	// Create test matrices
	size := 50
	matrixA := make([][]float64, size)
	matrixB := make([][]float64, size)
	for i := 0; i < size; i++ {
		matrixA[i] = make([]float64, size)
		matrixB[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			matrixA[i][j] = float64(i + j)
			matrixB[i][j] = float64(i * j)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matrixMultiply(matrixA, matrixB)
	}
}

// Comparative benchmarks to show optimization impact

func BenchmarkStringReverseRunes(b *testing.B) {
	str := "this is a test string for reversing operations"
	for i := 0; i < b.N; i++ {
		reverseString(str)
	}
}

func BenchmarkStringReverseBytes(b *testing.B) {
	str := "this is a test string for reversing operations"
	for i := 0; i < b.N; i++ {
		reverseStringBytes(str)
	}
}

// Alternative implementation for comparison
func reverseStringBytes(s string) string {
	bytes := []byte(s)
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return string(bytes)
}

// Sub-benchmarks for different data sizes
func BenchmarkPrimesVariousSizes(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000, 10000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				calculatePrimes(size)
			}
		})
	}
}

func BenchmarkFibonacciVariousSizes(b *testing.B) {
	sizes := []int{20, 25, 30, 35}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("fib-%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				calculateFibonacci(size)
			}
		})
	}
}

// Memory allocation benchmarks
func BenchmarkStringConcatenation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var result string
		for j := 0; j < 100; j++ {
			result += "test"
		}
		_ = result
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var builder strings.Builder
		for j := 0; j < 100; j++ {
			builder.WriteString("test")
		}
		_ = builder.String()
	}
}

// Example test functions
func TestCalculatePrimes(t *testing.T) {
	primes := calculatePrimes(10)
	expected := []int{2, 3, 5, 7}
	if len(primes) != len(expected) {
		t.Errorf("Expected %d primes, got %d", len(expected), len(primes))
	}
	for i, prime := range expected {
		if primes[i] != prime {
			t.Errorf("Expected prime %d, got %d", prime, primes[i])
		}
	}
}

func TestCalculateFibonacci(t *testing.T) {
	tests := []struct {
		n        int
		expected int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 2},
		{4, 3},
		{5, 5},
		{10, 55},
	}

	for _, test := range tests {
		result := calculateFibonacci(test.n)
		if result != test.expected {
			t.Errorf("fibonacci(%d) = %d, expected %d", test.n, result, test.expected)
		}
	}
}

func TestReverseString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "olleh"},
		{"world", "dlrow"},
		{"Go", "oG"},
		{"", ""},
		{"a", "a"},
	}

	for _, test := range tests {
		result := reverseString(test.input)
		if result != test.expected {
			t.Errorf("reverseString(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestBinarySearch(t *testing.T) {
	data := []int{1, 3, 5, 7, 9, 11, 13, 15}

	tests := []struct {
		target   int
		expected int
	}{
		{1, 0},
		{5, 2},
		{15, 7},
		{2, -1},  // Not found
		{16, -1}, // Not found
	}

	for _, test := range tests {
		result := binarySearch(data, test.target)
		if result != test.expected {
			t.Errorf("binarySearch(%v, %d) = %d, expected %d", data, test.target, result, test.expected)
		}
	}
}