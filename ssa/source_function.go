package ssa

import (
	"go/ast"
	"strings"
)

// SourceFunction is the shared declaration of a function, before any backend
// creates its module-local definition or import. Syntax is available even when
// an export-data-only types.Package omits the private function's types.Func.
type SourceFunction struct {
	Syntax     *ast.FuncDecl
	Attributes FunctionAttributes
}

func (p Program) DeclareSourceFunction(name string, source SourceFunction) {
	if source.Attributes == 0 {
		return
	}
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	if previous := p.packageSyntax.sourceFunctions[name]; previous != nil {
		source.Attributes |= previous.Attributes
	}
	p.packageSyntax.sourceFunctions[name] = &source
}

// SourceFunctionAttributes resolves source declarations before function creation.
// Linknames may be collected in either order, so combine all known declarations
// of the symbol under one syntax snapshot. Generic origins are resolved by the
// frontend's function objects and never registered here.
func (p Program) SourceFunctionAttributes(name string) FunctionAttributes {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	data := p.packageSyntax
	symbol := data.sourceFunctionSymbol(name)
	var attrs FunctionAttributes
	for name, source := range data.sourceFunctions {
		if source.Attributes != 0 && data.sourceFunctionSymbol(name) == symbol {
			attrs |= source.Attributes
		}
	}
	return attrs
}

// Linkname resolution follows one mapping, matching the frontend. Prefixes
// describe calling conventions and are not part of the LLVM symbol name.
func (data *packageSyntaxData) sourceFunctionSymbol(name string) string {
	link, ok := data.linknames[name]
	if !ok {
		return name
	}
	link = strings.TrimPrefix(link, "C.")
	return strings.TrimPrefix(link, "stdcall.")
}
