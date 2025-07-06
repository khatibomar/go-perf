# Optimization Analysis Script for Go Compiler Optimization Exercise
# This script analyzes compiler optimizations using various Go tools and flags

param(
    [string]$SourceFile = "main.go",
    [string]$OutputDir = "analysis",
    [switch]$Verbose,
    [switch]$ShowAssembly,
    [switch]$ShowInlining,
    [switch]$ShowEscape,
    [switch]$ShowSSA,
    [switch]$All
)

# Set all flags if -All is specified
if ($All) {
    $ShowAssembly = $true
    $ShowInlining = $true
    $ShowEscape = $true
    $ShowSSA = $true
    $Verbose = $true
}

# Ensure we're in the right directory
if (!(Test-Path "go.mod")) {
    Write-Error "go.mod not found. Please run this script from the project root."
    exit 1
}

# Create output directory
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

# Function to run Go command and capture output
function Invoke-GoCommand {
    param(
        [string[]]$Arguments,
        [string]$OutputFile = $null,
        [string]$Description = ""
    )
    
    if ($Description) {
        Write-Host "$Description..." -ForegroundColor Green
    }
    
    if ($Verbose) {
        Write-Host "Command: go $($Arguments -join ' ')" -ForegroundColor Cyan
    }
    
    try {
        if ($OutputFile) {
            $Output = & go @Arguments 2>&1
            $Output | Out-File -FilePath $OutputFile -Encoding UTF8
            Write-Host "  ✓ Output saved to: $OutputFile" -ForegroundColor Green
        } else {
            & go @Arguments
        }
        return $true
    } catch {
        Write-Host "  ✗ Command failed: $_" -ForegroundColor Red
        return $false
    }
}

# Function to analyze inlining decisions
function Analyze-Inlining {
    Write-Host "=== Inlining Analysis ===" -ForegroundColor Magenta
    
    # Basic inlining analysis
    $InliningOutput = Join-Path $OutputDir "inlining_analysis.txt"
    $Success = Invoke-GoCommand -Arguments @("build", "-gcflags=-m", ".") -OutputFile $InliningOutput -Description "Analyzing inlining decisions"
    
    if ($Success) {
        # Parse and summarize inlining decisions
        $Content = Get-Content $InliningOutput
        $InlinedFunctions = $Content | Where-Object { $_ -match "can inline" }
        $NotInlinedFunctions = $Content | Where-Object { $_ -match "cannot inline" }
        $InliningCalls = $Content | Where-Object { $_ -match "inlining call" }
        
        Write-Host "\nInlining Summary:" -ForegroundColor Yellow
        Write-Host "  Functions that can be inlined: $($InlinedFunctions.Count)" -ForegroundColor Green
        Write-Host "  Functions that cannot be inlined: $($NotInlinedFunctions.Count)" -ForegroundColor Red
        Write-Host "  Actual inlining calls: $($InliningCalls.Count)" -ForegroundColor Cyan
        
        if ($Verbose -and $InlinedFunctions.Count -gt 0) {
            Write-Host "\nInlined functions:" -ForegroundColor Green
            $InlinedFunctions | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
        }
        
        if ($Verbose -and $NotInlinedFunctions.Count -gt 0) {
            Write-Host "\nNon-inlined functions:" -ForegroundColor Red
            $NotInlinedFunctions | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
        }
    }
    
    # Detailed inlining analysis with different levels
    $InliningLevels = @(0, 1, 2, 3, 4)
    Write-Host "\nInlining level comparison:" -ForegroundColor Yellow
    
    foreach ($Level in $InliningLevels) {
        $LevelOutput = Join-Path $OutputDir "inlining_level_$Level.txt"
        $Success = Invoke-GoCommand -Arguments @("build", "-gcflags=-l=$Level -m", ".") -OutputFile $LevelOutput -Description "Analyzing inlining level $Level"
        
        if ($Success) {
            $Content = Get-Content $LevelOutput
            $InlinedCount = ($Content | Where-Object { $_ -match "can inline" }).Count
            $CallsCount = ($Content | Where-Object { $_ -match "inlining call" }).Count
            Write-Host "  Level $Level`: $InlinedCount functions can be inlined, $CallsCount calls inlined" -ForegroundColor Cyan
        }
    }
    
    Write-Host ""
}

# Function to analyze escape analysis
function Analyze-EscapeAnalysis {
    Write-Host "=== Escape Analysis ===" -ForegroundColor Magenta
    
    # Basic escape analysis
    $EscapeOutput = Join-Path $OutputDir "escape_analysis.txt"
    $Success = Invoke-GoCommand -Arguments @("build", "-gcflags=-m -m", ".") -OutputFile $EscapeOutput -Description "Analyzing escape decisions"
    
    if ($Success) {
        # Parse and summarize escape decisions
        $Content = Get-Content $EscapeOutput
        $EscapesToHeap = $Content | Where-Object { $_ -match "escapes to heap" }
        $DoesNotEscape = $Content | Where-Object { $_ -match "does not escape" }
        $MovedToHeap = $Content | Where-Object { $_ -match "moved to heap" }
        
        Write-Host "\nEscape Analysis Summary:" -ForegroundColor Yellow
        Write-Host "  Variables that escape to heap: $($EscapesToHeap.Count)" -ForegroundColor Red
        Write-Host "  Variables that do not escape: $($DoesNotEscape.Count)" -ForegroundColor Green
        Write-Host "  Variables moved to heap: $($MovedToHeap.Count)" -ForegroundColor Yellow
        
        if ($Verbose -and $EscapesToHeap.Count -gt 0) {
            Write-Host "\nVariables escaping to heap:" -ForegroundColor Red
            $EscapesToHeap | Select-Object -First 10 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            if ($EscapesToHeap.Count -gt 10) {
                Write-Host "  ... and $($EscapesToHeap.Count - 10) more" -ForegroundColor Gray
            }
        }
        
        if ($Verbose -and $DoesNotEscape.Count -gt 0) {
            Write-Host "\nVariables staying on stack:" -ForegroundColor Green
            $DoesNotEscape | Select-Object -First 10 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            if ($DoesNotEscape.Count -gt 10) {
                Write-Host "  ... and $($DoesNotEscape.Count - 10) more" -ForegroundColor Gray
            }
        }
    }
    
    Write-Host ""
}

# Function to generate assembly output
function Analyze-Assembly {
    Write-Host "=== Assembly Analysis ===" -ForegroundColor Magenta
    
    # Generate assembly for different optimization levels
    $OptimizationLevels = @(
        @{ Name = "debug"; Flags = "-N -l" },
        @{ Name = "default"; Flags = "" },
        @{ Name = "aggressive"; Flags = "-l=4" }
    )
    
    foreach ($Level in $OptimizationLevels) {
        $AssemblyOutput = Join-Path $OutputDir "assembly_$($Level.Name).s"
        $GCFlags = if ($Level.Flags) { "-gcflags=$($Level.Flags) -S" } else { "-gcflags=-S" }
        
        $Success = Invoke-GoCommand -Arguments @("build", $GCFlags, ".") -OutputFile $AssemblyOutput -Description "Generating assembly for $($Level.Name) optimization"
        
        if ($Success) {
            $Content = Get-Content $AssemblyOutput
            $InstructionCount = ($Content | Where-Object { $_ -match "^\s+[A-Z]+" }).Count
            $FunctionCount = ($Content | Where-Object { $_ -match "^TEXT" }).Count
            
            Write-Host "  $($Level.Name) optimization: $FunctionCount functions, ~$InstructionCount instructions" -ForegroundColor Cyan
        }
    }
    
    Write-Host ""
}

# Function to analyze SSA (Static Single Assignment) form
function Analyze-SSA {
    Write-Host "=== SSA Analysis ===" -ForegroundColor Magenta
    
    # Generate SSA dump for main optimization phases
    $SSAPhases = @("start", "opt", "lower", "final")
    
    foreach ($Phase in $SSAPhases) {
        $SSAOutput = Join-Path $OutputDir "ssa_$Phase.txt"
        $Success = Invoke-GoCommand -Arguments @("build", "-gcflags=-d=ssa/$Phase/on", ".") -OutputFile $SSAOutput -Description "Generating SSA dump for $Phase phase"
        
        if ($Success) {
            $Content = Get-Content $SSAOutput
            $BlockCount = ($Content | Where-Object { $_ -match "^b\d+:" }).Count
            $ValueCount = ($Content | Where-Object { $_ -match "v\d+" }).Count
            
            Write-Host "  $Phase phase: $BlockCount blocks, ~$ValueCount values" -ForegroundColor Cyan
        }
    }
    
    Write-Host ""
}

# Function to run benchmarks and analyze performance
function Analyze-Performance {
    Write-Host "=== Performance Analysis ===" -ForegroundColor Magenta
    
    # Run benchmarks with different optimization levels
    $BenchmarkOutput = Join-Path $OutputDir "benchmark_results.txt"
    
    Write-Host "Running benchmarks..." -ForegroundColor Green
    $Success = Invoke-GoCommand -Arguments @("test", "-bench=.", "-benchmem", "./benchmarks") -OutputFile $BenchmarkOutput -Description "Running performance benchmarks"
    
    if ($Success) {
        $Content = Get-Content $BenchmarkOutput
        $BenchmarkLines = $Content | Where-Object { $_ -match "^Benchmark" }
        
        Write-Host "\nBenchmark Summary:" -ForegroundColor Yellow
        Write-Host "  Total benchmarks run: $($BenchmarkLines.Count)" -ForegroundColor Cyan
        
        if ($Verbose -and $BenchmarkLines.Count -gt 0) {
            Write-Host "\nTop 5 benchmark results:" -ForegroundColor Green
            $BenchmarkLines | Select-Object -First 5 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
        }
    }
    
    Write-Host ""
}

# Function to analyze binary size with different flags
function Analyze-BinarySize {
    Write-Host "=== Binary Size Analysis ===" -ForegroundColor Magenta
    
    $BuildConfigs = @(
        @{ Name = "debug"; Flags = @("-gcflags=-N -l") },
        @{ Name = "default"; Flags = @() },
        @{ Name = "optimized"; Flags = @("-gcflags=-l=4") },
        @{ Name = "stripped"; Flags = @("-ldflags=-s -w") },
        @{ Name = "full"; Flags = @("-gcflags=-l=4", "-ldflags=-s -w") }
    )
    
    $SizeResults = @()
    
    foreach ($Config in $BuildConfigs) {
        $OutputFile = Join-Path $OutputDir "binary_$($Config.Name).exe"
        $BuildArgs = @("build") + $Config.Flags + @("-o", $OutputFile, ".")
        
        $Success = Invoke-GoCommand -Arguments $BuildArgs -Description "Building $($Config.Name) binary"
        
        if ($Success -and (Test-Path $OutputFile)) {
            $FileInfo = Get-Item $OutputFile
            $SizeResults += @{
                Name = $Config.Name
                Size = $FileInfo.Length
                SizeMB = [math]::Round($FileInfo.Length / 1MB, 2)
            }
            
            Write-Host "  $($Config.Name): $([math]::Round($FileInfo.Length / 1MB, 2)) MB" -ForegroundColor Cyan
        }
    }
    
    # Calculate size differences
    if ($SizeResults.Count -gt 1) {
        $SmallestSize = ($SizeResults | Sort-Object Size | Select-Object -First 1).Size
        $LargestSize = ($SizeResults | Sort-Object Size -Descending | Select-Object -First 1).Size
        $SizeReduction = [math]::Round((1 - $SmallestSize / $LargestSize) * 100, 1)
        
        Write-Host "\nSize optimization can reduce binary size by $SizeReduction%" -ForegroundColor Green
    }
    
    Write-Host ""
}

# Function to generate optimization report
function Generate-OptimizationReport {
    $ReportFile = Join-Path $OutputDir "optimization_report.md"
    
    $Report = @"
# Go Compiler Optimization Analysis Report

Generated on: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
Go Version: $(go version)

## Analysis Summary

This report contains the results of compiler optimization analysis for the Go project.

### Files Analyzed
- Source files in current directory
- Main package: $SourceFile

### Analysis Types Performed
"@
    
    if ($ShowInlining) {
        $Report += "`n- Function inlining analysis"
    }
    if ($ShowEscape) {
        $Report += "`n- Escape analysis"
    }
    if ($ShowAssembly) {
        $Report += "`n- Assembly code generation"
    }
    if ($ShowSSA) {
        $Report += "`n- SSA (Static Single Assignment) analysis"
    }
    
    $Report += @"

### Output Files

All analysis results have been saved to the `$OutputDir` directory:

"@
    
    # List all generated files
    $AnalysisFiles = Get-ChildItem $OutputDir -File
    foreach ($File in $AnalysisFiles) {
        $Report += "`n- ``$($File.Name)`` - $($File.Length) bytes"
    }
    
    $Report += @"

### Recommendations

1. **For Development**: Use debug builds (`-gcflags="-N -l"`) for better debugging experience
2. **For Production**: Use optimized builds (`-gcflags="-l=4" -ldflags="-s -w"`) for best performance and size
3. **For Profiling**: Use default optimizations to get realistic performance data
4. **For Testing**: Use race detection (`-race`) to catch concurrency issues

### Next Steps

1. Review the inlining analysis to identify optimization opportunities
2. Check escape analysis to minimize heap allocations
3. Compare assembly output between optimization levels
4. Run benchmarks to measure actual performance impact

---
*Generated by optimization_analysis.ps1*
"@
    
    $Report | Out-File -FilePath $ReportFile -Encoding UTF8
    Write-Host "Optimization report generated: $ReportFile" -ForegroundColor Green
}

# Main analysis process
Write-Host "=== Go Compiler Optimization Analysis ===" -ForegroundColor Magenta
Write-Host "Source File: $SourceFile" -ForegroundColor Gray
Write-Host "Output Directory: $OutputDir" -ForegroundColor Gray
Write-Host "Go Version: $(go version)" -ForegroundColor Gray
Write-Host ""

# Run selected analyses
if ($ShowInlining) {
    Analyze-Inlining
}

if ($ShowEscape) {
    Analyze-EscapeAnalysis
}

if ($ShowAssembly) {
    Analyze-Assembly
}

if ($ShowSSA) {
    Analyze-SSA
}

# Always run performance and binary size analysis
Analyze-Performance
Analyze-BinarySize

# Generate comprehensive report
Generate-OptimizationReport

Write-Host "=== Analysis Complete ===" -ForegroundColor Green
Write-Host "All analysis results have been saved to: $OutputDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "To view specific analysis results:" -ForegroundColor Yellow
Write-Host "  • Inlining: Get-Content $OutputDir\inlining_analysis.txt" -ForegroundColor Gray
Write-Host "  • Escape Analysis: Get-Content $OutputDir\escape_analysis.txt" -ForegroundColor Gray
Write-Host "  • Assembly: Get-Content $OutputDir\assembly_*.s" -ForegroundColor Gray
Write-Host "  • Benchmarks: Get-Content $OutputDir\benchmark_results.txt" -ForegroundColor Gray
Write-Host "  • Report: Get-Content $OutputDir\optimization_report.md" -ForegroundColor Gray