package build

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// The compact entry shares all archives with the ordinary entry. The final
// WASI link validates any remaining dependency on managed startup.
func buildCompactWASIEntry(ctx *context, pkg *packages.Package, prep *mainLinkPreparation, verbose bool) (string, error) {
	gen := prep.gen
	drops := compactWASIPassiveInits(ctx, prep.gen.packageInits)
	if !checkWASIStartupReads(ctx, ctx.wasiStartupGOROOT, drops) {
		return "", nil
	}
	gen.compactWASIStartup = wasiProcessDiagnostics(ctx)
	if gen.compactWASIStartup == nil {
		return "", nil
	}
	gen.rtInit = false
	gen.abiInit = 0
	gen.defaultGODEBUG = ""
	gen.funcInfo = nil
	gen.pcLineInfo = nil
	gen.packageInits = slices.DeleteFunc(slices.Clone(gen.packageInits), func(name string) bool {
		path, ok := strings.CutSuffix(name, ".init")
		return ok && slices.Contains(drops, path)
	})
	entry := genMainModule(ctx, llssa.PkgRuntime, pkg, &gen)
	defer entry.LPkg.Module().Dispose()
	for _, name := range ctx.initializerGuardsOff {
		cl.DefineInitializerGuard(entry.LPkg, name, false)
	}
	for _, path := range drops {
		cl.DefineInitializerGuard(entry.LPkg, cl.PackageInitGuardName(path), false)
	}
	roots := appendModuleDeclarationRoots(slices.Clone(ctx.sizePruningRoots), entry.LPkg.Module())
	if err := applyDeadcodeDropOverrides(prep.linkedOrder, entry, false, true, verbose, roots...); err != nil {
		return "", err
	}
	return exportObject(ctx, "entry_main_wasi_compact", entry.ExportFile+"-compact", entry.LPkg)
}

type wasiStartupDiagnostics struct {
	prestat     string
	prestatName string
	errnoTable  *types.Array
}

// Read source positions and the array layout from the standard library
// being linked. An unrecognized Go version keeps ordinary startup.
func wasiProcessDiagnostics(ctx *context) *wasiStartupDiagnostics {
	if ctx.progSSA == nil {
		return nil
	}
	pkg := ctx.progSSA.ImportedPackage("syscall")
	if pkg == nil {
		return nil
	}
	guard, _ := pkg.Members[wasiPathStateMember].(*ssa.Global)
	table, _ := pkg.Members["errorstr"].(*ssa.Global)
	if guard == nil || table == nil {
		return nil
	}
	tableType, _ := table.Type().Underlying().(*types.Pointer).Elem().Underlying().(*types.Array)
	if tableType == nil || tableType.Len() <= 0 || !types.Identical(tableType.Elem(), types.Typ[types.String]) {
		return nil
	}
	scan := wasiPathStateInitializer(pkg, guard)
	init := pkg.Func("init")
	if scan == nil || init == nil || !matchesWASIProcessContract(pkg, scan) {
		return nil
	}
	frame := func(fn *ssa.Function, pos token.Pos) string {
		at := ctx.progSSA.Fset.Position(pos)
		if !at.IsValid() || at.Filename == "" {
			return ""
		}
		return fmt.Sprintf("%s.%s(...)\n\t%s:%d\n", fn.Pkg.Pkg.Path(), fn.Name(), at.Filename, at.Line)
	}
	var caller string
	for _, block := range init.Blocks {
		var previous token.Pos
		for _, inst := range block.Instrs {
			if call, ok := inst.(*ssa.Call); ok {
				if call.Common().StaticCallee() == scan {
					if caller != "" {
						return nil
					}
					caller = frame(init, previous)
				}
				if call.Pos().IsValid() {
					previous = call.Pos()
				}
			}
		}
	}
	if caller == "" {
		return nil
	}
	sites := &wasiStartupDiagnostics{errnoTable: tableType}
	for _, block := range scan.Blocks {
		for _, inst := range block.Instrs {
			ins, ok := inst.(*ssa.Panic)
			if !ok {
				continue
			}
			box, _ := ins.X.(*ssa.MakeInterface)
			if box == nil {
				return nil
			}
			concat, _ := box.X.(*ssa.BinOp)
			if concat == nil || concat.Op != token.ADD {
				return nil
			}
			prefix, _ := concat.X.(*ssa.Const)
			call, _ := concat.Y.(*ssa.Call)
			if prefix == nil || prefix.Value == nil || prefix.Value.Kind() != constant.String || call == nil {
				return nil
			}
			format := call.Common().StaticCallee()
			if format == nil || format.Pkg != pkg || format.Name() != "Error" {
				return nil
			}
			where := frame(scan, ins.Pos())
			if where == "" {
				return nil
			}
			switch constant.StringVal(prefix.Value) {
			case "fd_prestat: ":
				if sites.prestat != "" {
					return nil
				}
				sites.prestat = where + caller
			case "fd_prestat_dir_name: ":
				if sites.prestatName != "" {
					return nil
				}
				sites.prestatName = where + caller
			default:
				return nil
			}
		}
	}
	if sites.prestat == "" || sites.prestatName == "" {
		return nil
	}
	return sites
}

func checkWASIStartupReads(ctx *context, goroot string, drops []string) bool {
	rootFuncs := wasmReflectRoots(ctx)
	effect := wasiStartupEffects(ctx, goroot)
	replaced := replacedStartupPackages(ctx)
	bySymbol := startupImplementations(ctx, replaced)
	roots := slices.Clone(rootFuncs)
	var omitted []*ssa.Function
	for _, p := range ctx.progSSA.AllPackages() {
		if !effect.trusted[p.Pkg] || replaced[p] != nil {
			continue
		}
		fn := p.Func("init")
		if fn == nil || len(fn.Blocks) == 0 {
			continue
		}
		if slices.ContainsFunc(drops, func(path string) bool { return startupSymbol(ctx, fn) == path+".init" }) {
			omitted = append(omitted, fn)
		} else {
			roots = append(roots, fn)
		}
	}
	if slices.Contains(drops, "internal/godebug") {
		if dbg := ctx.progSSA.ImportedPackage("internal/godebug"); dbg != nil && dbg.Func("update") != nil {
			// The ordinary runtime invokes this registered callback during
			// startup, including through a symbol alias invisible to RTA.
			omitted = append(omitted, dbg.Func("update"))
		}
	}
	calls := startupIndirectCalls(append(slices.Clone(roots), omitted...))
	changed := make(map[*ssa.Global]bool)
	for _, fn := range omitted {
		_, writes := scanStartupGlobals(fn, ctx, bySymbol, calls, replaced, nil)
		maps.Copy(changed, writes)
	}
	leaves := make(map[*ssa.Function]bool)
	if p := ctx.progSSA.ImportedPackage("syscall"); p != nil {
		leaves[p.Func("Getenv")] = true
	}
	if p := ctx.progSSA.ImportedPackage("internal/testlog"); p != nil {
		leaves[p.Func("Logger")] = true
	}
	if p := ctx.progSSA.ImportedPackage("os"); p != nil {
		leaves[p.Func("runtime_args")] = true
	}
	off := make(map[string]bool)
	for _, guard := range ctx.initializerGuardsOff {
		off[guard] = true
	}
	var optional []cl.OptionalInitializer
	for _, p := range ctx.pkgByID {
		for _, groups := range p.guardedInitializers {
			for _, group := range groups {
				if off[cl.InitializerGuardName(group.Global)] {
					optional = append(optional, group)
				}
			}
		}
	}
	restore := omitInitializerEdges(optional)
	defer restore()
	copies := compactStartupCopies(roots, changed)
	restoreCopies := omitInitializerEdges(copies)
	defer restoreCopies()
	safe := true
	for _, fn := range roots {
		reads, _ := scanStartupGlobals(fn, ctx, bySymbol, calls, replaced, leaves)
		var hazards []string
		for glob := range reads {
			if changed[glob] {
				hazards = append(hazards, glob.String())
			}
		}
		if len(hazards) != 0 {
			safe = false
			if ctx.shouldPrintCommands(ctx.buildConf.Verbose) {
				slices.Sort(hazards)
				fmt.Fprintf(os.Stderr, "llgo: retain WASI startup (%s observes %v)\n", fn, hazards)
			}
		}
	}
	return safe
}

// Require the existing whole-program proofs before changing only the final
// entry. The verifier at link time covers runtime dependencies, but a read of
// cwd or an environment observation may allocate nothing.
func canProbeWASIProcessStartup(ctx *context) bool {
	conf := ctx.buildConf
	if !ctx.probeWASIMutator || ctx.progSSA == nil || conf.PCLNLoad.startsAtEntry() || conf.Coverage != nil ||
		ctx.mode == ModeTest || ctx.frontendOptions.DebuggerFrames || ctx.prog.Target().WasmReflectBridges {
		return false
	}
	if logger := ctx.progSSA.ImportedPackage("internal/testlog"); logger != nil && logger.Members[wasiStartupTestlogGuard] == nil {
		return false
	}
	off := ctx.initializerGuardsOff
	return slices.Contains(off, wasiPathStateGuard) &&
		(slices.Contains(off, mutableEnvironmentGuard) || slices.Contains(off, wasiEnvironmentNeededGuard))
}

// A package can copy another package's sentinel without observing its value
// in control flow. Track the destination too, so an application cannot read
// the copied zero value when the producer is omitted.
func compactStartupCopies(roots []*ssa.Function, changed map[*ssa.Global]bool) []cl.OptionalInitializer {
	var candidates []struct {
		group cl.OptionalInitializer
		from  *ssa.Global
	}
	for _, init := range roots {
		if init.Synthetic != "package initializer" {
			continue
		}
		for _, block := range init.Blocks {
			var previous ssa.Instruction
			for _, ins := range block.Instrs {
				if _, ok := ins.(*ssa.DebugRef); ok {
					continue
				}
				if store, ok := ins.(*ssa.Store); ok {
					value, _ := previous.(ssa.Value)
					dest, _ := store.Addr.(*ssa.Global)
					if value != nil && store.Val == value && dest != nil &&
						dest.Pkg == init.Pkg && len(*value.Referrers()) == 1 {
						if source := compactCopiedGlobal(value); source != nil && source.Pkg != dest.Pkg {
							candidates = append(candidates, struct {
								group cl.OptionalInitializer
								from  *ssa.Global
							}{cl.OptionalInitializer{First: previous, Store: store, Global: dest}, source})
						}
					}
				}
				previous = ins
			}
		}
	}
	var copies []cl.OptionalInitializer
	for {
		remaining := candidates[:0]
		for _, candidate := range candidates {
			if !changed[candidate.from] {
				remaining = append(remaining, candidate)
			} else {
				changed[candidate.group.Global] = true
				copies = append(copies, candidate.group)
			}
		}
		if len(remaining) == len(candidates) {
			return copies
		}
		candidates = remaining
	}
}

// Direct reads and a getter whose complete body is a read and a return
// preserve the same value. Reject any computation or second observation.
func compactCopiedGlobal(value ssa.Value) *ssa.Global {
	if call, ok := value.(*ssa.Call); ok {
		fn := call.Common().StaticCallee()
		if fn == nil || len(fn.Params) != 0 || len(fn.FreeVars) != 0 || len(fn.Blocks) != 1 {
			return nil
		}
		var body []ssa.Instruction
		for _, ins := range fn.Blocks[0].Instrs {
			if _, debug := ins.(*ssa.DebugRef); !debug {
				body = append(body, ins)
			}
		}
		if len(body) != 2 {
			return nil
		}
		load, _ := body[0].(*ssa.UnOp)
		ret, _ := body[1].(*ssa.Return)
		if load == nil || ret == nil || len(ret.Results) != 1 || ret.Results[0] != load {
			return nil
		}
		value = load
	}
	if load, ok := value.(*ssa.UnOp); ok && load.Op == token.MUL {
		g, _ := load.X.(*ssa.Global)
		return g
	}
	return nil
}

// Discover passive initializers in the actual import closure. The set of
// guarded calls in the archives is independent of this executable's choices.
func compactWASIPassiveInits(ctx *context, names []string) []string {
	p := wasiStartupEffects(ctx, ctx.wasiStartupGOROOT)
	allowWASIGodebugSetup(ctx, &p)
	drops := []string{"syscall", "runtime", llssa.PkgRuntime}
	if timePkg := ctx.progSSA.ImportedPackage("time"); timePkg != nil {
		// This call reads the monotonic clock for startNano. Its value is only
		// used by time functions; check all surviving reads before omitting it.
		clock := timePkg.Func("runtimeNano")
		if resolved := p.providers[clock]; resolved != nil {
			clock = resolved
		}
		if clock != nil {
			p.passed[clock] = true
		}
	}
	implementations := startupImplementations(ctx, nil)
	for _, name := range names {
		path, ok := strings.CutSuffix(name, ".init")
		if !ok || path == "os" || slices.Contains(drops, path) {
			continue
		}
		candidates := implementations[name]
		if len(candidates) == 0 {
			continue
		}
		var proofError error
		for _, fn := range candidates {
			if proofError = p.check(fn); proofError != nil {
				break
			}
		}
		if proofError == nil {
			drops = append(drops, path)
		} else if ctx.shouldPrintCommands(ctx.buildConf.Verbose) {
			fmt.Fprintf(os.Stderr, "llgo: keep WASI initializer %s (%v)\n", name, proofError)
		}
	}
	return drops
}

func wasiStartupEffects(ctx *context, goroot string) startupEffects {
	p := newStartupEffects(ctx, goroot)
	p.separateInitOrder = true
	for path, patch := range ctx.patches {
		if pkg := ctx.pkgByID[path]; patch.Alt != nil && pkg != nil && isStandardLibraryPackage(pkg.Package, goroot) {
			p.trusted[patch.Alt.Pkg] = true
		}
	}
	return p
}

// The settings registration can be skipped if no code observes the runtime
// fields or the callback's writes. Start with the actual standard-library
// initializer rather than assuming that a symbol name proves its behavior.
func allowWASIGodebugSetup(ctx *context, p *startupEffects) {
	pkg := ctx.progSSA.ImportedPackage("internal/godebug")
	if pkg == nil || !p.trusted[pkg.Pkg] {
		return
	}
	update, factory := pkg.Func("update"), pkg.Func("newIncNonDefault")
	setUpdate, setFactory := pkg.Func("setUpdate"), pkg.Func("setNewIncNonDefault")
	if update == nil || factory == nil || setUpdate == nil || setFactory == nil {
		return
	}
	impl := startupImplementations(ctx, nil)
	updates := impl["internal/godebug.setUpdate"]
	if len(updates) == 0 || slices.ContainsFunc(updates, func(fn *ssa.Function) bool {
		return fn.Pkg.Pkg.Path() != wasiRuntimePath || !p.trusted[fn.Pkg.Pkg]
	}) {
		return
	}
	for _, fn := range pkg.Members {
		init, ok := fn.(*ssa.Function)
		if !ok || !strings.HasPrefix(init.Name(), "init#") || len(init.Blocks) != 1 {
			continue
		}
		var body []ssa.Instruction
		for _, ins := range init.Blocks[0].Instrs {
			if _, debug := ins.(*ssa.DebugRef); !debug {
				body = append(body, ins)
			}
		}
		if len(body) != 3 {
			continue
		}
		first, _ := body[0].(*ssa.Call)
		second, _ := body[1].(*ssa.Call)
		ret, _ := body[2].(*ssa.Return)
		if ret == nil || len(ret.Results) != 0 || !wasiRegistrationCall(first, setUpdate, update) ||
			!wasiRegistrationCall(second, setFactory, factory) {
			continue
		}
		p.passed[init] = true
		return
	}
}

func wasiRegistrationCall(call *ssa.Call, target, callback *ssa.Function) bool {
	if call == nil {
		return false
	}
	c := call.Common()
	return c.StaticCallee() == target && len(c.Args) == 1 && c.Args[0] == callback
}
