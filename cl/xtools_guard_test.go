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
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/types/typeutil"
)

// Compile-time assertions on the exact golang.org/x/tools surfaces depended on
// by LLGo (go/ssa, go/packages, go/ssa/ssautil, go/types/typeutil,
// go/callgraph/rta, go/ast/astutil). If an x/tools upgrade breaks or removes
// any of these APIs, compilation fails immediately.
var (
	_ []ssa.Instruction = []ssa.Instruction{
		(*ssa.Alloc)(nil),
		(*ssa.BinOp)(nil),
		(*ssa.Call)(nil),
		(*ssa.ChangeInterface)(nil),
		(*ssa.ChangeType)(nil),
		(*ssa.Convert)(nil),
		(*ssa.DebugRef)(nil),
		(*ssa.Defer)(nil),
		(*ssa.Extract)(nil),
		(*ssa.Field)(nil),
		(*ssa.FieldAddr)(nil),
		(*ssa.Go)(nil),
		(*ssa.If)(nil),
		(*ssa.Index)(nil),
		(*ssa.IndexAddr)(nil),
		(*ssa.Jump)(nil),
		(*ssa.Lookup)(nil),
		(*ssa.MakeChan)(nil),
		(*ssa.MakeClosure)(nil),
		(*ssa.MakeInterface)(nil),
		(*ssa.MakeMap)(nil),
		(*ssa.MakeSlice)(nil),
		(*ssa.MapUpdate)(nil),
		(*ssa.MultiConvert)(nil),
		(*ssa.Next)(nil),
		(*ssa.Panic)(nil),
		(*ssa.Phi)(nil),
		(*ssa.Range)(nil),
		(*ssa.Return)(nil),
		(*ssa.RunDefers)(nil),
		(*ssa.Select)(nil),
		(*ssa.Send)(nil),
		(*ssa.Slice)(nil),
		(*ssa.SliceToArrayPointer)(nil),
		(*ssa.Store)(nil),
		(*ssa.TypeAssert)(nil),
		(*ssa.UnOp)(nil),
	}
	_ []ssa.Value = []ssa.Value{
		(*ssa.Builtin)(nil),
		(*ssa.Const)(nil),
		(*ssa.FreeVar)(nil),
		(*ssa.Function)(nil),
		(*ssa.Global)(nil),
		(*ssa.Parameter)(nil),
	}
	_ []ssa.Member = []ssa.Member{
		(*ssa.Function)(nil),
		(*ssa.Global)(nil),
		(*ssa.NamedConst)(nil),
		(*ssa.Type)(nil),
	}
	_ ssa.BuilderMode = ssa.SanityCheckFunctions | ssa.GlobalDebug | ssa.InstantiateGenerics

	// golang.org/x/tools/go/packages surface relied on by internal/packages & internal/build.
	_ packages.LoadMode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
		packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax |
		packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedModule
	_ func(*packages.Config, ...string) ([]*packages.Package, error) = packages.Load

	// golang.org/x/tools/go/ssa/ssautil surface relied on by cl/ & internal/build.
	_ func(*ssa.Program) map[*ssa.Function]bool                 = ssautil.AllFunctions
	_ func([]*ssa.Package) []*ssa.Package                       = ssautil.MainPackages
	_ func([]*packages.Package, ssa.BuilderMode) (*ssa.Program, []*ssa.Package) = ssautil.Packages

	// golang.org/x/tools/go/types/typeutil surface relied on by internal/llvmgen & internal/build.
	_ *typeutil.Map            = (*typeutil.Map)(nil)
	_ *typeutil.MethodSetCache = (*typeutil.MethodSetCache)(nil)
	_                          = typeutil.IntuitiveMethodSet
	_                          = typeutil.Callee
	_                          = typeutil.StaticCallee

	// golang.org/x/tools/go/callgraph/rta surface relied on by internal/build/wasm_reflect.go.
	_ func([]*ssa.Function, bool) *rta.Result = rta.Analyze

	// golang.org/x/tools/go/ast/astutil surface relied on by internal/build & internal/midway.
	_ func(ast.Node, astutil.ApplyFunc, astutil.ApplyFunc) ast.Node = astutil.Apply
)

// TestXToolsSSAVersionGuard inspects the active golang.org/x/tools/go/ssa
// package from go.mod AND cross-checks cl/compile.go's instruction handling so
// any future x/tools version bump that introduces a new concrete ssa.Instruction,
// ssa.Value, or ssa.Member (or drops handling in cl/compile.go) fails immediately.
func TestXToolsSSAVersionGuard(t *testing.T) {
	out, err := exec.Command("go", "list", "-json", "golang.org/x/tools/go/ssa").Output()
	if err != nil {
		t.Fatalf("go list golang.org/x/tools/go/ssa: %v", err)
	}
	var pkgInfo struct {
		Dir     string
		GoFiles []string
	}
	if err := json.Unmarshal(out, &pkgInfo); err != nil {
		t.Fatalf("unmarshal go list: %v", err)
	}

	type structMeta struct {
		exported bool
		embedded []string
		methods  map[string]bool
	}
	structs := make(map[string]*structMeta)
	ensure := func(name string) *structMeta {
		m := structs[name]
		if m == nil {
			m = &structMeta{exported: ast.IsExported(name), methods: make(map[string]bool)}
			structs[name] = m
		}
		return m
	}

	fset := token.NewFileSet()
	for _, name := range pkgInfo.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(pkgInfo.Dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if st, isStruct := ts.Type.(*ast.StructType); isStruct {
						meta := ensure(ts.Name.Name)
						if st.Fields != nil {
							for _, field := range st.Fields.List {
								if len(field.Names) != 0 {
									continue
								}
								ft := field.Type
								if star, ok := ft.(*ast.StarExpr); ok {
									ft = star.X
								}
								if id, ok := ft.(*ast.Ident); ok {
									meta.embedded = append(meta.embedded, id.Name)
								}
							}
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || len(d.Recv.List) == 0 {
					continue
				}
				r := d.Recv.List[0].Type
				if star, ok := r.(*ast.StarExpr); ok {
					r = star.X
				}
				if id, ok := r.(*ast.Ident); ok {
					ensure(id.Name).methods[d.Name.Name] = true
				}
			}
		}
	}

	var hasMethod func(tname, mname string, seen map[string]bool) bool
	hasMethod = func(tname, mname string, seen map[string]bool) bool {
		if seen[tname] {
			return false
		}
		seen[tname] = true
		meta := structs[tname]
		if meta == nil {
			return false
		}
		if meta.methods[mname] {
			return true
		}
		for _, emb := range meta.embedded {
			if hasMethod(emb, mname, seen) {
				return true
			}
		}
		return false
	}

	actualInstrs := make(map[string]bool)
	actualNonInstrValues := make(map[string]bool)
	actualMembers := make(map[string]bool)
	for tname, meta := range structs {
		if !meta.exported {
			continue
		}
		isInstr := hasMethod(tname, "Block", map[string]bool{}) && hasMethod(tname, "Operands", map[string]bool{})
		isVal := hasMethod(tname, "Referrers", map[string]bool{}) && hasMethod(tname, "Type", map[string]bool{})
		isMember := hasMethod(tname, "Package", map[string]bool{}) && hasMethod(tname, "Token", map[string]bool{})
		if isInstr {
			actualInstrs[tname] = true
		} else if isVal {
			actualNonInstrValues[tname] = true
		}
		if isMember {
			actualMembers[tname] = true
		}
	}

	knownInstrs := map[string]bool{
		"Alloc": true, "BinOp": true, "Call": true, "ChangeInterface": true,
		"ChangeType": true, "Convert": true, "DebugRef": true, "Defer": true,
		"Extract": true, "Field": true, "FieldAddr": true, "Go": true,
		"If": true, "Index": true, "IndexAddr": true, "Jump": true,
		"Lookup": true, "MakeChan": true, "MakeClosure": true, "MakeInterface": true,
		"MakeMap": true, "MakeSlice": true, "MapUpdate": true, "MultiConvert": true,
		"Next": true, "Panic": true, "Phi": true, "Range": true,
		"Return": true, "RunDefers": true, "Select": true, "Send": true,
		"Slice": true, "SliceToArrayPointer": true, "Store": true, "TypeAssert": true,
		"UnOp": true,
	}
	knownNonInstrValues := map[string]bool{
		"Builtin": true, "Const": true, "FreeVar": true,
		"Function": true, "Global": true, "Parameter": true,
	}
	knownMembers := map[string]bool{
		"Function": true, "Global": true, "NamedConst": true, "Type": true,
	}

	compareSets(t, "x/tools/go/ssa concrete Instruction types (update cl/compile.go and llgo/ssa IR if x/tools added a new op)", knownInstrs, actualInstrs)
	compareSets(t, "x/tools/go/ssa concrete non-Instruction Value types (update cl/compile.go and llgo/ssa IR if x/tools changed)", knownNonInstrValues, actualNonInstrValues)
	compareSets(t, "x/tools/go/ssa concrete Member types", knownMembers, actualMembers)

	// Cross-check cl/compile.go actually handles every known x/tools instruction
	// except explicitly justified omissions (e.g. MultiConvert when generics are instantiated).
	t.Run("CLCompileInstructionCoverage", func(t *testing.T) {
		cf, err := parser.ParseFile(token.NewFileSet(), "compile.go", nil, 0)
		if err != nil {
			t.Fatalf("parse cl/compile.go: %v", err)
		}
		handled := make(map[string]bool)
		ast.Inspect(cf, func(n ast.Node) bool {
			// Match `case *ssa.<Name>:` in type switches and type assertions `.(*ssa.<Name>)`.
			switch x := n.(type) {
			case *ast.CaseClause:
				for _, expr := range x.List {
					if name := ssaPtrTypeName(expr); name != "" {
						handled[name] = true
					}
				}
			case *ast.TypeAssertExpr:
				if name := ssaPtrTypeName(x.Type); name != "" {
					handled[name] = true
				}
			}
			return true
		})

		// Instructions intentionally not handled directly by cl/compile.go.
		knownUnhandled := map[string]string{
			"MultiConvert": "only emitted when ssa.InstantiateGenerics is disabled; LLGo enables ssa.InstantiateGenerics",
		}

		var missing, staleUnhandled []string
		for instr := range actualInstrs {
			if handled[instr] {
				if _, listed := knownUnhandled[instr]; listed {
					staleUnhandled = append(staleUnhandled, instr)
				}
				continue
			}
			if _, ok := knownUnhandled[instr]; !ok {
				missing = append(missing, instr)
			}
		}
		for instr := range knownUnhandled {
			if !actualInstrs[instr] {
				staleUnhandled = append(staleUnhandled, instr)
			}
		}
		sort.Strings(missing)
		sort.Strings(staleUnhandled)
		if len(missing) > 0 {
			t.Errorf("cl/compile.go does not handle x/tools/go/ssa instructions (implement or justify in knownUnhandled): %v", missing)
		}
		if len(staleUnhandled) > 0 {
			t.Errorf("stale entries in knownUnhandled (now handled or removed): %v", staleUnhandled)
		}
	})
}

func ssaPtrTypeName(expr ast.Expr) string {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return ""
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok || id.Name != "ssa" {
		return ""
	}
	return sel.Sel.Name
}

// TestPureGoCoreDependencyIsolation guarantees the Phase 1 pure-Go core packages
// remain permanently isolated from golang.org/x/tools, github.com/xgo-dev/llvm,
// CGo, and llgo/internal/*.
func TestPureGoCoreDependencyIsolation(t *testing.T) {
	pkgs := []string{
		"github.com/xgo-dev/llgo/directive",
		"github.com/xgo-dev/llgo/target",
		"github.com/xgo-dev/llgo/types",
		"github.com/xgo-dev/llgo/ssa",
	}
	args := append([]string{"list", "-json"}, pkgs...)
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		t.Fatalf("go list core packages: %v", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	seen := 0
	for dec.More() {
		var p struct {
			ImportPath string
			Imports    []string
			Deps       []string
		}
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("decode go list: %v", err)
		}
		seen++
		all := append(append([]string(nil), p.Imports...), p.Deps...)
		for _, dep := range all {
			if strings.HasPrefix(dep, "golang.org/x/tools") ||
				strings.HasPrefix(dep, "github.com/xgo-dev/llvm") ||
				strings.HasPrefix(dep, "github.com/xgo-dev/llgo/internal") ||
				dep == "C" || dep == "runtime/cgo" {
				t.Errorf("%s must not depend on %s", p.ImportPath, dep)
			}
			if (p.ImportPath == "github.com/xgo-dev/llgo/directive" ||
				p.ImportPath == "github.com/xgo-dev/llgo/target") && dep == "go/types" {
				t.Errorf("%s must not depend on go/types", p.ImportPath)
			}
		}
		if p.ImportPath == "github.com/xgo-dev/llgo/ssa" {
			for _, imp := range p.Imports {
				if imp == "go/types" {
					t.Errorf("llgo/ssa must not directly import go/types (use llgo/types)")
				}
			}
		}
	}
	if seen != len(pkgs) {
		t.Fatalf("expected %d core packages, decoded %d", len(pkgs), seen)
	}
}

func compareSets(t *testing.T, label string, known, actual map[string]bool) {
	t.Helper()
	var added, removed []string
	for k := range actual {
		if !known[k] {
			added = append(added, k)
		}
	}
	for k := range known {
		if !actual[k] {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	if len(added) > 0 {
		t.Errorf("%s — NEW upstream types detected: %v", label, added)
	}
	if len(removed) > 0 {
		t.Errorf("%s — REMOVED upstream types detected: %v", label, removed)
	}
}
