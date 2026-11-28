package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

type cachedDefaultGODEBUG struct {
	Version int
	Entries map[string]string
}

const defaultGODEBUGCacheVersion = 1

func defaultGODEBUGCachePath(cfg *packages.Config, goroot string, roots []*packages.Package, patterns []string) string {
	if !cacheEnabled() || cfg == nil || goroot == "" {
		return ""
	}
	h := sha256.New()
	fmt.Fprintf(h, "v=%d\x00goroot=%s\x00dir=%s\x00tests=%v\x00", defaultGODEBUGCacheVersion, goroot, cfg.Dir, cfg.Tests)
	for _, flag := range cfg.BuildFlags {
		fmt.Fprintf(h, "flag=%s\x00", flag)
	}
	for _, pat := range patterns {
		fmt.Fprintf(h, "pat=%s\x00", pat)
	}
	cmdEnv := commandEnv{dir: cfg.Dir, environ: cfg.Env}
	for _, name := range []string{"GOOS", "GOARCH", "GOEXPERIMENT", "GOFLAGS", "GOWORK", "GOENV", "GO111MODULE", "GOTOOLCHAIN", "LLGO_TEST_GODEBUG_METADATA"} {
		fmt.Fprintf(h, "%s=%s\x00", name, commandEnvLookup(cmdEnv, name))
	}
	fmt.Fprintf(h, "dirmod=%s\x00", findUpwardFileStat(cfg.Dir, "go.mod"))
	recordEffectiveGoWork(h, cfg.Dir, cmdEnv, cfg.BuildFlags, cfg.Overlay)
	// Extract exact //go:debug directives from already-parsed entry syntax so
	// ordinary code edits hit the cache while //go:debug or go.mod edits miss.
	mainRoots := make([]*packages.Package, 0, len(roots))
	for _, pkg := range roots {
		if pkg != nil && pkg.Name == "main" {
			mainRoots = append(mainRoots, pkg)
		}
	}
	slices.SortFunc(mainRoots, func(a, b *packages.Package) int { return strings.Compare(a.ID, b.ID) })
	for _, pkg := range mainRoots {
		if len(pkg.Syntax) == 0 && len(pkg.CompiledGoFiles) > 0 {
			return ""
		}
		fmt.Fprintf(h, "pkg=%s\x00pdir=%s\x00", pkg.ID, pkg.Dir)
		if pkg.Module != nil {
			fmt.Fprintf(h, "modver=%s\x00", pkg.Module.GoVersion)
			if pkg.Module.GoMod != "" {
				if data, ok := cfg.Overlay[pkg.Module.GoMod]; ok {
					fmt.Fprintf(h, "modov=%s:%s\x00", pkg.Module.GoMod, digestBytes(data))
				} else if fi, err := os.Stat(pkg.Module.GoMod); err == nil {
					fmt.Fprintf(h, "modst=%s:%d:%d\x00", pkg.Module.GoMod, fi.Size(), fi.ModTime().UnixNano())
				}
			}
		}
		recordEffectiveGoWork(h, pkg.Dir, cmdEnv, cfg.BuildFlags, cfg.Overlay)
		var debugDirectives []string
		for _, file := range pkg.Syntax {
			for _, cg := range file.Comments {
				for _, c := range cg.List {
					if strings.HasPrefix(c.Text, "//go:debug") {
						debugDirectives = append(debugDirectives, c.Text)
					}
				}
			}
		}
		slices.Sort(debugDirectives)
		for _, d := range debugDirectives {
			fmt.Fprintf(h, "godebug=%s\x00", d)
		}
	}
	return filepath.Join(newCacheManager().root, "meta", "godebug-"+hex.EncodeToString(h.Sum(nil))+".json")
}

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
	cachePath := defaultGODEBUGCachePath(cfg, goroot, roots, patterns)
	if cachePath != "" {
		if raw, err := os.ReadFile(cachePath); err == nil {
			var cached cachedDefaultGODEBUG
			if json.Unmarshal(raw, &cached) == nil && cached.Version == defaultGODEBUGCacheVersion && len(cached.Entries) == len(entries) {
				matched := true
				for id := range entries {
					if _, ok := cached.Entries[id]; !ok {
						matched = false
						break
					}
				}
				if matched {
					return cached.Entries, nil
				}
			}
		}
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
	if cachePath != "" {
		if raw, err := json.Marshal(cachedDefaultGODEBUG{
			Version: defaultGODEBUGCacheVersion,
			Entries: entries,
		}); err == nil {
			_ = writeFileAtomicBytes(cachePath, raw)
		}
	}
	return entries, nil
}
