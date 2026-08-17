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
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
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
			PCLNMode:  PCLNNone, // Keep exact table assertions limited to supplied records.
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
		"@__llgo_funcinfo_symbol_index = hidden global ptr",
		"@__llgo_funcinfo_count = global i64 1",
		"@__llgo_funcinfo_symbol_index_count = hidden global i64 1",
		"@__llgo_funcinfo_entry_start = global ptr @__start_llgo_funcinfo_entry",
		"@__llgo_funcinfo_entry_end = global ptr @__stop_llgo_funcinfo_entry",
		"@__llgo_pcline_count = global i64 0",
		"@__llgo_funcinfo_hash_mask = global i64 1",
		"module asm \".section llgo_funcinfo_entry",
		`@"__llgo_funcinfo_table$data" = private unnamed_addr constant [1 x { i32, i32, i32, i32, i32, i32, i32 }]`,
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

func TestFuncInfoSiteLayoutArgs(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	prog.EnableFuncInfoSites(true)
	ctx := &context{prog: prog, buildConf: &Config{
		BuildMode: BuildModeExe,
		Goos:      "linux",
		Goarch:    "amd64",
	}}
	dir := t.TempDir()
	args, cleanup, err := funcInfoSiteLayoutArgs(ctx, filepath.Join(dir, "app"))
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 || !strings.HasPrefix(args[0], "-Wl,-T,") {
		t.Fatalf("layout args = %#v", args)
	}
	script := strings.TrimPrefix(args[0], "-Wl,-T,")
	raw, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"llgo_funcinfo_entry", "INSERT BEFORE .bss"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("linker script missing %q:\n%s", want, raw)
		}
	}
	cleanup()
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Fatalf("linker script was not removed: %v", err)
	}

	ctx.buildConf.Goos = "darwin"
	args, cleanup, err = funcInfoSiteLayoutArgs(ctx, filepath.Join(dir, "app"))
	if err != nil || len(args) != 0 {
		t.Fatalf("Darwin layout args = %#v, %v", args, err)
	}
	cleanup()
}

func TestFuncInfoSiteLayoutArgsEligibilityAndCreateFailure(t *testing.T) {
	newContext := func() (*context, func()) {
		prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
		prog.EnableFuncInfoSites(true)
		return &context{prog: prog, buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		}}, prog.Dispose
	}
	tests := []struct {
		name string
		edit func(*context)
	}{
		{"nil context", func(*context) {}},
		{"nil config", func(ctx *context) { ctx.buildConf = nil }},
		{"cross target", func(ctx *context) { ctx.buildConf.Target = "wasm32-wasi" }},
		{"non executable", func(ctx *context) { ctx.buildConf.BuildMode = BuildModeCArchive }},
		{"sites disabled", func(ctx *context) { ctx.prog.EnableFuncInfoSites(false) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, dispose := newContext()
			defer dispose()
			if test.name == "nil context" {
				ctx = nil
			} else {
				test.edit(ctx)
			}
			args, cleanup, err := funcInfoSiteLayoutArgs(ctx, filepath.Join(t.TempDir(), "app"))
			defer cleanup()
			if err != nil || len(args) != 0 {
				t.Fatalf("layout args = %#v, %v", args, err)
			}
		})
	}

	ctx, dispose := newContext()
	defer dispose()
	missing := filepath.Join(t.TempDir(), "missing", "app")
	if _, cleanup, err := funcInfoSiteLayoutArgs(ctx, missing); err == nil {
		cleanup()
		t.Fatal("layout script was created in a missing directory")
	}
}

func TestRuntimeEntrySiteSectionInfo(t *testing.T) {
	tests := []struct {
		name string
		ctx  *context
		want string
	}{
		{name: "nil context", want: "__DATA,__llgo_fie"},
		{name: "darwin without LTO", ctx: &context{buildConf: &Config{Goos: "darwin", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded, LTO: lto.Off}}, want: "__DATA,__llgo_fie"},
		{name: "darwin embedded executable full LTO", ctx: &context{buildConf: &Config{Goos: "darwin", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded, LTO: lto.Full}}, want: "__LLGO,__llgo_fie"},
		{name: "darwin embedded executable thin LTO", ctx: &context{buildConf: &Config{Goos: "darwin", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded, LTO: lto.Thin}}, want: "__LLGO,__llgo_fie"},
		{name: "darwin external full LTO", ctx: &context{buildConf: &Config{Goos: "darwin", BuildMode: BuildModeExe, PCLNMode: PCLNExternal, LTO: lto.Full}}, want: "__DATA,__llgo_fie"},
		{name: "darwin c-shared full LTO", ctx: &context{buildConf: &Config{Goos: "darwin", BuildMode: BuildModeCShared, PCLNMode: PCLNEmbedded, LTO: lto.Full}}, want: "__DATA,__llgo_fie"},
		{name: "darwin c-archive full LTO", ctx: &context{buildConf: &Config{Goos: "darwin", BuildMode: BuildModeCArchive, PCLNMode: PCLNEmbedded, LTO: lto.Full}}, want: "__DATA,__llgo_fie"},
		{name: "linux full LTO", ctx: &context{buildConf: &Config{Goos: "linux", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded, LTO: lto.Full}}, want: "__DATA,__llgo_fie"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := runtimeEntrySiteSectionInfo(test.ctx).machO; got != test.want {
				t.Fatalf("Mach-O entry site section = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFuncInfoTableMaterializesEntrySites(t *testing.T) {
	prog := llssa.NewProgram(nil)
	src := prog.NewPackage("example.com/p", "example.com/p")
	src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	// cl/ssawrap.MakeCallWrapper uses this suffix when an intrinsic is used as
	// a function value. It is a physical function, not a separate PCLN class.
	src.EmitFuncInfo("example.com/p.intrinsic$wrapper", "example.com/p.Live", "live.go", 17, 3)
	src.EmitFuncInfo("example.com/p.missing", "example.com/p.Missing", "missing.go", 19, 1)
	liveFn := src.NewFunc("example.com/p.live", llssa.NoArgsNoRet, llssa.InC)
	liveFn.MakeBody(1).Return()
	intrinsicWrapper := src.NewFunc("example.com/p.intrinsic$wrapper", llssa.NoArgsNoRet, llssa.InC)
	intrinsicWrapper.MakeBody(1).Return()
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
		`.quad \22example.com/p.intrinsic$wrapper\22`,
		`.quad \22example.com/p.other\22`,
		`.quad \22example.com/p.missing\22`,
	} {
		if strings.Contains(srcIR, bad) {
			t.Fatalf("package entry site IR should not contain %q:\n%s", bad, srcIR)
		}
	}
	if got := strings.Count(srcIR, ".pushsection llgo_funcinfo_entry"); got != 2 {
		t.Fatalf("entry site count = %d, want one per physical function with funcinfo:\n%s", got, srcIR)
	}
	for _, symbol := range []string{"example.com/p.live", "example.com/p.intrinsic$wrapper"} {
		if id := uint64Hex(funcInfoSymbolID(symbol)); !strings.Contains(srcIR, ".quad "+id) {
			t.Fatalf("entry site IR missing physical function %q (id %s):\n%s", symbol, id, srcIR)
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
	srcIR := src.String()
	for _, bad := range []string{"llgo_funcinfo_entry", "call void asm sideeffect"} {
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
		"@__start_llgo_pcline",
		"module asm \".section llgo_",
	} {
		if strings.Contains(ir, bad) {
			t.Fatalf("sites disabled: funcinfo table IR should not contain %q:\n%s", bad, ir)
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
			PCLNMode:  PCLNNone, // Keep exact table assertions limited to supplied records.
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
		`@"__llgo_pcline_table$data" = private unnamed_addr constant [1 x { i64, i32, i32, i32, i32 }]`,
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
			PCLNMode:  PCLNNone, // Keep exact table assertions limited to supplied records.
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
		"@__llgo_funcinfo_symbol_index = hidden global ptr null",
		"@__llgo_funcinfo_count = global i64 0",
		"@__llgo_funcinfo_symbol_index_count = hidden global i64 0",
		"@__llgo_funcinfo_entry_start = global ptr null",
		"@__llgo_funcinfo_entry_end = global ptr null",
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
		lto          lto.Mode
		entrySection string
	}{
		{goos: "linux", goarch: "amd64"},
		{goos: "darwin", goarch: "arm64", entrySection: "__DATA,__llgo_fie"},
		{goos: "darwin", goarch: "arm64", lto: lto.Full, entrySection: "__LLGO,__llgo_fie"},
		{goos: "linux", goarch: "386"},
		{goos: "linux", goarch: "amd64", empty: true},
		{goos: "darwin", goarch: "arm64", empty: true, entrySection: "__DATA,__llgo_fie"},
	}
	for _, c := range cases {
		name := c.goos + "/" + c.goarch
		if c.empty {
			name += "/empty"
		}
		if c.lto.Enabled() {
			name += "/" + c.lto.String()
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
			}
			ctx := &context{
				prog: prog,
				buildConf: &Config{
					BuildMode: BuildModeExe,
					Goos:      c.goos,
					Goarch:    c.goarch,
					LTO:       c.lto,
				},
			}
			records := collectFuncInfo([]Package{{LPkg: src}})
			pcLines := collectPCLineInfo([]Package{{LPkg: src}})
			emitFuncInfoTable(ctx, src, records, pcLines)
			emitFuncInfoEntrySites(ctx, src)
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
			if c.entrySection != "" && !strings.Contains(ir, c.entrySection) {
				t.Fatalf("missing Mach-O entry section %q:\n%s", c.entrySection, ir)
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

func TestELFFuncInfoSiteSectionsAllowSharedLibraryRelocations(t *testing.T) {
	if got, want := entrySiteSectionInfo.push(false, "anchor"), `.pushsection llgo_funcinfo_entry,"awo",@progbits,anchor`; got != want {
		t.Fatalf("ELF site section = %q, want %q", got, want)
	}
	if got, want := entrySiteSectionInfo.retain(false), `.section llgo_funcinfo_entry,"awR",@progbits`; got != want {
		t.Fatalf("ELF retained section = %q, want %q", got, want)
	}
}

func TestELFFuncInfoMetadataLinksIntoSharedLibrary(t *testing.T) {
	linker, err := exec.LookPath("ld.lld")
	if err != nil {
		t.Skip("ld.lld is required for the ELF shared-library regression test")
	}

	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	ctx := &context{
		prog: prog,
		buildConf: &Config{
			BuildMode: BuildModeCShared,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}

	src := prog.NewPackage("example.com/p", "example.com/p")
	src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "live.go", 17, 3)
	fn := src.NewFunc("example.com/p.live", llssa.NoArgsNoRet, llssa.InGo)
	fn.MakeBody(1).Return()
	records := collectFuncInfo([]Package{{LPkg: src}})
	emitFuncInfoEntrySites(ctx, src)

	metadata := prog.NewPackage("example.com/runtime", "example.com/runtime")
	emitFuncInfoTable(ctx, metadata, records, nil)

	dir := t.TempDir()
	writeObject := func(name string, mod llvm.Module) string {
		t.Helper()
		buf, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
		if err != nil {
			t.Fatalf("emit %s: %v", name, err)
		}
		defer buf.Dispose()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return path
	}
	srcObj := writeObject("src.o", src.Module())
	metadataObj := writeObject("metadata.o", metadata.Module())
	shared := filepath.Join(dir, "libfuncinfo.so")
	cmd := exec.Command(linker, "-shared", "-z", "text", "--gc-sections", "-o", shared, srcObj, metadataObj)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("link ELF shared library: %v\n%s", err, out)
	}

	ef, err := elf.Open(shared)
	if err != nil {
		t.Fatalf("open linked ELF: %v", err)
	}
	defer ef.Close()
	entry := ef.Section(entrySiteSectionInfo.elf)
	if entry == nil {
		t.Fatalf("linked ELF is missing %s", entrySiteSectionInfo.elf)
	}
	const shfGNURetain elf.SectionFlag = 0x200000
	wantFlags := elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_LINK_ORDER | shfGNURetain
	if got := entry.Flags & wantFlags; got != wantFlags {
		t.Fatalf("%s flags = %v, want at least %v", entry.Name, entry.Flags, wantFlags)
	}
	if entry.Flags&elf.SHF_EXECINSTR != 0 {
		t.Fatalf("%s must not be executable: flags=%v", entry.Name, entry.Flags)
	}

	rela := ef.Section(".rela.dyn")
	if rela == nil {
		t.Fatal("linked ELF is missing .rela.dyn")
	}
	relaData, err := rela.Data()
	if err != nil {
		t.Fatalf("read .rela.dyn: %v", err)
	}
	const rela64Size = 24
	relative := 0
	for off := 0; off+rela64Size <= len(relaData); off += rela64Size {
		relocOff := binary.LittleEndian.Uint64(relaData[off:])
		if relocOff < entry.Addr || relocOff >= entry.Addr+entry.Size {
			continue
		}
		info := binary.LittleEndian.Uint64(relaData[off+8:])
		if typ := elf.R_X86_64(uint32(info)); typ != elf.R_X86_64_RELATIVE {
			t.Fatalf("%s relocation at %#x has type %v, want R_X86_64_RELATIVE", entry.Name, relocOff, typ)
		}
		relative++
	}
	if relative != 3 {
		t.Fatalf("%s has %d relative relocations, want 3", entry.Name, relative)
	}

	symbols, err := ef.Symbols()
	if err != nil {
		t.Fatalf("read ELF symbols: %v", err)
	}
	for _, name := range []string{funcInfoSymbolIndexSymbol, funcInfoSymbolIndexCountSymbol} {
		found := false
		for _, sym := range symbols {
			if sym.Name != name {
				continue
			}
			found = true
			if vis := elf.ST_VISIBILITY(sym.Other); vis != elf.STV_HIDDEN {
				t.Fatalf("%s visibility = %v, want hidden", name, vis)
			}
		}
		if !found {
			t.Fatalf("linked ELF is missing %s", name)
		}
	}
	dynamicSymbols, err := ef.DynamicSymbols()
	if err != nil {
		t.Fatalf("read ELF dynamic symbols: %v", err)
	}
	for _, sym := range dynamicSymbols {
		if sym.Name == funcInfoSymbolIndexSymbol || sym.Name == funcInfoSymbolIndexCountSymbol {
			t.Fatalf("internal funcinfo symbol %s must not be dynamically exported", sym.Name)
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
	emitFuncInfoTable(ctx, src, nil, nil)
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

func TestExternalFuncInfoTableKeepsPayloadOutOfIR(t *testing.T) {
	for _, target := range []struct {
		name, goos, goarch string
		lto                lto.Mode
		identitySect       string
		entryBoundary      string
	}{
		{name: "linux", goos: "linux", goarch: "amd64", identitySect: "llgo_pclntab_id", entryBoundary: "__start_llgo_funcinfo_entry"},
		{name: "darwin/no-lto", goos: "darwin", goarch: "arm64", identitySect: "__llgo_pid", entryBoundary: "section$start$__DATA$__llgo_fie"},
		{name: "darwin/full-lto", goos: "darwin", goarch: "arm64", lto: lto.Full, identitySect: "__llgo_pid", entryBoundary: "section$start$__DATA$__llgo_fie"},
	} {
		t.Run(target.name, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: target.goos, GOARCH: target.goarch})
			prog.EnableFuncInfoMetadata(true)
			prog.EnableFuncInfoSites(true)
			src := prog.NewPackage("example.com/p", "example.com/p")
			src.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "external.go", 17, 3)
			src.EmitPCLineInfo(0x1234, "example.com/p.live", "external.go", 19, 5)
			fn := src.NewFunc("example.com/p.live", llssa.NoArgsNoRet, llssa.InGo)
			fn.MakeBody(1).Return()
			ctx := &context{
				prog: prog,
				buildConf: &Config{
					BuildMode: BuildModeExe,
					Goos:      target.goos,
					Goarch:    target.goarch,
					LTO:       target.lto,
					PCLNMode:  PCLNExternal,
				},
			}
			records := collectFuncInfo([]Package{{LPkg: src}})
			pcLines := collectPCLineInfo([]Package{{LPkg: src}})
			emitFuncInfoTable(ctx, src, records, pcLines)
			if ctx.pclnExternal == nil || len(ctx.pclnExternal.Table.Records) != 1 || len(ctx.pclnExternal.Table.PCLines) != 1 || len(ctx.pclnExternal.SymbolIndex) != 1 {
				t.Fatalf("external payload = %+v", ctx.pclnExternal)
			}
			ir := src.String()
			for _, want := range []string{
				"@__llgo_funcinfo_table = global ptr null",
				"@__llgo_funcinfo_count = global i64 0",
				"@__llgo_pclntab_identity = global [32 x i8] zeroinitializer",
				"@llvm.used = appending global [1 x ptr] [ptr @__llgo_pclntab_identity], section \"llvm.metadata\"",
				target.identitySect,
				target.entryBoundary,
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("external table IR missing %q:\n%s", want, ir)
				}
			}
			for _, unwanted := range []string{
				`"__llgo_funcinfo_table$data"`,
				`"__llgo_pcline_table$data"`,
				`"__llgo_funcinfo_strings$data"`,
				`external.go\00`,
			} {
				if strings.Contains(ir, unwanted) {
					t.Fatalf("external table IR contains payload %q:\n%s", unwanted, ir)
				}
			}
		})
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
	emitFuncInfoTable(ctx, src, nil, nil)
	if ir := src.String(); !strings.Contains(ir, "@__llgo_fp_chain = global i8 0") {
		t.Fatalf("missing fp_chain=0 in:\n%s", ir)
	}
}
