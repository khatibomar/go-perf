# Exercise 03-03: Work Distribution Patterns

**Objective:** Implement and optimize various work distribution patterns for concurrent processing.

**Duration:** 45-60 minutes  
**Difficulty:** Advanced

## Learning Goals

By completing this exercise, you will:
- ✅ Understand different work distribution strategies and their trade-offs
- ✅ Implement fan-out/fan-in patterns efficiently
- ✅ Design load balancing mechanisms for concurrent workers
- ✅ Optimize work partitioning and scheduling algorithms
- ✅ Handle dynamic workload balancing and redistribution

## Background

Efficient work distribution is crucial for maximizing CPU utilization and minimizing latency in concurrent systems. This exercise explores various patterns for distributing work across multiple goroutines.

### Distribution Patterns

1. **Round-Robin Distribution**
   - Simple cyclic assignment
   - Predictable load distribution
   - No consideration of worker capacity

2. **Load-Based Distribution**
   - Dynamic assignment based on current load
   - Adaptive to worker performance
   - Complex monitoring requirements

3. **Work-Stealing Distribution**
   - Workers steal from each other
   - Self-balancing system
   - Reduced central coordination

4. **Hierarchical Distribution**
   - Multi-level work distribution
   - Scalable to large worker counts
   - Complex coordination logic

5. **Affinity-Based Distribution**
   - Work assigned based on data locality
   - Cache-friendly processing
   - Reduced memory bandwidth

## Tasks

### Task 1: Basic Distribution Patterns
Implement fundamental distribution strategies:
- Round-robin distributor
- Random distribution
- Hash-based distribution
- Weighted distribution

### Task 2: Advanced Distribution Algorithms
Implement sophisticated distribution mechanisms:
- Load-aware distribution
- Work-stealing coordination
- Hierarchical distribution trees
- Affinity-based assignment

### Task 3: Dynamic Load Balancing
Implement adaptive load balancing:
- Real-time load monitoring
- Dynamic work redistribution
- Worker capacity estimation
- Bottleneck detection and mitigation

### Task 4: Performance Optimization
Optimize distribution performance:
- Minimize coordination overhead
- Reduce lock contention
- Optimize work granularity
- Implement efficient signaling

## Files Overview

- `distributors/` - Work distribution implementations
  - `round_robin.go` - Round-robin distributor
  - `load_based.go` - Load-aware distributor
  - `work_stealing.go` - Work-stealing coordinator
  - `hierarchical.go` - Hierarchical distributor
  - `affinity.go` - Affinity-based distributor

- `balancers/` - Load balancing algorithms
  - `dynamic_balancer.go` - Dynamic load balancer
  - `capacity_estimator.go` - Worker capacity estimation
  - `bottleneck_detector.go` - Bottleneck detection
  - `redistribution.go` - Work redistribution logic

- `workers/` - Worker implementations
  - `worker_pool.go` - Managed worker pool
  - `adaptive_worker.go` - Self-adapting worker
  - `monitored_worker.go` - Performance-monitored worker

- `coordination/` - Coordination mechanisms
  - `coordinator.go` - Central coordinator
  - `gossip.go` - Gossip-based coordination
  - `consensus.go` - Consensus algorithms

- `monitoring/` - Performance monitoring
  - `load_monitor.go` - Load monitoring
  - `performance_tracker.go` - Performance tracking
  - `metrics_collector.go` - Metrics collection

- `benchmarks/` - Performance benchmarks
  - `distribution_bench_test.go` - Distribution performance
  - `balancing_bench_test.go` - Load balancing tests
  - `scalability_bench_test.go` - Scalability tests

- `main.go` - Demonstration and testing
- `distribution_test.go` - Correctness tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific distribution benchmarks
go test -bench=BenchmarkRoundRobin -benchmem

# Profile CPU usage
go test -bench=BenchmarkDistribution -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Profile goroutine usage
go test -bench=BenchmarkDistribution -trace=trace.out
go tool trace trace.out

# Run demonstration
go run .

# Test correctness
go test -v
```

## Implementation Guidelines

### 1. Round-Robin Distributor
```go
type RoundRobinDistributor struct {
    workers []Worker
    current int64
    mu      sync.RWMutex
}

func (d *RoundRobinDistributor) Distribute(work Work) error {
    // Atomic increment for thread safety
    index := atomic.AddInt64(&d.current, 1) - 1
    workerIndex := index % int64(len(d.workers))
    
    return d.workers[workerIndex].Submit(work)
}

func (d *RoundRobinDistributor) AddWorker(worker Worker) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.workers = append(d.workers, worker)
}
```

### 2. Load-Based Distributor
```go
type LoadBasedDistributor struct {
    workers []LoadAwareWorker
    mu      sync.RWMutex
}

func (d *LoadBasedDistributor) Distribute(work Work) error {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    // Find worker with lowest load
    var bestWorker LoadAwareWorker
    minLoad := math.MaxFloat64
    
    for _, worker := range d.workers {
        if load := worker.GetLoad(); load < minLoad {
            minLoad = load
            bestWorker = worker
        }
    }
    
    if bestWorker == nil {
        return errors.New("no available workers")
    }
    
    return bestWorker.Submit(work)
}
```

### 3. Work-Stealing Coordinator
```go
type WorkStealingCoordinator struct {
    workers []*StealingWorker
    metrics *CoordinatorMetrics
}

type StealingWorker struct {
    id         int
    localQueue chan Work
    coordinator *WorkStealingCoordinator
    load       int64
}

func (w *StealingWorker) trySteal() Work {
    // Try to steal from a random worker
    for attempts := 0; attempts < len(w.coordinator.workers); attempts++ {
        victim := w.coordinator.workers[rand.Intn(len(w.coordinator.workers))]
        if victim.id != w.id {
            select {
            case work := <-victim.localQueue:
                atomic.AddInt64(&victim.load, -1)
                return work
            default:
                // Queue empty, try next worker
            }
        }
    }
    return nil
}

func (w *StealingWorker) run() {
    for {
        select {
        case work := <-w.localQueue:
            w.processWork(work)
            atomic.AddInt64(&w.load, -1)
            
        default:
            // Try to steal work
            if work := w.trySteal(); work != nil {
                w.processWork(work)
            } else {
                // No work available, yield
                runtime.Gosched()
            }
        }
    }
}
```

### 4. Hierarchical Distributor
```go
type HierarchicalDistributor struct {
    levels [][]Distributor
    fanout int
}

func (d *HierarchicalDistributor) Distribute(work Work) error {
    // Start from top level
    currentLevel := 0
    currentIndex := 0
    
    for currentLevel < len(d.levels)-1 {
        // Distribute to next level
        nextLevelStart := currentIndex * d.fanout
        nextLevelEnd := min(nextLevelStart+d.fanout, len(d.levels[currentLevel+1]))
        
        // Choose distributor in next level
        targetIndex := nextLevelStart + (work.Hash() % (nextLevelEnd - nextLevelStart))
        
        currentLevel++
        currentIndex = targetIndex
    }
    
    // Submit to final level worker
    return d.levels[currentLevel][currentIndex].Distribute(work)
}
```

### 5. Affinity-Based Distributor
```go
type AffinityDistributor struct {
    workers    []Worker
    affinityMap map[string]int // Data key -> worker index
    mu         sync.RWMutex
}

func (d *AffinityDistributor) Distribute(work Work) error {
    key := work.GetAffinityKey()
    
    d.mu.RLock()
    workerIndex, exists := d.affinityMap[key]
    d.mu.RUnlock()
    
    if !exists {
        // Assign new affinity
        d.mu.Lock()
        if workerIndex, exists = d.affinityMap[key]; !exists {
            workerIndex = len(d.affinityMap) % len(d.workers)
            d.affinityMap[key] = workerIndex
        }
        d.mu.Unlock()
    }
    
    return d.workers[workerIndex].Submit(work)
}
```

## Performance Targets

| Pattern | Throughput Target | Latency Target | Scalability |
|---------|------------------|----------------|-------------|
| Round-Robin | >200K ops/sec | <0.5ms P99 | Linear |
| Load-Based | >150K ops/sec | <1ms P99 | Sub-linear |
| Work-Stealing | >250K ops/sec | <0.5ms P99 | Near-linear |
| Hierarchical | >300K ops/sec | <1ms P99 | Logarithmic |
| Affinity-Based | >180K ops/sec | <0.8ms P99 | Linear |

## Key Metrics to Monitor

1. **Distribution Latency**: Time to assign work to worker
2. **Load Balance**: Variance in worker utilization
3. **Coordination Overhead**: Time spent in coordination
4. **Stealing Efficiency**: Success rate of work stealing
5. **Cache Locality**: Cache hit rates with affinity
6. **Scalability**: Performance vs. number of workers

## Optimization Techniques

### 1. Lock-Free Distribution
```go
// Use atomic operations for lock-free round-robin
type LockFreeRoundRobin struct {
    workers []Worker
    counter uint64
}

func (d *LockFreeRoundRobin) Distribute(work Work) error {
    index := atomic.AddUint64(&d.counter, 1) - 1
    workerIndex := index % uint64(len(d.workers))
    return d.workers[workerIndex].Submit(work)
}
```

### 2. Batch Distribution
```go
func (d *BatchDistributor) DistributeBatch(works []Work) error {
    batchSize := len(works) / len(d.workers)
    remainder := len(works) % len(d.workers)
    
    var wg sync.WaitGroup
    start := 0
    
    for i, worker := range d.workers {
        end := start + batchSize
        if i < remainder {
            end++ // Distribute remainder
        }
        
        if start < end {
            wg.Add(1)
            go func(w Worker, batch []Work) {
                defer wg.Done()
                w.SubmitBatch(batch)
            }(worker, works[start:end])
        }
        
        start = end
    }
    
    wg.Wait()
    return nil
}
```

### 3. Adaptive Load Estimation
```go
type AdaptiveLoadEstimator struct {
    samples    []float64
    sampleSize int
    index      int
    mu         sync.Mutex
}

func (e *AdaptiveLoadEstimator) UpdateLoad(load float64) {
    e.mu.Lock()
    defer e.mu.Unlock()
    
    e.samples[e.index] = load
    e.index = (e.index + 1) % e.sampleSize
}

func (e *AdaptiveLoadEstimator) EstimateLoad() float64 {
    e.mu.Lock()
    defer e.mu.Unlock()
    
    var sum float64
    for _, sample := range e.samples {
        sum += sample
    }
    return sum / float64(len(e.samples))
}
```

## Success Criteria

- [ ] Implement all five distribution patterns correctly
- [ ] Achieve target throughput and latency metrics
- [ ] Demonstrate proper load balancing
- [ ] Show scalability improvements
- [ ] Implement efficient coordination mechanisms
- [ ] Create comprehensive performance analysis

## Advanced Challenges

1. **Machine Learning Distribution**: Use ML for optimal work assignment
2. **Quantum-Inspired Algorithms**: Implement quantum-inspired optimization
3. **Distributed Coordination**: Extend to multi-node distribution
4. **Real-Time Constraints**: Implement real-time scheduling
5. **Energy-Aware Distribution**: Optimize for energy efficiency

## Real-World Applications

- **MapReduce Systems**: Distributed data processing
- **Load Balancers**: HTTP request distribution
- **Database Systems**: Query execution distribution
- **Game Engines**: Task distribution across cores
- **Video Processing**: Frame processing distribution

## Next Steps

After completing this exercise:
1. Apply patterns to your specific use cases
2. Experiment with hybrid distribution strategies
3. Study advanced scheduling algorithms
4. Explore distributed systems patterns
5. Investigate hardware-aware optimizations

---

**Estimated Completion Time:** 45-60 minutes  
**Prerequisites:** Exercises 03-01 and 03-02  
**Next Exercise:** [Exercise 03-04: Concurrent Pipeline Optimization](../exercise-03-04/README.md)