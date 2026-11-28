package cl

import (
	"fmt"
	"go/types"
	"os"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// The portable bridge composes archsimd operations, including private helpers.
// Check each supported width's entire static call graph, so changes in the
// selected Go toolchain cannot silently introduce an unimplemented fallback.
func TestSIMDPortableBridgeClosure(t *testing.T) {
	for _, target := range []struct {
		arch string
		bits int
	}{{"amd64", 128}, {"amd64", 256}, {"amd64", 512}, {"arm64", 128}, {"wasm", 128}} {
		arch := target.arch
		t.Run(fmt.Sprintf("%s/%d", arch, target.bits), func(t *testing.T) {
			goos := "linux"
			if arch == "wasm" {
				goos = "wasip1"
			}
			pkgs, err := packages.Load(&packages.Config{
				Mode: packages.LoadAllSyntax,
				Env:  append(os.Environ(), "GOOS="+goos, "GOARCH="+arch, "GOEXPERIMENT=simd"),
			}, "simd/internal/bridge")
			if err != nil {
				t.Fatal(err)
			}
			if packages.PrintErrors(pkgs) != 0 || len(pkgs) != 1 {
				t.Fatal("could not load the official portable bridge")
			}
			prog, roots := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
			prog.Build()
			var queue []*ssa.Function
			for fn := range ssautil.AllFunctions(prog) {
				if fn.Pkg == roots[0] && bridgeWidthSignature(fn.Signature, pkgs[0].TypesSizes, target.bits) {
					queue = append(queue, fn)
				}
			}
			if len(queue) == 0 {
				t.Fatalf("no %d-bit bridge entries were found", target.bits)
			}
			t.Logf("checking %d bridge entries", len(queue))
			seen := make(map[*ssa.Function]bool)
			for len(queue) != 0 {
				fn := queue[0]
				queue = queue[1:]
				if seen[fn] {
					continue
				}
				seen[fn] = true
				if fn.Pkg != nil && !strings.HasPrefix(fn.Pkg.Pkg.Path(), "simd/") {
					continue
				}
				if op, ok := lookupSIMD(fn, arch); ok && op.op == llssa.SIMDUnimplemented {
					t.Errorf("portable bridge reaches an unsupported intrinsic: %s", fn)
				}
				for _, bb := range fn.Blocks {
					for _, inst := range bb.Instrs {
						if call, ok := inst.(ssa.CallInstruction); ok {
							if callee := call.Common().StaticCallee(); callee != nil {
								queue = append(queue, callee)
							}
						}
					}
				}
			}
		})
	}
}

func bridgeWidthSignature(sig *types.Signature, sizes types.Sizes, bits int) bool {
	hasWidth, hasWider := false, false
	check := func(t types.Type) {
		named, ok := types.Unalias(t).(*types.Named)
		if !ok || named.Obj().Pkg() == nil {
			return
		}
		path := named.Obj().Pkg().Path()
		if path != "simd/internal/bridge" && path != "simd/archsimd" {
			return
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok || st.NumFields() != 2 {
			return
		}
		lanes, ok := st.Field(1).Type().(*types.Array)
		if !ok {
			return
		}
		if sizes.Sizeof(lanes)*8 == int64(bits) {
			hasWidth = true
		} else if sizes.Sizeof(lanes)*8 > int64(bits) {
			hasWider = true
		}
	}
	if sig.Recv() != nil {
		check(sig.Recv().Type())
	}
	for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
		for i := 0; i < tuple.Len(); i++ {
			check(tuple.At(i).Type())
		}
	}
	return hasWidth && !hasWider
}
