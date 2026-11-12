package build

import (
	"debug/elf"
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
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
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
	for _, mode := range []lto.Mode{lto.Off, lto.Thin, lto.Full} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel = optlevel.O2
			conf.LTO = mode
			conf.OutFile = filepath.Join(dir, "probe-"+fmt.Sprint(mode))
			if _, err := Do([]string{"."}, conf); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(conf.OutFile).CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "ok" {
				t.Fatalf("run: %v\n%s", err, out)
			}
		})
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "invalid")
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
		{name: "other target", goos: "windows"},
		{name: "no imports", goos: "darwin", asm: valid},
		{name: "Go ABI", goos: "darwin", decl: imports, asm: "TEXT ·f(SB), NOSPLIT, $0\nRET\n"},
		{name: "conflicting import", goos: "darwin", decl: imports + "//go:cgo_import_dynamic imported other\n", asm: valid, want: "conflicting dynamic import", handled: true},
		{name: "invalid instruction", goos: "darwin", decl: imports, asm: valid + "NOT_AN_INSTRUCTION\n", want: "unsupported native instruction", handled: true},
		{name: "undeclared import", goos: "darwin", decl: imports, asm: strings.ReplaceAll(valid, "JMP imported", "JMP undeclared"), want: "undeclared foreign symbol", handled: true},
		{name: "no native compiler needed", goos: "darwin", decl: imports, asm: valid, handled: true},
		{name: "no temporary files needed", goos: "darwin", decl: imports, asm: valid, handled: true, badTemp: true},
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
			prog := llssa.NewProgram(&llssa.Target{GOOS: "darwin", GOARCH: "arm64"})
			defer prog.Dispose()
			ctx := &context{prog: prog, buildConf: &Config{Goos: tc.goos, Goarch: "arm64"}, commands: commandEnv{environ: os.Environ()}}
			ctx.crossCompile = crosscompile.Export{CC: filepath.Join(t.TempDir(), "missing-clang")}
			pkg := &packages.Package{PkgPath: "probe", Types: types.NewPackage("probe", "p"), Syntax: []*ast.File{file}}
			apkg := &aPackage{Package: pkg, LPkg: prog.NewPackage("p", "probe")}
			mod, handled, err := translateForeignNativeAsm(ctx, apkg, pkg, []byte(tc.asm))
			if !mod.IsNil() {
				defer mod.Dispose()
			}
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

// Native assembly shares the ordinary Plan 9 .ll output and object compiler.
// Check saved IR, DATA binding and target relocations on every LLVM host.
func TestForeignNativeCrossCompileObject(t *testing.T) {
	for _, target := range []struct{ goos, goarch, triple string }{{"darwin", "amd64", "x86_64-apple-darwin"}, {"darwin", "arm64", "arm64-apple-darwin"}, {"linux", "amd64", "x86_64-unknown-linux-gnu"}, {"linux", "arm64", "aarch64-unknown-linux-gnu"}} {
		t.Run(target.goos+"/"+target.goarch, func(t *testing.T) {
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
					if err := os.WriteFile(sfile, []byte("#include \"implementation.h\"\n"), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "implementation.h"), []byte(src), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "textflag.h"), []byte("#define NOSPLIT 4\n#define RODATA 8\n"), 0600); err != nil {
						t.Fatal(err)
					}
					file, err := parser.ParseFile(token.NewFileSet(), "imports.go", "package p\n//go:cgo_import_dynamic imported strlen\n", parser.ParseComments)
					if err != nil {
						t.Fatal(err)
					}
					prog := llssa.NewProgram(&llssa.Target{GOOS: target.goos, GOARCH: target.goarch})
					defer prog.Dispose()
					pkg := &packages.Package{ID: "probe", PkgPath: "probe", Dir: dir, Types: types.NewPackage("probe", "p"), Syntax: []*ast.File{file}, OtherFiles: []string{sfile}}
					apkg := &aPackage{Package: pkg, LPkg: prog.NewPackage("p", "probe")}
					apkg.ExportFile = filepath.Join(dir, "probe.a")
					mod := apkg.LPkg.Module()
					global := llvm.AddGlobal(mod, mod.Context().Int64Type(), "probe.entry")
					global.SetInitializer(llvm.ConstNull(global.GlobalValueType()))
					ctx := &context{prog: prog, buildConf: &Config{Goos: target.goos, Goarch: target.goarch, GenLL: true}, commands: commandEnv{environ: os.Environ()}, crossCompile: crosscompile.Export{CC: "clang", CCFLAGS: []string{"--target=" + target.triple}}, plan9asmReady: true, plan9asmMode: plan9asmEnvAll}
					// No Go assembler or Go object reader is needed. The selected
					// source SDK needs only its version here: headers are local.
					goroot := filepath.Join(dir, "source-sdk")
					if err := os.Mkdir(goroot, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(goroot, "VERSION"), []byte("go1.27.2\n"), 0600); err != nil {
						t.Fatal(err)
					}
					ctx.commands.environ = withEnv(ctx.commands.environ, "GOROOT="+goroot)
					ctx.pkgs = map[*packages.Package]Package{pkg: {Package: pkg}}
					if sigs, err := plan9asmSigsForPkg(ctx, pkg.PkgPath); err != nil || len(sigs) != 0 {
						t.Fatalf("native entries acquired Go signatures: %v, %v", sigs, err)
					}
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
					if !global.IsDeclaration() {
						t.Fatal("Go global must refer to the separate native DATA definition")
					}
					if strings.Contains(mod.String(), "asm sideeffect") {
						t.Fatal("native carrier was merged into the Go module")
					}
					ll, err := os.ReadFile(apkg.ExportFile + filepath.Base(sfile) + ".ll")
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(ll), "naked noinline") || !strings.Contains(string(ll), "asm sideeffect") || strings.Contains(string(ll), "module asm") {
						t.Fatalf("missing naked function carrier in saved IR:\n%s", ll)
					}
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
					object := objects[0]
					if target.goos == "linux" {
						obj, err := elf.Open(object)
						if err != nil {
							t.Fatal(err)
						}
						defer obj.Close()
						machine := elf.EM_X86_64
						if target.goarch == "arm64" {
							machine = elf.EM_AARCH64
						}
						if obj.Machine != machine || obj.Type != elf.ET_REL {
							t.Fatal(obj.FileHeader)
						}
						syms, err := obj.Symbols()
						if err != nil {
							t.Fatal(err)
						}
						found := false
						for _, sym := range syms {
							if sym.Name == "probe.entry" {
								found = true
							}
						}
						if !found {
							t.Fatal("native ELF missing DATA symbol")
						}
						return
					}
					obj, err := macho.Open(object)
					if err != nil {
						t.Fatal(err)
					}
					defer obj.Close()
					cpu := macho.CpuArm64
					if target.goarch == "amd64" {
						cpu = macho.CpuAmd64
					}
					if obj.Cpu != cpu || obj.Type != macho.TypeObj {
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

		})
	}
}

// Run on each native backend pair: linux/{amd64,arm64} and darwin/{amd64,arm64}.
// The library directive must be sufficient to link; no cgo package supplies -l.
func TestForeignNativeSharedLibrary(t *testing.T) {
	if (runtime.GOOS != "linux" && runtime.GOOS != "darwin") || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("native ELF/Mach-O backend")
	}
	dir := t.TempDir()
	libDir := filepath.Join(dir, "library with spaces")
	appDir := filepath.Join(dir, "app")
	for _, path := range []string{libDir, appDir} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	lib := filepath.Join(libDir, "libprobe.so.1")
	args := []string{"-shared", "-fPIC"}
	if runtime.GOOS == "darwin" {
		lib = filepath.Join(libDir, "libprobe.dylib")
		args = []string{"-dynamiclib", "-fPIC", "-Wl,-install_name," + lib}
	} else {
		args = append(args, "-Wl,-soname,"+filepath.Base(lib))
	}
	cfile := filepath.Join(libDir, "probe.c")
	if err := os.WriteFile(cfile, []byte(`long long answer(long long x) { return x + 7; }
long long invoke(long long (*fn)(long long), long long x) { return fn(x); }
`), 0600); err != nil {
		t.Fatal(err)
	}
	args = append(args, cfile, "-o", lib)
	if out, err := exec.Command("clang", args...).CombinedOutput(); err != nil {
		t.Fatalf("shared library: %v\n%s", err, out)
	}
	files := map[string]string{
		"go.mod": "module example.com/native-shared-library\n\ngo 1.20\n",
		"bridge.s": `#include "textflag.h"
TEXT bridge<>(SB), NOSPLIT, $0
 JMP imported(SB)
GLOBL ·entry(SB), RODATA, $8
DATA ·entry(SB)/8, $bridge<>(SB)
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(appDir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(appDir)
	t.Setenv("LIBRARY_PATH", libDir)
	libraries := []string{lib}
	if runtime.GOOS == "linux" {
		libraries = append(libraries, filepath.Base(lib))
	}
	for _, library := range libraries {
		t.Run(library, func(t *testing.T) {
			source := fmt.Sprintf(`package main
import _ "unsafe"
//go:cgo_import_dynamic imported answer %q
var entry uintptr
//go:linkname invoke C.invoke
func invoke(fn uintptr, x int64) int64
func main() {
 if invoke(entry, 35) != 42 || invoke(entry, -9) != -2 { panic("native argument/result") }
 println("ok")
}
`, library)
			if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []lto.Mode{lto.Off, lto.Thin, lto.Full} {
				t.Run(fmt.Sprint(mode), func(t *testing.T) {
					conf := NewDefaultConf(ModeBuild)
					conf.OptLevel, conf.LTO = optlevel.O2, mode
					conf.OutFile = filepath.Join(dir, "probe-"+fmt.Sprint(mode))
					if _, err := Do([]string{"."}, conf); err != nil {
						t.Fatal(err)
					}
					cmd := exec.Command(conf.OutFile)
					cmd.Env = withEnv(os.Environ(), "LD_LIBRARY_PATH="+libDir)
					if out, err := cmd.CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "ok" {
						t.Fatalf("run: %v\n%s", err, out)
					}
				})
			}
		})
	}
}
