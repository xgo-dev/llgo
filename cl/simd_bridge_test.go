package cl

import (
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestSIMDFMVBridgeIdentity(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
		Env:  append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "GOEXPERIMENT=simd"),
	}, "simd/internal/bridge", "simd/archsimd", "simd")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(pkgs) != 0 || len(pkgs) != 3 {
		t.Fatal("could not load official SIMD types")
	}
	byPath := make(map[string]*types.Package)
	for _, pkg := range pkgs {
		byPath[pkg.PkgPath] = pkg.Types
	}
	bridge := byPath["simd/internal/bridge"]
	arch := byPath["simd/archsimd"]
	for _, name := range bridge.Scope().Names() {
		obj, ok := bridge.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		want := false
		if peer := arch.Scope().Lookup(name); peer != nil {
			_, want = llssa.SIMDVectorShape(peer.Type())
		}
		if got := simdFMVVector(obj.Type()); got != want {
			t.Errorf("%s eligibility = %t, want %t", name, got, want)
		}
	}
	vector := bridge.Scope().Lookup("Uint32x4").Type()
	alias := types.NewAlias(types.NewTypeName(token.NoPos, bridge, "Alias", nil), vector)
	if !simdFMVVector(alias) {
		t.Fatal("alias lost bridge eligibility")
	}
	other := types.NewPackage("example/bridge", "bridge")
	for _, typ := range []types.Type{
		byPath["simd"].Scope().Lookup("Uint32s").Type(),
		types.NewPointer(vector),
		types.NewNamed(types.NewTypeName(token.NoPos, other, "Uint32x4", nil), vector.Underlying(), nil),
		types.NewNamed(types.NewTypeName(token.NoPos, bridge, "Uint32x4", nil), types.NewStruct(nil, nil), nil),
	} {
		if simdFMVVector(typ) {
			t.Errorf("unexpected vector eligibility: %s", typ)
		}
	}
}

// The portable bridge composes archsimd operations, including private helpers.
// Check that the entire static call graph of its 128-bit entries is supported,
// so changes in the selected Go toolchain cannot silently introduce a fallback.
func TestSIMDPortableBridgeClosure(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64", "wasm"} {
		t.Run(arch, func(t *testing.T) {
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
				if fn.Pkg == roots[0] && bridge128Signature(fn.Signature, pkgs[0].TypesSizes) {
					queue = append(queue, fn)
				}
			}
			if len(queue) == 0 {
				t.Fatal("no 128-bit bridge entries were found")
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

func bridge128Signature(sig *types.Signature, sizes types.Sizes) bool {
	has128, hasWide := false, false
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
		if sizes.Sizeof(lanes) == 16 {
			has128 = true
		} else {
			hasWide = true
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
	return has128 && !hasWide
}

func TestSIMDFMVBridgeRequiresArchsimdPeer(t *testing.T) {
	// A bridge-looking name is insufficient: its package must import the
	// canonical vector, and the peer must be a type with matching storage.
	for _, tc := range []struct {
		name    string
		imports []*types.Package
	}{
		{"no imports", nil},
		{"unrelated import", []*types.Package{types.NewPackage("example/archsimd", "archsimd")}},
		{"missing peer", []*types.Package{types.NewPackage("simd/archsimd", "archsimd")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bridge := types.NewPackage("simd/internal/bridge", "bridge")
			bridge.SetImports(tc.imports)
			vector := types.NewNamed(types.NewTypeName(token.NoPos, bridge, "Uint32x4", nil), types.NewStruct(nil, nil), nil)
			if simdFMVVector(vector) {
				t.Fatal("accepted bridge without canonical archsimd peer")
			}
		})
	}
}
