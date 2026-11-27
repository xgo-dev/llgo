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

package exportdata_test

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/types"
)

func TestExportDataV3RoundTripLLGoMetadata(t *testing.T) {
	const src = `package extpkg

const LLGoPackage = "py.numpy"

//llgo:skip SkipA SkipB
type marker int

//llgo:link CAdd C.add
//go:noinline
//go:noescape
func CAdd(a, b int32) int32

//go:wasmimport env host_log
func HostLog(ptr, n int32)

//go:wasmexport exported_step
func Step() {}

//llgo:env
func EnvFunc() {}

//export my_c_export
func my_c_export() {}

type Handle struct{}

//go:nointerface
//llgo:link (*Handle).Close C.handle_close
//llgo:env
func (*Handle) Close()

//llgointernal:tls
//llgo:link GlobalSlot C.global_slot
var GlobalSlot uintptr

//llgointernal:gls
var RoutineSlot uintptr
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "extpkg.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	pkg, err := (&types.Config{}).Check("example.com/extpkg", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	pkg.SetSkipAll(true)

	var buf bytes.Buffer
	if err := exportdata.Write(&buf, fset, pkg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	imports := make(map[string]*types.Package)
	got, err := exportdata.Read(bytes.NewReader(buf.Bytes()), token.NewFileSet(), imports, pkg.Path())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.Kind() != directive.PkgPyModule || got.KindParam() != "numpy" || !got.SkipAll() {
		t.Fatalf("pkg metadata = (Kind=%v, Param=%q, SkipAll=%v)", got.Kind(), got.KindParam(), got.SkipAll())
	}
	if !got.Skipped("SkipA") || !got.Skipped("SkipB") || got.Skipped("Other") {
		t.Fatalf("pkg skips = %v", got.SkipNames())
	}

	cAdd := got.Scope().Lookup("CAdd").(*types.Func)
	if cAdd.LinkName() != "C.add" || !cAdd.NoInline() || !cAdd.NoEscape() {
		t.Fatalf("CAdd = (LinkName=%q NoInline=%v NoEscape=%v)", cAdd.LinkName(), cAdd.NoInline(), cAdd.NoEscape())
	}

	hostLog := got.Scope().Lookup("HostLog").(*types.Func)
	if wi := hostLog.WasmImport(); wi == nil || wi.Module != "env" || wi.Name != "host_log" {
		t.Fatalf("HostLog.WasmImport() = %+v", wi)
	}

	step := got.Scope().Lookup("Step").(*types.Func)
	if step.WasmExport() != "exported_step" {
		t.Fatalf("Step.WasmExport() = %q", step.WasmExport())
	}

	envFn := got.Scope().Lookup("EnvFunc").(*types.Func)
	if !envFn.HasClosureEnv() {
		t.Fatal("EnvFunc.HasClosureEnv() = false after round-trip")
	}

	expFn := got.Scope().Lookup("my_c_export").(*types.Func)
	if expFn.ExportName() != "my_c_export" || expFn.LinkName() != "my_c_export" {
		t.Fatalf("my_c_export = (Export=%q Link=%q)", expFn.ExportName(), expFn.LinkName())
	}

	handle := got.Scope().Lookup("Handle").(*types.TypeName).Type().(*types.Named)
	if handle.NumMethods() != 1 {
		t.Fatalf("Handle.NumMethods() = %d, want 1", handle.NumMethods())
	}
	closeM := handle.Method(0)
	if !closeM.NoInterface() || closeM.LinkName() != "C.handle_close" || !closeM.HasClosureEnv() {
		t.Fatalf("Handle.Close = (NoInterface=%v Link=%q HasClosureEnv=%v)",
			closeM.NoInterface(), closeM.LinkName(), closeM.HasClosureEnv())
	}

	gSlot := got.Scope().Lookup("GlobalSlot").(*types.Var)
	if gSlot.Locality() != directive.LocalityThread || gSlot.LinkName() != "C.global_slot" {
		t.Fatalf("GlobalSlot = (Locality=%v Link=%q)", gSlot.Locality(), gSlot.LinkName())
	}
	rSlot := got.Scope().Lookup("RoutineSlot").(*types.Var)
	if rSlot.Locality() != directive.LocalityGoroutine {
		t.Fatalf("RoutineSlot.Locality() = %v", rSlot.Locality())
	}
}

func TestExportDataRejectsLegacyV2Payload(t *testing.T) {
	// Header: magic "LLGO" + version 2 (0x0002).
	legacy := []byte{'L', 'L', 'G', 'O', 2, 0}
	if _, err := exportdata.Read(bytes.NewReader(legacy), token.NewFileSet(), map[string]*types.Package{}, "p"); err == nil {
		t.Fatal("expected legacy v2 payload to be rejected")
	}
}
