package main

import (
	"bufio"
	"fmt"
	"gamelauncher/game"
	"gamelauncher/models"
	"gamelauncher/monitor"
	"gamelauncher/storage"
	"os"
	"strconv"
	"strings"
)

// ConsoleApp represents the console-based game launcher
type ConsoleApp struct {
	gameManager *game.Manager
	storage     *storage.Manager
	monitor     *monitor.SourceMonitor
	games       []*models.Game
	settings    *models.Settings
}

// NewConsoleApp creates a new console application
func NewConsoleApp() *ConsoleApp {
	return &ConsoleApp{
		gameManager: game.NewManager(),
		storage:     storage.NewManager(),
		monitor:     monitor.NewSourceMonitor(),
	}
}

// Run starts the console application
func (app *ConsoleApp) Run() {
	app.loadData()

	for {
		app.showMenu()
		choice := app.getUserChoice()
		app.handleChoice(choice)
	}
}

// loadData loads games and settings from storage
func (app *ConsoleApp) loadData() {
	var err error

	app.games, err = app.storage.LoadGames()
	if err != nil {
		fmt.Printf("Error loading games: %v\n", err)
		app.games = []*models.Game{}
	}

	app.settings, err = app.storage.LoadSettings()
	if err != nil {
		fmt.Printf("Error loading settings: %v\n", err)
		app.settings = models.DefaultSettings()
	}
}

// showMenu displays the main menu
func (app *ConsoleApp) showMenu() {
	fmt.Println("\n=== Game Launcher Console ===")
	fmt.Println("1. List Games")
	fmt.Println("2. Add Game")
	fmt.Println("3. Import Games from Folder")
	fmt.Println("4. Launch Game")
	fmt.Println("5. Edit Game")
	fmt.Println("6. Delete Game")
	fmt.Println("7. Check for Updates")
	fmt.Println("8. Settings")
	fmt.Println("9. Exit")
	fmt.Print("Choose an option: ")
}

// getUserChoice gets user input for menu selection
func (app *ConsoleApp) getUserChoice() int {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice, err := strconv.Atoi(input)
	if err != nil {
		return 0
	}

	return choice
}

// handleChoice processes the user's menu choice
func (app *ConsoleApp) handleChoice(choice int) {
	switch choice {
	case 1:
		app.listGames()
	case 2:
		app.addGame()
	case 3:
		app.importGames()
	case 4:
		app.launchGame()
	case 5:
		app.editGame()
	case 6:
		app.deleteGame()
	case 7:
		app.checkUpdates()
	case 8:
		app.showSettings()
	case 9:
		fmt.Println("Goodbye!")
		os.Exit(0)
	default:
		fmt.Println("Invalid choice. Please try again.")
	}
}

// listGames displays all games
func (app *ConsoleApp) listGames() {
	fmt.Println("\n=== Games ===")
	if len(app.games) == 0 {
		fmt.Println("No games found.")
		return
	}

	for i, game := range app.games {
		fmt.Printf("%d. %s\n", i+1, game.Name)
		fmt.Printf("   Executable: %s\n", game.Executable)
		if game.CurrentVersion != "" {
			fmt.Printf("   Current Version: %s\n", game.CurrentVersion)
		}
		if game.SourceURL != "" {
			fmt.Printf("   Source: %s\n", game.SourceURL)
		}
		if game.VersionSelector != "" {
			fmt.Printf("   Version Selector: %s\n", game.VersionSelector)
		}
		fmt.Println()
	}
}

// addGame adds a new game manually
func (app *ConsoleApp) addGame() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter game name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Enter executable path: ")
	executable, _ := reader.ReadString('\n')
	executable = strings.TrimSpace(executable)

	// Clean the executable path
	executable = strings.Trim(executable, `"'`)

	fmt.Print("Enter source URL (optional): ")
	sourceURL, _ := reader.ReadString('\n')
	sourceURL = strings.TrimSpace(sourceURL)

	if name == "" || executable == "" {
		fmt.Println("Name and executable are required.")
		return
	}

	newGame := models.NewGame(name, executable, "")
	newGame.SourceURL = sourceURL

	app.games = append(app.games, newGame)
	app.saveGames()

	fmt.Printf("Game '%s' added successfully!\n", name)
}

// importGames imports games from a folder
func (app *ConsoleApp) importGames() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter folder path to scan: ")
	folderPath, _ := reader.ReadString('\n')
	folderPath = strings.TrimSpace(folderPath)

	// Clean the folder path
	folderPath = strings.Trim(folderPath, `"'`)

	games, err := app.gameManager.ScanFolder(folderPath)
	if err != nil {
		fmt.Printf("Error scanning folder: %v\n", err)
		return
	}

	if len(games) == 0 {
		fmt.Println("No executable games found in the folder.")
		return
	}

	fmt.Printf("Found %d games:\n", len(games))
	for i, game := range games {
		fmt.Printf("%d. %s (%s)\n", i+1, game.Name, game.Executable)
	}

	fmt.Print("Import all games? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "y" || response == "yes" {
		// Add new games to the list
		for _, newGame := range games {
			// Check if game already exists
			exists := false
			for _, existingGame := range app.games {
				if existingGame.Executable == newGame.Executable {
					exists = true
					break
				}
			}

			if !exists {
				app.games = append(app.games, newGame)
			}
		}

		app.saveGames()
		fmt.Printf("Imported %d new games.\n", len(games))
	}
}

// launchGame launches a selected game
func (app *ConsoleApp) launchGame() {
	if len(app.games) == 0 {
		fmt.Println("No games available.")
		return
	}

	app.listGames()
	fmt.Print("Enter game number to launch: ")
	choice := app.getUserChoice()

	if choice < 1 || choice > len(app.games) {
		fmt.Println("Invalid game number.")
		return
	}

	game := app.games[choice-1]
	fmt.Printf("Launching %s...\n", game.Name)

	err := app.gameManager.LaunchGame(game)
	if err != nil {
		fmt.Printf("Error launching game: %v\n", err)
	} else {
		fmt.Printf("Game '%s' launched successfully!\n", game.Name)
	}
}

// editGame edits a selected game
func (app *ConsoleApp) editGame() {
	if len(app.games) == 0 {
		fmt.Println("No games available.")
		return
	}

	app.listGames()
	fmt.Print("Enter game number to edit: ")
	choice := app.getUserChoice()

	if choice < 1 || choice > len(app.games) {
		fmt.Println("Invalid game number.")
		return
	}

	game := app.games[choice-1]
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Current name: %s\n", game.Name)
	fmt.Print("New name (press Enter to keep current): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name != "" {
		game.Name = name
	}

	fmt.Printf("Current executable: %s\n", game.Executable)
	fmt.Print("New executable (press Enter to keep current): ")
	executable, _ := reader.ReadString('\n')
	executable = strings.TrimSpace(executable)
	if executable != "" {
		game.Executable = executable
	}

	fmt.Printf("Current source URL: %s\n", game.SourceURL)
	fmt.Print("New source URL (press Enter to keep current): ")
	sourceURL, _ := reader.ReadString('\n')
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL != "" {
		game.SourceURL = sourceURL
	}

	fmt.Printf("Current version selector: %s\n", game.VersionSelector)
	fmt.Print("New version selector (CSS, e.g., .version, #version) (press Enter to keep current): ")
	versionSelector, _ := reader.ReadString('\n')
	versionSelector = strings.TrimSpace(versionSelector)
	if versionSelector != "" {
		game.VersionSelector = versionSelector
	}

	fmt.Printf("Current version pattern: %s\n", game.VersionPattern)
	fmt.Print("New version pattern (Regex, e.g., v(\\d+\\.\\d+\\.\\d+)) (press Enter to keep current): ")
	versionPattern, _ := reader.ReadString('\n')
	versionPattern = strings.TrimSpace(versionPattern)
	if versionPattern != "" {
		game.VersionPattern = versionPattern
	}

	fmt.Printf("Current version: %s\n", game.CurrentVersion)
	fmt.Print("New current version (press Enter to keep current): ")
	currentVersion, _ := reader.ReadString('\n')
	currentVersion = strings.TrimSpace(currentVersion)
	if currentVersion != "" {
		game.CurrentVersion = currentVersion
	}

	app.saveGames()
	fmt.Printf("Game '%s' updated successfully!\n", game.Name)
}

// deleteGame deletes a game
func (app *ConsoleApp) deleteGame() {
	if len(app.games) == 0 {
		fmt.Println("No games to delete.")
		return
	}

	app.listGames()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the number of the game to delete: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(app.games) {
		fmt.Println("Invalid choice.")
		return
	}

	// Convert to 0-based index
	index := choice - 1
	game := app.games[index]

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete '%s'? (y/N): ", game.Name)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "y" && confirm != "yes" {
		fmt.Println("Deletion cancelled.")
		return
	}

	// Remove game from slice
	app.games = append(app.games[:index], app.games[index+1:]...)

	app.saveGames()
	fmt.Printf("Game '%s' deleted successfully!\n", game.Name)
}

// checkUpdates checks for updates on all games
func (app *ConsoleApp) checkUpdates() {
	fmt.Println("Checking for updates...")

	updatedCount := 0
	for _, game := range app.games {
		if game.SourceURL != "" {
			fmt.Printf("Checking %s...\n", game.Name)

			updateInfo, err := app.monitor.CheckForUpdates(game)
			if err == nil && updateInfo.HasUpdate {
				game.UpdateInfo(updateInfo.Version)
				game.MarkChecked()
				fmt.Printf("Update available for %s: %s\n", game.Name, updateInfo.Version)
				updatedCount++
			} else if err != nil {
				fmt.Printf("Error checking %s: %v\n", game.Name, err)
			}
		}
	}

	if updatedCount > 0 {
		app.saveGames()
		fmt.Printf("Found %d updates.\n", updatedCount)
	} else {
		fmt.Println("No updates found.")
	}
}

// showSettings displays and allows editing of settings
func (app *ConsoleApp) showSettings() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n=== Settings ===")
	fmt.Printf("Check interval: %d seconds\n", app.settings.CheckInterval)
	fmt.Printf("Notifications: %t\n", app.settings.Notifications)

	fmt.Print("New check interval (seconds, press Enter to keep current): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		if interval, err := strconv.Atoi(input); err == nil && interval > 0 {
			app.settings.CheckInterval = interval
		}
	}

	fmt.Print("Enable notifications? (y/n, press Enter to keep current): ")
	input, _ = reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "y" || input == "yes" {
		app.settings.Notifications = true
	} else if input == "n" || input == "no" {
		app.settings.Notifications = false
	}

	app.saveSettings()
	fmt.Println("Settings saved.")
}

// saveGames saves the games list to storage
func (app *ConsoleApp) saveGames() {
	err := app.storage.SaveGames(app.games)
	if err != nil {
		fmt.Printf("Error saving games: %v\n", err)
	}
}

// saveSettings saves the settings to storage
func (app *ConsoleApp) saveSettings() {
	err := app.storage.SaveSettings(app.settings)
	if err != nil {
		fmt.Printf("Error saving settings: %v\n", err)
	}
}

func main() {
	fmt.Println("Game Launcher Console Version")
	fmt.Println("=============================")

	app := NewConsoleApp()
	app.Run()
}
