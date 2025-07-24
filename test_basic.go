package main

import (
	"testing"
	"gamelauncher/models"
	"gamelauncher/storage"
	"gamelauncher/game"
	"gamelauncher/monitor"
)

// TestBasicStructures tests that basic structures can be created
func TestBasicStructures(t *testing.T) {
	// Test game creation
	testGame := models.NewGame("Test Game", "/path/to/game.exe", "/path/to/folder")
	if testGame.Name != "Test Game" {
		t.Errorf("Expected game name 'Test Game', got '%s'", testGame.Name)
	}
	
	// Test settings creation
	settings := models.DefaultSettings()
	if settings.CheckInterval != 3600 {
		t.Errorf("Expected check interval 3600, got %d", settings.CheckInterval)
	}
	
	// Test managers creation
	storageManager := storage.NewManager()
	if storageManager == nil {
		t.Error("Storage manager should not be nil")
	}
	
	gameManager := game.NewManager()
	if gameManager == nil {
		t.Error("Game manager should not be nil")
	}
	
	sourceMonitor := monitor.NewSourceMonitor()
	if sourceMonitor == nil {
		t.Error("Source monitor should not be nil")
	}
}

// TestGameOperations tests basic game operations
func TestGameOperations(t *testing.T) {
	testGame := models.NewGame("Test Game", "/path/to/game.exe", "/path/to/folder")
	
	// Test update info
	testGame.UpdateInfo("1.0.0")
	if testGame.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", testGame.Version)
	}
	
	// Test mark checked
	originalTime := testGame.LastCheck
	testGame.MarkChecked()
	if testGame.LastCheck.Equal(originalTime) {
		t.Error("LastCheck should be updated when marking as checked")
	}
} 