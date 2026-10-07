package ssa

// MarkSIMDFMV records compiler-checked source identities for the early LLVM
// transform. An available variant is a cross-package ABI promise: the defining
// package emits it, and guarded callers may refer to it without LTO.
func (f Function) MarkSIMDFMV(available bool, query string) {
	if f.Prog.Target().GOARCH != "amd64" {
		return
	}
	if available {
		f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute("llgo.fmv.avx2-entry", ""))
	}
	if query != "" {
		f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute("llgo.cpu.query", query))
	}
}
