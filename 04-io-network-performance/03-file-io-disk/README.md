# Chapter 03: File I/O and Disk Performance

**Focus:** File operations, disk I/O, async patterns

## Chapter Overview

This chapter focuses on optimizing file I/O operations and disk performance in Go applications. You'll learn to implement asynchronous file operations, use memory-mapped files, optimize batch processing, and profile disk I/O performance for maximum throughput and minimal latency.

## Learning Objectives

By the end of this chapter, you will:

- ✅ Implement high-performance async file operations
- ✅ Use memory-mapped files for large file processing
- ✅ Design efficient batch file processing systems
- ✅ Profile and optimize disk I/O performance
- ✅ Handle file operations with proper error handling
- ✅ Implement zero-copy file operations
- ✅ Optimize buffer management for file I/O

## Prerequisites

- Understanding of Go file I/O packages (os, io, bufio)
- Basic knowledge of operating system file systems
- Familiarity with Linux file I/O concepts
- Understanding of memory management

## Chapter Structure

### Exercise 03-01: Async File Operations
**Objective:** Implement high-performance asynchronous file operations

**Key Concepts:**
- Asynchronous I/O patterns
- Worker pool for file operations
- Non-blocking file processing
- Error handling in async operations

**Performance Targets:**
- Process 1000+ files concurrently
- File operation throughput >100MB/s
- Memory usage <500MB for 10K files
- Error rate <0.01%

### Exercise 03-02: Memory-Mapped Files
**Objective:** Use memory-mapped files for efficient large file processing

**Key Concepts:**
- Memory mapping techniques
- Virtual memory optimization
- Large file handling
- Cross-platform compatibility

**Performance Targets:**
- Handle files >10GB efficiently
- Memory usage <1% of file size
- Random access time <1μs
- Zero-copy operations

### Exercise 03-03: Batch File Processing
**Objective:** Implement efficient batch processing for multiple files

**Key Concepts:**
- Batch processing patterns
- Pipeline architecture
- Resource management
- Progress tracking

**Performance Targets:**
- Process 10K+ files per batch
- Throughput >1GB/s
- CPU utilization >80%
- Memory efficiency <100MB overhead

### Exercise 03-04: Disk I/O Profiling and Optimization
**Objective:** Profile and optimize disk I/O performance

**Key Concepts:**
- I/O profiling techniques
- Disk performance analysis
- Buffer optimization
- System-level optimization

**Performance Targets:**
- Identify I/O bottlenecks
- Optimize buffer sizes
- Reduce syscall overhead
- Achieve >90% disk utilization

## Key Technologies

### Core Packages
- `os` - File system operations
- `io` - I/O primitives
- `bufio` - Buffered I/O
- `syscall` - System calls
- `unsafe` - Memory operations

### Advanced Packages
- `golang.org/x/sys/unix` - Unix-specific operations
- `github.com/edsrzf/mmap-go` - Memory mapping
- `sync` - Synchronization primitives
- `context` - Operation lifecycle

### Profiling Tools
- `go tool pprof` - Performance profiling
- `go tool trace` - Execution tracing
- `iotop` - I/O monitoring
- `iostat` - I/O statistics
- `strace` - System call tracing

## Implementation Patterns

### Async File Operations Pattern
```go
type AsyncFileProcessor struct {
    workerPool chan struct{}
    resultChan chan FileResult
    errorChan  chan error
    wg         sync.WaitGroup
}

type FileResult struct {
    Filename string
    Data     []byte
    Size     int64
    Duration time.Duration
}

func (p *AsyncFileProcessor) ProcessFile(filename string) {
    p.wg.Add(1)
    go func() {
        defer p.wg.Done()
        
        // Acquire worker slot
        p.workerPool <- struct{}{}
        defer func() { <-p.workerPool }()
        
        start := time.Now()
        data, err := p.readFileOptimized(filename)
        if err != nil {
            p.errorChan <- fmt.Errorf("failed to read %s: %w", filename, err)
            return
        }
        
        result := FileResult{
            Filename: filename,
            Data:     data,
            Size:     int64(len(data)),
            Duration: time.Since(start),
        }
        
        select {
        case p.resultChan <- result:
        case <-time.After(5 * time.Second):
            p.errorChan <- fmt.Errorf("timeout sending result for %s", filename)
        }
    }()
}

func (p *AsyncFileProcessor) readFileOptimized(filename string) ([]byte, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    stat, err := file.Stat()
    if err != nil {
        return nil, err
    }
    
    // Pre-allocate buffer based on file size
    data := make([]byte, stat.Size())
    _, err = io.ReadFull(file, data)
    return data, err
}
```

### Memory-Mapped Files Pattern
```go
type MMapFile struct {
    file *os.File
    data []byte
    size int64
}

func OpenMMapFile(filename string) (*MMapFile, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    
    stat, err := file.Stat()
    if err != nil {
        file.Close()
        return nil, err
    }
    
    data, err := mmap.Map(file, mmap.RDONLY, 0)
    if err != nil {
        file.Close()
        return nil, err
    }
    
    return &MMapFile{
        file: file,
        data: data,
        size: stat.Size(),
    }, nil
}

func (m *MMapFile) ReadAt(offset, length int64) []byte {
    if offset+length > m.size {
        length = m.size - offset
    }
    return m.data[offset : offset+length]
}

func (m *MMapFile) Close() error {
    if err := mmap.Unmap(m.data); err != nil {
        return err
    }
    return m.file.Close()
}
```

### Batch File Processing Pattern
```go
type BatchProcessor struct {
    batchSize   int
    workerCount int
    inputChan   chan string
    outputChan  chan BatchResult
    pipeline    []ProcessingStage
}

type ProcessingStage func([]byte) ([]byte, error)

type BatchResult struct {
    Files     []string
    Results   [][]byte
    Errors    []error
    Duration  time.Duration
    BytesRead int64
}

func (b *BatchProcessor) ProcessBatch(filenames []string) <-chan BatchResult {
    resultChan := make(chan BatchResult, 1)
    
    go func() {
        defer close(resultChan)
        
        start := time.Now()
        var totalBytes int64
        results := make([][]byte, len(filenames))
        errors := make([]error, len(filenames))
        
        // Process files in parallel
        var wg sync.WaitGroup
        semaphore := make(chan struct{}, b.workerCount)
        
        for i, filename := range filenames {
            wg.Add(1)
            go func(index int, file string) {
                defer wg.Done()
                
                semaphore <- struct{}{}
                defer func() { <-semaphore }()
                
                data, err := b.processFile(file)
                if err != nil {
                    errors[index] = err
                    return
                }
                
                results[index] = data
                atomic.AddInt64(&totalBytes, int64(len(data)))
            }(i, filename)
        }
        
        wg.Wait()
        
        resultChan <- BatchResult{
            Files:     filenames,
            Results:   results,
            Errors:    errors,
            Duration:  time.Since(start),
            BytesRead: totalBytes,
        }
    }()
    
    return resultChan
}

func (b *BatchProcessor) processFile(filename string) ([]byte, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    // Apply processing pipeline
    for _, stage := range b.pipeline {
        data, err = stage(data)
        if err != nil {
            return nil, err
        }
    }
    
    return data, nil
}
```

## Profiling and Debugging

### I/O Performance Monitoring
```bash
# Monitor disk I/O
iotop -o  # Show only processes doing I/O
iostat -x 1  # Extended I/O statistics

# Monitor file system usage
df -h  # Disk space usage
du -sh *  # Directory sizes

# Monitor file descriptors
lsof -p <pid>  # Open files for process
cat /proc/<pid>/limits  # Process limits
```

### System Call Tracing
```bash
# Trace file operations
strace -e trace=file ./your-program
strace -e trace=read,write,open,close ./your-program

# Count system calls
strace -c ./your-program

# Monitor specific file operations
strace -e trace=openat,read,write -f ./your-program
```

### Go I/O Profiling
```bash
# Profile I/O operations
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Trace I/O operations
go tool trace trace.out

# Monitor goroutines (check for I/O blocking)
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Performance Optimization Techniques

### 1. Buffer Optimization
- Use appropriate buffer sizes
- Implement buffer pooling
- Align buffers to page boundaries
- Use direct I/O when appropriate

### 2. Async I/O Patterns
- Use worker pools for file operations
- Implement non-blocking I/O
- Pipeline file processing
- Handle backpressure properly

### 3. Memory Management
- Use memory-mapped files for large files
- Implement zero-copy operations
- Manage memory allocation carefully
- Use streaming for large datasets

### 4. System-Level Optimization
- Tune file system parameters
- Use appropriate I/O schedulers
- Optimize disk layout
- Monitor system resources

## Advanced Optimization Techniques

### Zero-Copy File Operations
```go
func CopyFileZeroCopy(src, dst string) error {
    srcFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer srcFile.Close()
    
    dstFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer dstFile.Close()
    
    // Use sendfile for zero-copy transfer (Linux)
    srcFd := int(srcFile.Fd())
    dstFd := int(dstFile.Fd())
    
    stat, err := srcFile.Stat()
    if err != nil {
        return err
    }
    
    _, err = unix.Sendfile(dstFd, srcFd, nil, int(stat.Size()))
    return err
}
```

### Direct I/O Operations
```go
func ReadFileDirect(filename string) ([]byte, error) {
    file, err := os.OpenFile(filename, os.O_RDONLY|syscall.O_DIRECT, 0)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    stat, err := file.Stat()
    if err != nil {
        return nil, err
    }
    
    // Align buffer to page boundary for direct I/O
    pageSize := os.Getpagesize()
    bufferSize := int(stat.Size())
    if bufferSize%pageSize != 0 {
        bufferSize = ((bufferSize / pageSize) + 1) * pageSize
    }
    
    buffer := make([]byte, bufferSize)
    n, err := file.Read(buffer)
    if err != nil && err != io.EOF {
        return nil, err
    }
    
    return buffer[:n], nil
}
```

## Common Pitfalls

❌ **File I/O Issues:**
- Not handling partial reads/writes
- Inappropriate buffer sizes
- File descriptor leaks
- Not using buffered I/O appropriately

❌ **Performance Problems:**
- Synchronous I/O in hot paths
- Excessive memory allocations
- Poor error handling
- Not profiling I/O operations

❌ **Memory Management:**
- Memory leaks with large files
- Not using memory mapping appropriately
- Poor buffer management
- Inefficient data structures

❌ **Concurrency Issues:**
- Race conditions in file access
- Goroutine leaks
- Poor resource management
- Inadequate error handling

## Best Practices

### File Operations
- Always handle partial reads/writes
- Use appropriate buffer sizes
- Implement proper error handling
- Close files and release resources
- Use context for cancellation

### Performance Optimization
- Profile before optimizing
- Use async I/O for concurrent operations
- Implement buffer pooling
- Monitor system resources
- Test under realistic load

### Memory Management
- Use memory mapping for large files
- Implement streaming for large datasets
- Manage buffer allocation carefully
- Monitor memory usage
- Use zero-copy techniques when possible

### Production Considerations
- Implement comprehensive monitoring
- Handle disk space issues
- Plan for capacity scaling
- Implement proper logging
- Use health checks

## Success Criteria

- [ ] Async file operations handle 1000+ files concurrently
- [ ] Memory-mapped files handle >10GB efficiently
- [ ] Batch processing achieves >1GB/s throughput
- [ ] I/O profiling identifies and resolves bottlenecks
- [ ] Zero file descriptor leaks
- [ ] Proper error handling implemented
- [ ] Memory usage optimized for large files

## Real-World Applications

- **Log Processing:** High-volume log file analysis
- **Data Processing:** ETL pipelines and data transformation
- **Backup Systems:** Efficient file backup and restore
- **Media Processing:** Video and image file processing
- **Database Systems:** Data file management
- **Content Delivery:** Static file serving optimization

## Next Steps

After completing this chapter:
1. Proceed to **Chapter 04: Caching Strategies**
2. Apply file I/O optimizations to your projects
3. Study advanced file system concepts
4. Explore distributed file systems
5. Implement comprehensive I/O monitoring

---

**Ready to optimize file I/O performance?** Start with [Exercise 03-01: Async File Operations](./exercise-03-01/README.md)

*Chapter 03 of Module 04: I/O and Network Performance*