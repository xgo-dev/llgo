//go:build !llgo
// +build !llgo

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
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestCompileFuncForPCInterfaceUsesFuncPCWrapper(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

import "reflect"

func target() {}

func pointer() uintptr {
	return reflect.ValueOf(target).Pointer()
}

func plain() any {
	return 1
}
`)
	if ir := m.String(); !strings.Contains(ir, "__llgo_funcpc_stub.foo.target") {
		t.Fatalf("compiled IR missing FuncForPC wrapper for function declaration:\n%s", ir)
	}
}

func TestMakeInterfaceNeedsFuncPC(t *testing.T) {
	ctx := &context{}

	reflectPkg := buildGoSSAPkgWithPath(t, "foo", `
package foo

import "reflect"

func target() {}

func pointer() uintptr {
	return reflect.ValueOf(target).Pointer()
}
`)
	if mi := findFuncMakeInterface(t, reflectPkg, "target"); !ctx.makeInterfaceNeedsFuncPC(mi) {
		t.Fatal("reflect.ValueOf(function) should preserve FuncForPC name")
	}

	commandLinePkg := buildGoSSAPkgWithPath(t, "command-line-arguments", `
package main

var sink any

func target() {}

func main() {
	sink = target
}
`)
	if mi := findFuncMakeInterface(t, commandLinePkg, "target"); !ctx.makeInterfaceNeedsFuncPC(mi) {
		t.Fatal("command-line-arguments function interfaces should preserve FuncForPC name")
	}

	storePkg := buildGoSSAPkgWithPath(t, "foo", `
package foo

var sink any

func target() {}

func use() {
	sink = target
}
`)
	if mi := findFuncMakeInterface(t, storePkg, "target"); ctx.makeInterfaceNeedsFuncPC(mi) {
		t.Fatal("plain function interface store should not require FuncForPC wrapper")
	}

	plainPkg := buildGoSSAPkgWithPath(t, "foo", `
package foo

func target() {}

func sink(any) {}

func use() {
	sink(target)
}
`)
	if mi := findFuncMakeInterface(t, plainPkg, "target"); ctx.makeInterfaceNeedsFuncPC(mi) {
		t.Fatal("plain function interface conversion should not require FuncForPC wrapper")
	}

	nonFuncPkg := buildGoSSAPkgWithPath(t, "foo", `
package foo

var sink any

func use() {
	sink = 1
}
`)
	if mi := findFirstMakeInterface(t, nonFuncPkg); ctx.makeInterfaceNeedsFuncPC(mi) {
		t.Fatal("non-function interface conversion should not require FuncForPC wrapper")
	}
}

func buildGoSSAPkgWithPath(t *testing.T, pkgPath, src string) *gossa.Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage(pkgPath, f.Name.Name)
	imp := packages.NewImporter(fset)
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssaPkg
}

func findFuncMakeInterface(t *testing.T, pkg *gossa.Package, name string) *gossa.MakeInterface {
	t.Helper()
	for _, mi := range allMakeInterfaces(pkg) {
		if fn, ok := mi.X.(*gossa.Function); ok && fn.Name() == name {
			return mi
		}
	}
	t.Fatalf("missing MakeInterface for function %q", name)
	return nil
}

func findFirstMakeInterface(t *testing.T, pkg *gossa.Package) *gossa.MakeInterface {
	t.Helper()
	if mis := allMakeInterfaces(pkg); len(mis) > 0 {
		return mis[0]
	}
	t.Fatal("missing MakeInterface instruction")
	return nil
}

func allMakeInterfaces(pkg *gossa.Package) []*gossa.MakeInterface {
	var ret []*gossa.MakeInterface
	for fn := range ssautil.AllFunctions(pkg.Prog) {
		if fn.Pkg != pkg {
			continue
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if mi, ok := instr.(*gossa.MakeInterface); ok {
					ret = append(ret, mi)
				}
			}
		}
	}
	return ret
}
