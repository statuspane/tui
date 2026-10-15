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
	EnvAPIKey  = "STATUSPANE_API_KEY" //nolint:gosec // the name of an environment variable, not a credential
	EnvBaseURL = "STATUSPANE_URL"

	DefaultBaseURL = "https://statuspane.io"
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
// and fails when the API key is still unset. The base URL defaults to DefaultBaseURL.
func Load(path string, getenv func(string) string) (Config, error) {
	cfg := Config{BaseURL: DefaultBaseURL}
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}
	if apiKey := getenv(EnvAPIKey); apiKey != "" {
		cfg.APIKey = apiKey
	}
	if baseURL := getenv(EnvBaseURL); baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if cfg.APIKey == "" {
		return Config{}, missingAPIKeyError(path)
	}
	return cfg, nil
}

// missingAPIKeyError shows a complete example, since a first run usually lacks the whole file.
func missingAPIKeyError(path string) error {
	return fmt.Errorf(`no API key configured

Set %s, or create %s containing:

  api_key = "sp_..."   # an organization API key`, EnvAPIKey, path)
}
