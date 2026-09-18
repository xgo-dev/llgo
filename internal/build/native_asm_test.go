//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestForeignARM64Selection(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want bool
	}{
		{"TEXT raw<>(SB), NOSPLIT, $0-0\n JMP imported(SB)\n", true},
		{"TEXT raw<>(SB), NOSPLIT|NOFRAME, $0\n BL imported(SB)\n RET\n", true},
		{"TEXT ·declared(SB), NOSPLIT, $0-0\n RET\n", false},
		{"TEXT raw<>(SB), 0, $0-0\n RET\n", false},
		{"TEXT raw<>(SB), NOSPLIT, $8-0\n RET\n", false},
		{"TEXT raw<>(SB), NOSPLIT, $0-0\n BL imported(SB)\n RET\n", false},
		{"TEXT raw<>(SB), NOSPLIT, $0-0\n RET\nTEXT ·goFunc(SB),NOSPLIT,$0\nRET\n", false},
	} {
		if got := len(foreignARM64Functions([]byte(tc.src))) != 0; got != tc.want {
			t.Errorf("selection(%q) = %v", tc.src, got)
		}
	}
}

func TestForeignARM64Callback(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("native darwin/arm64 execution")
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/nativecallback\n\ngo 1.20\n",
		"main.go": `package main
import (_ "syscall"; "unsafe")
//go:cgo_import_dynamic imported_strlen strlen "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic imported_cf CFBooleanGetTypeID "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
var entry, cfEntry uintptr
//go:linkname call syscall.syscall6
func call(fn,a1,a2,a3,a4,a5,a6 uintptr)(r1,r2,err uintptr)
func main(){
 s:=[]byte("native ABI\x00")
 n,_,_:=call(entry,uintptr(unsafe.Pointer(&s[0])),0,0,0,0,0)
 if n!=10 {panic("native callback lost its argument or result")}
 id,_,_:=call(cfEntry,0,0,0,0,0,0)
 if id==0 {panic("missing framework function")}
 println("ok")
}
`,
		"callback.s": `#include "textflag.h"
TEXT callback<>(SB), NOSPLIT|NOFRAME, $0
 SUB $16, RSP
 MOVD R30, (RSP)
 BL imported_strlen(SB)
 MOVD (RSP), R30
 ADD $16, RSP
 RET
GLOBL ·entry(SB), RODATA, $8
DATA ·entry(SB)/8, $callback<>(SB)
TEXT cftramp<>(SB), NOSPLIT, $0-0
 JMP imported_cf(SB)
GLOBL ·cfEntry(SB), RODATA, $8
DATA ·cfEntry(SB)/8, $cftramp<>(SB)
`,
	}
	for name, s := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "probe")
	if _, err := Do([]string{"."}, conf); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(conf.OutFile).CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("run: %v\n%s", err, out)
	}
}
