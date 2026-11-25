// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package frontcache

import (
	"fmt"
	"go/types"
	"golang.org/x/tools/go/ssa"
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
	return fmt.Sprintf("%q|%q|%q|%q|%q", typs.PackageID(Owner(fn)), fn.String(),
		typs.Key(fn.Signature), fn.Synthetic, Position(fn.Prog.Fset, fn.Pos()))
}

// Describe retains RTA's static edges, address-taken operands, dynamic-call
// signatures and interface materializations. Arithmetic and storage operations
// are intentionally absent: the record is an analysis contract, not cached SSA.
