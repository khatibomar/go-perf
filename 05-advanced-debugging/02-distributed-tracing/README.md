# Chapter 02: Distributed Tracing and Observability

**Focus:** Comprehensive observability for distributed Go applications

## Chapter Overview

This chapter focuses on implementing comprehensive observability for distributed Go applications using modern tracing, metrics, and logging solutions. You'll learn to implement OpenTelemetry, build custom observability solutions, create effective dashboards, and establish robust alerting systems.

## Learning Objectives

By the end of this chapter, you will:

- ✅ Implement OpenTelemetry for comprehensive tracing
- ✅ Build custom metrics and logging solutions
- ✅ Create effective monitoring dashboards
- ✅ Establish robust alerting and incident response
- ✅ Correlate traces, metrics, and logs effectively
- ✅ Implement distributed system observability patterns
- ✅ Optimize observability overhead and costs

## Prerequisites

- Understanding of distributed systems
- Experience with microservices architecture
- Knowledge of monitoring concepts
- Familiarity with time-series databases

## Chapter Structure

### Exercise 02-01: OpenTelemetry Implementation
**Objective:** Implement comprehensive OpenTelemetry instrumentation

**Key Concepts:**
- Distributed tracing setup
- Automatic and manual instrumentation
- Trace sampling strategies
- Context propagation

**Performance Targets:**
- Trace completion rate >99.9%
- Instrumentation overhead <2%
- Trace latency <10ms
- Support for 1M+ spans/minute

### Exercise 02-02: Custom Metrics and Logging
**Objective:** Build comprehensive metrics and structured logging

**Key Concepts:**
- Custom metric collection
- Structured logging implementation
- Log aggregation strategies
- Metric cardinality management

**Performance Targets:**
- Metric ingestion >100K metrics/second
- Log processing latency <100ms
- Storage efficiency >80%
- Query response time <1 second

### Exercise 02-03: Monitoring Dashboards
**Objective:** Create effective monitoring and alerting dashboards

**Key Concepts:**
- Dashboard design principles
- SLI/SLO implementation
- Alert rule configuration
- Visualization best practices

**Performance Targets:**
- Dashboard load time <2 seconds
- Real-time data updates
- Alert response time <30 seconds
- 99.9% dashboard availability

### Exercise 02-04: Alerting and Incident Response
**Objective:** Implement robust alerting and incident response systems

**Key Concepts:**
- Alert rule optimization
- Incident response automation
- Escalation procedures
- Post-incident analysis

**Performance Targets:**
- Alert accuracy >95%
- False positive rate <5%
- Mean time to detection <2 minutes
- Mean time to resolution <15 minutes

## Key Technologies

### Tracing Solutions
- **OpenTelemetry** - Unified observability framework
- **Jaeger** - Distributed tracing platform
- **Zipkin** - Distributed tracing system
- **AWS X-Ray** - Cloud-native tracing

### Metrics and Monitoring
- **Prometheus** - Time-series database
- **Grafana** - Visualization platform
- **InfluxDB** - Time-series database
- **DataDog** - Cloud monitoring

### Logging Solutions
- **ELK Stack** - Elasticsearch, Logstash, Kibana
- **Fluentd** - Log collector
- **Loki** - Log aggregation system
- **Splunk** - Enterprise logging

### Alerting Systems
- **AlertManager** - Prometheus alerting
- **PagerDuty** - Incident management
- **Opsgenie** - Alert management
- **Slack/Teams** - Communication integration

## Implementation Patterns

### OpenTelemetry Setup
```go
// telemetry/tracer.go
package telemetry

import (
    "context"
    "fmt"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/semconv/v1.12.0"
    oteltrace "go.opentelemetry.io/otel/trace"
)

type TracerConfig struct {
    ServiceName     string
    ServiceVersion  string
    Environment     string
    JaegerEndpoint  string
    SamplingRatio   float64
    BatchTimeout    time.Duration
    MaxBatchSize    int
}

type TracerProvider struct {
    provider *trace.TracerProvider
    tracer   oteltrace.Tracer
    config   TracerConfig
}

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

    // Create trace provider
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
    }, nil
}

func (tp *TracerProvider) StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
    return tp.tracer.Start(ctx, name, opts...)
}

func (tp *TracerProvider) Shutdown(ctx context.Context) error {
    return tp.provider.Shutdown(ctx)
}

// Middleware for HTTP tracing
func (tp *TracerProvider) HTTPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
        
        spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
        ctx, span := tp.StartSpan(ctx, spanName,
            oteltrace.WithAttributes(
                semconv.HTTPMethodKey.String(r.Method),
                semconv.HTTPURLKey.String(r.URL.String()),
                semconv.HTTPUserAgentKey.String(r.UserAgent()),
            ),
        )
        defer span.End()

        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        next.ServeHTTP(wrapped, r.WithContext(ctx))
        
        span.SetAttributes(
            semconv.HTTPStatusCodeKey.Int(wrapped.statusCode),
        )
        
        if wrapped.statusCode >= 400 {
            span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapped.statusCode))
        }
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

### Custom Metrics Implementation
```go
// metrics/collector.go
package metrics

import (
    "context"
    "sync"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsCollector struct {
    // HTTP metrics
    httpRequestsTotal    *prometheus.CounterVec
    httpRequestDuration  *prometheus.HistogramVec
    httpRequestsInFlight prometheus.Gauge
    
    // Business metrics
    businessOperationsTotal    *prometheus.CounterVec
    businessOperationDuration *prometheus.HistogramVec
    businessErrors             *prometheus.CounterVec
    
    // System metrics
    goroutineCount     prometheus.Gauge
    memoryUsage        prometheus.Gauge
    gcDuration         *prometheus.HistogramVec
    
    // Custom metrics
    customCounters   map[string]*prometheus.CounterVec
    customGauges     map[string]*prometheus.GaugeVec
    customHistograms map[string]*prometheus.HistogramVec
    
    mu sync.RWMutex
}

func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
    mc := &MetricsCollector{
        httpRequestsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "http_requests_total",
                Help:      "Total number of HTTP requests",
            },
            []string{"method", "endpoint", "status_code"},
        ),
        
        httpRequestDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "http_request_duration_seconds",
                Help:      "HTTP request duration in seconds",
                Buckets:   prometheus.DefBuckets,
            },
            []string{"method", "endpoint"},
        ),
        
        httpRequestsInFlight: promauto.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "http_requests_in_flight",
                Help:      "Number of HTTP requests currently being processed",
            },
        ),
        
        businessOperationsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "business_operations_total",
                Help:      "Total number of business operations",
            },
            []string{"operation", "status"},
        ),
        
        businessOperationDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "business_operation_duration_seconds",
                Help:      "Business operation duration in seconds",
                Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
            },
            []string{"operation"},
        ),
        
        businessErrors: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "business_errors_total",
                Help:      "Total number of business errors",
            },
            []string{"operation", "error_type"},
        ),
        
        goroutineCount: promauto.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "goroutines_count",
                Help:      "Number of goroutines",
            },
        ),
        
        memoryUsage: promauto.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "memory_usage_bytes",
                Help:      "Memory usage in bytes",
            },
        ),
        
        gcDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "gc_duration_seconds",
                Help:      "Garbage collection duration in seconds",
                Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
            },
            []string{"gc_type"},
        ),
        
        customCounters:   make(map[string]*prometheus.CounterVec),
        customGauges:     make(map[string]*prometheus.GaugeVec),
        customHistograms: make(map[string]*prometheus.HistogramVec),
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
    mc.businessOperationsTotal.WithLabelValues(operation, status).Inc()
    mc.businessOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (mc *MetricsCollector) RecordBusinessError(operation, errorType string) {
    mc.businessErrors.WithLabelValues(operation, errorType).Inc()
}

// Custom Metrics
func (mc *MetricsCollector) RegisterCustomCounter(name, help string, labels []string) {
    mc.mu.Lock()
    defer mc.mu.Unlock()
    
    if _, exists := mc.customCounters[name]; !exists {
        mc.customCounters[name] = promauto.NewCounterVec(
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
        mc.customGauges[name] = promauto.NewGaugeVec(
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
        mc.memoryUsage.Set(float64(m.Alloc))
        
        // GC stats are collected via runtime hooks
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
```

### Structured Logging Implementation
```go
// logging/logger.go
package logging

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "runtime"
    "time"

    "go.opentelemetry.io/otel/trace"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type LogLevel string

const (
    DebugLevel LogLevel = "debug"
    InfoLevel  LogLevel = "info"
    WarnLevel  LogLevel = "warn"
    ErrorLevel LogLevel = "error"
    FatalLevel LogLevel = "fatal"
)

type LoggerConfig struct {
    Level       LogLevel
    Environment string
    ServiceName string
    Version     string
    OutputPath  string
    ErrorPath   string
    Encoding    string // "json" or "console"
}

type Logger struct {
    zap    *zap.Logger
    config LoggerConfig
}

func NewLogger(config LoggerConfig) (*Logger, error) {
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

    return &Logger{
        zap:    zapLogger,
        config: config,
    }, nil
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
    fields := l.extractContextFields(ctx)
    return &Logger{
        zap:    l.zap.With(fields...),
        config: l.config,
    }
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
    zapFields := make([]zap.Field, 0, len(fields))
    for k, v := range fields {
        zapFields = append(zapFields, zap.Any(k, v))
    }
    
    return &Logger{
        zap:    l.zap.With(zapFields...),
        config: l.config,
    }
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
    l.zap.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
    l.zap.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
    l.zap.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
    l.zap.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
    l.zap.Fatal(msg, fields...)
}

// Structured logging methods
func (l *Logger) LogHTTPRequest(r *http.Request, statusCode int, duration time.Duration, responseSize int64) {
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

func (l *Logger) LogBusinessOperation(operation string, success bool, duration time.Duration, metadata map[string]interface{}) {
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

func (l *Logger) LogError(err error, operation string, metadata map[string]interface{}) {
    fields := []zap.Field{
        zap.Error(err),
        zap.String("operation", operation),
    }
    
    for k, v := range metadata {
        fields = append(fields, zap.Any(k, v))
    }
    
    l.Error("Operation error", fields...)
}

func (l *Logger) extractContextFields(ctx context.Context) []zap.Field {
    var fields []zap.Field
    
    // Extract trace information
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        fields = append(fields,
            zap.String("trace_id", span.SpanContext().TraceID().String()),
            zap.String("span_id", span.SpanContext().SpanID().String()),
        )
    }
    
    // Extract request ID if available
    if requestID := ctx.Value("request_id"); requestID != nil {
        fields = append(fields, zap.String("request_id", fmt.Sprintf("%v", requestID)))
    }
    
    // Extract user ID if available
    if userID := ctx.Value("user_id"); userID != nil {
        fields = append(fields, zap.String("user_id", fmt.Sprintf("%v", userID)))
    }
    
    return fields
}

func getZapLevel(level LogLevel) zapcore.Level {
    switch level {
    case DebugLevel:
        return zapcore.DebugLevel
    case InfoLevel:
        return zapcore.InfoLevel
    case WarnLevel:
        return zapcore.WarnLevel
    case ErrorLevel:
        return zapcore.ErrorLevel
    case FatalLevel:
        return zapcore.FatalLevel
    default:
        return zapcore.InfoLevel
    }
}

func (l *Logger) Sync() error {
    return l.zap.Sync()
}

// HTTP Middleware
func (l *Logger) HTTPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        // Add request ID to context
        requestID := generateRequestID()
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        
        next.ServeHTTP(wrapped, r.WithContext(ctx))
        
        duration := time.Since(start)
        l.WithContext(ctx).LogHTTPRequest(r, wrapped.statusCode, duration, wrapped.size)
    })
}

func generateRequestID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

## Profiling and Debugging

### Observability Stack Setup
```yaml
# docker-compose.observability.yml
version: '3.8'

services:
  # Jaeger for distributed tracing
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "14268:14268"
    environment:
      - COLLECTOR_OTLP_ENABLED=true

  # Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'

  # Grafana for visualization
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-storage:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources

  # Elasticsearch for logs
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.5.0
    environment:
      - discovery.type=single-node
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
      - xpack.security.enabled=false
    ports:
      - "9200:9200"
    volumes:
      - elasticsearch-data:/usr/share/elasticsearch/data

  # Kibana for log visualization
  kibana:
    image: docker.elastic.co/kibana/kibana:8.5.0
    ports:
      - "5601:5601"
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    depends_on:
      - elasticsearch

  # Logstash for log processing
  logstash:
    image: docker.elastic.co/logstash/logstash:8.5.0
    volumes:
      - ./logstash/pipeline:/usr/share/logstash/pipeline
    ports:
      - "5044:5044"
    depends_on:
      - elasticsearch

volumes:
  grafana-storage:
  elasticsearch-data:
```

### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  - job_name: 'go-app'
    static_configs:
      - targets: ['host.docker.internal:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s
    
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
```

### Alert Rules
```yaml
# alert_rules.yml
groups:
  - name: go-app-alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status_code=~"5.."}[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors per second"
          
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "95th percentile latency is {{ $value }} seconds"
          
      - alert: HighMemoryUsage
        expr: memory_usage_bytes / (1024 * 1024 * 1024) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value }}GB"
```

## Performance Optimization Techniques

### 1. Tracing Optimization
- Implement intelligent sampling
- Optimize span creation overhead
- Use batch exporters
- Minimize context propagation cost

### 2. Metrics Optimization
- Control metric cardinality
- Use appropriate metric types
- Implement efficient aggregation
- Optimize collection intervals

### 3. Logging Optimization
- Use structured logging
- Implement log levels properly
- Use asynchronous logging
- Optimize log formatting

### 4. Storage Optimization
- Implement data retention policies
- Use appropriate compression
- Optimize query performance
- Implement data archiving

## Common Pitfalls

❌ **Observability Issues:**
- High observability overhead
- Poor sampling strategies
- Excessive metric cardinality
- Inadequate correlation

❌ **Performance Problems:**
- Blocking observability operations
- Inefficient data collection
- Poor storage optimization
- Inadequate monitoring

❌ **Operational Issues:**
- Poor alert configuration
- Inadequate dashboard design
- Missing SLI/SLO definitions
- Poor incident response

❌ **Cost Management:**
- Excessive data retention
- Poor resource allocation
- Inadequate cost monitoring
- Missing optimization strategies

## Best Practices

### Observability Strategy
- Define clear observability goals
- Implement comprehensive instrumentation
- Use standardized formats
- Establish data governance
- Monitor observability costs

### Implementation Guidelines
- Start with essential metrics
- Implement gradual rollout
- Use feature flags for observability
- Implement proper error handling
- Document observability practices

### Operational Excellence
- Establish SLI/SLO definitions
- Implement effective alerting
- Create runbooks for incidents
- Conduct regular reviews
- Train team members

### Cost Optimization
- Implement data retention policies
- Monitor observability costs
- Optimize sampling strategies
- Use appropriate storage tiers
- Regular cost reviews

## Success Criteria

- [ ] Comprehensive tracing implementation
- [ ] Effective metrics and logging
- [ ] Functional monitoring dashboards
- [ ] Robust alerting system
- [ ] Observability overhead <2%
- [ ] Alert accuracy >95%
- [ ] Mean time to detection <2 minutes
- [ ] Team trained on observability tools

## Real-World Applications

- **Microservices Monitoring:** End-to-end observability
- **Performance Optimization:** Data-driven improvements
- **Incident Response:** Quick problem identification
- **Capacity Planning:** Resource requirement analysis
- **Business Intelligence:** Operational insights
- **Compliance:** Audit trail maintenance

## Next Steps

After completing this chapter:
1. Proceed to **Chapter 03: Production Debugging Techniques**
2. Implement observability in your applications
3. Study advanced observability patterns
4. Explore cloud-native observability solutions
5. Develop observability best practices

---

**Ready to implement comprehensive observability?** Start with [Exercise 02-01: OpenTelemetry Implementation](./exercise-02-01/README.md)

*Chapter 02 of Module 05: Advanced Debugging and Production*