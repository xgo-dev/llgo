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

// Package types defines LLGo's physical and ABI type system mapped from
// read-only go/types.Type without synthesizing fake go/types nodes or depending
// on LLVM/CGo.
//
// First-class LLGo representations—including Closure ({fn, env}), C/Stdcall
// function pointers, GoPtr vs NativePtr pointer kinds, HostLayout structs,
// Wasm32 wide Go storage, and SIMD vectors—are modeled explicitly so compiler
// passes, static analyzers, and custom GC/ABI emitters share a single pure-Go
// type model.
package types

import (
	"fmt"
	gotypes "go/types"
	"strings"
)

// CallConv identifies the calling convention and storage view of a function or
// type declaration.
type CallConv uint8

const (
	CallConvUnknown CallConv = iota
	GoDecl
	CDecl
	PyDecl
	Stdcall
)

// IsNative reports whether cc represents a C/native calling convention rather
// than a Go closure calling convention.
func (cc CallConv) IsNative() bool {
	return cc == CDecl || cc == Stdcall
}

func (cc CallConv) String() string {
	switch cc {
	case GoDecl:
		return "go"
	case CDecl:
		return "cdecl"
	case PyDecl:
		return "py"
	case Stdcall:
		return "stdcall"
	default:
		return "unknown"
	}
}

// PtrKind distinguishes GC-traced Go heap/stack pointers from unmanaged native
// C/FFI pointers, enabling precise stack maps, write barriers, and NoScan
// optimizations for future GC implementations.
type PtrKind uint8

const (
	GoPtr PtrKind = iota
	NativePtr
)

func (k PtrKind) String() string {
	if k == NativePtr {
		return "nativeptr"
	}
	return "goptr"
}

// Type represents an LLGo physical/ABI type.
type Type interface {
	Underlying() Type
	String() string
	HasPtrData() bool
	GoType() gotypes.Type
	llTypeMarker()
}

// Basic represents a scalar basic type (integers, floats, complex, bool,
// unsafe.Pointer). Go strings are represented by *String.
type Basic struct {
	Kind    gotypes.BasicKind
	PtrKind PtrKind // meaningful when Kind == gotypes.UnsafePointer
	goType  gotypes.Type
}

func (t *Basic) Underlying() Type     { return t }
func (t *Basic) GoType() gotypes.Type { return t.goType }
func (t *Basic) llTypeMarker()        {}
func (t *Basic) HasPtrData() bool {
	return t.Kind == gotypes.UnsafePointer && t.PtrKind == GoPtr
}
func (t *Basic) String() string {
	if t.goType != nil {
		return t.goType.String()
	}
	return gotypes.Typ[t.Kind].String()
}

// Pointer represents a typed pointer with explicit GC traceability (GoPtr vs NativePtr).
type Pointer struct {
	Elem   Type
	Kind   PtrKind
	goType gotypes.Type
}

func (t *Pointer) Underlying() Type     { return t }
func (t *Pointer) GoType() gotypes.Type { return t.goType }
func (t *Pointer) llTypeMarker()        {}
func (t *Pointer) HasPtrData() bool     { return t.Kind == GoPtr }
func (t *Pointer) String() string {
	if t.Kind == NativePtr {
		return "native*" + t.Elem.String()
	}
	return "*" + t.Elem.String()
}

// Var represents a function parameter, receiver, or result.
type Var struct {
	Name string
	Type Type
}

// Func represents a concrete callable function prototype (Go code entry, C
// callback, Stdcall callback, or Python callable).
type Func struct {
	CallConv CallConv
	Recv     *Var
	Params   []*Var
	Results  []*Var
	Variadic bool
	HasEnv   bool
	goType   gotypes.Type
}

func (t *Func) Underlying() Type     { return t }
func (t *Func) GoType() gotypes.Type { return t.goType }
func (t *Func) llTypeMarker()        {}
func (t *Func) HasPtrData() bool     { return t.CallConv == GoDecl }
func (t *Func) String() string {
	var b strings.Builder
	if t.CallConv != GoDecl && t.CallConv != CallConvUnknown {
		b.WriteString(t.CallConv.String())
		b.WriteByte(' ')
	}
	b.WriteString("func(")
	for i, p := range t.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		if t.Variadic && i == len(t.Params)-1 {
			b.WriteString("...")
		}
		b.WriteString(p.Type.String())
	}
	b.WriteByte(')')
	if len(t.Results) == 1 {
		b.WriteByte(' ')
		b.WriteString(t.Results[0].Type.String())
	} else if len(t.Results) > 1 {
		b.WriteString(" (")
		for i, r := range t.Results {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(r.Type.String())
		}
		b.WriteByte(')')
	}
	return b.String()
}

// Closure represents a first-class Go function value ({codePtr, envPtr}).
// Unlike ssa/type_cvt.go's synthetic go/types.Struct, Closure preserves its
// exact Go signature and GC-traced env pointer directly.
type Closure struct {
	Sig    *Func
	goType gotypes.Type
}

func (t *Closure) Underlying() Type     { return t }
func (t *Closure) GoType() gotypes.Type { return t.goType }
func (t *Closure) llTypeMarker()        {}
func (t *Closure) HasPtrData() bool     { return true }
func (t *Closure) String() string       { return "closure<" + t.Sig.String() + ">" }

// String represents a Go string header ({data *byte, len int}).
type String struct {
	goType gotypes.Type
}

func (t *String) Underlying() Type     { return t }
func (t *String) GoType() gotypes.Type { return t.goType }
func (t *String) llTypeMarker()        {}
func (t *String) HasPtrData() bool     { return true }
func (t *String) String() string       { return "string" }

// Slice represents a Go slice header ({data *Elem, len int, cap int}).
type Slice struct {
	Elem   Type
	goType gotypes.Type
}

func (t *Slice) Underlying() Type     { return t }
func (t *Slice) GoType() gotypes.Type { return t.goType }
func (t *Slice) llTypeMarker()        {}
func (t *Slice) HasPtrData() bool     { return true }
func (t *Slice) String() string       { return "[]" + t.Elem.String() }

// Interface represents a Go interface value ({type/itab ptr, data ptr}).
type Interface struct {
	Empty  bool
	goType gotypes.Type
}

func (t *Interface) Underlying() Type     { return t }
func (t *Interface) GoType() gotypes.Type { return t.goType }
func (t *Interface) llTypeMarker()        {}
func (t *Interface) HasPtrData() bool     { return true }
func (t *Interface) String() string {
	if t.Empty {
		return "any"
	}
	if t.goType != nil {
		return t.goType.String()
	}
	return "interface{...}"
}

// Map represents a Go map reference pointer.
type Map struct {
	Key    Type
	Elem   Type
	goType gotypes.Type
}

func (t *Map) Underlying() Type     { return t }
func (t *Map) GoType() gotypes.Type { return t.goType }
func (t *Map) llTypeMarker()        {}
func (t *Map) HasPtrData() bool     { return true }
func (t *Map) String() string       { return "map[" + t.Key.String() + "]" + t.Elem.String() }

// Chan represents a Go channel reference pointer.
type Chan struct {
	Dir    gotypes.ChanDir
	Elem   Type
	goType gotypes.Type
}

func (t *Chan) Underlying() Type     { return t }
func (t *Chan) GoType() gotypes.Type { return t.goType }
func (t *Chan) llTypeMarker()        {}
func (t *Chan) HasPtrData() bool     { return true }
func (t *Chan) String() string       { return "chan " + t.Elem.String() }

// Array represents a fixed-length array.
type Array struct {
	Elem          Type
	Len           int64
	NativeStorage bool
	goType        gotypes.Type
}

func (t *Array) Underlying() Type     { return t }
func (t *Array) GoType() gotypes.Type { return t.goType }
func (t *Array) llTypeMarker()        {}
func (t *Array) HasPtrData() bool     { return t.Len > 0 && t.Elem.HasPtrData() }
func (t *Array) String() string       { return fmt.Sprintf("[%d]%s", t.Len, t.Elem.String()) }

// Field represents one field in a Struct.
type Field struct {
	Name     string
	Type     Type
	Tag      string
	Embedded bool
}

// Struct represents a Go or C/HostLayout aggregate struct.
type Struct struct {
	Fields        []*Field
	HostLayout    bool
	NativeStorage bool
	goType        gotypes.Type
}

func (t *Struct) Underlying() Type     { return t }
func (t *Struct) GoType() gotypes.Type { return t.goType }
func (t *Struct) llTypeMarker()        {}
func (t *Struct) HasPtrData() bool {
	for _, f := range t.Fields {
		if f.Type.HasPtrData() {
			return true
		}
	}
	return false
}
func (t *Struct) String() string {
	var b strings.Builder
	b.WriteString("struct{")
	for i, f := range t.Fields {
		if i > 0 {
			b.WriteString("; ")
		}
		if f.Name != "" {
			b.WriteString(f.Name)
			b.WriteByte(' ')
		}
		b.WriteString(f.Type.String())
	}
	b.WriteByte('}')
	return b.String()
}

// Tuple represents a multi-value result tuple.
type Tuple struct {
	Elems  []*Var
	goType gotypes.Type
}

func (t *Tuple) Underlying() Type     { return t }
func (t *Tuple) GoType() gotypes.Type { return t.goType }
func (t *Tuple) llTypeMarker()        {}
func (t *Tuple) HasPtrData() bool {
	for _, e := range t.Elems {
		if e.Type.HasPtrData() {
			return true
		}
	}
	return false
}
func (t *Tuple) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, e := range t.Elems {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(e.Type.String())
	}
	b.WriteByte(')')
	return b.String()
}

// SIMDVec represents a fixed-width SIMD vector value (e.g. <4 x float>).
type SIMDVec struct {
	Elem   *Basic
	Len    int
	goType gotypes.Type
}

func (t *SIMDVec) Underlying() Type     { return t }
func (t *SIMDVec) GoType() gotypes.Type { return t.goType }
func (t *SIMDVec) llTypeMarker()        {}
func (t *SIMDVec) HasPtrData() bool     { return false }
func (t *SIMDVec) String() string       { return fmt.Sprintf("vec<%dx%s>", t.Len, t.Elem.String()) }

// Named represents a defined type preserving its Go package identity and
// calling-convention metadata while resolving recursive back-edges cleanly.
type Named struct {
	PkgPath    string
	Name       string
	CallConv   CallConv
	underlying Type
	goType     *gotypes.Named
}

func (t *Named) Underlying() Type {
	if t.underlying == nil {
		return nil
	}
	return t.underlying.Underlying()
}
func (t *Named) GoType() gotypes.Type { return t.goType }
func (t *Named) llTypeMarker()        {}
func (t *Named) HasPtrData() bool {
	if t.underlying == nil {
		return false
	}
	return t.underlying.HasPtrData()
}
func (t *Named) String() string {
	if t.PkgPath != "" {
		return t.PkgPath + "." + t.Name
	}
	return t.Name
}
