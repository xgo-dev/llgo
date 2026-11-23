package compile

import (
	"go/ast"
	"go/types"
	"reflect"
	"strings"
	"unicode"

	"github.com/xgo-dev/llgo/internal/build/calltemplate"
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

var Generator = calltemplate.Generator{
	Package:           "encoding/json",
	FallbackEntries:   []string{"MarshalIndent", "NewEncoder", "NewDecoder", "LlgoDecodeTyped"},
	ConflictingImport: "encoding/json/v2",
	Apply:             rewriteJSON,
}

func rewriteJSON(input calltemplate.Generation) (bool, error) {
	entry, library := input.Entry, input.Library
	deps := make(map[string]*packages.Package)
	for _, pkg := range input.Packages {
		deps[pkg.PkgPath] = pkg
	}
	for _, dep := range []string{"strconv", "unicode/utf8"} {
		if deps[dep] == nil || deps[dep].Types == nil {
			return false, nil
		}
	}
	sortPkg := deps["sort"]
	if sortPkg == nil {
		sortPkg = deps["slices"]
	}
	if sortPkg == nil || sortPkg.Types == nil {
		return false, nil
	}
	decodeSupported := library.Types.Scope().Lookup("LlgoDecodeTyped") != nil && deps["strings"] != nil
	plan, ok := planJSONPackage(entry, input.Calls, decodeSupported)
	if !ok {
		return false, nil
	}
	return true, plan.apply(library.Imports["encoding/json/v2"] != nil, sortPkg, deps["strconv"], deps["unicode/utf8"], library, deps["strings"])
}

func planJSONPackage(pkg *packages.Package, calls []calltemplate.GeneratedCall, decodeSupported bool) (packagePlan, bool) {
	plan := packagePlan{pkg: pkg}
	var known typeutil.Map
	for _, c := range calls {
		var input types.Type
		call := c.Call
		decode := c.Candidate.Name() == "llgoUnmarshal"
		switch {
		case c.Candidate.Name() == "llgoMarshal" && len(call.Args) == 1:
			input = pkg.TypesInfo.TypeOf(call.Args[0])
		case decode && decodeSupported && len(call.Args) == 2:
			// Preserve the type in InvalidUnmarshalError for named pointers.
			if ptr, ok := types.Unalias(pkg.TypesInfo.TypeOf(call.Args[1])).(*types.Pointer); ok {
				input = ptr.Elem()
			}
		}
		if input == nil || llssa.HasTypeParam(input) {
			return plan, false
		}
		value := planShape(types.Default(input), &known)
		if value == nil || decode && !canDecode(value) {
			return plan, false
		}
		plan.calls = append(plan.calls, callSite{call, value, decode})
	}
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
		for i := 0; i < base.NumFields(); i++ {
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
