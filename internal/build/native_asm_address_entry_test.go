package build

import (
	"debug/elf"
	"debug/macho"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/abi"
	llvm "github.com/xgo-dev/llvm"
)

const nativeAddressFixturePath = "example.com/plan9asm-native-address"

func nativeAddressFixtureFiles(t *testing.T) map[string]string {
	t.Helper()
	files := make(map[string]string)
	for _, name := range []string{"go.mod", "main.go", "entry.s"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "plan9asm-native-address", name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(contents)
	}
	return files
}

func nativeAddressFixtureWrite(t *testing.T, files map[string]string) string {
	t.Helper()
	directory := t.TempDir()
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

// This uses the real selected SDK for preprocessing, not a synthetic VERSION
// file or an assumed Go signature for the bare rawEntry symbol.
func nativeAddressFixturePackage(t *testing.T, files map[string]string, goos, arch, triple string) (*context, *aPackage) {
	t.Helper()
	directory := nativeAddressFixtureWrite(t, files)
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, filepath.Join(directory, "main.go"), nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	sizes := types.SizesFor("gc", arch)
	config := types.Config{Sizes: sizes, Importer: importer.Default()}
	typed, err := config.Check(nativeAddressFixturePath, fset, []*ast.File{syntax}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assembly := filepath.Join(directory, "entry.s")
	pkg := &packages.Package{
		ID: typed.Path(), PkgPath: typed.Path(), Name: typed.Name(), Dir: directory,
		Types: typed, TypesSizes: sizes, Fset: fset, Syntax: []*ast.File{syntax},
		GoFiles: []string{filepath.Join(directory, "main.go")}, OtherFiles: []string{assembly},
	}
	prog := llssa.NewProgram(&llssa.Target{GOOS: goos, GOARCH: arch})
	t.Cleanup(prog.Dispose)
	apkg := &aPackage{Package: pkg, LPkg: prog.NewPackage(typed.Name(), typed.Path())}
	apkg.ExportFile = filepath.Join(directory, "fixture.a")
	if variable, ok := typed.Scope().Lookup("entry").(*types.Var); ok && variable.Type() == types.Typ[types.Uintptr] {
		mod := apkg.LPkg.Module()
		global := llvm.AddGlobal(mod, mod.Context().Int64Type(), abi.PathOf(typed)+".entry")
		global.SetInitializer(llvm.ConstNull(global.GlobalValueType()))
	}
	ctx := &context{
		mode: ModeBuild, prog: prog, plan9asmReady: true, plan9asmMode: plan9asmEnvAll,
		conf:         &packages.Config{},
		buildConf:    &Config{Goos: goos, Goarch: arch, GenLL: true},
		commands:     commandEnv{dir: directory, environ: withEnv(os.Environ(), "GOROOT="+runtime.GOROOT(), "GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off")},
		crossCompile: crosscompile.Export{CC: "clang", CCFLAGS: []string{"--target=" + triple}},
		pkgs:         map[*packages.Package]Package{pkg: {Package: pkg}},
	}
	return ctx, apkg
}

// No C signature is inferred. DATA must keep the native carrier alive as an
// address while the actual Go global is supplied by the assembly member.
func TestPlan9AsmAddressOnlyNativeEntryObjects(t *testing.T) {
	version, err := abi0FixtureCommand(t, 10*time.Second, "clang", "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(version), "clang version 22.") {
		t.Fatalf("LLVM22 clang is required: %v\n%s", err, version)
	}
	for _, target := range []struct{ goos, arch, triple string }{
		{"linux", "amd64", "x86_64-unknown-linux-gnu"},
		{"linux", "arm64", "aarch64-unknown-linux-gnu"},
		{"darwin", "amd64", "x86_64-apple-darwin"},
		{"darwin", "arm64", "arm64-apple-darwin"},
	} {
		t.Run(target.goos+"/"+target.arch, func(t *testing.T) {
			ctx, apkg := nativeAddressFixturePackage(t, nativeAddressFixtureFiles(t), target.goos, target.arch, target.triple)
			signatures, err := plan9asmSigsForPkg(ctx, apkg.PkgPath)
			if err != nil || len(signatures) != 0 {
				t.Fatalf("DATA-only bare entry must not acquire a guessed Go signature: signatures=%v err=%v", signatures, err)
			}
			objects, err := compilePkgSFiles(ctx, apkg, apkg.Package, false)
			for _, object := range objects {
				t.Cleanup(func() { os.Remove(object) })
			}
			if err != nil || len(objects) != 1 {
				t.Fatalf("native address carrier object: objects=%v err=%v", objects, err)
			}
			mod := apkg.LPkg.Module()
			global := mod.NamedGlobal(abi.PathOf(apkg.Types) + ".entry")
			if global.IsNil() || !global.IsDeclaration() {
				t.Fatal("Go global did not become a reference to the assembly DATA owner")
			}
			if strings.Contains(mod.String(), "asm sideeffect") {
				t.Fatal("address-only native code was merged into the typed Go module")
			}
			ir, err := os.ReadFile(apkg.ExportFile + "entry.s.ll")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(ir), "naked noinline") || !strings.Contains(string(ir), "asm sideeffect") {
				t.Fatalf("missing address-only native carrier:\n%s", ir)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if target.goos == "linux" {
				object, err := elf.Open(objects[0])
				if err != nil {
					t.Fatal(err)
				}
				defer object.Close()
				machine := elf.EM_X86_64
				if target.arch == "arm64" {
					machine = elf.EM_AARCH64
				}
				if object.Type != elf.ET_REL || object.Machine != machine {
					t.Fatal(object.FileHeader)
				}
			} else {
				object, err := macho.Open(objects[0])
				if err != nil {
					t.Fatal(err)
				}
				defer object.Close()
				cpu := macho.CpuAmd64
				if target.arch == "arm64" {
					cpu = macho.CpuArm64
				}
				if object.Type != macho.TypeObj || object.Cpu != cpu {
					t.Fatal(object.FileHeader)
				}
			}
		})
	}
}

func TestPlan9AsmAddressOnlyNativeEntryRejectsUnprovedSources(t *testing.T) {
	base := nativeAddressFixtureFiles(t)
	for _, row := range []struct{ name, goSource, assembly string }{
		{"explicit Go initializer", strings.Replace(base["main.go"], "var entry uintptr", "var entry uintptr = 0", 1), base["entry.s"]},
		{"dynamic Go initializer", strings.Replace(base["main.go"], "var entry uintptr", "var entry uintptr = makeEntry()\nfunc makeEntry() uintptr { return 7 }", 1), base["entry.s"]},
		{"no Go DATA owner", strings.Replace(base["main.go"], "var entry uintptr", "const entry = 1", 1), base["entry.s"]},
		{"pointer is not uintptr", strings.Replace(base["main.go"], "var entry uintptr", "var entry *byte", 1), base["entry.s"]},
		{"no code relocation", base["main.go"], strings.Replace(base["entry.s"], "$rawEntry(SB)", "$7", 1)},
		{"interior PC", base["main.go"], strings.Replace(base["entry.s"], "$rawEntry(SB)", "$rawEntry+1(SB)", 1)},
		{"nonzero frame", base["main.go"], strings.Replace(strings.Replace(base["entry.s"], "NOSPLIT|NOFRAME", "NOSPLIT", 1), "$0-0", "$16-0", 1)},
		{"Go argument frame", base["main.go"], strings.Replace(base["entry.s"], "$0-0", "$0-8", 1)},
		{"FP access", base["main.go"], strings.Replace(base["entry.s"], "\tRET", "\tMOVQ argument+0(FP), AX\n\tRET", 1)},
		{"aliased DATA owner", strings.Replace(base["main.go"], "package main\n", "package main\nimport _ \"unsafe\"\n//go:linkname entry example.com/other.entry\n", 1), base["entry.s"]},
		{"typed CALL does not supply a native ABI", base["main.go"] + "\nfunc Invoke()\n", base["entry.s"] + "\nTEXT ·Invoke(SB), NOSPLIT|NOFRAME, $0-0\nCALL rawEntry(SB)\nRET\n"},
		{"unselected cross-file relocation", base["main.go"], "#include \"textflag.h\"\nTEXT rawEntry(SB), NOSPLIT|NOFRAME, $0-0\nRET\n"},
		{"unowned sibling", base["main.go"], base["entry.s"] + "\nTEXT unknownEntry(SB), NOSPLIT|NOFRAME, $0-0\nRET\n"},
	} {
		t.Run(row.name, func(t *testing.T) {
			files := map[string]string{"go.mod": base["go.mod"], "main.go": row.goSource, "entry.s": row.assembly}
			if row.name == "pointer is not uintptr" {
				files["main.go"] = strings.Replace(files["main.go"], "entry == 0", "entry == nil", 1)
			}
			if row.name == "unselected cross-file relocation" {
				// This exists on disk but is deliberately absent from loaded
				// OtherFiles. A directory glob cannot grant source authority.
				files["unselected.s"] = "#include \"textflag.h\"\nGLOBL ·entry(SB), RODATA|NOPTR, $8\nDATA ·entry(SB)/8, $rawEntry(SB)\n"
			}
			ctx, apkg := nativeAddressFixturePackage(t, files, "linux", "amd64", "x86_64-unknown-linux-gnu")
			if signatures, err := plan9asmSigsForPkg(ctx, apkg.PkgPath); err == nil || len(signatures) != 0 {
				t.Fatalf("unproved source bypassed declaration checks: signatures=%v err=%v", signatures, err)
			}
		})
	}
}

func TestPlan9AsmAddressOnlyNativeEntryKeepsGoDeclaration(t *testing.T) {
	files := nativeAddressFixtureFiles(t)
	files["main.go"] += "\nfunc rawEntry()\n"
	files["entry.s"] = strings.ReplaceAll(files["entry.s"], "rawEntry(SB)", "·rawEntry(SB)")
	ctx, apkg := nativeAddressFixturePackage(t, files, "linux", "amd64", "x86_64-unknown-linux-gnu")
	if _, native, err := foreignNativeImports(ctx, apkg.Package, []byte(files["entry.s"])); err != nil || native {
		t.Fatal("a real Go declaration acquired a native-entry role", native, err)
	}
	signatures, err := plan9asmSigsForPkg(ctx, apkg.PkgPath)
	if err != nil {
		t.Fatal(err)
	}
	name := abi.PathOf(apkg.Types) + ".rawEntry"
	if _, present := signatures[name]; !present || len(signatures) != 1 {
		t.Fatalf("real Go declaration disappeared or was replaced with a guessed signature: %v", signatures)
	}
}
