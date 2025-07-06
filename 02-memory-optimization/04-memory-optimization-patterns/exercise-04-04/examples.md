# Memory-Conscious Pipeline Examples

This document provides practical examples of using the memory-conscious pipeline system.

## Basic Usage Examples

### 1. Create Sample Data and Process

```bash
# Create a sample input file
./pipeline.exe -create-sample

# Process with default settings (uppercase transformation)
./pipeline.exe

# Process with specific settings
./pipeline.exe -input=sample.txt -output=result.txt -workers=4 -chunk-size=1024
```

### 2. Different Text Transformations

```bash
# Convert to uppercase
./pipeline.exe -processor=uppercase -input=data.txt -output=upper.txt

# Convert to lowercase
./pipeline.exe -processor=lowercase -input=data.txt -output=lower.txt

# Count words in the text
./pipeline.exe -processor=wordcount -input=document.txt -output=stats.txt

# Reverse text content
./pipeline.exe -processor=reverse -input=text.txt -output=reversed.txt

# Trim whitespace from lines
./pipeline.exe -processor=trim -input=messy.txt -output=clean.txt
```

### 3. Memory Management Examples

```bash
# Set memory limit to 100MB
./pipeline.exe -memory=104857600 -input=large.txt

# Process with memory monitoring enabled
./pipeline.exe -monitor -input=data.txt -memory=50000000

# High-memory processing with more workers
./pipeline.exe -memory=200000000 -workers=8 -chunk-size=4096
```

### 4. Performance Tuning Examples

```bash
# Single-threaded processing (debugging)
./pipeline.exe -workers=1 -chunk-size=512 -monitor

# High-throughput processing
./pipeline.exe -workers=8 -chunk-size=8192 -memory=100000000

# Small chunk processing (memory-constrained)
./pipeline.exe -workers=2 -chunk-size=256 -memory=10000000

# Large chunk processing (high-performance)
./pipeline.exe -workers=6 -chunk-size=16384 -memory=200000000
```

## Advanced Usage Scenarios

### 1. Processing Large Files

```bash
# For files > 100MB
./pipeline.exe -input=largefile.txt -memory=500000000 -workers=8 -chunk-size=32768 -monitor

# Memory-constrained environment
./pipeline.exe -input=largefile.txt -memory=20000000 -workers=2 -chunk-size=1024
```

### 2. Real-time Monitoring

```bash
# Monitor processing with detailed metrics
./pipeline.exe -monitor -input=data.txt -output=processed.txt

# Monitor memory pressure during processing
./pipeline.exe -monitor -memory=30000000 -input=largefile.txt
```

### 3. Development and Testing

```bash
# Create test data
./pipeline.exe -create-sample -input=test_data.txt

# Debug with single worker
./pipeline.exe -workers=1 -monitor -input=test_data.txt

# Performance testing
./pipeline.exe -workers=4 -chunk-size=2048 -monitor -input=test_data.txt
```

## Build and Test Examples

### Using Make Commands

```bash
# Build the project
make build

# Run all tests
make test

# Run benchmarks
make benchmark

# Run with coverage
make test-coverage

# Create sample and run
make sample
make run

# Run with monitoring
make run-monitor

# Run specific transformations
make run-uppercase
make run-wordcount

# Performance profiling
make profile-cpu
make profile-mem

# Code quality checks
make fmt
make vet
make lint
```

### Manual Go Commands

```bash
# Build manually
go build -o pipeline.exe .

# Run tests
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem

# Run with race detection
go test -race ./...

# Build with optimizations
go build -ldflags="-s -w" -o pipeline.exe .
```

## Performance Optimization Tips

### 1. Memory Settings

- **Small files (<10MB)**: Use default memory settings
- **Medium files (10-100MB)**: Set memory limit to 50-100MB
- **Large files (>100MB)**: Set memory limit to 200MB+ and increase workers

### 2. Worker Configuration

- **CPU cores**: Start with worker count = CPU cores
- **I/O bound**: Use 2x CPU cores
- **Memory constrained**: Use fewer workers (1-2)
- **High throughput**: Use more workers (up to 8-12)

### 3. Chunk Size Tuning

- **Small chunks (256-1024 bytes)**: Better for memory-constrained environments
- **Medium chunks (2048-8192 bytes)**: Good balance for most use cases
- **Large chunks (16384+ bytes)**: Better for high-throughput scenarios

### 4. Monitoring Guidelines

- Always use `-monitor` for large file processing
- Watch for memory pressure events
- Adjust settings if throughput is low
- Monitor error rates during processing

## Troubleshooting Examples

### Common Issues and Solutions

```bash
# Issue: High memory usage
# Solution: Reduce chunk size and workers
./pipeline.exe -workers=2 -chunk-size=512 -memory=20000000

# Issue: Slow processing
# Solution: Increase workers and chunk size
./pipeline.exe -workers=6 -chunk-size=4096

# Issue: Memory pressure warnings
# Solution: Increase memory limit or reduce load
./pipeline.exe -memory=100000000 -workers=4

# Issue: File not found errors
# Solution: Create sample file first
./pipeline.exe -create-sample
./pipeline.exe -input=sample.txt
```

### Debug Mode

```bash
# Enable verbose output with single worker
./pipeline.exe -workers=1 -monitor -chunk-size=256

# Test with minimal settings
./pipeline.exe -workers=1 -chunk-size=128 -memory=10000000
```

## Integration Examples

### Shell Scripts

```bash
#!/bin/bash
# Process multiple files
for file in *.txt; do
    ./pipeline.exe -input="$file" -output="processed_$file" -processor=uppercase
done

# Batch processing with different transformations
./pipeline.exe -input=data.txt -output=upper.txt -processor=uppercase
./pipeline.exe -input=data.txt -output=lower.txt -processor=lowercase
./pipeline.exe -input=data.txt -output=stats.txt -processor=wordcount
```

### Performance Testing Script

```bash
#!/bin/bash
# Test different configurations
echo "Testing different worker counts..."
for workers in 1 2 4 8; do
    echo "Workers: $workers"
    time ./pipeline.exe -workers=$workers -input=large.txt -output=out_$workers.txt
done

echo "Testing different chunk sizes..."
for chunk in 512 1024 2048 4096; do
    echo "Chunk size: $chunk"
    time ./pipeline.exe -chunk-size=$chunk -input=large.txt -output=out_$chunk.txt
done
```

These examples demonstrate the flexibility and power of the memory-conscious pipeline system for various text processing scenarios.