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
	"go/ast"
	"go/constant"
	"go/parser"
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

	// Alias, Var, Universe error, and instantiated generic Box[int].
	aliasObj := types.NewTypeName(0, pkg, "MyInt", nil)
	_ = types.NewAlias(aliasObj, types.Typ[types.Int32])
	pkg.Scope().Insert(aliasObj)
	pkg.Scope().Insert(types.NewVar(0, pkg, "GlobalCount", types.Typ[types.Int]))
	pkg.Scope().Insert(types.NewVar(0, pkg, "LastErr", types.Universe.Lookup("error").Type()))
	boxInt, err := types.Instantiate(nil, boxNamed, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		panic(err)
	}
	pkg.Scope().Insert(types.NewVar(0, pkg, "IntBox", boxInt))

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

	// Verify external Named identity reuse, Universe identity, and instantiated generics.
	if gotItem := st.Field(2).Type(); gotItem != dep.Scope().Lookup("Item").Type() {
		t.Fatalf("external dep.Item identity not preserved across Read: got %p, want %p", gotItem, dep.Scope().Lookup("Item").Type())
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

	// Verify types.Config.Check can type-check a downstream package importing
	// the exportdata-decoded package while preserving exact *types.Named identity.
	const downstreamSrc = `package user
import "example.com/demo"
var _ demo.Item
func Use(n *demo.Node, b demo.Box[int]) int {
	_ = demo.Answer
	_ = demo.CFunc(1)
	_ = demo.GlobalCount
	return n.Len() + b.Val
}
`
	f, err := parser.ParseFile(fset, "user.go", strings.Replace(downstreamSrc, "var _ demo.Item\n", "", 1), 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := &types.Info{
		Uses: make(map[*ast.Ident]types.Object),
	}
	userPkg, err := (&types.Config{Importer: imp}).Check("example.com/user", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("Config.Check against exportdata-decoded pkg: %v", err)
	}
	useFn := userPkg.Scope().Lookup("Use").(*types.Func)
	paramNode := useFn.Signature().Params().At(0).Type().(*types.Pointer).Elem()
	if paramNode != nodeNamed {
		t.Fatalf("imported demo.Node identity mismatch: got %p, want %p", paramNode, nodeNamed)
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
	gotNode := got[1].Scope().Lookup("Node").Type().(*types.Named).Underlying().(*types.Struct)
	if gotNode.Field(2).Type() != got[0].Scope().Lookup("Item").Type() {
		t.Fatal("ReadBundle did not share dep.Item identity between bundle packages")
	}
	if got[1].Scope().Lookup("LastErr").Type() != types.Universe.Lookup("error").Type() {
		t.Fatal("ReadBundle did not resolve Universe error identity")
	}
	if gotIntBox, ok := got[1].Scope().Lookup("IntBox").Type().(*types.Named); !ok || gotIntBox.TypeArgs().Len() != 1 || gotIntBox.Origin() != got[1].Scope().Lookup("Box").Type() {
		t.Fatalf("ReadBundle did not preserve instantiated generic Box[int]: %v", got[1].Scope().Lookup("IntBox").Type())
	}

	// Reversed bundle order: importer precedes dependency. Type and package
	// identities must remain identical regardless of bundle order.
	var revBuf bytes.Buffer
	if err := exportdata.WriteBundle(&revBuf, fset, []*types.Package{pkg, dep}); err != nil {
		t.Fatalf("WriteBundle (reversed): %v", err)
	}
	revImports := make(map[string]*types.Package)
	revGot, err := exportdata.ReadBundle(bytes.NewReader(revBuf.Bytes()), fset, revImports)
	if err != nil {
		t.Fatalf("ReadBundle (reversed): %v", err)
	}
	if len(revGot) != 2 || revGot[0].Path() != pkg.Path() || revGot[1].Path() != dep.Path() {
		t.Fatalf("ReadBundle (reversed) got %v", revGot)
	}
	revNode := revGot[0].Scope().Lookup("Node").Type().(*types.Named).Underlying().(*types.Struct)
	revItem := revGot[1].Scope().Lookup("Item").Type()
	if revNode.Field(2).Type() != revItem || !types.Identical(revNode.Field(2).Type(), revItem) {
		t.Fatalf("ReadBundle (reversed) split dep.Item identity: field=%p dep=%p identical=%v",
			revNode.Field(2).Type(), revItem, types.Identical(revNode.Field(2).Type(), revItem))
	}
	if len(revGot[0].Imports()) == 0 || revGot[0].Imports()[0] != revGot[1] {
		t.Fatalf("ReadBundle (reversed) split dep package identity: importer.Imports=%v dep=%p",
			revGot[0].Imports(), revGot[1])
	}
	if revImports[dep.Path()] != revGot[1] || revImports[pkg.Path()] != revGot[0] {
		t.Fatalf("ReadBundle (reversed) imports map mismatch: %v", revImports)
	}

	// Re-reading the same bundle into an already-populated imports map must
	// reuse the existing complete packages without duplicating identities.
	reusedGot, err := exportdata.ReadBundle(bytes.NewReader(revBuf.Bytes()), fset, revImports)
	if err != nil {
		t.Fatalf("ReadBundle (reused): %v", err)
	}
	if reusedGot[0] != revGot[0] || reusedGot[1] != revGot[1] {
		t.Fatalf("ReadBundle (reused) did not preserve package pointers")
	}

	// Two bundle packages referencing a shared external dependency (not in the
	// bundle itself) must still share Named and Alias identities for that dep.
	extPkg := types.NewPackage("example.com/ext", "ext")
	extNamedObj := types.NewTypeName(token.NoPos, extPkg, "Handle", nil)
	extNamed := types.NewNamed(extNamedObj, types.Typ[types.Int], nil)
	extPkg.Scope().Insert(extNamedObj)
	extAliasObj := types.NewTypeName(token.NoPos, extPkg, "HandleAlias", nil)
	extAlias := types.NewAlias(extAliasObj, types.Typ[types.String])
	extPkg.Scope().Insert(extAliasObj)
	extPkg.MarkComplete()

	userA := types.NewPackage("example.com/usera", "usera")
	userA.SetImports([]*types.Package{extPkg})
	userA.Scope().Insert(types.NewVar(token.NoPos, userA, "H", extNamed))
	userA.Scope().Insert(types.NewVar(token.NoPos, userA, "A", extAlias))
	userA.MarkComplete()

	userB := types.NewPackage("example.com/userb", "userb")
	userB.SetImports([]*types.Package{extPkg})
	userB.Scope().Insert(types.NewVar(token.NoPos, userB, "H", extNamed))
	userB.Scope().Insert(types.NewVar(token.NoPos, userB, "A", extAlias))
	userB.MarkComplete()

	var extBuf bytes.Buffer
	if err := exportdata.WriteBundle(&extBuf, fset, []*types.Package{userA, userB}); err != nil {
		t.Fatalf("WriteBundle (shared ext): %v", err)
	}
	extGot, err := exportdata.ReadBundle(bytes.NewReader(extBuf.Bytes()), fset, nil)
	if err != nil {
		t.Fatalf("ReadBundle (shared ext): %v", err)
	}
	if extGot[0].Scope().Lookup("H").Type() != extGot[1].Scope().Lookup("H").Type() {
		t.Fatalf("ReadBundle (shared ext) split Named identity")
	}
	if extGot[0].Scope().Lookup("A").Type() != extGot[1].Scope().Lookup("A").Type() {
		t.Fatalf("ReadBundle (shared ext) split Alias identity")
	}

	// Sequential Read(importer) followed by Read(dep) with the same imports map:
	// reading importer first must leave dep incomplete so the subsequent Read(dep)
	// populates all of dep's declarations while preserving shared type identities.
	depFull := types.NewPackage("example.com/seqdep", "seqdep")
	seqItemObj := types.NewTypeName(token.NoPos, depFull, "Item", nil)
	seqItem := types.NewNamed(seqItemObj, types.Typ[types.Int64], nil)
	depFull.Scope().Insert(seqItemObj)
	seqAliasObj := types.NewTypeName(token.NoPos, depFull, "Tag", nil)
	seqAlias := types.NewAlias(seqAliasObj, types.Typ[types.String])
	depFull.Scope().Insert(seqAliasObj)
	depFull.Scope().Insert(types.NewConst(token.NoPos, depFull, "DepConst", types.Typ[types.Int], constant.MakeInt64(99)))
	depFull.Scope().Insert(types.NewFunc(token.NoPos, depFull, "MakeItem", types.NewSignatureType(
		nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, depFull, "", seqItem)), false,
	)))
	depFull.MarkComplete()

	seqImp := types.NewPackage("example.com/seqimp", "seqimp")
	seqImp.SetImports([]*types.Package{depFull})
	seqImp.Scope().Insert(types.NewVar(token.NoPos, seqImp, "V", seqItem))
	seqImp.Scope().Insert(types.NewVar(token.NoPos, seqImp, "T", seqAlias))
	seqImp.MarkComplete()

	var seqImpBuf, seqDepBuf bytes.Buffer
	if err := exportdata.Write(&seqImpBuf, fset, seqImp); err != nil {
		t.Fatalf("Write(seqImp): %v", err)
	}
	if err := exportdata.Write(&seqDepBuf, fset, depFull); err != nil {
		t.Fatalf("Write(depFull): %v", err)
	}
	seqImports := make(map[string]*types.Package)
	gotImp, err := exportdata.Read(bytes.NewReader(seqImpBuf.Bytes()), fset, seqImports, seqImp.Path())
	if err != nil {
		t.Fatalf("Read(seqImp): %v", err)
	}
	if gotImp.Imports()[0].Complete() {
		t.Fatal("placeholder dependency package was prematurely marked Complete before its own Read")
	}
	gotDep, err := exportdata.Read(bytes.NewReader(seqDepBuf.Bytes()), fset, seqImports, depFull.Path())
	if err != nil {
		t.Fatalf("Read(depFull): %v", err)
	}
	if gotDep != gotImp.Imports()[0] || !gotDep.Complete() {
		t.Fatalf("sequential Read(dep) identity/complete mismatch: same=%v complete=%v", gotDep == gotImp.Imports()[0], gotDep.Complete())
	}
	if gotDep.Scope().Lookup("DepConst") == nil || gotDep.Scope().Lookup("MakeItem") == nil {
		t.Fatalf("sequential Read(dep) left dependency scope incomplete: names=%v", gotDep.Scope().Names())
	}
	if gotImp.Scope().Lookup("V").Type() != gotDep.Scope().Lookup("Item").Type() {
		t.Fatal("sequential Read split dep.Item Named identity")
	}
	if gotImp.Scope().Lookup("T").Type() != gotDep.Scope().Lookup("Tag").Type() {
		t.Fatal("sequential Read split dep.Tag Alias identity")
	}
	makeItemRet := gotDep.Scope().Lookup("MakeItem").Type().(*types.Signature).Results().At(0).Type()
	if makeItemRet != gotImp.Scope().Lookup("V").Type() {
		t.Fatal("sequential Read MakeItem return type does not share dep.Item identity")
	}

	// Struct fields referencing package Aliases, chained aliases, mutually
	// recursive Named/Alias types, and aliases to instantiated generics must
	// all resolve to the final populated Alias objects (never nil-Rhs shells).
	aliasPkg := types.NewPackage("example.com/aliasgraph", "aliasgraph")
	tagObj := types.NewTypeName(token.NoPos, aliasPkg, "Tag", nil)
	tagAlias := types.NewAlias(tagObj, types.Typ[types.String])
	aliasPkg.Scope().Insert(tagObj)

	tag2Obj := types.NewTypeName(token.NoPos, aliasPkg, "Tag2", nil)
	tag2Alias := types.NewAlias(tag2Obj, tagAlias)
	aliasPkg.Scope().Insert(tag2Obj)

	itemObj := types.NewTypeName(token.NoPos, aliasPkg, "Item", nil)
	_ = types.NewNamed(itemObj, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, aliasPkg, "Value", tagAlias, false),
		types.NewField(token.NoPos, aliasPkg, "Nested", tag2Alias, false),
	}, nil), nil)
	aliasPkg.Scope().Insert(itemObj)

	// Mutually recursive Named <-> Alias: type Rec struct { Next RecPtr }; type RecPtr = *Rec.
	recObj := types.NewTypeName(token.NoPos, aliasPkg, "Rec", nil)
	recNamed := types.NewNamed(recObj, nil, nil)
	recPtrObj := types.NewTypeName(token.NoPos, aliasPkg, "RecPtr", nil)
	recPtrAlias := types.NewAlias(recPtrObj, types.NewPointer(recNamed))
	recNamed.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, aliasPkg, "Next", recPtrAlias, false),
	}, nil))
	aliasPkg.Scope().Insert(recObj)
	aliasPkg.Scope().Insert(recPtrObj)

	// Generic Named + Alias referencing its instantiation + struct referencing the Alias.
	tp := types.NewTypeParam(types.NewTypeName(token.NoPos, aliasPkg, "T", nil), types.Universe.Lookup("any").Type())
	genBoxObj := types.NewTypeName(token.NoPos, aliasPkg, "GenBox", nil)
	genBox := types.NewNamed(genBoxObj, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, aliasPkg, "Label", tagAlias, false),
		types.NewField(token.NoPos, aliasPkg, "Elem", tp, false),
	}, nil), nil)
	genBox.SetTypeParams([]*types.TypeParam{tp})
	aliasPkg.Scope().Insert(genBoxObj)
	strBoxInst, err := types.Instantiate(nil, genBox, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatalf("Instantiate GenBox[int]: %v", err)
	}
	intBoxAliasObj := types.NewTypeName(token.NoPos, aliasPkg, "IntBoxAlias", nil)
	intBoxAlias := types.NewAlias(intBoxAliasObj, strBoxInst)
	aliasPkg.Scope().Insert(intBoxAliasObj)
	holderObj := types.NewTypeName(token.NoPos, aliasPkg, "Holder", nil)
	_ = types.NewNamed(holderObj, types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, aliasPkg, "B", intBoxAlias, false),
	}, nil), nil)
	aliasPkg.Scope().Insert(holderObj)
	aliasPkg.MarkComplete()

	var aliasBuf bytes.Buffer
	if err := exportdata.Write(&aliasBuf, fset, aliasPkg); err != nil {
		t.Fatalf("Write(aliasPkg): %v", err)
	}
	gotAliasPkg, err := exportdata.Read(bytes.NewReader(aliasBuf.Bytes()), fset, nil, aliasPkg.Path())
	if err != nil {
		t.Fatalf("Read(aliasPkg): %v", err)
	}
	gotTag := gotAliasPkg.Scope().Lookup("Tag").Type().(*types.Alias)
	gotTag2 := gotAliasPkg.Scope().Lookup("Tag2").Type().(*types.Alias)
	gotItemSt := gotAliasPkg.Scope().Lookup("Item").Type().(*types.Named).Underlying().(*types.Struct)
	field0, ok := gotItemSt.Field(0).Type().(*types.Alias)
	if !ok || field0 != gotTag || field0.Rhs() != types.Typ[types.String] {
		t.Fatalf("Item.Value = %v (rhs=%v, ptr=%p), want Tag (%p, rhs=string)", gotItemSt.Field(0).Type(), field0.Rhs(), field0, gotTag)
	}
	field1, ok := gotItemSt.Field(1).Type().(*types.Alias)
	if !ok || field1 != gotTag2 || field1.Rhs() != gotTag {
		t.Fatalf("Item.Nested = %v (rhs=%v), want Tag2 -> Tag", gotItemSt.Field(1).Type(), field1.Rhs())
	}
	gotRec := gotAliasPkg.Scope().Lookup("Rec").Type().(*types.Named)
	gotRecPtr := gotAliasPkg.Scope().Lookup("RecPtr").Type().(*types.Alias)
	if gotRec.Underlying().(*types.Struct).Field(0).Type() != gotRecPtr {
		t.Fatal("Rec.Next did not preserve RecPtr alias identity")
	}
	if ptr, ok := gotRecPtr.Rhs().(*types.Pointer); !ok || ptr.Elem() != gotRec {
		t.Fatalf("RecPtr.Rhs = %v, want *Rec", gotRecPtr.Rhs())
	}
	gotIntBoxAlias := gotAliasPkg.Scope().Lookup("IntBoxAlias").Type().(*types.Alias)
	gotHolderSt := gotAliasPkg.Scope().Lookup("Holder").Type().(*types.Named).Underlying().(*types.Struct)
	if gotHolderSt.Field(0).Type() != gotIntBoxAlias {
		t.Fatal("Holder.B did not preserve IntBoxAlias identity")
	}
	if instNamed, ok := gotIntBoxAlias.Rhs().(*types.Named); !ok || instNamed.TypeArgs().Len() != 1 ||
		instNamed.Underlying().(*types.Struct).Field(0).Type() != gotTag {
		t.Fatalf("IntBoxAlias.Rhs did not preserve instantiated GenBox[int] with Tag field: %v", gotIntBoxAlias.Rhs())
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
	// Every strict prefix of a valid package or bundle must fail cleanly without panicking.
	var singleBuf bytes.Buffer
	if err := exportdata.Write(&singleBuf, fset, pkg); err != nil {
		t.Fatalf("Write(pkg): %v", err)
	}
	singleBytes := singleBuf.Bytes()
	for cut := 0; cut < len(singleBytes); cut++ {
		if _, err := exportdata.Read(bytes.NewReader(singleBytes[:cut]), fset, nil, pkg.Path()); err == nil {
			t.Fatalf("Read succeeded on truncated single package at offset %d/%d", cut, len(singleBytes))
		}
	}
	bundleBytes := revBuf.Bytes()
	for cut := 0; cut < len(bundleBytes); cut++ {
		if _, err := exportdata.ReadBundle(bytes.NewReader(bundleBytes[:cut]), fset, nil); err == nil {
			t.Fatalf("ReadBundle succeeded on truncated bundle at offset %d/%d", cut, len(bundleBytes))
		}
	}
}
