package cl

import (
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
