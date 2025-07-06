# Exercise 04-02: Lock-Free Queue Implementation

**Objective:** Implement production-ready lock-free queue algorithms with proper memory management.

**Duration:** 60-75 minutes  
**Difficulty:** Advanced

## Learning Goals

By completing this exercise, you will:
- ✅ Implement Michael & Scott lock-free queue algorithm
- ✅ Understand and prevent ABA problems
- ✅ Design safe memory reclamation strategies
- ✅ Compare bounded vs unbounded queue implementations
- ✅ Ensure linearizability and correctness properties

## Background

Lock-free queues are fundamental data structures in high-performance concurrent systems. This exercise focuses on implementing correct, efficient, and safe lock-free queue algorithms that can handle high contention scenarios.

### Queue Algorithms

1. **Michael & Scott Algorithm**
   - Classic lock-free FIFO queue
   - Uses CAS for atomic updates
   - Requires careful memory management

2. **LCRQ (Load-Linked/Store-Conditional Ring Queue)**
   - Bounded lock-free queue
   - Better cache locality
   - Architecture-dependent

3. **Basket Queue**
   - Relaxed ordering for better performance
   - Allows temporary inconsistencies
   - Complex correctness properties

4. **Hazard Pointer Queue**
   - Safe memory reclamation
   - Prevents use-after-free
   - Higher memory overhead

### Key Challenges

1. **ABA Problem**
   - Pointer reuse causing false CAS success
   - Mitigation through versioning or hazard pointers
   - Critical for correctness

2. **Memory Reclamation**
   - Safe deallocation in concurrent environment
   - Hazard pointers vs epoch-based reclamation
   - Performance vs safety trade-offs

3. **Linearizability**
   - Operations appear atomic
   - Consistent ordering guarantees
   - Formal verification requirements

## Tasks

### Task 1: Basic Lock-Free Queue
Implement fundamental queue operations:
- Michael & Scott algorithm
- Basic enqueue and dequeue
- Simple memory management
- Correctness verification

### Task 2: ABA-Safe Implementation
Implement ABA problem prevention:
- Versioned pointers
- Hazard pointer integration
- Safe memory reclamation
- Stress testing under contention

### Task 3: Bounded Queue Variants
Implement bounded queue algorithms:
- Ring buffer based queue
- Fixed-size allocation
- Backpressure handling
- Performance comparison

### Task 4: Production Optimizations
Implement production-ready optimizations:
- Memory pooling
- Cache-friendly layouts
- NUMA awareness
- Comprehensive monitoring

## Files Overview

- `queue/` - Queue implementations
  - `michael_scott.go` - Classic Michael & Scott queue
  - `versioned_queue.go` - ABA-safe versioned queue
  - `hazard_queue.go` - Hazard pointer based queue
  - `bounded_queue.go` - Bounded lock-free queue
  - `ring_queue.go` - Ring buffer queue

- `memory/` - Memory management
  - `hazard_pointers.go` - Hazard pointer implementation
  - `epoch_reclamation.go` - Epoch-based reclamation
  - `memory_pool.go` - Node memory pooling
  - `safe_reclaim.go` - Safe memory reclamation

- `correctness/` - Correctness verification
  - `linearizability_test.go` - Linearizability testing
  - `stress_test.go` - High contention stress tests
  - `race_detector.go` - Race condition detection
  - `property_test.go` - Property-based testing

- `benchmarks/` - Performance benchmarks
  - `queue_bench_test.go` - Queue operation benchmarks
  - `contention_bench_test.go` - Contention analysis
  - `memory_bench_test.go` - Memory usage benchmarks
  - `scalability_bench_test.go` - Scalability tests

- `monitoring/` - Performance monitoring
  - `queue_metrics.go` - Queue performance metrics
  - `contention_monitor.go` - Contention monitoring
  - `memory_tracker.go` - Memory usage tracking

- `main.go` - Demonstration and testing
- `queue_test.go` - Comprehensive correctness tests

## Running the Exercise

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific queue benchmarks
go test -bench=BenchmarkMichaelScott -benchmem

# Profile CPU usage
go test -bench=BenchmarkQueue -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Profile memory usage
go test -bench=BenchmarkQueue -memprofile=mem.prof
go tool pprof mem.prof

# Test with race detector
go test -race

# Stress testing
go test -run=TestStress -timeout=30s

# Run demonstration
go run .

# Linearizability testing
go test -run=TestLinearizability
```

## Implementation Guidelines

### 1. Michael & Scott Queue
```go
package queue

import (
    "sync/atomic"
    "unsafe"
)

// Node represents a queue node
type Node struct {
    data interface{}
    next unsafe.Pointer // *Node
}

// MichaelScottQueue implements the classic lock-free queue
type MichaelScottQueue struct {
    head unsafe.Pointer // *Node
    tail unsafe.Pointer // *Node
}

func NewMichaelScottQueue() *MichaelScottQueue {
    dummy := &Node{}
    q := &MichaelScottQueue{
        head: unsafe.Pointer(dummy),
        tail: unsafe.Pointer(dummy),
    }
    return q
}

func (q *MichaelScottQueue) Enqueue(data interface{}) {
    newNode := &Node{data: data}
    newNodePtr := unsafe.Pointer(newNode)
    
    for {
        tail := atomic.LoadPointer(&q.tail)
        tailNode := (*Node)(tail)
        next := atomic.LoadPointer(&tailNode.next)
        
        // Check if tail is still the last node
        if tail == atomic.LoadPointer(&q.tail) {
            if next == nil {
                // Try to link new node at the end of the list
                if atomic.CompareAndSwapPointer(&tailNode.next, nil, newNodePtr) {
                    break // Successfully linked
                }
            } else {
                // Tail is lagging, try to advance it
                atomic.CompareAndSwapPointer(&q.tail, tail, next)
            }
        }
    }
    
    // Try to advance tail pointer
    atomic.CompareAndSwapPointer(&q.tail, unsafe.Pointer((*Node)(atomic.LoadPointer(&q.tail))), newNodePtr)
}

func (q *MichaelScottQueue) Dequeue() (interface{}, bool) {
    for {
        head := atomic.LoadPointer(&q.head)
        tail := atomic.LoadPointer(&q.tail)
        headNode := (*Node)(head)
        next := atomic.LoadPointer(&headNode.next)
        
        // Check consistency
        if head == atomic.LoadPointer(&q.head) {
            if head == tail {
                if next == nil {
                    // Queue is empty
                    return nil, false
                }
                // Tail is lagging, try to advance it
                atomic.CompareAndSwapPointer(&q.tail, tail, next)
            } else {
                if next == nil {
                    // Inconsistent state, retry
                    continue
                }
                
                nextNode := (*Node)(next)
                data := nextNode.data
                
                // Try to move head to next node
                if atomic.CompareAndSwapPointer(&q.head, head, next) {
                    // Successfully dequeued
                    return data, true
                }
            }
        }
    }
}

func (q *MichaelScottQueue) IsEmpty() bool {
    head := atomic.LoadPointer(&q.head)
    tail := atomic.LoadPointer(&q.tail)
    headNode := (*Node)(head)
    next := atomic.LoadPointer(&headNode.next)
    
    return head == tail && next == nil
}
```

### 2. ABA-Safe Versioned Queue
```go
package queue

import (
    "sync/atomic"
    "unsafe"
)

// VersionedPointer combines pointer with version to prevent ABA
type VersionedPointer struct {
    ptr     unsafe.Pointer
    version uint64
}

// VersionedNode with version information
type VersionedNode struct {
    data interface{}
    next VersionedPointer
}

// VersionedQueue prevents ABA problem using versioning
type VersionedQueue struct {
    head VersionedPointer
    tail VersionedPointer
}

func NewVersionedQueue() *VersionedQueue {
    dummy := &VersionedNode{}
    vp := VersionedPointer{
        ptr:     unsafe.Pointer(dummy),
        version: 0,
    }
    
    return &VersionedQueue{
        head: vp,
        tail: vp,
    }
}

// compareAndSwapVersionedPointer atomically compares and swaps versioned pointer
func compareAndSwapVersionedPointer(addr *VersionedPointer, old, new VersionedPointer) bool {
    // This is a simplified implementation
    // In practice, you'd need architecture-specific double-width CAS
    oldVal := (*uint128)(unsafe.Pointer(addr))
    newVal := packVersionedPointer(new)
    expectedVal := packVersionedPointer(old)
    
    return atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(oldVal)), 
        expectedVal.low, newVal.low) &&
        atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(uintptr(unsafe.Pointer(oldVal))+8)), 
        expectedVal.high, newVal.high)
}

type uint128 struct {
    low  uint64
    high uint64
}

func packVersionedPointer(vp VersionedPointer) uint128 {
    return uint128{
        low:  uint64(uintptr(vp.ptr)),
        high: vp.version,
    }
}

func unpackVersionedPointer(val uint128) VersionedPointer {
    return VersionedPointer{
        ptr:     unsafe.Pointer(uintptr(val.low)),
        version: val.high,
    }
}

func (q *VersionedQueue) Enqueue(data interface{}) {
    newNode := &VersionedNode{data: data}
    
    for {
        tail := atomic.LoadUint64((*uint64)(unsafe.Pointer(&q.tail)))
        tailVP := unpackVersionedPointer(uint128{low: tail})
        tailNode := (*VersionedNode)(tailVP.ptr)
        
        next := atomic.LoadUint64((*uint64)(unsafe.Pointer(&tailNode.next)))
        nextVP := unpackVersionedPointer(uint128{low: next})
        
        // Verify tail hasn't changed
        if tail == atomic.LoadUint64((*uint64)(unsafe.Pointer(&q.tail))) {
            if nextVP.ptr == nil {
                // Try to link new node
                newNext := VersionedPointer{
                    ptr:     unsafe.Pointer(newNode),
                    version: nextVP.version + 1,
                }
                
                if compareAndSwapVersionedPointer(&tailNode.next, nextVP, newNext) {
                    break
                }
            } else {
                // Advance tail
                newTail := VersionedPointer{
                    ptr:     nextVP.ptr,
                    version: tailVP.version + 1,
                }
                compareAndSwapVersionedPointer(&q.tail, tailVP, newTail)
            }
        }
    }
    
    // Advance tail
    tail := atomic.LoadUint64((*uint64)(unsafe.Pointer(&q.tail)))
    tailVP := unpackVersionedPointer(uint128{low: tail})
    newTail := VersionedPointer{
        ptr:     unsafe.Pointer(newNode),
        version: tailVP.version + 1,
    }
    compareAndSwapVersionedPointer(&q.tail, tailVP, newTail)
}

func (q *VersionedQueue) Dequeue() (interface{}, bool) {
    for {
        head := atomic.LoadUint64((*uint64)(unsafe.Pointer(&q.head)))
        tail := atomic.LoadUint64((*uint64)(unsafe.Pointer(&q.tail)))
        
        headVP := unpackVersionedPointer(uint128{low: head})
        tailVP := unpackVersionedPointer(uint128{low: tail})
        
        headNode := (*VersionedNode)(headVP.ptr)
        next := atomic.LoadUint64((*uint64)(unsafe.Pointer(&headNode.next)))
        nextVP := unpackVersionedPointer(uint128{low: next})
        
        // Verify head hasn't changed
        if head == atomic.LoadUint64((*uint64)(unsafe.Pointer(&q.head))) {
            if headVP.ptr == tailVP.ptr {
                if nextVP.ptr == nil {
                    return nil, false // Empty queue
                }
                // Advance tail
                newTail := VersionedPointer{
                    ptr:     nextVP.ptr,
                    version: tailVP.version + 1,
                }
                compareAndSwapVersionedPointer(&q.tail, tailVP, newTail)
            } else {
                if nextVP.ptr == nil {
                    continue // Inconsistent state
                }
                
                nextNode := (*VersionedNode)(nextVP.ptr)
                data := nextNode.data
                
                // Advance head
                newHead := VersionedPointer{
                    ptr:     nextVP.ptr,
                    version: headVP.version + 1,
                }
                
                if compareAndSwapVersionedPointer(&q.head, headVP, newHead) {
                    return data, true
                }
            }
        }
    }
}
```

### 3. Hazard Pointer Queue
```go
package memory

import (
    "runtime"
    "sync"
    "sync/atomic"
    "unsafe"
)

// HazardPointer manages safe memory reclamation
type HazardPointer struct {
    ptr unsafe.Pointer
    _   [7]uintptr // Cache line padding
}

// HazardPointerManager manages hazard pointers for all threads
type HazardPointerManager struct {
    hazards   []HazardPointer
    retired   []unsafe.Pointer
    retiredMu sync.Mutex
    threshold int
}

func NewHazardPointerManager(maxThreads int) *HazardPointerManager {
    return &HazardPointerManager{
        hazards:   make([]HazardPointer, maxThreads),
        retired:   make([]unsafe.Pointer, 0),
        threshold: maxThreads * 2, // Retire when we have 2x hazards
    }
}

func (hpm *HazardPointerManager) Acquire(threadID int, ptr unsafe.Pointer) {
    atomic.StorePointer(&hpm.hazards[threadID].ptr, ptr)
}

func (hpm *HazardPointerManager) Release(threadID int) {
    atomic.StorePointer(&hpm.hazards[threadID].ptr, nil)
}

func (hpm *HazardPointerManager) Retire(ptr unsafe.Pointer) {
    hpm.retiredMu.Lock()
    hpm.retired = append(hpm.retired, ptr)
    
    if len(hpm.retired) >= hpm.threshold {
        hpm.reclaimMemory()
    }
    hpm.retiredMu.Unlock()
}

func (hpm *HazardPointerManager) reclaimMemory() {
    // Collect all currently protected pointers
    protected := make(map[unsafe.Pointer]bool)
    for i := range hpm.hazards {
        ptr := atomic.LoadPointer(&hpm.hazards[i].ptr)
        if ptr != nil {
            protected[ptr] = true
        }
    }
    
    // Reclaim unprotected pointers
    newRetired := make([]unsafe.Pointer, 0)
    for _, ptr := range hpm.retired {
        if !protected[ptr] {
            // Safe to reclaim
            // In practice, you'd call the appropriate deallocator
            _ = ptr // Placeholder for actual deallocation
        } else {
            newRetired = append(newRetired, ptr)
        }
    }
    
    hpm.retired = newRetired
}

// HazardQueue implements queue with hazard pointer protection
type HazardQueue struct {
    head unsafe.Pointer // *Node
    tail unsafe.Pointer // *Node
    hpm  *HazardPointerManager
}

func NewHazardQueue(maxThreads int) *HazardQueue {
    dummy := &Node{}
    return &HazardQueue{
        head: unsafe.Pointer(dummy),
        tail: unsafe.Pointer(dummy),
        hpm:  NewHazardPointerManager(maxThreads),
    }
}

func (q *HazardQueue) getThreadID() int {
    // Simplified thread ID - in practice use proper thread-local storage
    return runtime.NumGoroutine() % len(q.hpm.hazards)
}

func (q *HazardQueue) Enqueue(data interface{}) {
    newNode := &Node{data: data}
    threadID := q.getThreadID()
    
    for {
        tail := atomic.LoadPointer(&q.tail)
        q.hpm.Acquire(threadID, tail)
        
        // Verify tail hasn't changed after acquiring hazard pointer
        if tail != atomic.LoadPointer(&q.tail) {
            continue
        }
        
        tailNode := (*Node)(tail)
        next := atomic.LoadPointer(&tailNode.next)
        
        if tail == atomic.LoadPointer(&q.tail) {
            if next == nil {
                if atomic.CompareAndSwapPointer(&tailNode.next, nil, unsafe.Pointer(newNode)) {
                    q.hpm.Release(threadID)
                    break
                }
            } else {
                atomic.CompareAndSwapPointer(&q.tail, tail, next)
            }
        }
    }
    
    atomic.CompareAndSwapPointer(&q.tail, atomic.LoadPointer(&q.tail), unsafe.Pointer(newNode))
}

func (q *HazardQueue) Dequeue() (interface{}, bool) {
    threadID := q.getThreadID()
    
    for {
        head := atomic.LoadPointer(&q.head)
        q.hpm.Acquire(threadID, head)
        
        // Verify head hasn't changed
        if head != atomic.LoadPointer(&q.head) {
            continue
        }
        
        tail := atomic.LoadPointer(&q.tail)
        headNode := (*Node)(head)
        next := atomic.LoadPointer(&headNode.next)
        
        if head == atomic.LoadPointer(&q.head) {
            if head == tail {
                if next == nil {
                    q.hpm.Release(threadID)
                    return nil, false
                }
                atomic.CompareAndSwapPointer(&q.tail, tail, next)
            } else {
                if next == nil {
                    continue
                }
                
                nextNode := (*Node)(next)
                data := nextNode.data
                
                if atomic.CompareAndSwapPointer(&q.head, head, next) {
                    q.hpm.Release(threadID)
                    q.hpm.Retire(head) // Safe to retire old head
                    return data, true
                }
            }
        }
    }
}
```

### 4. Bounded Lock-Free Queue
```go
package queue

import (
    "sync/atomic"
    "unsafe"
)

// BoundedQueue implements a bounded lock-free queue using ring buffer
type BoundedQueue struct {
    buffer   []unsafe.Pointer
    mask     uint64
    head     uint64
    tail     uint64
    _        [7]uint64 // Padding
}

func NewBoundedQueue(size int) *BoundedQueue {
    // Size must be power of 2
    if size&(size-1) != 0 {
        panic("size must be power of 2")
    }
    
    return &BoundedQueue{
        buffer: make([]unsafe.Pointer, size),
        mask:   uint64(size - 1),
    }
}

func (q *BoundedQueue) Enqueue(data interface{}) bool {
    for {
        tail := atomic.LoadUint64(&q.tail)
        head := atomic.LoadUint64(&q.head)
        
        // Check if queue is full
        if tail-head >= uint64(len(q.buffer)) {
            return false
        }
        
        index := tail & q.mask
        
        // Try to reserve slot
        if atomic.CompareAndSwapUint64(&q.tail, tail, tail+1) {
            // Store data
            for {
                if atomic.CompareAndSwapPointer(&q.buffer[index], nil, unsafe.Pointer(&data)) {
                    return true
                }
                // Spin until slot is available
                runtime.Gosched()
            }
        }
    }
}

func (q *BoundedQueue) Dequeue() (interface{}, bool) {
    for {
        head := atomic.LoadUint64(&q.head)
        tail := atomic.LoadUint64(&q.tail)
        
        // Check if queue is empty
        if head >= tail {
            return nil, false
        }
        
        index := head & q.mask
        
        // Try to reserve slot
        if atomic.CompareAndSwapUint64(&q.head, head, head+1) {
            // Load data
            for {
                ptr := atomic.LoadPointer(&q.buffer[index])
                if ptr != nil {
                    if atomic.CompareAndSwapPointer(&q.buffer[index], ptr, nil) {
                        data := *(*interface{})(ptr)
                        return data, true
                    }
                }
                // Spin until data is available
                runtime.Gosched()
            }
        }
    }
}

func (q *BoundedQueue) Size() int {
    tail := atomic.LoadUint64(&q.tail)
    head := atomic.LoadUint64(&q.head)
    return int(tail - head)
}

func (q *BoundedQueue) Capacity() int {
    return len(q.buffer)
}

func (q *BoundedQueue) IsEmpty() bool {
    return q.Size() == 0
}

func (q *BoundedQueue) IsFull() bool {
    return q.Size() >= q.Capacity()
}
```

## Performance Targets

| Queue Type | Enqueue Throughput | Dequeue Throughput | Memory Overhead |
|------------|-------------------|-------------------|----------------|
| Michael & Scott | >10M ops/sec | >10M ops/sec | 2 pointers/node |
| Versioned | >8M ops/sec | >8M ops/sec | 3 words/node |
| Hazard Pointer | >6M ops/sec | >6M ops/sec | Variable |
| Bounded | >20M ops/sec | >20M ops/sec | Fixed array |

## Key Metrics to Monitor

1. **Throughput**: Operations per second under various loads
2. **Latency**: Time per operation (P50, P99, P99.9)
3. **Contention**: CAS failure rates
4. **Memory Usage**: Bytes per node, total memory consumption
5. **Correctness**: Linearizability violations
6. **Scalability**: Performance vs number of threads

## Success Criteria

- [ ] Implement correct Michael & Scott queue
- [ ] Demonstrate ABA problem prevention
- [ ] Achieve target throughput metrics
- [ ] Pass linearizability tests
- [ ] Implement safe memory reclamation
- [ ] Show performance scaling across cores

## Advanced Challenges

1. **NUMA-Aware Queue**: Optimize for NUMA architectures
2. **Priority Queue**: Implement lock-free priority queue
3. **Batched Operations**: Support batch enqueue/dequeue
4. **Persistent Queue**: Add durability guarantees
5. **Distributed Queue**: Extend to distributed systems

## Real-World Applications

- **Message Passing**: Inter-thread communication
- **Task Scheduling**: Work queue implementations
- **Network Processing**: Packet queues in routers
- **Database Systems**: Transaction queues
- **Real-Time Systems**: Bounded latency guarantees

---

**Estimated Completion Time:** 60-75 minutes  
**Prerequisites:** Exercise 04-01  
**Next Exercise:** [Exercise 04-03: Compare-and-Swap Patterns](../exercise-04-03/README.md)