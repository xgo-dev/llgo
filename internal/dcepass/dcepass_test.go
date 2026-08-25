package dcepass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qtest "github.com/qiniu/x/test"
	"github.com/xgo-dev/llvm"
)

const (
	taskTypeName    = "_llgo_main.Task"
	ptrTaskTypeName = "*_llgo_main.Task"
)

func TestEmitStrongTypeOverrides(t *testing.T) {
	tests := []struct {
		name      string
		liveSlots map[string][]int
	}{
		{
			name: "method_slots",
			liveSlots: map[string][]int{
				taskTypeName:    {1}, // Run
				ptrTaskTypeName: {1}, // Run
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcCtx := llvm.NewContext()
			defer srcCtx.Dispose()
			dstCtx := llvm.NewContext()
			defer dstCtx.Dispose()
			dir := filepath.Join("testdata", tt.name)
			src := parseModule(t, &srcCtx, filepath.Join(dir, "in.ll"))
			defer src.Dispose()
			dst := dstCtx.NewModule("dst")
			defer dst.Dispose()

			EmitStrongTypeOverrides(dst, []llvm.Module{src}, tt.liveSlots, true)
			for global := dst.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
				if init := global.Initializer(); !init.IsNil() && global.GlobalValueType().C != init.Type().C {
					t.Errorf("global %s type %s does not match initializer type %s", global.Name(), global.GlobalValueType(), init.Type())
				}
			}
			if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("cross-context override is invalid: %v\n%s", err, dst.String())
			}
			want, err := os.ReadFile(filepath.Join(dir, "expect.ll"))
			if err != nil {
				t.Fatal(err)
			}
			qtest.Diff(t, filepath.Join(dir, "expect.ll.new"), []byte(dst.String()), want)
		})
	}
}

func TestRewriteTypeMethodTablesInPlace(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := parseModule(t, &ctx, filepath.Join("testdata", "method_slots", "in.ll"))
	defer mod.Dispose()

	typeName := taskTypeName
	g := mod.NamedGlobal(typeName)
	if g.IsNil() {
		t.Fatalf("missing package-owned type global %q", typeName)
	}
	linkage := g.Linkage()
	if got := RewriteTypeMethodTables(mod, map[string][]int{typeName: {1}, ptrTaskTypeName: {1}}, false); got != 2 {
		t.Fatalf("RewriteTypeMethodTables rewrote %d globals, want 2", got)
	}
	if got := g.Linkage(); got != linkage {
		t.Fatalf("type global linkage changed from %v to %v", linkage, got)
	}

	out := mod.String()
	if !strings.Contains(out, `ptr @"main.(*Task).Run", ptr @main.Task.Run`) {
		t.Fatalf("live method slot was not preserved:\n%s", out)
	}
	if strings.Contains(out, `ptr @"main.(*Task).Drop", ptr @"main.Task.Drop"`) {
		t.Fatalf("dead method slot still references Drop:\n%s", out)
	}
	if !strings.Contains(out, unreachableMethodName) {
		t.Fatalf("rewritten module does not reference unreachable method:\n%s", out)
	}
	count := 0
	for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		if global.Name() == typeName {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("package type global count = %d, want exactly one", count)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("rewritten package module is invalid: %v\n%s", err, out)
	}
}

func TestMethodArray(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	initWithLast := func(last llvm.Value) llvm.Value {
		return llvm.ConstStruct([]llvm.Value{llvm.ConstNull(ctx.Int8Type()), last}, false)
	}
	intValue := llvm.ConstInt(ctx.Int8Type(), 1, false)
	methodTy := ctx.StructCreateNamed(abiMethodTypeName)
	methodTy.StructSetBody([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
	method := llvm.ConstNamedStruct(methodTy, []llvm.Value{intValue, intValue, intValue, intValue})
	methods := llvm.ConstArray(methodTy, []llvm.Value{method, method})

	methodsVal, elemTy, ok := methodArray(initWithLast(methods))
	if !ok {
		t.Fatal("methodArray failed to recognize an ABI method array")
	}
	if methodsVal.OperandsCount() != 2 {
		t.Fatalf("methodArray returned %d methods, want 2", methodsVal.OperandsCount())
	}
	if elemTy.StructElementTypesCount() != 4 {
		t.Fatalf("methodArray returned %d fields, want 4", elemTy.StructElementTypesCount())
	}

	arrayOfInts := llvm.ConstArray(ctx.Int8Type(), []llvm.Value{intValue})
	wrongFieldsTy := ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
	wrongFields := llvm.ConstNamedStruct(wrongFieldsTy, []llvm.Value{intValue, intValue, intValue})
	wrongNameTy := ctx.StructCreateNamed("external/" + abiMethodTypeName)
	wrongNameTy.StructSetBody([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
	wrongName := llvm.ConstNamedStruct(wrongNameTy, []llvm.Value{intValue, intValue, intValue, intValue})

	tests := []struct {
		name string
		init llvm.Value
	}{
		{name: "nil", init: llvm.Value{}},
		{name: "no operands", init: llvm.ConstNull(ctx.Int32Type())},
		{name: "last operand is not array", init: initWithLast(intValue)},
		{name: "array element is not struct", init: initWithLast(arrayOfInts)},
		{name: "struct has wrong field count", init: initWithLast(llvm.ConstArray(wrongFieldsTy, []llvm.Value{wrongFields}))},
		{name: "struct name only contains ABI name", init: initWithLast(llvm.ConstArray(wrongNameTy, []llvm.Value{wrongName}))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, ok := methodArray(tt.init); ok {
				t.Fatalf("methodArray recognized invalid initializer: %s", tt.name)
			}
		})
	}
}

func TestCloneTypesAcrossContexts(t *testing.T) {
	srcCtx := llvm.NewContext()
	defer srcCtx.Dispose()
	dstCtx := llvm.NewContext()
	defer dstCtx.Dispose()
	dst := dstCtx.NewModule("dst")
	defer dst.Dispose()

	emitter := newOverrideEmitter(dst)
	named := srcCtx.StructCreateNamed("named")
	named.StructSetBody([]llvm.Type{srcCtx.Int32Type()}, false)
	existing := srcCtx.StructCreateNamed("existing")
	existing.StructSetBody([]llvm.Type{srcCtx.Int32Type()}, false)
	dstExisting := dstCtx.StructCreateNamed("existing")
	dstExisting.StructSetBody([]llvm.Type{dstCtx.Int64Type()}, false)

	types := []llvm.Type{
		srcCtx.VoidType(),
		srcCtx.FloatType(),
		srcCtx.DoubleType(),
		srcCtx.X86FP80Type(),
		srcCtx.FP128Type(),
		srcCtx.PPCFP128Type(),
		srcCtx.LabelType(),
		srcCtx.IntType(17),
		llvm.FunctionType(srcCtx.VoidType(), []llvm.Type{srcCtx.Int32Type()}, true),
		named,
		existing,
		srcCtx.StructType([]llvm.Type{srcCtx.Int8Type(), srcCtx.Int16Type()}, true),
		llvm.ArrayType(srcCtx.Int32Type(), 3),
		llvm.PointerType(srcCtx.Int8Type(), 5),
		llvm.VectorType(srcCtx.Int16Type(), 4),
		srcCtx.MetadataType(),
		srcCtx.TokenType(),
	}
	for _, src := range types {
		got := emitter.cloneType(src)
		if got.IsNil() {
			t.Fatalf("cloneType(%s) returned nil", src)
		}
		if got.TypeKind() != src.TypeKind() {
			t.Fatalf("cloneType(%s) kind = %v, want %v", src, got.TypeKind(), src.TypeKind())
		}
		if cached := emitter.cloneType(src); cached.C != got.C {
			t.Fatalf("cloneType(%s) did not reuse the destination type", src)
		}
	}

	clonedNamed := emitter.cloneType(named)
	if clonedNamed.C == named.C || clonedNamed.StructElementTypesCount() != 1 {
		t.Fatalf("identified struct was not recreated in the destination context: %s", clonedNamed)
	}
	if got := emitter.cloneType(existing); got.C != dstExisting.C {
		t.Fatalf("existing destination type was not reused: got %s, want %s", got, dstExisting)
	}
	if got := dstExisting.StructElementTypes()[0].IntTypeWidth(); got != 64 {
		t.Fatalf("existing destination type body was overwritten: width = %d, want 64", got)
	}
}

func TestCloneConstantsAcrossContexts(t *testing.T) {
	srcCtx := llvm.NewContext()
	defer srcCtx.Dispose()
	dstCtx := llvm.NewContext()
	defer dstCtx.Dispose()
	src := srcCtx.NewModule("src")
	defer src.Dispose()
	dst := dstCtx.NewModule("dst")
	defer dst.Dispose()

	emitter := newOverrideEmitter(dst)
	i32 := srcCtx.Int32Type()
	i64 := srcCtx.Int64Type()
	zero := llvm.ConstInt(i32, 0, false)
	one := llvm.ConstInt(i32, 1, false)
	arrayTy := llvm.ArrayType(i32, 2)
	array := llvm.AddGlobal(src, arrayTy, "array")
	array.SetInitializer(llvm.ConstArray(i32, []llvm.Value{zero, one}))
	ptrTy := llvm.PointerType(srcCtx.Int8Type(), 0)
	local := llvm.AddGlobal(src, i32, "local")
	local.SetLinkage(llvm.InternalLinkage)
	local.SetGlobalConstant(true)
	local.SetAlignment(8)
	local.SetInitializer(one)
	fn := llvm.AddFunction(src, "function", llvm.FunctionType(srcCtx.VoidType(), nil, false))

	structTy := srcCtx.StructType([]llvm.Type{i32, srcCtx.DoubleType()}, false)
	constants := []llvm.Value{
		llvm.ConstNull(i32),
		llvm.Undef(i32),
		one,
		llvm.ConstFloat(srcCtx.DoubleType(), 1.5),
		srcCtx.ConstString("llgo", false),
		llvm.ConstNamedStruct(structTy, []llvm.Value{one, llvm.ConstFloat(srcCtx.DoubleType(), 2.5)}),
		llvm.ConstArray(ptrTy, []llvm.Value{array, array}),
		llvm.ConstVector([]llvm.Value{array, array}, false),
		fn,
		array,
		local,
	}
	for i, source := range constants {
		cloned := emitter.cloneConst(source)
		if cloned.IsNil() {
			t.Fatalf("cloneConst(%s) returned nil", source)
		}
		if cached := emitter.cloneConst(source); cached.C != cloned.C && !source.IsAConstantInt().IsNil() {
			t.Fatalf("cloneConst(%s) did not reuse the destination value", source)
		}
		if source.IsAGlobalValue().IsNil() {
			global := llvm.AddGlobal(dst, cloned.Type(), fmt.Sprintf("clone.%d", i))
			global.SetInitializer(cloned)
		}
	}

	ptrInt := llvm.ConstPtrToInt(array, i64)
	expressions := []llvm.Value{
		llvm.ConstGEP(arrayTy, array, []llvm.Value{zero, one}),
		llvm.ConstIntToPtr(llvm.ConstInt(i64, 16, false), llvm.PointerType(srcCtx.Int8Type(), 0)),
		ptrInt,
		llvm.ConstTrunc(ptrInt, i32),
		llvm.ConstAdd(ptrInt, llvm.ConstInt(i64, 1, false)),
		llvm.ConstSub(ptrInt, llvm.ConstInt(i64, 1, false)),
		llvm.ConstXor(ptrInt, llvm.ConstInt(i64, 1, false)),
	}
	for i, source := range expressions {
		cloned := emitter.cloneConst(source)
		global := llvm.AddGlobal(dst, cloned.Type(), fmt.Sprintf("expr.%d", i))
		global.SetInitializer(cloned)
	}

	if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("cloned constants cross LLVM Contexts: %v\n%s", err, dst.String())
	}
}

func TestCloneConstantsRejectsUnsupportedLLVMForms(t *testing.T) {
	srcCtx := llvm.NewContext()
	defer srcCtx.Dispose()
	dstCtx := llvm.NewContext()
	defer dstCtx.Dispose()

	path := filepath.Join(t.TempDir(), "constants.ll")
	ir := `
@target = external global i8
@bitcast_expr = global double bitcast (i64 ptrtoint (ptr @target to i64) to double)
@trunc_expr = global i32 trunc (i64 add (i64 ptrtoint (ptr @target to i64), i64 1) to i32)
@unsupported_expr = global i64 mul (i64 ptrtoint (ptr @target to i64), i64 2)
@block_addr = global ptr blockaddress(@block_target, %entry)

define void @block_target() {
entry:
  ret void
}

declare half @unsupported_type()
`
	if err := os.WriteFile(path, []byte(ir), 0o644); err != nil {
		t.Fatal(err)
	}
	src := parseModule(t, &srcCtx, path)
	defer src.Dispose()
	dst := dstCtx.NewModule("dst")
	defer dst.Dispose()
	emitter := newOverrideEmitter(dst)

	for _, name := range []string{"bitcast_expr", "trunc_expr"} {
		value := src.NamedGlobal(name).Initializer()
		clone := emitter.cloneConst(value)
		if clone.IsNil() {
			t.Fatalf("cloneConst(%s) returned nil", name)
		}
		global := llvm.AddGlobal(dst, clone.Type(), "clone."+name)
		global.SetInitializer(clone)
	}
	if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("cloned bitcast/trunc expressions are invalid: %v\n%s", err, dst.String())
	}

	assertPanic := func(name, want string, run func()) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil || !strings.Contains(fmt.Sprint(got), want) {
					t.Fatalf("panic = %v, want substring %q", got, want)
				}
			}()
			run()
		})
	}
	assertPanic("constant", "unsupported constant", func() {
		emitter.cloneConst(src.NamedGlobal("block_addr").Initializer())
	})
	assertPanic("constant expression", "unsupported constant expression", func() {
		emitter.cloneConst(src.NamedGlobal("unsupported_expr").Initializer())
	})
	assertPanic("type", "unsupported LLVM type kind", func() {
		emitter.cloneType(src.NamedFunction("unsupported_type").GlobalValueType().ReturnType())
	})
}

func parseModule(t *testing.T, ctx *llvm.Context, path string) llvm.Module {
	t.Helper()
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	return mod
}
