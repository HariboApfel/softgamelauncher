package main

import (
	"fmt"
	"gamelauncher/game"
	"gamelauncher/search"
	"gamelauncher/storage"
	"gamelauncher/ui"
	"log"
	"os"
	"strconv"
)

func main() {
	// Check for command-line arguments
	if len(os.Args) > 1 {
		handleCommandLineArgs()
		return
	}

	// Normal GUI mode
	log.Println("Starting Game Launcher...")
	app := ui.NewMainWindow()
	app.ShowAndRun()
}

// handleCommandLineArgs processes command-line arguments
func handleCommandLineArgs() {
	args := os.Args[1:]

	if len(args) == 0 {
		showUsage()
		return
	}

	switch args[0] {
	case "-game", "--game":
		if len(args) < 2 {
			fmt.Println("Error: Game number required")
			showUsage()
			return
		}
		launchGameByNumber(args[1])
	case "-list", "--list":
		listGames()
	case "-search", "--search":
		if len(args) < 2 {
			fmt.Println("Error: Game name required")
			showUsage()
			return
		}
		searchForGame(args[1])
	case "-help", "--help", "-h", "--h":
		showUsage()
	default:
		fmt.Printf("Unknown option: %s\n", args[0])
		showUsage()
	}
}

// launchGameByNumber launches a game by its number in the list
func launchGameByNumber(gameNumber string) {
	// Load games from storage
	storage := storage.NewManager()
	games, err := storage.LoadGames()
	if err != nil {
		fmt.Printf("Error loading games: %v\n", err)
		return
	}

	if len(games) == 0 {
		fmt.Println("No games found.")
		return
	}

	// Parse game number
	num, err := strconv.Atoi(gameNumber)
	if err != nil {
		fmt.Printf("Invalid game number: %s\n", gameNumber)
		return
	}

	// Convert to 0-based index
	index := num - 1
	if index < 0 || index >= len(games) {
		fmt.Printf("Game number %d not found. Available games:\n", num)
		listGames()
		return
	}

	// Launch the game
	gameItem := games[index]
	fmt.Printf("Launching %s...\n", gameItem.Name)

	gameManager := game.NewManager()
	err = gameManager.LaunchGame(gameItem)
	if err != nil {
		fmt.Printf("Error launching game: %v\n", err)
	} else {
		fmt.Printf("Successfully launched %s\n", gameItem.Name)
	}
}

// listGames lists all available games
func listGames() {
	storage := storage.NewManager()
	games, err := storage.LoadGames()
	if err != nil {
		fmt.Printf("Error loading games: %v\n", err)
		return
	}

	if len(games) == 0 {
		fmt.Println("No games found.")
		return
	}

	fmt.Println("Available games:")
	fmt.Println("================")
	for i, gameItem := range games {
		fmt.Printf("%d. %s\n", i+1, gameItem.Name)
		fmt.Printf("   Executable: %s\n", gameItem.Executable)
		if gameItem.CurrentVersion != "" {
			fmt.Printf("   Version: %s\n", gameItem.CurrentVersion)
		}
		fmt.Println()
	}
}

// searchForGame searches for a game on F95Zone and displays the results
func searchForGame(gameName string) {
	searchService := search.NewService()

	fmt.Printf("Searching for '%s' on F95Zone...\n", gameName)

	results, err := searchService.SearchGame(gameName)
	if err != nil {
		fmt.Printf("Error searching for game: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Printf("No matches found for '%s' on F95Zone.\n", gameName)
		return
	}

	fmt.Printf("\nFound %d matches for '%s':\n", len(results), gameName)
	fmt.Println("==========================================")

	for i, result := range results {
		score := fmt.Sprintf("%.1f%%", result.MatchScore*100)
		fmt.Printf("%d. [%s] %s\n", i+1, score, result.Title)
		fmt.Printf("   Link: %s\n", result.Link)
		if result.Description != "" {
			fmt.Printf("   Description: %s\n", result.Description)
		}
		fmt.Println()
	}

	// Show the best match
	if len(results) > 0 {
		bestMatch := results[0]
		for _, result := range results {
			if result.MatchScore > bestMatch.MatchScore {
				bestMatch = result
			}
		}

		fmt.Printf("Best match: %s (%.1f%%)\n", bestMatch.Title, bestMatch.MatchScore*100)
		fmt.Printf("Link: %s\n", bestMatch.Link)
	}
}

// showUsage displays command-line usage information
func showUsage() {
	fmt.Println("Game Launcher - Command Line Usage")
	fmt.Println("==================================")
	fmt.Println()
	fmt.Println("GUI Mode (default):")
	fmt.Println("  gamelauncher.exe")
	fmt.Println()
	fmt.Println("Command Line Options:")
	fmt.Println("  -game <number>     Launch game by number")
	fmt.Println("  -list              List all available games")
	fmt.Println("  -search <name>     Search for game on F95Zone")
	fmt.Println("  -help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gamelauncher.exe -game 1        # Launch the first game")
	fmt.Println("  gamelauncher.exe -list          # List all games")
	fmt.Println("  gamelauncher.exe -search \"My Pig Princess\"  # Search for a game")
	fmt.Println("  gamelauncher.exe -help          # Show help")
}
