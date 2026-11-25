/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package typepatch

import (
	"sync"

	"github.com/xgo-dev/llgo/types"
)

var patchedPkgs sync.Map // map[*types.Package]struct{}

func IsPatched(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	_, ok := patchedPkgs.Load(pkg)
	return ok
}

func Clone(alt *types.Package) *types.Package {
	if alt == nil {
		return nil
	}
	cp := types.NewPackage(alt.Path(), alt.Name())
	cp.SetImports(alt.Imports())
	if alt.Complete() {
		cp.MarkComplete()
	}
	if src := alt.Scope(); src != nil {
		dst := cp.Scope()
		for _, name := range src.Names() {
			if obj := src.Lookup(name); obj != nil {
				dst.Insert(obj)
			}
		}
	}
	return cp
}

func Merge(alt, pkg *types.Package, skips map[string]struct{}, skipall bool) {
	if pkg != nil {
		patchedPkgs.Store(pkg, struct{}{})
	}
	if skipall || alt == nil || pkg == nil {
		return
	}
	dst := alt.Scope()
	src := pkg.Scope()
	if dst == nil || src == nil {
		return
	}
	for _, name := range src.Names() {
		if dst.Lookup(name) != nil {
			continue
		}
		if _, skip := skips[name]; skip {
			continue
		}
		if obj := src.Lookup(name); obj != nil {
			dst.Insert(obj)
		}
	}
}
