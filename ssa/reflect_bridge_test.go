//go:build !llgo

package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWasmReflectBridgeProviderSelection(t *testing.T) {
	tests := []struct {
		name     string
		target   *Target
		expected bool
	}{
		{"GoJS J32", &Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "gojs", WasmReflectBridges: true}, false},
		{"WASI W32", &Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi", WasmReflectBridges: true}, true},
		{"unused GoJS J32", &Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "gojs"}, false},
		{"Emscripten J32", &Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"}, false},
		{"Emscripten J64", &Target{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", WasmProvider: "emscripten"}, false},
		{"unresolved wasm", &Target{GOOS: "js", GOARCH: "wasm"}, false},
		{"native", &Target{GOOS: "linux", GOARCH: "amd64", WasmProvider: "gojs"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.target.usesWasmReflectBridges(); got != test.expected {
				t.Fatalf("usesWasmReflectBridges() = %v, want %v", got, test.expected)
			}
		})
	}
}

func TestWasmReflectBridgeShapeDeduplication(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi", WasmReflectBridges: true})
	defer prog.Dispose()
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")

	signature := func(in, out types.Type) *types.Signature {
		return types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewParam(token.NoPos, nil, "x", in)),
			types.NewTuple(types.NewParam(token.NoPos, nil, "", out)), false)
	}
	intBridge := pkg.wasmReflectBridge(signature(types.Typ[types.Int64], types.Typ[types.Uint64]))
	unsignedBridge := pkg.wasmReflectBridge(signature(types.Typ[types.Uint64], types.Typ[types.Int64]))
	if intBridge != unsignedBridge {
		t.Fatal("signedness-only distinction generated duplicate bridges")
	}

	pointerBridge := pkg.wasmReflectBridge(signature(types.Typ[types.UnsafePointer], types.Typ[types.UnsafePointer]))
	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	mapBridge := pkg.wasmReflectBridge(signature(mapType, mapType))
	if pointerBridge != mapBridge {
		t.Fatal("equivalent directly rooted pointers generated duplicate bridges")
	}

	emptyInterface := types.NewInterfaceType(nil, nil).Complete()
	interfaceBridge := pkg.wasmReflectBridge(signature(emptyInterface, emptyInterface))
	closureBridge := pkg.wasmReflectBridge(signature(NoArgsNoRet, NoArgsNoRet))
	if interfaceBridge == closureBridge {
		t.Fatal("interface and closure storage conversions shared a bridge")
	}
	if got := len(pkg.wasmReflectBridges); got != 4 {
		t.Fatalf("bridge shapes = %d, want 4", got)
	}
	if intBridge.call.Name() != "$ca" || intBridge.make.Name() != "$ma" ||
		pointerBridge.call.Name() != "$cb" || pointerBridge.make.Name() != "$mb" ||
		interfaceBridge.call.Name() != "$cc" || interfaceBridge.make.Name() != "$mc" ||
		closureBridge.call.Name() != "$cd" || closureBridge.make.Name() != "$md" {
		t.Fatalf("bridge names = %q/%q, %q/%q", intBridge.call.Name(), intBridge.make.Name(), interfaceBridge.call.Name(), interfaceBridge.make.Name())
	}
	if err := llvm.VerifyModule(pkg.mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := pkg.String()
	for _, want := range []string{"define internal", "load i64", "br i1"} {
		if !strings.Contains(ir, want) {
			t.Errorf("missing %q in bridge:\n%s", want, ir)
		}
	}
	for _, unwanted := range []string{"comdat", "reflect.call", "reflect.make"} {
		if strings.Contains(ir, unwanted) {
			t.Errorf("unexpected %q in compact bridge:\n%s", unwanted, ir)
		}
	}
}

func TestCompactWasmBridgeID(t *testing.T) {
	tests := map[int]string{0: "a", 25: "z", 26: "A", 61: "9", 62: "aa", 63: "ab", 123: "a9", 124: "ba"}
	for value, expected := range tests {
		if got := compactWasmBridgeID(value); got != expected {
			t.Errorf("compactWasmBridgeID(%d) = %q, want %q", value, got, expected)
		}
	}
}

func TestWasmLLVMTypeShape(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	tests := []struct {
		name string
		typ  llvm.Type
		want string
	}{
		{"void", ctx.VoidType(), "v"},
		{"float", ctx.FloatType(), "f"},
		{"double", ctx.DoubleType(), "d"},
		{"x86 fp80", ctx.X86FP80Type(), "x80"},
		{"fp128", ctx.FP128Type(), "f128"},
		{"ppc fp128", ctx.PPCFP128Type(), "p128"},
		{"integer", ctx.IntType(17), "i17"},
		{"function", llvm.FunctionType(ctx.VoidType(), []llvm.Type{ctx.Int32Type()}, true), "(i32,*)v"},
		{"struct", ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int16Type()}, false), "{i8,i16,}"},
		{"packed struct", ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int16Type()}, true), "<i8,i16,>"},
		{"array", llvm.ArrayType(ctx.Int32Type(), 3), "[3:i32]"},
		{"pointer", llvm.PointerType(ctx.Int8Type(), 5), "p5"},
		{"vector", llvm.VectorType(ctx.Int16Type(), 4), "V4:i16"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var shape strings.Builder
			appendWasmLLVMTypeShape(&shape, test.typ)
			if got := shape.String(); got != test.want {
				t.Fatalf("shape = %q, want %q", got, test.want)
			}
		})
	}
	t.Run("unsupported", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("unsupported LLVM type did not panic")
			}
		}()
		appendWasmLLVMTypeShape(new(strings.Builder), ctx.LabelType())
	})
}

func TestWasmReflectBridgeEmptyAndMultipleResults(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi", WasmReflectBridges: true})
	defer prog.Dispose()
	setTestRuntime(t, prog)
	prog.EnableGCRoots(true)
	pkg := prog.NewPackage("p", "example.com/p")

	pkg.wasmReflectBridge(NoArgsNoRet)
	results := types.NewTuple(
		types.NewParam(token.NoPos, nil, "", types.Typ[types.Int64]),
		types.NewParam(token.NoPos, nil, "", types.Typ[types.String]),
	)
	pkg.wasmReflectBridge(types.NewSignatureType(nil, nil, nil, nil, results, false))
	if err := llvm.VerifyModule(pkg.mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	if got := len(pkg.wasmReflectBridges); got != 2 {
		t.Fatalf("bridge shapes = %d, want 2", got)
	}
}

func TestWasmMethodExpressionSignature(t *testing.T) {
	plain := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	if got := methodExprSignature(plain); got != plain {
		t.Fatal("non-method signature was rewritten")
	}
	pkg := types.NewPackage("example.com/p", "p")
	recv := types.NewVar(token.NoPos, pkg, "receiver", types.NewPointer(types.Typ[types.Int]))
	param := types.NewParam(token.NoPos, pkg, "values", types.NewSlice(types.Typ[types.String]))
	result := types.NewParam(token.NoPos, pkg, "", types.Typ[types.Bool])
	method := types.NewSignatureType(recv, nil, nil, types.NewTuple(param), types.NewTuple(result), true)
	expression := methodExprSignature(method)
	if expression.Recv() != nil || expression.Params().Len() != 2 || expression.Params().At(0).Type() != recv.Type() || expression.Params().At(1) != param || expression.Results().At(0) != result || !expression.Variadic() {
		t.Fatalf("method expression signature = %s", expression)
	}
}

func TestWasmReflectRootShape(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi", WasmReflectBridges: true})
	defer prog.Dispose()
	setTestRuntime(t, prog)

	shape := func(typ types.Type) string {
		var value strings.Builder
		prog.appendWasmRootShape(&value, prog.Type(typ, InC))
		return value.String()
	}
	ptr := types.NewPointer(types.Typ[types.Int])
	directRoot := shape(ptr)
	for name, typ := range map[string]types.Type{
		"map":     types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		"channel": types.NewChan(types.SendRecv, types.Typ[types.Int]),
	} {
		if got := shape(typ); got != directRoot {
			t.Errorf("%s root shape = %q, want direct root %q", name, got, directRoot)
		}
	}
	if got := shape(types.Typ[types.Int]); got == directRoot {
		t.Fatalf("scalar root shape = %q, want a non-root shape", got)
	}

	stringShape := shape(types.Typ[types.String])
	sliceShape := shape(types.NewSlice(types.Typ[types.Int]))
	if stringShape == sliceShape || !strings.HasSuffix(stringShape, "0r") || !strings.HasSuffix(sliceShape, "0r") {
		t.Fatalf("string/slice root shapes = %q/%q", stringShape, sliceShape)
	}
	emptyInterface := types.NewInterfaceType(nil, nil).Complete()
	if got := shape(emptyInterface); got != "i1r" {
		t.Fatalf("interface root shape = %q", got)
	}

	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "Pointer", ptr, false),
		types.NewField(token.NoPos, nil, "Scalar", types.Typ[types.Int], false),
	}
	if got := shape(types.NewStruct(fields, nil)); got != "{r,-,}" {
		t.Fatalf("struct root shape = %q", got)
	}
	tuple := types.NewTuple(
		types.NewParam(token.NoPos, nil, "pointer", ptr),
		types.NewParam(token.NoPos, nil, "scalar", types.Typ[types.Int]),
	)
	if got := shape(tuple); got != "{r,-,}" {
		t.Fatalf("tuple root shape = %q", got)
	}
	if got := shape(types.NewArray(ptr, 2)); got != "[2:r]" {
		t.Fatalf("array root shape = %q", got)
	}
}

func TestExtractConstStringFromWideWasmStorage(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi", WasmReflectBridges: true})
	defer prog.Dispose()
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")
	fn := pkg.NewFunc("f", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)
	defer b.Dispose()
	value := b.Str("Add")
	if got, ok := extractConstString(value.impl); !ok || got != "Add" {
		t.Fatalf("extractConstString(%s) = %q, %v", value.impl.String(), got, ok)
	}
	b.Return()
	b.EndBuild()
}
