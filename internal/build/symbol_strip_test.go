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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestNativeSymbolStripScope(t *testing.T) {
	for _, tt := range []struct {
		name, goos string
		format     crosscompile.ObjectFormat
		mode       Mode
		buildmode  BuildMode
		target     string
		want       bool
	}{
		{"ELF", "linux", crosscompile.ObjectFormatELF, ModeBuild, BuildModeExe, "", true},
		{"Mach-O", "darwin", crosscompile.ObjectFormatMachO, ModeBuild, BuildModeExe, "", true},
		{"shared", "linux", crosscompile.ObjectFormatELF, ModeBuild, BuildModeCShared, "", true},
		{"archive", "linux", crosscompile.ObjectFormatELF, ModeBuild, BuildModeCArchive, "", false},
		{"generation", "linux", crosscompile.ObjectFormatELF, ModeGen, BuildModeExe, "", false},
		{"fixed", "linux", crosscompile.ObjectFormatELF, ModeBuild, BuildModeExe, "esp32c3", false},
		{"COFF", "windows", crosscompile.ObjectFormatCOFF, ModeBuild, BuildModeExe, "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Goos: tt.goos, Mode: tt.mode, BuildMode: tt.buildmode, Target: tt.target, LinkOptions: LinkOptions{OmitSymbolTable: true}}
			export := &crosscompile.Export{Toolchain: crosscompile.NativeToolchain{ObjectFormat: tt.format}}
			if got := shouldStripNativeSymbols(c, export); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			c.LinkOptions.OmitSymbolTable = false
			if shouldStripNativeSymbols(c, export) {
				t.Fatal("strip without -s")
			}
		})
	}
}

func TestNativeSymbolStripStaging(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native symbol-strip file permissions")
	}
	for _, fail := range []string{"strip", "codesign", ""} {
		t.Run("fail-"+fail, func(t *testing.T) {
			dir := t.TempDir()
			out := &OutFmtDetails{Out: filepath.Join(dir, "app"), DSYM: filepath.Join(dir, "app.dSYM")}
			if err := os.WriteFile(out.Out, []byte("linked"), 0o751); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(out.DSYM, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(out.DSYM, "old"), []byte("previous"), 0o644); err != nil {
				t.Fatal(err)
			}
			ctx := &context{buildConf: &Config{Goos: "darwin", LinkOptions: LinkOptions{DWARF: DWARFPreserve}}}
			run := func(name string, args ...string) error {
				if name == "dsymutil" {
					if err := os.MkdirAll(args[2], 0o755); err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(args[2], "new"), []byte("debug"), 0o644)
				}
				if name == "strip" {
					if err := os.WriteFile(args[len(args)-1], []byte("stripped"), 0o751); err != nil {
						return err
					}
				}
				if name == fail {
					return errors.New("tool failed")
				}
				return nil
			}
			err := stripNativeSymbolsWith(ctx, out, nil, false, run)
			if (err != nil) != (fail != "") {
				t.Fatalf("error=%v, failing tool=%s", err, fail)
			}
			contents, err := os.ReadFile(out.Out)
			if err != nil {
				t.Fatal(err)
			}
			if info, err := os.Stat(out.Out); err != nil || info.Mode().Perm() != 0o751 {
				t.Fatalf("executable mode was not preserved: %v, %v", info, err)
			}
			if fail != "" {
				if string(contents) != "linked" {
					t.Fatal("failed tool replaced executable")
				}
				if data, err := os.ReadFile(filepath.Join(out.DSYM, "old")); err != nil || string(data) != "previous" {
					t.Fatal("failed tool replaced dSYM")
				}
			} else {
				if string(contents) != "stripped" {
					t.Fatalf("output=%q", contents)
				}
				if data, err := os.ReadFile(filepath.Join(out.DSYM, "new")); err != nil || string(data) != "debug" {
					t.Fatal("missing replacement dSYM")
				}
			}
			if leftovers, _ := filepath.Glob(filepath.Join(dir, ".app.symbols-*")); len(leftovers) != 0 {
				t.Fatalf("temporary outputs: %v", leftovers)
			}
		})
	}
}

func TestNativeSymbolStripIntegration(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native ELF/Mach-O symbols")
	}
	setPCLNIntegrationEnv(t)
	for _, mode := range []PCLNMode{PCLNEmbedded, PCLNNone} {
		for _, preserve := range []bool{false, true} {
			name := mode.String()
			if preserve {
				name += "-dwarf"
			}
			t.Run(name, func(t *testing.T) {
				bin := filepath.Join(t.TempDir(), "strip-symbols")
				opts := LinkOptions{OmitSymbolTable: true}
				if preserve {
					opts.DWARF = DWARFPreserve
				}
				conf := &Config{Mode: ModeBuild, OutFile: bin, OptLevel: optlevel.Os, PCLNMode: mode, PCLNModeSet: true, LinkOptions: opts}
				if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
					t.Fatal(err)
				}
				out := runBinary(t, bin)
				if mode != PCLNNone && !strings.Contains(out, "main.caller main.go true") {
					t.Fatalf("runtime symbols: %s", out)
				}
				if pclnIntegrationHasSymbolTable(t, bin) {
					t.Fatal("-s retained ordinary Go symbols")
				}
				if pclnIntegrationHasDebugInfo(t, bin) != preserve {
					t.Fatal("DWARF retention does not match -w")
				}
				verifyPCLNIntegrationSignature(t, bin)
				if preserve {
					verifyStrippedDebuggerSymbols(t, bin)
					var data *dwarf.Data
					if runtime.GOOS == "linux" {
						f, err := elf.Open(bin)
						if err != nil {
							t.Fatal(err)
						}
						defer f.Close()
						data, err = f.DWARF()
						if err != nil {
							t.Fatal(err)
						}
					} else {
						f, err := macho.Open(filepath.Join(bin+".dSYM", "Contents", "Resources", "DWARF", filepath.Base(bin)))
						if err != nil {
							t.Fatal(err)
						}
						defer f.Close()
						data, err = f.DWARF()
						if err != nil {
							t.Fatal(err)
						}
					}
					if findDWARFNode(readDWARFTree(t, data), dwarf.TagSubprogram, "main.caller") == nil {
						t.Fatal("stripped output lost DWARF function")
					}
				}
			})
		}
	}
}

func TestELFSymbolStripStaging(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("POSIX executable permissions")
	}
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			out := &OutFmtDetails{Out: filepath.Join(t.TempDir(), "app")}
			if err := os.WriteFile(out.Out, []byte("linked"), 0o751); err != nil {
				t.Fatal(err)
			}
			ctx := &context{buildConf: &Config{Goos: "linux", LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, crossCompile: configurableDebugInfo()}
			run := func(_ string, args ...string) error {
				input, output := args[len(args)-2], args[len(args)-1]
				if input != out.Out || input == output {
					t.Fatalf("objcopy must read the linked file and write a separate staged result: %v", args)
				}
				for _, option := range []string{"--keep-section=.debug_*", "--keep-symbol=" + debugabi.NativeRecordSymbol, "--keep-symbol=llgo_debugger_threads_v1"} {
					if !strings.Contains(strings.Join(args, " "), option) {
						t.Fatalf("retained DWARF lost a debugger option %s: %v", option, args)
					}
				}
				if _, err := os.Stat(output); !os.IsNotExist(err) {
					t.Fatalf("ELF staging made a redundant executable copy: %v", err)
				}
				if err := os.WriteFile(output, []byte("stripped"), 0o600); err != nil {
					return err
				}
				if fail {
					return errors.New("objcopy failed after writing partial output")
				}
				return nil
			}
			err := stripNativeSymbolsWith(ctx, out, nil, false, run)
			if (err != nil) != fail {
				t.Fatalf("strip error: %v", err)
			}
			want := "stripped"
			if fail {
				want = "linked"
			}
			if data, err := os.ReadFile(out.Out); err != nil || string(data) != want {
				t.Fatalf("published executable: %q, %v", data, err)
			}
			if info, err := os.Stat(out.Out); err != nil || info.Mode().Perm() != 0o751 {
				t.Fatalf("executable mode: %v, %v", info, err)
			}
			if dirs, _ := filepath.Glob(filepath.Join(filepath.Dir(out.Out), ".app.symbols-*")); len(dirs) != 0 {
				t.Fatalf("staging survived: %v", dirs)
			}
		})
	}
}

func TestDSYMArtifactSize(t *testing.T) {
	dir := t.TempDir()
	out := &OutFmtDetails{Out: filepath.Join(dir, "app"), DSYM: filepath.Join(dir, "app.dSYM")}
	if err := os.WriteFile(out.Out, []byte("exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(out.DSYM, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"Contents/info": "meta", "Contents/debug": "dwarf"} {
		if err := os.WriteFile(filepath.Join(out.DSYM, name), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	artifacts, err := CollectArtifacts(&Config{DebugArtifactMode: DebugArtifactEmbedded}, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 2 || artifacts[0].Role != ArtifactRoleDeployment || artifacts[1].Role != ArtifactRoleDebug || artifacts[1].Format != "dsym" || artifacts[1].Size != 9 {
		t.Fatalf("artifacts: %+v", artifacts)
	}
}

func TestStrippedSharedLibraryExports(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native ELF/Mach-O shared library")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte("package main\n\n//export Answer\nfunc Answer() int32 { return 42 }\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ext := ".so"
	if runtime.GOOS == "darwin" {
		ext = ".dylib"
	}
	lib := filepath.Join(dir, "libanswer"+ext)
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeCShared, OutFile: lib, LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}}
	if _, err := Do([]string{"file=" + file}, conf); err != nil {
		t.Fatal(err)
	}
	loader := filepath.Join(dir, "loader.c")
	source := `#include <dlfcn.h>
#include <stdio.h>
int main(int argc, char **argv) {
  void *lib = dlopen(argv[1], RTLD_NOW);
  if (!lib) { puts(dlerror()); return 1; }
  int (*answer)(void) = (int (*)(void))dlsym(lib, "Answer");
  if (!answer || answer() != 42) return 2;
  puts("SHARED_OK");
  return 0;
}

`
	if err := os.WriteFile(loader, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "loader")
	args := []string{loader, "-o", app}
	if runtime.GOOS == "linux" {
		args = append(args, "-ldl")
	}
	if out, err := exec.Command("cc", args...).CombinedOutput(); err != nil {
		t.Fatalf("C loader: %v\n%s", err, out)
	}
	if out, err := exec.Command(app, lib).CombinedOutput(); err != nil || string(out) != "SHARED_OK\n" {
		t.Fatalf("load stripped C export: %v\n%s", err, out)
	}
	if !pclnIntegrationHasDebugInfo(t, lib) {
		t.Fatal("stripped library lost retained DWARF")
	}
}

func TestNativeSymbolStripLTO(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native ELF/Mach-O full LTO")
	}
	setPCLNIntegrationEnv(t)
	bin := filepath.Join(t.TempDir(), "strip-lto")
	conf := &Config{Mode: ModeBuild, OutFile: bin, OptLevel: optlevel.Oz, LTO: lto.Full, LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}}
	if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
		t.Fatal(err)
	}
	if out := runBinary(t, bin); !strings.Contains(out, "main.caller main.go true") {
		t.Fatalf("runtime line metadata: %s", out)
	}
	if pclnIntegrationHasSymbolTable(t, bin) {
		t.Fatal("full LTO -s retained ordinary function symbols")
	}
	if !pclnIntegrationHasDebugInfo(t, bin) {
		t.Fatal("full LTO -s -w=false lost debugger DWARF")
	}
	verifyStrippedDebuggerSymbols(t, bin)
	verifyPCLNIntegrationSignature(t, bin)
}

func TestDarwinStrippedPanicTraceback(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin DWARF runtime dlsym lookup")
	}
	setPCLNIntegrationEnv(t)
	fixture := writePCLNIntegrationSource(t, darwinDWARFPanicProbe)
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "stripped-panic")
			conf := &Config{Mode: ModeBuild, OutFile: bin, OptLevel: optlevel.Os, LTO: mode,
				LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve}}
			if _, err := Do([]string{fixture}, conf); err != nil {
				t.Fatal(err)
			}
			if out := runBinary(t, bin); out != "PCLN_DWARF_OK\n" {
				t.Fatalf("stripping lost the recovered panic stack: %s", out)
			}
		})
	}
}

func verifyStrippedDebuggerSymbols(t *testing.T, path string) {
	t.Helper()
	found := make(map[string]bool)
	if runtime.GOOS == "linux" {
		f, err := elf.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		symbols, err := f.Symbols()
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range symbols {
			if symbol.Section != elf.SHN_UNDEF {
				found[symbol.Name] = true
			}
		}
	} else {
		f, err := macho.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if f.Symtab != nil {
			for _, symbol := range f.Symtab.Syms {
				if symbol.Sect != 0 {
					found[strings.TrimPrefix(symbol.Name, "_")] = true
				}
			}
		}
	}
	for _, name := range []string{debugabi.NativeRecordSymbol, debugabi.LegacyMarkerSymbol, "llgo_debugger_threads_v1"} {
		if !found[name] {
			t.Errorf("-s -w=false lost debugger discovery symbol %s", name)
		}
	}
}
