package cl

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/goplus/gogen/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildConstIfaceSSA(t *testing.T, src string, debug bool) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "constiface.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	if debug {
		mode |= ssa.GlobalDebug
	}
	pkg := types.NewPackage("constiface", "constiface")
	imp := packages.NewImporter(fset)
	ssapkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, []*ast.File{file}, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssapkg
}

func TestFoldConstantMakeValueGuards(t *testing.T) {
	ctx, b := newNilCompileContext(t)
	if _, ok := ctx.foldConstantMakeValue(b, nil, nil); ok {
		t.Error("foldConstantMakeValue accepted nil inputs")
	}

	pkg := buildConstIfaceSSA(t, `package constiface
func box(x int) any { return x }
func no(x int) any { return x + 1 }
`, false)
	goFn := pkg.Func("box")
	call := &ssa.CallCommon{
		Value: goFn,
		Args:  []ssa.Value{ssa.NewConst(nil, types.Typ[types.Int])},
	}
	ctx, b = newNilCompileContext(t)
	ctx.goFn = goFn
	if _, ok := ctx.foldConstantMakeValue(b, pkg.Func("no"), call); ok {
		t.Error("foldConstantMakeValue accepted a non-box function")
	}
	if _, ok := ctx.foldConstantMakeValue(b, goFn, nil); ok {
		t.Error("foldConstantMakeValue accepted a nil call")
	}
	if _, ok := ctx.foldConstantMakeValue(b, goFn, &ssa.CallCommon{}); ok {
		t.Error("foldConstantMakeValue accepted a call without callee")
	}
}

func TestFoldConstantMakeValue(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface
func box(x int) any { return x }
func F() any { return box(42) }
`, false)
	var call *ssa.Call
	for _, instr := range pkg.Func("F").Blocks[0].Instrs {
		if candidate, ok := instr.(*ssa.Call); ok {
			call = candidate
			break
		}
	}
	if call == nil {
		t.Fatal("missing constructor call")
	}
	ctx, b := newNilCompileContext(t)
	value, ok := ctx.foldConstantMakeValue(b, pkg.Func("box"), &call.Call)
	if !ok || value.IsNil() {
		t.Fatal("constant interface constructor was not folded")
	}
	b.Return()
	b.EndBuild()
}

func TestFoldConstantMakeValueArgumentGuards(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface
func box(x, y int) any { return y }
`, false)
	ctx, b := newNilCompileContext(t)
	call := &ssa.CallCommon{
		Value: pkg.Func("box"),
		Args:  []ssa.Value{ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])},
	}
	if _, ok := ctx.foldConstantMakeValue(b, pkg.Func("box"), call); ok {
		t.Error("foldConstantMakeValue accepted a missing parameter argument")
	}
}

func TestAnalyzeTrivialIfaceBoxGuards(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface

type I interface{ M() }
type T int
func plain(x int) any { return x }
func changed(x T) any { return int(x) }
func iface(x I) any { return x }
func side(x int) any { println(x); return x }
func dynamic(x int) any { return x + 1 }
func direct(x int) int { return x }
func duplicate(x int) any { if false { return x }; return x }
func branched(x int) any { if x > 0 { return x }; return x }
func multi(x int) (any, int) { return x, 1 }
//go:noinline
func noinline(x int) any { return x }
func recovering(x int) any { defer func() { recover() }(); return x }
func (T) method(x int) any { return x }
`, false)
	checks := map[string]bool{
		"plain":      true,
		"changed":    true,
		"iface":      false,
		"side":       false,
		"dynamic":    false,
		"direct":     false,
		"duplicate":  false,
		"branched":   false,
		"multi":      false,
		"noinline":   false,
		"recovering": false,
		"(T).method": false,
	}
	for name, want := range checks {
		fn := pkg.Func(name)
		if name == "(T).method" {
			for _, member := range pkg.Members {
				if candidate, ok := member.(*ssa.Function); ok && candidate.Name() == "method" {
					fn = candidate
					break
				}
			}
		}
		_, _, got := analyzeTrivialIfaceBox(fn)
		if got != want {
			t.Errorf("analyzeTrivialIfaceBox(%s) = %v, want %v", name, got, want)
		}
	}
	var nilFn *ssa.Function
	if _, _, ok := analyzeTrivialIfaceBox(nilFn); ok {
		t.Error("nil function was accepted")
	}
}

func TestAnalyzeTrivialIfaceBoxIgnoresDebugRefs(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface
func box(x int) any { return x }
`, true)
	fn := pkg.Func("box")
	if _, _, ok := analyzeTrivialIfaceBox(fn); !ok {
		t.Fatal("DebugRef changed trivial interface-box classification")
	}
}

func TestAnalyzeTrivialIfaceBoxRejectsUnexpectedValue(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface
func box(x int) any { return x }
`, false)
	fn := pkg.Func("box")
	for _, instr := range fn.Blocks[0].Instrs {
		if mi, ok := instr.(*ssa.MakeInterface); ok {
			mi.X = &ssa.ChangeType{X: ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])}
			break
		}
	}
	if _, _, ok := analyzeTrivialIfaceBox(fn); ok {
		t.Fatal("non-parameter interface payload was accepted")
	}
}

func TestBlockHasExtraInstrs(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface
func box(x int) any { return x }
`, true)
	fn := pkg.Func("box")
	var ret *ssa.Return
	for _, instr := range fn.Blocks[0].Instrs {
		if r, ok := instr.(*ssa.Return); ok {
			ret = r
		}
	}
	if ret == nil {
		t.Fatal("box has no return")
	}
	chain := make(map[ssa.Instruction]bool)
	for _, instr := range fn.Blocks[0].Instrs {
		chain[instr] = true
	}
	if blockHasExtraInstrs(fn.Blocks[0], chain) {
		t.Error("complete instruction chain was classified as having extra instructions")
	}
}

func TestAnalyzeTrivialIfaceBoxRejectsDuplicateReturn(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface
func box(x int) any { return x }
`, false)
	fn := pkg.Func("box")
	for _, instr := range fn.Blocks[0].Instrs {
		if ret, ok := instr.(*ssa.Return); ok {
			fn.Blocks[0].Instrs = append(fn.Blocks[0].Instrs, ret)
			break
		}
	}
	if _, _, ok := analyzeTrivialIfaceBox(fn); ok {
		t.Fatal("duplicate return was accepted")
	}
}

func TestAnalyzeTrivialIfaceBoxSyntheticAndRecover(t *testing.T) {
	pkg := buildConstIfaceSSA(t, `package constiface
func box(x int) any { return x }
`, false)
	fn := pkg.Func("box")
	fn.Synthetic = "wrapper"
	if _, _, ok := analyzeTrivialIfaceBox(fn); ok {
		t.Error("synthetic function was accepted")
	}
	fn.Synthetic = ""
	fn.Recover = new(ssa.BasicBlock)
	if _, _, ok := analyzeTrivialIfaceBox(fn); ok {
		t.Error("recovering function was accepted")
	}
}
