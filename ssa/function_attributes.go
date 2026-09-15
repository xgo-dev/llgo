package ssa

import "github.com/xgo-dev/llvm"

// FunctionAttributes describes properties independent of the Go signature.
type FunctionAttributes uint8

const (
	FunctionCold FunctionAttributes = 1 << iota
	FunctionNoReturn
)

func (attrs FunctionAttributes) apply(ctx llvm.Context, fn llvm.Value) {
	if attrs&FunctionCold != 0 {
		fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("cold"), 0))
	}
	if attrs&FunctionNoReturn != 0 {
		fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noreturn"), 0))
	}
}

// Attributes returns the properties carried by this function.
func (f Function) Attributes() FunctionAttributes {
	return f.attributes
}

// SetAttributes adds properties to a source declaration or a module function.
// Properties set before LLVM materialization are applied when it is created.
func (f Function) SetAttributes(attrs FunctionAttributes) {
	f.attributes |= attrs
	if !f.impl.IsNil() {
		attrs.apply(f.Prog.ctx, f.impl)
	}
}
