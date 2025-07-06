# Exercise 03-02: Goroutine Pool Optimization

**Objective:** Design and implement high-performance goroutine pools for various workload patterns.

**Duration:** 45-60 minutes  
**Difficulty:** Advanced

## Learning Goals

By completing this exercise, you will:
- ✅ Understand goroutine pool design patterns and trade-offs
- ✅ Implement efficient work distribution mechanisms
- ✅ Optimize pool sizing and lifecycle management
- ✅ Handle backpressure and flow control effectively
- ✅ Measure and optimize pool performance characteristics

## Background

Goroutine pools are essential for controlling resource usage and optimizing performance in concurrent Go applications. This exercise explores various pool implementations and optimization techniques.

### Pool Design Patterns

1. **Fixed-Size Pool**
   - Pre-allocated worker goroutines
   - Simple work distribution
   - Predictable resource usage

2. **Dynamic Pool**
   - Adaptive worker scaling
   - Load-based expansion/contraction
   - Complex lifecycle management

3. **Work-Stealing Pool**
   - Per-worker queues
   - Load balancing through stealing
   - Reduced contention

4. **Priority Pool**
   - Multiple priority levels
   - Priority-based scheduling
   - Starvation prevention

## Tasks

### Task 1: Basic Pool Implementations
Implement fundamental pool patterns:
- Fixed-size worker pool
- Dynamic scaling pool
- Work-stealing pool
- Priority-based pool

### Task 2: Performance Optimization
Optimize pool performance:
- Minimize lock contention
- Optimize work distribution
- Reduce allocation overhead
- Implement efficient signaling

### Task 3: Advanced Features
Implement advanced pool features:
- Graceful shutdown
- Health monitoring
- Metrics collection
- Circuit breaker integration

### Task 4: Workload-Specific Optimization
Optimize for specific workload patterns:
- CPU-intensive tasks
- I/O-bound operations
- Mixed workloads
- Batch processing

## Files Overview

- `pools/` - Pool implementations
  - `fixed_pool.go` - Fixed-size worker pool
  - `dynamic_pool.go` - Dynamic scaling pool
  - `workstealing_pool.go` - Work-stealing pool
  - `priority_pool.go` - Priority-based pool

- `workers/` - Worker implementations
  - `worker.go` - Basic worker interface
  - `cpu_worker.go` - CPU-intensive worker
  - `io_worker.go` - I/O-bound worker
  - `batch_worker.go` - Batch processing worker

- `queue/` - Work queue implementations
  - `channel_queue.go` - Channel-based queue
  - `lockfree_queue.go` - Lock-free queue
  - `priority_queue.go` - Priority queue
  - `stealing_queue.go` - Work-stealing queue

- `monitoring/` - Pool monitoring
  - `metrics.go` - Performance metrics
  - `health.go` - Health monitoring
  - `profiler.go` - Pool profiler

- `benchmarks/` - Performance benchmarks
  - `pool_bench_test.go` - Pool performance tests
  - `workload_bench_test.go` - Workload-specific tests
  - `scaling_bench_test.go` - Scaling performance tests

- `main.go` - Demonstration and testing
- `pool_test.go` - Correctness tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific pool benchmarks
go test -bench=BenchmarkFixedPool -benchmem

# Profile CPU usage
go test -bench=BenchmarkPool -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Profile goroutine usage
go test -bench=BenchmarkPool -trace=trace.out
go tool trace trace.out

# Run demonstration
go run .

# Test correctness
go test -v
```

## Implementation Guidelines

### 1. Fixed-Size Pool
```go
type FixedPool struct {
    workers   int
    workChan  chan Work
    quitChan  chan struct{}
    wg        sync.WaitGroup
    metrics   *PoolMetrics
}

func NewFixedPool(workers int, queueSize int) *FixedPool {
    pool := &FixedPool{
        workers:  workers,
        workChan: make(chan Work, queueSize),
        quitChan: make(chan struct{}),
        metrics:  NewPoolMetrics(),
    }
    
    // Start workers
    for i := 0; i < workers; i++ {
        pool.wg.Add(1)
        go pool.worker(i)
    }
    
    return pool
}

func (p *FixedPool) worker(id int) {
    defer p.wg.Done()
    
    for {
        select {
        case work := <-p.workChan:
            start := time.Now()
            work.Execute()
            p.metrics.RecordExecution(time.Since(start))
            
        case <-p.quitChan:
            return
        }
    }
}
```

### 2. Dynamic Pool
```go
type DynamicPool struct {
    minWorkers    int
    maxWorkers    int
    currentWorkers int64
    workChan      chan Work
    workerChan    chan chan Work
    quitChan      chan struct{}
    mu            sync.RWMutex
    metrics       *PoolMetrics
}

func (p *DynamicPool) scaleUp() {
    current := atomic.LoadInt64(&p.currentWorkers)
    if current < int64(p.maxWorkers) {
        if atomic.CompareAndSwapInt64(&p.currentWorkers, current, current+1) {
            go p.worker()
        }
    }
}

func (p *DynamicPool) scaleDown() {
    current := atomic.LoadInt64(&p.currentWorkers)
    if current > int64(p.minWorkers) {
        // Signal worker to terminate
        select {
        case p.quitChan <- struct{}{}:
            atomic.AddInt64(&p.currentWorkers, -1)
        default:
        }
    }
}
```

### 3. Work-Stealing Pool
```go
type WorkStealingPool struct {
    workers     []*WorkStealingWorker
    globalQueue chan Work
    metrics     *PoolMetrics
}

type WorkStealingWorker struct {
    id         int
    localQueue chan Work
    pool       *WorkStealingPool
    quit       chan struct{}
}

func (w *WorkStealingWorker) run() {
    for {
        select {
        case work := <-w.localQueue:
            work.Execute()
            
        case work := <-w.pool.globalQueue:
            work.Execute()
            
        case <-w.quit:
            return
            
        default:
            // Try to steal work from other workers
            if work := w.stealWork(); work != nil {
                work.Execute()
            } else {
                // No work available, yield
                runtime.Gosched()
            }
        }
    }
}

func (w *WorkStealingWorker) stealWork() Work {
    // Try to steal from random worker
    for i := 0; i < len(w.pool.workers); i++ {
        victim := w.pool.workers[rand.Intn(len(w.pool.workers))]
        if victim.id != w.id {
            select {
            case work := <-victim.localQueue:
                return work
            default:
            }
        }
    }
    return nil
}
```

### 4. Priority Pool
```go
type PriorityPool struct {
    workers      int
    priorityQueues []*PriorityQueue
    quitChan     chan struct{}
    wg           sync.WaitGroup
    metrics      *PoolMetrics
}

func (p *PriorityPool) worker() {
    defer p.wg.Done()
    
    for {
        select {
        case <-p.quitChan:
            return
        default:
            // Check queues in priority order
            work := p.getNextWork()
            if work != nil {
                start := time.Now()
                work.Execute()
                p.metrics.RecordExecution(time.Since(start))
            } else {
                // No work available, yield
                runtime.Gosched()
            }
        }
    }
}

func (p *PriorityPool) getNextWork() Work {
    // Check high priority first, then medium, then low
    for _, queue := range p.priorityQueues {
        if work := queue.TryDequeue(); work != nil {
            return work
        }
    }
    return nil
}
```

## Performance Targets

| Pool Type | Throughput Target | Latency Target | Memory Overhead |
|-----------|------------------|----------------|------------------|
| Fixed Pool | >100K ops/sec | <1ms P99 | Low |
| Dynamic Pool | >80K ops/sec | <2ms P99 | Medium |
| Work-Stealing | >120K ops/sec | <1ms P99 | Medium |
| Priority Pool | >90K ops/sec | <1.5ms P99 | High |

## Key Metrics to Monitor

1. **Throughput**: Operations processed per second
2. **Latency**: Time from submission to completion
3. **Queue Depth**: Number of pending work items
4. **Worker Utilization**: Percentage of time workers are busy
5. **Scaling Events**: Frequency of pool size changes
6. **Contention**: Lock contention and blocking time

## Optimization Techniques

### 1. Lock-Free Queues
```go
// Use atomic operations for lock-free queue
type LockFreeQueue struct {
    head unsafe.Pointer
    tail unsafe.Pointer
}

func (q *LockFreeQueue) Enqueue(item interface{}) {
    node := &queueNode{data: item}
    
    for {
        tail := (*queueNode)(atomic.LoadPointer(&q.tail))
        next := (*queueNode)(atomic.LoadPointer(&tail.next))
        
        if tail == (*queueNode)(atomic.LoadPointer(&q.tail)) {
            if next == nil {
                if atomic.CompareAndSwapPointer(&tail.next, nil, unsafe.Pointer(node)) {
                    atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer(tail), unsafe.Pointer(node))
                    break
                }
            } else {
                atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer(tail), unsafe.Pointer(next))
            }
        }
    }
}
```

### 2. Batch Processing
```go
func (p *BatchPool) processBatch() {
    batch := make([]Work, 0, p.batchSize)
    
    for {
        // Collect batch
        for len(batch) < p.batchSize {
            select {
            case work := <-p.workChan:
                batch = append(batch, work)
            case <-time.After(p.batchTimeout):
                goto processBatch
            }
        }
        
    processBatch:
        if len(batch) > 0 {
            p.executeBatch(batch)
            batch = batch[:0] // Reset slice
        }
    }
}
```

### 3. Adaptive Scaling
```go
func (p *DynamicPool) monitor() {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            queueDepth := len(p.workChan)
            utilization := p.metrics.GetUtilization()
            
            if queueDepth > p.scaleUpThreshold && utilization > 0.8 {
                p.scaleUp()
            } else if queueDepth < p.scaleDownThreshold && utilization < 0.2 {
                p.scaleDown()
            }
            
        case <-p.quitChan:
            return
        }
    }
}
```

## Success Criteria

- [ ] Implement all four pool types correctly
- [ ] Achieve target throughput and latency metrics
- [ ] Demonstrate proper resource management
- [ ] Handle graceful shutdown correctly
- [ ] Implement comprehensive monitoring
- [ ] Show performance improvements over naive approaches

## Advanced Challenges

1. **NUMA-Aware Pools**: Optimize for NUMA topology
2. **GPU Work Pools**: Integrate GPU computation
3. **Distributed Pools**: Implement distributed work pools
4. **ML-Based Scaling**: Use machine learning for scaling decisions
5. **Real-Time Pools**: Implement real-time scheduling

## Real-World Applications

- **Web Servers**: Request processing pools
- **Data Processing**: ETL pipeline workers
- **Image Processing**: Parallel image transformation
- **Database Systems**: Query execution pools
- **Microservices**: Service mesh worker pools

## Next Steps

After completing this exercise:
1. Apply pool patterns to your applications
2. Experiment with hybrid pool designs
3. Integrate with observability systems
4. Study advanced scheduling algorithms
5. Explore actor model implementations

---

**Estimated Completion Time:** 45-60 minutes  
**Prerequisites:** Exercise 03-01  
**Next Exercise:** [Exercise 03-03: Work Distribution Patterns](../exercise-03-03/README.md)