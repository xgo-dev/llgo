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
	"debug/macho"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestPlanDarwinSizeSymbols(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	base := Config{
		Goos:               "darwin",
		BuildMode:          BuildModeExe,
		OptLevel:           optlevel.Os,
		OmitDWARFByDefault: true,
		PCLNMode:           PCLNNone,
	}
	for name, ctx := range map[string]*context{
		"nil-context": nil,
		"nil-config":  {},
	} {
		t.Run(name, func(t *testing.T) {
			if plan := planDarwinSizeSymbolsFor(ctx, nil, nil, true); len(plan.linkerArgs) != 0 || plan.stripLTOLocals {
				t.Fatalf("plan = %+v, want no automatic symbol compaction", plan)
			}
		})
	}
	tests := []struct {
		name      string
		conf      Config
		native    bool
		linkArgs  []string
		wantArgs  []string
		wantStrip bool
	}{
		{name: "non-lto", conf: base, native: true, wantArgs: []string{"-Wl,-no_exported_symbols"}},
		{name: "full-lto", conf: withLTO(base, lto.Full), native: true, wantStrip: true},
		{name: "thin-lto", conf: withLTO(base, lto.Thin), native: true, wantStrip: true},
		{name: "oz", conf: withOptLevel(base, optlevel.Oz), native: true, wantArgs: []string{"-Wl,-no_exported_symbols"}},
		{name: "o2", conf: withOptLevel(base, optlevel.O2), native: true},
		{name: "cross-host", conf: base},
		{name: "named-target", conf: withTarget(base, "wasi"), native: true},
		{name: "c-shared", conf: withBuildMode(base, BuildModeCShared), native: true},
		{name: "preserve-dwarf", conf: withDWARF(base, DWARFPreserve), native: true},
		{name: "embedded-without-sites", conf: withPCLN(base, PCLNEmbedded), native: true},
		{name: "explicit-export", conf: base, native: true, linkArgs: []string{"-Wl,-exported_symbol,_entry"}},
		{name: "dynamic-lookup", conf: base, native: true, linkArgs: []string{"-Wl,-undefined,dynamic_lookup"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &context{buildConf: &tt.conf}
			plan := planDarwinSizeSymbolsFor(ctx, nil, tt.linkArgs, tt.native)
			if !reflect.DeepEqual(plan.linkerArgs, tt.wantArgs) || plan.stripLTOLocals != tt.wantStrip {
				t.Fatalf("plan = %+v, want args=%v strip=%v", plan, tt.wantArgs, tt.wantStrip)
			}
		})
	}
}

func TestHasDarwinExportControl(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{args: []string{"-Wl,-dead_strip"}},
		{args: []string{"-Wl,-export_dynamic"}, want: true},
		{args: []string{"-Wl,-exported_symbols_list,exports.txt"}, want: true},
		{args: []string{"-Wl,-no_exported_symbols"}, want: true},
		{args: []string{"-Wl,-undefined,dynamic_lookup"}, want: true},
	}
	for _, tt := range tests {
		if got := hasDarwinExportControl(tt.args); got != tt.want {
			t.Errorf("hasDarwinExportControl(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestPlanDarwinSizeSymbolsHonorsEnvironmentExportControl(t *testing.T) {
	t.Setenv("LDFLAGS", "-Wl,-exported_symbol,_entry")
	conf := Config{
		Goos:               "darwin",
		BuildMode:          BuildModeExe,
		OptLevel:           optlevel.Os,
		OmitDWARFByDefault: true,
		PCLNMode:           PCLNNone,
	}
	ctx := &context{buildConf: &conf}
	if plan := planDarwinSizeSymbolsFor(ctx, nil, nil, true); len(plan.linkerArgs) != 0 || plan.stripLTOLocals {
		t.Fatalf("plan with explicit environment export = %+v, want no automatic symbol compaction", plan)
	}
}

func TestMainPackageHasExports(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	runtimePkg := prog.NewPackage("runtime", "runtime")
	runtimePkg.SetExport("runtime.callback", "callback")
	mainPkg := prog.NewPackage("main", "main")
	pkgs := []*aPackage{
		nil,
		{Package: &packages.Package{Name: "runtime", PkgPath: "runtime"}, LPkg: runtimePkg},
		{Package: &packages.Package{Name: "main", PkgPath: "main"}, LPkg: mainPkg},
	}
	if mainPackageHasExports(pkgs) {
		t.Fatal("runtime callback was treated as a user-facing executable export")
	}
	mainPkg.SetExport("Entry", "Entry")
	if !mainPackageHasExports(pkgs) {
		t.Fatal("main package C export was not detected")
	}
	t.Setenv("LDFLAGS", "")
	conf := Config{
		Goos:               "darwin",
		BuildMode:          BuildModeExe,
		OptLevel:           optlevel.Os,
		OmitDWARFByDefault: true,
		PCLNMode:           PCLNNone,
	}
	if plan := planDarwinSizeSymbolsFor(&context{buildConf: &conf}, pkgs, nil, true); len(plan.linkerArgs) != 0 || plan.stripLTOLocals {
		t.Fatalf("plan with main package export = %+v, want no automatic symbol compaction", plan)
	}
}

func TestFinalizeDarwinSizeExecutable(t *testing.T) {
	if err := finalizeDarwinSizeExecutable(nil, "", false); err != nil {
		t.Fatalf("nil context: %v", err)
	}
	if err := finalizeDarwinSizeExecutable(&context{}, "", false); err != nil {
		t.Fatalf("disabled compaction: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	err := finalizeDarwinSizeExecutable(&context{stripDarwinLTOLocals: true}, missing, false)
	if err == nil || !strings.Contains(err.Error(), "compact Darwin LTO symbols") {
		t.Fatalf("missing executable error = %v", err)
	}
}

func TestStripAndSignDarwinLocalsStaging(t *testing.T) {
	newExecutable := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "program")
		if err := os.WriteFile(path, []byte("original"), 0o751); err != nil {
			t.Fatal(err)
		}
		return path
	}
	readExecutable := func(t *testing.T, path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	t.Run("success", func(t *testing.T) {
		path := newExecutable(t)
		var commands []string
		run := func(name string, args ...string) ([]byte, error) {
			commands = append(commands, name)
			staged := args[len(args)-1]
			if err := os.WriteFile(staged, []byte(name), 0); err != nil {
				return nil, err
			}
			return nil, nil
		}
		if err := stripAndSignDarwinLocalsWith(path, true, run); err != nil {
			t.Fatal(err)
		}
		if got := readExecutable(t, path); got != "codesign" {
			t.Fatalf("final executable = %q, want staged signed contents", got)
		}
		if !reflect.DeepEqual(commands, []string{"strip", "codesign"}) {
			t.Fatalf("commands = %v", commands)
		}
		st, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0o751 {
			t.Fatalf("final executable mode = %v, want 0751", st.Mode().Perm())
		}
	})

	tests := []struct {
		name        string
		failCommand string
	}{
		{name: "strip-failure", failCommand: "strip"},
		{name: "codesign-failure", failCommand: "codesign"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := newExecutable(t)
			run := func(name string, args ...string) ([]byte, error) {
				staged := args[len(args)-1]
				if err := os.WriteFile(staged, []byte("mutated"), 0); err != nil {
					return nil, err
				}
				if name == tt.failCommand {
					return []byte("tool output"), errors.New("tool failure")
				}
				return nil, nil
			}
			err := stripAndSignDarwinLocalsWith(path, false, run)
			if err == nil || !strings.Contains(err.Error(), tt.failCommand) {
				t.Fatalf("%s error = %v", tt.failCommand, err)
			}
			if got := readExecutable(t, path); got != "original" {
				t.Fatalf("original executable changed to %q after %s failure", got, tt.failCommand)
			}
			matches, globErr := filepath.Glob(filepath.Join(filepath.Dir(path), ".program.strip-*"))
			if globErr != nil || len(matches) != 0 {
				t.Fatalf("staged files after %s failure = %v, %v", tt.failCommand, matches, globErr)
			}
		})
	}

	t.Run("copy-failure", func(t *testing.T) {
		calls := 0
		run := func(string, ...string) ([]byte, error) {
			calls++
			return nil, nil
		}
		if err := stripAndSignDarwinLocalsWith(t.TempDir(), false, run); err == nil {
			t.Fatal("copying a directory as an executable succeeded")
		}
		if calls != 0 {
			t.Fatalf("external commands called %d times after copy failure", calls)
		}
	})
}

func TestStripAndSignDarwinLocalsFileFailures(t *testing.T) {
	newExecutable := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "program")
		if err := os.WriteFile(path, []byte("original"), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	fileFailure := errors.New("file failure")
	tests := []struct {
		name   string
		mutate func(*darwinSizeFileOps)
	}{
		{
			name: "open-source",
			mutate: func(files *darwinSizeFileOps) {
				files.open = func(string) (*os.File, error) { return nil, fileFailure }
			},
		},
		{
			name: "create-stage",
			mutate: func(files *darwinSizeFileOps) {
				files.createTemp = func(string, string) (*os.File, error) { return nil, fileFailure }
			},
		},
		{
			name: "chmod-stage",
			mutate: func(files *darwinSizeFileOps) {
				files.createTemp = func(dir, pattern string) (*os.File, error) {
					file, err := os.CreateTemp(dir, pattern)
					if err == nil {
						err = file.Close()
					}
					return file, err
				}
			},
		},
		{
			name: "open-signed-stage",
			mutate: func(files *darwinSizeFileOps) {
				files.openFile = func(string, int, os.FileMode) (*os.File, error) { return nil, fileFailure }
			},
		},
		{
			name: "rename-stage",
			mutate: func(files *darwinSizeFileOps) {
				files.rename = func(string, string) error { return fileFailure }
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := newExecutable(t)
			files := darwinSizeOSFileOps()
			tt.mutate(&files)
			run := func(string, ...string) ([]byte, error) { return nil, nil }
			if err := stripAndSignDarwinLocalsUsing(path, false, run, files); err == nil {
				t.Fatal("file operation failure was ignored")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "original" {
				t.Fatalf("original executable changed to %q", data)
			}
			matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".program.strip-*"))
			if err != nil || len(matches) != 0 {
				t.Fatalf("staged files after failure = %v, %v", matches, err)
			}
		})
	}

	t.Run("sync-signed-stage", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("pipe sync failure uses Unix file semantics")
		}
		path := newExecutable(t)
		files := darwinSizeOSFileOps()
		files.openFile = func(string, int, os.FileMode) (*os.File, error) {
			reader, writer, err := os.Pipe()
			if err != nil {
				return nil, err
			}
			t.Cleanup(func() {
				_ = reader.Close()
				_ = writer.Close()
			})
			return writer, nil
		}
		run := func(string, ...string) ([]byte, error) { return nil, nil }
		if err := stripAndSignDarwinLocalsUsing(path, false, run, files); err == nil {
			t.Fatal("syncing a signed pipe succeeded")
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "original" {
			t.Fatalf("original executable after sync failure = %q, %v", data, err)
		}
	})
}

func TestDarwinSizeSymbolsIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Mach-O symbol compaction integration test")
	}
	t.Setenv("LDFLAGS", "")
	t.Setenv(llgoFuncInfo, "1")
	tests := []struct {
		name      string
		lto       lto.Mode
		pcln      PCLNMode
		wantLocal bool
	}{
		{name: "non-lto", pcln: PCLNEmbedded, wantLocal: true},
		{name: "full-lto", lto: lto.Full, pcln: PCLNEmbedded},
		{name: "external-full-lto", lto: lto.Full, pcln: PCLNExternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binaryPath := filepath.Join(t.TempDir(), "size-symbols")
			level := optlevel.Os
			if tt.lto.Enabled() {
				// LTO compaction ordering is independent of the optimizer
				// pipeline. Link at lld's numeric O2 level here, then exercise
				// the planned post-PCLN compaction below. The policy tests above
				// cover automatic Os/Oz selection.
				level = optlevel.O2
			}
			cfg := &Config{
				Mode:               ModeBuild,
				OutFile:            binaryPath,
				OptLevel:           level,
				LTO:                tt.lto,
				OmitDWARFByDefault: true,
				PCLNMode:           tt.pcln,
				PCLNModeSet:        true,
			}
			if _, err := Do([]string{"./testdata/ldflagsstrip"}, cfg); err != nil {
				t.Fatal(err)
			}
			if tt.lto.Enabled() {
				ctx := &context{stripDarwinLTOLocals: true}
				if err := finalizeDarwinSizeExecutable(ctx, binaryPath, false); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(binaryPath)
			cmd.Env = append(os.Environ(), "LLGO_FUNCINFO_DEBUG=1")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run compact binary: %v\n%s", err, output)
			}
			if !strings.Contains(string(output), "main.caller main.go true") ||
				(tt.pcln == PCLNEmbedded && !strings.Contains(string(output), "entries= prebuilt")) {
				t.Fatalf("runtime PCLN output:\n%s", output)
			}
			if output, err := exec.Command("codesign", "--verify", "--verbose=4", binaryPath).CombinedOutput(); err != nil {
				t.Fatalf("codesign verification: %v\n%s", err, output)
			}
			if tt.pcln == PCLNExternal {
				if _, err := os.Stat(pclnSidecarPath(binaryPath)); err != nil {
					t.Fatalf("external PCLN sidecar: %v", err)
				}
			}

			file, err := macho.Open(binaryPath)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if file.Symtab == nil {
				t.Fatal("Mach-O has no symbol table")
			}
			localDefined := 0
			externalDefined := 0
			for _, symbol := range file.Symtab.Syms {
				if symbol.Sect == 0 {
					continue
				}
				if symbol.Type&0x01 != 0 && symbol.Type&0x10 == 0 { // N_EXT without N_PEXT
					externalDefined++
				} else {
					localDefined++
				}
			}
			if tt.wantLocal {
				if localDefined == 0 || externalDefined != 0 {
					t.Fatalf("non-LTO defined symbols: local=%d external=%d", localDefined, externalDefined)
				}
			} else if localDefined != 0 {
				t.Fatalf("LTO retained %d local defined symbols", localDefined)
			}
		})
	}
}

func withLTO(conf Config, mode lto.Mode) Config {
	conf.LTO = mode
	return conf
}

func withOptLevel(conf Config, level optlevel.Level) Config {
	conf.OptLevel = level
	return conf
}

func withTarget(conf Config, target string) Config {
	conf.Target = target
	return conf
}

func withBuildMode(conf Config, mode BuildMode) Config {
	conf.BuildMode = mode
	return conf
}

func withDWARF(conf Config, mode DWARFMode) Config {
	conf.LinkOptions.DWARF = mode
	return conf
}

func withPCLN(conf Config, mode PCLNMode) Config {
	conf.PCLNMode = mode
	return conf
}
