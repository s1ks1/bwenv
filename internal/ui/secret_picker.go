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

const maxVisibleSecrets = 12

type SecretPickerModel struct {
	allItems    []provider.SecretItem
	filtered    []provider.SecretItem
	selected    map[string]bool
	cursor      int
	offset      int
	cancelled   bool
	searchInput textinput.Model
	searching   bool
	width       int
	providerName string
	folderName   string
}

func NewSecretPicker(items []provider.SecretItem, providerName, folderName string) SecretPickerModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Prompt = "  🔍 "
	ti.CharLimit = 100
	ti.Width = 40

	return SecretPickerModel{
		allItems:     items,
		filtered:     items,
		selected:     make(map[string]bool),
		cursor:       0,
		offset:       0,
		searchInput:  ti,
		searching:    false,
		width:        60,
		providerName: providerName,
		folderName:   folderName,
	}
}

func (m SecretPickerModel) Init() tea.Cmd {
	return nil
}

func (m SecretPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		if m.searching {
			return m.updateSearching(msg)
		}
		return m.updateNavigating(msg)
	}

	return m, nil
}

func (m SecretPickerModel) updateNavigating(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {

	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = len(m.filtered) - 1
		}
		m.adjustScroll()

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}
		m.adjustScroll()

	case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
		if len(m.filtered) > 0 {
			item := m.filtered[m.cursor]
			m.selected[item.ID] = !m.selected[item.ID]
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		if len(m.filtered) > 0 {
			allSelected := true
			for _, item := range m.filtered {
				if !m.selected[item.ID] {
					allSelected = false
					break
				}
			}
			for _, item := range m.filtered {
				m.selected[item.ID] = !allSelected
			}
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("/", "s"))):
		m.searching = true
		m.searchInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
		if m.searchInput.Value() != "" {
			m.searchInput.SetValue("")
			m.applyFilter()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if len(m.filtered) > 0 {
			return m, tea.Quit
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q", "ctrl+c"))):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}

func (m SecretPickerModel) updateSearching(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.searching = false
		m.searchInput.Blur()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		m.searching = false
		m.searchInput.Blur()
		if len(m.filtered) > 0 {
			return m, tea.Quit
		}
		return m, nil

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

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.applyFilter()
		return m, cmd
	}
}

func (m *SecretPickerModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	if query == "" {
		m.filtered = m.allItems
	} else {
		m.filtered = make([]provider.SecretItem, 0)
		for _, item := range m.allItems {
			if strings.Contains(strings.ToLower(item.Name), query) {
				m.filtered = append(m.filtered, item)
			}
		}
	}

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

func (m *SecretPickerModel) adjustScroll() {
	if len(m.filtered) <= maxVisibleSecrets {
		m.offset = 0
		return
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	}

	if m.cursor >= m.offset+maxVisibleSecrets {
		m.offset = m.cursor - maxVisibleSecrets + 1
	}
}

func (m SecretPickerModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(fmt.Sprintf("Select items from %s / %s", m.providerName, m.folderName))

	b.WriteString(title)
	b.WriteString("\n")

	count := len(m.selected)
	countStr := "none selected"
	if count > 0 {
		countStr = fmt.Sprintf("%d selected", count)
	}
	countInfo := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(fmt.Sprintf("  %d item(s) found | %s", len(m.filtered), countStr))

	b.WriteString(countInfo)
	b.WriteString("\n\n")

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

	if len(m.filtered) == 0 {
		noResults := lipgloss.NewStyle().
			Foreground(ColorWarning).
			Italic(true).
			PaddingLeft(2).
			Render("No items match the current filter")
		b.WriteString(noResults)
		b.WriteString("\n")
	} else {
		if m.offset > 0 {
			scrollUp := lipgloss.NewStyle().
				Foreground(ColorMuted).
				PaddingLeft(4).
				Render(fmt.Sprintf("↑ %d more above", m.offset))
			b.WriteString(scrollUp)
			b.WriteString("\n")
		}

		end := m.offset + maxVisibleSecrets
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := m.offset; i < end; i++ {
			item := m.filtered[i]
			isCursor := i == m.cursor
			isChecked := m.selected[item.ID]

			checkbox := "[ ]"
			if isChecked {
				checkbox = "[x]"
			}

			if isCursor {
				name := lipgloss.NewStyle().
					Bold(true).
					Foreground(ColorPrimary).
					Render(item.Name)
				b.WriteString(fmt.Sprintf("  %s %s %s", Arrow, checkbox, name))
			} else {
				name := lipgloss.NewStyle().
					Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"}).
					Render(item.Name)
				b.WriteString(fmt.Sprintf("    %s %s", checkbox, name))
			}

			b.WriteString("\n")
		}

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

	b.WriteString("\n")
	if m.searching {
		b.WriteString(helpBar("↑/↓", "navigate", "enter", "select", "esc", "stop search"))
	} else {
		b.WriteString(helpBar("↑/↓", "navigate", "space", "toggle", "a", "all", "/", "search", "enter", "confirm", "esc", "cancel"))
	}

	return b.String()
}

func (m SecretPickerModel) Selected() []provider.SecretItem {
	if m.cancelled {
		return nil
	}
	var selected []provider.SecretItem
	for _, item := range m.allItems {
		if m.selected[item.ID] {
			selected = append(selected, item)
		}
	}
	return selected
}

func (m SecretPickerModel) Cancelled() bool {
	return m.cancelled
}
