//go:build !llgo

package build

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

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
	fn := buildSSAOrderTestPackage(t, src)
	got := instrOrder(fn, "fc(", "<-", "fp(", "*t")
	if !inOrder(got, "fc(", "<-", "fp(") {
		t.Fatalf("single-case select receive assignment order = %v, want fc/receive before fp", got)
	}
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
	fn := buildSSAOrderTestPackage(t, src)
	got := instrOrder(fn, "fc(", "<-", "fn(")
	if !inOrder(got, "fc(", "<-", "fn(") {
		t.Fatalf("single-case select map receive assignment order = %v, want fc/receive before fn", got)
	}
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
	fn := buildSSAOrderTestPackage(t, src)
	got := instrOrder(fn, "fc(", "<-", "fp(", "*t")
	if !inOrder(got, "fc(", "<-", "fp(") {
		t.Fatalf("single-case select two-value receive assignment order = %v, want fc/receive before fp", got)
	}
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

func buildSSAOrderTestPackage(t *testing.T, src string) *ssa.Function {
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
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics,
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
