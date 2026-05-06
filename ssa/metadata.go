package ssa

import (
	"github.com/goplus/llgo/internal/semmeta"
)

// EmitUseNamedMethod records a MethodByName-like exact method-name demand for
// later whole-program deadcode analysis.
func (p Package) EmitUseNamedMethod(owner, name string) {
	p.semMetaEmitter.AddUseNamedMethod(semmeta.Symbol(owner), name)
}

// EmitReflectMethod records a conservative reflection marker for later
// whole-program deadcode analysis.
func (p Package) EmitReflectMethod(owner string) {
	p.semMetaEmitter.AddReflectMethod(semmeta.Symbol(owner))
}
