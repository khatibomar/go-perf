# Go Benchmarking Cheat Sheet

Quick reference for Go's built-in benchmarking and testing tools.

## Basic Benchmarking

### Writing Benchmarks
```go
// Basic benchmark function
func BenchmarkFunction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Code to benchmark
        result := functionToTest()
        _ = result // Prevent optimization
    }
}

// Benchmark with setup
func BenchmarkWithSetup(b *testing.B) {
    // Setup code (not timed)
    data := generateTestData()
    
    b.ResetTimer() // Reset timer after setup
    
    for i := 0; i < b.N; i++ {
        // Only this code is timed
        result := processData(data)
        _ = result
    }
}

// Benchmark with cleanup
func BenchmarkWithCleanup(b *testing.B) {
    for i := 0; i < b.N; i++ {
        b.StopTimer()
        data := setupIteration()
        b.StartTimer()
        
        result := processData(data)
        _ = result
    }
}
```

### Running Benchmarks
```bash
# Run all benchmarks
go test -bench=.

# Run specific benchmark
go test -bench=BenchmarkFunction

# Run benchmarks matching pattern
go test -bench=BenchmarkString.*

# Run benchmarks multiple times
go test -bench=. -count=5

# Set benchmark time
go test -bench=. -benchtime=10s
go test -bench=. -benchtime=1000x  # 1000 iterations
```

## Memory Benchmarking

### Memory Allocation Tracking
```bash
# Include memory statistics
go test -bench=. -benchmem

# Example output:
# BenchmarkFunction-8    1000000    1234 ns/op    456 B/op    7 allocs/op
#                       iterations  time/op      bytes/op  allocations/op
```

### Memory-Focused Benchmarks
```go
func BenchmarkMemoryUsage(b *testing.B) {
    b.ReportAllocs() // Enable allocation reporting
    
    for i := 0; i < b.N; i++ {
        data := make([]byte, 1024)
        _ = data
    }
}

// Custom memory metrics
func BenchmarkCustomMetrics(b *testing.B) {
    var m1, m2 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    for i := 0; i < b.N; i++ {
        // Code to benchmark
    }
    
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "B/op")
}
```

## Sub-benchmarks

### Benchmark Suites
```go
func BenchmarkSuite(b *testing.B) {
    sizes := []int{10, 100, 1000, 10000}
    
    for _, size := range sizes {
        b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
            data := make([]int, size)
            b.ResetTimer()
            
            for i := 0; i < b.N; i++ {
                sort.Ints(data)
            }
        })
    }
}

// Running specific sub-benchmarks
// go test -bench=BenchmarkSuite/size-100
```

### Parallel Benchmarks
```go
func BenchmarkParallel(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            // Code to benchmark in parallel
            result := expensiveFunction()
            _ = result
        }
    })
}

// Set parallelism
// go test -bench=. -cpu=1,2,4,8
```

## Advanced Benchmarking

### Custom Metrics
```go
func BenchmarkCustomMetrics(b *testing.B) {
    var totalBytes int64
    
    for i := 0; i < b.N; i++ {
        data := processData()
        totalBytes += int64(len(data))
    }
    
    // Report custom metrics
    b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
    b.ReportMetric(float64(totalBytes)/(1024*1024), "MB/total")
}
```

### Benchmark Helpers
```go
// Prevent compiler optimizations
var result interface{}

func BenchmarkPreventOptimization(b *testing.B) {
    var r interface{}
    
    for i := 0; i < b.N; i++ {
        r = expensiveFunction()
    }
    
    result = r // Store result globally
}

// Benchmark with different inputs
func BenchmarkWithInputs(b *testing.B) {
    inputs := []string{"small", "medium input", "very long input string"}
    
    for _, input := range inputs {
        b.Run(input, func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                result := process(input)
                _ = result
            }
        })
    }
}
```

## Profiling Integration

### CPU Profiling
```bash
# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof

# Block profiling
go test -bench=. -blockprofile=block.prof
go tool pprof block.prof

# Mutex profiling
go test -bench=. -mutexprofile=mutex.prof
go tool pprof mutex.prof
```

### Trace Analysis
```bash
# Generate execution trace
go test -bench=. -trace=trace.out
go tool trace trace.out
```

## Benchmark Analysis

### Statistical Analysis
```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Compare benchmarks
go test -bench=. -count=10 > old.txt
# Make changes
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt
```

### Benchmark Output Format
```
BenchmarkFunction-8    1000000    1234 ns/op    456 B/op    7 allocs/op
│                  │         │            │          │            │
│                  │         │            │          │            └─ Allocations per operation
│                  │         │            │          └─ Bytes allocated per operation
│                  │         │            └─ Nanoseconds per operation
│                  │         └─ Number of iterations
│                  └─ GOMAXPROCS value
└─ Benchmark name
```

## Command Line Options

### Basic Options
```bash
# Run benchmarks
go test -bench=.                    # All benchmarks
go test -bench=BenchmarkName        # Specific benchmark
go test -bench="Benchmark.*"        # Pattern matching

# Control execution
go test -bench=. -benchtime=5s      # Run for 5 seconds
go test -bench=. -benchtime=100x    # Run 100 iterations
go test -bench=. -count=5           # Run 5 times
go test -bench=. -cpu=1,2,4         # Different GOMAXPROCS
```

### Memory and Profiling
```bash
# Memory analysis
go test -bench=. -benchmem          # Include memory stats
go test -bench=. -memprofile=mem.prof

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof

# Other profiles
go test -bench=. -blockprofile=block.prof
go test -bench=. -mutexprofile=mutex.prof
go test -bench=. -trace=trace.out
```

### Output Control
```bash
# Verbose output
go test -bench=. -v

# Short mode (skip long tests)
go test -bench=. -short

# Timeout
go test -bench=. -timeout=30m

# Parallel execution
go test -bench=. -parallel=4
```

## Best Practices

### Writing Good Benchmarks
```go
// DO: Use b.N in the loop
func BenchmarkGood(b *testing.B) {
    for i := 0; i < b.N; i++ {
        result := function()
        _ = result
    }
}

// DON'T: Fixed iteration count
func BenchmarkBad(b *testing.B) {
    for i := 0; i < 1000; i++ { // Wrong!
        result := function()
        _ = result
    }
}

// DO: Reset timer after setup
func BenchmarkWithSetup(b *testing.B) {
    data := expensiveSetup()
    b.ResetTimer() // Important!
    
    for i := 0; i < b.N; i++ {
        process(data)
    }
}

// DO: Prevent compiler optimizations
var sink interface{}

func BenchmarkPreventOpt(b *testing.B) {
    for i := 0; i < b.N; i++ {
        sink = expensiveFunction()
    }
}
```

### Benchmark Organization
```go
// Group related benchmarks
func BenchmarkStringOperations(b *testing.B) {
    b.Run("concat", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = "hello" + "world"
        }
    })
    
    b.Run("sprintf", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = fmt.Sprintf("%s%s", "hello", "world")
        }
    })
    
    b.Run("builder", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            var builder strings.Builder
            builder.WriteString("hello")
            builder.WriteString("world")
            _ = builder.String()
        }
    })
}
```

## Common Patterns

### Benchmarking Different Sizes
```go
func BenchmarkSizes(b *testing.B) {
    sizes := []int{1, 10, 100, 1000, 10000}
    
    for _, size := range sizes {
        b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
            data := make([]int, size)
            b.ResetTimer()
            
            for i := 0; i < b.N; i++ {
                sort.Ints(data)
            }
        })
    }
}
```

### Benchmarking Algorithms
```go
func BenchmarkSortAlgorithms(b *testing.B) {
    data := generateRandomData(1000)
    
    b.Run("stdlib", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            dataCopy := make([]int, len(data))
            copy(dataCopy, data)
            sort.Ints(dataCopy)
        }
    })
    
    b.Run("quicksort", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            dataCopy := make([]int, len(data))
            copy(dataCopy, data)
            quickSort(dataCopy)
        }
    })
}
```

### HTTP Benchmarking
```go
func BenchmarkHTTPHandler(b *testing.B) {
    handler := http.HandlerFunc(myHandler)
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req := httptest.NewRequest("GET", "/", nil)
            w := httptest.NewRecorder()
            handler.ServeHTTP(w, req)
        }
    })
}
```

## Troubleshooting

### Common Issues
```go
// Issue: Benchmark too fast
func BenchmarkTooFast(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = 1 + 1 // Too simple, may be optimized away
    }
}

// Solution: Add complexity or prevent optimization
var result int
func BenchmarkFixed(b *testing.B) {
    var r int
    for i := 0; i < b.N; i++ {
        r = complexCalculation(i)
    }
    result = r
}

// Issue: Setup in benchmark loop
func BenchmarkWithSetupIssue(b *testing.B) {
    for i := 0; i < b.N; i++ {
        data := expensiveSetup() // Wrong! Setup in loop
        process(data)
    }
}

// Solution: Move setup outside or use ResetTimer
func BenchmarkWithSetupFixed(b *testing.B) {
    data := expensiveSetup()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        process(data)
    }
}
```

### Debugging Benchmarks
```bash
# Verbose output
go test -bench=. -v

# Check what's being optimized
go test -bench=. -gcflags="-m"

# Disable optimizations
go test -bench=. -gcflags="-N -l"

# Assembly output
go test -bench=. -gcflags="-S"
```

## Environment Variables

```bash
# Benchmark settings
export GOMAXPROCS=4
export GOGC=100
export GODEBUG=gctrace=1

# Disable CPU frequency scaling
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

## Quick Reference

| Command | Description |
|---------|-------------|
| `go test -bench=.` | Run all benchmarks |
| `go test -bench=Name` | Run specific benchmark |
| `go test -bench=. -benchmem` | Include memory stats |
| `go test -bench=. -count=5` | Run 5 times |
| `go test -bench=. -benchtime=10s` | Run for 10 seconds |
| `go test -bench=. -cpu=1,2,4` | Different GOMAXPROCS |
| `go test -bench=. -cpuprofile=cpu.prof` | Generate CPU profile |
| `go test -bench=. -memprofile=mem.prof` | Generate memory profile |
| `benchstat old.txt new.txt` | Compare benchmark results |

---

*For more details, see: `go help test` and `go help testflag`*