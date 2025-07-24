package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"gamelauncher/models"
)

// Manager handles data persistence
type Manager struct {
	dataPath string
}

// NewManager creates a new storage manager
func NewManager() *Manager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	
	dataPath := filepath.Join(homeDir, ".gamelauncher")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		// Fallback to current directory
		dataPath = "."
	}
	
	return &Manager{
		dataPath: dataPath,
	}
}

// SaveGames saves the games list to disk
func (m *Manager) SaveGames(games []*models.Game) error {
	data, err := json.MarshalIndent(games, "", "  ")
	if err != nil {
		return err
	}
	
	filePath := filepath.Join(m.dataPath, "games.json")
	return os.WriteFile(filePath, data, 0644)
}

// LoadGames loads the games list from disk
func (m *Manager) LoadGames() ([]*models.Game, error) {
	filePath := filepath.Join(m.dataPath, "games.json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.Game{}, nil
		}
		return nil, err
	}
	
	var games []*models.Game
	if err := json.Unmarshal(data, &games); err != nil {
		return nil, err
	}
	
	// Clean up paths for existing games
	for _, game := range games {
		game.Executable = m.cleanPath(game.Executable)
		game.Folder = m.cleanPath(game.Folder)
	}
	
	return games, nil
}

// SaveSettings saves the settings to disk
func (m *Manager) SaveSettings(settings *models.Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	
	filePath := filepath.Join(m.dataPath, "settings.json")
	return os.WriteFile(filePath, data, 0644)
}

// LoadSettings loads the settings from disk
func (m *Manager) LoadSettings() (*models.Settings, error) {
	filePath := filepath.Join(m.dataPath, "settings.json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return models.DefaultSettings(), nil
		}
		return nil, err
	}
	
	var settings models.Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	
	return &settings, nil
}

// cleanPath cleans and normalizes a file path
func (m *Manager) cleanPath(path string) string {
	// Remove surrounding quotes
	path = strings.Trim(path, `"'`)
	
	// Normalize path separators
	path = filepath.Clean(path)
	
	return path
} 