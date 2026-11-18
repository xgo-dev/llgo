package build

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestRuntimeSourceVersionSelection(t *testing.T) {
	for _, test := range []struct {
		version, indexType, adapter string
		crash                       bool
	}{
		{"go1.22", "int", "utf8_index.go", false},
		{"go1.26", "int", "utf8_index.go", true},
		{"go1.27", "uint", "utf8_index_go127.go", true},
	} {
		t.Run(test.version, func(t *testing.T) {
			root, goroot := runtimeSourceTestRoot(t), t.TempDir()
			// Small source fixtures isolate version selection. The toolchain
			// qualification below checks the actual upstream algorithms.
			base := map[string][]byte{
				filepath.Join(goroot, "src/runtime/utf8.go"): []byte(fmt.Sprintf(`package runtime
func countrunes(s string) int { return len(s) }
func decoderune(s string, k %[1]s) (r rune, pos %[1]s) { return 0, k }
func encoderune(p []byte, r rune) int { return 0 }
`, test.indexType)),
			}
			if test.crash {
				base[filepath.Join(goroot, "src/runtime/runtime.go")] = []byte(
					"package runtime\nfunc setCrashFD(fd uintptr) uintptr { return fd }\n")
			}
			for _, name := range []string{"complex.go", "float.go"} {
				data, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/runtime", name))
				if err != nil {
					t.Fatal(err)
				}
				base[filepath.Join(goroot, "src/runtime", name)] = data
			}
			overlay, err := runtimeSourceOverlay(base, root, goroot, test.version)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := overlay[filepath.Join(root, "internal/runtime", test.adapter)]; !ok {
				t.Fatal("selected the wrong decoder index adapter")
			}
			_, hasCrash := overlay[filepath.Join(root, "internal/lib/runtime/crash_fd_go123.go")]
			if hasCrash != test.crash {
				t.Fatalf("crash hook present = %v, want %v", hasCrash, test.crash)
			}
		})
	}
}

func TestRuntimeSourceInvalidAdapter(t *testing.T) {
	if _, err := runtimeSourceOverlay(nil, t.TempDir(), runtime.GOROOT(), runtime.Version()); err == nil ||
		!strings.Contains(err.Error(), "read runtime UTF-8 ABI adapter") {
		t.Fatalf("missing adapter: %v", err)
	}
	root := runtimeSourceTestRoot(t)
	base := make(map[string][]byte)
	for _, name := range []string{"utf8_index.go", "utf8_index_go127.go"} {
		base[filepath.Join(root, "internal/runtime", name)] = []byte("package runtime\nfunc (")
	}
	if _, err := runtimeSourceOverlay(base, root, runtime.GOROOT(), runtime.Version()); err == nil {
		t.Fatal("accepted malformed adapter source")
	}
}

func TestRuntimeSourceBuildRejectsMissingHelpers(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "unused")
	conf.Overlay = map[string][]byte{
		filepath.Join(runtime.GOROOT(), "src/runtime/utf8.go"): []byte("package runtime\n"),
	}
	_, err = Build(Invocation{Args: []string{"."}, Config: conf,
		Dir: filepath.Join(root, "internal/build/testdata/runtimecompiler")})
	if err == nil || !strings.Contains(err.Error(), "missing helpers") {
		t.Fatalf("build did not report incomplete upstream runtime source: %v", err)
	}
}

func runtimeSourceTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "internal/runtime")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"utf8_index.go", "utf8_index_go127.go"} {
		data, err := os.ReadFile(filepath.Join(env.LLGoRuntimeDir(), "internal/runtime", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestRuntimeSourceOverlay(t *testing.T) {
	root := runtimeSourceTestRoot(t)
	overlay, err := runtimeSourceOverlay(nil, root, runtime.GOROOT(), runtime.Version())
	if err != nil {
		t.Fatal(err)
	}
	utf8Path := filepath.Join(root, "internal/runtime/utf8.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, utf8Path, overlay[utf8Path], 0)
	if err != nil {
		t.Fatal(err)
	}
	// The imported algorithm must be self-contained on both pointer widths.
	for _, arch := range []string{"386", "amd64"} {
		config := types.Config{Sizes: types.SizesFor("gc", arch)}
		if _, err := config.Check("runtime", fset, []*ast.File{file}, nil); err != nil {
			t.Fatalf("%s: %v", arch, err)
		}
	}
	if got := fset.Position(file.Decls[0].Pos()).Filename; filepath.Clean(got) != filepath.Join(runtime.GOROOT(), "src", "runtime", "utf8.go") {
		t.Fatalf("source identity = %q", got)
	}
	crashPath := filepath.Join(root, "internal/lib/runtime/crash_fd_go123.go")
	crash, err := parser.ParseFile(token.NewFileSet(), crashPath, overlay[crashPath], parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range crash.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name != "setCrashFD" {
			t.Fatalf("selected unrelated runtime helper %s", fn.Name.Name)
		}
	}
	if !strings.Contains(string(overlay[crashPath]), "//go:linkname setCrashFD") {
		t.Fatal("lost the standard library linkname")
	}
	if _, err := os.Stat(utf8Path); !os.IsNotExist(err) {
		t.Fatal("reuse wrote to the runtime source tree")
	}
}

func TestRuntimeSourceRejectsUnreviewedUpgrade(t *testing.T) {
	root := runtimeSourceTestRoot(t)
	srcPath := filepath.Join(runtime.GOROOT(), "src/runtime/utf8.go")
	src, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, old, replacement, want string
	}{
		{"missing", "func encoderune", "func renamedEncoder", "unreviewed helper"},
		{"signature", "p []byte", "p []int", "unreviewed helper"},
		{"dependency", "package runtime", "package runtime\nimport _ \"unsafe\"", "unreviewed import"},
		{"state", "package runtime", "package runtime\nvar global int", "unreviewed var"},
		{"constant", "runeSelf", "newRuneSelf", "unreviewed constant"},
		{"linkname", "func encoderune", "//go:linkname encoderune\nfunc encoderune", "unreviewed directive"},
		{"noescape", "func encoderune", "//go:noescape\nfunc encoderune", "unreviewed directive"},
		{"build", "package runtime", "//go:build go1.27\n\npackage runtime", "unreviewed directive"},
		{"legacy-build", "package runtime", "// +build go1.27\n\npackage runtime", "unreviewed directive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !strings.Contains(string(src), test.old) {
				t.Fatalf("GOROOT fixture no longer contains %q", test.old)
			}
			base := map[string][]byte{srcPath: []byte(strings.Replace(string(src), test.old, test.replacement, 1))}
			_, err := runtimeSourceOverlay(base, root, runtime.GOROOT(), runtime.Version())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
			if len(base) != 1 {
				t.Fatal("changed caller overlay")
			}
		})
	}
}

func TestRuntimeSourceBodyChangesFingerprint(t *testing.T) {
	root := runtimeSourceTestRoot(t)
	srcPath := filepath.Join(runtime.GOROOT(), "src/runtime/utf8.go")
	src, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "internal/runtime/utf8.go")
	first, err := runtimeSourceOverlay(nil, root, runtime.GOROOT(), runtime.Version())
	if err != nil {
		t.Fatal(err)
	}
	const old = "return runeError, k + 1"
	if !strings.Contains(string(src), old) {
		t.Fatalf("GOROOT fixture no longer contains %q", old)
	}
	second, err := runtimeSourceOverlay(map[string][]byte{
		srcPath: []byte(strings.Replace(string(src), old, "return runeError, k + 2", 1)),
	}, root, runtime.GOROOT(), runtime.Version())
	if err != nil {
		t.Fatal(err)
	}
	a, err := digestFilesWithOverlay([]string{target}, first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := digestFilesWithOverlay([]string{target}, second)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(a, b) {
		t.Fatal("GOROOT body change did not invalidate the runtime cache")
	}
}

func TestRuntimeSourceMissingAdapterDeclaration(t *testing.T) {
	root := runtimeSourceTestRoot(t)
	base := make(map[string][]byte)
	for _, name := range []string{"utf8_index.go", "utf8_index_go127.go"} {
		base[filepath.Join(root, "internal/runtime", name)] = []byte("package runtime")
	}
	_, err := runtimeSourceOverlay(base, root, runtime.GOROOT(), runtime.Version())
	if err == nil || !strings.Contains(err.Error(), "missing decoderune declaration") {
		t.Fatalf("missing adapter declaration: %v", err)
	}
}

func TestRuntimeSourceOverlayIsIdempotent(t *testing.T) {
	root := runtimeSourceTestRoot(t)
	first, err := runtimeSourceOverlay(nil, root, runtime.GOROOT(), runtime.Version())
	if err != nil {
		t.Fatal(err)
	}
	second, err := runtimeSourceOverlay(first, root, runtime.GOROOT(), runtime.Version())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated package load changed the runtime source overlay")
	}
}

func TestRuntimeSourceMissingOrChangedCrashHook(t *testing.T) {
	root := runtimeSourceTestRoot(t)
	path := filepath.Join(runtime.GOROOT(), "src", "runtime", "runtime.go")
	for _, src := range []string{
		"package runtime",
		"package runtime\nfunc setCrashFD(fd int) int { return fd }",
		"package runtime\nfunc setCrashFD(fd uintptr) uintptr",
		"package runtime\nfunc setCrashFD(",
	} {
		if _, err := runtimeSourceOverlay(map[string][]byte{path: []byte(src)}, root, runtime.GOROOT(), runtime.Version()); err == nil {
			t.Fatalf("accepted missing or changed crash hook: %s", src)
		}
	}
	if _, err := runtimeSourceOverlay(nil, root, t.TempDir(), runtime.Version()); err == nil {
		t.Fatal("accepted a GOROOT with missing runtime sources")
	}
}

// Qualification can supply additional pinned source trees without compiling
// this Go 1.27 compiler itself with the older tools.
func TestRuntimeSourceToolchains(t *testing.T) {
	roots := filepath.SplitList(os.Getenv("LLGO_REUSE_TEST_GOROOTS"))
	if len(roots) == 0 {
		roots = []string{runtime.GOROOT()}
	}
	for _, goroot := range roots {
		version, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
		if err != nil {
			t.Fatal(err)
		}
		goVersion := strings.Fields(string(version))[0]
		t.Run(goVersion, func(t *testing.T) {
			root := runtimeSourceTestRoot(t)
			overlay, err := runtimeSourceOverlay(nil, root, goroot, goVersion)
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			var files []*ast.File
			for path, data := range overlay {
				if filepath.Dir(path) != filepath.Join(root, "internal", "runtime") {
					continue
				}
				file, err := parser.ParseFile(fset, path, data, 0)
				if err != nil {
					t.Fatal(err)
				}
				files = append(files, file)
			}
			config := types.Config{Importer: importer.Default()}
			if _, err := config.Check("runtime", fset, files, nil); err != nil {
				t.Fatal(err)
			}
			for _, arch := range []string{"386", "amd64"} {
				abi, err := auditRuntimeABI(env.LLGoRuntimeDir(), goroot, goVersion, &llssa.Target{GOOS: "linux", GOARCH: arch})
				if err != nil {
					t.Fatal(err)
				}
				wantGo, wantLLGo, pointer := uint64(48), uint64(72), uint64(8)
				if arch == "386" {
					wantGo, wantLLGo, pointer = 32, 40, 4
				}
				if abi.GoType.Size != wantGo || abi.LLGoType.Size != wantLLGo || abi.GoFuncValueBytes != pointer || abi.LLGoFuncValueBytes != 2*pointer || abi.GoRuntimeASMReady {
					t.Fatalf("%s: unreviewed runtime ABI: %+v", arch, abi)
				}
			}
		})
	}
}
