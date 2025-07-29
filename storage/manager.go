package storage

import (
	"encoding/json"
	"fmt"
	"gamelauncher/models"
	"os"
	"path/filepath"
	"strings"
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
	fmt.Printf("DEBUG: SaveGames called with %d games\n", len(games))
	for i, game := range games {
		fmt.Printf("DEBUG: Game %d: %s (SourceURL: %s)\n", i+1, game.Name, game.SourceURL)
	}

	data, err := json.MarshalIndent(games, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(m.dataPath, "games.json")
	fmt.Printf("DEBUG: Saving games to %s\n", filePath)
	return os.WriteFile(filePath, data, 0644)
}

// LoadGames loads the games list from disk
func (m *Manager) LoadGames() ([]*models.Game, error) {
	filePath := filepath.Join(m.dataPath, "games.json")
	fmt.Printf("DEBUG: Loading games from %s\n", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("DEBUG: Games file does not exist, returning empty list\n")
			return []*models.Game{}, nil
		}
		return nil, err
	}

	var games []*models.Game
	if err := json.Unmarshal(data, &games); err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG: Loaded %d games from file\n", len(games))
	for i, game := range games {
		fmt.Printf("DEBUG: Loaded game %d: %s (SourceURL: %s)\n", i+1, game.Name, game.SourceURL)
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
