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

package ssa

import (
	"go/token"
	"go/types"

	"github.com/goplus/llgo/internal/metadata"
	"github.com/goplus/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

func (b Builder) recordTypeChildren(parentName string, t types.Type) {
	mb := b.Pkg.MetaBuilder
	if mb == nil {
		return
	}
	parent := mb.Symbol(parentName)
	for _, child := range b.directTypeChildren(t) {
		childName, _ := b.Prog.abi.TypeName(child)
		mb.AddTypeChild(parent, mb.Symbol(childName))
	}
}

func (b Builder) directTypeChildren(t types.Type) []types.Type {
	switch t := types.Unalias(t).(type) {
	case *types.Basic:
		return nil
	case *types.Pointer:
		return []types.Type{abi.PublicType(t.Elem())}
	case *types.Chan:
		return []types.Type{abi.PublicType(t.Elem())}
	case *types.Slice:
		return []types.Type{abi.PublicType(t.Elem())}
	case *types.Array:
		elem := abi.PublicType(t.Elem())
		return []types.Type{elem, types.NewSlice(elem)}
	case *types.Map:
		return []types.Type{
			abi.PublicType(t.Key()),
			abi.PublicType(t.Elem()),
			// Map type descriptors reference their runtime bucket type too.
			b.Prog.abi.MapBucket(t),
		}
	case *types.Signature:
		var children []types.Type
		children = appendTupleTypeChildren(children, t.Params())
		children = appendTupleTypeChildren(children, t.Results())
		return children
	case *types.Struct:
		children := make([]types.Type, 0, t.NumFields())
		for i := 0; i < t.NumFields(); i++ {
			children = append(children, abi.PublicType(t.Field(i).Type()))
		}
		return children
	case *types.Interface:
		children := make([]types.Type, 0, t.NumMethods())
		for i := 0; i < t.NumMethods(); i++ {
			children = append(children, funcType(b.Prog, t.Method(i).Type()))
		}
		return children
	case *types.Named:
		return b.directTypeChildren(t.Underlying())
	}
	return nil
}

func appendTupleTypeChildren(children []types.Type, tuple *types.Tuple) []types.Type {
	if tuple == nil {
		return children
	}
	for i := 0; i < tuple.Len(); i++ {
		children = append(children, abi.PublicType(tuple.At(i).Type()))
	}
	return children
}

func (b Builder) recordInterfaceMethodCall(intf Expr, rawIntf *types.Interface, method *types.Func) {
	mb := b.Pkg.MetaBuilder
	if mb == nil {
		return
	}
	intfTypeName, _ := b.Prog.abi.TypeName(intf.raw.Type)
	mtypeName, _ := b.Prog.abi.TypeName(funcType(b.Prog, method.Type()))
	mb.AddUseIfaceMethod(mb.Symbol(b.Func.Name()), []metadata.IfaceMethodDemand{{
		Target: mb.Symbol(intfTypeName),
		Sig: metadata.MethodSig{
			Name:  mb.Name(mthName(method)),
			MType: mb.Symbol(mtypeName),
		},
	}})

	methods := make([]metadata.MethodSig, 0, rawIntf.NumMethods())
	for i := 0; i < rawIntf.NumMethods(); i++ {
		im := rawIntf.Method(i)
		imtypeName, _ := b.Prog.abi.TypeName(funcType(b.Prog, im.Type()))
		methods = append(methods, metadata.MethodSig{
			Name:  mb.Name(mthName(im)),
			MType: mb.Symbol(imtypeName),
		})
	}
	mb.AddIfaceEntry(mb.Symbol(intfTypeName), methods)
}

func (b Builder) recordInterfaceUse(t types.Type) {
	mb := b.Pkg.MetaBuilder
	if mb == nil {
		return
	}
	if _, ok := types.Unalias(t).Underlying().(*types.Interface); ok {
		return
	}
	typeName, _ := b.Prog.abi.TypeName(t)
	mb.AddUseIface(mb.Symbol(b.Func.Name()), []metadata.Symbol{mb.Symbol(typeName)})
}

type methodInfoRecorder struct {
	b       Builder
	mb      *metadata.Builder
	typeSym metadata.Symbol
	slots   []metadata.MethodSlot
}

func (b Builder) newMethodInfoRecorder(t types.Type, n int) methodInfoRecorder {
	mb := b.Pkg.MetaBuilder
	if mb == nil {
		return methodInfoRecorder{}
	}
	typeName, _ := b.Prog.abi.TypeName(t)
	return methodInfoRecorder{
		b:       b,
		mb:      mb,
		typeSym: mb.Symbol(typeName),
		slots:   make([]metadata.MethodSlot, 0, n),
	}
}

func (r *methodInfoRecorder) add(method *types.Func, ftyp types.Type, ifn, tfn llvm.Value) {
	if r.mb == nil {
		return
	}
	mtypeName, _ := r.b.Prog.abi.TypeName(ftyp)
	r.slots = append(r.slots, metadata.MethodSlot{
		Sig: metadata.MethodSig{
			Name:  r.mb.Name(mthName(method)),
			MType: r.mb.Symbol(mtypeName),
		},
		IFn: r.mb.Symbol(ifn.Name()),
		TFn: r.mb.Symbol(tfn.Name()),
	})
}

func (r *methodInfoRecorder) finish() {
	if r.mb == nil {
		return
	}
	r.mb.AddMethodInfo(r.typeSym, r.slots)
}

func mthName(method *types.Func) string {
	name := method.Name()
	if token.IsExported(name) {
		return name
	}
	return abi.FullName(method.Pkg(), name)
}
