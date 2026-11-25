package locality

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestKindEncodingAndNames(t *testing.T) {
	tests := []struct {
		kind Kind
		name string
	}{
		{None, ""},
		{Thread, "tls"},
		{Goroutine, "gls"},
	}
	for _, test := range tests {
		if got := test.kind.String(); got != test.name {
			t.Fatalf("Kind(%d).String() = %q, want %q", test.kind, got, test.name)
		}
		if got, ok := Parse(test.name); !ok || got != test.kind {
			t.Fatalf("Parse(%q) = %v, %v", test.name, got, ok)
		}
	}
	if got := Kind(99).String(); got != "invalid:99" {
		t.Fatalf("invalid locality name = %q", got)
	}
	if _, ok := Parse("invalid:99"); ok {
		t.Fatal("Parse accepted invalid locality")
	}
	if Directive(Thread) != ThreadDirective || Directive(Goroutine) != GoroutineDirective {
		t.Fatal("unexpected locality directive names")
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		a, b Kind
		want Kind
		ok   bool
	}{
		{want: None, ok: true},
		{a: Thread, want: Thread, ok: true},
		{b: Goroutine, want: Goroutine, ok: true},
		{a: Thread, b: Thread, want: Thread, ok: true},
		{a: Thread, b: Goroutine, want: None},
	}
	for _, test := range tests {
		got, ok := Merge(test.a, test.b)
		if got != test.want || ok != test.ok {
			t.Fatalf("Merge(%v, %v) = %v, %v", test.a, test.b, got, ok)
		}
	}
}

func TestScanPackageVar(t *testing.T) {
	fset, file := parseFile(t, `package p

//llgointernal:tls
var (
	first int
	//llgointernal:gls
	second int
)
`)
	decl := file.Decls[0].(*ast.GenDecl)
	if _, err := ScanPackageVar(fset, decl); err == nil || !strings.Contains(err.Error(), "cannot apply to the same variable declaration") {
		t.Fatalf("ScanPackageVar conflict error = %v", err)
	}

	fset, file = parseFile(t, `package p

var (
	//llgointernal:tls
	first, second = 1, 2
)
`)
	vars, err := ScanPackageVar(fset, file.Decls[0].(*ast.GenDecl))
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 || vars[0].Info.Locality != Thread || !vars[0].Info.HasInitializer || vars[1].Name != "second" {
		t.Fatalf("ScanPackageVar = %+v", vars)
	}
}

func TestScanPackageVarBranches(t *testing.T) {
	comment := func(text string) *ast.CommentGroup {
		doc := &ast.CommentGroup{}
		for _, line := range strings.Split(text, "\n") {
			doc.List = append(doc.List, &ast.Comment{Text: line})
		}
		return doc
	}
	tests := []struct {
		name string
		decl *ast.GenDecl
		want string
	}{
		{
			name: "declaration error",
			decl: &ast.GenDecl{Tok: token.VAR, Doc: comment("//llgointernal:tls extra")},
			want: "does not accept arguments",
		},
		{
			name: "spec error",
			decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Doc: comment("//llgointernal:gls extra")}}},
			want: "does not accept arguments",
		},
		{
			name: "embed conflict",
			decl: &ast.GenDecl{Tok: token.VAR, Doc: comment("//llgointernal:tls\n//go:embed value.txt"), Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("Value")}}}},
			want: "//go:embed",
		},
		{
			name: "blank name",
			decl: &ast.GenDecl{Tok: token.VAR, Doc: comment("//llgointernal:tls"), Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("_")}}}},
			want: "blank identifier",
		},
		{
			name: "exported name",
			decl: &ast.GenDecl{Tok: token.VAR, Doc: comment("//llgointernal:gls"), Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("Value")}}}},
			want: "requires an unexported package variable",
		},
		{
			name: "linkname",
			decl: &ast.GenDecl{Tok: token.VAR, Doc: comment("//go:linkname value example.com/p.value\n//llgointernal:tls"), Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("value")}}}},
			want: "cannot apply to a //go:linkname variable",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ScanPackageVar(nil, test.decl); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ScanPackageVar error = %v, want %q", err, test.want)
			}
		})
	}

	ordinary := &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
		&ast.ImportSpec{},
		&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("Value")}},
	}}
	if vars, err := ScanPackageVar(nil, ordinary); err != nil || len(vars) != 0 {
		t.Fatalf("ordinary ScanPackageVar = %+v, %v", vars, err)
	}
}

func TestDirectivePlacementDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "grouped import spec",
			src: `package p
import (
	//llgointernal:tls
	"unsafe"
)
var _ = unsafe.Sizeof(0)
`,
			want: "package-level var",
		},
		{
			name: "local var",
			src: `package p
func f() {
	//llgointernal:gls
	var value int
	_ = value
}
`,
			want: "package-level var",
		},
		{
			name: "nested function local",
			src: `package p
func f() {
	_ = func() {
		//llgointernal:tls extra
		var value int
		_ = value
	}
}
`,
			want: "does not accept arguments",
		},
		{
			name: "nested function in local initializer",
			src: `package p
func f() {
	var nested = func() {
		//llgointernal:gls
		var value int
		_ = value
	}
	_ = nested
}
`,
			want: "package-level var",
		},
		{
			name: "function",
			src: `package p
//llgointernal:tls
func f() {}
`,
			want: "package-level var",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fset, file := parseFile(t, test.src)
			err := validateFile(fset, file)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestDirectiveDiagnostics(t *testing.T) {
	tests := []struct {
		comment string
		want    string
	}{
		{"//llgo:threadlocal", "use //llgointernal:tls"},
		{"//llgo:goroutinelocal", "use //llgointernal:gls"},
		{"//llgointernal:tls extra", "does not accept arguments"},
		{"//llgointernal:tls\n//llgointernal:gls", "cannot apply to the same variable declaration"},
	}
	for _, test := range tests {
		doc := &ast.CommentGroup{}
		for _, line := range strings.Split(test.comment, "\n") {
			doc.List = append(doc.List, &ast.Comment{Text: line})
		}
		if _, _, err := FromDoc(nil, doc); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("FromDoc(%q) error = %v", test.comment, err)
		}
	}
	if err := ValidateDoc(nil, nil); err != nil {
		t.Fatal(err)
	}
	if kind, _, err := FromDoc(nil, &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}); err != nil || kind != None {
		t.Fatalf("ordinary directive = %v, %v", kind, err)
	}
	if err := ValidateFuncBody(nil, nil); err != nil {
		t.Fatal(err)
	}
	typeDecl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{
		&ast.TypeSpec{Name: ast.NewIdent("T"), Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//llgointernal:tls"}}}},
	}}
	if err := ValidateNonPackageVar(nil, typeDecl); err == nil || !strings.Contains(err.Error(), "package-level var") {
		t.Fatalf("type spec validation error = %v", err)
	}
	if hasDirective(&ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}, "go:embed") {
		t.Fatal("hasDirective matched an unrelated directive")
	}
}

func TestInitializerLocalityDiagnostics(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	thread := types.NewVar(token.NoPos, pkg, "thread", types.Typ[types.Int])
	goroutine := types.NewVar(token.NoPos, pkg, "goroutine", types.Typ[types.Int])
	ordinary := types.NewVar(token.NoPos, pkg, "ordinary", types.Typ[types.Int])
	vars := map[string]Info{
		"thread":    {Locality: Thread, HasInitializer: true},
		"goroutine": {Locality: Goroutine, HasInitializer: true},
	}
	rhs := ast.NewIdent("rhs")
	tests := []struct {
		lhs  []*types.Var
		want string
	}{
		{[]*types.Var{thread, ordinary}, "mix local and ordinary"},
		{[]*types.Var{ordinary, thread}, "mix local and ordinary"},
		{[]*types.Var{thread, goroutine}, "mix thread-local and goroutine-local"},
	}
	for _, test := range tests {
		if _, _, err := initializerLocality(nil, &types.Initializer{Lhs: test.lhs, Rhs: rhs}, vars); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("initializerLocality error = %v, want %q", err, test.want)
		}
	}
	if kind, found, err := initializerLocality(nil, &types.Initializer{Lhs: []*types.Var{ordinary}, Rhs: rhs}, vars); err != nil || found || kind != None {
		t.Fatalf("ordinary initializer = %v, %v, %v", kind, found, err)
	}
}

func TestPrepareIsIdempotentAcrossPrograms(t *testing.T) {
	fset, file := parseFile(t, `package p
func makeValue() *int { value := 42; return &value }
//llgointernal:tls
var value = makeValue()
`)
	files := []*ast.File{file}
	info := newTypeInfo()
	pkg, err := (&types.Config{}).Check("example.com/p", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	vars, err := ScanPackageVar(fset, file.Decls[1].(*ast.GenDecl))
	if err != nil {
		t.Fatal(err)
	}
	raw := make(map[string]Info)
	for _, variable := range vars {
		raw[variable.Name] = variable.Info
	}

	prepared, err := Prepare(fset, pkg.Path(), pkg, info, files, raw)
	if err != nil {
		t.Fatal(err)
	}
	declCount := len(file.Decls)
	scopeCount := len(pkg.Scope().Names())
	initName := prepared["value"].InitFunc
	if initName != "example.com/p.__llgo_local_init_0" || prepared["value"].InitOrder != 1 {
		t.Fatalf("prepared metadata = %+v", prepared["value"])
	}

	again, err := Prepare(fset, pkg.Path(), pkg, info, files, prepared)
	if err != nil {
		t.Fatal(err)
	}
	reused, err := Prepare(fset, pkg.Path(), pkg, info, files, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Decls) != declCount || len(pkg.Scope().Names()) != scopeCount {
		t.Fatalf("repeated Prepare changed syntax/scope: decls=%d/%d scope=%d/%d", len(file.Decls), declCount, len(pkg.Scope().Names()), scopeCount)
	}
	if again["value"] != prepared["value"] || reused["value"] != prepared["value"] {
		t.Fatalf("repeated metadata = %+v, reused = %+v, want %+v", again["value"], reused["value"], prepared["value"])
	}
}

func TestPrepareZeroValuePointerAndValidation(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	value := types.NewVar(token.NoPos, pkg, "Value", types.NewPointer(types.Typ[types.Int]))
	pkg.Scope().Insert(value)
	vars := map[string]Info{"Value": {Locality: Goroutine}}
	prepared, err := Prepare(nil, pkg.Path(), pkg, &types.Info{}, nil, vars)
	if err != nil {
		t.Fatal(err)
	}
	if got := prepared["Value"]; got.InitFunc != "" || got.InitOrder != 0 {
		t.Fatalf("zero-value initializer = %+v", got)
	}
	if err := ValidatePrepared(pkg.Path(), map[string]Info{"Value": {Locality: Thread, HasInitializer: true}}); err == nil {
		t.Fatal("ValidatePrepared accepted missing initializer metadata")
	}
	if err := ValidatePrepared(pkg.Path(), map[string]Info{"Value": {Locality: Thread, InitFunc: "p.init", InitOrder: 1}}); err == nil {
		t.Fatal("ValidatePrepared accepted initializer metadata on a zero-value declaration")
	}
	if err := ValidatePrepared(pkg.Path(), map[string]Info{
		"Ordinary": {},
		"Value":    {Locality: Thread, HasInitializer: true, InitFunc: "p.init", InitOrder: 1},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareEarlyReturnsAndMissingFiles(t *testing.T) {
	if got, err := Prepare(nil, "", nil, nil, nil, nil); err != nil || len(got) != 0 {
		t.Fatalf("nil Prepare = %+v, %v", got, err)
	}
	pkg := types.NewPackage("example.com/p", "p")
	value := types.NewVar(token.NoPos, pkg, "Value", types.Typ[types.Int])
	pkg.Scope().Insert(value)
	info := &types.Info{InitOrder: []*types.Initializer{{Lhs: []*types.Var{value}, Rhs: ast.NewIdent("rhs")}}}
	vars := map[string]Info{"Value": {Locality: Thread, HasInitializer: true}}
	if _, err := Prepare(nil, pkg.Path(), pkg, info, nil, vars); err == nil || !strings.Contains(err.Error(), "without syntax files") {
		t.Fatalf("Prepare without files error = %v", err)
	}
	if got, err := Prepare(nil, pkg.Path(), pkg, &types.Info{}, nil, nil); err != nil || len(got) != 0 {
		t.Fatalf("Prepare without localities = %+v, %v", got, err)
	}
}

func TestPrepareInitializerBranches(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	local := types.NewVar(token.NoPos, pkg, "Local", types.Typ[types.Int])
	ordinary := types.NewVar(token.NoPos, pkg, "Ordinary", types.Typ[types.Int])
	pkg.Scope().Insert(local)
	pkg.Scope().Insert(ordinary)
	rhs := ast.NewIdent("rhs")
	info := &types.Info{InitOrder: []*types.Initializer{
		{Lhs: []*types.Var{ordinary}, Rhs: ast.NewIdent("ordinary")},
		{Lhs: []*types.Var{local, ordinary}, Rhs: rhs},
	}}
	vars := map[string]Info{"Local": {Locality: Thread, HasInitializer: true}}
	if _, err := Prepare(nil, pkg.Path(), pkg, info, []*ast.File{{}}, vars); err == nil || !strings.Contains(err.Error(), "mix local and ordinary") {
		t.Fatalf("mixed Prepare error = %v", err)
	}

	info.InitOrder = []*types.Initializer{{Lhs: []*types.Var{local}, Rhs: rhs}}
	prepared, err := Prepare(nil, pkg.Path(), pkg, info, []*ast.File{{}}, vars)
	if err != nil {
		t.Fatal(err)
	}
	if prepared["Local"].InitFunc == "" || info.Uses == nil || info.Defs == nil {
		t.Fatalf("Prepare did not initialize type maps: %+v", prepared["Local"])
	}

	conflicting := &types.Initializer{Lhs: []*types.Var{local, ordinary}, Rhs: rhs}
	if got := preparedInitName(conflicting, map[string]Info{
		"Local":    {InitFunc: "p.first", InitOrder: 1},
		"Ordinary": {InitFunc: "p.second", InitOrder: 1},
	}, 1); got != "" {
		t.Fatalf("preparedInitName accepted conflicting helpers: %q", got)
	}
	if got := qualify("", "Value"); got != "Value" {
		t.Fatalf("qualify empty package = %q", got)
	}
}

func TestFindLocalInitializerRejectsLookalikes(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	value := types.NewVar(token.NoPos, pkg, "Value", types.Typ[types.Int])
	rhs := ast.NewIdent("rhs")
	initializer := &types.Initializer{Lhs: []*types.Var{value}, Rhs: rhs}
	files := []*ast.File{{Decls: []ast.Decl{
		&ast.FuncDecl{
			Name: ast.NewIdent(InitPrefix + "0"),
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: rhs}}},
		},
		&ast.FuncDecl{
			Name: ast.NewIdent(InitPrefix + "1"),
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{rhs},
			}}},
		},
	}}}
	if got := findLocalInitializer(pkg, &types.Info{}, files, initializer); got != "" {
		t.Fatalf("findLocalInitializer accepted lookalike %q", got)
	}
}

func parseFile(t *testing.T, src string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return fset, file
}

func validateFile(fset *token.FileSet, file *ast.File) error {
	for _, node := range file.Decls {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			if err := ValidateDoc(fset, decl.Doc); err != nil {
				return err
			}
			if err := ValidateFuncBody(fset, decl.Body); err != nil {
				return err
			}
		case *ast.GenDecl:
			if decl.Tok == token.VAR {
				if _, err := ScanPackageVar(fset, decl); err != nil {
					return err
				}
			} else if err := ValidateNonPackageVar(fset, decl); err != nil {
				return err
			}
		}
	}
	return nil
}

func newTypeInfo() *types.Info {
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
}
