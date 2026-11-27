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
	"strconv"
	"strings"
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

	ctx           *Context
	fset          *token.FileSet
	pkg           *Package
	goPkg         *gotypes.Package
	goInfo        *gotypes.Info
	goImp         gotypes.ImporterFrom
	goChecker     *gotypes.Checker
	goVersion     string
	erasedAliases *erasedAliasState
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
		if pkg != nil && pkg.ctx != nil {
			ctx = pkg.ctx
		} else {
			ctx = NewContext()
		}
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
	var goImp gotypes.ImporterFrom
	if cfg.Importer != nil {
		goImp = &goImporterAdapter{ctx: ctx, imp: cfg.Importer}
		goConf.Importer = goImp
	}
	goConf.Sizes = bridgeToGoSizes(ctx, cfg.Sizes)

	goInfo := makeMirrorGoInfo(info)
	return &Checker{
		Info:      info,
		ctx:       ctx,
		fset:      fset,
		pkg:       pkg,
		goPkg:     goPkg,
		goInfo:    goInfo,
		goImp:     goImp,
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
	check.ctx.registerCheckedGoPackage(check.pkg, check.goPkg)
	checkFiles := files
	var prep erasedAliasPrep
	if goTypesErasesAliases() {
		if check.erasedAliases == nil {
			check.erasedAliases = newErasedAliasState()
		}
		prep = check.erasedAliases.prepare(files, check.goPkg, check.fset, check.goImp)
		checkFiles = prep.files
	}
	check.ctx.prescanTypeDirectives(check.pkg.path, files, prep.isolated)
	prevChildren := 0
	if check.goPkg != nil && check.goPkg.Scope() != nil {
		prevChildren = check.goPkg.Scope().NumChildren()
	}
	err := check.goChecker.Files(checkFiles)
	prep.restoreGoInfo(check.goInfo)
	check.ctx.populateFromGoPackage(check.pkg, check.goPkg, check.Info, check.goInfo, check.goVersion, files, prevChildren)
	applyDeclarationDirectives(check.pkg, check.Info, files)
	clearGoInfoBatchMaps(check.goInfo)
	if check.pkg != nil {
		if check.erasedAliases != nil && len(check.erasedAliases.synthNames) > 0 {
			// Keep check.pkg.goPkg intact so file/function lexical scopes remain
			// available for Eval/CheckExpr with source positions, while recording
			// all synthetic names created across batches to block them in CheckExpr
			// and filter them out of the public LLGo Package.Scope().
			if check.pkg.synthGoNames == nil {
				check.pkg.synthGoNames = make(map[string]bool, len(check.erasedAliases.synthNames))
			}
			for name := range check.erasedAliases.synthNames {
				check.pkg.synthGoNames[name] = true
			}
		}
		if check.pkg.scope != nil && len(check.pkg.synthGoNames) > 0 {
			for name := range check.pkg.synthGoNames {
				delete(check.pkg.scope.elems, name)
			}
		}
	}
	return wrapGoError(err)
}

type erasedAliasPrep struct {
	files        []*ast.File
	isolated     map[string]bool
	synthNames   map[string]bool
	synthIdents  []*ast.Ident
	synthExprs   []ast.Expr
	synthNodes   []ast.Node
	clonedToOrig map[*ast.File]*ast.File
}

func (p *erasedAliasPrep) restoreGoInfo(goInfo *gotypes.Info) {
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
		if goInfo.Instances != nil {
			delete(goInfo.Instances, id)
		}
	}
	for _, expr := range p.synthExprs {
		if goInfo.Types != nil {
			delete(goInfo.Types, expr)
		}
		if goInfo.Scopes != nil {
			delete(goInfo.Scopes, expr)
		}
		if sel, ok := expr.(*ast.SelectorExpr); ok && goInfo.Selections != nil {
			delete(goInfo.Selections, sel)
		}
	}
	for _, node := range p.synthNodes {
		if goInfo.Implicits != nil {
			delete(goInfo.Implicits, node)
		}
		if goInfo.Scopes != nil {
			delete(goInfo.Scopes, node)
		}
	}
	if len(p.clonedToOrig) > 0 {
		for cloned, orig := range p.clonedToOrig {
			if goInfo.Scopes != nil {
				if sc, ok := goInfo.Scopes[cloned]; ok {
					goInfo.Scopes[orig] = sc
					delete(goInfo.Scopes, cloned)
				}
			}
			remapGoInfoFileVersion(goInfo, cloned, orig)
		}
	}
}

func clearGoInfoBatchMaps(goInfo *gotypes.Info) {
	if goInfo == nil {
		return
	}
	for k := range goInfo.Defs {
		delete(goInfo.Defs, k)
	}
	for k := range goInfo.Uses {
		delete(goInfo.Uses, k)
	}
	for k := range goInfo.Implicits {
		delete(goInfo.Implicits, k)
	}
	for k := range goInfo.Types {
		delete(goInfo.Types, k)
	}
	for k := range goInfo.Selections {
		delete(goInfo.Selections, k)
	}
	for k := range goInfo.Scopes {
		delete(goInfo.Scopes, k)
	}
	for k := range goInfo.Instances {
		delete(goInfo.Instances, k)
	}
}

type localAliasDecl struct {
	expr ast.Expr
	file *ast.File
}

// erasedAliasState persists package-level alias declarations, identifier
// reservations, and synthetic alias names across incremental Checker.Files
// calls on Go toolchains where go/types erases aliases into *gotypes.Signature.
type erasedAliasState struct {
	aliasDecls map[string]localAliasDecl
	pkgDecls   map[string]bool
	usedNames  map[string]bool
	synthSeq   int
	synthNames map[string]bool
}

func newErasedAliasState() *erasedAliasState {
	return &erasedAliasState{
		aliasDecls: make(map[string]localAliasDecl),
		pkgDecls:   make(map[string]bool),
		usedNames:  make(map[string]bool),
		synthNames: make(map[string]bool),
	}
}

func (st *erasedAliasState) nextSynthName(goPkg *gotypes.Package) string {
	for {
		// Prefix with '$' (not a valid Go identifier character) so user Go
		// source and Eval expressions can never parse or resolve this name.
		name := fmt.Sprintf("$llgo_ccalias_%d", st.synthSeq)
		st.synthSeq++
		if st.usedNames[name] {
			continue
		}
		if goPkg != nil && goPkg.Scope() != nil && goPkg.Scope().Lookup(name) != nil {
			st.usedNames[name] = true
			continue
		}
		st.usedNames[name] = true
		st.synthNames[name] = true
		return name
	}
}

func fileImportDir(fset *token.FileSet, f *ast.File) string {
	if fset == nil || f == nil || f.Name == nil {
		return "."
	}
	p := fset.Position(f.Name.Pos()).Filename
	if i := strings.LastIndexAny(p, `/\`); i > 0 {
		return p[:i]
	}
	return "."
}

func lookupAliasedSignature(gp *gotypes.Package, name string, exportedOnly bool) *gotypes.Signature {
	if gp == nil || gp.Scope() == nil {
		return nil
	}
	obj, ok := gp.Scope().Lookup(name).(*gotypes.TypeName)
	if !ok || !obj.IsAlias() {
		return nil
	}
	if exportedOnly && !obj.Exported() {
		return nil
	}
	sig, _ := unaliasType(obj.Type()).(*gotypes.Signature)
	return sig
}

// prepare isolates native-callconv function type aliases for one Checker.Files
// batch on Go toolchains where go/types erases aliases into bare *gotypes.Signature
// values (Go 1.20/1.21, or Go 1.22 with gotypesalias=0). When a directive alias
// such as `//llgo:cdecl\ntype CFn = GoFn` aliases a plain Go function alias
// (local in the same or earlier batch, or imported from another package),
// isolating CFn allocates a distinct *gotypes.Signature for CFn while keeping
// the original RHS type-checked via a blank alias declaration so Info.Types,
// Info.Uses, and unused-import checks remain intact.
func (st *erasedAliasState) prepare(files []*ast.File, goPkg *gotypes.Package, fset *token.FileSet, goImp gotypes.ImporterFrom) erasedAliasPrep {
	res := erasedAliasPrep{files: files}
	if len(files) == 0 {
		return res
	}
	batchIndex := make(map[*ast.File]int, len(files))
	hasDirectiveAlias := false
	for fi, f := range files {
		if f == nil {
			continue
		}
		batchIndex[f] = fi
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id != nil {
				st.usedNames[id.Name] = true
			}
			return true
		})
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name != nil && d.Name.Name != "_" && d.Name.Name != "init" {
					st.pkgDecls[d.Name.Name] = true
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.CONST, token.VAR:
					for _, spec := range d.Specs {
						if vs, ok := spec.(*ast.ValueSpec); ok {
							for _, name := range vs.Names {
								if name != nil && name.Name != "_" {
									st.pkgDecls[name.Name] = true
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
						if ts.Name.Name != "_" {
							st.pkgDecls[ts.Name.Name] = true
						}
						if !ts.Assign.IsValid() {
							continue
						}
						st.aliasDecls[ts.Name.Name] = localAliasDecl{expr: ts.Type, file: f}
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

	resolveImportedSig := func(pkgIdent, selName string, f *ast.File) *gotypes.Signature {
		if f == nil || goImp == nil || !token.IsExported(selName) {
			return nil
		}
		dir := fileImportDir(fset, f)
		for _, imp := range f.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}
			impPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil || impPath == "C" || impPath == "unsafe" {
				continue
			}
			if imp.Name != nil {
				if imp.Name.Name == "." || imp.Name.Name == "_" || imp.Name.Name != pkgIdent {
					continue
				}
				gp, err := goImp.ImportFrom(impPath, dir, 0)
				if err != nil || gp == nil {
					return nil
				}
				return lookupAliasedSignature(gp, selName, true)
			}
			gp, err := goImp.ImportFrom(impPath, dir, 0)
			if err != nil || gp == nil {
				continue
			}
			if gp.Name() == pkgIdent {
				return lookupAliasedSignature(gp, selName, true)
			}
		}
		return nil
	}

	resolveDotImportedSig := func(name string, f *ast.File) *gotypes.Signature {
		if f == nil || goImp == nil || !token.IsExported(name) {
			return nil
		}
		dir := fileImportDir(fset, f)
		var matched *gotypes.Signature
		matches := 0
		for _, imp := range f.Imports {
			if imp == nil || imp.Path == nil {
				continue
			}
			impPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil || impPath == "C" || impPath == "unsafe" {
				continue
			}
			if imp.Name != nil {
				if imp.Name.Name == "_" {
					continue
				}
				if imp.Name.Name != "." {
					if imp.Name.Name == name {
						// File-scoped named import shadows dot-imports.
						return nil
					}
					continue
				}
				gp, err := goImp.ImportFrom(impPath, dir, 0)
				if err != nil || gp == nil || gp.Scope() == nil {
					continue
				}
				if obj := gp.Scope().Lookup(name); obj != nil && obj.Exported() {
					matches++
					if matches > 1 {
						return nil
					}
					matched = lookupAliasedSignature(gp, name, true)
				}
				continue
			}
			gp, err := goImp.ImportFrom(impPath, dir, 0)
			if err == nil && gp != nil && gp.Name() == name {
				return nil
			}
		}
		if matches == 1 {
			return matched
		}
		return nil
	}

	// resolveAliasTarget follows local and imported alias chains to either:
	//   - a direct *ast.FuncType literal on ts.Type (isDirectLiteral == true),
	//   - a *gotypes.Signature already resolved in an earlier batch or imported package (prevSig), or
	//   - an *ast.FuncType declared in the current batch (ft, ftFile).
	resolveAliasTarget := func(expr ast.Expr, startFile *ast.File) (ft *ast.FuncType, ftFile *ast.File, prevSig *gotypes.Signature, isDirectLiteral bool) {
		e := unwrapAstParens(expr)
		if f, ok := e.(*ast.FuncType); ok {
			return f, startFile, nil, true
		}
		curFile := startFile
		visited := make(map[string]bool)
		for {
			e = unwrapAstParens(e)
			switch x := e.(type) {
			case *ast.FuncType:
				return x, curFile, nil, false
			case *ast.SelectorExpr:
				if pkgID, ok := x.X.(*ast.Ident); ok && pkgID != nil && x.Sel != nil {
					if sig := resolveImportedSig(pkgID.Name, x.Sel.Name, curFile); sig != nil {
						return nil, curFile, sig, false
					}
				}
				return nil, nil, nil, false
			case *ast.Ident:
				if x == nil || visited[x.Name] {
					return nil, nil, nil, false
				}
				visited[x.Name] = true
				ad, exists := st.aliasDecls[x.Name]
				if !exists {
					if st.pkgDecls[x.Name] {
						return nil, nil, nil, false
					}
					if goPkg != nil && goPkg.Scope() != nil && goPkg.Scope().Lookup(x.Name) != nil {
						if sig := lookupAliasedSignature(goPkg, x.Name, false); sig != nil {
							return nil, curFile, sig, false
						}
						return nil, nil, nil, false
					}
					if sig := resolveDotImportedSig(x.Name, curFile); sig != nil {
						return nil, curFile, sig, false
					}
					return nil, nil, nil, false
				}
				if _, inBatch := batchIndex[ad.file]; !inBatch {
					if sig := lookupAliasedSignature(goPkg, x.Name, false); sig != nil {
						return nil, ad.file, sig, false
					}
					return nil, nil, nil, false
				}
				e = ad.expr
				curFile = ad.file
			default:
				return nil, nil, nil, false
			}
		}
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
				ft, ftFile, prevSig, isDirectLiteral := resolveAliasTarget(ts.Type, f)
				if ft == nil && prevSig == nil {
					continue
				}
				if isDirectLiteral {
					// Direct `type CFn = func(...)`: go/types already allocates
					// a fresh *gotypes.Signature exclusively for this alias.
					res.isolated[ts.Name.Name] = true
					continue
				}
				var newRHS ast.Expr
				var synthDeclFile *ast.File
				var synthDecl *ast.GenDecl
				if prevSig != nil {
					if goPkg == nil || goPkg.Scope() == nil {
						continue
					}
					// Target came from an earlier batch or an imported package:
					// allocate a distinct, type-identical *gotypes.Signature
					// directly without creating any synthetic *ast.File.
					synthName := st.nextSynthName(goPkg)
					freshSig := gotypes.NewSignatureType(
						prevSig.Recv(), nil, nil,
						prevSig.Params(), prevSig.Results(), prevSig.Variadic(),
					)
					if alt := goPkg.Scope().Insert(gotypes.NewTypeName(token.NoPos, goPkg, synthName, freshSig)); alt != nil {
						continue
					}
					if res.synthNames == nil {
						res.synthNames = make(map[string]bool)
					}
					res.synthNames[synthName] = true
					refIdent := &ast.Ident{NamePos: token.NoPos, Name: synthName}
					paren := &ast.ParenExpr{X: refIdent}
					res.synthIdents = append(res.synthIdents, refIdent)
					res.synthExprs = append(res.synthExprs, paren)
					newRHS = paren
				} else {
					cloner := &astExprCloner{}
					clonedFT := cloner.cloneFuncType(ft)
					if cloner.failed || clonedFT == nil {
						continue
					}
					res.synthIdents = append(res.synthIdents, cloner.idents...)
					res.synthExprs = append(res.synthExprs, cloner.exprs...)
					res.synthNodes = append(res.synthNodes, cloner.nodes...)
					if ftFile == f {
						newRHS = clonedFT
					} else {
						// Keep the cloned FuncType in its declaring file so all
						// file-scoped imports and import aliases bind identically.
						synthName := st.nextSynthName(goPkg)
						if res.synthNames == nil {
							res.synthNames = make(map[string]bool)
						}
						res.synthNames[synthName] = true
						declIdent := &ast.Ident{NamePos: token.NoPos, Name: synthName}
						refIdent := &ast.Ident{NamePos: token.NoPos, Name: synthName}
						res.synthIdents = append(res.synthIdents, declIdent, refIdent)
						synthDeclFile = ftFile
						synthDecl = &ast.GenDecl{
							TokPos: ft.Pos(),
							Tok:    token.TYPE,
							Specs: []ast.Spec{
								&ast.TypeSpec{
									Name:   declIdent,
									Assign: ft.Pos(),
									Type:   clonedFT,
								},
							},
						}
						newRHS = refIdent
					}
				}
				res.isolated[ts.Name.Name] = true
				fCopy := ensureFileCopy(fi)
				if gdCopy == nil {
					gc := *gd
					gc.Specs = make([]ast.Spec, len(gd.Specs))
					copy(gc.Specs, gd.Specs)
					gdCopy = &gc
					fCopy.Decls[di] = gdCopy
				}
				if synthDecl != nil {
					origFileCopy := ensureFileCopy(batchIndex[synthDeclFile])
					origFileCopy.Decls = append(origFileCopy.Decls, synthDecl)
				}
				tsCopy := *ts
				tsCopy.Type = newRHS
				gdCopy.Specs[si] = &tsCopy
				// Type-check the original RHS AST (e.g. `(GoFn)` or `dep.Fn`)
				// via a blank alias declaration in the same file so Info.Types,
				// Info.Uses, and unused-import checks match unmodified source.
				blankIdent := &ast.Ident{NamePos: token.NoPos, Name: "_"}
				res.synthIdents = append(res.synthIdents, blankIdent)
				fCopy.Decls = append(fCopy.Decls, &ast.GenDecl{
					TokPos: ts.Pos(),
					Tok:    token.TYPE,
					Specs: []ast.Spec{
						&ast.TypeSpec{
							Name:   blankIdent,
							Assign: ts.Assign,
							Type:   ts.Type,
						},
					},
				})
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

// astExprCloner deep-clones type expressions so re-checking a cloned FuncType
// never overwrites Info.Defs/Uses/Types/Scopes/Implicits/Selections/Instances
// of any original AST node, and clears positions so synthetic function scopes
// never overlap real source positions.
type astExprCloner struct {
	idents []*ast.Ident
	exprs  []ast.Expr
	nodes  []ast.Node
	failed bool
}

func (c *astExprCloner) cloneIdent(id *ast.Ident) *ast.Ident {
	if id == nil {
		return nil
	}
	cp := &ast.Ident{Name: id.Name}
	c.idents = append(c.idents, cp)
	return cp
}

func (c *astExprCloner) cloneFieldList(fl *ast.FieldList) *ast.FieldList {
	if fl == nil {
		return nil
	}
	out := &ast.FieldList{
		List: make([]*ast.Field, len(fl.List)),
	}
	for i, field := range fl.List {
		if field == nil {
			continue
		}
		fc := &ast.Field{}
		if len(field.Names) > 0 {
			fc.Names = make([]*ast.Ident, len(field.Names))
			for j, name := range field.Names {
				fc.Names[j] = c.cloneIdent(name)
			}
		}
		fc.Type = c.cloneExpr(field.Type)
		if field.Tag != nil {
			tc := &ast.BasicLit{Kind: field.Tag.Kind, Value: field.Tag.Value}
			fc.Tag = tc
			c.exprs = append(c.exprs, tc)
		}
		c.nodes = append(c.nodes, fc)
		out.List[i] = fc
	}
	return out
}

func (c *astExprCloner) cloneFuncType(ft *ast.FuncType) *ast.FuncType {
	if ft == nil {
		return nil
	}
	tparams := c.cloneFieldList(ft.TypeParams)
	params := c.cloneFieldList(ft.Params)
	results := c.cloneFieldList(ft.Results)
	out := &ast.FuncType{
		TypeParams: tparams,
		Params:     params,
		Results:    results,
	}
	c.exprs = append(c.exprs, out)
	return out
}

func (c *astExprCloner) cloneExprs(list []ast.Expr) []ast.Expr {
	if len(list) == 0 {
		return nil
	}
	out := make([]ast.Expr, len(list))
	for i, e := range list {
		out[i] = c.cloneExpr(e)
	}
	return out
}

func (c *astExprCloner) cloneExpr(expr ast.Expr) ast.Expr {
	if expr == nil || c.failed {
		return nil
	}
	var out ast.Expr
	switch e := expr.(type) {
	case *ast.Ident:
		return c.cloneIdent(e)
	case *ast.BasicLit:
		out = &ast.BasicLit{Kind: e.Kind, Value: e.Value}
	case *ast.ParenExpr:
		out = &ast.ParenExpr{X: c.cloneExpr(e.X)}
	case *ast.SelectorExpr:
		out = &ast.SelectorExpr{X: c.cloneExpr(e.X), Sel: c.cloneIdent(e.Sel)}
	case *ast.StarExpr:
		out = &ast.StarExpr{X: c.cloneExpr(e.X)}
	case *ast.Ellipsis:
		out = &ast.Ellipsis{Elt: c.cloneExpr(e.Elt)}
	case *ast.ArrayType:
		out = &ast.ArrayType{Len: c.cloneExpr(e.Len), Elt: c.cloneExpr(e.Elt)}
	case *ast.StructType:
		out = &ast.StructType{Fields: c.cloneFieldList(e.Fields), Incomplete: e.Incomplete}
	case *ast.FuncType:
		return c.cloneFuncType(e)
	case *ast.InterfaceType:
		out = &ast.InterfaceType{Methods: c.cloneFieldList(e.Methods), Incomplete: e.Incomplete}
	case *ast.MapType:
		out = &ast.MapType{Key: c.cloneExpr(e.Key), Value: c.cloneExpr(e.Value)}
	case *ast.ChanType:
		out = &ast.ChanType{Dir: e.Dir, Value: c.cloneExpr(e.Value)}
	case *ast.IndexExpr:
		out = &ast.IndexExpr{X: c.cloneExpr(e.X), Index: c.cloneExpr(e.Index)}
	case *ast.IndexListExpr:
		out = &ast.IndexListExpr{X: c.cloneExpr(e.X), Indices: c.cloneExprs(e.Indices)}
	case *ast.UnaryExpr:
		out = &ast.UnaryExpr{Op: e.Op, X: c.cloneExpr(e.X)}
	case *ast.BinaryExpr:
		out = &ast.BinaryExpr{X: c.cloneExpr(e.X), Op: e.Op, Y: c.cloneExpr(e.Y)}
	case *ast.CallExpr:
		out = &ast.CallExpr{Fun: c.cloneExpr(e.Fun), Args: c.cloneExprs(e.Args), Ellipsis: e.Ellipsis}
	case *ast.CompositeLit:
		out = &ast.CompositeLit{Type: c.cloneExpr(e.Type), Elts: c.cloneExprs(e.Elts), Incomplete: e.Incomplete}
	case *ast.KeyValueExpr:
		out = &ast.KeyValueExpr{Key: c.cloneExpr(e.Key), Value: c.cloneExpr(e.Value)}
	case *ast.SliceExpr:
		out = &ast.SliceExpr{
			X:      c.cloneExpr(e.X),
			Low:    c.cloneExpr(e.Low),
			High:   c.cloneExpr(e.High),
			Max:    c.cloneExpr(e.Max),
			Slice3: e.Slice3,
		}
	case *ast.TypeAssertExpr:
		out = &ast.TypeAssertExpr{X: c.cloneExpr(e.X), Type: c.cloneExpr(e.Type)}
	case *ast.FuncLit:
		out = &ast.FuncLit{Type: c.cloneFuncType(e.Type), Body: c.cloneBlockStmt(e.Body)}
	default:
		c.failed = true
		return nil
	}
	c.exprs = append(c.exprs, out)
	return out
}

func (c *astExprCloner) cloneBlockStmt(bs *ast.BlockStmt) *ast.BlockStmt {
	if bs == nil || c.failed {
		return nil
	}
	out := &ast.BlockStmt{List: c.cloneStmts(bs.List)}
	c.nodes = append(c.nodes, out)
	return out
}

func (c *astExprCloner) cloneStmts(list []ast.Stmt) []ast.Stmt {
	if len(list) == 0 {
		return nil
	}
	out := make([]ast.Stmt, len(list))
	for i, s := range list {
		out[i] = c.cloneStmt(s)
	}
	return out
}

func (c *astExprCloner) cloneStmt(stmt ast.Stmt) ast.Stmt {
	if stmt == nil || c.failed {
		return nil
	}
	var out ast.Stmt
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return c.cloneBlockStmt(s)
	case *ast.DeclStmt:
		out = &ast.DeclStmt{Decl: c.cloneDecl(s.Decl)}
	case *ast.EmptyStmt:
		out = &ast.EmptyStmt{Implicit: s.Implicit}
	case *ast.LabeledStmt:
		out = &ast.LabeledStmt{Label: c.cloneIdent(s.Label), Stmt: c.cloneStmt(s.Stmt)}
	case *ast.ExprStmt:
		out = &ast.ExprStmt{X: c.cloneExpr(s.X)}
	case *ast.SendStmt:
		out = &ast.SendStmt{Chan: c.cloneExpr(s.Chan), Value: c.cloneExpr(s.Value)}
	case *ast.IncDecStmt:
		out = &ast.IncDecStmt{X: c.cloneExpr(s.X), Tok: s.Tok}
	case *ast.AssignStmt:
		out = &ast.AssignStmt{Lhs: c.cloneExprs(s.Lhs), Tok: s.Tok, Rhs: c.cloneExprs(s.Rhs)}
	case *ast.GoStmt:
		call, _ := c.cloneExpr(s.Call).(*ast.CallExpr)
		out = &ast.GoStmt{Call: call}
	case *ast.DeferStmt:
		call, _ := c.cloneExpr(s.Call).(*ast.CallExpr)
		out = &ast.DeferStmt{Call: call}
	case *ast.ReturnStmt:
		out = &ast.ReturnStmt{Results: c.cloneExprs(s.Results)}
	case *ast.BranchStmt:
		out = &ast.BranchStmt{Tok: s.Tok, Label: c.cloneIdent(s.Label)}
	case *ast.IfStmt:
		out = &ast.IfStmt{
			Init: c.cloneStmt(s.Init),
			Cond: c.cloneExpr(s.Cond),
			Body: c.cloneBlockStmt(s.Body),
			Else: c.cloneStmt(s.Else),
		}
	case *ast.CaseClause:
		out = &ast.CaseClause{List: c.cloneExprs(s.List), Body: c.cloneStmts(s.Body)}
	case *ast.SwitchStmt:
		out = &ast.SwitchStmt{
			Init: c.cloneStmt(s.Init),
			Tag:  c.cloneExpr(s.Tag),
			Body: c.cloneBlockStmt(s.Body),
		}
	case *ast.TypeSwitchStmt:
		out = &ast.TypeSwitchStmt{
			Init:   c.cloneStmt(s.Init),
			Assign: c.cloneStmt(s.Assign),
			Body:   c.cloneBlockStmt(s.Body),
		}
	case *ast.CommClause:
		out = &ast.CommClause{Comm: c.cloneStmt(s.Comm), Body: c.cloneStmts(s.Body)}
	case *ast.SelectStmt:
		out = &ast.SelectStmt{Body: c.cloneBlockStmt(s.Body)}
	case *ast.ForStmt:
		out = &ast.ForStmt{
			Init: c.cloneStmt(s.Init),
			Cond: c.cloneExpr(s.Cond),
			Post: c.cloneStmt(s.Post),
			Body: c.cloneBlockStmt(s.Body),
		}
	case *ast.RangeStmt:
		out = &ast.RangeStmt{
			Key:   c.cloneExpr(s.Key),
			Value: c.cloneExpr(s.Value),
			Tok:   s.Tok,
			X:     c.cloneExpr(s.X),
			Body:  c.cloneBlockStmt(s.Body),
		}
	default:
		c.failed = true
		return nil
	}
	c.nodes = append(c.nodes, out)
	return out
}

func (c *astExprCloner) cloneDecl(decl ast.Decl) ast.Decl {
	if decl == nil || c.failed {
		return nil
	}
	switch d := decl.(type) {
	case *ast.GenDecl:
		specs := make([]ast.Spec, len(d.Specs))
		for i, sp := range d.Specs {
			specs[i] = c.cloneSpec(sp)
		}
		out := &ast.GenDecl{Tok: d.Tok, Lparen: d.Lparen, Specs: specs}
		c.nodes = append(c.nodes, out)
		return out
	default:
		c.failed = true
		return nil
	}
}

func (c *astExprCloner) cloneSpec(spec ast.Spec) ast.Spec {
	if spec == nil || c.failed {
		return nil
	}
	switch sp := spec.(type) {
	case *ast.ValueSpec:
		names := make([]*ast.Ident, len(sp.Names))
		for i, n := range sp.Names {
			names[i] = c.cloneIdent(n)
		}
		out := &ast.ValueSpec{
			Names:  names,
			Type:   c.cloneExpr(sp.Type),
			Values: c.cloneExprs(sp.Values),
		}
		c.nodes = append(c.nodes, out)
		return out
	case *ast.TypeSpec:
		out := &ast.TypeSpec{
			Name:       c.cloneIdent(sp.Name),
			TypeParams: c.cloneFieldList(sp.TypeParams),
			Assign:     sp.Assign,
			Type:       c.cloneExpr(sp.Type),
		}
		c.nodes = append(c.nodes, out)
		return out
	default:
		c.failed = true
		return nil
	}
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

type goImportCacheKey struct {
	path string
	dir  string
	mode gotypes.ImportMode
}

type goImporterAdapter struct {
	ctx   *Context
	imp   Importer
	cache map[goImportCacheKey]*gotypes.Package
}

func (a *goImporterAdapter) Import(path string) (*gotypes.Package, error) {
	return a.ImportFrom(path, "", 0)
}

func (a *goImporterAdapter) ImportFrom(path, dir string, mode gotypes.ImportMode) (*gotypes.Package, error) {
	key := goImportCacheKey{path: path, dir: dir, mode: mode}
	if a.cache != nil {
		if gp, ok := a.cache[key]; ok {
			return gp, nil
		}
	}
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
	var gp *gotypes.Package
	if pkg == Unsafe {
		gp = gotypes.Unsafe
	} else if a.ctx != nil {
		gp = a.ctx.toGoPackage(pkg)
	} else if pkg.goPkg != nil {
		gp = pkg.goPkg
	} else {
		gp = gotypes.NewPackage(pkg.path, pkg.name)
		if pkg.complete {
			gp.MarkComplete()
		}
	}
	if err == nil && gp != nil && gp.Complete() {
		if a.cache == nil {
			a.cache = make(map[goImportCacheKey]*gotypes.Package)
		}
		a.cache[key] = gp
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
	if pkg != nil && len(pkg.synthGoNames) > 0 && expr != nil {
		var leaked string
		ast.Inspect(expr, func(n ast.Node) bool {
			if leaked != "" {
				return false
			}
			if id, ok := n.(*ast.Ident); ok && id != nil && pkg.synthGoNames[id.Name] {
				leaked = id.Name
				return false
			}
			return true
		})
		if leaked != "" {
			return fmt.Errorf("undefined: %s", leaked)
		}
	}
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
