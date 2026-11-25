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

package locality

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
)

// Prepare rewrites local package initializers into replayable synthetic
// functions and returns updated metadata. It is idempotent for repeated calls,
// including calls made by another compiler Program reusing the same syntax and
// types.Package objects.
func Prepare(fset *token.FileSet, pkgPath string, pkg *types.Package, typeInfo *types.Info, files []*ast.File, vars map[string]Info) (map[string]Info, error) {
	ret := cloneInfo(vars)
	if pkg == nil || typeInfo == nil || !hasLocality(ret) {
		return ret, nil
	}

	nextName := 0
	for order, initializer := range typeInfo.InitOrder {
		_, found, err := initializerLocality(fset, initializer, ret)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		initOrder := order + 1
		if initName := preparedInitName(initializer, ret, initOrder); initName != "" {
			setInitializerNames(initializer, ret, initName, initOrder)
			continue
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("cannot prepare local initializer for package %q without syntax files", pkgPath)
		}
		name := findLocalInitializer(pkg, typeInfo, files, initializer)
		if name == "" {
			for {
				name = fmt.Sprintf("%s%d", InitPrefix, nextName)
				nextName++
				if pkg.Scope().Lookup(name) == nil {
					break
				}
			}
			fnObj, decl := makeLocalInitializer(pkg, typeInfo, name, initializer)
			pkg.Scope().Insert(fnObj)
			files[len(files)-1].Decls = append(files[len(files)-1].Decls, decl)
		}
		setInitializerNames(initializer, ret, qualify(pkgPath, name), initOrder)
	}

	return ret, nil
}

// ValidatePrepared verifies that every explicit local initializer was prepared
// before Go SSA construction.
func ValidatePrepared(pkgPath string, vars map[string]Info) error {
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info := vars[name]
		if info.Locality == None {
			continue
		}
		prepared := info.InitFunc != "" && info.InitOrder != 0
		if info.HasInitializer != prepared {
			return fmt.Errorf("local variable %s has inconsistent initializer metadata before SSA compilation", qualify(pkgPath, name))
		}
	}
	return nil
}

func initializerLocality(fset *token.FileSet, initializer *types.Initializer, vars map[string]Info) (Kind, bool, error) {
	var kind Kind
	localCount := 0
	for _, variable := range initializer.Lhs {
		info, ok := vars[variable.Name()]
		if !ok || info.Locality == None {
			if localCount != 0 {
				return None, false, errorAt(fset, initializer.Rhs.Pos(), "one initializer cannot mix local and ordinary package variables")
			}
			continue
		}
		if localCount == 0 {
			kind = info.Locality
		} else if kind != info.Locality {
			return None, false, errorAt(fset, initializer.Rhs.Pos(), "one initializer cannot mix thread-local and goroutine-local variables")
		}
		localCount++
	}
	if localCount != 0 && len(initializer.Lhs) != localCount {
		return None, false, errorAt(fset, initializer.Rhs.Pos(), "one initializer cannot mix local and ordinary package variables")
	}
	return kind, localCount != 0, nil
}

func preparedInitName(initializer *types.Initializer, vars map[string]Info, order int) string {
	var name string
	for _, variable := range initializer.Lhs {
		info := vars[variable.Name()]
		if info.InitFunc == "" || info.InitOrder != order {
			return ""
		}
		if name == "" {
			name = info.InitFunc
		} else if name != info.InitFunc {
			return ""
		}
	}
	return name
}

func findLocalInitializer(pkg *types.Package, info *types.Info, files []*ast.File, initializer *types.Initializer) string {
	for _, file := range files {
		for _, node := range file.Decls {
			decl, ok := node.(*ast.FuncDecl)
			if !ok || len(decl.Name.Name) < len(InitPrefix) || decl.Name.Name[:len(InitPrefix)] != InitPrefix || decl.Body == nil || len(decl.Body.List) != 1 {
				continue
			}
			assign, ok := decl.Body.List[0].(*ast.AssignStmt)
			if !ok || assign.Tok != token.ASSIGN || len(assign.Rhs) != 1 || assign.Rhs[0] != initializer.Rhs || len(assign.Lhs) != len(initializer.Lhs) {
				continue
			}
			matches := true
			for i, lhs := range assign.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || info.Uses[ident] != initializer.Lhs[i] {
					matches = false
					break
				}
			}
			if matches {
				if object := pkg.Scope().Lookup(decl.Name.Name); object == info.Defs[decl.Name] {
					return decl.Name.Name
				}
			}
		}
	}
	return ""
}

func makeLocalInitializer(pkg *types.Package, info *types.Info, name string, initializer *types.Initializer) (*types.Func, *ast.FuncDecl) {
	if info.Uses == nil {
		info.Uses = make(map[*ast.Ident]types.Object)
	}
	if info.Defs == nil {
		info.Defs = make(map[*ast.Ident]types.Object)
	}
	lhs := make([]ast.Expr, len(initializer.Lhs))
	for i, variable := range initializer.Lhs {
		ident := ast.NewIdent(variable.Name())
		info.Uses[ident] = variable
		lhs[i] = ident
	}
	nameIdent := ast.NewIdent(name)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fnObj := types.NewFunc(token.NoPos, pkg, name, sig)
	info.Defs[nameIdent] = fnObj
	decl := &ast.FuncDecl{
		Name: nameIdent,
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{Lhs: lhs, Tok: token.ASSIGN, Rhs: []ast.Expr{initializer.Rhs}},
		}},
	}
	return fnObj, decl
}

func setInitializerNames(initializer *types.Initializer, vars map[string]Info, initName string, order int) {
	for _, variable := range initializer.Lhs {
		info := vars[variable.Name()]
		info.InitFunc = initName
		info.InitOrder = order
		vars[variable.Name()] = info
	}
}

func cloneInfo(vars map[string]Info) map[string]Info {
	ret := make(map[string]Info, len(vars))
	for name, info := range vars {
		ret[name] = info
	}
	return ret
}

func hasLocality(vars map[string]Info) bool {
	for _, info := range vars {
		if info.Locality != None {
			return true
		}
	}
	return false
}

func qualify(pkgPath, name string) string {
	if pkgPath == "" {
		return name
	}
	return pkgPath + "." + name
}
