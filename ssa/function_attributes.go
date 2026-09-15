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

// SetAttributes adds source properties to this module's function object.
func (f Function) SetAttributes(attrs FunctionAttributes) {
	attrs.apply(f.Prog.ctx, f.impl)
}
