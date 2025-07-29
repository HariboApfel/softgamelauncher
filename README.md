# Game Launcher

A cross-platform game launcher built in Go that allows you to import games from local folders, manage game executables, and monitor game sources for updates with real-time version comparison.

## Features

- **Game Management**: Import games from folders, set executables, and manage game properties
- **Real-time Version Comparison**: Automatic version checking with visual status indicators
- **Cross-platform**: Works on Windows, macOS, and Linux  
- **Source Monitoring**: Monitor GitHub repositories, F95zone, and other web sources for updates
- **Automatic Updates**: Periodic checking for updates with configurable intervals
- **Game Search**: Automatic game link discovery via F95Zone API
- **Command Line Support**: Launch games from command line or console interface
- **Simple UI**: Clean, intuitive interface built with Fyne
- **Data Persistence**: Saves game list and settings automatically

## Quick Start

### Windows (GUI Version)
```bash
# Download and run the build script
.\build.ps1
# or
build.bat

# Run the application
.\gamelauncher.exe
```

### Windows (Console Version - No OpenGL dependencies)
```bash
build_console.bat
.\gamelauncher_console.exe
```

### macOS/Linux
```bash
./build.sh
./gamelauncher
```

## Requirements

- **Go 1.21+**: Download from [golang.org](https://golang.org/dl/)
- **Windows GUI**: C compiler (TDM-GCC, MinGW-w64, or Visual Studio Build Tools)
- **Git**: For cloning the repository

## Installation

### 1. Clone Repository
```bash
git clone <repository-url>
cd gamelauncher
```

### 2. Install Dependencies (Windows GUI only)

**Option A: TDM-GCC (Easiest)**
1. Download from [jmeubank.github.io/tdm-gcc](https://jmeubank.github.io/tdm-gcc/)
2. Run installer and check "Add to PATH"
3. Restart terminal

**Option B: MSYS2**
1. Download from [msys2.org](https://www.msys2.org/)
2. Install and run: `pacman -S mingw-w64-x86_64-gcc`
3. Add `C:\msys64\mingw64\bin` to PATH

**Option C: Visual Studio Build Tools**
1. Download from Microsoft
2. Install with "C++ build tools" workload

### 3. Build Application

**Windows:**
```bash
# Automatic build (tries multiple configurations)
.\build.ps1

# Or specific versions
build.bat              # GUI version
build_console.bat      # Console version
```

**macOS/Linux:**
```bash
./build.sh
```

### 4. Run Application
```bash
# GUI versions
.\gamelauncher.exe     # Windows
./gamelauncher         # macOS/Linux

# Console version
.\gamelauncher_console.exe

# Command line usage
.\gamelauncher.exe -list          # List games
.\gamelauncher.exe -game 1        # Launch game 1
.\gamelauncher.exe -help          # Show help
```

## Usage

### Adding Games

1. **Import from Folder**: Click folder icon → scan directory for games
2. **Manual Addition**: Click plus icon → add game details manually
   - **Automatic Link Discovery**: When you enter a game name and select an executable, the system automatically searches F95Zone for matching links
   - **Smart Auto-fill**: If a good match is found (>70% confidence), the source URL is automatically filled
   - **Manual Selection**: For lower confidence matches, you can choose from multiple results
   - **Manual Search**: Click "Search for Link" button to manually trigger a search

### Game Management

- **Launch**: Click "Launch" button or use command line
- **Edit**: Click "Edit" to modify properties, version settings
- **Delete**: Select game → click delete button (🗑️) → confirm
- **Source URL**: Add GitHub, F95zone, or other web sources
- **Search**: Click search button (🔍) to automatically find game links on F95Zone

### Game Search

The launcher includes automatic game link discovery via the F95Zone API:

#### GUI Search
1. Select a game from the list
2. Click the search button (🔍) in the toolbar
3. Review search results with match scores
4. Select the best match to automatically fill in the source URL

#### Automatic Search (Add Game Dialog)
1. Click the "+" button to add a new game
2. Enter the game name
3. Browse and select the executable file
4. **Automatic**: The system automatically searches for matching links
5. **Auto-fill**: If a good match is found (>70%), the source URL is filled automatically
6. **Manual Selection**: For lower confidence matches, choose from the results dialog
7. **Manual Trigger**: Click "Search for Link" button to manually search anytime

#### Command Line Search
```bash
# Search for a game by name
gamelauncher.exe -search "My Pig Princess"
```

#### Search Features
- **Smart Matching**: Intelligent game name matching with score calculation
- **Multiple Results**: Shows all matches with confidence scores
- **Automatic URL Filling**: Updates game source URL with selected result
- **F95Zone Integration**: Direct integration with F95Zone RSS API

#### Example Search
```bash
gamelauncher.exe -search "My Pig Princess"
```
Output:
```
Searching for 'My Pig Princess' on F95Zone...

Found 3 matches for 'My Pig Princess':
==========================================
1. [85.7%] My Pig Princess [v0.514.0.3]
   Link: https://f95zone.to/threads/my-pig-princess-v0-514-0-3.12345/
   Description: A visual novel game...

2. [42.9%] Princess Pig Adventure
   Link: https://f95zone.to/threads/princess-pig-adventure.67890/

Best match: My Pig Princess [v0.514.0.3] (85.7%)
Link: https://f95zone.to/threads/my-pig-princess-v0-514-0-3.12345/
```

### Version Monitoring

#### Real-time Status Indicators
- **`1.2.3 ✓`** - Up to date (green)
- **`1.2.4 ⚠ NEW`** - Update available (yellow)
- **`1.2.1 ⚠ DIFF`** - Version mismatch (yellow)
- **`Checking...`** - Currently checking
- **`Error`** - Failed to fetch
- **`No source`** - No URL configured

#### Automatic Detection
- **F95zone**: Automatically extracts version from threads
- **GitHub**: Uses GitHub API for release information
- **Other sites**: Configurable CSS selectors and regex patterns

#### Manual Configuration
1. Edit game → Advanced settings
2. Set **Version Selector** (CSS): `.version`, `#version`, `h1`
3. Set **Version Pattern** (Regex): `v(\d+\.\d+\.\d+)`, `(\d+\.\d+\.\d+)`
4. Set **Current Version**: Your installed version

### Settings

- **Check Interval**: How often to check for updates (seconds)
- **Notifications**: Enable/disable update notifications
- Access via gear icon in toolbar

## Version Configuration Examples

### GitHub Release
```
URL: https://github.com/user/game/releases
Selector: h1
Pattern: v(\d+\.\d+\.\d+)
```

### F95zone (Automatic)
```
URL: https://f95zone.to/threads/game-name.123/
# No configuration needed - automatic detection
```

### Custom Website
```
URL: https://game.com/download
Selector: .version-info
Pattern: Version: (\d+\.\d+\.\d+)
```

## Command Line Interface

### Available Commands
```bash
# List all games
gamelauncher.exe -list

# Launch specific game
gamelauncher.exe -game 1

# Search for game on F95Zone
gamelauncher.exe -search "Game Name"

# Show help
gamelauncher.exe -help

# Run GUI (default)
gamelauncher.exe
```

### Integration Examples
```bash
# Desktop shortcuts
gamelauncher.exe -game 1

# Batch scripts
@echo off
gamelauncher.exe -game 2

# Task scheduler
schtasks /create /tn "Launch Game" /tr "gamelauncher.exe -game 1"
```

## Project Structure

```
gamelauncher/
├── main.go              # GUI application entry point
├── main_console.go      # Console application entry point
├── models/              # Data models
│   ├── game.go         # Game structure and methods
│   └── settings.go     # Settings structure
├── storage/            # Data persistence
│   └── manager.go      # JSON file storage
├── game/               # Game operations
│   └── manager.go      # Game launching and scanning
├── monitor/            # Update monitoring
│   └── source.go       # Web source monitoring
├── search/             # Game search functionality
│   └── service.go      # F95Zone API integration
├── ui/                 # User interface
│   ├── main_window.go  # Main application window
│   └── colored_label.go # UI components
├── build.ps1           # Windows build script
├── build.sh            # Unix/Linux build script
├── build_console.bat   # Console version build
└── README.md           # This file
```

## Data Storage

**Location:**
- **Windows**: `%USERPROFILE%\.gamelauncher\`
- **macOS/Linux**: `~/.gamelauncher/`

**Files:**
- `games.json`: List of imported games
- `settings.json`: Application settings

## Troubleshooting

### Windows Build Issues

**Problem**: `build constraints exclude all Go files in go-gl`
**Solution**: Install C compiler (see Installation section)

**Quick fix**: Use console version
```bash
build_console.bat  # No OpenGL dependencies
```

### Common Issues

1. **Games not launching**: Check executable path is correct
2. **Update checks failing**: Verify internet connection and URLs
3. **Permission errors**: Ensure read/write access to data directory
4. **Path issues with quotes**: Run `fix_paths.bat`

### Path Issues

If you see errors like: `exec: "\"C:\\path\\game.exe\"": file does not exist`

**Solutions:**
1. Run path fixer: `fix_paths.bat`
2. Edit game and remove quotes from executable path
3. Rebuild application for latest path handling

### Debug Mode
```bash
go run main.go -debug
```

## Testing Build Scripts

Test the build system:
```bash
# Test path cleaning
test_paths.bat

# Fix path issues
fix_paths.bat

# Verify build
build.ps1
```

## Building for Distribution

### Windows
```bash
# GUI (no console window)
go build -ldflags="-H windowsgui" -o gamelauncher.exe

# Console version
go build -o gamelauncher_console.exe main_console.go
```

### macOS
```bash
go build -o gamelauncher
```

### Linux
```bash
go build -o gamelauncher
```

## Dependencies

- **Fyne v2**: Cross-platform GUI framework
- **goquery**: HTML parsing for web scraping
- **uuid**: Unique identifier generation

## Advanced Features

### F95zone Integration
- Automatic URL detection for `f95zone.to`
- Specialized version parsing: `0.514.0.3 with RTP`
- No manual configuration required

### GitHub Integration  
- Automatic repository detection
- Release API integration
- Tag and release version tracking

### Custom Site Support
- Configurable CSS selectors
- Regex pattern matching
- Real-time version comparison

## Future Enhancements

- Automatic update downloads
- Game categories and tags
- Screenshot management
- Play time tracking
- Cloud sync
- Plugin system for different game sources
- Steam/Epic Games integration

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the MIT License.

---

## Quick Reference

### First Time Setup
```bash
git clone <repo>
cd gamelauncher
.\build.ps1        # Windows
./build.sh         # macOS/Linux
```

### Daily Usage
```bash
.\gamelauncher.exe           # Start GUI
.\gamelauncher.exe -list     # List games
.\gamelauncher.exe -game 1   # Launch game 1
```

### Troubleshooting
```bash
build_console.bat           # No OpenGL issues
fix_paths.bat              # Fix path problems
test_paths.bat             # Test path handling
``` 