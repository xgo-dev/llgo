//go:build !llgo

package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestFloatMinMaxLowering(t *testing.T) {
	const src = `package minmax
 type Float float64
 func Min32(a, b, c float32) float32 { return min(a, b, c) }
 func Max32(a, b, c float32) float32 { return max(a, b, c) }
 func Min64(a, b, c Float) Float { return min(a, b, c) }
 func Max64(a, b, c Float) Float { return max(a, b, c) }
 func Identity(a Float) Float { return min(a) }
 func Integer(a, b int) int { return min(a, b) }
 `
	for _, tt := range []struct {
		target    llssa.Target
		intrinsic bool
	}{
		{llssa.Target{GOOS: "linux", GOARCH: "arm64"}, true},
		{llssa.Target{GOOS: "windows", GOARCH: "amd64"}, true},
		{llssa.Target{GOOS: "windows", GOARCH: "386"}, true},
		{llssa.Target{GOOS: "windows", GOARCH: "386", GO386: "softfloat"}, false},
		{llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}, true},
		{llssa.Target{GOOS: "js", GOARCH: "wasm", Target: "emscripten", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"}, true},
		{llssa.Target{GOOS: "linux", GOARCH: "arm", GOARM: "7"}, false},
		{llssa.Target{GOOS: "linux", GOARCH: "arm", GOARM: "5"}, false},
		{llssa.Target{GOOS: "linux", GOARCH: "riscv64"}, false},
		{llssa.Target{GOOS: "linux", GOARCH: "arm64", Target: "custom"}, false},
	} {
		target := tt.target
		name := strings.Join([]string{target.GOARCH, target.GO386, target.GOARM, target.Target}, "-")
		t.Run(strings.TrimRight(name, "-"), func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, width := range []string{"32", "64"} {
				for _, op := range []struct{ name, intrinsic string }{{"Min", "minimum"}, {"Max", "maximum"}} {
					ir := mustNamedFunction(t, pkg.Module(), "minmax."+op.name+width).String()
					want := "@llvm." + op.intrinsic + ".f" + width + "("
					calls := 0
					if tt.intrinsic {
						calls = 2
					}
					if strings.Count(ir, want) != calls {
						t.Errorf("want %d %s calls:\n%s", calls, want, ir)
					}
					if strings.Contains(ir, "fcmp") || (tt.intrinsic && strings.Contains(ir, "select")) {
						t.Errorf("floating min/max still uses compare/select:\n%s", ir)
					}
				}
			}
			ir := mustNamedFunction(t, pkg.Module(), "minmax.Identity").String()
			if strings.Contains(ir, "call") {
				t.Errorf("one operand must remain unchanged:\n%s", ir)
			}
			ir = mustNamedFunction(t, pkg.Module(), "minmax.Integer").String()
			if !strings.Contains(ir, "icmp slt") || strings.Contains(ir, "@llvm.minimum") {
				t.Errorf("integer lowering changed:\n%s", ir)
			}
			mod := pkg.Module()
			mod.SetDataLayout(prog.DataLayout())
			mod.SetTarget(target.Spec().Triple)
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			opts.SetVerifyEach(true)
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			asm, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			defer asm.Dispose()
			if text := string(asm.Bytes()); strings.Contains(text, "fminimum") || strings.Contains(text, "fmaximum") {
				t.Fatalf("min/max requires unavailable C23 libcalls:\n%s", text)
			}
		})
	}
}
