package main

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
)

// CPU-intensive function with deliberate inefficiencies
// This function demonstrates common performance anti-patterns

// Inefficient string concatenation in loop
func buildLargeStringBad(size int) string {
	if size <= 0 {
		return ""
	}
	if size > 100000 {
		fmt.Printf("Warning: Large size %d may cause excessive memory usage\n", size)
	}
	
	var result string
	for i := 0; i < size; i++ {
		result += "item" + strconv.Itoa(i) + ","
	}
	return result
}

// Inefficient sorting - bubble sort O(n²)
func bubbleSortBad(arr []int) []int {
	if arr == nil {
		return nil
	}
	n := len(arr)
	if n > 10000 {
		fmt.Printf("Warning: Bubble sort with %d elements will be very slow\n", n)
	}
	
	data := make([]int, n)
	copy(data, arr)
	
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if data[j] > data[j+1] {
				data[j], data[j+1] = data[j+1], data[j]
			}
		}
	}
	return data
}

// Inefficient prime checking with poor algorithm
func isPrimeBad(n int) bool {
	if n < 2 {
		return false
	}
	// Check all numbers up to n-1 (very inefficient)
	for i := 2; i < n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// Find primes up to n using inefficient method
func findPrimesBad(limit int) []int {
	if limit < 2 {
		return []int{}
	}
	if limit > 10000 {
		fmt.Printf("Warning: Finding primes up to %d will be very slow\n", limit)
	}
	
	var primes []int
	for i := 2; i <= limit; i++ {
		if isPrimeBad(i) {
			primes = append(primes, i)
		}
	}
	return primes
}

// Inefficient matrix multiplication with poor cache locality
func matrixMultiplyBad(a, b [][]int) [][]int {
	if a == nil || b == nil || len(a) == 0 || len(b) == 0 || len(b[0]) == 0 {
		return nil
	}
	
	n := len(a)
	m := len(b[0])
	p := len(b)
	
	if len(a[0]) != p {
		fmt.Println("Error: Matrix dimensions incompatible for multiplication")
		return nil
	}
	
	result := make([][]int, n)
	for i := range result {
		result[i] = make([]int, m)
	}
	
	// Poor cache locality: accessing columns first
	for j := 0; j < m; j++ {
		for i := 0; i < n; i++ {
			for k := 0; k < p; k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return result
}

// Inefficient recursive Fibonacci (exponential time complexity)
func fibonacciBad(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacciBad(n-1) + fibonacciBad(n-2)
}

// Inefficient string processing with multiple allocations
func processTextBad(text string) string {
	words := strings.Split(text, " ")
	var result string
	
	for _, word := range words {
		// Multiple string operations causing allocations
		processed := strings.ToUpper(word)
		processed = strings.TrimSpace(processed)
		processed = strings.ReplaceAll(processed, "A", "@")
		processed = strings.ReplaceAll(processed, "E", "3")
		processed = strings.ReplaceAll(processed, "I", "1")
		processed = strings.ReplaceAll(processed, "O", "0")
		result += processed + " "
	}
	return strings.TrimSpace(result)
}

// Inefficient data structure operations
func findDuplicatesBad(items []string) []string {
	var duplicates []string
	
	// O(n²) algorithm
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i] == items[j] {
				// Check if already added (another O(n) operation)
				found := false
				for _, dup := range duplicates {
					if dup == items[i] {
						found = true
						break
					}
				}
				if !found {
					duplicates = append(duplicates, items[i])
				}
			}
		}
	}
	return duplicates
}

// Inefficient mathematical computation
func computeExpensiveMathBad(n int) float64 {
	var result float64
	for i := 1; i <= n; i++ {
		// Expensive operations in loop
		result += math.Pow(float64(i), 3) * math.Sin(float64(i)) * math.Cos(float64(i))
		result += math.Sqrt(float64(i)) * math.Log(float64(i))
	}
	return result
}

// Helper function to generate test data
func generateTestData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = size - i // Reverse order for worst-case sorting
	}
	return data
}

func generateMatrix(size int) [][]int {
	matrix := make([][]int, size)
	for i := range matrix {
		matrix[i] = make([]int, size)
		for j := range matrix[i] {
			matrix[i][j] = i*size + j + 1
		}
	}
	return matrix
}

func generateTestText(words int) string {
	var builder strings.Builder
	wordList := []string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape", "honeydew"}
	
	for i := 0; i < words; i++ {
		if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(wordList[i%len(wordList)])
	}
	return builder.String()
}

func main() {
	// Validate runtime environment
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys < 100*1024*1024 { // Less than 100MB
		fmt.Println("Warning: Low system memory may affect performance results")
	}
	
	fmt.Println("=== CPU Intensive Functions Demo ===")
	
	// Test string building
	fmt.Println("\n1. String Building:")
	result1 := buildLargeStringBad(1000)
	fmt.Printf("Built string length: %d\n", len(result1))
	
	// Test sorting
	fmt.Println("\n2. Sorting:")
	data := generateTestData(1000)
	sorted := bubbleSortBad(data)
	fmt.Printf("Sorted %d elements\n", len(sorted))
	
	// Test prime finding
	fmt.Println("\n3. Prime Finding:")
	primes := findPrimesBad(100)
	fmt.Printf("Found %d primes up to 100\n", len(primes))
	
	// Test matrix multiplication
	fmt.Println("\n4. Matrix Multiplication:")
	matrixA := generateMatrix(50)
	matrixB := generateMatrix(50)
	result4 := matrixMultiplyBad(matrixA, matrixB)
	fmt.Printf("Multiplied %dx%d matrices\n", len(result4), len(result4[0]))
	
	// Test Fibonacci
	fmt.Println("\n5. Fibonacci:")
	fib := fibonacciBad(30)
	fmt.Printf("Fibonacci(30) = %d\n", fib)
	
	// Test text processing
	fmt.Println("\n6. Text Processing:")
	text := generateTestText(100)
	processed := processTextBad(text)
	fmt.Printf("Processed text length: %d\n", len(processed))
	
	// Test duplicate finding
	fmt.Println("\n7. Duplicate Finding:")
	items := []string{"a", "b", "c", "a", "d", "b", "e", "c", "f"}
	duplicates := findDuplicatesBad(items)
	fmt.Printf("Found duplicates: %v\n", duplicates)
	
	// Test expensive math
	fmt.Println("\n8. Expensive Math:")
	mathResult := computeExpensiveMathBad(1000)
	fmt.Printf("Math result: %.2f\n", mathResult)
	
	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("Run benchmarks with: go test -bench=. -cpuprofile=cpu.prof")
	fmt.Println("Analyze with: go tool pprof cpu.prof")
}