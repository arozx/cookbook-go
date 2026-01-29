package models

import "time"

// Ingredient represents a single ingredient in a recipe
type Ingredient struct {
	Name     string `json:"name"`
	Amount   string `json:"amount"`
	Unit     string `json:"unit"`
	Original string `json:"original"` // Original text from recipe
}

// Recipe represents a complete recipe
type Recipe struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	URL          string       `json:"url"`
	ImageURLs    []string     `json:"image_urls"`
	ImagePaths   []string     `json:"image_paths"` // Local paths to downloaded images
	Ingredients  []Ingredient `json:"ingredients"`
	Instructions []string     `json:"instructions"`
	PrepTime     string       `json:"prep_time"`
	CookTime     string       `json:"cook_time"`
	Servings     string       `json:"servings"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// RecipeStore holds all recipes
type RecipeStore struct {
	Recipes []Recipe `json:"recipes"`
}
