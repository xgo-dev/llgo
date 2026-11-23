package calltemplate

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
)

// Generator supplies the compile-time implementation of typed Go candidates
// declared by a source patch. The normal template matcher selects candidates.
type Generator struct {
	Package           string
	FallbackEntries   []string
	ConflictingImport string
	Apply             func(Generation) (bool, error)
}

type Generation struct {
	Entry, Library *packages.Package
	Packages       []*packages.Package
	Calls          []GeneratedCall
}

type GeneratedCall struct {
	Call      *ast.CallExpr
	Candidate *types.Func
}

type GeneratedUse struct {
	Entry     *types.Package
	Providers map[string]bool
	Changes   []string
}

type generatorBinding struct {
	library    *packages.Package
	generator  Generator
	candidates map[*types.Func][]candidate
	enabled    bool
}

// RewriteGenerated selects a provider only when each declared entry can
// specialize in the executable package. Dependency archives are reusable.
func (b *Bindings) RewriteGenerated(roots, runtimePkgs []*packages.Package) (GeneratedUse, error) {
	var result GeneratedUse
	if len(roots) != 1 || roots[0].Name != "main" || len(b.generated) == 0 {
		return result, nil
	}
	var closure []*packages.Package
	var aliases []string
	packages.Visit(slices.Concat(roots, runtimePkgs), func(pkg *packages.Package) bool {
		closure = append(closure, pkg)
		for _, file := range pkg.Syntax {
			for _, group := range file.Comments {
				for _, d := range directive.ParseGroup(group) {
					if d.Name == "go:linkname" || d.Name == "llgo:link" {
						aliases = append(aliases, strings.Fields(d.Args)...)
					}
				}
			}
		}
		return true
	}, nil)
	entry := roots[0]
	result.Entry = entry.Types
	result.Providers = make(map[string]bool)
	for _, binding := range b.generated {
		if !binding.enabled || !binding.closed(entry, closure, aliases) {
			continue
		}
		calls, ok := binding.match(entry)
		if !ok || len(calls) == 0 {
			continue
		}
		applied, err := binding.generator.Apply(Generation{entry, binding.library, closure, calls})
		if err != nil {
			return result, err
		}
		if applied {
			result.Providers[binding.library.PkgPath] = true
			for _, c := range calls {
				result.Changes = append(result.Changes, changeKey(entry, c.Call, "any", binding.library.PkgPath+":"+c.Candidate.Name()))
			}
		}
	}
	return result, nil
}

func (b generatorBinding) isEntry(fn *types.Func) bool {
	return len(b.candidates[fn]) != 0 || slices.Contains(b.generator.FallbackEntries, fn.Name())
}

func (b generatorBinding) closed(entry *packages.Package, closure []*packages.Package, aliases []string) bool {
	path := b.library.PkgPath
	for _, alias := range aliases {
		if strings.HasPrefix(alias, path+".") {
			return false
		}
	}
	for _, pkg := range closure {
		if pkg == b.library {
			continue
		}
		if conflict := b.generator.ConflictingImport; conflict != "" && pkg.Imports[conflict] != nil {
			return false
		}
		if dep := pkg.Imports[path]; dep != nil {
			if dep.Types != b.library.Types || pkg.TypesInfo == nil {
				return false
			}
			if pkg != entry {
				for _, obj := range pkg.TypesInfo.Uses {
					if fn, ok := obj.(*types.Func); ok && fn.Pkg() == b.library.Types && b.isEntry(fn) {
						return false
					}
				}
			}
		}
	}
	return true
}

func (b generatorBinding) match(pkg *packages.Package) ([]GeneratedCall, bool) {
	if pkg.TypesInfo == nil || pkg.TypesSizes == nil || pkg.Imports[b.library.PkgPath] == nil {
		return nil, false
	}
	return typesynth.PlanClosedCalls(pkg, b.library.Types, func(name string) bool {
		fn, ok := b.library.Types.Scope().Lookup(name).(*types.Func)
		return ok && b.isEntry(fn)
	}, func(call *ast.CallExpr, name string) (GeneratedCall, bool) {
		fn := b.library.Types.Scope().Lookup(name).(*types.Func)
		c, _ := selectCandidate(pkg.TypesInfo, call, fn.Signature(), b.candidates[fn])
		if c == nil {
			return GeneratedCall{}, false
		}
		return GeneratedCall{call, c.fn}, true
	})
}
