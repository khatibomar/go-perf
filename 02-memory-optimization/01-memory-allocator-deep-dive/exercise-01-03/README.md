# Exercise 01-03: Size Class Optimization

## Overview
This exercise focuses on understanding and optimizing Go's memory allocator size classes. You'll learn how to analyze allocation patterns, identify wastage, and optimize memory usage through better size class alignment and struct layout optimization.

## Learning Objectives
- Understand Go's size class system and its impact on memory efficiency
- Analyze real-world allocation patterns and identify optimization opportunities
- Learn struct alignment and padding optimization techniques
- Implement tools for tracking and optimizing memory allocation patterns
- Measure the performance impact of size class optimization

## Files Structure
```
exercise-01-03/
├── README.md                 # This file
├── size_optimization.go      # Size class optimization implementation
└── size_benchmark_test.go    # Comprehensive benchmarks and tests
```

## Key Components

### 1. SizeClassOptimizer
**Purpose**: Tracks and analyzes memory allocation patterns to identify wastage

**Features**:
- Records allocation sizes and frequencies
- Maps allocations to Go's size classes
- Calculates wastage and efficiency metrics
- Thread-safe concurrent recording

**Key Methods**:
- `RecordAllocation(size int)`: Records an allocation
- `AnalyzeWastage()`: Analyzes wastage patterns
- `findSizeClass(size int)`: Maps size to size class

### 2. OptimizedSizeClassGenerator
**Purpose**: Generates optimized size classes based on allocation patterns

**Features**:
- Analyzes allocation frequency patterns
- Generates size classes with configurable waste thresholds
- Optimizes for common allocation sizes
- Balances memory efficiency vs. fragmentation

**Configuration**:
- `maxWastage`: Maximum acceptable waste percentage (e.g., 0.25 = 25%)
- `minEfficiency`: Minimum efficiency threshold (e.g., 0.75 = 75%)

### 3. AllocationTracker
**Purpose**: Real-time tracking of allocation patterns in running applications

**Features**:
- Start/stop tracking capability
- Pattern collection and analysis
- Thread-safe operation
- Reset functionality for clean measurements

### 4. MemoryAlignmentAnalyzer
**Purpose**: Analyzes struct memory layout and suggests optimizations

**Features**:
- Struct size and alignment analysis
- Field-by-field breakdown
- Padding calculation
- Optimization suggestions with field reordering

## Go's Size Class System

### Understanding Size Classes
Go uses a size class system similar to TCMalloc:

```
Size Class | Object Size | Objects/Page | Waste
-----------|-------------|--------------|-------
1          | 8 bytes     | 512          | 0%
2          | 16 bytes    | 256          | 0%
3          | 24 bytes    | 170          | 0.44%
4          | 32 bytes    | 128          | 0%
5          | 48 bytes    | 85           | 0.59%
...
```

### Common Wastage Scenarios
```go
// High waste examples
make([]byte, 25)  // Uses 32-byte class, wastes 7 bytes (21.9%)
make([]byte, 65)  // Uses 96-byte class, wastes 31 bytes (32.3%)
make([]byte, 129) // Uses 160-byte class, wastes 31 bytes (19.4%)

// Low waste examples
make([]byte, 32)  // Perfect fit, 0% waste
make([]byte, 64)  // Perfect fit, 0% waste
make([]byte, 96)  // Perfect fit, 0% waste
```

## Running the Exercises

### Basic Analysis
```bash
# Run the main demonstration
go run size_optimization.go

# This will show:
# - Size class efficiency analysis
# - Optimized size class generation
# - Struct alignment analysis
# - Real-world allocation simulation
```

### Comprehensive Benchmarks
```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark categories
go test -bench=BenchmarkSizeClassEfficiency -benchmem
go test -bench=BenchmarkStructAlignment -benchmem
go test -bench=BenchmarkAllocationPatterns -benchmem
go test -bench=BenchmarkRealWorldScenarios -benchmem
```

### Memory Profiling
```bash
# Profile memory usage
go test -bench=BenchmarkMemoryWaste -benchmem -memprofile=mem.prof
go tool pprof mem.prof

# Analyze allocation patterns
go test -bench=BenchmarkAllocationPatterns -trace=trace.out
go tool trace trace.out
```

### Correctness Testing
```bash
# Run all tests
go test -v

# Test with race detection
go test -race -v

# Test for memory leaks
go test -run=TestMemoryLeaks -v
```

## Key Metrics to Monitor

### Allocation Efficiency
- **Waste Percentage**: `(size_class - requested_size) / size_class * 100`
- **Hit Rate**: Frequency of allocations that fit size classes well
- **Fragmentation**: Internal vs. external fragmentation

### Performance Metrics
- **Allocations per operation**: Lower is better
- **Bytes allocated per operation**: Lower is better
- **GC pressure**: Fewer allocations = less GC work

### Struct Optimization Metrics
- **Padding bytes**: Wasted space due to alignment
- **Efficiency**: `utilized_bytes / total_size`
- **Cache line utilization**: How well structs fit in cache lines

## Expected Results

### Size Class Efficiency Benchmark
```
BenchmarkSizeClassEfficiency/Size_24-8    	50000000	        30 ns/op	      32 B/op	       1 allocs/op
BenchmarkSizeClassEfficiency/Size_48-8    	50000000	        30 ns/op	      48 B/op	       1 allocs/op
BenchmarkSizeClassEfficiency/Size_96-8    	50000000	        30 ns/op	      96 B/op	       1 allocs/op
BenchmarkSizeClassEfficiency/Size_128-8   	50000000	        30 ns/op	     128 B/op	       1 allocs/op
```

### Struct Alignment Impact
```
BenchmarkStructAlignment/BadAlignment-8       	100000000	        15 ns/op	       0 B/op	       0 allocs/op
BenchmarkStructAlignment/GoodAlignment-8      	100000000	        12 ns/op	       0 B/op	       0 allocs/op
BenchmarkStructAlignment/ArrayBadAlignment-8  	  500000	      3000 ns/op	    2400 B/op	       1 allocs/op
BenchmarkStructAlignment/ArrayGoodAlignment-8 	  500000	      2500 ns/op	    2000 B/op	       1 allocs/op
```

### Memory Waste Comparison
```
BenchmarkMemoryWaste/HighWasteScenario-8    	10000000	       150 ns/op	     448 B/op	       4 allocs/op
BenchmarkMemoryWaste/LowWasteScenario-8     	10000000	       120 ns/op	     384 B/op	       4 allocs/op
BenchmarkMemoryWaste/OptimizedSizes-8       	10000000	       100 ns/op	     352 B/op	       7 allocs/op
```

## Analysis Questions

### 1. Size Class Efficiency
- Which allocation sizes cause the most waste in your application?
- How does allocation frequency correlate with waste percentage?
- What's the trade-off between fewer size classes and lower waste?

### 2. Struct Optimization
- How much memory can be saved through field reordering?
- What's the performance impact of better struct alignment?
- How does struct size affect cache performance?

### 3. Real-World Patterns
- What are the most common allocation sizes in your application?
- How do allocation patterns change under different workloads?
- Which optimizations provide the biggest impact?

### 4. Performance Trade-offs
- When does size class optimization hurt performance?
- How does memory efficiency relate to GC pressure?
- What's the cost of tracking allocation patterns?

## Optimization Strategies

### 1. Size-Aware Allocation
```go
// Bad: Causes waste
buf := make([]byte, 100) // Uses 112-byte class, wastes 12 bytes

// Good: Aligns with size class
buf := make([]byte, 96)  // Perfect fit, no waste
```

### 2. Struct Field Ordering
```go
// Bad: Poor alignment
type BadStruct struct {
    A bool   // 1 byte + 7 padding
    B int64  // 8 bytes
    C bool   // 1 byte + 3 padding
    D int32  // 4 bytes
} // Total: 24 bytes

// Good: Optimal alignment
type GoodStruct struct {
    B int64  // 8 bytes
    D int32  // 4 bytes
    A bool   // 1 byte
    C bool   // 1 byte + 2 padding
} // Total: 16 bytes
```

### 3. Pool Size Optimization
```go
// Optimize pool sizes based on size classes
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 96) // Aligns with size class
    },
}
```

### 4. Batch Allocation
```go
// Instead of many small allocations
for i := 0; i < 1000; i++ {
    item := make([]Item, 1) // 1000 allocations
}

// Use batch allocation
items := make([]Item, 1000) // 1 allocation
```

## Common Pitfalls

### 1. Over-Optimization
```go
// Don't sacrifice readability for minor gains
type OverOptimized struct {
    // Fields ordered by size, not logical grouping
    BigField1   [100]byte
    BigField2   [50]byte
    MediumField int64
    SmallField1 bool
    SmallField2 bool
} // Hard to understand and maintain
```

### 2. Ignoring Access Patterns
```go
// Consider which fields are accessed together
type CacheUnfriendly struct {
    HotField1  int64    // Frequently accessed
    ColdField  [1000]byte // Rarely accessed
    HotField2  int64    // Frequently accessed
} // Hot fields separated by cold data
```

### 3. Premature Optimization
```go
// Measure first, optimize second
// Don't optimize without profiling data
```

## Advanced Techniques

### 1. Custom Size Classes
```go
// Implement application-specific size classes
type CustomAllocator struct {
    pools map[int]*sync.Pool // Size -> Pool
}

func (ca *CustomAllocator) Allocate(size int) []byte {
    sizeClass := ca.findOptimalSizeClass(size)
    pool := ca.pools[sizeClass]
    return pool.Get().([]byte)
}
```

### 2. Allocation Pattern Analysis
```go
// Track allocation patterns in production
type AllocationProfiler struct {
    patterns map[int]int64
    mutex    sync.RWMutex
}

func (ap *AllocationProfiler) Profile() {
    // Analyze patterns and suggest optimizations
}
```

### 3. Struct Layout Optimization
```go
// Use build tags for optimized layouts
//go:build optimize
type OptimizedStruct struct {
    // Optimized field order
}

//go:build !optimize
type OptimizedStruct struct {
    // Readable field order
}
```

## Integration with Applications

### Web Server Optimization
```go
// Optimize HTTP handler allocations
func optimizedHandler(w http.ResponseWriter, r *http.Request) {
    // Use size-class-aligned buffers
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf[:0])
    
    // Process request with optimized buffer
}
```

### JSON Processing Optimization
```go
// Optimize JSON parsing buffers
type JSONProcessor struct {
    parseBuffer []byte // Pre-allocated, size-class-aligned
}

func (jp *JSONProcessor) Parse(data []byte) {
    // Reuse optimally-sized buffer
}
```

### Database Query Optimization
```go
// Optimize result set buffers
type QueryResult struct {
    rows []Row // Size based on typical result sets
}
```

## Monitoring and Alerting

### 1. Allocation Waste Monitoring
```go
func monitorAllocationWaste() {
    optimizer := NewSizeClassOptimizer()
    // ... collect data ...
    
    report := optimizer.AnalyzeWastage()
    wastePercentage := float64(report.TotalWastage) / float64(report.TotalAllocations)
    
    if wastePercentage > 0.20 { // Alert if >20% waste
        log.Printf("High allocation waste: %.2f%%", wastePercentage*100)
    }
}
```

### 2. Struct Size Monitoring
```go
func validateStructSizes() {
    analyzer := &MemoryAlignmentAnalyzer{}
    
    structs := []interface{}{
        MyStruct{},
        AnotherStruct{},
    }
    
    for _, s := range structs {
        analysis := analyzer.AnalyzeStruct(s)
        if analysis.Efficiency < 0.75 { // Alert if <75% efficient
            log.Printf("Inefficient struct %s: %.2f%% efficient", 
                analysis.Name, analysis.Efficiency*100)
        }
    }
}
```

## Next Steps

After completing this exercise:
1. Move to Exercise 01-04: Memory Alignment and Padding
2. Apply size class optimization to your own applications
3. Implement allocation pattern monitoring in production
4. Study Go's runtime allocator source code for deeper insights

## Resources

- [Go Memory Allocator Design](https://golang.org/src/runtime/malloc.go)
- [TCMalloc: Thread-Caching Malloc](https://google.github.io/tcmalloc/design.html)
- [Go Memory Management](https://blog.golang.org/ismmkeynote)
- [Struct Alignment in Go](https://dave.cheney.net/2015/10/09/padding-is-hard)
- [Go Memory Layout](https://research.swtch.com/godata)