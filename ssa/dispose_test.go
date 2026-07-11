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

import "testing"

func TestProgramDispose(t *testing.T) {
	var nilProg Program
	nilProg.Dispose() // nil-safe

	prog := NewProgram(nil)
	prog.NewPackage("foo", "foo")
	prog.Dispose()
	if !prog.disposed {
		t.Fatal("Dispose should mark the program disposed")
	}
	prog.Dispose() // idempotent
}

func TestProgramDisposeDropsMayRecoverMarks(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("recovering", NoArgsNoRet, InGo)
	fn.Expr.MarkMayRecover()
	if !fn.Expr.mayRecover() {
		t.Fatal("marked function should be recover-capable")
	}

	ctx := prog.ctx
	prog.Dispose()
	if _, ok := mayRecoverPrograms.Load(ctx); ok {
		t.Fatal("disposed program remains in may-recover registry")
	}

	next := NewProgram(nil)
	defer next.Dispose()
	nextFn := next.NewPackage("bar", "bar").NewFunc("plain", NoArgsNoRet, InGo)
	if nextFn.Expr.mayRecover() {
		t.Fatal("may-recover mark leaked into a later program")
	}
}
