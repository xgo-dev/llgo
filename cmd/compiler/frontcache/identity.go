// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package frontcache

import (
	"fmt"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func Owner(fn *ssa.Function) *types.Package {
	for f := fn; f != nil; f = f.Parent() {
		if f.Pkg != nil {
			return f.Pkg.Pkg
		}
		if obj := f.Object(); obj != nil && obj.Pkg() != nil {
			return obj.Pkg()
		}
	}
	return nil
}

func FunctionID(fn *ssa.Function, typs *Types) string {
	if fn == nil {
		return ""
	}
	synthetic := fn.Synthetic
	if synthetic == "from type information" {
		synthetic = ""
	}
	pos := ""
	if !isPositionIndependentFunction(fn, synthetic) {
		pos = Position(fn.Prog.Fset, fn.Pos())
	}
	return fmt.Sprintf("%q|%q|%q|%q|%q", typs.PackageID(Owner(fn)), fn.String(),
		typs.Key(fn.Signature), synthetic, pos)
}

func isPositionIndependentFunction(fn *ssa.Function, synthetic string) bool {
	if fn == nil || fn.Parent() != nil || fn.Name() == "_" {
		return false
	}
	if recv := fn.Signature.Recv(); recv != nil && isLocalNamedType(recv.Type()) {
		return false
	}
	if synthetic != "" {
		if len(fn.FreeVars) == 1 && isLocalNamedType(fn.FreeVars[0].Type()) {
			return false
		}
		return true
	}
	return fn.Object() != nil
}

func isLocalNamedType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	typ = types.Unalias(typ)
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(ptr.Elem())
	}
	named, ok := typ.(*types.Named)
	if !ok || named.Obj() == nil {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Parent() != nil && obj.Parent() != obj.Pkg().Scope()
}

// Describe retains RTA's static edges, address-taken operands, dynamic-call
// signatures and interface materializations. Arithmetic and storage operations
// are intentionally absent: the record is an analysis contract, not cached SSA.
