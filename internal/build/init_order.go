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

package build

import (
	"fmt"
	"sort"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

// packageInitOrder returns the packages reachable from root in the order in
// which Go initializes them: among packages whose imports are initialized,
// choose the one with the lexicographically first import path.
func packageInitOrder(root *packages.Package) ([]*packages.Package, error) {
	if root == nil {
		return nil, nil
	}

	seen := make(map[*packages.Package]bool)
	var all []*packages.Package
	var collect func(*packages.Package)
	collect = func(pkg *packages.Package) {
		if pkg == nil || seen[pkg] {
			return
		}
		seen[pkg] = true
		all = append(all, pkg)
		for _, imported := range pkg.Imports {
			collect(imported)
		}
	}
	collect(root)
	sort.Slice(all, func(i, j int) bool {
		if all[i].PkgPath != all[j].PkgPath {
			return all[i].PkgPath < all[j].PkgPath
		}
		return all[i].ID < all[j].ID
	})

	remaining := make(map[*packages.Package]int, len(all))
	importers := make(map[*packages.Package][]*packages.Package, len(all))
	for _, pkg := range all {
		imports := make(map[*packages.Package]bool, len(pkg.Imports))
		for _, imported := range pkg.Imports {
			if imported == nil || !seen[imported] || imports[imported] {
				continue
			}
			imports[imported] = true
			remaining[pkg]++
			importers[imported] = append(importers[imported], pkg)
		}
	}

	order := make([]*packages.Package, 0, len(all))
	initialized := make(map[*packages.Package]bool, len(all))
	for len(order) != len(all) {
		var next *packages.Package
		for _, pkg := range all {
			if !initialized[pkg] && remaining[pkg] == 0 {
				next = pkg
				break
			}
		}
		if next == nil {
			return nil, fmt.Errorf("package initialization graph rooted at %s contains a cycle", root.PkgPath)
		}
		initialized[next] = true
		order = append(order, next)
		for _, importer := range importers[next] {
			remaining[importer]--
		}
	}
	return order, nil
}

func linkedPackageInitNames(root *packages.Package, linked []Package) ([]string, error) {
	order, err := packageInitOrder(root)
	if err != nil {
		return nil, err
	}
	builtByID := make(map[string]Package, len(linked))
	for _, pkg := range linked {
		if pkg != nil && pkg.Package != nil {
			builtByID[pkg.ID] = pkg
		}
	}

	names := make([]string, 0, len(order))
	for _, pkg := range order {
		if pkg == root {
			continue
		}
		built := builtByID[pkg.ID]
		if built == nil || built.SSA == nil || built.SSA.Func("init") == nil || pkg.Types == nil {
			continue
		}
		if kind, _ := cl.PkgKindOf(pkg.Types); cl.PkgSkipsInit(kind) {
			continue
		}
		names = append(names, llssa.FullName(pkg.Types, "init"))
	}
	return names, nil
}
