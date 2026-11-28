// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package dwarfcompress

import (
	"bytes"
	"debug/macho"
	"debug/pe"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func nativeTool(t *testing.T, name string) string {
	t.Helper()
	for _, dir := range []string{os.Getenv("LLGO_TEST_LLD_DIR"), filepath.Dir(os.Getenv("LLVM_CONFIG"))} {
		if dir != "" && dir != "." {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	t.Skipf("%s is not installed", name)
	return ""
}

func runTool(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %q: %v\n%s", name, args, err, out)
	}
}

func TestNativeToolchainArtifacts(t *testing.T) {
	clang := nativeTool(t, "clang")
	for _, target := range []struct{ name, triple, machine string }{
		{"PE-amd64", "x86_64-pc-windows-msvc", "x64"},
		{"PE-386", "i686-pc-windows-msvc", "x86"},
		{"PE-arm64", "aarch64-pc-windows-msvc", "arm64"},
		{"MachO-amd64", "x86_64-apple-macos11", "x86_64"},
		{"MachO-arm64", "arm64-apple-macos11", "arm64"},
	} {
		t.Run(target.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "probe.c")
			var text strings.Builder
			for i := range 3000 {
				fmt.Fprintf(&text, "struct unique_type_%d { int x; long y; }; struct unique_type_%d global_%d;\n", i, i, i)
			}
			text.WriteString("int main(void) { return global_0.x; }\n")
			if err := os.WriteFile(source, []byte(text.String()), 0o600); err != nil {
				t.Fatal(err)
			}
			object, image := filepath.Join(dir, "probe.o"), filepath.Join(dir, "probe")
			runTool(t, clang, "-target", target.triple, "-gdwarf-4", "-fno-stack-protector", "-c", source, "-o", object)
			isPE := strings.HasPrefix(target.name, "PE")
			if isPE {
				runTool(t, nativeTool(t, "lld-link"), "/entry:main", "/machine:"+target.machine, "/subsystem:console", "/nodefaultlib", "/debug:dwarf", "/opt:noref", "/out:"+image, object)
			} else {
				runTool(t, nativeTool(t, "ld64.lld"), "-arch", target.machine, "-platform_version", "macos", "11.0", "11.0", "-e", "_main", "-o", image, object)
				dsym := filepath.Join(dir, "probe.dwarf")
				runTool(t, nativeTool(t, "dsymutil"), "-f", "-o", dsym, image)
				image = dsym
			}
			raw, err := os.ReadFile(image)
			if err != nil {
				t.Fatal(err)
			}
			out, err := Compress(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(out) >= len(raw) {
				t.Fatalf("native artifact did not shrink: %d -> %d", len(raw), len(out))
			}
			t.Logf("%d -> %d bytes", len(raw), len(out))
			if isPE {
				a, err := pe.NewFile(bytes.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				b, err := pe.NewFile(bytes.NewReader(out))
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(dwarfEntries(t, a), dwarfEntries(t, b)) {
					t.Fatal("native PE DWARF changed")
				}
				if runtime.GOOS == "windows" && (target.name == "PE-"+runtime.GOARCH || runtime.GOARCH == "amd64" && target.name == "PE-386") {
					for name, data := range map[string][]byte{"original": raw, "compressed": out} {
						executable := filepath.Join(dir, name+".exe")
						if err := os.WriteFile(executable, data, 0o755); err != nil {
							t.Fatal(err)
						}
						runTool(t, executable)
					}
				}
			} else {
				a, err := macho.NewFile(bytes.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				b, err := macho.NewFile(bytes.NewReader(out))
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(dwarfEntries(t, a), dwarfEntries(t, b)) {
					t.Fatal("native Mach-O DWARF changed")
				}
			}
		})
	}
}
