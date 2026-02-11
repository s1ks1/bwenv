// Package ui provides all terminal user interface components for bwenv.
// This file defines the shared Lipgloss styles used across the application
// for consistent, beautiful terminal output on all platforms.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// -- Color palette --
// These colors are chosen to look good on both light and dark terminals.
// We use adaptive colors where possible so the UI adapts to the terminal theme.

var (
	// Primary brand color — a pleasant blue/cyan.
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#58A6FF"}

	// Secondary accent — a warm purple.
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#6B21A8", Dark: "#C084FC"}

	// Success green for positive status messages.
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"}

	// Warning amber for cautionary messages.
	ColorWarning = lipgloss.AdaptiveColor{Light: "#CA8A04", Dark: "#FACC15"}

	// Error red for failure messages.
	ColorError = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}

	// Muted color for less important text (hints, descriptions).
	ColorMuted = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}

	// Subtle color for borders and dividers.
	ColorSubtle = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#4B5563"}
)

// -- Text styles --

var (
	// Bold applies bold weight to text.
	Bold = lipgloss.NewStyle().Bold(true)

	// Title is the main heading style — bold, primary color, with bottom margin.
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1)

	// Subtitle is for secondary headings — slightly muted, italic.
	Subtitle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)

	// Description is for help text and explanations — muted color.
	Description = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// SuccessText renders text in the success (green) color.
	SuccessText = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// WarningText renders text in the warning (amber) color.
	WarningText = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// ErrorText renders text in the error (red) color with bold weight.
	ErrorText = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)
)

// -- Status indicators --
// These are prefixed icons for status messages (checkmarks, crosses, etc.).

var (
	// CheckMark is a styled green checkmark for success indicators.
	CheckMark = SuccessText.Render("✓")

	// CrossMark is a styled red cross for failure indicators.
	CrossMark = ErrorText.Render("✗")

	// WarningMark is a styled amber warning indicator.
	WarningMark = WarningText.Render("!")

	// InfoMark is a styled blue info indicator.
	InfoMark = lipgloss.NewStyle().Foreground(ColorPrimary).Render("●")

	// Arrow is a styled indicator for the currently selected item.
	Arrow = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▸")
)

// -- Box and container styles --

var (
	// Banner is the large header box shown at the top of the app.
	Banner = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSubtle).
		Padding(0, 2).
		MarginBottom(1)

	// StatusBox wraps diagnostic and status output in a bordered container.
	StatusBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	// SuccessBox is a box styled for success messages — green border.
	SuccessBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Foreground(ColorSuccess).
			Padding(0, 2).
			MarginTop(1).
			MarginBottom(1)

	// ErrorBox is a box styled for error messages — red border.
	ErrorBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Foreground(ColorError).
			Padding(0, 2).
			MarginTop(1).
			MarginBottom(1)

	// WarningBox is a box styled for warning messages — amber border.
	WarningBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Foreground(ColorWarning).
			Padding(0, 2).
			MarginTop(1).
			MarginBottom(1)
)

// -- List / selection styles --

var (
	// SelectedItem is the style for the currently highlighted item in a list.
	SelectedItem = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			PaddingLeft(2)

	// UnselectedItem is the style for non-highlighted items in a list.
	UnselectedItem = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"}).
			PaddingLeft(4)

	// SelectedItemDescription shows the description of a selected item.
	SelectedItemDescription = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				PaddingLeft(2)

	// UnselectedItemDescription shows the description of a non-selected item.
	UnselectedItemDescription = lipgloss.NewStyle().
					Foreground(ColorMuted).
					PaddingLeft(4)
)

// -- Input styles --

var (
	// Prompt is the style for input prompts (the label before the cursor).
	Prompt = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	// Cursor is the style for the text cursor in input fields.
	Cursor = lipgloss.NewStyle().
		Foreground(ColorSecondary)

	// HelpKey shows the key binding in the help bar at the bottom.
	HelpKey = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Bold(true)

	// HelpValue shows the key description in the help bar at the bottom.
	HelpValue = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// -- Divider --

// Divider returns a horizontal line of the given width for visual separation.
func Divider(width int) string {
	if width <= 0 {
		width = 50
	}
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return lipgloss.NewStyle().Foreground(ColorSubtle).Render(line)
}
