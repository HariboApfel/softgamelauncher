@echo off
echo Building Game Launcher...

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

echo Building application with OpenGL support...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

go build -ldflags="-H windowsgui" -tags="gles2" -o gamelauncher.exe
if errorlevel 1 (
    echo OpenGL build failed, trying alternative build...
    go build -ldflags="-H windowsgui" -tags="no_native_menus" -o gamelauncher.exe
    if errorlevel 1 (
        echo Error: Build failed
        echo This might be due to missing OpenGL drivers or build tools
        echo Try installing Visual Studio Build Tools or MinGW-w64
        pause
        exit /b 1
    )
)

echo Build successful! Run gamelauncher.exe to start the application.
pause 