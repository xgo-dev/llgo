package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestFunctionAttributesOnRuntimeDeclarations(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	runtime := types.NewPackage(PkgRuntime, "runtime")
	runtime.Scope().Insert(types.NewFunc(token.NoPos, runtime, "Stop", NoArgsNoRet))
	prog.SetRuntime(runtime)
	prog.DeclareSourceFunction(PkgRuntime+".Stop", SourceFunction{Attributes: FunctionCold | FunctionNoReturn})
	for _, environment := range []bool{false, true} {
		backend := prog.NewBackendProgram()
		defer backend.Dispose()
		pkg := backend.NewPackage("caller", "caller")
		var fn Expr
		if environment {
			fn = pkg.rtEnvFunc("Stop")
		} else {
			fn = pkg.rtFunc("Stop")
		}
		for _, name := range []string{"cold", "noreturn"} {
			if fn.impl.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(name)).IsNil() {
				t.Fatalf("environment=%v: runtime declaration lost %s", environment, name)
			}
		}
		if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFunctionAttributeSymbolResolution(t *testing.T) {
	for _, order := range []string{"links first", "attributes first"} {
		t.Run(order, func(t *testing.T) {
			linksFirst := order == "links first"
			prog := NewProgram(nil)
			defer prog.Dispose()
			setLinks := func() {
				prog.SetLinkname("p.Rare", "C.shared")
				prog.SetLinkname("p.Stop", "stdcall.shared")
				prog.SetLinkname("p.Indirect", "p.Rare")
			}
			if linksFirst {
				setLinks()
			}
			prog.DeclareSourceFunction("p.Rare", SourceFunction{Attributes: FunctionCold})
			prog.DeclareSourceFunction("p.Stop", SourceFunction{Attributes: FunctionNoReturn})
			prog.DeclareSourceFunction("p.Indirect", SourceFunction{Attributes: FunctionNoReturn})
			if !linksFirst {
				setLinks()
			}
			for _, test := range []struct {
				name string
				want FunctionAttributes
			}{
				{"p.Rare", FunctionCold | FunctionNoReturn},
				{"p.Stop", FunctionCold | FunctionNoReturn},
				{"shared", FunctionCold | FunctionNoReturn},
				// One linkname mapping does not imply a transitive alias chain.
				{"p.Indirect", FunctionNoReturn},
				{"p.Unknown", 0},
			} {
				if got := prog.SourceFunctionAttributes(test.name); got != test.want {
					t.Errorf("linksFirst=%v, %s: got %v, want %v", linksFirst, test.name, got, test.want)
				}
			}
		})
	}
}
