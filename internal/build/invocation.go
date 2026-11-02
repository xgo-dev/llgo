package build

import (
	"go/version"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Invocation contains the explicit inputs used by one build. An empty Dir
// uses the working directory captured when Build starts. Build inherits one
// snapshot of the process environment; per-invocation overrides are not
// supported.
type Invocation struct {
	Args   []string
	Config *Config
	Dir    string

	// compileOnly and disableMultiFallback are used by the multi-package
	// failure recovery path. They are intentionally not part of Config: a
	// caller-visible ModeBuild still has normal cmd/go output semantics.
	compileOnly          bool
	disableMultiFallback bool
	// Feature groups preserve the original multi-build output contract while
	// compiling only initials that can safely share package artifacts.
	multipleInitials bool
	initialFeatures  *initialBuildFeatures
	// A child borrows this tracer; only its owning parent may close/write it.
	parentBuildTrace *buildTracer
}

// commandEnv is the per-invocation process-execution state. It deliberately
// contains only the working directory and environment used by child processes.
type commandEnv struct {
	dir     string
	environ []string
}

func (e commandEnv) configure(cmd *exec.Cmd) *exec.Cmd {
	cmd.Dir = e.dir
	cmd.Env = slices.Clone(e.environ)
	return cmd
}

func (e commandEnv) lookup(name string) string {
	for i := len(e.environ) - 1; i >= 0; i-- {
		key, value, ok := strings.Cut(e.environ[i], "=")
		if !ok {
			continue
		}
		if key == name || runtime.GOOS == "windows" && strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

func resolveOutputs(dir string, out *OutFmtDetails) {
	out.WasmManifest = resolvePath(dir, out.WasmManifest)
	out.Out = resolvePath(dir, out.Out)
	out.PCLN = resolvePath(dir, out.PCLN)
	out.DWARF = resolvePath(dir, out.DWARF)
	out.DSYM = resolvePath(dir, out.DSYM)
	out.Bin = resolvePath(dir, out.Bin)
	out.Hex = resolvePath(dir, out.Hex)
	out.Img = resolvePath(dir, out.Img)
	out.Uf2 = resolvePath(dir, out.Uf2)
	out.Zip = resolvePath(dir, out.Zip)
}

func resolvePath(dir, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}

func withEnv(environ []string, values ...string) []string {
	keys := make(map[string]struct{}, len(values))
	for _, value := range values {
		if key, _, ok := strings.Cut(value, "="); ok {
			keys[key] = struct{}{}
		}
	}
	ret := make([]string, 0, len(environ)+len(values))
	for _, value := range environ {
		key, _, ok := strings.Cut(value, "=")
		// Drop malformed passthrough entries: exec.Cmd requires KEY=VALUE.
		if _, replace := keys[key]; ok && replace {
			continue
		}
		if ok {
			ret = append(ret, value)
		}
	}
	return append(ret, values...)
}

func withResolvedGoToolchain(environ []string, goversion string) []string {
	if !version.IsValid(goversion) {
		return environ
	}
	return withEnv(environ, "GOTOOLCHAIN="+goversion)
}
