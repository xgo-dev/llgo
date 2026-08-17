//go:build !llgo
// +build !llgo

package cl

import (
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	gossa "golang.org/x/tools/go/ssa"
)

func newNilCompileContext(t *testing.T) (*context, llssa.Builder) {
	t.Helper()
	prog := newLLSSAProg(t)
	pkg := prog.NewPackage("foo", "foo")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fn := pkg.NewFunc("test", sig, llssa.InGo)
	b := fn.MakeBody(1)
	ctx := &context{
		prog:  prog,
		pkg:   pkg,
		fn:    fn,
		bvals: make(map[gossa.Value]llssa.Expr),
	}
	return ctx, b
}

func TestCompileNilBinOpAndHelpers(t *testing.T) {
	ctx, b := newNilCompileContext(t)
	untypedNil := gossa.NewConst(nil, types.Typ[types.UntypedNil])
	typedNilPtr := gossa.NewConst(nil, types.NewPointer(types.Typ[types.Int]))
	one := gossa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	two := gossa.NewConst(constant.MakeInt64(2), types.Typ[types.Int])

	if !isUntypedNilConst(untypedNil) {
		t.Fatal("untyped nil const not detected")
	}
	if isUntypedNilConst(typedNilPtr) {
		t.Fatal("typed nil pointer should not be treated as untyped nil")
	}
	if isUntypedNilConst(one) {
		t.Fatal("non-nil const should not be treated as untyped nil")
	}
	if isUntypedNilConst(&gossa.Parameter{}) {
		t.Fatal("non-const value should not be treated as untyped nil")
	}

	if ret := ctx.nilOf(types.NewPointer(types.Typ[types.Int])); ret.IsNil() {
		t.Fatal("nilOf returned an empty expression")
	}
	if ret := ctx.compileValueAs(b, untypedNil, types.NewPointer(types.Typ[types.Int])); ret.IsNil() {
		t.Fatal("compileValueAs did not lower untyped nil")
	}
	if ret := ctx.compileValueAs(b, one, types.Typ[types.Int]); ret.IsNil() {
		t.Fatal("compileValueAs did not compile non-nil const")
	}

	nilableTypes := []struct {
		name string
		typ  types.Type
	}{
		{"pointer", types.NewPointer(types.Typ[types.Int])},
		{"slice", types.NewSlice(types.Typ[types.Int])},
		{"map", types.NewMap(types.Typ[types.Int], types.Typ[types.String])},
		{"func", types.NewSignatureType(nil, nil, nil, nil, nil, false)},
		{"chan", types.NewChan(types.SendRecv, types.Typ[types.Int])},
	}
	for _, tt := range nilableTypes {
		t.Run(tt.name, func(t *testing.T) {
			typedNil := gossa.NewConst(nil, tt.typ)
			for _, tc := range []struct {
				name string
				op   token.Token
				x    gossa.Value
				y    gossa.Value
			}{
				{"left-untyped-nil-eq", token.EQL, untypedNil, typedNil},
				{"right-untyped-nil-eq", token.EQL, typedNil, untypedNil},
				{"right-untyped-nil-neq", token.NEQ, typedNil, untypedNil},
			} {
				ret := ctx.compileInstrOrValue(b, &gossa.BinOp{
					Op: tc.op,
					X:  tc.x,
					Y:  tc.y,
				}, false)
				if ret.IsNil() {
					t.Fatalf("%s lowered to an empty expression", tc.name)
				}
			}
		})
	}

	for _, op := range []token.Token{token.EQL, token.NEQ} {
		ret := ctx.compileInstrOrValue(b, &gossa.BinOp{
			Op: op,
			X:  untypedNil,
			Y:  untypedNil,
		}, false)
		if ret.IsNil() {
			t.Fatalf("nil %s nil lowered to an empty expression", op)
		}
	}

	ret := ctx.compileInstrOrValue(b, &gossa.BinOp{
		Op: token.ADD,
		X:  one,
		Y:  two,
	}, false)
	if ret.IsNil() {
		t.Fatal("non-nil BinOp lowered to an empty expression")
	}
}

func TestCompileNilInstructionLoweringBranches(t *testing.T) {
	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

import "unsafe"

type P *int

func change(p *int) P {
	return P(p)
}

func convert(p *int) unsafe.Pointer {
	return unsafe.Pointer(p)
}

func iface(v int) any {
	return v
}
`)
	ctx, b := newNilCompileContext(t)
	untypedNil := gossa.NewConst(nil, types.Typ[types.UntypedNil])

	change := findFirstInstr[*gossa.ChangeType](t, ssaPkg.Func("change"))
	change.X = untypedNil
	if ret := ctx.compileInstrOrValue(b, change, false); ret.IsNil() {
		t.Fatal("ChangeType untyped nil lowered to an empty expression")
	}

	convert := findFirstInstr[*gossa.Convert](t, ssaPkg.Func("convert"))
	convert.X = untypedNil
	if ret := ctx.compileInstrOrValue(b, convert, false); ret.IsNil() {
		t.Fatal("Convert untyped nil lowered to an empty expression")
	}

	makeInterface := findFirstInstr[*gossa.MakeInterface](t, ssaPkg.Func("iface"))
	makeInterface.X = untypedNil
	if ret := ctx.compileInstrOrValue(b, makeInterface, false); ret.IsNil() {
		t.Fatal("MakeInterface untyped nil lowered to an empty expression")
	}
}

func TestFoldConstComparison(t *testing.T) {
	a := gossa.NewConst(constant.MakeString("a"), types.Typ[types.String])
	b := gossa.NewConst(constant.MakeString("b"), types.Typ[types.String])
	for _, tt := range []struct {
		op   token.Token
		want bool
	}{
		{token.EQL, false},
		{token.NEQ, true},
		{token.LSS, true},
		{token.LEQ, true},
		{token.GTR, false},
		{token.GEQ, false},
	} {
		if got, ok := foldConstComparison(&gossa.BinOp{Op: tt.op, X: a, Y: b}); !ok || got != tt.want {
			t.Errorf("foldConstComparison(%s) = %v, %v; want %v, true", tt.op, got, ok, tt.want)
		}
	}
	if _, ok := foldConstComparison(&gossa.BinOp{Op: token.ADD, X: a, Y: b}); ok {
		t.Fatal("non-comparison operation was folded")
	}
	if _, ok := foldConstComparison(&gossa.BinOp{Op: token.EQL, X: &gossa.Parameter{}, Y: b}); ok {
		t.Fatal("non-constant operand was folded")
	}
	if _, ok := foldConstComparison(&gossa.BinOp{
		Op: token.EQL,
		X:  gossa.NewConst(nil, types.NewPointer(types.Typ[types.Int])),
		Y:  gossa.NewConst(nil, types.NewPointer(types.Typ[types.Int])),
	}); ok {
		t.Fatal("nil comparison was folded through go/constant")
	}

	for _, tt := range []struct {
		name string
		x, y constant.Value
	}{
		{"integer", constant.MakeInt64(0), constant.MakeInt64(1)},
		{"string", constant.MakeString("a"), constant.MakeString("b")},
		{"rune", constant.MakeInt64('☃'), constant.MakeInt64('☀')},
		{"float", constant.MakeFloat64(0), constant.MakeFloat64(1)},
		{
			"complex",
			constant.MakeFromLiteral("1i", token.IMAG, 0),
			constant.MakeFromLiteral("-1i", token.IMAG, 0),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			typ := types.Typ[types.UntypedInt]
			if tt.name == "string" {
				typ = types.Typ[types.UntypedString]
			}
			if got, ok := foldConstComparison(&gossa.BinOp{
				Op: token.EQL,
				X:  gossa.NewConst(tt.x, typ),
				Y:  gossa.NewConst(tt.y, typ),
			}); !ok || got {
				t.Fatalf("foldConstComparison(%s equality) = %v, %v; want false, true", tt.name, got, ok)
			}
		})
	}
}

func TestCompileFoldsConstComparisons(t *testing.T) {
	_, mod := mustCompileLLPkgFromSrc(t, `
package foo

func intEqual() bool     { return 0 == 1 }
func stringEqual() bool  { return "a" == "b" }
func runeEqual() bool    { return '☃' == '☀' }
func floatEqual() bool   { return 0.0 == 1.0 }
func complexEqual() bool { return 1i == -1i }
func stringLess() bool   { return "a" < "b" }
`)
	ir := mod.String()
	if strings.Contains(ir, "StringEqual") {
		t.Fatalf("constant string comparison called the runtime helper:\n%s", ir)
	}
	for _, name := range []string{"intEqual", "stringEqual", "runeEqual", "floatEqual", "complexEqual"} {
		fn := llvmFunction(t, ir, "foo."+name)
		if !strings.Contains(fn, "ret i1 false") {
			t.Fatalf("%s was not folded to false:\n%s", name, fn)
		}
	}
	if fn := llvmFunction(t, ir, "foo.stringLess"); !strings.Contains(fn, "ret i1 true") {
		t.Fatalf("stringLess was not folded to true:\n%s", fn)
	}
}

func TestCompileGenericNilSwitchAndCompare(t *testing.T) {
	mustCompileLLPkgFromSrc(t, `
package foo

func entry() {
	f[int]()
}

func f[T any]() {
	switch []T(nil) {
	case nil:
	default:
		panic("slice switch")
	}

	switch (func() T)(nil) {
	case nil:
	default:
		panic("func switch")
	}

	switch (map[int]T)(nil) {
	case nil:
	default:
		panic("map switch")
	}

	if []T(nil) != nil {
		panic("slice compare")
	}
	if (func() T)(nil) != nil {
		panic("func compare")
	}
	if (map[int]T)(nil) != nil {
		panic("map compare")
	}
}
`)
}

func findFirstInstr[T gossa.Instruction](t *testing.T, fn *gossa.Function) T {
	t.Helper()
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if typed, ok := instr.(T); ok {
				return typed
			}
		}
	}
	var zero T
	t.Fatalf("missing %T in %s", zero, fn.Name())
	return zero
}
