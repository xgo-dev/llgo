/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

// parseSyntaxCapturingStderr parses src through ParsePkgSyntax and returns the
// error together with whatever the pass wrote to os.Stderr (deprecation
// warnings).
func parseSyntaxCapturingStderr(t *testing.T, pkgPath, src string) (string, error) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	prog := llssa.NewProgram(nil)
	pkg := types.NewPackage(pkgPath, "p")

	old := os.Stderr
	r, w, perr := os.Pipe()
	if perr != nil {
		t.Fatalf("os.Pipe failed: %v", perr)
	}
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	// Restore os.Stderr and close the pipe writer even if ParsePkgSyntax
	// panics, so a failing parse cannot leave the process-global os.Stderr
	// pointing at a closed pipe or leak the writer.
	defer func() {
		w.Close()
		os.Stderr = old
	}()

	parseErr := ParsePkgSyntax(prog, fset, pkg, []*ast.File{file})

	// Close the writer and restore os.Stderr before draining the reader so the
	// goroutine observes EOF; the deferred cleanup is then a safe no-op.
	w.Close()
	os.Stderr = old
	return <-done, parseErr
}

// cdeclTmpl exercises a C-convention callback: a direct call and a callback
// passed through a native function pointer. %s is the directive on Callback.
const cdeclTmpl = `package equiv

//go:linkname invoke C.invoke
func invoke(Callback, int32) int32

%s
type Callback func(int32) int32

func callback(v int32) int32 { return v + 1 }

func use(cb Callback, v int32) int32 {
	return cb(v) + invoke(callback, v)
}
`

// stdcallTmpl exercises a stdcall callback through the stdcall linkname
// namespace. %s is the directive on Callback.
const stdcallTmpl = `package equiv

import _ "unsafe"

%s
type Callback func(int32) int32

//go:linkname invoke stdcall.Invoke
func invoke(Callback, int32) int32

//go:linkname direct stdcall.Direct
func direct(int32) int32

func callback(v int32) int32 { return v + 1 }

func use(cb Callback, v int32) int32 {
	return direct(v) + cb(v) + invoke(callback, v)
}
`

// structTmpl exercises an opaque C struct (the documented JSON/Reader pattern)
// passed by pointer to a linked C function. The //llgo:cdecl / //llgo:type C
// directive sits on the struct *type* declaration, not a field. %s is the
// directive on Opaque.
const structTmpl = `package equiv

%s
type Opaque struct {
	Unused [0]byte
}

//go:linkname consume C.consume
func consume(*Opaque) int32

func use(p *Opaque) int32 {
	return consume(p)
}
`

// TestCdeclStdcallEquivalentToLegacyType asserts that the new //llgo:cdecl and
// //llgo:stdcall directives generate exactly the same code, ABI and calling
// convention as the deprecated //llgo:type C / //llgo:type stdcall forms. The
// comparison is done on the fully compiled IR so it covers the whole codegen
// path, not just the parsed background value.
func TestCdeclStdcallEquivalentToLegacyType(t *testing.T) {
	cases := []struct {
		name   string
		tmpl   string
		new    string
		old    string
		target *llssa.Target
	}{
		{"cdecl", cdeclTmpl, "//llgo:cdecl", "//llgo:type C", nil},
		{"cdecl alias", strings.Replace(cdeclTmpl, "type Callback func", "type Callback = func", 1), "//llgo:cdecl", "//llgo:type C", nil},
		// //llgo:cdecl on a struct type (the opaque-C-struct case) must be
		// honored exactly like the legacy //llgo:type C spelling.
		{"cdecl struct", structTmpl, "//llgo:cdecl", "//llgo:type C", nil},
		// stdcall only diverges from the C ABI on windows/386; use that target
		// to exercise the dedicated x86 stdcall path.
		{"stdcall", stdcallTmpl, "//llgo:stdcall", "//llgo:type stdcall", &llssa.Target{GOOS: "windows", GOARCH: "386"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			irNew := compileWithRewritesTarget(t, strings.Replace(c.tmpl, "%s", c.new, 1), nil, c.target)
			irOld := compileWithRewritesTarget(t, strings.Replace(c.tmpl, "%s", c.old, 1), nil, c.target)
			if irNew != irOld {
				t.Fatalf("%s and %s produced different IR:\n--- %s ---\n%s\n--- %s ---\n%s",
					c.new, c.old, c.new, irNew, c.old, irOld)
			}
		})
	}
}

// TestLegacyTypeEmitsDeprecationWarning asserts that //llgo:type C and
// //llgo:type stdcall still work but print a deprecation warning naming the
// replacement, while the new directives are silent.
func TestLegacyTypeEmitsDeprecationWarning(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantMsg string // empty => no warning expected
	}{
		{
			name:    "legacy C warns and names cdecl",
			src:     "package p\n//llgo:type C\ntype Callback func()\n",
			wantMsg: "//llgo:cdecl",
		},
		{
			name:    "legacy stdcall warns and names stdcall",
			src:     "package p\n//llgo:type stdcall\ntype Callback func()\n",
			wantMsg: "//llgo:stdcall",
		},
		{
			name:    "new cdecl is silent",
			src:     "package p\n//llgo:cdecl\ntype Callback func()\n",
			wantMsg: "",
		},
		{
			name:    "new stdcall is silent",
			src:     "package p\n//llgo:stdcall\ntype Callback func()\n",
			wantMsg: "",
		},
		{
			name:    "other llgo:type forms are untouched",
			src:     "package p\n//llgo:type Python\ntype Callback func()\n",
			wantMsg: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stderr, err := parseSyntaxCapturingStderr(t, "example.com/"+strings.ReplaceAll(c.name, " ", "-"), c.src)
			if err != nil {
				t.Fatalf("ParsePkgSyntax failed: %v", err)
			}
			if c.wantMsg == "" {
				if strings.Contains(stderr, "DEPRECATED") {
					t.Fatalf("unexpected deprecation warning: %q", stderr)
				}
				return
			}
			if !strings.Contains(stderr, "DEPRECATED") || !strings.Contains(stderr, c.wantMsg) {
				t.Fatalf("deprecation warning = %q, want DEPRECATED mentioning %q", stderr, c.wantMsg)
			}
		})
	}
}

// TestTypeCallConvConflictsAndInvalidPlacement asserts that conflicting or
// misplaced calling-convention directives are rejected with a clear error
// rather than silently ignored.
func TestTypeCallConvConflictsAndInvalidPlacement(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "two new directives conflict",
			src:  "package p\n//llgo:cdecl\n//llgo:stdcall\ntype Callback func()\n",
			want: "conflicting calling-convention directives",
		},
		{
			name: "new and legacy conflict",
			src:  "package p\n//llgo:cdecl\n//llgo:type stdcall\ntype Callback func()\n",
			want: "conflicting calling-convention directives",
		},
		{
			name: "duplicate same directive conflicts",
			src:  "package p\n//llgo:cdecl\n//llgo:cdecl\ntype Callback func()\n",
			want: "conflicting calling-convention directives",
		},
		{
			name: "directive on grouped type block",
			src:  "package p\n//llgo:cdecl\ntype (\n\tA func()\n\tB func()\n)\n",
			want: "single type declaration",
		},
		{
			name: "legacy directive on grouped type block",
			src:  "package p\n//llgo:type C\ntype (A func(); B func())\n",
			want: "single type declaration",
		},
		{
			name: "directive with argument",
			src:  "package p\n//llgo:cdecl extra\ntype Callback func()\n",
			want: "does not accept arguments",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stderr, err := parseSyntaxCapturingStderr(t, "example.com/"+strings.ReplaceAll(c.name, " ", "-"), c.src)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("ParsePkgSyntax error = %v, want %q", err, c.want)
			}
			if strings.Contains(stderr, "DEPRECATED") {
				t.Fatalf("invalid directive also emitted a deprecation warning: %q", stderr)
			}
		})
	}
}

// TestCdeclOnStructTypeAccepted asserts that a calling-convention directive on a
// struct *type* declaration (the opaque-C-struct case documented for
// //llgo:cdecl) is accepted and silent. A directive on a struct *field* carries
// no calling-convention semantics yet and is silently ignored rather than
// rejected. The directive's background value is checked separately via
// prepared records in TestTypeBackgroundAndParsePkgSyntaxCoverage; IR-level
// equivalence with the legacy spelling is covered by
// TestCdeclStdcallEquivalentToLegacyType.
func TestCdeclOnStructTypeAccepted(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "cdecl on struct type",
			src:  "package p\n//llgo:cdecl\ntype Opaque struct {\n\tUnused [0]byte\n}\n",
		},
		{
			name: "spaced cdecl on struct type",
			src:  "package p\n// llgo:cdecl\ntype Opaque struct {\n\tUnused [0]byte\n}\n",
		},
		{
			name: "cdecl on a struct field is ignored",
			src:  "package p\ntype S struct {\n\t//llgo:cdecl\n\tCallback func()\n}\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stderr, err := parseSyntaxCapturingStderr(t, "example.com/"+strings.ReplaceAll(c.name, " ", "-"), c.src)
			if err != nil {
				t.Fatalf("ParsePkgSyntax rejected //llgo:cdecl: %v", err)
			}
			if strings.Contains(stderr, "DEPRECATED") {
				t.Fatalf("unexpected deprecation warning for a new directive: %q", stderr)
			}
		})
	}
}
