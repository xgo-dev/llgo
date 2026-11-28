//go:build !llgo

package cabi

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/debuginfo"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llvm"
)

func TestParamHomeReusePreservesDebugStorage(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, optimized := range []bool{false, true} {
		name := "O0"
		if optimized {
			name = "O2"
		}
		t.Run(name, func(t *testing.T) {
			testParamHomeReuseWithDebug(t, optimized)
		})
	}
}

func testParamHomeReuseWithDebug(t *testing.T, optimized bool) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("cabi-debug")
	defer mod.Dispose()

	di := debuginfo.New(mod, debuginfo.Config{Producer: "LLGo", Optimized: optimized})
	cu := di.CompileUnit("cabi.go", "/src")
	file := di.File("/src/cabi.go")
	intType := di.CreateBasicType(llvm.DIBasicType{Name: "int", SizeInBits: 64, Encoding: 5})
	subroutine := di.CreateSubroutineType(llvm.DISubroutineType{File: file})
	subprogram := di.CreateFunction(cu, llvm.DIFunction{
		Name:         "cabi",
		LinkageName:  "cabi",
		File:         file,
		Line:         1,
		ScopeLine:    1,
		Type:         subroutine,
		IsDefinition: true,
	})
	variable := di.CreateAutoVariable(subprogram, llvm.DIAutoVariable{
		Name:           "value",
		File:           file,
		Line:           1,
		Type:           intType,
		AlwaysPreserve: true,
	})

	int64Type := ctx.Int64Type()
	fnType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{int64Type, llvm.PointerType(int64Type, 0)}, false)
	fn := llvm.AddFunction(mod, "cabi", fnType)
	fn.SetSubprogram(subprogram)
	param := fn.Param(0)
	param.SetName("param")
	replacement := fn.Param(1)
	replacement.SetName("replacement")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	block := llvm.AddBasicBlock(fn, "entry")
	builder.SetInsertPointAtEnd(block)
	home := builder.CreateAlloca(int64Type, "home")
	if optimized {
		di.InsertValueAtEnd(param, variable, di.CreateExpression(nil), llvm.DebugLoc{Line: 1, Scope: subprogram}, block)
	} else {
		di.InsertDeclareAtEnd(home, variable, di.CreateExpression(nil), llvm.DebugLoc{Line: 1, Scope: subprogram}, block)
	}
	builder.CreateStore(param, home)
	loaded := builder.CreateLoad(int64Type, home, "loaded")
	builder.CreateStore(loaded, replacement)
	builder.CreateRetVoid()

	prog := llvmgen.NewProgram(nil)
	defer prog.Dispose()
	prog.SetDebugInfoOptimized(optimized)
	transformer := &Transformer{prog: prog}
	transformer.reuseParamHome(param, replacement, block, 8, 8)
	di.Finalize()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("rewritten module is invalid: %v\n%s", err, mod.String())
	}
	ir := mod.String()
	if optimized {
		if !strings.Contains(ir, "#dbg_value(i64 %param") || loaded.Operand(0) != replacement {
			t.Fatalf("optimized DWARF prevented parameter-home reuse:\n%s", ir)
		}
		return
	}
	if !strings.Contains(ir, "#dbg_declare(ptr %home") {
		t.Fatalf("dbg.declare lost its authoritative home:\n%s", ir)
	}
	if !strings.Contains(ir, "%loaded = load i64, ptr %home") {
		t.Fatalf("executable uses no longer read the declared home:\n%s", ir)
	}
}
