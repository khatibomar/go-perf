# Go pprof Cheat Sheet

Quick reference for Go's built-in profiling tool.

## Basic Usage

### CPU Profiling
```bash
# Enable CPU profiling in code
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

# Collect CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# From file
go tool pprof cpu.prof

# With binary
go tool pprof ./myapp cpu.prof
```

### Memory Profiling
```bash
# Heap profile (current allocations)
go tool pprof http://localhost:6060/debug/pprof/heap

# All allocations since start
go tool pprof http://localhost:6060/debug/pprof/allocs

# Memory profile from test
go test -memprofile=mem.prof
go tool pprof mem.prof
```

### Goroutine Profiling
```bash
# Current goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Goroutine blocking
go tool pprof http://localhost:6060/debug/pprof/block

# Mutex contention
go tool pprof http://localhost:6060/debug/pprof/mutex
```

## Interactive Commands

### Navigation
```
top           # Show top functions
top10         # Show top 10 functions
list <func>   # Show source code for function
web           # Open web interface
web <func>    # Focus on specific function
png           # Generate PNG graph
svg           # Generate SVG graph
pdf           # Generate PDF graph
```

### Analysis
```
cum           # Sort by cumulative time
flat          # Sort by flat time
sample_index  # Change sample type
focus <regex> # Focus on matching functions
ignore <regex># Ignore matching functions
hide <regex>  # Hide matching functions
show <regex>  # Show only matching functions
```

### Filtering
```
# Focus on specific package
focus main

# Ignore runtime functions
ignore runtime

# Show only allocations > 1MB
sample_index=alloc_space
show .*

# Hide small functions
hide <10ms
```

## Command Line Options

### Collection
```bash
# Specify duration
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=60

# Different sample rates
go tool pprof -seconds=30 http://localhost:6060/debug/pprof/profile

# Save to file
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
```

### Analysis
```bash
# Different output formats
go tool pprof -png cpu.prof > profile.png
go tool pprof -svg cpu.prof > profile.svg
go tool pprof -pdf cpu.prof > profile.pdf
go tool pprof -text cpu.prof

# Web interface
go tool pprof -http=:8080 cpu.prof

# Compare profiles
go tool pprof -base=old.prof new.prof
```

## Memory Analysis

### Sample Types
```bash
# In-use space (current heap)
go tool pprof -sample_index=inuse_space heap.prof

# In-use objects (current objects)
go tool pprof -sample_index=inuse_objects heap.prof

# Allocated space (total allocated)
go tool pprof -sample_index=alloc_space heap.prof

# Allocated objects (total objects)
go tool pprof -sample_index=alloc_objects heap.prof
```

### Memory Commands
```
# Show memory usage
top -cum

# List function source
list <function>

# Show call graph
web

# Focus on allocations
focus alloc
```

## Programmatic Profiling

### CPU Profiling
```go
import (
    "os"
    "runtime/pprof"
)

// Start CPU profiling
f, err := os.Create("cpu.prof")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

if err := pprof.StartCPUProfile(f); err != nil {
    log.Fatal(err)
}
defer pprof.StopCPUProfile()

// Your code here
```

### Memory Profiling
```go
import (
    "os"
    "runtime"
    "runtime/pprof"
)

// Memory profiling
f, err := os.Create("mem.prof")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

runtime.GC() // Force GC
if err := pprof.WriteHeapProfile(f); err != nil {
    log.Fatal(err)
}
```

### Custom Profiles
```go
// Custom profile
var customProfile = pprof.NewProfile("custom")

// Add to profile
customProfile.Add(someValue, 1)

// Write profile
f, _ := os.Create("custom.prof")
customProfile.WriteTo(f, 0)
f.Close()
```

## Web Interface

### Starting Web UI
```bash
# Modern web interface
go tool pprof -http=:8080 profile.prof

# Access at http://localhost:8080
```

### Web UI Features
- **Top**: Function rankings
- **Graph**: Call graph visualization
- **Flame Graph**: Flame graph view
- **Peek**: Source code view
- **Source**: Annotated source
- **Disasm**: Assembly view

## Benchmarking Integration

### CPU Profile from Benchmark
```bash
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### Memory Profile from Benchmark
```bash
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

### Block Profile from Benchmark
```bash
go test -bench=. -blockprofile=block.prof
go tool pprof block.prof
```

## Common Patterns

### Find Memory Leaks
```bash
# Compare heap profiles over time
go tool pprof -base=heap1.prof heap2.prof

# Focus on growing allocations
focus -inuse_space
top -cum
```

### Find CPU Hotspots
```bash
# CPU profile analysis
go tool pprof cpu.prof
top10 -cum
list <hotspot_function>
```

### Analyze Goroutine Leaks
```bash
# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
top
web
```

## Tips and Best Practices

### Collection Tips
- Profile production-like workloads
- Collect profiles during peak usage
- Use sufficient duration (30+ seconds)
- Profile different scenarios separately

### Analysis Tips
- Start with `top` command
- Use `web` for visual analysis
- Focus on cumulative time for CPU
- Look for unexpected allocations
- Compare profiles over time

### Common Issues
- Profiling overhead (~5% CPU)
- Short profiles may be inaccurate
- Memory profiles show allocations, not usage
- Goroutine profiles are point-in-time

## Environment Variables

```bash
# CPU profile rate (default: 100 Hz)
export CPUPROFILE_RATE=1000

# Memory profile rate (default: 512KB)
export MEMPROFILE_RATE=1024

# Block profile rate
export BLOCKPROFILE_RATE=1

# Mutex profile fraction
export MUTEXPROFILE_FRACTION=1
```

## Quick Reference

| Profile Type | URL | File Flag | Description |
|--------------|-----|-----------|-------------|
| CPU | `/debug/pprof/profile` | `-cpuprofile` | CPU usage |
| Heap | `/debug/pprof/heap` | `-memprofile` | Memory allocations |
| Allocs | `/debug/pprof/allocs` | | All allocations |
| Goroutine | `/debug/pprof/goroutine` | | Goroutine stack traces |
| Block | `/debug/pprof/block` | `-blockprofile` | Blocking operations |
| Mutex | `/debug/pprof/mutex` | `-mutexprofile` | Mutex contention |
| Trace | `/debug/pprof/trace` | | Execution trace |

---

*For more details, see: `go tool pprof -help`*