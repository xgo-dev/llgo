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
	st := newErasedAliasState()
	prep := st.prepare([]*ast.File{f}, nil, fset, nil)
	if !prep.isolated["AliasC"] || !prep.isolated["CFn"] || prep.isolated["GoFn"] {
		t.Fatalf("isolated aliases = %#v, want AliasC and CFn only", prep.isolated)
	}
	if prep.files[0] == f {
		t.Fatal("erasedAliasState.prepare mutated input file instead of shallow copy")
	}
	// Verify the caller's original AST still references Mid verbatim.
	origSpec := f.Decls[3].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	if id, ok := origSpec.Type.(*ast.Ident); !ok || id.Name != "Mid" {
		t.Fatalf("original AST CFn RHS was mutated: %T", origSpec.Type)
	}
	// Verify rewritten CFn RHS is a deep-cloned *ast.FuncType with fresh param idents.
	newSpec := prep.files[0].Decls[3].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	origFT := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.FuncType)
	clonedFT, ok := newSpec.Type.(*ast.FuncType)
	if !ok || clonedFT == origFT || clonedFT.Params.List[0].Names[0] == origFT.Params.List[0].Names[0] || clonedFT.Params.List[0].Type == origFT.Params.List[0].Type {
		t.Fatalf("rewritten CFn RHS did not deep-clone FuncType cleanly")
	}

	// Cross-file and multi-batch incremental prepare: batch 1 declares GoFn with
	// a renamed import; batch 2 adds //llgo:cdecl CFn = (GoFn) (reusing the
	// resolved signature directly with zero synthetic files); batch 3 adds a
	// cross-file pair with nested struct/func types and //go:build go1.22 to
	// verify FileVersions remapping, Info.TypeOf on (GoFn), and Info node hygiene.
	fa, err := parser.ParseFile(fset, "a.go", "//go:build go1.22\n\npackage x\nimport wire \"unsafe\"\ntype GoFn = func(wire.Pointer)\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile a.go: %v", err)
	}
	fb, err := parser.ParseFile(fset, "b.go", "//go:build go1.22\n\npackage x\n//llgo:cdecl\ntype CFn = (GoFn)\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile b.go: %v", err)
	}
	fc, err := parser.ParseFile(fset, "c.go", "//go:build go1.22\n\npackage x\ntype GoFn2 = func(struct{ Value int }, func(v int))\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile c.go: %v", err)
	}
	fd, err := parser.ParseFile(fset, "d.go", "//go:build go1.22\n\npackage x\n//llgo:cdecl\ntype CFn2 = (GoFn2)\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile d.go: %v", err)
	}
	xst := newErasedAliasState()
	xGoPkg := gotypes.NewPackage("example.com/x", "x")
	xGoInfo := &gotypes.Info{
		Defs:       make(map[*ast.Ident]gotypes.Object),
		Uses:       make(map[*ast.Ident]gotypes.Object),
		Types:      make(map[ast.Expr]gotypes.TypeAndValue),
		Implicits:  make(map[ast.Node]gotypes.Object),
		Selections: make(map[*ast.SelectorExpr]*gotypes.Selection),
		Scopes:     make(map[ast.Node]*gotypes.Scope),
	}
	initGoInfoFileVersions(xGoInfo)
	var xGoCfg gotypes.Config
	xGoCfg.GoVersion = "go1.20"
	xGoImp := &unsafeOnlyGoImporter{}
	xGoCfg.Importer = xGoImp
	xGoChecker := gotypes.NewChecker(&xGoCfg, fset, xGoPkg, xGoInfo)

	// Batch 1: fa alone (records GoFn without directive).
	p1 := xst.prepare([]*ast.File{fa}, xGoPkg, fset, xGoImp)
	if err := xGoChecker.Files(p1.files); err != nil {
		t.Fatalf("batch 1 go/types check failed: %v", err)
	}
	p1.restoreGoInfo(xGoInfo)

	// Batch 2: fb alone (reuses GoFn's resolved *gotypes.Signature directly).
	p2 := xst.prepare([]*ast.File{fb}, xGoPkg, fset, xGoImp)
	if !p2.isolated["CFn"] || len(p2.files) != 1 || len(p2.synthNames) != 1 {
		t.Fatalf("batch 2 prep = %#v", p2)
	}
	if err := xGoChecker.Files(p2.files); err != nil {
		t.Fatalf("batch 2 go/types check failed: %v", err)
	}
	p2.restoreGoInfo(xGoInfo)

	// Batch 3: fc + fd together (must get a distinct synth name without collision).
	p3 := xst.prepare([]*ast.File{fc, fd}, xGoPkg, fset, xGoImp)
	if !p3.isolated["CFn2"] || len(xst.synthNames) != 2 {
		t.Fatalf("batch 3 prep = %#v, all synthNames = %#v", p3, xst.synthNames)
	}
	if err := xGoChecker.Files(p3.files); err != nil {
		t.Fatalf("batch 3 go/types check failed: %v", err)
	}
	p3.restoreGoInfo(xGoInfo)

	// Exercise imported, renamed-import, and dot-import resolution plus fileImportDir
	// and astExprCloner across all type expression kinds (ensuring Go 1.27 coverage
	// of Go 1.20 / gotypesalias=0 paths as well).
	depPkg := gotypes.NewPackage("example.com/dep", "dep")
	depSig := gotypes.NewSignatureType(
		nil, nil, nil,
		gotypes.NewTuple(gotypes.NewParam(token.NoPos, depPkg, "", gotypes.Typ[gotypes.Int32])),
		nil, false,
	)
	depPkg.Scope().Insert(gotypes.NewTypeName(token.NoPos, depPkg, "Fn", depSig))
	depPkg.MarkComplete()
	otherPkg := gotypes.NewPackage("example.com/other", "other")
	otherPkg.MarkComplete()
	xGoImp.pkgs = map[string]*gotypes.Package{
		"example.com/dep":   depPkg,
		"example.com/other": otherPkg,
	}
	if got := fileImportDir(fset, fa); got != "." {
		t.Fatalf("fileImportDir(a.go) = %q, want .", got)
	}
	fe, err := parser.ParseFile(fset, "/src/x/e.go", `package x
import (
	"example.com/other"
	"example.com/dep"
	d "example.com/dep"
	. "example.com/dep"
)
var _ = other.Invalid
type GoImp = dep.Fn
type GoRen = d.Fn
type GoDot = Fn
//llgo:cdecl
type CImp = GoImp
//llgo:cdecl
type CRen = GoRen
//llgo:cdecl
type CDot = GoDot
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile e.go: %v", err)
	}
	if got := fileImportDir(fset, fe); got != "/src/x" {
		t.Fatalf("fileImportDir(/src/x/e.go) = %q, want /src/x", got)
	}
	impState := newErasedAliasState()
	impGoPkg := gotypes.NewPackage("example.com/imp", "x")
	pImp := impState.prepare([]*ast.File{fe}, impGoPkg, fset, xGoImp)
	if !pImp.isolated["CImp"] || !pImp.isolated["CRen"] || !pImp.isolated["CDot"] {
		t.Fatalf("imported alias isolation = %#v, want CImp, CRen, CDot", pImp.isolated)
	}

	richSrc := `package x
type G[A, B any] struct{}
type Rich = func(
	a *int,
	b []string,
	c [1 + 2]byte,
	d map[string]chan<- int,
	e interface{ ~int | ~string; M(x int) error },
	f struct{ G[int, string]; V int "tag" },
	g G[int, int],
	h [len("ab")]int,
	i [unsafe.Sizeof((int)(0))]int,
	j ...func(int),
)
//llgo:cdecl
type CRich = Rich
`
	fRich, err := parser.ParseFile(fset, "rich.go", richSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile rich.go: %v", err)
	}
	cloner := &astExprCloner{}
	richFT := fRich.Decls[1].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.FuncType)
	if cloned := cloner.cloneFuncType(richFT); cloned == nil || cloner.failed {
		t.Fatalf("astExprCloner failed on rich FuncType: cloned=%v failed=%v", cloned, cloner.failed)
	}
	// Exercise remaining expression & statement cloning cases (CompositeLit,
	// KeyValueExpr, SliceExpr, TypeAssertExpr, IndexExpr, FuncLit with statements,
	// and fallback).
	for _, exprSrc := range []string{
		`S{k: v}`,
		`s[1:2:3]`,
		`x.(int)`,
		`G[int]`,
		`func(ch chan int, s []int, iface any) int {
			const c = 1
			type t = int
			var v t = c
			v++
			if u := v; u > 0 {
				ch <- u
			} else {
				;
			}
		L:
			for i := 0; i < 1; i++ {
				break L
			}
			for k, val := range s {
				_ = k + val
			}
			switch v {
			case 1:
				go f(v)
			}
			switch w := iface.(type) {
			case int:
				defer f(w)
			}
			select {
			case x := <-ch:
				return x
			}
		}`,
	} {
		expr, err := parser.ParseExpr(exprSrc)
		if err != nil {
			t.Fatalf("ParseExpr(%q): %v", exprSrc, err)
		}
		c2 := &astExprCloner{}
		if got := c2.cloneExpr(expr); got == nil || c2.failed {
			t.Fatalf("cloneExpr(%q) failed: got=%v failed=%v", exprSrc, got, c2.failed)
		}
	}
	cBad := &astExprCloner{}
	if got := cBad.cloneExpr(&ast.BadExpr{}); got != nil || !cBad.failed {
		t.Fatalf("cloneExpr(BadExpr) = %v, failed=%v; want nil, true", got, cBad.failed)
	}
	cBadStmt := &astExprCloner{}
	if got := cBadStmt.cloneStmt(&ast.BadStmt{}); got != nil || !cBadStmt.failed {
		t.Fatalf("cloneStmt(BadStmt) = %v, failed=%v; want nil, true", got, cBadStmt.failed)
	}
	cBadDecl := &astExprCloner{}
	if got := cBadDecl.cloneDecl(&ast.BadDecl{}); got != nil || !cBadDecl.failed {
		t.Fatalf("cloneDecl(BadDecl) = %v, failed=%v; want nil, true", got, cBadDecl.failed)
	}
	cBadSpec := &astExprCloner{}
	if got := cBadSpec.cloneSpec(&ast.ImportSpec{}); got != nil || !cBadSpec.failed {
		t.Fatalf("cloneSpec(ImportSpec) = %v, failed=%v; want nil, true", got, cBadSpec.failed)
	}

	if xGoInfo.Scopes[fa] == nil || xGoInfo.Scopes[fb] == nil || xGoInfo.Scopes[fc] == nil || xGoInfo.Scopes[fd] == nil {
		t.Fatal("restoreGoInfo did not preserve original file scopes across batches")
	}
	// Original parenthesized RHS (GoFn) and (GoFn2) must retain TypeOf entries.
	fbRHS := fb.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type
	fdRHS := fd.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type
	if xGoInfo.TypeOf(fbRHS) == nil || xGoInfo.TypeOf(fdRHS) == nil {
		t.Fatalf("TypeOf lost for parenthesized alias RHS: fb=%v fd=%v", xGoInfo.TypeOf(fbRHS), xGoInfo.TypeOf(fdRHS))
	}
	// Verify every node recorded in xGoInfo belongs to the original caller ASTs,
	// and FileVersions was migrated back to original files on Go 1.22+.
	origNodes := make(map[ast.Node]bool)
	for _, rf := range []*ast.File{fa, fb, fc, fd} {
		ast.Inspect(rf, func(n ast.Node) bool {
			if n != nil {
				origNodes[n] = true
			}
			return true
		})
	}
	for id := range xGoInfo.Defs {
		if !origNodes[id] {
			t.Fatalf("synthetic ident %q leaked into goInfo.Defs", id.Name)
		}
	}
	for id := range xGoInfo.Uses {
		if !origNodes[id] {
			t.Fatalf("synthetic ident %q leaked into goInfo.Uses", id.Name)
		}
	}
	for expr := range xGoInfo.Types {
		if !origNodes[expr] {
			t.Fatalf("synthetic expr %T leaked into goInfo.Types", expr)
		}
	}
	for node := range xGoInfo.Scopes {
		if !origNodes[node] {
			t.Fatalf("synthetic node %T leaked into goInfo.Scopes", node)
		}
	}
	for node := range xGoInfo.Implicits {
		if !origNodes[node] {
			t.Fatalf("synthetic node %T leaked into goInfo.Implicits", node)
		}
	}
	for sel := range xGoInfo.Selections {
		if !origNodes[sel] {
			t.Fatalf("synthetic selector leaked into goInfo.Selections")
		}
	}
	fv := make(map[*ast.File]string)
	if copyGoInfoFileVersions(fv, xGoInfo) {
		for _, rf := range []*ast.File{fa, fb, fc, fd} {
			if fv[rf] != "go1.22" {
				t.Fatalf("FileVersions[%s] = %q, want go1.22", rf.Name.Name, fv[rf])
			}
		}
		if len(fv) != 4 {
			t.Fatalf("synthetic file leaked into FileVersions: len=%d", len(fv))
		}
	}
	synthPkg := NewPackage("example.com/synthblock", "synthblock")
	synthPkg.synthGoNames = xst.synthNames
	for name := range xst.synthNames {
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
	ctx.registerCheckedGoPackage(NewPackage("example.com/demo", "demo"), goPkg)
	ctx.registerCheckedGoPackage(NewPackage("example.com/empty", "empty"), nil)

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

type unsafeOnlyGoImporter struct {
	pkgs map[string]*gotypes.Package
}

func (u *unsafeOnlyGoImporter) Import(path string) (*gotypes.Package, error) {
	if path == "unsafe" {
		return gotypes.Unsafe, nil
	}
	if u.pkgs != nil {
		if p, ok := u.pkgs[path]; ok {
			return p, nil
		}
	}
	return nil, nil
}

func (u *unsafeOnlyGoImporter) ImportFrom(path, _ string, _ gotypes.ImportMode) (*gotypes.Package, error) {
	return u.Import(path)
}
