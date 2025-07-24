@echo off
echo Building Game Launcher for Windows...

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

echo Attempting build with different configurations...

REM Try build with gles2 tags first
echo Trying build with GLES2 support...
go build -ldflags="-H windowsgui" -tags="gles2" -o gamelauncher_gles2.exe
if not errorlevel 1 (
    echo GLES2 build successful!
    copy gamelauncher_gles2.exe gamelauncher.exe
    del gamelauncher_gles2.exe
    goto :success
)

REM Try build with no_native_menus
echo GLES2 failed, trying no_native_menus...
go build -ldflags="-H windowsgui" -tags="no_native_menus" -o gamelauncher_native.exe
if not errorlevel 1 (
    echo Native build successful!
    copy gamelauncher_native.exe gamelauncher.exe
    del gamelauncher_native.exe
    goto :success
)

REM Try build without any special tags
echo Native build failed, trying standard build...
go build -ldflags="-H windowsgui" -o gamelauncher.exe
if not errorlevel 1 (
    echo Standard build successful!
    goto :success
)

REM Try build with console window (for debugging)
echo Standard build failed, trying console build...
go build -o gamelauncher_console.exe
if not errorlevel 1 (
    echo Console build successful!
    copy gamelauncher_console.exe gamelauncher.exe
    del gamelauncher_console.exe
    echo Note: This version will show a console window
    goto :success
)

echo All build attempts failed!
echo.
echo Possible solutions:
echo 1. Install Visual Studio Build Tools
echo 2. Install MinGW-w64
echo 3. Update your graphics drivers
echo 4. Try running: go install fyne.io/fyne/v2/cmd/fyne@latest
echo.
pause
exit /b 1

:success
echo.
echo Build successful! Run gamelauncher.exe to start the application.
echo.
pause 