// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package build

import (
	"bytes"
	"debug/dwarf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
)

func TestNativeDWARFCompressionIOErrors(t *testing.T) {
	root := t.TempDir()
	if err := compressNativeArtifact(filepath.Join(root, "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing input: %v", err)
	}
	output := filepath.Join(root, "program")
	if err := os.MkdirAll(dsymMarkerPath(nativeDSYMPath(output)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := removeGeneratedDSYM(output); err == nil {
		t.Fatal("unreadable ownership marker was ignored")
	}
	ctx := &context{commands: commandEnv{environ: os.Environ()}}
	if err := compressDarwinDSYM(ctx, filepath.Join(root, "missing-parent", "program"), false); err == nil {
		t.Fatal("invalid staging directory was accepted")
	}
	// A linked image without DWARF is a no-op, including its modification time.
	raw := make([]byte, 512)
	copy(raw, "MZ")
	binary.LittleEndian.PutUint32(raw[60:], 0x80)
	copy(raw[0x80:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(raw[0x84:], 0x8664)
	binary.LittleEndian.PutUint16(raw[0x84+16:], 240)
	binary.LittleEndian.PutUint16(raw[0x98:], 0x20b)
	binary.LittleEndian.PutUint32(raw[0x98+36:], 512)
	binary.LittleEndian.PutUint32(raw[0x98+108:], 16)
	if err := os.WriteFile(output, raw, 0o751); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1_600_000_000, 0)
	if err := os.Chtimes(output, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if err := compressNativeArtifact(output); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || !info.ModTime().Equal(stamp) {
		t.Fatalf("no-op compression replaced the file: %v", err)
	}
}

func TestRejectMissingDSYMDWARF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.dwarf")
	for _, contents := range [][]byte{[]byte("not Mach-O"), emptyDSYMFixture(false), emptyDSYMFixture(true)} {
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := validateDSYMDWARF(path); err == nil {
			t.Fatal("missing or empty DWARF was accepted")
		}
	}
}

func emptyDSYMFixture(emptyUnit bool) []byte {
	// A minimal dSYM with either no sections or one header-only compilation unit.
	raw := make([]byte, 32)
	binary.LittleEndian.PutUint32(raw, 0xfeedfacf)
	binary.LittleEndian.PutUint32(raw[4:], 0x100000c)
	binary.LittleEndian.PutUint32(raw[12:], 0xa)
	if !emptyUnit {
		return raw
	}
	const commandSize = 72 + 2*80
	raw = append(raw, make([]byte, commandSize+12)...)
	u32 := func(p int, v uint32) { binary.LittleEndian.PutUint32(raw[p:], v) }
	u64 := func(p int, v uint64) { binary.LittleEndian.PutUint64(raw[p:], v) }
	u32(16, 1)
	u32(20, commandSize)
	u32(32, 0x19)
	u32(36, commandSize)
	copy(raw[40:], "__DWARF")
	u64(32+40, 32+commandSize)
	u64(32+48, 12)
	u32(32+64, 2)
	for i, name := range []string{"__debug_abbrev", "__debug_info"} {
		p := 32 + 72 + i*80
		copy(raw[p:p+16], name)
		copy(raw[p+16:p+32], "__DWARF")
		u64(p+40, uint64(1+10*i))
		u32(p+48, uint32(32+commandSize+i))
	}
	copy(raw[32+commandSize+1:], []byte{7, 0, 0, 0, 4, 0, 0, 0, 0, 0, 8})
	return raw
}

func TestNativeDWARFCompressionPolicy(t *testing.T) {
	for _, name := range []string{"default", "disabled", "disabled PE", "missing PE", "omit", "archive", "named-target", "ELF"} {
		t.Run(name, func(t *testing.T) {
			conf := Config{Goos: "darwin", LinkOptions: LinkOptions{DWARF: DWARFPreserve, DWARFCompression: DWARFCompressionEnabled}}
			target := configurableDebugInfo()
			target.Toolchain.ObjectFormat = crosscompile.ObjectFormatMachO
			switch name {
			case "default":
				conf.LinkOptions.DWARFCompression = DWARFCompressionDefault
			case "disabled":
				conf.LinkOptions.DWARFCompression = DWARFCompressionDisabled
			case "disabled PE":
				conf.LinkOptions.DWARFCompression = DWARFCompressionDisabled
				target.Toolchain.ObjectFormat = crosscompile.ObjectFormatCOFF
			case "missing PE":
				target.Toolchain.ObjectFormat = crosscompile.ObjectFormatCOFF
			case "omit":
				conf.LinkOptions.DWARF = DWARFOmit
			case "archive":
				conf.BuildMode = BuildModeCArchive
			case "named-target":
				conf.Target = "custom"
			case "ELF":
				target.Toolchain.ObjectFormat = crosscompile.ObjectFormatELF
			}
			// A skipped policy must not open the missing artifact or run dsymutil.
			err := finalizeNativeDWARFCompression(&context{buildConf: &conf, crossCompile: target}, filepath.Join(t.TempDir(), "missing"), false)
			if name == "missing PE" {
				if !os.IsNotExist(err) {
					t.Fatalf("missing PE image: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDarwinDSYMRejectsInvalidPublication(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires native Mach-O dsymutil")
	}
	root := t.TempDir()
	source, output := filepath.Join(root, "probe.c"), filepath.Join(root, "probe")
	if err := os.WriteFile(source, []byte("int main(void) { return 0; }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	object := filepath.Join(root, "probe.o")
	if data, err := exec.Command("clang", "-g", "-c", source, "-o", object).CombinedOutput(); err != nil {
		t.Fatalf("build debug fixture: %v: %s", err, data)
	}
	if data, err := exec.Command("clang", object, "-o", output).CombinedOutput(); err != nil {
		t.Fatalf("link debug fixture: %v: %s", err, data)
	}
	destination := nativeDSYMPath(output)
	if err := os.WriteFile(destination, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := &context{commands: commandEnv{environ: os.Environ()}}
	if err := compressDarwinDSYM(ctx, output, true); err == nil || !strings.Contains(err.Error(), "non-directory") {
		t.Fatalf("occupied publication path: %v", err)
	}
	plain := filepath.Join(root, "plain.dwarf")
	if data, err := exec.Command("dsymutil", "-f", "-o", plain, output).CombinedOutput(); err != nil {
		t.Fatalf("generate debug fixture: %v: %s", err, data)
	}
	raw, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	// Valid DWARF in an executable is not an acceptable compressible dSYM.
	binary.LittleEndian.PutUint32(raw[12:], uint32(macho.TypeExec))
	if err := os.WriteFile(plain, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	fakeBin := filepath.Join(root, "tools")
	if err := os.Mkdir(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nmkdir -p \"$3/Contents/Resources/DWARF\"\ncp \"$LLGO_TEST_DSYM\" \"$3/Contents/Resources/DWARF/${1##*/}\"\n"
	if err := os.WriteFile(filepath.Join(fakeBin, "dsymutil"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx.commands.environ = append(os.Environ(), "LLGO_TEST_DSYM="+plain)
	if err := compressDarwinDSYM(ctx, output, false); err == nil || !strings.Contains(err.Error(), "requires a dSYM") {
		t.Fatalf("invalid staged dSYM: %v", err)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "occupied" {
		t.Fatalf("failed publication replaced the destination: %q, %v", data, err)
	}
}

func TestNativeDWARFCompressionRejectsWithoutReplacing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.exe")
	original := []byte("MZ invalid input")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := compressNativeArtifact(path); err == nil {
		t.Fatal("invalid input accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("failed compression replaced original: %q, %v", got, err)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 {
		t.Fatalf("failed compression leaked temporary files: %v, %v", files, err)
	}
}

func TestGeneratedDSYMLifecycle(t *testing.T) {
	for _, generated := range []bool{false, true} {
		output := filepath.Join(t.TempDir(), "program")
		marker := dsymMarkerPath(nativeDSYMPath(output))
		if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
			t.Fatal(err)
		}
		contents := "manual bundle"
		if generated {
			contents = generatedDSYMMarker
		}
		if err := os.WriteFile(marker, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
		conf := Config{Goos: "darwin", LinkOptions: LinkOptions{DWARFCompression: DWARFCompressionDisabled}}
		target := configurableDebugInfo()
		target.Toolchain.ObjectFormat = crosscompile.ObjectFormatMachO
		if err := finalizeNativeDWARFCompression(&context{buildConf: &conf, crossCompile: target}, output, false); err != nil {
			t.Fatal(err)
		}
		_, err := os.Stat(nativeDSYMPath(output))
		if os.IsNotExist(err) != generated {
			t.Fatalf("dSYM removal did not respect ownership (generated=%v): %v", generated, err)
		}
	}
}

func TestDarwinDSYMFailurePreservesPrevious(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires dsymutil")
	}
	root := t.TempDir()
	output := filepath.Join(root, "missing-executable")
	previous := filepath.Join(nativeDSYMPath(output), "previous")
	if err := os.MkdirAll(filepath.Dir(previous), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(previous, []byte("older bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := &context{commands: commandEnv{dir: root, environ: os.Environ()}}
	if err := compressDarwinDSYM(ctx, output, false); err == nil {
		t.Fatal("missing debug map was accepted")
	}
	got, err := os.ReadFile(previous)
	if err != nil || string(got) != "older bundle" {
		t.Fatalf("failed generation damaged older bundle: %q, %v", got, err)
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("failed generation leaked staging files: %v, %v", files, err)
	}
}

func TestDarwinDSYMLeadingDashOutput(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires native Mach-O dsymutil")
	}
	t.Chdir(t.TempDir())
	if err := os.WriteFile("main.c", []byte("int main(void) { return 0; }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-g", "-c", "main.c", "-o", "main.o"}, {"main.o", "-o", "-program"}} {
		if data, err := exec.Command("clang", args...).CombinedOutput(); err != nil {
			t.Fatalf("compile debug fixture: %v: %s", err, data)
		}
	}
	ctx := &context{commands: commandEnv{environ: os.Environ()}}
	if err := compressDarwinDSYM(ctx, "-program", false); err != nil {
		t.Fatal(err)
	}
	if err := validateDSYMDWARF(filepath.Join("-program.dSYM", "Contents", "Resources", "DWARF", "-program")); err != nil {
		t.Fatal(err)
	}
	// The stripping path must pass the same safe relative input to dsymutil.
	ctx.buildConf = &Config{Goos: "darwin", LinkOptions: LinkOptions{DWARF: DWARFPreserve}}
	stop := errors.New("stop after checking dsymutil input")
	err := stripNativeSymbolsWith(ctx, &OutFmtDetails{Out: "-program", DSYM: "-program.dSYM"}, nil, false, func(name string, args ...string) error {
		if name != "dsymutil" || len(args) == 0 || args[0] != "./-program" {
			t.Fatalf("unsafe dsymutil invocation: %s %q", name, args)
		}
		return stop
	})
	if !errors.Is(err, stop) {
		t.Fatalf("unexpected stripping result: %v", err)
	}
}

func TestDarwinNativeDWARFCompression(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native dSYM integration")
	}
	for _, mode := range []lto.Mode{lto.Off, lto.Full, lto.Thin} {
		t.Run(mode.String(), func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "compressed-dwarf")
			conf := &Config{Mode: ModeBuild, OutFile: bin, LTO: mode, LinkOptions: LinkOptions{DWARF: DWARFPreserve, DWARFCompression: DWARFCompressionEnabled}}
			if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
				t.Fatal(err)
			}
			if out := runBinary(t, bin); !strings.Contains(out, "main.caller main.go true") {
				t.Fatalf("runtime PCLN output: %s", out)
			}
			verifyPCLNIntegrationSignature(t, bin)
			path := filepath.Join(nativeDSYMPath(bin), "Contents", "Resources", "DWARF", filepath.Base(bin))
			f, err := macho.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if f.Section("__zdebug_info") == nil {
				t.Fatal("generated dSYM lacks compressed debug info")
			}
			data, err := f.DWARF()
			if err != nil {
				t.Fatal(err)
			}
			if findDWARFNode(readDWARFTree(t, data), dwarf.TagSubprogram, "main.caller") == nil {
				t.Fatal("compressed dSYM lost main.caller")
			}
			plain := filepath.Join(t.TempDir(), "plain.dwarf")
			if out, err := exec.Command("dsymutil", "-f", "-o", plain, bin).CombinedOutput(); err != nil {
				t.Fatalf("dsymutil: %v: %s", err, out)
			}
			a, _ := os.Stat(plain)
			b, _ := os.Stat(path)
			if a == nil || b == nil || b.Size() >= a.Size() {
				t.Fatal("dSYM compression did not reduce file size")
			}
			t.Logf("dSYM: %d -> %d bytes", a.Size(), b.Size())
			// Rebuild also exercises replacing a previously published bundle.
			if mode == lto.Off {
				if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
					t.Fatal(err)
				}
				if err := validateDSYMDWARF(path); err != nil {
					t.Fatal(err)
				}
				conf.LinkOptions.DWARFCompression = DWARFCompressionDisabled
				if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(nativeDSYMPath(bin)); !os.IsNotExist(err) {
					t.Fatalf("opt-out retained a compressed dSYM that masks the debug map: %v", err)
				}
			}
			f.Close()
			removeOutFmts(&OutFmtDetails{Out: bin, DSYM: nativeDSYMPath(bin)})
			if _, err := os.Stat(nativeDSYMPath(bin)); !os.IsNotExist(err) {
				t.Fatalf("temporary-output cleanup retained dSYM: %v", err)
			}
		})
	}
}

func TestCompressedDSYMOutput(t *testing.T) {
	conf := &Config{Mode: ModeBuild, Goos: "darwin", DebugArtifactMode: DebugArtifactEmbedded, OutFile: filepath.Join(t.TempDir(), "program"), LinkOptions: LinkOptions{DWARF: DWARFPreserve, DWARFCompression: DWARFCompressionEnabled}}
	target := configurableDebugInfo()
	target.Toolchain.ObjectFormat = crosscompile.ObjectFormatMachO
	out, err := buildOutFmts("program", conf, false, &target)
	if err != nil {
		t.Fatal(err)
	}
	if out.DSYM != nativeDSYMPath(out.Out) {
		t.Fatalf("compressed dSYM is missing from owned outputs: %+v", out)
	}
	if err := os.WriteFile(out.Out, []byte("exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out.DSYM, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out.DSYM, "debug"), []byte("dwarf"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifacts, err := CollectArtifacts(conf, out)
	if err != nil || len(artifacts) != 2 || artifacts[1].Format != "dsym" || artifacts[1].Size != 5 {
		t.Fatalf("compressed dSYM was not reported: %v, %v", artifacts, err)
	}
	removeOutFmts(out)
	if _, err := os.Stat(out.DSYM); !os.IsNotExist(err) {
		t.Fatalf("cleanup retained the compressed dSYM: %v", err)
	}
}

func TestCompressedSymbolStripFailurePreservesPrevious(t *testing.T) {
	root := t.TempDir()
	out := &OutFmtDetails{Out: filepath.Join(root, "program"), DSYM: filepath.Join(root, "program.dSYM")}
	if err := os.WriteFile(out.Out, []byte("linked executable"), 0o751); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(out.DSYM, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := filepath.Join(out.DSYM, "previous")
	if err := os.WriteFile(previous, []byte("older bundle"), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := &Config{Goos: "darwin", LinkOptions: LinkOptions{DWARF: DWARFPreserve, DWARFCompression: DWARFCompressionEnabled}}
	ctx := &context{buildConf: conf, crossCompile: configurableDebugInfo()}
	run := func(name string, args ...string) error {
		if name != "dsymutil" {
			t.Fatalf("reached %s after generating a dSYM without DWARF", name)
		}
		return os.Mkdir(args[2], 0o755)
	}
	if err := stripNativeSymbolsWith(ctx, out, nil, false, run); err == nil {
		t.Fatal("missing staged DWARF was accepted")
	}
	for path, want := range map[string]string{out.Out: "linked executable", previous: "older bundle"} {
		if data, err := os.ReadFile(path); err != nil || string(data) != want {
			t.Fatalf("failed compression replaced %s: %q, %v", path, data, err)
		}
	}
	if files, err := os.ReadDir(root); err != nil || len(files) != 2 {
		t.Fatalf("failed compression leaked staging files: %v, %v", files, err)
	}
}

func TestDarwinStrippedNativeDWARFCompression(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native stripped dSYM integration")
	}
	bin := filepath.Join(t.TempDir(), "stripped-compressed-dwarf")
	conf := &Config{Mode: ModeBuild, OutFile: bin, LinkOptions: LinkOptions{OmitSymbolTable: true, DWARF: DWARFPreserve, DWARFCompression: DWARFCompressionEnabled}}
	for _, compression := range []DWARFCompressionMode{DWARFCompressionEnabled, DWARFCompressionDisabled} {
		conf.LinkOptions.DWARFCompression = compression
		if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
			t.Fatal(err)
		}
		if out := runBinary(t, bin); !strings.Contains(out, "main.caller main.go true") {
			t.Fatalf("stripped output lost runtime caller information: %s", out)
		}
		verifyPCLNIntegrationSignature(t, bin)
		if machoHasStabs(t, bin) || pclnIntegrationHasSymbolTable(t, bin) {
			t.Fatal("-s retained the debug map or ordinary symbols")
		}
		path := filepath.Join(nativeDSYMPath(bin), "Contents", "Resources", "DWARF", filepath.Base(bin))
		f, err := macho.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if (f.Section("__zdebug_info") != nil) != (compression == DWARFCompressionEnabled) {
			t.Fatalf("stripped dSYM compression does not match mode %d", compression)
		}
		data, err := f.DWARF()
		if err != nil {
			t.Fatal(err)
		}
		if findDWARFNode(readDWARFTree(t, data), dwarf.TagSubprogram, "main.caller") == nil {
			t.Fatal("stripped dSYM lost main.caller")
		}
		f.Close()
	}
}

func TestWindowsNativeDWARFCompression(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native PE runtime integration")
	}
	for _, mode := range []DWARFCompressionMode{DWARFCompressionDefault, DWARFCompressionEnabled, DWARFCompressionDisabled} {
		bin := filepath.Join(t.TempDir(), "compressed-dwarf.exe")
		conf := &Config{Mode: ModeBuild, OutFile: bin, LinkOptions: LinkOptions{DWARF: DWARFPreserve, DWARFCompression: mode}}
		if _, err := Do([]string{"./testdata/ldflagsstrip"}, conf); err != nil {
			t.Fatal(err)
		}
		if out := runBinary(t, bin); !strings.Contains(out, "main.caller main.go true") {
			t.Fatalf("runtime PCLN output: %s", out)
		}
		f, err := pe.Open(bin)
		if err != nil {
			t.Fatal(err)
		}
		if (f.Section(".zdebug_info") != nil) != (mode == DWARFCompressionEnabled) {
			t.Fatalf("compression did not match explicit mode %d", mode)
		}
		data, err := f.DWARF()
		if err != nil {
			t.Fatal(err)
		}
		if findDWARFNode(readDWARFTree(t, data), dwarf.TagSubprogram, "main.caller") == nil {
			t.Fatal("compressed PE lost main.caller")
		}
		f.Close()
	}
}
