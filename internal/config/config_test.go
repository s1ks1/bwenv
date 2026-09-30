package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateConfig points config resolution at a temp XDG dir and clears the
// package-level cache so each test starts from a clean slate.
func isolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	Invalidate()
	t.Cleanup(Invalidate)
	return filepath.Join(dir, "bwenv", "config.json")
}

func TestLoadReturnsDefaultsWhenNoFile(t *testing.T) {
	isolateConfig(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg != DefaultConfig() {
		t.Fatalf("Load() = %+v, want defaults %+v", cfg, DefaultConfig())
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := isolateConfig(t)
	want := Config{ActivationMode: "shell", ShowEmoji: false, ShowDirenvOutput: true, ShowExportSummary: false}

	if err := Save(want); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Save() did not write %s: %v", path, err)
	}

	Invalidate()
	got, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestLoadRejectsCorruptFile(t *testing.T) {
	path := isolateConfig(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted a corrupt config file")
	}
}

func TestResetRestoresDefaults(t *testing.T) {
	isolateConfig(t)
	if err := Save(Config{ShowEmoji: false}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if err := Reset(); err != nil {
		t.Fatalf("Reset() error: %v", err)
	}

	Invalidate()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() after Reset() error: %v", err)
	}
	if cfg != DefaultConfig() {
		t.Fatalf("after Reset() = %+v, want defaults %+v", cfg, DefaultConfig())
	}
}

func TestEmojiHonoursPreference(t *testing.T) {
	isolateConfig(t)

	if err := Save(Config{ShowEmoji: false}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if got := Emoji("lock", "[lock]"); got != "[lock]" {
		t.Fatalf("Emoji() with ShowEmoji=false = %q, want the fallback", got)
	}

	if err := Save(Config{ShowEmoji: true}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if got := Emoji("lock", "[lock]"); got != "lock" {
		t.Fatalf("Emoji() with ShowEmoji=true = %q, want the emoji", got)
	}
}

func TestLegacyConfigGetsNativeDefaultAndRejectsUnknownMode(t *testing.T) {
	path := isolateConfig(t)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"show_emoji":false}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg.ActivationMode != "shell" || cfg.ShowEmoji {
		t.Fatalf("legacy preferences: %+v, %v", cfg, err)
	}
	for _, mode := range []string{"shell", "direnv", "mise"} {
		cfg.ActivationMode = mode
		if err := Save(cfg); err != nil {
			t.Fatal(err)
		}
		Invalidate()
		got, err := Load()
		if err != nil || got.ActivationMode != mode {
			t.Fatalf("saved mode %s: %+v, %v", mode, got, err)
		}
	}
	cfg.ActivationMode = "unknown"
	if err := Save(cfg); err == nil {
		t.Fatal("unknown hook accepted")
	}
}
