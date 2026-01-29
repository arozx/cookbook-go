package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cookbook-go/recipe-tracker/internal/images"
	"github.com/cookbook-go/recipe-tracker/internal/parser"
	"github.com/cookbook-go/recipe-tracker/internal/storage"
	"github.com/cookbook-go/recipe-tracker/internal/ui"
)

func main() {
	// Get data directory (use ~/.recipe-tracker)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	dataDir := filepath.Join(homeDir, ".recipe-tracker")
	cacheDir := filepath.Join(dataDir, "images")

	// Initialize storage
	store, err := storage.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}

	// Initialize parser
	recipeParser := parser.NewParser()

	// Initialize image downloader
	downloader, err := images.NewDownloader(cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing image downloader: %v\n", err)
		os.Exit(1)
	}

	// Create the UI model
	model := ui.NewModel(store, recipeParser, downloader)

	// Run the program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
