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
	"errors"
	"go/importer"
	"go/types"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

func TestNewBackendSessionCreatesIndependentLLVMState(t *testing.T) {
	coordinator := llvmgen.NewProgram(&llvmgen.Target{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	defer coordinator.Dispose()
	ctx := &context{
		prog: coordinator,
		buildConf: &Config{
			Goos:   runtime.GOOS,
			Goarch: runtime.GOARCH,
		},
	}
	first := ctx.newBackendSession()
	defer first.prog.Dispose()
	second := ctx.newBackendSession()
	defer second.prog.Dispose()
	if first.transformer == nil || second.transformer == nil {
		t.Fatal("backend session missing C ABI transformer")
	}
	firstModule := first.prog.NewPackage("first", "example.com/first").Module()
	secondModule := second.prog.NewPackage("second", "example.com/second").Module()
	if firstModule.Context().C == secondModule.Context().C {
		t.Fatal("backend sessions share an LLVM context")
	}
}

func addBackendProgramPackage(ctx *context, prog llvmgen.Program, path string) *aPackage {
	source := &packages.Package{ID: path, PkgPath: path}
	pkg := &aPackage{Package: source, LPkg: prog.NewPackage("p", path)}
	ctx.pkgs[source] = pkg
	return pkg
}

func TestBackendProgramsStayOwnedByPackagesUntilExplicitDispose(t *testing.T) {
	coordinator := llvmgen.NewProgram(nil)
	defer coordinator.Dispose()
	ctx := &context{
		prog: coordinator,
		pkgs: make(map[*packages.Package]Package),
	}
	coordinatorPkg := addBackendProgramPackage(ctx, coordinator, "example.com/coordinator")
	pkgs := make([]*aPackage, 2)
	paths := []string{"example.com/p0", "example.com/p1"}
	for i, path := range paths {
		prog := llvmgen.NewProgram(nil)
		pkgs[i] = addBackendProgramPackage(ctx, prog, path)
	}
	for _, pkg := range pkgs {
		if pkg.LPkg == nil || pkg.LPkg.Module().IsNil() {
			t.Fatal("retained package module was cleared before whole-program use")
		}
	}

	ctx.disposeBackendPrograms()
	for _, pkg := range pkgs {
		if pkg.LPkg != nil {
			t.Fatal("isolated package still references LPkg after dispose")
		}
	}
	if coordinatorPkg.LPkg == nil {
		t.Fatal("backend disposal cleared coordinator package")
	}
	ctx.disposeBackendPrograms()
}

func TestBackendAbiTypesFollowLinkedPackageOrder(t *testing.T) {
	runtimePkg, err := importer.For("source", nil).Import(llvmgen.PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	newBackend := func(path, name string, raw types.Type) (llvmgen.Program, *aPackage) {
		prog := llvmgen.NewProgram(nil)
		prog.SetRuntime(runtimePkg)
		pkg := prog.NewPackage("p", path)
		pkg.RegisterAbiTypes([]llvmgen.AbiTypeInfo{{Name: name, Raw: raw}})
		return prog, &aPackage{LPkg: pkg}
	}

	coordinator := llvmgen.NewProgram(nil)
	defer coordinator.Dispose()
	firstProg, first := newBackend("example.com/first", "z.type", types.Typ[types.Int])
	defer firstProg.Dispose()
	secondProg, second := newBackend("example.com/second", "a.type", types.Typ[types.String])
	defer secondProg.Dispose()
	duplicate := &aPackage{LPkg: firstProg.NewPackage("duplicate", "example.com/duplicate")}
	coordinatorPkg := &aPackage{LPkg: coordinator.NewPackage("coordinator", "example.com/coordinator")}

	ctx := &context{prog: coordinator}
	infos := ctx.backendAbiTypes([]Package{nil, {}, coordinatorPkg, first, duplicate, second})
	if len(infos) != 2 || infos[0].Name != "z.type" || infos[1].Name != "a.type" {
		t.Fatalf("backend ABI types = %#v, want linked order [z.type a.type]", infos)
	}
}

func TestBackendProgramsReleaseOnErrorAndPanic(t *testing.T) {
	coordinator := llvmgen.NewProgram(nil)
	defer coordinator.Dispose()
	ctx := &context{
		prog: coordinator,
		pkgs: make(map[*packages.Package]Package),
	}
	retain := func(path string) *aPackage {
		prog := llvmgen.NewProgram(nil)
		return addBackendProgramPackage(ctx, prog, path)
	}

	errorPkg := retain("example.com/error")
	wantErr := errors.New("link failed")
	runError := func() (err error) {
		defer ctx.disposeBackendPrograms()
		return wantErr
	}
	if err := runError(); !errors.Is(err, wantErr) {
		t.Fatalf("error unwind = %v, want %v", err, wantErr)
	}
	if errorPkg.LPkg != nil {
		t.Fatal("error unwind retained a backend Program")
	}

	panicPkg := retain("example.com/panic")
	func() {
		defer func() {
			if got := recover(); got != "link panic" {
				t.Fatalf("panic unwind recovered %v", got)
			}
		}()
		defer ctx.disposeBackendPrograms()
		panic("link panic")
	}()
	if panicPkg.LPkg != nil {
		t.Fatal("panic unwind retained a backend Program")
	}
}

func TestReleaseSingleExecutableBackendPreservesOtherConsumers(t *testing.T) {
	for _, test := range []struct {
		name      string
		mode      Mode
		buildMode BuildMode
		roots     int
		release   bool
	}{
		{"single executable", ModeBuild, BuildModeExe, 1, true},
		{"multiple executables", ModeBuild, BuildModeExe, 2, false},
		{"shared library header", ModeBuild, BuildModeCShared, 1, false},
		{"archive header", ModeBuild, BuildModeCArchive, 1, false},
		{"generated module", ModeGen, BuildModeExe, 1, false},
		{"single test executable", ModeTest, BuildModeExe, 1, true},
		{"test DAG", ModeTest, BuildModeExe, 2, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			coordinator := llvmgen.NewProgram(nil)
			defer coordinator.Dispose()
			backend := llvmgen.NewProgram(nil)
			defer backend.Dispose()
			ctx := &context{
				prog:      coordinator,
				pkgs:      make(map[*packages.Package]Package),
				initial:   make([]*packages.Package, test.roots),
				buildConf: &Config{Mode: test.mode, BuildMode: test.buildMode},
			}
			entry := addBackendProgramPackage(ctx, coordinator, "example.com/main")
			dep := addBackendProgramPackage(ctx, backend, "example.com/dep")
			ctx.releaseSingleExecutableBackend()
			for _, pkg := range []*aPackage{entry, dep} {
				if (pkg.LPkg == nil) != test.release {
					t.Fatalf("%s backend released = %v, want %v", pkg.PkgPath, pkg.LPkg == nil, test.release)
				}
				if !test.release && pkg.LPkg.Module().IsNil() {
					t.Fatalf("%s lost module needed by later consumers", pkg.PkgPath)
				}
			}
			// Deferred cleanup must remain safe after the early release.
			ctx.disposeBackendPrograms()
		})
	}
}
