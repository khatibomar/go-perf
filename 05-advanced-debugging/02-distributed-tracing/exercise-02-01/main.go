package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TracerConfig holds configuration for the tracer
type TracerConfig struct {
	ServiceName     string
	ServiceVersion  string
	Environment     string
	JaegerEndpoint  string
	SamplingRatio   float64
	BatchTimeout    time.Duration
	MaxBatchSize    int
}

// TracerProvider manages OpenTelemetry tracing
type TracerProvider struct {
	provider *trace.TracerProvider
	tracer   oteltrace.Tracer
	config   TracerConfig
	mu       sync.RWMutex
	stats    *TracingStats
}

// TracingStats holds tracing statistics
type TracingStats struct {
	SpansCreated    int64     `json:"spans_created"`
	SpansCompleted  int64     `json:"spans_completed"`
	SpansErrored    int64     `json:"spans_errored"`
	TracesSampled   int64     `json:"traces_sampled"`
	TracesDropped   int64     `json:"traces_dropped"`
	AvgLatency      float64   `json:"avg_latency_ms"`
	StartTime       time.Time `json:"start_time"`
	mu              sync.RWMutex
}

// NewTracerProvider creates a new tracer provider
func NewTracerProvider(config TracerConfig) (*TracerProvider, error) {
	// Create Jaeger exporter
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(config.JaegerEndpoint),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(config.ServiceName),
		semconv.ServiceVersionKey.String(config.ServiceVersion),
		semconv.DeploymentEnvironmentKey.String(config.Environment),
	)

	// Create trace provider with custom sampler
	tp := trace.NewTracerProvider(
		trace.WithBatcher(
			exporter,
			trace.WithBatchTimeout(config.BatchTimeout),
			trace.WithMaxExportBatchSize(config.MaxBatchSize),
		),
		trace.WithResource(resource),
		trace.WithSampler(trace.TraceIDRatioBased(config.SamplingRatio)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tracer := tp.Tracer(config.ServiceName)

	return &TracerProvider{
		provider: tp,
		tracer:   tracer,
		config:   config,
		stats: &TracingStats{
			StartTime: time.Now(),
		},
	}, nil
}

// StartSpan starts a new span
func (tp *TracerProvider) StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	ctx, span := tp.tracer.Start(ctx, name, opts...)
	
	// Update statistics
	tp.stats.mu.Lock()
	tp.stats.SpansCreated++
	tp.stats.mu.Unlock()
	
	return ctx, span
}

// Shutdown shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	return tp.provider.Shutdown(ctx)
}

// GetStats returns current tracing statistics
func (tp *TracerProvider) GetStats() TracingStats {
	tp.stats.mu.RLock()
	defer tp.stats.mu.RUnlock()
	
	stats := *tp.stats
	
	// Calculate average latency
	if stats.SpansCompleted > 0 {
		elapsed := time.Since(stats.StartTime)
		stats.AvgLatency = elapsed.Seconds() * 1000 / float64(stats.SpansCompleted)
	}
	
	return stats
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// HTTPMiddleware provides HTTP tracing middleware
func (tp *TracerProvider) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract context from headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		
		// Create span name
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		
		// Start span with HTTP attributes
		ctx, span := tp.StartSpan(ctx, spanName,
			oteltrace.WithAttributes(
				semconv.HTTPMethodKey.String(r.Method),
				semconv.HTTPURLKey.String(r.URL.String()),
				attribute.String("http.user_agent", r.UserAgent()),
				semconv.HTTPSchemeKey.String(r.URL.Scheme),
				attribute.String("http.host", r.Host),
				semconv.HTTPTargetKey.String(r.URL.Path),
			),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		)
		defer func() {
			span.End()
			tp.stats.mu.Lock()
			tp.stats.SpansCompleted++
			tp.stats.mu.Unlock()
		}()
		
		// Wrap response writer
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		// Add request ID to context
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		wrapped.Header().Set("X-Request-ID", requestID)
		ctx = context.WithValue(ctx, "request_id", requestID)
		
		// Process request
		next.ServeHTTP(wrapped, r.WithContext(ctx))
		
		// Add response attributes
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(wrapped.statusCode),
		)
		span.SetAttributes(
			attribute.String("http.request_id", requestID),
			attribute.Int64("http.response_size", wrapped.bytesWritten),
		)
		
		// Set span status based on response code
		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapped.statusCode))
			tp.stats.mu.Lock()
			tp.stats.SpansErrored++
			tp.stats.mu.Unlock()
		} else {
			span.SetStatus(codes.Ok, "")
		}
	})
}

// BusinessOperation represents a business operation with tracing
type BusinessOperation struct {
	tp *TracerProvider
}

// NewBusinessOperation creates a new business operation
func NewBusinessOperation(tp *TracerProvider) *BusinessOperation {
	return &BusinessOperation{
		tp: tp,
	}
}

// ProcessOrder simulates order processing with tracing
func (bo *BusinessOperation) ProcessOrder(ctx context.Context, orderID string) error {
	ctx, span := bo.tp.StartSpan(ctx, "process_order",
		oteltrace.WithAttributes(
			attribute.String("order.id", orderID),
			attribute.String("operation.type", "business"),
		),
	)
	defer span.End()
	
	// Simulate validation
	if err := bo.validateOrder(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	
	// Simulate payment processing
	if err := bo.processPayment(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	
	// Simulate inventory update
	if err := bo.updateInventory(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	
	// Simulate notification
	if err := bo.sendNotification(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	
	span.SetAttributes(attribute.String("order.status", "completed"))
	span.SetStatus(codes.Ok, "Order processed successfully")
	return nil
}

func (bo *BusinessOperation) validateOrder(ctx context.Context, orderID string) error {
	ctx, span := bo.tp.StartSpan(ctx, "validate_order",
		oteltrace.WithAttributes(
			attribute.String("order.id", orderID),
		),
	)
	defer span.End()
	
	// Simulate validation time
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	
	// Simulate occasional validation failure
	if rand.Float32() < 0.05 {
		err := fmt.Errorf("invalid order: %s", orderID)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	
	span.SetStatus(codes.Ok, "Order validated")
	return nil
}

func (bo *BusinessOperation) processPayment(ctx context.Context, orderID string) error {
	ctx, span := bo.tp.StartSpan(ctx, "process_payment",
		oteltrace.WithAttributes(
			attribute.String("order.id", orderID),
			attribute.String("payment.method", "credit_card"),
		),
	)
	defer span.End()
	
	// Simulate payment processing time
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
	
	// Simulate occasional payment failure
	if rand.Float32() < 0.03 {
		err := fmt.Errorf("payment failed for order: %s", orderID)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	
	span.SetAttributes(
		attribute.String("payment.status", "completed"),
		attribute.Float64("payment.amount", rand.Float64()*1000),
	)
	span.SetStatus(codes.Ok, "Payment processed")
	return nil
}

func (bo *BusinessOperation) updateInventory(ctx context.Context, orderID string) error {
	ctx, span := bo.tp.StartSpan(ctx, "update_inventory",
		oteltrace.WithAttributes(
			attribute.String("order.id", orderID),
		),
	)
	defer span.End()
	
	// Simulate database operation
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	
	// Simulate occasional inventory failure
	if rand.Float32() < 0.02 {
		err := fmt.Errorf("insufficient inventory for order: %s", orderID)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	
	span.SetAttributes(
		attribute.Int("inventory.items_updated", rand.Intn(10)+1),
	)
	span.SetStatus(codes.Ok, "Inventory updated")
	return nil
}

func (bo *BusinessOperation) sendNotification(ctx context.Context, orderID string) error {
	ctx, span := bo.tp.StartSpan(ctx, "send_notification",
		oteltrace.WithAttributes(
			attribute.String("order.id", orderID),
			attribute.String("notification.type", "email"),
		),
	)
	defer span.End()
	
	// Simulate notification sending
	time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)
	
	span.SetAttributes(
		attribute.String("notification.status", "sent"),
	)
	span.SetStatus(codes.Ok, "Notification sent")
	return nil
}

// HTTP Handlers
func (tp *TracerProvider) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, span := tp.StartSpan(r.Context(), "health_check")
	defer span.End()
	
	// Simulate health check
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   tp.config.ServiceName,
		"version":   tp.config.ServiceVersion,
	}
	
	span.SetAttributes(
		attribute.String("health.status", "healthy"),
	)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (tp *TracerProvider) orderHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	orderID := r.URL.Query().Get("id")
	if orderID == "" {
		orderID = generateOrderID()
	}
	
	// Create business operation
	bo := NewBusinessOperation(tp)
	
	// Process order
	err := bo.ProcessOrder(ctx, orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"order_id": orderID,
		"status":   "completed",
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (tp *TracerProvider) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := tp.GetStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (tp *TracerProvider) simulateLoadHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tp.StartSpan(r.Context(), "simulate_load")
	defer span.End()
	
	// Get number of requests from query parameter
	numRequests := 100
	if n := r.URL.Query().Get("requests"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil {
			numRequests = parsed
		}
	}
	
	span.SetAttributes(
		attribute.Int("load.requests", numRequests),
	)
	
	// Simulate concurrent load
	var wg sync.WaitGroup
	start := time.Now()
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Create child span for each simulated request
			childCtx, childSpan := tp.StartSpan(ctx, fmt.Sprintf("simulated_request_%d", id),
				oteltrace.WithAttributes(
					attribute.Int("request.id", id),
				),
			)
			defer childSpan.End()
			
			// Simulate work
			bo := NewBusinessOperation(tp)
			orderID := fmt.Sprintf("sim-%d-%d", time.Now().Unix(), id)
			bo.ProcessOrder(childCtx, orderID)
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	response := map[string]interface{}{
		"requests_processed": numRequests,
		"duration_ms":        duration.Milliseconds(),
		"requests_per_second": float64(numRequests) / duration.Seconds(),
	}
	
	span.SetAttributes(
		attribute.Int64("load.duration_ms", duration.Milliseconds()),
		attribute.Float64("load.rps", float64(numRequests)/duration.Seconds()),
	)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Utility functions
func generateRequestID() string {
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), rand.Intn(10000))
}

func generateOrderID() string {
	return fmt.Sprintf("order-%d-%d", time.Now().Unix(), rand.Intn(10000))
}

func main() {
	// Configuration
	config := TracerConfig{
		ServiceName:     "distributed-tracing-demo",
		ServiceVersion:  "1.0.0",
		Environment:     "development",
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		SamplingRatio:   1.0, // Sample all traces for demo
		BatchTimeout:    time.Second * 5,
		MaxBatchSize:    512,
	}
	
	// Create tracer provider
	tp, err := NewTracerProvider(config)
	if err != nil {
		log.Fatalf("Failed to create tracer provider: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		tp.Shutdown(ctx)
	}()
	
	// Setup HTTP server
	mux := http.NewServeMux()
	
	// Add routes
	mux.HandleFunc("/health", tp.healthHandler)
	mux.HandleFunc("/order", tp.orderHandler)
	mux.HandleFunc("/stats", tp.statsHandler)
	mux.HandleFunc("/simulate", tp.simulateLoadHandler)
	
	// Add middleware
	handler := tp.HTTPMiddleware(mux)
	
	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}
	
	fmt.Printf("Starting distributed tracing demo server on :8080\n")
	fmt.Printf("Service: %s v%s\n", config.ServiceName, config.ServiceVersion)
	fmt.Printf("Environment: %s\n", config.Environment)
	fmt.Printf("Jaeger endpoint: %s\n", config.JaegerEndpoint)
	fmt.Printf("Sampling ratio: %.2f\n", config.SamplingRatio)
	fmt.Println()
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /health          - Health check")
	fmt.Println("  GET  /order?id=<id>   - Process order")
	fmt.Println("  GET  /stats           - Tracing statistics")
	fmt.Println("  GET  /simulate?requests=<n> - Simulate load")
	fmt.Println()
	
	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	
	fmt.Println("\nShutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	fmt.Println("Server stopped")
}