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

const rotateSource = `package rotates
import "math/bits"
func r8(x uint8, k int) uint8 { return bits.RotateLeft8(x, k) }
func r16(x uint16, k int) uint16 { return bits.RotateLeft16(x, k) }
func r32(x uint32, k int) uint32 { return bits.RotateLeft32(x, k) }
func r64(x uint64, k int) uint64 { return bits.RotateLeft64(x, k) }
func word(x uint, k int) uint { return bits.RotateLeft(x, k) }
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

func TestMathRotateLeftLowerAtCaller(t *testing.T) {
	for _, target := range []llssa.Target{
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, rotateSource, nil, &target)
			verifyRotateIR(t, ir)
			word := 64
			if target.GOARCH == "386" || target.GOARCH == "wasm" {
				word = 32
			}
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
	pkg, _, files := buildPackageWithFiles(t, rotateSource)
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

func TestMathRotateLeftCallForms(t *testing.T) {
	const src = `package rotates
import "math/bits"
func indirect() func(uint32, int) uint32 { return bits.RotateLeft32 }
func later(x uint32, k int) { defer bits.RotateLeft32(x, k) }
func concurrent(x uint32, k int) { go bits.RotateLeft32(x, k) }
`
	ir := compileWithRewrites(t, src, nil)
	verifyRotateIR(t, ir)
	if strings.Contains(ir, "@llvm.fshl") || !strings.Contains(ir, "math/bits.RotateLeft32") {
		t.Fatalf("non-direct calls must retain the real library symbol:\n%s", ir)
	}
}
