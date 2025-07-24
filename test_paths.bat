@echo off
echo Testing Path Cleaning Functionality...

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    pause
    exit /b 1
)

echo Building path test...
go build -o test_paths.exe test_paths.go
if errorlevel 1 (
    echo Error: Build failed
    pause
    exit /b 1
)

echo Running path test...
test_paths.exe

echo.
echo Path test completed!
pause 