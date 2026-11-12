package plan9asm

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/types"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	extplan9asm "github.com/xgo-dev/plan9asm"
)

func TestARM64TailSelectionSourceBounds(t *testing.T) {
	for _, mutation := range []string{"duplicate", "file-count", "file-bytes", "total-bytes"} {
		t.Run(mutation, func(t *testing.T) {
			pkg, opt, source := arm64TailSelectionFixture(t, true, false)
			want := "duplicate selected Go source"
			if mutation == "duplicate" {
				pkg.CompiledGoFiles[1] = pkg.CompiledGoFiles[0]
			} else {
				count, size := 1, (4<<20)+1
				switch mutation {
				case "file-count":
					count, size, want = 255, 64, "complete actual selected Go package"
				case "file-bytes":
					want = "input byte bound"
				case "total-bytes":
					count, size, want = 4, 4<<20, "selected Go source exceeds its input bound"
				}
				// Parse and type-check the complete selected files, including
				// bounded comments; the size test must not borrow a stale AST.
				for i := 0; i < count; i++ {
					filename := filepath.Join(filepath.Dir(pkg.CompiledGoFiles[0]), fmt.Sprintf("bound%d.go", i))
					prefix, suffix := "package abizero\n/*", "*/\n"
					data := []byte(prefix + strings.Repeat(" ", size-len(prefix)-len(suffix)) + suffix)
					file, err := parser.ParseFile(pkg.Fset, filename, data, parser.ParseComments)
					if err != nil {
						t.Fatal(err)
					}
					pkg.CompiledGoFiles = append(pkg.CompiledGoFiles, filename)
					pkg.GoFiles = append(pkg.GoFiles, filename)
					pkg.Syntax = append(pkg.Syntax, file)
					opt.Overlay[filename] = data
				}
				pkg.TypesInfo = &types.Info{Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object)}
				var err error
				pkg.Types, err = (&types.Config{Sizes: pkg.TypesSizes}).Check(pkg.PkgPath, pkg.Fset, pkg.Syntax, pkg.TypesInfo)
				if err != nil {
					t.Fatal("actual selected package prerequisite", err)
				}
			}
			before := append([]string(nil), pkg.CompiledGoFiles...)
			selection, err := arm64TailSourceSelection(pkg, []byte(source), "arm64", opt.Overlay)
			if err == nil || !strings.Contains(err.Error(), want) || selection != nil {
				t.Fatalf("source selection %s: present=%t, error=%v; want %q", mutation, selection != nil, err, want)
			}
			if mutation != "file-bytes" && !errors.Is(err, extplan9asm.ErrProbeNeedsContext) {
				t.Fatal("source-selection bound lost its context error identity", err)
			}
			if !reflect.DeepEqual(pkg.CompiledGoFiles, before) {
				t.Fatal("failed selection changed the compiler's file inventory")
			}
		})
	}
}

func TestARM64TailSelectionIsolatedSourceSnapshot(t *testing.T) {
	pkg, opt, source := arm64TailSelectionFixture(t, true, false)
	var before bytes.Buffer
	for _, file := range pkg.Syntax {
		if err := format.Node(&before, pkg.Fset, file); err != nil {
			t.Fatal(err)
		}
	}
	selection, err := arm64TailSourceSelection(pkg, []byte(source), "arm64", opt.Overlay)
	if err != nil || selection == nil {
		t.Fatal("complete original selection prerequisite", err)
	}
	first := pkg.CompiledGoFiles[0]
	if !bytes.Equal(selection.GoSources[first], opt.Overlay[first]) {
		t.Fatal("selected bytes differ from the actual parser overlay")
	}
	selection.SelectedGoFiles[0] = "changed.go"
	selection.GoSources[first][0] = '!'
	if pkg.CompiledGoFiles[0] != first || opt.Overlay[first][0] != 'p' {
		t.Fatal("selection snapshot aliases the compiler's input slices")
	}
	var after bytes.Buffer
	for _, file := range pkg.Syntax {
		if err := format.Node(&after, pkg.Fset, file); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatal("selecting source rewrote the compiler's typed syntax")
	}
}

func TestARM64TailSelectionOnlyLoadsConsumedSource(t *testing.T) {
	for _, test := range []struct{ arch, source string }{
		{"amd64", "not ARM64 assembly"},
		{"arm64", "TEXT ·NoTail(SB),4,$0-0\nRET\n"},
	} {
		// No package/source metadata is required for an unconsumed contract.
		selection, err := arm64TailSourceSelection(nil, []byte(test.source), test.arch, nil)
		if err != nil || selection != nil {
			t.Fatal("unconsumed source was loaded", test.arch, selection, err)
		}
	}
	if selection, err := arm64TailSourceSelection(nil, []byte("TEXT"), "arm64", nil); err == nil || selection != nil {
		t.Fatal("malformed assembly silently bypassed selection", selection, err)
	}
}
