package cabi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestSIMD256CallCarriers(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	const ir = `
declare <8 x float> @external(<8 x float>)
declare <8 x float> @llvm.sqrt.v8f32(<8 x float>)
define {<8 x float>, i64} @forward(<8 x float> %x, ptr %indirect) {
 %a = call <8 x float> @external(<8 x float> %x)
 %b = call <8 x float> %indirect(<8 x float> %a)
 %c = call <8 x float> @llvm.sqrt.v8f32(<8 x float> %b)
 %r = insertvalue {<8 x float>, i64} zeroinitializer, <8 x float> %c, 0
 ret {<8 x float>, i64} %r
}
`
	for _, goos := range []string{"linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			prog := ssa.NewProgram(&ssa.Target{GOOS: goos, GOARCH: "amd64"})
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "wide.ll")
			if err := os.WriteFile(path, []byte(ir), 0600); err != nil {
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
			tr := NewTransformer(prog, "", "", true)
			tr.LowerWideVectorCalls(mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			body := mod.NamedFunction("forward").String()
			for _, want := range []string{"sret({ <8 x float>, i64 })", "load volatile <8 x float>", "store volatile { <8 x float>, i64 }", "call void @external(ptr sret(<8 x float>)", "call void %2(ptr sret(<8 x float>)"} {
				if !strings.Contains(body, want) {
					t.Fatalf("missing %q:\n%s", want, body)
				}
			}
			intrinsic := mod.NamedFunction("llvm.sqrt.v8f32")
			if intrinsic.GlobalValueType().ReturnType().TypeKind() != llvm.VectorTypeKind || intrinsic.Param(0).Type().TypeKind() != llvm.VectorTypeKind {
				t.Fatal("changed LLVM intrinsic ABI")
			}
			before := mod.String()
			tr.LowerWideVectorCalls(mod)
			if mod.String() != before {
				t.Fatal("wide ABI lowering is not idempotent")
			}
		})
	}
}
