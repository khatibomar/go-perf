#!/usr/bin/env pwsh
# PowerShell script to check compilation of all Go files in the project

Write-Host "=== Go Build Checker ===" -ForegroundColor Green
Write-Host "Scanning for Go files and testing compilation..." -ForegroundColor Yellow

$projectRoot = Get-Location
$totalFiles = 0
$successfulBuilds = 0
$failedBuilds = 0
$errors = @()

# Find all Go files (excluding vendor directories and test files for main compilation)
$goFiles = Get-ChildItem -Path . -Recurse -Filter "*.go" | Where-Object {
    $_.FullName -notmatch "vendor" -and 
    $_.Name -notmatch "_test\.go$"
}

# Group Go files by directory to handle packages correctly
$packageDirs = $goFiles | Group-Object DirectoryName

Write-Host "Found $($goFiles.Count) Go files in $($packageDirs.Count) directories" -ForegroundColor Cyan
Write-Host ""

foreach ($packageDir in $packageDirs) {
    $dirPath = $packageDir.Name
    $relativeDir = $dirPath.Replace($projectRoot.Path, "").TrimStart("\\")
    $goFilesInDir = $packageDir.Group
    
    Write-Host "Checking directory: $relativeDir" -ForegroundColor Blue
    
    # Change to the directory
    Push-Location $dirPath
    
    try {
        # Check if there's a go.mod file in this directory or parent directories
        $hasGoMod = $false
        $currentDir = Get-Location
        while ($currentDir.Path -ne $currentDir.Root) {
            if (Test-Path (Join-Path $currentDir.Path "go.mod")) {
                $hasGoMod = $true
                break
            }
            $currentDir = $currentDir.Parent
        }
        
        if (-not $hasGoMod) {
            Write-Host "  ⚠️  No go.mod found - skipping package build" -ForegroundColor Yellow
            continue
        }
        
        # Try to build all Go files in the directory as a package
        $buildOutput = & go build . 2>&1
        $buildExitCode = $LASTEXITCODE
        
        if ($buildExitCode -eq 0) {
            Write-Host "  ✅ Package build successful" -ForegroundColor Green
            $successfulBuilds++
        } else {
            Write-Host "  ❌ Package build failed" -ForegroundColor Red
            $failedBuilds++
            $errors += [PSCustomObject]@{
                Directory = $relativeDir
                Error = $buildOutput -join "`n"
                Type = "Package Build"
            }
            
            # If package build fails, try individual files
            Write-Host "  Trying individual file builds..." -ForegroundColor Yellow
            
            foreach ($goFile in $goFilesInDir) {
                $fileName = $goFile.Name
                Write-Host "    Testing: $fileName" -ForegroundColor Gray
                
                $fileOutput = & go build $fileName 2>&1
                $fileExitCode = $LASTEXITCODE
                
                if ($fileExitCode -eq 0) {
                    Write-Host "      ✅ $fileName builds successfully" -ForegroundColor Green
                } else {
                    Write-Host "      ❌ $fileName failed to build" -ForegroundColor Red
                    $errors += [PSCustomObject]@{
                        Directory = $relativeDir
                        File = $fileName
                        Error = $fileOutput -join "`n"
                        Type = "Individual File"
                    }
                }
                $totalFiles++
            }
        }
    }
    catch {
        Write-Host "  ❌ Error processing directory: $_" -ForegroundColor Red
        $errors += [PSCustomObject]@{
            Directory = $relativeDir
            Error = $_.Exception.Message
            Type = "Directory Error"
        }
        $failedBuilds++
    }
    finally {
        Pop-Location
    }
    
    Write-Host ""
}

# Summary
Write-Host "=== Build Summary ===" -ForegroundColor Green
Write-Host "Total directories checked: $($packageDirs.Count)" -ForegroundColor Cyan
Write-Host "Successful builds: $successfulBuilds" -ForegroundColor Green
Write-Host "Failed builds: $failedBuilds" -ForegroundColor Red

if ($errors.Count -gt 0) {
    Write-Host ""
    Write-Host "=== Detailed Error Report ===" -ForegroundColor Red
    
    foreach ($errorItem in $errors) {
        Write-Host "Directory: $($errorItem.Directory)" -ForegroundColor Yellow
        if ($errorItem.File) {
            Write-Host "File: $($errorItem.File)" -ForegroundColor Yellow
        }
        Write-Host "Type: $($errorItem.Type)" -ForegroundColor Yellow
        Write-Host "Error:" -ForegroundColor Red
        Write-Host $errorItem.Error -ForegroundColor Gray
        Write-Host "" + ("-" * 50) + ""
    }
}

# Clean up any generated executable files
Write-Host "\n=== Cleaning up executable files ===" -ForegroundColor Yellow
try {
    $exeFiles = Get-ChildItem -Recurse -Include *.exe
    if ($exeFiles.Count -gt 0) {
        Write-Host "Found $($exeFiles.Count) executable files to clean up" -ForegroundColor Cyan
        $exeFiles | Remove-Item -Force -Verbose
        Write-Host "✅ All executable files cleaned up" -ForegroundColor Green
    } else {
        Write-Host "No executable files found to clean up" -ForegroundColor Green
    }
}
catch {
    Write-Host "⚠️  Error cleaning up executable files: $_" -ForegroundColor Yellow
}

if ($failedBuilds -eq 0) {
    Write-Host "🎉 All builds successful!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "⚠️  Some builds failed. See details above." -ForegroundColor Red
    exit 1
}
