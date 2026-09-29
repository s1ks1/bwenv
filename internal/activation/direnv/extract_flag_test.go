package direnv

import "testing"

// TestExtractShellFlag covers extractShellFlag against legacy .envrc lines.
// The escaped-quote case is the PER-44 regression: shell.Quote renders an
// apostrophe as '\” and the old IndexByte scan stopped at the first quote.
func TestExtractShellFlag(t *testing.T) {
	tests := []struct {
		name string
		line string
		flag string
		want string
	}{
		{
			name: "single-quoted value",
			line: `eval "$(bwenv export --provider bitwarden --folder 'MyFolder')"`,
			flag: "--folder",
			want: "MyFolder",
		},
		{
			name: "escaped quotes in folder name",
			line: `eval "$(bwenv export --provider bitwarden --folder 'it'\''s-secret')"`,
			flag: "--folder",
			want: "it's-secret",
		},
		{
			name: "double-quoted value with spaces",
			line: `eval "$(bwenv export --provider 1password --folder "My Folder")"`,
			flag: "--folder",
			want: "My Folder",
		},
		{
			name: "unquoted value",
			line: `eval "$(bwenv export --provider bitwarden --folder MyFolder)"`,
			flag: "--folder",
			want: "MyFolder",
		},
		{
			name: "equals form",
			line: `bwenv export --folder=Plain --provider bitwarden`,
			flag: "--folder",
			want: "Plain",
		},
		{
			name: "quoted value with spaces",
			line: `bwenv export --folder 'with spaces' --provider bitwarden`,
			flag: "--folder",
			want: "with spaces",
		},
		{
			name: "missing value",
			line: `bwenv export --folder`,
			flag: "--folder",
			want: "",
		},
		{
			name: "flag prefix must not match longer flag",
			line: `bwenv export --folder-id 'abc123' --folder 'real'`,
			flag: "--folder",
			want: "real",
		},
		{
			name: "repeated flag uses first occurrence",
			line: `bwenv export --folder 'first' --folder 'second'`,
			flag: "--folder",
			want: "first",
		},
		{
			name: "flag absent",
			line: `bwenv export --provider bitwarden`,
			flag: "--folder",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractShellFlag(tt.line, tt.flag); got != tt.want {
				t.Errorf("extractShellFlag(%q, %q) = %q, want %q", tt.line, tt.flag, got, tt.want)
			}
		})
	}
}
