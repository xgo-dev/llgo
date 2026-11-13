package cabi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/heap2stack"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestHeapToStackBeforeABI(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	const input = `
%Large = type { i64, i64, i64 }
declare nonnull ptr @alloc(i64) allocsize(0)
define %Large @get(ptr %p) noinline {
  %v = load i64, ptr %p
  %r1 = insertvalue %Large zeroinitializer, i64 %v, 0
  %r2 = insertvalue %Large %r1, i64 %v, 1
  %r3 = insertvalue %Large %r2, i64 %v, 2
  ret %Large %r3
}
define i64 @caller(i64 %v) {
  %p = call ptr @alloc(i64 8) allockind("alloc,uninitialized")
  store i64 %v, ptr %p
  %s = call %Large @get(ptr %p)
  %r = extractvalue %Large %s, 1
  ret i64 %r
}
define i64 @byvalue(%Large %v) noinline memory(none) {
  %r = extractvalue %Large %v, 1
  ret i64 %r
}
define i64 @scalar(i64 %v) memory(none) { ret i64 %v }
define { i64 } @registers({ i64 } %v) memory(none) { ret { i64 } %v }
`
	for _, target := range []ssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := ssa.NewProgram(&target)
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "before-abi.ll")
			if err := os.WriteFile(path, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			buf, err := llvm.NewMemoryBufferFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			mod, err := ctx.ParseIR(buf)
			if err != nil {
				t.Fatal(err)
			}
			defer mod.Dispose()
			mod.SetTarget(target.Spec().Triple)
			mod.SetDataLayout(prog.DataLayout())
			if err := heap2stack.Run(mod, prog.TargetMachine()); err != nil {
				t.Fatal(err)
			}
			memory := llvm.AttributeKindID("memory")
			if mod.NamedFunction("get").GetEnumFunctionAttribute(memory).IsNil() {
				t.Fatal("fixture did not infer the pre-ABI memory effects")
			}
			NewTransformer(prog, target.Spec().Triple, "", true).TransformModule("test", mod)
			for _, name := range []string{"get", "byvalue"} {
				if !mod.NamedFunction(name).GetEnumFunctionAttribute(memory).IsNil() {
					t.Errorf("%s retained pre-ABI memory effects after introducing indirect storage:\n%s", name, mod.NamedFunction(name))
				}
			}
			for _, name := range []string{"scalar", "registers"} {
				if mod.NamedFunction(name).GetEnumFunctionAttribute(memory).IsNil() {
					t.Errorf("%s lost memory effects without introducing indirect storage", name)
				}
			}
			pbo := llvm.NewPassBuilderOptions()
			defer pbo.Dispose()
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), pbo); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			body := mod.NamedFunction("caller").String()
			if strings.Contains(body, "ret i64 undef") || strings.Contains(body, "ret i64 poison") || !strings.Contains(body, "@get(") {
				t.Fatalf("ABI result store was optimized away:\n%s", body)
			}
		})
	}
}
