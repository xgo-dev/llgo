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
