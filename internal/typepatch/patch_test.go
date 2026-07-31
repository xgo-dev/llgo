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
	"go/token"
	"go/types"
	"testing"
)

func packageWithVars(path string, names ...string) *types.Package {
	pkg := types.NewPackage(path, "p")
	for _, name := range names {
		pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, name, types.Typ[types.Int]))
	}
	return pkg
}

func TestMergePreparedDoesNotMarkOriginal(t *testing.T) {
	original := packageWithVars("example.com/original", "Keep", "Skip")
	merged := Clone(packageWithVars("example.com/alternate", "Alt"))

	MergePrepared(merged, original, map[string]struct{}{"Skip": {}}, false)

	for _, name := range []string{"Alt", "Keep"} {
		if merged.Scope().Lookup(name) == nil {
			t.Fatalf("merged package is missing %s", name)
		}
	}
	if merged.Scope().Lookup("Skip") != nil {
		t.Fatal("merged package retained skipped declaration")
	}
	if IsPatched(original) {
		t.Fatal("MergePrepared marked the original package as patched")
	}
	if original.Scope().Lookup("Keep") == nil || original.Scope().Lookup("Skip") == nil {
		t.Fatal("MergePrepared modified the original package scope")
	}
}

func TestMergePreparedSkipAll(t *testing.T) {
	original := packageWithVars("example.com/original", "Keep")
	merged := Clone(packageWithVars("example.com/alternate", "Alt"))

	MergePrepared(merged, original, nil, true)

	if merged.Scope().Lookup("Keep") != nil {
		t.Fatal("skipall merged an original declaration")
	}
	if merged.Scope().Lookup("Alt") == nil {
		t.Fatal("skipall removed an alternate declaration")
	}
}

func TestMergeMarksOriginal(t *testing.T) {
	original := packageWithVars("example.com/original", "Keep")
	merged := Clone(packageWithVars("example.com/alternate", "Alt"))

	Merge(merged, original, nil, false)

	if !IsPatched(original) {
		t.Fatal("Merge did not mark the original package as patched")
	}
	if merged.Scope().Lookup("Keep") == nil {
		t.Fatal("Merge did not include the original declaration")
	}
}
