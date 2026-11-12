package build

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/internal/reflectnames"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

const reflectSiteAttribute = "llgo.deadcode.reflectsite"

// Both Drop backends use the same optimized snapshots and name analysis. Keep
// the experiment on native macOS executables; Thin's temporary link archives
// cannot currently supply the persistent paths required by a DWARF debug map.
func useDeadcodeReflectNames(ctx *context) bool {
	if ctx == nil || ctx.buildConf == nil || runtime.GOOS != "darwin" {
		return false
	}
	c := ctx.buildConf
	backendSupported := c.ltoMode() == lto.Off ||
		c.ltoMode() == lto.Thin && c.DebugArtifactMode == DebugArtifactNone
	return c.deadcodeDropEnabled() && backendSupported &&
		c.BuildMode == BuildModeExe && c.Target == "" && !c.GenLL &&
		c.Goos == runtime.GOOS && c.Goarch == runtime.GOARCH && len(ctx.initial) == 1 &&
		(ctx.mode == ModeBuild || ctx.mode == ModeTest)
}

func needsDeadcodeReflectIR(ctx *context, pm *meta.PackageMeta) bool {
	return useDeadcodeReflectNames(ctx) && pm != nil && len(pm.ReflectMethodSites()) != 0
}

// Snapshot after optimization but before native codegen, which may transform
// the module further. Fresh and cached builds analyze these same bytes. The
// snapshot is never passed to the linker and contains no program-specific
// pruning; global visibility is checked anew for every final program.
func captureDeadcodeReflectIR(ctx *context, pkg Package) error {
	if !needsDeadcodeReflectIR(ctx, pkg.Meta) {
		return nil
	}
	buffer := llvm.WriteBitcodeToMemoryBuffer(pkg.LPkg.Module())
	defer buffer.Dispose()
	f, err := os.CreateTemp("", "llgo-reflect-*.bc")
	if err != nil {
		return err
	}
	if _, err := f.Write(buffer.Bytes()); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return err
	}
	pkg.reflectIRFile, pkg.reflectIRTemp = f.Name(), f.Name()
	return nil
}

// Read optimized analysis snapshots, never the unoptimized type-registration
// modules created on cache hits. Missing calls or unresolved clones retain the
// original demand. Only Drop's demand view is refined; package IR is unchanged.
func refineDeadcodeReflectNames(ctx *context, pkgs []Package, verbose bool) (map[string][]string, error) {
	if !useDeadcodeReflectNames(ctx) {
		return nil, nil
	}
	var candidates []Package
	for _, pkg := range pkgs {
		if pkg.Meta != nil && len(pkg.Meta.ReflectMethodSites()) != 0 {
			candidates = append(candidates, pkg)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	exported := make(map[string]bool)
	if reason := opaqueGlobalInitPackageWithPolicy(ctx, exported, reflectGlobalVisibilityPolicy(ctx)); reason != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "[dce] retain dynamic reflection: opaque inputs in %s\n", reason)
		}
		return nil, nil
	}
	// Include one-name export directives as well as two-name aliases. Program's
	// linkname map alone does not contain the former.
	inspect := func(pkg *types.Package, files []*ast.File) {
		for _, file := range files {
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(comment.Text)
					if len(fields) >= 2 && fields[0] == "//go:linkname" {
						exported[llssa.FullName(pkg, fields[1])] = true
						if len(fields) >= 3 {
							exported[fields[2]] = true
						}
					}
				}
			}
		}
	}
	for _, pkg := range pkgs {
		inspect(pkg.Types, pkg.Syntax)
		if pkg.AltPkg != nil {
			inspect(pkg.AltPkg.Types, pkg.AltPkg.Syntax)
		}
		for name := range pkg.rewriteVars {
			exported[llssa.FullName(pkg.Types, name)] = true
		}
	}
	var accumulator reflectNameAccumulator
	for _, pkg := range candidates {
		analyze := func(mod llvm.Module) error {
			canReadExternal := func(global llvm.Value) bool {
				return privateReflectGlobal(ctx, pkgs, pkg, global.Name(), exported)
			}
			for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
				for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
					for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
						if instr.IsACallInst().IsNil() && instr.IsAInvokeInst().IsNil() {
							continue
						}
						attr := instr.GetCallSiteStringAttribute(-1, reflectSiteAttribute)
						if attr.IsNil() {
							continue
						}
						names, ok := reflectnames.ResolveCall(mod, instr, canReadExternal)
						accumulator.add(attr.GetStringValue(), names, ok)
					}
				}
			}
			return nil
		}
		if err := visitDeadcodeReflectIR(pkg.reflectIRFile, analyze); err != nil {
			return nil, fmt.Errorf("refine reflection in %s: %w", pkg.PkgPath, err)
		}
	}
	names := accumulator.result()
	if verbose {
		fmt.Fprintf(os.Stderr, "[dce] refined %d dynamic reflection sites from optimized LLVM IR\n", len(names))
		for _, site := range sortedReflectSites(names) {
			fmt.Fprintf(os.Stderr, "[dce] %s: %q\n", site, names[site])
		}
	}
	return names, nil
}

// These flags select the backend or hide exports; they cannot introduce extra
// references to a package-private Go variable. Other linker overrides retain
// the existing opaque-native-input fallback.
func reflectGlobalVisibilityPolicy(ctx *context) globalVisibilityPolicy {
	return globalVisibilityPolicy{
		safeLinkFlag: safeReflectAnalysisLinkFlag,
		safeCgoLinkFlag: func(pkg, flag string) bool {
			return ctx.buildConf.Goos == "darwin" && pkg == "internal/syscall/unix" && flag == "-lresolv"
		},
		// Darwin's syscall packages import libSystem and libresolv. These
		// system libraries do not reference private Go globals by name. Pointer
		// arguments are still subject to the resolver's escape checks.
		safeDynamicImport: func(imp cgoImportDynamicDecl) bool {
			return ctx.buildConf.Goos == "darwin" && (imp.library == "/usr/lib/libSystem.B.dylib" || imp.library == "/usr/lib/libresolv.9.dylib")
		},
	}
}

func safeReflectAnalysisLinkFlag(flag string) bool {
	return flag == "-Wl,-no_exported_symbols" || flag == "-fuse-ld=lld" ||
		strings.HasPrefix(flag, "--ld-path=") && len(flag) > len("--ld-path=")
}

func privateReflectGlobal(ctx *context, pkgs []Package, owner Package, name string, exported map[string]bool) bool {
	if owner.Types == nil || owner.LPkg == nil {
		return false
	}
	// $data is the compiler-generated backing storage for a named global.
	base := strings.TrimSuffix(name, "$data")
	local, ok := strings.CutPrefix(base, llssa.FullName(owner.Types, ""))
	if !ok || local == "" || exported[name] || exported[base] {
		return false
	}
	variable, ok := owner.Types.Scope().Lookup(local).(*types.Var)
	if !ok || variable.Exported() {
		return false
	}
	if _, alias := ctx.prog.Linkname(base); alias || ctx.prog.HasLinknameTarget(base) || ctx.prog.HasLinknameTarget(name) {
		return false
	}
	if _, locality, ok, err := ctx.prog.ResolveLocalityFor(owner.Types, base); err != nil || ok && locality.Locality != llssa.LocalityNone {
		return false
	}
	for _, other := range pkgs {
		if other == owner {
			continue
		}
		if other.LPkg == nil {
			return false
		}
		for _, symbol := range []string{name, base} {
			g := other.LPkg.Module().NamedGlobal(symbol)
			if !g.IsNil() && (!g.IsDeclaration() || !g.FirstUse().IsNil()) {
				return false
			}
		}
	}
	return true
}

type reflectNameAccumulator struct {
	names   map[string][]string
	unknown map[string]bool
}

func (a *reflectNameAccumulator) add(site string, names []string, proven bool) {
	if a.names == nil {
		a.names = make(map[string][]string)
		a.unknown = make(map[string]bool)
	}
	if !proven || len(names) == 0 {
		a.unknown[site] = true
		return
	}
	a.names[site] = append(a.names[site], names...)
}

func (a *reflectNameAccumulator) result() map[string][]string {
	for site, names := range a.names {
		if a.unknown[site] {
			delete(a.names, site)
			continue
		}
		slices.Sort(names)
		a.names[site] = slices.Compact(names)
	}
	return a.names
}

func sortedReflectSites(names map[string][]string) []string {
	keys := make([]string, 0, len(names))
	for key := range names {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func visitDeadcodeReflectIR(path string, visit func(llvm.Module) error) error {
	// Packages without an optimized snapshot retain their original demand.
	if path == "" {
		return nil
	}
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod, err := ctx.ParseBitcodeFile(path)
	if err != nil {
		return err
	}
	defer mod.Dispose()
	return visit(mod)
}
