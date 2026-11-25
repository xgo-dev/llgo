package llvmgen

import (
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"testing"

	"github.com/xgo-dev/llvm"
)

type nativeStringTestCase struct {
	name  string
	typ   types.Type
	paths [][]int
}

func nativeStringTestCases() []nativeStringTestCase {
	pkg := types.NewPackage("example.com/stringtypes", "stringtypes")
	named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Text", nil), types.Typ[types.String], nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "Alias", nil), named)
	nested := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Values", types.NewArray(alias, 2), false),
	}, nil)
	return []nativeStringTestCase{
		{"string", types.Typ[types.String], [][]int{nil}},
		{"alias", alias, [][]int{nil}},
		{"array", types.NewArray(named, 2), [][]int{{0}, {1}}},
		{"nested", nested, [][]int{{0, 0}, {0, 1}}},
	}
}

func testWasmStringProfiles(t *testing.T, shape string, test func(*testing.T, Program, Package)) {
	t.Helper()
	for _, target := range []Target{
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
	} {
		t.Run(shape+"/"+target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(&target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			test(t, prog, prog.NewPackage("p", "example.com/p"))
		})
	}
}

func stringAtPath(b Builder, value Expr, path []int) Expr {
	part := value.impl
	for _, index := range path {
		part = b.impl.CreateExtractValue(part, index, "")
	}
	return Expr{part, b.Prog.String()}
}

// Read the entire high pointer word, as a Go unsafe string-header observation
// can do. Ordinary string operations inspect only the physical low pointer.
func nativeStringHighWords(b Builder, value Expr, paths [][]int) Expr {
	result := llvm.ConstInt(b.Prog.tyInt32(), 0, false)
	for _, path := range paths {
		text := stringAtPath(b, value, path)
		pointer := b.impl.CreateExtractValue(text.impl, 0, "")
		high := b.impl.CreateExtractValue(pointer, 1, "")
		result = b.impl.CreateOr(result, high, "")
	}
	return Expr{result, b.Prog.Uint32()}
}

func checkNativeStringZeroResults(t *testing.T, pkg Package, observers ...llvm.Value) {
	t.Helper()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid string adapter: %v\n%s", err, pkg.String())
	}
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	if err := pkg.Module().RunPasses("default<O2>", pkg.Prog.TargetMachine(), options); err != nil {
		t.Fatal(err)
	}
	for _, observer := range observers {
		returns := 0
		for block := observer.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			ret := block.LastInstruction()
			if ret.IsNil() || ret.InstructionOpcode() != llvm.Ret {
				continue
			}
			returns++
			if ret.OperandsCount() != 1 || !ret.Operand(0).IsNull() {
				t.Errorf("string padding, payload, or callback identity changed in %s:\n%s", observer.Name(), observer)
			}
		}
		if returns == 0 {
			t.Errorf("observer %s has no return sites", observer.Name())
		}
	}
}

func TestWasmStringCallbacks(t *testing.T) {
	for _, tc := range nativeStringTestCases() {
		testWasmStringProfiles(t, tc.name, func(t *testing.T, prog Program, pkg Package) {
			text := types.NewParam(token.NoPos, nil, "text", tc.typ)
			result := types.NewParam(token.NoPos, nil, "", types.Typ[types.Uint32])
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(text), types.NewTuple(result, text), false)
			choices := []Function{
				pkg.NewFunc(pkg.Path()+".first", sig, GoDecl),
				pkg.NewFunc(pkg.Path()+".second", sig, GoDecl),
			}
			for _, choice := range choices {
				choice.Inline(AlwaysInline)
				body := choice.MakeBody(1)
				value := choice.Param(0)
				body.Return(nativeStringHighWords(body, value, tc.paths), value)
				body.EndBuild()
				body.Dispose()
			}
			// Read native storage through a pointer so neither the observer's nor
			// the selector's entry can normalize the value before the callback.
			storage := types.NewParam(token.NoPos, nil, "storage", types.NewPointer(tc.typ))
			ownerSig := types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "which", types.Typ[types.Bool]), storage), types.NewTuple(result), false)
			owner := pkg.NewFunc(pkg.Path()+".owner", ownerSig, GoDecl)
			owner.Inline(AlwaysInline)
			body := owner.MakeBody(1)
			defer body.Dispose()
			alias := types.NewAlias(types.NewTypeName(token.NoPos, types.NewPackage(pkg.Path(), "p"), "Callback", nil), sig)
			prog.SetTypeCallConv(pkg.Path()+".Callback", CDecl)
			dst, closure := prog.Type(alias, GoDecl), prog.Type(sig, GoDecl)
			values := make([]Expr, len(choices))
			for i, choice := range choices {
				values[i] = body.ChangeType(closure, choice.Expr)
				if body.Field(values[i], 0).impl != choice.impl ||
					body.ChangeType(dst, choice.Expr).impl != choice.impl ||
					body.ChangeType(dst, values[i]).impl != choice.impl ||
					checkExpr(choice.Expr, alias, body).impl != choice.impl {
					t.Fatal("Go funcval or callback conversion changed its code pointer")
				}
			}
			if !body.ChangeType(dst, prog.Nil(closure)).impl.IsNull() {
				t.Fatal("nil Go callback became non-nil")
			}
			selected := Expr{body.impl.CreateSelect(owner.Param(0).impl, values[0].impl, values[1].impl, ""), closure}
			callback := body.ChangeType(dst, selected)
			// Read the incoming bytes as C does. A typed Go load would already
			// clear the high word and hide a missing callback entry conversion.
			inputType := prog.Elem(owner.Param(1).Type)
			input := Expr{llvm.CreateLoad(body.impl, inputType.ll, owner.Param(1).impl), inputType}
			returned := body.Call(callback, input)
			padding, payload := body.Extract(returned, 0), body.Extract(returned, 1)
			changed := llvm.CreateICmp(body.impl, llvm.IntNE, callback.impl, body.Field(selected, 0).impl)
			for _, path := range tc.paths {
				before, after := stringAtPath(body, input, path), stringAtPath(body, payload, path)
				pointerChanged := llvm.CreateICmp(body.impl, llvm.IntNE, body.StringData(before).impl, body.StringData(after).impl)
				lengthChanged := llvm.CreateICmp(body.impl, llvm.IntNE, body.StringLen(before).impl, body.StringLen(after).impl)
				changed = body.impl.CreateOr(changed, body.impl.CreateOr(pointerChanged, lengthChanged, ""), "")
			}
			status := body.impl.CreateOr(padding.impl, body.impl.CreateZExt(changed, prog.tyInt32(), ""), "")
			body.Return(Expr{status, prog.Uint32()})
			body.EndBuild()
			// Both runtime-selected entries must clear padding and return the
			// original pointer/length fields without changing their code address.
			// Exercise each selector so LLVM can evaluate the selected call.
			var observers []llvm.Value
			probeSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(storage), types.NewTuple(result), false)
			for i, choice := range choices {
				probe := pkg.NewFunc(choice.Name()+".probe", probeSig, GoDecl)
				body := probe.MakeBody(1)
				body.Return(body.Call(owner.Expr, prog.BoolVal(i == 0), probe.Param(0)))
				body.EndBuild()
				body.Dispose()
				observers = append(observers, probe.impl)
			}
			checkNativeStringZeroResults(t, pkg, observers...)
		})
	}
}

func TestWasmStringNativeFuncvalAdapters(t *testing.T) {
	for _, tc := range nativeStringTestCases() {
		testWasmStringProfiles(t, tc.name, func(t *testing.T, prog Program, pkg Package) {
			sig := types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "", tc.typ)), false)
			// An imported result has unknown padding. Keep this declaration
			// opaque so only the adapter can make the observation zero.
			native := pkg.NewFunc("native", sig, CDecl)
			dst := prog.Type(sig, GoDecl)
			ownerSig := types.NewSignatureType(nil, nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "", types.Typ[types.Uint32])), false)
			var observers []llvm.Value
			for _, conversion := range []string{"declaration", "pointer", "implicit"} {
				owner := pkg.NewFunc(pkg.Path()+"."+conversion, ownerSig, GoDecl)
				body := owner.MakeBody(1)
				value := native.Expr
				if conversion == "pointer" {
					value = Expr{native.impl, prog.rawType(sig)}
				}
				var converted Expr
				if conversion == "implicit" {
					converted = checkExpr(value, dst.raw.Type, body)
				} else {
					converted = body.ChangeType(dst, value)
				}
				if conversion == "declaration" {
					nilFunc := body.ChangeType(dst, prog.Nil(prog.rawType(sig)))
					if !body.Field(nilFunc, 0).impl.IsNull() || !body.Field(nilFunc, 1).impl.IsNull() {
						t.Fatal("nil native function became non-nil")
					}
				}
				body.Return(nativeStringHighWords(body, body.Call(converted), tc.paths))
				body.EndBuild()
				body.Dispose()
				observers = append(observers, owner.impl)
			}
			checkNativeStringZeroResults(t, pkg, observers...)
		})
	}
}

func TestWasmStringAdapterDirections(t *testing.T) {
	testWasmStringProfiles(t, "string", func(t *testing.T, prog Program, pkg Package) {
		text := types.NewParam(token.NoPos, nil, "", types.Typ[types.String])
		produces := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(text), false)
		consumes := types.NewSignatureType(nil, nil, nil, types.NewTuple(text), nil, false)
		goProducer := pkg.NewFunc(pkg.Path()+".produce", produces, GoDecl)
		producerBody := goProducer.MakeBody(1)
		producerBody.Return(prog.Zero(prog.String()))
		producerBody.EndBuild()
		producerBody.Dispose()
		nativeConsumer := pkg.NewFunc("consume", consumes, CDecl)
		owner := pkg.NewFunc(pkg.Path()+".owner", NoArgsNoRet, GoDecl)
		body := owner.MakeBody(1)
		defer body.Dispose()
		if got := body.ChangeType(prog.rawType(produces), goProducer.Expr); got.impl != goProducer.impl {
			t.Fatal("Go callback returning a string acquired an unnecessary adapter")
		}
		converted := body.ChangeType(prog.Type(consumes, GoDecl), nativeConsumer.Expr)
		if body.Field(converted, 0).impl != nativeConsumer.impl {
			t.Fatal("native function consuming strings acquired an unnecessary funcval adapter")
		}
		if got := body.ChangeType(prog.rawType(consumes), nativeConsumer.Expr); got.impl != nativeConsumer.impl {
			t.Fatal("native-to-native conversion acquired an unnecessary adapter")
		}
		body.Return()
		body.EndBuild()
		if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatalf("invalid directional string adapter: %v\n%s", err, pkg.String())
		}
	})
}
