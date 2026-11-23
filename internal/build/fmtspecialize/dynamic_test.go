package fmtspecialize

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"go/version"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuiltinFormatSource(t *testing.T) {
	if version.Lang(runtime.Version()) != "go1.27" {
		t.Skip("requires the Go 1.27 fmt implementation")
	}
	data, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src", "fmt", "print.go"))
	if err != nil {
		t.Fatal(err)
	}
	generated := BuiltinFormatSource(data, false)
	file, err := parser.ParseFile(token.NewFileSet(), "builtin_format.go", generated, 0)
	if err != nil || len(file.Decls) != 12 {
		t.Fatalf("unexpected generated builtin formatter: %v", err)
	}
	text := string(generated)
	if !strings.Contains(text, "//llgointernal:tls\nvar llgoBuiltinPrinter *pp") {
		t.Fatal("hosted formatter lacks its thread-local printer")
	}
	if bytes.Contains(BuiltinFormatSource(data, true), []byte("//llgointernal:tls")) {
		t.Fatal("bare-metal formatter requires thread-local storage")
	}
	for _, call := range []string{"p.printArg(", "p.printValue(", "p.handleMethods(", "p.badVerb(", "reflect.TypeOf("} {
		if strings.Contains(text, call) {
			t.Fatalf("builtin formatter still reaches %s", call)
		}
	}
	if got := strings.Count(text, "llgoBuiltin_typeName(arg)"); got != 3 {
		t.Fatalf("builtin formatter rewrote %d type name lookups, want 3", got)
	}
	if got := strings.Count(text, "llgoKinds&"); got != int(types.String-types.Bool+1) {
		t.Fatalf("builtin formatter guarded %d scalar printers", got)
	}
	if BuiltinFormatSource([]byte("package fmt\n"), false) != nil {
		t.Fatal("missing standard-library methods did not disable specialization")
	}
	if BuiltinFormatSource([]byte("package"), false) != nil {
		t.Fatal("invalid standard-library source did not disable specialization")
	}
	for _, test := range []struct {
		name, old, replacement string
	}{
		{"changed receiver", "func (p *pp) printArg(", "func (p pp) printArg("},
		{"changed parameter type", "printArg(arg any, verb rune)", "printArg(arg interface{}, verb rune)"},
		{"changed pointer handling", "case 'p':\n\t\tp.fmtPointer", "case 'x':\n\t\tp.fmtPointer"},
		{"combined type cases", "case bool:", "case bool, int:"},
		{"missing scalar type", "case bool:", "case fmt.Stringer:"},
		{"non-scalar case", "case bool:", "case error:"},
		{"unknown scalar case", "case bool:", "case unknownBuiltin:"},
		{"duplicate scalar alias", "case uint16:", "case byte:"},
		{"changed bad verb guard", "case value.IsValid():", "case value.CanAddr():"},
		{"reflective call from scalar helper", "func (p *pp) fmtBool(arg any, value reflect.Value, v bool, verb rune) {",
			"func (p *pp) fmtBool(arg any, value reflect.Value, v bool, verb rune) { p.printValue(value, verb, 0);"},
		{"unbound parser method", "func (p *pp) fmtBool(arg any, value reflect.Value, v bool, verb rune) {",
			"func (p *pp) fmtBool(arg any, value reflect.Value, v bool, verb rune) { _ = p.fmtString;"},
	} {
		t.Run(test.name, func(t *testing.T) {
			modified := bytes.ReplaceAll(data, []byte(test.old), []byte(test.replacement))
			if bytes.Equal(modified, data) {
				t.Fatalf("standard-library source lacks %q", test.old)
			}
			if BuiltinFormatSource(modified, false) != nil {
				t.Fatal("unsupported parser shape enabled specialization")
			}
		})
	}
}

func TestRewriteBuiltinTypeName(t *testing.T) {
	for _, test := range []struct {
		expression string
		want       bool
	}{
		{"reflect.TypeOf(arg).String()", true},
		{"reflect.TypeOf(other).String()", false},
		{"reflect.ValueOf(arg).String()", false},
		{"format.String()", false},
	} {
		t.Run(test.expression, func(t *testing.T) {
			expr, err := parser.ParseExpr(test.expression)
			if err != nil {
				t.Fatal(err)
			}
			call := expr.(*ast.CallExpr)
			if got := rewriteBuiltinTypeName(call); got != test.want {
				t.Fatalf("rewrote call = %t, want %t", got, test.want)
			}
			if test.want && (!identifier(call.Fun, "llgoBuiltin_typeName") || len(call.Args) != 1 || !identifier(call.Args[0], "arg")) {
				t.Fatalf("rewritten call = %#v", call)
			}
		})
	}
}
