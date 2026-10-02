package testsyntax

import (
	"errors"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

func dependencySyntaxFixture(t *testing.T) (*ssa.Package, []*ast.File) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPACKAGESDRIVER", "off")
	t.Setenv("GOFLAGS", "")
	for name, source := range map[string]string{
		"go.mod": "module example.com/fixture\n\ngo 1.27.0\n",
		"root.go": `package root
import (_ "example.com/fixture/bridge"; _ "example.com/fixture/linked")
func Root() {}
`,
		"bridge/bridge.go": `package bridge
import (_ "example.com/fixture/linked"; _ "example.com/fixture/transitive")
`,
		"linked/linked.go": `package linked
const LLGoPackage = "decl"
//llgo:link F C.fixture
//go:noinline
func F() {}
`,
		"transitive/transitive.go": `package transitive
const LLGoPackage = "decl"
//llgo:link F C.transitive
//go:noinline
func F() {}
`,
		// Invalid syntax must stay excluded by the package loader's build selection.
		"linked/ignored.go": "//go:build ignore\n\npackage linked\ninvalid source\n",
	} {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax}, ".")
	if err != nil {
		t.Fatal(err)
	}
	packages.Visit(loaded, nil, func(pkg *packages.Package) {
		if len(pkg.Errors) != 0 {
			t.Fatal(pkg.Errors)
		}
	})
	root := loaded[0]
	return ssa.NewProgram(root.Fset, 0).CreatePackage(root.Types, nil, nil, true), root.Syntax
}

func TestPrepareDependencySyntax(t *testing.T) {
	pkg, files := dependencySyntaxFixture(t)
	seen := make(map[string]int)
	err := Prepare(pkg, files, func(dep *types.Package, syntax []*ast.File) error {
		seen[dep.Path()]++
		if dep == pkg.Pkg {
			if len(syntax) != 1 || syntax[0] != files[0] {
				t.Fatal("root syntax was reloaded")
			}
			return nil
		}
		want := map[string]string{
			"example.com/fixture/linked":     "C.fixture",
			"example.com/fixture/transitive": "C.transitive",
		}[dep.Path()]
		if want == "" || len(syntax) != 1 {
			t.Fatalf("unexpected dependency syntax: %s, %d files", dep.Path(), len(syntax))
		}
		records, err := directive.Collect(pkg.Prog.Fset, new(directive.Index).Files(syntax), false, false)
		if err != nil {
			return err
		}
		BindScope(records, pkg.Prog.Fset, dep)
		fn, ok := records.Objects[dep.Scope().Lookup("F")].(*directive.FunctionDecl)
		if !ok || !fn.NoInline || fn.Linkname != want {
			t.Fatalf("imported object lost its source directives: %+v", fn)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 3 || seen[pkg.Pkg.Path()] != 1 || seen["example.com/fixture/linked"] != 1 || seen["example.com/fixture/transitive"] != 1 {
		t.Fatalf("shared transitive dependency must be collected once: %v", seen)
	}
}

func TestPrepareWithoutDependencyLoading(t *testing.T) {
	fset, file := parseSource(t, "package p\nfunc F() {}\n")
	checked, _ := checkRecordSource(t, fset, file)
	// Ordinary export-only imports need no LLGo source discovery.
	checked.SetImports([]*types.Package{types.NewPackage("example.com/ordinary", "ordinary")})
	pkg := ssa.NewProgram(fset, 0).CreatePackage(checked, nil, nil, true)
	t.Setenv("GOPACKAGESDRIVER", filepath.Join(t.TempDir(), "missing-driver"))
	for _, want := range []error{nil, errors.New("root collection failed")} {
		calls := 0
		err := Prepare(pkg, []*ast.File{file}, func(dep *types.Package, syntax []*ast.File) error {
			calls++
			if dep != checked || len(syntax) != 1 || syntax[0] != file {
				t.Fatal("unexpected collection input")
			}
			return want
		})
		if !errors.Is(err, want) || calls != 1 {
			t.Fatalf("Prepare = %v, %d calls; want %v, 1 call", err, calls, want)
		}
	}
}

func TestPrepareDependencyErrors(t *testing.T) {
	for _, scenario := range []string{"collector", "syntax", "driver"} {
		t.Run(scenario, func(t *testing.T) {
			pkg, files := dependencySyntaxFixture(t)
			want := errors.New("dependency collection failed")
			switch scenario {
			case "syntax":
				if err := os.WriteFile("linked/linked.go", []byte("package linked\nfunc"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "driver":
				t.Setenv("GOPACKAGESDRIVER", filepath.Join(t.TempDir(), "missing-driver"))
			}
			calls := 0
			err := Prepare(pkg, files, func(dep *types.Package, _ []*ast.File) error {
				calls++
				if dep != pkg.Pkg {
					return want
				}
				return nil
			})
			switch scenario {
			case "collector":
				if !errors.Is(err, want) || calls != 2 {
					t.Fatalf("dependency error = %v, %d calls", err, calls)
				}
			default:
				name := "linked.go"
				if scenario == "driver" {
					name = "missing-driver"
				}
				if err == nil || !strings.Contains(err.Error(), name) || calls != 1 {
					t.Fatalf("load error = %v, %d calls; want %s error before dependency collection", err, calls, name)
				}
			}
		})
	}
}
