package service

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidateData validates data against business rules
func (bv *BusinessValidator) ValidateData(data interface{}) bool {
	bv.resetCounters()

	// Apply all validation rules
	for _, rule := range bv.rules {
		if !rule.Validator(data) {
			bv.errorCount++
			// Log validation error (in real implementation)
			fmt.Printf("Validation failed: %s - %s\n", rule.Name, rule.ErrorMessage)
			
			// For critical errors, fail immediately
			if rule.Severity == "critical" {
				return false
			}
		} else {
			bv.successCount++
		}
	}

	// Calculate success rate
	totalRules := len(bv.rules)
	if totalRules == 0 {
		return true // No rules means validation passes
	}

	successRate := float64(bv.successCount) / float64(totalRules)
	
	// Require at least 80% of rules to pass
	return successRate >= 0.8
}

// initializeRules sets up default business validation rules
func (bv *BusinessValidator) initializeRules() {
	// Data completeness rule
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "data_completeness",
		Validator:   bv.validateDataCompleteness,
		ErrorMessage: "Data is incomplete or empty",
		Severity:    "critical",
	})

	// Data type consistency rule
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "type_consistency",
		Validator:   bv.validateTypeConsistency,
		ErrorMessage: "Data types are inconsistent",
		Severity:    "high",
	})

	// Numeric range validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "numeric_range",
		Validator:   bv.validateNumericRange,
		ErrorMessage: "Numeric values are outside acceptable range",
		Severity:    "medium",
	})

	// String format validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "string_format",
		Validator:   bv.validateStringFormat,
		ErrorMessage: "String format is invalid",
		Severity:    "medium",
	})

	// Data size validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "data_size",
		Validator:   bv.validateDataSize,
		ErrorMessage: "Data size exceeds limits",
		Severity:    "high",
	})

	// Business logic validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "business_logic",
		Validator:   bv.validateBusinessLogic,
		ErrorMessage: "Data violates business logic rules",
		Severity:    "high",
	})

	// Data quality validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "data_quality",
		Validator:   bv.validateDataQuality,
		ErrorMessage: "Data quality is below acceptable standards",
		Severity:    "medium",
	})

	// Security validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "security",
		Validator:   bv.validateSecurity,
		ErrorMessage: "Data contains potential security risks",
		Severity:    "critical",
	})

	// Performance validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "performance",
		Validator:   bv.validatePerformance,
		ErrorMessage: "Data processing may cause performance issues",
		Severity:    "low",
	})

	// Consistency validation
	bv.rules = append(bv.rules, ValidationRule{
		Name:        "consistency",
		Validator:   bv.validateConsistency,
		ErrorMessage: "Data contains inconsistencies",
		Severity:    "medium",
	})
}

// resetCounters resets validation counters
func (bv *BusinessValidator) resetCounters() {
	bv.errorCount = 0
	bv.successCount = 0
}

// Validation rule implementations

func (bv *BusinessValidator) validateDataCompleteness(data interface{}) bool {
	if data == nil {
		return false
	}

	switch d := data.(type) {
	case map[string]interface{}:
		// Check if map has any data
		if len(d) == 0 {
			return false
		}
		
		// Check for required fields
		requiredFields := []string{"data_type"}
		for _, field := range requiredFields {
			if _, exists := d[field]; !exists {
				return false
			}
		}
		
		// Check if data values are not empty
		for key, value := range d {
			if !bv.isValueComplete(key, value) {
				return false
			}
		}
		return true
	case []interface{}:
		return len(d) > 0
	case []int:
		return len(d) > 0
	case []float64:
		return len(d) > 0
	case []string:
		return len(d) > 0
	case string:
		return len(strings.TrimSpace(d)) > 0
	default:
		return true // Unknown types pass by default
	}
}

func (bv *BusinessValidator) isValueComplete(key string, value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return len(strings.TrimSpace(v)) > 0
	case []interface{}:
		return len(v) > 0
	case []int:
		return len(v) > 0
	case []float64:
		return len(v) > 0
	case []string:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func (bv *BusinessValidator) validateTypeConsistency(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true // Non-map data doesn't need type consistency validation
	}

	// Check array type consistency
	for _, value := range dataMap {
		switch v := value.(type) {
		case []interface{}:
			if !bv.isArrayTypeConsistent(v) {
				return false
			}
		}
	}

	// Check for expected data types based on field names
	for key, value := range dataMap {
		if !bv.isFieldTypeValid(key, value) {
			return false
		}
	}

	return true
}

func (bv *BusinessValidator) isArrayTypeConsistent(arr []interface{}) bool {
	if len(arr) == 0 {
		return true
	}

	// Get the type of the first element
	firstType := reflect.TypeOf(arr[0])
	
	// Check if all elements have the same type
	for _, item := range arr[1:] {
		if reflect.TypeOf(item) != firstType {
			return false
		}
	}

	return true
}

func (bv *BusinessValidator) isFieldTypeValid(fieldName string, value interface{}) bool {
	// Define expected types for certain field names
	expectedTypes := map[string][]string{
		"numeric_data":    {"[]int", "[]float64", "int", "float64"},
		"string_data":     {"[]string", "string"},
		"data_type":       {"string"},
		"length":          {"int"},
		"count":           {"int"},
		"transformed_at":  {"time.Time"},
	}

	if expectedTypeList, exists := expectedTypes[fieldName]; exists {
		actualType := fmt.Sprintf("%T", value)
		for _, expectedType := range expectedTypeList {
			if actualType == expectedType {
				return true
			}
		}
		return false
	}

	return true // Unknown fields pass validation
}

func (bv *BusinessValidator) validateNumericRange(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	for _, value := range dataMap {
		switch v := value.(type) {
		case int:
			if !bv.isIntInRange(v) {
				return false
			}
		case float64:
			if !bv.isFloatInRange(v) {
				return false
			}
		case []int:
			for _, num := range v {
				if !bv.isIntInRange(num) {
					return false
				}
			}
		case []float64:
			for _, num := range v {
				if !bv.isFloatInRange(num) {
					return false
				}
			}
		}
	}

	return true
}

func (bv *BusinessValidator) isIntInRange(value int) bool {
	// Define acceptable range for integers
	minInt := -1000000
	maxInt := 1000000
	return value >= minInt && value <= maxInt
}

func (bv *BusinessValidator) isFloatInRange(value float64) bool {
	// Check for NaN and Inf
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}

	// Define acceptable range for floats
	minFloat := -1e6
	maxFloat := 1e6
	return value >= minFloat && value <= maxFloat
}

func (bv *BusinessValidator) validateStringFormat(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	for key, value := range dataMap {
		switch v := value.(type) {
		case string:
			if !bv.isStringFormatValid(key, v) {
				return false
			}
		case []string:
			for _, str := range v {
				if !bv.isStringFormatValid(key, str) {
					return false
				}
			}
		}
	}

	return true
}

func (bv *BusinessValidator) isStringFormatValid(fieldName, value string) bool {
	// Basic string validation
	if len(value) > 10000 {
		return false // Too long
	}

	// Check for null bytes
	if strings.Contains(value, "\x00") {
		return false
	}

	// Field-specific validation
	switch fieldName {
	case "data_type":
		// Data type should be alphanumeric with underscores
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, value)
		return matched
	case "email":
		// Basic email validation
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, value)
		return matched
	case "url":
		// Basic URL validation
		matched, _ := regexp.MatchString(`^https?://[a-zA-Z0-9.-]+`, value)
		return matched
	default:
		// General string validation - no control characters except newlines and tabs
		for _, char := range value {
			if char < 32 && char != 9 && char != 10 && char != 13 {
				return false
			}
		}
		return true
	}
}

func (bv *BusinessValidator) validateDataSize(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	// Calculate total data size
	totalSize := bv.calculateDataSize(dataMap)

	// Define size limits (in bytes)
	maxSize := 100 * 1024 * 1024 // 100 MB

	return totalSize <= maxSize
}

func (bv *BusinessValidator) calculateDataSize(data map[string]interface{}) int {
	size := 0

	for key, value := range data {
		size += len(key) // Key size
		size += bv.calculateValueSize(value)
	}

	return size
}

func (bv *BusinessValidator) calculateValueSize(value interface{}) int {
	switch v := value.(type) {
	case string:
		return len(v)
	case []string:
		size := 0
		for _, str := range v {
			size += len(str)
		}
		return size
	case []int:
		return len(v) * 8 // 8 bytes per int64
	case []float64:
		return len(v) * 8 // 8 bytes per float64
	case []interface{}:
		size := 0
		for _, item := range v {
			size += bv.calculateValueSize(item)
		}
		return size
	case map[string]interface{}:
		size := 0
		for key, val := range v {
			size += len(key) + bv.calculateValueSize(val)
		}
		return size
	case int:
		return 8
	case float64:
		return 8
	case bool:
		return 1
	case time.Time:
		return 24 // Approximate size of time.Time
	default:
		return len(fmt.Sprintf("%v", value))
	}
}

func (bv *BusinessValidator) validateBusinessLogic(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	// Business rule: If numeric_data exists, it should have reasonable statistics
	if numericData, exists := dataMap["numeric_data"]; exists {
		switch nums := numericData.(type) {
		case []int:
			if !bv.validateNumericStatistics(nums) {
				return false
			}
		case []float64:
			if !bv.validateFloatStatistics(nums) {
				return false
			}
		}
	}

	// Business rule: Data type should match actual data
	if dataType, exists := dataMap["data_type"]; exists {
		if !bv.validateDataTypeConsistency(dataType, dataMap) {
			return false
		}
	}

	// Business rule: Length field should match actual data length
	if length, exists := dataMap["length"]; exists {
		if !bv.validateLengthConsistency(length, dataMap) {
			return false
		}
	}

	return true
}

func (bv *BusinessValidator) validateNumericStatistics(data []int) bool {
	if len(data) == 0 {
		return true
	}

	// Calculate basic statistics
	sum := 0
	min := data[0]
	max := data[0]

	for _, v := range data {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	mean := float64(sum) / float64(len(data))

	// Business rules for statistics
	// 1. Mean should not be extremely large
	if math.Abs(mean) > 1e6 {
		return false
	}

	// 2. Range should not be too extreme
	if max-min > 1e6 {
		return false
	}

	// 3. Data should not be all the same (unless it's a single value)
	if len(data) > 1 && min == max {
		return false
	}

	return true
}

func (bv *BusinessValidator) validateFloatStatistics(data []float64) bool {
	if len(data) == 0 {
		return true
	}

	// Check for NaN or Inf values
	for _, v := range data {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}

	// Calculate basic statistics
	sum := 0.0
	min := data[0]
	max := data[0]

	for _, v := range data {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	mean := sum / float64(len(data))

	// Business rules for statistics
	if math.Abs(mean) > 1e6 {
		return false
	}

	if max-min > 1e6 {
		return false
	}

	return true
}

func (bv *BusinessValidator) validateDataTypeConsistency(dataType interface{}, dataMap map[string]interface{}) bool {
	dtStr, ok := dataType.(string)
	if !ok {
		return false
	}

	// Check if declared data type matches actual data
	switch dtStr {
	case "integer_array":
		_, exists := dataMap["numeric_data"]
		return exists
	case "float_array":
		_, exists := dataMap["numeric_data"]
		return exists
	case "string_array":
		_, exists := dataMap["string_data"]
		return exists
	case "json_string":
		_, exists := dataMap["parsed_json"]
		return exists
	case "raw_string":
		_, exists := dataMap["raw_string"]
		return exists
	default:
		return true // Unknown types pass
	}
}

func (bv *BusinessValidator) validateLengthConsistency(length interface{}, dataMap map[string]interface{}) bool {
	lengthInt, ok := length.(int)
	if !ok {
		return false
	}

	// Check if length matches actual data length
	for key, value := range dataMap {
		if strings.Contains(key, "data") {
			actualLength := bv.getDataLength(value)
			if actualLength > 0 && actualLength != lengthInt {
				return false
			}
		}
	}

	return true
}

func (bv *BusinessValidator) getDataLength(value interface{}) int {
	switch v := value.(type) {
	case []int:
		return len(v)
	case []float64:
		return len(v)
	case []string:
		return len(v)
	case []interface{}:
		return len(v)
	case string:
		return len(v)
	default:
		return 0
	}
}

func (bv *BusinessValidator) validateDataQuality(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	qualityScore := 0.0
	totalChecks := 0.0

	// Check 1: Data completeness
	if bv.validateDataCompleteness(data) {
		qualityScore += 1.0
	}
	totalChecks += 1.0

	// Check 2: Type consistency
	if bv.validateTypeConsistency(data) {
		qualityScore += 1.0
	}
	totalChecks += 1.0

	// Check 3: Numeric data quality
	for _, value := range dataMap {
		switch v := value.(type) {
		case []int:
			if bv.assessNumericQuality(v) {
				qualityScore += 1.0
			}
			totalChecks += 1.0
		case []float64:
			if bv.assessFloatQuality(v) {
				qualityScore += 1.0
			}
			totalChecks += 1.0
		}
	}

	// Check 4: String data quality
	for _, value := range dataMap {
		switch v := value.(type) {
		case []string:
			if bv.assessStringQuality(v) {
				qualityScore += 1.0
			}
			totalChecks += 1.0
		case string:
			if bv.assessSingleStringQuality(v) {
				qualityScore += 1.0
			}
			totalChecks += 1.0
		}
	}

	if totalChecks == 0 {
		return true
	}

	// Require at least 70% quality score
	return (qualityScore / totalChecks) >= 0.7
}

func (bv *BusinessValidator) assessNumericQuality(data []int) bool {
	if len(data) == 0 {
		return false
	}

	// Check for reasonable distribution
	uniqueValues := make(map[int]bool)
	for _, v := range data {
		uniqueValues[v] = true
	}

	// Should have some variety (not all the same)
	if len(uniqueValues) == 1 && len(data) > 1 {
		return false
	}

	// Should not have too many outliers
	outlierCount := bv.countIntOutliers(data)
	outlierRatio := float64(outlierCount) / float64(len(data))

	return outlierRatio < 0.1 // Less than 10% outliers
}

func (bv *BusinessValidator) assessFloatQuality(data []float64) bool {
	if len(data) == 0 {
		return false
	}

	// Check for NaN or Inf
	for _, v := range data {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}

	// Check for reasonable precision (not too many decimal places)
	for _, v := range data {
		str := strconv.FormatFloat(v, 'f', -1, 64)
		if strings.Contains(str, ".") {
			parts := strings.Split(str, ".")
			if len(parts[1]) > 10 {
				return false // Too many decimal places
			}
		}
	}

	return true
}

func (bv *BusinessValidator) assessStringQuality(data []string) bool {
	if len(data) == 0 {
		return false
	}

	// Check for empty strings
	emptyCount := 0
	for _, s := range data {
		if len(strings.TrimSpace(s)) == 0 {
			emptyCount++
		}
	}

	// Should not have too many empty strings
	emptyRatio := float64(emptyCount) / float64(len(data))
	if emptyRatio > 0.2 {
		return false
	}

	// Check for reasonable string lengths
	for _, s := range data {
		if len(s) > 1000 {
			return false // Too long
		}
	}

	return true
}

func (bv *BusinessValidator) assessSingleStringQuality(data string) bool {
	// Check for reasonable length
	if len(data) == 0 || len(data) > 10000 {
		return false
	}

	// Check for control characters (except common ones)
	for _, char := range data {
		if char < 32 && char != 9 && char != 10 && char != 13 {
			return false
		}
	}

	return true
}

func (bv *BusinessValidator) countIntOutliers(data []int) int {
	if len(data) < 3 {
		return 0
	}

	// Calculate mean and standard deviation
	sum := 0
	for _, v := range data {
		sum += v
	}
	mean := float64(sum) / float64(len(data))

	varSum := 0.0
	for _, v := range data {
		diff := float64(v) - mean
		varSum += diff * diff
	}
	stdDev := math.Sqrt(varSum / float64(len(data)))

	// Count outliers (more than 2 standard deviations from mean)
	outlierCount := 0
	for _, v := range data {
		if math.Abs(float64(v)-mean) > 2*stdDev {
			outlierCount++
		}
	}

	return outlierCount
}

func (bv *BusinessValidator) validateSecurity(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	// Check for potential security issues
	for key, value := range dataMap {
		// Check for suspicious field names
		if bv.isSuspiciousFieldName(key) {
			return false
		}

		// Check string values for potential security risks
		switch v := value.(type) {
		case string:
			if bv.containsSecurityRisk(v) {
				return false
			}
		case []string:
			for _, str := range v {
				if bv.containsSecurityRisk(str) {
					return false
				}
			}
		}
	}

	return true
}

func (bv *BusinessValidator) isSuspiciousFieldName(fieldName string) bool {
	suspiciousNames := []string{
		"password", "passwd", "pwd", "secret", "key", "token",
		"api_key", "private_key", "auth", "credential",
	}

	lowerName := strings.ToLower(fieldName)
	for _, suspicious := range suspiciousNames {
		if strings.Contains(lowerName, suspicious) {
			return true
		}
	}

	return false
}

func (bv *BusinessValidator) containsSecurityRisk(value string) bool {
	// Check for potential SQL injection patterns
	sqlPatterns := []string{
		"DROP TABLE", "DELETE FROM", "INSERT INTO", "UPDATE SET",
		"UNION SELECT", "OR 1=1", "'; --", "/*", "*/",
	}

	upperValue := strings.ToUpper(value)
	for _, pattern := range sqlPatterns {
		if strings.Contains(upperValue, pattern) {
			return true
		}
	}

	// Check for potential XSS patterns
	xssPatterns := []string{
		"<script", "javascript:", "onload=", "onerror=",
		"<iframe", "<object", "<embed",
	}

	lowerValue := strings.ToLower(value)
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerValue, pattern) {
			return true
		}
	}

	// Check for potential path traversal
	if strings.Contains(value, "../") || strings.Contains(value, "..\\") {
		return true
	}

	return false
}

func (bv *BusinessValidator) validatePerformance(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	// Check for performance-impacting data characteristics
	for _, value := range dataMap {
		switch v := value.(type) {
		case []int:
			// Very large arrays may cause performance issues
			if len(v) > 1000000 {
				return false
			}
		case []float64:
			if len(v) > 1000000 {
				return false
			}
		case []string:
			if len(v) > 100000 {
				return false
			}
			// Check for very long strings
			for _, str := range v {
				if len(str) > 100000 {
					return false
				}
			}
		case string:
			if len(v) > 1000000 {
				return false
			}
		case map[string]interface{}:
			// Deeply nested structures may cause performance issues
			if bv.calculateNestingDepth(v) > 10 {
				return false
			}
		}
	}

	return true
}

func (bv *BusinessValidator) calculateNestingDepth(data map[string]interface{}) int {
	maxDepth := 0
	for _, value := range data {
		if nested, ok := value.(map[string]interface{}); ok {
			depth := 1 + bv.calculateNestingDepth(nested)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}
	return maxDepth
}

func (bv *BusinessValidator) validateConsistency(data interface{}) bool {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return true
	}

	// Check for internal consistency
	// Example: If both numeric_data and aggregated statistics exist, they should be consistent
	if numericData, hasNumeric := dataMap["numeric_data"]; hasNumeric {
		if aggregated, hasAgg := dataMap["aggregated"]; hasAgg {
			if !bv.validateAggregationConsistency(numericData, aggregated) {
				return false
			}
		}
	}

	// Check timestamp consistency
	if transformedAt, hasTime := dataMap["transformed_at"]; hasTime {
		if !bv.validateTimestampConsistency(transformedAt) {
			return false
		}
	}

	return true
}

func (bv *BusinessValidator) validateAggregationConsistency(numericData, aggregated interface{}) bool {
	// This would check if aggregated statistics match the raw numeric data
	// For now, just check if both exist and are reasonable
	switch numericData.(type) {
	case []int, []float64:
		// Numeric data exists
		if aggMap, ok := aggregated.(map[string]interface{}); ok {
			// Check if aggregation has expected fields
			requiredFields := []string{"count", "mean", "min", "max"}
			for _, field := range requiredFields {
				if _, exists := aggMap[field]; !exists {
					return false
				}
			}
			return true
		}
		return false
	default:
		return true
	}
}

func (bv *BusinessValidator) validateTimestampConsistency(timestamp interface{}) bool {
	switch ts := timestamp.(type) {
	case time.Time:
		// Check if timestamp is reasonable (not too far in the past or future)
		now := time.Now()
		if ts.After(now.Add(time.Hour)) {
			return false // Too far in the future
		}
		if ts.Before(now.Add(-24 * time.Hour)) {
			return false // Too far in the past
		}
		return true
	case string:
		// Try to parse as timestamp
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			return false
		}
		return true
	default:
		return false
	}
}

// GetStats returns validation statistics
func (bv *BusinessValidator) GetStats() (int64, int64) {
	return bv.errorCount, bv.successCount
}

// AddCustomRule adds a custom validation rule
func (bv *BusinessValidator) AddCustomRule(rule ValidationRule) {
	bv.rules = append(bv.rules, rule)
}

// RemoveRule removes a validation rule by name
func (bv *BusinessValidator) RemoveRule(ruleName string) {
	for i, rule := range bv.rules {
		if rule.Name == ruleName {
			bv.rules = append(bv.rules[:i], bv.rules[i+1:]...)
			break
		}
	}
}

// GetRules returns all validation rules
func (bv *BusinessValidator) GetRules() []ValidationRule {
	return bv.rules
}