package models

import (
	"time"

	"github.com/google/uuid"
)

// Game represents a game in the launcher
type Game struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Executable  string    `json:"executable"`
	Folder      string    `json:"folder"`
	SourceURL   string    `json:"source_url"`
	LastCheck   time.Time `json:"last_check"`
	LastUpdate  time.Time `json:"last_update"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	IconPath    string    `json:"icon_path"`
	ImagePath   string    `json:"image_path"` // Path to downloaded game image
	IsInstalled bool      `json:"is_installed"`

	// Version checking configuration
	VersionSelector string `json:"version_selector"` // CSS selector for version element
	VersionPattern  string `json:"version_pattern"`  // Regex pattern to extract version
	CurrentVersion  string `json:"current_version"`  // Current version for comparison
}

// NewGame creates a new game instance with a unique ID
func NewGame(name, executable, folder string) *Game {
	return &Game{
		ID:          uuid.New().String(),
		Name:        name,
		Executable:  executable,
		Folder:      folder,
		LastCheck:   time.Now(),
		LastUpdate:  time.Now(),
		IsInstalled: true,
	}
}

// UpdateInfo updates the game's update information
func (g *Game) UpdateInfo(version string) {
	g.Version = version
	g.LastUpdate = time.Now()
}

// MarkChecked updates the last check time
func (g *Game) MarkChecked() {
	g.LastCheck = time.Now()
}
