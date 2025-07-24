package models

// Settings represents application settings
type Settings struct {
	CheckInterval  int  `json:"check_interval"`  // in seconds
	AutoLaunch     bool `json:"auto_launch"`
	Notifications  bool `json:"notifications"`
	StartMinimized bool `json:"start_minimized"`
	Theme          string `json:"theme"`
}

// DefaultSettings returns default application settings
func DefaultSettings() *Settings {
	return &Settings{
		CheckInterval:  3600, // 1 hour
		AutoLaunch:     false,
		Notifications:  true,
		StartMinimized: false,
		Theme:          "light",
	}
} 