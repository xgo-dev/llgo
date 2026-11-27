package env

import (
	"runtime/debug"
	"testing"
)

func TestVersionPrefersBuildVersion(t *testing.T) {
	origBuildVersion := buildVersion
	origReadBuildInfo := readBuildInfo
	defer func() { buildVersion = origBuildVersion }()
	defer func() { readBuildInfo = origReadBuildInfo }()

	buildVersion = "v1.2.3"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		t.Fatal("readBuildInfo should not be called when buildVersion is set")
		return nil, false
	}
	if got := Version(); got != "v1.2.3" {
		t.Fatalf("Version() = %q, want %q", got, "v1.2.3")
	}
}

func TestVersionUsesModuleVersion(t *testing.T) {
	origBuildVersion := buildVersion
	origReadBuildInfo := readBuildInfo
	defer func() { buildVersion = origBuildVersion }()
	defer func() { readBuildInfo = origReadBuildInfo }()

	buildVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.4.0"}}, true
	}
	if got := Version(); got != "v0.4.0" {
		t.Fatalf("Version() = %q, want %q", got, "v0.4.0")
	}
}

func TestVersionDirtyFallsBack(t *testing.T) {
	origBuildVersion := buildVersion
	origReadBuildInfo := readBuildInfo
	defer func() { buildVersion = origBuildVersion }()
	defer func() { readBuildInfo = origReadBuildInfo }()

	buildVersion = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.5.0+dirty"}}, true
	}
	if got := Version(); got != devel {
		t.Fatalf("Version() = %q, want %q", got, devel)
	}
}
