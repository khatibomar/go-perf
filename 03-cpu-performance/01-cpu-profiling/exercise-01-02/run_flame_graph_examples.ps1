#!/usr/bin/env pwsh
# PowerShell script for Exercise 01-02: Flame Graph Analysis
# This script demonstrates various CPU profiling scenarios for flame graph generation

Write-Host "=== Exercise 01-02: Flame Graph Analysis Examples ===" -ForegroundColor Green
Write-Host ""

# Check if we're in the correct directory
if (-not (Test-Path "main.go")) {
    Write-Host "Error: Please run this script from the exercise-01-02 directory" -ForegroundColor Red
    exit 1
}

# Create profiles directory
if (-not (Test-Path "profiles")) {
    New-Item -ItemType Directory -Name "profiles" | Out-Null
    Write-Host "Created profiles directory" -ForegroundColor Yellow
}

Write-Host "Building the application..." -ForegroundColor Cyan
try {
    go build -o flame_graph_app.exe .
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed"
    }
    Write-Host "✓ Build successful" -ForegroundColor Green
} catch {
    Write-Host "✗ Build failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== Scenario 1: Basic CPU Profiling for Flame Graph ===" -ForegroundColor Yellow
Write-Host "Running realistic workload with CPU profiling..."

# Basic CPU profiling
.\flame_graph_app.exe -cpuprofile="profiles\realistic_cpu.prof" -duration=10s -workload=realistic

if (Test-Path "profiles\realistic_cpu.prof") {
    Write-Host "✓ CPU profile generated: profiles\realistic_cpu.prof" -ForegroundColor Green
    Write-Host "  Analysis commands:"
    Write-Host "    go tool pprof profiles\realistic_cpu.prof" -ForegroundColor Cyan
    Write-Host "    go tool pprof -http=:8080 profiles\realistic_cpu.prof" -ForegroundColor Cyan
    Write-Host "    go tool pprof -web profiles\realistic_cpu.prof" -ForegroundColor Cyan
else
    Write-Host "✗ CPU profile not generated" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Scenario 2: CPU-Heavy Workload Profiling ===" -ForegroundColor Yellow
Write-Host "Running CPU-intensive workload..."

# CPU-heavy workload profiling
.\flame_graph_app.exe -cpuprofile="profiles\cpu_heavy.prof" -duration=8s -workload=cpu-heavy

if (Test-Path "profiles\cpu_heavy.prof") {
    Write-Host "✓ CPU-heavy profile generated: profiles\cpu_heavy.prof" -ForegroundColor Green
else
    Write-Host "✗ CPU-heavy profile not generated" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Scenario 3: I/O-Heavy Workload Profiling ===" -ForegroundColor Yellow
Write-Host "Running I/O-intensive workload..."

# I/O-heavy workload profiling
.\flame_graph_app.exe -cpuprofile="profiles\io_heavy.prof" -duration=6s -workload=io-heavy

if (Test-Path "profiles\io_heavy.prof") {
    Write-Host "✓ I/O-heavy profile generated: profiles\io_heavy.prof" -ForegroundColor Green
else
    Write-Host "✗ I/O-heavy profile not generated" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Scenario 4: Mixed Workload Profiling ===" -ForegroundColor Yellow
Write-Host "Running mixed workload..."

# Mixed workload profiling
.\flame_graph_app.exe -cpuprofile="profiles\mixed.prof" -duration=8s -workload=mixed

if (Test-Path "profiles\mixed.prof") {
    Write-Host "✓ Mixed workload profile generated: profiles\mixed.prof" -ForegroundColor Green
else
    Write-Host "✗ Mixed workload profile not generated" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Scenario 5: Comprehensive Profiling with All Profile Types ===" -ForegroundColor Yellow
Write-Host "Running comprehensive profiling session..."

# Comprehensive profiling with multiple profile types
.\flame_graph_app.exe `
    -cpuprofile="profiles\comprehensive_cpu.prof" `
    -memprofile="profiles\comprehensive_mem.prof" `
    -duration=12s `
    -workload=realistic `
    -output-dir="profiles"

Write-Host "✓ Comprehensive profiling completed" -ForegroundColor Green

Write-Host ""
Write-Host "=== Scenario 6: HTTP Profiling Server (Background) ===" -ForegroundColor Yellow
Write-Host "Starting HTTP profiling server in background..."
Write-Host "Note: This will run in the background. Use Ctrl+C to stop when done."

# Start HTTP profiling server in background
Start-Process -FilePath ".\flame_graph_app.exe" -ArgumentList "-http=localhost:6060", "-duration=30s", "-workload=mixed" -WindowStyle Hidden

Start-Sleep -Seconds 2
Write-Host "✓ HTTP profiling server started on http://localhost:6060" -ForegroundColor Green
Write-Host "  Available endpoints:"
Write-Host "    http://localhost:6060/debug/pprof/          - Index page" -ForegroundColor Cyan
Write-Host "    http://localhost:6060/debug/pprof/profile   - CPU profile" -ForegroundColor Cyan
Write-Host "    http://localhost:6060/debug/pprof/heap      - Heap profile" -ForegroundColor Cyan
Write-Host "    http://localhost:6060/debug/pprof/goroutine - Goroutine profile" -ForegroundColor Cyan

Write-Host ""
Write-Host "=== Scenario 7: Benchmark Profiling ===" -ForegroundColor Yellow
Write-Host "Running benchmarks with CPU profiling..."

# Benchmark profiling
try {
    go test -bench=BenchmarkRealisticWorkload -cpuprofile="profiles\benchmark_cpu.prof" -benchtime=5s
    if (Test-Path "profiles\benchmark_cpu.prof") {
        Write-Host "✓ Benchmark CPU profile generated: profiles\benchmark_cpu.prof" -ForegroundColor Green
    }
} catch {
    Write-Host "Note: Benchmark profiling may require additional setup" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Scenario 8: Comparative Analysis Setup ===" -ForegroundColor Yellow
Write-Host "Generating profiles for comparative analysis..."

# Generate profiles for different scenarios for comparison
$scenarios = @(
    @{name="algorithms_heavy"; workload="cpu-heavy"; duration="6s"},
    @{name="data_structures"; workload="realistic"; duration="6s"},
    @{name="server_processing"; workload="io-heavy"; duration="6s"}
)

foreach ($scenario in $scenarios) {
    Write-Host "  Generating profile for $($scenario.name)..."
    .\flame_graph_app.exe `
        -cpuprofile="profiles\$($scenario.name).prof" `
        -duration=$($scenario.duration) `
        -workload=$($scenario.workload)
    
    if (Test-Path "profiles\$($scenario.name).prof") {
        Write-Host "    ✓ $($scenario.name).prof generated" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "=== Profile Analysis Commands ===" -ForegroundColor Magenta
Write-Host ""
Write-Host "1. Interactive Analysis:" -ForegroundColor Cyan
Write-Host "   go tool pprof profiles\realistic_cpu.prof"
Write-Host "   # Then use commands: top, list, web, png, svg"
Write-Host ""
Write-Host "2. Web Interface (Flame Graph):" -ForegroundColor Cyan
Write-Host "   go tool pprof -http=:8080 profiles\realistic_cpu.prof"
Write-Host "   # Navigate to http://localhost:8080 and select 'Flame Graph' view"
Write-Host ""
Write-Host "3. Generate SVG Flame Graph:" -ForegroundColor Cyan
Write-Host "   go tool pprof -svg profiles\realistic_cpu.prof > flame_graph.svg"
Write-Host ""
Write-Host "4. Generate PNG Graph:" -ForegroundColor Cyan
Write-Host "   go tool pprof -png profiles\realistic_cpu.prof > call_graph.png"
Write-Host ""
Write-Host "5. Top Functions:" -ForegroundColor Cyan
Write-Host "   go tool pprof -top profiles\realistic_cpu.prof"
Write-Host ""
Write-Host "6. Compare Profiles:" -ForegroundColor Cyan
Write-Host "   go tool pprof -base profiles\algorithms_heavy.prof profiles\data_structures.prof"
Write-Host ""
Write-Host "7. Memory Profile Analysis:" -ForegroundColor Cyan
Write-Host "   go tool pprof profiles\comprehensive_mem.prof"
Write-Host ""
Write-Host "8. Trace Analysis (if trace files exist):" -ForegroundColor Cyan
Write-Host "   go tool trace profiles\trace.out"
Write-Host ""

Write-Host "=== Flame Graph Analysis Tips ===" -ForegroundColor Magenta
Write-Host ""
Write-Host "Understanding Flame Graphs:" -ForegroundColor Yellow
Write-Host "• Width = Time spent in function (including children)"
Write-Host "• Height = Call stack depth"
Write-Host "• Color = Different functions (no semantic meaning)"
Write-Host "• Flat top = Functions that are on-CPU"
Write-Host "• Wide functions = Hot spots (optimization targets)"
Write-Host ""
Write-Host "Key Patterns to Look For:" -ForegroundColor Yellow
Write-Host "• Wide plateaus = CPU-intensive functions"
Write-Host "• Tall stacks = Deep call chains"
Write-Host "• Multiple narrow spikes = Fragmented execution"
Write-Host "• Gaps = Time spent in system calls or waiting"
Write-Host ""
Write-Host "Optimization Strategies:" -ForegroundColor Yellow
Write-Host "• Focus on the widest functions first"
Write-Host "• Look for unexpected function calls"
Write-Host "• Identify algorithmic inefficiencies"
Write-Host "• Check for excessive memory allocations"
Write-Host "• Analyze goroutine contention patterns"
Write-Host ""

Write-Host "=== Generated Files ===" -ForegroundColor Magenta
Write-Host ""
if (Test-Path "profiles") {
    $profiles = Get-ChildItem -Path "profiles" -Filter "*.prof"
    if ($profiles.Count -gt 0) {
        Write-Host "Profile files generated:" -ForegroundColor Green
        foreach ($profile in $profiles) {
            $size = [math]::Round($profile.Length / 1KB, 2)
            Write-Host "  $($profile.Name) ($size KB)" -ForegroundColor Cyan
        }
    } else {
        Write-Host "No profile files found in profiles directory" -ForegroundColor Yellow
    }
    
    # Check for other files
    $otherFiles = Get-ChildItem -Path "profiles" -Exclude "*.prof"
    if ($otherFiles.Count -gt 0) {
        Write-Host ""
        Write-Host "Other files:" -ForegroundColor Green
        foreach ($file in $otherFiles) {
            Write-Host "  $($file.Name)" -ForegroundColor Cyan
        }
    }
else
    Write-Host "Profiles directory not found" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Next Steps ===" -ForegroundColor Magenta
Write-Host ""
Write-Host "1. Analyze the generated profiles using the commands above"
Write-Host "2. Generate flame graphs and identify hot spots"
Write-Host "3. Compare different workload profiles"
Write-Host "4. Implement optimizations based on findings"
Write-Host "5. Re-profile to measure improvement"
Write-Host ""
Write-Host "=== Exercise Complete ===" -ForegroundColor Green
Write-Host "All flame graph analysis examples have been generated."
Write-Host "Use the analysis commands above to explore the profiles and create flame graphs."
Write-Host ""

# Cleanup
if (Test-Path "flame_graph_app.exe") {
    Write-Host "Cleaning up executable..." -ForegroundColor Yellow
    Remove-Item "flame_graph_app.exe" -Force
}

Write-Host "Note: HTTP profiling server may still be running in background." -ForegroundColor Yellow
Write-Host "Use Task Manager or 'taskkill /f /im flame_graph_app.exe' to stop it if needed." -ForegroundColor Yellow