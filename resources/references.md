# Go Performance References

A comprehensive collection of resources for Go performance optimization and debugging.

## Official Documentation

### Go Team Resources
- [Go Diagnostics](https://golang.org/doc/diagnostics.html) - Official debugging guide
- [Effective Go](https://golang.org/doc/effective_go.html) - Best practices
- [Go Memory Model](https://golang.org/ref/mem) - Memory semantics
- [Go Execution Tracer](https://golang.org/doc/trace.html) - Execution tracing
- [Go pprof](https://golang.org/pkg/net/http/pprof/) - Profiling package
- [Go Race Detector](https://golang.org/doc/articles/race_detector.html) - Concurrency debugging

### Language Specifications
- [Go Language Specification](https://golang.org/ref/spec) - Complete language reference
- [Go Assembly](https://go.dev/doc/asm) - Assembly programming guide
- [Go Compiler Directives](https://pkg.go.dev/cmd/compile) - Compiler optimization hints

## Performance Books

### Essential Reading
- **"High Performance Go"** by Dave Cheney - [Workshop Materials](https://dave.cheney.net/high-performance-go-workshop/)
- **"Systems Performance"** by Brendan Gregg - Linux performance bible
- **"Concurrency in Go"** by Katherine Cox-Buday - Goroutines and channels
- **"The Go Programming Language"** by Donovan & Kernighan - Comprehensive Go guide

### Specialized Topics
- **"Linux Performance and Tuning Guidelines"** by IBM Redbooks
- **"BPF Performance Tools"** by Brendan Gregg
- **"Database Internals"** by Alex Petrov - Database performance
- **"Designing Data-Intensive Applications"** by Martin Kleppmann

## Video Resources

### Must-Watch Talks

#### Go Performance Fundamentals
- [High Performance Go - Dave Cheney (GopherCon 2019)](https://www.youtube.com/watch?v=2557w0qsDV0)
- [Profiling Go Programs - Brad Fitzpatrick (Google I/O)](https://www.youtube.com/watch?v=xxDZuPEgbBU)
- [Go Performance Tales - Dmitry Vyukov (GopherCon 2014)](https://www.youtube.com/watch?v=2h_NFBFrciI)

#### Memory Management
- [Understanding Go's Memory Allocator - Andrei Tudor Călin](https://www.youtube.com/watch?v=ZMZpH4yT7M0)
- [Garbage Collection in Go - Rick Hudson (GopherCon 2015)](https://www.youtube.com/watch?v=aiv1JOfMjm0)
- [Go Memory Management - Bill Kennedy](https://www.youtube.com/watch?v=gPzTWk_3xKA)

#### Concurrency & Scheduling
- [Go Execution Tracer - Dmitry Vyukov (GopherCon 2017)](https://www.youtube.com/watch?v=mmqDlbWk_XA)
- [Concurrency Patterns in Go - Arne Claus](https://www.youtube.com/watch?v=rDRa23k70CU)
- [Advanced Go Concurrency - Sameer Ajmani](https://www.youtube.com/watch?v=QDDwwePbDtw)

#### Optimization Techniques
- [Optimizing Go Code without a Blindfold - Daniel Lemire](https://www.youtube.com/watch?v=9Ac1Jn1b2X4)
- [Go Performance Optimization - Filippo Valsorda](https://www.youtube.com/watch?v=NS1hmEWv4Ac)
- [Writing High Performance Go - Erik Dubbelboer](https://www.youtube.com/watch?v=zWp0N9unJFc)

#### Linux Performance
- [Linux Performance Tools - Brendan Gregg (Netflix)](https://www.youtube.com/watch?v=FJW8nGV4jxY)
- [Systems Performance - Brendan Gregg (USENIX LISA)](https://www.youtube.com/watch?v=fhBHvsi0Mk0)
- [BPF Performance Tools - Brendan Gregg](https://www.youtube.com/watch?v=bj3qdEDbCD4)

## GitHub Resources

### Performance Guides
- [Go Performance Book - Damian Gryski](https://github.com/dgryski/go-perfbook)
- [Awesome Go Performance](https://github.com/cristaloleg/awesome-go-performance)
- [Go Optimization Guide](https://github.com/astavonin/go-optimization-guide)
- [High Performance Go Workshop](https://github.com/davecheney/high-performance-go-workshop)

### Tools and Libraries
- [FlameGraph - Brendan Gregg](https://github.com/brendangregg/FlameGraph)
- [Go Torch - Uber](https://github.com/uber-archive/go-torch)
- [Pprof++](https://github.com/google/pprof)
- [Go Callvis](https://github.com/ofabry/go-callvis)

### Benchmarking Suites
- [Go Benchmarks Game](https://github.com/golang/benchmarks)
- [Go Performance Tests](https://github.com/golang/go/tree/master/test/bench)
- [Micro Benchmarks](https://github.com/golang/go/tree/master/src/testing)

## Research Papers

### Go Runtime
- ["The Go Programming Language and Environment"](https://golang.org/doc/go_lang_faq.html)
- ["Go's Work-Stealing Scheduler"](https://docs.google.com/document/d/1TTj4T2JO42uD5ID9e89oa0sLKhJYD0Y_kqxDv3I3XMw)
- ["Scalable Go Scheduler Design"](https://golang.org/s/go11sched)

### Garbage Collection
- ["Getting to Go: The Journey of Go's Garbage Collector"](https://blog.golang.org/ismmkeynote)
- ["Go GC: Prioritizing low latency and simplicity"](https://golang.org/doc/gc-guide)
- ["Concurrent Garbage Collection in Go"](https://golang.org/s/go15gcpacing)

### Memory Management
- ["TCMalloc: Thread-Caching Malloc"](https://google.github.io/tcmalloc/design.html)
- ["Go Memory Allocator Design"](https://golang.org/s/go14malloc)

## Blogs and Articles

### Official Go Blog
- [Go Blog - Performance](https://blog.golang.org/)
- [Profiling Go Programs](https://blog.golang.org/pprof)
- [Go Execution Tracer](https://blog.golang.org/execution-tracer)

### Community Blogs
- [Dave Cheney's Blog](https://dave.cheney.net/) - Performance insights
- [Brendan Gregg's Blog](http://www.brendangregg.com/) - Systems performance
- [Ardan Labs Blog](https://www.ardanlabs.com/blog/) - Go education
- [Segment Engineering](https://segment.com/blog/engineering/) - Production Go

### Performance Case Studies
- [Dropbox: Optimizing Go Performance](https://dropbox.tech/infrastructure/optimizing-web-servers-for-high-throughput-and-low-latency)
- [Uber: Go Performance at Scale](https://eng.uber.com/go-geofence/)
- [Netflix: Performance Engineering](https://netflixtechblog.com/)
- [Google: Go at Google](https://talks.golang.org/2012/splash.article)

## Tools Documentation

### Profiling Tools
- [pprof User Guide](https://github.com/google/pprof/blob/master/doc/README.md)
- [Go tool trace](https://golang.org/cmd/trace/)
- [Go tool compile](https://golang.org/cmd/compile/)
- [Delve Debugger](https://github.com/go-delve/delve)

### Linux Performance Tools
- [perf Examples](http://www.brendangregg.com/perf.html)
- [BPF Tools](https://github.com/iovisor/bcc)
- [Linux Performance Analysis](http://www.brendangregg.com/linuxperf.html)

### Load Testing
- [wrk Documentation](https://github.com/wg/wrk)
- [hey Documentation](https://github.com/rakyll/hey)
- [k6 Documentation](https://k6.io/docs/)

## Online Courses

### Free Resources
- [Go by Example](https://gobyexample.com/)
- [A Tour of Go](https://tour.golang.org/)
- [Go Web Examples](https://gowebexamples.com/)
- [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests/)

### Paid Courses
- [Ultimate Go - Ardan Labs](https://www.ardanlabs.com/ultimate-go/)
- [Go: The Complete Developer's Guide - Udemy](https://www.udemy.com/course/go-the-complete-developers-guide/)
- [Mastering Go Programming - Packt](https://www.packtpub.com/product/mastering-go-programming/)

## Community Resources

### Forums and Discussion
- [r/golang](https://www.reddit.com/r/golang/) - Reddit community
- [Gopher Slack](https://gophers.slack.com/) - #performance channel
- [Go Forum](https://forum.golangbridge.org/) - Official forum
- [Stack Overflow - Go](https://stackoverflow.com/questions/tagged/go) - Q&A

### Conferences
- [GopherCon](https://www.gophercon.com/) - Annual Go conference
- [dotGo](https://www.dotgo.eu/) - European Go conference
- [GopherCon UK](https://www.gophercon.co.uk/) - UK Go conference
- [GoLab](https://golab.io/) - Italian Go conference

## Newsletters and Podcasts

### Newsletters
- [Golang Weekly](https://golangweekly.com/) - Weekly Go news
- [Go Newsletter](https://golang.ch/newsletter/) - Community newsletter

### Podcasts
- [Go Time](https://changelog.com/gotime) - Weekly Go podcast
- [Cup o' Go](https://cupogo.dev/) - Short Go discussions

## Performance Monitoring

### APM Solutions
- [Datadog Go Tracing](https://docs.datadoghq.com/tracing/setup_overview/setup/go/)
- [New Relic Go Agent](https://docs.newrelic.com/docs/agents/go-agent/)
- [AppDynamics Go](https://docs.appdynamics.com/display/PRO45/Go+Agent)

### Open Source Monitoring
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Jaeger Go Client](https://github.com/jaegertracing/jaeger-client-go)
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go)

## Benchmarking Resources

### Benchmark Suites
- [Computer Language Benchmarks Game](https://benchmarksgame-team.pages.debian.net/benchmarksgame/)
- [TechEmpower Framework Benchmarks](https://www.techempower.com/benchmarks/)
- [Go HTTP Router Benchmark](https://github.com/julienschmidt/go-http-routing-benchmark)

### Performance Databases
- [Go Performance Dashboard](https://perf.golang.org/)
- [Continuous Benchmarking](https://github.com/golang/benchmarks)

## Contributing

To add new resources:
1. Verify the resource quality and relevance
2. Add appropriate categorization
3. Include brief descriptions
4. Maintain alphabetical ordering within sections

---

*Last updated: 2024*
*Maintained by: Go Performance Mastery Course*