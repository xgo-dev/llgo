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
	source := prog.DeclareFunction(PkgRuntime + ".Stop")
	source.SetAttributes(FunctionCold | FunctionNoReturn)
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
			prog.DeclareFunction("p.Rare").SetAttributes(FunctionCold)
			prog.DeclareFunction("p.Stop").SetAttributes(FunctionNoReturn)
			prog.DeclareFunction("p.Indirect").SetAttributes(FunctionNoReturn)
			if !linksFirst {
				setLinks()
			}
			pkg := prog.NewPackage("caller", "caller")
			for _, test := range []struct {
				name string
				want FunctionAttributes
			}{
				{"shared", FunctionCold | FunctionNoReturn},
				// One linkname mapping does not imply a transitive alias chain.
				{"p.Rare", FunctionNoReturn},
				{"p.Unknown", 0},
			} {
				fn := pkg.NewFunc(test.name, NoArgsNoRet, InGo)
				if got := fn.Attributes(); got != test.want {
					t.Errorf("linksFirst=%v, %s: got %v, want %v", linksFirst, test.name, got, test.want)
				}
			}
		})
	}
}

func TestFunctionAttributesFollowFunctionObject(t *testing.T) {
	coordinator := NewProgram(nil)
	defer coordinator.Dispose()
	source := coordinator.DeclareFunction("owner.Stop")
	source.SetAttributes(FunctionCold)
	coordinator.SetLinkname("owner.Stop", "shared_stop")
	for i := 0; i < 2; i++ {
		backend := coordinator.NewBackendProgram()
		defer backend.Dispose()
		pkg := backend.NewPackage("caller", "caller")
		pending := pkg.fns["shared_stop"]
		if pending == nil || pending == source || pending.Name() != "shared_stop" {
			t.Fatal("expected a module-local function object")
		}
		if pkg.FuncOf("shared_stop") != nil || !pkg.Module().FirstFunction().IsNil() {
			t.Fatal("preloading must not emit an LLVM declaration")
		}
		fn := pkg.NewFunc("shared_stop", NoArgsNoRet, InGo)
		if fn != pending || pkg.FuncOf("shared_stop") != fn {
			t.Fatal("materialization replaced the function object")
		}
		if fn.Attributes() != FunctionCold {
			t.Fatal("module-local changes leaked into another backend")
		}
		fn.SetAttributes(FunctionNoReturn)
		if fn.Attributes() != FunctionCold|FunctionNoReturn {
			t.Fatal("attributes are not carried by the function object")
		}
		for _, name := range []string{"cold", "noreturn"} {
			if fn.impl.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(name)).IsNil() {
				t.Fatalf("function lost %s during materialization or update", name)
			}
		}
		if pkg.NewFunc("shared_stop", NoArgsNoRet, InGo) != fn {
			t.Fatal("redeclaration replaced the function object")
		}
		if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
	if source.Name() != "owner.Stop" || source.Attributes() != FunctionCold {
		t.Fatal("materialization mutated the shared source declaration")
	}
}
