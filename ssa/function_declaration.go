package ssa

import (
	"go/token"
	"go/types"
)

// FunctionDeclaration owns the source information for one function or method.
// Distinct declarations remain independent even when they link to one symbol.
// It contains no LLVM state and is shared read-only after syntax preparation.
type FunctionDeclaration struct {
	pkg  *types.Package
	fset *token.FileSet
	name string
	pos  token.Pos

	linkname, export         string
	hasLink, hasExport       bool
	explicitEnv, noInterface bool
	wasm                     *wasmImport
	replacement              *FunctionDeclaration
}

func (f *FunctionDeclaration) Name() string            { return f.name }
func (f *FunctionDeclaration) SetLinkname(name string) { f.linkname, f.hasLink = name, true }
func (f *FunctionDeclaration) Linkname() (string, bool) {
	if f == nil {
		return "", false
	}
	return f.linkname, f.hasLink
}
func (f *FunctionDeclaration) SetExport(name string) { f.export, f.hasExport = name, true }
func (f *FunctionDeclaration) Export() (string, bool) {
	if f == nil {
		return "", false
	}
	return f.export, f.hasExport
}
func (f *FunctionDeclaration) SetExplicitEnv(value bool) { f.explicitEnv = value }
func (f *FunctionDeclaration) HasExplicitEnv() bool      { return f != nil && f.explicitEnv }
func (f *FunctionDeclaration) SetNoInterface(value bool) { f.noInterface = value }
func (f *FunctionDeclaration) NoInterface() bool         { return f != nil && f.noInterface }
func (f *FunctionDeclaration) SetWasmImport(module, name string) {
	f.wasm = &wasmImport{module: module, name: name}
}
func (f *FunctionDeclaration) WasmImport() (module, name string, ok bool) {
	if f == nil || f.wasm == nil {
		return "", "", false
	}
	return f.wasm.module, f.wasm.name, true
}

// DeclareFunction returns the object for this source declaration. A name is an
// index into declarations, not their identity: packages and source positions
// distinguish package variants and separate declarations of the same name.
func (p Program) DeclareFunction(pkg *types.Package, fset *token.FileSet, name string, pos token.Pos) *FunctionDeclaration {
	data := p.packageSyntax
	data.mu.Lock()
	defer data.mu.Unlock()
	for _, fn := range data.functions[name] {
		if fn.pkg == pkg && fn.fset == fset && fn.pos == pos {
			return fn
		}
	}
	for _, fn := range data.functions[name] {
		if fn.pkg == nil && fn.fset == nil && fn.pos == token.NoPos {
			fn.pkg, fn.fset, fn.pos = pkg, fset, pos
			return fn
		}
	}
	fn := &FunctionDeclaration{pkg: pkg, fset: fset, name: name, pos: pos}
	// Symbol-only clients can install a link or export before syntax is loaded.
	// Once its declaration exists, the function owns this information.
	if link, ok := data.linknames[name]; ok {
		fn.SetLinkname(link)
		delete(data.linknames, name)
	}
	if export, ok := data.exports[name]; ok {
		fn.SetExport(export)
		delete(data.exports, name)
	}
	data.functions[name] = append(data.functions[name], fn)
	return fn
}

// SourceFunctionDeclaration binds a frontend function to its preloaded source.
// A unique syntax identity can survive re-typechecking or package patching;
// matching package identity takes precedence when several variants exist.
func (p Program) SourceFunctionDeclaration(pkg *types.Package, fset *token.FileSet, name string, pos token.Pos) *FunctionDeclaration {
	data := p.packageSyntax
	data.mu.RLock()
	defer data.mu.RUnlock()
	var candidate *FunctionDeclaration
	matches := 0
	for _, fn := range data.functions[name] {
		if fn.fset != fset || fn.pos != pos {
			continue
		}
		if fn.pkg == pkg {
			return fn
		}
		candidate, matches = fn, matches+1
	}
	if matches == 1 {
		return candidate
	}
	return nil
}

// FunctionDeclarationOf resolves an imported function or method without syntax.
func (p Program) FunctionDeclarationOf(fn *types.Func) *FunctionDeclaration {
	if fn == nil {
		return nil
	}
	fn = fn.Origin()
	sig := fn.Type().(*types.Signature)
	name := FuncName(fn.Pkg(), fn.Name(), sig.Recv(), true)
	data := p.packageSyntax
	data.mu.RLock()
	defer data.mu.RUnlock()
	entries := data.functions[name]
	for _, decl := range entries {
		if decl.pkg == fn.Pkg() {
			return decl.Effective()
		}
	}
	if len(entries) == 1 {
		return entries[0].Effective()
	}
	return nil
}

// NamedFunctionDeclaration is for symbol-only clients without source identity.
// Source lowering uses SourceFunctionDeclaration or FunctionDeclarationOf.
func (p Program) NamedFunctionDeclaration(name string) *FunctionDeclaration {
	data := p.packageSyntax
	data.mu.RLock()
	defer data.mu.RUnlock()
	return data.namedFunction(name)
}

func (data *packageSyntaxData) namedFunction(name string) *FunctionDeclaration {
	entries := data.functions[name]
	if len(entries) != 0 {
		return entries[len(entries)-1]
	}
	return nil
}

// Effective returns the declaration selected by an explicit package patch.
// Linkname aliases never establish this relationship.
func (f *FunctionDeclaration) Effective() *FunctionDeclaration {
	if f != nil && f.replacement != nil {
		return f.replacement
	}
	return f
}

// ReplaceFunctionDeclarations records the build driver's package replacement.
// The original declarations retain their own metadata and source identity.
func (p Program) ReplaceFunctionDeclarations(pkg *types.Package, replacement *FunctionDeclaration) {
	data := p.packageSyntax
	data.mu.Lock()
	defer data.mu.Unlock()
	for _, fn := range data.functions[replacement.name] {
		if fn.pkg == pkg && fn != replacement {
			fn.replacement = replacement
		}
	}
}
