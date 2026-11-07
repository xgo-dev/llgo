package llvmintrinsic

import (
	"fmt"
	"math"
	"strings"

	"github.com/xgo-dev/llvm"
)

// Type is a reusable LLVM type pattern. Nil means to leave that part to LLVM.
// It is a compiler description, not a Go parameter type or a conversion rule.
type Type interface {
	match(*state, llvm.Type, location) error
	fmt.Stringer
}

type location struct {
	path   string
	arg    int
	nested bool
}

func (l location) child(name string) location {
	l.path += name
	l.nested = true
	return l
}

func mismatch(l location, want any, got llvm.Type) error {
	return fmt.Errorf("%s: expected %v, got %s", l.path, want, got)
}

func match(t Type, s *state, actual llvm.Type, l location) error {
	if actual.IsNil() {
		return fmt.Errorf("%s: nil LLVM type", l.path)
	}
	if t == nil {
		return nil
	}
	return t.match(s, actual, l)
}

type domain string

var (
	Any        Type = domain("any")
	Integer    Type = domain("scalar integer")
	Integers   Type = domain("integer or integer vector")
	Floating   Type = domain("scalar floating-point")
	Floats     Type = domain("floating-point or floating-point vector")
	AnyVector  Type = domain("vector")
	AnyPointer Type = domain("pointer")
	Void       Type = kindType(llvm.VoidTypeKind)
	I1              = Int(1)
	I8              = Int(8)
	I16             = Int(16)
	I32             = Int(32)
	I64             = Int(64)
	F32        Type = kindType(llvm.FloatTypeKind)
	F64        Type = kindType(llvm.DoubleTypeKind)
)

func (d domain) String() string { return string(d) }
func (d domain) match(_ *state, t llvm.Type, l location) error {
	scalar := t
	_, _, vector := vectorShape(t)
	if vector {
		scalar = t.ElementType()
	}
	ok := d == "any"
	switch d {
	case "scalar integer":
		ok = !vector && scalar.TypeKind() == llvm.IntegerTypeKind
	case "integer or integer vector":
		ok = scalar.TypeKind() == llvm.IntegerTypeKind
	case "scalar floating-point":
		ok = !vector && isFloat(scalar)
	case "floating-point or floating-point vector":
		ok = isFloat(scalar)
	case "vector":
		ok = vector
	case "pointer":
		ok = t.TypeKind() == llvm.PointerTypeKind
	}
	if !ok {
		return mismatch(l, d, t)
	}
	return nil
}

func isFloat(t llvm.Type) bool {
	switch t.TypeKind() {
	case halfKind, bfloatKind, llvm.FloatTypeKind, llvm.DoubleTypeKind,
		llvm.X86_FP80TypeKind, llvm.FP128TypeKind, llvm.PPC_FP128TypeKind:
		return true
	}
	return false
}

type kindType llvm.TypeKind

func (k kindType) String() string { return fmt.Sprint(llvm.TypeKind(k)) }
func (k kindType) match(_ *state, t llvm.Type, l location) error {
	if t.TypeKind() != llvm.TypeKind(k) {
		return mismatch(l, k, t)
	}
	return nil
}

type intType uint

// Int matches an exact, signless LLVM integer width. Named Go integer types
// and signedness are handled by the caller's Go-to-LLVM mapping.
func Int(bits uint) Type         { return intType(bits) }
func (i intType) String() string { return fmt.Sprintf("i%d", i) }
func (i intType) match(_ *state, t llvm.Type, l location) error {
	if t.TypeKind() != llvm.IntegerTypeKind || uint(t.IntTypeWidth()) != uint(i) {
		return mismatch(l, i, t)
	}
	return nil
}

type exactType struct{ t llvm.Type }

// Exact uses LLVM type identity, including address spaces, packing and float
// format. Equal storage sizes are not sufficient.
func Exact(t llvm.Type) Type { return exactType{t} }
func (e exactType) String() string {
	if e.t.IsNil() {
		return "<nil>"
	}
	return e.t.String()
}
func (e exactType) match(_ *state, t llvm.Type, l location) error {
	if e.t != t {
		return mismatch(l, e, t)
	}
	return nil
}
func (e exactType) resolve(*state) (llvm.Type, error) { return e.t, nil }

type pointerType uint32

// Ptr matches an opaque pointer in an exact address space. LLVM carries no
// pointee type; optional Go pointee restrictions belong before type erasure.
func Ptr(addressSpace uint32) Type   { return pointerType(addressSpace) }
func (p pointerType) String() string { return fmt.Sprintf("ptr addrspace(%d)", p) }
func (p pointerType) match(_ *state, t llvm.Type, l location) error {
	if t.TypeKind() != llvm.PointerTypeKind || uint32(t.PointerAddressSpace()) != uint32(p) {
		return mismatch(l, p, t)
	}
	return nil
}

type arrayType struct {
	n    uint64
	elem Type
}

func Array(n uint64, elem Type) Type { return arrayType{n, elem} }
func (a arrayType) String() string   { return fmt.Sprintf("[%d x %v]", a.n, a.elem) }
func (a arrayType) match(s *state, t llvm.Type, l location) error {
	if t.TypeKind() != llvm.ArrayTypeKind || arrayLength(t) != a.n {
		return mismatch(l, a, t)
	}
	return match(a.elem, s, t.ElementType(), l.child(" element"))
}

type structType struct {
	fields []Type
	packed bool
}

func Struct(fields ...Type) Type       { return structType{append([]Type(nil), fields...), false} }
func PackedStruct(fields ...Type) Type { return structType{append([]Type(nil), fields...), true} }
func (st structType) String() string {
	return fmt.Sprintf("struct(packed=%v, %v)", st.packed, st.fields)
}
func (st structType) match(s *state, t llvm.Type, l location) error {
	if t.TypeKind() != llvm.StructTypeKind || t.IsStructOpaque() || t.IsStructPacked() != st.packed {
		return mismatch(l, st, t)
	}
	fields := t.StructElementTypes()
	if len(fields) != len(st.fields) {
		return mismatch(l, st, t)
	}
	for i, f := range fields {
		if err := match(st.fields[i], s, f, l.child(fmt.Sprintf(" field %d", i))); err != nil {
			return err
		}
	}
	return nil
}

// Count describes a vector's minimum lane count and its scalable flag.
type Count interface {
	eval(*state) (shape, error)
	fmt.Stringer
}
type shape struct {
	n        uint64
	scalable bool
}

func Fixed(n uint64) Count    { return shape{n, false} }
func Scalable(n uint64) Count { return shape{n, true} }
func (n shape) eval(*state) (shape, error) {
	if n.n == 0 || n.n > math.MaxUint32 {
		return shape{}, fmt.Errorf("invalid vector lane count %d", n.n)
	}
	return n, nil
}
func (n shape) String() string { return fmt.Sprintf("%d lanes (scalable=%v)", n.n, n.scalable) }

type vectorPattern struct {
	n    Count
	elem Type
}

func Vector(n Count, elem Type) Type   { return vectorPattern{n, elem} }
func (v vectorPattern) String() string { return fmt.Sprintf("vector(%v, %v)", v.n, v.elem) }
func (v vectorPattern) match(s *state, t llvm.Type, l location) error {
	n, scalable, ok := vectorShape(t)
	if !ok || v.n == nil {
		return mismatch(l, v, t)
	}
	if err := match(v.elem, s, t.ElementType(), l.child(" element")); err != nil {
		return err
	}
	s.deferCheck(func() error {
		want, err := v.n.eval(s)
		if err != nil {
			return err
		}
		if want != (shape{n, scalable}) {
			return mismatch(l, v, t)
		}
		return nil
	})
	return nil
}

type reference string

// Result and Arg refer to the complete actual signature and need no Bind.
var Result Type = reference("$result")

func Arg(index int) Type           { return reference(fmt.Sprintf("$arg%d", index)) }
func Ref(name string) Type         { return reference(name) }
func (r reference) String() string { return string(r) }
func (r reference) resolve(s *state) (llvm.Type, error) {
	if t, ok := s.refs[string(r)]; ok {
		return t, nil
	}
	return llvm.Type{}, unboundReference(r)
}
func (r reference) match(s *state, t llvm.Type, l location) error {
	s.deferCheck(func() error {
		want, err := r.resolve(s)
		if err != nil {
			return err
		}
		if want != t {
			return mismatch(l, r, t)
		}
		return nil
	})
	return nil
}

type boundType struct {
	name string
	base Type
}

func Bind(name string, base Type) Type { return boundType{name, base} }
func (b boundType) String() string     { return fmt.Sprintf("%s:%v", b.name, b.base) }
func (b boundType) match(s *state, t llvm.Type, l location) error {
	if b.name == "" || strings.HasPrefix(b.name, "$") {
		return fmt.Errorf("invalid type binding %q", b.name)
	}
	if old, ok := s.refs[b.name]; ok && old != t {
		return mismatch(l, b, t)
	}
	s.refs[b.name] = t
	return match(b.base, s, t, l)
}

func resolve(t Type, s *state) (llvm.Type, error) {
	if r, ok := t.(interface {
		resolve(*state) (llvm.Type, error)
	}); ok {
		typ, err := r.resolve(s)
		if err == nil && typ.IsNil() {
			err = fmt.Errorf("nil LLVM type reference")
		}
		return typ, err
	}
	return llvm.Type{}, fmt.Errorf("%v is not a concrete type reference", t)
}

type elementType struct{ source Type }

func Elem(source Type) Type          { return elementType{source} }
func (e elementType) String() string { return fmt.Sprintf("element(%v)", e.source) }
func (e elementType) resolve(s *state) (llvm.Type, error) {
	t, err := resolve(e.source, s)
	if err != nil {
		return llvm.Type{}, err
	}
	_, _, vector := vectorShape(t)
	if !vector && t.TypeKind() != llvm.ArrayTypeKind {
		return llvm.Type{}, fmt.Errorf("%s has no array/vector element type", t)
	}
	return t.ElementType(), nil
}
func (e elementType) match(s *state, t llvm.Type, l location) error {
	s.deferCheck(func() error {
		want, err := e.resolve(s)
		if err != nil {
			return err
		}
		if want != t {
			return mismatch(l, e, t)
		}
		return nil
	})
	return nil
}

type laneCount struct{ source Type }

func Lanes(source Type) Count      { return laneCount{source} }
func (l laneCount) String() string { return fmt.Sprintf("lanes(%v)", l.source) }
func (l laneCount) eval(s *state) (shape, error) {
	t, err := resolve(l.source, s)
	if err != nil {
		return shape{}, err
	}
	n, scale, ok := vectorShape(t)
	if !ok {
		return shape{}, fmt.Errorf("%s is not a vector", t)
	}
	return shape{n, scale}, nil
}

type dividedCount struct {
	source  Count
	divisor uint64
}

// Divide requires exact division, including for scalable minimum lane counts.
func Divide(source Count, divisor uint64) Count { return dividedCount{source, divisor} }
func (d dividedCount) String() string           { return fmt.Sprintf("%v/%d", d.source, d.divisor) }
func (d dividedCount) eval(s *state) (shape, error) {
	if d.source == nil || d.divisor == 0 {
		return shape{}, fmt.Errorf("invalid lane divisor %d", d.divisor)
	}
	n, err := d.source.eval(s)
	if err != nil {
		return shape{}, err
	}
	if n.n%d.divisor != 0 {
		return shape{}, fmt.Errorf("%d lanes are not divisible by %d", n.n, d.divisor)
	}
	n.n /= d.divisor
	return n.eval(s)
}

type shapedType struct{ source, elem Type }

// Shape replaces the element type, preserving scalar/vector shape exactly.
func Shape(source, elem Type) Type   { return shapedType{source, elem} }
func (sh shapedType) String() string { return fmt.Sprintf("shape(%v, %v)", sh.source, sh.elem) }
func (sh shapedType) match(s *state, t llvm.Type, l location) error {
	s.deferCheck(func() error {
		source, err := resolve(sh.source, s)
		if err != nil {
			return err
		}
		n, scale, vec := vectorShape(source)
		tn, ts, tv := vectorShape(t)
		if vec != tv || vec && (n != tn || scale != ts) {
			return mismatch(l, sh, t)
		}
		if tv {
			t = t.ElementType()
		}
		return match(sh.elem, s, t, l.child(" element"))
	})
	return nil
}

// Context supplies complete signature types to custom type predicates.
type Context struct{ s *state }

func (c Context) Resolve(pattern Type) (llvm.Type, error) { return resolve(pattern, c.s) }
func (c Context) DataLayout() string                      { return c.s.layout }

type predicateType struct {
	base        Type
	description string
	check       func(Context, llvm.Type) error
}

func Where(base Type, description string, check func(Context, llvm.Type) error) Type {
	return predicateType{base, description, check}
}
func (p predicateType) String() string { return p.description }
func (p predicateType) match(s *state, t llvm.Type, l location) error {
	if p.check == nil {
		return fmt.Errorf("nil type predicate %q", p.description)
	}
	if err := match(p.base, s, t, l); err != nil {
		return err
	}
	s.deferCheck(func() error {
		if err := p.check(Context{s}, t); err != nil {
			return fmt.Errorf("%s: %s: %w", l.path, p.description, err)
		}
		return nil
	})
	return nil
}

// Bits returns an integer/floating scalar width, including for vector elements.
func Bits(t llvm.Type) (uint64, error) {
	if t.IsNil() {
		return 0, fmt.Errorf("nil type")
	}
	scalar := t
	if _, _, vector := vectorShape(t); vector {
		scalar = t.ElementType()
	}
	if scalar.TypeKind() != llvm.IntegerTypeKind && !isFloat(scalar) {
		return 0, fmt.Errorf("%s has no integer/floating bit width", t)
	}
	return scalarBits(t), nil
}
