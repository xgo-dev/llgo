package binaryspecialize

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// Keep the standard library error object, including interface identity when
// a caller compares errors. This declaration exists only in the checked AST.
func addShortBufferAccessor(pkg *packages.Package) (string, error) {
	errVar, _ := pkg.Types.Scope().Lookup("errBufferTooSmall").(*types.Var)
	if errVar == nil || !types.Identical(errVar.Type(), types.Universe.Lookup("error").Type()) {
		return "", nil
	}
	name := "LLGoBinaryShortBuffer"
	for pkg.Types.Scope().Lookup(name) != nil {
		name += "_"
	}
	source := fmt.Sprintf("package binary\nfunc %s() error { return errBufferTooSmall }\n", name)
	return name, typesynth.AddFile(pkg, "__llgo_binary_error.go", source, nil, "")
}

type codecKey struct {
	operation string
	order     types.Type
}

type codecSource struct {
	owner     *types.Package
	aliases   typeutil.Map
	codecs    map[codecKey]*typeutil.Map
	functions strings.Builder
	serial    int
	widen     string
}

func (p packageRewrites) apply(library *packages.Package, shortBuffer string) error {
	g := codecSource{owner: p.pkg.Types, codecs: make(map[codecKey]*typeutil.Map)}
	binaryAlias, mathAlias := g.nextName(), g.nextName()
	for i := range p.calls {
		call := &p.calls[i]
		key := codecKey{call.op, call.order}
		codecs := g.codecs[key]
		if codecs == nil {
			codecs = new(typeutil.Map)
			g.codecs[key] = codecs
		}
		if helper, ok := codecs.At(call.input).(string); ok {
			call.helper = helper
			continue
		}
		call.helper = g.nextName()
		codecs.Set(call.input, call.helper)
		g.emit(*call, binaryAlias+"."+shortBuffer, mathAlias)
	}

	if g.widen != "" {
		// Reflect widens float32 before the conversion back to wire bits.
		// Keep that boundary through LTO so signaling NaN payloads are
		// converted exactly as in the standard library's composite path.
		fmt.Fprintf(&g.functions, "//go:noinline\nfunc %s(v float32) float64 { return float64(v) }\n", g.widen)
	}
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n", p.pkg.Name)
	imports := make(typesynth.Importer)
	generated := g.functions.String()
	if strings.Contains(generated, binaryAlias+".") {
		fmt.Fprintf(&source, "import %s \"encoding/binary\"\n", binaryAlias)
		imports["encoding/binary"] = library.Types
	}
	if strings.Contains(generated, mathAlias+".") {
		fmt.Fprintf(&source, "import %s \"math\"\n", mathAlias)
		dep := library.Imports["math"]
		imports["math"] = dep.Types
		p.pkg.Imports["math"] = dep
	}
	source.WriteString(generated)
	if err := typesynth.AddFile(p.pkg, "__llgo_binary.go", source.String(), imports, ""); err != nil {
		return err
	}
	for _, call := range p.calls {
		old := call.call.Fun
		fn := p.pkg.Types.Scope().Lookup(call.helper).(*types.Func)
		id := &ast.Ident{Name: call.helper, NamePos: old.Pos()}
		tv := p.pkg.TypesInfo.Types[ast.Unparen(old)]
		tv.Type = fn.Type()
		p.pkg.TypesInfo.Uses[id] = fn
		p.pkg.TypesInfo.Types[id] = tv
		call.call.Fun = id
	}
	return nil
}

func (g *codecSource) nextName() string {
	for {
		g.serial++
		name := fmt.Sprintf("__llgo_binary_%d", g.serial)
		if g.owner.Scope().Lookup(name) == nil {
			return name
		}
	}
}

func (g *codecSource) typeName(typ types.Type) string {
	if name, ok := g.aliases.At(typ).(string); ok {
		return name
	}
	name := g.nextName()
	obj := types.NewTypeName(token.NoPos, g.owner, name, nil)
	types.NewAlias(obj, types.Unalias(typ))
	g.owner.Scope().Insert(obj)
	g.aliases.Set(typ, name)
	return name
}

func (g *codecSource) emit(call rewrite, shortBuffer, mathAlias string) {
	b := &g.functions
	value := "v"
	if call.ptr {
		value = "(*v)"
	}
	size := fmt.Sprint(call.wire.width)
	if call.slice {
		size = fmt.Sprintf("len(%s)*%s", value, size)
	}
	if call.op == "Size" {
		fmt.Fprintf(b, "func %s(v %s) int {\n", call.helper, g.typeName(call.input))
		if call.ptr {
			fmt.Fprintln(b, "if v==nil { return -1 }")
		}
		fmt.Fprintf(b, "return %s\n}\n", size)
		return
	}

	fmt.Fprintf(b, "func %s(buf []byte, order %s, v %s) (int,error) {\n", call.helper, g.typeName(call.order), g.typeName(call.input))
	fmt.Fprintf(b, "n := %s\nif len(buf)<n { return 0,%s() }\n", size, shortBuffer)
	decode := call.op == "Decode"
	if call.slice && call.wire.width != 0 {
		fmt.Fprintf(b, "for i:=range %s {\n", value)
		g.field(call.wire, value+"[i]", fmt.Sprintf("(i*%d)", call.wire.width), mathAlias, decode)
		fmt.Fprintln(b, "}")
	} else if !call.slice {
		g.field(call.wire, value, "0", mathAlias, decode)
	}
	fmt.Fprintln(b, "return n,nil\n}")
}

func (g *codecSource) field(wire *layout, value, offset, mathAlias string, decode bool) {
	if wire.width == 0 {
		return
	}
	b := &g.functions
	switch base := wire.typ.Underlying().(type) {
	case *types.Array:
		i := g.nextName()
		fmt.Fprintf(b, "for %s:=range %s {\n", i, value)
		g.field(wire.elem, value+"["+i+"]", fmt.Sprintf("(%s+%s*%d)", offset, i, wire.elem.width), mathAlias, decode)
		fmt.Fprintln(b, "}")
	case *types.Struct:
		for _, field := range wire.fields {
			next := addOffset(offset, field.offset)
			if field.field.Name() != "_" {
				g.field(field.value, value+"."+field.field.Name(), next, mathAlias, decode)
			} else if !decode && field.value.width != 0 {
				i := g.nextName()
				fmt.Fprintf(b, "for %s:=%s; %s<%s; %s++ { buf[%s]=0 }\n", i, next, i, addOffset(next, field.value.width), i, i)
			}
		}
	case *types.Basic:
		g.basic(wire, base.Kind(), value, offset, mathAlias, decode)
	}
}

func addOffset(offset string, n int64) string {
	if n == 0 {
		return offset
	}
	return fmt.Sprintf("(%s+%d)", offset, n)
}

func (g *codecSource) basic(wire *layout, kind types.BasicKind, value, offset, mathAlias string, decode bool) {
	b := &g.functions
	width := wire.width
	// Capture both complex components before a write to a potentially
	// overlapping destination, as the reflective encoder does.
	switch kind {
	case types.Bool:
		if decode {
			fmt.Fprintf(b, "%s=%s(buf[%s]!=0)\n", value, g.typeName(wire.typ), offset)
		} else {
			fmt.Fprintf(b, "if %s { buf[%s]=1 } else { buf[%s]=0 }\n", value, offset, offset)
		}
		return
	case types.Complex64, types.Complex128:
		width /= 2
		second := addOffset(offset, width)
		if decode {
			re := g.decodeFloat(width, offset, mathAlias)
			im := g.decodeFloat(width, second, mathAlias)
			fmt.Fprintf(b, "%s=%s(complex(%s,%s))\n", value, g.typeName(wire.typ), re, im)
		} else {
			tmp := g.nextName()
			fmt.Fprintf(b, "%s:=%s\n", tmp, value)
			fmt.Fprintln(b, putBits(width, offset, g.encodeFloat(width, "real("+tmp+")", mathAlias)))
			fmt.Fprintln(b, putBits(width, second, g.encodeFloat(width, "imag("+tmp+")", mathAlias)))
		}
		return
	}
	if decode {
		expr := getBits(width, offset)
		if kind == types.Float32 || kind == types.Float64 {
			expr = g.decodeFloat(width, offset, mathAlias)
		}
		fmt.Fprintf(b, "%s=%s(%s)\n", value, g.typeName(wire.typ), expr)
	} else {
		expr := value
		if kind == types.Float32 || kind == types.Float64 {
			expr = g.encodeFloat(width, value, mathAlias)
		}
		fmt.Fprintln(b, putBits(width, offset, expr))
	}
}

func getBits(width int64, offset string) string {
	if width == 1 {
		return fmt.Sprintf("buf[%s]", offset)
	}
	return fmt.Sprintf("order.Uint%d(buf[%s:%s])", 8*width, offset, addOffset(offset, width))
}

func putBits(width int64, offset, value string) string {
	if width == 1 {
		return fmt.Sprintf("buf[%s]=byte(%s)", offset, value)
	}
	return fmt.Sprintf("order.PutUint%d(buf[%s:%s],uint%d(%s))", 8*width, offset, addOffset(offset, width), 8*width, value)
}

func (g *codecSource) encodeFloat(width int64, value, mathAlias string) string {
	if width == 4 {
		// The composite codec converts via reflect.Value.Float, which
		// widens the float32 before encoding it.
		return fmt.Sprintf("%s.Float32bits(%s)", mathAlias, g.roundFloat32(value))
	}
	return fmt.Sprintf("%s.Float64bits(float64(%s))", mathAlias, value)
}

func (g *codecSource) decodeFloat(width int64, offset, mathAlias string) string {
	if width == 4 {
		return g.roundFloat32(fmt.Sprintf("%s.Float32frombits(%s)", mathAlias, getBits(width, offset)))
	}
	return fmt.Sprintf("%s.Float64frombits(%s)", mathAlias, getBits(width, offset))
}

func (g *codecSource) roundFloat32(value string) string {
	if g.widen == "" {
		g.widen = g.nextName()
	}
	return fmt.Sprintf("float32(%s(float32(%s)))", g.widen, value)
}
