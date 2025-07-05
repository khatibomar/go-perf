package main

import (
	"fmt"
	"time"
)

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func main() {
	n := 35
	
	// Measure execution time
	start := time.Now()
	result := fibonacci(n)
	duration := time.Since(start)
	
	fmt.Printf("fibonacci(%d) = %d\n", n, result)
	fmt.Printf("Execution time: %v\n", duration)
	fmt.Printf("Nanoseconds: %d\n", duration.Nanoseconds())
	
	// Additional measurements
	fmt.Printf("\n=== Performance Analysis ===\n")
	fmt.Printf("Operations per second: %.2f\n", 1.0/duration.Seconds())
	
	// Test different input sizes
	sizes := []int{20, 25, 30, 35}
	fmt.Printf("\n=== Scaling Analysis ===\n")
	for _, size := range sizes {
		start := time.Now()
		result := fibonacci(size)
		duration := time.Since(start)
		fmt.Printf("fib(%d) = %d, time: %v\n", size, result, duration)
	}
}