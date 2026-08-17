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

package ssa

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa/abi"
)

func TestLocalityInfos(t *testing.T) {
	prog := NewProgram(nil)
	pkg := types.NewPackage("example.com/p", "p")
	if prog.PackageSyntaxParsed(pkg) {
		t.Fatal("new package was already marked as parsed")
	}
	prog.MarkPackageSyntaxParsed(pkg)
	if !prog.PackageSyntaxParsed(pkg) {
		t.Fatal("package syntax parsed marker was not retained")
	}
	if prog.PackageSyntaxParsed(types.NewPackage("example.com/p", "p")) {
		t.Fatal("syntax marker was shared by distinct package objects")
	}

	name := "example.com/p.value"
	prog.SetLocalityInfo(name, LocalityInfo{
		Locality:       GoroutineLocal,
		HasInitializer: true,
		InitFunc:       "example.com/p.initLocal",
		InitOrder:      1,
	})
	prog.SetLocalStorage(name, LocalStoragePackage)

	want, ok := prog.VariableLocality(name)
	if !ok || want.Locality != GoroutineLocal || want.LocalStorage != LocalStoragePackage || !want.HasInitializer || want.InitFunc == "" || want.InitOrder != 1 {
		t.Fatalf("VariableLocality(%q) = %+v, %v", name, want, ok)
	}

	prog.SetLocalStorage("example.com/p.ordinary", LocalStorageNativeTLS)
	decls := prog.PackageLocalities("example.com/p")
	if len(decls) != 1 || decls[name] != want {
		t.Fatalf("PackageLocalities = %+v", decls)
	}
	delete(decls, name)
	if _, ok := prog.VariableLocality(name); !ok {
		t.Fatal("mutating PackageLocalities changed program metadata")
	}
}

func TestPackageLocalitiesRetainDeclarationOwners(t *testing.T) {
	prog := NewProgram(nil)
	std := types.NewPackage("runtime", "runtime")
	alt := types.NewPackage(abi.PatchPathPrefix+"runtime", "runtime")

	prog.DeclareLocality(std, "stdState", LocalityInfo{Locality: ThreadLocal})
	prog.DeclareLocality(alt, "altState", LocalityInfo{Locality: GoroutineLocal})
	prog.DeclareLocality(std, "sharedState", LocalityInfo{Locality: ThreadLocal})
	prog.DeclareLocality(alt, "sharedState", LocalityInfo{Locality: GoroutineLocal})
	prog.SetLocalStorageFor(std, "runtime.sharedState", LocalStorageNativeTLS)
	prog.SetLocalStorageFor(alt, "runtime.sharedState", LocalStoragePackage)
	prog.SetLocalityInfo("runtime.preloadedState", LocalityInfo{Locality: ThreadLocal})

	check := func(pkg *types.Package, wants ...string) {
		t.Helper()
		got := prog.PackageLocalitiesFor(pkg)
		if len(got) != len(wants) {
			t.Fatalf("PackageLocalitiesFor(%q) = %+v, want %v", pkg.Path(), got, wants)
		}
		for _, name := range wants {
			if _, ok := got["runtime."+name]; !ok {
				t.Fatalf("PackageLocalitiesFor(%q) = %+v, missing %q", pkg.Path(), got, name)
			}
		}
	}
	check(std, "stdState", "sharedState", "preloadedState")
	check(alt, "altState", "sharedState", "preloadedState")

	stdShared, ok := prog.VariableLocalityFor(std, "runtime.sharedState")
	if !ok || stdShared.Locality != ThreadLocal || stdShared.LocalStorage != LocalStorageNativeTLS {
		t.Fatalf("standard sharedState = %+v, %v", stdShared, ok)
	}
	altShared, ok := prog.VariableLocalityFor(alt, "runtime.sharedState")
	if !ok || altShared.Locality != GoroutineLocal || altShared.LocalStorage != LocalStoragePackage {
		t.Fatalf("alternate sharedState = %+v, %v", altShared, ok)
	}
	reloadedAlt := types.NewPackage(abi.PatchPathPrefix+"runtime", "runtime")
	reloadedShared, ok := prog.VariableLocalityFor(reloadedAlt, "runtime.sharedState")
	if !ok || reloadedShared != altShared {
		t.Fatalf("reloaded alternate sharedState = %+v, %v; want %+v", reloadedShared, ok, altShared)
	}
	prog.DeclareLocality(reloadedAlt, "sharedState", LocalityInfo{Locality: GoroutineLocal})
	redeclaredShared, ok := prog.VariableLocalityFor(reloadedAlt, "runtime.sharedState")
	if !ok || redeclaredShared != altShared {
		t.Fatalf("redeclared alternate sharedState = %+v, %v; want prepared %+v", redeclaredShared, ok, altShared)
	}

	if got := prog.PackageLocalities("runtime"); len(got) != 4 {
		t.Fatalf("canonical PackageLocalities(runtime) = %+v, want all four declaration names", got)
	}
}

func TestNeedsLocalContext(t *testing.T) {
	prog := NewProgram(nil)
	if prog.NeedsLocalContext() {
		t.Fatal("empty program needs a local context")
	}
	name := "example.com/p.value"
	prog.SetLocalityInfo(name, LocalityInfo{Locality: ThreadLocal})
	if !prog.NeedsLocalContext() {
		t.Fatal("unknown local storage did not conservatively require a context")
	}
	prog.SetLocalStorage(name, LocalStorageNativeTLS)
	if prog.NeedsLocalContext() {
		t.Fatal("native TLS required a local context")
	}
	prog.SetLocalityInfo(name, LocalityInfo{Locality: ThreadLocal, HasInitializer: true, InitFunc: "example.com/p.initValue", InitOrder: 1})
	if !prog.NeedsLocalContext() {
		t.Fatal("native TLS initializer failure storage did not require a context")
	}
	prog.SetLocalityInfo(name, LocalityInfo{Locality: ThreadLocal})
	prog.SetLocalStorage(name, LocalStoragePackage)
	if !prog.NeedsLocalContext() {
		t.Fatal("context storage was not detected")
	}
}

func TestNeedsLocalContextIgnoresInactiveDeclarations(t *testing.T) {
	prog := NewProgram(nil)
	std := types.NewPackage("runtime", "runtime")
	alt := types.NewPackage(abi.PatchPathPrefix+"runtime", "runtime")
	name := "runtime.state"

	prog.DeclareLocality(std, "state", LocalityInfo{Locality: ThreadLocal})
	prog.SetLocalStorageFor(std, name, LocalStorageNativeTLS)
	prog.DeclareLocality(alt, "state", LocalityInfo{Locality: GoroutineLocal})
	prog.SetLocalStorageFor(alt, name, LocalStoragePackage)
	if prog.NeedsLocalContext() {
		t.Fatal("scanned inactive alternate declaration required a local context")
	}

	prog.ActivateLocalitiesFor(std)
	if prog.NeedsLocalContext() {
		t.Fatal("active native-TLS standard declaration used alternate package storage")
	}
	prog.ActivateLocalitiesFor(alt)
	if !prog.NeedsLocalContext() {
		t.Fatal("active alternate package storage did not require a local context")
	}
}

func TestLocalityMetadataFallbacks(t *testing.T) {
	t.Run("owner update and ownerless fallback", func(t *testing.T) {
		prog := NewProgram(nil)
		pkg := types.NewPackage("example.com/p", "p")
		ownedName := "example.com/p.owned"
		prog.SetLocalityInfoFor(pkg, ownedName, LocalityInfo{Locality: ThreadLocal})
		owned, ok := prog.VariableLocalityFor(pkg, ownedName)
		if !ok || owned.Locality != ThreadLocal {
			t.Fatalf("owner-specific locality = %+v, %v", owned, ok)
		}

		fallbackName := "example.com/p.fallback"
		prog.SetLocalityInfo(fallbackName, LocalityInfo{Locality: GoroutineLocal})
		fallback, ok := prog.VariableLocalityFor(pkg, fallbackName)
		if !ok || fallback.Locality != GoroutineLocal {
			t.Fatalf("ownerless fallback locality = %+v, %v", fallback, ok)
		}

		prog.SetLocalityInfo("example.com/p.ordinary", LocalityInfo{})
		prog.SetLocalityInfo("example.com/other.local", LocalityInfo{Locality: ThreadLocal})
		got := prog.PackageLocalitiesFor(pkg)
		if _, ok := got["example.com/p.ordinary"]; ok {
			t.Fatalf("ordinary variable returned as local: %+v", got)
		}
		if _, ok := got["example.com/other.local"]; ok {
			t.Fatalf("other package locality returned: %+v", got)
		}

		// A nil package is intentionally a no-op for callers walking optional
		// package roots.
		prog.ActivateLocalitiesFor(nil)
	})

	t.Run("metadata without local variables", func(t *testing.T) {
		prog := NewProgram(nil)
		prog.SetLocalityInfo("example.com/p.ordinary", LocalityInfo{})
		if err := prog.ValidateLocalities("example.com/p"); err != nil {
			t.Fatalf("ordinary metadata failed locality validation: %v", err)
		}
	})

	t.Run("legacy canonical entry", func(t *testing.T) {
		prog := NewProgram(nil)
		// The entries map predates declaration ownership. Keep its compatibility
		// path covered for callers that preload canonical metadata directly.
		prog.localities.mu.Lock()
		prog.localities.entries["example.com/p.legacy"] = VariableLocality{
			Info:         LocalityInfo{Locality: GoroutineLocal},
			LocalStorage: LocalStoragePackage,
		}
		prog.localities.mu.Unlock()
		if !prog.NeedsLocalContext() {
			t.Fatal("legacy canonical locality did not require a context")
		}
	})
}

func TestRejectsLinknameLocality(t *testing.T) {
	prog := NewProgram(nil)
	target := "example.com/target.value"
	alias := "example.com/alias.value"
	prog.SetLocalityInfo(target, LocalityInfo{Locality: ThreadLocal, HasInitializer: true, InitFunc: "example.com/target.initValue", InitOrder: 1})
	prog.SetLocalStorage(target, LocalStoragePackage)
	if canonical, got, ok, err := prog.ResolveLocality(target); err != nil || canonical != target || !ok || got.LocalStorage != LocalStoragePackage {
		t.Fatalf("direct ResolveLocality(%q) = %q, %+v, %v, %v", target, canonical, got, ok, err)
	}

	prog.SetLinkname(alias, target)
	if err := prog.ValidateLocalities("example.com/alias"); err == nil || !strings.Contains(err.Error(), "cannot reference local variable") {
		t.Fatalf("alias-to-local error = %v", err)
	}

	localAlias := "example.com/alias.local"
	prog.SetLocalityInfo(localAlias, LocalityInfo{Locality: GoroutineLocal})
	prog.SetLinkname(localAlias, "example.com/target.ordinary")
	if err := prog.ValidateLocalities("example.com/alias"); err == nil || !strings.Contains(err.Error(), "cannot use go:linkname") {
		t.Fatalf("local-alias error = %v", err)
	}
}

func TestValidateLocalitiesIgnoresOrdinaryLinknameCycle(t *testing.T) {
	prog := NewProgram(nil)
	first := "example.com/p.first"
	second := "example.com/p.second"
	prog.SetLinkname(first, second)
	prog.SetLinkname(second, first)
	if err := prog.ValidateLocalities("example.com/p"); err != nil {
		t.Fatalf("ordinary linkname cycle affected locality validation: %v", err)
	}
	if _, _, _, err := prog.ResolveLocality(first); err == nil || !strings.Contains(err.Error(), "linkname cycle") {
		t.Fatalf("ResolveLocality cycle error = %v", err)
	}
}

func TestValidateLocalitySelfLinkname(t *testing.T) {
	prog := NewProgram(nil)
	name := "example.com/p.value"
	prog.SetLinkname(name, name)
	if err := prog.ValidateLocalities("example.com/p"); err != nil {
		t.Fatal(err)
	}
	if canonical, _, ok, err := prog.ResolveLocality(name); err != nil || canonical != name || ok {
		t.Fatalf("ordinary self-link ResolveLocality(%q) = %q, %v, %v", name, canonical, ok, err)
	}
	prog.SetLocalityInfo(name, LocalityInfo{Locality: ThreadLocal})
	if err := prog.ValidateLocalities("example.com/p"); err == nil || !strings.Contains(err.Error(), "cannot use go:linkname") {
		t.Fatalf("local self-linkname error = %v", err)
	}
}
