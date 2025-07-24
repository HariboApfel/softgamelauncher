@echo off
echo Game Launcher - Path Fixer
echo =========================

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    pause
    exit /b 1
)

echo Building path fixer...
go build -o fix_paths.exe fix_paths.go
if errorlevel 1 (
    echo Error: Build failed
    pause
    exit /b 1
)

echo Running path fixer...
fix_paths.exe

echo.
echo Path fixer completed!
pause 