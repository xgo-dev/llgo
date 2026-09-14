//go:build !llgo

package cl

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestUMulOverflowLowering(t *testing.T) {
	const src = `package overflow
type Unsigned interface { ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint | ~uintptr }
//llgo:link multiply llgo.umulOverflow
func multiply[T Unsigned](a, b T) (T, bool) { panic("intrinsic") }
type Word uint64
func Mul8(a, b uint8) (uint8, bool) { return multiply(a, b) }
func Mul16(a, b uint16) (uint16, bool) { return multiply(a, b) }
func Mul32(a, b uint32) (uint32, bool) { return multiply(a, b) }
func Mul64(a, b Word) (Word, bool) { return multiply(a, b) }
func MulPtr(a, b uintptr) (uintptr, bool) { return multiply(a, b) }
var multiplyValue = multiply[Word]
func Value64(a, b Word) (Word, bool) { return multiplyValue(a, b) }
func Defer64(a, b Word) { defer multiply(a, b) }
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "386", GO386: "softfloat"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "5"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "7"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "linux", GOARCH: "riscv64"},
	} {
		t.Run(target.GOARCH+target.GO386+target.GOARM, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, suffix := range []string{"8", "16", "32", "64", "Ptr"} {
				width := suffix
				if suffix == "Ptr" {
					width = strconv.Itoa(int(prog.SizeOf(prog.Uintptr())) * 8)
				}
				ir := mustNamedFunction(t, mod, "overflow.Mul"+suffix).String()
				if strings.Count(ir, "@llvm.umul.with.overflow.i"+width+"(") != 1 {
					t.Errorf("expected one unsigned multiply-with-overflow call:\n%s", ir)
				}
				if strings.Contains(ir, " udiv ") || strings.Contains(ir, "AssertDivideByZero") || strings.Contains(ir, "br i1") {
					t.Errorf("unsigned overflow introduced division or a conditional branch:\n%s", ir)
				}
			}
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
			asm.Dispose()
		})
	}
}

func TestUMulOverflowInvalidDeclarations(t *testing.T) {
	for _, tc := range []struct{ name, decl, call, want string }{
		{"arity", "func mul(a uint) (uint, bool)", "mul(a)", "invalid arguments"},
		{"result-count", "func mul(a, b uint) uint", "mul(a, a)", "invalid arguments"},
		{"product-type", "func mul(a, b uint) (uint64, bool)", "mul(a, a)", "invalid arguments"},
		{"flag-type", "func mul(a, b uint) (uint, uint)", "mul(a, a)", "invalid arguments"},
		{"signed", "func mul(a, b int) (int, bool)", "mul(int(a), int(a))", "same unsigned integer type"},
		{"mismatched-operands", "func mul(a uint, b uint64) (uint, bool)", "mul(a, uint64(a))", "same unsigned integer type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package overflow\n//llgo:link mul llgo.umulOverflow\n" + tc.decl + "\nfunc bad(a uint) { " + tc.call + " }"
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			defer func() {
				if got := fmt.Sprint(recover()); !strings.Contains(got, tc.want) {
					t.Fatalf("diagnostic = %q, want %q", got, tc.want)
				}
			}()
			_, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				panic(err)
			}
		})
	}
}
