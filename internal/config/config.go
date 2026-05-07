package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const appName = "slack-cli"

type File struct {
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

type Profile struct {
	Token string `json:"token,omitempty"`
}

func ResolvePath(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return expandPath(override)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}

	return filepath.Join(configDir, appName, "config.json"), nil
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{Profiles: map[string]Profile{}}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}

	return &cfg, nil
}

func Save(path string, cfg *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func (f *File) Profile(name string) Profile {
	if f == nil || f.Profiles == nil {
		return Profile{}
	}
	return f.Profiles[name]
}

func (f *File) SetToken(name, token string) {
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}

	profile := f.Profiles[name]
	profile.Token = strings.TrimSpace(token)
	f.Profiles[name] = profile
}

func (f *File) ClearToken(name string) {
	if f == nil || f.Profiles == nil {
		return
	}

	profile := f.Profiles[name]
	profile.Token = ""
	if profile == (Profile{}) {
		delete(f.Profiles, name)
		return
	}
	f.Profiles[name] = profile
}

func expandPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("config path cannot be empty")
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if trimmed == "~" {
			trimmed = home
		} else {
			trimmed = filepath.Join(home, trimmed[2:])
		}
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}

	return abs, nil
}
