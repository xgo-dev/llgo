package ccgobridge

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const slotFixture = `package libc
import "unsafe"
type stackSlot struct { p uintptr; sz uint64 }
type TLS struct { sp int; stack []stackSlot; checkSignals bool }
func (tls *TLS) Free(n int) { tls.sp--; if !tls.checkSignals { return }; tls.checkSignal() }
func (tls *TLS) checkSignal() {}
func (tls *TLS) Close() { defer func() { *tls = TLS{} }() }
func __ccgo_abi0_compare(tls *TLS, left, right uintptr, pc uintptr) int32 { return int32(pc + left - right) }
func __ccgo_abiInternal_compare(tls *TLS, dest, abi0CodePtr uintptr) {
 f := func(tls *TLS, left, right uintptr) int32 {
  return __ccgo_abi0_compare(tls, left, right, abi0CodePtr)
 }
 *(*[2]uintptr)(unsafe.Pointer(dest)) = *(*[2]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(&f))))
}
`

func slotParse(t *testing.T, source string) (*token.FileSet, *types.Package, *types.Info, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "slots.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	config := types.Config{Importer: importer.Default()}
	pkg, err := config.Check("example.com/libc", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return fset, pkg, info, []*ast.File{file}
}

func slotFormat(t *testing.T, fset *token.FileSet, file *ast.File) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestRewriteSlots(t *testing.T) {
	fset, pkg, info, files := slotParse(t, slotFixture)
	count, err := RewriteSlots(fset, pkg, info, files)
	if err != nil || count != 1 {
		t.Fatalf("RewriteSlots = %d, %v", count, err)
	}
	result := slotFormat(t, fset, files[0])
	for _, want := range []string{
		"__llgoCCGoRoots []__llgoCCGoRoot",
		"var _ [16 - unsafe.Sizeof(f)]byte",
		"tls.__llgoCCGoKeep(dest, unsafe.Sizeof(f), unsafe.Alignof(f), f)",
		"*(*func(tls *TLS, left, right uintptr) int32)(unsafe.Pointer(dest)) = f",
		"tls.sp--\n\ttls.__llgoCCGoRelease()",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("rewritten source lacks %q:\n%s", want, result)
		}
	}
	// The adapted source remains ordinary well-typed Go, not a compiler-layout
	// special case. Execution under both compilers is a separate integration gate.
	slotParse(t, result)
}

func TestRewriteSlotsRejectsBeforeMutation(t *testing.T) {
	for _, test := range []struct{ name, old, replacement string }{
		{"method callback", "func __ccgo_abiInternal_compare", "func (*TLS) __ccgo_abiInternal_compare"},
		{"extra parameter", "dest, abi0CodePtr uintptr)", "dest, abi0CodePtr uintptr, extra int)"},
		{"non-pointer owner", "func __ccgo_abiInternal_compare(tls *TLS", "func __ccgo_abiInternal_compare(tls uintptr"},
		{"unnamed owner", "func __ccgo_abiInternal_compare(tls *TLS", "func __ccgo_abiInternal_compare(tls *struct{}"},
		{"variable declaration", "f := func", "var f = func"},
		{"no closure", "f := func(tls *TLS, left, right uintptr) int32 {\n  return __ccgo_abi0_compare(tls, left, right, abi0CodePtr)\n }", "f := __ccgo_abi0_compare"},
		{"multiple assignments", "f := func", "f, _ := 0, func"},
		{"closure statements", "return __ccgo_abi0_compare", "println(abi0CodePtr); return __ccgo_abi0_compare"},
		{"variadic closure", "f := func(tls *TLS, left, right uintptr)", "f := func(tls *TLS, left, right uintptr, extra ...int)"},
		{"indirect target", "return __ccgo_abi0_compare(", "return (__ccgo_abi0_compare)("},
		{"function variable target", "func __ccgo_abi0_compare(", "var __ccgo_abi0_compare = func("},
		{"copy width", "*(*[2]uintptr)(unsafe.Pointer(dest)) = *(*[2]uintptr)", "*(*[3]uintptr)(unsafe.Pointer(dest)) = *(*[3]uintptr)"},
		{"unexpected statement", "f := func", "println(dest); f := func"},
		{"wrong capture", "right, abi0CodePtr)", "right, dest)"},
		{"wrong argument order", "compare(tls, left, right, abi0CodePtr)", "compare(tls, right, left, abi0CodePtr)"},
		{"computed capture", "right, abi0CodePtr)", "right, abi0CodePtr+0)"},
		{"owner depth type", "sp int;", "sp uint;"},
		{"missing stack", "stack []stackSlot", "frames []stackSlot"},
		{"stack element type", "stack []stackSlot", "stack []int"},
		{"slot address type", "p uintptr", "p uint64"},
		{"slot size type", "sz uint64", "sz string"},
		{"free parameter", "Free(n int)", "Free(n uint)"},
		{"missing free", "Free(n int)", "OtherFree(n int)"},
		{"free order", "tls.sp--;", "n++; tls.sp--;"},
		{"free direction", "tls.sp--;", "tls.sp++;"},
		{"free extra mutation", "tls.checkSignal() }", "tls.checkSignal(); tls.sp-- }"},
		{"close lifetime", "defer func() { *tls = TLS{} }()", "println(tls.sp)"},
		{"close parameter", "Close()", "Close(n int)"},
		{"helper type conflict", "type TLS struct", "type __llgoCCGoRoot struct{}; type TLS struct"},
		{"helper method conflict", "type TLS struct", "func (*TLS) __llgoCCGoKeep() {}; type TLS struct"},
		{"field conflict", "sp int;", "sp int; __llgoCCGoRoots []int;"},
		{"helper field conflict", "sp int;", "sp int; __llgoCCGoKeep int;"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(slotFixture, test.old, test.replacement, 1)
			if source == slotFixture {
				t.Fatal("mutation did not match source")
			}
			fset, pkg, info, files := slotParse(t, source)
			before := slotFormat(t, fset, files[0])
			if _, err := RewriteSlots(fset, pkg, info, files); err == nil {
				t.Fatal("accepted an unverified slot/lifetime shape")
			}
			if after := slotFormat(t, fset, files[0]); after != before {
				t.Fatal("failed validation partially modified the package")
			}
		})
	}
}

func TestRewriteVoidSlot(t *testing.T) {
	source := strings.Replace(slotFixture,
		"int32 { return int32(pc + left - right) }", "{}", 1)
	source = strings.Replace(source, "right uintptr) int32 {", "right uintptr) {", 1)
	source = strings.Replace(source, "return __ccgo_abi0_compare", "__ccgo_abi0_compare", 1)
	fset, pkg, info, files := slotParse(t, source)
	count, err := RewriteSlots(fset, pkg, info, files)
	if err != nil || count != 1 {
		t.Fatalf("void slot = %d, %v", count, err)
	}
	slotParse(t, slotFormat(t, fset, files[0]))
}

func TestRewriteManySlotsValidatesWholeBatch(t *testing.T) {
	prefix, callback, ok := strings.Cut(slotFixture, "func __ccgo_abi0_compare")
	if !ok {
		t.Fatal("fixture callback is missing")
	}
	var source strings.Builder
	source.WriteString(prefix)
	for i := 0; i < 37; i++ {
		source.WriteString(strings.ReplaceAll("func __ccgo_abi0_compare"+callback, "compare", fmt.Sprintf("compare%d", i)))
	}
	fset, pkg, info, files := slotParse(t, source.String())
	count, err := RewriteSlots(fset, pkg, info, files)
	if err != nil || count != 37 {
		t.Fatalf("37-shape batch = %d, %v", count, err)
	}
	slotParse(t, slotFormat(t, fset, files[0]))

	last := strings.LastIndex(source.String(), "f := func")
	invalid := source.String()[:last] + strings.Replace(source.String()[last:], "f := func", "println(dest); f := func", 1)
	fset, pkg, info, files = slotParse(t, invalid)
	before := slotFormat(t, fset, files[0])
	if _, err := RewriteSlots(fset, pkg, info, files); err == nil {
		t.Fatal("accepted an invalid member of the batch")
	}
	if slotFormat(t, fset, files[0]) != before {
		t.Fatal("batch failure modified other callbacks")
	}
}

func TestRewriteSlotsDoesNotGuessOrdinaryUnsafe(t *testing.T) {
	source := strings.ReplaceAll(slotFixture, "__ccgo_abiInternal_compare", "ordinaryFunction")
	fset, pkg, info, files := slotParse(t, source)
	before := slotFormat(t, fset, files[0])
	count, err := RewriteSlots(fset, pkg, info, files)
	if err != nil || count != 0 || slotFormat(t, fset, files[0]) != before {
		t.Fatalf("ordinary unsafe code was changed: %d, %v", count, err)
	}
}

func TestRewrittenSlotLifetimes(t *testing.T) {
	fset, pkg, info, files := slotParse(t, slotFixture)
	if _, err := RewriteSlots(fset, pkg, info, files); err != nil {
		t.Fatal(err)
	}
	source := slotFormat(t, fset, files[0])
	source = strings.Replace(source, "package libc", "package main", 1)
	source = strings.Replace(source, `import "unsafe"`, `import ("runtime"; "unsafe")`, 1)
	source += slotRuntimeFixture
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", path)
	command.WaitDelay = 5 * time.Second
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("adapted callback slot execution: %v\n%s", err, output)
	}
}

const slotRuntimeFixture = `
// Keep the raw storage itself reachable independently of its integer address.
var storage []*[2]uintptr

func frame() stackSlot {
	p := new([2]uintptr)
	storage = append(storage, p)
	return stackSlot{uintptr(unsafe.Pointer(p)), 16}
}

func check(condition bool) {
	if !condition { panic("callback slot lifetime check failed") }
}

func rejects(fn func()) {
	defer func() { if recover() == nil { panic("invalid slot was accepted") } }()
	fn()
}

func load(p uintptr) func(*TLS, uintptr, uintptr) int32 {
	return *(*func(*TLS, uintptr, uintptr) int32)(unsafe.Pointer(p))
}

func main() {
	first, second := frame(), frame()
	tls := &TLS{sp: 1, stack: []stackSlot{first, second}}
	__ccgo_abiInternal_compare(tls, first.p, 40)
	check(len(tls.__llgoCCGoRoots) == 1)
	runtime.GC()
	check(load(first.p)(tls, 3, 1) == 42)

	// Replacing a callback in one slot must not retain every previous closure.
	__ccgo_abiInternal_compare(tls, first.p, 60)
	check(len(tls.__llgoCCGoRoots) == 1)
	check(load(first.p)(tls, 3, 1) == 62)

	tls.sp = 2
	__ccgo_abiInternal_compare(tls, second.p, 100)
	check(len(tls.__llgoCCGoRoots) == 2)
	retained := load(second.p)
	tls.Free(0)
	check(tls.sp == 1 && len(tls.__llgoCCGoRoots) == 1)
	check(tls.__llgoCCGoRoots[:cap(tls.__llgoCCGoRoots)][1].fn == nil)
	runtime.GC()
	check(load(first.p)(tls, 3, 1) == 62)
	check(retained(tls, 3, 1) == 102)

	// Reuse has a new environment, while the previously imported value lives.
	tls.sp = 2
	__ccgo_abiInternal_compare(tls, second.p, 200)
	check(load(second.p)(tls, 3, 1) == 202)
	tls.Free(16)
	tls.Free(16)
	check(len(tls.__llgoCCGoRoots) == 0)
	rejects(func() { __ccgo_abiInternal_compare(tls, first.p, 1) })

	tls.sp = 1
	rejects(func() { __ccgo_abiInternal_compare(tls, first.p+16, 1) })
	rejects(func() { __ccgo_abiInternal_compare(tls, first.p+1, 1) })
	rejects(func() { tls.__llgoCCGoKeep(first.p, ^uintptr(0), 8, retained) })
	check(len(tls.__llgoCCGoRoots) == 0)
	__ccgo_abiInternal_compare(tls, first.p, 300)
	tls.Close()
	check(tls.sp == 0 && tls.stack == nil && tls.__llgoCCGoRoots == nil)
	runtime.GC()
	check(retained(tls, 3, 1) == 102)
}
`
