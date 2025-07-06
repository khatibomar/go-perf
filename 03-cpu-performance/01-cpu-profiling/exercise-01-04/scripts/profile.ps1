# Profile-Guided Optimization Script for Windows
# This script helps automate the profiling and optimization workflow

param(
    [string]$Command = "all",
    [int]$Size = 1000,
    [int]$Iterations = 10
)

$ProfileDir = "./profiles"
$BeforeDir = "$ProfileDir/before"
$AfterDir = "$ProfileDir/after"
$ReportsDir = "./reports"

# Create directories
New-Item -ItemType Directory -Force -Path $BeforeDir, $AfterDir, $ReportsDir | Out-Null

Write-Host "=== Profile-Guided Optimization Workflow ===" -ForegroundColor Green
Write-Host

# Function to run profiling
function Run-Profile {
    param(
        [string]$Name,
        [string]$OutputDir,
        [string]$Optimized,
        [int]$DataSize = 10000,
        [int]$IterationCount = 100
    )
    
    Write-Host "Running $Name profile..." -ForegroundColor Yellow
    
    # Create output directory
    New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
    
    # CPU Profile
    $args = @(
        "-size=$DataSize",
        "-iterations=$IterationCount"
    )
    
    if ($Optimized -eq "true") {
        $args += "-optimized"
    }
    
    & go run . @args
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "$Name profile completed: $OutputDir" -ForegroundColor Green
    } else {
        Write-Host "$Name profile failed!" -ForegroundColor Red
        exit 1
    }
}

# Function to analyze profile
function Analyze-Profile {
    param(
        [string]$Name,
        [string]$ProfilePath,
        [string]$ReportPath
    )
    
    Write-Host "Analyzing $Name profile..." -ForegroundColor Yellow
    
    # Create report directory
    New-Item -ItemType Directory -Force -Path $ReportPath | Out-Null
    
    try {
        # Generate top functions report
        & go tool pprof -top -cum $ProfilePath > "$ReportPath/top.txt" 2>$null
        
        # Generate call graph
        & go tool pprof -svg $ProfilePath > "$ReportPath/callgraph.svg" 2>$null
        
        Write-Host "$Name analysis completed: $ReportPath" -ForegroundColor Green
    }
    catch {
        Write-Host "Warning: Some analysis steps failed" -ForegroundColor Yellow
    }
}

# Function to compare profiles
function Compare-Profiles {
    Write-Host "Comparing profiles..." -ForegroundColor Yellow
    
    try {
        # Compare CPU profiles
        & go tool pprof -top -diff_base="$BeforeDir/cpu.prof" "$AfterDir/cpu.prof" > "$ReportsDir/comparison.txt" 2>$null
        
        # Read report files
        $beforeReport = ""
        $afterReport = ""
        $comparisonReport = ""
        
        if (Test-Path "$ReportsDir/before/top.txt") {
            $beforeReport = (Get-Content "$ReportsDir/before/top.txt" -Head 20) -join "`n"
        } else {
            $beforeReport = "Profile not found"
        }
        
        if (Test-Path "$ReportsDir/after/top.txt") {
            $afterReport = (Get-Content "$ReportsDir/after/top.txt" -Head 20) -join "`n"
        } else {
            $afterReport = "Profile not found"
        }
        
        if (Test-Path "$ReportsDir/comparison.txt") {
            $comparisonReport = (Get-Content "$ReportsDir/comparison.txt" -Head 20) -join "`n"
        } else {
            $comparisonReport = "Comparison not available"
        }
        
        # Generate comparison report
        $summaryContent = @"
# Profile Comparison Summary

## Before Optimization
``````
$beforeReport
``````

## After Optimization
``````
$afterReport
``````

## Comparison
``````
$comparisonReport
``````

## Analysis

- Review the top CPU-consuming functions
- Identify functions that improved after optimization
- Look for new bottlenecks that may have emerged
- Calculate overall performance improvement

## Next Steps

1. Focus on remaining hot functions
2. Consider algorithmic improvements
3. Look for memory allocation patterns
4. Validate optimizations with benchmarks
"@
        
        $summaryContent | Out-File -FilePath "$ReportsDir/summary.md" -Encoding UTF8
        
        Write-Host "Profile comparison completed: $ReportsDir/summary.md" -ForegroundColor Green
    }
    catch {
        Write-Host "Warning: Profile comparison failed" -ForegroundColor Yellow
    }
}

# Function to show results
function Show-Results {
    Write-Host "=== Results Summary ===" -ForegroundColor Green
    Write-Host
    
    if (Test-Path "$ReportsDir/summary.md") {
        Write-Host "Summary report: $ReportsDir/summary.md" -ForegroundColor Cyan
        Write-Host
        
        # Show first few lines of summary
        $summary = Get-Content "$ReportsDir/summary.md" -Head 10
        $summary | ForEach-Object { Write-Host $_ }
        Write-Host "..." -ForegroundColor Gray
    }
    
    Write-Host
    Write-Host "Available files:" -ForegroundColor Yellow
    Write-Host "- Before profiles: $BeforeDir/" -ForegroundColor White
    Write-Host "- After profiles: $AfterDir/" -ForegroundColor White
    Write-Host "- Analysis reports: $ReportsDir/" -ForegroundColor White
    
    if (Test-Path "$ReportsDir/before/callgraph.svg") {
        Write-Host "- Before call graph: $ReportsDir/before/callgraph.svg" -ForegroundColor White
    }
    
    if (Test-Path "$ReportsDir/after/callgraph.svg") {
        Write-Host "- After call graph: $ReportsDir/after/callgraph.svg" -ForegroundColor White
    }
}

# Main workflow
switch ($Command.ToLower()) {
    "before" {
        Write-Host "Step 1: Profiling BEFORE optimization" -ForegroundColor Cyan
        Run-Profile "BEFORE" $BeforeDir "false" $Size $Iterations
        Analyze-Profile "BEFORE" "$BeforeDir/cpu.prof" "$ReportsDir/before"
    }
    
    "after" {
        Write-Host "Step 2: Profiling AFTER optimization" -ForegroundColor Cyan
        Run-Profile "AFTER" $AfterDir "true" $Size $Iterations
        Analyze-Profile "AFTER" "$AfterDir/cpu.prof" "$ReportsDir/after"
    }
    
    "compare" {
        Write-Host "Step 3: Comparing profiles" -ForegroundColor Cyan
        Compare-Profiles
        Show-Results
    }
    
    "all" {
        Write-Host "Running complete workflow..." -ForegroundColor Cyan
        Write-Host
        
        # Step 1: Before optimization
        Write-Host "Step 1: Profiling BEFORE optimization" -ForegroundColor Cyan
        Run-Profile "BEFORE" $BeforeDir "false" $Size $Iterations
        Analyze-Profile "BEFORE" "$BeforeDir/cpu.prof" "$ReportsDir/before"
        Write-Host
        
        # Step 2: After optimization
        Write-Host "Step 2: Profiling AFTER optimization" -ForegroundColor Cyan
        Run-Profile "AFTER" $AfterDir "true" $Size $Iterations
        Analyze-Profile "AFTER" "$AfterDir/cpu.prof" "$ReportsDir/after"
        Write-Host
        
        # Step 3: Compare
        Write-Host "Step 3: Comparing profiles" -ForegroundColor Cyan
        Compare-Profiles
        Write-Host
        
        Write-Host "=== Workflow Complete ===" -ForegroundColor Green
        Show-Results
    }
    
    "clean" {
        Write-Host "Cleaning up profiles and reports..." -ForegroundColor Yellow
        Remove-Item -Recurse -Force -Path $ProfileDir, $ReportsDir -ErrorAction SilentlyContinue
        Write-Host "Cleanup completed" -ForegroundColor Green
    }
    
    "help" {
        Write-Host "Usage: .\profile.ps1 [command] [options]" -ForegroundColor White
        Write-Host
        Write-Host "Commands:" -ForegroundColor Yellow
        Write-Host "  before   - Profile before optimization" -ForegroundColor White
        Write-Host "  after    - Profile after optimization" -ForegroundColor White
        Write-Host "  compare  - Compare before/after profiles" -ForegroundColor White
        Write-Host "  all      - Run complete workflow (default)" -ForegroundColor White
        Write-Host "  clean    - Clean up generated files" -ForegroundColor White
        Write-Host "  help     - Show this help" -ForegroundColor White
        Write-Host
        Write-Host "Options:" -ForegroundColor Yellow
        Write-Host "  -Workload <type>     - Workload type: sorting, searching, graph, mixed (default: mixed)" -ForegroundColor White
        Write-Host "  -Size <number>       - Data size for workload (default: 10000)" -ForegroundColor White
        Write-Host "  -Iterations <number> - Number of iterations (default: 100)" -ForegroundColor White
        Write-Host
        Write-Host "Examples:" -ForegroundColor Yellow
        Write-Host "  .\profile.ps1                              # Run complete workflow" -ForegroundColor White
        Write-Host "  .\profile.ps1 before                       # Profile unoptimized code" -ForegroundColor White
        Write-Host "  .\profile.ps1 after                        # Profile optimized code" -ForegroundColor White
        Write-Host "  .\profile.ps1 all -Workload sorting -Size 50000  # Custom workflow" -ForegroundColor White
    }
    
    default {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Write-Host "Use '.\profile.ps1 help' for usage information" -ForegroundColor Yellow
        exit 1
    }
}