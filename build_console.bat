@echo off
echo Building Game Launcher Console Version...

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

echo Installing dependencies...
go mod tidy
if errorlevel 1 (
    echo Error: Failed to install dependencies
    pause
    exit /b 1
)

echo Building console application...
go build -o gamelauncher_console.exe main_console.go
if errorlevel 1 (
    echo Error: Build failed
    pause
    exit /b 1
)

echo.
echo Console build successful!
echo Run gamelauncher_console.exe to start the console version.
echo.
echo This version includes all core features:
echo - Game management
echo - Folder scanning
echo - Update monitoring
echo - Settings management
echo.
echo Note: This is a console application (no GUI).
echo.
pause 