package cl

import (
	"go/constant"
	"go/types"
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
func constantCapacity() []byte { return append([]byte(nil),make([]byte,2,4)...) }
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
			for _, name := range []string{"nonnil", "capacity", "constantCapacity", "shared", "literal"} {
				if body := llvmFunction(t, ir, "fresh."+name); !strings.Contains(body, "SliceAppend") {
					t.Errorf("%s incorrectly forwarded its source backing array:\n%s", name, body)
				}
			}
		})
	}
}

func TestFreshNilAppendSourceGuards(t *testing.T) {
	for _, name := range []string{"destination conversion", "missing allocation type", "non-array allocation", "dynamic extent", "missing extent", "partial extent"} {
		t.Run(name, func(t *testing.T) {
			pkg, _ := buildCallerFrameSSAPackage(t, "fresh", `package fresh
func constant(n int) []byte { return append([]byte(nil),make([]byte,4)...) }
func scalar() *int { return new(int) }
`)
			fn := pkg.Func("constant")
			var call *ssa.Call
			for _, block := range fn.Blocks {
				for _, instruction := range block.Instrs {
					if candidate, ok := instruction.(*ssa.Call); ok {
						call = candidate
					}
				}
			}
			if call == nil || freshNilAppendSource(call) == nil {
				t.Fatal("constant make is no longer a forwarding candidate")
			}
			slice, ok := call.Call.Args[1].(*ssa.Slice)
			if !ok {
				t.Fatal("constant make no longer uses array-backed SSA")
			}
			switch name {
			case "destination conversion":
				// Canonical SSA normally folds this nil conversion. The matcher
				// also accepts a frontend retaining the value-preserving node.
				call.Call.Args[0] = &ssa.ChangeType{X: call.Call.Args[0]}
				if freshNilAppendSource(call) != slice {
					t.Fatal("nil destination conversion hid the fresh source")
				}
				return
			case "missing allocation type":
				// Incomplete SSA must fail closed instead of panicking while
				// inspecting the allocation's backing-array type.
				slice.X = &ssa.Alloc{Comment: "makeslice"}
			case "non-array allocation":
				for _, block := range pkg.Func("scalar").Blocks {
					for _, instruction := range block.Instrs {
						if allocation, ok := instruction.(*ssa.Alloc); ok {
							allocation.Comment = "makeslice"
							slice.X = allocation
						}
					}
				}
			case "dynamic extent":
				slice.High = fn.Params[0]
			case "missing extent":
				slice.High = ssa.NewConst(nil, types.Typ[types.Int])
			case "partial extent":
				slice.High = ssa.NewConst(constant.MakeInt64(2), types.Typ[types.Int])
			}
			if freshNilAppendSource(call) != nil {
				t.Fatal("unsupported storage was forwarded")
			}
			if freshNilAppendAllocation(fn.Params[0]) {
				t.Fatal("ordinary parameter was treated as append storage")
			}
		})
	}
}

func TestFreshNilAppendEscapePlacement(t *testing.T) {
	pkg, _ := buildCallerFrameSSAPackage(t, "fresh", `package fresh
func dynamic(n int) []byte { return append([]byte(nil), make([]byte,n)...) }
func constant() []byte { return append([]byte(nil), make([]byte,4)...) }
func borrowed(n int) int { return len(make([]byte,n)) }
`)
	prog := newLLSSAProgForTarget(t, nil)
	defer prog.Dispose()
	facts := PrepareEscapeAnalysis(prog, pkg.Prog, nil)
	for _, name := range []string{"dynamic", "constant", "borrowed"} {
		found := false
		for _, block := range pkg.Func(name).Blocks {
			for _, instruction := range block.Instrs {
				switch instruction.(type) {
				case *ssa.MakeSlice, *ssa.Alloc:
				default:
					continue
				}
				decision, ok := facts.LookupDecision(instruction)
				if !ok {
					t.Fatalf("%s has no placement decision", name)
				}
				if name == "borrowed" {
					if decision.Heap {
						t.Fatalf("ordinary make lost stack eligibility: %+v", decision)
					}
				} else if !decision.Heap || decision.Reason != "fresh append result storage" {
					t.Fatalf("%s did not give the forwarded source result lifetime: %+v", name, decision)
				}
				found = true
			}
		}
		if !found {
			t.Fatalf("%s no longer exercises slice allocation", name)
		}
	}
}
