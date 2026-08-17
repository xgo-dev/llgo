//go:build !llgo

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
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/xtool/env/llvm"
)

func TestDwarfLinkerArgs(t *testing.T) {
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
		{name: "target linker already suppresses DWARF", conf: Config{Target: "rp2040", LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: alwaysOmitDebugInfo()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dwarfLinkerArgs(&tt.conf, &tt.target); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("dwarfLinkerArgs(%+v) = %v, want %v", tt.conf.LinkOptions, got, tt.want)
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
		{name: "target baseline", target: alwaysOmitDebugInfo(), want: true},
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
		{name: "target always omits", conf: Config{Mode: ModeBuild}, target: alwaysOmitDebugInfo()},
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
		{name: "c-shared omit", conf: Config{Goos: "linux", BuildMode: BuildModeCShared, LinkOptions: w}, target: configurableDebugInfo()},
		{name: "c-archive omit", conf: Config{Goos: "linux", BuildMode: BuildModeCArchive, LinkOptions: w}, target: configurableDebugInfo()},
		{name: "c-shared preserve", conf: Config{Goos: "linux", BuildMode: BuildModeCShared, LinkOptions: wFalse}, target: configurableDebugInfo()},
		{name: "c-archive preserve", conf: Config{Goos: "linux", BuildMode: BuildModeCArchive, LinkOptions: wFalse}, target: configurableDebugInfo()},
		{name: "fixed target omit", conf: Config{Target: "rp2040", Goos: "linux", BuildMode: BuildModeExe, LinkOptions: w}, target: alwaysOmitDebugInfo()},
		{name: "fixed target explicit DWARF", conf: Config{Target: "rp2040", Goos: "linux", BuildMode: BuildModeExe, LinkOptions: wFalse}, target: alwaysOmitDebugInfo(), wantErr: true},
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
	args := append(dwarfLinkerArgs(conf, &target), "-o", bin, object)
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

func configurableDebugInfo() crosscompile.Export {
	return crosscompile.Export{DebugInfo: crosscompile.DebugInfoPolicy{OmitLinkFlags: []string{"-Wl,-S"}}}
}

func alwaysOmitDebugInfo() crosscompile.Export {
	return crosscompile.Export{DebugInfo: crosscompile.DebugInfoPolicy{AlwaysOmit: true}}
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
