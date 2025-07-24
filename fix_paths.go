package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"gamelauncher/models"
	"gamelauncher/storage"
)

func main() {
	fmt.Println("Game Launcher - Path Fixer")
	fmt.Println("==========================")
	
	// Create storage manager
	storageManager := storage.NewManager()
	
	// Load existing games
	games, err := storageManager.LoadGames()
	if err != nil {
		fmt.Printf("Error loading games: %v\n", err)
		return
	}
	
	if len(games) == 0 {
		fmt.Println("No games found to fix.")
		return
	}
	
	fmt.Printf("Found %d games to check...\n", len(games))
	
	fixedCount := 0
	for i, game := range games {
		originalExecutable := game.Executable
		originalFolder := game.Folder
		
		// Clean paths
		game.Executable = cleanPath(game.Executable)
		game.Folder = cleanPath(game.Folder)
		
		// Check if paths were changed
		if originalExecutable != game.Executable || originalFolder != game.Folder {
			fmt.Printf("Game %d: %s\n", i+1, game.Name)
			if originalExecutable != game.Executable {
				fmt.Printf("  Executable: %s -> %s\n", originalExecutable, game.Executable)
			}
			if originalFolder != game.Folder {
				fmt.Printf("  Folder: %s -> %s\n", originalFolder, game.Folder)
			}
			fixedCount++
		}
	}
	
	if fixedCount > 0 {
		// Save the fixed games
		err = storageManager.SaveGames(games)
		if err != nil {
			fmt.Printf("Error saving fixed games: %v\n", err)
			return
		}
		
		fmt.Printf("\nFixed %d games with path issues.\n", fixedCount)
	} else {
		fmt.Println("No path issues found.")
	}
}

// cleanPath cleans and normalizes a file path
func cleanPath(path string) string {
	// Remove surrounding quotes
	path = strings.Trim(path, `"'`)
	
	// Normalize path separators
	path = filepath.Clean(path)
	
	return path
} 