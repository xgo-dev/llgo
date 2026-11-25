package cl

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNeedsNamedClosureChange(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	params := types.NewTuple(types.NewParam(token.NoPos, pkg, "value", types.Typ[types.Int]))
	results := types.NewTuple(types.NewParam(token.NoPos, pkg, "", types.Typ[types.Bool]))
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)
	named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Func", nil), sig, nil)
	different := types.NewSignatureType(nil, nil, nil, nil, results, false)

	tests := []struct {
		name string
		got  types.Type
		want types.Type
		ok   bool
	}{
		{name: "anonymous to named", got: sig, want: named, ok: true},
		{name: "named to anonymous", got: named, want: sig, ok: true},
		{name: "identical named", got: named, want: named},
		{name: "different signatures", got: different, want: named},
		{name: "non functions", got: types.Typ[types.Int], want: types.Typ[types.Int64]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsNamedClosureChange(tt.got, tt.want); got != tt.ok {
				t.Fatalf("needsNamedClosureChange(%v, %v) = %v, want %v", tt.got, tt.want, got, tt.ok)
			}
		})
	}
}

func TestNamedClosureValuesKeepTheirDeclaredType(t *testing.T) {
	const source = `package main

type Func func(int) bool

func call(fn Func) bool { return fn(1) }

func direct() bool {
	want := 1
	return call(func(got int) bool { return got == want })
}

func iterator() func(Func) {
	return func(yield Func) { _ = yield(1) }
}

func ranged() int {
	n := 0
	for range iterator() { n++ }
	return n
}

func main() {
	if !direct() || ranged() != 1 { panic("named closure conversion failed") }
}
`
	_, module := mustCompileLLPkgFromSrc(t, source)
	if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("named closure module is invalid: %v\n%s", err, module.String())
	}
}
