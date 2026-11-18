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

package ssa_test

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
)

func TestWideMakeOperandsUseCheckedHelpersOn386(t *testing.T) {
	t.Setenv("GOOS", "windows")
	t.Setenv("GOARCH", "386")
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "windows", GOARCH: "386"})
	t.Cleanup(prog.Dispose)

	params := types.NewTuple(
		types.NewVar(0, nil, "length", types.Typ[types.Int64]),
		types.NewVar(0, nil, "capacity", types.Typ[types.Uint64]),
		types.NewVar(0, nil, "narrowSigned", types.Typ[types.Int32]),
		types.NewVar(0, nil, "narrowUnsigned", types.Typ[types.Uint32]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	pkg := prog.NewPackage("wide-make", "example.com/wide-make")
	fn := pkg.NewFunc("makeWide", sig, ssa.GoDecl)
	b := fn.MakeBody(1)
	b.MakeSlice(prog.Type(types.NewSlice(types.Typ[types.Byte]), ssa.GoDecl), fn.Param(0), fn.Param(1))
	b.MakeSlice(prog.Type(types.NewSlice(types.Typ[types.Byte]), ssa.GoDecl), fn.Param(0), fn.Param(2))
	b.MakeSlice(prog.Type(types.NewSlice(types.Typ[types.Byte]), ssa.GoDecl), fn.Param(0), fn.Param(3))
	b.MakeSliceEx(prog.Type(types.NewSlice(types.Typ[types.Byte]), ssa.GoDecl), fn.Param(0), fn.Param(1), false)
	b.MakeChan(prog.Type(types.NewChan(types.SendRecv, types.Typ[types.Byte]), ssa.GoDecl), fn.Param(1))
	b.Return()
	b.EndBuild()

	ir := pkg.String()
	for _, helper := range []string{"MakeSlice64", "MakeSlice64Buffer", "NewChan64"} {
		if !strings.Contains(ir, helper) {
			t.Errorf("386 wide make IR does not call %s:\n%s", helper, ir)
		}
	}
	for _, conversion := range []string{"sext i32 %2 to i64", "zext i32 %3 to i64"} {
		if !strings.Contains(ir, conversion) {
			t.Errorf("386 mixed-width make IR does not contain %q:\n%s", conversion, ir)
		}
	}
}
