package ssa

import (
	"go/token"
	"go/types"
	"testing"
	"unsafe"
)

var typeCvtTestPkg = types.NewPackage("github.com/goplus/llgo/ssa/typecvttest", "typecvttest")

func newTypeCvtNamed(name string, underlying types.Type) *types.Named {
	return types.NewNamed(types.NewTypeName(token.NoPos, typeCvtTestPkg, name, nil), underlying, nil)
}

func newTypeCvtMethod(name string, param types.Type) *types.Func {
	params := types.NewTuple(types.NewParam(token.NoPos, typeCvtTestPkg, "v", param))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	return types.NewFunc(token.NoPos, typeCvtTestPkg, name, sig)
}

func newTypeCvtNamedPair(gt goTypes, oldName, rawName string, underlying types.Type) (*types.Named, *types.Named) {
	old := newTypeCvtNamed(oldName, underlying)
	raw := newTypeCvtNamed(rawName, underlying)
	gt.typs[unsafe.Pointer(old)] = unsafe.Pointer(raw)
	return old, raw
}

func TestCvtTupleCopiesPrefixWhenLaterParamConverts(t *testing.T) {
	gt := newGoTypes()
	old, raw := newTypeCvtNamedPair(gt, "OldTuple", "RawTuple", types.Typ[types.Int])
	keep := types.NewParam(token.NoPos, typeCvtTestPkg, "keep", types.Typ[types.String])
	convert := types.NewParam(token.NoPos, typeCvtTestPkg, "convert", old)
	tuple := types.NewTuple(keep, convert)

	got, converted := gt.cvtTuple(tuple)
	if !converted {
		t.Fatal("cvtTuple did not report conversion")
	}
	if got.At(0) != keep {
		t.Fatalf("cvtTuple did not preserve prefix var: got %v, want %v", got.At(0), keep)
	}
	if got.At(1).Type() != raw {
		t.Fatalf("cvtTuple converted type = %v, want %v", got.At(1).Type(), raw)
	}
}

func TestCvtInterfaceConvertsExplicitMethodsAndKeepsEmbeddeds(t *testing.T) {
	gt := newGoTypes()
	oldParam, rawParam := newTypeCvtNamedPair(gt, "OldParam", "RawParam", types.Typ[types.Int])
	plainMethod := newTypeCvtMethod("APlain", types.Typ[types.Int])
	convertedMethod := newTypeCvtMethod("ZConverted", oldParam)
	plainEmbedded := types.NewInterfaceType(nil, nil)
	plainEmbedded.Complete()
	iface := types.NewInterfaceType([]*types.Func{plainMethod, convertedMethod}, []types.Type{plainEmbedded})
	iface.Complete()

	methods, converted := gt.cvtExplicitMethods(iface)
	if !converted {
		t.Fatal("cvtExplicitMethods did not report conversion")
	}
	if methods[0] != plainMethod {
		t.Fatalf("cvtExplicitMethods did not preserve prefix method: got %v, want %v", methods[0], plainMethod)
	}
	gotSig := methods[1].Type().(*types.Signature)
	if gotSig.Params().At(0).Type() != rawParam {
		t.Fatalf("converted method param = %v, want %v", gotSig.Params().At(0).Type(), rawParam)
	}

	got, converted := gt.cvtInterface(iface)
	if !converted {
		t.Fatal("cvtInterface did not report method conversion")
	}
	if got.EmbeddedType(0) != plainEmbedded {
		t.Fatalf("cvtInterface did not keep unchanged embedded type: got %v, want %v", got.EmbeddedType(0), plainEmbedded)
	}
}

func TestCvtInterfaceConvertsEmbeddedTypesAndKeepsMethods(t *testing.T) {
	gt := newGoTypes()
	embeddedUnderlying := types.NewInterfaceType(nil, nil)
	embeddedUnderlying.Complete()
	oldEmbedded, rawEmbedded := newTypeCvtNamedPair(gt, "OldEmbedded", "RawEmbedded", embeddedUnderlying)
	plainEmbedded := types.NewInterfaceType(nil, nil)
	plainEmbedded.Complete()
	plainMethod := newTypeCvtMethod("Plain", types.Typ[types.Int])
	iface := types.NewInterfaceType([]*types.Func{plainMethod}, []types.Type{plainEmbedded, oldEmbedded})
	iface.Complete()

	embeddeds, converted := gt.cvtEmbeddedTypes(iface)
	if !converted {
		t.Fatal("cvtEmbeddedTypes did not report conversion")
	}
	if embeddeds[0] != plainEmbedded {
		t.Fatalf("cvtEmbeddedTypes did not preserve prefix embedded type: got %v, want %v", embeddeds[0], plainEmbedded)
	}
	if embeddeds[1] != rawEmbedded {
		t.Fatalf("converted embedded type = %v, want %v", embeddeds[1], rawEmbedded)
	}

	got, converted := gt.cvtInterface(iface)
	if !converted {
		t.Fatal("cvtInterface did not report embedded conversion")
	}
	if got.ExplicitMethod(0) != plainMethod {
		t.Fatalf("cvtInterface did not keep unchanged method: got %v, want %v", got.ExplicitMethod(0), plainMethod)
	}
	if got.EmbeddedType(1) != rawEmbedded {
		t.Fatalf("cvtInterface embedded type = %v, want %v", got.EmbeddedType(1), rawEmbedded)
	}
}
