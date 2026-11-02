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
		wantLog   string
	}{
		{
			name:    "method_slots",
			wantLog: "[dce] drop method _llgo_main.Task[0] ifn=main.(*Task).Drop tfn=main.Task.Drop\n",
			liveSlots: map[string][]int{
				taskTypeName:    {1}, // Run
				ptrTaskTypeName: {1}, // Run
			},
		},
		{
			name:    "method_slots_wasm32",
			wantLog: "[dce] drop method _llgo_main.Task[0] ifn=Drop tfn=Run\n",
			liveSlots: map[string][]int{
				taskTypeName: {1}, // Run shares its entry with a dropped slot.
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

			logPath := filepath.Join(t.TempDir(), "dce.log")
			logFile, err := os.Create(logPath)
			if err != nil {
				t.Fatal(err)
			}
			func() {
				stderr := os.Stderr
				os.Stderr = logFile
				defer func() {
					os.Stderr = stderr
					logFile.Close()
				}()
				EmitStrongTypeOverrides(dst, []llvm.Module{src}, tt.liveSlots, nil, true)
			}()
			log, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(log), tt.wantLog) {
				t.Errorf("verbose log = %q, want line %q", log, tt.wantLog)
			}
			for global := dst.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
				if init := global.Initializer(); !init.IsNil() && global.GlobalValueType().C != init.Type().C {
					t.Errorf("global %s type %s does not match initializer type %s", global.Name(), global.GlobalValueType(), init.Type())
				}
			}
			if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("cross-context override is invalid: %v\n%s", err, dst.String())
			}
			// LLVM's verifier does not catch every malformed aggregate constant.
			// The build writes textual IR for Clang, so it must also parse again.
			// Parsing itself is the check: parseModule fails the test on invalid IR.
			roundTripCtx := llvm.NewContext()
			defer roundTripCtx.Dispose()
			roundTripPath := filepath.Join(t.TempDir(), "override.ll")
			if err := os.WriteFile(roundTripPath, []byte(dst.String()), 0600); err != nil {
				t.Fatal(err)
			}
			roundTrip := parseModule(t, &roundTripCtx, roundTripPath)
			defer roundTrip.Dispose()
			want, err := os.ReadFile(filepath.Join(dir, "expect.ll"))
			if err != nil {
				t.Fatal(err)
			}
			qtest.Diff(t, filepath.Join(dir, "expect.ll.new"), []byte(dst.String()), want)
		})
	}
}

func TestMethodPointerConstant(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("method_pointer")
	defer mod.Dispose()
	fn := llvm.AddFunction(mod, "method", llvm.FunctionType(ctx.VoidType(), nil, false))
	ptr := fn.Type()
	named := ctx.StructCreateNamed("pointer.slot")
	named.StructSetBody([]llvm.Type{ptr, ctx.Int32Type()}, false)

	for _, typ := range []llvm.Type{
		ptr,
		ctx.StructType([]llvm.Type{ptr, ctx.Int32Type()}, false),
		named,
		ctx.StructType([]llvm.Type{ptr, ctx.Int16Type()}, true),
	} {
		t.Run(typ.String(), func(t *testing.T) {
			got := methodPointerConstant(typ, fn)
			if got.Type() != typ {
				t.Fatalf("type = %s, want exact slot type %s", got.Type(), typ)
			}
			if typ == ptr {
				if got != fn {
					t.Fatal("bare method pointer changed")
				}
			} else if got.Operand(0) != fn || !got.Operand(1).IsNull() {
				t.Fatalf("slot = %s, want method pointer with zero padding", got)
			}
		})
	}

	for _, tt := range []struct {
		name string
		typ  llvm.Type
	}{
		{"integer", ctx.Int64Type()},
		{"different address space", llvm.PointerType(ctx.Int8Type(), 1)},
		{"empty struct", ctx.StructType(nil, false)},
		{"missing padding", ctx.StructType([]llvm.Type{ptr}, false)},
		{"extra field", ctx.StructType([]llvm.Type{ptr, ctx.Int32Type(), ctx.Int32Type()}, false)},
		{"non-pointer first field", ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int32Type()}, false)},
		{"non-integer padding", ctx.StructType([]llvm.Type{ptr, ptr}, false)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				want := fmt.Sprintf("dcepass: unsupported method pointer storage type %s", tt.typ)
				if got := recover(); got != want {
					t.Fatalf("panic = %v, want %q", got, want)
				}
			}()
			methodPointerConstant(tt.typ, fn)
		})
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
	empty := srcCtx.StructCreateNamed("empty")
	empty.StructSetBody(nil, false)
	opaque := srcCtx.StructCreateNamed("opaque")
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
		empty,
		opaque,
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
	if got := emitter.cloneType(empty); got.IsStructOpaque() || got.StructElementTypesCount() != 0 {
		t.Fatalf("defined empty struct was not preserved: %s", got)
	}
	if got := emitter.cloneType(opaque); !got.IsStructOpaque() {
		t.Fatalf("opaque struct was not preserved: %s", got)
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
@addrspace_target = external addrspace(1) global i8
@unsupported_expr = global ptr addrspacecast (ptr addrspace(1) @addrspace_target to ptr)
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

func TestCloneFloatConstantsAcrossContexts(t *testing.T) {
	// Wider significands and NaN payloads must survive without converting to
	// float64. Use parsed source IR so construction is independent of cloning.
	ir := `
@negative_zero = constant double 0x8000000000000000
@signaling_nan = constant double 0x7FF0000000000001
@float_nan = constant float 0x7FF82468A0000000
@infinity = constant double 0x7FF0000000000000
@subnormal = constant double 0x0000000000000001
@fp80_precision = constant x86_fp80 0xK3FFF8000000000000001
@fp80_nan = constant x86_fp80 0xK7FFFC000000000000123
@fp128_precision = constant fp128 0xL00000000000000013FFF000000000000
@fp128_nan = constant fp128 0xL00000000000001237FFF800000000000
@ppc_low_only = constant ppc_fp128 0xM00000000000000003FF0000000000000
@ppc_precision = constant ppc_fp128 0xM3FF00000000000003C90000000000000
`
	path := filepath.Join(t.TempDir(), "floats.ll")
	if err := os.WriteFile(path, []byte(ir), 0600); err != nil {
		t.Fatal(err)
	}
	srcCtx, dstCtx := llvm.NewContext(), llvm.NewContext()
	defer dstCtx.Dispose()
	src := parseModule(t, &srcCtx, path)
	dst := dstCtx.NewModule("cloned-floats")
	defer dst.Dispose()
	emitter := newOverrideEmitter(dst)
	want := make(map[string]string)
	for global := src.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		value := global.Initializer()
		// Also exercise the recursive path used by ABI metadata initializers.
		for _, nested := range []bool{false, true} {
			name := global.Name()
			if nested {
				name += "_nested"
				value = srcCtx.ConstStruct([]llvm.Value{value}, false)
			}
			want[name] = value.String()
			cloned := emitter.cloneConst(value)
			out := llvm.AddGlobal(dst, cloned.Type(), name)
			out.SetInitializer(cloned)
		}
	}
	src.Dispose()
	srcCtx.Dispose()
	for name, expected := range want {
		if got := dst.NamedGlobal(name).Initializer().String(); got != expected {
			t.Errorf("%s: cloned %s, want %s", name, got, expected)
		}
	}
	if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
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

func TestDropUnobservedMethodMetadata(t *testing.T) {
	srcContext := llvm.NewContext()
	defer srcContext.Dispose()
	dstContext := llvm.NewContext()
	defer dstContext.Dispose()
	text := `
%"github.com/xgo-dev/llgo/runtime/abi.UncommonType" = type { { ptr, i64 }, i16, i16, i32 }
%"github.com/xgo-dev/llgo/runtime/abi.Method" = type { { ptr, i64 }, ptr, ptr, ptr }
@method.name = private constant [4 x i8] c"Drop"
@method.type = external constant i8
@_llgo_hidden = weak_odr constant { i64, %"github.com/xgo-dev/llgo/runtime/abi.UncommonType", [1 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] } {
 i64 42,
 %"github.com/xgo-dev/llgo/runtime/abi.UncommonType" { { ptr, i64 } zeroinitializer, i16 1, i16 1, i32 24 },
 [1 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [ %"github.com/xgo-dev/llgo/runtime/abi.Method" { { ptr, i64 } { ptr @method.name, i64 4 }, ptr @method.type, ptr @method, ptr @method } ]
}
declare void @method()
`
	path := filepath.Join(t.TempDir(), "type.ll")
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	src := parseModule(t, &srcContext, path)
	defer src.Dispose()
	for _, drop := range []bool{false, true} {
		dst := dstContext.NewModule("override")
		EmitStrongTypeOverrides(dst, []llvm.Module{src}, nil, func(string) bool { return drop }, false)
		if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		init := dst.NamedGlobal("_llgo_hidden").Initializer()
		wantCount := uint64(1)
		if drop {
			wantCount = 0
		}
		if got := init.Operand(1).Operand(1).ZExtValue(); got != wantCount {
			t.Fatalf("drop=%v: method count=%d", drop, got)
		}
		if got := init.Operand(2).IsNull(); got != drop {
			t.Fatalf("drop=%v: zero method storage=%v", drop, got)
		}
		if got := init.Operand(2).Type().ArrayLength(); got != int(wantCount) {
			t.Fatalf("drop=%v: method storage length=%d", drop, got)
		}
		if drop && (!dst.NamedGlobal("method.type").IsNil() || !dst.NamedFunction("method").IsNil()) {
			t.Fatal("cleared methods still reference signature or function")
		}
		dst.Dispose()
	}
}

func TestCompactMethodStorageReferences(t *testing.T) {
	for _, bits := range []int{32, 64} {
		for _, mode := range []string{"new", "declaration", "in-place", "live"} {
			t.Run(fmt.Sprintf("%d/%s", bits, mode), func(t *testing.T) {
				srcCtx, dstCtx := llvm.NewContext(), llvm.NewContext()
				defer srcCtx.Dispose()
				defer dstCtx.Dispose()
				text := `
%"github.com/xgo-dev/llgo/runtime/abi.UncommonType" = type { { ptr, i64 }, i16, i16, i32 }
%"github.com/xgo-dev/llgo/runtime/abi.Method" = type { { ptr, i64 }, ptr, ptr, ptr }
%descriptor = type { ptr, %"github.com/xgo-dev/llgo/runtime/abi.UncommonType", [1 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] }
@first = weak_odr hidden constant %descriptor { ptr @second, %"github.com/xgo-dev/llgo/runtime/abi.UncommonType" { { ptr, i64 } zeroinitializer, i16 1, i16 1, i32 24 }, [1 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [%"github.com/xgo-dev/llgo/runtime/abi.Method" { { ptr, i64 } zeroinitializer, ptr null, ptr @method, ptr @method }] }, align 32
@second = weak_odr constant %descriptor { ptr @first, %"github.com/xgo-dev/llgo/runtime/abi.UncommonType" { { ptr, i64 } zeroinitializer, i16 1, i16 1, i32 24 }, [1 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [%"github.com/xgo-dev/llgo/runtime/abi.Method" { { ptr, i64 } zeroinitializer, ptr null, ptr @method, ptr @method }] }
@reference = constant ptr getelementptr (%descriptor, ptr @second, i32 0, i32 1)
declare void @method()
`
				path := filepath.Join(t.TempDir(), "types.ll")
				if err := os.WriteFile(path, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
				src := parseModule(t, &srcCtx, path)
				defer src.Dispose()
				dst := src
				if mode != "in-place" {
					dst = dstCtx.NewModule("dst")
					defer dst.Dispose()
				}
				layout := fmt.Sprintf("e-p:%d:%d-i64:64", bits, bits)
				src.SetDataLayout(layout)
				dst.SetDataLayout(layout)
				data := llvm.NewTargetData(layout)
				defer data.Dispose()
				original := src.NamedGlobal("first").GlobalValueType()
				before := data.TypeAllocSize(original)
				if mode == "declaration" {
					e := newOverrideEmitter(dst)
					// Existing uses of a full-size declaration must also be rebound.
					r := llvm.AddGlobal(dst, e.cloneType(src.NamedGlobal("reference").GlobalValueType()), "reference")
					r.SetInitializer(e.cloneConst(src.NamedGlobal("reference").Initializer()))
				}
				var live map[string][]int
				if mode == "live" {
					live = map[string][]int{"first": {0}, "second": {0}}
				}
				EmitStrongTypeOverrides(dst, []llvm.Module{src}, live, func(string) bool { return true }, false)
				if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"first", "second"} {
					g := dst.NamedGlobal(name)
					typ := g.GlobalValueType()
					want := before
					if mode != "live" {
						want -= data.TypeAllocSize(original.StructElementTypes()[2])
					}
					if got := data.TypeAllocSize(typ); got != want {
						t.Fatalf("%s: storage=%d, want %d", name, got, want)
					}
					for i := 0; i < 3; i++ {
						if data.ElementOffset(typ, i) != data.ElementOffset(original, i) {
							t.Fatalf("%s: field %d moved", name, i)
						}
					}
					other := "first"
					if name == "first" {
						other = "second"
					}
					if g.Initializer().Operand(0) != dst.NamedGlobal(other) {
						t.Fatalf("%s: stale cross-descriptor reference", name)
					}
				}
				first := dst.NamedGlobal("first")
				if first.Visibility() != llvm.HiddenVisibility || first.Alignment() != 32 || first.Linkage() != llvm.ExternalLinkage {
					t.Fatal("descriptor attributes changed")
				}
				if ref := dst.NamedGlobal("reference"); !ref.IsNil() && ref.Initializer().Operand(0) != dst.NamedGlobal("second") {
					t.Fatal("existing GEP still refers to the old descriptor")
				}
			})
		}
	}
}

func TestEmitInterfaceUseOverrides(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(fmt.Sprint(known), func(t *testing.T) {
			sc, dc := llvm.NewContext(), llvm.NewContext()
			defer sc.Dispose()
			defer dc.Dispose()
			src, dst := sc.NewModule("src"), dc.NewModule("dst")
			defer src.Dispose()
			defer dst.Dispose()
			for _, name := range []string{"dead", "live", "unknown"} {
				llvm.AddFunction(src, "_llgo_iface_use."+name, llvm.FunctionType(sc.Int8Type(), nil, false))
			}
			llvm.AddFunction(src, "unrelated", llvm.FunctionType(sc.VoidType(), nil, false))
			llvm.AddFunction(dst, "_llgo_iface_use.live", llvm.FunctionType(dc.Int8Type(), nil, false))
			var reject func(string) bool
			if known {
				reject = func(name string) bool { return name == "dead" }
			}
			EmitInterfaceUseOverrides(dst, []llvm.Module{src, src}, reject)
			// Idempotent when a destination module also appears in the input.
			EmitInterfaceUseOverrides(dst, []llvm.Module{dst}, reject)
			for _, name := range []string{"dead", "live", "unknown"} {
				g := dst.NamedFunction("_llgo_iface_use." + name)
				want := uint64(1)
				if known && name == "dead" {
					want = 0
				}
				if g.FirstBasicBlock().FirstInstruction().Operand(0).ZExtValue() != want {
					t.Fatalf("%s: %s", name, g.String())
				}
			}
			if !dst.NamedFunction("unrelated").IsNil() {
				t.Fatal("copied an unrelated function")
			}
			if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
