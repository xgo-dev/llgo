//go:build !llgo
// +build !llgo

package build

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
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
			line: "#cgo linux CPPFLAGS: -I/usr/lib/llvm-19/include -D_GNU_SOURCE",
			want: []cgoDecl{
				{
					tag:    "linux",
					cflags: []string{"-I/usr/lib/llvm-19/include", "-D_GNU_SOURCE"},
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
			want: []cgoDecl{{tag: "darwin", ldflags: []string{"-framework CoreFoundation", "-lz"}}},
		},
		{name: "missing colon", line: "#cgo CFLAGS -I/missing", wantErr: "invalid cgo format"},
		{name: "missing directive", line: "CFLAGS: -I/missing", wantErr: "invalid cgo directive"},
		{name: "unsupported flag", line: "#cgo FOOFLAGS: -unsupported", wantErr: "unsupported cgo flag type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCgoDeclWithCommandEnv(commands, tt.line)
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
	got, err := parseCgoDeclWithCommandEnv(commands, "#cgo linux pkg-config: request")
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
			if _, err := parseCgoDeclWithCommandEnv(commands, "#cgo pkg-config: request"); err == nil || !strings.Contains(err.Error(), "pkg-config") {
				t.Fatalf("parseCgoDeclWithCommandEnv(pkg-config) error = %v, want pkg-config failure", err)
			}
		})
	}
}

func TestParseCgoPreambleDelegatesToCommandEnvParser(t *testing.T) {
	preamble, decls, err := parseCgoPreamble(token.Position{Filename: "request.go", Line: 7}, "#cgo CFLAGS: -I/request/include")
	if err != nil {
		t.Fatal(err)
	}
	if preamble.goFile != "request.go" || preamble.src == "" || !reflect.DeepEqual(decls, []cgoDecl{{cflags: []string{"-I/request/include"}}}) {
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

func TestGenExternDeclsUsesProcessLLVMPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	clang := filepath.Join(t.TempDir(), "clang")
	script := `#!/bin/sh
for arg in "$@"; do
	if [ "$arg" = "-dM" ]; then
		printf '#define request_macro 1\n'
		exit 0
	fi
done
printf '%s\n' '{"kind":"TranslationUnitDecl","inner":[{"kind":"FunctionDecl","name":"request_func"}]}'
`
	if err := os.WriteFile(clang, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	symbols := map[string]string{
		"cgo_func":  "request_func",
		"cgo_macro": "request_macro",
	}
	t.Setenv("PATH", filepath.Dir(clang))
	got, err := genExternDeclsByClang(commandEnv{}, nil, "", nil, symbols, true)
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
			clang := filepath.Join(t.TempDir(), "clang")
			if err := os.WriteFile(clang, []byte(tt.script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", filepath.Dir(clang))
			if _, err := genExternDeclsByClang(commandEnv{}, nil, "", nil, nil, false); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("genExternDeclsByClang() error = %v, want %q", err, tt.wantErr)
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

func TestParseCgoCollectsCXXFiles(t *testing.T) {
	dir := t.TempDir()
	src := `package demo

/*
#cgo CFLAGS: -I/c
#cgo CXXFLAGS: -I/cxx
*/
import "unsafe"
`
	goFile := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(goFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"foo.c", "bar.cc", "baz.cpp", "qux.cxx", "skip_test.cpp"} {
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
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("parseCgo_ files = %#v, want %#v", gotFiles, wantFiles)
	}
	if !reflect.DeepEqual(decls, []cgoDecl{
		{cflags: []string{"-I/c"}},
		{cxxflags: []string{"-I/cxx"}},
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

func TestParseCgoSkipsBuildTaggedCXXFile(t *testing.T) {
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
	cxxSrc := "//go:build missingtag\n\n"
	if err := os.WriteFile(filepath.Join(dir, "skip.cpp"), []byte(cxxSrc), 0644); err != nil {
		t.Fatal(err)
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
