# Module 01: Performance Fundamentals

**Duration:** 2-3 days  
**Focus:** Understanding performance basics, measurement, and profiling foundations

## Module Overview

This foundational module establishes the performance mindset and introduces essential tools for measuring and analyzing Go application performance. You'll learn the fundamentals of Go's runtime, memory model, and profiling techniques that form the basis for all subsequent optimization work.

## Learning Objectives

By the end of this module, you will:
- ✅ Understand performance optimization methodology
- ✅ Know Go's runtime architecture and memory model
- ✅ Master basic profiling with pprof
- ✅ Write effective benchmarks and interpret results
- ✅ Set up essential performance tools

## Prerequisites

- Go 1.24+ installed
- Basic Go programming knowledge
- Linux environment (Ubuntu/Debian preferred)
- Text editor or IDE

## Module Structure

### [Chapter 01: Performance Mindset and Methodology](./01-performance-mindset/)
**Duration:** 4-6 hours
- Performance optimization principles
- Measurement-driven development
- Common performance anti-patterns
- Setting up development environment

### [Chapter 02: Go Runtime and Memory Model](./02-go-runtime-memory/)
**Duration:** 6-8 hours
- Go runtime architecture
- Memory layout and allocation
- Goroutine scheduler basics
- Stack vs heap allocation

### [Chapter 03: Basic Profiling with pprof](./03-basic-profiling/)
**Duration:** 6-8 hours
- Introduction to pprof
- CPU profiling fundamentals
- Memory profiling basics
- Reading and interpreting profiles

### [Chapter 04: Benchmarking Best Practices](./04-benchmarking-best-practices/)
**Duration:** 4-6 hours
- Writing effective benchmarks
- Benchmark analysis and interpretation
- Avoiding common benchmarking pitfalls
- Continuous performance monitoring

## Tools Setup

Before starting this module, ensure you have the following tools installed:

```bash
# Go tools (included with Go installation)
go version  # Should be 1.24+

# Additional profiling tools
go install github.com/google/pprof@latest

# Linux performance tools
sudo apt update
sudo apt install -y htop iotop sysstat

# Optional: Graphviz for pprof visualization
sudo apt install -y graphviz
```

## Module Exercises

Each chapter contains hands-on exercises. Complete them in order:

1. **Exercise 01-01:** Environment setup and first benchmark
2. **Exercise 01-02:** Memory allocation analysis
3. **Exercise 01-03:** CPU profiling a simple program
4. **Exercise 01-04:** Benchmark comparison and optimization

## Key Concepts Covered

- **Performance Mindset:** Measure first, optimize second
- **Go Runtime:** Understanding the foundation
- **Profiling:** Making performance visible
- **Benchmarking:** Quantifying improvements

## Video Resources for This Module

- ["High Performance Go" by Dave Cheney](https://www.youtube.com/watch?v=2557w0qsDV0) - Watch after Chapter 01
- ["Profiling Go Programs" by Brad Fitzpatrick](https://www.youtube.com/watch?v=xxDZuPEgbBU) - Watch after Chapter 03
- ["Understanding Go's Memory Allocator" by Andrei Tudor Călin](https://www.youtube.com/watch?v=ZMZpH4yT7M0) - Watch after Chapter 02

## Success Criteria

You've successfully completed this module when you can:
- [ ] Set up and run basic performance profiling
- [ ] Write and interpret Go benchmarks
- [ ] Explain Go's memory allocation basics
- [ ] Identify performance bottlenecks using pprof
- [ ] Apply performance optimization methodology

## Next Steps

After completing this module, proceed to [Module 02: Memory Optimization](../02-memory-optimization/) to dive deeper into memory-specific performance techniques.

---

**Ready to start?** Begin with [Chapter 01: Performance Mindset and Methodology](./01-performance-mindset/README.md)