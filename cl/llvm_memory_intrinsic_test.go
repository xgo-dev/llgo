package cl

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestLLVMMemoryIntrinsicLinks(t *testing.T) {
	const src = `package memory
import "unsafe"
//go:linkname copyBytes llvm.memcpy
func copyBytes(dst, src unsafe.Pointer, n uintptr, volatile bool)
//go:linkname copyInline llvm.memcpy.inline
func copyInline(dst, src *uint64, n uintptr, volatile bool)
//go:linkname moveBytes llvm.memmove
func moveBytes(dst, src unsafe.Pointer, n uintptr, volatile bool)
//go:linkname fillBytes llvm.memset
func fillBytes(dst unsafe.Pointer, value byte, n uintptr, volatile bool)
//go:linkname fillInline llvm.memset.inline
func fillInline(dst unsafe.Pointer, value byte, n uintptr, volatile bool)
func Copy(dst, src unsafe.Pointer, n uintptr) { copyBytes(dst, src, n, false) }
func CopyInline(dst, src *uint64) { copyInline(dst, src, 8, false) }
func Move(dst, src unsafe.Pointer, n uintptr) { moveBytes(dst, src, n, false) }
func Fill(dst unsafe.Pointer, value byte, n uintptr) { fillBytes(dst, value, n, false) }
func FillInline(dst unsafe.Pointer, value byte) { fillInline(dst, value, 8, false) }
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "7"},
		{GOOS: "linux", GOARCH: "riscv64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", WasmProvider: "emscripten"},
		{GOOS: "js", GOARCH: "wasm", Target: "emscripten-memory64", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64", WasmProvider: "emscripten"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			defer mod.Dispose()
			for name, intrinsic := range map[string]string{
				"Copy": "memcpy", "CopyInline": "memcpy.inline", "Move": "memmove",
				"Fill": "memset", "FillInline": "memset.inline",
			} {
				body := mustNamedFunction(t, mod, "memory."+name).String()
				if !strings.Contains(body, "@llvm."+intrinsic+".") ||
					!strings.Contains(body, "i"+strconv.Itoa(prog.GoWordSize()*8)+" ") {
					t.Fatalf("%s lost its direct memory intrinsic or Go length width:\n%s", name, body)
				}
			}
			mod.SetDataLayout(prog.DataLayout())
			mod.SetTarget(target.Spec().Triple)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			opts.SetVerifyEach(true)
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"CopyInline", "FillInline"} {
				body := mustNamedFunction(t, mod, "memory."+name).String()
				// Some targets keep the intrinsic until instruction selection.
				for _, line := range strings.Split(body, "\n") {
					if strings.Contains(line, "call ") && !strings.Contains(line, "@llvm.") {
						t.Fatalf("fixed-size inline intrinsic retained an external call:\n%s", body)
					}
				}
			}
			buf, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			buf.Dispose()
		})
	}
}

func TestLLVMMemoryIntrinsicLinkErrors(t *testing.T) {
	for _, tc := range []struct{ name, target, declaration, use, want string }{
		{"integer-pointer", "llvm.memcpy", "func f(dst, src uintptr, n uintptr, v bool)", "", "expected pointer"},
		{"result", "llvm.memcpy", "func f(dst, src unsafe.Pointer, n uintptr, v bool) unsafe.Pointer", "", "result:"},
		{"fill-value", "llvm.memset", "func f(dst unsafe.Pointer, value int, n uintptr, v bool)", "", "expected i8"},
		{"slice", "llvm.memcpy", "func f(dst, src []byte, n uintptr, v bool)", "", "expected pointer"},
		{"copy-length", "llvm.memcpy.inline", "func f(dst, src unsafe.Pointer, n uintptr, v bool)", "func use(p unsafe.Pointer, n uintptr) { f(p, p, n, false) }", "argument 3 must be an immediate constant"},
		{"fill-length", "llvm.memset.inline", "func f(dst unsafe.Pointer, value byte, n uintptr, v bool)", "func use(p unsafe.Pointer, n uintptr) { f(p, 0, n, false) }", "argument 3 must be an immediate constant"},
		{"copy-volatile", "llvm.memcpy", "func f(dst, src unsafe.Pointer, n uintptr, v bool)", "func use(p unsafe.Pointer, v bool) { f(p, p, 0, v) }", "argument 4 must be an immediate constant"},
		{"move-volatile", "llvm.memmove", "func f(dst, src unsafe.Pointer, n uintptr, v bool)", "func use(p unsafe.Pointer, v bool) { f(p, p, 0, v) }", "argument 4 must be an immediate constant"},
		{"fill-volatile", "llvm.memset", "func f(dst unsafe.Pointer, value byte, n uintptr, v bool)", "func use(p unsafe.Pointer, v bool) { f(p, 0, 0, v) }", "argument 4 must be an immediate constant"},
		{"function-value", "llvm.memcpy.inline", "func f(dst, src unsafe.Pointer, n uintptr, v bool)", "var Value = f", "cannot use LLVM intrinsic as a function value"},
		{"suffix", "llvm.memcpy.p0.p0.i32", "func f(dst, src unsafe.Pointer, n uintptr, v bool)", "", "expected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package invalid\nimport \"unsafe\"\nvar _ unsafe.Pointer\n//go:linkname f " + tc.target + "\n" + tc.declaration + "\n" + tc.use
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			defer func() {
				got := fmt.Sprint(recover())
				if !strings.Contains(got, tc.want) || !strings.Contains(got, "foo.go:") {
					t.Fatalf("diagnostic = %q, want source location and %q", got, tc.want)
				}
			}()
			if _, err := NewPackage(prog, ssapkg, files); err != nil {
				panic(err)
			}
		})
	}
}
