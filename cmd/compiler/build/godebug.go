package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

// The selected Go compiler owns GODEBUG policy, including version defaults,
// workspace directives, selected source files, and generated test mains.
// Keep its result per entry instead of baking application policy into a
// cached runtime package shared by multiple executables.
func resolveDefaultGODEBUG(cfg *packages.Config, goroot string, roots []*packages.Package, patterns []string) (map[string]string, error) {
	entries := make(map[string]string)
	var entryPaths []string
	localArguments := false
	for _, pkg := range roots {
		if pkg.Name == "main" {
			entries[pkg.ID] = ""
			path := pkg.ID
			if cfg.Tests {
				// Generated .test IDs are not importable packages. Listing
				// their source package with -test recreates the same entry.
				path = strings.TrimSuffix(path, ".test")
			}
			localArguments = localArguments || path == "command-line-arguments" || strings.HasPrefix(path, "_/")
			entryPaths = append(entryPaths, path)
		}
	}
	if len(entries) == 0 {
		return entries, nil
	}
	if !localArguments {
		// Resolve only known entries, avoiding a second wildcard expansion.
		// File arguments and GOPATH-local packages retain invocation patterns:
		// their synthetic IDs cannot be passed back as import paths.
		slices.Sort(entryPaths)
		patterns = slices.Compact(entryPaths)
	}
	goExe := "go"
	if runtime.GOOS == "windows" {
		goExe += ".exe"
	}
	args := []string{"list", "-e", "-json"}
	if cfg.Tests {
		args = append(args, "-test")
	}
	args = append(args, cfg.BuildFlags...)
	args = append(args, "--")
	args = append(args, patterns...)
	// Match the package driver's logical working directory. Inheriting the
	// parent's PWD can make Go canonicalize symlinked paths differently and
	// miss absolute overlay keys (notably /var and /private/var on macOS).
	commands := commandEnv{dir: cfg.Dir, environ: withEnv(cfg.Env, "PWD="+cfg.Dir)}
	cmd := commands.configure(exec.Command(filepath.Join(goroot, "bin", goExe), args...))
	output, err := cmd.Output()
	if err != nil {
		return nil, sourceGoConfigError("resolve default GODEBUG", err)
	}
	dec := json.NewDecoder(bytes.NewReader(output))
	for {
		var pkg struct {
			ImportPath     string
			DefaultGODEBUG string
			Error          *struct{ Err string }
		}
		if err := dec.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode default GODEBUG: %w", err)
		}
		if _, ok := entries[pkg.ImportPath]; !ok {
			continue
		}
		if pkg.Error != nil {
			return nil, fmt.Errorf("default GODEBUG for %s: %s", pkg.ImportPath, pkg.Error.Err)
		}
		// Go versions before 1.21 omit the field, yielding an empty default.
		entries[pkg.ImportPath] = pkg.DefaultGODEBUG
	}
	return entries, nil
}
