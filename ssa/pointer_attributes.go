package ssa

import (
	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
)

// CheckPointerEffects also accepts functions recreated by ABI conversion.
func (p Program) CheckPointerEffects(fn llvm.Value, reason string) error {
	return funcattrs.CheckPointerEffects(fn, p.pointerEffects[fn.Name()], reason)
}

// CheckImplicitRuntimeEffects guards compiler-owned runtime protocols. Explicit
// source operations remain the responsibility of the annotation's author.
func (f Function) CheckImplicitRuntimeEffects(reason string) {
	if err := f.Prog.CheckPointerEffects(f.impl, reason); err != nil {
		panic(err)
	}
}
