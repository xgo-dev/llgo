package cl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

// Exercise ordinary Go calls with addresses that no longer carry knowledge of
// their containing HostLayout record. Direct field accesses alone cannot catch
// a narrow field slot being overwritten by a normal Go pointer store.
func TestHostLayoutWasmAddressEscape(t *testing.T) {
	const src = `package hostescape
import "structs"
import "unsafe"
type Word struct { _ structs.HostLayout; U uintptr; Tail uint32 }
type Pointer struct { _ structs.HostLayout; P *byte; Tail uint32 }
type Words struct { _ structs.HostLayout; U [2]uintptr; Tail uint32 }
type Pointers struct { _ structs.HostLayout; P [2]*byte; Tail uint32 }
type Tail struct { _ structs.HostLayout; Value uint64; Tail struct{} }
var tails [2]Tail
var word Word
var pointer Pointer
var words Words
var pointers Pointers
var saved *uintptr
var value byte
//go:noinline
func putWord(p *uintptr) { *p = 0x1122334455667788 }
//go:noinline
func putPointer(p **byte) { *p = &value }
//go:noinline
func putWords(p *[2]uintptr) { *p = [2]uintptr{0x1122334455667788, 0x2233445566778899} }
//go:noinline
func save(p *uintptr) { saved = p }
func TailProbe() uint32 {
 first := uintptr(unsafe.Pointer(&tails[0]))
 next := uintptr(unsafe.Pointer(&tails[1]))
 tail := uintptr(unsafe.Pointer(&tails[0].Tail))
 if tail < first || tail >= next || next-first != 16 { return 7 }
 return 0
}
func WordProbe() uint32 {
 word.Tail = 0xdeadbeef
 putWord(&word.U)
 if word.Tail != 0xdeadbeef || word.U != 0x1122334455667788 { return 1 }
 return 0
}
func PointerProbe() uint32 {
 pointer.Tail = 0xdeadbeef
 putPointer(&pointer.P)
 if pointer.Tail != 0xdeadbeef || pointer.P != &value { return 2 }
 return 0
}
func ElementProbe() uint32 {
 words.Tail = 0xdeadbeef
 words.U[1] = 9
 putWord(&words.U[0])
 if words.U[1] != 9 || words.Tail != 0xdeadbeef || words.U[0] != 0x1122334455667788 { return 3 }
 return 0
}
func ArrayProbe() uint32 {
 words.Tail = 0xdeadbeef
 putWords(&words.U)
 if words.Tail != 0xdeadbeef || words.U[0] != 0x1122334455667788 || words.U[1] != 0x2233445566778899 { return 4 }
 return 0
}
func SavedProbe() uint32 {
 word.Tail = 0xdeadbeef
 save(&word.U)
 putWord(saved)
 if word.Tail != 0xdeadbeef || word.U != 0x1122334455667788 { return 5 }
 return 0
}
func PointerArrayProbe() uint32 {
 pointers.Tail = 0xdeadbeef
 putPointer(&pointers.P[1])
 if pointers.Tail != 0xdeadbeef || pointers.P[1] != &value { return 6 }
 return 0
}
`
	ir := compileWithRewritesTarget(t, src, nil, &llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"})
	// The IR test runs even on hosts without a WebAssembly execution toolchain.
	if !strings.Contains(ir, "store i64 1234605616436508552") {
		t.Fatalf("ordinary uintptr store missing:\n%s", ir)
	}
	t.Run("execute", func(t *testing.T) {
		for _, tool := range []string{"opt", "llc", "wasm-ld", "wasmtime"} {
			if _, err := exec.LookPath(tool); err != nil {
				t.Skipf("%s required for Wasm execution", tool)
			}
		}
		dir := t.TempDir()
		input := filepath.Join(dir, "probe.ll")
		if err := os.WriteFile(input, []byte(ir), 0600); err != nil {
			t.Fatal(err)
		}
		run := func(t *testing.T, tool string, args ...string) {
			t.Helper()
			if out, err := exec.Command(tool, args...).CombinedOutput(); err != nil {
				t.Fatalf("%s: %v\n%s", tool, err, out)
			}
		}
		for _, level := range []string{"O0", "O2"} {
			t.Run(level, func(t *testing.T) {
				optimized := filepath.Join(dir, level+".ll")
				object := filepath.Join(dir, level+".o")
				module := filepath.Join(dir, level+".wasm")
				run(t, "opt", "-S", "-passes=default<"+level+">", input, "-o", optimized)
				run(t, "llc", "-filetype=obj", "-"+level, optimized, "-o", object)
				// These probes need no runtime services. Unresolved runtime error paths
				// become traps, so accidentally executing one still fails the test.
				args := []string{"--no-entry", "--unresolved-symbols=ignore-all", "-o", module, object}
				probes := []string{"WordProbe", "PointerProbe", "ElementProbe", "ArrayProbe", "SavedProbe", "PointerArrayProbe", "TailProbe"}
				for _, name := range probes {
					args = append(args, "--export=hostescape."+name)
				}
				run(t, "wasm-ld", args...)
				for _, name := range probes {
					t.Run(name, func(t *testing.T) {
						out, err := exec.Command("wasmtime", "run", "--invoke", "hostescape."+name, module).Output()
						if err != nil {
							if exit, ok := err.(*exec.ExitError); ok {
								t.Fatalf("Wasm execution: %v\n%s", err, exit.Stderr)
							}
							t.Fatalf("Wasm execution: %v", err)
						}
						if strings.TrimSpace(string(out)) != "0" {
							t.Fatalf("field-address probe returned %s, want 0", out)
						}
					})
				}
			})
		}
	})
}
