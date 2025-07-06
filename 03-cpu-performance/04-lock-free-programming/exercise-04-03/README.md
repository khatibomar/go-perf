# Exercise 04-03: Compare-and-Swap Patterns

**Objective:** Master CAS-based algorithm design patterns and implement optimized lock-free data structures.

**Duration:** 60-75 minutes  
**Difficulty:** Advanced

## Learning Goals

By completing this exercise, you will:
- ✅ Design CAS-based algorithm patterns
- ✅ Implement lock-free stack operations
- ✅ Master optimistic concurrency control
- ✅ Develop effective retry strategies and backoff mechanisms
- ✅ Build lock-free linked list operations

## Background

Compare-and-Swap (CAS) is the fundamental building block of lock-free programming. This exercise explores advanced CAS-based patterns and their application to various data structures and algorithms.

### CAS Operation Fundamentals

1. **Atomic Nature**
   - Compares memory location with expected value
   - Updates only if comparison succeeds
   - Returns success/failure indication

2. **ABA Problem**
   - Value changes from A to B and back to A
   - CAS succeeds but intermediate state was different
   - Requires careful algorithm design

3. **Memory Ordering**
   - Sequential consistency by default
   - Can be relaxed for performance
   - Must maintain algorithm correctness

### Common CAS Patterns

1. **CAS Loop**
   - Retry until success or abort condition
   - Most common lock-free pattern
   - Requires backoff for performance

2. **Optimistic Update**
   - Read, compute, attempt update
   - Retry on failure
   - Good for low contention

3. **Helping Mechanism**
   - Threads help complete others' operations
   - Ensures progress in high contention
   - Complex but wait-free

4. **Versioning**
   - Combine data with version counter
   - Prevents ABA problems
   - Requires wider CAS operations

## Tasks

### Task 1: Basic CAS Patterns
Implement fundamental CAS-based operations:
- Simple CAS loops with retry logic
- Optimistic update patterns
- Basic backoff strategies
- Performance measurement

### Task 2: Lock-Free Stack
Implement complete lock-free stack:
- CAS-based push and pop operations
- ABA problem handling
- Memory management
- Correctness verification

### Task 3: Advanced CAS Algorithms
Implement sophisticated CAS-based structures:
- Lock-free linked list with insert/delete
- Optimistic concurrency control
- Helping mechanisms
- Complex retry strategies

### Task 4: Performance Optimization
Optimize CAS-based algorithms:
- Adaptive backoff mechanisms
- Contention-aware algorithms
- Cache-friendly implementations
- Scalability analysis

## Files Overview

- `cas/` - CAS pattern implementations
  - `basic_patterns.go` - Fundamental CAS patterns
  - `optimistic_update.go` - Optimistic concurrency patterns
  - `helping_mechanism.go` - Wait-free helping patterns
  - `versioning.go` - ABA-safe versioning patterns

- `stack/` - Lock-free stack implementations
  - `simple_stack.go` - Basic CAS-based stack
  - `aba_safe_stack.go` - ABA-resistant stack
  - `versioned_stack.go` - Version-based stack
  - `helping_stack.go` - Wait-free stack with helping

- `list/` - Lock-free linked list
  - `lock_free_list.go` - Lock-free linked list
  - `optimistic_list.go` - Optimistic list operations
  - `lazy_list.go` - Lazy synchronization list
  - `harris_list.go` - Harris lock-free list

- `backoff/` - Retry and backoff strategies
  - `exponential_backoff.go` - Exponential backoff
  - `adaptive_backoff.go` - Adaptive backoff
  - `contention_manager.go` - Contention management
  - `yield_strategies.go` - CPU yield strategies

- `benchmarks/` - Performance benchmarks
  - `cas_bench_test.go` - CAS operation benchmarks
  - `stack_bench_test.go` - Stack performance tests
  - `list_bench_test.go` - List operation benchmarks
  - `contention_bench_test.go` - Contention analysis

- `correctness/` - Correctness verification
  - `linearizability_test.go` - Linearizability testing
  - `aba_test.go` - ABA problem detection
  - `stress_test.go` - High contention stress tests

- `main.go` - Demonstration and analysis
- `cas_test.go` - Comprehensive tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific CAS benchmarks
go test -bench=BenchmarkCAS -benchmem

# Profile CPU usage
go test -bench=BenchmarkStack -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Test with race detector
go test -race

# Stress testing
go test -run=TestStress -timeout=30s

# Run demonstration
go run .

# ABA problem testing
go test -run=TestABA
```

## Implementation Guidelines

### 1. Basic CAS Patterns
```go
package cas

import (
    "sync/atomic"
    "runtime"
)

// SimpleCASLoop demonstrates basic CAS retry pattern
func SimpleCASLoop(addr *int64, update func(int64) int64) bool {
    for {
        old := atomic.LoadInt64(addr)
        new := update(old)
        
        if atomic.CompareAndSwapInt64(addr, old, new) {
            return true
        }
        
        // Simple yield on failure
        runtime.Gosched()
    }
}

// OptimisticUpdate implements optimistic concurrency pattern
func OptimisticUpdate(addr *int64, compute func(int64) (int64, bool)) bool {
    old := atomic.LoadInt64(addr)
    new, shouldUpdate := compute(old)
    
    if !shouldUpdate {
        return false
    }
    
    return atomic.CompareAndSwapInt64(addr, old, new)
}

// CASWithRetryLimit limits retry attempts
func CASWithRetryLimit(addr *int64, old, new int64, maxRetries int) bool {
    for i := 0; i < maxRetries; i++ {
        if atomic.CompareAndSwapInt64(addr, old, new) {
            return true
        }
        
        // Check if value changed (someone else succeeded)
        current := atomic.LoadInt64(addr)
        if current != old {
            return false
        }
        
        runtime.Gosched()
    }
    return false
}

// VersionedCAS prevents ABA using version numbers
type VersionedValue struct {
    value   int64
    version int64
}

func VersionedCAS(addr *VersionedValue, expectedValue, newValue int64) bool {
    for {
        current := atomic.LoadInt64((*int64)(&addr.value))
        version := atomic.LoadInt64(&addr.version)
        
        if current != expectedValue {
            return false
        }
        
        // Try to update both value and version atomically
        // This is simplified - real implementation needs double-width CAS
        if atomic.CompareAndSwapInt64((*int64)(&addr.value), current, newValue) {
            atomic.AddInt64(&addr.version, 1)
            return true
        }
    }
}
```

### 2. Lock-Free Stack Implementation
```go
package stack

import (
    "sync/atomic"
    "unsafe"
)

// Node represents a stack node
type Node struct {
    data interface{}
    next unsafe.Pointer // *Node
}

// LockFreeStack implements a lock-free stack using CAS
type LockFreeStack struct {
    head unsafe.Pointer // *Node
}

func NewLockFreeStack() *LockFreeStack {
    return &LockFreeStack{}
}

func (s *LockFreeStack) Push(data interface{}) {
    newNode := &Node{data: data}
    
    for {
        head := atomic.LoadPointer(&s.head)
        newNode.next = head
        
        if atomic.CompareAndSwapPointer(&s.head, head, unsafe.Pointer(newNode)) {
            return
        }
    }
}

func (s *LockFreeStack) Pop() (interface{}, bool) {
    for {
        head := atomic.LoadPointer(&s.head)
        if head == nil {
            return nil, false
        }
        
        headNode := (*Node)(head)
        next := atomic.LoadPointer(&headNode.next)
        
        if atomic.CompareAndSwapPointer(&s.head, head, next) {
            return headNode.data, true
        }
    }
}

func (s *LockFreeStack) IsEmpty() bool {
    return atomic.LoadPointer(&s.head) == nil
}

// ABA-Safe Stack using hazard pointers
type ABASafeStack struct {
    head unsafe.Pointer // *Node
    hp   *HazardPointerManager
}

func NewABASafeStack(maxThreads int) *ABASafeStack {
    return &ABASafeStack{
        hp: NewHazardPointerManager(maxThreads),
    }
}

func (s *ABASafeStack) Push(data interface{}) {
    newNode := &Node{data: data}
    
    for {
        head := atomic.LoadPointer(&s.head)
        newNode.next = head
        
        if atomic.CompareAndSwapPointer(&s.head, head, unsafe.Pointer(newNode)) {
            return
        }
    }
}

func (s *ABASafeStack) Pop() (interface{}, bool) {
    threadID := s.getThreadID()
    
    for {
        head := atomic.LoadPointer(&s.head)
        if head == nil {
            return nil, false
        }
        
        // Protect head with hazard pointer
        s.hp.Acquire(threadID, head)
        
        // Verify head hasn't changed
        if head != atomic.LoadPointer(&s.head) {
            continue
        }
        
        headNode := (*Node)(head)
        next := atomic.LoadPointer(&headNode.next)
        
        if atomic.CompareAndSwapPointer(&s.head, head, next) {
            s.hp.Release(threadID)
            data := headNode.data
            s.hp.Retire(head)
            return data, true
        }
    }
}

func (s *ABASafeStack) getThreadID() int {
    // Simplified thread ID
    return runtime.NumGoroutine() % s.hp.maxThreads
}
```

### 3. Lock-Free Linked List
```go
package list

import (
    "sync/atomic"
    "unsafe"
)

// ListNode represents a node in the linked list
type ListNode struct {
    key     int64
    value   interface{}
    next    unsafe.Pointer // *ListNode
    marked  int32          // Deletion mark
}

// LockFreeList implements Harris's lock-free linked list
type LockFreeList struct {
    head *ListNode
    tail *ListNode
}

func NewLockFreeList() *LockFreeList {
    head := &ListNode{key: -1} // Sentinel
    tail := &ListNode{key: 1<<63 - 1} // Max sentinel
    head.next = unsafe.Pointer(tail)
    
    return &LockFreeList{
        head: head,
        tail: tail,
    }
}

// Search finds the position for a key
func (l *LockFreeList) Search(key int64) (*ListNode, *ListNode) {
    var pred, curr *ListNode
    
retry:
    pred = l.head
    curr = (*ListNode)(atomic.LoadPointer(&pred.next))
    
    for {
        // Remove marked nodes
        for atomic.LoadInt32(&curr.marked) == 1 {
            next := (*ListNode)(atomic.LoadPointer(&curr.next))
            if !atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next)) {
                goto retry
            }
            curr = next
        }
        
        if curr.key >= key {
            return pred, curr
        }
        
        pred = curr
        curr = (*ListNode)(atomic.LoadPointer(&curr.next))
    }
}

func (l *LockFreeList) Insert(key int64, value interface{}) bool {
    newNode := &ListNode{
        key:   key,
        value: value,
    }
    
    for {
        pred, curr := l.Search(key)
        
        if curr.key == key {
            return false // Key already exists
        }
        
        newNode.next = unsafe.Pointer(curr)
        
        if atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(newNode)) {
            return true
        }
    }
}

func (l *LockFreeList) Delete(key int64) bool {
    for {
        pred, curr := l.Search(key)
        
        if curr.key != key {
            return false // Key not found
        }
        
        next := (*ListNode)(atomic.LoadPointer(&curr.next))
        
        // Mark node for deletion
        if !atomic.CompareAndSwapInt32(&curr.marked, 0, 1) {
            continue // Already marked
        }
        
        // Try to physically remove
        atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next))
        return true
    }
}

func (l *LockFreeList) Contains(key int64) bool {
    curr := (*ListNode)(atomic.LoadPointer(&l.head.next))
    
    for curr.key < key {
        curr = (*ListNode)(atomic.LoadPointer(&curr.next))
    }
    
    return curr.key == key && atomic.LoadInt32(&curr.marked) == 0
}

// OptimisticList uses optimistic synchronization
type OptimisticList struct {
    head *ListNode
    tail *ListNode
}

func NewOptimisticList() *OptimisticList {
    head := &ListNode{key: -1}
    tail := &ListNode{key: 1<<63 - 1}
    head.next = unsafe.Pointer(tail)
    
    return &OptimisticList{
        head: head,
        tail: tail,
    }
}

func (l *OptimisticList) validate(pred, curr *ListNode) bool {
    return atomic.LoadInt32(&pred.marked) == 0 &&
           atomic.LoadInt32(&curr.marked) == 0 &&
           atomic.LoadPointer(&pred.next) == unsafe.Pointer(curr)
}

func (l *OptimisticList) Insert(key int64, value interface{}) bool {
    for {
        pred, curr := l.optimisticSearch(key)
        
        if curr.key == key {
            return false
        }
        
        newNode := &ListNode{
            key:   key,
            value: value,
            next:  unsafe.Pointer(curr),
        }
        
        if l.validate(pred, curr) {
            if atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(newNode)) {
                return true
            }
        }
    }
}

func (l *OptimisticList) optimisticSearch(key int64) (*ListNode, *ListNode) {
    pred := l.head
    curr := (*ListNode)(atomic.LoadPointer(&pred.next))
    
    for curr.key < key {
        pred = curr
        curr = (*ListNode)(atomic.LoadPointer(&curr.next))
    }
    
    return pred, curr
}
```

### 4. Advanced Backoff Strategies
```go
package backoff

import (
    "math/rand"
    "runtime"
    "sync/atomic"
    "time"
)

// ContentionManager manages backoff strategies
type ContentionManager struct {
    failures    int64
    successes   int64
    currentWait time.Duration
    minWait     time.Duration
    maxWait     time.Duration
    factor      float64
}

func NewContentionManager(min, max time.Duration) *ContentionManager {
    return &ContentionManager{
        minWait:     min,
        maxWait:     max,
        currentWait: min,
        factor:      2.0,
    }
}

func (cm *ContentionManager) OnSuccess() {
    atomic.AddInt64(&cm.successes, 1)
    
    // Reduce wait time on success
    if cm.currentWait > cm.minWait {
        cm.currentWait = time.Duration(float64(cm.currentWait) / cm.factor)
        if cm.currentWait < cm.minWait {
            cm.currentWait = cm.minWait
        }
    }
}

func (cm *ContentionManager) OnFailure() {
    atomic.AddInt64(&cm.failures, 1)
    
    // Increase wait time on failure
    if cm.currentWait < cm.maxWait {
        cm.currentWait = time.Duration(float64(cm.currentWait) * cm.factor)
        if cm.currentWait > cm.maxWait {
            cm.currentWait = cm.maxWait
        }
    }
}

func (cm *ContentionManager) Backoff() {
    if cm.currentWait > 0 {
        // Add jitter to prevent thundering herd
        jitter := time.Duration(rand.Int63n(int64(cm.currentWait / 2)))
        time.Sleep(cm.currentWait + jitter)
    } else {
        runtime.Gosched()
    }
}

func (cm *ContentionManager) GetStats() (int64, int64, float64) {
    successes := atomic.LoadInt64(&cm.successes)
    failures := atomic.LoadInt64(&cm.failures)
    total := successes + failures
    
    var successRate float64
    if total > 0 {
        successRate = float64(successes) / float64(total)
    }
    
    return successes, failures, successRate
}

// AdaptiveBackoff adjusts strategy based on system load
type AdaptiveBackoff struct {
    cm           *ContentionManager
    cpuThreshold float64
    lastCPUCheck time.Time
    highCPUMode  bool
}

func NewAdaptiveBackoff(min, max time.Duration) *AdaptiveBackoff {
    return &AdaptiveBackoff{
        cm:           NewContentionManager(min, max),
        cpuThreshold: 0.8,
        lastCPUCheck: time.Now(),
    }
}

func (ab *AdaptiveBackoff) Backoff() {
    // Check CPU usage periodically
    if time.Since(ab.lastCPUCheck) > time.Second {
        ab.updateCPUMode()
        ab.lastCPUCheck = time.Now()
    }
    
    if ab.highCPUMode {
        // More aggressive backoff under high CPU load
        time.Sleep(ab.cm.currentWait * 2)
    } else {
        ab.cm.Backoff()
    }
}

func (ab *AdaptiveBackoff) updateCPUMode() {
    // Simplified CPU usage check
    // In practice, you'd use system calls to get actual CPU usage
    numGoroutines := runtime.NumGoroutine()
    numCPU := runtime.NumCPU()
    
    cpuRatio := float64(numGoroutines) / float64(numCPU)
    ab.highCPUMode = cpuRatio > ab.cpuThreshold
}

// YieldStrategy implements different yielding strategies
type YieldStrategy int

const (
    YieldGosched YieldStrategy = iota
    YieldSleep
    YieldSpin
    YieldAdaptive
)

func Yield(strategy YieldStrategy, iteration int) {
    switch strategy {
    case YieldGosched:
        runtime.Gosched()
        
    case YieldSleep:
        time.Sleep(time.Nanosecond)
        
    case YieldSpin:
        // Spin for a few cycles
        for i := 0; i < 100; i++ {
            runtime.KeepAlive(i)
        }
        
    case YieldAdaptive:
        if iteration < 10 {
            // Spin for first few iterations
            for i := 0; i < iteration*10; i++ {
                runtime.KeepAlive(i)
            }
        } else if iteration < 100 {
            runtime.Gosched()
        } else {
            time.Sleep(time.Nanosecond * time.Duration(iteration-100))
        }
    }
}
```

## Performance Targets

| Operation | Target Throughput | Target Latency | Scalability |
|-----------|------------------|----------------|-------------|
| CAS Loop | >50M ops/sec | <20ns P99 | Linear |
| Stack Push | >30M ops/sec | <30ns P99 | Near-linear |
| Stack Pop | >30M ops/sec | <30ns P99 | Near-linear |
| List Insert | >10M ops/sec | <100ns P99 | Sub-linear |
| List Delete | >10M ops/sec | <100ns P99 | Sub-linear |

## Key Metrics to Monitor

1. **CAS Success Rate**: Percentage of successful CAS operations
2. **Retry Count**: Average number of retries per operation
3. **Contention Level**: Number of concurrent operations
4. **Backoff Effectiveness**: Time spent in backoff vs useful work
5. **Memory Usage**: Allocation rate and garbage collection pressure
6. **Cache Performance**: Cache miss rates and false sharing

## Success Criteria

- [ ] Implement correct CAS-based algorithms
- [ ] Achieve target performance metrics
- [ ] Demonstrate ABA problem prevention
- [ ] Show effective backoff strategies
- [ ] Pass linearizability tests
- [ ] Scale performance across multiple cores

## Advanced Challenges

1. **Wait-Free Algorithms**: Implement wait-free data structures
2. **Hardware Transactional Memory**: Use HTM when available
3. **Persistent Data Structures**: Add durability to lock-free structures
4. **Distributed CAS**: Implement distributed consensus algorithms
5. **Formal Verification**: Prove correctness of implementations

## Real-World Applications

- **Database Systems**: Lock-free indexes and transaction processing
- **Operating Systems**: Kernel data structures and synchronization
- **High-Frequency Trading**: Ultra-low latency data structures
- **Game Engines**: Real-time collision detection and physics
- **Network Processing**: Lock-free packet processing pipelines

---

**Estimated Completion Time:** 60-75 minutes  
**Prerequisites:** Exercises 04-01 and 04-02  
**Next Exercise:** [Exercise 04-04: High-Performance Counter Design](../exercise-04-04/README.md)