#!/usr/bin/env pwsh
# PowerShell script to demonstrate CPU profiling examples

Write-Host "=== CPU Profiling Exercise 01-01 Examples ===" -ForegroundColor Green
Write-Host ""

# Example 1: Basic CPU profiling to file
Write-Host "Example 1: Basic CPU profiling to file" -ForegroundColor Yellow
Write-Host "Command: go run . -cpuprofile=cpu.prof -duration=10s -verbose"
go run . -cpuprofile=cpu.prof -duration=10s -verbose
Write-Host ""

# Example 2: Different workload types
Write-Host "Example 2: Testing different workload types" -ForegroundColor Yellow
$workloads = @("primes", "fibonacci", "strings", "hashes", "sort")
foreach ($workload in $workloads) {
    Write-Host "Running workload: $workload"
    go run . -cpuprofile="cpu_$workload.prof" -duration=5s -workload=$workload
}
Write-Host ""

# Example 3: HTTP profiling (background)
Write-Host "Example 3: HTTP profiling server" -ForegroundColor Yellow
Write-Host "Starting HTTP profiling server in background..."
$job = Start-Job -ScriptBlock {
    Set-Location $using:PWD
    go run . -httpprofile=true -duration=15s
}
Start-Sleep 2
Write-Host "HTTP profiling available at: http://localhost:6060/debug/pprof/"
Write-Host "Collecting profile via HTTP..."
Invoke-WebRequest -Uri "http://localhost:6060/debug/pprof/profile?seconds=10" -OutFile "cpu_http.prof"
Wait-Job $job | Remove-Job
Write-Host ""

# Example 4: Benchmark profiling
Write-Host "Example 4: Benchmark profiling" -ForegroundColor Yellow
Write-Host "Running benchmarks with CPU profiling..."
go test -bench=BenchmarkCalculatePrimes -cpuprofile=bench.prof -benchtime=3s
Write-Host ""

# Example 5: Profile analysis
Write-Host "Example 5: Profile analysis examples" -ForegroundColor Yellow
Write-Host "Generated profiles:"
Get-ChildItem -Name "*.prof"
Write-Host ""
Write-Host "Analysis commands you can run:"
Write-Host "  go tool pprof cpu.prof                    # Interactive analysis"
Write-Host "  go tool pprof -top cpu.prof              # Top functions"
Write-Host "  go tool pprof -http=:8080 cpu.prof       # Web interface"
Write-Host "  go tool pprof -list=main.calculatePrimes cpu.prof  # Function details"
Write-Host ""

# Example 6: Comparative analysis
Write-Host "Example 6: Comparative analysis" -ForegroundColor Yellow
Write-Host "Generating before/after profiles for comparison..."
go run . -cpuprofile=before.prof -duration=5s -workload=fibonacci
Write-Host "Now you could optimize the code and generate an 'after.prof'"
Write-Host "Then compare with: go tool pprof -base=before.prof after.prof"
Write-Host ""

Write-Host "=== Examples completed! ===" -ForegroundColor Green
Write-Host "Check the generated .prof files and try the analysis commands above."