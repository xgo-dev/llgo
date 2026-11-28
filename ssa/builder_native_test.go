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
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/callgraph/rta"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
)

func buildNativeTestPkg(t *testing.T, src string, mode ssa.BuilderMode) (*ssa.Package, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	pkg := types.NewPackage("example.com/native", "p")
	ssapkg, info, err := ssautil.BuildPackage(&types.Config{}, fset, pkg, []*ast.File{file}, mode)
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	return ssapkg, info
}

func TestNativeASTToSSABuilderFeatures(t *testing.T) {
	const src = `package p

type Speaker interface {
	Speak() string
}

type Dog struct{ Name string }

func (d *Dog) Speak() string { return d.Name }

type Box[T any] struct{ V T }

func (b *Box[T]) Get() T { return b.V }

func Identity[T any](v T) T { return v }

func useClosure(base int) int {
	fn := func(x int) int { return base + x }
	return fn(5)
}

func useDeferRecover() (res int) {
	defer func() {
		if r := recover(); r != nil {
			res = 42
		}
	}()
	panic("boom")
}

func useSelect(c chan int) int {
	select {
	case v := <-c:
		return v
	default:
		return -1
	}
}

func main() {
	var s Speaker = &Dog{Name: "rex"}
	_ = s.Speak()
	b := &Box[int]{V: Identity(7)}
	_ = b.Get()
	_ = useClosure(3)
	_ = useDeferRecover()
	_ = useSelect(nil)
}
`
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics | ssa.GlobalDebug
	ssapkg, _ := buildNativeTestPkg(t, src, mode)

	mainFn := ssapkg.Func("main")
	if mainFn == nil || len(mainFn.Blocks) == 0 {
		t.Fatal("expected built main function")
	}
	if err := ssa.VerifyFunction(mainFn); err != nil {
		t.Fatalf("VerifyFunction(main): %v", err)
	}

	defRec := ssapkg.Func("useDeferRecover")
	if defRec == nil || defRec.Recover == nil {
		t.Fatalf("expected non-nil Recover block on useDeferRecover")
	}

	all := ssautil.AllFunctions(ssapkg.Prog)
	if len(all) < 6 {
		t.Fatalf("expected >= 6 functions in AllFunctions, got %d", len(all))
	}
	var foundInst bool
	for fn := range all {
		if strings.Contains(fn.String(), "Identity[int]") {
			foundInst = true
			if fn.Origin() == nil || len(fn.TypeArgs()) != 1 {
				t.Fatalf("generic instantiation missing origin/typeargs: %s", fn)
			}
		}
	}
	if !foundInst {
		t.Fatalf("expected instantiated generic Identity[int] in AllFunctions")
	}

	res := rta.Analyze([]*ssa.Function{mainFn}, true)
	if res == nil || res.CallGraph == nil {
		t.Fatal("expected non-nil RTA result with callgraph")
	}
	if _, ok := res.Reachable[defRec]; !ok {
		t.Fatalf("expected useDeferRecover reachable in RTA")
	}
	var sawSpeak bool
	for fn := range res.Reachable {
		if fn.Name() == "Speak" {
			sawSpeak = true
		}
	}
	if !sawSpeak {
		t.Fatalf("expected (*Dog).Speak reachable through interface dispatch in RTA")
	}
}

func TestNativeReturnLocalLoadOrder(t *testing.T) {
	const src = `package p

type value struct{ n int }

func (v *value) mutate() bool {
	v.n = 1
	return true
}

func f() (value, bool) {
	var v value
	return v, v.mutate()
}
`
	for _, tc := range []struct {
		name string
		mode ssa.BuilderMode
	}{
		{name: "default", mode: ssa.SanityCheckFunctions | ssa.InstantiateGenerics},
		{name: "global-debug", mode: ssa.SanityCheckFunctions | ssa.InstantiateGenerics | ssa.GlobalDebug},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ssapkg, _ := buildNativeTestPkg(t, src, tc.mode)
			fn := ssapkg.Func("f")
			if fn == nil {
				t.Fatal("missing f")
			}
			callIdx, loadIdx, debugRefIdx := -1, -1, -1
			var resultLoad *ssa.UnOp
			for _, b := range fn.Blocks {
				for i, ins := range b.Instrs {
					if c, ok := ins.(ssa.CallInstruction); ok {
						if callee := c.Common().StaticCallee(); callee != nil && callee.Name() == "mutate" {
							callIdx = i
						}
					}
					if u, ok := ins.(*ssa.UnOp); ok && u.Op == token.MUL {
						if _, ok := u.X.(*ssa.Alloc); ok {
							loadIdx = i
							resultLoad = u
						}
					}
					if dr, ok := ins.(*ssa.DebugRef); ok && resultLoad != nil && dr.X == resultLoad {
						debugRefIdx = i
					}
				}
			}
			if callIdx < 0 || loadIdx <= callIdx {
				t.Fatalf("expected load (%d) after mutate call (%d)\n%s", loadIdx, callIdx, fn)
			}
			if debugRefIdx >= 0 && debugRefIdx <= loadIdx {
				t.Fatalf("expected DebugRef (%d) after load (%d)\n%s", debugRefIdx, loadIdx, fn)
			}
		})
	}
}
