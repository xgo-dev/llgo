/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package exportdata_test

import (
	"bytes"
	"go/constant"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/types"
)

func buildRichTestPackage() (*types.Package, *types.Package) {
	dep := types.NewPackage("example.com/dep", "dep")
	depType := types.NewNamed(types.NewTypeName(0, dep, "Item", nil), types.Typ[types.Int64], nil)
	dep.Scope().Insert(depType.Obj())
	dep.MarkComplete()

	pkg := types.NewPackage("example.com/demo", "demo")
	pkg.SetImports([]*types.Package{dep})

	pkg.Scope().Insert(types.NewConst(0, pkg, "Answer", types.Typ[types.Int], constant.MakeInt64(42)))
	pkg.Scope().Insert(types.NewConst(0, pkg, "Pi", types.Typ[types.Float64], constant.MakeFloat64(3.14)))
	pkg.Scope().Insert(types.NewConst(0, pkg, "Flag", types.Typ[types.Bool], constant.MakeBool(true)))
	pkg.Scope().Insert(types.NewConst(0, pkg, "Tag", types.Typ[types.String], constant.MakeString("llgo")))

	// Recursive struct with CPtr and dep reference.
	nodeObj := types.NewTypeName(0, pkg, "Node", nil)
	nodeNamed := types.NewNamed(nodeObj, nil, nil)
	nodeStruct := types.NewStruct([]*types.Var{
		types.NewField(0, pkg, "Next", types.NewPointerWithKind(nodeNamed, types.GoPtr), false),
		types.NewField(0, pkg, "Raw", types.NewPointerWithKind(types.Typ[types.Uint8], types.CPtr), false),
		types.NewField(0, pkg, "Item", depType, false),
	}, []string{"", `json:"raw"`, ""})
	nodeNamed.SetUnderlying(nodeStruct)
	recv := types.NewVar(0, pkg, "n", types.NewPointer(nodeNamed))
	methSig := types.NewSignatureType(recv, nil, nil, nil, types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.Int])), false)
	nodeNamed.AddMethod(types.NewFunc(0, pkg, "Len", methSig))
	pkg.Scope().Insert(nodeObj)

	// Generic type Box[T any].
	tpName := types.NewTypeName(0, pkg, "T", nil)
	tp := types.NewTypeParam(tpName, types.Universe.Lookup("any").Type())
	boxObj := types.NewTypeName(0, pkg, "Box", nil)
	boxNamed := types.NewNamed(boxObj, types.NewStruct([]*types.Var{
		types.NewField(0, pkg, "Val", tp, false),
	}, nil), nil)
	boxNamed.SetTypeParams([]*types.TypeParam{tp})
	pkg.Scope().Insert(boxObj)

	// Native CDecl function.
	cSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, pkg, "x", types.Typ[types.Int32])),
		types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.Int32])),
		false,
	).WithCallConv(types.CDecl)
	pkg.Scope().Insert(types.NewFunc(0, pkg, "CFunc", cSig))

	// Alias & Var.
	aliasObj := types.NewTypeName(0, pkg, "MyInt", nil)
	_ = types.NewAlias(aliasObj, types.Typ[types.Int32])
	pkg.Scope().Insert(aliasObj)
	pkg.Scope().Insert(types.NewVar(0, pkg, "GlobalCount", types.Typ[types.Int]))

	pkg.MarkComplete()
	return dep, pkg
}

func TestWriteReadDeterministicAndImporter(t *testing.T) {
	fset := token.NewFileSet()
	dep, pkg := buildRichTestPackage()

	var buf1, buf2 bytes.Buffer
	if err := exportdata.Write(&buf1, fset, pkg); err != nil {
		t.Fatalf("Write 1: %v", err)
	}
	if err := exportdata.Write(&buf2, fset, pkg); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Fatal("Write output is not deterministic")
	}

	imports := map[string]*types.Package{dep.Path(): dep}
	got, err := exportdata.Read(bytes.NewReader(buf1.Bytes()), fset, imports, pkg.Path())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Path() != pkg.Path() || got.Name() != pkg.Name() || !got.Complete() {
		t.Fatalf("decoded package header mismatch: %v", got)
	}
	if imports[pkg.Path()] != got {
		t.Fatal("Read did not register package in imports map")
	}

	// Verify recursive Node struct + NativePtr + CDecl preserved.
	nodeObj := got.Scope().Lookup("Node")
	if nodeObj == nil {
		t.Fatal("missing Node")
	}
	nodeNamed := nodeObj.Type().(*types.Named)
	if nodeNamed.NumMethods() != 1 || nodeNamed.Method(0).Name() != "Len" {
		t.Fatalf("Node methods mismatch: %d", nodeNamed.NumMethods())
	}
	st := nodeNamed.Underlying().(*types.Struct)
	if ptr := st.Field(1).Type().(*types.Pointer); ptr.Kind() != types.CPtr {
		t.Fatalf("Raw field PtrKind = %v, want CPtr", ptr.Kind())
	}
	cFunc := got.Scope().Lookup("CFunc").(*types.Func)
	if sig := cFunc.Type().(*types.Signature); sig.CallConv() != types.CDecl {
		t.Fatalf("CFunc CallConv = %v, want CDecl", sig.CallConv())
	}

	// Verify NewImporter (types.ImporterFrom) compatibility.
	var imp types.ImporterFrom = exportdata.NewImporter(fset, imports)
	resolved, err := imp.Import(pkg.Path())
	if err != nil || resolved != got {
		t.Fatalf("Import = (%v, %v), want %v", resolved, err, got)
	}
	resolvedUnsafe, err := imp.Import("unsafe")
	if err != nil || resolvedUnsafe != types.Unsafe {
		t.Fatalf("Import(unsafe) = (%v, %v)", resolvedUnsafe, err)
	}
	if _, err := imp.Import("example.com/missing"); err == nil {
		t.Fatal("expected error for missing package")
	}
	if _, err := imp.ImportFrom(pkg.Path(), "", 1); err == nil {
		t.Fatal("expected error for unsupported import mode")
	}
}

func TestWriteReadBundleAndErrors(t *testing.T) {
	fset := token.NewFileSet()
	dep, pkg := buildRichTestPackage()

	var buf bytes.Buffer
	if err := exportdata.WriteBundle(&buf, fset, []*types.Package{dep, pkg}); err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	imports := make(map[string]*types.Package)
	got, err := exportdata.ReadBundle(bytes.NewReader(buf.Bytes()), fset, imports)
	if err != nil {
		t.Fatalf("ReadBundle: %v", err)
	}
	if len(got) != 2 || got[0].Path() != dep.Path() || got[1].Path() != pkg.Path() {
		t.Fatalf("ReadBundle got %v", got)
	}

	// Corrupt/bad-magic/kind-mismatch checks.
	if _, err := exportdata.Read(bytes.NewReader([]byte("BADMAGIC12345")), fset, nil, ""); err == nil || !strings.Contains(err.Error(), "magic") {
		t.Fatalf("expected invalid magic error, got %v", err)
	}
	if _, err := exportdata.Read(bytes.NewReader(buf.Bytes()), fset, nil, ""); err == nil || !strings.Contains(err.Error(), "kind") {
		t.Fatalf("expected kind mismatch error, got %v", err)
	}
	if _, err := exportdata.ReadBundle(bytes.NewReader([]byte("LL")), fset, nil); err == nil {
		t.Fatal("expected corrupt error for truncated bundle")
	}
}
