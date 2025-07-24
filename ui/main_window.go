package ui

import (
	"fmt"
	"gamelauncher/game"
	"gamelauncher/models"
	"gamelauncher/monitor"
	"gamelauncher/storage"
	"image/color"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MainWindow represents the main application window
type MainWindow struct {
	app          fyne.App
	window       fyne.Window
	gameManager  *game.Manager
	storage      *storage.Manager
	monitor      *monitor.SourceMonitor
	games        []*models.Game
	settings     *models.Settings
	gameList     *widget.List
	refreshTimer *time.Timer
	selectedGame int // Track selected game index
}

// NewMainWindow creates a new main window
func NewMainWindow() *MainWindow {
	myApp := app.New()
	myApp.SetIcon(theme.ComputerIcon())

	window := myApp.NewWindow("Game Launcher")
	window.Resize(fyne.NewSize(800, 600))

	mw := &MainWindow{
		app:          myApp,
		window:       window,
		gameManager:  game.NewManager(),
		storage:      storage.NewManager(),
		monitor:      monitor.NewSourceMonitor(),
		selectedGame: -1, // Initialize to no selection
	}

	mw.loadData()
	mw.setupUI()
	mw.startUpdateTimer()

	return mw
}

// ShowAndRun shows the window and runs the application
func (mw *MainWindow) ShowAndRun() {
	mw.window.ShowAndRun()
}

// loadData loads games and settings from storage
func (mw *MainWindow) loadData() {
	var err error

	mw.games, err = mw.storage.LoadGames()
	if err != nil {
		dialog.ShowError(err, mw.window)
		mw.games = []*models.Game{}
	}

	mw.settings, err = mw.storage.LoadSettings()
	if err != nil {
		dialog.ShowError(err, mw.window)
		mw.settings = models.DefaultSettings()
	}
}

// setupUI sets up the user interface
func (mw *MainWindow) setupUI() {
	// Create toolbar
	toolbar := mw.createToolbar()

	// Create game list with fixed-width columns using list widget
	mw.gameList = widget.NewList(
		func() int { return len(mw.games) },
		func() fyne.CanvasObject {
			// Use Border container to put name on left, everything else on right
			nameLabel := widget.NewLabel("Game Name")

			// Create right-side container with all other elements
			rightContainer := container.NewHBox()

			// Current Version column - compact
			currentVersionLabel := widget.NewLabel("Current Version")
			currentVersionContainer := container.NewHBox(currentVersionLabel)

			// Fetched Version column - compact
			fetchedVersionLabel := NewColoredLabel("Fetched Version", color.White, color.Black)
			fetchedVersionContainer := container.NewHBox(fetchedVersionLabel)

			// Launch button column - compact
			launchBtn := widget.NewButton("Launch", nil)
			launchContainer := container.NewHBox(launchBtn)

			// Edit button column - compact
			editBtn := widget.NewButton("Edit", nil)
			editContainer := container.NewHBox(editBtn)

			// Add all right-side elements
			rightContainer.Add(currentVersionContainer)
			rightContainer.Add(fetchedVersionContainer)
			rightContainer.Add(launchContainer)
			rightContainer.Add(editContainer)

			// Use Border to put name on left, everything else on right side
			return container.NewBorder(nil, nil, nil, rightContainer, nameLabel)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			game := mw.games[id]
			borderContainer := obj.(*fyne.Container)

			// Border structure: [center, right] - only 2 objects
			if len(borderContainer.Objects) >= 2 {
				// Update name label (center - index 0)
				if nameLabel, ok := borderContainer.Objects[0].(*widget.Label); ok {
					nameLabel.SetText(game.Name)
				}

				// Update right-side container elements (right - index 1)
				if rightContainer, ok := borderContainer.Objects[1].(*fyne.Container); ok {
					if len(rightContainer.Objects) >= 4 {
						// Update current version (first element)
						if currentVersionContainer, ok := rightContainer.Objects[0].(*fyne.Container); ok {
							if len(currentVersionContainer.Objects) > 0 {
								if currentVersionLabel, ok := currentVersionContainer.Objects[0].(*widget.Label); ok {
									if game.CurrentVersion != "" {
										currentVersionLabel.SetText(mw.truncateText(game.CurrentVersion, 15))
									} else {
										currentVersionLabel.SetText("Unknown")
									}
								}
							}
						}

						// Update fetched version (second element)
						if fetchedVersionContainer, ok := rightContainer.Objects[1].(*fyne.Container); ok {
							if len(fetchedVersionContainer.Objects) > 0 {
								if fetchedVersionLabel, ok := fetchedVersionContainer.Objects[0].(*ColoredLabel); ok {
									mw.updateFetchedVersionLabel(game, fetchedVersionLabel)
								}
							}
						}

						// Update launch button (third element)
						if launchContainer, ok := rightContainer.Objects[2].(*fyne.Container); ok {
							if len(launchContainer.Objects) > 0 {
								if launchBtn, ok := launchContainer.Objects[0].(*widget.Button); ok {
									launchBtn.OnTapped = func() {
										mw.launchGame(game)
									}
								}
							}
						}

						// Update edit button (fourth element)
						if editContainer, ok := rightContainer.Objects[3].(*fyne.Container); ok {
							if len(editContainer.Objects) > 0 {
								if editBtn, ok := editContainer.Objects[0].(*widget.Button); ok {
									editBtn.OnTapped = func() {
										mw.editGame(game)
									}
								}
							}
						}
					}
				}
			}
		},
	)

	// Add selection tracking
	mw.gameList.OnSelected = func(id widget.ListItemID) {
		mw.selectedGame = int(id)
	}

	// Create main container
	content := container.NewBorder(toolbar, nil, nil, nil, mw.gameList)
	mw.window.SetContent(content)

	// Start version checking for all games
	mw.refreshAllVersionChecks()
}

// createToolbar creates the main toolbar
func (mw *MainWindow) createToolbar() *widget.Toolbar {
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
			mw.importGames()
		}),
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			mw.addGame()
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			mw.deleteSelectedGame()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			mw.checkAllUpdates()
		}),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			mw.showSettings()
		}),
	)
}

// importGames shows a dialog to import games from a folder
func (mw *MainWindow) importGames() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			if err != nil {
				dialog.ShowError(err, mw.window)
			}
			return
		}

		games, err := mw.gameManager.ScanFolder(uri.Path())
		if err != nil {
			dialog.ShowError(err, mw.window)
			return
		}

		if len(games) == 0 {
			dialog.ShowInformation("No Games Found",
				"No executable games were found in the selected folder.", mw.window)
			return
		}

		// Add new games to the list
		for _, newGame := range games {
			// Check if game already exists
			exists := false
			for _, existingGame := range mw.games {
				if existingGame.Executable == newGame.Executable {
					exists = true
					break
				}
			}

			if !exists {
				mw.games = append(mw.games, newGame)
			}
		}

		mw.saveGames()
		mw.gameList.Refresh()

		dialog.ShowInformation("Import Complete",
			fmt.Sprintf("Imported %d new games.", len(games)), mw.window)
	}, mw.window)
}

// addGame shows a dialog to manually add a game
func (mw *MainWindow) addGame() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Game Name")

	execEntry := widget.NewEntry()
	execEntry.SetPlaceHolder("Click 'Browse' to select executable")
	execEntry.Disable() // Make it read-only

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Source URL (optional)")

	// Create browse button
	browseBtn := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				if err != nil {
					dialog.ShowError(err, mw.window)
				}
				return
			}
			defer reader.Close()
			execEntry.SetText(reader.URI().Path())
		}, mw.window)
	})

	// Create executable selection container
	execContainer := container.NewBorder(nil, nil, nil, browseBtn, execEntry)

	form := dialog.NewForm("Add Game", "Add", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Executable", execContainer),
			widget.NewFormItem("Source URL", urlEntry),
		},
		func(confirm bool) {
			if !confirm {
				return
			}

			if nameEntry.Text == "" || execEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("name and executable are required"), mw.window)
				return
			}

			newGame := models.NewGame(nameEntry.Text, execEntry.Text, "")
			newGame.SourceURL = urlEntry.Text

			mw.games = append(mw.games, newGame)
			mw.saveGames()
			mw.gameList.Refresh()
		},
		mw.window)

	form.Resize(fyne.NewSize(500, 250))
	form.Show()
}

// launchGame launches a game
func (mw *MainWindow) launchGame(game *models.Game) {
	err := mw.gameManager.LaunchGame(game)
	if err != nil {
		dialog.ShowError(err, mw.window)
	} else {
		dialog.ShowInformation("Game Launched",
			fmt.Sprintf("Launched %s successfully!", game.Name), mw.window)
	}
}

// editGame shows a dialog to edit game properties
func (mw *MainWindow) editGame(game *models.Game) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(game.Name)

	execEntry := widget.NewEntry()
	execEntry.SetText(game.Executable)
	execEntry.Disable() // Make it read-only

	// Create browse button for edit dialog
	browseBtn := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				if err != nil {
					dialog.ShowError(err, mw.window)
				}
				return
			}
			defer reader.Close()
			execEntry.SetText(reader.URI().Path())
		}, mw.window)
	})

	// Create executable selection container
	execContainer := container.NewBorder(nil, nil, nil, browseBtn, execEntry)

	urlEntry := widget.NewEntry()
	urlEntry.SetText(game.SourceURL)

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetText(game.Description)

	// Version checking configuration
	versionSelectorEntry := widget.NewEntry()
	versionSelectorEntry.SetText(game.VersionSelector)
	versionSelectorEntry.SetPlaceHolder("e.g., .version, #version, h1")

	versionPatternEntry := widget.NewEntry()
	versionPatternEntry.SetText(game.VersionPattern)
	versionPatternEntry.SetPlaceHolder("e.g., v(\\d+\\.\\d+\\.\\d+), (\\d+\\.\\d+\\.\\d+)")

	currentVersionEntry := widget.NewEntry()
	currentVersionEntry.SetText(game.CurrentVersion)
	currentVersionEntry.SetPlaceHolder("Current version for comparison")

	form := dialog.NewForm("Edit Game", "Save", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Executable", execContainer),
			widget.NewFormItem("Source URL", urlEntry),
			widget.NewFormItem("Description", descEntry),
			widget.NewFormItem("Version Selector (CSS)", versionSelectorEntry),
			widget.NewFormItem("Version Pattern (Regex)", versionPatternEntry),
			widget.NewFormItem("Current Version", currentVersionEntry),
		},
		func(confirm bool) {
			if !confirm {
				return
			}

			game.Name = nameEntry.Text
			game.Executable = execEntry.Text
			game.SourceURL = urlEntry.Text
			game.Description = descEntry.Text
			game.VersionSelector = versionSelectorEntry.Text
			game.VersionPattern = versionPatternEntry.Text
			game.CurrentVersion = currentVersionEntry.Text

			mw.saveGames()
			mw.gameList.Refresh()
		},
		mw.window)

	form.Resize(fyne.NewSize(500, 400))
	form.Show()
}

// deleteSelectedGame deletes the currently selected game
func (mw *MainWindow) deleteSelectedGame() {
	if mw.selectedGame < 0 || mw.selectedGame >= len(mw.games) {
		dialog.ShowInformation("No Game Selected",
			"Please select a game to delete.", mw.window)
		return
	}

	game := mw.games[mw.selectedGame]

	// Show confirmation dialog
	dialog.ShowConfirm("Delete Game",
		fmt.Sprintf("Are you sure you want to delete '%s'?\n\nThis action cannot be undone.", game.Name),
		func(confirm bool) {
			if !confirm {
				return
			}

			// Remove game from slice
			mw.games = append(mw.games[:mw.selectedGame], mw.games[mw.selectedGame+1:]...)

			// Reset selection
			mw.selectedGame = -1

			// Save changes
			mw.saveGames()

			// Refresh the list
			mw.gameList.Refresh()

			dialog.ShowInformation("Game Deleted",
				fmt.Sprintf("'%s' has been deleted successfully.", game.Name), mw.window)
		}, mw.window)
}

// updateFetchedVersionLabel updates the fetched version label with color coding
func (mw *MainWindow) updateFetchedVersionLabel(game *models.Game, label *ColoredLabel) {
	// Set default text and color
	label.SetText("Checking...")
	label.Refresh()

	// If no source URL, show as unavailable
	if game.SourceURL == "" {
		label.SetText("No source")
		label.Refresh()
		return
	}

	// Check for updates in a goroutine to avoid blocking the UI
	go func() {
		updateInfo, err := mw.monitor.CheckForUpdates(game)

		// Update UI on the main thread using the app's main thread
		mw.app.SendNotification(&fyne.Notification{
			Title: fmt.Sprintf("Version Check - %s", game.Name),
		})

		// Use the main thread to update the label
		mw.window.Canvas().Refresh(label)

		if err != nil {
			label.SetText("Error")
			label.Refresh()
			return
		}

		if updateInfo.Version == "" {
			label.SetText("Not found")
			label.Refresh()
			return
		}

		// Debug: Print the exact values being compared
		fmt.Printf("DEBUG: Game='%s', CurrentVersion='%s' (len=%d), FoundVersion='%s' (len=%d), Equal=%t\n",
			game.Name, game.CurrentVersion, len(game.CurrentVersion), updateInfo.Version, len(updateInfo.Version),
			updateInfo.Version == game.CurrentVersion)

		// Compare versions and set text with color indicators
		var displayText string
		var bgColor, textColor color.Color

		if updateInfo.Version == game.CurrentVersion {
			// Same version - show with green background
			displayText = updateInfo.Version
			bgColor = color.NRGBA{R: 0, G: 255, B: 0, A: 100} // Light green
			textColor = color.Black
			fmt.Printf("DEBUG: Setting label to: '%s' (GREEN)\n", displayText)
		} else if updateInfo.Version != "" && updateInfo.Version != game.CurrentVersion {
			// Different version - determine if it's newer
			if mw.isVersionNewer(updateInfo.Version, game.CurrentVersion) {
				// Newer version available - show with red background
				displayText = updateInfo.Version + " [NEW]"
				bgColor = color.NRGBA{R: 255, G: 0, B: 0, A: 100} // Light red
				textColor = color.White
			} else {
				// Different version but not newer - show with orange background
				displayText = updateInfo.Version + " [DIFF]"
				bgColor = color.NRGBA{R: 255, G: 165, B: 0, A: 100} // Light orange
				textColor = color.Black
			}
		} else {
			// No version found or error
			displayText = "Not found"
			bgColor = color.NRGBA{R: 128, G: 128, B: 128, A: 100} // Light gray
			textColor = color.Black
		}

		// Update the label on the main thread
		label.SetText(mw.truncateText(displayText, 20)) // Truncate fetched version text
		label.SetColors(bgColor, textColor)
		label.Refresh()

		// Force a canvas refresh
		mw.window.Canvas().Refresh(label)
	}()
}

// truncateText truncates text to the specified length and adds ellipsis if needed
func (mw *MainWindow) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
}

// isVersionNewer compares two version strings and returns true if version1 is newer than version2
func (mw *MainWindow) isVersionNewer(version1, version2 string) bool {
	// Clean up version strings
	v1 := strings.TrimSpace(version1)
	v2 := strings.TrimSpace(version2)

	// If either version is empty, can't compare
	if v1 == "" || v2 == "" {
		return false
	}

	// If versions are identical, neither is newer
	if v1 == v2 {
		return false
	}

	// Try to parse as semantic versions first
	if mw.compareSemanticVersions(v1, v2) {
		return true
	}

	// Fallback to string comparison for non-semantic versions
	return v1 > v2
}

// compareSemanticVersions compares semantic version strings (e.g., "1.2.3")
func (mw *MainWindow) compareSemanticVersions(v1, v2 string) bool {
	// Split versions into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Find the maximum length
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	// Compare each part
	for i := 0; i < maxLen; i++ {
		var part1, part2 string
		if i < len(parts1) {
			part1 = parts1[i]
		}
		if i < len(parts2) {
			part2 = parts2[i]
		}

		// Convert to integers for comparison
		num1 := mw.parseVersionPart(part1)
		num2 := mw.parseVersionPart(part2)

		if num1 > num2 {
			return true
		} else if num1 < num2 {
			return false
		}
	}

	return false // Versions are equal
}

// parseVersionPart converts a version part string to an integer
func (mw *MainWindow) parseVersionPart(part string) int {
	// Remove any non-numeric characters
	clean := ""
	for _, char := range part {
		if char >= '0' && char <= '9' {
			clean += string(char)
		}
	}

	if clean == "" {
		return 0
	}

	// Convert to integer
	if num, err := strconv.Atoi(clean); err == nil {
		return num
	}

	return 0
}

// refreshAllVersionChecks refreshes version checks for all games
func (mw *MainWindow) refreshAllVersionChecks() {
	// Refresh the list to trigger version checks for all games
	mw.gameList.Refresh()
}

// checkAllUpdates checks for updates on all games
func (mw *MainWindow) checkAllUpdates() {
	progress := dialog.NewProgress("Checking Updates", "Checking for game updates...", mw.window)
	progress.Show()

	go func() {
		defer progress.Hide()

		for i, game := range mw.games {
			progress.SetValue(float64(i) / float64(len(mw.games)))

			if game.SourceURL != "" {
				updateInfo, err := mw.monitor.CheckForUpdates(game)
				if err == nil && updateInfo.HasUpdate {
					game.UpdateInfo(updateInfo.Version)
					game.MarkChecked()

					// Show notification
					if mw.settings.Notifications {
						dialog.ShowInformation("Update Available",
							fmt.Sprintf("%s has an update available: %s", game.Name, updateInfo.Version), mw.window)
					}
				}
			}
		}

		mw.saveGames()
		mw.gameList.Refresh()
	}()
}

// showSettings shows the settings dialog
func (mw *MainWindow) showSettings() {
	intervalEntry := widget.NewEntry()
	intervalEntry.SetText(fmt.Sprintf("%d", mw.settings.CheckInterval))

	notificationsCheck := widget.NewCheck("Enable Notifications", nil)
	notificationsCheck.SetChecked(mw.settings.Notifications)

	form := dialog.NewForm("Settings", "Save", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Check Interval (seconds)", intervalEntry),
			widget.NewFormItem("", notificationsCheck),
		},
		func(confirm bool) {
			if !confirm {
				return
			}

			// Update settings
			if interval, err := fmt.Sscanf(intervalEntry.Text, "%d", &mw.settings.CheckInterval); err != nil || interval == 0 {
				mw.settings.CheckInterval = 3600
			}
			mw.settings.Notifications = notificationsCheck.Checked

			mw.saveSettings()
			mw.restartUpdateTimer()
		},
		mw.window)

	form.Resize(fyne.NewSize(400, 200))
	form.Show()
}

// saveGames saves the games list to storage
func (mw *MainWindow) saveGames() {
	err := mw.storage.SaveGames(mw.games)
	if err != nil {
		dialog.ShowError(err, mw.window)
	}
}

// saveSettings saves the settings to storage
func (mw *MainWindow) saveSettings() {
	err := mw.storage.SaveSettings(mw.settings)
	if err != nil {
		dialog.ShowError(err, mw.window)
	}
}

// startUpdateTimer starts the periodic update timer
func (mw *MainWindow) startUpdateTimer() {
	if mw.settings.CheckInterval > 0 {
		mw.refreshTimer = time.AfterFunc(time.Duration(mw.settings.CheckInterval)*time.Second, func() {
			mw.checkAllUpdates()
			mw.startUpdateTimer() // Restart timer
		})
	}
}

// restartUpdateTimer restarts the update timer
func (mw *MainWindow) restartUpdateTimer() {
	if mw.refreshTimer != nil {
		mw.refreshTimer.Stop()
	}
	mw.startUpdateTimer()
}
