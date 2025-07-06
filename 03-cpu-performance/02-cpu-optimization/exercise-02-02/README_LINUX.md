# Go Compiler Optimization Techniques - Linux/Unix Guide

This guide provides comprehensive instructions for running the Go compiler optimization exercises on Linux and Unix systems.

## Prerequisites

### Required Tools
- **Go 1.19+**: Install from [golang.org](https://golang.org/dl/)
- **Make**: Usually pre-installed on most Linux distributions
- **bc**: Basic calculator for shell scripts
- **Bash**: Version 4.0+ recommended

### Installation Commands

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install golang-go make bc
```

#### CentOS/RHEL/Fedora
```bash
# Fedora
sudo dnf install golang make bc

# CentOS/RHEL
sudo yum install golang make bc
```

#### Arch Linux
```bash
sudo pacman -S go make bc
```

#### macOS
```bash
brew install go make bc
```

## Quick Start

### 1. Make Scripts Executable
```bash
chmod +x scripts/*.sh
```

### 2. Verify Installation
```bash
make check-tools
```

### 3. Run Complete Analysis
```bash
# Using Makefile (recommended)
make ci

# Or using shell script
./scripts/optimization_analysis.sh -A -v
```

## Available Tools

### 1. Makefile (`make`)

The Makefile provides a comprehensive build and analysis system:

```bash
# Show all available targets
make help

# Development workflow
make dev              # Format, vet, test, and build
make prod             # Production-ready build

# Build variants
make build-all        # All optimization levels
make build-debug      # Debug build (no optimizations)
make build-optimized  # Aggressive optimizations
make build-full       # Full optimization + stripped

# Analysis
make analyze-all      # Complete compiler analysis
make benchmark        # Performance benchmarks
make report           # Generate comprehensive report

# Utilities
make clean            # Clean build artifacts
make status           # Show project status
```

### 2. Optimization Analysis Script

The `optimization_analysis.sh` script provides detailed compiler analysis:

```bash
# Complete analysis
./scripts/optimization_analysis.sh -A -v

# Specific analyses
./scripts/optimization_analysis.sh -i    # Inlining analysis
./scripts/optimization_analysis.sh -e    # Escape analysis
./scripts/optimization_analysis.sh -a    # Assembly generation
./scripts/optimization_analysis.sh -s    # SSA analysis
./scripts/optimization_analysis.sh -b    # Benchmarks
./scripts/optimization_analysis.sh -z    # Binary size analysis

# Combined analyses
./scripts/optimization_analysis.sh -i -e -v  # Inlining + escape with verbose
```

### 3. Build Comparison Script

The `build_comparison.sh` script compares different build configurations:

```bash
# Run build comparison
./scripts/build_comparison.sh

# With specific options
./scripts/build_comparison.sh --verbose --report
```

## Detailed Usage Examples

### Example 1: Development Workflow

```bash
# 1. Set up the environment
make check-tools
make deps

# 2. Development cycle
make fmt              # Format code
make vet              # Static analysis
make test             # Run tests
make build-debug      # Build for debugging

# 3. Test the application
./bin/exercise-02-02-debug --help
./bin/exercise-02-02-debug --optimization inlining --size 1000 --iterations 100
```

### Example 2: Performance Analysis

```bash
# 1. Build all variants
make build-all

# 2. Run comprehensive analysis
./scripts/optimization_analysis.sh -A -v

# 3. Compare performance
make perf-compare

# 4. Generate report
make report

# 5. View results
cat optimization_analysis_report.md
ls -la analysis_output/
```

### Example 3: Production Deployment

```bash
# 1. Complete CI/CD pipeline
make ci

# 2. Build production binary
make prod

# 3. Verify the binary
./bin/exercise-02-02-full --help
file ./bin/exercise-02-02-full
ls -lh ./bin/exercise-02-02-full
```

### Example 4: Specific Optimization Analysis

```bash
# Analyze function inlining
./scripts/optimization_analysis.sh -i -v
cat analysis_output/inlining_analysis.txt

# Analyze escape analysis
./scripts/optimization_analysis.sh -e -v
cat analysis_output/escape_analysis.txt

# Generate assembly code
./scripts/optimization_analysis.sh -a -v
less analysis_output/assembly_debug.s
less analysis_output/assembly_default.s
```

## Understanding the Output

### 1. Build Variants

| Variant | Flags | Use Case |
|---------|-------|----------|
| `debug` | `-gcflags='-N -l'` | Development, debugging |
| `default` | (none) | Standard build |
| `optimized` | `-gcflags='-l=4'` | Maximum performance |
| `stripped` | `-ldflags='-s -w'` | Smaller binary size |
| `full` | `-gcflags='-l=4' -ldflags='-s -w'` | Production deployment |
| `race` | `-race` | Concurrency testing |

### 2. Analysis Files

```
analysis_output/
├── inlining_analysis.txt      # Function inlining decisions
├── inlining_detailed.txt      # Detailed inlining analysis
├── inlining_levels.txt        # Comparison of inlining levels
├── escape_analysis.txt        # Escape analysis results
├── escape_detailed.txt        # Detailed escape analysis
├── memory_analysis.txt        # Stack vs heap allocation
├── assembly_debug.s           # Debug assembly code
├── assembly_default.s         # Optimized assembly code
├── assembly_aggressive.s      # Aggressively optimized assembly
├── assembly_summary.txt       # Assembly comparison summary
├── ssa/                       # SSA analysis files
│   ├── ssa_start.txt
│   ├── ssa_opt.txt
│   └── ssa_optimizations.txt
├── benchmark_results.txt      # Performance benchmarks
├── build_times.txt           # Build time comparison
└── binary_sizes.txt          # Binary size analysis
```

### 3. Performance Metrics

The tools measure several key metrics:

- **Execution Time**: Function call overhead and optimization impact
- **Memory Allocations**: Heap vs stack allocation patterns
- **Binary Size**: Impact of different optimization levels
- **Build Time**: Compilation time for different configurations
- **Inlining Decisions**: Which functions are inlined and why
- **Escape Analysis**: Which variables escape to the heap

## Troubleshooting

### Common Issues

#### 1. Permission Denied
```bash
# Make scripts executable
chmod +x scripts/*.sh
```

#### 2. Missing Dependencies
```bash
# Install missing tools
sudo apt install bc  # Ubuntu/Debian
sudo dnf install bc  # Fedora
```

#### 3. Go Module Issues
```bash
# Reset Go modules
go mod tidy
go mod download
```

#### 4. Build Failures
```bash
# Clean and rebuild
make clean-all
make deps
make build-default
```

### Debugging Tips

1. **Use Verbose Mode**: Add `-v` flag to scripts for detailed output
2. **Check Go Version**: Ensure Go 1.19+ is installed
3. **Verify File Permissions**: Scripts must be executable
4. **Check Dependencies**: Run `make check-tools` to verify setup

## Advanced Usage

### Custom Build Flags

```bash
# Custom optimization level
go build -gcflags='-l=2' -o custom-app .

# With custom linker flags
go build -gcflags='-l=4' -ldflags='-s -w -X main.version=1.0' -o app .

# Race detection with optimizations
go build -race -gcflags='-l=2' -o app-race .
```

### Profiling Integration

```bash
# CPU profiling
make benchmark-cpu
go tool pprof cpu.prof

# Memory profiling
make benchmark-mem
go tool pprof mem.prof

# Trace analysis
go test -trace=trace.out ./benchmarks
go tool trace trace.out
```

### Continuous Integration

```bash
# Complete CI pipeline
make ci

# Custom CI workflow
make deps fmt vet test build-all analyze-all
```

## Performance Tips

### 1. Optimization Guidelines

- Use `-gcflags='-l=4'` for CPU-intensive applications
- Use `-ldflags='-s -w'` to reduce binary size
- Enable race detection during testing: `-race`
- Profile before optimizing: `go test -bench=. -cpuprofile=cpu.prof`

### 2. Code Patterns for Better Optimization

```go
// Good: Small, simple functions inline well
func add(a, b int) int {
    return a + b
}

// Good: Avoid interface{} in hot paths
func processInts(data []int) int {
    // Direct type, no boxing
}

// Good: Use value receivers for small structs
func (p Point) Distance() float64 {
    return math.Sqrt(p.X*p.X + p.Y*p.Y)
}
```

### 3. Monitoring Optimization Impact

```bash
# Before optimization
./scripts/optimization_analysis.sh -b > before.txt

# After code changes
./scripts/optimization_analysis.sh -b > after.txt

# Compare results
diff before.txt after.txt
```

## Integration with IDEs

### VS Code

Add to `.vscode/tasks.json`:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Go: Analyze Optimizations",
            "type": "shell",
            "command": "./scripts/optimization_analysis.sh",
            "args": ["-A", "-v"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "Go: Build All Variants",
            "type": "shell",
            "command": "make",
            "args": ["build-all"],
            "group": "build"
        }
    ]
}
```

### Vim/Neovim

Add to your configuration:

```vim
" Quick optimization analysis
nnoremap <leader>oa :!./scripts/optimization_analysis.sh -A -v<CR>

" Build all variants
nnoremap <leader>ba :!make build-all<CR>

" Run benchmarks
nnoremap <leader>bm :!make benchmark<CR>
```

## Contributing

To contribute improvements to the Linux/Unix tools:

1. Test on multiple distributions
2. Ensure POSIX compliance where possible
3. Add appropriate error handling
4. Update documentation
5. Test with different Go versions

## Resources

- [Go Compiler Optimization Guide](https://golang.org/doc/gc-guide)
- [Go Build Modes](https://golang.org/cmd/go/#hdr-Build_modes)
- [Go Performance Tips](https://github.com/golang/go/wiki/Performance)
- [Escape Analysis in Go](https://www.ardanlabs.com/blog/2017/05/language-mechanics-on-escape-analysis.html)

## License

This project is part of the Go Performance Optimization course materials.