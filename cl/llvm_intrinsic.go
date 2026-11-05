package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func (p *context) llvmIntrinsicError(pos token.Pos, fn *ssa.Function, err error) {
	if !pos.IsValid() {
		pos = fn.Pos()
	}
	panic(fmt.Errorf("%s: %s: %w", p.fset.Position(pos), fn, err))
}

func (p *context) resolveLLVMIntrinsic(fn *ssa.Function, name string) *llssa.LLVMIntrinsic {
	if d := p.llvmIntrinsics[fn]; d != nil {
		return d
	}
	if fn.Origin() != nil {
		p.llvmIntrinsicError(fn.Pos(), fn, fmt.Errorf("LLVM intrinsic bindings do not support generic functions"))
	}
	if p.llvmIntrinsicHasEnv(fn) {
		p.llvmIntrinsicError(fn.Pos(), fn, fmt.Errorf("LLVM intrinsic bindings do not support //llgo:env"))
	}
	sig := p.patchType(fn.Signature).(*types.Signature)
	d, err := p.prog.ResolveLLVMIntrinsic(name, sig)
	if err != nil {
		p.llvmIntrinsicError(fn.Pos(), fn, err)
	}
	if p.llvmIntrinsics == nil {
		p.llvmIntrinsics = make(map[*ssa.Function]*llssa.LLVMIntrinsic)
	}
	p.llvmIntrinsics[fn] = d
	return d
}

func (p *context) llvmIntrinsicHasEnv(fn *ssa.Function) bool {
	if decl, ok := fn.Syntax().(*ast.FuncDecl); ok && fn.Pkg != nil {
		name := declarationFuncName(llssa.PathOf(fn.Pkg.Pkg), decl, fn.Object(), fn.Signature.Recv())
		return p.prog.HasClosureEnvDirective(p.goProg.Fset, name, decl.Pos())
	}
	return false
}

// Intrinsics have no addressable LLVM entry. Materialize a Go ABI entry only
// when the binding is used as a function value, using the same checked call
// lowering as direct calls. This also supports bodyless go:linkname bindings.
func (p *context) llvmIntrinsicValue(fn *ssa.Function) llssa.Function {
	if wrapper := p.llvmIntrinsicValues[fn]; wrapper != nil {
		return wrapper
	}
	_, name, _ := p.funcName(fn)
	d := p.resolveLLVMIntrinsic(fn, name)
	sig := p.patchType(fn.Signature).(*types.Signature)
	entry := llssa.FullName(fn.Pkg.Pkg, fn.Name()) + "$llvm"
	// One canonical entry across importing modules preserves function identity.
	// NewFuncEx supplies the required COMDAT on COFF targets.
	wrapper := p.pkg.NewFuncEx(entry, sig, llssa.GoDecl, false, true)
	b := wrapper.MakeBody(1)
	defer b.Dispose()
	args := make([]llssa.Expr, sig.Params().Len())
	for i := range args {
		args[i] = wrapper.Param(i)
	}
	ret, err := b.CallLLVMIntrinsic(d, args...)
	if err != nil {
		p.llvmIntrinsicError(fn.Pos(), fn, fmt.Errorf("cannot use LLVM intrinsic as a function value: %w", err))
	}
	b.Return(ret)
	b.EndBuild()
	if p.prog.FuncInfoMetadataEnabled() {
		pos := p.fset.Position(fn.Pos())
		p.pkg.EmitFuncInfo(wrapper.Name(), llssa.FullName(fn.Pkg.Pkg, fn.Name()), pos.Filename, pos.Line, pos.Column)
	}
	if p.llvmIntrinsicValues == nil {
		p.llvmIntrinsicValues = make(map[*ssa.Function]llssa.Function)
	}
	p.llvmIntrinsicValues[fn] = wrapper
	return wrapper
}

func (p *context) callLLVMIntrinsic(b llssa.Builder, act llssa.DoAction, fn *ssa.Function, call *ssa.CallCommon) llssa.Expr {
	if act != llssa.Call {
		p.llvmIntrinsicError(call.Pos(), fn, fmt.Errorf("LLVM intrinsic requires a direct call; use a Go wrapper for go or defer"))
	}
	_, name, _ := p.funcName(fn)
	d := p.resolveLLVMIntrinsic(fn, name)
	ret, err := b.CallLLVMIntrinsic(d, p.compileValues(b, call.Args, fnNormal)...)
	if err != nil {
		p.llvmIntrinsicError(call.Pos(), fn, err)
	}
	return ret
}
