package build

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// sourceGoConfig records the toolchain and effective feature tags used to select
// source files. Resolve through the selected Go command: experiment defaults,
// aliases, and dependencies belong to that toolchain, not the LLGo executable.
type sourceGoConfig struct {
	GOROOT       string
	GOVERSION    string
	GOEXPERIMENT string
	toolTags     []string
}

func resolveSourceGoConfig(commands commandEnv, experiment string) (sourceGoConfig, error) {
	if experiment != "" {
		commands.environ = withEnv(commands.environ, "GOEXPERIMENT="+experiment)
	}
	var cfg sourceGoConfig
	cmd := commands.configure(exec.Command("go", "env", "-json", "GOROOT", "GOVERSION", "GOEXPERIMENT"))
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
	// An empty environment value makes cmd/go consult GOENV again. A comma
	// means the same default experiment set but pins that choice even if GOENV
	// changes before the next subprocess starts.
	resolvedExperiment := cfg.GOEXPERIMENT
	if resolvedExperiment == "" {
		resolvedExperiment = ","
	}
	commands.environ = withResolvedGoToolchain(commands.environ, cfg.GOVERSION)
	commands.environ = withEnv(commands.environ, "GOROOT="+cfg.GOROOT, "GOEXPERIMENT="+resolvedExperiment)
	cmd = commands.configure(exec.Command("go", "list", "-f", "{{join context.ToolTags \",\"}}", "unsafe"))
	output, err = cmd.Output()
	if err != nil {
		return cfg, sourceGoConfigError("resolve Go tool tags", err)
	}
	// Keep an explicitly empty snapshot distinct from an unspecified context.
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

func (cfg sourceGoConfig) apply(environ []string) []string {
	return withEnv(withResolvedGoToolchain(environ, cfg.GOVERSION),
		"GOROOT="+cfg.GOROOT, "GOEXPERIMENT="+cfg.GOEXPERIMENT)
}
