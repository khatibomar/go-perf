# Memory-Conscious Pipeline System Makefile

.PHONY: all build test benchmark clean run help deps

# Default target
all: deps build test

# Build the application
build:
	@echo "Building memory-conscious pipeline..."
	go build -o bin/pipeline.exe .

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -v

# Run specific benchmark
benchmark-pool:
	@echo "Running buffer pool benchmarks..."
	go test -bench=BenchmarkBufferPool -benchmem -v

benchmark-pipeline:
	@echo "Running pipeline benchmarks..."
	go test -bench=BenchmarkPipeline -benchmem -v

# Run the application with default settings
run:
	@echo "Running pipeline with default settings..."
	go run . -input=sample.txt -output=output.txt -workers=4 -chunk-size=1024

# Run with memory monitoring
run-monitor:
	@echo "Running pipeline with memory monitoring..."
	go run . -input=sample.txt -output=output.txt -workers=4 -chunk-size=1024 -memory-limit=50MB -monitor

# Run with uppercase transformation
run-uppercase:
	@echo "Running pipeline with uppercase transformation..."
	go run . -input=sample.txt -output=output.txt -processor=uppercase -workers=2

# Run with word count
run-wordcount:
	@echo "Running pipeline with word count transformation..."
	go run . -input=sample.txt -output=output.txt -processor=wordcount -workers=2

# Create sample input file
sample:
	@echo "Creating sample input file..."
	go run . -create-sample

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bin/pipeline.exe
	rm -f coverage.out coverage.html
	rm -f sample.txt output.txt
	rm -f cpu.prof mem.prof

# Profile CPU usage
profile-cpu:
	@echo "Running CPU profiling..."
	go run . -input=sample.txt -output=output.txt -workers=4 -cpuprofile=cpu.prof
	go tool pprof cpu.prof

# Profile memory usage
profile-mem:
	@echo "Running memory profiling..."
	go run . -input=sample.txt -output=output.txt -workers=4 -memprofile=mem.prof
	go tool pprof mem.prof

# Run race detection
race:
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Install tools
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  all           - Download deps, build, and test"
	@echo "  build         - Build the application"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  benchmark     - Run all benchmarks"
	@echo "  benchmark-pool - Run buffer pool benchmarks"
	@echo "  benchmark-pipeline - Run pipeline benchmarks"
	@echo "  run           - Run with default settings"
	@echo "  run-monitor   - Run with memory monitoring"
	@echo "  run-uppercase - Run with uppercase transformation"
	@echo "  run-wordcount - Run with word count transformation"
	@echo "  sample        - Create sample input file"
	@echo "  clean         - Clean build artifacts"
	@echo "  profile-cpu   - Run CPU profiling"
	@echo "  profile-mem   - Run memory profiling"
	@echo "  race          - Run tests with race detection"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  vet           - Vet code"
	@echo "  tools         - Install development tools"
	@echo "  help          - Show this help"