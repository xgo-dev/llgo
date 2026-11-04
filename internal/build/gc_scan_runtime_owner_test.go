//go:build go1.26

package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

const gcScanRuntimePackage = "internal/runtime/gc/scan"

// These are the complete supported expansion factors, not a sample of the
// first failing private entry. The SDK's PCs carry AX input and Z1/Z2 output;
// they are not ordinary void(ptr) Go functions.
func gcScanRuntimeFactors() []int {
	return []int{
		1, 2, 3, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32,
		36, 40, 44, 48, 52, 56, 60, 64,
	}
}

func TestGCScanRuntimeAMD64CompleteOriginalFamily(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "gc", "scan")
	var expanders []string
	for _, factor := range gcScanRuntimeFactors() {
		expanders = append(expanders, fmt.Sprintf("expandAVX512_%d<>", factor))
	}
	text := regexp.MustCompile(`(?m)^\s*TEXT\s+·([^\s(,]+)\(SB\),`)
	for _, owner := range []struct {
		file string
		want []string
	}{
		{"expand_amd64.s", expanders},
		{"scan_amd64.s", []string{"ExpandAVX512", "scanSpanPackedAVX512"}},
		{"filter_amd64.s", []string{"FilterNilAVX512"}},
	} {
		t.Run(owner.file, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(dir, owner.file))
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, match := range text.FindAllSubmatch(source, -1) {
				names = append(names, string(match[1]))
			}
			if !reflect.DeepEqual(names, owner.want) {
				t.Fatalf("complete original SDK owner changed: got %v, want %v", names, owner.want)
			}
			if owner.file == "expand_amd64.s" {
				pc := regexp.MustCompile(`\$·(expandAVX512_[0-9]+<>)\(SB\)`)
				var entries []string
				for _, match := range pc.FindAllSubmatch(source, -1) {
					entries = append(entries, string(match[1]))
				}
				if !reflect.DeepEqual(entries, expanders) {
					t.Fatal("the actual PC table does not own every private expander exactly once", entries)
				}
			}
		})
	}
}

func TestGCScanRuntimeAMD64SourceOwnerBoundaries(t *testing.T) {
	ctx := sourcePatchBuildContext{goos: runtime.GOOS, goarch: "amd64", goversion: runtime.Version()}
	runtimeDir := env.LLGoRuntimeDir()
	dir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(gcScanRuntimePackage))
	patch := filepath.Join(runtimeDir, "_patch", filepath.FromSlash(gcScanRuntimePackage), "scan_amd64.go")
	body, err := os.ReadFile(patch)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(base map[string][]byte) (map[string][]byte, error) {
		changed, overlay, _, err := applySourcePatchForPkg(base, nil, runtimeDir, runtime.GOROOT(), gcScanRuntimePackage, ctx)
		if err == nil && !changed {
			return nil, fmt.Errorf("missing complete source owner")
		}
		return overlay, err
	}
	good, err := apply(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"expand_amd64.s", "scan_amd64.s", "filter_amd64.s"} {
		if string(good[filepath.Join(dir, name)]) != "// replaced by LLGo source patch\n" {
			t.Fatal("missing whole assembly owner", name)
		}
	}
	for _, name := range []string{"expand_amd64.go", "scan_amd64.go", "filter_amd64.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), name, good[filepath.Join(dir, name)], 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && ident.Name == "gcExpandersAVX512" {
				t.Fatal("the selected Go view retains a fake or unowned PC table")
			}
			return true
		})
	}
	if _, err := apply(good); err != nil {
		t.Fatal("the complete owner cannot be safely reused", err)
	}
	parsed := token.NewFileSet()
	file, err := parser.ParseFile(parsed, patch, body, 0)
	if err != nil {
		t.Fatal(err)
	}
	var noBody []byte
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "ExpandAVX512" {
			start, end := parsed.Position(fn.Body.Pos()).Offset, parsed.Position(fn.Body.End()).Offset
			noBody = append(append([]byte{}, body[:start]...), body[end:]...)
		}
	}
	if len(noBody) == 0 {
		t.Fatal("missing-body negative did not modify the actual owner")
	}
	for _, test := range []struct {
		name string
		base map[string][]byte
	}{
		{"missing body", map[string][]byte{patch: bytes.Replace(body, []byte("func ExpandAVX512("), []byte("func MissingExpand("), 1)}},
		{"declaration only", map[string][]byte{patch: noBody}},
		{"wrong width", map[string][]byte{patch: bytes.Replace(body, []byte("n int32"), []byte("n int64"), 1)}},
		{"empty source", map[string][]byte{filepath.Join(dir, "expand_amd64.s"): nil}},
		{"new public TEXT", map[string][]byte{filepath.Join(dir, "scan_amd64.s"): []byte("TEXT ·Unexpected(SB),$0-0\nRET\n")}},
		{"remaining PC consumer", map[string][]byte{filepath.Join(dir, "scan_go.go"): []byte("package scan\nfunc usePC() uintptr { return gcExpandersAVX512[1] }\n")}},
		{"duplicate PC declaration", map[string][]byte{filepath.Join(dir, "scan_go.go"): []byte("package scan\nvar gcExpandersAVX512 [1]uintptr\n")}},
		{"partial marker", map[string][]byte{filepath.Join(dir, "scan_amd64.s"): good[filepath.Join(dir, "scan_amd64.s")]}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := apply(test.base); err == nil {
				t.Fatal("accepted an incomplete or changed gc scan owner")
			}
		})
	}
	t.Run("new assembly owner", func(t *testing.T) {
		goroot := t.TempDir()
		fixture := filepath.Join(goroot, "src", filepath.FromSlash(gcScanRuntimePackage))
		if err := os.MkdirAll(fixture, 0o700); err != nil {
			t.Fatal(err)
		}
		for name := range gcScanReviewedSources {
			source, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			assemblyLanguageTestWrite(t, fixture, name, string(source))
		}
		assemblyLanguageTestWrite(t, fixture, "new_amd64.s", "TEXT ·NewEntry(SB),$0-0\nRET\n")
		if _, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, gcScanRuntimePackage, ctx); err == nil {
			t.Fatal("accepted a new assembly owner outside the complete reviewed group")
		}
	})
	for _, arch := range []string{"arm64", "386"} {
		t.Run("retain "+arch, func(t *testing.T) {
			other := ctx
			other.goarch = arch
			changed, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, runtime.GOROOT(), gcScanRuntimePackage, other)
			if err != nil || changed {
				t.Fatal("AMD64 owner changed another architecture", err)
			}
		})
	}
	t.Run("retain older SDK", func(t *testing.T) {
		older := ctx
		older.goversion = "go1.25.14"
		changed, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, runtime.GOROOT(), gcScanRuntimePackage, older)
		if err != nil || changed {
			t.Fatal("the Go1.26+ owner changed an older SDK route", err)
		}
	})
}
