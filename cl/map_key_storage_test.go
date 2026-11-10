package cl

import (
	"fmt"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestMapKeyTemporaryStorage(t *testing.T) {
	const src = `package mapkeys
import _ "unsafe"
//go:linkname KeepAlive llgo.keepAlive
func KeepAlive(any)
type Key struct { A, B int64 }
type PointerKey struct { P *int; N int }
func lookup(m map[Key]int, k Key) int { return m[k] }
func lookupOK(m map[Key]int, k Key) (int, bool) { v, ok := m[k]; return v, ok }
func update(m map[Key]int, k Key, v int) { m[k] = v }
func remove(m map[Key]int, k Key) { delete(m, k) }
func loop(m map[Key]int, n int) int { s:=0; for i:=0;i<n;i++ { s+=m[Key{int64(i),1}] }; return s }
func pointer(m map[PointerKey]int, k PointerKey) int { return m[k] }
func large(m map[[65537]byte]int, k *[65537]byte) int { return m[*k] }
func budget(m map[Key]int, k Key, p [65536]byte) int { KeepAlive(p); return m[k] }
func pointerBudget(m map[PointerKey]int, k PointerKey, p [65520]byte) int { KeepAlive(p); return m[k] }
func mapRange(m map[Key]int) int { sum:=0; for k,v:=range m { sum+=int(k.A)+v }; return sum }
func stringRange(s string) int { sum:=0; for _,r:=range s { sum+=int(r) }; return sum }
func mapBudget(m map[Key]int,p [65536]byte) int { KeepAlive(p); sum:=0; for _,v:=range m { sum+=v }; return sum }
func stringBudget(s string,p [65536]byte) int { KeepAlive(p); sum:=0; for _,r:=range s { sum+=int(r) }; return sum }
`
	for _, target := range []llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64"},
	} {
		roots := target.GOARCH == "wasm"
		t.Run(fmt.Sprintf("%s/%s/%s", target.GOOS, target.GOARCH, target.WasmProfile), func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, &target, roots)
			for _, name := range []string{"lookup", "lookupOK", "update", "remove", "loop", "pointer"} {
				body := llvmFunction(t, ir, "mapkeys."+name)
				if strings.Contains(body, "runtime.Alloc") || !strings.Contains(body, "alloca ") {
					t.Errorf("%s must borrow caller-owned key storage:\n%s", name, body)
				}
			}
			for _, name := range []string{"large", "budget"} {
				if body := llvmFunction(t, ir, "mapkeys."+name); !strings.Contains(body, "runtime.AllocAtomicU") {
					t.Errorf("%s must keep the over-budget key on the heap:\n%s", name, body)
				}
			}
			for _, name := range []string{"map", "string"} {
				helper := "NewMapIter"
				if name == "string" {
					helper = "NewStringIter"
				}
				body := llvmFunction(t, ir, "mapkeys."+name+"Range")
				if !strings.Contains(body, helper+"Buffer") || strings.Contains(body, "runtime.Alloc") {
					t.Errorf("%s range must use invocation-owned iteration storage:\n%s", name, body)
				}
				if roots && !strings.Contains(body, "gcroot.init") {
					t.Errorf("%s iterator lost typed field roots:\n%s", name, body)
				}
				if body := llvmFunction(t, ir, "mapkeys."+name+"Budget"); strings.Contains(body, helper+"Buffer") || !strings.Contains(body, helper+`"`) {
					t.Errorf("%s iterator ignored the shared budget:\n%s", name, body)
				}
			}
			if roots {
				if body := llvmFunction(t, ir, "mapkeys.pointerBudget"); !strings.Contains(body, "runtime.AllocU") {
					t.Errorf("key field roots exceeded the remaining stack budget:\n%s", body)
				}
				body := llvmFunction(t, ir, "mapkeys.pointer")
				if !strings.Contains(body, "gcroot.init") || !strings.Contains(ir, "__llgo_gcroot_stack_pointer_") {
					t.Fatalf("pointer-bearing key lost typed field roots:\n%s", body)
				}
			} else if body := llvmFunction(t, ir, "mapkeys.pointerBudget"); strings.Contains(body, "runtime.Alloc") {
				t.Errorf("native key fits the remaining stack budget:\n%s", body)
			}
		})
	}
}
