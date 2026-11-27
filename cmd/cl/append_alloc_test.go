package cl

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/ssa"
)

func TestAppendTemporaryStorage(t *testing.T) {
	const src = `package appendalloc
type Item struct { P *int; A, B uint64 }
type encoder struct{}
func (encoder) AppendWord(s []byte, v uint32) []byte { return append(s, byte(v), byte(v>>8), byte(v>>16), byte(v>>24)) }
func (*encoder) appendPointer(s []*int, v *int) []*int { return append(s, v) }
func (*encoder) escaped(v int) *int { return &v }
func appendUint64(s []byte, v uint64) []byte { return append(s, byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56)) }
func appendString(s []byte, v string) []byte { return append(s, v...) }
func appendEmptyString(s []byte) []byte { return append(s, ""...) }
func appendNil(v int) []int { return append([]int(nil), v) }
func scalar(s []Item, v Item) []Item { return append(s, v) }
func several(s []int, x, y int) []int { return append(s, x, y) }
func loop(s []int, n int) []int { for i:=0; i<n; i++ { s=append(s,i) }; return s }
func spread(s, v []int) []int { return append(s, v...) }
func save(v ...int) []int { return v }
func escaped(v int) []int { return save(v) }
func saveTwo(_ int, v ...int) []int { return v }
func escapedTwo(v int) []int { return saveTwo(0,v) }
var retained *int
func addressEscapes(s []int,v int) []int { retained=&v; return append(s,v) }
func zero(s []struct{}) []struct{} { return append(s, struct{}{}) }
`
	for _, target := range []llvmgen.Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
				ir := compileWithRewritesModeTarget(t, src, nil, mode, &target)
				for _, name := range []string{"appendUint64", "appendString", "scalar", "several", "loop", "spread"} {
					body := llvmFunction(t, ir, "appendalloc."+name)
					if strings.Contains(body, "runtime.Alloc") {
						t.Errorf("%s retains temporary heap allocation:\n%s", name, body)
					}
					if !strings.Contains(body, "@llvm.memmove.") || !strings.Contains(body, "runtime.SliceAppend") {
						t.Errorf("%s misses inline memmove fast path or SliceAppend growth fallback:\n%s", name, body)
					}
				}
				if body := llvmFunction(t, ir, "appendalloc.zero"); strings.Contains(body, "runtime.Alloc") || strings.Contains(body, "@llvm.memmove.") || !strings.Contains(body, "runtime.SliceAppend") {
					t.Errorf("zero-sized append should skip memmove and keep SliceAppend fallback:\n%s", body)
				}
				if body := llvmFunction(t, ir, "appendalloc.appendNil"); strings.Contains(body, "@llvm.memmove.") || !strings.Contains(body, "runtime.SliceAppend") {
					t.Errorf("append to nil slice should skip dead spare-capacity fast path:\n%s", body)
				}
				if body := llvmFunction(t, ir, "appendalloc.appendEmptyString"); strings.Contains(body, "@llvm.memmove.") || strings.Contains(body, "runtime.SliceAppend") {
					t.Errorf("append of empty string literal should be a no-op:\n%s", body)
				}
				for _, name := range []string{"encoder.AppendWord", "(*encoder).appendPointer"} {
					body := llvmFunction(t, ir, "appendalloc."+name)
					if strings.Contains(body, "runtime.Alloc") {
						t.Errorf("%s (method on unexported type) retains temporary heap allocation:\n%s", name, body)
					}
					if !strings.Contains(body, "@llvm.memmove.") || !strings.Contains(body, "runtime.SliceAppend") {
						t.Errorf("%s misses inline memmove fast path or SliceAppend growth fallback:\n%s", name, body)
					}
				}
				for _, name := range []string{"escaped", "escapedTwo", "addressEscapes", "(*encoder).escaped"} {
					if body := llvmFunction(t, ir, "appendalloc."+name); !strings.Contains(body, "runtime.Alloc") {
						t.Fatalf("%s lost escaping storage:\n%s", name, body)
					}
				}
			}
		})
	}
}

func TestAppendGrowthRecordsCallerLocation(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/appendgrow", `package appendgrow
import "runtime/debug"

func grow(dst, src []byte) []byte {
	_ = debug.Stack()
	word := src[1:5]
//line append_growth.go:267
	return append(dst, word...)
}
`)
	prog := newLLSSAProgForTarget(t, &llvmgen.Target{GOOS: "linux", GOARCH: "amd64"})
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg, _, err := NewPackageExWithEmbedMetaOptions(
		prog, nil, nil, nil, ssapkg, files, nil, false, Options{ShadowStack: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	body := llvmFunction(t, pkg.Module().String(), "example.com/appendgrow.grow")
	foundGrowth := false
	for _, block := range strings.Split(body, "\n\n") {
		if strings.Contains(block, "@llvm.memmove.") && strings.Contains(block, "RecordPanicLocation") {
			t.Fatalf("fast-path block should not record panic location:\n%s", block)
		}
		beforeGrowth, _, ok := strings.Cut(block, "runtime.SliceAppend")
		if !ok {
			continue
		}
		foundGrowth = true
		if !strings.Contains(beforeGrowth, "RecordPanicLocation") ||
			!strings.Contains(beforeGrowth, "__llgo_pcsite_") ||
			!strings.Contains(beforeGrowth, "i64 267") {
			t.Fatalf("growth call misses its source location:\n%s", block)
		}
	}
	if !foundGrowth {
		t.Fatalf("missing growth call:\n%s", body)
	}
}
