package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cookbook-go/recipe-tracker/internal/api"
	"github.com/cookbook-go/recipe-tracker/internal/parser"
	"github.com/cookbook-go/recipe-tracker/internal/storage"
)

func main() {
	// Command line flags
	port := flag.String("port", "9005", "Port for server")
	dataDir := flag.String("data", "", "Data directory (default: ~/.recipe-tracker)")
	flag.Parse()

	// Default data directory
	if *dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		*dataDir = filepath.Join(homeDir, ".recipe-tracker")
	}

	// Get local IP for display
	localIP := getLocalIP()
	webAddr := ":" + *port

	// Initialize storage
	store, err := storage.NewStore(*dataDir)
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
	fmt.Printf("║  Local:   http://localhost:%-12s║\n", *port)
	if localIP != "" {
		fmt.Printf("║  Network: http://%-15s:%-4s ║\n", localIP, *port)
	}
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Println("║  Recipes saved to disk at:             ║")
	fmt.Printf("║    %s\n", filepath.Join(*dataDir, "recipes.json"))
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
