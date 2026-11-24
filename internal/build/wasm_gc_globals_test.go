package build

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ssa "github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llgo/internal/llvmgen/ssatest"
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
			constant := llvm.AddGlobal(mod, llvm.ArrayType(ctx.Int32Type(), 2), "packed_numbers")
			constant.SetInitializer(llvm.ConstArray(ctx.Int32Type(), []llvm.Value{llvm.ConstInt(ctx.Int32Type(), 12345678, false), llvm.ConstInt(ctx.Int32Type(), 23456789, false)}))
			constant.SetGlobalConstant(true)
			mutable := llvm.AddGlobal(mod, ctx.Int32Type(), "mutable_root")
			mutable.SetInitializer(llvm.ConstInt(ctx.Int32Type(), 0, false))
			decl := llvm.AddGlobal(mod, ctx.Int32Type(), "foreign_constant")
			decl.SetGlobalConstant(true)
			custom := llvm.AddGlobal(mod, ctx.Int32Type(), "custom_constant")
			custom.SetInitializer(llvm.ConstInt(ctx.Int32Type(), 1, false))
			custom.SetGlobalConstant(true)
			custom.SetSection("custom")
			relocated := llvm.AddGlobal(mod, llvm.PointerType(ctx.Int32Type(), 0), "type_test_member")
			relocated.SetInitializer(mutable)
			relocated.SetGlobalConstant(true)
			encoded := llvm.AddGlobal(mod, ctx.Int64Type(), "type_test_encoded_pointer")
			encoded.SetInitializer(llvm.ConstPtrToInt(mutable, ctx.Int64Type()))
			encoded.SetGlobalConstant(true)
			prepareWasmGCGlobals(mod, tc.roots, "roots", true)
			if constant.Section() != tc.want || mutable.Section() != "" || decl.Section() != "" || custom.Section() != "custom" || relocated.Section() != "" || encoded.Section() != "" {
				t.Fatal("incorrect root exclusion", mod.String())
			}
		})
	}
}

func TestWasmNoScanStringsShareAcrossPackages(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("Clang unavailable")
	}
	linker, err := exec.LookPath("wasm-ld")
	if err != nil {
		t.Skip("wasm-ld unavailable")
	}
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-unknown"})
	defer prog.Dispose()
	prog.EnableGCRoots(true)
	const text = "shared-no-scan-string-across-packages"
	dir := t.TempDir()
	args := []string{"--no-entry", "--export=get0", "--export=get1", "--export=bytes0", "--export=bytes1"}
	for i := 0; i < 2; i++ {
		pkg := prog.NewPackage("literal"+string(rune('a'+i)), "literal")
		pkg.ConstString(text)
		pkg.ConstBytes([]byte(text))
		mod := pkg.Module()
		var data, mutable llvm.Value
		for g := mod.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
			if strings.HasPrefix(g.Name(), "__llgo_noscan_string$") {
				data = g
			} else if !g.IsGlobalConstant() && g.GlobalValueType().TypeKind() == llvm.ArrayTypeKind {
				mutable = g
			}
		}
		if data.IsNil() || mutable.IsNil() {
			t.Fatal("missing string or writable byte storage")
		}
		ctx := mod.Context()
		name := "get" + string(rune('0'+i))
		fn := llvm.AddFunction(mod, name, llvm.FunctionType(llvm.PointerType(ctx.Int8Type(), 0), nil, false))
		b := ctx.NewBuilder()
		b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
		b.CreateRet(data)
		b.Dispose()
		fn = llvm.AddFunction(mod, "bytes"+string(rune('0'+i)), llvm.FunctionType(llvm.PointerType(ctx.Int8Type(), 0), nil, false))
		b = ctx.NewBuilder()
		b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
		b.CreateRet(mutable)
		b.Dispose()
		prepareWasmGCGlobals(mod, true, name, false)
		input, object := filepath.Join(dir, name+".ll"), filepath.Join(dir, name+".o")
		if err := os.WriteFile(input, []byte(mod.String()), 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(clang, "--target=wasm32-unknown-unknown", "-flto=full", "-c", input, "-o", object)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("compile: %v\n%s", err, out)
		}
		args = append(args, object)
	}
	output := filepath.Join(dir, "shared.wasm")
	args = append(args, "-o", output)
	if out, err := exec.Command(linker, args...).CombinedOutput(); err != nil {
		t.Fatalf("link: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	// One immutable string is shared; each writable []byte retains its own copy.
	if n := bytes.Count(raw, []byte(text)); n != 3 {
		t.Fatalf("storage copies after full LTO: %d, want 3", n)
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
	prepareWasmGCGlobals(mod, true, "linker-test", false)
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
