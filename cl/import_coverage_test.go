//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestReplaceGoNameRuntimeBranch(t *testing.T) {
	const sym = "runtime.memmove"
	got := replaceGoName(sym, len("runtime"))
	want := env.LLGoRuntimePkg + "/internal/runtime.memmove"
	if got != want {
		t.Fatalf("replaceGoName(%q)=%q, want %q", sym, got, want)
	}
}

func TestTypeBackgroundAndParsePkgSyntaxCoverage(t *testing.T) {
	if got := typeBackground(nil); got != "" {
		t.Fatalf("typeBackground(nil)=%q, want empty", got)
	}

	doc1 := &ast.CommentGroup{List: []*ast.Comment{{Text: "//llgo:type C"}}}
	if got := typeBackground(doc1); got != "C" {
		t.Fatalf("typeBackground(//llgo:type C)=%q, want C", got)
	}
	doc2 := &ast.CommentGroup{List: []*ast.Comment{{Text: "// llgo:type C"}}}
	if got := typeBackground(doc2); got != "C" {
		t.Fatalf("typeBackground(// llgo:type C)=%q, want C", got)
	}

	src := `package p
//llgo:type C
type A int
type (
	B int
	C int
)

//go:nointerface
func (A) Hidden() {}

//go:other
//go:nointerface
func (A) StackedHidden() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	prog := llssa.NewProgram(nil)
	pkg := types.NewPackage("example.com/p", "p")
	if err := ParsePkgSyntax(prog, fset, pkg, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}

	ctx := &context{prog: prog}
	ctx.processNoInterfaceByDoc(nil, "example.com/p.NilDoc")
	ctx.processNoInterfaceByDoc(&ast.CommentGroup{List: []*ast.Comment{
		{Text: "// not a directive"},
		{Text: "//go:nointerface"},
	}}, "example.com/p.NonDirectiveStops")

	if !prog.PackageSyntaxParsed(pkg) {
		t.Fatal("package syntax was not marked as parsed")
	}
	badFile, err := parser.ParseFile(fset, "bad.go", "package p\n//llgo:tls\nfunc Bad() {}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if err := ParsePkgSyntax(prog, fset, pkg, []*ast.File{badFile}); err != nil {
		t.Fatalf("already parsed package was scanned again: %v", err)
	}
}

func TestParsePkgSyntaxReportsLocalityErrors(t *testing.T) {
	prog := llssa.NewProgram(nil)
	if err := ParsePkgSyntax(prog, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "function body",
			src:  "package p\nfunc f() {\n//llgo:gls\nvar value int\n_ = value\n}\n",
			want: "package-level var",
		},
		{
			name: "package var",
			src:  "package p\n//llgo:tls extra\nvar Value int\n",
			want: "does not accept arguments",
		},
		{
			name: "non-var declaration",
			src:  "package p\n//llgo:gls\nconst Value = 1\n",
			want: "package-level var",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", test.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := types.NewPackage("example.com/"+strings.ReplaceAll(test.name, " ", "-"), "p")
			err = ParsePkgSyntax(llssa.NewProgram(nil), fset, pkg, []*ast.File{file})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParsePkgSyntax error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPkgSymInfoAddSymAndInitLinknamesCoverage(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "p.go")
	src := "package p\n\n//go:linkname Foo c_foo\nfunc Foo()\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	var fnPos token.Pos
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "Foo" {
			fnPos = fn.Name.Pos()
			break
		}
	}
	if fnPos == token.NoPos {
		t.Fatalf("failed to find Foo position")
	}

	syms := newPkgSymInfo()
	syms.addSym(fset, fnPos, "example.com/p.Foo", "Foo", false)

	tf := fset.File(file.Pos())
	syms.addSym(fset, tf.LineStart(2), "example.com/p.Skip", "Skip", false)
	if _, ok := syms.syms["Skip"]; ok {
		t.Fatalf("symbol with line<=2 should be skipped")
	}

	// cover os.ReadFile error branch inside addSym.
	ff := fset.AddFile(filepath.Join(dir, "missing.go"), -1, 100)
	ff.SetLines([]int{0, 10, 20, 30, 40, 50})
	syms.addSym(fset, ff.LineStart(4), "example.com/p.Miss", "Miss", false)
	if _, ok := syms.syms["Miss"]; !ok {
		t.Fatalf("symbol should still be recorded when file read fails")
	}

	prog := llssa.NewProgram(nil)
	ctx := &context{prog: prog}
	syms.initLinknames(ctx)
	if got, ok := prog.Linkname("example.com/p.Foo"); !ok || got != "c_foo" {
		t.Fatalf("linkname = (%q,%v), want (%q,%v)", got, ok, "c_foo", true)
	}
}

func TestAstAndTypesFuncNameCoverage(t *testing.T) {
	full, inPkg := astFuncName("example.com/p", &ast.FuncDecl{Name: &ast.Ident{Name: "F"}})
	if full != "example.com/p.F" || inPkg != "F" {
		t.Fatalf("astFuncName(func)=(%q,%q), want (%q,%q)", full, inPkg, "example.com/p.F", "F")
	}

	ptrRecv := &ast.FuncDecl{
		Name: &ast.Ident{Name: "M"},
		Recv: &ast.FieldList{List: []*ast.Field{
			{Type: &ast.StarExpr{X: &ast.ParenExpr{X: &ast.Ident{Name: "T"}}}},
		}},
	}
	full, inPkg = astFuncName("example.com/p", ptrRecv)
	if full != "example.com/p.(*T).M" || inPkg != "(*T).M" {
		t.Fatalf("astFuncName(method ptr)=(%q,%q), want (%q,%q)", full, inPkg, "example.com/p.(*T).M", "(*T).M")
	}

	pkg := types.NewPackage("example.com/p", "p")
	tObj := types.NewTypeName(token.NoPos, pkg, "T", nil)
	named := types.NewNamed(tObj, types.NewStruct(nil, nil), nil)

	methodPtr := types.NewFunc(token.NoPos, pkg, "M", types.NewSignature(
		types.NewVar(token.NoPos, pkg, "", types.NewPointer(named)), nil, nil, false,
	))
	full, inPkg = typesFuncName(pkg.Path(), methodPtr)
	if full != "example.com/p.(*T).M" || inPkg != "(*T).M" {
		t.Fatalf("typesFuncName(method ptr)=(%q,%q), want (%q,%q)", full, inPkg, "example.com/p.(*T).M", "(*T).M")
	}

	methodVal := types.NewFunc(token.NoPos, pkg, "N", types.NewSignature(
		types.NewVar(token.NoPos, pkg, "", named), nil, nil, false,
	))
	full, inPkg = typesFuncName(pkg.Path(), methodVal)
	if full != "example.com/p.T.N" || inPkg != "T.N" {
		t.Fatalf("typesFuncName(method val)=(%q,%q), want (%q,%q)", full, inPkg, "example.com/p.T.N", "T.N")
	}

	fn := types.NewFunc(token.NoPos, pkg, "Top", types.NewSignature(nil, nil, nil, false))
	full, inPkg = typesFuncName(pkg.Path(), fn)
	if full != "example.com/p.Top" || inPkg != "Top" {
		t.Fatalf("typesFuncName(func)=(%q,%q), want (%q,%q)", full, inPkg, "example.com/p.Top", "Top")
	}
}

func TestParsePkgSyntaxCollectsLinknames(t *testing.T) {
	cases := []struct {
		name      string
		directive string
		want      string
	}{
		{name: "go-linkname", directive: "//go:linkname Sigsetjmp C.sigsetjmp\nfunc Sigsetjmp()\n", want: "C.sigsetjmp"},
		{name: "go-linkname-after", directive: "func Sigsetjmp()\n//go:linkname Sigsetjmp C.sigsetjmp\n", want: "C.sigsetjmp"},
		{name: "llgo-linkname", directive: "//llgo:link Sigsetjmp C.sigsetjmp\nfunc Sigsetjmp()\n", want: "C.sigsetjmp"},
		{name: "llgo-linkname-spaced", directive: "// llgo:link Sigsetjmp C.sigsetjmp\nfunc Sigsetjmp()\n", want: "C.sigsetjmp"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			src := "package runtime\nimport _ \"unsafe\"\n" + tt.directive
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "runtime.go", src, parser.ParseComments)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			prog := llssa.NewProgram(nil)
			pkg := types.NewPackage(llssa.PkgRuntime, "runtime")
			if err := ParsePkgSyntax(prog, fset, pkg, []*ast.File{file}); err != nil {
				t.Fatal(err)
			}
			if got, ok := prog.Linkname(llssa.PkgRuntime + ".Sigsetjmp"); !ok || got != tt.want {
				t.Fatalf("pre-collected linkname = (%q,%v), want (%q,%v)", got, ok, tt.want, true)
			}
		})
	}
	prog := llssa.NewProgram(nil)
	collectDeclarationDirectives(prog, nil, &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:linkname Other C.other"}}}, llssa.PkgRuntime+".Sigsetjmp", "Sigsetjmp", token.NoPos)
	if _, ok := prog.Linkname(llssa.PkgRuntime + ".Sigsetjmp"); ok {
		t.Fatal("mismatched linkname was collected")
	}

	crossSrc := []string{
		"package p\nimport _ \"unsafe\"\nfunc main() {}\n//go:linkname alias C.alias\n",
		"package p\nfunc alias()\n",
	}
	crossFset := token.NewFileSet()
	var crossFiles []*ast.File
	for i, src := range crossSrc {
		file, err := parser.ParseFile(crossFset, "cross"+string(rune('0'+i))+".go", src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		crossFiles = append(crossFiles, file)
	}
	prog = llssa.NewProgram(nil)
	crossPkg := types.NewPackage("example.com/p", "p")
	crossPkg.Scope().Insert(types.NewFunc(token.NoPos, crossPkg, "alias", types.NewSignature(nil, nil, nil, false)))
	if err := ParsePkgSyntax(prog, crossFset, crossPkg, crossFiles); err != nil {
		t.Fatal(err)
	}
	if got, ok := prog.Linkname("example.com/p.alias"); !ok || got != "C.alias" {
		t.Fatalf("cross-file linkname = (%q,%v), want (C.alias,true)", got, ok)
	}
}

func TestParsePkgSyntaxCollectsClosureEnvDirectives(t *testing.T) {
	const src = `package p
//go:linkname env C.old
//llgo:env
//go:linkname env C.new
func env() {}

// llgo:env
func spaced() {}

func plain() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	prog := llssa.NewProgram(nil)
	pkg := types.NewPackage("example.com/p", "p")
	if err := ParsePkgSyntax(prog, fset, pkg, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	if link, ok := prog.Linkname("example.com/p.env"); !ok || link != "C.new" {
		t.Fatalf("combined declaration linkname = (%q, %v), want (C.new, true)", link, ok)
	}
	want := map[string]bool{"env": true, "spaced": true, "plain": false}
	for _, node := range file.Decls {
		decl := node.(*ast.FuncDecl)
		fullName, _ := astFuncName(pkg.Path(), decl)
		got := prog.HasClosureEnvDirective(fset, fullName, decl.Pos())
		if got != want[decl.Name.Name] {
			t.Fatalf("HasClosureEnvDirective(%s) = %v, want %v", decl.Name.Name, got, want[decl.Name.Name])
		}
	}
	if prog.HasClosureEnvDirective(fset, "example.com/p.missing", token.NoPos) {
		t.Fatal("missing declaration unexpectedly has cached directives")
	}
}

func TestCollectDeclarationDirectivesIgnoresOtherDirectives(t *testing.T) {
	prog := llssa.NewProgram(nil)
	doc := &ast.CommentGroup{List: []*ast.Comment{
		{Text: "//go:noinline"},
		{Text: "//llgo:tls"},
	}}
	const fullName = "example.com/p.Value"
	collectDeclarationDirectives(prog, nil, doc, fullName, "Value", token.NoPos)
	if _, ok := prog.Linkname(fullName); ok {
		t.Fatal("non-link directives installed a linkname")
	}
}

func TestBoolToUint8InvalidArgs(t *testing.T) {
	ctx := &context{}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("boolToUint8 should panic on invalid arguments")
		}
		if msg := r.(string); !strings.Contains(msg, "invalid arguments") {
			t.Fatalf("panic = %q, want invalid arguments", msg)
		}
	}()
	_ = ctx.boolToUint8(nil, []llssa.Expr{})
}
