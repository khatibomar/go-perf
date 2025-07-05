package main

import (
	"fmt"
	"strings"
	"time"
)

// Anti-pattern 1: String concatenation in loop
func buildStringBad(items []string) string {
	var result string
	for _, item := range items {
		result += item + ","
	}
	return result
}

// Anti-pattern 2: Unnecessary allocations
func processItemsBad(items []string) []string {
	var result []string
	for _, item := range items {
		// Creating new slice every iteration
		temp := []string{strings.ToUpper(item)}
		result = append(result, temp[0])
	}
	return result
}

// Anti-pattern 3: Inefficient data structure usage
func findItemBad(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// Anti-pattern 4: Repeated expensive operations
func processDataBad(items []string) []string {
	var result []string
	for _, item := range items {
		// Expensive operation repeated unnecessarily
		processed := strings.ToLower(strings.TrimSpace(strings.ToUpper(item)))
		result = append(result, processed)
	}
	return result
}

func timeFunction(name string, fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start)
	fmt.Printf("%s took: %v\n", name, duration)
}

func main() {
	// Create test data
	items := make([]string, 10000)
	for i := range items {
		items[i] = fmt.Sprintf("  Item%d  ", i)
	}
	
	fmt.Println("=== Performance Anti-Patterns Demo ===")
	fmt.Printf("Testing with %d items\n\n", len(items))
	
	// Test anti-pattern 1: String concatenation
	fmt.Println("1. String Concatenation Anti-Pattern:")
	timeFunction("buildStringBad", func() {
		result := buildStringBad(items[:1000]) // Use smaller subset for demo
		_ = result
	})
	
	// Test anti-pattern 2: Unnecessary allocations
	fmt.Println("\n2. Unnecessary Allocations Anti-Pattern:")
	timeFunction("processItemsBad", func() {
		result := processItemsBad(items)
		_ = result
	})
	
	// Test anti-pattern 3: Inefficient lookups
	fmt.Println("\n3. Inefficient Lookup Anti-Pattern:")
	timeFunction("findItemBad (multiple lookups)", func() {
		for i := 0; i < 100; i++ {
			target := fmt.Sprintf("  Item%d  ", i*100)
			_ = findItemBad(items, target)
		}
	})
	
	// Test anti-pattern 4: Repeated expensive operations
	fmt.Println("\n4. Repeated Expensive Operations Anti-Pattern:")
	timeFunction("processDataBad", func() {
		result := processDataBad(items)
		_ = result
	})
	
	fmt.Println("\n=== Analysis ===")
	fmt.Println("Notice the performance differences. Run the benchmark to see exact numbers.")
	fmt.Println("Next, check improved.go for optimized versions.")
}