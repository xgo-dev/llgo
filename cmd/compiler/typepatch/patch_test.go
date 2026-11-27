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

package typepatch

import (
	"runtime"
	"testing"
	"weak"

	"github.com/xgo-dev/llgo/types"
)

func TestMergeDoesNotPinPatchedPackage(t *testing.T) {
	makePatched := func() weak.Pointer[types.Package] {
		alt := types.NewPackage("example.com/alt", "alt")
		pkg := types.NewPackage("example.com/orig", "orig")
		Merge(alt, pkg, nil, true)
		if !IsPatched(pkg) {
			t.Fatal("patched package not marked")
		}
		if IsPatched(alt) {
			t.Fatal("alt package unexpectedly marked patched")
		}
		return weak.Make(pkg)
	}
	wp := makePatched()
	for i := 0; i < 10 && wp.Value() != nil; i++ {
		runtime.GC()
	}
	if wp.Value() != nil {
		t.Fatal("typepatch.Merge retained patched *types.Package across GC")
	}
}
