//go:build !llgo
// +build !llgo

package cl

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/goplus/llgo/ssa"
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
