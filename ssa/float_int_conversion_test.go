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
	"go/types"
	"strings"
	"testing"
)

func TestAMD64FloatToIntegerConversionIR(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("floatconvert", "floatconvert")

	tests := []struct {
		name  string
		typ   *types.Basic
		wants []string
	}{
		{name: "I8", typ: types.Typ[types.Int8], wants: []string{"fptosi double", "to i32", "trunc i32", "to i8"}},
		{name: "I32", typ: types.Typ[types.Int32], wants: []string{"fcmp olt double", "fcmp oge double", "fptosi double", "i32 -2147483648"}},
		{name: "I64", typ: types.Typ[types.Int64], wants: []string{"fptosi double", "to i64", "i64 -9223372036854775808"}},
		{name: "U8", typ: types.Typ[types.Uint8], wants: []string{"fptosi double", "to i32", "trunc i32", "to i8"}},
		{name: "U32", typ: types.Typ[types.Uint32], wants: []string{"fptosi double", "to i64", "trunc i64", "to i32"}},
		{name: "U64", typ: types.Typ[types.Uint64], wants: []string{"fcmp oge double", "fsub double", "fptosi double", "or i64"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := types.NewTuple(types.NewParam(0, nil, "x", types.Typ[types.Float64]))
			results := types.NewTuple(types.NewParam(0, nil, "", tt.typ))
			sig := types.NewSignatureType(nil, nil, nil, params, results, false)
			fn := pkg.NewFunc(tt.name, sig, InGo)
			b := fn.MakeBody(1)
			b.Return(b.Convert(prog.Type(tt.typ, InGo), fn.Param(0)))

			ir := fn.impl.String()
			for _, want := range tt.wants {
				if !strings.Contains(ir, want) {
					t.Errorf("conversion IR missing %q:\n%s", want, ir)
				}
			}
		})
	}
}
