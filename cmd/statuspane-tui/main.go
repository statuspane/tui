package main

import (
	"fmt"
	"os"
	"runtime/debug"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
	"github.com/statuspane/tui/internal/config"
	"github.com/statuspane/tui/internal/ui"
)

const (
	developmentVersion = "dev"

	// What runtime/debug reports as the main module's version for a build that isn't a tagged module.
	localBuildModuleVersion = "(devel)"
)

// Set with -ldflags "-X main.version=..." when a build needs to name itself explicitly.
var version = developmentVersion

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

	client := api.NewClient(cfg.BaseURL, cfg.APIKey, resolveVersion(version, debug.ReadBuildInfo))
	_, err = tea.NewProgram(ui.NewRootModel(client)).Run()
	return err
}

// resolveVersion prefers a version set with -ldflags, then the module version Go embeds when
// installed with `go install ...@vX.Y.Z`; a local build has neither and stays "dev".
func resolveVersion(linkerVersion string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if linkerVersion != developmentVersion {
		return linkerVersion
	}
	if buildInfo, ok := readBuildInfo(); ok && buildInfo.Main.Version != "" && buildInfo.Main.Version != localBuildModuleVersion {
		return buildInfo.Main.Version
	}
	return developmentVersion
}
