# Makefile for Go Compiler Optimization Exercise
# Compatible with Linux/Unix systems

# Variables
APP_NAME := exercise-02-02
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
OUTPUT_DIR := analysis_output
REPORT_FILE := optimization_analysis_report.md
BINARY_DIR := bin

# Build flags
DEBUG_FLAGS := -gcflags='-N -l'
DEFAULT_FLAGS := 
OPTIMIZED_FLAGS := -gcflags='-l=4'
STRIPPED_FLAGS := -ldflags='-s -w'
FULL_FLAGS := -gcflags='-l=4' -ldflags='-s -w'
RACE_FLAGS := -race

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
MAGENTA := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m

# Default target
.PHONY: all
all: help

# Help target
.PHONY: help
help:
	@echo "$(CYAN)Go Compiler Optimization Exercise - Makefile$(NC)"
	@echo "================================================="
	@echo ""
	@echo "$(YELLOW)Build Targets:$(NC)"
	@echo "  build-debug     - Build with debug flags (no optimizations)"
	@echo "  build-default   - Build with default flags"
	@echo "  build-optimized - Build with aggressive optimizations"
	@echo "  build-stripped  - Build with stripped debug info"
	@echo "  build-full      - Build with full optimizations + stripped"
	@echo "  build-race      - Build with race detection"
	@echo "  build-all       - Build all variants"
	@echo ""
	@echo "$(YELLOW)Analysis Targets:$(NC)"
	@echo "  analyze-inlining    - Analyze function inlining"
	@echo "  analyze-escape      - Analyze escape analysis"
	@echo "  analyze-assembly    - Generate assembly output"
	@echo "  analyze-ssa         - Analyze SSA optimizations"
	@echo "  analyze-binary-size - Compare binary sizes"
	@echo "  analyze-all         - Run all analyses"
	@echo ""
	@echo "$(YELLOW)Testing Targets:$(NC)"
	@echo "  test            - Run all tests"
	@echo "  test-verbose    - Run tests with verbose output"
	@echo "  benchmark       - Run performance benchmarks"
	@echo "  benchmark-mem   - Run benchmarks with memory stats"
	@echo "  benchmark-cpu   - Run CPU profiling benchmarks"
	@echo ""
	@echo "$(YELLOW)Utility Targets:$(NC)"
	@echo "  clean           - Clean build artifacts"
	@echo "  clean-all       - Clean all generated files"
	@echo "  deps            - Download dependencies"
	@echo "  fmt             - Format Go code"
	@echo "  vet             - Run go vet"
	@echo "  lint            - Run golint (if available)"
	@echo "  report          - Generate comprehensive analysis report"
	@echo ""
	@echo "$(YELLOW)Demo Targets:$(NC)"
	@echo "  demo-inlining   - Run inlining demonstration"
	@echo "  demo-escape     - Run escape analysis demonstration"
	@echo "  demo-flags      - Run build flags demonstration"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make build-all analyze-all  # Build all variants and run all analyses"
	@echo "  make benchmark report       # Run benchmarks and generate report"
	@echo "  make clean build-optimized  # Clean and build optimized version"

# Create directories
$(OUTPUT_DIR):
	@mkdir -p $(OUTPUT_DIR)

$(BINARY_DIR):
	@mkdir -p $(BINARY_DIR)

# Build targets
.PHONY: build-debug
build-debug: $(BINARY_DIR)
	@echo "$(BLUE)[BUILD]$(NC) Building debug version..."
	@go build $(DEBUG_FLAGS) -o $(BINARY_DIR)/$(APP_NAME)-debug .
	@echo "$(GREEN)[SUCCESS]$(NC) Debug build completed: $(BINARY_DIR)/$(APP_NAME)-debug"

.PHONY: build-default
build-default: $(BINARY_DIR)
	@echo "$(BLUE)[BUILD]$(NC) Building default version..."
	@go build $(DEFAULT_FLAGS) -o $(BINARY_DIR)/$(APP_NAME)-default .
	@echo "$(GREEN)[SUCCESS]$(NC) Default build completed: $(BINARY_DIR)/$(APP_NAME)-default"

.PHONY: build-optimized
build-optimized: $(BINARY_DIR)
	@echo "$(BLUE)[BUILD]$(NC) Building optimized version..."
	@go build $(OPTIMIZED_FLAGS) -o $(BINARY_DIR)/$(APP_NAME)-optimized .
	@echo "$(GREEN)[SUCCESS]$(NC) Optimized build completed: $(BINARY_DIR)/$(APP_NAME)-optimized"

.PHONY: build-stripped
build-stripped: $(BINARY_DIR)
	@echo "$(BLUE)[BUILD]$(NC) Building stripped version..."
	@go build $(STRIPPED_FLAGS) -o $(BINARY_DIR)/$(APP_NAME)-stripped .
	@echo "$(GREEN)[SUCCESS]$(NC) Stripped build completed: $(BINARY_DIR)/$(APP_NAME)-stripped"

.PHONY: build-full
build-full: $(BINARY_DIR)
	@echo "$(BLUE)[BUILD]$(NC) Building fully optimized version..."
	@go build $(FULL_FLAGS) -o $(BINARY_DIR)/$(APP_NAME)-full .
	@echo "$(GREEN)[SUCCESS]$(NC) Full optimization build completed: $(BINARY_DIR)/$(APP_NAME)-full"

.PHONY: build-race
build-race: $(BINARY_DIR)
	@echo "$(BLUE)[BUILD]$(NC) Building race detection version..."
	@go build $(RACE_FLAGS) -o $(BINARY_DIR)/$(APP_NAME)-race .
	@echo "$(GREEN)[SUCCESS]$(NC) Race detection build completed: $(BINARY_DIR)/$(APP_NAME)-race"

.PHONY: build-all
build-all: build-debug build-default build-optimized build-stripped build-full build-race
	@echo "$(GREEN)[SUCCESS]$(NC) All build variants completed"
	@echo "$(BLUE)[INFO]$(NC) Binary sizes:"
	@ls -lh $(BINARY_DIR)/$(APP_NAME)-* | awk '{print "  " $$9 ": " $$5}'

# Analysis targets
.PHONY: analyze-inlining
analyze-inlining: $(OUTPUT_DIR)
	@echo "$(MAGENTA)[ANALYSIS]$(NC) Running inlining analysis..."
	@go build -gcflags='-m' . 2>&1 | tee $(OUTPUT_DIR)/inlining_analysis.txt
	@echo "$(GREEN)[SUCCESS]$(NC) Inlining analysis saved to $(OUTPUT_DIR)/inlining_analysis.txt"

.PHONY: analyze-escape
analyze-escape: $(OUTPUT_DIR)
	@echo "$(MAGENTA)[ANALYSIS]$(NC) Running escape analysis..."
	@go build -gcflags='-m' . 2>&1 | grep -E "(escapes|moved to heap|does not escape)" | tee $(OUTPUT_DIR)/escape_analysis.txt
	@echo "$(GREEN)[SUCCESS]$(NC) Escape analysis saved to $(OUTPUT_DIR)/escape_analysis.txt"

.PHONY: analyze-assembly
analyze-assembly: $(OUTPUT_DIR)
	@echo "$(MAGENTA)[ANALYSIS]$(NC) Generating assembly output..."
	@go build -gcflags='-N -l -S' . 2>&1 > $(OUTPUT_DIR)/assembly_debug.s
	@go build -gcflags='-S' . 2>&1 > $(OUTPUT_DIR)/assembly_optimized.s
	@echo "$(GREEN)[SUCCESS]$(NC) Assembly files generated in $(OUTPUT_DIR)/"

.PHONY: analyze-ssa
analyze-ssa: $(OUTPUT_DIR)
	@echo "$(MAGENTA)[ANALYSIS]$(NC) Running SSA analysis..."
	@mkdir -p $(OUTPUT_DIR)/ssa
	@go build -gcflags='-d=ssa/start/on' . 2>&1 | head -100 > $(OUTPUT_DIR)/ssa/ssa_start.txt || true
	@go build -gcflags='-d=ssa/opt/on' . 2>&1 | head -100 > $(OUTPUT_DIR)/ssa/ssa_opt.txt || true
	@echo "$(GREEN)[SUCCESS]$(NC) SSA analysis saved to $(OUTPUT_DIR)/ssa/"

.PHONY: analyze-binary-size
analyze-binary-size: build-all $(OUTPUT_DIR)
	@echo "$(MAGENTA)[ANALYSIS]$(NC) Analyzing binary sizes..."
	@echo "Binary Size Comparison:" > $(OUTPUT_DIR)/binary_sizes.txt
	@echo "=====================" >> $(OUTPUT_DIR)/binary_sizes.txt
	@ls -lh $(BINARY_DIR)/$(APP_NAME)-* | awk '{print $$9 ": " $$5}' >> $(OUTPUT_DIR)/binary_sizes.txt
	@echo "$(GREEN)[SUCCESS]$(NC) Binary size analysis saved to $(OUTPUT_DIR)/binary_sizes.txt"

.PHONY: analyze-all
analyze-all: analyze-inlining analyze-escape analyze-assembly analyze-ssa analyze-binary-size
	@echo "$(GREEN)[SUCCESS]$(NC) All analyses completed"

# Testing targets
.PHONY: test
test:
	@echo "$(BLUE)[TEST]$(NC) Running tests..."
	@go test ./...

.PHONY: test-verbose
test-verbose:
	@echo "$(BLUE)[TEST]$(NC) Running tests with verbose output..."
	@go test -v ./...

.PHONY: benchmark
benchmark:
	@echo "$(BLUE)[BENCHMARK]$(NC) Running performance benchmarks..."
	@if [ -d "benchmarks" ]; then \
		go test -bench=. -benchmem ./benchmarks; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) No benchmarks directory found"; \
	fi

.PHONY: benchmark-mem
benchmark-mem:
	@echo "$(BLUE)[BENCHMARK]$(NC) Running memory benchmarks..."
	@if [ -d "benchmarks" ]; then \
		go test -bench=. -benchmem -memprofile=mem.prof ./benchmarks; \
		echo "$(GREEN)[SUCCESS]$(NC) Memory profile saved to mem.prof"; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) No benchmarks directory found"; \
	fi

.PHONY: benchmark-cpu
benchmark-cpu:
	@echo "$(BLUE)[BENCHMARK]$(NC) Running CPU benchmarks..."
	@if [ -d "benchmarks" ]; then \
		go test -bench=. -cpuprofile=cpu.prof ./benchmarks; \
		echo "$(GREEN)[SUCCESS]$(NC) CPU profile saved to cpu.prof"; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) No benchmarks directory found"; \
	fi

# Demo targets
.PHONY: demo-inlining
demo-inlining: build-default
	@echo "$(CYAN)[DEMO]$(NC) Running inlining demonstration..."
	@./$(BINARY_DIR)/$(APP_NAME)-default --optimization inlining --size 5000 --iterations 500 --verbose

.PHONY: demo-escape
demo-escape: build-default
	@echo "$(CYAN)[DEMO]$(NC) Running escape analysis demonstration..."
	@./$(BINARY_DIR)/$(APP_NAME)-default --optimization escape --size 1000 --iterations 100 --analysis

.PHONY: demo-flags
demo-flags: build-default
	@echo "$(CYAN)[DEMO]$(NC) Running build flags demonstration..."
	@./$(BINARY_DIR)/$(APP_NAME)-default --optimization flags --verbose

# Utility targets
.PHONY: clean
clean:
	@echo "$(BLUE)[CLEAN]$(NC) Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)
	@rm -f *.prof
	@echo "$(GREEN)[SUCCESS]$(NC) Build artifacts cleaned"

.PHONY: clean-all
clean-all: clean
	@echo "$(BLUE)[CLEAN]$(NC) Cleaning all generated files..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f $(REPORT_FILE)
	@echo "$(GREEN)[SUCCESS]$(NC) All generated files cleaned"

.PHONY: deps
deps:
	@echo "$(BLUE)[DEPS]$(NC) Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies updated"

.PHONY: fmt
fmt:
	@echo "$(BLUE)[FORMAT]$(NC) Formatting Go code..."
	@go fmt ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Code formatted"

.PHONY: vet
vet:
	@echo "$(BLUE)[VET]$(NC) Running go vet..."
	@go vet ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Vet completed"

.PHONY: lint
lint:
	@echo "$(BLUE)[LINT]$(NC) Running golint..."
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) golint not installed. Install with: go install golang.org/x/lint/golint@latest"; \
	fi

.PHONY: report
report: analyze-all benchmark $(OUTPUT_DIR)
	@echo "$(MAGENTA)[REPORT]$(NC) Generating comprehensive analysis report..."
	@echo "# Go Compiler Optimization Analysis Report" > $(REPORT_FILE)
	@echo "" >> $(REPORT_FILE)
	@echo "**Generated:** $$(date)" >> $(REPORT_FILE)
	@echo "**Go Version:** $$(go version)" >> $(REPORT_FILE)
	@echo "**Platform:** $$(uname -s) $$(uname -m)" >> $(REPORT_FILE)
	@echo "" >> $(REPORT_FILE)
	@echo "## Binary Size Analysis" >> $(REPORT_FILE)
	@echo "\`\`\`" >> $(REPORT_FILE)
	@cat $(OUTPUT_DIR)/binary_sizes.txt >> $(REPORT_FILE)
	@echo "\`\`\`" >> $(REPORT_FILE)
	@echo "" >> $(REPORT_FILE)
	@echo "## Inlining Analysis" >> $(REPORT_FILE)
	@echo "\`\`\`" >> $(REPORT_FILE)
	@head -20 $(OUTPUT_DIR)/inlining_analysis.txt >> $(REPORT_FILE)
	@echo "\`\`\`" >> $(REPORT_FILE)
	@echo "" >> $(REPORT_FILE)
	@echo "## Escape Analysis" >> $(REPORT_FILE)
	@echo "\`\`\`" >> $(REPORT_FILE)
	@head -20 $(OUTPUT_DIR)/escape_analysis.txt >> $(REPORT_FILE)
	@echo "\`\`\`" >> $(REPORT_FILE)
	@echo "" >> $(REPORT_FILE)
	@echo "## Recommendations" >> $(REPORT_FILE)
	@echo "" >> $(REPORT_FILE)
	@echo "- Use optimized builds for production: \`make build-full\`" >> $(REPORT_FILE)
	@echo "- Monitor inlining decisions for hot paths" >> $(REPORT_FILE)
	@echo "- Minimize heap allocations in critical code" >> $(REPORT_FILE)
	@echo "- Use race detection during testing: \`make build-race\`" >> $(REPORT_FILE)
	@echo "" >> $(REPORT_FILE)
	@echo "For detailed analysis, see files in \`$(OUTPUT_DIR)/\`" >> $(REPORT_FILE)
	@echo "$(GREEN)[SUCCESS]$(NC) Comprehensive report generated: $(REPORT_FILE)"

# Performance comparison target
.PHONY: perf-compare
perf-compare: build-all
	@echo "$(CYAN)[PERFORMANCE]$(NC) Comparing build performance..."
	@echo "Running performance comparison..."
	@echo "Debug build:"
	@time ./$(BINARY_DIR)/$(APP_NAME)-debug --optimization inlining --size 1000 --iterations 100 2>/dev/null || true
	@echo "Optimized build:"
	@time ./$(BINARY_DIR)/$(APP_NAME)-optimized --optimization inlining --size 1000 --iterations 100 2>/dev/null || true
	@echo "Full optimization build:"
	@time ./$(BINARY_DIR)/$(APP_NAME)-full --optimization inlining --size 1000 --iterations 100 2>/dev/null || true

# Quick development cycle
.PHONY: dev
dev: fmt vet test build-default
	@echo "$(GREEN)[SUCCESS]$(NC) Development cycle completed"

# Production ready build
.PHONY: prod
prod: fmt vet test build-full
	@echo "$(GREEN)[SUCCESS]$(NC) Production build ready: $(BINARY_DIR)/$(APP_NAME)-full"

# CI/CD target
.PHONY: ci
ci: deps fmt vet test benchmark build-all analyze-all report
	@echo "$(GREEN)[SUCCESS]$(NC) CI/CD pipeline completed"

# Check if required tools are available
.PHONY: check-tools
check-tools:
	@echo "$(BLUE)[CHECK]$(NC) Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "$(RED)[ERROR]$(NC) Go is not installed"; exit 1; }
	@command -v make >/dev/null 2>&1 || { echo "$(RED)[ERROR]$(NC) Make is not installed"; exit 1; }
	@echo "$(GREEN)[SUCCESS]$(NC) All required tools are available"
	@echo "Go version: $$(go version)"
	@echo "Make version: $$(make --version | head -1)"

# Show project status
.PHONY: status
status:
	@echo "$(CYAN)Project Status$(NC)"
	@echo "=============="
	@echo "Go files: $$(find . -name '*.go' -not -path './vendor/*' | wc -l)"
	@echo "Total lines: $$(find . -name '*.go' -not -path './vendor/*' -exec wc -l {} + | tail -1 | awk '{print $$1}')"
	@if [ -d "$(BINARY_DIR)" ]; then echo "Binaries: $$(ls $(BINARY_DIR) 2>/dev/null | wc -l)"; fi
	@if [ -d "$(OUTPUT_DIR)" ]; then echo "Analysis files: $$(find $(OUTPUT_DIR) -type f 2>/dev/null | wc -l)"; fi
	@echo "Go version: $$(go version)"
	@echo "Module: $$(go list -m)"

# Install development tools
.PHONY: install-tools
install-tools:
	@echo "$(BLUE)[INSTALL]$(NC) Installing development tools..."
	@go install golang.org/x/lint/golint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/kisielk/errcheck@latest
	@echo "$(GREEN)[SUCCESS]$(NC) Development tools installed"

# Show available make targets
.PHONY: targets
targets:
	@echo "$(CYAN)Available Make Targets:$(NC)"
	@echo "======================"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "For detailed help, run: make help"