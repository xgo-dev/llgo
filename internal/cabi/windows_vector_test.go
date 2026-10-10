package cabi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestWindowsVectorParameterABI(t *testing.T) {
	const ir = `
declare <4 x float> @external(<4 x float>)
declare fastcc <4 x float> @fast(<4 x float>)
declare <4 x float> @llvm.sqrt.v4f32(<4 x float>)
define <4 x float> @forward(<4 x float> %x, ptr %indirect) {
  %a = call <4 x float> @external(<4 x float> %x)
  %b = call <4 x float> %indirect(<4 x float> %a)
  %c = call fastcc <4 x float> @fast(<4 x float> %b)
  %d = call <4 x float> @llvm.sqrt.v4f32(<4 x float> %c)
  ret <4 x float> %d
}
`
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, goos := range []string{"windows", "linux"} {
		t.Run(goos, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "vector.ll")
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
			prog := ssa.NewProgram(&ssa.Target{GOOS: goos, GOARCH: "amd64"})
			defer prog.Dispose()
			tr := NewTransformer(prog, "", "", true)
			tr.LowerWindowsVectorParams(mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			body := mod.NamedFunction("forward").String()
			if goos == "windows" {
				if mod.NamedFunction("forward").Param(0).Type().TypeKind() != llvm.PointerTypeKind ||
					!strings.Contains(body, "load <4 x float>") || !strings.Contains(body, "call <4 x float> @external(ptr") ||
					!strings.Contains(body, "call <4 x float> %1(ptr") {
					t.Fatalf("indirect vector ABI not exposed:\n%s", body)
				}
			} else if mod.NamedFunction("forward").Param(0).Type().TypeKind() != llvm.VectorTypeKind {
				t.Fatalf("changed SysV vector ABI:\n%s", body)
			}
			for _, name := range []string{"fast", "llvm.sqrt.v4f32"} {
				if mod.NamedFunction(name).Param(0).Type().TypeKind() != llvm.VectorTypeKind {
					t.Fatalf("changed non-C ABI: %s", name)
				}
			}
			if mod.NamedFunction("forward").GlobalValueType().ReturnType().TypeKind() != llvm.VectorTypeKind {
				t.Fatal("changed vector return ABI")
			}
		})
	}
}
