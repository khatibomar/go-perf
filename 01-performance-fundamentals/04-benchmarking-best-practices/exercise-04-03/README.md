# Exercise 04-03: Advanced Benchmarking Techniques

## Overview
This exercise explores advanced benchmarking techniques in Go, including custom metrics, conditional execution, setup/teardown patterns, integration testing, regression detection, and production-ready benchmarking strategies. You'll learn to create sophisticated benchmark suites that provide deep insights into application performance.

## Learning Objectives
After completing this exercise, you will be able to:
- Implement custom metrics and advanced measurement techniques
- Design conditional benchmarks that adapt to different environments
- Create proper setup and teardown patterns for complex benchmarks
- Integrate benchmarking with databases, HTTP services, and external systems
- Detect and analyze performance regressions automatically
- Build production-ready benchmarking infrastructure
- Use context-aware and parametric benchmarking approaches
- Implement resource monitoring and system requirement validation

## Prerequisites
- Completion of Exercise 04-01 and 04-02
- Understanding of Go interfaces, contexts, and concurrency
- Basic knowledge of databases, HTTP services, and system programming
- Familiarity with testing frameworks and CI/CD concepts

## Exercise Structure

### Part 1: Custom Metrics and Advanced Measurements

#### 1.1 Understanding Custom Metrics
Go's testing package provides basic metrics (time, memory, allocations), but real-world applications often need custom measurements:

- **Throughput metrics**: Operations per second, bytes per second
- **Latency percentiles**: P50, P95, P99 response times
- **Error rates**: Success/failure ratios
- **Resource utilization**: CPU, memory, network usage
- **Business metrics**: Cache hit rates, queue depths, connection pools

#### 1.2 Running Custom Metrics Benchmarks

```bash
# Navigate to exercise directory
cd 01-performance-fundamentals/04-benchmarking-best-practices/exercise-04-03

# Install required dependencies
go mod init advanced-benchmarks
go get github.com/mattn/go-sqlite3

# Run custom metrics benchmarks
go test -bench=BenchmarkCustomMetrics -benchmem -v

# Run with extended output
go test -bench=BenchmarkCustomMetrics -benchmem -v -benchtime=5s
```

#### 1.3 Expected Custom Metrics Output

```
BenchmarkCustomMetrics-8    	    1000	   1200000 ns/op	     512 B/op	      10 allocs/op
                           	        	          	         	           	     168.0 primes/op
                           	        	          	         	           	 1180000.0 compute_ns/op
                           	        	          	         	           	   140.0 primes/sec
                           	        	          	         	           	    1000 total_ops
                           	        	          	         	           	 1200000 avg_latency_ns
                           	        	          	         	           	     0.0 error_rate_%
                           	        	          	         	           	   833.3 throughput_ops/sec
```

**Key Insights:**
- Custom metrics provide domain-specific performance insights
- Multiple metrics can be reported per benchmark
- Metrics help correlate performance with business outcomes

#### 1.4 Implementing Custom Metrics

```go
func BenchmarkWithCustomMetrics(b *testing.B) {
    var totalLatency time.Duration
    var successCount int64
    var errorCount int64
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        start := time.Now()
        
        // Your operation here
        err := performOperation()
        
        latency := time.Since(start)
        totalLatency += latency
        
        if err != nil {
            errorCount++
        } else {
            successCount++
        }
    }
    
    // Report custom metrics
    avgLatency := totalLatency / time.Duration(b.N)
    errorRate := float64(errorCount) / float64(b.N) * 100
    throughput := float64(b.N) / b.Elapsed().Seconds()
    
    b.ReportMetric(float64(avgLatency.Nanoseconds()), "avg_latency_ns")
    b.ReportMetric(errorRate, "error_rate_%")
    b.ReportMetric(throughput, "throughput_ops/sec")
    b.ReportMetric(float64(successCount), "success_count")
}
```

### Part 2: Conditional Benchmark Execution

#### 2.1 Environment-Based Conditions
Conditional benchmarks adapt to different environments and requirements:

- **System requirements**: CPU count, memory availability
- **Environment flags**: Short vs. long tests, specific subsystems
- **External dependencies**: Database availability, network access
- **Performance thresholds**: Skip slow tests in CI

#### 2.2 Running Conditional Benchmarks

```bash
# Run all benchmarks
go test -bench=BenchmarkConditionalExecution -v

# Run only short benchmarks
go test -bench=BenchmarkConditionalExecution -short -v

# Skip specific benchmark types
SKIP_MEMORY=true go test -bench=BenchmarkConditionalExecution -v
SKIP_DATABASE=true go test -bench=BenchmarkConditionalExecution -v

# Set minimum system requirements
MIN_CPUS=4 go test -bench=BenchmarkSystemRequirements -v
```

#### 2.3 Expected Conditional Output

```
=== RUN   BenchmarkConditionalExecution
=== RUN   BenchmarkConditionalExecution/FastOperation
=== RUN   BenchmarkConditionalExecution/SlowOperation
=== RUN   BenchmarkConditionalExecution/MemoryIntensive
--- SKIP: BenchmarkConditionalExecution/MemoryIntensive (0.00s)
    advanced_benchmarks_test.go:123: Skipping based on conditional configuration
BenchmarkConditionalExecution/FastOperation-8         	100000000	        12.0 ns/op
BenchmarkConditionalExecution/SlowOperation-8         	       30	  45000000 ns/op
```

#### 2.4 Implementing Conditional Logic

```go
func BenchmarkConditional(b *testing.B) {
    // Check system requirements
    if runtime.NumCPU() < 4 {
        b.Skip("Requires at least 4 CPUs")
    }
    
    // Check environment variables
    if os.Getenv("SKIP_SLOW") == "true" {
        b.Skip("Slow benchmarks disabled")
    }
    
    // Check for short mode
    if testing.Short() {
        b.Skip("Skipping in short mode")
    }
    
    // Your benchmark code here
}
```

### Part 3: Setup and Teardown Patterns

#### 3.1 Complex Environment Setup
Advanced benchmarks often require sophisticated setup:

- **Database initialization**: Schema creation, test data population
- **Service dependencies**: HTTP servers, message queues, caches
- **Resource allocation**: Temporary files, network ports, memory pools
- **State management**: Consistent initial conditions

#### 3.2 Running Setup/Teardown Benchmarks

```bash
# Run benchmarks with setup/teardown
go test -bench=BenchmarkSetupTeardown -v

# Run database benchmarks
go test -bench=BenchmarkDatabaseOperations -v

# Run HTTP benchmarks
go test -bench=BenchmarkHTTPOperations -v
```

#### 3.3 Expected Setup/Teardown Output

```
=== RUN   BenchmarkSetupTeardown
=== RUN   BenchmarkSetupTeardown/Small
BenchmarkSetupTeardown/Small-8    	   10000	    120000 ns/op	     256 B/op	       5 allocs/op
=== RUN   BenchmarkSetupTeardown/Medium
BenchmarkSetupTeardown/Medium-8   	    1000	   1200000 ns/op	    2560 B/op	      50 allocs/op
=== RUN   BenchmarkSetupTeardown/Large
BenchmarkSetupTeardown/Large-8    	     100	  12000000 ns/op	   25600 B/op	     500 allocs/op
```

#### 3.4 Implementing Proper Setup/Teardown

```go
func BenchmarkWithSetup(b *testing.B) {
    // Setup phase - not measured
    db, err := setupDatabase()
    if err != nil {
        b.Fatalf("Setup failed: %v", err)
    }
    defer db.Close() // Teardown
    
    testData := generateTestData(1000)
    
    b.ResetTimer() // Start measuring from here
    
    for i := 0; i < b.N; i++ {
        b.StopTimer() // Stop for per-iteration setup
        
        // Prepare clean state for this iteration
        cleanData := copyData(testData)
        
        b.StartTimer() // Resume measurement
        
        // Actual work being measured
        result := processData(db, cleanData)
        
        // Prevent optimization
        runtime.KeepAlive(result)
    }
    
    // Teardown happens in defer
}
```

### Part 4: Database Integration Benchmarking

#### 4.1 Database Benchmark Patterns
Database benchmarking requires special considerations:

- **Connection management**: Pool sizing, connection reuse
- **Transaction handling**: ACID properties, isolation levels
- **Data consistency**: Clean state between iterations
- **Query optimization**: Index usage, execution plans
- **Concurrency**: Multiple connections, lock contention

#### 4.2 Running Database Benchmarks

```bash
# Run database operation benchmarks
go test -bench=BenchmarkDatabaseOperations -v

# Run with memory profiling
go test -bench=BenchmarkDatabaseOperations -memprofile=db_mem.prof

# Run with CPU profiling
go test -bench=BenchmarkDatabaseOperations -cpuprofile=db_cpu.prof

# Analyze profiles
go tool pprof db_cpu.prof
go tool pprof db_mem.prof
```

#### 4.3 Expected Database Benchmark Output

```
=== RUN   BenchmarkDatabaseOperations
=== RUN   BenchmarkDatabaseOperations/Query10
BenchmarkDatabaseOperations/Query10-8     	   10000	    150000 ns/op	    1024 B/op	      25 allocs/op
=== RUN   BenchmarkDatabaseOperations/Query50
BenchmarkDatabaseOperations/Query50-8     	    2000	    750000 ns/op	    5120 B/op	     125 allocs/op
=== RUN   BenchmarkDatabaseOperations/Query100
BenchmarkDatabaseOperations/Query100-8    	    1000	   1500000 ns/op	   10240 B/op	     250 allocs/op
=== RUN   BenchmarkDatabaseOperations/CRUD
BenchmarkDatabaseOperations/CRUD-8        	    5000	    300000 ns/op	     512 B/op	      15 allocs/op
```

#### 4.4 Database Benchmark Best Practices

```go
func BenchmarkDatabase(b *testing.B) {
    // Setup database once per benchmark
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        b.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    // Create schema
    if err := createSchema(db); err != nil {
        b.Fatalf("Failed to create schema: %v", err)
    }
    
    // Populate test data
    if err := populateTestData(db, 1000); err != nil {
        b.Fatalf("Failed to populate data: %v", err)
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        // Use transactions for consistency
        tx, err := db.Begin()
        if err != nil {
            b.Fatalf("Failed to begin transaction: %v", err)
        }
        
        // Perform database operations
        err = performDatabaseWork(tx)
        
        if err != nil {
            tx.Rollback()
            b.Fatalf("Database operation failed: %v", err)
        } else {
            tx.Commit()
        }
    }
}
```

### Part 5: HTTP Service Benchmarking

#### 5.1 HTTP Benchmark Considerations
HTTP service benchmarking involves:

- **Server lifecycle**: Setup, configuration, shutdown
- **Client configuration**: Timeouts, connection pooling
- **Request/response handling**: Serialization, headers, status codes
- **Concurrency**: Multiple clients, connection limits
- **Error handling**: Network failures, timeouts, retries

#### 5.2 Running HTTP Benchmarks

```bash
# Run HTTP operation benchmarks
go test -bench=BenchmarkHTTPOperations -v

# Run with race detection
go test -bench=BenchmarkHTTPOperations -race

# Run with timeout
go test -bench=BenchmarkHTTPOperations -timeout=30s
```

#### 5.3 Expected HTTP Benchmark Output

```
=== RUN   BenchmarkHTTPOperations
=== RUN   BenchmarkHTTPOperations/GetUsers
BenchmarkHTTPOperations/GetUsers-8         	    5000	    250000 ns/op	    1024 B/op	      20 allocs/op
                                           	        	          	         	           	     0.0 error_rate_%
=== RUN   BenchmarkHTTPOperations/CreateUser
BenchmarkHTTPOperations/CreateUser-8       	    2000	    500000 ns/op	    1536 B/op	      25 allocs/op
                                           	        	          	         	           	     0.0 error_rate_%
```

#### 5.4 HTTP Benchmark Implementation

```go
func BenchmarkHTTP(b *testing.B) {
    // Setup test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Simulate processing time
        time.Sleep(time.Millisecond * 10)
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }))
    defer server.Close()
    
    // Configure client
    client := &http.Client{
        Timeout: time.Second * 5,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 100,
        },
    }
    
    b.ResetTimer()
    
    var errors int64
    for i := 0; i < b.N; i++ {
        resp, err := client.Get(server.URL)
        if err != nil {
            errors++
            continue
        }
        resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            errors++
        }
    }
    
    // Report error rate
    errorRate := float64(errors) / float64(b.N) * 100
    b.ReportMetric(errorRate, "error_rate_%")
}
```

### Part 6: Concurrency and Scaling Benchmarks

#### 6.1 Concurrency Benchmark Patterns
Concurrency benchmarking focuses on:

- **Goroutine scaling**: Performance vs. goroutine count
- **Channel throughput**: Buffer sizes, blocking behavior
- **Synchronization overhead**: Mutexes, atomic operations
- **Worker pool efficiency**: Queue management, load balancing
- **Resource contention**: Lock contention, memory bandwidth

#### 6.2 Running Concurrency Benchmarks

```bash
# Run concurrency pattern benchmarks
go test -bench=BenchmarkConcurrencyPatterns -v

# Run channel throughput benchmarks
go test -bench=BenchmarkChannelThroughput -v

# Run with different GOMAXPROCS
GOMAXPROCS=1 go test -bench=BenchmarkConcurrencyPatterns
GOMAXPROCS=4 go test -bench=BenchmarkConcurrencyPatterns
GOMAXPROCS=8 go test -bench=BenchmarkConcurrencyPatterns
```

#### 6.3 Expected Concurrency Output

```
BenchmarkConcurrencyPatterns/Workers1-8    	     500	   2400000 ns/op	    1024 B/op	      50 allocs/op
                                           	        	          	         	           	  1000.0 items_per_worker
BenchmarkConcurrencyPatterns/Workers2-8    	    1000	   1200000 ns/op	    1024 B/op	      50 allocs/op
                                           	        	          	         	           	   500.0 items_per_worker
BenchmarkConcurrencyPatterns/Workers4-8    	    2000	    600000 ns/op	    1024 B/op	      50 allocs/op
                                           	        	          	         	           	   250.0 items_per_worker
BenchmarkConcurrencyPatterns/Workers8-8    	    4000	    300000 ns/op	    1024 B/op	      50 allocs/op
                                           	        	          	         	           	   125.0 items_per_worker
```

### Part 7: Memory and Resource Monitoring

#### 7.1 Resource Monitoring Techniques
Advanced benchmarks monitor system resources:

- **Memory usage**: Heap growth, GC pressure, allocation patterns
- **CPU utilization**: Core usage, context switches
- **Goroutine count**: Goroutine leaks, scheduling overhead
- **File descriptors**: Connection limits, resource leaks
- **Network usage**: Bandwidth, connection counts

#### 7.2 Running Resource Monitoring Benchmarks

```bash
# Run memory allocation benchmarks
go test -bench=BenchmarkMemoryAllocation -benchmem -v

# Run GC impact benchmarks
go test -bench=BenchmarkGarbageCollection -benchmem -v

# Run resource utilization benchmarks
go test -bench=BenchmarkResourceUtilization -v

# Generate memory profile
go test -bench=BenchmarkMemoryAllocation -memprofile=mem.prof
go tool pprof mem.prof
```

#### 7.3 Expected Resource Monitoring Output

```
BenchmarkMemoryAllocation/Size100-8        	   10000	    120000 ns/op	   12800 B/op	     110 allocs/op
BenchmarkMemoryAllocation/Size1000-8       	    1000	   1200000 ns/op	  128000 B/op	    1100 allocs/op
BenchmarkMemoryAllocation/Size10000-8      	     100	  12000000 ns/op	 1280000 B/op	   11000 allocs/op

BenchmarkResourceUtilization-8             	   50000	     24000 ns/op	    1024 B/op	       1 allocs/op
                                           	        	          	         	           	  51200.0 memory_growth_bytes
                                           	        	          	         	           	      2.0 gc_cycles
                                           	        	          	         	           	  50000.0 malloc_count
                                           	        	          	         	           	      8.0 goroutines
                                           	        	          	         	           	      8.0 cpu_count
```

### Part 8: Performance Regression Detection

#### 8.1 Regression Detection Strategies
Automated regression detection involves:

- **Baseline establishment**: Historical performance data
- **Statistical analysis**: Confidence intervals, significance testing
- **Threshold configuration**: Acceptable performance degradation
- **Trend analysis**: Long-term performance trends
- **Alert mechanisms**: Automated notifications

#### 8.2 Running Regression Detection

```bash
# Run regression detection benchmarks
go test -bench=BenchmarkRegressionDetection -v

# Generate baseline data
go test -bench=BenchmarkComparative -count=10 > baseline.txt

# Generate current data (after changes)
go test -bench=BenchmarkComparative -count=10 > current.txt

# Compare with benchstat
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt current.txt
```

#### 8.3 Expected Regression Output

```
=== RUN   BenchmarkRegressionDetection
=== RUN   BenchmarkRegressionDetection/RegressionAnalysis
BenchmarkRegressionDetection/RegressionAnalysis-8    	 1000000	      1200 ns/op
                                                     	        	          	           	      2.0 regressions_found
--- BENCH: BenchmarkRegressionDetection/RegressionAnalysis-8
    advanced_benchmarks_test.go:890: Regression 1: Operation1: time regression 20.00%
    advanced_benchmarks_test.go:890: Regression 2: Operation2: memory regression 25.00%
```

#### 8.4 benchstat Output Example

```
name                    old time/op    new time/op    delta
Comparative/Iterative/Size10-8    120ns ± 2%    125ns ± 3%   +4.17%  (p=0.008 n=10+10)
Comparative/Iterative/Size20-8    240ns ± 1%    250ns ± 2%   +4.17%  (p=0.000 n=10+10)
Comparative/Recursive/Size10-8   1.20µs ± 3%   1.30µs ± 4%   +8.33%  (p=0.000 n=10+10)

name                    old alloc/op   new alloc/op   delta
Comparative/Iterative/Size10-8     0.00B          0.00B          ~     (all equal)
Comparative/Iterative/Size20-8     0.00B          0.00B          ~     (all equal)
Comparative/Recursive/Size10-8     0.00B          0.00B          ~     (all equal)

name                    old allocs/op  new allocs/op  delta
Comparative/Iterative/Size10-8      0.00           0.00          ~     (all equal)
Comparative/Iterative/Size20-8      0.00           0.00          ~     (all equal)
Comparative/Recursive/Size10-8      0.00           0.00          ~     (all equal)
```

### Part 9: Parametric and Contextual Benchmarking

#### 9.1 Parametric Benchmarking
Parametric benchmarks test multiple parameter combinations:

- **Input size scaling**: Performance vs. data size
- **Configuration variations**: Different algorithm parameters
- **Environment conditions**: Various system configurations
- **Workload patterns**: Different usage scenarios

#### 9.2 Running Parametric Benchmarks

```bash
# Run parametric benchmarks
go test -bench=BenchmarkParametric -v

# Run scaling analysis
go test -bench=BenchmarkScaling -v

# Run contextual benchmarks
go test -bench=BenchmarkContextual -v
```

#### 9.3 Expected Parametric Output

```
BenchmarkParametric/Combo0_algorithm_iterative_size_100_workers_1-8    	   10000	    120000 ns/op
                                                                       	        	          	           	   100.0 param_size
                                                                       	        	          	           	     1.0 param_workers
BenchmarkParametric/Combo1_algorithm_iterative_size_100_workers_2-8    	   20000	     60000 ns/op
                                                                       	        	          	           	   100.0 param_size
                                                                       	        	          	           	     2.0 param_workers
```

#### 9.4 Contextual Benchmarking

```go
func BenchmarkContextual(b *testing.B) {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
    defer cancel()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        select {
        case <-ctx.Done():
            b.Fatalf("Context expired: %v", ctx.Err())
        default:
            // Perform work with context
            err := performWorkWithContext(ctx)
            if err != nil {
                b.Fatalf("Work failed: %v", err)
            }
        }
    }
}
```

### Part 10: Production-Ready Benchmarking

#### 10.1 Production Benchmarking Considerations
Production-ready benchmarks require:

- **Environment isolation**: Consistent test conditions
- **Data management**: Realistic test datasets
- **Security**: No sensitive data exposure
- **Scalability**: Benchmarks that scale with system size
- **Automation**: CI/CD integration
- **Reporting**: Actionable performance insights

#### 10.2 CI/CD Integration

```yaml
# .github/workflows/benchmarks.yml
name: Performance Benchmarks

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run Benchmarks
      run: |
        cd 01-performance-fundamentals/04-benchmarking-best-practices/exercise-04-03
        go test -bench=. -benchmem -count=5 > benchmarks.txt
    
    - name: Compare with Baseline
      run: |
        # Download baseline from previous run
        # Compare with benchstat
        # Report regressions
    
    - name: Upload Results
      uses: actions/upload-artifact@v3
      with:
        name: benchmark-results
        path: benchmarks.txt
```

#### 10.3 Benchmark Configuration

```go
// benchmark_config.go
type BenchmarkConfig struct {
    Environment   string        `json:"environment"`
    Duration      time.Duration `json:"duration"`
    Iterations    int           `json:"iterations"`
    SkipSlow      bool          `json:"skip_slow"`
    SkipMemory    bool          `json:"skip_memory"`
    SkipNetwork   bool          `json:"skip_network"`
    SkipDatabase  bool          `json:"skip_database"`
    MinCPUs       int           `json:"min_cpus"`
    MinMemory     int64         `json:"min_memory"`
    OutputFormat  string        `json:"output_format"`
    ProfileTypes  []string      `json:"profile_types"`
}

func LoadConfig() (*BenchmarkConfig, error) {
    configFile := os.Getenv("BENCHMARK_CONFIG")
    if configFile == "" {
        configFile = "benchmark_config.json"
    }
    
    data, err := os.ReadFile(configFile)
    if err != nil {
        return getDefaultConfig(), nil // Use defaults if no config
    }
    
    var config BenchmarkConfig
    err = json.Unmarshal(data, &config)
    return &config, err
}
```

## Advanced Techniques and Best Practices

### 1. Statistical Analysis

#### 1.1 Using benchstat for Statistical Comparison

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks multiple times
go test -bench=BenchmarkComparative -count=10 > old.txt
# Make changes to code
go test -bench=BenchmarkComparative -count=10 > new.txt

# Statistical comparison
benchstat old.txt new.txt

# Filter specific benchmarks
benchstat -filter="Iterative" old.txt new.txt

# Different output formats
benchstat -html old.txt new.txt > comparison.html
benchstat -csv old.txt new.txt > comparison.csv
```

#### 1.2 Understanding Statistical Significance

```
name                    old time/op    new time/op    delta
Iterative/Size10-8        120ns ± 2%     125ns ± 3%   +4.17%  (p=0.008 n=10+10)
Iterative/Size20-8        240ns ± 1%     250ns ± 2%   +4.17%  (p=0.000 n=10+10)
Recursive/Size10-8       1.20µs ± 3%    1.30µs ± 4%   +8.33%  (p=0.000 n=10+10)
```

**Key Metrics:**
- **±%**: Standard deviation as percentage of mean
- **delta**: Percentage change between old and new
- **p-value**: Statistical significance (p < 0.05 = significant)
- **n=10+10**: Number of samples in each group

### 2. Profiling Integration

#### 2.1 Automated Profile Generation

```bash
# Generate all profile types
go test -bench=BenchmarkCustomMetrics -cpuprofile=cpu.prof -memprofile=mem.prof -blockprofile=block.prof -mutexprofile=mutex.prof

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
go tool pprof block.prof
go tool pprof mutex.prof

# Generate execution trace
go test -bench=BenchmarkCustomMetrics -trace=trace.out
go tool trace trace.out
```

#### 2.2 Profile Analysis Commands

```bash
# CPU profile analysis
go tool pprof cpu.prof
(pprof) top10
(pprof) list BenchmarkCustomMetrics
(pprof) web
(pprof) png > cpu_profile.png

# Memory profile analysis
go tool pprof -alloc_space mem.prof
(pprof) top10
(pprof) list AllocateSlices
(pprof) web

# Block profile analysis
go tool pprof block.prof
(pprof) top10
(pprof) list
```

### 3. Continuous Performance Monitoring

#### 3.1 Performance Dashboard

```go
// performance_dashboard.go
type PerformanceDashboard struct {
    metrics map[string][]BenchmarkResult
    mu      sync.RWMutex
}

func (pd *PerformanceDashboard) RecordBenchmark(result BenchmarkResult) {
    pd.mu.Lock()
    defer pd.mu.Unlock()
    
    pd.metrics[result.Name] = append(pd.metrics[result.Name], result)
    
    // Keep only last 100 results
    if len(pd.metrics[result.Name]) > 100 {
        pd.metrics[result.Name] = pd.metrics[result.Name][1:]
    }
}

func (pd *PerformanceDashboard) GetTrend(benchmarkName string) []BenchmarkResult {
    pd.mu.RLock()
    defer pd.mu.RUnlock()
    
    results := pd.metrics[benchmarkName]
    trend := make([]BenchmarkResult, len(results))
    copy(trend, results)
    return trend
}

func (pd *PerformanceDashboard) DetectAnomalies(benchmarkName string, threshold float64) []string {
    trend := pd.GetTrend(benchmarkName)
    if len(trend) < 10 {
        return nil // Need more data
    }
    
    // Calculate moving average and detect outliers
    var anomalies []string
    // Implementation details...
    
    return anomalies
}
```

#### 3.2 Alerting System

```go
// alerting.go
type AlertManager struct {
    thresholds map[string]float64
    notifiers  []Notifier
}

type Notifier interface {
    Notify(alert Alert) error
}

type Alert struct {
    BenchmarkName string
    Severity      string
    Message       string
    Timestamp     time.Time
    Metrics       map[string]float64
}

func (am *AlertManager) CheckThresholds(result BenchmarkResult) []Alert {
    var alerts []Alert
    
    // Check time regression
    if threshold, exists := am.thresholds[result.Name+"_time"]; exists {
        // Compare with baseline and check threshold
        // Generate alert if threshold exceeded
    }
    
    // Check memory regression
    if threshold, exists := am.thresholds[result.Name+"_memory"]; exists {
        // Similar logic for memory
    }
    
    return alerts
}
```

### 4. Custom Benchmark Frameworks

#### 4.1 Domain-Specific Benchmark Suite

```go
// custom_benchmark_suite.go
type BenchmarkSuite struct {
    name        string
    benchmarks  []BenchmarkCase
    setup       func() error
    teardown    func() error
    config      *BenchmarkConfig
    results     []BenchmarkResult
}

type BenchmarkCase struct {
    Name        string
    Description string
    Setup       func() error
    Benchmark   func(*testing.B)
    Teardown    func() error
    Tags        []string
    MinDuration time.Duration
    MaxDuration time.Duration
}

func (bs *BenchmarkSuite) Run(t *testing.T) {
    if err := bs.setup(); err != nil {
        t.Fatalf("Suite setup failed: %v", err)
    }
    defer bs.teardown()
    
    for _, benchmark := range bs.benchmarks {
        if bs.shouldSkip(benchmark) {
            continue
        }
        
        t.Run(benchmark.Name, func(t *testing.T) {
            if err := benchmark.Setup(); err != nil {
                t.Fatalf("Benchmark setup failed: %v", err)
            }
            defer benchmark.Teardown()
            
            result := testing.Benchmark(benchmark.Benchmark)
            bs.recordResult(benchmark.Name, result)
        })
    }
}
```

## Exercise Tasks

### Task 1: Custom Metrics Implementation
1. Implement a benchmark that measures custom business metrics
2. Add throughput, latency percentiles, and error rate measurements
3. Create a metrics collector that tracks multiple performance indicators
4. Generate reports showing correlation between different metrics
5. Implement alerting based on custom metric thresholds

### Task 2: Conditional Benchmark System
1. Create a conditional benchmark framework
2. Implement environment-based benchmark selection
3. Add system requirement validation
4. Create configuration-driven benchmark execution
5. Implement benchmark tagging and filtering

### Task 3: Integration Benchmark Suite
1. Build a comprehensive database benchmark suite
2. Create HTTP service benchmarks with realistic workloads
3. Implement cache performance benchmarks
4. Add concurrent access pattern benchmarks
5. Create end-to-end integration benchmarks

### Task 4: Performance Regression Detection
1. Implement automated regression detection
2. Create baseline management system
3. Add statistical significance testing
4. Implement trend analysis and anomaly detection
5. Create automated alerting for performance regressions

### Task 5: Production Benchmark Infrastructure
1. Design a production-ready benchmark framework
2. Implement CI/CD integration
3. Create performance monitoring dashboard
4. Add automated report generation
5. Implement benchmark result storage and analysis

### Task 6: Advanced Analysis Tools
1. Create custom profiling integration
2. Implement scaling analysis tools
3. Add resource utilization monitoring
4. Create comparative analysis framework
5. Implement performance prediction models

## Common Pitfalls and Solutions

### 1. Inconsistent Test Environment
**Problem**: Benchmarks produce inconsistent results due to environmental factors
**Solution**: 
- Use dedicated benchmark machines
- Control for background processes
- Implement environment validation
- Use statistical analysis to detect outliers

### 2. Inadequate Sample Size
**Problem**: High variance in results due to insufficient iterations
**Solution**:
- Use `-count` flag for multiple runs
- Implement statistical significance testing
- Set minimum benchmark duration
- Use benchstat for proper analysis

### 3. Measurement Overhead
**Problem**: Benchmark infrastructure affects measurements
**Solution**:
- Minimize measurement code in hot paths
- Use sampling for expensive metrics
- Separate setup/teardown from measurement
- Profile the benchmark code itself

### 4. Unrealistic Test Data
**Problem**: Benchmarks use artificial data that doesn't reflect production
**Solution**:
- Use production-like datasets
- Implement realistic data generation
- Test with various data distributions
- Include edge cases and error conditions

### 5. Missing Context
**Problem**: Benchmark results lack context for decision making
**Solution**:
- Include system information in results
- Add business context to metrics
- Implement comparative analysis
- Create actionable recommendations

## Expected Learning Outcomes

After completing this exercise, you should be able to:
- Design and implement sophisticated benchmark suites
- Create custom metrics that provide business value
- Build conditional benchmarks that adapt to different environments
- Integrate benchmarking with databases, HTTP services, and external systems
- Implement automated performance regression detection
- Create production-ready benchmarking infrastructure
- Use statistical methods to validate performance improvements
- Build monitoring and alerting systems for performance
- Design scalable benchmark frameworks for large applications

## Next Steps

1. **Real-World Application**: Apply advanced benchmarking to production systems
2. **Performance Engineering**: Integrate benchmarking into development workflow
3. **Monitoring Integration**: Connect benchmarks with observability platforms
4. **Team Training**: Share advanced benchmarking practices across teams
5. **Continuous Improvement**: Establish performance-driven development culture

## Additional Resources

- [Go Testing Package Documentation](https://golang.org/pkg/testing/)
- [benchstat Tool Documentation](https://godoc.org/golang.org/x/perf/cmd/benchstat)
- [Go Performance Tuning](https://github.com/golang/go/wiki/Performance)
- [Statistical Methods for Performance Analysis](https://www.brendangregg.com/blog/2018-02-09/kpis-slis-slos.html)
- [Production Performance Monitoring](https://sre.google/sre-book/monitoring-distributed-systems/)
- [Continuous Profiling Best Practices](https://grafana.com/blog/2022/09/13/continuous-profiling-best-practices/)

This exercise provides a comprehensive foundation for advanced performance analysis and benchmarking in Go, enabling you to build production-ready performance monitoring and optimization systems.