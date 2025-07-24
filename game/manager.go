package game

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"gamelauncher/models"
)

// Manager handles game operations
type Manager struct{}

// NewManager creates a new game manager
func NewManager() *Manager {
	return &Manager{}
}

// LaunchGame launches a game executable
func (m *Manager) LaunchGame(game *models.Game) error {
	if !game.IsInstalled {
		return fmt.Errorf("game is not installed")
	}
	
	// Clean the executable path (remove quotes and normalize)
	executable := m.cleanPath(game.Executable)
	
	// Check if executable exists
	if _, err := os.Stat(executable); os.IsNotExist(err) {
		return fmt.Errorf("executable not found: %s", executable)
	}
	
	// Launch the game
	cmd := exec.Command(executable)
	
	// Set working directory if available
	if game.Folder != "" {
		cmd.Dir = m.cleanPath(game.Folder)
	}
	
	return cmd.Start()
}

// ScanFolder scans a folder for potential games
func (m *Manager) ScanFolder(folderPath string) ([]*models.Game, error) {
	var games []*models.Game
	
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check if file is an executable
		if m.isExecutable(path) {
			game := m.createGameFromPath(path)
			if game != nil {
				games = append(games, game)
			}
		}
		
		return nil
	})
	
	return games, err
}

// isExecutable checks if a file is an executable
func (m *Manager) isExecutable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch runtime.GOOS {
	case "windows":
		return ext == ".exe" || ext == ".bat" || ext == ".cmd"
	case "darwin":
		return ext == ".app" || ext == "" // macOS apps can have no extension
	default: // Linux
		return ext == "" || ext == ".sh"
	}
}

// createGameFromPath creates a game from an executable path
func (m *Manager) createGameFromPath(path string) *models.Game {
	// Clean the path first
	cleanPath := m.cleanPath(path)
	dir := filepath.Dir(cleanPath)
	name := filepath.Base(dir)
	
	// Clean up the name
	name = strings.TrimSpace(name)
	if name == "" {
		name = filepath.Base(cleanPath)
	}
	
	// Remove file extension from name
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	
	return models.NewGame(name, cleanPath, dir)
}

// cleanPath cleans and normalizes a file path
func (m *Manager) cleanPath(path string) string {
	// Remove surrounding quotes
	path = strings.Trim(path, `"'`)
	
	// Normalize path separators
	path = filepath.Clean(path)
	
	// Convert to absolute path if it's not already
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}
	
	return path
}

// FindExecutableInFolder searches for executables in a folder
func (m *Manager) FindExecutableInFolder(folderPath string) ([]string, error) {
	var executables []string
	
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && m.isExecutable(path) {
			executables = append(executables, path)
		}
		
		return nil
	})
	
	return executables, err
} 