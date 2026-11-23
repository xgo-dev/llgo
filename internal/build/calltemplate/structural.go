package calltemplate

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"slices"
	"strings"
	"text/template"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
)

type structuralBindings struct {
	library *packages.Package
	entries map[string]structuralRule
}

type structuralRule struct {
	helper *types.Func
	body   *template.Template
	// A second implementation of the same API can keep the general
	// implementation live, even if all calls to these entries match.
	excludeImport string
}

func (b *structuralBindings) bind(fn *types.Func, args string) error {
	if _, exists := b.entries[fn.Name()]; exists {
		return fmt.Errorf("duplicate any template for %s", fn.Name())
	}
	if args == "fallback" {
		b.entries[fn.Name()] = structuralRule{}
		return nil
	}
	opts, err := bindingOptions(args)
	if err != nil {
		return err
	}
	for key := range opts {
		switch key {
		case "helper", "body", "exclude-import":
		default:
			return fmt.Errorf("unknown any template option %q", key)
		}
	}
	scope := fn.Pkg().Scope()
	helper, _ := scope.Lookup(opts["helper"]).(*types.Func)
	if err := validateCandidate(fn, helper, opts["helper"]); err != nil {
		return err
	}
	text, _ := scope.Lookup(opts["body"]).(*types.Const)
	if text == nil || text.Val().Kind() != constant.String {
		return fmt.Errorf("any template body %q must be a string constant", opts["body"])
	}
	rule, err := template.New(text.Name()).Funcs(expansionFunctions).Option("missingkey=error").Parse(constant.StringVal(text.Val()))
	if err != nil {
		return err
	}
	b.entries[fn.Name()] = structuralRule{helper, rule, opts["exclude-import"]}
	return nil
}

// StructuralUse records selections in the executable and the providers whose
// generic entry points the closed-world check eliminated.
type StructuralUse struct {
	Entry     *types.Package
	Providers map[string]bool
	Changes   []string
}

type structuralCall struct {
	call *ast.CallExpr
	name string
}

// RewriteStructural specializes a group only when the executable package can
// replace each use of the provider's declared entries.
func (b *Bindings) RewriteStructural(roots, runtimePkgs []*packages.Package) (StructuralUse, error) {
	var result StructuralUse
	if len(roots) != 1 || roots[0].Name != "main" || len(b.structural) == 0 {
		return result, nil
	}
	var closure []*packages.Package
	deps := make(map[string]*packages.Package)
	aliases := make(map[string]bool)
	packages.Visit(slices.Concat(roots, runtimePkgs), func(pkg *packages.Package) bool {
		closure = append(closure, pkg)
		if _, exists := deps[pkg.PkgPath]; !exists {
			deps[pkg.PkgPath] = pkg
		}
		for _, file := range pkg.Syntax {
			for _, comments := range file.Comments {
				for _, d := range directive.ParseGroup(comments) {
					if d.Name != "go:linkname" && d.Name != "llgo:link" {
						continue
					}
					for _, arg := range strings.Fields(d.Args) {
						aliases[arg] = true
					}
				}
			}
		}
		return true
	}, nil)
	result.Providers = make(map[string]bool)
	entry := roots[0]
	result.Entry = entry.Types
	for _, group := range b.structural {
		if !group.closed(entry, closure, aliases) {
			continue
		}
		calls, err := group.rewrite(entry, deps)
		if err != nil {
			return result, err
		}
		for _, call := range calls {
			result.Changes = append(result.Changes, changeKey(entry, call.call, "any", group.library.PkgPath+":"+call.name))
		}
		if len(calls) != 0 {
			result.Providers[group.library.PkgPath] = true
		}
	}
	return result, nil
}

func (b *structuralBindings) closed(entry *packages.Package, closure []*packages.Package, aliases map[string]bool) bool {
	path := b.library.PkgPath
	for target := range aliases {
		if strings.HasPrefix(target, path+".") {
			return false
		}
	}
	for _, pkg := range closure {
		if pkg == b.library {
			continue
		}
		for _, rule := range b.entries {
			if rule.excludeImport != "" && pkg.Imports[rule.excludeImport] != nil {
				return false
			}
		}
		if dep := pkg.Imports[path]; dep != nil {
			if dep.Types != b.library.Types || pkg.TypesInfo == nil {
				return false
			}
			if pkg != entry {
				for _, obj := range pkg.TypesInfo.Uses {
					if fn, ok := obj.(*types.Func); ok && fn.Pkg() == b.library.Types && b.hasEntry(fn.Name()) {
						return false
					}
				}
			}
		}
	}
	return true
}

func (b *structuralBindings) hasEntry(name string) bool {
	_, ok := b.entries[name]
	return ok
}

func (b *structuralBindings) rewrite(pkg *packages.Package, deps map[string]*packages.Package) ([]structuralCall, error) {
	if pkg.Types == nil || pkg.TypesInfo == nil || pkg.TypesSizes == nil || pkg.Imports[b.library.PkgPath] == nil {
		return nil, nil
	}
	gen := newExpansion(pkg, deps, b.library.Name)
	var err error
	calls, closed := typesynth.PlanClosedCalls(pkg, b.library.Types, b.hasEntry, func(call *ast.CallExpr, name string) (structuralCall, bool) {
		if err != nil {
			return structuralCall{}, false
		}
		rule := b.entries[name]
		if rule.helper == nil {
			return structuralCall{}, false
		}
		original := b.library.Types.Scope().Lookup(name).(*types.Func)
		candidate, instance := selectCandidate(pkg.TypesInfo, call, original.Signature(), []candidate{{fn: rule.helper}})
		if candidate == nil {
			return structuralCall{}, false
		}
		generated, expandErr := gen.expand(rule, instance.Type.(*types.Signature), call)
		if errors.Is(expandErr, errExpansionRejected) {
			return structuralCall{}, false
		}
		if expandErr != nil {
			err = fmt.Errorf("expand %s.%s: %w", b.library.PkgPath, name, expandErr)
			return structuralCall{}, false
		}
		return structuralCall{call, generated}, true
	})
	if err != nil || !closed || len(calls) == 0 {
		return nil, err
	}
	if err := gen.apply(b.library.Name, calls); err != nil {
		return nil, err
	}
	return calls, nil
}
