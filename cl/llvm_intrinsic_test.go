package cl

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestLLVMIntrinsicLinks(t *testing.T) {
	const src = `package intrinsics
import _ "unsafe"
type Byte uint8
//llgo:link funnel llvm.fshl
func funnel(a, b, k Byte) Byte
//go:linkname population llvm.ctpop.i32
func population(x uint32) uint32
//llgo:link leading llvm.ctlz
func leading(x uint64, zeroPoison bool) uint64
//llgo:link root llvm.sqrt
func root(x float64) float64
//llgo:link overflow llvm.umul.with.overflow
func overflow(a, b uint64) (uint64, bool)
//llgo:link word llvm.ctpop
func word(x uintptr) uintptr
//llgo:link signed llvm.sadd.sat
func signed(x, y int32) int32
//llgo:link saturate llvm.fptosi.sat
func saturate(x float64) int32
func Rotate(x Byte, k int) Byte { return funnel(x, x, Byte(k)) }
func UseRotate(x Byte, k int) Byte { return Rotate(x, k) }
//go:noinline
func KeepRotate(x Byte, k int) Byte { return funnel(x, x, Byte(k)) }
func UseKeepRotate(x Byte, k int) Byte { return KeepRotate(x, k) }
func Population(x uint32) uint32 { return population(x) }
func Leading(x uint64) uint64 { return leading(x, false) }
func Root(x float64) float64 { return root(x) }
func Overflow(x, y uint64) (uint64, bool) { return overflow(x, y) }
func Word(x uintptr) uintptr { return word(x) }
func Signed(x, y int32) int32 { return signed(x, y) }
func Saturate(x float64) int32 { return saturate(x) }
var RotateValue = Rotate
var OverflowValue = Overflow
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "7"},
		{GOOS: "linux", GOARCH: "riscv64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32", WasmProvider: "emscripten"},
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
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%v\n%s", err, mod.String())
			}
			for fn, intrinsic := range map[string]string{
				"Rotate": "llvm.fshl.i8", "UseRotate": "llvm.fshl.i8", "Population": "llvm.ctpop.i32",
				"Leading": "llvm.ctlz.i64", "Root": "llvm.sqrt.f64",
				"Overflow": "llvm.umul.with.overflow.i64",
				"Word":     "llvm.ctpop.i" + strconv.Itoa(prog.GoWordSize()*8),
				"Signed":   "llvm.sadd.sat.i32", "Saturate": "llvm.fptosi.sat.i32.f64",
			} {
				ir := mustNamedFunction(t, mod, "intrinsics."+fn).String()
				if strings.Count(ir, "@"+intrinsic+"(") != 1 {
					t.Fatalf("%s: expected one direct %s call:\n%s", fn, intrinsic, ir)
				}
			}
			if ir := mustNamedFunction(t, mod, "intrinsics.UseKeepRotate").String(); !strings.Contains(ir, "@intrinsics.KeepRotate(") {
				t.Fatalf("wrapper expansion ignored go:noinline:\n%s", ir)
			}
			if !mod.NamedFunction("intrinsics.funnel").IsNil() || !mod.NamedFunction("check").IsNil() {
				t.Fatal("intrinsic binding or validation function escaped into output")
			}
			mod.SetDataLayout(prog.DataLayout())
			mod.SetTarget(target.Spec().Triple)
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			opts.SetVerifyEach(true)
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			buf, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			buf.Dispose()
		})
	}
}

func TestLLVMIntrinsicWrapperEvaluatesArgumentsOnce(t *testing.T) {
	const src = `package adapter
//llgo:link fshl llvm.fshl
func fshl(a, b, k uint8) uint8
func rotate(x uint8, k int) uint8 { return fshl(x, x, uint8(k)) }
var counter uint8
func next() uint8 { counter++; return counter }
func Use() uint8 { return rotate(next(), 3) }
`
	ssapkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	defer pkg.Module().Dispose()
	ir := mustNamedFunction(t, pkg.Module(), "adapter.Use").String()
	if strings.Count(ir, "@adapter.next(") != 1 || strings.Contains(ir, "@adapter.rotate(") || !strings.Contains(ir, "@llvm.fshl.i8(") {
		t.Fatalf("wrapper did not preserve single evaluation:\n%s", ir)
	}
}

func TestLLVMIntrinsicLinkErrors(t *testing.T) {
	for _, tc := range []struct{ name, target, decl, use, want string }{
		{"unknown-unused", "llvm.not.an.intrinsic", "func f(x uint8) uint8", "", "unknown LLVM intrinsic"},
		{"bad-suffix", "llvm.fshl.invalid", "func f(a, b, k uint8) uint8", "", `expected "llvm.fshl.i8"`},
		{"wrong-suffix", "llvm.fshl.i32", "func f(a, b, k uint8) uint8", "", `expected "llvm.fshl.i8"`},
		{"missing-arg", "llvm.fshl", "func f(x uint8, k int) uint8", "", "does not match"},
		{"extra-arg", "llvm.fshl", "func f(a, b, k, extra uint8) uint8", "", "does not match"},
		{"wrong-arg", "llvm.fshl", "func f(a, b uint8, k int) uint8", "", "does not match"},
		{"wrong-result", "llvm.fshl", "func f(a, b, k uint8) uint32", "", "does not match"},
		{"no-result", "llvm.ctpop", "func f(x uint32)", "", "does not match"},
		{"wrong-tuple", "llvm.umul.with.overflow", "func f(a, b uint64) (uint64, uint8)", "", "does not match"},
		{"bad-bswap-width", "llvm.bswap", "func f(x uint8) uint8", "", "bswap must be an even number of bytes"},
		{"non-immediate", "llvm.ctlz", "func f(x uint32, z bool) uint32", "func use(x uint32, z bool) uint32 { return f(x, z) }", "argument 2 must be an immediate constant"},
		{"variadic", "llvm.ctpop", "func f(x ...uint32) uint32", "", "non-variadic"},
		{"pointer", "llvm.ctpop", "func f(x *uint32) uint32", "", "unsupported type"},
		{"struct", "llvm.ctpop", "func f(x struct{ N int }) int", "", "unsupported type"},
		{"complex", "llvm.sqrt", "func f(x complex128) complex128", "", "unsupported type"},
		{"generic", "llvm.ctpop", "func f[T ~uint32](x T) T { return x }", "", "non-generic"},
		{"environment", "llvm.ctpop", "//llgo:env\nfunc f(x uint32) uint32 { return x }", "", "do not support //llgo:env"},
		{"target-specific", "llvm.x86.sse2.pause", "func f()", "", "only portable scalar math"},
		{"memory", "llvm.memcpy", "func f()", "", "only portable scalar math"},
		{"function-value", "llvm.ctpop", "func f(x uint32) uint32", "var Value = f", "cannot be used as a function value"},
		{"go", "llvm.ctpop", "func f(x uint32) uint32", "func use() { go f(1) }", "use a Go wrapper for go or defer"},
		{"defer", "llvm.ctpop", "func f(x uint32) uint32", "func use() { defer f(1) }", "use a Go wrapper for go or defer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package invalid\n//llgo:link f " + tc.target + "\n" + tc.decl + "\n" + tc.use
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			defer func() {
				got := fmt.Sprint(recover())
				if !strings.Contains(got, tc.want) || !strings.Contains(got, "foo.go:") {
					t.Fatalf("diagnostic = %q, want source location and %q", got, tc.want)
				}
			}()
			_, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				panic(err)
			}
		})
	}
}
