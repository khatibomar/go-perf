# Exercise 04-01: Atomic Operations Benchmarking

**Objective:** Master atomic operations and understand their performance characteristics across different scenarios.

**Duration:** 45-60 minutes  
**Difficulty:** Intermediate

## Learning Goals

By completing this exercise, you will:
- ✅ Understand atomic operation performance characteristics
- ✅ Master memory ordering semantics (Acquire, Release, SeqCst)
- ✅ Analyze cache coherency impact on atomic operations
- ✅ Implement architecture-specific optimizations
- ✅ Measure and optimize false sharing effects

## Background

Atomic operations are the foundation of lock-free programming. Understanding their performance characteristics, memory ordering semantics, and interaction with CPU cache hierarchies is crucial for building high-performance concurrent systems.

### Atomic Operation Types

1. **Load/Store Operations**
   - Simple read/write with memory ordering
   - Fastest atomic operations
   - Foundation for other operations

2. **Compare-and-Swap (CAS)**
   - Atomic conditional update
   - Core of most lock-free algorithms
   - Higher latency than load/store

3. **Fetch-and-Add/Sub**
   - Atomic arithmetic operations
   - Useful for counters and accumulators
   - Moderate performance cost

4. **Bitwise Operations**
   - Atomic AND, OR, XOR operations
   - Specialized use cases
   - Similar performance to arithmetic

### Memory Ordering

1. **Relaxed Ordering**
   - No synchronization guarantees
   - Highest performance
   - Suitable for counters and statistics

2. **Acquire/Release Ordering**
   - Synchronization without full ordering
   - Good balance of performance and correctness
   - Common in lock-free data structures

3. **Sequential Consistency**
   - Strongest ordering guarantee
   - Highest overhead
   - Easiest to reason about

## Tasks

### Task 1: Basic Atomic Operations
Benchmark fundamental atomic operations:
- Load and Store with different memory ordering
- Compare-and-Swap variations
- Fetch-and-Add operations
- Bitwise atomic operations

### Task 2: Memory Ordering Analysis
Analyze memory ordering impact:
- Performance comparison across ordering types
- Correctness verification with different orderings
- Memory barrier overhead measurement
- Architecture-specific behavior

### Task 3: Cache Coherency Effects
Study cache coherency impact:
- False sharing detection and mitigation
- Cache line alignment optimization
- NUMA-aware atomic operations
- Multi-core scaling analysis

### Task 4: Advanced Optimizations
Implement advanced optimization techniques:
- Batching atomic operations
- Exponential backoff strategies
- Architecture-specific intrinsics
- Compiler optimization analysis

## Files Overview

- `atomic/` - Atomic operation implementations
  - `basic_ops.go` - Basic atomic operations
  - `memory_ordering.go` - Memory ordering examples
  - `cas_operations.go` - Compare-and-swap variants
  - `arithmetic_ops.go` - Atomic arithmetic operations

- `benchmarks/` - Performance benchmarks
  - `basic_bench_test.go` - Basic operation benchmarks
  - `ordering_bench_test.go` - Memory ordering benchmarks
  - `contention_bench_test.go` - Contention analysis
  - `scaling_bench_test.go` - Multi-core scaling tests

- `cache/` - Cache optimization
  - `false_sharing.go` - False sharing examples
  - `alignment.go` - Cache line alignment
  - `numa_aware.go` - NUMA-aware implementations
  - `locality_test.go` - Cache locality tests

- `optimization/` - Advanced optimizations
  - `batching.go` - Batch operation strategies
  - `backoff.go` - Exponential backoff
  - `intrinsics.go` - Architecture-specific code
  - `compiler_opts.go` - Compiler optimization analysis

- `correctness/` - Correctness verification
  - `ordering_test.go` - Memory ordering correctness
  - `race_detector.go` - Race condition detection
  - `linearizability.go` - Linearizability testing

- `main.go` - Demonstration and analysis
- `atomic_test.go` - Comprehensive tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific atomic operation benchmarks
go test -bench=BenchmarkAtomicLoad -benchmem

# Profile CPU usage
go test -bench=BenchmarkAtomic -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Analyze memory ordering
go test -bench=BenchmarkOrdering -benchmem

# Test with race detector
go test -race

# Run demonstration
go run .

# Assembly analysis
go build -gcflags=-S main.go
```

## Implementation Guidelines

### 1. Basic Atomic Operations
```go
package atomic

import (
    "sync/atomic"
    "unsafe"
)

// AtomicCounter demonstrates basic atomic operations
type AtomicCounter struct {
    value int64
    _     [7]int64 // Cache line padding
}

func (c *AtomicCounter) Load() int64 {
    return atomic.LoadInt64(&c.value)
}

func (c *AtomicCounter) Store(val int64) {
    atomic.StoreInt64(&c.value, val)
}

func (c *AtomicCounter) Add(delta int64) int64 {
    return atomic.AddInt64(&c.value, delta)
}

func (c *AtomicCounter) CompareAndSwap(old, new int64) bool {
    return atomic.CompareAndSwapInt64(&c.value, old, new)
}

func (c *AtomicCounter) Swap(new int64) int64 {
    return atomic.SwapInt64(&c.value, new)
}

// Pointer-based atomic operations
type AtomicPointer struct {
    ptr unsafe.Pointer
    _   [7]uintptr // Cache line padding
}

func (p *AtomicPointer) Load() unsafe.Pointer {
    return atomic.LoadPointer(&p.ptr)
}

func (p *AtomicPointer) Store(ptr unsafe.Pointer) {
    atomic.StorePointer(&p.ptr, ptr)
}

func (p *AtomicPointer) CompareAndSwap(old, new unsafe.Pointer) bool {
    return atomic.CompareAndSwapPointer(&p.ptr, old, new)
}

func (p *AtomicPointer) Swap(new unsafe.Pointer) unsafe.Pointer {
    return atomic.SwapPointer(&p.ptr, new)
}
```

### 2. Memory Ordering Examples
```go
package atomic

import (
    "sync/atomic"
    "runtime"
)

// MemoryOrderingDemo demonstrates different memory ordering semantics
type MemoryOrderingDemo struct {
    flag int32
    data int32
}

// RelaxedOrdering - No synchronization guarantees
func (m *MemoryOrderingDemo) RelaxedWrite(value int32) {
    atomic.StoreInt32(&m.data, value)
    atomic.StoreInt32(&m.flag, 1)
}

func (m *MemoryOrderingDemo) RelaxedRead() (int32, bool) {
    if atomic.LoadInt32(&m.flag) == 1 {
        return atomic.LoadInt32(&m.data), true
    }
    return 0, false
}

// AcquireReleaseOrdering - Synchronization semantics
func (m *MemoryOrderingDemo) AcquireReleaseWrite(value int32) {
    atomic.StoreInt32(&m.data, value)
    // Release: all previous operations complete before this store
    atomic.StoreInt32(&m.flag, 1)
}

func (m *MemoryOrderingDemo) AcquireReleaseRead() (int32, bool) {
    // Acquire: all subsequent operations happen after this load
    if atomic.LoadInt32(&m.flag) == 1 {
        return atomic.LoadInt32(&m.data), true
    }
    return 0, false
}

// SequentialConsistency - Strongest ordering
func (m *MemoryOrderingDemo) SeqCstWrite(value int32) {
    atomic.StoreInt32(&m.data, value)
    runtime.KeepAlive(m) // Prevent reordering
    atomic.StoreInt32(&m.flag, 1)
}

func (m *MemoryOrderingDemo) SeqCstRead() (int32, bool) {
    flag := atomic.LoadInt32(&m.flag)
    runtime.KeepAlive(m) // Prevent reordering
    if flag == 1 {
        return atomic.LoadInt32(&m.data), true
    }
    return 0, false
}
```

### 3. False Sharing Mitigation
```go
package cache

import (
    "sync/atomic"
    "unsafe"
)

const CacheLineSize = 64

// FalseSharingExample demonstrates false sharing problem
type FalseSharingExample struct {
    counter1 int64
    counter2 int64 // Same cache line as counter1
}

func (f *FalseSharingExample) IncrementCounter1() {
    atomic.AddInt64(&f.counter1, 1)
}

func (f *FalseSharingExample) IncrementCounter2() {
    atomic.AddInt64(&f.counter2, 1)
}

// CacheAlignedCounters prevents false sharing
type CacheAlignedCounters struct {
    counter1 int64
    _        [CacheLineSize/8 - 1]int64 // Padding
    counter2 int64
    _        [CacheLineSize/8 - 1]int64 // Padding
}

func (c *CacheAlignedCounters) IncrementCounter1() {
    atomic.AddInt64(&c.counter1, 1)
}

func (c *CacheAlignedCounters) IncrementCounter2() {
    atomic.AddInt64(&c.counter2, 1)
}

// DynamicAlignment for runtime cache line alignment
type DynamicAlignedCounter struct {
    data []byte
}

func NewDynamicAlignedCounter() *DynamicAlignedCounter {
    // Allocate extra space for alignment
    data := make([]byte, CacheLineSize*2)
    
    // Align to cache line boundary
    offset := CacheLineSize - (uintptr(unsafe.Pointer(&data[0])) % CacheLineSize)
    
    return &DynamicAlignedCounter{
        data: data[offset:],
    }
}

func (d *DynamicAlignedCounter) counter() *int64 {
    return (*int64)(unsafe.Pointer(&d.data[0]))
}

func (d *DynamicAlignedCounter) Increment() {
    atomic.AddInt64(d.counter(), 1)
}

func (d *DynamicAlignedCounter) Load() int64 {
    return atomic.LoadInt64(d.counter())
}
```

### 4. Exponential Backoff
```go
package optimization

import (
    "runtime"
    "sync/atomic"
    "time"
)

// ExponentialBackoff implements exponential backoff for CAS loops
type ExponentialBackoff struct {
    minDelay time.Duration
    maxDelay time.Duration
    factor   float64
    current  time.Duration
}

func NewExponentialBackoff(min, max time.Duration) *ExponentialBackoff {
    return &ExponentialBackoff{
        minDelay: min,
        maxDelay: max,
        factor:   2.0,
        current:  min,
    }
}

func (b *ExponentialBackoff) Backoff() {
    if b.current < b.maxDelay {
        time.Sleep(b.current)
        b.current = time.Duration(float64(b.current) * b.factor)
        if b.current > b.maxDelay {
            b.current = b.maxDelay
        }
    } else {
        time.Sleep(b.maxDelay)
    }
}

func (b *ExponentialBackoff) Reset() {
    b.current = b.minDelay
}

// CASWithBackoff demonstrates CAS with exponential backoff
func CASWithBackoff(addr *int64, old, new int64) bool {
    backoff := NewExponentialBackoff(time.Nanosecond, time.Microsecond)
    
    for {
        if atomic.CompareAndSwapInt64(addr, old, new) {
            return true
        }
        
        // Check if value changed (someone else succeeded)
        current := atomic.LoadInt64(addr)
        if current != old {
            return false
        }
        
        backoff.Backoff()
    }
}

// AdaptiveBackoff adjusts backoff based on contention
type AdaptiveBackoff struct {
    failures    int64
    successes   int64
    currentWait time.Duration
    minWait     time.Duration
    maxWait     time.Duration
}

func NewAdaptiveBackoff(min, max time.Duration) *AdaptiveBackoff {
    return &AdaptiveBackoff{
        minWait:     min,
        maxWait:     max,
        currentWait: min,
    }
}

func (a *AdaptiveBackoff) Success() {
    atomic.AddInt64(&a.successes, 1)
    
    // Reduce wait time on success
    if a.currentWait > a.minWait {
        a.currentWait = time.Duration(float64(a.currentWait) * 0.9)
        if a.currentWait < a.minWait {
            a.currentWait = a.minWait
        }
    }
}

func (a *AdaptiveBackoff) Failure() {
    atomic.AddInt64(&a.failures, 1)
    
    // Increase wait time on failure
    if a.currentWait < a.maxWait {
        a.currentWait = time.Duration(float64(a.currentWait) * 1.5)
        if a.currentWait > a.maxWait {
            a.currentWait = a.maxWait
        }
    }
    
    time.Sleep(a.currentWait)
}

func (a *AdaptiveBackoff) GetStats() (int64, int64) {
    return atomic.LoadInt64(&a.successes), atomic.LoadInt64(&a.failures)
}
```

### 5. NUMA-Aware Operations
```go
package cache

import (
    "runtime"
    "sync/atomic"
)

// NUMACounter implements NUMA-aware atomic counter
type NUMACounter struct {
    counters []CacheAlignedCounter
    numNodes int
}

type CacheAlignedCounter struct {
    value int64
    _     [7]int64 // Cache line padding
}

func NewNUMACounter() *NUMACounter {
    numNodes := runtime.NumCPU() // Simplified: use CPU count as proxy
    if numNodes > 8 {
        numNodes = 8 // Limit to reasonable number
    }
    
    return &NUMACounter{
        counters: make([]CacheAlignedCounter, numNodes),
        numNodes: numNodes,
    }
}

func (n *NUMACounter) getNodeIndex() int {
    // Simplified NUMA node detection
    // In practice, you'd use syscalls or cgo to get actual NUMA node
    return runtime.NumGoroutine() % n.numNodes
}

func (n *NUMACounter) Add(delta int64) {
    nodeIndex := n.getNodeIndex()
    atomic.AddInt64(&n.counters[nodeIndex].value, delta)
}

func (n *NUMACounter) Load() int64 {
    var total int64
    for i := range n.counters {
        total += atomic.LoadInt64(&n.counters[i].value)
    }
    return total
}

func (n *NUMACounter) LoadLocal() int64 {
    nodeIndex := n.getNodeIndex()
    return atomic.LoadInt64(&n.counters[nodeIndex].value)
}

// PerCPUCounter implements per-CPU atomic counters
type PerCPUCounter struct {
    counters []CacheAlignedCounter
}

func NewPerCPUCounter() *PerCPUCounter {
    numCPU := runtime.NumCPU()
    return &PerCPUCounter{
        counters: make([]CacheAlignedCounter, numCPU),
    }
}

func (p *PerCPUCounter) Add(delta int64) {
    // Pin to current CPU (simplified)
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    
    cpuID := runtime.NumGoroutine() % len(p.counters)
    atomic.AddInt64(&p.counters[cpuID].value, delta)
}

func (p *PerCPUCounter) Load() int64 {
    var total int64
    for i := range p.counters {
        total += atomic.LoadInt64(&p.counters[i].value)
    }
    return total
}
```

## Performance Targets

| Operation | Target Throughput | Target Latency | Notes |
|-----------|------------------|----------------|-------|
| Atomic Load | >100M ops/sec | <5ns P99 | Single thread |
| Atomic Store | >100M ops/sec | <5ns P99 | Single thread |
| Atomic Add | >50M ops/sec | <10ns P99 | Single thread |
| CAS Success | >30M ops/sec | <15ns P99 | Low contention |
| CAS Failure | >100M ops/sec | <5ns P99 | High contention |

## Key Metrics to Monitor

1. **Operation Latency**: Time per atomic operation
2. **Throughput**: Operations per second
3. **Contention Rate**: CAS failure percentage
4. **Cache Misses**: L1/L2/L3 cache miss rates
5. **False Sharing**: Cache line bouncing frequency
6. **Memory Bandwidth**: Bytes transferred per second
7. **CPU Utilization**: Core usage efficiency

## Optimization Techniques

### 1. Batch Operations
```go
func BatchedIncrement(counter *int64, count int) {
    // Batch multiple increments into single atomic operation
    atomic.AddInt64(counter, int64(count))
}

func BatchedCAS(addr *int64, updates []int64) bool {
    for _, update := range updates {
        old := atomic.LoadInt64(addr)
        new := old + update
        if !atomic.CompareAndSwapInt64(addr, old, new) {
            return false
        }
    }
    return true
}
```

### 2. Lock-Free Accumulator
```go
type LockFreeAccumulator struct {
    buckets []CacheAlignedCounter
    mask    int
}

func NewLockFreeAccumulator(size int) *LockFreeAccumulator {
    // Size must be power of 2
    if size&(size-1) != 0 {
        panic("size must be power of 2")
    }
    
    return &LockFreeAccumulator{
        buckets: make([]CacheAlignedCounter, size),
        mask:    size - 1,
    }
}

func (a *LockFreeAccumulator) Add(value int64) {
    // Hash to bucket to reduce contention
    bucket := runtime.NumGoroutine() & a.mask
    atomic.AddInt64(&a.buckets[bucket].value, value)
}

func (a *LockFreeAccumulator) Sum() int64 {
    var total int64
    for i := range a.buckets {
        total += atomic.LoadInt64(&a.buckets[i].value)
    }
    return total
}
```

### 3. Memory Ordering Optimization
```go
// Relaxed ordering for performance-critical counters
func RelaxedIncrement(counter *int64) {
    // Use relaxed ordering when synchronization not needed
    for {
        old := atomic.LoadInt64(counter)
        if atomic.CompareAndSwapInt64(counter, old, old+1) {
            break
        }
        runtime.Gosched() // Yield to reduce contention
    }
}

// Acquire-release for synchronization
func SynchronizedUpdate(flag *int32, data *int64, value int64) {
    atomic.StoreInt64(data, value)
    atomic.StoreInt32(flag, 1) // Release semantics
}

func SynchronizedRead(flag *int32, data *int64) (int64, bool) {
    if atomic.LoadInt32(flag) == 1 { // Acquire semantics
        return atomic.LoadInt64(data), true
    }
    return 0, false
}
```

## Success Criteria

- [ ] Implement comprehensive atomic operation benchmarks
- [ ] Achieve target throughput and latency metrics
- [ ] Demonstrate memory ordering understanding
- [ ] Show false sharing mitigation techniques
- [ ] Create NUMA-aware optimizations
- [ ] Implement effective backoff strategies

## Advanced Challenges

1. **Hardware Intrinsics**: Use assembly or cgo for architecture-specific optimizations
2. **Custom Memory Ordering**: Implement custom memory barriers
3. **Lock-Free Algorithms**: Build complex data structures using only atomics
4. **Performance Modeling**: Create mathematical models for atomic operation performance
5. **Cross-Platform Optimization**: Optimize for different CPU architectures

## Real-World Applications

- **High-Frequency Trading**: Ultra-low latency counters and flags
- **Database Systems**: Lock-free transaction counters
- **Web Servers**: Request counting and rate limiting
- **Game Engines**: Real-time statistics collection
- **Operating Systems**: Kernel synchronization primitives

## Next Steps

After completing this exercise:
1. Apply atomic operations to build lock-free data structures
2. Study advanced memory models and ordering semantics
3. Explore hardware-specific optimization techniques
4. Investigate formal verification of atomic algorithms
5. Contribute to high-performance open source projects

---

**Estimated Completion Time:** 45-60 minutes  
**Prerequisites:** Understanding of concurrent programming  
**Next Exercise:** [Exercise 04-02: Lock-Free Queue Implementation](../exercise-04-02/README.md)