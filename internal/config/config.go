// Package config loads the API key and base URL from the environment or a TOML file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	EnvAPIKey  = "STATUSPANE_API_KEY"
	EnvBaseURL = "STATUSPANE_URL"
)

type Config struct {
	APIKey  string `toml:"api_key"`
	BaseURL string `toml:"base_url"`
}

// DefaultPath follows XDG: $XDG_CONFIG_HOME/statuspane/config.toml, else ~/.config/statuspane/config.toml.
func DefaultPath(getenv func(string) string, userHomeDir func() (string, error)) (string, error) {
	dir := getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := userHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "statuspane", "config.toml"), nil
}

// Load reads the file at path (a missing file is fine), lets the environment override it,
// and fails when the API key or base URL is still unset.
func Load(path string, getenv func(string) string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}
	if v := getenv(EnvAPIKey); v != "" {
		cfg.APIKey = v
	}
	if v := getenv(EnvBaseURL); v != "" {
		cfg.BaseURL = v
	}
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("no API key configured: set %s or api_key in %s", EnvAPIKey, path)
	}
	if cfg.BaseURL == "" {
		return Config{}, fmt.Errorf("no base URL configured: set %s or base_url in %s", EnvBaseURL, path)
	}
	return cfg, nil
}
