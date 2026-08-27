package build

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestFixSSAOrderSingleCaseSelectRecvAssign(t *testing.T) {
	const src = `package p
var c = make(chan int, 1)
var x int
func checkorder(o int) {}
func fc(c chan int, o int) chan int { checkorder(o); return c }
func fp(p *int, o int) *int { checkorder(o); return p }
func f() {
	c <- 1
	select {
	case *fp(&x, 100) = <-fc(c, 1):
	}
}`
	testSSAOrderModes(t, src, func(t *testing.T, fn *ssa.Function) {
		got := instrOrder(fn, "fc(", "<-", "fp(", "*t")
		if !inOrder(got, "fc(", "<-", "fp(") {
			t.Fatalf("single-case select receive assignment order = %v, want fc/receive before fp", got)
		}
	})
}

func TestFixSSAOrderPlainRecvAssignKeepsLeftToRight(t *testing.T) {
	const src = `package p
var c = make(chan int, 1)
var x int
func checkorder(o int) {}
func fc(c chan int, o int) chan int { checkorder(o); return c }
func fp(p *int, o int) *int { checkorder(o); return p }
func f() {
	c <- 1
	*fp(&x, 100) = <-fc(c, 1)
}`
	fn := buildSSAOrderTestPackage(t, src)
	got := instrOrder(fn, "fp(", "fc(", "<-")
	if !inOrder(got, "fp(", "fc(", "<-") {
		t.Fatalf("plain receive assignment order = %v, want fp before fc/receive", got)
	}
}

func TestFixSSAOrderSingleCaseSelectMapAssign(t *testing.T) {
	const src = `package p
var c = make(chan int, 1)
var m = make(map[int]int)
func checkorder(o int) {}
func fc(c chan int, o int) chan int { checkorder(o); return c }
func fn(n, o int) int { checkorder(o); return n }
func f() {
	c <- 1
	select {
	case m[fn(13, 100)] = <-fc(c, 1):
	}
}`
	testSSAOrderModes(t, src, func(t *testing.T, fn *ssa.Function) {
		got := instrOrder(fn, "fc(", "<-", "fn(")
		if !inOrder(got, "fc(", "<-", "fn(") {
			t.Fatalf("single-case select map receive assignment order = %v, want fc/receive before fn", got)
		}
	})
}

func TestFixSSAOrderSingleCaseSelectTwoValueRecv(t *testing.T) {
	const src = `package p
var c = make(chan int, 1)
var x int
var ok bool
func checkorder(o int) {}
func fc(c chan int, o int) chan int { checkorder(o); return c }
func fp(p *int, o int) *int { checkorder(o); return p }
func f() {
	c <- 1
	select {
	case *fp(&x, 100), ok = <-fc(c, 1):
	}
}`
	testSSAOrderModes(t, src, func(t *testing.T, fn *ssa.Function) {
		got := instrOrder(fn, "fc(", "<-", "fp(", "*t")
		if !inOrder(got, "fc(", "<-", "fp(") {
			t.Fatalf("single-case select two-value receive assignment order = %v, want fc/receive before fp", got)
		}
	})
}

func TestFixSSAOrderMultiCaseSelectKeepsLeftToRight(t *testing.T) {
	const src = `package p
var c = make(chan int, 1)
var x int
func checkorder(o int) {}
func fc(c chan int, o int) chan int { checkorder(o); return c }
func fp(p *int, o int) *int { checkorder(o); return p }
func f() {
	c <- 1
	select {
	case *fp(&x, 100) = <-fc(c, 1):
	case <-c:
	}
}`
	fn := buildSSAOrderTestPackage(t, src)
	got := instrOrder(fn, "fc(", "select", "fp(")
	if !inOrder(got, "fc(", "select", "fp(") {
		t.Fatalf("multi-case select receive assignment order = %v, want fp after select", got)
	}
}

func TestFixSSAOrderReturnLoadWithDebugRefs(t *testing.T) {
	const src = `package p
type value struct { n int }
func (v *value) mutate() bool { v.n = 1; return true }
func f() (value, bool) {
	var v value
	return v, v.mutate()
}`
	base := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	for _, test := range []struct {
		name string
		mode ssa.BuilderMode
	}{
		{name: "default", mode: base},
		{name: "global-debug", mode: base | ssa.GlobalDebug},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn := buildSSAOrderTestPackageMode(t, src, test.mode)
			checkReturnLoadAfterMutation(t, fn, "mutate")
		})
	}
}

func TestFixSSAOrderCryptoX509ParseOID(t *testing.T) {
	fset := token.NewFileSet()
	loaded, err := packages.Load(&packages.Config{
		Mode: packages.LoadSyntax,
		Fset: fset,
	}, "crypto/x509")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(loaded) != 0 || len(loaded) != 1 {
		t.Fatal("failed to load crypto/x509")
	}
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics | ssa.GlobalDebug
	prog, ssaPackages := ssautil.Packages(loaded, mode)
	prog.Build()
	pkg := ssaPackages[0]
	fixSSAOrder(pkg, loaded[0].Syntax)
	checkReturnLoadAfterMutation(t, pkg.Func("ParseOID"), "unmarshalOIDText")
}

func TestDebugRefMoveHelpers(t *testing.T) {
	movedValue := &ssa.BinOp{}
	instrs := []ssa.Instruction{
		movedValue,
		&ssa.DebugRef{X: movedValue},
		&ssa.Return{Results: []ssa.Value{movedValue}},
	}

	moved := movedValuesForIndices(instrs, map[int]struct{}{
		-1:          {},
		0:           {},
		2:           {},
		len(instrs): {},
	})
	if len(moved) != 1 {
		t.Fatalf("moved values = %d, want 1", len(moved))
	}
	if _, ok := moved[movedValue]; !ok {
		t.Fatal("moved values do not contain the selected SSA value")
	}

	move := map[int]struct{}{0: {}}
	includeDebugRefsForMovedValues(instrs, move, moved, -1, len(instrs)+1)
	if _, ok := move[1]; !ok {
		t.Fatal("DebugRef for the moved value was not included")
	}
}

func TestMoveInstrsAfter(t *testing.T) {
	const src = `package p
func f() {
	println(1)
	println(2)
	println(3)
}`
	fn := buildSSAOrderTestPackage(t, src)
	instrs := fn.Blocks[0].Instrs
	if len(instrs) < 4 {
		t.Fatalf("instructions = %d, want at least 4", len(instrs))
	}

	assertOrder := func(t *testing.T, got, want []ssa.Instruction) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("instruction count = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("instruction %d = %v, want %v", i, got[i], want[i])
			}
		}
	}

	t.Run("empty", func(t *testing.T) {
		assertOrder(t, moveInstrsAfter(instrs, nil, instrs[1]), instrs)
	})
	t.Run("nil-anchor", func(t *testing.T) {
		moving := map[ssa.Instruction]struct{}{instrs[0]: {}}
		assertOrder(t, moveInstrsAfter(instrs, moving, nil), instrs)
	})
	t.Run("missing-anchor", func(t *testing.T) {
		moving := map[ssa.Instruction]struct{}{instrs[0]: {}}
		assertOrder(t, moveInstrsAfter(instrs, moving, &ssa.Return{}), instrs)
	})
	t.Run("anchor-in-moving", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("moveInstrsAfter did not reject an anchor in the moving set")
			}
		}()
		moving := map[ssa.Instruction]struct{}{instrs[1]: {}}
		moveInstrsAfter(instrs, moving, instrs[1])
	})
	t.Run("stable", func(t *testing.T) {
		moving := map[ssa.Instruction]struct{}{
			instrs[0]: {},
			instrs[2]: {},
		}
		want := []ssa.Instruction{instrs[1], instrs[0], instrs[2]}
		want = append(want, instrs[3:]...)
		assertOrder(t, moveInstrsAfter(instrs, moving, instrs[1]), want)
	})
}

func checkReturnLoadAfterMutation(t *testing.T, fn *ssa.Function, mutation string) {
	t.Helper()
	var ret *ssa.Return
	var mutationCall ssa.CallInstruction
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if candidate, ok := instr.(*ssa.Return); ok {
				ret = candidate
			}
			if call, ok := instr.(ssa.CallInstruction); ok {
				callee := call.Common().StaticCallee()
				if callee != nil && callee.Name() == mutation {
					mutationCall = call
				}
			}
		}
	}
	if ret == nil || len(ret.Results) != 2 || mutationCall == nil {
		t.Fatalf("unexpected return instruction: %v", ret)
	}
	callInstr := mutationCall.(ssa.Instruction)
	callIdx := indexOfInstr(callInstr.Block().Instrs, callInstr)
	loadIdx := -1
	debugRefIdx := -1
	var resultLoad *ssa.UnOp
	for i, instr := range callInstr.Block().Instrs {
		if load, ok := instr.(*ssa.UnOp); ok && load.Op == token.MUL {
			if alloc, ok := load.X.(*ssa.Alloc); ok && callUsesValue(mutationCall, alloc) {
				loadIdx = i
				resultLoad = load
			}
		}
		if ref, ok := instr.(*ssa.DebugRef); ok && resultLoad != nil && instrUsesValue(ref, resultLoad) {
			debugRefIdx = i
		}
	}
	if callIdx < 0 || loadIdx <= callIdx {
		t.Fatalf("return load index = %d, mutation call index = %d; want load after call\n%s", loadIdx, callIdx, fn)
	}
	if debugRefIdx >= 0 && debugRefIdx <= loadIdx {
		t.Fatalf("DebugRef index = %d, load index = %d; want DebugRef after its load\n%s", debugRefIdx, loadIdx, fn)
	}
}

func buildSSAOrderTestPackage(t *testing.T, src string) *ssa.Function {
	return buildSSAOrderTestPackageMode(t, src, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
}

func testSSAOrderModes(t *testing.T, src string, check func(*testing.T, *ssa.Function)) {
	t.Helper()
	base := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	for _, test := range []struct {
		name string
		mode ssa.BuilderMode
	}{
		{name: "default", mode: base},
		{name: "global-debug", mode: base | ssa.GlobalDebug},
	} {
		t.Run(test.name, func(t *testing.T) {
			check(t, buildSSAOrderTestPackageMode(t, src, test.mode))
		})
	}
}

func buildSSAOrderTestPackageMode(t *testing.T, src string, mode ssa.BuilderMode) *ssa.Function {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	files := []*ast.File{file}
	pkg := types.NewPackage("p", "p")
	ssapkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		pkg,
		files,
		mode,
	)
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	fixSSAOrder(ssapkg, files)
	fn, ok := ssapkg.Members["f"].(*ssa.Function)
	if !ok {
		t.Fatalf("missing function f")
	}
	return fn
}

func instrOrder(fn *ssa.Function, needles ...string) []string {
	var ret []string
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			s := instr.String()
			for _, needle := range needles {
				if strings.Contains(s, needle) {
					ret = append(ret, s)
					break
				}
			}
		}
	}
	return ret
}

func inOrder(instrs []string, needles ...string) bool {
	pos := 0
	for _, instr := range instrs {
		if pos < len(needles) && strings.Contains(instr, needles[pos]) {
			pos++
		}
	}
	return pos == len(needles)
}
