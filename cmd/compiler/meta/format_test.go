package meta

import "testing"

func TestPackageMetaString(t *testing.T) {
	b := NewBuilder()
	main := b.Sym("pkg.main")
	helper := b.Sym("pkg.helper")
	typ := b.Sym("_llgo_pkg.T")
	child := b.Sym("_llgo_pkg.Child")
	iface := b.Sym("_llgo_iface$Reader")
	mtype := b.Sym("_llgo_func$Read")
	ifn := b.Sym("pkg.(*T).Read")
	tfn := b.Sym("pkg.T.Read")
	b.AddOrdinaryEdge(main, helper)
	b.AddIfaceUse(main, typ)
	b.AddIfaceMethodUse(main, iface, 0)
	b.AddNamedMethodUse(helper, "Keep")
	b.AddTypeChild(typ, child)
	b.AddMethodSlot(typ, "Read", mtype, ifn, tfn)
	b.AddIfaceMethod(iface, "Read", mtype)
	b.MarkReflect(helper)

	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()

	const want = `[TypeChildren]
_llgo_pkg.T:
    _llgo_pkg.Child

[OrdinaryEdges]
pkg.main:
    pkg.helper

[UseIface]
pkg.main:
    _llgo_pkg.T

[UseIfaceMethod]
pkg.main:
    _llgo_iface$Reader Read _llgo_func$Read

[UseNamedMethod]
pkg.helper:
    Keep

[MethodInfo]
_llgo_pkg.T:
    0 Read _llgo_func$Read pkg.(*T).Read pkg.T.Read

[InterfaceInfo]
_llgo_iface$Reader:
    Read _llgo_func$Read

[Reflect]
    pkg.helper

`
	if got := pm.String(); got != want {
		t.Fatalf("metadata mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}
