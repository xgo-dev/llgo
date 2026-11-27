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
	"github.com/xgo-dev/llgo/types"
	"hash"
	"log"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/env"
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
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
	switch raw.Underlying().(type) {
	case *types.Basic, *types.Pointer, *types.Signature, *types.Map, *types.Chan,
		*types.Struct, *types.Interface, *types.Slice, *types.Array:
	default:
		panic("unknown type")
	}
	dk, inner, depth := types.DataKindOf(raw, is32Bits)
	return DataKind(dk), inner, lvl + depth
}

// -----------------------------------------------------------------------------

type llgoContextProvider interface {
	LLGoContext() *lltypes.Context
}

// Builder is a helper for constructing ABI types.
type Builder struct {
	buf     []byte
	PtrSize uintptr
	Sizes   types.Sizes
	typeCtx *lltypes.Context
	// NativeFuncAliasABI reports the storage ABI of an alias to an unnamed function.
	NativeFuncAliasABI func(*types.Alias) string
	// NativeFuncType reports whether t uses a 1-word native function pointer ABI
	// rather than LLGo's 2-word Go closure representation.
	NativeFuncType func(types.Type) bool
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
	if p, ok := sizes.(llgoContextProvider); ok && p.LLGoContext() != nil {
		b.typeCtx = p.LLGoContext()
		return
	}
	maxAlign := int(ptrSize)
	if sizes != nil {
		if ma := int(sizes.Alignof(types.Typ[types.Complex128])); ma > 0 {
			maxAlign = ma
		}
	}
	b.typeCtx = lltypes.NewContextWithConfig(lltypes.ContextConfig{
		Spec: target.Spec{
			WordSize: int(ptrSize),
			PtrSize:  int(ptrSize),
			MaxAlign: maxAlign,
		},
	})
}

func (b *Builder) context() *lltypes.Context {
	if b.typeCtx == nil {
		b.Init(b.PtrSize, b.Sizes)
	}
	return b.typeCtx
}

func (b *Builder) isNativeFunc(t types.Type) bool {
	if b.NativeFuncType != nil && b.NativeFuncType(t) {
		return true
	}
	if alias, ok := t.(*types.Alias); ok && b.NativeFuncAliasABI != nil && b.NativeFuncAliasABI(alias) != "" {
		return true
	}
	switch u := t.(type) {
	case *types.Alias:
		if u.CallConv().IsNative() {
			_, ok := types.Unalias(u).(*types.Signature)
			return ok
		}
	case *types.Named:
		if u.CallConv().IsNative() {
			_, ok := u.Underlying().(*types.Signature)
			return ok
		}
	case *types.Signature:
		return u.CallConv().IsNative()
	}
	return false
}

// IsClosureType reports whether t is emitted as a two-word Go closure struct
// runtime descriptor (_llgo_closure$... or a named TFlagClosure structtype).
// When publicFunc is true, unnamed Go function types (and plain aliases to
// them) use the one-word public abi.Func descriptor (_llgo_func$...), while
// named Go function types keep their named closure descriptor.
func (b *Builder) IsClosureType(t types.Type, publicFunc bool) bool {
	if b.isNativeFunc(t) {
		return false
	}
	switch u := types.Unalias(t).(type) {
	case *types.Signature:
		if publicFunc {
			return false
		}
		return u.CallConv().IsClosure()
	case *types.Named:
		sig, ok := u.Underlying().(*types.Signature)
		return ok && sig.CallConv().IsClosure()
	}
	return false
}

// TypeName returns the ABI type name for the specified type.
func (b *Builder) TypeName(t types.Type) (ret string, pub bool) {
	return b.TypeNameEx(t, false)
}

// TypeNameEx returns the ABI type name for t. When publicFunc is true, an
// unnamed Go function type selects its public _llgo_func$... descriptor rather
// than its _llgo_closure$... value descriptor.
func (b *Builder) TypeNameEx(t types.Type, publicFunc bool) (ret string, pub bool) {
	switch t := t.(type) {
	case *types.Basic:
		return BasicName(t), true
	case *types.Pointer:
		ret, pub = b.TypeName(t.Elem())
		return "*" + ret, pub
	case *types.Struct:
		return b.StructName(t)
	case *types.Signature:
		if b.IsClosureType(t, publicFunc) {
			return b.ClosureName(t), true
		}
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
		if b.NativeFuncAliasABI != nil {
			if cc := b.NativeFuncAliasABI(t); cc != "" {
				name, pub := b.TypeNameEx(types.Unalias(t), true)
				return name + "$" + cc, pub
			}
		}
		return b.TypeNameEx(types.Unalias(t), publicFunc)
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
		// byte and rune are aliases for uint8 and int32. go/types may cache a
		// generic instance using either spelling, so ABI symbols must not
		// depend on which spelling was instantiated first.
		switch t.Kind() {
		case types.Byte:
			return types.Typ[types.Uint8].String()
		case types.Rune:
			return types.Typ[types.Int32].String()
		}
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
	ForTestMarker   = "@ForTest"
)

// PathOf returns the package path of the specified package.
func PathOf(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	if pkg.Name() == "main" && pkg.Scope().Lookup(ForTestMarker) == nil {
		return "main"
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

// ClosureName returns the ABI type name for the two-word closure descriptor
// of a Go function signature.
func (b *Builder) ClosureName(t *types.Signature) string {
	h := sha256.New()
	fmt.Fprintln(h, "struct", 2)
	fmt.Fprintln(h, "$f", b.FuncName(t))
	fmt.Fprintln(h, "$data", BasicName(types.Typ[types.UnsafePointer]))
	hashStr := base64.RawURLEncoding.EncodeToString(h.Sum(b.buf[:0]))
	return "_llgo_closure$" + hashStr
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
		ft, _ := b.TypeNameEx(v.Type(), true)
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
			pkg = m.Pkg().Path()
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
	if pkg != "" {
		return pkg + ".struct$" + hashStr, false
	}
	return "_llgo_struct$" + hashStr, false
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
