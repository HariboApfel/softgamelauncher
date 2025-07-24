# Game Launcher - Utility Script
# Consolidates path testing and fixing functionality

param(
    [string]$Action = "help",  # help, test, fix
    [switch]$Help
)

function Show-Help {
    Write-Host @"
Game Launcher Utility Script

Usage: .\utils.ps1 [-Action <action>] [-Help]

Actions:
  help    Show this help message (default)
  test    Test path handling functionality  
  fix     Fix quoted path issues in game database

Examples:
  .\utils.ps1 -Action test    # Test path handling
  .\utils.ps1 -Action fix     # Fix path issues
  .\utils.ps1 -Help           # Show help
"@ -ForegroundColor Cyan
}

function Test-Paths {
    Write-Host "Testing path handling functionality..." -ForegroundColor Green
    
    # Check if Go is installed
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
        return $false
    }
    
    # Build path test
    Write-Host "Building path test..." -ForegroundColor Yellow
    go build -o test_paths.exe test_paths.go
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: Failed to build path test" -ForegroundColor Red
        return $false
    }
    
    # Run path test
    Write-Host "Running path tests..." -ForegroundColor Yellow
    & .\test_paths.exe
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Path tests passed" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ Path tests failed" -ForegroundColor Red
        return $false
    }
}

function Fix-Paths {
    Write-Host "Fixing quoted path issues..." -ForegroundColor Green
    
    # Check if Go is installed
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
        return $false
    }
    
    # Build path fixer
    Write-Host "Building path fixer..." -ForegroundColor Yellow
    go build -o fix_paths.exe fix_paths.go
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: Failed to build path fixer" -ForegroundColor Red
        return $false
    }
    
    # Run path fixer
    Write-Host "Running path fixer..." -ForegroundColor Yellow
    & .\fix_paths.exe
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Path fixing completed" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ Path fixing failed" -ForegroundColor Red
        return $false
    }
}

function Clean-Utils {
    Write-Host "Cleaning utility artifacts..." -ForegroundColor Green
    $artifacts = @("test_paths.exe", "fix_paths.exe")
    foreach ($artifact in $artifacts) {
        if (Test-Path $artifact) {
            Remove-Item $artifact -Force
            Write-Host "Removed $artifact" -ForegroundColor Yellow
        }
    }
    Write-Host "✓ Clean complete" -ForegroundColor Green
}

# Main execution
function Main {
    if ($Help) {
        Show-Help
        return
    }
    
    Write-Host "Game Launcher Utility Script" -ForegroundColor Cyan
    Write-Host "===========================" -ForegroundColor Cyan
    
    switch ($Action.ToLower()) {
        "help" {
            Show-Help
        }
        "test" {
            if (Test-Paths) {
                Write-Host "`n✓ Path testing completed successfully!" -ForegroundColor Green
            } else {
                Write-Host "`n✗ Path testing failed!" -ForegroundColor Red
                exit 1
            }
            Clean-Utils
        }
        "fix" {
            if (Fix-Paths) {
                Write-Host "`n✓ Path fixing completed successfully!" -ForegroundColor Green
            } else {
                Write-Host "`n✗ Path fixing failed!" -ForegroundColor Red
                exit 1
            }
            Clean-Utils
        }
        default {
            Write-Host "Invalid action: $Action" -ForegroundColor Red
            Show-Help
            exit 1
        }
    }
}

# Execute main function
Main