package parser

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cookbook-go/recipe-tracker/internal/models"
)

// Parser handles extracting recipe data from web pages
type Parser struct {
	client *http.Client
}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchAndParse fetches a URL and extracts recipe data
func (p *Parser) FetchAndParse(recipeURL string) (*models.Recipe, error) {
	req, err := http.NewRequest("GET", recipeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	recipe := &models.Recipe{
		ID:        generateID(recipeURL),
		URL:       recipeURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Try to extract JSON-LD structured data first (most reliable)
	if p.extractJSONLD(doc, recipe) {
		// Supplement with additional image extraction if needed
		if len(recipe.ImageURLs) == 0 {
			recipe.ImageURLs = p.extractImages(doc, recipeURL)
		}
		return recipe, nil
	}

	// Fallback to HTML parsing
	recipe.Title = p.extractTitle(doc)
	recipe.Description = p.extractDescription(doc)
	recipe.ImageURLs = p.extractImages(doc, recipeURL)
	recipe.Ingredients = p.extractIngredients(doc)
	recipe.Instructions = p.extractInstructions(doc)
	recipe.PrepTime = p.extractTime(doc, "prep")
	recipe.CookTime = p.extractTime(doc, "cook")
	recipe.Servings = p.extractServings(doc)

	return recipe, nil
}

// extractJSONLD attempts to parse JSON-LD structured recipe data
func (p *Parser) extractJSONLD(doc *goquery.Document, recipe *models.Recipe) bool {
	found := false

	doc.Find(`script[type="application/ld+json"]`).Each(func(i int, s *goquery.Selection) {
		if found {
			return
		}

		text := s.Text()

		// Try to parse as a single recipe
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(text), &data); err != nil {
			// Try as array
			var arr []map[string]interface{}
			if err := json.Unmarshal([]byte(text), &arr); err == nil {
				for _, item := range arr {
					if p.parseRecipeJSON(item, recipe) {
						found = true
						return
					}
				}
			}
			return
		}

		// Check if it's a graph
		if graph, ok := data["@graph"].([]interface{}); ok {
			for _, item := range graph {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if p.parseRecipeJSON(itemMap, recipe) {
						found = true
						return
					}
				}
			}
		} else if p.parseRecipeJSON(data, recipe) {
			found = true
		}
	})

	return found
}

// parseRecipeJSON parses a JSON object that should be a Recipe schema
func (p *Parser) parseRecipeJSON(data map[string]interface{}, recipe *models.Recipe) bool {
	schemaType, _ := data["@type"].(string)
	if schemaType != "Recipe" {
		// Check for array type
		if types, ok := data["@type"].([]interface{}); ok {
			isRecipe := false
			for _, t := range types {
				if ts, ok := t.(string); ok && ts == "Recipe" {
					isRecipe = true
					break
				}
			}
			if !isRecipe {
				return false
			}
		} else {
			return false
		}
	}

	if name, ok := data["name"].(string); ok {
		recipe.Title = name
	}

	if desc, ok := data["description"].(string); ok {
		recipe.Description = cleanText(desc)
	}

	// Extract images
	if img, ok := data["image"].(string); ok {
		recipe.ImageURLs = append(recipe.ImageURLs, img)
	} else if imgs, ok := data["image"].([]interface{}); ok {
		for _, img := range imgs {
			if imgStr, ok := img.(string); ok {
				recipe.ImageURLs = append(recipe.ImageURLs, imgStr)
			} else if imgMap, ok := img.(map[string]interface{}); ok {
				if url, ok := imgMap["url"].(string); ok {
					recipe.ImageURLs = append(recipe.ImageURLs, url)
				}
			}
		}
	} else if imgMap, ok := data["image"].(map[string]interface{}); ok {
		if url, ok := imgMap["url"].(string); ok {
			recipe.ImageURLs = append(recipe.ImageURLs, url)
		}
	}

	// Extract ingredients
	if ingredients, ok := data["recipeIngredient"].([]interface{}); ok {
		for _, ing := range ingredients {
			if ingStr, ok := ing.(string); ok {
				recipe.Ingredients = append(recipe.Ingredients, parseIngredient(ingStr))
			}
		}
	}

	// Extract instructions
	if instructions, ok := data["recipeInstructions"].([]interface{}); ok {
		for _, inst := range instructions {
			switch v := inst.(type) {
			case string:
				recipe.Instructions = append(recipe.Instructions, cleanText(v))
			case map[string]interface{}:
				if text, ok := v["text"].(string); ok {
					recipe.Instructions = append(recipe.Instructions, cleanText(text))
				}
			}
		}
	} else if instStr, ok := data["recipeInstructions"].(string); ok {
		// Split by newlines or periods for single string instructions
		for _, line := range strings.Split(instStr, "\n") {
			line = cleanText(line)
			if line != "" {
				recipe.Instructions = append(recipe.Instructions, line)
			}
		}
	}

	// Extract times
	if prepTime, ok := data["prepTime"].(string); ok {
		recipe.PrepTime = parseDuration(prepTime)
	}
	if cookTime, ok := data["cookTime"].(string); ok {
		recipe.CookTime = parseDuration(cookTime)
	}

	// Extract servings
	if servings, ok := data["recipeYield"].(string); ok {
		recipe.Servings = servings
	} else if servings, ok := data["recipeYield"].([]interface{}); ok && len(servings) > 0 {
		if s, ok := servings[0].(string); ok {
			recipe.Servings = s
		}
	}

	return true
}

// extractTitle gets the recipe title from HTML
func (p *Parser) extractTitle(doc *goquery.Document) string {
	// Try common recipe title selectors
	selectors := []string{
		"h1.recipe-title",
		"h1.entry-title",
		"h1[itemprop='name']",
		".recipe-header h1",
		"h1",
	}

	for _, sel := range selectors {
		if title := doc.Find(sel).First().Text(); title != "" {
			return cleanText(title)
		}
	}

	// Fallback to page title
	return cleanText(doc.Find("title").Text())
}

// extractDescription gets the recipe description
func (p *Parser) extractDescription(doc *goquery.Document) string {
	selectors := []string{
		"[itemprop='description']",
		".recipe-summary",
		".recipe-description",
		"meta[name='description']",
	}

	for _, sel := range selectors {
		if sel == "meta[name='description']" {
			if content, exists := doc.Find(sel).Attr("content"); exists {
				return cleanText(content)
			}
		} else if desc := doc.Find(sel).First().Text(); desc != "" {
			return cleanText(desc)
		}
	}

	return ""
}

// extractImages extracts recipe images from HTML
func (p *Parser) extractImages(doc *goquery.Document, baseURL string) []string {
	var images []string
	seen := make(map[string]bool)

	base, _ := url.Parse(baseURL)

	// Common image selectors for recipes
	selectors := []string{
		"[itemprop='image']",
		".recipe-image img",
		".recipe-photo img",
		"article img",
		".post-content img",
		".entry-content img",
	}

	for _, sel := range selectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			var imgURL string

			// Check various attributes
			for _, attr := range []string{"src", "data-src", "data-lazy-src", "srcset"} {
				if val, exists := s.Attr(attr); exists && val != "" {
					if attr == "srcset" {
						// Get first URL from srcset
						parts := strings.Split(val, ",")
						if len(parts) > 0 {
							val = strings.Fields(parts[0])[0]
						}
					}
					imgURL = val
					break
				}
			}

			if imgURL == "" {
				return
			}

			// Make absolute URL
			imgURL = resolveURL(base, imgURL)

			// Filter out small images (likely icons)
			if strings.Contains(imgURL, "icon") || strings.Contains(imgURL, "logo") {
				return
			}

			if !seen[imgURL] {
				seen[imgURL] = true
				images = append(images, imgURL)
			}
		})

		if len(images) > 0 {
			break
		}
	}

	return images
}

// extractIngredients extracts ingredients from HTML
func (p *Parser) extractIngredients(doc *goquery.Document) []models.Ingredient {
	var ingredients []models.Ingredient

	selectors := []string{
		"[itemprop='recipeIngredient']",
		"[itemprop='ingredients']",
		".recipe-ingredients li",
		".ingredients li",
		".ingredient-list li",
		".wprm-recipe-ingredient",
	}

	for _, sel := range selectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			text := cleanText(s.Text())
			if text != "" {
				ingredients = append(ingredients, parseIngredient(text))
			}
		})

		if len(ingredients) > 0 {
			break
		}
	}

	return ingredients
}

// extractInstructions extracts cooking instructions from HTML
func (p *Parser) extractInstructions(doc *goquery.Document) []string {
	var instructions []string

	selectors := []string{
		"[itemprop='recipeInstructions'] li",
		"[itemprop='recipeInstructions'] p",
		".recipe-instructions li",
		".instructions li",
		".recipe-directions li",
		".wprm-recipe-instruction",
	}

	for _, sel := range selectors {
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			text := cleanText(s.Text())
			if text != "" {
				instructions = append(instructions, text)
			}
		})

		if len(instructions) > 0 {
			break
		}
	}

	return instructions
}

// extractTime extracts prep or cook time
func (p *Parser) extractTime(doc *goquery.Document, timeType string) string {
	var selectors []string
	if timeType == "prep" {
		selectors = []string{
			"[itemprop='prepTime']",
			".prep-time",
			".recipe-prep-time",
		}
	} else {
		selectors = []string{
			"[itemprop='cookTime']",
			".cook-time",
			".recipe-cook-time",
		}
	}

	for _, sel := range selectors {
		el := doc.Find(sel).First()
		if content, exists := el.Attr("content"); exists {
			return parseDuration(content)
		}
		if text := cleanText(el.Text()); text != "" {
			return text
		}
	}

	return ""
}

// extractServings extracts serving information
func (p *Parser) extractServings(doc *goquery.Document) string {
	selectors := []string{
		"[itemprop='recipeYield']",
		".recipe-yield",
		".servings",
	}

	for _, sel := range selectors {
		if text := cleanText(doc.Find(sel).First().Text()); text != "" {
			return text
		}
	}

	return ""
}

// parseIngredient parses an ingredient string into structured data
func parseIngredient(text string) models.Ingredient {
	text = cleanText(text)

	ing := models.Ingredient{
		Original: text,
	}

	// Common patterns for amounts and units
	amountPattern := regexp.MustCompile(`^([\d½¼¾⅓⅔⅛⅜⅝⅞\s\/\-\.]+)`)
	unitPattern := regexp.MustCompile(`(?i)\b(cups?|tbsp|tsp|tablespoons?|teaspoons?|oz|ounces?|lbs?|pounds?|grams?|g|kg|ml|liters?|l|quarts?|pints?|gallons?|pinch|dash|cloves?|slices?|pieces?|cans?|packages?|bunche?s?)\b`)

	// Extract amount
	if matches := amountPattern.FindStringSubmatch(text); len(matches) > 1 {
		ing.Amount = strings.TrimSpace(matches[1])
		text = strings.TrimPrefix(text, matches[0])
	}

	// Extract unit
	if matches := unitPattern.FindStringSubmatch(text); len(matches) > 1 {
		ing.Unit = strings.TrimSpace(matches[1])
		text = unitPattern.ReplaceAllString(text, "")
	}

	// Remaining text is the ingredient name
	ing.Name = cleanText(text)
	if ing.Name == "" {
		ing.Name = ing.Original
	}

	return ing
}

// parseDuration converts ISO 8601 duration to readable format
func parseDuration(iso string) string {
	// Handle ISO 8601 duration format (PT1H30M)
	if strings.HasPrefix(iso, "PT") || strings.HasPrefix(iso, "P") {
		iso = strings.TrimPrefix(iso, "P")
		iso = strings.TrimPrefix(iso, "T")

		var parts []string

		// Extract hours
		if idx := strings.Index(iso, "H"); idx != -1 {
			hours := iso[:idx]
			parts = append(parts, hours+" hr")
			iso = iso[idx+1:]
		}

		// Extract minutes
		if idx := strings.Index(iso, "M"); idx != -1 {
			mins := iso[:idx]
			parts = append(parts, mins+" min")
		}

		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}

	return iso
}

// cleanText removes extra whitespace and trims text
func cleanText(text string) string {
	// Remove HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")

	// Collapse whitespace
	space := regexp.MustCompile(`\s+`)
	text = space.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// resolveURL makes a relative URL absolute
func resolveURL(base *url.URL, ref string) string {
	if base == nil {
		return ref
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}

	return base.ResolveReference(refURL).String()
}

// generateID creates a unique ID for a recipe based on URL
func generateID(url string) string {
	hash := md5.Sum([]byte(url))
	return fmt.Sprintf("%x", hash)[:12]
}

// FetchHTML fetches raw HTML for debugging
func (p *Parser) FetchHTML(recipeURL string) (string, error) {
	req, err := http.NewRequest("GET", recipeURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
