// Package ui — folder picker TUI component.
// This file implements a Bubble Tea model that lets the user select
// a folder (or vault) from a list fetched from their secret provider.
// It supports arrow key navigation, type-to-filter search, and displays
// the folder list in a clean, styled layout.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/provider"
)

// maxVisibleFolders is the maximum number of folders shown at once
// before the list starts scrolling. Keeps the UI compact.
const maxVisibleFolders = 12

// FolderPickerModel is the Bubble Tea model for selecting a folder/vault.
// It displays a filterable, scrollable list of folders and lets the user
// pick one with arrow keys + Enter.
type FolderPickerModel struct {
	// allFolders is the complete unfiltered list of folders from the provider.
	allFolders []provider.Folder

	// filtered is the current subset of folders matching the search query.
	filtered []provider.Folder

	// cursor is the index of the currently highlighted folder in the filtered list.
	cursor int

	// offset is the scroll offset for the visible window into the filtered list.
	offset int

	// chosen holds the selected folder after the user presses Enter.
	// It is nil-like (zero value) until a selection is made.
	chosen *provider.Folder

	// cancelled is true if the user pressed Escape or Ctrl+C.
	cancelled bool

	// searchInput is the text input component used for type-to-filter searching.
	searchInput textinput.Model

	// searching is true when the search input is focused and accepting keystrokes.
	searching bool

	// providerName is the name of the provider, shown in the title for context.
	providerName string

	// width is the terminal width for responsive layout.
	width int
}

// NewFolderPicker creates a new folder picker model with the given list of folders.
// The providerName is displayed in the UI title for context (e.g. "Bitwarden", "1Password").
func NewFolderPicker(folders []provider.Folder, providerName string) FolderPickerModel {
	// Set up the search text input with a styled prompt.
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Prompt = "  🔍 "
	ti.CharLimit = 100
	ti.Width = 40

	return FolderPickerModel{
		allFolders:   folders,
		filtered:     folders,
		cursor:       0,
		offset:       0,
		searchInput:  ti,
		searching:    false,
		providerName: providerName,
		width:        60,
	}
}

// Init is the Bubble Tea initialization function. No initial command is needed.
func (m FolderPickerModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input for the folder picker.
// When not searching: arrow keys navigate, "/" or "s" starts search, Enter selects.
// When searching: typing filters the list, Escape exits search mode.
func (m FolderPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle terminal resize events to adjust the layout width.
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		// If the search input is active, route most keys to it.
		if m.searching {
			return m.updateSearching(msg)
		}
		return m.updateNavigating(msg)
	}

	return m, nil
}

// updateNavigating handles key events when the user is browsing the folder list
// (i.e. the search input is NOT focused).
func (m FolderPickerModel) updateNavigating(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {

	// Move cursor up in the list (with wrapping).
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = len(m.filtered) - 1
		}
		m.adjustScroll()

	// Move cursor down in the list (with wrapping).
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}
		m.adjustScroll()

	// Enter search / filter mode.
	case key.Matches(msg, key.NewBinding(key.WithKeys("/", "s"))):
		m.searching = true
		m.searchInput.Focus()
		return m, textinput.Blink

	// Clear the current filter without entering search mode.
	case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
		if m.searchInput.Value() != "" {
			m.searchInput.SetValue("")
			m.applyFilter()
		}

	// Confirm selection — pick the highlighted folder.
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if len(m.filtered) > 0 {
			chosen := m.filtered[m.cursor]
			m.chosen = &chosen
			return m, tea.Quit
		}

	// Cancel and exit.
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q", "ctrl+c"))):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}

// updateSearching handles key events when the search input is focused.
// Typing filters the folder list in real time. Escape or Enter exits search mode.
func (m FolderPickerModel) updateSearching(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {

	// Exit search mode but keep the filter applied.
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.searching = false
		m.searchInput.Blur()
		return m, nil

	// Exit search mode and also confirm the current selection.
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		m.searching = false
		m.searchInput.Blur()
		// If there are results, select the highlighted one.
		if len(m.filtered) > 0 {
			chosen := m.filtered[m.cursor]
			m.chosen = &chosen
			return m, tea.Quit
		}
		return m, nil

	// Navigate while searching — allow up/down even while typing.
	case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.adjustScroll()
		}
		return m, nil

	// All other keys go to the text input for filtering.
	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.applyFilter()
		return m, cmd
	}
}

// applyFilter updates the filtered list based on the current search query.
// It performs a case-insensitive substring match on folder names.
// The cursor and scroll offset are reset to safe positions after filtering.
func (m *FolderPickerModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	if query == "" {
		// No filter — show all folders.
		m.filtered = m.allFolders
	} else {
		// Filter folders whose name contains the query string.
		m.filtered = make([]provider.Folder, 0)
		for _, f := range m.allFolders {
			if strings.Contains(strings.ToLower(f.Name), query) {
				m.filtered = append(m.filtered, f)
			}
		}
	}

	// Reset cursor to stay within bounds after the list changed.
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		} else {
			m.cursor = 0
		}
	}
	m.offset = 0
	m.adjustScroll()
}

// adjustScroll ensures the cursor is within the visible window of the list.
// If the cursor moves above the visible area, scroll up. If below, scroll down.
func (m *FolderPickerModel) adjustScroll() {
	if len(m.filtered) <= maxVisibleFolders {
		m.offset = 0
		return
	}

	// Scroll up if cursor is above the visible window.
	if m.cursor < m.offset {
		m.offset = m.cursor
	}

	// Scroll down if cursor is below the visible window.
	if m.cursor >= m.offset+maxVisibleFolders {
		m.offset = m.cursor - maxVisibleFolders + 1
	}
}

// View renders the folder picker UI with the list, search bar, and help hints.
func (m FolderPickerModel) View() string {
	var b strings.Builder

	// Title with provider context.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(fmt.Sprintf("Select a folder from %s", m.providerName))

	b.WriteString(title)
	b.WriteString("\n")

	// Show the folder count and search input.
	countInfo := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(fmt.Sprintf("  %d folder(s) found", len(m.filtered)))

	b.WriteString(countInfo)
	b.WriteString("\n\n")

	// Show search input if in search mode, or show the current filter if set.
	if m.searching {
		b.WriteString(m.searchInput.View())
		b.WriteString("\n\n")
	} else if m.searchInput.Value() != "" {
		filterLabel := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Render(fmt.Sprintf("  Filter: %q (press / to edit, backspace to clear)", m.searchInput.Value()))
		b.WriteString(filterLabel)
		b.WriteString("\n\n")
	}

	// Render the folder list with scrolling.
	if len(m.filtered) == 0 {
		noResults := lipgloss.NewStyle().
			Foreground(ColorWarning).
			Italic(true).
			PaddingLeft(2).
			Render("No folders match the current filter")
		b.WriteString(noResults)
		b.WriteString("\n")
	} else {
		// Show a scroll-up indicator if we're not at the top.
		if m.offset > 0 {
			scrollUp := lipgloss.NewStyle().
				Foreground(ColorMuted).
				PaddingLeft(4).
				Render(fmt.Sprintf("↑ %d more above", m.offset))
			b.WriteString(scrollUp)
			b.WriteString("\n")
		}

		// Determine the visible window of folders.
		end := m.offset + maxVisibleFolders
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		// Render each visible folder as a list item.
		for i := m.offset; i < end; i++ {
			folder := m.filtered[i]
			isSelected := i == m.cursor

			if isSelected {
				// Selected folder: arrow indicator with bold, colored name.
				name := lipgloss.NewStyle().
					Bold(true).
					Foreground(ColorPrimary).
					Render(folder.Name)
				b.WriteString(fmt.Sprintf("  %s %s", Arrow, name))
			} else {
				// Non-selected folder: dimmer styling with padding to align with the arrow.
				name := lipgloss.NewStyle().
					Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"}).
					Render(folder.Name)
				b.WriteString(fmt.Sprintf("    %s", name))
			}

			b.WriteString("\n")
		}

		// Show a scroll-down indicator if there are more items below.
		remaining := len(m.filtered) - end
		if remaining > 0 {
			scrollDown := lipgloss.NewStyle().
				Foreground(ColorMuted).
				PaddingLeft(4).
				Render(fmt.Sprintf("↓ %d more below", remaining))
			b.WriteString(scrollDown)
			b.WriteString("\n")
		}
	}

	// Help bar at the bottom with contextual key hints.
	b.WriteString("\n")
	if m.searching {
		b.WriteString(helpBar("↑/↓", "navigate", "enter", "select", "esc", "stop search"))
	} else {
		b.WriteString(helpBar("↑/↓", "navigate", "/", "search", "enter", "select", "esc", "cancel"))
	}

	return b.String()
}

// Chosen returns the folder the user selected, or nil if cancelled or nothing was picked.
func (m FolderPickerModel) Chosen() *provider.Folder {
	return m.chosen
}

// Cancelled returns true if the user cancelled the selection.
func (m FolderPickerModel) Cancelled() bool {
	return m.cancelled
}
