package llvmintrinsic

import (
	"errors"
	"fmt"
	"maps"

	"github.com/xgo-dev/llvm"
)

type unboundReference string

func (r unboundReference) Error() string { return fmt.Sprintf("unbound type reference %q", string(r)) }

type state struct {
	refs    map[string]llvm.Type
	pending []func() error
	layout  string
	plan    *Plan
}

func (s *state) deferCheck(check func() error) { s.pending = append(s.pending, check) }
func (s *state) run() error {
	for len(s.pending) != 0 {
		checks := s.pending
		s.pending = nil
		progress := false
		var missing error
		for _, check := range checks {
			err := check()
			if err == nil {
				progress = true
				continue
			}
			var unbound unboundReference
			if !errors.As(err, &unbound) {
				return err
			}
			missing = err
			s.pending = append(s.pending, check)
		}
		if !progress && len(s.pending) != 0 {
			return missing
		}
	}
	return nil
}

type Option func(*state)

func WithDataLayout(layout string) Option { return func(s *state) { s.layout = layout } }

// Plan contains declaration checks compiled once and the remaining operand
// checks. It neither changes types nor inserts conversions.
type Plan struct {
	signature llvm.Type
	rules     map[int]*operandRule
	tail      Type
	refs      map[string]llvm.Type
	layout    string
}
type interval struct {
	low, high uint64
	unsigned  bool
}

func (i interval) String() string {
	if i.unsigned {
		return fmt.Sprintf("[%d, %d)", i.low, i.high)
	}
	return fmt.Sprintf("[%d, %d)", int64(i.low), int64(i.high))
}

type operandRule struct {
	constant bool
	ranges   []interval
}

func newState(signature llvm.Type, options ...Option) *state {
	s := &state{refs: make(map[string]llvm.Type), plan: &Plan{signature: signature, rules: make(map[int]*operandRule)}}
	if !signature.IsNil() && signature.TypeKind() == llvm.FunctionTypeKind {
		s.refs["$result"] = signature.ReturnType()
		for i, param := range signature.ParamTypes() {
			s.refs[fmt.Sprintf("$arg%d", i)] = param
		}
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func Match(pattern Type, actual llvm.Type, options ...Option) (*Plan, error) {
	s := newState(actual, options...)
	if err := match(pattern, s, actual, location{path: "signature", arg: -1}); err != nil {
		return nil, err
	}
	if err := s.run(); err != nil {
		return nil, err
	}
	s.plan.refs, s.plan.layout = s.refs, s.layout
	return s.plan, nil
}

func (p *Plan) rule(index int) *operandRule {
	rule := p.rules[index]
	if rule == nil {
		rule = &operandRule{}
		p.rules[index] = rule
	}
	return rule
}

func (r *operandRule) check(index int, value llvm.Value) error {
	if r.constant && value.IsAConstantInt().IsNil() && value.IsAConstantFP().IsNil() {
		return fmt.Errorf("argument %d must be an immediate constant", index+1)
	}
	for _, interval := range r.ranges {
		if !integerInRange(value, interval.low, interval.high, interval.unsigned) {
			return fmt.Errorf("argument %d must be a constant integer in %s", index+1, interval)
		}
	}
	return nil
}

// CheckArguments validates actual logical LLVM operands, including each
// statically expanded variadic tail value. It does not apply C promotions.
func (p *Plan) CheckArguments(args ...llvm.Value) error {
	if p.signature.IsNil() || p.signature.TypeKind() != llvm.FunctionTypeKind {
		return fmt.Errorf("operand checks require a function signature")
	}
	params := p.signature.ParamTypes()
	variadic := p.signature.IsFunctionVarArg()
	if len(args) < len(params) || !variadic && len(args) != len(params) {
		return fmt.Errorf("expected %d arguments (variadic=%v), got %d", len(params), variadic, len(args))
	}
	for i, value := range args {
		if value.IsNil() {
			return fmt.Errorf("argument %d is nil", i+1)
		}
		if i < len(params) && value.Type() != params[i] {
			return fmt.Errorf("argument %d has type %s, expected %s", i+1, value.Type(), params[i])
		}
		if rule := p.rules[i]; rule != nil {
			if err := rule.check(i, value); err != nil {
				return err
			}
		}
		if i >= len(params) && p.tail != nil {
			s := newState(p.signature, WithDataLayout(p.layout))
			s.refs = maps.Clone(p.refs)
			if err := match(p.tail, s, value.Type(), location{path: fmt.Sprintf("argument %d", i+1), arg: i}); err != nil {
				return err
			}
			if err := s.run(); err != nil {
				return err
			}
			if rule := s.plan.rules[i]; rule != nil {
				if err := rule.check(i, value); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type functionType struct {
	result Type
	params []Type
}

func Func(result Type, params ...Type) Type {
	return functionType{result, append([]Type(nil), params...)}
}
func (f functionType) String() string { return fmt.Sprintf("function(%v) -> %v", f.params, f.result) }
func (f functionType) match(s *state, t llvm.Type, l location) error {
	if t.TypeKind() != llvm.FunctionTypeKind {
		return mismatch(l, f, t)
	}
	params := f.params
	var tail *variadicType
	if len(params) > 0 {
		if v, ok := params[len(params)-1].(variadicType); ok {
			tail = &v
			params = params[:len(params)-1]
		}
	}
	if (tail != nil) != t.IsFunctionVarArg() || len(params) != t.ParamTypesCount() {
		return mismatch(l, f, t)
	}
	if err := match(f.result, s, t.ReturnType(), location{path: "result", arg: -1}); err != nil {
		return err
	}
	for i, param := range t.ParamTypes() {
		if err := match(params[i], s, param, location{path: fmt.Sprintf("parameter %d", i+1), arg: i}); err != nil {
			return err
		}
	}
	if tail != nil {
		s.plan.tail = tail.elem
	}
	return nil
}

type variadicType struct{ elem Type }

// VarArgs describes an LLVM ellipsis, not a Go variadic slice. The Go caller
// must recognize __llgo_va_list before constructing the LLVM function type.
func VarArgs(elem Type) Type          { return variadicType{elem} }
func (v variadicType) String() string { return fmt.Sprintf("...%v", v.elem) }
func (v variadicType) match(_ *state, _ llvm.Type, _ location) error {
	return fmt.Errorf("VarArgs must be the last function parameter")
}

type constantType struct{ base Type }

// Const requires an integer or floating-point literal at a direct argument.
// Undef, poison and constant expressions do not satisfy LLVM immarg semantics.
func Const(base Type) Type            { return constantType{base} }
func (c constantType) String() string { return fmt.Sprintf("constant(%v)", c.base) }
func (c constantType) match(s *state, t llvm.Type, l location) error {
	if l.arg < 0 || l.nested {
		return fmt.Errorf("%s: Const requires a direct function parameter", l.path)
	}
	if t.TypeKind() != llvm.IntegerTypeKind && !isFloat(t) {
		return mismatch(l, "scalar integer or floating-point constant", t)
	}
	if err := match(c.base, s, t, l); err != nil {
		return err
	}
	s.plan.rule(l.arg).constant = true
	return nil
}

type rangeType struct {
	base   Type
	bounds interval
}

// Range requires a signed integer literal in the half-open interval. This
// describes an operand constraint, not an LLVM runtime range promise.
func Range(base Type, low, high int64) Type {
	return rangeType{base, interval{uint64(low), uint64(high), false}}
}

// UnsignedRange interprets the literal's signless LLVM bits as unsigned.
func UnsignedRange(base Type, low, high uint64) Type {
	return rangeType{base, interval{low, high, true}}
}
func (r rangeType) String() string { return fmt.Sprintf("%v in %s", r.base, r.bounds) }
func (r rangeType) match(s *state, t llvm.Type, l location) error {
	invalid := r.bounds.low >= r.bounds.high
	if !r.bounds.unsigned {
		invalid = int64(r.bounds.low) >= int64(r.bounds.high)
	}
	if invalid {
		return fmt.Errorf("invalid constant range %s", r.bounds)
	}
	if err := match(Const(r.base), s, t, l); err != nil {
		return err
	}
	if t.TypeKind() != llvm.IntegerTypeKind {
		return mismatch(l, "integer range", t)
	}
	s.plan.rule(l.arg).ranges = append(s.plan.rule(l.arg).ranges, r.bounds)
	return nil
}
