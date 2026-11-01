//go:build go1.27

package build

import (
	"bytes"
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

// Use unchanged selected SDK sources and the actual replacement body. Adding
// another source must not merely fail an earlier digest check on a reviewed file.
type sourceOwnerBoundaryFixture struct {
	kind, dir, generated string
	context              build.Context
	body                 []byte
	skips                map[string]struct{}
	base                 map[string][]byte
	readFailure          string
}

var errSourceOwnerBoundaryRead = errors.New("selected source reader failed")

func newSourceOwnerBoundaryFixture(t *testing.T, kind, arch string) *sourceOwnerBoundaryFixture {
	t.Helper()
	pkg, patch, goos := "internal/runtime/syscall/windows", "syscall_windows_go127.go", "windows"
	skips := []string{"StdCall", "AsmStdCallAddr", "asmstdcall", "llgoStdCall", "llgoStdCallAddr"}
	if kind == "gc" {
		pkg, patch, goos = gcScanRuntimePackage, "scan_amd64.go", "linux"
		skips = []string{"ExpandAVX512", "scanSpanPackedAVX512", "FilterNilAVX512", "llgoGCScanFactor", "gcExpandersAVX512"}
	}
	dir := t.TempDir()
	original := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkg))
	entries, err := os.ReadDir(original)
	if err != nil {
		t.Fatal("selected SDK package prerequisite", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" && filepath.Ext(entry.Name()) != ".s" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(original, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		assemblyLanguageTestWrite(t, dir, entry.Name(), string(data))
	}
	patchFile := filepath.Join(env.LLGoRuntimeDir(), "_patch", filepath.FromSlash(pkg), patch)
	body, err := os.ReadFile(patchFile)
	if err != nil {
		t.Fatal(err)
	}
	f := &sourceOwnerBoundaryFixture{
		kind: kind, dir: dir, generated: filepath.Join(dir, "z_llgo_patch_"+patch),
		context: build.Default, body: buildInjectedSourcePatchFile(patchFile, body),
		skips: make(map[string]struct{}), base: make(map[string][]byte),
	}
	f.context.GOOS, f.context.GOARCH = goos, arch
	for _, name := range skips {
		f.skips[name] = struct{}{}
	}
	return f
}

func (f *sourceOwnerBoundaryFixture) check(t *testing.T) (map[string][]byte, error) {
	t.Helper()
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) ([]byte, error) {
		if filepath.Base(path) == f.readFailure {
			return nil, errSourceOwnerBoundaryRead
		}
		if data, ok := f.base[path]; ok {
			return data, nil
		}
		return os.ReadFile(path)
	}
	if f.kind == "gc" {
		return gcScanSourcePatchView(f.dir, entries, f.context, read, f.base, f.body, f.skips)
	}
	return windowsStdCallSourcePatchView(f.dir, entries, f.context, read, f.base, f.body, f.skips)
}

func (f *sourceOwnerBoundaryFixture) reuse(t *testing.T) {
	t.Helper()
	view, err := f.check(t)
	if err != nil {
		t.Fatal("original complete owner prerequisite", err)
	}
	for path, source := range view {
		f.base[path] = source
	}
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(f.dir, entry.Name())
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		filtered, _, err := filterSourcePatchFile(source, f.skips)
		if err != nil {
			t.Fatal(err)
		}
		f.base[path] = filtered
	}
	f.base[f.generated] = bytes.Clone(f.body)
	if _, err := f.check(t); err != nil {
		t.Fatal("unchanged generated view prerequisite", err)
	}
}

func sourceOwnerWithoutBody(t *testing.T, source []byte, name string) []byte {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "replacement.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name && function.Body != nil {
			start, end := fset.Position(function.Body.Pos()).Offset, fset.Position(function.Body.End()).Offset
			return append(append([]byte{}, source[:start]...), source[end:]...)
		}
	}
	t.Fatal("real replacement body not found", name)
	return nil
}

func TestWindowsStdCallSourceOwnerErrorBoundaries(t *testing.T) {
	for _, arch := range []string{"386", "amd64", "arm64"} {
		for _, mutation := range []string{
			"valid", "read-assembly", "read-go", "marker-owner", "generated-view", "invalid-build-tag",
			"extra-assembly", "private-consumer", "incomplete-selection", "malformed-body", "extra-body",
			"helper-collision", "missing-body", "wrong-signature", "duplicate-go-function", "malformed-extra-go",
		} {
			t.Run(arch+"/"+mutation, func(t *testing.T) {
				f := newSourceOwnerBoundaryFixture(t, "windows", arch)
				if _, err := f.check(t); err != nil {
					t.Fatal("complete original source prerequisite", err)
				}
				want := ""
				switch mutation {
				case "valid":
					assemblyLanguageTestWrite(t, f.dir, "notes.txt", "not a selected source")
				case "read-assembly":
					f.readFailure, want = "asm_windows_"+arch+".s", errSourceOwnerBoundaryRead.Error()
				case "read-go":
					f.readFailure, want = "syscall_windows.go", errSourceOwnerBoundaryRead.Error()
				case "marker-owner":
					f.reuse(t)
					delete(f.base, f.generated)
					want = "marker lacks its exact generated body owner"
				case "generated-view":
					f.reuse(t)
					path := filepath.Join(f.dir, "syscall_windows.go")
					f.base[path] = append(bytes.Clone(f.base[path]), []byte("\nvar changedView int\n")...)
					want = "changed generated Windows declaration view"
				case "invalid-build-tag":
					assemblyLanguageTestWrite(t, f.dir, "extra.go", "//go:build (\n\npackage windows\n")
					want = "build"
				case "extra-assembly":
					assemblyLanguageTestWrite(t, f.dir, "extra.s", "TEXT ·Other(SB),$0-0\nRET\n")
					want = "unreviewed Windows syscall assembly owner"
				case "private-consumer":
					assemblyLanguageTestWrite(t, f.dir, "extra.go", "package windows\nfunc extra() { asmstdcall(nil) }\n")
					want = "remaining SDK Go consumer of native asmstdcall"
				case "incomplete-selection":
					if err := os.Rename(filepath.Join(f.dir, "syscall_windows.go"), filepath.Join(f.dir, "unused.txt")); err != nil {
						t.Fatal(err)
					}
					want = "incomplete selected Windows syscall source group"
				case "malformed-body":
					f.body, want = []byte("package"), "expected"
				case "extra-body":
					f.body = append(bytes.Clone(f.body), []byte("\nfunc unexpected() {}\n")...)
					want = "replacement body group changed"
				case "helper-collision":
					assemblyLanguageTestWrite(t, f.dir, "extra.go", "package windows\nfunc llgoStdCall() {}\n")
					want = "replacement body group changed"
				case "missing-body":
					f.body = sourceOwnerWithoutBody(t, f.body, "StdCall")
					want = "lacks its original signature and real Go body"
				case "wrong-signature":
					f.body = bytes.Replace(f.body, []byte("func StdCall(fn *StdCallInfo) uint32"), []byte("func StdCall(fn *StdCallInfo) uint64"), 1)
					want = "signature changed"
				case "duplicate-go-function":
					assemblyLanguageTestWrite(t, f.dir, "extra.go", "package windows\nfunc StdCall() {}\n")
					want = "duplicate ARM64 atomic Go owner"
				case "malformed-extra-go":
					assemblyLanguageTestWrite(t, f.dir, "extra.go", "package windows\nfunc extra(\n")
					want = "expected"
				}
				view, err := f.check(t)
				if want == "" {
					if err != nil || len(view) != 1 || string(view[filepath.Join(f.dir, "asm_windows_"+arch+".s")]) != "// replaced by LLGo source patch\n" {
						t.Fatal("complete owner lost its exact assembly selection", view, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), want) || view != nil {
					t.Fatalf("owner %s: got %v, %v; want %q and no replacement view", mutation, view, err, want)
				}
				if f.readFailure != "" && !errors.Is(err, errSourceOwnerBoundaryRead) {
					t.Fatal("selected reader failure lost its identity", err)
				}
			})
		}
	}
}

func TestGCScanSourceOwnerErrorBoundaries(t *testing.T) {
	for _, mutation := range []string{
		"read-assembly", "read-go", "generated-view", "invalid-build-tag", "malformed-extra-go",
		"malformed-body", "extra-body", "helper-collision", "helper-without-body", "duplicate-go-function",
	} {
		t.Run(mutation, func(t *testing.T) {
			f := newSourceOwnerBoundaryFixture(t, "gc", "amd64")
			if _, err := f.check(t); err != nil {
				t.Fatal("complete original source prerequisite", err)
			}
			want := ""
			switch mutation {
			case "read-assembly":
				f.readFailure, want = "expand_amd64.s", errSourceOwnerBoundaryRead.Error()
			case "read-go":
				f.readFailure, want = "expand_amd64.go", errSourceOwnerBoundaryRead.Error()
			case "generated-view":
				f.reuse(t)
				path := filepath.Join(f.dir, "expand_amd64.go")
				f.base[path] = append(bytes.Clone(f.base[path]), []byte("\nvar changedView int\n")...)
				want = "changed generated gc scan declaration view"
			case "invalid-build-tag":
				assemblyLanguageTestWrite(t, f.dir, "extra.go", "//go:build (\n\npackage scan\n")
				want = "build"
			case "malformed-extra-go":
				assemblyLanguageTestWrite(t, f.dir, "extra.go", "package scan\nfunc extra(\n")
				want = "expected"
			case "malformed-body":
				f.body, want = []byte("package"), "expected"
			case "extra-body":
				f.body = append(bytes.Clone(f.body), []byte("\nfunc unexpected() {}\n")...)
				want = "replacement changed its complete body group"
			case "helper-collision":
				assemblyLanguageTestWrite(t, f.dir, "extra.go", "package scan\nfunc llgoGCScanFactor() {}\n")
				want = "replacement changed its complete body group"
			case "helper-without-body":
				f.body = sourceOwnerWithoutBody(t, f.body, "llgoGCScanFactor")
				want = "lacks its actual bounded size-class helper"
			case "duplicate-go-function":
				assemblyLanguageTestWrite(t, f.dir, "extra.go", "package scan\nfunc ExpandAVX512() {}\n")
				want = "duplicate ARM64 atomic Go owner"
			}
			view, err := f.check(t)
			if err == nil || !strings.Contains(err.Error(), want) || view != nil {
				t.Fatalf("owner %s: got %v, %v; want %q and no replacement view", mutation, view, err, want)
			}
			if f.readFailure != "" && !errors.Is(err, errSourceOwnerBoundaryRead) {
				t.Fatal("selected reader failure lost its identity", err)
			}
		})
	}
}

func TestRuntimeSourceOwnerSelectionBoundaries(t *testing.T) {
	for _, kind := range []string{"windows", "gc"} {
		for _, mutation := range []string{"skip-all", "renamed-owner", "additional-patch"} {
			t.Run(kind+"/"+mutation, func(t *testing.T) {
				pkg, filename, goos := "internal/runtime/syscall/windows", "syscall_windows_go127.go", "windows"
				want := "Windows syscall owner must retain the other SDK sources"
				if kind == "gc" {
					pkg, filename, goos = gcScanRuntimePackage, "scan_amd64.go", "linux"
					want = "AMD64 gc scan owner must retain the other SDK Go sources"
				}
				body, err := os.ReadFile(filepath.Join(env.LLGoRuntimeDir(), "_patch", filepath.FromSlash(pkg), filename))
				if err != nil {
					t.Fatal(err)
				}
				goroot, runtimeDir := t.TempDir(), t.TempDir()
				patchDir := filepath.Join(runtimeDir, "_patch", filepath.FromSlash(pkg))
				sourceDir := filepath.Join(goroot, "src", filepath.FromSlash(pkg))
				for _, directory := range []string{sourceDir, patchDir} {
					if err := os.MkdirAll(directory, 0o700); err != nil {
						t.Fatal(err)
					}
				}
				// Selection must fail before it can claim ownership of any SDK assembly.
				assemblyLanguageTestWrite(t, sourceDir, "unrelated.go", "package fixture\n")
				switch mutation {
				case "skip-all":
					body = append(bytes.Clone(body), []byte("\n//llgo:skipall\n")...)
				case "renamed-owner":
					filename = "unrecognized.go"
					want = "missing complete Go1.27 Windows syscall owner"
					if kind == "gc" {
						want = "AMD64 gc scan owner lacks its complete Go body source"
					}
				case "additional-patch":
					assemblyLanguageTestWrite(t, patchDir, "unrelated.go", "package fixture\n")
				}
				assemblyLanguageTestWrite(t, patchDir, filename, string(body))
				changed, view, files, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkg,
					sourcePatchBuildContext{goos: goos, goarch: "amd64", goversion: "go1.27.2"})
				if err == nil || !strings.Contains(err.Error(), want) || changed || view != nil || files != nil {
					t.Fatal("changed patch selection acquired SDK ownership", changed, view, files, err)
				}
			})
		}
	}
}
