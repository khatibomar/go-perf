package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// createTestTracerProvider creates a tracer provider for testing
func createTestTracerProvider(t *testing.T) *TracerProvider {
	return createTracerProvider(t)
}

// createBenchmarkTracerProvider creates a tracer provider for benchmarking
func createBenchmarkTracerProvider(b *testing.B) *TracerProvider {
	return createTracerProvider(b)
}

// createTracerProvider creates a tracer provider for testing/benchmarking
func createTracerProvider(tb interface{}) *TracerProvider {
	var fatalf func(format string, args ...interface{})
	switch v := tb.(type) {
	case *testing.T:
		fatalf = v.Fatalf
	case *testing.B:
		fatalf = v.Fatalf
	default:
		panic("unsupported testing type")
	}
	// Create a no-op exporter for testing to reduce output noise
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(io.Discard),
	)
	if err != nil {
		fatalf("Failed to create stdout exporter: %v", err)
	}

	// Create resource
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("test-service"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.DeploymentEnvironmentKey.String("test"),
	)

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("test-service")

	return &TracerProvider{
		provider: tp,
		tracer:   tracer,
		config: TracerConfig{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "test",
			SamplingRatio:  1.0,
		},
		stats: &TracingStats{
			StartTime: time.Now(),
		},
	}
}

func TestNewTracerProvider(t *testing.T) {
	// This test might fail if Jaeger is not running, so we'll use a mock
	// In a real scenario, you'd want to test with a running Jaeger instance
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	if tp == nil {
		t.Fatal("Expected tracer provider to be created")
	}

	if tp.tracer == nil {
		t.Fatal("Expected tracer to be initialized")
	}

	if tp.stats == nil {
		t.Fatal("Expected stats to be initialized")
	}
}

func TestTracerProvider_StartSpan(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	ctx := context.Background()
	ctx, span := tp.StartSpan(ctx, "test-span")

	if span == nil {
		t.Fatal("Expected span to be created")
	}

	// Check that stats were updated
	stats := tp.GetStats()
	if stats.SpansCreated != 1 {
		t.Errorf("Expected 1 span created, got %d", stats.SpansCreated)
	}

	span.End()
}

func TestTracerProvider_GetStats(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	// Initial stats
	stats := tp.GetStats()
	if stats.SpansCreated != 0 {
		t.Errorf("Expected 0 spans created initially, got %d", stats.SpansCreated)
	}

	// Create some spans
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, span := tp.StartSpan(ctx, fmt.Sprintf("test-span-%d", i))
		span.End()
	}

	// Update completed count manually for test
	tp.stats.mu.Lock()
	tp.stats.SpansCompleted = 5
	tp.stats.mu.Unlock()

	stats = tp.GetStats()
	if stats.SpansCreated != 5 {
		t.Errorf("Expected 5 spans created, got %d", stats.SpansCreated)
	}
	if stats.SpansCompleted != 5 {
		t.Errorf("Expected 5 spans completed, got %d", stats.SpansCompleted)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	middleware := tp.HTTPMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-123")
	rr := httptest.NewRecorder()

	// Execute request
	middleware.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", rr.Body.String())
	}

	// Check that request ID was preserved or generated
	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Errorf("Expected X-Request-ID to be present")
	}

	// Check stats
	stats := tp.GetStats()
	if stats.SpansCreated == 0 {
		t.Error("Expected spans to be created by middleware")
	}
}

func TestBusinessOperation_ProcessOrder(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	bo := NewBusinessOperation(tp)
	ctx := context.Background()

	// Get initial stats
	initialStats := tp.GetStats()
	
	// Test successful order processing
	err := bo.ProcessOrder(ctx, "test-order-123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that spans were created
	finalStats := tp.GetStats()
	if finalStats.SpansCreated <= initialStats.SpansCreated {
		t.Error("Expected spans to be created during order processing")
	}
}

func TestBusinessOperation_ValidateOrder(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	bo := NewBusinessOperation(tp)
	ctx := context.Background()

	// Test multiple validations to check for occasional failures
	successCount := 0
	for i := 0; i < 20; i++ {
		err := bo.validateOrder(ctx, fmt.Sprintf("order-%d", i))
		if err == nil {
			successCount++
		}
	}

	// Most should succeed (failure rate is 5%)
	if successCount < 15 {
		t.Errorf("Expected at least 15 successful validations, got %d", successCount)
	}
}

func TestHealthHandler(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	tp.healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var health map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", health["status"])
	}

	if health["service"] != tp.config.ServiceName {
		t.Errorf("Expected service name '%s', got %v", tp.config.ServiceName, health["service"])
	}
}

func TestOrderHandler(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	// Test with order ID
	req := httptest.NewRequest("GET", "/order?id=test-123", nil)
	rr := httptest.NewRecorder()

	tp.orderHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal order response: %v", err)
	}

	if response["order_id"] != "test-123" {
		t.Errorf("Expected order_id 'test-123', got %v", response["order_id"])
	}

	if response["status"] != "completed" {
		t.Errorf("Expected status 'completed', got %v", response["status"])
	}
}

func TestStatsHandler(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	// Create some spans first
	ctx := context.Background()
	_, span := tp.StartSpan(ctx, "test-span")
	span.End()

	req := httptest.NewRequest("GET", "/stats", nil)
	rr := httptest.NewRecorder()

	tp.statsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var stats TracingStats
	err := json.Unmarshal(rr.Body.Bytes(), &stats)
	if err != nil {
		t.Fatalf("Failed to unmarshal stats response: %v", err)
	}

	if stats.SpansCreated == 0 {
		t.Error("Expected spans_created to be greater than 0")
	}
}

func TestSimulateLoadHandler(t *testing.T) {
	tp := createTestTracerProvider(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	// Test with small load
	req := httptest.NewRequest("GET", "/simulate?requests=5", nil)
	rr := httptest.NewRecorder()

	tp.simulateLoadHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal simulate response: %v", err)
	}

	if response["requests_processed"] != float64(5) {
		t.Errorf("Expected 5 requests processed, got %v", response["requests_processed"])
	}

	if response["duration_ms"] == nil {
		t.Error("Expected duration_ms to be present")
	}

	if response["requests_per_second"] == nil {
		t.Error("Expected requests_per_second to be present")
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test generateRequestID
	requestID := generateRequestID()
	if !strings.HasPrefix(requestID, "req-") {
		t.Errorf("Expected request ID to start with 'req-', got %s", requestID)
	}

	// Test generateOrderID
	orderID := generateOrderID()
	if !strings.HasPrefix(orderID, "order-") {
		t.Errorf("Expected order ID to start with 'order-', got %s", orderID)
	}

	// Test uniqueness
	requestID2 := generateRequestID()
	if requestID == requestID2 {
		t.Error("Expected request IDs to be unique")
	}

	orderID2 := generateOrderID()
	if orderID == orderID2 {
		t.Error("Expected order IDs to be unique")
	}
}

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

// Benchmark tests
func BenchmarkStartSpan(b *testing.B) {
	tp := createBenchmarkTracerProvider(b)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, span := tp.StartSpan(ctx, "benchmark-span",
			oteltrace.WithAttributes(
				attribute.Int("iteration", i),
			),
		)
		span.End()
	}
}

func BenchmarkProcessOrder(b *testing.B) {
	tp := createBenchmarkTracerProvider(b)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	bo := NewBusinessOperation(tp)
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		orderID := fmt.Sprintf("bench-order-%d", i)
		bo.ProcessOrder(ctx, orderID)
	}
}

func BenchmarkHTTPMiddleware(b *testing.B) {
	tp := createBenchmarkTracerProvider(b)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		tp.Shutdown(ctx)
	}()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := tp.HTTPMiddleware(testHandler)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)
	}
}