package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cookbook-go/recipe-tracker/internal/client"
	"github.com/cookbook-go/recipe-tracker/internal/images"
	"github.com/cookbook-go/recipe-tracker/internal/ui"
)

func main() {
	// Command line flags
	serverURL := flag.String("server", "https://cook.jackx.dev", "Server URL to connect to (default: https://cook.jackx.dev)")
	cacheDir := flag.String("cache", "", "Cache directory for images (default: ~/.recipe-tracker/images)")
	flag.Parse()

	// Ensure URL has scheme
	if !strings.HasPrefix(*serverURL, "http://") && !strings.HasPrefix(*serverURL, "https://") {
		*serverURL = "http://" + *serverURL
	}

	// Default cache directory
	if *cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		*cacheDir = fmt.Sprintf("%s/.recipe-tracker/images", homeDir)
	}

	fmt.Printf("Connecting to %s...\n", *serverURL)

	// Create API client
	apiClient := client.NewClient(*serverURL)

	// Test connection
	if err := apiClient.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nMake sure the server is running.\n")
		os.Exit(1)
	}

	// Initialize image downloader for local caching
	downloader, err := images.NewDownloader(*cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing image downloader: %v\n", err)
		os.Exit(1)
	}

	// Create the UI model in remote mode
	model := ui.NewRemoteModel(apiClient, apiClient, downloader, *serverURL)

	// Subscribe to remote events and forward to TUI
	eventChan := apiClient.Subscribe()

	// Run the program
	p := tea.NewProgram(&model, tea.WithAltScreen())

	// Forward remote events to TUI
	go func() {
		for event := range eventChan {
			p.Send(ui.RemoteEventMsg{
				EventType: event.Type,
				Recipe:    event.Recipe,
				RecipeID:  event.RecipeID,
			})
		}
	}()

	fmt.Printf("Connected! Starting TUI...\n")
	time.Sleep(500 * time.Millisecond)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	apiClient.Unsubscribe(eventChan)
}
