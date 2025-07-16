# Chapter 03: Basic Profiling with pprof

## Learning Objectives
By the end of this chapter, you will be able to:
- Use Go's built-in pprof profiler effectively
- Generate and analyze CPU profiles
- Generate and analyze memory profiles
- Interpret profiling data and identify bottlenecks
- Use pprof's web interface and command-line tools
- Profile running applications and benchmarks
- Understand different profiling modes and their use cases

## Prerequisites
- Completed Chapter 01 (Performance Mindset) and Chapter 02 (Go Runtime)
- Basic understanding of Go benchmarks
- Familiarity with command-line tools

## Theory: Introduction to pprof

### What is pprof?
pprof is Go's built-in profiling tool that helps you understand where your program spends time and allocates memory. It's based on Google's pprof tool and is integrated into the Go runtime.

### Types of Profiles

#### 1. CPU Profile
- **Purpose**: Shows where your program spends CPU time
- **Use Case**: Identifying computational bottlenecks
- **Sampling**: Statistical sampling every 10ms by default
- **Best For**: CPU-bound performance issues

#### 2. Memory Profile (Heap)
- **Purpose**: Shows memory allocation patterns
- **Use Case**: Identifying memory leaks and excessive allocations
- **Sampling**: Samples allocations based on size and frequency
- **Best For**: Memory-related performance issues

#### 3. Goroutine Profile
- **Purpose**: Shows all current goroutines and their stack traces
- **Use Case**: Debugging goroutine leaks and concurrency issues
- **Sampling**: Point-in-time snapshot
- **Best For**: Concurrency debugging

#### 4. Block Profile
- **Purpose**: Shows where goroutines block on synchronization primitives
- **Use Case**: Identifying contention issues
- **Sampling**: Tracks blocking events
- **Best For**: Synchronization bottlenecks

#### 5. Mutex Profile
- **Purpose**: Shows mutex contention
- **Use Case**: Identifying lock contention
- **Sampling**: Tracks mutex wait times
- **Best For**: Lock optimization

### Profiling Methods

#### 1. Benchmark Profiling
```bash
# CPU profiling
go test -bench=. -cpuprofile=cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof

# Both CPU and memory
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
```

#### 2. Runtime Profiling
```go
import (
    "os"
    "runtime/pprof"
)

// CPU profiling
f, _ := os.Create("cpu.prof")
defer f.Close()
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()

// Memory profiling
f, _ := os.Create("mem.prof")
defer f.Close()
runtime.GC()
pprof.WriteHeapProfile(f)
```

#### 3. HTTP Profiling
```go
import _ "net/http/pprof"

// Access profiles at:
// http://localhost:6060/debug/pprof/
```

### Analyzing Profiles

#### Command Line Analysis
```bash
# Interactive mode
go tool pprof cpu.prof

# Top functions
go tool pprof -top cpu.prof

# List specific function
go tool pprof -list=functionName cpu.prof

# Web interface
go tool pprof -http=:8080 cpu.prof
```

#### Key pprof Commands
- `top`: Show top functions by resource usage
- `list <func>`: Show source code with annotations
- `web`: Generate SVG call graph
- `peek <func>`: Show callers and callees
- `traces`: Show sample traces
- `disasm <func>`: Show assembly code

## Tools Setup

### Required Tools
```bash
# Graphviz for call graphs (optional but recommended)
# Windows: Download from https://graphviz.org/download/
# macOS: brew install graphviz
# Linux: sudo apt-get install graphviz

# Verify installation
dot -V
```

### IDE Integration
- **VS Code**: Go extension supports pprof integration
- **GoLand**: Built-in profiling support
- **Vim/Neovim**: vim-go plugin supports pprof

## Video Resources
- [Profiling Go Programs - Go Team](https://www.youtube.com/watch?v=N3PWzBeLX2M)
- [Advanced Go Profiling - Dave Cheney](https://www.youtube.com/watch?v=xxDZuPEgbBU)

## Exercises

### Exercise 03-01: Basic CPU Profiling
**Objective**: Learn to generate and analyze CPU profiles

**Tasks**:
1. Create a CPU-intensive function with deliberate inefficiencies
2. Write benchmarks for the function
3. Generate CPU profiles using `go test -cpuprofile`
4. Analyze profiles using `go tool pprof`
5. Identify and fix performance bottlenecks
6. Compare before/after profiles

**Files**: `exercise-03-01/`
- `cpu_intensive.go` - Functions with performance issues
- `benchmark_test.go` - Benchmarks for profiling
- `optimized.go` - Optimized versions
- `profile_analysis.md` - Analysis documentation

### Exercise 03-02: Memory Profiling
**Objective**: Learn to identify memory allocation patterns and leaks

**Tasks**:
1. Create functions with various memory allocation patterns
2. Generate memory profiles
3. Analyze heap usage and allocation patterns
4. Identify memory leaks and excessive allocations
5. Optimize memory usage
6. Use `go tool pprof` for memory analysis

**Files**: `exercise-03-02/`
- `memory_patterns.go` - Various allocation patterns
- `benchmark_test.go` - Memory benchmarks
- `leak_detection.go` - Memory leak examples
- `optimized_memory.go` - Optimized versions

### Exercise 03-03: Live Application Profiling
**Objective**: Profile a running web application

**Tasks**:
1. Create a simple web server with performance issues
2. Integrate `net/http/pprof` for live profiling
3. Generate load and collect profiles
4. Analyze profiles from a running application
5. Identify and fix bottlenecks
6. Monitor improvements in real-time

**Files**: `exercise-03-03/`
- `web_server.go` - Web application with issues
- `load_generator.go` - Tool to generate load
- `profiling_integration.go` - pprof integration
- `optimized_server.go` - Improved version

### Exercise 03-04: Advanced Profiling Techniques
**Objective**: Explore goroutine, block, and mutex profiling

**Tasks**:
1. Create applications with concurrency issues
2. Generate goroutine profiles
3. Analyze block and mutex profiles
4. Identify contention and synchronization issues
5. Optimize concurrent code
6. Compare different profiling types

**Files**: `exercise-03-04/`
- `concurrency_issues.go` - Concurrent code with problems
- `profiling_types.go` - Different profile generation
- `analysis_tools.go` - Profile analysis utilities
- `optimized_concurrent.go` - Improved concurrent code

## Key Concepts

### Profile Interpretation
- **Flat Time**: Time spent in the function itself
- **Cumulative Time**: Time spent in function and its callees
- **Sample Count**: Number of samples collected
- **Call Graph**: Visual representation of function calls

### Profiling Best Practices
1. **Profile Representative Workloads**: Use realistic data and scenarios
2. **Profile Long Enough**: Ensure sufficient samples for accuracy
3. **Avoid Profiling Overhead**: Be aware that profiling adds overhead
4. **Focus on Hot Paths**: Optimize functions that appear at the top
5. **Validate Optimizations**: Always measure before and after changes

### Common Profiling Patterns
- **Top-Down Analysis**: Start with highest resource consumers
- **Bottom-Up Analysis**: Look at leaf functions first
- **Differential Profiling**: Compare profiles before/after changes
- **Continuous Profiling**: Monitor production applications

## Analysis Tools

### pprof Web Interface
```bash
# Start web interface
go tool pprof -http=:8080 profile.prof

# Features:
# - Interactive call graphs
# - Flame graphs
# - Source code view
# - Top functions
# - Comparative analysis
```

### Command Line Tools
```bash
# Basic analysis
go tool pprof -top profile.prof
go tool pprof -list=main profile.prof

# Advanced analysis
go tool pprof -traces profile.prof
go tool pprof -peek=main profile.prof
go tool pprof -disasm=main profile.prof
```

### Third-Party Tools
- **pprof++**: Enhanced pprof with additional features
- **go-torch**: Flame graph generator (deprecated, use pprof web)
- **Continuous Profiling**: Pyroscope, Polar Signals

## Common Optimization Patterns

### CPU Optimization
1. **Algorithm Improvements**: Better time complexity
2. **Loop Optimization**: Reduce iterations and improve cache locality
3. **Function Inlining**: Reduce call overhead
4. **Data Structure Optimization**: Choose appropriate data structures

### Memory Optimization
1. **Reduce Allocations**: Reuse objects, use object pools
2. **Optimize Data Structures**: Reduce memory footprint
3. **Fix Memory Leaks**: Ensure proper cleanup
4. **Improve GC Pressure**: Reduce allocation rate

## Success Criteria
By the end of this chapter, you should be able to:
- [ ] Generate CPU and memory profiles from benchmarks
- [ ] Use `go tool pprof` to analyze profiles
- [ ] Interpret profile data and identify bottlenecks
- [ ] Use the pprof web interface effectively
- [ ] Profile running applications using `net/http/pprof`
- [ ] Understand different types of profiles and their use cases
- [ ] Apply profiling-driven optimization techniques
- [ ] Validate performance improvements using profiles

## Next Steps
After completing this chapter, proceed to:
- **Chapter 04**: Benchmarking Best Practices
- **Module 02**: Memory Optimization Techniques
- Practice profiling real-world applications
- Explore continuous profiling solutions

## Additional Resources
- [Go Blog: Profiling Go Programs](https://go.dev/blog/pprof)
- [pprof Documentation](https://pkg.go.dev/runtime/pprof)
- [Graphviz Documentation](https://graphviz.org/documentation/)
- [Go Wiki: Debugging performance issues in Go programs](https://go.dev/wiki/Performance)