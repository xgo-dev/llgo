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
	fn := pkg.NewFunc("pkg.caller", types.NewSignatureType(nil, nil, nil, nil, nil, false), llssa.InGo)
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
