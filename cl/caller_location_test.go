package cl

import (
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestRuntimeLocationRecordDeduplication(t *testing.T) {
	ssapkg, _ := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"
func f() { runtime.Caller(0) }
`)
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "js", GOARCH: "wasm", Target: "emscripten-memory64", LLVMTarget: "wasm64-unknown-emscripten"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.Target, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &target)
			if target.Target == "emscripten-memory64" {
				prog.TypeSizes(types.SizesFor("gc", "amd64"))
			}
			pkg := prog.NewPackage("foo", "example.com/foo")
			fn := pkg.NewFunc("example.com/foo.f", llssa.NoArgsNoRet, llssa.GoDecl)
			b := fn.MakeBody(1)
			ctx := &context{
				prog: prog, pkg: pkg, fn: fn, goFn: ssapkg.Func("f"), fset: ssapkg.Prog.Fset,
				options: Options{ShadowStack: true}, trackCallerFrames: true,
				runtimeCallerFuncs: runtimeCallerFuncSet(NewCallerTracking(prog.Directives()), ssapkg),
			}
			pos := ctx.goFn.Pos()
			ctx.recordPanicSite(b, pos)
			ctx.recordPanicSite(b, pos)
			ctx.recordCallerLocation(b, pos)
			ctx.recordCallerLocation(b, pos)
			// A lowering-generated call must break reuse even though it did not
			// pass through the frontend's ordinary Go-call instrumentation.
			b.Call(pkg.NewFunc("helper", llssa.NoArgsNoRet, llssa.GoDecl).Expr)
			ctx.recordCallerLocation(b, pos)
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			counts := make(map[string]int)
			for block := pkg.Module().NamedFunction(fn.Name()).FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
				for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if inst.InstructionOpcode() == llvm.Call && !inst.CalledValue().IsAFunction().IsNil() {
						counts[strings.TrimSuffix(inst.CalledValue().Name(), "Wasm")]++
					}
				}
			}
			for name, want := range map[string]int{"RecordPanicLocation": 1, "RecordCallerLocation": 2} {
				if got := counts[llssa.PkgRuntime+"."+name]; got != want {
					t.Fatalf("%s calls = %d, want %d:\n%s", name, got, want, pkg.Module().String())
				}
			}
		})
	}
}
