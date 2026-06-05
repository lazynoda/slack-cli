package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestSetAndClearCookie(t *testing.T) {
	cfg := &File{}
	cfg.SetCookie("default", "  d=cookie-val  ")

	if got := cfg.Profile("default").Cookie; got != "d=cookie-val" {
		t.Fatalf("profile cookie = %q, want %q", got, "d=cookie-val")
	}

	cfg.ClearCookie("default")
	if _, ok := cfg.Profiles["default"]; ok {
		t.Fatalf("profile still exists after ClearCookie")
	}

	(*File)(nil).ClearCookie("default")
	(&File{}).ClearCookie("default")
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

func TestResolvePathNoConfigDir(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("APPDATA", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	// On some OSes os.UserConfigDir might still succeed or fail.
	// We just check if it runs without panic.
	_, _ = ResolvePath("")
}

func TestLoadReadError(t *testing.T) {
	// Try loading a directory, which should cause a read error.
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error reading directory as config file")
	}
}

func TestLoadNullProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"profiles": null}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Profiles == nil {
		t.Fatal("expected cfg.Profiles to be initialized to non-nil map")
	}
}

func TestExpandPathTildeWithSubdir(t *testing.T) {
	expanded, err := expandPath("~/some/subdir")
	if err != nil {
		t.Fatalf("expandPath(~/...) error = %v", err)
	}
	if !filepath.IsAbs(expanded) {
		t.Fatalf("expandPath(~/...) = %q, want absolute path", expanded)
	}
	if !strings.Contains(expanded, filepath.Join("some", "subdir")) {
		t.Fatalf("expandPath(~/...) = %q, want path to contain 'some/subdir'", expanded)
	}
}

func TestExpandPathNoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	_, err := expandPath("~/foo")
	if err == nil {
		t.Fatal("expected error expanding tilde when HOME/USERPROFILE is empty")
	}
}
