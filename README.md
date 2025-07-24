# Game Launcher

A cross-platform game launcher built in Go that allows you to import games from local folders, manage game executables, and monitor game sources for updates.

## Features

- **Game Management**: Import games from folders, set executables, and manage game properties
- **Cross-platform**: Works on Windows, macOS, and Linux
- **Source Monitoring**: Monitor GitHub repositories and other web sources for game updates
- **Automatic Updates**: Periodic checking for updates with configurable intervals
- **Simple UI**: Clean, intuitive interface built with Fyne
- **Data Persistence**: Saves game list and settings automatically

## Requirements

- Go 1.21 or later
- Git (for cloning the repository)

## Installation

### Option 1: GUI Version (Requires C Compiler on Windows)

1. Clone the repository:
```bash
git clone <repository-url>
cd gamelauncher
```

2. **For Windows users**: Install a C compiler (required for OpenGL support):
   - **MinGW-w64**: Download from [https://www.mingw-w64.org/](https://www.mingw-w64.org/)
   - **TDM-GCC**: Download from [https://jmeubank.github.io/tdm-gcc/](https://jmeubank.github.io/tdm-gcc/)
   - **Visual Studio Build Tools**: Download from Microsoft

3. Build the application:
   ```bash
   # Windows (use provided scripts)
   build_windows.bat
   # or
   .\build.ps1
   
   # macOS/Linux
   go build -o gamelauncher
   ```

4. Run the application:
   ```bash
   ./gamelauncher  # or gamelauncher.exe on Windows
   ```

### Option 2: Console Version (No GUI Dependencies)

If you encounter OpenGL build issues, use the console version:

1. Clone the repository:
```bash
git clone <repository-url>
cd gamelauncher
```

2. Build the console version:
   ```bash
   # Windows
   build_console.bat
   
   # macOS/Linux
   go build -o gamelauncher_console main_console.go
   ```

3. Run the console application:
   ```bash
   ./gamelauncher_console  # or gamelauncher_console.exe on Windows
   ```

The console version provides all the same functionality as the GUI version but through a text-based interface.

## Usage

### Adding Games

1. **Import from Folder**: Click the folder icon in the toolbar to scan a folder for games
2. **Manual Addition**: Click the plus icon to manually add a game with custom settings

### Game Management

- **Launch**: Click the "Launch" button next to any game to start it
- **Edit**: Click the "Edit" button to modify game properties
- **Source URL**: Add GitHub URLs or other web sources to monitor for updates

### Settings

- **Check Interval**: Set how often to check for updates (in seconds)
- **Notifications**: Enable/disable update notifications
- Access settings via the gear icon in the toolbar

### Update Monitoring

- **Manual Check**: Click the refresh icon to check for updates immediately
- **Automatic**: Updates are checked periodically based on your settings
- **GitHub Support**: Automatically detects and monitors GitHub repositories

## Project Structure

```
gamelauncher/
├── main.go              # Application entry point
├── models/              # Data models
│   ├── game.go         # Game structure and methods
│   └── settings.go     # Settings structure
├── storage/            # Data persistence
│   └── manager.go      # JSON file storage
├── game/               # Game operations
│   └── manager.go      # Game launching and scanning
├── monitor/            # Update monitoring
│   └── source.go       # Web source monitoring
├── ui/                 # User interface
│   └── main_window.go  # Main application window
└── README.md           # This file
```

## Data Storage

The application stores data in:
- **Windows**: `%USERPROFILE%\.gamelauncher\`
- **macOS/Linux**: `~/.gamelauncher/`

Files:
- `games.json`: List of imported games
- `settings.json`: Application settings

## Building for Distribution

### Windows
```bash
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

## Dependencies

- **Fyne v2**: Cross-platform GUI framework
- **goquery**: HTML parsing for web scraping
- **uuid**: Unique identifier generation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the MIT License.

## Troubleshooting

### Common Issues

1. **Games not launching**: Ensure the executable path is correct and the file exists
2. **Update checks failing**: Check your internet connection and verify source URLs
3. **Permission errors**: Ensure the application has read/write access to the data directory
4. **Path issues with quotes**: If you see errors like "file does not exist" with quoted paths, run the path fixer

### Path Issues

If you encounter errors like:
```
Error launching game: exec: "\"C:\\Users\\alexa\\Downloads\\MyPigPrincess-0.9.0-pc\\MyPigPrincess-0.9.0-pc\\MyPigPrincess.exe\"": file does not exist
```

This means the executable path has extra quotes. To fix this:

1. **Run the path fixer**:
   ```cmd
   fix_paths.bat
   ```

2. **Or manually edit** the game and remove quotes from the executable path

3. **Rebuild the application** to get the latest path handling improvements

### Debug Mode

Run with debug logging:
```bash
go run main.go -debug
```

## Future Enhancements

- Automatic update downloads
- Game categories and tags
- Screenshot management
- Play time tracking
- Cloud sync
- Plugin system for different game sources
- Game library import from Steam, Epic, etc. 