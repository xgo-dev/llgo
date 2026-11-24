package build

import (
	"go/version"
	"os"
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
	if e.environ != nil {
		cmd.Env = slices.Clone(e.environ)
		if len(cmd.Args) > 0 && !strings.ContainsAny(cmd.Args[0], `/\`) {
			if pathVal, ok := e.lookupOK("PATH"); ok {
				if resolved := lookupExecutableInPath(cmd.Args[0], pathVal, e.lookup("PATHEXT")); resolved != "" {
					cmd.Path = resolved
					cmd.Err = nil
				} else {
					cmd.Path = cmd.Args[0]
					cmd.Err = &exec.Error{Name: cmd.Args[0], Err: exec.ErrNotFound}
				}
			}
		}
	}
	return cmd
}

func (e commandEnv) lookPath(name string) (string, error) {
	if e.environ == nil || strings.ContainsAny(name, `/\`) {
		return exec.LookPath(name)
	}
	pathVal, ok := e.lookupOK("PATH")
	if !ok {
		return exec.LookPath(name)
	}
	if resolved := lookupExecutableInPath(name, pathVal, e.lookup("PATHEXT")); resolved != "" {
		return resolved, nil
	}
	return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
}

func (e commandEnv) lookupOK(name string) (string, bool) {
	for i := len(e.environ) - 1; i >= 0; i-- {
		key, value, ok := strings.Cut(e.environ[i], "=")
		if !ok {
			continue
		}
		if key == name || runtime.GOOS == "windows" && strings.EqualFold(key, name) {
			return value, true
		}
	}
	return "", false
}

func (e commandEnv) lookup(name string) string {
	v, _ := e.lookupOK(name)
	return v
}

func lookupExecutableInPath(name, pathEnv, pathExtEnv string) string {
	exts := []string{""}
	if runtime.GOOS == "windows" {
		if pathExtEnv == "" {
			pathExtEnv = ".COM;.EXE;.BAT;.CMD"
		}
		exts = nil
		if filepath.Ext(name) != "" {
			exts = append(exts, "")
		}
		for _, raw := range strings.Split(pathExtEnv, ";") {
			if raw == "" {
				continue
			}
			if raw[0] != '.' {
				raw = "." + raw
			}
			exts = append(exts, raw)
		}
		if len(exts) == 0 {
			exts = []string{""}
		}
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			candidate := filepath.Join(dir, name+ext)
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() {
				continue
			}
			if runtime.GOOS == "windows" || info.Mode()&0o111 != 0 {
				return candidate
			}
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

// withLLVMPath returns a copy of environ whose PATH begins with binDir,
// moving any later duplicate to the front so the LLVM installation selected
// for this invocation takes precedence over ambient PATH entries. It never
// mutates the process environment.
func withLLVMPath(environ []string, binDir string) []string {
	if binDir == "" {
		return environ
	}
	isPathKey := func(k string) bool {
		return k == "PATH" || (runtime.GOOS == "windows" && strings.EqualFold(k, "PATH"))
	}
	sameDir := func(a, b string) bool {
		a, b = filepath.Clean(a), filepath.Clean(b)
		if runtime.GOOS == "windows" {
			return strings.EqualFold(a, b)
		}
		return a == b
	}
	pathKey := "PATH"
	pathVal := ""
	for i := len(environ) - 1; i >= 0; i-- {
		k, v, ok := strings.Cut(environ[i], "=")
		if !ok {
			continue
		}
		if isPathKey(k) {
			pathKey = k
			pathVal = v
			break
		}
	}
	dirs := filepath.SplitList(pathVal)
	if len(dirs) > 0 && sameDir(dirs[0], binDir) {
		return environ
	}
	filtered := make([]string, 0, len(dirs)+1)
	filtered = append(filtered, binDir)
	for _, dir := range dirs {
		if dir != "" && sameDir(dir, binDir) {
			continue
		}
		filtered = append(filtered, dir)
	}
	newPath := strings.Join(filtered, string(os.PathListSeparator))
	ret := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		k, _, ok := strings.Cut(entry, "=")
		if !ok || isPathKey(k) {
			continue
		}
		ret = append(ret, entry)
	}
	return append(ret, pathKey+"="+newPath)
}
