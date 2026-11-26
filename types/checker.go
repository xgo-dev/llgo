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
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	gotypes "go/types"
)

// Error describes a type-checking error (mirroring go/types.Error).
type Error struct {
	Fset *token.FileSet
	Pos  token.Pos
	Msg  string
	Soft bool
}

func (err Error) Error() string {
	if err.Fset != nil && err.Pos.IsValid() {
		return fmt.Sprintf("%s: %s", err.Fset.Position(err.Pos), err.Msg)
	}
	return err.Msg
}

// ArgumentError holds an error associated with a type argument index
// (mirroring go/types.ArgumentError).
type ArgumentError struct {
	Index int
	Err   error
}

func (e *ArgumentError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ArgumentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapGoError(err error) error {
	if err == nil {
		return nil
	}
	var ge gotypes.Error
	if errors.As(err, &ge) {
		return Error{
			Fset: ge.Fset,
			Pos:  ge.Pos,
			Msg:  ge.Msg,
			Soft: ge.Soft,
		}
	}
	var ae *gotypes.ArgumentError
	if errors.As(err, &ae) {
		return &ArgumentError{
			Index: ae.Index,
			Err:   wrapGoError(ae.Err),
		}
	}
	return err
}

// Importer resolves import paths to Packages (mirroring go/types.Importer).
type Importer interface {
	Import(path string) (*Package, error)
}

// ImporterFrom resolves import paths relative to a directory
// (mirroring go/types.ImporterFrom).
type ImporterFrom interface {
	Importer
	ImportFrom(path, dir string, mode ImportMode) (*Package, error)
}

// Config specifies the configuration for type checking (mirroring go/types.Config).
type Config struct {
	Context                  *Context
	GoVersion                string
	IgnoreFuncBodies         bool
	FakeImportC              bool
	Error                    func(err error)
	Importer                 Importer
	Sizes                    Sizes
	DisableUnusedImportCheck bool
}

// Check type-checks a package and returns the resulting package object and the
// first error if any (mirroring go/types.Config.Check).
func (conf *Config) Check(path string, fset *token.FileSet, files []*ast.File, info *Info) (*Package, error) {
	pkg := NewPackage(path, "")
	err := NewChecker(conf, fset, pkg, info).Files(files)
	return pkg, err
}

// Checker maintains the state of the type checker (mirroring go/types.Checker).
type Checker struct {
	*Info

	ctx       *Context
	pkg       *Package
	goPkg     *gotypes.Package
	goInfo    *gotypes.Info
	goChecker *gotypes.Checker
	goVersion string
}

type llgoContextProvider interface {
	LLGoContext() *Context
}

// NewChecker returns a new Checker instance for a given package
// (mirroring go/types.NewChecker).
func NewChecker(conf *Config, fset *token.FileSet, pkg *Package, info *Info) *Checker {
	var cfg Config
	if conf != nil {
		cfg = *conf
	}
	ctx := cfg.Context
	if ctx == nil && cfg.Sizes != nil {
		switch s := cfg.Sizes.(type) {
		case *Context:
			ctx = s
		case *archSizes:
			ctx = NewContextWithConfig(ContextConfig{
				Spec:             s.spec,
				HostBasicLayouts: s.hostBasicLayouts,
				SIMDAlignments:   s.simdAlignments,
			})
		case llgoContextProvider:
			ctx = s.LLGoContext()
		}
	}
	if ctx == nil {
		ctx = NewContext()
	}
	if pkg == nil {
		pkg = NewPackage("", "")
	}
	goPkg := pkg.goPkg
	if goPkg == nil {
		goPkg = gotypes.NewPackage(pkg.path, pkg.name)
	}

	goConf := &gotypes.Config{
		Context:                  ctx.ensureGoContext(),
		GoVersion:                cfg.GoVersion,
		IgnoreFuncBodies:         cfg.IgnoreFuncBodies,
		FakeImportC:              cfg.FakeImportC,
		DisableUnusedImportCheck: cfg.DisableUnusedImportCheck,
	}
	if cfg.Error != nil {
		errFn := cfg.Error
		goConf.Error = func(e error) {
			errFn(wrapGoError(e))
		}
	}
	if cfg.Importer != nil {
		goConf.Importer = &goImporterAdapter{ctx: ctx, imp: cfg.Importer}
	}
	goConf.Sizes = bridgeToGoSizes(ctx, cfg.Sizes)

	goInfo := makeMirrorGoInfo(info)
	return &Checker{
		Info:      info,
		ctx:       ctx,
		pkg:       pkg,
		goPkg:     goPkg,
		goInfo:    goInfo,
		goChecker: gotypes.NewChecker(goConf, fset, goPkg, goInfo),
		goVersion: cfg.GoVersion,
	}
}

// Files checks the provided files as part of the checker's package
// (mirroring go/types.Checker.Files).
func (check *Checker) Files(files []*ast.File) error {
	if check == nil || check.goChecker == nil {
		return nil
	}
	check.ctx.registerCheckedGoPackage(check.pkg.path, check.goPkg)
	checkFiles := files
	var prep erasedAliasPrep
	if goTypesErasesAliases() {
		prep = prepareErasedCallConvAliasFiles(files)
		checkFiles = prep.files
	}
	check.ctx.prescanTypeDirectives(check.pkg.path, files, prep.isolated)
	err := check.goChecker.Files(checkFiles)
	prep.restoreGoInfo(check.goPkg, check.goInfo, files)
	check.ctx.populateFromGoPackage(check.pkg, check.goPkg, check.Info, check.goInfo, check.goVersion, files)
	if len(prep.synthNames) > 0 && check.pkg != nil && check.pkg.scope != nil {
		for name := range prep.synthNames {
			delete(check.pkg.scope.elems, name)
		}
	}
	return wrapGoError(err)
}

type erasedAliasUse struct {
	ident      *ast.Ident
	targetName string
}

type erasedAliasPrep struct {
	files        []*ast.File
	isolated     map[string]bool
	uses         []erasedAliasUse
	synthNames   map[string]bool
	synthIdents  []*ast.Ident
	synthExprs   []ast.Expr
	clonedToOrig map[*ast.File]*ast.File
}

func (p *erasedAliasPrep) restoreGoInfo(goPkg *gotypes.Package, goInfo *gotypes.Info, origFiles []*ast.File) {
	if goInfo == nil {
		return
	}
	for _, id := range p.synthIdents {
		if goInfo.Defs != nil {
			delete(goInfo.Defs, id)
		}
		if goInfo.Uses != nil {
			delete(goInfo.Uses, id)
		}
		if goInfo.Types != nil {
			delete(goInfo.Types, id)
		}
	}
	for _, expr := range p.synthExprs {
		if goInfo.Types != nil {
			delete(goInfo.Types, expr)
		}
	}
	if len(p.clonedToOrig) > 0 && goInfo.Scopes != nil {
		for cloned, orig := range p.clonedToOrig {
			if sc, ok := goInfo.Scopes[cloned]; ok {
				goInfo.Scopes[orig] = sc
				delete(goInfo.Scopes, cloned)
			}
		}
	}
	if len(p.uses) > 0 && goPkg != nil && goInfo.Uses != nil {
		scope := goPkg.Scope()
		for _, u := range p.uses {
			if obj := scope.Lookup(u.targetName); obj != nil {
				goInfo.Uses[u.ident] = obj
			}
		}
	}
}

type localAliasDecl struct {
	expr    ast.Expr
	fileIdx int
}

// prepareErasedCallConvAliasFiles isolates native-callconv function type aliases
// on Go toolchains where go/types erases aliases into bare *gotypes.Signature
// values (Go 1.20/1.21, or Go 1.22 with gotypesalias=0). When a directive alias
// such as `//llgo:cdecl\ntype CFn = GoFn` aliases a plain Go function alias,
// expanding CFn's RHS to a cloned *ast.FuncType allocates a distinct
// *gotypes.Signature for CFn while keeping GoFn's signature untouched and
// type-identical. When the underlying FuncType lives in a different file, the
// cloned FuncType stays in that originating file (as a synthetic package-level
// alias) so file-scoped imports and import aliases remain bound identically.
func prepareErasedCallConvAliasFiles(files []*ast.File) erasedAliasPrep {
	res := erasedAliasPrep{files: files}
	if len(files) == 0 {
		return res
	}
	aliasDecls := make(map[string]localAliasDecl)
	usedNames := make(map[string]bool)
	hasDirectiveAlias := false
	for fi, f := range files {
		if f == nil {
			continue
		}
		for _, imp := range f.Imports {
			if imp != nil && imp.Name != nil {
				usedNames[imp.Name.Name] = true
			}
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name != nil {
					usedNames[d.Name.Name] = true
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.CONST, token.VAR:
					for _, spec := range d.Specs {
						if vs, ok := spec.(*ast.ValueSpec); ok {
							for _, name := range vs.Names {
								if name != nil {
									usedNames[name.Name] = true
								}
							}
						}
					}
				case token.TYPE:
					for _, spec := range d.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok || ts.Name == nil {
							continue
						}
						usedNames[ts.Name.Name] = true
						if !ts.Assign.IsValid() {
							continue
						}
						aliasDecls[ts.Name.Name] = localAliasDecl{expr: ts.Type, fileIdx: fi}
						doc := ts.Doc
						if doc == nil && len(d.Specs) == 1 {
							doc = d.Doc
						}
						if _, ok := parseTypeCallConvDirective(doc); ok {
							hasDirectiveAlias = true
						}
					}
				}
			}
		}
	}
	if !hasDirectiveAlias {
		return res
	}

	resolveLocalFuncType := func(expr ast.Expr, startFileIdx int) (*ast.FuncType, int, *ast.Ident) {
		e := unwrapAstParens(expr)
		if ft, ok := e.(*ast.FuncType); ok {
			return ft, startFileIdx, nil
		}
		origIdent, ok := e.(*ast.Ident)
		if !ok {
			return nil, -1, nil
		}
		visited := make(map[string]bool)
		cur := origIdent.Name
		for !visited[cur] {
			visited[cur] = true
			ad, exists := aliasDecls[cur]
			if !exists {
				return nil, -1, nil
			}
			rhs := unwrapAstParens(ad.expr)
			if ft, ok := rhs.(*ast.FuncType); ok {
				return ft, ad.fileIdx, origIdent
			}
			nextID, ok := rhs.(*ast.Ident)
			if !ok {
				return nil, -1, nil
			}
			cur = nextID.Name
		}
		return nil, -1, nil
	}

	var outFiles []*ast.File
	fileCopies := make(map[int]*ast.File)
	ensureFileCopy := func(idx int) *ast.File {
		if fc, ok := fileCopies[idx]; ok {
			return fc
		}
		if outFiles == nil {
			outFiles = make([]*ast.File, len(files))
			copy(outFiles, files)
		}
		orig := files[idx]
		fc := *orig
		fc.Decls = make([]ast.Decl, len(orig.Decls))
		copy(fc.Decls, orig.Decls)
		fileCopies[idx] = &fc
		outFiles[idx] = &fc
		if res.clonedToOrig == nil {
			res.clonedToOrig = make(map[*ast.File]*ast.File)
		}
		res.clonedToOrig[&fc] = orig
		return &fc
	}

	synthSeq := 0
	nextSynthName := func() string {
		for {
			name := fmt.Sprintf("__llgo_ccalias_%d", synthSeq)
			synthSeq++
			if !usedNames[name] {
				usedNames[name] = true
				return name
			}
		}
	}

	res.isolated = make(map[string]bool)

	for fi, f := range files {
		if f == nil {
			continue
		}
		for di, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			var gdCopy *ast.GenDecl
			for si, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name == nil || !ts.Assign.IsValid() {
					continue
				}
				doc := ts.Doc
				if doc == nil && len(gd.Specs) == 1 {
					doc = gd.Doc
				}
				if _, ok := parseTypeCallConvDirective(doc); !ok {
					continue
				}
				ft, ftFileIdx, origIdent := resolveLocalFuncType(ts.Type, fi)
				if ft == nil {
					continue
				}
				res.isolated[ts.Name.Name] = true
				if origIdent == nil {
					// Direct `type CFn = func(...)`: go/types already allocates
					// a fresh *gotypes.Signature exclusively for this alias.
					continue
				}
				fCopy := ensureFileCopy(fi)
				if gdCopy == nil {
					gc := *gd
					gc.Specs = make([]ast.Spec, len(gd.Specs))
					copy(gc.Specs, gd.Specs)
					gdCopy = &gc
					fCopy.Decls[di] = gdCopy
				}
				tsCopy := *ts
				clonedFT, clonedIdents := cloneFuncTypeForAlias(ft)
				res.synthExprs = append(res.synthExprs, clonedFT)
				res.synthIdents = append(res.synthIdents, clonedIdents...)
				if ftFileIdx == fi {
					tsCopy.Type = clonedFT
				} else {
					// Keep the cloned FuncType in its declaring file so all
					// file-scoped imports and import aliases bind identically.
					synthName := nextSynthName()
					if res.synthNames == nil {
						res.synthNames = make(map[string]bool)
					}
					res.synthNames[synthName] = true
					declIdent := &ast.Ident{NamePos: ft.Pos(), Name: synthName}
					refIdent := &ast.Ident{NamePos: origIdent.Pos(), Name: synthName}
					res.synthIdents = append(res.synthIdents, declIdent, refIdent)
					origFileCopy := ensureFileCopy(ftFileIdx)
					origFileCopy.Decls = append(origFileCopy.Decls, &ast.GenDecl{
						TokPos: ft.Pos(),
						Tok:    token.TYPE,
						Specs: []ast.Spec{
							&ast.TypeSpec{
								Name:   declIdent,
								Assign: ft.Pos(),
								Type:   clonedFT,
							},
						},
					})
					tsCopy.Type = refIdent
				}
				gdCopy.Specs[si] = &tsCopy
				res.uses = append(res.uses, erasedAliasUse{ident: origIdent, targetName: origIdent.Name})
			}
		}
	}
	if outFiles != nil {
		res.files = outFiles
	}
	return res
}

func unwrapAstParens(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

func cloneFuncTypeForAlias(ft *ast.FuncType) (*ast.FuncType, []*ast.Ident) {
	if ft == nil {
		return nil, nil
	}
	var clonedIdents []*ast.Ident
	cloneList := func(fl *ast.FieldList) *ast.FieldList {
		if fl == nil {
			return nil
		}
		out := &ast.FieldList{
			Opening: fl.Opening,
			Closing: fl.Closing,
			List:    make([]*ast.Field, len(fl.List)),
		}
		for i, field := range fl.List {
			fc := *field
			if len(field.Names) > 0 {
				fc.Names = make([]*ast.Ident, len(field.Names))
				for j, name := range field.Names {
					nc := *name
					fc.Names[j] = &nc
					clonedIdents = append(clonedIdents, &nc)
				}
			}
			out.List[i] = &fc
		}
		return out
	}
	return &ast.FuncType{
		Func:    ft.Func,
		Params:  cloneList(ft.Params),
		Results: cloneList(ft.Results),
	}, clonedIdents
}

func makeMirrorGoInfo(info *Info) *gotypes.Info {
	if info == nil {
		return nil
	}
	goInfo := &gotypes.Info{}
	if info.Types != nil {
		goInfo.Types = make(map[ast.Expr]gotypes.TypeAndValue)
	}
	if info.Instances != nil {
		goInfo.Instances = make(map[*ast.Ident]gotypes.Instance)
	}
	if info.Defs != nil {
		goInfo.Defs = make(map[*ast.Ident]gotypes.Object)
	}
	if info.Uses != nil {
		goInfo.Uses = make(map[*ast.Ident]gotypes.Object)
	}
	if info.Implicits != nil {
		goInfo.Implicits = make(map[ast.Node]gotypes.Object)
	}
	if info.Selections != nil {
		goInfo.Selections = make(map[*ast.SelectorExpr]*gotypes.Selection)
	}
	if info.Scopes != nil {
		goInfo.Scopes = make(map[ast.Node]*gotypes.Scope)
	}
	if info.FileVersions != nil {
		initGoInfoFileVersions(goInfo)
	}
	return goInfo
}

func bridgeToGoSizes(ctx *Context, s Sizes) gotypes.Sizes {
	if s == nil {
		return ctx.GoSizes()
	}
	switch v := s.(type) {
	case *Context:
		return v.GoSizes()
	case *archSizes:
		if ctx != nil && ctx.spec.GOARCH == v.spec.GOARCH &&
			ctx.spec.WordSize == v.spec.WordSize && ctx.spec.MaxAlign == v.spec.MaxAlign {
			return ctx.GoSizes()
		}
		return NewContextWithConfig(ContextConfig{
			Spec:             v.spec,
			HostBasicLayouts: v.hostBasicLayouts,
			SIMDAlignments:   v.simdAlignments,
		}).GoSizes()
	case llgoContextProvider:
		if c := v.LLGoContext(); c != nil {
			return c.GoSizes()
		}
		return &goSizesBridge{ctx: ctx, sizes: s}
	case *StdSizes:
		return &gotypes.StdSizes{WordSize: v.WordSize, MaxAlign: v.MaxAlign}
	default:
		return &goSizesBridge{ctx: ctx, sizes: s}
	}
}

type goSizesBridge struct {
	ctx   *Context
	sizes Sizes
}

func (b *goSizesBridge) Alignof(T gotypes.Type) int64 {
	return b.sizes.Alignof(b.ctx.FromGo(T))
}

func (b *goSizesBridge) Sizeof(T gotypes.Type) int64 {
	return b.sizes.Sizeof(b.ctx.FromGo(T))
}

func (b *goSizesBridge) Offsetsof(fields []*gotypes.Var) []int64 {
	llFields := make([]*Var, len(fields))
	for i, f := range fields {
		if v, ok := b.ctx.FromGoObject(f).(*Var); ok {
			llFields[i] = v
		}
	}
	return b.sizes.Offsetsof(llFields)
}

type goImporterAdapter struct {
	ctx *Context
	imp Importer
}

func (a *goImporterAdapter) Import(path string) (*gotypes.Package, error) {
	return a.ImportFrom(path, "", 0)
}

func (a *goImporterAdapter) ImportFrom(path, dir string, mode gotypes.ImportMode) (*gotypes.Package, error) {
	var (
		pkg *Package
		err error
	)
	if impFrom, ok := a.imp.(ImporterFrom); ok {
		pkg, err = impFrom.ImportFrom(path, dir, ImportMode(mode))
	} else {
		pkg, err = a.imp.Import(path)
	}
	if pkg == nil {
		return nil, err
	}
	if pkg == Unsafe {
		return gotypes.Unsafe, err
	}
	if a.ctx != nil {
		return a.ctx.toGoPackage(pkg), err
	}
	if pkg.goPkg != nil {
		return pkg.goPkg, err
	}
	gp := gotypes.NewPackage(pkg.path, pkg.name)
	if pkg.complete {
		gp.MarkComplete()
	}
	return gp, err
}

func evalGoPackage(pkg *Package) (*Context, *gotypes.Package, func()) {
	if pkg == nil {
		return NewContext(), nil, func() {}
	}
	if pkg.goPkg != nil {
		ctx := pkg.ctx
		if ctx == nil {
			ctx = NewContext()
		}
		pkg.evalMu.RLock()
		if ctx.packageScopeNeedsSyncLocked(pkg) {
			pkg.evalMu.RUnlock()
			pkg.evalMu.Lock()
			ctx.syncPackageScopeToGoLocked(pkg)
			pkg.evalMu.Unlock()
			pkg.evalMu.RLock()
		}
		return ctx, pkg.goPkg, pkg.evalMu.RUnlock
	}
	// Use a fresh Context when bridging a publicly constructed Package so
	// subsequent pkg.Scope().Insert mutations are always observed without
	// mutating shared state across concurrent callers.
	ctx := NewContext()
	return ctx, ctx.toGoPackage(pkg), func() {}
}

// Eval returns the type and, if constant, the value for the expression expr,
// evaluated at position pos of package pkg (mirroring go/types.Eval).
func Eval(fset *token.FileSet, pkg *Package, pos token.Pos, expr string) (_ TypeAndValue, err error) {
	ctx, goPkg, unlock := evalGoPackage(pkg)
	defer unlock()
	gtv, err := gotypes.Eval(fset, goPkg, pos, expr)
	if err != nil {
		return TypeAndValue{}, wrapGoError(err)
	}
	ctx.mu.Lock()
	tv := ctx.fromGoTypeAndValueLocked(gtv)
	ctx.mu.Unlock()
	return tv, nil
}

// CheckExpr type-checks the expression expr as if it had appeared at position
// pos of package pkg (mirroring go/types.CheckExpr).
func CheckExpr(fset *token.FileSet, pkg *Package, pos token.Pos, expr ast.Expr, info *Info) error {
	ctx, goPkg, unlock := evalGoPackage(pkg)
	defer unlock()
	goInfo := makeMirrorGoInfo(info)
	err := gotypes.CheckExpr(fset, goPkg, pos, expr, goInfo)
	if info != nil && goInfo != nil {
		ctx.mu.Lock()
		ctx.populateInfoFromGoLocked(info, goInfo, "", nil)
		ctx.mu.Unlock()
	}
	return wrapGoError(err)
}

// Instantiate instantiates the generic type orig with type arguments targs
// (mirroring go/types.Instantiate).
func Instantiate(ctxt *Context, orig Type, targs []Type, validate bool) (Type, error) {
	if orig == nil {
		if validate {
			return nil, fmt.Errorf("cannot instantiate nil type")
		}
		return nil, nil
	}

	// Fast path: delegate directly to go/types.Instantiate when backed by go/types.
	var ownerCtx *Context
	switch o := orig.(type) {
	case *Named:
		ownerCtx = o.ctx
	case *Alias:
		if o.obj != nil && o.obj.pkg != nil {
			ownerCtx = o.obj.pkg.ctx
		}
	case *Signature:
		ownerCtx = o.ctx
	}
	if ownerCtx == nil {
		for _, a := range targs {
			switch ta := Unalias(a).(type) {
			case *Named:
				if ta.ctx != nil {
					ownerCtx = ta.ctx
				}
			case *Signature:
				if ta.ctx != nil {
					ownerCtx = ta.ctx
				}
			}
			if ownerCtx != nil {
				break
			}
		}
	}
	switch orig.(type) {
	case *Named, *Alias, *Signature:
		if goOrig := orig.goTypeRef(); goOrig != nil {
			goArgs := make([]gotypes.Type, len(targs))
			allGo := true
			for i, a := range targs {
				if a == nil || a.goTypeRef() == nil {
					allGo = false
					break
				}
				goArgs[i] = a.goTypeRef()
			}
			if allGo {
				instCtx := ctxt
				if instCtx == nil {
					instCtx = ownerCtx
				}
				if instCtx == nil {
					instCtx = NewContext()
				}
				res, err := gotypes.Instantiate(instCtx.ensureGoContext(), goOrig, goArgs, validate)
				if err != nil {
					return nil, wrapGoError(err)
				}
				lowerCtx := ownerCtx
				if lowerCtx == nil {
					lowerCtx = instCtx
				}
				return lowerCtx.FromGo(res), nil
			}
		}
	}

	// Synthetic pure-Go instantiation for programmatically constructed generic types.
	return instantiateSynthetic(orig, targs, validate)
}

func instantiateSynthetic(orig Type, targs []Type, validate bool) (Type, error) {
	var tparams *TypeParamList
	switch t := orig.(type) {
	case *Named:
		tparams = t.TypeParams()
	case *Alias:
		tparams = t.TypeParams()
	case *Signature:
		tparams = t.TypeParams()
	default:
		if validate {
			return nil, fmt.Errorf("cannot instantiate non-generic type %s", orig)
		}
		return orig, nil
	}

	n := tparams.Len()
	if validate {
		if n == 0 {
			return nil, fmt.Errorf("cannot instantiate non-generic type %s", orig)
		}
		if len(targs) != n {
			return nil, fmt.Errorf("got %d type arguments but want %d", len(targs), n)
		}
		for i := 0; i < n; i++ {
			tp := tparams.At(i)
			arg := targs[i]
			if tp != nil && tp.constraint != nil {
				if iface, ok := tp.constraint.Underlying().(*Interface); ok {
					if !Satisfies(arg, iface) {
						return nil, &ArgumentError{
							Index: i,
							Err:   fmt.Errorf("%s does not satisfy %s", arg, tp.constraint),
						}
					}
				}
			}
		}
	}

	subst := make(map[*TypeParam]Type, n)
	for i := 0; i < n && i < len(targs); i++ {
		subst[tparams.At(i)] = targs[i]
	}

	switch t := orig.(type) {
	case *Named:
		inst := &Named{
			synthInstOrigin: t,
			synthInstSubst:  subst,
			obj:             t.obj,
			origin:          t.Origin(),
			typeArgs:        newTypeList(targs),
			callConv:        t.callConv,
			hasCallConv:     t.hasCallConv,
			hostLayout:      t.hostLayout,
			nativeLayout:    t.nativeLayout,
			simdTagIndex:    t.simdTagIndex,
			simdVec:         t.simdVec,
		}
		return inst, nil

	case *Alias:
		seen := make(map[Type]Type)
		return &Alias{
			obj:         t.obj,
			origin:      t.Origin(),
			typeArgs:    newTypeList(targs),
			rhs:         substType(t.rhs, subst, seen),
			callConv:    t.callConv,
			hasCallConv: t.hasCallConv,
		}, nil

	case *Signature:
		seen := make(map[Type]Type)
		res := substSignature(t, subst, seen)
		res.typeParams = nil
		return res, nil
	}
	return orig, nil
}

func substType(t Type, subst map[*TypeParam]Type, seen map[Type]Type) Type {
	if t == nil {
		return nil
	}
	if cached, ok := seen[t]; ok {
		return cached
	}
	switch u := t.(type) {
	case *TypeParam:
		if rep, ok := subst[u]; ok {
			return rep
		}
		return u
	case *Basic, *SIMDVec:
		return u
	case *Pointer:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return &Pointer{elem: e, kind: u.kind}
	case *Slice:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return NewSlice(e)
	case *Array:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return &Array{elem: e, length: u.length, nativeStorage: u.nativeStorage}
	case *Map:
		k := substType(u.key, subst, seen)
		e := substType(u.elem, subst, seen)
		if k == u.key && e == u.elem {
			return u
		}
		return NewMap(k, e)
	case *Chan:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return NewChan(u.dir, e)
	case *Tuple:
		return substTuple(u, subst, seen)
	case *Signature:
		return substSignature(u, subst, seen)
	case *Struct:
		fields := make([]*Var, len(u.fields))
		changed := false
		for i, f := range u.fields {
			ft := substType(f.typ, subst, seen)
			if ft != f.typ {
				changed = true
				nf := NewField(f.pos, f.pkg, f.name, ft, f.embedded)
				nf.origin = f.Origin()
				fields[i] = nf
			} else {
				fields[i] = f
			}
		}
		if !changed {
			return u
		}
		return &Struct{
			fields:          fields,
			tags:            u.tags,
			hostLayout:      u.hostLayout,
			nativeLayout:    u.nativeLayout,
			isClosureStruct: u.isClosureStruct,
			simdTagIndex:    u.simdTagIndex,
		}
	case *Interface:
		u.resolve()
		out := &Interface{resolved: true, methodSet: true}
		seen[u] = out
		methods := make([]*Func, len(u.explicitMethods))
		changed := false
		for i, m := range u.explicitMethods {
			if sig, ok := m.typ.(*Signature); ok && sig != nil {
				params := substTuple(sig.params, subst, seen)
				results := substTuple(sig.results, subst, seen)
				if params != sig.params || results != sig.results {
					changed = true
					cp := *sig
					cp.params = params
					cp.results = results
					cp.recv = NewVar(m.pos, m.pkg, "", out)
					nm := NewFunc(m.pos, m.pkg, m.name, &cp)
					nm.origin = m.Origin()
					methods[i] = nm
					continue
				}
			}
			methods[i] = m
		}
		embs := make([]Type, len(u.embeddeds))
		for i, e := range u.embeddeds {
			ne := substType(e, subst, seen)
			if ne != e {
				changed = true
			}
			embs[i] = ne
		}
		if !changed {
			seen[u] = u
			return u
		}
		out.explicitMethods = methods
		out.embeddeds = embs
		return out.Complete()
	case *Named:
		if u.TypeArgs().Len() > 0 {
			args := make([]Type, u.TypeArgs().Len())
			changed := false
			for i := 0; i < u.TypeArgs().Len(); i++ {
				a := u.TypeArgs().At(i)
				na := substType(a, subst, seen)
				if na != a {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if existing, ok := seen[u.Origin()]; ok {
					if en, ok := existing.(*Named); ok && typeListMatches(en.TypeArgs(), args) {
						return en
					}
				}
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		} else if u.TypeParams().Len() > 0 {
			args := make([]Type, u.TypeParams().Len())
			changed := false
			for i := 0; i < u.TypeParams().Len(); i++ {
				tp := u.TypeParams().At(i)
				na := substType(tp, subst, seen)
				if na != tp {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if existing, ok := seen[u.Origin()]; ok {
					if en, ok := existing.(*Named); ok && typeListMatches(en.TypeArgs(), args) {
						return en
					}
				}
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		}
		return u
	case *Alias:
		if u.TypeArgs().Len() > 0 {
			args := make([]Type, u.TypeArgs().Len())
			changed := false
			for i := 0; i < u.TypeArgs().Len(); i++ {
				a := u.TypeArgs().At(i)
				na := substType(a, subst, seen)
				if na != a {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		} else if u.TypeParams().Len() > 0 {
			args := make([]Type, u.TypeParams().Len())
			changed := false
			for i := 0; i < u.TypeParams().Len(); i++ {
				tp := u.TypeParams().At(i)
				na := substType(tp, subst, seen)
				if na != tp {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		}
		return u
	case *Union:
		terms := make([]*Term, len(u.terms))
		changed := false
		for i, term := range u.terms {
			nt := substType(term.typ, subst, seen)
			if nt != term.typ {
				changed = true
				terms[i] = NewTerm(term.tilde, nt)
			} else {
				terms[i] = term
			}
		}
		if !changed {
			return u
		}
		return NewUnion(terms)
	}
	return t
}

func typeListMatches(tl *TypeList, args []Type) bool {
	if tl.Len() != len(args) {
		return false
	}
	for i, a := range args {
		if !Identical(tl.At(i), a) {
			return false
		}
	}
	return true
}

func substTuple(t *Tuple, subst map[*TypeParam]Type, seen map[Type]Type) *Tuple {
	if t == nil {
		return nil
	}
	vars := make([]*Var, len(t.vars))
	changed := false
	for i, v := range t.vars {
		vt := substType(v.typ, subst, seen)
		if vt != v.typ {
			changed = true
			nv := &Var{
				object:   object{pos: v.pos, pkg: v.pkg, name: v.name, typ: vt},
				origin:   v.Origin(),
				embedded: v.embedded,
				isField:  v.isField,
				kind:     v.kind,
			}
			vars[i] = nv
		} else {
			vars[i] = v
		}
	}
	if !changed {
		return t
	}
	return &Tuple{vars: vars}
}

func substSignature(s *Signature, subst map[*TypeParam]Type, seen map[Type]Type) *Signature {
	if s == nil {
		return nil
	}
	cp := *s
	if s.recv != nil {
		rt := substType(s.recv.typ, subst, seen)
		if rt != s.recv.typ {
			cp.recv = &Var{
				object: object{pos: s.recv.pos, pkg: s.recv.pkg, name: s.recv.name, typ: rt},
				origin: s.recv.Origin(),
				kind:   RecvVar,
			}
		}
	}
	cp.params = substTuple(s.params, subst, seen)
	cp.results = substTuple(s.results, subst, seen)
	return &cp
}
