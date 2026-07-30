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
	"go/types"
	"testing"

	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/packages"
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

	validTypes := types.NewPackage("example.com/valid", "valid")
	valid := &packages.Package{PkgPath: "example.com/valid", Types: validTypes}
	duplicate := &packages.Package{PkgPath: "example.com/duplicate", Types: validTypes}
	missingTypes := &packages.Package{PkgPath: "example.com/missing"}
	illTyped := &packages.Package{PkgPath: "example.com/ill", Types: types.NewPackage("example.com/ill", "ill"), IllTyped: true}
	inputs := collectBackendProgramInputs([]*packages.Package{missingTypes, illTyped, valid, duplicate})
	if len(inputs) != 1 || inputs[0].pkg != validTypes {
		t.Fatalf("backend inputs = %#v, want one deduplicated package", inputs)
	}

	patched := backendProgramTemplate{}
	appendPatchedBackendInputs(&patched, cl.Patches{
		"example.com/missing": {},
		"example.com/other":   {},
	}, packages.NewDeduper())
	if len(patched.inputs) != 0 {
		t.Fatalf("missing patched packages produced inputs: %#v", patched.inputs)
	}
}
