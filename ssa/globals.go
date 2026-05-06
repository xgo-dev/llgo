/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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
	"fmt"

	"github.com/xgo-dev/llvm"
)

func (pkg Package) AddGlobalString(name string, value string) {
	prog := pkg.Prog
	styp := prog.String()
	data := pkg.createGlobalStr(value)
	length := prog.IntVal(uint64(len(value)), prog.Uintptr())
	cv := llvm.ConstNamedStruct(styp.ll, []llvm.Value{data, length.impl})
	pkg.NewVarEx(name, prog.Pointer(styp)).Init(Expr{cv, styp})
}

// ConstString creates an SSA expression representing a Go string literal. The
// returned value is backed by an anonymous global constant and can be used to
// initialize package-level variables or other constant contexts that expect a
// Go string value.
func (pkg Package) ConstString(value string) Expr {
	prog := pkg.Prog
	styp := prog.String()
	data := pkg.createGlobalStr(value)
	length := prog.IntVal(uint64(len(value)), prog.Uintptr())
	cv := llvm.ConstNamedStruct(styp.ll, []llvm.Value{data, length.impl})
	return Expr{cv, styp}
}

// ConstBytes creates an SSA expression for a []byte backed by writable static data.
// Each call gets its own backing store, matching []byte mutability semantics.
func (pkg Package) ConstBytes(value []byte) Expr {
	prog := pkg.Prog
	styp := prog.Slice(prog.Byte())
	if len(value) == 0 {
		return prog.Zero(styp)
	}
	data := pkg.createGlobalBytes(value)
	n := prog.IntVal(uint64(len(value)), prog.Int())
	cv := llvm.ConstNamedStruct(styp.ll, []llvm.Value{data, n.impl, n.impl})
	return Expr{cv, styp}
}

// Undefined global string var by names
func (pkg Package) Undefined(names ...string) error {
	prog := pkg.Prog
	styp := prog.rtString()
	for _, name := range names {
		global := pkg.VarOf(name)
		if global == nil {
			continue
		}
		typ := prog.Elem(global.Type)
		if typ.ll != styp {
			return fmt.Errorf("%s: not a var of type string (type:%v)", name, typ.RawType())
		}
		newGlobal := llvm.AddGlobal(pkg.mod, styp, "")
		global.impl.ReplaceAllUsesWith(newGlobal)
		global.impl.EraseFromParentAsGlobal()
		newGlobal.SetName(name)
	}
	return nil
}
