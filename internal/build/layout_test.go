package build

import (
	llssa "github.com/xgo-dev/llgo/ssa"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLayoutAgainstCPP(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Itanium empty-subobject reference; MSVC uses a different attribute")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "layout")
	_, err = Build(Invocation{Args: []string{"."}, Config: conf, Dir: filepath.Join(root, "internal/build/testdata/layoutcpp")})
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(conf.OutFile).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "layout cpp ok" {
		t.Fatalf("C++ layout/ABI comparison: %v\n%s", err, out)
	}
}

func TestPackedLayoutParticipatesInFingerprint(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	ctx := &context{buildConf: &Config{Goos: "linux", Goarch: "amd64"}, prog: prog}
	plain := newManifestBuilder()
	ctx.collectCommonInputs(plain)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "packed.go", "package p\n//llgo:packed\ntype T struct { X uint64 }", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if err := prog.ParseLayoutSyntax(fset, types.NewPackage("p", "p"), []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	packed := newManifestBuilder()
	ctx.collectCommonInputs(packed)
	if plain.common.PackedLayout || !packed.common.PackedLayout || plain.Fingerprint() == packed.Fingerprint() {
		t.Fatal("packed field pointers could reach cached code with natural-alignment assumptions")
	}
}
