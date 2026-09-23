/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/genmethod"
	"github.com/xgo-dev/llgo/internal/locality"
	llssa "github.com/xgo-dev/llgo/ssa"
)

// -----------------------------------------------------------------------------

type symInfo struct {
	file     string
	fullName string
	isVar    bool
}

type pkgSymInfo struct {
	store *directive.Store
	files map[string][]directive.LegacyLink // file => parsed links
	syms  map[string]symInfo                // name => isVar
}

func newPkgSymInfo(stores ...*directive.Store) *pkgSymInfo {
	store := new(directive.Store)
	if len(stores) > 0 {
		store = stores[0]
	}
	return &pkgSymInfo{
		store: store,
		files: make(map[string][]directive.LegacyLink),
		syms:  make(map[string]symInfo),
	}
}

func (p *pkgSymInfo) addSym(fset *token.FileSet, pos token.Pos, fullName, inPkgName string, isVar bool) {
	f := fset.File(pos)
	if fp := f.Position(pos); fp.Line > 2 {
		file := fp.Filename
		if _, ok := p.files[file]; !ok {
			p.files[file] = p.store.ReadLegacyLinks(file)
		}
		p.syms[inPkgName] = symInfo{file, fullName, isVar}
	}
}

func (p *pkgSymInfo) initLinknames(ctx *context) {
	for file, links := range p.files {
		for _, link := range links {
			ctx.applyLegacyLink(link, func(name string, isExport bool) (string, bool, bool) {
				if sym, ok := p.syms[name]; ok && file == sym.file {
					return sym.fullName, sym.isVar, true
				}
				return "", false, false
			})
		}
	}
}

// PkgKindOf returns the kind of a package.
func PkgKindOf(pkg *types.Package) (int, string) {
	scope := pkg.Scope()
	kind, param := pkgKindByScope(scope)
	if kind == PkgNormal {
		kind = pkgKindByPath(pkg.Path())
	}
	return kind, param
}

// PkgSkipsInit reports whether packages of kind are excluded from Go package
// initialization.
func PkgSkipsInit(kind int) bool {
	return kind >= PkgNoInit
}

// decl: a package that only contains declarations
// noinit: a package that does not need to be initialized
func pkgKind(v string) (int, string) {
	switch v {
	case "link":
		return PkgLinkIR, ""
	case "decl":
		return PkgDeclOnly, ""
	case "noinit":
		return PkgNoInit, ""
	default:
		// case "link:bc":
		//	return PkgLinkBitCode
		if strings.HasPrefix(v, "link:") { // "link: <libpath>"
			return PkgLinkExtern, v[5:]
		} else if strings.HasPrefix(v, "py.") { // "py.<module>"
			return PkgPyModule, v[3:]
		} else if strings.HasPrefix(v, "decl:") { // "decl: <param>"
			return PkgDeclOnly, v[5:]
		}
	}
	return PkgLLGo, ""
}

func pkgKindByScope(scope *types.Scope) (int, string) {
	if v, ok := scope.Lookup("LLGoPackage").(*types.Const); ok {
		if v := v.Val(); v.Kind() == constant.String {
			return pkgKind(constant.StringVal(v))
		}
		return PkgLLGo, ""
	}
	return PkgNormal, ""
}

func (p *context) importSourcePackage(pkg *types.Package) (*types.Package, int) {
	pkgPath := llssa.PathOf(pkg)
	scope := pkg.Scope()
	kind, _ := pkgKindByScope(scope)
	if kind == PkgNormal {
		if patch, ok := p.patches[pkgPath]; ok {
			pkg = patch.Alt.Pkg
			scope = pkg.Scope()
			if kind, _ = pkgKindByScope(scope); kind != PkgNormal {
				goto start
			}
		}
		return pkg, kind
	}
start:
	return pkg, kind
}

func (p *context) importPkg(pkg *types.Package, i *pkgInfo) {
	source, kind := p.importSourcePackage(pkg)
	if kind == PkgNormal {
		return
	}
	i.kind = kind
	if p.options.PreloadedSyntax {
		return
	}
	if syms, ok := p.importSources[source]; ok {
		syms.initLinknames(p)
		return
	}
	panic("import directives were not prepared for " + source.Path())
}

func (p *context) prepareImportSource(pkg *types.Package) {
	pkgPath := llssa.PathOf(pkg)
	source, kind := p.importSourcePackage(pkg)
	if kind == PkgNormal {
		return
	}
	if p.importSources == nil {
		p.importSources = make(map[*types.Package]*pkgSymInfo)
	}
	if _, ok := p.importSources[source]; ok {
		return
	}
	scope := source.Scope()
	fset := p.fset
	names := scope.Names()
	syms := newPkgSymInfo(p.prog.Directives())
	for _, name := range names {
		obj := scope.Lookup(name)
		switch obj := obj.(type) {
		case *types.Func:
			if pos := obj.Pos(); pos != token.NoPos {
				fullName, inPkgName := typesFuncName(pkgPath, obj)
				syms.addSym(fset, pos, fullName, inPkgName, false)
			}
		case *types.TypeName:
			if !obj.IsAlias() {
				if t, ok := obj.Type().(*types.Named); ok {
					for i, n := 0, t.NumMethods(); i < n; i++ {
						fn := t.Method(i)
						fullName, inPkgName := typesFuncName(pkgPath, fn)
						syms.addSym(fset, fn.Pos(), fullName, inPkgName, false)
					}
				}
			}
		case *types.Var:
			if pos := obj.Pos(); pos != token.NoPos {
				syms.addSym(fset, pos, pkgPath+"."+name, name, true)
			}
		}
	}
	p.importSources[source] = syms
}

func (p *context) initFiles(pkgPath string, files []*ast.File, cPkg bool) {
	// Every compile entry prepares records before constructing its context.
	records := p.prog.PackageDirectives(p.goTyps)
	if records == nil {
		panic("missing package directives for " + pkgPath)
	}
	for _, r := range records.Functions {
		if records.Names[r.Name] == r && r.ExportName != "" {
			p.pkg.SetExport(pkgPath+"."+r.Name, r.ExportName)
		}
	}
	for _, r := range records.Variables {
		if records.Names[r.Name] == r && r.ExportName != "" {
			p.pkg.SetExport(pkgPath+"."+r.Name, r.ExportName)
		}
	}
	p.skipall = records.Skip.All
	for _, name := range records.Skip.Names {
		p.skips[name] = none{}
	}
	for _, file := range records.Files {
		for _, node := range file.Syntax.Decls {
			if d, ok := node.(*ast.GenDecl); ok && d.Tok == token.IMPORT {
				g := file.Group(d.Doc)
				if g.LastSkip {
					fmt.Fprintf(os.Stderr, "DEPRECATED: llgo:skip on import is deprecated %v\n", g.LastLine)
				}
			}
		}
	}
}

// Collect skip names and skip other annotations, such as go: and llgo:
// llgo:skip symbol1 symbol2 ...
// llgo:skipall
func (p *context) collectSkipNames(line string) bool {
	all, names, ok := directive.LegacySkip(line)
	p.skipall = p.skipall || all
	for _, name := range names {
		p.skips[name] = none{}
	}
	return ok
}

func (p *context) collectSkipNamesByDoc(doc *ast.CommentGroup) {
	skip := new(directive.Store).Group(doc).Skip
	p.skipall = p.skipall || skip.All
	for _, name := range skip.Names {
		p.skips[name] = none{}
	}
}

// collectDeclarationDirectives caches source metadata needed after the syntax
// pass. funcPos is token.NoPos for non-function declarations.
func collectDeclarationDirectives(prog llssa.Program, fset *token.FileSet, doc *ast.CommentGroup, fullName, inPkgName string, funcPos token.Pos) {
	_, _ = collectDeclarationDirectivesWithOptions(prog, fset, doc, fullName, inPkgName, funcPos, Options{})
}

func collectDeclarationDirectivesWithOptions(prog llssa.Program, fset *token.FileSet, doc *ast.CommentGroup, fullName, inPkgName string, funcPos token.Pos, options Options) (bool, error) {
	g := prog.Directives().Group(doc)
	l, ok, err := g.DeclarationLink(inPkgName, options.ExportRename)
	if err != nil {
		return false, err
	}
	if ok {
		prog.SetLinkname(fullName, l.Target)
		if l.Export {
			prog.SetPackageExport(fullName, l.Target)
		}
	}
	if funcPos.IsValid() {
		if g.Function.ClosureEnv {
			prog.SetClosureEnvDirective(fset, fullName, funcPos)
		}
		if w := g.Function.WasmImport; w != nil {
			prog.SetWasmImport(fullName, w.Module, w.Name)
		}
	}
	return ok, nil
}

func (p *context) processLinknameByDoc(doc *ast.CommentGroup, fullName, inPkgName string, isVar, allowExport bool) bool {
	for _, r := range new(directive.Store).Group(doc).LegacyLinks(allowExport) {
		ret := p.applyLegacyLink(r, func(name string, export bool) (string, bool, bool) {
			return fullName, isVar, name == inPkgName || export && p.options.ExportRename
		})
		if ret != unknownDirective {
			return ret == hasLinkname
		}
	}
	return false
}

func (p *context) processNoInterfaceByDoc(doc *ast.CommentGroup, fullName string) {
	if new(directive.Store).Group(doc).NoInterface {
		p.prog.SetNoInterfaceMethod(fullName)
	}
}

const (
	noDirective = iota
	hasLinkname
	unknownDirective = -1
)

func (p *context) initLinkname(line string, allowExport bool, f func(string, bool) (string, bool, bool)) int {
	return p.applyLegacyLink(directive.ParseLegacyLink(line, allowExport), f)
}
func (p *context) applyLegacyLink(r directive.LegacyLink, f func(string, bool) (string, bool, bool)) int {
	if !r.Valid {
		return r.Status
	}
	if full, _, ok := f(r.Local, r.Export); ok {
		p.prog.SetLinkname(full, r.Target)
		if r.Export {
			p.pkg.SetExport(full, r.Target)
		}
	} else {
		if r.Export && p.options.ExportRename {
			return r.Status
		}
		if r.Export {
			panic(fmt.Sprintf("export comment has wrong name %q", r.Local))
		}
		fmt.Fprintln(os.Stderr, "==>", r.Raw)
		fmt.Fprintf(os.Stderr, "llgo: linkname %s not found and ignored\n", r.Local)
	}
	return r.Status
}

func recvTypeName(typ ast.Expr) string {
retry:
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return trecvTypeName(t.X, t.Index)
	case *ast.IndexListExpr:
		return trecvTypeName(t.X, t.Indices...)
	case *ast.ParenExpr:
		typ = t.X
		goto retry
	}
	panic("unreachable")
}

// TODO(xsw): support generic type
func trecvTypeName(t ast.Expr, indices ...ast.Expr) string {
	_ = indices
	return t.(*ast.Ident).Name
}

// inPkgName:
// - func: name
// - method: T.name, (*T).name
// fullName:
// - func: pkg.name
// - method: pkg.(T).name, pkg.(*T).name
func astFuncName(pkgPath string, fn *ast.FuncDecl) (string, string) {
	name := directive.FuncName(fn)
	return pkgPath + "." + name, name
}

func typesFuncName(pkgPath string, fn *types.Func) (fullName, inPkgName string) {
	sig := fn.Type().(*types.Signature)
	name := fn.Name()
	if recv := sig.Recv(); recv != nil {
		var method string
		t := recv.Type()
		if tp, ok := t.(*types.Pointer); ok {
			method = "(*" + tp.Elem().(*types.Named).Obj().Name() + ")." + name
		} else {
			method = t.(*types.Named).Obj().Name() + "." + name
		}
		return pkgPath + "." + method, method
	}
	return pkgPath + "." + name, name
}

// TODO(xsw): may can use typesFuncName
// fullName:
// - func: pkg.name
// - method: pkg.(T).name, pkg.(*T).name
func funcName(pkg *types.Package, fn *ssa.Function, org bool) string {
	// Closures in methods can be nested (closure inside closure inside method).
	// Walking only one Parent() loses the receiver for deeper nests, producing
	// names like "pkg.marshal$1$1" that can collide across receiver types.
	// Walk parents until we find a receiver.
	var recv *types.Var
	for f := fn; f != nil; f = f.Parent() {
		recv = f.Signature.Recv()
		if recv != nil {
			break
		}
	}
	// For wrappers, fall back to metadata available on fn itself.
	name := fn.Name()
	if recv == nil && strings.HasSuffix(name, "$thunk") {
		// For thunks, extract receiver from first parameter.
		if params := fn.Signature.Params(); params.Len() > 0 {
			recv = params.At(0)
		}
	} else if recv == nil && strings.HasSuffix(name, "$bound") && len(fn.FreeVars) == 1 {
		// For bound method wrappers, synthesize receiver var from free var type.
		recv = types.NewVar(token.NoPos, nil, "", fn.FreeVars[0].Type())
	}
	var fnName string
	if org := fn.Origin(); org != nil {
		fnName = org.Name()
		if fn.Signature.Recv() == nil {
			fnName += llssa.TypeArgs(fn.TypeArgs())
		}
	} else {
		fnName = fn.Name()
	}
	if recv != nil {
		// funcName's pkg is the receiver named type's package for wrappers.
		// Keep it identical to the receiver package passed by abiUncommonMethods
		// when it declares the itab target, or definition and reference symbols
		// will diverge for promoted unexported methods.
		if method, ok := fn.Object().(*types.Func); ok {
			fnName = llssa.MethodSymbolName(pkg, method, fnName)
			if genmethod.SupportsGenericMethods && genmethod.IsGenericMethod(method.Type()) {
				if targs := fn.TypeArgs(); len(targs) > 0 {
					fnName += llssa.TypeArgs(targs)
				}
			}
		}
		// Synthesized $thunk/$bound functions have no Object. Their references
		// are produced through this same funcName path, so the wrapper name is
		// already internally consistent and needs no declaring-package suffix.
	}
	return llssa.FuncName(pkg, fnName, recv, org)
}

func checkCgo(fnName string) bool {
	return len(fnName) > 4 && fnName[0] == '_' && fnName[2] == 'g' && fnName[3] == 'o' &&
		(fnName[1] == 'C' || fnName[1] == 'c') &&
		(fnName[4] == '_' || strings.HasPrefix(fnName[4:], "Check"))
}

var cgoIgnoredNames = map[string]none{
	"_Cgo_ptr":        {},
	"_Cgo_use":        {},
	"_cgoCheckResult": {},
	"cgoCheckResult":  {},
}

func cgoIgnored(fnName string) bool {
	_, ok := cgoIgnoredNames[fnName]
	return ok
}

const (
	ignoredFunc = iota
	goFunc      = int(llssa.InGo)
	cFunc       = int(llssa.InC)
	pyFunc      = int(llssa.InPython)
	stdcallFunc = int(llssa.InStdcall)
	llgoInstr   = -1

	llgoInstrBase   = 0x80
	llgoUnreachable = llgoInstrBase + 0
	llgoCstr        = llgoInstrBase + 1
	llgoAlloca      = llgoInstrBase + 2
	llgoAllocaCStr  = llgoInstrBase + 3
	llgoAllocaCStrs = llgoInstrBase + 4
	llgoAllocCStr   = llgoInstrBase + 5
	llgoAdvance     = llgoInstrBase + 6
	llgoIndex       = llgoInstrBase + 7
	llgoStringData  = llgoInstrBase + 8
	llgoString      = llgoInstrBase + 9
	llgoDeferData   = llgoInstrBase + 0xa

	llgoSigjmpbuf  = llgoInstrBase + 0xb
	llgoSigsetjmp  = llgoInstrBase + 0xc
	llgoSiglongjmp = llgoInstrBase + 0xd
	llgoFuncAddr   = llgoInstrBase + 0xe
	llgoSetjmp     = llgoInstrBase + 0x13
	llgoLongjmp    = llgoInstrBase + 0x14

	llgoPyList  = llgoInstrBase + 0x10
	llgoPyStr   = llgoInstrBase + 0x11
	llgoPyTuple = llgoInstrBase + 0x12

	llgoAtomicLoad    = llgoInstrBase + 0x1d
	llgoAtomicStore   = llgoInstrBase + 0x1e
	llgoAtomicCmpXchg = llgoInstrBase + 0x1f
	llgoAtomicOpBase  = llgoInstrBase + 0x20

	llgoAtomicXchg = int(llgoAtomicOpBase + llssa.OpXchg)
	llgoAtomicAdd  = int(llgoAtomicOpBase + llssa.OpAdd)
	llgoAtomicSub  = int(llgoAtomicOpBase + llssa.OpSub)
	llgoAtomicAnd  = int(llgoAtomicOpBase + llssa.OpAnd)
	llgoAtomicNand = int(llgoAtomicOpBase + llssa.OpNand)
	llgoAtomicOr   = int(llgoAtomicOpBase + llssa.OpOr)
	llgoAtomicXor  = int(llgoAtomicOpBase + llssa.OpXor)
	llgoAtomicMax  = int(llgoAtomicOpBase + llssa.OpMax)
	llgoAtomicMin  = int(llgoAtomicOpBase + llssa.OpMin)
	llgoAtomicUMax = int(llgoAtomicOpBase + llssa.OpUMax)
	llgoAtomicUMin = int(llgoAtomicOpBase + llssa.OpUMin)

	llgoCgoBase         = llgoInstrBase + 0x30
	llgoCgoCString      = llgoCgoBase + 0x0
	llgoCgoCBytes       = llgoCgoBase + 0x1
	llgoCgoGoString     = llgoCgoBase + 0x2
	llgoCgoGoStringN    = llgoCgoBase + 0x3
	llgoCgoGoBytes      = llgoCgoBase + 0x4
	llgoCgoCMalloc      = llgoCgoBase + 0x5
	llgoCgoCheckPointer = llgoCgoBase + 0x6
	llgoCgoCgocall      = llgoCgoBase + 0x7

	llgoAsm                = llgoInstrBase + 0x40
	llgoStackSave          = llgoInstrBase + 0x41
	llgoFuncPCABI0         = llgoInstrBase + 0x42
	llgoSkip               = llgoInstrBase + 0x43
	llgoSyscall            = llgoInstrBase + 0x44
	llgoAtomicCmpXchgOK    = llgoInstrBase + 0x45
	llgoAtomicAddReturnNew = llgoInstrBase + 0x46
	llgoBoolToUint8        = llgoInstrBase + 0x47
	llgoClosureEnv         = llgoInstrBase + 0x48
	llgoFloat32FromBits    = llgoInstrBase + 0x49
	llgoFloat32Bits        = llgoInstrBase + 0x4a
	llgoFloat64FromBits    = llgoInstrBase + 0x4b
	llgoFloat64Bits        = llgoInstrBase + 0x4c
	llgoUMulOverflow       = llgoInstrBase + 0x4d

	llgoAtomicOpLast = llgoAtomicOpBase + int(llssa.OpUMin)
)

func recvNamed(typ types.Type) *types.Named {
	if named := recvNamedOk(typ); named != nil {
		return named
	}
	panic(fmt.Errorf("invalid recv type: %v", typ))
}

func recvNamedOk(typ types.Type) *types.Named {
retry:
	switch t := types.Unalias(typ).(type) {
	case *types.Named:
		return t
	case *types.Pointer:
		typ = t.Elem()
		goto retry
	}
	return nil
}

func hasTypeArgs(named *types.Named) bool {
	targs := named.TypeArgs()
	return targs != nil && targs.Len() > 0
}

// extractTrampolineCName extracts the C function name from a trampoline function name.
// Handles patterns:
//   - "libc_XXX_trampoline" -> "XXX"
//   - "XXX_trampoline" -> "XXX"
//
// Returns empty string if name does not match the trampoline pattern.
func extractTrampolineCName(name string) string {
	if !strings.HasSuffix(name, "_trampoline") {
		return ""
	}
	base := strings.TrimSuffix(name, "_trampoline")
	base = strings.TrimPrefix(base, "libc_")
	return base
}

func (p *context) funcName(fn *ssa.Function) (*types.Package, string, int) {
	var pkg *types.Package
	var orgName string
	if origin := fn.Origin(); origin != nil {
		pkg = origin.Pkg.Pkg
		p.ensureLoaded(pkg)
		orgName = funcName(pkg, origin, true)
	} else {
		fname := fn.Name()
		if checkCgo(fname) && !cgoIgnored(fname) {
			return nil, fname, llgoInstr
		}
		if strings.HasPrefix(fname, "_cgoexp_") {
			return nil, fname, ignoredFunc
		}
		if isCgoExternSymbol(fn) {
			if _, ok := llgoInstrs[fname]; ok {
				return nil, fname, llgoInstr
			}
		}
		if fnPkg := fn.Pkg; fnPkg != nil {
			pkg = fnPkg.Pkg
		} else if recv := fn.Type().(*types.Signature).Recv(); recv != nil {
			if named := recvNamedOk(recv.Type()); named != nil && named.Obj().Pkg() != nil && (hasTypeArgs(named) || p.needsLinkOnce(fn)) {
				pkg = named.Obj().Pkg()
			} else {
				pkg = p.goTyps
			}
		} else {
			pkg = p.goTyps
		}
		p.ensureLoaded(pkg)
		orgName = funcName(pkg, fn, false)
	}
	obj := fn.Object()
	// Promoted method wrappers can expose the embedded method's object. Their
	// receiver and symbol belong to the wrapper, so do not inherit its linkname.
	if fn.Origin() == nil && fn.Synthetic != "" && fn.Syntax() == nil {
		obj = nil
	}
	if v, ok := p.prog.LinknameFor(p.directivePackage(pkg), obj, orgName); ok {
		if p.options.CExportWrappers {
			if export, ok := p.pkg.ExportFuncs()[orgName]; ok && export == v {
				return pkg, funcName(pkg, fn, false), goFunc
			}
		}
		if strings.HasPrefix(v, "C.") {
			return nil, v[2:], cFunc
		}
		if strings.HasPrefix(v, "stdcall.") {
			return nil, v[len("stdcall."):], stdcallFunc
		}
		if strings.HasPrefix(v, "py.") {
			return pkg, v[3:], pyFunc
		}
		if strings.HasPrefix(v, "llgo.") {
			return nil, v[5:], llgoInstr
		}
		return pkg, v, goFunc
	}
	// Stdlib compiler intrinsics that are defined as `panic("intrinsic")` in
	// source form. LLGo doesn't run Go escape analysis, so we can lower these to
	// a no-op.
	//
	// See: $(GOROOT)/src/hash/maphash/maphash.go: escapeForHash.
	if orgName == "hash/maphash.escapeForHash" {
		return nil, "skip", llgoInstr
	}
	// The 386 C ABI returns floating-point values through x87. Loading a
	// signaling NaN into x87 quiets it, so a normal call to the standard
	// library's pointer-based frombits helpers would not preserve the bit
	// pattern promised by package math. Lower all four bit conversions at the
	// caller on 386, matching the Go compiler's intrinsic treatment.
	if target := p.prog.Target(); target != nil && target.GOARCH == "386" {
		switch orgName {
		case "math.Float32frombits":
			return nil, "float32FromBits", llgoInstr
		case "math.Float32bits":
			return nil, "float32Bits", llgoInstr
		case "math.Float64frombits":
			return nil, "float64FromBits", llgoInstr
		case "math.Float64bits":
			return nil, "float64Bits", llgoInstr
		}
	}
	return pkg, funcName(pkg, fn, false), goFunc
}

const (
	ignoredVar = iota
	goVar      = int(llssa.InGo)
	cVar       = int(llssa.InC)
	pyVar      = int(llssa.InPython)
)

func (p *context) varName(pkg *types.Package, v *ssa.Global) (vName string, vtype int, define bool) {
	name := llssa.FullName(pkg, v.Name())
	// TODO(lijie): need a bettery way to process linkname (maybe alias)
	if !isCgoCfpvar(v.Name()) && !isCgoVar(v.Name()) {
		if v, ok := p.prog.LinknameFor(p.directivePackage(pkg), v.Object(), name); ok {
			if strings.HasPrefix(v, "stdcall.") {
				panic(fmt.Errorf("stdcall linkname namespace applies only to functions: %s", name))
			}
			if strings.HasPrefix(v, "go:") {
				if llssa.PathOf(pkg) == "runtime" {
					return v, goVar, true
				}
				return v, goVar, false
			}
			if pos := strings.IndexByte(v, '.'); pos >= 0 {
				if pos == 2 && v[0] == 'p' && v[1] == 'y' {
					return v[3:], pyVar, false
				}
				return replaceGoName(v, pos), goVar, false
			}
			return v, cVar, false
		}
	}
	return name, goVar, true
}

func (p *context) varOf(b llssa.Builder, v *ssa.Global) llssa.Expr {
	pkgTypes := p.ensureLoaded(v.Pkg.Pkg)
	pkg := p.pkg
	name, vtype, _ := p.varName(pkgTypes, v)
	if vtype == pyVar {
		if kind, mod := pkgKindByScope(pkgTypes.Scope()); kind == PkgPyModule {
			return b.PyNewVar(pysymPrefix+mod, name).Expr
		}
		panic("unreachable")
	}
	if local, ok := p.localVariableAddress(b, v, name); ok {
		return local
	}
	ret := pkg.VarOf(name)
	if ret == nil {
		ret = pkg.NewVar(name, p.patchType(v.Type()), llssa.Background(vtype))
	}
	return ret.Expr
}

func (p *context) ensureLoaded(pkgTypes *types.Package) *types.Package {
	if p.goTyps != pkgTypes {
		if _, ok := p.loaded[pkgTypes]; !ok {
			i := &pkgInfo{
				kind: pkgKindByPath(pkgTypes.Path()),
			}
			p.loaded[pkgTypes] = i
			p.importPkg(pkgTypes, i)
		}
	}
	return pkgTypes
}

// -----------------------------------------------------------------------------

const (
	pysymPrefix = "__llgo_py."
)

func (p *context) initPyModule() {
	if kind, mod := pkgKindByScope(p.goTyps.Scope()); kind == PkgPyModule {
		p.pyMod = mod
	}
}

// ParsePkgSyntax collects declaration directives in one syntax pass before SSA
// creation using default frontend options.
func ParsePkgSyntax(prog llssa.Program, fset *token.FileSet, pkg *types.Package, files []*ast.File) error {
	return ParsePkgSyntaxWithOptions(prog, fset, pkg, files, Options{})
}

// ParsePkgSyntaxWithOptions collects all Program-side declaration metadata.
// LLVM Package effects such as preserving //export symbols are applied later.
func ParsePkgSyntaxWithOptions(prog llssa.Program, fset *token.FileSet, pkg *types.Package, files []*ast.File, options Options) error {
	if pkg == nil || prog.PackageSyntaxParsed(pkg) {
		return nil
	}
	store := prog.Directives()
	sources := store.Files(files)
	if err := validateInternalRecords(fset, pkg.Path(), sources, options.AllowInternalDirectives); err != nil {
		return err
	}
	records := directive.Collect(sources, pkg.Name() == "C", options.ExportRename)
	path := llssa.PathOf(pkg)
	for _, file := range sources {
		for _, node := range file.Syntax.Decls {
			switch decl := node.(type) {
			case *ast.FuncDecl:
				if err := locality.ValidateDoc(fset, decl.Doc, store); err != nil {
					return err
				}
				if err := locality.ValidateFuncBody(fset, decl.Body, store); err != nil {
					return err
				}
				r := records.Functions[decl]
				if r.Err != nil {
					return r.Err
				}
				full := path + "." + r.Name
				if r.HasLinkname {
					prog.SetLinkname(full, r.Linkname)
				}
				if records.Names[r.Name] == r && r.ExportName != "" {
					prog.SetPackageExport(full, r.ExportName)
				}
				if r.NoInterface {
					prog.SetNoInterfaceMethod(full)
				}
				if r.ClosureEnv {
					prog.SetClosureEnvDirective(fset, full, decl.Pos())
				}
				if w := r.WasmImport; w != nil {
					prog.SetWasmImport(full, w.Module, w.Name)
				}
			case *ast.GenDecl:
				if decl.Tok == token.VAR {
					for _, spec := range decl.Specs {
						for _, id := range spec.(*ast.ValueSpec).Names {
							r := records.Variables[id]
							if r.Err != nil {
								return r.Err
							}
							if r.HasLinkname {
								prog.SetLinkname(path+"."+r.Name, r.Linkname)
							}
							if records.Names[r.Name] == r && r.ExportName != "" {
								prog.SetPackageExport(path+"."+r.Name, r.ExportName)
							}
						}
					}
					vars, err := locality.ScanPackageVar(fset, decl, store)
					if err != nil {
						return err
					}
					for _, v := range vars {
						prog.DeclareLocality(pkg, v.Name, v.Info)
					}
				} else {
					if err := locality.ValidateNonPackageVar(fset, decl, store); err != nil {
						return err
					}
					if decl.Tok == token.TYPE {
						for _, spec := range decl.Specs {
							r := records.Types[spec.(*ast.TypeSpec)]
							if r.Background != "" {
								prog.SetTypeBackground(pkg.Path()+"."+r.Name, toBackground(r.Background))
							}
						}
					}
				}
			}
		}
	}
	prog.SetPackageDirectives(pkg, records)
	prog.MarkPackageSyntaxParsed(pkg)
	return nil
}

func validateInternalDirectives(fset *token.FileSet, pkgPath string, files []*ast.File, allow bool) error {
	return validateInternalRecords(fset, pkgPath, new(directive.Store).Files(files), allow)
}
func validateInternalRecords(fset *token.FileSet, pkgPath string, files []*directive.File, allow bool) error {
	if allow || pkgPath == env.LLGoRuntimePkg || strings.HasPrefix(pkgPath, env.LLGoRuntimePkg+"/") {
		return nil
	}
	for _, file := range files {
		for _, d := range file.Internal {
			return fmt.Errorf("%s: //%s is only allowed in the Go standard library or %s", fset.Position(d.Pos), d.Name, env.LLGoRuntimePkg)
		}
	}
	return nil
}

func typeBackground(doc *ast.CommentGroup) string {
	return new(directive.Store).Group(doc).TypeBackground
}

func toBackground(bg string) llssa.Background {
	switch bg {
	case "C":
		return llssa.InC
	case "stdcall":
		return llssa.InStdcall
	}
	return llssa.InGo
}

// -----------------------------------------------------------------------------

func pkgKindByPath(pkgPath string) int {
	switch pkgPath {
	case "runtime/cgo", "unsafe":
		return PkgDeclOnly
	}
	return PkgNormal
}

func replaceGoName(v string, pos int) string {
	switch v[:pos] {
	case "runtime":
		return env.LLGoRuntimePkg + "/internal/runtime" + v[pos:]
	}
	return v
}

// -----------------------------------------------------------------------------

// directivePackage selects the effective declaration view for a package patch.
// Source properties still belong to each individual source declaration.
func (p *context) directivePackage(pkg *types.Package) *types.Package {
	if pkg != nil {
		if patch, ok := p.patches[llssa.PathOf(pkg)]; ok && patch.Types != nil {
			return patch.Types
		}
	}
	return pkg
}

// prepareImportSources is the standalone entrypoint's discovery boundary.
// All dependency source fallback happens here, before lowering begins.
func (p *context) prepareImportSources() {
	seen := make(map[*types.Package]bool)
	var visit func(*types.Package)
	visit = func(pkg *types.Package) {
		if pkg == nil || seen[pkg] {
			return
		}
		seen[pkg] = true
		if pkg != p.goTyps {
			p.prepareImportSource(pkg)
		}
		for _, dep := range pkg.Imports() {
			visit(dep)
		}
	}
	for _, pkg := range p.goProg.AllPackages() {
		visit(pkg.Pkg)
		// Standalone SSA clients can supply dependency bodies without their
		// AST files. Snapshot those declarations at the same boundary.
		funcs, _ := collectRuntimeCallerFunctions(pkg)
		for fn := range funcs {
			if origin := fn.Origin(); origin != nil {
				fn = origin
			}
			if decl, ok := fn.Syntax().(*ast.FuncDecl); ok {
				p.prog.Directives().Function(decl)
			}
		}
	}
}
