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
	data := p.packageSyntax
	if origin, ok := data.attributeOrigins[name]; ok {
		name = origin
	}

	resolved := data.resolveAttributeName(name)
	var attrs FunctionAttributes
	for source, flags := range data.functionAttributes {
		if source == name || data.resolveAttributeName(source) == resolved {
			attrs |= flags
		}
	}
	return attrs
}

func (p Program) applyFunctionAttributes(fn llvm.Value, name string) {
	attrs := p.functionAttributes(name)
	if attrs&FunctionCold != 0 {
		fn.AddFunctionAttr(p.ctx.CreateEnumAttribute(llvm.AttributeKindID("cold"), 0))
	}
	if attrs&FunctionNoReturn != 0 {
		fn.AddFunctionAttr(p.ctx.CreateEnumAttribute(llvm.AttributeKindID("noreturn"), 0))
	}
}

func (data *packageSyntaxData) resolveAttributeName(name string) string {
	if link, ok := data.linknames[name]; ok {
		return strings.TrimPrefix(strings.TrimPrefix(link, "C."), "stdcall.")
	}
	return name
}
