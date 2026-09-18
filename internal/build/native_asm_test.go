//go:build !llgo

package build

import (
	"debug/macho"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	llvm "github.com/xgo-dev/llvm"
)

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
	t.Run("reject mismatched DATA", func(t *testing.T) {
		bad := strings.ReplaceAll(files["callback.s"], "GLOBL ·entry(SB), RODATA, $8", "GLOBL ·entry(SB), RODATA, $16")
		if err := os.WriteFile(filepath.Join(dir, "callback.s"), []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Do([]string{"."}, conf); err == nil || !strings.Contains(err.Error(), "Go size 8 but DATA size 16") {
			t.Fatalf("mismatched DATA: %v", err)
		}
	})
}

func TestForeignARM64SelectionAndErrors(t *testing.T) {
	const valid = "#include \"textflag.h\"\nTEXT callback<>(SB), NOSPLIT, $0\n JMP imported(SB)\n"
	const imports = "//go:cgo_import_dynamic imported strlen\n"
	cases := []struct {
		name, goos, decl, asm, want string
		handled                     bool
		badTemp                     bool
	}{
		{name: "other target", goos: "linux"},
		{name: "no imports", goos: "darwin", asm: valid},
		{name: "Go ABI", goos: "darwin", decl: imports, asm: "TEXT ·f(SB), NOSPLIT, $0\nRET\n"},
		{name: "conflicting import", goos: "darwin", decl: imports + "//go:cgo_import_dynamic imported other\n", asm: valid, want: "conflicting dynamic import", handled: true},
		{name: "invalid instruction", goos: "darwin", decl: imports, asm: valid + "NOT_AN_INSTRUCTION\n", want: "assemble foreign ABI code", handled: true},
		{name: "undeclared import", goos: "darwin", decl: imports, asm: strings.ReplaceAll(valid, "JMP imported", "JMP undeclared"), want: "undeclared foreign symbol", handled: true},
		{name: "compiler failure", goos: "darwin", decl: imports, asm: valid, want: "missing-clang", handled: true},
		{name: "temporary directory failure", goos: "darwin", decl: imports, asm: valid, want: "missing", handled: true, badTemp: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.badTemp {
				t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
				t.Setenv("TMP", os.Getenv("TMPDIR"))
				t.Setenv("TEMP", os.Getenv("TMPDIR"))
			}
			file, err := parser.ParseFile(token.NewFileSet(), "imports.go", "package p\n"+tc.decl, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			ctx := &context{buildConf: &Config{Goos: tc.goos, Goarch: "arm64"}, commands: commandEnv{environ: os.Environ()}}
			ctx.crossCompile = crosscompile.Export{CC: filepath.Join(t.TempDir(), "missing-clang")}
			pkg := &packages.Package{PkgPath: "probe", Types: types.NewPackage("probe", "p"), Syntax: []*ast.File{file}}
			_, handled, err := compileForeignARM64Asm(ctx, nil, pkg, "callback.s", []byte(tc.asm))
			if handled != tc.handled {
				t.Fatalf("handled=%v, want %v", handled, tc.handled)
			}
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}

// Object generation and DATA binding need no Darwin SDK or execution host.
// Exercise the real assembly driver on every CI host; only TestForeignARM64Callback
// needs native darwin/arm64 execution.
func TestForeignARM64CrossCompileObject(t *testing.T) {
	for _, size := range []int{8, 16} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			dir := t.TempDir()
			src := fmt.Sprintf(`#include "textflag.h"
TEXT callback<>(SB), NOSPLIT, $0
 JMP imported(SB)
GLOBL ·entry(SB), RODATA, $%d
DATA ·entry(SB)/8, $callback<>(SB)
`, size)
			sfile := filepath.Join(dir, "callback.s")
			if err := os.WriteFile(sfile, []byte(src), 0600); err != nil {
				t.Fatal(err)
			}
			file, err := parser.ParseFile(token.NewFileSet(), "imports.go", "package p\n//go:cgo_import_dynamic imported strlen\n", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			prog := llssa.NewProgram(&llssa.Target{GOOS: "darwin", GOARCH: "arm64"})
			defer prog.Dispose()
			pkg := &packages.Package{ID: "probe", PkgPath: "probe", Dir: dir, Types: types.NewPackage("probe", "p"), Syntax: []*ast.File{file}, OtherFiles: []string{sfile}}
			apkg := &aPackage{Package: pkg, LPkg: prog.NewPackage("p", "probe")}
			mod := apkg.LPkg.Module()
			global := llvm.AddGlobal(mod, mod.Context().Int64Type(), "probe.entry")
			global.SetInitializer(llvm.ConstNull(global.GlobalValueType()))
			ctx := &context{prog: prog, buildConf: &Config{Goos: "darwin", Goarch: "arm64"}, commands: commandEnv{environ: os.Environ()}, crossCompile: crosscompile.Export{CC: "clang", CCFLAGS: []string{"--target=arm64-apple-darwin"}}, plan9asmReady: true, plan9asmMode: plan9asmEnvAll}
			objects, err := compilePkgSFiles(ctx, apkg, pkg, false)
			for _, object := range objects {
				defer os.Remove(object)
			}
			if size != 8 {
				if err == nil || !strings.Contains(err.Error(), "Go size 8 but DATA size 16") {
					t.Fatalf("invalid DATA: %v", err)
				}
				if global.IsDeclaration() {
					t.Fatal("invalid DATA changed Go global")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(objects) != 1 {
				t.Fatalf("objects=%v", objects)
			}
			if !global.IsDeclaration() || global.Linkage() != llvm.ExternalLinkage {
				t.Fatal("Go global did not bind to native DATA")
			}
			obj, err := macho.Open(objects[0])
			if err != nil {
				t.Fatal(err)
			}
			defer obj.Close()
			if obj.Cpu != macho.CpuArm64 || obj.Type != macho.TypeObj {
				t.Fatalf("unexpected object header: %+v", obj.FileHeader)
			}
			found := false
			for _, sym := range obj.Symtab.Syms {
				// debug/macho strips the leading underscore from Go symbol names.
				if sym.Name == "probe.entry" {
					found = true
				}
			}
			if !found {
				t.Fatal("native object missing DATA symbol")
			}
		})
	}
}
