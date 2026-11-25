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
	"bytes"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/crosscompile"
	"github.com/xgo-dev/llgo/cmd/compiler/firmware"
	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/xtool/env/llvm"
)

func TestDebugInfoLinkerArgs(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		target crosscompile.Export
		want   []string
	}{
		{name: "default"},
		{name: "safe default", conf: Config{BuildMode: BuildModeExe, OmitDWARFByDefault: true}, target: configurableDebugInfo(), want: []string{"-Wl,-S"}},
		{name: "safe default c-shared", conf: Config{BuildMode: BuildModeCShared, OmitDWARFByDefault: true}, target: configurableDebugInfo(), want: []string{"-Wl,-S"}},
		{name: "safe default c-archive", conf: Config{BuildMode: BuildModeCArchive, OmitDWARFByDefault: true}, target: configurableDebugInfo()},
		{name: "explicit c-shared w", conf: Config{BuildMode: BuildModeCShared, LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: configurableDebugInfo(), want: []string{"-Wl,-S"}},
		{name: "explicit c-archive w", conf: Config{BuildMode: BuildModeCArchive, LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: configurableDebugInfo()},
		{name: "w", conf: Config{LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: configurableDebugInfo(), want: []string{"-Wl,-S"}},
		{name: "s implies w", conf: Config{LinkOptions: LinkOptions{OmitSymbolTable: true}}, target: configurableDebugInfo(), want: []string{"-Wl,-S"}},
		{name: "explicit w false", conf: Config{LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}}},
		{name: "windows safe default", conf: Config{BuildMode: BuildModeExe, OmitDWARFByDefault: true}, target: windowsDebugInfo(), want: []string{"-Wl,/debug:none"}},
		{name: "windows explicit w", conf: Config{BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: windowsDebugInfo(), want: []string{"-Wl,/debug:none"}},
		{name: "windows explicit w false", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: windowsDebugInfo(), want: []string{"-Wl,/debug:dwarf"}},
		{name: "windows explicit PDB", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFPreserve, ExternalLinkerFlags: "-Wl,/debug:full,/pdb:app.pdb"}}, target: windowsDebugInfo()},
		{name: "windows explicit PDB via Xlinker", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFPreserve, ExternalLinkerFlags: "-Xlinker /debug:full"}}, target: windowsDebugInfo()},
		{name: "windows w overrides external PDB", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFOmit, ExternalLinkerFlags: "/debug:full"}}, target: windowsDebugInfo(), want: []string{"-Wl,/debug:none"}},
		{name: "windows c-archive", conf: Config{BuildMode: BuildModeCArchive, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: windowsDebugInfo()},
		{name: "fixed target w", conf: Config{Target: "rp2040", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: targetDebugInfo(), want: []string{"-S"}},
		{name: "fixed target w false", conf: Config{Target: "rp2040", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: targetDebugInfo()},
		{name: "target without DWARF support", conf: Config{Target: "rp2040", LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: unavailableDebugInfo()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := debugInfoLinkerArgs(&tt.conf, &tt.target); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("debugInfoLinkerArgs(%+v) = %v, want %v", tt.conf.LinkOptions, got, tt.want)
			}
		})
	}
}

func TestDebugInfoCompilerArgs(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		target crosscompile.Export
		want   []string
	}{
		{name: "linked safe default", conf: Config{Mode: ModeBuild, OmitDWARFByDefault: true}},
		{name: "linked explicit omit", conf: Config{Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFOmit}}},
		{name: "linked explicit preserve", conf: Config{Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, want: []string{"-gdwarf-4"}},
		{name: "Windows PDB", conf: Config{Goos: "windows", Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFPreserve, ExternalLinkerFlags: "-Xlinker /debug:full"}}, target: windowsDebugInfo(), want: []string{"-gdwarf-4", "-gcodeview"}},
		{name: "Windows DWARF", conf: Config{Goos: "windows", Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFPreserve, ExternalLinkerFlags: "-Xlinker /debug:dwarf"}}, target: windowsDebugInfo(), want: []string{"-gdwarf-4"}},
		{name: "MinGW PDB flag", conf: Config{Goos: "windows", Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFPreserve, ExternalLinkerFlags: "-Xlinker /debug:full"}}, target: windowsGNUToolchainDebugInfo(), want: []string{"-gdwarf-4"}},
		{name: "generation default", conf: Config{Mode: ModeGen}},
		{name: "generation explicit preserve", conf: Config{Mode: ModeGen, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, want: []string{"-gdwarf-4"}},
		{name: "target always omits", conf: Config{Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: unavailableDebugInfo()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := debugInfoCompilerArgs(&tt.conf, &tt.target); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("debugInfoCompilerArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasCOFFDebugFlag(t *testing.T) {
	for _, test := range []struct {
		value string
		want  bool
	}{
		{value: "/debug", want: true},
		{value: "/DEBUG:FULL", want: true},
		{value: "-Wl,/debug:ghash,/pdb:app.pdb", want: true},
		{value: "-Xlinker /debug:dwarf", want: true},
		{value: "/pdb:app.pdb"},
		{value: "-Wl,/incremental:no"},
		{value: "'/debug"},
	} {
		if got := hasCOFFDebugFlag(test.value); got != test.want {
			t.Errorf("hasCOFFDebugFlag(%q) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestHasCOFFPDBFlag(t *testing.T) {
	for _, test := range []struct {
		value string
		want  bool
	}{
		{value: "/debug", want: true},
		{value: "/DEBUG:FULL", want: true},
		{value: "-Wl,/debug:ghash,/pdb:app.pdb", want: true},
		{value: "/debug:none /debug:full", want: true},
		{value: "/debug:full /debug:none"},
		{value: "-Xlinker /debug:full -Xlinker /debug:dwarf"},
		{value: "-Xlinker /debug:dwarf"},
		{value: "/debug:none"},
		{value: "/pdb:app.pdb"},
	} {
		if got := hasCOFFPDBFlag(test.value); got != test.want {
			t.Errorf("hasCOFFPDBFlag(%q) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestShouldEmitCodeView(t *testing.T) {
	pdb := LinkOptions{DWARF: DWARFPreserve, ExternalLinkerFlags: "-Xlinker /debug:full"}
	for _, test := range []struct {
		name   string
		conf   Config
		target crosscompile.Export
		want   bool
	}{
		{name: "MSVC PDB", conf: Config{Goos: "windows", Mode: ModeBuild, LinkOptions: pdb}, target: windowsDebugInfo(), want: true},
		{name: "default Windows DWARF", conf: Config{Goos: "windows", Mode: ModeBuild}, target: windowsDebugInfo()},
		{name: "omitted debug", conf: Config{Goos: "windows", Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFOmit, ExternalLinkerFlags: "/debug:full"}}, target: windowsDebugInfo()},
		{name: "MinGW", conf: Config{Goos: "windows", Mode: ModeBuild, LinkOptions: pdb}, target: windowsGNUToolchainDebugInfo()},
		{name: "non-Windows", conf: Config{Goos: "linux", Mode: ModeBuild, LinkOptions: pdb}, target: windowsDebugInfo()},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldEmitCodeView(&test.conf, &test.target); got != test.want {
				t.Fatalf("shouldEmitCodeView() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEffectiveOmitDWARF(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		target crosscompile.Export
		want   bool
	}{
		{name: "default"},
		{name: "safe default", conf: Config{BuildMode: BuildModeExe, OmitDWARFByDefault: true}, want: true},
		{name: "safe default c-shared", conf: Config{BuildMode: BuildModeCShared, OmitDWARFByDefault: true}, want: true},
		{name: "safe default c-archive", conf: Config{BuildMode: BuildModeCArchive, OmitDWARFByDefault: true}, want: true},
		{name: "requested", conf: Config{LinkOptions: LinkOptions{DWARF: DWARFOmit}}, want: true},
		{name: "target baseline", target: unavailableDebugInfo(), want: true},
		{name: "explicit preserve", conf: Config{LinkOptions: LinkOptions{DWARF: DWARFPreserve}}},
		{name: "explicit preserve overrides safe default", conf: Config{OmitDWARFByDefault: true, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveOmitDWARF(&tt.conf, &tt.target); got != tt.want {
				t.Fatalf("effectiveOmitDWARF() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldEmitDebugInfo(t *testing.T) {
	tests := []struct {
		name   string
		conf   Config
		target crosscompile.Export
		want   bool
	}{
		{name: "linked default", conf: Config{Mode: ModeBuild}, want: true},
		{name: "linked safe default", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, OmitDWARFByDefault: true}},
		{name: "c-shared safe default", conf: Config{Mode: ModeBuild, BuildMode: BuildModeCShared, OmitDWARFByDefault: true}},
		{name: "c-archive safe default", conf: Config{Mode: ModeBuild, BuildMode: BuildModeCArchive, OmitDWARFByDefault: true}},
		{name: "linked w", conf: Config{Mode: ModeBuild, LinkOptions: LinkOptions{DWARF: DWARFOmit}}},
		{name: "linked s", conf: Config{Mode: ModeBuild, LinkOptions: LinkOptions{OmitSymbolTable: true}}},
		{name: "linked s w false", conf: Config{Mode: ModeBuild, LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}}, want: true},
		{name: "generation default", conf: Config{Mode: ModeGen}},
		{name: "generation requested", conf: Config{Mode: ModeGen, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, want: true},
		{name: "target without DWARF support", conf: Config{Mode: ModeBuild}, target: unavailableDebugInfo()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldEmitDebugInfo(&tt.conf, &tt.target); got != tt.want {
				t.Fatalf("shouldEmitDebugInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateLinkOptions(t *testing.T) {
	w := LinkOptions{DWARF: DWARFOmit}
	wFalse := LinkOptions{DWARF: DWARFPreserve}
	tests := []struct {
		name    string
		conf    Config
		target  crosscompile.Export
		wantErr bool
	}{
		{name: "linux executable", conf: Config{Goos: "linux", BuildMode: BuildModeExe, LinkOptions: w}, target: configurableDebugInfo()},
		{name: "linux executable safe default", conf: Config{Goos: "linux", BuildMode: BuildModeExe, OmitDWARFByDefault: true}, target: configurableDebugInfo()},
		{name: "darwin executable", conf: Config{Goos: "darwin", BuildMode: BuildModeExe, LinkOptions: w}, target: configurableDebugInfo()},
		{name: "c-shared safe default", conf: Config{Goos: "linux", BuildMode: BuildModeCShared, OmitDWARFByDefault: true}, target: configurableDebugInfo()},
		{name: "c-archive safe default", conf: Config{Goos: "linux", BuildMode: BuildModeCArchive, OmitDWARFByDefault: true}, target: configurableDebugInfo()},
		{name: "unsupported native OS", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: w}, wantErr: true},
		{name: "windows safe default", conf: Config{Goos: "windows", BuildMode: BuildModeExe, OmitDWARFByDefault: true}, target: windowsDebugInfo()},
		{name: "windows omit", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: w}, target: windowsDebugInfo()},
		{name: "windows preserve", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: wFalse}, target: windowsDebugInfo()},
		{name: "c-shared omit", conf: Config{Goos: "linux", BuildMode: BuildModeCShared, LinkOptions: w}, target: configurableDebugInfo()},
		{name: "c-archive omit", conf: Config{Goos: "linux", BuildMode: BuildModeCArchive, LinkOptions: w}, target: configurableDebugInfo()},
		{name: "c-shared preserve", conf: Config{Goos: "linux", BuildMode: BuildModeCShared, LinkOptions: wFalse}, target: configurableDebugInfo()},
		{name: "c-archive preserve", conf: Config{Goos: "linux", BuildMode: BuildModeCArchive, LinkOptions: wFalse}, target: configurableDebugInfo()},
		{name: "target without DWARF support omit", conf: Config{Target: "custom", Goos: "linux", BuildMode: BuildModeExe, LinkOptions: w}, target: unavailableDebugInfo()},
		{name: "target without DWARF support preserve", conf: Config{Target: "custom", Goos: "linux", BuildMode: BuildModeExe, LinkOptions: wFalse}, target: unavailableDebugInfo(), wantErr: true},
		{name: "fixed target omit", conf: Config{Target: "rp2040", Goos: "linux", BuildMode: BuildModeExe, LinkOptions: w}, target: targetDebugInfo(), wantErr: false},
		{name: "fixed target preserve", conf: Config{Target: "rp2040", Goos: "linux", BuildMode: BuildModeExe, LinkOptions: wFalse}, target: targetDebugInfo()},
		{name: "configurable WASI omit", conf: Config{Target: "wasi", Goos: "wasip1", BuildMode: BuildModeExe, LinkOptions: w}, target: configurableDebugInfo()},
		{name: "configurable WASI preserve", conf: Config{Target: "wasi", Goos: "wasip1", BuildMode: BuildModeExe, LinkOptions: wFalse}, target: configurableDebugInfo()},
		{name: "no omission", conf: Config{Goos: "windows", BuildMode: BuildModeExe, LinkOptions: wFalse}},
		{name: "invalid DWARF mode", conf: Config{Goos: "linux", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFMode(255)}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLinkOptions(&tt.conf, &tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateLinkOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDwarfLinkerArgsSuppressNativeInputDWARF(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ELF debug-section integration test")
	}

	llvmEnv := llvm.New("")
	dir := t.TempDir()
	source := filepath.Join(dir, "main.c")
	object := filepath.Join(dir, "main.o")
	bin := filepath.Join(dir, "app")
	if err := os.WriteFile(source, []byte("int main(void) { return 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	clang := filepath.Join(llvmEnv.BinDir(), "clang++")
	if out, err := exec.Command(clang, "-g", "-c", "-o", object, source).CombinedOutput(); err != nil {
		t.Fatalf("compile native DWARF fixture: %v\n%s", err, out)
	}
	if !elfHasDebugInfo(t, object) {
		t.Fatal("native input has no debug information")
	}

	conf := &Config{Goos: "linux", BuildMode: BuildModeExe, LinkOptions: LinkOptions{DWARF: DWARFOmit}}
	target := configurableDebugInfo()
	args := append(debugInfoLinkerArgs(conf, &target), "-o", bin, object)
	if out, err := exec.Command(clang, args...).CombinedOutput(); err != nil {
		t.Fatalf("link native fixture with DWARF omission: %v\n%s", err, out)
	}
	if elfHasDebugInfo(t, bin) {
		t.Fatal("debug information from native input reached the linked artifact")
	}
	if out, err := exec.Command(bin).CombinedOutput(); err != nil {
		t.Fatalf("run linked fixture: %v\n%s", err, out)
	}
}

func TestTargetDWARFDoesNotChangeLoadableELF(t *testing.T) {
	// Use the embedded target's toolchain and policy. In particular, MSYS2's
	// host ld.lld defaults to MinGW rather than the ELF driver used by LLGo's
	// bundled ESP Clang; looking up the host linker would test another backend.
	target, err := crosscompile.UseTarget("cortex-m-qemu", optlevel.O0, lto.Off)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "Cortex-M fixture")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "main.c")
	object := filepath.Join(dir, "main.o")
	debugELF := filepath.Join(dir, "debug.elf")
	strippedELF := filepath.Join(dir, "stripped.elf")
	debugBin := filepath.Join(dir, "debug.bin")
	strippedBin := filepath.Join(dir, "stripped.bin")
	if err := os.WriteFile(source, []byte("volatile int value = 41; void Reset_Handler(void) { value++; for (;;) {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compileArgs := append([]string{}, target.CCFLAGS...)
	compileArgs = append(compileArgs, target.CFLAGS...)
	compileArgs = append(compileArgs, "-x", "c", "-g", "-ffreestanding", "-c", "-o", object, source)
	if out, err := exec.Command(target.CC, compileArgs...).CombinedOutput(); err != nil {
		t.Fatalf("compile Cortex-M DWARF fixture: %v\n%s", err, out)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	link := func(path string, opts LinkOptions) {
		t.Helper()
		conf := &Config{Target: "cortex-m-qemu", BuildMode: BuildModeExe, LinkOptions: opts}
		args := append([]string{}, target.LDFLAGS...)
		args = append(args, debugInfoLinkerArgs(conf, &target)...)
		args = append(args, "-o", path, object)
		cmd := exec.Command(target.Linker, args...)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("link Cortex-M fixture: %v\n%s", err, out)
		}
	}
	link(debugELF, LinkOptions{DWARF: DWARFPreserve})
	link(strippedELF, LinkOptions{DWARF: DWARFOmit})
	if !elfHasDebugInfo(t, debugELF) {
		t.Fatal("preserved Cortex-M ELF has no debug information")
	}
	if elfHasDebugInfo(t, strippedELF) {
		t.Fatal("omitted Cortex-M ELF still has debug information")
	}
	if err := firmware.ConvertFormats("", "", map[string]string{"out": debugELF, "bin": debugBin}); err != nil {
		t.Fatal(err)
	}
	if err := firmware.ConvertFormats("", "", map[string]string{"out": strippedELF, "bin": strippedBin}); err != nil {
		t.Fatal(err)
	}
	debugImage, err := os.ReadFile(debugBin)
	if err != nil {
		t.Fatal(err)
	}
	strippedImage, err := os.ReadFile(strippedBin)
	if err != nil {
		t.Fatal(err)
	}
	if len(debugImage) == 0 || !bytes.Equal(debugImage, strippedImage) {
		t.Fatal("DWARF changed Cortex-M flashed bytes")
	}
}

func windowsDebugInfo() crosscompile.Export {
	return crosscompile.Export{Toolchain: crosscompile.NativeToolchain{ABI: crosscompile.PlatformABIMsvc}, DebugInfo: crosscompile.DebugInfoPolicy{
		OmitLinkFlags:     []string{"-Wl,/debug:none"},
		PreserveLinkFlags: []string{"-Wl,/debug:dwarf"},
	}}
}

func windowsGNUToolchainDebugInfo() crosscompile.Export {
	target := windowsDebugInfo()
	target.Toolchain.ABI = crosscompile.PlatformABIGNU
	return target
}

func configurableDebugInfo() crosscompile.Export {
	return crosscompile.Export{DebugInfo: crosscompile.DebugInfoPolicy{
		OmitLinkFlags: []string{"-Wl,-S"},
	}}
}

func targetDebugInfo() crosscompile.Export {
	return crosscompile.Export{DebugInfo: crosscompile.DebugInfoPolicy{OmitLinkFlags: []string{"-S"}}}
}

func unavailableDebugInfo() crosscompile.Export {
	return crosscompile.Export{DebugInfo: crosscompile.DebugInfoPolicy{Capability: crosscompile.DebugInfoUnavailable}}
}

func elfHasDebugInfo(t *testing.T, path string) bool {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, section := range f.Sections {
		if strings.HasPrefix(section.Name, ".debug_") || strings.HasPrefix(section.Name, ".zdebug_") {
			return true
		}
	}
	return false
}

func TestNativeDWARFDefaults(t *testing.T) {
	for _, tt := range []struct {
		name, goos, goarch, target string
		mode                       BuildMode
		wantOmit                   bool
	}{
		{name: "darwin executable", goos: "darwin", goarch: "arm64"},
		{name: "linux executable", goos: "linux", goarch: "amd64"},
		{name: "windows executable", goos: "windows", goarch: "amd64"},
		{name: "darwin archive", goos: "darwin", goarch: "arm64", mode: BuildModeCArchive},
		{name: "linux shared", goos: "linux", goarch: "amd64", mode: BuildModeCShared},
		{name: "darwin shared", goos: "darwin", goarch: "arm64", mode: BuildModeCShared, wantOmit: true},
		{name: "browser", goos: "js", goarch: "wasm", wantOmit: true},
		{name: "wasi", goos: "wasip1", goarch: "wasm", wantOmit: true},
		{name: "embedded", target: "cortex-m-qemu", wantOmit: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.Goos, conf.Goarch, conf.Target, conf.BuildMode = tt.goos, tt.goarch, tt.target, tt.mode
			if got := omitDWARFRequested(conf); got != tt.wantOmit {
				t.Fatalf("omit DWARF = %v, want %v", got, tt.wantOmit)
			}
			conf.LinkOptions.DWARF = DWARFPreserve
			if omitDWARFRequested(conf) {
				t.Fatal("explicit preserve must override the default")
			}
			conf.LinkOptions.DWARF = DWARFOmit
			if !omitDWARFRequested(conf) {
				t.Fatal("explicit omit must override the default")
			}
		})
	}
}
