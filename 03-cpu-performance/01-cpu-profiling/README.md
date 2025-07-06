# Chapter 01: CPU Profiling Techniques

**Duration:** 6-8 hours  
**Difficulty:** Intermediate  
**Prerequisites:** Module 01 and 02 completion

## Learning Objectives

By the end of this chapter, you will:
- Understand different types of CPU profiling
- Master Go's built-in CPU profiling tools
- Analyze CPU profiles using pprof and flame graphs
- Implement production-ready CPU profiling
- Identify and optimize CPU bottlenecks

## Chapter Overview

CPU profiling is essential for understanding where your application spends its time and identifying performance bottlenecks. This chapter covers both development and production profiling techniques.

## Theory: CPU Profiling Fundamentals

### What is CPU Profiling?

CPU profiling is the process of collecting data about where a program spends its execution time. It helps identify:
- Hot functions (functions consuming the most CPU time)
- Call paths leading to expensive operations
- Inefficient algorithms or implementations
- Unexpected CPU usage patterns

### Types of CPU Profiling

1. **Sampling Profiling**
   - Periodically samples the call stack
   - Low overhead (~1-5%)
   - Statistical approximation
   - Good for production use

2. **Instrumentation Profiling**
   - Instruments every function call
   - High accuracy
   - Higher overhead
   - Better for development

3. **Hardware Profiling**
   - Uses CPU performance counters
   - Very detailed metrics
   - Platform-specific
   - Requires special tools (perf, Intel VTune)

### Go's CPU Profiling

Go uses sampling-based CPU profiling:
- Samples at 100Hz by default (every 10ms)
- Uses SIGPROF signal on Unix systems
- Minimal runtime overhead
- Built into the runtime

## Tools and Techniques

### 1. Built-in Profiling

```go
import (
    "os"
    "runtime/pprof"
)

// Start CPU profiling
f, err := os.Create("cpu.prof")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

if err := pprof.StartCPUProfile(f); err != nil {
    log.Fatal(err)
}
defer pprof.StopCPUProfile()

// Your application code here
```

### 2. HTTP Profiling Endpoint

```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Access profiles at:
- `http://localhost:6060/debug/pprof/profile?seconds=30`
- `http://localhost:6060/debug/pprof/`

### 3. Test Profiling

```bash
# Profile benchmarks
go test -bench=. -cpuprofile=cpu.prof

# Profile tests
go test -cpuprofile=cpu.prof
```

### 4. Command Line Profiling

```bash
# Analyze profile
go tool pprof cpu.prof

# Web interface
go tool pprof -http=:8080 cpu.prof

# Compare profiles
go tool pprof -base=old.prof new.prof
```

## Profile Analysis

### pprof Commands

```
(pprof) top10          # Top 10 functions by CPU time
(pprof) list main.func # Source code with annotations
(pprof) web           # Generate SVG call graph
(pprof) peek regex    # Functions matching regex
(pprof) disasm func   # Assembly code for function
```

### Understanding Profile Output

```
Showing nodes accounting for 2.50s, 83.33% of 3.00s total
Dropped 45 nodes (cum <= 0.015s)
      flat  flat%   sum%        cum   cum%
     0.89s 29.67% 29.67%      0.89s 29.67%  runtime.mallocgc
     0.45s 15.00% 44.67%      0.45s 15.00%  main.expensiveFunc
     0.34s 11.33% 56.00%      1.20s 40.00%  main.processData
```

- **flat**: Time spent in function itself
- **cum**: Cumulative time (including callees)
- **flat%**: Percentage of total time
- **sum%**: Cumulative percentage

### Flame Graphs

Flame graphs provide visual representation of CPU usage:
- Width represents time spent
- Height shows call stack depth
- Color coding for different functions
- Interactive exploration

## Production Profiling

### Continuous Profiling

```go
type ContinuousProfiler struct {
    interval time.Duration
    duration time.Duration
    storage  ProfileStorage
}

func (cp *ContinuousProfiler) Start() {
    ticker := time.NewTicker(cp.interval)
    defer ticker.Stop()
    
    for range ticker.C {
        cp.collectProfile()
    }
}

func (cp *ContinuousProfiler) collectProfile() {
    var buf bytes.Buffer
    if err := pprof.StartCPUProfile(&buf); err != nil {
        return
    }
    
    time.Sleep(cp.duration)
    pprof.StopCPUProfile()
    
    cp.storage.Store(buf.Bytes())
}
```

### Profiling Best Practices

1. **Profile Duration**
   - Minimum 30 seconds for meaningful data
   - Longer for low-traffic applications
   - Consider workload patterns

2. **Sampling Rate**
   - Default 100Hz is usually sufficient
   - Higher rates for short-lived functions
   - Lower rates to reduce overhead

3. **Environment**
   - Profile in production-like conditions
   - Use representative workloads
   - Consider system load

4. **Storage and Rotation**
   - Implement profile retention policies
   - Compress old profiles
   - Monitor storage usage

## Common CPU Performance Issues

### 1. Inefficient Algorithms

```go
// Bad: O(n²) algorithm
func findDuplicates(items []string) []string {
    var duplicates []string
    for i := 0; i < len(items); i++ {
        for j := i + 1; j < len(items); j++ {
            if items[i] == items[j] {
                duplicates = append(duplicates, items[i])
            }
        }
    }
    return duplicates
}

// Good: O(n) algorithm
func findDuplicatesOptimized(items []string) []string {
    seen := make(map[string]bool)
    var duplicates []string
    
    for _, item := range items {
        if seen[item] {
            duplicates = append(duplicates, item)
        } else {
            seen[item] = true
        }
    }
    return duplicates
}
```

### 2. Excessive String Operations

```go
// Bad: Creates many temporary strings
func buildString(parts []string) string {
    result := ""
    for _, part := range parts {
        result += part
    }
    return result
}

// Good: Uses strings.Builder
func buildStringOptimized(parts []string) string {
    var builder strings.Builder
    for _, part := range parts {
        builder.WriteString(part)
    }
    return builder.String()
}
```

### 3. Reflection Overhead

```go
// Bad: Heavy reflection usage
func processWithReflection(data interface{}) {
    v := reflect.ValueOf(data)
    for i := 0; i < v.NumField(); i++ {
        field := v.Field(i)
        // Process field...
    }
}

// Good: Type-specific processing
func processTyped(data MyStruct) {
    // Direct field access
    processField(data.Field1)
    processField(data.Field2)
    // ...
}
```

## Chapter Exercises

### Exercise 01-01: Basic CPU Profiling Setup
**Objective:** Set up CPU profiling for a simple application

**Tasks:**
1. Create a CPU-intensive application
2. Add CPU profiling instrumentation
3. Generate and analyze profiles
4. Identify performance bottlenecks

### Exercise 01-02: Flame Graph Analysis
**Objective:** Use flame graphs to visualize CPU usage

**Tasks:**
1. Generate flame graphs from CPU profiles
2. Analyze call stack patterns
3. Identify hot paths
4. Compare before/after optimizations

### Exercise 01-03: Production Profiling Integration
**Objective:** Implement production-ready profiling

**Tasks:**
1. Add HTTP profiling endpoints
2. Implement continuous profiling
3. Create profile storage system
4. Build monitoring dashboard

### Exercise 01-04: Profile-Guided Optimization
**Objective:** Use profiles to guide optimization efforts

**Tasks:**
1. Profile a complex application
2. Identify optimization opportunities
3. Implement optimizations
4. Validate improvements with new profiles

## Key Takeaways

1. **Profile First** - Always profile before optimizing
2. **Focus on Hot Paths** - Optimize functions with highest CPU usage
3. **Use Representative Workloads** - Profile with realistic data
4. **Continuous Monitoring** - Implement ongoing profiling in production
5. **Validate Optimizations** - Measure improvements objectively

## Next Steps

After completing this chapter:
1. Complete all exercises
2. Practice with your own applications
3. Set up profiling in your development workflow
4. Move to [Chapter 02: Algorithmic Optimization](../02-algorithmic-optimization/README.md)

## Resources

### Documentation
- [Go pprof Package](https://golang.org/pkg/runtime/pprof/)
- [Profiling Go Programs](https://blog.golang.org/pprof)
- [pprof User Guide](https://github.com/google/pprof/blob/master/doc/README.md)

### Tools
- [FlameGraph](https://github.com/brendangregg/FlameGraph)
- [go-torch](https://github.com/uber/go-torch) (deprecated but educational)
- [pprof Web UI](https://github.com/google/pprof)

### Videos
- ["Profiling Go Programs" by Brad Fitzpatrick](https://www.youtube.com/watch?v=xxDZuPEgbBU)
- ["Go tool pprof" by JBD](https://www.youtube.com/watch?v=N3PWzBeLX2M)

---

*Chapter 01 of Module 03: CPU Performance*