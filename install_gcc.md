# Installing GCC for Game Launcher

## Option 1: TDM-GCC (Recommended - Easiest)

1. Go to https://jmeubank.github.io/tdm-gcc/
2. Click "Download" and choose the latest version
3. Run the installer
4. **Important**: During installation, make sure to check "Add to PATH"
5. Restart your terminal/PowerShell after installation

## Option 2: MinGW-w64

1. Go to https://www.mingw-w64.org/
2. Download the installer
3. Install with default settings
4. Add the bin directory to your PATH (usually `C:\mingw64\bin`)

## Option 3: MSYS2

1. Go to https://www.msys2.org/
2. Download and install MSYS2
3. Open MSYS2 terminal and run: `pacman -S mingw-w64-x86_64-gcc`
4. Add `C:\msys64\mingw64\bin` to your PATH

## Option 4: Visual Studio Build Tools

1. Download Visual Studio Build Tools from Microsoft
2. Install with "C++ build tools" workload
3. Use the Developer Command Prompt

## After Installation

1. Restart your terminal/PowerShell
2. Run: `gcc --version` to verify installation
3. Run: `.\build.ps1` to build the Game Launcher

## Quick Test

After installing GCC, you can test if it's working by running:
```powershell
gcc --version
```

If you see version information, GCC is properly installed and you can run the build script. 