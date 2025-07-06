# Testing Framework for Memory Leak Detection System

This document provides comprehensive information about the testing framework for the Memory Leak Detection System, including test types, execution methods, and validation criteria.

## 📋 Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Test Types](#test-types)
- [Running Tests](#running-tests)
- [Test Coverage](#test-coverage)
- [Benchmarking](#benchmarking)
- [Continuous Integration](#continuous-integration)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## 🎯 Overview

The testing framework validates the functionality, performance, and reliability of the Memory Leak Detection System. It includes:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test component interactions and end-to-end workflows
- **Benchmark Tests**: Measure performance characteristics
- **Stress Tests**: Validate system behavior under load
- **Coverage Analysis**: Ensure comprehensive test coverage

## 🏗️ Test Structure

```
tests/
├── unit_test.go           # Unit tests for individual components
├── integration_test.go    # Integration and end-to-end tests
├── benchmark_test.go      # Performance benchmarks
run_tests.go              # Go-based test runner
run_tests.ps1             # PowerShell test runner for Windows
TESTING.md               # This documentation
```

## 🧪 Test Types

### Unit Tests

Unit tests validate individual components in isolation:

#### Heap Analyzer Tests
- **TestHeapAnalyzer**: Validates heap analysis functionality
- **TestDetectorMetricsValidation**: Ensures metric consistency
- **TestDetectorErrorHandling**: Tests error scenarios
- **TestDetectorConcurrency**: Validates thread safety

#### Goroutine Detector Tests
- **TestGoroutineDetector**: Validates goroutine leak detection
- **TestStackTraceCollection**: Tests stack trace capture

#### Reference Detector Tests
- **TestReferenceDetector**: Validates reference cycle detection
- **TestReferenceTracking**: Tests object tracking

#### Resource Detector Tests
- **TestResourceDetector**: Validates resource leak detection
- **TestSystemResourceUsage**: Tests system resource monitoring

#### Simulator Tests
- **TestLeakSimulator**: Validates leak simulation
- **TestSimulatorPatterns**: Tests different simulation patterns
- **TestSimulatorIntensities**: Tests various intensity levels

### Integration Tests

Integration tests validate component interactions and complete workflows:

#### Basic Functionality
- **TestBasicLeakDetection**: End-to-end leak detection without simulation
- **TestAnalyzerLifecycle**: Tests analyzer start/stop lifecycle

#### Leak Detection with Simulation
- **TestHeapLeakDetection**: Heap leak detection with simulation
- **TestGoroutineLeakDetection**: Goroutine leak detection with simulation
- **TestResourceLeakDetection**: Resource leak detection with simulation
- **TestMultiTypeLeakDetection**: Multiple leak types simultaneously

#### Real-time Features
- **TestRealTimeAlerts**: Real-time alert generation
- **TestDetectorConfiguration**: Configuration validation

### Benchmark Tests

Benchmark tests measure performance characteristics:

#### Component Performance
- **BenchmarkHeapAnalyzer**: Heap analyzer performance
- **BenchmarkGoroutineDetector**: Goroutine detector performance
- **BenchmarkReferenceDetector**: Reference detector performance
- **BenchmarkResourceDetector**: Resource detector performance

#### System Performance
- **BenchmarkAnalyzerOverhead**: Overall system overhead
- **BenchmarkMetricsCollection**: Metrics collection performance
- **BenchmarkConcurrentDetection**: Concurrent operation performance
- **BenchmarkMemoryUsage**: Memory usage analysis

#### Scalability
- **BenchmarkLargeDataset**: Performance with large datasets
- **BenchmarkReportGeneration**: Report generation performance
- **BenchmarkAlertProcessing**: Alert processing performance

## 🚀 Running Tests

### Prerequisites

1. **Go 1.21+** installed and configured
2. **Project dependencies** installed (`go mod download`)
3. **Sufficient system resources** (recommended: 4GB RAM, 2 CPU cores)

### Quick Start

```bash
# Run all tests
go run run_tests.go all

# Run quick tests only
go run run_tests.go quick

# Run with PowerShell (Windows)
.\run_tests.ps1 -Command all -Verbose
```

### Test Commands

#### Using Go Test Runner

```bash
# All tests with coverage and benchmarks
go run run_tests.go all

# Quick validation tests
go run run_tests.go quick

# Stress tests for load validation
go run run_tests.go stress

# Benchmarks only
go run run_tests.go bench

# Coverage report only
go run run_tests.go coverage

# Environment validation
go run run_tests.go validate
```

#### Using PowerShell Runner (Windows)

```powershell
# All tests with verbose output
.\run_tests.ps1 -Command all -Verbose

# Quick tests without coverage
.\run_tests.ps1 -Command quick -NoCoverage

# Benchmarks with custom output directory
.\run_tests.ps1 -Command bench -OutputDir "my-results"

# Stress tests with extended timeout
.\run_tests.ps1 -Command stress -Timeout 900

# Clean test artifacts
.\run_tests.ps1 -Command clean
```

#### Using Standard Go Commands

```bash
# Run all tests
go test ./tests -v

# Run specific test
go test ./tests -run TestHeapAnalyzer -v

# Run with race detection
go test ./tests -race -v

# Run with coverage
go test ./tests -cover -v

# Run benchmarks
go test ./tests -bench=. -benchmem
```

### Test Configuration

Tests can be configured through environment variables:

```bash
# Set test timeout
export TEST_TIMEOUT=600s

# Enable verbose output
export TEST_VERBOSE=true

# Set output directory
export TEST_OUTPUT_DIR=./custom-results

# Disable race detection
export TEST_NO_RACE=true
```

## 📊 Test Coverage

### Coverage Targets

- **Overall Coverage**: ≥ 80%
- **Critical Components**: ≥ 90%
  - Heap Analyzer
  - Goroutine Detector
  - Resource Detector
- **Integration Paths**: ≥ 85%

### Coverage Reports

Coverage reports are generated in multiple formats:

1. **HTML Report**: `test-results/coverage.html` - Interactive coverage visualization
2. **Text Summary**: `test-results/coverage-summary.txt` - Function-level coverage
3. **Raw Data**: `test-results/coverage.out` - Go coverage format

### Viewing Coverage

```bash
# Generate and view HTML coverage report
go test ./tests -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View coverage summary
go tool cover -func=coverage.out
```

## ⚡ Benchmarking

### Benchmark Categories

1. **Component Benchmarks**: Individual detector performance
2. **System Benchmarks**: Overall system performance
3. **Memory Benchmarks**: Memory usage and allocation patterns
4. **Concurrency Benchmarks**: Multi-threaded performance

### Running Benchmarks

```bash
# Run all benchmarks
go test ./tests -bench=. -benchmem

# Run specific benchmark
go test ./tests -bench=BenchmarkHeapAnalyzer -benchmem

# Run benchmarks multiple times for stability
go test ./tests -bench=. -benchmem -count=5

# Generate benchmark comparison
go test ./tests -bench=. -benchmem > old.txt
# ... make changes ...
go test ./tests -bench=. -benchmem > new.txt
benchcmp old.txt new.txt
```

### Benchmark Interpretation

```
BenchmarkHeapAnalyzer-8    1000    1234567 ns/op    1024 B/op    10 allocs/op
                      │       │           │           │          │
                      │       │           │           │          └─ Allocations per operation
                      │       │           │           └─ Bytes allocated per operation
                      │       │           └─ Nanoseconds per operation
                      │       └─ Number of iterations
                      └─ Number of CPU cores
```

### Performance Targets

- **Heap Analysis**: < 10ms per analysis
- **Goroutine Detection**: < 5ms per analysis
- **Resource Monitoring**: < 2ms per collection
- **Memory Overhead**: < 50MB for typical workloads

## 🔄 Continuous Integration

### CI Pipeline

The testing framework supports CI/CD integration:

```yaml
# Example GitHub Actions workflow
name: Test Suite
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    - run: go mod download
    - run: go run run_tests.go all
    - uses: actions/upload-artifact@v3
      with:
        name: test-results
        path: test-results/
```

### Quality Gates

1. **All tests must pass**
2. **Coverage ≥ 80%**
3. **No race conditions detected**
4. **Benchmarks within performance targets**
5. **No memory leaks in test suite itself**

## 🔧 Troubleshooting

### Common Issues

#### Test Timeouts
```bash
# Increase timeout for slow systems
go test ./tests -timeout=10m
```

#### Race Condition Failures
```bash
# Run with race detection for debugging
go test ./tests -race -v
```

#### Memory Issues
```bash
# Monitor memory usage during tests
go test ./tests -memprofile=mem.prof
go tool pprof mem.prof
```

#### Coverage Issues
```bash
# Check which files are missing coverage
go test ./tests -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -v "100.0%"
```

### Debug Mode

Enable debug mode for detailed test output:

```bash
# Set debug environment variable
export DEBUG=true
go test ./tests -v
```

### Test Data Cleanup

Tests create temporary files and directories. Cleanup is automatic, but manual cleanup:

```bash
# Clean test artifacts
rm -rf test-results/
rm -rf /tmp/test_*
```

## 🤝 Contributing

### Adding New Tests

1. **Unit Tests**: Add to `tests/unit_test.go`
2. **Integration Tests**: Add to `tests/integration_test.go`
3. **Benchmarks**: Add to `tests/benchmark_test.go`

### Test Naming Conventions

- **Unit Tests**: `TestComponentName`
- **Integration Tests**: `TestFeatureName`
- **Benchmarks**: `BenchmarkComponentName`
- **Helper Functions**: `testHelperName` or `benchmarkHelperName`

### Test Structure Template

```go
func TestNewFeature(t *testing.T) {
    // Setup
    config := &ComponentConfig{
        // test configuration
    }
    component := NewComponent(config)
    
    // Test execution
    result, err := component.DoSomething()
    
    // Assertions
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    if result == nil {
        t.Error("Result should not be nil")
    }
    
    // Cleanup
    component.Cleanup()
}
```

### Benchmark Template

```go
func BenchmarkNewFeature(b *testing.B) {
    // Setup
    component := setupBenchmarkComponent()
    defer component.Cleanup()
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        result := component.DoSomething()
        if result == nil {
            b.Fatal("Unexpected nil result")
        }
    }
}
```

### Code Coverage Guidelines

1. **Test all public functions**
2. **Test error conditions**
3. **Test edge cases**
4. **Test concurrent access**
5. **Avoid testing implementation details**

### Performance Testing Guidelines

1. **Use realistic data sizes**
2. **Test under various loads**
3. **Measure memory allocations**
4. **Test concurrent scenarios**
5. **Establish performance baselines**

## 📈 Test Metrics

### Success Criteria

- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ Code coverage ≥ 80%
- ✅ No race conditions
- ✅ Benchmarks within targets
- ✅ No memory leaks

### Monitoring

Track these metrics over time:

1. **Test execution time**
2. **Coverage percentage**
3. **Benchmark performance**
4. **Flaky test rate**
5. **Test maintenance overhead**

## 🔗 Related Documentation

- [README.md](README.md) - Project overview and setup
- [examples/](examples/) - Usage examples
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Go Coverage Tools](https://golang.org/doc/code.html#Testing)

---

**Note**: This testing framework is designed to be comprehensive and maintainable. Regular updates ensure it stays aligned with the evolving codebase and testing best practices.