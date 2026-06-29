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
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"

	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
)

func TestFuncInfoTableMaterializesMetadataWithoutFunctionPointers(t *testing.T) {
	prog := llssa.NewProgram(nil)
	src := prog.NewPackage("example.com/p", "example.com/p")
	src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	src.EmitFuncInfo("example.com/p.live", "example.com/p.LiveDuplicate", "dup.go", 19, 1)

	records := collectFuncInfo([]Package{{LPkg: src}})
	if len(records) != 1 {
		t.Fatalf("collectFuncInfo returned %d records, want 1", len(records))
	}
	if got := records[0]; got.symbol != "example.com/p.live" || got.name != "example.com/p.Live" || got.file != "live.go" || got.line != 17 || got.column != 3 {
		t.Fatalf("unexpected record: %+v", got)
	}

	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	entry := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{funcInfo: records})
	ir := entry.LPkg.String()
	for _, want := range []string{
		"@__llgo_funcinfo_table = global ptr",
		"@__llgo_funcinfo_strings = global ptr",
		"@__llgo_funcinfo_hash = global ptr",
		"@__llgo_funcinfo_count = global i64 1",
		"@__llgo_funcinfo_hash_mask = global i64 1",
		`@"__llgo_funcinfo_table$data" = private unnamed_addr constant [1 x { i32, i32, i32, i32, i32 }]`,
		`@"__llgo_funcinfo_strings$data" = private unnamed_addr constant [47 x i8]`,
		`@"__llgo_funcinfo_hash$data" = private unnamed_addr constant [2 x i32]`,
		`example.com/p.live\00`,
		`example.com/p.Live\00`,
		`live.go\00`,
		"i32 17",
		"i32 3",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("funcinfo table IR missing %q:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, `ptr @"example.com/p.live"`) {
		t.Fatalf("funcinfo table must not reference function pointers:\n%s", ir)
	}
}

func TestPrepareFuncInfoTableRecordsFiltersLiveSymbols(t *testing.T) {
	records := []funcInfoRecord{
		{symbol: "dead", name: "dead"},
		{symbol: "live", name: "live"},
	}
	if got := prepareFuncInfoTableRecords(nil, nil); got != nil {
		t.Fatalf("empty records = %+v, want nil", got)
	}
	if got := prepareFuncInfoTableRecords(records, nil); len(got) != 2 {
		t.Fatalf("nil live set kept %d records, want 2", len(got))
	}
	got := prepareFuncInfoTableRecords(records, map[string]none{"live": {}})
	if len(got) != 1 || got[0].symbol != "live" {
		t.Fatalf("filtered records = %+v, want live only", got)
	}
	if got := prepareFuncInfoTableRecords(records, map[string]none{}); got != nil {
		t.Fatalf("empty live set = %+v, want nil", got)
	}
}

func TestFuncInfoTablePoolsRepeatedStrings(t *testing.T) {
	prog := llssa.NewProgram(nil)
	records := []funcInfoRecord{
		{symbol: "example.com/p.a", name: "example.com/p.A", file: "shared.go", line: 10},
		{symbol: "example.com/p.b", name: "example.com/p.B", file: "shared.go", line: 20},
	}
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	entry := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{funcInfo: records})
	if got := strings.Count(entry.LPkg.String(), `shared.go\00`); got != 1 {
		t.Fatalf("shared file string emitted %d times, want 1", got)
	}
}

func TestFuncInfoTableEmptyDefinitions(t *testing.T) {
	prog := llssa.NewProgram(nil)
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	entry := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{})
	ir := entry.LPkg.String()
	for _, want := range []string{
		"@__llgo_funcinfo_table = global ptr null",
		"@__llgo_funcinfo_strings = global ptr null",
		"@__llgo_funcinfo_hash = global ptr null",
		"@__llgo_funcinfo_count = global i64 0",
		"@__llgo_funcinfo_hash_mask = global i64 0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("empty funcinfo table IR missing %q:\n%s", want, ir)
		}
	}
}

func TestFuncInfoTableIgnoresInvalidMetadata(t *testing.T) {
	prog := llssa.NewProgram(nil)
	pkg := prog.NewPackage("example.com/p", "example.com/p")
	mod := pkg.Module()
	ctx := mod.Context()
	i32 := ctx.Int32Type()
	mdstr := func(s string) llvm.Metadata { return ctx.MDString(s) }
	mdint := func(v uint64) llvm.Metadata {
		return llvm.ConstInt(i32, v, false).ConstantAsMetadata()
	}
	add := func(fields ...llvm.Metadata) {
		mod.AddNamedMetadataOperand(llssa.FuncInfoMetadataName, ctx.MDNode(fields))
	}

	add(mdstr("short"))
	add(mdint(2), mdstr("bad.version"), mdstr("bad.version"), mdstr("bad.go"), mdint(1), mdint(1))
	add(mdint(1), mdint(0), mdstr("bad.symbol"), mdstr("bad.go"), mdint(1), mdint(1))
	add(mdint(1), mdstr(""), mdstr("empty.symbol"), mdstr("empty.go"), mdint(1), mdint(1))

	if got := readFuncInfo(mod); len(got) != 1 || got[0].symbol != "" {
		t.Fatalf("readFuncInfo invalid rows = %+v, want one empty-symbol row", got)
	}
	if got := collectFuncInfo([]Package{nil, {}, {LPkg: pkg}}); len(got) != 0 {
		t.Fatalf("collectFuncInfo invalid rows = %+v, want none", got)
	}

	empty := ctx.NewModule("empty")
	defer empty.Dispose()
	if got := readFuncInfo(empty); got != nil {
		t.Fatalf("readFuncInfo(empty) = %+v, want nil", got)
	}
}
