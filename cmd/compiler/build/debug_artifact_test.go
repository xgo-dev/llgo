//go:build !llgo

package build

import (
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/crosscompile"
)

func TestDebugArtifactMode(t *testing.T) {
	tests := []struct {
		mode  DebugArtifactMode
		name  string
		valid bool
	}{
		{DebugArtifactDefault, "default", true},
		{DebugArtifactEmbedded, "embedded", true},
		{DebugArtifactExternal, "external", true},
		{DebugArtifactHost, "host", true},
		{DebugArtifactNone, "none", true},
		{DebugArtifactMode(255), "DebugArtifactMode(255)", false},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.name {
			t.Errorf("DebugArtifactMode(%d).String() = %q, want %q", tt.mode, got, tt.name)
		}
		if got := tt.mode.IsValid(); got != tt.valid {
			t.Errorf("DebugArtifactMode(%d).IsValid() = %v, want %v", tt.mode, got, tt.valid)
		}
	}
}

func TestResolveDebugArtifactMode(t *testing.T) {
	native := crosscompile.Export{LLVMTarget: "arm64-apple-darwin"}
	wasm := crosscompile.Export{LLVMTarget: "wasm32-wasip1"}
	fixed := crosscompile.Export{LLVMTarget: "thumbv7em-none-unknown-eabi"}
	base := func() Config {
		return Config{Mode: ModeBuild, BuildMode: BuildModeExe}
	}
	tests := []struct {
		name      string
		conf      Config
		target    crosscompile.Export
		wantMode  DebugArtifactMode
		wantDWARF DWARFMode
		wantErr   bool
	}{
		{name: "safe default", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, OmitDWARFByDefault: true}, target: native, wantMode: DebugArtifactNone},
		{name: "native explicit preserve overrides safe default", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, OmitDWARFByDefault: true, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: native, wantMode: DebugArtifactEmbedded, wantDWARF: DWARFPreserve},
		{name: "native default with DWARF", conf: base(), target: native, wantMode: DebugArtifactEmbedded},
		{name: "derived host when DWARF requested", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "rp2040", LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: fixed, wantMode: DebugArtifactHost, wantDWARF: DWARFPreserve},
		{name: "fixed explicit preserve overrides safe default", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "rp2040", OmitDWARFByDefault: true, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: fixed, wantMode: DebugArtifactHost, wantDWARF: DWARFPreserve},
		{name: "derived embedded when DWARF requested", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "wasi", Goarch: "wasm", LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: wasm, wantMode: DebugArtifactEmbedded, wantDWARF: DWARFPreserve},
		{name: "explicit none", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactNone, DebugArtifactModeSet: true}, target: native, wantMode: DebugArtifactNone, wantDWARF: DWARFOmit},
		{name: "none conflicts preserve", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactNone, DebugArtifactModeSet: true, LinkOptions: LinkOptions{DWARF: DWARFPreserve}}, target: native, wantErr: true},
		{name: "embedded native", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactEmbedded, DebugArtifactModeSet: true}, target: native, wantMode: DebugArtifactEmbedded, wantDWARF: DWARFPreserve},
		{name: "embedded overrides s implication", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactEmbedded, DebugArtifactModeSet: true, LinkOptions: LinkOptions{OmitSymbolTable: true}}, target: native, wantMode: DebugArtifactEmbedded, wantDWARF: DWARFPreserve},
		{name: "embedded conflicts w", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactEmbedded, DebugArtifactModeSet: true, LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: native, wantErr: true},
		{name: "embedded rejects generation", conf: Config{Mode: ModeGen, DebugArtifactMode: DebugArtifactEmbedded, DebugArtifactModeSet: true}, target: native, wantErr: true},
		{name: "embedded rejects fixed", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "rp2040", DebugArtifactMode: DebugArtifactEmbedded, DebugArtifactModeSet: true}, target: fixed, wantErr: true},
		{name: "host fixed", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "rp2040", DebugArtifactMode: DebugArtifactHost, DebugArtifactModeSet: true}, target: fixed, wantMode: DebugArtifactHost, wantDWARF: DWARFPreserve},
		{name: "host rejects native", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactHost, DebugArtifactModeSet: true}, target: native, wantErr: true},
		{name: "host rejects wasm", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "wasi", Goarch: "wasm", DebugArtifactMode: DebugArtifactHost, DebugArtifactModeSet: true}, target: wasm, wantErr: true},
		{name: "host rejects archive", conf: Config{Mode: ModeBuild, BuildMode: BuildModeCArchive, Target: "rp2040", DebugArtifactMode: DebugArtifactHost, DebugArtifactModeSet: true}, target: fixed, wantErr: true},
		{name: "host conflicts w", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "rp2040", DebugArtifactMode: DebugArtifactHost, DebugArtifactModeSet: true, LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: fixed, wantErr: true},
		{name: "external wasm", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "wasi", Goarch: "wasm", DebugArtifactMode: DebugArtifactExternal, DebugArtifactModeSet: true}, target: wasm, wantMode: DebugArtifactExternal, wantDWARF: DWARFPreserve},
		{name: "external rejects native", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactExternal, DebugArtifactModeSet: true}, target: native, wantErr: true},
		{name: "external rejects archive", conf: Config{Mode: ModeBuild, BuildMode: BuildModeCArchive, Target: "wasi", Goarch: "wasm", DebugArtifactMode: DebugArtifactExternal, DebugArtifactModeSet: true}, target: wasm, wantErr: true},
		{name: "external conflicts w", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, Target: "wasi", Goarch: "wasm", DebugArtifactMode: DebugArtifactExternal, DebugArtifactModeSet: true, LinkOptions: LinkOptions{DWARF: DWARFOmit}}, target: wasm, wantErr: true},
		{name: "explicit default rejected", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactDefault, DebugArtifactModeSet: true}, target: native, wantErr: true},
		{name: "invalid", conf: Config{Mode: ModeBuild, BuildMode: BuildModeExe, DebugArtifactMode: DebugArtifactMode(255)}, target: native, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := tt.conf
			err := resolveDebugArtifactMode(&conf, &tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveDebugArtifactMode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if conf.DebugArtifactMode != tt.wantMode || conf.LinkOptions.DWARF != tt.wantDWARF {
				t.Fatalf("resolved mode/options = %v/%v, want %v/%v", conf.DebugArtifactMode, conf.LinkOptions.DWARF, tt.wantMode, tt.wantDWARF)
			}
		})
	}
}

func TestDebugArtifactTargetDefaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		goarch string
		triple string
	}{
		{name: "rp2040", goarch: "arm", triple: "thumbv6m-none-unknown-eabi"},
		{name: "wasi", goarch: "wasm", triple: "wasm32-wasip1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.Target = tc.name
			conf.Goarch = tc.goarch
			if err := resolveDebugArtifactMode(conf, &crosscompile.Export{LLVMTarget: tc.triple}); err != nil {
				t.Fatal(err)
			}
			if conf.DebugArtifactMode != DebugArtifactNone || conf.LinkOptions.DWARF != DWARFDefault {
				t.Fatalf("target defaults resolved to %v/%v, want none/default", conf.DebugArtifactMode, conf.LinkOptions.DWARF)
			}
		})
	}
}
