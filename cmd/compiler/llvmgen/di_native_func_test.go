package llvmgen

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestDebugNativeFunctionSnapshots(t *testing.T) {
	for _, abi := range nativeIntegerABITests {
		for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
			for _, kind := range []string{"fixed", "variadic", "Go-declaration"} {
				target := abi.target
				target.OptLevel = level
				t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.LLVMTarget+"/"+level.String()+"/"+kind, func(t *testing.T) {
					prog := NewProgram(&target)
					defer prog.Dispose()
					prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
					prog.SetRuntime(newDebugRuntimePackage())
					pkg := prog.NewPackage("p", "example.com/p")
					pkg.InitDebug("p", pkg.Path(), token.NewFileSet())
					params := []*types.Var{types.NewVar(token.NoPos, nil, "fixed", types.Typ[types.Int8])}
					if kind == "variadic" {
						params = append(params, VArg())
					}
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), nil, kind == "variadic")
					native := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage(pkg.Path(), "p"), "Native", nil), sig, nil)
					prog.SetTypeCallConv(pkg.Path()+".Native", CDecl)
					variable := types.NewVar(token.NoPos, nil, "entry", native)
					ownerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(variable), nil, false)
					fn := pkg.NewFunc(pkg.Path()+".snapshot", ownerSig, GoDecl)
					b := fn.MakeBody(2)
					defer b.Dispose()
					pos := token.Position{Filename: "snapshot.go", Line: 1, Column: 1}
					b.DebugFunction(fn, nil, pos, pos)
					b.Jump(fn.Block(1))
					b.SetBlock(fn.Block(1))
					value := fn.Param(0)
					want := prog.storageType(value.Type)
					if kind == "Go-declaration" {
						value = pkg.NewFunc(pkg.Path()+".goFunction", sig, GoDecl).Expr
						variable = types.NewVar(token.NoPos, nil, "entry", sig)
						want = prog.storageType(prog.Closure(sig))
					}
					// A native pointer needs one pointer slot, including when its
					// signature has a C ellipsis. Only Go declarations become closures.
					home, store := b.constructDebugAddrWithStore(value)
					if got := home.impl.AllocatedType(); got != want {
						t.Fatalf("debug snapshot reserves %s, want %s", got.String(), want.String())
					}
					if home.impl.InstructionParent() != fn.impl.EntryBasicBlock() || store.impl.InstructionParent() != fn.Block(1).first {
						t.Fatal("debug storage must be allocated in the entry block and updated at the source location")
					}
					b.DIValue(variable, value, b.DIVarAuto(fn, pos, "entry", value.Type), fn, pos, fn.Block(1))
					b.Return()
					b.EndBuild()
					pkg.FinalizeDebug()
					if strings.Contains(pkg.String(), ".__llgo_c_funcval$") {
						t.Fatalf("debug snapshot created a native-to-Go adapter:\n%s", pkg.String())
					}
					if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
						t.Fatalf("invalid function debug snapshot: %v\n%s", err, pkg.String())
					}
				})
			}
		}
	}
}
