// Package jsonspecialize lowers statically typed calls to encoding/json.Marshal.
package jsonspecialize

import (
	"go/ast"
	"go/types"
	"reflect"
	"strings"
	"unicode"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

type shape struct {
	typ    types.Type
	elem   *shape
	fields []field
}

type field struct {
	goName string
	name   string
	value  *shape
	omit   bool
}

type callSite struct {
	call  *ast.CallExpr
	shape *shape
}

type packagePlan struct {
	pkg   *packages.Package
	calls []callSite
}

// Rewrite uses the compact path when all calls to the legacy encoder can be
// lowered. Keeping a single encoder avoids duplicating the standard codec in
// programs that still need dynamic reflection or custom JSON methods.
func Rewrite(closure []*packages.Package, library, entry *packages.Package) ([]*packages.Package, error) {
	deps := make(map[string]*packages.Package)
	for _, p := range closure {
		deps[p.PkgPath] = p
	}
	for _, dep := range []string{"strconv", "unicode/utf8"} {
		if deps[dep] == nil || deps[dep].Types == nil {
			return nil, nil
		}
	}
	sortPkg := deps["sort"]
	if sortPkg == nil {
		sortPkg = deps["slices"]
	}
	if sortPkg == nil || sortPkg.Types == nil {
		return nil, nil
	}
	var plans []packagePlan
	for _, pkg := range closure {
		if pkg == library {
			continue
		}
		if pkg.Imports["encoding/json/v2"] != nil {
			return nil, nil
		}
		if pkg.Imports["encoding/json"] == nil {
			continue
		}
		plan, ok := planPackage(pkg, library.Types)
		if !ok {
			return nil, nil
		}
		if len(plan.calls) != 0 {
			if pkg != entry {
				return nil, nil
			}
			plans = append(plans, plan)
		}
	}
	if len(plans) == 0 {
		return nil, nil
	}
	var changed []*packages.Package
	for _, plan := range plans {
		if err := plan.apply(library.Imports["encoding/json/v2"] != nil, sortPkg, deps["strconv"], deps["unicode/utf8"]); err != nil {
			return nil, err
		}
		changed = append(changed, plan.pkg)
	}
	return changed, nil
}

func planPackage(pkg *packages.Package, library *types.Package) (packagePlan, bool) {
	plan := packagePlan{pkg: pkg}
	if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Imports["encoding/json"].Types != library {
		return plan, false
	}
	info := pkg.TypesInfo
	accepted := make(map[*ast.Ident]bool)
	var known typeutil.Map
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			id := calleeIdent(call.Fun)
			fn, _ := info.Uses[id].(*types.Func)
			if fn == nil || fn.Pkg() != library || fn.Name() != "Marshal" {
				return true
			}
			input := info.TypeOf(call.Args[0])
			if input == nil || llssa.HasTypeParam(input) {
				return true
			}
			if value := planShape(types.Default(input), &known); value != nil {
				plan.calls = append(plan.calls, callSite{call, value})
				accepted[id] = true
			}
			return true
		})
	}
	for id, obj := range info.Uses {
		if obj.Pkg() != library {
			continue
		}
		if fn, ok := obj.(*types.Func); ok {
			switch fn.Name() {
			case "Marshal":
				if !accepted[id] {
					return plan, false
				}
			case "MarshalIndent", "NewEncoder", "Unmarshal", "NewDecoder":
				// A dynamic codec still retains much of the same graph.
				return plan, false
			}
		}
	}
	if len(plan.calls) != 0 {
		for _, name := range []string{"append", "byte", "error", "int64", "len", "make", "nil", "string", "uint64"} {
			if pkg.Types.Scope().Lookup(name) != nil {
				return plan, false
			}
		}
	}
	return plan, true
}

func calleeIdent(expr ast.Expr) *ast.Ident {
	switch expr := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return expr
	case *ast.SelectorExpr:
		return expr.Sel
	}
	return nil
}

func planShape(t types.Type, known *typeutil.Map) *shape {
	t = types.Unalias(t)
	if value := known.At(t); value != nil {
		s, _ := value.(*shape)
		return s
	}
	// Keep encoder graphs finite. Recursive shapes (even through maps or
	// slices) can contain cycles, which need the standard encoder's errors.
	known.Set(t, false)
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj.Pkg() != nil && obj.Pkg().Path() == "encoding/json" && obj.Name() == "Number" {
			return nil
		}
	}
	for _, candidate := range []types.Type{t, types.NewPointer(t)} {
		methods := types.NewMethodSet(candidate)
		for i := 0; i < methods.Len(); i++ {
			switch methods.At(i).Obj().Name() {
			case "MarshalJSON", "MarshalText", "MarshalJSONTo":
				return nil
			}
		}
	}
	s := &shape{typ: t}
	switch base := t.Underlying().(type) {
	case *types.Basic:
		if base.Info()&(types.IsBoolean|types.IsInteger|types.IsString) == 0 || base.Info()&types.IsUntyped != 0 {
			return nil
		}
	case *types.Array:
		s.elem = planShape(base.Elem(), known)
	case *types.Slice:
		if b, ok := base.Elem().Underlying().(*types.Basic); ok && b.Kind() == types.Uint8 {
			return nil // The standard encoder treats byte slices as base64.
		}
		s.elem = planShape(base.Elem(), known)
	case *types.Pointer:
		s.elem = planShape(base.Elem(), known)
	case *types.Map:
		if !types.Identical(base.Key(), types.Typ[types.String]) {
			return nil
		}
		s.elem = planShape(base.Elem(), known)
	case *types.Struct:
		names := make(map[string]bool)
		for i := range base.NumFields() {
			member := base.Field(i)
			tag := reflect.StructTag(base.Tag(i)).Get("json")
			if tag == "-" {
				continue
			}
			if member.Embedded() {
				return nil
			}
			if !member.Exported() {
				continue
			}
			name, options, _ := strings.Cut(tag, ",")
			if name == "" {
				name = member.Name()
			}
			if names[name] || !validTag(name) || options != "" && options != "omitempty" {
				return nil
			}
			names[name] = true
			value := planShape(member.Type(), known)
			if value == nil {
				return nil
			}
			s.fields = append(s.fields, field{member.Name(), name, value, options == "omitempty"})
		}
	default:
		return nil
	}
	switch t.Underlying().(type) {
	case *types.Array, *types.Slice, *types.Pointer, *types.Map:
		if s.elem == nil {
			return nil
		}
	}
	known.Set(t, s)
	return s
}

func validTag(name string) bool {
	const punctuation = "!#$%&()*+-./:;<=>?@[]^_{|}~ "
	for _, r := range name {
		if !strings.ContainsRune(punctuation, r) && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
