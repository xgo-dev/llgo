//go:build !llgo

package packages

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestValidateCompilerDirectives(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		allowParseError bool
		wantFile        string
		wantLine        int
		wantColumn      int
	}{
		{
			name: "noescape function body",
			source: `package p
//go:noescape
func f() {}
`,
			wantLine:   3,
			wantColumn: 6,
		},
		{
			name: "noescape directive argument",
			source: `package p
//go:noescape ignored
func f() {}
`,
			wantLine: 3,
		},
		{
			name: "blank line before function",
			source: `package p
//go:noescape

func f() {}
`,
			wantLine: 4,
		},
		{
			name: "ordinary comment before function",
			source: `package p
//go:noescape
// ordinary

func f() {}
`,
			wantLine: 5,
		},
		{
			name: "between func keyword and name",
			source: `package p
func
//go:noescape
f() {}
`,
			wantLine:   4,
			wantColumn: 1,
		},
		{
			name: "method body",
			source: `package p
type T int
//go:noescape
func (T) f() {}
`,
			wantLine:   4,
			wantColumn: 6,
		},
		{
			name: "between func keyword and receiver",
			source: `package p
type T int
func
//go:noescape
(T) f() {}
`,
			wantLine:   5,
			wantColumn: 1,
		},
		{
			name: "between receiver and method name",
			source: `package p
type T int
func (T)
//go:noescape
f() {}
`,
			allowParseError: true,
		},
		{
			name: "external implementation",
			source: `package p
//go:noescape
func f()
`,
		},
		{
			name: "spaced comment",
			source: `package p
// go:noescape
func f() {}
`,
		},
		{
			name: "lookalike directive",
			source: `package p
//go:noescapex
func f() {}
`,
		},
		{
			name:   "tab argument",
			source: "package p\n//go:noescape\tignored\nfunc f() {}\n",
		},
		{
			name: "block comment",
			source: `package p
/*go:noescape*/
func f() {}
`,
		},
		{
			name: "intervening variable declaration",
			source: `package p
//go:noescape
var (
	x int
)
func f() {}
`,
		},
		{
			name: "intervening type declaration",
			source: `package p
//go:noescape
type T int
func f() {}
`,
		},
		{
			name: "misplaced on previous declaration line",
			source: `package p
var x int //go:noescape
func f() {}
`,
		},
		{
			name: "ordinary block comment on directive line",
			source: `package p
/* ordinary */ //go:noescape
func f() {}
`,
		},
		{
			name: "after func keyword on same line",
			source: `package p
func //go:noescape
f() {}
`,
		},
		{
			name: "before package declaration",
			source: `//go:noescape
package p
func f() {}
`,
		},
		{
			name: "physical line with logical line remapping",
			source: `package p
var x int
//line remapped.go:2:1
//go:noescape
func f() {}
`,
			wantFile:   "remapped.go",
			wantLine:   3,
			wantColumn: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "pragma.go", tt.source, parser.ParseComments)
			if err != nil && !tt.allowParseError {
				t.Fatal(err)
			}
			errs := validateCompilerDirectives(fset, []*ast.File{nil, file})
			if tt.wantLine == 0 {
				if len(errs) != 0 {
					t.Fatalf("validateCompilerDirectives returned %+v, want no errors", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("validateCompilerDirectives returned %d errors, want 1: %+v", len(errs), errs)
			}
			if errs[0].Msg != noescapeBodyDiagnostic {
				t.Fatalf("diagnostic = %q, want %q", errs[0].Msg, noescapeBodyDiagnostic)
			}
			got := fset.Position(errs[0].Pos)
			if got.Line != tt.wantLine {
				t.Fatalf("diagnostic line = %d, want %d", got.Line, tt.wantLine)
			}
			if tt.wantColumn != 0 && got.Column != tt.wantColumn {
				t.Fatalf("diagnostic column = %d, want %d", got.Column, tt.wantColumn)
			}
			if tt.wantFile != "" && got.Filename != tt.wantFile {
				t.Fatalf("diagnostic file = %q, want %q", got.Filename, tt.wantFile)
			}
		})
	}

	if errs := validateCompilerDirectives(nil, nil); len(errs) != 0 {
		t.Fatalf("nil file set returned %+v, want no errors", errs)
	}
}

func TestPackageHasCompilerDiagnostic(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "/tmp/pragma.go", "package p\n\nfunc f() {}\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	want := types.Error{Fset: fset, Pos: fn.Pos(), Msg: noescapeBodyDiagnostic}
	tests := []struct {
		name string
		errs []Error
		want bool
	}{
		{
			name: "structured diagnostic",
			errs: []Error{{Pos: "/tmp/pragma.go:3:1", Msg: noescapeBodyDiagnostic}},
			want: true,
		},
		{
			name: "driver diagnostic",
			errs: []Error{{Msg: "# p\n/tmp/pragma.go:3: " + noescapeBodyDiagnostic}},
			want: true,
		},
		{
			name: "wrong line",
			errs: []Error{{Pos: "/tmp/pragma.go:2:1", Msg: noescapeBodyDiagnostic}},
		},
		{
			name: "wrong message",
			errs: []Error{{Pos: "/tmp/pragma.go:3:1", Msg: "other error"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := packageHasCompilerDiagnostic(tt.errs, want); got != tt.want {
				t.Fatalf("packageHasCompilerDiagnostic() = %v, want %v", got, tt.want)
			}
		})
	}
}
