package ssa

import (
	"strings"

	"github.com/xgo-dev/llvm"
)

// FunctionAttributes describes properties independent of the Go signature.
type FunctionAttributes uint8

const (
	FunctionCold FunctionAttributes = 1 << iota
	FunctionNoReturn
)

func (p Program) SetFunctionAttributes(name string, attrs FunctionAttributes) {
	p.packageSyntax.mu.Lock()
	p.packageSyntax.functionAttributes[name] |= attrs
	p.packageSyntax.mu.Unlock()
}

// Generic instances inherit the properties of their source declaration.
func (p Program) SetFunctionAttributeOrigin(instance, origin string) {
	if instance == origin {
		return
	}
	p.packageSyntax.mu.Lock()
	p.packageSyntax.attributeOrigins[instance] = origin
	p.packageSyntax.mu.Unlock()
}

func (p Program) functionAttributes(name string) FunctionAttributes {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	return p.packageSyntax.lookupFunctionAttributes(name)
}

// lookupFunctionAttributes combines declarations naming the same symbol. Resolve
// a generic instance to its source declaration before matching aliases. The
// caller holds the syntax lock, so both mappings come from the same snapshot.
func (data *packageSyntaxData) lookupFunctionAttributes(name string) FunctionAttributes {
	if origin, ok := data.attributeOrigins[name]; ok {
		name = origin
	}
	symbol := data.functionAttributeSymbol(name)
	var attrs FunctionAttributes
	for source, flags := range data.functionAttributes {
		if data.functionAttributeSymbol(source) == symbol {
			attrs |= flags
		}
	}
	return attrs
}

// functionAttributeSymbol follows one linkname mapping, matching the frontend.
// Calling-convention prefixes are not part of the LLVM symbol name. The caller
// holds the syntax lock; this deliberately does not follow alias chains.
func (data *packageSyntaxData) functionAttributeSymbol(name string) string {
	link, ok := data.linknames[name]
	if !ok {
		return name
	}
	link = strings.TrimPrefix(link, "C.")
	return strings.TrimPrefix(link, "stdcall.")
}

func (p Program) applyFunctionAttributes(fn llvm.Value, name string) {
	p.functionAttributes(name).apply(p.ctx, fn)
}

func (attrs FunctionAttributes) apply(ctx llvm.Context, fn llvm.Value) {
	if attrs&FunctionCold != 0 {
		fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("cold"), 0))
	}
	if attrs&FunctionNoReturn != 0 {
		fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noreturn"), 0))
	}
}
