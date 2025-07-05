# Exercise 01-01: Memory Allocator Analysis and Size Classes

**Objective:** Understand Go's memory allocator internals and size class system.

**Duration:** 120-150 minutes

## Tasks

### Task 1: Memory Allocator Fundamentals
1. Examine `allocator_analysis.go` to understand:
   - Go's memory allocator architecture
   - Size class system
   - Thread-local caches (mcache)
   - Central lists (mcentral)
   - Page heap (mheap)

### Task 2: Size Class Analysis
1. Study `size_classes.go` for:
   - Size class definitions and ranges
   - Memory overhead calculations
   - Allocation efficiency analysis
   - Fragmentation patterns

### Task 3: Allocator Performance Benchmarking
1. Run benchmarks in `benchmark_test.go`:
   ```bash
   go test -bench=. -benchmem
   ```
2. Analyze allocation patterns for different object sizes:
   - Small objects (< 32KB)
   - Large objects (> 32KB)
   - Tiny objects (< 16 bytes)
   - Size class alignment effects

### Task 4: Memory Overhead Analysis
1. Calculate memory overhead for different allocation sizes
2. Understand internal fragmentation
3. Analyze allocation efficiency across size classes
4. Identify optimal allocation sizes

### Task 5: Allocator Optimization Strategies
1. Implement allocation-aware optimizations:
   - Size-conscious data structure design
   - Allocation batching techniques
   - Memory pool strategies
   - Cache-friendly allocation patterns

### Task 6: Real-world Application
1. Apply allocator knowledge to optimize:
   - Data structure layouts
   - Batch processing algorithms
   - Memory-intensive applications
   - High-frequency allocation scenarios

## Key Learning Points
- Go's memory allocator architecture (tcmalloc-based)
- Size class system and allocation efficiency
- Memory overhead and fragmentation analysis
- Allocation performance characteristics
- Optimization strategies based on allocator behavior

## Expected Outcomes
- Deep understanding of Go's memory allocator
- Ability to optimize allocation patterns
- Knowledge of memory overhead and fragmentation
- Skills in allocation-aware programming

## Files
- `allocator_analysis.go` - Memory allocator analysis and utilities
- `size_classes.go` - Size class system analysis
- `benchmark_test.go` - Allocator performance benchmarks

## Size Class System
Go uses 67 size classes ranging from 8 bytes to 32KB:
- **Tiny allocations** (< 16 bytes): Special handling
- **Small allocations** (16 bytes - 32KB): Size classes
- **Large allocations** (> 32KB): Direct from heap

## Allocation Performance Characteristics
- **Tiny objects**: ~5-10ns (combined allocation)
- **Small objects**: ~10-50ns (size class allocation)
- **Large objects**: ~100-1000ns (direct heap allocation)
- **Thread-local cache hit**: ~5-15ns
- **Central list access**: ~50-200ns
- **Page heap access**: ~200-1000ns

## Optimization Guidelines
1. **Size Awareness**: Choose sizes that align with size classes
2. **Batch Allocation**: Reduce allocator overhead
3. **Pool Reuse**: Minimize allocation frequency
4. **Layout Optimization**: Reduce memory overhead
5. **Cache Locality**: Improve allocation patterns

## Next Steps
Proceed to Exercise 01-02 to learn about memory pools and object pooling strategies.