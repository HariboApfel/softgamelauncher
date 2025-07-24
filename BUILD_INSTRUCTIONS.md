# Build Instructions

## Prerequisites

### Installing Go

1. **Download Go** from [https://golang.org/dl/](https://golang.org/dl/)
2. **Install Go** following the official installation guide for your platform:
   - **Windows**: Run the MSI installer and follow the prompts
   - **macOS**: Run the PKG installer or use Homebrew: `brew install go`
   - **Linux**: Use your package manager or download the binary

3. **Verify Installation**:
   ```bash
   go version
   ```

### Windows-Specific Requirements

**For Windows users, you also need a C compiler for OpenGL support:**

1. **Option 1: Install MinGW-w64** (Recommended)
   - Download from [https://www.mingw-w64.org/](https://www.mingw-w64.org/)
   - Or use MSYS2: [https://www.msys2.org/](https://www.msys2.org/)
   - Add MinGW-w64 to your PATH

2. **Option 2: Install Visual Studio Build Tools**
   - Download from Microsoft Visual Studio website
   - Install the C++ build tools

3. **Option 3: Install TDM-GCC**
   - Download from [https://jmeubank.github.io/tdm-gcc/](https://jmeubank.github.io/tdm-gcc/)
   - Simple installer for Windows

### Setting up the Environment

1. **Set GOPATH** (if not already set):
   - **Windows**: Add to System Environment Variables
   - **macOS/Linux**: Add to `~/.bashrc` or `~/.zshrc`:
     ```bash
     export GOPATH=$HOME/go
     export PATH=$PATH:$GOPATH/bin
     ```

## Building the Game Launcher

### Windows (Recommended Method)

1. **Use the provided build script**:
   ```cmd
   build_windows.bat
   ```
   or
   ```powershell
   .\build.ps1
   ```

2. **Manual build** (if scripts don't work):
   ```cmd
   go mod tidy
   go build -ldflags="-H windowsgui" -tags="gles2" -o gamelauncher.exe
   ```

### macOS/Linux

1. **Navigate to the project directory**:
   ```bash
   cd gamelauncher
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Build the application**:
   ```bash
   go build -o gamelauncher
   ```

4. **Run the application**:
   ```bash
   ./gamelauncher
   ```

## Platform-Specific Builds

### Windows
```bash
# Try different build configurations
go build -ldflags="-H windowsgui" -tags="gles2" -o gamelauncher.exe
go build -ldflags="-H windowsgui" -tags="no_native_menus" -o gamelauncher.exe
go build -ldflags="-H windowsgui" -o gamelauncher.exe
```

### macOS
```bash
go build -o gamelauncher
```

### Linux
```bash
go build -o gamelauncher
```

## Troubleshooting

### Common Issues

1. **"go: command not found"**:
   - Go is not installed or not in PATH
   - Reinstall Go and ensure it's added to PATH

2. **Module dependencies not found**:
   - Run `go mod tidy` to download dependencies
   - Check your internet connection

3. **OpenGL build constraints error on Windows**:
   - Install a C compiler (MinGW-w64, Visual Studio Build Tools, or TDM-GCC)
   - Try different build tags: `gles2`, `no_native_menus`
   - Use the provided build scripts that try multiple configurations

4. **Build errors**:
   - Ensure you're using Go 1.21 or later
   - Check that all files are in the correct directories

### Windows OpenGL Issues

If you encounter OpenGL-related build errors:

1. **Install GCC compiler**:
   ```cmd
   # Using Chocolatey
   choco install mingw
   
   # Or download MinGW-w64 manually
   ```

2. **Try alternative build tags**:
   ```cmd
   go build -tags="gles2" -o gamelauncher.exe
   go build -tags="no_native_menus" -o gamelauncher.exe
   ```

3. **Use Fyne's build tool**:
   ```cmd
   go install fyne.io/fyne/v2/cmd/fyne@latest
   fyne build -o gamelauncher.exe
   ```

### Alternative: Using Go Modules

If you encounter module issues, try:
```bash
go mod init gamelauncher
go mod tidy
go build
```

## Development

For development, you can run the application directly:
```bash
go run main.go
```

This will compile and run the application in one step. 