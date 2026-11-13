package llvmintrinsic

import (
	"fmt"

	"github.com/xgo-dev/llvm"
)

// Intrinsic is one registration. Type is an optional refinement, not a second
// source of LLVM overloads, attributes or declarations. Availability and target
// feature policy belong to the caller, independently of type validation.
type Intrinsic struct {
	Name string
	Type Type
}

type Binding struct {
	id        int
	base      string
	name      string
	requested string
	signature llvm.Type
	overloads []llvm.Type
	plan      *Plan
}

func (b *Binding) ID() int { return b.id }

// Name is resolved in isolation. An unnamed identified struct may receive a
// different module-local suffix; Declare returns the actual declaration.
func (b *Binding) Name() string         { return b.name }
func (b *Binding) Signature() llvm.Type { return b.signature }

// Resolve checks refinements before invoking LLVM, then obtains the canonical
// signature, mangled name and immarg attributes without manufacturing a call.
func (i Intrinsic) Resolve(signature llvm.Type, options ...Option) (*Binding, error) {
	id := llvm.LookupIntrinsicID(i.Name)
	if id == 0 {
		return nil, fmt.Errorf("unknown LLVM intrinsic %q", i.Name)
	}
	if signature.IsNil() || signature.TypeKind() != llvm.FunctionTypeKind {
		return nil, fmt.Errorf("%s: expected a function signature", i.Name)
	}
	plan, err := Match(i.Type, signature, options...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i.Name, err)
	}
	overloads, ok := llvm.MatchIntrinsicSignature(id, signature)
	if !ok {
		return nil, fmt.Errorf("%s: signature does not match LLVM's intrinsic signature", i.Name)
	}
	base := llvm.IntrinsicName(id)
	mod := signature.Context().NewModule("intrinsic.types")
	name := mod.IntrinsicName(id, overloads)
	mod.Dispose()
	if i.Name != base && i.Name != name {
		return nil, fmt.Errorf("%s: name does not match declaration; expected %q", i.Name, name)
	}
	for _, index := range llvm.IntrinsicImmediateArguments(id, signature) {
		plan.rule(index).constant = true
	}
	return &Binding{id: id, base: base, name: name, requested: i.Name, signature: signature, overloads: overloads, plan: plan}, nil
}

func (b *Binding) CheckArguments(args ...llvm.Value) error {
	if err := b.plan.CheckArguments(args...); err != nil {
		return fmt.Errorf("%s: %w", b.name, err)
	}
	return nil
}

// Declare checks conflicts and inserts LLVM's declaration in the destination
// module. Each module owns its declaration; Binding itself is context-local.
func (b *Binding) Declare(mod llvm.Module) (llvm.Value, error) {
	if mod.IsNil() || mod.Context() != b.signature.Context() {
		return llvm.Value{}, fmt.Errorf("%s: module has a different LLVM context", b.name)
	}
	name := mod.IntrinsicName(b.id, b.overloads)
	if b.requested != b.base && b.requested != name {
		return llvm.Value{}, fmt.Errorf("%s: explicit intrinsic name does not match destination module", b.name)
	}
	if existing := mod.NamedValue(name); !existing.IsNil() {
		if existing.IsAFunction().IsNil() || existing.GlobalValueType() != b.signature ||
			!existing.IsDeclaration() || existing.FunctionCallConv() != llvm.CCallConv {
			return llvm.Value{}, fmt.Errorf("%s: conflicting LLVM intrinsic declaration", b.name)
		}
	}
	return mod.IntrinsicDeclaration(b.id, b.overloads), nil
}

// Call preserves the canonical function type, including LLVM varargs. Operand
// rejection happens before inserting a declaration or instruction.
func (b *Binding) Call(builder llvm.Builder, mod llvm.Module, args ...llvm.Value) (llvm.Value, error) {
	if err := b.CheckArguments(args...); err != nil {
		return llvm.Value{}, err
	}
	decl, err := b.Declare(mod)
	if err != nil {
		return llvm.Value{}, err
	}
	return llvm.CreateCall(builder, b.signature, decl, args), nil
}
