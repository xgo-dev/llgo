//go:build !llgo

package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplicitAlignmentDiagnostics(t *testing.T) {
	runtimeDir, err := filepath.Abs("../../runtime")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, decl, body, diagnostic string }{
		{"unused-map", "type Unused map[int]Block", "", "explicit alignment above 8 bytes"},
		{"derived-map", "type Derived Block; type Unused map[[0]Derived]int", "", "explicit alignment above 8 bytes"},
		{"generic-body", "func f[T any]() { m := make(map[int]T); println(len(m)) }", "f[Block]()", "explicit alignment above 8 bytes"},
		{"pointer-map", "type Unused map[int]*Block", "m := make(Unused); m[1] = new(Block)", ""},
		{"native-argument", "//go:linkname consume C.consume\nfunc consume(Block)", "consume(Block{})", "explicitly aligned values at native C call boundaries"},
		{"native-result", "//go:linkname produce C.produce\nfunc produce() Block", "_ = produce()", "explicitly aligned values at native C call boundaries"},
		{"native-vararg", "//go:linkname consume C.consume\nfunc consume(n int32, __llgo_va_list ...any)", "consume(1, Block{})", "explicitly aligned values at native C call boundaries"},
		{"native-pointer-vararg", "//go:linkname consume C.consume\nfunc consume(n int32, __llgo_va_list ...any)", "consume(1, new(Block))", ""},
		{"native-pointer", "//go:linkname consume C.consume\nfunc consume(*Block)", "consume(new(Block))", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mod := fmt.Sprintf("module alignmenttest\n\ngo 1.27\nrequire github.com/xgo-dev/llgo/runtime v0.0.0\nreplace github.com/xgo-dev/llgo/runtime => %q\n", filepath.ToSlash(runtimeDir))
			source := "package main\nimport \"structs\"\nimport _ \"unsafe\"\ntype Block struct { _ structs.Align64; Data [24]byte }\n" + tc.decl + "\nfunc main() {" + tc.body + "}\n"
			for name, data := range map[string]string{"go.mod": mod, "main.go": source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch = "linux", "amd64"
			_, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if tc.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
				t.Fatalf("expected %q, got %v", tc.diagnostic, err)
			}
		})
	}
}
