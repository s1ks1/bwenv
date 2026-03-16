// Package ui — config flow for interactive settings management.
// This file implements the "bwenv config" command which lets users
// toggle preferences like emoji display, direnv output visibility,
// export summary display, and auto-sync behavior.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/config"
)

// configOption represents a single toggleable setting in the config TUI.
type configOption struct {
	key         string // Internal key for identification.
	label       string // Display label shown to the user.
	description string // Help text explaining what this setting does.
	enabled     bool   // Current value of the setting.
}

// ConfigFlowModel is the Bubble Tea model for the interactive config editor.
// It displays a list of toggleable settings with descriptions and lets the
// user flip them on/off with Enter or Space.
type ConfigFlowModel struct {
	options   []configOption
	cursor    int
	saved     bool
	cancelled bool
	width     int
	err       error
}

// NewConfigFlow creates a new config flow model, pre-populated with the
// current settings loaded from disk (or defaults if no config exists).
func NewConfigFlow(cfg config.Config) ConfigFlowModel {
	options := []configOption{
		{
			key:         "show_emoji",
			label:       "Show Emoji",
			description: "Display emoji icons in output (disable for cleaner text-only output)",
			enabled:     cfg.ShowEmoji,
		},
		{
			key:         "show_direnv_output",
			label:       "Show Direnv Output",
			description: "Show direnv loading/unloading messages (hidden by default for cleaner output)",
			enabled:     cfg.ShowDirenvOutput,
		},
		{
			key:         "show_export_summary",
			label:       "Show Export Summary",
			description: "Show the boxed summary every time secrets are loaded via direnv",
			enabled:     cfg.ShowExportSummary,
		},
		{
			key:         "auto_sync",
			label:       "Auto Sync",
			description: "Automatically sync the vault before fetching secrets (Bitwarden)",
			enabled:     cfg.AutoSync,
		},
	}

	return ConfigFlowModel{
		options: options,
		cursor:  0,
		width:   60,
	}
}

// Init is the Bubble Tea initialization function. No initial command is needed.
func (m ConfigFlowModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input for the config flow.
func (m ConfigFlowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch {
		// Navigate up.
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.options) - 1
			}

		// Navigate down.
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}

		// Toggle the current option.
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			if m.cursor >= 0 && m.cursor < len(m.options) {
				m.options[m.cursor].enabled = !m.options[m.cursor].enabled
			}

		// Save and exit.
		case key.Matches(msg, key.NewBinding(key.WithKeys("s", "ctrl+s"))):
			m.saved = true
			return m, tea.Quit

		// Cancel and exit without saving.
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q", "ctrl+c"))):
			m.cancelled = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the config editor UI.
func (m ConfigFlowModel) View() string {
	var b strings.Builder

	// Title.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("bwenv Settings")

	b.WriteString(title)
	b.WriteString("\n")

	subtitle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		Render("  Toggle options with Enter or Space, save with S")

	b.WriteString(subtitle)
	b.WriteString("\n\n")

	// Render each option.
	for i, opt := range m.options {
		isSelected := i == m.cursor

		// Toggle indicator.
		var toggle string
		if opt.enabled {
			toggle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true).
				Render("[ON] ")
		} else {
			toggle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true).
				Render("[OFF]")
		}

		if isSelected {
			// Selected option — arrow indicator, bold name.
			name := lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Render(opt.label)

			b.WriteString(fmt.Sprintf("  %s %s  %s\n", Arrow, toggle, name))

			// Show description for the selected item.
			desc := lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true).
				PaddingLeft(6).
				Render(opt.description)
			b.WriteString(fmt.Sprintf("      %s\n", desc))
		} else {
			// Non-selected option — dimmer styling.
			name := lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"}).
				Render(opt.label)

			b.WriteString(fmt.Sprintf("    %s  %s\n", toggle, name))
		}

		// Spacing between items.
		if i < len(m.options)-1 {
			b.WriteString("\n")
		}
	}

	// Help bar.
	b.WriteString("\n\n")
	b.WriteString(helpBar("↑/↓", "navigate", "enter", "toggle", "s", "save", "esc", "cancel"))

	return b.String()
}

// Saved returns true if the user chose to save settings.
func (m ConfigFlowModel) Saved() bool {
	return m.saved
}

// Cancelled returns true if the user cancelled without saving.
func (m ConfigFlowModel) Cancelled() bool {
	return m.cancelled
}

// ToConfig converts the current option states back into a Config struct.
func (m ConfigFlowModel) ToConfig() config.Config {
	cfg := config.DefaultConfig()
	for _, opt := range m.options {
		switch opt.key {
		case "show_emoji":
			cfg.ShowEmoji = opt.enabled
		case "show_direnv_output":
			cfg.ShowDirenvOutput = opt.enabled
		case "show_export_summary":
			cfg.ShowExportSummary = opt.enabled
		case "auto_sync":
			cfg.AutoSync = opt.enabled
		}
	}
	return cfg
}

// RunConfigFlow executes the interactive config editor and saves the result.
// This is the main entry point for the "bwenv config" command.
func RunConfigFlow(version string) error {
	PrintBanner(version)

	// Load current config (or defaults).
	cfg, err := config.Load()
	if err != nil {
		PrintWarning(fmt.Sprintf("Could not load config: %v (using defaults)", err))
		cfg = config.DefaultConfig()
	}

	// Show config file location.
	if path, pathErr := config.ConfigPath(); pathErr == nil {
		PrintInfo(fmt.Sprintf("Config file: %s", path))
	}
	fmt.Println()

	// Launch the interactive TUI.
	model := NewConfigFlow(cfg)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("config editor failed: %w", err)
	}

	result := finalModel.(ConfigFlowModel)

	if result.Cancelled() {
		fmt.Println()
		PrintWarning("Cancelled — no changes saved")
		return nil
	}

	if result.Saved() {
		newCfg := result.ToConfig()
		if err := config.Save(newCfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println()
		PrintSuccess("Settings saved")
		fmt.Println()

		// Show a summary of what was set.
		printConfigSummary(newCfg)

		return nil
	}

	// If neither saved nor cancelled (shouldn't happen), treat as cancel.
	fmt.Println()
	PrintWarning("No changes saved")
	return nil
}

// printConfigSummary displays the current configuration values in a compact format.
func printConfigSummary(cfg config.Config) {
	PrintKeyValue("Show Emoji", OnOff(cfg.ShowEmoji))
	PrintKeyValue("Direnv Output", OnOff(cfg.ShowDirenvOutput))
	PrintKeyValue("Export Summary", OnOff(cfg.ShowExportSummary))
	PrintKeyValue("Auto Sync", OnOff(cfg.AutoSync))
	fmt.Println()

	hint := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		Render("  Settings take effect on next bwenv invocation.")
	fmt.Println(hint)
	fmt.Println()
}
