#!/bin/bash

# Profile-Guided Optimization Script
# This script helps automate the profiling and optimization workflow

set -e

PROFILE_DIR="./profiles"
BEFORE_DIR="$PROFILE_DIR/before"
AFTER_DIR="$PROFILE_DIR/after"
REPORTS_DIR="./reports"

# Create directories
mkdir -p "$BEFORE_DIR" "$AFTER_DIR" "$REPORTS_DIR"

echo "=== Profile-Guided Optimization Workflow ==="
echo

# Function to run profiling
run_profile() {
    local name=$1
    local output_dir=$2
    local optimized=$3
    local workload=${4:-"mixed"}
    local size=${5:-10000}
    local iterations=${6:-100}
    
    echo "Running $name profile..."
    
    # CPU Profile
    go run . \
        -workload="$workload" \
        -size="$size" \
        -iterations="$iterations" \
        -optimized="$optimized" \
        -cpuprofile="$output_dir/cpu.prof" \
        -memprofile="$output_dir/mem.prof"
    
    echo "$name profile completed: $output_dir"
}

# Function to analyze profile
analyze_profile() {
    local name=$1
    local profile_path=$2
    local report_path=$3
    
    echo "Analyzing $name profile..."
    
    # Generate top functions report
    go tool pprof -top -cum "$profile_path" > "$report_path/top.txt" 2>/dev/null || true
    
    # Generate call graph
    go tool pprof -svg "$profile_path" > "$report_path/callgraph.svg" 2>/dev/null || true
    
    # Generate flame graph (if available)
    go tool pprof -http=:0 "$profile_path" &
    PPROF_PID=$!
    sleep 2
    kill $PPROF_PID 2>/dev/null || true
    
    echo "$name analysis completed: $report_path"
}

# Function to compare profiles
compare_profiles() {
    echo "Comparing profiles..."
    
    # Compare CPU profiles
    go tool pprof -top -diff_base="$BEFORE_DIR/cpu.prof" "$AFTER_DIR/cpu.prof" > "$REPORTS_DIR/comparison.txt" 2>/dev/null || true
    
    # Generate comparison report
    cat > "$REPORTS_DIR/summary.md" << EOF
# Profile Comparison Summary

## Before Optimization
\`\`\`
$(head -20 "$REPORTS_DIR/before/top.txt" 2>/dev/null || echo "Profile not found")
\`\`\`

## After Optimization
\`\`\`
$(head -20 "$REPORTS_DIR/after/top.txt" 2>/dev/null || echo "Profile not found")
\`\`\`

## Comparison
\`\`\`
$(head -20 "$REPORTS_DIR/comparison.txt" 2>/dev/null || echo "Comparison not available")
\`\`\`

## Analysis

- Review the top CPU-consuming functions
- Identify functions that improved after optimization
- Look for new bottlenecks that may have emerged
- Calculate overall performance improvement

EOF
    
    echo "Profile comparison completed: $REPORTS_DIR/summary.md"
}

# Main workflow
case "${1:-all}" in
    "before")
        echo "Step 1: Profiling BEFORE optimization"
        mkdir -p "$REPORTS_DIR/before"
        run_profile "BEFORE" "$BEFORE_DIR" "false"
        analyze_profile "BEFORE" "$BEFORE_DIR/cpu.prof" "$REPORTS_DIR/before"
        ;;
    
    "after")
        echo "Step 2: Profiling AFTER optimization"
        mkdir -p "$REPORTS_DIR/after"
        run_profile "AFTER" "$AFTER_DIR" "true"
        analyze_profile "AFTER" "$AFTER_DIR/cpu.prof" "$REPORTS_DIR/after"
        ;;
    
    "compare")
        echo "Step 3: Comparing profiles"
        compare_profiles
        ;;
    
    "all")
        echo "Running complete workflow..."
        echo
        
        # Step 1: Before optimization
        echo "Step 1: Profiling BEFORE optimization"
        mkdir -p "$REPORTS_DIR/before"
        run_profile "BEFORE" "$BEFORE_DIR" "false"
        analyze_profile "BEFORE" "$BEFORE_DIR/cpu.prof" "$REPORTS_DIR/before"
        echo
        
        # Step 2: After optimization
        echo "Step 2: Profiling AFTER optimization"
        mkdir -p "$REPORTS_DIR/after"
        run_profile "AFTER" "$AFTER_DIR" "true"
        analyze_profile "AFTER" "$AFTER_DIR/cpu.prof" "$REPORTS_DIR/after"
        echo
        
        # Step 3: Compare
        echo "Step 3: Comparing profiles"
        compare_profiles
        echo
        
        echo "=== Workflow Complete ==="
        echo "Results available in: $REPORTS_DIR/"
        echo "- Before: $REPORTS_DIR/before/"
        echo "- After: $REPORTS_DIR/after/"
        echo "- Summary: $REPORTS_DIR/summary.md"
        ;;
    
    "clean")
        echo "Cleaning up profiles and reports..."
        rm -rf "$PROFILE_DIR" "$REPORTS_DIR"
        echo "Cleanup completed"
        ;;
    
    "help")
        echo "Usage: $0 [command]"
        echo
        echo "Commands:"
        echo "  before   - Profile before optimization"
        echo "  after    - Profile after optimization"
        echo "  compare  - Compare before/after profiles"
        echo "  all      - Run complete workflow (default)"
        echo "  clean    - Clean up generated files"
        echo "  help     - Show this help"
        echo
        echo "Examples:"
        echo "  $0                    # Run complete workflow"
        echo "  $0 before            # Profile unoptimized code"
        echo "  $0 after             # Profile optimized code"
        echo "  $0 compare           # Compare profiles"
        ;;
    
    *)
        echo "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac