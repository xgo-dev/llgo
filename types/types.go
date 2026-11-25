/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package types provides LLGo's pure-Go semantic, physical layout, and ABI
// type system.
//
// All Type, Object, Package, Scope, Selection, MethodSet, and Info models speak
// LLGo types natively without leaking go/types.Type into downstream compiler
// phases. Standard Go source checking and generic type inference are delegated
// once to go/types at the Context.Check boundary, after which every LLGo
// concept — calling conventions, 2-word Go closures vs 1-word native function
// pointers, pointer kinds (GoPtr/UnsafePtr/CPtr), structs.HostLayout, Wasm32
// wide Go vs native C storage, simd/archsimd vectors, generic instantiations,
// and //llgo:* symbol bindings — is represented as first-class data.
package types

import (
	"fmt"
	gotypes "go/types"
	"strings"
)

// CallConv classifies the calling convention and physical representation of a
// Signature or named function type.
type CallConv uint8

const (
	GoClosure CallConv = iota // Go function value: 2-word {fn, data} closure
	GoDirect                  // Direct Go function symbol / non-closure code pointer (1-word)
	CDecl                     // C cdecl function pointer (1-word; //llgo:cdecl or //llgo:type C)
	Stdcall                   // Windows stdcall function pointer (1-word; //llgo:stdcall or //llgo:type stdcall)
	Fastcall                  // Fastcall function pointer (1-word)
	PyFunc                    // Python C-API bound function (//llgo:link ... py.*)
	WasmFunc                  // WebAssembly host/export function (//go:wasmimport / //go:wasmexport)
)

// IsNative reports whether cc uses a 1-word native (non-Go) calling convention.
func (cc CallConv) IsNative() bool {
	return cc == CDecl || cc == Stdcall || cc == Fastcall || cc == WasmFunc
}

// IsGo reports whether cc is a Go calling convention (closure or direct).
func (cc CallConv) IsGo() bool {
	return cc == GoClosure || cc == GoDirect
}

// IsClosure reports whether cc uses LLGo's 2-word {fn, data} closure layout.
func (cc CallConv) IsClosure() bool {
	return cc == GoClosure
}

func (cc CallConv) String() string {
	switch cc {
	case GoClosure:
		return "closure"
	case GoDirect:
		return "go"
	case CDecl:
		return "cdecl"
	case Stdcall:
		return "stdcall"
	case Fastcall:
		return "fastcall"
	case PyFunc:
		return "py"
	case WasmFunc:
		return "wasm"
	default:
		return fmt.Sprintf("CallConv(%d)", uint8(cc))
	}
}

// PtrKind distinguishes GC-traced Go pointers from unmanaged foreign pointers.
type PtrKind uint8

const (
	GoPtr     PtrKind = iota // GC-traced Go pointer (*T)
	UnsafePtr                // GC-traced opaque pointer (unsafe.Pointer)
	CPtr                     // Unmanaged C / foreign pointer (non-GC-traced)
)

func (k PtrKind) String() string {
	switch k {
	case GoPtr:
		return "go"
	case UnsafePtr:
		return "unsafe"
	case CPtr:
		return "c"
	default:
		return fmt.Sprintf("PtrKind(%d)", uint8(k))
	}
}

// Type is the canonical interface implemented by all LLGo types.
type Type interface {
	Underlying() Type
	HasPtrData() bool
	String() string
	GoType() gotypes.Type
	llTypeMarker()
}

// Unalias resolves any *Alias chain and returns the underlying non-alias Type.
func Unalias(t Type) Type {
	for {
		a, ok := t.(*Alias)
		if !ok || a.rhs == nil || a.rhs == t {
			return t
		}
		t = a.rhs
	}
}

// Basic represents a predeclared scalar, string, or unsafe.Pointer type.
type Basic struct {
	kind   BasicKind
	info   BasicInfo
	name   string
	goType gotypes.Type
}

// Typ holds the predeclared canonical *Basic types, indexed by BasicKind.
var Typ = [...]*Basic{
	Invalid:        {kind: Invalid, name: "invalid type", goType: gotypes.Typ[gotypes.Invalid]},
	Bool:           {kind: Bool, info: IsBoolean, name: "bool", goType: gotypes.Typ[gotypes.Bool]},
	Int:            {kind: Int, info: IsInteger, name: "int", goType: gotypes.Typ[gotypes.Int]},
	Int8:           {kind: Int8, info: IsInteger, name: "int8", goType: gotypes.Typ[gotypes.Int8]},
	Int16:          {kind: Int16, info: IsInteger, name: "int16", goType: gotypes.Typ[gotypes.Int16]},
	Int32:          {kind: Int32, info: IsInteger, name: "int32", goType: gotypes.Typ[gotypes.Int32]},
	Int64:          {kind: Int64, info: IsInteger, name: "int64", goType: gotypes.Typ[gotypes.Int64]},
	Uint:           {kind: Uint, info: IsInteger | IsUnsigned, name: "uint", goType: gotypes.Typ[gotypes.Uint]},
	Uint8:          {kind: Uint8, info: IsInteger | IsUnsigned, name: "uint8", goType: gotypes.Typ[gotypes.Uint8]},
	Uint16:         {kind: Uint16, info: IsInteger | IsUnsigned, name: "uint16", goType: gotypes.Typ[gotypes.Uint16]},
	Uint32:         {kind: Uint32, info: IsInteger | IsUnsigned, name: "uint32", goType: gotypes.Typ[gotypes.Uint32]},
	Uint64:         {kind: Uint64, info: IsInteger | IsUnsigned, name: "uint64", goType: gotypes.Typ[gotypes.Uint64]},
	Uintptr:        {kind: Uintptr, info: IsInteger | IsUnsigned, name: "uintptr", goType: gotypes.Typ[gotypes.Uintptr]},
	Float32:        {kind: Float32, info: IsFloat, name: "float32", goType: gotypes.Typ[gotypes.Float32]},
	Float64:        {kind: Float64, info: IsFloat, name: "float64", goType: gotypes.Typ[gotypes.Float64]},
	Complex64:      {kind: Complex64, info: IsComplex, name: "complex64", goType: gotypes.Typ[gotypes.Complex64]},
	Complex128:     {kind: Complex128, info: IsComplex, name: "complex128", goType: gotypes.Typ[gotypes.Complex128]},
	String:         {kind: String, info: IsString, name: "string", goType: gotypes.Typ[gotypes.String]},
	UnsafePointer:  {kind: UnsafePointer, name: "unsafe.Pointer", goType: gotypes.Typ[gotypes.UnsafePointer]},
	UntypedBool:    {kind: UntypedBool, info: IsBoolean | IsUntyped, name: "untyped bool", goType: gotypes.Typ[gotypes.UntypedBool]},
	UntypedInt:     {kind: UntypedInt, info: IsInteger | IsUntyped, name: "untyped int", goType: gotypes.Typ[gotypes.UntypedInt]},
	UntypedRune:    {kind: UntypedRune, info: IsInteger | IsUntyped, name: "untyped rune", goType: gotypes.Typ[gotypes.UntypedRune]},
	UntypedFloat:   {kind: UntypedFloat, info: IsFloat | IsUntyped, name: "untyped float", goType: gotypes.Typ[gotypes.UntypedFloat]},
	UntypedComplex: {kind: UntypedComplex, info: IsComplex | IsUntyped, name: "untyped complex", goType: gotypes.Typ[gotypes.UntypedComplex]},
	UntypedString:  {kind: UntypedString, info: IsString | IsUntyped, name: "untyped string", goType: gotypes.Typ[gotypes.UntypedString]},
	UntypedNil:     {kind: UntypedNil, info: IsUntyped, name: "untyped nil", goType: gotypes.Typ[gotypes.UntypedNil]},
}

func (b *Basic) Kind() BasicKind      { return b.kind }
func (b *Basic) Info() BasicInfo      { return b.info }
func (b *Basic) Name() string         { return b.name }
func (b *Basic) Underlying() Type     { return b }
func (b *Basic) GoType() gotypes.Type { return b.goType }
func (b *Basic) llTypeMarker()        {}
func (b *Basic) HasPtrData() bool     { return b.kind == String || b.kind == UnsafePointer }
func (b *Basic) String() string       { return b.name }

// Pointer represents a typed pointer with an explicit GC/ownership PtrKind.
type Pointer struct {
	elem   Type
	kind   PtrKind
	goType gotypes.Type
}

// NewPointer returns a GC-traced Go pointer (*elem).
func NewPointer(elem Type) *Pointer {
	return &Pointer{elem: elem, kind: GoPtr}
}

// NewPointerWithKind returns a pointer with an explicit PtrKind.
func NewPointerWithKind(elem Type, kind PtrKind) *Pointer {
	return &Pointer{elem: elem, kind: kind}
}

func (p *Pointer) Elem() Type           { return p.elem }
func (p *Pointer) Kind() PtrKind        { return p.kind }
func (p *Pointer) Underlying() Type     { return p }
func (p *Pointer) GoType() gotypes.Type { return p.goType }
func (p *Pointer) llTypeMarker()        {}
func (p *Pointer) HasPtrData() bool     { return p.kind != CPtr }
func (p *Pointer) String() string {
	elem := "<nil>"
	if p.elem != nil {
		elem = p.elem.String()
	}
	if p.kind == GoPtr {
		return "*" + elem
	}
	return "*" + p.kind.String() + "(" + elem + ")"
}

// Tuple represents an ordered list of variables (signature params/results).
type Tuple struct {
	vars   []*Var
	goType gotypes.Type
}

// NewTuple constructs a Tuple from variables.
func NewTuple(x ...*Var) *Tuple {
	if len(x) == 0 {
		return nil
	}
	cp := make([]*Var, len(x))
	copy(cp, x)
	return &Tuple{vars: cp}
}

func (t *Tuple) Len() int {
	if t == nil {
		return 0
	}
	return len(t.vars)
}

func (t *Tuple) At(i int) *Var        { return t.vars[i] }
func (t *Tuple) Underlying() Type     { return t }
func (t *Tuple) GoType() gotypes.Type { return t.goType }
func (t *Tuple) llTypeMarker()        {}
func (t *Tuple) HasPtrData() bool {
	if t == nil {
		return false
	}
	for _, v := range t.vars {
		if v != nil && v.typ != nil && v.typ.HasPtrData() {
			return true
		}
	}
	return false
}
func (t *Tuple) String() string {
	if t == nil || len(t.vars) == 0 {
		return "()"
	}
	var b strings.Builder
	b.WriteByte('(')
	for i, v := range t.vars {
		if i > 0 {
			b.WriteString(", ")
		}
		if v != nil {
			if v.name != "" {
				b.WriteString(v.name)
				b.WriteByte(' ')
			}
			if v.typ != nil {
				b.WriteString(v.typ.String())
			}
		}
	}
	b.WriteByte(')')
	return b.String()
}

// Signature represents a unified Go closure, direct Go function, or native
// (CDecl / Stdcall / Fastcall / PyFunc / WasmFunc) function type.
//
// Variadic semantics follow CallConv:
//   - GoClosure / GoDirect: standard Go ...T slice variadic
//   - CDecl / Stdcall / Fastcall / WasmFunc: C variadic (...)
//   - PyFunc: Python variadic
type Signature struct {
	callConv       CallConv
	recv           *Var
	recvTypeParams []*TypeParam
	typeParams     []*TypeParam
	params         *Tuple
	results        *Tuple
	variadic       bool
	goType         gotypes.Type
}

// NewSignature constructs a non-generic Signature with the given CallConv.
func NewSignature(recv *Var, params, results *Tuple, variadic bool, cc CallConv) *Signature {
	return &Signature{
		callConv: cc,
		recv:     recv,
		params:   params,
		results:  results,
		variadic: variadic,
	}
}

// NewSignatureType constructs a (possibly generic) Signature.
func NewSignatureType(recv *Var, recvTypeParams, typeParams []*TypeParam, params, results *Tuple, variadic bool, cc CallConv) *Signature {
	return &Signature{
		callConv:       cc,
		recv:           recv,
		recvTypeParams: recvTypeParams,
		typeParams:     typeParams,
		params:         params,
		results:        results,
		variadic:       variadic,
	}
}

func (s *Signature) CallConv() CallConv           { return s.callConv }
func (s *Signature) Recv() *Var                   { return s.recv }
func (s *Signature) RecvTypeParams() []*TypeParam { return s.recvTypeParams }
func (s *Signature) TypeParams() []*TypeParam     { return s.typeParams }
func (s *Signature) Params() *Tuple               { return s.params }
func (s *Signature) Results() *Tuple              { return s.results }
func (s *Signature) Variadic() bool               { return s.variadic }
func (s *Signature) IsClosure() bool              { return s.callConv == GoClosure }
func (s *Signature) IsNative() bool               { return s.callConv.IsNative() }
func (s *Signature) IsGoVariadic() bool           { return s.variadic && s.callConv.IsGo() }
func (s *Signature) IsCVariadic() bool            { return s.variadic && s.callConv.IsNative() }
func (s *Signature) IsPyVariadic() bool           { return s.variadic && s.callConv == PyFunc }
func (s *Signature) Underlying() Type             { return s }
func (s *Signature) GoType() gotypes.Type         { return s.goType }
func (s *Signature) llTypeMarker()                {}
func (s *Signature) HasPtrData() bool             { return s.callConv == GoClosure }

func (s *Signature) String() string {
	var b strings.Builder
	b.WriteString("func")
	if s.callConv != GoClosure {
		b.WriteByte('<')
		b.WriteString(s.callConv.String())
		b.WriteByte('>')
	}
	b.WriteByte('(')
	n := s.params.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		p := s.params.At(i)
		if s.variadic && i == n-1 {
			if sl, ok := p.typ.(*Slice); ok {
				b.WriteString("...")
				b.WriteString(sl.elem.String())
				continue
			}
			b.WriteString("...")
		}
		if p.typ != nil {
			b.WriteString(p.typ.String())
		}
	}
	b.WriteByte(')')
	if rn := s.results.Len(); rn == 1 {
		b.WriteByte(' ')
		b.WriteString(s.results.At(0).typ.String())
	} else if rn > 1 {
		b.WriteByte(' ')
		b.WriteString(s.results.String())
	}
	return b.String()
}

// Slice represents a Go slice type ([]T).
type Slice struct {
	elem   Type
	goType gotypes.Type
}

func NewSlice(elem Type) *Slice { return &Slice{elem: elem} }

func (s *Slice) Elem() Type           { return s.elem }
func (s *Slice) Underlying() Type     { return s }
func (s *Slice) GoType() gotypes.Type { return s.goType }
func (s *Slice) llTypeMarker()        {}
func (s *Slice) HasPtrData() bool     { return true }
func (s *Slice) String() string       { return "[]" + s.elem.String() }

// Array represents a fixed-length array ([N]T). NativeStorage marks arrays
// whose element storage follows C/native rules on targets such as Wasm32.
type Array struct {
	elem          Type
	length        int64
	nativeStorage bool
	goType        gotypes.Type
}

func NewArray(elem Type, length int64) *Array {
	return &Array{elem: elem, length: length}
}

func (a *Array) Elem() Type           { return a.elem }
func (a *Array) Len() int64           { return a.length }
func (a *Array) NativeStorage() bool  { return a.nativeStorage }
func (a *Array) Underlying() Type     { return a }
func (a *Array) GoType() gotypes.Type { return a.goType }
func (a *Array) llTypeMarker()        {}
func (a *Array) HasPtrData() bool     { return a.length > 0 && a.elem != nil && a.elem.HasPtrData() }
func (a *Array) String() string       { return fmt.Sprintf("[%d]%s", a.length, a.elem.String()) }

// Map represents a Go map type (map[K]V).
type Map struct {
	key    Type
	elem   Type
	goType gotypes.Type
}

func NewMap(key, elem Type) *Map { return &Map{key: key, elem: elem} }

func (m *Map) Key() Type            { return m.key }
func (m *Map) Elem() Type           { return m.elem }
func (m *Map) Underlying() Type     { return m }
func (m *Map) GoType() gotypes.Type { return m.goType }
func (m *Map) llTypeMarker()        {}
func (m *Map) HasPtrData() bool     { return true }
func (m *Map) String() string       { return "map[" + m.key.String() + "]" + m.elem.String() }

// Chan represents a Go channel type.
type Chan struct {
	dir    ChanDir
	elem   Type
	goType gotypes.Type
}

func NewChan(dir ChanDir, elem Type) *Chan { return &Chan{dir: dir, elem: elem} }

func (c *Chan) Dir() ChanDir         { return c.dir }
func (c *Chan) Elem() Type           { return c.elem }
func (c *Chan) Underlying() Type     { return c }
func (c *Chan) GoType() gotypes.Type { return c.goType }
func (c *Chan) llTypeMarker()        {}
func (c *Chan) HasPtrData() bool     { return true }
func (c *Chan) String() string {
	switch c.dir {
	case SendOnly:
		return "chan<- " + c.elem.String()
	case RecvOnly:
		return "<-chan " + c.elem.String()
	default:
		return "chan " + c.elem.String()
	}
}

// Interface represents an empty (any) or non-empty Go interface type.
type Interface struct {
	methods    []*Func
	embeddeds  []Type
	comparable bool
	goType     gotypes.Type
}

func NewInterfaceType(methods []*Func, embeddeds []Type) *Interface {
	return &Interface{
		methods:    methods,
		embeddeds:  embeddeds,
		comparable: true,
	}
}

func (t *Interface) NumMethods() int         { return len(t.methods) }
func (t *Interface) Method(i int) *Func      { return t.methods[i] }
func (t *Interface) NumEmbeddeds() int       { return len(t.embeddeds) }
func (t *Interface) EmbeddedType(i int) Type { return t.embeddeds[i] }
func (t *Interface) Empty() bool             { return len(t.methods) == 0 && len(t.embeddeds) == 0 }
func (t *Interface) Comparable() bool        { return t.comparable }
func (t *Interface) Underlying() Type        { return t }
func (t *Interface) GoType() gotypes.Type    { return t.goType }
func (t *Interface) llTypeMarker()           {}
func (t *Interface) HasPtrData() bool        { return true }
func (t *Interface) String() string {
	if t.Empty() {
		return "any"
	}
	var b strings.Builder
	b.WriteString("interface{")
	for i, m := range t.methods {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(m.name)
		if sig := m.Signature(); sig != nil {
			b.WriteString(strings.TrimPrefix(sig.String(), "func"))
		}
	}
	b.WriteByte('}')
	return b.String()
}

// Struct represents a Go or native C struct with first-class LLGo layout flags.
type Struct struct {
	fields          []*Var
	tags            []string
	hostLayout      bool // directly contains a structs.HostLayout marker field
	nativeLayout    bool // explicit native C storage (e.g. //llgo:cdecl / Wasm32 native)
	isClosureStruct bool // legacy synthetic {$f, $data} closure struct
	simdTagIndex    int  // -1 normally; 0=v128, 1=v256, 2=v512 for simd/archsimd tag structs
	goType          gotypes.Type
}

// NewStruct constructs a standard Go Struct.
func NewStruct(fields []*Var, tags []string) *Struct {
	cp := make([]*Var, len(fields))
	copy(cp, fields)
	var tagCopy []string
	if len(tags) > 0 {
		tagCopy = make([]string, len(tags))
		copy(tagCopy, tags)
	}
	return &Struct{
		fields:       cp,
		tags:         tagCopy,
		simdTagIndex: -1,
	}
}

func (s *Struct) NumFields() int   { return len(s.fields) }
func (s *Struct) Field(i int) *Var { return s.fields[i] }
func (s *Struct) Tag(i int) string {
	if i < len(s.tags) {
		return s.tags[i]
	}
	return ""
}
func (s *Struct) HostLayout() bool       { return s.hostLayout }
func (s *Struct) NativeLayout() bool     { return s.nativeLayout }
func (s *Struct) IsClosureStruct() bool  { return s.isClosureStruct }
func (s *Struct) SIMDTagIndex() int      { return s.simdTagIndex }
func (s *Struct) SetHostLayout(v bool)   { s.hostLayout = v }
func (s *Struct) SetNativeLayout(v bool) { s.nativeLayout = v }
func (s *Struct) Underlying() Type       { return s }
func (s *Struct) GoType() gotypes.Type   { return s.goType }
func (s *Struct) llTypeMarker()          {}
func (s *Struct) HasPtrData() bool {
	for _, f := range s.fields {
		if f != nil && f.typ != nil && f.typ.HasPtrData() {
			return true
		}
	}
	return false
}
func (s *Struct) String() string {
	if len(s.fields) == 0 {
		return "struct{}"
	}
	var b strings.Builder
	b.WriteString("struct{")
	for i, f := range s.fields {
		if i > 0 {
			b.WriteString("; ")
		}
		if f.name != "" {
			b.WriteString(f.name)
			b.WriteByte(' ')
		}
		if f.typ != nil {
			b.WriteString(f.typ.String())
		}
	}
	b.WriteByte('}')
	return b.String()
}

// SIMDVec represents a fixed-width SIMD vector or mask value (simd/archsimd).
type SIMDVec struct {
	elem     *Basic
	length   int
	isMask   bool
	tagIndex int // 0=128-bit, 1=256-bit, 2=512-bit
	goType   gotypes.Type
}

// NewSIMDVec constructs a canonical SIMD vector or mask type.
func NewSIMDVec(elem *Basic, length int, isMask bool) *SIMDVec {
	tagIdx := simdTagForBits(basicBits(elem) * length)
	return &SIMDVec{
		elem:     elem,
		length:   length,
		isMask:   isMask,
		tagIndex: tagIdx,
	}
}

func basicBits(b *Basic) int {
	if b == nil {
		return 0
	}
	switch b.kind {
	case Int8, Uint8:
		return 8
	case Int16, Uint16:
		return 16
	case Int32, Uint32, Float32:
		return 32
	case Int64, Uint64, Float64:
		return 64
	default:
		return 0
	}
}

func simdTagForBits(bits int) int {
	switch bits {
	case 128:
		return 0
	case 256:
		return 1
	case 512:
		return 2
	default:
		return 0
	}
}

func (v *SIMDVec) Elem() *Basic         { return v.elem }
func (v *SIMDVec) Len() int             { return v.length }
func (v *SIMDVec) IsMask() bool         { return v.isMask }
func (v *SIMDVec) TagIndex() int        { return v.tagIndex }
func (v *SIMDVec) BitWidth() int        { return basicBits(v.elem) * v.length }
func (v *SIMDVec) Underlying() Type     { return v }
func (v *SIMDVec) GoType() gotypes.Type { return v.goType }
func (v *SIMDVec) llTypeMarker()        {}
func (v *SIMDVec) HasPtrData() bool     { return false }
func (v *SIMDVec) String() string {
	prefix := "vec"
	if v.isMask {
		prefix = "mask"
	}
	return fmt.Sprintf("%s<%dx%s>", prefix, v.length, v.elem.String())
}

// Named represents a defined (named) type, including instantiated generic types
// (mirroring go/types.Named with Origin, TypeParams, and TypeArgs) plus
// first-class LLGo directive/layout attributes.
type Named struct {
	obj          *TypeName
	origin       *Named
	typeParams   []*TypeParam
	typeArgs     []Type
	underlying   Type
	methods      []*Func
	callConv     CallConv
	hasCallConv  bool
	hostLayout   bool
	nativeLayout bool
	simdTagIndex int // -1 normally; 0=v128, 1=v256, 2=v512
	simdVec      *SIMDVec
	goType       gotypes.Type
	goUnderlying gotypes.Type
}

// NewNamed constructs a defined type bound to obj.
func NewNamed(obj *TypeName, underlying Type, methods []*Func) *Named {
	n := &Named{
		obj:          obj,
		underlying:   underlying,
		methods:      methods,
		simdTagIndex: -1,
	}
	if obj != nil {
		obj.typ = n
	}
	return n
}

func (n *Named) Obj() *TypeName { return n.obj }
func (n *Named) Origin() *Named {
	if n.origin != nil {
		return n.origin
	}
	return n
}
func (n *Named) TypeParams() []*TypeParam  { return n.typeParams }
func (n *Named) TypeArgs() []Type          { return n.typeArgs }
func (n *Named) NumMethods() int           { return len(n.methods) }
func (n *Named) Method(i int) *Func        { return n.methods[i] }
func (n *Named) AddMethod(m *Func)         { n.methods = append(n.methods, m) }
func (n *Named) CallConv() CallConv        { return n.callConv }
func (n *Named) HasExplicitCallConv() bool { return n.hasCallConv }
func (n *Named) HostLayout() bool          { return n.hostLayout }
func (n *Named) NativeLayout() bool        { return n.nativeLayout }
func (n *Named) SIMDTagIndex() int         { return n.simdTagIndex }
func (n *Named) SIMDVec() *SIMDVec         { return n.simdVec }
func (n *Named) GoType() gotypes.Type      { return n.goType }
func (n *Named) llTypeMarker()             {}

func (n *Named) SetUnderlying(u Type) {
	n.underlying = u
}

func (n *Named) Underlying() Type {
	if n.underlying == nil {
		return n
	}
	return n.underlying.Underlying()
}

func (n *Named) HasPtrData() bool {
	if n.underlying == nil {
		return false
	}
	return n.underlying.HasPtrData()
}

func (n *Named) String() string {
	var b strings.Builder
	if n.obj != nil {
		if n.obj.pkg != nil && n.obj.pkg.path != "" {
			b.WriteString(n.obj.pkg.path)
			b.WriteByte('.')
		}
		b.WriteString(n.obj.name)
	} else {
		b.WriteString("<named>")
	}
	if len(n.typeArgs) > 0 {
		b.WriteByte('[')
		for i, arg := range n.typeArgs {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(arg.String())
		}
		b.WriteByte(']')
	}
	return b.String()
}

// Alias represents a type alias (type A = Rhs), preserving any LLGo calling
// convention directive attached directly to the alias declaration (e.g.
// //llgo:cdecl type Cb = func()).
type Alias struct {
	obj         *TypeName
	rhs         Type
	origin      *Alias
	typeParams  []*TypeParam
	typeArgs    []Type
	callConv    CallConv
	hasCallConv bool
	goType      gotypes.Type
}

// NewAlias constructs a type alias.
func NewAlias(obj *TypeName, rhs Type) *Alias {
	a := &Alias{obj: obj, rhs: rhs}
	if obj != nil {
		obj.typ = a
		obj.isAlias = true
	}
	return a
}

func (a *Alias) Obj() *TypeName { return a.obj }
func (a *Alias) Rhs() Type      { return a.rhs }
func (a *Alias) Origin() *Alias {
	if a.origin != nil {
		return a.origin
	}
	return a
}
func (a *Alias) TypeParams() []*TypeParam  { return a.typeParams }
func (a *Alias) TypeArgs() []Type          { return a.typeArgs }
func (a *Alias) CallConv() CallConv        { return a.callConv }
func (a *Alias) HasExplicitCallConv() bool { return a.hasCallConv }
func (a *Alias) GoType() gotypes.Type      { return a.goType }
func (a *Alias) llTypeMarker()             {}

func (a *Alias) Underlying() Type {
	if a.rhs == nil {
		return a
	}
	return a.rhs.Underlying()
}

func (a *Alias) HasPtrData() bool {
	if a.rhs == nil {
		return false
	}
	return a.rhs.HasPtrData()
}

func (a *Alias) String() string {
	if a.obj != nil {
		if a.obj.pkg != nil && a.obj.pkg.path != "" {
			return a.obj.pkg.path + "." + a.obj.name
		}
		return a.obj.name
	}
	if a.rhs != nil {
		return a.rhs.String()
	}
	return "<alias>"
}

// TypeParam represents a generic type parameter.
type TypeParam struct {
	obj        *TypeName
	index      int
	constraint Type
	goType     gotypes.Type
}

// NewTypeParam constructs a generic type parameter.
func NewTypeParam(obj *TypeName, constraint Type) *TypeParam {
	tp := &TypeParam{obj: obj, constraint: constraint}
	if obj != nil {
		obj.typ = tp
	}
	return tp
}

func (tp *TypeParam) Obj() *TypeName   { return tp.obj }
func (tp *TypeParam) Index() int       { return tp.index }
func (tp *TypeParam) Constraint() Type { return tp.constraint }
func (tp *TypeParam) Underlying() Type {
	if tp.constraint != nil {
		return tp.constraint.Underlying()
	}
	return tp
}
func (tp *TypeParam) HasPtrData() bool     { return true }
func (tp *TypeParam) GoType() gotypes.Type { return tp.goType }
func (tp *TypeParam) llTypeMarker()        {}
func (tp *TypeParam) String() string {
	if tp.obj != nil {
		return tp.obj.name
	}
	return fmt.Sprintf("T%d", tp.index)
}

// Identical reports whether x and y are identical LLGo types, respecting
// LLGo-specific attributes such as CallConv, PtrKind, HostLayout, NativeLayout,
// and SIMDVec shapes.
func Identical(x, y Type) bool {
	x = Unalias(x)
	y = Unalias(y)
	if x == y {
		return true
	}
	if x == nil || y == nil {
		return false
	}
	switch x := x.(type) {
	case *Basic:
		y, ok := y.(*Basic)
		return ok && x.kind == y.kind
	case *Pointer:
		y, ok := y.(*Pointer)
		return ok && x.kind == y.kind && Identical(x.elem, y.elem)
	case *Slice:
		y, ok := y.(*Slice)
		return ok && Identical(x.elem, y.elem)
	case *Array:
		y, ok := y.(*Array)
		return ok && x.length == y.length && x.nativeStorage == y.nativeStorage && Identical(x.elem, y.elem)
	case *Map:
		y, ok := y.(*Map)
		return ok && Identical(x.key, y.key) && Identical(x.elem, y.elem)
	case *Chan:
		y, ok := y.(*Chan)
		return ok && x.dir == y.dir && Identical(x.elem, y.elem)
	case *Tuple:
		y, ok := y.(*Tuple)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for i := 0; i < x.Len(); i++ {
			if !Identical(x.At(i).typ, y.At(i).typ) {
				return false
			}
		}
		return true
	case *Signature:
		y, ok := y.(*Signature)
		if !ok || x.callConv != y.callConv || x.variadic != y.variadic {
			return false
		}
		return Identical(x.params, y.params) && Identical(x.results, y.results)
	case *Struct:
		y, ok := y.(*Struct)
		if !ok || len(x.fields) != len(y.fields) ||
			x.hostLayout != y.hostLayout || x.nativeLayout != y.nativeLayout ||
			x.isClosureStruct != y.isClosureStruct || x.simdTagIndex != y.simdTagIndex {
			return false
		}
		for i, xf := range x.fields {
			yf := y.fields[i]
			if xf.name != yf.name || xf.embedded != yf.embedded || x.Tag(i) != y.Tag(i) || !Identical(xf.typ, yf.typ) {
				return false
			}
		}
		return true
	case *Interface:
		y, ok := y.(*Interface)
		if !ok || len(x.methods) != len(y.methods) || len(x.embeddeds) != len(y.embeddeds) {
			return false
		}
		for i, xm := range x.methods {
			ym := y.methods[i]
			if xm.name != ym.name || !Identical(xm.typ, ym.typ) {
				return false
			}
		}
		return true
	case *SIMDVec:
		y, ok := y.(*SIMDVec)
		return ok && x.length == y.length && x.isMask == y.isMask && Identical(x.elem, y.elem)
	case *Named:
		y, ok := y.(*Named)
		if !ok {
			return false
		}
		if x.obj == y.obj && len(x.typeArgs) == len(y.typeArgs) {
			for i := range x.typeArgs {
				if !Identical(x.typeArgs[i], y.typeArgs[i]) {
					return false
				}
			}
			return true
		}
		return false
	case *TypeParam:
		y, ok := y.(*TypeParam)
		return ok && x.obj == y.obj && x.index == y.index
	}
	return false
}
