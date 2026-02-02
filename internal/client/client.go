package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cookbook-go/recipe-tracker/internal/models"
)

// Client provides a remote API client that implements the same interface as storage.Store
type Client struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex
	cache      []models.Recipe
	listeners  []chan RecipeEvent
	listenerMu sync.RWMutex
	connected  bool
}

// RecipeEvent represents a real-time sync event
type RecipeEvent struct {
	Type     string         `json:"type"` // "add", "update", "delete"
	Recipe   *models.Recipe `json:"recipe,omitempty"`
	RecipeID string         `json:"recipe_id,omitempty"`
}

// NewClient creates a new API client
func NewClient(serverURL string) *Client {
	// Ensure URL doesn't have trailing slash
	serverURL = strings.TrimSuffix(serverURL, "/")

	return &Client{
		baseURL: serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:     []models.Recipe{},
		listeners: []chan RecipeEvent{},
	}
}

// Connect tests the connection and starts SSE listener
func (c *Client) Connect() error {
	// Test connection by fetching recipes
	if err := c.refresh(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	c.connected = true

	// Start SSE listener for real-time updates
	go c.listenSSE()

	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.connected
}

// ServerURL returns the server URL
func (c *Client) ServerURL() string {
	return c.baseURL
}

// Subscribe returns a channel that receives recipe events
func (c *Client) Subscribe() chan RecipeEvent {
	c.listenerMu.Lock()
	defer c.listenerMu.Unlock()

	ch := make(chan RecipeEvent, 10)
	c.listeners = append(c.listeners, ch)
	return ch
}

// Unsubscribe removes a listener channel
func (c *Client) Unsubscribe(ch chan RecipeEvent) {
	c.listenerMu.Lock()
	defer c.listenerMu.Unlock()

	for i, listener := range c.listeners {
		if listener == ch {
			c.listeners = append(c.listeners[:i], c.listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

// broadcast sends an event to all listeners
func (c *Client) broadcast(event RecipeEvent) {
	c.listenerMu.RLock()
	defer c.listenerMu.RUnlock()

	for _, ch := range c.listeners {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// listenSSE connects to the server's SSE endpoint for real-time updates
func (c *Client) listenSSE() {
	for {
		if err := c.connectSSE(); err != nil {
			// Reconnect after delay
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (c *Client) connectSSE() error {
	resp, err := c.httpClient.Get(c.baseURL + "/api/events")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	var eventType string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			if eventType == "connected" {
				continue
			}

			var event RecipeEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			event.Type = eventType

			// Update local cache
			c.handleEvent(event)

			// Broadcast to listeners
			c.broadcast(event)
		}
	}
}

// handleEvent updates the local cache based on the event
func (c *Client) handleEvent(event RecipeEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch event.Type {
	case "add":
		if event.Recipe != nil {
			// Check if already exists
			for i, r := range c.cache {
				if r.ID == event.Recipe.ID {
					c.cache[i] = *event.Recipe
					return
				}
			}
			c.cache = append([]models.Recipe{*event.Recipe}, c.cache...)
		}

	case "update":
		if event.Recipe != nil {
			for i, r := range c.cache {
				if r.ID == event.Recipe.ID {
					c.cache[i] = *event.Recipe
					return
				}
			}
		}

	case "delete":
		for i, r := range c.cache {
			if r.ID == event.RecipeID {
				c.cache = append(c.cache[:i], c.cache[i+1:]...)
				return
			}
		}
	}
}

// refresh fetches all recipes from the server
func (c *Client) refresh() error {
	resp, err := c.httpClient.Get(c.baseURL + "/api/recipes")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var result struct {
		Recipes []models.Recipe `json:"recipes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.mu.Lock()
	c.cache = result.Recipes
	c.mu.Unlock()

	return nil
}

// AddRecipe adds a new recipe via the API
func (c *Client) AddRecipe(recipe models.Recipe) error {
	data, err := json.Marshal(recipe)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/recipes",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add recipe: %s", string(body))
	}

	// Decode the saved recipe
	var savedRecipe models.Recipe
	if err := json.NewDecoder(resp.Body).Decode(&savedRecipe); err != nil {
		return err
	}

	// Update local cache
	c.mu.Lock()
	c.cache = append([]models.Recipe{savedRecipe}, c.cache...)
	c.mu.Unlock()

	return nil
}

// GetRecipe retrieves a recipe by ID
func (c *Client) GetRecipe(id string) (*models.Recipe, error) {
	// Check cache first
	c.mu.RLock()
	for _, r := range c.cache {
		if r.ID == id {
			c.mu.RUnlock()
			return &r, nil
		}
	}
	c.mu.RUnlock()

	// Fetch from server
	resp, err := c.httpClient.Get(c.baseURL + "/api/recipes/" + id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("recipe not found: %s", id)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var recipe models.Recipe
	if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
		return nil, err
	}

	return &recipe, nil
}

// GetAllRecipes returns all recipes (from cache)
func (c *Client) GetAllRecipes() []models.Recipe {
	c.mu.RLock()
	defer c.mu.RUnlock()

	recipes := make([]models.Recipe, len(c.cache))
	copy(recipes, c.cache)
	return recipes
}

// DeleteRecipe removes a recipe by ID
func (c *Client) DeleteRecipe(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/recipes/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("recipe not found: %s", id)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete recipe: status %d", resp.StatusCode)
	}

	// Update local cache
	c.mu.Lock()
	for i, r := range c.cache {
		if r.ID == id {
			c.cache = append(c.cache[:i], c.cache[i+1:]...)
			break
		}
	}
	c.mu.Unlock()

	return nil
}

// SearchRecipes searches recipes by title
func (c *Client) SearchRecipes(query string) []models.Recipe {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []models.Recipe
	queryLower := strings.ToLower(query)

	for _, r := range c.cache {
		if strings.Contains(strings.ToLower(r.Title), queryLower) {
			results = append(results, r)
		}
	}

	return results
}

// Count returns the number of cached recipes
func (c *Client) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Refresh forces a refresh from the server
func (c *Client) Refresh() error {
	return c.refresh()
}

// ParseRecipeURL parses a recipe from a URL via the server
func (c *Client) ParseRecipeURL(url string) (*models.Recipe, error) {
	data, err := json.Marshal(map[string]string{"url": url})
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/parse",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to parse recipe: %s", string(body))
	}

	var recipe models.Recipe
	if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
		return nil, err
	}

	return &recipe, nil
}
