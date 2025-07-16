#!/bin/bash
# monitor.sh

while true; do
    STATS=$(curl -s "http://localhost:8080/stats")
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
    MEMORY=$(echo "$STATS" | jq -r '.memory_stats.alloc_mb')
    GOROUTINES=$(echo "$STATS" | jq -r '.goroutines')
    REQUESTS=$(echo "$STATS" | jq -r '.requests')
    
    echo "$TIMESTAMP - Memory: ${MEMORY}MB, Goroutines: $GOROUTINES, Requests: $REQUESTS"
    sleep 10
done