// Package output renders concise terminal messages shared by commands and hooks.
// Messages go to stderr; stdout remains safe for shell evaluation.
package output

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/v3/internal/config"
	"github.com/s1ks1/bwenv/v3/internal/provider"
)

func Success(message string) { print("✅", "[OK]", "#16A34A", "#4ADE80", message) }
func Warning(message string) { print("⚠️", "[!]", "#CA8A04", "#FACC15", message) }
func Info(message string)    { print("ℹ️", "[i]", "#6B7280", "#9CA3AF", message) }
func Error(label string, err error) {
	message := label
	if err != nil {
		detail := err.Error()
		switch {
		case errors.Is(err, provider.ErrProviderTimeout):
			detail = "Provider timed out · try again"
		case errors.Is(err, provider.ErrSessionExpired), errors.Is(err, provider.ErrNotAuthenticated):
			detail = "Session locked or expired · run bwenv login"
		case errors.Is(err, provider.ErrProviderUnavailable):
			detail = "Provider unavailable · check the CLI and run bwenv login"
		case errors.Is(err, provider.ErrMalformedProviderResponse):
			detail = "Provider returned an invalid response"
		}
		message += ": " + detail
	}
	print("❌", "[X]", "#DC2626", "#F87171", message)
}

func print(emoji, fallback, light, dark, message string) {
	// Keep subprocess errors and user-provided names on one readable line.
	message = strings.Join(strings.Fields(message), " ")
	style := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: light, Dark: dark})
	fmt.Fprintf(os.Stderr, "  %s %s\n", style.Render(config.Emoji(emoji, fallback)), style.Render("bwenv · "+message))
}
