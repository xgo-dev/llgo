package cabi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestFunctionAlignmentSurvivesLowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	const input = `
%Large = type { i64, i64, i64 }
declare %Large @external(%Large) align 64
declare %Large @ordinary_external(%Large)
define %Large @aligned(%Large %x) align 128 { ret %Large %x }
define %Large @ordinary(%Large %x) { ret %Large %x }
define i64 @scalar(i64 %x) align 64 { ret i64 %x }
define %Large @caller() {
 %x = call %Large @external(%Large zeroinitializer)
 %y = call %Large @aligned(%Large %x)
 ret %Large %y
}
`
	for _, target := range []ssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "riscv64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := ssa.NewProgram(&target)
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "alignment.ll")
			if err := os.WriteFile(path, []byte(input), 0644); err != nil {
				t.Fatal(err)
			}
			buf, err := llvm.NewMemoryBufferFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			m, err := ctx.ParseIR(buf)
			if err != nil {
				t.Fatal(err)
			}
			defer m.Dispose()
			before := m.NamedFunction("aligned").GlobalValueType().String()
			NewTransformer(prog, target.Spec().Triple, "", true).TransformModule("test", m)
			if m.NamedFunction("aligned").GlobalValueType().String() == before {
				t.Fatal("test did not exercise replacement")
			}
			for _, test := range []struct {
				name string
				want int
			}{
				{"external", 64},
				{"ordinary_external", 0},
				{"aligned", 128},
				{"ordinary", 0},
				// The scalar control keeps its original function. Aggregate
				// entries above exercise alignment on replacement functions.
				{"scalar", 64},
			} {
				if got := m.NamedFunction(test.name).Alignment(); got != test.want {
					t.Errorf("%s alignment = %d, want %d", test.name, got, test.want)
				}
			}
			if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
