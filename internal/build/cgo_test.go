package build

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/cabi"
	llclang "github.com/xgo-dev/llgo/internal/clang"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	gllvm "github.com/xgo-dev/llvm"
)

func TestParseCgoDeclFlags(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		want        []cgoDecl
		wantErrText string
	}{
		{
			name: "CPPFLAGS with tag",
			line: "#cgo linux CPPFLAGS: -I/usr/lib/llvm-22/include -D_GNU_SOURCE",
			want: []cgoDecl{
				{
					tag:    "linux",
					cflags: []string{"-I/usr/lib/llvm-22/include", "-D_GNU_SOURCE"},
				},
			},
		},
		{
			name: "CFLAGS without tag",
			line: "#cgo CFLAGS: -I/usr/include/python3.12",
			want: []cgoDecl{
				{
					cflags: []string{"-I/usr/include/python3.12"},
				},
			},
		},
		{
			name: "CXXFLAGS without tag",
			line: "#cgo CXXFLAGS: -O2 -stdlib=libc++",
			want: []cgoDecl{
				{
					cxxflags: []string{"-O2", "-stdlib=libc++"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCgoDecl(tt.line)
			if tt.wantErrText != "" {
				if err == nil {
					t.Fatalf("parseCgoDecl expected error containing %q, got nil", tt.wantErrText)
				}
				if !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("parseCgoDecl error = %q, want contains %q", err.Error(), tt.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCgoDecl returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseCgoDecl = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCollectCgoFlags(t *testing.T) {
	cflags, cxxflags, ldflags := collectCgoFlags([]cgoDecl{
		{cflags: []string{"-I/a"}, ldflags: []string{"-la"}},
		{cflags: []string{"-I/b"}, cxxflags: []string{"-std=c++17"}},
	})
	if want := []string{"-I/a", "-I/b"}; !reflect.DeepEqual(cflags, want) {
		t.Fatalf("CFLAGS = %v, want %v", cflags, want)
	}
	if want := []string{"-std=c++17"}; !reflect.DeepEqual(cxxflags, want) {
		t.Fatalf("CXXFLAGS = %v, want %v", cxxflags, want)
	}
	if want := []string{"-la"}; !reflect.DeepEqual(ldflags, want) {
		t.Fatalf("LDFLAGS = %v, want %v", ldflags, want)
	}
}

func TestParseCgoDeclWithCommandEnvBranches(t *testing.T) {
	commands := commandEnv{dir: t.TempDir(), environ: []string{"CGO_TEST_MARKER=1"}}
	tests := []struct {
		name    string
		line    string
		want    []cgoDecl
		wantErr string
	}{
		{
			name: "LDFLAGS with tag",
			line: "#cgo darwin LDFLAGS: -framework CoreFoundation -lz",
			want: []cgoDecl{{tag: "darwin", ldflags: []string{"-framework", "CoreFoundation", "-lz"}}},
		},
		{name: "missing colon", line: "#cgo CFLAGS -I/missing", wantErr: "invalid cgo format"},
		{name: "missing directive", line: "CFLAGS: -I/missing", wantErr: "invalid cgo directive"},
		{name: "unsupported flag", line: "#cgo FOOFLAGS: -unsupported", wantErr: "unsupported cgo flag type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCgoDeclWithCommandEnv(commands, nil, tt.line)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCgoDeclWithCommandEnv(%q) error = %v, want %q", tt.line, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseCgoDeclWithCommandEnv(%q) = %#v, want %#v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseCgoDeclWithCommandEnvPkgConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	dir := t.TempDir()
	tool := filepath.Join(dir, "pkg-config")
	script := `#!/bin/sh
if [ "$PKG_CONFIG_TEST_FAIL" = "$1" ]; then
	exit 1
fi
if [ "$1" = "--libs" ]; then
	printf '%s\n' '-L/request/lib -lrequest'
	exit 0
fi
printf '%s\n' '-I/request/include -DREQUEST="request value"'
`
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	commands := commandEnv{dir: dir, environ: []string{"PATH=" + dir}}
	got, err := parseCgoDeclWithCommandEnv(commands, nil, "#cgo linux pkg-config: request")
	if err != nil {
		t.Fatal(err)
	}
	want := []cgoDecl{{
		tag:     "linux",
		cflags:  []string{"-I/request/include", `-DREQUEST="request value"`},
		ldflags: []string{"-L/request/lib", "-lrequest"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCgoDeclWithCommandEnv(pkg-config) = %#v, want %#v", got, want)
	}

	for _, arg := range []string{"--libs", "--cflags"} {
		t.Run("failed "+arg, func(t *testing.T) {
			commands := commandEnv{dir: dir, environ: []string{"PATH=" + dir, "PKG_CONFIG_TEST_FAIL=" + arg}}
			if _, err := parseCgoDeclWithCommandEnv(commands, nil, "#cgo pkg-config: request"); err == nil || !strings.Contains(err.Error(), "pkg-config") {
				t.Fatalf("parseCgoDeclWithCommandEnv(pkg-config) error = %v, want pkg-config failure", err)
			}
		})
	}
}

func TestParseCgoPreambleRunsOnlyMatchingPkgConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	dir := t.TempDir()
	tool := filepath.Join(dir, "pkg-config")
	logFile := filepath.Join(dir, "calls.log")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$CGO_TEST_LOG"
if [ "$1" = "--libs" ]; then
	printf '%s\n' "-l$2"
fi
`
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	commands := commandEnv{
		dir:     dir,
		environ: []string{"PATH=" + dir, "CGO_TEST_LOG=" + logFile},
	}
	buildCtx := build.Default
	buildCtx.GOOS = "linux"
	buildCtx.GOARCH = "amd64"
	buildCtx.BuildTags = []string{"webkit2_41"}
	text := `#cgo !webkit2_41 pkg-config: webkit2gtk-4.0
#cgo webkit2_41 pkg-config: webkit2gtk-4.1
#cgo amd64 arm64 CFLAGS: -DMULTI_ARCH
#cgo windows plan9 CFLAGS: -DINACTIVE_OS`

	_, decls, err := parseCgoPreambleWithCommandEnv(
		commands,
		&buildCtx,
		token.Position{Filename: "request.go", Line: 1},
		text,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []cgoDecl{
		{tag: "webkit2_41", ldflags: []string{"-lwebkit2gtk-4.1"}},
		{tag: "amd64 arm64", cflags: []string{"-DMULTI_ARCH"}},
	}
	if !reflect.DeepEqual(decls, want) {
		t.Fatalf("cgo declarations = %#v, want %#v", decls, want)
	}
	calls, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(calls), "--libs webkit2gtk-4.1\n--cflags webkit2gtk-4.1\n"; got != want {
		t.Fatalf("pkg-config calls = %q, want %q", got, want)
	}
}

func TestParseCgoDeclWithCommandEnvUsesPkgConfigSetting(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := commandEnv{dir: t.TempDir(), environ: append(os.Environ(),
		`PKG_CONFIG='`+executable+`' --ignored-like-go`,
		"LLGO_TEST_PKG_CONFIG_HELPER=1",
	)}
	got, err := parseCgoDeclWithCommandEnv(commands, nil, "#cgo windows pkg-config: request")
	if err != nil {
		t.Fatal(err)
	}
	want := []cgoDecl{{
		tag:     "windows",
		cflags:  []string{"-I/request/include", `-DREQUEST="request value"`},
		ldflags: []string{"-L/request/lib", "-lrequest"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCgoDeclWithCommandEnv(PKG_CONFIG) = %#v, want %#v", got, want)
	}
}

func TestParseCgoPreambleDelegatesToCommandEnvParser(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "request.go")
	preamble, decls, err := parseCgoPreamble(token.Position{Filename: goFile, Line: 7}, "#cgo CFLAGS: -Iinclude")
	if err != nil {
		t.Fatal(err)
	}
	want := []cgoDecl{{cflags: []string{"-I" + filepath.Join(dir, "include")}}}
	if preamble.goFile != goFile || preamble.src == "" || !reflect.DeepEqual(decls, want) {
		t.Fatalf("parseCgoPreamble() = %#v, %#v", preamble, decls)
	}
}

func TestCollectCgoSymbolsStripsPackagePrefix(t *testing.T) {
	externs := []string{
		"command-line-arguments._cgo_96608f8de8c8_Cfunc_fputs",
		"_cgo_96608f8de8c8_Cfunc_puts",
		"demo._cgo_123456789abc_C2func_errno",
		"demo.__cgo_callback",
	}

	got := collectCgoSymbols(externs)
	want := map[string]string{
		"_cgo_96608f8de8c8_Cfunc__Cmalloc": "_Cmalloc",
		"_cgo_96608f8de8c8_Cfunc_fputs":    "fputs",
		"_cgo_96608f8de8c8_Cfunc_puts":     "puts",
		"_cgo_123456789abc_C2func_errno":   "errno",
		"demo.__cgo_callback":              "__cgo_callback",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectCgoSymbols = %#v, want %#v", got, want)
	}
}

func TestGenExternDeclsUsesConfiguredCompilerAndFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	clangPath := filepath.Join(t.TempDir(), "clang")
	script := `#!/bin/sh
saw_target=false
for arg in "$@"; do
	if [ "$arg" = "configured-target" ]; then
		saw_target=true
	fi
	if [ "$arg" = "-dM" ]; then
		if [ "$saw_target" != "true" ]; then
			exit 2
		fi
		printf '#define request_macro 1\n'
		exit 0
	fi
done
if [ "$saw_target" != "true" ]; then
	exit 2
fi
printf '%s\n' '{"kind":"TranslationUnitDecl","inner":[{"kind":"FunctionDecl","name":"request_func"}]}'
`
	if err := os.WriteFile(clangPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	symbols := map[string]string{
		"cgo_func":  "request_func",
		"cgo_macro": "request_macro",
	}
	compiler := llclang.NewCompiler(llclang.Config{CC: clangPath, CCFLAGS: []string{"-target", "configured-target"}})
	got, err := genExternDeclsByClang(compiler, nil, "", nil, symbols, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"typeof(request_func)* cgo_func;",
		"typeof(request_macro) cgo_macro;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated declarations missing %q:\n%s", want, got)
		}
	}
	if len(symbols) != 0 {
		t.Fatalf("resolved symbols were not removed: %v", symbols)
	}

	for _, tt := range []struct {
		name    string
		script  string
		wantErr string
	}{
		{
			name:    "function probe",
			script:  "#!/bin/sh\nexit 1\n",
			wantErr: "failed to get func names",
		},
		{
			name: "macro probe",
			script: `#!/bin/sh
for arg in "$@"; do
	if [ "$arg" = "-dM" ]; then
		exit 1
	fi
done
printf '%s\n' '{"kind":"TranslationUnitDecl"}'
`,
			wantErr: "failed to get macro names",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clangPath := filepath.Join(t.TempDir(), "clang")
			if err := os.WriteFile(clangPath, []byte(tt.script), 0o755); err != nil {
				t.Fatal(err)
			}
			compiler := llclang.NewCompiler(llclang.Config{CC: clangPath})
			if _, err := genExternDeclsByClang(compiler, nil, "", nil, nil, false); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("genExternDeclsByClang() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGenExternDeclsSkipsMissingAddressPlaceholders(t *testing.T) {
	gllvm.InitializeAllTargets()
	gllvm.InitializeAllTargetMCs()
	gllvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	lpkg := prog.NewPackage("p", "example.com/p")
	pkg := &aPackage{LPkg: lpkg}
	const existing = "example.com/p.__cgo_callback"
	placeholder := lpkg.NewVar(existing, types.NewPointer(types.Typ[types.UnsafePointer]), llssa.GoDecl)
	keep := lpkg.NewVar("example.com/p.keep", types.NewPointer(types.NewPointer(types.Typ[types.UnsafePointer])), llssa.GoDecl)
	keep.Init(placeholder.Expr)
	compiler := llclang.NewCompiler(llclang.Config{})
	for pass := range 2 {
		// The second file can list an already-lowered placeholder, or an
		// address that executable compilation did not materialize at all.
		symbols := map[string]string{
			existing:                      "__cgo_callback",
			"example.com/p.__cgo_missing": "__cgo_missing",
		}
		if _, err := genExternDeclsByClang(compiler, pkg, "void callback(void);", nil, symbols, false); err != nil {
			t.Fatal(err)
		}
		if len(symbols) != 0 {
			t.Fatalf("pass %d retained resolved symbols: %v", pass, symbols)
		}
		if lpkg.VarOf(existing) != nil || lpkg.FuncOf("callback") == nil {
			t.Fatalf("pass %d lost the real callback address replacement", pass)
		}
		if lpkg.Module().NamedGlobal("example.com/p.keep").Initializer() != lpkg.Module().NamedFunction("callback") {
			t.Fatalf("pass %d did not preserve the callback address", pass)
		}
		if lpkg.FuncOf("missing") != nil || !lpkg.Module().NamedFunction("missing").IsNil() {
			t.Fatalf("pass %d created a C declaration for a missing placeholder:\n%s", pass, lpkg.Module().String())
		}
		if err := gllvm.VerifyModule(lpkg.Module(), gllvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenExternDeclsUsesPublicExportAddress(t *testing.T) {
	gllvm.InitializeAllTargets()
	gllvm.InitializeAllTargetMCs()
	gllvm.InitializeAllTargetInfos()
	for _, test := range []struct {
		name, exportGoName, want string
	}{
		{name: "ordinary Go function", want: "example.com/p.callback"},
		{name: "exported Go function", exportGoName: "example.com/p.callback", want: "callback"},
		{name: "renamed export shadows private Go function", exportGoName: "example.com/p.Exported", want: "callback"},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := llssa.NewProgram(nil)
			defer prog.Dispose()
			lpkg := prog.NewPackage("p", "example.com/p")
			lpkg.NewFunc("example.com/p.callback", llssa.NoArgsNoRet, llssa.GoDecl)
			if test.exportGoName != "" {
				if test.exportGoName != "example.com/p.callback" {
					lpkg.NewFunc(test.exportGoName, llssa.NoArgsNoRet, llssa.GoDecl)
				}
				lpkg.SetExport(test.exportGoName, "callback")
			}
			const name = "example.com/p.__cgo_callback"
			placeholder := lpkg.NewVar(name, types.NewPointer(types.Typ[types.UnsafePointer]), llssa.GoDecl)
			keep := lpkg.NewVar("example.com/p.keep", types.NewPointer(types.NewPointer(types.Typ[types.UnsafePointer])), llssa.GoDecl)
			keep.Init(placeholder.Expr)
			compiler := llclang.NewCompiler(llclang.Config{})
			if _, err := genExternDeclsByClang(compiler, &aPackage{LPkg: lpkg}, "void callback(void);", nil,
				map[string]string{name: "__cgo_callback"}, false); err != nil {
				t.Fatal(err)
			}
			if got, want := lpkg.Module().NamedGlobal("example.com/p.keep").Initializer(), lpkg.Module().NamedFunction(test.want); got != want {
				t.Fatalf("C callback address = %s, want %s", got.Name(), test.want)
			}
			if err := gllvm.VerifyModule(lpkg.Module(), gllvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBuildCgoReportsClangProbeFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	dir := t.TempDir()
	clang := filepath.Join(dir, "clang")
	if err := os.WriteFile(clang, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	src := `package demo

/*
int request_value;
*/
import "unsafe"
`
	goFile := filepath.Join(dir, "demo.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, goFile, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &context{
		conf:      &packages.Config{},
		buildConf: &Config{},
	}
	pkg := &aPackage{Package: &packages.Package{Fset: fset}}
	if _, _, err := buildCgo(ctx, pkg, []*ast.File{file}, nil, false); err == nil || !strings.Contains(err.Error(), "failed to generate extern decls") {
		t.Fatalf("buildCgo() error = %v, want clang probe failure", err)
	}
}

func buildCgoTestFile(t *testing.T, dir, preamble string) (*ast.File, *aPackage, *context) {
	t.Helper()
	src := "package demo\n\n/*\n" + preamble + "\n*/\nimport \"unsafe\"\n"
	goFile := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(goFile, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, goFile, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{
		Fset:       fset,
		PkgPath:    "example.com/demo",
		ExportFile: filepath.Join(dir, "demo.a"),
	}}
	ctx := &context{
		conf:      &packages.Config{},
		buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
		commands: commandEnv{
			dir:     dir,
			environ: os.Environ(),
		},
	}
	return file, pkg, ctx
}

func TestBuildCgoCompilesSourcesAndPreambles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "extra.c"), []byte("int extra_value = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, pkg, ctx := buildCgoTestFile(t, dir, "int preamble_value = 2;")
	objects, _, err := buildCgo(ctx, pkg, []*ast.File{file}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	defer removeFiles(objects)
	if len(objects) != 2 {
		t.Fatalf("buildCgo objects = %v, want source and preamble objects", objects)
	}
	for _, object := range objects {
		if _, err := os.Stat(object); err != nil {
			t.Fatalf("compiled object %q: %v", object, err)
		}
	}
}

func TestBuildCgoReportsSourceAndPreambleCompileErrors(t *testing.T) {
	t.Run("source", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "broken.c"), []byte("this is not C;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		file, pkg, ctx := buildCgoTestFile(t, dir, "")
		if _, _, err := buildCgo(ctx, pkg, []*ast.File{file}, nil, false); err == nil {
			t.Fatal("buildCgo unexpectedly compiled an invalid C source")
		}
	})

	t.Run("preamble assembly", func(t *testing.T) {
		dir := t.TempDir()
		// Clang accepts this during the syntax and preprocessor probes used to
		// discover cgo symbols, but its assembler rejects the directive when
		// clFile performs the real object compilation.
		file, pkg, ctx := buildCgoTestFile(t, dir, `__asm__(".llgo_invalid_directive");`)
		if _, _, err := buildCgo(ctx, pkg, []*ast.File{file}, nil, false); err == nil {
			t.Fatal("buildCgo unexpectedly compiled an invalid preamble directive")
		}
	})
}

func TestParseCgoCollectsNativeFiles(t *testing.T) {
	dir := t.TempDir()
	src := `package demo

/*
#cgo CFLAGS: -Ic
#cgo CXXFLAGS: -Icxx
*/
import "unsafe"
`
	goFile := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(goFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"foo.c", "bar.cc", "baz.cpp", "qux.cxx", "objc.m", "skip_test.cpp", "skip_test.m"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, goFile, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{Fset: fset}}
	buildCtx := build.Default
	srcFiles, _, decls, err := parseCgo_(&buildCtx, pkg, []*ast.File{file})
	if err != nil {
		t.Fatalf("parseCgo_ returned error: %v", err)
	}

	gotFiles := map[string]bool{}
	for _, src := range srcFiles {
		gotFiles[filepath.Base(src.path)] = src.isCXX
	}
	wantFiles := map[string]bool{
		"foo.c":   false,
		"bar.cc":  true,
		"baz.cpp": true,
		"qux.cxx": true,
		"objc.m":  false,
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("parseCgo_ files = %#v, want %#v", gotFiles, wantFiles)
	}
	if !reflect.DeepEqual(decls, []cgoDecl{
		{cflags: []string{"-I" + filepath.Join(dir, "c")}},
		{cxxflags: []string{"-I" + filepath.Join(dir, "cxx")}},
	}) {
		t.Fatalf("parseCgo_ decls = %#v", decls)
	}
}

func TestParseCgoIgnoresDirectoryNamedLikeCFile(t *testing.T) {
	dir := t.TempDir()
	src := `package demo

/*
#cgo CFLAGS: -I/c
*/
import "unsafe"
`
	goFile := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(goFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "foo.c"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bar.c"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, goFile, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{Fset: fset}}
	buildCtx := build.Default
	srcFiles, _, _, err := parseCgo_(&buildCtx, pkg, []*ast.File{file})
	if err != nil {
		t.Fatalf("parseCgo_ returned error: %v", err)
	}
	if len(srcFiles) != 1 || filepath.Base(srcFiles[0].path) != "bar.c" {
		t.Fatalf("parseCgo_ files = %#v, want only bar.c", srcFiles)
	}
}

func TestParseCgoSkipsBuildTaggedNativeFiles(t *testing.T) {
	dir := t.TempDir()
	goSrc := `package demo

/*
*/
import "unsafe"
`
	goFile := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(goFile, []byte(goSrc), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"skip.cpp", "skip.m"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("//go:build missingtag\n\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.cpp"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, goFile, goSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{Fset: fset}}
	buildCtx := build.Default
	srcFiles, _, _, err := parseCgo_(&buildCtx, pkg, []*ast.File{file})
	if err != nil {
		t.Fatalf("parseCgo_ returned error: %v", err)
	}
	if len(srcFiles) != 1 || filepath.Base(srcFiles[0].path) != "keep.cpp" || !srcFiles[0].isCXX {
		t.Fatalf("parseCgo_ files = %#v, want only keep.cpp as C++", srcFiles)
	}
}

func TestEmitDarwinDynimportTrampolineIncludesLocalAddress(t *testing.T) {
	for _, goarch := range []string{"arm64", "amd64"} {
		t.Run(goarch, func(t *testing.T) {
			var b strings.Builder
			emitDarwinDynimportTrampoline(&b, goarch, "local", "alias")
			got := b.String()
			for _, want := range []string{
				"_local:\n",
				"_local_trampoline:\n",
				"_local_trampoline_addr:\n\t.quad _local_trampoline\n",
			} {
				if !strings.Contains(got, want) {
					t.Fatalf("trampoline asm for %s missing %q:\n%s", goarch, want, got)
				}
			}
		})
	}
}

func TestLowerWindowsCgoImportPointer(t *testing.T) {
	llvmCtx := gllvm.NewContext()
	defer llvmCtx.Dispose()
	mod := llvmCtx.NewModule("windows-dynimport")
	defer mod.Dispose()

	ptrType := gllvm.PointerType(llvmCtx.Int8Type(), 0)
	global := gllvm.AddGlobal(mod, ptrType, "syscall.__LoadLibraryExW")
	global.SetInitializer(gllvm.ConstPointerNull(ptrType))
	file, err := parser.ParseFile(token.NewFileSet(), "dll_windows.go", `package syscall
//go:cgo_import_dynamic syscall.__LoadLibraryExW LoadLibraryExW%3 "kernel32.dll"
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if err := lowerWindowsCgoImportPointers("windows", "386", "syscall", []*ast.File{file}, mod); err != nil {
		t.Fatal(err)
	}
	fn := mod.NamedFunction("LoadLibraryExW")
	if fn.IsNil() {
		t.Fatal("Windows dynamic import declaration was not emitted")
	}
	if fn.DLLStorageClass() != gllvm.DLLImportStorageClass {
		t.Fatal("Windows dynamic import declaration is not dllimport")
	}
	if fn.FunctionCallConv() != gllvm.X86StdcallCallConv {
		t.Fatal("Windows 386 dynamic import did not preserve stdcall decoration")
	}
	if init := global.Initializer(); init.IsNil() || init != fn {
		t.Fatalf("Windows dynamic import pointer initializer = %v, want %v", init, fn)
	}
	if err := gllvm.VerifyModule(mod, gllvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid Windows dynamic import module: %v\n%s", err, mod.String())
	}
}

func TestCompilePackageModuleLowersWindowsCgoImportPointer(t *testing.T) {
	gllvm.InitializeAllTargets()
	gllvm.InitializeAllTargetMCs()
	gllvm.InitializeAllTargetInfos()

	target := &llssa.Target{GOOS: "windows", GOARCH: "amd64"}
	prog := llssa.NewProgram(target)
	defer prog.Dispose()
	lpkg := prog.NewPackage("syscall", "syscall")
	ptrType := gllvm.PointerType(lpkg.Module().Context().Int8Type(), 0)
	global := gllvm.AddGlobal(lpkg.Module(), ptrType, "syscall.__LoadLibraryExW")
	global.SetInitializer(gllvm.ConstPointerNull(ptrType))
	file, err := parser.ParseFile(token.NewFileSet(), "dll_windows.go", `package syscall
//go:cgo_import_dynamic syscall.__LoadLibraryExW LoadLibraryExW%3 "kernel32.dll"
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	conf := &Config{
		Goos:   "windows",
		Goarch: "amd64",
	}
	ctx := &context{
		prog:         prog,
		mode:         ModeGen,
		buildConf:    conf,
		cTransformer: cabi.NewTransformer(prog, target.Spec().Triple, "", true),
	}
	pkg := &aPackage{
		Package: &packages.Package{PkgPath: "syscall"},
		AltPkg: &packages.Cached{
			Package: &packages.Package{},
			Syntax:  []*ast.File{file},
		},
		LPkg: lpkg,
	}
	if _, dynimports := collectGoCgoPragmas(pkg.AltPkg.Syntax); len(dynimports) != 1 {
		t.Fatalf("alternate package dynamic imports = %d, want 1", len(dynimports))
	}
	if err := compilePackageModule(ctx, pkg, nil, false); err != nil {
		t.Fatal(err)
	}
	fn := lpkg.Module().NamedFunction("LoadLibraryExW")
	if fn.IsNil() || fn.DLLStorageClass() != gllvm.DLLImportStorageClass {
		t.Fatalf("compiled Windows import = %v, want dllimport declaration", fn)
	}
	if init := global.Initializer(); init.IsNil() || init != fn {
		t.Fatalf("compiled Windows import pointer initializer = %v, want %v", init, fn)
	}
}

func TestCompilePackageModuleReportsWindowsCgoImportError(t *testing.T) {
	gllvm.InitializeAllTargets()
	gllvm.InitializeAllTargetMCs()
	gllvm.InitializeAllTargetInfos()

	target := &llssa.Target{GOOS: "windows", GOARCH: "amd64"}
	prog := llssa.NewProgram(target)
	defer prog.Dispose()
	lpkg := prog.NewPackage("syscall", "syscall")
	ptrType := gllvm.PointerType(lpkg.Module().Context().Int8Type(), 0)
	gllvm.AddGlobal(lpkg.Module(), ptrType, "syscall.value")
	file, err := parser.ParseFile(token.NewFileSet(), "dll_windows.go", `package syscall
//go:cgo_import_dynamic syscall.value Value%bad "kernel32.dll"
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	conf := &Config{
		Goos:   "windows",
		Goarch: "amd64",
	}
	ctx := &context{
		prog:         prog,
		mode:         ModeGen,
		buildConf:    conf,
		cTransformer: cabi.NewTransformer(prog, target.Spec().Triple, "", true),
	}
	pkg := &aPackage{
		Package: &packages.Package{PkgPath: "syscall", Syntax: []*ast.File{file}},
		LPkg:    lpkg,
	}
	err = compilePackageModule(ctx, pkg, nil, false)
	if err == nil || !strings.Contains(err.Error(), "invalid go:cgo_import_dynamic alias") {
		t.Fatalf("compilePackageModule error = %v, want invalid dynamic-import alias", err)
	}
}

func TestCompilePackageModulePropagatesSFileErrors(t *testing.T) {
	gllvm.InitializeAllTargets()
	gllvm.InitializeAllTargetMCs()
	gllvm.InitializeAllTargetInfos()

	for _, withAltPkg := range []bool{false, true} {
		name := "package"
		if withAltPkg {
			name = "alternate package"
		}
		t.Run(name, func(t *testing.T) {
			target := &llssa.Target{GOOS: "linux", GOARCH: "amd64"}
			prog := llssa.NewProgram(target)
			defer prog.Dispose()
			lpkg := prog.NewPackage("example.com/p", "example.com/p")
			conf := &Config{
				Goos:   "linux",
				Goarch: "amd64",
			}
			ctx := &context{
				prog:         prog,
				mode:         ModeBuild,
				buildConf:    conf,
				sfilesFrozen: true,
				cTransformer: cabi.NewTransformer(prog, target.Spec().Triple, "", true),
			}
			pkg := &aPackage{
				Package: &packages.Package{ID: "example.com/p", PkgPath: "example.com/p"},
				LPkg:    lpkg,
			}
			if withAltPkg {
				pkg.AltPkg = &packages.Cached{Package: &packages.Package{
					ID:      "example.com/p.alt",
					PkgPath: "example.com/p.alt",
				}}
			}
			err := compilePackageModule(ctx, pkg, nil, false)
			if err == nil || !strings.Contains(err.Error(), "assembly files were not prepared") {
				t.Fatalf("compilePackageModule error = %v, want frozen SFiles error", err)
			}
		})
	}
}

func TestLowerWindowsCgoImportPointerErrors(t *testing.T) {
	parse := func(t *testing.T, src string) *ast.File {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), "dll_windows.go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}

	t.Run("non-pointer", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("windows-dynimport-non-pointer")
		defer mod.Dispose()
		gllvm.AddGlobal(mod, llvmCtx.Int32Type(), "syscall.value")
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.value Value%0 "kernel32.dll"
`)
		err := lowerWindowsCgoImportPointers("windows", "arm64", "syscall", []*ast.File{file}, mod)
		if err == nil || !strings.Contains(err.Error(), "is not a pointer variable") {
			t.Fatalf("lowerWindowsCgoImportPointers error = %v, want non-pointer error", err)
		}
	})

	t.Run("conflict", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("windows-dynimport-conflict")
		defer mod.Dispose()
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.value First%0 "kernel32.dll"
//go:cgo_import_dynamic syscall.value Second%0 "kernel32.dll"
`)
		err := lowerWindowsCgoImportPointers("windows", "arm64", "syscall", []*ast.File{file}, mod)
		if err == nil || !strings.Contains(err.Error(), "conflicting go:cgo_import_dynamic") {
			t.Fatalf("lowerWindowsCgoImportPointers error = %v, want conflict error", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("windows-dynimport-duplicate")
		defer mod.Dispose()
		ptrType := gllvm.PointerType(llvmCtx.Int8Type(), 0)
		global := gllvm.AddGlobal(mod, ptrType, "syscall.value")
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.value Value%0 "kernel32.dll"
//go:cgo_import_dynamic syscall.value Value%0 "kernel32.dll"
`)
		if err := lowerWindowsCgoImportPointers("windows", "386", "syscall", []*ast.File{file}, mod); err != nil {
			t.Fatal(err)
		}
		if init := global.Initializer(); init.IsNil() || init != mod.NamedFunction("Value") {
			t.Fatalf("duplicate import initializer = %v, want Value", init)
		}
	})

	t.Run("invalid alias", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("windows-dynimport-invalid-alias")
		defer mod.Dispose()
		ptrType := gllvm.PointerType(llvmCtx.Int8Type(), 0)
		gllvm.AddGlobal(mod, ptrType, "syscall.value")
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.value Value%bad "kernel32.dll"
`)
		err := lowerWindowsCgoImportPointers("windows", "386", "syscall", []*ast.File{file}, mod)
		if err == nil || !strings.Contains(err.Error(), "invalid go:cgo_import_dynamic alias") {
			t.Fatalf("lowerWindowsCgoImportPointers error = %v, want invalid-alias error", err)
		}
	})

	t.Run("defined function collision", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("windows-dynimport-defined-function")
		defer mod.Dispose()
		ptrType := gllvm.PointerType(llvmCtx.Int8Type(), 0)
		gllvm.AddGlobal(mod, ptrType, "syscall.value")
		fn := gllvm.AddFunction(mod, "Value", gllvm.FunctionType(llvmCtx.VoidType(), nil, false))
		llvmCtx.AddBasicBlock(fn, "entry")
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.value Value%0 "kernel32.dll"
`)
		err := lowerWindowsCgoImportPointers("windows", "386", "syscall", []*ast.File{file}, mod)
		if err == nil || !strings.Contains(err.Error(), "collides with a defined function") {
			t.Fatalf("lowerWindowsCgoImportPointers error = %v, want defined-function collision", err)
		}
		if got := fn.DLLStorageClass(); got == gllvm.DLLImportStorageClass {
			t.Fatalf("defined function DLL storage class = %v, want unchanged", got)
		}
	})

	t.Run("existing declaration", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("windows-dynimport-declaration")
		defer mod.Dispose()
		ptrType := gllvm.PointerType(llvmCtx.Int8Type(), 0)
		global := gllvm.AddGlobal(mod, ptrType, "syscall.value")
		fn := gllvm.AddFunction(mod, "Value", gllvm.FunctionType(llvmCtx.VoidType(), nil, false))
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.value Value%0 "kernel32.dll"
`)
		if err := lowerWindowsCgoImportPointers("windows", "386", "syscall", []*ast.File{file}, mod); err != nil {
			t.Fatal(err)
		}
		if got := fn.DLLStorageClass(); got != gllvm.DLLImportStorageClass {
			t.Fatalf("existing declaration DLL storage class = %v, want dllimport", got)
		}
		if got := fn.FunctionCallConv(); got != gllvm.X86StdcallCallConv {
			t.Fatalf("existing declaration calling convention = %v, want stdcall", got)
		}
		if init := global.Initializer(); init.IsNil() || init != fn {
			t.Fatalf("Windows dynamic import pointer initializer = %v, want %v", init, fn)
		}
	})

	t.Run("conflicting alias argument counts", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("windows-dynimport-conflicting-alias-counts")
		defer mod.Dispose()
		ptrType := gllvm.PointerType(llvmCtx.Int8Type(), 0)
		gllvm.AddGlobal(mod, ptrType, "syscall.first")
		gllvm.AddGlobal(mod, ptrType, "syscall.second")
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.first Value%1 "kernel32.dll"
//go:cgo_import_dynamic syscall.second Value%2 "kernel32.dll"
`)
		err := lowerWindowsCgoImportPointers("windows", "386", "syscall", []*ast.File{file}, mod)
		if err == nil || !strings.Contains(err.Error(), "conflicting go:cgo_import_dynamic argument counts") {
			t.Fatalf("lowerWindowsCgoImportPointers error = %v, want conflicting argument-count error", err)
		}
	})

	t.Run("other target", func(t *testing.T) {
		llvmCtx := gllvm.NewContext()
		defer llvmCtx.Dispose()
		mod := llvmCtx.NewModule("non-windows-dynimport")
		defer mod.Dispose()
		file := parse(t, `package syscall
//go:cgo_import_dynamic syscall.value Value%bad "kernel32.dll"
`)
		if err := lowerWindowsCgoImportPointers("linux", "amd64", "syscall", []*ast.File{file}, mod); err != nil {
			t.Fatalf("non-Windows lowering failed: %v", err)
		}
	})
}

func TestSplitWindowsCgoImportAlias(t *testing.T) {
	tests := []struct {
		alias       string
		name        string
		argc        int
		hasArgCount bool
		wantErr     bool
	}{
		{alias: "GetProcAddress%2", name: "GetProcAddress", argc: 2, hasArgCount: true},
		{alias: "GetCurrentProcessId%0", name: "GetCurrentProcessId", hasArgCount: true},
		{alias: "plain", name: "plain"},
		{alias: "bad%", wantErr: true},
		{alias: "bad%no", wantErr: true},
		{alias: "bad%1%2", wantErr: true},
	}
	for _, test := range tests {
		name, argc, hasArgCount, err := splitWindowsCgoImportAlias(test.alias)
		if (err != nil) != test.wantErr {
			t.Fatalf("splitWindowsCgoImportAlias(%q) error = %v, wantErr %v", test.alias, err, test.wantErr)
		}
		if err == nil && (name != test.name || argc != test.argc || hasArgCount != test.hasArgCount) {
			t.Fatalf("splitWindowsCgoImportAlias(%q) = (%q, %d, %v), want (%q, %d, %v)", test.alias, name, argc, hasArgCount, test.name, test.argc, test.hasArgCount)
		}
	}
}

func TestShouldSkipDarwinDynimportTrampolineAsm(t *testing.T) {
	src := []byte("TEXT _trampoline<>(SB),$0-0\nDATA _trampoline_addr(SB)/8,$0\n")
	fileSrc := `package unix
//go:cgo_import_dynamic libc_read read
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "unix.go", fileSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{Goos: "darwin", Goarch: "arm64"}}
	pkg := &packages.Package{PkgPath: "golang.org/x/sys/unix", Syntax: []*ast.File{file}}
	enabled := shouldCheckDarwinDynimportTrampolineAsm(ctx, pkg)
	if !shouldSkipDarwinDynimportTrampolineAsm(enabled, "zsyscall_darwin_arm64.s", src) {
		t.Fatal("expected generated dynimport trampoline asm to be skipped")
	}
}
