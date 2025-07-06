# Exercise 03-04: Concurrent Pipeline Optimization

**Objective:** Design and optimize high-performance concurrent pipelines for data processing.

**Duration:** 60-75 minutes  
**Difficulty:** Expert

## Learning Goals

By completing this exercise, you will:
- ✅ Design efficient concurrent pipeline architectures
- ✅ Implement advanced buffering and backpressure mechanisms
- ✅ Optimize pipeline stages for maximum throughput
- ✅ Handle dynamic pipeline reconfiguration
- ✅ Implement fault tolerance and error propagation

## Background

Concurrent pipelines are fundamental to high-performance data processing systems. This exercise explores advanced techniques for building scalable, efficient, and robust pipeline architectures.

### Pipeline Patterns

1. **Linear Pipeline**
   - Sequential stages with buffering
   - Simple coordination
   - Limited parallelism within stages

2. **Fan-Out/Fan-In Pipeline**
   - Parallel processing within stages
   - Complex synchronization
   - Higher throughput potential

3. **Dynamic Pipeline**
   - Runtime stage addition/removal
   - Adaptive resource allocation
   - Complex lifecycle management

4. **Hierarchical Pipeline**
   - Nested pipeline structures
   - Multi-level optimization
   - Scalable architecture

5. **Stream Processing Pipeline**
   - Continuous data flow
   - Real-time processing
   - Low-latency requirements

## Tasks

### Task 1: Basic Pipeline Implementation
Implement fundamental pipeline structures:
- Linear pipeline with buffered stages
- Fan-out/fan-in pipeline
- Pipeline with error handling
- Configurable stage parallelism

### Task 2: Advanced Buffering Strategies
Implement sophisticated buffering mechanisms:
- Adaptive buffer sizing
- Backpressure propagation
- Priority-based buffering
- Memory-efficient buffering

### Task 3: Dynamic Pipeline Management
Implement runtime pipeline modification:
- Dynamic stage addition/removal
- Pipeline reconfiguration
- Load-based scaling
- Resource reallocation

### Task 4: Performance Optimization
Optimize pipeline performance:
- Minimize coordination overhead
- Optimize data flow patterns
- Implement efficient signaling
- Reduce memory allocations

## Files Overview

- `pipeline/` - Core pipeline implementations
  - `linear_pipeline.go` - Linear pipeline
  - `fanout_pipeline.go` - Fan-out/fan-in pipeline
  - `dynamic_pipeline.go` - Dynamic pipeline
  - `hierarchical_pipeline.go` - Hierarchical pipeline
  - `stream_pipeline.go` - Stream processing pipeline

- `stages/` - Pipeline stage implementations
  - `stage.go` - Base stage interface
  - `transform_stage.go` - Data transformation stage
  - `filter_stage.go` - Data filtering stage
  - `aggregate_stage.go` - Data aggregation stage
  - `parallel_stage.go` - Parallel processing stage

- `buffering/` - Buffering strategies
  - `adaptive_buffer.go` - Adaptive buffer sizing
  - `backpressure.go` - Backpressure mechanisms
  - `priority_buffer.go` - Priority-based buffering
  - `ring_buffer.go` - Lock-free ring buffer

- `coordination/` - Pipeline coordination
  - `coordinator.go` - Pipeline coordinator
  - `scheduler.go` - Stage scheduler
  - `load_balancer.go` - Load balancing
  - `fault_detector.go` - Fault detection

- `monitoring/` - Performance monitoring
  - `pipeline_monitor.go` - Pipeline monitoring
  - `stage_metrics.go` - Stage-level metrics
  - `throughput_tracker.go` - Throughput tracking
  - `latency_tracker.go` - Latency tracking

- `optimization/` - Performance optimizations
  - `batch_processor.go` - Batch processing
  - `cache_optimizer.go` - Cache optimization
  - `memory_optimizer.go` - Memory optimization
  - `cpu_optimizer.go` - CPU optimization

- `benchmarks/` - Performance benchmarks
  - `pipeline_bench_test.go` - Pipeline benchmarks
  - `stage_bench_test.go` - Stage benchmarks
  - `scalability_bench_test.go` - Scalability tests

- `main.go` - Demonstration and testing
- `pipeline_test.go` - Correctness tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific pipeline benchmarks
go test -bench=BenchmarkLinearPipeline -benchmem

# Profile CPU usage
go test -bench=BenchmarkPipeline -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Profile memory usage
go test -bench=BenchmarkPipeline -memprofile=mem.prof
go tool pprof mem.prof

# Trace execution
go test -bench=BenchmarkPipeline -trace=trace.out
go tool trace trace.out

# Run demonstration
go run .

# Test correctness
go test -v
```

## Implementation Guidelines

### 1. Linear Pipeline
```go
type LinearPipeline struct {
    stages []Stage
    buffers []chan interface{}
    ctx context.Context
    cancel context.CancelFunc
    wg sync.WaitGroup
    metrics *PipelineMetrics
}

func NewLinearPipeline(stages []Stage, bufferSize int) *LinearPipeline {
    ctx, cancel := context.WithCancel(context.Background())
    
    p := &LinearPipeline{
        stages: stages,
        buffers: make([]chan interface{}, len(stages)+1),
        ctx: ctx,
        cancel: cancel,
        metrics: NewPipelineMetrics(),
    }
    
    // Create buffers between stages
    for i := range p.buffers {
        p.buffers[i] = make(chan interface{}, bufferSize)
    }
    
    return p
}

func (p *LinearPipeline) Start() error {
    // Start each stage
    for i, stage := range p.stages {
        p.wg.Add(1)
        go p.runStage(i, stage)
    }
    
    return nil
}

func (p *LinearPipeline) runStage(index int, stage Stage) {
    defer p.wg.Done()
    
    input := p.buffers[index]
    output := p.buffers[index+1]
    
    for {
        select {
        case <-p.ctx.Done():
            return
            
        case data, ok := <-input:
            if !ok {
                close(output)
                return
            }
            
            start := time.Now()
            result, err := stage.Process(data)
            duration := time.Since(start)
            
            p.metrics.RecordStageLatency(index, duration)
            
            if err != nil {
                p.handleError(index, err)
                continue
            }
            
            select {
            case output <- result:
                p.metrics.IncrementThroughput(index)
            case <-p.ctx.Done():
                return
            }
        }
    }
}
```

### 2. Fan-Out/Fan-In Pipeline
```go
type FanOutFanInPipeline struct {
    inputStage Stage
    parallelStages []Stage
    outputStage Stage
    parallelism int
    bufferSize int
    ctx context.Context
    cancel context.CancelFunc
    wg sync.WaitGroup
}

func (p *FanOutFanInPipeline) Start() error {
    inputChan := make(chan interface{}, p.bufferSize)
    outputChan := make(chan interface{}, p.bufferSize)
    resultChans := make([]chan interface{}, p.parallelism)
    
    // Create result channels
    for i := range resultChans {
        resultChans[i] = make(chan interface{}, p.bufferSize)
    }
    
    // Start input stage
    p.wg.Add(1)
    go p.runInputStage(inputChan)
    
    // Start parallel stages
    for i := 0; i < p.parallelism; i++ {
        p.wg.Add(1)
        go p.runParallelStage(i, inputChan, resultChans[i])
    }
    
    // Start fan-in stage
    p.wg.Add(1)
    go p.runFanInStage(resultChans, outputChan)
    
    // Start output stage
    p.wg.Add(1)
    go p.runOutputStage(outputChan)
    
    return nil
}

func (p *FanOutFanInPipeline) runParallelStage(index int, input <-chan interface{}, output chan<- interface{}) {
    defer p.wg.Done()
    defer close(output)
    
    stage := p.parallelStages[index%len(p.parallelStages)]
    
    for {
        select {
        case <-p.ctx.Done():
            return
            
        case data, ok := <-input:
            if !ok {
                return
            }
            
            result, err := stage.Process(data)
            if err != nil {
                continue // Handle error appropriately
            }
            
            select {
            case output <- result:
            case <-p.ctx.Done():
                return
            }
        }
    }
}

func (p *FanOutFanInPipeline) runFanInStage(inputs []<-chan interface{}, output chan<- interface{}) {
    defer p.wg.Done()
    defer close(output)
    
    // Use select with reflection for dynamic fan-in
    cases := make([]reflect.SelectCase, len(inputs)+1)
    
    // Add context case
    cases[0] = reflect.SelectCase{
        Dir: reflect.SelectRecv,
        Chan: reflect.ValueOf(p.ctx.Done()),
    }
    
    // Add input cases
    for i, input := range inputs {
        cases[i+1] = reflect.SelectCase{
            Dir: reflect.SelectRecv,
            Chan: reflect.ValueOf(input),
        }
    }
    
    activeInputs := len(inputs)
    
    for activeInputs > 0 {
        chosen, value, ok := reflect.Select(cases)
        
        if chosen == 0 {
            // Context cancelled
            return
        }
        
        if !ok {
            // Input channel closed
            cases[chosen].Chan = reflect.Value{}
            activeInputs--
            continue
        }
        
        select {
        case output <- value.Interface():
        case <-p.ctx.Done():
            return
        }
    }
}
```

### 3. Adaptive Buffer
```go
type AdaptiveBuffer struct {
    buffer chan interface{}
    minSize int
    maxSize int
    currentSize int
    loadFactor float64
    resizeThreshold float64
    mu sync.RWMutex
    metrics *BufferMetrics
}

func NewAdaptiveBuffer(minSize, maxSize int) *AdaptiveBuffer {
    return &AdaptiveBuffer{
        buffer: make(chan interface{}, minSize),
        minSize: minSize,
        maxSize: maxSize,
        currentSize: minSize,
        resizeThreshold: 0.8,
        metrics: NewBufferMetrics(),
    }
}

func (b *AdaptiveBuffer) Send(data interface{}) error {
    b.updateLoadFactor()
    
    select {
    case b.buffer <- data:
        b.metrics.IncrementSent()
        return nil
    default:
        // Buffer full, try to resize
        if b.tryResize() {
            select {
            case b.buffer <- data:
                b.metrics.IncrementSent()
                return nil
            default:
                b.metrics.IncrementDropped()
                return errors.New("buffer full")
            }
        }
        b.metrics.IncrementDropped()
        return errors.New("buffer full")
    }
}

func (b *AdaptiveBuffer) tryResize() bool {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.loadFactor > b.resizeThreshold && b.currentSize < b.maxSize {
        // Increase buffer size
        newSize := min(b.currentSize*2, b.maxSize)
        newBuffer := make(chan interface{}, newSize)
        
        // Transfer existing data
        close(b.buffer)
        for data := range b.buffer {
            newBuffer <- data
        }
        
        b.buffer = newBuffer
        b.currentSize = newSize
        b.metrics.RecordResize(newSize)
        return true
    }
    
    return false
}

func (b *AdaptiveBuffer) updateLoadFactor() {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    b.loadFactor = float64(len(b.buffer)) / float64(b.currentSize)
    b.metrics.RecordLoadFactor(b.loadFactor)
}
```

### 4. Backpressure Mechanism
```go
type BackpressureController struct {
    thresholds map[string]float64
    actions map[string]BackpressureAction
    monitors map[string]*StageMonitor
    mu sync.RWMutex
}

type BackpressureAction interface {
    Execute(severity float64) error
}

type ThrottleAction struct {
    stage Stage
    maxRate float64
    limiter *rate.Limiter
}

func (a *ThrottleAction) Execute(severity float64) error {
    // Adjust rate limit based on severity
    newRate := a.maxRate * (1.0 - severity)
    a.limiter.SetLimit(rate.Limit(newRate))
    return nil
}

type DropAction struct {
    dropRate float64
}

func (a *DropAction) Execute(severity float64) error {
    a.dropRate = severity
    return nil
}

func (bc *BackpressureController) CheckAndApply() {
    bc.mu.RLock()
    defer bc.mu.RUnlock()
    
    for stageName, monitor := range bc.monitors {
        metrics := monitor.GetMetrics()
        
        // Calculate pressure based on multiple factors
        queuePressure := float64(metrics.QueueLength) / float64(metrics.QueueCapacity)
        latencyPressure := metrics.AvgLatency.Seconds() / metrics.TargetLatency.Seconds()
        
        pressure := math.Max(queuePressure, latencyPressure)
        
        if threshold, exists := bc.thresholds[stageName]; exists && pressure > threshold {
            if action, exists := bc.actions[stageName]; exists {
                severity := (pressure - threshold) / (1.0 - threshold)
                action.Execute(severity)
            }
        }
    }
}
```

### 5. Dynamic Pipeline Reconfiguration
```go
type DynamicPipeline struct {
    stages map[string]Stage
    connections map[string][]string
    buffers map[string]chan interface{}
    ctx context.Context
    cancel context.CancelFunc
    mu sync.RWMutex
    wg sync.WaitGroup
    running map[string]bool
}

func (p *DynamicPipeline) AddStage(name string, stage Stage, inputs, outputs []string) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    // Add stage
    p.stages[name] = stage
    p.connections[name] = outputs
    p.running[name] = false
    
    // Create buffers for outputs
    for _, output := range outputs {
        if _, exists := p.buffers[output]; !exists {
            p.buffers[output] = make(chan interface{}, 100)
        }
    }
    
    return nil
}

func (p *DynamicPipeline) RemoveStage(name string) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    // Stop stage if running
    if p.running[name] {
        // Signal stage to stop
        // Implementation depends on stage interface
    }
    
    // Remove from maps
    delete(p.stages, name)
    delete(p.connections, name)
    delete(p.running, name)
    
    return nil
}

func (p *DynamicPipeline) StartStage(name string) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.running[name] {
        return errors.New("stage already running")
    }
    
    stage := p.stages[name]
    outputs := p.connections[name]
    
    p.wg.Add(1)
    go p.runDynamicStage(name, stage, outputs)
    p.running[name] = true
    
    return nil
}

func (p *DynamicPipeline) runDynamicStage(name string, stage Stage, outputs []string) {
    defer p.wg.Done()
    
    inputBuffer := p.buffers[name]
    outputBuffers := make([]chan interface{}, len(outputs))
    
    for i, output := range outputs {
        outputBuffers[i] = p.buffers[output]
    }
    
    for {
        select {
        case <-p.ctx.Done():
            return
            
        case data, ok := <-inputBuffer:
            if !ok {
                return
            }
            
            result, err := stage.Process(data)
            if err != nil {
                continue
            }
            
            // Send to all output buffers
            for _, outputBuffer := range outputBuffers {
                select {
                case outputBuffer <- result:
                case <-p.ctx.Done():
                    return
                }
            }
        }
    }
}
```

## Performance Targets

| Pipeline Type | Throughput Target | Latency Target | Memory Usage |
|---------------|------------------|----------------|-------------|
| Linear | >500K items/sec | <1ms P99 | <100MB |
| Fan-Out/Fan-In | >1M items/sec | <2ms P99 | <200MB |
| Dynamic | >300K items/sec | <3ms P99 | <150MB |
| Hierarchical | >800K items/sec | <2ms P99 | <250MB |
| Stream | >1.5M items/sec | <0.5ms P99 | <300MB |

## Key Metrics to Monitor

1. **Throughput**: Items processed per second
2. **Latency**: End-to-end processing time
3. **Buffer Utilization**: Average buffer fill levels
4. **Backpressure Events**: Frequency of backpressure activation
5. **Stage Balance**: Load distribution across stages
6. **Memory Efficiency**: Memory usage per item
7. **CPU Utilization**: CPU usage across cores

## Optimization Techniques

### 1. Batch Processing
```go
type BatchProcessor struct {
    batchSize int
    timeout time.Duration
    processor func([]interface{}) ([]interface{}, error)
}

func (bp *BatchProcessor) Process(input <-chan interface{}, output chan<- interface{}) {
    batch := make([]interface{}, 0, bp.batchSize)
    timer := time.NewTimer(bp.timeout)
    
    for {
        select {
        case item := <-input:
            batch = append(batch, item)
            
            if len(batch) >= bp.batchSize {
                bp.processBatch(batch, output)
                batch = batch[:0]
                timer.Reset(bp.timeout)
            }
            
        case <-timer.C:
            if len(batch) > 0 {
                bp.processBatch(batch, output)
                batch = batch[:0]
            }
            timer.Reset(bp.timeout)
        }
    }
}
```

### 2. Lock-Free Ring Buffer
```go
type LockFreeRingBuffer struct {
    buffer []interface{}
    mask uint64
    readPos uint64
    writePos uint64
}

func NewLockFreeRingBuffer(size int) *LockFreeRingBuffer {
    // Size must be power of 2
    if size&(size-1) != 0 {
        panic("size must be power of 2")
    }
    
    return &LockFreeRingBuffer{
        buffer: make([]interface{}, size),
        mask: uint64(size - 1),
    }
}

func (rb *LockFreeRingBuffer) Write(item interface{}) bool {
    writePos := atomic.LoadUint64(&rb.writePos)
    readPos := atomic.LoadUint64(&rb.readPos)
    
    if writePos-readPos >= uint64(len(rb.buffer)) {
        return false // Buffer full
    }
    
    rb.buffer[writePos&rb.mask] = item
    atomic.StoreUint64(&rb.writePos, writePos+1)
    return true
}

func (rb *LockFreeRingBuffer) Read() (interface{}, bool) {
    readPos := atomic.LoadUint64(&rb.readPos)
    writePos := atomic.LoadUint64(&rb.writePos)
    
    if readPos >= writePos {
        return nil, false // Buffer empty
    }
    
    item := rb.buffer[readPos&rb.mask]
    atomic.StoreUint64(&rb.readPos, readPos+1)
    return item, true
}
```

### 3. Memory Pool Integration
```go
type PooledPipeline struct {
    *LinearPipeline
    objectPool sync.Pool
    bufferPool sync.Pool
}

func NewPooledPipeline(stages []Stage) *PooledPipeline {
    p := &PooledPipeline{
        LinearPipeline: NewLinearPipeline(stages, 0),
    }
    
    p.objectPool.New = func() interface{} {
        return &ProcessingContext{}
    }
    
    p.bufferPool.New = func() interface{} {
        return make([]byte, 4096)
    }
    
    return p
}

func (p *PooledPipeline) getContext() *ProcessingContext {
    return p.objectPool.Get().(*ProcessingContext)
}

func (p *PooledPipeline) putContext(ctx *ProcessingContext) {
    ctx.Reset()
    p.objectPool.Put(ctx)
}

func (p *PooledPipeline) getBuffer() []byte {
    return p.bufferPool.Get().([]byte)
}

func (p *PooledPipeline) putBuffer(buf []byte) {
    p.bufferPool.Put(buf[:0])
}
```

## Success Criteria

- [ ] Implement all five pipeline types correctly
- [ ] Achieve target throughput and latency metrics
- [ ] Demonstrate effective backpressure handling
- [ ] Show dynamic reconfiguration capabilities
- [ ] Implement comprehensive monitoring
- [ ] Create detailed performance analysis

## Advanced Challenges

1. **Distributed Pipeline**: Extend to multi-node processing
2. **ML-Optimized Pipeline**: Use machine learning for optimization
3. **Real-Time Pipeline**: Implement hard real-time guarantees
4. **Fault-Tolerant Pipeline**: Add comprehensive fault tolerance
5. **Energy-Efficient Pipeline**: Optimize for energy consumption

## Real-World Applications

- **Data Processing Systems**: ETL pipelines
- **Media Processing**: Video/audio processing
- **Financial Systems**: Trading pipelines
- **IoT Platforms**: Sensor data processing
- **Machine Learning**: Training data pipelines

## Next Steps

After completing this exercise:
1. Apply patterns to your specific use cases
2. Experiment with domain-specific optimizations
3. Study distributed pipeline architectures
4. Explore stream processing frameworks
5. Investigate hardware-accelerated processing

---

**Estimated Completion Time:** 60-75 minutes  
**Prerequisites:** Exercises 03-01, 03-02, and 03-03  
**Next Module:** [Module 04: Lock-Free Programming Patterns](../../04-lock-free-programming/README.md)