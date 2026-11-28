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

package llvmgen

import (
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"strings"
	"testing"
)

func TestShiftCountCheckedBeforeNarrowing(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	pkg := prog.NewPackage("shiftcount", "shiftcount")

	for _, tt := range []struct {
		name string
		op   token.Token
		left *types.Basic
	}{
		{name: "signedLeft", op: token.SHL, left: types.Typ[types.Int]},
		{name: "signedRight", op: token.SHR, left: types.Typ[types.Int]},
		{name: "unsignedRight", op: token.SHR, left: types.Typ[types.Uint]},
	} {
		t.Run(tt.name, func(t *testing.T) {
			params := types.NewTuple(
				types.NewParam(0, nil, "x", tt.left),
				types.NewParam(0, nil, "count", types.Typ[types.Uint64]),
			)
			results := types.NewTuple(types.NewParam(0, nil, "", tt.left))
			fn := pkg.NewFunc(tt.name, types.NewSignatureType(nil, nil, nil, params, results, false), GoDecl)
			body := fn.MakeBody(1)
			body.Return(body.BinOp(tt.op, fn.Param(0), fn.Param(1)))
			ir := fn.impl.String()
			compareAt := strings.Index(ir, "icmp uge i64 %1, 32")
			truncateAt := strings.Index(ir, "trunc i64 %1 to i32")
			if compareAt < 0 || truncateAt < 0 || compareAt >= truncateAt {
				t.Fatalf("shift count %%1 must be compared as i64 before narrowing to i32:\n%s", ir)
			}
		})
	}
}
