// Package calltemplate specializes calls using declarations and call-site facts.
package calltemplate

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

type candidate struct {
	fn    *types.Func
	guard *guard
}

type guard struct {
	param int
	value constant.Value
}

type rules struct {
	candidates map[*types.Func][]candidate
	formats    map[*types.Func]formatRule
	cstrings   map[*types.Func]bool
}

// Rewrite analyzes calls in each package from its syntax and the declarations
// it imports. Each returned entry describes only rewrites in that package.
func (b *Bindings) Rewrite(roots []*packages.Package, formats bool) (map[*types.Package][]string, error) {
	r := rules{
		candidates: make(map[*types.Func][]candidate),
		formats:    make(map[*types.Func]formatRule),
		cstrings:   make(map[*types.Func]bool),
	}

	if formats {
		for fn, rule := range b.formats {
			r.formats[fn] = rule
		}
	}
	var pkgs []*packages.Package
	var err error
	packages.Visit(roots, func(p *packages.Package) bool {
		if err != nil {
			return false
		}
		if p.TypesInfo != nil && !p.IllTyped {
			err = r.collect(p, formats, p == b.fmt)
			pkgs = append(pkgs, p)
		}
		return err == nil
	}, nil)
	if err != nil {
		return nil, err
	}
	// Keep the generic fallback inside each candidate callable without
	// selecting that candidate recursively.
	protected := make(map[*types.Func]bool)
	providers := make(map[*types.Package]bool)
	for _, group := range b.generated {
		for fn := range group.candidates {
			delete(r.candidates, fn)
		}
	}
	for fn, candidates := range r.candidates {
		providers[fn.Pkg()] = true
		protected[fn] = true
		for _, c := range candidates {
			protected[c.fn] = true
		}
	}
	if len(r.candidates) == 0 && len(r.formats) == 0 {
		return nil, nil
	}
	selected := make(map[*types.Package][]string)
	for _, pkg := range pkgs {
		if pkg.Types == nil || (!importsProvider(pkg, providers) && !(pkg.Name == "main" && len(r.formats) != 0)) {
			continue
		}
		changes, err := r.rewrite(pkg, b.fmt, protected, !otherFmtUsers(pkgs, pkg, b.fmt))
		if err != nil {
			return nil, err
		}
		if len(changes) != 0 {
			slices.Sort(changes)
			selected[pkg.Types] = changes
		}
	}
	return selected, nil
}

func importsProvider(pkg *packages.Package, providers map[*types.Package]bool) bool {
	if providers[pkg.Types] {
		return true
	}
	for _, imported := range pkg.Types.Imports() {
		if providers[imported] {
			return true
		}
	}
	return false
}

func (r rules) collect(pkg *packages.Package, formats, trustedFmt bool) error {
	for _, file := range pkg.Syntax {
		for _, node := range file.Decls {
			decl, ok := node.(*ast.FuncDecl)
			if !ok {
				continue
			}
			original, ok := pkg.TypesInfo.Defs[decl.Name].(*types.Func)
			if !ok {
				continue
			}
			for _, item := range directive.ParseGroup(decl.Doc) {
				switch item.Name {
				case "llgo:template":
					if err := r.addCandidates(original, item.Args); err != nil {
						return fmt.Errorf("%s: %w", pkg.Fset.Position(item.Pos), err)
					}
				case "llgointernal:template:gofmt", "llgointernal:template:cfmt":
					if formats {
						if item.Name == "llgointernal:template:cfmt" && decl.Body != nil {
							return fmt.Errorf("%s: cfmt requires an external C declaration", pkg.Fset.Position(item.Pos))
						}
						rule, err := bindFormat(original, item, decl.Doc, trustedFmt)
						if err != nil {
							return fmt.Errorf("%s: %w", pkg.Fset.Position(item.Pos), err)
						}
						r.formats[original] = rule
					}
				}
			}
			if formats && decl.Body == nil {
				link := externalName(decl.Doc)
				if link == "llgo.cstr" {
					r.cstrings[original] = true
				}
				if _, explicit := r.formats[original]; !explicit && pkg.PkgPath == "github.com/goplus/lib/c" && cFormatIndex(link) >= 0 {
					d := directive.Directive{Name: "llgointernal:template:cfmt", Args: "format=format args=" + llssa.NameValist}
					if rule, err := bindFormat(original, d, decl.Doc, false); err == nil {
						r.formats[original] = rule
					}
				}
			}
		}
	}
	return nil
}

func (r rules) addCandidates(original *types.Func, text string) error {
	sig := original.Signature()
	if sig.Recv() != nil || sig.TypeParams().Len() != 0 || sig.Variadic() {
		return fmt.Errorf("llgo:template requires a non-variadic, non-generic function")
	}
	names, condition, hasGuard := strings.Cut(" "+text+" ", " if ")
	ids := strings.Fields(names)
	if len(ids) == 0 {
		return fmt.Errorf("llgo:template requires a candidate function")
	}
	var when *guard
	if hasGuard {
		if len(ids) != 1 {
			return fmt.Errorf("a conditional llgo:template requires one candidate")
		}
		var err error
		when, err = parseGuard(sig, condition)
		if err != nil {
			return err
		}
	}
	for _, name := range ids {
		fn, _ := original.Pkg().Scope().Lookup(name).(*types.Func)
		if err := validateCandidate(original, fn, name); err != nil {
			return err
		}
		r.candidates[original] = append(r.candidates[original], candidate{fn, when})
	}
	return nil
}

func validateCandidate(original, fn *types.Func, name string) error {
	sig := original.Signature()
	if sig.Recv() != nil || sig.TypeParams().Len() != 0 || sig.Variadic() {
		return fmt.Errorf("llgo:template requires a non-variadic, non-generic function")
	}
	if fn == nil || fn == original {
		return fmt.Errorf("invalid llgo:template candidate %q", name)
	}
	s := fn.Signature()
	if s.Recv() != nil || s.Variadic() || s.Params().Len() != sig.Params().Len() ||
		!types.Identical(s.Results(), sig.Results()) {
		return fmt.Errorf("candidate %s has an incompatible signature for %s", name, original.Name())
	}
	for i := 0; i < s.Params().Len(); i++ {
		old, next := sig.Params().At(i).Type(), s.Params().At(i).Type()
		if !types.Identical(old, next) && !types.IsInterface(old) {
			return fmt.Errorf("candidate %s changes non-interface parameter %d", name, i+1)
		}
	}
	return nil
}

func parseGuard(sig *types.Signature, text string) (*guard, error) {
	expr, err := parser.ParseExpr(text)
	if err == nil {
		if test, ok := expr.(*ast.BinaryExpr); ok && test.Op == token.EQL {
			if id, ok := test.X.(*ast.Ident); ok {
				i := paramIndex(sig, id.Name)
				if value := literal(test.Y); i >= 0 && value != nil {
					return &guard{i, value}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("llgo:template condition must compare a parameter with a literal")
}

func literal(expr ast.Expr) constant.Value {
	switch x := ast.Unparen(expr).(type) {
	case *ast.BasicLit:
		if value := constant.MakeFromLiteral(x.Value, x.Kind, 0); value.Kind() != constant.Unknown {
			return value
		}
	case *ast.Ident:
		switch x.Name {
		case "true", "false":
			return constant.MakeBool(x.Name == "true")
		}
	case *ast.UnaryExpr:
		if x.Op == token.SUB || x.Op == token.ADD {
			if v := literal(x.X); v != nil && v.Kind() >= constant.Int && v.Kind() <= constant.Complex {
				return constant.UnaryOp(x.Op, v, 0)
			}
		}
	}
	return nil
}

func otherFmtUsers(pkgs []*packages.Package, entry, fmtPkg *packages.Package) bool {
	if fmtPkg == nil || entry.Name != "main" {
		return false
	}
	for _, pkg := range pkgs {
		if pkg != entry && pkg != fmtPkg && pkg.Imports["fmt"] != nil {
			return true
		}
	}
	return false
}

func (r rules) rewrite(pkg, fmtPkg *packages.Package, protected map[*types.Func]bool, gofmt bool) ([]string, error) {
	var changes []string
	fmts := newFormatSource(pkg, fmtPkg)
	cfmts := newCFormatSource(pkg, r.cstrings)
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if decl, ok := node.(*ast.FuncDecl); ok {
				fn, _ := pkg.TypesInfo.Defs[decl.Name].(*types.Func)
				return !protected[fn]
			}
			call, ok := node.(*ast.CallExpr)
			if !ok || call.Ellipsis.IsValid() {
				return true
			}
			base := calledFunc(pkg.TypesInfo, call.Fun)
			if base == nil {
				return true
			}
			if list := r.candidates[base]; len(list) != 0 {
				c, inst := selectCandidate(pkg.TypesInfo, call, base.Signature(), list)
				if c != nil {
					typesynth.BindCall(pkg.TypesInfo, call, c.fn, inst)
					changes = append(changes, changeKey(pkg, call, "direct", c.fn.FullName()))
				}
				return true
			}
			if rule, ok := r.formats[base]; ok && pkg.Name == "main" {
				if rule.c {
					if cfmts.fold(call, rule) {
						changes = append(changes, changeKey(pkg, call, "cfmt", ""))
					}
				} else if gofmt && rule.format >= 0 {
					fmts.plan(call, rule)
				}
			}
			return true
		})
	}
	goChanges, err := fmts.apply()
	return append(changes, goChanges...), err
}

func changeKey(pkg *packages.Package, call *ast.CallExpr, kind, target string) string {
	pos := pkg.Fset.PositionFor(call.Pos(), false)
	return fmt.Sprintf("%s:%d:%s:%s", filepath.Base(pos.Filename), pos.Offset, kind, target)
}

func calledFunc(info *types.Info, expr ast.Expr) *types.Func {
	fn, _ := typesynth.Callee(info, expr)
	return fn
}

func selectCandidate(info *types.Info, call *ast.CallExpr, base *types.Signature, list []candidate) (*candidate, types.Instance) {
	if len(call.Args) != base.Params().Len() {
		return nil, types.Instance{}
	}
	actual := actualTypes(info, call)
	var matches []candidate
	var instances []types.Instance
	for _, c := range list {
		if c.guard != nil && !matchesGuard(info.Types[call.Args[c.guard.param]].Value, c.guard.value) {
			continue
		}
		inst, ok := matchSignature(base, c.fn.Signature(), actual)
		if ok {
			matches = append(matches, c)
			instances = append(instances, inst)
		}
	}
	for i := range matches {
		unique := true
		for j := range matches {
			if i != j && !moreSpecific(base, matches[i], matches[j]) {
				unique = false
				break
			}
		}
		if unique {
			return &matches[i], instances[i]
		}
	}
	return nil, types.Instance{}
}

func actualTypes(info *types.Info, call *ast.CallExpr) []types.Type {
	actual := make([]types.Type, len(call.Args))
	for i, e := range call.Args {
		if t := info.TypeOf(e); t != nil && !llssa.HasTypeParam(t) {
			actual[i] = types.Default(t)
		}
	}
	return actual
}

func matchesGuard(actual, want constant.Value) bool {
	if actual == nil {
		return false
	}
	if actual.Kind() != want.Kind() && !(actual.Kind() >= constant.Int && want.Kind() >= constant.Int) {
		return false
	}
	return constant.Compare(actual, token.EQL, want)
}

func matchSignature(base, next *types.Signature, actual []types.Type) (types.Instance, bool) {
	args := slices.Clone(actual)
	for i := range args {
		old, param := base.Params().At(i).Type(), next.Params().At(i).Type()
		if types.Identical(old, param) {
			args[i] = old
		} else if args[i] == nil || types.IsInterface(args[i]) || args[i] == types.Typ[types.UntypedNil] {
			return types.Instance{}, false
		}
	}
	inst := types.Instance{Type: next}
	if next.TypeParams().Len() != 0 {
		inst = infer(next, args)
		if inst.Type == nil {
			return types.Instance{}, false
		}
		next = inst.Type.(*types.Signature)
	}
	for i, arg := range args {
		if !types.Identical(arg, next.Params().At(i).Type()) {
			return types.Instance{}, false
		}
	}
	return inst, true
}

// Use ordinary Go inference and constraint checking in an isolated scope.
// No bindings are added to the caller or the template declaration's package.
func infer(sig *types.Signature, args []types.Type) types.Instance {
	pkg := types.NewPackage("llgo/template", "templatecheck")
	id := ast.NewIdent("candidate")
	pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, id.Name, sig))
	call := &ast.CallExpr{Fun: id}
	for i, arg := range args {
		name := fmt.Sprintf("arg%d", i)
		pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, name, arg))
		call.Args = append(call.Args, ast.NewIdent(name))
	}
	info := &types.Info{Instances: make(map[*ast.Ident]types.Instance)}
	if types.CheckExpr(token.NewFileSet(), pkg, token.NoPos, call, info) != nil {
		return types.Instance{}
	}
	return info.Instances[id]
}

func moreSpecific(base *types.Signature, a, b candidate) bool {
	if b.guard != nil && (a.guard == nil || a.guard.param != b.guard.param ||
		!matchesGuard(a.guard.value, b.guard.value)) {
		return false
	}
	ap, bp := a.fn.Signature().Params(), b.fn.Signature().Params()
	strict := a.guard != nil && b.guard == nil
	for i := 0; i < ap.Len(); i++ {
		at, bt := ap.At(i).Type(), bp.At(i).Type()
		if types.Identical(at, bt) {
			continue
		}
		if !types.Identical(bt, base.Params().At(i).Type()) {
			return false
		}
		strict = true
	}
	return strict
}

func paramIndex(sig *types.Signature, name string) int {
	for i := 0; i < sig.Params().Len(); i++ {
		if sig.Params().At(i).Name() == name {
			return i
		}
	}
	return -1
}
