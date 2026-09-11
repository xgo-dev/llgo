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
