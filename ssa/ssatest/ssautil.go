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

package ssatest

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/goplus/llgo/ssa"
)

func NewProgram(t *testing.T, target *ssa.Target) ssa.Program {
	fset := token.NewFileSet()
	imp := packages.NewImporter(fset)
	return NewProgramEx(t, target, imp)
}

func NewProgramEx(t *testing.T, target *ssa.Target, imp types.Importer) ssa.Program {
	prog := ssa.NewProgram(target)
	prog.SetRuntime(func() *types.Package {
		rt, err := imp.Import(ssa.PkgRuntime)
		if err != nil {
			t.Fatal("load runtime failed:", err)
		}
		return rt
	})
	prog.SetPython(func() *types.Package {
		rt, err := imp.Import(ssa.PkgPython)
		if err != nil {
			t.Fatal("load python failed:", err)
		}
		return rt
	})
	return prog
}

func Assert(t *testing.T, p ssa.Package, expected string) {
	t.Helper()
	if v := p.String(); ssa.StripModuleTarget(v) != expected {
		t.Fatalf("\n==> got:\n%s\n==> expected:\n%s\n", v, expected)
	}
}
