package calltemplate

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"math"
	"reflect"
	"slices"
	"strings"
	"text/template"
	"unicode"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
	"golang.org/x/tools/go/types/typeutil"
)

var errExpansionRejected = errors.New("call is outside the template's supported inputs")

type expansion struct {
	pkg       *packages.Package
	deps      map[string]*packages.Package
	prefix    string
	serial    int
	aliases   typeutil.Map
	imports   map[string]string
	functions strings.Builder
	// Each helper is shared across callers with the same instantiated type.
	helpers map[*template.Template]map[string]*typeutil.Map
	active  map[string]bool
	roots   map[*template.Template][]expandedCall
}

type expandedCall struct {
	sig   *types.Signature
	facts []argumentFacts
	name  string
}

type argumentFacts struct {
	NonNil      bool
	SliceLength int64 // -1 for unknown
}

type expansionContext struct {
	g      *expansion
	rule   *template.Template
	Name   string
	Type   types.Type
	Params []argumentFacts
}

func newExpansion(pkg *packages.Package, deps map[string]*packages.Package, prefix string) *expansion {
	return &expansion{
		pkg: pkg, deps: deps, prefix: "__llgo_" + prefix + "_",
		imports: make(map[string]string), roots: make(map[*template.Template][]expandedCall),
		helpers: make(map[*template.Template]map[string]*typeutil.Map), active: make(map[string]bool),
	}
}

func (g *expansion) expand(rule structuralRule, sig *types.Signature, call *ast.CallExpr) (string, error) {
	facts := make([]argumentFacts, len(call.Args))
	for i, arg := range call.Args {
		facts[i] = argumentFacts{nonNilPointer(g.pkg.TypesInfo, arg), arraySliceLength(g.pkg.TypesInfo, arg)}
	}
	for _, prev := range g.roots[rule.body] {
		if types.Identical(prev.sig, sig) && slices.Equal(prev.facts, facts) {
			return prev.name, nil
		}
	}
	name := g.name()
	ctx := expansionContext{g: g, rule: rule.body, Name: name, Type: sig, Params: facts}
	var body strings.Builder
	if err := rule.body.Execute(&body, ctx); err != nil {
		return "", err
	}
	fmt.Fprintf(&g.functions, "func %s(", name)
	for i := range sig.Params().Len() {
		if i > 0 {
			g.functions.WriteByte(',')
		}
		fmt.Fprintf(&g.functions, "a%d %s", i, ctx.TypeName(sig.Params().At(i).Type()))
	}
	g.functions.WriteString(") (")
	for i := range sig.Results().Len() {
		if i > 0 {
			g.functions.WriteByte(',')
		}
		g.functions.WriteString(ctx.TypeName(sig.Results().At(i).Type()))
	}
	fmt.Fprintf(&g.functions, ") {\n%s\n}\n", body.String())
	g.roots[rule.body] = append(g.roots[rule.body], expandedCall{sig, facts, name})
	return name, nil
}

func (g *expansion) name() string {
	for {
		g.serial++
		name := fmt.Sprint(g.prefix, g.serial)
		if g.pkg.Types.Scope().Lookup(name) == nil {
			return name
		}
	}
}

func (g *expansion) apply(prefix string, calls []structuralCall) error {
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n", g.pkg.Name)
	imports := make(typesynth.Importer)
	var paths []string
	for path := range g.imports {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		dep := g.deps[path]
		fmt.Fprintf(&source, "import %s %q\n", g.imports[path], path)
		imports[path] = dep.Types
		g.pkg.Imports[path] = dep
	}
	g.aliases.Iterate(func(typ types.Type, value any) {
		obj := types.NewTypeName(token.NoPos, g.pkg.Types, value.(string), nil)
		types.NewAlias(obj, types.Unalias(typ))
		g.pkg.Types.Scope().Insert(obj)
	})
	source.WriteString(g.functions.String())
	if err := typesynth.AddFile(g.pkg, "__llgo_"+prefix+".go", source.String(), imports, ""); err != nil {
		return err
	}
	for _, call := range calls {
		fn := g.pkg.Types.Scope().Lookup(call.name).(*types.Func)
		typesynth.BindCall(g.pkg.TypesInfo, call.call, fn, types.Instance{Type: fn.Type()})
	}
	return nil
}

func (c expansionContext) Arg(i int) types.Type {
	return c.Type.(*types.Signature).Params().At(i).Type()
}

func (c expansionContext) TypeName(typ types.Type) string {
	if name, ok := c.g.aliases.At(typ).(string); ok {
		return name
	}
	name := c.g.name()
	c.g.aliases.Set(typ, name)
	return name
}

func (c expansionContext) Local() string { return c.g.name() }

func (c expansionContext) IntMax() int64 {
	if c.g.pkg.TypesSizes.Sizeof(types.Typ[types.Int]) == 4 {
		return math.MaxInt32
	}
	return math.MaxInt64
}

func (c expansionContext) HasImport(path string) bool {
	dep := c.g.deps[path]
	return dep != nil && dep.Types != nil
}

func (c expansionContext) HasObject(path, name string) bool {
	return c.HasImport(path) && c.g.deps[path].Types.Scope().Lookup(name) != nil
}

func (c expansionContext) Import(path string) (string, error) {
	if !c.HasImport(path) || path == c.g.pkg.PkgPath {
		return "", errExpansionRejected
	}
	if name := c.g.imports[path]; name != "" {
		return name, nil
	}
	name := c.g.name()
	c.g.imports[path] = name
	return name, nil
}

func (c expansionContext) RequireBuiltins(names ...string) (string, error) {
	for _, name := range names {
		if c.g.pkg.Types.Scope().Lookup(name) != nil {
			return "", errExpansionRejected
		}
	}
	return "", nil
}

// Emit expands a helper declaration once per template and instantiated type.
// A recursive expansion declines the call and leaves the fallback in place.
func (c expansionContext) Emit(name string, typ types.Type) (string, error) {
	entries := c.g.helpers[c.rule]
	if entries == nil {
		entries = make(map[string]*typeutil.Map)
		c.g.helpers[c.rule] = entries
	}
	cached := entries[name]
	if cached == nil {
		cached = new(typeutil.Map)
		entries[name] = cached
	}
	if helper, ok := cached.At(typ).(string); ok {
		if c.g.active[helper] {
			return "", errExpansionRejected
		}
		return helper, nil
	}
	if len(c.g.active) >= 256 {
		return "", errExpansionRejected
	}

	helper := c.g.name()
	cached.Set(typ, helper)
	c.g.active[helper] = true
	var source strings.Builder
	ctx := expansionContext{g: c.g, rule: c.rule, Name: helper, Type: typ}
	err := c.rule.ExecuteTemplate(&source, name, ctx)
	delete(c.g.active, helper)
	if err != nil {
		return "", err
	}
	c.g.functions.WriteString(source.String())
	c.g.functions.WriteByte('\n')
	return helper, nil
}

type fieldInfo struct {
	Name                           string
	Type                           types.Type
	Tag                            reflect.StructTag
	Embedded, Exported, Accessible bool
}

func (c expansionContext) Fields(typ types.Type) []fieldInfo {
	shape := typ.Underlying().(*types.Struct)
	fields := make([]fieldInfo, shape.NumFields())
	for i := range fields {
		v := shape.Field(i)
		fields[i] = fieldInfo{
			Name: v.Name(), Type: v.Type(), Tag: reflect.StructTag(shape.Tag(i)),
			Embedded: v.Embedded(), Exported: v.Exported(),
			Accessible: v.Exported() || v.Pkg() == c.g.pkg.Types,
		}
	}
	return fields
}

func (c expansionContext) SliceLength(i int) int64 { return c.Params[i].SliceLength }
func (c expansionContext) NonNil(i int) bool       { return c.Params[i].NonNil }

func typeKind(typ types.Type) string {
	switch t := typ.Underlying().(type) {
	case *types.Basic:
		return types.Typ[t.Kind()].Name()
	case *types.Array:
		return "array"
	case *types.Slice:
		return "slice"
	case *types.Struct:
		return "struct"
	case *types.Map:
		return "map"
	case *types.Pointer:
		return "pointer"
	case *types.Interface:
		return "interface"
	case *types.Chan:
		return "chan"
	case *types.Signature:
		return "func"
	}
	return ""
}

func typeElem(typ types.Type) types.Type {
	t, _ := typ.Underlying().(interface{ Elem() types.Type })
	if t == nil {
		return nil
	}
	return t.Elem()
}

func namedIn(typ types.Type, path string, names ...string) bool {
	named, ok := types.Unalias(typ).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == path && slices.Contains(names, named.Obj().Name())
}

func hasMethod(typ types.Type, names ...string) bool {
	for _, t := range []types.Type{typ, types.NewPointer(typ)} {
		set := types.NewMethodSet(t)
		for i := range set.Len() {
			if slices.Contains(names, set.At(i).Obj().Name()) {
				return true
			}
		}
	}
	return false
}

func basicInfo(typ types.Type, info types.BasicInfo) bool {
	t, ok := typ.Underlying().(*types.Basic)
	return ok && t.Info()&info != 0
}

func validRunes(s, extra string) bool {
	for _, r := range s {
		if !strings.ContainsRune(extra, r) && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func dictionary(args ...any) (map[string]any, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("dict expects key/value pairs")
	}
	result := make(map[string]any, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict key %d must be a string", i/2)
		}
		result[key] = args[i+1]
	}
	return result, nil
}

var expansionFunctions = template.FuncMap{
	"reject": func() (string, error) { return "", errExpansionRejected },
	"kind":   typeKind, "elem": typeElem, "namedIn": namedIn, "hasMethod": hasMethod,
	"key":         func(typ types.Type) types.Type { return typ.Underlying().(*types.Map).Key() },
	"arrayLength": func(typ types.Type) int64 { return typ.Underlying().(*types.Array).Len() },
	"isString":    func(typ types.Type) bool { return types.Identical(typ, types.Typ[types.String]) },
	"isInteger":   func(typ types.Type) bool { return basicInfo(typ, types.IsInteger) },
	"isUnsigned":  func(typ types.Type) bool { return basicInfo(typ, types.IsUnsigned) },
	"validRunes":  validRunes, "replace": strings.ReplaceAll,
	"stringType": func() types.Type { return types.Typ[types.String] },
	"split":      strings.SplitN,
	"dict":       dictionary,
	"list":       func(args ...any) []any { return args },
	"get":        func(m map[string]any, k string) any { return m[k] },
	"set":        func(m map[string]any, k string, value any) string { m[k] = value; return "" },
	"add":        func(a, b int64) int64 { return a + b },
	"sub":        func(a, b int64) int64 { return a - b },
	"mul":        func(a, b int64) int64 { return a * b },
	"div":        func(a, b int64) int64 { return a / b },
}
