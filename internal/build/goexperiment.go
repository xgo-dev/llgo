package build

import (
	"encoding/json"
	"fmt"
	"go/version"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
)

// sourceGoConfig records the toolchain and effective feature tags used to select
// source files. Resolve through the selected Go command: experiment defaults,
// aliases, and dependencies belong to that toolchain, not the LLGo executable.
type sourceGoConfig struct {
	GOROOT       string
	GOVERSION    string
	GOEXPERIMENT string
	GOFLAGS      string
	CGO_ENABLED  string
	CGO_CPPFLAGS string
	CGO_CFLAGS   string
	CGO_CXXFLAGS string
	CGO_LDFLAGS  string
	GOPATH       string
	GOMODCACHE   string
	GOCACHE      string
	// Preserve the actual SDK setting separately from its canonical cache key.
	originalExperiment string
	// True when `go env GOVERSION` reported a non-standard distro/experiment/devel
	// build (for example Fedora's "go1.26.11 X:strictfipsruntime"). In that case
	// setting GOTOOLCHAIN from GOROOT/VERSION ("go1.26.11") would make cmd/go
	// download a different upstream toolchain whose GOROOT does not match
	// sourcePatchGOROOT.
	nonStandardLauncher bool
	toolTags            []string
}

func resolveSourceGoConfig(commands commandEnv, experiment string, buildFlags ...string) (sourceGoConfig, error) {
	if experiment != "" {
		commands.environ = withEnv(commands.environ, "GOEXPERIMENT="+experiment)
	}
	var cfg sourceGoConfig
	cmd := commands.configure(exec.Command("go", "env", "-json",
		"GOROOT", "GOVERSION", "GOEXPERIMENT", "GOFLAGS",
		"CGO_ENABLED", "CGO_CPPFLAGS", "CGO_CFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS",
		"GOPATH", "GOMODCACHE", "GOCACHE"))
	output, err := cmd.Output()
	if err != nil {
		return cfg, sourceGoConfigError("resolve Go source configuration", err)
	}
	if err := json.Unmarshal(output, &cfg); err != nil {
		return cfg, fmt.Errorf("decode Go source configuration: %w", err)
	}
	if cfg.GOROOT == "" || cfg.GOVERSION == "" {
		return cfg, fmt.Errorf("Go source configuration is missing GOROOT or GOVERSION")
	}
	cfg.nonStandardLauncher = !version.IsValid(cfg.GOVERSION)
	cfg.GOVERSION, err = llplan9asm.AssemblySDKVersion(cfg.GOROOT)
	if err != nil {
		return cfg, fmt.Errorf("read selected SDK version: %w", err)
	}
	cfg.originalExperiment = cfg.GOEXPERIMENT
	// An empty environment value makes cmd/go consult GOENV again. A comma
	// means the same default experiment set but pins that choice even if GOENV
	// changes before the next subprocess starts.
	resolvedExperiment := cfg.GOEXPERIMENT
	if resolvedExperiment == "" {
		resolvedExperiment = ","
	}
	commands.environ = withResolvedGoToolchain(commands.environ, cfg.resolvedToolchainVersion())
	commands.environ = withEnv(commands.environ, "GOROOT="+cfg.GOROOT, "GOEXPERIMENT="+resolvedExperiment)
	// Query the resolved compiler while preserving the invocation's module,
	// workspace, and GOFLAGS settings. The launcher on PATH may be a different
	// Go version, especially when module mode disables toolchain switching.
	goExe := "go"
	if runtime.GOOS == "windows" {
		goExe += ".exe"
	}
	args := append([]string{"list", "-f", "{{join context.ToolTags \",\"}}"}, buildFlags...)
	args = append(args, "unsafe")
	cmd = commands.configure(exec.Command(filepath.Join(cfg.GOROOT, "bin", goExe), args...))
	output, err = cmd.Output()
	if err != nil {
		return cfg, sourceGoConfigError("resolve Go tool tags", err)
	}
	// Start with a non-nil slice so an explicitly empty snapshot stays
	// distinct from an unspecified context, which inherits host tool tags.
	cfg.toolTags = append([]string{}, strings.FieldsFunc(strings.TrimSpace(string(output)), func(r rune) bool { return r == ',' })...)
	slices.Sort(cfg.toolTags)
	cfg.toolTags = slices.Compact(cfg.toolTags)
	// Serialize the effective state rather than the user's spelling, so
	// simd,nosimd and nosimd share a cache entry. Starting with none also freezes
	// defaults for later subprocesses using this same resolved toolchain.
	experiments := []string{"none"}
	for _, tag := range cfg.toolTags {
		if name, ok := strings.CutPrefix(tag, "goexperiment."); ok {
			experiments = append(experiments, name)
		}
	}
	cfg.GOEXPERIMENT = strings.Join(experiments, ",")
	return cfg, nil
}

func sourceGoConfigError(action string, err error) error {
	if exit, ok := err.(*exec.ExitError); ok && len(exit.Stderr) != 0 {
		return fmt.Errorf("%s: %s: %w", action, strings.TrimSpace(string(exit.Stderr)), err)
	}
	return fmt.Errorf("%s: %w", action, err)
}

func (cfg sourceGoConfig) releaseVersion() string { return llplan9asm.SDKReleaseVersion(cfg.GOVERSION) }

func (cfg sourceGoConfig) resolvedToolchainVersion() string {
	if cfg.nonStandardLauncher {
		return ""
	}
	return cfg.GOVERSION
}

func (cfg sourceGoConfig) apply(environ []string) []string {
	environ = withEnv(withResolvedGoToolchain(environ, cfg.resolvedToolchainVersion()),
		"GOROOT="+cfg.GOROOT,
		"GOEXPERIMENT="+cfg.GOEXPERIMENT,
		"CGO_ENABLED="+cfg.CGO_ENABLED,
		"CGO_CPPFLAGS="+cfg.CGO_CPPFLAGS,
		"CGO_CFLAGS="+cfg.CGO_CFLAGS,
		"CGO_CXXFLAGS="+cfg.CGO_CXXFLAGS,
		"CGO_LDFLAGS="+cfg.CGO_LDFLAGS,
	)
	for _, setting := range []struct{ key, value string }{
		{"GOPATH", cfg.GOPATH}, {"GOMODCACHE", cfg.GOMODCACHE}, {"GOCACHE", cfg.GOCACHE},
	} {
		if setting.value != "" {
			environ = withEnv(environ, setting.key+"="+setting.value)
		}
	}
	return environ
}
