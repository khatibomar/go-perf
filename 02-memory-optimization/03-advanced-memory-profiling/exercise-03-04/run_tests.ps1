# Memory Leak Detection System - Test Runner Script
# PowerShell script for running tests on Windows

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("all", "quick", "stress", "bench", "coverage", "validate", "clean", "help")]
    [string]$Command,
    
    [switch]$Verbose,
    [switch]$NoCoverage,
    [switch]$NoBenchmark,
    [switch]$NoRace,
    [string]$OutputDir = "test-results",
    [int]$Timeout = 600  # 10 minutes default
)

# Color output functions
function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Cyan
}

function Write-Header {
    param([string]$Message)
    Write-Host "`n🚀 $Message" -ForegroundColor Magenta
    Write-Host ("=" * 60) -ForegroundColor Gray
}

function Write-SubHeader {
    param([string]$Message)
    Write-Host "`n📋 $Message" -ForegroundColor Blue
    Write-Host ("-" * 40) -ForegroundColor Gray
}

# Validate environment
function Test-Environment {
    Write-SubHeader "Validating Test Environment"
    
    # Check Go installation
    try {
        $goVersion = go version
        Write-Success "Go Version: $goVersion"
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        return $false
    }
    
    # Check Go modules
    try {
        go mod verify
        Write-Success "Go modules verified"
    }
    catch {
        Write-Error "Go modules verification failed"
        return $false
    }
    
    # Check test files
    $testFiles = @(
        "tests\unit_test.go",
        "tests\integration_test.go",
        "tests\benchmark_test.go"
    )
    
    foreach ($file in $testFiles) {
        if (-not (Test-Path $file)) {
            Write-Error "Test file not found: $file"
            return $false
        }
    }
    Write-Success "Test files found"
    
    # Check output directory
    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
        Write-Success "Created output directory: $OutputDir"
    }
    
    return $true
}

# Run a specific test suite
function Invoke-TestSuite {
    param(
        [string]$Name,
        [string]$Pattern,
        [string[]]$ExtraArgs = @(),
        [int]$TimeoutSeconds = 300
    )
    
    Write-SubHeader "Running $Name"
    
    $args = @("test")
    
    if ($Verbose) {
        $args += "-v"
    }
    
    if (-not $NoRace) {
        $args += "-race"
    }
    
    if (-not $NoCoverage) {
        $coverageFile = Join-Path $OutputDir "$($Name.Replace(' ', '_'))-coverage.out"
        $args += "-coverprofile", $coverageFile, "-covermode", "atomic"
    }
    
    $args += "-run", $Pattern
    $args += "-timeout", "${TimeoutSeconds}s"
    $args += $ExtraArgs
    $args += "./tests"
    
    $outputFile = Join-Path $OutputDir "$($Name.Replace(' ', '_'))-output.txt"
    
    $startTime = Get-Date
    
    try {
        $process = Start-Process -FilePath "go" -ArgumentList $args -NoNewWindow -Wait -PassThru -RedirectStandardOutput $outputFile -RedirectStandardError $outputFile
        $duration = (Get-Date) - $startTime
        
        if ($process.ExitCode -eq 0) {
            Write-Success "$Name completed successfully (Duration: $($duration.ToString('mm\:ss')))"
            return $true
        } else {
            Write-Error "$Name failed (Duration: $($duration.ToString('mm\:ss')))"
            if ($Verbose) {
                Write-Host "Output saved to: $outputFile" -ForegroundColor Gray
            }
            return $false
        }
    }
    catch {
        Write-Error "Failed to run $Name : $_"
        return $false
    }
}

# Run benchmarks
function Invoke-Benchmarks {
    Write-SubHeader "Running Benchmarks"
    
    $args = @("test", "-bench=.", "-benchmem", "-run=^$", "./tests")
    $benchmarkFile = Join-Path $OutputDir "benchmarks.txt"
    
    try {
        $process = Start-Process -FilePath "go" -ArgumentList $args -NoNewWindow -Wait -PassThru -RedirectStandardOutput $benchmarkFile -RedirectStandardError $benchmarkFile
        
        if ($process.ExitCode -eq 0) {
            Write-Success "Benchmarks completed"
            Write-Info "Results saved to: $benchmarkFile"
            return $true
        } else {
            Write-Error "Benchmarks failed"
            return $false
        }
    }
    catch {
        Write-Error "Failed to run benchmarks: $_"
        return $false
    }
}

# Generate coverage report
function New-CoverageReport {
    Write-SubHeader "Generating Coverage Report"
    
    # Find coverage files
    $coverageFiles = Get-ChildItem -Path $OutputDir -Filter "*-coverage.out"
    
    if ($coverageFiles.Count -eq 0) {
        Write-Warning "No coverage files found"
        return $false
    }
    
    # Merge coverage files
    $mergedCoverage = Join-Path $OutputDir "coverage.out"
    
    try {
        # Create merged coverage file
        "mode: atomic" | Out-File -FilePath $mergedCoverage -Encoding UTF8
        
        foreach ($file in $coverageFiles) {
            $content = Get-Content $file.FullName
            # Skip the mode line and append the rest
            $content[1..($content.Length-1)] | Where-Object { $_ -ne "" } | Add-Content -Path $mergedCoverage -Encoding UTF8
        }
        
        # Generate HTML coverage report
        $htmlReport = Join-Path $OutputDir "coverage.html"
        $process = Start-Process -FilePath "go" -ArgumentList @("tool", "cover", "-html", $mergedCoverage, "-o", $htmlReport) -NoNewWindow -Wait -PassThru
        
        if ($process.ExitCode -eq 0) {
            Write-Success "HTML coverage report generated: $htmlReport"
        }
        
        # Generate coverage summary
        $summaryFile = Join-Path $OutputDir "coverage-summary.txt"
        $process = Start-Process -FilePath "go" -ArgumentList @("tool", "cover", "-func", $mergedCoverage) -NoNewWindow -Wait -PassThru -RedirectStandardOutput $summaryFile
        
        if ($process.ExitCode -eq 0) {
            Write-Success "Coverage summary generated: $summaryFile"
            
            # Display coverage summary
            if ($Verbose) {
                Write-Info "Coverage Summary:"
                Get-Content $summaryFile | Select-Object -Last 10 | ForEach-Object {
                    Write-Host "  $_" -ForegroundColor Gray
                }
            }
        }
        
        return $true
    }
    catch {
        Write-Error "Failed to generate coverage report: $_"
        return $false
    }
}

# Generate test report
function New-TestReport {
    param(
        [hashtable]$Results
    )
    
    $reportFile = Join-Path $OutputDir "test-report.md"
    
    $report = @"
# Memory Leak Detection System - Test Report

Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')

## Test Results Summary

| Test Suite | Status | Notes |
|------------|--------|---------|

"@
    
    foreach ($suite in $Results.Keys) {
        $status = if ($Results[$suite]) { "✅ PASS" } else { "❌ FAIL" }
        $report += "| $suite | $status | |
"
    }
    
    $report += @"

## Coverage Report

Detailed coverage report available in ``coverage.html``

## Benchmark Results

Detailed benchmark results available in ``benchmarks.txt``

## Recommendations

- Maintain test coverage above 80%
- Run integration tests before each release
- Monitor benchmark performance for regressions
- Update tests when adding new features

## Files Generated

- ``coverage.html`` - Interactive coverage report
- ``coverage-summary.txt`` - Coverage summary
- ``benchmarks.txt`` - Benchmark results
- Test output files for each suite

"@
    
    $report | Out-File -FilePath $reportFile -Encoding UTF8
    Write-Success "Test report generated: $reportFile"
}

# Clean test artifacts
function Clear-TestArtifacts {
    Write-SubHeader "Cleaning Test Artifacts"
    
    if (Test-Path $OutputDir) {
        Remove-Item -Path $OutputDir -Recurse -Force
        Write-Success "Cleaned test artifacts"
    } else {
        Write-Info "No test artifacts to clean"
    }
}

# Show help
function Show-Help {
    Write-Host @"
🧪 Memory Leak Detection System - Test Runner

USAGE:
    .\run_tests.ps1 -Command <command> [options]

COMMANDS:
    all        Run all tests (unit, integration, benchmarks, coverage)
    quick      Run quick tests (unit tests only)
    stress     Run stress tests (concurrency and load tests)
    bench      Run benchmarks only
    coverage   Generate coverage report only
    validate   Validate test environment
    clean      Clean test artifacts
    help       Show this help

OPTIONS:
    -Verbose        Enable verbose output
    -NoCoverage     Skip coverage collection
    -NoBenchmark    Skip benchmarks (for 'all' command)
    -NoRace         Skip race detection
    -OutputDir      Specify output directory (default: test-results)
    -Timeout        Test timeout in seconds (default: 600)

EXAMPLES:
    .\run_tests.ps1 -Command all -Verbose
    .\run_tests.ps1 -Command quick -NoCoverage
    .\run_tests.ps1 -Command bench -OutputDir "my-results"
    .\run_tests.ps1 -Command coverage

"@ -ForegroundColor White
}

# Main execution
function Main {
    Write-Header "Memory Leak Detection System Test Suite"
    
    # Handle help command
    if ($Command -eq "help") {
        Show-Help
        return
    }
    
    # Handle clean command
    if ($Command -eq "clean") {
        Clear-TestArtifacts
        return
    }
    
    # Validate environment
    if (-not (Test-Environment)) {
        Write-Error "Environment validation failed"
        exit 1
    }
    
    $results = @{}
    $overallSuccess = $true
    
    switch ($Command) {
        "all" {
            # Run all test suites
            $testSuites = @(
                @{ Name = "Unit Tests"; Pattern = "Test.*" },
                @{ Name = "Integration Tests"; Pattern = "TestBasicLeakDetection|TestHeapLeakDetection|TestGoroutineLeakDetection|TestResourceLeakDetection|TestMultiTypeLeakDetection" },
                @{ Name = "Real-time Tests"; Pattern = "TestRealTimeAlerts|TestAnalyzerLifecycle" },
                @{ Name = "Configuration Tests"; Pattern = "TestDetectorConfiguration|TestDetectorErrorHandling|TestDetectorConcurrency" }
            )
            
            foreach ($suite in $testSuites) {
                $success = Invoke-TestSuite -Name $suite.Name -Pattern $suite.Pattern -TimeoutSeconds $Timeout
                $results[$suite.Name] = $success
                if (-not $success) { $overallSuccess = $false }
            }
            
            # Run benchmarks if not disabled
            if (-not $NoBenchmark) {
                $success = Invoke-Benchmarks
                $results["Benchmarks"] = $success
                if (-not $success) { $overallSuccess = $false }
            }
            
            # Generate coverage report if not disabled
            if (-not $NoCoverage) {
                $success = New-CoverageReport
                $results["Coverage"] = $success
            }
            
            # Generate test report
            New-TestReport -Results $results
        }
        
        "quick" {
            $success = Invoke-TestSuite -Name "Quick Tests" -Pattern "TestHeapAnalyzer|TestGoroutineDetector|TestResourceDetector|TestLeakSimulator" -TimeoutSeconds 120
            $results["Quick Tests"] = $success
            $overallSuccess = $success
        }
        
        "stress" {
            $success = Invoke-TestSuite -Name "Stress Tests" -Pattern "TestDetectorConcurrency|TestConcurrentDetection" -ExtraArgs @("-count=3") -TimeoutSeconds 300
            $results["Stress Tests"] = $success
            $overallSuccess = $success
        }
        
        "bench" {
            $success = Invoke-Benchmarks
            $results["Benchmarks"] = $success
            $overallSuccess = $success
        }
        
        "coverage" {
            $success = New-CoverageReport
            $results["Coverage"] = $success
            $overallSuccess = $success
        }
        
        "validate" {
            Write-Success "Environment validation passed"
            return
        }
    }
    
    # Print summary
    Write-Header "Test Summary"
    
    $totalTests = $results.Count
    $passedTests = ($results.Values | Where-Object { $_ }).Count
    $failedTests = $totalTests - $passedTests
    
    if ($totalTests -gt 0) {
        $successRate = [math]::Round(($passedTests / $totalTests) * 100, 1)
        
        Write-Host "Total Test Suites: $totalTests" -ForegroundColor White
        Write-Host "Passed: $passedTests" -ForegroundColor Green
        Write-Host "Failed: $failedTests" -ForegroundColor Red
        Write-Host "Success Rate: $successRate%" -ForegroundColor $(if ($successRate -ge 80) { "Green" } else { "Yellow" })
        Write-Host "Results saved to: $OutputDir" -ForegroundColor Cyan
    }
    
    if ($overallSuccess) {
        Write-Host "`n🎉 All tests completed successfully!" -ForegroundColor Green
    } else {
        Write-Host "`n💥 Some tests failed. Check the output files for details." -ForegroundColor Red
        exit 1
    }
}

# Execute main function
Main