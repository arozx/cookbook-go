package api

// webAppHTML contains the embedded mobile webapp with terminal theme using Catppuccin Mocha
const webAppHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <meta name="theme-color" content="#1e1e2e">
    <title>Recipe Tracker</title>
    <link rel="manifest" href="data:application/json,{&quot;name&quot;:&quot;Recipe Tracker&quot;,&quot;short_name&quot;:&quot;Recipes&quot;,&quot;display&quot;:&quot;standalone&quot;,&quot;background_color&quot;:&quot;%231e1e2e&quot;,&quot;theme_color&quot;:&quot;%231e1e2e&quot;}">
    <style>
        @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap');

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        /* Catppuccin Mocha Palette */
        :root {
            --ctp-rosewater: #f5e0dc;
            --ctp-flamingo: #f2cdcd;
            --ctp-pink: #f5c2e7;
            --ctp-mauve: #cba6f7;
            --ctp-red: #f38ba8;
            --ctp-maroon: #eba0ac;
            --ctp-peach: #fab387;
            --ctp-yellow: #f9e2af;
            --ctp-green: #a6e3a1;
            --ctp-teal: #94e2d5;
            --ctp-sky: #89dceb;
            --ctp-sapphire: #74c7ec;
            --ctp-blue: #89b4fa;
            --ctp-lavender: #b4befe;
            --ctp-text: #cdd6f4;
            --ctp-subtext1: #bac2de;
            --ctp-subtext0: #a6adc8;
            --ctp-overlay2: #9399b2;
            --ctp-overlay1: #7f849c;
            --ctp-overlay0: #6c7086;
            --ctp-surface2: #585b70;
            --ctp-surface1: #45475a;
            --ctp-surface0: #313244;
            --ctp-base: #1e1e2e;
            --ctp-mantle: #181825;
            --ctp-crust: #11111b;
        }

        body {
            font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', Consolas, monospace;
            background: var(--ctp-base);
            color: var(--ctp-text);
            min-height: 100vh;
            min-height: 100dvh;
            overflow-x: hidden;
            font-size: 14px;
            line-height: 1.5;
        }

        .app {
            max-width: 100%;
            min-height: 100vh;
            min-height: 100dvh;
            display: flex;
            flex-direction: column;
        }

        /* Terminal Header */
        .header {
            background: var(--ctp-crust);
            padding: 12px 16px;
            position: sticky;
            top: 0;
            z-index: 100;
            border-bottom: 1px solid var(--ctp-surface0);
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 12px;
        }

        .header-title {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .header-title::before {
            content: '>';
            color: var(--ctp-green);
            font-weight: 700;
        }

        .header h1 {
            font-size: 1rem;
            font-weight: 600;
            color: var(--ctp-mauve);
        }

        .header-actions {
            display: flex;
            gap: 8px;
        }

        /* Terminal Prompt Search */
        .search-container {
            padding: 12px 16px;
            background: var(--ctp-mantle);
            border-bottom: 1px solid var(--ctp-surface0);
        }

        .search-box {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .search-prompt {
            color: var(--ctp-green);
            font-weight: 600;
            white-space: nowrap;
        }

        .search-input {
            flex: 1;
            padding: 10px 12px;
            border: 1px solid var(--ctp-surface1);
            border-radius: 4px;
            background: var(--ctp-surface0);
            color: var(--ctp-text);
            font-family: inherit;
            font-size: 14px;
            outline: none;
            transition: border-color 0.2s;
        }

        .search-input:focus {
            border-color: var(--ctp-mauve);
            box-shadow: 0 0 0 2px rgba(203, 166, 247, 0.2);
        }

        .search-input::placeholder {
            color: var(--ctp-overlay0);
        }

        /* Buttons */
        .btn {
            padding: 8px 16px;
            border: 1px solid var(--ctp-surface1);
            border-radius: 4px;
            font-family: inherit;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            background: var(--ctp-surface0);
            color: var(--ctp-text);
        }

        .btn:hover {
            background: var(--ctp-surface1);
            border-color: var(--ctp-surface2);
        }

        .btn-primary {
            background: var(--ctp-mauve);
            border-color: var(--ctp-mauve);
            color: var(--ctp-crust);
        }

        .btn-primary:hover {
            background: var(--ctp-lavender);
            border-color: var(--ctp-lavender);
        }

        .btn-icon {
            width: 36px;
            height: 36px;
            padding: 0;
            border-radius: 4px;
        }

        .btn-danger {
            background: var(--ctp-red);
            border-color: var(--ctp-red);
            color: var(--ctp-crust);
        }

        .btn-danger:hover {
            background: var(--ctp-maroon);
            border-color: var(--ctp-maroon);
        }

        /* Recipe List */
        .recipe-list {
            flex: 1;
            padding: 8px;
            display: flex;
            flex-direction: column;
            gap: 2px;
            overflow-y: auto;
            padding-bottom: 100px;
        }

        .recipe-card {
            background: var(--ctp-surface0);
            border: 1px solid var(--ctp-surface1);
            border-radius: 4px;
            overflow: hidden;
            cursor: pointer;
            transition: all 0.15s;
            display: flex;
            gap: 12px;
            padding: 12px;
        }

        .recipe-card:hover {
            background: var(--ctp-surface1);
            border-color: var(--ctp-mauve);
        }

        .recipe-card::before {
            content: '>';
            color: var(--ctp-overlay0);
            font-weight: 600;
            padding-top: 2px;
            transition: color 0.15s;
        }

        .recipe-card:hover::before {
            color: var(--ctp-green);
        }

        .recipe-card-image {
            width: 60px;
            height: 60px;
            border-radius: 4px;
            object-fit: cover;
            background: var(--ctp-mantle);
            flex-shrink: 0;
            border: 1px solid var(--ctp-surface1);
        }

        .recipe-card-content {
            flex: 1;
            min-width: 0;
            display: flex;
            flex-direction: column;
            justify-content: center;
        }

        .recipe-card-title {
            font-size: 0.95rem;
            font-weight: 600;
            margin-bottom: 4px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            color: var(--ctp-text);
        }

        .recipe-card-meta {
            font-size: 0.8rem;
            color: var(--ctp-subtext0);
            display: flex;
            gap: 12px;
        }

        .recipe-card-meta span::before {
            content: '[';
            color: var(--ctp-overlay0);
        }

        .recipe-card-meta span::after {
            content: ']';
            color: var(--ctp-overlay0);
        }

        /* Recipe Detail View */
        .recipe-detail {
            display: none;
            flex-direction: column;
            min-height: 100vh;
            min-height: 100dvh;
            background: var(--ctp-base);
        }

        .recipe-detail.active {
            display: flex;
        }

        .recipe-hero {
            position: relative;
            height: 200px;
            background: var(--ctp-mantle);
            border-bottom: 1px solid var(--ctp-surface0);
        }

        .recipe-hero img {
            width: 100%;
            height: 100%;
            object-fit: cover;
            opacity: 0.8;
        }

        .recipe-hero-overlay {
            position: absolute;
            bottom: 0;
            left: 0;
            right: 0;
            padding: 60px 16px 16px;
            background: linear-gradient(transparent, var(--ctp-crust));
        }

        .recipe-hero h2 {
            font-size: 1.2rem;
            font-weight: 700;
            color: var(--ctp-text);
            text-shadow: 0 2px 4px rgba(0,0,0,0.5);
        }

        .recipe-hero h2::before {
            content: '# ';
            color: var(--ctp-mauve);
        }

        .back-btn {
            position: absolute;
            top: 12px;
            left: 12px;
            background: rgba(17, 17, 27, 0.8);
            backdrop-filter: blur(8px);
            -webkit-backdrop-filter: blur(8px);
        }

        .delete-btn {
            position: absolute;
            top: 12px;
            right: 12px;
            background: rgba(243, 139, 168, 0.9);
            backdrop-filter: blur(8px);
            -webkit-backdrop-filter: blur(8px);
        }

        .recipe-info-bar {
            display: flex;
            gap: 8px;
            padding: 12px 16px;
            background: var(--ctp-mantle);
            border-bottom: 1px solid var(--ctp-surface0);
            overflow-x: auto;
            font-size: 0.85rem;
        }

        .info-item {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 6px 10px;
            background: var(--ctp-surface0);
            border-radius: 4px;
            border: 1px solid var(--ctp-surface1);
            white-space: nowrap;
        }

        .info-label {
            color: var(--ctp-overlay1);
        }

        .info-value {
            color: var(--ctp-peach);
            font-weight: 600;
        }

        /* Tabs */
        .tabs {
            display: flex;
            background: var(--ctp-crust);
            border-bottom: 1px solid var(--ctp-surface0);
            padding: 0 8px;
        }

        .tab {
            flex: 1;
            padding: 12px 8px;
            text-align: center;
            font-size: 0.85rem;
            font-weight: 500;
            color: var(--ctp-overlay1);
            cursor: pointer;
            border-bottom: 2px solid transparent;
            transition: all 0.2s;
        }

        .tab:hover {
            color: var(--ctp-text);
        }

        .tab.active {
            color: var(--ctp-green);
            border-bottom-color: var(--ctp-green);
        }

        .tab-content {
            flex: 1;
            padding: 16px;
            overflow-y: auto;
        }

        .tab-panel {
            display: none;
        }

        .tab-panel.active {
            display: block;
        }

        /* Ingredients */
        .section-header {
            color: var(--ctp-mauve);
            font-weight: 600;
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px dashed var(--ctp-surface1);
        }

        .section-header::before {
            content: '## ';
            color: var(--ctp-overlay0);
        }

        .ingredient-item {
            display: flex;
            align-items: flex-start;
            padding: 10px 0;
            border-bottom: 1px solid var(--ctp-surface0);
        }

        .ingredient-checkbox {
            width: 20px;
            height: 20px;
            border: 2px solid var(--ctp-surface2);
            border-radius: 3px;
            margin-right: 12px;
            flex-shrink: 0;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.2s;
            background: var(--ctp-surface0);
        }

        .ingredient-checkbox:hover {
            border-color: var(--ctp-green);
        }

        .ingredient-checkbox.checked {
            background: var(--ctp-green);
            border-color: var(--ctp-green);
        }

        .ingredient-checkbox.checked::after {
            content: 'x';
            color: var(--ctp-crust);
            font-size: 12px;
            font-weight: 700;
        }

        .ingredient-text {
            flex: 1;
            color: var(--ctp-text);
        }

        .ingredient-text::before {
            content: '- ';
            color: var(--ctp-overlay0);
        }

        .ingredient-text.checked {
            text-decoration: line-through;
            color: var(--ctp-overlay0);
        }

        /* Instructions */
        .instruction-item {
            display: flex;
            margin-bottom: 16px;
            padding: 12px;
            background: var(--ctp-surface0);
            border-radius: 4px;
            border-left: 3px solid var(--ctp-mauve);
        }

        .instruction-number {
            min-width: 28px;
            height: 28px;
            background: var(--ctp-mauve);
            border-radius: 4px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 700;
            font-size: 0.85rem;
            margin-right: 12px;
            flex-shrink: 0;
            color: var(--ctp-crust);
        }

        .instruction-text {
            flex: 1;
            line-height: 1.6;
            color: var(--ctp-subtext1);
        }

        /* Add Recipe Modal */
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(17, 17, 27, 0.9);
            z-index: 200;
            align-items: flex-end;
            justify-content: center;
        }

        .modal.active {
            display: flex;
        }

        .modal-content {
            background: var(--ctp-mantle);
            width: 100%;
            max-height: 90vh;
            border-radius: 8px 8px 0 0;
            border: 1px solid var(--ctp-surface0);
            border-bottom: none;
            padding: 20px;
            overflow-y: auto;
        }

        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 12px;
            border-bottom: 1px solid var(--ctp-surface0);
        }

        .modal-header h3 {
            font-size: 1rem;
            font-weight: 600;
            color: var(--ctp-mauve);
        }

        .modal-header h3::before {
            content: '> ';
            color: var(--ctp-green);
        }

        .form-group {
            margin-bottom: 16px;
        }

        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-size: 0.85rem;
            color: var(--ctp-subtext0);
        }

        .form-group label::before {
            content: '$ ';
            color: var(--ctp-green);
        }

        .form-group input {
            width: 100%;
            padding: 12px;
            border: 1px solid var(--ctp-surface1);
            border-radius: 4px;
            background: var(--ctp-surface0);
            color: var(--ctp-text);
            font-family: inherit;
            font-size: 14px;
            outline: none;
        }

        .form-group input:focus {
            border-color: var(--ctp-mauve);
            box-shadow: 0 0 0 2px rgba(203, 166, 247, 0.2);
        }

        /* Sync Status */
        .sync-status {
            position: fixed;
            bottom: 20px;
            left: 50%;
            transform: translateX(-50%);
            background: var(--ctp-surface0);
            border: 1px solid var(--ctp-surface1);
            padding: 10px 20px;
            border-radius: 4px;
            font-size: 0.85rem;
            display: flex;
            align-items: center;
            gap: 8px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.4);
            opacity: 0;
            transition: opacity 0.3s;
            z-index: 150;
        }

        .sync-status.visible {
            opacity: 1;
        }

        .sync-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: var(--ctp-green);
        }

        .sync-dot.syncing {
            background: var(--ctp-yellow);
            animation: pulse 1s infinite;
        }

        .sync-dot.error {
            background: var(--ctp-red);
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        /* Loading */
        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px;
            color: var(--ctp-subtext0);
        }

        .spinner {
            width: 20px;
            height: 20px;
            border: 2px solid var(--ctp-surface2);
            border-top-color: var(--ctp-mauve);
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin-right: 12px;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* Empty State */
        .empty-state {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 60px 20px;
            text-align: center;
        }

        .empty-state-icon {
            font-size: 3rem;
            margin-bottom: 16px;
            opacity: 0.5;
            color: var(--ctp-overlay0);
        }

        .empty-state h3 {
            font-size: 1rem;
            margin-bottom: 8px;
            color: var(--ctp-text);
        }

        .empty-state p {
            color: var(--ctp-subtext0);
            margin-bottom: 20px;
            font-size: 0.9rem;
        }

        .empty-state code {
            background: var(--ctp-surface0);
            padding: 2px 6px;
            border-radius: 3px;
            color: var(--ctp-green);
        }

        /* FAB */
        .fab {
            position: fixed;
            bottom: 24px;
            right: 24px;
            width: 56px;
            height: 56px;
            border-radius: 8px;
            background: var(--ctp-mauve);
            color: var(--ctp-crust);
            border: none;
            font-size: 24px;
            font-weight: 700;
            cursor: pointer;
            box-shadow: 0 4px 16px rgba(203, 166, 247, 0.3);
            transition: transform 0.2s, box-shadow 0.2s, background 0.2s;
            z-index: 100;
            font-family: inherit;
        }

        .fab:hover {
            transform: scale(1.05);
            background: var(--ctp-lavender);
            box-shadow: 0 6px 24px rgba(203, 166, 247, 0.4);
        }

        /* Terminal cursor blink effect */
        .cursor {
            display: inline-block;
            width: 8px;
            height: 16px;
            background: var(--ctp-text);
            animation: blink 1s step-end infinite;
            vertical-align: text-bottom;
            margin-left: 2px;
        }

        @keyframes blink {
            0%, 100% { opacity: 1; }
            50% { opacity: 0; }
        }

        /* Scrollbar */
        ::-webkit-scrollbar {
            width: 8px;
            height: 8px;
        }

        ::-webkit-scrollbar-track {
            background: var(--ctp-mantle);
        }

        ::-webkit-scrollbar-thumb {
            background: var(--ctp-surface2);
            border-radius: 4px;
        }

        ::-webkit-scrollbar-thumb:hover {
            background: var(--ctp-overlay0);
        }

        /* Hide when viewing recipe */
        .recipe-detail.active ~ .fab {
            display: none;
        }
    </style>
</head>
<body>
    <div class="app">
        <!-- Main List View -->
        <div class="list-view">
            <header class="header">
                <div class="header-title">
                    <h1>recipe-tracker</h1>
                </div>
                <div class="header-actions">
                    <button class="btn btn-icon" onclick="syncRecipes()" title="Sync">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M23 4v6h-6M1 20v-6h6"/>
                            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                        </svg>
                    </button>
                </div>
            </header>

            <div class="search-container">
                <div class="search-box">
                    <span class="search-prompt">~/recipes $</span>
                    <input type="text" class="search-input" id="searchInput" placeholder="grep ..." oninput="handleSearch(this.value)">
                </div>
            </div>

            <div class="recipe-list" id="recipeList">
                <div class="loading">
                    <div class="spinner"></div>
                    Loading recipes...
                </div>
            </div>
        </div>

        <!-- Recipe Detail View -->
        <div class="recipe-detail" id="recipeDetail">
            <div class="recipe-hero">
                <img id="recipeImage" src="" alt="">
                <button class="btn btn-icon back-btn" onclick="hideRecipeDetail()">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M19 12H5M12 19l-7-7 7-7"/>
                    </svg>
                </button>
                <button class="btn btn-icon delete-btn" onclick="deleteCurrentRecipe()" title="Delete">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                    </svg>
                </button>
                <div class="recipe-hero-overlay">
                    <h2 id="recipeTitle"></h2>
                </div>
            </div>

            <div class="recipe-info-bar">
                <div class="info-item">
                    <span class="info-label">prep:</span>
                    <span class="info-value" id="recipePrepTime">-</span>
                </div>
                <div class="info-item">
                    <span class="info-label">cook:</span>
                    <span class="info-value" id="recipeCookTime">-</span>
                </div>
                <div class="info-item">
                    <span class="info-label">serves:</span>
                    <span class="info-value" id="recipeServings">-</span>
                </div>
            </div>

            <div class="tabs">
                <div class="tab active" data-tab="ingredients" onclick="switchTab('ingredients')">ingredients</div>
                <div class="tab" data-tab="instructions" onclick="switchTab('instructions')">instructions</div>
                <div class="tab" data-tab="notes" onclick="switchTab('notes')">notes</div>
            </div>

            <div class="tab-content">
                <div class="tab-panel active" id="ingredientsPanel"></div>
                <div class="tab-panel" id="instructionsPanel"></div>
                <div class="tab-panel" id="notesPanel">
                    <div class="section-header">notes</div>
                    <textarea id="notesArea" style="width: 100%; min-height: 200px; background: var(--ctp-surface0); color: var(--ctp-text); border: 1px solid var(--ctp-surface1); border-radius: 4px; padding: 12px; font-family: inherit; font-size: 14px; outline: none; margin-bottom: 12px; resize: vertical;" placeholder="Add notes here..."></textarea>
                    <button class="btn btn-primary" style="width: 100%;" onclick="saveNotes()">$ save-notes</button>
                </div>
            </div>
        </div>

        <!-- Add Recipe FAB -->
        <button class="fab" onclick="showAddModal()">+</button>

        <!-- Add Recipe Modal -->
        <div class="modal" id="addModal">
            <div class="modal-content">
                <div class="modal-header">
                    <h3>add-recipe</h3>
                    <button class="btn btn-icon" onclick="hideAddModal()">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M18 6L6 18M6 6l12 12"/>
                        </svg>
                    </button>
                </div>
                <div class="form-group">
                    <label>url</label>
                    <input type="url" id="recipeUrl" placeholder="https://example.com/recipe">
                </div>
                <button class="btn btn-primary" style="width: 100%;" onclick="addRecipe()" id="addRecipeBtn">
                    $ fetch --save
                </button>
            </div>
        </div>

        <!-- Sync Status -->
        <div class="sync-status" id="syncStatus">
            <div class="sync-dot"></div>
            <span class="sync-text">connected</span>
        </div>
    </div>

    <script>
        // State
        let recipes = [];
        let currentRecipe = null;
        let eventSource = null;

        // API Base URL
        const API_BASE = window.location.origin;

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            loadRecipes();
            setupSSE();
        });

        // Load recipes from API
        async function loadRecipes() {
            try {
                const response = await fetch(API_BASE + '/api/recipes');
                const data = await response.json();
                recipes = data.recipes || [];
                renderRecipeList();
            } catch (error) {
                console.error('Failed to load recipes:', error);
                showSyncStatus('error', 'connection failed');
            }
        }

        // Render recipe list
        function renderRecipeList(filteredRecipes = null) {
            const list = document.getElementById('recipeList');
            const displayRecipes = filteredRecipes || recipes;

            if (displayRecipes.length === 0) {
                list.innerHTML = ` + "`" + `
                    <div class="empty-state">
                        <div class="empty-state-icon">[]</div>
                        <h3>No recipes found</h3>
                        <p>Press <code>+</code> to add your first recipe</p>
                    </div>
                ` + "`" + `;
                return;
            }

            list.innerHTML = displayRecipes.map(recipe => ` + "`" + `
                <div class="recipe-card" onclick="showRecipeDetail('${recipe.id}')">
                    <img class="recipe-card-image" 
                         src="${recipe.image_urls && recipe.image_urls[0] ? recipe.image_urls[0] : ''}" 
                         alt="${recipe.title}"
                         onerror="this.style.display='none'">
                    <div class="recipe-card-content">
                        <div class="recipe-card-title">${recipe.title}</div>
                        <div class="recipe-card-meta">
                            ${recipe.prep_time ? '<span>' + recipe.prep_time + '</span>' : ''}
                            ${recipe.cook_time ? '<span>' + recipe.cook_time + '</span>' : ''}
                        </div>
                    </div>
                </div>
            ` + "`" + `).join('');
        }

        // Search recipes
        function handleSearch(query) {
            if (!query.trim()) {
                renderRecipeList();
                return;
            }
            const filtered = recipes.filter(r => 
                r.title.toLowerCase().includes(query.toLowerCase())
            );
            renderRecipeList(filtered);
        }

        // Show recipe detail
        function showRecipeDetail(id) {
            currentRecipe = recipes.find(r => r.id === id);
            if (!currentRecipe) return;

            document.getElementById('recipeTitle').textContent = currentRecipe.title;
            document.getElementById('recipeImage').src = currentRecipe.image_urls?.[0] || '';
            document.getElementById('recipePrepTime').textContent = currentRecipe.prep_time || '-';
            document.getElementById('recipeCookTime').textContent = currentRecipe.cook_time || '-';
            document.getElementById('recipeServings').textContent = currentRecipe.servings || '-';

            // Render ingredients
            const ingredientsHtml = '<div class="section-header">ingredients</div>' + 
                ((currentRecipe.ingredients || []).map((ing, i) => ` + "`" + `
                <div class="ingredient-item">
                    <div class="ingredient-checkbox" onclick="toggleIngredient(this)" data-index="${i}"></div>
                    <span class="ingredient-text">${ing.original || ing.name}</span>
                </div>
            ` + "`" + `).join('') || '<p style="color: var(--ctp-overlay0)">No ingredients listed</p>');
            document.getElementById('ingredientsPanel').innerHTML = ingredientsHtml;

            // Render instructions
            const instructionsHtml = '<div class="section-header">instructions</div>' +
                ((currentRecipe.instructions || []).map((inst, i) => ` + "`" + `
                <div class="instruction-item">
                    <div class="instruction-number">${i + 1}</div>
                    <div class="instruction-text">${inst}</div>
                </div>
            ` + "`" + `).join('') || '<p style="color: var(--ctp-overlay0)">No instructions listed</p>');
            document.getElementById('instructionsPanel').innerHTML = instructionsHtml;
            document.getElementById('notesArea').value = currentRecipe.notes || '';

            // Show detail view
            document.querySelector('.list-view').style.display = 'none';
            document.getElementById('recipeDetail').classList.add('active');
            document.querySelector('.fab').style.display = 'none';

            // Reset tabs
            switchTab('ingredients');
        }

        // Hide recipe detail
        function hideRecipeDetail() {
            document.querySelector('.list-view').style.display = 'block';
            document.getElementById('recipeDetail').classList.remove('active');
            document.querySelector('.fab').style.display = 'block';
            currentRecipe = null;
        }

        // Toggle ingredient checkbox
        function toggleIngredient(el) {
            el.classList.toggle('checked');
            el.nextElementSibling.classList.toggle('checked');
        }

        // Switch tabs
        function switchTab(tabName) {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
            document.querySelector(` + "`" + `.tab[data-tab="${tabName}"]` + "`" + `).classList.add('active');
            document.getElementById(tabName + 'Panel').classList.add('active');
        }

        // Show add modal
        function showAddModal() {
            document.getElementById('addModal').classList.add('active');
            document.getElementById('recipeUrl').focus();
        }

        // Hide add modal
        function hideAddModal() {
            document.getElementById('addModal').classList.remove('active');
            document.getElementById('recipeUrl').value = '';
        }

        // Add recipe
        async function addRecipe() {
            const url = document.getElementById('recipeUrl').value.trim();
            if (!url) return;

            const btn = document.getElementById('addRecipeBtn');
            btn.textContent = '$ fetching...';
            btn.disabled = true;

            try {
                // Parse the recipe URL
                const parseResponse = await fetch(API_BASE + '/api/parse', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url })
                });

                if (!parseResponse.ok) {
                    throw new Error(await parseResponse.text());
                }

                const recipe = await parseResponse.json();

                // Save the recipe
                const saveResponse = await fetch(API_BASE + '/api/recipes', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(recipe)
                });

                if (!saveResponse.ok) {
                    throw new Error(await saveResponse.text());
                }

                const savedRecipe = await saveResponse.json();
                recipes.unshift(savedRecipe);
                renderRecipeList();
                hideAddModal();
                showSyncStatus('success', 'recipe saved');

            } catch (error) {
                console.error('Failed to add recipe:', error);
                showSyncStatus('error', error.message || 'fetch failed');
            } finally {
                btn.textContent = '$ fetch --save';
                btn.disabled = false;
            }
        }

        // Save notes
        async function saveNotes() {
            if (!currentRecipe) return;
            const notes = document.getElementById('notesArea').value;
            
            try {
                const updatedRecipe = { ...currentRecipe, notes };
                const response = await fetch(API_BASE + '/api/recipes/' + currentRecipe.id, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(updatedRecipe)
                });

                if (response.ok) {
                    currentRecipe.notes = notes;
                    // Update in main list too
                    const index = recipes.findIndex(r => r.id === currentRecipe.id);
                    if (index >= 0) recipes[index].notes = notes;
                    
                    showSyncStatus('success', 'notes saved');
                } else {
                    throw new Error(await response.text());
                }
            } catch (error) {
                console.error('Failed to save notes:', error);
                showSyncStatus('error', 'save failed');
            }
        }

        // Delete recipe
        async function deleteCurrentRecipe() {
            if (!currentRecipe) return;
            if (!confirm('Delete this recipe?')) return;

            try {
                const response = await fetch(API_BASE + '/api/recipes/' + currentRecipe.id, {
                    method: 'DELETE'
                });

                if (response.ok) {
                    recipes = recipes.filter(r => r.id !== currentRecipe.id);
                    hideRecipeDetail();
                    renderRecipeList();
                    showSyncStatus('success', 'recipe deleted');
                }
            } catch (error) {
                console.error('Failed to delete recipe:', error);
                showSyncStatus('error', 'delete failed');
            }
        }

        // Sync recipes
        async function syncRecipes() {
            showSyncStatus('syncing', 'syncing...');
            await loadRecipes();
            showSyncStatus('success', 'synced');
        }

        // Setup Server-Sent Events for real-time sync
        function setupSSE() {
            if (eventSource) {
                eventSource.close();
            }

            eventSource = new EventSource(API_BASE + '/api/events');

            eventSource.onopen = () => {
                showSyncStatus('success', 'connected');
            };

            eventSource.addEventListener('add', (e) => {
                const data = JSON.parse(e.data);
                if (data.recipe && !recipes.find(r => r.id === data.recipe.id)) {
                    recipes.unshift(data.recipe);
                    renderRecipeList();
                    showSyncStatus('success', 'new recipe synced');
                }
            });

            eventSource.addEventListener('update', (e) => {
                const data = JSON.parse(e.data);
                if (data.recipe) {
                    const index = recipes.findIndex(r => r.id === data.recipe.id);
                    if (index >= 0) {
                        recipes[index] = data.recipe;
                        renderRecipeList();
                    }
                }
            });

            eventSource.addEventListener('delete', (e) => {
                const data = JSON.parse(e.data);
                if (data.recipe_id) {
                    recipes = recipes.filter(r => r.id !== data.recipe_id);
                    renderRecipeList();
                    if (currentRecipe && currentRecipe.id === data.recipe_id) {
                        hideRecipeDetail();
                    }
                }
            });

            eventSource.onerror = () => {
                showSyncStatus('error', 'connection lost');
                setTimeout(setupSSE, 5000);
            };
        }

        // Show sync status
        function showSyncStatus(type, message) {
            const status = document.getElementById('syncStatus');
            const dot = status.querySelector('.sync-dot');
            const text = status.querySelector('.sync-text');

            dot.className = 'sync-dot';
            if (type === 'syncing') dot.classList.add('syncing');
            if (type === 'error') dot.classList.add('error');

            text.textContent = message;
            status.classList.add('visible');

            setTimeout(() => {
                status.classList.remove('visible');
            }, 3000);
        }

        // Handle back button
        window.addEventListener('popstate', () => {
            if (document.getElementById('recipeDetail').classList.contains('active')) {
                hideRecipeDetail();
            }
        });

        // Push state when showing recipe
        const originalShowRecipeDetail = showRecipeDetail;
        showRecipeDetail = function(id) {
            history.pushState({ recipe: id }, '');
            originalShowRecipeDetail(id);
        };
    </script>
</body>
</html>`
