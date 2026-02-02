package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cookbook-go/recipe-tracker/internal/api"
	"github.com/cookbook-go/recipe-tracker/internal/client"
	"github.com/cookbook-go/recipe-tracker/internal/images"
	"github.com/cookbook-go/recipe-tracker/internal/parser"
	"github.com/cookbook-go/recipe-tracker/internal/storage"
	"github.com/cookbook-go/recipe-tracker/internal/ui"
)

func main() {
	// Command line flags
	serverMode := flag.Bool("server", false, "Run as server (hosts webapp and API)")
	connectTo := flag.String("connect", "https://cook.jackx.dev", "Connect to a remote server (e.g., http://192.168.1.100:8080)")
	webPort := flag.String("port", "8080", "Port for server mode")
	syncTo := flag.String("sync", "", "Sync local changes to remote server (for local mode)")

	// Legacy flags for backward compatibility
	webOnly := flag.Bool("web", false, "Alias for -server")
	withWeb := flag.Bool("serve", false, "Run TUI with local web server (legacy mode)")

	flag.Parse()

	// Handle legacy flags
	if *webOnly {
		*serverMode = true
	}

	// Get data directory (use ~/.recipe-tracker)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	dataDir := filepath.Join(homeDir, ".recipe-tracker")
	cacheDir := filepath.Join(dataDir, "images")

	// Get local IP for display
	localIP := getLocalIP()
	webAddr := ":" + *webPort

	// MODE 1: Server mode - hosts the webapp and API
	if *serverMode {
		runServer(dataDir, webAddr, localIP, *webPort)
		return
	}

	// MODE 2: Client mode - connect to remote server
	if *connectTo != "" {
		runClient(*connectTo, cacheDir)
		return
	}

	// MODE 3: Legacy mode - TUI with optional local web server
	runLegacy(dataDir, cacheDir, webAddr, localIP, *webPort, *withWeb, *syncTo)
}

// runServer runs the application in server mode
func runServer(dataDir, webAddr, localIP, port string) {
	// Initialize storage
	store, err := storage.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}

	// Initialize parser
	recipeParser := parser.NewParser()

	// Create API server
	apiServer := api.NewServer(store, recipeParser)

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║       Recipe Tracker Server            ║")
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Printf("║  Local:   http://localhost:%-12s║\n", port)
	if localIP != "" {
		fmt.Printf("║  Network: http://%-15s:%-4s ║\n", localIP, port)
	}
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Println("║  Open the URL above on your phone or   ║")
	fmt.Println("║  connect with: recipe-tracker -connect ║")
	fmt.Printf("║    http://%s:%-4s                  ║\n", localIP, port)
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Println("║  Press Ctrl+C to stop                  ║")
	fmt.Println("╚════════════════════════════════════════╝")

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		apiServer.Stop(ctx)
		os.Exit(0)
	}()

	if err := apiServer.Start(webAddr); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}

// runClient runs the TUI connected to a remote server
func runClient(serverURL, cacheDir string) {
	// Ensure URL has scheme
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "http://" + serverURL
	}

	fmt.Printf("Connecting to %s...\n", serverURL)

	// Create API client
	apiClient := client.NewClient(serverURL)

	// Test connection
	if err := apiClient.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nMake sure the server is running:\n")
		fmt.Fprintf(os.Stderr, "  recipe-tracker -server\n")
		os.Exit(1)
	}

	// Initialize image downloader for local caching
	downloader, err := images.NewDownloader(cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing image downloader: %v\n", err)
		os.Exit(1)
	}

	// Create the UI model in remote mode
	model := ui.NewRemoteModel(apiClient, apiClient, downloader, serverURL)

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

// runLegacy runs the application in legacy mode (local storage with optional web server)
func runLegacy(dataDir, cacheDir, webAddr, localIP, port string, withWeb bool, syncTo string) {
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

	var apiClient *client.Client
	if syncTo != "" {
		// Ensure URL has scheme
		if !strings.HasPrefix(syncTo, "http://") && !strings.HasPrefix(syncTo, "https://") {
			syncTo = "http://" + syncTo
		}

		fmt.Printf("Connecting to %s for sync...\n", syncTo)

		// Create API client
		apiClient = client.NewClient(syncTo)

		// Test connection
		if err := apiClient.Connect(); err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to sync server: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Connected for sync!\n")
	}

	// Create the UI model
	model := ui.NewModel(store, recipeParser, downloader)

	// If syncing to server, set remote client and subscribe to events
	if apiClient != nil {
		model.SetRemoteClient(apiClient)

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

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}

		// Cleanup
		apiClient.Unsubscribe(eventChan)
	} else {
		// Run without remote sync
		p := tea.NewProgram(&model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}
	}
}

// getLocalIP returns the local network IP address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return ""
}
