# Exercise 01-02: Performance Anti-patterns and Optimization

**Objective:** Identify and fix common performance anti-patterns in Go code.

**Duration:** 90-120 minutes

## Tasks

### Task 1: Analyze Anti-patterns
1. Examine `antipatterns.go` for performance issues
2. Identify the following anti-patterns:
   - Inefficient string building
   - Poor memory allocation patterns
   - Suboptimal data structure usage
   - Unnecessary allocations

### Task 2: Benchmark Current Performance
1. Run the provided benchmarks in `benchmark_test.go`
2. Record baseline performance metrics:
   ```bash
   go test -bench=. -benchmem
   ```
3. Analyze memory allocation patterns
4. Identify the most expensive operations

### Task 3: Implement Optimizations
1. Study the optimized implementations in `improved.go`
2. Compare the approaches used:
   - String building with `strings.Builder`
   - Pre-allocated slices
   - Efficient data structures
   - Reduced allocations

### Task 4: Measure Improvements
1. Run benchmarks again after studying optimizations
2. Compare performance improvements:
   ```bash
   go test -bench=. -benchmem > after.txt
   # Compare with baseline results
   ```
3. Calculate performance gains (throughput, allocations)

### Task 5: Validation
1. Run correctness tests to ensure optimizations don't break functionality
2. Verify that optimized code produces the same results

## Key Learning Points
- String concatenation performance (`+` vs `strings.Builder`)
- Slice growth patterns and pre-allocation
- Map vs slice performance trade-offs
- Memory allocation reduction techniques
- Benchmark interpretation and analysis

## Expected Outcomes
- Understanding of common Go performance anti-patterns
- Ability to identify inefficient code patterns
- Knowledge of optimization techniques
- Experience with benchmark-driven optimization

## Files
- `antipatterns.go` - Code with performance issues
- `improved.go` - Optimized implementations
- `benchmark_test.go` - Comprehensive benchmarks with validation

## Performance Targets
After optimization, you should see:
- 5-10x improvement in string building operations
- 2-3x reduction in memory allocations
- Significant improvement in lookup operations

## Next Steps
Proceed to Exercise 01-03 to learn about measurement techniques and methodology.