package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/s1ks1/bwenv/v3/internal/config"
)

func TestConfigCyclesActivationWithoutChangingOtherPreferences(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ShowEmoji = false
	m := NewConfigFlow(cfg)
	for _, want := range []string{"direnv", "mise", "shell"} {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(ConfigFlowModel)
		got := m.ToConfig()
		if got.ActivationMode != want || got.ShowEmoji != cfg.ShowEmoji || got.ShowExportSummary != cfg.ShowExportSummary {
			t.Fatalf("config = %+v; expected hook %s and preserved preferences", got, want)
		}
	}
}
