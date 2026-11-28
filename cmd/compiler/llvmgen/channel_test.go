package llvmgen

import (
	llpkgs "github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestChannelReceiveBufferLoads(t *testing.T) {
	rt, err := llpkgs.SourceImporter().Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	for _, elem := range []types.Type{
		types.Typ[types.Int],
		types.NewPointer(types.Typ[types.Int]),
		types.NewArray(types.Typ[types.Int], 4),
		types.NewInterfaceType(nil, nil).Complete(),
		types.NewStruct(nil, nil),
		types.NewArray(types.Typ[types.Int], 0),
	} {
		t.Run(elem.String(), func(t *testing.T) {
			for _, mode := range []string{"direct", "comma-ok", "select", "try-select"} {
				t.Run(mode, func(t *testing.T) {
					prog := NewProgram(nil)
					defer prog.Dispose()
					prog.SetRuntime(func() *types.Package { return rt })
					pkg := prog.NewPackage("p", "example.com/receive")
					params := types.NewTuple(types.NewVar(token.NoPos, nil, "ch", types.NewChan(types.SendRecv, elem)))
					results := types.NewTuple(types.NewVar(token.NoPos, nil, "", elem))
					fn := pkg.NewFunc("receive", types.NewSignatureType(nil, nil, nil, params, results, false), GoDecl)
					b := fn.MakeBody(1)
					var value Expr
					switch mode {
					case "direct":
						value = b.Recv(fn.Param(0), false)
					case "comma-ok":
						value = b.Extract(b.Recv(fn.Param(0), true), 0)
					default:
						value = b.Extract(b.Select([]*SelectState{{Chan: fn.Param(0)}}, mode == "select"), 2)
					}
					b.Return(value)
					b.EndBuild()
					if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
					ir := fn.impl.String()
					load := strings.Index(ir, "load volatile")
					restore := strings.Index(ir, "call void @llvm.stackrestore")
					if restore < 0 {
						t.Fatalf("missing stack restoration:\n%s", ir)
					}
					if prog.SizeOf(prog.Type(elem, GoDecl)) == 0 {
						// Comma-ok and select wrap the constant in a tuple with
						// dynamic fields, so extraction need not be folded yet.
						if load >= 0 || (mode == "direct" && !value.impl.IsConstant()) {
							t.Fatalf("zero-sized receive must use a constant without a volatile load:\n%s", ir)
						}
					} else if load < 0 || load > restore {
						t.Fatalf("receive buffer must be loaded before restoring the stack:\n%s", ir)
					}
				})
			}
		})
	}
}
