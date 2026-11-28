package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	GOFLAGS      string
	CGO_ENABLED  string
	toolTags     []string
}

type cachedSourceGoConfig struct {
	Version      int
	GOROOT       string
	GOVERSION    string
	GOEXPERIMENT string
	GOFLAGS      string
	CGO_ENABLED  string
	ToolTags     []string
}

const sourceGoConfigCacheVersion = 1

func findUpwardFileStat(dir, filename string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	cur := filepath.Clean(dir)
	for {
		cand := filepath.Join(cur, filename)
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return fmt.Sprintf("%s:%d:%d", cand, fi.Size(), fi.ModTime().UnixNano())
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

func lookPathInCommandEnv(commands commandEnv, exe string) (string, bool) {
	if filepath.IsAbs(exe) || strings.ContainsRune(exe, filepath.Separator) {
		if _, err := os.Stat(exe); err == nil {
			return exe, true
		}
		return "", false
	}
	pathVal := ""
	if len(commands.environ) > 0 {
		pathVal = commands.lookup("PATH")
	} else {
		pathVal = os.Getenv("PATH")
	}
	if pathVal == "" {
		return "", false
	}
	names := []string{exe}
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(exe), ".exe") {
		names = []string{exe + ".exe", exe}
	}
	for _, dir := range filepath.SplitList(pathVal) {
		if dir == "" {
			dir = "."
		}
		for _, name := range names {
			cand := filepath.Join(dir, name)
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
				return cand, true
			}
		}
	}
	return "", false
}

func commandEnvLookup(commands commandEnv, name string) string {
	if len(commands.environ) > 0 {
		return commands.lookup(name)
	}
	return os.Getenv(name)
}

func sourceGoConfigCachePath(commands commandEnv, experiment string, buildFlags []string) string {
	if !cacheEnabled() {
		return ""
	}
	goBin, ok := lookPathInCommandEnv(commands, "go")
	if !ok {
		return ""
	}
	fi, err := os.Stat(goBin)
	if err != nil {
		return ""
	}
	h := sha256.New()
	fmt.Fprintf(h, "v=%d\x00gobin=%s:%d:%d\x00exp=%s\x00", sourceGoConfigCacheVersion, goBin, fi.Size(), fi.ModTime().UnixNano(), experiment)
	for _, f := range buildFlags {
		fmt.Fprintf(h, "flag=%s\x00", f)
	}
	for _, name := range []string{
		"GOROOT", "GOEXPERIMENT", "GOFLAGS", "CGO_ENABLED", "GOOS", "GOARCH",
		"GOTOOLCHAIN", "GOWORK", "GOENV", "GOMODCACHE", "GOPATH", "GO111MODULE",
		"GO386", "GOAMD64", "GOARM", "GOARM64", "GOMIPS", "GOMIPS64", "GOPPC64", "GORISCV64", "GOWASM", "GOFIPS140",
	} {
		fmt.Fprintf(h, "%s=%s\x00", name, commandEnvLookup(commands, name))
	}
	recordEffectiveGoMod(h, commands.dir, "", nil)
	recordEffectiveGoWork(h, commands.dir, commands, buildFlags, nil)
	recordEffectiveGoEnv(h, commands, nil)
	return filepath.Join(newCacheManager().root, "meta", "goenv-"+hex.EncodeToString(h.Sum(nil))+".json")
}

func resolveSourceGoConfig(commands commandEnv, experiment string, buildFlags ...string) (sourceGoConfig, error) {
	if experiment != "" {
		commands.environ = withEnv(commands.environ, "GOEXPERIMENT="+experiment)
	}
	cachePath := sourceGoConfigCachePath(commands, experiment, buildFlags)
	if cachePath != "" {
		if raw, err := os.ReadFile(cachePath); err == nil {
			var cached cachedSourceGoConfig
			if json.Unmarshal(raw, &cached) == nil && cached.Version == sourceGoConfigCacheVersion &&
				cached.GOROOT != "" && cached.GOVERSION != "" {
				goExe := "go"
				if runtime.GOOS == "windows" {
					goExe += ".exe"
				}
				if _, err := os.Stat(filepath.Join(cached.GOROOT, "bin", goExe)); err == nil {
					return sourceGoConfig{
						GOROOT:       cached.GOROOT,
						GOVERSION:    cached.GOVERSION,
						GOEXPERIMENT: cached.GOEXPERIMENT,
						GOFLAGS:      cached.GOFLAGS,
						CGO_ENABLED:  cached.CGO_ENABLED,
						toolTags:     cached.ToolTags,
					}, nil
				}
			}
		}
	}
	var cfg sourceGoConfig
	cmd := commands.configure(exec.Command("go", "env", "-json", "GOROOT", "GOVERSION", "GOEXPERIMENT", "GOFLAGS", "CGO_ENABLED"))
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
	if cachePath != "" {
		if raw, err := json.Marshal(cachedSourceGoConfig{
			Version:      sourceGoConfigCacheVersion,
			GOROOT:       cfg.GOROOT,
			GOVERSION:    cfg.GOVERSION,
			GOEXPERIMENT: cfg.GOEXPERIMENT,
			GOFLAGS:      cfg.GOFLAGS,
			CGO_ENABLED:  cfg.CGO_ENABLED,
			ToolTags:     cfg.toolTags,
		}); err == nil {
			_ = writeFileAtomicBytes(cachePath, raw)
		}
	}
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
		"GOROOT="+cfg.GOROOT, "GOEXPERIMENT="+cfg.GOEXPERIMENT, "CGO_ENABLED="+cfg.CGO_ENABLED)
}
