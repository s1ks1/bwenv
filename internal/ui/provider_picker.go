// Package ui — provider picker TUI component.
// This file implements a Bubble Tea model that lets the user select
// a secret provider (Bitwarden, 1Password, etc.) from a list using
// arrow keys. It shows each provider's name, description, and whether
// its CLI tool is currently installed.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/provider"
)

// ProviderPickerModel is the Bubble Tea model for selecting a secret provider.
// It displays a vertical list of available providers with descriptions
// and lets the user navigate with arrow keys and confirm with Enter.
type ProviderPickerModel struct {
	// providers is the list of all registered providers to display.
	providers []provider.Provider

	// cursor tracks the currently highlighted index in the list.
	cursor int

	// chosen holds the selected provider after the user presses Enter.
	// It is nil until a selection is made.
	chosen provider.Provider

	// cancelled is true if the user pressed Escape or Ctrl+C.
	cancelled bool

	// width is the terminal width, used for responsive layout.
	width int
}

// NewProviderPicker creates a new provider picker model.
// It takes the full list of registered providers (both available and unavailable)
// so the user can see what's supported even if not yet installed.
func NewProviderPicker(providers []provider.Provider) ProviderPickerModel {
	return ProviderPickerModel{
		providers: providers,
		cursor:    0,
		width:     60,
	}
}

// Init is the Bubble Tea initialization function. No initial command is needed.
func (m ProviderPickerModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input for the provider picker.
// Supported keys: up/down/j/k to navigate, Enter to select, Esc/q/Ctrl+C to cancel.
func (m ProviderPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle terminal resize events to adjust the layout width.
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch {

		// Move cursor up in the list (wrap around to the bottom).
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.providers) - 1
			}

		// Move cursor down in the list (wrap around to the top).
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(m.providers)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}

		// Confirm selection — only if the provider's CLI is available.
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			selected := m.providers[m.cursor]
			if selected.IsAvailable() {
				m.chosen = selected
				return m, tea.Quit
			}
			// If not available, do nothing (the user sees the "not installed" label).

		// Cancel and exit.
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q", "ctrl+c"))):
			m.cancelled = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the provider picker UI. It shows a title, a list of providers
// with the current selection highlighted, and a help bar at the bottom.
func (m ProviderPickerModel) View() string {
	var b strings.Builder

	// Title section.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1).
		Render("Select a secret provider")

	b.WriteString(title)
	b.WriteString("\n\n")

	// Render each provider as a list item.
	for i, p := range m.providers {
		isSelected := i == m.cursor
		isAvailable := p.IsAvailable()

		// Build the provider line with name and availability indicator.
		var line string
		if isSelected {
			// Selected item: show arrow indicator and bold name.
			name := lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Render(p.Name())

			if isAvailable {
				line = fmt.Sprintf("  %s %s  %s", Arrow, name, availableBadge())
			} else {
				line = fmt.Sprintf("  %s %s  %s", Arrow, name, unavailableBadge(p.CLICommand()))
			}

			// Show description below the selected item.
			desc := lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Italic(true).
				PaddingLeft(6).
				Render(p.Description())

			b.WriteString(line)
			b.WriteString("\n")
			b.WriteString(desc)
		} else {
			// Non-selected item: dimmer styling.
			name := lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"}).
				Render(p.Name())

			if isAvailable {
				line = fmt.Sprintf("    %s  %s", name, availableBadge())
			} else {
				line = fmt.Sprintf("    %s  %s", name, unavailableBadge(p.CLICommand()))
			}

			b.WriteString(line)
		}

		b.WriteString("\n")

		// Add some vertical spacing between items.
		if i < len(m.providers)-1 {
			b.WriteString("\n")
		}
	}

	// Help bar at the bottom.
	b.WriteString("\n")
	b.WriteString(helpBar("↑/↓", "navigate", "enter", "select", "esc", "cancel"))

	return b.String()
}

// Chosen returns the provider the user selected, or nil if cancelled.
func (m ProviderPickerModel) Chosen() provider.Provider {
	return m.chosen
}

// Cancelled returns true if the user cancelled the selection.
func (m ProviderPickerModel) Cancelled() bool {
	return m.cancelled
}

// availableBadge returns a small green "installed" tag.
func availableBadge() string {
	return lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Render("✓ installed")
}

// unavailableBadge returns a small red "not installed" tag with the CLI command name.
func unavailableBadge(cliCmd string) string {
	return lipgloss.NewStyle().
		Foreground(ColorError).
		Render(fmt.Sprintf("✗ '%s' not found", cliCmd))
}

// helpBar formats a row of key binding hints for the bottom of the TUI.
// It accepts alternating key/description pairs, e.g. helpBar("↑/↓", "navigate", "enter", "select").
func helpBar(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		k := HelpKey.Render(pairs[i])
		v := HelpValue.Render(pairs[i+1])
		parts = append(parts, fmt.Sprintf("%s %s", k, v))
	}
	return lipgloss.NewStyle().
		Foreground(ColorMuted).
		PaddingLeft(2).
		Render(strings.Join(parts, "  •  "))
}
