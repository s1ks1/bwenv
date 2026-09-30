// Package shell provides POSIX shell quoting, environment-variable name
// sanitisation, and shell RC discovery/wrapper installation. It is the single
// home for shell-specific concerns shared by the activation and UI layers.
package shell

import "strings"

// Quote wraps a value in single quotes for safe use in shell export
// statements. Single quotes in the value are escaped using the standard POSIX
// shell trick: end the quoted string, add an escaped single quote, then start
// a new quoted string.
//
// Example: it's -> 'it'\”s'
func Quote(value string) string {
	escaped := strings.ReplaceAll(value, "'", `'\''`)
	return "'" + escaped + "'"
}

// Escape strips shell metacharacters from identifiers such as provider slugs.
// Only alphanumerics, hyphens, underscores and dots are kept.
func Escape(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// SanitizeKey ensures an environment variable name is valid for POSIX shells.
// Non-alphanumeric characters become underscores; a leading digit is prefixed
// with an underscore because env var names cannot start with a digit.
func SanitizeKey(key string) string {
	if key == "" {
		return "_EMPTY_KEY"
	}

	var b strings.Builder
	for i, ch := range key {
		switch {
		case ch >= 'A' && ch <= 'Z':
			b.WriteRune(ch)
		case ch >= 'a' && ch <= 'z':
			b.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			if i == 0 {
				b.WriteRune('_')
			}
			b.WriteRune(ch)
		case ch == '_':
			b.WriteRune(ch)
		default:
			b.WriteRune('_')
		}
	}

	result := b.String()
	if result == "" {
		return "_EMPTY_KEY"
	}
	return result
}
