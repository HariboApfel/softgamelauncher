package ui

import (
	"fmt"
	"gamelauncher/game"
	"gamelauncher/models"
	"gamelauncher/monitor"
	"gamelauncher/search"
	"gamelauncher/steam"
	"gamelauncher/storage"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fynestorage "fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
)

// MainWindow represents the main application window
type MainWindow struct {
	app           fyne.App
	window        fyne.Window
	gameManager   *game.Manager
	storage       *storage.Manager
	monitor       *monitor.SourceMonitor
	searchService *search.Service
	steamManager  *steam.Manager
	games         []*models.Game
	gamesMutex    sync.RWMutex // Protects concurrent access to games slice
	settings      *models.Settings
	gameList      *widget.List
	refreshTimer  *time.Timer
	selectedGame  int // Track selected game index
}

// NewMainWindow creates a new main window
func NewMainWindow() *MainWindow {
	myApp := app.New()
	myApp.SetIcon(theme.ComputerIcon())

	window := myApp.NewWindow("Game Launcher")
	window.Resize(fyne.NewSize(800, 600))

	mw := &MainWindow{
		app:           myApp,
		window:        window,
		gameManager:   game.NewManager(),
		storage:       storage.NewManager(),
		monitor:       monitor.NewSourceMonitor(),
		searchService: search.NewService(),
		steamManager:  steam.NewManager(),
		selectedGame:  -1, // Initialize to no selection
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
		func() int {
			mw.gamesMutex.RLock()
			defer mw.gamesMutex.RUnlock()
			return len(mw.games)
		},
		func() fyne.CanvasObject {
			// Create image and name container on the left
			gameImage := canvas.NewImageFromResource(theme.ComputerIcon())
			gameImage.SetMinSize(fyne.NewSize(60, 40))
			gameImage.FillMode = canvas.ImageFillContain

			nameLabel := widget.NewLabel("Game Name")
			nameContainer := container.NewHBox(gameImage, nameLabel)

			// Create right-side container with all other elements
			rightContainer := container.NewHBox()

			// Current Version column - compact
			currentVersionLabel := widget.NewLabel("Current Version")
			currentVersionContainer := container.NewHBox(currentVersionLabel)

			// Fetched Version column - compact
			fetchedVersionLabel := NewColoredLabel("Fetched Version", color.White, color.Black)
			fetchedVersionContainer := container.NewHBox(fetchedVersionLabel)

			// Source URL column - compact
			sourceURLLabel := widget.NewLabel("Source URL")
			sourceURLContainer := container.NewHBox(sourceURLLabel)

			// Launch button column - compact
			launchBtn := widget.NewButton("Launch", nil)
			launchContainer := container.NewHBox(launchBtn)

			// Edit button column - compact
			editBtn := widget.NewButton("Edit", nil)
			editContainer := container.NewHBox(editBtn)

			// Add all right-side elements
			rightContainer.Add(currentVersionContainer)
			rightContainer.Add(fetchedVersionContainer)
			rightContainer.Add(sourceURLContainer)
			rightContainer.Add(launchContainer)
			rightContainer.Add(editContainer)

			// Use Border to put image+name on left, everything else on right side
			return container.NewBorder(nil, nil, nil, rightContainer, nameContainer)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			mw.gamesMutex.RLock()
			if int(id) >= len(mw.games) {
				mw.gamesMutex.RUnlock()
				return // Prevent index out of bounds
			}
			game := mw.games[id]
			mw.gamesMutex.RUnlock()
			borderContainer := obj.(*fyne.Container)

			// Border structure: [center, right] - only 2 objects
			if len(borderContainer.Objects) >= 2 {
				// Update image and name (center - index 0)
				if nameContainer, ok := borderContainer.Objects[0].(*fyne.Container); ok {
					if len(nameContainer.Objects) >= 2 {
						// Update image (first element)
						if gameImage, ok := nameContainer.Objects[0].(*canvas.Image); ok {
							// Try to load game image if available
							if game.ImagePath != "" {
								// Check if image file exists
								if _, err := os.Stat(game.ImagePath); err == nil {
									// Load image from file
									fmt.Printf("DEBUG: Loading image for %s: %s\n", game.Name, game.ImagePath)
									gameImage.File = game.ImagePath
									gameImage.Resource = nil // Clear resource to use file
								} else {
									// Image file is missing, just show default icon
									fmt.Printf("DEBUG: Image file missing for %s, using default icon\n", game.Name)
									gameImage.File = "" // Clear file
									gameImage.Resource = theme.ComputerIcon()
								}
							} else {
								fmt.Printf("DEBUG: No image path for %s, using default icon\n", game.Name)
								gameImage.File = "" // Clear file
								gameImage.Resource = theme.ComputerIcon()
							}
							gameImage.Refresh()
						}

						// Update name label (second element)
						if nameLabel, ok := nameContainer.Objects[1].(*widget.Label); ok {
							nameLabel.SetText(game.Name)
						}
					}
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

						// Update source URL (third element)
						if sourceURLContainer, ok := rightContainer.Objects[2].(*fyne.Container); ok {
							if len(sourceURLContainer.Objects) > 0 {
								if sourceURLLabel, ok := sourceURLContainer.Objects[0].(*widget.Label); ok {
									sourceURLLabel.SetText(mw.truncateText(game.SourceURL, 20))
								}
							}
						}

						// Update launch button (fourth element)
						if launchContainer, ok := rightContainer.Objects[3].(*fyne.Container); ok {
							if len(launchContainer.Objects) > 0 {
								if launchBtn, ok := launchContainer.Objects[0].(*widget.Button); ok {
									launchBtn.OnTapped = func() {
										mw.launchGame(game)
									}
								}
							}
						}

						// Update edit button (fifth element)
						if editContainer, ok := rightContainer.Objects[4].(*fyne.Container); ok {
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
		widget.NewToolbarAction(theme.SearchIcon(), func() {
			mw.searchForGame()
		}),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			mw.checkAllUpdates()
		}),
		widget.NewToolbarAction(theme.DownloadIcon(), func() {
			mw.fetchImagesForAllGames()
		}),
		widget.NewToolbarAction(theme.ComputerIcon(), func() {
			mw.addSelectedGameToSteam()
		}),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
			mw.cleanupSteamShortcuts()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			mw.showSettings()
		}),
	)
}

// getLastUsedPath returns the last used path or user's home directory
func (mw *MainWindow) getLastUsedPath() string {
	if mw.settings.LastUsedPath != "" {
		return mw.settings.LastUsedPath
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return homeDir
}

// saveLastUsedPath saves the last used directory path
func (mw *MainWindow) saveLastUsedPath(path string) {
	if path != "" {
		// Extract directory from path if it's a file
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			path = filepath.Dir(path)
		}
		mw.settings.LastUsedPath = path
		mw.saveSettings()
	}
}

// openNativeFileDialog opens the system's native file dialog
// Priority order: 1) Dolphin/kdialog (KDE), 2) Zenity (GTK), 3) Fyne (fallback)
func (mw *MainWindow) openNativeFileDialog() (string, error) {
	startPath := mw.getLastUsedPath()
	
	// Try Dolphin first (KDE file manager)
	if mw.isDolphinAvailable() {
		if filename, err := mw.openDolphinFileDialog(startPath); err == nil {
			if filename != "" {
				mw.saveLastUsedPath(filename)
			}
			return filename, nil
		}
		// If Dolphin fails, continue to other options
	}
	
	// Check if zenity is available as second option
	if zenity.IsAvailable() {
		filename, err := zenity.SelectFile(
			zenity.Title("Select Executable"),
			zenity.Filename(startPath),
			zenity.FileFilters{
				{"Executable files", []string{"*.exe", "*.sh", "*.run", "*.AppImage"}, false},
				{"All files", []string{"*"}, false},
			},
		)
		
		if err != nil {
			// Check if user cancelled
			if err == zenity.ErrCanceled {
				return "", nil
			}
			// On error, fallback to Fyne dialog
			return mw.openFyneFileDialog(startPath)
		}
		
		// Save the directory for future use
		if filename != "" {
			mw.saveLastUsedPath(filename)
		}
		
		return filename, nil
	}
	
	// Fallback to Fyne file dialog if neither Dolphin nor zenity is available
	return mw.openFyneFileDialog(startPath)
}

// openFyneFileDialog is a fallback that uses the Fyne file dialog
func (mw *MainWindow) openFyneFileDialog(startPath string) (string, error) {
	// Create a channel to receive the result
	resultChan := make(chan string, 1)
	errorChan := make(chan error, 1)
	
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			errorChan <- err
			return
		}
		if reader == nil {
			resultChan <- "" // User cancelled
			return
		}
		defer reader.Close()
		selectedPath := reader.URI().Path()
		resultChan <- selectedPath
	}, mw.window)
	
	// Set the starting location to the last used path
	if startPath != "" {
		if listableURI := fynestorage.NewFileURI(startPath); listableURI != nil {
			if listable, ok := listableURI.(fyne.ListableURI); ok {
				fileDialog.SetLocation(listable)
			}
		}
	}
	
	fileDialog.Show()
	
	// Wait for result
	select {
	case filename := <-resultChan:
		if filename != "" {
			mw.saveLastUsedPath(filename)
		}
		return filename, nil
	case err := <-errorChan:
		return "", err
	}
}

// isDolphinAvailable checks if Dolphin file manager is available
func (mw *MainWindow) isDolphinAvailable() bool {
	// Check if kdialog is available, which is the actual dialog tool we'll use
	// kdialog comes with KDE/Dolphin installations
	_, err := exec.LookPath("kdialog")
	if err != nil {
		return false
	}
	
	// Optionally also check for dolphin itself
	_, err = exec.LookPath("dolphin")
	return err == nil
}

// openDolphinFileDialog opens a file dialog using Dolphin
func (mw *MainWindow) openDolphinFileDialog(startPath string) (string, error) {
	// Dolphin command for file selection: dolphin --select <path>
	// However, for file picking, we'll use a different approach
	// Since Dolphin doesn't have a direct file picker mode, we'll use kdialog instead
	// which is the KDE dialog utility that Dolphin/KDE uses
	
	// Check if kdialog is available (comes with KDE/Dolphin)
	if _, err := exec.LookPath("kdialog"); err != nil {
		return "", err
	}
	
	// Build kdialog command for file selection
	args := []string{
		"--getopenfilename",
		startPath,
		"*.exe *.sh *.run *.AppImage *", // Common executable file filters
		"--title", "Select Executable",
	}
	
	cmd := exec.Command("kdialog", args...)
	output, err := cmd.Output()
	
	if err != nil {
		// Check if this is due to user cancellation
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// Exit code 1 typically means user cancelled
			return "", nil
		}
		return "", err
	}
	
	filename := strings.TrimSpace(string(output))
	return filename, nil
}

// importGames shows a dialog to import games from a folder
func (mw *MainWindow) importGames() {
	// Create a file dialog that starts at the last used path
	folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			if err != nil {
				dialog.ShowError(err, mw.window)
			}
			return
		}

		// Save the selected path for future use
		mw.saveLastUsedPath(uri.Path())

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
			// Check if game already exists by name (case-insensitive)
			exists := false
			normalizedNewName := strings.ToLower(strings.TrimSpace(newGame.Name))

			for _, existingGame := range mw.games {
				normalizedExistingName := strings.ToLower(strings.TrimSpace(existingGame.Name))
				if normalizedExistingName == normalizedNewName {
					// Game with same name exists, update the executable path instead of adding duplicate
					existingGame.Executable = newGame.Executable
					existingGame.Folder = newGame.Folder
					exists = true
					break
				}
			}

			if !exists {
				mw.gamesMutex.Lock()
				mw.games = append(mw.games, newGame)
				mw.gamesMutex.Unlock()
			}
		}

		mw.saveGames()
		mw.gameList.Refresh()

		dialog.ShowInformation("Import Complete",
			fmt.Sprintf("Imported %d new games.", len(games)), mw.window)
	}, mw.window)
	
	// Set the starting location to the last used path
	if startLocation := mw.getLastUsedPath(); startLocation != "" {
		if listableURI := fynestorage.NewFileURI(startLocation); listableURI != nil {
			if listable, ok := listableURI.(fyne.ListableURI); ok {
				folderDialog.SetLocation(listable)
			}
		}
	}
	
	folderDialog.Show()
}

// addGame shows a dialog to manually add a game
func (mw *MainWindow) addGame() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Game Name")

	execEntry := widget.NewEntry()
	execEntry.SetPlaceHolder("Click 'Browse' to select executable")
	execEntry.Disable() // Make it read-only

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Source URL (will be auto-filled if found)")

	// Create browse button
	browseBtn := widget.NewButton("Browse", func() {
		selectedPath, err := mw.openNativeFileDialog()
		if err != nil {
			dialog.ShowError(err, mw.window)
			return
		}
		if selectedPath != "" {
			execEntry.SetText(selectedPath)
		}
	})

	// Create executable selection container
	execContainer := container.NewBorder(nil, nil, nil, browseBtn, execEntry)

	// Add search button
	searchBtn := widget.NewButton("Search for Link", func() {
		if nameEntry.Text == "" {
			dialog.ShowInformation("No Game Name", "Please enter a game name first.", mw.window)
			return
		}
		mw.autoSearchForGame(nameEntry.Text, urlEntry)
	})

	// Create URL container with search button
	urlContainer := container.NewBorder(nil, nil, nil, searchBtn, urlEntry)

	form := dialog.NewForm("Add Game", "Add", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Executable", execContainer),
			widget.NewFormItem("Source URL", urlContainer),
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

			mw.gamesMutex.Lock()
			mw.games = append(mw.games, newGame)
			mw.gamesMutex.Unlock()

			mw.saveGames()
			mw.gameList.Refresh()
		},
		mw.window)

	form.Resize(fyne.NewSize(500, 300))
	form.Show()
}

// autoSearchForGame automatically searches for a game link and updates the URL entry
func (mw *MainWindow) autoSearchForGame(gameName string, urlEntry *widget.Entry) {
	// Show progress dialog
	progress := dialog.NewProgress("Searching", "Searching for game link...", mw.window)
	progress.Show()

	go func() {
		defer progress.Hide()

		// Search for the game
		results, err := mw.searchService.SearchGame(gameName)
		if err != nil {
			dialog.ShowError(fmt.Errorf("search failed: %w", err), mw.window)
			return
		}

		if len(results) == 0 {
			dialog.ShowInformation("No Results",
				fmt.Sprintf("No matches found for '%s' on F95Zone.", gameName), mw.window)
			return
		}

		// Find the best match
		bestMatch := results[0]
		for _, result := range results {
			if result.MatchScore > bestMatch.MatchScore {
				bestMatch = result
			}
		}

		// If we have a good match (above 70%), auto-fill it
		if bestMatch.MatchScore > 0.7 {
			// Download image for the best match
			if bestMatch.ImageURL != "" {
				err := mw.searchService.DownloadImageForResult(&bestMatch)
				if err != nil {
					fmt.Printf("Warning: Failed to download image: %v\n", err)
				} else {
					fmt.Printf("Downloaded image to: %s\n", bestMatch.ImagePath)
				}
			}

			// Update UI on main thread using the app's main thread
			mw.app.SendNotification(&fyne.Notification{
				Title: fmt.Sprintf("Link Found for %s", gameName),
			})

			// Use the main thread to update the entry
			urlEntry.SetText(bestMatch.Link)
			urlEntry.Refresh()

			dialog.ShowInformation("Link Found",
				fmt.Sprintf("Auto-filled source URL for '%s':\n%s", gameName, bestMatch.Link), mw.window)
		} else {
			// Show results dialog for manual selection
			mw.showSearchResultsForNewGame(gameName, results, urlEntry)
		}
	}()
}

// showSearchResultsForNewGame shows search results for a new game being added
func (mw *MainWindow) showSearchResultsForNewGame(gameName string, results []search.SearchResult, urlEntry *widget.Entry) {
	var selectedIndex int

	// Create a list widget for results with images
	resultList := widget.NewList(
		func() int { return len(results) },
		func() fyne.CanvasObject {
			// Create a container with image and text
			image := canvas.NewImageFromResource(theme.ComputerIcon())
			image.SetMinSize(fyne.NewSize(80, 60))
			image.FillMode = canvas.ImageFillContain

			scoreLabel := widget.NewLabel("Score")
			titleLabel := widget.NewLabel("Title")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}

			// Create a vertical container for text
			textContainer := container.NewVBox(scoreLabel, titleLabel)

			// Create a horizontal container with image and text
			return container.NewHBox(image, textContainer)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			result := results[id]
			container := obj.(*fyne.Container)

			// Update image if available
			if len(container.Objects) > 0 {
				if image, ok := container.Objects[0].(*canvas.Image); ok {
					if result.ImagePath != "" {
						// Load image from file
						image.File = result.ImagePath
						image.Refresh()
					} else {
						// Use default icon
						image.Resource = theme.ComputerIcon()
						image.Refresh()
					}
				}
			}

			// Update text
			if len(container.Objects) > 1 {
				if textContainer, ok := container.Objects[1].(*fyne.Container); ok {
					if len(textContainer.Objects) > 0 {
						if scoreLabel, ok := textContainer.Objects[0].(*widget.Label); ok {
							score := fmt.Sprintf("%.1f%%", result.MatchScore*100)
							scoreLabel.SetText(score)
						}
					}
					if len(textContainer.Objects) > 1 {
						if titleLabel, ok := textContainer.Objects[1].(*widget.Label); ok {
							titleLabel.SetText(result.Title)
						}
					}
				}
			}
		},
	)

	// Track selected item
	resultList.OnSelected = func(id widget.ListItemID) {
		selectedIndex = int(id)
	}

	// Create dialog content
	content := container.NewBorder(
		widget.NewLabel(fmt.Sprintf("Search results for '%s':", gameName)),
		nil, nil, nil,
		resultList,
	)

	// Create the dialog
	dialog.ShowCustomConfirm("Search Results", "Select", "Cancel", content,
		func(confirm bool) {
			if !confirm {
				return
			}

			// Get selected result
			if selectedIndex < 0 || selectedIndex >= len(results) {
				return
			}

			selectedResult := results[selectedIndex]

			// Download image for the selected result
			if selectedResult.ImageURL != "" {
				err := mw.searchService.DownloadImageForResult(&selectedResult)
				if err != nil {
					fmt.Printf("Warning: Failed to download image: %v\n", err)
				} else {
					fmt.Printf("Downloaded image to: %s\n", selectedResult.ImagePath)
				}
			}

			// Update the URL entry on main thread
			urlEntry.SetText(selectedResult.Link)
			urlEntry.Refresh()

			// Store the image path for the new game
			if selectedResult.ImagePath != "" {
				// We'll need to update the game's ImagePath when it's created
				// This will be handled in the form submission
			}

			dialog.ShowInformation("Link Selected",
				fmt.Sprintf("Source URL updated to:\n%s", selectedResult.Link), mw.window)
		}, mw.window)

	// Set initial selection
	if len(results) > 0 {
		resultList.Select(0)
	}
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
		selectedPath, err := mw.openNativeFileDialog()
		if err != nil {
			dialog.ShowError(err, mw.window)
			return
		}
		if selectedPath != "" {
			execEntry.SetText(selectedPath)
		}
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
	mw.gamesMutex.RLock()
	if mw.selectedGame < 0 || mw.selectedGame >= len(mw.games) {
		mw.gamesMutex.RUnlock()
		dialog.ShowInformation("No Game Selected",
			"Please select a game to delete.", mw.window)
		return
	}

	game := mw.games[mw.selectedGame]
	mw.gamesMutex.RUnlock()

	// Show confirmation dialog
	dialog.ShowConfirm("Delete Game",
		fmt.Sprintf("Are you sure you want to delete '%s'?\n\nThis action cannot be undone.", game.Name),
		func(confirm bool) {
			if !confirm {
				return
			}

			// Remove game from slice
			mw.gamesMutex.Lock()
			mw.games = append(mw.games[:mw.selectedGame], mw.games[mw.selectedGame+1:]...)
			mw.gamesMutex.Unlock()

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

// updateFetchedVersionLabel updates the fetched version label with cached information only
func (mw *MainWindow) updateFetchedVersionLabel(game *models.Game, label *ColoredLabel) {
	// If no source URL, show as unavailable
	if game.SourceURL == "" {
		label.SetText("No source")
		label.Refresh()
		return
	}

	// Check if we have cached version information
	if game.Version != "" {
		// Display cached version information
		var displayText string
		var bgColor, textColor color.Color

		if game.Version == game.CurrentVersion {
			// Same version - show with green background
			displayText = game.Version
			bgColor = color.NRGBA{R: 0, G: 255, B: 0, A: 100} // Light green
			textColor = color.Black
		} else if game.Version != "" && game.Version != game.CurrentVersion {
			// Different version - determine if it's newer
			if mw.isVersionNewer(game.Version, game.CurrentVersion) {
				// Newer version available - show with red background
				displayText = game.Version + " [NEW]"
				bgColor = color.NRGBA{R: 255, G: 0, B: 0, A: 100} // Light red
				textColor = color.White
			} else {
				// Different version but not newer - show with orange background
				displayText = game.Version + " [DIFF]"
				bgColor = color.NRGBA{R: 255, G: 165, B: 0, A: 100} // Light orange
				textColor = color.Black
			}
		} else {
			// No cached version
			displayText = "Not checked"
			bgColor = color.NRGBA{R: 128, G: 128, B: 128, A: 100} // Light gray
			textColor = color.Black
		}

		// Update the label
		label.SetText(mw.truncateText(displayText, 20))
		label.SetColors(bgColor, textColor)
		label.Refresh()
	} else {
		// No cached version - show as not checked
		label.SetText("Not checked")
		label.SetColors(color.NRGBA{R: 128, G: 128, B: 128, A: 100}, color.Black)
		label.Refresh()
	}
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
	// Run initial version checks for all games at startup
	go func() {
		// Get a copy of games to iterate over (to avoid holding lock for too long)
		mw.gamesMutex.RLock()
		gamesCopy := make([]*models.Game, len(mw.games))
		copy(gamesCopy, mw.games)
		mw.gamesMutex.RUnlock()

		fmt.Printf("DEBUG: Running startup version checks for %d games\n", len(gamesCopy))

		for _, game := range gamesCopy {
			if game.SourceURL != "" {
				fmt.Printf("DEBUG: Checking version for %s\n", game.Name)
				updateInfo, err := mw.monitor.CheckForUpdates(game)
				if err == nil {
					game.UpdateInfo(updateInfo.Version)
					game.MarkChecked()
					fmt.Printf("DEBUG: Updated %s version to %s\n", game.Name, updateInfo.Version)
				} else {
					fmt.Printf("DEBUG: Error checking %s: %v\n", game.Name, err)
				}
			}
		}

		// Save the updated version information
		mw.saveGames()

		// Refresh the UI to show the updated version information
		mw.gameList.Refresh()
	}()
}

// checkAllUpdates checks for updates on all games
func (mw *MainWindow) checkAllUpdates() {
	progress := dialog.NewProgress("Checking Updates", "Checking for game updates...", mw.window)
	progress.Show()

	go func() {
		defer progress.Hide()

		// Get a copy of games to iterate over (to avoid holding lock for too long)
		mw.gamesMutex.RLock()
		gamesCopy := make([]*models.Game, len(mw.games))
		copy(gamesCopy, mw.games)
		mw.gamesMutex.RUnlock()

		for i, game := range gamesCopy {
			progress.SetValue(float64(i) / float64(len(gamesCopy)))

			if game.SourceURL != "" {
				updateInfo, err := mw.monitor.CheckForUpdates(game)
				if err == nil {
					game.UpdateInfo(updateInfo.Version)
					game.MarkChecked()

					// Show notification only if there's an update
					if updateInfo.HasUpdate && mw.settings.Notifications {
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
	mw.gamesMutex.RLock()
	defer mw.gamesMutex.RUnlock()

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
			// Stop the current timer before creating a new one to prevent memory leaks
			if mw.refreshTimer != nil {
				mw.refreshTimer.Stop()
			}
			mw.startUpdateTimer() // Restart timer
		})
	}
}

// restartUpdateTimer restarts the update timer
func (mw *MainWindow) restartUpdateTimer() {
	if mw.refreshTimer != nil {
		mw.refreshTimer.Stop()
		mw.refreshTimer = nil // Clear the reference
	}
	mw.startUpdateTimer()
}

// searchForGame searches for a game on F95Zone and allows the user to select a result
func (mw *MainWindow) searchForGame() {
	fmt.Printf("DEBUG: searchForGame called\n")

	// Check if a game is selected
	mw.gamesMutex.RLock()
	if mw.selectedGame < 0 || mw.selectedGame >= len(mw.games) {
		gameCount := len(mw.games)
		mw.gamesMutex.RUnlock()
		fmt.Printf("DEBUG: No game selected (selectedGame=%d, len(games)=%d)\n", mw.selectedGame, gameCount)
		dialog.ShowInformation("No Game Selected",
			"Please select a game to search for its source link.", mw.window)
		return
	}

	selectedGame := mw.games[mw.selectedGame]
	mw.gamesMutex.RUnlock()
	fmt.Printf("DEBUG: Selected game: %s (current SourceURL: %s)\n", selectedGame.Name, selectedGame.SourceURL)

	// Show progress dialog
	progress := dialog.NewProgress("Searching", "Searching for game links...", mw.window)
	progress.Show()

	go func() {
		defer progress.Hide()

		fmt.Printf("DEBUG: Starting search for game: %s\n", selectedGame.Name)

		// Search for the game
		results, err := mw.searchService.SearchGame(selectedGame.Name)
		if err != nil {
			fmt.Printf("DEBUG: Search error: %v\n", err)
			dialog.ShowError(fmt.Errorf("search failed: %w", err), mw.window)
			return
		}

		fmt.Printf("DEBUG: Found %d search results\n", len(results))
		for i, result := range results {
			fmt.Printf("DEBUG: Result %d: %s (score: %.2f)\n", i+1, result.Title, result.MatchScore)
		}

		if len(results) == 0 {
			fmt.Printf("DEBUG: No results found\n")
			dialog.ShowInformation("No Results",
				fmt.Sprintf("No matches found for '%s' on F95Zone.", selectedGame.Name), mw.window)
			return
		}

		// Find the best match
		bestMatch := results[0]
		for _, result := range results {
			if result.MatchScore > bestMatch.MatchScore {
				bestMatch = result
			}
		}

		fmt.Printf("DEBUG: Best match: %s (score: %.2f)\n", bestMatch.Title, bestMatch.MatchScore)

		// Directly update the game's source URL with the best match
		fmt.Printf("DEBUG: Updating game SourceURL from '%s' to '%s'\n", selectedGame.SourceURL, bestMatch.Link)
		selectedGame.SourceURL = bestMatch.Link

		// Save the changes
		fmt.Printf("DEBUG: Saving games to storage\n")
		mw.saveGames()
		fmt.Printf("DEBUG: Refreshing game list\n")
		mw.gameList.Refresh()

		fmt.Printf("DEBUG: Showing confirmation dialog\n")
		dialog.ShowInformation("Link Updated",
			fmt.Sprintf("Source URL updated for '%s' to:\n%s", selectedGame.Name, bestMatch.Link), mw.window)
	}()
}

// fetchImagesForAllGames downloads images for all games that have source URLs but no images
func (mw *MainWindow) fetchImagesForAllGames() {
	// Show progress dialog
	progress := dialog.NewProgress("Fetching Images", "Downloading images for all games...", mw.window)
	progress.Show()

	go func() {
		defer progress.Hide()

		// Get a copy of games to iterate over
		mw.gamesMutex.RLock()
		gamesCopy := make([]*models.Game, len(mw.games))
		copy(gamesCopy, mw.games)
		mw.gamesMutex.RUnlock()

		totalGames := len(gamesCopy)
		downloadedCount := 0
		failedCount := 0

		fmt.Printf("DEBUG: Starting image fetch for %d games\n", totalGames)

		for i, game := range gamesCopy {
			// Update progress
			progress.SetValue(float64(i) / float64(totalGames))

			fmt.Printf("DEBUG: Processing game %d/%d: %s (ImagePath: %s, SourceURL: %s)\n",
				i+1, totalGames, game.Name, game.ImagePath, game.SourceURL)

			// Skip games that already have valid images or no source URL
			if game.SourceURL == "" {
				fmt.Printf("DEBUG: Skipping %s - no source URL\n", game.Name)
				continue
			}

			// Check if image exists and is valid
			hasValidImage := false
			if game.ImagePath != "" {
				if _, err := os.Stat(game.ImagePath); err == nil {
					hasValidImage = true
					fmt.Printf("DEBUG: Skipping %s - valid image exists: %s\n", game.Name, game.ImagePath)
				} else {
					fmt.Printf("DEBUG: %s has ImagePath but file is missing: %s\n", game.Name, game.ImagePath)
				}
			}

			if hasValidImage {
				continue
			}

			// Search for the game to get image URL
			fmt.Printf("DEBUG: Searching for %s...\n", game.Name)
			results, err := mw.searchService.SearchGame(game.Name)
			if err != nil {
				fmt.Printf("DEBUG: Search failed for %s: %v\n", game.Name, err)
				failedCount++
				continue
			}

			fmt.Printf("DEBUG: Found %d search results for %s\n", len(results), game.Name)

			if len(results) > 0 {
				// Find the best match
				bestMatch := results[0]
				for _, result := range results {
					if result.MatchScore > bestMatch.MatchScore {
						bestMatch = result
					}
				}

				fmt.Printf("DEBUG: Best match for %s: %s (score: %.2f, imageURL: %s)\n",
					game.Name, bestMatch.Title, bestMatch.MatchScore, bestMatch.ImageURL)

				// Download image if we have a good match and image URL
				if bestMatch.MatchScore > 0.7 && bestMatch.ImageURL != "" {
					fmt.Printf("DEBUG: Attempting to download image for %s from %s\n", game.Name, bestMatch.ImageURL)
					err := mw.searchService.DownloadImageForResult(&bestMatch)
					if err == nil && bestMatch.ImagePath != "" {
						game.ImagePath = bestMatch.ImagePath
						downloadedCount++
						fmt.Printf("DEBUG: Successfully downloaded image for %s: %s\n", game.Name, game.ImagePath)
					} else {
						failedCount++
						fmt.Printf("DEBUG: Failed to download image for %s: %v\n", game.Name, err)
					}
				} else {
					fmt.Printf("DEBUG: Skipping download for %s - score: %.2f, imageURL: %s\n",
						game.Name, bestMatch.MatchScore, bestMatch.ImageURL)
				}
			} else {
				fmt.Printf("DEBUG: No search results found for %s\n", game.Name)
			}
		}

		// Save games with updated image paths
		mw.saveGames()
		mw.gameList.Refresh()

		// Show completion dialog
		dialog.ShowInformation("Image Fetch Complete",
			fmt.Sprintf("Downloaded %d images, %d failed.\nGames with images will now display them in the list.", downloadedCount, failedCount), mw.window)
	}()
}

// redownloadImageForGame attempts to re-download the image for a game
func (mw *MainWindow) redownloadImageForGame(game *models.Game) {
	// Search for the game to get image URL
	results, err := mw.searchService.SearchGame(game.Name)
	if err != nil {
		fmt.Printf("DEBUG: Failed to search for %s: %v\n", game.Name, err)
		return
	}

	if len(results) > 0 {
		// Find the best match
		bestMatch := results[0]
		for _, result := range results {
			if result.MatchScore > bestMatch.MatchScore {
				bestMatch = result
			}
		}

		// Download image if we have a good match and image URL
		if bestMatch.MatchScore > 0.7 && bestMatch.ImageURL != "" {
			err := mw.searchService.DownloadImageForResult(&bestMatch)
			if err == nil && bestMatch.ImagePath != "" {
				// Update the game's image path
				game.ImagePath = bestMatch.ImagePath
				mw.saveGames()
				mw.gameList.Refresh()
				fmt.Printf("DEBUG: Successfully re-downloaded image for %s: %s\n", game.Name, game.ImagePath)
			} else {
				fmt.Printf("DEBUG: Failed to re-download image for %s: %v\n", game.Name, err)
			}
		}
	}
}

// cleanupSteamShortcuts cleans up duplicate Steam shortcuts
func (mw *MainWindow) cleanupSteamShortcuts() {
	// Show confirmation dialog
	dialog.ShowConfirm("Clean Up Steam Shortcuts",
		"This will remove duplicate Steam shortcuts and update existing ones to use the new format.\n\nThis operation is safe and will preserve your game library.\n\nProceed with cleanup?",
		func(confirm bool) {
			if !confirm {
				return
			}

			// Show progress dialog
			progressDialog := dialog.NewProgressInfinite("Cleaning Up", "Cleaning up duplicate Steam shortcuts...", mw.window)
			progressDialog.Show()

			// Run cleanup in background
			go func() {
				err := mw.steamManager.CleanupDuplicateShortcuts()

				// Close progress dialog
				progressDialog.Hide()

				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to cleanup Steam shortcuts: %w", err), mw.window)
				} else {
					dialog.ShowInformation("Cleanup Complete",
						"Successfully cleaned up duplicate Steam shortcuts!\n\nNote: Steam must be restarted to see changes.", mw.window)
				}
			}()
		}, mw.window)
}

// addSelectedGameToSteam adds the currently selected game to Steam as a non-Steam shortcut
func (mw *MainWindow) addSelectedGameToSteam() {
	mw.gamesMutex.RLock()
	if mw.selectedGame < 0 || mw.selectedGame >= len(mw.games) {
		mw.gamesMutex.RUnlock()
		dialog.ShowInformation("No Game Selected",
			"Please select a game to add to Steam.", mw.window)
		return
	}

	selectedGame := mw.games[mw.selectedGame]
	mw.gamesMutex.RUnlock()

	// Check if game already exists in Steam
	exists, err := mw.steamManager.CheckGameExistsInSteam(selectedGame)
	if err != nil {
		// If we can't check, proceed anyway but log the warning
		fmt.Printf("Warning: Could not check if game exists in Steam: %v\n", err)
	}

	// Show confirmation dialog with Steam information
	appID := mw.steamManager.GetSteamAppID(selectedGame)
	steamURL := mw.steamManager.GetShortcutURL(appID)

	var actionText, titleText string
	if exists {
		actionText = "update"
		titleText = "Update Steam Shortcut"
	} else {
		actionText = "add"
		titleText = "Add to Steam"
	}

	message := fmt.Sprintf("%s '%s' %s Steam as a non-Steam shortcut?\n\nSteam App ID: %d\nSteam URL: %s\n\nNote: Steam must be restarted to see changes.",
		strings.Title(actionText), selectedGame.Name,
		map[bool]string{true: "in", false: "to"}[exists],
		appID, steamURL)

	dialog.ShowConfirm(titleText, message,
		func(confirm bool) {
			if !confirm {
				return
			}

			// Show progress dialog
			progressText := fmt.Sprintf("%sing game %s Steam...", strings.Title(actionText),
				map[bool]string{true: "in", false: "to"}[exists])
			progress := dialog.NewProgress(titleText, progressText, mw.window)
			progress.Show()

			go func() {
				defer progress.Hide()

				// Add game to Steam
				err := mw.steamManager.AddGameToSteam(selectedGame)
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to %s game %s Steam: %w", actionText,
						map[bool]string{true: "in", false: "to"}[exists], err), mw.window)
					return
				}

				// Show success dialog
				successMessage := fmt.Sprintf("Successfully %sd '%s' %s Steam!\n\nApp ID: %d\nSteam URL: %s\n\nPlease restart Steam to see the changes in your library.",
					actionText, selectedGame.Name,
					map[bool]string{true: "in", false: "to"}[exists],
					appID, steamURL)

				dialog.ShowInformation(fmt.Sprintf("%sd to Steam", strings.Title(actionText)), successMessage, mw.window)
			}()
		}, mw.window)
}
