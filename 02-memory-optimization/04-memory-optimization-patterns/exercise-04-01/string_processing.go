package main

import (
	"strings"
)

// StringProcessor provides zero-allocation string processing functions
type StringProcessor struct {
	buffer []byte
	parts []string
	indices []int
}

// NewStringProcessor creates a new string processor with pre-allocated buffers
func NewStringProcessor(bufferSize int) *StringProcessor {
	return &StringProcessor{
		buffer: make([]byte, bufferSize),
		parts: make([]string, 0, 100),
		indices: make([]int, 0, 100),
	}
}

// ConcatZeroAlloc concatenates strings using pre-allocated buffer
func (sp *StringProcessor) ConcatZeroAlloc(strs ...string) string {
	// Calculate total length
	totalLen := 0
	for _, s := range strs {
		totalLen += len(s)
	}
	
	// Ensure buffer is large enough
	if totalLen > len(sp.buffer) {
		sp.buffer = make([]byte, totalLen*2)
	}
	
	// Copy strings to buffer
	offset := 0
	for _, s := range strs {
		offset += copy(sp.buffer[offset:], s)
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:totalLen])
}

// JoinZeroAlloc joins strings with separator using pre-allocated buffer
func (sp *StringProcessor) JoinZeroAlloc(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	
	// Calculate total length
	totalLen := 0
	for i, s := range strs {
		totalLen += len(s)
		if i > 0 {
			totalLen += len(sep)
		}
	}
	
	// Ensure buffer is large enough
	if totalLen > len(sp.buffer) {
		sp.buffer = make([]byte, totalLen*2)
	}
	
	// Build result
	offset := 0
	for i, s := range strs {
		if i > 0 {
			offset += copy(sp.buffer[offset:], sep)
		}
		offset += copy(sp.buffer[offset:], s)
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:totalLen])
}

// SplitZeroAlloc splits string and reuses slice for parts
func (sp *StringProcessor) SplitZeroAlloc(s, sep string) []string {
	// Reset parts slice
	sp.parts = sp.parts[:0]
	
	if len(sep) == 0 {
		// Split into individual characters
		for i := 0; i < len(s); i++ {
			sp.parts = append(sp.parts, s[i:i+1])
		}
		return sp.parts
	}
	
	start := 0
	for {
		index := strings.Index(s[start:], sep)
		if index == -1 {
			// Add remaining part
			sp.parts = append(sp.parts, s[start:])
			break
		}
		
		// Add part before separator
		sp.parts = append(sp.parts, s[start:start+index])
		start += index + len(sep)
	}
	
	return sp.parts
}

// TrimZeroAlloc trims whitespace without allocation
func (sp *StringProcessor) TrimZeroAlloc(s string) string {
	start := 0
	end := len(s)
	
	// Trim leading whitespace
	for start < end && isWhitespace(s[start]) {
		start++
	}
	
	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}
	
	return s[start:end]
}

// isWhitespace checks if character is whitespace
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// ToUpperZeroAlloc converts string to uppercase using buffer
func (sp *StringProcessor) ToUpperZeroAlloc(s string) string {
	if len(s) > len(sp.buffer) {
		sp.buffer = make([]byte, len(s)*2)
	}
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c = c - 'a' + 'A'
		}
		sp.buffer[i] = c
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:len(s)])
}

// ToLowerZeroAlloc converts string to lowercase using buffer
func (sp *StringProcessor) ToLowerZeroAlloc(s string) string {
	if len(s) > len(sp.buffer) {
		sp.buffer = make([]byte, len(s)*2)
	}
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c - 'A' + 'a'
		}
		sp.buffer[i] = c
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:len(s)])
}

// ReplaceZeroAlloc replaces occurrences of old with new using buffer
func (sp *StringProcessor) ReplaceZeroAlloc(s, old, new string) string {
	if len(old) == 0 {
		return s
	}
	
	// Count occurrences to estimate buffer size
	count := 0
	start := 0
	for {
		index := strings.Index(s[start:], old)
		if index == -1 {
			break
		}
		count++
		start += index + len(old)
	}
	
	if count == 0 {
		return s
	}
	
	// Calculate required buffer size
	requiredSize := len(s) + count*(len(new)-len(old))
	if requiredSize > len(sp.buffer) {
		sp.buffer = make([]byte, requiredSize*2)
	}
	
	// Build result
	offset := 0
	start = 0
	for {
		index := strings.Index(s[start:], old)
		if index == -1 {
			// Copy remaining part
			offset += copy(sp.buffer[offset:], s[start:])
			break
		}
		
		// Copy part before old
		offset += copy(sp.buffer[offset:], s[start:start+index])
		// Copy new
		offset += copy(sp.buffer[offset:], new)
		start += index + len(old)
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:offset])
}

// ContainsZeroAlloc checks if string contains substring without allocation
func (sp *StringProcessor) ContainsZeroAlloc(s, substr string) bool {
	return strings.Index(s, substr) != -1
}

// IndexZeroAlloc finds index of substring without allocation
func (sp *StringProcessor) IndexZeroAlloc(s, substr string) int {
	return strings.Index(s, substr)
}

// CountZeroAlloc counts occurrences of substring without allocation
func (sp *StringProcessor) CountZeroAlloc(s, substr string) int {
	if len(substr) == 0 {
		return len(s) + 1
	}
	
	count := 0
	start := 0
	for {
		index := strings.Index(s[start:], substr)
		if index == -1 {
			break
		}
		count++
		start += index + len(substr)
	}
	return count
}

// HasPrefixZeroAlloc checks if string has prefix without allocation
func (sp *StringProcessor) HasPrefixZeroAlloc(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// HasSuffixZeroAlloc checks if string has suffix without allocation
func (sp *StringProcessor) HasSuffixZeroAlloc(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// FieldsZeroAlloc splits string by whitespace using pre-allocated slice
func (sp *StringProcessor) FieldsZeroAlloc(s string) []string {
	sp.parts = sp.parts[:0]
	
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || isWhitespace(s[i]) {
			if i > start {
				sp.parts = append(sp.parts, s[start:i])
			}
			// Skip whitespace
			for i < len(s) && isWhitespace(s[i]) {
				i++
			}
			start = i
			i-- // Compensate for loop increment
		}
	}
	
	return sp.parts
}

// ReverseZeroAlloc reverses string using buffer
func (sp *StringProcessor) ReverseZeroAlloc(s string) string {
	if len(s) > len(sp.buffer) {
		sp.buffer = make([]byte, len(s)*2)
	}
	
	for i := 0; i < len(s); i++ {
		sp.buffer[i] = s[len(s)-1-i]
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:len(s)])
}

// RepeatZeroAlloc repeats string n times using buffer
func (sp *StringProcessor) RepeatZeroAlloc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if n == 1 {
		return s
	}
	
	totalLen := len(s) * n
	if totalLen > len(sp.buffer) {
		sp.buffer = make([]byte, totalLen*2)
	}
	
	offset := 0
	for i := 0; i < n; i++ {
		offset += copy(sp.buffer[offset:], s)
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:totalLen])
}

// FormatZeroAlloc provides simple string formatting without allocations
func (sp *StringProcessor) FormatZeroAlloc(template string, args ...interface{}) string {
	// Simple template replacement for %s and %d
	if len(template) > len(sp.buffer) {
		sp.buffer = make([]byte, len(template)*4)
	}
	
	offset := 0
	argIndex := 0
	
	for i := 0; i < len(template); i++ {
		if template[i] == '%' && i+1 < len(template) && argIndex < len(args) {
			switch template[i+1] {
			case 's':
				if str, ok := args[argIndex].(string); ok {
					offset += copy(sp.buffer[offset:], str)
				}
				argIndex++
				i++ // Skip format specifier
			case 'd':
				if num, ok := args[argIndex].(int); ok {
					// Convert int to string manually
					numStr := sp.intToString(num)
					offset += copy(sp.buffer[offset:], numStr)
				}
				argIndex++
				i++ // Skip format specifier
			default:
				sp.buffer[offset] = template[i]
				offset++
			}
		} else {
			sp.buffer[offset] = template[i]
			offset++
		}
	}
	
	return BytesToStringZeroAlloc(sp.buffer[:offset])
}

// intToString converts int to string (helper for FormatZeroAlloc)
func (sp *StringProcessor) intToString(n int) string {
	if n == 0 {
		return "0"
	}
	
	negative := n < 0
	if negative {
		n = -n
	}
	
	// Use end of buffer for temporary conversion
	bufEnd := len(sp.buffer)
	digits := 0
	
	for n > 0 {
		bufEnd--
		sp.buffer[bufEnd] = byte('0' + n%10)
		n /= 10
		digits++
	}
	
	if negative {
		bufEnd--
		sp.buffer[bufEnd] = '-'
		digits++
	}
	
	return BytesToStringZeroAlloc(sp.buffer[bufEnd : bufEnd+digits])
}

// Example usage
func demonstrateStringProcessing() {
	processor := NewStringProcessor(1024)
	
	// String concatenation
	result := processor.ConcatZeroAlloc("Hello", ", ", "World", "!")
	SinkString(result)
	
	// String joining
	parts := []string{"apple", "banana", "cherry"}
	joined := processor.JoinZeroAlloc(parts, ", ")
	SinkString(joined)
	
	// String splitting
	text := "one,two,three,four"
	splitParts := processor.SplitZeroAlloc(text, ",")
	for _, part := range splitParts {
		SinkString(part)
	}
	
	// Case conversion
	upper := processor.ToUpperZeroAlloc("hello world")
	SinkString(upper)
	
	lower := processor.ToLowerZeroAlloc("HELLO WORLD")
	SinkString(lower)
	
	// String replacement
	replaced := processor.ReplaceZeroAlloc("hello world", "world", "Go")
	SinkString(replaced)
	
	// String formatting
	formatted := processor.FormatZeroAlloc("Hello %s, you have %d messages", "Alice", 5)
	SinkString(formatted)
}