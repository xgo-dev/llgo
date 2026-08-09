//go:build !llgo

package cabi

import (
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/debuginfo"
	"github.com/xgo-dev/llvm"
)

func TestReplaceAllocaInstrsUpdatesDebugDeclare(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("cabi-debug")
	defer mod.Dispose()

	di := debuginfo.New(mod, debuginfo.Config{Producer: "LLGo"})
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
	di.InsertDeclareAtEnd(home, variable, di.CreateExpression(nil), llvm.DebugLoc{Line: 1, Scope: subprogram}, block)
	zero := llvm.ConstInt(ctx.Int32Type(), 0, false)
	setup := builder.CreateGEP(int64Type, home, []llvm.Value{zero}, "setup")
	builder.CreateStore(param, home)
	loaded := builder.CreateLoad(int64Type, home, "loaded")
	builder.CreateStore(loaded, replacement)
	builder.CreateRetVoid()

	replaceAllocaInstrs(param, replacement)
	di.Finalize()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("rewritten module is invalid: %v\n%s", err, mod.String())
	}
	if setup.Operand(0) != home {
		t.Fatalf("setup operand was rewritten to the ABI home:\n%s", mod.String())
	}
	ir := mod.String()
	if !strings.Contains(ir, "#dbg_declare(ptr %replacement") {
		t.Fatalf("dbg.declare did not follow the ABI home:\n%s", ir)
	}
	if !strings.Contains(ir, "%loaded = load i64, ptr %replacement") {
		t.Fatalf("executable alloca use did not follow the ABI home:\n%s", ir)
	}
}

func TestPreserveDebugPointerHomesPolicy(t *testing.T) {
	tr := &Transformer{optimize: true}
	tests := []struct {
		name      string
		preserve  bool
		kind      AttrKind
		replaceOK bool
	}{
		{name: "optimized pointer", kind: AttrPointer, replaceOK: true},
		{name: "debug pointer", preserve: true, kind: AttrPointer},
		{name: "debug scalar", preserve: true, kind: AttrWidthType, replaceOK: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr.SetPreserveDebugPointerHomes(tc.preserve)
			if got := tr.shouldReplaceParameterHome(tc.kind); got != tc.replaceOK {
				t.Fatalf("shouldReplaceParameterHome(%v) = %v, want %v", tc.kind, got, tc.replaceOK)
			}
		})
	}
}
