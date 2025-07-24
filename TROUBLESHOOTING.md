# Troubleshooting Guide

## Common Build Issues and Solutions

### 1. "go: command not found" Error

**Problem**: Go is not installed or not in your system PATH.

**Solutions**:

#### Install Go on Windows:
1. **Download Go** from [https://golang.org/dl/](https://golang.org/dl/)
2. **Run the installer** and follow the prompts
3. **Restart your terminal/command prompt**
4. **Verify installation**:
   ```cmd
   go version
   ```

#### If Go is installed but still not found:
1. **Check PATH environment variable**:
   - Open System Properties → Advanced → Environment Variables
   - Look for Go in the PATH (usually `C:\Go\bin`)
   - Add it if missing: `C:\Go\bin`

2. **Alternative installation methods**:
   ```cmd
   # Using Chocolatey
   choco install golang
   
   # Using Scoop
   scoop install go
   ```

### 2. OpenGL Build Constraints Error

**Problem**: 
```
build constraints exclude all Go files in ...\go-gl\gl@v0.0.0-20211210172815-726fda9656d6\v3.2-core\gl
```

**Root Cause**: This happens because the Fyne GUI framework requires OpenGL support, which needs a C compiler on Windows.

**Solutions**:

#### Option 1: Install MinGW-w64 (Recommended)
1. **Download MSYS2** from [https://www.msys2.org/](https://www.msys2.org/)
2. **Install MSYS2** and open MSYS2 terminal
3. **Install MinGW-w64**:
   ```bash
   pacman -S mingw-w64-x86_64-gcc
   pacman -S mingw-w64-x86_64-toolchain
   ```
4. **Add to PATH**: Add `C:\msys64\mingw64\bin` to your system PATH
5. **Verify installation**:
   ```cmd
   gcc --version
   ```

#### Option 2: Install TDM-GCC
1. **Download TDM-GCC** from [https://jmeubank.github.io/tdm-gcc/](https://jmeubank.github.io/tdm-gcc/)
2. **Run the installer** and follow the prompts
3. **Restart your terminal**

#### Option 3: Install Visual Studio Build Tools
1. **Download Visual Studio Build Tools** from Microsoft
2. **Install C++ build tools** during installation
3. **Use Developer Command Prompt** or set up environment variables

#### Option 4: Use Alternative Build Tags
Try building with different OpenGL configurations:
```cmd
# Try GLES2 (OpenGL ES 2.0)
go build -tags="gles2" -o gamelauncher.exe

# Try without native menus
go build -tags="no_native_menus" -o gamelauncher.exe

# Try console version (shows console window)
go build -o gamelauncher.exe
```

### 3. Using the Provided Build Scripts

The project includes several build scripts that try different configurations automatically:

#### Windows Batch Script:
```cmd
build_windows.bat
```

#### PowerShell Script:
```powershell
.\build.ps1
```

#### Manual Build Commands:
```cmd
# Install dependencies
go mod tidy

# Try different build configurations
go build -ldflags="-H windowsgui" -tags="gles2" -o gamelauncher.exe
go build -ldflags="-H windowsgui" -tags="no_native_menus" -o gamelauncher.exe
go build -ldflags="-H windowsgui" -o gamelauncher.exe
```

### 4. Alternative: Use Fyne's Build Tool

Fyne provides its own build tool that handles many issues automatically:

```cmd
# Install Fyne CLI
go install fyne.io/fyne/v2/cmd/fyne@latest

# Build using Fyne
fyne build -o gamelauncher.exe
```

### 5. Alternative: Use Different GUI Framework

If OpenGL issues persist, consider using a different GUI framework:

#### Option A: Web-based GUI (Easier to build)
- Use a web framework like Gin + HTML templates
- No OpenGL dependencies
- Cross-platform via web browser

#### Option B: Console-based Interface
- Remove GUI dependencies
- Use command-line interface
- Still provides all core functionality

### 6. Environment Setup Checklist

Before building, ensure you have:

- [ ] Go 1.21+ installed and in PATH
- [ ] C compiler installed (MinGW-w64, TDM-GCC, or Visual Studio Build Tools)
- [ ] CGO_ENABLED=1 (usually set automatically)
- [ ] Internet connection for downloading dependencies

### 7. Verification Steps

After installation, verify everything works:

```cmd
# Check Go
go version

# Check C compiler
gcc --version

# Check environment
echo %CGO_ENABLED%

# Test basic Go build
go build -o test.exe main.go
```

### 8. Getting Help

If you're still having issues:

1. **Check Go installation**: `go version`
2. **Check C compiler**: `gcc --version`
3. **Check environment variables**: `echo %PATH%`
4. **Try minimal test**: Create a simple Go program and build it
5. **Search for similar issues**: Many others have faced this problem

### 9. Quick Fix for Testing

If you just want to test the application logic without GUI:

1. **Comment out GUI imports** in `ui/main_window.go`
2. **Create a simple console version** in `main.go`
3. **Test core functionality** without GUI dependencies

This will let you verify that the game management, storage, and monitoring features work correctly. 