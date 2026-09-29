package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"API_KEY", "API_KEY"},
		{"db-host", "db_host"},
		{"123start", "_123start"},
		{"my.var.name", "my_var_name"},
		{"Hello World!", "Hello_World_"},
		{"", "_EMPTY_KEY"},
		{"___", "___"},
		{"a", "a"},
		{"a-b-c", "a_b_c"},
	}

	for _, tt := range tests {
		if result := SanitizeKey(tt.input); result != tt.expected {
			t.Errorf("SanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"it's", "'it'\\''s'"},
		{"simple text", "'simple text'"},
		{"", "''"},
		{"price is $5", "'price is $5'"},
	}

	for _, tt := range tests {
		if result := Quote(tt.input); result != tt.expected {
			t.Errorf("Quote(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bitwarden", "bitwarden"},
		{"1password", "1password"},
		{"my-provider", "my-provider"},
		{"hello world!", "helloworld"},
		{"test.provider_v2", "test.provider_v2"},
		{"", ""},
	}

	for _, tt := range tests {
		if result := Escape(tt.input); result != tt.expected {
			t.Errorf("Escape(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestShortenHomePath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{filepath.Join(home, ".zshrc"), "~/.zshrc"},
		{home, "~"},
		{"/tmp/somewhere", "/tmp/somewhere"},
		{"/nonexistent/path", "/nonexistent/path"},
	}

	for _, tt := range tests {
		if result := ShortenHomePath(tt.input); result != tt.expected {
			t.Errorf("ShortenHomePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
