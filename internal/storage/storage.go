package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cookbook-go/recipe-tracker/internal/models"
)

// Store handles recipe persistence
type Store struct {
	mu       sync.RWMutex
	filepath string
	data     models.RecipeStore
}

// NewStore creates a new storage instance
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	filepath := filepath.Join(dataDir, "recipes.json")
	s := &Store{
		filepath: filepath,
		data:     models.RecipeStore{},
	}

	// Load existing data if available
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading data: %w", err)
	}

	return s, nil
}

// load reads recipes from disk
func (s *Store) load() error {
	data, err := os.ReadFile(s.filepath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.data)
}

// save writes recipes to disk
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling data: %w", err)
	}

	return os.WriteFile(s.filepath, data, 0644)
}

// AddRecipe adds a new recipe to the store
func (s *Store) AddRecipe(recipe models.Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate
	for i, r := range s.data.Recipes {
		if r.ID == recipe.ID || r.URL == recipe.URL {
			// Update existing recipe
			s.data.Recipes[i] = recipe
			return s.save()
		}
	}

	s.data.Recipes = append(s.data.Recipes, recipe)
	return s.save()
}

// GetRecipe retrieves a recipe by ID
func (s *Store) GetRecipe(id string) (*models.Recipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.data.Recipes {
		if r.ID == id {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("recipe not found: %s", id)
}

// GetAllRecipes returns all stored recipes
func (s *Store) GetAllRecipes() []models.Recipe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recipes := make([]models.Recipe, len(s.data.Recipes))
	copy(recipes, s.data.Recipes)
	return recipes
}

// DeleteRecipe removes a recipe by ID
func (s *Store) DeleteRecipe(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.data.Recipes {
		if r.ID == id {
			s.data.Recipes = append(s.data.Recipes[:i], s.data.Recipes[i+1:]...)
			return s.save()
		}
	}

	return fmt.Errorf("recipe not found: %s", id)
}

// SearchRecipes searches recipes by title
func (s *Store) SearchRecipes(query string) []models.Recipe {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []models.Recipe
	for _, r := range s.data.Recipes {
		// Simple case-insensitive contains search
		if containsIgnoreCase(r.Title, query) {
			results = append(results, r)
		}
	}

	return results
}

// containsIgnoreCase performs case-insensitive substring search
func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))

	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			sLower[i] = s[i] + 32
		} else {
			sLower[i] = s[i]
		}
	}

	for i := 0; i < len(substr); i++ {
		if substr[i] >= 'A' && substr[i] <= 'Z' {
			substrLower[i] = substr[i] + 32
		} else {
			substrLower[i] = substr[i]
		}
	}

	return contains(string(sLower), string(substrLower))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Count returns the number of stored recipes
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.Recipes)
}
