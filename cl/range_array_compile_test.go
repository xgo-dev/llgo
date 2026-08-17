//go:build !llgo
// +build !llgo

package cl

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
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

func TestRangeArrayPointerEffectfulOperandKeepsNilCheck(t *testing.T) {
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

func rangeArrayReceive(ch <-chan *[3]int) {
	for i := range *<-ch {
		sink += i
	}
}
`)

	for _, name := range []string{"rangeArrayCall", "rangeArrayReceive"} {
		ir := mustNamedFunction(t, m, "foo."+name).String()
		if got := strings.Count(ir, "AssertNilDeref"); got != 1 {
			t.Fatalf("%s nil-check count = %d, want 1:\n%s", name, got, ir)
		}
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

func capCallDeref() int {
	return cap(*nextArray())
}

func lenReceiveDeref(ch <-chan *[3]int) int {
	return len(*<-ch)
}

type arrayPointer *[3]int

func lenConvertedCallDeref() int {
	return len(*arrayPointer(nextArray()))
}

func lenAssignedCallDeref() int {
	p := nextArray()
	return len(*p)
}

type holder struct { p *[3]int }

func nextHolder() holder { return holder{} }

func lenFieldCallDeref() int {
	return len(*nextHolder().p)
}
`)

	for _, name := range []string{"lenPtr", "lenDeref", "capPtr", "capDeref", "lenCallPtr", "lenCallDeref", "capCallDeref", "lenReceiveDeref", "lenConvertedCallDeref", "lenAssignedCallDeref", "lenFieldCallDeref"} {
		ir := mustNamedFunction(t, m, "foo."+name).String()
		if !strings.Contains(ir, "ret i64 3") {
			t.Fatalf("%s should return static array length/capacity 3:\n%s", name, ir)
		}
		wantNilCheck := name == "lenCallDeref" || name == "capCallDeref" || name == "lenReceiveDeref" || name == "lenConvertedCallDeref" || name == "lenFieldCallDeref"
		if got := strings.Contains(ir, "AssertNilDeref"); got != wantNilCheck {
			t.Fatalf("%s nil check = %v, want %v:\n%s", name, got, wantNilCheck, ir)
		}
		wantCall := ""
		if strings.Contains(name, "Call") {
			wantCall = "foo.nextArray"
		}
		if name == "lenFieldCallDeref" {
			wantCall = "foo.nextHolder"
		}
		if wantCall != "" && !strings.Contains(ir, wantCall) {
			t.Fatalf("%s should still evaluate %s:\n%s", name, wantCall, ir)
		}
	}
}

func TestIsEffectfulArrayPointerDerefRejectsOtherShapes(t *testing.T) {
	if isEffectfulArrayPointerDeref(nil) {
		t.Fatal("nil instruction should not be detected")
	}
	if isEffectfulArrayPointerDeref(&ssa.UnOp{Op: token.SUB}) {
		t.Fatal("non-deref instruction should not be detected")
	}

	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

func nextArray() *[3]int { return nil }

func copyCall() [3]int { return *nextArray() }

func use([3]int) {}

func copyParam(p *[3]int) [3]int { return *p }

func copyAssignedCall() [3]int {
	p := nextArray()
	return *p
}

func singleRef() { use(*nextArray()) }

func multiRef() {
	x := *nextArray()
	use(x)
	use(x)
}
`)

	if isEffectfulArrayPointerDeref(findUnOp(t, ssaPkg.Func("copyParam"), token.MUL, true)) {
		t.Fatal("deref without a function call or channel receive should not be detected")
	}
	assignedCall := findUnOp(t, ssaPkg.Func("copyAssignedCall"), token.MUL, true)
	if _, ok := assignedCall.X.(*ssa.Call); !ok {
		t.Fatalf("assigned call deref operand = %T, want *ssa.Call", assignedCall.X)
	}
	if !assignedCall.X.Pos().IsValid() || assignedCall.X.Pos() >= assignedCall.Pos() {
		t.Fatalf("assigned call position %v should precede deref position %v", assignedCall.X.Pos(), assignedCall.Pos())
	}
	if arrayPointerOperandHasEffectAfter(assignedCall.X, assignedCall.Pos(), nil) {
		t.Fatal("call assigned before deref expression should not be detected")
	}
	if isEffectfulArrayPointerDeref(findUnOp(t, ssaPkg.Func("copyCall"), token.MUL, true)) {
		t.Fatal("deref referenced by return should not be detected")
	}
	if isEffectfulArrayPointerDeref(findUnOp(t, ssaPkg.Func("singleRef"), token.MUL, true)) {
		t.Fatal("deref passed to a non-builtin call should not be detected")
	}
	if isEffectfulArrayPointerDeref(findUnOp(t, ssaPkg.Func("multiRef"), token.MUL, true)) {
		t.Fatal("deref with multiple referrers should not be detected")
	}
}

func TestArrayPointerOperandHasEffect(t *testing.T) {
	call := &ssa.Call{}
	cycle := &ssa.Phi{}
	cycle.Edges = []ssa.Value{cycle}
	for _, tc := range []struct {
		name  string
		value ssa.Value
		want  bool
	}{
		{name: "call", value: call, want: true},
		{name: "receive", value: &ssa.UnOp{Op: token.ARROW}, want: true},
		{name: "other unary operation", value: &ssa.UnOp{Op: token.SUB}},
		{name: "nested instruction", value: &ssa.ChangeType{X: call}, want: true},
		{name: "cycle", value: cycle},
		{name: "parameter", value: &ssa.Parameter{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := arrayPointerOperandHasEffectAfter(tc.value, token.NoPos, nil); got != tc.want {
				t.Fatalf("arrayPointerOperandHasEffectAfter(%T) = %v, want %v", tc.value, got, tc.want)
			}
		})
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

func TestEffectfulArrayDerefWithBuiltinRefKeepsNilCheck(t *testing.T) {
	ssaPkg, _, files := buildGoSSAPkg(t, `
package foo

func nextArray() *[3]int { return nil }

func discard() { _ = *nextArray() }

func lenSlice(s []int) int { return len(s) }
`)

	load := findUnOp(t, ssaPkg.Func("discard"), token.MUL, true)
	refs := load.Referrers()
	if refs == nil {
		t.Fatal("array deref has no referrer list")
	}
	oldRefs := *refs
	// Current x/tools folds len(*nextArray()) to a static constant and leaves
	// the deref unused. Model the builtin ref shape retained by other supported
	// SSA versions so the compiler compatibility path remains covered.
	builtin := findBuiltinCall(t, ssaPkg.Func("lenSlice"), "len")
	*refs = []ssa.Instruction{&ssa.Call{Call: ssa.CallCommon{Value: builtin}}}
	defer func() { *refs = oldRefs }()

	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := mustNamedFunction(t, pkg.Module(), "foo.discard").String()
	if !strings.Contains(ir, "foo.nextArray") || !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("builtin-referenced array deref should retain its call and nil check:\n%s", ir)
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
