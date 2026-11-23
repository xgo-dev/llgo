package build

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	llssa "github.com/xgo-dev/llgo/ssa"

	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

// Track changed globals through passive initializers that still run. An
// effectful initializer must never observe a value the compact link omits.
func checkStartupReads(ctx *context, effect *startupEffects, dropped map[*ssa.Package]bool) bool {
	replaced := make(map[*ssa.Package]*ssa.Package)
	for _, entry := range ctx.pkgByID {
		if entry.AltPkg != nil && cl.PatchSkipsOriginalPackage(entry.AltPkg.Syntax) {
			replaced[ctx.progSSA.Package(entry.Types)] = ctx.progSSA.Package(entry.AltPkg.Types)
		}
	}
	bySymbol := startupImplementations(ctx, replaced)
	roots := make([]*ssa.Function, 0, len(effect.trusted))
	for _, pkg := range ctx.progSSA.AllPackages() {
		if effect.trusted[pkg.Pkg] && replaced[pkg] == nil {
			if init := pkg.Func("init"); init != nil && len(init.Blocks) != 0 {
				roots = append(roots, init)
			}
		}
	}
	calls := make(map[ssa.CallInstruction][]*ssa.Function)
	for _, node := range rta.Analyze(roots, true).CallGraph.Nodes {
		for _, edge := range node.Out {
			if edge.Site != nil && edge.Site.Common().StaticCallee() == nil {
				calls[edge.Site] = append(calls[edge.Site], edge.Callee.Func)
			}
		}
	}
	type effects struct {
		passive bool
		reads   map[*ssa.Global]bool
		writes  map[*ssa.Global]bool
	}
	retained := make(map[*ssa.Package]effects)
	changedGlobals := make(map[*ssa.Global]bool)
	for _, pkg := range ctx.progSSA.AllPackages() {
		if !effect.trusted[pkg.Pkg] || replaced[pkg] != nil {
			continue
		}
		fn := pkg.Func("init")
		if fn == nil {
			continue
		}
		reads, writes := scanStartupGlobals(fn, ctx, bySymbol, calls, replaced)
		if dropped[pkg] {
			maps.Copy(changedGlobals, writes)
			continue
		}
		retained[pkg] = effects{effect.check(fn) == nil, reads, writes}
	}
	for {
		before := len(changedGlobals)
		for _, row := range retained {
			if row.passive && startupObserves(row.reads, changedGlobals) {
				maps.Copy(changedGlobals, row.writes)
			}
		}
		if len(changedGlobals) == before {
			break
		}
	}
	for global := range changedGlobals {
		name := llssa.FullName(global.Pkg.Pkg, global.Name())
		if _, ok := ctx.prog.Linkname(name); ok || ctx.prog.HasLinknameTarget(name) {
			return false
		}
	}
	for pkg, row := range retained {
		if row.passive {
			continue
		}

		var observations []string
		for global := range row.reads {
			if changedGlobals[global] {
				observations = append(observations, global.String())
			}
		}
		if len(observations) != 0 {
			slices.Sort(observations)
			if ctx.shouldPrintCommands(ctx.buildConf.Verbose) {
				fmt.Fprintf(os.Stderr, "llgo: retain firmware initialization (%s reads %v)\n", pkg.Pkg.Path(), observations)
			}
			return false
		}
	}
	return true
}

func startupObserves(reads, changed map[*ssa.Global]bool) bool {
	for g := range reads {
		if changed[g] {
			return true
		}
	}
	return false
}

func scanStartupGlobals(fn *ssa.Function, ctx *context, bySymbol map[string][]*ssa.Function, dynamic map[ssa.CallInstruction][]*ssa.Function, replaced map[*ssa.Package]*ssa.Package) (reads, writes map[*ssa.Global]bool) {
	reads, writes = make(map[*ssa.Global]bool), make(map[*ssa.Global]bool)
	seen := make(map[*ssa.Function]bool)
	var visit func(*ssa.Function)
	visit = func(f *ssa.Function) {
		if f == nil || seen[f] {
			return
		}
		seen[f] = true
		if f.Pkg != nil && replaced[f.Pkg] != nil {
			if alt := cl.ResolveInlineTarget(f, ctx.patches); alt != f {
				visit(alt)
			}
			return
		}
		if f.Pkg != nil && f.Signature.Recv() == nil {
			symbol := startupSymbol(ctx, f)
			for _, impl := range bySymbol[symbol] {
				visit(impl)
			}
		}
		for _, block := range f.Blocks {
			for _, inst := range block.Instrs {
				if call, ok := inst.(ssa.CallInstruction); ok {
					callee := call.Common().StaticCallee()
					if callee != nil && callee.Synthetic == "package initializer" && f.Synthetic == "package initializer" {
						continue
					}
					visit(callee)
					for _, target := range dynamic[call] {
						visit(target)
					}
				}
				var write *ssa.Global
				if store, ok := inst.(*ssa.Store); ok {
					write, _ = startupAddress(store.Addr).(*ssa.Global)
					if write != nil {
						writes[write] = true
					}
				}
				for _, operand := range inst.Operands(nil) {
					if operand != nil {
						if g, ok := (*operand).(*ssa.Global); ok && g != write {
							reads[g] = true
						}
					}
				}
			}
		}
	}
	visit(fn)
	return
}

func startupImplementations(ctx *context, replaced map[*ssa.Package]*ssa.Package) map[string][]*ssa.Function {
	bySymbol := make(map[string][]*ssa.Function)
	for _, pkg := range ctx.progSSA.AllPackages() {
		if replaced[pkg] != nil {
			continue
		}
		for _, member := range pkg.Members {
			if fn, ok := member.(*ssa.Function); ok && len(fn.Blocks) != 0 {
				symbol := startupSymbol(ctx, fn)
				bySymbol[symbol] = append(bySymbol[symbol], fn)
			}
		}
	}
	return bySymbol
}

func startupSymbol(ctx *context, fn *ssa.Function) string {
	path := fn.Pkg.Pkg.Path()
	name := llssa.FullName(fn.Pkg.Pkg, fn.Name())
	if path == wasiRuntimePath {
		name = "runtime." + fn.Name()
	}
	if alias, ok := ctx.prog.Linkname(name); ok {
		return strings.TrimPrefix(alias, "go:")
	}
	return name
}
