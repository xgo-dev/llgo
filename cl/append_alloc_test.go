package cl

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestAppendTemporaryStorage(t *testing.T) {
	const src = `package appendalloc
type Item struct { P *int; A, B uint64 }
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
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
				ir := compileWithRewritesModeTarget(t, src, nil, mode, &target)
				for _, name := range []string{"scalar", "several", "loop", "spread", "zero"} {
					body := llvmFunction(t, ir, "appendalloc."+name)
					if strings.Contains(body, "runtime.Alloc") {
						t.Errorf("%s retains temporary heap allocation:\n%s", name, body)
					}
				}
				for _, name := range []string{"escaped", "escapedTwo", "addressEscapes"} {
					if body := llvmFunction(t, ir, "appendalloc."+name); !strings.Contains(body, "runtime.Alloc") {
						t.Fatalf("%s lost escaping storage:\n%s", name, body)
					}
				}
			}
		})
	}
}

func TestAppendTemporaryRejectsExposedStorage(t *testing.T) {
	// The generated array has no source-level name. Model subsequent SSA
	// rewrites exposing its address, adding a consumer, or extending its life
	// across blocks; none may make it eligible for stack-slot reuse.
	for _, scenario := range []string{"cross block", "array read", "element address escapes", "element read", "returned slice", "second consumer", "append destination"} {
		t.Run(scenario, func(t *testing.T) {
			pkg, _, _ := buildGoSSAPkg(t, `package appenduses
var saved *int
var snapshot [1]int
func f(s []int, x int, b bool, a *[1]int) []int {
 saved = &x
 snapshot = *a
 v := append(s,x)
 if b { return v }; return a[:]
}`)
			fn := pkg.Func("f")
			var alloc *ssa.Alloc
			var elementRead, arrayRead *ssa.UnOp
			var escapeStore *ssa.Store
			var returned *ssa.Return
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					switch instr := instr.(type) {
					case *ssa.Alloc:
						if instr.Comment == "varargs" {
							alloc = instr
						}
					case *ssa.UnOp:
						if _, ok := instr.Type().Underlying().(*types.Array); ok {
							arrayRead = instr
						} else {
							elementRead = instr
						}
					case *ssa.Store:
						if instr.Addr == pkg.Members["saved"].(*ssa.Global) {
							escapeStore = instr
						}
					case *ssa.Return:
						returned = instr
					}
				}
			}
			if alloc == nil || elementRead == nil || arrayRead == nil || escapeStore == nil || returned == nil {
				t.Fatal("missing append graph")
			}
			var index *ssa.IndexAddr
			var slice *ssa.Slice
			for _, ref := range *alloc.Referrers() {
				switch ref := ref.(type) {
				case *ssa.IndexAddr:
					index = ref
				case *ssa.Slice:
					slice = ref
				}
			}
			if index == nil || slice == nil {
				t.Fatal("missing temporary users")
			}
			call, ok := (*slice.Referrers())[0].(*ssa.Call)
			if !ok {
				t.Fatal("missing append call")
			}
			ctx := &context{prog: newLLSSAProg(t)}
			if !ctx.canStackAllocateAppendArgs(alloc) {
				t.Fatal("control array was not eligible")
			}
			switch scenario {
			case "cross block":
				var foreign *ssa.Slice
				for _, block := range fn.Blocks {
					for _, instr := range block.Instrs {
						if v, ok := instr.(*ssa.Slice); ok && v.Block() != alloc.Block() {
							foreign = v
						}
					}
				}
				if foreign == nil {
					t.Fatal("missing cross-block slice")
				}
				copy := *foreign
				copy.X = alloc
				*alloc.Referrers() = append(*alloc.Referrers(), &copy)
			case "array read":
				copy := *arrayRead
				copy.X = alloc
				*alloc.Referrers() = append(*alloc.Referrers(), &copy)
			case "element address escapes":
				copy := *escapeStore
				copy.Val = index
				*index.Referrers() = append(*index.Referrers(), &copy)
			case "element read":
				copy := *elementRead
				copy.X = index
				copy.Op = token.MUL
				*index.Referrers() = append(*index.Referrers(), &copy)
			case "returned slice":
				copy := *returned
				copy.Results = []ssa.Value{slice}
				*slice.Referrers() = []ssa.Instruction{&copy}
			case "second consumer":
				copy := *call
				*slice.Referrers() = append(*slice.Referrers(), &copy)
			case "append destination":
				call.Call.Args[0] = slice
			}
			if ctx.canStackAllocateAppendArgs(alloc) {
				t.Fatal("exposed storage was accepted")
			}
		})
	}
}
