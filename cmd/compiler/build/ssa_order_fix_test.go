package build

import (
	"github.com/xgo-dev/llgo/types"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
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

func TestFixSSAOrderReturnLoadWithDebugRefsIntegration(t *testing.T) {
	name := "ssa-order-debug"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(t.TempDir(), name)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = bin
	conf.LinkOptions.DWARF = DWARFPreserve
	if _, err := Do([]string{"./testdata/ssaorderdebug"}, conf); err != nil {
		t.Fatalf("build DebugRef return-order fixture: %v", err)
	}
	if got := runBinary(t, bin); got != "RETURN_ORDER_OK\n" {
		t.Fatalf("DebugRef return-order fixture output = %q", got)
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
	prog, ssaPackages := packages.Packages(loaded, mode)
	prog.Build()
	pkg := ssaPackages[0]
	checkReturnLoadAfterMutation(t, pkg.Func("ParseOID"), "unmarshalOIDText")
}

func indexOfInstr(instrs []ssa.Instruction, target ssa.Instruction) int {
	for i, ins := range instrs {
		if ins == target {
			return i
		}
	}
	return -1
}

func instrUsesValue(ins ssa.Instruction, v ssa.Value) bool {
	if ins == nil || v == nil {
		return false
	}
	for _, op := range ins.Operands(nil) {
		if op != nil && *op == v {
			return true
		}
	}
	return false
}

func callUsesValue(ci ssa.CallInstruction, v ssa.Value) bool {
	if ci == nil || v == nil {
		return false
	}
	c := ci.Common()
	if c == nil {
		return false
	}
	for _, op := range c.Operands(nil) {
		if op != nil && *op == v {
			return true
		}
	}
	return false
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
		&types.Config{Importer: packages.DefaultImporter()},
		fset,
		pkg,
		files,
		mode,
	)
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
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
