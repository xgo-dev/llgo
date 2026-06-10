package ssa

import "github.com/goplus/llgo/internal/metadata"

// EmitUseNamedMethod records a MethodByName-like exact method-name demand for
// later whole-program deadcode analysis.
func (p Package) EmitUseNamedMethod(owner, name string) {
	if mb := p.MetaBuilder(); mb != nil {
		mb.AddUseNamedMethod(mb.String(owner), []metadata.Symbol{mb.String(name)})
	}
}

// EmitReflectMethod records a conservative reflection marker for later
// whole-program deadcode analysis.
func (p Package) EmitReflectMethod(owner string) {
	if mb := p.MetaBuilder(); mb != nil {
		mb.AddReflectMethod(mb.String(owner))
	}
}
