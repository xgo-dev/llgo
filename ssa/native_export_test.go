package ssa

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNativeExportBoundary(t *testing.T) {
	for _, arch := range []string{"386", "amd64", "arm64"} {
		for _, cc := range []CallConv{CDecl, Stdcall} {
			for _, result := range []bool{false, true} {
				prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
				pkg := prog.NewPackage("entries", "entries")
				param := types.NewVar(0, nil, "value", types.Typ[types.Int32])
				var results *types.Tuple
				if result {
					results = types.NewTuple(param)
				}
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), results, false)
				var ensure Function
				if result {
					ensure = pkg.NewFunc("ensureInit", types.NewSignatureType(nil, nil, nil, nil, nil, false), GoDecl)
				}
				pkg.SetNativeExportSignature("entries.Go", sig)
				wrapper := pkg.DefineNativeExport("entries.Go", "native_entry", pkg.NativeExportSignature("entries.Go"), cc, ensure)
				wantCC := llvm.CCallConv
				if cc == Stdcall && arch == "386" {
					wantCC = llvm.X86StdcallCallConv
				}
				if wrapper.impl.FunctionCallConv() != wantCC {
					t.Fatalf("%s/%v has wrong entry ABI", arch, cc)
				}
				ir := wrapper.impl.String()
				previous := -1
				calls := []string{"EnterForeignThread", "entries.Go", "ExitForeignThread"}
				if result {
					calls = append([]string{"ensureInit"}, calls...)
				}
				for _, name := range calls {
					pos := strings.Index(ir, name)
					if pos <= previous {
						t.Fatalf("entry call order lost %s:\n%s", name, ir)
					}
					previous = pos
				}
				if !pkg.NeedRuntime || pkg.FuncOf("entries.Go").impl.BasicBlocksCount() != 0 {
					t.Fatal("entry replaced its Go implementation")
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				prog.Dispose()
			}
		}
	}
}
