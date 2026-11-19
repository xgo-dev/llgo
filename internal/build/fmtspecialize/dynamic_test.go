package fmtspecialize

import (
	"go/parser"
	"go/token"
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
	generated := BuiltinFormatSource(data)
	file, err := parser.ParseFile(token.NewFileSet(), "builtin_format.go", generated, 0)
	if err != nil || len(file.Decls) != 10 {
		t.Fatalf("unexpected generated builtin formatter: %v", err)
	}
	text := string(generated)
	for _, call := range []string{"p.printArg(", "p.printValue(", "p.handleMethods(", "p.badVerb("} {
		if strings.Contains(text, call) {
			t.Fatalf("builtin formatter still reaches %s", call)
		}
	}
	if BuiltinFormatSource([]byte("package fmt\n")) != nil {
		t.Fatal("missing standard-library methods did not disable specialization")
	}
}
