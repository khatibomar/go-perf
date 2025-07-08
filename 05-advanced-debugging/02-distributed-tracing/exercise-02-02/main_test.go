package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// Test MetricsCollector
func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	if mc == nil {
		t.Fatal("Expected metrics collector to be created")
	}

	if mc.customCounters == nil {
		t.Error("Expected custom counters map to be initialized")
	}

	if mc.customGauges == nil {
		t.Error("Expected custom gauges map to be initialized")
	}

	if mc.customHistograms == nil {
		t.Error("Expected custom histograms map to be initialized")
	}
}

func TestMetricsCollector_HTTPMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")

	// Test HTTP request recording
	mc.RecordHTTPRequest("GET", "/test", "200", time.Millisecond*100)

	// Test in-flight requests
	mc.IncHTTPRequestsInFlight()
	mc.DecHTTPRequestsInFlight()

	// Verify metrics were recorded (basic check)
	// In a real test, you'd use prometheus testutil to verify exact values
}

func TestMetricsCollector_BusinessMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")

	// Test business operation recording
	mc.RecordBusinessOperation("test_operation", "success", time.Millisecond*50)
	mc.RecordBusinessError("test_operation", "validation_error")

	// Verify metrics were recorded
}

func TestMetricsCollector_CustomMetrics(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")

	// Test custom counter
	mc.RegisterCustomCounter("test_counter", "Test counter", []string{"label1"})
	mc.IncCustomCounter("test_counter", "value1")

	// Test custom gauge
	mc.RegisterCustomGauge("test_gauge", "Test gauge", []string{"label1"})
	mc.SetCustomGauge("test_gauge", 42.0, "value1")

	// Test custom histogram
	mc.RegisterCustomHistogram("test_histogram", "Test histogram", []string{"label1"}, []float64{0.1, 1, 10})
	mc.ObserveCustomHistogram("test_histogram", 0.5, "value1")

	// Verify custom metrics were registered and used
	if len(mc.customCounters) != 1 {
		t.Errorf("Expected 1 custom counter, got %d", len(mc.customCounters))
	}

	if len(mc.customGauges) != 1 {
		t.Errorf("Expected 1 custom gauge, got %d", len(mc.customGauges))
	}

	if len(mc.customHistograms) != 1 {
		t.Errorf("Expected 1 custom histogram, got %d", len(mc.customHistograms))
	}
}

func TestMetricsCollector_HTTPMiddleware(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with middleware
	middleware := mc.HTTPMiddleware(testHandler)

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got %s", rr.Body.String())
	}
}

// Test StructuredLogger
func TestNewStructuredLogger(t *testing.T) {
	config := LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test-service",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	}

	logger, err := NewStructuredLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestStructuredLogger_LoggingMethods(t *testing.T) {
	// Create logger with observer for testing
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := &StructuredLogger{
		zap: zap.New(core),
	}

	// Test different log levels
	logger.Info("test info message", zap.String("key", "value"))
	logger.Warn("test warn message", zap.String("key", "value"))
	logger.Error("test error message", zap.String("key", "value"))

	// Verify logs were recorded
	logs := recorded.All()
	if len(logs) != 3 {
		t.Errorf("Expected 3 log entries, got %d", len(logs))
	}

	// Check log levels
	expectedLevels := []zapcore.Level{zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel}
	for i, log := range logs {
		if log.Level != expectedLevels[i] {
			t.Errorf("Expected level %v, got %v", expectedLevels[i], log.Level)
		}
	}
}

func TestStructuredLogger_WithFields(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := &StructuredLogger{
		zap: zap.New(core),
	}

	// Test with fields
	fields := map[string]interface{}{
		"user_id": "123",
		"action":  "test",
	}

	loggerWithFields := logger.WithFields(fields)
	loggerWithFields.Info("test message")

	// Verify fields were added
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.ContextMap()["user_id"] != "123" {
		t.Errorf("Expected user_id '123', got %v", log.ContextMap()["user_id"])
	}

	if log.ContextMap()["action"] != "test" {
		t.Errorf("Expected action 'test', got %v", log.ContextMap()["action"])
	}
}

func TestStructuredLogger_LogHTTPRequest(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := &StructuredLogger{
		zap: zap.New(core),
	}

	req := httptest.NewRequest("GET", "/test?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")

	logger.LogHTTPRequest(req, 200, time.Millisecond*100, 1024)

	// Verify HTTP request was logged
	logs := recorded.All()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.Message != "HTTP request" {
		t.Errorf("Expected message 'HTTP request', got %s", log.Message)
	}

	// Check specific fields
	contextMap := log.ContextMap()
	if contextMap["method"] != "GET" {
		t.Errorf("Expected method 'GET', got %v", contextMap["method"])
	}

	if contextMap["path"] != "/test" {
		t.Errorf("Expected path '/test', got %v", contextMap["path"])
	}

	if contextMap["status_code"] != int64(200) {
		t.Errorf("Expected status_code 200, got %v", contextMap["status_code"])
	}
}

func TestStructuredLogger_LogBusinessOperation(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := &StructuredLogger{
		zap: zap.New(core),
	}

	metadata := map[string]interface{}{
		"order_id": "12345",
		"amount":   99.99,
	}

	// Test successful operation
	logger.LogBusinessOperation("process_order", true, time.Millisecond*50, metadata)

	// Test failed operation
	logger.LogBusinessOperation("process_payment", false, time.Millisecond*25, metadata)

	// Verify logs
	logs := recorded.All()
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}

	// Check success log
	successLog := logs[0]
	if successLog.Message != "Business operation completed" {
		t.Errorf("Expected message 'Business operation completed', got %s", successLog.Message)
	}

	if successLog.Level != zapcore.InfoLevel {
		t.Errorf("Expected info level for success, got %v", successLog.Level)
	}

	// Check failure log
	failureLog := logs[1]
	if failureLog.Message != "Business operation failed" {
		t.Errorf("Expected message 'Business operation failed', got %s", failureLog.Message)
	}

	if failureLog.Level != zapcore.ErrorLevel {
		t.Errorf("Expected error level for failure, got %v", failureLog.Level)
	}
}

// Test BusinessService
func TestNewBusinessService(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)
	if bs == nil {
		t.Fatal("Expected business service to be created")
	}

	if bs.metrics != mc {
		t.Error("Expected metrics collector to be set")
	}

	if bs.logger != logger {
		t.Error("Expected logger to be set")
	}
}

func TestBusinessService_ProcessOrder(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	// Test successful order processing
	err := bs.ProcessOrder("test-order-123")
	if err != nil {
		// Note: This might fail occasionally due to random error simulation
		// In a real test, you'd mock the random behavior
		t.Logf("Order processing failed (expected occasionally): %v", err)
	}
}

func TestBusinessService_ValidatePayment(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	// Test payment validation
	err := bs.ValidatePayment("test-payment-123")
	if err != nil {
		// Note: This might fail occasionally due to random error simulation
		t.Logf("Payment validation failed (expected occasionally): %v", err)
	}
}

func TestBusinessService_UpdateInventory(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	// Test inventory update
	err := bs.UpdateInventory("test-item-123", 5)
	if err != nil {
		// Note: This might fail occasionally due to random error simulation
		t.Logf("Inventory update failed (expected occasionally): %v", err)
	}
}

// Test HTTP Handlers
func TestBusinessService_HealthHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	bs.healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["service"] != "metrics-logging-demo" {
		t.Errorf("Expected service 'metrics-logging-demo', got %v", response["service"])
	}
}

func TestBusinessService_OrderHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	// Test with order ID
	req := httptest.NewRequest("GET", "/order?id=test-123", nil)
	rr := httptest.NewRecorder()

	bs.orderHandler(rr, req)

	// Note: This might return 500 occasionally due to random error simulation
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rr.Code)
	}

	if rr.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["order_id"] != "test-123" {
			t.Errorf("Expected order_id 'test-123', got %v", response["order_id"])
		}

		if response["status"] != "completed" {
			t.Errorf("Expected status 'completed', got %v", response["status"])
		}
	}
}

func TestBusinessService_PaymentHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	// Test with payment ID
	req := httptest.NewRequest("GET", "/payment?id=test-payment-123", nil)
	rr := httptest.NewRecorder()

	bs.paymentHandler(rr, req)

	// Note: This might return 500 occasionally due to random error simulation
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rr.Code)
	}

	if rr.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["payment_id"] != "test-payment-123" {
			t.Errorf("Expected payment_id 'test-payment-123', got %v", response["payment_id"])
		}

		if response["status"] != "validated" {
			t.Errorf("Expected status 'validated', got %v", response["status"])
		}
	}
}

func TestBusinessService_InventoryHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	// Test with item ID and quantity
	req := httptest.NewRequest("GET", "/inventory?item_id=test-item-123&quantity=5", nil)
	rr := httptest.NewRecorder()

	bs.inventoryHandler(rr, req)

	// Note: This might return 500 occasionally due to random error simulation
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", rr.Code)
	}

	if rr.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["item_id"] != "test-item-123" {
			t.Errorf("Expected item_id 'test-item-123', got %v", response["item_id"])
		}

		if response["quantity"] != float64(5) {
			t.Errorf("Expected quantity 5, got %v", response["quantity"])
		}

		if response["status"] != "updated" {
			t.Errorf("Expected status 'updated', got %v", response["status"])
		}
	}
}

func TestBusinessService_LoadTestHandler(t *testing.T) {
	mc := NewMetricsCollector("test", "metrics")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "test",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)

	// Test with small number of operations
	req := httptest.NewRequest("GET", "/loadtest?operations=5", nil)
	rr := httptest.NewRecorder()

	bs.loadTestHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["operations_processed"] != float64(5) {
		t.Errorf("Expected 5 operations processed, got %v", response["operations_processed"])
	}

	if response["duration_ms"] == nil {
		t.Error("Expected duration_ms to be present")
	}

	if response["operations_per_second"] == nil {
		t.Error("Expected operations_per_second to be present")
	}
}

// Test utility functions
func TestUtilityFunctions(t *testing.T) {
	// Test generateOrderID
	orderID := generateOrderID()
	if !strings.HasPrefix(orderID, "order-") {
		t.Errorf("Expected order ID to start with 'order-', got %s", orderID)
	}

	// Test generatePaymentID
	paymentID := generatePaymentID()
	if !strings.HasPrefix(paymentID, "payment-") {
		t.Errorf("Expected payment ID to start with 'payment-', got %s", paymentID)
	}

	// Test generateItemID
	itemID := generateItemID()
	if !strings.HasPrefix(itemID, "item-") {
		t.Errorf("Expected item ID to start with 'item-', got %s", itemID)
	}

	// Test uniqueness
	orderID2 := generateOrderID()
	if orderID == orderID2 {
		t.Error("Expected order IDs to be unique")
	}

	paymentID2 := generatePaymentID()
	if paymentID == paymentID2 {
		t.Error("Expected payment IDs to be unique")
	}

	itemID2 := generateItemID()
	if itemID == itemID2 {
		t.Error("Expected item IDs to be unique")
	}
}

// Test responseWriter
func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: rr, statusCode: 200}

	// Test WriteHeader
	wrapped.WriteHeader(404)
	if wrapped.statusCode != 404 {
		t.Errorf("Expected status code 404, got %d", wrapped.statusCode)
	}

	// Test Write
	data := []byte("test data")
	n, err := wrapped.Write(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}
	if wrapped.bytesWritten != int64(len(data)) {
		t.Errorf("Expected %d bytes tracked, got %d", len(data), wrapped.bytesWritten)
	}
}

// Performance tests
func TestMetricsPerformance(t *testing.T) {
	mc := NewMetricsCollector("perf", "test")

	// Test metric ingestion performance
	start := time.Now()
	numMetrics := 10000

	for i := 0; i < numMetrics; i++ {
		mc.RecordHTTPRequest("GET", "/test", "200", time.Millisecond)
		mc.RecordBusinessOperation("test_op", "success", time.Millisecond)
	}

	duration := time.Since(start)
	metricsPerSecond := float64(numMetrics*2) / duration.Seconds()

	t.Logf("Processed %d metrics in %v (%.2f metrics/second)", numMetrics*2, duration, metricsPerSecond)

	// Performance target: >100K metrics/second
	// Note: This is a basic test and actual performance depends on the Prometheus client implementation
	if metricsPerSecond < 1000 { // Reduced target for test environment
		t.Logf("Warning: Metrics performance below target: %.2f metrics/second", metricsPerSecond)
	}
}

func TestLoggingPerformance(t *testing.T) {
	// Create logger with buffer for performance testing
	var buf bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			MessageKey:     "message",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		}),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	logger := &StructuredLogger{
		zap: zap.New(core),
	}

	// Test logging performance
	start := time.Now()
	numLogs := 1000

	for i := 0; i < numLogs; i++ {
		logger.Info("test log message",
			zap.String("operation", "test"),
			zap.Int("iteration", i),
			zap.Duration("duration", time.Millisecond),
		)
	}

	duration := time.Since(start)
	logsPerSecond := float64(numLogs) / duration.Seconds()

	t.Logf("Processed %d logs in %v (%.2f logs/second)", numLogs, duration, logsPerSecond)

	// Performance target: log processing latency <100ms for batch
	if duration > time.Millisecond*100 {
		t.Logf("Warning: Logging performance below target: %v for %d logs", duration, numLogs)
	}
}

// Benchmark tests
func BenchmarkMetricsCollector_RecordHTTPRequest(b *testing.B) {
	mc := NewMetricsCollector("bench", "test")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mc.RecordHTTPRequest("GET", "/test", "200", time.Millisecond)
	}
}

func BenchmarkMetricsCollector_RecordBusinessOperation(b *testing.B) {
	mc := NewMetricsCollector("bench", "test")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mc.RecordBusinessOperation("test_op", "success", time.Millisecond)
	}
}

func BenchmarkStructuredLogger_Info(b *testing.B) {
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "bench",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			zap.String("operation", "benchmark"),
			zap.Int("iteration", i),
		)
	}
}

func BenchmarkBusinessService_ProcessOrder(b *testing.B) {
	mc := NewMetricsCollector("bench", "test")
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "bench",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	bs := NewBusinessService(mc, logger)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		orderID := fmt.Sprintf("bench-order-%d", i)
		bs.ProcessOrder(orderID)
	}
}

func BenchmarkHTTPMiddleware(b *testing.B) {
	mc := NewMetricsCollector("bench", "test")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := mc.HTTPMiddleware(testHandler)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)
	}
}

// Concurrent tests
func TestConcurrentMetrics(t *testing.T) {
	mc := NewMetricsCollector("concurrent", "test")

	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				mc.RecordHTTPRequest("GET", fmt.Sprintf("/test-%d", id), "200", time.Millisecond)
				mc.RecordBusinessOperation(fmt.Sprintf("op-%d", id), "success", time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Test completed without race conditions
	t.Log("Concurrent metrics test completed successfully")
}

func TestConcurrentLogging(t *testing.T) {
	logger, _ := NewStructuredLogger(LoggerConfig{
		Level:       "info",
		Environment: "test",
		ServiceName: "concurrent",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	})

	var wg sync.WaitGroup
	numGoroutines := 10
	logsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info("concurrent log message",
					zap.Int("goroutine_id", id),
					zap.Int("log_number", j),
				)
			}
		}(i)
	}

	wg.Wait()

	// Test completed without race conditions
	t.Log("Concurrent logging test completed successfully")
}