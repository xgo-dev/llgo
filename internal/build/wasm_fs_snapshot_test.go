package build

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

func TestWasmFSSnapshotTag(t *testing.T) {
	for _, test := range []struct {
		flags, want []string
	}{
		{nil, []string{"-tags=" + wasmFSSnapshotTag}},
		{[]string{"-tags=older", "-tags=active", "-trimpath"}, []string{"-tags=older", "-tags=active," + wasmFSSnapshotTag, "-trimpath"}},
		{[]string{"-tags", "a,b"}, []string{"-tags", "a,b," + wasmFSSnapshotTag}},
	} {
		if got := appendWasmFSSnapshotTag(test.flags); !slices.Equal(got, test.want) {
			t.Errorf("tags = %v, want %v", got, test.want)
		}
	}
}

func TestWasmFSSnapshotSourcePatch(t *testing.T) {
	legacy := "package syscall\n" + wasmFSStart + wasmFSValues + "\nvar (" + wasmFSFlags + "\n)\ntype jsFile struct{}\n"
	modern := "package syscall\n" + wasmFSStart + "\n" + wasmFSPath + wasmFSValues +
		"\nvar (" + wasmFSFlags + "\nnodeDIRECTORY = -1\n)" + wasmFSDirectory + "\ntype jsFile struct{}\n"
	for _, test := range []struct {
		name, source string
		want         bool
	}{
		{"legacy", legacy, true},
		{"modern", modern, true},
		{"different lookup", strings.Replace(legacy, `Get("fs")`, `Get("otherFS")`, 1), false},
		{"extra initializer", strings.Replace(modern, "type jsFile", "func init() { observe(constants) }\ntype jsFile", 1), false},
		{"known use", modern + "func use() { jsFS.Call(\"open\") }\n", true},
		{"new method", modern + "func use() { jsFS.IsUndefined() }\n", false},
		{"plain value", modern + "func use() { _ = jsFS }\n", false},
		{"incomplete", "package syscall\n" + wasmFSStart, false},
		{"invalid", "not Go source", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := rewriteWasmFSInitializers([]byte(test.source))
			if (result != nil) != test.want {
				t.Fatalf("rewrite = %v, want %v", result != nil, test.want)
			}
			if result == nil {
				return
			}
			if bytes.Count(result, []byte("\n")) != strings.Count(test.source, "\n") {
				t.Fatal("rewrite changed source line positions")
			}
			if _, err := parser.ParseFile(token.NewFileSet(), "fs_js.go", result, parser.SkipObjectResolution); err != nil {
				t.Fatal(err)
			}
			for _, part := range []string{"llgoCaptureFSValue(0)", "llgoCaptureFSConstants(jsFS)", "constants.intFlag(0)", "constants.intFlag(5)"} {
				if !bytes.Contains(result, []byte(part)) {
					t.Errorf("rewritten source lacks %s", part)
				}
			}
			for _, optional := range []struct{ before, after string }{
				{"var jsPath", "llgoCaptureFSValue(1)"},
				{"nodeDIRECTORY", "constants.intFlag(6)"},
			} {
				if bytes.Contains(result, []byte(optional.after)) != strings.Contains(test.source, optional.before) {
					t.Errorf("unexpected rewrite for %s", optional.before)
				}
			}
		})
	}

	goroot := runtime.GOROOT()
	path := filepath.Join(goroot, "src", "syscall", "fs_js.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcePatchBuildContext{goos: "js", goarch: "wasm", goversion: runtime.Version(), targetTags: []string{"llgo", wasmFSSnapshotTag}, fsSnapshot: rewriteWasmFSInitializers(source)}
	changed, overlay, files, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), goroot, "syscall", ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := ctx.fsSnapshot != nil
	if changed != want || (overlay[path] != nil) != want || (len(files) != 0) != want {
		t.Fatalf("source and helper must be selected together, got changed=%v overlay=%v files=%v", changed, overlay[path] != nil, files)
	}
	changed, _, _, err = applySourcePatchForPkg(map[string][]byte{path: source}, nil, env.LLGoRuntimeDir(), goroot, "syscall", ctx)
	if err != nil || changed {
		t.Fatalf("rewrote a user overlay: changed=%v, err=%v", changed, err)
	}
}
