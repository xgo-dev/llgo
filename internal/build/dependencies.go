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

	"github.com/goplus/llgo/internal/packages"
)

// effectiveDependencies returns the package graph that contributes code to an
// aPackage. An alternate package can add imports beyond the original package,
// so using only Package.Imports would allow stale cache entries after an alt
// dependency changes.
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

// packageBuildDependencies returns the true Go import edges for scheduler
// ordering. Alternate-package imports still participate in cache fingerprints,
// but may intentionally form cycles with the runtime replacement graph after
// all packages have already been built into SSA.
func packageBuildDependencies(pkg *aPackage) []*packages.Package {
	if pkg == nil || pkg.Package == nil {
		return nil
	}
	deps := make(map[string]*packages.Package, len(pkg.Imports))
	for _, dep := range pkg.Imports {
		if dep != nil && dep.ID != pkg.ID {
			deps[dep.ID] = dep
		}
	}
	ret := make([]*packages.Package, 0, len(deps))
	for _, dep := range deps {
		ret = append(ret, dep)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
	return ret
}
