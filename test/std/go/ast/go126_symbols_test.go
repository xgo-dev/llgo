//go:build go1.26

package ast_test

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"
)

func TestDirective(t *testing.T) {
	const text = "//go:generate tool plain \"quoted value\" `raw value`"
	const pos = token.Pos(10)
	directive, ok := ast.ParseDirective(pos, text)
	if !ok {
		t.Fatal("ParseDirective rejected a valid directive")
	}
	if directive.Pos() != pos || directive.End() != pos+token.Pos(len(text)) {
		t.Fatalf("directive positions = [%d, %d), want [%d, %d)", directive.Pos(), directive.End(), pos, pos+token.Pos(len(text)))
	}
	args, err := directive.ParseArgs()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"tool", "plain", "quoted value", "raw value"}
	got := make([]string, len(args))
	for i, arg := range args {
		got[i] = arg.Arg
		if arg.Pos < directive.ArgsPos || arg.Pos >= directive.End() {
			t.Fatalf("argument %q has invalid position %d", arg.Arg, arg.Pos)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseArgs = %q, want %q", got, want)
	}
}
