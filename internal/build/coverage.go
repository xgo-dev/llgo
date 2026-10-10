// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package build

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/packages"
)

// CoverageConfig describes Go's source coverage for builds and tests. A non-nil
// configuration enables coverage; Mode defaults to set. Profile and OutputDir
// are test-only; application binaries use GOCOVERDIR at execution time.
type CoverageConfig struct {
	Mode      string
	Packages  string
	Profile   string
	OutputDir string
}

type coverageBuild struct {
	options  CoverageConfig
	dir      string
	commands commandEnv
	// inputOverlay is the package-loader overlay before coverage adds generated
	// sources or replaces instrumented files. Isolated initial groups must start
	// from this snapshot and instrument their freshly loaded graphs themselves.
	inputOverlay    map[string][]byte
	inputOverlaySet bool
	manifest        []byte
	mergeMu         sync.Mutex
	goCommand       string
	noTests         []*packages.Package
	metaPaths       map[string]string
	trace           *buildTracer
	go120           bool
	local           bool
}

// These wire structs mirror cmd/internal/cov/covcmd in the selected GOROOT.
// Use that toolchain's cover executable and runtime together; never mix them
// with the Go version used to compile the llgo command itself.
type coverPkgConfig struct {
	OutConfig    string
	PkgPath      string
	PkgName      string
	Granularity  string
	ModulePath   string
	Local        bool
	EmitMetaFile string
}

type coverFixupConfig struct {
	MetaVar            string
	MetaLen            int
	MetaHash           string
	Strategy           string
	CounterPrefix      string
	PkgIdVar           string
	CounterMode        string
	CounterGranularity string
}

type coverMetaFiles struct {
	ImportPaths       []string
	MetaFileFragments []string
}

func newCoverageBuild(conf *Config, commands commandEnv) (*coverageBuild, error) {
	if conf.Coverage == nil {
		return nil, nil
	}
	options := *conf.Coverage
	if options.Mode == "" {
		options.Mode = "set"
	}
	switch options.Mode {
	case "set", "count", "atomic":
	default:
		return nil, fmt.Errorf("invalid value %q for flag -covermode: valid modes are \"set\", \"count\", or \"atomic\"", options.Mode)
	}
	if conf.Mode != ModeTest && conf.Mode != ModeBuild {
		return nil, fmt.Errorf("coverage requires llgo build or llgo test")
	}
	if conf.Mode == ModeBuild && options.Profile != "" {
		return nil, fmt.Errorf("-coverprofile requires llgo test; use GOCOVERDIR for covered applications")
	}
	if conf.BuildMode != "" && conf.BuildMode != BuildModeExe {
		return nil, fmt.Errorf("coverage is not yet supported with -buildmode=%s", conf.BuildMode)
	}
	if conf.Target != "" || isWasmTarget(conf.Goos) {
		return nil, fmt.Errorf("coverage is not yet supported on this target")
	}
	for _, arg := range conf.RunArgs {
		if arg == "--" {
			break
		}
		name, _, _ := strings.Cut(arg, "=")
		switch name {
		case "-test.gocoverdir", "--test.gocoverdir", "-test.coverprofile", "--test.coverprofile":
			if conf.Mode == ModeTest && !conf.CompileOnly {
				// Each test process needs its own metadata and profile fragment.
				// Reject overrides before truncating the user's merged profile.
				return nil, fmt.Errorf("%s is reserved for llgo test coverage; use -coverprofile to select the output profile", name)
			}
		case "-test.fuzz", "--test.fuzz":
			if options.Profile != "" {
				return nil, fmt.Errorf("cannot use -coverprofile flag with -fuzz flag")
			}
		}
	}
	if options.Profile != "" && !conf.CompileOnly {
		outputDir := commands.dir
		if options.OutputDir != "" {
			outputDir = resolvePath(commands.dir, options.OutputDir)
		}
		options.Profile = resolvePath(outputDir, options.Profile)
		// An initial-group parent prepares the destination before its child
		// invocations rebuild isolated package graphs. Do not let each child
		// truncate fragments already appended by an earlier group.
		if !conf.coverageProfileInitialized {
			if err := os.WriteFile(options.Profile, []byte("mode: "+options.Mode+"\n"), 0666); err != nil {
				return nil, err
			}
			conf.coverageProfileInitialized = true
		}
	}
	dir, err := os.MkdirTemp("", "llgo-cover-")
	if err != nil {
		return nil, err
	}
	return &coverageBuild{
		options:  options,
		dir:      dir,
		commands: commands,
	}, nil
}

func (c *coverageBuild) close() {
	os.RemoveAll(c.dir)
}

func (c *coverageBuild) prepare(
	roots []*packages.Package,
	cfg *packages.Config,
	conf *Config,
	goroot string,
) error {
	c.inputOverlay = cloneOverlay(conf.Overlay)
	c.inputOverlaySet = true
	c.goCommand = filepath.Join(goroot, "bin", "go")
	// Pattern expansion must see the same target and module environment as the
	// package loader, not just the host environment used by native build tools.
	c.commands.environ = withEnv(cfg.Env, "GOROOT="+goroot, "GOTOOLCHAIN=local")
	version, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return err
	}
	c.go120 = strings.HasPrefix(string(version), "go1.20")
	byID := make(map[string]*packages.Package)
	var all []*packages.Package
	packages.Visit(roots, func(p *packages.Package) bool {
		byID[p.ID] = p
		all = append(all, p)
		return true
	}, nil)
	selected := make(map[string]bool)
	tested := make(map[string]bool)
	rootPaths := make(map[string]bool)
	for _, p := range roots {
		if strings.HasSuffix(p.ID, ".test") {
			tested[strings.TrimSuffix(p.PkgPath, ".test")] = true
		} else if p.ForTest == "" {
			rootPaths[p.PkgPath] = true
		}
	}
	for _, p := range roots {
		if conf.Mode == ModeTest && p.ForTest == "" && rootPaths[p.PkgPath] && !tested[p.PkgPath] {
			c.noTests = append(c.noTests, p)
		}
	}
	if c.options.Packages == "" {
		for _, p := range roots {
			if p.ForTest == "" && !strings.HasSuffix(p.ID, ".test") {
				selected[p.PkgPath] = true
			}
		}
		if conf.Mode == ModeBuild {
			// go build covers command-line roots and their main-module
			// dependencies, unlike go test's default of just the tested roots.
			for _, p := range all {
				if p.Module != nil && p.Module.Main {
					selected[p.PkgPath] = true
				}
			}
		}
	} else {
		for _, pattern := range strings.Split(c.options.Packages, ",") {
			if pattern == "all" {
				// cmd/go's coverage matcher means the entire loaded graph,
				// including command-line-arguments and test-only dependencies,
				// not the different package set produced by "go list all".
				for _, p := range all {
					selected[p.PkgPath] = true
				}
				continue
			}
			args := append([]string{"list", "-e", "-f={{.ImportPath}}"}, cfg.BuildFlags...)
			cmd := c.commands.configure(exec.Command(c.goCommand, append(args, pattern)...))
			cmd.Dir = cfg.Dir
			output, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("resolve -coverpkg %q: %w", pattern, err)
			}
			matches := make(map[string]bool)
			for _, path := range strings.Fields(string(output)) {
				matches[path] = true
			}
			matched := false
			for _, p := range all {
				// go list pattern expansion intentionally skips testdata, dot,
				// and underscore directories. -coverpkg instead matches the
				// already-loaded dependency graph, so apply the same package
				// pattern directly as well as retaining go list for special
				// patterns such as std, cmd, tool, and work.
				if matches[p.PkgPath] || matchLoadedCoveragePackage(pattern, cfg.Dir, p) {
					selected[p.PkgPath], matched = true, true
				}
			}
			if !matched {
				action := "tested"
				if conf.Mode == ModeBuild {
					action = "built"
				}
				fmt.Fprintf(os.Stderr, "warning: no packages being %s depend on matches for pattern %s\n", action, pattern)
			}
		}
	}
	delete(selected, "unsafe")
	if c.options.Mode == "atomic" {
		delete(selected, "sync/atomic")
		delete(selected, "internal/runtime/atomic")
		delete(selected, "runtime/internal/atomic")
	}

	// Load only metadata for the imports introduced by coverage, then reconnect
	// to the existing package identities before the one shared type-check pass.
	supportCfg := *cfg
	supportCfg.Tests = false
	support, err := packages.LoadMetadata(&supportCfg, "runtime/coverage", "sync/atomic", "unsafe")
	if err != nil {
		return err
	}
	var canonical func(*packages.Package) *packages.Package
	canonical = func(p *packages.Package) *packages.Package {
		if prev := byID[p.ID]; prev != nil {
			return prev
		}
		byID[p.ID] = p
		for name, dep := range p.Imports {
			p.Imports[name] = canonical(dep)
		}
		return p
	}
	for _, p := range support {
		canonical(p)
	}
	if byID["runtime/coverage"] == nil {
		return fmt.Errorf("selected Go toolchain has no runtime/coverage")
	}
	if conf.Overlay == nil {
		conf.Overlay = make(map[string][]byte)
	}
	ids, err := coveragePackageIDs(goroot)
	if err != nil {
		return err
	}
	var manifest coverMetaFiles
	metaPaths := make(map[string]string)
	c.metaPaths = metaPaths
	var targets []*packages.Package
	for _, p := range all {
		if !selected[p.PkgPath] || strings.HasSuffix(p.ID, ".test") {
			continue
		}
		p.Imports["unsafe"] = byID["unsafe"]
		if p.Name == "main" {
			// go tool cover adds this import to instrumented main packages.
			p.Imports["runtime/coverage"] = byID["runtime/coverage"]
		}
		if c.options.Mode == "atomic" {
			p.Imports["sync/atomic"] = byID["sync/atomic"]
		}
		targets = append(targets, p)
	}
	// Source instrumentation must finish before the shared type-check pass.
	// Use the same -p budget as SSA/backend/run work; worker-local overlays
	// avoid publishing partially instrumented source into the shared graph.
	// The base overlay stays read-only until all workers finish. Each worker
	// returns only its changes, without copying every dependency's patches.
	indexes := make([]int, len(targets))
	outputs := make([]map[string][]byte, len(targets))
	metadata := make([]string, len(targets))
	for i := range indexes {
		indexes[i] = i
	}
	err = runBoundedPackageJobs(conf.parallelism(), indexes, func(i int) error {
		p := targets[i]
		span := c.trace.startWorker("coverage", p.PkgPath)
		defer span.done()
		local := *conf
		local.Overlay = make(map[string][]byte)
		meta, err := c.instrument(p, &local, cfg, goroot, ids[p.PkgPath], conf.Overlay)
		if err != nil {
			return fmt.Errorf("cover %s: %w", p.ID, err)
		}
		outputs[i] = local.Overlay
		metadata[i] = meta
		return nil
	})
	if err != nil {
		return err
	}
	for i, p := range targets {
		maps.Copy(conf.Overlay, outputs[i])
		if meta := metadata[i]; meta != "" && metaPaths[p.PkgPath] == "" {
			metaPaths[p.PkgPath] = meta
		}
	}
	paths := make([]string, 0, len(metaPaths))
	for path := range metaPaths {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		// cmd/go's auxiliary metadata barrier contains tested roots. Other
		// selected dependencies contribute when linked into that test binary.
		if rootPaths[path] {
			manifest.ImportPaths = append(manifest.ImportPaths, path)
			manifest.MetaFileFragments = append(manifest.MetaFileFragments, metaPaths[path])
		}
	}
	if c.options.Packages != "" {
		c.manifest, err = json.Marshal(manifest)
		if err != nil {
			return err
		}
	}
	for _, p := range roots {
		isTest := conf.Mode == ModeTest && strings.HasSuffix(p.ID, ".test")
		if !isTest && (conf.Mode != ModeBuild || p.Name != "main") {
			continue
		}
		covered := ""
		selectedPaths := paths
		if c.options.Packages != "" {
			covered = " in " + c.options.Packages
		} else {
			selectedPaths = []string{strings.TrimSuffix(p.PkgPath, ".test")}
		}
		var source string
		if cfile := byID["internal/coverage/cfile"]; cfile != nil {
			p.Imports["internal/coverage/cfile"] = cfile
			p.Imports["internal/runtime/exithook"] = byID["internal/runtime/exithook"]
			p.Imports["runtime"] = byID["runtime"]
			p.Imports["unsafe"] = byID["unsafe"]
			if isTest {
				source = fmt.Sprintf(coverageTestMain, c.options.Mode, covered, selectedPaths)
			} else {
				source = coverageBuildMain
			}
		} else {
			// Go 1.20–1.22 keep reporting in runtime/coverage and register
			// it directly with testing instead of through testdeps.
			p.Imports["runtime/coverage"] = byID["runtime/coverage"]
			p.Imports["unsafe"] = byID["unsafe"]
			if isTest {
				source = legacyCoverageMain(c.options.Mode, covered, c.go120)
			} else {
				source = coverageBuildMainLegacy + coverageLegacyExitSupport
			}
		}
		body := []byte(source + coverageExitSupport)
		// A package replacement also replaces its init import list. Retain
		// counter-only packages from the original graph as explicit main
		// dependencies, without changing the replacement's ordinary imports.
		var counterPackages []*packages.Package
		packages.Visit([]*packages.Package{p}, func(dep *packages.Package) bool {
			if strings.HasPrefix(dep.PkgPath, "llgo.coverage/") {
				counterPackages = append(counterPackages, dep)
			}
			return true
		}, nil)
		for _, dep := range counterPackages {
			p.Imports[dep.PkgPath] = dep
			body, err = addCoverageImport(body, "_", dep.PkgPath)
			if err != nil {
				return err
			}
		}
		addCoverageSource(p, conf, "main", body)
	}
	cfg.Overlay = conf.Overlay
	return nil
}
