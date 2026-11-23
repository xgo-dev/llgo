// Package jsonspecialize lowers statically typed encoding/json calls.
package jsonspecialize

import (
	"go/ast"
	"go/types"
	"reflect"
	"strings"
	"unicode"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
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
	call   *ast.CallExpr
	shape  *shape
	decode bool
}

type packagePlan struct {
	pkg   *packages.Package
	calls []callSite
}

// Rewrite uses the compact path when all calls to the legacy codec can be
// lowered. Keeping one codec avoids duplicating the standard implementation in
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
	decodeSupported := library.Types.Scope().Lookup("LlgoDecodeTyped") != nil && deps["strings"] != nil
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
		plan, ok := planPackage(pkg, library.Types, decodeSupported)
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
		if err := plan.apply(library.Imports["encoding/json/v2"] != nil, sortPkg, deps["strconv"], deps["unicode/utf8"], library, deps["strings"]); err != nil {
			return nil, err
		}
		changed = append(changed, plan.pkg)
	}
	return changed, nil
}

func planPackage(pkg *packages.Package, library *types.Package, decodeSupported bool) (packagePlan, bool) {
	plan := packagePlan{pkg: pkg}
	if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Imports["encoding/json"].Types != library {
		return plan, false
	}
	var known typeutil.Map
	calls, closed := typesynth.PlanClosedCalls(pkg, library, jsonEntry, func(call *ast.CallExpr, name string) (callSite, bool) {
		var input types.Type
		decode := name == "Unmarshal"
		switch {
		case name == "Marshal" && len(call.Args) == 1:
			input = pkg.TypesInfo.TypeOf(call.Args[0])
		case decode && decodeSupported && len(call.Args) == 2:
			// Preserve the type in InvalidUnmarshalError for named pointers.
			if ptr, ok := types.Unalias(pkg.TypesInfo.TypeOf(call.Args[1])).(*types.Pointer); ok {
				input = ptr.Elem()
			}
		}
		if input == nil || llssa.HasTypeParam(input) {
			return callSite{}, false
		}
		value := planShape(types.Default(input), &known)
		return callSite{call: call, shape: value, decode: decode}, value != nil && (!decode || canDecode(value))
	})
	if !closed {
		return plan, false
	}
	plan.calls = calls
	if len(plan.calls) != 0 {
		reserved := []string{"append", "byte", "error", "int64", "len", "make", "nil", "string", "uint64"}
		if decodeSupported {
			reserved = append(reserved, "cap", "new")
		}
		for _, name := range reserved {
			if pkg.Types.Scope().Lookup(name) != nil {
				return plan, false
			}
		}
	}
	return plan, true
}

func jsonEntry(name string) bool {
	switch name {
	case "Marshal", "Unmarshal", "MarshalIndent", "NewEncoder", "NewDecoder", "LlgoDecodeTyped":
		return true
	}
	return false
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
