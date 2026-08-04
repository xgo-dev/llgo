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
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/goplus/llgo/ssa"
)

func TestBackendProgramTemplateCreatesIsolatedSessions(t *testing.T) {
	conf := &Config{Goos: "linux", Goarch: "amd64"}
	template := newBackendProgramTemplate(
		&llssa.Target{GOOS: conf.Goos, GOARCH: conf.Goarch},
		conf,
		true,
		true,
	)
	first, err := template.newSession()
	if err != nil {
		t.Fatal(err)
	}
	defer first.prog.Dispose()
	second, err := template.newSession()
	if err != nil {
		t.Fatal(err)
	}
	defer second.prog.Dispose()
	if first.transformer == nil || second.transformer == nil {
		t.Fatal("backend session missing C ABI transformer")
	}
	if !first.prog.FuncInfoMetadataEnabled() || !first.prog.FuncInfoSitesEnabled() {
		t.Fatal("backend template did not preserve funcinfo configuration")
	}
	firstModule := first.prog.NewPackage("first", "example.com/first").Module()
	secondModule := second.prog.NewPackage("second", "example.com/second").Module()
	if firstModule.Context().C == secondModule.Context().C {
		t.Fatal("backend sessions share an LLVM context")
	}
}

func TestBackendProgramTemplateOptionalState(t *testing.T) {
	conf := &Config{Goos: "linux", Goarch: "amd64", DisableBoundsChecks: true, PthreadStackSize: 4096}
	template := newBackendProgramTemplate(nil, conf, false, false)
	template.typeSizes = &types.StdSizes{WordSize: 8, MaxAlign: 8}
	template.runtimePackage = types.NewPackage(llssa.PkgRuntime, "runtime")
	template.pythonPackage = types.NewPackage(llssa.PkgPython, "python")
	prog := template.newProgram()
	defer prog.Dispose()
	if prog.Target() == nil {
		t.Fatal("backend program created without a default target")
	}

	coordinator := llssa.NewProgram(nil)
	defer coordinator.Dispose()
	pkg := types.NewPackage("example.com/shared", "shared")
	fset := token.NewFileSet()
	coordinator.SetLinkname("example.com/shared.Entry", "shared_entry")
	coordinator.SetPackageExport("example.com/shared.Entry", "shared_entry")
	coordinator.SetClosureEnvDirective(fset, "example.com/shared.Entry", token.Pos(7))
	coordinator.MarkPackageSyntaxParsed(pkg)
	template.packageSyntax = coordinator.FreezePackageSyntaxState()
	sharedSession, err := template.newSession()
	if err != nil {
		t.Fatal(err)
	}
	shared := sharedSession.prog
	defer shared.Dispose()
	if !shared.PackageSyntaxParsed(pkg) {
		t.Fatal("backend Program lost shared parsed-package state")
	}
	if link, ok := shared.Linkname("example.com/shared.Entry"); !ok || link != "shared_entry" {
		t.Fatalf("shared linkname = (%q, %v), want (shared_entry, true)", link, ok)
	}
	if export, ok := shared.PackageExport("example.com/shared.Entry"); !ok || export != "shared_entry" {
		t.Fatalf("shared export = (%q, %v), want (shared_entry, true)", export, ok)
	}
	if !shared.HasClosureEnvDirective(fset, "example.com/shared.Entry", token.Pos(7)) {
		t.Fatal("backend Program lost shared closure environment directive")
	}
}
