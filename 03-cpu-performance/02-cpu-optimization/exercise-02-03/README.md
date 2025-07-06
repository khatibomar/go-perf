# Exercise 02-03: Sorting Algorithm Comparison

**Objective:** Compare and optimize different sorting algorithms for various data patterns and sizes.

**Duration:** 45-60 minutes  
**Difficulty:** Intermediate

## Learning Goals

By completing this exercise, you will:
- ✅ Understand performance characteristics of different sorting algorithms
- ✅ Implement optimized versions of common sorting algorithms
- ✅ Analyze algorithm performance across different data patterns
- ✅ Apply hybrid sorting strategies for optimal performance
- ✅ Measure and compare CPU usage patterns

## Background

Sorting is a fundamental operation in computer science, and choosing the right algorithm can significantly impact performance. This exercise explores various sorting algorithms and their optimization techniques.

### Algorithms to Implement

1. **Quick Sort**
   - Standard implementation
   - Optimized pivot selection
   - Hybrid with insertion sort

2. **Merge Sort**
   - Standard recursive implementation
   - Iterative bottom-up approach
   - In-place optimization

3. **Heap Sort**
   - Standard implementation
   - Optimized heapify

4. **Radix Sort**
   - LSD (Least Significant Digit)
   - MSD (Most Significant Digit)
   - Optimized for different data types

5. **Tim Sort**
   - Hybrid merge sort
   - Run detection optimization
   - Galloping mode

## Tasks

### Task 1: Algorithm Implementation
Implement all sorting algorithms with proper optimizations:
- Focus on cache-friendly memory access patterns
- Minimize allocations in hot paths
- Use appropriate data structures

### Task 2: Performance Analysis
Benchmark algorithms across different scenarios:
- Various data sizes (100, 1K, 10K, 100K, 1M elements)
- Different data patterns (random, sorted, reverse, partially sorted)
- Different data types (int32, int64, string, custom structs)

### Task 3: Optimization Techniques
Apply advanced optimization techniques:
- SIMD instructions where applicable
- Branch prediction optimization
- Cache-aware implementations
- Parallel sorting for large datasets

### Task 4: Hybrid Strategies
Implement hybrid sorting strategies:
- Algorithm selection based on data characteristics
- Adaptive threshold tuning
- Runtime performance monitoring

## Files Overview

- `algorithms/` - Sorting algorithm implementations
  - `quicksort.go` - Quick sort variants
  - `mergesort.go` - Merge sort implementations
  - `heapsort.go` - Heap sort implementation
  - `radixsort.go` - Radix sort variants
  - `timsort.go` - Tim sort implementation

- `benchmarks/` - Performance benchmarking
  - `sort_bench_test.go` - Comprehensive benchmarks
  - `data_generators.go` - Test data generation
  - `profiling_test.go` - CPU profiling tests

- `analysis/` - Performance analysis tools
  - `analyzer.go` - Algorithm performance analyzer
  - `visualizer.go` - Performance visualization
  - `reporter.go` - Benchmark report generator

- `hybrid/` - Hybrid sorting strategies
  - `adaptive.go` - Adaptive algorithm selection
  - `parallel.go` - Parallel sorting implementations
  - `optimizer.go` - Runtime optimization

- `main.go` - Demonstration and testing
- `sort_test.go` - Correctness tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific algorithm benchmarks
go test -bench=BenchmarkQuickSort -benchmem

# Profile CPU usage
go test -bench=BenchmarkSort -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Generate performance report
go run . --analyze --report=performance.html

# Test correctness
go test -v
```

## Expected Results

Performance characteristics you should observe:

| Algorithm | Best Case | Average Case | Worst Case | Space | Stable |
|-----------|-----------|--------------|------------|-------|--------|
| Quick Sort | O(n log n) | O(n log n) | O(n²) | O(log n) | No |
| Merge Sort | O(n log n) | O(n log n) | O(n log n) | O(n) | Yes |
| Heap Sort | O(n log n) | O(n log n) | O(n log n) | O(1) | No |
| Radix Sort | O(d×n) | O(d×n) | O(d×n) | O(n+k) | Yes |
| Tim Sort | O(n) | O(n log n) | O(n log n) | O(n) | Yes |

### Performance Targets

- **Small arrays (≤100):** Insertion sort should be fastest
- **Medium arrays (100-10K):** Quick sort should excel
- **Large arrays (≥10K):** Merge sort or Tim sort should perform best
- **Integer data:** Radix sort should outperform comparison sorts
- **Nearly sorted data:** Tim sort should show significant advantage

## Key Metrics to Monitor

1. **Throughput:** Elements sorted per second
2. **Latency:** Time to sort specific dataset sizes
3. **CPU Usage:** Processor utilization patterns
4. **Cache Performance:** L1/L2/L3 cache hit rates
5. **Branch Prediction:** Misprediction rates
6. **Memory Allocations:** Allocation patterns and GC pressure

## Optimization Techniques

### 1. Cache-Friendly Access Patterns
```go
// Bad: Random memory access
for i := 0; i < n; i++ {
    process(data[indices[i]])
}

// Good: Sequential access
for i := 0; i < n; i++ {
    process(data[i])
}
```

### 2. Branch Prediction Optimization
```go
// Bad: Unpredictable branches
if data[i] < pivot {
    // ...
} else {
    // ...
}

// Good: Branchless operations where possible
result := (data[i] < pivot) * leftValue + (data[i] >= pivot) * rightValue
```

### 3. SIMD Optimization
```go
// Use SIMD instructions for parallel comparisons
// (Implementation depends on specific use case)
```

## Success Criteria

- [ ] Implement all 5 sorting algorithms correctly
- [ ] Achieve expected time complexity characteristics
- [ ] Demonstrate performance advantages for appropriate use cases
- [ ] Show cache-friendly optimizations improve performance
- [ ] Implement working hybrid strategy
- [ ] Generate comprehensive performance analysis

## Advanced Challenges

1. **SIMD Optimization**: Use assembly or compiler intrinsics for SIMD
2. **GPU Sorting**: Implement GPU-accelerated sorting
3. **External Sorting**: Handle datasets larger than memory
4. **Distributed Sorting**: Implement distributed sorting algorithm
5. **Real-time Constraints**: Optimize for real-time systems

## Next Steps

After completing this exercise:
1. Apply learnings to your specific use cases
2. Experiment with domain-specific optimizations
3. Explore parallel and distributed sorting
4. Study advanced algorithms like sample sort
5. Investigate hardware-specific optimizations

---

**Estimated Completion Time:** 45-60 minutes  
**Prerequisites:** Exercises 02-01 and 02-02  
**Next Exercise:** [Exercise 02-04: Search Optimization Techniques](../exercise-02-04/README.md)