package plan9asm

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func nativeObjectFixture(t *testing.T) []byte {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "native.s")
	out := filepath.Join(dir, "native.o")
	code := `#include "textflag.h"
TEXT callback<>(SB), NOSPLIT|NOFRAME, $0
 SUB $16, RSP
 MOVD R30, (RSP)
 BL imported_strlen(SB)
 MOVD (RSP), R30
 ADD $16, RSP
 RET
GLOBL ·entry(SB), RODATA, $8
DATA ·entry(SB)/8, $callback<>(SB)
TEXT mixedtramp<>(SB), NOSPLIT, $0-0
 FMOVD R1, F0
 JMP imported_mixed(SB)
GLOBL ·mixedEntry(SB), RODATA, $8
DATA ·mixedEntry(SB)/8, $mixedtramp<>(SB)
`
	if err := os.WriteFile(src, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "tool", "asm", "-p", "probe", "-I", filepath.Join(runtime.GOROOT(), "pkg", "include"), "-o", out, src)
	cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=arm64")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("asm: %v\n%s", err, b)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNativeARM64Object(t *testing.T) {
	obj := nativeObjectFixture(t)
	funcs := map[string]bool{"callback": true, "mixedtramp": true}
	imports := map[string]string{"imported_strlen": "strlen", "imported_mixed": "mixed"}
	asm, data, err := NativeARM64Object(obj, funcs, imports, "probe")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 || data[0].Name != "probe.entry" || data[0].Size != 8 {
		t.Fatalf("data = %v", data)
	}
	if !strings.Contains(asm, `bl "_strlen"`) || !strings.Contains(asm, `.quad "Lllgo_native_`) {
		t.Fatalf("missing relocations:\n%s", asm)
	}
	if _, _, err := NativeARM64Object(obj, funcs, nil, "probe"); err == nil {
		t.Fatal("accepted undeclared foreign call")
	}
	if _, _, err := NativeARM64Object(obj, map[string]bool{"missing": true}, imports, "probe"); err == nil {
		t.Fatal("accepted missing function")
	}
	for n := 0; n < len(obj); n++ {
		if _, _, err := NativeARM64Object(obj[:n], funcs, imports, "probe"); err == nil {
			t.Fatalf("accepted truncated object at %d", n)
		}
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return
	}
	dir := t.TempDir()
	s := filepath.Join(dir, "native.s")
	c := filepath.Join(dir, "main.c")
	exe := filepath.Join(dir, "probe")
	os.WriteFile(s, []byte(asm), 0600)
	os.WriteFile(c, []byte(`extern void *entry __asm("_probe.entry");
extern void *mixedEntry __asm("_probe.mixedEntry");
unsigned long mixed(unsigned long x, double y) { return x + (unsigned long)y; }
int main(void) {
 if (((unsigned long (*)(const char *))entry)("native ABI") != 10) return 1;
 return ((unsigned long (*)(unsigned long, unsigned long))mixedEntry)(5, 0x4000000000000000UL) != 7;
}
`), 0600)
	cmd := exec.Command("clang", s, c, "-o", exe)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("link: %v\n%s", err, b)
	}
	if b, err := exec.Command(exe).CombinedOutput(); err != nil {
		t.Fatalf("callback: %v\n%s", err, b)
	}
}
