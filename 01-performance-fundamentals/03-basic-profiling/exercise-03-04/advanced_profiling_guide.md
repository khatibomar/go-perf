# Exercise 03-04: Advanced Profiling Techniques

## Overview
This exercise demonstrates advanced profiling techniques in Go, including custom profilers, execution trace analysis, performance regression detection, and automated profiling workflows. These techniques are essential for production monitoring and continuous performance optimization.

## Learning Objectives
After completing this exercise, you will be able to:
- Implement custom profiling solutions
- Analyze execution traces for detailed performance insights
- Detect performance regressions automatically
- Set up continuous profiling workflows
- Integrate profiling into CI/CD pipelines
- Use advanced pprof features and analysis techniques

## Prerequisites
- Completion of Exercises 03-01, 03-02, and 03-03
- Understanding of Go profiling basics
- Familiarity with pprof and trace tools

## Exercise Structure

### Part 1: Custom Profiling Implementation

#### 1.1 Understanding Custom Profilers

Custom profilers allow you to collect application-specific metrics beyond what standard profiling tools provide. They're useful for:
- Business logic performance metrics
- Custom resource usage tracking
- Application-specific bottleneck detection
- Integration with monitoring systems

#### 1.2 Running the Custom Profiler

```bash
# Navigate to exercise directory
cd 01-performance-fundamentals/03-basic-profiling/exercise-03-04

# Run the advanced profiling demo
go run advanced_profiling.go
```

#### 1.3 Custom Profiler Features

The `CustomProfiler` class demonstrates:
- **Periodic Sampling**: Collects metrics at regular intervals
- **Multi-metric Collection**: Goroutines, memory, GC, CPU usage
- **Custom Data Points**: Application-specific metrics
- **Export Capabilities**: CSV export for analysis
- **Thread-safe Operations**: Concurrent access protection

#### 1.4 Analyzing Custom Metrics

```bash
# View collected metrics
cat advanced_profiles/custom_metrics.csv

# Import into analysis tools
# Excel, Google Sheets, or data analysis tools
# Python pandas, R, or other statistical tools
```

#### 1.5 Custom Profiler Implementation Details

```go
// Key components of the custom profiler:

type CustomProfiler struct {
    mu       sync.RWMutex
    samples  []ProfileSample
    active   bool
    interval time.Duration
    stopCh   chan struct{}
}

type ProfileSample struct {
    Timestamp   time.Time
    Goroutines  int
    MemoryAlloc uint64
    MemorySys   uint64
    NumGC       uint32
    CPUUsage    float64
    CustomData  map[string]interface{}
}
```

### Part 2: Execution Trace Analysis

#### 2.1 Understanding Execution Traces

Execution traces provide detailed information about:
- Goroutine scheduling and execution
- Garbage collection events
- Network and system call blocking
- Lock contention and synchronization
- Processor utilization patterns

#### 2.2 Collecting Execution Traces

```bash
# The demo automatically collects traces
# Manual trace collection:
go run advanced_profiling.go

# Analyze the generated trace
go tool trace advanced_profiles/trace.out
```

#### 2.3 Trace Analysis Web Interface

The trace tool provides several views:
- **Timeline View**: Goroutine execution over time
- **Goroutine Analysis**: Individual goroutine behavior
- **Network Blocking Profile**: Network I/O analysis
- **Synchronization Blocking Profile**: Lock contention
- **Syscall Blocking Profile**: System call analysis
- **Scheduler Latency Profile**: Scheduling delays

#### 2.4 Key Trace Analysis Techniques

```bash
# Open trace in browser
go tool trace advanced_profiles/trace.out

# Command-line trace analysis
go tool trace -pprof=net advanced_profiles/trace.out > net.prof
go tool trace -pprof=sync advanced_profiles/trace.out > sync.prof
go tool trace -pprof=syscall advanced_profiles/trace.out > syscall.prof

# Analyze extracted profiles
go tool pprof net.prof
go tool pprof sync.prof
go tool pprof syscall.prof
```

#### 2.5 Trace Analysis Insights

Look for:
- **Long-running Goroutines**: Potential blocking operations
- **Frequent GC**: Memory allocation issues
- **Scheduler Delays**: CPU contention or inefficient scheduling
- **Network Blocking**: I/O bottlenecks
- **Lock Contention**: Synchronization issues

### Part 3: Performance Regression Detection

#### 3.1 Understanding Regression Detection

Performance regression detection involves:
- Establishing baseline performance metrics
- Comparing current performance against baselines
- Identifying significant performance degradations
- Categorizing regression severity
- Automated alerting and reporting

#### 3.2 Regression Detection Components

```go
type PerformanceRegression struct {
    baseline    map[string]float64  // Baseline metrics
    current     map[string]float64  // Current metrics
    thresholds  map[string]float64  // Regression thresholds
    regressions []Regression        // Detected regressions
}

type Regression struct {
    Metric        string
    Baseline      float64
    Current       float64
    Change        float64
    ChangePercent float64
    Severity      string
}
```

#### 3.3 Running Regression Detection

```bash
# The demo includes regression detection
go run advanced_profiling.go

# Check regression report
cat advanced_profiles/regression_report.csv
```

#### 3.4 Regression Detection Logic

The system categorizes metrics into:
- **Latency Metrics**: Response time, duration (increase = bad)
- **Throughput Metrics**: QPS, RPS (decrease = bad)
- **Resource Metrics**: Memory, CPU (increase = bad)

Severity levels:
- **CRITICAL**: >3x threshold
- **HIGH**: >2x threshold
- **MEDIUM**: >1x threshold
- **LOW**: <1x threshold

#### 3.5 Integration with CI/CD

```yaml
# Example GitHub Actions workflow
name: Performance Regression Check
on: [pull_request]

jobs:
  performance-test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - uses: actions/setup-go@v2
      with:
        go-version: 1.21
    
    - name: Run Baseline Benchmarks
      run: |
        git checkout main
        go test -bench=. -benchmem -count=5 > baseline.txt
    
    - name: Run Current Benchmarks
      run: |
        git checkout ${{ github.sha }}
        go test -bench=. -benchmem -count=5 > current.txt
    
    - name: Detect Regressions
      run: |
        go run regression_detector.go baseline.txt current.txt
    
    - name: Upload Results
      uses: actions/upload-artifact@v2
      with:
        name: performance-results
        path: regression_report.csv
```

### Part 4: Advanced pprof Techniques

#### 4.1 Comparative Analysis

```bash
# Compare two CPU profiles
go tool pprof -base=baseline_cpu.prof current_cpu.prof

# Compare two memory profiles
go tool pprof -base=baseline_mem.prof current_mem.prof

# Focus on differences
(pprof) top -cum
(pprof) list function_name
```

#### 4.2 Advanced pprof Commands

```bash
# Web interface with custom port
go tool pprof -http=:8080 cpu.prof

# Generate flame graph
go tool pprof -png cpu.prof > flame.png
go tool pprof -svg cpu.prof > flame.svg

# Focus on specific functions
go tool pprof -focus=main.* cpu.prof
go tool pprof -ignore=runtime.* cpu.prof

# Sample-based analysis
go tool pprof -sample_index=alloc_space mem.prof
go tool pprof -sample_index=alloc_objects mem.prof
```

#### 4.3 Profile Filtering and Analysis

```bash
# Filter by function patterns
(pprof) focus main.*
(pprof) ignore runtime.*
(pprof) tagfocus key=value

# Different views
(pprof) top -cum          # Cumulative
(pprof) top -flat         # Flat
(pprof) tree              # Tree view
(pprof) peek function     # Function details

# Source code analysis
(pprof) list function_name
(pprof) weblist function_name
```

#### 4.4 Memory Profile Analysis

```bash
# Different memory views
go tool pprof -inuse_space mem.prof    # Current memory usage
go tool pprof -inuse_objects mem.prof  # Current object count
go tool pprof -alloc_space mem.prof    # Total allocated space
go tool pprof -alloc_objects mem.prof  # Total allocated objects

# Memory leak detection
go tool pprof -base=early.prof late.prof
```

### Part 5: Continuous Profiling Setup

#### 5.1 Production Profiling Strategy

```go
// Safe production profiling
func setupProductionProfiling() {
    // Enable profiling endpoints with authentication
    http.HandleFunc("/debug/pprof/", auth(pprof.Index))
    http.HandleFunc("/debug/pprof/cmdline", auth(pprof.Cmdline))
    http.HandleFunc("/debug/pprof/profile", auth(pprof.Profile))
    http.HandleFunc("/debug/pprof/symbol", auth(pprof.Symbol))
    http.HandleFunc("/debug/pprof/trace", auth(pprof.Trace))
    
    // Rate limiting for profile collection
    setupRateLimiting()
    
    // Automated profile collection
    go collectPeriodicProfiles()
}
```

#### 5.2 Automated Profile Collection

```bash
#!/bin/bash
# continuous_profiling.sh

APP_URL="https://your-app.com"
OUTPUT_DIR="/var/log/profiles"
INTERVAL=300  # 5 minutes

while true; do
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    PROFILE_DIR="$OUTPUT_DIR/$TIMESTAMP"
    mkdir -p "$PROFILE_DIR"
    
    # Collect profiles
    curl -o "$PROFILE_DIR/cpu.prof" "$APP_URL/debug/pprof/profile?seconds=30"
    curl -o "$PROFILE_DIR/heap.prof" "$APP_URL/debug/pprof/heap"
    curl -o "$PROFILE_DIR/goroutine.prof" "$APP_URL/debug/pprof/goroutine"
    
    # Analyze and alert if needed
    ./analyze_profiles.sh "$PROFILE_DIR"
    
    sleep $INTERVAL
done
```

#### 5.3 Profile Storage and Management

```bash
# Compress old profiles
find /var/log/profiles -name "*.prof" -mtime +7 -exec gzip {} \;

# Clean up old profiles
find /var/log/profiles -name "*.prof.gz" -mtime +30 -delete

# Upload to cloud storage
aws s3 sync /var/log/profiles s3://your-bucket/profiles/
```

#### 5.4 Monitoring and Alerting

```go
// Performance monitoring
type PerformanceMonitor struct {
    thresholds map[string]float64
    alerter    Alerter
}

func (pm *PerformanceMonitor) checkMetrics(metrics map[string]float64) {
    for metric, value := range metrics {
        if threshold, exists := pm.thresholds[metric]; exists {
            if value > threshold {
                pm.alerter.Alert(fmt.Sprintf(
                    "Performance threshold exceeded: %s = %.2f (threshold: %.2f)",
                    metric, value, threshold))
            }
        }
    }
}
```

### Part 6: Advanced Analysis Techniques

#### 6.1 Multi-dimensional Analysis

```bash
# Analyze by different dimensions
go tool pprof -tagfocus=version=v1.2.3 cpu.prof
go tool pprof -tagfocus=region=us-east-1 cpu.prof
go tool pprof -tagfocus=user_type=premium cpu.prof
```

#### 6.2 Time-series Analysis

```python
# Python script for time-series analysis
import pandas as pd
import matplotlib.pyplot as plt

# Load custom metrics
df = pd.read_csv('advanced_profiles/custom_metrics.csv')
df['timestamp'] = pd.to_datetime(df['timestamp'])

# Plot memory usage over time
plt.figure(figsize=(12, 6))
plt.plot(df['timestamp'], df['memory_alloc'] / 1024 / 1024, label='Memory (MB)')
plt.plot(df['timestamp'], df['goroutines'], label='Goroutines')
plt.xlabel('Time')
plt.ylabel('Value')
plt.legend()
plt.title('Performance Metrics Over Time')
plt.show()
```

#### 6.3 Statistical Analysis

```r
# R script for statistical analysis
library(ggplot2)
library(dplyr)

# Load data
data <- read.csv('advanced_profiles/custom_metrics.csv')

# Statistical summary
summary(data)

# Correlation analysis
cor(data[, c('memory_alloc', 'goroutines', 'num_gc')])

# Trend analysis
ggplot(data, aes(x = as.POSIXct(timestamp), y = memory_alloc)) +
  geom_line() +
  geom_smooth(method = 'lm') +
  labs(title = 'Memory Usage Trend', x = 'Time', y = 'Memory (bytes)')
```

### Part 7: Integration with Observability Tools

#### 7.1 Prometheus Integration

```go
// Prometheus metrics from profiling data
var (
    memoryUsage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "app_memory_usage_bytes",
            Help: "Current memory usage in bytes",
        },
        []string{"type"},
    )
    
    goroutineCount = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "app_goroutines_total",
            Help: "Current number of goroutines",
        },
    )
)

func updatePrometheusMetrics(sample ProfileSample) {
    memoryUsage.WithLabelValues("alloc").Set(float64(sample.MemoryAlloc))
    memoryUsage.WithLabelValues("sys").Set(float64(sample.MemorySys))
    goroutineCount.Set(float64(sample.Goroutines))
}
```

#### 7.2 Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Go Application Performance",
    "panels": [
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "app_memory_usage_bytes{type=\"alloc\"}",
            "legendFormat": "Allocated"
          },
          {
            "expr": "app_memory_usage_bytes{type=\"sys\"}",
            "legendFormat": "System"
          }
        ]
      },
      {
        "title": "Goroutine Count",
        "type": "singlestat",
        "targets": [
          {
            "expr": "app_goroutines_total",
            "legendFormat": "Goroutines"
          }
        ]
      }
    ]
  }
}
```

#### 7.3 Jaeger Tracing Integration

```go
// Distributed tracing with profiling context
func profiledHandler(w http.ResponseWriter, r *http.Request) {
    span, ctx := opentracing.StartSpanFromContext(r.Context(), "handler")
    defer span.Finish()
    
    // Add profiling context to span
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    span.SetTag("goroutines", runtime.NumGoroutine())
    span.SetTag("memory.alloc", m.Alloc)
    span.SetTag("memory.sys", m.Sys)
    
    // Continue with handler logic
    handleRequest(ctx, w, r)
}
```

## Exercise Tasks

### Task 1: Custom Profiler Implementation
1. Run the advanced profiling demo
2. Examine the custom profiler implementation
3. Modify the profiler to collect additional metrics:
   - HTTP request latency
   - Database connection pool usage
   - Cache hit/miss ratios
4. Export the enhanced metrics and analyze trends

### Task 2: Execution Trace Analysis
1. Collect execution traces during different workload patterns
2. Use `go tool trace` to analyze:
   - Goroutine scheduling patterns
   - GC impact on performance
   - Lock contention issues
3. Identify optimization opportunities from trace analysis
4. Document findings and recommendations

### Task 3: Regression Detection System
1. Implement a benchmark-based regression detection system
2. Create baseline performance metrics for your application
3. Set up automated regression detection in CI/CD
4. Test the system with intentional performance regressions
5. Configure alerting for critical regressions

### Task 4: Production Profiling Setup
1. Design a safe production profiling strategy
2. Implement rate-limited profile collection
3. Set up automated profile storage and rotation
4. Create monitoring dashboards for key metrics
5. Establish alerting thresholds and procedures

### Task 5: Advanced Analysis Workflow
1. Create a comprehensive analysis workflow combining:
   - Standard pprof analysis
   - Custom metric collection
   - Trace analysis
   - Regression detection
2. Automate the workflow with scripts
3. Generate automated performance reports
4. Integrate with existing monitoring infrastructure

## Common Patterns and Best Practices

### 1. Safe Production Profiling
- Use authentication for profiling endpoints
- Implement rate limiting for profile collection
- Limit profile duration (typically 10-30 seconds)
- Monitor profiling overhead
- Use sampling for heap profiles

### 2. Effective Regression Detection
- Establish stable baseline metrics
- Use appropriate statistical methods
- Set context-aware thresholds
- Consider metric interdependencies
- Implement severity-based alerting

### 3. Comprehensive Monitoring
- Combine multiple profiling techniques
- Use both reactive and proactive monitoring
- Correlate performance with business metrics
- Maintain historical performance data
- Regular performance review cycles

### 4. Automation and Integration
- Automate profile collection and analysis
- Integrate with CI/CD pipelines
- Use infrastructure as code for monitoring setup
- Implement self-healing mechanisms
- Regular backup and disaster recovery

## Troubleshooting Common Issues

### 1. High Profiling Overhead
```bash
# Reduce profiling frequency
# Use shorter profile durations
# Implement sampling strategies
# Monitor profiling impact
```

### 2. False Positive Regressions
```bash
# Adjust statistical thresholds
# Use longer baseline periods
# Implement noise filtering
# Consider environmental factors
```

### 3. Storage and Performance
```bash
# Compress old profiles
# Implement retention policies
# Use efficient storage backends
# Optimize analysis queries
```

### 4. Integration Challenges
```bash
# Use standard formats (pprof, OpenTelemetry)
# Implement proper error handling
# Use circuit breakers for external dependencies
# Maintain backward compatibility
```

## Expected Learning Outcomes

After completing this exercise, you should be able to:
- Implement custom profiling solutions for specific needs
- Analyze execution traces for detailed performance insights
- Set up automated performance regression detection
- Design and implement production-ready profiling systems
- Integrate profiling with observability and monitoring tools
- Use advanced pprof features for deep performance analysis
- Create comprehensive performance monitoring workflows

## Next Steps

1. **Chapter 04**: Proceed to benchmarking best practices
2. **Production Implementation**: Apply these techniques to real applications
3. **Advanced Topics**: Explore distributed profiling and tracing
4. **Tool Integration**: Integrate with your existing monitoring stack
5. **Team Training**: Share knowledge with your development team

## Additional Resources

- [Go Execution Tracer](https://golang.org/doc/diagnostics.html#execution-tracer)
- [pprof Documentation](https://github.com/google/pprof)
- [Continuous Profiling Best Practices](https://grafana.com/docs/pyroscope/)
- [Performance Testing in CI/CD](https://martinfowler.com/articles/practical-test-pyramid.html)
- [Observability Engineering](https://www.oreilly.com/library/view/observability-engineering/9781492076438/)

This completes the advanced profiling techniques exercise. The skills learned here form the foundation for sophisticated performance monitoring and optimization in production Go applications.