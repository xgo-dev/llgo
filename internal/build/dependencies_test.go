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
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestEffectiveDependenciesIncludesAlternateImports(t *testing.T) {
	base := &packages.Package{ID: "base"}
	shared := &packages.Package{ID: "shared"}
	altOnly := &packages.Package{ID: "alt-only"}
	alt := &packages.Package{
		ID:      "alt",
		Imports: map[string]*packages.Package{"shared": shared, "alt-only": altOnly},
	}
	pkg := &aPackage{
		Package: &packages.Package{ID: "pkg", Imports: map[string]*packages.Package{"base": base, "shared": shared}},
		AltPkg:  &packages.Cached{Package: alt},
	}
	deps := effectiveDependencies(pkg)
	got := make([]string, len(deps))
	for i, dep := range deps {
		got[i] = dep.ID
	}
	if want := []string{"alt-only", "base", "shared"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("effectiveDependencies = %v, want %v", got, want)
	}
}

func TestEffectiveDependenciesHandlesNilPackage(t *testing.T) {
	if deps := effectiveDependencies(nil); deps != nil {
		t.Fatalf("effectiveDependencies(nil) = %v, want nil", deps)
	}
}
