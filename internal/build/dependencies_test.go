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
	llssa "github.com/xgo-dev/llgo/ssa"
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

func TestLinkedPackageClosureUsesOnlyRootAndEffectiveDependencies(t *testing.T) {
	root := &packages.Package{ID: "root", ExportFile: "root.a"}
	base := &packages.Package{ID: "base", ExportFile: "base.a"}
	altOnly := &packages.Package{ID: "alt-only", ExportFile: "alt.a"}
	unrelated := &packages.Package{ID: "unrelated", ExportFile: "unrelated.a"}
	runtimePkg := &packages.Package{ID: "runtime", PkgPath: llssa.PkgRuntime, ExportFile: "runtime.a"}
	// An alternate package prepared only for another initial must not pull its
	// imports (or their runtime requirements and link arguments) into this link.
	unrelated.PkgPath = "github.com/xgo-dev/llgo/runtime/internal/lib/fmt"
	unrelatedDep := &packages.Package{ID: "fmt-dependency", ExportFile: "fmt-dependency.a"}
	unrelated.Imports = map[string]*packages.Package{unrelatedDep.ID: unrelatedDep}
	root.Imports = map[string]*packages.Package{"base": base}

	wrapped := func(pkg *packages.Package) *aPackage { return &aPackage{Package: pkg} }
	rootPkg := wrapped(root)
	rootPkg.AltPkg = &packages.Cached{Package: &packages.Package{
		ID: "alt-root", Imports: map[string]*packages.Package{"alt-only": altOnly},
	}}
	basePkg, altPkg := wrapped(base), wrapped(altOnly)
	unrelatedPkg, runtimeWrapped := wrapped(unrelated), wrapped(runtimePkg)
	runtimeWrapped.ArchiveFile = "runtime.a"
	unrelatedDepPkg := wrapped(unrelatedDep)
	ctx := &context{
		pkgs: map[*packages.Package]Package{
			root: rootPkg, base: basePkg, altOnly: altPkg, unrelated: unrelatedPkg, runtimePkg: runtimeWrapped,
			unrelatedDep: unrelatedDepPkg,
		},
		pkgByID: map[string]Package{
			"root": rootPkg, "base": basePkg, "alt-only": altPkg, "unrelated": unrelatedPkg, "runtime": runtimeWrapped,
			unrelatedDep.ID: unrelatedDepPkg,
		},
	}
	gotPkgs := linkedPackageClosure(ctx, root, []*aPackage{rootPkg, basePkg, altPkg, unrelatedPkg, runtimeWrapped, unrelatedDepPkg})
	got := make([]string, len(gotPkgs))
	for i, pkg := range gotPkgs {
		got[i] = pkg.ID
	}
	if want := []string{"alt-only", "base", "root", "runtime"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("linkedPackageClosure = %v, want %v", got, want)
	}
}

func TestLinkedPackageClosureSkipsUnbuiltRuntime(t *testing.T) {
	root := &packages.Package{ID: "root", PkgPath: "root", ExportFile: "root.a"}
	runtimePkg := &packages.Package{ID: llssa.PkgRuntime, PkgPath: llssa.PkgRuntime, ExportFile: "go-runtime.a"}
	rootBuilt := &aPackage{Package: root, ArchiveFile: "root.a"}
	runtimeLoaded := &aPackage{Package: runtimePkg}
	ctx := &context{pkgs: map[*packages.Package]Package{root: rootBuilt, runtimePkg: runtimeLoaded}}
	got := linkedPackageClosure(ctx, root, []Package{rootBuilt, runtimeLoaded})
	if len(got) != 1 || got[0] != rootBuilt {
		t.Fatalf("C-only link includes unbuilt runtime: %v", got)
	}
}
