package build

import (
	stdcontext "context"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestPackageAssemblyNativeObjectsCompileLinkAndRun(t *testing.T) {
	for _, extension := range []string{".s", ".S"} {
		for _, genLL := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/GenLL=%t", extension, genLL), func(t *testing.T) {
				ctx, pkg, source, _ := nativeAssemblyTestFixture(t, extension)
				ctx.buildConf.GenLL = genLL
				nativeAssemblyTestCompileAndRun(t, ctx, pkg, source)
			})
		}
	}
}

func TestPackageAssemblyNativeIncludesFromOtherInvocationDir(t *testing.T) {
	for _, extension := range []string{".s", ".S"} {
		for _, flags := range []string{`-I"include space"`, `-I"${SRCDIR}/include space"`} {
			t.Run(extension+"/"+flags, func(t *testing.T) {
				ctx, pkg, source, _ := nativeAssemblyTestFixture(t, extension)
				ctx.commands.dir = t.TempDir()
				original := "package fixture\n/*\n#cgo CFLAGS: -DOWNER_FLAG=1 " + flags + "\n*/\nimport \"C\"\n"
				if err := os.WriteFile(pkg.GoFiles[0], []byte(original), 0o600); err != nil {
					t.Fatal(err)
				}
				nativeAssemblyTestCompileAndRun(t, ctx, pkg, source)
			})
		}
	}
}

func nativeAssemblyTestCompileAndRun(t *testing.T, ctx *context, pkg *packages.Package, source string) {
	t.Helper()
	objects, err := concatPkgLinkFiles(ctx, pkg, false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeFiles(objects)
	if len(objects) != 1 {
		t.Fatalf("native assembly produced %d objects, want one", len(objects))
	}
	main := assemblyLanguageTestWrite(t, pkg.Dir, "main.c", "extern int answer(void);\nint main(void) { return answer() != 42; }\n")
	executable := filepath.Join(pkg.Dir, "probe")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if err := ctx.compiler().Compile(objects[0], main, "-o", executable); err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	command := ctx.commands.configure(exec.CommandContext(runCtx, executable))
	command.WaitDelay = 2 * time.Second
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("actual native object execution: %v\n%s", err, output)
	}
	// Native assembly must not publish an empty LLVM IR sidecar, including
	// when GenLL requests debug output for the package's other source files.
	if _, err := os.Stat(pkg.ExportFile + filepath.Base(source) + ".ll"); !os.IsNotExist(err) {
		t.Fatalf("native assembly published an invalid LLVM IR sidecar: %v", err)
	}
}

func TestPackageAssemblyNativeHeaderChangesObjectFingerprint(t *testing.T) {
	for _, extension := range []string{".s", ".S"} {
		t.Run(extension, func(t *testing.T) {
			ctx, pkg, _, header := nativeAssemblyTestFixture(t, extension)
			fingerprint := func() (string, llgoFileDigest) {
				t.Helper()
				// The existing per-context hash cache is intentionally not an
				// independent observation of a later filesystem state.
				fresh := &context{
					mode: ctx.mode, conf: ctx.conf, buildConf: ctx.buildConf,
					commands: ctx.commands, crossCompile: ctx.crossCompile, pkgs: ctx.pkgs,
					plan9asmReady: ctx.plan9asmReady, plan9asmMode: ctx.plan9asmMode,
				}
				manifest := newManifestBuilder()
				if err := fresh.collectPackageInputs(manifest, &aPackage{Package: pkg}); err != nil {
					t.Fatal(err)
				}
				if len(manifest.pkg.Plan9AsmFiles) != 0 || len(manifest.pkg.LLGoFiles) != 1 {
					t.Fatalf("native assembly lacks its actual compiler fingerprint: %+v", manifest.pkg)
				}
				digest := manifest.pkg.LLGoFiles[0]
				if digest.ObjectHash == "" || digest.PreprocessedHash != "" {
					t.Fatalf("object bytes were mislabeled as preprocessor output: %+v", digest)
				}
				return manifest.Fingerprint(), digest
			}
			before, beforeFile := fingerprint()
			if repeated, _ := fingerprint(); repeated != before {
				t.Fatal("unchanged native source produced an unstable object fingerprint")
			}
			info, err := os.Stat(header)
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(header)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.Replace(string(body), "42", "43", 1)
			if changed == string(body) || len(changed) != len(body) {
				t.Fatal("header mutation did not preserve the intended size")
			}
			if err := os.WriteFile(header, []byte(changed), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(header, info.ModTime(), info.ModTime()); err != nil {
				t.Fatal(err)
			}
			after, afterFile := fingerprint()
			if before == after || beforeFile.ObjectHash == afterFile.ObjectHash {
				t.Fatal("same-size/mtime assembly include change did not invalidate the object fingerprint")
			}
		})
	}
}

func TestPackageAssemblyExplicitNativeOwnership(t *testing.T) {
	for _, test := range []struct {
		name, declaration, wantError string
		cgo                          bool
	}{
		{name: "explicit native has no Plan9 owner", declaration: "native.s"},
		{name: "explicit and cgo conflict", declaration: "native.s", cgo: true, wantError: "both cgo and explicit"},
		{name: "duplicate explicit owner", declaration: "native.s; ./native.s", wantError: "duplicate explicit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, pkg := assemblyLanguageTestFixture(t, "example.com/explicit", test.cgo, false)
			source := assemblyLanguageTestWrite(t, pkg.Dir, "native.s", ".text\n")
			pkg.OtherFiles = []string{source}
			pkg.Types.Scope().Insert(types.NewConst(token.NoPos, pkg.Types, "LLGoFiles", types.Typ[types.UntypedString], constant.MakeString(test.declaration)))
			inputs, err := llgoPkgFileInputs(ctx, pkg)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("native ownership error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil || len(inputs) != 1 || inputs[0].path != source {
				t.Fatalf("explicit native compiler owner = %+v, %v", inputs, err)
			}
			files, err := pkgSFiles(ctx, pkg)
			if err != nil || len(files) != 0 {
				t.Fatalf("explicit native also acquired a Plan 9 owner: %v, %v", files, err)
			}
		})
	}
}

func TestPackageAssemblyNativeFlagsFollowOriginalOverlay(t *testing.T) {
	ctx, pkg := assemblyLanguageTestFixture(t, "example.com/flags", true, false)
	source := assemblyLanguageTestWrite(t, pkg.Dir, "native.S", ".text\n")
	pkg.OtherFiles = []string{source}
	ctx.buildConf.Overlay[pkg.GoFiles[0]] = []byte("package fixture\n/*\n#cgo linux CFLAGS: -DOWNER_FLAG=7\n#cgo windows CFLAGS: -DINACTIVE_FLAG=1\n*/\nimport \"C\"\n")
	inputs, err := llgoPkgFileInputs(ctx, pkg)
	if err != nil || len(inputs) != 1 {
		t.Fatalf("actual native input = %+v, %v", inputs, err)
	}
	flags := inputs[0].compilerArgs
	if !slices.Contains(flags, "-DOWNER_FLAG=7") || slices.Contains(flags, "-DINACTIVE_FLAG=1") || !slices.Contains(flags, "-I"+pkg.Dir) {
		t.Fatalf("native compiler lost selected original flags/package include: %q", flags)
	}
}

func TestPackageAssemblyNativeLanguageOverrides(t *testing.T) {
	t.Setenv("CFLAGS", "")
	t.Setenv("CCFLAGS", "")
	for _, test := range []struct {
		name, source string
		args         []string
		assembly     bool
	}{
		{name: "raw extension", source: "file.s", assembly: true},
		{name: "CPP extension", source: "file.S", assembly: true},
		{name: "explicit assembler", source: "file.c", args: []string{"-x", "assembler"}, assembly: true},
		{name: "explicit assembler with CPP", source: "file.c", args: []string{"-xassembler-with-cpp"}, assembly: true},
		{name: "last C wins", source: "file.S", args: []string{"-x", "assembler", "-xc"}},
		{name: "last assembler wins", source: "file.s", args: []string{"-xc", "-x", "assembler"}, assembly: true},
		{name: "none restores assembly extension", source: "file.S", args: []string{"-xc", "-x", "none"}, assembly: true},
		{name: "none restores C extension", source: "file.c", args: []string{"-xassembler-with-cpp", "-xnone"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := llgoFilesBuildContext("", true)
			// Driver-prefix flags precede explicit per-file flags too.
			ctx.crossCompile.CCArgs = []string{"-x", "c"}
			if len(test.args) == 0 {
				ctx.crossCompile.CCArgs = nil
			}
			if got := llgoFileIsAssembly(ctx, test.args, test.source); got != test.assembly {
				t.Fatalf("assembly language = %t, want %t for %q", got, test.assembly, test.args)
			}
		})
	}
}

func nativeAssemblyTestFixture(t *testing.T, extension string) (*context, *packages.Package, string, string) {
	t.Helper()
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("native object execution fixture requires amd64 or arm64")
	}
	ctx, pkg := assemblyLanguageTestFixture(t, "example.com/native", true, false)
	ctx.buildConf.Goos, ctx.buildConf.Goarch = runtime.GOOS, runtime.GOARCH
	ctx.commands.environ = os.Environ()
	pkg.ExportFile = filepath.Join(pkg.Dir, "fixture.a")
	includeDir := filepath.Join(pkg.Dir, "include space")
	if err := os.Mkdir(includeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := fmt.Sprintf("package fixture\n/*\n#cgo CFLAGS: -DOWNER_FLAG=1 -I%q\n*/\nimport \"C\"\n", includeDir)
	if err := os.WriteFile(pkg.GoFiles[0], []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	name := "answer"
	if runtime.GOOS == "darwin" {
		name = "_answer"
	}
	headerName, prefix, headerBody := "answer.inc", ".include \"answer.inc\"\n", ".equ ANSWER_VALUE,42\n"
	if extension == ".S" {
		headerName = "answer.h"
		prefix = "#include \"answer.h\"\n#if OWNER_FLAG != 1\n#error original Cgo flags were lost\n#endif\n"
		headerBody = "#define ANSWER_VALUE 42\n"
	}
	header := assemblyLanguageTestWrite(t, includeDir, headerName, headerBody)
	body := "movl $ANSWER_VALUE,%eax\nret\n"
	if runtime.GOARCH == "arm64" {
		body = "mov w0,#ANSWER_VALUE\nret\n"
	}
	source := assemblyLanguageTestWrite(t, pkg.Dir, "answer"+extension, prefix+".text\n.globl "+name+"\n"+name+":\n"+body)
	pkg.OtherFiles = []string{source}
	return ctx, pkg, source, header
}
