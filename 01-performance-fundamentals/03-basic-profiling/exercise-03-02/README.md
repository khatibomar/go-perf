# Exercise 03-02: Memory Profiling Analysis

## Overview
This exercise demonstrates how to use Go's memory profiler to identify memory bottlenecks, allocation patterns, and optimization opportunities. We'll profile memory-intensive functions with deliberate inefficiencies and analyze heap usage patterns.

## Running Memory Profiling

### Step 1: Generate Memory Profile
```bash
# Run benchmarks with memory profiling
go test -bench=. -memprofile=mem.prof -benchtime=10s

# Or run specific benchmark
go test -bench=BenchmarkMemoryIntensiveWorkload -memprofile=mem.prof -benchtime=10s

# Generate allocation profile
go test -bench=. -memprofile=alloc.prof -memprofilerate=1 -benchtime=10s
```

### Step 2: Analyze Memory Profiles
```bash
# Interactive analysis
go tool pprof mem.prof

# Web interface (recommended)
go tool pprof -http=:8080 mem.prof

# Analyze allocations
go tool pprof -alloc_space mem.prof
go tool pprof -alloc_objects mem.prof
```

### Step 3: Generate Memory Reports
```bash
# Top memory consumers
go tool pprof -top mem.prof

# List specific function
go tool pprof -list=stringConcatenationBad mem.prof

# Generate memory usage graph
go tool pprof -svg mem.prof > memory_graph.svg
```

## Expected Memory Profile Analysis Results

### 1. Top Memory Allocators
When you run memory profile analysis, you should see output similar to:

```
(pprof) top
Showing nodes accounting for 512.45MB, 89.23% of 574.32MB total
Dropped 23 nodes (cum <= 2.87MB)
      flat  flat%   sum%        cum   cum%
   156.78MB 27.31% 27.31%   156.78MB 27.31%  main.mapAllocationsBad
   98.45MB  17.14% 44.45%    98.45MB  17.14%  main.sliceGrowthBad
   87.23MB  15.19% 59.64%    87.23MB  15.19%  main.stringConcatenationBad
   65.34MB  11.38% 71.02%    65.34MB  11.38%  main.createDeepTreeBad
   45.67MB   7.95% 78.97%    45.67MB   7.95%  main.interfaceAllocationsBad
   34.89MB   6.08% 85.05%    34.89MB   6.08%  main.closureAllocationsBad
   24.09MB   4.19% 89.24%    24.09MB   4.19%  main.temporaryAllocationsBad
```

### 2. Allocation Analysis

#### String Concatenation Issues
```bash
(pprof) list stringConcatenationBad
```

Expected findings:
- **Problem**: O(n²) memory allocations due to string immutability
- **Pattern**: Each concatenation creates new string, copying all previous data
- **Memory Growth**: Exponential memory usage with input size

#### Slice Growth Problems
```bash
(pprof) list sliceGrowthBad
```

Expected findings:
- **Problem**: Multiple slice reallocations due to zero initial capacity
- **Pattern**: Slice doubles in size each time capacity is exceeded
- **Waste**: Temporary allocations during copying

#### Map Allocation Inefficiencies
```bash
(pprof) list mapAllocationsBad
```

Expected findings:
- **Problem**: No initial capacity hint causes multiple rehashing
- **Pattern**: Complex nested structures create many small allocations
- **Overhead**: Interface{} boxing causes additional heap allocations

### 3. Memory Allocation Patterns

#### Allocation Rate Analysis
```bash
# View allocation objects
(pprof) top -cum alloc_objects

# View allocation space
(pprof) top -cum alloc_space
```

Expected patterns:
- **High allocation rate**: Functions creating many temporary objects
- **Large allocations**: Functions allocating big chunks of memory
- **Frequent small allocations**: Interface boxing, string operations

### 4. Heap Growth Analysis
```bash
# Analyze in-use memory
(pprof) top -inuse_space

# Check for memory leaks
(pprof) top -inuse_objects
```

## Key Memory Profiling Insights

### 1. Allocation Hotspots

#### String Operations
- **Issue**: String concatenation creates O(n²) allocations
- **Solution**: Use `strings.Builder` with pre-allocated capacity
- **Impact**: 10-100x reduction in allocations

#### Slice Growth
- **Issue**: Uncontrolled slice growth causes reallocations
- **Solution**: Pre-allocate slices with known capacity
- **Impact**: Eliminates reallocation overhead

#### Interface Boxing
- **Issue**: Storing primitives in `interface{}` causes heap allocations
- **Solution**: Use typed containers or avoid interfaces
- **Impact**: Reduces allocation count and GC pressure

### 2. Memory Leak Patterns

#### Slice References
```go
// Problem: Keeps entire slice in memory
bigSlice := make([]byte, 1024*1024)
smallRef := bigSlice[:100] // Entire 1MB stays allocated

// Solution: Copy what you need
bigSlice := make([]byte, 1024*1024)
smallCopy := make([]byte, 100)
copy(smallCopy, bigSlice[:100])
// bigSlice can now be garbage collected
```

#### Closure Captures
```go
// Problem: Closure captures large data structures
func createClosures(data [][]byte) []func() {
    funcs := make([]func(), len(data))
    for i, d := range data {
        funcs[i] = func() { // Captures entire 'd' slice
            fmt.Println(len(d))
        }
    }
    return funcs
}

// Solution: Capture only what's needed
func createClosures(data [][]byte) []func() {
    funcs := make([]func(), len(data))
    for i, d := range data {
        length := len(d) // Capture only the length
        funcs[i] = func() {
            fmt.Println(length)
        }
    }
    return funcs
}
```

### 3. GC Pressure Analysis

#### Allocation Rate Impact
- **High allocation rate**: Triggers frequent GC cycles
- **Short-lived objects**: Increase GC overhead
- **Large objects**: May cause GC pauses

#### Memory Fragmentation
- **Many small allocations**: Can cause heap fragmentation
- **Mixed object sizes**: Reduces memory locality
- **Object pooling**: Reduces allocation pressure

## Optimization Strategies

### 1. Pre-allocation
```go
// Before: Multiple reallocations
var result []int
for i := 0; i < n; i++ {
    result = append(result, i)
}

// After: Single allocation
result := make([]int, 0, n) // Pre-allocate capacity
for i := 0; i < n; i++ {
    result = append(result, i)
}
```

### 2. Object Pooling
```go
// Buffer pool for reusing allocations
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1024)
    },
}

func processData() {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    
    // Use buffer...
}
```

### 3. String Building
```go
// Before: O(n²) allocations
result := ""
for i := 0; i < n; i++ {
    result += fmt.Sprintf("item_%d", i)
}

// After: O(n) allocations
var builder strings.Builder
builder.Grow(n * 10) // Estimate capacity
for i := 0; i < n; i++ {
    builder.WriteString("item_")
    builder.WriteString(strconv.Itoa(i))
}
result := builder.String()
```

### 4. Avoiding Interface Allocations
```go
// Before: Boxing causes allocations
values := make([]interface{}, n)
for i := 0; i < n; i++ {
    values[i] = i // int -> interface{} allocation
}

// After: Use typed containers
type Container struct {
    Ints    []int
    Strings []string
    // ... other typed fields
}
```

## Comparative Analysis

### Running Before/After Comparison
```bash
# Profile original functions
go test -bench=BenchmarkStringConcatenationBad -memprofile=mem_bad.prof

# Profile optimized functions
go test -bench=BenchmarkStringConcatenationGood -memprofile=mem_good.prof

# Compare profiles
go tool pprof -base mem_bad.prof mem_good.prof
```

### Expected Memory Improvements
- **String concatenation**: 100x+ reduction in allocations
- **Slice growth**: 10x+ reduction in memory usage
- **Map allocations**: 50%+ reduction in allocation count
- **Interface boxing**: 90%+ reduction in heap allocations
- **Object pooling**: 80%+ reduction in GC pressure

## Advanced Memory Analysis

### 1. Escape Analysis
```bash
# Check what escapes to heap
go build -gcflags="-m" .

# Detailed escape analysis
go build -gcflags="-m -m" .
```

### 2. GC Trace Analysis
```bash
# Enable GC tracing
GODEBUG=gctrace=1 go test -bench=.

# Memory allocator tracing
GODEBUG=allocfreetrace=1 go test -bench=BenchmarkSpecific
```

### 3. Live Heap Analysis
```bash
# Analyze live objects
go tool pprof -inuse_space mem.prof

# Check for memory leaks
go tool pprof -inuse_objects mem.prof
```

## Common pprof Memory Commands

### Interactive Mode Commands
```
(pprof) top                    # Show top memory consumers
(pprof) top -cum               # Show cumulative memory usage
(pprof) list main.functionName # Show source with annotations
(pprof) web                    # Generate web view
(pprof) svg                    # Generate SVG graph
(pprof) peek main.functionName # Show callers and callees
(pprof) traces                 # Show allocation traces
```

### Command Line Options
```bash
# Different memory views
go tool pprof -alloc_space mem.prof    # Total allocated space
go tool pprof -alloc_objects mem.prof  # Total allocated objects
go tool pprof -inuse_space mem.prof    # Currently in-use space
go tool pprof -inuse_objects mem.prof  # Currently in-use objects

# Focus on specific functions
go tool pprof -focus=stringConcatenation mem.prof
go tool pprof -ignore=runtime mem.prof

# Generate reports
go tool pprof -top mem.prof
go tool pprof -svg mem.prof > memory.svg
go tool pprof -png mem.prof > memory.png
```

## Memory Profiling Best Practices

### 1. Profiling Environment
- Use consistent test data sizes
- Run benchmarks for sufficient duration
- Minimize background memory usage
- Use `-memprofilerate=1` for detailed allocation tracking

### 2. Profile Interpretation
- Focus on "flat" allocations for direct optimization targets
- Use "cum" allocations to understand call hierarchies
- Compare allocation space vs. allocation objects
- Look for unexpected allocation patterns

### 3. Optimization Workflow
1. Profile to identify memory hotspots
2. Understand allocation patterns
3. Implement targeted optimizations
4. Re-profile to validate improvements
5. Monitor GC impact and overall performance

## Exercise Tasks

### Task 1: Basic Memory Profiling
1. Run memory-intensive benchmarks with profiling
2. Analyze memory profiles using `go tool pprof`
3. Identify top 3 memory allocators
4. Document allocation patterns

### Task 2: Allocation Analysis
1. Use different profile views (alloc_space, alloc_objects, inuse_space)
2. Understand why each function allocates memory
3. Identify optimization opportunities
4. Analyze GC pressure indicators

### Task 3: Optimization Validation
1. Implement optimized versions using provided examples
2. Create benchmarks for optimized functions
3. Profile optimized code
4. Compare before/after memory usage

### Task 4: Advanced Analysis
1. Use escape analysis to understand heap allocations
2. Analyze GC traces for allocation patterns
3. Generate memory usage graphs
4. Document optimization strategies and results

### Task 5: Memory Leak Detection
1. Profile long-running benchmarks
2. Compare inuse vs. alloc profiles
3. Identify potential memory leaks
4. Validate fixes with follow-up profiling

## Expected Learning Outcomes

After completing this exercise, you should be able to:
- Generate and analyze memory profiles from Go benchmarks
- Interpret different memory profile views (alloc vs. inuse)
- Identify common memory allocation anti-patterns
- Understand the relationship between allocations and GC pressure
- Implement memory optimizations using pooling, pre-allocation, and efficient data structures
- Validate memory optimizations using profiling data
- Detect and fix memory leaks using profiling tools

## Next Steps

Proceed to Exercise 03-03 to learn about live application profiling and production monitoring techniques.