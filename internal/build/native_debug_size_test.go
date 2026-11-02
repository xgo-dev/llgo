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
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

const darwinDWARFPanicProbe = `package main
import (
	"os"
	"runtime"
)
//go:noinline
func panicSite() { panic("dwarf panic") }
func checkPanic() {
	defer func() {
		if recover() != "dwarf panic" { panic("bad recovery") }
		var pcs [32]uintptr
		n := runtime.Callers(1, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			if frame.Function == "main.panicSite" && frame.Line > 0 { return }
			if !more { panic("lost panic site") }
		}
	}()
	panicSite()
}
func main() { checkPanic(); os.Stdout.WriteString("PCLN_DWARF_OK\n") }
`

func TestDarwinLTOObjectLifecycle(t *testing.T) {
	conf := Config{Goos: "darwin", LTO: lto.Full, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}
	ctx := &context{buildConf: &conf, crossCompile: configurableDebugInfo()}
	app := filepath.Join(t.TempDir(), "app")
	first, err := darwinLTOObjectArgs(ctx, app, ctx.linker(), nil)
	if err != nil {
		t.Fatal(err)
	}
	object := first[len(first)-1]
	if err := os.WriteFile(object, []byte("older debug object"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := darwinLTOObjectArgs(ctx, app, ctx.linker(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if second[len(second)-1] == object {
		t.Fatal("rebuild reused the previous executable's debug object")
	}
	if data, err := os.ReadFile(object); err != nil || string(data) != "older debug object" {
		t.Fatalf("rebuild invalidated the older debug map: %q, %v", data, err)
	}
	if err := removeExecutableArtifacts(app, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(darwinLTOObjectDir(app)); !os.IsNotExist(err) {
		t.Fatalf("clean retained LTO objects: %v", err)
	}
	if _, err := darwinLTOObjectArgs(ctx, app, ctx.linker(), nil); err != nil {
		t.Fatal(err)
	}
	removeOutFmts(&OutFmtDetails{Out: app})
	if _, err := os.Stat(darwinLTOObjectDir(app)); !os.IsNotExist(err) {
		t.Fatalf("temporary-output cleanup retained LTO objects: %v", err)
	}
	if err := os.WriteFile(darwinLTOObjectDir(app), []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := darwinLTOObjectArgs(ctx, app, ctx.linker(), nil); err == nil {
		t.Fatal("accepted an occupied object-directory path")
	}
}

func TestDarwinLTOObjectPolicy(t *testing.T) {
	base := Config{Goos: "darwin", LTO: lto.Full, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}
	for _, name := range []string{"linux", "no-lto", "named-target", "omit-dwarf", "driver-option", "linker-option", "environment-option"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("LDFLAGS", "")
			t.Setenv("CCFLAGS", "")
			conf := base
			ctx := &context{buildConf: &conf, crossCompile: configurableDebugInfo()}
			var args []string
			switch name {
			case "linux":
				conf.Goos = "linux"
			case "no-lto":
				conf.LTO = lto.Off
			case "named-target":
				conf.Target = "custom"
			case "omit-dwarf":
				conf.LinkOptions.DWARF = DWARFOmit
			case "driver-option":
				args = []string{"-Xlinker", "-object_path_lto", "-Xlinker", "chosen.o"}
			case "linker-option":
				args = []string{"-Wl,-object_path_lto,chosen.o"}
			case "environment-option":
				t.Setenv("LDFLAGS", "-Wl,-object_path_lto,chosen.o")
			}
			app := filepath.Join(t.TempDir(), "app")
			got, err := darwinLTOObjectArgs(ctx, app, ctx.linker(), args)
			if err != nil || len(got) != 0 {
				t.Fatalf("unexpected automatic object path: %v, %v", got, err)
			}
			if _, err := os.Stat(darwinLTOObjectDir(app)); !os.IsNotExist(err) {
				t.Fatalf("unused object directory was created: %v", err)
			}
		})
	}
}

func TestDarwinLTOObjectPathArguments(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want bool
	}{
		{"direct option", []string{"-object_path_lto", "chosen.o"}, true},
		{"driver option", []string{"-Xlinker", "-object_path_lto", "-Xlinker", "chosen.o"}, true},
		{"comma options", []string{"-Wl,-dead_strip,-object_path_lto,chosen.o"}, true},
		{"output substring", []string{"-o", "object_path_lto-probe"}, false},
		{"output matching token", []string{"-o", "-object_path_lto"}, false},
		{"library substring", []string{"-L/tmp/object_path_lto/lib", "-lobject_path_lto"}, false},
		{"separate library path", []string{"-L", "/tmp/object_path_lto"}, false},
		{"linker path value", []string{"-Wl,-rpath,/tmp/object_path_lto"}, false},
		{"input after separator", []string{"--", "-object_path_lto"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasDarwinLTOObjectPath(tt.args); got != tt.want {
				t.Fatalf("hasDarwinLTOObjectPath(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestDarwinLTOFailedLinkPreservesOlderObjects(t *testing.T) {
	t.Setenv("LLGO_TEST_LINKER_HELPER", "fail")
	for _, occupied := range []bool{false, true} {
		t.Run(strconv.FormatBool(occupied), func(t *testing.T) {
			conf := Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "darwin", LTO: lto.Full, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}
			ctx := &context{buildConf: &conf, crossCompile: configurableDebugInfo()}
			ctx.crossCompile.CC = os.Args[0]
			app := filepath.Join(t.TempDir(), "app")
			if occupied {
				if err := os.WriteFile(darwinLTOObjectDir(app), []byte("occupied"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := linkObjFiles(ctx, app, nil, nil, false); err == nil || !strings.Contains(err.Error(), "retain Darwin LTO debug object") {
					t.Fatalf("retention failure was not reported: %v", err)
				}
				return
			}
			older, err := darwinLTOObjectArgs(ctx, app, ctx.linker(), nil)
			if err != nil {
				t.Fatal(err)
			}
			object := older[len(older)-1]
			if err := os.WriteFile(object, []byte("older"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := linkObjFiles(ctx, app, nil, nil, false); err == nil {
				t.Fatal("failing linker succeeded")
			}
			entries, err := os.ReadDir(darwinLTOObjectDir(app))
			if err != nil || len(entries) != 1 {
				t.Fatalf("failed link retained its new object directory: %v, %v", entries, err)
			}
			if data, err := os.ReadFile(object); err != nil || string(data) != "older" {
				t.Fatalf("failed link invalidated an older debug object: %q, %v", data, err)
			}
		})
	}
}

func TestDarwinLTOObjectDirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires POSIX directory permissions without root bypass")
	}
	conf := Config{Goos: "darwin", LTO: lto.Full, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}
	ctx := &context{buildConf: &conf, crossCompile: configurableDebugInfo()}
	app := filepath.Join(t.TempDir(), "app")
	dir := darwinLTOObjectDir(app)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o700)
	if _, err := darwinLTOObjectArgs(ctx, app, ctx.linker(), nil); err == nil {
		t.Fatal("created an LTO object directory without write permission")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(dir, "keep.o")
	if err := os.WriteFile(keep, []byte("debug object"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := removeArtifactDirectory(dir, false); err == nil {
		t.Fatal("clean silently ignored an inaccessible retained object")
	}
	if data, err := os.ReadFile(keep); err != nil || string(data) != "debug object" {
		t.Fatalf("failed clean changed the debug object: %q, %v", data, err)
	}
}

func TestLinuxDWARFCompressionArgs(t *testing.T) {
	target := crosscompile.Export{
		Toolchain: crosscompile.NativeToolchain{ObjectFormat: crosscompile.ObjectFormatELF},
		DebugInfo: configurableDebugInfo().DebugInfo,
	}
	for _, tt := range []struct {
		name string
		conf Config
		want bool
	}{
		{"default", Config{Goos: "linux", BuildMode: BuildModeExe}, true},
		{"shared", Config{Goos: "linux", BuildMode: BuildModeCShared, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, true},
		{"disabled", Config{Goos: "linux", LinkOptions: LinkOptions{DWARFCompression: DWARFCompressionDisabled}}, false},
		{"omit", Config{Goos: "linux", LinkOptions: LinkOptions{DWARF: DWARFOmit}}, false},
		{"archive", Config{Goos: "linux", BuildMode: BuildModeCArchive}, false},
		{"fixed-target", Config{Goos: "linux", Target: "esp32c3", LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, false},
		{"other-platform", Config{Goos: "darwin"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := strings.Join(debugInfoLinkerArgs(&tt.conf, &target), " ")
			if got := strings.Contains(args, "--compress-debug-sections=zlib"); got != tt.want {
				t.Fatalf("linker args %q: compression=%v, want %v", args, got, tt.want)
			}
		})
	}
}

func TestLinuxCompressedDWARF(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("native ELF compression integration")
	}
	for _, disabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(disabled), func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "compressed-dwarf")
			compression := DWARFCompressionDefault
			if disabled {
				compression = DWARFCompressionDisabled
			}
			conf := &Config{Mode: ModeBuild, OutFile: bin, LinkOptions: LinkOptions{DWARF: DWARFPreserve, DWARFCompression: compression}}
			if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
				t.Fatal(err)
			}
			if out := runBinary(t, bin); !strings.Contains(out, "main.caller main.go true") {
				t.Fatalf("runtime PCLN output: %s", out)
			}
			f, err := elf.Open(bin)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			info := f.Section(".debug_info")
			if info == nil || (info.Flags&elf.SHF_COMPRESSED != 0) == disabled {
				t.Fatalf(".debug_info compression does not match disabled=%v", disabled)
			}
			if f.Section(".symtab") == nil {
				t.Fatal("DWARF compression removed ordinary symbols")
			}
			data, err := f.DWARF()
			if err != nil {
				t.Fatal(err)
			}
			if findDWARFNode(readDWARFTree(t, data), dwarf.TagSubprogram, "main.caller") == nil {
				t.Fatal("compressed DWARF lost main.caller")
			}
		})
	}
}

func TestDarwinDWARFSizeSymbols(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native Mach-O export and debug-map integration")
	}
	t.Setenv("LDFLAGS", "")
	t.Setenv(llgoFuncInfo, "1")
	t.Setenv(llgoFuncInfoSites, "1")
	source := dwarfPCLNProbe
	for _, marker := range []string{"CALLER", "FRAMES", "LOG"} {
		source = strings.ReplaceAll(source, marker+"_LINE", strconv.Itoa(dwarfPCLNMarkerLine(source, marker+"_MARK")))
	}
	fixture := writePCLNIntegrationSource(t, source)
	panicFixture := writePCLNIntegrationSource(t, darwinDWARFPanicProbe)
	for _, mode := range []lto.Mode{lto.Off, lto.Full, lto.Thin} {
		t.Run(mode.String(), func(t *testing.T) {
			input := fixture
			if mode.Enabled() {
				// LTO keeps its previous visibility and inlining policy. Check
				// physical panic frames without imposing new inline-line rules.
				input = panicFixture
			}
			bin := filepath.Join(t.TempDir(), "object_path_lto-debug-size")
			conf := &Config{Mode: ModeBuild, OutFile: bin, OptLevel: optlevel.Os, LTO: mode, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}
			if _, err := Do([]string{input}, conf); err != nil {
				t.Fatal(err)
			}
			if got := runBinary(t, bin); got != "PCLN_DWARF_OK\n" {
				t.Fatalf("runtime line metadata: %s", got)
			}
			if !machoHasStabs(t, bin) {
				t.Fatal("DWARF size optimization removed the dsymutil debug map")
			}
			f, err := macho.Open(bin)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			functionExports := 0
			for _, sym := range f.Symtab.Syms {
				if sym.Sect != 0 && sym.Type&0xe0 == 0 && sym.Type&0x11 == 1 {
					if mode == lto.Off && (strings.HasPrefix(sym.Name, "__llgo_") || strings.HasPrefix(sym.Name, "___llgo_") || strings.HasPrefix(sym.Name, "main.") && f.Sections[sym.Sect-1].Name != "__text") {
						t.Fatalf("unexpected public Go data definition %s", sym.Name)
					}
					if f.Sections[sym.Sect-1].Name == "__text" {
						functionExports++
					}
				}
			}
			if functionExports == 0 {
				t.Fatal("DWARF runtime dlsym fallback lost its function exports")
			}
			data, _ := openNativeDWARF(t, bin)
			if findDWARFNode(readDWARFTree(t, data), dwarf.TagSubprogram, "main.main") == nil {
				t.Fatal("dsymutil lost main.main")
			}
			if mode == lto.Full {
				older := filepath.Join(t.TempDir(), "older-debug-size")
				if err := copyFileAtomic(bin, older); err != nil {
					t.Fatal(err)
				}
				if _, err := Do([]string{input}, conf); err != nil {
					t.Fatal(err)
				}
				data, _ := openNativeDWARF(t, older)
				if findDWARFNode(readDWARFTree(t, data), dwarf.TagSubprogram, "main.main") == nil {
					t.Fatal("rebuild invalidated the previous LTO debug map")
				}
			}
			verifyPCLNIntegrationSignature(t, bin)
		})
	}
}
