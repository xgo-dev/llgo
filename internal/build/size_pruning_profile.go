package build

import (
	"go/ast"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func appendSymbolRoots(prog *ssa.Program, roots []*ssa.Function, symbols []string) []*ssa.Function {
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

// appendExportRoots collects modeled C entry points. The second result
// reports an entry that can enter Go outside the modeled call graph.
func appendExportRoots(prog *ssa.Program, roots []*ssa.Function, pkg *packages.Package, trusted bool) ([]*ssa.Function, bool) {
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
