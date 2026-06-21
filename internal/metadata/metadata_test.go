package metadata

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestBuilderSeparatesSymbolAndNameReferences(t *testing.T) {
	b := NewBuilder()

	mainSym := b.Symbol("main")
	mainName := b.Name("main")
	readName := b.Name("Read")
	fnType := b.Symbol("_llgo_func$A1")
	typ := b.Symbol("*reader")
	ifn := b.Symbol("(*reader).Read$iface")
	tfn := b.Symbol("(*reader).Read")

	if uint32(mainSym) != uint32(mainName) {
		t.Fatalf("shared string table should allow equal backing IDs for equal text: Symbol=%d Name=%d", mainSym, mainName)
	}

	b.AddEdge(mainSym, tfn)
	b.AddUseNamedMethod(mainSym, []Name{mainName, readName})
	b.AddMethodInfo(typ, []MethodSlot{{
		Sig: MethodSig{Name: readName, MType: fnType},
		IFn: ifn,
		TFn: tfn,
	}})

	pm := b.Build()
	if got := pm.SymbolName(mainSym); got != "main" {
		t.Fatalf("SymbolName(main) = %q, want main", got)
	}
	if got := pm.Name(mainName); got != "main" {
		t.Fatalf("Name(main) = %q, want main", got)
	}
	slots := collectMethodSlots(pm, typ)
	if got := pm.Name(slots[0].Sig.Name); got != "Read" {
		t.Fatalf("method name = %q, want Read", got)
	}
	if got := pm.SymbolName(slots[0].Sig.MType); got != "_llgo_func$A1" {
		t.Fatalf("method type = %q, want _llgo_func$A1", got)
	}
}

func TestWriteReadMetaRoundTrip(t *testing.T) {
	pm := buildFullTestMeta()

	var buf bytes.Buffer
	if err := pm.WriteMeta(&buf); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}

	got, err := ReadMeta(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}

	if !reflect.DeepEqual(got, pm) {
		t.Fatalf("round trip mismatch\ngot:  %#v\nwant: %#v", got, pm)
	}

	var buf2 bytes.Buffer
	if err := got.WriteMeta(&buf2); err != nil {
		t.Fatalf("WriteMeta after round trip: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
		t.Fatalf("WriteMeta should be deterministic across round trip")
	}
}

func TestReadMetaRejectsInvalidFiles(t *testing.T) {
	if _, err := ReadMeta(strings.NewReader("NOPE")); err == nil {
		t.Fatal("ReadMeta accepted bad magic")
	}

	var buf bytes.Buffer
	pm := buildFullTestMeta()
	if err := pm.WriteMeta(&buf); err != nil {
		t.Fatal(err)
	}
	data := append([]byte(nil), buf.Bytes()...)
	data[4] = 99
	if _, err := ReadMeta(bytes.NewReader(data)); err == nil {
		t.Fatal("ReadMeta accepted unsupported version")
	}
}

func TestFormatMetaStableOutput(t *testing.T) {
	got := MetaString(buildFullTestMeta())
	want := `[TypeChildren]
*_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to:
    _llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to
*_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Inner:
    _llgo_github.com/goplus/llgo/cl/_testmeta/nested.Inner
*_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Outer:
    _llgo_github.com/goplus/llgo/cl/_testmeta/nested.Outer
*_llgo_int:
    _llgo_int
*_llgo_string:
    _llgo_string
_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to:
    _llgo_string
_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Inner:
    _llgo_string
_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Outer:
    _llgo_github.com/goplus/llgo/cl/_testmeta/nested.Inner
    _llgo_int

[InterfaceInfo]
_llgo_iface$f14WsslTA1u5wwC83jLU0HU2u2mmAWxBVE38vPBbRAo:
    M _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac
    N _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac

[OrdinaryEdges]
github.com/goplus/llgo/cl/_testmeta/interface_anonymous.main:
    github.com/goplus/llgo/cl/_testmeta/interface_anonymous.use

[UseIface]
github.com/goplus/llgo/cl/_testmeta/interface_anonymous.main:
    _llgo_github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T

[UseIfaceMethod]
github.com/goplus/llgo/cl/_testmeta/interface_anonymous.use:
    _llgo_iface$f14WsslTA1u5wwC83jLU0HU2u2mmAWxBVE38vPBbRAo M _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac
    _llgo_iface$f14WsslTA1u5wwC83jLU0HU2u2mmAWxBVE38vPBbRAo N _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac

[MethodInfo]
*_llgo_github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T:
    0 M _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).M github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).M
    1 N _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).N github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).N
_llgo_github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T:
    0 M _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).M github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T.M
    1 N _llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).N github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T.N

[UseNamedMethod]
github.com/goplus/llgo/cl/_testmeta/interface_anonymous.use:
    M

[ReflectMethod]
github.com/goplus/llgo/cl/_testmeta/interface_anonymous.use

`
	if got != want {
		t.Fatalf("MetaString mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func buildFullTestMeta() *PackageMeta {
	b := NewBuilder()

	nestedFuncPtr := b.Symbol("*_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to")
	nestedFunc := b.Symbol("_llgo_func$zNDVRsWTIpUPKouNUS805RGX--IV9qVK8B31IZbg5to")
	nestedInnerPtr := b.Symbol("*_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Inner")
	nestedInner := b.Symbol("_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Inner")
	nestedOuterPtr := b.Symbol("*_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Outer")
	nestedOuter := b.Symbol("_llgo_github.com/goplus/llgo/cl/_testmeta/nested.Outer")
	intPtr := b.Symbol("*_llgo_int")
	intType := b.Symbol("_llgo_int")
	stringPtr := b.Symbol("*_llgo_string")
	stringType := b.Symbol("_llgo_string")

	b.AddTypeChild(nestedFuncPtr, nestedFunc)
	b.AddTypeChild(nestedInnerPtr, nestedInner)
	b.AddTypeChild(nestedOuterPtr, nestedOuter)
	b.AddTypeChild(intPtr, intType)
	b.AddTypeChild(stringPtr, stringType)
	b.AddTypeChild(nestedFunc, stringType)
	b.AddTypeChild(nestedInner, stringType)
	b.AddTypeChild(nestedOuter, nestedInner)
	b.AddTypeChild(nestedOuter, intType)

	main := b.Symbol("github.com/goplus/llgo/cl/_testmeta/interface_anonymous.main")
	use := b.Symbol("github.com/goplus/llgo/cl/_testmeta/interface_anonymous.use")
	iface := b.Symbol("_llgo_iface$f14WsslTA1u5wwC83jLU0HU2u2mmAWxBVE38vPBbRAo")
	typ := b.Symbol("_llgo_github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T")
	ptrTyp := b.Symbol("*_llgo_github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T")
	methodType := b.Symbol("_llgo_func$2_iS07vIlF2_rZqWB5eU0IvP_9HviM4MYZNkXZDvbac")
	mName := b.Name("M")
	nName := b.Name("N")
	mSig := MethodSig{Name: mName, MType: methodType}
	nSig := MethodSig{Name: nName, MType: methodType}
	ptrM := b.Symbol("github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).M")
	ptrN := b.Symbol("github.com/goplus/llgo/cl/_testmeta/interface_anonymous.(*T).N")
	valM := b.Symbol("github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T.M")
	valN := b.Symbol("github.com/goplus/llgo/cl/_testmeta/interface_anonymous.T.N")

	b.AddEdge(main, use)
	b.AddIfaceEntry(iface, []MethodSig{mSig, nSig})
	b.AddUseIface(main, []Symbol{typ})
	b.AddUseIfaceMethod(use, []IfaceMethodDemand{
		{Target: iface, Sig: mSig},
		{Target: iface, Sig: nSig},
	})
	b.AddMethodInfo(ptrTyp, []MethodSlot{
		{Sig: mSig, IFn: ptrM, TFn: ptrM},
		{Sig: nSig, IFn: ptrN, TFn: ptrN},
	})
	b.AddMethodInfo(typ, []MethodSlot{
		{Sig: mSig, IFn: ptrM, TFn: valM},
		{Sig: nSig, IFn: ptrN, TFn: valN},
	})
	b.AddUseNamedMethod(use, []Name{mName})
	b.AddReflectMethod(use)

	return b.Build()
}

func collectMethodSlots(pm *PackageMeta, want Symbol) []MethodSlot {
	var got []MethodSlot
	pm.ForEachMethodInfo(func(typ Symbol, slots []MethodSlot) {
		if typ == want {
			got = append(got, slots...)
		}
	})
	return got
}
