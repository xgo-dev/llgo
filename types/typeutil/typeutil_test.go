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

package typeutil_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llgo/types/typeutil"
)

func TestMapAndLLGoTypeDistinction(t *testing.T) {
	var nilMap *typeutil.Map
	if nilMap.Len() != 0 || nilMap.At(types.Typ[types.Int]) != nil || nilMap.Delete(types.Typ[types.Int]) || nilMap.String() != "{}" || nilMap.Keys() != nil {
		t.Fatal("nil *Map behavior mismatch")
	}

	var m typeutil.Map
	m.SetHasher(typeutil.MakeHasher())

	goPtr := types.NewPointerWithKind(types.Typ[types.Int32], types.GoPtr)
	cPtr := types.NewPointerWithKind(types.Typ[types.Int32], types.CPtr)
	goSig := types.NewSignature(nil, nil, nil, false)
	cSig := types.NewSignatureWithCallConv(nil, nil, nil, false, types.CDecl)
	s1 := types.NewSlice(types.Typ[types.Int])
	s2 := types.NewSlice(types.Typ[types.Int]) // distinct pointer, Identical to s1

	m.Set(goPtr, "go")
	m.Set(cPtr, "c")
	m.Set(goSig, "gofn")
	m.Set(cSig, "cfn")
	m.Set(s1, 1)
	if prev := m.Set(s2, 2); prev != 1 {
		t.Fatalf("Set(identical slice) prev = %v, want 1", prev)
	}
	if m.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", m.Len())
	}
	if m.At(goPtr) != "go" || m.At(cPtr) != "c" || m.At(goSig) != "gofn" || m.At(cSig) != "cfn" || m.At(s1) != 2 {
		t.Fatalf("Map.At values mismatch: %s", m.String())
	}
	if len(m.Keys()) != 5 || !strings.Contains(m.KeysString(), "[]int") || !strings.Contains(m.String(), "cfn") {
		t.Fatalf("Keys/String mismatch: keys=%v str=%s", m.Keys(), m.String())
	}
	if !m.Delete(cPtr) || m.Delete(cPtr) || m.At(cPtr) != nil || m.Len() != 4 {
		t.Fatal("Delete failed")
	}
}

func TestMethodSetCacheIntuitiveDependenciesAndCallee(t *testing.T) {
	src := `package demo

type Reader interface { Read() int }
type Box struct{ X int }
func (b Box) Read() int { return b.X }
func (b *Box) Reset() { b.X = 0 }

func Helper[T any](x T) T { return x }

func Use(r Reader, b Box, fn func()) {
	_ = r.Read()
	_ = b.Read()
	b.Reset()
	_ = Helper[int](1)
	_ = int(1)
	_ = len("hi")
	fn()
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "demo.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	pkg, err := (&types.Config{}).Check("example.com/demo", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	box := pkg.Scope().Lookup("Box").Type()
	ptrBox := types.NewPointer(box)
	reader := pkg.Scope().Lookup("Reader").Type()

	var cache typeutil.MethodSetCache
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cache.MethodSet(box).Len() != 1 || cache.MethodSet(ptrBox).Len() != 2 || cache.MethodSet(reader).Len() != 1 {
				t.Errorf("MethodSetCache mismatch")
			}
			_ = cache.MethodSet(types.NewSlice(box))
		}()
	}
	wg.Wait()

	if got := typeutil.IntuitiveMethodSet(box, &cache); len(got) != 2 {
		t.Fatalf("IntuitiveMethodSet(Box) len = %d, want 2", len(got))
	}
	if got := typeutil.IntuitiveMethodSet(ptrBox, nil); len(got) != 2 {
		t.Fatalf("IntuitiveMethodSet(*Box, nil) len = %d, want 2", len(got))
	}
	if got := typeutil.IntuitiveMethodSet(reader, &cache); len(got) != 1 {
		t.Fatalf("IntuitiveMethodSet(Reader) len = %d, want 1", len(got))
	}

	// Dependencies topological order.
	a := types.NewPackage("a", "a")
	b := types.NewPackage("b", "b")
	c := types.NewPackage("c", "c")
	b.SetImports([]*types.Package{a})
	c.SetImports([]*types.Package{b, a})
	deps := typeutil.Dependencies(c)
	if len(deps) != 3 || deps[0] != a || deps[1] != b || deps[2] != c {
		t.Fatalf("Dependencies = %v, want [a b c]", deps)
	}

	// Callee & StaticCallee across interface call, value/ptr method, generic call, conversion, builtin, var call.
	var callees []string
	var statics []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if obj := typeutil.Callee(info, call); obj != nil {
			callees = append(callees, obj.Name())
		}
		if fn := typeutil.StaticCallee(info, call); fn != nil {
			statics = append(statics, fn.Name())
		}
		return true
	})
	if len(callees) != 6 {
		t.Fatalf("Callee count = %d (%v), want 6 (Read, Read, Reset, Helper, len, fn)", len(callees), callees)
	}
	if len(statics) != 3 {
		t.Fatalf("StaticCallee count = %d (%v), want 3 (Read, Reset, Helper)", len(statics), statics)
	}
}

func TestMethodSetCacheCanonicalizesAliasReceivers(t *testing.T) {
	pkgA := types.NewPackage("example.com/a", "a")
	pkgB := types.NewPackage("example.com/b", "b")

	baseObj := types.NewTypeName(token.NoPos, pkgA, "Base", nil)
	baseNamed := types.NewNamed(baseObj, types.NewStruct(nil, nil), nil)
	sig := types.NewSignatureType(types.NewVar(token.NoPos, pkgA, "b", types.NewPointer(baseNamed)), nil, nil, nil, nil, false)
	baseNamed.AddMethod(types.NewFunc(token.NoPos, pkgA, "Ping", sig))
	pkgA.Scope().Insert(baseObj)

	aliasObj := types.NewTypeName(token.NoPos, pkgB, "Alias", nil)
	aliasType := types.NewAlias(aliasObj, baseNamed)
	pkgB.Scope().Insert(aliasObj)

	var cache typeutil.MethodSetCache
	// Query pointer-to-alias BEFORE pointer-to-named; Selection.Recv must still
	// be canonical *a.Base (matching x/tools typeutil.MethodSetCache).
	ms := cache.MethodSet(types.NewPointer(aliasType))
	sel := ms.Lookup(nil, "Ping")
	if sel == nil {
		t.Fatal("missing Ping method on *Alias")
	}
	if got := types.TypeString(sel.Recv(), nil); got != "*example.com/a.Base" {
		t.Fatalf("MethodSet(*Alias).Recv() = %q, want %q", got, "*example.com/a.Base")
	}
	// Value alias too.
	valSig := types.NewSignatureType(types.NewVar(token.NoPos, pkgA, "b", baseNamed), nil, nil, nil, nil, false)
	baseNamed.AddMethod(types.NewFunc(token.NoPos, pkgA, "Val", valSig))
	var cache2 typeutil.MethodSetCache
	msVal := cache2.MethodSet(aliasType)
	selVal := msVal.Lookup(nil, "Val")
	if selVal == nil {
		t.Fatal("missing Val method on Alias")
	}
	if got := types.TypeString(selVal.Recv(), nil); got != "example.com/a.Base" {
		t.Fatalf("MethodSet(Alias).Recv() = %q, want %q", got, "example.com/a.Base")
	}
}
