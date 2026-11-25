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
// type system, designed for source-level familiarity with go/types while
// natively expressing LLGo's extensions.
//
// All Type, Object, Package, Scope, Selection, MethodSet, and Info models speak
// LLGo types natively without leaking go/types.Type into downstream compiler
// phases. Standard Go source checking and generic type inference are delegated
// once to go/types at the Context.Check boundary, after which every LLGo
// concept — calling conventions (CallConv), pointer kinds (GoPtr/UnsafePtr/CPtr),
// structs.HostLayout, Wasm32 wide Go vs native C storage, simd/archsimd
// vectors, generic instantiations, Func.FreeVars, and //llgo:* symbol bindings —
// is represented as first-class data.
package types

import (
	"fmt"
	gotypes "go/types"
	"strings"
)

// CallConv classifies the calling convention, value representation, and
// variadic ABI of a Signature or named function type (analogous to Swift's
// @convention(...)).
//
// The zero value (Go) matches standard Go / go/types.Signature semantics: a
// 2-word Go-GC closure {codePtr, envPtr} with Go slice variadics.
type CallConv uint8

const (
	Go       CallConv = iota // Standard Go ABI: 2-word Go-GC closure {fn, env}, Go slice variadic
	CDecl                    // C cdecl function pointer: 1-word, non-GC, C va_list variadic (//llgo:cdecl / //llgo:type C)
	Stdcall                  // Windows stdcall function pointer: 1-word, non-GC (//llgo:stdcall / //llgo:type stdcall)
	Fastcall                 // Fastcall function pointer: 1-word, non-GC
	Py                       // Python C-API bound function (//llgo:link ... py.*)
	Wasm                     // WebAssembly host/export function (//go:wasmimport / //go:wasmexport)
)

// IsGo reports whether cc is the standard Go calling convention.
func (cc CallConv) IsGo() bool {
	return cc == Go
}

// IsNative reports whether cc uses a 1-word native (non-Go) calling convention.
func (cc CallConv) IsNative() bool {
	return cc == CDecl || cc == Stdcall || cc == Fastcall || cc == Wasm
}

// IsClosure reports whether values with calling convention cc carry a closure
// environment representation (2-word {fn, env} for Go).
func (cc CallConv) IsClosure() bool {
	return cc == Go
}

func (cc CallConv) String() string {
	switch cc {
	case Go:
		return "go"
	case CDecl:
		return "cdecl"
	case Stdcall:
		return "stdcall"
	case Fastcall:
		return "fastcall"
	case Py:
		return "py"
	case Wasm:
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
	IsHostLayoutMarker() bool
	HasHostLayout() bool
	ContainsHostLayout() bool
	ContainsSIMDStorage() bool
	SIMDTagIndex() int
	SIMDVec() *SIMDVec
	String() string
	goTypeRef() gotypes.Type
	llTypeMarker()
}

func isStructsPkgPath(path string) bool {
	return path == "structs" || path == "github.com/xgo-dev/llgo/runtime/internal/structs"
}

// Qualifier controls how named package-level objects are printed in TypeString
// and ObjectString (mirroring go/types.Qualifier).
type Qualifier func(*Package) string

// RelativeTo returns a Qualifier that fully qualifies members of all packages
// other than pkg (mirroring go/types.RelativeTo).
func RelativeTo(pkg *Package) Qualifier {
	if pkg == nil {
		return nil
	}
	return func(other *Package) string {
		if pkg == other {
			return ""
		}
		return other.Path()
	}
}

// TypeString returns the string representation of typ qualified by qf.
func TypeString(typ Type, qf Qualifier) string {
	if typ == nil {
		return "<nil>"
	}
	return writeTypeString(typ, qf)
}

func writeTypeString(typ Type, qf Qualifier) string {
	if typ == nil {
		return "<nil>"
	}
	switch t := typ.(type) {
	case *Named:
		var b strings.Builder
		if t.obj != nil {
			writeQualPkg(&b, t.obj.pkg, qf)
			b.WriteString(t.obj.name)
		} else {
			b.WriteString("<named>")
		}
		if args := t.TypeArgs(); args != nil && args.Len() > 0 {
			b.WriteByte('[')
			for i := 0; i < args.Len(); i++ {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(writeTypeString(args.At(i), qf))
			}
			b.WriteByte(']')
		}
		return b.String()
	case *Alias:
		if t.obj != nil {
			var b strings.Builder
			writeQualPkg(&b, t.obj.pkg, qf)
			b.WriteString(t.obj.name)
			return b.String()
		}
		return writeTypeString(t.rhs, qf)
	case *Pointer:
		if t.kind == GoPtr {
			return "*" + writeTypeString(t.elem, qf)
		}
		return "*" + t.kind.String() + "(" + writeTypeString(t.elem, qf) + ")"
	case *Slice:
		return "[]" + writeTypeString(t.elem, qf)
	case *Array:
		return fmt.Sprintf("[%d]%s", t.length, writeTypeString(t.elem, qf))
	case *Map:
		return "map[" + writeTypeString(t.key, qf) + "]" + writeTypeString(t.elem, qf)
	case *Chan:
		switch t.dir {
		case SendOnly:
			return "chan<- " + writeTypeString(t.elem, qf)
		case RecvOnly:
			return "<-chan " + writeTypeString(t.elem, qf)
		default:
			return "chan " + writeTypeString(t.elem, qf)
		}
	default:
		return typ.String()
	}
}

func writeQualPkg(b *strings.Builder, pkg *Package, qf Qualifier) {
	if pkg == nil {
		return
	}
	var s string
	if qf != nil {
		s = qf(pkg)
	} else {
		s = pkg.path
	}
	if s != "" {
		b.WriteString(s)
		b.WriteByte('.')
	}
}

// TypeList holds an immutable list of Types (mirroring go/types.TypeList).
type TypeList struct {
	types []Type
}

func newTypeList(list []Type) *TypeList {
	if len(list) == 0 {
		return nil
	}
	cp := make([]Type, len(list))
	copy(cp, list)
	return &TypeList{types: cp}
}

func (l *TypeList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.types)
}

func (l *TypeList) At(i int) Type { return l.types[i] }

// TypeParamList holds an immutable list of TypeParams (mirroring go/types.TypeParamList).
type TypeParamList struct {
	tparams []*TypeParam
}

func newTypeParamList(list []*TypeParam) *TypeParamList {
	if len(list) == 0 {
		return nil
	}
	cp := make([]*TypeParam, len(list))
	copy(cp, list)
	return &TypeParamList{tparams: cp}
}

func (l *TypeParamList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.tparams)
}

func (l *TypeParamList) At(i int) *TypeParam { return l.tparams[i] }

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

func (b *Basic) Kind() BasicKind         { return b.kind }
func (b *Basic) Info() BasicInfo         { return b.info }
func (b *Basic) Name() string            { return b.name }
func (b *Basic) Underlying() Type        { return b }
func (b *Basic) goTypeRef() gotypes.Type { return b.goType }
func (b *Basic) llTypeMarker()           {}
func (b *Basic) HasPtrData() bool        { return b.kind == String || b.kind == UnsafePointer }
func (b *Basic) IsHostLayoutMarker() bool { return false }
func (b *Basic) HasHostLayout() bool      { return false }
func (b *Basic) ContainsHostLayout() bool { return false }
func (b *Basic) ContainsSIMDStorage() bool { return false }
func (b *Basic) SIMDTagIndex() int        { return -1 }
func (b *Basic) SIMDVec() *SIMDVec        { return nil }
func (b *Basic) String() string          { return b.name }

// Pointer represents a typed pointer with an explicit GC/ownership PtrKind.
type Pointer struct {
	elem   Type
	kind   PtrKind
	goType gotypes.Type
}

// NewPointer returns a GC-traced Go pointer (*elem), matching go/types.NewPointer.
func NewPointer(elem Type) *Pointer {
	return &Pointer{elem: elem, kind: GoPtr}
}

// NewPointerWithKind returns a pointer with an explicit PtrKind.
func NewPointerWithKind(elem Type, kind PtrKind) *Pointer {
	return &Pointer{elem: elem, kind: kind}
}

func (p *Pointer) Elem() Type              { return p.elem }
func (p *Pointer) Kind() PtrKind           { return p.kind }
func (p *Pointer) Underlying() Type        { return p }
func (p *Pointer) goTypeRef() gotypes.Type { return p.goType }
func (p *Pointer) llTypeMarker()           {}
func (p *Pointer) HasPtrData() bool        { return p.kind != CPtr }
func (p *Pointer) IsHostLayoutMarker() bool { return false }
func (p *Pointer) HasHostLayout() bool      { return false }
func (p *Pointer) ContainsHostLayout() bool { return false }
func (p *Pointer) ContainsSIMDStorage() bool { return false }
func (p *Pointer) SIMDTagIndex() int        { return -1 }
func (p *Pointer) SIMDVec() *SIMDVec        { return nil }
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

// NewTuple constructs a Tuple from variables (matching go/types.NewTuple).
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

func (t *Tuple) At(i int) *Var           { return t.vars[i] }
func (t *Tuple) Underlying() Type        { return t }
func (t *Tuple) goTypeRef() gotypes.Type { return t.goType }
func (t *Tuple) llTypeMarker()           {}
func (t *Tuple) IsHostLayoutMarker() bool { return false }
func (t *Tuple) HasHostLayout() bool      { return false }
func (t *Tuple) ContainsHostLayout() bool { return false }
func (t *Tuple) ContainsSIMDStorage() bool { return false }
func (t *Tuple) SIMDTagIndex() int        { return -1 }
func (t *Tuple) SIMDVec() *SIMDVec        { return nil }
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

// Signature represents a Go or foreign function type, mirroring
// go/types.Signature with a single CallConv extension.
//
// Variadic semantics follow CallConv:
//   - Go: standard Go ...T slice variadic
//   - CDecl / Stdcall / Fastcall / Wasm: C variadic (...)
//   - Py: Python variadic
type Signature struct {
	callConv       CallConv
	recv           *Var
	recvTypeParams *TypeParamList
	typeParams     *TypeParamList
	params         *Tuple
	results        *Tuple
	variadic       bool
	goType         gotypes.Type
}

// NewSignature constructs a standard Go Signature (CallConv = Go),
// matching go/types.NewSignature's 4-parameter form.
func NewSignature(recv *Var, params, results *Tuple, variadic bool) *Signature {
	return &Signature{
		callConv: Go,
		recv:     recv,
		params:   params,
		results:  results,
		variadic: variadic,
	}
}

// NewSignatureType constructs a (possibly generic) standard Go Signature
// (CallConv = Go), matching go/types.NewSignatureType's 6-parameter form.
func NewSignatureType(recv *Var, recvTypeParams, typeParams []*TypeParam, params, results *Tuple, variadic bool) *Signature {
	return &Signature{
		callConv:       Go,
		recv:           recv,
		recvTypeParams: newTypeParamList(recvTypeParams),
		typeParams:     newTypeParamList(typeParams),
		params:         params,
		results:        results,
		variadic:       variadic,
	}
}

// NewSignatureWithCallConv constructs a Signature with an explicit CallConv.
func NewSignatureWithCallConv(recv *Var, params, results *Tuple, variadic bool, cc CallConv) *Signature {
	return &Signature{
		callConv: cc,
		recv:     recv,
		params:   params,
		results:  results,
		variadic: variadic,
	}
}

// WithCallConv returns a shallow copy of s using cc.
func (s *Signature) WithCallConv(cc CallConv) *Signature {
	if s == nil {
		return nil
	}
	cp := *s
	cp.callConv = cc
	return &cp
}

func (s *Signature) CallConv() CallConv             { return s.callConv }
func (s *Signature) Recv() *Var                     { return s.recv }
func (s *Signature) RecvTypeParams() *TypeParamList { return s.recvTypeParams }
func (s *Signature) TypeParams() *TypeParamList     { return s.typeParams }
func (s *Signature) Params() *Tuple                 { return s.params }
func (s *Signature) Results() *Tuple                { return s.results }
func (s *Signature) Variadic() bool                 { return s.variadic }
func (s *Signature) IsClosure() bool                { return s.callConv.IsClosure() }
func (s *Signature) IsNative() bool                 { return s.callConv.IsNative() }
func (s *Signature) IsGoVariadic() bool             { return s.variadic && s.callConv.IsGo() }
func (s *Signature) IsCVariadic() bool              { return s.variadic && s.callConv.IsNative() }
func (s *Signature) IsPyVariadic() bool             { return s.variadic && s.callConv == Py }
func (s *Signature) Underlying() Type         { return s }
func (s *Signature) goTypeRef() gotypes.Type  { return s.goType }
func (s *Signature) llTypeMarker()            {}
func (s *Signature) HasPtrData() bool         { return s.callConv.IsClosure() }
func (s *Signature) IsHostLayoutMarker() bool { return false }
func (s *Signature) HasHostLayout() bool      { return false }
func (s *Signature) ContainsHostLayout() bool { return false }
func (s *Signature) ContainsSIMDStorage() bool { return false }
func (s *Signature) SIMDTagIndex() int        { return -1 }
func (s *Signature) SIMDVec() *SIMDVec        { return nil }

func (s *Signature) String() string {
	var b strings.Builder
	b.WriteString("func")
	if s.callConv != Go {
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

func (s *Slice) Elem() Type              { return s.elem }
func (s *Slice) Underlying() Type        { return s }
func (s *Slice) goTypeRef() gotypes.Type { return s.goType }
func (s *Slice) llTypeMarker()           {}
func (s *Slice) HasPtrData() bool        { return true }
func (s *Slice) IsHostLayoutMarker() bool { return false }
func (s *Slice) HasHostLayout() bool      { return false }
func (s *Slice) ContainsHostLayout() bool { return false }
func (s *Slice) ContainsSIMDStorage() bool { return false }
func (s *Slice) SIMDTagIndex() int        { return -1 }
func (s *Slice) SIMDVec() *SIMDVec        { return nil }
func (s *Slice) String() string          { return "[]" + s.elem.String() }

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

func (a *Array) Elem() Type              { return a.elem }
func (a *Array) Len() int64              { return a.length }
func (a *Array) NativeStorage() bool     { return a.nativeStorage }
func (a *Array) Underlying() Type        { return a }
func (a *Array) goTypeRef() gotypes.Type { return a.goType }
func (a *Array) llTypeMarker()           {}
func (a *Array) HasPtrData() bool        { return a.length > 0 && a.elem != nil && a.elem.HasPtrData() }
func (a *Array) IsHostLayoutMarker() bool { return false }
func (a *Array) HasHostLayout() bool      { return false }
func (a *Array) ContainsHostLayout() bool { return a.elem != nil && a.elem.ContainsHostLayout() }
func (a *Array) ContainsSIMDStorage() bool { return a.elem != nil && a.elem.ContainsSIMDStorage() }
func (a *Array) SIMDTagIndex() int        { return -1 }
func (a *Array) SIMDVec() *SIMDVec        { return nil }
func (a *Array) String() string          { return fmt.Sprintf("[%d]%s", a.length, a.elem.String()) }

// Map represents a Go map type (map[K]V).
type Map struct {
	key    Type
	elem   Type
	goType gotypes.Type
}

func NewMap(key, elem Type) *Map { return &Map{key: key, elem: elem} }

func (m *Map) Key() Type               { return m.key }
func (m *Map) Elem() Type              { return m.elem }
func (m *Map) Underlying() Type        { return m }
func (m *Map) goTypeRef() gotypes.Type { return m.goType }
func (m *Map) llTypeMarker()           {}
func (m *Map) HasPtrData() bool        { return true }
func (m *Map) IsHostLayoutMarker() bool { return false }
func (m *Map) HasHostLayout() bool      { return false }
func (m *Map) ContainsHostLayout() bool { return false }
func (m *Map) ContainsSIMDStorage() bool { return false }
func (m *Map) SIMDTagIndex() int        { return -1 }
func (m *Map) SIMDVec() *SIMDVec        { return nil }
func (m *Map) String() string          { return "map[" + m.key.String() + "]" + m.elem.String() }

// Chan represents a Go channel type.
type Chan struct {
	dir    ChanDir
	elem   Type
	goType gotypes.Type
}

func NewChan(dir ChanDir, elem Type) *Chan { return &Chan{dir: dir, elem: elem} }

func (c *Chan) Dir() ChanDir            { return c.dir }
func (c *Chan) Elem() Type              { return c.elem }
func (c *Chan) Underlying() Type        { return c }
func (c *Chan) goTypeRef() gotypes.Type { return c.goType }
func (c *Chan) llTypeMarker()           {}
func (c *Chan) HasPtrData() bool        { return true }
func (c *Chan) IsHostLayoutMarker() bool { return false }
func (c *Chan) HasHostLayout() bool      { return false }
func (c *Chan) ContainsHostLayout() bool { return false }
func (c *Chan) ContainsSIMDStorage() bool { return false }
func (c *Chan) SIMDTagIndex() int        { return -1 }
func (c *Chan) SIMDVec() *SIMDVec        { return nil }
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
// Methods and embeddeds lowered from go/types are resolved lazily so that
// referencing an unchecked interface behind a pointer during go/types.Check
// never forces premature resolution.
type Interface struct {
	ctx        *Context
	goIface    *gotypes.Interface
	resolved   bool
	methods    []*Func
	embeddeds  []Type
	comparable bool
	goType     gotypes.Type
}

// NewInterfaceType constructs an Interface type (matching go/types.NewInterfaceType).
func NewInterfaceType(methods []*Func, embeddeds []Type) *Interface {
	return &Interface{
		resolved:   true,
		methods:    methods,
		embeddeds:  embeddeds,
		comparable: true,
	}
}

// NewInterface constructs an Interface from methods and embedded *Named types
// (matching go/types.NewInterface).
func NewInterface(methods []*Func, embeddeds []*Named) *Interface {
	var embs []Type
	if len(embeddeds) > 0 {
		embs = make([]Type, len(embeddeds))
		for i, e := range embeddeds {
			embs[i] = e
		}
	}
	return NewInterfaceType(methods, embs)
}

func (t *Interface) resolve() {
	if t.resolved || t.ctx == nil || t.goIface == nil {
		return
	}
	t.ctx.resolveInterface(t)
}

// Complete computes the interface's method set and returns t (matching go/types).
func (t *Interface) Complete() *Interface {
	t.resolve()
	return t
}

func (t *Interface) NumMethods() int {
	t.resolve()
	return len(t.methods)
}
func (t *Interface) Method(i int) *Func {
	t.resolve()
	return t.methods[i]
}
func (t *Interface) NumEmbeddeds() int {
	t.resolve()
	return len(t.embeddeds)
}
func (t *Interface) EmbeddedType(i int) Type {
	t.resolve()
	return t.embeddeds[i]
}
func (t *Interface) Empty() bool {
	t.resolve()
	return len(t.methods) == 0 && len(t.embeddeds) == 0
}
func (t *Interface) Comparable() bool         { return t.comparable }
func (t *Interface) Underlying() Type         { return t }
func (t *Interface) goTypeRef() gotypes.Type  { return t.goType }
func (t *Interface) llTypeMarker()            {}
func (t *Interface) HasPtrData() bool         { return true }
func (t *Interface) IsHostLayoutMarker() bool { return false }
func (t *Interface) HasHostLayout() bool      { return false }
func (t *Interface) ContainsHostLayout() bool { return false }
func (t *Interface) ContainsSIMDStorage() bool { return false }
func (t *Interface) SIMDTagIndex() int        { return -1 }
func (t *Interface) SIMDVec() *SIMDVec        { return nil }
func (t *Interface) String() string {
	t.resolve()
	if len(t.methods) == 0 && len(t.embeddeds) == 0 {
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

// NewStruct constructs a standard Go Struct (matching go/types.NewStruct).
func NewStruct(fields []*Var, tags []string) *Struct {
	cp := make([]*Var, len(fields))
	copy(cp, fields)
	var tagCopy []string
	if len(tags) > 0 {
		tagCopy = make([]string, len(tags))
		copy(tagCopy, tags)
	}
	host := false
	for _, f := range cp {
		if f != nil && f.typ != nil && f.typ.IsHostLayoutMarker() {
			host = true
			break
		}
	}
	return &Struct{
		fields:       cp,
		tags:         tagCopy,
		hostLayout:   host,
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
func (s *Struct) HasHostLayout() bool      { return s.hostLayout }
func (s *Struct) NativeLayout() bool       { return s.nativeLayout }
func (s *Struct) IsClosureStruct() bool    { return s.isClosureStruct }
func (s *Struct) SIMDTagIndex() int        { return s.simdTagIndex }
func (s *Struct) SIMDVec() *SIMDVec        { return nil }
func (s *Struct) SetHostLayout(v bool)     { s.hostLayout = v }
func (s *Struct) SetNativeLayout(v bool)   { s.nativeLayout = v }
func (s *Struct) Underlying() Type         { return s }
func (s *Struct) goTypeRef() gotypes.Type  { return s.goType }
func (s *Struct) llTypeMarker()            {}
func (s *Struct) IsHostLayoutMarker() bool {
	if len(s.fields) != 1 || s.fields[0] == nil || s.fields[0].name != "_" || s.fields[0].typ == nil {
		return false
	}
	n, ok := Unalias(s.fields[0].typ).(*Named)
	if !ok || n.obj == nil || n.obj.pkg == nil || !isStructsPkgPath(n.obj.pkg.path) || n.obj.name != "hostLayout" {
		return false
	}
	body, ok := n.Underlying().(*Struct)
	return ok && len(body.fields) == 0
}
func (s *Struct) ContainsHostLayout() bool {
	if s.hostLayout {
		return true
	}
	for _, f := range s.fields {
		if f != nil && f.typ != nil && f.typ.ContainsHostLayout() {
			return true
		}
	}
	return false
}
func (s *Struct) ContainsSIMDStorage() bool {
	if s.simdTagIndex >= 0 {
		return true
	}
	for _, f := range s.fields {
		if f != nil && f.typ != nil && f.typ.ContainsSIMDStorage() {
			return true
		}
	}
	return false
}
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

func (v *SIMDVec) Elem() *Basic            { return v.elem }
func (v *SIMDVec) Len() int                { return v.length }
func (v *SIMDVec) IsMask() bool            { return v.isMask }
func (v *SIMDVec) TagIndex() int           { return v.tagIndex }
func (v *SIMDVec) BitWidth() int           { return basicBits(v.elem) * v.length }
func (v *SIMDVec) Underlying() Type        { return v }
func (v *SIMDVec) goTypeRef() gotypes.Type { return v.goType }
func (v *SIMDVec) llTypeMarker()           {}
func (v *SIMDVec) HasPtrData() bool        { return false }
func (v *SIMDVec) IsHostLayoutMarker() bool { return false }
func (v *SIMDVec) HasHostLayout() bool      { return false }
func (v *SIMDVec) ContainsHostLayout() bool { return false }
func (v *SIMDVec) ContainsSIMDStorage() bool { return true }
func (v *SIMDVec) SIMDTagIndex() int        { return -1 }
func (v *SIMDVec) SIMDVec() *SIMDVec        { return v }
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
//
// Named types lowered from go/types resolve their underlying type and method
// list lazily on demand, mirroring go/types.Named so referencing an unchecked
// Named type behind a pointer during go/types.Check never triggers premature
// go/types resolution assertions.
type Named struct {
	ctx             *Context
	goNamed         *gotypes.Named
	overrideCC      CallConv
	hasOverrideCC   bool
	resolved        bool
	methodsResolved bool

	obj          *TypeName
	origin       *Named
	typeParams   *TypeParamList
	typeArgs     *TypeList
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

// NewNamed constructs a defined type bound to obj (matching go/types.NewNamed).
func NewNamed(obj *TypeName, underlying Type, methods []*Func) *Named {
	n := &Named{
		resolved:        true,
		methodsResolved: true,
		obj:             obj,
		underlying:      underlying,
		methods:         methods,
		simdTagIndex:    -1,
	}
	if obj != nil {
		obj.typ = n
	}
	return n
}

func (n *Named) resolve() {
	if n.ctx == nil || n.goNamed == nil {
		return
	}
	n.ctx.resolveNamed(n)
}

func (n *Named) resolveMethods() {
	if n.methodsResolved || n.ctx == nil || n.goNamed == nil {
		return
	}
	n.ctx.resolveNamedMethods(n)
}

func (n *Named) Obj() *TypeName { return n.obj }

func (n *Named) Origin() *Named {
	n.resolve()
	if n.origin != nil {
		return n.origin
	}
	return n
}

func (n *Named) TypeParams() *TypeParamList {
	n.resolve()
	return n.typeParams
}

func (n *Named) TypeArgs() *TypeList {
	n.resolve()
	return n.typeArgs
}

func (n *Named) NumMethods() int {
	n.resolveMethods()
	return len(n.methods)
}

func (n *Named) Method(i int) *Func {
	n.resolveMethods()
	return n.methods[i]
}

func (n *Named) AddMethod(m *Func) {
	n.resolveMethods()
	n.methods = append(n.methods, m)
}

func (n *Named) CallConv() CallConv        { return n.callConv }
func (n *Named) HasExplicitCallConv() bool { return n.hasCallConv }

func (n *Named) IsHostLayoutMarker() bool {
	if n.obj != nil && n.obj.pkg != nil && isStructsPkgPath(n.obj.pkg.path) && n.obj.name == "HostLayout" {
		return true
	}
	n.resolve()
	if n.underlying == nil || n.underlying == n {
		return false
	}
	return n.underlying.IsHostLayoutMarker()
}

func (n *Named) HasHostLayout() bool {
	n.resolve()
	if n.hostLayout {
		return true
	}
	if n.underlying == nil || n.underlying == n {
		return false
	}
	return n.underlying.HasHostLayout()
}

func (n *Named) ContainsHostLayout() bool {
	n.resolve()
	if n.hostLayout {
		return true
	}
	if n.underlying == nil || n.underlying == n {
		return false
	}
	return n.underlying.ContainsHostLayout()
}

func (n *Named) ContainsSIMDStorage() bool {
	n.resolve()
	if n.simdTagIndex >= 0 || n.simdVec != nil {
		return true
	}
	if n.underlying == nil || n.underlying == n {
		return false
	}
	return n.underlying.ContainsSIMDStorage()
}

func (n *Named) NativeLayout() bool {
	n.resolve()
	return n.nativeLayout
}

func (n *Named) SIMDTagIndex() int {
	n.resolve()
	return n.simdTagIndex
}

func (n *Named) SIMDVec() *SIMDVec {
	n.resolve()
	return n.simdVec
}

func (n *Named) goTypeRef() gotypes.Type { return n.goType }
func (n *Named) llTypeMarker()           {}

func (n *Named) SetUnderlying(u Type) {
	n.underlying = u
	n.resolved = true
}

func (n *Named) Underlying() Type {
	n.resolve()
	if n.underlying == nil {
		return n
	}
	return n.underlying.Underlying()
}

func (n *Named) HasPtrData() bool {
	n.resolve()
	if n.underlying == nil {
		return false
	}
	return n.underlying.HasPtrData()
}

func (n *Named) String() string {
	return writeTypeString(n, nil)
}

// Alias represents a type alias (type A = Rhs), preserving any LLGo calling
// convention directive attached directly to the alias declaration (e.g.
// //llgo:cdecl type Cb = func()).
type Alias struct {
	obj         *TypeName
	rhs         Type
	origin      *Alias
	typeParams  *TypeParamList
	typeArgs    *TypeList
	callConv    CallConv
	hasCallConv bool
	goType      gotypes.Type
}

// NewAlias constructs a type alias (matching go/types.NewAlias).
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
func (a *Alias) TypeParams() *TypeParamList { return a.typeParams }
func (a *Alias) TypeArgs() *TypeList        { return a.typeArgs }
func (a *Alias) CallConv() CallConv        { return a.callConv }
func (a *Alias) HasExplicitCallConv() bool { return a.hasCallConv }
func (a *Alias) goTypeRef() gotypes.Type   { return a.goType }
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

func (a *Alias) IsHostLayoutMarker() bool {
	if a.rhs == nil || a.rhs == a {
		return false
	}
	return a.rhs.IsHostLayoutMarker()
}

func (a *Alias) HasHostLayout() bool {
	if a.rhs == nil || a.rhs == a {
		return false
	}
	return a.rhs.HasHostLayout()
}

func (a *Alias) ContainsHostLayout() bool {
	if a.rhs == nil || a.rhs == a {
		return false
	}
	return a.rhs.ContainsHostLayout()
}

func (a *Alias) ContainsSIMDStorage() bool {
	if a.rhs == nil || a.rhs == a {
		return false
	}
	return a.rhs.ContainsSIMDStorage()
}

func (a *Alias) SIMDTagIndex() int {
	if a.rhs == nil || a.rhs == a {
		return -1
	}
	return a.rhs.SIMDTagIndex()
}

func (a *Alias) SIMDVec() *SIMDVec {
	if a.rhs == nil || a.rhs == a {
		return nil
	}
	return a.rhs.SIMDVec()
}

func (a *Alias) String() string {
	return writeTypeString(a, nil)
}

// TypeParam represents a generic type parameter.
type TypeParam struct {
	obj        *TypeName
	index      int
	constraint Type
	goType     gotypes.Type
}

// NewTypeParam constructs a generic type parameter (matching go/types.NewTypeParam).
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
func (tp *TypeParam) HasPtrData() bool         { return true }
func (tp *TypeParam) IsHostLayoutMarker() bool { return false }
func (tp *TypeParam) HasHostLayout() bool      { return false }
func (tp *TypeParam) ContainsHostLayout() bool { return false }
func (tp *TypeParam) ContainsSIMDStorage() bool { return false }
func (tp *TypeParam) SIMDTagIndex() int        { return -1 }
func (tp *TypeParam) SIMDVec() *SIMDVec        { return nil }
func (tp *TypeParam) goTypeRef() gotypes.Type  { return tp.goType }
func (tp *TypeParam) llTypeMarker()            {}
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
		if !ok || x.NumMethods() != y.NumMethods() || x.NumEmbeddeds() != y.NumEmbeddeds() {
			return false
		}
		for i := 0; i < x.NumMethods(); i++ {
			xm := x.Method(i)
			ym := y.Method(i)
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
		xArgs := x.TypeArgs()
		yArgs := y.TypeArgs()
		if x.obj == y.obj && xArgs.Len() == yArgs.Len() {
			for i := 0; i < xArgs.Len(); i++ {
				if !Identical(xArgs.At(i), yArgs.At(i)) {
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
