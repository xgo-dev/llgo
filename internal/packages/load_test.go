//go:build !llgo

package packages

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	xpackages "golang.org/x/tools/go/packages"
)

func TestLoadExWithGoVersion(t *testing.T) {
	dir := t.TempDir()
	writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/loadversion\ngo 1.24\n")
	writeLoadTestFile(t, filepath.Join(dir, "load.go"), `package loadversion

func identity[T any](value T) T { return value }
`)

	oldVersion, err := LoadExWithGoVersion(nil, nil, loadTestConfig(dir), "go1.17", ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(oldVersion) != 1 {
		t.Fatalf("LoadExWithGoVersion returned %d packages, want 1", len(oldVersion))
	}
	if !oldVersion[0].IllTyped || !packageErrorsContain(oldVersion[0], "requires go1.18 or later") {
		t.Fatalf("go1.17 load did not reject generics: %+v", oldVersion[0].Errors)
	}

	currentVersion, err := LoadEx(nil, nil, loadTestConfig(dir), ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(currentVersion) != 1 {
		t.Fatalf("LoadEx returned %d packages, want 1", len(currentVersion))
	}
	pkg := currentVersion[0]
	if pkg.IllTyped || len(pkg.Errors) != 0 {
		t.Fatalf("module-version load failed: %+v", pkg.Errors)
	}
	if len(pkg.Syntax) != 1 || pkg.TypesInfo == nil {
		t.Fatalf("load did not return syntax and type information: %+v", pkg)
	}
	if got := pkg.TypesInfo.FileVersions[pkg.Syntax[0]]; got != "go1.24" {
		t.Fatalf("file language version = %q, want go1.24", got)
	}
}

func TestNormalizeEmbedDriverDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		errorLine int
		goVersion string
		want      string
	}{
		{
			name: "local var",
			source: `package p
func f() {
	//go:embed x.txt // ERROR
	var x string
}`,
			errorLine: 3,
			goVersion: "go1.24",
			want:      "go:embed cannot apply to var inside func",
		},
		{
			name: "old language version",
			source: `package p
//go:embed x.txt // ERROR
var x string`,
			errorLine: 2,
			goVersion: "go1.15",
			want:      "go:embed requires go1.16 or later (-lang was set to go1.15; check go.mod)",
		},
		{
			name: "current language version remains driver error",
			source: `package p
//go:embed x.txt // ERROR
var x string`,
			errorLine: 2,
			goVersion: "go1.24",
			want:      embedPatternDriverDiagnostic,
		},
		{
			name: "wrong source line remains driver error",
			source: `package p
func f() {
	//go:embed x.txt // ERROR
	var x string
}`,
			errorLine: 4,
			goVersion: "go1.24",
			want:      embedPatternDriverDiagnostic,
		},
		{
			name: "non-var directive remains driver error",
			source: `package p
//go:embed x.txt // ERROR
const x = ""`,
			errorLine: 2,
			goVersion: "go1.15",
			want:      embedPatternDriverDiagnostic,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			const filename = "/tmp/embedcase.go"
			file, err := parser.ParseFile(fset, filename, tt.source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			errs := []Error{
				{Pos: filename + ":" + strconv.Itoa(tt.errorLine) + ":20", Msg: embedPatternDriverDiagnostic},
				{Pos: filename + ":1:1", Msg: "unrelated driver diagnostic"},
			}
			normalizeEmbedDriverDiagnostics(errs, fset, []*ast.File{file}, tt.goVersion)
			if errs[0].Msg != tt.want {
				t.Fatalf("normalized embed error = %q, want %q", errs[0].Msg, tt.want)
			}
			if errs[1].Msg != "unrelated driver diagnostic" {
				t.Fatalf("unrelated error changed to %q", errs[1].Msg)
			}
		})
	}
}

func TestEmbedDiagnosticHelperBoundaries(t *testing.T) {
	t.Run("directive comments", func(t *testing.T) {
		tests := []struct {
			name    string
			comment *ast.Comment
			want    bool
		}{
			{name: "nil"},
			{name: "block comment", comment: &ast.Comment{Text: "/*go:embed x*/"}},
			{name: "exact directive", comment: &ast.Comment{Text: "//go:embed"}, want: true},
			{name: "different comment", comment: &ast.Comment{Text: "// not a directive"}},
			{name: "lookalike directive", comment: &ast.Comment{Text: "//go:embedx"}},
			{name: "directive argument", comment: &ast.Comment{Text: "//go:embed x"}, want: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := isEmbedDirectiveComment(tt.comment); got != tt.want {
					t.Fatalf("isEmbedDirectiveComment() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("nil syntax context", func(t *testing.T) {
		if got := embedDirectiveContextAt(nil, &ast.File{}, "case.go:1:1"); got != embedDirectiveUnknown {
			t.Fatalf("nil fileset context = %v, want unknown", got)
		}
		if got := embedDirectiveContextAt(token.NewFileSet(), nil, "case.go:1:1"); got != embedDirectiveUnknown {
			t.Fatalf("nil file context = %v, want unknown", got)
		}
	})

	t.Run("diagnostic positions", func(t *testing.T) {
		commentPos := token.Position{Filename: filepath.Join("tmp", "case.go"), Line: 12}
		tests := []struct {
			name     string
			errorPos string
			comment  token.Position
			want     bool
		}{
			{name: "empty error position", comment: commentPos},
			{name: "empty comment position", errorPos: "case.go:12:3"},
			{name: "missing separator", errorPos: "case.go", comment: commentPos},
			{name: "invalid line", errorPos: "case.go:line", comment: commentPos},
			{name: "wrong line", errorPos: "case.go:11:3", comment: commentPos},
			{name: "relative suffix", errorPos: "case.go:12:3", comment: commentPos, want: true},
			{name: "different file", errorPos: "other.go:12:3", comment: commentPos},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := sameDiagnosticLine(tt.errorPos, tt.comment); got != tt.want {
					t.Fatalf("sameDiagnosticLine(%q, %+v) = %v, want %v", tt.errorPos, tt.comment, got, tt.want)
				}
			})
		}
	})

	t.Run("position parsing", func(t *testing.T) {
		tests := []struct {
			position string
			wantFile string
			wantLine int
			wantOK   bool
		}{
			{position: "case.go"},
			{position: "case.go:line"},
			{position: "case.go:12", wantFile: "case.go", wantLine: 12, wantOK: true},
			{position: "case.go:12:3", wantFile: "case.go", wantLine: 12, wantOK: true},
		}
		for _, tt := range tests {
			file, line, ok := diagnosticFileLine(tt.position)
			if file != tt.wantFile || line != tt.wantLine || ok != tt.wantOK {
				t.Fatalf("diagnosticFileLine(%q) = (%q, %d, %v), want (%q, %d, %v)", tt.position, file, line, ok, tt.wantFile, tt.wantLine, tt.wantOK)
			}
		}
	})

	t.Run("declaration comments", func(t *testing.T) {
		directive := &ast.Comment{Text: "//go:embed x"}
		other := &ast.Comment{Text: "// other"}
		matching := &ast.GenDecl{Specs: []ast.Spec{
			&ast.ValueSpec{},
			&ast.ValueSpec{Doc: &ast.CommentGroup{List: []*ast.Comment{nil, other, directive}}},
		}}
		if !genDeclHasDocComment(matching, directive) {
			t.Fatal("value-spec directive was not found")
		}
		missing := &ast.GenDecl{Specs: []ast.Spec{
			&ast.ValueSpec{Doc: &ast.CommentGroup{List: []*ast.Comment{other}}},
		}}
		if genDeclHasDocComment(missing, directive) {
			t.Fatal("unrelated value-spec comment matched directive")
		}
	})
}

func TestLoadExNormalizesFrontendDiagnostics(t *testing.T) {
	t.Run("compiler syntax errors are authoritative", func(t *testing.T) {
		dir := t.TempDir()
		writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/syntaxerror\ngo 1.24\n")
		writeLoadTestFile(t, filepath.Join(dir, "load.go"), `package syntaxerror
func f() {
	if missing {}
	else {}
}
`)
		cfg := loadTestConfig(dir)
		cfg.BuildFlags = []string{"-gcflags=all=-e"}
		cfg.ParseFile = func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			return parser.ParseFile(fset, filename, src, parser.AllErrors|parser.ParseComments)
		}
		pkgs, err := LoadExWithGoVersion(nil, nil, cfg, "go1.24", ".")
		if err != nil {
			t.Fatal(err)
		}
		if len(pkgs) != 1 {
			t.Fatalf("load returned %d packages, want 1", len(pkgs))
		}
		pkg := pkgs[0]
		assertPackageError(t, pkg, "syntax error:")
		for _, secondary := range []string{"expected statement", "found 'EOF'", "undefined: missing"} {
			assertPackageErrorAbsent(t, pkg, secondary)
		}
	})

	t.Run("compiler type errors are authoritative", func(t *testing.T) {
		dir := t.TempDir()
		writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/typeerror\ngo 1.24\n")
		writeLoadTestFile(t, filepath.Join(dir, "load.go"), `package typeerror
var x = struct{ X int }{Y: 1}
`)
		cfg := loadTestConfig(dir)
		cfg.BuildFlags = []string{"-gcflags=all=-e"}
		pkgs, err := LoadExWithGoVersion(nil, nil, cfg, "go1.24", ".")
		if err != nil {
			t.Fatal(err)
		}
		if len(pkgs) != 1 {
			t.Fatalf("load returned %d packages, want 1", len(pkgs))
		}
		pkg := pkgs[0]
		assertPackageError(t, pkg, "unknown field Y")
		if len(pkg.Errors) != 1 {
			t.Fatalf("load returned %d package errors, want the compiler diagnostic only: %+v", len(pkg.Errors), pkg.Errors)
		}
	})

	t.Run("embed local var", func(t *testing.T) {
		dir := t.TempDir()
		writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/embedlocal\ngo 1.24\n")
		writeLoadTestFile(t, filepath.Join(dir, "x.txt"), "x")
		writeLoadTestFile(t, filepath.Join(dir, "load.go"), `package embedlocal
import _ "embed"
func f() {
	//go:embed x.txt // ERROR
	var x string
	_ = x
}`)
		pkg := loadOnePackage(t, dir, "go1.24")
		assertPackageError(t, pkg, "go:embed cannot apply to var inside func")
		assertPackageErrorAbsent(t, pkg, embedPatternDriverDiagnostic)
	})

	t.Run("embed language version", func(t *testing.T) {
		dir := t.TempDir()
		writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/embedversion\ngo 1.24\n")
		writeLoadTestFile(t, filepath.Join(dir, "x.txt"), "x")
		writeLoadTestFile(t, filepath.Join(dir, "load.go"), `package embedversion
import _ "embed"
//go:embed x.txt // ERROR
var x string`)
		pkg := loadOnePackage(t, dir, "go1.15")
		assertPackageError(t, pkg, "go:embed requires go1.16 or later")
		assertPackageErrorAbsent(t, pkg, embedPatternDriverDiagnostic)
	})

	t.Run("absolute import", func(t *testing.T) {
		dir := t.TempDir()
		writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/absoluteimport\ngo 1.24\n")
		writeLoadTestFile(t, filepath.Join(dir, "load.go"), "package absoluteimport\nimport _ \"/foo\"\n")
		pkg := loadOnePackage(t, dir, "go1.24")
		assertPackageError(t, pkg, "import path cannot be absolute path")
		assertPackageErrorAbsent(t, pkg, "no metadata for /foo")
	})

}

func TestHasGoSourceListDiagnostics(t *testing.T) {
	goFile := filepath.Join(string(filepath.Separator), "tmp", "example", "load.go")
	cgoFile := filepath.Join(string(filepath.Separator), "tmp", "example", "cgoerr", "cgo.go")
	tests := []struct {
		name            string
		err             xpackages.Error
		compiledGoFiles []string
		want            bool
	}{
		{
			name:            "absolute Go source",
			err:             xpackages.Error{Kind: xpackages.ListError, Msg: "# example.com/p\n" + goFile + ":2:3: undefined: missing"},
			compiledGoFiles: []string{goFile},
			want:            true,
		},
		{
			name:            "relative Go source",
			err:             xpackages.Error{Kind: xpackages.ListError, Msg: "# example.com/p\nload.go:2: undefined: missing"},
			compiledGoFiles: []string{goFile},
			want:            true,
		},
		{
			name:            "different Go source",
			err:             xpackages.Error{Kind: xpackages.ListError, Msg: "# example.com/p\nother.go:2: undefined: missing"},
			compiledGoFiles: []string{goFile},
		},
		{
			name: "cgo diagnostic in Go source",
			err: xpackages.Error{Kind: xpackages.ListError, Msg: `# example.com/diagcompare/cgoerr
cgoerr/cgo.go:4:2: error: "intentional cgo failure"
#error "intentional cgo failure"
 ^
1 error generated.
`},
			compiledGoFiles: []string{cgoFile},
			want:            true,
		},
		{
			name:            "unpositioned build error",
			err:             xpackages.Error{Kind: xpackages.ListError, Msg: "# example.com/p\ncompile: internal compiler error"},
			compiledGoFiles: []string{goFile},
		},
		{
			name:            "metadata list error",
			err:             xpackages.Error{Kind: xpackages.ListError, Msg: "import cycle not allowed"},
			compiledGoFiles: []string{goFile},
		},
		{
			name:            "non-list error",
			err:             xpackages.Error{Kind: xpackages.TypeError, Msg: "# example.com/p\nload.go:2: undefined: missing"},
			compiledGoFiles: []string{goFile},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasGoSourceListDiagnostics([]xpackages.Error{tt.err}, tt.compiledGoFiles); got != tt.want {
				t.Fatalf("hasGoSourceListDiagnostics() = %t, want %t", got, tt.want)
			}
		})
	}
}

func loadOnePackage(t *testing.T, dir, goVersion string) *Package {
	t.Helper()
	pkgs, err := LoadExWithGoVersion(nil, nil, loadTestConfig(dir), goVersion, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("load returned %d packages, want 1", len(pkgs))
	}
	return pkgs[0]
}

func assertPackageError(t *testing.T, pkg *Package, text string) {
	t.Helper()
	if !packageErrorsContain(pkg, text) {
		t.Fatalf("package errors do not contain %q: %+v", text, pkg.Errors)
	}
}

func assertPackageErrorAbsent(t *testing.T, pkg *Package, text string) {
	t.Helper()
	if packageErrorsContain(pkg, text) {
		t.Fatalf("package errors unexpectedly contain %q: %+v", text, pkg.Errors)
	}
}

func loadTestConfig(dir string) *Config {
	return &Config{
		Dir: dir,
		Mode: NeedName | NeedFiles | NeedCompiledGoFiles | NeedSyntax |
			NeedTypes | NeedTypesInfo | NeedTypesSizes | NeedModule,
	}
}

func packageErrorsContain(pkg *Package, text string) bool {
	for _, err := range pkg.Errors {
		if strings.Contains(err.Msg, text) {
			return true
		}
	}
	return false
}

func writeLoadTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
