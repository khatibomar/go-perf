# Go Performance Mastery Course

A comprehensive, hands-on course designed for backend developers to master Go performance optimization and Linux debugging techniques.

## Course Overview

This course is structured as a progressive learning journey, taking you from fundamental performance concepts to advanced optimization techniques. Each module builds upon the previous one, providing practical examples and real-world scenarios.

## Prerequisites

- Go 1.24+ installed
- Linux environment (Ubuntu/Debian preferred)
- Basic Go programming knowledge
- Command line familiarity

## Course Structure

### Module 01: Performance Fundamentals
**Duration:** 2-3 days
**Focus:** Understanding performance basics, measurement, and profiling foundations

- **Chapter 01:** Performance Mindset and Methodology
- **Chapter 02:** Go Runtime and Memory Model
- **Chapter 03:** Basic Profiling with pprof
- **Chapter 04:** Benchmarking Best Practices

### Module 02: Memory Optimization
**Duration:** 3-4 days
**Focus:** Memory allocation, garbage collection, and memory-efficient programming

- **Chapter 01:** Understanding Go Memory Allocator
- **Chapter 02:** Garbage Collection Deep Dive
- **Chapter 03:** Memory Profiling and Analysis
- **Chapter 04:** Memory Optimization Patterns

### Module 03: CPU Performance
**Duration:** 3-4 days
**Focus:** CPU profiling, algorithmic optimization, and concurrency patterns

- **Chapter 01:** CPU Profiling Techniques
- **Chapter 02:** Algorithmic Optimization
- **Chapter 03:** Goroutines and Scheduler Optimization
- **Chapter 04:** Lock-Free Programming Patterns

### Module 04: I/O and Network Performance
**Duration:** 3-4 days
**Focus:** Network optimization, database performance, and I/O patterns

- **Chapter 01:** Network Programming Optimization
- **Chapter 02:** Database Connection Pooling
- **Chapter 03:** File I/O and Disk Performance
- **Chapter 04:** Caching Strategies

### Module 05: Advanced Debugging and Production
**Duration:** 4-5 days
**Focus:** Production debugging, monitoring, and advanced tooling

- **Chapter 01:** Linux Performance Tools for Go
- **Chapter 02:** Distributed Tracing and Observability
- **Chapter 03:** Production Debugging Techniques
- **Chapter 04:** Performance Testing and Load Testing

## Learning Methodology

Each chapter follows this structure:
1. **Theory** - Core concepts and principles
2. **Examples** - Practical code demonstrations
3. **Exercises** - Hands-on practice problems
4. **Video Resources** - Expert talks and demonstrations
5. **References** - Additional learning resources

### Study Tips
- 📚 Read theory first, then watch related videos
- 💻 Code along with examples in your own environment
- 🔬 Complete all exercises before moving to next chapter
- 📊 Measure performance before and after optimizations
- 🎯 Focus on understanding "why" not just "how"

## Tools You'll Master

- **Go Tools:** pprof, trace, benchmark, race detector, go tool compile
- **Linux Tools:** perf, strace, tcpdump, iotop, htop, vmstat, iostat
- **Monitoring:** Prometheus, Grafana, Jaeger, OpenTelemetry
- **Load Testing:** wrk, hey, Apache Bench, k6
- **Profiling:** FlameGraph, pprof web UI, go-torch
- **Memory Analysis:** valgrind, AddressSanitizer, go tool trace

## Getting Started

1. Clone this repository
2. Ensure Go 1.24+ is installed
3. Install required Linux tools (see Module 01)
4. Start with Module 01, Chapter 01

## Course Navigation

```
go-perf/
├── README.md                 # This file
├── 01-performance-fundamentals/
│   ├── README.md
│   ├── 01-performance-mindset/
│   ├── 02-go-runtime-memory/
│   ├── 03-basic-profiling/
│   └── 04-benchmarking-best-practices/
├── 02-memory-optimization/
│   ├── README.md
│   ├── 01-memory-allocator/
│   ├── 02-garbage-collection/
│   ├── 03-memory-profiling/
│   └── 04-optimization-patterns/
├── 03-cpu-performance/
│   ├── README.md
│   ├── 01-cpu-profiling/
│   ├── 02-algorithmic-optimization/
│   ├── 03-goroutines-scheduler/
│   └── 04-lock-free-patterns/
├── 04-io-network-performance/
│   ├── README.md
│   ├── 01-network-optimization/
│   ├── 02-database-performance/
│   ├── 03-file-io-disk/
│   └── 04-caching-strategies/
├── 05-advanced-debugging/
│   ├── README.md
│   ├── 01-linux-tools/
│   ├── 02-distributed-tracing/
│   ├── 03-production-debugging/
│   └── 04-performance-testing/
└── resources/
    ├── tools-setup.md
    ├── references.md
    └── cheatsheets/
```

## Key Learning Outcomes

By the end of this course, you will:

- ✅ Master Go's performance profiling tools
- ✅ Understand memory management and GC optimization
- ✅ Optimize CPU-intensive applications
- ✅ Debug performance issues in production
- ✅ Use Linux tools for Go application debugging
- ✅ Implement high-performance concurrent patterns
- ✅ Design scalable backend architectures

## Expert Video Resources 🎥

### Must-Watch Go Performance Talks
- **["High Performance Go" by Dave Cheney](https://www.youtube.com/watch?v=2557w0qsDV0)** - GopherCon 2019
- **["Profiling Go Programs" by Brad Fitzpatrick](https://www.youtube.com/watch?v=xxDZuPEgbBU)** - Google I/O 2011
- **["Go Performance Tales" by Dmitry Vyukov](https://www.youtube.com/watch?v=2h_NFBFrciI)** - GopherCon 2014
- **["Optimizing Go Code without a Blindfold" by Daniel Lemire](https://www.youtube.com/watch?v=9Ac1Jn1b2X4)** - GopherCon 2018
- **["Understanding Go's Memory Allocator" by Andrei Tudor Călin](https://www.youtube.com/watch?v=ZMZpH4yT7M0)** - GopherCon 2018
- **["Garbage Collection in Go" by Rick Hudson](https://www.youtube.com/watch?v=aiv1JOfMjm0)** - GopherCon 2015
- **["Go Execution Tracer" by Dmitry Vyukov](https://www.youtube.com/watch?v=mmqDlbWk_XA)** - GopherCon 2017
- **["Concurrency Patterns in Go" by Arne Claus](https://www.youtube.com/watch?v=rDRa23k70CU)** - GopherCon 2018

### Linux Performance & Debugging
- **["Linux Performance Tools" by Brendan Gregg](https://www.youtube.com/watch?v=FJW8nGV4jxY)** - Netflix Tech Talk
- **["Systems Performance" by Brendan Gregg](https://www.youtube.com/watch?v=fhBHvsi0Mk0)** - USENIX LISA
- **["BPF Performance Tools" by Brendan Gregg](https://www.youtube.com/watch?v=bj3qdEDbCD4)** - Linux Foundation

## References and Further Reading 📚

### Official Documentation
- [Go Optimization Guide](https://github.com/astavonin/go-optimization-guide) - Comprehensive optimization patterns
- [Effective Go](https://golang.org/doc/effective_go.html) - Official Go best practices
- [Go Memory Model](https://golang.org/ref/mem) - Understanding Go's memory semantics
- [Go Diagnostics](https://golang.org/doc/diagnostics.html) - Official debugging guide

### Performance Resources
- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html) - Dave Cheney's workshop
- [Linux Performance Tools](http://www.brendangregg.com/linuxperf.html) - Brendan Gregg's performance tools
- [Go Performance Tips](https://github.com/dgryski/go-perfbook) - Damian Gryski's performance book
- [Awesome Go Performance](https://github.com/cristaloleg/awesome-go-performance) - Curated performance resources

### Go GitHub Issues & Proposals (Deep Learning)
- **[Proposal: Go 2 Generics](https://github.com/golang/go/issues/43651)** - Performance implications of generics
- **[Runtime: Scalable Go Scheduler](https://github.com/golang/go/issues/51347)** - Scheduler improvements and GOMAXPROCS
- **[Proposal: Arena Allocator](https://github.com/golang/go/issues/51317)** - Manual memory management for performance
- **[Runtime: GC Pacer Improvements](https://github.com/golang/go/issues/44167)** - Garbage collector tuning insights
- **[Proposal: Profile-Guided Optimization](https://github.com/golang/go/issues/55022)** - PGO for better performance
- **[Runtime: Stack Scanning Optimization](https://github.com/golang/go/issues/22350)** - Stack management performance
- **[Proposal: Structured Logging](https://github.com/golang/go/issues/56345)** - slog performance considerations
- **[Runtime: Memory Allocator Improvements](https://github.com/golang/go/issues/35112)** - tcmalloc-style improvements
- **[Proposal: Faster JSON](https://github.com/golang/go/issues/5683)** - JSON encoding/decoding optimizations
- **[Runtime: Better CPU Profiling](https://github.com/golang/go/issues/35057)** - Profiling accuracy improvements

### Go Runtime Internals
- [Go Runtime Source Code](https://github.com/golang/go/tree/master/src/runtime) - Official runtime implementation
- [Go Compiler Optimizations](https://github.com/golang/go/tree/master/src/cmd/compile) - Understanding compiler behavior
- [Go Assembly Guide](https://go.dev/doc/asm) - Low-level optimization techniques
- [Go Build Cache](https://github.com/golang/go/issues/26809) - Build performance insights

### Books
- "Systems Performance" by Brendan Gregg
- "The Go Programming Language" by Alan Donovan & Brian Kernighan
- "Concurrency in Go" by Katherine Cox-Buday
- "Linux Performance and Tuning Guidelines" by IBM Redbooks

## Course Features ✨

- 🎯 **Hands-on Learning** - Every concept backed by practical examples
- 📊 **Real Metrics** - Learn to measure and validate optimizations
- 🔧 **Production Ready** - Techniques used in real-world applications
- 🎥 **Video Integration** - Expert talks complement each module
- 🐧 **Linux Focus** - Deep integration with Linux performance tools
- 📈 **Progressive Difficulty** - From basics to advanced optimization
- 🧪 **Lab Exercises** - Structured practice problems
- 📋 **Cheat Sheets** - Quick reference guides for tools and commands

## Support and Community 🤝

- 🐛 Create issues for questions or improvements
- 💡 Share your optimization discoveries
- 🔄 Contribute additional examples and exercises
- 📢 Join Go performance discussions on Reddit r/golang
- 💬 Participate in Gopher Slack #performance channel

---

**Ready to become a Go performance expert?** Start with [Module 01: Performance Fundamentals](./01-performance-fundamentals/README.md)
