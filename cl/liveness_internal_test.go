//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildSSAPackageWithPath(t *testing.T, pkgPath, pkgName, src string) *ssa.Package {
	t.Helper()
	ssapkg, _ := buildSSAPackageWithPathAndFiles(t, pkgPath, pkgName, src)
	return ssapkg
}

func buildSSAPackageWithPathAndFiles(t *testing.T, pkgPath, pkgName, src string) (*ssa.Package, []*ast.File) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage(pkgPath, pkgName)
	imp := packages.NewImporter(fset)
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	ssapkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssapkg, files
}

func TestConservativeGCPointerTypeAnalysis(t *testing.T) {
	if hasConservativeGCPointers(nil, map[types.Type]bool{}) {
		t.Fatal("nil type should not report conservative pointers")
	}
	if hasConservativeGCPointers(types.Typ[types.Int], map[types.Type]bool{}) {
		t.Fatal("int should not report conservative pointers")
	}
	if hasConservativeGCPointers(types.Typ[types.String], map[types.Type]bool{types.Typ[types.String]: true}) {
		t.Fatal("seen type should short-circuit")
	}
	for _, typ := range []types.Type{
		types.Typ[types.String],
		types.Typ[types.UnsafePointer],
		types.NewPointer(types.Typ[types.Int]),
		types.NewSlice(types.Typ[types.Int]),
		types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
		types.NewInterfaceType(nil, nil),
		types.NewArray(types.NewPointer(types.Typ[types.Int]), 2),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "p", types.NewPointer(types.Typ[types.Int]), false)}, nil),
	} {
		if !hasConservativeGCPointers(typ, map[types.Type]bool{}) {
			t.Fatalf("%v should report conservative pointers", typ)
		}
	}
	if hasConservativeGCPointers(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "i", types.Typ[types.Int], false),
	}, nil), map[types.Type]bool{}) {
		t.Fatal("struct without pointer fields should not report conservative pointers")
	}
}

func TestShouldClearAlloc(t *testing.T) {
	ssapkg := buildSSAPackageWithPath(t, "example.com/live", "live", `package live

type Box struct{ p *int }

func allocs(p *int) {
	var box Box
	var i int
	_ = &box
	_ = &i
}
`)
	fn := ssapkg.Func("allocs")
	ctx := &context{}
	if ctx.shouldClearAlloc(nil) {
		t.Fatal("nil alloc should not be cleared")
	}

	var boxAlloc, intAlloc *ssa.Alloc
	for _, local := range fn.Locals {
		ptr := local.Type().Underlying().(*types.Pointer)
		if _, ok := ptr.Elem().Underlying().(*types.Struct); ok {
			boxAlloc = local
		}
		if ptr.Elem() == types.Typ[types.Int] {
			intAlloc = local
		}
	}
	if boxAlloc == nil || intAlloc == nil {
		t.Fatalf("missing expected locals: %v", fn.Locals)
	}
	if !ctx.shouldClearAlloc(boxAlloc) {
		t.Fatal("struct containing a pointer should be cleared")
	}
	if ctx.shouldClearAlloc(intAlloc) {
		t.Fatal("int alloc should not be cleared")
	}

	boxAlloc.Comment = "varargs"
	if ctx.shouldClearAlloc(boxAlloc) {
		t.Fatal("varargs alloc should not be cleared")
	}
	boxAlloc.Comment = "makeslice"
	if ctx.shouldClearAlloc(boxAlloc) {
		t.Fatal("synthetic makeslice alloc should not be cleared")
	}
}

func TestRuntimeSetFinalizerDetection(t *testing.T) {
	ssapkg := buildSSAPackageWithPath(t, "github.com/goplus/llgo/runtime/livetest", "livetest", `package livetest

import rt "github.com/goplus/llgo/runtime/internal/lib/runtime"

func direct(p *int) {
	rt.SetFinalizer(p, func(*int) {})
}

func deferred(p *int) {
	defer rt.SetFinalizer(p, nil)
}

func goroutine(p *int) {
	go rt.SetFinalizer(p, nil)
}

func nested(p *int) {
	func() {
		rt.SetFinalizer(p, nil)
	}()
}

func none(p *int) {}
`)
	ctx := &context{}
	if ctx.enableConservativeLivenessClears(nil) {
		t.Fatal("nil function should not enable conservative clears")
	}
	for _, name := range []string{"direct", "deferred", "goroutine", "nested"} {
		if !ctx.functionUsesRuntimeSetFinalizer(ssapkg.Func(name), map[*ssa.Function]bool{}) {
			t.Fatalf("%s should be detected as SetFinalizer user", name)
		}
	}
	if ctx.functionUsesRuntimeSetFinalizer(nil, map[*ssa.Function]bool{}) {
		t.Fatal("nil function should not use SetFinalizer")
	}
	direct := ssapkg.Func("direct")
	if ctx.functionUsesRuntimeSetFinalizer(direct, map[*ssa.Function]bool{direct: true}) {
		t.Fatal("seen function should short-circuit")
	}
	if ctx.functionUsesRuntimeSetFinalizer(ssapkg.Func("none"), map[*ssa.Function]bool{}) {
		t.Fatal("none should not use SetFinalizer")
	}
	if ctx.packageUsesRuntimeSetFinalizer(&ssa.Package{Members: map[string]ssa.Member{"none": ssapkg.Func("none")}}) {
		t.Fatal("package without SetFinalizer should not report use")
	}
	if !ctx.packageUsesRuntimeSetFinalizer(ssapkg) {
		t.Fatal("package should report SetFinalizer use")
	}
	if ctx.enableConservativeLivenessClears(direct) {
		t.Fatal("non command-line-arguments package should not enable conservative clears")
	}
	ssapkg.Pkg = types.NewPackage("command-line-arguments", "main")
	if !ctx.enableConservativeLivenessClears(direct) {
		t.Fatal("command-line-arguments package with SetFinalizer should enable conservative clears")
	}
}

func TestRuntimeSetFinalizerLateValueSkips(t *testing.T) {
	ssapkg := buildSSAPackageWithPath(t, "github.com/goplus/llgo/runtime/livetest", "livetest", `package livetest

import rt "github.com/goplus/llgo/runtime/internal/lib/runtime"

func direct(p *int) {
	rt.SetFinalizer(p, func(*int) {})
}
`)
	ctx := &context{}
	fn := ssapkg.Func("direct")
	var makeIface *ssa.MakeInterface
	var deref *ssa.UnOp
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch instr := instr.(type) {
			case *ssa.MakeInterface:
				makeIface = instr
			case *ssa.UnOp:
				if instr.Op == token.MUL {
					deref = instr
				}
			}
		}
	}
	if makeIface == nil {
		t.Fatal("missing MakeInterface for SetFinalizer argument")
	}
	if ctx.isRuntimeSetFinalizerCall(nil) {
		t.Fatal("nil call should not be SetFinalizer")
	}
	if !ctx.shouldSkipLateSetFinalizerValue(makeIface) {
		t.Fatal("SetFinalizer-only MakeInterface should be skipped")
	}
	if deref != nil && !ctx.shouldSkipLateSetFinalizerValue(deref) {
		t.Fatal("SetFinalizer-only deref should be skipped")
	}
	if ctx.shouldSkipLateSetFinalizerValue(&ssa.Return{}) {
		t.Fatal("unrelated instruction should not be skipped")
	}
	if ctx.shouldSkipLateSetFinalizerValue(&ssa.UnOp{Op: token.SUB}) {
		t.Fatal("non-deref unary op should not be skipped")
	}
}

func TestConservativeLivenessPlanCollectors(t *testing.T) {
	ssapkg := buildSSAPackageWithPath(t, "example.com/live", "live", `package live

type Box struct{ p *int }

var Sink any

func linear(p *int) {
	var box Box
	box.p = p
	Sink = box.p
	Sink = 1
}

func branch(p *int, cond bool) {
	var box Box
	box.p = p
	if cond {
		Sink = box.p
	} else {
		Sink = 0
	}
	Sink = 1
}

func paramUse(p *int) {
	Sink = p
	Sink = 1
}

func takes(*int) {}

func callWithPointer(p *int) {
	takes(p)
	Sink = 1
}

func callWithInt(i int) {
	Sink = i
}
`)
	ctx := &context{}
	linear := ssapkg.Func("linear")
	stackPlans := ctx.collectStackClearPlans(linear)
	if len(stackPlans) == 0 {
		t.Fatal("linear should produce stack clear plans")
	}
	for instr := range stackPlans {
		if isTerminatingInstruction(instr) {
			t.Fatalf("stack clear should not be scheduled after terminator %T", instr)
		}
	}

	entryPlans := ctx.collectEntryClearPlans(ssapkg.Func("branch"))
	if len(entryPlans) == 0 {
		t.Fatal("branch should produce entry clear plans for dead successor")
	}

	paramFn := ssapkg.Func("paramUse")
	if len(ctx.collectParamClobberPlans(paramFn)) == 0 {
		t.Fatal("paramUse should produce param clobber plans")
	}
	if len(ctx.collectParamScanPlans(paramFn)) == 0 {
		t.Fatal("paramUse should produce param scan plans")
	}
	if len(ctx.collectCallClobberPlans(ssapkg.Func("callWithPointer"))) == 0 {
		t.Fatal("pointer call should clobber pointer regs")
	}
	if len(ctx.collectCallClobberPlans(ssapkg.Func("callWithInt"))) != 0 {
		t.Fatal("int-only call should not clobber pointer regs")
	}
}

func TestConservativeLivenessGraphHelpers(t *testing.T) {
	ssapkg := buildSSAPackageWithPath(t, "example.com/live", "live", `package live

var Sink any

func flow(p *int, cond bool) {
	if cond {
		Sink = p
	} else {
		Sink = 0
	}
}

func target(*int) {}

func withCall(p *int) {
	target(p)
}
`)
	fn := ssapkg.Func("flow")
	if blockCanReach(nil, fn.Blocks[0], map[*ssa.BasicBlock]bool{}) {
		t.Fatal("nil block should not reach anything")
	}
	if !blockCanReach(fn.Blocks[0], fn.Blocks[0], map[*ssa.BasicBlock]bool{}) {
		t.Fatal("block should reach itself")
	}
	if instructionUsesValue(nil, fn.Params[0]) {
		t.Fatal("nil instruction should not use values")
	}
	if instructionUsesValue(fn.Blocks[0].Instrs[0], nil) {
		t.Fatal("nil value should not be used")
	}
	if isCallLikeInstruction(fn.Blocks[0].Instrs[0]) {
		t.Fatal("if instruction should not be call-like")
	}
	if !isTerminatingInstruction(fn.Blocks[0].Instrs[len(fn.Blocks[0].Instrs)-1]) {
		t.Fatal("entry block should end with a terminator")
	}

	ctx := &context{}
	if blk := refBlock(nil); blk != nil {
		t.Fatalf("refBlock(nil) = %v", blk)
	}
	blocks := make(map[*ssa.BasicBlock]bool)
	if !ctx.collectValueUseBlocks(nil, blocks, map[ssa.Value]bool{}, false) {
		t.Fatal("nil collectValueUseBlocks should succeed")
	}
	if !ctx.collectValueUseBlocks(fn.Params[0], blocks, map[ssa.Value]bool{fn.Params[0]: true}, false) {
		t.Fatal("seen collectValueUseBlocks should succeed")
	}
	if !ctx.collectValueUseBlocks(fn.Params[0], blocks, map[ssa.Value]bool{}, false) {
		t.Fatal("collectValueUseBlocks failed")
	}
	if len(blocks) == 0 {
		t.Fatal("expected use blocks for parameter")
	}
	if blk, ok := ctx.valueLastUseBlock(fn.Params[0]); !ok || blk == nil {
		t.Fatalf("valueLastUseBlock = %v, %v", blk, ok)
	}
	if blk, ok := ctx.valueLastUseBlock(nil); !ok || blk != nil {
		t.Fatalf("valueLastUseBlock(nil) = %v, %v", blk, ok)
	}
	if last, ok := ctx.lastUseInBlock(nil, fn.Blocks[0], map[ssa.Instruction]int{}, map[ssa.Value]bool{}); !ok || last != nil {
		t.Fatalf("lastUseInBlock(nil) = %v, %v", last, ok)
	}

	var callLike int
	for _, block := range ssapkg.Func("withCall").Blocks {
		for _, instr := range block.Instrs {
			if isCallLikeInstruction(instr) {
				callLike++
			}
		}
	}
	if callLike == 0 {
		t.Fatal("flow should include at least one call-like instruction")
	}
}

func TestCompileWithoutConservativeLivenessClears(t *testing.T) {
	ssapkg, files := buildSSAPackageWithPathAndFiles(t, "command-line-arguments", "main", `package main

func main() {
	x := 1
	_ = &x
}
`)

	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pkg.String(), "llgo_clear_stack_ptr") {
		t.Fatalf("package without SetFinalizer should not emit liveness clear helpers:\n%s", pkg.String())
	}
}

func TestCompileConservativeLivenessClears(t *testing.T) {
	ssapkg, files := buildSSAPackageWithPathAndFiles(t, "github.com/goplus/llgo/runtime/livetest", "main", `package main

import rt "github.com/goplus/llgo/runtime/internal/lib/runtime"

type Box struct{ p *int }

var Sink any

func main() {
	x := 1
	var box Box
	box.p = &x
	Sink = box.p
	rt.SetFinalizer(&box, func(*Box) {})
	Sink = 1
}
`)
	ssapkg.Pkg = types.NewPackage("command-line-arguments", "main")

	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.String()
	for _, want := range []string{"llgo_clear_stack_ptr", "llgo_clobber_pointer_regs"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("compiled liveness module missing %s:\n%s", want, ir)
		}
	}
	if !pkg.NeedRuntime {
		t.Fatal("liveness clear helpers should mark runtime as needed")
	}
}
