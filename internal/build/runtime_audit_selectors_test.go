package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRuntimeAuditBoundedSelectors(t *testing.T) {
	for _, tc := range []struct {
		name, expression string
		missing, dynamic int
	}{
		{"all-candidates", `RuntimeSymbol(name, "Present", "Future")`, 1, 0},
		{"known", `RuntimeSymbol(name, "Present")`, 0, 0},
		{"unbounded", `name`, 0, 1},
		{"nonliteral-candidate", `RuntimeSymbol(name, other)`, 0, 1},
		{"empty-candidate", `RuntimeSymbol(name, "")`, 0, 1},
		{"duplicate-candidate", `RuntimeSymbol(name, "Present", "Present")`, 0, 1},
		{"spread", `RuntimeSymbol(name, candidates...)`, 0, 1},
		{"unrelated-helper", `other.RuntimeSymbol(name, "Present")`, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := runtimeAuditFixture(t)
			for path, src := range map[string]string{
				"runtime/internal/runtime/base.go": "package runtime\nfunc Present() {}",
				// Include the real declaration shape so ParseFile resolves the
				// unqualified call to its package-level function object.
				"ssa/use.go": "package ssa\nfunc RuntimeSymbol(name string, candidates ...string) string { return name }\nfunc use() { p.rtFunc(" + tc.expression + ") }",
				"cl/use.go":  "package cl",
			} {
				path = filepath.Join(filepath.Dir(a.runtimeDir), path)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(src), 0644); err != nil {
					t.Fatal(err)
				}
			}
			name := "github.com/xgo-dev/llgo/runtime/internal/runtime.Present"
			a.definitions[name] = "fixture"
			report := RuntimeOwnership{Symbols: []RuntimeSymbol{{Name: name, Kind: "func", Body: true}}}
			if err := a.compilerUses(&report); err != nil {
				t.Fatal(err)
			}
			a.resolve(&report)
			if len(report.Missing) != tc.missing || len(report.DynamicCompilerUses) != tc.dynamic {
				t.Fatalf("missing: %+v; unbounded: %+v", report.Missing, report.DynamicCompilerUses)
			}
		})
	}
}

func TestRuntimeAuditQualifiedSelector(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "cl.go", `package cl
import low "github.com/xgo-dev/llgo/ssa"
var choice = low.RuntimeSymbol(name, "One", "Two")
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	expr := file.Decls[1].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0]
	names, bounded := runtimeAuditCandidates(file, expr)
	if !bounded || !reflect.DeepEqual(names, []string{"One", "Two"}) {
		t.Fatalf("qualified selector: %v, %v", names, bounded)
	}
}

func TestRuntimeAuditForwarderMustBeTransparent(t *testing.T) {
	for _, tc := range []struct {
		method, body string
		transparent  bool
	}{
		{"RuntimeFunc", `return p.rtFunc(name)`, true},
		{"RuntimeFunc", `name = change(name); return p.rtFunc(name)`, false},
		{"RuntimeFunc", `return p.rtFunc(change(name))`, false},
		{"RuntimeFunc", `p.rtFunc(name)`, false},
		{"RuntimeFunc", `return cached`, false},
		{"RuntimeFunc", `return rtFunc(name)`, false},
		{"RuntimeFunc", `return p.rtFunc(name, extra)`, false},
		{"RuntimeFunc", `return other.rtFunc(name)`, false},
		{"rtType", `return p.rawType(p.rtNamed(name))`, true},
		{"rtType", `return p.changedType(p.rtNamed(name))`, false},
		{"rtType", `return p.rawType(name)`, false},
	} {
		receiver := "Package"
		if tc.method == "rtType" {
			receiver = "Program"
		}
		file, err := parser.ParseFile(token.NewFileSet(), "package.go",
			"package ssa\nfunc (p "+receiver+") "+tc.method+"(name string) Expr {"+tc.body+"}", 0)
		if err != nil {
			t.Fatal(err)
		}
		got := runtimeAuditForwarders("ssa", "package.go", file)
		if (len(got) == 1) != tc.transparent {
			t.Fatalf("incorrect forwarding classification for %s %s: %v", tc.method, tc.body, got)
		}
	}
}

func TestRuntimeAuditRejectsShadowedSelector(t *testing.T) {
	for _, src := range []string{
		`package ssa; func use() { RuntimeSymbol := other; _ = RuntimeSymbol(name, "Known") }`,
		`package ssa; func RuntimeSymbol(name string, candidates ...string) string { return name }; func use() { RuntimeSymbol := other; _ = RuntimeSymbol(name, "Known") }`,
		`package ssa; var RuntimeSymbol = other; func use() { _ = RuntimeSymbol(name, "Known") }`,
		`package ssa; func use(RuntimeSymbol func(string, ...string) string) { _ = RuntimeSymbol(name, "Known") }`,
		`package ssa; type T struct{}; func (RuntimeSymbol T) use() { _ = RuntimeSymbol(name, "Known") }`,
		`package cl; import low "github.com/xgo-dev/llgo/ssa"; func use() { low := other; _ = low.RuntimeSymbol(name, "Known") }`,
	} {
		file, err := parser.ParseFile(token.NewFileSet(), "shadow.go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		calls := 0
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				calls++
				if names, bounded := runtimeAuditCandidates(file, call); bounded || len(names) != 0 {
					t.Errorf("shadowed selector incorrectly qualified: %v", names)
				}
			}
			return true
		})
		if calls != 1 {
			t.Fatalf("fixture calls = %d", calls)
		}
	}
}
