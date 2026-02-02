package models

// RecipeRepository defines the interface for recipe storage
// Both local storage and remote client implement this interface
type RecipeRepository interface {
	AddRecipe(recipe Recipe) error
	GetRecipe(id string) (*Recipe, error)
	GetAllRecipes() []Recipe
	DeleteRecipe(id string) error
	SearchRecipes(query string) []Recipe
	Count() int
}
