package build

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWasmGCConstantSections(t *testing.T) {
	for _, tc := range []struct {
		triple string
		roots  bool
		want   string
	}{
		{"wasm32-unknown-emscripten", true, wasmGCNoScanSection},
		{"wasm64-unknown-emscripten", true, wasmGCNoScanSection},
		{"wasm32-unknown-wasip1", true, wasmGCNoScanSection},
		{"wasm32-unknown-wasip1", false, ""},
		{"aarch64-unknown-linux-gnu", true, ""},
	} {
		t.Run(tc.triple+tc.want, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			mod := ctx.NewModule("roots")
			defer mod.Dispose()
			mod.SetTarget(tc.triple)
			constant := llvm.AddGlobal(mod, ctx.Int32Type(), "packed_numbers")
			constant.SetInitializer(llvm.ConstInt(ctx.Int32Type(), 12345678, false))
			constant.SetGlobalConstant(true)
			mutable := llvm.AddGlobal(mod, ctx.Int32Type(), "mutable_root")
			mutable.SetInitializer(llvm.ConstInt(ctx.Int32Type(), 0, false))
			decl := llvm.AddGlobal(mod, ctx.Int32Type(), "foreign_constant")
			decl.SetGlobalConstant(true)
			custom := llvm.AddGlobal(mod, ctx.Int32Type(), "custom_constant")
			custom.SetInitializer(llvm.ConstInt(ctx.Int32Type(), 1, false))
			custom.SetGlobalConstant(true)
			custom.SetSection("custom")
			prepareWasmGCGlobals(mod, tc.roots, "roots")
			if constant.Section() != tc.want || mutable.Section() != "" || decl.Section() != "" || custom.Section() != "custom" {
				t.Fatal("incorrect root exclusion", mod.String())
			}
		})
	}
}

func TestWasmGCNoScanPreservesLinkerGC(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("Clang unavailable")
	}
	linker, err := exec.LookPath("wasm-ld")
	if err != nil {
		t.Skip("wasm-ld unavailable")
	}
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	dir := t.TempDir()
	input, object, output := filepath.Join(dir, "input.ll"), filepath.Join(dir, "input.o"), filepath.Join(dir, "output.wasm")
	if err := os.WriteFile(input, []byte(`target triple = "wasm32-unknown-unknown"
@a = private constant [8 x i8] c"LIVE_DAT"
@b = private constant {ptr, [9 x i8]} {ptr @dead_function, [9 x i8] c"DEAD_DATA"}
@__start_llgo_gc_noscan = external global i8
@__stop_llgo_gc_noscan = external global i8
define ptr @keep() { ret ptr @a }
define ptr @start() { ret ptr @__start_llgo_gc_noscan }
define ptr @stop() { ret ptr @__stop_llgo_gc_noscan }
define i32 @dead_function() { ret i32 42 }
`), 0600); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(input)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Dispose()
	prepareWasmGCGlobals(mod, true, "linker-test")
	if err := os.WriteFile(input, []byte(mod.String()), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []*exec.Cmd{
		exec.Command(clang, "--target=wasm32-unknown-unknown", "-c", input, "-o", object),
		exec.Command(linker, "--no-entry", "--export=keep", "--export=start", "--export=stop", object, "-o", output),
	} {
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", command.Args, err, out)
		}
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("LIVE_DAT")) || bytes.Contains(raw, []byte("DEAD_DATA")) || bytes.Contains(raw, []byte("dead_function")) {
		t.Fatal("root scan boundaries kept unused data or its function alive")
	}
}
