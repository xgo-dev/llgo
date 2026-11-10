package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestFreshNilAppendStorage(t *testing.T) {
	const src = `package fresh
type Bytes []byte
func dynamic(n int) []byte { return append([]byte(nil), make([]byte, n)...) }
func constant() []byte { return append([]byte(nil), make([]byte, 4)...) }
func named(n int) Bytes { return append(Bytes(nil), make(Bytes, n)...) }
func pointers(n int) []*int { return append([]*int(nil), make([]*int, n)...) }
func zero(n int) []struct{} { return append([]struct{}(nil), make([]struct{}, n)...) }
func nonnil(dst []byte, n int) []byte { return append(dst, make([]byte,n)...) }
func capacity(n,c int) []byte { return append([]byte(nil),make([]byte,n,c)...) }
func shared(n int) []byte { s:=make([]byte,n);r:=append([]byte(nil),s...);s[0]=1;return r }
func literal() []byte { return append([]byte(nil), []byte{0,0,0,0}...) }
`
	for _, target := range []llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasip1", WasmProfile: "w32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, &target, target.GOARCH == "wasm")
			for _, name := range []string{"dynamic", "constant", "named", "pointers", "zero"} {
				body := llvmFunction(t, ir, "fresh."+name)
				if strings.Contains(body, "SliceAppend") || strings.Contains(body, "MakeSliceBuffer") {
					t.Errorf("%s retains append copying or returns temporary storage:\n%s", name, body)
				}
			}
			if body := llvmFunction(t, ir, "fresh.constant"); strings.Count(body, "runtime.AllocAtomicU\"") != 1 {
				t.Errorf("constant append must retain one escaping backing array:\n%s", body)
			}
			for _, name := range []string{"nonnil", "capacity", "shared", "literal"} {
				if body := llvmFunction(t, ir, "fresh."+name); !strings.Contains(body, "SliceAppend") {
					t.Errorf("%s incorrectly forwarded its source backing array:\n%s", name, body)
				}
			}
		})
	}
}
