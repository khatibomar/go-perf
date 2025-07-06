# Exercise 01-01: Basic CPU Profiling Setup

**Difficulty:** Beginner  
**Estimated Time:** 45-60 minutes  
**Prerequisites:** Basic Go knowledge, Module 01-02 completion

## Objective

Learn to set up and use CPU profiling in Go applications. You'll create a CPU-intensive application, instrument it for profiling, and analyze the results to identify performance bottlenecks.

## Learning Goals

- Set up CPU profiling in a Go application
- Generate CPU profiles using different methods
- Analyze profiles using `go tool pprof`
- Understand profile output and metrics
- Identify performance bottlenecks

## Background

CPU profiling is the foundation of performance optimization. Before you can optimize code, you need to understand where your application spends its time. This exercise introduces the basic tools and techniques for CPU profiling in Go.

## Tasks

### Task 1: Create a CPU-Intensive Application

Create an application that performs various CPU-intensive operations to generate meaningful profiling data.

**Requirements:**
1. Implement multiple CPU-intensive functions
2. Include different types of operations (math, string processing, data structures)
3. Make the application run long enough to collect meaningful profile data
4. Add command-line flags for controlling behavior

### Task 2: Add CPU Profiling Instrumentation

Instrument your application with CPU profiling capabilities.

**Requirements:**
1. Add file-based CPU profiling
2. Add HTTP profiling endpoint
3. Make profiling configurable via command-line flags
4. Ensure proper cleanup of profiling resources

### Task 3: Generate and Analyze Profiles

Run your application and generate CPU profiles using different methods.

**Requirements:**
1. Generate profiles using file output
2. Generate profiles using HTTP endpoint
3. Generate profiles from benchmarks
4. Analyze profiles using `go tool pprof`

### Task 4: Profile Analysis and Optimization

Analyze the generated profiles and implement optimizations.

**Requirements:**
1. Identify the top CPU-consuming functions
2. Analyze call graphs and hot paths
3. Implement at least one optimization
4. Compare before/after profiles

## Implementation Guide

### Step 1: Application Structure

Create the following file structure:
```
exercise-01-01/
├── main.go
├── cpu_intensive.go
├── profiling.go
├── go.mod
└── README.md
```

### Step 2: CPU-Intensive Operations

Implement various CPU-intensive functions:

```go
// cpu_intensive.go
package main

import (
    "crypto/md5"
    "fmt"
    "math"
    "strings"
)

// Mathematical operations
func calculatePrimes(limit int) []int {
    // Implement sieve of Eratosthenes
}

func calculateFibonacci(n int) int {
    // Implement recursive Fibonacci (intentionally inefficient)
}

// String operations
func processStrings(data []string) []string {
    // Implement string processing operations
}

// Hash operations
func calculateHashes(data []string) []string {
    // Calculate MD5 hashes
}

// Data structure operations
func sortAndSearch(data []int) int {
    // Implement sorting and searching
}
```

### Step 3: Profiling Infrastructure

```go
// profiling.go
package main

import (
    "flag"
    "log"
    "net/http"
    _ "net/http/pprof"
    "os"
    "runtime/pprof"
)

type ProfilingConfig struct {
    CPUProfile   string
    HTTPProfile  bool
    ProfilePort  string
}

func setupProfiling(config ProfilingConfig) func() {
    // Setup CPU profiling
    // Setup HTTP profiling if enabled
    // Return cleanup function
}
```

### Step 4: Main Application

```go
// main.go
package main

import (
    "flag"
    "fmt"
    "log"
    "time"
)

func main() {
    // Parse command-line flags
    var (
        cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
        httpProfile = flag.Bool("httpprofile", false, "enable HTTP profiling")
        profilePort = flag.String("profileport", "6060", "HTTP profiling port")
        duration = flag.Duration("duration", 30*time.Second, "run duration")
        workers = flag.Int("workers", 4, "number of worker goroutines")
    )
    flag.Parse()

    // Setup profiling
    cleanup := setupProfiling(ProfilingConfig{
        CPUProfile:  *cpuProfile,
        HTTPProfile: *httpProfile,
        ProfilePort: *profilePort,
    })
    defer cleanup()

    // Run CPU-intensive workload
    runWorkload(*duration, *workers)
}

func runWorkload(duration time.Duration, workers int) {
    // Implement workload that runs for specified duration
    // Use multiple goroutines to simulate realistic load
}
```

## Expected Output

After completing this exercise, you should be able to:

1. **Generate CPU Profiles:**
   ```bash
   # File-based profiling
   go run . -cpuprofile=cpu.prof -duration=30s
   
   # HTTP profiling
   go run . -httpprofile=true &
   curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
   ```

2. **Analyze Profiles:**
   ```bash
   # Interactive analysis
   go tool pprof cpu.prof
   
   # Web interface
   go tool pprof -http=:8080 cpu.prof
   
   # Top functions
   go tool pprof -top cpu.prof
   ```

3. **Profile Output Example:**
   ```
   (pprof) top10
   Showing nodes accounting for 2.50s, 83.33% of 3.00s total
         flat  flat%   sum%        cum   cum%
        0.89s 29.67% 29.67%      0.89s 29.67%  main.calculateFibonacci
        0.45s 15.00% 44.67%      0.45s 15.00%  main.calculatePrimes
        0.34s 11.33% 56.00%      1.20s 40.00%  main.processStrings
   ```

## Verification Steps

1. **Profile Generation:**
   - [ ] Application generates CPU profiles successfully
   - [ ] HTTP profiling endpoint works
   - [ ] Profiles contain meaningful data

2. **Profile Analysis:**
   - [ ] Can identify top CPU-consuming functions
   - [ ] Can navigate call graphs
   - [ ] Can understand flat vs cumulative time

3. **Optimization:**
   - [ ] Identified at least one optimization opportunity
   - [ ] Implemented optimization
   - [ ] Measured improvement with new profile

## Common Issues and Solutions

### Issue: Empty or Minimal Profile Data
**Solution:** Ensure your application runs long enough and performs sufficient work

### Issue: HTTP Profiling Not Working
**Solution:** Check that the import `_ "net/http/pprof"` is included

### Issue: Profile Analysis Confusion
**Solution:** Focus on "flat" time first - this shows time spent in the function itself

### Issue: No Obvious Bottlenecks
**Solution:** Make your CPU-intensive functions more expensive or run longer

## Extension Challenges

1. **Comparative Analysis:**
   - Generate profiles before and after optimization
   - Use `go tool pprof -base=old.prof new.prof` to compare

2. **Benchmark Integration:**
   - Create benchmarks for your CPU-intensive functions
   - Use `go test -bench=. -cpuprofile=bench.prof`

3. **Continuous Profiling:**
   - Implement a simple continuous profiling system
   - Collect profiles every minute for 10 minutes

4. **Profile Visualization:**
   - Generate flame graphs
   - Create call graph visualizations
   - Export profiles to different formats

## Key Learning Points

1. **Profiling Overhead:** CPU profiling has minimal overhead (~1-5%)
2. **Sampling Nature:** Profiles are statistical samples, not exact measurements
3. **Hot Path Focus:** Optimize functions with highest flat time first
4. **Representative Workloads:** Profile with realistic data and usage patterns
5. **Iterative Process:** Profile → Optimize → Measure → Repeat

## Next Steps

After completing this exercise:
1. Experiment with different profiling durations
2. Try profiling real applications you've written
3. Practice interpreting different types of profile output
4. Move to [Exercise 01-02: Flame Graph Analysis](../exercise-01-02/README.md)

## Resources

- [Go pprof Documentation](https://golang.org/pkg/runtime/pprof/)
- [Profiling Go Programs Blog Post](https://blog.golang.org/pprof)
- [pprof Command Reference](https://github.com/google/pprof/blob/master/doc/README.md)

---

*Exercise 01-01 of Chapter 01: CPU Profiling Techniques*