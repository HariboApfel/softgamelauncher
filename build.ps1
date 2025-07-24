# Game Launcher Build Script for Windows
# This script adds GCC to PATH and builds the application

Write-Host "Building Game Launcher..." -ForegroundColor Green

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Common GCC installation paths to check
$gccPaths = @(
    "C:\mingw64\bin",
    "C:\mingw-w64\x86_64-8.1.0-posix-seh-rt_v6-rev0\mingw64\bin",
    "C:\TDM-GCC-64\bin",
    "C:\msys64\mingw64\bin",
    "C:\msys64\ucrt64\bin",
    "C:\msys64\clang64\bin",
    "C:\msys64\clang32\bin",
    "C:\Program Files\mingw-w64\x86_64-8.1.0-posix-seh-rt_v6-rev0\mingw64\bin",
    "C:\Program Files (x86)\mingw-w64\i686-8.1.0-posix-dwarf-rt_v6-rev0\mingw32\bin"
)

# Check if GCC is already in PATH
$gccInPath = $false
if (Get-Command gcc -ErrorAction SilentlyContinue) {
    Write-Host "GCC found in PATH" -ForegroundColor Green
    $gccInPath = $true
} else {
    Write-Host "GCC not found in PATH, searching for installation..." -ForegroundColor Yellow
    
    # Search for GCC in common installation paths
    foreach ($path in $gccPaths) {
        if (Test-Path $path) {
            $gccExe = Join-Path $path "gcc.exe"
            if (Test-Path $gccExe) {
                Write-Host "Found GCC at: $path" -ForegroundColor Green
                $env:PATH = "$path;$env:PATH"
                $gccInPath = $true
                break
            }
        }
    }
}

if (-not $gccInPath) {
    Write-Host "Error: GCC not found. Please install one of the following:" -ForegroundColor Red
    Write-Host "  - MinGW-w64: https://www.mingw-w64.org/" -ForegroundColor Yellow
    Write-Host "  - TDM-GCC: https://jmeubank.github.io/tdm-gcc/" -ForegroundColor Yellow
    Write-Host "  - MSYS2: https://www.msys2.org/" -ForegroundColor Yellow
    Write-Host "  - Visual Studio Build Tools with C++ workload" -ForegroundColor Yellow
    exit 1
}

# Verify GCC is working
Write-Host "Verifying GCC installation..." -ForegroundColor Green
try {
    $gccVersion = & gcc --version 2>&1 | Select-Object -First 1
    Write-Host "GCC Version: $gccVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: GCC verification failed" -ForegroundColor Red
    exit 1
}

# Install dependencies
Write-Host "Installing dependencies..." -ForegroundColor Green
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to install dependencies" -ForegroundColor Red
    exit 1
}

# Build application (without console window)
Write-Host "Building application..." -ForegroundColor Green
go build -ldflags "-H windowsgui" -o gamelauncher.exe main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Build successful!" -ForegroundColor Green
Write-Host "Run .\gamelauncher.exe to start the application." -ForegroundColor Cyan

# Optional: Run the application
$runApp = Read-Host "Would you like to run the application now? (y/n)"
if ($runApp -eq "y" -or $runApp -eq "Y") {
    Write-Host "Starting Game Launcher..." -ForegroundColor Green
    & .\gamelauncher.exe
} 