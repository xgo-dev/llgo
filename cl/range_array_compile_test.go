//go:build !llgo
// +build !llgo

package cl

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/goplus/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestSkipUnusedArrayDeref(t *testing.T) {
	if skipUnusedArrayDeref(&ssa.UnOp{Op: token.SUB}) {
		t.Fatal("non-deref unop should not be skipped")
	}

	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

	var sink int
	var calls int

	func rangeArray(p *[3]int) {
		for i := range *p {
			sink += i
		}
	}

	func nextArray() *[3]int {
		calls++
		return nil
	}

	func rangeArrayCall() {
		for i := range *nextArray() {
			sink += i
		}
	}

	func explicitDiscard(p *[3]int) {
		_ = *p
	}

	func copyArray(p *[3]int) [3]int {
		return *p
	}

func useNonArray(p *int) int {
	return *p
}
`)

	if !skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("rangeArray"), token.MUL, true)) {
		t.Fatal("range array deref should be skipped")
	}
	if !skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("rangeArrayCall"), token.MUL, true)) {
		t.Fatal("range array call deref should be skipped")
	}
	if skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("explicitDiscard"), token.MUL, true)) {
		t.Fatal("explicit array deref discard should not be skipped")
	}
	if skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("copyArray"), token.MUL, true)) {
		t.Fatal("referenced array deref should not be skipped")
	}
	if skipUnusedArrayDeref(findUnOp(t, ssaPkg.Func("useNonArray"), token.MUL, false)) {
		t.Fatal("non-array deref should not be skipped")
	}
}

func TestZeroLengthSliceToArrayConversionKeepsNilCheck(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

func convert(p *[]byte) {
	_ = [0]byte(*p)
}
`)

	ir := mustNamedFunction(t, m, "foo.convert").String()
	if !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("zero-length slice-to-array conversion should keep operand nil check:\n%s", ir)
	}
}

func TestIsInterfaceCompareDeref(t *testing.T) {
	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

func compareInterfacePtr(p *interface{}, q interface{}) bool {
	return *p == q
}

func derefOnly(p *interface{}) interface{} {
	return *p
}
`)

	if !isInterfaceCompareDeref(findUnOp(t, ssaPkg.Func("compareInterfacePtr"), token.MUL, false)) {
		t.Fatal("interface deref used by comparison should be detected")
	}
	derefOnly := findUnOp(t, ssaPkg.Func("derefOnly"), token.MUL, false)
	if isInterfaceCompareDeref(derefOnly) {
		t.Fatal("interface deref without comparison referrer should not be detected")
	}
	refs := derefOnly.Referrers()
	if refs == nil {
		t.Fatal("derefOnly has no referrer slice")
	}
	oldRefs := *refs
	*refs = nil
	defer func() { *refs = oldRefs }()
	if isInterfaceCompareDeref(derefOnly) {
		t.Fatal("interface deref without referrers should not be detected")
	}
}

func TestRangeArrayPointerCallEvaluatesWithoutNilCheck(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

var sink int

func nextArray() *[3]int {
	return nil
}

func rangeArrayCall() {
	for i := range *nextArray() {
		sink += i
	}
}
`)

	ir := mustNamedFunction(t, m, "foo.rangeArrayCall").String()
	if !strings.Contains(ir, "foo.nextArray") {
		t.Fatalf("range over call operand should still evaluate the call:\n%s", ir)
	}
	if strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("range over nil *array call should not nil-check the array pointer:\n%s", ir)
	}
}

func TestStaticArrayLenCapEvaluatesOperands(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

var calls int

func nextArray() *[3]int {
	calls++
	return nil
}

func lenPtr(p *[3]int) int {
	return len(p)
}

func lenDeref(p *[3]int) int {
	return len(*p)
}

func capPtr(p *[3]int) int {
	return cap(p)
}

func capDeref(p *[3]int) int {
	return cap(*p)
}

func lenCallPtr() int {
	return len(nextArray())
}

func lenCallDeref() int {
	return len(*nextArray())
}
`)

	for _, name := range []string{"lenPtr", "lenDeref", "capPtr", "capDeref", "lenCallPtr", "lenCallDeref"} {
		ir := mustNamedFunction(t, m, "foo."+name).String()
		if !strings.Contains(ir, "ret i64 3") {
			t.Fatalf("%s should return static array length/capacity 3:\n%s", name, ir)
		}
		if strings.Contains(ir, "AssertNilDeref") {
			t.Fatalf("%s should not nil-check array pointer length/capacity:\n%s", name, ir)
		}
		if strings.Contains(name, "Call") && !strings.Contains(ir, "foo.nextArray") {
			t.Fatalf("%s should still evaluate call operand:\n%s", name, ir)
		}
	}
}

func TestStaticArrayLenBuiltinArgCoversPointerForms(t *testing.T) {
	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

func lenSlice(s []int) int {
	return len(s)
}

func copyArray(p *[5]int) [5]int {
	return *p
}
`)

	builtin := findBuiltinCall(t, ssaPkg.Func("lenSlice"), "len")
	load := findUnOp(t, ssaPkg.Func("copyArray"), token.MUL, true)

	prog := newLLSSAProg(t)
	pkg := prog.NewPackage("foo", "foo")
	goPkg := types.NewPackage("foo", "foo")
	ctx := &context{
		prog:   prog,
		pkg:    pkg,
		goTyps: goPkg,
	}

	if _, ok := ctx.staticArrayLenBuiltinArg(nil, ssaPkg.Func("lenSlice").Params[0]); ok {
		t.Fatal("slice len argument should not use the static array length path")
	}

	params := types.NewTuple(types.NewVar(token.NoPos, goPkg, "p", load.X.Type()))
	results := types.NewTuple(types.NewVar(token.NoPos, goPkg, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)

	for _, tc := range []struct {
		name string
		arg  ssa.Value
	}{
		{name: "staticArrayLenPtr", arg: load.X},
		{name: "staticArrayLenDeref", arg: load},
	} {
		fn := pkg.NewFunc(tc.name, sig, llssa.InGo)
		b := fn.MakeBody(1)
		ret := ctx.callEx(b, llssa.Call, &ssa.CallCommon{
			Value: builtin,
			Args:  []ssa.Value{tc.arg},
		}, nil)
		if ret.IsNil() {
			t.Fatalf("%s did not return a length expression", tc.name)
		}
		b.Return(ret)

		ir := mustNamedFunction(t, pkg.Module(), tc.name).String()
		if !strings.Contains(ir, "ret i64 5") {
			t.Fatalf("%s should return static array length 5:\n%s", tc.name, ir)
		}
	}
}

func findUnOp(t *testing.T, fn *ssa.Function, op token.Token, wantArray bool) *ssa.UnOp {
	t.Helper()
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			unop, ok := instr.(*ssa.UnOp)
			if !ok || unop.Op != op {
				continue
			}
			_, isArray := unop.Type().Underlying().(*types.Array)
			if isArray == wantArray {
				return unop
			}
		}
	}
	t.Fatalf("missing %s unop in %s", op, fn.Name())
	return nil
}

func findBuiltinCall(t *testing.T, fn *ssa.Function, name string) ssa.Value {
	t.Helper()
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			builtin, ok := call.Common().Value.(*ssa.Builtin)
			if ok && builtin.Name() == name {
				return builtin
			}
		}
	}
	t.Fatalf("missing builtin %s call in %s", name, fn.Name())
	return nil
}
