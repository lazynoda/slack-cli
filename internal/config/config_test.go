package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg == nil || cfg.Profiles == nil {
		t.Fatalf("Load() returned nil config or profiles")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := &File{}
	cfg.SetToken("default", "  token-value  ")

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := loaded.Profile("default").Token; got != "token-value" {
		t.Fatalf("loaded token = %q, want %q", got, "token-value")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestClearTokenRemovesEmptyProfile(t *testing.T) {
	cfg := &File{}
	cfg.SetToken("default", "token")
	cfg.ClearToken("default")

	if _, ok := cfg.Profiles["default"]; ok {
		t.Fatalf("profile still exists after ClearToken")
	}

	(*File)(nil).ClearToken("default")
	(&File{}).ClearToken("default")
}

func TestResolvePathUsesOverride(t *testing.T) {
	path, err := ResolvePath(" ./config.json ")
	if err != nil {
		t.Fatalf("ResolvePath() error = %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("ResolvePath() = %q, want absolute path", path)
	}
}

func TestLoadInvalidJSONAndHelpers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() invalid JSON error = nil")
	}

	if got := (*File)(nil).Profile("default"); got.Token != "" {
		t.Fatalf("nil Profile() = %#v", got)
	}
	if _, err := expandPath("   "); err == nil {
		t.Fatal("expandPath(empty) error = nil")
	}
	if expanded, err := expandPath("~"); err != nil || !filepath.IsAbs(expanded) {
		t.Fatalf("expandPath(~) = (%q, %v)", expanded, err)
	}
	if resolved, err := ResolvePath(""); err != nil || !filepath.IsAbs(resolved) {
		t.Fatalf("ResolvePath(default) = (%q, %v)", resolved, err)
	}

	parentFile := filepath.Join(t.TempDir(), "file-parent")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile(parentFile) error = %v", err)
	}
	if err := Save(filepath.Join(parentFile, "config.json"), &File{}); err == nil {
		t.Fatal("Save(file parent) error = nil")
	}
}
