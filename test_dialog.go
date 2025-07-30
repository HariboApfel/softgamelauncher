package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ncruces/zenity"
)

func main() {
	fmt.Println("Testing native file dialog...")
	
	// Check if zenity is available
	if !zenity.IsAvailable() {
		fmt.Println("Zenity is not available on this system")
		os.Exit(1)
	}
	
	fmt.Println("Zenity is available!")
	
	// Test opening a file dialog
	homeDir, _ := os.UserHomeDir()
	filename, err := zenity.SelectFile(
		zenity.Title("Test File Selection"),
		zenity.Filename(homeDir),
	)
	
	if err != nil {
		if err == zenity.ErrCanceled {
			fmt.Println("User cancelled the dialog")
		} else {
			log.Printf("Error: %v", err)
		}
		return
	}
	
	if filename != "" {
		fmt.Printf("Selected file: %s\n", filename)
	} else {
		fmt.Println("No file selected")
	}
}