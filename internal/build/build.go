/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/go/ssa"

	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/cabi"
	"github.com/goplus/llgo/internal/clang"
	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/internal/firmware"
	"github.com/goplus/llgo/internal/flash"
	"github.com/goplus/llgo/internal/goembed"
	"github.com/goplus/llgo/internal/header"
	"github.com/goplus/llgo/internal/mockable"
	"github.com/goplus/llgo/internal/monitor"
	"github.com/goplus/llgo/internal/optlevel"
	"github.com/goplus/llgo/internal/packages"
	"github.com/goplus/llgo/internal/typepatch"
	"github.com/goplus/llgo/ssa/abi"
	xenv "github.com/goplus/llgo/xtool/env"
	"github.com/goplus/llgo/xtool/env/llvm"
	gllvm "github.com/xgo-dev/llvm"

	llruntime "github.com/goplus/llgo/runtime"
	llssa "github.com/goplus/llgo/ssa"
)

type Mode int

const (
	ModeBuild Mode = iota
	ModeInstall
	ModeRun
	ModeTest
	ModeCmpTest
	ModeGen
)

type BuildMode string

const (
	BuildModeExe      BuildMode = "exe"
	BuildModeCArchive BuildMode = "c-archive"
	BuildModeCShared  BuildMode = "c-shared"
)

// ValidateBuildMode checks if the build mode is valid
func ValidateBuildMode(mode string) error {
	switch BuildMode(mode) {
	case BuildModeExe, BuildModeCArchive, BuildModeCShared:
		return nil
	default:
		return fmt.Errorf("invalid build mode %q, must be one of: exe, c-archive, c-shared", mode)
	}
}

type AbiMode = cabi.Mode

const (
	debugBuild = packages.DebugPackagesLoad
)

// OutFmts contains output format specifications for embedded targets
type OutFmts struct {
	Bin bool // Generate binary output (.bin)
	Hex bool // Generate Intel hex output (.hex)
	Img bool // Generate image output (.img)
	Uf2 bool // Generate UF2 output (.uf2)
	Zip bool // Generate ZIP/DFU output (.zip)
}

// OutFmtDetails contains detailed output file paths for each format
type OutFmtDetails struct {
	Out string // Base output file path
	Bin string // Binary output file path (.bin)
	Hex string // Intel hex output file path (.hex)
	Img string // Image output file path (.img)
	Uf2 string // UF2 output file path (.uf2)
	Zip string // ZIP/DFU output file path (.zip)
}

// ModuleHook observes a package module immediately after it is generated and
// before TransformModule mutates it. The callback runs synchronously and
// receives the live llvm.Module, so callers that need a stable snapshot should
// consume it immediately (for example, by calling mod.String() inside the
// hook).
type ModuleHook func(pkg Package)

type Config struct {
	Goos          string
	Goarch        string
	Target        string // target name (e.g., "rp2040", "wasi") - takes precedence over Goos/Goarch
	OptLevel      optlevel.Level
	BinPath       string
	AppExt        string  // ".exe" on Windows, empty on Unix
	OutFile       string  // only valid for ModeBuild when len(pkgs) == 1
	OutFmts       OutFmts // Output format specifications (only for Target != "")
	CompileOnly   bool    // compile test binary but do not run it (only valid for ModeTest)
	Emulator      bool    // run in emulator mode
	Port          string  // target port for flashing
	BaudRate      int     // baudrate for serial communication
	RunArgs       []string
	Mode          Mode
	BuildMode     BuildMode // Build mode: exe, c-archive, c-shared
	AbiMode       AbiMode
	GenExpect     bool // only valid for ModeCmpTest
	Verbose       bool
	PrintCommands bool
	GenLL         bool // generate pkg .ll files
	CheckLLFiles  bool // check .ll files valid
	CheckLinkArgs bool // check linkargs valid
	ForceEspClang bool // force to use esp-clang
	ForceRebuild  bool // force rebuilding of packages that are already up-to-date
	Tags          string
	SizeReport    bool   // print size report after successful build
	SizeFormat    string // size report format: text,json (default text)
	SizeLevel     string // size aggregation level: full,module,package (default module)
	CompilerHash  string // metadata hash for the running compiler (development builds only)
	// GlobalRewrites specifies compile-time overrides for global string variables.
	// Keys are fully qualified package paths (e.g. "main" or "github.com/user/pkg").
	// Each Rewrites entry maps variable names to replacement string values. Only
	// string-typed globals are supported and "main" applies to all root main
	// packages in the current build.
	GlobalRewrites map[string]Rewrites
	ModuleHook     ModuleHook
}

type Rewrites map[string]string

func NewDefaultConf(mode Mode) *Config {
	bin := os.Getenv("GOBIN")
	if bin == "" {
		gopath, err := envGOPATH()
		if err != nil {
			panic(fmt.Errorf("cannot get GOPATH: %v", err))
		}
		bin = filepath.Join(gopath, "bin")
	}
	if err := os.MkdirAll(bin, 0755); err != nil {
		panic(fmt.Errorf("cannot create bin directory: %v", err))
	}
	goos, goarch := os.Getenv("GOOS"), os.Getenv("GOARCH")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	conf := &Config{
		Goos:      goos,
		Goarch:    goarch,
		BinPath:   bin,
		Mode:      mode,
		BuildMode: BuildModeExe,
		AbiMode:   cabi.ModeAllFunc,
	}
	return conf
}

func envGOPATH() (string, error) {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "go"), nil
}

func prependEnvPath(dir string) {
	if dir == "" {
		return
	}
	sep := string(os.PathListSeparator)
	pathEnv := os.Getenv("PATH")
	parts := filepath.SplitList(pathEnv)
	if len(parts) > 0 && parts[0] == dir {
		return
	}
	out := make([]string, 0, len(parts)+1)
	out = append(out, dir)
	for _, part := range parts {
		if part != "" && part != dir {
			out = append(out, part)
		}
	}
	os.Setenv("PATH", strings.Join(out, sep))
}

// -----------------------------------------------------------------------------

const (
	loadFiles   = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles
	loadImports = loadFiles | packages.NeedImports
	loadTypes   = loadImports | packages.NeedTypes | packages.NeedTypesSizes
	loadSyntax  = loadTypes | packages.NeedSyntax | packages.NeedTypesInfo
)

func Do(args []string, conf *Config) ([]Package, error) {
	if conf.Goos == "" {
		conf.Goos = runtime.GOOS
	}
	if conf.Goarch == "" {
		conf.Goarch = runtime.GOARCH
	}
	if conf.AppExt == "" {
		conf.AppExt = defaultAppExt(conf)
	}
	if conf.BuildMode == "" {
		conf.BuildMode = BuildModeExe
	}
	if conf.SizeReport && conf.SizeFormat == "" {
		conf.SizeFormat = "text"
	}
	if conf.SizeReport && conf.SizeLevel == "" {
		conf.SizeLevel = "module"
	}
	if err := ensureSizeReporting(conf); err != nil {
		return nil, err
	}
	conf.OptLevel = effectiveOptLevel(conf)
	// Handle crosscompile configuration first to set correct GOOS/GOARCH
	forceEspClang := conf.ForceEspClang || conf.Target != ""
	export, err := crosscompile.Use(conf.Goos, conf.Goarch, conf.Target, IsWasiThreadsEnabled(), forceEspClang, conf.OptLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to setup crosscompile: %w", err)
	}

	// Update GOOS/GOARCH from export if target was used
	if conf.Target != "" && export.GOOS != "" {
		conf.Goos = export.GOOS
	}
	if conf.Target != "" && export.GOARCH != "" {
		conf.Goarch = export.GOARCH
	}

	// Enable different export names for TinyGo compatibility when using -target
	if conf.Target != "" {
		cl.EnableExportRename(true)
	}

	verbose := conf.Verbose
	patterns := args
	tags := "llgo,math_big_pure_go,purego"
	if conf.AbiMode == cabi.ModeAllFunc {
		tags += ",llgo_abi_2"
	}
	if conf.Tags != "" {
		tags += "," + conf.Tags
	}
	if len(export.BuildTags) > 0 {
		tags += "," + strings.Join(export.BuildTags, ",")
	}
	cfg := &packages.Config{
		Mode:       loadSyntax | packages.NeedDeps | packages.NeedModule | packages.NeedExportFile,
		BuildFlags: []string{"-tags=" + tags},
		Fset:       token.NewFileSet(),
		Tests:      conf.Mode == ModeTest,
		Env:        append(slices.Clone(os.Environ()), "GOOS="+conf.Goos, "GOARCH="+conf.Goarch),
		ParseFile:  parseBuildFile,
	}
	if conf.Mode == ModeTest {
		cfg.Mode |= packages.NeedForTest
	}

	cl.EnableDebug(IsDbgEnabled())
	cl.EnableDbgSyms(IsDbgSymsEnabled())
	cl.EnableTrace(IsTraceEnabled())
	llssa.Initialize(llssa.InitAll)

	target := &llssa.Target{
		GOOS:     conf.Goos,
		GOARCH:   conf.Goarch,
		Target:   conf.Target,
		OptLevel: conf.OptLevel,
	}

	prog := llssa.NewProgram(target)
	sizes := func(sizes types.Sizes, compiler, arch string) types.Sizes {
		if arch == "wasm" {
			sizes = &types.StdSizes{WordSize: 4, MaxAlign: 4}
		}
		return prog.TypeSizes(sizes)
	}
	dedup := packages.NewDeduper()
	dedup.SetPreload(func(pkg *types.Package, files []*ast.File) {
		if llruntime.SkipToBuild(pkg.Path()) {
			return
		}
		cl.ParsePkgSyntax(prog, pkg, files)
	})

	if patterns == nil {
		patterns = []string{"."}
	}
	sourcePatchGOROOT, sourcePatchGoVersion, err := env.GOROOTAndGOVERSIONWithEnv(cfg.Env)
	if err != nil {
		return nil, err
	}
	cfg.Overlay, err = buildSourcePatchOverlayForGOROOT(cfg.Overlay, env.LLGoRuntimeDir(), sourcePatchGOROOT, sourcePatchBuildContext{
		goos:       conf.Goos,
		goarch:     conf.Goarch,
		goversion:  sourcePatchGoVersion,
		buildFlags: cfg.BuildFlags,
	})
	if err != nil {
		return nil, err
	}
	initial, err := packages.LoadEx(dedup, sizes, cfg, patterns...)
	if err != nil {
		return nil, err
	}
	mode := conf.Mode
	if mode == ModeTest {
		initial, err = filterTestPackages(initial, conf.OutFile)
		if err != nil {
			return nil, err
		}
		if len(initial) == 0 {
			return nil, nil
		}
	} else if len(initial) > 1 {
		switch mode {
		case ModeBuild:
			if conf.OutFile != "" {
				return nil, fmt.Errorf("cannot build multiple packages with -o")
			}
		case ModeInstall:
			if conf.Target != "" {
				return nil, fmt.Errorf("cannot install multiple packages to embedded target")
			}
		case ModeRun:
			return nil, fmt.Errorf("cannot run multiple packages")
		}
	}

	altPkgPaths := altPkgs(initial, conf, llssa.PkgRuntime)
	cfg.Dir = env.LLGoRuntimeDir()
	altPkgs, err := packages.LoadEx(dedup, sizes, cfg, altPkgPaths...)
	if err != nil {
		return nil, err
	}

	prog.SetRuntime(func() *types.Package {
		return altPkgs[0].Types
	})
	prog.SetPython(func() *types.Package {
		return dedup.Check(llssa.PkgPython).Types
	})
	preCollectRuntimeLinknames(prog, altPkgs)

	buildMode := ssaBuildMode()
	cabiOptimize := true
	passOpt := true
	if IsDbgEnabled() || mode == ModeGen {
		passOpt = false
	}
	if IsDbgEnabled() {
		buildMode |= ssa.GlobalDebug
		cabiOptimize = false
	}
	if !IsOptimizeEnabled() {
		buildMode |= ssa.NaiveForm
	}
	progSSA := ssa.NewProgram(initial[0].Fset, buildMode)
	patches := make(cl.Patches, len(altPkgPaths))
	altSSAPkgs(progSSA, patches, altPkgs[1:], conf, verbose)

	env := llvm.New("")
	prependEnvPath(env.BinDir())

	output := conf.OutFile != ""
	ctx := &context{env: env, conf: cfg, progSSA: progSSA, prog: prog, dedup: dedup,
		patches: patches, built: make(map[string]none), initial: initial, mode: mode,
		fingerprinting: make(map[string]bool),
		pkgs:           map[*packages.Package]Package{},
		pkgByID:        map[string]Package{},
		output:         output,
		passOpt:        passOpt,
		buildConf:      conf,
		crossCompile:   export,
		cTransformer:   cabi.NewTransformer(prog, export.LLVMTarget, export.TargetABI, conf.AbiMode, cabiOptimize),
	}

	// default runtime globals must be registered before packages are built
	addGlobalString(conf, "runtime.defaultGOROOT="+runtime.GOROOT(), nil)
	addGlobalString(conf, "runtime.buildVersion="+runtime.Version(), nil)
	pkgs, err := buildSSAPkgs(ctx, initial, verbose)
	if err != nil {
		return nil, err
	}
	depPkgs, err := buildSSAPkgs(ctx, altPkgs, verbose)
	if err != nil {
		return nil, err
	}

	allPkgs := append([]*aPackage{}, pkgs...)
	allPkgs = append(allPkgs, depPkgs...)
	allPkgs, err = buildAllPkgs(ctx, allPkgs, verbose)
	if err != nil {
		return nil, err
	}

	if mode == ModeGen {
		for _, pkg := range allPkgs {
			if pkg.Package == initial[0] {
				return []*aPackage{pkg}, nil
			}
		}
		return nil, fmt.Errorf("initial package not found")
	}

	for _, pkg := range initial {
		if needLink(pkg, mode) {
			name := path.Base(pkg.PkgPath)

			// Create output format details
			outFmts, err := buildOutFmts(name, conf, len(ctx.initial) > 1, &ctx.crossCompile)
			if err != nil {
				return nil, err
			}

			// Link main package using the output path from buildOutFmts
			err = linkMainPkg(ctx, pkg, allPkgs, outFmts.Out, verbose)
			if err != nil {
				return nil, err
			}
			if conf.Mode == ModeBuild && conf.SizeReport {
				if err := reportBinarySize(outFmts.Out, conf.SizeFormat, conf.SizeLevel, allPkgs); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: size report failed: %v\n", err)
				}
			}

			// Generate C headers for c-archive and c-shared modes before linking
			if ctx.buildConf.BuildMode == BuildModeCArchive || ctx.buildConf.BuildMode == BuildModeCShared {
				libname := strings.TrimSuffix(filepath.Base(outFmts.Out), conf.AppExt)
				headerPath := filepath.Join(filepath.Dir(outFmts.Out), libname) + ".h"
				pkgs := make([]llssa.Package, 0, len(allPkgs))
				for _, p := range allPkgs {
					if p.LPkg != nil {
						pkgs = append(pkgs, p.LPkg)
					}
				}
				headerErr := header.GenHeaderFile(prog, pkgs, libname, headerPath, verbose)
				if headerErr != nil {
					return nil, headerErr
				}
				continue
			}

			envMap := outFmts.ToEnvMap()

			// Only convert formats when Target is specified
			if conf.Target != "" {
				// Process format conversions for embedded targets
				err = firmware.ConvertFormats(ctx.crossCompile.BinaryFormat, ctx.crossCompile.FormatDetail, envMap)
				if err != nil {
					return nil, err
				}
			}

			switch mode {
			case ModeBuild:
				// Do nothing

			case ModeInstall:
				// Native already installed in linkMainPkg
				if conf.Target != "" {
					err = flash.FlashDevice(ctx.crossCompile.Device, envMap, ctx.buildConf.Port, verbose)
					if err != nil {
						return nil, err
					}
				}

			case ModeRun, ModeTest, ModeCmpTest:
				if conf.Target == "" {
					err = runNative(ctx, outFmts.Out, pkg.Dir, pkg.PkgPath, conf, mode)
				} else if conf.Emulator {
					err = runInEmulator(ctx.crossCompile.Emulator, envMap, pkg.Dir, pkg.PkgPath, conf, mode, verbose)
				} else {
					err = flash.FlashDevice(ctx.crossCompile.Device, envMap, ctx.buildConf.Port, verbose)
					if err != nil {
						return nil, err
					}
					monitorConfig := monitor.MonitorConfig{
						Port:       ctx.buildConf.Port,
						Target:     conf.Target,
						Executable: outFmts.Out,
						BaudRate:   conf.BaudRate,
						SerialPort: ctx.crossCompile.Device.SerialPort,
					}
					err = monitor.Monitor(monitorConfig, verbose)
				}
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if mode == ModeTest && ctx.testFail {
		mockable.Exit(1)
	}

	return allPkgs, nil
}

func needLink(pkg *packages.Package, mode Mode) bool {
	if mode == ModeTest {
		return strings.HasSuffix(pkg.ID, ".test")
	}
	return pkg.Name == "main"
}

func filterTestPackages(initial []*packages.Package, outFile string) ([]*packages.Package, error) {
	filtered := initial[:0]
	for _, pkg := range initial {
		if needLink(pkg, ModeTest) {
			filtered = append(filtered, pkg)
		}
	}
	if len(filtered) > 1 && outFile != "" {
		return nil, fmt.Errorf("cannot use -o flag with multiple packages")
	}
	return filtered, nil
}

func (p Package) setNeedRuntimeOrPyInit(needRuntime, needPyInit bool) {
	p.NeedRt = needRuntime
	p.NeedPyInit = needPyInit
}

func (p Package) isNeedRuntimeOrPyInit() (needRuntime, needPyInit bool) {
	needRuntime = p.NeedRt
	needPyInit = p.NeedPyInit
	return
}

func ssaBuildMode() ssa.BuilderMode {
	mode := ssa.InstantiateGenerics
	if ssaSanityEnabled() {
		mode |= ssa.SanityCheckFunctions
	}
	return mode
}

func ssaSanityEnabled() bool {
	return isEnvOn(llgoSSASanity, false)
}

type context struct {
	env            *llvm.Env
	conf           *packages.Config
	progSSA        *ssa.Program
	prog           llssa.Program
	dedup          packages.Deduper
	patches        cl.Patches
	built          map[string]none
	fingerprinting map[string]bool
	initial        []*packages.Package
	pkgs           map[*packages.Package]Package // cache for lookup
	pkgByID        map[string]Package            // cache for lookup by pkg.ID
	mode           Mode
	nLibdir        int32
	output         bool
	passOpt        bool

	buildConf    *Config
	crossCompile crosscompile.Export

	cTransformer *cabi.Transformer

	testFail bool

	// Cache related fields
	cacheManager       *cacheManager
	pendingCacheSaves  []*pendingCacheSave
	cacheSaveSem       chan struct{}
	objectEmitSem      chan struct{}
	llvmVersion        string
	targetTripleCached string
	buildTagsRaw       string
	buildTagsCached    []string
	envInputsCached    envSection
	envInputsReady     bool

	// go list derived file lists (SFiles, etc.)
	sfilesCache map[string][]string // pkg.ID -> absolute .s/.S file paths

	// plan9asm package policy parsed from env.
	plan9asmOnce sync.Once
	plan9asmMode plan9asmPkgsEnvMode
	plan9asmPkgs map[string]bool
}

type pendingCacheSave struct {
	pkg  *aPackage
	done chan cacheSaveResult
}

type cacheSaveResult struct {
	saveErr error
	err     error
}

func (c *context) startCacheSave(pkg *aPackage, verbose bool) {
	c.ensureCacheManager()
	c.targetTriple()
	if c.cacheSaveSem == nil {
		limit := runtime.GOMAXPROCS(0) / 2
		if limit < 1 {
			limit = 1
		} else if limit > 4 {
			limit = 4
		}
		c.cacheSaveSem = make(chan struct{}, limit)
	}
	pending := &pendingCacheSave{pkg: pkg, done: make(chan cacheSaveResult, 1)}
	c.pendingCacheSaves = append(c.pendingCacheSaves, pending)
	go func() {
		c.cacheSaveSem <- struct{}{}
		defer func() { <-c.cacheSaveSem }()

		var res cacheSaveResult
		if res.err = waitPackageObjectEmit(pkg); res.err != nil {
			pending.done <- res
			return
		}
		if cacheEnabled() && pkg.Name != "main" && pkg.Fingerprint != "" && pkg.Manifest != "" {
			res.saveErr = c.saveToCache(pkg)
		}
		if pkg.ArchiveFile == "" {
			res.err = normalizeToArchive(c, pkg, verbose)
		}
		pending.done <- res
	}()
}

func (c *context) waitCacheSaves(verbose bool) error {
	var firstErr error
	for _, pending := range c.pendingCacheSaves {
		res := <-pending.done
		if res.saveErr != nil && verbose {
			fmt.Fprintf(os.Stderr, "warning: failed to save cache for %s: %v\n", pending.pkg.PkgPath, res.saveErr)
		}
		if firstErr == nil && res.err != nil {
			firstErr = res.err
		}
	}
	c.pendingCacheSaves = nil
	return firstErr
}

func (c *context) compiler() *clang.Cmd {
	config := clang.NewConfig(
		c.crossCompile.CC,
		c.crossCompile.CCFLAGS,
		c.crossCompile.CFLAGS,
		c.crossCompile.LDFLAGS,
		c.crossCompile.Linker,
	)
	cmd := clang.NewCompiler(config)
	cmd.Verbose = c.shouldPrintCommands(false)
	return cmd
}

func (c *context) linker() *clang.Cmd {
	config := clang.NewConfig(
		c.crossCompile.CC,
		c.crossCompile.CCFLAGS,
		c.crossCompile.CFLAGS,
		c.crossCompile.LDFLAGS,
		c.crossCompile.Linker,
	)
	cmd := clang.NewLinker(config)
	cmd.Verbose = c.shouldPrintCommands(false)
	return cmd
}

// shouldPrintCommands reports whether command tracing should be enabled.
func (c *context) shouldPrintCommands(verbose bool) bool {
	return c.buildConf.PrintCommands || c.buildConf.Verbose || verbose
}

func (c *context) hasAltPkg(pkgPath string) bool {
	return hasAltPkgForTarget(c.buildConf, pkgPath)
}

// normalizeToArchive creates an archive from object files and sets ArchiveFile.
// This ensures the link step always consumes .a archives regardless of cache state.
func normalizeToArchive(ctx *context, aPkg *aPackage, verbose bool) error {
	if len(aPkg.ObjFiles) == 0 {
		return nil
	}

	archiveFile, err := os.CreateTemp("", "pkg-*.a")
	if err != nil {
		return fmt.Errorf("create temp archive: %w", err)
	}
	archiveFile.Close()
	archivePath := archiveFile.Name()

	if err := ctx.createArchiveFile(archivePath, aPkg.ObjFiles, verbose); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("create archive for %s: %w", aPkg.PkgPath, err)
	}

	aPkg.ObjFiles = nil
	aPkg.ArchiveFile = archivePath
	return nil
}

func parseBuildFile(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	return parser.ParseFile(fset, filename, src, parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution)
}

func buildAllPkgs(ctx *context, pkgs []*aPackage, verbose bool) ([]*aPackage, error) {
	built := ctx.built

	// Split packages into runtime tree vs others so we can defer runtime build.
	var runtimePkgs []*aPackage
	var normalPkgs []*aPackage
	for _, p := range pkgs {
		if isRuntimePkg(p.PkgPath) {
			runtimePkgs = append(runtimePkgs, p)
		} else {
			normalPkgs = append(normalPkgs, p)
		}
	}

	var needRuntime, needPyInit bool

	buildOne := func(aPkg *aPackage) error {
		pkg := aPkg.Package
		if _, ok := built[pkg.ID]; ok {
			// Already built, skip but keep ExportFile for linking
			return nil
		}
		built[pkg.ID] = none{}

		switch kind, param := cl.PkgKindOf(pkg.Types); kind {
		case cl.PkgDeclOnly:
			pkg.ExportFile = ""
		case cl.PkgLinkIR, cl.PkgLinkExtern, cl.PkgPyModule:
			if len(pkg.GoFiles) > 0 {
				if err := ctx.collectFingerprint(aPkg); err != nil {
					return err
				}
				ctx.tryLoadFromCache(aPkg)
				if verbose {
					if aPkg.CacheHit {
						fmt.Fprintf(os.Stderr, "CACHE HIT: %s\n", pkg.PkgPath)
					} else {
						fmt.Fprintf(os.Stderr, "CACHE MISS: %s\n", pkg.PkgPath)
					}
				}
				if err := buildPkg(ctx, aPkg, verbose); err != nil {
					return err
				}
				if !aPkg.CacheHit {
					if kind == cl.PkgLinkExtern {
						appendExternalLinkArgs(ctx, aPkg, param)
					}
					ctx.startCacheSave(aPkg, verbose)
				}
			} else {
				pkg.ExportFile = ""
				if kind == cl.PkgLinkExtern {
					appendExternalLinkArgs(ctx, aPkg, param)
				}
			}
		default:
			if err := ctx.collectFingerprint(aPkg); err != nil {
				return err
			}
			ctx.tryLoadFromCache(aPkg)
			if verbose {
				if aPkg.CacheHit {
					fmt.Fprintf(os.Stderr, "CACHE HIT: %s\n", pkg.PkgPath)
				} else {
					fmt.Fprintf(os.Stderr, "CACHE MISS: %s\n", pkg.PkgPath)
				}
			}
			if err := buildPkg(ctx, aPkg, verbose); err != nil {
				return err
			}
			aPkg.setNeedRuntimeOrPyInit(aPkg.LPkg.NeedRuntime, aPkg.LPkg.NeedPyInit)
			needRuntime = needRuntime || aPkg.NeedRt
			needPyInit = needPyInit || aPkg.NeedPyInit
			if !aPkg.CacheHit {
				ctx.startCacheSave(aPkg, verbose)
			}
		}
		return nil
	}

	// Build non-runtime packages first, so we know whether runtime is actually needed.
	for _, p := range normalPkgs {
		if err := buildOne(p); err != nil {
			_ = ctx.waitCacheSaves(verbose)
			return nil, err
		}
	}

	// Only build runtime packages when required (or host build with empty Target).
	if needRuntime || needPyInit || ctx.buildConf.Target == "" {
		for _, p := range runtimePkgs {
			if err := buildOne(p); err != nil {
				_ = ctx.waitCacheSaves(verbose)
				return nil, err
			}
		}
	}

	if err := ctx.waitCacheSaves(verbose); err != nil {
		return nil, err
	}
	if err := ctx.waitObjectEmits(pkgs); err != nil {
		return nil, err
	}

	return pkgs, nil
}

func appendExternalLinkArgs(ctx *context, aPkg *aPackage, spec string) {
	// need to be linked with external library
	// format: ';' separated alternative link methods. e.g.
	//   link: $LLGO_LIB_PYTHON; $(pkg-config --libs python3-embed); -lpython3
	altParts := strings.Split(spec, ";")
	expdArgs := make([]string, 0, len(altParts))
	for _, alt := range altParts {
		alt = strings.TrimSpace(alt)
		if strings.ContainsRune(alt, '$') {
			expdArgs = append(expdArgs, xenv.ExpandEnvToArgs(alt)...)
			atomic.AddInt32(&ctx.nLibdir, 1)
		} else {
			fields := strings.Fields(alt)
			expdArgs = append(expdArgs, fields...)
		}
		if len(expdArgs) > 0 {
			break
		}
	}
	if len(expdArgs) == 0 {
		panic(fmt.Sprintf("'%s' cannot locate the external library", spec))
	}

	pkgLinkArgs := make([]string, 0, 3)
	if expdArgs[0][0] == '-' {
		pkgLinkArgs = append(pkgLinkArgs, expdArgs...)
	} else {
		linkFile := expdArgs[0]
		dir, lib := filepath.Split(linkFile)
		pkgLinkArgs = append(pkgLinkArgs, "-l"+lib)
		if dir != "" {
			pkgLinkArgs = append(pkgLinkArgs, "-L"+dir)
			atomic.AddInt32(&ctx.nLibdir, 1)
		}
	}
	if ctx.buildConf.CheckLinkArgs {
		if err := ctx.compiler().CheckLinkArgs(pkgLinkArgs, isWasmTarget(ctx.buildConf.Goos)); err != nil {
			panic(fmt.Sprintf("test link args '%s' failed\n\texpanded to: %v\n\tresolved to: %v\n\terror: %v", spec, expdArgs, pkgLinkArgs, err))
		}
	}
	aPkg.LinkArgs = append(aPkg.LinkArgs, pkgLinkArgs...)
}

var (
	errXflags = errors.New("-X flag requires argument of the form importpath.name=value")
)

// maxRewriteValueLength limits the size of rewrite values to prevent
// excessive memory usage and potential resource exhaustion during compilation.
const maxRewriteValueLength = 1 << 20 // 1 MiB cap per rewrite value

func addGlobalString(conf *Config, arg string, mainPkgs []string) {
	addGlobalStringWith(conf, arg, mainPkgs, true)
}

func addGlobalStringWith(conf *Config, arg string, mainPkgs []string, skipIfExists bool) {
	eq := strings.Index(arg, "=")
	dot := strings.LastIndex(arg[:eq+1], ".")
	if eq < 0 || dot < 0 {
		panic(errXflags)
	}
	pkg := arg[:dot]
	varName := arg[dot+1 : eq]
	value := arg[eq+1:]
	validateRewriteInput(pkg, varName, value)
	pkgs := []string{pkg}
	if pkg == "main" {
		pkgs = mainPkgs
	}
	if len(pkgs) == 0 {
		return
	}
	if conf.GlobalRewrites == nil {
		conf.GlobalRewrites = make(map[string]Rewrites)
	}
	for _, realPkg := range pkgs {
		vars := conf.GlobalRewrites[realPkg]
		if vars == nil {
			vars = make(Rewrites)
			conf.GlobalRewrites[realPkg] = vars
		}
		if skipIfExists {
			if _, exists := vars[varName]; exists {
				continue
			}
		}
		vars[varName] = value
	}
}

func validateRewriteInput(pkg, varName, value string) {
	if pkg == "" || strings.ContainsAny(pkg, " \t\r\n") {
		panic(fmt.Errorf("invalid package path for rewrite: %q", pkg))
	}
	if !token.IsIdentifier(varName) {
		panic(fmt.Errorf("invalid variable name for rewrite: %q", varName))
	}
	if len(value) > maxRewriteValueLength {
		panic(fmt.Errorf("rewrite value too large: %d bytes", len(value)))
	}
}

// compileExtraFiles compiles extra files (.s/.c) from target configuration and returns object files
func compileExtraFiles(ctx *context, verbose bool) ([]string, error) {
	if len(ctx.crossCompile.ExtraFiles) == 0 {
		return nil, nil
	}

	printCmds := ctx.shouldPrintCommands(verbose)
	var objFiles []string
	llgoRoot := env.LLGoROOT()

	for _, extraFile := range ctx.crossCompile.ExtraFiles {
		// Resolve the file path relative to llgo root
		srcFile := filepath.Join(llgoRoot, extraFile)

		// Check if file exists
		if _, err := os.Stat(srcFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("extra file not found: %s", srcFile)
		}

		// Generate output file name
		tmpFile, err := os.CreateTemp("", "extra-*"+filepath.Base(extraFile))
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file for %s: %w", extraFile, err)
		}
		tmpFile.Close()
		baseName := tmpFile.Name()

		ext := filepath.Ext(srcFile)

		// Prepare base compilation arguments
		var baseArgs []string

		// Handle different file types
		switch ext {
		case ".c":
			baseArgs = append(baseArgs, "-x", "c")
		case ".S", ".s":
			baseArgs = append(baseArgs, "-x", "assembler-with-cpp")
		}

		// If GenLL is enabled, first emit .ll for debugging
		if ctx.buildConf.GenLL {
			llFile := baseName + ".ll"
			llArgs := append(slices.Clone(baseArgs), "-emit-llvm", "-S", "-o", llFile, "-c", srcFile)
			if printCmds {
				fmt.Fprintf(os.Stderr, "Compiling extra file (ll): clang %s\n", strings.Join(llArgs, " "))
			}
			cmd := ctx.compiler()
			if err := cmd.Compile(llArgs...); err != nil {
				return nil, fmt.Errorf("failed to compile extra file %s to .ll: %w", srcFile, err)
			}
		}

		// Always compile to .o for linking
		objFile := baseName + ".o"
		objArgs := append(baseArgs, "-o", objFile, "-c", srcFile)
		if printCmds {
			fmt.Fprintf(os.Stderr, "Compiling extra file: clang %s\n", strings.Join(objArgs, " "))
		}
		cmd := ctx.compiler()
		if err := cmd.Compile(objArgs...); err != nil {
			return nil, fmt.Errorf("failed to compile extra file %s: %w", srcFile, err)
		}

		objFiles = append(objFiles, objFile)
		os.Remove(baseName) // Remove the temp file we created for naming
	}

	return objFiles, nil
}

func linkMainPkg(ctx *context, pkg *packages.Package, pkgs []*aPackage, outputPath string, verbose bool) error {
	needRuntime := false
	needPyInit := false
	var needAbiInit int
	methodByIndex := make(map[int]none)
	methodByName := make(map[string]none)
	allPkgs := []*packages.Package{pkg}
	for _, v := range pkgs {
		allPkgs = append(allPkgs, v.Package)
	}
	visitRoots := allPkgs
	if ctx.mode == ModeTest {
		visitRoots = []*packages.Package{pkg}
		for _, p := range allPkgs {
			if isRuntimePkg(p.PkgPath) {
				visitRoots = append(visitRoots, p)
			}
		}
	}
	// archiveInputs contains package .a files. Object files are prepended later so
	// archive extraction can see their undefined references in a single linker pass.
	var archiveInputs []string
	var linkArgs []string
	var rtLinkInputs []string
	var rtLinkArgs []string
	linkedPkgs := make(map[string]bool) // Track linked packages by ID to avoid duplicates
	var linkedOrder []Package
	packages.Visit(visitRoots, nil, func(p *packages.Package) {
		// Skip if already linked this package (by ID)
		if linkedPkgs[p.ID] {
			return
		}
		aPkg := ctx.pkgs[p]
		if aPkg == nil {
			// Fallback: lookup by pkg.ID for packages that may be different instances
			aPkg = ctx.pkgByID[p.ID]
		}
		if p.ExportFile != "" && aPkg != nil { // skip packages that only contain declarations
			linkedPkgs[p.ID] = true
			linkedOrder = append(linkedOrder, aPkg)
		}
	})

	// packages.Visit with a post callback yields dependencies before importers.
	// Reverse that order so static archives are linked after the objects that use them.
	for i := len(linkedOrder) - 1; i >= 0; i-- {
		aPkg := linkedOrder[i]
		p := aPkg.Package
		// Defer linking runtime packages unless we actually need the runtime.
		if isRuntimePkg(p.PkgPath) {
			rtLinkArgs = append(rtLinkArgs, aPkg.LinkArgs...)
			if aPkg.ArchiveFile != "" {
				rtLinkInputs = append(rtLinkInputs, aPkg.ArchiveFile)
			}
			continue
		}
		// Only let non-runtime packages influence whether runtime is needed.
		need1, need2 := aPkg.isNeedRuntimeOrPyInit()
		needRuntime = needRuntime || need1
		needPyInit = needPyInit || need2
		needAbiInit |= aPkg.LPkg.NeedAbiInit
		for k, _ := range aPkg.LPkg.MethodByIndex {
			methodByIndex[k] = none{}
		}
		for k, _ := range aPkg.LPkg.MethodByName {
			methodByName[k] = none{}
		}

		linkArgs = append(linkArgs, aPkg.LinkArgs...)
		if aPkg.ArchiveFile != "" {
			archiveInputs = append(archiveInputs, aPkg.ArchiveFile)
		}
	}

	// Only link runtime objects when needed (or for host builds where runtime is always required).
	if needRuntime || needPyInit || ctx.buildConf.Target == "" {
		linkArgs = append(linkArgs, rtLinkArgs...)
		archiveInputs = append(archiveInputs, rtLinkInputs...)
	}

	// Generate main module file (needed for global variables even in library modes)
	// This is compiled directly to .o and added to linkInputs (not cached)
	// Use a stable synthetic name to avoid confusing it with the real main package in traces/logs.
	entryPkg := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{
		rtInit:        needRuntime,
		pyInit:        needPyInit,
		abiInit:       needAbiInit,
		methodByIndex: methodByIndex,
		methodByName:  methodByName,
		abiSymbols:    linkedModuleGlobals(linkedOrder),
	})
	entryObjFile, err := exportObject(ctx, "entry_main", entryPkg.ExportFile, entryPkg.LPkg)
	if err != nil {
		return err
	}
	linkInputs := []string{entryObjFile}

	// Compile extra files from target configuration
	extraObjFiles, err := compileExtraFiles(ctx, verbose)
	if err != nil {
		return err
	}
	linkInputs = append(linkInputs, extraObjFiles...)
	linkInputs = append(linkInputs, archiveInputs...)

	if IsFullRpathEnabled() {
		// Treat every link-time library search path, specified by the -L parameter, as a runtime search path as well.
		// This is to ensure the final executable can locate libraries with a relocatable install_name
		// (e.g., "@rpath/libfoo.dylib") at runtime.
		rpaths := make(map[string]none)
		for _, arg := range linkArgs {
			if strings.HasPrefix(arg, "-L") {
				path := arg[2:]
				if _, ok := rpaths[path]; ok {
					continue
				}
				rpaths[path] = none{}
				linkArgs = append(linkArgs, "-rpath", path)
			}
		}
	}

	err = linkObjFiles(ctx, outputPath, linkInputs, linkArgs, verbose)
	if err != nil {
		return err
	}

	return nil
}

func linkedModuleGlobals(pkgs []Package) map[string]none {
	if len(pkgs) == 0 {
		return nil
	}
	seen := make(map[string]none)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.LPkg == nil {
			continue
		}
		for g := pkg.LPkg.Module().FirstGlobal(); !g.IsNil(); g = gllvm.NextGlobal(g) {
			seen[g.Name()] = none{}
		}
	}
	return seen
}

// isRuntimePkg reports whether the package path belongs to the llgo runtime tree.
func isRuntimePkg(pkgPath string) bool {
	rtRoot := env.LLGoRuntimePkg
	return pkgPath == rtRoot || strings.HasPrefix(pkgPath, rtRoot+"/")
}

func linkObjFiles(ctx *context, app string, objFiles, linkArgs []string, verbose bool) error {
	printCmds := ctx.shouldPrintCommands(verbose)
	// Handle c-archive mode differently - use ar tool instead of linker
	if ctx.buildConf.BuildMode == BuildModeCArchive {
		return ctx.createArchiveFile(app, objFiles, printCmds)
	}

	buildArgs := []string{"-o", app}
	buildArgs = append(buildArgs, linkArgs...)

	// Add build mode specific linker arguments
	switch ctx.buildConf.BuildMode {
	case BuildModeCShared:
		buildArgs = append(buildArgs, "-shared", "-fPIC")
	case BuildModeExe:
		if needsLinuxNoPIE(ctx, linkArgs) {
			buildArgs = append(buildArgs, "-no-pie")
		}
	}

	// Add common linker arguments based on target OS and architecture
	if IsDbgSymsEnabled() {
		buildArgs = append(buildArgs, "-gdwarf-4")
	}

	if ctx.buildConf.GenLL {
		var compiledObjFiles []string
		for _, objFile := range objFiles {
			if strings.HasSuffix(objFile, ".ll") {
				oFile := strings.TrimSuffix(objFile, ".ll") + ".o"
				args := []string{"-o", oFile, "-c", objFile, "-Wno-override-module"}
				if printCmds {
					fmt.Fprintln(os.Stderr, "clang", args)
				}
				if err := ctx.compiler().Compile(args...); err != nil {
					return fmt.Errorf("failed to compile %s: %v", objFile, err)
				}
				compiledObjFiles = append(compiledObjFiles, oFile)
			} else {
				compiledObjFiles = append(compiledObjFiles, objFile)
			}
		}
		objFiles = compiledObjFiles
	}

	buildArgs = append(buildArgs, objFiles...)

	cmd := ctx.linker()
	cmd.Verbose = printCmds
	return cmd.Link(buildArgs...)
}

func needsLinuxNoPIE(ctx *context, linkArgs []string) bool {
	if ctx.buildConf.Target != "" || ctx.buildConf.Goos != "linux" {
		return false
	}
	// Host Linux toolchains commonly default to PIE executables, which can
	// break runtime assumptions unless the user explicitly requested a PIE mode.
	for _, arg := range linkArgs {
		if arg == "-pie" || arg == "-static-pie" || arg == "-no-pie" || arg == "-nopie" {
			return false
		}
	}
	return true
}

// archiver returns the archiving tool to use for the current context.
// For wasm targets, it prefers llvm-ar because wasm-ld requires archives
// created with llvm-ar (system ar cannot create valid wasm archive indexes).
func (c *context) archiver() string {
	// First check toolchain directory (for cross-compilation)
	if c.crossCompile.CC != "" {
		clangDir := filepath.Dir(c.crossCompile.CC)
		if clangDir != "" {
			llvmAr := filepath.Join(clangDir, "llvm-ar")
			if _, err := os.Stat(llvmAr); err == nil {
				return llvmAr
			}
		}
	}
	// Allow user override
	if ar := os.Getenv("LLGO_AR"); ar != "" {
		return ar
	}
	// For wasm targets, prefer llvm-ar from PATH (system ar cannot create valid wasm archives)
	if c.buildConf.Goarch == "wasm" || strings.Contains(c.crossCompile.LLVMTarget, "wasm") {
		if llvmAr, err := exec.LookPath("llvm-ar"); err == nil {
			return llvmAr
		}
	}
	return "ar"
}

// createArchiveFile builds an archive at archivePath atomically to avoid races when
// multiple builds target the same output concurrently.
func (c *context) createArchiveFile(archivePath string, objFiles []string, verbose ...bool) error {
	if len(objFiles) == 0 {
		return fmt.Errorf("no object files provided for archive %s", archivePath)
	}

	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(archivePath), filepath.Base(archivePath)+".tmp-*")
	if err != nil {
		return err
	}
	tmp.Close()
	tmpName := tmp.Name()
	// Remove the placeholder so ar can create the archive fresh.
	_ = os.Remove(tmpName)

	args := append([]string{"rcs", tmpName}, objFiles...)
	arCmd := c.archiver()
	cmd := exec.Command(arCmd, args...)
	printCmds := c.shouldPrintCommands(len(verbose) > 0 && verbose[0])
	if printCmds {
		fmt.Fprintf(os.Stderr, "%s %s\n", filepath.Base(arCmd), strings.Join(args, " "))
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("create archive %s: %w\n%s", archivePath, err, output)
	}

	if err := os.Rename(tmpName, archivePath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("publish archive %s: %w", archivePath, err)
	}
	return nil
}

func isWasmTarget(goos string) bool {
	return slices.Contains([]string{"wasi", "js", "wasip1"}, goos)
}

func needStart(ctx *context) bool {
	if ctx.buildConf.Target == "" {
		return !isWasmTarget(ctx.buildConf.Goos)
	}
	switch ctx.buildConf.Target {
	case "wasip2":
		return false
	default:
		// since newlib-esp32 provides _start, we don't need to provide a fake _start function
		return ctx.crossCompile.Libc != "newlib-esp32"
	}
}

func is32Bits(goarch string) bool {
	return goarch == "386" || goarch == "arm" || goarch == "mips" || goarch == "wasm"
}

func buildPkg(ctx *context, aPkg *aPackage, verbose bool) error {
	pkg := aPkg.Package
	pkgPath := pkg.PkgPath
	if debugBuild || verbose {
		fmt.Fprintln(os.Stderr, pkgPath)
	}
	if llruntime.SkipToBuild(pkgPath) {
		pkg.ExportFile = ""
		return nil
	}
	var syntax = pkg.Syntax
	if altPkg := aPkg.AltPkg; altPkg != nil {
		syntax = append(syntax, altPkg.Syntax...)
	}
	showDetail := verbose && pkgExists(ctx.initial, pkg)
	if showDetail {
		llssa.SetDebug(llssa.DbgFlagAll)
		cl.SetDebug(cl.DbgFlagAll)
		defer func() {
			llssa.SetDebug(0)
			cl.SetDebug(0)
		}()
	}

	embedMap, err := goembed.LoadDirectives(ctx.conf.Fset, syntax)
	if err != nil {
		return fmt.Errorf("load go:embed directives for %s failed: %w", pkgPath, err)
	}

	ret, externs, err := cl.NewPackageExWithEmbed(ctx.prog, ctx.patches, aPkg.rewriteVars, aPkg.SSA, syntax, embedMap)
	check(err)

	aPkg.LPkg = ret
	if hook := ctx.buildConf.ModuleHook; hook != nil {
		hook(aPkg)
	}

	// If cache hit, we only needed to register types - skip compilation
	if aPkg.CacheHit {
		return nil
	}

	ctx.cTransformer.SetSkipFuncs(cabiSkipFuncsForPlan9Asm(ctx, pkgPath, ret.Module()))
	ctx.cTransformer.TransformModule(ret.Path(), ret.Module())
	ctx.cTransformer.SetSkipFuncs(nil)

	// Run the default LLVM optimization pipeline selected by the requested -O level.
	if ctx.passOpt {
		mod := ret.Module()
		mod.SetDataLayout(ctx.prog.DataLayout())
		mod.SetTarget(ctx.prog.Target().Spec().Triple)
		pbo := gllvm.NewPassBuilderOptions()
		defer pbo.Dispose()
		if err := mod.RunPasses(llvmPassPipeline(ctx.buildConf.OptLevel), ctx.prog.TargetMachine(), pbo); err != nil {
			return fmt.Errorf("run LLVM passes failed for %v: %v", pkgPath, err)
		}
	}

	printCmds := ctx.shouldPrintCommands(verbose)
	cgoLLFiles, cgoLdflags, err := buildCgo(ctx, aPkg, aPkg.Package.Syntax, externs, printCmds)
	if err != nil {
		return fmt.Errorf("build cgo of %v failed: %v", pkgPath, err)
	}
	goCgoLdflags, goCgoDynimports := collectGoCgoPragmas(aPkg.Package.Syntax)
	aPkg.ObjFiles = append(aPkg.ObjFiles, cgoLLFiles...)
	aPkg.ObjFiles = append(aPkg.ObjFiles, concatPkgLinkFiles(ctx, pkg, printCmds)...)
	if asmObjFiles, err := compilePkgSFiles(ctx, aPkg, pkg, goCgoDynimports, printCmds); err != nil {
		return err
	} else {
		aPkg.ObjFiles = append(aPkg.ObjFiles, asmObjFiles...)
	}
	if aliasObjs, err := buildGoCgoAliasObjects(ctx, pkgPath, goCgoDynimports, printCmds); err != nil {
		return err
	} else {
		aPkg.ObjFiles = append(aPkg.ObjFiles, aliasObjs...)
	}
	aPkg.LinkArgs = append(aPkg.LinkArgs, cgoLdflags...)
	aPkg.LinkArgs = append(aPkg.LinkArgs, goCgoLdflags...)
	if aPkg.AltPkg != nil {
		altLLFiles, altLdflags, e := buildCgo(ctx, aPkg, aPkg.AltPkg.Syntax, externs, printCmds)
		if e != nil {
			return fmt.Errorf("build cgo of %v failed: %v", pkgPath, e)
		}
		altGoCgoLdflags, altGoCgoDynimports := collectGoCgoPragmas(aPkg.AltPkg.Syntax)
		aPkg.ObjFiles = append(aPkg.ObjFiles, altLLFiles...)
		aPkg.ObjFiles = append(aPkg.ObjFiles, concatPkgLinkFiles(ctx, aPkg.AltPkg.Package, printCmds)...)
		if asmObjFiles, err := compilePkgSFiles(ctx, aPkg, aPkg.AltPkg.Package, altGoCgoDynimports, printCmds); err != nil {
			return err
		} else {
			aPkg.ObjFiles = append(aPkg.ObjFiles, asmObjFiles...)
		}
		if aliasObjs, err := buildGoCgoAliasObjects(ctx, pkgPath, altGoCgoDynimports, printCmds); err != nil {
			return err
		} else {
			aPkg.ObjFiles = append(aPkg.ObjFiles, aliasObjs...)
		}
		aPkg.LinkArgs = append(aPkg.LinkArgs, altLdflags...)
		aPkg.LinkArgs = append(aPkg.LinkArgs, altGoCgoLdflags...)
	}
	if pkg.ExportFile != "" {
		if parallelObjectEmitEnabled(ctx) && !verbose {
			ctx.startObjectEmit(aPkg, pkg.PkgPath, pkg.ExportFile, ret)
		} else {
			exportFile, err := exportObject(ctx, pkg.PkgPath, pkg.ExportFile, ret)
			if err != nil {
				return fmt.Errorf("export object of %v failed: %v", pkgPath, err)
			}
			aPkg.ObjFiles = append(aPkg.ObjFiles, exportFile)
		}
		if debugBuild || verbose {
			fmt.Fprintf(os.Stderr, "==> Export %s: %s\n", aPkg.PkgPath, pkg.ExportFile)
		}
	}
	return nil
}

func exportObject(ctx *context, pkgPath string, exportFile string, pkg llssa.Package) (string, error) {
	if useInMemoryNativeCodegen(ctx) {
		return exportObjectInMemory(ctx, pkgPath, exportFile, pkg)
	}
	return exportObjectWithClang(ctx, pkgPath, exportFile, []byte(pkg.String()))
}

func useInMemoryNativeCodegen(ctx *context) bool {
	return useInMemoryNativeCodegenConf(ctx.buildConf)
}

func useInMemoryNativeCodegenConf(conf *Config) bool {
	return conf != nil && !conf.GenLL &&
		conf.Target == "" &&
		conf.Goos == runtime.GOOS &&
		conf.Goarch == runtime.GOARCH &&
		conf.Goarch != "wasm"
}

// parallelObjectEmitEnabled overlaps native object emission with later package
// work by sending serialized LLVM IR to bounded external clang workers. It is
// disabled for debug/tracing paths so command output and .ll generation remain
// synchronous and deterministic.
func parallelObjectEmitEnabled(ctx *context) bool {
	return isEnvOn(llgoParallelObjectEmit, true) &&
		useInMemoryNativeCodegen(ctx) &&
		!ctx.buildConf.CheckLLFiles &&
		!ctx.shouldPrintCommands(false)
}

type objectEmitResult struct {
	path string
	err  error
}

func (c *context) objectEmitSemaphore() chan struct{} {
	if c.objectEmitSem == nil {
		limit := runtime.GOMAXPROCS(0)
		if limit < 1 {
			limit = 1
		} else if limit > 4 {
			limit = 4
		}
		c.objectEmitSem = make(chan struct{}, limit)
	}
	return c.objectEmitSem
}

func (c *context) startObjectEmit(pkg *aPackage, pkgPath string, exportFile string, llpkg llssa.Package) {
	data := []byte(llpkg.String())
	done := make(chan objectEmitResult, 1)
	sem := c.objectEmitSemaphore()
	pkg.pendingObj = done
	go func() {
		sem <- struct{}{}
		defer func() { <-sem }()
		obj, err := exportObjectWithClang(c, pkgPath, exportFile, data)
		done <- objectEmitResult{path: obj, err: err}
	}()
}

func waitPackageObjectEmit(pkg *aPackage) error {
	if pkg == nil || pkg.pendingObj == nil {
		return nil
	}
	res := <-pkg.pendingObj
	pkg.pendingObj = nil
	if res.err != nil {
		return fmt.Errorf("export object of %v failed: %v", pkg.PkgPath, res.err)
	}
	pkg.ObjFiles = append(pkg.ObjFiles, res.path)
	return nil
}

func (c *context) waitObjectEmits(pkgs []*aPackage) error {
	for _, pkg := range pkgs {
		if err := waitPackageObjectEmit(pkg); err != nil {
			return err
		}
	}
	return nil
}

func dumpLLVMIRIfNeeded(ctx *context, pkgPath string, exportFile string, data string) error {
	if !ctx.buildConf.CheckLLFiles && !ctx.buildConf.GenLL {
		return nil
	}

	base := filepath.Base(exportFile)
	f, err := os.CreateTemp("", base+"-*.ll")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(data); err != nil {
		f.Close()
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	if ctx.buildConf.CheckLLFiles {
		if msg, err := llcCheck(ctx.env, f.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "==> llc %v: %v\n%v\n", pkgPath, f.Name(), msg)
		}
	}
	// If GenLL is enabled, keep a copy of the .ll file for debugging
	if ctx.buildConf.GenLL {
		llFile := exportFile + ".ll"
		if err := os.Chmod(f.Name(), 0644); err != nil {
			return err
		}
		if err := copyFileAtomic(f.Name(), llFile); err != nil {
			return err
		}
	}
	return nil
}

func exportObjectInMemory(ctx *context, pkgPath string, exportFile string, pkg llssa.Package) (string, error) {
	if ctx.buildConf.CheckLLFiles || ctx.buildConf.GenLL {
		// Avoid formatting large IR unless a debug/check path needs it.
		if err := dumpLLVMIRIfNeeded(ctx, pkgPath, exportFile, pkg.String()); err != nil {
			return "", err
		}
	}
	buf, err := ctx.prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), gllvm.ObjectFile)
	if err != nil {
		return "", err
	}
	defer buf.Dispose()

	base := filepath.Base(exportFile)
	objFile, err := os.CreateTemp("", base+"-*.o")
	if err != nil {
		return "", err
	}
	objFileName := objFile.Name()
	if ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", objFileName, pkgPath)
		fmt.Fprintln(os.Stderr, "# using in-memory LLVM object emission")
	}
	if _, err := objFile.Write(buf.Bytes()); err != nil {
		objFile.Close()
		os.Remove(objFileName)
		return "", err
	}
	if err := objFile.Close(); err != nil {
		os.Remove(objFileName)
		return "", err
	}
	return objFileName, nil
}

func exportObjectWithClang(ctx *context, pkgPath string, exportFile string, data []byte) (string, error) {
	base := filepath.Base(exportFile)
	f, err := os.CreateTemp("", base+"-*.ll")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return "", err
	}
	err = f.Close()
	if err != nil {
		return exportFile, err
	}
	if ctx.buildConf.CheckLLFiles {
		if msg, err := llcCheck(ctx.env, f.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "==> llc %v: %v\n%v\n", pkgPath, f.Name(), msg)
		}
	}
	if ctx.buildConf.GenLL {
		llFile := exportFile + ".ll"
		if err := os.Chmod(f.Name(), 0644); err != nil {
			return "", err
		}
		// Copy instead of rename so we can still compile to .o
		if err := copyFileAtomic(f.Name(), llFile); err != nil {
			return "", err
		}
	}
	objFile, err := os.CreateTemp("", base+"-*.o")
	if err != nil {
		return "", err
	}
	objFile.Close()
	args := []string{"-o", objFile.Name(), "-c", f.Name(), "-Wno-override-module"}
	if ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", f.Name(), pkgPath)
		fmt.Fprintln(os.Stderr, "clang", args)
	}
	cmd := ctx.compiler()
	return objFile.Name(), cmd.Compile(args...)
}

func llcCheck(env *llvm.Env, exportFile string) (msg string, err error) {
	bin := filepath.Join(env.BinDir(), "llc")
	cmd := exec.Command(bin, "-filetype=null", exportFile)
	var buf bytes.Buffer
	cmd.Stderr = &buf
	if err = cmd.Run(); err != nil {
		msg = buf.String()
	}
	return
}

const (
	altPkgPathPrefix = abi.PatchPathPrefix
)

func altPkgs(initial []*packages.Package, conf *Config, alts ...string) []string {
	packages.Visit(initial, nil, func(p *packages.Package) {
		if p.Types != nil && !p.IllTyped {
			if hasAltPkgForTarget(conf, p.PkgPath) {
				alts = append(alts, altPkgPathPrefix+p.PkgPath)
			}
		}
	})
	return alts
}

func preCollectRuntimeLinknames(prog llssa.Program, pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		if pkg != nil && pkg.PkgPath == llssa.PkgRuntime && len(pkg.Syntax) != 0 {
			cl.PreCollectLinknames(prog, pkg.PkgPath, pkg.Syntax)
			return
		}
	}
}

func altSSAPkgs(prog *ssa.Program, patches cl.Patches, alts []*packages.Package, conf *Config, verbose bool) {
	packages.Visit(alts, nil, func(p *packages.Package) {
		if typs := p.Types; typs != nil && !p.IllTyped {
			if debugBuild || verbose {
				log.Println("==> BuildSSA", p.ID)
			}
			pkgSSA := prog.CreatePackage(typs, p.Syntax, p.TypesInfo, true)
			if strings.HasPrefix(p.ID, altPkgPathPrefix) {
				path := p.ID[len(altPkgPathPrefix):]
				// Even if an alt package exists and is pulled in as a dependency of other
				// patches (e.g. runtime/reflect), we should only apply it when it is
				// enabled for the target (and not overridden by Plan9 asm translation).
				if !hasAltPkgForTarget(conf, path) {
					return
				}
				patches[path] = cl.Patch{Alt: pkgSSA, Types: typepatch.Clone(typs)}
				if debugBuild || verbose {
					log.Println("==> Patching", path)
				}
			}
		}
	})
	prog.Build()
}

type aPackage struct {
	*packages.Package
	SSA    *ssa.Package
	AltPkg *packages.Cached
	LPkg   llssa.Package

	NeedRt     bool
	NeedPyInit bool

	LinkArgs    []string
	ObjFiles    []string // object files: .o or .ll (output of compiler, input to archiver)
	ArchiveFile string   // archive file: .a (output of archiver, used for linking)
	rewriteVars map[string]string
	pendingObj  chan objectEmitResult

	// Cache related fields
	Fingerprint string // fingerprint digest
	Manifest    string // manifest text content
	CacheHit    bool   // whether cache was hit
}

type Package = *aPackage

func buildSSAPkgs(ctx *context, initial []*packages.Package, verbose bool) ([]*aPackage, error) {
	prog := ctx.progSSA
	var all []*aPackage
	var errs []*packages.Package
	packages.Visit(initial, nil, func(p *packages.Package) {
		if p.Types != nil && !p.IllTyped {
			pkgPath := p.PkgPath
			// Use p.ID to check duplicates since same pkgPath may have different IDs
			if _, ok := ctx.pkgByID[p.ID]; ok || strings.HasPrefix(pkgPath, altPkgPathPrefix) {
				return
			}
			var altPkg *packages.Cached
			var ssaPkg = createSSAPkg(ctx, prog, p, verbose)
			if ctx.hasAltPkg(pkgPath) {
				if altPkg = ctx.dedup.Check(altPkgPathPrefix + pkgPath); altPkg == nil {
					return
				}
			}
			rewrites := collectRewriteVars(ctx, pkgPath)
			aPkg := &aPackage{
				Package:     p,
				SSA:         ssaPkg,
				AltPkg:      altPkg,
				LPkg:        nil,
				NeedRt:      false,
				NeedPyInit:  false,
				LinkArgs:    nil,
				ObjFiles:    nil,
				rewriteVars: rewrites,
			}
			ctx.pkgs[p] = aPkg
			ctx.pkgByID[p.ID] = aPkg
			all = append(all, aPkg)
		} else {
			errs = append(errs, p)
		}
	})
	if len(errs) > 0 {
		for _, errPkg := range errs {
			for _, err := range errPkg.Errors {
				fmt.Fprintln(os.Stderr, err)
			}
			fmt.Fprintln(os.Stderr, "cannot build SSA for package", errPkg)
		}
		return nil, fmt.Errorf("cannot build SSA for packages")
	}
	return all, nil
}

func collectRewriteVars(ctx *context, pkgPath string) map[string]string {
	data := ctx.buildConf.GlobalRewrites
	if len(data) == 0 {
		return nil
	}
	basePath := strings.TrimPrefix(pkgPath, altPkgPathPrefix)
	if vars := data[basePath]; vars != nil {
		return cloneRewrites(vars)
	}
	if vars := data[pkgPath]; vars != nil {
		return cloneRewrites(vars)
	}
	return nil
}

func cloneRewrites(src Rewrites) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dup := make(map[string]string, len(src))
	for k, v := range src {
		dup[k] = v
	}
	return dup
}

func toTypeList(args *types.TypeList) []types.Type {
	if args == nil {
		return nil
	}
	result := make([]types.Type, args.Len())
	for i := 0; i < args.Len(); i++ {
		result[i] = args.At(i)
	}
	return result
}

// fixUntypedShiftTypes fixes a bug in go/types where non-constant shift expressions
// with untyped constant left operands have type "untyped int" instead of "int".
//
// According to the Go spec: "If the left operand of a non-constant shift expression
// is an untyped constant, it is first implicitly converted to the type it would assume
// if the shift expression were replaced by its left operand alone."
//
// This causes go/ssa sanity check to fail when the type remains untyped.
// See: https://github.com/golang/go/issues/77067
func fixUntypedShiftTypes(p *packages.Package) {
	// First pass: identify expressions needing fixes
	// (avoid modifying map during iteration)
	var toFix []ast.Expr
	for expr, tv := range p.TypesInfo.Types {
		switch e := expr.(type) {
		case *ast.BinaryExpr:
			if e.Op != token.SHL && e.Op != token.SHR {
				break
			}
			basic, ok := tv.Type.(*types.Basic)
			if !ok || basic.Info()&types.IsUntyped == 0 {
				break
			}
			toFix = append(toFix, expr)
		}
	}

	// Second pass: apply fixes
	for _, expr := range toFix {
		tv := p.TypesInfo.Types[expr]
		p.TypesInfo.Types[expr] = types.TypeAndValue{
			Type:  types.Default(tv.Type),
			Value: tv.Value,
		}
	}
}

func applyPatches(ctx *context, p *packages.Package, verbose bool) {
	// Fix untyped shift types before SSA sanity checking.
	// See: https://github.com/golang/go/issues/77067
	if ssaSanityEnabled() {
		fixUntypedShiftTypes(p)
	}

	// fix instance patch
	for id, inst := range p.TypesInfo.Instances {
		if obj := p.TypesInfo.Uses[id]; obj != nil && obj.Pkg() != nil && obj.Pkg() != p.Types {
			if pkg := obj.Pkg(); pkg != nil && pkg != p.Types {
				if patch, ok := ctx.patches[pkg.Path()]; ok {
					if robj := patch.Alt.Pkg.Scope().Lookup(obj.Name()); robj != nil {
						typ, err := types.Instantiate(nil, robj.Type(), toTypeList(inst.TypeArgs), true)
						if err != nil {
							if debugBuild || verbose {
								log.Printf("==> Instance patch failed for %q: %v\n", obj.Id(), err)
							}
							continue
						}
						inst.Type = typ
						p.TypesInfo.Instances[id] = inst
						p.TypesInfo.Uses[id] = robj
					}
				}
			}
		}
	}
}

func createSSAPkg(ctx *context, prog *ssa.Program, p *packages.Package, verbose bool) *ssa.Package {
	pkgSSA := prog.ImportedPackage(p.ID)
	if pkgSSA == nil {
		if debugBuild || verbose {
			log.Println("==> BuildSSA", p.ID)
		}
		applyPatches(ctx, p, verbose)
		pkgSSA = prog.CreatePackage(p.Types, p.Syntax, p.TypesInfo, true)
		pkgSSA.Build() // TODO(xsw): build concurrently
		// Apply local SSA fixups once when package SSA is first built.
		fixSSAOrder(pkgSSA, p.Syntax)
	}
	return pkgSSA
}

/*
var (
	// TODO(xsw): complete build flags
	buildFlags = map[string]bool{
		"-C":         true,  // -C dir: Change to dir before running the command
		"-a":         false, // -a: force rebuilding of packages that are already up-to-date
		"-n":         false, // -n: print the commands but do not run them
		"-p":         true,  // -p n: the number of programs to run in parallel
		"-race":      false, // -race: enable data race detection
		"-cover":     false, // -cover: enable coverage analysis
		"-covermode": true,  // -covermode mode: set the mode for coverage analysis
		"-v":         false, // -v: print the names of packages as they are compiled
		"-work":      false, // -work: print the name of the temporary work directory and do not delete it when exiting
		"-x":         false, // -x: print the commands
		"-tags":      true,  // -tags 'tag,list': a space-separated list of build tags to consider satisfied during the build
		"-pkgdir":    true,  // -pkgdir dir: install and load all packages from dir instead of the usual locations
		"-ldflags":   true,  // --ldflags 'flag list': arguments to pass on each go tool link invocation
	}
)
*/

const llgoDebug = "LLGO_DEBUG"
const llgoDbgSyms = "LLGO_DEBUG_SYMBOLS"
const llgoTrace = "LLGO_TRACE"
const llgoOptimize = "LLGO_OPTIMIZE"
const llgoWasmRuntime = "LLGO_WASM_RUNTIME"
const llgoWasiThreads = "LLGO_WASI_THREADS"
const llgoStdioNobuf = "LLGO_STDIO_NOBUF"
const llgoFullRpath = "LLGO_FULL_RPATH"
const llgoBuildCache = "LLGO_BUILD_CACHE"
const llgoSSASanity = "LLGO_SSA_SANITY"
const llgoParallelObjectEmit = "LLGO_PARALLEL_OBJECT_EMIT"

// for Plan9 asm translation debug
const llgoPlan9ASMPkgs = "LLGO_PLAN9ASM_PKGS"

const defaultWasmRuntime = "wasmtime"

func defaultEnv(env string, defVal string) string {
	envVal := os.Getenv(env)
	if envVal == "" {
		return defVal
	}
	return envVal
}

func isEnvOn(env string, defVal bool) bool {
	envVal := strings.ToLower(os.Getenv(env))
	if envVal == "" {
		return defVal
	}
	return envVal == "1" || envVal == "true" || envVal == "on"
}

// cacheEnabled checks if build cache is enabled.
// Cache can be disabled by setting LLGO_BUILD_CACHE=off|0
func cacheEnabled() bool {
	return isEnvOn(llgoBuildCache, true)
}

func IsTraceEnabled() bool {
	return isEnvOn(llgoTrace, false)
}

func IsStdioNobuf() bool {
	return isEnvOn(llgoStdioNobuf, false)
}

func IsDbgEnabled() bool {
	return isEnvOn(llgoDebug, false) || isEnvOn(llgoDbgSyms, false)
}

func IsDbgSymsEnabled() bool {
	return isEnvOn(llgoDbgSyms, false)
}

func IsOptimizeEnabled() bool {
	return isEnvOn(llgoOptimize, true)
}

func effectiveOptLevel(conf *Config) optlevel.Level {
	if conf != nil && conf.OptLevel.IsValid() {
		return conf.OptLevel
	}
	if conf != nil && conf.Target != "" {
		return optlevel.Oz
	}
	return optlevel.O2
}

func llvmPassPipeline(level optlevel.Level) string {
	return "default<" + level.Name() + ">"
}

func IsWasiThreadsEnabled() bool {
	return isEnvOn(llgoWasiThreads, true)
}

func IsFullRpathEnabled() bool {
	return isEnvOn(llgoFullRpath, true)
}

func Plan9ASMPkgs() string {
	return defaultEnv(llgoPlan9ASMPkgs, "")
}

func WasmRuntime() string {
	return defaultEnv(llgoWasmRuntime, defaultWasmRuntime)
}

func concatPkgLinkFiles(ctx *context, pkg *packages.Package, verbose bool) (parts []string) {
	llgoPkgLinkFiles(ctx, pkg, func(linkFile string) {
		parts = append(parts, linkFile)
	}, verbose)
	return
}

// const LLGoFiles = "file1; file2; ..."
func llgoPkgLinkFiles(ctx *context, pkg *packages.Package, procFile func(linkFile string), verbose bool) {
	if o := pkg.Types.Scope().Lookup("LLGoFiles"); o != nil {
		val := o.(*types.Const).Val()
		if val.Kind() == constant.String {
			clFiles(ctx, constant.StringVal(val), pkg, procFile, verbose)
		}
	}
}

// files = "file1; file2; ..."
// files = "$(pkg-config --cflags xxx): file1; file2; ..."
func clFiles(ctx *context, files string, pkg *packages.Package, procFile func(linkFile string), verbose bool) {
	dir := filepath.Dir(pkg.GoFiles[0])
	expFile := pkg.ExportFile
	args := make([]string, 0, 16)
	if strings.HasPrefix(files, "$") { // has cflags
		if pos := strings.IndexByte(files, ':'); pos > 0 {
			cflags := xenv.ExpandEnvToArgs(files[:pos])
			files = files[pos+1:]
			args = append(args, cflags...)
		}
	}
	for _, file := range strings.Split(files, ";") {
		cFile := filepath.Join(dir, strings.TrimSpace(file))
		clFile(ctx, args, cFile, expFile, pkg.PkgPath, procFile, verbose)
	}
}

func clFile(ctx *context, args []string, cFile, expFile, pkgPath string, procFile func(linkFile string), verbose bool) {
	baseName := expFile + filepath.Base(cFile)
	ext := filepath.Ext(cFile)

	// default clang++ will use c++ to compile c file,will cause symbol be mangled
	if ext == ".c" {
		args = append(args, "-x", "c")
	}

	// If GenLL is enabled, first emit .ll for debugging, then compile to .o
	printCmds := ctx.shouldPrintCommands(verbose)
	if ctx.buildConf.GenLL {
		llFile := baseName + ".ll"
		llArgs := append(slices.Clone(args), "-emit-llvm", "-S", "-o", llFile, "-c", cFile)
		if printCmds {
			fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", llFile, pkgPath)
			fmt.Fprintln(os.Stderr, "clang", llArgs)
		}
		cmd := ctx.compiler()
		err := cmd.Compile(llArgs...)
		check(err)
	}

	// Always compile to .o for linking
	objFile := baseName + ".o"
	objArgs := append(args, "-o", objFile, "-c", cFile)
	if printCmds {
		fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", objFile, pkgPath)
		fmt.Fprintln(os.Stderr, "clang", objArgs)
	}
	cmd := ctx.compiler()
	err := cmd.Compile(objArgs...)
	check(err)
	procFile(objFile)
}

func pkgExists(initial []*packages.Package, pkg *packages.Package) bool {
	for _, v := range initial {
		if v == pkg {
			return true
		}
	}
	return false
}

type none struct{}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// -----------------------------------------------------------------------------
