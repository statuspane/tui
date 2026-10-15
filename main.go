package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
	"github.com/statuspane/tui/internal/config"
	"github.com/statuspane/tui/internal/ui"
)

// Overridden at release time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath, err := config.DefaultPath(os.Getenv, os.UserHomeDir)
	if err != nil {
		return err
	}
	cfg, err := config.Load(configPath, os.Getenv)
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.BaseURL, cfg.APIKey, version)
	_, err = tea.NewProgram(ui.NewRootModel(client)).Run()
	return err
}
