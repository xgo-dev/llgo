package cl

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llvm"
)

func TestWasmCallerLocationScalarABI(t *testing.T) {
	for _, test := range []struct {
		name   string
		target llvmgen.Target
	}{
		{"GJS", llvmgen.Target{GOOS: "js", GOARCH: "wasm"}},
		{"GWASI", llvmgen.Target{GOOS: "wasip1", GOARCH: "wasm"}},
		{"EC64", llvmgen.Target{GOOS: "js", GOARCH: "wasm", Target: "emscripten-memory64", LLVMTarget: "wasm64-unknown-emscripten"}},
		{"native", llvmgen.Target{GOOS: "linux", GOARCH: "amd64"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &test.target)
			if test.name == "EC64" {
			}
			pkg := prog.NewPackage("foo", "example.com/foo")
			fn := pkg.NewFunc("example.com/foo.f", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
			b := fn.MakeBody(1)
			ctx := &context{prog: prog, pkg: pkg, fn: fn}
			for _, name := range []string{"PushCallerLocationFrame", "RecordCallerLocation", "RecordPanicLocation"} {
				ctx.callRuntimeLocation(b, name, b.Convert(prog.Uintptr(), fn.Expr), b.Str("foo.f"), b.Str("foo.go"), prog.Val(42))
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			ir := pkg.Module().String()
			for _, name := range []string{"PushCallerLocationFrame", "RecordCallerLocation", "RecordPanicLocation"} {
				fullName := llvmgen.PkgRuntime + "." + name
				if test.target.GOARCH == "wasm" {
					if !pkg.Module().NamedFunction(fullName).IsNil() {
						t.Fatalf("Wasm still calls aggregate-string helper %s:\n%s", name, ir)
					}
					if helper := pkg.Module().NamedFunction(fullName + "Wasm"); helper.IsNil() || helper.ParamsCount() != 6 {
						t.Fatalf("Wasm helper %s must take six scalar arguments:\n%s", name, ir)
					}
				} else if !pkg.Module().NamedFunction(fullName+"Wasm").IsNil() || pkg.Module().NamedFunction(fullName).IsNil() {
					t.Fatalf("native instrumentation must retain its existing ABI:\n%s", ir)
				}
			}
			if test.target.GOARCH == "wasm" && strings.Contains(ir, "alloca") {
				t.Fatalf("Wasm caller-location instrumentation reserved stack arguments:\n%s", ir)
			}
		})
	}
}
