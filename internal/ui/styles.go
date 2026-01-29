package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	primaryColor   = lipgloss.Color("#FF6B6B")
	secondaryColor = lipgloss.Color("#4ECDC4")
	accentColor    = lipgloss.Color("#FFE66D")
	textColor      = lipgloss.Color("#FAFAFA")
	mutedColor     = lipgloss.Color("#888888")
	bgColor        = lipgloss.Color("#2D3436")
	successColor   = lipgloss.Color("#00B894")
	errorColor     = lipgloss.Color("#E74C3C")
)

// Styles for the application
var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	// Subtitle style
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			MarginBottom(1)

	// List item styles
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				PaddingLeft(2)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(textColor).
			PaddingLeft(2)

	// Recipe details styles
	RecipeTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				MarginBottom(1)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				Underline(true).
				MarginTop(1).
				MarginBottom(1)

	IngredientStyle = lipgloss.NewStyle().
			Foreground(textColor).
			PaddingLeft(2)

	InstructionStyle = lipgloss.NewStyle().
				Foreground(textColor).
				PaddingLeft(2).
				MarginBottom(1)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(0, 1)

	InputLabelStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Help style
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(errorColor)

	// Success style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(1, 2)

	// Tab styles
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Underline(true).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 2)

	// Image placeholder
	ImagePlaceholderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(mutedColor).
				Padding(2, 4).
				Align(lipgloss.Center)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Meta info style
	MetaStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Highlighted text
	HighlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 2).
			MarginRight(1)

	ButtonSelectedStyle = lipgloss.NewStyle().
				Foreground(bgColor).
				Background(accentColor).
				Bold(true).
				Padding(0, 2).
				MarginRight(1)
)

// GetWidth returns the width for centered content
func GetWidth(width int) int {
	if width > 120 {
		return 120
	}
	return width
}

// Truncate truncates a string to the given length
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
