package steam

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gamelauncher/models"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Manager handles Steam integration operations
type Manager struct{}

// NewManager creates a new Steam manager
func NewManager() *Manager {
	return &Manager{}
}

// SteamShortcut represents a Steam non-Steam shortcut
type SteamShortcut struct {
	AppID              uint32
	AppName            string
	Exe                string
	StartDir           string
	Icon               string
	ShortcutPath       string
	LaunchOptions      string
	IsHidden           bool
	AllowDesktopConfig bool
	AllowOverlay       bool
	OpenVR             bool
	Devkit             bool
	DevkitGameID       string
	DevkitOverrideAppID uint32
	LastPlayTime       uint32
	FlatpakAppID       string
	Tags               []string
}

// AddGameToSteam adds a game to Steam as a non-Steam shortcut
func (m *Manager) AddGameToSteam(game *models.Game) error {
	// Find Steam installation
	steamPath, err := m.findSteamPath()
	if err != nil {
		return fmt.Errorf("failed to find Steam installation: %w", err)
	}

	// Find user data directory
	userDataPath, err := m.findUserDataPath(steamPath)
	if err != nil {
		return fmt.Errorf("failed to find Steam user data: %w", err)
	}

	// Create shortcut from game
	shortcut := m.createShortcutFromGame(game)
	
	// Check if game already exists in Steam
	shortcutsPath := filepath.Join(userDataPath, "config", "shortcuts.vdf")
	isUpdate, err := m.checkGameExistsInSteam(shortcutsPath, game)
	if err != nil {
		log.Printf("Warning: Could not check for existing shortcuts: %v", err)
	}
	
	if isUpdate {
		log.Printf("Updating existing Steam shortcut for game: %s (AppID: %d)", game.Name, shortcut.AppID)
	} else {
		log.Printf("Adding new Steam shortcut for game: %s (AppID: %d)", game.Name, shortcut.AppID)
	}

	// Add shortcut to Steam
	err = m.addShortcutToFile(shortcutsPath, shortcut)
	if err != nil {
		return fmt.Errorf("failed to add shortcut to Steam: %w", err)
	}

	return nil
}

// CheckGameExistsInSteam checks if a game already exists in Steam as a shortcut
func (m *Manager) CheckGameExistsInSteam(game *models.Game) (bool, error) {
	// Find Steam installation
	steamPath, err := m.findSteamPath()
	if err != nil {
		return false, fmt.Errorf("failed to find Steam installation: %w", err)
	}

	// Find user data directory
	userDataPath, err := m.findUserDataPath(steamPath)
	if err != nil {
		return false, fmt.Errorf("failed to find Steam user data: %w", err)
	}

	shortcutsPath := filepath.Join(userDataPath, "config", "shortcuts.vdf")
	return m.checkGameExistsInSteam(shortcutsPath, game)
}

// checkGameExistsInSteam internal function to check if game exists in shortcuts file
func (m *Manager) checkGameExistsInSteam(shortcutsPath string, game *models.Game) (bool, error) {
	// Read existing shortcuts
	shortcuts, err := m.readShortcutsFile(shortcutsPath)
	if err != nil {
		// If file doesn't exist, game doesn't exist
		return false, nil
	}

	// Generate the AppID and normalized values for the game
	appID := m.generateAppID(game.Name, game.Executable)
	normalizedName := m.normalizeName(game.Name)
	normalizedExe := m.normalizePath(game.Executable)

	// Check if shortcut already exists
	for _, existing := range shortcuts {
		// Primary check: same AppID
		if existing.AppID == appID {
			return true, nil
		}
		
		// Secondary check: same normalized name and executable
		existingNormalizedName := m.normalizeName(existing.AppName)
		existingNormalizedExe := m.normalizePath(existing.Exe)
		
		if existingNormalizedName == normalizedName && existingNormalizedExe == normalizedExe {
			return true, nil
		}
	}

	return false, nil
}

// findSteamPath attempts to find the Steam installation directory
func (m *Manager) findSteamPath() (string, error) {
	var possiblePaths []string

	switch runtime.GOOS {
	case "windows":
		possiblePaths = []string{
			"C:\\Program Files (x86)\\Steam",
			"C:\\Program Files\\Steam",
			filepath.Join(os.Getenv("PROGRAMFILES"), "Steam"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Steam"),
		}
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		possiblePaths = []string{
			filepath.Join(homeDir, "Library", "Application Support", "Steam"),
			"/Applications/Steam.app",
		}
	default: // Linux
		homeDir, _ := os.UserHomeDir()
		possiblePaths = []string{
			filepath.Join(homeDir, ".steam", "steam"),
			filepath.Join(homeDir, ".local", "share", "Steam"),
			"/usr/share/steam",
			"/opt/steam",
		}
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Steam installation not found")
}

// findUserDataPath finds the Steam userdata directory for the current user
func (m *Manager) findUserDataPath(steamPath string) (string, error) {
	userDataDir := filepath.Join(steamPath, "userdata")
	
	// Check if userdata directory exists
	if _, err := os.Stat(userDataDir); os.IsNotExist(err) {
		return "", fmt.Errorf("Steam userdata directory not found")
	}

	// Find user directories (they are numbered)
	entries, err := os.ReadDir(userDataDir)
	if err != nil {
		return "", fmt.Errorf("failed to read userdata directory: %w", err)
	}

	// Find the most recently modified user directory (most likely the current user)
	var latestUserDir string
	var latestModTime int64

	for _, entry := range entries {
		if entry.IsDir() {
			userPath := filepath.Join(userDataDir, entry.Name())
			configPath := filepath.Join(userPath, "config")
			
			// Check if this user directory has a config folder
			if stat, err := os.Stat(configPath); err == nil {
				if stat.ModTime().Unix() > latestModTime {
					latestModTime = stat.ModTime().Unix()
					latestUserDir = userPath
				}
			}
		}
	}

	if latestUserDir == "" {
		return "", fmt.Errorf("no valid Steam user found")
	}

	return latestUserDir, nil
}

// createShortcutFromGame creates a Steam shortcut from a game model
func (m *Manager) createShortcutFromGame(game *models.Game) *SteamShortcut {
	// Generate AppID
	appID := m.generateAppID(game.Name, game.Executable)

	// Determine start directory
	startDir := game.Folder
	if startDir == "" {
		startDir = filepath.Dir(game.Executable)
	}

	// Use game image as icon if available
	icon := game.IconPath
	if icon == "" && game.ImagePath != "" {
		icon = game.ImagePath
	}

	return &SteamShortcut{
		AppID:              appID,
		AppName:            game.Name,
		Exe:                game.Executable,
		StartDir:           startDir,
		Icon:               icon,
		ShortcutPath:       "",
		LaunchOptions:      "",
		IsHidden:           false,
		AllowDesktopConfig: true,
		AllowOverlay:       true,
		OpenVR:             false,
		Devkit:             false,
		DevkitGameID:       "",
		DevkitOverrideAppID: 0,
		LastPlayTime:       0,
		FlatpakAppID:       "",
		Tags:               []string{},
	}
}

// generateAppID generates a unique AppID for the shortcut based on name and executable
func (m *Manager) generateAppID(appName, exe string) uint32 {
	// Normalize name and executable path to ensure consistency
	normalizedName := m.normalizeName(appName)
	normalizedExe := m.normalizePath(exe)
	
	// Steam uses CRC32 of name + exe + null terminator, with high bit set
	input := normalizedName + normalizedExe + "\x00"
	crc := crc32.ChecksumIEEE([]byte(input))
	return crc | 0x80000000
}

// normalizeName normalizes a game name for consistent AppID generation
func (m *Manager) normalizeName(name string) string {
	// Remove extra whitespace and convert to lowercase for consistency
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return name
}

// normalizePath normalizes a file path for consistent AppID generation
func (m *Manager) normalizePath(path string) string {
	// Remove surrounding quotes
	path = strings.Trim(path, `"'`)
	
	// Normalize path separators and clean the path
	path = filepath.Clean(path)
	
	// Convert to absolute path if possible for consistency
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}
	
	// Convert to lowercase for case-insensitive comparison
	path = strings.ToLower(path)
	
	return path
}

// addShortcutToFile adds a shortcut to the shortcuts.vdf file
func (m *Manager) addShortcutToFile(shortcutsPath string, shortcut *SteamShortcut) error {
	// Read existing shortcuts
	shortcuts, err := m.readShortcutsFile(shortcutsPath)
	if err != nil {
		// If file doesn't exist, create empty list
		shortcuts = []*SteamShortcut{}
	}

	// Check if shortcut already exists (by AppID or by normalized name/exe combination)
	existingIndex := -1
	normalizedName := m.normalizeName(shortcut.AppName)
	normalizedExe := m.normalizePath(shortcut.Exe)
	
	for i, existing := range shortcuts {
		// Primary check: same AppID
		if existing.AppID == shortcut.AppID {
			existingIndex = i
			break
		}
		
		// Secondary check: same normalized name and executable
		existingNormalizedName := m.normalizeName(existing.AppName)
		existingNormalizedExe := m.normalizePath(existing.Exe)
		
		if existingNormalizedName == normalizedName && existingNormalizedExe == normalizedExe {
			existingIndex = i
			break
		}
	}

	if existingIndex >= 0 {
		// Update existing shortcut with new settings
		shortcuts[existingIndex] = shortcut
	} else {
		// Add new shortcut
		shortcuts = append(shortcuts, shortcut)
	}

	// Write shortcuts back to file
	return m.writeShortcutsFile(shortcutsPath, shortcuts)
}

// readShortcutsFile reads shortcuts from the shortcuts.vdf file
func (m *Manager) readShortcutsFile(filePath string) ([]*SteamShortcut, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return m.parseShortcutsVDF(data)
}

// writeShortcutsFile writes shortcuts to the shortcuts.vdf file
func (m *Manager) writeShortcutsFile(filePath string, shortcuts []*SteamShortcut) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data := m.buildShortcutsVDF(shortcuts)

	return os.WriteFile(filePath, data, 0644)
}

// parseShortcutsVDF parses the binary VDF format
func (m *Manager) parseShortcutsVDF(data []byte) ([]*SteamShortcut, error) {
	var shortcuts []*SteamShortcut
	reader := bytes.NewReader(data)

	// Read root type and key
	var rootType byte
	if err := binary.Read(reader, binary.LittleEndian, &rootType); err != nil {
		return nil, err
	}

	rootKey, err := m.readNullTerminatedString(reader)
	if err != nil {
		return nil, err
	}

	if rootType != 0x00 || rootKey != "shortcuts" {
		return nil, fmt.Errorf("invalid shortcuts file format")
	}

	// Read shortcuts
	for {
		var entryType byte
		if err := binary.Read(reader, binary.LittleEndian, &entryType); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if entryType == 0x08 {
			// End of shortcuts
			break
		}

		if entryType != 0x00 {
			return nil, fmt.Errorf("expected dictionary entry, got type %x", entryType)
		}

		// Read entry index
		_, err := m.readNullTerminatedString(reader)
		if err != nil {
			return nil, err
		}

		// Parse shortcut
		shortcut, err := m.parseShortcut(reader)
		if err != nil {
			return nil, err
		}

		shortcuts = append(shortcuts, shortcut)
	}

	return shortcuts, nil
}

// parseShortcut parses a single shortcut from the binary data
func (m *Manager) parseShortcut(reader *bytes.Reader) (*SteamShortcut, error) {
	shortcut := &SteamShortcut{}

	for {
		var fieldType byte
		if err := binary.Read(reader, binary.LittleEndian, &fieldType); err != nil {
			return nil, err
		}

		if fieldType == 0x08 {
			// End of shortcut
			break
		}

		fieldName, err := m.readNullTerminatedString(reader)
		if err != nil {
			return nil, err
		}

		switch fieldType {
		case 0x00: // Dictionary (tags)
			if fieldName == "tags" {
				tags, err := m.parseTags(reader)
				if err != nil {
					return nil, err
				}
				shortcut.Tags = tags
			} else {
				// Skip unknown dictionary
				err := m.skipDictionary(reader)
				if err != nil {
					return nil, err
				}
			}
		case 0x01: // String
			value, err := m.readNullTerminatedString(reader)
			if err != nil {
				return nil, err
			}
			m.assignStringField(shortcut, fieldName, value)
		case 0x02: // Integer
			var value uint32
			if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
				return nil, err
			}
			m.assignIntField(shortcut, fieldName, value)
		default:
			return nil, fmt.Errorf("unknown field type %x", fieldType)
		}
	}

	return shortcut, nil
}

// parseTags parses the tags dictionary
func (m *Manager) parseTags(reader *bytes.Reader) ([]string, error) {
	var tags []string

	for {
		var fieldType byte
		if err := binary.Read(reader, binary.LittleEndian, &fieldType); err != nil {
			return nil, err
		}

		if fieldType == 0x08 {
			// End of tags
			break
		}

		// Read tag index (ignore)
		_, err := m.readNullTerminatedString(reader)
		if err != nil {
			return nil, err
		}

		if fieldType == 0x01 {
			// Read tag value
			tag, err := m.readNullTerminatedString(reader)
			if err != nil {
				return nil, err
			}
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

// skipDictionary skips over a dictionary in the binary data
func (m *Manager) skipDictionary(reader *bytes.Reader) error {
	for {
		var fieldType byte
		if err := binary.Read(reader, binary.LittleEndian, &fieldType); err != nil {
			return err
		}

		if fieldType == 0x08 {
			// End of dictionary
			break
		}

		// Read field name
		_, err := m.readNullTerminatedString(reader)
		if err != nil {
			return err
		}

		switch fieldType {
		case 0x00: // Nested dictionary
			err := m.skipDictionary(reader)
			if err != nil {
				return err
			}
		case 0x01: // String
			_, err := m.readNullTerminatedString(reader)
			if err != nil {
				return err
			}
		case 0x02: // Integer
			var value uint32
			if err := binary.Read(reader, binary.LittleEndian, &value); err != nil {
				return err
			}
		}
	}

	return nil
}

// readNullTerminatedString reads a null-terminated string from the reader
func (m *Manager) readNullTerminatedString(reader *bytes.Reader) (string, error) {
	var result []byte
	for {
		var b byte
		if err := binary.Read(reader, binary.LittleEndian, &b); err != nil {
			return "", err
		}
		if b == 0 {
			break
		}
		result = append(result, b)
	}
	return string(result), nil
}

// assignStringField assigns a string value to the appropriate shortcut field
func (m *Manager) assignStringField(shortcut *SteamShortcut, fieldName, value string) {
	switch fieldName {
	case "appname":
		shortcut.AppName = value
	case "exe":
		shortcut.Exe = value
	case "StartDir":
		shortcut.StartDir = value
	case "icon":
		shortcut.Icon = value
	case "ShortcutPath":
		shortcut.ShortcutPath = value
	case "LaunchOptions":
		shortcut.LaunchOptions = value
	case "DevkitGameID":
		shortcut.DevkitGameID = value
	case "FlatpakAppID":
		shortcut.FlatpakAppID = value
	}
}

// assignIntField assigns an integer value to the appropriate shortcut field
func (m *Manager) assignIntField(shortcut *SteamShortcut, fieldName string, value uint32) {
	switch fieldName {
	case "appid":
		shortcut.AppID = value
	case "IsHidden":
		shortcut.IsHidden = value != 0
	case "AllowDesktopConfig":
		shortcut.AllowDesktopConfig = value != 0
	case "AllowOverlay":
		shortcut.AllowOverlay = value != 0
	case "OpenVR":
		shortcut.OpenVR = value != 0
	case "Devkit":
		shortcut.Devkit = value != 0
	case "DevkitOverrideAppID":
		shortcut.DevkitOverrideAppID = value
	case "LastPlayTime":
		shortcut.LastPlayTime = value
	}
}

// buildShortcutsVDF builds the binary VDF format for shortcuts
func (m *Manager) buildShortcutsVDF(shortcuts []*SteamShortcut) []byte {
	var buffer bytes.Buffer

	// Write root dictionary header
	buffer.WriteByte(0x00) // Dictionary type
	buffer.WriteString("shortcuts")
	buffer.WriteByte(0x00) // Null terminator

	// Write shortcuts
	for i, shortcut := range shortcuts {
		// Write shortcut index
		buffer.WriteByte(0x00) // Dictionary type
		buffer.WriteString(fmt.Sprintf("%d", i))
		buffer.WriteByte(0x00) // Null terminator

		// Write shortcut data
		m.writeShortcutData(&buffer, shortcut)

		// End shortcut dictionary
		buffer.WriteByte(0x08)
	}

	// End shortcuts dictionary
	buffer.WriteByte(0x08)
	buffer.WriteByte(0x08) // End root dictionary

	return buffer.Bytes()
}

// writeShortcutData writes a single shortcut's data to the buffer
func (m *Manager) writeShortcutData(buffer *bytes.Buffer, shortcut *SteamShortcut) {
	// Write AppID (int32)
	buffer.WriteByte(0x02)
	buffer.WriteString("appid")
	buffer.WriteByte(0x00)
	binary.Write(buffer, binary.LittleEndian, shortcut.AppID)

	// Write string fields
	m.writeStringField(buffer, "appname", shortcut.AppName)
	m.writeStringField(buffer, "exe", shortcut.Exe)
	m.writeStringField(buffer, "StartDir", shortcut.StartDir)
	m.writeStringField(buffer, "icon", shortcut.Icon)
	m.writeStringField(buffer, "ShortcutPath", shortcut.ShortcutPath)
	m.writeStringField(buffer, "LaunchOptions", shortcut.LaunchOptions)

	// Write boolean fields (as int32)
	m.writeBoolField(buffer, "IsHidden", shortcut.IsHidden)
	m.writeBoolField(buffer, "AllowDesktopConfig", shortcut.AllowDesktopConfig)
	m.writeBoolField(buffer, "AllowOverlay", shortcut.AllowOverlay)
	m.writeBoolField(buffer, "OpenVR", shortcut.OpenVR)
	m.writeBoolField(buffer, "Devkit", shortcut.Devkit)

	// Write other fields
	m.writeStringField(buffer, "DevkitGameID", shortcut.DevkitGameID)
	m.writeIntField(buffer, "DevkitOverrideAppID", shortcut.DevkitOverrideAppID)
	m.writeIntField(buffer, "LastPlayTime", shortcut.LastPlayTime)
	m.writeStringField(buffer, "FlatpakAppID", shortcut.FlatpakAppID)

	// Write tags
	m.writeTags(buffer, shortcut.Tags)
}

// writeStringField writes a string field to the buffer
func (m *Manager) writeStringField(buffer *bytes.Buffer, name, value string) {
	buffer.WriteByte(0x01) // String type
	buffer.WriteString(name)
	buffer.WriteByte(0x00)
	buffer.WriteString(value)
	buffer.WriteByte(0x00)
}

// writeBoolField writes a boolean field as an int32 to the buffer
func (m *Manager) writeBoolField(buffer *bytes.Buffer, name string, value bool) {
	intValue := uint32(0)
	if value {
		intValue = 1
	}
	m.writeIntField(buffer, name, intValue)
}

// writeIntField writes an int32 field to the buffer
func (m *Manager) writeIntField(buffer *bytes.Buffer, name string, value uint32) {
	buffer.WriteByte(0x02) // Int32 type
	buffer.WriteString(name)
	buffer.WriteByte(0x00)
	binary.Write(buffer, binary.LittleEndian, value)
}

// writeTags writes the tags dictionary to the buffer
func (m *Manager) writeTags(buffer *bytes.Buffer, tags []string) {
	buffer.WriteByte(0x00) // Dictionary type
	buffer.WriteString("tags")
	buffer.WriteByte(0x00)

	for i, tag := range tags {
		buffer.WriteByte(0x01) // String type
		buffer.WriteString(fmt.Sprintf("%d", i))
		buffer.WriteByte(0x00)
		buffer.WriteString(tag)
		buffer.WriteByte(0x00)
	}

	buffer.WriteByte(0x08) // End tags dictionary
}

// GetShortcutURL returns the steam:// URL for launching the game
func (m *Manager) GetShortcutURL(appID uint32) string {
	// Steam URL format: steam://rungameid/<appid>
	return fmt.Sprintf("steam://rungameid/%d", appID)
}

// GetSteamAppID returns the Steam AppID that would be generated for a game
func (m *Manager) GetSteamAppID(game *models.Game) uint32 {
	return m.generateAppID(game.Name, game.Executable)
}

// CheckSteamRunning checks if Steam is currently running
func (m *Manager) CheckSteamRunning() (bool, error) {
	// This is a simple check - you might want to make it more robust
	switch runtime.GOOS {
	case "windows":
		// Check for steam.exe process
		// Implementation would depend on running tasklist command and checking output
		return false, nil
	default:
		// Check for steam process on Unix-like systems
		// Implementation would use ps or similar
		return false, nil
	}
} 