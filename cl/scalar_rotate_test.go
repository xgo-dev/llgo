package cl

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

const rotateDeclaration = `package rotates
type Integer interface { ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 }
//llgo:link rotate llgo.rotateLeft
func rotate[T Integer](x T, k int) T { panic("intrinsic") }
`

const rotateSource = rotateDeclaration + `
func r8(x uint8, k int) uint8 { return rotate(x, k) }
func r16(x uint16, k int) uint16 { return rotate(x, k) }
func r32(x uint32, k int) uint32 { return rotate(x, k) }
func r64(x uint64, k int) uint64 { return rotate(x, k) }
func word(x uint, k int) uint { return rotate(x, k) }
`

func rotateFunctionIR(t *testing.T, ir, name string) string {
	t.Helper()
	body := regexp.MustCompile(`(?ms)^define[^\n]*@rotates\.` + name + `\(.*?^}`).FindString(ir)
	if body == "" {
		t.Fatalf("missing function %s:\n%s", name, ir)
	}
	return body
}

func verifyRotateIR(t *testing.T, ir string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rotate.ll")
	if err := os.WriteFile(path, []byte(ir), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Dispose()
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestRotateLeftIntrinsic(t *testing.T) {
	for _, target := range []llssa.Target{
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, rotateSource)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			ir := pkg.String()
			verifyRotateIR(t, ir)
			word := int(prog.SizeOf(prog.Uintptr())) * 8
			for _, test := range []struct {
				name  string
				width int
			}{{"r8", 8}, {"r16", 16}, {"r32", 32}, {"r64", 64}, {"word", word}} {
				body := rotateFunctionIR(t, ir, test.name)
				want := fmt.Sprintf("call i%d @llvm.fshl.i%d(", test.width, test.width)
				if !strings.Contains(body, want) || strings.Contains(body, "math/bits.RotateLeft") {
					t.Errorf("%s does not lower to %s:\n%s", test.name, want, body)
				}
			}
		})
	}
}

func TestMathRotateLeftExplicitRewrite(t *testing.T) {
	const src = `package rotates
import "math/bits"
func r32(x uint32, k int) uint32 { return bits.RotateLeft32(x, k) }
`
	pkg, _, files := buildPackageWithFiles(t, src)
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	prog.SetLinkname("math/bits.RotateLeft32", "other.rotate")
	compiled, _, err := NewPackageEx(prog, nil, nil, pkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := compiled.String()
	verifyRotateIR(t, ir)
	body := rotateFunctionIR(t, ir, "r32")
	if !strings.Contains(body, "@other.rotate(") || strings.Contains(body, "@llvm.fshl") {
		t.Fatalf("explicit callee rewrite was bypassed:\n%s", body)
	}
}

func TestRotateLeftIntrinsicCallForms(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"function-value", "func indirect() func(uint32, int) uint32 { return rotate[uint32] }"},
		{"defer", "func later(x uint32, k int) { defer rotate(x, k) }"},
		{"go", "func concurrent(x uint32, k int) { go rotate(x, k) }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Compile each form alone so another call cannot supply its funnel shift.
			ir := compileWithRewrites(t, rotateDeclaration+tc.source, nil)
			verifyRotateIR(t, ir)
			if !strings.Contains(ir, "call i32 @llvm.fshl.i32(") || strings.Contains(ir, "@llgo.rotateLeft") {
				t.Fatalf("intrinsic call form must emit a funnel shift without an external llgo symbol:\n%s", ir)
			}
		})
	}
}

func TestRotateLeftInvalidDeclarations(t *testing.T) {
	for _, tc := range []struct{ name, decl, call, want string }{
		{"arity", "func rotate(uint) uint", "rotate(x)", "invalid arguments or result"},
		{"result-count", "func rotate(uint, int)", "rotate(x, 1)", "invalid arguments or result"},
		{"result-type", "func rotate(uint, int) uint64", "rotate(x, 1)", "invalid arguments or result"},
		{"float-value", "func rotate(float64, int) float64", "rotate(1, 1)", "expected scalar integer arguments"},
		{"float-count", "func rotate(uint, float64) uint", "rotate(x, 1)", "expected scalar integer arguments"},
		{"aggregate", "func rotate([1]uint, int) [1]uint", "rotate([1]uint{x}, 1)", "expected scalar integer arguments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package rotates\n//llgo:link rotate llgo.rotateLeft\n" + tc.decl + "\nfunc bad(x uint) { " + tc.call + " }"
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			defer func() {
				if got := fmt.Sprint(recover()); !strings.Contains(got, tc.want) {
					t.Fatalf("diagnostic = %q, want %q", got, tc.want)
				}
			}()
			if _, err := NewPackage(prog, ssapkg, files); err != nil {
				panic(err)
			}
		})
	}
}
