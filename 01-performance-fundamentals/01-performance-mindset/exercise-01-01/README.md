# Exercise 01-01: Environment Setup and First Measurement

**Objective:** Set up your environment and create your first performance measurement.

**Duration:** 45-60 minutes

## Tasks

### Task 1: Environment Verification
1. Verify Go installation (Go 1.21+ required)
2. Install required tools:
   ```bash
   go install github.com/google/pprof@latest
   go install golang.org/x/tools/cmd/benchcmp@latest
   ```
3. Test basic benchmark functionality

### Task 2: Create Your First Benchmark
1. Examine the provided `main.go` file
2. Write benchmarks in `benchmark_test.go` for the functions
3. Run benchmarks and analyze results:
   ```bash
   go test -bench=.
   go test -bench=. -benchmem
   ```

### Task 3: Performance Measurement
1. Measure baseline performance
2. Document results
3. Identify potential optimization opportunities

## Expected Outcomes
- Working Go performance environment
- Understanding of basic benchmark syntax
- First performance measurements
- Baseline for future optimizations

## Files
- `main.go` - Sample functions to benchmark
- `benchmark_test.go` - Benchmark implementations
- `go.mod` - Module definition

## Next Steps
Proceed to Exercise 01-02 to learn about performance anti-patterns and improvements.