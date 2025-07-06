package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TransformData applies transformation rules to the data
func (dt *DataTransformer) TransformData(data interface{}) interface{} {
	// Check cache first
	cacheKey := dt.generateCacheKey(data)
	if cached := dt.getCachedTransformation(cacheKey); cached != nil {
		return cached
	}

	// Apply transformation pipeline
	transformed := dt.applyTransformationPipeline(data)

	// Cache result
	dt.cacheTransformation(cacheKey, transformed)

	return transformed
}

// initializeRules sets up default transformation rules
func (dt *DataTransformer) initializeRules() {
	// Normalization rule
	dt.transformationRules["normalize"] = TransformRule{
		Name:        "normalize",
		Function:    dt.normalizeData,
		Validation:  dt.validateNormalizedData,
		Description: "Normalizes numeric data to 0-1 range",
	}

	// Standardization rule
	dt.transformationRules["standardize"] = TransformRule{
		Name:        "standardize",
		Function:    dt.standardizeData,
		Validation:  dt.validateStandardizedData,
		Description: "Standardizes data to zero mean and unit variance",
	}

	// Logarithmic transformation
	dt.transformationRules["log_transform"] = TransformRule{
		Name:        "log_transform",
		Function:    dt.logTransform,
		Validation:  dt.validateLogTransform,
		Description: "Applies logarithmic transformation",
	}

	// Outlier removal
	dt.transformationRules["remove_outliers"] = TransformRule{
		Name:        "remove_outliers",
		Function:    dt.removeOutliers,
		Validation:  dt.validateOutlierRemoval,
		Description: "Removes statistical outliers from data",
	}

	// Smoothing transformation
	dt.transformationRules["smooth"] = TransformRule{
		Name:        "smooth",
		Function:    dt.smoothData,
		Validation:  dt.validateSmoothedData,
		Description: "Applies moving average smoothing",
	}

	// Aggregation transformation
	dt.transformationRules["aggregate"] = TransformRule{
		Name:        "aggregate",
		Function:    dt.aggregateData,
		Validation:  dt.validateAggregatedData,
		Description: "Aggregates data using various statistical measures",
	}

	// Encoding transformation
	dt.transformationRules["encode"] = TransformRule{
		Name:        "encode",
		Function:    dt.encodeData,
		Validation:  dt.validateEncodedData,
		Description: "Encodes categorical data to numeric format",
	}

	// Binning transformation
	dt.transformationRules["bin"] = TransformRule{
		Name:        "bin",
		Function:    dt.binData,
		Validation:  dt.validateBinnedData,
		Description: "Groups continuous data into discrete bins",
	}
}

// applyTransformationPipeline applies multiple transformations in sequence
func (dt *DataTransformer) applyTransformationPipeline(data interface{}) interface{} {
	// Convert input to a standardized format
	standardized := dt.standardizeInput(data)

	// Apply transformations based on data characteristics
	transformed := standardized

	// Apply normalization if data contains numeric values
	if dt.containsNumericData(transformed) {
		if result, ok := dt.transformationRules["normalize"].Function(transformed).(map[string]interface{}); ok {
			transformed = result
		}
	}

	// Apply outlier removal for large datasets
	if dt.isLargeDataset(transformed) {
		if result, ok := dt.transformationRules["remove_outliers"].Function(transformed).(map[string]interface{}); ok {
			transformed = result
		}
	}

	// Apply smoothing for noisy data
	if dt.isNoisyData(transformed) {
		if result, ok := dt.transformationRules["smooth"].Function(transformed).(map[string]interface{}); ok {
			transformed = result
		}
	}

	// Apply aggregation for very large datasets
	if dt.isVeryLargeDataset(transformed) {
		if result, ok := dt.transformationRules["aggregate"].Function(transformed).(map[string]interface{}); ok {
			transformed = result
		}
	}

	// Apply encoding for mixed data types
	if dt.containsMixedTypes(transformed) {
		if result, ok := dt.transformationRules["encode"].Function(transformed).(map[string]interface{}); ok {
			transformed = result
		}
	}

	return transformed
}

// standardizeInput converts various input types to a common format
func (dt *DataTransformer) standardizeInput(data interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	switch d := data.(type) {
	case map[string]interface{}:
		// Already in desired format
		for key, value := range d {
			result[key] = dt.processValue(value)
		}
	case []int:
		// Convert int slice to map
		result["numeric_data"] = d
		result["data_type"] = "integer_array"
		result["length"] = len(d)
	case []float64:
		// Convert float slice to map
		result["numeric_data"] = d
		result["data_type"] = "float_array"
		result["length"] = len(d)
	case []string:
		// Convert string slice to map
		result["string_data"] = d
		result["data_type"] = "string_array"
		result["length"] = len(d)
	case string:
		// Try to parse as JSON, otherwise treat as raw string
		var jsonData interface{}
		if err := json.Unmarshal([]byte(d), &jsonData); err == nil {
			result["parsed_json"] = jsonData
			result["data_type"] = "json_string"
		} else {
			result["raw_string"] = d
			result["data_type"] = "raw_string"
			result["string_analysis"] = dt.analyzeString(d)
		}
	default:
		// Generic handling
		result["raw_data"] = data
		result["data_type"] = fmt.Sprintf("%T", data)
	}

	// Add metadata
	result["transformed_at"] = time.Now()
	result["transformer_version"] = "1.0"

	return result
}

// processValue recursively processes individual values
func (dt *DataTransformer) processValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []interface{}:
		// Process array elements
		processed := make([]interface{}, len(v))
		for i, item := range v {
			processed[i] = dt.processValue(item)
		}
		return processed
	case map[string]interface{}:
		// Process nested map
		processed := make(map[string]interface{})
		for key, val := range v {
			processed[key] = dt.processValue(val)
		}
		return processed
	case string:
		// Try to convert string to number if possible
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num
		}
		return v
	default:
		return value
	}
}

// analyzeString performs basic string analysis
func (dt *DataTransformer) analyzeString(s string) map[string]interface{} {
	analysis := make(map[string]interface{})

	analysis["length"] = len(s)
	analysis["word_count"] = len(strings.Fields(s))
	analysis["char_frequency"] = dt.calculateCharFrequency(s)
	analysis["has_numbers"] = dt.containsNumbers(s)
	analysis["has_special_chars"] = dt.containsSpecialChars(s)
	analysis["entropy"] = dt.calculateStringEntropy(s)

	return analysis
}

// calculateCharFrequency calculates character frequency in a string
func (dt *DataTransformer) calculateCharFrequency(s string) map[string]int {
	freq := make(map[string]int)
	for _, char := range s {
		freq[string(char)]++
	}
	return freq
}

// containsNumbers checks if string contains numeric characters
func (dt *DataTransformer) containsNumbers(s string) bool {
	for _, char := range s {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}

// containsSpecialChars checks if string contains special characters
func (dt *DataTransformer) containsSpecialChars(s string) bool {
	for _, char := range s {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == ' ') {
			return true
		}
	}
	return false
}

// calculateStringEntropy calculates entropy of a string
func (dt *DataTransformer) calculateStringEntropy(s string) float64 {
	if len(s) == 0 {
		return 0.0
	}

	freq := dt.calculateCharFrequency(s)
	entropy := 0.0
	length := float64(len(s))

	for _, count := range freq {
		if count > 0 {
			p := float64(count) / length
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// Data characteristic detection methods

func (dt *DataTransformer) containsNumericData(data map[string]interface{}) bool {
	for _, value := range data {
		switch value.(type) {
		case int, float64, []int, []float64:
			return true
		case []interface{}:
			// Check array elements
			if arr, ok := value.([]interface{}); ok {
				for _, item := range arr {
					switch item.(type) {
					case int, float64:
						return true
					}
				}
			}
		}
	}
	return false
}

func (dt *DataTransformer) isLargeDataset(data map[string]interface{}) bool {
	for _, value := range data {
		switch v := value.(type) {
		case []int:
			return len(v) > 1000
		case []float64:
			return len(v) > 1000
		case []interface{}:
			return len(v) > 1000
		}
	}
	return false
}

func (dt *DataTransformer) isVeryLargeDataset(data map[string]interface{}) bool {
	for _, value := range data {
		switch v := value.(type) {
		case []int:
			return len(v) > 10000
		case []float64:
			return len(v) > 10000
		case []interface{}:
			return len(v) > 10000
		}
	}
	return false
}

func (dt *DataTransformer) isNoisyData(data map[string]interface{}) bool {
	// Simple heuristic: check variance in numeric data
	for _, value := range data {
		switch v := value.(type) {
		case []int:
			return dt.calculateIntVariance(v) > dt.calculateIntMean(v)
		case []float64:
			return dt.calculateFloatVariance(v) > dt.calculateFloatMean(v)
		}
	}
	return false
}

func (dt *DataTransformer) containsMixedTypes(data map[string]interface{}) bool {
	typeCount := make(map[string]bool)
	for _, value := range data {
		typeCount[fmt.Sprintf("%T", value)] = true
	}
	return len(typeCount) > 2
}

// Statistical helper methods

func (dt *DataTransformer) calculateIntMean(data []int) float64 {
	if len(data) == 0 {
		return 0.0
	}
	sum := 0
	for _, v := range data {
		sum += v
	}
	return float64(sum) / float64(len(data))
}

func (dt *DataTransformer) calculateFloatMean(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func (dt *DataTransformer) calculateIntVariance(data []int) float64 {
	if len(data) == 0 {
		return 0.0
	}
	mean := dt.calculateIntMean(data)
	sum := 0.0
	for _, v := range data {
		diff := float64(v) - mean
		sum += diff * diff
	}
	return sum / float64(len(data))
}

func (dt *DataTransformer) calculateFloatVariance(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}
	mean := dt.calculateFloatMean(data)
	sum := 0.0
	for _, v := range data {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(data))
}

// Transformation functions

func (dt *DataTransformer) normalizeData(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.normalizeValue(value)
	}

	return result
}

func (dt *DataTransformer) normalizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []int:
		return dt.normalizeIntSlice(v)
	case []float64:
		return dt.normalizeFloatSlice(v)
	case int:
		// Single values can't be normalized without context
		return float64(v)
	case float64:
		return v
	default:
		return value
	}
}

func (dt *DataTransformer) normalizeIntSlice(data []int) []float64 {
	if len(data) == 0 {
		return []float64{}
	}

	min := data[0]
	max := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if min == max {
		// All values are the same
		result := make([]float64, len(data))
		for i := range result {
			result[i] = 0.5 // Normalized value for constant data
		}
		return result
	}

	result := make([]float64, len(data))
	range_ := float64(max - min)
	for i, v := range data {
		result[i] = float64(v-min) / range_
	}

	return result
}

func (dt *DataTransformer) normalizeFloatSlice(data []float64) []float64 {
	if len(data) == 0 {
		return []float64{}
	}

	min := data[0]
	max := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if min == max {
		// All values are the same
		result := make([]float64, len(data))
		for i := range result {
			result[i] = 0.5
		}
		return result
	}

	result := make([]float64, len(data))
	range_ := max - min
	for i, v := range data {
		result[i] = (v - min) / range_
	}

	return result
}

func (dt *DataTransformer) standardizeData(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.standardizeValue(value)
	}

	return result
}

func (dt *DataTransformer) standardizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []int:
		return dt.standardizeIntSlice(v)
	case []float64:
		return dt.standardizeFloatSlice(v)
	default:
		return value
	}
}

func (dt *DataTransformer) standardizeIntSlice(data []int) []float64 {
	if len(data) == 0 {
		return []float64{}
	}

	mean := dt.calculateIntMean(data)
	stdDev := math.Sqrt(dt.calculateIntVariance(data))

	if stdDev == 0 {
		// All values are the same
		result := make([]float64, len(data))
		return result // All zeros
	}

	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = (float64(v) - mean) / stdDev
	}

	return result
}

func (dt *DataTransformer) standardizeFloatSlice(data []float64) []float64 {
	if len(data) == 0 {
		return []float64{}
	}

	mean := dt.calculateFloatMean(data)
	stdDev := math.Sqrt(dt.calculateFloatVariance(data))

	if stdDev == 0 {
		// All values are the same
		result := make([]float64, len(data))
		return result // All zeros
	}

	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = (v - mean) / stdDev
	}

	return result
}

func (dt *DataTransformer) logTransform(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.logTransformValue(value)
	}

	return result
}

func (dt *DataTransformer) logTransformValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []int:
		return dt.logTransformIntSlice(v)
	case []float64:
		return dt.logTransformFloatSlice(v)
	case int:
		if v > 0 {
			return math.Log(float64(v))
		}
		return 0.0
	case float64:
		if v > 0 {
			return math.Log(v)
		}
		return 0.0
	default:
		return value
	}
}

func (dt *DataTransformer) logTransformIntSlice(data []int) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		if v > 0 {
			result[i] = math.Log(float64(v))
		} else {
			result[i] = 0.0 // or math.Log(1) for v <= 0
		}
	}
	return result
}

func (dt *DataTransformer) logTransformFloatSlice(data []float64) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		if v > 0 {
			result[i] = math.Log(v)
		} else {
			result[i] = 0.0
		}
	}
	return result
}

func (dt *DataTransformer) removeOutliers(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.removeOutliersFromValue(value)
	}

	return result
}

func (dt *DataTransformer) removeOutliersFromValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []int:
		return dt.removeOutliersFromIntSlice(v)
	case []float64:
		return dt.removeOutliersFromFloatSlice(v)
	default:
		return value
	}
}

func (dt *DataTransformer) removeOutliersFromIntSlice(data []int) []int {
	if len(data) < 3 {
		return data // Too few points to detect outliers
	}

	mean := dt.calculateIntMean(data)
	stdDev := math.Sqrt(dt.calculateIntVariance(data))
	threshold := 2.0 * stdDev

	result := make([]int, 0)
	for _, v := range data {
		if math.Abs(float64(v)-mean) <= threshold {
			result = append(result, v)
		}
	}

	// If too many outliers removed, return original data
	if len(result) < len(data)/2 {
		return data
	}

	return result
}

func (dt *DataTransformer) removeOutliersFromFloatSlice(data []float64) []float64 {
	if len(data) < 3 {
		return data
	}

	mean := dt.calculateFloatMean(data)
	stdDev := math.Sqrt(dt.calculateFloatVariance(data))
	threshold := 2.0 * stdDev

	result := make([]float64, 0)
	for _, v := range data {
		if math.Abs(v-mean) <= threshold {
			result = append(result, v)
		}
	}

	// If too many outliers removed, return original data
	if len(result) < len(data)/2 {
		return data
	}

	return result
}

func (dt *DataTransformer) smoothData(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.smoothValue(value)
	}

	return result
}

func (dt *DataTransformer) smoothValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []int:
		return dt.smoothIntSlice(v)
	case []float64:
		return dt.smoothFloatSlice(v)
	default:
		return value
	}
}

func (dt *DataTransformer) smoothIntSlice(data []int) []float64 {
	if len(data) < 3 {
		// Convert to float64 for consistency
		result := make([]float64, len(data))
		for i, v := range data {
			result[i] = float64(v)
		}
		return result
	}

	windowSize := 3
	result := make([]float64, len(data))

	// Handle edges
	result[0] = float64(data[0])
	result[len(data)-1] = float64(data[len(data)-1])

	// Apply moving average
	for i := 1; i < len(data)-1; i++ {
		sum := 0
		count := 0
		for j := i - windowSize/2; j <= i+windowSize/2; j++ {
			if j >= 0 && j < len(data) {
				sum += data[j]
				count++
			}
		}
		result[i] = float64(sum) / float64(count)
	}

	return result
}

func (dt *DataTransformer) smoothFloatSlice(data []float64) []float64 {
	if len(data) < 3 {
		return data
	}

	windowSize := 3
	result := make([]float64, len(data))

	// Handle edges
	result[0] = data[0]
	result[len(data)-1] = data[len(data)-1]

	// Apply moving average
	for i := 1; i < len(data)-1; i++ {
		sum := 0.0
		count := 0
		for j := i - windowSize/2; j <= i+windowSize/2; j++ {
			if j >= 0 && j < len(data) {
				sum += data[j]
				count++
			}
		}
		result[i] = sum / float64(count)
	}

	return result
}

func (dt *DataTransformer) aggregateData(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.aggregateValue(value)
	}

	return result
}

func (dt *DataTransformer) aggregateValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []int:
		return dt.aggregateIntSlice(v)
	case []float64:
		return dt.aggregateFloatSlice(v)
	default:
		return value
	}
}

func (dt *DataTransformer) aggregateIntSlice(data []int) map[string]interface{} {
	if len(data) == 0 {
		return map[string]interface{}{}
	}

	// Sort for percentile calculations
	sorted := make([]int, len(data))
	copy(sorted, data)
	sort.Ints(sorted)

	result := make(map[string]interface{})
	result["count"] = len(data)
	result["sum"] = dt.sumIntSlice(data)
	result["mean"] = dt.calculateIntMean(data)
	result["median"] = dt.medianIntSlice(sorted)
	result["min"] = sorted[0]
	result["max"] = sorted[len(sorted)-1]
	result["range"] = sorted[len(sorted)-1] - sorted[0]
	result["std_dev"] = math.Sqrt(dt.calculateIntVariance(data))
	result["q1"] = dt.percentileIntSlice(sorted, 25)
	result["q3"] = dt.percentileIntSlice(sorted, 75)

	return result
}

func (dt *DataTransformer) aggregateFloatSlice(data []float64) map[string]interface{} {
	if len(data) == 0 {
		return map[string]interface{}{}
	}

	// Sort for percentile calculations
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	result := make(map[string]interface{})
	result["count"] = len(data)
	result["sum"] = dt.sumFloatSlice(data)
	result["mean"] = dt.calculateFloatMean(data)
	result["median"] = dt.medianFloatSlice(sorted)
	result["min"] = sorted[0]
	result["max"] = sorted[len(sorted)-1]
	result["range"] = sorted[len(sorted)-1] - sorted[0]
	result["std_dev"] = math.Sqrt(dt.calculateFloatVariance(data))
	result["q1"] = dt.percentileFloatSlice(sorted, 25)
	result["q3"] = dt.percentileFloatSlice(sorted, 75)

	return result
}

// Helper methods for aggregation

func (dt *DataTransformer) sumIntSlice(data []int) int {
	sum := 0
	for _, v := range data {
		sum += v
	}
	return sum
}

func (dt *DataTransformer) sumFloatSlice(data []float64) float64 {
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum
}

func (dt *DataTransformer) medianIntSlice(sortedData []int) float64 {
	n := len(sortedData)
	if n%2 == 0 {
		return float64(sortedData[n/2-1]+sortedData[n/2]) / 2.0
	}
	return float64(sortedData[n/2])
}

func (dt *DataTransformer) medianFloatSlice(sortedData []float64) float64 {
	n := len(sortedData)
	if n%2 == 0 {
		return (sortedData[n/2-1] + sortedData[n/2]) / 2.0
	}
	return sortedData[n/2]
}

func (dt *DataTransformer) percentileIntSlice(sortedData []int, percentile float64) float64 {
	n := len(sortedData)
	index := (percentile / 100.0) * float64(n-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return float64(sortedData[lower])
	}

	weight := index - float64(lower)
	return float64(sortedData[lower])*(1-weight) + float64(sortedData[upper])*weight
}

func (dt *DataTransformer) percentileFloatSlice(sortedData []float64, percentile float64) float64 {
	n := len(sortedData)
	index := (percentile / 100.0) * float64(n-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedData[lower]
	}

	weight := index - float64(lower)
	return sortedData[lower]*(1-weight) + sortedData[upper]*weight
}

func (dt *DataTransformer) encodeData(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.encodeValue(value)
	}

	return result
}

func (dt *DataTransformer) encodeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []string:
		return dt.encodeStringSlice(v)
	case string:
		return dt.encodeString(v)
	default:
		return value
	}
}

func (dt *DataTransformer) encodeStringSlice(data []string) []int {
	// Create encoding map
	encodingMap := make(map[string]int)
	nextCode := 0

	result := make([]int, len(data))
	for i, s := range data {
		if code, exists := encodingMap[s]; exists {
			result[i] = code
		} else {
			encodingMap[s] = nextCode
			result[i] = nextCode
			nextCode++
		}
	}

	return result
}

func (dt *DataTransformer) encodeString(s string) int {
	// Simple hash-based encoding
	hash := 0
	for _, char := range s {
		hash = hash*31 + int(char)
	}
	return hash
}

func (dt *DataTransformer) binData(data interface{}) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})
	for key, value := range dataMap {
		result[key] = dt.binValue(value)
	}

	return result
}

func (dt *DataTransformer) binValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []int:
		return dt.binIntSlice(v, 10) // 10 bins by default
	case []float64:
		return dt.binFloatSlice(v, 10)
	default:
		return value
	}
}

func (dt *DataTransformer) binIntSlice(data []int, numBins int) []int {
	if len(data) == 0 || numBins <= 0 {
		return data
	}

	min := data[0]
	max := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if min == max {
		// All values are the same
		result := make([]int, len(data))
		return result // All in bin 0
	}

	binWidth := float64(max-min) / float64(numBins)
	result := make([]int, len(data))

	for i, v := range data {
		binIndex := int(float64(v-min) / binWidth)
		if binIndex >= numBins {
			binIndex = numBins - 1 // Handle edge case
		}
		result[i] = binIndex
	}

	return result
}

func (dt *DataTransformer) binFloatSlice(data []float64, numBins int) []int {
	if len(data) == 0 || numBins <= 0 {
		return []int{}
	}

	min := data[0]
	max := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if min == max {
		// All values are the same
		result := make([]int, len(data))
		return result // All in bin 0
	}

	binWidth := (max - min) / float64(numBins)
	result := make([]int, len(data))

	for i, v := range data {
		binIndex := int((v - min) / binWidth)
		if binIndex >= numBins {
			binIndex = numBins - 1 // Handle edge case
		}
		result[i] = binIndex
	}

	return result
}

// Validation functions

func (dt *DataTransformer) validateNormalizedData(data interface{}) bool {
	// Check if data is in [0, 1] range
	switch v := data.(type) {
	case []float64:
		for _, val := range v {
			if val < 0.0 || val > 1.0 {
				return false
			}
		}
		return true
	default:
		return true // Non-numeric data passes validation
	}
}

func (dt *DataTransformer) validateStandardizedData(data interface{}) bool {
	// Check if data has approximately zero mean and unit variance
	switch v := data.(type) {
	case []float64:
		if len(v) == 0 {
			return true
		}
		mean := dt.calculateFloatMean(v)
		stdDev := math.Sqrt(dt.calculateFloatVariance(v))
		// Allow some tolerance
		return math.Abs(mean) < 0.1 && math.Abs(stdDev-1.0) < 0.1
	default:
		return true
	}
}

func (dt *DataTransformer) validateLogTransform(data interface{}) bool {
	// Check if all values are positive (log transform requirement)
	switch v := data.(type) {
	case []float64:
		for _, val := range v {
			if val <= 0 {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func (dt *DataTransformer) validateOutlierRemoval(data interface{}) bool {
	// Check if reasonable amount of data remains
	switch v := data.(type) {
	case []int:
		return len(v) > 0
	case []float64:
		return len(v) > 0
	default:
		return true
	}
}

func (dt *DataTransformer) validateSmoothedData(data interface{}) bool {
	// Check if data length is preserved
	switch v := data.(type) {
	case []float64:
		return len(v) > 0
	default:
		return true
	}
}

func (dt *DataTransformer) validateAggregatedData(data interface{}) bool {
	// Check if aggregation contains expected fields
	switch v := data.(type) {
	case map[string]interface{}:
		requiredFields := []string{"count", "mean", "min", "max"}
		for _, field := range requiredFields {
			if _, exists := v[field]; !exists {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func (dt *DataTransformer) validateEncodedData(data interface{}) bool {
	// Check if encoding produced numeric data
	switch v := data.(type) {
	case []int:
		return len(v) > 0
	case int:
		return true
	default:
		return true
	}
}

func (dt *DataTransformer) validateBinnedData(data interface{}) bool {
	// Check if binning produced valid bin indices
	switch v := data.(type) {
	case []int:
		for _, bin := range v {
			if bin < 0 {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// Cache management methods

func (dt *DataTransformer) generateCacheKey(data interface{}) string {
	// Generate a simple hash of the data
	return fmt.Sprintf("transform-%d", dt.simpleHash(fmt.Sprintf("%v", data)))
}

func (dt *DataTransformer) simpleHash(s string) uint32 {
	hash := uint32(0)
	for i, c := range s {
		hash = hash*31 + uint32(c)
		if i > 1000 { // Limit hash calculation for large data
			break
		}
	}
	return hash
}

func (dt *DataTransformer) getCachedTransformation(key string) interface{} {
	if result, exists := dt.cache[key]; exists {
		return result
	}
	return nil
}

func (dt *DataTransformer) cacheTransformation(key string, result interface{}) {
	dt.cache[key] = result

	// Cleanup cache if it gets too large
	if len(dt.cache) > 100 {
		dt.cleanupCache()
	}
}

func (dt *DataTransformer) cleanupCache() {
	// Simple cleanup: remove half of the cache entries
	count := 0
	for key := range dt.cache {
		delete(dt.cache, key)
		count++
		if count >= len(dt.cache)/2 {
			break
		}
	}
}