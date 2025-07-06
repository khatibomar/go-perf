package server

import (
	"fmt"
	"strings"
	"time"
)

// Middleware handles request preprocessing
type Middleware struct {
	logger    *Logger
	auth      *AuthHandler
	cache     *CacheManager
	metrics   *MetricsCollector
}

// Logger handles request logging
type Logger struct {
	requestCount int64
	errorCount   int64
}

// AuthHandler handles authentication and authorization
type AuthHandler struct {
	validTokens map[string]bool
	sessions    map[string]*Session
}

// Session represents a user session
type Session struct {
	UserID    string
	CreatedAt time.Time
	LastSeen  time.Time
	Permissions []string
}

// CacheManager handles request caching
type CacheManager struct {
	cache     map[string]*CacheEntry
	hitCount  int64
	missCount int64
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Data      interface{}
	CreatedAt time.Time
	TTL       time.Duration
}

// MetricsCollector collects middleware metrics
type MetricsCollector struct {
	requestTimes    []time.Duration
	processingTimes map[string][]time.Duration
	errorCounts    map[string]int64
}

// NewMiddleware creates a new middleware instance
func NewMiddleware() *Middleware {
	return &Middleware{
		logger: &Logger{},
		auth: &AuthHandler{
			validTokens: make(map[string]bool),
			sessions:    make(map[string]*Session),
		},
		cache: &CacheManager{
			cache: make(map[string]*CacheEntry),
		},
		metrics: &MetricsCollector{
			processingTimes: make(map[string][]time.Duration),
			errorCounts:    make(map[string]int64),
		},
	}
}

// ProcessRequest processes a request through the middleware chain
func (m *Middleware) ProcessRequest(data *RequestData) *RequestData {
	// Check if data is nil
	if data == nil {
		return nil
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.metrics.requestTimes = append(m.metrics.requestTimes, duration)
	}()

	// Log request
	m.logRequest(data)

	// Check cache
	if cachedData := m.checkCache(data); cachedData != nil {
		return cachedData
	}

	// Authenticate request
	if !m.authenticateRequest(data) {
		m.logger.errorCount++
		return nil
	}

	// Authorize request
	if !m.authorizeRequest(data) {
		m.logger.errorCount++
		return nil
	}

	// Transform request
	transformed := m.transformRequest(data)

	// Validate transformed request
	validated := m.validateTransformedRequest(transformed)

	// Cache result if appropriate
	m.cacheRequest(validated)

	// Collect metrics
	m.collectMetrics(validated)

	return validated
}

// logRequest logs the incoming request
func (m *Middleware) logRequest(data *RequestData) {
	m.logger.requestCount++

	// Simulate expensive logging operations
	logEntry := m.createLogEntry(data)
	m.processLogEntry(logEntry)
	m.writeLogEntry(logEntry)
}

// createLogEntry creates a detailed log entry
func (m *Middleware) createLogEntry(data *RequestData) map[string]interface{} {
	entry := make(map[string]interface{})

	// Basic information
	entry["request_id"] = data.ID
	entry["timestamp"] = time.Now().Format(time.RFC3339)
	entry["payload_size"] = len(data.Payload)

	// Header analysis
	headerInfo := make(map[string]interface{})
	headerInfo["count"] = len(data.Headers)
	headerInfo["total_size"] = m.calculateHeaderSize(data.Headers)
	headerInfo["content_type"] = data.Headers["Content-Type"]
	entry["headers"] = headerInfo

	// Metadata analysis
	metadataInfo := make(map[string]interface{})
	metadataInfo["count"] = len(data.Metadata)
	metadataInfo["types"] = m.analyzeMetadataTypes(data.Metadata)
	entry["metadata"] = metadataInfo

	// Request fingerprint
	entry["fingerprint"] = m.calculateRequestFingerprint(data)

	return entry
}

// calculateHeaderSize calculates total size of headers
func (m *Middleware) calculateHeaderSize(headers map[string]string) int {
	totalSize := 0
	for key, value := range headers {
		totalSize += len(key) + len(value) + 4 // +4 for ": " and "\r\n"
	}
	return totalSize
}

// analyzeMetadataTypes analyzes the types of metadata values
func (m *Middleware) analyzeMetadataTypes(metadata map[string]interface{}) map[string]int {
	types := make(map[string]int)
	for _, value := range metadata {
		switch value.(type) {
		case string:
			types["string"]++
		case int:
			types["int"]++
		case float64:
			types["float"]++
		case bool:
			types["bool"]++
		default:
			types["other"]++
		}
	}
	return types
}

// calculateRequestFingerprint creates a unique fingerprint for the request
func (m *Middleware) calculateRequestFingerprint(data *RequestData) string {
	// Simple fingerprinting algorithm
	fingerprint := fmt.Sprintf("%d", data.ID)
	
	// Add payload hash
	payloadHash := m.simpleHash(string(data.Payload))
	fingerprint += fmt.Sprintf("-%d", payloadHash)
	
	// Add header hash
	headerHash := m.hashHeaders(data.Headers)
	fingerprint += fmt.Sprintf("-%d", headerHash)
	
	return fingerprint
}

// simpleHash implements a simple hash function
func (m *Middleware) simpleHash(s string) uint32 {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// hashHeaders creates a hash of all headers
func (m *Middleware) hashHeaders(headers map[string]string) uint32 {
	hash := uint32(0)
	for key, value := range headers {
		hash += m.simpleHash(key + value)
	}
	return hash
}

// processLogEntry processes the log entry (expensive operation)
func (m *Middleware) processLogEntry(entry map[string]interface{}) {
	// Simulate log processing
	for key, value := range entry {
		// Simulate expensive string operations
		processedKey := strings.ToUpper(key)
		processedValue := fmt.Sprintf("%v", value)
		
		// Simulate validation
		if len(processedKey) > 0 && len(processedValue) > 0 {
			// Simulate additional processing
			_ = strings.Contains(processedValue, "error")
		}
	}
}

// writeLogEntry writes the log entry (I/O simulation)
func (m *Middleware) writeLogEntry(entry map[string]interface{}) {
	// Simulate log writing overhead
	logLine := fmt.Sprintf("%v", entry)
	_ = len(logLine) // Simulate writing
}

// checkCache checks if the request is cached
func (m *Middleware) checkCache(data *RequestData) *RequestData {
	cacheKey := m.generateCacheKey(data)
	
	if entry, exists := m.cache.cache[cacheKey]; exists {
		// Check if cache entry is still valid
		if time.Since(entry.CreatedAt) < entry.TTL {
			m.cache.hitCount++
			if cachedData, ok := entry.Data.(*RequestData); ok {
				return cachedData
			}
		} else {
			// Cache expired, remove entry
			delete(m.cache.cache, cacheKey)
		}
	}
	
	m.cache.missCount++
	return nil
}

// generateCacheKey generates a cache key for the request
func (m *Middleware) generateCacheKey(data *RequestData) string {
	// Generate cache key based on request characteristics
	key := fmt.Sprintf("req-%d", data.ID)
	
	// Add payload hash for uniqueness
	payloadHash := m.simpleHash(string(data.Payload))
	key += fmt.Sprintf("-%d", payloadHash)
	
	return key
}

// authenticateRequest performs request authentication
func (m *Middleware) authenticateRequest(data *RequestData) bool {
	// Extract authentication token
	token := data.Headers["Authorization"]
	if token == "" {
		// Allow requests without token for demo purposes
		return true
	}
	
	// Validate token format
	if !m.isValidTokenFormat(token) {
		return false
	}
	
	// Check token validity (expensive operation)
	return m.validateToken(token)
}

// isValidTokenFormat checks if the token has a valid format
func (m *Middleware) isValidTokenFormat(token string) bool {
	// Simulate token format validation
	if len(token) < 10 {
		return false
	}
	
	// Check for required prefixes
	validPrefixes := []string{"Bearer ", "Token ", "API-Key "}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(token, prefix) {
			return true
		}
	}
	
	return false
}

// validateToken validates the authentication token
func (m *Middleware) validateToken(token string) bool {
	// Simulate expensive token validation
	for i := 0; i < len(token); i++ {
		// Simulate cryptographic operations
		_ = token[i] ^ 0x55
	}
	
	// Check against valid tokens (simplified)
	return len(token) > 15 // Simple validation rule
}

// authorizeRequest performs request authorization
func (m *Middleware) authorizeRequest(data *RequestData) bool {
	// Extract user information
	userID := data.Headers["User-ID"]
	if userID == "" {
		// Allow anonymous requests for demo
		return true
	}
	
	// Check user permissions
	return m.checkUserPermissions(userID, data)
}

// checkUserPermissions checks if user has required permissions
func (m *Middleware) checkUserPermissions(userID string, data *RequestData) bool {
	// Simulate permission checking
	requiredPermissions := m.extractRequiredPermissions(data)
	userPermissions := m.getUserPermissions(userID)
	
	return m.hasAllPermissions(userPermissions, requiredPermissions)
}

// extractRequiredPermissions extracts required permissions from request
func (m *Middleware) extractRequiredPermissions(data *RequestData) []string {
	// Simulate permission extraction based on request
	permissions := []string{"read"}
	
	if len(data.Payload) > 0 {
		permissions = append(permissions, "write")
	}
	
	if data.Headers["Admin"] == "true" {
		permissions = append(permissions, "admin")
	}
	
	return permissions
}

// getUserPermissions gets user permissions (expensive operation)
func (m *Middleware) getUserPermissions(userID string) []string {
	// Simulate database lookup
	defaultPermissions := []string{"read", "write"}
	
	// Simulate expensive permission calculation
	for i := 0; i < len(userID); i++ {
		if userID[i]%2 == 0 {
			defaultPermissions = append(defaultPermissions, "special")
			break
		}
	}
	
	return defaultPermissions
}

// hasAllPermissions checks if user has all required permissions
func (m *Middleware) hasAllPermissions(userPerms, requiredPerms []string) bool {
	for _, required := range requiredPerms {
		found := false
		for _, userPerm := range userPerms {
			if userPerm == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// transformRequest transforms the request data
func (m *Middleware) transformRequest(data *RequestData) *RequestData {
	// Create a copy for transformation
	transformed := &RequestData{
		ID:       data.ID,
		Payload:  make([]byte, len(data.Payload)),
		Headers:  make(map[string]string),
		Metadata: make(map[string]interface{}),
	}
	
	// Copy payload
	copy(transformed.Payload, data.Payload)
	
	// Transform headers
	for key, value := range data.Headers {
		// Normalize header names
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		normalizedValue := strings.TrimSpace(value)
		transformed.Headers[normalizedKey] = normalizedValue
	}
	
	// Transform metadata
	for key, value := range data.Metadata {
		transformed.Metadata[key] = m.transformMetadataValue(value)
	}
	
	// Add transformation metadata
	transformed.Metadata["transformed_at"] = time.Now()
	transformed.Metadata["transformation_version"] = "1.0"
	
	return transformed
}

// transformMetadataValue transforms individual metadata values
func (m *Middleware) transformMetadataValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Trim and normalize strings
		return strings.TrimSpace(v)
	case int:
		// Ensure positive integers
		if v < 0 {
			return 0
		}
		return v
	case float64:
		// Round floats to 2 decimal places
		return float64(int(v*100)) / 100
	default:
		return value
	}
}

// validateTransformedRequest validates the transformed request
func (m *Middleware) validateTransformedRequest(data *RequestData) *RequestData {
	// Perform post-transformation validation
	if data == nil {
		return nil
	}
	
	// Validate transformed headers
	for key, value := range data.Headers {
		if !m.isValidTransformedHeader(key, value) {
			return nil
		}
	}
	
	return data
}

// isValidTransformedHeader validates transformed headers
func (m *Middleware) isValidTransformedHeader(key, value string) bool {
	// Check for required transformations
	if strings.ToLower(key) != key {
		return false // Should be lowercase
	}
	
	if strings.TrimSpace(value) != value {
		return false // Should be trimmed
	}
	
	return true
}

// cacheRequest caches the processed request
func (m *Middleware) cacheRequest(data *RequestData) {
	if data == nil {
		return
	}
	
	cacheKey := m.generateCacheKey(data)
	entry := &CacheEntry{
		Data:      data,
		CreatedAt: time.Now(),
		TTL:       time.Minute * 5, // 5 minute TTL
	}
	
	m.cache.cache[cacheKey] = entry
	
	// Clean up old cache entries periodically
	if len(m.cache.cache) > 1000 {
		m.cleanupCache()
	}
}

// cleanupCache removes expired cache entries
func (m *Middleware) cleanupCache() {
	now := time.Now()
	for key, entry := range m.cache.cache {
		if now.Sub(entry.CreatedAt) > entry.TTL {
			delete(m.cache.cache, key)
		}
	}
}

// collectMetrics collects processing metrics
func (m *Middleware) collectMetrics(data *RequestData) {
	if data == nil {
		return
	}
	
	// Record processing stage metrics
	stage := "middleware"
	processingTime := time.Microsecond * 100 // Simulated processing time
	
	if _, exists := m.metrics.processingTimes[stage]; !exists {
		m.metrics.processingTimes[stage] = make([]time.Duration, 0)
	}
	
	m.metrics.processingTimes[stage] = append(m.metrics.processingTimes[stage], processingTime)
	
	// Limit metrics history
	if len(m.metrics.processingTimes[stage]) > 1000 {
		m.metrics.processingTimes[stage] = m.metrics.processingTimes[stage][100:]
	}
}