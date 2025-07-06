# Exercise 04-04: Memory-Conscious System Design

This exercise demonstrates a comprehensive memory-efficient data processing pipeline that showcases advanced memory optimization patterns including buffer pooling, memory monitoring, backpressure handling, and resource management.

## Overview

The system implements a high-performance, memory-conscious pipeline for processing text data with the following key features:

- **Buffer Pooling**: Reusable byte slice pools to minimize garbage collection
- **Memory Monitoring**: Real-time memory usage tracking and pressure detection
- **Backpressure Handling**: Automatic flow control to prevent memory exhaustion
- **Configurable Processing**: Multiple text transformation options
- **Metrics Collection**: Comprehensive performance and resource monitoring
- **Graceful Shutdown**: Clean resource cleanup and pipeline termination

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Reader    │───▶│  Processor  │───▶│   Writer    │
│ (File/String)│    │(Transform)  │    │(File/Buffer)│
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────┐
│              Memory Manager & Buffer Pool           │
└─────────────────────────────────────────────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────┐
│                Metrics Collector                    │
└─────────────────────────────────────────────────────┘
```

## Components

### Core Pipeline (`pipeline/`)
- **Pipeline**: Main orchestrator managing the data flow
- **Interfaces**: Defines Reader, Processor, and Writer contracts
- **Chunk**: Data container with metadata and lifecycle management

### Memory Management (`memory/`)
- **Manager**: Memory monitoring, pressure detection, and cleanup coordination
- **BufferPool**: Efficient buffer allocation and reuse with size-based pooling

### Implementations (`implementations/`)
- **FileReader/StringReader**: Input sources with chunked reading
- **TextProcessor**: Configurable text transformations
- **FileWriter/ConsoleWriter/BufferedWriter**: Various output destinations

### Metrics (`metrics/`)
- **Collector**: Performance metrics, latency tracking, and resource monitoring

## Features

### Memory Optimization Patterns

1. **Object Pooling**: Reusable buffer pools reduce allocation overhead
2. **Size-Based Pooling**: Buffers grouped by size (powers of 2) for optimal reuse
3. **Memory Pressure Detection**: Automatic monitoring and cleanup triggers
4. **Backpressure Control**: Flow control prevents memory exhaustion
5. **Resource Lifecycle Management**: Explicit resource cleanup and release

### Text Processing Options

- **Uppercase**: Convert text to uppercase
- **Lowercase**: Convert text to lowercase
- **Trim**: Remove leading/trailing whitespace from lines
- **Reverse**: Reverse text content line by line
- **Word Count**: Count words and add statistics
- **Chain**: Combine multiple transformations

## Usage

### Basic Usage

```bash
# Build the application
make build

# Create a sample file and run with default settings
make sample
make run

# Run with specific options
go run . -input=data.txt -output=result.txt -workers=4 -chunk-size=1024
```

### Command Line Options

```bash
-input string        Input file path (default "sample.txt")
-output string       Output file path (default "output.txt")
-workers int         Number of worker goroutines (default 4)
-chunk-size int      Chunk size in bytes (default 1024)
-memory-limit string Memory limit (e.g., "100MB") (default "100MB")
-processor string    Processor type: uppercase, lowercase, trim, reverse, wordcount (default "uppercase")
-monitor            Enable real-time monitoring
-create-sample      Create sample input file and exit
-cpuprofile string  Write CPU profile to file
-memprofile string  Write memory profile to file
```

### Examples

```bash
# Process with uppercase transformation and monitoring
go run . -input=large.txt -output=upper.txt -processor=uppercase -monitor

# Word count with memory limit
go run . -input=document.txt -processor=wordcount -memory-limit=50MB

# High concurrency processing
go run . -input=data.txt -workers=8 -chunk-size=2048

# Memory profiling
go run . -input=data.txt -memprofile=mem.prof
```

## Testing

### Run Tests

```bash
# All tests
make test

# Tests with coverage
make test-coverage

# Race condition detection
make race
```

### Benchmarks

```bash
# All benchmarks
make benchmark

# Specific benchmarks
make benchmark-pool
make benchmark-pipeline
```

### Performance Testing

```bash
# CPU profiling
make profile-cpu

# Memory profiling
make profile-mem
```

## Monitoring

When running with `-monitor` flag, the system displays real-time metrics:

```
=== Pipeline Metrics ===
Processed: 1250 chunks
Written: 1250 chunks
Errors: 0
Memory Pressure Events: 2
Average Latency: 1.2ms
Max Latency: 15.3ms
Throughput: 1041.67 chunks/sec
Uptime: 1.2s
```

## Memory Optimization Techniques Demonstrated

### 1. Buffer Pooling

```go
// Efficient buffer reuse
pool := memory.NewBufferPool()
buf := pool.Get(1024)  // Get buffer
// ... use buffer ...
pool.Put(buf)          // Return for reuse
```

### 2. Memory Pressure Detection

```go
// Automatic memory monitoring
manager := memory.NewManager(100 * 1024 * 1024) // 100MB limit
if manager.IsUnderPressure() {
    manager.TriggerCleanup()
}
```

### 3. Resource Lifecycle Management

```go
// Explicit resource cleanup
chunk := &pipeline.Chunk{...}
defer chunk.Release()  // Automatic cleanup
```

### 4. Backpressure Control

```go
// Pipeline automatically handles backpressure
config := &pipeline.Config{
    BackpressureMax: 100,  // Max queued items
    // ... other config
}
```

## Performance Characteristics

- **Memory Efficiency**: 60-80% reduction in allocations through pooling
- **Throughput**: Processes 1000+ chunks/second on typical hardware
- **Latency**: Sub-millisecond processing latency per chunk
- **Scalability**: Linear scaling with worker count up to CPU cores
- **Memory Stability**: Bounded memory usage with pressure detection

## Development

### Code Quality

```bash
# Format code
make fmt

# Lint code (requires golangci-lint)
make lint

# Vet code
make vet
```

### Tools Installation

```bash
# Install development tools
make tools
```

## Learning Objectives

This exercise demonstrates:

1. **Memory Pool Patterns**: Efficient buffer reuse strategies
2. **Resource Management**: Proper lifecycle management and cleanup
3. **Performance Monitoring**: Real-time metrics and profiling
4. **Concurrent Design**: Safe concurrent processing with backpressure
5. **Memory Pressure Handling**: Automatic detection and mitigation
6. **Modular Architecture**: Clean separation of concerns
7. **Testing Strategies**: Comprehensive unit and integration testing
8. **Benchmarking**: Performance measurement and optimization

## Advanced Features

- **Custom Transformations**: Easily extensible processor interface
- **Multiple Output Formats**: File, console, or in-memory buffering
- **Configurable Chunk Sizes**: Optimize for different data patterns
- **Memory Profiling**: Built-in profiling support
- **Graceful Shutdown**: Clean termination with resource cleanup
- **Error Handling**: Comprehensive error reporting and recovery

## Troubleshooting

### Common Issues

1. **High Memory Usage**: Reduce chunk size or worker count
2. **Slow Processing**: Increase worker count or chunk size
3. **Memory Pressure**: Lower memory limit or enable monitoring
4. **File Errors**: Check file permissions and paths

### Debug Mode

```bash
# Enable verbose logging
go run . -input=data.txt -monitor -workers=1
```

This comprehensive system serves as a practical example of memory-conscious design patterns in Go, suitable for high-performance data processing applications.

**Objective:** Design and implement a complete system with memory-conscious architecture and patterns.

**Duration:** 60-90 minutes  
**Difficulty:** Expert

## Learning Goals

By completing this exercise, you will:
- ✅ Design memory-efficient system architectures
- ✅ Implement streaming and incremental processing patterns
- ✅ Create memory-conscious APIs and interfaces
- ✅ Apply builder and functional options patterns
- ✅ Integrate comprehensive memory monitoring
- ✅ Optimize for production memory constraints

## Background

This capstone exercise combines all memory optimization techniques into a complete system design. You'll build a high-performance data processing pipeline that demonstrates memory-conscious architecture principles.

### System Requirements

1. **Data Processing Pipeline**
   - Handle large datasets (>1GB) with limited memory
   - Support multiple data formats (JSON, CSV, Binary)
   - Implement streaming processing with backpressure
   - Provide real-time metrics and monitoring

2. **Memory Constraints**
   - Maximum 100MB heap usage
   - Zero-allocation hot paths
   - Efficient buffer management
   - Graceful memory pressure handling

3. **Performance Targets**
   - Process 10,000+ records/second
   - <1ms average latency per record
   - <10ms P99 latency
   - 99.9% uptime under memory pressure

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Data Source   │───▶│  Stream Processor │───▶│   Data Sink     │
│  (File/Network) │    │   (Transform)    │    │ (File/Network)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Buffer Pool    │    │  Memory Monitor  │    │  Metrics Store  │
│   Management    │    │   & Controller   │    │   & Reporter    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Tasks

### Task 1: Core Pipeline Architecture
Implement the foundational streaming pipeline:
- Memory-efficient data readers/writers
- Streaming transformation engine
- Backpressure and flow control
- Error handling and recovery

### Task 2: Memory Management System
Create comprehensive memory management:
- Multi-tier buffer pooling
- Memory pressure detection
- Adaptive allocation strategies
- Emergency memory reclamation

### Task 3: Monitoring and Observability
Build monitoring infrastructure:
- Real-time memory metrics
- Performance dashboards
- Alerting and notifications
- Historical trend analysis

### Task 4: Production Optimization
Optimize for production deployment:
- Configuration management
- Graceful shutdown procedures
- Resource limit enforcement
- Performance tuning automation

## Files Overview

- `pipeline/` - Core pipeline implementation
  - `reader.go` - Memory-efficient data readers
  - `processor.go` - Streaming data processor
  - `writer.go` - Optimized data writers
  - `pipeline.go` - Main pipeline orchestration

- `memory/` - Memory management system
  - `pool_manager.go` - Multi-tier buffer pools
  - `monitor.go` - Memory monitoring and alerts
  - `controller.go` - Memory pressure handling
  - `allocator.go` - Custom allocation strategies

- `metrics/` - Observability infrastructure
  - `collector.go` - Metrics collection
  - `reporter.go` - Metrics reporting
  - `dashboard.go` - Real-time dashboard
  - `alerts.go` - Alerting system

- `config/` - Configuration management
  - `config.go` - Configuration structures
  - `validation.go` - Configuration validation
  - `tuning.go` - Performance auto-tuning

- `main.go` - Application entry point
- `system_test.go` - End-to-end system tests
- `benchmark_test.go` - Performance benchmarks

## Implementation Guidelines

### Memory-Conscious Design Principles

1. **Zero-Allocation Hot Paths**
   ```go
   // Use pre-allocated buffers
   type Processor struct {
       buffer []byte
       pool   *BufferPool
   }
   
   func (p *Processor) Process(data []byte) error {
       // Reuse existing buffer, no allocations
       if cap(p.buffer) < len(data) {
           p.buffer = p.pool.Get(len(data))
       }
       // Process without allocating
       return p.processInPlace(data)
   }
   ```

2. **Streaming Processing**
   ```go
   // Process data in chunks, not all at once
   func (r *Reader) ReadChunks(chunkSize int) <-chan []byte {
       ch := make(chan []byte, 10)
       go func() {
           defer close(ch)
           buffer := make([]byte, chunkSize)
           for {
               n, err := r.source.Read(buffer)
               if n > 0 {
                   ch <- buffer[:n]
               }
               if err != nil {
                   break
               }
           }
       }()
       return ch
   }
   ```

3. **Memory Pressure Handling**
   ```go
   type MemoryController struct {
       threshold float64
       pools     []*BufferPool
   }
   
   func (mc *MemoryController) HandlePressure() {
       if mc.getMemoryUsage() > mc.threshold {
           // Trigger emergency cleanup
           for _, pool := range mc.pools {
               pool.Shrink(0.5) // Reduce pool size by 50%
           }
           runtime.GC() // Force garbage collection
       }
   }
   ```

## Running the Exercise

```bash
# Build the system
go build -o memory-system .

# Run with sample data
./memory-system --input sample.json --output processed.json

# Run performance benchmarks
go test -bench=. -benchmem

# Run system tests
go test -v ./...

# Profile memory usage
go test -bench=BenchmarkSystem -memprofile=mem.prof
go tool pprof mem.prof

# Monitor real-time metrics
./memory-system --monitor --dashboard-port 8080
```

## Performance Targets

Your implementation should achieve:

| Metric | Target | Measurement |
|--------|--------|--------------|
| Throughput | >10,000 records/sec | `go test -bench=BenchmarkThroughput` |
| Latency (avg) | <1ms | `go test -bench=BenchmarkLatency` |
| Latency (P99) | <10ms | Statistical analysis |
| Memory Usage | <100MB | Runtime monitoring |
| Allocation Rate | <1MB/sec | Memory profiling |
| GC Frequency | <1/sec | GC metrics |

## Success Criteria

- [ ] Complete pipeline processes 1GB+ datasets within memory limits
- [ ] Achieve all performance targets consistently
- [ ] Implement comprehensive memory monitoring
- [ ] Handle memory pressure gracefully without crashes
- [ ] Demonstrate zero-allocation processing in hot paths
- [ ] Create production-ready configuration and deployment
- [ ] Document architecture decisions and trade-offs

## Advanced Challenges

1. **Dynamic Scaling**: Implement auto-scaling based on memory pressure
2. **Multi-Format Support**: Add support for Protocol Buffers, Avro
3. **Distributed Processing**: Extend to multi-node processing
4. **ML Integration**: Add machine learning model inference
5. **Real-time Analytics**: Implement streaming analytics

## Evaluation Rubric

| Aspect | Excellent (4) | Good (3) | Satisfactory (2) | Needs Improvement (1) |
|--------|---------------|----------|------------------|----------------------|
| **Architecture** | Clean, modular, extensible | Well-structured | Basic organization | Poor structure |
| **Memory Efficiency** | <50MB usage, zero-alloc | <75MB, minimal alloc | <100MB, some alloc | >100MB, many alloc |
| **Performance** | Exceeds all targets | Meets most targets | Meets some targets | Below targets |
| **Monitoring** | Comprehensive metrics | Good coverage | Basic monitoring | Limited monitoring |
| **Code Quality** | Excellent practices | Good practices | Acceptable | Poor practices |

## Next Steps

After completing this exercise:
1. Deploy to production environment
2. Implement additional data sources/sinks
3. Add machine learning capabilities
4. Explore distributed processing patterns
5. Contribute optimizations back to the community

---

**Estimated Completion Time:** 60-90 minutes  
**Prerequisites:** Exercises 04-01, 04-02, and 04-03  
**Next Module:** [CPU Performance Optimization](../../../03-cpu-performance/README.md)