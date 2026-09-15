package ssa

import "strings"

// DeclareFunction creates a source declaration before its Go signature or LLVM
// representation is available. Set its attributes during syntax preparation,
// before creating packages or backend Programs.
func (p Program) DeclareFunction(name string) Function {
	fn := &aFunction{sourceName: name}
	p.packageSyntax.mu.Lock()
	p.packageSyntax.functions = append(p.packageSyntax.functions, fn)
	p.packageSyntax.mu.Unlock()
	return fn
}

// importFunctionDeclarations gives each module its own function objects. The
// existing function table resolves symbols; no LLVM declarations are emitted
// until NewFunc or NewEnvFunc supplies a signature. Linkname aliases of one
// symbol contribute to the same object.
func (p Package) importFunctionDeclarations() {
	data := p.Prog.packageSyntax
	data.mu.RLock()
	defer data.mu.RUnlock()
	for _, source := range data.functions {
		name := data.functionSymbol(source.sourceName)
		fn := p.fns[name]
		if fn == nil {
			fn = &aFunction{sourceName: name, Pkg: p, Prog: p.Prog}
			p.fns[name] = fn
		}
		fn.SetAttributes(source.Attributes())
	}
}

// Follow one linkname mapping, as the frontend does. Calling-convention
// prefixes are not part of the LLVM symbol name.
func (data *packageSyntaxData) functionSymbol(name string) string {
	if link, ok := data.linknames[name]; ok {
		return strings.TrimPrefix(strings.TrimPrefix(link, "C."), "stdcall.")
	}
	return name
}
