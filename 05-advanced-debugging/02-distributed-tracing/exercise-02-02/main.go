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
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MetricsCollector handles custom metrics collection
type MetricsCollector struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge

	// Business metrics
	businessOpsTotal    *prometheus.CounterVec
	businessOpsDuration *prometheus.HistogramVec
	businessErrors      *prometheus.CounterVec

	// System metrics
	systemCPUUsage    prometheus.Gauge
	systemMemoryUsage prometheus.Gauge
	goroutineCount    prometheus.Gauge

	// Custom metrics
	customCounters   map[string]*prometheus.CounterVec
	customGauges     map[string]*prometheus.GaugeVec
	customHistograms map[string]*prometheus.HistogramVec

	// Registry for this collector
	registry *prometheus.Registry

	mu sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	mc := &MetricsCollector{
		httpRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),

		httpRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		httpRequestsInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),

		businessOpsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "business_operations_total",
				Help:      "Total number of business operations",
			},
			[]string{"operation", "status"},
		),

		businessOpsDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "business_operation_duration_seconds",
				Help:      "Business operation duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"operation", "status"},
		),

		businessErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "business_errors_total",
				Help:      "Total number of business errors",
			},
			[]string{"operation", "error_type"},
		),

		systemCPUUsage: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "system_cpu_usage_percent",
				Help:      "Current CPU usage percentage",
			},
		),

		systemMemoryUsage: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "system_memory_usage_bytes",
				Help:      "Current memory usage in bytes",
			},
		),

		goroutineCount: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "goroutines_count",
				Help:      "Current number of goroutines",
			},
		),

		customCounters:   make(map[string]*prometheus.CounterVec),
		customGauges:     make(map[string]*prometheus.GaugeVec),
		customHistograms: make(map[string]*prometheus.HistogramVec),
		registry:         registry,
	}

	// Start system metrics collection
	go mc.collectSystemMetrics()

	return mc
}

// HTTP Metrics
func (mc *MetricsCollector) RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration) {
	mc.httpRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	mc.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func (mc *MetricsCollector) IncHTTPRequestsInFlight() {
	mc.httpRequestsInFlight.Inc()
}

func (mc *MetricsCollector) DecHTTPRequestsInFlight() {
	mc.httpRequestsInFlight.Dec()
}

// Business Metrics
func (mc *MetricsCollector) RecordBusinessOperation(operation, status string, duration time.Duration) {
	mc.businessOpsTotal.WithLabelValues(operation, status).Inc()
	mc.businessOpsDuration.WithLabelValues(operation, status).Observe(duration.Seconds())
}

func (mc *MetricsCollector) RecordBusinessError(operation, errorType string) {
	mc.businessErrors.WithLabelValues(operation, errorType).Inc()
}

// Custom Metrics
func (mc *MetricsCollector) RegisterCustomCounter(name, help string, labels []string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.customCounters[name]; !exists {
		factory := promauto.With(mc.registry)
		mc.customCounters[name] = factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: help,
			},
			labels,
		)
	}
}

func (mc *MetricsCollector) IncCustomCounter(name string, labelValues ...string) {
	mc.mu.RLock()
	counter, exists := mc.customCounters[name]
	mc.mu.RUnlock()

	if exists {
		counter.WithLabelValues(labelValues...).Inc()
	}
}

func (mc *MetricsCollector) RegisterCustomGauge(name, help string, labels []string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.customGauges[name]; !exists {
		factory := promauto.With(mc.registry)
		mc.customGauges[name] = factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name,
				Help: help,
			},
			labels,
		)
	}
}

func (mc *MetricsCollector) SetCustomGauge(name string, value float64, labelValues ...string) {
	mc.mu.RLock()
	gauge, exists := mc.customGauges[name]
	mc.mu.RUnlock()

	if exists {
		gauge.WithLabelValues(labelValues...).Set(value)
	}
}

func (mc *MetricsCollector) RegisterCustomHistogram(name, help string, labels []string, buckets []float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.customHistograms[name]; !exists {
		factory := promauto.With(mc.registry)
		mc.customHistograms[name] = factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    name,
				Help:    help,
				Buckets: buckets,
			},
			labels,
		)
	}
}

func (mc *MetricsCollector) ObserveCustomHistogram(name string, value float64, labelValues ...string) {
	mc.mu.RLock()
	histogram, exists := mc.customHistograms[name]
	mc.mu.RUnlock()

	if exists {
		histogram.WithLabelValues(labelValues...).Observe(value)
	}
}

// System metrics collection
func (mc *MetricsCollector) collectSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Collect goroutine count
		mc.goroutineCount.Set(float64(runtime.NumGoroutine()))

		// Collect memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		mc.systemMemoryUsage.Set(float64(m.Alloc))
		
		// Simulate CPU usage (in real implementation, use actual CPU metrics)
		mc.systemCPUUsage.Set(float64(rand.Intn(100)))
	}
}

// HTTP Middleware
func (mc *MetricsCollector) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mc.IncHTTPRequestsInFlight()
		defer mc.DecHTTPRequestsInFlight()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		mc.RecordHTTPRequest(
			r.Method,
			r.URL.Path,
			fmt.Sprintf("%d", wrapped.statusCode),
			duration,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.bytesWritten += int64(n)
	return n, err
}

// StructuredLogger handles structured logging
type StructuredLogger struct {
	zap    *zap.Logger
	config LoggerConfig
}

type LoggerConfig struct {
	Level       string
	Environment string
	ServiceName string
	Version     string
	OutputPath  string
	ErrorPath   string
	Encoding    string // "json" or "console"
}

func NewStructuredLogger(config LoggerConfig) (*StructuredLogger, error) {
	// Configure zap
	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(getZapLevel(config.Level)),
		Development: config.Environment == "development",
		Encoding:    config.Encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{config.OutputPath},
		ErrorOutputPaths: []string{config.ErrorPath},
		InitialFields: map[string]interface{}{
			"service":     config.ServiceName,
			"version":     config.Version,
			"environment": config.Environment,
		},
	}

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &StructuredLogger{
		zap:    zapLogger,
		config: config,
	}, nil
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	return &StructuredLogger{
		zap:    l.zap.With(zapFields...),
		config: l.config,
	}
}

func (l *StructuredLogger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

func (l *StructuredLogger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

func (l *StructuredLogger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

func (l *StructuredLogger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

func (l *StructuredLogger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
}

// Structured logging methods
func (l *StructuredLogger) LogHTTPRequest(r *http.Request, statusCode int, duration time.Duration, responseSize int64) {
	l.Info("HTTP request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("query", r.URL.RawQuery),
		zap.String("user_agent", r.UserAgent()),
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", duration),
		zap.Int64("response_size", responseSize),
	)
}

func (l *StructuredLogger) LogBusinessOperation(operation string, success bool, duration time.Duration, metadata map[string]interface{}) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.Bool("success", success),
		zap.Duration("duration", duration),
	}

	for k, v := range metadata {
		fields = append(fields, zap.Any(k, v))
	}

	if success {
		l.Info("Business operation completed", fields...)
	} else {
		l.Error("Business operation failed", fields...)
	}
}

func (l *StructuredLogger) LogError(err error, operation string, metadata map[string]interface{}) {
	fields := []zap.Field{
		zap.Error(err),
		zap.String("operation", operation),
	}

	for k, v := range metadata {
		fields = append(fields, zap.Any(k, v))
	}

	l.Error("Operation error", fields...)
}

// BusinessService simulates business operations with metrics and logging
type BusinessService struct {
	metrics *MetricsCollector
	logger  *StructuredLogger
}

func NewBusinessService(metrics *MetricsCollector, logger *StructuredLogger) *BusinessService {
	return &BusinessService{
		metrics: metrics,
		logger:  logger,
	}
}

func (bs *BusinessService) ProcessOrder(orderID string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		bs.metrics.RecordBusinessOperation("process_order", "success", duration)
		bs.logger.LogBusinessOperation("process_order", true, duration, map[string]interface{}{
			"order_id": orderID,
		})
	}()

	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	// Simulate occasional errors
	if rand.Float32() < 0.05 {
		err := fmt.Errorf("order processing failed: %s", orderID)
		bs.metrics.RecordBusinessError("process_order", "processing_error")
		bs.logger.LogError(err, "process_order", map[string]interface{}{
			"order_id": orderID,
		})
		return err
	}

	return nil
}

func (bs *BusinessService) ValidatePayment(paymentID string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		bs.metrics.RecordBusinessOperation("validate_payment", "success", duration)
		bs.logger.LogBusinessOperation("validate_payment", true, duration, map[string]interface{}{
			"payment_id": paymentID,
		})
	}()

	// Simulate validation time
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	// Simulate occasional validation failures
	if rand.Float32() < 0.03 {
		err := fmt.Errorf("payment validation failed: %s", paymentID)
		bs.metrics.RecordBusinessError("validate_payment", "validation_error")
		bs.logger.LogError(err, "validate_payment", map[string]interface{}{
			"payment_id": paymentID,
		})
		return err
	}

	return nil
}

func (bs *BusinessService) UpdateInventory(itemID string, quantity int) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		bs.metrics.RecordBusinessOperation("update_inventory", "success", duration)
		bs.logger.LogBusinessOperation("update_inventory", true, duration, map[string]interface{}{
			"item_id":  itemID,
			"quantity": quantity,
		})
	}()

	// Simulate database operation
	time.Sleep(time.Duration(rand.Intn(75)) * time.Millisecond)

	// Simulate occasional inventory errors
	if rand.Float32() < 0.02 {
		err := fmt.Errorf("inventory update failed: %s", itemID)
		bs.metrics.RecordBusinessError("update_inventory", "database_error")
		bs.logger.LogError(err, "update_inventory", map[string]interface{}{
			"item_id":  itemID,
			"quantity": quantity,
		})
		return err
	}

	return nil
}

// HTTP Handlers
func (bs *BusinessService) healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		bs.logger.LogHTTPRequest(r, 200, duration, 0)
	}()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "metrics-logging-demo",
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (bs *BusinessService) orderHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	statusCode := 200
	defer func() {
		duration := time.Since(start)
		bs.logger.LogHTTPRequest(r, statusCode, duration, 0)
	}()

	orderID := r.URL.Query().Get("id")
	if orderID == "" {
		orderID = generateOrderID()
	}

	err := bs.ProcessOrder(orderID)
	if err != nil {
		statusCode = 500
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"order_id":  orderID,
		"status":    "completed",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (bs *BusinessService) paymentHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	statusCode := 200
	defer func() {
		duration := time.Since(start)
		bs.logger.LogHTTPRequest(r, statusCode, duration, 0)
	}()

	paymentID := r.URL.Query().Get("id")
	if paymentID == "" {
		paymentID = generatePaymentID()
	}

	err := bs.ValidatePayment(paymentID)
	if err != nil {
		statusCode = 500
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"payment_id": paymentID,
		"status":     "validated",
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (bs *BusinessService) inventoryHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	statusCode := 200
	defer func() {
		duration := time.Since(start)
		bs.logger.LogHTTPRequest(r, statusCode, duration, 0)
	}()

	itemID := r.URL.Query().Get("item_id")
	quantityStr := r.URL.Query().Get("quantity")

	if itemID == "" {
		itemID = generateItemID()
	}

	quantity := 1
	if quantityStr != "" {
		if parsed, err := strconv.Atoi(quantityStr); err == nil {
			quantity = parsed
		}
	}

	err := bs.UpdateInventory(itemID, quantity)
	if err != nil {
		statusCode = 500
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"item_id":   itemID,
		"quantity":  quantity,
		"status":    "updated",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (bs *BusinessService) loadTestHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	statusCode := 200
	defer func() {
		duration := time.Since(start)
		bs.logger.LogHTTPRequest(r, statusCode, duration, 0)
	}()

	// Get number of operations from query parameter
	numOps := 100
	if n := r.URL.Query().Get("operations"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil {
			numOps = parsed
		}
	}

	// Simulate concurrent load
	var wg sync.WaitGroup
	loadStart := time.Now()

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Randomly choose operation type
			switch rand.Intn(3) {
			case 0:
				orderID := fmt.Sprintf("load-order-%d", id)
				bs.ProcessOrder(orderID)
			case 1:
				paymentID := fmt.Sprintf("load-payment-%d", id)
				bs.ValidatePayment(paymentID)
			case 2:
				itemID := fmt.Sprintf("load-item-%d", id)
				bs.UpdateInventory(itemID, rand.Intn(10)+1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(loadStart)

	response := map[string]interface{}{
		"operations_processed": numOps,
		"duration_ms":         duration.Milliseconds(),
		"operations_per_second": float64(numOps) / duration.Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Utility functions
func generateOrderID() string {
	return fmt.Sprintf("order-%d-%d", time.Now().Unix(), rand.Intn(10000))
}

func generatePaymentID() string {
	return fmt.Sprintf("payment-%d-%d", time.Now().Unix(), rand.Intn(10000))
}

func generateItemID() string {
	return fmt.Sprintf("item-%d-%d", time.Now().Unix(), rand.Intn(10000))
}

func main() {
	// Configuration
	loggerConfig := LoggerConfig{
		Level:       "info",
		Environment: "development",
		ServiceName: "metrics-logging-demo",
		Version:     "1.0.0",
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Encoding:    "json",
	}

	// Create structured logger
	logger, err := NewStructuredLogger(loggerConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// Create metrics collector
	metrics := NewMetricsCollector("demo", "metrics_logging")

	// Register custom metrics
	metrics.RegisterCustomCounter("custom_events_total", "Total custom events", []string{"event_type"})
	metrics.RegisterCustomGauge("custom_queue_size", "Current queue size", []string{"queue_name"})
	metrics.RegisterCustomHistogram("custom_processing_time", "Custom processing time", []string{"processor"}, []float64{0.001, 0.01, 0.1, 1, 10})

	// Create business service
	businessService := NewBusinessService(metrics, logger)

	// Setup HTTP server
	mux := http.NewServeMux()

	// Add routes
	mux.HandleFunc("/health", businessService.healthHandler)
	mux.HandleFunc("/order", businessService.orderHandler)
	mux.HandleFunc("/payment", businessService.paymentHandler)
	mux.HandleFunc("/inventory", businessService.inventoryHandler)
	mux.HandleFunc("/loadtest", businessService.loadTestHandler)
	mux.Handle("/metrics", promhttp.Handler())

	// Add middleware
	handler := metrics.HTTPMiddleware(mux)

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	logger.Info("Starting metrics and logging demo server",
		zap.String("address", ":8080"),
		zap.String("service", loggerConfig.ServiceName),
		zap.String("version", loggerConfig.Version),
		zap.String("environment", loggerConfig.Environment),
	)

	fmt.Println("Starting metrics and logging demo server on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /health                    - Health check")
	fmt.Println("  GET  /order?id=<id>             - Process order")
	fmt.Println("  GET  /payment?id=<id>           - Validate payment")
	fmt.Println("  GET  /inventory?item_id=<id>&quantity=<n> - Update inventory")
	fmt.Println("  GET  /loadtest?operations=<n>   - Load test")
	fmt.Println("  GET  /metrics                   - Prometheus metrics")
	fmt.Println()

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped")
}