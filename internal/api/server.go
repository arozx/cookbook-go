package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cookbook-go/recipe-tracker/internal/models"
	"github.com/cookbook-go/recipe-tracker/internal/parser"
	"github.com/cookbook-go/recipe-tracker/internal/storage"
)

// Server provides a REST API for the recipe tracker
type Server struct {
	store      *storage.Store
	parser     *parser.Parser
	httpServer *http.Server
	clients    map[chan SyncEvent]bool
	clientsMu  sync.RWMutex
}

// SyncEvent represents a change event for real-time sync
type SyncEvent struct {
	Type      string         `json:"type"` // "add", "update", "delete"
	Recipe    *models.Recipe `json:"recipe,omitempty"`
	RecipeID  string         `json:"recipe_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// NewServer creates a new API server
func NewServer(store *storage.Store, parser *parser.Parser) *Server {
	return &Server{
		store:   store,
		parser:  parser,
		clients: make(map[chan SyncEvent]bool),
	}
}

// Start begins serving the API on the specified address
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/recipes", s.handleRecipes)
	mux.HandleFunc("/api/recipes/", s.handleRecipe)
	mux.HandleFunc("/api/parse", s.handleParse)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/events", s.handleSSE)

	// Serve static webapp
	mux.HandleFunc("/", s.handleWebApp)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Address returns the server's address
func (s *Server) Address() string {
	if s.httpServer != nil {
		return s.httpServer.Addr
	}
	return ""
}

// corsMiddleware adds CORS headers for mobile webapp
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleRecipes handles GET /api/recipes and POST /api/recipes
func (s *Server) handleRecipes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getRecipes(w, r)
	case http.MethodPost:
		s.createRecipe(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRecipe handles /api/recipes/{id}
func (s *Server) handleRecipe(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/recipes/")
	id := strings.TrimSuffix(path, "/")

	// Check for /refresh suffix
	if strings.HasSuffix(id, "/refresh") {
		id = strings.TrimSuffix(id, "/refresh")
		if r.Method == http.MethodPost {
			s.refreshRecipe(w, r, id)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if id == "" {
		http.Error(w, "Recipe ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getRecipe(w, r, id)
	case http.MethodPut:
		s.updateRecipe(w, r, id)
	case http.MethodDelete:
		s.deleteRecipe(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getRecipes returns all recipes
func (s *Server) getRecipes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var recipes []models.Recipe
	if query != "" {
		recipes = s.store.SearchRecipes(query)
	} else {
		recipes = s.store.GetAllRecipes()
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"recipes": recipes,
		"count":   len(recipes),
	})
}

// getRecipe returns a single recipe
func (s *Server) getRecipe(w http.ResponseWriter, r *http.Request, id string) {
	recipe, err := s.store.GetRecipe(id)
	if err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, http.StatusOK, recipe)
}

// createRecipe adds a new recipe
func (s *Server) createRecipe(w http.ResponseWriter, r *http.Request) {
	var recipe models.Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Set timestamps
	now := time.Now()
	recipe.CreatedAt = now
	recipe.UpdatedAt = now

	if err := s.store.AddRecipe(recipe); err != nil {
		http.Error(w, "Failed to add recipe: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify connected clients
	s.broadcast(SyncEvent{
		Type:      "add",
		Recipe:    &recipe,
		Timestamp: now,
	})

	s.jsonResponse(w, http.StatusCreated, recipe)
}

// updateRecipe updates an existing recipe
func (s *Server) updateRecipe(w http.ResponseWriter, r *http.Request, id string) {
	var recipe models.Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	recipe.ID = id
	recipe.UpdatedAt = time.Now()

	if err := s.store.AddRecipe(recipe); err != nil {
		http.Error(w, "Failed to update recipe: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify connected clients
	s.broadcast(SyncEvent{
		Type:      "update",
		Recipe:    &recipe,
		Timestamp: recipe.UpdatedAt,
	})

	s.jsonResponse(w, http.StatusOK, recipe)
}

// deleteRecipe removes a recipe
func (s *Server) deleteRecipe(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.store.DeleteRecipe(id); err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	// Notify connected clients
	s.broadcast(SyncEvent{
		Type:      "delete",
		RecipeID:  id,
		Timestamp: time.Now(),
	})

	w.WriteHeader(http.StatusNoContent)
}

// refreshRecipe re-fetches recipe data from the source URL while preserving user data
func (s *Server) refreshRecipe(w http.ResponseWriter, r *http.Request, id string) {
	// Get existing recipe
	existing, err := s.store.GetRecipe(id)
	if err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	// Check if recipe has a URL to refresh from
	if existing.URL == "" {
		http.Error(w, "Recipe has no source URL to refresh from", http.StatusBadRequest)
		return
	}

	// Fetch fresh data from the URL
	refreshed, err := s.parser.FetchAndParse(existing.URL)
	if err != nil {
		http.Error(w, "Failed to refresh recipe: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Preserve user data from existing recipe
	refreshed.ID = existing.ID
	refreshed.Notes = existing.Notes
	refreshed.CreatedAt = existing.CreatedAt
	refreshed.UpdatedAt = time.Now()

	// Preserve local image paths if they exist and no new images were found
	if len(refreshed.ImagePaths) == 0 && len(existing.ImagePaths) > 0 {
		refreshed.ImagePaths = existing.ImagePaths
	}

	// Save the refreshed recipe
	if err := s.store.AddRecipe(*refreshed); err != nil {
		http.Error(w, "Failed to save refreshed recipe: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify connected clients
	s.broadcast(SyncEvent{
		Type:      "update",
		Recipe:    refreshed,
		Timestamp: refreshed.UpdatedAt,
	})

	s.jsonResponse(w, http.StatusOK, refreshed)
}

// handleParse parses a recipe URL
func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	recipe, err := s.parser.FetchAndParse(req.URL)
	if err != nil {
		http.Error(w, "Failed to parse recipe: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, http.StatusOK, recipe)
}

// handleSync handles bulk sync operations
func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return all recipes with their timestamps for sync
		recipes := s.store.GetAllRecipes()
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"recipes":    recipes,
			"count":      len(recipes),
			"synced_at":  time.Now(),
			"server_ver": "1.0.0",
		})

	case http.MethodPost:
		// Accept bulk updates from client
		var syncReq struct {
			Recipes []models.Recipe `json:"recipes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&syncReq); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		var errors []string
		for _, recipe := range syncReq.Recipes {
			if err := s.store.AddRecipe(recipe); err != nil {
				errors = append(errors, fmt.Sprintf("recipe %s: %v", recipe.ID, err))
			}
		}

		if len(errors) > 0 {
			s.jsonResponse(w, http.StatusPartialContent, map[string]interface{}{
				"synced":  len(syncReq.Recipes) - len(errors),
				"errors":  errors,
				"message": "Some recipes failed to sync",
			})
			return
		}

		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"synced":  len(syncReq.Recipes),
			"message": "Sync completed successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSSE handles Server-Sent Events for real-time sync
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create client channel
	clientChan := make(chan SyncEvent, 10)
	s.addClient(clientChan)
	defer s.removeClient(clientChan)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\": \"connected\"}\n\n")
	w.(http.Flusher).Flush()

	// Listen for events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-clientChan:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			w.(http.Flusher).Flush()
		}
	}
}

// addClient registers a new SSE client
func (s *Server) addClient(ch chan SyncEvent) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	s.clients[ch] = true
}

// removeClient unregisters an SSE client
func (s *Server) removeClient(ch chan SyncEvent) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	delete(s.clients, ch)
	close(ch)
}

// broadcast sends an event to all connected clients
func (s *Server) broadcast(event SyncEvent) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for ch := range s.clients {
		select {
		case ch <- event:
		default:
			// Client channel full, skip
		}
	}
}

// NotifyChange allows external code (like the TUI) to notify the API of changes
func (s *Server) NotifyChange(eventType string, recipe *models.Recipe, recipeID string) {
	s.broadcast(SyncEvent{
		Type:      eventType,
		Recipe:    recipe,
		RecipeID:  recipeID,
		Timestamp: time.Now(),
	})
}

// jsonResponse sends a JSON response
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// handleWebApp serves the mobile web application
func (s *Server) handleWebApp(w http.ResponseWriter, r *http.Request) {
	// Serve the embedded webapp
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(webAppHTML))
}
