package fmtspecialize

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

// RuntimePath is the shared runtime support for generated formatters.
const RuntimePath = "github.com/xgo-dev/llgo/runtime/internal/llgofmt"

type formatSource struct {
	owner   *types.Package
	imports typesynth.Importer
	types   typeutil.Map
	sizes   types.Sizes
	top     typeutil.Map
	nested  typeutil.Map
	compare typeutil.Map
	tops    typeutil.Map
	funcs   strings.Builder
	methods map[*types.Func]bool
	nextID  int
}

// AddValueSource generates typed functions for concrete default-format operands.
// The second result reports whether the generated file imports runtime support.
func AddValueSource(p *packages.Package, runtime *types.Package) (*typeutil.Map, bool, error) {
	for _, name := range []string{"append", "bool", "byte", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64",
		"complex128", "string", "len", "real", "imag", "nil", "__llgo_fmt_runtime", "__llgo_fmt_unsafe"} {
		if p.Types.Scope().Lookup(name) != nil {
			return nil, false, nil
		}
	}
	gen := &formatSource{
		owner: p.Types, imports: typesynth.Importer{RuntimePath: runtime, "unsafe": types.Unsafe},
		sizes: p.TypesSizes, methods: valueMethods(p),
	}
	for _, file := range p.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 || call.Ellipsis.IsValid() {
				return true
			}
			fun := ast.Unparen(call.Fun)
			var name *ast.Ident
			switch x := fun.(type) {
			case *ast.Ident:
				name = x
			case *ast.SelectorExpr:
				name = x.Sel
			}
			fn, _ := p.TypesInfo.Uses[name].(*types.Func)
			if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != "fmt" || fn.Name() != "Sprintf" {
				return true
			}
			f := p.TypesInfo.Types[call.Args[0]].Value
			if f == nil || f.Kind() != constant.String || constant.StringVal(f) != "%v" {
				return true
			}
			typ := p.TypesInfo.TypeOf(call.Args[1])
			if typ == nil {
				return true
			}
			typ = types.Default(typ)
			if gen.tops.At(typ) == nil && !llssa.HasTypeParam(typ) && defaultFormatType(typ, true, make(map[types.Type]bool), gen.methods) {
				gen.addTop(typ)
			}
			return true
		})
	}
	if gen.tops.Len() == 0 {
		return nil, false, nil
	}
	var src strings.Builder
	fmt.Fprintf(&src, "package %s\n", p.Name)
	// Bare append and typed operations may be the entire generated body.
	usesRuntime := strings.Contains(gen.funcs.String(), "__llgo_fmt_runtime.")
	if usesRuntime {
		fmt.Fprintf(&src, "import __llgo_fmt_runtime %q\n", RuntimePath)
	}
	if strings.Contains(gen.funcs.String(), "__llgo_fmt_unsafe.") {
		src.WriteString("import __llgo_fmt_unsafe \"unsafe\"\n")
	}
	src.WriteString(gen.funcs.String())
	// Only the generated file requires the newer formatter language features.
	if err := typesynth.AddFile(p, "__llgo_format.go", src.String(), gen.imports, "go1.21"); err != nil {
		return nil, false, err
	}
	return &gen.tops, usesRuntime, nil
}

func (g *formatSource) nextName() string {
	for {
		g.nextID++
		name := fmt.Sprintf("__llgo_fmt_%d", g.nextID)
		if g.owner.Scope().Lookup(name) == nil {
			return name
		}
	}
}

// Bind a compiler-only name to the exact checked type, including local names
// and unexported fields in types declared by a dependency.
func (g *formatSource) typeName(t types.Type) string {
	if name := g.types.At(t); name != nil {
		return name.(string)
	}
	name := g.nextName()
	obj := types.NewTypeName(token.NoPos, g.owner, name, nil)
	types.NewAlias(obj, types.Unalias(t))
	g.owner.Scope().Insert(obj)
	g.types.Set(t, name)
	return name
}

func (g *formatSource) addTop(typ types.Type) {
	name := g.nextName()
	g.tops.Set(typ, name)
	render := g.appendFunc(typ, true)
	fmt.Fprintf(&g.funcs, "func %s(v %s) string { return string(%s(nil,v)) }\n", name, g.typeName(typ), render)
}

func (g *formatSource) appendFunc(typ types.Type, top bool) string {
	cache := &g.nested
	if top {
		cache = &g.top
	}
	if name := cache.At(typ); name != nil {
		return name.(string)
	}
	name := g.nextName()
	cache.Set(typ, name)
	var b strings.Builder
	fmt.Fprintf(&b, "func %s(dst []byte, v %s) []byte {\n", name, g.typeName(typ))
	if method, _ := defaultValueMethod(typ, g.methods); method != "" {
		fmt.Fprintf(&b, "return append(dst,v.%s()...)\n}\n", method)
		g.funcs.WriteString(b.String())
		return name
	}
	switch u := typ.Underlying().(type) {
	case *types.Basic:
		g.appendBasic(&b, u.Kind())
	case *types.Array:
		g.appendSequence(&b, u.Elem())
	case *types.Slice:
		g.appendSequence(&b, u.Elem())
	case *types.Struct:
		b.WriteString("dst=append(dst,'{')\n")
		for i := 0; i < u.NumFields(); i++ {
			if i != 0 {
				b.WriteString("dst=append(dst,' ')\n")
			}
			field := u.Field(i)
			var arg string
			if field.Name() == "_" {
				arg = fmt.Sprintf("zero%d", i)
				fmt.Fprintf(&b, "var %s %s\n", arg, g.typeName(field.Type()))
			} else {
				arg = g.selectField("v", u, i)
			}
			fmt.Fprintf(&b, "dst=%s(dst,%s)\n", g.appendFunc(field.Type(), false), arg)
		}
		b.WriteString("return append(dst,'}')\n")
	case *types.Map:
		cmp := g.compareFunc(u.Key())
		key, elem := g.appendFunc(u.Key(), false), g.appendFunc(u.Elem(), false)
		fmt.Fprintf(&b, "return __llgo_fmt_runtime.AppendMap[%s,%s](dst,v,%s,%s,%s)\n",
			g.typeName(u.Key()), g.typeName(u.Elem()), cmp, key, elem)
	case *types.Chan:
		fmt.Fprintf(&b, "return __llgo_fmt_runtime.AppendPointer(dst,%s)\n", formatChanAddress("v"))
	case *types.Pointer:
		if top && formatDereference(u.Elem()) {
			fmt.Fprintf(&b, "if v!=nil { return %s(append(dst,'&'),*v) }\n", g.appendFunc(u.Elem(), false))
		}
		g.appendPointer(&b)
	}
	b.WriteString("}\n")
	g.funcs.WriteString(b.String())
	return name
}

func (g *formatSource) appendBasic(b *strings.Builder, kind types.BasicKind) {
	switch kind {
	case types.Bool:
		b.WriteString("if v { return append(dst,\"true\"...) }; return append(dst,\"false\"...)\n")
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
		b.WriteString("return __llgo_fmt_runtime.AppendInt(dst,int64(v))\n")
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		b.WriteString("return __llgo_fmt_runtime.AppendUint(dst,uint64(v))\n")
	case types.String:
		b.WriteString("return append(dst,string(v)...)\n")
	case types.Float32:
		b.WriteString("return __llgo_fmt_runtime.AppendFloat(dst,float64(v),32)\n")
	case types.Float64:
		b.WriteString("return __llgo_fmt_runtime.AppendFloat(dst,float64(v),64)\n")
	case types.Complex64:
		b.WriteString("return __llgo_fmt_runtime.AppendComplex(dst,complex128(v),64)\n")
	case types.Complex128:
		b.WriteString("return __llgo_fmt_runtime.AppendComplex(dst,complex128(v),128)\n")
	case types.UnsafePointer:
		g.appendPointer(b)
	}
}

func (g *formatSource) appendPointer(b *strings.Builder) {
	b.WriteString("return __llgo_fmt_runtime.AppendPointer(dst,uintptr(__llgo_fmt_unsafe.Pointer(v)))\n")
}

func (g *formatSource) appendSequence(b *strings.Builder, elem types.Type) {
	fmt.Fprintf(b, "dst=append(dst,'[')\nfor i:=range v { if i!=0 { dst=append(dst,' ') }; dst=%s(dst,v[i]) }\nreturn append(dst,']')\n",
		g.appendFunc(elem, false))
}

func (g *formatSource) compareFunc(typ types.Type) string {
	if name := g.compare.At(typ); name != nil {
		return name.(string)
	}
	name := g.nextName()
	g.compare.Set(typ, name)
	var b strings.Builder
	fmt.Fprintf(&b, "func %s(a,b %s) int {\n", name, g.typeName(typ))
	switch u := typ.Underlying().(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Bool:
			b.WriteString("if a==b {return 0}; if !a {return -1}; return 1\n")
		case types.Complex64, types.Complex128:
			b.WriteString("if c:=__llgo_fmt_runtime.Compare(real(a),real(b)); c!=0 {return c}; return __llgo_fmt_runtime.Compare(imag(a),imag(b))\n")
		case types.UnsafePointer:
			b.WriteString("return __llgo_fmt_runtime.Compare(uintptr(a),uintptr(b))\n")
		default:
			b.WriteString("return __llgo_fmt_runtime.Compare(a,b)\n")
		}
	case *types.Chan:
		fmt.Fprintf(&b, "return __llgo_fmt_runtime.Compare(%s,%s)\n", formatChanAddress("a"), formatChanAddress("b"))
	case *types.Pointer:
		b.WriteString("return __llgo_fmt_runtime.Compare(uintptr(__llgo_fmt_unsafe.Pointer(a)),uintptr(__llgo_fmt_unsafe.Pointer(b)))\n")
	case *types.Array:
		fmt.Fprintf(&b, "for i:=range a { if c:=%s(a[i],b[i]); c!=0 { return c } }; return 0\n", g.compareFunc(u.Elem()))
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			f := u.Field(i)
			if f.Name() != "_" {
				fmt.Fprintf(&b, "if c:=%s(%s,%s); c!=0 { return c }\n",
					g.compareFunc(f.Type()), g.selectField("a", u, i), g.selectField("b", u, i))
			}
		}
		b.WriteString("return 0\n")
	}
	b.WriteString("}\n")
	g.funcs.WriteString(b.String())
	return name
}

func (g *formatSource) selectField(value string, typ *types.Struct, field int) string {
	f := typ.Field(field)
	if f.Exported() || f.Pkg() == g.owner {
		return value + "." + f.Name()
	}
	// Use the target layout for a field whose selector is private to a dependency.
	fields := make([]*types.Var, typ.NumFields())
	for i := range fields {
		fields[i] = typ.Field(i)
	}
	off := g.sizes.Offsetsof(fields)[field]
	return fmt.Sprintf("*(*%s)(__llgo_fmt_unsafe.Add(__llgo_fmt_unsafe.Pointer(&%s),%d))",
		g.typeName(f.Type()), value, off)
}

func formatChanAddress(value string) string {
	return "*(*uintptr)(__llgo_fmt_unsafe.Pointer(&" + value + "))"
}
