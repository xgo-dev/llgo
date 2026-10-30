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
	"go/token"
	"go/types"
	"maps"
	"testing"
)

func TestNewBackendProgramSharesPreparedGoState(t *testing.T) {
	coordinator := NewProgram(nil)
	defer coordinator.Dispose()
	coordinator.DisableBoundsChecks(true)
	coordinator.EnableGoGlobalDCE(true)
	coordinator.EnableDeadcodeDrop(true)
	coordinator.EnableGCRoots(true)
	coordinator.EnableLogicalGoroutineLocality(true)
	coordinator.EnableCooperativeSafepoints(true)
	coordinator.SetPthreadStackSize(4096)
	coordinator.EnableLTOPluginMarkers(true)
	coordinator.EnableFuncInfoMetadata(true)
	coordinator.EnableFuncInfoSites(true)
	coordinator.SetDebugInfoOptimized(false)
	coordinator.EnableCodeViewDebugInfo(true)

	pkg := types.NewPackage("example.com/p", "p")
	fset := token.NewFileSet()
	coordinator.SetLinkname("example.com/p.Entry", "entry")
	coordinator.SetNoInterfaceMethod("example.com/p.T.Hidden")
	coordinator.SetTypeCallConv("example.com/p.CType", CDecl)
	coordinator.SetClosureEnvDirective(fset, "example.com/p.Entry", token.Pos(7))
	coordinator.MarkPackageSyntaxParsed(pkg, map[string]string{"example.com/p.Entry": "entry"})
	coordinator.SetLocalityInfo("example.com/p.Value", LocalityInfo{Locality: ThreadLocal})
	coordinator.SetPython(func() *types.Package { return nil })

	backend := coordinator.NewBackendProgram()
	defer backend.Dispose()
	if backend.ctx.C == coordinator.ctx.C {
		t.Fatal("backend Program shares the coordinator LLVM context")
	}
	if backend.tm.C == coordinator.tm.C {
		t.Fatal("backend Program shares the coordinator TargetMachine")
	}
	if backend.packageSyntax != coordinator.packageSyntax || backend.localities != coordinator.localities {
		t.Fatal("backend Program did not share prepared Go metadata")
	}
	if backend.gocvt.typs == nil || len(backend.gocvt.typs) != 0 || backend.named == nil || backend.abiSymbol == nil {
		t.Fatal("backend Program did not start with fresh lowering caches")
	}
	if link, ok := backend.Linkname("example.com/p.Entry"); !ok || link != "entry" {
		t.Fatalf("Linkname = (%q, %v), want (entry, true)", link, ok)
	}
	if !backend.HasLinknameTarget("entry") || backend.HasLinknameTarget("missing") {
		t.Fatal("backend Program did not preserve linkname target lookup")
	}
	if exports := maps.Collect(backend.PackageExports(pkg)); len(exports) != 1 || exports["example.com/p.Entry"] != "entry" {
		t.Fatalf("backend lost prepared package exports: %v", exports)
	}
	if !backend.HasClosureEnvDirective(fset, "example.com/p.Entry", token.Pos(7)) || !backend.PackageSyntaxParsed(pkg) {
		t.Fatal("backend Program lost prepared syntax metadata")
	}
	if background, ok := backend.packageTypeCallConv("example.com/p.CType"); !ok || background != CDecl {
		t.Fatalf("type background = (%v, %v), want (InC, true)", background, ok)
	}
	if locality, ok := backend.VariableLocality("example.com/p.Value"); !ok || locality.Locality != ThreadLocal {
		t.Fatalf("locality = (%+v, %v), want ThreadLocal", locality, ok)
	}
	if backend.python() != nil {
		t.Fatal("backend Program changed the prepared optional Python package")
	}
	if !backend.disableBoundsChecks || !backend.enableGoGlobalDCE || !backend.enableDeadcodeDrop || !backend.enableGCRoots ||
		!backend.logicalGoroutineLocal || !backend.enableSafepoints ||
		backend.pthreadStackSize != 4096 || !backend.enableLTOPluginMarker ||
		!backend.enableFuncInfoMetadata || !backend.enableFuncInfoSites || backend.debugInfoOptimized ||
		!backend.emitCodeViewDebugInfo {
		t.Fatal("backend Program did not preserve coordinator configuration")
	}
}

func TestPreparedPackageExportsKeepPackageIdentity(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	first := types.NewPackage("example.com/p", "p")
	second := types.NewPackage("example.com/p", "p")
	empty := types.NewPackage("example.com/p", "p")
	unknown := types.NewPackage("example.com/p", "p")
	exports := map[string]string{"example.com/p.Entry": "first_entry"}
	prog.MarkPackageSyntaxParsed(first, exports)
	exports["example.com/p.Entry"] = "second_entry"
	prog.MarkPackageSyntaxParsed(second, exports)
	prog.MarkPackageSyntaxParsed(empty, nil)
	for pkg, want := range map[*types.Package]string{first: "first_entry", second: "second_entry"} {
		got := maps.Collect(prog.PackageExports(pkg))
		if len(got) != 1 || got["example.com/p.Entry"] != want {
			t.Fatalf("prepared package exports = %v, want Entry: %s", got, want)
		}
	}
	if !prog.PackageSyntaxParsed(empty) || len(maps.Collect(prog.PackageExports(empty))) != 0 {
		t.Fatal("prepared empty package inherited another package's export")
	}
	if prog.PackageSyntaxParsed(unknown) || len(maps.Collect(prog.PackageExports(unknown))) != 0 {
		t.Fatal("unknown package inherited another package's preparation")
	}
}

func TestPreparedPackageExportsEarlyStop(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("example.com/p", "p")
	exports := map[string]string{"example.com/p.A": "a", "example.com/p.B": "b"}
	prog.MarkPackageSyntaxParsed(pkg, exports)
	seq := prog.PackageExports(pkg)
	visited := 0
	for name, export := range seq {
		visited++
		if name != "example.com/p.A" || export != "a" {
			t.Fatalf("first export = (%q, %q), want (example.com/p.A, a)", name, export)
		}
		break
	}
	if visited != 1 {
		t.Fatalf("visited %d exports, want 1", visited)
	}
	if got := maps.Collect(seq); !maps.Equal(got, exports) {
		t.Fatalf("iteration after early stop = %v, want %v", got, exports)
	}
}

func TestPackageExportSymbolReplacement(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")
	pkg.SetExport("example.com/p.First", "shared")
	pkg.SetExport("example.com/p.Second", "shared")
	pkg.SetExport("example.com/p.First", "first")
	if !pkg.IsExportedSymbol("shared") || !pkg.IsExportedSymbol("first") {
		t.Fatal("replacing one export lost a symbol still exported by another declaration")
	}
	pkg.SetExport("example.com/p.Second", "second")
	if pkg.IsExportedSymbol("shared") || !pkg.IsExportedSymbol("second") {
		t.Fatal("symbol lookup retained a replaced export or lost its replacement")
	}
	if got, ok := pkg.LookupExport("example.com/p.First"); !ok || got != "first" {
		t.Fatalf("first export = (%q, %v), want (first, true)", got, ok)
	}
}
