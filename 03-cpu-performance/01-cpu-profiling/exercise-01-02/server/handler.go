package server

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Server represents the application server
type Server struct {
	middleware *Middleware
	stats      *ServerStats
}

// ServerStats tracks server performance metrics
type ServerStats struct {
	RequestsProcessed int64
	TotalProcessingTime time.Duration
	ErrorCount int64
}

// RequestData represents incoming request data
type RequestData struct {
	ID       int
	Payload  []byte
	Headers  map[string]string
	Metadata map[string]interface{}
}

// ProcessedRequest represents a request after server processing
type ProcessedRequest struct {
	Original    *RequestData
	ProcessedAt time.Time
	Validated   bool
	Enriched    map[string]interface{}
	Context     map[string]string
}

// New creates a new server instance
func New() *Server {
	return &Server{
		middleware: NewMiddleware(),
		stats:      &ServerStats{},
	}
}

// HandleRequest processes an incoming request through the server pipeline
func (s *Server) HandleRequest(data *RequestData) *ProcessedRequest {
	start := time.Now()
	defer func() {
		s.stats.RequestsProcessed++
		s.stats.TotalProcessingTime += time.Since(start)
	}()

	// Apply middleware chain
	processedData := s.middleware.ProcessRequest(data)

	// Validate request
	validated := s.validateRequest(processedData)

	// Handle validation failure
	if validated == nil {
		return &ProcessedRequest{
			Original:    data,
			ProcessedAt: time.Now(),
			Validated:   false,
			Enriched:    make(map[string]interface{}),
			Context:     make(map[string]string),
		}
	}

	// Enrich request with additional data
	enriched := s.enrichRequest(validated)

	// Add server context
	contextData := s.addServerContext(enriched)

	return &ProcessedRequest{
		Original:    data,
		ProcessedAt: time.Now(),
		Validated:   true,
		Enriched:    enriched,
		Context:     contextData,
	}
}

// validateRequest performs comprehensive request validation
func (s *Server) validateRequest(data *RequestData) *RequestData {
	// Simulate complex validation logic
	if data == nil {
		s.stats.ErrorCount++
		return nil
	}

	// Validate headers
	for key, value := range data.Headers {
		if !s.isValidHeader(key, value) {
			s.stats.ErrorCount++
			return nil
		}
	}

	// Validate payload
	if !s.isValidPayload(data.Payload) {
		s.stats.ErrorCount++
		return nil
	}

	// Validate metadata
	for key, value := range data.Metadata {
		if !s.isValidMetadata(key, value) {
			s.stats.ErrorCount++
			return nil
		}
	}

	return data
}

// isValidHeader validates individual headers
func (s *Server) isValidHeader(key, value string) bool {
	// Simulate header validation logic
	if len(key) == 0 || len(value) == 0 {
		return false
	}

	// Check for forbidden characters
	forbidden := []string{"<", ">", "&", "\""}
	for _, char := range forbidden {
		if strings.Contains(key, char) || strings.Contains(value, char) {
			return false
		}
	}

	// Simulate expensive validation
	for i := 0; i < len(value); i++ {
		if value[i] < 32 || value[i] > 126 {
			return false
		}
	}

	return true
}

// isValidPayload validates request payload
func (s *Server) isValidPayload(payload []byte) bool {
	if len(payload) == 0 {
		return true // Empty payload is valid
	}

	// Simulate payload validation
	if len(payload) > 1024*1024 { // 1MB limit
		return false
	}

	// Check for valid UTF-8 (expensive operation)
	payloadStr := string(payload)
	for _, r := range payloadStr {
		if r == 0xFFFD { // Unicode replacement character
			return false
		}
	}

	return true
}

// isValidMetadata validates metadata fields
func (s *Server) isValidMetadata(key string, value interface{}) bool {
	// Simulate metadata validation
	if len(key) == 0 {
		return false
	}

	// Type-specific validation
	switch v := value.(type) {
	case string:
		return len(v) <= 1000
	case int:
		return v >= 0
	case float64:
		return v >= 0.0
	case bool:
		return true
	default:
		return false
	}
}

// enrichRequest adds additional data to the request
func (s *Server) enrichRequest(data *RequestData) map[string]interface{} {
	enriched := make(map[string]interface{})

	// Add timestamp information
	enriched["server_timestamp"] = time.Now().Unix()
	enriched["processing_start"] = time.Now()

	// Add request analysis
	enriched["payload_size"] = len(data.Payload)
	enriched["header_count"] = len(data.Headers)
	enriched["metadata_count"] = len(data.Metadata)

	// Perform content analysis
	contentAnalysis := s.analyzeContent(data.Payload)
	enriched["content_analysis"] = contentAnalysis

	// Add security information
	securityInfo := s.performSecurityCheck(data)
	enriched["security_info"] = securityInfo

	// Add performance metrics
	perfMetrics := s.calculatePerformanceMetrics(data)
	enriched["performance_metrics"] = perfMetrics

	return enriched
}

// analyzeContent performs content analysis on the payload
func (s *Server) analyzeContent(payload []byte) map[string]interface{} {
	analysis := make(map[string]interface{})

	if len(payload) == 0 {
		analysis["type"] = "empty"
		return analysis
	}

	// Character frequency analysis
	charFreq := make(map[rune]int)
	payloadStr := string(payload)
	for _, r := range payloadStr {
		charFreq[r]++
	}
	analysis["char_frequency"] = charFreq

	// Content type detection (simplified)
	if strings.Contains(payloadStr, "{") && strings.Contains(payloadStr, "}") {
		analysis["likely_type"] = "json"
	} else if strings.Contains(payloadStr, "<") && strings.Contains(payloadStr, ">") {
		analysis["likely_type"] = "xml"
	} else {
		analysis["likely_type"] = "text"
	}

	// Statistical analysis
	analysis["length"] = len(payload)
	analysis["unique_chars"] = len(charFreq)
	analysis["entropy"] = s.calculateEntropy(charFreq, len(payloadStr))

	return analysis
}

// calculateEntropy calculates the entropy of the content
func (s *Server) calculateEntropy(charFreq map[rune]int, totalChars int) float64 {
	if totalChars == 0 {
		return 0.0
	}

	entropy := 0.0
	for _, freq := range charFreq {
		if freq > 0 {
			p := float64(freq) / float64(totalChars)
			entropy -= p * s.logBase2(p)
		}
	}

	return entropy
}

// logBase2 calculates log base 2
func (s *Server) logBase2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log(x) / math.Log(2)
}

// performSecurityCheck performs security analysis
func (s *Server) performSecurityCheck(data *RequestData) map[string]interface{} {
	securityInfo := make(map[string]interface{})

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"<script", "javascript:", "eval(", "exec(",
		"DROP TABLE", "SELECT *", "UNION SELECT",
		"../", "..\\\\", "/etc/passwd",
	}

	payloadStr := string(data.Payload)
	suspiciousCount := 0
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(payloadStr), strings.ToLower(pattern)) {
			suspiciousCount++
		}
	}

	securityInfo["suspicious_patterns"] = suspiciousCount
	securityInfo["risk_level"] = s.calculateRiskLevel(suspiciousCount)

	// Rate limiting simulation
	securityInfo["rate_limit_check"] = s.checkRateLimit(data.ID)

	return securityInfo
}

// calculateRiskLevel determines risk level based on suspicious patterns
func (s *Server) calculateRiskLevel(suspiciousCount int) string {
	switch {
	case suspiciousCount == 0:
		return "low"
	case suspiciousCount <= 2:
		return "medium"
	default:
		return "high"
	}
}

// checkRateLimit simulates rate limiting check
func (s *Server) checkRateLimit(requestID int) bool {
	// Simulate rate limiting logic
	return requestID%100 != 0 // Allow 99% of requests
}

// calculatePerformanceMetrics calculates performance-related metrics
func (s *Server) calculatePerformanceMetrics(data *RequestData) map[string]interface{} {
	metrics := make(map[string]interface{})

	// Estimate processing complexity
	complexity := len(data.Payload) + len(data.Headers)*10 + len(data.Metadata)*5
	metrics["estimated_complexity"] = complexity

	// Memory usage estimation
	memoryUsage := len(data.Payload) + len(data.Headers)*50 + len(data.Metadata)*100
	metrics["estimated_memory"] = memoryUsage

	// Processing time prediction
	predictedTime := float64(complexity) * 0.001 // milliseconds
	metrics["predicted_processing_time"] = predictedTime

	return metrics
}

// addServerContext adds server-specific context information
func (s *Server) addServerContext(enriched map[string]interface{}) map[string]string {
	context := make(map[string]string)

	// Add server information
	context["server_id"] = "server-001"
	context["server_version"] = "1.0.0"
	context["processing_node"] = fmt.Sprintf("node-%d", time.Now().Unix()%10)

	// Add processing information
	context["processing_stage"] = "server_handler"
	context["next_stage"] = "business_logic"

	// Add correlation ID
	context["correlation_id"] = fmt.Sprintf("corr-%d-%d", time.Now().Unix(), time.Now().Nanosecond())

	return context
}

// GetStats returns current server statistics
func (s *Server) GetStats() *ServerStats {
	return s.stats
}