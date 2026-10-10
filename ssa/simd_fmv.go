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

// AllowSIMDDispatcherInlining separates a compiler-required physical source
// frame from the synthetic FMV entry. The implementation retains its frame;
// the tail-only dispatcher may inline without changing the source call stack.
// Callers must not set this for an explicit noinline directive or -l.
func (f Function) AllowSIMDDispatcherInlining() {
	if f.Prog.Target().GOARCH == "amd64" {
		f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute("llgo.fmv.inline-entry", ""))
	}
}
