package main

import (
	"strings"
)

// Improvement 1: Use strings.Builder for efficient string concatenation
func buildStringGood(items []string) string {
	var builder strings.Builder
	// Pre-allocate capacity if we know approximate size
	builder.Grow(len(items) * 10) // Estimate average item length

	for _, item := range items {
		builder.WriteString(item)
		builder.WriteString(",")
	}
	return builder.String()
}

// Improvement 2: Pre-allocate and avoid unnecessary allocations
func processItemsGood(items []string) []string {
	result := make([]string, 0, len(items)) // Pre-allocate with capacity
	for _, item := range items {
		// Direct operation without intermediate slice
		result = append(result, strings.ToUpper(item))
	}
	return result
}

// Improvement 3: Use map for O(1) lookups
func createLookupMap(items []string) map[string]bool {
	lookup := make(map[string]bool, len(items))
	for _, item := range items {
		lookup[item] = true
	}
	return lookup
}

func findItemGood(lookup map[string]bool, target string) bool {
	return lookup[target]
}

// Improvement 4: Optimize expensive operations
func processDataGood(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		// Combine operations efficiently
		trimmed := strings.TrimSpace(item)
		lower := strings.ToLower(trimmed)
		result = append(result, lower)
	}
	return result
}

// Even better: Process in-place when possible
func processDataBetter(items []string) []string {
	result := make([]string, len(items))
	for i, item := range items {
		// Single allocation, direct assignment
		result[i] = strings.ToLower(strings.TrimSpace(item))
	}
	return result
}

// Removed timeFunction and main to avoid conflict with antipatterns.go
// The main function is in antipatterns.go which demonstrates both
// antipatterns and improved implementations
