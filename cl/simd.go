package cl

import (
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// The table describes operation differences; the receiver supplies lane shape.
// These families have baseline-safe LLVM implementations on all three targets.
var simd128Methods = map[string]token.Token{
	"Add": token.ADD, "Sub": token.SUB,
	"And": token.AND, "Or": token.OR, "Xor": token.XOR,
	"GetElem": token.LBRACK, "SetElem": token.ASSIGN,
}

func (p *context) simd128Method(fn *ssa.Function) (token.Token, bool) {
	switch p.prog.Target().GOARCH {
	case "amd64", "arm64", "wasm":
	default:
		return 0, false
	}
	recv := fn.Signature.Recv()
	if recv == nil || fn.Object() == nil {
		return 0, false
	}
	named, ok := types.Unalias(recv.Type()).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "simd/archsimd" || fn.Object().Pkg() != named.Obj().Pkg() {
		return 0, false
	}
	switch named.Obj().Name() {
	case "Int8x16", "Uint8x16", "Int16x8", "Uint16x8", "Int32x4", "Uint32x4", "Int64x2", "Uint64x2", "Float32x4", "Float64x2":
	default:
		return 0, false
	}
	op, ok := simd128Methods[fn.Name()]
	return op, ok
}

func (p *context) simd128Call(b llssa.Builder, fn *ssa.Function, args []ssa.Value) llssa.Expr {
	op, ok := p.simd128Method(fn)
	if !ok {
		panic("invalid SIMD128 intrinsic")
	}
	values := p.compileValues(b, args, fnNormal)
	switch op {
	case token.LBRACK:
		return b.SIMD128GetElem(values[0], values[1])
	case token.ASSIGN:
		return b.SIMD128SetElem(values[0], values[1], values[2])
	default:
		return b.SIMD128Binary(op, values[0], values[1])
	}
}
