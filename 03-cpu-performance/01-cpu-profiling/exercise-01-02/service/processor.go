package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"exercise-01-02/data"
)

// Processor handles business logic processing
type Processor struct {
	analyzer    *Analyzer
	transformer *DataTransformer
	validator   *BusinessValidator
	cache       map[string]*ProcessResult
	cacheMutex  sync.RWMutex
	stats       *ProcessorStats
	statsMutex  sync.Mutex
}

// ProcessorStats tracks processor performance
type ProcessorStats struct {
	TotalProcessed   int64
	SuccessfulCount  int64
	ErrorCount       int64
	AverageTime      time.Duration
	TotalTime        time.Duration
}

// ProcessResult represents the result of data processing
type ProcessResult struct {
	Data         []int
	StringData   []string
	Analysis     *AnalysisResult
	Transformed  interface{}
	Validated    bool
	ProcessedAt  time.Time
	ProcessingTime time.Duration
	Metadata     map[string]interface{}
}

// DataTransformer handles data transformation operations
type DataTransformer struct {
	transformationRules map[string]TransformRule
	cache              map[string]interface{}
}

// TransformRule defines a data transformation rule
type TransformRule struct {
	Name        string
	Function    func(interface{}) interface{}
	Validation  func(interface{}) bool
	Description string
}

// BusinessValidator validates business logic rules
type BusinessValidator struct {
	rules       []ValidationRule
	errorCount  int64
	successCount int64
}

// ValidationRule defines a business validation rule
type ValidationRule struct {
	Name        string
	Validator   func(interface{}) bool
	ErrorMessage string
	Severity    string
}

// NewProcessor creates a new processor instance
func NewProcessor() *Processor {
	return &Processor{
		analyzer:    NewAnalyzer(),
		transformer: NewDataTransformer(),
		validator:   NewBusinessValidator(),
		cache:       make(map[string]*ProcessResult),
		stats:       &ProcessorStats{},
	}
}

// NewDataTransformer creates a new data transformer
func NewDataTransformer() *DataTransformer {
	transformer := &DataTransformer{
		transformationRules: make(map[string]TransformRule),
		cache:              make(map[string]interface{}),
	}
	
	// Initialize transformation rules
	transformer.initializeRules()
	return transformer
}

// NewBusinessValidator creates a new business validator
func NewBusinessValidator() *BusinessValidator {
	validator := &BusinessValidator{
		rules: make([]ValidationRule, 0),
	}
	
	// Initialize validation rules
	validator.initializeRules()
	return validator
}

// ProcessData processes request data through the complete pipeline
func (p *Processor) ProcessData(request interface{}) (result *ProcessResult) {
	start := time.Now()
	defer func() {
		p.statsMutex.Lock()
		p.stats.TotalProcessed++
		p.stats.TotalTime += time.Since(start)
		if p.stats.TotalProcessed > 0 {
			p.stats.AverageTime = time.Duration(int64(p.stats.TotalTime) / p.stats.TotalProcessed)
		}
		p.statsMutex.Unlock()
		// Top-level panic recovery
		if r := recover(); r != nil {
			p.statsMutex.Lock()
			p.stats.ErrorCount++
			p.statsMutex.Unlock()
			// Return minimal fallback result
			result = &ProcessResult{
				Data:           []int{},
				StringData:     []string{},
				Analysis:       nil,
				Transformed:    request,
				Validated:      false,
				ProcessedAt:    time.Now(),
				ProcessingTime: time.Since(start),
				Metadata:       map[string]interface{}{"error": "panic_recovered"},
			}
		}
	}()

	// Check cache first
	cacheKey := p.generateCacheKey(request)
	if cacheKey != "" {
		if cached := p.getCachedResult(cacheKey); cached != nil {
			return cached
		}
	}

	// Parse and extract data
	parsedData := p.parseRequestData(request)
	if parsedData == nil {
		p.stats.ErrorCount++
		// Create minimal fallback data
		parsedData = map[string]interface{}{
			"data_type": "fallback",
			"raw_data": request,
			"parsed_at": time.Now(),
		}
	}

	// Transform data
	var transformed interface{} = parsedData
	if p.transformer != nil {
		if result := p.transformer.TransformData(parsedData); result != nil {
			transformed = result
		}
	}

	// Validate business rules
	validated := false
	if p.validator != nil {
		validated = p.validator.ValidateData(transformed)
	}
	
	if !validated {
		p.statsMutex.Lock()
		p.stats.ErrorCount++
		p.statsMutex.Unlock()
		// Return partial result even if validation fails
		var analysis *AnalysisResult
		if p.analyzer != nil {
			analysis = p.analyzer.AnalyzeData(transformed)
		}
		result := p.generateResult(parsedData, transformed, analysis, start)
		result.Validated = false
		return result
	}

	// Perform analysis
	var analysis *AnalysisResult
	if p.analyzer != nil {
		analysis = p.analyzer.AnalyzeData(transformed)
	}

	// Generate final result
	result = p.generateResult(parsedData, transformed, analysis, start)
	result.Validated = true

	// Cache result
	if cacheKey != "" {
		p.cacheResult(cacheKey, result)
	}

	p.statsMutex.Lock()
	p.stats.SuccessfulCount++
	p.statsMutex.Unlock()
	return result
}

// parseRequestData extracts and parses data from the request
func (p *Processor) parseRequestData(request interface{}) map[string]interface{} {
	parsed := make(map[string]interface{})

	// Handle different request types
	switch req := request.(type) {
	case map[string]interface{}:
		// Direct map processing
		for key, value := range req {
			parsed[key] = p.parseValue(value)
		}
	case string:
		// JSON string parsing
		if err := json.Unmarshal([]byte(req), &parsed); err != nil {
			// Fallback to simple string processing
			parsed["raw_string"] = req
			parsed["string_analysis"] = p.analyzeString(req)
		}
	case []byte:
		// Byte array processing
		if err := json.Unmarshal(req, &parsed); err != nil {
			// Fallback to binary analysis
			parsed["raw_bytes"] = req
			parsed["binary_analysis"] = p.analyzeBinary(req)
		}
	default:
		// Generic processing
		parsed["raw_data"] = request
		parsed["type"] = fmt.Sprintf("%T", request)
	}

	// Add parsing metadata
	parsed["parsed_at"] = time.Now()
	parsed["parser_version"] = "1.0"
	parsed["data_size"] = p.calculateDataSize(request)

	return parsed
}

// parseValue recursively parses individual values
func (p *Processor) parseValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Try to parse as JSON
		var jsonValue interface{}
		if err := json.Unmarshal([]byte(v), &jsonValue); err == nil {
			return jsonValue
		}
		// Try to parse as number
		if num := p.tryParseNumber(v); num != nil {
			return num
		}
		// Return as string with analysis
		return map[string]interface{}{
			"value":    v,
			"type":     "string",
			"length":   len(v),
			"analysis": p.analyzeString(v),
		}
	case []interface{}:
		// Process array
		processed := make([]interface{}, len(v))
		for i, item := range v {
			processed[i] = p.parseValue(item)
		}
		return processed
	case map[string]interface{}:
		// Process nested map
		processed := make(map[string]interface{})
		for key, val := range v {
			processed[key] = p.parseValue(val)
		}
		return processed
	default:
		return value
	}
}

// tryParseNumber attempts to parse a string as a number
func (p *Processor) tryParseNumber(s string) interface{} {
	// Simple number parsing logic
	if strings.Contains(s, ".") {
		// Try float parsing
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
			return f
		}
	} else {
		// Try integer parsing
		var i int
		if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
			return i
		}
	}
	return nil
}

// analyzeString performs string analysis
func (p *Processor) analyzeString(s string) map[string]interface{} {
	analysis := make(map[string]interface{})

	// Basic metrics
	analysis["length"] = len(s)
	analysis["word_count"] = len(strings.Fields(s))
	analysis["line_count"] = strings.Count(s, "\n") + 1

	// Character analysis
	charCounts := make(map[rune]int)
	for _, r := range s {
		charCounts[r]++
	}
	analysis["unique_chars"] = len(charCounts)
	analysis["most_frequent_char"] = p.findMostFrequentChar(charCounts)

	// Pattern detection
	patterns := p.detectPatterns(s)
	analysis["patterns"] = patterns

	// Complexity metrics
	analysis["complexity_score"] = p.calculateStringComplexity(s)

	return analysis
}

// findMostFrequentChar finds the most frequent character
func (p *Processor) findMostFrequentChar(charCounts map[rune]int) rune {
	maxCount := 0
	var mostFrequent rune
	for char, count := range charCounts {
		if count > maxCount {
			maxCount = count
			mostFrequent = char
		}
	}
	return mostFrequent
}

// detectPatterns detects common patterns in strings
func (p *Processor) detectPatterns(s string) []string {
	patterns := make([]string, 0)

	// Email pattern
	if strings.Contains(s, "@") && strings.Contains(s, ".") {
		patterns = append(patterns, "email")
	}

	// URL pattern
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		patterns = append(patterns, "url")
	}

	// Phone pattern (simple)
	if strings.Count(s, "-") >= 2 && len(s) >= 10 {
		patterns = append(patterns, "phone")
	}

	// JSON pattern
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		patterns = append(patterns, "json")
	}

	return patterns
}

// calculateStringComplexity calculates string complexity score
func (p *Processor) calculateStringComplexity(s string) float64 {
	if len(s) == 0 {
		return 0.0
	}

	// Factor in length
	lengthScore := float64(len(s)) * 0.1

	// Factor in character diversity
	uniqueChars := make(map[rune]bool)
	for _, r := range s {
		uniqueChars[r] = true
	}
	diversityScore := float64(len(uniqueChars)) / float64(len(s))

	// Factor in special characters
	specialCount := 0
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ') {
			specialCount++
		}
	}
	specialScore := float64(specialCount) * 0.5

	return lengthScore + diversityScore*10 + specialScore
}

// analyzeBinary performs binary data analysis
func (p *Processor) analyzeBinary(data []byte) map[string]interface{} {
	analysis := make(map[string]interface{})

	// Basic metrics
	analysis["size"] = len(data)
	analysis["entropy"] = p.calculateBinaryEntropy(data)

	// Byte frequency
	byteFreq := make(map[byte]int)
	for _, b := range data {
		byteFreq[b]++
	}
	analysis["unique_bytes"] = len(byteFreq)
	analysis["most_frequent_byte"] = p.findMostFrequentByte(byteFreq)

	// Pattern detection
	analysis["has_null_bytes"] = strings.Contains(string(data), "\x00")
	analysis["printable_ratio"] = p.calculatePrintableRatio(data)

	return analysis
}

// calculateBinaryEntropy calculates entropy of binary data
func (p *Processor) calculateBinaryEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	entropy := 0.0
	for _, count := range freq {
		if count > 0 {
			p := float64(count) / float64(len(data))
			entropy -= p * p // Simplified entropy calculation
		}
	}

	return entropy
}

// findMostFrequentByte finds the most frequent byte
func (p *Processor) findMostFrequentByte(byteFreq map[byte]int) byte {
	maxCount := 0
	var mostFrequent byte
	for b, count := range byteFreq {
		if count > maxCount {
			maxCount = count
			mostFrequent = b
		}
	}
	return mostFrequent
}

// calculatePrintableRatio calculates ratio of printable characters
func (p *Processor) calculatePrintableRatio(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	printableCount := 0
	for _, b := range data {
		if b >= 32 && b <= 126 {
			printableCount++
		}
	}

	return float64(printableCount) / float64(len(data))
}

// calculateDataSize calculates the size of arbitrary data
func (p *Processor) calculateDataSize(data interface{}) int {
	switch d := data.(type) {
	case string:
		return len(d)
	case []byte:
		return len(d)
	case map[string]interface{}:
		size := 0
		for key, value := range d {
			size += len(key) + p.calculateDataSize(value)
		}
		return size
	case []interface{}:
		size := 0
		for _, item := range d {
			size += p.calculateDataSize(item)
		}
		return size
	default:
		return len(fmt.Sprintf("%v", data))
	}
}

// generateResult creates the final processing result
func (p *Processor) generateResult(parsed map[string]interface{}, transformed interface{}, 
	analysis *AnalysisResult, startTime time.Time) *ProcessResult {
	
	// Extract numeric data for algorithms
	numericData := p.extractNumericData(parsed)
	stringData := p.extractStringData(parsed)

	// Create metadata
	metadata := make(map[string]interface{})
	metadata["processing_stages"] = []string{"parse", "transform", "validate", "analyze"}
	metadata["data_types"] = p.analyzeDataTypes(parsed)
	metadata["complexity_metrics"] = p.calculateComplexityMetrics(parsed)

	return &ProcessResult{
		Data:           numericData,
		StringData:     stringData,
		Analysis:       analysis,
		Transformed:    transformed,
		Validated:      true,
		ProcessedAt:    time.Now(),
		ProcessingTime: time.Since(startTime),
		Metadata:       metadata,
	}
}

// extractNumericData extracts numeric data for algorithm processing
func (p *Processor) extractNumericData(parsed map[string]interface{}) []int {
	numericData := make([]int, 0)

	// Recursively extract numbers
	p.extractNumbersRecursive(parsed, &numericData)

	// If no numbers found, generate some based on data characteristics
	if len(numericData) == 0 {
		generator := data.NewDataGenerator()
		intValues := generator.GenerateValues(100, "integer")
		numericData = make([]int, len(intValues))
		for i, v := range intValues {
			numericData[i] = v.(int)
		}
	}

	// Sort for consistent processing
	sort.Ints(numericData)

	return numericData
}

// extractNumbersRecursive recursively extracts numbers from data
func (p *Processor) extractNumbersRecursive(data interface{}, result *[]int) {
	switch d := data.(type) {
	case int:
		*result = append(*result, d)
	case float64:
		*result = append(*result, int(d))
	case map[string]interface{}:
		for _, value := range d {
			p.extractNumbersRecursive(value, result)
		}
	case []interface{}:
		for _, item := range d {
			p.extractNumbersRecursive(item, result)
		}
	case string:
		// Try to extract numbers from strings
		if num := p.tryParseNumber(d); num != nil {
			if i, ok := num.(int); ok {
				*result = append(*result, i)
			} else if f, ok := num.(float64); ok {
				*result = append(*result, int(f))
			}
		}
	}
}

// extractStringData extracts string data for processing
func (p *Processor) extractStringData(parsed map[string]interface{}) []string {
	stringData := make([]string, 0)

	// Recursively extract strings
	p.extractStringsRecursive(parsed, &stringData)

	// If no strings found, generate some
	if len(stringData) == 0 {
		generator := data.NewDataGenerator()
		stringValues := generator.GenerateValues(50, "string")
		stringData = make([]string, len(stringValues))
		for i, v := range stringValues {
			stringData[i] = v.(string)
		}
	}

	return stringData
}

// extractStringsRecursive recursively extracts strings from data
func (p *Processor) extractStringsRecursive(data interface{}, result *[]string) {
	switch d := data.(type) {
	case string:
		if len(d) > 0 {
			*result = append(*result, d)
		}
	case map[string]interface{}:
		for key, value := range d {
			*result = append(*result, key)
			p.extractStringsRecursive(value, result)
		}
	case []interface{}:
		for _, item := range d {
			p.extractStringsRecursive(item, result)
		}
	}
}

// analyzeDataTypes analyzes the types of data in the parsed structure
func (p *Processor) analyzeDataTypes(parsed map[string]interface{}) map[string]int {
	types := make(map[string]int)
	p.analyzeTypesRecursive(parsed, types)
	return types
}

// analyzeTypesRecursive recursively analyzes data types
func (p *Processor) analyzeTypesRecursive(data interface{}, types map[string]int) {
	switch data.(type) {
	case string:
		types["string"]++
	case int:
		types["int"]++
	case float64:
		types["float"]++
	case bool:
		types["bool"]++
	case map[string]interface{}:
		types["object"]++
		for _, value := range data.(map[string]interface{}) {
			p.analyzeTypesRecursive(value, types)
		}
	case []interface{}:
		types["array"]++
		for _, item := range data.([]interface{}) {
			p.analyzeTypesRecursive(item, types)
		}
	default:
		types["other"]++
	}
}

// calculateComplexityMetrics calculates complexity metrics for the data
func (p *Processor) calculateComplexityMetrics(parsed map[string]interface{}) map[string]interface{} {
	metrics := make(map[string]interface{})

	// Depth calculation
	maxDepth := p.calculateMaxDepth(parsed, 0)
	metrics["max_depth"] = maxDepth

	// Node count
	nodeCount := p.countNodes(parsed)
	metrics["node_count"] = nodeCount

	// Complexity score
	complexityScore := float64(maxDepth)*2 + float64(nodeCount)*0.1
	metrics["complexity_score"] = complexityScore

	return metrics
}

// calculateMaxDepth calculates maximum depth of nested data
func (p *Processor) calculateMaxDepth(data interface{}, currentDepth int) int {
	maxDepth := currentDepth

	switch d := data.(type) {
	case map[string]interface{}:
		for _, value := range d {
			depth := p.calculateMaxDepth(value, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	case []interface{}:
		for _, item := range d {
			depth := p.calculateMaxDepth(item, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}

// countNodes counts total nodes in the data structure
func (p *Processor) countNodes(data interface{}) int {
	count := 1 // Count current node

	switch d := data.(type) {
	case map[string]interface{}:
		for _, value := range d {
			count += p.countNodes(value)
		}
	case []interface{}:
		for _, item := range d {
			count += p.countNodes(item)
		}
	}

	return count
}

// Cache management methods

// generateCacheKey generates a cache key for the request
func (p *Processor) generateCacheKey(request interface{}) string {
	// Safely convert request to string, handling potential panics
	var requestStr string
	defer func() {
		if r := recover(); r != nil {
			// If formatting fails, use a fallback
			requestStr = fmt.Sprintf("fallback-%p", request)
		}
	}()
	
	// Try to format the request safely
	switch req := request.(type) {
	case map[string]interface{}:
		// For maps, create a simplified string representation
		requestStr = p.formatMapSafely(req)
	default:
		requestStr = fmt.Sprintf("%T", request)
	}
	
	return fmt.Sprintf("proc-%d", p.simpleHash(requestStr))
}

// formatMapSafely safely formats a map for cache key generation
func (p *Processor) formatMapSafely(m map[string]interface{}) string {
	var parts []string
	for key, value := range m {
		switch v := value.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s:%s", key, v))
		case int, int64, float64, bool:
			parts = append(parts, fmt.Sprintf("%s:%v", key, v))
		case []byte:
			parts = append(parts, fmt.Sprintf("%s:bytes[%d]", key, len(v)))
		default:
			// For complex types like time.Time, use type name
			parts = append(parts, fmt.Sprintf("%s:%T", key, v))
		}
	}
	return strings.Join(parts, "|")
}

// simpleHash implements a simple hash function
func (p *Processor) simpleHash(s string) uint32 {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// getCachedResult retrieves a cached result
func (p *Processor) getCachedResult(key string) *ProcessResult {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()
	
	if result, exists := p.cache[key]; exists {
		// Check if cache is still valid (5 minutes)
		if time.Since(result.ProcessedAt) < time.Minute*5 {
			return result
		}
		// Remove expired cache (upgrade to write lock)
		p.cacheMutex.RUnlock()
		p.cacheMutex.Lock()
		delete(p.cache, key)
		p.cacheMutex.Unlock()
		p.cacheMutex.RLock()
	}
	return nil
}

// cacheResult caches a processing result
func (p *Processor) cacheResult(key string, result *ProcessResult) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	
	p.cache[key] = result
	
	// Cleanup old cache entries if cache is too large
	if len(p.cache) > 100 {
		p.cleanupCache()
	}
}

// cleanupCache removes old cache entries
// Note: This method assumes the caller already holds the cache write lock
func (p *Processor) cleanupCache() {
	now := time.Now()
	for key, result := range p.cache {
		if now.Sub(result.ProcessedAt) > time.Minute*5 {
			delete(p.cache, key)
		}
	}
}

// GetStats returns processor statistics
func (p *Processor) GetStats() *ProcessorStats {
	p.statsMutex.Lock()
	defer p.statsMutex.Unlock()
	
	// Return a copy to avoid race conditions
	statsCopy := *p.stats
	return &statsCopy
}