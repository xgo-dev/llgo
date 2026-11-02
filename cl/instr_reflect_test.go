package cl

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestReflectTypeMethodCheckRecordsDemands(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackageEx("pkg", "pkg", true)
	fn := pkg.NewFunc("pkg.caller", types.NewSignatureType(nil, nil, nil, nil, nil, false), llssa.GoDecl)
	ctx := &context{pkg: pkg, fn: fn}

	reflectPkg := types.NewPackage("reflect", "reflect")
	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()
	reflectType := types.NewNamed(types.NewTypeName(token.NoPos, reflectPkg, "Type", nil), iface, nil)
	recv := ssa.NewConst(nil, reflectType)
	method := func(name string) *types.Func {
		return types.NewFunc(token.NoPos, reflectPkg, name,
			types.NewSignatureType(nil, nil, nil, nil, nil, false))
	}

	check := ctx.reflectTypeMethodCheck(&ssa.CallCommon{
		Value: ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
	}, method("Method"))
	if check != (llssa.ReflectMethodCheck{}) {
		t.Fatalf("non-reflect receiver check = %+v", check)
	}

	otherPkg := types.NewPackage("other", "other")
	otherMethod := types.NewFunc(token.NoPos, otherPkg, "Method",
		types.NewSignatureType(nil, nil, nil, nil, nil, false))
	check = ctx.reflectTypeMethodCheck(&ssa.CallCommon{Value: recv}, otherMethod)
	if check != (llssa.ReflectMethodCheck{}) {
		t.Fatalf("non-reflect method check = %+v", check)
	}

	check = ctx.reflectTypeMethodCheck(&ssa.CallCommon{Value: recv}, method("Method"))
	if check != (llssa.ReflectMethodCheck{}) {
		t.Fatalf("Method without index check = %+v", check)
	}
	check = ctx.reflectTypeMethodCheck(&ssa.CallCommon{Value: recv}, method("MethodByName"))
	if check != (llssa.ReflectMethodCheck{}) {
		t.Fatalf("MethodByName without name check = %+v", check)
	}

	check = ctx.reflectTypeMethodCheck(&ssa.CallCommon{
		Value: recv,
		Args:  []ssa.Value{ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])},
	}, method("Method"))
	if check.Kind != llssa.ReflectTypeMethodByIndex {
		t.Fatalf("constant Method check = %+v", check)
	}

	check = ctx.reflectTypeMethodCheck(&ssa.CallCommon{
		Value: recv,
		Args:  []ssa.Value{&ssa.Parameter{}},
	}, method("Method"))
	if check.Kind != llssa.ReflectTypeMethodDynamic {
		t.Fatalf("dynamic Method check = %+v", check)
	}

	check = ctx.reflectTypeMethodCheck(&ssa.CallCommon{
		Value: recv,
		Args:  []ssa.Value{ssa.NewConst(constant.MakeString("Keep"), types.Typ[types.String])},
	}, method("MethodByName"))
	if check.Kind != llssa.ReflectTypeMethodByName || check.Name != "Keep" {
		t.Fatalf("constant MethodByName check = %+v", check)
	}

	check = ctx.reflectTypeMethodCheck(&ssa.CallCommon{
		Value: recv,
		Args:  []ssa.Value{&ssa.Parameter{}},
	}, method("MethodByName"))
	if check.Kind != llssa.ReflectTypeMethodDynamic|llssa.ReflectTypeMethodByName || check.Name != "" {
		t.Fatalf("dynamic MethodByName check = %+v", check)
	}

	if err := pkg.FinishMetaCollection(); err != nil {
		t.Fatal(err)
	}
	pm := pkg.Meta
	defer pm.Close()

	const want = `[UseNamedMethod]
pkg.caller:
    Keep

[Reflect]
    pkg.caller

`
	if got := pm.String(); got != want {
		t.Fatalf("metadata mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestReflectValueMethodInvokeDemands(t *testing.T) {
	reflectPkg := types.NewPackage("reflect", "reflect")
	value := types.NewNamed(types.NewTypeName(token.NoPos, reflectPkg, "Value", nil), types.NewStruct(nil, nil), nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "V", nil), value)
	for _, tt := range []struct {
		name, method string
		result       types.Type
		args         []ssa.Value
		want         string
	}{
		{"index", "Method", value, []ssa.Value{ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])}, "[Reflect]\n    pkg.caller\n\n"},
		{"alias", "Method", alias, []ssa.Value{&ssa.Parameter{}}, "[Reflect]\n    pkg.caller\n\n"},
		{"name", "MethodByName", alias, []ssa.Value{ssa.NewConst(constant.MakeString("Keep"), types.Typ[types.String])}, "[UseNamedMethod]\npkg.caller:\n    Keep\n\n"},
		{"dynamic name", "MethodByName", value, []ssa.Value{&ssa.Parameter{}}, "[Reflect]\n    pkg.caller\n\n"},
		{"other method", "Other", value, []ssa.Value{&ssa.Parameter{}}, ""},
		{"other result", "Method", types.Typ[types.Int], []ssa.Value{&ssa.Parameter{}}, ""},
		{"no argument", "Method", value, nil, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prog := llssa.NewProgram(nil)
			defer prog.Dispose()
			pkg := prog.NewPackageEx("pkg", "pkg", true)
			fn := pkg.NewFunc("pkg.caller", types.NewSignatureType(nil, nil, nil, nil, nil, false), llssa.GoDecl)
			ctx := &context{pkg: pkg, fn: fn}
			result := types.NewTuple(types.NewVar(token.NoPos, nil, "", tt.result))
			method := types.NewFunc(token.NoPos, nil, tt.method, types.NewSignatureType(nil, nil, nil, nil, result, false))
			ctx.recordReflectValueMethodInvoke(&ssa.CallCommon{Args: tt.args}, method)
			if err := pkg.FinishMetaCollection(); err != nil {
				t.Fatal(err)
			}
			defer pkg.Meta.Close()
			if got := pkg.Meta.String(); got != tt.want {
				t.Fatalf("metadata = %q, want %q", got, tt.want)
			}
		})
	}
}
