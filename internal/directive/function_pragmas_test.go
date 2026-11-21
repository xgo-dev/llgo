package directive

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func pragmaPackage(t *testing.T, source string) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "pragmas.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(&types.Config{}, fset,
		types.NewPackage("example.com/pragmas", "p"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestReadFunctionPragmas(t *testing.T) {
	for _, test := range []struct {
		name string
		doc  string
		want FunctionPragmas
	}{
		{name: "no doc"},
		{name: "ordinary doc", doc: "// F does nothing."},
		{name: "noinline", doc: "//go:noinline", want: FunctionPragmas{NoInline: true}},
		{name: "nosplit", doc: "//go:nosplit", want: FunctionPragmas{NoSplit: true}},
		{name: "uintptrescapes", doc: "//go:uintptrescapes", want: FunctionPragmas{UintptrEscapes: true}},
		{
			name: "mixed and repeated",
			doc:  "//go:noinline\n// ordinary\n//go:nosplit\n//go:uintptrescapes\n//go:noinline",
			want: FunctionPragmas{NoInline: true, NoSplit: true, UintptrEscapes: true},
		},
		{
			// Preserve ParseGroup's existing whitespace and argument handling.
			name: "normalized spelling",
			doc:  "//go: noinline\n//go:nosplit\tignored\n//go:uintptrescapes ignored",
			want: FunctionPragmas{NoInline: true, NoSplit: true, UintptrEscapes: true},
		},
		{name: "other spellings", doc: "//llgo:noinline\n// go:nosplit\n//go:uintptrescapes_extra"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg := pragmaPackage(t, "package p\n"+test.doc+"\nfunc F() {}")
			if got := ReadFunctionPragmas(pkg.Func("F")); got != test.want {
				t.Fatalf("pragmas = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestReadFunctionPragmasUsesOwnCurrentDeclaration(t *testing.T) {
	pkg := pragmaPackage(t, `package p
//go:noinline
//go:nosplit
//go:uintptrescapes
func F() func() { return func() {} }
func Caller() { F() }
`)
	fn := pkg.Func("F")
	if got := ReadFunctionPragmas(fn); got != (FunctionPragmas{true, true, true}) {
		t.Fatalf("declared pragmas = %+v", got)
	}
	for name, function := range map[string]*ssa.Function{
		"nil":       nil,
		"synthetic": pkg.Prog.NewFunction("synthetic", fn.Signature, "test"),
		"closure":   fn.AnonFuncs[0],
		"caller":    pkg.Func("Caller"),
	} {
		if got := ReadFunctionPragmas(function); got != (FunctionPragmas{}) {
			t.Errorf("%s inherited declaration pragmas: %+v", name, got)
		}
	}
	// This extraction deliberately preserves read-time behavior. Preparation
	// and immutable metadata are separate changes.
	fn.Syntax().(*ast.FuncDecl).Doc = nil
	if got := ReadFunctionPragmas(fn); got != (FunctionPragmas{}) {
		t.Fatalf("read cached attributes after removing the doc: %+v", got)
	}
}
