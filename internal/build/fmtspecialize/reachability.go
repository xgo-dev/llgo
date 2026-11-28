package fmtspecialize

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

// ApplyStandalone commits the rewrite only when it can drop the generic formatter.
func (plan *ValuePlan) ApplyStandalone(roots []*ssa.Function) map[*types.Package]bool {
	if len(roots) == 0 {
		return nil
	}
	plan.resolveBuiltinCalls(roots)
	if len(plan.calls) == 0 {
		return nil
	}
	return plan.apply(func() bool {
		// A direct call path is sufficient to reject the rewrite. Mixed
		// formatters commonly fail here without requiring whole-program RTA.
		if plan.hasDirectGenericPath(roots) {
			return false
		}
		for fn := range rta.Analyze(roots, false).Reachable {
			if plan.isGenericFormatter(fn) {
				return false
			}
		}
		return true
	})
}

func (plan *ValuePlan) hasDirectGenericPath(roots []*ssa.Function) bool {
	work := append([]*ssa.Function(nil), roots...)
	seen := make(map[*ssa.Function]bool)
	for len(work) != 0 {
		fn := work[len(work)-1]
		work = work[:len(work)-1]
		if seen[fn] {
			continue
		}
		seen[fn] = true
		if plan.isGenericFormatter(fn) {
			return true
		}
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				if call, ok := inst.(ssa.CallInstruction); ok {
					if callee := call.Common().StaticCallee(); callee != nil {
						work = append(work, callee)
					}
				}
			}
		}
	}
	return false
}

// AppendSymbolRoots includes Go functions referenced by translated assembly.
func AppendSymbolRoots(prog *ssa.Program, roots []*ssa.Function, symbols []string) []*ssa.Function {
	if len(symbols) == 0 {
		return roots
	}
	packagesBySymbolPath := make(map[string][]*ssa.Package)
	for _, pkg := range prog.AllPackages() {
		path := llssa.PathOf(pkg.Pkg)
		packagesBySymbolPath[path] = append(packagesBySymbolPath[path], pkg)
	}
	for _, symbol := range symbols {
		i := strings.LastIndexByte(symbol, '.')
		if i < 0 {
			continue
		}
		for _, pkg := range packagesBySymbolPath[symbol[:i]] {
			if fn := pkg.Func(symbol[i+1:]); fn != nil {
				roots = append(roots, fn)
			}
		}
	}
	return roots
}

// AppendExportRoots collects modeled C entry points. The second result
// reports an entry that can enter Go outside the modeled call graph.
func AppendExportRoots(prog *ssa.Program, roots []*ssa.Function, pkg *packages.Package, trusted bool) ([]*ssa.Function, bool) {
	if pkg == nil {
		return roots, false
	}
	ssaPkg := prog.Package(pkg.Types)
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				switch item.Name {
				case "go:wasmexport", "llgo:export":
					return roots, true
				case "go:linkname", "llgo:link":
					if !trusted {
						return roots, true
					}
				}
			}
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			for _, item := range directive.ParseGroup(fn.Doc) {
				if item.Name != "export" {
					continue
				}
				if ssaPkg == nil || fn.Recv != nil || ssaPkg.Func(fn.Name.Name) == nil {
					return roots, true
				}
				roots = append(roots, ssaPkg.Func(fn.Name.Name))
			}
		}
	}
	return roots, false
}
