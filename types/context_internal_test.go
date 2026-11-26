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

package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"testing"
)

func TestLookupErasedAliasCallConv(t *testing.T) {
	ctx := NewContext()
	fset := token.NewFileSet()
	src := `package demo
type GoFn = func(x int32) (r int32)
type Mid = (GoFn)
//llgo:type C
type AliasC = func(int32)
//llgo:cdecl
type CFn = Mid
`
	f, err := parser.ParseFile(fset, "demo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	prep := prepareErasedCallConvAliasFiles([]*ast.File{f})
	if !prep.isolated["AliasC"] || !prep.isolated["CFn"] || prep.isolated["GoFn"] {
		t.Fatalf("isolated aliases = %#v, want AliasC and CFn only", prep.isolated)
	}
	if len(prep.uses) != 1 || prep.uses[0].targetName != "Mid" {
		t.Fatalf("erased uses = %#v, want Mid", prep.uses)
	}
	if prep.files[0] == f {
		t.Fatal("prepareErasedCallConvAliasFiles mutated input file instead of shallow copy")
	}
	// Verify the caller's original AST still references Mid verbatim.
	origSpec := f.Decls[3].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	if id, ok := origSpec.Type.(*ast.Ident); !ok || id.Name != "Mid" {
		t.Fatalf("original AST CFn RHS was mutated: %T", origSpec.Type)
	}
	// Verify rewritten CFn RHS is a cloned *ast.FuncType with fresh param idents.
	newSpec := prep.files[0].Decls[3].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	origFT := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.FuncType)
	clonedFT, ok := newSpec.Type.(*ast.FuncType)
	if !ok || clonedFT == origFT || clonedFT.Params.List[0].Names[0] == origFT.Params.List[0].Names[0] {
		t.Fatalf("rewritten CFn RHS did not clone FuncType cleanly")
	}

	// Cross-file rewrite must keep the cloned FuncType in the declaring file.
	fa, err := parser.ParseFile(fset, "a.go", "package x\nimport wire \"unsafe\"\ntype GoFn = func(wire.Pointer)\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile a.go: %v", err)
	}
	fb, err := parser.ParseFile(fset, "b.go", "package x\n//llgo:cdecl\ntype CFn = GoFn\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile b.go: %v", err)
	}
	xprep := prepareErasedCallConvAliasFiles([]*ast.File{fa, fb})
	if !xprep.isolated["CFn"] || len(xprep.synthNames) != 1 || len(xprep.files[0].Decls) != len(fa.Decls)+1 {
		t.Fatalf("cross-file prep = %#v", xprep)
	}
	xGoPkg := gotypes.NewPackage("example.com/x", "x")
	xGoInfo := &gotypes.Info{
		Defs:   make(map[*ast.Ident]gotypes.Object),
		Uses:   make(map[*ast.Ident]gotypes.Object),
		Types:  make(map[ast.Expr]gotypes.TypeAndValue),
		Scopes: make(map[ast.Node]*gotypes.Scope),
	}
	var xGoCfg gotypes.Config
	xGoCfg.Importer = &unsafeOnlyGoImporter{}
	if err := gotypes.NewChecker(&xGoCfg, fset, xGoPkg, xGoInfo).Files(xprep.files); err != nil {
		t.Fatalf("cross-file go/types check failed: %v", err)
	}
	xprep.restoreGoInfo(xGoPkg, xGoInfo, []*ast.File{fa, fb})
	if xGoInfo.Scopes[fa] == nil || xGoInfo.Scopes[fb] == nil {
		t.Fatal("restoreGoInfo did not remap cloned file scopes back to original files")
	}
	synthPkg := NewPackage("example.com/synthblock", "synthblock")
	synthPkg.synthGoNames = xprep.synthNames
	for name := range xprep.synthNames {
		if err := CheckExpr(fset, synthPkg, token.NoPos, ast.NewIdent(name), nil); err == nil {
			t.Fatalf("CheckExpr did not block synthetic alias %q", name)
		}
	}

	// Calling prescan twice exercises the existing-entry update path.
	ctx.prescanTypeDirectives("example.com/demo", []*ast.File{f}, prep.isolated)
	ctx.prescanTypeDirectives("example.com/demo", []*ast.File{f}, prep.isolated)

	// Simulate a Go 1.20/1.21 go/types package where AliasC's TypeName holds an
	// erased *gotypes.Signature directly rather than a *gotypes.Alias wrapper.
	goPkg := gotypes.NewPackage("example.com/demo", "demo")
	erasedSig := gotypes.NewSignatureType(
		nil, nil, nil,
		gotypes.NewTuple(gotypes.NewParam(token.NoPos, goPkg, "", gotypes.Typ[gotypes.Int32])),
		nil, false,
	)
	otherSig := gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)
	goPkg.Scope().Insert(gotypes.NewTypeName(token.NoPos, goPkg, "AliasC", erasedSig))
	ctx.registerCheckedGoPackage("example.com/demo", goPkg)
	ctx.registerCheckedGoPackage("example.com/empty", nil)

	ctx.mu.Lock()
	gotType := ctx.fromGoLocked(erasedSig, Go, false)
	otherType := ctx.fromGoLocked(otherSig, Go, false)
	ctx.mu.Unlock()

	gotSig, ok := gotType.(*Signature)
	if !ok || gotSig.CallConv() != CDecl {
		t.Fatalf("erased alias signature = %v (cc=%v), want CDecl calling convention", gotType, gotSig.CallConv())
	}
	if otherSigLL, ok := otherType.(*Signature); !ok || otherSigLL.CallConv() != Go {
		t.Fatalf("unrelated signature = %v, want Go calling convention", otherType)
	}
}

type unsafeOnlyGoImporter struct{}

func (unsafeOnlyGoImporter) Import(path string) (*gotypes.Package, error) {
	if path == "unsafe" {
		return gotypes.Unsafe, nil
	}
	return nil, nil
}
