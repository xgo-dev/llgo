package cl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestRegularMemoryEqualLowering(t *testing.T) {
	const src = `package keys
 type Zero [0]uint64
 type One struct { A byte }
 type Two [2]byte
 type Four struct { A uint32 }
 type Eight [2]uint32
 type Sixteen struct { A, B uint64 }
 type Large struct { A, B, C uint64 }
 type SameSize [3]uint64
 type Float struct { A float64 }
 type String struct { A string }
 type Padded struct { A byte; B uint64 }
 type Blank struct { A uint64; _ uint64 }
 type Interface struct { A any }
 type Uncomparable struct { A []byte }
 func Keys() []any { return []any{Zero{}, One{}, Two{}, Four{}, Eight{}, Sixteen{}, Large{}, SameSize{}, Float{}, String{}, Padded{}, Blank{}, Interface{}, Uncomparable{}} }
 `
	for _, tc := range []struct {
		arch    string
		enabled bool
	}{{"wasm", true}, {"wasm", false}, {"amd64", true}, {"amd64", false}, {"386", true}, {"386", false}, {"arm64", true}, {"arm64", false}} {
		t.Run(fmt.Sprintf("%s/%v", tc.arch, tc.enabled), func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOARCH: tc.arch})
			defer prog.Dispose()
			prog.EnableDeadcodeDrop(tc.enabled)
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			optimized := tc.arch == "wasm" && tc.enabled
			for _, item := range []struct{ name, equal, native, fallback string }{
				{"Zero", "memequal0", "memequal_varlen", "arrayequal"}, {"One", "memequal8", "memequal_varlen", "structequal"},
				{"Two", "memequal16", "memequal_varlen", "arrayequal"}, {"Four", "memequal32", "memequal_varlen", "structequal"},
				{"Eight", "memequal64", "memequal_varlen", "arrayequal"}, {"Sixteen", "memequal128", "memequal_varlen", "structequal"},
				{"Large", "_llgo_memequal.24", "memequal_varlen", "structequal"}, {"SameSize", "_llgo_memequal.24", "memequal_varlen", "arrayequal"},
				{"Float", "structequal", "structequal", "structequal"}, {"String", "structequal", "structequal", "structequal"},
				{"Padded", "structequal", "structequal", "structequal"}, {"Blank", "structequal", "structequal", "structequal"},
				{"Interface", "structequal", "structequal", "structequal"},
			} {
				global := mod.NamedGlobal("_llgo_keys." + item.name)
				if global.IsNil() {
					t.Fatalf("missing descriptor for %s", item.name)
				}
				want := item.fallback
				if optimized {
					want = item.equal
				} else if tc.arch != "wasm" {
					// The qualified native bridge carries the Type* environment;
					// Wasm DCE instead uses metadata-independent equality.
					want = item.native
				}
				if !strings.Contains(global.Initializer().String(), want) {
					t.Errorf("%s: missing %s in %s", item.name, want, global)
				}
			}
			helper := mod.NamedFunction("_llgo_memequal.24")
			if helper.IsNil() == optimized {
				t.Fatal("unexpected shared equality helper")
			}
			if optimized && (helper.Linkage() != llvm.WeakODRLinkage || !strings.Contains(helper.String(), "i32 24")) {
				t.Fatalf("invalid shared helper: %s", helper)
			}
			// The external compiler parses textual IR; VerifyModule alone does not
			// reject every mismatched closure initializer.
			path := filepath.Join(t.TempDir(), "keys.ll")
			if err := os.WriteFile(path, []byte(mod.String()), 0600); err != nil {
				t.Fatal(err)
			}
			buf, err := llvm.NewMemoryBufferFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			parsed, err := ctx.ParseIR(buf)
			if err != nil {
				t.Fatal(err)
			}
			parsed.Dispose()
		})
	}
}
