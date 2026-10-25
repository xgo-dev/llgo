package cl

import (
	"go/types"
	"os"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Public helpers such as byte multiplication and broadcasting call narrower
// intrinsic declarations. Qualify those transitive dependencies too.
func TestSIMD256BasicClosure(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax, Env: append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "GOEXPERIMENT=simd")}, "simd/archsimd")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(pkgs) != 0 || len(pkgs) != 1 {
		t.Fatal("could not load archsimd")
	}
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()
	basics := map[string]bool{}
	for _, name := range strings.Fields("Add Sub Mul Div Neg Not Abs And Or Xor AndNot OrNot Equal NotEqual Less LessEqual Greater GreaterEqual ToBits BitsToInt8 BitsToInt16 BitsToInt32 BitsToInt64 BitsToFloat32 BitsToFloat64 ReshapeToUint8s ReshapeToUint16s ReshapeToUint32s ReshapeToUint64s GetLo GetHi SetLo SetHi Store StoreArray StoreArrayMasked ShiftAllLeft ShiftAllRight") {
		basics[name] = true
	}
	var queue []*ssa.Function
	for fn := range ssautil.AllFunctions(prog) {
		obj, ok := fn.Object().(*types.Func)
		if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "simd/archsimd" {
			continue
		}
		sig := fn.Signature
		if sig.Recv() != nil {
			if _, ok := llssa.SIMDVectorShape(sig.Recv().Type()); !ok {
				continue // In particular, 512->256 extraction is a later stage.
			}
		}
		wide := sig.Recv() != nil && simdVectorBits(sig.Recv().Type()) == 256
		for i := 0; i < sig.Results().Len(); i++ {
			wide = wide || simdVectorBits(sig.Results().At(i).Type()) == 256
		}
		name := obj.Name()
		if wide && (basics[name] || strings.HasPrefix(name, "Load") || strings.HasPrefix(name, "Broadcast") || strings.HasSuffix(name, "FromBits")) {
			queue = append(queue, fn)
		}
	}
	if len(queue) < 100 {
		t.Fatalf("only %d wide roots", len(queue))
	}
	t.Logf("checking %d SIMD256 basic API roots", len(queue))
	seen := map[*ssa.Function]bool{}
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
		if op, ok := lookupSIMD(fn, "amd64"); ok && op.op == llssa.SIMDUnimplemented {
			t.Errorf("basic API reaches unsupported intrinsic: %s", fn)
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
}
