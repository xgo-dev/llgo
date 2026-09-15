//go:build !llgo

package cl

import (
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestWasmCallerLocationScalarABI(t *testing.T) {
	for _, test := range []struct {
		name   string
		target llssa.Target
	}{
		{"GJS", llssa.Target{GOOS: "js", GOARCH: "wasm"}},
		{"GWASI", llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}},
		{"EC64", llssa.Target{GOOS: "js", GOARCH: "wasm", Target: "emscripten-memory64", LLVMTarget: "wasm64-unknown-emscripten"}},
		{"native", llssa.Target{GOOS: "linux", GOARCH: "amd64"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &test.target)
			if test.name == "EC64" {
				prog.TypeSizes(types.SizesFor("gc", "amd64"))
			}
			pkg := prog.NewPackage("foo", "example.com/foo")
			fn := pkg.NewFunc("example.com/foo.f", llssa.NoArgsNoRet, llssa.InGo)
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
				fullName := llssa.PkgRuntime + "." + name
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
