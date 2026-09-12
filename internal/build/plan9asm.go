package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/packages"
	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
	llruntime "github.com/xgo-dev/llgo/runtime"
	gllvm "github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

func plan9asmTranslateOptions(conf *Config) llplan9asm.TranslateOptions {
	opt := llplan9asm.TranslateOptions{GOARM: conf.GOARM}
	if conf.Goarch == "386" && conf.GO386 == "softfloat" {
		opt.X87Mode = extplan9asm.X87Software
	}
	return opt
}

// compilePkgSFiles translates Go/Plan9 assembly files selected by the package
// loader for this package/target into LLVM IR, compiles them to .o, and returns
// the object files for linking.
func compilePkgSFiles(ctx *context, aPkg *aPackage, pkg *packages.Package, verbose bool) ([]string, error) {
	if len(ctx.patchFiles[pkg.PkgPath]) != 0 && llruntime.SourcePatchReplacesAsmForGOARCH(pkg.PkgPath, ctx.buildConf.Goarch) {
		return nil, nil
	}
	sfiles, err := pkgSFiles(ctx, pkg)
	if err != nil {
		return nil, err
	}
	if len(sfiles) == 0 {
		return nil, nil
	}
	if !ctx.plan9asmEnabled(pkg.PkgPath) {
		// Strong policy: selected Plan9 asm must be handled either by
		// translation or by an explicit runtime alt patch.
		if !llruntime.HasAltPkg(pkg.PkgPath) {
			// Some stdlib .s files are placeholders without any TEXT bodies
			// (e.g. runtime/debug/debug.s). They carry no executable asm and
			// are safe to ignore.
			hasText, err := llplan9asm.HasAnyTextAsm(ctx.buildConf.Overlay, sfiles)
			if err != nil {
				return nil, fmt.Errorf("%s: inspect asm files: %w", pkg.PkgPath, err)
			}
			if !hasText {
				return nil, nil
			}
			return nil, fmt.Errorf("%s: selected .s files require plan9asm translation; add support or whitelist via runtime/build.go hasAltPkg", pkg.PkgPath)
		}
		return nil, nil
	}
	if pkg.Types == nil || pkg.Types.Scope() == nil {
		return nil, fmt.Errorf("%s: missing types (needed for asm signatures)", pkg.PkgPath)
	}

	skipDarwinDynimportTrampolines := shouldCheckDarwinDynimportTrampolineAsm(ctx, pkg)
	objFiles := make([]string, 0, len(sfiles))
	for _, sfile := range sfiles {
		src, err := llplan9asm.ReadFileWithOverlay(ctx.buildConf.Overlay, sfile)
		if err != nil {
			return nil, fmt.Errorf("%s: read %s: %w", pkg.PkgPath, sfile, err)
		}
		if shouldSkipDarwinDynimportTrampolineAsm(skipDarwinDynimportTrampolines, sfile, src) {
			continue
		}
		tr, err := llplan9asm.TranslateSourceModuleForPkgWithOptions(pkg, sfile, src, ctx.buildConf.Goos, ctx.buildConf.Goarch, plan9asmTranslateOptions(ctx.buildConf))
		if err != nil {
			// Some stdlib .s files are comment-only placeholders (e.g. internal/cpu/cpu.s).
			// Skip those silently.
			if strings.Contains(err.Error(), "no TEXT directive found") {
				continue
			}
			return nil, fmt.Errorf("%s: translate %s: %w", pkg.PkgPath, sfile, err)
		}
		mod := tr.Module

		// Apply cabi rewrites to translated asm modules for declaration-driven
		// aggregates (slice/string/interface headers).
		// runtime asm uses hand-written calling conventions and must stay on
		// original Go ABI semantics.
		if pkg.PkgPath != "runtime" {
			lowerLargeAggregates(ctx.prog, mod)
			ctx.cTransformer.TransformModule(pkg.PkgPath, mod)
		}
		applySizeOptimizationAttributes(mod, ctx.buildConf.OptLevel)
		if err := externalizePlan9DataGlobals(aPkg.LPkg.Module(), mod, ctx.prog.TargetData()); err != nil {
			mod.Dispose()
			return nil, fmt.Errorf("%s: bind DATA globals from %s: %w", pkg.PkgPath, sfile, err)
		}
		ll := mod.String()
		mod.Dispose()

		baseName := aPkg.ExportFile + filepath.Base(sfile) // used for stable debug output paths
		tmpPrefix := "plan9asm-" + filepath.Base(sfile) + "-"

		llFile, err := os.CreateTemp("", tmpPrefix+"*.ll")
		if err != nil {
			return nil, fmt.Errorf("%s: create temp .ll for %s: %w", pkg.PkgPath, sfile, err)
		}
		llPath := llFile.Name()
		if _, err := llFile.WriteString(ll); err != nil {
			llFile.Close()
			os.Remove(llPath)
			return nil, fmt.Errorf("%s: write temp .ll for %s: %w", pkg.PkgPath, sfile, err)
		}
		if err := llFile.Close(); err != nil {
			os.Remove(llPath)
			return nil, fmt.Errorf("%s: close temp .ll for %s: %w", pkg.PkgPath, sfile, err)
		}
		defer os.Remove(llPath)

		// Keep a copy of translated .ll when GenLL is enabled (mirrors clFile/exportObject).
		if ctx.buildConf.GenLL {
			dst := baseName + ".ll"
			if err := os.Chmod(llPath, 0644); err != nil {
				return nil, fmt.Errorf("%s: chmod temp .ll for %s: %w", pkg.PkgPath, sfile, err)
			}
			if err := copyFileAtomic(llPath, dst); err != nil {
				return nil, fmt.Errorf("%s: keep .ll for %s: %w", pkg.PkgPath, sfile, err)
			}
		}

		objFile, err := os.CreateTemp("", tmpPrefix+"*.o")
		if err != nil {
			return nil, fmt.Errorf("%s: create temp .o for %s: %w", pkg.PkgPath, sfile, err)
		}
		objPath := objFile.Name()
		objFile.Close()

		args := []string{"-o", objPath, "-c", llPath, "-Wno-override-module"}
		if ctx.shouldPrintCommands(verbose) {
			fmt.Fprintf(os.Stderr, "# compiling %s for pkg: %s\n", objPath, pkg.PkgPath)
			fmt.Fprintln(os.Stderr, "clang", args)
		}
		if err := ctx.irCompiler().Compile(args...); err != nil {
			os.Remove(objPath)
			return nil, fmt.Errorf("%s: clang compile asm ll for %s: %w", pkg.PkgPath, sfile, err)
		}
		objFiles = append(objFiles, objPath)
	}

	return objFiles, nil
}

// externalizePlan9DataGlobals turns zero-initialized Go globals that are
// defined by Plan 9 DATA/GLOBL into declarations. Otherwise the Go object
// satisfies its own references and the linker has no reason to extract a
// data-only assembly member from the package archive.
func externalizePlan9DataGlobals(goMod, asmMod gllvm.Module, td gllvm.TargetData) error {
	for asmGlobal := asmMod.FirstGlobal(); !asmGlobal.IsNil(); asmGlobal = gllvm.NextGlobal(asmGlobal) {
		if asmGlobal.Initializer().IsNil() {
			continue
		}
		goGlobal := goMod.NamedGlobal(asmGlobal.Name())
		if goGlobal.IsNil() || goGlobal.IsDeclaration() {
			continue
		}
		if goGlobal.IsThreadLocal() != asmGlobal.IsThreadLocal() {
			return fmt.Errorf("global %s has incompatible thread-local storage", asmGlobal.Name())
		}
		goSize := td.TypeAllocSize(goGlobal.GlobalValueType())
		asmSize := td.TypeAllocSize(asmGlobal.GlobalValueType())
		if goSize != asmSize {
			return fmt.Errorf("global %s has Go size %d but DATA size %d", asmGlobal.Name(), goSize, asmSize)
		}
		initializer := goGlobal.Initializer()
		if initializer.IsNil() {
			continue
		}
		if !initializer.IsNull() {
			return fmt.Errorf("global %s has both a Go initializer and DATA", asmGlobal.Name())
		}
		goGlobal.SetInitializer(gllvm.Value{})
		goGlobal.SetLinkage(gllvm.ExternalLinkage)
		goGlobal.SetGlobalConstant(false)
	}
	return nil
}

func shouldCheckDarwinDynimportTrampolineAsm(ctx *context, pkg *packages.Package) bool {
	if ctx == nil || ctx.buildConf == nil || ctx.buildConf.Goos != "darwin" {
		return false
	}
	// golang.org/x/sys/unix generates zsyscall_darwin_*.s trampolines for
	// go:cgo_import_dynamic symbols. llgo emits equivalent trampolines from
	// the Go pragmas, so the generated asm trampolines must be skipped there.
	if pkg == nil || pkg.PkgPath != "golang.org/x/sys/unix" {
		return false
	}
	_, dynimports := collectGoCgoPragmas(pkg.Syntax)
	return len(dynimports) != 0
}

func shouldSkipDarwinDynimportTrampolineAsm(enabled bool, sfile string, src []byte) bool {
	if !enabled {
		return false
	}
	if !strings.HasPrefix(filepath.Base(sfile), "zsyscall_darwin_") {
		return false
	}
	return bytes.Contains(src, []byte("_trampoline<>(SB)")) &&
		bytes.Contains(src, []byte("_trampoline_addr(SB)"))
}

type plan9AsmSigCacheKey struct {
	ctx     *context
	pkgPath string
}

var plan9AsmSigCache sync.Map // key: plan9AsmSigCacheKey, value: map[string]struct{}

func archSupportsPlan9AsmDefaults(goarch string) bool {
	return goarch == "386" || goarch == "arm64" || goarch == "amd64" || goarch == "wasm"
}

type plan9asmPkgsEnvMode int

const (
	plan9asmEnvDefaults plan9asmPkgsEnvMode = iota
	plan9asmEnvAll
	plan9asmEnvNone
	plan9asmEnvSelected
)

type plan9asmPkgsEnv struct {
	mode plan9asmPkgsEnvMode
	pkgs map[string]bool
}

func parsePlan9AsmPkgsEnv(raw string) plan9asmPkgsEnv {
	v := strings.TrimSpace(raw)
	switch {
	case v == "":
		return plan9asmPkgsEnv{mode: plan9asmEnvDefaults}
	case v == "0" || strings.EqualFold(v, "off") || strings.EqualFold(v, "false"):
		return plan9asmPkgsEnv{mode: plan9asmEnvNone}
	case v == "*" || strings.EqualFold(v, "all") || strings.EqualFold(v, "on") || strings.EqualFold(v, "true"):
		return plan9asmPkgsEnv{mode: plan9asmEnvAll}
	default:
		pkgs := make(map[string]bool)
		split := func(r rune) bool {
			switch r {
			case ',', ';', ' ', '\t', '\n', '\r':
				return true
			default:
				return false
			}
		}
		for _, p := range strings.FieldsFunc(v, split) {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			pkgs[p] = true
		}
		return plan9asmPkgsEnv{mode: plan9asmEnvSelected, pkgs: pkgs}
	}
}

func plan9asmSigsForPkg(ctx *context, pkgPath string) (map[string]struct{}, error) {
	if ctx == nil || pkgPath == "" {
		return nil, nil
	}
	key := plan9AsmSigCacheKey{ctx: ctx, pkgPath: pkgPath}
	if v, ok := plan9AsmSigCache.Load(key); ok {
		return v.(map[string]struct{}), nil
	}

	sigs := make(map[string]struct{})
	if !ctx.plan9asmEnabled(pkgPath) {
		plan9AsmSigCache.Store(key, sigs)
		return sigs, nil
	}
	if hasAltPkgForTarget(ctx.buildConf, pkgPath) && !llruntime.HasAdditiveAltPkgForGOARCH(pkgPath, ctx.buildConf.Goarch) {
		plan9AsmSigCache.Store(key, sigs)
		return sigs, nil
	}

	var pkg *packages.Package
	for p := range ctx.pkgs {
		if p != nil && p.PkgPath == pkgPath {
			pkg = p
			break
		}
	}
	if pkg == nil {
		plan9AsmSigCache.Store(key, sigs)
		return sigs, nil
	}

	sfiles, err := pkgSFiles(ctx, pkg)
	if err != nil {
		return nil, err
	}
	for _, sfile := range sfiles {
		src, err := llplan9asm.ReadFileWithOverlay(ctx.buildConf.Overlay, sfile)
		if err != nil {
			return nil, fmt.Errorf("%s: read %s: %w", pkg.PkgPath, sfile, err)
		}
		tr, err := llplan9asm.TranslateSourceForPkgWithOptions(pkg, sfile, src, ctx.buildConf.Goos, ctx.buildConf.Goarch, plan9asmTranslateOptions(ctx.buildConf))
		if err != nil {
			if strings.Contains(err.Error(), "no TEXT directive found") {
				continue
			}
			return nil, fmt.Errorf("%s: translate %s: %w", pkg.PkgPath, sfile, err)
		}
		for name := range tr.Signatures {
			sigs[name] = struct{}{}
		}
	}
	plan9AsmSigCache.Store(key, sigs)
	return sigs, nil
}

func cabiSkipFuncsForPlan9Asm(ctx *context, pkgPath string, mod gllvm.Module) []string {
	if ctx == nil || mod.IsNil() || ctx.buildConf == nil {
		return nil
	}
	// Plan9 asm modules are translated to LLVM and transformed by cabi in
	// compilePkgSFiles. Most packages should not skip any rewrite.
	//
	// runtime is special: many runtime asm entry points use hand-crafted
	// conventions that are not declaration-driven. Keep Go-side declarations
	// untouched for those symbols.
	if pkgPath == "runtime" || pkgPath == "reflect" {
		ownSigs, err := plan9asmSigsForPkg(ctx, pkgPath)
		check(err)
		if len(ownSigs) == 0 {
			return nil
		}
		names := make([]string, 0, len(ownSigs))
		for name := range ownSigs {
			names = append(names, name)
		}
		return names
	}
	return nil
}

func (ctx *context) plan9asmEnabled(pkgPath string) bool {
	if !ctx.plan9asmReady {
		ctx.plan9asmOnce.Do(func() {
			cfg := parsePlan9AsmPkgsEnv(Plan9ASMPkgs())
			ctx.plan9asmMode = cfg.mode
			switch cfg.mode {
			case plan9asmEnvSelected:
				ctx.plan9asmPkgs = make(map[string]bool, len(cfg.pkgs))
				for p := range cfg.pkgs {
					ctx.plan9asmPkgs[p] = true
				}
			default:
				ctx.plan9asmPkgs = make(map[string]bool)
			}
			ctx.plan9asmReady = true
		})
	}

	switch ctx.plan9asmMode {
	case plan9asmEnvAll:
		return true
	case plan9asmEnvNone:
		return false
	case plan9asmEnvSelected:
		return ctx.plan9asmPkgs[pkgPath]
	case plan9asmEnvDefaults:
		return plan9asmEnabledByDefault(ctx.buildConf, pkgPath)
	default:
		return false
	}
}

func hasAltPkgForTarget(conf *Config, pkgPath string) bool {
	if conf == nil || !llruntime.HasAltPkgForGOARCH(pkgPath, conf.Goarch) {
		return false
	}
	if pkgPath == "syscall/js" && conf.Target == "" && conf.Goos == "js" && conf.Goarch == "wasm" {
		// J32/GoJS retains the selected GOROOT's Go implementation. Its
		// source patch adapts only host imports and the runtime event entry.
		return false
	}
	if llruntime.HasAdditiveAltPkgForGOARCH(pkgPath, conf.Goarch) {
		return true
	}
	// When Plan9 asm translation is enabled, avoid also pulling in alt packages
	// that provide the same symbols as pure-Go fallbacks.
	if plan9asmEnabledByDefault(conf, pkgPath) && !plan9asmDisabledByEnv() {
		return false
	}
	return true
}

func plan9asmDisabledByEnv() bool {
	return parsePlan9AsmPkgsEnv(Plan9ASMPkgs()).mode == plan9asmEnvNone
}

func plan9asmEnabledByDefault(conf *Config, pkgPath string) bool {
	if conf == nil {
		return false
	}
	if !archSupportsPlan9AsmDefaults(conf.Goarch) {
		return false
	}
	return !llruntime.HasAltPkgForGOARCH(pkgPath, conf.Goarch) || llruntime.HasAdditiveAltPkgForGOARCH(pkgPath, conf.Goarch)
}

func pkgSFiles(ctx *context, pkg *packages.Package) ([]string, error) {
	if pkg == nil || pkg.PkgPath == "" {
		return nil, nil
	}
	if ctx.sfilesCache == nil {
		if ctx.sfilesFrozen {
			return nil, fmt.Errorf("package %s assembly files were not prepared before backend execution", pkg.PkgPath)
		}
		ctx.sfilesCache = make(map[string][]string)
	}
	if v, ok := ctx.sfilesCache[pkg.ID]; ok {
		return v, nil
	}
	// The synthetic test main shares the tested package's directory, but it does
	// not own that package's assembly files.
	if ctx.mode == ModeTest && pkg.Name == "main" && strings.HasSuffix(pkg.ID, ".test") {
		ctx.sfilesCache[pkg.ID] = nil
		return nil, nil
	}
	// Some unit tests construct synthetic packages without an on-disk package
	// directory. Treat those packages as having no selected assembly files.
	if pkg.Dir == "" {
		ctx.sfilesCache[pkg.ID] = nil
		return nil, nil
	}
	if ctx.sfilesFrozen {
		return nil, fmt.Errorf("package %s assembly files were not prepared before backend execution", pkg.PkgPath)
	}
	paths := selectedSFiles(pkg.OtherFiles)

	// internal/chacha8rand has highly optimized arch asm on amd64/arm64.
	// Until full vector lowering lands, force the generic stub entry, which
	// tail-jumps to block_generic and preserves package behavior.
	// Scope the stub override to targets that actually select chacha8 assembly.
	if len(paths) != 0 && pkg.PkgPath == "internal/chacha8rand" {
		stub := filepath.Join(pkg.Dir, "chacha8_stub.s")
		if _, err := os.Stat(stub); err == nil {
			paths = []string{stub}
			ctx.sfilesCache[pkg.ID] = paths
			return paths, nil
		}
	}
	// Embedded ARM targets currently reuse GOOS=linux metadata, but they do not
	// have a Linux syscall surface. Skip syscall asm in that mode so embedded
	// builds do not inherit Linux/ARM-specific frame layouts.
	if shouldSkipPlan9AsmSFilesForTarget(ctx.buildConf, pkg.PkgPath) {
		ctx.sfilesCache[pkg.ID] = nil
		return nil, nil
	}

	ctx.sfilesCache[pkg.ID] = paths
	return paths, nil
}

func selectedSFiles(files []string) []string {
	if len(files) == 0 {
		return nil
	}
	var paths []string
	for _, f := range files {
		if !strings.HasSuffix(f, ".s") && !strings.HasSuffix(f, ".S") {
			continue
		}
		if strings.HasSuffix(f, "_test.s") || strings.HasSuffix(f, "_test.S") {
			continue
		}
		paths = append(paths, f)
	}
	return paths
}

func shouldSkipPlan9AsmSFilesForTarget(conf *Config, pkgPath string) bool {
	return conf != nil && conf.Target != "" && conf.Goarch == "arm" && pkgPath == "syscall"
}
