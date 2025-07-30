package models

// Settings represents application settings
type Settings struct {
	CheckInterval  int    `json:"check_interval"` // in seconds
	AutoLaunch     bool   `json:"auto_launch"`
	Notifications  bool   `json:"notifications"`
	StartMinimized bool   `json:"start_minimized"`
	Theme          string `json:"theme"`
	LastUsedPath   string `json:"last_used_path"` // Last used directory path for file dialogs
}

// DefaultSettings returns default application settings
func DefaultSettings() *Settings {
	return &Settings{
		CheckInterval:  3600, // 1 hour
		AutoLaunch:     false,
		Notifications:  true,
		StartMinimized: false,
		Theme:          "light",
		LastUsedPath:   "", // Will be set to user's home directory on first use
	}
}
