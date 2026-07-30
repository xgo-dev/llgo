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
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/goplus/llgo/internal/locality"
	localitylayout "github.com/goplus/llgo/internal/locality/layout"
	llssa "github.com/goplus/llgo/ssa"
)

// PrepareLocalVariables extracts replayable initializer helpers before Go SSA
// construction, then records the storage strategy selected by the independent
// package-layout planner.
func PrepareLocalVariables(prog llssa.Program, fset *token.FileSet, pkg *types.Package, info *types.Info, files []*ast.File) error {
	if pkg == nil || info == nil {
		return nil
	}
	path := prog.PathOf(pkg)
	prepared, err := locality.Prepare(fset, path, pkg, info, files, packageLocalities(prog, path))
	if err != nil {
		return err
	}
	for name, local := range prepared {
		prog.SetLocalityInfo(prog.FullName(pkg, name), local)
	}
	for fullName := range prog.PackageLocalities(path) {
		name := strings.TrimPrefix(fullName, path+".")
		object, _ := pkg.Scope().Lookup(name).(*types.Var)
		if object == nil {
			return fmt.Errorf("locality layout: package %s has no variable %s", path, name)
		}
		_, _, _, err := prog.ResolveLocality(fullName)
		if err != nil {
			return err
		}
		prog.SetLocalStorage(fullName, localitylayout.StorageForType(object.Type()))
	}
	_, err = planLocalPackage(prog, pkg)
	return err
}

func validateLocalInitializers(prog llssa.Program, pkg *types.Package) error {
	return locality.ValidatePrepared(prog.PathOf(pkg), packageLocalities(prog, prog.PathOf(pkg)))
}

func packageLocalities(prog llssa.Program, pkgPath string) map[string]locality.Info {
	prefix := pkgPath + "."
	ret := make(map[string]locality.Info)
	for name, info := range prog.PackageLocalities(pkgPath) {
		ret[strings.TrimPrefix(name, prefix)] = info.Info
	}
	return ret
}

func planLocalPackage(prog llssa.Program, pkg *types.Package) (localitylayout.Package, error) {
	if pkg == nil {
		return localitylayout.Package{}, nil
	}
	path := prog.PathOf(pkg)
	prefix := path + "."
	decls := prog.PackageLocalities(path)
	input := make([]localitylayout.Declaration, 0, len(decls))
	for fullName := range decls {
		_, info, _, err := prog.ResolveLocality(fullName)
		if err != nil {
			return localitylayout.Package{}, err
		}
		name := strings.TrimPrefix(fullName, prefix)
		object, _ := pkg.Scope().Lookup(name).(*types.Var)
		if object == nil {
			return localitylayout.Package{}, fmt.Errorf("locality layout: package %s has no variable %s", path, name)
		}
		input = append(input, localitylayout.Declaration{Name: fullName, Type: object.Type(), Info: info.Info})
	}
	return localitylayout.Plan(path, input)
}
