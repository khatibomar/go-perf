package stages

import (
	"time"
)
// FilterFunc defines a filter predicate function
type FilterFunc func(interface{}) bool

// FilterStage filters data based on a predicate
type FilterStage struct {
	*BaseStage
	filterFunc FilterFunc
	filtered   int64
}

// NewFilterStage creates a new filter stage
func NewFilterStage(name string, filterFunc FilterFunc) *FilterStage {
	return &FilterStage{
		BaseStage:  NewBaseStage(name),
		filterFunc: filterFunc,
	}
}

// Process filters the input data based on the predicate
func (s *FilterStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	if s.filterFunc == nil || s.filterFunc(data) {
		return data, nil
	}
	
	s.filtered++
	return nil, nil // Return nil without error to indicate filtered out
}

// GetFilteredCount returns the number of filtered items
func (s *FilterStage) GetFilteredCount() int64 {
	return s.filtered
}

// NumericFilterStage filters numeric values
type NumericFilterStage struct {
	*BaseStage
	minValue float64
	maxValue float64
	filtered int64
}

// NewNumericFilterStage creates a numeric filter stage
func NewNumericFilterStage(name string, minValue, maxValue float64) *NumericFilterStage {
	return &NumericFilterStage{
		BaseStage: NewBaseStage(name),
		minValue:  minValue,
		maxValue:  maxValue,
	}
}

// Process filters numeric values within the specified range
func (s *NumericFilterStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	var value float64
	var ok bool
	
	switch v := data.(type) {
	case int:
		value = float64(v)
		ok = true
	case int64:
		value = float64(v)
		ok = true
	case float64:
		value = v
		ok = true
	case float32:
		value = float64(v)
		ok = true
	default:
		return data, nil // Pass through non-numeric data
	}
	
	if ok && value >= s.minValue && value <= s.maxValue {
		return data, nil
	}
	
	s.filtered++
	return nil, nil // Return nil without error to indicate filtered out
}

// GetFilteredCount returns the number of filtered items
func (s *NumericFilterStage) GetFilteredCount() int64 {
	return s.filtered
}

// StringFilterStage filters strings based on patterns
type StringFilterStage struct {
	*BaseStage
	pattern  string
	contains bool
	filtered int64
}

// NewStringFilterStage creates a string filter stage
func NewStringFilterStage(name, pattern string, contains bool) *StringFilterStage {
	return &StringFilterStage{
		BaseStage: NewBaseStage(name),
		pattern:   pattern,
		contains:  contains,
	}
}

// Process filters strings based on pattern matching
func (s *StringFilterStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	if str, ok := data.(string); ok {
		containsPattern := len(s.pattern) == 0 || 
			(len(s.pattern) > 0 && len(str) >= len(s.pattern) && 
				findSubstring(str, s.pattern))
		
		if s.contains == containsPattern {
			return data, nil
		}
		
		s.filtered++
		return nil, nil // Return nil without error to indicate filtered out
	}
	
	return data, nil // Pass through non-string data
}

// GetFilteredCount returns the number of filtered items
func (s *StringFilterStage) GetFilteredCount() int64 {
	return s.filtered
}

// Simple substring search
func findSubstring(str, pattern string) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(str) < len(pattern) {
		return false
	}
	
	for i := 0; i <= len(str)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if str[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}