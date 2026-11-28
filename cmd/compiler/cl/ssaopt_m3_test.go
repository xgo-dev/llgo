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

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llvm"
)

func compileSSAOptIR(t *testing.T, src string, level optlevel.Level, target *llvmgen.Target) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "ssaoptm3.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	prog := newLLSSAProgForTarget(t, target)
	defer prog.Dispose()
	if target == nil {
	}
	typeCtx := prog.TypeContext()
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Context: typeCtx, Sizes: typeCtx, Importer: packages.NewImporter(fset)},
		fset,
		types.NewPackage(file.Name.Name, file.Name.Name),
		files,
		ssa.SanityCheckFunctions,
	)
	if err != nil {
		t.Fatal(err)
	}
	compiled, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, pkg, files, nil, false, Options{
		OptLevel: level,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	return compiled.String()
}

func TestSSAOptM3BoundsIfaceDevirtIR(t *testing.T) {
	const src = `package ssaoptm3

type Reader interface {
	Read() int
}

type ReadCloser interface {
	Read() int
	Close()
}

type Counter struct {
	N int
}

//go:noinline
func (c *Counter) Read() int { return c.N }

//go:noinline
func (c *Counter) Close() {}

func sumLoop(s []int) int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += s[i]
	}
	return total
}

func readArrPtr(p *[4]int) int {
	return p[0]
}

func changeIface(c *Counter) Reader {
	var rc ReadCloser = c
	var rd Reader = rc
	return rd
}

func assertKnown(n int) int {
	var a any = n
	return a.(int)
}

func devirtDirect(c *Counter) int {
	var rd Reader = c
	return rd.Read()
}

func devirtPhiMaybeNil(cond bool, c *Counter) int {
	var rd Reader
	if cond {
		rd = c
	}
	return rd.Read()
}
`
	irO0 := compileSSAOptIR(t, src, optlevel.O0, nil)
	irO2 := compileSSAOptIR(t, src, optlevel.O2, nil)

	// 1. BCE: sumLoop retains PanicIndex at -O0 and eliminates it at -O2.
	sumO0 := llvmFunction(t, irO0, "ssaoptm3.sumLoop")
	sumO2 := llvmFunction(t, irO2, "ssaoptm3.sumLoop")
	if !strings.Contains(sumO0, "PanicIndex") {
		t.Fatalf("-O0 sumLoop should retain PanicIndex:\n%s", sumO0)
	}
	if strings.Contains(sumO2, "PanicIndex") {
		t.Fatalf("-O2 sumLoop should eliminate PanicIndex:\n%s", sumO2)
	}

	// 2. *array nil dereference check MUST be preserved even when bounds check is elided.
	arrO2 := llvmFunction(t, irO2, "ssaoptm3.readArrPtr")
	if strings.Contains(arrO2, "PanicIndex") {
		t.Fatalf("-O2 readArrPtr should not emit PanicIndex for constant index 0:\n%s", arrO2)
	}
	if !strings.Contains(arrO2, "AssertNilDeref") {
		t.Fatalf("-O2 readArrPtr lost *array AssertNilDeref:\n%s", arrO2)
	}

	// 3. ChangeInterface from known concrete uses static itab at -O2 without NewItab/IfaceType.
	chgO0 := llvmFunction(t, irO0, "ssaoptm3.changeIface")
	chgO2 := llvmFunction(t, irO2, "ssaoptm3.changeIface")
	if !strings.Contains(chgO0, "NewItab") {
		t.Fatalf("-O0 changeIface expected runtime NewItab:\n%s", chgO0)
	}
	if strings.Contains(chgO2, "NewItab") || strings.Contains(chgO2, "IfaceType") {
		t.Fatalf("-O2 changeIface should avoid runtime NewItab/IfaceType:\n%s", chgO2)
	}
	if !strings.Contains(chgO2, "_llgo_itab$") {
		t.Fatalf("-O2 changeIface expected static itab reference:\n%s", chgO2)
	}

	// 4. TypeAssert on known concrete eliminates PanicTypeAssert at -O2.
	taO0 := llvmFunction(t, irO0, "ssaoptm3.assertKnown")
	taO2 := llvmFunction(t, irO2, "ssaoptm3.assertKnown")
	if !strings.Contains(taO0, "PanicTypeAssert") {
		t.Fatalf("-O0 assertKnown expected PanicTypeAssert:\n%s", taO0)
	}
	if strings.Contains(taO2, "PanicTypeAssert") {
		t.Fatalf("-O2 assertKnown should fold away PanicTypeAssert:\n%s", taO2)
	}

	// 5. Tier-1 devirtualization: direct call at -O2, IfacePtrData at -O0.
	dvO0 := llvmFunction(t, irO0, "ssaoptm3.devirtDirect")
	dvO2 := llvmFunction(t, irO2, "ssaoptm3.devirtDirect")
	if !strings.Contains(dvO0, "IfacePtrData") {
		t.Fatalf("-O0 devirtDirect expected dynamic interface dispatch:\n%s", dvO0)
	}
	if strings.Contains(dvO2, "IfacePtrData") {
		t.Fatalf("-O2 devirtDirect should not use dynamic IfacePtrData:\n%s", dvO2)
	}
	if !strings.Contains(dvO2, `"ssaoptm3.(*Counter).Read"`) {
		t.Fatalf("-O2 devirtDirect expected direct call to (*Counter).Read:\n%s", dvO2)
	}

	// 6. Tier-1 Phi maybe-nil devirtualization: direct call + AssertNilDeref preserved.
	dvPhiO2 := llvmFunction(t, irO2, "ssaoptm3.devirtPhiMaybeNil")
	if strings.Contains(dvPhiO2, "IfacePtrData") {
		t.Fatalf("-O2 devirtPhiMaybeNil should devirtualize:\n%s", dvPhiO2)
	}
	if !strings.Contains(dvPhiO2, `"ssaoptm3.(*Counter).Read"`) || !strings.Contains(dvPhiO2, "AssertNilDeref") {
		t.Fatalf("-O2 devirtPhiMaybeNil must emit direct call AND AssertNilDeref:\n%s", dvPhiO2)
	}
}
