# Exercise 03-03: Live Application Profiling

## Overview
This exercise demonstrates how to profile live Go applications using pprof's HTTP endpoints. We'll analyze a running web server under various load conditions to identify performance bottlenecks in real-world scenarios.

## Setup and Running the Application

### Step 1: Start the Web Server
```bash
# Navigate to exercise directory
cd 01-performance-fundamentals/03-basic-profiling/exercise-03-03

# Start the web server
go run web_server.go
```

The server will start on `http://localhost:8080` with the following endpoints:
- `/cpu` - CPU-intensive operations
- `/memory` - Memory-intensive operations
- `/inefficient` - Demonstrates inefficient patterns
- `/database` - Simulates database operations
- `/cache` - Cache operations with potential issues
- `/concurrent` - Concurrent operations
- `/leaky` - Memory leak simulation
- `/stats` - Server statistics
- `/debug/pprof/` - pprof profiling endpoints

### Step 2: Generate Load (Optional)
```bash
# In a separate terminal, run load tests
go run load_test.go -scenario=mixed -duration=5m

# Or run specific scenarios:
go run load_test.go -scenario=cpu -duration=2m
go run load_test.go -scenario=memory -duration=2m
go run load_test.go -scenario=leaky -duration=10m
```

## Live Profiling Techniques

### 1. CPU Profiling of Live Application

#### Basic CPU Profile Collection
```bash
# Collect 30-second CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Save profile to file
curl -o cpu.prof "http://localhost:8080/debug/pprof/profile?seconds=30"
go tool pprof cpu.prof

# Web interface
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=30
```

#### Analyzing CPU Profiles Under Load
1. Start load test: `go run load_test.go -scenario=cpu`
2. While load is running, collect profile:
   ```bash
   go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=60
   ```
3. Analyze the flame graph and top functions
4. Compare with baseline (no load) profile

### 2. Memory Profiling of Live Application

#### Heap Profile Analysis
```bash
# Current heap usage
go tool pprof http://localhost:8080/debug/pprof/heap

# Allocation profile (since server start)
go tool pprof http://localhost:8080/debug/pprof/allocs

# Web interface for heap analysis
go tool pprof -http=:8082 http://localhost:8080/debug/pprof/heap
```

#### Memory Leak Detection
1. Start server and take baseline heap profile:
   ```bash
   curl -o heap_baseline.prof "http://localhost:8080/debug/pprof/heap"
   ```

2. Run leaky scenario:
   ```bash
   go run load_test.go -scenario=leaky -duration=10m
   ```

3. Take heap profile after load:
   ```bash
   curl -o heap_after_load.prof "http://localhost:8080/debug/pprof/heap"
   ```

4. Compare profiles:
   ```bash
   go tool pprof -base heap_baseline.prof heap_after_load.prof
   ```

### 3. Goroutine Analysis

#### Goroutine Profile Collection
```bash
# Current goroutines
go tool pprof http://localhost:8080/debug/pprof/goroutine

# Web interface
go tool pprof -http=:8083 http://localhost:8080/debug/pprof/goroutine

# Text output
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

#### Analyzing Goroutine Leaks
1. Check baseline goroutine count:
   ```bash
   curl "http://localhost:8080/stats" | jq '.goroutines'
   ```

2. Run concurrent load:
   ```bash
   go run load_test.go -scenario=mixed -concurrency=50
   ```

3. Monitor goroutine growth:
   ```bash
   # Check periodically
   watch -n 5 'curl -s "http://localhost:8080/stats" | jq ".goroutines"'
   ```

4. Analyze goroutine profile:
   ```bash
   go tool pprof -http=:8083 http://localhost:8080/debug/pprof/goroutine
   ```

### 4. Block and Mutex Profiling

#### Enable Block Profiling
```bash
# Enable block profiling (add to web_server.go main function)
runtime.SetBlockProfileRate(1)

# Collect block profile
go tool pprof http://localhost:8080/debug/pprof/block
```

#### Enable Mutex Profiling
```bash
# Enable mutex profiling (add to web_server.go main function)
runtime.SetMutexProfileFraction(1)

# Collect mutex profile
go tool pprof http://localhost:8080/debug/pprof/mutex
```

## Analysis Scenarios

### Scenario 1: CPU Bottleneck Analysis

#### Setup
1. Start server: `go run web_server.go`
2. Generate CPU load: `go run load_test.go -scenario=cpu -duration=5m`
3. Profile during load: `go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=60`

#### Expected Findings
- High CPU usage in `performCPUIntensiveTask`
- Hot spots in mathematical computations
- Function call overhead in nested loops

#### Analysis Commands
```bash
# Top CPU consumers
(pprof) top

# Focus on specific function
(pprof) list performCPUIntensiveTask

# Call graph
(pprof) web
```

### Scenario 2: Memory Allocation Analysis

#### Setup
1. Start server: `go run web_server.go`
2. Generate memory load: `go run load_test.go -scenario=memory -duration=3m`
3. Profile allocations: `go tool pprof -http=:8082 http://localhost:8080/debug/pprof/allocs`

#### Expected Findings
- High allocation rate in `performMemoryIntensiveTask`
- Interface boxing allocations
- Slice growth reallocations
- String concatenation overhead

#### Analysis Commands
```bash
# Top allocators
(pprof) top -cum

# Allocation details
(pprof) list performMemoryIntensiveTask

# Compare alloc_space vs alloc_objects
(pprof) top -alloc_space
(pprof) top -alloc_objects
```

### Scenario 3: Memory Leak Detection

#### Setup
1. Start server and take baseline: `curl -o baseline.prof "http://localhost:8080/debug/pprof/heap"`
2. Run leaky load: `go run load_test.go -scenario=leaky -duration=10m`
3. Take final profile: `curl -o final.prof "http://localhost:8080/debug/pprof/heap"`
4. Compare: `go tool pprof -base baseline.prof final.prof`

#### Expected Findings
- Growing heap usage in `createLeakyStructures`
- Accumulating data in server's data map
- References preventing garbage collection

#### Analysis Commands
```bash
# Show growth
(pprof) top

# Examine leak sources
(pprof) list createLeakyStructures

# Check inuse vs alloc
go tool pprof -inuse_space final.prof
go tool pprof -alloc_space final.prof
```

### Scenario 4: Concurrency Analysis

#### Setup
1. Start server: `go run web_server.go`
2. Generate concurrent load: `go run load_test.go -scenario=mixed -concurrency=100`
3. Monitor goroutines: `go tool pprof -http=:8083 http://localhost:8080/debug/pprof/goroutine`

#### Expected Findings
- Goroutine count correlation with load
- Potential goroutine leaks
- Blocking operations
- Lock contention

#### Analysis Commands
```bash
# Goroutine analysis
(pprof) top
(pprof) traces

# Check for blocks (if enabled)
go tool pprof http://localhost:8080/debug/pprof/block

# Check for mutex contention (if enabled)
go tool pprof http://localhost:8080/debug/pprof/mutex
```

## Real-time Monitoring

### 1. Continuous Profiling Setup

#### Script for Periodic Profile Collection
```bash
#!/bin/bash
# collect_profiles.sh

BASE_URL="http://localhost:8080"
OUTPUT_DIR="profiles/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$OUTPUT_DIR"

echo "Collecting profiles to $OUTPUT_DIR"

# Collect different profile types
curl -o "$OUTPUT_DIR/cpu.prof" "$BASE_URL/debug/pprof/profile?seconds=30"
curl -o "$OUTPUT_DIR/heap.prof" "$BASE_URL/debug/pprof/heap"
curl -o "$OUTPUT_DIR/allocs.prof" "$BASE_URL/debug/pprof/allocs"
curl -o "$OUTPUT_DIR/goroutine.prof" "$BASE_URL/debug/pprof/goroutine"

# Collect server stats
curl -s "$BASE_URL/stats" > "$OUTPUT_DIR/stats.json"

echo "Profiles collected in $OUTPUT_DIR"
```

#### Automated Monitoring
```bash
# Run every 5 minutes
watch -n 300 './collect_profiles.sh'

# Or use cron for production
# */5 * * * * /path/to/collect_profiles.sh
```

### 2. Performance Metrics Dashboard

#### Key Metrics to Monitor
```bash
# Memory usage
curl -s "http://localhost:8080/stats" | jq '.memory_stats.alloc_mb'

# Request rate
curl -s "http://localhost:8080/stats" | jq '.requests'

# Goroutine count
curl -s "http://localhost:8080/stats" | jq '.goroutines'

# GC statistics
curl -s "http://localhost:8080/stats" | jq '.memory_stats.num_gc'
```

#### Simple Monitoring Script
```bash
#!/bin/bash
# monitor.sh

while true; do
    STATS=$(curl -s "http://localhost:8080/stats")
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
    MEMORY=$(echo "$STATS" | jq -r '.memory_stats.alloc_mb')
    GOROUTINES=$(echo "$STATS" | jq -r '.goroutines')
    REQUESTS=$(echo "$STATS" | jq -r '.requests')
    
    echo "$TIMESTAMP - Memory: ${MEMORY}MB, Goroutines: $GOROUTINES, Requests: $REQUESTS"
    sleep 10
done
```

## Advanced Profiling Techniques

### 1. Differential Profiling

#### Before/After Comparison
```bash
# Baseline profile
curl -o before.prof "http://localhost:8080/debug/pprof/heap"

# Generate load
go run load_test.go -scenario=memory -duration=2m

# After profile
curl -o after.prof "http://localhost:8080/debug/pprof/heap"

# Compare
go tool pprof -base before.prof after.prof
```

#### Time-series Analysis
```bash
# Collect profiles over time
for i in {1..10}; do
    curl -o "heap_$i.prof" "http://localhost:8080/debug/pprof/heap"
    sleep 60
done

# Analyze growth patterns
go tool pprof -base heap_1.prof heap_10.prof
```

### 2. Custom Profile Collection

#### Programmatic Profile Collection
```go
// Add to web_server.go for custom profiling
func collectCustomProfile() {
    f, err := os.Create("custom_cpu.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
    
    // Run specific workload
    performCPUIntensiveTask(1000000)
}
```

### 3. Production Profiling Best Practices

#### Safe Profiling in Production
```bash
# Limit profile duration
go tool pprof "http://localhost:8080/debug/pprof/profile?seconds=10"

# Use sampling for heap profiles
go tool pprof "http://localhost:8080/debug/pprof/heap?gc=1"

# Rate-limited profiling
# Only profile during low-traffic periods
```

#### Profile Storage and Analysis
```bash
# Compress profiles for storage
gzip *.prof

# Batch analysis
for prof in *.prof.gz; do
    echo "Analyzing $prof"
    gunzip -c "$prof" | go tool pprof -top -
done
```

## Common Issues and Solutions

### 1. High Memory Usage

#### Symptoms
- Increasing heap size over time
- High allocation rate
- Frequent GC cycles

#### Investigation
```bash
# Check heap growth
go tool pprof -inuse_space http://localhost:8080/debug/pprof/heap

# Check allocation patterns
go tool pprof -alloc_space http://localhost:8080/debug/pprof/allocs

# Look for leaks
go tool pprof -base baseline.prof current.prof
```

### 2. High CPU Usage

#### Symptoms
- High CPU utilization
- Slow response times
- Hot functions in profiles

#### Investigation
```bash
# Identify hot functions
go tool pprof -top http://localhost:8080/debug/pprof/profile

# Analyze call patterns
go tool pprof -web http://localhost:8080/debug/pprof/profile

# Check for inefficient algorithms
go tool pprof -list=functionName http://localhost:8080/debug/pprof/profile
```

### 3. Goroutine Leaks

#### Symptoms
- Increasing goroutine count
- Memory growth
- Resource exhaustion

#### Investigation
```bash
# Check goroutine count
curl "http://localhost:8080/debug/pprof/goroutine?debug=1"

# Analyze goroutine profiles
go tool pprof http://localhost:8080/debug/pprof/goroutine

# Look for blocked goroutines
go tool pprof http://localhost:8080/debug/pprof/block
```

### 4. Lock Contention

#### Symptoms
- High latency
- Poor scalability
- Blocked goroutines

#### Investigation
```bash
# Enable mutex profiling first
# runtime.SetMutexProfileFraction(1)

# Analyze mutex contention
go tool pprof http://localhost:8080/debug/pprof/mutex

# Check block profile
go tool pprof http://localhost:8080/debug/pprof/block
```

## Exercise Tasks

### Task 1: Basic Live Profiling
1. Start the web server
2. Generate load using the load tester
3. Collect CPU and memory profiles during load
4. Analyze profiles to identify bottlenecks
5. Document findings

### Task 2: Memory Leak Detection
1. Run the leaky scenario for 10 minutes
2. Collect heap profiles before, during, and after
3. Use differential profiling to identify leaks
4. Analyze the leak sources
5. Propose fixes

### Task 3: Concurrency Analysis
1. Run high-concurrency load tests
2. Monitor goroutine count and patterns
3. Collect goroutine profiles
4. Identify potential goroutine leaks
5. Analyze blocking operations

### Task 4: Performance Optimization
1. Identify the top 3 performance bottlenecks
2. Implement optimizations
3. Compare before/after profiles
4. Measure performance improvements
5. Document optimization strategies

### Task 5: Production Monitoring Setup
1. Create automated profile collection scripts
2. Set up continuous monitoring
3. Define alerting thresholds
4. Create a performance dashboard
5. Document operational procedures

## Expected Learning Outcomes

After completing this exercise, you should be able to:
- Profile live Go applications using pprof HTTP endpoints
- Collect and analyze different types of profiles (CPU, memory, goroutine)
- Detect memory leaks and goroutine leaks in running applications
- Use differential profiling to understand performance changes
- Set up continuous profiling for production monitoring
- Identify and resolve common performance issues
- Implement safe profiling practices for production environments

## Next Steps

Proceed to Exercise 03-04 to learn about advanced profiling techniques including custom profilers, trace analysis, and performance regression detection.