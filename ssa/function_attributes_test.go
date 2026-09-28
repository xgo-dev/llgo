package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestBackendFunctionInitializer(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	runtime := types.NewPackage(PkgRuntime, "runtime")
	halt := types.NewFunc(token.NoPos, runtime, "Halt", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	runtime.Scope().Insert(halt)
	prog.SetRuntime(runtime)
	pkg := prog.NewPackage("caller", "caller")
	// A preexisting declaration must also receive properties when its source
	// becomes known through a backend-generated runtime call.
	entry := pkg.NewFunc(PkgRuntime+".Halt", halt.Type().(*types.Signature), InGo)
	var sources []*types.Func
	pkg.SetFunctionInitializer(func(fn Function, source *types.Func) {
		sources = append(sources, source)
		fn.SetCold()
		fn.SetNoReturn()
	})
	pkg.RuntimeFunc("Halt")
	check := func(fn Function) {
		for _, name := range []string{"cold", "noreturn"} {
			if fn.impl.GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil() {
				t.Errorf("%s lost %s", fn.Name(), name)
			}
		}
	}
	check(entry)
	if len(sources) != 1 || sources[0] != halt {
		t.Fatal("runtime entry lost its source function")
	}
	owner := types.NewPackage("p", "p")
	named := types.NewNamed(types.NewTypeName(0, owner, "T", nil), types.NewStruct(nil, nil), nil)
	recv := types.NewVar(0, owner, "", named)
	method := types.NewFunc(0, owner, "Stop", types.NewSignatureType(recv, nil, nil, nil, nil, false))
	named.AddMethod(method)
	fn := pkg.NewFunc("caller", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)
	b.abiMethodFuncs(named, types.NewMethodSet(named).At(0))
	check(pkg.FuncOf("p.T.Stop"))
	check(pkg.FuncOf("p.(*T).Stop"))
	if len(sources) != 3 || sources[1] != method || sources[2] != method {
		t.Fatal("method entry lost its source function")
	}
}
