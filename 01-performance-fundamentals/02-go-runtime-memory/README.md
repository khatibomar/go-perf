# Chapter 02: Go Runtime and Memory Model

**Duration:** 6-8 hours  
**Difficulty:** Intermediate

## Learning Objectives

By the end of this chapter, you will:
- ✅ Understand Go's runtime architecture and components
- ✅ Know how Go manages memory allocation and layout
- ✅ Understand the goroutine scheduler basics
- ✅ Distinguish between stack and heap allocation
- ✅ Analyze memory allocation patterns in your code
- ✅ Optimize memory usage based on Go's memory model

## Theory: Go Runtime Architecture

### Go Runtime Components

The Go runtime is responsible for:

1. **Memory Management**
   - Garbage collection
   - Memory allocation (stack/heap)
   - Memory layout optimization

2. **Goroutine Scheduling**
   - M:N threading model
   - Work-stealing scheduler
   - Preemptive scheduling

3. **Channel Operations**
   - Communication between goroutines
   - Synchronization primitives

4. **System Calls**
   - Network I/O
   - File operations
   - OS interaction

### Memory Layout in Go

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Memory Layout                         │
├─────────────────────────────────────────────────────────────┤
│  Stack (per goroutine)                                      │
│  ├─ Local variables                                         │
│  ├─ Function parameters                                     │
│  ├─ Return addresses                                        │
│  └─ Small objects (escape analysis)                         │
├─────────────────────────────────────────────────────────────┤
│  Heap (shared)                                              │
│  ├─ Large objects                                           │
│  ├─ Objects that escape                                     │
│  ├─ Slices, maps, channels                                  │
│  └─ Interface values                                        │
├─────────────────────────────────────────────────────────────┤
│  Data Segment                                               │
│  ├─ Global variables                                        │
│  ├─ String literals                                         │
│  └─ Constants                                               │
└─────────────────────────────────────────────────────────────┘
```

### Stack vs Heap Allocation

**Stack Allocation (Fast)**
- Local variables with known size
- Function parameters
- Return values
- Small objects that don't escape
- Automatic cleanup when function returns

**Heap Allocation (Slower)**
- Large objects
- Objects that escape function scope
- Dynamic data structures (slices, maps)
- Objects referenced by pointers
- Requires garbage collection

### Escape Analysis

Go's compiler performs escape analysis to determine whether a variable should be allocated on the stack or heap:

```go
// Stack allocation - variable doesn't escape
func stackAllocation() int {
    x := 42  // Allocated on stack
    return x // Value copied, x doesn't escape
}

// Heap allocation - variable escapes
func heapAllocation() *int {
    x := 42   // Allocated on heap
    return &x // Pointer returned, x escapes
}

// Heap allocation - large size
func largeAllocation() [1000000]int {
    var arr [1000000]int // Too large for stack
    return arr           // Allocated on heap
}
```

### Goroutine Scheduler (GMP Model)

```
G = Goroutine (user-level thread)
M = Machine (OS thread)
P = Processor (logical CPU)

┌─────────────────────────────────────────────────────────────┐
│                    GMP Scheduler Model                      │
├─────────────────────────────────────────────────────────────┤
│   P0          P1           P2          P3                   │
│ ┌───┐       ┌───┐        ┌───┐       ┌───┐                  │
│ │ G │       │ G │        │ G │       │ G │                  │
│ ├───┤       ├───┤        ├───┤       ├───┤                  │
│ │ G │       │ G │        │ G │       │ G │                  │
│ ├───┤       ├───┤        ├───┤       ├───┤                  │
│ │ G │       │ G │        │ G │       │ G │                  │
│ └───┘       └───┘        └───┘       └───┘                  │
│   │           │            │           │                    │
│   M0          M1           M2          M3                   │
│                                                             │
│ Global Run Queue: [G] [G] [G] [G] [G]                       │
└─────────────────────────────────────────────────────────────┘
```

**Key Concepts:**
- **P (Processor):** Logical processor, usually equals GOMAXPROCS
- **M (Machine):** OS thread that executes goroutines
- **G (Goroutine):** User-level thread
- **Work Stealing:** Idle P steals work from other P's queues

### Memory Allocator

Go uses a sophisticated memory allocator based on tcmalloc:

1. **Size Classes:** Objects grouped by size for efficient allocation
2. **Thread Caches:** Per-P caches to reduce contention
3. **Central Lists:** Shared pools for different size classes
4. **Page Heap:** Large object allocation

## Video Resources

📺 **Watch Before Exercises:**
- ["Understanding Allocations: the Stack and the Heap" by Jacob Walker](https://www.youtube.com/watch?v=ZMZpH4yT7M0)
- ["Go Execution Tracer" by Rhys Hiltner](https://www.youtube.com/watch?v=mmqDlbWk_XA)

## Exercises

### Exercise 02-01: Escape Analysis Investigation

**Objective:** Understand when variables escape to the heap.

**Tasks:**
1. Write functions that demonstrate stack vs heap allocation
2. Use escape analysis to verify allocation behavior
3. Measure performance differences

### Exercise 02-02: Memory Allocation Patterns

**Objective:** Analyze different memory allocation patterns and their performance.

**Tasks:**
1. Compare slice allocation strategies
2. Analyze map allocation behavior
3. Understand interface allocation costs

### Exercise 02-03: Goroutine Scheduler Analysis

**Objective:** Understand how the Go scheduler works.

**Tasks:**
1. Create goroutines with different workloads
2. Analyze scheduler behavior with GODEBUG
3. Measure context switching overhead

### Exercise 02-04: Memory Layout Visualization

**Objective:** Visualize memory usage patterns.

**Tasks:**
1. Create programs with different allocation patterns
2. Use runtime.MemStats to analyze memory usage
3. Understand GC impact on performance

## Key Concepts Summary

### Memory Management
- **Stack:** Fast, automatic cleanup, limited size
- **Heap:** Flexible, requires GC, slower allocation
- **Escape Analysis:** Compiler optimization for allocation placement

### Scheduler
- **M:N Model:** Many goroutines on fewer OS threads
- **Work Stealing:** Load balancing between processors
- **Preemptive:** Prevents goroutine monopolization

### Performance Implications
- Prefer stack allocation when possible
- Minimize heap allocations in hot paths
- Understand scheduler behavior for concurrent code
- Use appropriate data structures for your use case

## Tools for Analysis

```bash
# Escape analysis
go build -gcflags="-m" main.go

# Detailed escape analysis
go build -gcflags="-m -m" main.go

# Scheduler tracing
GODEBUG=schedtrace=1000 go run main.go

# Memory statistics
GODEBUG=gctrace=1 go run main.go

# Execution tracer
go run main.go 2> trace.out
go tool trace trace.out
```

## Common Optimization Patterns

1. **Reduce Allocations**
   ```go
   // Bad: Multiple allocations
   result := make([]string, 0)
   for _, item := range items {
       result = append(result, process(item))
   }
   
   // Good: Pre-allocate
   result := make([]string, 0, len(items))
   for _, item := range items {
       result = append(result, process(item))
   }
   ```

2. **Reuse Objects**
   ```go
   // Use sync.Pool for expensive objects
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return make([]byte, 1024)
       },
   }
   ```

3. **Avoid Interface Allocations**
   ```go
   // Bad: Interface allocation
   func process(data interface{}) {}
   
   // Good: Type-specific function
   func processString(data string) {}
   ```

## Next Steps

After completing this chapter, proceed to [Chapter 03: Basic Profiling with pprof](../03-basic-profiling/) to learn how to measure and analyze the concepts covered here.

---

**Checkpoint:** Can you explain the difference between stack and heap allocation, and describe Go's scheduler model? If yes, you're ready for the exercises!