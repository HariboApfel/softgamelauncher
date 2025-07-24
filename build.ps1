# Game Launcher - Universal Build Script
# Consolidates all build functionality into one script

param(
    [string]$Target = "gui",     # gui, console, cli, or all
    [switch]$Test,               # Run tests after build
    [switch]$Run,                # Run application after build
    [switch]$Clean,              # Clean build artifacts
    [switch]$Help                # Show help
)

function Show-Help {
    Write-Host @"
Game Launcher Build Script

Usage: .\build.ps1 [-Target <target>] [-Test] [-Run] [-Clean] [-Help]

Parameters:
  -Target <target>  What to build:
                    gui     - GUI version (default)
                    console - Console version  
                    cli     - Command-line interface
                    all     - All versions
  -Test            Run tests after building
  -Run             Run the application after building
  -Clean           Clean build artifacts before building
  -Help            Show this help message

Examples:
  .\build.ps1                    # Build GUI version
  .\build.ps1 -Target console    # Build console version
  .\build.ps1 -Target all -Test  # Build all versions and test
  .\build.ps1 -Clean -Run        # Clean, build GUI, and run
"@ -ForegroundColor Cyan
}

function Test-Prerequisites {
    Write-Host "Checking prerequisites..." -ForegroundColor Green
    
    # Check Go
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
        Write-Host "Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
        return $false
    }
    
    $goVersion = go version
    Write-Host "✓ Go found: $goVersion" -ForegroundColor Green
    
    return $true
}

function Find-GCC {
    # Check if GCC is already in PATH
    if (Get-Command gcc -ErrorAction SilentlyContinue) {
        Write-Host "✓ GCC found in PATH" -ForegroundColor Green
        return $true
    }
    
    Write-Host "GCC not found in PATH, searching..." -ForegroundColor Yellow
    
    # Common GCC installation paths
    $gccPaths = @(
        "C:\mingw64\bin",
        "C:\TDM-GCC-64\bin", 
        "C:\msys64\mingw64\bin",
        "C:\msys64\ucrt64\bin",
        "C:\Program Files\mingw-w64\x86_64-8.1.0-posix-seh-rt_v6-rev0\mingw64\bin"
    )
    
    foreach ($path in $gccPaths) {
        $gccExe = Join-Path $path "gcc.exe"
        if (Test-Path $gccExe) {
            Write-Host "✓ Found GCC at: $path" -ForegroundColor Green
            $env:PATH = "$path;$env:PATH"
            return $true
        }
    }
    
    Write-Host "⚠ GCC not found. GUI build may fail." -ForegroundColor Yellow
    Write-Host "Install from: https://jmeubank.github.io/tdm-gcc/" -ForegroundColor Yellow
    return $false
}

function Install-Dependencies {
    Write-Host "Installing dependencies..." -ForegroundColor Green
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: Failed to install dependencies" -ForegroundColor Red
        return $false
    }
    Write-Host "✓ Dependencies installed" -ForegroundColor Green
    return $true
}

function Build-GUI {
    Write-Host "Building GUI version..." -ForegroundColor Green
    
    # Try different build configurations
    $buildConfigs = @(
        @{ Args = @("-ldflags", "-H windowsgui", "-tags", "gles2", "-o", "gamelauncher.exe"); Name = "GLES2" },
        @{ Args = @("-ldflags", "-H windowsgui", "-tags", "no_native_menus", "-o", "gamelauncher.exe"); Name = "No Native Menus" },
        @{ Args = @("-ldflags", "-H windowsgui", "-o", "gamelauncher.exe"); Name = "Standard" },
        @{ Args = @("-o", "gamelauncher.exe"); Name = "Console Window" }
    )
    
    foreach ($config in $buildConfigs) {
        Write-Host "Trying $($config.Name) build..." -ForegroundColor Yellow
        & go build @($config.Args) main.go
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ GUI build successful ($($config.Name))" -ForegroundColor Green
            return $true
        }
    }
    
    Write-Host "✗ All GUI build attempts failed" -ForegroundColor Red
    return $false
}

function Build-Console {
    Write-Host "Building console version..." -ForegroundColor Green
    go build -o gamelauncher_console.exe main_console.go
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Console build successful" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ Console build failed" -ForegroundColor Red
        return $false
    }
}

function Build-CLI {
    Write-Host "Building CLI version..." -ForegroundColor Green
    # Assuming there's a CLI version or use console version
    go build -o gamelauncher_cli.exe main_console.go
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ CLI build successful" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ CLI build failed" -ForegroundColor Red
        return $false
    }
}

function Clean-Artifacts {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Green
    $artifacts = @("gamelauncher.exe", "gamelauncher_console.exe", "gamelauncher_cli.exe", "test_paths.exe", "fix_paths.exe")
    foreach ($artifact in $artifacts) {
        if (Test-Path $artifact) {
            Remove-Item $artifact -Force
            Write-Host "Removed $artifact" -ForegroundColor Yellow
        }
    }
    Write-Host "✓ Clean complete" -ForegroundColor Green
}

function Run-Tests {
    Write-Host "Running tests..." -ForegroundColor Green
    
    # Build and run path tests
    Write-Host "Building path tests..." -ForegroundColor Yellow
    go build -o test_paths.exe test_paths.go
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Running path tests..." -ForegroundColor Yellow
        & .\test_paths.exe
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Path tests passed" -ForegroundColor Green
        } else {
            Write-Host "✗ Path tests failed" -ForegroundColor Red
        }
    }
    
    # Run Go tests
    Write-Host "Running Go tests..." -ForegroundColor Yellow
    go test .\test_basic.go
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Go tests passed" -ForegroundColor Green
    } else {
        Write-Host "✗ Go tests failed" -ForegroundColor Red
    }
}

function Run-Application {
    param([string]$TargetType)
    
    Write-Host "Starting application..." -ForegroundColor Green
    
    switch ($TargetType) {
        "gui" {
            if (Test-Path "gamelauncher.exe") {
                & .\gamelauncher.exe
            }
        }
        "console" {
            if (Test-Path "gamelauncher_console.exe") {
                & .\gamelauncher_console.exe  
            }
        }
        "cli" {
            if (Test-Path "gamelauncher_cli.exe") {
                & .\gamelauncher_cli.exe -help
            }
        }
    }
}

# Main execution
function Main {
    if ($Help) {
        Show-Help
        return
    }
    
    Write-Host "Game Launcher Build Script" -ForegroundColor Cyan
    Write-Host "=========================" -ForegroundColor Cyan
    
    if (-not (Test-Prerequisites)) {
        exit 1
    }
    
    if ($Clean) {
        Clean-Artifacts
    }
    
    if (-not (Install-Dependencies)) {
        exit 1
    }
    
    $success = $false
    
    switch ($Target.ToLower()) {
        "gui" {
            Find-GCC | Out-Null
            $success = Build-GUI
        }
        "console" {
            $success = Build-Console
        }
        "cli" {
            $success = Build-CLI
        }
        "all" {
            Find-GCC | Out-Null
            $guiSuccess = Build-GUI
            $consoleSuccess = Build-Console  
            $cliSuccess = Build-CLI
            $success = $guiSuccess -or $consoleSuccess -or $cliSuccess
            
            Write-Host "`nBuild Summary:" -ForegroundColor Cyan
            Write-Host "GUI:     $(if($guiSuccess){'✓'}else{'✗'})" -ForegroundColor $(if($guiSuccess){'Green'}else{'Red'})
            Write-Host "Console: $(if($consoleSuccess){'✓'}else{'✗'})" -ForegroundColor $(if($consoleSuccess){'Green'}else{'Red'})
            Write-Host "CLI:     $(if($cliSuccess){'✓'}else{'✗'})" -ForegroundColor $(if($cliSuccess){'Green'}else{'Red'})
        }
        default {
            Write-Host "Invalid target: $Target" -ForegroundColor Red
            Show-Help
            exit 1
        }
    }
    
    if ($Test -and $success) {
        Run-Tests
    }
    
    if ($Run -and $success) {
        Run-Application $Target
    }
    
    if ($success) {
        Write-Host "`n✓ Build completed successfully!" -ForegroundColor Green
        Write-Host "Run .\gamelauncher.exe to start the application." -ForegroundColor Cyan
    } else {
        Write-Host "`n✗ Build failed!" -ForegroundColor Red
        exit 1
    }
}

# Execute main function
Main 