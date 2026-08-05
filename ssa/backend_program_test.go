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
	"testing"
)

func TestNewBackendProgramSharesPreparedGoState(t *testing.T) {
	coordinator := NewProgram(nil)
	defer coordinator.Dispose()
	coordinator.DisableBoundsChecks(true)
	coordinator.EnableGoGlobalDCE(true)
	coordinator.EnableDeadcodeDrop(true)
	coordinator.SetPthreadStackSize(4096)
	coordinator.EnableLTOPluginMarkers(true)
	coordinator.EnableFuncInfoMetadata(true)
	coordinator.EnableFuncInfoSites(true)
	coordinator.SetDebugInfoOptimized(false)

	pkg := types.NewPackage("example.com/p", "p")
	fset := token.NewFileSet()
	coordinator.SetLinkname("example.com/p.Entry", "entry")
	coordinator.SetPackageExport("example.com/p.Entry", "entry")
	coordinator.SetNoInterfaceMethod("example.com/p.T.Hidden")
	coordinator.SetTypeBackground("example.com/p.CType", InC)
	coordinator.SetClosureEnvDirective(fset, "example.com/p.Entry", token.Pos(7))
	coordinator.MarkPackageSyntaxParsed(pkg)
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
	if export, ok := backend.PackageExport("example.com/p.Entry"); !ok || export != "entry" {
		t.Fatalf("PackageExport = (%q, %v), want (entry, true)", export, ok)
	}
	if !backend.HasClosureEnvDirective(fset, "example.com/p.Entry", token.Pos(7)) || !backend.PackageSyntaxParsed(pkg) {
		t.Fatal("backend Program lost prepared syntax metadata")
	}
	if background, ok := backend.packageTypeBackground("example.com/p.CType"); !ok || background != InC {
		t.Fatalf("type background = (%v, %v), want (InC, true)", background, ok)
	}
	if locality, ok := backend.VariableLocality("example.com/p.Value"); !ok || locality.Locality != ThreadLocal {
		t.Fatalf("locality = (%+v, %v), want ThreadLocal", locality, ok)
	}
	if backend.python() != nil {
		t.Fatal("backend Program changed the prepared optional Python package")
	}
	if !backend.disableBoundsChecks || !backend.enableGoGlobalDCE || !backend.enableDeadcodeDrop ||
		backend.pthreadStackSize != 4096 || !backend.enableLTOPluginMarker ||
		!backend.enableFuncInfoMetadata || !backend.enableFuncInfoSites || backend.debugInfoOptimized {
		t.Fatal("backend Program did not preserve coordinator configuration")
	}
}
