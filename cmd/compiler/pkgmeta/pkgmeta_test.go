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

package pkgmeta_test

import (
	"bytes"
	"go/constant"
	"go/token"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/pkgmeta"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestBuildMarshalUnmarshalAndDeadcode(t *testing.T) {
	prog := ssa.NewProgram(token.NewFileSet(), 0)
	goPkg := types.NewPackage("example.com/pm", "pm")
	pkg := prog.CreatePackage(goPkg, nil, nil, true)

	intPtr := types.NewPointer(types.Typ[types.Int])
	leakSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, goPkg, "p", intPtr)),
		types.NewTuple(types.NewVar(0, goPkg, "", intPtr)),
		false,
	)
	leakFn := prog.NewFunction("Leak", leakSig, "")
	pkg.AddFunction(leakFn, types.NewFunc(0, goPkg, "Leak", leakSig))
	p := leakFn.AddParam("p", nil, intPtr)
	leakFn.NewBasicBlock("entry").Emit(&ssa.Return{Results: []ssa.Value{p}}, nil, token.NoPos)

	deadSig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, goPkg, "", types.Typ[types.Int])), false)
	deadFn := prog.NewFunction("Dead", deadSig, "")
	pkg.AddFunction(deadFn, types.NewFunc(0, goPkg, "Dead", deadSig))
	deadFn.NewBasicBlock("entry").Emit(&ssa.Return{Results: []ssa.Value{ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])}}, nil, token.NoPos)

	prog.Build()

	meta := pkgmeta.BuildPackageMeta(pkg, nil, nil, pkgmeta.DefaultBudget)
	raw1, err := pkgmeta.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	raw2, err := pkgmeta.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal 2: %v", err)
	}
	if !bytes.Equal(raw1, raw2) {
		t.Fatal("Marshal output is not deterministic")
	}
	decoded, err := pkgmeta.Unmarshal(raw1)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.PkgPath != "example.com/pm" {
		t.Fatalf("PkgPath = %q", decoded.PkgPath)
	}
	dc := pkgmeta.AnalyzeDeadcode([]*pkgmeta.PackageMeta{decoded}, []string{"example.com/pm.Leak"})
	if !slices.Contains(dc.ReachableFuncs, "example.com/pm.Leak") || !slices.Contains(dc.DeadFuncs, "example.com/pm.Dead") {
		t.Fatalf("unexpected deadcode result: reachable=%v dead=%v", dc.ReachableFuncs, dc.DeadFuncs)
	}
}
