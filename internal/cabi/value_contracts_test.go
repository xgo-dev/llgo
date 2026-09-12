package cabi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/funcattrs"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func contractTestDeclaration(t *testing.T, prog llssa.Program, pkg llssa.Package, source, name string) llssa.Function {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "contracts.go", "package contracts\n"+source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	if _, err := (&types.Config{}).Check("contracts", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	decl := file.Decls[len(file.Decls)-1].(*ast.FuncDecl)
	attrs, err := funcattrs.Parse(fset, decl)
	if err != nil {
		t.Fatal(err)
	}
	if err := prog.SetFunctionAttributes(name, attrs); err != nil {
		t.Fatal(err)
	}
	return pkg.NewFunc(name, info.Defs[decl.Name].Type().(*types.Signature), llssa.InGo)
}

func TestValueContractsSurviveByvalSretAndPackedABI(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm32"},
	} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := llssa.NewProgram(&target)
			defer prog.Dispose()
			if target.GOARCH == "386" {
				prog.TypeSizes(types.SizesFor("gc", "386"))
			}
			pkg := prog.NewPackage("contracts", "contracts")
			callee := contractTestDeclaration(t, prog, pkg, `
type Pair struct { P *int; N int64; Extra [24]byte }
//llgo:attr param(p) nonnull noalias
//llgo:attr result(q) nonnull same_as(param(p))
//llgo:attr result(n) range(0,7)
func F(input Pair, p *int) (q *int, n int64, output Pair)
`, "contracts.F")
			calleeType := callee.Type.RawType().(*types.Signature)
			callerSig := types.NewSignatureType(nil, nil, nil, calleeType.Params(),
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)
			caller := pkg.NewFunc("contracts.Caller", callerSig, llssa.InGo)
			b := caller.MakeBody(1)
			r := b.Call(callee.Expr, caller.Param(0), caller.Param(1))
			pointer := b.Extract(r, 0)
			badPointer := b.BinOp(token.EQL, pointer, prog.Nil(pointer.Type))
			n := b.Extract(r, 1)
			negative := b.BinOp(token.LSS, n, prog.IntVal(0, n.Type))
			large := b.BinOp(token.GEQ, n, prog.IntVal(7, n.Type))
			b.Return(b.BinOp(token.OR, badPointer, b.BinOp(token.OR, negative, large)))
			b.EndBuild()

			packed := contractTestDeclaration(t, prog, pkg, `
type Packed struct { N int8; Other [7]byte }
//llgo:attr param(n) range(-3,4)
func PackedInput(p Packed, n int8) bool
`, "contracts.PackedInput")
			pbody := packed.MakeBody(1)
			value := packed.Param(1)
			pbody.Return(pbody.BinOp(token.GTR, value, prog.IntVal(3, value.Type)))
			pbody.EndBuild()

			mod := pkg.Module()
			logicalType := mod.NamedFunction("contracts.F").GlobalValueType()
			if err := funcattrs.MaterializeValueContracts(mod); err != nil {
				t.Fatal(err)
			}
			NewTransformer(prog, mod.Target(), "", false).TransformModule("contracts", mod)
			physical := mod.NamedFunction("contracts.F")
			if !prog.GCRootsEnabled() && !prog.CooperativeSafepointsEnabled() && physical.GetEnumAttributeAtIndex(physical.ParamsCount(), llvm.AttributeKindID("noalias")).IsNil() {
				t.Fatalf("pointer noalias lost across ABI conversion: %s", physical.String())
			}
			if physical.GlobalValueType() == logicalType {
				t.Fatal("test did not exercise aggregate ABI rewriting")
			}
			if physical.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("sret")).IsNil() {
				t.Fatalf("large source results were not transported via sret:\n%s", physical.String())
			}
			if target.GOARCH == "amd64" && physical.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("byval")).IsNil() {
				t.Fatalf("amd64 input did not exercise native byval:\n%s", physical.String())
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid transformed contracts: %v\n%s", err, mod.String())
			}
			options := llvm.NewPassBuilderOptions()
			defer options.Dispose()
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), options); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"contracts.Caller", "contracts.PackedInput"} {
				body := mod.NamedFunction(name).String()
				if !strings.Contains(body, "ret i1 false") {
					t.Fatalf("%s did not consume reconstructed value facts:\n%s", name, body)
				}
			}
			if !strings.Contains(mod.NamedFunction("contracts.Caller").String(), "@contracts.F(") {
				t.Fatal("result facts removed the unknown external invocation")
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			assembly, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatalf("cannot emit transformed contract assembly: %v", err)
			}
			assembly.Dispose()
		})
	}
}
