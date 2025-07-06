# Build Comparison Script for Go Compiler Optimization Exercise
# This script builds the same Go program with different optimization flags
# and compares the resulting binaries and performance

param(
    [string]$SourceDir = ".",
    [string]$OutputDir = "builds",
    [switch]$Verbose,
    [switch]$RunBenchmarks,
    [switch]$CleanFirst
)

# Ensure we're in the right directory
if (!(Test-Path "go.mod")) {
    Write-Error "go.mod not found. Please run this script from the project root."
    exit 1
}

# Clean previous builds if requested
if ($CleanFirst -and (Test-Path $OutputDir)) {
    Write-Host "Cleaning previous builds..." -ForegroundColor Yellow
    Remove-Item $OutputDir -Recurse -Force
}

# Create output directory
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

# Define build configurations
$BuildConfigs = @(
    @{
        Name = "debug"
        Description = "Debug build with no optimizations"
        GCFlags = "-N -l"
        LDFlags = ""
        Tags = ""
    },
    @{
        Name = "default"
        Description = "Default build with standard optimizations"
        GCFlags = ""
        LDFlags = ""
        Tags = ""
    },
    @{
        Name = "aggressive"
        Description = "Aggressive optimization with maximum inlining"
        GCFlags = "-l=4"
        LDFlags = ""
        Tags = ""
    },
    @{
        Name = "size-optimized"
        Description = "Size-optimized build"
        GCFlags = ""
        LDFlags = "-s -w"
        Tags = ""
    },
    @{
        Name = "production"
        Description = "Production build with all optimizations"
        GCFlags = "-l=4"
        LDFlags = "-s -w"
        Tags = ""
    },
    @{
        Name = "race"
        Description = "Race detection build"
        GCFlags = "-N -l"
        LDFlags = ""
        Tags = "-race"
    }
)

# Function to build with specific flags
function Build-WithFlags {
    param(
        [string]$Name,
        [string]$Description,
        [string]$GCFlags,
        [string]$LDFlags,
        [string]$Tags
    )
    
    $OutputFile = Join-Path $OutputDir "optimization-demo-$Name.exe"
    $BuildArgs = @("build")
    
    if ($GCFlags) {
        $BuildArgs += "-gcflags=$GCFlags"
    }
    
    if ($LDFlags) {
        $BuildArgs += "-ldflags=$LDFlags"
    }
    
    if ($Tags) {
        $BuildArgs += $Tags
    }
    
    $BuildArgs += @("-o", $OutputFile)
    
    Write-Host "Building $Name ($Description)..." -ForegroundColor Green
    if ($Verbose) {
        Write-Host "Command: go $($BuildArgs -join ' ')" -ForegroundColor Cyan
    }
    
    $StartTime = Get-Date
    $Process = Start-Process -FilePath "go" -ArgumentList $BuildArgs -Wait -PassThru -NoNewWindow
    $BuildTime = (Get-Date) - $StartTime
    
    if ($Process.ExitCode -eq 0) {
        $FileInfo = Get-Item $OutputFile
        Write-Host "  ✓ Build successful" -ForegroundColor Green
        Write-Host "  ✓ Build time: $($BuildTime.TotalSeconds.ToString('F2'))s" -ForegroundColor Green
        Write-Host "  ✓ Binary size: $([math]::Round($FileInfo.Length / 1MB, 2)) MB" -ForegroundColor Green
        
        return @{
            Success = $true
            BuildTime = $BuildTime.TotalSeconds
            BinarySize = $FileInfo.Length
            Path = $OutputFile
        }
    } else {
        Write-Host "  ✗ Build failed" -ForegroundColor Red
        return @{
            Success = $false
            BuildTime = $BuildTime.TotalSeconds
            BinarySize = 0
            Path = $null
        }
    }
}

# Function to run performance test
function Test-Performance {
    param(
        [string]$BinaryPath,
        [string]$Name
    )
    
    if (!(Test-Path $BinaryPath)) {
        return $null
    }
    
    Write-Host "Testing performance of $Name..." -ForegroundColor Yellow
    
    # Test different optimization types
    $TestCases = @("inlining", "escape", "deadcode", "flags")
    $Results = @{}
    
    foreach ($TestCase in $TestCases) {
        $StartTime = Get-Date
        $Process = Start-Process -FilePath $BinaryPath -ArgumentList @("-optimization", $TestCase, "-size", "1000", "-iterations", "100") -Wait -PassThru -NoNewWindow -RedirectStandardOutput "temp_output.txt"
        $ExecutionTime = (Get-Date) - $StartTime
        
        if ($Process.ExitCode -eq 0) {
            $Results[$TestCase] = $ExecutionTime.TotalMilliseconds
            if ($Verbose) {
                Write-Host "  $TestCase: $($ExecutionTime.TotalMilliseconds.ToString('F2'))ms" -ForegroundColor Cyan
            }
        }
        
        # Clean up temp file
        if (Test-Path "temp_output.txt") {
            Remove-Item "temp_output.txt" -Force
        }
    }
    
    return $Results
}

# Main build process
Write-Host "=== Go Compiler Optimization Build Comparison ===" -ForegroundColor Magenta
Write-Host "Source Directory: $SourceDir" -ForegroundColor Gray
Write-Host "Output Directory: $OutputDir" -ForegroundColor Gray
Write-Host ""

$BuildResults = @()

foreach ($Config in $BuildConfigs) {
    $Result = Build-WithFlags -Name $Config.Name -Description $Config.Description -GCFlags $Config.GCFlags -LDFlags $Config.LDFlags -Tags $Config.Tags
    $Result.Name = $Config.Name
    $Result.Description = $Config.Description
    $BuildResults += $Result
    Write-Host ""
}

# Performance testing
if ($RunBenchmarks) {
    Write-Host "=== Performance Testing ===" -ForegroundColor Magenta
    
    foreach ($Result in $BuildResults) {
        if ($Result.Success -and $Result.Path) {
            $PerfResults = Test-Performance -BinaryPath $Result.Path -Name $Result.Name
            $Result.Performance = $PerfResults
        }
        Write-Host ""
    }
}

# Generate comparison report
Write-Host "=== Build Comparison Report ===" -ForegroundColor Magenta
Write-Host ""

# Table header
Write-Host ("Name".PadRight(15) + "Status".PadRight(10) + "Build Time".PadRight(12) + "Binary Size".PadRight(12) + "Description") -ForegroundColor White
Write-Host ("-" * 80) -ForegroundColor Gray

# Table rows
foreach ($Result in $BuildResults) {
    $Status = if ($Result.Success) { "✓ Success" } else { "✗ Failed" }
    $BuildTime = "$($Result.BuildTime.ToString('F2'))s"
    $BinarySize = if ($Result.BinarySize -gt 0) { "$([math]::Round($Result.BinarySize / 1MB, 2)) MB" } else { "N/A" }
    
    $Color = if ($Result.Success) { "Green" } else { "Red" }
    Write-Host ($Result.Name.PadRight(15) + $Status.PadRight(10) + $BuildTime.PadRight(12) + $BinarySize.PadRight(12) + $Result.Description) -ForegroundColor $Color
}

Write-Host ""

# Size comparison
$SuccessfulBuilds = $BuildResults | Where-Object { $_.Success }
if ($SuccessfulBuilds.Count -gt 1) {
    Write-Host "=== Size Comparison ===" -ForegroundColor Magenta
    $SmallestBuild = $SuccessfulBuilds | Sort-Object BinarySize | Select-Object -First 1
    $LargestBuild = $SuccessfulBuilds | Sort-Object BinarySize -Descending | Select-Object -First 1
    
    Write-Host "Smallest binary: $($SmallestBuild.Name) ($([math]::Round($SmallestBuild.BinarySize / 1MB, 2)) MB)" -ForegroundColor Green
    Write-Host "Largest binary: $($LargestBuild.Name) ($([math]::Round($LargestBuild.BinarySize / 1MB, 2)) MB)" -ForegroundColor Yellow
    
    $SizeDifference = $LargestBuild.BinarySize - $SmallestBuild.BinarySize
    $SizeRatio = [math]::Round($LargestBuild.BinarySize / $SmallestBuild.BinarySize, 2)
    Write-Host "Size difference: $([math]::Round($SizeDifference / 1MB, 2)) MB ($($SizeRatio)x larger)" -ForegroundColor Cyan
    Write-Host ""
}

# Performance comparison
if ($RunBenchmarks) {
    Write-Host "=== Performance Comparison ===" -ForegroundColor Magenta
    
    $PerfBuilds = $BuildResults | Where-Object { $_.Success -and $_.Performance }
    if ($PerfBuilds.Count -gt 0) {
        $TestCases = @("inlining", "escape", "deadcode", "flags")
        
        foreach ($TestCase in $TestCases) {
            Write-Host "$TestCase optimization:" -ForegroundColor Yellow
            
            foreach ($Build in $PerfBuilds) {
                if ($Build.Performance[$TestCase]) {
                    Write-Host "  $($Build.Name): $($Build.Performance[$TestCase].ToString('F2'))ms" -ForegroundColor Cyan
                }
            }
            Write-Host ""
        }
    }
}

# Recommendations
Write-Host "=== Recommendations ===" -ForegroundColor Magenta
Write-Host "• Use 'debug' build for development and debugging" -ForegroundColor Green
Write-Host "• Use 'production' build for deployment (smallest size + best performance)" -ForegroundColor Green
Write-Host "• Use 'race' build for testing concurrent code" -ForegroundColor Green
Write-Host "• Use 'aggressive' build for CPU-intensive applications" -ForegroundColor Green
Write-Host ""

# Export results to JSON for further analysis
$JsonOutput = Join-Path $OutputDir "build_comparison.json"
$BuildResults | ConvertTo-Json -Depth 3 | Out-File $JsonOutput -Encoding UTF8
Write-Host "Detailed results exported to: $JsonOutput" -ForegroundColor Gray

Write-Host "Build comparison completed!" -ForegroundColor Green