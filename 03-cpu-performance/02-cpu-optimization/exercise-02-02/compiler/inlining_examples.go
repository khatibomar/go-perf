package compiler

import (
	"fmt"
)

// ShowInliningAnalysis displays information about function inlining
func ShowInliningAnalysis() {
	fmt.Println("Function Inlining Analysis:")
	fmt.Println("- Small functions (< 80 nodes) are candidates for inlining")
	fmt.Println("- Inlining eliminates function call overhead")
	fmt.Println("- Use -gcflags='-m' to see inlining decisions")
	fmt.Println("- Use -gcflags='-l=4' to increase inlining aggressiveness")
	fmt.Println("- Use -gcflags='-l=0' to disable inlining")
}

// Simple function that should be inlined
//go:noinline
func add(a, b int) int {
	return a + b
}

// Simple function that will be inlined (no directive)
func addInlined(a, b int) int {
	return a + b
}

// More complex function that might not be inlined
func complexAdd(a, b int) int {
	result := a + b
	if result > 100 {
		result *= 2
	}
	if result > 1000 {
		result /= 3
	}
	return result
}

// Very simple function that should always be inlined
func multiply(a, b int) int {
	return a * b
}

// Function with loop - less likely to be inlined
func sumArray(arr []int) int {
	sum := 0
	for _, v := range arr {
		sum += v
	}
	return sum
}

// Small helper function - good candidate for inlining
func square(x int) int {
	return x * x
}

// Function that uses inlined helpers
func ProcessWithInlining(data []int) int {
	result := 0
	for i, v := range data {
		// These should be inlined
		temp := addInlined(v, i)
		temp = multiply(temp, 2)
		result = addInlined(result, square(temp))
	}
	return result
}

// Function that uses non-inlined helpers
func ProcessWithoutInlining(data []int) int {
	result := 0
	for i, v := range data {
		// This won't be inlined due to directive
		temp := add(v, i)
		temp = complexAdd(temp, 2)
		result = add(result, temp*temp)
	}
	return result
}

// Recursive function - won't be inlined
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

// Tail-recursive function - might be optimized differently
func factorialTail(n, acc int) int {
	if n <= 1 {
		return acc
	}
	return factorialTail(n-1, n*acc)
}

// Interface method - harder to inline
type Calculator interface {
	Calculate(a, b int) int
}

type SimpleCalculator struct{}

func (c SimpleCalculator) Calculate(a, b int) int {
	return a + b
}

// Function using interface - prevents inlining
func ProcessWithInterface(data []int, calc Calculator) int {
	result := 0
	for i, v := range data {
		result = calc.Calculate(result, v+i)
	}
	return result
}

// Direct function call - can be inlined
func ProcessDirect(data []int) int {
	result := 0
	for i, v := range data {
		result = addInlined(result, v+i)
	}
	return result
}

// Function with defer - affects inlining
func ProcessWithDefer(data []int) int {
	defer func() {
		// Cleanup code
	}()
	
	result := 0
	for _, v := range data {
		result = addInlined(result, v)
	}
	return result
}

// Function without defer - more likely to be inlined
func ProcessWithoutDefer(data []int) int {
	result := 0
	for _, v := range data {
		result = addInlined(result, v)
	}
	return result
}

// Hot path function - should be optimized
func HotPathCalculation(x, y int) int {
	// Simple operations that should be inlined
	a := addInlined(x, y)
	b := multiply(a, 2)
	c := square(b)
	return addInlined(c, 1)
}

// Cold path function - less critical for inlining
func ColdPathCalculation(x, y int) int {
	// More complex operations
	result := complexAdd(x, y)
	if result > 0 {
		result = fibonacci(result % 10)
	}
	return result
}

// Benchmark helper functions
func InliningBenchmarkSetup(size int) []int {
	data := make([]int, size)
	for i := range data {
		data[i] = i % 1000
	}
	return data
}

func MeasureInliningImpact(data []int, iterations int) (inlinedTime, nonInlinedTime int64) {
	// This would typically use time.Now() but we'll simulate
	// In real benchmarks, use testing.B
	return int64(len(data) * iterations), int64(len(data) * iterations * 2)
}