package cl

import (
	"bytes"
	"debug/elf"
	"fmt"
	gobuild "go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestAtomicStoreUnorderedOrdering(t *testing.T) {
	const source = `package atomicordering
import _ "unsafe"

//go:linkname store llgo.atomicStore
func store(*uintptr, uintptr)

//go:linkname storeUnordered llgo.atomicStoreUnordered
func storeUnordered(*uintptr, uintptr)

func Ordered(p *uintptr, v uintptr) { store(p, v) }
func Unordered(p *uintptr, v uintptr) { storeUnordered(p, v) }
func Deferred(p *uintptr, v uintptr) { defer storeUnordered(p, v) }
func Asynchronous(p *uintptr, v uintptr) { go storeUnordered(p, v) }
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ssaPkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			defer pkg.Module().Dispose()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid atomic-store IR: %v\n%s", err, pkg.String())
			}
			for _, test := range []struct{ name, ordering string }{
				{"Ordered", "seq_cst"},
				{"Unordered", "unordered"},
				{"Deferred", "unordered"},
				{"_llgo_routine$1", "unordered"},
			} {
				body := mustNamedFunction(t, pkg.Module(), "atomicordering."+test.name).String()
				stores := 0
				for _, line := range strings.Split(body, "\n") {
					if strings.Contains(line, "store atomic ") {
						stores++
						if !strings.Contains(line, " "+test.ordering+",") {
							t.Errorf("%s store must use %s ordering:\n%s", test.name, test.ordering, body)
						}
					}
				}
				if stores != 1 {
					t.Errorf("%s has %d atomic stores, want one:\n%s", test.name, stores, body)
				}
			}
			// A go statement must put the atomic store in the worker instead of
			// executing it synchronously while constructing the startup record.
			body := mustNamedFunction(t, pkg.Module(), "atomicordering.Asynchronous").String()
			if strings.Contains(body, "store atomic ") || !strings.Contains(body, llssa.PkgRuntime+".NewProc") {
				t.Errorf("go statement did not defer the store to its worker:\n%s", body)
			}
		})
	}
}

func TestAtomicStoreUnorderedInvalidArity(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			defer func() {
				if got := fmt.Sprint(recover()); got != "atomicStoreUnordered(addr *T, val T): invalid arguments" {
					t.Errorf("invalid-arity panic = %q", got)
				}
			}()
			var ctx context
			ctx.atomicStoreUnordered(nil, make([]llssa.Expr, count))
		})
	}
}

func TestBaremetalItabPointersDoNotRequireLibatomic(t *testing.T) {
	runtimeDir := filepath.Join("..", "runtime", "internal", "runtime")
	ctx := gobuild.Default
	ctx.GOOS, ctx.GOARCH = "linux", "arm"
	ctx.BuildTags = []string{"llgo", "baremetal", "esp32c3", "tinygo.riscv32"}
	for name, want := range map[string]bool{
		"itab_pointer_baremetal.go": true,
		"itab_pointer.go":           false,
	} {
		selected, err := ctx.MatchFile(runtimeDir, name)
		if err != nil {
			t.Fatal(err)
		}
		if selected != want {
			t.Fatalf("baremetal selected %s = %v, want %v", name, selected, want)
		}
	}
	source, err := os.ReadFile(filepath.Join(runtimeDir, "itab_pointer_baremetal.go"))
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, files := buildGoSSAPkg(t, string(source))
	prog := newLLSSAProgForTarget(t, &llssa.Target{
		GOOS:       "linux",
		GOARCH:     "arm",
		Target:     "esp32c3-basic",
		LLVMTarget: "riscv32-esp-elf",
	})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	defer mod.Dispose()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid baremetal itab-pointer IR: %v\n%s", err, pkg.String())
	}
	for _, name := range []string{"loadItabPointer", "storeItabPointer"} {
		body := mustNamedFunction(t, mod, "runtime."+name).String()
		if strings.Contains(body, "load atomic ") || strings.Contains(body, "store atomic ") {
			t.Fatalf("baremetal %s retained an atomic operation:\n%s", name, body)
		}
	}

	// Named embedded targets use an ARM-compatible Go frontend, then emit
	// through their physical target's backend. RV32IMC lacks the A extension:
	// even pointer-sized atomics can become unavailable __atomic_* libcalls.
	const triple = "riscv32-esp-elf"
	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		t.Fatal(err)
	}
	machine := target.CreateTargetMachine(triple, "generic-rv32", "+m,+c,-a",
		llvm.CodeGenLevelDefault, llvm.RelocDefault, llvm.CodeModelDefault)
	defer machine.Dispose()
	data := machine.CreateTargetData()
	defer data.Dispose()
	mod.SetTarget(triple)
	mod.SetDataLayout(data.String())
	assembly, err := machine.EmitToMemoryBuffer(mod, llvm.AssemblyFile)
	if err != nil {
		t.Fatal(err)
	}
	defer assembly.Dispose()
	if strings.Contains(string(assembly.Bytes()), "__atomic_") {
		t.Fatalf("baremetal itab-pointer helpers require libatomic:\n%s", assembly.Bytes())
	}
	object, err := machine.EmitToMemoryBuffer(mod, llvm.ObjectFile)
	if err != nil {
		t.Fatal(err)
	}
	defer object.Dispose()
	file, err := elf.NewFile(bytes.NewReader(object.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.Machine != elf.EM_RISCV {
		t.Fatalf("object architecture = %v, want RISC-V", file.Machine)
	}
	symbols, err := file.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range symbols {
		if symbol.Section == elf.SHN_UNDEF && strings.HasPrefix(symbol.Name, "__atomic_") {
			t.Fatalf("baremetal itab-pointer object imports %s", symbol.Name)
		}
	}
}
