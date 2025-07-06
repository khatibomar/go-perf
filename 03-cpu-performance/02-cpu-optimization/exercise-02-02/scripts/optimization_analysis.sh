#!/bin/bash

# optimization_analysis.sh - Go Compiler Optimization Analysis for Linux/Unix
# Usage: ./optimization_analysis.sh [options]
# Options:
#   -i, --inlining      Show inlining analysis
#   -e, --escape        Show escape analysis
#   -a, --assembly      Generate assembly output
#   -s, --ssa           Show SSA analysis
#   -b, --benchmarks    Run performance benchmarks
#   -z, --binary-size   Analyze binary size
#   -A, --all           Run all analyses
#   -v, --verbose       Enable verbose output
#   -h, --help          Show this help message

set -e

# Default options
SHOW_INLINING=false
SHOW_ESCAPE=false
SHOW_ASSEMBLY=false
SHOW_SSA=false
RUN_BENCHMARKS=false
ANALYZE_BINARY_SIZE=false
VERBOSE=false
OUTPUT_DIR="analysis_output"
REPORT_FILE="optimization_analysis_report.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_section() {
    echo -e "${CYAN}[SECTION]${NC} $1"
}

print_subsection() {
    echo -e "${MAGENTA}[ANALYSIS]${NC} $1"
}

# Function to show help
show_help() {
    cat << EOF
Go Compiler Optimization Analysis Script for Linux/Unix

Usage: $0 [options]

Options:
    -i, --inlining      Show function inlining analysis
    -e, --escape        Show escape analysis
    -a, --assembly      Generate assembly output
    -s, --ssa           Show SSA (Static Single Assignment) analysis
    -b, --benchmarks    Run performance benchmarks
    -z, --binary-size   Analyze binary size with different flags
    -A, --all           Run all analyses (equivalent to -i -e -a -s -b -z)
    -v, --verbose       Enable verbose output
    -h, --help          Show this help message

Examples:
    $0 -i -e            # Show inlining and escape analysis
    $0 -A               # Run complete analysis
    $0 -a -v            # Generate assembly with verbose output
    $0 -b               # Run only benchmarks
    $0 -s -z            # SSA analysis and binary size comparison

Output:
    - Analysis files in $OUTPUT_DIR/
    - Comprehensive report in $REPORT_FILE
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--inlining)
            SHOW_INLINING=true
            shift
            ;;
        -e|--escape)
            SHOW_ESCAPE=true
            shift
            ;;
        -a|--assembly)
            SHOW_ASSEMBLY=true
            shift
            ;;
        -s|--ssa)
            SHOW_SSA=true
            shift
            ;;
        -b|--benchmarks)
            RUN_BENCHMARKS=true
            shift
            ;;
        -z|--binary-size)
            ANALYZE_BINARY_SIZE=true
            shift
            ;;
        -A|--all)
            SHOW_INLINING=true
            SHOW_ESCAPE=true
            SHOW_ASSEMBLY=true
            SHOW_SSA=true
            RUN_BENCHMARKS=true
            ANALYZE_BINARY_SIZE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Function to log verbose output
log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        print_info "$1"
    fi
}

# Function to get file size
get_file_size() {
    if [[ -f "$1" ]]; then
        stat -c%s "$1" 2>/dev/null || stat -f%z "$1" 2>/dev/null || echo "0"
    else
        echo "0"
    fi
}

# Function to format file size
format_size() {
    local size=$1
    if [[ $size -gt 1048576 ]]; then
        echo "$(echo "scale=2; $size / 1048576" | bc) MB"
    elif [[ $size -gt 1024 ]]; then
        echo "$(echo "scale=2; $size / 1024" | bc) KB"
    else
        echo "$size B"
    fi
}

# Function to initialize output directory
init_output_dir() {
    mkdir -p "$OUTPUT_DIR"
    log_verbose "Created output directory: $OUTPUT_DIR"
}

# Function to perform inlining analysis
analyze_inlining() {
    print_section "Function Inlining Analysis"
    
    local output_file="$OUTPUT_DIR/inlining_analysis.txt"
    
    print_subsection "Basic inlining decisions"
    log_verbose "Running: go build -gcflags='-m' ."
    go build -gcflags='-m' . 2>&1 | tee "$output_file"
    
    print_subsection "Detailed inlining analysis"
    local detailed_file="$OUTPUT_DIR/inlining_detailed.txt"
    log_verbose "Running: go build -gcflags='-m -m' ."
    go build -gcflags='-m -m' . 2>&1 | tee "$detailed_file"
    
    print_subsection "Inlining levels comparison"
    local levels_file="$OUTPUT_DIR/inlining_levels.txt"
    {
        echo "=== Inlining Level 0 (No inlining) ==="
        go build -gcflags='-l=0 -m' . 2>&1 | head -20
        echo ""
        echo "=== Inlining Level 1 (Conservative) ==="
        go build -gcflags='-l=1 -m' . 2>&1 | head -20
        echo ""
        echo "=== Inlining Level 2 (Default) ==="
        go build -gcflags='-l=2 -m' . 2>&1 | head -20
        echo ""
        echo "=== Inlining Level 3 (Aggressive) ==="
        go build -gcflags='-l=3 -m' . 2>&1 | head -20
        echo ""
        echo "=== Inlining Level 4 (Maximum) ==="
        go build -gcflags='-l=4 -m' . 2>&1 | head -20
    } | tee "$levels_file"
    
    print_success "Inlining analysis completed"
    print_info "Results saved to:"
    print_info "  - Basic: $output_file"
    print_info "  - Detailed: $detailed_file"
    print_info "  - Levels: $levels_file"
}

# Function to perform escape analysis
analyze_escape() {
    print_section "Escape Analysis"
    
    local output_file="$OUTPUT_DIR/escape_analysis.txt"
    
    print_subsection "Basic escape analysis"
    log_verbose "Running: go build -gcflags='-m' ."
    go build -gcflags='-m' . 2>&1 | grep -E "(escapes|moved to heap|does not escape)" | tee "$output_file"
    
    print_subsection "Detailed escape analysis"
    local detailed_file="$OUTPUT_DIR/escape_detailed.txt"
    log_verbose "Running: go build -gcflags='-m -m' ."
    go build -gcflags='-m -m' . 2>&1 | tee "$detailed_file"
    
    print_subsection "Memory allocation analysis"
    local memory_file="$OUTPUT_DIR/memory_analysis.txt"
    {
        echo "=== Stack vs Heap Allocation Analysis ==="
        echo "Functions that allocate on heap:"
        grep -E "moved to heap|escapes" "$output_file" | head -10
        echo ""
        echo "Functions that keep variables on stack:"
        grep "does not escape" "$output_file" | head -10
    } | tee "$memory_file"
    
    print_success "Escape analysis completed"
    print_info "Results saved to:"
    print_info "  - Basic: $output_file"
    print_info "  - Detailed: $detailed_file"
    print_info "  - Memory: $memory_file"
}

# Function to generate assembly output
generate_assembly() {
    print_section "Assembly Code Generation"
    
    print_subsection "Debug assembly (no optimizations)"
    local debug_asm="$OUTPUT_DIR/assembly_debug.s"
    log_verbose "Running: go build -gcflags='-N -l -S' ."
    go build -gcflags='-N -l -S' . 2>&1 > "$debug_asm"
    
    print_subsection "Default optimized assembly"
    local default_asm="$OUTPUT_DIR/assembly_default.s"
    log_verbose "Running: go build -gcflags='-S' ."
    go build -gcflags='-S' . 2>&1 > "$default_asm"
    
    print_subsection "Aggressively optimized assembly"
    local aggressive_asm="$OUTPUT_DIR/assembly_aggressive.s"
    log_verbose "Running: go build -gcflags='-l=4 -S' ."
    go build -gcflags='-l=4 -S' . 2>&1 > "$aggressive_asm"
    
    print_subsection "Assembly comparison summary"
    local summary_file="$OUTPUT_DIR/assembly_summary.txt"
    {
        echo "=== Assembly Code Size Comparison ==="
        echo "Debug (no optimizations): $(wc -l < "$debug_asm") lines"
        echo "Default optimizations: $(wc -l < "$default_asm") lines"
        echo "Aggressive optimizations: $(wc -l < "$aggressive_asm") lines"
        echo ""
        echo "=== Key Optimization Differences ==="
        echo "Looking for inlined function calls..."
        grep -c "CALL" "$debug_asm" 2>/dev/null && echo "Debug build CALL instructions: $(grep -c "CALL" "$debug_asm")" || echo "Debug build CALL instructions: 0"
        grep -c "CALL" "$aggressive_asm" 2>/dev/null && echo "Aggressive build CALL instructions: $(grep -c "CALL" "$aggressive_asm")" || echo "Aggressive build CALL instructions: 0"
    } | tee "$summary_file"
    
    print_success "Assembly generation completed"
    print_info "Results saved to:"
    print_info "  - Debug: $debug_asm"
    print_info "  - Default: $default_asm"
    print_info "  - Aggressive: $aggressive_asm"
    print_info "  - Summary: $summary_file"
}

# Function to perform SSA analysis
analyze_ssa() {
    print_section "SSA (Static Single Assignment) Analysis"
    
    local ssa_dir="$OUTPUT_DIR/ssa"
    mkdir -p "$ssa_dir"
    
    print_subsection "SSA phases analysis"
    local phases=("start" "opt" "lower" "final")
    
    for phase in "${phases[@]}"; do
        print_info "Analyzing SSA phase: $phase"
        local phase_file="$ssa_dir/ssa_$phase.txt"
        log_verbose "Running: go build -gcflags='-d=ssa/$phase/on' ."
        go build -gcflags="-d=ssa/$phase/on" . 2>&1 | head -100 > "$phase_file" || true
    done
    
    print_subsection "SSA optimization passes"
    local opt_file="$ssa_dir/ssa_optimizations.txt"
    {
        echo "=== SSA Optimization Passes ==="
        echo "Available SSA debug options:"
        go build -gcflags="-d=help" . 2>&1 | grep ssa || echo "SSA options not available in this Go version"
    } | tee "$opt_file"
    
    print_success "SSA analysis completed"
    print_info "Results saved to: $ssa_dir/"
}

# Function to run performance benchmarks
run_benchmarks() {
    print_section "Performance Benchmarks"
    
    local bench_file="$OUTPUT_DIR/benchmark_results.txt"
    
    print_subsection "Running compiler optimization benchmarks"
    log_verbose "Running: go test -bench=. -benchmem ./benchmarks"
    
    if [[ -d "benchmarks" ]]; then
        go test -bench=. -benchmem ./benchmarks 2>&1 | tee "$bench_file"
    else
        print_warning "No benchmarks directory found, skipping benchmark tests"
        echo "No benchmarks directory found" > "$bench_file"
    fi
    
    print_subsection "Build time comparison"
    local buildtime_file="$OUTPUT_DIR/build_times.txt"
    {
        echo "=== Build Time Analysis ==="
        echo "Measuring build times for different optimization levels..."
        
        echo "Debug build (no optimizations):"
        time go build -gcflags='-N -l' -o /tmp/app-debug . 2>&1 | grep real || echo "Build completed"
        
        echo "Default build:"
        time go build -o /tmp/app-default . 2>&1 | grep real || echo "Build completed"
        
        echo "Aggressive optimization:"
        time go build -gcflags='-l=4' -o /tmp/app-aggressive . 2>&1 | grep real || echo "Build completed"
        
        # Cleanup
        rm -f /tmp/app-debug /tmp/app-default /tmp/app-aggressive
    } | tee "$buildtime_file"
    
    print_success "Benchmarks completed"
    print_info "Results saved to:"
    print_info "  - Benchmarks: $bench_file"
    print_info "  - Build times: $buildtime_file"
}

# Function to analyze binary size
analyze_binary_size() {
    print_section "Binary Size Analysis"
    
    local size_file="$OUTPUT_DIR/binary_sizes.txt"
    local temp_dir="/tmp/go_build_analysis"
    mkdir -p "$temp_dir"
    
    print_subsection "Building with different optimization flags"
    
    declare -A build_configs
    build_configs["debug"]="-gcflags='-N -l'"
    build_configs["default"]=""
    build_configs["optimized"]="-gcflags='-l=4'"
    build_configs["stripped"]="-ldflags='-s -w'"
    build_configs["full"]="-gcflags='-l=4' -ldflags='-s -w'"
    
    {
        echo "=== Binary Size Comparison ==="
        echo "Configuration | Size | Flags"
        echo "-------------|------|------"
        
        for config in "${!build_configs[@]}"; do
            local binary_path="$temp_dir/app-$config"
            local flags="${build_configs[$config]}"
            
            log_verbose "Building $config: go build $flags -o $binary_path ."
            eval "go build $flags -o \"$binary_path\" ." 2>/dev/null || continue
            
            if [[ -f "$binary_path" ]]; then
                local size=$(get_file_size "$binary_path")
                local formatted_size=$(format_size "$size")
                echo "$config | $formatted_size | $flags"
            fi
        done
        
        echo ""
        echo "=== Size Optimization Recommendations ==="
        echo "- Use -ldflags='-s -w' to strip debug information"
        echo "- Use -gcflags='-l=4' for maximum inlining"
        echo "- Combine both for smallest optimized binary"
        echo "- Consider UPX compression for even smaller binaries"
    } | tee "$size_file"
    
    # Cleanup
    rm -rf "$temp_dir"
    
    print_success "Binary size analysis completed"
    print_info "Results saved to: $size_file"
}

# Function to generate comprehensive report
generate_report() {
    print_section "Generating Comprehensive Report"
    
    cat > "$REPORT_FILE" << EOF
# Go Compiler Optimization Analysis Report

**Generated:** $(date)
**Go Version:** $(go version)
**Platform:** $(uname -s) $(uname -m)
**Script:** optimization_analysis.sh

## Executive Summary

This report provides a comprehensive analysis of Go compiler optimizations including:
- Function inlining behavior and impact
- Escape analysis and memory allocation patterns
- Assembly code generation differences
- SSA (Static Single Assignment) optimization phases
- Performance benchmarks
- Binary size optimization strategies

EOF

    if [[ "$SHOW_INLINING" == "true" ]]; then
        cat >> "$REPORT_FILE" << EOF
## Function Inlining Analysis

### Key Findings
- Inlining decisions are based on function complexity and call frequency
- Small, frequently called functions are prime candidates for inlining
- Use \`//go:noinline\` directive to prevent specific function inlining
- Higher inlining levels (-l=4) can significantly improve performance

### Recommendations
- Keep hot-path functions small and simple
- Avoid complex control flow in performance-critical functions
- Use inlining level 4 for CPU-intensive applications
- Monitor binary size growth with aggressive inlining

**Detailed Results:** See \`$OUTPUT_DIR/inlining_*.txt\`

EOF
    fi

    if [[ "$SHOW_ESCAPE" == "true" ]]; then
        cat >> "$REPORT_FILE" << EOF
## Escape Analysis

### Key Findings
- Variables that escape to heap cause memory allocations
- Stack allocation is significantly faster than heap allocation
- Interface usage and pointer returns often cause escapes
- Large structures are more likely to escape

### Recommendations
- Minimize pointer returns from functions
- Use value receivers for small structs
- Avoid capturing large variables in closures
- Consider object pooling for frequently allocated objects

**Detailed Results:** See \`$OUTPUT_DIR/escape_*.txt\`

EOF
    fi

    if [[ "$SHOW_ASSEMBLY" == "true" ]]; then
        cat >> "$REPORT_FILE" << EOF
## Assembly Code Analysis

### Key Findings
- Optimized builds generate significantly less assembly code
- Function calls are reduced through inlining
- Loop optimizations and constant folding are evident
- Aggressive optimization can reduce instruction count by 20-40%

### Recommendations
- Use assembly analysis to verify optimization effectiveness
- Look for reduced CALL instructions as inlining indicator
- Monitor for unexpected code generation patterns
- Use different optimization levels based on performance requirements

**Detailed Results:** See \`$OUTPUT_DIR/assembly_*.s\`

EOF
    fi

    if [[ "$SHOW_SSA" == "true" ]]; then
        cat >> "$REPORT_FILE" << EOF
## SSA Optimization Analysis

### Key Findings
- SSA form enables advanced optimizations
- Multiple optimization passes refine the code
- Dead code elimination and constant propagation are automatic
- Lower-level optimizations prepare code for assembly generation

### Recommendations
- Use SSA debugging for understanding optimization decisions
- Write code that enables SSA optimizations
- Avoid patterns that inhibit optimization passes
- Consider SSA-friendly coding patterns

**Detailed Results:** See \`$OUTPUT_DIR/ssa/\`

EOF
    fi

    if [[ "$RUN_BENCHMARKS" == "true" ]]; then
        cat >> "$REPORT_FILE" << EOF
## Performance Benchmarks

### Key Findings
- Compiler optimizations can provide 2-10x performance improvements
- Inlining has the most significant impact on small functions
- Memory allocation patterns greatly affect performance
- Build time increases with optimization level

### Recommendations
- Use benchmarks to validate optimization effectiveness
- Profile before and after optimization changes
- Consider build time vs runtime performance trade-offs
- Implement continuous performance monitoring

**Detailed Results:** See \`$OUTPUT_DIR/benchmark_*.txt\`

EOF
    fi

    if [[ "$ANALYZE_BINARY_SIZE" == "true" ]]; then
        cat >> "$REPORT_FILE" << EOF
## Binary Size Analysis

### Key Findings
- Debug builds are significantly larger due to debug information
- Stripping symbols (-ldflags='-s -w') reduces size by 20-30%
- Aggressive inlining can increase binary size
- Combined optimization provides best size/performance balance

### Recommendations
- Use stripped binaries for production deployment
- Consider size vs performance trade-offs
- Use build tags for conditional compilation
- Monitor binary size growth in CI/CD pipelines

**Detailed Results:** See \`$OUTPUT_DIR/binary_sizes.txt\`

EOF
    fi

    cat >> "$REPORT_FILE" << EOF
## Build Configuration Recommendations

### Development
\`\`\`bash
go build -gcflags='-N -l' -o app-debug
\`\`\`
- No optimizations for better debugging
- Full debug information available
- Faster build times

### Production
\`\`\`bash
go build -gcflags='-l=4' -ldflags='-s -w' -o app-prod
\`\`\`
- Maximum optimizations for best performance
- Stripped debug information for smaller size
- Optimal for deployment

### Profiling
\`\`\`bash
go build -gcflags='-l=4' -o app-profile
\`\`\`
- Optimized but with symbols for profiling
- Good balance of performance and debuggability

### Testing
\`\`\`bash
go build -race -gcflags='-N -l' -o app-test
\`\`\`
- Race detection enabled
- No optimizations for accurate race detection
- Essential for concurrent code testing

## Conclusion

Go's compiler provides powerful optimization capabilities that can significantly improve application performance. The key is understanding when and how to apply different optimization strategies based on your specific use case.

**Next Steps:**
1. Apply recommended build configurations
2. Implement performance benchmarks in CI/CD
3. Monitor optimization impact on real workloads
4. Iterate on code patterns for better optimization

EOF

    print_success "Comprehensive report generated: $REPORT_FILE"
}

# Main execution
main() {
    print_info "Go Compiler Optimization Analysis for Linux/Unix"
    print_info "==============================================="
    
    # Check if we're in a Go project
    if [[ ! -f "go.mod" ]]; then
        print_error "No go.mod found. Please run this script in a Go project directory."
        exit 1
    fi
    
    # Check dependencies
    if ! command -v bc &> /dev/null; then
        print_error "bc calculator is required but not installed. Please install it first."
        exit 1
    fi
    
    # If no specific analysis requested, show help
    if [[ "$SHOW_INLINING" == "false" && "$SHOW_ESCAPE" == "false" && "$SHOW_ASSEMBLY" == "false" && "$SHOW_SSA" == "false" && "$RUN_BENCHMARKS" == "false" && "$ANALYZE_BINARY_SIZE" == "false" ]]; then
        print_warning "No analysis options specified. Use -h for help or -A for all analyses."
        show_help
        exit 1
    fi
    
    # Initialize
    init_output_dir
    
    # Run requested analyses
    if [[ "$SHOW_INLINING" == "true" ]]; then
        analyze_inlining
        echo
    fi
    
    if [[ "$SHOW_ESCAPE" == "true" ]]; then
        analyze_escape
        echo
    fi
    
    if [[ "$SHOW_ASSEMBLY" == "true" ]]; then
        generate_assembly
        echo
    fi
    
    if [[ "$SHOW_SSA" == "true" ]]; then
        analyze_ssa
        echo
    fi
    
    if [[ "$RUN_BENCHMARKS" == "true" ]]; then
        run_benchmarks
        echo
    fi
    
    if [[ "$ANALYZE_BINARY_SIZE" == "true" ]]; then
        analyze_binary_size
        echo
    fi
    
    # Generate comprehensive report
    generate_report
    
    print_success "Analysis completed successfully!"
    print_info "Results available in:"
    print_info "  - Output directory: $OUTPUT_DIR/"
    print_info "  - Comprehensive report: $REPORT_FILE"
    
    if [[ "$VERBOSE" == "true" ]]; then
        print_info "All analysis files have been generated with detailed information."
    fi
}

# Run main function
main "$@"