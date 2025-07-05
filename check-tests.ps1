# Go Performance Test Runner Script
# This script runs tests across all directories in the Go performance codebase

Write-Host "🧪 Go Performance Test Runner" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan
Write-Host ""

# Initialize counters
$totalDirectories = 0
$successfulTests = 0
$failedTests = 0
$errorDetails = @()

# Function to run tests in a directory
function Test-GoDirectory {
    param(
        [string]$DirectoryPath,
        [string]$RelativePath
    )
    
    Write-Host "Checking directory: $RelativePath" -ForegroundColor Yellow
    
    # Check if directory has Go files
    $goFiles = Get-ChildItem -Path $DirectoryPath -Filter "*.go" -File
    if ($goFiles.Count -eq 0) {
        Write-Host "  ⏭️  No Go files found, skipping" -ForegroundColor Gray
        return $true
    }
    
    # Check if directory has test files
    $testFiles = Get-ChildItem -Path $DirectoryPath -Filter "*_test.go" -File
    if ($testFiles.Count -eq 0) {
        Write-Host "  ⏭️  No test files found, skipping" -ForegroundColor Gray
        return $true
    }
    
    try {
        # Change to the directory and run go test
        Push-Location $DirectoryPath
        $testOutput = & go test -v . 2>&1
        $testExitCode = $LASTEXITCODE
        Pop-Location
        
        if ($testExitCode -eq 0) {
            Write-Host "  ✅ Tests passed" -ForegroundColor Green
            return $true
        } else {
            Write-Host "  ❌ Tests failed" -ForegroundColor Red
            
            # Store error details
            $script:errorDetails += [PSCustomObject]@{
                Directory = $RelativePath
                Type = "Test Execution"
                Error = ($testOutput -join "`n")
            }
            return $false
        }
    }
    catch {
        Write-Host "  ❌ Test execution error" -ForegroundColor Red
        
        # Store error details
        $script:errorDetails += [PSCustomObject]@{
            Directory = $RelativePath
            Type = "Test Execution Error"
            Error = $_.Exception.Message
        }
        return $false
    }
}

# Function to run benchmarks in a directory
function Benchmark-GoDirectory {
    param(
        [string]$DirectoryPath,
        [string]$RelativePath
    )
    
    # Check if directory has benchmark files
    $benchmarkPattern = Get-ChildItem -Path $DirectoryPath -Filter "*_test.go" -File | 
        ForEach-Object { Select-String -Path $_.FullName -Pattern "func Benchmark" -Quiet }
    
    if ($benchmarkPattern -contains $true) {
        Write-Host "  🏃 Running benchmarks..." -ForegroundColor Magenta
        try {
            Push-Location $DirectoryPath
            $benchOutput = & go test -bench=. -benchmem . 2>&1
            $benchExitCode = $LASTEXITCODE
            Pop-Location
            
            if ($benchExitCode -eq 0) {
                Write-Host "  ✅ Benchmarks completed" -ForegroundColor Green
                return $true
            } else {
                Write-Host "  ⚠️  Benchmark issues" -ForegroundColor Yellow
                return $false
            }
        }
        catch {
            Write-Host "  ⚠️  Benchmark execution error" -ForegroundColor Yellow
            return $false
        }
    }
    return $true
}

# Get all directories that might contain Go code
$directories = Get-ChildItem -Path . -Directory -Recurse | Where-Object {
    $_.Name -match "exercise-" -or $_.Name -eq "go-perf"
}

# Add current directory if it has Go files
$currentDirGoFiles = Get-ChildItem -Path . -Filter "*.go" -File
if ($currentDirGoFiles.Count -gt 0) {
    $directories = @(Get-Item .) + $directories
}

foreach ($dir in $directories) {
    $totalDirectories++
    $relativePath = Resolve-Path -Path $dir.FullName -Relative
    
    # Test the directory
    $testSuccess = Test-GoDirectory -DirectoryPath $dir.FullName -RelativePath $relativePath
    
    if ($testSuccess) {
        $successfulTests++
        
        # Run benchmarks if tests passed
        Benchmark-GoDirectory -DirectoryPath $dir.FullName -RelativePath $relativePath
    } else {
        $failedTests++
    }
    
    Write-Host ""
}

# Print summary
Write-Host "=== Test Summary ===" -ForegroundColor Cyan
Write-Host "Total directories checked: $totalDirectories" -ForegroundColor White
Write-Host "Successful tests: $successfulTests" -ForegroundColor Green
Write-Host "Failed tests: $failedTests" -ForegroundColor Red

# Print detailed error report if there are failures
if ($failedTests -gt 0) {
    Write-Host ""
    Write-Host "=== Detailed Error Report ===" -ForegroundColor Red
    foreach ($errorDetail in $errorDetails) {
        Write-Host "Directory: $($errorDetail.Directory)" -ForegroundColor Yellow
        Write-Host "Type: $($errorDetail.Type)" -ForegroundColor Yellow
        Write-Host "Error:" -ForegroundColor Yellow
        Write-Host $errorDetail.Error -ForegroundColor Red
        Write-Host " + $('-' * 50) + " -ForegroundColor Gray
    }
    Write-Host "⚠️  Some tests failed. See details above." -ForegroundColor Red
    exit 1
} else {
    Write-Host "🎉 All tests successful!" -ForegroundColor Green
    exit 0
}