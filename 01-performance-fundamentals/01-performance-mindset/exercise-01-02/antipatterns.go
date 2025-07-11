package main

import (
	"strings"
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
		// Expensive operation used unnecessarily
		processed := strings.ToLower(strings.TrimSpace(strings.ToUpper(item)))
		result = append(result, processed)
	}
	return result
}
