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

package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	llssa "github.com/goplus/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

const (
	threadLocalDirective    = "//llgo:tls"
	goroutineLocalDirective = "//llgo:gls"
	legacyThreadLocal       = "llgo:threadlocal"
	legacyGoroutineLocal    = "llgo:goroutinelocal"
	localInitPrefix         = "__llgo_local_init_"
)

// PrepareLocalVariables processes package variable locality directives before
// Go SSA is created. Explicit initializers get a synthetic function that can
// replay the initializer once for each local execution context.
func PrepareLocalVariables(prog llssa.Program, fset *token.FileSet, pkg *types.Package, info *types.Info, files []*ast.File) error {
	if pkg == nil || info == nil {
		return nil
	}
	if len(prog.PackageDecls(llssa.PathOf(pkg))) == 0 {
		return nil
	}

	nextName := 0
	for _, initializer := range info.InitOrder {
		_, found, err := initializerLocality(prog, fset, pkg, initializer)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if len(files) == 0 {
			return fmt.Errorf("cannot prepare local initializer for package %q without syntax files", llssa.PathOf(pkg))
		}
		file := files[len(files)-1]
		name := ""
		for {
			name = fmt.Sprintf("%s%d", localInitPrefix, nextName)
			nextName++
			if pkg.Scope().Lookup(name) == nil {
				break
			}
		}
		fnObj, decl := makeLocalInitializer(pkg, info, name, initializer)
		if alt := pkg.Scope().Insert(fnObj); alt != nil {
			return fmt.Errorf("local initializer %q conflicts with %v", name, alt)
		}
		file.Decls = append(file.Decls, decl)
		initName := llssa.FullName(pkg, name)
		ensureName := localEnsureName(initName)
		for _, variable := range initializer.Lhs {
			fullName := llssa.FullName(pkg, variable.Name())
			prog.SetLocalInitFunc(fullName, initName)
			prog.SetLocalEnsureFunc(fullName, ensureName)
		}
	}
	for _, name := range pkg.Scope().Names() {
		variable, ok := pkg.Scope().Lookup(name).(*types.Var)
		if !ok {
			continue
		}
		fullName := llssa.FullName(pkg, name)
		decl, ok := prog.DeclInfo(fullName)
		if !ok || decl.Locality == llssa.LocalityNone || decl.HasInitializer || !typeHasPointers(variable.Type()) {
			continue
		}
		prog.SetLocalEnsureFunc(fullName, fullName+"$ensure")
	}
	return nil
}

func validateLocalInitializers(prog llssa.Program, pkg *types.Package) error {
	for name, info := range prog.PackageDecls(llssa.PathOf(pkg)) {
		if info.Locality != llssa.LocalityNone && info.HasInitializer && (info.InitFunc == "" || info.EnsureFunc == "") {
			return fmt.Errorf("local variable %s has not had its initializer prepared before SSA compilation", name)
		}
	}
	return nil
}

func registerVarLocalities(prog llssa.Program, fset *token.FileSet, pkg *types.Package, decl *ast.GenDecl) error {
	declLocality, declPos, err := localityFromDoc(fset, decl.Doc)
	if err != nil {
		return err
	}
	for _, node := range decl.Specs {
		spec := node.(*ast.ValueSpec)
		specLocality, specPos, err := localityFromDoc(fset, spec.Doc)
		if err != nil {
			return err
		}
		locality, localityPos, err := mergeLocality(fset, declLocality, declPos, specLocality, specPos)
		if err != nil {
			return err
		}
		if locality == llssa.LocalityNone {
			continue
		}
		if target := prog.Target(); target != nil && target.Target != "" {
			return localityError(fset, localityPos, "%s is not supported for target %q", localityDirective(locality), target.Target)
		}
		if hasDirective(decl.Doc, "go:embed") || hasDirective(spec.Doc, "go:embed") {
			return localityError(fset, spec.Pos(), "%s and %s cannot apply to the same variable declaration", localityDirective(locality), "//go:embed")
		}
		for _, ident := range spec.Names {
			if ident.Name == "_" {
				return localityError(fset, ident.Pos(), "locality directive cannot apply to the blank identifier")
			}
			prog.SetVarLocality(llssa.FullName(pkg, ident.Name), locality, len(spec.Values) != 0)
		}
	}
	return nil
}

func localityFromDoc(fset *token.FileSet, doc *ast.CommentGroup) (llssa.Locality, token.Pos, error) {
	var locality llssa.Locality
	var pos token.Pos
	if doc == nil {
		return locality, pos, nil
	}
	for _, directive := range sourceDirectives(doc) {
		var next llssa.Locality
		switch directive.Name {
		case "llgo:tls":
			next = llssa.ThreadLocal
		case "llgo:gls":
			next = llssa.GoroutineLocal
		case legacyThreadLocal:
			return llssa.LocalityNone, token.NoPos, localityError(fset, directive.Pos, "//%s is not supported; use %s", legacyThreadLocal, threadLocalDirective)
		case legacyGoroutineLocal:
			return llssa.LocalityNone, token.NoPos, localityError(fset, directive.Pos, "//%s is not supported; use %s", legacyGoroutineLocal, goroutineLocalDirective)
		default:
			continue
		}
		if directive.Args != "" {
			return llssa.LocalityNone, token.NoPos, localityError(fset, directive.Pos, "//%s does not accept arguments", directive.Name)
		}
		if locality != llssa.LocalityNone && locality != next {
			return llssa.LocalityNone, token.NoPos, localityError(fset, directive.Pos, "%s and %s cannot apply to the same variable declaration", threadLocalDirective, goroutineLocalDirective)
		}
		locality, pos = next, directive.Pos
	}
	return locality, pos, nil
}

func rejectNonVarLocality(fset *token.FileSet, doc *ast.CommentGroup) error {
	locality, pos, err := localityFromDoc(fset, doc)
	if err != nil {
		return err
	}
	if locality != llssa.LocalityNone {
		return localityError(fset, pos, "%s applies only to package-level var declarations", localityDirective(locality))
	}
	return nil
}

func hasDirective(doc *ast.CommentGroup, name string) bool {
	for _, directive := range sourceDirectives(doc) {
		if directive.Name == name {
			return true
		}
	}
	return false
}

func localityDirective(locality llssa.Locality) string {
	if locality == llssa.GoroutineLocal {
		return goroutineLocalDirective
	}
	return threadLocalDirective
}

func mergeLocality(fset *token.FileSet, a llssa.Locality, apos token.Pos, b llssa.Locality, bpos token.Pos) (llssa.Locality, token.Pos, error) {
	if a != llssa.LocalityNone && b != llssa.LocalityNone && a != b {
		return llssa.LocalityNone, token.NoPos, localityError(fset, bpos, "%s and %s cannot apply to the same variable declaration", threadLocalDirective, goroutineLocalDirective)
	}
	if b != llssa.LocalityNone {
		return b, bpos, nil
	}
	return a, apos, nil
}

func initializerLocality(prog llssa.Program, fset *token.FileSet, pkg *types.Package, initializer *types.Initializer) (llssa.Locality, bool, error) {
	var locality llssa.Locality
	found := false
	for _, variable := range initializer.Lhs {
		decl, ok := prog.DeclInfo(llssa.FullName(pkg, variable.Name()))
		if !ok || decl.Locality == llssa.LocalityNone {
			if found {
				return llssa.LocalityNone, false, localityError(fset, initializer.Rhs.Pos(), "one initializer cannot mix local and ordinary package variables")
			}
			continue
		}
		if !found {
			locality, found = decl.Locality, true
		} else if locality != decl.Locality {
			return llssa.LocalityNone, false, localityError(fset, initializer.Rhs.Pos(), "one initializer cannot mix thread-local and goroutine-local variables")
		}
	}
	if found && len(initializer.Lhs) != countLocalInitializerVariables(prog, pkg, initializer) {
		return llssa.LocalityNone, false, localityError(fset, initializer.Rhs.Pos(), "one initializer cannot mix local and ordinary package variables")
	}
	return locality, found, nil
}

func countLocalInitializerVariables(prog llssa.Program, pkg *types.Package, initializer *types.Initializer) int {
	n := 0
	for _, variable := range initializer.Lhs {
		if info, ok := prog.DeclInfo(llssa.FullName(pkg, variable.Name())); ok && info.Locality != llssa.LocalityNone {
			n++
		}
	}
	return n
}

func typeHasPointers(typ types.Type) bool {
	typ = types.Unalias(typ)
	switch typ := typ.(type) {
	case *types.Basic:
		return typ.Kind() == types.String || typ.Kind() == types.UnsafePointer
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return true
	case *types.Array:
		return typeHasPointers(typ.Elem())
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if typeHasPointers(typ.Field(i).Type()) {
				return true
			}
		}
		return false
	case *types.Named:
		return typeHasPointers(typ.Underlying())
	case *types.TypeParam:
		return true
	default:
		return false
	}
}

func makeLocalInitializer(pkg *types.Package, info *types.Info, name string, initializer *types.Initializer) (*types.Func, *ast.FuncDecl) {
	lhs := make([]ast.Expr, len(initializer.Lhs))
	for i, variable := range initializer.Lhs {
		ident := ast.NewIdent(variable.Name())
		info.Uses[ident] = variable
		lhs[i] = ident
	}
	nameIdent := ast.NewIdent(name)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fnObj := types.NewFunc(token.NoPos, pkg, name, sig)
	info.Defs[nameIdent] = fnObj
	decl := &ast.FuncDecl{
		Name: nameIdent,
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{Lhs: lhs, Tok: token.ASSIGN, Rhs: []ast.Expr{initializer.Rhs}},
		}},
	}
	return fnObj, decl
}

func localityError(fset *token.FileSet, pos token.Pos, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if fset == nil || pos == token.NoPos {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s: %s", fset.Position(pos), msg)
}

const (
	localInitInitializing = 1
	localInitDone         = 2
)

func localEnsureName(initFunc string) string {
	return initFunc + "$ensure"
}

func localGuardName(ensureFunc string) string {
	return strings.TrimSuffix(ensureFunc, "$ensure") + "$guard"
}

type localInitGroup struct {
	guard    llssa.Global
	initFunc string
	roots    []llssa.Global
}

func (p *context) registerLocalInitializer(pkg llssa.Package, local llssa.DeclInfo, global llssa.Global, goGlobal *ssa.Global) {
	if p.localInitGroups == nil {
		p.localInitGroups = make(map[string]*localInitGroup)
	}
	group := p.localInitGroups[local.EnsureFunc]
	if group == nil {
		guard := pkg.NewThreadLocalVar(localGuardName(local.EnsureFunc), types.NewPointer(types.Typ[types.Uint8]), llssa.InGo)
		guard.InitNil()
		group = &localInitGroup{guard: guard, initFunc: local.InitFunc}
		p.localInitGroups[local.EnsureFunc] = group
	}
	if typeHasPointers(goGlobal.Type().(*types.Pointer).Elem()) {
		group.roots = append(group.roots, global)
	}
}

func (p *context) buildLocalInitializers(pkg llssa.Package) {
	for _, name := range p.localInitGroupNames() {
		group := p.localInitGroups[name]
		ensure := pkg.NewFunc(name, llssa.NoArgsNoRet, llssa.InGo)
		if ensure.HasBody() {
			continue
		}
		b := ensure.MakeBody(3)
		zero := p.prog.IntVal(0, p.prog.Byte())
		initialized := b.BinOp(token.NEQ, b.Load(group.guard.Expr), zero)
		b.If(initialized, ensure.Block(2), ensure.Block(1))
		b.SetBlock(ensure.Block(1))
		b.Store(group.guard.Expr, p.prog.IntVal(localInitInitializing, p.prog.Byte()))
		p.registerLocalRoots(b, group)
		if group.initFunc != "" {
			helper := pkg.NewFunc(group.initFunc, llssa.NoArgsNoRet, llssa.InGo)
			b.Call(helper.Expr)
		}
		b.Store(group.guard.Expr, p.prog.IntVal(localInitDone, p.prog.Byte()))
		b.Jump(ensure.Block(2))
		b.SetBlock(ensure.Block(2))
		b.Return()
		b.EndBuild()
	}
}

func (p *context) localInitGroupNames() []string {
	names := make([]string, 0, len(p.localInitGroups))
	for name := range p.localInitGroups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *context) registerLocalRoots(b llssa.Builder, group *localInitGroup) {
	if len(group.roots) == 0 {
		return
	}
	register := p.pkg.RuntimeFunc("RegisterLocalRoot")
	for _, root := range group.roots {
		size := p.prog.SizeOf(p.prog.Elem(root.Expr.Type))
		b.Call(register, root.Expr, p.prog.IntVal(size, p.prog.Uintptr()))
	}
}

func (p *context) initializeLocalGuards(b llssa.Builder) {
	if len(p.localInitGroups) == 0 {
		return
	}
	for _, name := range p.localInitGroupNames() {
		group := p.localInitGroups[name]
		// Root-only groups are initialized lazily. This is required by the
		// runtime package itself: the root registry's pthread key is an
		// ordinary package variable and must be initialized before currentG
		// registers its TLS range.
		if group.initFunc == "" {
			continue
		}
		p.registerLocalRoots(b, group)
		b.Store(group.guard.Expr, p.prog.IntVal(localInitDone, p.prog.Byte()))
	}
}
