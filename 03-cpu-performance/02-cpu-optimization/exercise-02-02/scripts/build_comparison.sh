#!/bin/bash

# build_comparison.sh - Compare Go build configurations on Linux/Unix
# Usage: ./build_comparison.sh [options]
# Options:
#   -c, --clean         Clean before building
#   -b, --benchmarks    Run performance benchmarks
#   -v, --verbose       Enable verbose output
#   -h, --help          Show this help message

set -e

# Default options
CLEAN_FIRST=false
RUN_BENCHMARKS=false
VERBOSE=false
OUTPUT_DIR="build_output"
REPORT_FILE="build_comparison_report.md"
JSON_FILE="build_results.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Function to show help
show_help() {
    cat << EOF
Go Build Comparison Script for Linux/Unix

Usage: $0 [options]

Options:
    -c, --clean         Clean build artifacts before building
    -b, --benchmarks    Run performance benchmarks after building
    -v, --verbose       Enable verbose output
    -h, --help          Show this help message

Examples:
    $0                  # Basic build comparison
    $0 -c -v            # Clean build with verbose output
    $0 -b               # Build and run benchmarks
    $0 -c -b -v         # Full analysis with all options

Output:
    - Binaries in $OUTPUT_DIR/
    - Report in $REPORT_FILE
    - JSON results in $JSON_FILE
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--clean)
            CLEAN_FIRST=true
            shift
            ;;
        -b|--benchmarks)
            RUN_BENCHMARKS=true
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

# Function to get file size in bytes
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

# Function to measure build time
measure_build_time() {
    local config_name="$1"
    local build_flags="$2"
    local output_file="$3"
    
    log_verbose "Building $config_name configuration..."
    
    local start_time=$(date +%s.%N)
    
    if [[ "$VERBOSE" == "true" ]]; then
        eval "go build $build_flags -o \"$output_file\" ." 2>&1
    else
        eval "go build $build_flags -o \"$output_file\" ." 2>/dev/null
    fi
    
    local end_time=$(date +%s.%N)
    local build_time=$(echo "$end_time - $start_time" | bc)
    
    echo "$build_time"
}

# Function to run performance benchmark
run_performance_test() {
    local binary_path="$1"
    local config_name="$2"
    
    if [[ ! -f "$binary_path" ]]; then
        print_warning "Binary not found: $binary_path"
        return 1
    fi
    
    log_verbose "Running performance test for $config_name..."
    
    local start_time=$(date +%s.%N)
    "$binary_path" --optimization inlining --size 10000 --iterations 100 > /dev/null 2>&1
    local end_time=$(date +%s.%N)
    
    local execution_time=$(echo "$end_time - $start_time" | bc)
    echo "$execution_time"
}

# Function to initialize output directory
init_output_dir() {
    if [[ "$CLEAN_FIRST" == "true" ]]; then
        print_info "Cleaning previous build artifacts..."
        rm -rf "$OUTPUT_DIR"
        rm -f "$REPORT_FILE" "$JSON_FILE"
    fi
    
    mkdir -p "$OUTPUT_DIR"
}

# Function to generate JSON results
generate_json_results() {
    local json_content="{"
    json_content+="\"timestamp\": \"$(date -Iseconds)\","
    json_content+="\"go_version\": \"$(go version | cut -d' ' -f3)\","
    json_content+="\"platform\": \"$(uname -s)-$(uname -m)\","
    json_content+="\"configurations\": ["
    
    local first=true
    for config in "${!BUILD_CONFIGS[@]}"; do
        if [[ "$first" != "true" ]]; then
            json_content+=","
        fi
        first=false
        
        local binary_path="$OUTPUT_DIR/app-$config"
        local size=$(get_file_size "$binary_path")
        
        json_content+="{"
        json_content+="\"name\": \"$config\","
        json_content+="\"flags\": \"${BUILD_CONFIGS[$config]}\","
        json_content+="\"binary_size\": $size,"
        json_content+="\"build_time\": ${BUILD_TIMES[$config]:-0},"
        if [[ "$RUN_BENCHMARKS" == "true" ]]; then
            json_content+="\"execution_time\": ${EXECUTION_TIMES[$config]:-0}"
        else
            json_content+="\"execution_time\": null"
        fi
        json_content+="}"
    done
    
    json_content+="]}"
    
    echo "$json_content" > "$JSON_FILE"
}

# Function to generate markdown report
generate_report() {
    cat > "$REPORT_FILE" << EOF
# Go Build Comparison Report

**Generated:** $(date)
**Go Version:** $(go version)
**Platform:** $(uname -s) $(uname -m)
**Script:** build_comparison.sh

## Build Configurations

| Configuration | Build Flags | Binary Size | Build Time | Performance |
|---------------|-------------|-------------|------------|-----------|
EOF

    for config in "${!BUILD_CONFIGS[@]}"; do
        local binary_path="$OUTPUT_DIR/app-$config"
        local size=$(get_file_size "$binary_path")
        local formatted_size=$(format_size "$size")
        local build_time="${BUILD_TIMES[$config]:-N/A}s"
        
        local performance="N/A"
        if [[ "$RUN_BENCHMARKS" == "true" && -n "${EXECUTION_TIMES[$config]}" ]]; then
            performance="${EXECUTION_TIMES[$config]}s"
        fi
        
        echo "| $config | \`${BUILD_CONFIGS[$config]}\` | $formatted_size | $build_time | $performance |" >> "$REPORT_FILE"
    done

    cat >> "$REPORT_FILE" << EOF

## Configuration Details

### Debug Build
- **Purpose:** Development and debugging
- **Flags:** \`-gcflags='-N -l'\`
- **Characteristics:** No optimizations, full debug info
- **Use Case:** Local development, debugging

### Default Build
- **Purpose:** Standard production build
- **Flags:** (none)
- **Characteristics:** Standard Go optimizations
- **Use Case:** General production deployment

### Aggressive Optimization
- **Purpose:** Maximum performance
- **Flags:** \`-gcflags='-l=4'\`
- **Characteristics:** Maximum inlining
- **Use Case:** CPU-intensive applications

### Size Optimized
- **Purpose:** Minimal binary size
- **Flags:** \`-ldflags='-s -w'\`
- **Characteristics:** Stripped symbols and debug info
- **Use Case:** Container deployments, embedded systems

### Production Build
- **Purpose:** Optimized production deployment
- **Flags:** \`-gcflags='-l=4' -ldflags='-s -w'\`
- **Characteristics:** Maximum performance + minimal size
- **Use Case:** Production servers

### Race Detection
- **Purpose:** Concurrency testing
- **Flags:** \`-race -gcflags='-N -l'\`
- **Characteristics:** Race condition detection
- **Use Case:** Testing concurrent code

## Recommendations

- **Development:** Use debug build for local development
- **Testing:** Use race detection for concurrent code testing
- **Production:** Use production build for deployment
- **Containers:** Use size-optimized build for Docker images
- **Performance-Critical:** Use aggressive optimization

## Usage Examples

\`\`\`bash
# Development build
go build -gcflags='-N -l' -o app-debug

# Production build
go build -gcflags='-l=4' -ldflags='-s -w' -o app-prod

# Size-optimized build
go build -ldflags='-s -w' -o app-small

# Race detection build
go build -race -gcflags='-N -l' -o app-race
\`\`\`
EOF

    print_success "Report generated: $REPORT_FILE"
}

# Main execution
main() {
    print_info "Go Build Comparison Script for Linux/Unix"
    print_info "========================================"
    
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
    
    # Initialize
    init_output_dir
    
    # Define build configurations
    declare -A BUILD_CONFIGS
    BUILD_CONFIGS["debug"]="-gcflags='-N -l'"
    BUILD_CONFIGS["default"]=""
    BUILD_CONFIGS["aggressive"]="-gcflags='-l=4'"
    BUILD_CONFIGS["size"]="-ldflags='-s -w'"
    BUILD_CONFIGS["production"]="-gcflags='-l=4' -ldflags='-s -w'"
    BUILD_CONFIGS["race"]="-race -gcflags='-N -l'"
    
    declare -A BUILD_TIMES
    declare -A EXECUTION_TIMES
    
    # Build each configuration
    print_info "Building configurations..."
    for config in "${!BUILD_CONFIGS[@]}"; do
        local binary_path="$OUTPUT_DIR/app-$config"
        local build_time=$(measure_build_time "$config" "${BUILD_CONFIGS[$config]}" "$binary_path")
        BUILD_TIMES[$config]="$build_time"
        
        if [[ -f "$binary_path" ]]; then
            local size=$(get_file_size "$binary_path")
            local formatted_size=$(format_size "$size")
            print_success "$config: $formatted_size (${build_time}s)"
        else
            print_error "Failed to build $config configuration"
        fi
    done
    
    # Run performance benchmarks if requested
    if [[ "$RUN_BENCHMARKS" == "true" ]]; then
        print_info "Running performance benchmarks..."
        for config in "${!BUILD_CONFIGS[@]}"; do
            local binary_path="$OUTPUT_DIR/app-$config"
            if [[ -f "$binary_path" ]]; then
                local exec_time=$(run_performance_test "$binary_path" "$config")
                EXECUTION_TIMES[$config]="$exec_time"
                print_success "$config performance: ${exec_time}s"
            fi
        done
    fi
    
    # Generate outputs
    generate_json_results
    generate_report
    
    print_success "Build comparison completed!"
    print_info "Results saved to:"
    print_info "  - Report: $REPORT_FILE"
    print_info "  - JSON: $JSON_FILE"
    print_info "  - Binaries: $OUTPUT_DIR/"
    
    if [[ "$RUN_BENCHMARKS" == "true" ]]; then
        print_info "Performance benchmarks included in results"
    fi
}

# Run main function
main "$@"