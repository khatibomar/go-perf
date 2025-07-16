# Exercise 03-01: Basic CPU Profiling Analysis

## Overview
This exercise demonstrates how to use Go's built-in pprof profiler to identify CPU bottlenecks and optimize performance. We'll profile CPU-intensive functions with deliberate inefficiencies and then analyze the results.

## Running the Profiling

### Step 1: Generate CPU Profile
```bash
# Run benchmarks with CPU profiling
go test -bench=. -cpuprofile=cpu.prof -benchtime=10s

# Or run specific benchmark
go test -bench=BenchmarkCPUIntensiveWorkload -cpuprofile=cpu.prof -benchtime=10s
```

### Step 2: Analyze the Profile
```bash
# Interactive analysis
go tool pprof cpu.prof

# Web interface (recommended)
go tool pprof -http=:8080 cpu.prof
```

### Step 3: Generate Reports
```bash
# Top functions by CPU usage
go tool pprof -top cpu.prof

# List specific function
go tool pprof -list=fibonacciBad cpu.prof

# Generate call graph
go tool pprof -svg cpu.prof > callgraph.svg
```

## Expected Profile Analysis Results

### 1. Top CPU Consumers
When you run the profile analysis, you should see output similar to:

```
(pprof) top
Showing nodes accounting for 15.23s, 95.12% of 16.01s total
Dropped 45 nodes (cum <= 0.08s)
      flat  flat%   sum%        cum   cum%
     8.45s 52.78% 52.78%      8.45s 52.78%  main.fibonacciBad
     2.34s 14.62% 67.40%      2.34s 14.62%  main.bubbleSortBad
     1.89s 11.81% 79.21%      1.89s 11.81%  main.matrixMultiplyBad
     1.23s  7.68% 86.89%      1.23s  7.68%  main.computeExpensiveMathBad
     0.87s  5.43% 92.32%      0.87s  5.43%  main.processTextBad
     0.45s  2.81% 95.13%      0.45s  2.81%  main.findDuplicatesBad
```

### 2. Function-Level Analysis

#### Fibonacci Function (Expected Hotspot)
```bash
(pprof) list fibonacciBad
```

Expected output:
```
Total: 16.01s
SHOWING NODES ACCOUNTING FOR 8.45s, 52.78% of 16.01s total
----------------------------------------------------------+-------------
      flat  flat%   sum%        cum   cum%   calls calls% + context
----------------------------------------------------------+-------------
                                             8.45s   100% |   main.fibonacciBad
     8.45s 52.78% 52.78%      8.45s 52.78%                | main.fibonacciBad
                                             7.23s 85.56% |   main.fibonacciBad (recursive)
----------------------------------------------------------+-------------
```

**Analysis**: The recursive Fibonacci function dominates CPU usage due to exponential time complexity O(2^n).

#### Bubble Sort Function
```bash
(pprof) list bubbleSortBad
```

**Analysis**: Shows O(n²) time complexity with nested loops causing high CPU usage.

#### Matrix Multiplication
```bash
(pprof) list matrixMultiplyBad
```

**Analysis**: Poor cache locality due to column-first access pattern causes cache misses.

### 3. Call Graph Analysis
The call graph will show:
- Function call relationships
- Time spent in each function
- Recursive call patterns (especially visible in Fibonacci)

## Key Profiling Insights

### 1. Algorithmic Issues
- **Fibonacci**: Exponential time complexity O(2^n)
- **Bubble Sort**: Quadratic time complexity O(n²)
- **Prime Finding**: Inefficient trial division
- **Duplicate Finding**: Nested loops creating O(n³) complexity

### 2. Memory Access Patterns
- **Matrix Multiplication**: Poor cache locality
- **String Operations**: Excessive allocations

### 3. Function Call Overhead
- **Math Operations**: Repeated expensive function calls
- **String Processing**: Multiple string operations

## Optimization Strategies

### 1. Algorithm Improvements
```go
// Before: O(2^n) recursive Fibonacci
func fibonacciBad(n int) int {
    if n <= 1 { return n }
    return fibonacciBad(n-1) + fibonacciBad(n-2)
}

// After: O(n) iterative Fibonacci
func fibonacciGood(n int) int {
    if n <= 1 { return n }
    a, b := 0, 1
    for i := 2; i <= n; i++ {
        a, b = b, a+b
    }
    return b
}
```

### 2. Data Structure Optimization
```go
// Before: O(n²) bubble sort
func bubbleSortBad(arr []int) []int {
    // Nested loops...
}

// After: O(n log n) built-in sort
func sortGood(arr []int) []int {
    data := make([]int, len(arr))
    copy(data, arr)
    sort.Ints(data)
    return data
}
```

### 3. Cache Optimization
```go
// Before: Poor cache locality (column-first)
for j := 0; j < m; j++ {
    for i := 0; i < n; i++ {
        for k := 0; k < p; k++ {
            result[i][j] += a[i][k] * b[k][j]
        }
    }
}

// After: Better cache locality (row-first)
for i := 0; i < n; i++ {
    for k := 0; k < p; k++ {
        for j := 0; j < m; j++ {
            result[i][j] += a[i][k] * b[k][j]
        }
    }
}
```

## Comparative Analysis

### Running Before/After Comparison
```bash
# Profile original functions
go test -bench=BenchmarkFibonacciBad -cpuprofile=cpu_bad.prof

# Profile optimized functions (add benchmarks for optimized versions)
go test -bench=BenchmarkFibonacciGood -cpuprofile=cpu_good.prof

# Compare profiles
go tool pprof -base cpu_bad.prof cpu_good.prof
```

### Expected Performance Improvements
- **Fibonacci**: 1000x+ improvement (exponential to linear)
- **Sorting**: 100x+ improvement (O(n²) to O(n log n))
- **Prime Finding**: 10x+ improvement (trial division to sieve)
- **Matrix Multiplication**: 2-5x improvement (cache optimization)
- **String Processing**: 5-10x improvement (reduced allocations)

## Common pprof Commands

### Interactive Mode Commands
```
(pprof) top           # Show top functions
(pprof) top10         # Show top 10 functions
(pprof) list main.fibonacciBad  # Show source code with annotations
(pprof) web           # Generate web view
(pprof) svg           # Generate SVG call graph
(pprof) peek main.fibonacciBad  # Show callers and callees
(pprof) traces        # Show sample traces
(pprof) help          # Show all commands
```

### Command Line Options
```bash
# Different output formats
go tool pprof -top cpu.prof
go tool pprof -list=main.fibonacciBad cpu.prof
go tool pprof -svg cpu.prof > graph.svg
go tool pprof -png cpu.prof > graph.png
go tool pprof -pdf cpu.prof > graph.pdf

# Web interface
go tool pprof -http=:8080 cpu.prof

# Focus on specific functions
go tool pprof -focus=fibonacci cpu.prof
go tool pprof -ignore=runtime cpu.prof
```

## Best Practices

### 1. Profiling Environment
- Use consistent hardware and environment
- Minimize background processes
- Run benchmarks for sufficient duration (10s+)
- Use multiple runs for statistical significance

### 2. Profile Interpretation
- Focus on "flat" time for direct optimization targets
- Use "cum" time to understand call hierarchies
- Look for unexpected hot spots
- Validate optimizations with before/after profiles

### 3. Optimization Workflow
1. Profile to identify bottlenecks
2. Understand why the bottleneck exists
3. Implement targeted optimization
4. Re-profile to validate improvement
5. Repeat for next biggest bottleneck

## Exercise Tasks

### Task 1: Basic Profiling
1. Run the CPU-intensive benchmarks with profiling
2. Analyze the profile using `go tool pprof`
3. Identify the top 3 CPU consumers
4. Document your findings

### Task 2: Function Analysis
1. Use `list` command to examine hot functions
2. Understand why each function is expensive
3. Identify optimization opportunities

### Task 3: Optimization Validation
1. Implement optimized versions of hot functions
2. Create benchmarks for optimized versions
3. Profile optimized code
4. Compare before/after performance

### Task 4: Advanced Analysis
1. Use web interface for visual analysis
2. Generate call graphs
3. Analyze different benchmark scenarios
4. Document optimization strategies

## Expected Learning Outcomes

After completing this exercise, you should be able to:
- Generate CPU profiles from Go benchmarks
- Use pprof tools for profile analysis
- Interpret profile data to identify bottlenecks
- Understand the relationship between algorithms and CPU usage
- Validate optimizations using profiling data
- Use various pprof output formats effectively

## Next Steps

Proceed to Exercise 03-02 to learn about memory profiling and heap analysis.