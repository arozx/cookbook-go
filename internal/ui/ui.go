package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cookbook-go/recipe-tracker/internal/images"
	"github.com/cookbook-go/recipe-tracker/internal/models"
	"github.com/cookbook-go/recipe-tracker/internal/parser"
	"github.com/cookbook-go/recipe-tracker/internal/storage"
)

// View represents different screens in the app
type View int

const (
	ViewList View = iota
	ViewAdd
	ViewRecipe
	ViewLoading
	ViewSearch
)

// Model represents the application state
type Model struct {
	// Core dependencies
	store      *storage.Store
	parser     *parser.Parser
	downloader *images.Downloader

	// UI state
	view         View
	previousView View
	width        int
	height       int
	ready        bool

	// Recipe list
	recipes       []models.Recipe
	selectedIndex int
	listOffset    int

	// Add recipe form
	urlInput   textinput.Model
	addError   string
	addSuccess string

	// Recipe detail view
	currentRecipe *models.Recipe
	viewport      viewport.Model
	recipeImage   string
	showImage     bool
	imageTab      int // 0: ingredients, 1: instructions, 2: image

	// Loading state
	spinner    spinner.Model
	loadingMsg string

	// Search state
	searchInput   textinput.Model
	searchResults []models.Recipe
	allRecipes    []models.Recipe
	isSearching   bool
}

// Messages
type recipeLoadedMsg struct {
	recipe *models.Recipe
	err    error
}

type imageLoadedMsg struct {
	image string
	err   error
}

type recipesRefreshedMsg struct {
	recipes []models.Recipe
}

// NewModel creates a new application model
func NewModel(store *storage.Store, parser *parser.Parser, downloader *images.Downloader) Model {
	// URL input
	ti := textinput.New()
	ti.Placeholder = "https://example.com/recipe"
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	// Search input
	si := textinput.New()
	si.Placeholder = "Search recipes..."
	si.CharLimit = 100
	si.Width = 40

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	// Viewport
	vp := viewport.New(80, 20)

	allRecipes := store.GetAllRecipes()

	return Model{
		store:       store,
		parser:      parser,
		downloader:  downloader,
		view:        ViewList,
		recipes:     allRecipes,
		allRecipes:  allRecipes,
		urlInput:    ti,
		searchInput: si,
		spinner:     sp,
		viewport:    vp,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		m.ready = true

	case recipeLoadedMsg:
		if msg.err != nil {
			m.addError = fmt.Sprintf("Error: %v", msg.err)
			m.view = ViewAdd
			return m, nil
		}

		// Download images
		if len(msg.recipe.ImageURLs) > 0 {
			msg.recipe.ImagePaths = m.downloader.DownloadAll(msg.recipe.ImageURLs)
		}

		// Save recipe
		if err := m.store.AddRecipe(*msg.recipe); err != nil {
			m.addError = fmt.Sprintf("Error saving: %v", err)
			m.view = ViewAdd
			return m, nil
		}

		m.recipes = m.store.GetAllRecipes()
		m.allRecipes = m.recipes
		m.currentRecipe = msg.recipe
		m.addSuccess = "Recipe added successfully!"
		m.urlInput.SetValue("")
		m.addError = ""
		m.view = ViewRecipe
		m.imageTab = 0

		// Load image for display
		if len(msg.recipe.ImagePaths) > 0 {
			return m, m.loadImage(msg.recipe.ImagePaths[0])
		}
		return m, nil

	case imageLoadedMsg:
		if msg.err == nil {
			m.recipeImage = msg.image
		}
		return m, nil

	case recipesRefreshedMsg:
		m.recipes = msg.recipes
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update components based on view
	switch m.view {
	case ViewAdd:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)

	case ViewSearch:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)

	case ViewRecipe:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		if m.view == ViewList {
			return m, tea.Quit
		}
		// Go back to list from other views
		m.view = ViewList
		m.addError = ""
		m.addSuccess = ""
		m.recipeImage = ""
		return m, nil

	case "esc":
		if m.view != ViewList {
			m.view = ViewList
			m.addError = ""
			m.addSuccess = ""
			m.recipeImage = ""
			return m, nil
		}
	}

	// View-specific keys
	switch m.view {
	case ViewList:
		return m.handleListKeys(msg)
	case ViewAdd:
		return m.handleAddKeys(msg)
	case ViewRecipe:
		return m.handleRecipeKeys(msg)
	case ViewSearch:
		return m.handleSearchKeys(msg)
	}

	return m, nil
}

// handleListKeys handles keys in list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
			if m.selectedIndex < m.listOffset {
				m.listOffset = m.selectedIndex
			}
		}

	case "down", "j":
		if m.selectedIndex < len(m.recipes)-1 {
			m.selectedIndex++
			maxVisible := m.height - 12
			if maxVisible < 1 {
				maxVisible = 5
			}
			if m.selectedIndex >= m.listOffset+maxVisible {
				m.listOffset = m.selectedIndex - maxVisible + 1
			}
		}

	case "enter":
		if len(m.recipes) > 0 && m.selectedIndex < len(m.recipes) {
			recipe := m.recipes[m.selectedIndex]
			m.currentRecipe = &recipe
			m.view = ViewRecipe
			m.imageTab = 0
			m.recipeImage = ""
			m.viewport.SetContent(m.renderRecipeContent())
			m.viewport.GotoTop()

			// Load image
			if len(recipe.ImagePaths) > 0 {
				return m, m.loadImage(recipe.ImagePaths[0])
			}
		}

	case "a", "n":
		m.view = ViewAdd
		m.urlInput.Focus()
		m.addError = ""
		m.addSuccess = ""

	case "/":
		m.view = ViewSearch
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		m.isSearching = true

	case "d", "delete":
		if len(m.recipes) > 0 && m.selectedIndex < len(m.recipes) {
			recipe := m.recipes[m.selectedIndex]
			m.store.DeleteRecipe(recipe.ID)
			m.recipes = m.store.GetAllRecipes()
			m.allRecipes = m.recipes
			if m.selectedIndex >= len(m.recipes) && m.selectedIndex > 0 {
				m.selectedIndex--
			}
		}

	case "r":
		m.recipes = m.store.GetAllRecipes()
		m.allRecipes = m.recipes
	}

	return m, nil
}

// handleAddKeys handles keys in add view
func (m Model) handleAddKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			m.addError = "Please enter a URL"
			return m, nil
		}

		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}

		m.loadingMsg = "Fetching recipe..."
		m.view = ViewLoading
		return m, m.fetchRecipe(url)

	case "esc":
		m.view = ViewList
		m.addError = ""
		m.urlInput.SetValue("")
	}

	// Update text input
	var cmd tea.Cmd
	m.urlInput, cmd = m.urlInput.Update(msg)
	return m, cmd
}

// handleRecipeKeys handles keys in recipe view
func (m Model) handleRecipeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.imageTab = (m.imageTab + 1) % 3
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()

	case "shift+tab":
		m.imageTab = (m.imageTab + 2) % 3
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()

	case "i":
		m.imageTab = 0
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()

	case "s":
		m.imageTab = 1
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()

	case "p":
		m.imageTab = 2
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()
	}

	// Viewport navigation
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleSearchKeys handles keys in search view
func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(m.searchInput.Value())
		if query == "" {
			m.recipes = m.allRecipes
		} else {
			m.recipes = m.store.SearchRecipes(query)
		}
		m.selectedIndex = 0
		m.listOffset = 0
		m.view = ViewList
		m.isSearching = false
		return m, nil

	case "esc":
		m.view = ViewList
		m.recipes = m.allRecipes
		m.isSearching = false
		m.searchInput.SetValue("")
		return m, nil
	}

	// Update text input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// fetchRecipe fetches a recipe from URL
func (m Model) fetchRecipe(url string) tea.Cmd {
	return func() tea.Msg {
		recipe, err := m.parser.FetchAndParse(url)
		return recipeLoadedMsg{recipe: recipe, err: err}
	}
}

// loadImage loads an image for display
func (m Model) loadImage(path string) tea.Cmd {
	return func() tea.Msg {
		width := 60
		height := 30
		img, err := images.ImageToHalfBlock(path, width, height)
		return imageLoadedMsg{image: img, err: err}
	}
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	switch m.view {
	case ViewList:
		return m.renderList()
	case ViewAdd:
		return m.renderAdd()
	case ViewRecipe:
		return m.renderRecipe()
	case ViewLoading:
		return m.renderLoading()
	case ViewSearch:
		return m.renderSearch()
	default:
		return "Unknown view"
	}
}

// renderList renders the recipe list view
func (m Model) renderList() string {
	var b strings.Builder

	// Title
	title := TitleStyle.Render("Recipe Tracker")
	b.WriteString(title + "\n\n")

	// Recipe count and search info
	if m.isSearching || len(m.recipes) != len(m.allRecipes) {
		countText := fmt.Sprintf("%d of %d recipes (filtered)", len(m.recipes), len(m.allRecipes))
		b.WriteString(MetaStyle.Render(countText) + "\n\n")
	} else {
		countText := fmt.Sprintf("%d recipes", len(m.recipes))
		b.WriteString(MetaStyle.Render(countText) + "\n\n")
	}

	if len(m.recipes) == 0 {
		b.WriteString(HelpStyle.Render("No recipes yet. Press 'a' to add one!") + "\n")
	} else {
		maxVisible := m.height - 12
		if maxVisible < 1 {
			maxVisible = 5
		}

		endIndex := m.listOffset + maxVisible
		if endIndex > len(m.recipes) {
			endIndex = len(m.recipes)
		}

		for i := m.listOffset; i < endIndex; i++ {
			recipe := m.recipes[i]
			title := Truncate(recipe.Title, m.width-10)

			var line string
			if i == m.selectedIndex {
				line = SelectedItemStyle.Render("▸ " + title)
			} else {
				line = NormalItemStyle.Render("  " + title)
			}
			b.WriteString(line + "\n")
		}

		// Scroll indicator
		if len(m.recipes) > maxVisible {
			scrollInfo := fmt.Sprintf("\n  (%d-%d of %d)", m.listOffset+1, endIndex, len(m.recipes))
			b.WriteString(MetaStyle.Render(scrollInfo))
		}
	}

	// Help
	help := "\n\n" + HelpStyle.Render("j/k: navigate | enter: view | a: add | /: search | d: delete | q: quit")
	b.WriteString(help)

	return BaseStyle.Render(b.String())
}

// renderAdd renders the add recipe view
func (m Model) renderAdd() string {
	var b strings.Builder

	title := TitleStyle.Render("Add Recipe")
	b.WriteString(title + "\n\n")

	b.WriteString(InputLabelStyle.Render("Recipe URL:") + "\n")
	b.WriteString(InputStyle.Render(m.urlInput.View()) + "\n\n")

	if m.addError != "" {
		b.WriteString(ErrorStyle.Render(m.addError) + "\n\n")
	}

	if m.addSuccess != "" {
		b.WriteString(SuccessStyle.Render(m.addSuccess) + "\n\n")
	}

	help := HelpStyle.Render("enter: fetch recipe | esc: cancel")
	b.WriteString(help)

	return BaseStyle.Render(b.String())
}

// renderSearch renders the search view
func (m Model) renderSearch() string {
	var b strings.Builder

	title := TitleStyle.Render("Search Recipes")
	b.WriteString(title + "\n\n")

	b.WriteString(InputLabelStyle.Render("Search:") + "\n")
	b.WriteString(InputStyle.Render(m.searchInput.View()) + "\n\n")

	help := HelpStyle.Render("enter: search | esc: cancel")
	b.WriteString(help)

	return BaseStyle.Render(b.String())
}

// renderRecipe renders the recipe detail view
func (m Model) renderRecipe() string {
	if m.currentRecipe == nil {
		return BaseStyle.Render("No recipe selected")
	}

	var b strings.Builder

	// Title
	title := RecipeTitleStyle.Render(m.currentRecipe.Title)
	b.WriteString(title + "\n")

	// Meta info
	var meta []string
	if m.currentRecipe.PrepTime != "" {
		meta = append(meta, "Prep: "+m.currentRecipe.PrepTime)
	}
	if m.currentRecipe.CookTime != "" {
		meta = append(meta, "Cook: "+m.currentRecipe.CookTime)
	}
	if m.currentRecipe.Servings != "" {
		meta = append(meta, "Serves: "+m.currentRecipe.Servings)
	}
	if len(meta) > 0 {
		b.WriteString(MetaStyle.Render(strings.Join(meta, " • ")) + "\n")
	}

	// Tabs
	tabs := []string{"[i]ngredients", "[s]teps", "[p]hoto"}
	var tabLine strings.Builder
	for i, tab := range tabs {
		if i == m.imageTab {
			tabLine.WriteString(ActiveTabStyle.Render(tab))
		} else {
			tabLine.WriteString(InactiveTabStyle.Render(tab))
		}
	}
	b.WriteString("\n" + tabLine.String() + "\n\n")

	// Viewport
	b.WriteString(m.viewport.View() + "\n")

	// Help
	help := HelpStyle.Render("tab: switch tabs | j/k: scroll | esc: back to list")
	b.WriteString(help)

	return BaseStyle.Render(b.String())
}

// renderRecipeContent renders the content for the viewport based on selected tab
func (m Model) renderRecipeContent() string {
	if m.currentRecipe == nil {
		return ""
	}

	switch m.imageTab {
	case 0: // Ingredients
		return m.renderIngredients()
	case 1: // Instructions
		return m.renderInstructions()
	case 2: // Image
		return m.renderImage()
	default:
		return ""
	}
}

// renderIngredients renders the ingredients list
func (m Model) renderIngredients() string {
	var b strings.Builder

	b.WriteString(SectionTitleStyle.Render("Ingredients") + "\n\n")

	if len(m.currentRecipe.Ingredients) == 0 {
		b.WriteString(MetaStyle.Render("No ingredients found") + "\n")
		return b.String()
	}

	for _, ing := range m.currentRecipe.Ingredients {
		var line string
		if ing.Amount != "" {
			line = fmt.Sprintf("• %s %s %s", ing.Amount, ing.Unit, ing.Name)
		} else {
			line = "• " + ing.Original
		}
		b.WriteString(IngredientStyle.Render(line) + "\n")
	}

	return b.String()
}

// renderInstructions renders the instructions
func (m Model) renderInstructions() string {
	var b strings.Builder

	b.WriteString(SectionTitleStyle.Render("Instructions") + "\n\n")

	if len(m.currentRecipe.Instructions) == 0 {
		b.WriteString(MetaStyle.Render("No instructions found") + "\n")
		return b.String()
	}

	for i, inst := range m.currentRecipe.Instructions {
		step := fmt.Sprintf("%d. %s", i+1, inst)
		// Word wrap long instructions
		wrapped := wordWrap(step, m.viewport.Width-4)
		b.WriteString(InstructionStyle.Render(wrapped) + "\n\n")
	}

	return b.String()
}

// renderImage renders the recipe image
func (m Model) renderImage() string {
	var b strings.Builder

	b.WriteString(SectionTitleStyle.Render("Photo") + "\n\n")

	if m.recipeImage != "" {
		b.WriteString(m.recipeImage)
	} else if len(m.currentRecipe.ImagePaths) > 0 {
		b.WriteString(MetaStyle.Render("Loading image...") + "\n")
	} else {
		placeholder := ImagePlaceholderStyle.Render("No image available")
		b.WriteString(placeholder + "\n")
	}

	return b.String()
}

// renderLoading renders the loading screen
func (m Model) renderLoading() string {
	var b strings.Builder

	title := TitleStyle.Render("Recipe Tracker")
	b.WriteString(title + "\n\n")

	loadingText := m.spinner.View() + " " + m.loadingMsg
	b.WriteString(SpinnerStyle.Render(loadingText) + "\n")

	return BaseStyle.Render(b.String())
}

// wordWrap wraps text to a specified width
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	var line strings.Builder

	words := strings.Fields(text)
	for i, word := range words {
		if line.Len()+len(word)+1 > width && line.Len() > 0 {
			result.WriteString(line.String() + "\n")
			line.Reset()
		}
		if line.Len() > 0 {
			line.WriteString(" ")
		}
		line.WriteString(word)

		if i == len(words)-1 {
			result.WriteString(line.String())
		}
	}

	return result.String()
}
