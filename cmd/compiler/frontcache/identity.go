// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package frontcache

import (
	"fmt"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
	"strings"
)

func Owner(fn *ssa.Function) *types.Package {
	for f := fn; f != nil; f = f.Parent() {
		if f.Pkg != nil {
			return f.Pkg.Pkg
		}
		if f.Signature != nil {
			if recv := f.Signature.Recv(); recv != nil {
				if pkg := receiverPkg(recv.Type()); pkg != nil {
					return pkg
				}
			}
			if strings.HasPrefix(f.Synthetic, "thunk for ") && f.Signature.Params().Len() > 0 {
				if pkg := receiverPkg(f.Signature.Params().At(0).Type()); pkg != nil {
					return pkg
				}
			}
		}
		if obj := f.Object(); obj != nil && obj.Pkg() != nil {
			return obj.Pkg()
		}
	}
	return nil
}

func receiverPkg(typ types.Type) *types.Package {
	if typ == nil {
		return nil
	}
	typ = types.Unalias(typ)
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(ptr.Elem())
	}
	if named, ok := typ.(*types.Named); ok && named.Obj() != nil {
		return named.Obj().Pkg()
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
	if localKey := localReceiverKey(fn, synthetic, typs); localKey != "" {
		// Synthetic wrappers for promoted methods on local types inherit
		// fn.Pos() from the embedded method's declaring package (which may
		// skip source parsing on warm builds). Key them by the receiver type's
		// stable TypeGraph identity instead of an Fset lookup.
		pos = localKey
	} else if !isPositionIndependentFunction(fn, synthetic) {
		pos = Position(fn.Prog.Fset, fn.Pos())
	}
	return fmt.Sprintf("%q|%q|%q|%q|%q", typs.PackageID(Owner(fn)), fn.String(),
		typs.Key(fn.Signature), synthetic, pos)
}

func localReceiverKey(fn *ssa.Function, synthetic string, typs *Types) string {
	if fn == nil || typs == nil {
		return ""
	}
	if fn.Signature != nil {
		if recv := fn.Signature.Recv(); recv != nil {
			if key := typs.Key(recv.Type()); strings.Contains(key, "local:") {
				return key
			}
		}
	}
	if synthetic != "" && len(fn.FreeVars) == 1 {
		if key := typs.Key(fn.FreeVars[0].Type()); strings.Contains(key, "local:") {
			return key
		}
	}
	return ""
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
