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
	AppID               uint32
	AppName             string
	Exe                 string
	StartDir            string
	Icon                string
	ShortcutPath        string
	LaunchOptions       string
	IsHidden            bool
	AllowDesktopConfig  bool
	AllowOverlay        bool
	OpenVR              bool
	Devkit              bool
	DevkitGameID        string
	DevkitOverrideAppID uint32
	LastPlayTime        uint32
	FlatpakAppID        string
	Tags                []string

	// Preserve unknown fields to prevent corruption
	UnknownStrings map[string]string `json:"unknown_strings,omitempty"`
	UnknownInts    map[string]uint32 `json:"unknown_ints,omitempty"`
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

	// Check if shortcut already exists
	for _, existing := range shortcuts {
		// Primary check: same AppID (now based only on game name)
		if existing.AppID == appID {
			return true, nil
		}

		// Secondary check: same normalized name
		// This provides additional safety for games that might have been added
		// before the AppID generation change
		existingNormalizedName := m.normalizeName(existing.AppName)
		if existingNormalizedName == normalizedName {
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

// formatPathsForPlatform formats executable and start directory paths according to platform requirements
func (m *Manager) formatPathsForPlatform(exe, startDir string) (string, string) {
	// On Linux/Unix systems, Steam requires specific formatting
	if runtime.GOOS != "windows" {
		// Quote executable path if it contains spaces or special characters
		if strings.Contains(exe, " ") || m.needsQuoting(exe) {
			exe = fmt.Sprintf(`"%s"`, exe)
		}

		// Ensure start directory has trailing slash on Linux
		if !strings.HasSuffix(startDir, "/") {
			startDir = startDir + "/"
		}
	}

	return exe, startDir
}

// needsQuoting checks if a path needs to be quoted for Steam on Linux
func (m *Manager) needsQuoting(path string) bool {
	// Check for characters that require quoting in Steam shortcuts on Linux
	specialChars := []string{" ", "(", ")", "[", "]", "{", "}", "&", "|", ";", "<", ">", "?", "*", "'", "`", "\"", "\\"}

	for _, char := range specialChars {
		if strings.Contains(path, char) {
			return true
		}
	}

	return false
}

// CreateShortcutFromGame creates a Steam shortcut from a game model (public for testing)
func (m *Manager) CreateShortcutFromGame(game *models.Game) *SteamShortcut {
	return m.createShortcutFromGame(game)
}

// createShortcutFromGame creates a Steam shortcut from a game model (internal)
func (m *Manager) createShortcutFromGame(game *models.Game) *SteamShortcut {
	// Generate AppID
	appID := m.generateAppID(game.Name, game.Executable)

	// Always use the executable's parent directory as StartDir
	// This ensures Steam starts the game from the correct directory
	startDir := filepath.Dir(game.Executable)

	// Format executable path and start directory according to platform requirements
	exe, startDir := m.formatPathsForPlatform(game.Executable, startDir)

	// Use game image as icon if available
	icon := game.IconPath
	if icon == "" && game.ImagePath != "" {
		icon = game.ImagePath
	}

	return &SteamShortcut{
		AppID:               appID,
		AppName:             game.Name,
		Exe:                 exe,      // Use formatted executable path
		StartDir:            startDir, // Use executable's parent directory
		Icon:                icon,
		ShortcutPath:        "",
		LaunchOptions:       "",
		IsHidden:            false,
		AllowDesktopConfig:  true,
		AllowOverlay:        true,
		OpenVR:              false,
		Devkit:              false,
		DevkitGameID:        "",
		DevkitOverrideAppID: 0,
		LastPlayTime:        0,
		FlatpakAppID:        "",
		Tags:                []string{},
		UnknownStrings:      make(map[string]string),
		UnknownInts:         make(map[string]uint32),
	}
}

// generateAppID generates a unique AppID for the shortcut based on name only
func (m *Manager) generateAppID(appName, exe string) uint32 {
	// Normalize name to ensure consistency across different executable paths
	normalizedName := m.normalizeName(appName)

	// Steam uses CRC32 of name + null terminator, with high bit set
	// Note: We only use the name now, not the executable path
	// This ensures that games with the same name but different executable paths
	// (e.g., when updating to newer versions) get the same AppID
	input := normalizedName + "\x00"
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

	// Check if shortcut already exists (by AppID or by normalized name)
	existingIndex := -1
	normalizedName := m.normalizeName(shortcut.AppName)

	for i, existing := range shortcuts {
		// Primary check: same AppID (now based only on game name)
		if existing.AppID == shortcut.AppID {
			existingIndex = i
			break
		}

		// Secondary check: same normalized name
		// This catches shortcuts created with the old AppID system
		existingNormalizedName := m.normalizeName(existing.AppName)
		if existingNormalizedName == normalizedName {
			existingIndex = i
			break
		}
	}

	if existingIndex >= 0 {
		// Update existing shortcut by preserving important existing data
		// and only updating the fields that should change
		existingShortcut := shortcuts[existingIndex]

		// Create a new shortcut with updated data but preserve existing values
		updatedShortcut := &SteamShortcut{
			AppID:               shortcut.AppID,                       // Use new AppID format
			AppName:             shortcut.AppName,                     // Update name (should be same anyway)
			Exe:                 shortcut.Exe,                         // Update executable path
			StartDir:            shortcut.StartDir,                    // Update start directory
			Icon:                shortcut.Icon,                        // Update icon
			ShortcutPath:        existingShortcut.ShortcutPath,        // Preserve existing
			LaunchOptions:       existingShortcut.LaunchOptions,       // Preserve existing
			IsHidden:            existingShortcut.IsHidden,            // Preserve existing
			AllowDesktopConfig:  existingShortcut.AllowDesktopConfig,  // Preserve existing
			AllowOverlay:        existingShortcut.AllowOverlay,        // Preserve existing
			OpenVR:              existingShortcut.OpenVR,              // Preserve existing
			Devkit:              existingShortcut.Devkit,              // Preserve existing
			DevkitGameID:        existingShortcut.DevkitGameID,        // Preserve existing
			DevkitOverrideAppID: existingShortcut.DevkitOverrideAppID, // Preserve existing
			LastPlayTime:        existingShortcut.LastPlayTime,        // Preserve play time
			FlatpakAppID:        existingShortcut.FlatpakAppID,        // Preserve existing
			Tags:                existingShortcut.Tags,                // Preserve tags
			UnknownStrings:      existingShortcut.UnknownStrings,      // Preserve unknown string fields
			UnknownInts:         existingShortcut.UnknownInts,         // Preserve unknown int fields
		}

		shortcuts[existingIndex] = updatedShortcut
		log.Printf("Updated existing Steam shortcut: %s (AppID: %d)", updatedShortcut.AppName, updatedShortcut.AppID)
	} else {
		// Add new shortcut
		shortcuts = append(shortcuts, shortcut)
		log.Printf("Added new Steam shortcut: %s (AppID: %d)", shortcut.AppName, shortcut.AppID)
	}

	// Write shortcuts back to file
	return m.writeShortcutsFile(shortcutsPath, shortcuts)
}

// ReadShortcutsFile reads shortcuts from the shortcuts.vdf file (public for testing)
func (m *Manager) ReadShortcutsFile(filePath string) ([]*SteamShortcut, error) {
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

// WriteShortcutsFile writes shortcuts to the shortcuts.vdf file (public for testing)
func (m *Manager) WriteShortcutsFile(filePath string, shortcuts []*SteamShortcut) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data := m.buildShortcutsVDF(shortcuts)

	return os.WriteFile(filePath, data, 0644)
}

// readShortcutsFile reads shortcuts from the shortcuts.vdf file (internal)
func (m *Manager) readShortcutsFile(filePath string) ([]*SteamShortcut, error) {
	return m.ReadShortcutsFile(filePath)
}

// writeShortcutsFile writes shortcuts to the shortcuts.vdf file (internal)
func (m *Manager) writeShortcutsFile(filePath string, shortcuts []*SteamShortcut) error {
	return m.WriteShortcutsFile(filePath, shortcuts)
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
	case "appname", "AppName": // Handle both cases Steam might use
		shortcut.AppName = value
	case "exe", "Exe": // Handle both cases Steam might use
		shortcut.Exe = value
	case "StartDir": // Steam uses this exact casing
		shortcut.StartDir = value
	case "icon", "Icon": // Handle both cases
		shortcut.Icon = value
	case "ShortcutPath": // Steam uses this exact casing
		shortcut.ShortcutPath = value
	case "LaunchOptions": // Steam uses this exact casing
		shortcut.LaunchOptions = value
	case "DevkitGameID": // Steam uses this exact casing
		shortcut.DevkitGameID = value
	case "FlatpakAppID": // Steam uses this exact casing
		shortcut.FlatpakAppID = value
	default:
		// Preserve unknown string fields to prevent corruption
		if shortcut.UnknownStrings == nil {
			shortcut.UnknownStrings = make(map[string]string)
		}
		shortcut.UnknownStrings[fieldName] = value
	}
}

// assignIntField assigns an integer value to the appropriate shortcut field
func (m *Manager) assignIntField(shortcut *SteamShortcut, fieldName string, value uint32) {
	switch fieldName {
	case "appid", "AppID": // Handle both cases Steam might use
		shortcut.AppID = value
	case "IsHidden": // Steam uses this exact casing
		shortcut.IsHidden = value != 0
	case "AllowDesktopConfig": // Steam uses this exact casing
		shortcut.AllowDesktopConfig = value != 0
	case "AllowOverlay": // Steam uses this exact casing
		shortcut.AllowOverlay = value != 0
	case "OpenVR": // Steam uses this exact casing
		shortcut.OpenVR = value != 0
	case "Devkit": // Steam uses this exact casing
		shortcut.Devkit = value != 0
	case "DevkitOverrideAppID": // Steam uses this exact casing
		shortcut.DevkitOverrideAppID = value
	case "LastPlayTime": // Steam uses this exact casing
		shortcut.LastPlayTime = value
	default:
		// Preserve unknown integer fields to prevent corruption
		if shortcut.UnknownInts == nil {
			shortcut.UnknownInts = make(map[string]uint32)
		}
		shortcut.UnknownInts[fieldName] = value
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

	// Write string fields using Steam's expected casing
	m.writeStringField(buffer, "AppName", shortcut.AppName) // Use Steam's casing
	m.writeStringField(buffer, "Exe", shortcut.Exe)         // Use Steam's casing
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

	// Write unknown string fields to preserve all data
	for fieldName, value := range shortcut.UnknownStrings {
		m.writeStringField(buffer, fieldName, value)
	}

	// Write unknown integer fields to preserve all data
	for fieldName, value := range shortcut.UnknownInts {
		m.writeIntField(buffer, fieldName, value)
	}

	// Write tags
	m.writeTags(buffer, shortcut.Tags)
}

// writeStringField writes a string field to the buffer
func (m *Manager) writeStringField(buffer *bytes.Buffer, name, value string) {
	// ALWAYS write all fields to preserve existing shortcut data
	// Skipping fields based on content was causing corruption of existing shortcuts
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

// AddAllGamesToSteam adds all games in the list to Steam as non-Steam shortcuts
func (m *Manager) AddAllGamesToSteam(games []*models.Game) error {
	if len(games) == 0 {
		return fmt.Errorf("no games to add")
	}

	// Find Steam installation and user data path once
	steamPath, err := m.findSteamPath()
	if err != nil {
		return fmt.Errorf("failed to find Steam installation: %w", err)
	}

	userDataPath, err := m.findUserDataPath(steamPath)
	if err != nil {
		return fmt.Errorf("failed to find Steam user data: %w", err)
	}

	shortcutsPath := filepath.Join(userDataPath, "config", "shortcuts.vdf")

	// Read existing shortcuts once
	existingShortcuts, err := m.readShortcutsFile(shortcutsPath)
	if err != nil {
		// If file doesn't exist, create empty list
		existingShortcuts = []*SteamShortcut{}
	}

	addedCount := 0
	updatedCount := 0
	errors := []string{}

	// Process each game
	for _, game := range games {
		// Create shortcut from game
		shortcut := m.createShortcutFromGame(game)

		// Check if shortcut already exists
		existingIndex := -1
		normalizedName := m.normalizeName(shortcut.AppName)

		for i, existing := range existingShortcuts {
			// Primary check: same AppID
			if existing.AppID == shortcut.AppID {
				existingIndex = i
				break
			}

			// Secondary check: same normalized name
			existingNormalizedName := m.normalizeName(existing.AppName)
			if existingNormalizedName == normalizedName {
				existingIndex = i
				break
			}
		}

		if existingIndex >= 0 {
			// Update existing shortcut
			existingShortcut := existingShortcuts[existingIndex]
			updatedShortcut := &SteamShortcut{
				AppID:               shortcut.AppID,
				AppName:             shortcut.AppName,
				Exe:                 shortcut.Exe,
				StartDir:            shortcut.StartDir,
				Icon:                shortcut.Icon,
				ShortcutPath:        existingShortcut.ShortcutPath,
				LaunchOptions:       existingShortcut.LaunchOptions,
				IsHidden:            existingShortcut.IsHidden,
				AllowDesktopConfig:  existingShortcut.AllowDesktopConfig,
				AllowOverlay:        existingShortcut.AllowOverlay,
				OpenVR:              existingShortcut.OpenVR,
				Devkit:              existingShortcut.Devkit,
				DevkitGameID:        existingShortcut.DevkitGameID,
				DevkitOverrideAppID: existingShortcut.DevkitOverrideAppID,
				LastPlayTime:        existingShortcut.LastPlayTime,
				FlatpakAppID:        existingShortcut.FlatpakAppID,
				Tags:                existingShortcut.Tags,
				UnknownStrings:      existingShortcut.UnknownStrings,
				UnknownInts:         existingShortcut.UnknownInts,
			}
			existingShortcuts[existingIndex] = updatedShortcut
			updatedCount++
			log.Printf("Updated existing Steam shortcut: %s (AppID: %d)", updatedShortcut.AppName, updatedShortcut.AppID)
		} else {
			// Add new shortcut
			existingShortcuts = append(existingShortcuts, shortcut)
			addedCount++
			log.Printf("Added new Steam shortcut: %s (AppID: %d)", shortcut.AppName, shortcut.AppID)
		}
	}

	// Write all shortcuts back to file
	err = m.writeShortcutsFile(shortcutsPath, existingShortcuts)
	if err != nil {
		return fmt.Errorf("failed to write shortcuts file: %w", err)
	}

	log.Printf("Bulk Steam operation completed: %d added, %d updated", addedCount, updatedCount)

	if len(errors) > 0 {
		return fmt.Errorf("completed with %d errors: %v", len(errors), errors)
	}

	return nil
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
