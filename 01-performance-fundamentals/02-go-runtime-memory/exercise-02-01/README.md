# Exercise 02-01: Escape Analysis Investigation

**Objective:** Understand when variables escape to the heap and how to control allocation behavior.

**Duration:** 90-120 minutes

## Tasks

### Task 1: Escape Analysis Basics
1. Examine the `escape_analysis.go` file
2. Understand different allocation scenarios:
   - Stack allocation (local variables)
   - Heap allocation (escaped variables)
   - Interface allocation behavior
   - Slice and map allocation patterns

### Task 2: Compiler Analysis
1. Use Go's escape analysis to understand allocation decisions:
   ```bash
   go build -gcflags="-m" escape_analysis.go
   go build -gcflags="-m -m" escape_analysis.go  # More verbose
   ```
2. Analyze compiler output for each function
3. Understand why variables escape or stay on stack

### Task 3: Benchmark Different Allocation Patterns
1. Run benchmarks in `benchmark_test.go`:
   ```bash
   go test -bench=. -benchmem
   ```
2. Compare performance between:
   - Stack vs heap allocation
   - Value vs pointer receivers
   - Interface vs concrete types
   - Different data structure patterns

### Task 4: Optimization Techniques
1. Implement optimizations to reduce heap allocations:
   - Avoid unnecessary pointer returns
   - Use value receivers when possible
   - Optimize interface usage
   - Control slice and map allocation

### Task 5: Memory Profiling
1. Generate memory profiles:
   ```bash
   go test -bench=. -memprofile=mem.prof
   go tool pprof mem.prof
   ```
2. Analyze allocation patterns
3. Identify optimization opportunities

## Key Learning Points
- When and why variables escape to the heap
- Impact of escape analysis on performance
- Techniques to control allocation behavior
- Understanding compiler optimization decisions
- Memory allocation profiling and analysis

## Expected Outcomes
- Deep understanding of Go's escape analysis
- Ability to write allocation-efficient code
- Skills in memory profiling and optimization
- Knowledge of stack vs heap trade-offs

## Files
- `escape_analysis.go` - Functions demonstrating different allocation patterns
- `benchmark_test.go` - Benchmarks for allocation analysis

## Escape Analysis Rules
- Variables that don't escape stay on stack
- Returned pointers cause escape
- Large objects may escape regardless
- Interface assignments often cause escape
- Closures can cause escape

## Performance Impact
- Stack allocation: ~1ns
- Heap allocation: ~10-100ns + GC overhead
- Heap allocations require garbage collection
- Stack allocations are automatically freed

## Next Steps
Proceed to Exercise 02-02 to learn about memory allocation patterns and optimization strategies.