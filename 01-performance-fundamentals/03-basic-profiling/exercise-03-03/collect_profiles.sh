#!/bin/bash
# collect_profiles.sh

BASE_URL="http://localhost:8080"
OUTPUT_DIR="profiles/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$OUTPUT_DIR"

echo "Collecting profiles to $OUTPUT_DIR"

# Collect different profile types
curl -o "$OUTPUT_DIR/cpu.prof" "$BASE_URL/debug/pprof/profile?seconds=30"
curl -o "$OUTPUT_DIR/heap.prof" "$BASE_URL/debug/pprof/heap"
curl -o "$OUTPUT_DIR/allocs.prof" "$BASE_URL/debug/pprof/allocs"
curl -o "$OUTPUT_DIR/goroutine.prof" "$BASE_URL/debug/pprof/goroutine"

# Collect server stats
curl -s "$BASE_URL/stats" > "$OUTPUT_DIR/stats.json"

echo "Profiles collected in $OUTPUT_DIR"