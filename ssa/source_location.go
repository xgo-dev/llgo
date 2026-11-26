// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package ssa

import (
	"go/constant"
	"go/token"

	"github.com/xgo-dev/llgo/internal/directive"
)

// SourceLocation lowers a compiler variable read without allocating variable
// storage. The caller supplies the adjusted position of that particular read.
func (b Builder) SourceLocation(kind directive.VariableKind, pos token.Position, typ Type) Expr {
	if !pos.IsValid() {
		panic("compiler variable read has no source position")
	}
	switch kind {
	case directive.LineVariable:
		return b.Const(constant.MakeInt64(int64(pos.Line)), typ)
	case directive.FileVariable:
		// Reuse the module's private constant pool, including the NUL terminator.
		return Expr{b.Pkg.createGlobalStr(pos.Filename + "\x00"), typ}
	default:
		panic("invalid source location variable")
	}
}
