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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/go/ssa"

	"github.com/goplus/llgo/cl"
	llabi "github.com/goplus/llgo/internal/abi"
	"github.com/goplus/llgo/internal/buildenv"
	"github.com/goplus/llgo/internal/cabi"
	"github.com/goplus/llgo/internal/clang"
	"github.com/goplus/llgo/internal/crosscompile"
	"github.com/goplus/llgo/internal/dcepass"
	"github.com/goplus/llgo/internal/deadcode"
	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/internal/firmware"
	"github.com/goplus/llgo/internal/flash"
	"github.com/goplus/llgo/internal/goembed"
	"github.com/goplus/llgo/internal/header"
	"github.com/goplus/llgo/internal/lto"
	"github.com/goplus/llgo/internal/meta"
	"github.com/goplus/llgo/internal/mockable"
	"github.com/goplus/llgo/internal/monitor"
	"github.com/goplus/llgo/internal/optlevel"
	"github.com/goplus/llgo/internal/packages"
	"github.com/goplus/llgo/internal/pclnmap"
	"github.com/goplus/llgo/internal/pclnpost"
	"github.com/goplus/llgo/internal/typepatch"
	"github.com/goplus/llgo/ssa/abi"
	xenv "github.com/goplus/llgo/xtool/env"
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
	Out  string // Base output file path
	PCLN string // PCLN sidecar output file path (.pclntab)
	Bin  string // Binary output file path (.bin)
	Hex  string // Intel hex output file path (.hex)
	Img  string // Image output file path (.img)
	Uf2  string // UF2 output file path (.uf2)
	Zip  string // ZIP/DFU output file path (.zip)
}

// ModuleHook observes a package module immediately after it is generated and
// before TransformModule mutates it. The callback runs synchronously and
// receives the live llvm.Module, so callers that need a stable snapshot should
// consume it immediately (for example, by calling mod.String() inside the
// hook).
type ModuleHook func(pkg Package)

type Config struct {
	Goos               string
	Goarch             string
	Target             string // target name (e.g., "rp2040", "wasi") - takes precedence over Goos/Goarch
	OptLevel           optlevel.Level
	LTO                lto.Mode
	LTOPlugin          lto.PassPlugin
	BinPath            string
	AppExt             string  // ".exe" on Windows, empty on Unix
	OutFile            string  // only valid for ModeBuild when len(pkgs) == 1
	OutFmts            OutFmts // Output format specifications (only for Target != "")
	CompileOnly        bool    // compile test binary but do not run it (only valid for ModeTest)
	Emulator           bool    // run in emulator mode
	Port               string  // target port for flashing
	BaudRate           int     // baudrate for serial communication
	RunArgs            []string
	Mode               Mode
	BuildMode          BuildMode // Build mode: exe, c-archive, c-shared
	AbiMode            AbiMode
	GenExpect          bool // only valid for ModeCmpTest
	Verbose            bool
	PrintPackages      bool // print package paths as they are compiled, like go build -v
	PrintCommands      bool
	GenLL              bool // generate pkg .ll files
	DeadcodeDrop       bool // enable Go dead code drop (development builds only)
	CollectPackageMeta bool // collect package metadata without enabling dead code drop
	CheckLLFiles       bool // check .ll files valid
	CheckLinkArgs      bool // check linkargs valid
	ForceEspClang      bool // force to use esp-clang
	ForceRebuild       bool // force rebuilding of packages that are already up-to-date
	Tags               string
	SizeReport         bool   // print size report after successful build
	SizeFormat         string // size report format: text,json (default text)
	SizeLevel          string // size aggregation level: full,module,package (default module)
	CompilerHash       string // metadata hash for the running compiler (development builds only)
	GoVersion          string // Go language version accepted by the frontend (for example, "go1.22")
	NoErrorColumn      bool   // omit source columns from frontend diagnostics
	DebugTypeAssert    bool   // report type-assertion lowering decisions
	// GoBuildFlags contains normalized raw Go build flags forwarded to
	// go/packages. Callers use internal/goflags to parse supported compiler and
	// linker semantics into typed Config fields before calling Do.
	GoBuildFlags []string
	// BuildParallelism is the package-level concurrency requested by Go's -p
	// build flag for llgo test. Zero uses the Go default, GOMAXPROCS.
	BuildParallelism int
	LinkOptions      LinkOptions
	// OmitDWARFByDefault controls linked builds only when -w was not
	// explicitly specified. Explicit -w and -w=false always win.
	OmitDWARFByDefault bool
	PCLNMode           PCLNMode
	// PCLNModeSet marks PCLNMode as authoritative. Command flags set it for
	// explicit requests; Do sets it after resolving the legacy environment
	// default.
	PCLNModeSet bool
	AllowNoBody bool // allow declarations without bodies, as go tool compile does
	// DisableBoundsChecks disables index, slice, and slice-to-array conversion
	// bounds checks while retaining required integer conversions and nil checks.
	DisableBoundsChecks bool

	// PthreadStackSize sets a custom stack size, in bytes, for pthread-backed
	// goroutines. A zero value keeps the platform pthread default.
	PthreadStackSize int64

	// DisableGoGlobalDCE disables Go-specific global DCE metadata emission
	// when it would otherwise be enabled by full LTO.
	DisableGoGlobalDCE bool

	// GlobalRewrites specifies compile-time overrides for global string variables.
	// Keys are fully qualified package paths (e.g. "main" or "github.com/user/pkg").
	// Each Rewrites entry maps variable names to replacement string values. Only
	// string-typed globals are supported and "main" applies to all root main
	// packages in the current build.
	GlobalRewrites map[string]Rewrites
	ModuleHook     ModuleHook
	Overlay        map[string][]byte
}

type Rewrites map[string]string

// clone returns an independent copy of c for use by a single build. Do
// resolves defaults and target-specific values on this copy so callers can
// safely reuse their input configuration after Do returns.
func (c *Config) clone() *Config {
	if c == nil {
		return nil
	}
	cloned := *c
	cloned.RunArgs = slices.Clone(c.RunArgs)
	cloned.GoBuildFlags = slices.Clone(c.GoBuildFlags)
	cloned.Overlay = cloneOverlay(c.Overlay)
	if c.GlobalRewrites != nil {
		cloned.GlobalRewrites = make(map[string]Rewrites, len(c.GlobalRewrites))
		for pkgPath, rewrites := range c.GlobalRewrites {
			if rewrites == nil {
				cloned.GlobalRewrites[pkgPath] = nil
				continue
			}
			copied := make(Rewrites, len(rewrites))
			for name, value := range rewrites {
				copied[name] = value
			}
			cloned.GlobalRewrites[pkgPath] = copied
		}
	}
	return &cloned
}

// resolveBuildConfig validates and fills build-local defaults without
// modifying the caller's Config. Target-derived GOOS/GOARCH values are
// resolved later, after crosscompile.Use has selected the toolchain.
func resolveBuildConfig(input *Config) (*Config, error) {
	if input == nil {
		return nil, errors.New("build config must not be nil")
	}
	conf := input.clone()
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
	if conf.BuildMode != BuildModeExe {
		conf.DeadcodeDrop = false
	}
	conf.PCLNMode = effectivePCLNMode(conf)
	conf.PCLNModeSet = true
	if conf.SizeReport && conf.SizeFormat == "" {
		conf.SizeFormat = "text"
	}
	if conf.SizeReport && conf.SizeLevel == "" {
		conf.SizeLevel = "module"
	}
	if err := validatePCLNMode(conf); err != nil {
		return nil, err
	}
	if err := ensureSizeReporting(conf); err != nil {
		return nil, err
	}
	if err := conf.LinkOptions.validate(); err != nil {
		return nil, err
	}
	conf.OptLevel = effectiveOptLevel(conf)
	return conf, nil
}

func NewDefaultConf(mode Mode) *Config {
	bin := os.Getenv("GOBIN")
	if bin == "" {
		gopath, err := envGOPATH()
		if err != nil {
			panic(fmt.Errorf("cannot get GOPATH: %v", err))
		}
		bin = filepath.Join(gopath, "bin")
	}
	goos, goarch := os.Getenv("GOOS"), os.Getenv("GOARCH")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	conf := &Config{
		Goos:               goos,
		Goarch:             goarch,
		BinPath:            bin,
		Mode:               mode,
		BuildMode:          BuildModeExe,
		AbiMode:            cabi.ModeAllFunc,
		OmitDWARFByDefault: mode != ModeGen,
		PCLNMode:           PCLNEmbedded,
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

func (c *Config) ltoMode() lto.Mode {
	if c == nil {
		return lto.Off
	}
	return c.LTO
}

func (c *Config) ltoEnabled() bool {
	return c.ltoMode().Enabled()
}

func (c *Config) goGlobalDCEEnabled() bool {
	if c == nil {
		return false
	}
	return buildenv.Dev && c.ltoMode() == lto.Full && !c.DisableGoGlobalDCE
}

func (c *Config) deadcodeDropEnabled() bool {
	return buildenv.Dev && c.DeadcodeDrop && !c.goGlobalDCEEnabled()
}

func (c *Config) packageMetaEnabled() bool {
	return c.CollectPackageMeta || c.deadcodeDropEnabled()
}

func (c *Config) parallelism() int {
	if c != nil && c.BuildParallelism > 0 {
		return c.BuildParallelism
	}
	return max(1, runtime.GOMAXPROCS(0))
}

// -----------------------------------------------------------------------------

const (
	loadFiles   = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles
	loadImports = loadFiles | packages.NeedImports
	loadTypes   = loadImports | packages.NeedTypes | packages.NeedTypesSizes
	loadSyntax  = loadTypes | packages.NeedSyntax | packages.NeedTypesInfo
)

var llssaInitOnce sync.Once

func Do(args []string, conf *Config) ([]Package, error) {
	return Build(Invocation{Args: args, Config: conf})
}

// Build executes one build invocation.
func Build(inv Invocation) ([]Package, error) {
	dir := inv.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	environ := os.Environ()
	commands := commandEnv{dir: dir, environ: environ}
	conf, err := resolveBuildConfig(inv.Config)
	if err != nil {
		return nil, err
	}
	// Handle crosscompile configuration first to set correct GOOS/GOARCH
	forceEspClang := conf.ForceEspClang || conf.Target != ""
	export, err := crosscompile.Use(conf.Goos, conf.Goarch, conf.Target, IsWasiThreadsEnabled(), forceEspClang, conf.OptLevel, conf.ltoMode(), conf.goGlobalDCEEnabled())
	if err != nil {
		return nil, fmt.Errorf("failed to setup crosscompile: %w", err)
	}
	applyBuildModeCompileFlags(conf.BuildMode, &export)
	// Update GOOS/GOARCH from export if target was used
	if conf.Target != "" && export.GOOS != "" {
		conf.Goos = export.GOOS
	}
	if conf.Target != "" && export.GOARCH != "" {
		conf.Goarch = export.GOARCH
	}
	if err := validateLinkOptions(conf, &export); err != nil {
		return nil, err
	}
	verbose := conf.Verbose
	patterns := slices.Clone(inv.Args)
	target := &llssa.Target{
		GOOS:       conf.Goos,
		GOARCH:     conf.Goarch,
		Target:     conf.Target,
		LLVMTarget: export.LLVMTarget,
		OptLevel:   conf.OptLevel,
	}
	tags := defaultBuildTags(conf.Goarch, conf.Target) + "," + target.ClosureEnvBuildTag()
	if conf.PCLNMode == PCLNExternal {
		// Select the optional runtime loader as part of the normal package
		// cache key. Embedded and none builds do not compile any loader or
		// sidecar probing code.
		tags += ",llgo_pclntab_external"
	}
	if conf.AbiMode == cabi.ModeAllFunc {
		tags += ",llgo_abi_2"
	}
	if conf.Tags != "" {
		tags += "," + conf.Tags
	}
	if len(export.BuildTags) > 0 {
		tags += "," + strings.Join(export.BuildTags, ",")
	}
	goBuildFlags := []string{"-tags=" + tags}
	goBuildFlags = append(goBuildFlags, conf.GoBuildFlags...)
	cfg := &packages.Config{
		Mode:       loadSyntax | packages.NeedDeps | packages.NeedModule | packages.NeedExportFile,
		BuildFlags: goBuildFlags,
		Dir:        dir,
		Fset:       token.NewFileSet(),
		Tests:      conf.Mode == ModeTest,
		Env:        withEnv(environ, "GOOS="+conf.Goos, "GOARCH="+conf.Goarch),
	}
	if conf.Mode == ModeTest {
		cfg.Mode |= packages.NeedForTest
	}
	emitDebugInfo := shouldEmitDebugInfo(conf, &export)
	frontendOptions := cl.Options{
		Debug:           emitDebugInfo,
		DebugSymbols:    emitDebugInfo,
		DebugTypeAssert: conf.DebugTypeAssert,
		NoErrorColumn:   conf.NoErrorColumn,
		Trace:           IsTraceEnabled(),
		ExportRename:    conf.Target != "",
		ShadowStack:     isEnvOn(llgoShadowStack, false),
	}
	llssaInitOnce.Do(func() {
		llssa.Initialize(llssa.InitAll)
	})

	prog := llssa.NewProgram(target)
	prog.DisableBoundsChecks(conf.DisableBoundsChecks)
	if conf.Mode != ModeGen {
		// ModeGen callers (llgen and the golden suites) read LPkg.String()
		// after Do returns and dispose the program themselves; every other
		// mode's outputs are files or a spawned process, so the compile's
		// LLVM context can be released when Do finishes. In-process
		// drivers that build many packages per process (the cltest run
		// harness) otherwise accumulate every compile's C++-side memory.
		defer prog.Dispose()
	}
	prog.EnableGoGlobalDCE(conf.goGlobalDCEEnabled())
	prog.EnableDeadcodeDrop(conf.deadcodeDropEnabled())
	if conf.PthreadStackSize > 0 {
		prog.SetPthreadStackSize(uint64(conf.PthreadStackSize))
	}
	prog.EnableLTOPluginMarkers(conf.LTOPlugin.Enabled())
	funcInfo := conf.Mode != ModeGen && conf.PCLNMode != PCLNNone
	prog.EnableFuncInfoMetadata(funcInfo)
	// Site records are inline-asm fragments inside function bodies. Darwin
	// DWARF builds avoid them because they disturb LLDB lexical scopes; Linux
	// still needs them because its restricted dynamic symbol table cannot
	// reconstruct every Go entry PC through dlsym. External mode always needs
	// final-PC sites for sidecar construction.
	prog.EnableFuncInfoSites(shouldEnablePCLNSites(conf, funcInfo, emitDebugInfo))
	sizes := func(sizes types.Sizes, compiler, arch string) types.Sizes {
		if arch == "wasm" {
			sizes = &types.StdSizes{WordSize: 4, MaxAlign: 4}
		}
		return prog.TypeSizes(sizes)
	}
	dedup := packages.NewDeduper()
	var syntaxErr error
	var syntaxErrMu sync.Mutex
	recordSyntaxErr := func(err error) {
		syntaxErrMu.Lock()
		defer syntaxErrMu.Unlock()
		if syntaxErr == nil {
			syntaxErr = err
		}
	}
	loadSyntaxErr := func() error {
		syntaxErrMu.Lock()
		defer syntaxErrMu.Unlock()
		return syntaxErr
	}
	dedup.SetPreload(func(pkg *types.Package, files []*ast.File) {
		if llruntime.SkipToBuild(pkg.Path()) {
			return
		}
		if err := cl.ParsePkgSyntax(prog, cfg.Fset, pkg, files); err != nil {
			recordSyntaxErr(err)
		}
	})

	if patterns == nil {
		patterns = []string{"."}
	}
	sourcePatchGOROOT, sourcePatchGoVersion, err := env.GOROOTAndGOVERSIONWithEnv(cfg.Env)
	if err != nil {
		return nil, err
	}
	var llgoFiles map[string][]string
	conf.Overlay, llgoFiles, err = buildSourcePatchOverlayForGOROOT(conf.Overlay, env.LLGoRuntimeDir(), sourcePatchGOROOT, sourcePatchBuildContext{
		goos:       conf.Goos,
		goarch:     conf.Goarch,
		goversion:  sourcePatchGoVersion,
		buildFlags: cfg.BuildFlags,
	})
	if err != nil {
		return nil, err
	}
	dedup.SetLLGoFiles(llgoFiles)
	cfg.ParseFile = func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		if data, ok := conf.Overlay[filename]; ok {
			src = data
		}
		// We implicitly promise to keep doing ast.Object resolution. :(
		const mode = parser.AllErrors | parser.ParseComments
		return parser.ParseFile(fset, filename, src, mode)
	}

	initial, err := packages.LoadExWithGoVersion(dedup, sizes, cfg, conf.GoVersion, patterns...)
	if err != nil {
		return nil, err
	}
	if err := loadSyntaxErr(); err != nil {
		return nil, err
	}
	if conf.AllowNoBody {
		allowMissingFunctionBodies(initial)
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
	altCfg := *cfg
	altCfg.Dir = env.LLGoRuntimeDir()
	altPkgs, err := packages.LoadEx(dedup, sizes, &altCfg, altPkgPaths...)
	if err != nil {
		return nil, err
	}
	if err := loadSyntaxErr(); err != nil {
		return nil, err
	}

	prog.SetRuntime(func() *types.Package {
		return altPkgs[0].Types
	})
	prog.SetPython(func() *types.Package {
		return dedup.Check(llssa.PkgPython).Types
	})
	if err := prepareLocalVariables(prog, initial, altPkgs); err != nil {
		return nil, err
	}

	buildMode := ssaBuildMode
	cabiOptimize := true
	passOpt := shouldRunLLVMPasses(mode)
	if emitDebugInfo {
		buildMode |= ssa.GlobalDebug
		cabiOptimize = false
	}
	if !IsOptimizeEnabled() {
		buildMode |= ssa.NaiveForm
	}
	prog.SetDebugInfoOptimized(passOpt && conf.OptLevel != optlevel.O0)
	progSSA := ssa.NewProgram(initial[0].Fset, buildMode)
	patches := make(cl.Patches, len(altPkgPaths))
	altEntries := registerAltSSAPkgs(progSSA, patches, altPkgs[1:], conf, verbose)

	output := conf.OutFile != ""
	ctx := &context{conf: cfg, progSSA: progSSA, prog: prog, dedup: dedup,
		patches: patches, callerTracking: cl.NewCallerTracking(),
		built: make(map[string]none), initial: initial, mode: mode,
		fingerprinting:  make(map[string]bool),
		pkgs:            map[*packages.Package]Package{},
		pkgByID:         map[string]Package{},
		output:          output,
		passOpt:         passOpt,
		buildConf:       conf,
		crossCompile:    export,
		commands:        commands,
		frontendOptions: frontendOptions,
		cTransformer:    cabi.NewTransformer(prog, export.LLVMTarget, export.TargetABI, conf.AbiMode, cabiOptimize),
	}
	defer ctx.closePackageMetas()
	defer ctx.closePackageArchiveBuffers()

	// default runtime globals must be registered before packages are built
	addGlobalString(conf, "runtime.defaultGOROOT="+runtime.GOROOT(), nil)
	addGlobalString(conf, "runtime.buildVersion="+runtime.Version(), nil)
	pkgs, pkgEntries, err := registerSSAPkgs(ctx, initial, verbose)
	if err != nil {
		return nil, err
	}
	depPkgs, depEntries, err := registerSSAPkgs(ctx, altPkgs, verbose)
	if err != nil {
		return nil, err
	}
	buildSSAPkgs(ctx, append(append(altEntries, pkgEntries...), depEntries...))

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
			resolveOutputs(ctx.commands.dir, outFmts)

			// Link main package using the output path from buildOutFmts
			err = linkMainPkg(ctx, pkg, allPkgs, outFmts.Out, verbose)
			if err != nil {
				return nil, err
			}
			if err := finalizeRuntimePCLN(ctx, outFmts, verbose); err != nil {
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
				pkgs := cHeaderPackages(allPkgs)
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
					err = runInEmulator(ctx.commands, ctx.crossCompile.Emulator, envMap, pkg.Dir, pkg.PkgPath, conf, mode, verbose)
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

// cHeaderPackages excludes the patched standard runtime implementation. Its
// //export callbacks are linker implementation details and may use internal C
// types that are deliberately not representable in a public generated header.
func cHeaderPackages(allPkgs []*aPackage) []llssa.Package {
	pkgs := make([]llssa.Package, 0, len(allPkgs))
	for _, pkg := range allPkgs {
		if pkg == nil || pkg.LPkg == nil || pkg.Package == nil || pkg.PkgPath == "runtime" || isRuntimePkg(pkg.PkgPath) || !hasLocalCExports(pkg.LPkg) {
			continue
		}
		pkgs = append(pkgs, pkg.LPkg)
	}
	return pkgs
}

func hasLocalCExports(pkg llssa.Package) bool {
	if pkg == nil {
		return false
	}
	for name := range pkg.ExportFuncs() {
		if !strings.Contains(name, ".") || strings.HasPrefix(name, pkg.Path()+".") {
			return true
		}
	}
	return false
}

// applyBuildModeCompileFlags adds code-generation flags that must be present
// while package C/C++ sources are compiled. Passing -fPIC only to the final
// shared-library link is too late for objects containing global references.
func applyBuildModeCompileFlags(mode BuildMode, export *crosscompile.Export) {
	if mode == BuildModeCShared && export != nil && !slices.Contains(export.CCFLAGS, "-fPIC") {
		export.CCFLAGS = append(export.CCFLAGS, "-fPIC")
	}
}

// DefaultBuildTags returns the build tags LLGo always enables for a target.
func DefaultBuildTags(goarch, target string) string {
	return defaultBuildTags(goarch, target)
}

func defaultBuildTags(goarch, target string) string {
	tags := "llgo,math_big_pure_go,purego"
	// Raw GOOS/GOARCH wasm builds do not have a target configuration that
	// selects a collector. BDWGC is not available in either wasm host, so use
	// the supported collector-free runtime unless a named target supplies its
	// own runtime configuration.
	if goarch == "wasm" && target == "" {
		tags += ",nogc"
	}
	return tags
}

func allowMissingFunctionBodies(initial []*packages.Package) {
	for _, pkg := range initial {
		hasMissingBody := false
		hasOtherError := false
		for _, pkgErr := range pkg.Errors {
			switch {
			case strings.Contains(pkgErr.Msg, "missing function body"):
				hasMissingBody = true
			case strings.HasPrefix(pkgErr.Msg, "# "):
				// go list prefixes compiler diagnostics with the package name.
			default:
				hasOtherError = true
			}
		}
		if hasMissingBody && !hasOtherError {
			pkg.Errors = nil
			pkg.TypeErrors = nil
			pkg.IllTyped = false
		}
	}
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
		if pkg.Types != nil && pkg.Types.Name() == "main" {
			pkg.Types.SetName("main.test")
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

const (
	ssaBuildMode = ssa.SanityCheckFunctions | ssa.InstantiateGenerics
)

type context struct {
	conf           *packages.Config
	progSSA        *ssa.Program
	prog           llssa.Program
	dedup          packages.Deduper
	patches        cl.Patches
	callerTracking *cl.CallerTracking
	built          map[string]none
	fingerprinting map[string]bool
	cacheDisabled  map[string]none
	initial        []*packages.Package
	pkgs           map[*packages.Package]Package // cache for lookup
	pkgByID        map[string]Package            // cache for lookup by pkg.ID
	mode           Mode
	nLibdir        int32
	output         bool
	passOpt        bool

	buildConf       *Config
	crossCompile    crosscompile.Export
	commands        commandEnv
	frontendOptions cl.Options

	cTransformer *cabi.Transformer

	testFail bool

	// Cache related fields
	cacheManager *cacheManager
	llvmVersion  string

	// go list derived file lists (SFiles, etc.)
	sfilesCache map[string][]string // pkg.ID -> absolute .s/.S file paths

	// plan9asm package policy parsed from env.
	plan9asmOnce sync.Once
	plan9asmMode plan9asmPkgsEnvMode
	plan9asmPkgs map[string]bool

	// pclnExternal is populated while generating the synthetic main module
	// and completed with final linked PCs by the post-link externalizer.
	pclnExternal *pclnmap.Data
}

// closePackageMetas releases metadata mappings owned by this build. Metadata
// remains available to hooks and whole-program consumers until Do returns.
func (c *context) closePackageMetas() {
	for _, pkg := range c.pkgs {
		if pkg.Meta == nil {
			continue
		}
		_ = pkg.Meta.Close()
		pkg.Meta = nil
	}
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
	cmd.Dir = c.commands.dir
	cmd.Env = slices.Clone(c.commands.environ)
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
	cmd.Dir = c.commands.dir
	cmd.Env = slices.Clone(c.commands.environ)
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

// normalizeToArchive creates an archive from file and memory members and sets ArchiveFile.
// This ensures the link step always consumes .a archives regardless of cache state.
func normalizeToArchive(ctx *context, aPkg *aPackage, verbose bool) error {
	if len(aPkg.ObjFiles) == 0 && len(aPkg.ObjBuffers) == 0 {
		return nil
	}
	defer aPkg.disposeArchiveBuffers()

	archiveFile, err := os.CreateTemp("", "pkg-*.a")
	if err != nil {
		return fmt.Errorf("create temp archive: %w", err)
	}
	archiveFile.Close()
	archivePath := archiveFile.Name()

	if err := ctx.createPackageArchiveFile(archivePath, aPkg, verbose); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("create archive for %s: %w", aPkg.PkgPath, err)
	}

	aPkg.ObjFiles = nil
	aPkg.ArchiveFile = archivePath
	return nil
}

func buildAllPkgs(ctx *context, pkgs []*aPackage, verbose bool) ([]*aPackage, error) {
	// Split packages into runtime tree vs others so we can defer runtime build.
	var runtimePkgs []*packageBuildTask
	var normalPkgs []*packageBuildTask
	for _, p := range pkgs {
		task := newPackageBuildTask(p)
		if task.isRuntime() {
			runtimePkgs = append(runtimePkgs, task)
		} else {
			normalPkgs = append(normalPkgs, task)
		}
	}

	var needRuntime, needPyInit bool

	// Build non-runtime packages first, so we know whether runtime is actually needed.
	for _, task := range normalPkgs {
		result, err := buildOnePackage(ctx, task, verbose)
		if err != nil {
			return nil, err
		}
		needRuntime = needRuntime || result.needRuntime
		needPyInit = needPyInit || result.needPyInit
	}

	// Only build runtime packages when required (or host build with empty Target).
	if needRuntime || needPyInit || ctx.buildConf.Target == "" {
		for _, task := range runtimePkgs {
			if _, err := buildOnePackage(ctx, task, verbose); err != nil {
				return nil, err
			}
		}
	}

	return pkgs, nil
}

// buildOnePackage is the serial package pipeline. Its explicit stages are the
// contract used by later package workers; this commit deliberately preserves
// serial LLVM execution.
func buildOnePackage(ctx *context, task *packageBuildTask, verbose bool) (packageBuildResult, error) {
	if err := prePackageBuild(ctx, task, verbose); err != nil || task.skip {
		return packageBuildResultFor(task), err
	}
	if err := executePackageBuild(ctx, task, verbose); err != nil {
		return packageBuildResultFor(task), err
	}
	return finalizePackageBuild(ctx, task, verbose)
}

// prePackageBuild performs classification, fingerprinting, and cache
// lookup without creating or transforming an LLVM module.
func prePackageBuild(ctx *context, task *packageBuildTask, verbose bool) error {
	aPkg := task.pkg
	pkg := aPkg.Package
	if _, ok := ctx.built[pkg.ID]; ok {
		task.skip = true
		return nil
	}
	ctx.built[pkg.ID] = none{}
	if task.isDeclOnly() {
		pkg.ExportFile = ""
		task.skip = true
		return nil
	}
	if task.isLinkOnly() && !task.hasSource() {
		pkg.ExportFile = ""
		if task.kind == cl.PkgLinkExtern {
			appendExternalLinkArgs(ctx, aPkg, task.kindParam)
		}
		task.skip = true
		return nil
	}
	if err := ctx.collectFingerprint(aPkg); err != nil {
		return err
	}
	ctx.tryLoadFromCache(aPkg)
	if verbose {
		status := "MISS"
		if aPkg.CacheHit {
			status = "HIT"
		}
		fmt.Fprintf(os.Stderr, "CACHE %s: %s\n", status, pkg.PkgPath)
	}
	return nil
}

// executePackageBuild creates the package module and runs its LLVM backend.
func executePackageBuild(ctx *context, task *packageBuildTask, verbose bool) error {
	aPkg := task.pkg
	if err := buildPkg(ctx, aPkg, verbose); err != nil {
		return err
	}
	if task.needsRuntimeSignals() {
		aPkg.setNeedRuntimeOrPyInit(aPkg.LPkg.NeedRuntime, aPkg.LPkg.NeedPyInit)
	}
	return nil
}

// finalizePackageBuild publishes the archive and cache metadata. Cache hits
// already carry both and therefore require no publication.
func finalizePackageBuild(ctx *context, task *packageBuildTask, verbose bool) (packageBuildResult, error) {
	aPkg := task.pkg
	if aPkg.CacheHit {
		return packageBuildResultFor(task), nil
	}
	if err := normalizeToArchive(ctx, aPkg, verbose); err != nil {
		return packageBuildResultFor(task), err
	}
	if task.kind == cl.PkgLinkExtern {
		appendExternalLinkArgs(ctx, aPkg, task.kindParam)
	}
	if err := ctx.saveToCache(aPkg); err != nil && verbose {
		fmt.Fprintf(os.Stderr, "warning: failed to save cache for %s: %v\n", aPkg.PkgPath, err)
	}
	return packageBuildResultFor(task), nil
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

// rewritePrebuiltFuncTab runs the link-phase prebuilt-table rewrite on the
// linked executable: it deduplicates LTO inline copies of the funcinfo entry
// records against the symbol table and replaces the entry section with a
// sorted ftab plus findfunctab that the runtime adopts zero-copy (see
// internal/pclnpost and doc/design/pclntab-linkphase.md). Any failure leaves
// the binary fully functional on the first-use construction fallback.
func rewritePrebuiltFuncTab(ctx *context, out string, verbose bool) {
	if ctx == nil || ctx.prog == nil || !ctx.prog.FuncInfoSitesEnabled() || !shouldEmitRuntimeSites(ctx) {
		return
	}
	if ctx.buildConf.BuildMode != BuildModeExe {
		return
	}
	if os.Getenv("LLGO_PCLNPOST") == "0" { // escape hatch: keep first-use construction
		return
	}
	st, err := pclnpost.Rewrite(out)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "llgo: prebuilt functab rewrite skipped: %v\n", err)
		}
		return
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "llgo: prebuilt functab: %d entries (%d LTO inline copies removed), %d buckets\n",
			st.FtabEntries, st.InlineCopies, st.Buckets)
	}
}

func linkMainPkg(ctx *context, pkg *packages.Package, pkgs []*aPackage, outputPath string, verbose bool) error {
	ctx.pclnExternal = nil
	needRuntime := false
	needPyInit := false
	var needAbiInit int
	methodByIndex := make(map[int]none)
	methodByName := make(map[string]none)
	allPkgs := []*packages.Package{pkg}
	for _, v := range pkgs {
		if v.PkgPath != pkg.PkgPath && v.Types != nil && v.Types.Name() == "main" {
			continue
		}
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
	var funcInfo []funcInfoRecord
	var pcLineInfo []pcLineRecord
	if ctx.buildConf.PCLNMode != PCLNNone {
		funcInfo = prepareFuncInfoTableRecords(collectFuncInfo(linkedOrder), nil)
		pcLineInfo = collectPCLineInfo(linkedOrder)
	}
	entryPkg := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{
		rtInit:        needRuntime,
		pyInit:        needPyInit,
		abiInit:       needAbiInit,
		methodByIndex: methodByIndex,
		methodByName:  methodByName,
		abiSymbols:    linkedModuleGlobals(linkedOrder),
		funcInfo:      funcInfo,
		pcLineInfo:    pcLineInfo,
	})
	if ctx.buildConf.deadcodeDropEnabled() {
		if err := applyDeadcodeDropOverrides(linkedOrder, entryPkg, needRuntime, verbose); err != nil {
			return err
		}
	}
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
	linkArgs = append(linkArgs, cSharedExportArgs(ctx, linkedOrder)...)

	err = linkObjFiles(ctx, outputPath, linkInputs, linkArgs, verbose)
	if err != nil {
		return err
	}

	return nil
}

func linkedPackageMetas(pkgs []Package) []*meta.PackageMeta {
	metas := make([]*meta.PackageMeta, 0, len(pkgs))
	for _, pkg := range pkgs {
		metas = append(metas, pkg.Meta)
	}
	return metas
}

func applyDeadcodeDropOverrides(pkgs []Package, entryPkg Package, needRuntime bool, verbose bool) error {
	metas := linkedPackageMetas(pkgs)
	summary, err := meta.NewGlobalSummary(metas)
	if err != nil {
		return err
	}

	roots := dceEntryRootCandidates(pkgs, needRuntime)
	liveSlots := deadcode.Analyze(summary, roots)
	dcepass.EmitStrongTypeOverrides(entryPkg.LPkg.Module(), dceSourceModules(pkgs), liveSlots, verbose)
	return nil
}

func dceSourceModules(pkgs []Package) []gllvm.Module {
	mods := make([]gllvm.Module, 0, len(pkgs))
	for _, pkg := range pkgs {
		mods = append(mods, pkg.LPkg.Module())
	}
	return mods
}

func dceEntryRootCandidates(pkgs []Package, needRuntime bool) []string {
	roots := []string{"main.init", "main.main"}
	// C code can call //export functions without an ordinary edge from a Go
	// root, so their final linker names must seed the analysis explicitly.
	var exports []string
	for _, pkg := range pkgs {
		for _, name := range pkg.LPkg.ExportFuncs() {
			exports = append(exports, name)
		}
	}
	slices.Sort(exports)
	roots = append(roots, exports...)
	if needRuntime {
		roots = append(roots, llssa.PkgRuntime+".init")
	}
	return roots
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
			if g.IsDeclaration() {
				continue
			}
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
	if dir := filepath.Dir(app); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory %s: %w", dir, err)
		}
	}
	// Handle c-archive mode differently - use ar tool instead of linker
	if ctx.buildConf.BuildMode == BuildModeCArchive {
		return ctx.createMergedArchiveFile(app, objFiles, printCmds)
	}

	buildArgs := []string{"-o", app}
	buildArgs = append(buildArgs, linkArgs...)
	buildArgs = append(buildArgs, dwarfLinkerArgs(ctx.buildConf, &ctx.crossCompile)...)
	ltoPluginFlags, err := ctx.buildConf.LTOPlugin.LinkerFlags(ctx.buildConf.Goos)
	if err != nil {
		return err
	}
	buildArgs = append(buildArgs, ltoPluginFlags...)

	// Add build mode specific linker arguments
	switch ctx.buildConf.BuildMode {
	case BuildModeCShared:
		buildArgs = append(buildArgs, "-shared", "-fPIC")
	case BuildModeExe:
		if needsLinuxNoPIE(ctx, linkArgs) {
			buildArgs = append(buildArgs, "-no-pie")
		}
		buildArgs = append(buildArgs, linuxExportDynamicArgs(ctx)...)
	}

	if shouldEmitDebugInfo(ctx.buildConf, &ctx.crossCompile) {
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

// cSharedExportArgs keeps //export functions and synthetic test entry points as
// shared-library link roots. They live in package archives and otherwise remain
// unreferenced, so the linker can omit both their object files and symbols.
func cSharedExportArgs(ctx *context, pkgs []*aPackage) []string {
	if ctx == nil || ctx.buildConf == nil || ctx.buildConf.BuildMode != BuildModeCShared {
		return nil
	}
	exports := make(map[string]none)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.LPkg == nil {
			continue
		}
		for _, name := range pkg.LPkg.ExportFuncs() {
			if name != "" {
				exports[name] = none{}
			}
		}
		if ctx.mode == ModeTest && pkg.Package != nil && pkg.Name == "main" && strings.HasSuffix(pkg.PkgPath, ".test") {
			exports[pkg.PkgPath+".init"] = none{}
			exports[pkg.PkgPath+".main"] = none{}
		}
	}
	names := make([]string, 0, len(exports))
	for name := range exports {
		names = append(names, name)
	}
	slices.Sort(names)
	args := make([]string, 0, len(names))
	for _, name := range names {
		if ctx.buildConf.Goos == "darwin" {
			args = append(args, "-Wl,-u,_"+name)
		} else {
			args = append(args, "-Wl,--undefined="+name)
		}
	}
	return args
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

func needsLinuxExportDynamic(ctx *context) bool {
	return ctx.buildConf.Target == "" && ctx.buildConf.Goos == "linux" && effectivePCLNMode(ctx.buildConf) != PCLNNone
}

func linuxExportDynamicArgs(ctx *context) []string {
	if !needsLinuxExportDynamic(ctx) {
		return nil
	}
	return []string{
		"-Wl,--export-dynamic-symbol=main.*",
		"-Wl,--export-dynamic-symbol=command-line-arguments.*",
	}
}

// archiver returns the archiving tool to use for the current context.
// For wasm targets and LTO builds, it prefers llvm-ar because linkers need
// LLVM-aware archive indexes for wasm objects and bitcode members.
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
	if c.buildConf.ltoEnabled() || c.buildConf.Goarch == "wasm" || strings.Contains(c.crossCompile.LLVMTarget, "wasm") {
		if llvmAr, err := exec.LookPath("llvm-ar"); err == nil {
			return llvmAr
		}
	}
	return "ar"
}

// archiveMerger returns an archiver with MRI support, which is required to
// flatten package archives into the final c-archive instead of nesting .a
// files as members. LLVM is already a required LLGo toolchain dependency.
func (c *context) archiveMerger() (string, error) {
	if ar := os.Getenv("LLGO_AR"); ar != "" {
		return ar, nil
	}
	if c.crossCompile.CC != "" {
		llvmAr := filepath.Join(filepath.Dir(c.crossCompile.CC), "llvm-ar")
		if _, err := os.Stat(llvmAr); err == nil {
			return llvmAr, nil
		}
	}
	if llvmAr, err := exec.LookPath("llvm-ar"); err == nil {
		return llvmAr, nil
	}
	return "", errors.New("llvm-ar is required to create a flat c-archive")
}

// createMergedArchiveFile combines object files and package archives into one
// flat archive. A regular `ar rcs output.a input.a` stores input.a as a nested
// member, which C linkers cannot search or load.
func (c *context) createMergedArchiveFile(archivePath string, inputs []string, verbose ...bool) error {
	if len(inputs) == 0 {
		return fmt.Errorf("no inputs provided for archive %s", archivePath)
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(archivePath), filepath.Base(archivePath)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	_ = os.Remove(tmpName)

	var script strings.Builder
	fmt.Fprintf(&script, "CREATE %s\n", strconv.Quote(tmpName))
	for _, input := range inputs {
		command := "ADDMOD"
		if strings.HasSuffix(strings.ToLower(input), ".a") {
			command = "ADDLIB"
		}
		fmt.Fprintf(&script, "%s %s\n", command, strconv.Quote(input))
	}
	script.WriteString("SAVE\nEND\n")

	arCmd, err := c.archiveMerger()
	if err != nil {
		return err
	}
	cmd := c.commands.configure(exec.Command(arCmd, "-M"))
	cmd.Stdin = strings.NewReader(script.String())
	printCmds := c.shouldPrintCommands(len(verbose) > 0 && verbose[0])
	if printCmds {
		fmt.Fprintf(os.Stderr, "%s -M\n%s", filepath.Base(arCmd), script.String())
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("merge archive %s: %w\n%s", archivePath, err, output)
	}
	if err := os.Rename(tmpName, archivePath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("publish archive %s: %w", archivePath, err)
	}
	return nil
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
	cmd := c.commands.configure(exec.Command(arCmd, args...))
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
	externs, err := preparePackageModule(ctx, aPkg, verbose)
	if err != nil || aPkg.CacheHit || aPkg.LPkg == nil {
		return err
	}
	return compilePackageModule(ctx, aPkg, externs, verbose)
}

// preparePackageModule runs the frontend and creates the package LLVM module.
func preparePackageModule(ctx *context, aPkg *aPackage, verbose bool) ([]string, error) {
	pkg := aPkg.Package
	pkgPath := pkg.PkgPath
	if debugBuild || verbose {
		fmt.Fprintln(os.Stderr, pkgPath)
	} else {
		printCompiledPackage(ctx.buildConf, aPkg)
	}
	if llruntime.SkipToBuild(pkgPath) {
		pkg.ExportFile = ""
		return nil, nil
	}
	var syntax = pkg.Syntax
	if altPkg := aPkg.AltPkg; altPkg != nil {
		syntax = append(syntax, altPkg.Syntax...)
	}
	initialPkg := pkgExists(ctx.initial, pkg)
	showDetail := verbose && initialPkg
	needMeta := !aPkg.CacheHit && ctx.buildConf.packageMetaEnabled()
	if showDetail {
		fmt.Fprintf(os.Stderr, "==> Compile %s\n", pkgPath)
	}
	embedMap, err := goembed.LoadDirectives(ctx.conf.Fset, syntax)
	if err != nil {
		return nil, fmt.Errorf("load go:embed directives for %s failed: %w", pkgPath, err)
	}
	packageOptions := ctx.frontendOptions
	packageOptions.DebugTypeAssert = initialPkg && packageOptions.DebugTypeAssert
	ret, externs, err := cl.NewPackageExWithEmbedMetaOptions(
		ctx.prog, ctx.callerTracking, ctx.patches, aPkg.rewriteVars,
		aPkg.SSA, syntax, embedMap, needMeta, packageOptions)
	check(err)

	aPkg.LPkg = ret
	if !aPkg.CacheHit {
		aPkg.Meta = ret.Meta
	}
	if hook := ctx.buildConf.ModuleHook; hook != nil {
		hook(aPkg)
	}

	// If cache hit, we only needed to register types - skip compilation
	if aPkg.CacheHit {
		return nil, nil
	}
	return externs, nil
}

// compilePackageModule applies LLVM transforms and emits package objects.
func compilePackageModule(ctx *context, aPkg *aPackage, externs []string, verbose bool) error {
	pkg := aPkg.Package
	pkgPath := pkg.PkgPath
	ret := aPkg.LPkg

	ctx.cTransformer.SetSkipFuncs(cabiSkipFuncsForPlan9Asm(ctx, pkgPath, ret.Module()))
	llabi.LowerLargeAggregates(ctx.prog.TargetData(), ret.Module())
	ctx.cTransformer.TransformModule(ret.Path(), ret.Module())
	ctx.cTransformer.SetSkipFuncs(nil)

	// Run the default LLVM optimization pipeline selected by the requested -O level.
	if ctx.passOpt {
		mod := ret.Module()
		mod.SetDataLayout(ctx.prog.DataLayout())
		mod.SetTarget(ctx.prog.Target().Spec().Triple)
		pbo := gllvm.NewPassBuilderOptions()
		defer pbo.Dispose()
		if err := gllvm.VerifyModule(mod, gllvm.ReturnStatusAction); err != nil {
			return fmt.Errorf("verify LLVM module for %v failed: %w", pkgPath, err)
		}
		if err := mod.RunPasses(llvmPassPipeline(ctx.buildConf.OptLevel, ctx.buildConf.ltoMode()), ctx.prog.TargetMachine(), pbo); err != nil {
			return fmt.Errorf("run LLVM passes failed for %v: %w", pkgPath, err)
		}
	}
	emitFuncInfoEntrySites(ctx, ret)
	// ModeGen callers consume the in-memory LLVM module directly. They do not
	// need cgo/link objects or a package archive for a later link step.
	if ctx.mode == ModeGen {
		return nil
	}

	printCmds := ctx.shouldPrintCommands(verbose)
	cgoLLFiles, cgoLdflags, err := buildCgo(ctx, aPkg, aPkg.Package.Syntax, externs, printCmds)
	if err != nil {
		return fmt.Errorf("build cgo of %v failed: %v", pkgPath, err)
	}
	aPkg.ObjFiles = append(aPkg.ObjFiles, cgoLLFiles...)
	aPkg.ObjFiles = append(aPkg.ObjFiles, concatPkgLinkFiles(ctx, pkg, printCmds)...)
	if aPkg.AltPkg == nil || llruntime.HasAdditiveAltPkg(pkgPath) {
		if asmObjFiles, err := compilePkgSFiles(ctx, aPkg, pkg, printCmds); err != nil {
			return err
		} else {
			aPkg.ObjFiles = append(aPkg.ObjFiles, asmObjFiles...)
		}
	}
	if aliasObjs, err := buildGoCgoAliasObjects(ctx, pkgPath, aPkg.Package.Syntax, printCmds); err != nil {
		return err
	} else {
		aPkg.ObjFiles = append(aPkg.ObjFiles, aliasObjs...)
	}
	aPkg.LinkArgs = append(aPkg.LinkArgs, cgoLdflags...)
	aPkg.LinkArgs = append(aPkg.LinkArgs, goCgoLinkArgs(ctx.buildConf.Goos, aPkg.Package.Syntax)...)
	if aPkg.AltPkg != nil {
		altLLFiles, altLdflags, e := buildCgo(ctx, aPkg, aPkg.AltPkg.Syntax, externs, printCmds)
		if e != nil {
			return fmt.Errorf("build cgo of %v failed: %v", pkgPath, e)
		}
		aPkg.ObjFiles = append(aPkg.ObjFiles, altLLFiles...)
		aPkg.ObjFiles = append(aPkg.ObjFiles, concatPkgLinkFiles(ctx, aPkg.AltPkg.Package, printCmds)...)
		if asmObjFiles, err := compilePkgSFiles(ctx, aPkg, aPkg.AltPkg.Package, printCmds); err != nil {
			return err
		} else {
			aPkg.ObjFiles = append(aPkg.ObjFiles, asmObjFiles...)
		}
		if aliasObjs, err := buildGoCgoAliasObjects(ctx, pkgPath, aPkg.AltPkg.Syntax, printCmds); err != nil {
			return err
		} else {
			aPkg.ObjFiles = append(aPkg.ObjFiles, aliasObjs...)
		}
		aPkg.LinkArgs = append(aPkg.LinkArgs, altLdflags...)
		aPkg.LinkArgs = append(aPkg.LinkArgs, goCgoLinkArgs(ctx.buildConf.Goos, aPkg.AltPkg.Syntax)...)
	}
	if pkg.ExportFile != "" {
		exportFile, exportBuffer, err := exportPackageObject(ctx, pkg.PkgPath, pkg.ExportFile, ret)
		if err != nil {
			return fmt.Errorf("export object of %v failed: %v", pkgPath, err)
		}
		if exportFile != "" {
			aPkg.ObjFiles = append(aPkg.ObjFiles, exportFile)
		} else {
			aPkg.ObjBuffers = append(aPkg.ObjBuffers, exportBuffer)
		}
		if debugBuild || verbose {
			fmt.Fprintf(os.Stderr, "==> Export %s: %s\n", aPkg.PkgPath, pkg.ExportFile)
		}
	}
	return nil
}

func printCompiledPackage(conf *Config, pkg *aPackage) {
	if conf.PrintPackages && !pkg.CacheHit {
		fmt.Fprintln(os.Stderr, pkg.PkgPath)
	}
}

func exportObject(ctx *context, pkgPath string, exportFile string, pkg llssa.Package) (string, error) {
	if useInMemoryNativeCodegen(ctx) {
		return exportObjectInMemory(ctx, pkgPath, exportFile, pkg)
	}
	return exportObjectWithClang(ctx, pkgPath, exportFile, []byte(pkg.String()))
}

func exportPackageObject(ctx *context, pkgPath string, exportFile string, pkg llssa.Package) (string, packageArchiveBuffer, error) {
	if !useInMemoryNativeCodegen(ctx) {
		path, err := exportObjectWithClang(ctx, pkgPath, exportFile, []byte(pkg.String()))
		return path, packageArchiveBuffer{}, err
	}
	if ctx.buildConf.CheckLLFiles || ctx.buildConf.GenLL {
		if err := dumpLLVMIRIfNeeded(ctx, pkgPath, exportFile, pkg.String()); err != nil {
			return "", packageArchiveBuffer{}, err
		}
	}
	buf, kind, err := emitObjectToMemoryBuffer(ctx, pkg)
	if err != nil {
		return "", packageArchiveBuffer{}, err
	}
	name := filepath.Base(exportFile) + ".o"
	if ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "# compiling archive member %s for pkg: %s\n", name, pkgPath)
		fmt.Fprintf(os.Stderr, "# using %s\n", kind)
	}
	return "", packageArchiveBuffer{name: name, buffer: buf}, nil
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
		if msg, err := llcCheck(ctx.commands, f.Name()); err != nil {
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
	buf, kind, err := emitObjectToMemoryBuffer(ctx, pkg)
	if err != nil {
		return "", err
	}
	defer buf.Dispose()
	return writeObjectBufferToFile(ctx, pkgPath, exportFile, buf, kind)
}

func emitObjectToMemoryBuffer(ctx *context, pkg llssa.Package) (gllvm.MemoryBuffer, string, error) {
	ltoMode := ctx.buildConf.ltoMode()
	var (
		buf  gllvm.MemoryBuffer
		err  error
		kind = "in-memory LLVM object emission"
	)
	switch ltoMode {
	case lto.Full:
		// reference to https: //github.com/espressif/llvm-project/blob/04a1a3482ce3ee00b5bbec1ce852e58410e4b6ad/clang/lib/CodeGen/BackendUtil.cpp#L197
		// Clang emit SplitLTOUnit for full lto bitcode except on darwin.
		buf = gllvm.WriteFullLTOBitcodeToMemoryBuffer(pkg.Module(), ctx.buildConf.Goos != "darwin")
		kind = "in-memory LLVM full LTO bitcode emission"
	case lto.Thin:
		buf = gllvm.WriteThinLTOBitcodeToMemoryBuffer(pkg.Module())
		kind = "in-memory LLVM ThinLTO bitcode emission"
	default:
		buf, err = ctx.prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), gllvm.ObjectFile)
		if err != nil {
			return gllvm.MemoryBuffer{}, "", err
		}
	}
	return buf, kind, nil
}

func writeObjectBufferToFile(ctx *context, pkgPath, exportFile string, buf gllvm.MemoryBuffer, kind string) (string, error) {
	base := filepath.Base(exportFile)
	objFile, err := os.CreateTemp("", base+"-*.o")
	if err != nil {
		return "", err
	}
	objFileName := objFile.Name()
	if ctx.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", objFileName, pkgPath)
		fmt.Fprintf(os.Stderr, "# using %s\n", kind)
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
		if msg, err := llcCheck(ctx.commands, f.Name()); err != nil {
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

func llcCheck(commands commandEnv, exportFile string) (msg string, err error) {
	cmd := commands.configure(exec.Command("llc", "-filetype=null", exportFile))
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

func prepareLocalVariables(prog llssa.Program, groups ...[]*packages.Package) error {
	seen := make(map[*types.Package]bool)
	var firstErr error
	for _, roots := range groups {
		packages.Visit(roots, nil, func(p *packages.Package) {
			if firstErr != nil || p.Types == nil || p.IllTyped || seen[p.Types] {
				return
			}
			seen[p.Types] = true
			firstErr = cl.PrepareInactiveLocalVariables(prog, p.Fset, p.Types, p.TypesInfo, p.Syntax)
		})
		if firstErr != nil {
			return firstErr
		}
	}

	if len(groups) == 0 {
		return nil
	}
	active := make(map[string]bool)
	activate := func(p *packages.Package) {
		if p.Types == nil || p.IllTyped {
			return
		}
		active[llssa.PathOf(p.Types)] = true
		prog.ActivateLocalitiesFor(p.Types)
	}
	packages.Visit(groups[0], nil, activate)
	for _, roots := range groups[1:] {
		for _, root := range roots {
			if root == nil || root.Types == nil || !active[llssa.PathOf(root.Types)] {
				continue
			}
			packages.Visit([]*packages.Package{root}, nil, activate)
		}
	}
	return nil
}

type ssaBuildEntry struct {
	pkg      *ssa.Package
	syntax   []*ast.File
	fixOrder bool
}

func registerAltSSAPkgs(prog *ssa.Program, patches cl.Patches, alts []*packages.Package, conf *Config, verbose bool) []ssaBuildEntry {
	var entries []ssaBuildEntry
	packages.Visit(alts, nil, func(p *packages.Package) {
		if typs := p.Types; typs != nil && !p.IllTyped {
			if debugBuild || verbose {
				log.Println("==> BuildSSA", p.ID)
			}
			pkgSSA := prog.CreatePackage(typs, p.Syntax, p.TypesInfo, true)
			entries = append(entries, ssaBuildEntry{pkg: pkgSSA, syntax: p.Syntax})
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
	return entries
}

type aPackage struct {
	*packages.Package
	SSA    *ssa.Package
	AltPkg *packages.Cached
	LPkg   llssa.Package

	NeedRt     bool
	NeedPyInit bool

	LinkArgs    []string
	ObjFiles    []string               // file-backed archive members: .o or .ll
	ObjBuffers  []packageArchiveBuffer // LLVM-produced in-memory archive members
	ArchiveFile string                 // archive file: .a (output of archiver, used for linking)
	Meta        *meta.PackageMeta
	rewriteVars map[string]string

	// Cache related fields
	Fingerprint string // fingerprint digest
	Manifest    string // manifest text content
	CacheHit    bool   // whether cache was hit
}

type Package = *aPackage

func registerSSAPkgs(ctx *context, initial []*packages.Package, verbose bool) ([]*aPackage, []ssaBuildEntry, error) {
	prog := ctx.progSSA
	var all []*aPackage
	var entries []ssaBuildEntry
	var errs []*packages.Package
	packages.Visit(initial, nil, func(p *packages.Package) {
		if p.Types != nil && !p.IllTyped {
			pkgPath := p.PkgPath
			// Use p.ID to check duplicates since same pkgPath may have different IDs
			if _, ok := ctx.pkgByID[p.ID]; ok || strings.HasPrefix(pkgPath, altPkgPathPrefix) {
				return
			}
			var altPkg *packages.Cached
			ssaPkg, created := createSSAPkg(ctx, prog, p, verbose)
			if created {
				entries = append(entries, ssaBuildEntry{pkg: ssaPkg, syntax: p.Syntax, fixOrder: true})
			}
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
				fmt.Fprintln(os.Stderr, formatPackageError(err, ctx.buildConf.NoErrorColumn))
			}
			fmt.Fprintln(os.Stderr, "cannot build SSA for package", errPkg)
		}
		return nil, nil, fmt.Errorf("cannot build SSA for packages")
	}
	return all, entries, nil
}

// buildSSAPkgs builds registered packages with the requested bound, then
// performs ordering repair serially because it mutates instruction slices.
func buildSSAPkgs(ctx *context, entries []ssaBuildEntry) {
	if len(entries) == 0 {
		return
	}
	unique := make([]ssaBuildEntry, 0, len(entries))
	index := make(map[*ssa.Package]int, len(entries))
	for _, entry := range entries {
		if entry.pkg == nil {
			continue
		}
		if i, ok := index[entry.pkg]; ok {
			unique[i].fixOrder = unique[i].fixOrder || entry.fixOrder
			continue
		}
		index[entry.pkg] = len(unique)
		unique = append(unique, entry)
	}
	jobs := make(chan ssaBuildEntry, len(unique))
	var wg sync.WaitGroup
	for range min(ctx.buildConf.parallelism(), len(unique)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entry := range jobs {
				entry.pkg.Build()
			}
		}()
	}
	for _, entry := range unique {
		jobs <- entry
	}
	close(jobs)
	wg.Wait()
	for _, entry := range unique {
		if entry.fixOrder {
			fixSSAOrder(entry.pkg, entry.syntax)
		}
	}
}

func formatPackageError(err packages.Error, noColumn bool) string {
	formatted := err.Error()
	if !noColumn {
		return formatted
	}
	if pos, ok := positionWithoutColumn(err.Pos); ok {
		return pos + ": " + err.Msg
	}
	lines := strings.Split(formatted, "\n")
	for i, line := range lines {
		if line, ok := diagnosticWithoutColumn(line); ok {
			lines[i] = line
		}
	}
	return strings.Join(lines, "\n")
}

func positionWithoutColumn(pos string) (string, bool) {
	lastColon := strings.LastIndexByte(pos, ':')
	if lastColon < 0 {
		return "", false
	}
	if _, parseErr := strconv.Atoi(pos[lastColon+1:]); parseErr != nil {
		return "", false
	}
	linePos := pos[:lastColon]
	lineColon := strings.LastIndexByte(linePos, ':')
	if lineColon < 0 {
		return "", false
	}
	if _, parseErr := strconv.Atoi(linePos[lineColon+1:]); parseErr != nil {
		return "", false
	}
	return linePos, true
}

func diagnosticWithoutColumn(line string) (string, bool) {
	separator := strings.Index(line, ": ")
	if separator < 0 {
		return "", false
	}
	pos, ok := positionWithoutColumn(line[:separator])
	if !ok {
		return "", false
	}
	return pos + line[separator:], true
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
// Parent expressions can inherit that untyped result. This causes go/ssa sanity
// check to fail when a non-constant instruction result remains untyped.
// See: https://github.com/golang/go/issues/77067
func fixUntypedShiftTypes(p *packages.Package) {
	var toFix []ast.Expr
	for expr, tv := range p.TypesInfo.Types {
		if tv.Value != nil {
			continue
		}
		basic, ok := tv.Type.(*types.Basic)
		if !ok || basic.Info()&types.IsUntyped == 0 {
			continue
		}
		toFix = append(toFix, expr)
	}

	for _, expr := range toFix {
		tv := p.TypesInfo.Types[expr]
		p.TypesInfo.Types[expr] = types.TypeAndValue{
			Type:  types.Default(tv.Type),
			Value: tv.Value,
		}
	}
}

func applyPatches(ctx *context, p *packages.Package, verbose bool) {
	// Fix untyped shift types before SSA build
	// See: https://github.com/golang/go/issues/77067
	fixUntypedShiftTypes(p)

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

func createSSAPkg(ctx *context, prog *ssa.Program, p *packages.Package, verbose bool) (*ssa.Package, bool) {
	pkgSSA := prog.ImportedPackage(p.ID)
	if pkgSSA == nil {
		if debugBuild || verbose {
			log.Println("==> BuildSSA", p.ID)
		}
		applyPatches(ctx, p, verbose)
		pkgSSA = prog.CreatePackage(p.Types, p.Syntax, p.TypesInfo, true)
		return pkgSSA, true
	}
	return pkgSSA, false
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

const llgoFuncInfo = "LLGO_FUNCINFO"
const llgoFuncInfoSites = "LLGO_FUNCINFO_SITES"
const llgoTrace = "LLGO_TRACE"
const llgoOptimize = "LLGO_OPTIMIZE"
const llgoWasmRuntime = "LLGO_WASM_RUNTIME"
const llgoWasiThreads = "LLGO_WASI_THREADS"
const llgoStdioNobuf = "LLGO_STDIO_NOBUF"
const llgoFullRpath = "LLGO_FULL_RPATH"
const llgoBuildCache = "LLGO_BUILD_CACHE"
const llgoShadowStack = "LLGO_SHADOW_STACK"

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

func IsFuncInfoEnabled() bool {
	return isEnvOn(llgoFuncInfo, true)
}

// IsFuncInfoSitesEnabled controls the body-embedded site records
// independently of the funcinfo tables (LLGO_FUNCINFO_SITES=0 keeps the
// metadata but drops entry and PC-line inline-asm sites). Useful for
// isolating codegen perturbation caused by the in-body asm anchors.
func IsFuncInfoSitesEnabled() bool {
	return isEnvOn(llgoFuncInfoSites, true)
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

func llvmPassPipeline(level optlevel.Level, ltoMode lto.Mode) string {
	switch ltoMode {
	case lto.Full:
		return "lto-pre-link<" + level.Name() + ">"
	case lto.Thin:
		return "thinlto-pre-link<" + level.Name() + ">"
	default:
		return "default<" + level.Name() + ">"
	}
}

func shouldRunLLVMPasses(mode Mode) bool {
	return mode != ModeGen
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
