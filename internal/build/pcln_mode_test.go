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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	buildfuncinfo "github.com/xgo-dev/llgo/internal/build/funcinfo"
	"github.com/xgo-dev/llgo/internal/pclnmap"
)

func TestPCLNModeStringAndValidity(t *testing.T) {
	tests := []struct {
		mode  PCLNMode
		want  string
		valid bool
	}{
		{mode: PCLNEmbedded, want: "embedded", valid: true},
		{mode: PCLNExternal, want: "external", valid: true},
		{mode: PCLNNone, want: "none", valid: true},
		{mode: PCLNMode(255), want: "PCLNMode(255)", valid: false},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("PCLNMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
		if got := tt.mode.IsValid(); got != tt.valid {
			t.Errorf("PCLNMode(%d).IsValid() = %v, want %v", tt.mode, got, tt.valid)
		}
	}
}

func TestValidatePCLNMode(t *testing.T) {
	tests := []struct {
		name          string
		conf          Config
		funcInfoSites string
		wantErr       bool
	}{
		{name: "embedded linux executable", conf: Config{Goos: "linux", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded}},
		{name: "embedded named target", conf: Config{Goos: "wasip1", Target: "wasi", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded}},
		{name: "embedded shared library", conf: Config{Goos: "windows", BuildMode: BuildModeCShared, PCLNMode: PCLNEmbedded}},
		{name: "embedded allows sites disabled", conf: Config{Goos: "linux", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded}, funcInfoSites: "0"},
		{name: "none named target", conf: Config{Goos: "wasip1", Target: "wasi", BuildMode: BuildModeExe, PCLNMode: PCLNNone}},
		{name: "none archive", conf: Config{Goos: "linux", BuildMode: BuildModeCArchive, PCLNMode: PCLNNone}},
		{name: "none allows sites disabled", conf: Config{Goos: "linux", BuildMode: BuildModeExe, PCLNMode: PCLNNone}, funcInfoSites: "0"},
		{name: "external linux amd64", conf: Config{Goos: "linux", Goarch: "amd64", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}},
		{name: "external linux arm64", conf: Config{Goos: "linux", Goarch: "arm64", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}},
		{name: "external darwin amd64", conf: Config{Goos: "darwin", Goarch: "amd64", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}},
		{name: "external darwin arm64", conf: Config{Goos: "darwin", Goarch: "arm64", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}},
		{name: "external unsupported OS", conf: Config{Goos: "windows", Goarch: "amd64", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}, wantErr: true},
		{name: "external unsupported linux arch", conf: Config{Goos: "linux", Goarch: "386", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}, wantErr: true},
		{name: "external unsupported darwin arch", conf: Config{Goos: "darwin", Goarch: "riscv64", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}, wantErr: true},
		{name: "external named target", conf: Config{Goos: "wasip1", Goarch: "wasm", Target: "wasi", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}, wantErr: true},
		{name: "external archive", conf: Config{Goos: "linux", Goarch: "amd64", BuildMode: BuildModeCArchive, PCLNMode: PCLNExternal}, wantErr: true},
		{name: "external shared library", conf: Config{Goos: "darwin", Goarch: "arm64", BuildMode: BuildModeCShared, PCLNMode: PCLNExternal}, wantErr: true},
		{name: "external generation mode", conf: Config{Goos: "linux", Goarch: "amd64", Mode: ModeGen, BuildMode: BuildModeExe, PCLNMode: PCLNExternal}, wantErr: true},
		{name: "external requires sites", conf: Config{Goos: "linux", Goarch: "amd64", BuildMode: BuildModeExe, PCLNMode: PCLNExternal}, funcInfoSites: "0", wantErr: true},
		{name: "invalid mode", conf: Config{Goos: "linux", BuildMode: BuildModeExe, PCLNMode: PCLNMode(255)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(llgoFuncInfoSites, tt.funcInfoSites)
			err := validatePCLNMode(&tt.conf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validatePCLNMode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.funcInfoSites == "0" && err != nil && !strings.Contains(err.Error(), llgoFuncInfoSites) {
				t.Fatalf("validatePCLNMode() error = %v, want %s diagnostic", err, llgoFuncInfoSites)
			}
		})
	}
}

func TestEffectivePCLNModeLegacyPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		funcInfo string
		conf     Config
		want     PCLNMode
	}{
		{name: "default", conf: Config{}, want: PCLNEmbedded},
		{name: "legacy enabled", funcInfo: "1", conf: Config{}, want: PCLNEmbedded},
		{name: "legacy disabled", funcInfo: "0", conf: Config{}, want: PCLNNone},
		{name: "explicit embedded wins", funcInfo: "0", conf: Config{PCLNMode: PCLNEmbedded, PCLNModeSet: true}, want: PCLNEmbedded},
		{name: "explicit external wins", funcInfo: "0", conf: Config{PCLNMode: PCLNExternal, PCLNModeSet: true}, want: PCLNExternal},
		{name: "explicit none wins", funcInfo: "1", conf: Config{PCLNMode: PCLNNone, PCLNModeSet: true}, want: PCLNNone},
		{name: "typed external is authoritative", funcInfo: "0", conf: Config{PCLNMode: PCLNExternal}, want: PCLNExternal},
		{name: "typed none is authoritative", funcInfo: "1", conf: Config{PCLNMode: PCLNNone}, want: PCLNNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(llgoFuncInfo, tt.funcInfo)
			if got := effectivePCLNMode(&tt.conf); got != tt.want {
				t.Fatalf("effectivePCLNMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldEnablePCLNSites(t *testing.T) {
	t.Setenv(llgoFuncInfoSites, "1")
	tests := []struct {
		name      string
		conf      Config
		funcInfo  bool
		debugInfo bool
		want      bool
	}{
		{name: "embedded without debug", conf: Config{Goos: "darwin", PCLNMode: PCLNEmbedded}, funcInfo: true, want: true},
		{name: "darwin embedded debug", conf: Config{Goos: "darwin", PCLNMode: PCLNEmbedded}, funcInfo: true, debugInfo: true},
		{name: "linux embedded debug", conf: Config{Goos: "linux", PCLNMode: PCLNEmbedded}, funcInfo: true, debugInfo: true, want: true},
		{name: "external debug", conf: Config{Goos: "darwin", PCLNMode: PCLNExternal}, funcInfo: true, debugInfo: true, want: true},
		{name: "metadata disabled", conf: Config{Goos: "linux", PCLNMode: PCLNEmbedded}, debugInfo: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldEnablePCLNSites(&tt.conf, tt.funcInfo, tt.debugInfo); got != tt.want {
				t.Fatalf("shouldEnablePCLNSites() = %v, want %v", got, tt.want)
			}
		})
	}
	t.Setenv(llgoFuncInfoSites, "0")
	if shouldEnablePCLNSites(&Config{Goos: "linux", PCLNMode: PCLNExternal}, true, true) {
		t.Fatal("LLGO_FUNCINFO_SITES=0 did not disable sites")
	}
}

func TestNewDefaultConfMetadataDefaults(t *testing.T) {
	t.Setenv("GOBIN", t.TempDir())
	t.Setenv(llgoFuncInfo, "")
	conf := NewDefaultConf(ModeBuild)
	if got := conf.PCLNMode; got != PCLNEmbedded {
		t.Fatalf("NewDefaultConf().PCLNMode = %v, want %v", got, PCLNEmbedded)
	}
	if conf.PCLNModeSet {
		t.Fatal("NewDefaultConf().PCLNModeSet = true, want unresolved legacy default")
	}
	if !conf.OmitDWARFByDefault {
		t.Fatal("NewDefaultConf().OmitDWARFByDefault = false, want safe provisional-DWARF default")
	}
	genConf := NewDefaultConf(ModeGen)
	if genConf.OmitDWARFByDefault {
		t.Fatal("NewDefaultConf(ModeGen).OmitDWARFByDefault = true, want no linked-build policy")
	}
}

func TestDoValidatesPCLNMode(t *testing.T) {
	_, err := Do(nil, &Config{PCLNMode: PCLNMode(255)})
	if err == nil {
		t.Fatal("Do() succeeded with an invalid PCLN mode")
	}
	if !strings.Contains(err.Error(), "invalid PCLN mode") {
		t.Fatalf("Do() error = %v, want invalid PCLN mode", err)
	}
}

func TestDoNormalizesLegacyPCLNMode(t *testing.T) {
	t.Setenv(llgoFuncInfo, "0")
	tests := []struct {
		name string
		conf Config
		want PCLNMode
	}{
		{name: "legacy default", conf: Config{}, want: PCLNNone},
		{name: "explicit embedded", conf: Config{PCLNMode: PCLNEmbedded, PCLNModeSet: true}, want: PCLNEmbedded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := tt.conf
			before := conf
			// Stop after PCLN normalization without setting up a toolchain.
			conf.LinkOptions.DWARF = DWARFMode(255)
			before.LinkOptions.DWARF = DWARFMode(255)
			if _, err := Do(nil, &conf); err == nil {
				t.Fatal("Do() succeeded with an invalid DWARF mode")
			}
			if !reflect.DeepEqual(conf, before) {
				t.Fatalf("Do() modified input config:\n got: %#v\nwant: %#v", conf, before)
			}
			conf.LinkOptions.DWARF = DWARFDefault
			resolved, err := resolveBuildConfig(&conf)
			if err != nil {
				t.Fatal(err)
			}
			if resolved.PCLNMode != tt.want || !resolved.PCLNModeSet {
				t.Fatalf("resolved PCLN config = (%v, set=%v), want (%v, set=true)", resolved.PCLNMode, resolved.PCLNModeSet, tt.want)
			}
		})
	}
}

func TestFinalizeRuntimePCLNRemovesStaleSidecar(t *testing.T) {
	for _, mode := range []PCLNMode{PCLNEmbedded, PCLNNone} {
		t.Run(mode.String(), func(t *testing.T) {
			executable := filepath.Join(t.TempDir(), "app")
			sidecar := pclnSidecarPath(executable)
			if err := os.WriteFile(executable, []byte("executable"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(sidecar, []byte("stale"), 0o644); err != nil {
				t.Fatal(err)
			}
			ctx := &context{buildConf: &Config{PCLNMode: mode}}
			if err := finalizeRuntimePCLN(ctx, &OutFmtDetails{Out: executable}, false); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(executable); err != nil {
				t.Fatalf("finalize removed executable: %v", err)
			}
			if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
				t.Fatalf("stale sidecar still exists: %v", err)
			}
		})
	}
}

func TestFilterExternalPCLNJoins(t *testing.T) {
	idA := funcInfoSymbolID("example.com/p.A")
	idB := funcInfoSymbolID("example.com/p.B")
	data := pclnmap.Data{
		GOOS: "darwin",
		Table: buildfuncinfo.Table{PCLines: []buildfuncinfo.EncodedPCLineRecord{
			{ID: 101, Func: 1},
			{ID: 202, Func: 2},
		}},
		SymbolIndex: []pclnmap.SymbolIndexEntry{
			{SymbolID: idA, FuncIndex: 1},
			{SymbolID: idB, FuncIndex: 2},
		},
		EntrySites: []pclnmap.Site{
			{PCOffset: 0x100, ID: idA},
			{PCOffset: 0x100, ID: idB}, // same-PC alias is deterministically dropped
			{PCOffset: 0x200, ID: 99},  // missing funcinfo join
		},
		PCSites: []pclnmap.Site{
			{PCOffset: 0x110, ID: 101},
			{PCOffset: 0x120, ID: 101}, // A's pcline copied into B
			{PCOffset: 0x130, ID: 202},
		},
	}
	if err := filterExternalPCLNJoins(&data, []string{"__example.com/p.A", "_example.com/p.B", "_example.com/p.B"}); err != nil {
		t.Fatal(err)
	}
	if len(data.EntrySites) != 1 || data.EntrySites[0] != (pclnmap.Site{PCOffset: 0x100, ID: idA}) {
		t.Fatalf("entry sites = %#v", data.EntrySites)
	}
	wantPCSites := []pclnmap.Site{{PCOffset: 0x110, ID: 101}, {PCOffset: 0x130, ID: 202}}
	if !reflect.DeepEqual(data.PCSites, wantPCSites) {
		t.Fatalf("pcline sites = %#v, want %#v", data.PCSites, wantPCSites)
	}
}
