package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func tempConfigFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadReadsFile(t *testing.T) {
	path := tempConfigFile(t, "api_key = \"sp_file\"\nbase_url = \"https://file.example\"\n")

	cfg, err := Load(path, env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sp_file" || cfg.BaseURL != "https://file.example" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	path := tempConfigFile(t, "api_key = \"sp_file\"\nbase_url = \"https://file.example\"\n")

	cfg, err := Load(path, env(map[string]string{EnvAPIKey: "sp_env", EnvBaseURL: "https://env.example"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sp_env" || cfg.BaseURL != "https://env.example" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestLoadWorksWithoutFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.toml")

	cfg, err := Load(missing, env(map[string]string{EnvAPIKey: "sp_env", EnvBaseURL: "https://env.example"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sp_env" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestLoadRequiresKeyAndURL(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.toml")

	_, err := Load(missing, env(map[string]string{EnvBaseURL: "https://env.example"}))
	if err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("want API key error, got %v", err)
	}

	_, err = Load(missing, env(map[string]string{EnvAPIKey: "sp_env"}))
	if err == nil || !strings.Contains(err.Error(), "base URL") {
		t.Fatalf("want base URL error, got %v", err)
	}
}

func TestLoadRejectsMalformedFile(t *testing.T) {
	path := tempConfigFile(t, "api_key = ")

	if _, err := Load(path, env(nil)); err == nil {
		t.Fatal("want parse error")
	}
}

func TestDefaultPathPrefersXDGConfigHome(t *testing.T) {
	homeDir := func() (string, error) { return "/home/me", nil }

	path, err := DefaultPath(env(map[string]string{"XDG_CONFIG_HOME": "/xdg"}), homeDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/xdg", "statuspane", "config.toml"); path != want {
		t.Fatalf("got %q, want %q", path, want)
	}
}

func TestDefaultPathFallsBackToHomeConfigDir(t *testing.T) {
	homeDir := func() (string, error) { return "/home/me", nil }

	path, err := DefaultPath(env(nil), homeDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/home/me", ".config", "statuspane", "config.toml"); path != want {
		t.Fatalf("got %q, want %q", path, want)
	}
}

func TestDefaultPathReportsMissingHomeDir(t *testing.T) {
	homeErr := errors.New("no home")

	_, err := DefaultPath(env(nil), func() (string, error) { return "", homeErr })

	if !errors.Is(err, homeErr) {
		t.Fatalf("got %v", err)
	}
}
