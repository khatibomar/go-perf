# Exercise 01-02: Custom Memory Pools Implementation

## Overview
This exercise demonstrates the implementation and benchmarking of custom memory pools in Go. You'll learn how to create efficient memory management systems that reduce allocation overhead and garbage collection pressure.

## Learning Objectives
- Implement various types of memory pools (fixed-size, size-class, stack-based)
- Understand object pooling patterns for common Go types
- Measure and compare pool performance against standard allocation
- Analyze memory usage patterns and optimization opportunities
- Learn about pool contention and scaling characteristics

## Files Structure
```
exercise-01-02/
├── README.md                 # This file
├── memory_pools.go          # Custom allocator implementations
├── object_pools.go          # Object pool implementations
└── pool_benchmark_test.go   # Comprehensive benchmarks and tests
```

## Custom Allocators Implemented

### 1. FixedSizePool
- **Purpose**: Manages fixed-size memory blocks
- **Use Case**: When you know the exact size needed (e.g., network packets)
- **Benefits**: Zero fragmentation, predictable performance
- **Limitations**: Only works for single size

### 2. SizeClassAllocator
- **Purpose**: Multiple pools for different size classes
- **Use Case**: Variable-size allocations with known patterns
- **Benefits**: Reduced fragmentation, good for mixed workloads
- **Implementation**: Uses Go's size class system as reference

### 3. StackAllocator
- **Purpose**: Linear allocation with marker-based deallocation
- **Use Case**: Temporary allocations with known lifetimes
- **Benefits**: Extremely fast allocation, automatic cleanup
- **Limitations**: LIFO deallocation pattern required

### 4. PoolAllocator (sync.Pool wrapper)
- **Purpose**: Wrapper around Go's sync.Pool with statistics
- **Use Case**: General-purpose object reuse
- **Benefits**: Integrates with Go's GC, automatic cleanup

## Object Pools Implemented

### 1. BufferPool
- **Type**: `[]byte` buffers with size classes
- **Features**: Automatic size class selection, growth handling
- **Statistics**: Hit rate, allocation counts, memory usage

### 2. StringBuilderPool
- **Type**: `strings.Builder` objects
- **Features**: Capacity management, automatic reset
- **Use Case**: String concatenation operations

### 3. WorkerPool
- **Type**: Reusable worker goroutines
- **Features**: Task queuing, lifecycle management
- **Use Case**: Avoiding goroutine creation overhead

### 4. ConnectionPool
- **Type**: Generic connection objects
- **Features**: Resource lifecycle, cleanup callbacks
- **Use Case**: Database connections, network connections

## Running the Exercises

### Basic Benchmarks
```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark categories
go test -bench=BenchmarkBufferPool -benchmem
go test -bench=BenchmarkCustomAllocators -benchmem
go test -bench=BenchmarkWorkerPool -benchmem
```

### Memory Profiling
```bash
# Generate memory profile
go test -bench=BenchmarkBufferPool -benchmem -memprofile=mem.prof

# Analyze memory profile
go tool pprof mem.prof
```

### CPU Profiling
```bash
# Generate CPU profile
go test -bench=BenchmarkCustomAllocators -cpuprofile=cpu.prof

# Analyze CPU profile
go tool pprof cpu.prof
```

### Correctness Tests
```bash
# Run correctness tests
go test -v -run=Test

# Test with race detection
go test -race -v
```

## Key Metrics to Analyze

### Performance Metrics
- **Allocations per operation**: Lower is better
- **Bytes allocated per operation**: Lower is better
- **Nanoseconds per operation**: Lower is better
- **Hit rate**: Higher is better (for pools)

### Memory Metrics
- **Peak memory usage**: Monitor with `-memprofile`
- **GC pressure**: Check `runtime.ReadMemStats()`
- **Memory growth**: Look for leaks in long-running tests

### Concurrency Metrics
- **Contention**: Performance under multiple goroutines
- **Scalability**: Performance vs. number of goroutines
- **Lock contention**: Use `-blockprofile` if applicable

## Expected Results

### Buffer Pool Performance
```
BenchmarkBufferPool/WithPool-8         	10000000	       120 ns/op	       0 B/op	       0 allocs/op
BenchmarkBufferPool/WithoutPool-8      	 3000000	       450 ns/op	    1024 B/op	       1 allocs/op
BenchmarkBufferPool/SyncPool-8         	 8000000	       150 ns/op	       0 B/op	       0 allocs/op
```

### Custom Allocator Performance
```
BenchmarkCustomAllocators/FixedSizePool-8      	50000000	        30 ns/op	       0 B/op	       0 allocs/op
BenchmarkCustomAllocators/SizeClassAllocator-8 	20000000	        80 ns/op	       0 B/op	       0 allocs/op
BenchmarkCustomAllocators/StackAllocator-8     	100000000	        15 ns/op	       0 B/op	       0 allocs/op
BenchmarkCustomAllocators/StandardAllocation-8 	10000000	       150 ns/op	      64 B/op	       1 allocs/op
```

## Analysis Questions

1. **Pool Efficiency**:
   - Which pool type performs best for your use case?
   - How does hit rate affect overall performance?
   - What's the overhead of pool management vs. benefits?

2. **Memory Patterns**:
   - How do different allocation patterns affect pool performance?
   - What's the relationship between pool size and performance?
   - How does GC pressure change with pooling?

3. **Concurrency**:
   - How does pool performance scale with goroutine count?
   - Where do you see contention bottlenecks?
   - How do different pool implementations handle contention?

4. **Trade-offs**:
   - When is pooling worth the complexity?
   - What are the memory vs. CPU trade-offs?
   - How do you balance pool size vs. memory usage?

## Common Pitfalls

### 1. Pool Size Tuning
```go
// Bad: Pool too small, frequent misses
pool := NewBufferPool() // Default size classes

// Good: Tune based on actual usage patterns
pool := NewBufferPoolWithSizes([]int{64, 256, 1024, 4096})
```

### 2. Memory Leaks
```go
// Bad: Not resetting pooled objects
func (p *BufferPool) Put(buf []byte) {
    p.pool.Put(buf) // May retain references
}

// Good: Reset before returning to pool
func (p *BufferPool) Put(buf []byte) {
    buf = buf[:0] // Reset length
    p.pool.Put(buf)
}
```

### 3. Incorrect Size Classes
```go
// Bad: Size classes don't match usage
sizes := []int{10, 20, 30} // Too granular

// Good: Based on actual allocation patterns
sizes := []int{64, 256, 1024, 4096} // Power of 2, common sizes
```

## Advanced Experiments

### 1. Custom Size Class Analysis
```bash
# Analyze your application's allocation patterns
go build -gcflags="-m" your_app.go
go tool trace trace.out
```

### 2. Pool Warming
```go
// Pre-populate pools for consistent performance
func (p *BufferPool) Warm(count int, size int) {
    for i := 0; i < count; i++ {
        buf := make([]byte, 0, size)
        p.Put(buf)
    }
}
```

### 3. Adaptive Pool Sizing
```go
// Implement pools that adjust size based on usage
type AdaptivePool struct {
    pool     sync.Pool
    hitRate  float64
    maxSize  int
    // ... adaptive logic
}
```

## Integration with Real Applications

### Web Server Example
```go
// Use buffer pools for HTTP response building
var responsePool = NewBufferPool()

func handler(w http.ResponseWriter, r *http.Request) {
    buf := responsePool.Get(4096)
    defer responsePool.Put(buf)
    
    // Build response in pooled buffer
    buf = append(buf, "HTTP response data"...)
    w.Write(buf)
}
```

### JSON Processing Example
```go
// Use string builder pools for JSON marshaling
var jsonPool = NewStringBuilderPool(1024)

func marshalJSON(data interface{}) ([]byte, error) {
    sb := jsonPool.Get()
    defer jsonPool.Put(sb)
    
    // Build JSON in pooled string builder
    // ... JSON marshaling logic
    return []byte(sb.String()), nil
}
```

## Next Steps

After completing this exercise:
1. Move to Exercise 01-03: Size Class Optimization
2. Experiment with your own application's allocation patterns
3. Consider implementing custom pools for your specific use cases
4. Study the Go runtime's allocator implementation for deeper insights

## Resources

- [Go Memory Allocator Design](https://golang.org/src/runtime/malloc.go)
- [TCMalloc Paper](https://google.github.io/tcmalloc/design.html)
- [Go sync.Pool Documentation](https://pkg.go.dev/sync#Pool)
- [Memory Management in Go](https://blog.golang.org/ismmkeynote)