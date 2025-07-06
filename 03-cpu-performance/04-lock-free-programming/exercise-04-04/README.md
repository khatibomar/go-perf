# Exercise 04-04: High-Performance Counter Design

**Objective:** Design and implement high-performance, scalable counter systems using advanced lock-free techniques.

**Duration:** 75-90 minutes  
**Difficulty:** Expert

## Learning Goals

By completing this exercise, you will:
- ✅ Design scalable counter architectures
- ✅ Implement striped and sharded counters
- ✅ Master NUMA-aware counter design
- ✅ Develop approximate counting algorithms
- ✅ Build production-ready counter systems

## Background

High-performance counters are critical components in many systems, from web analytics to database systems. This exercise explores advanced counter designs that scale across many cores while maintaining accuracy and performance.

### Counter Design Challenges

1. **Cache Line Contention**
   - Multiple cores updating same cache line
   - False sharing between adjacent counters
   - Cache coherency overhead

2. **Memory Bandwidth**
   - Limited bandwidth to shared memory
   - NUMA topology considerations
   - Memory access patterns

3. **Accuracy vs Performance**
   - Exact counting vs approximate counting
   - Consistency requirements
   - Read vs write performance trade-offs

### Counter Types

1. **Simple Atomic Counter**
   - Single atomic variable
   - High contention, poor scalability
   - Exact counting

2. **Striped Counter**
   - Multiple counter stripes
   - Reduces contention
   - Thread-local stripe selection

3. **Sharded Counter**
   - Per-core or per-thread shards
   - Minimal contention
   - Aggregation for reads

4. **Approximate Counter**
   - Relaxed consistency
   - High performance
   - Bounded error

## Tasks

### Task 1: Basic Counter Implementations
Implement fundamental counter types:
- Simple atomic counter
- Striped counter with configurable stripes
- Basic sharded counter
- Performance comparison

### Task 2: Advanced Sharding Strategies
Implement sophisticated sharding:
- NUMA-aware sharding
- Dynamic shard allocation
- Load-balanced sharding
- Cache-aligned data structures

### Task 3: Approximate Counting
Implement approximate counter algorithms:
- Probabilistic counters
- Bounded-error counters
- Hierarchical counters
- Consistency models

### Task 4: Production Optimization
Optimize for production use:
- Memory-efficient implementations
- Adaptive algorithms
- Monitoring and observability
- Fault tolerance

## Files Overview

- `counters/` - Counter implementations
  - `atomic_counter.go` - Simple atomic counter
  - `striped_counter.go` - Striped counter implementation
  - `sharded_counter.go` - Sharded counter with various strategies
  - `numa_counter.go` - NUMA-aware counter design

- `approximate/` - Approximate counting algorithms
  - `probabilistic_counter.go` - Probabilistic counting
  - `bounded_counter.go` - Bounded-error counter
  - `hierarchical_counter.go` - Hierarchical counter design
  - `sketch_counter.go` - Count-Min Sketch based counter

- `sharding/` - Sharding strategies
  - `hash_sharding.go` - Hash-based sharding
  - `numa_sharding.go` - NUMA-topology aware sharding
  - `dynamic_sharding.go` - Dynamic shard management
  - `affinity_sharding.go` - CPU affinity based sharding

- `optimization/` - Performance optimizations
  - `cache_aligned.go` - Cache-aligned data structures
  - `memory_pool.go` - Memory pool for counters
  - `batch_operations.go` - Batched counter operations
  - `adaptive_counter.go` - Adaptive counter selection

- `monitoring/` - Observability and monitoring
  - `metrics.go` - Counter metrics collection
  - `profiling.go` - Performance profiling tools
  - `health_check.go` - Counter health monitoring
  - `diagnostics.go` - Diagnostic tools

- `benchmarks/` - Performance benchmarks
  - `counter_bench_test.go` - Counter performance tests
  - `scalability_bench_test.go` - Scalability analysis
  - `contention_bench_test.go` - Contention measurement
  - `numa_bench_test.go` - NUMA performance tests

- `correctness/` - Correctness verification
  - `accuracy_test.go` - Counter accuracy tests
  - `consistency_test.go` - Consistency verification
  - `stress_test.go` - High-load stress tests
  - `race_test.go` - Race condition detection

- `main.go` - Demonstration and analysis
- `counter_test.go` - Comprehensive tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run scalability benchmarks
go test -bench=BenchmarkScalability -benchmem

# Profile memory usage
go test -bench=BenchmarkCounter -memprofile=mem.prof
go tool pprof mem.prof

# Test with race detector
go test -race

# NUMA topology testing
go test -bench=BenchmarkNUMA -benchmem

# Stress testing
go test -run=TestStress -timeout=60s

# Run demonstration
go run .

# Accuracy testing
go test -run=TestAccuracy -v
```

## Implementation Guidelines

### 1. Basic Counter Implementations
```go
package counters

import (
    "sync/atomic"
    "runtime"
    "unsafe"
)

// AtomicCounter implements a simple atomic counter
type AtomicCounter struct {
    value int64
    _     [56]byte // Padding to prevent false sharing
}

func NewAtomicCounter() *AtomicCounter {
    return &AtomicCounter{}
}

func (c *AtomicCounter) Add(delta int64) {
    atomic.AddInt64(&c.value, delta)
}

func (c *AtomicCounter) Inc() {
    atomic.AddInt64(&c.value, 1)
}

func (c *AtomicCounter) Dec() {
    atomic.AddInt64(&c.value, -1)
}

func (c *AtomicCounter) Get() int64 {
    return atomic.LoadInt64(&c.value)
}

func (c *AtomicCounter) Reset() {
    atomic.StoreInt64(&c.value, 0)
}

// StripedCounter reduces contention using multiple stripes
type StripedCounter struct {
    stripes []paddedCounter
    mask    uint64
}

type paddedCounter struct {
    value int64
    _     [56]byte // Cache line padding
}

func NewStripedCounter(numStripes int) *StripedCounter {
    // Ensure power of 2 for efficient masking
    stripes := nextPowerOf2(numStripes)
    
    return &StripedCounter{
        stripes: make([]paddedCounter, stripes),
        mask:    uint64(stripes - 1),
    }
}

func (c *StripedCounter) Add(delta int64) {
    // Use goroutine ID for stripe selection
    stripe := c.getStripeIndex()
    atomic.AddInt64(&c.stripes[stripe].value, delta)
}

func (c *StripedCounter) Inc() {
    c.Add(1)
}

func (c *StripedCounter) Dec() {
    c.Add(-1)
}

func (c *StripedCounter) Get() int64 {
    var total int64
    for i := range c.stripes {
        total += atomic.LoadInt64(&c.stripes[i].value)
    }
    return total
}

func (c *StripedCounter) Reset() {
    for i := range c.stripes {
        atomic.StoreInt64(&c.stripes[i].value, 0)
    }
}

func (c *StripedCounter) getStripeIndex() uint64 {
    // Simple hash based on goroutine ID
    // In practice, you might use thread-local storage
    return uint64(runtime.NumGoroutine()) & c.mask
}

func nextPowerOf2(n int) int {
    if n <= 1 {
        return 1
    }
    n--
    n |= n >> 1
    n |= n >> 2
    n |= n >> 4
    n |= n >> 8
    n |= n >> 16
    n |= n >> 32
    return n + 1
}

// ShardedCounter uses per-core shards for maximum performance
type ShardedCounter struct {
    shards    []paddedCounter
    numShards int
}

func NewShardedCounter() *ShardedCounter {
    numCPU := runtime.NumCPU()
    // Use 2x CPU count for better distribution
    numShards := numCPU * 2
    
    return &ShardedCounter{
        shards:    make([]paddedCounter, numShards),
        numShards: numShards,
    }
}

func (c *ShardedCounter) Add(delta int64) {
    shard := c.getShardIndex()
    atomic.AddInt64(&c.shards[shard].value, delta)
}

func (c *ShardedCounter) Inc() {
    c.Add(1)
}

func (c *ShardedCounter) Dec() {
    c.Add(-1)
}

func (c *ShardedCounter) Get() int64 {
    var total int64
    for i := range c.shards {
        total += atomic.LoadInt64(&c.shards[i].value)
    }
    return total
}

func (c *ShardedCounter) GetShardValues() []int64 {
    values := make([]int64, len(c.shards))
    for i := range c.shards {
        values[i] = atomic.LoadInt64(&c.shards[i].value)
    }
    return values
}

func (c *ShardedCounter) Reset() {
    for i := range c.shards {
        atomic.StoreInt64(&c.shards[i].value, 0)
    }
}

func (c *ShardedCounter) getShardIndex() int {
    // Use a combination of goroutine ID and CPU to distribute load
    gid := runtime.NumGoroutine()
    return gid % c.numShards
}
```

### 2. NUMA-Aware Counter Design
```go
package counters

import (
    "runtime"
    "sync/atomic"
    "syscall"
    "unsafe"
)

// NUMACounter implements NUMA-topology aware counting
type NUMACounter struct {
    nodes     []numaNode
    numNodes  int
    topology  *NUMATopology
}

type numaNode struct {
    counters []paddedCounter
    nodeID   int
    _        [48]byte // Padding
}

type NUMATopology struct {
    nodes        int
    coresPerNode int
    cpuToNode    map[int]int
}

func NewNUMACounter() *NUMACounter {
    topology := detectNUMATopology()
    
    nc := &NUMACounter{
        nodes:    make([]numaNode, topology.nodes),
        numNodes: topology.nodes,
        topology: topology,
    }
    
    // Initialize each NUMA node with multiple counters
    for i := range nc.nodes {
        nc.nodes[i] = numaNode{
            counters: make([]paddedCounter, topology.coresPerNode),
            nodeID:   i,
        }
    }
    
    return nc
}

func (nc *NUMACounter) Add(delta int64) {
    nodeID, counterID := nc.getOptimalCounter()
    atomic.AddInt64(&nc.nodes[nodeID].counters[counterID].value, delta)
}

func (nc *NUMACounter) Inc() {
    nc.Add(1)
}

func (nc *NUMACounter) Dec() {
    nc.Add(-1)
}

func (nc *NUMACounter) Get() int64 {
    var total int64
    for nodeIdx := range nc.nodes {
        for counterIdx := range nc.nodes[nodeIdx].counters {
            total += atomic.LoadInt64(&nc.nodes[nodeIdx].counters[counterIdx].value)
        }
    }
    return total
}

func (nc *NUMACounter) GetByNode() []int64 {
    nodeTotals := make([]int64, nc.numNodes)
    for nodeIdx := range nc.nodes {
        var nodeTotal int64
        for counterIdx := range nc.nodes[nodeIdx].counters {
            nodeTotal += atomic.LoadInt64(&nc.nodes[nodeIdx].counters[counterIdx].value)
        }
        nodeTotals[nodeIdx] = nodeTotal
    }
    return nodeTotals
}

func (nc *NUMACounter) getOptimalCounter() (int, int) {
    // Get current CPU (simplified)
    currentCPU := getCurrentCPU()
    nodeID := nc.topology.cpuToNode[currentCPU]
    
    // Use goroutine ID to select counter within node
    gid := runtime.NumGoroutine()
    counterID := gid % len(nc.nodes[nodeID].counters)
    
    return nodeID, counterID
}

func detectNUMATopology() *NUMATopology {
    // Simplified NUMA detection
    // In practice, you'd use system calls or /proc/cpuinfo
    numCPU := runtime.NumCPU()
    
    // Assume 2 NUMA nodes for demonstration
    nodes := 2
    if numCPU <= 4 {
        nodes = 1
    }
    
    coresPerNode := numCPU / nodes
    cpuToNode := make(map[int]int)
    
    for cpu := 0; cpu < numCPU; cpu++ {
        cpuToNode[cpu] = cpu / coresPerNode
    }
    
    return &NUMATopology{
        nodes:        nodes,
        coresPerNode: coresPerNode,
        cpuToNode:    cpuToNode,
    }
}

func getCurrentCPU() int {
    // Simplified CPU detection
    // In practice, you'd use sched_getcpu() or similar
    return runtime.NumGoroutine() % runtime.NumCPU()
}
```

### 3. Approximate Counting Algorithms
```go
package approximate

import (
    "math"
    "math/rand"
    "sync/atomic"
)

// ProbabilisticCounter implements probabilistic counting
type ProbabilisticCounter struct {
    value       int64
    probability float64
    scale       float64
    _           [48]byte // Padding
}

func NewProbabilisticCounter(probability float64) *ProbabilisticCounter {
    return &ProbabilisticCounter{
        probability: probability,
        scale:       1.0 / probability,
    }
}

func (pc *ProbabilisticCounter) Add(delta int64) {
    for i := int64(0); i < delta; i++ {
        if rand.Float64() < pc.probability {
            atomic.AddInt64(&pc.value, 1)
        }
    }
}

func (pc *ProbabilisticCounter) Inc() {
    if rand.Float64() < pc.probability {
        atomic.AddInt64(&pc.value, 1)
    }
}

func (pc *ProbabilisticCounter) Get() int64 {
    observed := atomic.LoadInt64(&pc.value)
    return int64(float64(observed) * pc.scale)
}

func (pc *ProbabilisticCounter) GetExact() int64 {
    return atomic.LoadInt64(&pc.value)
}

func (pc *ProbabilisticCounter) Reset() {
    atomic.StoreInt64(&pc.value, 0)
}

// BoundedCounter maintains bounded error guarantees
type BoundedCounter struct {
    localCounters []localCounter
    threshold     int64
    globalCounter int64
    numCounters   int
    _             [40]byte // Padding
}

type localCounter struct {
    value int64
    _     [56]byte // Padding
}

func NewBoundedCounter(threshold int64, numCounters int) *BoundedCounter {
    return &BoundedCounter{
        localCounters: make([]localCounter, numCounters),
        threshold:     threshold,
        numCounters:   numCounters,
    }
}

func (bc *BoundedCounter) Add(delta int64) {
    counterIdx := bc.getCounterIndex()
    newValue := atomic.AddInt64(&bc.localCounters[counterIdx].value, delta)
    
    // Flush to global counter if threshold exceeded
    if newValue >= bc.threshold {
        if atomic.CompareAndSwapInt64(&bc.localCounters[counterIdx].value, newValue, 0) {
            atomic.AddInt64(&bc.globalCounter, newValue)
        }
    }
}

func (bc *BoundedCounter) Inc() {
    bc.Add(1)
}

func (bc *BoundedCounter) Get() int64 {
    global := atomic.LoadInt64(&bc.globalCounter)
    
    // Add local counter values
    var local int64
    for i := range bc.localCounters {
        local += atomic.LoadInt64(&bc.localCounters[i].value)
    }
    
    return global + local
}

func (bc *BoundedCounter) GetBounds() (int64, int64) {
    value := bc.Get()
    maxError := int64(bc.numCounters) * (bc.threshold - 1)
    return value, value + maxError
}

func (bc *BoundedCounter) Flush() {
    for i := range bc.localCounters {
        value := atomic.SwapInt64(&bc.localCounters[i].value, 0)
        if value > 0 {
            atomic.AddInt64(&bc.globalCounter, value)
        }
    }
}

func (bc *BoundedCounter) getCounterIndex() int {
    // Simple distribution based on goroutine ID
    return runtime.NumGoroutine() % bc.numCounters
}

// HierarchicalCounter implements hierarchical counting
type HierarchicalCounter struct {
    levels    []counterLevel
    numLevels int
    fanout    int
}

type counterLevel struct {
    counters  []paddedCounter
    threshold int64
}

func NewHierarchicalCounter(numLevels, fanout int, baseThreshold int64) *HierarchicalCounter {
    hc := &HierarchicalCounter{
        levels:    make([]counterLevel, numLevels),
        numLevels: numLevels,
        fanout:    fanout,
    }
    
    // Initialize levels with increasing thresholds
    for level := 0; level < numLevels; level++ {
        numCounters := int(math.Pow(float64(fanout), float64(level)))
        threshold := baseThreshold * int64(math.Pow(2, float64(level)))
        
        hc.levels[level] = counterLevel{
            counters:  make([]paddedCounter, numCounters),
            threshold: threshold,
        }
    }
    
    return hc
}

func (hc *HierarchicalCounter) Add(delta int64) {
    hc.addToLevel(0, delta)
}

func (hc *HierarchicalCounter) addToLevel(level int, delta int64) {
    if level >= hc.numLevels {
        return
    }
    
    counterIdx := hc.getCounterIndex(level)
    newValue := atomic.AddInt64(&hc.levels[level].counters[counterIdx].value, delta)
    
    // Propagate to next level if threshold exceeded
    if newValue >= hc.levels[level].threshold {
        if atomic.CompareAndSwapInt64(&hc.levels[level].counters[counterIdx].value, newValue, 0) {
            hc.addToLevel(level+1, newValue)
        }
    }
}

func (hc *HierarchicalCounter) Get() int64 {
    var total int64
    
    for level := range hc.levels {
        for i := range hc.levels[level].counters {
            value := atomic.LoadInt64(&hc.levels[level].counters[i].value)
            total += value
        }
    }
    
    return total
}

func (hc *HierarchicalCounter) GetByLevel() []int64 {
    levelTotals := make([]int64, hc.numLevels)
    
    for level := range hc.levels {
        var levelTotal int64
        for i := range hc.levels[level].counters {
            levelTotal += atomic.LoadInt64(&hc.levels[level].counters[i].value)
        }
        levelTotals[level] = levelTotal
    }
    
    return levelTotals
}

func (hc *HierarchicalCounter) getCounterIndex(level int) int {
    numCounters := len(hc.levels[level].counters)
    return runtime.NumGoroutine() % numCounters
}
```

### 4. Adaptive Counter System
```go
package optimization

import (
    "sync/atomic"
    "time"
)

// AdaptiveCounter automatically selects optimal counter type
type AdaptiveCounter struct {
    current     Counter
    metrics     *CounterMetrics
    config      *AdaptiveConfig
    lastSwitch  time.Time
    switchCount int64
}

type Counter interface {
    Add(delta int64)
    Inc()
    Dec()
    Get() int64
    Reset()
}

type CounterMetrics struct {
    operations    int64
    contentions   int64
    avgLatency    int64
    throughput    int64
    lastMeasured  time.Time
}

type AdaptiveConfig struct {
    measureInterval   time.Duration
    switchThreshold   float64
    contentionLimit   float64
    latencyLimit      time.Duration
    minSwitchInterval time.Duration
}

func NewAdaptiveCounter(config *AdaptiveConfig) *AdaptiveCounter {
    if config == nil {
        config = &AdaptiveConfig{
            measureInterval:   time.Second,
            switchThreshold:   0.2,
            contentionLimit:   0.8,
            latencyLimit:      time.Microsecond * 100,
            minSwitchInterval: time.Second * 5,
        }
    }
    
    ac := &AdaptiveCounter{
        current: NewAtomicCounter(),
        metrics: &CounterMetrics{lastMeasured: time.Now()},
        config:  config,
    }
    
    go ac.monitor()
    return ac
}

func (ac *AdaptiveCounter) Add(delta int64) {
    start := time.Now()
    ac.current.Add(delta)
    
    // Update metrics
    atomic.AddInt64(&ac.metrics.operations, 1)
    latency := time.Since(start).Nanoseconds()
    atomic.StoreInt64(&ac.metrics.avgLatency, latency)
}

func (ac *AdaptiveCounter) Inc() {
    ac.Add(1)
}

func (ac *AdaptiveCounter) Dec() {
    ac.Add(-1)
}

func (ac *AdaptiveCounter) Get() int64 {
    return ac.current.Get()
}

func (ac *AdaptiveCounter) Reset() {
    ac.current.Reset()
    ac.resetMetrics()
}

func (ac *AdaptiveCounter) monitor() {
    ticker := time.NewTicker(ac.config.measureInterval)
    defer ticker.Stop()
    
    for range ticker.C {
        ac.evaluatePerformance()
    }
}

func (ac *AdaptiveCounter) evaluatePerformance() {
    now := time.Now()
    elapsed := now.Sub(ac.metrics.lastMeasured)
    
    if elapsed < ac.config.measureInterval {
        return
    }
    
    operations := atomic.LoadInt64(&ac.metrics.operations)
    avgLatency := time.Duration(atomic.LoadInt64(&ac.metrics.avgLatency))
    
    // Calculate throughput
    throughput := float64(operations) / elapsed.Seconds()
    atomic.StoreInt64(&ac.metrics.throughput, int64(throughput))
    
    // Decide if we should switch counter type
    if ac.shouldSwitch(avgLatency, throughput) {
        ac.switchCounterType(avgLatency, throughput)
    }
    
    ac.resetMetrics()
    ac.metrics.lastMeasured = now
}

func (ac *AdaptiveCounter) shouldSwitch(latency time.Duration, throughput float64) bool {
    // Don't switch too frequently
    if time.Since(ac.lastSwitch) < ac.config.minSwitchInterval {
        return false
    }
    
    // Switch if latency is too high
    if latency > ac.config.latencyLimit {
        return true
    }
    
    // Switch if throughput is declining
    prevThroughput := atomic.LoadInt64(&ac.metrics.throughput)
    if prevThroughput > 0 {
        decline := (float64(prevThroughput) - throughput) / float64(prevThroughput)
        if decline > ac.config.switchThreshold {
            return true
        }
    }
    
    return false
}

func (ac *AdaptiveCounter) switchCounterType(latency time.Duration, throughput float64) {
    currentValue := ac.current.Get()
    
    var newCounter Counter
    
    // Select new counter type based on performance characteristics
    if latency > time.Microsecond*50 {
        // High latency - use sharded counter
        newCounter = NewShardedCounter()
    } else if throughput < 1000000 {
        // Low throughput - use striped counter
        newCounter = NewStripedCounter(runtime.NumCPU() * 2)
    } else {
        // High throughput - use NUMA-aware counter
        newCounter = NewNUMACounter()
    }
    
    // Transfer current value to new counter
    newCounter.Add(currentValue)
    
    // Atomic switch
    ac.current = newCounter
    ac.lastSwitch = time.Now()
    atomic.AddInt64(&ac.switchCount, 1)
}

func (ac *AdaptiveCounter) resetMetrics() {
    atomic.StoreInt64(&ac.metrics.operations, 0)
    atomic.StoreInt64(&ac.metrics.contentions, 0)
    atomic.StoreInt64(&ac.metrics.avgLatency, 0)
}

func (ac *AdaptiveCounter) GetMetrics() CounterMetrics {
    return CounterMetrics{
        operations:   atomic.LoadInt64(&ac.metrics.operations),
        contentions:  atomic.LoadInt64(&ac.metrics.contentions),
        avgLatency:   atomic.LoadInt64(&ac.metrics.avgLatency),
        throughput:   atomic.LoadInt64(&ac.metrics.throughput),
        lastMeasured: ac.metrics.lastMeasured,
    }
}

func (ac *AdaptiveCounter) GetSwitchCount() int64 {
    return atomic.LoadInt64(&ac.switchCount)
}
```

## Performance Targets

| Counter Type | Target Throughput | Target Latency | Scalability | Memory Usage |
|--------------|------------------|----------------|-------------|-------------|
| Atomic | >10M ops/sec | <50ns P99 | Poor | Minimal |
| Striped | >50M ops/sec | <30ns P99 | Good | Low |
| Sharded | >100M ops/sec | <20ns P99 | Excellent | Medium |
| NUMA-Aware | >150M ops/sec | <15ns P99 | Excellent | Medium |
| Approximate | >200M ops/sec | <10ns P99 | Excellent | Low |

## Key Metrics to Monitor

1. **Throughput**: Operations per second across all cores
2. **Latency Distribution**: P50, P95, P99 latencies
3. **Scalability**: Performance vs number of cores
4. **Cache Performance**: Cache miss rates and coherency traffic
5. **Memory Usage**: Memory footprint and allocation rate
6. **Accuracy**: Error bounds for approximate counters

## Success Criteria

- [ ] Implement all counter types correctly
- [ ] Achieve target performance metrics
- [ ] Demonstrate linear scalability for sharded counters
- [ ] Show NUMA-awareness benefits
- [ ] Implement bounded-error approximate counting
- [ ] Build adaptive counter selection system

## Advanced Challenges

1. **Persistent Counters**: Add durability and crash recovery
2. **Distributed Counters**: Implement distributed counting across nodes
3. **Real-time Analytics**: Build streaming analytics on counter data
4. **Hardware Acceleration**: Use specialized hardware for counting
5. **Formal Verification**: Prove correctness and bounds

## Real-World Applications

- **Web Analytics**: Page view and user interaction counting
- **Database Systems**: Transaction and query counters
- **Monitoring Systems**: Metrics collection and aggregation
- **Game Engines**: Score and statistics tracking
- **Financial Systems**: Transaction volume monitoring
- **Network Processing**: Packet and bandwidth counters

---

**Estimated Completion Time:** 75-90 minutes  
**Prerequisites:** Exercises 04-01, 04-02, and 04-03  
**Next Chapter:** [Chapter 05: I/O Performance Optimization](../../05-io-performance/README.md)