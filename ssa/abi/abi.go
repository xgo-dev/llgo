/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package abi

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"go/types"
	"hash"
	"log"
	"strconv"
	"strings"

	"github.com/goplus/llgo/internal/env"
	"github.com/goplus/llgo/runtime/abi"
)

// -----------------------------------------------------------------------------

func BasicKind(t *types.Basic) abi.Kind {
	kind := t.Kind()
	switch kind {
	case types.String:
		return abi.String
	case types.UnsafePointer:
		return abi.UnsafePointer
	}
	return abi.Kind(kind)
}

func UnderlyingKind(t types.Type) abi.Kind {
	switch t := t.(type) {
	case *types.Basic:
		return BasicKind(t)
	case *types.Pointer:
		return abi.Pointer
	case *types.Slice:
		return abi.Slice
	case *types.Signature:
		return abi.Func
	case *types.Interface:
		return abi.Interface
	case *types.Struct:
		return abi.Struct
	case *types.Map:
		return abi.Map
	case *types.Array:
		return abi.Array
	case *types.Chan:
		return abi.Chan
	}
	panic("todo")
}

func ChanDir(dir types.ChanDir) (abi.ChanDir, string) {
	switch dir {
	case types.SendRecv:
		return abi.BothDir, "chan"
	case types.SendOnly:
		return abi.SendDir, "chan<-"
	case types.RecvOnly:
		return abi.RecvDir, "<-chan"
	}
	panic("invalid chan dir")
}

// -----------------------------------------------------------------------------

type DataKind int

const (
	Invalid  DataKind = iota
	Indirect          // allocate memory for the value
	Pointer           // store a pointer value directly in the interface value
	Integer           // store a integer value directly in the interface value
	BitCast           // store other value (need bitcast) directly in the interface value
)

func DataKindOf(raw types.Type, lvl int, is32Bits bool) (DataKind, types.Type, int) {
	switch t := raw.Underlying().(type) {
	case *types.Basic:
		kind := t.Kind()
		switch {
		case types.Bool <= kind && kind <= types.Uintptr:
			if is32Bits && (kind == types.Int64 || kind == types.Uint64) {
				return Indirect, raw, lvl
			}
			return Integer, raw, lvl
		case kind == types.Float32:
			return BitCast, raw, lvl
		case kind == types.Float64:
			if is32Bits {
				return Indirect, raw, lvl
			}
			return BitCast, raw, lvl
		case kind == types.UnsafePointer:
			return Pointer, raw, lvl
		}
	case *types.Pointer, *types.Signature, *types.Map, *types.Chan:
		return Pointer, raw, lvl
	case *types.Struct:
		if t.NumFields() == 1 {
			return DataKindOf(t.Field(0).Type(), lvl+1, is32Bits)
		}
	case *types.Interface, *types.Slice:
	case *types.Array:
		if t.Len() == 1 {
			return DataKindOf(t.Elem(), lvl+1, is32Bits)
		}
	default:
		panic("unkown type")
	}
	return Indirect, raw, lvl
}

// -----------------------------------------------------------------------------

// Builder is a helper for constructing ABI types.
type Builder struct {
	buf     []byte
	PtrSize uintptr
	Sizes   types.Sizes
}

// New creates a new ABI type Builder.
func New(ptrSize uintptr, sizes types.Sizes) *Builder {
	ret := new(Builder)
	ret.Init(ptrSize, sizes)
	return ret
}

func (b *Builder) Init(ptrSize uintptr, sizes types.Sizes) {
	b.buf = make([]byte, sha256.Size)
	b.PtrSize = ptrSize
	b.Sizes = sizes
}

// TypeName returns the ABI type name for the specified type.
func (b *Builder) TypeName(t types.Type) (ret string, pub bool) {
	switch t := t.(type) {
	case *types.Basic:
		return BasicName(t), true
	case *types.Pointer:
		ret, pub = b.TypeName(t.Elem())
		return "*" + ret, pub
	case *types.Struct:
		return b.StructName(t)
	case *types.Signature:
		return b.FuncName(t), true
	case *types.Slice:
		ret, pub = b.TypeName(t.Elem())
		return "[]" + ret, pub
	case *types.Array:
		ret, pub = b.TypeName(t.Elem())
		return fmt.Sprintf("[%v]%s", t.Len(), ret), pub
	case *types.Named:
		o := t.Obj()
		pkg := o.Pkg()
		ids := scopeIndices(o)
		return "_llgo_" + FullName(pkg, NamedName(t)+ids), (pkg == nil || o.Exported() && ids == "")
	case *types.Interface:
		if t.Empty() {
			return "_llgo_any", true
		}
		return b.InterfaceName(t)
	case *types.Map:
		key, pub1 := b.TypeName(t.Key())
		elem, pub2 := b.TypeName(t.Elem())
		return fmt.Sprintf("map[%s]%s", key, elem), pub1 && pub2
	case *types.Chan:
		elem, pub := b.TypeName(t.Elem())
		var s string
		switch t.Dir() {
		case types.SendRecv:
			s = "chan"
		case types.SendOnly:
			s = "chan<-"
		case types.RecvOnly:
			s = "<-chan"
		}
		return fmt.Sprintf("%s %s", s, elem), pub
	case *types.Alias:
		return b.TypeName(types.Unalias(t))
	}
	log.Panicf("todo: %T\n", t)
	return
}

func NamedName(t *types.Named) string {
	if targs := t.TypeArgs(); targs != nil {
		n := targs.Len()
		infos := make([]string, n)
		for i := 0; i < n; i++ {
			infos[i] = typeArgString(targs.At(i))
		}
		return t.Obj().Name() + "[" + strings.Join(infos, ",") + "]"
	}
	return t.Obj().Name()
}

func TypeArgs(typeArgs []types.Type) string {
	targs := make([]string, len(typeArgs))
	for i, t := range typeArgs {
		targs[i] = typeArgString(t)
	}
	return "[" + strings.Join(targs, ",") + "]"
}

func namedLikeTypeArgString(obj types.Object, targs *types.TypeList) string {
	name := obj.Name()
	if targs != nil {
		n := targs.Len()
		infos := make([]string, n)
		for i := 0; i < n; i++ {
			infos[i] = typeArgString(targs.At(i))
		}
		name += "[" + strings.Join(infos, ",") + "]"
	}
	// Distinct local types may share the same object name (e.g. in stdlib
	// tests). Disambiguate them in symbol names using stable scope indices.
	name += scopeIndices(obj)
	if pkg := obj.Pkg(); pkg != nil {
		return PathOf(pkg) + "." + name
	}
	return name
}

func typeArgString(t types.Type) string {
	switch t := t.(type) {
	case *types.Alias:
		return typeArgString(types.Unalias(t))
	case *types.Basic:
		return t.String()
	case *types.Named:
		return namedLikeTypeArgString(t.Obj(), t.TypeArgs())
	case *types.Pointer:
		return "*" + typeArgString(t.Elem())
	case *types.Slice:
		return "[]" + typeArgString(t.Elem())
	case *types.Array:
		return fmt.Sprintf("[%v]%s", t.Len(), typeArgString(t.Elem()))
	case *types.Map:
		return fmt.Sprintf("map[%s]%s", typeArgString(t.Key()), typeArgString(t.Elem()))
	case *types.Chan:
		_, s := ChanDir(t.Dir())
		elem := t.Elem()
		elemStr := typeArgString(elem)
		// Keep canonical channel formatting for nested directional channels.
		// Example: chan (<-chan int), not "chan <-chan int" (ambiguous).
		if t.Dir() == types.SendRecv {
			if ch, ok := elem.(*types.Chan); ok && ch.Dir() == types.RecvOnly {
				elemStr = "(" + elemStr + ")"
			}
		}
		return fmt.Sprintf("%s %s", s, elemStr)
	default:
		// Fallback for rare type arguments (e.g. signature/interface/struct).
		// Collisions are mainly caused by local named types, handled above.
		return types.TypeString(t, PathOf)
	}
}

const (
	PatchPathPrefix = env.LLGoRuntimePkg + "/internal/lib/"
)

// PathOf returns the package path of the specified package.
func PathOf(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return strings.TrimPrefix(pkg.Path(), PatchPathPrefix)
}

// FullName returns the full name of a package member.
func FullName(pkg *types.Package, name string) string {
	if pkg == nil {
		return name
	}
	return PathOf(pkg) + "." + name
}

// BasicName returns the ABI type name for the specified basic type.
func BasicName(t *types.Basic) string {
	name := t.Name()
	switch name {
	case "byte":
		name = "uint8"
	case "rune":
		name = "int32"
	}
	return "_llgo_" + name
}

// FuncName returns the ABI type name for the specified function type.
func (b *Builder) FuncName(t *types.Signature) string {
	hash := b.funcHash(t)
	hashStr := base64.RawURLEncoding.EncodeToString(hash)
	return "_llgo_func$" + hashStr
}

func (b *Builder) funcHash(t *types.Signature) []byte {
	h := sha256.New()
	params, results := t.Params(), t.Results()
	fmt.Fprintln(h, "func", params.Len(), results.Len(), t.Variadic())
	b.tuple(h, params)
	b.tuple(h, results)
	return h.Sum(b.buf[:0])
}

func (b *Builder) tuple(h hash.Hash, t *types.Tuple) {
	n := t.Len()
	for i := 0; i < n; i++ {
		v := t.At(i)
		ft, _ := b.TypeName(PublicType(v.Type()))
		fmt.Fprintln(h, ft)
	}
}

// InterfaceName returns the ABI type name for the specified interface type.
func (b *Builder) InterfaceName(t *types.Interface) (ret string, pub bool) {
	hash, pkg := b.interfaceHash(t)
	hashStr := base64.RawURLEncoding.EncodeToString(hash)
	if pkg != "" {
		return pkg + ".iface$" + hashStr, false
	}
	return "_llgo_iface$" + hashStr, true
}

func (b *Builder) interfaceHash(t *types.Interface) (ret []byte, pkg string) {
	h := sha256.New()
	n := t.NumMethods()
	fmt.Fprintln(h, "interface", n)
	for i := 0; i < n; i++ {
		m := t.Method(i)
		if !m.Exported() && pkg == "" {
			pkg = PathOf(m.Pkg())
		}
		ft := b.FuncName(m.Type().(*types.Signature))
		fmt.Fprintln(h, m.Name(), ft)
	}
	ret = h.Sum(b.buf[:0])
	return
}

// StructName returns the ABI type name for the specified struct type.
func (b *Builder) StructName(t *types.Struct) (ret string, pub bool) {
	hash, pkg := b.structHash(t)
	hashStr := base64.RawURLEncoding.EncodeToString(hash)
	if IsClosure(t) {
		return "_llgo_closure$" + hashStr, true
	}
	if pkg != "" {
		return pkg + ".struct$" + hashStr, false
	}
	return "_llgo_struct$" + hashStr, false
}

func IsClosure(raw *types.Struct) bool {
	n := raw.NumFields()
	if n == 2 {
		f1, f2 := raw.Field(0), raw.Field(1)
		if _, ok := f1.Type().(*types.Signature); ok && f1.Name() == "$f" {
			return f2.Type() == types.Typ[types.UnsafePointer] && f2.Name() == "$data"
		}
	}
	return false
}

func IsClosureFields(fields []*types.Var) bool {
	if len(fields) == 2 {
		f1, f2 := fields[0], fields[1]
		if _, ok := f1.Type().(*types.Signature); ok && f1.Name() == "$f" {
			return f2.Type() == types.Typ[types.UnsafePointer] && f2.Name() == "$data"
		}
	}
	return false
}

func (b *Builder) structHash(t *types.Struct) (ret []byte, pkg string) {
	h := sha256.New()
	n := t.NumFields()
	fmt.Fprintln(h, "struct", n)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if pkg == "" && !f.Exported() && f.Pkg() != nil {
			pkg = f.Pkg().Path()
		}
		name := f.Name()
		if f.Embedded() {
			name = "-"
		}
		ft, _ := b.TypeName(f.Type())
		fmt.Fprintln(h, name, ft)
		if tag := t.Tag(i); tag != "" {
			fmt.Fprintln(h, "tag", tag)
		}
	}
	ret = h.Sum(b.buf[:0])
	return
}

func scopeIndex(scope, root *types.Scope, id string) (string, bool) {
	if scope == nil || root == nil {
		return "", false
	}
	parent := scope.Parent()
	if parent == nil {
		return "", false
	}
	n := parent.NumChildren()
	for i := 0; i < n; i++ {
		if parent.Child(i) == scope {
			id += "." + strconv.Itoa(i)
			break
		}
	}
	if parent == root {
		return id, true
	}
	return scopeIndex(parent, root, id)
}

func scopeIndices(obj types.Object) string {
	pkg := obj.Pkg()
	if pkg == nil {
		return ""
	}
	if obj.Parent() != pkg.Scope() {
		if ids, ok := scopeIndex(obj.Parent(), pkg.Scope(), ""); ok {
			return ids
		}
		if pos := obj.Pos(); pos.IsValid() {
			return ".p" + strconv.Itoa(int(pos))
		}
	}
	return ""
}

// -----------------------------------------------------------------------------
