# Exercise 02-02: Custom Metrics and Structured Logging

This exercise implements a comprehensive custom metrics and structured logging system using Prometheus and Zap.

## Features Implemented

### Custom Metrics (Prometheus)
- **HTTP Metrics**: Request count, duration, and in-flight requests
- **Business Metrics**: Operation counters, duration histograms, and error tracking
- **System Metrics**: CPU usage, memory usage, and goroutine count
- **Custom Metrics**: Dynamic registration of counters, gauges, and histograms

### Structured Logging (Zap)
- **Configurable Levels**: Debug, Info, Warn, Error, Fatal
- **Multiple Formats**: JSON and console encoding
- **Contextual Fields**: Service metadata and custom fields
- **HTTP Request Logging**: Detailed request/response logging
- **Business Operation Logging**: Operation success/failure tracking

### Performance Features
- **High Throughput**: Optimized for >100K metrics/second
- **Low Latency**: Sub-millisecond logging performance
- **Concurrent Safe**: Thread-safe metrics and logging
- **Memory Efficient**: Minimal allocation overhead

## Architecture

### MetricsCollector
- Isolated Prometheus registries per instance
- HTTP middleware for automatic request tracking
- System metrics collection via background goroutine
- Dynamic custom metrics registration

### StructuredLogger
- Zap-based high-performance logging
- Configurable output paths and formats
- Contextual field support
- Specialized logging methods for different operations

### BusinessService
- Integrated metrics and logging
- Simulated business operations with random failures
- HTTP handlers for testing different scenarios
- Load testing endpoint for performance validation

## API Endpoints

- `GET /health` - Health check endpoint
- `GET /order?id=<order_id>` - Process order operation
- `GET /payment?id=<payment_id>` - Validate payment operation
- `GET /inventory?item_id=<item_id>&quantity=<qty>` - Update inventory
- `GET /loadtest?operations=<count>` - Load testing endpoint
- `GET /metrics` - Prometheus metrics endpoint

## Usage

### Running the Application
```bash
go run main.go
```

The server will start on port 8080 with the following endpoints available.

### Running Tests
```bash
go test -v
```

All tests should pass, including:
- Unit tests for metrics and logging components
- Integration tests for HTTP handlers
- Performance tests for throughput validation
- Concurrent safety tests

### Building
```bash
go build .
```

## Configuration

### Logger Configuration
```go
config := LoggerConfig{
    Level:       "info",
    Environment: "production",
    ServiceName: "my-service",
    Version:     "1.0.0",
    OutputPath:  "stdout",
    ErrorPath:   "stderr",
    Encoding:    "json",
}
```

### Metrics Configuration
```go
metrics := NewMetricsCollector("myapp", "api")
```

## Performance Targets

- **Metrics Ingestion**: >100,000 metrics/second
- **Logging Throughput**: >10,000 logs/second
- **HTTP Request Latency**: <1ms overhead
- **Memory Usage**: <100MB for typical workloads

## Testing

The implementation includes comprehensive tests:

- **Unit Tests**: Individual component testing
- **Integration Tests**: End-to-end HTTP handler testing
- **Performance Tests**: Throughput and latency validation
- **Concurrent Tests**: Thread safety verification
- **Benchmark Tests**: Performance profiling

## Dependencies

- `github.com/prometheus/client_golang` - Prometheus metrics
- `go.uber.org/zap` - High-performance logging
- Standard library packages for HTTP and runtime metrics

## Key Implementation Details

### Metrics Isolation
Each MetricsCollector instance uses its own Prometheus registry to avoid conflicts during testing and enable multiple instances.

### Error Simulation
Business operations include random failure simulation (2-5% failure rate) to demonstrate error tracking and logging.

### System Metrics
Background goroutine collects system metrics (memory, goroutines, simulated CPU) every 15 seconds.

### HTTP Middleware
Automatic request tracking with method, path, status code, and duration metrics.

### Structured Context
All logs include service metadata (name, version, environment) and operation-specific context.

This implementation provides a production-ready foundation for observability in Go applications with both metrics and logging capabilities.