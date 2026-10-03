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
	"sort"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

// effectiveDependencies returns every package whose source contributes to an
// aPackage. Alternate packages may add imports absent from the original graph.
func effectiveDependencies(pkg *aPackage) []*packages.Package {
	if pkg == nil || pkg.Package == nil {
		return nil
	}
	deps := make(map[string]*packages.Package)
	add := func(imports map[string]*packages.Package) {
		for _, dep := range imports {
			if dep == nil || dep.ID == pkg.ID || (pkg.AltPkg != nil && dep.ID == pkg.AltPkg.ID) {
				continue
			}
			deps[dep.ID] = dep
		}
	}
	add(pkg.Imports)
	if pkg.AltPkg != nil {
		add(pkg.AltPkg.Imports)
	}
	ret := make([]*packages.Package, 0, len(deps))
	for _, dep := range deps {
		ret = append(ret, dep)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
	return ret
}

// linkedPackageClosure returns dependency-first packages for one link. It
// follows alternate-package-only imports as well as the ordinary Go graph and
// includes the core runtime and its dependencies. Other prepared runtime-tree
// packages may be alternate implementations needed only by a different root.
func linkedPackageClosure(ctx *context, root *packages.Package, built []*aPackage) []Package {
	seen := make(map[string]bool)
	var order []Package
	var visit func(*packages.Package)
	visit = func(pkg *packages.Package) {
		if pkg == nil || seen[pkg.ID] {
			return
		}
		seen[pkg.ID] = true
		aPkg := ctx.pkgs[pkg]
		if aPkg == nil {
			aPkg = ctx.pkgByID[pkg.ID]
		}
		if aPkg == nil {
			return
		}
		for _, dep := range effectiveDependencies(aPkg) {
			visit(dep)
		}
		if pkg.ExportFile != "" {
			order = append(order, aPkg)
		}
	}
	visit(root)
	for _, pkg := range built {
		// Runtime packages are preloaded even for C-only entry points. They
		// have a Go export file but no built archive or backend in that case.
		if pkg.PkgPath == llssa.PkgRuntime && pkg.ArchiveFile != "" {
			visit(pkg.Package)
		}
	}
	return order
}
