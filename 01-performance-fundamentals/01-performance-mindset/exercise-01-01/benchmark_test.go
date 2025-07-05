package main

import (
	"runtime"
	"testing"
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

func BenchmarkFibonacci(b *testing.B) {
	validateBenchmarkEnvironment(b)
	for i := 0; i < b.N; i++ {
		fibonacci(35)
	}
}

func BenchmarkFibonacci20(b *testing.B) {
	validateBenchmarkEnvironment(b)
	for i := 0; i < b.N; i++ {
		fibonacci(20)
	}
}

func BenchmarkFibonacci25(b *testing.B) {
	validateBenchmarkEnvironment(b)
	for i := 0; i < b.N; i++ {
		fibonacci(25)
	}
}

func BenchmarkFibonacci30(b *testing.B) {
	validateBenchmarkEnvironment(b)
	for i := 0; i < b.N; i++ {
		fibonacci(30)
	}
}

// Benchmark with different approaches for comparison
func fibonacciIterative(n int) int {
	if n <= 1 {
		return n
	}
	
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

func BenchmarkFibonacciIterative35(b *testing.B) {
	validateBenchmarkEnvironment(b)
	for i := 0; i < b.N; i++ {
		fibonacciIterative(35)
	}
}

// Memoized version for comparison
var memo = make(map[int]int)

func fibonacciMemo(n int) int {
	if n <= 1 {
		return n
	}
	
	if val, exists := memo[n]; exists {
		return val
	}
	
	memo[n] = fibonacciMemo(n-1) + fibonacciMemo(n-2)
	return memo[n]
}

func BenchmarkFibonacciMemo35(b *testing.B) {
	validateBenchmarkEnvironment(b)
	b.ResetTimer() // Don't count setup time
	for i := 0; i < b.N; i++ {
		// Clear memo for fair comparison
		memo = make(map[int]int)
		fibonacciMemo(35)
	}
}

// Test correctness
func TestFibonacci(t *testing.T) {
	tests := []struct {
		input    int
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
		result := fibonacci(test.input)
		if result != test.expected {
			t.Errorf("fibonacci(%d) = %d; expected %d", test.input, result, test.expected)
		}
		
		// Test iterative version too
		resultIter := fibonacciIterative(test.input)
		if resultIter != test.expected {
			t.Errorf("fibonacciIterative(%d) = %d; expected %d", test.input, resultIter, test.expected)
		}
	}
}