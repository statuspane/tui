package main

import (
	"runtime/debug"
	"testing"
)

func buildInfoWith(moduleVersion string) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: moduleVersion}}, true
	}
}

func noBuildInfo() (*debug.BuildInfo, bool) { return nil, false }

func TestResolveVersion(t *testing.T) {
	cases := []struct {
		name          string
		linkerVersion string
		readBuildInfo func() (*debug.BuildInfo, bool)
		want          string
	}{
		{"linker version wins", "1.2.3", buildInfoWith("v0.1.0"), "1.2.3"},
		{"installed module version", developmentVersion, buildInfoWith("v0.1.0"), "v0.1.0"},
		{"local build reports dev", developmentVersion, buildInfoWith(localBuildModuleVersion), developmentVersion},
		{"empty module version reports dev", developmentVersion, buildInfoWith(""), developmentVersion},
		{"no build info reports dev", developmentVersion, noBuildInfo, developmentVersion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.linkerVersion, tc.readBuildInfo); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
