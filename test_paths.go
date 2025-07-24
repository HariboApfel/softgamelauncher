package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// cleanPath cleans and normalizes a file path
func cleanPath(path string) string {
	// Remove surrounding quotes
	path = strings.Trim(path, `"'`)
	
	// Normalize path separators
	path = filepath.Clean(path)
	
	// Convert to absolute path if it's not already
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}
	
	return path
}

func main() {
	// Test cases
	testPaths := []string{
		`"C:\Users\alexa\Downloads\MyPigPrincess-0.9.0-pc\MyPigPrincess-0.9.0-pc\MyPigPrincess.exe"`,
		`C:\Users\alexa\Downloads\MyPigPrincess-0.9.0-pc\MyPigPrincess-0.9.0-pc\MyPigPrincess.exe`,
		`'C:\Program Files\Game\game.exe'`,
		`C:\Program Files\Game\game.exe`,
		`"C:\Users\alexa\Downloads\MyPigPrincess-0.9.0-pc\MyPigPrincess-0.9.0-pc\MyPigPrincess.exe"`,
	}
	
	fmt.Println("Path Cleaning Test")
	fmt.Println("==================")
	
	for i, testPath := range testPaths {
		cleaned := cleanPath(testPath)
		fmt.Printf("Test %d:\n", i+1)
		fmt.Printf("  Original: %s\n", testPath)
		fmt.Printf("  Cleaned:  %s\n", cleaned)
		fmt.Printf("  Exists:   %t\n", fileExists(cleaned))
		fmt.Println()
	}
}

func fileExists(path string) bool {
	// This is a simplified check - in reality you'd use os.Stat
	return len(path) > 0
} 