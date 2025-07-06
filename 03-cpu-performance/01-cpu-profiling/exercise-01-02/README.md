# Exercise 01-02: Flame Graph Analysis

**Difficulty:** Intermediate  
**Estimated Time:** 60-75 minutes  
**Prerequisites:** Exercise 01-01 completion, basic understanding of CPU profiling

## Objective

Learn to generate, interpret, and analyze flame graphs for CPU performance analysis. You'll work with complex call stacks, understand flame graph visualization, and use flame graphs to identify performance bottlenecks in realistic scenarios.

## Learning Goals

- Generate flame graphs from CPU profiles
- Interpret flame graph visualizations
- Analyze call stacks and hot paths
- Identify performance bottlenecks using flame graphs
- Compare flame graphs before and after optimization
- Understand different flame graph types and use cases

## Background

Flame graphs are a powerful visualization technique for understanding CPU performance. They show the call stack hierarchy and time spent in each function, making it easy to identify performance bottlenecks. This exercise focuses on practical flame graph analysis techniques.

## Tasks

### Task 1: Create a Complex Application

Build an application with nested function calls and realistic performance patterns.

**Requirements:**
1. Implement a multi-layered application architecture
2. Create functions with different performance characteristics
3. Include recursive and iterative algorithms
4. Add realistic data processing pipelines

### Task 2: Generate Flame Graphs

Create flame graphs using different tools and techniques.

**Requirements:**
1. Generate flame graphs using `go tool pprof`
2. Create SVG flame graphs for web viewing
3. Generate interactive flame graphs
4. Export flame graphs in different formats

### Task 3: Flame Graph Analysis

Analyze flame graphs to identify performance issues.

**Requirements:**
1. Identify the widest flames (most CPU time)
2. Analyze call stack depth and complexity
3. Find unexpected function calls
4. Identify optimization opportunities

### Task 4: Optimization and Comparison

Optimize code based on flame graph analysis and compare results.

**Requirements:**
1. Implement optimizations based on flame graph insights
2. Generate "after" flame graphs
3. Compare before/after flame graphs
4. Measure and document performance improvements

## Implementation Guide

### Step 1: Application Structure

Create a realistic application with multiple layers:

```
exercise-01-02/
├── main.go
├── server/
│   ├── handler.go
│   └── middleware.go
├── service/
│   ├── processor.go
│   └── analyzer.go
├── data/
│   ├── generator.go
│   └── transformer.go
├── algorithms/
│   ├── sorting.go
│   ├── searching.go
│   └── graph.go
├── utils/
│   └── helpers.go
├── go.mod
└── README.md
```

### Step 2: Multi-Layer Architecture

```go
// main.go
package main

import (
    "context"
    "fmt"
    "time"
    
    "./server"
    "./service"
)

func main() {
    // Setup profiling
    cleanup := setupProfiling()
    defer cleanup()
    
    // Run realistic workload
    runRealisticWorkload()
}

func runRealisticWorkload() {
    // Simulate web server handling requests
    server := server.New()
    processor := service.NewProcessor()
    
    // Process multiple requests concurrently
    for i := 0; i < 100; i++ {
        go handleRequest(server, processor, i)
    }
}
```

### Step 3: Complex Processing Pipeline

```go
// service/processor.go
package service

type Processor struct {
    analyzer *Analyzer
    cache    map[string]interface{}
}

func (p *Processor) ProcessData(data []byte) Result {
    // Multi-stage processing pipeline
    parsed := p.parseData(data)
    validated := p.validateData(parsed)
    transformed := p.transformData(validated)
    analyzed := p.analyzer.Analyze(transformed)
    
    return p.generateResult(analyzed)
}

func (p *Processor) parseData(data []byte) ParsedData {
    // Complex parsing logic with nested calls
}

func (p *Processor) validateData(data ParsedData) ValidatedData {
    // Validation with multiple checks
}
```

### Step 4: Flame Graph Generation

```bash
# Generate CPU profile
go run . -cpuprofile=cpu.prof

# Generate flame graph using pprof
go tool pprof -http=:8080 cpu.prof

# Generate SVG flame graph
go tool pprof -svg cpu.prof > flamegraph.svg

# Generate interactive flame graph
go tool pprof -web cpu.prof
```

## Expected Output

### Flame Graph Characteristics

1. **Wide Flames:** Functions consuming most CPU time
2. **Tall Stacks:** Deep call hierarchies
3. **Flat Tops:** Functions doing actual work
4. **Narrow Spikes:** Infrequently called expensive functions

### Sample Analysis Results

```
Flame Graph Analysis:
├── main.runRealisticWorkload (100% - 5.2s)
│   ├── service.ProcessData (85% - 4.42s)
│   │   ├── service.parseData (35% - 1.82s)
│   │   │   └── json.Unmarshal (30% - 1.56s)
│   │   ├── service.transformData (25% - 1.30s)
│   │   │   └── algorithms.QuickSort (20% - 1.04s)
│   │   └── service.validateData (25% - 1.30s)
│   │       └── regexp.Match (22% - 1.14s)
│   └── server.HandleRequest (15% - 0.78s)
│       └── middleware.LogRequest (5% - 0.26s)
```

## Verification Steps

1. **Flame Graph Generation:**
   - [ ] Successfully generates flame graphs
   - [ ] Flame graphs show realistic call stacks
   - [ ] Multiple output formats work

2. **Analysis Accuracy:**
   - [ ] Can identify top CPU consumers
   - [ ] Can trace call paths
   - [ ] Can spot optimization opportunities

3. **Optimization Impact:**
   - [ ] Optimizations show measurable improvement
   - [ ] Before/after comparison is clear
   - [ ] Performance gains are documented

## Common Flame Graph Patterns

### 1. CPU-Bound Operations
```
████████████████████████████████ calculatePrimes (45%)
```
**Interpretation:** Wide flame indicates high CPU usage
**Action:** Optimize algorithm or parallelize

### 2. Deep Call Stacks
```
█ main
█ processRequest
█ validateInput
█ parseJSON
█ unmarshalField
█ convertType
█ parseNumber (2%)
```
**Interpretation:** Deep stack with narrow flame
**Action:** Consider flattening call hierarchy

### 3. Memory Allocation Hotspots
```
██████████████████ runtime.mallocgc (25%)
```
**Interpretation:** High memory allocation overhead
**Action:** Reduce allocations, use object pools

### 4. Lock Contention
```
████████ runtime.lock (12%)
```
**Interpretation:** Synchronization overhead
**Action:** Reduce lock scope, use lock-free algorithms

## Advanced Analysis Techniques

### 1. Differential Flame Graphs
```bash
# Compare two profiles
go tool pprof -base=before.prof -svg after.prof > diff.svg
```

### 2. Filtered Flame Graphs
```bash
# Focus on specific functions
go tool pprof -focus=calculatePrimes cpu.prof

# Ignore runtime functions
go tool pprof -ignore=runtime cpu.prof
```

### 3. Time-Based Analysis
```bash
# Sample at different intervals
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

## Optimization Strategies

### Based on Flame Graph Patterns:

1. **Wide Flames (High CPU):**
   - Algorithm optimization
   - Parallelization
   - Caching

2. **Tall Stacks (Deep Calls):**
   - Function inlining
   - Call hierarchy flattening
   - Tail call optimization

3. **Memory Patterns:**
   - Object pooling
   - Reduced allocations
   - Better data structures

4. **I/O Patterns:**
   - Batching
   - Async processing
   - Connection pooling

## Extension Challenges

1. **Custom Flame Graph Tools:**
   - Build a custom flame graph generator
   - Add custom annotations
   - Integrate with monitoring systems

2. **Real-Time Analysis:**
   - Continuous flame graph generation
   - Automated anomaly detection
   - Performance regression alerts

3. **Multi-Dimensional Analysis:**
   - Combine CPU and memory flame graphs
   - Add latency information
   - Include business metrics

4. **Production Integration:**
   - Safe production profiling
   - Sampling strategies
   - Performance budgets

## Key Learning Points

1. **Flame Graph Reading:** Width = time, height = call depth
2. **Pattern Recognition:** Common performance anti-patterns
3. **Optimization Prioritization:** Focus on widest flames first
4. **Tool Proficiency:** Multiple ways to generate and view flame graphs
5. **Iterative Process:** Profile → Analyze → Optimize → Repeat

## Troubleshooting

### Issue: Empty or Minimal Flame Graphs
**Solution:** Ensure sufficient CPU load and profiling duration

### Issue: Flame Graph Too Complex
**Solution:** Use filtering options to focus on specific areas

### Issue: Can't Identify Bottlenecks
**Solution:** Look for widest flames and unexpected call patterns

### Issue: Optimization Not Visible
**Solution:** Ensure changes affect the hot path shown in flame graph

## Next Steps

After completing this exercise:
1. Practice with real applications
2. Experiment with different flame graph tools
3. Learn about other profiling visualizations
4. Move to [Exercise 01-03: Production Profiling](../exercise-01-03/README.md)

## Resources

- [Flame Graphs by Brendan Gregg](http://www.brendangregg.com/flamegraphs.html)
- [Go pprof Flame Graphs](https://golang.org/pkg/net/http/pprof/)
- [Interactive Flame Graph Tools](https://github.com/spiermar/d3-flame-graph)
- [Flame Graph Interpretation Guide](http://www.brendangregg.com/FlameGraphs/cpuflamegraphs.html)

---

*Exercise 01-02 of Chapter 01: CPU Profiling Techniques*