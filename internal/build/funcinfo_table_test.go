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

	"github.com/goplus/llgo/internal/lto"
	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
)

func TestFuncInfoTableMaterializesMetadataWithoutFunctionPointers(t *testing.T) {
	prog := llssa.NewProgram(nil)
	prog.EnableFuncInfoSites(true)
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
		"@__llgo_pcline_table = global ptr null",
		"@__llgo_pcsite_start = global ptr null",
		"@__llgo_pcsite_end = global ptr null",
		"@__llgo_funcinfo_strings = global ptr",
		"@__llgo_funcinfo_string_offsets = global ptr",
		"@__llgo_funcinfo_string_count = global i64 5",
		"@__llgo_funcinfo_hash = global ptr",
		"@__llgo_funcinfo_symbol_index = global ptr",
		"@__llgo_funcinfo_count = global i64 1",
		"@__llgo_funcinfo_symbol_index_count = global i64 1",
		"@__llgo_funcinfo_entry_start = global ptr @__start_llgo_funcinfo_entry",
		"@__llgo_funcinfo_entry_end = global ptr @__stop_llgo_funcinfo_entry",
		"@__llgo_funcinfo_stub_indexes = global ptr null",
		"@__llgo_funcinfo_stub_count = global i64 0",
		"@__llgo_pcline_count = global i64 0",
		"@__llgo_funcinfo_hash_mask = global i64 1",
		"module asm \".section llgo_funcinfo_entry",
		`@"__llgo_funcinfo_table$data" = private unnamed_addr constant [1 x { i16, i16, i16, i16, i16, i16, i32 }]`,
		`@"__llgo_funcinfo_string_offsets$data" = private unnamed_addr constant`,
		`@"__llgo_funcinfo_hash$data" = private unnamed_addr constant [2 x i16]`,
		`@"__llgo_funcinfo_symbol_index$data" = private unnamed_addr constant [1 x { i64, i32 }]`,
		`example.com/p\00`,
		`live\00`,
		`Live\00`,
		`live.go\00`,
		"i32 17",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("funcinfo table IR missing %q:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, `ptr @"example.com/p.live"`) {
		t.Fatalf("funcinfo table must not reference function pointers:\n%s", ir)
	}
}

func TestFuncInfoTableMaterializesEntrySites(t *testing.T) {
	prog := llssa.NewProgram(nil)
	src := prog.NewPackage("example.com/p", "example.com/p")
	src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	src.EmitFuncInfo("example.com/p.missing", "example.com/p.Missing", "missing.go", 19, 1)
	liveFn := src.NewFunc("example.com/p.live", llssa.NoArgsNoRet, llssa.InC)
	liveFn.MakeBody(1).Return()
	otherFn := src.NewFunc("example.com/p.other", llssa.NoArgsNoRet, llssa.InC)
	otherFn.MakeBody(1).Return()
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	emitFuncInfoEntrySites(ctx, src)
	srcIR := src.String()
	for _, want := range []string{
		"call void asm sideeffect",
		".pushsection llgo_funcinfo_entry",
		".Lllgo_funcinfo_entry_anchor_",
		".quad .Lllgo_funcinfo_entry_anchor_",
		".quad 0x",
	} {
		if !strings.Contains(srcIR, want) {
			t.Fatalf("package entry site IR missing %q:\n%s", want, srcIR)
		}
	}
	for _, bad := range []string{
		`.quad \22example.com/p.live\22`,
		`.quad \22example.com/p.other\22`,
		`.quad \22example.com/p.missing\22`,
	} {
		if strings.Contains(srcIR, bad) {
			t.Fatalf("package entry site IR should not contain %q:\n%s", bad, srcIR)
		}
	}

	records := collectFuncInfo([]Package{{LPkg: src}})
	entry := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{funcInfo: records})
	ir := entry.LPkg.String()
	for _, want := range []string{
		"@__llgo_funcinfo_entry_start = global ptr @__start_llgo_funcinfo_entry",
		"@__llgo_funcinfo_entry_end = global ptr @__stop_llgo_funcinfo_entry",
		"module asm \".section llgo_funcinfo_entry",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("funcinfo entry table IR missing %q:\n%s", want, ir)
		}
	}

	ltoCtx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
			LTO:       lto.Full,
		},
	}
	ltoEntry := genMainModule(ltoCtx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{funcInfo: records})
	ltoIR := ltoEntry.LPkg.String()
	for _, want := range []string{
		"@__llgo_funcinfo_entry_start = global ptr @__start_llgo_funcinfo_entry",
		"@__llgo_funcinfo_entry_end = global ptr @__stop_llgo_funcinfo_entry",
		"module asm \".section llgo_funcinfo_entry",
	} {
		if !strings.Contains(ltoIR, want) {
			t.Fatalf("full LTO funcinfo table IR missing %q:\n%s", want, ltoIR)
		}
	}
}

func TestFuncInfoTableSitesDisabledKeepsTables(t *testing.T) {
	prog := llssa.NewProgram(nil)
	src := prog.NewPackage("example.com/p", "example.com/p")
	src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	src.EmitPCLineInfo(0x1234, "example.com/p.live", "call.go", 23, 5)
	liveFn := src.NewFunc("example.com/p.live", llssa.NoArgsNoRet, llssa.InC)
	liveFn.MakeBody(1).Return()
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(false)

	emitFuncInfoEntrySites(ctx, src)
	emitFuncInfoStubSites(ctx, src)
	srcIR := src.String()
	for _, bad := range []string{"llgo_funcinfo_entry", "llgo_funcinfo_stubsite", "call void asm sideeffect"} {
		if strings.Contains(srcIR, bad) {
			t.Fatalf("sites disabled: package IR should not contain %q:\n%s", bad, srcIR)
		}
	}

	records := collectFuncInfo([]Package{{LPkg: src}})
	pcLines := collectPCLineInfo([]Package{{LPkg: src}})
	entry := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{funcInfo: records, pcLineInfo: pcLines})
	ir := entry.LPkg.String()
	// The metadata tables must still materialize...
	for _, want := range []string{
		"@__llgo_funcinfo_table = global ptr",
		"@__llgo_funcinfo_count = global",
		"@__llgo_pcline_table = global ptr",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("sites disabled: funcinfo table IR missing %q:\n%s", want, ir)
		}
	}
	// ...while the site sections and their boundary symbols must not.
	for _, bad := range []string{
		"@__start_llgo_funcinfo_entry",
		"@__start_llgo_funcinfo_stubsite",
		"@__start_llgo_pcline",
		"module asm \".section llgo_",
	} {
		if strings.Contains(ir, bad) {
			t.Fatalf("sites disabled: funcinfo table IR should not contain %q:\n%s", bad, ir)
		}
	}
}

func TestFuncInfoTableMaterializesClosureStubIndexes(t *testing.T) {
	prog := llssa.NewProgram(nil)
	src := prog.NewPackage("example.com/p", "example.com/p")
	src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	src.EmitFuncInfo("example.com/p.other", "example.com/p.Other", "other.go", 23, 1)
	stubFn := src.NewFunc(closureStubPrefix+"example.com/p.live", llssa.NoArgsNoRet, llssa.InC)
	stubFn.MakeBody(1).Return()
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	emitFuncInfoStubSites(ctx, src)
	srcIR := src.String()
	for _, want := range []string{
		"call void asm sideeffect",
		".pushsection llgo_funcinfo_stubsite",
		".Lllgo_funcinfo_stubsite_anchor_",
		".quad .Lllgo_funcinfo_stubsite_anchor_",
		".quad 0x",
	} {
		if !strings.Contains(srcIR, want) {
			t.Fatalf("package stub site IR missing %q:\n%s", want, srcIR)
		}
	}
	if strings.Contains(srcIR, `.quad \22__llgo_stub.example.com/p.live\22`) {
		t.Fatalf("package stub site must not reference stub function symbols:\n%s", srcIR)
	}

	records := collectFuncInfo([]Package{{LPkg: src}})
	stubs := collectFuncInfoStubRecords([]Package{{LPkg: src}}, records)
	if len(stubs) != 1 || records[stubs[0].funcIndex-1].symbol != "example.com/p.live" ||
		stubs[0].symbol != closureStubPrefix+"example.com/p.live" {
		t.Fatalf("stub indexes = %+v for records %+v, want live", stubs, records)
	}

	entry := genMainModule(ctx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{funcInfo: records, funcInfoStubs: stubs})
	ir := entry.LPkg.String()
	for _, want := range []string{
		"@__llgo_funcinfo_stub_indexes = global ptr",
		"@__llgo_funcinfo_stub_count = global i64 1",
		"@__llgo_funcinfo_symbol_index = global ptr",
		"@__llgo_funcinfo_symbol_index_count = global i64 2",
		"@__llgo_funcinfo_stubsite_start = global ptr @__start_llgo_funcinfo_stubsite",
		"@__llgo_funcinfo_stubsite_end = global ptr @__stop_llgo_funcinfo_stubsite",
		`@"__llgo_funcinfo_stub_indexes$data" = private unnamed_addr constant [1 x i32]`,
		"@__llgo_funcinfo_count = global i64 2",
		"module asm \".section llgo_funcinfo_stubsite",
		".quad 0",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("funcinfo stub index table IR missing %q:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, closureStubPrefix+"example.com/p.live\\00") {
		t.Fatalf("stub index table should not add stub symbol strings:\n%s", ir)
	}

	ltoCtx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
			LTO:       lto.Full,
		},
	}
	ltoEntry := genMainModule(ltoCtx, llssa.PkgRuntime, &packages.Package{
		PkgPath:    "example.com/main",
		ExportFile: "main.a",
	}, &genConfig{funcInfo: records, funcInfoStubs: stubs})
	ltoIR := ltoEntry.LPkg.String()
	for _, want := range []string{
		"@__llgo_funcinfo_stubsite_start = global ptr @__start_llgo_funcinfo_stubsite",
		"@__llgo_funcinfo_stubsite_end = global ptr @__stop_llgo_funcinfo_stubsite",
		"module asm \".section llgo_funcinfo_stubsite",
	} {
		if !strings.Contains(ltoIR, want) {
			t.Fatalf("full LTO funcinfo stub site table IR missing %q:\n%s", want, ltoIR)
		}
	}
}

func TestFuncInfoTableMaterializesPCLineMetadata(t *testing.T) {
	prog := llssa.NewProgram(nil)
	prog.EnableFuncInfoSites(true)
	src := prog.NewPackage("example.com/p", "example.com/p")
	src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	src.EmitPCLineInfo(0x1234, "example.com/p.live", "call.go", 23, 5)
	src.EmitPCLineInfo(0x5678, "example.com/p.missing", "missing.go", 99, 1)

	records := collectFuncInfo([]Package{{LPkg: src}})
	pcLines := collectPCLineInfo([]Package{{LPkg: src}})
	if len(records) != 1 {
		t.Fatalf("collectFuncInfo returned %d records, want 1", len(records))
	}
	if len(pcLines) != 2 {
		t.Fatalf("collectPCLineInfo returned %d records, want 2", len(pcLines))
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
	}, &genConfig{funcInfo: records, pcLineInfo: pcLines})
	ir := entry.LPkg.String()
	for _, want := range []string{
		"@__llgo_pcline_table = global ptr",
		"@__llgo_pcsite_start = global ptr @__start_llgo_pcline",
		"@__llgo_pcsite_end = global ptr @__stop_llgo_pcline",
		"@__llgo_pcline_count = global i64 1",
		"@__llgo_funcinfo_string_count = global i64 6",
		"module asm \".section llgo_pcline",
		`@"__llgo_pcline_table$data" = private unnamed_addr constant [1 x { i64, i32, i32, i32 }]`,
		"i64 4660",
		"i32 23",
		`call.go\00`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("pcline table IR missing %q:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, "missing.go") || strings.Contains(ir, "i64 22136") {
		t.Fatalf("pcline table should drop records without matching function metadata:\n%s", ir)
	}
	if strings.Contains(ir, `ptr @"example.com/p.live"`) {
		t.Fatalf("pcline table must not reference function pointers:\n%s", ir)
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
		"@__llgo_pcline_table = global ptr null",
		"@__llgo_pcsite_start = global ptr null",
		"@__llgo_pcsite_end = global ptr null",
		"@__llgo_funcinfo_strings = global ptr null",
		"@__llgo_funcinfo_string_offsets = global ptr null",
		"@__llgo_funcinfo_string_count = global i64 0",
		"@__llgo_funcinfo_hash = global ptr null",
		"@__llgo_funcinfo_symbol_index = global ptr null",
		"@__llgo_funcinfo_count = global i64 0",
		"@__llgo_funcinfo_symbol_index_count = global i64 0",
		"@__llgo_funcinfo_entry_start = global ptr null",
		"@__llgo_funcinfo_entry_end = global ptr null",
		"@__llgo_funcinfo_stub_indexes = global ptr null",
		"@__llgo_funcinfo_stub_count = global i64 0",
		"@__llgo_pcline_count = global i64 0",
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

// TestFuncInfoTableEmissionMatrix sweeps the OS / pointer-size / content
// combinations so both the ELF and Mach-O directive branches, the 32-bit
// pointer directives, and the empty-table initializers stay covered on every
// platform's test run.
func TestFuncInfoTableEmissionMatrix(t *testing.T) {
	cases := []struct {
		goos, goarch string
		empty        bool
	}{
		{"linux", "amd64", false},
		{"darwin", "arm64", false},
		{"linux", "386", false},
		{"linux", "amd64", true},
		{"darwin", "arm64", true},
	}
	for _, c := range cases {
		name := c.goos + "/" + c.goarch
		if c.empty {
			name += "/empty"
		}
		t.Run(name, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: c.goos, GOARCH: c.goarch})
			prog.EnableFuncInfoMetadata(true)
			prog.EnableFuncInfoSites(true)
			src := prog.NewPackage("example.com/p", "example.com/p")
			if !c.empty {
				src.EmitFuncInfo(`example.com/p.we$ird"sym`, "example.com/p.Live", "live.go", 17, 3)
				src.EmitFuncInfo("example.com/p.other", "example.com/p.Other", "other.go", 5, 1)
				src.EmitPCLineInfo(0x1234, `example.com/p.we$ird"sym`, "call.go", 23, 5)
				fn := src.NewFunc(`example.com/p.we$ird"sym`, llssa.NoArgsNoRet, llssa.InGo)
				fn.MakeBody(1).Return()
				stub := src.NewFunc(`__llgo_stub.example.com/p.we$ird"sym`, llssa.NoArgsNoRet, llssa.InGo)
				stub.MakeBody(1).Return()
			}
			ctx := &context{
				prog: prog,
				buildConf: &Config{
					BuildMode: BuildModeExe,
					Goos:      c.goos,
					Goarch:    c.goarch,
				},
			}
			records := collectFuncInfo([]Package{{LPkg: src}})
			pcLines := collectPCLineInfo([]Package{{LPkg: src}})
			stubs := collectFuncInfoStubRecords([]Package{{LPkg: src}}, records)
			emitFuncInfoTable(ctx, src, records, pcLines, stubs)
			emitFuncInfoEntrySites(ctx, src)
			emitFuncInfoStubSites(ctx, src)
			ir := src.String()
			if c.empty {
				if !strings.Contains(ir, "__llgo_funcinfo_count") {
					t.Fatalf("missing empty table globals:\n%s", ir)
				}
				return
			}
			if !strings.Contains(ir, "__llgo_funcinfo_table") {
				t.Fatalf("missing table:\n%s", ir)
			}
			if c.goos == "darwin" && !strings.Contains(ir, "live_support") {
				t.Fatalf("darwin sections must be live_support:\n%s", ir)
			}
			if c.goos == "linux" && !strings.Contains(ir, "pushsection llgo_funcinfo_entry") {
				t.Fatalf("missing elf entry section:\n%s", ir)
			}
		})
	}
}

func TestAsmQuoteELFSymbol(t *testing.T) {
	cases := map[string]string{
		`plain`:      `"plain"`,
		`we$ird`:     `"we$$ird"`,
		`q"uote`:     `"q\"uote"`,
		`back\slash`: `"back\\slash"`,
	}
	for in, want := range cases {
		if got := asmQuoteELFSymbol(in); got != want {
			t.Fatalf("quote(%q) = %q, want %q", in, got, want)
		}
	}
}

// Empty encoded tables must materialize null initializers (the ~20-line
// branch in emitFuncInfoTable that only fires for funcinfo-less programs).
func TestFuncInfoTableEmptyEncodedInitializers(t *testing.T) {
	prog := llssa.NewProgram(nil)
	prog.EnableFuncInfoMetadata(true)
	src := prog.NewPackage("example.com/p", "example.com/p")
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	emitFuncInfoTable(ctx, src, nil, nil, nil)
	ir := src.String()
	for _, want := range []string{
		"@__llgo_funcinfo_table = global ptr null",
		"@__llgo_pcline_table = global ptr null",
		"@__llgo_funcinfo_count = global i64 0",
		"@__llgo_fp_chain = global i8 1",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %q in:\n%s", want, ir)
		}
	}
}

// Targets without the frame-pointer attribute must declare the chain
// broken so the runtime never attempts a physical walk there.
func TestFuncInfoTableFPChainOff(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "amd64"})
	prog.EnableFuncInfoMetadata(true)
	src := prog.NewPackage("example.com/p", "example.com/p")
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "windows",
			Goarch:    "amd64",
		},
	}
	emitFuncInfoTable(ctx, src, nil, nil, nil)
	if ir := src.String(); !strings.Contains(ir, "@__llgo_fp_chain = global i8 0") {
		t.Fatalf("missing fp_chain=0 in:\n%s", ir)
	}
}
