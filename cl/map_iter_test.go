package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestMapRangeStackStorage(t *testing.T) {
	const src = `package iteration
func sum(m map[int]*int) int {
	var n int
	for _, p := range m { n += *p }
	return n
}
func nested(m map[int]*int) int {
	var n int
	for _, p := range m { for _, q := range m { n += *p + *q } }
	return n
}
func repeated(maps []map[int]*int) int {
	var n int
	for _, m := range maps { for _, p := range m { n += *p } }
	return n
}
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
			roots := target.GOARCH == "wasm"
			ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, &target, roots)
			for name, count := range map[string]int{"sum": 1, "nested": 2, "repeated": 1} {
				body := llvmFunction(t, ir, "iteration."+name)
				if strings.Contains(body, "runtime.Alloc") || strings.Contains(body, "NewMapIter") {
					t.Fatalf("%s retains a heap iterator:\n%s", name, body)
				}
				if got := strings.Count(body, "alloca %\"github.com/xgo-dev/llgo/runtime/internal/runtime.llgoMapIter\""); got != count {
					t.Fatalf("%s has %d iterator slots, want %d:\n%s", name, got, count, body)
				}
				if got := strings.Count(body, "runtime.MapIterInit\""); got != count {
					t.Fatalf("%s has %d iterator initializations, want %d:\n%s", name, got, count, body)
				}
				if roots && !strings.Contains(body, "@llvm_gc_root_chain") {
					t.Fatalf("%s lost explicit iterator roots:\n%s", name, body)
				}
			}
			if roots && !strings.Contains(ir, "__llgo_gcroot_stack_pointer_") {
				t.Fatal("iterator fields have no typed stack root metadata")
			}
		})
	}
}
