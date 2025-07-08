# Sorting Algorithm Performance Analysis Makefile

.PHONY: all build test bench profile report clean help

# Default target
all: build test

# Build the main executable
build:
	@echo "Building sorting algorithm analyzer..."
	go build -o sort-analyzer.exe main.go
	@echo "✓ Build complete"

# Run correctness tests
test:
	@echo "Running correctness tests..."
	go run main.go -test
	@echo "✓ Correctness tests complete"

# Run quick benchmarks
bench-quick:
	@echo "Running quick benchmarks..."
	go run main.go -bench -quick -verbose
	@echo "✓ Quick benchmarks complete"

# Run full benchmarks
bench:
	@echo "Running comprehensive benchmarks..."
	go run main.go -bench -verbose
	@echo "✓ Comprehensive benchmarks complete"

# Run benchmarks and generate text report
report-text:
	@echo "Generating text performance report..."
	go run main.go -bench -report -format=text -output=performance_report.txt
	@echo "✓ Text report saved to performance_report.txt"

# Run benchmarks and generate CSV report
report-csv:
	@echo "Generating CSV performance report..."
	go run main.go -bench -report -format=csv -output=performance_report.csv
	@echo "✓ CSV report saved to performance_report.csv"

# Run benchmarks and generate Markdown report
report-markdown:
	@echo "Generating Markdown performance report..."
	go run main.go -bench -report -format=markdown -output=performance_report.md
	@echo "✓ Markdown report saved to performance_report.md"

# Generate all report formats
report-all: report-text report-csv report-markdown
	@echo "✓ All reports generated"

# Run profiling tests
profile:
	@echo "Running profiling tests..."
	go run main.go -profile -verbose
	@echo "✓ Profiling tests complete"

# Run Go benchmarks with CPU profiling
profile-cpu:
	@echo "Running CPU profiling benchmarks..."
	go test -bench=. -cpuprofile=cpu.prof ./benchmarks
	@echo "✓ CPU profile saved to cpu.prof"
	@echo "  View with: go tool pprof cpu.prof"

# Run Go benchmarks with memory profiling
profile-mem:
	@echo "Running memory profiling benchmarks..."
	go test -bench=. -memprofile=mem.prof ./benchmarks
	@echo "✓ Memory profile saved to mem.prof"
	@echo "  View with: go tool pprof mem.prof"

# Compare specific algorithms
compare-quick-merge:
	@echo "Comparing QuickSort vs MergeSort..."
	go run main.go -bench -compare=quicksort,mergesort

# Compare all algorithms
compare-all:
	@echo "Comparing all algorithms..."
	go run main.go -bench -compare=quicksort,mergesort,heapsort,radixsort,timsort

# Run comprehensive analysis (tests + benchmarks + reports)
analyze:
	@echo "Running comprehensive analysis..."
	go run main.go -test -bench -report -format=markdown -output=analysis_report.md
	@echo "✓ Comprehensive analysis complete - see analysis_report.md"

# Run performance regression tests
regression:
	@echo "Running performance regression tests..."
	go test -bench=BenchmarkRegression ./benchmarks
	@echo "✓ Regression tests complete"

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@if exist sort-analyzer.exe del sort-analyzer.exe
	@if exist *.prof del *.prof
	@if exist *.txt del *.txt
	@if exist *.csv del *.csv
	@if exist *.md del *.md
	@echo "✓ Clean complete"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "✓ Code formatting complete"

# Run Go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ Go vet complete"

# Run static analysis
lint: fmt vet
	@echo "✓ Static analysis complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	@echo "✓ Dependencies installed"

# Run all quality checks
check: lint test
	@echo "✓ All quality checks passed"

# Development workflow
dev: clean deps check bench-quick
	@echo "✓ Development workflow complete"

# Production analysis workflow
prod: clean deps check analyze profile-cpu
	@echo "✓ Production analysis workflow complete"

# Benchmark specific algorithm
bench-quicksort:
	@echo "Benchmarking QuickSort variants..."
	go test -bench=BenchmarkQuickSort ./benchmarks

bench-mergesort:
	@echo "Benchmarking MergeSort variants..."
	go test -bench=BenchmarkMergeSort ./benchmarks

bench-heapsort:
	@echo "Benchmarking HeapSort variants..."
	go test -bench=BenchmarkHeapSort ./benchmarks

bench-radixsort:
	@echo "Benchmarking RadixSort variants..."
	go test -bench=BenchmarkRadixSort ./benchmarks

bench-timsort:
	@echo "Benchmarking TimSort variants..."
	go test -bench=BenchmarkTimSort ./benchmarks

# Memory usage analysis
memory-analysis:
	@echo "Running memory usage analysis..."
	go test -bench=BenchmarkMemory -memprofile=memory.prof ./benchmarks
	go tool pprof -top memory.prof

# Cache performance analysis
cache-analysis:
	@echo "Running cache performance analysis..."
	go test -bench=BenchmarkCache ./benchmarks

# Branch prediction analysis
branch-analysis:
	@echo "Running branch prediction analysis..."
	go test -bench=BenchmarkBranch ./benchmarks

# Small array optimization analysis
small-array-analysis:
	@echo "Running small array optimization analysis..."
	go test -bench=BenchmarkSmallArray ./benchmarks

# Large array performance analysis
large-array-analysis:
	@echo "Running large array performance analysis..."
	go test -bench=BenchmarkLargeArray ./benchmarks

# Worst case scenario analysis
worst-case-analysis:
	@echo "Running worst case scenario analysis..."
	go test -bench=BenchmarkWorstCase ./benchmarks

# Best case scenario analysis
best-case-analysis:
	@echo "Running best case scenario analysis..."
	go test -bench=BenchmarkBestCase ./benchmarks

# Help target
help:
	@echo "Available targets:"
	@echo "  build              - Build the main executable"
	@echo "  test               - Run correctness tests"
	@echo "  bench-quick        - Run quick benchmarks"
	@echo "  bench              - Run comprehensive benchmarks"
	@echo "  report-text        - Generate text report"
	@echo "  report-csv         - Generate CSV report"
	@echo "  report-markdown    - Generate Markdown report"
	@echo "  report-all         - Generate all report formats"
	@echo "  profile            - Run profiling tests"
	@echo "  profile-cpu        - Run CPU profiling"
	@echo "  profile-mem        - Run memory profiling"
	@echo "  compare-quick-merge - Compare QuickSort vs MergeSort"
	@echo "  compare-all        - Compare all algorithms"
	@echo "  analyze            - Run comprehensive analysis"
	@echo "  clean              - Clean generated files"
	@echo "  fmt                - Format Go code"
	@echo "  vet                - Run go vet"
	@echo "  lint               - Run static analysis"
	@echo "  deps               - Install dependencies"
	@echo "  check              - Run all quality checks"
	@echo "  dev                - Development workflow"
	@echo "  prod               - Production analysis workflow"
	@echo ""
	@echo "Algorithm-specific benchmarks:"
	@echo "  bench-quicksort    - Benchmark QuickSort variants"
	@echo "  bench-mergesort    - Benchmark MergeSort variants"
	@echo "  bench-heapsort     - Benchmark HeapSort variants"
	@echo "  bench-radixsort    - Benchmark RadixSort variants"
	@echo "  bench-timsort      - Benchmark TimSort variants"
	@echo ""
	@echo "Specialized analysis:"
	@echo "  memory-analysis    - Memory usage analysis"
	@echo "  cache-analysis     - Cache performance analysis"
	@echo "  branch-analysis    - Branch prediction analysis"
	@echo "  small-array-analysis - Small array optimization"
	@echo "  large-array-analysis - Large array performance"
	@echo "  worst-case-analysis  - Worst case scenarios"
	@echo "  best-case-analysis   - Best case scenarios"