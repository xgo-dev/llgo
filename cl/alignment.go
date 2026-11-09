package cl

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/xgo-dev/llgo/internal/genmethod"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// Validate source and instantiated SSA before emitting package bodies. Invalid
// native signatures are ordinary compilation errors, not lowering panics to be
// recovered at the package boundary. Follow this package's functions and their
// instantiated/synthetic children, rather than rescanning the entire program.
func (p *context) validatePackageAlignment(pkg *ssa.Package) error {
	validate := p.prog.NativeAlignmentValidator()
	positioned := func(err error, pos token.Pos) error {
		if diagnostic, ok := err.(*llssa.AlignmentError); ok && diagnostic.Pos.IsValid() {
			pos = diagnostic.Pos
		}
		return fmt.Errorf("%s: %w", p.fset.Position(pos), err)
	}
	checkType := func(t types.Type, cc llssa.CallConv, pos token.Pos) error {
		if err := validate(t, cc); err != nil {
			return positioned(err, pos)
		}
		return nil
	}

	var work []*ssa.Function
	names := make([]string, 0, len(pkg.Members))
	for name := range pkg.Members {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, skip := p.skips[name]; skip {
			continue
		}
		member := pkg.Members[name]
		if fn, ok := member.(*ssa.Function); ok {
			work = append(work, fn)
			continue
		}
		cc := llssa.GoDecl
		if global, ok := member.(*ssa.Global); ok {
			_, kind, _ := p.varName(global.Pkg.Pkg, global)
			if kind == cVar {
				cc = llssa.CDecl
			}
		}
		if err := checkType(member.Type(), cc, member.Pos()); err != nil {
			return err
		}
		if typ, ok := member.(*ssa.Type); ok {
			for _, receiver := range []types.Type{typ.Type(), types.NewPointer(typ.Type())} {
				methods := p.goProg.MethodSets.MethodSet(receiver)
				for i := 0; i < methods.Len(); i++ {
					if genmethod.SupportsGenericMethods && genmethod.IsGenericMethod(methods.At(i).Type()) {
						continue
					}
					if fn := p.methodValue(methods.At(i)); fn != nil {
						work = append(work, fn)
					}
				}
			}
		}
	}
	seen := make(map[*ssa.Function]bool)
	var operands []*ssa.Value
	for len(work) != 0 {
		fn := work[0]
		work = work[1:]
		if seen[fn] || fn.TypeParams().Len() != 0 && len(fn.TypeArgs()) == 0 {
			continue
		}
		seen[fn] = true
		_, _, kind := p.funcName(fn)
		cc := llssa.GoDecl
		if kind == cFunc || kind == stdcallFunc {
			cc = llssa.CDecl
		}
		// Export wrappers also expose a native boundary, even when the body
		// retains the Go calling convention.
		if decl, ok := fn.Syntax().(*ast.FuncDecl); ok && fn.Object() != nil && fn.Object().Pkg() != nil {
			fullName := declarationFuncName(llssa.PathOf(fn.Object().Pkg()), decl, fn.Object(), fn.Signature.Recv())
			if _, exported := p.prog.PackageExport(fullName); exported {
				cc = llssa.CDecl
			}
		}
		if err := checkType(fn.Signature, cc, fn.Pos()); err != nil {
			return err
		}
		if fn.Pkg != pkg && fn.Pkg != nil && !p.needsLinkOnce(fn) {
			continue
		}
		work = append(work, fn.AnonFuncs...)
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				pos := instr.Pos()
				if !pos.IsValid() {
					pos = fn.Pos()
				}
				if value, ok := instr.(ssa.Value); ok {
					if err := checkType(value.Type(), llssa.GoDecl, pos); err != nil {
						return err
					}
				}
				operands = instr.Operands(operands[:0])
				for _, operand := range operands {
					if operand == nil || *operand == nil {
						continue
					}
					if err := checkType((*operand).Type(), llssa.GoDecl, pos); err != nil {
						return err
					}
					if callee, ok := (*operand).(*ssa.Function); ok {
						work = append(work, callee)
					}
				}
				if call, ok := instr.(ssa.CallInstruction); ok && llssa.HasNameValist(call.Common().Signature()) {
					if err := validateAlignmentVarargs(call.Common()); err != nil {
						return positioned(err, pos)
					}
				}
			}
		}
	}
	return nil
}

// C ellipsis values are represented by MakeInterface stores into a synthetic
// array. Lowering passes their concrete operands, so validate those types rather
// than the []any descriptor (also for generic instances, go, and defer calls).
func validateAlignmentVarargs(call *ssa.CallCommon) error {
	if len(call.Args) == 0 {
		return nil
	}
	slice, ok := call.Args[len(call.Args)-1].(*ssa.Slice)
	if !ok {
		// nil ellipsis or a wrapper forwarding its parameters
		return nil
	}
	refs := slice.X.Referrers()
	if refs == nil {
		return nil
	}
	for _, ref := range *refs {
		index, ok := ref.(*ssa.IndexAddr)
		if !ok || index.Referrers() == nil {
			continue
		}
		for _, ref := range *index.Referrers() {
			store, ok := ref.(*ssa.Store)
			if !ok || store.Addr != index {
				continue
			}
			if value, ok := store.Val.(*ssa.MakeInterface); ok {
				if err := llssa.ValidateNativeValueAlignment(value.X.Type()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
