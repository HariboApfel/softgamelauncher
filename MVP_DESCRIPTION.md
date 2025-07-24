# Game Launcher MVP Description

## Overview
A cross-platform game launcher built in Go that allows users to import games from local folders, manage game executables, and monitor game sources for updates.

## Core Features

### 1. Game Management
- **Import Games**: Scan and import games from specified folders
- **Game Configuration**: Set executable path, game name, and description
- **Game Launching**: Launch games directly from the launcher
- **Game List**: Display all imported games with basic information

### 2. Game Source Monitoring
- **Web Source Links**: Add web URLs to game sources (GitHub, Steam, etc.)
- **Periodic Monitoring**: Check for updates at configurable intervals
- **Update Notifications**: Alert users when new versions are detected
- **Manual Refresh**: Trigger immediate checks for updates

### 3. User Interface
- **Cross-platform GUI**: Built with Fyne for consistent experience across Windows, macOS, and Linux
- **Simple Layout**: Clean, intuitive interface with game grid/list view
- **Settings Panel**: Configure monitoring intervals and preferences

## Technical Architecture

### Backend (Go)
- **Game Manager**: Handles game import, configuration, and launching
- **Source Monitor**: Web scraping and update detection
- **Data Persistence**: JSON-based configuration storage
- **Process Management**: Game execution and monitoring

### Frontend (Fyne)
- **Main Window**: Game grid/list with launch buttons
- **Game Details**: Modal for editing game properties
- **Settings Dialog**: Configuration options
- **Notifications**: Update alerts and status messages

### Data Structure
```json
{
  "games": [
    {
      "id": "unique_id",
      "name": "Game Name",
      "executable": "/path/to/game.exe",
      "folder": "/path/to/game/folder",
      "source_url": "https://github.com/game/repo",
      "last_check": "2024-01-01T00:00:00Z",
      "last_update": "2024-01-01T00:00:00Z",
      "version": "1.0.0"
    }
  ],
  "settings": {
    "check_interval": 3600,
    "auto_launch": false,
    "notifications": true
  }
}
```

## MVP Scope
- Basic game import from folders
- Simple game launching
- Web source URL storage
- Manual update checking
- Basic notifications
- Cross-platform compatibility

## Future Enhancements (Post-MVP)
- Automatic update downloads
- Game categories and tags
- Screenshot management
- Play time tracking
- Cloud sync
- Plugin system for different game sources 