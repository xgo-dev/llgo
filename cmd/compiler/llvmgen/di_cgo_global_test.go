package llvmgen

import (
	"bytes"
	"debug/pe"
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestDebugReplacedGlobalDoesNotLeaveCOFFReference(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	for _, target := range []Target{
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-w64-windows-gnu"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-pc-windows-msvc"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			pkg := prog.NewPackage("p", "example.com/p")
			pkg.InitDebug("p", "example.com/p", token.NewFileSet())
			const placeholder = "example.com/p.__cgo_callback"
			global := pkg.NewVar(placeholder, types.NewPointer(types.Typ[types.UnsafePointer]), GoDecl)
			keep := pkg.NewVar("example.com/p.callback_address", types.NewPointer(types.NewPointer(types.Typ[types.UnsafePointer])), GoDecl)
			keep.Init(global.Expr)
			callback := pkg.NewFunc("c_callback", NoArgsNoRet, CDecl)
			builder := callback.MakeBody(1)
			defer builder.Dispose()
			builder.DIGlobal(global.Expr, placeholder, token.Position{Filename: "generated.cgo.go", Line: 1})
			builder.Return()
			builder.EndBuild()
			pkg.FinalizeDebug()

			// Cgo lowers its generated address placeholder after debug metadata
			// is finalized. A bare RAUW leaves a phantom DWARF symbol on COFF.
			if !pkg.ReplaceVarWith(placeholder, callback.Expr) {
				t.Fatal("address placeholder was not replaced")
			}
			if pkg.ReplaceVarWith(placeholder, callback.Expr) {
				t.Fatal("repeated cgo lowering replaced the placeholder twice")
			}
			if pkg.VarOf(placeholder) != nil || pkg.glbDbgVars[global.Expr] {
				t.Fatal("replaced placeholder remains in package indexes")
			}
			if keep.impl.Initializer() != callback.impl {
				t.Fatal("callback address initializer was not rewritten")
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			object, err := prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			defer object.Dispose()
			coff, err := pe.NewFile(bytes.NewReader(object.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			defer coff.Close()
			if info := coff.Section(".debug_info"); info == nil || info.Size == 0 {
				t.Fatal("object lost DWARF")
			}
			for _, symbol := range coff.Symbols {
				if strings.Contains(symbol.Name, "__cgo_callback") {
					t.Fatalf("replaced address placeholder remains in COFF: %+v", symbol)
				}
			}
		})
	}
}
