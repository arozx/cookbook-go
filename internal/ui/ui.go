package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cookbook-go/recipe-tracker/internal/images"
	"github.com/cookbook-go/recipe-tracker/internal/models"
)

// SyncNotifier interface for notifying about recipe changes
type SyncNotifier interface {
	NotifyChange(eventType string, recipe *models.Recipe, recipeID string)
}

// RecipeFetcher interface for parsing recipes from URLs
type RecipeFetcher interface {
	FetchAndParse(url string) (*models.Recipe, error)
}

// RemoteRecipeFetcher interface for remote parsing (via API)
type RemoteRecipeFetcher interface {
	ParseRecipeURL(url string) (*models.Recipe, error)
}

// View represents different screens in the app
type View int

const (
	ViewList View = iota
	ViewAdd
	ViewRecipe
	ViewLoading
	ViewSearch
	ViewEditNotes
)

// Model represents the application state
type Model struct {
	// Core dependencies - use interface for flexibility
	store       models.RecipeRepository
	fetcher     RecipeFetcher       // Local parser
	remoteFetch RemoteRecipeFetcher // Remote parser (when connected to server)
	downloader  *images.Downloader
	isRemote    bool // true if connected to remote server

	// API server for sync notifications (server mode only)
	apiServer SyncNotifier

	// Remote client for sync to external server
	remoteClient models.RecipeRepository

	// UI state
	view         View
	previousView View
	width        int
	height       int
	ready        bool

	// Remote connection info
	serverURL string

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
	imageTab      int  // 0: ingredients, 1: instructions, 2: image, 3: notes
	useGraphics   bool // true if using Kitty/iTerm2/Sixel (non-text graphics)

	// Loading state
	spinner    spinner.Model
	loadingMsg string

	// Search state
	searchInput   textinput.Model
	searchResults []models.Recipe
	allRecipes    []models.Recipe
	isSearching   bool

	// Notes state
	notesInput textarea.Model
}

// Messages
type recipeLoadedMsg struct {
	recipe *models.Recipe
	err    error
}

type recipeRefreshedMsg struct {
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

// remoteEventMsg is sent when a remote event is received
type remoteEventMsg struct {
	eventType string
	recipe    *models.Recipe
	recipeID  string
}

// RemoteEventMsg is the exported version for use by main.go
type RemoteEventMsg struct {
	EventType string
	Recipe    *models.Recipe
	RecipeID  string
}

// NewModel creates a new application model with local storage
func NewModel(store models.RecipeRepository, fetcher RecipeFetcher, downloader *images.Downloader) Model {
	return newModelInternal(store, fetcher, nil, downloader, false, "")
}

// NewRemoteModel creates a new application model connected to a remote server
func NewRemoteModel(store models.RecipeRepository, remoteFetcher RemoteRecipeFetcher, downloader *images.Downloader, serverURL string) Model {
	return newModelInternal(store, nil, remoteFetcher, downloader, true, serverURL)
}

func newModelInternal(store models.RecipeRepository, fetcher RecipeFetcher, remoteFetch RemoteRecipeFetcher, downloader *images.Downloader, isRemote bool, serverURL string) Model {
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

	// Notes input
	ta := textarea.New()
	ta.Placeholder = "Enter notes here..."
	ta.Focus()
	ta.CharLimit = 2000

	// Viewport
	vp := viewport.New(80, 20)

	allRecipes := store.GetAllRecipes()

	// Check if we're using a graphics protocol (non-text based)
	var useGraphics bool
	if downloader != nil {
		protocol := downloader.GetProtocol()
		useGraphics = protocol == images.ProtocolKitty ||
			protocol == images.ProtocolITerm2 ||
			protocol == images.ProtocolSixel
	}

	return Model{
		store:       store,
		fetcher:     fetcher,
		remoteFetch: remoteFetch,
		downloader:  downloader,
		isRemote:    isRemote,
		serverURL:   serverURL,
		view:        ViewList,
		recipes:     allRecipes,
		allRecipes:  allRecipes,
		urlInput:    ti,
		searchInput: si,
		notesInput:  ta,
		spinner:     sp,
		viewport:    vp,
		useGraphics: useGraphics,
	}
}

// SetAPIServer sets the API server for sync notifications
func (m *Model) SetAPIServer(server SyncNotifier) {
	m.apiServer = server
}

// SetRemoteClient sets the remote client for sync
func (m *Model) SetRemoteClient(client models.RecipeRepository) {
	m.remoteClient = client
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
		m.notesInput.SetWidth(msg.Width - 4)
		m.notesInput.SetHeight(msg.Height - 10)
		m.ready = true

		// Reload image if we're viewing a recipe with an image
		// Reload image if viewing recipe with photo tab or if using half-block (which is in viewport)
		if m.view == ViewRecipe && m.currentRecipe != nil && len(m.currentRecipe.ImagePaths) > 0 {
			// Only reload if on photo tab (graphics mode) or any tab (half-block needs resize too)
			if m.imageTab == 2 || !m.useGraphics {
				m.recipeImage = "" // Clear cached image to reload at new size
				return m, m.loadImage(m.currentRecipe.ImagePaths[0])
			}
		}

	case recipeLoadedMsg:
		if msg.err != nil {
			m.addError = fmt.Sprintf("Error: %v", msg.err)
			m.view = ViewAdd
			return m, nil
		}

		// Download images (only if we have a downloader)
		if m.downloader != nil && len(msg.recipe.ImageURLs) > 0 {
			msg.recipe.ImagePaths = m.downloader.DownloadAll(msg.recipe.ImageURLs)
		}

		// Save recipe
		if err := m.store.AddRecipe(*msg.recipe); err != nil {
			m.addError = fmt.Sprintf("Error saving: %v", err)
			m.view = ViewAdd
			return m, nil
		}

		// Notify remote client for sync
		if m.remoteClient != nil {
			go func() {
				_ = m.remoteClient.AddRecipe(*msg.recipe)
			}()
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
		if m.downloader != nil && len(msg.recipe.ImagePaths) > 0 {
			return m, m.loadImage(msg.recipe.ImagePaths[0])
		}
		return m, nil

	case imageLoadedMsg:
		if msg.err == nil {
			m.recipeImage = msg.image
		}
		return m, nil

	case recipeRefreshedMsg:
		if msg.err != nil {
			m.addError = fmt.Sprintf("Error refreshing: %v", msg.err)
			m.view = ViewRecipe
			return m, nil
		}

		// Download images if we have new ones
		if m.downloader != nil && len(msg.recipe.ImageURLs) > 0 {
			msg.recipe.ImagePaths = m.downloader.DownloadAll(msg.recipe.ImageURLs)
		}

		// Save the refreshed recipe
		if err := m.store.AddRecipe(*msg.recipe); err != nil {
			m.addError = fmt.Sprintf("Error saving refreshed recipe: %v", err)
			m.view = ViewRecipe
			return m, nil
		}

		// Notify remote client for sync
		if m.remoteClient != nil {
			go func() {
				_ = m.remoteClient.AddRecipe(*msg.recipe)
			}()
		}

		// Notify API server if in server mode
		if m.apiServer != nil {
			m.apiServer.NotifyChange("update", msg.recipe, msg.recipe.ID)
		}

		m.recipes = m.store.GetAllRecipes()
		m.allRecipes = m.recipes
		m.currentRecipe = msg.recipe
		m.addSuccess = "Recipe refreshed successfully!"
		m.view = ViewRecipe
		m.viewport.SetContent(m.renderRecipeContent())

		// Load image for display
		if m.downloader != nil && len(msg.recipe.ImagePaths) > 0 {
			return m, m.loadImage(msg.recipe.ImagePaths[0])
		}
		return m, nil

	case recipesRefreshedMsg:
		m.recipes = msg.recipes
		m.allRecipes = msg.recipes
		return m, nil

	case remoteEventMsg:
		// Handle real-time sync events from remote server
		switch msg.eventType {
		case "add", "update":
			if msg.recipe != nil {
				m.recipes = m.store.GetAllRecipes()
				m.allRecipes = m.recipes
			}
		case "delete":
			m.recipes = m.store.GetAllRecipes()
			m.allRecipes = m.recipes
			// If we're viewing the deleted recipe, go back to list
			if m.currentRecipe != nil && m.currentRecipe.ID == msg.recipeID {
				m.view = ViewList
				m.currentRecipe = nil
			}
		}
		return m, nil

	case RemoteEventMsg:
		// Handle exported remote event message (from main.go)
		switch msg.EventType {
		case "add", "update":
			if msg.Recipe != nil {
				m.recipes = m.store.GetAllRecipes()
				m.allRecipes = m.recipes
			}
		case "delete":
			m.recipes = m.store.GetAllRecipes()
			m.allRecipes = m.recipes
			// If we're viewing the deleted recipe, go back to list
			if m.currentRecipe != nil && m.currentRecipe.ID == msg.RecipeID {
				m.view = ViewList
				m.currentRecipe = nil
			}
		}
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

	case ViewEditNotes:
		var cmd tea.Cmd
		m.notesInput, cmd = m.notesInput.Update(msg)
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
		// Clear screen if leaving photo tab with graphics
		needsClear := m.view == ViewRecipe && m.imageTab == 2 && m.useGraphics
		m.view = ViewList
		m.addError = ""
		m.addSuccess = ""
		m.recipeImage = ""
		if needsClear {
			return m, tea.ClearScreen
		}
		return m, nil

	case "esc":
		if m.view != ViewList {
			// Clear screen if leaving photo tab with graphics
			needsClear := m.view == ViewRecipe && m.imageTab == 2 && m.useGraphics
			m.view = ViewList
			m.addError = ""
			m.addSuccess = ""
			m.recipeImage = ""
			if needsClear {
				return m, tea.ClearScreen
			}
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
	case ViewEditNotes:
		return m.handleEditNotesKeys(msg)
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

			// Notify remote client for sync
			if m.remoteClient != nil {
				go func() {
					_ = m.remoteClient.DeleteRecipe(recipe.ID)
				}()
			}

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
		oldTab := m.imageTab
		m.imageTab = (m.imageTab + 1) % 4
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()
		// Clear graphics and reload if switching to/from photo tab with graphics mode
		if m.useGraphics && oldTab == 2 {
			// Switching away from photo - clear graphics and redraw
			m.recipeImage = ""
			return m, tea.ClearScreen
		}
		if m.useGraphics && m.imageTab == 2 && len(m.currentRecipe.ImagePaths) > 0 {
			m.recipeImage = ""
			return m, m.loadImage(m.currentRecipe.ImagePaths[0])
		}

	case "shift+tab":
		oldTab := m.imageTab
		m.imageTab = (m.imageTab + 3) % 4
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()
		if m.useGraphics && oldTab == 2 {
			m.recipeImage = ""
			return m, tea.ClearScreen
		}
		if m.useGraphics && m.imageTab == 2 && len(m.currentRecipe.ImagePaths) > 0 {
			m.recipeImage = ""
			return m, m.loadImage(m.currentRecipe.ImagePaths[0])
		}

	case "i":
		wasOnPhoto := m.imageTab == 2
		m.imageTab = 0
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()
		if wasOnPhoto && m.useGraphics {
			m.recipeImage = ""
			return m, tea.ClearScreen
		}

	case "s":
		wasOnPhoto := m.imageTab == 2
		m.imageTab = 1
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()
		if wasOnPhoto && m.useGraphics {
			m.recipeImage = ""
			return m, tea.ClearScreen
		}

	case "p":
		m.imageTab = 2
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()
		// Reload image with fullscreen dimensions for graphics mode
		if m.useGraphics && len(m.currentRecipe.ImagePaths) > 0 {
			m.recipeImage = ""
			return m, m.loadImage(m.currentRecipe.ImagePaths[0])
		}

	case "n":
		wasOnPhoto := m.imageTab == 2
		m.imageTab = 3
		m.viewport.SetContent(m.renderRecipeContent())
		m.viewport.GotoTop()
		if wasOnPhoto && m.useGraphics {
			m.recipeImage = ""
			return m, tea.ClearScreen
		}

	case "e":
		if m.imageTab == 3 && m.currentRecipe != nil {
			m.view = ViewEditNotes
			m.notesInput.SetValue(m.currentRecipe.Notes)
			m.notesInput.Focus()
			return m, nil
		}

	case "r":
		// Refresh recipe from source URL
		if m.currentRecipe != nil && m.currentRecipe.URL != "" {
			m.loadingMsg = "Refreshing recipe..."
			m.view = ViewLoading
			return m, m.refreshRecipe(m.currentRecipe)
		}
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

// handleEditNotesKeys handles keys in edit notes view
func (m Model) handleEditNotesKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		if m.currentRecipe != nil {
			m.currentRecipe.Notes = m.notesInput.Value()
			m.currentRecipe.UpdatedAt = time.Now()

			// Save to store
			if err := m.store.AddRecipe(*m.currentRecipe); err != nil {
				// Handle error? For now just go back
			}

			// Notify remote client for sync
			if m.remoteClient != nil {
				go func(r models.Recipe) {
					_ = m.remoteClient.AddRecipe(r)
				}(*m.currentRecipe)
			}

			// Notify API server if in server mode
			if m.apiServer != nil {
				m.apiServer.NotifyChange("update", m.currentRecipe, m.currentRecipe.ID)
			}
		}
		m.view = ViewRecipe
		m.viewport.SetContent(m.renderRecipeContent())
		return m, nil

	case "esc":
		m.view = ViewRecipe
		return m, nil
	}

	var cmd tea.Cmd
	m.notesInput, cmd = m.notesInput.Update(msg)
	return m, cmd
}

// fetchRecipe fetches a recipe from URL
func (m Model) fetchRecipe(url string) tea.Cmd {
	// Use remote fetcher if connected to server, otherwise local
	if m.isRemote && m.remoteFetch != nil {
		return func() tea.Msg {
			recipe, err := m.remoteFetch.ParseRecipeURL(url)
			return recipeLoadedMsg{recipe: recipe, err: err}
		}
	}

	if m.fetcher != nil {
		return func() tea.Msg {
			recipe, err := m.fetcher.FetchAndParse(url)
			return recipeLoadedMsg{recipe: recipe, err: err}
		}
	}

	return func() tea.Msg {
		return recipeLoadedMsg{recipe: nil, err: fmt.Errorf("no recipe fetcher available")}
	}
}

// refreshRecipe re-fetches recipe data from URL while preserving user data
func (m Model) refreshRecipe(existing *models.Recipe) tea.Cmd {
	// Capture values needed in the closure
	url := existing.URL
	id := existing.ID
	notes := existing.Notes
	createdAt := existing.CreatedAt

	// Use remote fetcher if connected to server, otherwise local
	if m.isRemote && m.remoteFetch != nil {
		return func() tea.Msg {
			recipe, err := m.remoteFetch.ParseRecipeURL(url)
			if err != nil {
				return recipeRefreshedMsg{recipe: nil, err: err}
			}
			// Preserve user data
			recipe.ID = id
			recipe.Notes = notes
			recipe.CreatedAt = createdAt
			recipe.UpdatedAt = time.Now()
			return recipeRefreshedMsg{recipe: recipe, err: nil}
		}
	}

	if m.fetcher != nil {
		return func() tea.Msg {
			recipe, err := m.fetcher.FetchAndParse(url)
			if err != nil {
				return recipeRefreshedMsg{recipe: nil, err: err}
			}
			// Preserve user data
			recipe.ID = id
			recipe.Notes = notes
			recipe.CreatedAt = createdAt
			recipe.UpdatedAt = time.Now()
			return recipeRefreshedMsg{recipe: recipe, err: nil}
		}
	}

	return func() tea.Msg {
		return recipeRefreshedMsg{recipe: nil, err: fmt.Errorf("no recipe fetcher available")}
	}
}

// loadImage loads an image for display
func (m Model) loadImage(path string) tea.Cmd {
	// Calculate dimensions based on terminal size and current tab
	// For graphics protocols on photo tab, use more of the screen
	// For half-block or non-photo tabs, fit within viewport

	var width, height int

	// Use fullscreen dimensions for graphics mode on photo tab
	useFullscreen := m.useGraphics && m.imageTab == 2

	if useFullscreen {
		// Graphics protocols: use most of the screen height, leave room for header/footer
		width = m.width - 8
		height = m.height - 8 // Leave room for title bar and help text
		if width < 40 {
			width = 40
		}
		if height < 20 {
			height = 20
		}
	} else {
		// Half-block: fit within viewport
		width = m.viewport.Width - 4
		if width < 20 {
			width = 20
		}
		if width > 100 {
			width = 100
		}
		height = m.viewport.Height - 4
		if height < 10 {
			height = 10
		}
		if height > 50 {
			height = 50
		}
	}

	useGraphics := m.useGraphics && m.imageTab == 2
	downloader := m.downloader

	return func() tea.Msg {
		var img string
		var err error
		if useGraphics {
			// Use the detected graphics protocol
			img, err = downloader.RenderImage(path, width, height)
		} else {
			// Use safe half-block rendering for viewport
			img, err = downloader.RenderImageSafe(path, width, height)
		}
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
	case ViewEditNotes:
		return m.renderEditNotes()
	default:
		return "Unknown view"
	}
}

// renderEditNotes renders the edit notes view
func (m Model) renderEditNotes() string {
	var b strings.Builder

	title := TitleStyle.Render("Edit Notes")
	b.WriteString(title + "\n\n")

	b.WriteString(m.notesInput.View() + "\n\n")

	help := HelpStyle.Render("ctrl+s: save | esc: cancel")
	b.WriteString(help)

	return BaseStyle.Render(b.String())
}

// renderList renders the recipe list view
func (m Model) renderList() string {
	var b strings.Builder

	// Title with connection indicator
	if m.isRemote {
		title := TitleStyle.Render("Recipe Tracker") + "  " + SuccessStyle.Render("● connected")
		b.WriteString(title + "\n")
		b.WriteString(MetaStyle.Render("  "+m.serverURL) + "\n\n")
	} else if m.remoteClient != nil {
		title := TitleStyle.Render("Recipe Tracker") + "  " + SuccessStyle.Render("● syncing")
		b.WriteString(title + "\n")
		b.WriteString(MetaStyle.Render("  local + sync") + "\n\n")
	} else {
		title := TitleStyle.Render("Recipe Tracker")
		b.WriteString(title + "\n\n")
	}

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

	// Special handling for photo tab with graphics protocols
	// Render image directly without viewport to avoid overlap
	if m.imageTab == 2 && m.useGraphics {
		return m.renderPhotoFullscreen()
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
	tabs := []string{"[i]ngredients", "[s]teps", "[p]hoto", "[n]otes"}
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
	helpText := "tab: switch tabs | j/k: scroll | r: refresh | esc: back"
	if m.imageTab == 3 {
		helpText = "e: edit notes | " + helpText
	}
	help := HelpStyle.Render(helpText)
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
	case 3: // Notes
		return m.renderNotes()
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

// renderNotes renders the recipe notes
func (m Model) renderNotes() string {
	var b strings.Builder

	b.WriteString(SectionTitleStyle.Render("Notes") + "\n\n")

	if m.currentRecipe.Notes == "" {
		b.WriteString(MetaStyle.Render("No notes yet.") + "\n")
		return b.String()
	}

	// Word wrap notes
	wrapped := wordWrap(m.currentRecipe.Notes, m.viewport.Width-4)
	b.WriteString(InstructionStyle.Render(wrapped) + "\n")

	return b.String()
}

// renderImage renders the recipe image (for half-block in viewport)
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

// renderPhotoFullscreen renders the photo tab for graphics protocols
// This renders directly without a viewport to prevent text overlap
func (m Model) renderPhotoFullscreen() string {
	var b strings.Builder

	// Minimal header - just recipe title
	title := RecipeTitleStyle.Render(Truncate(m.currentRecipe.Title, m.width-10))
	b.WriteString(title + "\n")

	// Tabs on same line to save space
	tabs := []string{"[i]ngredients", "[s]teps", "[p]hoto", "[n]otes"}
	var tabLine strings.Builder
	for i, tab := range tabs {
		if i == m.imageTab {
			tabLine.WriteString(ActiveTabStyle.Render(tab))
		} else {
			tabLine.WriteString(InactiveTabStyle.Render(tab))
		}
	}
	b.WriteString(tabLine.String() + "\n\n")

	// Render image directly (no viewport wrapper)
	if m.recipeImage != "" {
		b.WriteString(m.recipeImage)
	} else if len(m.currentRecipe.ImagePaths) > 0 {
		b.WriteString(MetaStyle.Render("Loading image...") + "\n")
	} else {
		b.WriteString(MetaStyle.Render("No image available") + "\n")
	}

	// Help at bottom
	b.WriteString("\n" + HelpStyle.Render("tab: switch tabs | r: refresh | esc: back"))

	return BaseStyle.Render(b.String())
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
