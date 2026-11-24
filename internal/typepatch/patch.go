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
	"go/types"
	"runtime"
	"sync"
	"weak"
)

var patchedPkgs sync.Map // map[weak.Pointer[types.Package]]struct{}

func markPatched(pkg *types.Package) {
	if pkg == nil {
		return
	}
	wp := weak.Make(pkg)
	if _, loaded := patchedPkgs.LoadOrStore(wp, struct{}{}); !loaded {
		runtime.AddCleanup(pkg, func(key weak.Pointer[types.Package]) {
			patchedPkgs.Delete(key)
		}, wp)
	}
}

// IsPatched reports whether pkg has been merged into a replacement package via Merge.
func IsPatched(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	_, ok := patchedPkgs.Load(weak.Make(pkg))
	return ok
}

// Clone returns a shallow package copy with an independent package scope
// containing the same declarations as alt. Existing objects keep their original
// Parent scope.
func Clone(alt *types.Package) *types.Package {
	if alt == nil {
		return nil
	}
	ret := types.NewPackage(alt.Path(), alt.Name())
	ret.SetImports(alt.Imports())
	dst := ret.Scope()
	src := alt.Scope()
	for _, name := range src.Names() {
		dst.Insert(src.Lookup(name))
	}
	if alt.Complete() {
		ret.MarkComplete()
	}
	return ret
}

// Merge marks pkg as patched and copies non-skipped, non-conflicting objects
// from pkg's scope into alt's scope without altering each object's Parent.
func Merge(alt, pkg *types.Package, skips map[string]struct{}, skipall bool) {
	markPatched(pkg)
	if skipall || alt == nil || pkg == nil {
		return
	}
	dst := alt.Scope()
	src := pkg.Scope()
	for _, name := range src.Names() {
		if _, ok := skips[name]; ok {
			continue
		}
		// Scope.Insert preserves existing alt declarations and keeps obj.Parent intact.
		dst.Insert(src.Lookup(name))
	}
}
