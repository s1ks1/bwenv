// Package ui — shared output helpers.
// These functions provide styled terminal output used across all commands.
// They wrap Lipgloss styles so the rest of the codebase doesn't need to
// import lipgloss directly for simple status messages.
//
// Every user-facing print function lives here so the styling stays consistent.
// If you need a new kind of output, add it here rather than using fmt.Print
// with inline lipgloss calls scattered across the codebase.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/config"
)

// E returns the emoji string if ShowEmoji is enabled in the user config,
// otherwise returns the plain-text fallback. This is the single point of
// control for all emoji display throughout the application.
//
// Usage: E("🔐", "[lock]") → "🔐" or "[lock]" depending on config.
func E(emoji string, fallback string) string {
	return config.Emoji(emoji, fallback)
}

// ── Banner ─────────────────────────────────────────────────────────────────

// PrintBanner displays the application header with version info.
// This is shown at the top of the help output and during init/test flows.
func PrintBanner(version string) {
	// Build the banner text with the lock icon and app name.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(E("🔐", "[*]") + " bwenv")

	// Strip any existing "v" prefix to avoid "vv2.0.0" when version is already "v2.0.0".
	cleanVersion := strings.TrimPrefix(version, "v")
	ver := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("v" + cleanVersion)

	desc := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Italic(true).
		Render("Sync secrets from your password manager into your shell")

	// Compose the banner content.
	content := fmt.Sprintf("%s %s\n%s", title, ver, desc)
	banner := Banner.Render(content)

	fmt.Println(banner)
}

// ── Single-line status messages ────────────────────────────────────────────

// PrintSuccess prints a success message with a green checkmark prefix.
// Use this for operations that completed without errors.
func PrintSuccess(message string) {
	fmt.Printf("  %s %s\n", CheckMark, SuccessText.Render(message))
}

// PrintError prints an error message with a red cross prefix and detail line.
// The label provides context about what failed, and err gives the details.
func PrintError(label string, err error) {
	header := ErrorText.Render(E("✗", "X") + " " + label)
	detail := lipgloss.NewStyle().Foreground(ColorMuted).Render(err.Error())
	fmt.Fprintf(os.Stderr, "\n  %s\n    %s\n\n", header, detail)
}

// PrintWarning prints a warning message with an amber indicator prefix.
// Use this for non-fatal issues the user should be aware of.
func PrintWarning(message string) {
	fmt.Printf("  %s %s\n", WarningMark, WarningText.Render(message))
}

// PrintInfo prints an informational message with a blue dot prefix.
// Use this for neutral status updates and hints.
func PrintInfo(message string) {
	fmt.Printf("  %s %s\n", InfoMark, lipgloss.NewStyle().Foreground(ColorMuted).Render(message))
}

// ── Step progress ──────────────────────────────────────────────────────────

// PrintStep prints a numbered step in a multi-step process.
// Example output: "  [2/6] 🔓 Authenticating with Bitwarden..."
func PrintStep(step int, total int, message string) {
	num := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(fmt.Sprintf("[%d/%d]", step, total))

	fmt.Printf("  %s %s\n", num, message)
}

// ── Key-value and diagnostic lines ─────────────────────────────────────────

// PrintKeyValue prints a label-value pair with aligned formatting.
// The label is rendered in the primary color and padded to a fixed width
// so that multiple key-value lines align nicely.
func PrintKeyValue(label string, value string) {
	styledLabel := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Width(20).
		Render(label)

	fmt.Printf("  %s %s\n", styledLabel, value)
}

// PrintStatusLine prints a single diagnostic status line with a checkmark or cross.
// This is used in the "bwenv test" output for pass/fail checks.
func PrintStatusLine(ok bool, label string, detail string) {
	var mark string
	if ok {
		mark = CheckMark
	} else {
		mark = CrossMark
	}

	line := fmt.Sprintf("  %s %s", mark, label)
	if detail != "" {
		line += lipgloss.NewStyle().Foreground(ColorMuted).Render("  " + detail)
	}
	fmt.Println(line)
}

// PrintWarningLine prints a diagnostic line with a warning (amber) indicator.
// Used in "bwenv test" when something is not critical but noteworthy.
func PrintWarningLine(label string, detail string) {
	line := fmt.Sprintf("  %s %s", WarningMark, WarningText.Render(label))
	if detail != "" {
		line += lipgloss.NewStyle().Foreground(ColorMuted).Render("  " + detail)
	}
	fmt.Println(line)
}

// PrintInfoLine prints a diagnostic line with a blue info indicator.
// Used in "bwenv test" for neutral or optional information.
func PrintInfoLine(label string, detail string) {
	line := fmt.Sprintf("  %s %s", InfoMark, label)
	if detail != "" {
		line += lipgloss.NewStyle().Foreground(ColorMuted).Render("  " + detail)
	}
	fmt.Println(line)
}

// ── Section headers ────────────────────────────────────────────────────────

// PrintSection prints a section header with an emoji, bold label, and divider.
// Use this to visually separate groups of output lines in diagnostic reports.
func PrintSection(title string) {
	fmt.Println()
	styled := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(title)
	fmt.Printf("  %s\n", styled)
	fmt.Printf("  %s\n", Divider(40))
}

// ── Boxes ──────────────────────────────────────────────────────────────────

// PrintBoxSuccess prints a success message inside a green bordered box.
// Use this for major milestones like completing the init flow.
func PrintBoxSuccess(lines ...string) {
	content := strings.Join(lines, "\n")
	fmt.Println(SuccessBox.Render(content))
}

// PrintBoxError prints an error message inside a red bordered box.
// Use this for critical failures that need user attention.
func PrintBoxError(lines ...string) {
	content := strings.Join(lines, "\n")
	fmt.Fprintln(os.Stderr, ErrorBox.Render(content))
}

// PrintBoxWarning prints a warning message inside an amber bordered box.
// Use this for important non-fatal issues.
func PrintBoxWarning(lines ...string) {
	content := strings.Join(lines, "\n")
	fmt.Println(WarningBox.Render(content))
}

// ── Formatting helpers ─────────────────────────────────────────────────────

// FormatSecretHidden returns a placeholder string for a hidden secret value.
// Used when displaying secrets in non-debug mode.
func FormatSecretHidden() string {
	return lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		Render("••••••••")
}

// FormatProviderTag returns a styled tag for a provider name (e.g. "[bitwarden]").
// This is used in .envrc headers and status messages.
func FormatProviderTag(slug string) string {
	return lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true).
		Render("[" + slug + "]")
}

// ShortenHomePath replaces the user's home directory prefix with "~"
// for more compact and readable display in status messages.
// This is the single shared implementation — use this instead of local copies.
func ShortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// OnOff returns a styled "ON" or "OFF" string for boolean config values.
func OnOff(val bool) string {
	if val {
		return lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("ON")
	}
	return lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("OFF")
}
