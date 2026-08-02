package dcepass

import (
	"os"
	"path/filepath"
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
